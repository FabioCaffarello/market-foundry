package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"internal/interfaces/http/handlers"
)

// TestS466_ParseQueryKeyParams_RequiresSource validates that missing source
// returns a 400 with a descriptive error message.
func TestS466_ParseQueryKeyParams_RequiresSource(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(
		&mockGetLatestCandle{},
		nil, nil, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestCandle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if body := rec.Body.String(); !contains(body, "source") {
		t.Fatalf("expected error to mention 'source', got: %s", body)
	}
}

// TestS466_ParseQueryKeyParams_RequiresSymbol validates that missing symbol
// returns a 400 with a descriptive error message.
func TestS466_ParseQueryKeyParams_RequiresSymbol(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(
		&mockGetLatestCandle{},
		nil, nil, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?source=binancef&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestCandle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if body := rec.Body.String(); !contains(body, "symbol") {
		t.Fatalf("expected error to mention 'symbol', got: %s", body)
	}
}

// TestS466_ParseQueryKeyParams_RequiresTimeframe validates that missing timeframe
// returns a 400.
func TestS466_ParseQueryKeyParams_RequiresTimeframe(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(
		&mockGetLatestCandle{},
		nil, nil, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestCandle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if body := rec.Body.String(); !contains(body, "timeframe") {
		t.Fatalf("expected error to mention 'timeframe', got: %s", body)
	}
}

// TestS466_AnalyticalLimitConstants verifies that the exported limit constants
// have the expected canonical values.
func TestS466_AnalyticalLimitConstants(t *testing.T) {
	if handlers.AnalyticalDefaultLimit != 50 {
		t.Fatalf("AnalyticalDefaultLimit = %d, want 50", handlers.AnalyticalDefaultLimit)
	}
	if handlers.AnalyticalMinLimit != 1 {
		t.Fatalf("AnalyticalMinLimit = %d, want 1", handlers.AnalyticalMinLimit)
	}
	if handlers.AnalyticalMaxLimit != 500 {
		t.Fatalf("AnalyticalMaxLimit = %d, want 500", handlers.AnalyticalMaxLimit)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsBytes(s, substr))
}

func containsBytes(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
