package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	actorcommon "internal/actors/common"
	executeactor "internal/actors/scopes/execute"
	appexec "internal/application/execution"
	"internal/application/ports"
	"internal/shared/bootstrap"
	"internal/shared/healthz"
	"internal/shared/settings"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log)
	slog.SetDefault(logger)

	logger.Info("execute starting")

	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	// Build venue adapter via config-driven selection.
	venue, err := buildVenueAdapter(config)
	if err != nil {
		logger.Error("build venue adapter", "error", err)
		os.Exit(1)
	}
	logger.Info("venue adapter selected", "type", config.Venue.Type)

	// Build health trackers.
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  healthz.NewTracker("venue-adapter"),
		"venue-consumer": healthz.NewTracker("venue-consumer"),
	}

	pid := engine.Spawn(
		executeactor.NewExecuteSupervisor(config, venue, trackers),
		"execute",
	)

	// Collect all trackers for health server.
	allTrackers := make([]*healthz.Tracker, 0, len(trackers))
	for _, t := range trackers {
		allTrackers = append(allTrackers, t)
	}

	// Start health server for operational visibility.
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		[]healthz.ReadinessCheck{bootstrap.NATSReadinessCheck(config)},
		allTrackers,
	)
	srv.StartInBackground()

	actorcommon.WaitTillShutdown(engine, pid)

	_ = srv.GracefulShutdown(5 * time.Second)
}

func buildVenueAdapter(config settings.AppConfig) (ports.VenuePort, error) {
	switch config.Venue.Type {
	case settings.VenueTypePaperSimulator:
		return appexec.NewPaperVenueAdapter(0), nil
	case "":
		// Default to paper_simulator when venue config is absent (backward compatible).
		return appexec.NewPaperVenueAdapter(0), nil

	case settings.VenueTypeBinanceFuturesTestnet:
		creds, prob := appexec.LoadCredentials(string(config.Venue.Type), []string{"API_KEY", "API_SECRET"})
		if prob != nil {
			return nil, fmt.Errorf("venue %q credential load failed: %s", config.Venue.Type, prob.Message)
		}
		submitTimeout := config.Venue.SubmitTimeoutDuration()
		return appexec.NewBinanceFuturesTestnetAdapter(creds, submitTimeout), nil

	default:
		// Unknown venue types require credential loading and activation gate ceremony.
		creds, prob := appexec.LoadCredentials(string(config.Venue.Type), []string{"API_KEY", "API_SECRET"})
		if prob != nil {
			return nil, fmt.Errorf("venue %q credential load failed: %s", config.Venue.Type, prob.Message)
		}
		_ = creds
		return nil, fmt.Errorf("venue type %q is registered but has no adapter implementation yet; activation gate ceremony required", config.Venue.Type)
	}
}
