package delivery

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	deliverydomain "internal/domain/delivery"

	"github.com/anthdm/hollywood/actor"
)

// RouterActor is the delivery fan-out hub: it holds the set of live
// session actors and broadcasts each insights event to all of them.
// Each session decides, by subscription match, whether to forward the
// event to its client. The router holds no subscription state — that
// lives in the per-session actors (single-owner; the ADR-0008 invariant
// applied to in-memory session state).
type RouterActor struct {
	logger   *slog.Logger
	sessions map[deliverydomain.SessionID]*actor.PID
}

// NewRouterActor returns a producer for the delivery router.
func NewRouterActor() actor.Producer {
	return func() actor.Receiver {
		return &RouterActor{sessions: make(map[deliverydomain.SessionID]*actor.PID)}
	}
}

func (r *RouterActor) Receive(c *actor.Context) {
	if r.logger == nil {
		r.logger = slog.Default().With("actor", "delivery-router")
	}
	switch msg := c.Message().(type) {
	case actor.Started:
		r.logger.Info("delivery router started")
	case actor.Stopped:
		r.logger.Info("delivery router stopped", "sessions", len(r.sessions))
	case registerSessionMessage:
		r.sessions[msg.ID] = msg.PID
		r.logger.Info("session registered", "session", string(msg.ID), "total", len(r.sessions))
	case unregisterSessionMessage:
		delete(r.sessions, msg.ID)
		r.logger.Info("session unregistered", "session", string(msg.ID), "total", len(r.sessions))
	case eventReceivedMessage:
		// eventReceivedMessage and deliverFrameMessage share a shape; the
		// conversion keeps them as distinct mailbox types (router-inbound
		// vs session-inbound) without a redundant field-by-field literal.
		frame := deliverFrameMessage(msg)
		for _, pid := range r.sessions {
			c.Send(pid, frame)
		}
	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		r.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
