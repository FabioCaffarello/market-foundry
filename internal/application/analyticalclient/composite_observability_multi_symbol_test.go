package analyticalclient_test

// S303: Composite Observability Under Multi-Symbol Load
//
// This file validates that the explainability surfaces (chain, funnel, dispositions)
// remain correct, readable, and operationally useful when multiple symbols coexist.
//
// Focus areas:
//   OBS-1 — Cross-surface consistency: funnel counts align with chain presence.
//   OBS-2 — Disposition/attribution coherence across symbols.
//   OBS-3 — Causal metadata integrity under multi-symbol interleaving.
//   OBS-4 — Filter specificity: same type/source but different symbols yield isolated results.
//   OBS-5 — Explainability readability: attribution fields populated per symbol.

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/decision"
	"internal/domain/execution"
	"internal/domain/instrument"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
)

// instrumentFromVenue parses a venue symbol like "btcusdt" into a
// CanonicalInstrument (ContractPerpetual). Test-only helper used by S303
// fixture builders during the H-6.b transitory window. Panics on unknown
// venue symbols rather than threading t through every call site, since the
// inputs are hard-coded literals in this test file.
func instrumentFromVenue(venueSym string) instrument.CanonicalInstrument {
	upper := strings.ToUpper(venueSym)
	for _, quote := range []string{"USDT", "USD", "BUSD"} {
		if strings.HasSuffix(upper, quote) {
			base := strings.TrimSuffix(upper, quote)
			inst, prob := instrument.New(base, quote, instrument.ContractPerpetual)
			if prob != nil {
				panic("test setup: instrument.New(" + base + "/" + quote + "): " + prob.Message)
			}
			return inst
		}
	}
	panic("test setup: unrecognized venue symbol " + venueSym)
}

// ---------------------------------------------------------------------------
// OBS-1: Cross-surface consistency — funnel stage counts match chain counts
// ---------------------------------------------------------------------------

func TestS303_OBS1_FunnelChainConsistency(t *testing.T) {
	// Scenario: 3 symbols, each with a known number of complete/partial chains.
	// Funnel must reflect the exact per-symbol counts.
	type surfaceCheck struct {
		symbol         string
		totalChains    int
		approvedChains int // chains with execution
		rejectedChains int // chains without execution
	}

	checks := []surfaceCheck{
		{symbol: "btcusdt", totalChains: 3, approvedChains: 2, rejectedChains: 1},
		{symbol: "ethusdt", totalChains: 2, approvedChains: 1, rejectedChains: 1},
		{symbol: "solusdt", totalChains: 1, approvedChains: 1, rejectedChains: 0},
	}

	for _, sc := range checks {
		t.Run("cross_surface_"+sc.symbol, func(t *testing.T) {
			// Build funnel counts consistent with chain counts.
			funnelReader := &s303FunnelStubReader{
				funnelPerSymbol: map[string][]analyticalclient.StageFunnelCount{
					sc.symbol: {
						{Stage: "signal", Count: int64(sc.totalChains)},
						{Stage: "decision", Count: int64(sc.totalChains)},
						{Stage: "strategy", Count: int64(sc.totalChains)},
						{Stage: "risk", Count: int64(sc.totalChains)},
						{Stage: "execution", Count: int64(sc.approvedChains)},
					},
				},
				dispPerSymbol: map[string][]analyticalclient.DispositionCount{
					sc.symbol: buildDispositions(sc.approvedChains, sc.rejectedChains, 0),
				},
			}

			funnelUC := analyticalclient.NewGetPipelineFunnelUseCase(funnelReader, slog.Default())
			funnelReply, prob := funnelUC.Execute(context.Background(), analyticalclient.PipelineFunnelQuery{
				Type: "rsi", Source: "binancef", Symbol: sc.symbol, Timeframe: 60,
			})
			if prob != nil {
				t.Fatalf("funnel query failed: %v", prob)
			}

			// Validate monotonic decrease: signal >= decision >= ... >= execution.
			for i := 1; i < len(funnelReply.Stages); i++ {
				if funnelReply.Stages[i].Count > funnelReply.Stages[i-1].Count {
					t.Errorf("[%s] non-monotonic funnel: %s(%d) > %s(%d)",
						sc.symbol,
						funnelReply.Stages[i].Stage, funnelReply.Stages[i].Count,
						funnelReply.Stages[i-1].Stage, funnelReply.Stages[i-1].Count,
					)
				}
			}

			// Execution count must equal approved chains.
			var execCount int64
			for _, s := range funnelReply.Stages {
				if s.Stage == "execution" {
					execCount = s.Count
				}
			}
			if execCount != int64(sc.approvedChains) {
				t.Errorf("[%s] funnel execution count=%d, want %d (approved chains)", sc.symbol, execCount, sc.approvedChains)
			}

			// Disposition total must equal risk stage count.
			dispUC := analyticalclient.NewGetDispositionBreakdownUseCase(funnelReader, slog.Default())
			dispReply, prob := dispUC.Execute(context.Background(), analyticalclient.DispositionBreakdownQuery{
				Type: "rsi", Source: "binancef", Symbol: sc.symbol, Timeframe: 60,
			})
			if prob != nil {
				t.Fatalf("disposition query failed: %v", prob)
			}
			if dispReply.Total != int64(sc.totalChains) {
				t.Errorf("[%s] disposition total=%d, want %d (risk count)", sc.symbol, dispReply.Total, sc.totalChains)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// OBS-2: Disposition/attribution coherence across symbols
// ---------------------------------------------------------------------------

func TestS303_OBS2_DispositionAttributionCoherence(t *testing.T) {
	// For each symbol, the chain's attribution.disposition must match the
	// corresponding disposition count surface. This validates that a human
	// reading both surfaces sees consistent information.
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	wantDispositions := map[string]string{
		"btcusdt": "approved",
		"ethusdt": "rejected",
		"solusdt": "modified",
	}

	chainsPerSymbol := make(map[string]*analyticalclient.CompositeExecutionChain, 3)
	for _, sym := range symbols {
		corrID := "s303-obs2-" + sym
		disp := wantDispositions[sym]
		hasExec := disp != "rejected"
		chainsPerSymbol[sym] = buildS303Chain(corrID, sym, disp, hasExec)
	}

	reader := &multiSymbolStubReader{chainsPerSymbol: chainsPerSymbol}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	for _, sym := range symbols {
		t.Run("coherence_"+sym, func(t *testing.T) {
			reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
				CorrelationID: "s303-obs2-" + sym, Symbol: sym,
			})
			if prob != nil {
				t.Fatalf("unexpected problem: %v", prob)
			}
			if len(reply.Chains) != 1 {
				t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
			}

			chain := reply.Chains[0]
			if chain.Attribution == nil {
				t.Fatal("expected attribution")
			}
			if chain.Attribution.Disposition != wantDispositions[sym] {
				t.Errorf("attribution.disposition=%q, want %q", chain.Attribution.Disposition, wantDispositions[sym])
			}

			// Verify chain completeness matches disposition semantics.
			if wantDispositions[sym] == "rejected" {
				if chain.ChainComplete {
					t.Error("rejected chain should not be complete")
				}
				if chain.Execution != nil {
					t.Error("rejected chain should not have execution")
				}
			} else {
				if !chain.ChainComplete {
					t.Errorf("%s chain should be complete", wantDispositions[sym])
				}
				if chain.Execution == nil {
					t.Errorf("%s chain should have execution", wantDispositions[sym])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// OBS-3: Causal metadata integrity under multi-symbol interleaving
// ---------------------------------------------------------------------------

func TestS303_OBS3_CausalMetadataIntegrity(t *testing.T) {
	// Verify that when chains from 3 symbols are queried, each chain's
	// causal DAG (causation_id → parent event_id) is internally consistent
	// and belongs exclusively to the declared symbol.
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}

	chainsPerSymbol := make(map[string]*analyticalclient.CompositeExecutionChain, 3)
	for _, sym := range symbols {
		corrID := "s303-obs3-" + sym
		chainsPerSymbol[sym] = buildS303Chain(corrID, sym, "approved", true)
	}

	reader := &multiSymbolStubReader{chainsPerSymbol: chainsPerSymbol}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	for _, sym := range symbols {
		t.Run("causal_"+sym, func(t *testing.T) {
			reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
				CorrelationID: "s303-obs3-" + sym, Symbol: sym,
			})
			if prob != nil {
				t.Fatalf("unexpected problem: %v", prob)
			}
			chain := reply.Chains[0]
			corrID := "s303-obs3-" + sym

			// All stages share the same correlation_id.
			if chain.Signal.CorrelationID != corrID {
				t.Errorf("signal.correlation_id=%q, want %q", chain.Signal.CorrelationID, corrID)
			}
			if chain.Decision.CorrelationID != corrID {
				t.Errorf("decision.correlation_id=%q, want %q", chain.Decision.CorrelationID, corrID)
			}
			if chain.Strategy.CorrelationID != corrID {
				t.Errorf("strategy.correlation_id=%q, want %q", chain.Strategy.CorrelationID, corrID)
			}
			if chain.Risk.CorrelationID != corrID {
				t.Errorf("risk.correlation_id=%q, want %q", chain.Risk.CorrelationID, corrID)
			}
			if chain.Execution.EventCorrelationID != corrID {
				t.Errorf("execution.event_correlation_id=%q, want %q", chain.Execution.EventCorrelationID, corrID)
			}

			// Causation chain: signal(root) → decision → strategy → risk → execution.
			if chain.Signal.CausationID != "" {
				t.Errorf("signal.causation_id should be empty (root), got %q", chain.Signal.CausationID)
			}
			if chain.Decision.CausationID != chain.Signal.EventID {
				t.Errorf("decision.causation_id=%q, want signal.event_id=%q", chain.Decision.CausationID, chain.Signal.EventID)
			}
			if chain.Strategy.CausationID != chain.Decision.EventID {
				t.Errorf("strategy.causation_id=%q, want decision.event_id=%q", chain.Strategy.CausationID, chain.Decision.EventID)
			}
			if chain.Risk.CausationID != chain.Strategy.EventID {
				t.Errorf("risk.causation_id=%q, want strategy.event_id=%q", chain.Risk.CausationID, chain.Strategy.EventID)
			}
			if chain.Execution.EventCausationID != chain.Risk.EventID {
				t.Errorf("execution.event_causation_id=%q, want risk.event_id=%q", chain.Execution.EventCausationID, chain.Risk.EventID)
			}

			// All stage symbols must match.
			if chain.Signal.VenueSymbol() != sym {
				t.Errorf("signal.symbol=%q, want %q", chain.Signal.VenueSymbol(), sym)
			}
			if chain.Decision.VenueSymbol() != sym {
				t.Errorf("decision.symbol=%q, want %q", chain.Decision.VenueSymbol(), sym)
			}
			if chain.Strategy.VenueSymbol() != sym {
				t.Errorf("strategy.symbol=%q, want %q", chain.Strategy.VenueSymbol(), sym)
			}
			if chain.Risk.VenueSymbol() != sym {
				t.Errorf("risk.symbol=%q, want %q", chain.Risk.VenueSymbol(), sym)
			}
			if chain.Execution.Symbol != sym {
				t.Errorf("execution.symbol=%q, want %q", chain.Execution.Symbol, sym)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// OBS-4: Filter specificity — same type/source, different symbol
// ---------------------------------------------------------------------------

func TestS303_OBS4_FilterSpecificity(t *testing.T) {
	// All 3 symbols share the same type ("rsi") and source ("binancef"),
	// but funnel and disposition queries must return isolated results.
	funnelData := map[string][]analyticalclient.StageFunnelCount{
		"btcusdt": {
			{Stage: "signal", Count: 100}, {Stage: "decision", Count: 95},
			{Stage: "strategy", Count: 90}, {Stage: "risk", Count: 85},
			{Stage: "execution", Count: 80},
		},
		"ethusdt": {
			{Stage: "signal", Count: 60}, {Stage: "decision", Count: 55},
			{Stage: "strategy", Count: 50}, {Stage: "risk", Count: 45},
			{Stage: "execution", Count: 30},
		},
		"solusdt": {
			{Stage: "signal", Count: 20}, {Stage: "decision", Count: 18},
			{Stage: "strategy", Count: 16}, {Stage: "risk", Count: 14},
			{Stage: "execution", Count: 12},
		},
	}

	dispData := map[string][]analyticalclient.DispositionCount{
		"btcusdt": {{Disposition: "approved", Count: 80}, {Disposition: "rejected", Count: 5}},
		"ethusdt": {{Disposition: "approved", Count: 30}, {Disposition: "rejected", Count: 10}, {Disposition: "modified", Count: 5}},
		"solusdt": {{Disposition: "approved", Count: 12}, {Disposition: "modified", Count: 2}},
	}

	reader := &s303FunnelStubReader{funnelPerSymbol: funnelData, dispPerSymbol: dispData}
	funnelUC := analyticalclient.NewGetPipelineFunnelUseCase(reader, slog.Default())
	dispUC := analyticalclient.NewGetDispositionBreakdownUseCase(reader, slog.Default())

	for _, sym := range []string{"btcusdt", "ethusdt", "solusdt"} {
		t.Run("filter_"+sym, func(t *testing.T) {
			fReply, fProb := funnelUC.Execute(context.Background(), analyticalclient.PipelineFunnelQuery{
				Type: "rsi", Source: "binancef", Symbol: sym, Timeframe: 60,
			})
			if fProb != nil {
				t.Fatalf("funnel: %v", fProb)
			}

			// Signal count must match symbol-specific data, not any other symbol.
			wantSignal := funnelData[sym][0].Count
			if fReply.Stages[0].Count != wantSignal {
				t.Errorf("[%s] signal count=%d, want %d", sym, fReply.Stages[0].Count, wantSignal)
			}

			dReply, dProb := dispUC.Execute(context.Background(), analyticalclient.DispositionBreakdownQuery{
				Type: "rsi", Source: "binancef", Symbol: sym, Timeframe: 60,
			})
			if dProb != nil {
				t.Fatalf("disposition: %v", dProb)
			}

			// Total dispositions must match this symbol only.
			var wantTotal int64
			for _, d := range dispData[sym] {
				wantTotal += d.Count
			}
			if dReply.Total != wantTotal {
				t.Errorf("[%s] disposition total=%d, want %d", sym, dReply.Total, wantTotal)
			}

			// Verify percentages sum to ~100%.
			var pctSum float64
			for _, d := range dReply.Dispositions {
				pctSum += d.Percentage
			}
			if pctSum < 99.9 || pctSum > 100.1 {
				t.Errorf("[%s] percentages sum=%.2f, want ~100", sym, pctSum)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// OBS-5: Attribution explainability readability under multi-symbol
// ---------------------------------------------------------------------------

func TestS303_OBS5_AttributionReadability(t *testing.T) {
	// Each symbol has a different attribution profile. Verify every
	// human-readable field is populated and symbol-specific.
	type attrCheck struct {
		symbol       string
		disposition  string
		rationale    string
		severity     string
		direction    string
		strategyType string
		maxPos       string
		maxExposure  string
	}

	checks := []attrCheck{
		{
			symbol: "btcusdt", disposition: "approved", rationale: "within all risk limits",
			severity: "high", direction: "long", strategyType: "mean_reversion_entry",
			maxPos: "0.10", maxExposure: "1.0",
		},
		{
			symbol: "ethusdt", disposition: "rejected", rationale: "drawdown limit exceeded for eth exposure",
			severity: "moderate", direction: "short", strategyType: "trend_following_entry",
			maxPos: "0.05", maxExposure: "0.5",
		},
		{
			symbol: "solusdt", disposition: "modified", rationale: "position size capped due to sol volatility",
			severity: "low", direction: "long", strategyType: "squeeze_breakout_entry",
			maxPos: "0.03", maxExposure: "0.3",
		},
	}

	for _, ac := range checks {
		t.Run("readability_"+ac.symbol, func(t *testing.T) {
			corrID := "s303-obs5-" + ac.symbol
			chain := buildS303DetailedChain(corrID, ac.symbol, ac.disposition,
				ac.rationale, ac.severity, ac.direction, ac.strategyType,
				ac.maxPos, ac.maxExposure, ac.disposition != "rejected")

			reader := &multiSymbolStubReader{
				chainsPerSymbol: map[string]*analyticalclient.CompositeExecutionChain{ac.symbol: chain},
			}
			uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

			reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
				CorrelationID: corrID, Symbol: ac.symbol,
			})
			if prob != nil {
				t.Fatalf("unexpected problem: %v", prob)
			}
			ch := reply.Chains[0]

			// Attribution must be present.
			if ch.Attribution == nil {
				t.Fatal("expected attribution")
			}
			a := ch.Attribution

			// Disposition.
			if a.Disposition != ac.disposition {
				t.Errorf("disposition=%q, want %q", a.Disposition, ac.disposition)
			}

			// Rationale is non-empty and symbol-specific.
			if a.Rationale == "" {
				t.Error("rationale must not be empty")
			}
			if a.Rationale != ac.rationale {
				t.Errorf("rationale=%q, want %q", a.Rationale, ac.rationale)
			}

			// Constraints populated.
			if a.ActiveConstraints.MaxPositionSize != ac.maxPos {
				t.Errorf("max_position_size=%q, want %q", a.ActiveConstraints.MaxPositionSize, ac.maxPos)
			}
			if a.ActiveConstraints.MaxExposure != ac.maxExposure {
				t.Errorf("max_exposure=%q, want %q", a.ActiveConstraints.MaxExposure, ac.maxExposure)
			}

			// Strategy context populated with correct values.
			if len(a.StrategyContext) != 1 {
				t.Fatalf("expected 1 strategy context, got %d", len(a.StrategyContext))
			}
			sc := a.StrategyContext[0]
			if sc.Type != ac.strategyType {
				t.Errorf("strategy_context.type=%q, want %q", sc.Type, ac.strategyType)
			}
			if sc.Direction != ac.direction {
				t.Errorf("strategy_context.direction=%q, want %q", sc.Direction, ac.direction)
			}
			if sc.DecisionSeverity != ac.severity {
				t.Errorf("strategy_context.decision_severity=%q, want %q", sc.DecisionSeverity, ac.severity)
			}
			if sc.Confidence == "" {
				t.Error("strategy_context.confidence must not be empty")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// OBS-6: Batch explainability — multi-symbol batch queries remain readable
// ---------------------------------------------------------------------------

func TestS303_OBS6_BatchExplainability(t *testing.T) {
	// Verify batch queries for different symbols return non-overlapping chains
	// with independently readable attribution.
	batchData := map[string][]analyticalclient.CompositeExecutionChain{
		"btcusdt": {
			*buildS303Chain("s303-obs6-btc-1", "btcusdt", "approved", true),
			*buildS303Chain("s303-obs6-btc-2", "btcusdt", "modified", true),
		},
		"ethusdt": {
			*buildS303Chain("s303-obs6-eth-1", "ethusdt", "rejected", false),
		},
	}

	reader := &multiSymbolBatchStubReader{batchPerSymbol: batchData}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	// BTC batch: 2 chains, both with attribution, different dispositions.
	t.Run("batch_btcusdt", func(t *testing.T) {
		reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
		})
		if prob != nil {
			t.Fatalf("unexpected problem: %v", prob)
		}
		if len(reply.Chains) != 2 {
			t.Fatalf("expected 2 chains, got %d", len(reply.Chains))
		}

		// Each chain must have attribution.
		for i, ch := range reply.Chains {
			if ch.Attribution == nil {
				t.Errorf("chain[%d] missing attribution", i)
				continue
			}
			// All stages must belong to btcusdt.
			if ch.Signal != nil && ch.Signal.VenueSymbol() != "btcusdt" {
				t.Errorf("chain[%d] signal.symbol=%q, want btcusdt", i, ch.Signal.VenueSymbol())
			}
		}

		// Verify different dispositions coexist in same batch.
		disps := map[string]bool{}
		for _, ch := range reply.Chains {
			disps[ch.Attribution.Disposition] = true
		}
		if !disps["approved"] || !disps["modified"] {
			t.Errorf("expected approved+modified in btcusdt batch, got %v", disps)
		}
	})

	// ETH batch: 1 rejected chain, no execution.
	t.Run("batch_ethusdt", func(t *testing.T) {
		reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			Source: "binancef", Symbol: "ethusdt", Timeframe: 60,
		})
		if prob != nil {
			t.Fatalf("unexpected problem: %v", prob)
		}
		if len(reply.Chains) != 1 {
			t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
		}
		ch := reply.Chains[0]
		if ch.Attribution == nil {
			t.Fatal("expected attribution on rejected chain")
		}
		if ch.Attribution.Disposition != "rejected" {
			t.Errorf("disposition=%q, want rejected", ch.Attribution.Disposition)
		}
		if ch.Execution != nil {
			t.Error("rejected chain should not have execution")
		}
	})
}

// ---------------------------------------------------------------------------
// S303 fixture builders
// ---------------------------------------------------------------------------

func buildS303Chain(corrID, symbol, disposition string, hasExec bool) *analyticalclient.CompositeExecutionChain {
	return buildS303DetailedChain(corrID, symbol, disposition,
		"test rationale for "+symbol, "moderate", "long", "test_strategy",
		"0.10", "1.0", hasExec)
}

func buildS303DetailedChain(corrID, symbol, disposition, rationale, severity, dir, stratType, maxPos, maxExp string, hasExec bool) *analyticalclient.CompositeExecutionChain {
	now := time.Now()
	chain := &analyticalclient.CompositeExecutionChain{
		CorrelationID: corrID,
		Signal: &analyticalclient.SignalWithTrace{
			Signal: signal.Signal{
				Type: "rsi", Source: "binancef", Instrument: instrumentFromVenue(symbol), Timeframe: 60,
				Value: "42.5", Timestamp: now,
			},
			EventID: "sig-" + corrID, CorrelationID: corrID, OccurredAt: now,
		},
		Decision: &analyticalclient.DecisionWithTrace{
			Decision: decision.Decision{
				Type: "rsi_oversold", Source: "binancef", Instrument: instrumentFromVenue(symbol), Timeframe: 60,
				Outcome: "triggered", Severity: decision.Severity(severity), Confidence: "0.85", Timestamp: now,
			},
			EventID: "dec-" + corrID, CorrelationID: corrID, CausationID: "sig-" + corrID, OccurredAt: now,
		},
		Strategy: &analyticalclient.StrategyWithTrace{
			Strategy: strategy.Strategy{
				Type: stratType, Source: "binancef", Instrument: instrumentFromVenue(symbol), Timeframe: 60,
				Direction: strategy.Direction(dir), Confidence: "0.80", Timestamp: now,
			},
			EventID: "str-" + corrID, CorrelationID: corrID, CausationID: "dec-" + corrID, OccurredAt: now,
		},
		Risk: &analyticalclient.RiskWithTrace{
			RiskAssessment: risk.RiskAssessment{
				Type: "position_exposure", Source: "binancef", Instrument: instrumentFromVenue(symbol), Timeframe: 60,
				Disposition: risk.Disposition(disposition), Confidence: "0.75", Rationale: rationale,
				Constraints: risk.Constraints{MaxPositionSize: maxPos, MaxExposure: maxExp},
				Strategies: []risk.StrategyInput{{
					Type: stratType, Direction: dir,
					Confidence: "0.80", DecisionSeverity: severity,
				}},
				Timestamp: now,
			},
			EventID: "rsk-" + corrID, CorrelationID: corrID, CausationID: "str-" + corrID, OccurredAt: now,
		},
		StageCount: 4,
	}

	if hasExec {
		side := "buy"
		if dir == "short" {
			side = "sell"
		}
		chain.Execution = &analyticalclient.ExecutionWithTrace{
			ExecutionIntent: execution.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: symbol, Timeframe: 60,
				Side: execution.Side(side), Quantity: "0.1", Status: "submitted", Timestamp: now,
			},
			EventID: "exc-" + corrID, EventCorrelationID: corrID, EventCausationID: "rsk-" + corrID, OccurredAt: now,
		}
		chain.StageCount = 5
		chain.ChainComplete = true
	} else {
		chain.ChainComplete = false
		chain.MissingStages = []string{"execution"}
	}

	return chain
}

func buildDispositions(approved, rejected, modified int) []analyticalclient.DispositionCount {
	var result []analyticalclient.DispositionCount
	if approved > 0 {
		result = append(result, analyticalclient.DispositionCount{Disposition: "approved", Count: int64(approved)})
	}
	if rejected > 0 {
		result = append(result, analyticalclient.DispositionCount{Disposition: "rejected", Count: int64(rejected)})
	}
	if modified > 0 {
		result = append(result, analyticalclient.DispositionCount{Disposition: "modified", Count: int64(modified)})
	}
	return result
}

// ---------------------------------------------------------------------------
// S303 stub reader for funnel + disposition queries
// ---------------------------------------------------------------------------

type s303FunnelStubReader struct {
	funnelPerSymbol map[string][]analyticalclient.StageFunnelCount
	dispPerSymbol   map[string][]analyticalclient.DispositionCount
}

func (r *s303FunnelStubReader) QueryPipelineFunnel(_ context.Context, _, _, symbol string, _ int, _, _ int64) ([]analyticalclient.StageFunnelCount, error) {
	if stages, ok := r.funnelPerSymbol[symbol]; ok {
		return stages, nil
	}
	return nil, nil
}

func (r *s303FunnelStubReader) QueryDispositionBreakdown(_ context.Context, _, _, symbol string, _ int, _, _ int64) ([]analyticalclient.DispositionCount, error) {
	if disps, ok := r.dispPerSymbol[symbol]; ok {
		return disps, nil
	}
	return nil, nil
}
