# Stage Report: Raccoon CLI Architecture Guardian

> Date: 2026-03-17
> Status: Complete
> Scope: Transform raccoon-cli from quality-service toolkit to market-foundry architecture guardian

---

## Objective

The raccoon-cli had been inherited from the quality-service era and still contained significant references to the old architecture (Kafka, validator, consumer, emulator). This stage transforms it into an active architecture guardian for the current market-foundry system.

---

## Changes Made

### 1. Identity Update

| Aspect | Before | After |
|--------|--------|-------|
| About text | "Engineering quality toolkit for quality-service" | "Architecture guardian toolkit for market-foundry" |
| Help references | `--project-root /path/to/quality-service` | `--project-root /path/to/market-foundry` |
| Test fixtures | `/nonexistent/quality-service` | `/nonexistent/market-foundry` |

### 2. Legacy Commands Deprecated

| Command | Status | Replacement |
|---------|--------|-------------|
| `runtime-smoke` | [DEPRECATED] | `make smoke` / `make smoke-multi` |
| `scenario-smoke` | [DEPRECATED] | `make smoke` / `make smoke-multi` |
| `results-inspect` | [DEPRECATED] | No replacement (validator removed) |
| `trace-pack` | [DEPRECATED] | No replacement |

### 3. Analyzer Rewrites

#### doctor.rs (surgical update)
- Changed "quality-service" references to "market-foundry"
- Updated config file expectations: `gateway.jsonc, ingest.jsonc, derive.jsonc, store.jsonc`

#### topology.rs (complete rewrite)
- **Stage enum**: `Nats, ConfigCtl, Gateway, Ingest, Derive, Store`
- **Service expectations**: 6 services matching current compose
- **Stream validation**: CONFIGCTL_EVENTS, OBSERVATION_EVENTS, EVIDENCE_EVENTS
- **Durable validation**: derive-observation, store-evidence
- **Subject validation**: observation.events.market.*, evidence.events.candle.*, evidence.query.*
- **Port expectations**: Only gateway (8080) and nats (4222, 8222) require host ports
- **13 checks** validating configs, compose, streams, subjects, durables, consistency

#### drift_detect.rs (complete rewrite)
- **6 drift checks**: config-compose, binary-compose, naming-identity, docs-reality, actor-scope, stream-registry
- Naming identity check detects residual "server" references where "gateway" is canonical
- Docs-reality check validates runtime-target.md against actual binaries
- Actor-scope check validates cmd/ directories match actor scopes

#### coverage_map.rs (complete rewrite)
- **7 quality dimensions**: project-structure, topology, contracts, runtime-bindings, architecture, drift, smoke-e2e
- **14 sensitive areas** aligned with current architecture (domain-observation, domain-evidence, actors-gateway, etc.)
- Removed old references to kafka-adapters, validator-logic, consumer-pipeline

#### runtime_bindings.rs + submodules (complete rewrite)
- **Source scanner**: Extracts JetStream streams, durables, subjects, adapter files from Go source
- **7 checks**: stream-ownership, consumer-binding, query-routing, config-source-alignment, adapter-presence, adapter-files, lifecycle-events
- Validates the complete ingest → derive → store pipeline

#### contracts.rs (targeted update)
- Subject-type convention check relaxed to domain-level consistency (warnings, not errors)
- Versioned type patterns accepted (e.g., `evidence.events.v1.candle_sampled`)
- Event-stream coverage and consumer-filter checks use warnings for parser limitations

#### gate/mod.rs (surgical update)
- Runtime-bindings remediation hint updated for NATS pipeline
- Runtime-smoke hint now references `make smoke`
- Test fixtures updated

### 4. Integration Tests

- `cli_integration.rs`: Updated help text assertion for new identity
- All 1089 tests pass (912 unit + 80 integration + 97 validation matrix)

### 5. Documentation

#### New documents
- `docs/tooling/cli-architecture-guardrails.md` — Defines deterministic vs heuristic guardrails
- `docs/tooling/cli-drift-rules.md` — Defines 6 drift detection rules with examples
- `docs/tooling/cli-topology-audit.md` — Defines expected topology and validation phases

#### Updated documents
- `docs/tooling/cli-overview.md` — Rewritten for architecture guardian role
- `tools/raccoon-cli/README.md` — Updated with new identity and deprecated commands

---

## Quality Gate Results

```
=== quality-gate [profile: fast] ===
  [+] doctor         PASS — 7 checks
  [+] topology-doctor PASS — 13 checks
  [+] contract-audit  PASS — 13 checks
  [+] runtime-bindings PASS — 7 checks
  [+] arch-guard      PASS — 11 checks
  [+] drift-detect    PASS — 6 checks
Result: PASSED | 6 passed, 0 failed | 57 checks | 171ms
```

---

## Architectural Invariants Now Protected

| Invariant | Check | Type |
|-----------|-------|------|
| Five-binary ceiling | drift-detect (binary-compose) | Deterministic |
| Layer sovereignty | arch-guard (11 rules) | Deterministic |
| Gateway is stateless | arch-guard (cmd boundary, port leaks) | Deterministic |
| Single stream ownership | runtime-bindings (stream-ownership) | Deterministic |
| Naming identity (gateway not server) | drift-detect (naming-identity) | Heuristic |
| Docs-code alignment | drift-detect (docs-reality) | Heuristic |
| Service topology correctness | topology-doctor (13 checks) | Deterministic |
| Pipeline connectivity | topology-doctor (pipeline-continuity) | Deterministic |
| Contract domain consistency | contract-audit (subject-type) | Heuristic |

---

## Legacy Residues Neutralized

| Residue | Action | Impact |
|---------|--------|--------|
| "quality-service" in about text | Replaced with "market-foundry" | Identity corrected |
| Old service names in topology checks | Rewritten for current services | No false positives |
| Kafka broker validation | Removed entirely | No longer applicable |
| Validator/consumer/emulator expectations | Removed entirely | No longer applicable |
| runtime-smoke assuming quality cluster | Deprecated, replaced by make smoke | No false failures |
| Old scenario names (happy-path, invalid-payload) | Deprecated | No longer referenced |
| Coverage map referencing validator-logic | Replaced with current sensitive areas | Accurate coverage map |

---

## Known Limitations

| Limitation | Impact | Mitigation |
|-----------|--------|-----------|
| Legacy modules still in source (smoke/, results_inspect/, trace_pack/) | Dead code in binary | Marked deprecated, compile-time only |
| Contract-audit stream subject parsing is incomplete | Some stream-event coverage checks use warnings | Pattern matching is best-effort |
| Subject-type convention check is heuristic | May produce false warnings on valid versioned types | Warnings not errors, domain-level check only |
| runtime-smoke in deep profile references old infrastructure | Deep profile's smoke step is not functional | Use `make smoke` instead |

---

## Next Steps Recommended

1. **Remove legacy modules entirely** — Delete `src/smoke/`, `src/results_inspect/`, `src/trace_pack/` and their command registrations
2. **Add make smoke integration** — New `runtime-smoke` that delegates to `make smoke` and parses output
3. **Enhance contract-audit** — Parse Go source more deeply to extract full stream subject definitions
4. **Add gateway pattern check** — Verify gateway binary has no JetStream subscriptions or domain logic
5. **Add stream ownership enforcement** — Verify each stream is published from exactly one binary's scope directory
