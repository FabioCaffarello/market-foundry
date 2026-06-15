package delivery

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"internal/application/ports"
	deliverydomain "internal/domain/delivery"
	"internal/shared/metrics"

	"github.com/anthdm/hollywood/actor"
)

// Compile-time guarantees that the actor types satisfy the application
// ports the interfaces layer depends on (ADR-0005 / ADR-0028).
var (
	_ ports.DeliveryHub     = (*Hub)(nil)
	_ ports.DeliverySession = (*SessionHandle)(nil)
)

// Hub is the delivery entry point the gateway HTTP layer holds. It
// admits WebSocket connections by spawning a SessionActor and
// registering it with the router. The router and the NATS consumer are
// wired by Start; the Hub only needs the engine and the router PID.
type Hub struct {
	engine *actor.Engine
	router *actor.PID
	cfg    Config
	logger *slog.Logger
	seq    atomic.Uint64
	active atomic.Int64 // currently-admitted sessions (for the max-sessions cap)
}

// NewHub builds a Hub over an existing engine and a spawned router PID.
// cfg governs the per-session bounded buffer (size + backpressure policy).
func NewHub(engine *actor.Engine, router *actor.PID, cfg Config, logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{engine: engine, router: router, cfg: cfg, logger: logger}
}

// Admit spawns a session actor for a newly-connected client and
// registers it with the router. The returned handle drives the session
// for the connection's lifetime. Returns nil when the hub is at its
// configured max-sessions cap (ADR-0028 I4) — the caller must close the
// connection. Satisfies ports.DeliveryHub.
func (h *Hub) Admit(conn ports.DeliveryConn) ports.DeliverySession {
	// Optimistic increment-then-check: bounds total concurrent sessions
	// without a lock. On reject we roll the counter back.
	if n := h.active.Add(1); h.cfg.MaxSessions > 0 && n > int64(h.cfg.MaxSessions) {
		h.active.Add(-1)
		metrics.IncDeliverySessionRejected()
		h.logger.Warn("delivery session rejected: at capacity", "max", h.cfg.MaxSessions)
		return nil
	}

	id := deliverydomain.SessionID(fmt.Sprintf("delivery-session-%d", h.seq.Add(1)))
	pid := h.engine.Spawn(
		NewSessionActor(sessionConfig{id: id, conn: conn, maxQueue: h.cfg.QueueSize, policy: h.cfg.Policy, logger: h.logger}),
		string(id),
	)
	h.engine.Send(h.router, registerSessionMessage{ID: id, PID: pid})
	return &SessionHandle{engine: h.engine, router: h.router, pid: pid, id: id, release: func() { h.active.Add(-1) }}
}

// SessionHandle is the gateway HTTP handler's grip on one delivery
// session. Subscribe/Unsubscribe map client control frames to actor
// messages; Close tears the session down when the WebSocket closes.
type SessionHandle struct {
	engine  *actor.Engine
	router  *actor.PID
	pid     *actor.PID
	id      deliverydomain.SessionID
	release func() // frees the hub's session slot; called once on Close
	once    sync.Once
}

// Subscribe asks the session to add a subscription pattern. Validation
// happens in the session actor (domain delivery.NewSubscription).
func (s *SessionHandle) Subscribe(pattern string) {
	s.engine.Send(s.pid, subscribeMessage{Pattern: pattern})
}

// Unsubscribe asks the session to drop a subscription pattern.
func (s *SessionHandle) Unsubscribe(pattern string) {
	s.engine.Send(s.pid, unsubscribeMessage{Pattern: pattern})
}

// Close unregisters the session from the router, poisons the session
// actor (which closes the connection on stop), and frees the hub's
// session slot. Idempotent — safe to call more than once.
func (s *SessionHandle) Close() {
	s.once.Do(func() {
		s.engine.Send(s.router, unregisterSessionMessage{ID: s.id})
		s.engine.Poison(s.pid)
		if s.release != nil {
			s.release()
		}
	})
}
