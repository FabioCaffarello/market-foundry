package analyticalclient

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"internal/domain/consistency"
	"internal/domain/effectiveness"
	"internal/shared/problem"
)

const (
	decisionReviewDefaultLimit = 20
	decisionReviewMaxLimit     = 100
)

// GetDecisionReviewUseCase queries the analytical store for decision-centric review bundles.
// It reuses the CompositeReader to fetch full causal chains, then projects each chain
// into a decision-anchored review bundle with structured inputs/transform/constraints/output.
//
// S471: This is a read-side projection — no new write-side changes required.
type GetDecisionReviewUseCase struct {
	reader CompositeReader
	logger *slog.Logger
}

func NewGetDecisionReviewUseCase(reader CompositeReader, logger *slog.Logger) *GetDecisionReviewUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetDecisionReviewUseCase{reader: reader, logger: logger.With("component", "decision_review_usecase")}
}

func (uc *GetDecisionReviewUseCase) Execute(ctx context.Context, query DecisionReviewQuery) (DecisionReviewReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return DecisionReviewReply{}, problem.New(problem.Unavailable, "decision review reader is unavailable")
	}

	start := time.Now()

	if query.CorrelationID != "" {
		return uc.executeSingle(ctx, query, start)
	}
	return uc.executeBatch(ctx, query, start)
}

func (uc *GetDecisionReviewUseCase) executeSingle(ctx context.Context, query DecisionReviewQuery, start time.Time) (DecisionReviewReply, *problem.Problem) {
	if query.Symbol == "" {
		return DecisionReviewReply{}, problem.New(problem.InvalidArgument, "symbol is required for single-decision review (S301 isolation)")
	}

	chain, err := uc.reader.QueryChainByCorrelationID(ctx, query.CorrelationID, query.Symbol)
	elapsed := time.Since(start)
	if err != nil {
		uc.logger.Warn("decision review query failed",
			"correlation_id", query.CorrelationID, "symbol", query.Symbol, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return DecisionReviewReply{}, problem.Wrap(err, problem.Unavailable, "decision review query failed")
	}

	var reviews []DecisionReviewBundle
	if chain != nil && chain.StageCount > 0 {
		computeAttribution(chain)
		if bundle := projectChainToReview(chain); bundle != nil {
			if query.Outcome == "" || bundle.Transform.Outcome == query.Outcome {
				reviews = append(reviews, *bundle)
			}
		}
	}

	return DecisionReviewReply{
		Reviews: reviews,
		Source:  "clickhouse",
		Meta: CompositeQueryMeta{
			TotalMs:    elapsed.Milliseconds(),
			ChainCount: len(reviews),
		},
	}, nil
}

func (uc *GetDecisionReviewUseCase) executeBatch(ctx context.Context, query DecisionReviewQuery, start time.Time) (DecisionReviewReply, *problem.Problem) {
	if query.Source == "" {
		return DecisionReviewReply{}, problem.New(problem.InvalidArgument, "source is required for batch decision review")
	}
	if query.Symbol == "" {
		return DecisionReviewReply{}, problem.New(problem.InvalidArgument, "symbol is required for batch decision review")
	}
	if query.Timeframe <= 0 {
		return DecisionReviewReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive for batch decision review")
	}

	if query.Limit <= 0 {
		query.Limit = decisionReviewDefaultLimit
	}
	if query.Limit > decisionReviewMaxLimit {
		query.Limit = decisionReviewMaxLimit
	}

	// Fetch more chains than requested to account for post-filter (outcome filter).
	// The composite reader returns execution-rooted chains; we project decision-anchored views.
	fetchLimit := query.Limit
	if query.Outcome != "" {
		fetchLimit = query.Limit * 3
		if fetchLimit > compositeMaxLimit {
			fetchLimit = compositeMaxLimit
		}
	}

	chains, err := uc.reader.QueryChainsBatch(ctx, query.Source, query.Symbol, query.Timeframe, query.Since, query.Until, fetchLimit)
	elapsed := time.Since(start)
	if err != nil {
		uc.logger.Warn("decision review batch query failed",
			"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return DecisionReviewReply{}, problem.Wrap(err, problem.Unavailable, "decision review batch query failed")
	}

	var reviews []DecisionReviewBundle
	for i := range chains {
		computeAttribution(&chains[i])
		bundle := projectChainToReview(&chains[i])
		if bundle == nil {
			continue
		}
		if query.Outcome != "" && bundle.Transform.Outcome != query.Outcome {
			continue
		}
		reviews = append(reviews, *bundle)
		if len(reviews) >= query.Limit {
			break
		}
	}

	uc.logger.Info("decision review batch completed",
		"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"chains_fetched", len(chains), "reviews_returned", len(reviews), "total_ms", elapsed.Milliseconds(),
	)

	return DecisionReviewReply{
		Reviews: reviews,
		Source:  "clickhouse",
		Meta: CompositeQueryMeta{
			TotalMs:    elapsed.Milliseconds(),
			ChainCount: len(reviews),
		},
	}, nil
}

// projectChainToReview converts a CompositeExecutionChain into a DecisionReviewBundle.
// If the chain has no decision stage, it returns nil — the decision is the anchor.
func projectChainToReview(chain *CompositeExecutionChain) *DecisionReviewBundle {
	if chain.Decision == nil {
		return nil
	}

	d := chain.Decision
	bundle := &DecisionReviewBundle{
		CorrelationID: chain.CorrelationID,
		StageCount:    chain.StageCount,
		ChainComplete: chain.ChainComplete,
		MissingStages: chain.MissingStages,
	}

	// Inputs: signal evidence as recorded by the decision (decision.Signals carries the
	// decision-owned record of which signals contributed). The signal event from the chain
	// provides the event_id and timestamp of the originating signal.
	if len(d.Signals) > 0 || chain.Signal != nil {
		inputs := &ReviewInputs{
			Signals: d.Signals,
		}
		if chain.Signal != nil {
			inputs.EventID = chain.Signal.EventID
			inputs.At = chain.Signal.OccurredAt
		} else {
			inputs.At = d.OccurredAt
		}
		bundle.Inputs = inputs
	}

	// Transform: the decision itself.
	bundle.Transform = &ReviewTransform{
		Type:       d.Type,
		Source:     d.Source,
		Symbol:     d.VenueSymbol(),
		Timeframe:  d.Timeframe,
		Outcome:    string(d.Outcome),
		Severity:   string(d.Severity),
		Confidence: d.Confidence,
		Rationale:  d.Rationale,
		Final:      d.Final,
		EventID:    d.EventID,
		At:         d.OccurredAt,
		Metadata:   d.Metadata,
	}

	// Resolution: strategy derived.
	if chain.Strategy != nil {
		st := chain.Strategy
		var refs []ReviewDecisionRef
		for _, di := range st.Decisions {
			refs = append(refs, ReviewDecisionRef{
				Type:       di.Type,
				Outcome:    di.Outcome,
				Confidence: di.Confidence,
				Severity:   di.Severity,
				Rationale:  di.Rationale,
			})
		}
		bundle.Resolution = &ReviewResolution{
			Type:           st.Type,
			Direction:      string(st.Direction),
			Confidence:     st.Confidence,
			DecisionInputs: refs,
			Parameters:     st.Parameters,
			EventID:        st.EventID,
			At:             st.OccurredAt,
		}
	}

	// Constraints: risk assessment.
	if chain.Risk != nil {
		r := chain.Risk
		var stratCtx []AttributionStrategyContext
		for _, si := range r.Strategies {
			stratCtx = append(stratCtx, AttributionStrategyContext{
				Type:              si.Type,
				Direction:         si.Direction,
				Confidence:        si.Confidence,
				DecisionSeverity:  si.DecisionSeverity,
				DecisionRationale: si.DecisionRationale,
			})
		}
		bundle.Constraints = &ReviewConstraints{
			Type:            r.Type,
			Disposition:     string(r.Disposition),
			Confidence:      r.Confidence,
			Rationale:       r.Rationale,
			Limits:          r.Constraints,
			StrategyContext: stratCtx,
			EventID:         r.EventID,
			At:              r.OccurredAt,
		}
	}

	// Output: execution intent.
	if chain.Execution != nil {
		e := chain.Execution
		bundle.Output = &ReviewOutput{
			Type:           e.Type,
			Side:           string(e.Side),
			Quantity:       e.Quantity,
			FilledQuantity: e.FilledQuantity,
			Status:         string(e.Status),
			Final:          e.Final,
			CorrelationID:  e.CorrelationID,
			CausationID:    e.CausationID,
			EventID:        e.EventID,
			At:             e.OccurredAt,
		}
	}

	// S476: Compute effectiveness attribution when execution reached terminal state.
	if chain.Execution != nil {
		attr := effectiveness.Classify(chain.Execution.ExecutionIntent)
		if attr != nil {
			enrichFromChain(attr, chain)
			bundle.Effectiveness = &ReviewEffectiveness{
				Outcome:        string(attr.Outcome),
				RealizedPnL:    attr.RealizedPnL,
				GrossPnL:       attr.GrossPnL,
				NetPnL:         attr.NetPnL,
				TotalFees:      attr.TotalFees,
				EntryCostBasis: attr.EntryCostBasis,
				FillCount:      attr.FillCount,
				Simulated:      attr.Simulated,
				Explanation:    attr.Explain(),
			}
		}
	}

	// S472: Run cross-domain consistency checks on the chain.
	report := consistency.Check(buildChainSnapshot(chain))
	bundle.Consistency = &report

	bundle.Explanation = buildReviewExplanation(bundle)

	return bundle
}

// buildChainSnapshot converts a CompositeExecutionChain into a consistency.ChainSnapshot.
func buildChainSnapshot(chain *CompositeExecutionChain) consistency.ChainSnapshot {
	snap := consistency.ChainSnapshot{
		CorrelationID: chain.CorrelationID,
	}

	if chain.Decision != nil {
		d := chain.Decision
		snap.HasDecision = true
		snap.DecisionOutcome = string(d.Outcome)
		snap.DecisionSeverity = string(d.Severity)
		snap.DecisionConfidence = d.Confidence
		snap.DecisionSymbol = d.VenueSymbol()
		snap.DecisionSource = d.Source
		snap.DecisionTimeframe = d.Timeframe
	}

	if chain.Strategy != nil {
		st := chain.Strategy
		snap.HasStrategy = true
		snap.StrategyDirection = string(st.Direction)
		snap.StrategyConfidence = st.Confidence
		snap.StrategySymbol = st.VenueSymbol()
		snap.StrategySource = st.Source
		snap.StrategyTimeframe = st.Timeframe
	}

	if chain.Risk != nil {
		r := chain.Risk
		snap.HasRisk = true
		snap.RiskDisposition = string(r.Disposition)
		snap.RiskConfidence = r.Confidence
		snap.RiskSymbol = r.VenueSymbol()
		snap.RiskSource = r.Source
		snap.RiskTimeframe = r.Timeframe
		if len(r.Strategies) > 0 {
			snap.RiskStrategyRef = r.Strategies[0].Type
			snap.RiskStrategyDir = r.Strategies[0].Direction
		}
	}

	if chain.Execution != nil {
		e := chain.Execution
		snap.HasExecution = true
		snap.ExecutionSide = string(e.Side)
		snap.ExecutionQuantity = e.Quantity
		snap.ExecutionSymbol = e.Symbol
		snap.ExecutionSource = e.Source
		snap.ExecutionTimeframe = e.Timeframe
		snap.ExecutionRiskDisp = e.Risk.Disposition
	}

	return snap
}

// buildReviewExplanation generates a human-readable explanation of the decision review.
func buildReviewExplanation(b *DecisionReviewBundle) string {
	if b.Transform == nil {
		return "No decision data available."
	}

	var parts []string

	// Decision summary.
	parts = append(parts, fmt.Sprintf("Decision %q evaluated %s/%s/%d: outcome=%s, severity=%s, confidence=%s.",
		b.Transform.Type, b.Transform.Source, b.Transform.Symbol, b.Transform.Timeframe,
		b.Transform.Outcome, b.Transform.Severity, b.Transform.Confidence))

	if b.Transform.Rationale != "" {
		parts = append(parts, fmt.Sprintf("Rationale: %s", b.Transform.Rationale))
	}

	// Input context.
	if b.Inputs != nil && len(b.Inputs.Signals) > 0 {
		sigTypes := make([]string, len(b.Inputs.Signals))
		for i, s := range b.Inputs.Signals {
			sigTypes[i] = s.Type
		}
		parts = append(parts, fmt.Sprintf("Signal inputs: %s.", strings.Join(sigTypes, ", ")))
	} else {
		parts = append(parts, "Signal inputs: not available in chain.")
	}

	// Strategy resolution.
	if b.Resolution != nil {
		parts = append(parts, fmt.Sprintf("Strategy resolved: %s direction=%s confidence=%s.",
			b.Resolution.Type, b.Resolution.Direction, b.Resolution.Confidence))
	} else if b.Transform.Outcome == "triggered" {
		parts = append(parts, "Strategy: not yet resolved or missing from chain.")
	}

	// Risk constraints.
	if b.Constraints != nil {
		parts = append(parts, fmt.Sprintf("Risk gate: disposition=%s confidence=%s.",
			b.Constraints.Disposition, b.Constraints.Confidence))
		if b.Constraints.Rationale != "" {
			parts = append(parts, fmt.Sprintf("Risk rationale: %s", b.Constraints.Rationale))
		}
	} else if b.Resolution != nil {
		parts = append(parts, "Risk assessment: not yet evaluated or missing from chain.")
	}

	// Execution output.
	if b.Output != nil {
		parts = append(parts, fmt.Sprintf("Execution: %s side=%s status=%s quantity=%s filled=%s.",
			b.Output.Type, b.Output.Side, b.Output.Status, b.Output.Quantity, b.Output.FilledQuantity))
	} else if b.Constraints != nil && b.Constraints.Disposition == "approved" {
		parts = append(parts, "Execution: not yet submitted or missing from chain.")
	}

	// Completeness.
	if !b.ChainComplete && len(b.MissingStages) > 0 {
		parts = append(parts, fmt.Sprintf("Incomplete chain — missing stages: %s.", strings.Join(b.MissingStages, ", ")))
	}

	// S476: Effectiveness summary.
	if b.Effectiveness != nil {
		parts = append(parts, fmt.Sprintf("Effectiveness: outcome=%s, net_pnl=%.6f, fees=%.6f, fills=%d.",
			b.Effectiveness.Outcome, b.Effectiveness.NetPnL, b.Effectiveness.TotalFees, b.Effectiveness.FillCount))
	}

	// S472: Consistency check summary.
	if b.Consistency != nil && !b.Consistency.Clean {
		parts = append(parts, fmt.Sprintf("Consistency: %d violation(s), %d warning(s).", b.Consistency.Violations, b.Consistency.Warnings))
	}

	return strings.Join(parts, " ")
}
