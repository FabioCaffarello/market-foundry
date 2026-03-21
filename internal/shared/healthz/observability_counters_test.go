package healthz_test

import (
	"encoding/json"
	"internal/shared/healthz"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTracker_DomainCounterSemantics verifies that the domain-aware counter
// naming convention used by publisher actors produces correct /statusz output.
// S292: interleaved observability minimum for squeeze breakout slice.
func TestTracker_DomainCounterSemantics(t *testing.T) {
	tracker := healthz.NewTracker("derive-publisher")

	// Simulate squeeze breakout slice counters as emitted by publisher actors.
	tracker.Counter("signal:bollinger").Add(5)
	tracker.Counter("decision:bollinger_squeeze:triggered").Add(2)
	tracker.Counter("decision:bollinger_squeeze:not_triggered").Add(3)
	tracker.Counter("strategy:squeeze_breakout_entry:long").Add(2)
	tracker.Counter("strategy:squeeze_breakout_entry:flat").Add(3)
	tracker.Counter("risk:position_exposure:approved").Add(1)
	tracker.Counter("risk:drawdown_limit:modified").Add(1)
	tracker.Counter("risk:position_exposure:rejected").Add(0) // counter exists, value 0
	tracker.Counter("execution:paper_order:buy").Add(1)
	tracker.Counter("execution:paper_order:filled").Add(1)
	tracker.Counter("execution:gate_halted").Add(0)
	tracker.Counter("published:BTCUSDT").Add(5)

	// Verify counter snapshot contains all expected keys.
	counters := tracker.Counters()

	expected := map[string]int64{
		"signal:bollinger":                       5,
		"decision:bollinger_squeeze:triggered":    2,
		"decision:bollinger_squeeze:not_triggered": 3,
		"strategy:squeeze_breakout_entry:long":    2,
		"strategy:squeeze_breakout_entry:flat":    3,
		"risk:position_exposure:approved":         1,
		"risk:drawdown_limit:modified":            1,
		"risk:position_exposure:rejected":         0,
		"execution:paper_order:buy":               1,
		"execution:paper_order:filled":            1,
		"execution:gate_halted":                   0,
		"published:BTCUSDT":                       5,
	}

	for key, want := range expected {
		got, ok := counters[key]
		if !ok {
			t.Errorf("missing counter %q", key)
			continue
		}
		if got != want {
			t.Errorf("counter %q = %d, want %d", key, got, want)
		}
	}

	if len(counters) != len(expected) {
		t.Errorf("counter count = %d, want %d", len(counters), len(expected))
	}
}

// TestTracker_CountersVisibleInStatusz verifies that domain counters appear
// in the /statusz JSON response, making them queryable by operators.
func TestTracker_CountersVisibleInStatusz(t *testing.T) {
	tracker := healthz.NewTracker("derive-publisher")
	tracker.RecordEvent()
	tracker.Counter("signal:bollinger").Add(3)
	tracker.Counter("decision:bollinger_squeeze:triggered").Add(1)
	tracker.Counter("execution:paper_order:buy").Add(1)
	tracker.Counter("execution:paper_order:filled").Add(1)

	hs := healthz.NewHealthServer(":0", nil, []*healthz.Tracker{tracker})

	req := httptest.NewRequest(http.MethodGet, "/statusz", nil)
	w := httptest.NewRecorder()
	hs.HandleStatusz(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("statusz status = %d, want 200", w.Code)
	}

	var resp struct {
		Trackers []struct {
			Name     string           `json:"name"`
			Counters map[string]int64 `json:"counters"`
		} `json:"trackers"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode statusz: %v", err)
	}

	if len(resp.Trackers) != 1 {
		t.Fatalf("tracker count = %d, want 1", len(resp.Trackers))
	}

	tr := resp.Trackers[0]
	if tr.Name != "derive-publisher" {
		t.Errorf("tracker name = %q, want %q", tr.Name, "derive-publisher")
	}

	wantCounters := []string{
		"signal:bollinger",
		"decision:bollinger_squeeze:triggered",
		"execution:paper_order:buy",
		"execution:paper_order:filled",
	}
	for _, key := range wantCounters {
		if _, ok := tr.Counters[key]; !ok {
			t.Errorf("counter %q missing from statusz response", key)
		}
	}
}
