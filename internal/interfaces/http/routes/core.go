package routes

import (
	"context"
	"net/http"

	configctlcontracts "internal/application/configctl/contracts"
	"internal/application/decisionclient"
	"internal/application/evidenceclient"
	"internal/application/riskclient"
	"internal/application/signalclient"
	"internal/application/strategyclient"
	"internal/interfaces/http/handlers"
	"internal/interfaces/http/webserver"
	"internal/shared/problem"
)

// EvidenceFamilyDeps groups evidence query use cases by projection family.
// Adding a new evidence type means adding one field here and one route block
// in Evidence(). The gateway does not need to know how the store materializes
// these projections — it only holds use case references.
type EvidenceFamilyDeps struct {
	// Candle family — latest + history
	GetLatestCandle  handlersGetLatestCandleUseCase
	GetCandleHistory handlersGetCandleHistoryUseCase
	// TradeBurst family — latest only
	GetLatestTradeBurst handlersGetLatestTradeBurstUseCase
	// Volume family — latest only
	GetLatestVolume handlersGetLatestVolumeUseCase
}

// HasAny reports whether at least one evidence use case is available.
func (e EvidenceFamilyDeps) HasAny() bool {
	return e.GetLatestCandle != nil || e.GetCandleHistory != nil || e.GetLatestTradeBurst != nil || e.GetLatestVolume != nil
}

// SignalFamilyDeps groups signal query use cases.
// Adding a new signal operation means adding one field here and one route block in Signal().
type SignalFamilyDeps struct {
	GetLatestSignal handlersGetLatestSignalUseCase
}

// HasAny reports whether at least one signal use case is available.
func (s SignalFamilyDeps) HasAny() bool {
	return s.GetLatestSignal != nil
}

// DecisionFamilyDeps groups decision query use cases.
// Adding a new decision operation means adding one field here and one route block in Decision().
type DecisionFamilyDeps struct {
	GetLatestDecision handlersGetLatestDecisionUseCase
}

// HasAny reports whether at least one decision use case is available.
func (d DecisionFamilyDeps) HasAny() bool {
	return d.GetLatestDecision != nil
}

// StrategyFamilyDeps groups strategy query use cases.
// Adding a new strategy operation means adding one field here and one route block in Strategy().
type StrategyFamilyDeps struct {
	GetLatestStrategy handlersGetLatestStrategyUseCase
}

// HasAny reports whether at least one strategy use case is available.
func (s StrategyFamilyDeps) HasAny() bool {
	return s.GetLatestStrategy != nil
}

// RiskFamilyDeps groups risk query use cases.
// Adding a new risk operation means adding one field here and one route block in Risk().
type RiskFamilyDeps struct {
	GetLatestRisk handlersGetLatestRiskUseCase
}

// HasAny reports whether at least one risk use case is available.
func (r RiskFamilyDeps) HasAny() bool {
	return r.GetLatestRisk != nil
}

type Dependencies struct {
	Readiness                    handlers.ReadinessChecker
	CreateDraft                  handlersCreateDraftUseCase
	GetConfig                    handlersGetConfigUseCase
	GetActive                    handlersGetActiveConfigUseCase
	ListActiveRuntimeProjections handlersListActiveRuntimeProjectionsUseCase
	ListActiveIngestionBindings  handlersListActiveIngestionBindingsUseCase
	ListConfigs                  handlersListConfigsUseCase
	ValidateDraft                handlersValidateDraftUseCase
	ValidateConfig               handlersValidateConfigUseCase
	CompileConfig                handlersCompileConfigUseCase
	ActivateConfig               handlersActivateConfigUseCase
	Evidence                     EvidenceFamilyDeps
	Signal                       SignalFamilyDeps
	Decision                     DecisionFamilyDeps
	Strategy                     StrategyFamilyDeps
	Risk                         RiskFamilyDeps
}

type handlersCreateDraftUseCase interface {
	Execute(context.Context, configctlcontracts.CreateDraftCommand) (configctlcontracts.CreateDraftReply, *problem.Problem)
}

type handlersGetConfigUseCase interface {
	Execute(context.Context, configctlcontracts.GetConfigQuery) (configctlcontracts.GetConfigReply, *problem.Problem)
}

type handlersGetActiveConfigUseCase interface {
	Execute(context.Context, configctlcontracts.GetActiveConfigQuery) (configctlcontracts.GetActiveConfigReply, *problem.Problem)
}

type handlersListConfigsUseCase interface {
	Execute(context.Context, configctlcontracts.ListConfigsQuery) (configctlcontracts.ListConfigsReply, *problem.Problem)
}

type handlersListActiveRuntimeProjectionsUseCase interface {
	Execute(context.Context, configctlcontracts.ListActiveRuntimeProjectionsQuery) (configctlcontracts.ListActiveRuntimeProjectionsReply, *problem.Problem)
}

type handlersListActiveIngestionBindingsUseCase interface {
	Execute(context.Context, configctlcontracts.ListActiveIngestionBindingsQuery) (configctlcontracts.ListActiveIngestionBindingsReply, *problem.Problem)
}

type handlersValidateDraftUseCase interface {
	Execute(context.Context, configctlcontracts.ValidateDraftCommand) (configctlcontracts.ValidateDraftReply, *problem.Problem)
}

type handlersValidateConfigUseCase interface {
	Execute(context.Context, configctlcontracts.ValidateConfigCommand) (configctlcontracts.ValidateConfigReply, *problem.Problem)
}

type handlersCompileConfigUseCase interface {
	Execute(context.Context, configctlcontracts.CompileConfigCommand) (configctlcontracts.CompileConfigReply, *problem.Problem)
}

type handlersActivateConfigUseCase interface {
	Execute(context.Context, configctlcontracts.ActivateConfigCommand) (configctlcontracts.ActivateConfigReply, *problem.Problem)
}

type handlersGetLatestCandleUseCase interface {
	Execute(context.Context, evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem)
}

type handlersGetCandleHistoryUseCase interface {
	Execute(context.Context, evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem)
}

type handlersGetLatestTradeBurstUseCase interface {
	Execute(context.Context, evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem)
}

type handlersGetLatestVolumeUseCase interface {
	Execute(context.Context, evidenceclient.VolumeLatestQuery) (evidenceclient.VolumeLatestReply, *problem.Problem)
}

type handlersGetLatestSignalUseCase interface {
	Execute(context.Context, signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem)
}

type handlersGetLatestDecisionUseCase interface {
	Execute(context.Context, decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem)
}

type handlersGetLatestStrategyUseCase interface {
	Execute(context.Context, strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem)
}

type handlersGetLatestRiskUseCase interface {
	Execute(context.Context, riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem)
}

func DefaultRoutes(deps Dependencies) []webserver.Route {
	readiness := deps.Readiness
	if readiness == nil {
		readiness = handlers.NewAlwaysReadyChecker()
	}

	routes := Core(readiness)
	routes = append(routes, Configctl(
		deps.CreateDraft,
		deps.GetConfig,
		deps.GetActive,
		deps.ListConfigs,
		deps.ValidateDraft,
		deps.ValidateConfig,
		deps.CompileConfig,
		deps.ActivateConfig,
	)...)
	if deps.Evidence.HasAny() {
		routes = append(routes, Evidence(deps.Evidence)...)
	}
	if deps.Signal.HasAny() {
		routes = append(routes, Signal(deps.Signal)...)
	}
	if deps.Decision.HasAny() {
		routes = append(routes, Decision(deps.Decision)...)
	}
	if deps.Strategy.HasAny() {
		routes = append(routes, Strategy(deps.Strategy)...)
	}
	if deps.Risk.HasAny() {
		routes = append(routes, Risk(deps.Risk)...)
	}
	return routes
}

func Core(readiness handlers.ReadinessChecker) []webserver.Route {
	healthzHandler := handlers.NewHealthzWebHandler()
	readyzHandler := handlers.NewReadyzWebHandler(readiness)

	return []webserver.Route{
		{
			Method:  http.MethodGet,
			Path:    "/healthz",
			Handler: healthzHandler.Healthz,
		},
		{
			Method:  http.MethodGet,
			Path:    "/readyz",
			Handler: readyzHandler.Readyz,
		},
	}
}
