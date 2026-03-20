package clickhouse

import (
	"context"
	"fmt"
)

// Rows represents query result rows from a ClickHouse SELECT query.
// Callers must call Close() when done iterating.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// Query executes a SELECT query and returns result rows.
// The caller is responsible for calling Close() on the returned Rows.
func (c *Client) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := c.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query clickhouse: %w", err)
	}
	return rows, nil
}
