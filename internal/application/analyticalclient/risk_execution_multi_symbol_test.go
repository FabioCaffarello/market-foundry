package analyticalclient_test

// S304: Risk and Execution Behavior Under Multi-Symbol Concurrency
//
// Validates that the composite read model correctly reflects risk outcomes
// and execution paper intents when multiple symbols produce chains with
// different risk dispositions, strategy types, and execution states.
//
// Scenarios:
//   RX-1 — Three symbols: approved+filled, rejected+blocked, modified+filled.
//   RX-2 — Risk attribution diversity: each symbol has unique constraints and rationale.
//   RX-3 — Execution status coherence: filled chains vs blocked chains per symbol.
//   RX-4 — Cross-surface risk/execution alignment: funnel + disposition + chain.
//   RX-5 — Drawdown vs position_exposure risk type coexistence across symbols.

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/decision"
	"internal/domain/execution"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
)

// ---------------------------------------------------------------------------
// RX-1: Three symbols with approved/rejected/modified risk → execution coherence
// ---------------------------------------------------------------------------

func TestS304_RX1_RiskExecutionCoherence(t *testing.T) {
	now := time.Now()

	type symbolProfile struct {
		symbol       string
		riskDisp     string
		stratType    string
		severity     string
		direction    string
		maxPos       string
		maxExposure  string
		execSide     string
		hasExec      bool
		wantComplete bool
	}

	profiles := []symbolProfile{
		{
			symbol: "btcusdt", riskDisp: "approved", stratType: "mean_reversion_entry",
			severity: "high", direction: "long", maxPos: "0.0230", maxExposure: "0.1000",
			execSide: "buy", hasExec: true, wantComplete: true,
		},
		{
			symbol: "ethusdt", riskDisp: "rejected", stratType: "trend_following_entry",
			severity: "moderate", direction: "long", maxPos: "0.0190", maxExposure: "0.1000",
			hasExec: false, wantComplete: false,
		},
		{
			symbol: "solusdt", riskDisp: "modified", stratType: "squeeze_breakout_entry",
			severity: "low", direction: "short", maxPos: "0.0100", maxExposure: "0.0800",
			execSide: "sell", hasExec: true, wantComplete: true,
		},
	}

	chainsPerSymbol := make(map[string]*analyticalclient.CompositeExecutionChain)
	for _, p := range profiles {
		corrID := "s304-rx1-" + p.symbol
		chain := buildS304Chain(corrID, p.symbol, p.riskDisp, p.stratType, p.severity,
			p.direction, p.maxPos, p.maxExposure, p.hasExec, now)
		chainsPerSymbol[p.symbol] = chain
	}

	reader := &multiSymbolStubReader{chainsPerSymbol: chainsPerSymbol}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	for _, p := range profiles {
		t.Run(p.symbol+"_"+p.riskDisp, func(t *testing.T) {
			reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
				CorrelationID: "s304-rx1-" + p.symbol, Symbol: p.symbol,
			})
			if prob != nil {
				t.Fatalf("unexpected problem: %v", prob)
			}
			if len(reply.Chains) != 1 {
				t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
			}

			ch := reply.Chains[0]

			// Chain completeness matches disposition semantics.
			if ch.ChainComplete != p.wantComplete {
				t.Errorf("chain_complete=%v, want %v", ch.ChainComplete, p.wantComplete)
			}

			// Attribution disposition.
			if ch.Attribution == nil {
				t.Fatal("expected attribution")
			}
			if ch.Attribution.Disposition != p.riskDisp {
				t.Errorf("attribution.disposition=%q, want %q", ch.Attribution.Disposition, p.riskDisp)
			}

			// Execution presence.
			if p.hasExec {
				if ch.Execution == nil {
					t.Fatal("expected execution present")
				}
				if string(ch.Execution.Side) != p.execSide {
					t.Errorf("execution.side=%q, want %q", ch.Execution.Side, p.execSide)
				}
			} else {
				if ch.Execution != nil {
					t.Error("expected no execution for rejected risk")
				}
			}

			// Symbol isolation across all stages.
			if ch.Signal.VenueSymbol() != p.symbol {
				t.Errorf("signal.symbol=%q", ch.Signal.VenueSymbol())
			}
			if ch.Risk.VenueSymbol() != p.symbol {
				t.Errorf("risk.symbol=%q", ch.Risk.VenueSymbol())
			}

			// Risk constraints.
			if ch.Attribution.ActiveConstraints.MaxPositionSize != p.maxPos {
				t.Errorf("max_position_size=%q, want %q", ch.Attribution.ActiveConstraints.MaxPositionSize, p.maxPos)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RX-2: Risk attribution diversity per symbol
// ---------------------------------------------------------------------------

func TestS304_RX2_RiskAttributionDiversity(t *testing.T) {
	now := time.Now()

	type attrProfile struct {
		symbol    string
		riskDisp  string
		rationale string
		stratType string
		severity  string
		direction string
		maxPos    string
	}

	profiles := []attrProfile{
		{
			symbol: "btcusdt", riskDisp: "approved",
			rationale: "Position size 0.0170 within exposure limits; mean_reversion_entry (confidence ×0.90); decision severity high (limit ×1.15)",
			stratType: "mean_reversion_entry", severity: "high", direction: "long", maxPos: "0.0170",
		},
		{
			symbol: "ethusdt", riskDisp: "rejected",
			rationale: "Rejected: non-positive confidence 0.0000 for long",
			stratType: "trend_following_entry", severity: "moderate", direction: "long", maxPos: "0.0000",
		},
		{
			symbol: "solusdt", riskDisp: "modified",
			rationale: "Position size capped to 0.0080 by exposure limits; squeeze_breakout_entry (confidence ×0.93); decision severity low (limit ×0.80)",
			stratType: "squeeze_breakout_entry", severity: "low", direction: "short", maxPos: "0.0080",
		},
	}

	for _, p := range profiles {
		t.Run("attribution_"+p.symbol, func(t *testing.T) {
			corrID := "s304-rx2-" + p.symbol
			chain := buildS304DetailedChain(corrID, p.symbol, p.riskDisp, p.rationale,
				p.stratType, p.severity, p.direction, p.maxPos, "0.1000",
				p.riskDisp != "rejected", now)

			reader := &multiSymbolStubReader{
				chainsPerSymbol: map[string]*analyticalclient.CompositeExecutionChain{p.symbol: chain},
			}
			uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

			reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
				CorrelationID: corrID, Symbol: p.symbol,
			})
			if prob != nil {
				t.Fatalf("problem: %v", prob)
			}

			ch := reply.Chains[0]
			if ch.Attribution == nil {
				t.Fatal("missing attribution")
			}

			if ch.Attribution.Disposition != p.riskDisp {
				t.Errorf("disposition=%q, want %q", ch.Attribution.Disposition, p.riskDisp)
			}
			if ch.Attribution.Rationale != p.rationale {
				t.Errorf("rationale=%q, want %q", ch.Attribution.Rationale, p.rationale)
			}

			if len(ch.Attribution.StrategyContext) != 1 {
				t.Fatalf("expected 1 strategy context, got %d", len(ch.Attribution.StrategyContext))
			}
			sc := ch.Attribution.StrategyContext[0]
			if sc.Type != p.stratType {
				t.Errorf("strategy_context.type=%q, want %q", sc.Type, p.stratType)
			}
			if sc.DecisionSeverity != p.severity {
				t.Errorf("strategy_context.severity=%q, want %q", sc.DecisionSeverity, p.severity)
			}
			if sc.Direction != p.direction {
				t.Errorf("strategy_context.direction=%q, want %q", sc.Direction, p.direction)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RX-3: Execution status coherence per symbol
// ---------------------------------------------------------------------------

func TestS304_RX3_ExecutionStatusCoherence(t *testing.T) {
	now := time.Now()

	// Build chains: btcusdt approved+filled, ethusdt rejected+blocked, solusdt modified+filled.
	chains := map[string]*analyticalclient.CompositeExecutionChain{
		"btcusdt": buildS304Chain("s304-rx3-btcusdt", "btcusdt", "approved",
			"mean_reversion_entry", "high", "long", "0.0200", "0.1000", true, now),
		"ethusdt": buildS304Chain("s304-rx3-ethusdt", "ethusdt", "rejected",
			"trend_following_entry", "moderate", "long", "0.0000", "0.1000", false, now),
		"solusdt": buildS304Chain("s304-rx3-solusdt", "solusdt", "modified",
			"squeeze_breakout_entry", "low", "short", "0.0080", "0.0800", true, now),
	}

	reader := &multiSymbolStubReader{chainsPerSymbol: chains}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	// Approved: 5 stages, filled.
	t.Run("btcusdt_filled", func(t *testing.T) {
		reply, _ := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			CorrelationID: "s304-rx3-btcusdt", Symbol: "btcusdt",
		})
		ch := reply.Chains[0]
		if ch.StageCount != 5 {
			t.Errorf("stage_count=%d, want 5", ch.StageCount)
		}
		if ch.Execution == nil {
			t.Fatal("expected execution")
		}
		if ch.Execution.Status != "submitted" {
			t.Errorf("status=%q, want submitted", ch.Execution.Status)
		}
	})

	// Rejected: 4 stages, no execution.
	t.Run("ethusdt_blocked", func(t *testing.T) {
		reply, _ := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			CorrelationID: "s304-rx3-ethusdt", Symbol: "ethusdt",
		})
		ch := reply.Chains[0]
		if ch.StageCount != 4 {
			t.Errorf("stage_count=%d, want 4", ch.StageCount)
		}
		if ch.Execution != nil {
			t.Error("rejected should have no execution")
		}
		if len(ch.MissingStages) != 1 || ch.MissingStages[0] != "execution" {
			t.Errorf("missing_stages=%v, want [execution]", ch.MissingStages)
		}
	})

	// Modified: 5 stages, filled with capped quantity.
	t.Run("solusdt_modified_filled", func(t *testing.T) {
		reply, _ := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			CorrelationID: "s304-rx3-solusdt", Symbol: "solusdt",
		})
		ch := reply.Chains[0]
		if ch.StageCount != 5 {
			t.Errorf("stage_count=%d, want 5", ch.StageCount)
		}
		if ch.Execution == nil {
			t.Fatal("modified should have execution")
		}
		if ch.Execution.Side != "sell" {
			t.Errorf("side=%q, want sell", ch.Execution.Side)
		}
		if ch.Attribution.ActiveConstraints.MaxPositionSize != "0.0080" {
			t.Errorf("max_pos=%q, want 0.0080", ch.Attribution.ActiveConstraints.MaxPositionSize)
		}
	})
}

// ---------------------------------------------------------------------------
// RX-4: Cross-surface alignment: funnel + disposition + chain
// ---------------------------------------------------------------------------

func TestS304_RX4_CrossSurfaceRiskExecutionAlignment(t *testing.T) {
	// 3 symbols with known disposition distributions.
	funnelData := map[string][]analyticalclient.StageFunnelCount{
		"btcusdt": {
			{Stage: "signal", Count: 10}, {Stage: "decision", Count: 10},
			{Stage: "strategy", Count: 10}, {Stage: "risk", Count: 10},
			{Stage: "execution", Count: 8}, // 8 approved, 2 rejected
		},
		"ethusdt": {
			{Stage: "signal", Count: 5}, {Stage: "decision", Count: 5},
			{Stage: "strategy", Count: 5}, {Stage: "risk", Count: 5},
			{Stage: "execution", Count: 2}, // 2 approved, 3 rejected
		},
		"solusdt": {
			{Stage: "signal", Count: 3}, {Stage: "decision", Count: 3},
			{Stage: "strategy", Count: 3}, {Stage: "risk", Count: 3},
			{Stage: "execution", Count: 3}, // all approved
		},
	}

	dispData := map[string][]analyticalclient.DispositionCount{
		"btcusdt": {
			{Disposition: "approved", Count: 7},
			{Disposition: "modified", Count: 1},
			{Disposition: "rejected", Count: 2},
		},
		"ethusdt": {
			{Disposition: "approved", Count: 2},
			{Disposition: "rejected", Count: 3},
		},
		"solusdt": {
			{Disposition: "approved", Count: 2},
			{Disposition: "modified", Count: 1},
		},
	}

	reader := &s303FunnelStubReader{funnelPerSymbol: funnelData, dispPerSymbol: dispData}
	funnelUC := analyticalclient.NewGetPipelineFunnelUseCase(reader, slog.Default())
	dispUC := analyticalclient.NewGetDispositionBreakdownUseCase(reader, slog.Default())

	for _, sym := range []string{"btcusdt", "ethusdt", "solusdt"} {
		t.Run(sym, func(t *testing.T) {
			// Funnel query.
			fReply, prob := funnelUC.Execute(context.Background(), analyticalclient.PipelineFunnelQuery{
				Type: "position_exposure", Source: "binancef", Symbol: sym, Timeframe: 60,
			})
			if prob != nil {
				t.Fatalf("funnel: %v", prob)
			}

			// Execution count must be <= risk count.
			var riskCount, execCount int64
			for _, s := range fReply.Stages {
				if s.Stage == "risk" {
					riskCount = s.Count
				}
				if s.Stage == "execution" {
					execCount = s.Count
				}
			}
			if execCount > riskCount {
				t.Errorf("execution count (%d) exceeds risk count (%d)", execCount, riskCount)
			}

			// Disposition query.
			dReply, prob := dispUC.Execute(context.Background(), analyticalclient.DispositionBreakdownQuery{
				Type: "position_exposure", Source: "binancef", Symbol: sym, Timeframe: 60,
			})
			if prob != nil {
				t.Fatalf("disposition: %v", prob)
			}

			// Total dispositions = risk count.
			if dReply.Total != riskCount {
				t.Errorf("disposition total=%d, want %d (risk count)", dReply.Total, riskCount)
			}

			// Sum of approved+modified must equal execution count.
			var approvedModified int64
			for _, d := range dReply.Dispositions {
				if d.Disposition == "approved" || d.Disposition == "modified" {
					approvedModified += d.Count
				}
			}
			if approvedModified != execCount {
				t.Errorf("approved+modified=%d, want %d (execution count)", approvedModified, execCount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RX-5: Drawdown vs position_exposure risk type coexistence
// ---------------------------------------------------------------------------

func TestS304_RX5_RiskTypeCoexistence(t *testing.T) {
	now := time.Now()

	// btcusdt: position_exposure, ethusdt: drawdown_limit — same composite pipeline.
	chains := map[string]*analyticalclient.CompositeExecutionChain{
		"btcusdt": buildS304RiskTypeChain("s304-rx5-btcusdt", "btcusdt", "position_exposure",
			"approved", "within limits", "0.0200", "0.1000", true, now),
		"ethusdt": buildS304RiskTypeChain("s304-rx5-ethusdt", "ethusdt", "drawdown_limit",
			"approved", "stop within limits", "0.0000", "0.0500", true, now),
	}

	reader := &multiSymbolStubReader{chainsPerSymbol: chains}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	t.Run("btcusdt_position_exposure", func(t *testing.T) {
		reply, _ := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			CorrelationID: "s304-rx5-btcusdt", Symbol: "btcusdt",
		})
		ch := reply.Chains[0]
		if ch.Risk.Type != "position_exposure" {
			t.Errorf("risk.type=%q, want position_exposure", ch.Risk.Type)
		}
		if ch.Attribution.Disposition != "approved" {
			t.Errorf("attribution.disposition=%q", ch.Attribution.Disposition)
		}
	})

	t.Run("ethusdt_drawdown_limit", func(t *testing.T) {
		reply, _ := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			CorrelationID: "s304-rx5-ethusdt", Symbol: "ethusdt",
		})
		ch := reply.Chains[0]
		if ch.Risk.Type != "drawdown_limit" {
			t.Errorf("risk.type=%q, want drawdown_limit", ch.Risk.Type)
		}
		if ch.Attribution.Disposition != "approved" {
			t.Errorf("attribution.disposition=%q", ch.Attribution.Disposition)
		}
	})
}

// ---------------------------------------------------------------------------
// S304 fixture builders
// ---------------------------------------------------------------------------

func buildS304Chain(corrID, symbol, disposition, stratType, severity, dir, maxPos, maxExp string, hasExec bool, now time.Time) *analyticalclient.CompositeExecutionChain {
	rationale := "risk " + disposition + " for " + symbol
	return buildS304DetailedChain(corrID, symbol, disposition, rationale, stratType, severity, dir, maxPos, maxExp, hasExec, now)
}

func buildS304DetailedChain(corrID, symbol, disposition, rationale, stratType, severity, dir, maxPos, maxExp string, hasExec bool, now time.Time) *analyticalclient.CompositeExecutionChain {
	chain := &analyticalclient.CompositeExecutionChain{
		CorrelationID: corrID,
		Signal: &analyticalclient.SignalWithTrace{
			Signal: signal.Signal{
				Type: "rsi", Source: "binancef", Instrument: instrumentFromVenue(symbol), Timeframe: 60,
				Value: "35.0", Timestamp: now,
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
				Disposition: risk.Disposition(disposition), Confidence: "0.72", Rationale: rationale,
				Constraints: risk.Constraints{MaxPositionSize: maxPos, MaxExposure: maxExp},
				Strategies: []risk.StrategyInput{{
					Type: stratType, Direction: dir, Confidence: "0.80", DecisionSeverity: severity,
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
				Side: execution.Side(side), Quantity: maxPos, Status: "submitted", Timestamp: now,
				Risk: execution.RiskInput{
					Type: "position_exposure", Disposition: disposition,
					StrategyType: stratType, DecisionSeverity: severity,
				},
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

func buildS304RiskTypeChain(corrID, symbol, riskType, disposition, rationale, maxPos, maxExp string, hasExec bool, now time.Time) *analyticalclient.CompositeExecutionChain {
	chain := buildS304DetailedChain(corrID, symbol, disposition, rationale,
		"mean_reversion_entry", "moderate", "long", maxPos, maxExp, hasExec, now)
	// Override risk type.
	chain.Risk.Type = riskType
	if chain.Execution != nil {
		chain.Execution.Risk.Type = riskType
	}
	return chain
}
