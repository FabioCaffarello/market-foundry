package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	actorcommon "internal/actors/common"
	executeactor "internal/actors/scopes/execute"
	natsevidence "internal/adapters/nats/natsevidence"
	appexec "internal/application/execution"
	"internal/application/executionclient"
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

	// ── Phase -1: Wire credential provider from config ────────────
	// S439: Select credential backend before preflight so that
	// MainnetCredentialCheck uses the correct provider.
	switch config.Venue.CredentialProviderName() {
	case "file":
		fp := appexec.NewFileCredentialProvider(config.Venue.CredentialPath)
		appexec.SetCredentialProvider(fp)
		logger.Info("credential provider set to file", "path", config.Venue.CredentialPath)
	default:
		// "env" is the default — already set at package init.
		logger.Info("credential provider set to env")
	}

	// ── Phase 0: Preflight — fail fast on missing preconditions ────
	// S434: MainnetCredentialCheck verifies secrets are resolvable before bootstrap.
	// S439: CredentialPathCheck validates the file provider's base path.
	bootstrap.RunPreflight("execute", logger, []bootstrap.PreflightCheck{
		bootstrap.NATSEnabledCheck(config),
		bootstrap.NATSURLFormatCheck(config),
		bootstrap.CredentialPathCheck(config),
		bootstrap.MainnetCredentialCheck(config, appexec.DefaultCredentialProvider().Resolve),
	})

	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	// S399: Build venue adapter(s) via config-driven selection.
	// Unified segments model: resolve adapter from segments map when present.
	venueResult, err := buildVenueAdapter(config, logger)
	if err != nil {
		logger.Error("build venue adapter", "error", err)
		os.Exit(1)
	}
	// P4.2: release background goroutines owned by venue decorators
	// (e.g. RateLimiter's refill loop) on shutdown.
	defer func() {
		for _, c := range venueResult.closers {
			c()
		}
	}()

	// S387: Build PriceSource from CANDLE_LATEST KV bucket for realistic fill prices.
	// This closes the G1 gap from S384 — DryRunSubmitter and PaperVenueAdapter receive
	// live market prices instead of defaulting to "0".
	var priceSource ports.PriceSource
	candleStore := natsevidence.NewCandleKVStore(config.NATS.URL)
	if err := candleStore.Start(); err != nil {
		logger.Warn("candle KV store unavailable — price source disabled, fills will use default price",
			"error", err,
		)
	} else {
		priceSource = natsevidence.NewCandleKVPriceSource(candleStore, logger.With("component", "price-source"))
		logger.Info("price source enabled via CANDLE_LATEST KV")
	}

	// Inject PriceSource into PaperVenueAdapter if applicable.
	if pva, ok := venueResult.submit.(*appexec.PaperVenueAdapter); ok && priceSource != nil {
		pva.WithPriceSource(priceSource)
	}

	// S379: Wrap with DryRunSubmitter when dry_run is active (the default).
	// This is the outermost decorator — it intercepts all venue calls before
	// any retry, reconciliation, or real adapter code runs.
	dryRunActive := config.Venue.IsDryRun()
	if dryRunActive {
		drs := appexec.NewDryRunSubmitter(venueResult.submit).
			WithLogger(logger.With("component", "dry-run-submitter"))
		if priceSource != nil {
			drs.WithPriceSource(priceSource)
		}
		venueResult.submit = drs
		// DryRunSubmitter never reaches the venue, so query port is irrelevant.
		venueResult.query = nil
	}

	// S399: Log segment identity for compose-level auditability.
	enabledSegs := config.Venue.EnabledSegments()
	segmentStr := "none"
	if len(enabledSegs) > 0 {
		names := make([]string, len(enabledSegs))
		for i, s := range enabledSegs {
			names[i] = string(s)
		}
		segmentStr = strings.Join(names, ",")
	}

	logger.Info("venue adapter selected",
		"type", venueResult.activeType,
		"enabled_segments", segmentStr,
		"query_capable", venueResult.query != nil,
		"dry_run", dryRunActive,
		"credential_provider", appexec.DefaultCredentialProvider().Name(),
	)

	// S339: Log canonical activation surface at startup.
	// This is the single authoritative log line that shows the composite activation state.
	// Gate state is logged as "unknown" at startup because KV is not yet connected;
	// the adapter actor will log the resolved gate state after connecting.
	adapterState := domainexec.AdapterPaper
	if venueResult.credentialState == domainexec.CredentialPresent {
		adapterState = domainexec.AdapterVenue
	}
	if venueResult.activeType == settings.VenueTypePaperSimulator || venueResult.activeType == "" {
		adapterState = domainexec.AdapterPaper
	}
	logger.Info("activation surface at startup",
		"adapter", string(adapterState),
		"credentials", string(venueResult.credentialState),
		"dry_run", dryRunActive,
		"effective_without_gate", string(executionclient.ComputeEffectiveMode(adapterState, domainexec.GateActive, venueResult.credentialState)),
		"note", "gate state resolves after NATS KV connect",
	)

	// Build health trackers.
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":     healthz.NewTracker("venue-adapter"),
		"venue-consumer":    healthz.NewTracker("venue-consumer"),
		"strategy-consumer": healthz.NewTracker("strategy-consumer"), // S360
	}

	// S429: Build per-segment health registry for operational readiness signals.
	segmentRegistry := healthz.NewSegmentHealthRegistry()
	for _, seg := range enabledSegs {
		segmentRegistry.Register(healthz.SegmentDescriptor{
			Name:    string(seg),
			Enabled: true,
			Adapter: string(config.Venue.AdapterForSegment(seg)),
		}, trackers["venue-adapter"])
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
	// S429: WithSegments adds per-segment health breakdown to /statusz and /diagz.
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		[]healthz.ReadinessCheck{bootstrap.NATSReadinessCheck(config)},
		allTrackers,
		healthz.WithRuntime("execute"),
		healthz.WithSegments(segmentRegistry),
	)
	srv.StartInBackground()

	actorcommon.WaitTillShutdown(engine, pid)

	_ = srv.GracefulShutdown(5 * time.Second)
	_ = candleStore.Close() // S387: clean up price source KV connection.
}

// venueAdapterResult holds both the submit and optional query ports for a venue.
// The query port is used by Post200Reconciler (S322) for body-read-failure recovery.
// S339: credentialState tracks whether venue credentials were loaded at startup.
// S399: activeType records which adapter was selected (from segments or type).
// P4.2: closers carries lifecycle teardown hooks (e.g. RateLimiter.Close) that
// the caller must invoke on shutdown to release background goroutines.
type venueAdapterResult struct {
	submit          ports.VenuePort
	query           ports.VenueQueryPort         // nil for adapters without query capability (e.g. paper)
	credentialState domainexec.CredentialState
	activeType      settings.VenueType           // S399: which adapter is active
	closers         []func()                     // P4.2: invoke on shutdown
}

// buildVenueAdapter resolves the venue adapter from config.
// S399: When unified segments are present, resolves the adapter from the
// first enabled segment. When no segments are defined, falls back to Type.
func buildVenueAdapter(config settings.AppConfig, logger *slog.Logger) (venueAdapterResult, error) {
	// S399: If unified segments are configured, resolve adapter from segments.
	if config.Venue.HasUnifiedSegments() {
		return buildVenueAdapterFromSegments(config, logger)
	}

	// Standalone path: resolve from venue.type (paper_simulator or bare config).
	return buildVenueAdapterByType(config.Venue.Type, config.Venue.SubmitTimeoutDuration())
}

// buildVenueAdapterFromSegments resolves adapters from the unified segments map.
// S400: Builds an adapter for EACH enabled segment and wraps them in a
// SegmentRouter that dispatches intents by source. For single-segment configs,
// the router contains one adapter — no behavioral change from S399.
func buildVenueAdapterFromSegments(config settings.AppConfig, logger *slog.Logger) (venueAdapterResult, error) {
	enabledSegs := config.Venue.EnabledSegments()
	if len(enabledSegs) == 0 {
		return venueAdapterResult{}, fmt.Errorf("segments map present but no segments enabled — config validation should have caught this")
	}

	router := appexec.NewSegmentRouter()
	var lastType settings.VenueType
	var lastCredState domainexec.CredentialState
	var aggregatedClosers []func()
	timeout := config.Venue.SubmitTimeoutDuration()

	for _, seg := range enabledSegs {
		adapterType := config.Venue.AdapterForSegment(seg)
		result, err := buildVenueAdapterByType(adapterType, timeout)
		if err != nil {
			for _, c := range aggregatedClosers {
				c()
			}
			return venueAdapterResult{}, fmt.Errorf("build adapter for segment %q: %w", seg, err)
		}
		router.Register(seg, result.submit)
		if result.query != nil {
			router.RegisterQuery(seg, result.query)
		}
		aggregatedClosers = append(aggregatedClosers, result.closers...)
		lastType = result.activeType
		lastCredState = result.credentialState
	}

	names := make([]string, len(enabledSegs))
	for i, s := range enabledSegs {
		names[i] = string(s)
	}

	if len(enabledSegs) > 1 {
		logger.Info("multi-segment runtime — adapters built for all enabled segments",
			"enabled_segments", strings.Join(names, ","),
			"segment_count", len(enabledSegs),
		)
		// For multi-segment, report the type as composite.
		lastType = "multi_segment"
	}

	// Determine credential state: if any segment has credentials, report present.
	credState := lastCredState

	return venueAdapterResult{
		submit:          router,
		query:           router,
		credentialState: credState,
		activeType:      lastType,
		closers:         aggregatedClosers,
	}, nil
}

// buildVenueAdapterByType builds a venue adapter for the given type.
func buildVenueAdapterByType(venueType settings.VenueType, submitTimeout time.Duration) (venueAdapterResult, error) {
	switch venueType {
	case settings.VenueTypePaperSimulator, "":
		return venueAdapterResult{
			submit:          appexec.NewPaperVenueAdapter(0),
			credentialState: domainexec.CredentialAbsent,
			activeType:      settings.VenueTypePaperSimulator,
		}, nil

	case settings.VenueTypeBinanceFuturesTestnet:
		creds, prob := appexec.LoadCredentials(string(venueType), []string{"API_KEY", "API_SECRET"})
		if prob != nil {
			return venueAdapterResult{}, fmt.Errorf("venue %q credential load failed: %s", venueType, prob.Message)
		}
		adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, submitTimeout)
		return venueAdapterResult{
			submit:          adapter,
			query:           adapter,
			credentialState: domainexec.CredentialPresent,
			activeType:      venueType,
		}, nil

	case settings.VenueTypeBinanceSpotTestnet:
		creds, prob := appexec.LoadCredentials(string(venueType), []string{"API_KEY", "API_SECRET"})
		if prob != nil {
			return venueAdapterResult{}, fmt.Errorf("venue %q credential load failed: %s", venueType, prob.Message)
		}
		adapter := appexec.NewBinanceSpotTestnetAdapter(creds, submitTimeout)
		return venueAdapterResult{
			submit:          adapter,
			query:           adapter,
			credentialState: domainexec.CredentialPresent,
			activeType:      venueType,
		}, nil

	case settings.VenueTypeBinanceFuturesMainnet:
		creds, prob := appexec.LoadCredentials(string(venueType), []string{"API_KEY", "API_SECRET"})
		if prob != nil {
			return venueAdapterResult{}, fmt.Errorf("venue %q credential load failed: %s", venueType, prob.Message)
		}
		adapter := appexec.NewBinanceFuturesMainnetAdapter(creds, submitTimeout)
		rateLimited := appexec.NewRateLimiter(adapter, 10, 100*time.Millisecond)
		return venueAdapterResult{
			submit:          rateLimited,
			query:           adapter,
			credentialState: domainexec.CredentialPresent,
			activeType:      venueType,
			closers:         []func(){rateLimited.Close},
		}, nil

	case settings.VenueTypeBinanceSpotMainnet:
		creds, prob := appexec.LoadCredentials(string(venueType), []string{"API_KEY", "API_SECRET"})
		if prob != nil {
			return venueAdapterResult{}, fmt.Errorf("venue %q credential load failed: %s", venueType, prob.Message)
		}
		adapter := appexec.NewBinanceSpotMainnetAdapter(creds, submitTimeout)
		rateLimited := appexec.NewRateLimiter(adapter, 10, 100*time.Millisecond)
		return venueAdapterResult{
			submit:          rateLimited,
			query:           adapter,
			credentialState: domainexec.CredentialPresent,
			activeType:      venueType,
			closers:         []func(){rateLimited.Close},
		}, nil

	default:
		return venueAdapterResult{}, fmt.Errorf("venue type %q is not recognized; check venue.type or segments.*.adapter in config", venueType)
	}
}
