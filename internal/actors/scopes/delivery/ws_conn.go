// Package delivery is the actor scope for push delivery of insights
// events to connected WebSocket clients. A RouterActor fans each event
// out to per-connection SessionActors; each SessionActor owns its
// client's subscriptions (domain delivery.Session) and a bounded
// outbound buffer with DropNewest backpressure (ADR-0028 I4). The scope
// holds no transport dependency — connections are reached through the
// WSConn interface, adapted to gorilla by the interfaces layer.
package delivery

// WSConn is the minimal connection surface the delivery actors need to
// push frames to a client. The interfaces layer adapts a concrete
// WebSocket connection to this; the actor layer stays transport-free
// (layer sovereignty).
//
// Send is called only by the single write goroutine a SessionActor owns
// (gorilla permits one concurrent writer). Close must be idempotent: it
// may be called by both the write loop (on write error) and the actor
// (on stop).
type WSConn interface {
	Send(frame []byte) error
	Close() error
}
