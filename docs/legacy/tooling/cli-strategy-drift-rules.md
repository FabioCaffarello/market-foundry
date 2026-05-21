# Strategy Domain Drift Detection Rules

Raccoon-CLI enforces five strategy-specific drift detection rules (STD-1 through STD-5), analogous to the signal rules (SD-1 through SD-5) and decision rules (DD-1 through DD-5). These checks run as part of `raccoon-cli drift-detect` and the `quality-gate` profiles (fast/ci/deep).

## STD-1: Strategy Documentation Completeness

**Check name:** `strategy-docs-drift`

Verifies that all required strategy architecture documents exist:

| Document | Purpose |
|----------|---------|
| `docs/architecture/strategy-domain-design.md` | Canonical domain design, boundary invariants, and operational invariants |
| `docs/architecture/strategy-stream-families.md` | Stream family definitions (STF-01, STF-02, STF-03) |
| `docs/architecture/strategy-activation-and-ownership.md` | Activation model and ownership matrix |
| `docs/architecture/strategy-query-surface-guidelines.md` | Query chain and HTTP surface rules |
| `docs/architecture/strategy-readiness-review.md` | Pre-entry readiness assessment |
| `docs/architecture/strategy-entry-prerequisites.md` | Entry prerequisites checklist |
| `docs/architecture/strategy-risks-and-blockers.md` | Risk registry and mitigation plan |
| `docs/architecture/strategy-readiness-review-rerun.md` | Readiness review rerun after S52 |

**Severity:** ERROR if any doc is missing.

**Heuristic:** File existence only — the CLI does not validate document content or structure. A doc that exists but is empty or stale will pass this check.

## STD-2: Strategy Adapter File Presence

**Check name:** `strategy-adapter-drift`

Verifies that all expected NATS adapter files exist in `internal/adapters/nats/`:

| File | Role |
|------|------|
| `strategy_registry.go` | Stream, consumer, and query spec definitions |
| `strategy_publisher.go` | Publishes to STRATEGY_EVENTS |
| `strategy_consumer.go` | Durable consumer for store |
| `strategy_gateway.go` | NATS request/reply for gateway queries |
| `strategy_kv_store.go` | KV bucket materialization |

**Severity:** ERROR if any adapter file is missing.

## STD-3: Strategy Domain and Application Layer

**Check name:** `strategy-domain-drift`

Verifies structural completeness across three sublayers:

**Domain/Application files:**
- `internal/domain/strategy/strategy.go` — entity with Validate(), PartitionKey(), DeduplicationKey()
- `internal/domain/strategy/events.go` — StrategyResolvedEvent
- `internal/application/strategy/mean_reversion_entry_resolver.go` — mean reversion entry resolver
- `internal/application/strategyclient/contracts.go` — query/reply contracts
- `internal/application/strategyclient/get_latest_strategy.go` — GetLatestStrategy use case
- `internal/application/ports/strategy.go` — StrategyGateway port interface

**Actor scope files:**
- `internal/actors/scopes/derive/strategy_resolver_actor.go`
- `internal/actors/scopes/derive/strategy_publisher_actor.go`
- `internal/actors/scopes/store/strategy_consumer_actor.go`
- `internal/actors/scopes/store/strategy_projection_actor.go`

**HTTP interface files:**
- `internal/interfaces/http/handlers/strategy.go`
- `internal/interfaces/http/routes/strategy.go`

**Severity:** ERROR if any file is missing.

## STD-4: Strategy Configuration Symmetry

**Check name:** `strategy-config-drift`

Verifies that `pipeline.strategy_families` is declared symmetrically in both `derive.jsonc` and `store.jsonc`.

| State | Severity | Meaning |
|-------|----------|---------|
| Both present | INFO | Pipeline is correctly wired |
| derive only | ERROR | Events produced but never consumed |
| store only | ERROR | Consumer idles, no events produced |
| Neither | WARNING | Pipeline inactive (expected before implementation) |

**Note:** Unlike signal and decision families, `strategy_families` is expected to be absent from configs until S55/S56. The WARNING for "neither present" is informational, not actionable until implementation begins.

## STD-5: Strategy Runtime Contracts

**Check name:** `strategy-contracts-drift`

Scans Go source for expected runtime contracts:

**Subjects:**
- `strategy.events.mean_reversion_entry.resolved` — event subject for mean reversion entry strategies
- `strategy.query.mean_reversion_entry.latest` — query subject for latest strategy

**Durable consumers:**
- `store-strategy-mean-reversion-entry` — store consumer on STRATEGY_EVENTS

**KV buckets:**
- `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` — latest finalized strategy per partition key

**Severity:** ERROR if any contract is missing in source.

## Adding New Strategy Families

When a new strategy family is approved (e.g., `macd_momentum_entry` in S55+):

1. Add new subjects to `STRATEGY_EXPECTED_SUBJECTS` in `drift_detect.rs`
2. Add new durable to `STRATEGY_EXPECTED_DURABLES` in `drift_detect.rs`
3. Add new bucket to `STRATEGY_EXPECTED_BUCKETS` in `drift_detect.rs`
4. Add new durable to `EXPECTED_DURABLES` in `runtime_bindings.rs`
5. Add new query subject to `EXPECTED_QUERY_SUBJECTS` in `runtime_bindings.rs`
6. Update this document with the new family's contracts

## Relationship to Other Checks

These five checks complement:
1. **Base rules** (Rules 1–7) — config/compose/binary/docs/actor/stream alignment
2. **Signal rules** (SD-1 through SD-5) — signal domain structural and contract integrity
3. **Decision rules** (DD-1 through DD-5) — decision domain structural and contract integrity
4. **Strategy rules** (STD-1 through STD-5) — strategy domain structural and contract integrity

The strategy checks also integrate with:
- `runtime-bindings` — validates `STRATEGY_EVENTS` as a canonical stream
- `topology-doctor` — validates pipeline continuity (STRATEGY_EVENTS ↔ store-strategy-mean-reversion-entry durable)
- `coverage-map` — includes `domain-strategy` sensitive area
- `cross-config-family-consistency` — validates `strategy_families` alignment between derive and store configs

## Known Limitations

1. **Content-based drift** — The CLI checks file existence, not file contents. A registry file that exists but defines the wrong subjects will not be caught by drift-detect (but may be caught by runtime-bindings source scanning).
2. **Dependency chain validation** — The CLI does not enforce that `mean_reversion_entry` requires `rsi_oversold` in `decision_families`. Operator must ensure the full dependency chain is active.
3. **First-family bias** — Only `mean_reversion_entry` (STF-01) is governed. Additional families require explicit constant updates.
