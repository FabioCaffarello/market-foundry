# CC-02 Extensibility Frictions and Findings

> Evidence-based capture of extensibility friction points revealed by the EMA Crossover (CC-02) signal family implementation.

## 1. Context

CC-02 introduced `ema_crossover` as the second signal family alongside RSI. Its primary utility is **not** the signal itself, but revealing whether the monorepo absorbs a new family with low friction. This document captures the real friction points with honest classification.

## 2. Friction Inventory

### 2.1 Actor Boilerplate — CF-08 Confirmed

**Classification: Structural debt (tolerable at N=2, trigger at N=3)**

`ema_crossover_signal_sampler_actor.go` (97 lines) is ~95% identical to the RSI actor. The diff between them reduces to:
- Constructor name
- Sampler type instantiated
- Actor name string

No generic `SignalSamplerActor[T]` exists. Each new family requires copying ~97 lines of identical lifecycle, message handling, correlation ID propagation, and publish logic.

**Evidence:**
- `internal/actors/scopes/derive/ema_crossover_signal_sampler_actor.go` vs `signal_sampler_actor.go`
- Both implement identical `Receive` → `AddClose` → `publishSignalMessage` → `signalGeneratedMessage` flow
- Identical `SignalSamplerConfig` struct reused, but no shared behavior extracted

**Impact at scale:** At 5 signal families, this yields ~500 lines of near-identical actor code across 5 files. Maintenance cost grows linearly; a bug in the publish pattern requires fixing N files.

**Verdict:** Boilerplate is mechanical and correct today. Not a bug, not a fragility. Becomes structural debt at N=3 when a generic factory is justified by three concrete examples.

---

### 2.2 NATS Registry Switch Proliferation

**Classification: Structural debt (moderate friction, scales poorly)**

The NATS adapter layer requires 4+ manual touch points per new signal family:

| Site | What changes | Lines |
|------|-------------|-------|
| `signal_registry.go` | New `EventSpec` + `ControlSpec` fields + consumer function | ~30 |
| `signal_publisher.go` | New case in `specForType()` switch | ~3 |
| `signal_publisher.go` | New case in `LatestSpecByType()` switch (if present) | ~3 |
| `signal_kv_store.go` | New bucket constant | ~1 |

**Evidence:**
- `specForType()` is a hardcoded switch: `case "rsi"` / `case "ema_crossover"` / `default: nil`
- Each family adds a separate `StoreXxxSignalConsumer()` function (~15 lines)
- No centralized registry lookup — changes are scattered

**Impact at scale:** At 5 families, the switch becomes a 20-line dispatch table. Consumer spec functions proliferate. All must stay in sync with schema, supervisor, and store layers.

**Verdict:** Functional today. The scatter across files is the real friction — not the line count. A map-based registry replacing switch statements would centralize this.

---

### 2.3 Store Pipeline Registration Boilerplate

**Classification: Boilerplate (acceptable, mechanical)**

Each signal family requires ~25 lines in `store_supervisor.go`'s `declarePipelines()`:
- Pipeline struct with unique names, bucket reference, consumer spec, enable predicate, and two factory closures.

**Evidence:**
- `internal/actors/scopes/store/store_supervisor.go` lines 185-227
- RSI and EMA entries are structurally identical; only names/constants differ

**Impact at scale:** Linear growth (~25 lines per family). The pattern is declarative and self-documenting, which partially offsets the duplication.

**Verdict:** Acceptable boilerplate. The declarative pipeline struct is already a reasonable abstraction. Further reduction (e.g., generating entries from a catalog) is premature at N=2.

---

### 2.4 Derive Supervisor Processor Registration

**Classification: Boilerplate (acceptable, minimal)**

Each signal family adds ~10 lines to the `SignalFamilyProcessor` slice in `derive_supervisor.go`.

**Evidence:**
- `internal/actors/scopes/derive/derive_supervisor.go` lines 134-154
- `filterEnabled()` generic already handles activation/deactivation

**Impact at scale:** Low. The processor slice is clean and readable.

**Verdict:** Acceptable. The `filterEnabled()` pattern is good design. No action needed.

---

### 2.5 Correlation ID Manual Copy — CF-03 Reassessed

**Classification: Structural debt (design ready, implementation deferred)**

Every actor manually copies `events.NewMetadata().WithCorrelationID(msg.CorrelationID)`. CC-02 added one more actor with this pattern.

**Evidence:**
- `ema_crossover_signal_sampler_actor.go` copies the pattern identically
- CF-03 design sketch exists from S123 but was **not implemented** during CC-02
- S124 defined the trigger as "first new actor (CC-02 or equivalent)"

**Critical observation:** The trigger condition (first new actor) was met during CC-02, but the middleware was not implemented. This means either:
1. The team judged the cost/benefit unfavorable at implementation time, or
2. The trigger was missed

**Impact:** No incident has occurred from the manual pattern. The risk is future omission, not current breakage.

**Verdict:** The trigger fired but action was not taken. At N=2 families (and likely 10+ actors system-wide), the risk remains low. Recommend implementing the middleware opportunistically when the next actor is added (CC-03), with explicit tracking.

---

### 2.6 Configuration Schema — Manual Map Entries

**Classification: Boilerplate (acceptable, trivial)**

Adding a signal family requires 2 map entries:
- `knownSignalFamilies["ema_crossover"] = true`
- `signalDependsOnEvidence["ema_crossover"] = []string{"candle"}`

**Evidence:**
- `internal/shared/settings/schema.go` lines 28-31, 65-68
- Validation logic in `ValidatePipeline()` is already generic

**Verdict:** Trivial. The maps are the right abstraction. No action needed.

---

### 2.7 No Per-Family Algorithm Configuration

**Classification: Intentional limitation (not a friction)**

EMA periods (fast=9, slow=21) are hardcoded in the sampler constructor. No config-driven parameterization exists.

**Evidence:**
- `internal/application/signal/ema_crossover_sampler.go` — fixed constants in `NewEMACrossoverSampler()`
- RSI period (14) is similarly hardcoded

**Verdict:** This is a deliberate simplification documented in CC-02 scope. Per-family configuration is a feature, not a debt. It should be introduced when a use case requires A/B testing or per-binding tuning — not preemptively.

---

### 2.8 HTTP Route Layer

**Classification: No friction**

The `/signal/:type/latest` route is fully type-parameterized. Zero changes needed for CC-02.

**Evidence:**
- `internal/interfaces/http/routes/signal.go` — 28 lines, unchanged
- Type dispatch happens in the gateway query responder via `LatestSpecByType()`

**Verdict:** Exemplary extensibility. The route layer is proof that the parameterized design works.

---

### 2.9 Diagnostic Surfaces (healthz, statusz, diagz)

**Classification: No friction**

All diagnostic endpoints automatically include `ema_crossover` actors. Zero diagnostic code was written for CC-02.

**Evidence:**
- `/statusz` reports per-actor trackers (event/error counts, idle seconds)
- `/diagz` includes readiness checks
- `/healthz` confirms liveness
- All actors participate via the shared `healthz.Tracker` injection

**Verdict:** Diagnostic infrastructure is truly family-agnostic. No action needed.

---

### 2.10 Client UseCase Boilerplate — CF-08 (Partial)

**Classification: Structural debt (deferred, trigger = new domain family)**

6 domain client packages hand-write identical `struct + Execute` patterns (~30 LOC each, ~180 lines total). `configctlclient` was migrated to shared `usecase` type aliases; others were not.

**Evidence:**
- `internal/shared/usecase/usecase.go` exists with shared types
- `internal/application/configctlclient/` uses shared types
- `signalclient`, `evidenceclient`, `riskclient`, etc. still use hand-written patterns

**Verdict:** CC-02 did not trigger this because it added a signal family (existing domain), not a new domain. The trigger remains "when adding a new domain family."

---

## 3. Friction Cost Model (Evidence-Based)

| Category | Lines per Family | Touch Points | Friction Level |
|----------|-----------------|-------------|---------------|
| Domain sampler + tests | ~240 | 2 new files | Low (unique logic) |
| Sampler actor | ~97 | 1 new file | Medium-High (copy-paste) |
| NATS registry/publisher | ~37 | 3 files | Medium (scattered switches) |
| KV bucket constant | ~1 | 1 file | Trivial |
| Store pipeline entry | ~25 | 1 file | Low-Medium (declarative) |
| Derive processor entry | ~10 | 1 file | Low (clean pattern) |
| Config schema maps | ~4 | 1 file | Trivial |
| HTTP routes | 0 | 0 files | None |
| Diagnostics | 0 | 0 files | None |
| **Total** | **~414** | **3 new + 7 modified** | — |

Of the ~414 lines, ~240 are unique domain logic and tests. The remaining ~174 lines are mechanical registration boilerplate.

## 4. What Did NOT Confirm as a Problem

### 4.1 Domain Model Rigidity — Not Confirmed
Pre-CC-02 concern: `signal.Signal` might not accommodate categorical values. Reality: `Value: string` and `Metadata: map[string]string` handled RSI (numeric) and EMA Crossover (categorical: bullish/bearish/neutral) without any modification.

### 4.2 Stream Topology Pressure — Not Confirmed
Pre-CC-02 concern: NATS stream might need restructuring for multi-family signals. Reality: `SIGNAL_EVENTS` wildcard subjects automatically cover new families. Zero stream changes needed.

### 4.3 Config Lifecycle Breakage — Not Confirmed
Pre-CC-02 concern: adding a family might break config validation or activation. Reality: config validation is generic; activation via `pipeline.signal_families` array is additive.

### 4.4 Diagnostic Surface Gaps — Not Confirmed
Pre-CC-02 concern: new families might lack observability. Reality: all diagnostic surfaces are actor-driven and automatically include new actors.

### 4.5 Coexistence Interference — Not Confirmed
Pre-CC-02 concern: two signal families might interfere. Reality: separate KV buckets, separate consumers, shared stream with subject isolation. Zero interference.

## 5. Summary

CC-02 confirms that market-foundry's extensibility model works. The friction is real but bounded, mechanical, and predictable. The primary structural debts (actor boilerplate CF-08, NATS registry scatter, correlation ID CF-03) are documented with clear thresholds. The domain model, stream topology, config lifecycle, diagnostic surfaces, and HTTP routes are genuinely extensible with zero friction.
