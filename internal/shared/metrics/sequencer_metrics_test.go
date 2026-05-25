package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIncSeqGap_IncrementsCounter(t *testing.T) {
	const key = "binance.btcusdt.observation.trade"
	before := SeqGapCount(key)
	IncSeqGap(key)
	after := SeqGapCount(key)
	if delta := after - before; delta != 1 {
		t.Fatalf("expected counter delta 1; got %v (before=%v after=%v)", delta, before, after)
	}
}

func TestIncSeqGap_LabelsAreIndependent(t *testing.T) {
	const k1 = "binance.btcusdt.observation.trade"
	const k2 = "binance.ethusdt.observation.trade"
	beforeK1 := SeqGapCount(k1)
	beforeK2 := SeqGapCount(k2)

	IncSeqGap(k1)
	IncSeqGap(k1)
	IncSeqGap(k2)

	if delta := SeqGapCount(k1) - beforeK1; delta != 2 {
		t.Errorf("k1 delta = %v, want 2", delta)
	}
	if delta := SeqGapCount(k2) - beforeK2; delta != 1 {
		t.Errorf("k2 delta = %v, want 1", delta)
	}
}

func TestSeqGapTotal_ExposedOnMetricsEndpoint(t *testing.T) {
	IncSeqGap("binance.btcusdt.observation.trade")

	handler := HandlerFunc()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler(rec, req)
	body, _ := io.ReadAll(rec.Body)

	if !strings.Contains(string(body), "marketfoundry_consumer_seq_gap_total") {
		t.Fatal("expected marketfoundry_consumer_seq_gap_total in /metrics output")
	}
}
