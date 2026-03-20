# Anti-Debt Checklist

## Purpose

This checklist is a practical review tool for detecting and preventing technical debt in Market Foundry. Use it during PR reviews, stage closures, and readiness assessments.

Each category lists specific questions. A "no" answer to any question is a debt signal that must be investigated and either resolved or explicitly documented as a known gap.

---

## 1. Boundary Debt

Boundary debt occurs when responsibilities leak across binary or layer boundaries.

- [ ] Does each binary still have a single-sentence purpose that accurately describes ALL of its current responsibilities?
- [ ] Is gateway still purely stateless? (No KV access, no domain logic, no event publishing, no JetStream subscriptions)
- [ ] Is store still purely read-side? (No domain event production, no domain logic beyond projection)
- [ ] Is derive still purely write-side to EVIDENCE_EVENTS? (No persistent storage, no query serving)
- [ ] Does configctl remain the sole config authority? (No other binary manages configuration state)
- [ ] Are there any direct function calls between code owned by different binaries?
- [ ] Does every inter-binary communication go through NATS (request/reply or JetStream)?
- [ ] Does layer sovereignty hold? (domain → application → adapters → actors → interfaces → cmd, no outward imports)

**Debt signals:**
- Gateway caching evidence data.
- Store producing domain events.
- Derive serving queries (acceptable only in early slices before store existed; must be resolved).
- Application layer importing from actors or interfaces.

---

## 2. Naming Debt

Naming debt occurs when identifiers diverge from established conventions.

- [ ] Do all NATS subjects follow `{domain}.{plane}.{aggregate}.{verb}[.{key}]`?
- [ ] Do all message types follow `{domain}.{plane}.{version}.{name}`?
- [ ] Do all KV buckets follow `{TYPE}_LATEST` / `{TYPE}_HISTORY` conventions?
- [ ] Do all consumer durables follow `{service}-{family}` or `{service}-binding-watcher`?
- [ ] Do all actor names follow `{Domain}{Role}Actor` (e.g., CandleProjectionActor)?
- [ ] Do all HTTP endpoints follow `/evidence/{type}/{operation}`?
- [ ] Is there any residue of "server" (should be "gateway"), "consumer" (old Kafka bridge), "validator", or "emulator" naming?
- [ ] Do domain types follow `Evidence{Type}` naming (e.g., EvidenceCandle, EvidenceTradeBurst)?
- [ ] Do event names follow `{type}.sampled` convention for evidence?

**Debt signals:**
- Mixed casing in subject hierarchies.
- Consumer durables that don't match the owning service.
- Actor names that don't reflect their role.
- HTTP paths that diverge from the `/evidence/{type}/{operation}` pattern.

---

## 3. Ownership Debt

Ownership debt occurs when the single-writer or single-owner invariant is violated or unclear.

- [ ] Does every JetStream stream have exactly one writer binary?
- [ ] Does every KV bucket have exactly one writer actor?
- [ ] Does every query subject have exactly one server binary?
- [ ] Is the stream ownership matrix (stream-ownership-matrix.md) current?
- [ ] Is actor-ownership.md current with all actors, their supervisors, and their owned resources?
- [ ] Are there any "shared" resources written by multiple actors or binaries?

**Debt signals:**
- Two binaries publishing to the same stream.
- Two actors writing to the same KV bucket.
- Ownership matrix listing actors that no longer exist.
- Ownership matrix missing actors that were added in recent stages.

---

## 4. Stream Mesh Debt

Stream mesh debt occurs when the logical message topology diverges from the documented mesh.

- [ ] Does stream-family-catalog.md list every active family with correct status?
- [ ] Does every active stream have: name, retention, max bytes, writer, consumers, subjects, dimensions?
- [ ] Are there any streams in the codebase not listed in the catalog?
- [ ] Are there any streams in the catalog marked "Planned" that actually have working code?
- [ ] Do all consumer filter subjects match the documented subject patterns?
- [ ] Is the data flow still acyclic across the governed branches? (`configctl → ingest → derive`, `derive → store → gateway`, `derive → execute`, `derive/execute → writer → ClickHouse → gateway`)
- [ ] Are all deduplication mechanisms documented (message ID format, idempotency keys)?

**Debt signals:**
- A stream family implemented in code but still listed as "Planned" in catalog.
- Consumer filter subjects that don't match the family's documented publish subjects.
- Undocumented streams or consumers in NATS configuration.
- Circular data flow between binaries.

---

## 5. Configuration Debt

Configuration debt occurs when the config-driven activation model is incomplete or inconsistent.

- [ ] Does every binary that reacts to configuration changes have a BindingWatcherActor?
- [ ] Is the BindingWatcherActor in every binary subscribing to the correct configctl events?
- [ ] Are there any hardcoded activations that should be config-driven?
- [ ] Does the compose configuration match the config files in `deploy/configs/`?
- [ ] Are all compose services listed that should be? Are dependency chains correct?
- [ ] Do config files reference valid NATS subjects and stream names?

**Debt signals:**
- Binary spawning actors without waiting for configctl activation events.
- Store spawning all projections unconditionally instead of per-binding.
- Config files referencing subjects that don't match stream-taxonomy conventions.
- Compose services missing health check dependencies.

---

## 6. Premature Abstraction Debt

Premature abstraction debt occurs when code is generalized beyond current needs.

- [ ] Does every interface in the codebase have more than one implementation?
- [ ] If an interface has only one implementation, is the polymorphism actively needed (e.g., port contract)?
- [ ] Are there any "framework-like" patterns (plugin systems, dynamic loaders, registry-of-registries)?
- [ ] Are FamilyProcessor and ProjectionPipeline still structs (not interfaces)?
- [ ] Is there any code that anticipates features not yet in any readiness review?
- [ ] Are there helper functions used only once?

**Debt signals:**
- Interface with single implementation and no port/adapter justification.
- Generic `Processor[T]` or `Pipeline[T]` type parameters where concrete types suffice.
- Dynamic plugin loading for evidence types (should be compiled-in).
- "Utility" packages that serve only one caller.

---

## 7. Query / Read Model Debt

Query and read model debt occurs when the read path diverges from projection authority patterns.

- [ ] Is store the sole server for all latest-value evidence query subjects?
- [ ] Does gateway access latest operational read data through NATS request/reply to store and analytical history through ClickHouse reader adapters only?
- [ ] Does every projection have monotonicity guard on latest?
- [ ] Does every projection validate domain objects before KV write?
- [ ] Does every projection only materialize `Final=true` events?
- [ ] Does every projection type have its own dedicated consumer with type-specific filter subject?
- [ ] Does the QueryResponderActor route every evidence type correctly?
- [ ] Are all query response types documented in contracts?

**Debt signals:**
- Gateway reading directly from NATS KV.
- Gateway owning analytical SQL or storage translation outside the analytical reader adapters/use cases.
- Projection accepting non-final events.
- Missing monotonicity guard (latest projection can regress).
- QueryResponderActor missing routes for an implemented evidence type.
- Mismatched query response types between store and gateway client.

---

## 8. Documentation Debt

Documentation debt occurs when architectural docs do not reflect the current codebase.

- [ ] Is actor-ownership.md current? (within 1 stage of latest)
- [ ] Is stream-family-catalog.md current? (all active families listed with correct status)
- [ ] Is stream-ownership-matrix.md current? (all streams, consumers, writers accurate)
- [ ] Are pattern docs current? (derive-pipeline-pattern, projection-writer-pattern, etc.)
- [ ] Does every completed stage have a stage report in `docs/stages/`?
- [ ] Do stage reports accurately list deferred items and known gaps?
- [ ] Do architecture docs cross-reference each other correctly?

**Debt signals:**
- actor-ownership.md more than 2 stages behind.
- Stage report missing for a completed stage.
- Pattern doc describing behavior that code no longer follows.
- Cross-references pointing to renamed or deleted documents.

---

## 9. Governance Debt

Governance debt occurs when enforcement mechanisms lag behind the codebase.

- [ ] Does raccoon-cli know about all current binaries?
- [ ] Does raccoon-cli know about all current JetStream streams?
- [ ] Does raccoon-cli know about all current consumer durables?
- [ ] Does raccoon-cli know about all current NATS subject prefixes?
- [ ] Does raccoon-cli's architecture doc inventory match the actual docs?
- [ ] Can `make check` detect all violations of patterns introduced in the last 3 stages?
- [ ] Are all drift-detect rules current (config-compose, binary-compose, naming identity, actor-scope, stream-registry)?

**Debt signals:**
- raccoon-cli topology expects consumers that no longer exist.
- raccoon-cli topology missing consumers that were added.
- `make check` passes but manual review finds structural violations.
- Drift-detect rules reference old naming conventions.

---

## 10. Operational Debt

Operational debt occurs when runtime observability or deployment does not match the architectural state.

- [ ] Does every consumer actor have a health tracker?
- [ ] Does every projection actor have a health tracker?
- [ ] Do all headless services expose `/healthz` and `/statusz` endpoints?
- [ ] Does docker-compose have health checks for all services?
- [ ] Are compose service dependencies correctly ordered?
- [ ] Do smoke tests cover all active evidence types end-to-end?
- [ ] Is `make smoke` (or equivalent) exercising the current full pipeline?

**Debt signals:**
- New evidence type with no smoke test coverage.
- Service without health endpoint.
- Compose service without health check.
- Smoke test that only covers candle but not tradeburst or volume.

---

## Architectural Drift Signals

These are higher-level signals that the architecture is diverging from its intended direction. If you observe any of these, pause and assess before proceeding:

1. **Boundary erosion** — A binary starts doing something outside its single-sentence purpose.
2. **Pattern proliferation** — A third way of doing something that already has two established patterns.
3. **Governance lag** — More than 2 stages have passed since raccoon-cli rules were updated.
4. **Naming entropy** — New code uses different conventions than existing code for the same concepts.
5. **Scope creep in stages** — A stage report's "OUT of scope" section shrinks during implementation.
6. **Readiness bypass** — Implementation begins before readiness review prerequisites are met.
7. **Framework emergence** — Simple patterns acquire configuration, plugins, or dynamic loading.
8. **Feedback loops** — Data flows in a direction opposite to the canonical flow (store → derive, derive → ingest).
9. **Implicit temporariness** — Code introduced as "temporary" with no documented resolution path.
10. **Triad disconnection** — Changes that have no grounding in Foundry patterns, MarketMonkey references, or Market Raccoon domain guidance.

---

## Pre-Approval Questions

Before approving any change to Market Foundry, ask:

1. **Does this change have a single, clear purpose?** If not, it should be split.
2. **Does this change follow an established pattern?** If not, why is a new pattern needed?
3. **Is governance updated?** If structural changes were made, are docs and raccoon-cli current?
4. **Is this the simplest correct solution?** If there is a simpler approach that satisfies the same requirements, prefer it.
5. **Does this change make the system easier or harder to reason about?** Complexity must justify itself.
6. **Would this change be obvious to someone reading the code for the first time?** If not, it needs better naming, structure, or documentation.
7. **Does this change respect the readiness gate sequence?** (governance → contracts → activation → pattern → implementation)
8. **Can raccoon-cli detect a violation of whatever this change introduces?** If not, should it?
