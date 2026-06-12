package bybits

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	baseWSURL      = "wss://stream.bybit.com/v5/public/spot"
	initialBackoff = 1 * time.Second
	maxBackoff     = 60 * time.Second
	backoffFactor  = 2
	readWait       = 30 * time.Second
	writeWait      = 10 * time.Second
	pingInterval   = 20 * time.Second
)

// MessageHandler is called for each raw WebSocket message received.
type MessageHandler func(data []byte)

// WSClient connects to the Bybit v5 public Spot WebSocket and
// subscribes to a single symbol's publicTrade topic.
//
// Protocol differences from the Binance adapters' URL-per-stream
// model (both intrinsic to Bybit v5):
//   - Subscriptions are explicit frames sent after connect
//     ({"op":"subscribe","args":["publicTrade.BTCUSDT"]}), not
//     URL path segments.
//   - Keepalive is an application-level {"op":"ping"} the CLIENT
//     sends every ~20s; Bybit closes idle connections, so the ping
//     loop is mandatory.
type WSClient struct {
	symbol  string
	topic   string
	url     string
	logger  *slog.Logger
	handler MessageHandler
}

// NewWSClient creates a WebSocket client for the given symbol's
// publicTrade topic.
func NewWSClient(symbol string, handler MessageHandler, logger *slog.Logger) *WSClient {
	topic := publicTradeTopic + strings.ToUpper(strings.TrimSpace(symbol))
	return &WSClient{
		symbol:  strings.ToLower(symbol),
		topic:   topic,
		url:     baseWSURL,
		logger:  logger,
		handler: handler,
	}
}

// Run connects to the WebSocket and reads messages until the context is cancelled.
// It reconnects on failure with exponential backoff (1s -> 2s -> 4s -> ... -> 60s cap).
// The backoff resets after a successful connection that lasts at least 30 seconds.
func (c *WSClient) Run(ctx context.Context) {
	backoff := initialBackoff

	for {
		if ctx.Err() != nil {
			return
		}

		connStart := time.Now()
		c.readLoop(ctx)

		if ctx.Err() != nil {
			return
		}

		if time.Since(connStart) > 30*time.Second {
			backoff = initialBackoff
		}

		c.logger.Warn("websocket disconnected, reconnecting",
			"topic", c.topic,
			"backoff", backoff.String(),
		)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff = time.Duration(float64(backoff) * backoffFactor)
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (c *WSClient) readLoop(ctx context.Context) {
	c.logger.Info("connecting websocket", "url", c.url, "topic", c.topic)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, nil)
	if err != nil {
		c.logger.Error("websocket dial failed", "error", err, "topic", c.topic)
		return
	}
	defer func() { _ = conn.Close() }()

	if err := c.subscribe(conn); err != nil {
		c.logger.Error("websocket subscribe failed", "error", err, "topic", c.topic)
		return
	}

	c.logger.Info("websocket connected", "topic", c.topic)

	pingCtx, cancelPing := context.WithCancel(ctx)
	defer cancelPing()
	go c.pingLoop(pingCtx, conn)

	for {
		if ctx.Err() != nil {
			return
		}

		if err := conn.SetReadDeadline(time.Now().Add(readWait)); err != nil {
			c.logger.Error("set read deadline", "error", err)
			return
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Warn("websocket read error", "error", err, "topic", c.topic)
			return
		}

		c.handler(data)
	}
}

// subscribe sends the explicit v5 subscription frame for the topic.
func (c *WSClient) subscribe(conn *websocket.Conn) error {
	frame := map[string]any{"op": "subscribe", "args": []string{c.topic}}
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return conn.WriteJSON(frame)
}

// pingLoop sends the application-level {"op":"ping"} keepalive Bybit
// requires; the resulting pong arrives as a control frame on the data
// socket (skipped by ParsePublicTrade) and refreshes the read
// deadline like any other message.
func (c *WSClient) pingLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := conn.WriteJSON(map[string]string{"op": "ping"}); err != nil {
				c.logger.Warn("websocket ping failed", "error", err, "topic", c.topic)
				return
			}
		}
	}
}

// StreamURL returns the WebSocket URL plus topic for testing/logging.
func (c *WSClient) StreamURL() string {
	return fmt.Sprintf("%s#%s", c.url, c.topic)
}

// Symbol returns the normalized symbol.
func (c *WSClient) Symbol() string {
	return c.symbol
}
