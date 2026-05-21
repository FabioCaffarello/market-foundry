package binances

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	baseWSURL      = "wss://stream.binance.com:9443/ws/"
	initialBackoff = 1 * time.Second
	maxBackoff     = 60 * time.Second
	backoffFactor  = 2
	pongWait       = 30 * time.Second
	writeWait      = 10 * time.Second
)

// MessageHandler is called for each raw WebSocket message received.
type MessageHandler func(data []byte)

// WSClient connects to a single Binance Spot WebSocket stream.
type WSClient struct {
	symbol  string
	stream  string
	url     string
	logger  *slog.Logger
	handler MessageHandler
}

// NewWSClient creates a WebSocket client for the given symbol's aggTrade stream.
func NewWSClient(symbol string, handler MessageHandler, logger *slog.Logger) *WSClient {
	stream := strings.ToLower(symbol) + "@aggTrade"
	return &WSClient{
		symbol:  strings.ToLower(symbol),
		stream:  stream,
		url:     baseWSURL + stream,
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
			"stream", c.stream,
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
	c.logger.Info("connecting websocket", "url", c.url)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, nil)
	if err != nil {
		c.logger.Error("websocket dial failed", "error", err, "stream", c.stream)
		return
	}
	defer conn.Close()

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	c.logger.Info("websocket connected", "stream", c.stream)

	for {
		if ctx.Err() != nil {
			return
		}

		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			c.logger.Error("set read deadline", "error", err)
			return
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Warn("websocket read error", "error", err, "stream", c.stream)
			return
		}

		c.handler(data)
	}
}

// StreamURL returns the full WebSocket URL for testing/logging.
func (c *WSClient) StreamURL() string {
	return c.url
}

// Symbol returns the normalized symbol.
func (c *WSClient) Symbol() string {
	return c.symbol
}

// FormatStreamURL builds the aggTrade stream URL for a symbol without creating a client.
func FormatStreamURL(symbol string) string {
	return fmt.Sprintf("%s%s@aggTrade", baseWSURL, strings.ToLower(symbol))
}
