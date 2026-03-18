package healthz_test

import (
	"context"
	"encoding/json"
	"fmt"
	"internal/shared/healthz"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTracker_RecordEvent(t *testing.T) {
	tr := healthz.NewTracker("test")

	if tr.EventCount() != 0 {
		t.Fatalf("expected 0, got %d", tr.EventCount())
	}
	if !tr.LastEventAt().IsZero() {
		t.Fatal("expected zero time before any event")
	}

	before := time.Now()
	tr.RecordEvent()
	after := time.Now()

	if tr.EventCount() != 1 {
		t.Fatalf("expected 1, got %d", tr.EventCount())
	}
	last := tr.LastEventAt()
	if last.Before(before) || last.After(after) {
		t.Fatalf("last event time %v not between %v and %v", last, before, after)
	}

	tr.RecordEvent()
	tr.RecordEvent()
	if tr.EventCount() != 3 {
		t.Fatalf("expected 3, got %d", tr.EventCount())
	}
}

func TestTracker_IdleSince(t *testing.T) {
	tr := healthz.NewTracker("test")

	// No events yet — idle is zero.
	if tr.IdleSince() != 0 {
		t.Fatal("expected zero idle before any event")
	}

	tr.RecordEvent()
	time.Sleep(10 * time.Millisecond)
	idle := tr.IdleSince()
	if idle < 10*time.Millisecond {
		t.Fatalf("idle %v is too short", idle)
	}
}

func TestHealthServer_Healthz(t *testing.T) {
	srv := healthz.NewHealthServer(":0", nil, nil)
	handler := testHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	assertJSONField(t, rec, "status", "ok")
}

func TestHealthServer_Readyz_AllPass(t *testing.T) {
	checks := []healthz.ReadinessCheck{
		{Name: "nats", Check: func(context.Context) error { return nil }},
	}
	srv := healthz.NewHealthServer(":0", checks, nil)
	handler := testHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	assertJSONField(t, rec, "status", "ready")
}

func TestHealthServer_Readyz_CheckFails(t *testing.T) {
	checks := []healthz.ReadinessCheck{
		{Name: "nats", Check: func(context.Context) error { return fmt.Errorf("connection refused") }},
	}
	srv := healthz.NewHealthServer(":0", checks, nil)
	handler := testHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "not_ready" {
		t.Fatalf("expected not_ready, got %q", resp["status"])
	}
	if resp["check"] != "nats" {
		t.Fatalf("expected nats, got %q", resp["check"])
	}
}

func TestHealthServer_Statusz(t *testing.T) {
	tr := healthz.NewTracker("projection")
	tr.RecordEvent()
	tr.RecordEvent()

	srv := healthz.NewHealthServer(":0", nil, []*healthz.Tracker{tr})
	handler := testHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/statusz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	trackers, ok := resp["trackers"].([]any)
	if !ok || len(trackers) != 1 {
		t.Fatalf("expected 1 tracker, got %v", resp["trackers"])
	}
	first := trackers[0].(map[string]any)
	if first["name"] != "projection" {
		t.Fatalf("expected projection, got %v", first["name"])
	}
	if first["event_count"] != float64(2) {
		t.Fatalf("expected 2, got %v", first["event_count"])
	}
}

func TestHealthServer_Statusz_IdleWarning(t *testing.T) {
	tr := healthz.NewTracker("consumer")
	tr.RecordEvent()

	// Use a very short idle threshold so the warning triggers immediately.
	srv := healthz.NewHealthServer(":0", nil, []*healthz.Tracker{tr},
		healthz.WithIdleThreshold(1*time.Nanosecond))
	handler := testHandler(srv)

	time.Sleep(time.Millisecond)
	req := httptest.NewRequest(http.MethodGet, "/statusz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	trackers := resp["trackers"].([]any)
	first := trackers[0].(map[string]any)
	if first["idle_warning"] != true {
		t.Fatalf("expected idle_warning true, got %v", first["idle_warning"])
	}
}

// testHandler builds an http.Handler from the HealthServer for testing
// without starting a real listener.
func testHandler(srv *healthz.HealthServer) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", srv.HandleHealthz)
	mux.HandleFunc("GET /readyz", srv.HandleReadyz)
	mux.HandleFunc("GET /statusz", srv.HandleStatusz)
	return mux
}

func assertJSONField(t *testing.T, rec *httptest.ResponseRecorder, key, want string) {
	t.Helper()
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if got := resp[key]; got != want {
		t.Fatalf("%s: expected %q, got %q", key, want, got)
	}
}
