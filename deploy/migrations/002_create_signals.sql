-- Migration: 002_create_signals
-- Created: 2026-03-19
-- Description: Signal events storage for analytical queries (RSI, EMA crossover).
-- Source: internal/domain/signal/signal.go (Signal + events.Metadata)
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS)
-- Reversible: Yes (DROP TABLE signals)

CREATE TABLE IF NOT EXISTS signals (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from Signal)
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    value          Float64,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL toDateTime(timestamp) + INTERVAL 90 DAY
