package main

import (
	"log/slog"
	"os"
	"time"

	actorcommon "internal/actors/common"
	ingestactor "internal/actors/scopes/ingest"
	natsconfigctl "internal/adapters/nats/natsconfigctl"
	natskit "internal/adapters/nats/natskit"
	"internal/shared/bootstrap"
	"internal/shared/healthz"
	"internal/shared/settings"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log, "ingest")
	slog.SetDefault(logger)

	logger.Info("ingest starting")

	// ── Phase 0: Preflight — fail fast on missing preconditions ────
	bootstrap.RunPreflight("ingest", logger, []bootstrap.PreflightCheck{
		bootstrap.NATSEnabledCheck(config),
		bootstrap.NATSURLFormatCheck(config),
	})

	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	// Create configctl gateway for querying active bindings.
	var gateway *natsconfigctl.Gateway
	if config.NATS.Enabled {
		client, err := natskit.NewNATSRequestClientWithURL(config.NATS.URL, config.NATS.RequestTimeoutDuration())
		if err != nil {
			logger.Error("create configctl request client", "error", err)
			os.Exit(1)
		}
		defer func() { _ = client.Close() }()
		gateway = natsconfigctl.NewGateway(client, "ingest.binding-watcher")
	}

	publisherTracker := healthz.NewTracker("observation-publisher")

	pid := engine.Spawn(ingestactor.NewIngestSupervisor(config, gateway, publisherTracker), "ingest")

	// Start health server for operational visibility.
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		[]healthz.ReadinessCheck{bootstrap.NATSReadinessCheck(config)},
		[]*healthz.Tracker{publisherTracker},
		healthz.WithRuntime("ingest"),
	)
	srv.StartInBackground()

	actorcommon.WaitTillShutdown(engine, pid)

	_ = srv.GracefulShutdown(5 * time.Second)
}
