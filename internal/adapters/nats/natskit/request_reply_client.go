package natskit

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
)

// RequestReplyClient is the interface for sending request/reply messages.
type RequestReplyClient interface {
	Request(context.Context, string, []byte) ([]byte, error)
}

// NATSRequestClient is a NATS-based implementation of RequestReplyClient.
type NATSRequestClient struct {
	nc      *nats.Conn
	timeout time.Duration
}

// NewNATSRequestClientWithURL creates a new NATSRequestClient connected to the given URL.
func NewNATSRequestClientWithURL(url string, timeout time.Duration) (*NATSRequestClient, error) {
	nc, err := Connect(url)
	if err != nil {
		return nil, err
	}

	return &NATSRequestClient{
		nc:      nc,
		timeout: timeout,
	}, nil
}

func (c *NATSRequestClient) Request(ctx context.Context, subject string, payload []byte) ([]byte, error) {
	requestCtx := ctx
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && c.timeout > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, c.timeout)
	}
	defer cancel()

	msg, err := c.nc.RequestWithContext(requestCtx, subject, payload)
	if err != nil {
		return nil, err
	}

	return msg.Data, nil
}

func (c *NATSRequestClient) Close() error {
	if c != nil && c.nc != nil {
		c.nc.Close()
	}
	return nil
}
