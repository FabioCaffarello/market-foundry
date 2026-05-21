# Minimal Real Venue Adapter Contracts

> **Stage:** S90
> **Date:** 2026-03-19
> **Status:** Contracts defined, no adapter implemented

---

## 1. VenuePort Interface

The existing `VenuePort` interface is the adapter boundary:

```go
type VenuePort interface {
    SubmitOrder(ctx context.Context, req VenueOrderRequest) (VenueOrderReceipt, *problem.Problem)
}
```

### Input: VenueOrderRequest

```go
type VenueOrderRequest struct {
    Intent execution.ExecutionIntent
}
```

The intent carries: type, source, symbol, timeframe, side, quantity, risk input, correlation/causation IDs.

### Output: VenueOrderReceipt

```go
type VenueOrderReceipt struct {
    VenueOrderID string
    Status       execution.Status
    Intent       execution.ExecutionIntent  // Mutated with fill data
}
```

## 2. Adapter Implementation Contract

Any real venue adapter MUST satisfy these invariants:

### INV-1: Context Respect
The adapter MUST honor the context deadline. If the context expires, the adapter MUST return an error immediately. The caller (VenueAdapterActor) provides a configurable timeout context.

### INV-2: No Gate Bypass
The adapter MUST NOT bypass kill switch or staleness guard. Those checks happen in the actor layer before `SubmitOrder` is called. The adapter receives only pre-validated intents.

### INV-3: Credential Isolation
API credentials MUST be injected at construction time via `CredentialSet`. The adapter MUST NOT read environment variables directly or store credentials in any mutable state.

### INV-4: Problem Classification
The adapter MUST classify errors using the problem package:
- `problem.Unavailable` — transient errors (network timeout, rate limit, exchange maintenance).
- `problem.InvalidArgument` — permanent errors (invalid symbol, unsupported order type).
- `problem.Internal` — unexpected errors (JSON parse failure, unknown API response).

### INV-5: Fill Completeness
For filled orders, the returned `Intent` MUST have:
- `Status` set to `StatusFilled` (or `StatusPartiallyFilled` for partial fills).
- `FilledQuantity` reflecting the actual filled amount.
- `Fills` array with at least one `FillRecord` containing real price, quantity, fee, and `Simulated: false`.

### INV-6: VenueOrderID
Every submitted order MUST produce a unique `VenueOrderID` that corresponds to the exchange's order identifier. This ID is used for:
- Fill deduplication (`fill:{venue_order_id}:{timestamp}`).
- Operational debugging and audit trail.
- Future reconciliation (matching venue fills to platform intents).

### INV-7: Side-None Handling
Intents with `Side: "none"` represent no-action decisions. The adapter SHOULD return `StatusAccepted` without submitting to the exchange.

## 3. Registration Contract

To activate a new venue adapter, the following registration steps are required:

1. **Settings schema:** Add venue type constant to `knownVenueTypes` map.
2. **buildVenueAdapter:** Add case in `cmd/execute/run.go` to construct the adapter with credentials.
3. **Config:** Update `execute.jsonc` with the new `venue.type`.
4. **Drift rules:** Update raccoon-cli with the new venue type in registries.
5. **Documentation:** Create `docs/architecture/venue-{name}-adapter-design.md`.
6. **Tests:** Unit tests with mock HTTP server, integration test with embedded NATS.

## 4. Scope Constraints

The first real venue adapter is intentionally minimal:

| Feature | In Scope | Out of Scope |
|---------|----------|--------------|
| Market orders | Yes | Limit, stop, OCO |
| Single exchange | Yes | Multi-venue routing |
| Synchronous fills | Yes | Async fill tracking |
| Full fills | Yes | Partial fill accumulation |
| Single symbol at a time | Yes | Batch/portfolio orders |
| API key auth | Yes | OAuth, IP whitelisting |
| Testnet first | Yes | Production API |

These constraints match the S89 gate's "extremely guarded first step" directive.
