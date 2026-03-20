-- Migration: 000_create_migrations_metadata
-- Created: 2026-03-19
-- Description: Bootstrap the _migrations metadata table for schema version tracking.
-- Idempotent: Yes (CREATE TABLE IF NOT EXISTS)
-- Reversible: Yes (DROP TABLE _migrations)

CREATE TABLE IF NOT EXISTS _migrations (
    version    UInt32,
    name       String,
    applied_at DateTime64(3) DEFAULT now64(3),
    checksum   String
) ENGINE = MergeTree()
ORDER BY version
