# CLI Signal Drift Detection Rules

## Overview

The `raccoon-cli drift-detect` command includes five signal-specific checks that detect drift between signal architecture documentation, source code, configuration, and runtime contracts.

## Rule Catalog

### SD-1: signal-docs-drift

**What it checks:** All required signal architecture documents exist.

**Required documents:**
1. `docs/architecture/signal-domain-design.md`
2. `docs/architecture/signal-first-slice.md`
3. `docs/architecture/signal-stream-families.md`
4. `docs/architecture/signal-activation-and-ownership.md`
5. `docs/architecture/signal-projection-pattern.md`
6. `docs/architecture/signal-query-surface-guidelines.md`
7. `docs/architecture/signal-replay-idempotency-rules.md`
8. `docs/architecture/signal-family-01-contracts.md`

**Severity:** Error for missing docs.

**Why:** Signal governance requires canonical design docs to prevent drift between architecture and implementation. Missing docs mean decisions are undocumented and the architecture becomes oral tradition.

---

### SD-2: signal-adapter-drift

**What it checks:** All expected signal NATS adapter files exist in `internal/adapters/nats/`.

**Expected files:**
| File | Purpose |
|------|---------|
| `signal_registry.go` | SIGNAL_EVENTS stream, consumer, and query specs |
| `signal_publisher.go` | Publishes signal events to SIGNAL_EVENTS |
| `signal_consumer.go` | Durable consumer for signal events in store |
| `signal_gateway.go` | Gateway adapter for NATS request/reply queries |
| `signal_kv_store.go` | KV bucket store for latest signal projections |

**Severity:** Error for missing files.

---

### SD-3: signal-domain-drift

**What it checks:** Signal domain, application, actor, and HTTP layer files exist.

**Checked layers:**
- **Domain:** signal.go, events.go
- **Application:** rsi_sampler.go, contracts.go, get_latest_signal.go, ports/signal.go
- **Actors (derive):** signal_sampler_actor.go, signal_publisher_actor.go
- **Actors (store):** signal_consumer_actor.go, signal_projection_actor.go
- **HTTP:** handlers/signal.go, routes/signal.go

**Severity:** Error for missing files.

---

### SD-4: signal-config-drift

**What it checks:** `pipeline.signal_families` appears symmetrically in both `derive.jsonc` and `store.jsonc`.

**Scenarios:**
| derive | store | Verdict |
|--------|-------|---------|
| present | present | OK — aligned |
| present | absent | Error — derive produces, store ignores |
| absent | present | Error — store listens, derive silent |
| absent | absent | Warning — signal inactive |

**Why:** A signal family must be activated in both derive and store to function. Asymmetry means either wasted computation or idle consumers.

---

### SD-5: signal-contracts-drift

**What it checks:** Signal subjects, durable consumers, and KV bucket names exist in Go source.

**Expected subjects:**
| Subject | Purpose |
|---------|---------|
| `signal.events.rsi.generated` | RSI signal event — derive publishes finalized RSI signals |
| `signal.query.rsi.latest` | RSI query — gateway queries store for latest RSI |

**Expected durable consumers:**
| Durable | Purpose |
|---------|---------|
| `store-signal-rsi` | Store consumes RSI signal events for projection |

**Expected KV buckets:**
| Bucket | Purpose |
|--------|---------|
| `SIGNAL_RSI_LATEST` | Stores latest finalized RSI signal per partition key |

**Severity:** Error for missing contracts.

**Why:** These are the runtime wiring contracts that connect derive → NATS → store → KV → gateway. Missing any of them breaks the signal pipeline silently.

---

## Adding New Signal Families

When adding a new signal family (e.g., MACD), update the following constants in `tools/raccoon-cli/src/analyzers/drift_detect.rs`:

1. **SIGNAL_EXPECTED_SUBJECTS** — add the new event and query subjects
2. **SIGNAL_EXPECTED_DURABLES** — add the new durable consumer
3. **SIGNAL_EXPECTED_BUCKETS** — add the new KV bucket

And in `tools/raccoon-cli/src/analyzers/runtime_bindings.rs`:

1. **EXPECTED_DURABLES** — add the new durable consumer with stream binding
2. **EXPECTED_QUERY_SUBJECTS** — add the new query subject

Also update `SIGNAL_DOCS` if the new family requires dedicated architecture documentation.

## Heuristics and Known Limitations

1. **Subject matching is substring-based** — the CLI checks if a subject string appears anywhere in source, not whether it's used in the correct registry struct. This could produce false positives if the same string appears in a comment or test.

2. **KV bucket scanning is directory-scoped** — only `internal/adapters/nats/` is searched for bucket names. If bucket names were defined elsewhere, they would not be detected.

3. **Config parsing is text-based** — the CLI checks for `signal_families` as a substring in JSONC files, not as a parsed JSON key. This is sufficient because JSONC config files follow a strict naming convention, but could theoretically match inside a comment or value string.

4. **No cross-validation of signal family names** — the CLI does not verify that the families listed in `signal_families: ["rsi"]` match the families whose adapters exist. This would require JSONC value parsing.
