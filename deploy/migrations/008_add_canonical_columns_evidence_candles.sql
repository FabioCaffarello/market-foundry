-- Migration: 008_add_canonical_columns_evidence_candles
-- Created: 2026-05-27
-- Description: Add canonical instrument columns (base/quote/contract) to
--              evidence_candles per ADR-0021 criterion #4b.
-- Source: PROGRAM-0004 Onda H-6.d.1 (schema + writer migration).
-- Logical unit: H-6.d.1 canonical instrument columns — 008-013 add the same
--               3 columns to all 6 Instrument-bearing tables. Split into one
--               migration per table because the ClickHouse Go driver
--               (clickhouse-go/v2) accepts one statement per ExecContext;
--               multi-statement files fail with code 62 "Multi-statements
--               are not allowed". Migration-runner enhancement to handle
--               multi-statement files is recorded as a deferred tooling
--               improvement (PROGRAM-0004 H-6.f / dedicated wave).
-- Idempotent: Yes (ADD COLUMN IF NOT EXISTS).
-- Reversible: Yes (ALTER TABLE evidence_candles DROP COLUMN base,
--                  DROP COLUMN quote, DROP COLUMN contract).
--
-- Back-compat strategy: pre-migration rows get DEFAULT '' for the 3 new
-- columns. Readers detect empty canonical columns and fall back to the
-- existing reconstructInstrumentFromLegacy(source, symbol) helper for
-- legacy rows (H-6.d.2). After the 90-day MergeTree TTL retires all legacy
-- rows, the helper becomes dead code and is deleted in H-6.f cleanup pass.

ALTER TABLE evidence_candles
    ADD COLUMN IF NOT EXISTS base     LowCardinality(String) DEFAULT '' AFTER symbol,
    ADD COLUMN IF NOT EXISTS quote    LowCardinality(String) DEFAULT '' AFTER base,
    ADD COLUMN IF NOT EXISTS contract LowCardinality(String) DEFAULT '' AFTER quote
