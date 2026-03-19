# Stage S120 — Minimal Controlled Capability Implementation Report

> Status: Complete | Date: 2025-03-19

## 1. Executive Summary

Stage S120 implements CC-01 (Multi-Symbol Live Monitoring) as defined in S119.

**Result:** The capability was implemented with **zero application code changes**. Only operational tooling (3 files modified) and documentation (3 files created) were needed.

This outcome validates the architecture's core design thesis: multi-symbol operation is a config-driven scaling dimension, not a code change.

## 2. Capability Implemented

| Dimension | Value |
|-----------|-------|
| **Capability** | Multi-Symbol Live Monitoring (CC-01) |
| **Symbols** | btcusdt + ethusdt (extensible to N) |
| **Application code changes** | Zero |
| **Script changes** | 3 files (~62 lines) |
| **New documentation** | 3 files |
| **New Makefile targets** | `live-multi`, `live-multi-check` |

### What the Operator Can Now Do

```bash
make live-multi          # Full: build → start → seed (2 symbols) → validate
make live-multi-check    # Validate running multi-symbol stack
make smoke-multi         # Comprehensive 22-step E2E validation (2 symbols × 2 timeframes)
```

## 3. Files Changed

### 3.1 Modified

| File | Change Summary |
|------|---------------|
| `scripts/live-pipeline-activate.sh` | Added `--multi-symbol` flag. Phase 4 seeds with `--multi-symbol`. Phase 6 validates all domain endpoints per symbol. Phase 7 waits for candle materialization per symbol. Summary shows active mode. |
| `scripts/smoke-multi-symbol.sh` | Made `SYMBOLS` and `TIMEFRAMES` configurable via `SMOKE_SYMBOLS` and `SMOKE_TIMEFRAMES` env vars. Defaults unchanged (backward compatible). |
| `Makefile` | Added `live-multi` and `live-multi-check` targets. Updated `.PHONY` and help text. |

### 3.2 Created

| File | Purpose |
|------|---------|
| `docs/architecture/controlled-capability-01-implementation-notes.md` | Design decisions, simplifications, what the implementation proves |
| `docs/architecture/controlled-capability-01-runtime-activation-and-query-surface.md` | Activation flow, runtime wiring, query surface reference, operational procedures |
| `docs/stages/stage-s120-minimal-controlled-capability-implementation-report.md` | This report |

### 3.3 Not Changed (By Design)

Zero changes to:
- Go source code (any package)
- NATS stream/consumer definitions
- Docker compose topology
- Config validation logic
- Actor hierarchy or engine
- KV projection actors
- Gateway route handlers
- raccoon-cli Rust code

## 4. Simplifications Adopted

| # | Simplification | Rationale |
|---|---------------|-----------|
| S1 | Same 90s candle wait timeout for all symbols | ethusdt has sufficient trade volume for 60s candles. Tuning per-symbol timeout adds complexity without value. |
| S2 | Single config activation (not incremental) | Activating both symbols in one config lifecycle is simpler and tests the multi-binding path directly. |
| S3 | Aggregate tracker display (not per-symbol) | `/statusz` trackers don't partition by symbol. Visual inspection with 2 symbols is sufficient. |
| S4 | No new raccoon-cli scenario | The bash smoke-multi-symbol.sh provides 22-step comprehensive validation. Adding a Rust scenario for the same coverage would be duplicative. |
| S5 | No correlation ID injection | Explicitly deferred per S119 scope definition. Will be evaluated in S121 based on live friction evidence. |

## 5. Architecture Validation

### What This Proves

The zero-code-change implementation validates 5 architectural properties:

1. **Config-driven horizontal scaling** — Adding symbols is a config operation, not a code deployment
2. **Subject-based event partitioning** — NATS subjects naturally partition by symbol
3. **Composite KV key design** — Store projections address data by source+symbol+timeframe
4. **Parameterized query surfaces** — Gateway endpoints serve any symbol via `?symbol=` parameter
5. **Per-key actor state isolation** — Actors maintain independent state per source+symbol+timeframe without cross-contamination

### What Remains Unproven (Deferred to S121)

- Sustained operation under dual-symbol load (30+ minutes)
- Memory scaling linearity
- Cross-runtime debugging experience without correlation IDs
- Resource contention under doubled event throughput

These are validation concerns, not implementation concerns. They belong to S121 (operational validation).

## 6. Limits and Remaining Gaps

| # | Gap | Severity | When to Address |
|---|-----|----------|----------------|
| G1 | No endpoint to list active symbols | Low | When operator friction is confirmed during live validation |
| G2 | No per-symbol `/statusz` breakdown | Low | When debugging requires per-symbol tracker isolation |
| G3 | RSI warm-up period (~15 min for new symbol) | Known | By design — documented in implementation notes |
| G4 | Correlation ID absent in slog | Medium | S121 if confirmed blocking during multi-symbol debugging |
| G5 | Kill switch is global (halts all symbols) | Known | By design for CC-01. Per-symbol control is a separate capability. |

## 7. Guard Rail Compliance

| Guard Rail | Status |
|-----------|--------|
| Scope not amplified beyond S119 definition | **Compliant** — zero application code changes |
| Not a feature wave | **Compliant** — single capability, config-only |
| No new abstractions without pressure | **Compliant** — reused all existing patterns |
| No boundary violations | **Compliant** — no cross-layer changes |
| Simplifications documented | **Compliant** — 5 simplifications listed above |

## 8. Preparation for S121

S121 should be the **operational validation** stage for CC-01. Recommended scope:

### 8.1 What S121 Should Validate

1. **Activation criteria** (A1-A3 from S119 success criteria)
   - Config with 2 bindings activates without error
   - Both bindings appear in active config
   - Ingest discovers both without restart

2. **Pipeline flow criteria** (P1-P8)
   - All 7 pipeline stages produce data for both symbols
   - Full chain latency comparable to single-symbol baseline

3. **Stability criteria** (S1-S4)
   - No crashes for 30+ minutes
   - No domain errors
   - Memory scales linearly
   - Zero data loss in event chain

4. **Automation criteria** (T1-T3)
   - `make smoke-multi` passes
   - `make test` passes
   - Quality gate passes

### 8.2 How to Execute S121

```bash
# Step 1: Activate multi-symbol
make live-multi

# Step 2: Wait 15+ minutes for RSI warm-up

# Step 3: Run comprehensive smoke
make smoke-multi

# Step 4: Monitor resource usage
docker stats --no-stream  # at 10-min mark
docker stats --no-stream  # at 30-min mark

# Step 5: Check for domain errors
docker compose -f deploy/compose/docker-compose.yaml logs | grep -i error

# Step 6: Validate stability
make live-multi-check
```

### 8.3 Friction Capture

During S121, capture any friction using the protocol defined in S119:

```markdown
### Friction: [short name]
- **Severity:** Low / Medium / High
- **Evidence:** [what happened]
- **Impact:** [what it prevented or made harder]
- **Recommendation:** Fix in S122 / Defer / Investigate
```

### 8.4 Expected Outcomes

- **Likely smooth:** Config activation, pipeline flow, candle materialization
- **Likely friction:** Cross-runtime debugging without correlation IDs (F1)
- **Possible surprise:** Per-symbol tracker isolation needs, memory growth patterns

## 9. Verdict

S120 is **complete**. CC-01 is implemented and ready for live operational validation.

The implementation confirms the architecture's horizontal scaling design. The next step (S121) is to run it live, validate the success criteria from S119, and capture friction for future improvements.
