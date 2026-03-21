# Stage C14 Report: Smoke UX And Proof Execution Ergonomics

## 1. Executive Summary

Stage C14 improved the practical operator UX of the repository proof surface
without changing functional domain behavior.

The main result is a more explicit and predictable execution experience:

- proof selection is easier via `make smoke-help`
- setup assumptions are stated inside the harnesses, not only in comments
- common wait and URL overrides are available through the public surface
- hard failures now point the operator to the next diagnosis commands
- summaries are more useful for repeat execution and handoff

The canonical proof surface remains `make smoke*`.

## 2. Main Frictions Found

### 2.1 Discoverability friction

The repository had the right proof entrypoints, but no short operator-facing way
to answer:

- which smoke should I run now
- which setup is required first
- which logs should I inspect when it fails

Operators had to infer this from scattered docs, comments, or stage history.

### 2.2 Implicit preconditions

The smokes assumed stack bring-up and seeding, but the reminders were uneven
across scripts and Make/docs.

This made common failures feel ambiguous:

- gateway not ready
- stack not fully running
- wrong seed path for the chosen proof

### 2.3 Weak failure diagnosis at the first point of failure

Several harnesses correctly failed, but the failure text did not always tell the
operator what to do next.

This increased time-to-diagnosis for:

- readiness failures
- OS-process/service preflight failures
- analytical path preflight failures

### 2.4 Public overrides were inconsistent

Wait overrides existed, but not consistently through the public surface.
Routine cases still pushed operators toward direct script invocation.

## 3. UX Improvements Applied

### 3.1 Make surface

- added `make smoke-help`
- exposed `BASE_URL` and `SMOKE_WAIT` in Make help as supported smoke inputs

### 3.2 Shared harness ergonomics

- added reusable smoke banner output in `scripts/utils/lib.sh`
- added reusable diagnosis hints in `scripts/utils/lib.sh`
- added a small HTTP status helper to reduce unclear curl failure output

### 3.3 Smoke script improvements

- `scripts/smoke-first-slice.sh`
  - explicit setup hint
  - public `SMOKE_WAIT` support
  - clearer readiness failures
  - more useful success summary
- `scripts/smoke-multi-symbol.sh`
  - explicit setup hint
  - public `SMOKE_WAIT` support
  - better gateway failure text
  - better end summary and diagnosis guidance
- `scripts/smoke-analytical-e2e.sh`
  - explicit setup hint
  - `SMOKE_WAIT` support while preserving `FLUSH_WAIT`
  - better ClickHouse/writer/gateway abort guidance
  - clearer failed-run diagnosis guidance
- `scripts/smoke-os-process-operational.sh`
  - explicit setup hint
  - `SMOKE_WAIT` support while preserving `FLUSH_WAIT`
  - better service-preflight failure guidance
- `scripts/smoke-restart-recovery.sh`
  - explicit setup hint
  - `SMOKE_WAIT` support while preserving `FLUSH_WAIT`
  - better preflight and failure-summary diagnosis guidance

### 3.4 Documentation

Added:

- `docs/operations/smoke-ux-and-proof-execution-ergonomics.md`
- `docs/operations/proof-execution-user-flows-and-failure-diagnosis.md`

Updated:

- `README.md`
- `DEVELOPMENT.md`
- `docs/operations/README.md`
- `docs/operations/makefile-targets-reference-and-conventions.md`
- `docs/operations/scripts-catalog-and-usage-guide.md`
- `docs/operations/operational-proof-entrypoints-and-ownership.md`

## 4. New User Flows

### 4.1 Choose a proof quickly

```bash
make smoke-help
```

### 4.2 Run a proof with a longer warm-up budget without dropping to raw scripts

```bash
SMOKE_WAIT=180 make smoke
SMOKE_WAIT=240 make smoke-analytical
```

### 4.3 Follow the first-line diagnosis path consistently

```bash
make ps
make logs SERVICE=gateway
make diag
```

Then move to the nearest service:

- `derive`
- `store`
- `writer`
- `execute`

## 5. Improved Failure Examples

### Before

- readiness failure reported only the failing status code
- service-preflight failure often collapsed to “run make up && make seed first”
- analytical aborts identified the blocker but not the next diagnosis command

### After

- first hard failures name the expected setup and next commands
- analytical preflight failures point directly to `gateway` or `writer` logs
- operational and restart smokes print diagnosis hints after abort/failure summary
- baseline and multi-symbol smokes print runtime context in the summary

## 6. Guard Rails Preserved

- no new execution platform was introduced
- no smoke semantics were broadened or hidden
- no functional domain path was modified
- failures remain concrete and specific

## 7. Recommended Preparation For C15

The next safe refinement frontier is operational signal density, not new runtime
surface area.

Recommended preparation:

- observe whether operators still need repeated manual triage for warm-up-heavy proofs
- identify the highest-noise warning families in `smoke-multi` and `smoke-analytical`
- tighten any remaining summary sections that still read like implementation logs instead of operator output
- only extract further shared harness helpers if repetition becomes materially costly
