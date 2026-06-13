package main

import (
	"log/slog"

	"internal/adapters/clickhouse"
	"internal/adapters/exchanges/binancef"
	"internal/adapters/exchanges/binances"
	"internal/adapters/exchanges/bybitf"
	"internal/adapters/exchanges/bybits"
	natsconfigctl "internal/adapters/nats/natsconfigctl"
	natsdecision "internal/adapters/nats/natsdecision"
	natsevidence "internal/adapters/nats/natsevidence"
	natsexecution "internal/adapters/nats/natsexecution"
	natsinsights "internal/adapters/nats/natsinsights"
	natskit "internal/adapters/nats/natskit"
	natsrisk "internal/adapters/nats/natsrisk"
	natssignal "internal/adapters/nats/natssignal"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/application/analyticalclient"
	configctlclient "internal/application/configctlclient"
	"internal/application/decisionclient"
	"internal/application/evidenceclient"
	"internal/application/executionclient"
	"internal/application/insightsclient"
	"internal/application/monitoringclient"
	"internal/application/ports"
	"internal/application/riskclient"
	"internal/application/signalclient"
	"internal/application/strategyclient"
	"internal/application/triageclient"
	"internal/domain/monitoring"
	"internal/interfaces/http/routes"
	"internal/shared/clock"
	"internal/shared/problem"
	"internal/shared/settings"
)

// gatewayConns holds all NATS gateway connections created during composition.
// It owns their lifecycle — call Close() to release all underlying connections.
type gatewayConns struct {
	configctl        ports.ConfigctlGateway
	evidence         ports.EvidenceGateway
	signal           ports.SignalGateway
	decision         ports.DecisionGateway
	strategy         ports.StrategyGateway
	risk             ports.RiskGateway
	execution        ports.ExecutionGateway
	executionControl ports.ExecutionControlGateway
	session          ports.SessionGateway  // S460
	insights         ports.InsightsGateway // H-8.a (KV-direct reader)
	closers          []func() error
}

// Close releases all underlying NATS connections.
func (c *gatewayConns) Close(logger *slog.Logger) {
	for _, fn := range c.closers {
		if err := fn(); err != nil {
			logger.Error("close gateway connection", "error", err)
		}
	}
}

// buildGatewayConns creates all NATS gateway connections for the gateway runtime.
// Configctl is required; all other gateways are optional and degrade gracefully.
func buildGatewayConns(config settings.AppConfig, logger *slog.Logger) (*gatewayConns, *problem.Problem) {
	conns := &gatewayConns{}

	addCloser := func(fn func() error) {
		if fn != nil {
			conns.closers = append(conns.closers, fn)
		}
	}

	// connectOptional creates a gateway and logs a warning if unavailable.
	connectOptional := func(label string, closeFn func() error, prob *problem.Problem) {
		if prob != nil {
			logger.Warn(label+" gateway unavailable", "error", prob)
		}
		addCloser(closeFn)
	}

	// Configctl — required for gateway operation.
	gw, closeFn, prob := newGatewayConn(config, "configctl", func(rc *natskit.NATSRequestClient) ports.ConfigctlGateway {
		return natsconfigctl.NewGateway(rc, "gateway.http")
	})
	if prob != nil {
		return nil, prob
	}
	conns.configctl = gw
	addCloser(closeFn)

	// Optional domain gateways — each creates its own NATS request client.
	var cl func() error
	var p *problem.Problem

	conns.evidence, cl, p = newGatewayConn(config, "evidence", func(rc *natskit.NATSRequestClient) ports.EvidenceGateway {
		return natsevidence.NewGateway(rc, "gateway.http")
	})
	connectOptional("evidence", cl, p)

	conns.signal, cl, p = newGatewayConn(config, "signal", func(rc *natskit.NATSRequestClient) ports.SignalGateway {
		return natssignal.NewGateway(rc, "gateway.http")
	})
	connectOptional("signal", cl, p)

	conns.decision, cl, p = newGatewayConn(config, "decision", func(rc *natskit.NATSRequestClient) ports.DecisionGateway {
		return natsdecision.NewGateway(rc, "gateway.http")
	})
	connectOptional("decision", cl, p)

	conns.strategy, cl, p = newGatewayConn(config, "strategy", func(rc *natskit.NATSRequestClient) ports.StrategyGateway {
		return natsstrategy.NewGateway(rc, "gateway.http")
	})
	connectOptional("strategy", cl, p)

	conns.risk, cl, p = newGatewayConn(config, "risk", func(rc *natskit.NATSRequestClient) ports.RiskGateway {
		return natsrisk.NewGateway(rc, "gateway.http")
	})
	connectOptional("risk", cl, p)

	conns.execution, cl, p = newGatewayConn(config, "execution", func(rc *natskit.NATSRequestClient) ports.ExecutionGateway {
		return natsexecution.NewGateway(rc, "gateway.http")
	})
	connectOptional("execution", cl, p)

	conns.executionControl, cl, p = newGatewayConn(config, "execution-control", func(rc *natskit.NATSRequestClient) ports.ExecutionControlGateway {
		return natsexecution.NewControlGateway(rc, "gateway.http")
	})
	connectOptional("execution control", cl, p)

	// S460: Session metadata gateway.
	conns.session, cl, p = newGatewayConn(config, "session", func(rc *natskit.NATSRequestClient) ports.SessionGateway {
		return natsexecution.NewSessionGateway(rc, "gateway.http")
	})
	connectOptional("session", cl, p)

	// H-8.a: insights read gateway — KV-direct reader (not request/
	// reply). The gateway is a free KV reader (ADR-0008). Degrades
	// gracefully: a failed KV connect disables the insights endpoint.
	insightsKV := natsinsights.NewVolumeProfileKVStore(config.NATS.URL)
	tpoKV := natsinsights.NewTPOKVStore(config.NATS.URL)
	vpOK := insightsKV.Start()
	if vpOK != nil {
		logger.Warn("volume profile KV reader unavailable", "error", vpOK)
		insightsKV = nil
	} else {
		addCloser(insightsKV.Close)
	}
	tpoErr := tpoKV.Start()
	if tpoErr != nil {
		logger.Warn("tpo KV reader unavailable", "error", tpoErr)
		tpoKV = nil
	} else {
		addCloser(tpoKV.Close)
	}
	if insightsKV != nil || tpoKV != nil {
		conns.insights = natsinsights.NewGateway(insightsKV, tpoKV)
	}

	return conns, nil
}

// buildAnalyticalClient creates an optional ClickHouse client for analytical queries.
// Returns nil if ClickHouse is not configured. The gateway does NOT add ClickHouse
// to its readiness check — analytical endpoints simply return 503 when unavailable.
//
// When ClickHouse IS configured, the config is validated before attempting a connection.
// Invalid config (e.g. empty database) disables analytical endpoints with a warning
// rather than attempting a connection that would fail with an opaque error.
func buildAnalyticalClient(config settings.AppConfig, logger *slog.Logger) *clickhouse.Client {
	if config.ClickHouse.Addr == "" {
		logger.Info("clickhouse not configured, analytical endpoints disabled")
		return nil
	}

	// Validate config before opening connection — fail-fast with actionable message.
	if prob := config.ClickHouse.Validate(); prob != nil {
		logger.Warn("clickhouse config invalid, analytical endpoints disabled", "error", prob)
		return nil
	}

	client, err := clickhouse.Open(clickhouse.Config{
		Addr:     config.ClickHouse.Addr,
		Database: config.ClickHouse.Database,
		Username: config.ClickHouse.Username,
		Password: config.ClickHouse.Password,
	})
	if err != nil {
		logger.Warn("clickhouse connection failed, analytical endpoints disabled", "addr", config.ClickHouse.Addr, "error", err)
		return nil
	}

	logger.Info("clickhouse connected, analytical endpoints enabled",
		"addr", config.ClickHouse.Addr,
		"database", config.ClickHouse.Database,
	)
	return client
}

// buildRouteDependencies wires use cases from gateway connections and assembles
// the complete route dependency set for the gateway HTTP server.
// Returns route dependencies, the concrete VerifySessionUseCase (nil if not wired),
// and the GenerateUnifiedReportUseCase (nil if not wired).
// S490: The verify UC is also used by the event-driven trigger.
// S491: The unified report UC is also used by the event-driven trigger.
func buildRouteDependencies(config settings.AppConfig, conns *gatewayConns, chClient *clickhouse.Client, logger *slog.Logger) (routes.Dependencies, *executionclient.VerifySessionUseCase, *executionclient.GenerateUnifiedReportUseCase) {
	deps := routes.Dependencies{
		Readiness: newGatewayReadinessChecker(config, conns.configctl, conns.evidence),
	}

	// Configctl use cases — always wired (gateway requires NATS).
	deps.CreateDraft = configctlclient.NewCreateDraftUseCase(conns.configctl)
	deps.GetConfig = configctlclient.NewGetConfigUseCase(conns.configctl)
	deps.GetActive = configctlclient.NewGetActiveConfigUseCase(conns.configctl)
	deps.ListActiveRuntimeProjections = configctlclient.NewListActiveRuntimeProjectionsUseCase(conns.configctl)
	deps.ListActiveIngestionBindings = configctlclient.NewListActiveIngestionBindingsUseCase(conns.configctl)
	deps.ListConfigs = configctlclient.NewListConfigsUseCase(conns.configctl)
	deps.ValidateDraft = configctlclient.NewValidateDraftUseCase(conns.configctl)
	deps.ValidateConfig = configctlclient.NewValidateConfigUseCase(conns.configctl)
	deps.CompileConfig = configctlclient.NewCompileConfigUseCase(conns.configctl)
	deps.ActivateConfig = configctlclient.NewActivateConfigUseCase(conns.configctl)

	// Evidence use cases — conditional on gateway availability.
	if conns.evidence != nil {
		deps.Evidence = routes.EvidenceFamilyDeps{
			GetLatestCandle:     evidenceclient.NewGetLatestCandleUseCase(conns.evidence),
			GetCandleHistory:    evidenceclient.NewGetCandleHistoryUseCase(conns.evidence),
			GetLatestTradeBurst: evidenceclient.NewGetLatestTradeBurstUseCase(conns.evidence),
			GetLatestVolume:     evidenceclient.NewGetLatestVolumeUseCase(conns.evidence),
		}
	}

	// Signal use cases.
	if conns.signal != nil {
		deps.Signal = routes.SignalFamilyDeps{
			GetLatestSignal: signalclient.NewGetLatestSignalUseCase(conns.signal),
		}
	}

	// Decision use cases.
	if conns.decision != nil {
		deps.Decision = routes.DecisionFamilyDeps{
			GetLatestDecision: decisionclient.NewGetLatestDecisionUseCase(conns.decision),
		}
	}

	// Strategy use cases.
	if conns.strategy != nil {
		deps.Strategy = routes.StrategyFamilyDeps{
			GetLatestStrategy: strategyclient.NewGetLatestStrategyUseCase(conns.strategy),
		}
	}

	// Risk use cases.
	if conns.risk != nil {
		deps.Risk = routes.RiskFamilyDeps{
			GetLatestRisk: riskclient.NewGetLatestRiskUseCase(conns.risk),
		}
	}

	// Execution use cases — wired from two separate gateways.
	if conns.execution != nil || conns.executionControl != nil {
		execDeps := routes.ExecutionFamilyDeps{}
		if conns.execution != nil {
			execDeps.GetLatestExecution = executionclient.NewGetLatestExecutionUseCase(conns.execution)
			execDeps.GetExecutionStatus = executionclient.NewGetExecutionStatusUseCase(conns.execution)
			execDeps.GetLifecycleList = executionclient.NewGetLifecycleListUseCase(conns.execution)
		}
		if conns.executionControl != nil {
			execDeps.GetExecutionControl = executionclient.NewGetExecutionControlUseCase(conns.executionControl)
			execDeps.SetExecutionControl = executionclient.NewSetExecutionControlUseCase(conns.executionControl)
		}
		deps.Execution = execDeps
	}

	// Activation surface — wired from execution control gateway.
	if conns.executionControl != nil {
		deps.Activation = routes.ActivationFamilyDeps{
			GetActivationSurface: executionclient.NewGetActivationSurfaceUseCase(conns.executionControl),
		}
	}

	// S460: Session use cases — conditional on session gateway availability.
	// S462: Audit bundle wired when both session gateway and execution gateway are available.
	// S465: Verification and fill reader wired when ClickHouse is available.
	if conns.session != nil {
		getSessionUC := executionclient.NewGetSessionUseCase(conns.session)

		sessionDeps := routes.SessionFamilyDeps{
			GetSession:   getSessionUC,
			ListSessions: executionclient.NewListSessionsUseCase(conns.session),
		}

		// S465: Build ClickHouse-backed readers for verification and audit.
		var chSummary executionclient.VerifyCHSummary
		var chLister executionclient.VerifyCHLister
		var auditFillReader executionclient.AuditCHFillReader
		if chClient != nil {
			chExecReader := clickhouse.NewExecutionReader(chClient, logger)
			chSummary = newSessionCHSummaryAdapter(chExecReader, logger)
			listerAdapter := newSessionCHListerAdapter(chExecReader, logger)
			chLister = listerAdapter
			auditFillReader = listerAdapter
		}

		// S465: Build gate reader for verification (needs execution control gateway).
		var gateReader executionclient.VerifyGateReader
		if conns.executionControl != nil {
			gateReader = executionclient.NewGetExecutionControlUseCase(conns.executionControl)
		}

		// S465: Wire verification use case — closes G3 from S463.
		// H-4: thread the production Clock port into the use case;
		// consumed in commit 6a (DefaultVerificationScope migration).
		verifyUC := executionclient.NewVerifySessionUseCase(
			getSessionUC,
			gateReader,
			chSummary,
			chLister,
			nil, // consistency checker requires cross-surface reads not yet available
		).WithClock(clock.SystemClock{})
		sessionDeps.VerifySession = verifyUC

		// S465: Audit bundle with verification and fill reader — closes G4 from S463.
		var auditLifecycleReader executionclient.AuditLifecycleReader
		if conns.execution != nil {
			auditLifecycleReader = executionclient.NewGetLifecycleListUseCase(conns.execution)
		}
		auditUC := executionclient.NewAuditSessionUseCase(
			getSessionUC,
			verifyUC,
			auditLifecycleReader,
			auditFillReader,
		)
		sessionDeps.AuditSession = auditUC

		// S467: Batch audit wired from list + single audit use cases.
		sessionDeps.BatchAuditSession = executionclient.NewBatchAuditSessionUseCase(
			sessionDeps.ListSessions,
			auditUC,
		)

		deps.Session = sessionDeps
	}

	// Analytical use cases — conditional on ClickHouse availability.
	// These are additive endpoints (R-08) that do not modify existing behavior.
	if chClient != nil {
		candleReader := newAnalyticalCandleReader(chClient, logger)
		signalReader := newAnalyticalSignalReader(chClient, logger)
		decisionReader := newAnalyticalDecisionReader(chClient, logger)
		strategyReader := newAnalyticalStrategyReader(chClient, logger)
		riskReader := newAnalyticalRiskReader(chClient, logger)
		chExecutionReader := clickhouse.NewExecutionReader(chClient, logger)
		lifecycleReader := newAnalyticalLifecycleReader(chClient, logger)
		compositeReader := newAnalyticalCompositeReader(chClient, logger)
		// S455A: Session explain needs both ClickHouse (lifecycle history) and KV (execution status).
		// The KV reader is the execution status use case — nil if NATS execution gateway unavailable.
		var sessionExplainKVReader analyticalclient.SessionExplainKVReader
		if conns.execution != nil {
			sessionExplainKVReader = executionclient.NewGetExecutionStatusUseCase(conns.execution)
		}

		deps.Analytical = routes.AnalyticalFamilyDeps{
			GetCandleHistory:        analyticalclient.NewGetCandleHistoryUseCase(candleReader, logger),
			GetSignalHistory:        analyticalclient.NewGetSignalHistoryUseCase(signalReader, logger),
			GetDecisionHistory:      analyticalclient.NewGetDecisionHistoryUseCase(decisionReader, logger),
			GetStrategyHistory:      analyticalclient.NewGetStrategyHistoryUseCase(strategyReader, logger),
			GetRiskHistory:          analyticalclient.NewGetRiskHistoryUseCase(riskReader, logger),
			GetExecutionHistory:     analyticalclient.NewGetExecutionHistoryUseCase(chExecutionReader, logger),
			GetLifecycleHistory:     analyticalclient.NewGetLifecycleHistoryUseCase(lifecycleReader, logger),
			GetExecutionList:        analyticalclient.NewGetExecutionListUseCase(chExecutionReader, logger),
			GetExecutionSummary:     analyticalclient.NewGetExecutionSummaryUseCase(chExecutionReader, logger),
			GetSessionExplain:       analyticalclient.NewGetSessionExplainUseCase(lifecycleReader, sessionExplainKVReader, logger),
			GetCompositeChain:       analyticalclient.NewGetCompositeChainUseCase(compositeReader, logger),
			GetPipelineFunnel:       analyticalclient.NewGetPipelineFunnelUseCase(compositeReader, logger),
			GetDispositionBreakdown: analyticalclient.NewGetDispositionBreakdownUseCase(compositeReader, logger),
			GetDecisionReview:       analyticalclient.NewGetDecisionReviewUseCase(compositeReader, logger),
			GetEffectiveness:        analyticalclient.NewGetEffectivenessUseCase(compositeReader, logger),
			GetEffectivenessSummary: analyticalclient.NewGetEffectivenessSummaryUseCase(compositeReader, logger),
			GetPairing:              analyticalclient.NewGetPairingUseCase(compositeReader, logger),
			GetRoundTripReview:      analyticalclient.NewGetRoundTripReviewUseCase(compositeReader, logger),
		}

		// S495: Cross-session pairing — requires both ClickHouse (chains) and session gateway (KV).
		if conns.session != nil {
			sessionAdapter := &crossSessionSessionAdapter{listUC: deps.Session.ListSessions}
			deps.Analytical.GetCrossSessionPairing = analyticalclient.NewGetCrossSessionPairingUseCase(
				sessionAdapter,
				compositeReader,
				logger,
			)
			// S496: Continuity review — requires both ClickHouse (chains) and session gateway (KV).
			deps.Analytical.GetContinuityReview = analyticalclient.NewGetContinuityReviewUseCase(
				sessionAdapter,
				compositeReader,
				logger,
			)
		}
	}

	// S486: Monitoring use case — always wired with static surface availability.
	// Session lister and gate reader are optional — degrade gracefully.
	surfaces := monitoring.SurfaceAvailability{
		Evidence:   deps.Evidence.HasAny(),
		Signal:     deps.Signal.HasAny(),
		Decision:   deps.Decision.HasAny(),
		Strategy:   deps.Strategy.HasAny(),
		Risk:       deps.Risk.HasAny(),
		Execution:  deps.Execution.HasAny(),
		Session:    deps.Session.HasAny(),
		Analytical: deps.Analytical.HasAny(),
		Activation: deps.Activation.HasAny(),
	}

	var monSessionLister monitoringclient.SessionLister
	if deps.Session.ListSessions != nil {
		monSessionLister = deps.Session.ListSessions
	}
	var monGateReader monitoringclient.GateReader
	if deps.Execution.GetExecutionControl != nil {
		monGateReader = deps.Execution.GetExecutionControl
	}

	deps.Monitoring = routes.MonitoringFamilyDeps{
		GetOperationalState: monitoringclient.NewGetOperationalStateUseCase(
			monSessionLister,
			monGateReader,
			surfaces,
		),
	}

	// S487: Triage use cases — wired from session batch audit + analytical surfaces.
	var sessionTriageUC *triageclient.GetSessionTriageUseCase
	if deps.Session.BatchAuditSession != nil {
		sessionTriageUC = triageclient.NewGetSessionTriageUseCase(deps.Session.BatchAuditSession)
	}
	var decisionTriageUC *triageclient.GetDecisionTriageUseCase
	if deps.Analytical.GetDecisionReview != nil {
		decisionTriageUC = triageclient.NewGetDecisionTriageUseCase(deps.Analytical.GetDecisionReview)
	}
	var roundTripTriageUC *triageclient.GetRoundTripTriageUseCase
	if deps.Analytical.GetRoundTripReview != nil {
		roundTripTriageUC = triageclient.NewGetRoundTripTriageUseCase(deps.Analytical.GetRoundTripReview)
	}

	triageOverviewUC := triageclient.NewGetTriageOverviewUseCase(
		sessionTriageUC,
		decisionTriageUC,
		roundTripTriageUC,
	)

	deps.Triage = routes.TriageFamilyDeps{
		GetSessionTriage:   sessionTriageUC,
		GetDecisionTriage:  decisionTriageUC,
		GetRoundTripTriage: roundTripTriageUC,
		GetTriageOverview:  triageOverviewUC,
	}

	// Venue capabilities introspection (ADR-0022 R2, H-7.a) —
	// always wired: the declarations are static and ship with the
	// adapters compiled into the binary, no connection required.
	deps.Venues = routes.VenuesFamilyDeps{
		Capabilities: []ports.Capabilities{
			binances.Capabilities(),
			binancef.Capabilities(),
			bybits.Capabilities(),
			bybitf.Capabilities(),
		},
	}

	// Insights read surface (ADR-0027 / H-8.a) — KV-direct.
	if conns.insights != nil {
		deps.Insights = routes.InsightsFamilyDeps{
			GetLatestVolumeProfile: insightsclient.NewGetLatestVolumeProfileUseCase(conns.insights),
			GetLatestTPOProfile:    insightsclient.NewGetLatestTPOProfileUseCase(conns.insights),
		}
	}

	deps.Logger = logger

	// S490: Extract concrete VerifySessionUseCase for event-driven trigger.
	var verifyUC *executionclient.VerifySessionUseCase
	if v, ok := deps.Session.VerifySession.(*executionclient.VerifySessionUseCase); ok {
		verifyUC = v
	}

	// S491: Wire unified report use case — needs verify, audit, monitoring, triage.
	// Extract concrete audit UC from deps.
	var auditUC *executionclient.AuditSessionUseCase
	if a, ok := deps.Session.AuditSession.(*executionclient.AuditSessionUseCase); ok {
		auditUC = a
	}

	var monReader executionclient.UnifiedReportMonitoringReader
	if monUC, ok := deps.Monitoring.GetOperationalState.(*monitoringclient.GetOperationalStateUseCase); ok && monUC != nil {
		monReader = &monitoringReportAdapter{uc: monUC}
	}
	var triageReader executionclient.UnifiedReportTriageReader
	if triageUC, ok := deps.Triage.GetTriageOverview.(*triageclient.GetTriageOverviewUseCase); ok && triageUC != nil {
		triageReader = &triageReportAdapter{uc: triageUC}
	}

	reportUC := executionclient.NewGenerateUnifiedReportUseCase(verifyUC, auditUC, monReader, triageReader)
	deps.Session.UnifiedReport = reportUC

	return deps, verifyUC, reportUC
}
