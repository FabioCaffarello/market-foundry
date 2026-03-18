package signalclient_test

import (
	"context"
	"testing"
	"time"

	"internal/application/signalclient"
	"internal/domain/signal"
	"internal/shared/problem"
)

type mockSignalGateway struct {
	sig  *signal.Signal
	prob *problem.Problem
}

func (m *mockSignalGateway) GetLatestSignal(_ context.Context, _ signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem) {
	return signalclient.SignalLatestReply{Signal: m.sig}, m.prob
}

func TestGetLatestSignalUseCase_ValidatesInput(t *testing.T) {
	uc := signalclient.NewGetLatestSignalUseCase(&mockSignalGateway{})

	tests := []struct {
		name  string
		query signalclient.SignalLatestQuery
	}{
		{"empty type", signalclient.SignalLatestQuery{Type: "", Source: "binancef", Symbol: "btcusdt", Timeframe: 60}},
		{"empty source", signalclient.SignalLatestQuery{Type: "rsi", Source: "", Symbol: "btcusdt", Timeframe: 60}},
		{"empty symbol", signalclient.SignalLatestQuery{Type: "rsi", Source: "binancef", Symbol: "", Timeframe: 60}},
		{"zero timeframe", signalclient.SignalLatestQuery{Type: "rsi", Source: "binancef", Symbol: "btcusdt", Timeframe: 0}},
		{"negative timeframe", signalclient.SignalLatestQuery{Type: "rsi", Source: "binancef", Symbol: "btcusdt", Timeframe: -1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, prob := uc.Execute(context.Background(), tc.query)
			if prob == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestGetLatestSignalUseCase_ReturnsSignal(t *testing.T) {
	now := time.Now().UTC()
	sig := &signal.Signal{
		Type:      "rsi",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Value:     "65.32",
		Metadata:  map[string]string{"avg_gain": "1.20", "avg_loss": "0.64"},
		Timestamp: now,
		Final:     true,
	}

	uc := signalclient.NewGetLatestSignalUseCase(&mockSignalGateway{sig: sig})
	reply, prob := uc.Execute(context.Background(), signalclient.SignalLatestQuery{
		Type:      "rsi",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if reply.Signal == nil {
		t.Fatal("expected signal in reply")
	}
	if reply.Signal.Type != "rsi" {
		t.Fatalf("expected type rsi, got %s", reply.Signal.Type)
	}
	if reply.Signal.Value != "65.32" {
		t.Fatalf("expected value 65.32, got %s", reply.Signal.Value)
	}
}

func TestGetLatestSignalUseCase_NilGateway(t *testing.T) {
	var uc *signalclient.GetLatestSignalUseCase
	_, prob := uc.Execute(context.Background(), signalclient.SignalLatestQuery{
		Type:      "rsi",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected unavailable error")
	}
}
