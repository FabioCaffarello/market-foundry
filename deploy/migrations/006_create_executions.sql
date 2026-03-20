-- Migration: 006_create_executions
-- Created: 2026-03-19
-- Description: Execution events (paper orders, venue fills) for analytical queries and audit.
-- Source: internal/domain/execution/execution.go (ExecutionIntent + events.Metadata)
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS)
-- Reversible: Yes (DROP TABLE executions)

CREATE TABLE IF NOT EXISTS executions (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from ExecutionIntent)
    type                LowCardinality(String),
    source              LowCardinality(String),
    symbol              LowCardinality(String),
    timeframe           UInt32,
    side                LowCardinality(String),
    quantity            Float64,
    filled_quantity     Float64,
    status              LowCardinality(String),
    risk                String,
    fills               String,
    parameters          String,
    metadata            String,
    exec_correlation_id String DEFAULT '',
    exec_causation_id   String DEFAULT '',
    final               Bool,
    timestamp           DateTime64(3),

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL toDateTime(timestamp) + INTERVAL 90 DAY
