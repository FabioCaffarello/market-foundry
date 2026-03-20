-- Migration: 001_create_evidence_candles
-- Created: 2026-03-19
-- Description: Historical candle storage for backtesting and trend analysis.
-- Source: internal/domain/evidence/candle.go (EvidenceCandle + events.Metadata)
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS)
-- Reversible: Yes (DROP TABLE evidence_candles)

CREATE TABLE IF NOT EXISTS evidence_candles (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from EvidenceCandle)
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    open           Float64,
    high           Float64,
    low            Float64,
    close          Float64,
    volume         Float64,
    trade_count    Int64,
    open_time      DateTime64(3),
    close_time     DateTime64(3),
    final          Bool,

    -- Ingestion metadata
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (source, symbol, timeframe, open_time)
TTL toDateTime(open_time) + INTERVAL 90 DAY
