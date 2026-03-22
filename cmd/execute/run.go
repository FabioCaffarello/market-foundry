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
	domainexec "internal/domain/execution"
	"internal/shared/bootstrap"
	"internal/shared/healthz"
	"internal/shared/settings"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log, "execute")
	slog.SetDefault(logger)

	logger.Info("execute starting")

	// ── Phase 0: Preflight — fail fast on missing preconditions ────
	bootstrap.RunPreflight("execute", logger, []bootstrap.PreflightCheck{
		bootstrap.NATSEnabledCheck(config),
		bootstrap.NATSURLFormatCheck(config),
	})

	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	// Build venue adapter via config-driven selection.
	venueResult, err := buildVenueAdapter(config)
	if err != nil {
		logger.Error("build venue adapter", "error", err)
		os.Exit(1)
	}
	logger.Info("venue adapter selected",
		"type", config.Venue.Type,
		"query_capable", venueResult.query != nil,
	)

	// S339: Log canonical activation surface at startup.
	// This is the single authoritative log line that shows the composite activation state.
	// Gate state is logged as "unknown" at startup because KV is not yet connected;
	// the adapter actor will log the resolved gate state after connecting.
	adapterState := domainexec.AdapterPaper
	if venueResult.credentialState == domainexec.CredentialPresent {
		adapterState = domainexec.AdapterVenue
	}
	if config.Venue.Type == settings.VenueTypePaperSimulator || config.Venue.Type == "" {
		adapterState = domainexec.AdapterPaper
	}
	logger.Info("activation surface at startup",
		"adapter", string(adapterState),
		"credentials", string(venueResult.credentialState),
		"effective_without_gate", string(domainexec.ComputeEffectiveMode(adapterState, domainexec.GateActive, venueResult.credentialState)),
		"note", "gate state resolves after NATS KV connect",
	)

	// Build health trackers.
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":     healthz.NewTracker("venue-adapter"),
		"venue-consumer":    healthz.NewTracker("venue-consumer"),
		"strategy-consumer": healthz.NewTracker("strategy-consumer"), // S360
	}

	pid := engine.Spawn(
		executeactor.NewExecuteSupervisor(config, venueResult.submit, venueResult.query, trackers,
			executeactor.WithActivationState(adapterState, venueResult.credentialState),
		),
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
		healthz.WithRuntime("execute"),
	)
	srv.StartInBackground()

	actorcommon.WaitTillShutdown(engine, pid)

	_ = srv.GracefulShutdown(5 * time.Second)
}

// venueAdapterResult holds both the submit and optional query ports for a venue.
// The query port is used by Post200Reconciler (S322) for body-read-failure recovery.
// S339: credentialState tracks whether venue credentials were loaded at startup.
type venueAdapterResult struct {
	submit          ports.VenuePort
	query           ports.VenueQueryPort         // nil for adapters without query capability (e.g. paper)
	credentialState domainexec.CredentialState
}

func buildVenueAdapter(config settings.AppConfig) (venueAdapterResult, error) {
	switch config.Venue.Type {
	case settings.VenueTypePaperSimulator:
		return venueAdapterResult{submit: appexec.NewPaperVenueAdapter(0), credentialState: domainexec.CredentialAbsent}, nil
	case "":
		// Default to paper_simulator when venue config is absent (backward compatible).
		return venueAdapterResult{submit: appexec.NewPaperVenueAdapter(0), credentialState: domainexec.CredentialAbsent}, nil

	case settings.VenueTypeBinanceFuturesTestnet:
		creds, prob := appexec.LoadCredentials(string(config.Venue.Type), []string{"API_KEY", "API_SECRET"})
		if prob != nil {
			return venueAdapterResult{}, fmt.Errorf("venue %q credential load failed: %s", config.Venue.Type, prob.Message)
		}
		submitTimeout := config.Venue.SubmitTimeoutDuration()
		adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, submitTimeout)
		return venueAdapterResult{submit: adapter, query: adapter, credentialState: domainexec.CredentialPresent}, nil

	default:
		// Unknown venue types require credential loading and activation gate ceremony.
		creds, prob := appexec.LoadCredentials(string(config.Venue.Type), []string{"API_KEY", "API_SECRET"})
		if prob != nil {
			return venueAdapterResult{}, fmt.Errorf("venue %q credential load failed: %s", config.Venue.Type, prob.Message)
		}
		_ = creds
		return venueAdapterResult{}, fmt.Errorf("venue type %q is registered but has no adapter implementation yet; activation gate ceremony required", config.Venue.Type)
	}
}
