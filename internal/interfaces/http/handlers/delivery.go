package handlers

import (
	"log/slog"
	"net/http"

	"internal/application/ports"
	"internal/shared/problem"

	"github.com/gorilla/websocket"
)

// deliveryReadLimitBytes caps inbound control frames. Subscribe/
// unsubscribe frames are tiny; a larger frame is abuse, not protocol.
const deliveryReadLimitBytes = 4096

// DeliveryWebHandler upgrades HTTP requests to WebSocket and bridges them
// to the delivery hub. Read-only transport (ADR-0028 I1): the only
// inbound frames are subscribe/unsubscribe control frames; no event or
// directive is ever accepted from the client. Loopback-only (I2): the
// gateway binds loopback, so the origin check is not the access control
// — network isolation is.
type DeliveryWebHandler struct {
	hub      ports.DeliveryHub
	upgrader websocket.Upgrader
	logger   *slog.Logger
}

// NewDeliveryWebHandler builds the WebSocket delivery handler over a
// delivery hub.
func NewDeliveryWebHandler(hub ports.DeliveryHub, logger *slog.Logger) *DeliveryWebHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &DeliveryWebHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			// Loopback binding is the access control (ADR-0028 I2); the
			// origin check is intentionally permissive.
			CheckOrigin: func(*http.Request) bool { return true },
		},
		logger: logger,
	}
}

// controlFrame is the client→server wire shape: a control action over a
// NATS subject pattern (ADR-0028).
type controlFrame struct {
	Action  string `json:"action"`
	Subject string `json:"subject"`
}

// Connect handles GET /ws: upgrade, admit the session, then loop reading
// control frames until the client disconnects. Outbound frames are
// written by the session actor's own goroutine (gorilla permits one
// concurrent reader + one writer).
func (h *DeliveryWebHandler) Connect(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.hub == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "delivery is unavailable"))
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade writes its own HTTP error response on handshake failure.
		h.logger.Warn("websocket upgrade failed", "error", err)
		return
	}
	conn.SetReadLimit(deliveryReadLimitBytes)

	handle := h.hub.Admit(newGorillaConn(conn))
	defer handle.Close()

	for {
		var frame controlFrame
		if err := conn.ReadJSON(&frame); err != nil {
			return // normal close or read error → tear the session down
		}
		switch frame.Action {
		case "subscribe":
			handle.Subscribe(frame.Subject)
		case "unsubscribe":
			handle.Unsubscribe(frame.Subject)
		default:
			h.logger.Warn("unknown delivery control action", "action", frame.Action)
		}
	}
}

// gorillaConn adapts a gorilla *websocket.Conn to delivery.WSConn. Only
// the session's single write goroutine calls Send. Close is called once
// per teardown path (gorilla returns an error rather than panicking on a
// double close).
type gorillaConn struct {
	conn *websocket.Conn
}

func newGorillaConn(conn *websocket.Conn) *gorillaConn { return &gorillaConn{conn: conn} }

func (g *gorillaConn) Send(frame []byte) error {
	return g.conn.WriteMessage(websocket.TextMessage, frame)
}

func (g *gorillaConn) Close() error { return g.conn.Close() }
