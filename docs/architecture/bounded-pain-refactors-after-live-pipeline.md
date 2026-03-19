# Bounded Pain Refactors After Live Pipeline

> S116 — Targeted, small refactors justified by friction observed during S115 operational validation.

## Guiding Principle

Every refactor in this document has an explicit link to a friction point captured in the S115 friction matrix. No refactor was made for aesthetic improvement or speculative future needs.

---

## R1: Drift-Detect False Positive Suppression (F4)

**Pain observed:** The `naming-identity-drift` check in raccoon-cli scanned all of `internal/`, `cmd/`, and `deploy/` for the word "consumer" as a defunct service name. However, `internal/` legitimately uses "consumer" for NATS JetStream durable consumers (~260 matches in adapters, actors, and supporting code). This produced persistent warnings on every quality-gate run, reducing signal quality.

**Fix:** Split the scan in `scan_stale_references()` so that:
- "emulator" and "validator" are checked everywhere (no legitimate usage exists)
- "consumer" is only checked in `cmd/` and `deploy/` (where it would indicate old service binary or Docker image references)
- `internal/` is scanned only for emulator/validator patterns

**Files changed:**
- `tools/raccoon-cli/src/analyzers/drift_detect.rs` — scan logic restructured

**Impact:** Eliminates ~260 false-positive warnings per quality-gate run. Drift-detect now only flags actual stale references.

---

## R2: Stale Variable Name in Runtime Test (F4)

**Pain observed:** `validatorRecord` variable in `runtime_test.go` referenced the defunct "validator" service name. The test verifies that `RecordFromProjection` and `RecordFromIngestionProjection` produce identical runtime records — the variable name was misleading.

**Fix:** Renamed `validatorRecord` → `projectionRecord` and updated the error message.

**Files changed:**
- `internal/application/runtimecontracts/runtime_test.go`

**Impact:** Removes a confusing stale name from active test code. Prevents drift-detect from flagging this file.

---

## R3: AGENTS.md Prohibited Patterns Clarification (F4)

**Pain observed:** Line 65 of AGENTS.md listed "Validator/consumer/emulator services" as prohibited — using the bare old service names in a way that reads as if NATS consumers are prohibited.

**Fix:** Changed to "Old quality-service binaries (validator, consumer, emulator)" to make it clear that the prohibition is about the old service binaries, not about NATS consumer patterns.

**Files changed:**
- `AGENTS.md`

**Impact:** Reduces confusion for agents and developers reading the onboarding document.

---

## R4: Raccoon-CLI Test Fixture Modernization (F4)

**Pain observed:** The `parse_minimal_compose` test in `topology/compose.rs` used `quality-service/consumer:dev` as a test Docker image and `consumer` as the service name — direct references to the defunct quality-service architecture.

**Fix:** Changed to `market-foundry/ingest:dev` with service name `ingest`, reflecting the current architecture.

**Files changed:**
- `tools/raccoon-cli/src/analyzers/topology/compose.rs`

**Impact:** Test fixtures reflect reality. No drift-detect noise from test infrastructure.

---

## Validation

| Check | Result |
|-------|--------|
| `cargo test` (97 tests) | PASS — 97/97 |
| `go test ./internal/application/runtimecontracts/...` | PASS |
| All refactors linked to S115 friction | Yes — all linked to F4 |
| No horizontal refactoring introduced | Confirmed |
| No new abstractions introduced | Confirmed |
