# Fresh Remote CI Proof and Release Tag Closure

**Date:** 2026-03-20
**Stage:** S231
**Purpose:** Document the execution and outcome of the final remote CI proof, closing the mechanical gate pendency from S210/S228.

---

## 1. Objective

Convert the locally reconciled state from S229/S230 into formal remote proof by:

1. pushing the corrected baseline to origin/main,
2. verifying all three CI jobs pass on the GitHub Actions runner,
3. creating and publishing a release tag on the validated commit,
4. upgrading XC-6 / EC-7 from FAIL to PASS.

---

## 2. Pre-Push Validation

Before pushing, S231 validated the local baseline:

| Gate | Result |
|------|--------|
| `make test` | PASS — all Go modules |
| `make check` | PASS — 84 checks, 0 errors |
| `make quality-gate-ci` | PASS — 84 checks, 0 errors |
| `make codegen-check` | PASS — 14/14 |
| `make codegen-integrated` | PASS — 4/4 |
| `make build` | PASS — all 8 service binaries |

---

## 3. Additional Corrections Required for CI Green

### 3.1 Codegen template alignment (discovered during S231)

The S227 writer pipeline refactor introduced `writerpipeline.New*Starter` functions
and explicit column lists in `insertSQL`, but the codegen template and golden
snapshots were not updated. S231 corrected this:

- Added `writer.columns` field to all 7 family YAML specs
- Updated `spec.go` with `StarterFunc` derived field and column-aware `InsertSQL`
- Updated `pipeline_entry.go.tmpl` to emit the refactored pattern
- Updated all 7 `pipeline_entry.go.golden` files
- Updated `render_test.go` and `spec_test.go` assertions

### 3.2 Go 1.25 stdlib collision (discovered on first CI run)

The Go 1.25 toolchain reserves `cmd/` as a standard library prefix. The module
`cmd/migrate` imported its own subpackage as `"cmd/migrate/migrate"`, which
Go 1.25 resolved against the stdlib. S231 corrected this:

- Renamed `cmd/migrate/migrate/` to `cmd/migrate/engine/`
- Changed package name from `migrate` to `engine`
- Updated `main.go` import to `migrate "cmd/migrate/engine"`

---

## 4. Remote Push Sequence

| # | Commit SHA | Run ID | Result | Key finding |
|---|-----------|--------|--------|-------------|
| 1 | `5103f1c` | `23365278860` | FAIL | Codegen PASS, Unit PASS, Smoke FAIL — `make up` failed due to Go 1.25 `cmd/migrate/migrate` stdlib collision |
| 2 | `edb30107e2f1af3caf22490d90b9a709e5be6bdf` | `23365571775` | **PASS** | All three jobs green — first full CI green in project history |

---

## 5. Release Tag

| Field | Value |
|-------|-------|
| Tag | `v0.1.0-s231` |
| Commit | `edb30107e2f1af3caf22490d90b9a709e5be6bdf` |
| Run ID | `23365571775` |
| Jobs | Codegen Golden Equivalence (PASS), Unit Tests (PASS), Smoke Analytical E2E (PASS) |
| Gate | XC-6 / EC-7 upgraded from FAIL to **PASS** |

---

## 6. Scope Discipline

S231 did not:

- redesign the CI pipeline,
- add new CI jobs or steps,
- introduce new operational features,
- change the runtime behavior of any service,
- relax any guard rail or gate criterion.

S231 did:

- align the codegen template with the S227 refactor (mechanical),
- fix a Go 1.25 stdlib path collision (mechanical),
- push, observe, correct, and re-push (evidence-driven),
- create and publish the exit tag.

---

## 7. Residual Non-Blocking Notes

The following CI annotations are present but do not affect the gate:

1. Node.js 20 deprecation warnings for `actions/checkout@v4`, `actions/setup-go@v5`
2. Cache restore warning due to absent root-level `go.sum`

These are hygiene items for a future charter, not blockers.
