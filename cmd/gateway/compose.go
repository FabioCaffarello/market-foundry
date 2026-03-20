package main

import (
	"log/slog"

	"internal/adapters/clickhouse"
	natsconfigctl "internal/adapters/nats/natsconfigctl"
	natsdecision "internal/adapters/nats/natsdecision"
	natsevidence "internal/adapters/nats/natsevidence"
	natsexecution "internal/adapters/nats/natsexecution"
	natskit "internal/adapters/nats/natskit"
	natsrisk "internal/adapters/nats/natsrisk"
	natssignal "internal/adapters/nats/natssignal"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/application/analyticalclient"
	configctlclient "internal/application/configctlclient"
	"internal/application/decisionclient"
	"internal/application/evidenceclient"
	"internal/application/executionclient"
	"internal/application/ports"
	"internal/application/riskclient"
	"internal/application/signalclient"
	"internal/application/strategyclient"
	"internal/interfaces/http/routes"
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
func buildRouteDependencies(config settings.AppConfig, conns *gatewayConns, chClient *clickhouse.Client, logger *slog.Logger) routes.Dependencies {
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
		}
		if conns.executionControl != nil {
			execDeps.GetExecutionControl = executionclient.NewGetExecutionControlUseCase(conns.executionControl)
			execDeps.SetExecutionControl = executionclient.NewSetExecutionControlUseCase(conns.executionControl)
		}
		deps.Execution = execDeps
	}

	// Analytical use cases — conditional on ClickHouse availability.
	// These are additive endpoints (R-08) that do not modify existing behavior.
	if chClient != nil {
		candleReader := newAnalyticalCandleReader(chClient, logger)
		signalReader := newAnalyticalSignalReader(chClient, logger)
		decisionReader := newAnalyticalDecisionReader(chClient, logger)
		strategyReader := newAnalyticalStrategyReader(chClient, logger)
		riskReader := newAnalyticalRiskReader(chClient, logger)
		executionReader := newAnalyticalExecutionReader(chClient, logger)
		deps.Analytical = routes.AnalyticalFamilyDeps{
			GetCandleHistory:    analyticalclient.NewGetCandleHistoryUseCase(candleReader, logger),
			GetSignalHistory:    analyticalclient.NewGetSignalHistoryUseCase(signalReader, logger),
			GetDecisionHistory:  analyticalclient.NewGetDecisionHistoryUseCase(decisionReader, logger),
			GetStrategyHistory:  analyticalclient.NewGetStrategyHistoryUseCase(strategyReader, logger),
			GetRiskHistory:      analyticalclient.NewGetRiskHistoryUseCase(riskReader, logger),
			GetExecutionHistory: analyticalclient.NewGetExecutionHistoryUseCase(executionReader, logger),
		}
	}

	deps.Logger = logger

	return deps
}
