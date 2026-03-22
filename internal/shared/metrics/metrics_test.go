package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerFunc_ServesPrometheusMetrics(t *testing.T) {
	handler := HandlerFunc()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	// Default Prometheus registry always includes go_* metrics.
	if !strings.Contains(string(body), "go_goroutines") {
		t.Fatal("expected Go runtime metrics in /metrics output")
	}
}

func TestObserveHTTPRequest_RecordsMetrics(t *testing.T) {
	// Observe a request — this should not panic.
	ObserveHTTPRequest("GET", "/healthz", 200, 5*time.Millisecond)
	ObserveHTTPRequest("POST", "/configs", 201, 50*time.Millisecond)
	ObserveHTTPRequest("GET", "/healthz", 503, 100*time.Millisecond)

	// Verify the metrics appear in the handler output.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	HandlerFunc()(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "marketfoundry_http_request_duration_seconds") {
		t.Error("expected http_request_duration_seconds in metrics output")
	}
	if !strings.Contains(body, "marketfoundry_http_requests_total") {
		t.Error("expected http_requests_total in metrics output")
	}
}

func TestInstrumentHTTPHandler_CapturesStatusCode(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	wrapped := InstrumentHTTPHandler("GET", "/test", inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	wrapped(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestConsumerMetrics_DoNotPanic(t *testing.T) {
	// Verify consumer metric helpers are callable without panic.
	IncConsumerMessage("test-consumer", "delivered")
	IncConsumerMessage("test-consumer", "redelivered")
	IncConsumerMessage("test-consumer", "terminated")
	IncConsumerMessage("test-consumer", "nakked")
	ObserveConsumerProcessing("test-consumer", 10*time.Millisecond)
	SetConsumerLag("test-consumer", 42)

	// Verify consumer metrics appear in output.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	HandlerFunc()(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "marketfoundry_consumer_messages_total") {
		t.Error("expected consumer_messages_total in metrics output")
	}
	if !strings.Contains(body, "marketfoundry_consumer_processing_duration_seconds") {
		t.Error("expected consumer_processing_duration_seconds in metrics output")
	}
	if !strings.Contains(body, "marketfoundry_consumer_lag_messages") {
		t.Error("expected consumer_lag_messages in metrics output")
	}
}

func TestExecutionMetrics_DoNotPanic(t *testing.T) {
	// Verify execution metric helpers are callable without panic.
	IncStrategyEvaluation("mean_reversion_entry", "actionable")
	IncStrategyEvaluation("mean_reversion_entry", "flat")
	IncStrategyEvaluation("mean_reversion_entry", "skipped_low_confidence")
	IncStrategyEvaluation("mean_reversion_entry", "skipped_wrong_type")
	IncStrategyEvaluation("mean_reversion_entry", "error")
	IncGateCheck("kill_switch", "blocked")
	IncGateCheck("staleness", "blocked")
	IncGateCheck("all", "allowed")
	IncExecutionIntent("strategy_consumer.mean_reversion_entry", "buy")
	IncExecutionIntent("strategy_consumer.mean_reversion_entry", "sell")
	IncExecutionIntent("strategy_consumer.mean_reversion_entry", "none")
	SetGateActive(true)
	SetGateActive(false)

	// Verify execution metrics appear in output.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	HandlerFunc()(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "marketfoundry_execution_strategy_evaluations_total") {
		t.Error("expected execution_strategy_evaluations_total in metrics output")
	}
	if !strings.Contains(body, "marketfoundry_execution_gate_checks_total") {
		t.Error("expected execution_gate_checks_total in metrics output")
	}
	if !strings.Contains(body, "marketfoundry_execution_intents_total") {
		t.Error("expected execution_intents_total in metrics output")
	}
	if !strings.Contains(body, "marketfoundry_execution_gate_active") {
		t.Error("expected execution_gate_active in metrics output")
	}
}

func TestStatusWriter_DefaultsTo200(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler writes body without calling WriteHeader — default is 200.
		_, _ = w.Write([]byte("ok"))
	})

	wrapped := InstrumentHTTPHandler("GET", "/default", inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/default", nil)

	wrapped(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
