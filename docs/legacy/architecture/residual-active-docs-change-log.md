# Residual Active Docs — Change Log

**Date:** 2026-03-20
**Stage:** S230
**Purpose:** Trace every doc change made during residual active-doc reconciliation

---

## Changes Applied

### 1. cmd-migrate-and-migration-catalog.md

| Line | Before | After | Reason |
|------|--------|-------|--------|
| 63 | `Connect to ClickHouse (default database)` | `Connect to ClickHouse (initial bootstrap connection to system database)` | "default database" was ambiguous — could be misread as the target database name. The target is `market_foundry` (configured at line 31). The initial connection uses the ClickHouse system database to bootstrap `CREATE DATABASE IF NOT EXISTS`. |

### 2. codegen-current-usage-boundaries-and-limitations.md

| Lines | Before | After | Reason |
|-------|--------|-------|--------|
| 73 | `internal/adapters/nats/signal_registry.go` | `internal/adapters/nats/natssignal/registry.go` | H-01 restructuring moved flat registry files into per-domain packages. |
| 75 | `internal/adapters/nats/signal_registry.go` | `internal/adapters/nats/natssignal/registry.go` | Same as above. |

### 3. codegen-specification-and-schema.md

| Lines | Before | After | Reason |
|-------|--------|-------|--------|
| 202 | `internal/adapters/nats/{domain}_registry.go` | `internal/adapters/nats/nats{domain}/registry.go` | Artifact target path updated to match H-01 per-domain package structure. |
| 289-293 | `// --- BEGIN CODEGEN MANAGED SECTION ---` / `// --- END CODEGEN MANAGED SECTION ---` | `// codegen:begin <artifact_type> family=<family_name> source=<spec_path>` / `// codegen:end <artifact_type> family=<family_name>` | Old S202 marker format superseded by S201 `codegen:begin/end` format. The specification document was still showing the deprecated format as the operative example in Rule 4. |

---

## Files NOT Changed (and why)

| File | Contains old reference? | Why no change |
|------|------------------------|---------------|
| `codegen-boundaries-and-governance.md` | No — already uses correct `codegen:begin/end` and correct path `natssignal/registry.go` | Already reconciled in S225 or earlier |
| `analytical-generated-path-consolidation.md` | Yes — line 48 shows old markers | Already self-documented as deprecated at line 99; the document explicitly records the supersession |
| `cc-02-implementation-notes.md` | Yes — `signal_registry.go` | Historical stage record; references are to the state at time of S203, not current guidance |
| `cc-02-family-definition.md` | Yes — `signal_registry.go` | Historical stage record |
| `refactor-tranche-01-changes-rationale-and-impact.md` | Yes — `signal_registry.go` | Historical rationale doc; describes what was changed, not current state |
| `repository-architecture-census-and-refactor-map.md` | Yes — `signal_registry.go` | Census snapshot at time of creation; not active guidance |
| `strategic-runtime-and-package-refactor.md` | Yes — `signal_registry.go` | Historical analysis document |
| `post-restructure-gate-and-next-charter-decision.md` | Yes — identifies the drift | Meta-document that diagnosed the problem; its references are citations, not guidance |
| `final-pre-charter-gate.md` | Yes — identifies the drift | Same — diagnostic document citing the problem |
| `governance-tooling-before-and-after-restructure.md` | Yes — but correctly shows before/after | Already reconciled; the old path appears in the "Before" column |
| `raccoon-cli-and-quality-gate-reconciliation.md` | Yes — `signal_registry.go` | Historical reconciliation record |
