# Post-Paper CI and Validation Baseline

> Stage S88 — Documents the minimum CI/validation pipeline required to support the next frontier (real venue activation).
> Date: 2026-03-19
> Classification: DESIGN — no CI infrastructure changes implemented in this stage.

---

## 1. Purpose

The current validation surface (unit tests, smoke tests, drift rules) was built incrementally across S41–S87. Before the next frontier (real venue activation), the CI pipeline must be formalized as a coherent baseline with clear pass/fail gates. This document:

1. Inventories the current validation layers.
2. Identifies gaps in CI coverage.
3. Specifies the minimum pipeline for pre-venue confidence.
4. Defines the gate criteria for each pipeline stage.

---

## 2. Current Validation Inventory

### 2.1 Unit Tests

| Location | Scope | Count (approx.) |
|----------|-------|-----------------|
| `internal/domain/execution/` | Lifecycle transitions, validation, partition/dedup keys | ~30 |
| `internal/application/execution/` | Paper evaluator, simulator, venue adapter, staleness guard, pipeline chain | ~50 |
| `internal/actors/scopes/store/` | Execution/fill projection (gates, monotonicity, stats) | ~40 |
| `internal/application/risk/` | Position exposure evaluator | ~15 |
| `internal/shared/healthz/` | Health tracker, counter tests | ~10 |
| `internal/adapters/nats/` | Codec roundtrip, consumer dispatch | ~10 |
| **Total** | | **~155** |

**Run with**: `make test`

### 2.2 Smoke Tests (E2E)

| Script | Steps | Coverage |
|--------|-------|----------|
| `scripts/smoke-multi-symbol.sh` | 21 | Full pipeline: ingest → derive → store → execute → gateway |

**Coverage**: 2 symbols × 2 timeframes × all domain families (evidence, signal, decision, strategy, risk, execution, fill). Includes kill switch toggle, trace persistence, composite status validation.

**Run with**: `make up && make seed-multi && make smoke-multi`

### 2.3 Drift Detection

| Rule | Purpose |
|------|---------|
| config-compose-drift | Docker Compose ↔ configs alignment |
| binary-composition-drift | Makefile BUILDABLE_SERVICES ↔ compose |
| naming-identity-drift | No defunct service names in codebase |
| docs-reality-drift | Architecture docs mention all services |
| actor-scope-drift | Actor scope directories exist |
| stream-registry-drift | Canonical streams in adapters |
| premature-domain-entry | Prohibited streams not in source |
| signal-docs-drift | Signal architecture docs |
| signal-adapter-drift | Signal NATS adapters |
| signal-domain-drift | Signal domain files |
| signal-config-drift | Signal config alignment |
| signal-contracts-drift | Signal subjects/durables/buckets |
| decision-*-drift | Decision family (same pattern) |
| strategy-*-drift | Strategy family (same pattern) |
| risk-*-drift | Risk family (same pattern) |
| execution-docs-drift | 14 execution architecture docs |
| execution-adapter-drift | 7 NATS adapter files |
| execution-domain-drift | Domain + application layer files |
| execution-config-drift | Config alignment (derive, store, execute) |
| execution-contracts-drift | 6 subjects, 3 durables, 3 buckets |

**Run with**: `make quality-gate` (fast) or `make quality-gate-deep` (exhaustive)

### 2.4 Build Verification

| Target | Command | What it verifies |
|--------|---------|-----------------|
| `make build` | Compile all 6 binaries | Code compiles |
| `make docker-build` | Build Docker images | Dockerfile works |
| `make verify` | test + quality-gate | Tests pass + drift clean |

---

## 3. CI Pipeline Gaps

### 3.1 No Unified Pipeline Definition

**Gap**: There is no single CI pipeline configuration (e.g., GitHub Actions, Makefile target) that runs all validation layers in sequence with clear pass/fail.

**Current state**: Each layer is invoked independently. A developer must know to run `make test`, then `make quality-gate`, then `make up && make smoke-multi`.

**Impact**: A change can pass tests but fail drift detection. Or pass drift detection but fail smoke tests.

### 3.2 No Embedded NATS Integration Tests

**Gap**: Unit tests mock NATS interactions. No test spins up an embedded NATS server and validates KV monotonicity, consumer redelivery, dedup behavior, or inter-service message flow.

**Impact**: Paper mode masks NATS-specific failures. A real venue adapter would surface these failures under load.

**Status**: Identified in S86 (HB-POST-1), still unresolved.

### 3.3 No Config Validation Test

**Gap**: Config files (`deploy/configs/*.jsonc`) are not programmatically validated in CI. A typo in a config file is caught only at service startup.

**Impact**: A broken config file can deploy and fail at runtime.

### 3.4 No Docker Compose Health Check Validation

**Gap**: The smoke test starts docker-compose and validates health manually. There is no CI step that specifically validates all services reach `healthy` state within a timeout.

**Impact**: Silent service failures during compose startup are caught late in the smoke test.

### 3.5 No Regression Protection for Drift Rules

**Gap**: Drift rules are only checked when a developer runs `make quality-gate`. There is no pre-commit or CI hook to prevent drift from being committed.

---

## 4. Minimum CI Pipeline for Pre-Venue

### 4.1 Pipeline Stages

```
Stage 1: Build Gate
  ├── make build                    → all 6 binaries compile
  └── Exit on failure

Stage 2: Test Gate
  ├── make test                     → all unit tests pass
  └── Exit on failure

Stage 3: Quality Gate
  ├── make quality-gate             → all drift rules pass
  └── Exit on failure

Stage 4: Config Validation Gate (NEW)
  ├── Validate all deploy/configs/*.jsonc against schema
  └── Exit on failure

Stage 5: Docker Compose Gate (NEW)
  ├── make docker-build             → images build
  ├── make up                       → compose starts
  ├── Wait for all services healthy (timeout: 60s)
  └── Exit on failure (tear down compose)

Stage 6: Smoke Gate
  ├── make seed-multi               → seed test data
  ├── make smoke-multi              → 21-step validation
  └── Exit on failure (tear down compose)

Stage 7: Teardown
  └── make down
```

### 4.2 New Makefile Targets

```makefile
# Validate all JSONC config files against the application schema.
config-validate:
	@echo "Validating config files..."
	@for f in deploy/configs/*.jsonc; do \
		go run ./cmd/configctl validate --config "$$f" || exit 1; \
	done

# Run the full CI pipeline locally.
ci: build test quality-gate config-validate docker-build
	@echo "CI pipeline passed (compose + smoke require 'make ci-full')"

# Full CI pipeline including docker compose and smoke tests.
ci-full: ci
	$(MAKE) up
	@echo "Waiting for services to be healthy..."
	@sleep 15
	$(MAKE) seed-multi
	$(MAKE) smoke-multi
	$(MAKE) down
	@echo "Full CI pipeline passed."
```

### 4.3 Config Validation Command

The `configctl` binary already loads and validates configs. A `validate` subcommand formalizes this:

```
configctl validate --config deploy/configs/execute.jsonc
→ Exit 0 if valid, exit 1 with error details if invalid
```

This ensures config files are validated by the same `AppConfig.Validate()` code that services use at startup.

---

## 5. Embedded NATS Integration Test Design

### 5.1 Test Harness

```go
// internal/adapters/nats/integration_test.go (build tag: integration)

func TestWithEmbeddedNATS(t *testing.T) {
    // Start embedded NATS server with JetStream enabled
    srv := startEmbeddedNATS(t)
    defer srv.Shutdown()

    nc := connectToEmbedded(t, srv)
    js := jetStreamContext(t, nc)

    // Create streams and buckets matching production config
    createStreams(t, js)
    createBuckets(t, js)

    // Run test scenarios
    t.Run("KV monotonicity", testKVMonotonicity(js))
    t.Run("consumer redelivery", testConsumerRedelivery(js))
    t.Run("dedup key idempotency", testDedupIdempotency(js))
    t.Run("projection round-trip", testProjectionRoundTrip(js))
    t.Run("fill consumer delivery", testFillConsumerDelivery(js))
}
```

### 5.2 Key Test Scenarios

| Scenario | Validates |
|----------|----------|
| KV monotonicity | Put with older timestamp is rejected |
| KV monotonicity (same timestamp) | Put with same timestamp is deduplicated |
| Consumer redelivery | NAK'd message is redelivered up to MaxDeliver |
| Consumer dead letter | Message exhausting MaxDeliver is not redelivered |
| Dedup key idempotency | Same MsgId on publish is deduplicated by JetStream |
| Projection round-trip | Publish → consume → project → query returns correct data |
| Fill consumer delivery | Fill event reaches fill projection actor correctly |
| Multi-partition isolation | Two different partition keys don't interfere |

### 5.3 Build Tag Isolation

Integration tests use the `integration` build tag to prevent them from running in `make test`:

```go
//go:build integration
```

They are invoked separately:

```makefile
test-integration:
	go test -tags integration -count=1 -timeout 120s ./internal/adapters/nats/...
```

---

## 6. Pre-Venue CI Gate Criteria

Before the activation gate ceremony (S89), the following CI criteria must be met:

| # | Criterion | How Verified |
|---|-----------|-------------|
| CI-1 | All unit tests pass | `make test` exit 0 |
| CI-2 | All drift rules pass | `make quality-gate` exit 0 |
| CI-3 | All config files valid | `make config-validate` exit 0 |
| CI-4 | All services start healthy | Docker Compose health checks pass within 60s |
| CI-5 | Smoke tests pass | `make smoke-multi` exit 0 (21 steps, all PASS) |
| CI-6 | Integration tests pass | `make test-integration` exit 0 |
| CI-7 | No compilation warnings | `go vet ./...` exit 0 |
| CI-8 | No credential material in VCS | No `*.env` files tracked (only `*.env.example`) |

---

## 7. Gaps Closed by This Design

| Gap | Before S88 | After S88 |
|-----|-----------|-----------|
| Unified CI pipeline | Ad-hoc manual invocation | Documented 7-stage pipeline with Makefile targets |
| Config validation in CI | Not validated | `configctl validate` in pipeline |
| Embedded NATS tests | Not designed | Test harness with 8 scenarios |
| CI gate criteria | Implicit | 8 explicit criteria for pre-venue |
| Integration test isolation | Not considered | Build tag `integration` for separation |
| Docker Compose health gate | Not designed | Health check wait with timeout in ci-full |

---

## 8. What Remains Deferred

| Item | Reason | Earliest Stage |
|------|--------|---------------|
| GitHub Actions workflow file | Local pipeline first, CI service second | S89+ |
| Embedded NATS test implementation | Requires nats-server Go dependency | S89+ |
| configctl validate subcommand | Requires minor CLI extension | S89+ |
| Pre-commit hook for drift rules | Nice-to-have, not blocking | S89+ |
| Code coverage threshold | Establish baseline first | Future |
| Benchmark tests | Performance is not a current concern | Future |

---

## 9. Recommended Implementation Order

1. **`make config-validate`** — smallest change, highest value (catches config typos immediately).
2. **`make ci`** — combines existing targets into a single local gate.
3. **Embedded NATS test harness** — highest technical value, validates NATS assumptions.
4. **`make ci-full`** — adds compose + smoke to the pipeline.
5. **`make test-integration`** — integration tests as a separate gate.
