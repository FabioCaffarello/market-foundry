package delivery

import (
	deliverydomain "internal/domain/delivery"

	"github.com/anthdm/hollywood/actor"
)

// registerSessionMessage registers a session actor with the router so it
// receives broadcast events.
type registerSessionMessage struct {
	ID  deliverydomain.SessionID
	PID *actor.PID
}

// unregisterSessionMessage removes a session from the router (client
// disconnected).
type unregisterSessionMessage struct {
	ID deliverydomain.SessionID
}

// eventReceivedMessage is a delivery frame fresh off NATS, sent to the
// router for fan-out.
type eventReceivedMessage struct {
	Subject string
	Payload []byte
}

// deliverFrameMessage is a frame the router fanned out to one session
// actor; the session decides — by subscription match — whether to write
// it to its client.
type deliverFrameMessage struct {
	Subject string
	Payload []byte
}

// subscribeMessage adds a subscription pattern to a session (from a
// client control frame).
type subscribeMessage struct {
	Pattern string
}

// unsubscribeMessage removes a subscription pattern from a session.
type unsubscribeMessage struct {
	Pattern string
}
