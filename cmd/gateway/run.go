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

	// Phase 2: Wire use cases from connections → route dependencies.
	deps := buildRouteDependencies(config, conns)

	// Phase 3: Assemble routes and spawn the gateway actor.
	gatewayRoutes := routes.DefaultRoutes(deps)
	pid := engine.Spawn(actorgateway.NewGateway(config.HTTP, gatewayRoutes), "gateway")
	actorcommon.WaitTillShutdown(engine, pid)
}
