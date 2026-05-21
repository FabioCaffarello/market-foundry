# Control Gate Runtime Behavior: Findings and Limits

Status: documented (S273)

## Findings

### 1. KV-Backed State Transitions Are Immediate

Once `ControlKVStore.Put()` writes a new gate state, the very next `IsHalted()` call returns the updated value. There is no observable propagation delay within a single NATS connection. This means halt takes effect on the next intent evaluation — not batched or deferred.

### 2. Fail-Open Semantics Are Correct Under Missing Key

When no `global` key exists in the `EXECUTION_CONTROL` bucket, `Get()` returns `DefaultControlGate()` (active). This was verified at the runtime level (CG-RT-1), not just at the unit level. The system does not block execution when the control plane is uninitialized.

### 3. Audit Trail Fidelity

The `reason`, `updated_by`, and `updated_at` fields all survive the JSON serialization round-trip through NATS KV. This means the operational log of who halted and why is queryable from the KV store at any point.

### 4. Counter Accuracy

The `healthz.Tracker` counters (`processed`, `filled`, `skipped_halt`, `skipped_stale`) remain accurate across multiple halt/resume transitions within the same process lifetime. The counters are monotonic and never reset.

### 5. Halt Is Universal

During a halt, all intent types are blocked — both action intents (side=buy/sell) and no-action intents (side=none). This is by design: the kill switch is a blanket halt, not a selective filter.

## Limits

### 1. Single-Connection Scope

The runtime proof uses a single `ControlKVStore` connection per test. In production, the `store` binary writes and the `derive`/`execute` binaries read via separate connections. Cross-connection propagation latency is not measured (expected to be sub-millisecond on localhost, but not proven under network partition).

### 2. No Concurrent Writer Proof

The tests exercise sequential Put→Read cycles. They do not prove behavior under concurrent writes from multiple store binary replicas (not a current deployment topology, but worth noting).

### 3. No Bucket Deletion/Recreation Proof

If the `EXECUTION_CONTROL` bucket is deleted while the system is running, the fail-open behavior on read errors is proven at the unit level but not at the runtime level with a live NATS server.

### 4. No HTTP API Round-Trip

The tests write directly to `ControlKVStore`, bypassing the HTTP `PUT /execution/control` → NATS request → store binary → KV write path. The gateway-to-store-to-KV chain is not exercised here.

### 5. No Multi-Binary Proof

The full production path (gateway → store binary → KV → derive/execute binary) crosses process boundaries. This proof validates the KV-to-SafetyGate segment only.

### 6. Latency Characteristics Not Measured

While all tests complete in <200ms total, individual gate-read latency under load is not profiled. The 2s timeout on gate reads provides a safety margin but the actual distribution is unknown.

## Implications for Future Stages

- HTTP API round-trip proof would close limit #4 (requires running gateway + store binaries)
- Multi-binary integration test would close limit #5 (requires full deployment harness)
- Neither is required for the current hardening tranche — the KV-to-SafetyGate proof is sufficient to close the S269 debt
