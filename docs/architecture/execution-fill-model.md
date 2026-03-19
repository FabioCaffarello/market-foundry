# Execution Fill Model

> Introduced in S77. Defines the minimal fill record structure and fill semantics for the execution domain.

## FillRecord

Each fill event within an execution is represented by a `FillRecord`:

| Field       | Type      | Description                                         |
|-------------|-----------|-----------------------------------------------------|
| `price`     | string    | Fill price (decimal string, "0" in paper mode)      |
| `quantity`  | string    | Quantity filled in this record (decimal string)      |
| `fee`       | string    | Fee charged for this fill (decimal string, "0" in paper mode) |
| `simulated` | bool      | `true` for paper fills, `false` for real venue fills |
| `timestamp` | time.Time | When the fill occurred                              |

## ExecutionIntent Fill Fields

Two fields were added to `ExecutionIntent`:

| Field             | Type           | Description                                           |
|-------------------|----------------|-------------------------------------------------------|
| `filled_quantity` | string         | Cumulative quantity filled across all fill records     |
| `fills`           | `[]FillRecord` | Ordered list of fill records for this execution       |

## Fill Semantics

### Paper Mode (current)

- **Full fill**: Every actionable order (side = buy/sell) receives exactly one fill record covering the full requested quantity.
- **Price = "0"**: Paper mode does not simulate price discovery. Fill price is zero.
- **Fee = "0"**: Paper mode does not simulate fees.
- **Simulated = true**: All paper fills are flagged as simulated.
- **No-action orders**: Orders with side = none have no fills, empty `filled_quantity`, and remain in `submitted` status.

### Fill Accumulation (future)

When partial fills are introduced:
- Each fill event appends to the `fills` slice.
- `filled_quantity` is the sum of all `fills[].quantity`.
- Status transitions: `accepted → partially_filled` (on first partial fill), `partially_filled → filled` (when `filled_quantity == quantity`).
- This logic is not implemented in S77 — partial fills are modeled but not produced.

## Consistency Rules

| Rule | Description |
|------|-------------|
| CR-1 | If status = `filled`, `fills` must contain at least one record |
| CR-2 | If status = `filled`, `filled_quantity` must equal `quantity` |
| CR-3 | If status = `submitted`, `fills` must be empty |
| CR-4 | If `simulated = true`, the fill originated from paper execution |
| CR-5 | `filled_quantity` must equal the sum of individual fill quantities |

> **Note**: CR-1 through CR-5 are documented invariants. Enforcement in `Validate()` is deferred to a future stage to keep the validation surface minimal. The `PaperFillSimulator` produces correct fills by construction.

## PaperFillSimulator

Pure application-layer function. No I/O, no actor references, no NATS dependency.

**Input**: `ExecutionIntent` with `status = submitted`.

**Output**:
- For side = buy/sell: `ExecutionIntent` with `status = filled`, one fill record, `filled_quantity = quantity`.
- For side = none: `ExecutionIntent` unchanged (`status = submitted`, no fills).
- For non-submitted status: returns `false` (unexpected input).

## Query Surface Impact

The query response (`GET /execution/:type/latest`) now includes:
- `status`: reflects the lifecycle state (e.g., `filled` instead of always `submitted`).
- `filled_quantity`: cumulative fill quantity.
- `fills`: array of fill records.

No changes to query parameters or routes were needed — the response structure expanded naturally via the `ExecutionIntent` JSON serialization.

## Intentional Limitations (S77)

- **No price simulation**: Paper fills use price = "0". Price modeling requires venue integration.
- **No fee simulation**: Paper fills use fee = "0". Fee schedules require venue-specific configuration.
- **No partial fill production**: The `partially_filled` status exists in the domain model but is not produced by any evaluator or simulator in this stage.
- **No fill validation in Validate()**: Fill consistency (CR-1 through CR-5) is not enforced at the domain validation level. This is deferred to keep the validation gate simple while the model stabilizes.
