# Control Plane Runtime Propagation — Findings

Status: documented (S275)
Scope: runtime behavior findings from full-path control plane validation

## Key Findings

### 1. Immediate Propagation Confirmed

Control surface writes (`ControlKVStore.Put()`) are visible to subsequent reads with zero observable delay. CP-FP-5 proved this with 10 rapid state alternations — every read-after-write returned the correct state. This is consistent with NATS KV's strong consistency model (single-node JetStream).

**Implication:** No cache invalidation or polling delay affects control plane decisions. The gate check at publish time reads the latest committed state.

### 2. Dual-Path Coherence Verified

Both gate check points (derive publisher path and venue adapter path) observe identical state when reading from the same `ControlKVStore` instance. CP-FP-4 proved this by toggling active/halted and verifying both paths block or allow in lockstep.

**Implication:** A single KV write is sufficient to halt all execution across both derive and execute pipelines.

### 3. Stream-Level Observability

The full-path proof verified that gate decisions are observable at the NATS stream level:
- Active gate: message appears on `execution.events.paper_order.submitted.>` with correct correlation ID (CBOR envelope decoded)
- Halted gate: no message appears (verified with 500ms wait — conservative for local NATS)

**Implication:** An external observer subscribing to the EXECUTION_EVENTS stream can infer gate state from message flow. This enables future operational monitoring without introspecting the KV bucket directly.

### 4. Fail-Open Semantics Preserved

The derive publisher path follows the same fail-open contract as the venue adapter path:
- `ControlKVStore` nil → publish allowed (no gate check)
- KV key missing → `DefaultControlGate()` returned (active)
- KV read error → `IsHalted()` returns false (active)

**Implication:** Control plane degradation (NATS KV unavailable) does not halt the pipeline. This is a deliberate design choice — availability over safety for the control gate. Real-money execution would likely invert this (fail-closed).

### 5. Deduplication Key Isolation

Each test event uses a monotonic sequence counter to ensure unique JetStream dedup keys across test runs. This avoids false negatives where JetStream silently deduplicates a message that was published in a prior run within the dedup window (default: 2 minutes).

**Implication:** When testing against a persistent NATS server, dedup keys must be managed carefully. Production intents use timestamp-based dedup keys which are naturally unique.

## Operational Topology (Proven)

```
Control Write Path:
  operator / risk actor → ControlKVStore.Put(halted/active) → EXECUTION_CONTROL KV bucket

Control Read Path (Derive):
  ExecutionPublisherActor → ControlKVStore.IsHalted(ctx) → block or publish

Control Read Path (Execute):
  VenueAdapterActor → SafetyGate.Check() → ControlKVStore.IsHalted(ctx) → block or submit

Observable Effect:
  EXECUTION_EVENTS stream → messages present (active) or absent (halted)
```

## Limits and Open Items

### Not Proven by S275

1. **Gateway surface integration**: The `ControlGateway` (request/reply via `execution.control.set`) has not been exercised in a full-path test. S275 proves the KV write path directly, not through the gateway's NATS request/reply surface.

2. **Multi-node consistency**: All proofs run against a single NATS server. Clustered JetStream behavior (quorum writes, read-after-write across nodes) is not validated.

3. **Concurrent writer safety**: Only one writer changes the gate at a time. No test proves behavior when two writers race (last-write-wins is the expected contract, but not proven under contention).

4. **Watch/notification**: The control gate uses poll-on-read semantics. There is no KV watcher that pushes state changes to consumers. Each gate check is a fresh read.

5. **Audit log durability**: The `reason`, `updated_by`, and `updated_at` fields survive the KV round-trip (proven by S273 CG-RT-5), but there is no append-only audit log of gate transitions. Each Put overwrites the previous state.

### Pre-Existing Issue Noted

The S273 test `TestControlGateRuntime_DefaultState_FailOpen_IntentFlows` (CG-RT-1) can fail when run against a NATS server that retains KV data from prior test runs, because it assumes the `EXECUTION_CONTROL` bucket has no key. This is a test isolation issue, not a product bug.
