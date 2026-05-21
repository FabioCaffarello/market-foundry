# Foundation Confidence Rules

> Rules governing when a foundational layer is considered trusted for upper-layer dependency.

## Context

Market Foundry's domain layers form a strict dependency chain:

```
strategy → decision → signal → evidence → observation
```

Upper layers inherit all risks from lower layers. A bug in observation serialization silently corrupts everything above. These rules define the minimum confidence bar a foundational layer must meet before upper layers may depend on it.

## Rules

### R-1: Domain Validation Coverage

Every domain value type must have tests covering:
- Valid construction (happy path)
- Each required field rejection (empty/zero)
- Temporal ordering invariants (close_time > open_time)
- Multi-error accumulation (completely empty struct produces all issues)

### R-2: Adapter Contract Coverage

Every NATS adapter (publisher, consumer, registry, KV store) must have tests covering:
- Subject taxonomy validation (lowercase, versioned types, wildcard coverage)
- Consumer spec bounds (MaxDeliver ∈ [1,10], AckWait > 0, hyphenated durables)
- Stream constraints (finite MaxAge, finite MaxBytes)
- Nil and uninitialized guard behavior (returns Unavailable, not panic)
- Key isolation across symbol × timeframe × source dimensions

### R-3: Translation Fidelity

Exchange adapter normalization must have tests covering:
- Malformed input rejection (bad JSON, wrong event type, empty payload)
- Decimal string preservation through normalization (no truncation, no formatting)
- Timestamp precision (millisecond → UTC conversion)
- Source identity enforcement (hardcoded per adapter)
- Symbol routing (parameter, not wire value)

### R-4: Query Surface Coverage

Every HTTP handler must have tests covering:
- Happy path (200 with valid response)
- Nil use case guard (503)
- Missing required parameters (400)
- Invalid parameter format (400)
- Null result handling (200 with null/empty, not 500)
- Use case error propagation (correct HTTP status)

### R-5: Deduplication Key Isolation

Every event type with JetStream deduplication must verify:
- Key format includes all partitioning dimensions
- Different sources with same entity ID produce different keys
- Different event types use distinct key prefixes (e.g., `burst:`, `vol:`)

## Application

Before opening a new domain layer (e.g., strategy), all layers below it must satisfy R-1 through R-5. The readiness review (e.g., S49, S52) checks these rules explicitly.

## Established By

- S49: Strategy Readiness Review (identified gaps)
- S50: Foundation Trust Recovery (closed BG-1, BG-2, BG-4, BG-5)
