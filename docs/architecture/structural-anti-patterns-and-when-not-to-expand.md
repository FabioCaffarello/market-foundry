# Structural Anti-Patterns and When Not to Expand

> Real anti-patterns observed during market-foundry evolution, and explicit criteria for when expansion is NOT warranted. Based on concrete decisions made (and mistakes corrected) across S1–S104.

---

## Part 1: Observed Anti-Patterns

These are patterns that actually occurred in market-foundry and were corrected. They are not theoretical risks.

### AP-1: Duplicated Catalog Synchronization Points

**What happened:** Before S97, adding a store pipeline required editing both `declarePipelines()` AND a separate `allTrackerDefs` list. Adding a derive processor required editing the processor list AND a separate filtering block. These dual-write points led to "forgot to update the second list" bugs.

**Correction:** S97 unified catalogs so trackers, filtering, and scope wiring derive automatically from a single pipeline entry.

**Rule:** If adding a family requires touching more than one declaration site within the same runtime, the catalog abstraction is missing or broken. File a debt issue.

**How to detect:** If your PR modifies a catalog entry AND a separate list in the same runtime, you've found a synchronization point.

---

### AP-2: Phase Interleaving in Composition Roots

**What happened:** Before S96, the gateway `run.go` was 231 lines mixing connection setup, use case wiring, route assembly, and actor spawning. This made startup failure diagnosis difficult because concerns were tangled.

**Correction:** S96 established the 6-phase lifecycle: infrastructure → composition → wiring → spawn → health → shutdown. Gateway `run.go` dropped to 40 lines.

**Rule:** Each phase in `Run()` must do exactly one thing. If you find yourself creating a NATS connection in Phase 3 (wiring) or spawning an actor in Phase 2 (composition), you're interleaving.

**How to detect:** Read `Run()` top-to-bottom. If a line creates infrastructure after wiring has started, or wires logic after actors are spawned, phases are interleaved.

---

### AP-3: Semantic Drift in Terminology

**What happened:** "Scope" meant both actor supervision boundary AND domain classifier. "Service" appeared in error messages despite the system using gateways. "Quality" artifacts remained from the pre-rename (quality-service → market-foundry).

**Correction:** S98 established precise terminology: Scope = actor boundary. Domain = bounded context. Gateway = NATS request/reply port. All stale references were cleaned.

**Rule:** Use the terminology map from `boundary-naming-and-interface-hygiene.md`. If a new term is ambiguous, resolve it before writing code.

**How to detect:** `raccoon-cli drift-detect` checks for defunct names (`consumer`, `emulator`, `validator`). For new ambiguities, code review is the check.

---

### AP-4: Premature Domain Creation

**What happened (avoided):** During planning, there was temptation to create domain packages for concepts that didn't yet have a concrete family. The `PROHIBITED_STREAMS` guard in `drift_detect.rs` prevents this for the NATS layer.

**Rule:** Never create a domain package without at least one family ready for implementation. A domain without a family is speculative architecture that accrues maintenance cost without delivering value.

**How to detect:** `raccoon-cli drift-detect` flags streams that appear in source before they're approved. For domain packages, check that every `internal/domain/{name}/` has at least one family type with a corresponding `events.go`.

---

### AP-5: Force-Fitting Divergent Shapes into Uniform Catalogs

**What happened (correctly avoided):** The derive supervisor uses separate processor types per domain (`FamilyProcessor`, `SignalFamilyProcessor`, `ExecutionFamilyProcessor`, etc.) because their constructor signatures genuinely differ. Execution processors don't receive a `scopePID` because they're terminal in the pipeline chain.

**Rule:** Catalog-driven assembly works when entries have uniform shape. When a new entry requires a fundamentally different lifecycle or constructor signature, add it outside the catalog with explicit justification. Do not force-fit.

**How to detect:** If you're adding `interface{}` parameters or reflection to make a catalog entry fit, the shape is divergent.

---

### AP-6: Cross-Domain Imports in the Domain Layer

**What happened:** This was prevented from the start by architectural convention, and later mechanically enforced by `arch-guard`.

**Rule:** Domain packages import only from the standard library and `internal/shared/`. No cross-domain imports. No application-layer imports. No adapter imports.

**How to detect:** `make arch-guard` catches this automatically. This is a hard rule, not a guideline.

---

### AP-7: Infrastructure Types Leaking into Domain/Application

**What happened:** The potential for NATS, Hollywood, or HTTP types to appear in domain or application layer function signatures. `arch-guard` now checks both import paths AND type expressions.

**Rule:** Domain and application layers must not reference infrastructure types — not in imports, not in struct fields, not in function signatures.

**How to detect:** `make arch-guard` inspects both `import` statements and type expressions for infrastructure markers (`nats`, `kafka`, `hollywood`, `jetstream`, `http.Client`, `sql.DB`).

---

### AP-8: Hardcoded Family Activation

**What happened (prevented):** All families are config-driven. There is no `if family == "candle"` logic in production code. Activation is controlled via `deploy/configs/*.jsonc`.

**Rule:** Family activation must always go through the settings validation pipeline: config file → `AppConfig` → `PipelineConfig.IsXxxFamilyEnabled()` → catalog filter.

**How to detect:** Search for family name string literals in non-test, non-config Go code. They should only appear in `settings/schema.go` (known families registry) and `declarePipelines()` / processor slices (catalog entries).

---

### AP-9: Test Fixtures Diverging from Canonical Conventions

**What happened:** Test fixtures used `"validator:v1"` as runtime loader long after the canonical convention changed to `"configctl-sync/v1"`. S104 corrected this across all test files.

**Rule:** Test fixtures must use canonical values. When a convention changes, grep for the old value in test files. Stale fixtures create false confidence — tests pass but verify the wrong behavior.

**How to detect:** After any convention rename, run: `grep -r "old_value" --include="*_test.go"`.

---

### AP-10: Documentation Without Corresponding Tooling Enforcement

**What happened:** Architecture documents from S96–S99 established conventions, but `drift_detect.rs` ARCH_DOCS list didn't include the new canonical documents. This meant the tooling couldn't verify that these critical governance documents exist.

**Rule:** When creating a governance document that defines binding conventions, add it to the relevant raccoon-cli enforcement list in the same PR. Documentation without enforcement degrades to aspiration.

**How to detect:** Check that every document referenced in playbooks appears in either `drift_detect.rs` ARCH_DOCS or domain-specific doc lists.

---

## Part 2: When NOT to Expand

### Do Not Create a New Domain When...

| Situation | What To Do Instead |
|-----------|-------------------|
| You have a new data type but it shares the ubiquitous language of an existing domain | Add it as a family within the existing domain |
| You want to "organize" related concepts that span multiple existing domains | That's a cross-cutting concern — use `internal/shared/` |
| You don't yet have a concrete family ready for implementation | Wait. Document the concept in an architecture decision record, not in code |
| The "domain" would have exactly one family forever | It's a family, not a domain. Domains justify their existence through family diversity |

### Do Not Create a New Family When...

| Situation | What To Do Instead |
|-----------|-------------------|
| You want a variant of an existing family with different parameters | Add configuration options to the existing family's processor |
| The "family" doesn't produce its own distinct event type | It's a processing variant, not a family. Families are defined by their event boundary |
| The family would exist only in derive (no store projection, no gateway query) | Consider whether it's a transient computation that doesn't warrant persistence |
| You can't name its position in the dependency chain | Clarify the architecture before writing code |

### Do Not Create a New Runtime When...

| Situation | What To Do Instead |
|-----------|-------------------|
| You want to isolate a single use case | Add it to the runtime that owns the operational context |
| The new runtime always deploys alongside an existing one | Merge the concerns into the existing runtime |
| You want a runtime "just in case" we need independent scaling later | Scale when you need to scale, not before. The 6-phase lifecycle makes extraction straightforward |
| The runtime would have no actors (just HTTP endpoints) | It probably belongs in the gateway runtime |

### Do Not Create a New Adapter Group When...

| Situation | What To Do Instead |
|-----------|-------------------|
| You're adding a new instance of an existing technology (another NATS usage, another exchange) | Add to the existing adapter group |
| The technology is only used by one domain | It might still warrant its own group for layer hygiene, but evaluate whether a local helper suffices |

---

## Part 3: The Expansion Cost Budget

Every expansion has a cost that extends beyond the initial implementation. Understand the full budget before committing:

| Expansion Type | Initial Cost | Ongoing Cost |
|---------------|-------------|-------------|
| New family (same domain) | ~8-12 files across layers | Config maintenance, raccoon-cli governance updates, test coverage |
| New family (new scope) | ~15-20 files | All of above + derive processor type, store scope injection, dependency validation |
| New domain | ~20-30 files | All of above + drift-detect constants (~50 LOC), domain doc governance |
| New runtime | ~10-15 files + deploy config | Docker-compose entry, health monitoring, log stream, operational runbook |
| New adapter group | ~5-8 files + go.mod | go.work entry, build pipeline inclusion |
| New venue adapter | ~5-8 files | Credential management, exchange API versioning, venue-specific test infrastructure |

### The Hidden Cost: Raccoon-CLI Maintenance

Every domain added to market-foundry requires corresponding governance constants in `drift_detect.rs`. This is ~50 lines of Rust per domain covering: expected docs, NATS subjects, durable consumers, KV buckets, adapter files, domain/application files.

This is intentional — the governance cost acts as a natural brake on speculative expansion. If the cost of maintaining governance for a new domain feels disproportionate, the domain may not be justified.

---

## Related Documents

- `expansion-playbooks-refined.md` — how to expand correctly
- `structural-gains-tradeoffs-and-open-debts.md` — what consolidation delivered and what it cost
- `boundary-naming-and-interface-hygiene.md` — terminology precision
- `monorepo-structure-and-engineering-conventions.md` — structural conventions
