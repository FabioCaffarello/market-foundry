# Pre-Family-06 Mandatory Hardening Tranche

> Formal definition of the hardening iteration that must complete before the sixth analytical family can begin.
> This document is the binding scope reference for the hardening tranche.

## Strategic Context

After five successful family expansions plus one hardening tranche (S172), the Wave B pattern has delivered full vertical analytical coverage (L1–L6) with zero creative decisions across all expansions. However, Family 05 pushed the handler file to 615/620 lines — 5 lines from the declared hard ceiling. This is the first true structural blocker in the analytical layer's lifecycle.

The next iteration is **not** a sixth family. It is a mandatory hardening pass that resolves the handler ceiling before expansion continues.

## Why Hardening, Not Expansion

| Signal | Evidence |
|---|---|
| Handler at hard ceiling | 615/620 lines. Family 06 would add ~100 lines → ~715 lines, 95 past ceiling (S188 PF-1-ESC) |
| Limit/since/until duplication | 30 lines of identical parameter parsing repeated verbatim in all 6 handler methods (S188 T-1) |
| Codegen decision pending | Scope definition mandatory, but implementation not required for immediate unblock (S188 T-2) |

These are not aesthetic preferences. The handler file **cannot physically accommodate** a seventh method under the existing structure.

## Why This Tranche Is Small

The pre-Family-03 hardening tranche (S171/S172) addressed three separate structural items (struct DI, smoke extraction, naming). This tranche addresses **one root cause** with **one extraction**, because:

1. The ceiling is caused by parameter parsing duplication — not by method count, naming, or composition.
2. The struct DI (H-1), smoke helper (H-2), and naming (H-3) from the previous tranche remain healthy and have scaled without friction through three additional families.
3. Reader signature and codegen are real concerns but do not block Family 06 — they block Family 07+.

## Tranche Composition

The tranche consists of exactly **one hardening item**:

### H-5: Extract `parseAnalyticalParams()` Helper

**What it does:**
Extracts the repeated limit parsing (13 lines) and since/until parsing (17 lines) into a single `parseAnalyticalParams()` function that returns a struct with `Limit int`, `Since int64`, `Until int64`, and an error.

**Before (per method, ~30 lines):**
```go
limit := 50
if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
    parsed, err := strconv.Atoi(limitStr)
    if err != nil { ... }
    if parsed < 1 || parsed > 500 { ... }
    limit = parsed
}

var since, until int64
if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
    parsed, err := strconv.ParseInt(sinceStr, 10, 64)
    if err != nil { ... }
    since = parsed
}
if untilStr := r.URL.Query().Get("until"); untilStr != "" {
    parsed, err := strconv.ParseInt(untilStr, 10, 64)
    if err != nil { ... }
    until = parsed
}
```

**After (per method, ~5 lines):**
```go
params, prob := parseAnalyticalParams(r)
if prob != nil {
    writeProblemResponse(w, prob)
    return
}
```

**Line savings:** ~19 lines × 6 methods = ~114 lines eliminated. Handler drops from 615 → 501 lines.

**Runway gained:** At ~100 lines per family, the handler can absorb ~2 more families (to ~700 lines) before the ceiling returns — at which point codegen or split will be mandatory.

### Sequencing

```
Phase 1: H-5 — Extract parseAnalyticalParams()
  └─ Define analyticalParams struct (Limit int, Since int64, Until int64)
  └─ Extract parseAnalyticalParams(r *http.Request) (analyticalParams, *problem.Problem)
  └─ Update all 6 handler methods to use the helper
  └─ Verify: all existing tests pass unchanged
  └─ Verify: handler file ≤ 501 lines (down from 615)
```

No other phases. No dependencies on other changes.

## Tranche Boundaries

### In Scope

- H-5: `parseAnalyticalParams()` extraction.
- Test updates if required (existing tests should pass without changes since behavior is identical).
- Verification that handler line count drops below 500.

### Out of Scope (Non-Goals)

- **No new family expansion.** Family 06 is not opened.
- **No codegen implementation.** Codegen is the strategic solution but is not the minimum viable unblock.
- **No codegen scope definition.** That is a separate deliverable (recommended for a dedicated stage).
- **No handler file split.** Extraction addresses the root cause; split is unnecessary at 501 lines.
- **No reader signature refactor.** 10-param signature is tolerable for one more family.
- **No schema, migration, or ClickHouse changes.** Wrong layer entirely.
- **No writer changes.** Write path is stable.
- **No smoke test changes.** Smoke helper scales fine.
- **No new abstractions.** This is a pure mechanical extraction — no generics, no interfaces, no new patterns.

## Affected Files

| Item | Files Modified |
|---|---|
| H-5 | `internal/interfaces/http/handlers/analytical.go` |

One file. Zero new files. Zero files in other packages.

## Validation Criteria

| Criterion | Measurement |
|---|---|
| H-5 implemented | `parseAnalyticalParams` function exists and is called by all 6 handlers |
| Line count reduced | `wc -l analytical.go` ≤ 501 (down from 615) |
| No behavioral change | All existing handler tests pass without modification |
| No scope creep | Zero changes outside `analytical.go` |
| Build passes | `go build ./cmd/gateway/...` succeeds |
| All tests pass | `go test ./internal/interfaces/http/handlers/...` passes (zero test modifications) |

## Stop Conditions

The hardening tranche must halt and escalate if:

- The extraction requires changes to any file other than `analytical.go`.
- Any existing test fails after the extraction (signals behavioral divergence).
- The extraction introduces a new type or interface that other packages must import.
- The resulting handler exceeds 550 lines (extraction insufficient).

## What This Gate Unlocks

Once this gate passes, Family 06 may proceed following the Wave B Family Expansion Pattern v2. The expansion will:

- Add a 7th method to the handler (~100 lines → handler at ~601 lines, under the 620-line original ceiling).
- Use `parseAnalyticalParams()` to avoid duplicating 30 lines of parameter parsing.
- Continue the struct DI pattern without constructor changes.

## What This Gate Does NOT Unlock

- **Codegen.** Remains the strategic solution for Family 07+. Separate initiative.
- **Handler file split.** Not needed until handler approaches ~700 lines (~Family 08 without codegen).
- **Reader signature refactor.** Addressed by codegen when adopted.
- **Schema coherence tooling.** Still under threshold (6 tables, ~95 columns).
