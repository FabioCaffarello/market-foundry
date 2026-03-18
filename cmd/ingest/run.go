package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
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
	checks := buildReadinessChecks(config)
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		checks,
		[]*healthz.Tracker{publisherTracker},
	)
	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("health server failed", "error", err)
		}
	}()

	actorcommon.WaitTillShutdown(engine, pid)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func buildReadinessChecks(config settings.AppConfig) []healthz.ReadinessCheck {
	return []healthz.ReadinessCheck{
		{
			Name: "nats",
			Check: func(ctx context.Context) error {
				if !config.NATS.Enabled {
					return fmt.Errorf("nats is not enabled")
				}
				return dialNATS(config.NATS.URL)
			},
		},
	}
}

func dialNATS(natsURL string) error {
	u, err := url.Parse(natsURL)
	if err != nil {
		return fmt.Errorf("parse nats url: %w", err)
	}
	host := u.Host
	if host == "" {
		host = u.Opaque
	}
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return fmt.Errorf("nats dial: %w", err)
	}
	conn.Close()
	return nil
}
