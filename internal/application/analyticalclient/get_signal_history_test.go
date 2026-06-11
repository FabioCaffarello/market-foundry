package analyticalclient_test

import (
	"internal/domain/instrument"

	"context"
	"errors"
	"testing"

	"internal/application/analyticalclient"
	"internal/domain/signal"
)

type stubSignalReader struct {
	signals []signal.Signal
	err     error
}

func (s *stubSignalReader) QuerySignalHistory(_ context.Context, _, _ string, _ instrument.CanonicalInstrument, _ int, _, _ int64, _ int) ([]signal.Signal, error) {
	return s.signals, s.err
}

func TestGetSignalHistoryUseCase_MissingType(t *testing.T) {
	uc := analyticalclient.NewGetSignalHistoryUseCase(&stubSignalReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing type")
	}
}

func TestGetSignalHistoryUseCase_MissingSource(t *testing.T) {
	uc := analyticalclient.NewGetSignalHistoryUseCase(&stubSignalReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Type:       "rsi",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing source")
	}
}

func TestGetSignalHistoryUseCase_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetSignalHistoryUseCase(&stubSignalReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Type:      "rsi",
		Source:    "binancef",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
}

func TestGetSignalHistoryUseCase_InvalidTimeframe(t *testing.T) {
	uc := analyticalclient.NewGetSignalHistoryUseCase(&stubSignalReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Type:       "rsi",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  0,
	})
	if prob == nil {
		t.Fatal("expected problem for zero timeframe")
	}
}

func TestGetSignalHistoryUseCase_SinceAfterUntil(t *testing.T) {
	uc := analyticalclient.NewGetSignalHistoryUseCase(&stubSignalReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Type:       "rsi",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Since:      2000,
		Until:      1000,
	})
	if prob == nil {
		t.Fatal("expected problem for since > until")
	}
}

func TestGetSignalHistoryUseCase_DefaultLimit(t *testing.T) {
	reader := &stubSignalReader{signals: []signal.Signal{}}
	uc := analyticalclient.NewGetSignalHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Type:       "rsi",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetSignalHistoryUseCase_LimitClamped(t *testing.T) {
	reader := &stubSignalReader{signals: []signal.Signal{}}
	uc := analyticalclient.NewGetSignalHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Type:       "rsi",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Limit:      9999,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetSignalHistoryUseCase_ReaderError(t *testing.T) {
	reader := &stubSignalReader{err: errors.New("connection refused")}
	uc := analyticalclient.NewGetSignalHistoryUseCase(reader, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Type:       "rsi",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for reader error")
	}
}

func TestGetSignalHistoryUseCase_NilReader(t *testing.T) {
	uc := analyticalclient.NewGetSignalHistoryUseCase(nil, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Type:       "rsi",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil reader")
	}
}

func TestGetSignalHistoryUseCase_NilUseCaseExecute(t *testing.T) {
	var uc *analyticalclient.GetSignalHistoryUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.SignalHistoryQuery{
		Type:       "rsi",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
}
