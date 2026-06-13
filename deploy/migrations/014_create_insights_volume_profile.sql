-- Migration: 014_create_insights_volume_profile
-- Created: 2026-06-13
-- Description: Volume Profile (VPVR) history for insights analytics.
--              Decision-support read-only output (ADR-0027), trades-only.
-- Source: internal/domain/insights/volume_profile.go (VolumeProfile +
--         events.Metadata); PROGRAM-0005 Onda H-8.a.1 (G12 resolution).
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS).
-- Reversible: Yes (DROP TABLE insights_volume_profile).
--
-- Schema note (PROGRAM-0005 Decisão #6, Opção B): the VolumeProfile carries
-- a per-window slice of price buckets (buy/sell notional per level). Each
-- window persists as ONE row (1-event→1-row preserved) with the buckets held
-- in three PARALLEL Array(String) columns — bucket_price_level[i] pairs with
-- bucket_buy_volume[i] and bucket_sell_volume[i]. This is the first use of
-- Array columns in the foundry; it keeps the codegen writer contract intact
-- (the mapper builds the three slices and emits a single row) while staying
-- idiomatic for ClickHouse analytics (arrayJoin / array aggregations).
-- Decimal values are stored as String to preserve exact precision, matching
-- the domain representation (the binning is big.Rat-deterministic).
--
-- Canonical instrument columns (base/quote/contract) per ADR-0021. As a fresh
-- table every row is written with canonical columns populated by the writer
-- mapper; no legacy-fallback path exists here (unlike the H-6.d tables).

CREATE TABLE IF NOT EXISTS insights_volume_profile (
    -- Event metadata
    event_id           String,
    occurred_at        DateTime64(3),
    correlation_id     String DEFAULT '',
    causation_id       String DEFAULT '',

    -- Domain fields (from insights.VolumeProfile)
    source             LowCardinality(String),
    symbol             LowCardinality(String),
    base               LowCardinality(String),
    quote              LowCardinality(String),
    contract           LowCardinality(String),
    timeframe          UInt32,
    bucket_size        String,

    -- Price buckets — parallel arrays, index-aligned (Decisão #6, Opção B)
    bucket_price_level Array(String),
    bucket_buy_volume  Array(String),
    bucket_sell_volume Array(String),

    trade_count        Int64,
    overload           LowCardinality(String),
    open_time          DateTime64(3),
    close_time         DateTime64(3),
    final              Bool,

    -- Ingestion metadata
    ingested_at        DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (source, symbol, timeframe, open_time)
TTL toDateTime(open_time) + INTERVAL 90 DAY
