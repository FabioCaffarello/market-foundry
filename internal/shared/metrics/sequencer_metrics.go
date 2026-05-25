package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// ── Sequencer / consumer gap metrics ─────────────────────────────────

var consumerSeqGapTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "consumer",
		Name:      "seq_gap_total",
		Help: "Total per-(venue, event_type) sequence gaps observed at " +
			"consumer ingress. Per ADR-0020, consumers track the highest " +
			"seq seen per (venue, instrument, event_type) stream key; on " +
			"receiving an event with seq > last_observed + 1, this counter " +
			"is incremented and the consumer continues forward " +
			"(gap-filling is recovery, not a blocking error). Rising " +
			"values indicate either producer-side restart-redundancy or " +
			"genuine event loss; operators correlate with sequencer state " +
			"KV restores to disambiguate. Label cardinality is bounded " +
			"by (venue × event_type) per ADR-0024 MP-2; the high-" +
			"cardinality instrument dimension is compensated via " +
			"structured log per ADR-0024 MP-5.",
	},
	[]string{"venue", "event_type"},
)

func init() {
	prometheus.MustRegister(consumerSeqGapTotal)
}

// IncSeqGap increments the per-(venue, event_type) gap counter.
//
// Callers MUST emit a structured log record alongside the
// increment carrying the high-cardinality instrument dimension
// omitted from the label set, per ADR-0024 MP-5 "log
// compensation pattern". Reference shape:
//
//	slog.Warn("sequencer.gap_detected",
//	    "venue", venue,
//	    "instrument", instrument, // omitted from metric label
//	    "event_type", eventType,
//	    "last_seq", lastSeq,
//	    "current_seq", currentSeq,
//	    "gap_size", currentSeq-lastSeq-1)
//	metrics.IncSeqGap(venue, eventType)
//
// Operators correlating a high counter rate to a specific
// instrument do so via `docker logs <binary> | grep
// sequencer.gap_detected | grep <venue>` until log aggregation
// lands (PROGRAM-0003 non-goal).
func IncSeqGap(venue, eventType string) {
	consumerSeqGapTotal.WithLabelValues(venue, eventType).Inc()
}

// SeqGapCount returns the current counter value for the given
// (venue, event_type) labels. Exported so cross-module tests
// can verify counter behavior without taking a direct prometheus
// dependency.
func SeqGapCount(venue, eventType string) float64 {
	return testutil.ToFloat64(consumerSeqGapTotal.WithLabelValues(venue, eventType))
}
