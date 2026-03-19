package main

import (
	"log/slog"

	adapternats "internal/adapters/nats"
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
	gw, closeFn, prob := newGatewayConn(config, "configctl", func(rc *adapternats.NATSRequestClient) ports.ConfigctlGateway {
		return adapternats.NewConfigctlGateway(rc, "gateway.http")
	})
	if prob != nil {
		return nil, prob
	}
	conns.configctl = gw
	addCloser(closeFn)

	// Optional domain gateways — each creates its own NATS request client.
	var cl func() error
	var p *problem.Problem

	conns.evidence, cl, p = newGatewayConn(config, "evidence", func(rc *adapternats.NATSRequestClient) ports.EvidenceGateway {
		return adapternats.NewEvidenceGateway(rc, "gateway.http")
	})
	connectOptional("evidence", cl, p)

	conns.signal, cl, p = newGatewayConn(config, "signal", func(rc *adapternats.NATSRequestClient) ports.SignalGateway {
		return adapternats.NewSignalGateway(rc, "gateway.http")
	})
	connectOptional("signal", cl, p)

	conns.decision, cl, p = newGatewayConn(config, "decision", func(rc *adapternats.NATSRequestClient) ports.DecisionGateway {
		return adapternats.NewDecisionGateway(rc, "gateway.http")
	})
	connectOptional("decision", cl, p)

	conns.strategy, cl, p = newGatewayConn(config, "strategy", func(rc *adapternats.NATSRequestClient) ports.StrategyGateway {
		return adapternats.NewStrategyGateway(rc, "gateway.http")
	})
	connectOptional("strategy", cl, p)

	conns.risk, cl, p = newGatewayConn(config, "risk", func(rc *adapternats.NATSRequestClient) ports.RiskGateway {
		return adapternats.NewRiskGateway(rc, "gateway.http")
	})
	connectOptional("risk", cl, p)

	conns.execution, cl, p = newGatewayConn(config, "execution", func(rc *adapternats.NATSRequestClient) ports.ExecutionGateway {
		return adapternats.NewExecutionGateway(rc, "gateway.http")
	})
	connectOptional("execution", cl, p)

	conns.executionControl, cl, p = newGatewayConn(config, "execution-control", func(rc *adapternats.NATSRequestClient) ports.ExecutionControlGateway {
		return adapternats.NewExecutionControlGateway(rc, "gateway.http")
	})
	connectOptional("execution control", cl, p)

	return conns, nil
}

// buildRouteDependencies wires use cases from gateway connections and assembles
// the complete route dependency set for the gateway HTTP server.
func buildRouteDependencies(config settings.AppConfig, conns *gatewayConns) routes.Dependencies {
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

	return deps
}
