# Expansion Playbooks — Refined

> Refined playbooks for expanding the market-foundry monorepo. Based on concrete experience from S96–S104 consolidation and the full vertical slice implementation (candle → rsi → rsi_oversold → mean_reversion_entry → position_exposure → paper_order).

---

## Design Principles for These Playbooks

1. **Playbooks are decision aids, not substitutes for judgment.** They reduce the cost of doing the right thing, not the need to think.
2. **Each playbook assumes the developer has read the prerequisites.** They are not tutorials.
3. **Worked examples reference real history.** Abstract patterns mislead; concrete past decisions calibrate.

---

## Playbook 1: Adding a New Family (Same Domain)

This is the most common expansion. All 12 current families were added this way.

### Decision Gate

Before starting, answer these three questions:

| Question | If "No" |
|----------|---------|
| Does this family represent a distinct processing specialization within an existing domain? | Consider whether it's actually a new domain (Playbook 3) or a variant of an existing family. |
| Does the family have its own event type (distinct `*Event` struct)? | It's not a family — it's a configuration variant of an existing family. |
| Can you name the upstream family it depends on in the dependency chain? | The dependency chain must be explicit: evidence → signal → decision → strategy → risk → execution. |

### Steps (Ordered by Layer)

**Layer 1 — Domain** (`internal/domain/{domain}/`)
1. Create `{family}.go` with the domain entity: `Validate()`, `PartitionKey()`, `DeduplicationKey()`.
2. Create or extend `events.go` with the family's event type.
3. The event must implement `events.Event` with proper metadata.

**Layer 2 — Settings** (`internal/shared/settings/schema.go`)
1. Add family name to `knownXxxFamilies` set.
2. If this is NOT an evidence family: add upstream dependency in the appropriate `xxxDependsOnYyy` map.
3. Run `make test` — the duplicate detection and dependency graph tests will catch misconfiguration.

**Layer 3 — NATS Adapters** (`internal/adapters/nats/`)
1. Add registry entries: EventSpec, ControlSpec, ConsumerSpec.
2. Add KV bucket constant if the family needs materialized views.
3. Naming: durable = `{domain}-{family}-{runtime}`, bucket = `{DOMAIN}_{FAMILY}_LATEST`.

**Layer 4 — Application** (`internal/application/`)
1. If the family is derived (not evidence): create processor in `internal/application/{domain}/`.
2. Create client use case in `internal/application/{domain}client/`.
3. Create or extend port interface in `internal/application/ports/{domain}.go`.

**Layer 5 — Actors**
1. **Derive**: Add one entry to the scope-specific processor slice in `derive_supervisor.go:start()`.
2. **Store**: Add one `Pipeline` entry in `store_supervisor.go:declarePipelines()`.
3. **Store query**: Add handler in `query_responder_actor.go` for the existing scope.

**Layer 6 — Gateway** (if HTTP-queryable)
1. Add one field to `gatewayConns` in `cmd/gateway/compose.go`.
2. Add one `newGatewayConn()` call in `buildGatewayConns()`.
3. Add route + handler in `internal/interfaces/http/`.

**Layer 7 — Configuration**
1. Add family to relevant service JSONC configs in `deploy/configs/`.
2. Families are always config-driven — never hardcoded activation.

### Validation Sequence

```
make test          # catches settings validation issues early
make arch-guard    # layer boundary check
make verify        # full quality gate
```

### Worked Example: Adding `rsi` Signal Family (S35–S41)

The RSI signal family was the first non-evidence family added. Key decisions:
- Created `internal/domain/signal/signal.go` with `Signal` entity.
- Added `SignalFamilyProcessor` type in derive (distinct from `FamilyProcessor` because signal processors receive different constructor params).
- Added `rsi` to `knownSignalFamilies` and `signalDependsOnEvidence` map.
- The store pipeline entry used the existing `DomainSignal` scope — no new scope needed.

**What went right:** Following the catalog pattern made the store side trivial — one entry in `declarePipelines()`.
**What nearly went wrong:** The derive processor type was initially copied from `FamilyProcessor`. It was corrected to `SignalFamilyProcessor` because signal processors need a `signalPublisherPID`, not a generic `publisherPID`. This type-level distinction prevents runtime routing bugs.

---

## Playbook 2: Adding a New Venue Adapter

Venue adapters are intentionally NOT catalog-driven (security implications of auto-discovery). Each venue requires explicit registration.

### Decision Gate

| Question | If "No" |
|----------|---------|
| Is the venue supported by an approved exchange API? | Stop. Venue approval is a product decision, not an engineering decision. |
| Do you have testnet/sandbox credentials for initial development? | Do not develop against production APIs. |
| Is the execution domain already running end-to-end with paper_order? | The paper simulator must work first — it validates the execution pipeline. |

### Steps

**Layer 1 — Domain** (`internal/domain/execution/`)
1. If the venue has a distinct fill event shape: extend `events.go` (e.g., `VenueOrderFilledEvent` already exists).
2. If the venue uses the same fill shape as existing venues: no domain changes needed.

**Layer 2 — Application** (`internal/application/execution/`)
1. Create `{venue}_adapter.go` implementing `ports.VenuePort`.
2. The adapter must handle: order submission, fill event construction, error mapping to `*problem.Problem`.
3. Add staleness guard integration if the venue has latency characteristics requiring it.

**Layer 3 — Adapters** (`internal/adapters/exchanges/`)
1. Create `{venue}/` package with the exchange client.
2. This is the only place where exchange SDK imports are allowed.
3. The adapter translates between exchange SDK types and domain types.

**Layer 4 — Settings** (`internal/shared/settings/schema.go`)
1. Add the venue type to `knownVenueTypes` (the `VenueType` constants).
2. Add any venue-specific configuration fields to `VenueConfig` if needed.

**Layer 5 — Runtime Wiring** (`cmd/execute/run.go`)
1. Add a case to `buildVenueAdapter` switch statement.
2. The switch is intentionally explicit — no registry, no reflection, no auto-discovery.

**Layer 6 — Configuration**
1. Add venue config to `deploy/configs/execute.jsonc`.
2. Document any required environment variables (API keys, endpoints) in the config.

### Validation

```
make test          # adapter unit tests
make arch-guard    # verify exchange SDK imports stay in adapters/exchanges/
make verify        # full quality gate
```

### Security Considerations

- API credentials must NEVER appear in code, configs checked into git, or environment defaults.
- The venue adapter switch in `run.go` is the security boundary — adding a venue requires a deliberate code change, not a config toggle.
- Venue families in execution (`paper_order`, `venue_market_order`) are config-activated, but the venue adapter itself is wiring-level.

---

## Playbook 3: Adding a New Domain

Adding a domain is a significant structural decision. Market Foundry currently has 8 domains. The last domain added was `execution` (S68–S84), which took 16 stages.

### Decision Gate

| Question | If "No" |
|----------|---------|
| Does this represent a genuinely distinct bounded context with its own ubiquitous language? | It's probably a family within an existing domain. |
| Does the new domain have at least one family ready for implementation? | Do not create empty domains. A domain without a family is speculative architecture. |
| Can you define the domain's position in the dependency chain? | Every domain must have a clear upstream/downstream relationship. |
| Does the domain require its own JetStream stream? | If it shares a stream with an existing domain, it's likely a family, not a domain. |

### Steps

All steps from the existing `how-to-introduce-new-runtimes-domains-and-families.md` apply, plus:

1. **Add `PipelineDomain` constant** to `internal/actors/scopes/store/store_supervisor.go`.
2. **Add new processor type** in derive if the domain's processor constructor signature differs from existing types.
3. **Add publisher actor** in derive for the new scope.
4. **Add routing method** in `source_scope_actor.go`.
5. **Add registry type** in NATS adapter.
6. **Add `IsXxxFamilyEnabled()` method** on `PipelineConfig`.
7. **Add cross-layer dependency validation** in settings.
8. **Add scope to `queryResponderConfig()`** in store supervisor.
9. **Add domain governance constants** in `drift_detect.rs` for raccoon-cli enforcement.

### What This Costs

From observed history:
- A new domain touches ~15–20 files across all layers.
- The derive supervisor gains a new processor type and publisher actor.
- The store supervisor gains scope-level registry injection.
- raccoon-cli drift-detect gains ~50 lines of governance constants.
- Configuration gains a new family list with dependency validation.

This is intentionally expensive. The cost prevents speculative domain creation.

---

## Playbook 4: Adding a New Runtime (Service)

This is the rarest expansion. Market Foundry has 6 runtimes. No new runtime has been added since `execute` (S68).

### Decision Gate

| Question | If "No" |
|----------|---------|
| Does this service have a distinct operational concern that cannot be served by any existing runtime? | Add the functionality to an existing runtime. |
| Does the service need independent scaling or deployment? | If it always deploys with another service, it should be part of that service. |
| Can you justify the operational overhead of another binary, config, and health endpoint? | The marginal cost of a new runtime is real: deployment config, docker-compose entry, health monitoring, log streams. |

### Steps

Follow `runtime-assembly-guidelines.md` for the 6-phase lifecycle. Additionally:

1. Create `cmd/{runtime}/` with `main.go`, `run.go`, and optionally `compose.go`.
2. Follow the canonical `bootstrap.Main("{runtime}", Run)` entry point.
3. Add health server with `NATSReadinessCheck` if NATS-dependent.
4. Register in `go.work`, `Makefile`, `docker-compose.yaml`.
5. Add JSONC config in `deploy/configs/`.
6. Update `AGENTS.md` and `DEVELOPMENT.md` service tables.
7. Update `APP_BINARIES` in `drift_detect.rs`.

### Anti-Pattern: One-Concern Runtimes

Do not create a runtime for a single use case. The `execute` runtime handles both order submission (paper_order family) and venue fill processing (venue_market_order family) because they share operational context (venue adapter, control gate).

---

## Playbook 5: Adding a New Adapter Technology

Adapters implement port interfaces. Current adapter groups: `nats`, `exchanges`, `repositories`.

### Decision Gate

| Question | If "No" |
|----------|---------|
| Is this a genuinely new technology boundary (not just a new instance of NATS, a new exchange, or a new repository)? | Add to the existing adapter group. |
| Does the technology require its own `go.mod` (external SDK dependency)? | It can live within an existing adapter module. |

### Steps

1. Create `internal/adapters/{technology}/` with its own `go.mod`.
2. Implement the port interface from `internal/application/ports/`.
3. Add to `go.work`.
4. Wire in the appropriate runtime's composition root.

### Naming

- Adapter package names reflect the technology, not the domain: `nats`, `exchanges`, `repositories`.
- The package provides the technology-specific implementation; the port interface defines the domain contract.

---

## Cross-Cutting: Configuration Discipline

Every expansion must follow these configuration rules:

1. **Families are config-activated** — never hardcoded `if family == "candle"` logic.
2. **Config validation runs at startup** — invalid configs fail fast with `*problem.Problem`.
3. **Dependency validation is cross-layer** — enabling `rsi` signal without `candle` evidence fails validation.
4. **Duplicate detection is automatic** — `rejectDuplicates()` catches `["candle", "candle"]`.
5. **Binding topic format is validated** — must match `source.symbol` convention.

## Cross-Cutting: Raccoon-CLI Governance Updates

When adding a new domain or significant family, update raccoon-cli:

1. **drift-detect.rs**: Add domain governance constants (docs, subjects, durables, buckets, adapter files, domain files).
2. **topology.rs**: Add expected streams and durables to topology validation.
3. **coverage-map.rs**: Add the domain to sensitive areas if it warrants independent coverage tracking.

This is part of the expansion cost, not an afterthought.

---

## Related Documents

- `how-to-introduce-new-runtimes-domains-and-families.md` — original playbook (unchanged, remains valid)
- `family-runtime-registration-rules.md` — detailed per-runtime checklists
- `runtime-assembly-guidelines.md` — runtime lifecycle phases
- `naming-conventions-for-domains-families-and-runtimes.md` — binding naming rules
- `config-activation-and-dependency-map-model.md` — config activation model
- `structural-anti-patterns-and-when-not-to-expand.md` — when NOT to expand
