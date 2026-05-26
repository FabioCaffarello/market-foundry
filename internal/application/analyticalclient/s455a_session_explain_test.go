package analyticalclient_test

import (
	"context"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/application/executionclient"
	"internal/domain/execution"
	"internal/shared/problem"
)

// s455aLifecycleReader implements analyticalclient.LifecycleHistoryReader for tests.
type s455aLifecycleReader struct {
	intents []execution.ExecutionIntent
	err     error
}

func (s *s455aLifecycleReader) QueryLifecycleHistory(_ context.Context, _, _ string, _ int, _, _ string, _, _ int64, _ int) ([]execution.ExecutionIntent, error) {
	return s.intents, s.err
}

// s455aKVReader implements analyticalclient.SessionExplainKVReader for tests.
type s455aKVReader struct {
	reply executionclient.ExecutionStatusReply
	prob  *problem.Problem
}

func (s *s455aKVReader) Execute(_ context.Context, _ executionclient.ExecutionStatusQuery) (executionclient.ExecutionStatusReply, *problem.Problem) {
	return s.reply, s.prob
}

func TestSessionExplain_ConsistentFilled(t *testing.T) {
	now := time.Now()
	intent := execution.ExecutionIntent{
		Type: "paper_order", Source: "binance_spot", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60,
		Side: execution.SideBuy, Quantity: "0.5", FilledQuantity: "0", Status: execution.StatusSubmitted,
		Risk:      execution.RiskInput{Type: "position_exposure", Disposition: "approved"},
		Timestamp: now.Add(-2 * time.Minute),
	}
	fill := execution.ExecutionIntent{
		Type: "venue_market_order", Source: "binance_spot", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60,
		Side: execution.SideBuy, Quantity: "0.5", FilledQuantity: "0.5", Status: execution.StatusFilled,
		Risk:      execution.RiskInput{Type: "position_exposure", Disposition: "approved"},
		Timestamp: now.Add(-1 * time.Minute),
	}

	chReader := &s455aLifecycleReader{intents: []execution.ExecutionIntent{fill, intent}} // newest first
	kvReader := &s455aKVReader{reply: executionclient.ExecutionStatusReply{
		Intent:      &intent,
		Result:      &fill,
		Propagation: "filled",
	}}

	uc := analyticalclient.NewGetSessionExplainUseCase(chReader, kvReader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.SessionExplainQuery{
		Source: "binance_spot", Symbol: "BTCUSDT", Timeframe: 60,
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if !reply.KVAvailable {
		t.Error("expected KV to be available")
	}
	if !reply.CHAvailable {
		t.Error("expected CH to be available")
	}
	if reply.KVPropagation != "filled" {
		t.Errorf("expected KV propagation 'filled', got %q", reply.KVPropagation)
	}
	if reply.CHPropagation != "filled" {
		t.Errorf("expected CH propagation 'filled', got %q", reply.CHPropagation)
	}
	if !reply.Consistent {
		t.Error("expected consistent=true for matching states")
		for _, c := range reply.Consistency {
			if c.Status == "divergent" {
				t.Logf("  divergent: %s kv=%q ch=%q detail=%q", c.Field, c.KVValue, c.CHValue, c.Detail)
			}
		}
	}
	if len(reply.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(reply.History))
	}
	if reply.Explanation == "" {
		t.Error("expected non-empty explanation")
	}
}

func TestSessionExplain_Divergent(t *testing.T) {
	now := time.Now()
	intent := execution.ExecutionIntent{
		Type: "paper_order", Source: "binance_spot", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60,
		Side: execution.SideBuy, Quantity: "1", FilledQuantity: "0", Status: execution.StatusSubmitted,
		Risk:      execution.RiskInput{Type: "position_exposure", Disposition: "approved"},
		Timestamp: now,
	}
	fill := execution.ExecutionIntent{
		Type: "venue_market_order", Source: "binance_spot", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60,
		Side: execution.SideBuy, Quantity: "1", FilledQuantity: "1", Status: execution.StatusFilled,
		Risk:      execution.RiskInput{Type: "position_exposure", Disposition: "approved"},
		Timestamp: now.Add(time.Minute),
	}

	// KV says filled, CH only has intent (no fill yet — simulates timing gap)
	chReader := &s455aLifecycleReader{intents: []execution.ExecutionIntent{intent}}
	kvReader := &s455aKVReader{reply: executionclient.ExecutionStatusReply{
		Intent:      &intent,
		Result:      &fill,
		Propagation: "filled",
	}}

	uc := analyticalclient.NewGetSessionExplainUseCase(chReader, kvReader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.SessionExplainQuery{
		Source: "binance_spot", Symbol: "BTCUSDT", Timeframe: 60,
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.Consistent {
		t.Error("expected consistent=false for divergent states")
	}

	hasDivergent := false
	for _, c := range reply.Consistency {
		if c.Status == "divergent" {
			hasDivergent = true
		}
	}
	if !hasDivergent {
		t.Error("expected at least one divergent consistency check")
	}
}

func TestSessionExplain_KVUnavailable(t *testing.T) {
	now := time.Now()
	intent := execution.ExecutionIntent{
		Type: "paper_order", Source: "binance_spot", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60,
		Side: execution.SideBuy, Quantity: "0.5", FilledQuantity: "0", Status: execution.StatusSubmitted,
		Risk:      execution.RiskInput{Type: "position_exposure", Disposition: "approved"},
		Timestamp: now,
	}

	chReader := &s455aLifecycleReader{intents: []execution.ExecutionIntent{intent}}
	kvReader := &s455aKVReader{prob: problem.New(problem.Unavailable, "nats down")}

	uc := analyticalclient.NewGetSessionExplainUseCase(chReader, kvReader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.SessionExplainQuery{
		Source: "binance_spot", Symbol: "BTCUSDT", Timeframe: 60,
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.KVAvailable {
		t.Error("expected KV to be unavailable")
	}
	if !reply.CHAvailable {
		t.Error("expected CH to be available")
	}
	if reply.Consistent {
		t.Error("expected consistent=false when KV unavailable")
	}
}

func TestSessionExplain_NilKVReader(t *testing.T) {
	now := time.Now()
	intent := execution.ExecutionIntent{
		Type: "paper_order", Source: "binance_spot", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60,
		Side: execution.SideBuy, Quantity: "0.5", FilledQuantity: "0", Status: execution.StatusSubmitted,
		Risk:      execution.RiskInput{Type: "position_exposure", Disposition: "approved"},
		Timestamp: now,
	}

	chReader := &s455aLifecycleReader{intents: []execution.ExecutionIntent{intent}}

	uc := analyticalclient.NewGetSessionExplainUseCase(chReader, nil, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.SessionExplainQuery{
		Source: "binance_spot", Symbol: "BTCUSDT", Timeframe: 60,
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.KVAvailable {
		t.Error("expected KV to be unavailable with nil reader")
	}
	if !reply.CHAvailable {
		t.Error("expected CH to be available")
	}
}

func TestSessionExplain_ValidationErrors(t *testing.T) {
	uc := analyticalclient.NewGetSessionExplainUseCase(&s455aLifecycleReader{}, nil, nil)

	tests := []struct {
		name  string
		query analyticalclient.SessionExplainQuery
	}{
		{"missing source", analyticalclient.SessionExplainQuery{Symbol: "BTCUSDT", Timeframe: 60}},
		{"missing symbol", analyticalclient.SessionExplainQuery{Source: "binance_spot", Timeframe: 60}},
		{"invalid timeframe", analyticalclient.SessionExplainQuery{Source: "binance_spot", Symbol: "BTCUSDT"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, prob := uc.Execute(context.Background(), tt.query)
			if prob == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestSessionExplain_RejectionConsistency(t *testing.T) {
	now := time.Now()
	intent := execution.ExecutionIntent{
		Type: "paper_order", Source: "binance_spot", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60,
		Side: execution.SideBuy, Quantity: "1", FilledQuantity: "0", Status: execution.StatusSubmitted,
		Risk:      execution.RiskInput{Type: "position_exposure", Disposition: "approved"},
		Timestamp: now.Add(-2 * time.Minute),
	}
	rejection := execution.ExecutionIntent{
		Type: "venue_rejection", Source: "binance_spot", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60,
		Side: execution.SideBuy, Quantity: "1", FilledQuantity: "0", Status: execution.StatusRejected,
		Risk:      execution.RiskInput{Type: "position_exposure", Disposition: "approved"},
		Metadata:  map[string]string{"rejection_code": "INSUFFICIENT_MARGIN", "rejection_reason": "margin too low"},
		Timestamp: now.Add(-1 * time.Minute),
	}

	chReader := &s455aLifecycleReader{intents: []execution.ExecutionIntent{rejection, intent}}
	kvReader := &s455aKVReader{reply: executionclient.ExecutionStatusReply{
		Intent:      &intent,
		Rejection:   &rejection,
		Propagation: "rejected",
	}}

	uc := analyticalclient.NewGetSessionExplainUseCase(chReader, kvReader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.SessionExplainQuery{
		Source: "binance_spot", Symbol: "BTCUSDT", Timeframe: 60,
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.KVPropagation != "rejected" {
		t.Errorf("expected KV propagation 'rejected', got %q", reply.KVPropagation)
	}
	if reply.CHPropagation != "rejected" {
		t.Errorf("expected CH propagation 'rejected', got %q", reply.CHPropagation)
	}
	if !reply.Consistent {
		t.Error("expected consistent=true for matching rejection states")
	}
}
