package clickhouse

import (
	"context"
	"fmt"

	ch "github.com/ClickHouse/clickhouse-go/v2"
)

// Config holds ClickHouse connection parameters.
type Config struct {
	Addr     string // host:port (e.g., "clickhouse:9000")
	Database string // database name (e.g., "default")
	Username string // auth username
	Password string // auth password
}

// Client wraps a ClickHouse native protocol connection.
type Client struct {
	conn ch.Conn
}

// Open creates a new ClickHouse client using the native protocol.
func Open(cfg Config) (*Client, error) {
	conn, err := ch.Open(&ch.Options{
		Addr: []string{cfg.Addr},
		Auth: ch.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}
	return &Client{conn: conn}, nil
}

// Ping verifies the connection to ClickHouse.
func (c *Client) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Close shuts down the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// InsertBatch inserts multiple rows using the ClickHouse batch protocol.
// insertSQL must be an INSERT INTO statement (e.g., "INSERT INTO evidence_candles").
// Each row is a slice of column values in DDL column order.
func (c *Client) InsertBatch(ctx context.Context, insertSQL string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := c.conn.PrepareBatch(ctx, insertSQL)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}
	for _, row := range rows {
		if err := batch.Append(row...); err != nil {
			_ = batch.Abort()
			return fmt.Errorf("append row: %w", err)
		}
	}
	return batch.Send()
}
