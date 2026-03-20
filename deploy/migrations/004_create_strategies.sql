-- Migration: 004_create_strategies
-- Created: 2026-03-19
-- Description: Strategy resolution events for analytical queries.
-- Source: internal/domain/strategy/strategy.go (Strategy + events.Metadata)
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS)
-- Reversible: Yes (DROP TABLE strategies)

CREATE TABLE IF NOT EXISTS strategies (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from Strategy)
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    direction      LowCardinality(String),
    confidence     Float64,
    decisions      String,
    parameters     String,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
TTL toDateTime(timestamp) + INTERVAL 90 DAY
