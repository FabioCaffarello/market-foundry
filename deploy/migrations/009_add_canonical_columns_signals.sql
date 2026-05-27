-- Migration: 009_add_canonical_columns_signals
-- Created: 2026-05-27
-- Description: Add canonical instrument columns (base/quote/contract) to
--              signals per ADR-0021 criterion #4b.
-- Source: PROGRAM-0004 Onda H-6.d.1 (schema + writer migration).
-- Logical unit: H-6.d.1 canonical instrument columns — 008-013 add the same
--               3 columns to all 6 Instrument-bearing tables. See 008 header
--               for the per-file split rationale (ClickHouse Go driver
--               multi-statement constraint).
-- Idempotent: Yes (ADD COLUMN IF NOT EXISTS).
-- Reversible: Yes (ALTER TABLE signals DROP COLUMN base, DROP COLUMN quote,
--                  DROP COLUMN contract).

ALTER TABLE signals
    ADD COLUMN IF NOT EXISTS base     LowCardinality(String) DEFAULT '' AFTER symbol,
    ADD COLUMN IF NOT EXISTS quote    LowCardinality(String) DEFAULT '' AFTER base,
    ADD COLUMN IF NOT EXISTS contract LowCardinality(String) DEFAULT '' AFTER quote
