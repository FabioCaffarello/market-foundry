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
		Help: "Total per-stream-key sequence gaps observed at consumer ingress. " +
			"Per ADR-0020, consumers track the highest seq seen per stream key; " +
			"on receiving an event with seq > last_observed + 1, this counter is " +
			"incremented and the consumer continues forward (gap-filling is " +
			"recovery, not a blocking error). Rising values indicate either " +
			"producer-side restart-redundancy or genuine event loss; operators " +
			"correlate with sequencer state KV restores to disambiguate.",
	},
	[]string{"stream_key"},
)

func init() {
	prometheus.MustRegister(consumerSeqGapTotal)
}

// IncSeqGap increments the per-stream-key gap counter. The
// streamKey label is a caller-formatted string (typically
// "{venue}.{instrument}.{event_type}" matching ADR-0020 key
// taxonomy). Callers are responsible for keeping cardinality
// bounded — one label value per active stream key.
func IncSeqGap(streamKey string) {
	consumerSeqGapTotal.WithLabelValues(streamKey).Inc()
}

// SeqGapCount returns the current counter value for the given
// stream_key label. Exported so cross-module tests can verify
// counter behavior without taking a direct prometheus
// dependency.
func SeqGapCount(streamKey string) float64 {
	return testutil.ToFloat64(consumerSeqGapTotal.WithLabelValues(streamKey))
}
