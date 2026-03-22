package healthz

import (
	"context"
	"encoding/json"
	"errors"
	"internal/shared/metrics"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Tracker records event activity for a named component.
// It is safe for concurrent use from multiple goroutines.
type Tracker struct {
	name        string
	lastEventAt atomic.Int64 // unix nanoseconds
	eventCount  atomic.Int64
	errorCount  atomic.Int64

	mu      sync.Mutex
	counters map[string]*atomic.Int64
}

// NewTracker creates a tracker for the given component name.
func NewTracker(name string) *Tracker {
	return &Tracker{name: name}
}

// RecordEvent marks that an event was processed right now.
func (t *Tracker) RecordEvent() {
	t.lastEventAt.Store(time.Now().UnixNano())
	t.eventCount.Add(1)
}

// RecordError marks that an error occurred during event processing.
// This keeps the tracker alive (updates lastEventAt) while separately
// counting errors for operational visibility.
func (t *Tracker) RecordError() {
	t.lastEventAt.Store(time.Now().UnixNano())
	t.errorCount.Add(1)
}

// ErrorCount returns how many errors have been recorded.
func (t *Tracker) ErrorCount() int64 {
	return t.errorCount.Load()
}

// Counter returns a named atomic counter, creating it on first access.
// Use this to expose domain-specific stats (e.g., "filled", "skipped_stale")
// that will appear in /statusz alongside standard event/error counts.
func (t *Tracker) Counter(name string) *atomic.Int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.counters == nil {
		t.counters = make(map[string]*atomic.Int64)
	}
	c, ok := t.counters[name]
	if !ok {
		c = &atomic.Int64{}
		t.counters[name] = c
	}
	return c
}

// Counters returns a snapshot of all custom counters.
func (t *Tracker) Counters() map[string]int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.counters) == 0 {
		return nil
	}
	snap := make(map[string]int64, len(t.counters))
	for k, v := range t.counters {
		snap[k] = v.Load()
	}
	return snap
}

// Name returns the tracker's component name.
func (t *Tracker) Name() string { return t.name }

// LastEventAt returns when the last event was recorded, or zero if none.
func (t *Tracker) LastEventAt() time.Time {
	nanos := t.lastEventAt.Load()
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}

// EventCount returns how many events have been recorded.
func (t *Tracker) EventCount() int64 {
	return t.eventCount.Load()
}

// IdleSince returns how long since the last event.
// Returns zero if no event has been recorded yet.
func (t *Tracker) IdleSince() time.Duration {
	last := t.LastEventAt()
	if last.IsZero() {
		return 0
	}
	return time.Since(last)
}

// ReadinessCheck is a named check that returns nil on success.
type ReadinessCheck struct {
	Name  string
	Check func(ctx context.Context) error
}

// trackerStatus is the JSON representation of a single tracker.
type trackerStatus struct {
	Name        string           `json:"name"`
	LastEventAt string           `json:"last_event_at,omitempty"`
	EventCount  int64            `json:"event_count"`
	ErrorCount  int64            `json:"error_count,omitempty"`
	IdleSeconds int              `json:"idle_seconds,omitempty"`
	IdleWarning bool             `json:"idle_warning,omitempty"`
	Counters    map[string]int64 `json:"counters,omitempty"`
}

// statusResponse is returned by /statusz.
type statusResponse struct {
	Status          string          `json:"status"`
	Phase           string          `json:"phase"`
	Runtime         string          `json:"runtime,omitempty"`
	Uptime          string          `json:"uptime,omitempty"`
	StartedAt       string          `json:"started_at,omitempty"`
	DegradedTrackers []string       `json:"degraded_trackers,omitempty"`
	Trackers        []trackerStatus `json:"trackers,omitempty"`
}

// HealthServer provides /healthz, /readyz, /statusz, and /diagz endpoints.
type HealthServer struct {
	addr          string
	runtime       string
	checks        []ReadinessCheck
	trackers      []*Tracker
	idleThreshold time.Duration
	server        *http.Server
	logger        *slog.Logger
	startedAt     time.Time
	stopHeartbeat context.CancelFunc
	heartbeatWg   sync.WaitGroup
}

// Option configures the HealthServer.
type Option func(*HealthServer)

// WithRuntime sets the runtime name for diagnostic endpoints.
func WithRuntime(name string) Option {
	return func(s *HealthServer) { s.runtime = name }
}

// WithIdleThreshold sets the duration after which idle trackers emit a warning log.
// Default: 2 minutes.
func WithIdleThreshold(d time.Duration) Option {
	return func(s *HealthServer) { s.idleThreshold = d }
}

// NewHealthServer creates a health server listening on addr.
func NewHealthServer(addr string, checks []ReadinessCheck, trackers []*Tracker, opts ...Option) *HealthServer {
	s := &HealthServer{
		addr:          addr,
		checks:        checks,
		trackers:      trackers,
		idleThreshold: 2 * time.Minute,
		logger:        slog.Default().With("component", "healthz"),
		startedAt:     time.Now(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start starts the health HTTP server and the idle heartbeat monitor.
// It blocks until the server is stopped; call in a goroutine.
func (s *HealthServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.HandleHealthz)
	mux.HandleFunc("GET /readyz", s.HandleReadyz)
	mux.HandleFunc("GET /statusz", s.HandleStatusz)
	mux.HandleFunc("GET /diagz", s.HandleDiagz)
	mux.Handle("GET /metrics", metrics.Handler())

	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Start idle heartbeat monitor.
	ctx, cancel := context.WithCancel(context.Background())
	s.stopHeartbeat = cancel
	s.heartbeatWg.Add(1)
	go s.heartbeatLoop(ctx)

	s.logger.Info("health server starting", "addr", s.addr)
	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// StartInBackground starts the health server in a goroutine and logs any
// startup error. This is the canonical way to launch the health server
// alongside an actor engine.
func (s *HealthServer) StartInBackground() {
	go func() {
		if err := s.Start(); err != nil {
			s.logger.Error("health server failed", "error", err)
		}
	}()
}

// GracefulShutdown stops the health server with the given timeout.
// This is the canonical shutdown companion to StartInBackground.
func (s *HealthServer) GracefulShutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.Shutdown(ctx)
}

// Shutdown gracefully stops the health server.
func (s *HealthServer) Shutdown(ctx context.Context) error {
	if s.stopHeartbeat != nil {
		s.stopHeartbeat()
		s.heartbeatWg.Wait()
	}
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// HandleHealthz serves the /healthz liveness probe.
func (s *HealthServer) HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleReadyz serves the /readyz readiness probe.
func (s *HealthServer) HandleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	for _, c := range s.checks {
		if err := c.Check(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "not_ready",
				"check":  c.Name,
				"error":  err.Error(),
			})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// HandleStatusz serves the /statusz activity status endpoint.
func (s *HealthServer) HandleStatusz(w http.ResponseWriter, _ *http.Request) {
	resp := statusResponse{
		Status:    "ok",
		Runtime:   s.runtime,
		Uptime:    time.Since(s.startedAt).Truncate(time.Second).String(),
		StartedAt: s.startedAt.Format(time.RFC3339),
	}
	for _, t := range s.trackers {
		ts := trackerStatus{
			Name:       t.Name(),
			EventCount: t.EventCount(),
			ErrorCount: t.ErrorCount(),
		}
		if last := t.LastEventAt(); !last.IsZero() {
			ts.LastEventAt = last.Format(time.RFC3339)
			idle := t.IdleSince()
			ts.IdleSeconds = int(idle.Seconds())
			if idle > s.idleThreshold {
				ts.IdleWarning = true
			}
		}
		ts.Counters = t.Counters()
		resp.Trackers = append(resp.Trackers, ts)
		if counters := t.Counters(); len(counters) > 0 {
			if counters["pipeline_degraded"] > 0 {
				resp.DegradedTrackers = append(resp.DegradedTrackers, t.Name())
			}
		}
	}
	resp.Phase = s.computePhase()
	writeJSON(w, http.StatusOK, resp)
}

// computePhase derives the aggregate operational phase from tracker state.
//
//   - "starting"  — uptime < 30s and no tracker has recorded an event
//   - "warming"   — at least one tracker awaiting its first event
//   - "active"    — all trackers receiving events, none idle
//   - "idle"      — at least one tracker exceeds idle threshold
//   - "stalled"   — all active trackers exceed idle threshold
//   - "degraded"  — at least one tracker has pipeline_degraded > 0
//
// When no trackers are registered the phase follows uptime alone:
// "starting" for the first 30s, then "active".
func (s *HealthServer) computePhase() string {
	uptime := time.Since(s.startedAt)

	if len(s.trackers) == 0 {
		if uptime < 30*time.Second {
			return "starting"
		}
		return "active"
	}

	awaiting := 0
	idleCount := 0
	active := 0
	degraded := 0

	for _, t := range s.trackers {
		if counters := t.Counters(); len(counters) > 0 {
			if counters["pipeline_degraded"] > 0 {
				degraded++
			}
		}
		if t.EventCount() == 0 && t.ErrorCount() == 0 {
			awaiting++
			continue
		}
		if t.IdleSince() > s.idleThreshold {
			idleCount++
		} else {
			active++
		}
	}

	total := len(s.trackers)

	if degraded > 0 {
		return "degraded"
	}
	if awaiting == total && uptime < 30*time.Second {
		return "starting"
	}
	if awaiting > 0 {
		return "warming"
	}
	if idleCount == total {
		return "stalled"
	}
	if idleCount > 0 {
		return "idle"
	}
	return "active"
}

// HandleDiagz serves the /diagz diagnostic summary endpoint.
// It provides a machine-readable overview of the runtime's diagnostic state
// without exposing operational data or domain secrets.
func (s *HealthServer) HandleDiagz(w http.ResponseWriter, _ *http.Request) {
	diag := map[string]any{
		"runtime":        s.runtime,
		"phase":          s.computePhase(),
		"started_at":     s.startedAt.Format(time.RFC3339),
		"uptime":         time.Since(s.startedAt).Truncate(time.Second).String(),
		"go_version":     runtime.Version(),
		"num_goroutines": runtime.NumGoroutine(),
	}

	// Readiness check summary.
	checks := make([]map[string]any, 0, len(s.checks))
	for _, c := range s.checks {
		entry := map[string]any{"name": c.Name}
		if err := c.Check(context.Background()); err != nil {
			entry["status"] = "fail"
			entry["error"] = err.Error()
		} else {
			entry["status"] = "pass"
		}
		checks = append(checks, entry)
	}
	diag["readiness_checks"] = checks

	// Tracker summary.
	trackerSummary := make([]map[string]any, 0, len(s.trackers))
	for _, t := range s.trackers {
		entry := map[string]any{
			"name":        t.Name(),
			"event_count": t.EventCount(),
			"error_count": t.ErrorCount(),
		}
		if last := t.LastEventAt(); !last.IsZero() {
			entry["last_event_at"] = last.Format(time.RFC3339)
			entry["idle_seconds"] = int(t.IdleSince().Seconds())
		} else {
			entry["status"] = "awaiting_first_event"
		}
		if counters := t.Counters(); len(counters) > 0 {
			entry["counters"] = counters
		}
		trackerSummary = append(trackerSummary, entry)
	}
	diag["trackers"] = trackerSummary

	writeJSON(w, http.StatusOK, diag)
}

func (s *HealthServer) heartbeatLoop(ctx context.Context) {
	defer s.heartbeatWg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, t := range s.trackers {
				count := t.EventCount()
				errCount := t.ErrorCount()
				if count == 0 && errCount == 0 {
					continue // not yet active
				}
				idle := t.IdleSince()
				if idle > s.idleThreshold {
					attrs := []any{
						"tracker", t.Name(),
						"idle_seconds", int(idle.Seconds()),
						"last_event", t.LastEventAt().Format(time.RFC3339),
						"event_count", count,
						"error_count", errCount,
					}
					if counters := t.Counters(); len(counters) > 0 {
						for k, v := range counters {
							attrs = append(attrs, k, v)
						}
					}
					s.logger.Warn("component idle", attrs...)
				}
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
