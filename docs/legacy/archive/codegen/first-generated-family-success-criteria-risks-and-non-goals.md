# First Generated Family — Success Criteria, Risks, and Non-Goals

> Stage: S202 — First Codegen-First Family Definition
> Family: EMA (signal layer)
> Date: 2026-03-20

## Success Criteria

Each criterion must be satisfied for S203 to be considered complete.

### SC-1: Spec Validates Clean

The `ema.yaml` spec passes both per-spec validation and cross-spec uniqueness checks.

**Verification:** `cd codegen && go run . validate codegen/families/ema.yaml && go run . validate-all`

### SC-2: Golden Snapshots Match Generation

Both `consumer_spec.go.golden` and `pipeline_entry.go.golden` are produced by the codegen engine and pass structural comparison.

**Verification:** `cd codegen && go run . check-all` — all families PASS including EMA.

### SC-3: Code Compiles

After inserting generated fragments into target files via `codegen:begin`/`codegen:end` markers, the entire writer module and NATS adapter module compile without errors.

**Verification:** `go build ./cmd/writer/... && go build ./internal/adapters/nats/...`

### SC-4: Existing Tests Pass

No regression in existing unit tests. The addition of EMA does not break RSI or any other family.

**Verification:** `go test ./cmd/writer/... && go test ./internal/adapters/nats/... && go test ./codegen/...`

### SC-5: Integrated Check Passes

The CI drift detection script validates both RSI and EMA governed slices against their golden snapshots.

**Verification:** `scripts/codegen-integrated-check.sh` exits 0.

### SC-6: Pipeline Activates With Config

When `"ema"` is added to `signal_families` in `writer.jsonc`, the writer service starts the EMA pipeline alongside existing families.

**Verification:** Smoke test observes EMA consumer and inserter actors in health output.

### SC-7: No Manual Edits to Generated Code

All code inside `codegen:begin`/`codegen:end` markers was placed there by the codegen workflow, not by hand. No post-generation fixups were needed.

**Verification:** Code review confirms generated fragments match golden snapshots byte-for-byte.

### SC-8: Manifest Updated

`codegen/integrated.yaml` contains two new entries for EMA (consumer_spec + pipeline_entry) with correct metadata.

**Verification:** Manifest review during PR.

## Risks

### R1: Fragment Insertion Error (Medium)

**Description:** Manual placement of generated fragments into target files may introduce errors — wrong location, missing marker, broken surrounding code.

**Likelihood:** Medium — the target files (`signal_registry.go`, `pipeline.go`) already contain governed RSI fragments, providing placement reference.

**Impact:** Compilation failure or runtime misbehavior.

**Mitigation:**
- RSI markers serve as structural reference for EMA marker placement
- CI compilation gate catches structural errors
- Integrated check script validates marker content
- Code review specifically verifies marker placement

### R2: Cross-Family Interference (Low)

**Description:** Adding a second signal family pipeline entry could interfere with the existing RSI pipeline entry in `pipeline.go`.

**Likelihood:** Low — pipeline entries are independent struct literals in a slice; no shared mutable state.

**Impact:** RSI regression or EMA misconfiguration.

**Mitigation:**
- Existing RSI tests remain unchanged and must pass
- Each pipeline entry is self-contained
- Smoke test covers both RSI and EMA independently

### R3: Config Omission (Low)

**Description:** Forgetting to add `"ema"` to `signal_families` in config would make the pipeline technically correct but never activated.

**Likelihood:** Low — explicit in the manual checklist.

**Impact:** Silent non-activation; no data flows.

**Mitigation:**
- Smoke test expects EMA pipeline to activate
- Health endpoint lists active families
- Explicit checklist item in S203 implementation

### R4: Overconfidence Extrapolation (Medium)

**Description:** Success with EMA (same-layer, shared mapper, shared table) could be misinterpreted as proof that codegen works for cross-layer families or families requiring new mappers.

**Likelihood:** Medium — natural tendency to extrapolate from success.

**Impact:** Premature expansion into harder families without adequate preparation.

**Mitigation:**
- This document explicitly states what EMA proves and does NOT prove
- Non-goals section below explicitly defers cross-layer and new-mapper families
- Future family selection requires its own assessment stage

### R5: NATS Durable Consumer Collision (Low)

**Description:** If the durable consumer name `writer-signal-ema` conflicts with an existing consumer in the NATS cluster, subscription will fail.

**Likelihood:** Very low — cross-spec validation ensures uniqueness within codegen families, and no external system uses this naming pattern.

**Impact:** Consumer startup failure.

**Mitigation:**
- Cross-spec uniqueness validation
- Writer startup logs consumer creation failures
- Smoke test catches subscription failures

## Non-Goals

### NG-1: This Is Not Full Family Generation

EMA generates A1 + A2 only. Eight or more artifacts remain manual or reused. This iteration does not aim to generate mappers, tests, config, smoke, DDL, readers, handlers, or routes.

### NG-2: This Does Not Authorize Multi-Family Iteration

EMA is a single-family iteration. Success does not authorize generating multiple families in a single stage. Each family requires its own assessment.

### NG-3: This Does Not Prove Cross-Layer Generation

EMA shares the signal layer with RSI. Success does not prove the codegen model works for a family in a layer without an existing governed family (e.g., first codegen-first evidence family).

### NG-4: This Does Not Prove New Mapper Generation

EMA reuses `mapSignalRow`. Success does not prove the codegen model works for families requiring a new mapper function. Mapper generation (A3) remains explicitly deferred.

### NG-5: This Does Not Authorize Template Changes

Templates are frozen per S193. EMA must work with existing templates. If it doesn't, the family selection is wrong, not the templates.

### NG-6: This Does Not Prove New Table Generation

EMA writes to the existing `signals` table. Success does not prove the model works for families requiring new DDL or migrations.

### NG-7: This Is Not Performance Benchmarking

Time savings from codegen are a secondary observation, not a success criterion. The primary goal is correctness and process validation.

### NG-8: This Does Not Establish Automatic Integration

Fragment insertion remains manual in S203. Automation of the spec → target insertion workflow is a future enhancement, not an S203 deliverable.

## Failure Modes and Response

### FM-1: Golden Comparison Fails

**Response:** Debug spec YAML and derived field computation. Do not manually edit golden snapshots. Fix the spec or report a codegen engine bug.

### FM-2: Code Does Not Compile After Insertion

**Response:** Verify marker placement. Check that EMA fragments don't break syntax of surrounding code. Compare with RSI fragment placement pattern.

### FM-3: Existing Tests Regress

**Response:** Investigate whether the regression is caused by the EMA addition or by a pre-existing issue. EMA fragments should not affect existing test behavior. If they do, this reveals a coupling problem that must be addressed before proceeding.

### FM-4: Smoke Test Fails for EMA

**Response:** Verify config entry, NATS subject routing, and pipeline activation. Check writer logs for consumer startup errors. The generated artifacts are structurally correct if golden comparison passes — smoke failure likely indicates config or infrastructure issues.

### FM-5: Smoke Test Fails for RSI (Regression)

**Response:** This is the most serious failure mode. If adding EMA breaks RSI, there is a pipeline-level coupling that must be understood before proceeding with any further family expansion. Halt and investigate.

## Open Questions for S203

1. **Event producer:** Does an EMA signal producer exist in the derive service, or does S203 need to coordinate with a producer implementation? If no producer exists, smoke testing will require a synthetic event injection.
2. **Config gating:** Should EMA be enabled by default in the development config, or gated behind an explicit opt-in?
3. **Marker ordering:** Should EMA markers appear immediately after RSI markers in the target files, or at the end of the file?
