package main

import (
	"log/slog"
	"os"
	"time"

	actorcommon "internal/actors/common"
	ingestactor "internal/actors/scopes/ingest"
	adapternats "internal/adapters/nats"
	"internal/shared/bootstrap"
	"internal/shared/healthz"
	"internal/shared/settings"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log)
	slog.SetDefault(logger)

	logger.Info("ingest starting")

	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	// Create configctl gateway for querying active bindings.
	var gateway *adapternats.ConfigctlGateway
	if config.NATS.Enabled {
		client, err := adapternats.NewNATSRequestClientWithURL(config.NATS.URL, config.NATS.RequestTimeoutDuration())
		if err != nil {
			logger.Error("create configctl request client", "error", err)
			os.Exit(1)
		}
		defer client.Close()
		gateway = adapternats.NewConfigctlGateway(client, "ingest.binding-watcher")
	}

	publisherTracker := healthz.NewTracker("observation-publisher")

	pid := engine.Spawn(ingestactor.NewIngestSupervisor(config, gateway, publisherTracker), "ingest")

	// Start health server for operational visibility.
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		[]healthz.ReadinessCheck{bootstrap.NATSReadinessCheck(config)},
		[]*healthz.Tracker{publisherTracker},
	)
	srv.StartInBackground()

	actorcommon.WaitTillShutdown(engine, pid)

	_ = srv.GracefulShutdown(5 * time.Second)
}
