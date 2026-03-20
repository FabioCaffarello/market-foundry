-- Migration: 005_create_risk_assessments
-- Created: 2026-03-19
-- Description: Risk assessment events for analytical queries.
-- Source: internal/domain/risk/risk.go (RiskAssessment + events.Metadata)
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS)
-- Reversible: Yes (DROP TABLE risk_assessments)

CREATE TABLE IF NOT EXISTS risk_assessments (
    -- Event metadata
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',

    -- Domain fields (from RiskAssessment)
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    timeframe      UInt32,
    disposition    LowCardinality(String),
    confidence     Float64,
    strategies     String,
    constraints    String,
    rationale      String,
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
