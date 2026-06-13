-- Migration: 015_create_insights_tpo
-- Created: 2026-06-13
-- Description: TPO (Time-Price Opportunity / market profile) history for
--              insights analytics. Decision-support read-only (ADR-0027),
--              trades-only, timeframe-anchored.
-- Source: internal/domain/insights/tpo.go (TPOProfile + events.Metadata);
--         PROGRAM-0005 Onda H-8.b.1.
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS).
-- Reversible: Yes (DROP TABLE insights_tpo).
--
-- Schema note (PROGRAM-0005 Decisão T5, padrão H-8.a.1): a TPOProfile
-- carries TWO repeated structures — periods (A..X) and price levels. Each
-- window persists as ONE row (1-event->1-row preserved) with each structure
-- held in PARALLEL, index-aligned Array columns:
--   periods:  period_letter[i] / period_high[i] / period_low[i]
--   levels:   level_price[i] / level_letters[i] / level_count[i]
-- (level_letters[i] is the concatenated period labels at that level, e.g.
-- "ACF".) Decimals stay as String to preserve exact precision (the binning
-- is big.Rat-deterministic). Derived metrics (POC / value area / initial
-- balance / range) are scalar String columns. Idiomatic for ClickHouse
-- analytics (arrayJoin over the parallel arrays). Canonical instrument
-- columns per ADR-0021; fresh table, every row written canonical.

CREATE TABLE IF NOT EXISTS insights_tpo (
    -- Event metadata
    event_id             String,
    occurred_at          DateTime64(3),
    correlation_id       String DEFAULT '',
    causation_id         String DEFAULT '',

    -- Domain fields (from insights.TPOProfile)
    source               LowCardinality(String),
    symbol               LowCardinality(String),
    base                 LowCardinality(String),
    quote                LowCardinality(String),
    contract             LowCardinality(String),
    timeframe            UInt32,
    bucket_size          String,
    period_seconds       UInt32,

    -- Periods — parallel arrays, index-aligned (Decisão T5)
    period_letter        Array(String),
    period_high          Array(String),
    period_low           Array(String),

    -- Price levels — parallel arrays, index-aligned
    level_price          Array(String),
    level_letters        Array(String),
    level_count          Array(Int32),

    -- Derived metrics (scalar)
    poc_price            String,
    value_area_high      String,
    value_area_low       String,
    initial_balance_high String,
    initial_balance_low  String,
    range_high           String,
    range_low            String,

    trade_count          Int64,
    overload             LowCardinality(String),
    open_time            DateTime64(3),
    close_time           DateTime64(3),
    final                Bool,

    -- Ingestion metadata
    ingested_at          DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (source, symbol, timeframe, open_time)
TTL toDateTime(open_time) + INTERVAL 90 DAY
