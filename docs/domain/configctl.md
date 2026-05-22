# configctl — Configuration lifecycle

The `configctl` domain owns the lifecycle of configuration documents
in market-foundry. It is the single authority over configuration state
transitions; no other domain or binary may transition a config.

This is an **outlier** among family domains: while observation, evidence,
signal, etc., are about market data, configctl is about governance.

---

## What this domain models

A configuration document is a versioned, structured payload that
describes runtime behavior: which symbols to observe, which evidence
to derive, which signals to compute, which strategies to evaluate,
which risks to apply, and which executions to wire. It progresses
through a defined lifecycle:

```
Draft → Validated → Compiled → Active → Inactive → Archived
                                            │
                                            └→ Rejected (alternative terminal from any prior state)
```

Only configctl performs these transitions. All other binaries (ingest,
derive, store, execute, writer) react to **events** that configctl
publishes when transitions happen.

---

## Core types

### Lifecycle and versioning

- **`VersionLifecycle`** (string enum in `lifecycle.go`):
  `Draft`, `Validated`, `Compiled`, `Active`, `Inactive`, `Archived`, `Rejected`.
- **`ConfigVersion`** (`config_set.go`): one revision of a configuration.
- **`ConfigSet`** (`config_set.go`): a set of related versions with a
  `ValidateVersion(versionID, validatedAt)` method that returns
  `([]ValidationDiagnostic, *problem.Problem)`.

### Document content

- **`ConfigSource`** (`document.go`): raw source with a format hint;
  `ValidateForDraft()` returns `*problem.Problem`.
- **`SourceFormat`** (string enum): supported source formats.
- **`ConfigDocument`** (`document.go`): parsed and structured document;
  `Validate()` returns `[]ValidationDiagnostic` (multi-issue — see
  Known anomalies).
- **`ConfigMetadata`** (`document.go`): scope and identity metadata.
- **`Binding`** (`document.go`): an attachment between a config and a
  runtime surface (ingest binding, derive binding, etc.).
- **`Field`** with **`FieldType`** enum (`document.go`): typed config field.
- **`Rule`** with **`RuleOperator`** and **`RuleSeverity`** enums
  (`document.go`): a single validation rule.
- **`ValidationDiagnostic`** (`document.go`): one diagnostic produced by
  validation (carries field, severity, message).

### Runtime projection

- **`CompilationArtifact`** (`runtime.go`): output of compilation step.
- **`ActivationScope`** (`runtime.go`): scope of an activation;
  `Validate()` returns `*problem.Problem` (canonical pattern).
- **`Activation`** (`runtime.go`): a specific activation instance.
- **`RuntimeProjection`** (`runtime.go`): aggregate runtime view of
  active configs.
- **`IngestionRuntimeProjection`** (`runtime.go`): runtime view scoped
  to ingest.

### Events

8 event types (see "Event flow" below).

---

## State machine

Configctl is one of two family domains (the other is execution) with
an explicit state machine. Transitions are exposed as separate HTTP
endpoints — there is no implicit progression.

| Transition | Trigger | Result event |
|---|---|---|
| → Draft | `POST /configctl/configs` | `DraftCreatedEvent` |
| Draft → Validated | `POST /configctl/config-versions/:id/validate` | `ConfigValidatedEvent` |
| Validated → Compiled | `POST /configctl/config-versions/:id/compile` | `ConfigCompiledEvent` |
| Compiled → Active | `POST /configctl/config-versions/:id/activate` | `ConfigActivatedEvent` (and `IngestionRuntimeChangedEvent` if bindings change) |
| Active → Inactive | (deactivation operation) | `ConfigDeactivatedEvent` |
| Inactive → Archived | (archive operation) | `ConfigArchivedEvent` |
| any → Rejected | validation/compilation failure | `ConfigRejectedEvent` |

The `VersionLifecycle` enum uses `Inactive` for the deactivated state
even though the event type is named `ConfigDeactivatedEvent` — a minor
naming inconsistency between the lifecycle constant and the event name.

---

## Event flow

### Streams

- **Writer:** `configctl` binary (single writer)
- **Stream:** `CONFIGCTL_EVENTS`
- **Consumers:** `ingest` (via `ingest-binding-watcher` durable),
  `derive` (via `derive-binding-watcher` durable)

### Event types

| Event name (`events.Name` constant) | Type | Published when |
|---|---|---|
| `config.draft_created` | `DraftCreatedEvent` | A new draft is persisted |
| `config.validated` | `ConfigValidatedEvent` | Validation succeeds for a draft |
| `config.compiled` | `ConfigCompiledEvent` | Compilation succeeds for a validated version |
| `config.activated` | `ConfigActivatedEvent` | A compiled version is activated |
| `config.deactivated` | `ConfigDeactivatedEvent` | An active version is deactivated |
| `config.archived` | `ConfigArchivedEvent` | An inactive version is archived |
| `config.rejected` | `ConfigRejectedEvent` | Validation or compilation fails terminally |
| `config.ingestion_runtime_changed` | `IngestionRuntimeChangedEvent` (with `IngestionRuntimeChangeType` ∈ {`activated`, `cleared`}) | Ingest bindings change as a side effect of activation/deactivation |

Each event type implements the `DomainEvent` interface (`EventName()`
and `EventMetadata()` accessors).

### Subject taxonomy

configctl is the source of the **known inconsistency** noted in
[`../RUNTIME.md`](../RUNTIME.md): both singular (`configctl.event.*`)
and plural (`configctl.events.*`) subject patterns coexist. This is
transitional surface — a partial migration that never completed.

When working with configctl subjects, consult the registry directly:

```
internal/adapters/nats/natsconfigctl/registry.go
```

---

## Adapters

| Adapter | Location | Purpose |
|---|---|---|
| NATS | `internal/adapters/nats/natsconfigctl/` | Stream registration, publisher, consumer specs, binding-watcher consumer |
| Application | `internal/application/configctl/` | Use cases: `create_draft`, `validate_draft`, `validate_config`, `compile_config`, `activate_config`, `deactivate_config`, `reject_config`, `archive_config`, plus repository + memoryrepo |
| ClickHouse | _none_ | configctl events are not analytical history |

The absence of a ClickHouse adapter is intentional. Configuration state
is operational, not analytical — it lives in NATS KV and in-flight
events. Past configs can be reconstructed by replaying the stream.

---

## HTTP surface

configctl exposes 8 endpoints through the gateway. See
[`../HTTP-API.md`](../HTTP-API.md) → "configctl" group for details.

The endpoints map one-to-one to lifecycle operations: POST to drive
transitions, GET to inspect state.

---

## Known anomalies and patterns

### `ConfigDocument.Validate()` returns `[]ValidationDiagnostic`

The canonical `Validate()` signature in market-foundry returns
`*problem.Problem` (a single rich error). `ConfigDocument` breaks this
to return `[]ValidationDiagnostic` — a list of structured diagnostics.

**Why:** compiling a config can yield multiple independent problems
(e.g., "symbol X is unknown" AND "strategy Y references unknown
decision Z"). Reporting one at a time would force iterative recompiles.
Multi-diagnostic reporting is intentional here.

`ConfigSet.ValidateVersion()` shares the same shape, returning
`([]ValidationDiagnostic, *problem.Problem)` — diagnostics for
content issues plus a problem for operational failures.

`ConfigSource.ValidateForDraft()` and `ActivationScope.Validate()` do
follow the canonical pattern (single `*problem.Problem`), so both
shapes coexist in this domain depending on the use site.

### Lifecycle vs. event naming inconsistency

`VersionLifecycle` uses `Inactive` for the post-active state; the event
emitted on that transition is named `ConfigDeactivatedEvent` with
event name `config.deactivated`. The lifecycle constant and the event
verb don't match. This is a documentation-level inconsistency, not a
code defect, but it can confuse readers expecting `Deactivated` as a
lifecycle state.

### configctl as outlier

Other family domains follow a uniform shape: domain types + event types
+ NATS adapter + application package. configctl shares this shape but
its **content** is lifecycle governance, not market data. Some
abstractions used for evidence/signal/decision do not apply (e.g.,
there is no `FamilyProcessor` for configctl — its derivation is
projection compilation, done by a dedicated path).

---

## Reading further

| If you want | Go to |
|---|---|
| The HTTP endpoints | [`../HTTP-API.md`](../HTTP-API.md) |
| The configctl binary's wiring | `cmd/configctl/run.go` |
| Subject namespace inconsistency context | [`../RUNTIME.md`](../RUNTIME.md) |
| Cross-references to other domains | [Domain README](README.md) |
