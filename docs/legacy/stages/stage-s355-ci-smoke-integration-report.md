# S355: CI Smoke Integration And Reproducibility Hardening

## Summary

S355 integrates operational smoke verification into CI and hardens
reproducibility by eliminating inline readiness polling, adding missing
CI coverage for stackless smokes and repository governance checks, and
providing local preflight gates for developers.

**Scope:** CI smoke integration (OF-2 from the operational foundation charter).

**Key result:** the CI pipeline goes from 5 jobs to 7 jobs, covering stackless
smoke, repository consistency, and quality gate enforcement — surfaces that
were previously manual-only.

## Delivered Changes

### New CI Jobs

| Job | What it covers | Infrastructure |
|---|---|---|
| `smoke-composed` | S330 composed pipeline: supervisor composition, venue path, error classification, regression gate | none (Go tests only) |
| `repository-checks` | 21+ repository consistency invariants + raccoon-cli quality gate (CI profile) | Rust toolchain (cached) |

### Hardened Existing CI

- **`smoke-analytical`**: replaced inline ClickHouse/gateway readiness polling
  with shared `scripts/ci-wait-ready.sh` — single reusable script with
  configurable timeout, structured output, and `--skip-clickhouse` option.

### New Makefile Targets

| Target | Purpose |
|---|---|
| `make ci-smoke` | CI-safe stackless smoke suite (currently: `smoke-composed`) |
| `make ci-preflight` | Local pre-push gate: tests + consistency + quality gate + stackless smoke |
| `make ci-wait-ready` | Infrastructure readiness polling for stack-dependent smokes |

### New Scripts

- **`scripts/ci-wait-ready.sh`**: reusable readiness polling for ClickHouse and
  gateway with `--timeout` and `--skip-clickhouse` flags. Sources `lib.sh` for
  consistent logging and error handling.

### Updated Governance

- **`docs/operations/smoke-and-operational-harness-governance.md`**: added
  section 1b defining `make ci-*` as the CI integration surface. Updated entry
  classification table to include `make ci-*` as canonical surface.

### Architecture Documents

- **`docs/architecture/ci-smoke-integration-and-reproducibility-hardening.md`**:
  architecture and rationale for CI integration — job taxonomy, stackless vs
  stack-dependent design axis, readiness polling, local preflight model.

- **`docs/architecture/smoke-ci-shape-pass-fail-contract-and-operational-frictions.md`**:
  per-smoke shape reference — pass/fail contracts, time budgets, prerequisites,
  friction inventory, and future CI integration candidates.

## Files Changed

| File | Change |
|---|---|
| `.github/workflows/ci.yml` | Added `smoke-composed` and `repository-checks` jobs; replaced inline readiness in `smoke-analytical` with `ci-wait-ready.sh` |
| `Makefile` | Added `ci-smoke`, `ci-preflight`, `ci-wait-ready` targets; updated `.PHONY` and `smoke-help` |
| `scripts/ci-wait-ready.sh` | New: reusable infrastructure readiness polling |
| `docs/operations/smoke-and-operational-harness-governance.md` | Added CI target governance section |
| `docs/architecture/ci-smoke-integration-and-reproducibility-hardening.md` | New: integration architecture |
| `docs/architecture/smoke-ci-shape-pass-fail-contract-and-operational-frictions.md` | New: shape/contract reference |
| `docs/stages/stage-s355-ci-smoke-integration-report.md` | This report |

## Evidence

### CI Coverage Before vs After

| Verification surface | Before S355 | After S355 |
|---|---|---|
| Unit tests | CI | CI |
| Codegen golden | CI | CI |
| Behavioral scenarios | CI | CI |
| Integration tests (NATS) | CI | CI |
| Smoke analytical (compose) | CI | CI (hardened readiness) |
| Smoke composed (stackless) | local only | CI |
| Repository consistency checks | local only | CI |
| Quality gate | local only | CI |
| Local preflight gate | did not exist | `make ci-preflight` |

### Reproducibility Improvements

1. **Readiness polling centralized**: `ci-wait-ready.sh` replaces 3 inline
   shell blocks (ClickHouse wait + gateway wait + error handling) in ci.yml
   with a single script call.

2. **Consistent pass/fail signaling**: all CI-integrated smokes use `lib.sh`
   logging (`[PASS]`/`[FAIL]`) with structured error tracking.

3. **Raccoon-cli caching**: `repository-checks` job caches the Rust build
   artifact, reducing rebuild from ~60s to seconds on subsequent runs.

4. **Developer preflight**: `make ci-preflight` lets developers run the
   CI-equivalent check locally before pushing, catching issues earlier.

## Limits And Remaining Gaps

| Limit | Rationale |
|---|---|
| Most smokes remain local-only | They require live Binance WS or credentials that CI cannot provide |
| 120s flush wait is still a fixed sleep | Converting to polling requires writer status endpoint (future work) |
| No nightly/scheduled CI jobs | `smoke-operational` and `smoke-restart-recovery` are future CI candidates as optional/nightly |
| No CI notification integration | Uses GitHub Actions default; Slack/PagerDuty integration is out of scope |
| Error log scan remains advisory | By design — startup transients should not block CI |

## Preparation For S356

S356 can build on S355 by:

1. Adding `smoke-operational` and `smoke-restart-recovery` as optional nightly
   CI jobs (compose-only, no live WS dependency).
2. Converting the 120s flush wait to polled readiness if a writer status
   endpoint becomes available.
3. Adding CI timing metrics to track job duration trends.
4. Evaluating whether `make ci-preflight` should be enforced via pre-push hooks.
