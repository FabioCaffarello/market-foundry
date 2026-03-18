package healthz

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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
	Name         string `json:"name"`
	LastEventAt  string `json:"last_event_at,omitempty"`
	EventCount   int64  `json:"event_count"`
	IdleSeconds  int    `json:"idle_seconds,omitempty"`
	IdleWarning  bool   `json:"idle_warning,omitempty"`
}

// statusResponse is returned by /statusz.
type statusResponse struct {
	Status   string          `json:"status"`
	Trackers []trackerStatus `json:"trackers,omitempty"`
}

// HealthServer provides /healthz, /readyz, and /statusz endpoints.
type HealthServer struct {
	addr            string
	checks          []ReadinessCheck
	trackers        []*Tracker
	idleThreshold   time.Duration
	server          *http.Server
	logger          *slog.Logger
	stopHeartbeat   context.CancelFunc
	heartbeatWg     sync.WaitGroup
}

// Option configures the HealthServer.
type Option func(*HealthServer)

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
	resp := statusResponse{Status: "ok"}
	for _, t := range s.trackers {
		ts := trackerStatus{
			Name:       t.Name(),
			EventCount: t.EventCount(),
		}
		if last := t.LastEventAt(); !last.IsZero() {
			ts.LastEventAt = last.Format(time.RFC3339)
			idle := t.IdleSince()
			ts.IdleSeconds = int(idle.Seconds())
			if idle > s.idleThreshold {
				ts.IdleWarning = true
			}
		}
		resp.Trackers = append(resp.Trackers, ts)
	}
	writeJSON(w, http.StatusOK, resp)
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
				if count == 0 {
					continue // not yet active
				}
				idle := t.IdleSince()
				if idle > s.idleThreshold {
					s.logger.Warn("component idle",
						"tracker", t.Name(),
						"idle_seconds", int(idle.Seconds()),
						"last_event", t.LastEventAt().Format(time.RFC3339),
						"event_count", count,
					)
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
