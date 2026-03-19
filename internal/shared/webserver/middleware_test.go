package webserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"internal/shared/requestctx"
	"internal/shared/webserver"
)

func TestCorrelationID_InjectsHeader(t *testing.T) {
	var got string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = requestctx.CorrelationID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := webserver.CorrelationID(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Correlation-ID", "corr-abc-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got != "corr-abc-123" {
		t.Errorf("expected correlation ID %q, got %q", "corr-abc-123", got)
	}
}

func TestCorrelationID_NoHeaderNoContext(t *testing.T) {
	var got string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = requestctx.CorrelationID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := webserver.CorrelationID(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got != "" {
		t.Errorf("expected empty correlation ID, got %q", got)
	}
}
