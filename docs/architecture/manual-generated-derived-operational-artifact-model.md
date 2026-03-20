# Manual / Generated / Derived / Operational Artifact Model

> **Stage:** S214
> **Purpose:** Classify every artifact in the codebase by its production method and ownership
> **Scope:** Consolidation reference — no expansion

---

## 1. Classification Definitions

### Manual

Artifacts authored and maintained by humans. Changes require human judgment — architectural decisions, domain knowledge, or creative trade-offs. These are the primary codebase.

### Generated

Artifacts produced by the codegen engine from YAML specs via Go templates. Their content is deterministic given a spec + template pair. In the codebase, generated artifacts live between `codegen:begin` / `codegen:end` markers. CI validates them against golden snapshots.

### Derived

Artifacts that are **computed deterministically** from other artifacts but are NOT produced by the codegen engine. Examples: naming conventions computed by `spec.Derived()`, the `newConsumerSpec()` factory pattern, health tracker definitions extracted from pipeline declarations. Derived artifacts encode patterns but involve no codegen governance.

### Operational

Artifacts that exist purely for runtime configuration, deployment, or operational tooling. They reference families and domain types but are not generated or derived — they are authored to match the current deployment topology.

---

## 2. Full Artifact Map

### 2.1 Domain Layer (`internal/domain/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| Event types (structs) | Manual | Human | Architectural decisions per layer |
| Event interfaces | Manual | Human | Domain contracts |
| Shared types | Manual | Human | Cross-layer primitives |

### 2.2 NATS Adapter Layer (`internal/adapters/nats/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| Registry structs (EventSpec, ControlSpec fields) | Manual | Human | Domain contracts |
| DefaultRegistry() builders | Manual | Human | Stream config, event routing |
| LatestSpecByType() dispatchers | Manual | Human | Query routing logic |
| Writer consumer spec functions (RSI, EMA) | **Generated** | Machine | Between codegen markers |
| Writer consumer spec functions (all others) | Manual | Human | `manual:owned` annotated |
| Store consumer spec functions | Manual | Human | Always manual |
| `newConsumerSpec()` factory | **Derived** | Human | Encodes the pattern; used by both manual and generated specs |
| `consumer_spec_factory.go` | Derived | Human | Shared factory for all consumer specs |

### 2.3 Writer Service (`cmd/writer/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| Pipeline entry structs (RSI, EMA) | **Generated** | Machine | Between codegen markers |
| Pipeline entry structs (all others) | Manual | Human | `manual:owned` annotated |
| `writerPipeline` type definition | Manual | Human | Schema for all entries |
| Consumer actor logic | Manual | Human | NATS wiring |
| Inserter actor logic | Manual | Human | Batch + retry |
| Supervisor + recovery | Manual | Human | Lifecycle management |
| Row mappers (`mapCandleRow`, etc.) | Manual | Human | Domain knowledge required |
| Mapper tests | Manual | Human | Depends on manual mappers |

### 2.4 Codegen Engine (`codegen/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| Family YAML specs | Manual | Human | Source of truth for codegen |
| Go templates (`.go.tmpl`) | Manual | Human | Frozen; changes require authorization |
| Engine code (spec.go, render.go, compare.go) | Manual | Human | Codegen infrastructure |
| Golden snapshots (integrated families: rsi, ema) | **Generated** | Machine | Regenerated on spec/template change |
| Golden snapshots (non-integrated: candle, rsi_oversold, mean_reversion_entry, position_exposure, paper_order) | Manual | Human | Hand-crafted reference baselines |
| `integrated.yaml` manifest | Manual | Human | Governance authority |
| Tests | Manual | Human | Engine validation |

### 2.5 ClickHouse Adapter Layer (`internal/adapters/clickhouse/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| Client (connection, batch insert) | Manual | Human | Infrastructure |
| Reader interface | Manual | Human | Query contract |
| Query builder | **Derived** | Human | Shared pattern across all readers |
| Specialized readers (6 types) | Manual | Human | Per-table column mapping |
| Reader tests | Manual | Human | |

### 2.6 Analytical Application Layer (`internal/application/analyticalclient/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| Contracts (query/reply types) | Manual | Human | API contracts |
| Use cases (6 types) | Manual | Human | Validation + orchestration |
| Use case tests | Manual | Human | |

### 2.7 HTTP Layer (`internal/interfaces/http/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| Analytical handlers | Manual | Human | HTTP ↔ use case bridge |
| Analytical routes | Manual | Human | Endpoint registration |
| Domain handlers (evidence, signal, etc.) | Manual | Human | Operational query handlers |
| Domain routes | Manual | Human | |
| Handler tests | Manual | Human | |

### 2.8 Gateway (`cmd/gateway/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| Compose (DI wiring) | Manual | Human | Service composition |
| Run (lifecycle) | Manual | Human | Startup orchestration |
| Analytical reader adapters | Manual | Human | Bridge ClickHouse → use case |

### 2.9 Store Path (`internal/actors/scopes/store/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| Store supervisor | Manual | Human | All 13 pipelines |
| Projection actors | Manual | Human | KV materialization |
| Consumer actors | Manual | Human | Event consumption |
| Generic consumer actor | Manual | Human | Shared actor template |

### 2.10 Migrations (`deploy/migrations/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| SQL migration files | Manual | Human | Schema design decisions |
| Migration runner | Manual | Human | Execution orchestration |
| Catalog | **Derived** | Human | File discovery pattern |

### 2.11 Configuration (`deploy/configs/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| `gateway.jsonc` | **Operational** | Human | Deployment topology |
| `ingest.jsonc` | Operational | Human | |
| `store.jsonc` | Operational | Human | Family activation lists |
| `derive.jsonc` | Operational | Human | |
| `execute.jsonc` | Operational | Human | |
| `writer.jsonc` | Operational | Human | Batch/flush/retry tuning |
| `configctl.jsonc` | Operational | Human | |

### 2.12 Settings Schema (`internal/shared/settings/`)

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| `schema.go` (known families, validation) | Manual | Human | Family catalog + cross-layer rules |
| Settings tests | Manual | Human | |

### 2.13 CI / Scripts

| Artifact | Classification | Owner | Notes |
|----------|---------------|-------|-------|
| `.github/workflows/ci.yml` | **Operational** | Human | CI pipeline definition |
| `scripts/codegen-integrated-check.sh` | **Derived** | Human | Manifest-driven validation |
| `scripts/smoke-analytical-e2e.sh` | Operational | Human | Runtime validation |
| `scripts/smoke-first-slice.sh` | Operational | Human | |
| Other scripts | Operational | Human | |

---

## 3. Classification Statistics

| Classification | Count of Distinct Categories | Notes |
|----------------|------------------------------|-------|
| Manual | ~35 | Primary codebase |
| Generated | 4 slices | 2 families × 2 artifacts |
| Derived | ~5 | Patterns encoded as reusable logic |
| Operational | ~10 | Config, CI, scripts |

---

## 4. Why Each Classification Matters

### Manual artifacts
- **Change process:** Human review, PR, tests
- **Risk:** Inconsistency across families (addressed by codegen for repetitive cases)
- **Ownership signal:** No markers, or `manual:owned` annotation

### Generated artifacts
- **Change process:** Modify spec → regenerate → update golden → CI validates
- **Risk:** Drift between golden and target (caught by `codegen-integrated-check.sh`)
- **Ownership signal:** `codegen:begin` / `codegen:end` markers

### Derived artifacts
- **Change process:** Modify the shared pattern/factory; all consumers inherit
- **Risk:** Pattern divergence if someone bypasses the factory
- **Ownership signal:** None formal; convention enforced by code review

### Operational artifacts
- **Change process:** Edit config/script directly; validated at runtime
- **Risk:** Stale family lists if not updated when families change
- **Ownership signal:** None; operational artifacts live in `deploy/` or `scripts/`

---

## 5. What Remains Manual and Why

| Artifact | Why Manual |
|----------|-----------|
| Row mappers | Require domain column knowledge; `domain.columns` spec extension not implemented |
| Store pipelines | Projection + consumer actor pattern differs structurally from writer pipelines |
| ClickHouse readers | Query builder pattern differs from consumer spec pattern; Tier 2 not authorized |
| HTTP handlers/routes | Request/response mapping requires per-endpoint decisions |
| Use cases | Validation rules vary per family type |
| Domain events | Architectural contracts; not mechanical |
| Migrations | Schema design requires human judgment (partitioning, TTL, indexes) |
| Evidence writer consumer specs | Evidence naming exception encoded in spec.Derived() but integration not authorized |
| Decision/strategy/risk/execution writer specs | Specs exist but integration not authorized; manual code is structurally equivalent |
