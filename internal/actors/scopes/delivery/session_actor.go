package delivery

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	"internal/application/ports"
	deliverydomain "internal/domain/delivery"

	"github.com/anthdm/hollywood/actor"
)

// DefaultOutboundQueue bounds the per-session outbound buffer. Once full,
// the NEWEST frames are dropped (ADR-0028 I4, DropNewest) — the buffer
// never grows without bound, so a slow client can neither block the
// fan-out nor exhaust memory. H-11.c makes the policy and size
// configurable.
const DefaultOutboundQueue = 256

type sessionConfig struct {
	id       deliverydomain.SessionID
	conn     ports.DeliveryConn
	maxQueue int
	logger   *slog.Logger
}

// SessionActor owns one client's delivery state: its subscriptions
// (domain Session) and its connection. The actor mailbox serializes
// access, so the Session needs no locks (single-owner). A dedicated
// write goroutine drains the bounded outbound buffer to the connection,
// so a slow socket backpressures only this session.
type SessionActor struct {
	cfg     sessionConfig
	logger  *slog.Logger
	session *deliverydomain.Session
	out     chan []byte
	dropped uint64
}

// NewSessionActor returns a producer for a per-connection session actor.
func NewSessionActor(cfg sessionConfig) actor.Producer {
	return func() actor.Receiver {
		return &SessionActor{cfg: cfg}
	}
}

func (a *SessionActor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		a.start()
	case actor.Stopped:
		a.stop()
	case subscribeMessage:
		a.onSubscribe(msg)
	case unsubscribeMessage:
		a.onUnsubscribe(msg)
	case deliverFrameMessage:
		a.onDeliver(msg)
	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *SessionActor) start() {
	a.logger = a.cfg.logger
	if a.logger == nil {
		a.logger = slog.Default()
	}
	a.logger = a.logger.With("actor", "delivery-session", "session", string(a.cfg.id))
	a.session = deliverydomain.NewSession(a.cfg.id)
	q := a.cfg.maxQueue
	if q <= 0 {
		q = DefaultOutboundQueue
	}
	a.out = make(chan []byte, q)
	go a.writeLoop()
	a.logger.Info("delivery session started")
}

func (a *SessionActor) stop() {
	close(a.out) // signal the write loop to finish draining and exit
	a.logger.Info("delivery session stopped", "dropped", a.dropped)
}

// writeLoop is the single writer to the connection. It drains the
// bounded buffer until the actor stops (channel closed). On a write
// error the client is gone: it closes the connection (the HTTP read
// loop then observes the close and poisons the actor) and keeps
// draining so the actor's non-blocking offers never block.
func (a *SessionActor) writeLoop() {
	for frame := range a.out {
		if err := a.cfg.conn.Send(frame); err != nil {
			a.logger.Warn("write to client failed; closing connection", "error", err)
			_ = a.cfg.conn.Close()
			for range a.out { // drain until Stopped closes the channel
			}
			return
		}
	}
	_ = a.cfg.conn.Close()
}

func (a *SessionActor) onSubscribe(msg subscribeMessage) {
	sub, prob := deliverydomain.NewSubscription(msg.Pattern)
	if prob != nil {
		a.logger.Warn("rejected subscription", "pattern", msg.Pattern, "error", prob.Message)
		return
	}
	if a.session.Subscribe(sub) {
		a.logger.Info("subscribed", "pattern", msg.Pattern)
	}
}

func (a *SessionActor) onUnsubscribe(msg unsubscribeMessage) {
	if a.session.Unsubscribe(msg.Pattern) {
		a.logger.Info("unsubscribed", "pattern", msg.Pattern)
	}
}

func (a *SessionActor) onDeliver(msg deliverFrameMessage) {
	if !a.session.Matches(msg.Subject) {
		return
	}
	a.offer(msg.Payload)
}

// offer enqueues a frame for the write loop, dropping the newest frame
// if the bounded buffer is full (ADR-0028 I4). Called only on the actor
// goroutine, so dropped needs no synchronization.
func (a *SessionActor) offer(frame []byte) {
	select {
	case a.out <- frame:
	default:
		a.dropped++
	}
}
