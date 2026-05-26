package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsdecision "internal/adapters/nats/natsdecision"
	natsevidence "internal/adapters/nats/natsevidence"
	natsexecution "internal/adapters/nats/natsexecution"
	natsrisk "internal/adapters/nats/natsrisk"
	natssignal "internal/adapters/nats/natssignal"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// FamilyProcessor describes one evidence family's processing pipeline within derive.
// Each family gets one sampler actor per symbol/timeframe combination.
// This is the canonical unit of registration — adding a new evidence type means
// adding one FamilyProcessor entry in the supervisor, not modifying SourceScopeActor.
type FamilyProcessor struct {
	// Family is the canonical family name (e.g., "candle", "tradeburst").
	Family string
	// ActorPrefix is the name prefix for sampler actors (e.g., "sampler", "burst-sampler").
	ActorPrefix string
	// NewActor creates the sampler actor for this family given source, symbol, timeframe,
	// evidence publisher PID, and scope PID (for signal fan-out).
	NewActor func(source, symbol string, timeframe time.Duration, publisherPID, scopePID *actor.PID) actor.Producer
}

// SignalFamilyProcessor describes one signal family's processing pipeline within derive.
// Each signal family gets one sampler actor per symbol/timeframe combination.
type SignalFamilyProcessor struct {
	Family      string
	ActorPrefix string
	NewActor    func(source, symbol string, timeframe time.Duration, signalPublisherPID, scopePID *actor.PID) actor.Producer
}

// DecisionFamilyProcessor describes one decision family's processing pipeline within derive.
// Each decision family gets one evaluator actor per symbol/timeframe combination.
type DecisionFamilyProcessor struct {
	Family      string
	ActorPrefix string
	NewActor    func(source, symbol string, timeframe time.Duration, decisionPublisherPID, scopePID *actor.PID) actor.Producer
}

// StrategyFamilyProcessor describes one strategy family's processing pipeline within derive.
// Each strategy family gets one resolver actor per symbol/timeframe combination.
type StrategyFamilyProcessor struct {
	Family      string
	ActorPrefix string
	NewActor    func(source, symbol string, timeframe time.Duration, strategyPublisherPID, scopePID *actor.PID) actor.Producer
}

// RiskFamilyProcessor describes one risk family's processing pipeline within derive.
// Each risk family gets one evaluator actor per symbol/timeframe combination.
type RiskFamilyProcessor struct {
	Family      string
	ActorPrefix string
	NewActor    func(source, symbol string, timeframe time.Duration, riskPublisherPID, scopePID *actor.PID) actor.Producer
}

// ExecutionFamilyProcessor describes one execution family's processing pipeline within derive.
// Each execution family gets one evaluator actor per symbol/timeframe combination.
type ExecutionFamilyProcessor struct {
	Family      string
	ActorPrefix string
	NewActor    func(source, symbol string, timeframe time.Duration, executionPublisherPID *actor.PID) actor.Producer
}

// filterEnabled filters a processor slice by config enablement, logging skipped families.
// This is the canonical filter used during supervisor startup for all processor scopes.
func filterEnabled[T any](all []T, getFamily func(T) string, isEnabled func(string) bool, logger *slog.Logger, scopeLabel string) []T {
	var enabled []T
	for _, p := range all {
		name := getFamily(p)
		if isEnabled(name) {
			enabled = append(enabled, p)
		} else {
			logger.Info(scopeLabel+" family processor skipped", "family", name)
		}
	}
	return enabled
}

// familyNames extracts family names from a processor slice for logging.
func familyNames[T any](processors []T, getFamily func(T) string) []string {
	names := make([]string, len(processors))
	for i, p := range processors {
		names[i] = getFamily(p)
	}
	return names
}

// SourceScopeConfig holds the configuration for a source scope actor.
type SourceScopeConfig struct {
	Source              string
	NATSURL             string
	Registry            natsevidence.Registry
	SignalRegistry      natssignal.Registry
	DecisionRegistry    natsdecision.Registry
	StrategyRegistry    natsstrategy.Registry
	RiskRegistry        natsrisk.Registry
	ExecutionRegistry   natsexecution.Registry
	Timeframes          []time.Duration
	Processors          []FamilyProcessor
	SignalProcessors    []SignalFamilyProcessor
	DecisionProcessors  []DecisionFamilyProcessor
	StrategyProcessors  []StrategyFamilyProcessor
	RiskProcessors      []RiskFamilyProcessor
	ExecutionProcessors []ExecutionFamilyProcessor
	PublisherTracker    *healthz.Tracker
}

// SourceScopeActor supervises all actors for a single source/exchange in derive.
// It owns the evidence publisher, signal publisher, decision publisher, and all sampler actors for that source.
// Each symbol gets one sampler per configured timeframe.
// Trade routing within this source fans out to all samplers for the symbol.
// Candle finalization events fan out to signal samplers.
// Signal generation events fan out to decision evaluators.
type SourceScopeActor struct {
	cfg                   SourceScopeConfig
	logger                *slog.Logger
	publisherPID          *actor.PID
	signalPublisherPID    *actor.PID
	decisionPublisherPID  *actor.PID
	strategyPublisherPID  *actor.PID
	riskPublisherPID      *actor.PID
	executionPublisherPID *actor.PID
	samplers              map[string][]*actor.PID // key: symbol → evidence sampler PIDs
	signalSamplers        map[string][]*actor.PID // key: symbol → signal sampler PIDs
	decisionEvaluators    map[string][]*actor.PID // key: symbol → decision evaluator PIDs
	strategyResolvers     map[string][]*actor.PID // key: symbol → strategy resolver PIDs
	riskEvaluators        map[string][]*actor.PID // key: symbol → risk evaluator PIDs
	executionEvaluators   map[string][]*actor.PID // key: symbol → execution evaluator PIDs
}

func NewSourceScopeActor(cfg SourceScopeConfig) actor.Producer {
	return func() actor.Receiver {
		return &SourceScopeActor{
			cfg:                 cfg,
			logger:              slog.Default().With("actor", "source-scope", "source", cfg.Source),
			samplers:            make(map[string][]*actor.PID),
			signalSamplers:      make(map[string][]*actor.PID),
			decisionEvaluators:  make(map[string][]*actor.PID),
			strategyResolvers:   make(map[string][]*actor.PID),
			riskEvaluators:      make(map[string][]*actor.PID),
			executionEvaluators: make(map[string][]*actor.PID),
		}
	}
}

func (a *SourceScopeActor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		total := 0
		for _, pids := range a.samplers {
			total += len(pids)
		}
		for _, pids := range a.signalSamplers {
			total += len(pids)
		}
		for _, pids := range a.decisionEvaluators {
			total += len(pids)
		}
		for _, pids := range a.strategyResolvers {
			total += len(pids)
		}
		for _, pids := range a.riskEvaluators {
			total += len(pids)
		}
		for _, pids := range a.executionEvaluators {
			total += len(pids)
		}
		a.logger.Info("source scope stopped",
			"symbols", len(a.samplers),
			"total_samplers", total,
		)

	case activateSamplerMessage:
		a.onActivateSampler(c, msg)

	case tradeReceivedMessage:
		a.routeTrade(c, msg)

	case candleFinalizedMessage:
		a.routeCandleToSignal(c, msg)

	case signalGeneratedMessage:
		a.routeSignalToDecision(c, msg)

	case decisionEvaluatedMessage:
		a.routeDecisionToStrategy(c, msg)

	case strategyResolvedMessage:
		a.routeStrategyToRisk(c, msg)

	case riskAssessedMessage:
		a.routeRiskToExecution(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *SourceScopeActor) start(c *actor.Context) {
	a.publisherPID = c.SpawnChild(NewEvidencePublisherActor(EvidencePublisherConfig{
		URL:      a.cfg.NATSURL,
		Source:   "derive.evidence-publisher." + a.cfg.Source,
		Registry: a.cfg.Registry,
		Tracker:  a.cfg.PublisherTracker,
	}), "publisher")

	// Spawn signal publisher only if signal processors are configured.
	if len(a.cfg.SignalProcessors) > 0 {
		a.signalPublisherPID = c.SpawnChild(NewSignalPublisherActor(SignalPublisherConfig{
			URL:      a.cfg.NATSURL,
			Source:   "derive.signal-publisher." + a.cfg.Source,
			Registry: a.cfg.SignalRegistry,
			Tracker:  a.cfg.PublisherTracker,
		}), "signal-publisher")
	}

	// Spawn decision publisher only if decision processors are configured.
	if len(a.cfg.DecisionProcessors) > 0 {
		a.decisionPublisherPID = c.SpawnChild(NewDecisionPublisherActor(DecisionPublisherConfig{
			URL:      a.cfg.NATSURL,
			Source:   "derive.decision-publisher." + a.cfg.Source,
			Registry: a.cfg.DecisionRegistry,
			Tracker:  a.cfg.PublisherTracker,
		}), "decision-publisher")
	}

	// Spawn strategy publisher only if strategy processors are configured.
	if len(a.cfg.StrategyProcessors) > 0 {
		a.strategyPublisherPID = c.SpawnChild(NewStrategyPublisherActor(StrategyPublisherConfig{
			URL:      a.cfg.NATSURL,
			Source:   "derive.strategy-publisher." + a.cfg.Source,
			Registry: a.cfg.StrategyRegistry,
			Tracker:  a.cfg.PublisherTracker,
		}), "strategy-publisher")
	}

	// Spawn risk publisher only if risk processors are configured.
	if len(a.cfg.RiskProcessors) > 0 {
		a.riskPublisherPID = c.SpawnChild(NewRiskPublisherActor(RiskPublisherConfig{
			URL:      a.cfg.NATSURL,
			Source:   "derive.risk-publisher." + a.cfg.Source,
			Registry: a.cfg.RiskRegistry,
			Tracker:  a.cfg.PublisherTracker,
		}), "risk-publisher")
	}

	// Spawn execution publisher only if execution processors are configured.
	if len(a.cfg.ExecutionProcessors) > 0 {
		a.executionPublisherPID = c.SpawnChild(NewExecutionPublisherActor(ExecutionPublisherConfig{
			URL:      a.cfg.NATSURL,
			Source:   "derive.execution-publisher." + a.cfg.Source,
			Registry: a.cfg.ExecutionRegistry,
			Tracker:  a.cfg.PublisherTracker,
		}), "execution-publisher")
	}

	tfSeconds := make([]int, len(a.cfg.Timeframes))
	for i, tf := range a.cfg.Timeframes {
		tfSeconds[i] = int(tf.Seconds())
	}
	a.logger.Info("source scope started",
		"families", familyNames(a.cfg.Processors, func(p FamilyProcessor) string { return p.Family }),
		"signal_families", familyNames(a.cfg.SignalProcessors, func(p SignalFamilyProcessor) string { return p.Family }),
		"decision_families", familyNames(a.cfg.DecisionProcessors, func(p DecisionFamilyProcessor) string { return p.Family }),
		"strategy_families", familyNames(a.cfg.StrategyProcessors, func(p StrategyFamilyProcessor) string { return p.Family }),
		"risk_families", familyNames(a.cfg.RiskProcessors, func(p RiskFamilyProcessor) string { return p.Family }),
		"execution_families", familyNames(a.cfg.ExecutionProcessors, func(p ExecutionFamilyProcessor) string { return p.Family }),
		"timeframes_s", tfSeconds,
	)
}

func (a *SourceScopeActor) onActivateSampler(c *actor.Context, msg activateSamplerMessage) {
	symbol := msg.Target.Symbol
	if _, exists := a.samplers[symbol]; exists {
		return
	}

	scopePID := c.PID()

	// Spawn evidence samplers.
	var pids []*actor.PID
	for _, proc := range a.cfg.Processors {
		for _, tf := range a.cfg.Timeframes {
			name := fmt.Sprintf("%s-%s-%ds", proc.ActorPrefix, symbol, int(tf.Seconds()))
			pid := c.SpawnChild(proc.NewActor(a.cfg.Source, symbol, tf, a.publisherPID, scopePID), name)
			pids = append(pids, pid)
		}
	}
	a.samplers[symbol] = pids

	// Spawn signal samplers.
	var signalPids []*actor.PID
	for _, proc := range a.cfg.SignalProcessors {
		for _, tf := range a.cfg.Timeframes {
			name := fmt.Sprintf("%s-%s-%ds", proc.ActorPrefix, symbol, int(tf.Seconds()))
			pid := c.SpawnChild(proc.NewActor(a.cfg.Source, symbol, tf, a.signalPublisherPID, scopePID), name)
			signalPids = append(signalPids, pid)
		}
	}
	if len(signalPids) > 0 {
		a.signalSamplers[symbol] = signalPids
	}

	// Spawn decision evaluators.
	var decisionPids []*actor.PID
	for _, proc := range a.cfg.DecisionProcessors {
		for _, tf := range a.cfg.Timeframes {
			name := fmt.Sprintf("%s-%s-%ds", proc.ActorPrefix, symbol, int(tf.Seconds()))
			pid := c.SpawnChild(proc.NewActor(a.cfg.Source, symbol, tf, a.decisionPublisherPID, scopePID), name)
			decisionPids = append(decisionPids, pid)
		}
	}
	if len(decisionPids) > 0 {
		a.decisionEvaluators[symbol] = decisionPids
	}

	// Spawn strategy resolvers.
	var strategyPids []*actor.PID
	for _, proc := range a.cfg.StrategyProcessors {
		for _, tf := range a.cfg.Timeframes {
			name := fmt.Sprintf("%s-%s-%ds", proc.ActorPrefix, symbol, int(tf.Seconds()))
			pid := c.SpawnChild(proc.NewActor(a.cfg.Source, symbol, tf, a.strategyPublisherPID, scopePID), name)
			strategyPids = append(strategyPids, pid)
		}
	}
	if len(strategyPids) > 0 {
		a.strategyResolvers[symbol] = strategyPids
	}

	// Spawn risk evaluators.
	var riskPids []*actor.PID
	for _, proc := range a.cfg.RiskProcessors {
		for _, tf := range a.cfg.Timeframes {
			name := fmt.Sprintf("%s-%s-%ds", proc.ActorPrefix, symbol, int(tf.Seconds()))
			pid := c.SpawnChild(proc.NewActor(a.cfg.Source, symbol, tf, a.riskPublisherPID, scopePID), name)
			riskPids = append(riskPids, pid)
		}
	}
	if len(riskPids) > 0 {
		a.riskEvaluators[symbol] = riskPids
	}

	// Spawn execution evaluators.
	var executionPids []*actor.PID
	for _, proc := range a.cfg.ExecutionProcessors {
		for _, tf := range a.cfg.Timeframes {
			name := fmt.Sprintf("%s-%s-%ds", proc.ActorPrefix, symbol, int(tf.Seconds()))
			pid := c.SpawnChild(proc.NewActor(a.cfg.Source, symbol, tf, a.executionPublisherPID), name)
			executionPids = append(executionPids, pid)
		}
	}
	if len(executionPids) > 0 {
		a.executionEvaluators[symbol] = executionPids
	}

	total := 0
	for _, p := range a.samplers {
		total += len(p)
	}
	for _, p := range a.signalSamplers {
		total += len(p)
	}
	for _, p := range a.decisionEvaluators {
		total += len(p)
	}
	for _, p := range a.strategyResolvers {
		total += len(p)
	}
	for _, p := range a.riskEvaluators {
		total += len(p)
	}
	for _, p := range a.executionEvaluators {
		total += len(p)
	}
	a.logger.Info("samplers activated",
		"symbol", symbol,
		"evidence_families", len(a.cfg.Processors),
		"signal_families", len(a.cfg.SignalProcessors),
		"decision_families", len(a.cfg.DecisionProcessors),
		"strategy_families", len(a.cfg.StrategyProcessors),
		"risk_families", len(a.cfg.RiskProcessors),
		"execution_families", len(a.cfg.ExecutionProcessors),
		"timeframe_count", len(a.cfg.Timeframes),
		"total_samplers", total,
	)
}

// routeTrade fans out each trade to all evidence samplers for the symbol.
func (a *SourceScopeActor) routeTrade(c *actor.Context, msg tradeReceivedMessage) {
	symbol := msg.Event.Trade.VenueSymbol()
	pids, exists := a.samplers[symbol]
	if !exists {
		return
	}
	for _, pid := range pids {
		c.Send(pid, msg)
	}
}

// routeCandleToSignal fans out candle finalization events to signal samplers for the same symbol.
func (a *SourceScopeActor) routeCandleToSignal(c *actor.Context, msg candleFinalizedMessage) {
	pids, exists := a.signalSamplers[msg.Symbol]
	if !exists {
		return
	}
	for _, pid := range pids {
		c.Send(pid, msg)
	}
}

// routeSignalToDecision fans out signal generation notifications to decision evaluators.
// The signalGeneratedMessage is sent by signal sampler actors via the scope PID.
func (a *SourceScopeActor) routeSignalToDecision(c *actor.Context, msg signalGeneratedMessage) {
	pids, exists := a.decisionEvaluators[msg.Symbol]
	if !exists {
		return
	}
	for _, pid := range pids {
		c.Send(pid, msg)
	}
}

// routeDecisionToStrategy fans out decision evaluation notifications to strategy resolvers.
// The decisionEvaluatedMessage is sent by decision evaluator actors via the scope PID.
func (a *SourceScopeActor) routeDecisionToStrategy(c *actor.Context, msg decisionEvaluatedMessage) {
	pids, exists := a.strategyResolvers[msg.Symbol]
	if !exists {
		return
	}
	for _, pid := range pids {
		c.Send(pid, msg)
	}
}

// routeStrategyToRisk fans out strategy resolution notifications to risk evaluators.
// The strategyResolvedMessage is sent by strategy resolver actors via the scope PID.
func (a *SourceScopeActor) routeStrategyToRisk(c *actor.Context, msg strategyResolvedMessage) {
	pids, exists := a.riskEvaluators[msg.Symbol]
	if !exists {
		return
	}
	for _, pid := range pids {
		c.Send(pid, msg)
	}
}

// routeRiskToExecution fans out risk assessment notifications to execution evaluators.
// The riskAssessedMessage is sent by risk evaluator actors via the scope PID.
func (a *SourceScopeActor) routeRiskToExecution(c *actor.Context, msg riskAssessedMessage) {
	pids, exists := a.executionEvaluators[msg.Symbol]
	if !exists {
		return
	}
	for _, pid := range pids {
		c.Send(pid, msg)
	}
}
