// Package metrics defines Prometheus metric collectors for the market-foundry
// platform. All metrics use the "marketfoundry" namespace.
//
// Binaries expose these metrics via a /metrics endpoint — the gateway through
// its route table, other services through the shared HealthServer.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

const namespace = "marketfoundry"

// ── HTTP request metrics ────────────────────────────────────────────

var (
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds.",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status_code"},
	)

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status_code"},
	)
)

// ── Consumer metrics ────────────────────────────────────────────────

var (
	consumerMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "consumer",
			Name:      "messages_total",
			Help:      "Total consumer messages by outcome.",
		},
		[]string{"consumer", "outcome"},
	)

	consumerProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "consumer",
			Name:      "processing_duration_seconds",
			Help:      "Duration of consumer message processing in seconds.",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"consumer"},
	)

	consumerLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "consumer",
			Name:      "lag_messages",
			Help:      "Pending messages not yet delivered to the consumer.",
		},
		[]string{"consumer"},
	)
)

// ── Execution metrics ─────────────────────────────────────────────

var (
	executionStrategyEvaluationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "execution",
			Name:      "strategy_evaluations_total",
			Help:      "Total strategy evaluations by strategy type and outcome.",
		},
		[]string{"strategy_type", "outcome"},
	)

	executionGateChecksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "execution",
			Name:      "gate_checks_total",
			Help:      "Total pre-submit gate checks by gate and verdict.",
		},
		[]string{"gate", "verdict"},
	)

	executionIntentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "execution",
			Name:      "intents_total",
			Help:      "Total execution intents produced by source path and side.",
		},
		[]string{"source_path", "side"},
	)

	executionGateStatus = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "execution",
			Name:      "gate_active",
			Help:      "Whether the execution gate is active (1) or halted (0).",
		},
	)

	executionGateReadFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "execution",
			Name:      "gate_read_failures_total",
			Help: "Total ControlGate IsHalted read failures by reason. " +
				"IsHalted returns false on these (fail-open per ADR 0012); " +
				"this counter surfaces the failure mode for monitoring.",
		},
		[]string{"reason"},
	)
)

// ── Adapter metrics (ADR-0022 R3, H-7.a) ──────────────────────────

var (
	adapterUndeclaredEventTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "adapter",
			Name:      "undeclared_event_total",
			Help: "Venue-native events received for an (event_type, contract) " +
				"pair NOT declared in the adapter's Capabilities(). The event is " +
				"silently rejected at the producer (no NATS publish — ADR-0022 R3); " +
				"a non-zero rate means Capabilities() is out of date with adapter " +
				"parsing reality: fix the declaration or fix the parser.",
		},
		[]string{"venue", "event_type", "contract"},
	)
)

// ── Delivery metrics (ADR-0028 I4, H-11.c) ────────────────────────

var (
	deliveryFramesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "delivery",
			Name:      "frames_total",
			Help: "Insights frames handled by the WebSocket delivery fan-out, " +
				"by outcome: 'delivered' (written to a client) or 'dropped' " +
				"(shed by a session's bounded buffer per its backpressure " +
				"policy — ADR-0028 I4). A non-trivial dropped rate means clients " +
				"are slower than the event stream.",
		},
		[]string{"outcome"},
	)

	deliverySessions = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "delivery",
			Name:      "sessions",
			Help:      "Currently connected WebSocket delivery sessions (clients).",
		},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestDuration,
		httpRequestsTotal,
		consumerMessagesTotal,
		consumerProcessingDuration,
		consumerLag,
		executionStrategyEvaluationsTotal,
		executionGateChecksTotal,
		executionIntentsTotal,
		executionGateStatus,
		executionGateReadFailuresTotal,
		adapterUndeclaredEventTotal,
		deliveryFramesTotal,
		deliverySessions,
	)
}

// ── Delivery helpers (ADR-0028 I4) ──────────────────────────────────

// Canonical delivery frame outcomes.
const (
	DeliveryOutcomeDelivered = "delivered"
	DeliveryOutcomeDropped   = "dropped"
)

// IncDeliveryFrame counts a delivery frame by outcome
// (DeliveryOutcomeDelivered / DeliveryOutcomeDropped).
func IncDeliveryFrame(outcome string) {
	deliveryFramesTotal.WithLabelValues(outcome).Inc()
}

// IncDeliverySessions / DecDeliverySessions track the connected-client gauge.
func IncDeliverySessions() { deliverySessions.Inc() }
func DecDeliverySessions() { deliverySessions.Dec() }

// Handler returns the Prometheus metrics HTTP handler.
func Handler() http.Handler {
	return promhttp.Handler()
}

// HandlerFunc returns the Prometheus metrics handler as an http.HandlerFunc.
func HandlerFunc() http.HandlerFunc {
	return promhttp.Handler().ServeHTTP
}

// ── HTTP helpers ────────────────────────────────────────────────────

// ObserveHTTPRequest records an HTTP request duration and increments the counter.
func ObserveHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	httpRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
	httpRequestsTotal.WithLabelValues(method, path, status).Inc()
}

// InstrumentHTTPHandler wraps an http.HandlerFunc with request duration and
// counter instrumentation. The method and path are used as label values so
// that URL parameter cardinality is bounded to route patterns.
func InstrumentHTTPHandler(method, path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(sw, r)
		ObserveHTTPRequest(method, path, sw.code, time.Since(start))
	}
}

// statusWriter captures the HTTP status code written by the handler.
type statusWriter struct {
	http.ResponseWriter
	code    int
	written bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if !sw.written {
		sw.code = code
		sw.written = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

// ── Consumer helpers ────────────────────────────────────────────────

// IncConsumerMessage increments the consumer message counter for the given outcome.
// Canonical outcomes: "delivered", "redelivered", "terminated", "nakked".
func IncConsumerMessage(consumer, outcome string) {
	consumerMessagesTotal.WithLabelValues(consumer, outcome).Inc()
}

// ObserveConsumerProcessing records a consumer message processing duration.
func ObserveConsumerProcessing(consumer string, duration time.Duration) {
	consumerProcessingDuration.WithLabelValues(consumer).Observe(duration.Seconds())
}

// SetConsumerLag sets the consumer lag gauge to the given pending message count.
func SetConsumerLag(consumer string, pending float64) {
	consumerLag.WithLabelValues(consumer).Set(pending)
}

// ── Execution helpers ─────────────────────────────────────────────

// IncStrategyEvaluation increments the strategy evaluation counter.
// Canonical outcomes: "actionable", "flat", "skipped_wrong_type",
// "skipped_low_confidence", "error".
func IncStrategyEvaluation(strategyType, outcome string) {
	executionStrategyEvaluationsTotal.WithLabelValues(strategyType, outcome).Inc()
}

// IncGateCheck increments the gate check counter.
// Canonical gates: "kill_switch", "staleness".
// Canonical verdicts: "allowed", "blocked".
func IncGateCheck(gate, verdict string) {
	executionGateChecksTotal.WithLabelValues(gate, verdict).Inc()
}

// GateReadFailureReason enumerates the canonical reasons for an
// IsHalted read failure. See ADR 0012.
const (
	GateReadFailureNilBucket   = "nil_bucket"
	GateReadFailureKeyNotFound = "key_not_found"
	GateReadFailureCtxTimeout  = "ctx_timeout"
	GateReadFailureKVError     = "kv_error"
	GateReadFailureUnmarshal   = "unmarshal_error"
)

// IncGateReadFailure increments the gate read failure counter. Each
// failure mode in ControlKVStore.IsHalted maps to one canonical reason
// label so operators can distinguish transient KV outages from missing
// keys, parse errors, and uninitialized buckets.
func IncGateReadFailure(reason string) {
	executionGateReadFailuresTotal.WithLabelValues(reason).Inc()
}

// GateReadFailureCount returns the current counter value for the given
// reason label. Exported so cross-module tests can verify counter
// behavior without taking a direct prometheus dependency.
func GateReadFailureCount(reason string) float64 {
	return testutil.ToFloat64(executionGateReadFailuresTotal.WithLabelValues(reason))
}

// IncExecutionIntent increments the execution intent counter.
func IncExecutionIntent(sourcePath, side string) {
	executionIntentsTotal.WithLabelValues(sourcePath, side).Inc()
}

// IncAdapterUndeclaredEvent increments the undeclared-event counter
// for an (event_type, contract) pair outside the adapter's declared
// Capabilities() (ADR-0022 R3). Callers reject the event after
// counting — never publish it.
func IncAdapterUndeclaredEvent(venue, eventType, contract string) {
	adapterUndeclaredEventTotal.WithLabelValues(venue, eventType, contract).Inc()
}

// AdapterUndeclaredEventCount returns the current counter value for
// the given label triple. Exported so cross-module tests can verify
// the R3 guard without taking a direct prometheus dependency.
func AdapterUndeclaredEventCount(venue, eventType, contract string) float64 {
	return testutil.ToFloat64(adapterUndeclaredEventTotal.WithLabelValues(venue, eventType, contract))
}

// SetGateActive sets the execution gate status gauge.
// 1.0 = active, 0.0 = halted.
func SetGateActive(active bool) {
	if active {
		executionGateStatus.Set(1)
	} else {
		executionGateStatus.Set(0)
	}
}
