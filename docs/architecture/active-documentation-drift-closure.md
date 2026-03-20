# Active Documentation Drift Closure

**Stage:** S225
**Date:** 2026-03-20
**Scope:** Surgical reconciliation of the active documentation corpus after S217–S224

---

## 1. Purpose

S225 closes the residual active-doc drift that remained after the restructure tranche and the S224 tooling reconciliation.

This stage does **not** reopen broad documentation cleanup. It closes the specific contradictions that still described the pre-restructure architecture as current in the active corpus.

---

## 2. Current-State Baseline Used for Reconciliation

The active corpus was reconciled against these repository facts:

1. `go.work` lists **17 modules**, not 19.
2. `internal/adapters/nats/` is domain-organized (`natsconfigctl`, `natsdecision`, `natsevidence`, `natsexecution`, `natskit`, `natsobservation`, `natsrisk`, `natssignal`, `natsstrategy`).
3. The maintained binary set is:
   - Long-running runtimes: `configctl`, `gateway`, `ingest`, `derive`, `store`, `execute`, `writer`
   - Standalone deployment tool: `migrate`
4. The compose topology includes `clickhouse`, `execute`, and `writer` in addition to the original first-slice services.
5. The migrate library now lives at `cmd/migrate/migrate/`, not `internal/migrate/`.
6. The active codegen integration protocol uses `codegen:begin` / `codegen:end` markers and current NATS sub-package registry paths.

---

## 3. Drift Closed

### 3.1 Canonical and operational docs

The following active docs were updated because they are used as current-state entry points or governance rules:

- `README.md`
- `DEVELOPMENT.md`
- `docs/tooling/cli-overview.md`
- `docs/architecture/market-foundry-evolution-playbook.md`
- `docs/architecture/system-vision.md`
- `docs/architecture/stage-definition-of-done.md`
- `docs/architecture/anti-debt-checklist.md`

### 3.2 Residual factual drift in active technical docs

The following docs remained active but still pointed to deleted paths or superseded integration assumptions, so they were directly reconciled:

- `docs/architecture/analytical-generated-path-consolidation.md`
- `docs/architecture/analytical-vs-generated-ownership-and-boundaries.md`
- `docs/architecture/codegen-boundaries-and-governance.md`
- `docs/architecture/cmd-migrate-and-migration-catalog.md`
- `docs/architecture/migrations-infrastructure-architecture.md`

### 3.3 Historical active doc reframed instead of rewritten

- `docs/architecture/post-refactor-and-documentation-exit-gate.md`

This document was kept active for traceability, but it now explicitly states that its unresolved H-01/H-04/H-06 language is historical S216 gate context, not current-state architecture guidance.

---

## 4. Measurable Closure

S225 closed the following high-impact contradiction classes in the active corpus:

| Drift class | Before S225 | After S225 |
|-------------|-------------|------------|
| Current-state docs still using the five-binary ceiling | Present in canonical governance docs | Removed from canonical governance docs |
| Current-state docs still pointing to `internal/migrate/` as live layout | Present in active migrate/codegen docs | Repointed to `cmd/migrate/migrate/` |
| Current-state docs still using flat NATS registry paths | Present in active codegen docs | Repointed to `internal/adapters/nats/<domain>/registry.go` |
| Current-state docs still narrating old codegen marker protocol as active | Present in active governance docs | Replaced with the `codegen:begin` / `codegen:end` protocol |
| Top-level entry docs missing `execute`, `writer`, `clickhouse`, or `migrate` | Present in README / development workflow docs | Reconciled to current topology |
| Historical gate doc readable as current-state guidance | Present | Explicitly reframed as historical snapshot |

---

## 5. What S225 Did Not Reopen

S225 intentionally did **not**:

1. re-run broad active-doc consolidation or archival campaigns,
2. rewrite the entire architecture corpus around a new narrative,
3. reclassify archived documents,
4. change stage-history documents whose value is primarily historical unless they actively contradicted current navigation or governance,
5. absorb unrelated medium-priority structural debt.

---

## 6. Outcome

The principal active corpus now aligns with the post-S217–S224 codebase and governance surface:

- entry docs match the real topology,
- canonical governance docs no longer enforce deleted architectural assumptions,
- active codegen and migrate docs point to current paths and protocols,
- historical gate material preserves traceability without masquerading as current state.
