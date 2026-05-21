# Stage Report: First Slice Preparation

> Date: 2026-03-16
> Phase: Pre-implementation preparation for the first vertical slice.
> Goal: Prepare the codebase to receive the first slice without improvisation, legacy reintroduction, or principle violations.

---

## 1. Changes Applied

### 1.1 Domain Layer — Observation (`internal/domain/observation/`)

| File | Purpose |
|------|---------|
| `trade.go` | `ObservationTrade` canonical type — source, symbol, price (string), quantity (string), trade_id, buyer_maker, timestamp. Includes `Validate()` and `DeduplicationKey()`. |
| `events.go` | `TradeReceivedEvent` implementing `events.Event`. Event name: `market.trade_received`. |
| `trade_test.go` | Validation tests: required fields, deduplication key format. |

**Zero imports from application, adapters, actors, or interfaces.** Only depends on `internal/shared/events` and `internal/shared/problem`.

### 1.2 Domain Layer — Evidence (`internal/domain/evidence/`)

| File | Purpose |
|------|---------|
| `candle.go` | `EvidenceCandle` canonical type — source, symbol, timeframe (int), OHLCV (strings), trade_count, open_time, close_time, final flag. Includes `Validate()`. |
| `events.go` | `CandleSampledEvent` implementing `events.Event`. Event name: `candle.sampled`. |
| `candle_test.go` | Validation tests: required fields, timeframe > 0, close_time > open_time. |

**Zero imports from application, adapters, actors, or interfaces.** Same dependency profile as observation.

### 1.3 Adapter Layer — NATS Registries (`internal/adapters/nats/`)

| File | Purpose |
|------|---------|
| `observation_registry.go` | `ObservationRegistry` with `TradeReceived` event spec. `OBSERVATION_EVENTS` stream definition (6h retention, 1GB). `DeriveObservationConsumer()` durable consumer spec. |
| `evidence_registry.go` | `EvidenceRegistry` with `CandleSampled` event spec and `CandleLatest` query control spec. `EVIDENCE_EVENTS` stream definition (72h retention, 2GB). |
| `observation_registry_test.go` | Contract tests: subject naming, versioned types, stream config, lowercase enforcement. |
| `evidence_registry_test.go` | Contract tests: subject naming, versioned types, stream config, query queue group, lowercase enforcement. |

### 1.4 Application Layer — Evidence Client Contracts

| File | Purpose |
|------|---------|
| `evidenceclient/contracts.go` | `CandleLatestQuery` and `CandleLatestReply` — gateway-side query/response contracts for evidence candle queries. |
| `ports/evidence.go` | `EvidenceGateway` interface — port for querying evidence projections via NATS request/reply. |

### 1.5 Service Stubs — Ingest and Derive (`cmd/ingest/`, `cmd/derive/`)

| File | Purpose |
|------|---------|
| `cmd/ingest/main.go` | Entry point following existing pattern: flag-based config, `bootstrap.LoadAndValidate`, call `Run()`. |
| `cmd/ingest/run.go` | Stub `Run()` with TODO comments documenting the actor tree to be wired (IngestSupervisor → BindingWatcher → SourceSupervisor → ObservationPublisher). |
| `cmd/ingest/go.mod` | Module definition matching existing conventions. |
| `cmd/derive/main.go` | Entry point following existing pattern. |
| `cmd/derive/run.go` | Stub `Run()` with TODO comments documenting the actor tree to be wired (DeriveSupervisor → PipelineWatcher → ObservationConsumer → ExchangeScope → SymbolScope → CandleSampler → EvidencePublisher → QueryResponder). |
| `cmd/derive/go.mod` | Module definition matching existing conventions. |

Both stubs compile, start, and log a warning that they are not yet wired.

### 1.6 Infrastructure — Workspace, Build, Deploy

| File | Change |
|------|--------|
| `go.work` | Added `./cmd/derive` and `./cmd/ingest` to workspace. |
| `Makefile` | `BUILDABLE_SERVICES` updated to include `derive` and `ingest`. Help text updated. |
| `deploy/compose/docker-compose.yaml` | Added `ingest` and `derive` services following existing YAML anchor pattern. Both depend on `nats` (healthy), ingest also depends on `configctl` (healthy). |
| `deploy/configs/ingest.jsonc` | NATS-only config (no HTTP). |
| `deploy/configs/derive.jsonc` | NATS-only config (no HTTP). |
| `deploy/docker/go-service.Dockerfile` | Fixed legacy `quality-service` reference → `market-foundry` in directory creation. |

### 1.7 Adapter Fix — Missing Constant

| File | Change |
|------|--------|
| `internal/adapters/nats/common.go` | Added `defaultSetupTimeout = 10s` — was referenced by `jetstream_publisher.go` but missing (lost during sanitization). |

### 1.8 Documentation

| File | Change |
|------|--------|
| `DEVELOPMENT.md` | Updated services table (added ingest, derive as stubs). Updated project structure (added new cmd entries, expanded domain description). |

---

## 2. Structure Prepared for the First Slice

```
cmd/
├── configctl/           ✅ Exists (Phase 0)
├── derive/              ✅ NEW — stub, compilable
├── ingest/              ✅ NEW — stub, compilable
└── server/              ✅ Exists (Phase 0)

internal/domain/
├── configctl/           ✅ Exists (Phase 0)
├── evidence/            ✅ NEW — canonical types + events + tests
└── observation/         ✅ NEW — canonical types + events + tests

internal/application/
├── configctl/           ✅ Exists
├── configctlclient/     ✅ Exists
├── evidenceclient/      ✅ NEW — query contracts
├── ports/
│   ├── configctl.go     ✅ Exists
│   └── evidence.go      ✅ NEW — EvidenceGateway port
└── runtimecontracts/    ✅ Exists

internal/adapters/nats/
├── configctl_registry.go     ✅ Exists
├── observation_registry.go   ✅ NEW — stream + subject contracts
├── evidence_registry.go      ✅ NEW — stream + subject + query contracts
└── common.go                 ✅ FIXED — missing defaultSetupTimeout

deploy/
├── compose/docker-compose.yaml   ✅ UPDATED — 5 services
├── configs/ingest.jsonc          ✅ NEW
├── configs/derive.jsonc          ✅ NEW
└── docker/go-service.Dockerfile  ✅ FIXED — quality-service → market-foundry
```

---

## 3. Intentional Gaps (What Was NOT Implemented)

| Gap | Reason | When to Implement |
|-----|--------|-------------------|
| **Actor implementations** (IngestSupervisor, DeriveSupervisor, all child actors) | These are the core slice implementation, not preparation | First slice implementation |
| **Exchange adapter** (`internal/adapters/exchanges/binancef/`) | WebSocket client is implementation, not contract | First slice Step 3 |
| **Application use cases** (ingest bootstrap, derive sampling logic) | Business logic is implementation | First slice Steps 4-5 |
| **HTTP handlers/routes for evidence** (GET `/evidence/candles/latest`) | Gateway extension is implementation | First slice Step 7 |
| **NATS evidence gateway adapter** (implements `EvidenceGateway` port) | Adapter wiring is implementation | First slice Step 7 |
| **Integration tests** (embedded NATS, full-stack HTTP) | Require functional code to test | First slice Steps 5-7 |
| **Actor scope directories** (`actors/scopes/ingest/`, `actors/scopes/derive/`) | Created during implementation, not preparation | First slice Step 5 |
| **Subject rename migration** (configctl subjects → canonical taxonomy) | Separate concern, documented in `stream-taxonomy.md` | Pre-Phase 2 cleanup commit |
| **cmd/server → cmd/gateway rename** | Separate concern, documented in `runtime-target.md` | Separate commit |

---

## 4. Validation Results

| Check | Result |
|-------|--------|
| `go build ./cmd/ingest` | ✅ Pass |
| `go build ./cmd/derive` | ✅ Pass |
| `go build ./cmd/configctl` | ✅ Pass |
| `go build ./cmd/server` | ✅ Pass |
| `go test ./internal/domain/observation/...` | ✅ Pass |
| `go test ./internal/domain/evidence/...` | ✅ Pass |
| `go test ./internal/adapters/nats/...` | ✅ Pass |
| `docker compose config` | ✅ Valid |
| Pre-existing test failure in `application/configctl` | ⚠️ Unrelated (CompileUseCase default loader name mismatch) |

---

## 5. Readiness Checklist for First Slice Implementation

### Prerequisites (all met)

- [x] Observation domain types defined and tested
- [x] Evidence domain types defined and tested
- [x] NATS stream definitions registered (OBSERVATION_EVENTS, EVIDENCE_EVENTS)
- [x] NATS subject contracts registered with versioned types
- [x] Durable consumer spec defined (derive-observation)
- [x] Evidence query contracts defined (CandleLatestQuery/Reply)
- [x] Evidence gateway port defined (EvidenceGateway interface)
- [x] cmd/ingest stub compiles and starts
- [x] cmd/derive stub compiles and starts
- [x] Docker Compose topology includes all 5 services
- [x] Config files exist for ingest and derive
- [x] Makefile builds all 4 service binaries
- [x] go.work includes all modules
- [x] Dockerfile fixed (no quality-service reference)
- [x] DEVELOPMENT.md updated

### Ready to Implement (ordered by first-slice-contracts.md Step sequence)

1. **Step 1** — ✅ DONE (this preparation): domain types + events + tests
2. **Step 2** — ✅ DONE (this preparation): NATS registries + stream definitions
3. **Step 3** — Next: `internal/adapters/exchanges/binancef/` (aggTrade parser)
4. **Step 4** — Next: `internal/application/ingest/` and `internal/application/derive/` (candle sampling logic)
5. **Step 5** — Next: `internal/actors/scopes/ingest/` and `internal/actors/scopes/derive/` (supervision trees)
6. **Step 6** — Next: Wire cmd/ingest and cmd/derive `Run()` functions
7. **Step 7** — Next: Gateway evidence routes + handlers + NATS adapter
8. **Step 8** — Next: Docker Compose smoke test

### Blocking Issues

None. The codebase is ready to receive first slice implementation.
