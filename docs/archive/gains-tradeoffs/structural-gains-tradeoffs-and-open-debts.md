# Structural Gains, Trade-offs, and Open Debts

> Honest accounting of what the S96–S99 consolidation wave delivered, what it cost, and what remains unresolved.

---

## Gains: What Actually Improved

### G1. Composition Root Legibility

**Before:** Gateway `run.go` was 231 lines mixing connection setup, use case wiring, route assembly, and actor spawning. Store supervisor was 508 lines with 6 duplicated pipeline types.

**After:** Gateway `run.go` is 40 lines with 3 visible phases. Store supervisor dropped to ~280 lines. Every runtime follows the same 6-phase pattern.

**Value:** Any developer can understand a runtime's startup sequence by reading one file. Phase boundaries prevent concern interleaving. This is not cosmetic — it directly reduces the time to diagnose startup failures and the risk of wiring errors when adding dependencies.

### G2. Single-Entry Family Addition

**Before:** Adding a store pipeline required editing `declarePipelines()` AND a separate `allTrackerDefs` list. Adding a derive processor required editing the processor list AND a separate filtering block.

**After:** One catalog entry per family. Trackers, filtering, logging, and scope wiring derive automatically.

**Value:** Eliminates an entire class of "forgot to update the second list" bugs. The cost of adding a family is now O(1) in files touched.

### G3. Generic Factory Elimination of Boilerplate

**Before:** 8 near-identical gateway factory functions (135 lines). 6 identical tracker struct types. 6 near-identical filter-and-spawn blocks.

**After:** `newGatewayConn[T]()`, `filterEnabled[T]()`, `familyNames[T]()` — three generics that replaced ~250 lines of duplicated code.

**Value:** When the pattern changes (e.g., adding timeout configuration to all gateways), the change happens in one place. Type safety is preserved through generics.

### G4. Semantic Precision in Terminology

**Before:** "Scope" meant both actor supervision boundary AND domain classifier. "Service" appeared in error messages despite the system using gateways. "Quality" artifacts remained from pre-rename.

**After:** Precise terminology map enforced across code and documentation. Scope = actor boundary. Domain = bounded context. Gateway = NATS request/reply port.

**Value:** Eliminates cognitive overhead during code review and incident response. When an operator sees "evidence gateway is unavailable", they know exactly what component failed.

### G5. Automated Architecture Enforcement

**Before:** Architectural rules existed as tribal knowledge. Boundary violations could only be caught during review.

**After:** raccoon-cli enforces 11 structural rules via AST inspection. Quality gate profiles provide graduated enforcement for development (fast), CI (ci), and deep validation (deep).

**Value:** Architecture constraints are mechanically enforced. A developer cannot accidentally introduce a domain→adapter import — `arch-guard` catches it before merge.

### G6. Documented Growth Patterns

**Before:** Adding a new domain or family required reverse-engineering existing code to understand the pattern.

**After:** Step-by-step playbooks for domains, families, runtimes, and adapters. Naming conventions are binding, not advisory.

**Value:** Reduces onboarding time and prevents pattern drift. New additions follow the established pattern by default.

---

## Trade-offs: What the Consolidation Cost

### T1. Documentation Volume

S96–S99 produced 9 architecture documents and 4 stage reports. This is significant documentation overhead for a codebase with 6 services and <15 active families.

**Assessment:** The documentation is proportionate to the architectural complexity, but it introduces a maintenance burden. If conventions evolve, documents must be updated — and stale documentation is worse than no documentation. The three-tier structure (architecture, stages, tooling) helps by separating mutable conventions from immutable history.

**Mitigation:** Architecture docs should be reviewed quarterly. raccoon-cli's drift detection can partially automate this by flagging when code diverges from documented conventions.

### T2. Abstraction Ceiling Risk

The catalog-driven pattern (`declarePipelines()`, `filterEnabled[T]()`) is elegant at the current scale (~12 pipeline entries, ~6 processor types). If pipeline entry shapes diverge significantly (e.g., a pipeline that needs fundamentally different lifecycle management), the uniform catalog may require escape hatches.

**Assessment:** Currently acceptable. The derive supervisor already handles this correctly — separate processor types for evidence, signal, decision, strategy, risk, and execution because their signatures genuinely differ. The pattern was not forced where it didn't fit.

**Mitigation:** If a pipeline requires genuinely different lifecycle, add it outside the catalog with explicit justification. Do not force-fit divergent shapes into the uniform pattern.

### T3. Guardian Tooling Maintenance Cost

raccoon-cli is a substantial Rust codebase (~550KB of analyzer source). It must evolve with the Go codebase it guards. Changes to Go module structure, import paths, or naming conventions require corresponding raccoon-cli updates.

**Assessment:** The maintenance cost is justified because the alternative (manual architecture review) scales worse. However, raccoon-cli's line-based Go parser has inherent limitations — it cannot resolve types across packages without LSP integration.

**Mitigation:** Keep raccoon-cli's parser scope narrow (structural facts, not type resolution). Use optional gopls integration for deeper analysis where needed.

### T4. Composition Root Rigidity

The 6-phase lifecycle pattern is clean but rigid. A runtime that needs a fundamentally different startup sequence (e.g., a stateful runtime with migration steps, or a runtime with dynamic service discovery) would need to break the pattern.

**Assessment:** Currently acceptable. All 6 runtimes genuinely fit the pattern. The risk is that the pattern becomes dogma rather than guidance.

**Mitigation:** Document that the 6-phase pattern is a default, not a constraint. New runtimes may deviate with explicit justification.

---

## Open Debts: What Remains Unresolved

### D1. Test Infrastructure Gaps

The consolidation wave focused on structural patterns, not test coverage. Key gaps:
- **No integration tests for composition roots.** The catalog-driven assembly is tested implicitly through manual testing, not through automated integration tests that verify pipeline wiring.
- **No contract tests for NATS subjects.** raccoon-cli audits contract alignment statically, but there are no runtime tests verifying that a published event on subject X is consumed correctly by subject X's consumer.
- **Repository tests exist** (configctl has thorough tests), but other domains have less coverage.

**Recommendation:** Address in a future test infrastructure wave if test failures become a recurring pain point. Do not add integration tests preemptively.

### D2. Observability Gaps

No structured observability beyond health trackers:
- No distributed tracing through the actor pipeline (trade → sample → evaluate → resolve → execute).
- No metrics for pipeline throughput, latency, or backpressure.
- Health trackers exist but only report binary alive/dead status.

**Recommendation:** Address when operational issues (latency diagnosis, throughput debugging) become the primary blocker. Not justified as pure tech debt work.

### D3. Error Propagation Consistency

`*problem.Problem` is used uniformly, but error handling patterns vary:
- Some use cases return `*problem.Problem` directly.
- Some actor message handlers log errors and continue.
- Some composition root failures are fatal; others degrade gracefully (gateway's optional connections).

**Assessment:** The current inconsistency is largely appropriate — different contexts warrant different error handling strategies. However, there is no documented error handling convention that explains when to use which strategy.

**Recommendation:** Document error handling conventions if error handling bugs become a recurring issue. Do not standardize preemptively.

### D4. Configuration Validation Completeness

Cross-layer dependency validation exists (`signalDependsOnEvidence`, `decisionDependsOnSignal`, etc.) but is maintained as static maps in `settings/schema.go`. These maps must be updated manually when new families are added.

**Assessment:** This is a minor synchronization point. It's less risky than the pipeline catalog duplication that S97 fixed (config validation is checked at startup, not at runtime), but it's still a potential source of "forgot to update" bugs.

**Recommendation:** Consider deriving dependency maps from pipeline catalog metadata in a future iteration. Low priority.

### D5. Venue Adapter Expansion Path

The execute runtime's venue adapter selection is an explicit switch statement. This is intentionally not catalog-driven (security implications of auto-discovery). However, the pattern for adding a new venue adapter is not documented in the expansion playbooks.

**Recommendation:** Add venue adapter expansion to the how-to guide when the next venue is added. Do not over-engineer the selection mechanism.

---

## Refactors That Do NOT Warrant the Cost Now

### R1. Unified Pipeline Type Across Store and Derive

Store uses `Pipeline` struct. Derive uses separate processor types per domain (`FamilyProcessor`, `SignalFamilyProcessor`, etc.). Unifying them would lose type-level guarantees because their actor constructor signatures genuinely differ.

**Why not:** Type safety > DRY. The current separation is correct.

### R2. Generic Supervisor Framework

Store, derive, ingest, and execute supervisors share structural patterns (start, receive, shutdown). A generic supervisor framework could reduce this duplication.

**Why not:** Each supervisor has domain-specific lifecycle logic (store: pipeline catalog iteration; derive: dynamic source scope creation; execute: venue adapter selection). A framework would either be too generic to be useful or too specific to be reusable.

### R3. Automated Documentation Generation

Architecture documents could theoretically be generated from code annotations or raccoon-cli analysis.

**Why not:** Architecture documents capture intent and rationale, not structure. Generating them from code would produce accurate but useless documentation. The value is in the human-written "why", not the mechanically-extractable "what".

### R4. Event Schema Registry

Domain events are currently defined as Go structs with JSON serialization. A formal schema registry (Protobuf, Avro, JSON Schema) would add contract enforcement.

**Why not:** The system has a single producer per event type and communication is intra-cluster via NATS. Schema enforcement adds build complexity and deployment coupling without addressing a current problem. Consider only when multi-team or multi-language consumers exist.

### R5. Abstract Repository Interface

Each domain has its own repository interface. A generic `Repository[T]` interface could reduce boilerplate.

**Why not:** Repository methods are domain-specific (`GetConfigSetByKey`, `GetActiveConfigs`). A generic interface would either be too broad (exposing operations that don't make sense for all domains) or require per-domain extensions that negate the abstraction.
