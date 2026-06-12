package routes

import (
	"context"
	"log/slog"
	"net/http"

	configctlcontracts "internal/application/configctl/contracts"
	"internal/application/decisionclient"
	"internal/application/evidenceclient"
	"internal/application/executionclient"
	"internal/application/riskclient"
	"internal/application/signalclient"
	"internal/application/strategyclient"
	"internal/interfaces/http/handlers"
	"internal/shared/metrics"
	"internal/shared/problem"
	"internal/shared/webserver"
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

// ExecutionFamilyDeps groups execution query and control use cases.
// Adding a new execution operation means adding one field here and one route block in Execution().
type ExecutionFamilyDeps struct {
	GetLatestExecution  handlersGetLatestExecutionUseCase
	GetExecutionStatus  handlersGetExecutionStatusUseCase
	GetExecutionControl handlersGetExecutionControlUseCase
	SetExecutionControl handlersSetExecutionControlUseCase
	GetLifecycleList    handlersGetLifecycleListUseCase
}

// HasAny reports whether at least one execution use case is available.
func (e ExecutionFamilyDeps) HasAny() bool {
	return e.GetLatestExecution != nil || e.GetExecutionStatus != nil || e.GetExecutionControl != nil || e.GetLifecycleList != nil
}

// ActivationFamilyDeps groups activation surface query use cases.
type ActivationFamilyDeps struct {
	GetActivationSurface handlersGetActivationSurfaceUseCase
}

// HasAny reports whether at least one activation use case is available.
func (a ActivationFamilyDeps) HasAny() bool {
	return a.GetActivationSurface != nil
}

// SourceExplainFamilyDeps groups source-driven path explainability use cases.
// S361: Exposes the composite source explain endpoint.
type SourceExplainFamilyDeps struct {
	GetSourceExplanation handlersGetSourceExplanationUseCase
}

// HasAny reports whether the source explain use case is available.
func (s SourceExplainFamilyDeps) HasAny() bool {
	return s.GetSourceExplanation != nil
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
	Execution                    ExecutionFamilyDeps
	Activation                   ActivationFamilyDeps
	SourceExplain                SourceExplainFamilyDeps
	Analytical                   AnalyticalFamilyDeps
	Session                      SessionFamilyDeps
	Monitoring                   MonitoringFamilyDeps
	Triage                       TriageFamilyDeps // S487
	Venues                       VenuesFamilyDeps // ADR-0022 R2 (H-7.a)
	Logger                       *slog.Logger
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

type handlersGetLatestExecutionUseCase interface {
	Execute(context.Context, executionclient.ExecutionLatestQuery) (executionclient.ExecutionLatestReply, *problem.Problem)
}

type handlersGetExecutionStatusUseCase interface {
	Execute(context.Context, executionclient.ExecutionStatusQuery) (executionclient.ExecutionStatusReply, *problem.Problem)
}

type handlersGetExecutionControlUseCase interface {
	Execute(context.Context, executionclient.ExecutionControlQuery) (executionclient.ExecutionControlReply, *problem.Problem)
}

type handlersSetExecutionControlUseCase interface {
	Execute(context.Context, executionclient.SetExecutionControlCommand) (executionclient.ExecutionControlReply, *problem.Problem)
}

type handlersGetLifecycleListUseCase interface {
	Execute(context.Context, executionclient.LifecycleListQuery) (executionclient.LifecycleListReply, *problem.Problem)
}

type handlersGetActivationSurfaceUseCase interface {
	Execute(context.Context, executionclient.ActivationSurfaceQuery) (executionclient.ActivationSurfaceReply, *problem.Problem)
}

type handlersGetSourceExplanationUseCase interface {
	Execute(context.Context, executionclient.SourceExplainQuery) (executionclient.SourceExplainReply, *problem.Problem)
}

// S460: Session query use case interfaces.
type handlersGetSessionUseCase interface {
	Execute(context.Context, executionclient.SessionGetQuery) (executionclient.SessionGetReply, *problem.Problem)
}

type handlersListSessionsUseCase interface {
	Execute(context.Context, executionclient.SessionListQuery) (executionclient.SessionListReply, *problem.Problem)
}

// S461: Session verification use case interface.
type handlersVerifySessionUseCase interface {
	Execute(context.Context, executionclient.SessionVerifyQuery) (executionclient.SessionVerifyReply, *problem.Problem)
}

// S462: Session audit bundle use case interface.
type handlersAuditSessionUseCase interface {
	Execute(context.Context, executionclient.SessionAuditQuery) (executionclient.SessionAuditReply, *problem.Problem)
}

// S467: Batch audit use case interface.
type handlersBatchAuditSessionUseCase interface {
	Execute(context.Context, executionclient.SessionBatchAuditQuery) (executionclient.SessionBatchAuditReply, *problem.Problem)
}

// S491: Unified operational report use case interface.
type handlersUnifiedReportUseCase interface {
	Execute(context.Context, executionclient.SessionUnifiedReportQuery) (executionclient.SessionUnifiedReportReply, *problem.Problem)
}

// SessionFamilyDeps groups session query use cases.
type SessionFamilyDeps struct {
	GetSession        handlersGetSessionUseCase
	ListSessions      handlersListSessionsUseCase
	VerifySession     handlersVerifySessionUseCase     // S461
	AuditSession      handlersAuditSessionUseCase      // S462
	BatchAuditSession handlersBatchAuditSessionUseCase // S467
	UnifiedReport     handlersUnifiedReportUseCase     // S491
}

// HasAny reports whether at least one session use case is available.
func (s SessionFamilyDeps) HasAny() bool {
	return s.GetSession != nil || s.ListSessions != nil || s.VerifySession != nil || s.AuditSession != nil || s.BatchAuditSession != nil || s.UnifiedReport != nil
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
	if deps.Execution.HasAny() {
		routes = append(routes, Execution(deps.Execution)...)
	}
	if deps.Activation.HasAny() {
		routes = append(routes, Activation(deps.Activation)...)
	}
	if deps.SourceExplain.HasAny() {
		routes = append(routes, SourceExplain(deps.SourceExplain)...)
	}
	if deps.Analytical.HasAny() {
		routes = append(routes, Analytical(deps.Analytical, deps.Logger)...)
	}
	if deps.Session.HasAny() {
		routes = append(routes, Session(deps.Session)...)
	}
	if deps.Monitoring.HasAny() {
		routes = append(routes, Monitoring(deps.Monitoring)...)
	}
	if deps.Triage.HasAny() {
		routes = append(routes, Triage(deps.Triage, deps.Logger)...)
	}
	if deps.Venues.HasAny() {
		routes = append(routes, Venues(deps.Venues)...)
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
		{
			Method:  http.MethodGet,
			Path:    "/metrics",
			Handler: metrics.HandlerFunc(),
		},
	}
}
