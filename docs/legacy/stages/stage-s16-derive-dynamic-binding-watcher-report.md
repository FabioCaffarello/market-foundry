# Stage S16 — Derive Dynamic Binding Watcher

**Status:** Complete
**Objective:** Eliminate the need for manual `derive` restart when new ingestion bindings become active, by implementing dynamic binding activation driven by configctl runtime events.

---

## 1. Summary

S16 introduces a `BindingWatcherActor` into the derive scope, following the same proven pattern already established in the ingest scope. Before this change, the derive supervisor queried configctl for active bindings **only at startup** — any binding activated after the process started required a manual restart to take effect. Now, the derive process observes `IngestionRuntimeChangedEvent` via a dedicated JetStream durable consumer and activates new samplers in runtime without any restart.

### What Changed

| Layer | Before (S15) | After (S16) |
|-------|-------------|-------------|
| Binding discovery | Supervisor queries configctl once at startup | BindingWatcherActor queries at startup + subscribes to events |
| Dynamic activation | Not supported — restart required | Supported — new bindings activate in runtime |
| Actor tree | Supervisor → Consumer, SourceScopes | Supervisor → Consumer, BindingWatcher, SourceScopes |
| NATS consumer | `derive-observation` only | `derive-observation` + `derive-binding-watcher` |
| Ownership | Supervisor owned query + activation logic | Watcher owns discovery, Supervisor owns routing |

### Design Decisions

1. **Separate durable consumer name** (`derive-binding-watcher` vs `ingest-binding-watcher`): Each service maintains its own independent cursor on the configctl event stream. This ensures derive processes events at its own pace and can be restarted independently from ingest.

2. **Watcher as child of supervisor**: The binding watcher is spawned as a child actor of the derive supervisor, following hollywood's supervision tree. If the watcher crashes, the supervisor's lifecycle management handles it.

3. **Idempotent activation preserved**: The `SourceScopeActor.onActivateSampler()` already guards against duplicate symbol activation. The watcher may send the same binding multiple times (startup query + event), and the system handles it correctly.

4. **Non-fatal degradation**: If the JetStream subscription fails, the watcher logs the error and continues. The service operates with startup-state bindings, maintaining the same behavior as before S16.

---

## 2. Files Changed

### New Files
- `internal/actors/scopes/derive/binding_watcher_actor.go` — BindingWatcherActor with two-phase startup: initial configctl query + JetStream subscription for IngestionRuntimeChangedEvent

### Modified Files
- `internal/actors/scopes/derive/derive_supervisor.go` — Removed `queryAndActivateBindings()` method; supervisor now spawns BindingWatcherActor as child instead of querying configctl directly
- `internal/actors/scopes/derive/messages.go` — Updated `activateSamplerMessage` comment to reflect watcher origin
- `internal/adapters/nats/configctl_registry.go` — Added `DeriveBindingConsumer()` consumer spec with durable name `derive-binding-watcher`

### Unchanged (no modifications needed)
- `cmd/derive/run.go` — No changes; gateway is already passed to supervisor, which now propagates it to the watcher
- `deploy/configs/derive.jsonc` — No config changes; uses existing NATS configuration
- `internal/adapters/nats/binding_event_consumer.go` — Reused as-is; the adapter is generic over consumer spec
- `internal/actors/scopes/derive/source_scope_actor.go` — Idempotent activation already handles duplicate messages

---

## 3. Lifecycle and Ownership

```
DeriveSupervisor (root)
├── ConsumerActor ("observation-consumer")
│   └── owns: NATS durable consumer "derive-observation"
├── BindingWatcherActor ("binding-watcher")        ← NEW
│   └── owns: NATS durable consumer "derive-binding-watcher"
│   └── sends: activateSamplerMessage → supervisor
└── SourceScopeActor ("source-{exchange}")
    ├── EvidencePublisherActor ("publisher")
    └── SamplerActor ("sampler-{symbol}-{timeframe}s") × N
```

**Ownership rules:**
- The **BindingWatcherActor** owns the configctl event subscription and is responsible for closing it on shutdown.
- The **DeriveSupervisor** owns routing decisions: it receives `activateSamplerMessage` from the watcher and delegates to the appropriate source scope.
- **SourceScopeActor** owns sampler lifecycle: it spawns samplers and guards against duplicates.

---

## 4. Limitations

1. **No deactivation support yet**: When a scope is cleared (`IngestionRuntimeChangeCleared`), the event is logged but samplers are not stopped. Full deactivation requires tracking which bindings belong to which scope, which is deferred to a future stage.

2. **No reconciliation on reconnect**: If the watcher's JetStream subscription drops and reconnects, it relies on JetStream's `DeliverLastPerSubjectPolicy` to catch up. There is no explicit full-state reconciliation (re-querying configctl) on reconnect.

3. **Same event, dual consumers**: Both ingest and derive consume the same `IngestionRuntimeChangedEvent`. This is intentional — each service needs to react to binding changes independently. However, it means configctl publishes one event that drives two separate activation flows.

---

## 5. Validation Before S17

- [ ] **Runtime activation test**: Start derive, then activate a new binding via configctl. Verify that samplers for the new symbol appear in derive logs without restart.
- [ ] **Idempotency**: Activate the same binding twice. Verify no duplicate samplers are created.
- [ ] **Startup parity**: Verify that derive starts and activates the same bindings as before (behavioral equivalence with S15 for the startup path).
- [ ] **Graceful shutdown**: Verify that the binding watcher closes its NATS connection cleanly on SIGTERM.
- [ ] **Degradation**: Start derive without NATS available. Verify the watcher logs the failure and the service does not crash.
