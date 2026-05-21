# Residual Active Doc Reconciliation

**Date:** 2026-03-20
**Stage:** S230
**Status:** CLOSED — all four S228-identified drift items resolved

---

## 1. Problem Statement

S228 identified four residual drift items in the active documentation corpus:

1. obsolete migrate path example,
2. obsolete codegen marker example,
3. obsolete flat registry target path,
4. obsolete default-database execution-flow wording.

These contradicted the current repository state after the S218-S224
restructuring and the S229 tooling reconciliation.

---

## 2. Scope Definition

**In scope:** Active reference docs that provide current guidance to developers
or tooling. A doc is "active" if a reader following its instructions would
produce code or configuration that contradicts the current architecture.

**Out of scope:**
- Historical stage reports (cc-02-*, refactor-tranche-01-*, etc.)
- Diagnostic/meta documents that cite drift as a finding
- Documents that already self-document the supersession
- Archived documents

---

## 3. Drift Items Resolved

### 3.1 Obsolete default-database execution-flow wording

**File:** `cmd-migrate-and-migration-catalog.md`
**Problem:** Step 2 of the execution flow read "Connect to ClickHouse (default database)".
This was ambiguous: the ClickHouse system database named `default` is used for the
initial bootstrap connection, but readers could interpret it as the target database name
(which is `market_foundry`).
**Fix:** Clarified to "initial bootstrap connection to system database".

### 3.2 Obsolete flat registry target path

**File:** `codegen-current-usage-boundaries-and-limitations.md`
**Problem:** Integration map pointed governed consumer_spec slices to
`internal/adapters/nats/signal_registry.go` (pre-H-01 flat structure).
**Fix:** Updated to `internal/adapters/nats/natssignal/registry.go`.

**File:** `codegen-specification-and-schema.md`
**Problem:** Artifact coverage table showed target file as
`internal/adapters/nats/{domain}_registry.go` (pre-H-01 flat pattern).
**Fix:** Updated to `internal/adapters/nats/nats{domain}/registry.go`.

### 3.3 Obsolete codegen marker example

**File:** `codegen-specification-and-schema.md`
**Problem:** Rule 4 (Ownership Rules) showed the deprecated
`BEGIN/END CODEGEN MANAGED SECTION` markers as the operative integration format.
The current format is `codegen:begin/end` (established in S201).
**Fix:** Replaced with current `codegen:begin/end` format.

### 3.4 Obsolete migrate path example

**Resolution:** The migrate path example drift was subsumed by item 3.1.
The execution flow in `cmd-migrate-and-migration-catalog.md` was the only
active doc providing a misleading path description. The project layout
(lines 38-57) and configuration (lines 25-34) sections were already correct.

---

## 4. Verification

After all changes:

```
make check          → PASS (84 checks, 0 errors, 0 warnings)
make quality-gate-ci → PASS (84 checks, 0 errors, 0 warnings)
```

Documentation changes do not affect gate checks, but the gates confirm
no regressions were introduced.

---

## 5. Residual Assessment

### Active docs with old references that were intentionally NOT changed

11 documents contain historical references to `signal_registry.go` or
old marker formats. All were reviewed and determined to be:
- historical records (stage reports, rationale docs),
- diagnostic documents citing the drift as a finding, or
- documents that already self-document the supersession.

See `residual-active-docs-change-log.md` for the full exclusion list with rationale.

### Remaining known documentation limits

1. **Document count** — the `docs/architecture/` corpus remains large (~270 files).
   This is a structural characteristic, not actionable drift.
2. **Historical references** — old paths appear in historical docs where they are
   correct for the time period described. Changing them would falsify the record.
3. **Codegen spec referential integrity** — `codegen-specification-and-schema.md`
   line 162 references `cmd/writer/mappers.go` for mapper validation. This path
   is part of the codegen spec schema (describing where mappers would be placed),
   not a current-state assertion. It remains valid as a spec target.
