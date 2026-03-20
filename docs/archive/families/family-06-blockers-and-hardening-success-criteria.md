# Family 06 Blockers and Hardening Success Criteria

> Explicit gate conditions that must be satisfied before Family 06 expansion begins.

## Binding Commitments

The following blockers originate from S188 (Family 05 E2E Validation and Ceiling Evidence) and are formalized in S189 (Pre-Family-06 Mandatory Hardening Tranche). They are **not suggestions**; they are pre-committed gate conditions.

## Blockers

### Blocker 1: Handler Parameter Extraction (H-5)

**Status:** Blocking
**Origin:** S188 T-1, PF-1-ESC (handler at 615/620 hard ceiling)
**Severity:** Critical

**Condition to clear:**
- `parseAnalyticalParams()` function exists in `analytical.go`.
- Returns `analyticalParams` struct containing `Limit int`, `Since int64`, `Until int64`.
- All 6 existing handler methods use `parseAnalyticalParams()` instead of inline parsing.
- Zero lines of limit/since/until parsing remain inline in any handler method.
- Handler file total line count is ≤501 (down from 615).
- All existing handler tests pass without modification (zero behavioral change).

**Verification:**
```bash
# Function must exist and be called 6 times:
grep -c 'parseAnalyticalParams' internal/interfaces/http/handlers/analytical.go
# Expected: 7+ (1 definition + 6 calls)

# No inline limit parsing should remain:
grep -c 'strconv.Atoi(limitStr)' internal/interfaces/http/handlers/analytical.go
# Expected: 1 (inside parseAnalyticalParams only)

# Line count must be under 500:
wc -l internal/interfaces/http/handlers/analytical.go
# Expected: < 500

# All tests must pass:
go test ./internal/interfaces/http/handlers/...
```

---

## Non-Blockers (Tracked but Not Gating)

### NB-1: Codegen Scope Definition

**Status:** High priority, not blocking
**Origin:** S188 T-2
**Rationale:** Handler extraction buys sufficient runway for Family 06. Codegen becomes mandatory at Family 07+ but is not required to unblock the next expansion.
**Recommendation:** Pursue codegen scope definition as a dedicated stage (S190 or S191) before Family 07.

### NB-2: Reader 10-Parameter Signature

**Status:** Monitoring, not blocking
**Origin:** S188 PF-7
**Rationale:** 11 parameters (Family 06) remains within Go function signature limits. 12+ (Family 07) is the practical limit.
**Recommendation:** Absorbed by codegen tranche.

### NB-3: Smoke Test Line Count

**Status:** Healthy, not blocking
**Origin:** S188 PF-3
**Current:** 651 lines. Growth ~30 lines per family.
**Rationale:** `validate_analytical_family()` helper absorbs additions mechanically. Restructuring recommended at Family 08+.

---

## Composite Gate Criteria

| Criterion | Measurement |
|---|---|
| H-5 implemented | `parseAnalyticalParams` exists and is called by all 6 handlers |
| Handler under ceiling | `wc -l analytical.go` ≤ 501 (down from 615) |
| Zero behavioral regression | All handler tests pass without modification |
| Build green | `go build ./cmd/gateway/...` succeeds |
| CI green | `.github/workflows/ci.yml` passes |
| No scope creep | Only `analytical.go` modified |
| No new frictions introduced | Hardening did not create new scaling constraints |

## Stop Conditions

The hardening tranche must halt and escalate if:

- H-5 extraction requires changes to files outside `analytical.go`.
- Any existing test fails (behavioral divergence signal).
- The extraction introduces cross-package dependencies.
- Handler line count remains above 550 after extraction.

## What This Gate Unlocks

Once this gate passes, Family 06 may proceed. The expansion will:

- Add a 7th handler method (~100 lines) to a handler at ~501 lines → ~601 lines (under the 620-line original ceiling).
- Use `parseAnalyticalParams()` for parameter parsing (no duplication).
- Continue struct DI pattern without constructor signature changes.
- Add a new ClickHouse reader, use case, and tests following the proven Wave B v2 pattern.

## What This Gate Does NOT Unlock

- **Codegen implementation.** Separate initiative, tracked independently.
- **Handler file split.** Not needed at ~565 lines.
- **Reader signature refactoring.** Deferred to codegen.
- **Schema coherence compile-time checks.** Below threshold (6 tables, ~95 columns).
- **Any deferred item without committed trigger.** The 9 DEF-U items remain unchanged.

## Family 06 Candidate Assessment

The Family 06 candidate is not selected as part of this gate. Selection depends on:

1. Whether remaining `venue_market_order` events exist and merit a dedicated family.
2. Whether the next expansion targets a different event type or source binary.
3. Input from the strategic roadmap (outside the analytical layer's scope).

This gate only certifies that the **pattern is ready** for a sixth expansion — it does not dictate **which** family that expansion covers.
