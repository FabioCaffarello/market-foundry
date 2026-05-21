# Active Docs Reconciliation Log

**Stage:** S225
**Date:** 2026-03-20

---

## 1. Log

| Document | Drift Type | Action | Why |
|----------|------------|--------|-----|
| `README.md` | Missing binaries, stale internal path, broken forward reference | Updated | Top-level entry doc must match current topology |
| `DEVELOPMENT.md` | Stale stack description and project layout | Updated | Developer workflow doc must match Makefile + compose + `go.work` |
| `docs/tooling/cli-overview.md` | Stale five-binary invariant | Updated | Tooling overview must reflect current governed binary set |
| `docs/architecture/market-foundry-evolution-playbook.md` | Pre-execute/pre-writer topology assumptions | Updated | Canonical governance doc must describe current runtime and stream baseline |
| `docs/architecture/system-vision.md` | Future-state language where code already exists | Updated | Canonical identity doc must not understate the implemented system |
| `docs/architecture/stage-definition-of-done.md` | Stale mesh path and binary ceiling | Updated | Stage acceptance rules must match current architecture |
| `docs/architecture/anti-debt-checklist.md` | Stale unidirectional-path formulation | Updated | Review checklist must test the real topology |
| `docs/architecture/analytical-generated-path-consolidation.md` | Flat NATS registry paths | Updated | Active codegen doc must point to current sub-package layout |
| `docs/architecture/analytical-vs-generated-ownership-and-boundaries.md` | Flat NATS registry paths and stale migrate path | Updated | Ownership doc must reference real files |
| `docs/architecture/codegen-boundaries-and-governance.md` | Deprecated marker protocol presented as active, overstated generation scope | Updated | Governance doc must describe the current integration protocol only |
| `docs/architecture/cmd-migrate-and-migration-catalog.md` | Stale `internal/migrate/` path | Updated | Current migrate architecture lives under `cmd/migrate/migrate/` |
| `docs/architecture/migrations-infrastructure-architecture.md` | Stale migrate layout and unsupported make target | Updated | Active design doc must reconcile to implemented layout |
| `docs/architecture/post-refactor-and-documentation-exit-gate.md` | Historical assessment still readable as current | Reframed | Preserve traceability without leaving live contradictions |

---

## 2. Reconciliation Strategy

S225 used three different actions, depending on the document's role:

1. **Update in place** for entry-point and canonical governance docs that must always describe the current system.
2. **Update factual references** for active technical docs whose purpose remains valid but whose paths/protocols had drifted.
3. **Reframe, not rewrite** for historical gate material that still needs to remain visible for traceability.

---

## 3. No-Archive Decision

No active documents were archived in S225.

Reason:
- the residual drift was concentrated in current-state wording and path references,
- the affected docs still have live governance or traceability value,
- targeted correction was sufficient to remove the contradiction without reopening a broad archival wave.
