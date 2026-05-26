package analyticalclient_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/decision"
	"internal/domain/execution"
	"internal/domain/instrument"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/problem"
)

// fullChainBTC is a package-level BTC/USDT-perpetual fixture used by chain
// builders in this test file. Built once via instrument.New to keep the
// fixture definition compact.
var fullChainBTC = func() instrument.CanonicalInstrument {
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		panic("test setup: BTC/USDT-perpetual: " + prob.Message)
	}
	return inst
}()

// stubCompositeReader implements analyticalclient.CompositeReader for unit tests.
type stubCompositeReader struct {
	chain     *analyticalclient.CompositeExecutionChain
	chains    []analyticalclient.CompositeExecutionChain
	singleErr error
	batchErr  error
}

func (s *stubCompositeReader) QueryChainByCorrelationID(_ context.Context, _, _ string) (*analyticalclient.CompositeExecutionChain, error) {
	return s.chain, s.singleErr
}

func (s *stubCompositeReader) QueryChainsBatch(_ context.Context, _, _ string, _ int, _, _ int64, _ int) ([]analyticalclient.CompositeExecutionChain, error) {
	return s.chains, s.batchErr
}

func fullChain(corrID string) *analyticalclient.CompositeExecutionChain {
	now := time.Now()
	return &analyticalclient.CompositeExecutionChain{
		CorrelationID: corrID,
		Signal: &analyticalclient.SignalWithTrace{
			Signal:        signal.Signal{Type: "rsi", Source: "binance", Instrument: fullChainBTC, Timeframe: 60, Value: "42.5", Timestamp: now},
			EventID:       "sig-001",
			CorrelationID: corrID,
			OccurredAt:    now,
		},
		Decision: &analyticalclient.DecisionWithTrace{
			Decision:      decision.Decision{Type: "rsi_oversold", Source: "binance", Instrument: fullChainBTC, Timeframe: 60, Outcome: "triggered", Severity: "high", Confidence: "0.85", Timestamp: now},
			EventID:       "dec-001",
			CorrelationID: corrID,
			CausationID:   "sig-001",
			OccurredAt:    now,
		},
		Strategy: &analyticalclient.StrategyWithTrace{
			Strategy:      strategy.Strategy{Type: "mean_reversion_entry", Source: "binance", Instrument: fullChainBTC, Timeframe: 60, Direction: "long", Confidence: "0.80", Timestamp: now},
			EventID:       "str-001",
			CorrelationID: corrID,
			CausationID:   "dec-001",
			OccurredAt:    now,
		},
		Risk: &analyticalclient.RiskWithTrace{
			RiskAssessment: risk.RiskAssessment{
				Type: "position_exposure", Source: "binance", Instrument: fullChainBTC, Timeframe: 60,
				Disposition: "approved", Confidence: "0.75", Rationale: "within limits",
				Constraints: risk.Constraints{MaxPositionSize: "0.1", MaxExposure: "1.0"},
				Strategies:  []risk.StrategyInput{{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.80", DecisionSeverity: "high"}},
				Timestamp:   now,
			},
			EventID:       "rsk-001",
			CorrelationID: corrID,
			CausationID:   "str-001",
			OccurredAt:    now,
		},
		Execution: &analyticalclient.ExecutionWithTrace{
			ExecutionIntent:    execution.ExecutionIntent{Type: "paper_order", Source: "binance", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60, Side: "buy", Quantity: "0.1", Status: "submitted", Timestamp: now},
			EventID:            "exc-001",
			EventCorrelationID: corrID,
			EventCausationID:   "rsk-001",
			OccurredAt:         now,
		},
		StageCount:    5,
		ChainComplete: true,
	}
}

func TestGetCompositeChain_Single_FullChain(t *testing.T) {
	reader := &stubCompositeReader{chain: fullChain("corr-001")}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		CorrelationID: "corr-001",
		Symbol:        "btcusdt",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
	}
	chain := reply.Chains[0]
	if chain.CorrelationID != "corr-001" {
		t.Errorf("correlation_id: got %q, want %q", chain.CorrelationID, "corr-001")
	}
	if !chain.ChainComplete {
		t.Error("expected chain_complete=true")
	}
	if chain.StageCount != 5 {
		t.Errorf("stage_count: got %d, want 5", chain.StageCount)
	}
	if chain.Signal == nil || chain.Decision == nil || chain.Strategy == nil || chain.Risk == nil || chain.Execution == nil {
		t.Error("expected all 5 stages to be present")
	}
	if reply.Source != "clickhouse" {
		t.Errorf("source: got %q, want %q", reply.Source, "clickhouse")
	}
}

func TestGetCompositeChain_Single_EmptyResult(t *testing.T) {
	reader := &stubCompositeReader{chain: &analyticalclient.CompositeExecutionChain{
		CorrelationID: "corr-missing",
		StageCount:    0,
	}}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		CorrelationID: "corr-missing",
		Symbol:        "btcusdt",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Chains) != 0 {
		t.Errorf("expected 0 chains for missing correlation, got %d", len(reply.Chains))
	}
}

func TestGetCompositeChain_Single_ReaderError(t *testing.T) {
	reader := &stubCompositeReader{singleErr: errors.New("connection refused")}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		CorrelationID: "corr-err",
		Symbol:        "btcusdt",
	})
	if prob == nil {
		t.Fatal("expected problem for reader error")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %q", prob.Code)
	}
}

func TestGetCompositeChain_Batch_Success(t *testing.T) {
	chains := []analyticalclient.CompositeExecutionChain{
		*fullChain("corr-a"),
		*fullChain("corr-b"),
	}
	reader := &stubCompositeReader{chains: chains}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		Source:    "binance",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Chains) != 2 {
		t.Errorf("expected 2 chains, got %d", len(reply.Chains))
	}
	if reply.Meta.ChainCount != 2 {
		t.Errorf("meta.chain_count: got %d, want 2", reply.Meta.ChainCount)
	}
}

func TestGetCompositeChain_Batch_MissingSource(t *testing.T) {
	uc := analyticalclient.NewGetCompositeChainUseCase(&stubCompositeReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing source")
	}
	if prob.Code != problem.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %q", prob.Code)
	}
}

func TestGetCompositeChain_Batch_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetCompositeChainUseCase(&stubCompositeReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		Source:    "binance",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
}

func TestGetCompositeChain_Batch_InvalidTimeframe(t *testing.T) {
	uc := analyticalclient.NewGetCompositeChainUseCase(&stubCompositeReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		Source:    "binance",
		Symbol:    "btcusdt",
		Timeframe: -1,
	})
	if prob == nil {
		t.Fatal("expected problem for invalid timeframe")
	}
}

func TestGetCompositeChain_Batch_LimitClamping(t *testing.T) {
	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{}}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	// Over max limit — should clamp silently.
	reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		Source:    "binance",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Limit:     999,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(reply.Chains) != 0 {
		t.Errorf("expected 0 chains, got %d", len(reply.Chains))
	}
}

func TestGetCompositeChain_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetCompositeChainUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		CorrelationID: "corr-nil",
		Symbol:        "btcusdt",
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %q", prob.Code)
	}
}

func TestGetCompositeChain_Single_Attribution(t *testing.T) {
	reader := &stubCompositeReader{chain: fullChain("corr-attr")}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		CorrelationID: "corr-attr",
		Symbol:        "btcusdt",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
	}
	chain := reply.Chains[0]
	if chain.Attribution == nil {
		t.Fatal("expected attribution to be present on chain with risk stage")
	}
	if chain.Attribution.Disposition != "approved" {
		t.Errorf("expected disposition approved, got %q", chain.Attribution.Disposition)
	}
	if chain.Attribution.Rationale != "within limits" {
		t.Errorf("expected rationale 'within limits', got %q", chain.Attribution.Rationale)
	}
	if chain.Attribution.ActiveConstraints.MaxPositionSize != "0.1" {
		t.Errorf("expected max_position_size 0.1, got %q", chain.Attribution.ActiveConstraints.MaxPositionSize)
	}
	if len(chain.Attribution.StrategyContext) != 1 {
		t.Fatalf("expected 1 strategy context, got %d", len(chain.Attribution.StrategyContext))
	}
	if chain.Attribution.StrategyContext[0].DecisionSeverity != "high" {
		t.Errorf("expected decision_severity high, got %q", chain.Attribution.StrategyContext[0].DecisionSeverity)
	}
}

func TestGetCompositeChain_Single_NoRisk_NoAttribution(t *testing.T) {
	now := time.Now()
	chain := &analyticalclient.CompositeExecutionChain{
		CorrelationID: "corr-no-risk",
		Signal: &analyticalclient.SignalWithTrace{
			Signal:     signal.Signal{Type: "rsi", Source: "binance", Instrument: fullChainBTC, Timeframe: 60, Value: "42.5", Timestamp: now},
			OccurredAt: now,
		},
		StageCount: 1,
	}
	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		CorrelationID: "corr-no-risk",
		Symbol:        "btcusdt",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(reply.Chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(reply.Chains))
	}
	if reply.Chains[0].Attribution != nil {
		t.Error("expected no attribution when risk stage is absent")
	}
}

func TestGetCompositeChain_Single_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetCompositeChainUseCase(&stubCompositeReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		CorrelationID: "corr-no-symbol",
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol in single-chain mode")
	}
	if prob.Code != problem.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %q", prob.Code)
	}
}

func TestGetCompositeChain_Batch_Attribution(t *testing.T) {
	chains := []analyticalclient.CompositeExecutionChain{
		*fullChain("corr-batch-a"),
		*fullChain("corr-batch-b"),
	}
	reader := &stubCompositeReader{chains: chains}
	uc := analyticalclient.NewGetCompositeChainUseCase(reader, slog.Default())

	reply, prob := uc.Execute(context.Background(), analyticalclient.CompositeChainQuery{
		Source: "binance", Symbol: "btcusdt", Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	for i, ch := range reply.Chains {
		if ch.Attribution == nil {
			t.Errorf("chain[%d]: expected attribution to be present", i)
		}
	}
}
