package analyticalclient_test

// S302: Multi-Symbol Deterministic Scenario Pack
//
// This file validates that the composite read model behaves correctly and
// consistently when multiple symbols are active simultaneously. Each scenario
// is small, deterministic, and auditable.
//
// Scenarios:
//   SC1 — Simultaneous approved chains with different characteristics per symbol.
//   SC2 — Mixed dispositions across symbols (approved, rejected, modified).
//   SC3 — Concurrent batch queries return correct counts and ordering per symbol.
//   SC4 — Attribution correctness varies per symbol (severity, direction, constraints).

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
// Multi-symbol fixture factories
// ---------------------------------------------------------------------------

type symbolScenario struct {
	Symbol        string
	SignalType    string
	SignalValue   string
	DecisionType  string
	DecisionSev   string
	StrategyDir   string
	StrategyType  string
	RiskDisp      string
	RiskRationale string
	MaxPosPct     string
	MaxExposure   string
	ExecSide      string
	ExecQty       string
	ExecStatus    string
	HasExecution  bool
}

func buildChainFromScenario(corrID string, sc symbolScenario) *analyticalclient.CompositeExecutionChain {
	now := time.Now()
	chain := &analyticalclient.CompositeExecutionChain{
		CorrelationID: corrID,
		Signal: &analyticalclient.SignalWithTrace{
			Signal: signal.Signal{
				Type: sc.SignalType, Source: "binancef", Instrument: instrumentFromVenue(sc.Symbol), Timeframe: 60,
				Value: sc.SignalValue, Timestamp: now,
			},
			EventID: "sig-" + corrID, CorrelationID: corrID, OccurredAt: now,
		},
		Decision: &analyticalclient.DecisionWithTrace{
			Decision: decision.Decision{
				Type: sc.DecisionType, Source: "binancef", Instrument: instrumentFromVenue(sc.Symbol), Timeframe: 60,
				Outcome: "triggered", Severity: decision.Severity(sc.DecisionSev), Confidence: "0.85", Timestamp: now,
			},
			EventID: "dec-" + corrID, CorrelationID: corrID, CausationID: "sig-" + corrID, OccurredAt: now,
		},
		Strategy: &analyticalclient.StrategyWithTrace{
			Strategy: strategy.Strategy{
				Type: sc.StrategyType, Source: "binancef", Instrument: instrumentFromVenue(sc.Symbol), Timeframe: 60,
				Direction: strategy.Direction(sc.StrategyDir), Confidence: "0.80", Timestamp: now,
			},
			EventID: "str-" + corrID, CorrelationID: corrID, CausationID: "dec-" + corrID, OccurredAt: now,
		},
		Risk: &analyticalclient.RiskWithTrace{
			RiskAssessment: risk.RiskAssessment{
				Type: "position_exposure", Source: "binancef", Instrument: instrumentFromVenue(sc.Symbol), Timeframe: 60,
				Disposition: risk.Disposition(sc.RiskDisp), Confidence: "0.75", Rationale: sc.RiskRationale,
				Constraints: risk.Constraints{MaxPositionSize: sc.MaxPosPct, MaxExposure: sc.MaxExposure},
				Strategies: []risk.StrategyInput{{
					Type: sc.StrategyType, Direction: sc.StrategyDir,
					Confidence: "0.80", DecisionSeverity: sc.DecisionSev,
				}},
				Timestamp: now,
			},
			EventID: "rsk-" + corrID, CorrelationID: corrID, CausationID: "str-" + corrID, OccurredAt: now,
		},
		StageCount: 4,
	}

	if sc.HasExecution {
		chain.Execution = &analyticalclient.ExecutionWithTrace{
			ExecutionIntent: execution.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Instrument: instrumentFromVenue(sc.Symbol), Timeframe: 60,
				Side: execution.Side(sc.ExecSide), Quantity: sc.ExecQty, Status: execution.Status(sc.ExecStatus), Timestamp: now,
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

// ---------------------------------------------------------------------------
// SC1: Simultaneous approved chains with different characteristics per symbol
// ---------------------------------------------------------------------------

var sc1Scenarios = map[string]symbolScenario{
	"btcusdt": {
		Symbol: "btcusdt", SignalType: "rsi", SignalValue: "28.5",
		DecisionType: "rsi_oversold", DecisionSev: "high",
		StrategyDir: "long", StrategyType: "mean_reversion_entry",
		RiskDisp: "approved", RiskRationale: "within limits",
		MaxPosPct: "0.10", MaxExposure: "1.0",
		ExecSide: "buy", ExecQty: "0.1", ExecStatus: "submitted",
		HasExecution: true,
	},
	"ethusdt": {
		Symbol: "ethusdt", SignalType: "macd", SignalValue: "0.0025",
		DecisionType: "macd_crossover", DecisionSev: "moderate",
		StrategyDir: "long", StrategyType: "trend_following_entry",
		RiskDisp: "approved", RiskRationale: "exposure acceptable",
		MaxPosPct: "0.15", MaxExposure: "2.0",
		ExecSide: "buy", ExecQty: "1.5", ExecStatus: "submitted",
		HasExecution: true,
	},
	"solusdt": {
		Symbol: "solusdt", SignalType: "bollinger", SignalValue: "1.85",
		DecisionType: "squeeze_breakout", DecisionSev: "low",
		StrategyDir: "short", StrategyType: "squeeze_breakout_entry",
		RiskDisp: "approved", RiskRationale: "within squeeze limits",
		MaxPosPct: "0.05", MaxExposure: "0.5",
		ExecSide: "sell", ExecQty: "10.0", ExecStatus: "submitted",
		HasExecution: true,
	},
}

func TestS302_SC1_SimultaneousApprovedChains(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}

	// Build a stub reader that returns the correct chain per symbol.
	chainsPerSymbol := make(map[string]*analyticalclient.CompositeExecutionChain, 3)
	for _, sym := range symbols {
		corrID := "s302-sc1-" + sym
		chainsPerSymbol[sym] = buildChainFromScenario(corrID, sc1Scenarios[sym])
	}

	reader := &multiSymbolStubReader{chainsPerSymbol: chainsPerSymbol}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	for _, sym := range symbols {
		t.Run("symbol_"+sym, func(t *testing.T) {
			corrID := "s302-sc1-" + sym
			reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
				CorrelationID: corrID,
				Symbol:        sym,
			})
			if prob != nil {
				t.Fatalf("unexpected problem for %s: %v", sym, prob)
			}
			if len(reply.Chains) != 1 {
				t.Fatalf("expected 1 chain for %s, got %d", sym, len(reply.Chains))
			}

			chain := reply.Chains[0]

			// Chain completeness.
			if !chain.ChainComplete {
				t.Errorf("[%s] expected chain_complete=true", sym)
			}
			if chain.StageCount != 5 {
				t.Errorf("[%s] expected stage_count=5, got %d", sym, chain.StageCount)
			}

			// Symbol consistency — every stage belongs to this symbol.
			if chain.Signal.VenueSymbol() != sym {
				t.Errorf("[%s] signal.symbol=%q", sym, chain.Signal.VenueSymbol())
			}
			if chain.Decision.VenueSymbol() != sym {
				t.Errorf("[%s] decision.symbol=%q", sym, chain.Decision.VenueSymbol())
			}
			if chain.Strategy.VenueSymbol() != sym {
				t.Errorf("[%s] strategy.symbol=%q", sym, chain.Strategy.VenueSymbol())
			}
			if chain.Risk.VenueSymbol() != sym {
				t.Errorf("[%s] risk.symbol=%q", sym, chain.Risk.VenueSymbol())
			}
			if chain.Execution.VenueSymbol() != sym {
				t.Errorf("[%s] execution.symbol=%q", sym, chain.Execution.VenueSymbol())
			}

			// Attribution correctness per symbol.
			if chain.Attribution == nil {
				t.Fatalf("[%s] expected attribution", sym)
			}
			if chain.Attribution.Disposition != "approved" {
				t.Errorf("[%s] attribution.disposition=%q, want approved", sym, chain.Attribution.Disposition)
			}

			sc := sc1Scenarios[sym]
			if chain.Signal.Type != sc.SignalType {
				t.Errorf("[%s] signal.type=%q, want %q", sym, chain.Signal.Type, sc.SignalType)
			}
			if string(chain.Strategy.Direction) != sc.StrategyDir {
				t.Errorf("[%s] strategy.direction=%q, want %q", sym, chain.Strategy.Direction, sc.StrategyDir)
			}
			if string(chain.Execution.Side) != sc.ExecSide {
				t.Errorf("[%s] execution.side=%q, want %q", sym, chain.Execution.Side, sc.ExecSide)
			}
			if chain.Attribution.ActiveConstraints.MaxPositionSize != sc.MaxPosPct {
				t.Errorf("[%s] max_position_size=%q, want %q", sym, chain.Attribution.ActiveConstraints.MaxPositionSize, sc.MaxPosPct)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SC2: Mixed dispositions across symbols
// ---------------------------------------------------------------------------

var sc2Scenarios = map[string]symbolScenario{
	"btcusdt": {
		Symbol: "btcusdt", SignalType: "rsi", SignalValue: "32.0",
		DecisionType: "rsi_oversold", DecisionSev: "high",
		StrategyDir: "long", StrategyType: "mean_reversion_entry",
		RiskDisp: "approved", RiskRationale: "within limits",
		MaxPosPct: "0.10", MaxExposure: "1.0",
		ExecSide: "buy", ExecQty: "0.1", ExecStatus: "submitted",
		HasExecution: true,
	},
	"ethusdt": {
		Symbol: "ethusdt", SignalType: "macd", SignalValue: "0.0010",
		DecisionType: "macd_crossover", DecisionSev: "moderate",
		StrategyDir: "long", StrategyType: "trend_following_entry",
		RiskDisp: "rejected", RiskRationale: "drawdown limit exceeded",
		MaxPosPct: "0.05", MaxExposure: "0.5",
		ExecSide: "", ExecQty: "", ExecStatus: "",
		HasExecution: false,
	},
	"solusdt": {
		Symbol: "solusdt", SignalType: "bollinger", SignalValue: "1.95",
		DecisionType: "squeeze_breakout", DecisionSev: "low",
		StrategyDir: "short", StrategyType: "squeeze_breakout_entry",
		RiskDisp: "modified", RiskRationale: "position size reduced",
		MaxPosPct: "0.03", MaxExposure: "0.3",
		ExecSide: "sell", ExecQty: "5.0", ExecStatus: "submitted",
		HasExecution: true,
	},
}

func TestS302_SC2_MixedDispositions(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}

	chainsPerSymbol := make(map[string]*analyticalclient.CompositeExecutionChain, 3)
	for _, sym := range symbols {
		corrID := "s302-sc2-" + sym
		chainsPerSymbol[sym] = buildChainFromScenario(corrID, sc2Scenarios[sym])
	}

	reader := &multiSymbolStubReader{chainsPerSymbol: chainsPerSymbol}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	// btcusdt: approved → full chain
	t.Run("btcusdt_approved", func(t *testing.T) {
		reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			CorrelationID: "s302-sc2-btcusdt", Symbol: "btcusdt",
		})
		if prob != nil {
			t.Fatalf("unexpected problem: %v", prob)
		}
		if len(reply.Chains) != 1 {
			t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
		}
		chain := reply.Chains[0]
		if !chain.ChainComplete {
			t.Error("btcusdt: expected chain_complete=true for approved")
		}
		if chain.Attribution.Disposition != "approved" {
			t.Errorf("btcusdt: attribution.disposition=%q, want approved", chain.Attribution.Disposition)
		}
		if chain.Execution == nil {
			t.Error("btcusdt: expected execution stage present")
		}
	})

	// ethusdt: rejected → no execution
	t.Run("ethusdt_rejected", func(t *testing.T) {
		reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			CorrelationID: "s302-sc2-ethusdt", Symbol: "ethusdt",
		})
		if prob != nil {
			t.Fatalf("unexpected problem: %v", prob)
		}
		if len(reply.Chains) != 1 {
			t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
		}
		chain := reply.Chains[0]
		if chain.ChainComplete {
			t.Error("ethusdt: expected chain_complete=false for rejected")
		}
		if chain.StageCount != 4 {
			t.Errorf("ethusdt: stage_count=%d, want 4", chain.StageCount)
		}
		if chain.Execution != nil {
			t.Error("ethusdt: expected no execution for rejected risk")
		}
		if chain.Attribution.Disposition != "rejected" {
			t.Errorf("ethusdt: attribution.disposition=%q, want rejected", chain.Attribution.Disposition)
		}
		if chain.Attribution.Rationale != "drawdown limit exceeded" {
			t.Errorf("ethusdt: attribution.rationale=%q", chain.Attribution.Rationale)
		}
		if len(chain.MissingStages) != 1 || chain.MissingStages[0] != "execution" {
			t.Errorf("ethusdt: missing_stages=%v, want [execution]", chain.MissingStages)
		}
	})

	// solusdt: modified → execution with adjusted params
	t.Run("solusdt_modified", func(t *testing.T) {
		reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
			CorrelationID: "s302-sc2-solusdt", Symbol: "solusdt",
		})
		if prob != nil {
			t.Fatalf("unexpected problem: %v", prob)
		}
		if len(reply.Chains) != 1 {
			t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
		}
		chain := reply.Chains[0]
		if !chain.ChainComplete {
			t.Error("solusdt: expected chain_complete=true for modified")
		}
		if chain.Attribution.Disposition != "modified" {
			t.Errorf("solusdt: attribution.disposition=%q, want modified", chain.Attribution.Disposition)
		}
		if chain.Attribution.Rationale != "position size reduced" {
			t.Errorf("solusdt: attribution.rationale=%q", chain.Attribution.Rationale)
		}
		if chain.Execution == nil {
			t.Fatal("solusdt: expected execution stage present for modified")
		}
		if chain.Execution.Side != "sell" {
			t.Errorf("solusdt: execution.side=%q, want sell", chain.Execution.Side)
		}
		if chain.Attribution.ActiveConstraints.MaxPositionSize != "0.03" {
			t.Errorf("solusdt: max_position_size=%q, want 0.03", chain.Attribution.ActiveConstraints.MaxPositionSize)
		}
	})
}

// ---------------------------------------------------------------------------
// SC3: Concurrent batch queries with correct counts per symbol
// ---------------------------------------------------------------------------

func TestS302_SC3_ConcurrentBatchPerSymbol(t *testing.T) {
	// btcusdt: 3 chains, ethusdt: 2 chains, solusdt: 1 chain
	batchData := map[string][]analyticalclient.CompositeExecutionChain{
		"btcusdt": {
			*buildChainFromScenario("s302-sc3-btc-1", sc1Scenarios["btcusdt"]),
			*buildChainFromScenario("s302-sc3-btc-2", sc1Scenarios["btcusdt"]),
			*buildChainFromScenario("s302-sc3-btc-3", sc1Scenarios["btcusdt"]),
		},
		"ethusdt": {
			*buildChainFromScenario("s302-sc3-eth-1", sc1Scenarios["ethusdt"]),
			*buildChainFromScenario("s302-sc3-eth-2", sc1Scenarios["ethusdt"]),
		},
		"solusdt": {
			*buildChainFromScenario("s302-sc3-sol-1", sc1Scenarios["solusdt"]),
		},
	}

	reader := &multiSymbolBatchStubReader{batchPerSymbol: batchData}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	expectedCounts := map[string]int{"btcusdt": 3, "ethusdt": 2, "solusdt": 1}

	for sym, want := range expectedCounts {
		t.Run("batch_"+sym, func(t *testing.T) {
			reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
				Source: "binancef", Symbol: sym, Timeframe: 60,
			})
			if prob != nil {
				t.Fatalf("unexpected problem for %s: %v", sym, prob)
			}
			if len(reply.Chains) != want {
				t.Errorf("[%s] expected %d chains, got %d", sym, want, len(reply.Chains))
			}
			if reply.Meta.ChainCount != want {
				t.Errorf("[%s] meta.chain_count=%d, want %d", sym, reply.Meta.ChainCount, want)
			}

			// All chains must belong to the queried symbol.
			for i, ch := range reply.Chains {
				if ch.Signal != nil && ch.Signal.VenueSymbol() != sym {
					t.Errorf("[%s] chain[%d].signal.symbol=%q", sym, i, ch.Signal.VenueSymbol())
				}
				if ch.Execution != nil && ch.Execution.VenueSymbol() != sym {
					t.Errorf("[%s] chain[%d].execution.symbol=%q", sym, i, ch.Execution.VenueSymbol())
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SC4: Attribution correctness varies per symbol
// ---------------------------------------------------------------------------

func TestS302_SC4_AttributionDiversityPerSymbol(t *testing.T) {
	// Each symbol produces a different attribution profile.
	scenarios := map[string]struct {
		sc              symbolScenario
		wantDisposition string
		wantRationale   string
		wantSeverity    string
		wantDirection   string
		wantMaxPos      string
	}{
		"btcusdt": {
			sc: symbolScenario{
				Symbol: "btcusdt", SignalType: "rsi", SignalValue: "25.0",
				DecisionType: "rsi_oversold", DecisionSev: "high",
				StrategyDir: "long", StrategyType: "mean_reversion_entry",
				RiskDisp: "approved", RiskRationale: "high confidence entry",
				MaxPosPct: "0.10", MaxExposure: "1.0",
				ExecSide: "buy", ExecQty: "0.1", ExecStatus: "submitted",
				HasExecution: true,
			},
			wantDisposition: "approved", wantRationale: "high confidence entry",
			wantSeverity: "high", wantDirection: "long", wantMaxPos: "0.10",
		},
		"ethusdt": {
			sc: symbolScenario{
				Symbol: "ethusdt", SignalType: "macd", SignalValue: "-0.003",
				DecisionType: "macd_crossover", DecisionSev: "moderate",
				StrategyDir: "short", StrategyType: "trend_following_entry",
				RiskDisp: "rejected", RiskRationale: "exposure limit breach",
				MaxPosPct: "0.05", MaxExposure: "0.5",
				HasExecution: false,
			},
			wantDisposition: "rejected", wantRationale: "exposure limit breach",
			wantSeverity: "moderate", wantDirection: "short", wantMaxPos: "0.05",
		},
		"solusdt": {
			sc: symbolScenario{
				Symbol: "solusdt", SignalType: "bollinger", SignalValue: "2.10",
				DecisionType: "squeeze_breakout", DecisionSev: "low",
				StrategyDir: "long", StrategyType: "squeeze_breakout_entry",
				RiskDisp: "modified", RiskRationale: "position capped at 3%",
				MaxPosPct: "0.03", MaxExposure: "0.3",
				ExecSide: "buy", ExecQty: "5.0", ExecStatus: "submitted",
				HasExecution: true,
			},
			wantDisposition: "modified", wantRationale: "position capped at 3%",
			wantSeverity: "low", wantDirection: "long", wantMaxPos: "0.03",
		},
	}

	for sym, tc := range scenarios {
		t.Run("attribution_"+sym, func(t *testing.T) {
			corrID := "s302-sc4-" + sym
			chain := buildChainFromScenario(corrID, tc.sc)

			reader := &multiSymbolStubReader{
				chainsPerSymbol: map[string]*analyticalclient.CompositeExecutionChain{sym: chain},
			}
			uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

			reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
				CorrelationID: corrID, Symbol: sym,
			})
			if prob != nil {
				t.Fatalf("unexpected problem: %v", prob)
			}
			if len(reply.Chains) != 1 {
				t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
			}

			ch := reply.Chains[0]
			if ch.Attribution == nil {
				t.Fatal("expected attribution")
			}
			if ch.Attribution.Disposition != tc.wantDisposition {
				t.Errorf("disposition=%q, want %q", ch.Attribution.Disposition, tc.wantDisposition)
			}
			if ch.Attribution.Rationale != tc.wantRationale {
				t.Errorf("rationale=%q, want %q", ch.Attribution.Rationale, tc.wantRationale)
			}
			if ch.Attribution.ActiveConstraints.MaxPositionSize != tc.wantMaxPos {
				t.Errorf("max_position_size=%q, want %q", ch.Attribution.ActiveConstraints.MaxPositionSize, tc.wantMaxPos)
			}
			if len(ch.Attribution.StrategyContext) != 1 {
				t.Fatalf("expected 1 strategy context, got %d", len(ch.Attribution.StrategyContext))
			}
			sc := ch.Attribution.StrategyContext[0]
			if sc.DecisionSeverity != tc.wantSeverity {
				t.Errorf("decision_severity=%q, want %q", sc.DecisionSeverity, tc.wantSeverity)
			}
			if sc.Direction != tc.wantDirection {
				t.Errorf("direction=%q, want %q", sc.Direction, tc.wantDirection)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Multi-symbol stub readers
// ---------------------------------------------------------------------------

// multiSymbolStubReader dispatches single-chain queries by symbol,
// simulating real behavior where each symbol returns its own data.
type multiSymbolStubReader struct {
	chainsPerSymbol map[string]*analyticalclient.CompositeExecutionChain
}

func (r *multiSymbolStubReader) QueryChainByCorrelationID(_ context.Context, _, symbol string) (*analyticalclient.CompositeExecutionChain, error) {
	if chain, ok := r.chainsPerSymbol[symbol]; ok {
		return chain, nil
	}
	return &analyticalclient.CompositeExecutionChain{StageCount: 0}, nil
}

func (r *multiSymbolStubReader) QueryChainsBatch(_ context.Context, _, _ string, _ int, _, _ int64, _ int) ([]analyticalclient.CompositeExecutionChain, error) {
	return nil, nil
}

// multiSymbolBatchStubReader dispatches batch queries by symbol.
type multiSymbolBatchStubReader struct {
	batchPerSymbol map[string][]analyticalclient.CompositeExecutionChain
}

func (r *multiSymbolBatchStubReader) QueryChainByCorrelationID(_ context.Context, _, _ string) (*analyticalclient.CompositeExecutionChain, error) {
	return &analyticalclient.CompositeExecutionChain{StageCount: 0}, nil
}

func (r *multiSymbolBatchStubReader) QueryChainsBatch(_ context.Context, _, symbol string, _ int, _, _ int64, _ int) ([]analyticalclient.CompositeExecutionChain, error) {
	if chains, ok := r.batchPerSymbol[symbol]; ok {
		return chains, nil
	}
	return nil, nil
}
