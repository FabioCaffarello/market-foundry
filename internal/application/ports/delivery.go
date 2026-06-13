package ports

// Delivery ports (ADR-0028 / PROGRAM-0006) decouple the gateway's HTTP
// layer from the delivery actor subsystem. The interfaces layer depends
// on these interfaces; the actor scope implements them; the gateway
// binary wires the concrete implementation to the HTTP handler. This
// keeps interfaces/ free of any actors/ import (layer sovereignty,
// ADR-0005).

// DeliveryConn is the connection surface delivery writes frames to. The
// interfaces layer adapts a concrete WebSocket connection (gorilla) to
// this; the delivery session writes JSON frames through it.
//
// Send is called only by the session's single write goroutine; Close
// must be safe to call once per teardown path.
type DeliveryConn interface {
	Send(frame []byte) error
	Close() error
}

// DeliverySession is the gateway HTTP handler's grip on one admitted
// delivery session. Subscribe/Unsubscribe map client control frames to
// session state; Close tears the session down when the connection ends.
type DeliverySession interface {
	Subscribe(pattern string)
	Unsubscribe(pattern string)
	Close()
}

// DeliveryHub admits delivery connections, spawning a session for each.
// Implemented by the delivery actor scope; consumed by the gateway HTTP
// handler.
type DeliveryHub interface {
	Admit(conn DeliveryConn) DeliverySession
}
