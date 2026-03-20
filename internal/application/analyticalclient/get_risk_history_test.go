package analyticalclient_test

import (
	"context"
	"errors"
	"testing"

	"internal/application/analyticalclient"
	"internal/domain/risk"
)

type stubRiskReader struct {
	assessments []risk.RiskAssessment
	err         error
}

func (s *stubRiskReader) QueryRiskHistory(_ context.Context, _, _, _ string, _ int, _ string, _, _ int64, _ int) ([]risk.RiskAssessment, error) {
	return s.assessments, s.err
}

func TestGetRiskHistoryUseCase_MissingType(t *testing.T) {
	uc := analyticalclient.NewGetRiskHistoryUseCase(&stubRiskReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing type")
	}
}

func TestGetRiskHistoryUseCase_MissingSource(t *testing.T) {
	uc := analyticalclient.NewGetRiskHistoryUseCase(&stubRiskReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing source")
	}
}

func TestGetRiskHistoryUseCase_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetRiskHistoryUseCase(&stubRiskReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Source:    "binancef",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
}

func TestGetRiskHistoryUseCase_InvalidTimeframe(t *testing.T) {
	uc := analyticalclient.NewGetRiskHistoryUseCase(&stubRiskReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 0,
	})
	if prob == nil {
		t.Fatal("expected problem for zero timeframe")
	}
}

func TestGetRiskHistoryUseCase_SinceAfterUntil(t *testing.T) {
	uc := analyticalclient.NewGetRiskHistoryUseCase(&stubRiskReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Since:     2000,
		Until:     1000,
	})
	if prob == nil {
		t.Fatal("expected problem for since > until")
	}
}

func TestGetRiskHistoryUseCase_DefaultLimit(t *testing.T) {
	reader := &stubRiskReader{assessments: []risk.RiskAssessment{}}
	uc := analyticalclient.NewGetRiskHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetRiskHistoryUseCase_LimitClamped(t *testing.T) {
	reader := &stubRiskReader{assessments: []risk.RiskAssessment{}}
	uc := analyticalclient.NewGetRiskHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Limit:     9999,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetRiskHistoryUseCase_WithDisposition(t *testing.T) {
	reader := &stubRiskReader{assessments: []risk.RiskAssessment{}}
	uc := analyticalclient.NewGetRiskHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:        "position_exposure",
		Source:      "binancef",
		Symbol:      "btcusdt",
		Timeframe:   60,
		Disposition: "approved",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetRiskHistoryUseCase_ReaderError(t *testing.T) {
	reader := &stubRiskReader{err: errors.New("connection refused")}
	uc := analyticalclient.NewGetRiskHistoryUseCase(reader, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for reader error")
	}
}

func TestGetRiskHistoryUseCase_NilReader(t *testing.T) {
	uc := analyticalclient.NewGetRiskHistoryUseCase(nil, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil reader")
	}
}

func TestGetRiskHistoryUseCase_NilUseCaseExecute(t *testing.T) {
	var uc *analyticalclient.GetRiskHistoryUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
}

func TestGetRiskHistoryUseCase_NegativeSince(t *testing.T) {
	uc := analyticalclient.NewGetRiskHistoryUseCase(&stubRiskReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.RiskHistoryQuery{
		Type:      "position_exposure",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Since:     -1,
	})
	if prob == nil {
		t.Fatal("expected problem for negative since")
	}
}
