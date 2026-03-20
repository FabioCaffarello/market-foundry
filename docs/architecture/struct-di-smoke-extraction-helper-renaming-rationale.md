# Struct DI, Smoke Extraction, and Helper Renaming — Architectural Rationale

> Why each mandatory hardening item is a structural necessity, not cosmetic cleanup.

## Framing

The three items in the mandatory hardening tranche share a common trait: each addresses a **scaling constraint** that was acceptable at 1–2 families but becomes a liability at 3+. The decision to harden now — before the third family — is driven by documented thresholds, not taste.

---

## H-1: Struct-Based Dependency Injection

### Current State

```go
func NewAnalyticalWebHandler(
    getCandleHistory getAnalyticalCandleHistoryUseCase,
    getSignalHistory getAnalyticalSignalHistoryUseCase,
    getDecisionHistory getAnalyticalDecisionHistoryUseCase,
    logger *slog.Logger,
) *AnalyticalWebHandler
```

Four positional arguments. A fifth (Family 03) would make call sites fragile: argument reordering is silent, type confusion between use-case interfaces is possible, and every new family forces a signature change that ripples through compose, routes, and tests.

### Why This Is Structural

| Concern | Positional Args | Struct DI |
|---|---|---|
| **Argument ordering errors** | Silent at compile time if two interfaces share the same shape | Named fields eliminate ordering risk |
| **Nil-safety** | Must check each arg positionally | Struct fields have zero-value semantics; `HasAny()` pattern already exists in routes |
| **Signature stability** | Every family changes the function signature | Struct grows by field addition; signature stays `func New(deps Deps)` |
| **Test construction** | Tests must pass all args, even irrelevant ones | Tests set only the fields they need |
| **Grep/review surface** | Hard to see what's injected at a glance | Struct definition is a single point of truth |

### Target State

```go
type AnalyticalHandlerDeps struct {
    CandleHistory   getAnalyticalCandleHistoryUseCase
    SignalHistory    getAnalyticalSignalHistoryUseCase
    DecisionHistory getAnalyticalDecisionHistoryUseCase
    Logger          *slog.Logger
}

func NewAnalyticalWebHandler(deps AnalyticalHandlerDeps) *AnalyticalWebHandler
```

### Responsibility Addressed

**Composition and extensibility.** The handler's constructor is a composition boundary. Struct DI makes this boundary scale linearly with family count without signature instability.

### Pain Points Resolved

- S167 D-2: constructor argument accumulation (Medium severity)
- S170 PF-1: positional args at 4, threshold exceeded (Medium severity)
- Wave B Pattern Hardening A-5: hard requirement at family 3

---

## H-2: Smoke Test Function Extraction

### Current State

`scripts/smoke-analytical-e2e.sh` is 614 lines. Each family adds ~80 lines of validation logic that follows an identical structure:

1. Query the analytical endpoint with valid params.
2. Validate HTTP status, JSON structure, row count.
3. Query with invalid params, validate 400 responses.

This logic is repeated three times (candles, signals, decisions) with only the endpoint path, expected fields, and error conditions varying.

### Why This Is Structural

| Concern | Inline Repetition | Extracted Function |
|---|---|---|
| **Family addition cost** | Copy ~80 lines, manually adjust paths/fields, risk drift | Call `validate_analytical_family()` with parameters |
| **Consistency enforcement** | Each copy can diverge silently | Single function enforces identical validation contract |
| **Debugging surface** | 600+ lines to scan for a smoke failure | Function isolates family-specific failures |
| **Maintenance burden** | Changing validation logic requires N edits (one per family) | Single edit propagates to all families |

### Target State

```bash
validate_analytical_family() {
    local family="$1"
    local endpoint="$2"
    local expected_fields="$3"
    local required_params="$4"
    # ... standardized validation phases
}

# Usage:
validate_analytical_family "candles" "/analytical/evidence/candles" "..." ""
validate_analytical_family "signals" "/analytical/signal/history" "..." "type"
validate_analytical_family "decisions" "/analytical/decision/history" "..." "type"
```

### Responsibility Addressed

**Operability and repeatability.** The smoke test is the primary operational validation tool for the analytical pipeline. Its structure must support reliable, low-friction expansion.

### Pain Points Resolved

- S167 D-3: smoke script linear growth (Medium severity)
- S170 PF-3: approaching 615 lines, threshold exceeded (Medium severity)
- Wave B Pattern Hardening A-4: hard requirement at family 3

---

## H-3: Helper Renaming (`parseEvidenceKeyParams` → `parseAnalyticalKeyParams`)

### Current State

```go
func parseEvidenceKeyParams(r *http.Request) (source, symbol, timeframe string, prob *problem.Problem)
```

This function extracts `source`, `symbol`, and `timeframe` from query parameters. It was originally written for the evidence/candles family. It is now called by **three** handlers: candles, signals, and decisions. The name `parseEvidenceKeyParams` implies it is specific to the evidence domain, but it is actually a universal analytical key parser.

### Why This Is Structural

| Concern | Misleading Name | Accurate Name |
|---|---|---|
| **Discoverability** | A developer searching for "analytical key parsing" won't find it | Name matches the actual abstraction |
| **Boundary clarity** | Implies evidence-specific logic; reader may hesitate to reuse | Name signals universal applicability |
| **Code review friction** | Reviewers must mentally translate "evidence" → "analytical" | Name is self-documenting |
| **Grep accuracy** | `grep evidence` returns false positives from unrelated contexts | `grep analytical` scopes correctly |

### Target State

```go
func parseAnalyticalKeyParams(r *http.Request) (source, symbol, timeframe string, prob *problem.Problem)
```

Pure rename. Zero behavioral change. Zero signature change.

### Responsibility Addressed

**Semantic clarity and boundary hygiene.** The function name is a contract signal. When the name lies about the function's scope, it creates cognitive friction at every call site and review cycle.

### Pain Points Resolved

- S167 D-1: naming residue from original evidence scope (Low-Medium severity)
- S170 PF-2: misleading name across 3 families (Low-Medium severity)
- Wave B Pattern Hardening A-1: committed for family 3

---

## Common Thread

All three items follow the same pattern:

1. **Acceptable at small scale** — each was a reasonable shortcut when only 1–2 families existed.
2. **Documented threshold** — each has a defined trigger point (family 3) established in S167 and confirmed in S170.
3. **Structural, not cosmetic** — each addresses composition, operability, or semantic clarity at a boundary that scales with family count.
4. **Narrow blast radius** — each touches a well-defined set of files with predictable, testable changes.

The hardening is not about making the code "cleaner." It is about ensuring the pattern remains **mechanically reliable** as it scales beyond the initial proof-of-concept families.
