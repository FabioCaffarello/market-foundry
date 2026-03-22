package main

import (
	"log/slog"
	"os"

	actorcommon "internal/actors/common"
	actorgateway "internal/actors/scopes/gateway"
	"internal/interfaces/http/routes"
	"internal/shared/bootstrap"
	"internal/shared/settings"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log, "gateway")
	slog.SetDefault(logger)

	// ── Phase 0: Preflight — fail fast on missing preconditions ────
	bootstrap.RunPreflight("gateway", logger, []bootstrap.PreflightCheck{
		bootstrap.NATSEnabledCheck(config),
		bootstrap.NATSURLFormatCheck(config),
	})

	logger.Info("gateway starting", "addr", config.HTTP.Addr)
	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	// Phase 1: Create all NATS gateway connections.
	conns, prob := buildGatewayConns(config, logger)
	if prob != nil {
		logger.Error("build gateway connections", "error", prob)
		os.Exit(1)
	}
	defer conns.Close(logger)

	// Phase 2a: Create optional ClickHouse client for analytical queries.
	chClient := buildAnalyticalClient(config, logger)
	if chClient != nil {
		defer chClient.Close()
	}

	// Phase 2b: Wire use cases from connections → route dependencies.
	deps := buildRouteDependencies(config, conns, chClient, logger)

	// Phase 3: Assemble routes and spawn the gateway actor.
	gatewayRoutes := routes.DefaultRoutes(deps)
	pid := engine.Spawn(actorgateway.NewGateway(config.HTTP, gatewayRoutes), "gateway")
	actorcommon.WaitTillShutdown(engine, pid)
}
