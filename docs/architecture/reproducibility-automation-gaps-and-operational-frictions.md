# Reproducibility, Automation Gaps, and Operational Frictions

> Stage S351 — PRA-4 gap analysis artifact.
> Identifies concrete gaps between the current automation surface and reliable, unattended reproducibility.

## 1. Gap Inventory

### GAP-1: No Machine-Readable Smoke Output

**Current state**: All smoke scripts produce colored terminal output via lib.sh logging functions (pass/fail/info/warn). The only structured signal is the exit code (0 = all pass, non-zero = failures).

**Impact**: An automated runner cannot parse individual test results, extract timing data, or generate reports without screen-scraping.

**Minimum fix**: Emit a JSON summary file (e.g., `smoke-results.json`) at the end of each smoke run with scenario names, pass/fail status, and duration. Estimated effort: ~40 LOC change to lib.sh + ~5 LOC per smoke script for summary emission.

**Priority**: P2 — Required for CI integration, not for local reproducibility.

---

### GAP-2: No Aggregate Smoke Target

**Current state**: Nine canonical smoke targets exist independently. There is no `make smoke-all` or `make smoke-suite` that runs the appropriate subset in sequence.

**Impact**: An operator or CI system must know which smokes to run and in what order. The `make smoke-help` target documents this, but it requires human interpretation.

**Minimum fix**: Add a `make smoke-suite` target that runs the 5 core smokes in dependency order (smoke → smoke-multi → smoke-analytical → smoke-round-trip → smoke-activation), stopping on first failure. Estimated effort: ~15 lines in Makefile.

**Priority**: P2 — Ergonomic improvement for both operators and CI.

---

### GAP-3: No Failure Artifact Collection

**Current state**: When a smoke fails, the operator must manually inspect container logs, run `make diag`, and correlate timestamps. No automatic artifact capture occurs on failure.

**Impact**: CI failures are difficult to triage without SSH access to the runner. Local failures require manual follow-up steps.

**Minimum fix**: Add a `trap` handler in smoke scripts that runs `make diag` and captures `docker compose logs --tail=100` to a timestamped file on non-zero exit. Estimated effort: ~20 LOC in lib.sh.

**Priority**: P2 — High value for CI, moderate value for local development.

---

### GAP-4: Timeout Values Not Centralized

**Current state**: Three different timeout variables (`SMOKE_WAIT`, `FLUSH_WAIT`, `CANDLE_WAIT_MAX`) with different defaults across scripts. Some scripts use `SMOKE_WAIT`, others use `FLUSH_WAIT`, and the shared lib uses `CANDLE_WAIT_MAX`.

**Impact**: Operators must understand which variable applies to which smoke. CI environments with different performance profiles need to override multiple variables.

**Minimum fix**: Normalize to a single `SMOKE_TIMEOUT` variable with per-smoke defaults, keeping backward compatibility via fallback (`SMOKE_TIMEOUT="${SMOKE_TIMEOUT:-${FLUSH_WAIT:-120}}""`). Estimated effort: ~30 LOC across scripts.

**Priority**: P3 — Ergonomic friction, not a correctness issue.

---

### GAP-5: No Port Availability Pre-Check

**Current state**: `make up` starts compose services bound to specific ports (4222, 8080, 8123, 8222, 9000). If any port is in use, the error surfaces as a Docker compose failure mid-startup.

**Impact**: The failure message is a Docker error, not a helpful diagnostic. The operator must manually identify the conflicting process.

**Minimum fix**: Add a port-check step to `make bootstrap` or `make up` that tests key ports before starting compose. Estimated effort: ~15 LOC in a check script.

**Priority**: P3 — Nice-to-have for ergonomics.

---

### GAP-6: CLICKHOUSE_DSN Not Managed

**Current state**: `make test-clickhouse` requires `CLICKHOUSE_DSN` to be set, but this variable is not in `deploy/envs/local.env` and has no default in the Makefile.

**Impact**: Developers discovering ClickHouse integration tests for the first time must read test code or documentation to find the correct DSN format.

**Minimum fix**: Add `CLICKHOUSE_DSN` to `local.env` with the compose-local default (`clickhouse://default:clickhouse@127.0.0.1:9000/market_foundry`). Estimated effort: 1 line.

**Priority**: P3 — Minor friction.

---

### GAP-7: Venue Credentials Not Integrated into Automation

**Current state**: `deploy/configs/execute.env.example` documents required venue API credentials, but no smoke or deployment script references this file. Credentials must be manually set in the environment before running live venue paths.

**Impact**: The venue activation path (`smoke-activation`) works without real credentials (uses paper mode with simulated fills), but any future live-venue smoke would require manual credential setup.

**Minimum fix**: Document the credential flow in `smoke-help` and add a validation check in smoke scripts that use venue paths. Estimated effort: ~10 LOC.

**Priority**: P3 — Not blocking current smokes, relevant for future live-venue automation.

---

### GAP-8: No Idempotent "Reset and Rerun" Command

**Current state**: If a smoke fails mid-execution, the stack may be in a dirty state (e.g., activation gate left in non-default position, stale data in KV store). The operator must `make down && make up && make seed` to reset, then rerun the smoke.

**Impact**: Automated retry requires a full teardown/rebuild cycle. The `smoke-activation.sh` script has a trap-based gate restoration, but other smokes do not clean up after failure.

**Minimum fix**: Add `make reset` target that performs `down + up + seed` atomically, and ensure all smokes with state mutation have trap-based cleanup. Estimated effort: ~5 lines in Makefile + ~10 LOC per smoke with state mutation.

**Priority**: P2 — Important for CI retry strategies.

---

## 2. Friction Map

### 2.1 Low Friction (Works Today)

- Clone → bootstrap → live → smoke: **3 commands**, no undocumented steps
- Individual smoke selection: **1 command** with clear prerequisites
- Stack lifecycle (up/down/restart): **Fully scripted**
- Configuration seeding: **Fully scripted** with multi-symbol option
- Diagnostic capture: **Fully scripted** via `make diag`
- Architecture guard rails: **Fully scripted** via raccoon-cli

### 2.2 Medium Friction (Works with Operator Knowledge)

- Choosing the right smoke for a given change: Requires reading `make smoke-help` or governance docs
- Overriding timeouts for slow environments: Requires knowing which variable applies to which smoke
- Running ClickHouse integration tests: Requires knowing to set `CLICKHOUSE_DSN`
- Interpreting smoke failures: Requires manual log inspection and diagnostic correlation
- Retrying after failure: Requires knowing the reset sequence

### 2.3 High Friction (Not Yet Automatable)

- CI/CD integration: No machine-readable output, no artifact collection, no aggregate target
- Multi-environment deployment: No remote deployment support (out of scope)
- Credential-gated venue paths: Manual credential management
- Unattended operation monitoring: Gaps identified in S350 (Prometheus, alerting)

## 3. Trade-offs and Priorities

### 3.1 Priority Matrix

| Gap | Value for CI | Value for Local | Effort | Priority |
|-----|-------------|----------------|--------|----------|
| GAP-2: Aggregate smoke target | HIGH | MEDIUM | LOW | P2 |
| GAP-1: Machine-readable output | HIGH | LOW | MEDIUM | P2 |
| GAP-3: Failure artifact collection | HIGH | MEDIUM | LOW | P2 |
| GAP-8: Idempotent reset+rerun | HIGH | MEDIUM | LOW | P2 |
| GAP-4: Timeout normalization | MEDIUM | MEDIUM | LOW | P3 |
| GAP-5: Port pre-check | LOW | MEDIUM | LOW | P3 |
| GAP-6: CLICKHOUSE_DSN default | LOW | MEDIUM | TRIVIAL | P3 |
| GAP-7: Credential flow docs | LOW | LOW | TRIVIAL | P3 |

### 3.2 Recommended Sequencing

If automation gaps are addressed in a future stage:

1. **First**: GAP-2 + GAP-8 (aggregate target + reset) — unlocks "run all smokes reliably"
2. **Second**: GAP-3 + GAP-1 (artifacts + JSON output) — unlocks CI integration
3. **Third**: GAP-4 + GAP-5 + GAP-6 + GAP-7 (ergonomic polish) — reduces friction

### 3.3 What NOT to Do

- Do not build a CI pipeline in this assessment
- Do not create a remote deployment mechanism
- Do not add a container registry or image push workflow
- Do not build a smoke result dashboard
- Do not add Prometheus metrics (that is S350's identified gap, tracked separately)

## 4. Comparison with S350 Gaps

S350 identified monitoring/alertability gaps for **runtime operation**. This document identifies automation gaps for **deployment and validation**. The two are complementary:

| S350 Gap | S351 Gap | Relationship |
|----------|----------|-------------|
| No Prometheus export | No machine-readable smoke output | Both serve observability; different lifecycle phases |
| No push-based alerting | No failure artifact collection | Both serve incident response; different triggers |
| No consumer lag visibility | No aggregate smoke target | Both serve operational awareness; different scopes |

These gaps can be addressed independently. Neither blocks the other.
