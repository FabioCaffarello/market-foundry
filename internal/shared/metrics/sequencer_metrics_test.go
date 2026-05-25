package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIncSeqGap_IncrementsCounter(t *testing.T) {
	const venue = "binance"
	const eventType = "observation.trade"
	before := SeqGapCount(venue, eventType)
	IncSeqGap(venue, eventType)
	after := SeqGapCount(venue, eventType)
	if delta := after - before; delta != 1 {
		t.Fatalf("expected counter delta 1; got %v (before=%v after=%v)", delta, before, after)
	}
}

func TestIncSeqGap_LabelsAreIndependent(t *testing.T) {
	const venue1, eventType1 = "binance", "observation.trade"
	const venue2, eventType2 = "binancef", "observation.trade"
	const venue3, eventType3 = "binance", "observation.book.snapshot"

	before1 := SeqGapCount(venue1, eventType1)
	before2 := SeqGapCount(venue2, eventType2)
	before3 := SeqGapCount(venue3, eventType3)

	IncSeqGap(venue1, eventType1)
	IncSeqGap(venue1, eventType1)
	IncSeqGap(venue2, eventType2)

	if delta := SeqGapCount(venue1, eventType1) - before1; delta != 2 {
		t.Errorf("(venue1, eventType1) delta = %v, want 2", delta)
	}
	if delta := SeqGapCount(venue2, eventType2) - before2; delta != 1 {
		t.Errorf("(venue2, eventType2) delta = %v, want 1", delta)
	}
	if delta := SeqGapCount(venue3, eventType3) - before3; delta != 0 {
		t.Errorf("(venue3, eventType3) delta = %v, want 0 (no increments)", delta)
	}
}

func TestSeqGapTotal_ExposedOnMetricsEndpoint(t *testing.T) {
	IncSeqGap("binance", "observation.trade")

	handler := HandlerFunc()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler(rec, req)
	body, _ := io.ReadAll(rec.Body)

	if !strings.Contains(string(body), "marketfoundry_consumer_seq_gap_total") {
		t.Fatal("expected marketfoundry_consumer_seq_gap_total in /metrics output")
	}
	// Per ADR-0024 MP-2, labels are venue + event_type — not stream_key.
	if !strings.Contains(string(body), `venue="binance"`) {
		t.Errorf("expected venue label in /metrics output; got: %s", string(body))
	}
	if !strings.Contains(string(body), `event_type="observation.trade"`) {
		t.Errorf("expected event_type label in /metrics output; got: %s", string(body))
	}
	if strings.Contains(string(body), `stream_key=`) {
		t.Errorf("expected NO stream_key label after ADR-0024 refactor; got: %s", string(body))
	}
}
