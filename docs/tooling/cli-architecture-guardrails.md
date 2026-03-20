# CLI Architecture Guardrails

> Defines the architectural invariants that raccoon-cli enforces for market-foundry.
> Each guardrail is either deterministic (structural analysis) or heuristic (pattern matching).

---

## Deterministic Guardrails

These checks produce definitive pass/fail results based on structural analysis.

### Layer Boundary Enforcement (`arch-guard`)

| Rule | What It Checks | Enforcement |
|------|---------------|-------------|
| Layer dependency direction | domain → application → adapters → actors → interfaces | AST import analysis |
| Domain purity | `internal/domain/` has zero infrastructure imports | AST import scan |
| Application isolation | `internal/application/` does not import adapters directly | AST import scan |
| Interfaces isolation | HTTP handlers do not import adapters or actors | AST import scan |
| Cmd boundary | `cmd/` directories wire dependencies, do not define domain models | AST type counting |
| Tooling boundary | `tools/` contains no Go modules | Directory scan |
| No cross-cmd imports | Binary entrypoints are independently deployable | AST import scan |
| Deploy boundary | No hardcoded `deploy/` paths in Go source | Source text scan |
| Port contract leaks | Port interfaces do not reference infrastructure types | AST signature analysis |
| Domain type contamination | Struct fields do not embed infrastructure types | AST field analysis |
| Exported signature leaks | Domain/application exports do not expose infrastructure types | AST signature analysis |

### Project Structure (`doctor`)

| Check | What It Validates |
|-------|------------------|
| project-root | `go.work` exists at project root |
| required-dirs | `internal/`, `deploy/`, `tests/`, `tools/` exist |
| compose-file | `deploy/compose/docker-compose.yaml` exists |
| config-files | `deploy/configs/` contains `.jsonc` service configs |

### Service Topology (`topology-doctor`)

| Check | What It Validates |
|-------|------------------|
| config-presence | Core service configs exist and remain NATS-addressable (`configctl`, `gateway`, `ingest`, `derive`, `store`; `execute` when the execute surface is active) |
| config-nats | Each config has valid NATS configuration |
| compose-services | Docker Compose defines all expected services |
| compose-dependencies | Service dependency graph is correct |
| stream-definitions | Go source defines expected JetStream streams |
| subject-patterns | NATS subject patterns match canonical taxonomy |
| nats-url-consistency | NATS URLs are consistent across all configs |

---

## Heuristic Guardrails

These checks use pattern matching and may produce false positives in edge cases.

### Drift Detection (`drift-detect`)

| Check | What It Detects | Type |
|-------|----------------|------|
| config-compose drift | Config files without matching compose services (or vice versa) | Deterministic |
| binary-compose drift | cmd/ directories without matching compose services | Deterministic |
| naming-identity drift | Residual "server" references where "gateway" is canonical | Heuristic |
| docs-reality drift | Architecture docs mentioning services that don't exist (or missing ones that do) | Heuristic |
| actor-scope drift | Binaries without corresponding actor scope directories | Deterministic |
| stream-registry drift | JetStream stream names in source not matching canonical streams | Heuristic |

Post-S219 nuance:
- Store consumer governance no longer expects per-domain `*_consumer_actor.go` wrappers.
- Drift checks validate the generic store consumer infrastructure plus family wiring in `store_supervisor.go`.

### Contract Audit (`contract-audit`)

| Check | What It Validates | Type |
|-------|------------------|------|
| Registry completeness | EventRegistry entries have all required fields | Deterministic |
| Subject naming convention | Subjects follow `{domain}.{plane}.{aggregate}.{verb}` pattern | Heuristic |
| Envelope structure | Message envelopes have required metadata fields | Deterministic |
| Codec consistency | Encoding/decoding uses consistent codec (CBOR) | Heuristic |

Post-S218 nuance:
- Registry discovery scans both legacy `*_registry.go` files and the current `internal/adapters/nats/<domain>/registry.go` files.
- Consumer discovery accepts `ConsumerSpec{...}` blocks and `natskit.NewConsumerSpec(...)` factory calls.

### Coverage Map (`coverage-map`)

| Dimension | What It Covers |
|-----------|---------------|
| project-structure | Repository layout and config presence |
| topology | Service wiring and compose alignment |
| contracts | Messaging contract conformance |
| runtime-bindings | Config-to-stream routing |
| architecture | Layer boundary enforcement |
| drift | Cross-layer alignment |
| smoke-e2e | End-to-end pipeline validation (via `make smoke`) |

---

## When Checks Run

| Context | Profile | Commands Executed |
|---------|---------|-------------------|
| `make check` | fast | doctor, topology-doctor, contract-audit, runtime-bindings, arch-guard, drift-detect |
| `make verify` | fast + tests | Go tests + quality-gate fast |
| `make check-deep` | deep | All checks (runtime-smoke deprecated) |
| CI pipeline | ci | Same as fast, but warnings become failures |

---

## Adding New Guardrails

To add a new architectural check:

1. Implement in `tools/raccoon-cli/src/analyzers/` as a function returning `Result<Report>`
2. Add a step in `gate/mod.rs` for quality-gate inclusion
3. Register the command in `main.rs`
4. Document the invariant in this file
5. Add a Makefile target if the check should be independently runnable
