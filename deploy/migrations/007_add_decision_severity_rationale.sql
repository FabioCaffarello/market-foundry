-- Migration: 007_add_decision_severity_rationale
-- Created: 2026-03-20
-- Description: Add severity and rationale columns to decisions table.
-- Source: S234 decision domain deepening
-- Idempotent: Yes (ALTER TABLE ADD COLUMN IF NOT EXISTS)
-- Reversible: Yes (ALTER TABLE DROP COLUMN)

ALTER TABLE decisions ADD COLUMN IF NOT EXISTS severity LowCardinality(String) DEFAULT '' AFTER confidence;
ALTER TABLE decisions ADD COLUMN IF NOT EXISTS rationale String DEFAULT '' AFTER severity;
