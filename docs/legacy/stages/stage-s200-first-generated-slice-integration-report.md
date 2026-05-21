# Stage S200 — First Generated Slice Integration Report

**Status**: Complete
**Date**: 2026-03-20
**Scope**: Integrate one codegen-governed slice into the real monorepo flow

---

## 1. Executive Summary

S200 integrates the first codegen-governed slice into the real monorepo runtime. The RSI signal family's A1 (consumer spec) and A2 (pipeline entry) are now demarcated with governance markers, tracked in an integration manifest, and verified by a CI gate that compares marked regions against golden snapshots. No runtime behavior changed — the code was already structurally identical to the golden output. What changed is the governance model: these regions are now formally owned by the codegen pipeline.

---

## 2. Slice Selected and Rationale

**Family**: RSI
**Layer**: Signal
**Artifacts**: A1 (consumer_spec) + A2 (pipeline_entry)

| Selection criterion | RSI signal | Notes |
|---------------------|------------|-------|
| Non-evidence layer | Yes | Avoids naming exception complexity |
| Known abbreviation | Yes | RSI exercises PascalCase derivation |
| Mid-complexity | Yes | Signal is representative of 5/6 layers |
| Pre-validated (S196) | Yes | 0 structural drift |
| Minimal blast radius | Yes | No new infrastructure required |
| Existing table and stream | Yes | `signals` table, `SIGNAL_EVENTS` stream |

---

## 3. Integration Realized

### 3.1 Governance markers added

**`internal/adapters/nats/signal_registry.go`**:
```go
// codegen:begin consumer_spec family=rsi source=codegen/families/rsi.yaml
func WriterRSISignalConsumer() ConsumerSpec { ... }
// codegen:end consumer_spec family=rsi
```

**`cmd/writer/pipeline.go`**:
```go
// codegen:begin pipeline_entry family=rsi source=codegen/families/rsi.yaml
{ family: "rsi", ... },
// codegen:end pipeline_entry family=rsi
```

### 3.2 Integration manifest

`codegen/integrated.yaml` — YAML manifest tracking all governed slices with family, artifact, spec path, golden path, target file, marker string, date, and stage.

### 3.3 Verification script

`scripts/codegen-integrated-check.sh` — extracts marked regions from target files, normalizes them (same rules as `codegen/compare.go`), and compares against golden snapshots. Reports PASS/FAIL per slice.

### 3.4 Makefile target

`make codegen-integrated` — developer-facing command for integrated slice verification.

### 3.5 CI gate

Added `make codegen-integrated` step to the `codegen-golden` CI job, running after `codegen-check` and `codegen-test`.

---

## 4. Files Changed

| File | Change type | Description |
|------|-------------|-------------|
| `internal/adapters/nats/signal_registry.go` | Modified | Added `codegen:begin/end` markers around `WriterRSISignalConsumer()` |
| `cmd/writer/pipeline.go` | Modified | Added `codegen:begin/end` markers around RSI pipeline entry |
| `codegen/integrated.yaml` | New | Integration manifest tracking governed slices |
| `scripts/codegen-integrated-check.sh` | New | Verification script for integrated slice governance |
| `Makefile` | Modified | Added `codegen-integrated` target |
| `.github/workflows/ci.yml` | Modified | Added integrated slice check to codegen-golden CI job |
| `docs/architecture/first-generated-slice-integration.md` | New | Integration architecture document |
| `docs/architecture/generated-slice-01-runtime-participation-and-boundaries.md` | New | Runtime boundary document |
| `docs/stages/stage-s200-first-generated-slice-integration-report.md` | New | This report |

---

## 5. Verification Results

| Check | Result |
|-------|--------|
| `codegen check-all` (12 families × artifacts) | 12/12 PASS |
| `codegen-integrated` (RSI consumer_spec) | PASS |
| `codegen-integrated` (RSI pipeline_entry) | PASS |
| Writer compilation (`go build ./cmd/writer/`) | Clean |
| Writer unit tests (`go test ./...`) | PASS |

---

## 6. Limits and Frictions Observed

### 6.1 Frictions

| Friction | Severity | Notes |
|----------|----------|-------|
| Manual insertion required | Medium | Developer must copy generated output to correct location. No automated insertion yet. |
| macOS `head -n -1` incompatibility | Low | Fixed with `awk` equivalent. Linux/macOS portability must be tested. |
| Marker format requires exact match | Low | Scripted extraction depends on marker string. Typo = invisible governance gap. |
| Two CI gates instead of one | Low | `codegen-check` validates spec→golden; `codegen-integrated` validates golden→target. Could be unified in future. |

### 6.2 Limitations

1. **Only one family governed**: 5/6 families remain fully manual.
2. **Only A1 + A2 governed**: Mapper (A3), tests (A4), config (A5), smoke (A6) remain manual for RSI.
3. **No automated insertion**: Manual copy step is the primary toil source.
4. **Structural comparison only**: Byte-level differences that normalize identically are invisible to the checker.
5. **No Tier 2**: Read-path artifacts not in scope.

### 6.3 Risks accepted

| Risk | Acceptance rationale |
|------|---------------------|
| Single family doesn't prove scalability | S200 scope is proof-of-concept, not production rollout. Scalability is S201+. |
| Manual insertion may introduce subtle errors | Mitigated by CI gate + compilation + unit tests + smoke tests. |
| Marker deletion goes unnoticed outside CI | CI is the enforcement point. Local dev without CI has no guarantee. |

---

## 7. What Still Does NOT Enter the Generated Path

| Artifact | Reason | When it might |
|----------|--------|---------------|
| Mappers (A3) | Requires `domain.columns` spec extension | After first codegen-first family validates |
| Mapper tests (A4) | Depends on A3 | After A3 |
| Config entries (A5) | JSONC tooling not implemented | Dedicated stage |
| Smoke test phases (A6) | Shell template engine deferred | Dedicated stage |
| Other 5 families (A1+A2) | Deliberate single-family scope | S201+ per-family opt-in |
| Domain types | Permanently manual | Never |
| Migrations/DDL | Permanently manual | Never |
| Read-path artifacts | Tier 2 not authorized | After ≥2 codegen-first families validated |

---

## 8. Success Criteria Assessment

| Criterion | Status | Evidence |
|-----------|--------|----------|
| A generated slice participates in the real flow | Met | RSI A1+A2 are in target files with governance markers |
| The slice is auditable and governed | Met | Markers + manifest + CI gate |
| Relationship between generated and manual is clear | Met | Markers delimit exact boundaries; document maps every artifact |
| Base is ready for manual/generated coexistence hardening | Met | Pattern established; opt-in mechanism defined |
| "Magic codegen" risk is reduced | Met | Markers are visible; CI blocks drift; manifest is explicit |

---

## 9. Preparation Recommended for S201

### 9.1 Expand governance to second family

The next logical step is to add governance markers to a second family, validating that the `codegen-integrated-check` script and manifest scale. Recommended candidate: **candle (evidence)** — it exercises the evidence-layer naming exception, filling the gap RSI doesn't cover.

### 9.2 Automate the check script from manifest

Currently `codegen-integrated-check.sh` has hardcoded slice entries. S201 should read from `codegen/integrated.yaml` directly, so adding a new governed family requires only a manifest entry, not a script edit.

### 9.3 Evaluate automated insertion

With two governed families, the friction of manual insertion becomes measurable. S201 should assess whether marker-based automated insertion (regenerate → write directly to target) is justified or premature.

### 9.4 Unify CI gates

Consider merging `codegen-check` and `codegen-integrated` into a single `codegen-validate` target that runs all spec→golden→target verification in one pass.

### 9.5 Harden PR review checklist

Formalize a PR review checklist item: "If this PR modifies files listed in `codegen/integrated.yaml`, verify changes originated from the codegen pipeline."

---

## 10. Guard Rail Compliance

| Guard rail | Status |
|------------|--------|
| Did not open the first family entirely generated | Compliant — only A1+A2 governed |
| Did not expand to multiple slices | Compliant — RSI only |
| Did not mix integration with runtime redesign | Compliant — zero runtime change |
| Did not hide limitations | Compliant — frictions, limits, and risks documented |
| Documented what still has not entered the generated path | Compliant — section 7 |
