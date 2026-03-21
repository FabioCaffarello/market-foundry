# Domain Evolution — Permitted vs. Prohibited Changes

**Charter:** Domain Logic Depth — Decision, Strategy, and Risk Evolution
**Date:** 2026-03-20
**Reference:** `domain-evolution-charter-and-scope-freeze.md`

---

## 1. Permitted Changes

### 1.1 Domain Logic (Core Scope)

| Change Type | Permitted | Condition |
|-------------|-----------|-----------|
| New decision evaluator implementations | ✅ Yes | Must follow existing actor pattern, must have tests |
| New strategy resolver implementations | ✅ Yes | Must follow existing actor pattern, must have tests |
| New risk evaluator implementations | ✅ Yes | Must follow existing actor pattern, must have tests |
| New decision/strategy/risk type registry entries | ✅ Yes | Must have corresponding evaluator/resolver code |
| Enrichment of existing domain models (new fields) | ✅ Yes | Only if required by a concrete evaluator/resolver |
| New enum values (outcome, direction, disposition) | ✅ Yes | Only if required by a concrete evaluator/resolver |
| New `DecisionInput`, `StrategyInput` compositions | ✅ Yes | Multi-input patterns are a charter objective |

### 1.2 Adapter Layer (Feature-Pulled)

| Change Type | Permitted | Condition |
|-------------|-----------|-----------|
| New NATS registry specs for new types | ✅ Yes | Follows existing spec pattern |
| ClickHouse reader query adjustments for new fields | ✅ Yes | Only if domain model evolves |
| ClickHouse migration for new columns/tables | ✅ Yes | Only if domain model evolves |
| HTTP handler adjustments for new query parameters | ✅ Yes | Only if domain model evolves |
| New projection actor variants | ✅ Yes | Only if a new materialization pattern is needed |

### 1.3 Test and CI (Feature-Pulled)

| Change Type | Permitted | Condition |
|-------------|-----------|-----------|
| New unit tests for new evaluators/resolvers | ✅ Yes | Required for every new implementation |
| New integration tests for new pipeline paths | ✅ Yes | Required for pipeline-proven criterion |
| Codegen golden snapshot updates | ✅ Yes | Required if registry changes |
| `make test-integration` in remote CI | ✅ Yes | High-value, low-cost addition |
| raccoon-cli rule adjustments for new patterns | ✅ Yes | Only if new patterns conflict with existing rules |

### 1.4 Documentation (Minimal)

| Change Type | Permitted | Condition |
|-------------|-----------|-----------|
| Stage reports (S234, S235, etc.) | ✅ Yes | Required per governance |
| Architecture docs for new evaluator/resolver patterns | ✅ Yes | Only if the pattern is genuinely novel |
| Charter amendment documents | ✅ Yes | Only if scope changes are needed |

---

## 2. Prohibited Changes

### 2.1 Domain Expansion (Hard Freeze)

| Change Type | Prohibited | Rationale |
|-------------|------------|-----------|
| New domain families (e.g., new `portfolio`, `order`, `position` domains) | ❌ No | Charter scope is decision/strategy/risk only |
| Signal domain changes beyond consumption | ❌ No | Signal is upstream; charter is downstream |
| Indicator domain changes | ❌ No | Indicator is upstream; charter is downstream |
| Execution domain changes beyond consumption | ❌ No | Execution is downstream of risk; not in charter |
| New event types outside decision/strategy/risk | ❌ No | Scope freeze |

### 2.2 Infrastructure Expansion (Hard Freeze)

| Change Type | Prohibited | Rationale |
|-------------|------------|-----------|
| New microservices | ❌ No | No new services in this charter |
| New databases or data stores | ❌ No | ClickHouse and NATS KV are sufficient |
| New message broker patterns | ❌ No | Existing NATS patterns are sufficient |
| New external dependencies (libraries, frameworks) | ❌ No | Unless strictly required by a new evaluator |
| Docker/deployment configuration changes | ❌ No | No operational readiness in this charter |

### 2.3 Cleanup and Refactoring (Hard Freeze)

| Change Type | Prohibited | Rationale |
|-------------|------------|-----------|
| Documentation cleanup wave | ❌ No | S232 explicitly deferred this |
| Broad refactoring of existing evaluators | ❌ No | Existing evaluators must remain stable |
| raccoon-cli comprehensive audit | ❌ No | Only targeted adjustments for new patterns |
| Code style or formatting sweeps | ❌ No | No cosmetic changes |
| Package restructuring | ❌ No | Architecture is stable |

### 2.4 Platform and Operational (Hard Freeze)

| Change Type | Prohibited | Rationale |
|-------------|------------|-----------|
| Observability (metrics, tracing, structured logging) | ❌ No | Future charter |
| Production deployment configurations | ❌ No | Future charter |
| Load testing infrastructure | ❌ No | Future charter |
| Security scanning additions | ❌ No | Future charter |
| marketmonkey absorption | ❌ No | Future charter |

---

## 3. Gray Zone — Requires Justification

These changes are neither automatically permitted nor automatically prohibited. They require explicit justification tied to a specific feature requirement:

| Change Type | Condition for Approval |
|-------------|----------------------|
| New shared utility in `pkg/` | Must be required by ≥2 new evaluators/resolvers |
| Modification to gateway HTTP router | Must be for a new analytical endpoint serving charter domains |
| Changes to actor framework (`internal/actors/`) | Must be for a new derive/store pattern required by a charter evaluator |
| Changes to codegen templates | Must be for new registry patterns required by charter types |
| New Makefile targets | Must directly support charter testing or validation |

**Rule:** If a change falls in the gray zone, the stage report must document why it was necessary and which charter objective it serves.

---

## 4. Enforcement

- `make quality-gate-ci` must remain at 0 errors after every stage.
- Each stage report must map changes to permitted categories above.
- Any prohibited change discovered in review is a **stop condition** (see entry-exit-and-stop-conditions document).
- Gray zone changes must be justified in the stage report.
