package main

import (
	"log/slog"
	"os"
	"time"

	actorcommon "internal/actors/common"
	configactor "internal/actors/scopes/configctl"
	"internal/shared/bootstrap"
	"internal/shared/healthz"
	"internal/shared/settings"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log, "configctl")
	slog.SetDefault(logger)

	logger.Info("configctl starting")

	// ── Phase 0: Preflight — fail fast on missing preconditions ────
	bootstrap.RunPreflight("configctl", logger, []bootstrap.PreflightCheck{
		bootstrap.NATSEnabledCheck(config),
		bootstrap.NATSURLFormatCheck(config),
	})

	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	pid := engine.Spawn(configactor.NewConfigSupervisor(config), "configctl")

	// Start health server for operational visibility.
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		[]healthz.ReadinessCheck{bootstrap.NATSReadinessCheck(config)},
		nil, // configctl has no pipeline trackers
		healthz.WithRuntime("configctl"),
	)
	srv.StartInBackground()

	actorcommon.WaitTillShutdown(engine, pid)

	_ = srv.GracefulShutdown(5 * time.Second)
}
