-- Migration: 016_create_insights_cross_venue
-- Created: 2026-06-13
-- Description: Cross-venue trade fusion history for insights analytics.
--              Decision-support read-only (ADR-0027), trades-only, windowed.
-- Source: internal/domain/insights/cross_venue.go (CrossVenueSnapshot +
--         events.Metadata); PROGRAM-0005 Onda H-8.c.1 (FECHA a Fase).
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS).
-- Reversible: Yes (DROP TABLE insights_cross_venue).
--
-- Schema note (PROGRAM-0005 Decisão C5 / T5 pattern): one snapshot fuses
-- one canonical instrument across venues for a timeframe window. The
-- per-venue rows persist as PARALLEL, index-aligned Array columns
-- (venue_name[i] / venue_trade_count[i] / venue_notional[i] /
-- venue_last_price[i] / ...); 1-event->1-row preserved. Consolidated
-- metrics (spread/mid/dominant) are scalar columns. Decimals as String.
--
-- NO `source` column — cross-venue fusion spans sources by design; the
-- canonical instrument (base/quote/contract, ADR-0021) is the join key.

CREATE TABLE IF NOT EXISTS insights_cross_venue (
    -- Event metadata
    event_id           String,
    occurred_at        DateTime64(3),
    correlation_id     String DEFAULT '',
    causation_id       String DEFAULT '',

    -- Domain fields (from insights.CrossVenueSnapshot) — no source
    symbol             LowCardinality(String),
    base               LowCardinality(String),
    quote              LowCardinality(String),
    contract           LowCardinality(String),
    timeframe          UInt32,

    -- Per-venue rows — parallel arrays, index-aligned
    venue_name         Array(String),
    venue_trade_count  Array(Int64),
    venue_notional     Array(String),
    venue_last_price   Array(String),
    venue_high_price   Array(String),
    venue_low_price    Array(String),

    -- Consolidated metrics (scalar)
    spread_abs         String,
    spread_bps         String,
    mid_price          String,
    dominant_venue     LowCardinality(String),

    trade_count        Int64,
    open_time          DateTime64(3),
    close_time         DateTime64(3),
    final              Bool,

    -- Ingestion metadata
    ingested_at        DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (symbol, timeframe, open_time)
TTL toDateTime(open_time) + INTERVAL 90 DAY
