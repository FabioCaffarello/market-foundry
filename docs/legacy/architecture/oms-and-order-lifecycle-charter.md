# OMS and Order Lifecycle Charter

**Stage:** S309
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Status:** DELIVERED

---

## 1. Purpose

This charter defines the **minimum Order Management System (OMS) semantics** the Foundry requires to support venue readiness — without opening a full OMS platform. It establishes what the system needs, what it explicitly does not need, and why.

The Foundry is not a broker, not an EMS, and not a portfolio management system. It is a **signal-to-execution pipeline** that generates execution intents from analytical signals and submits them to a single venue under safety controls. The OMS scope must reflect this identity.

---

## 2. Strategic Context

### 2.1 Where We Are

| Capability | Status | Stage |
|-----------|--------|-------|
| Execution intent domain model | Proven | S264–S274 |
| 7-state lifecycle with transitions | Formalised | S308 |
| Fill record schema | Sufficient for real venue | S308 |
| VenuePort adapter (Binance testnet) | Implemented | S91 |
| Execution contracts (C-SUB through C-FAIL) | Documented | S308 |
| State monotonicity invariants | Documented | S308 |
| Safety gate (kill switch + staleness) | Operational | S273 |

### 2.2 What S309 Decides

S309 answers one question: **What is the minimum OMS surface the Foundry needs to safely submit market orders to a single venue and track their outcomes?**

The answer is deliberately narrow. The Foundry needs:
- A clear model of what an "order" means in this system;
- A lifecycle that tracks intent through execution to terminal state;
- Ownership rules for who creates, transitions, and persists each state;
- Guard rails that prevent scope inflation toward broker/EMS territory.

---

## 3. OMS Capability Assessment

### 3.1 Capabilities the Foundry NEEDS

| ID | Capability | Justification | Already Exists |
|----|-----------|---------------|----------------|
| OMS-C1 | **Order intent creation** | Derive binary produces ExecutionIntent from risk evaluation | Yes — `ExecutionIntent` struct |
| OMS-C2 | **Synchronous order submission** | Execute binary submits to venue and receives immediate response | Yes — `VenuePort.SubmitOrder()` |
| OMS-C3 | **Status tracking per intent** | Each intent carries its own status through lifecycle | Yes — `Status` field on `ExecutionIntent` |
| OMS-C4 | **Fill recording** | Venue fills are captured with price, quantity, fee, timestamp | Yes — `FillRecord` struct, `Fills []FillRecord` |
| OMS-C5 | **Terminal state recognition** | System knows when an order is done (filled, rejected, cancelled) | Yes — `IsTerminal()` method |
| OMS-C6 | **State transition validation** | Invalid transitions are rejected at domain level | Yes — `ValidTransition()` function |
| OMS-C7 | **Safety gate enforcement** | Kill switch and staleness guard block submission | Yes — `ControlGate` |
| OMS-C8 | **Correlation tracing** | Each intent carries correlation and causation IDs through pipeline | Yes — `CorrelationID`, `CausationID` fields |
| OMS-C9 | **Paper/venue discrimination** | Single field distinguishes simulated from real fills | Yes — `FillRecord.Simulated` |
| OMS-C10 | **Analytical persistence** | Terminal intents with fills are written to ClickHouse for query | Yes — writer + composite read model |

### 3.2 Capabilities the Foundry DOES NOT NEED

| ID | Capability | Reason for Exclusion |
|----|-----------|---------------------|
| OMS-X1 | **Order book / active orders ledger** | Fire-and-forget model — no in-flight order tracking needed |
| OMS-X2 | **Order amendment (modify price/qty)** | Market orders only; no limit orders to amend |
| OMS-X3 | **User-initiated cancellation** | Market orders fill instantly; no cancel window |
| OMS-X4 | **Smart order routing** | Single venue (Binance testnet); no routing decisions |
| OMS-X5 | **Portfolio-level position tracking** | Per-symbol, per-intent only; no net position aggregation |
| OMS-X6 | **Allocation / splitting** | Single venue, single account; no allocation needed |
| OMS-X7 | **Multi-venue arbitrage** | One adapter, one venue |
| OMS-X8 | **Async fill reconciliation** | Synchronous response model; fills arrive with submit response |
| OMS-X9 | **Order queuing / throttling** | Actor processes one intent at a time per symbol |
| OMS-X10 | **Historical order search / blotter** | ClickHouse composite read model serves this need without OMS |
| OMS-X11 | **Compliance / regulatory audit trail** | Not applicable at testnet stage |
| OMS-X12 | **Multi-account management** | Single credential set per adapter |

### 3.3 Assessment Verdict

**The Foundry already has all required OMS capabilities embedded in its domain model.** No new OMS module, service, or data store is needed. The `ExecutionIntent` struct with its status lifecycle, fill records, and correlation IDs constitutes the Foundry's minimal OMS.

What was missing was the **explicit charter** defining these boundaries. This document provides that charter.

---

## 4. Foundry Order Model

### 4.1 What Is an "Order" in the Foundry?

The Foundry does not use the term "order" as its primary abstraction. Instead, it uses **ExecutionIntent** — a domain object that represents the system's intent to execute a trade based on analytical evaluation.

| Concept | Traditional OMS | Foundry |
|---------|----------------|---------|
| Order creation | User/algo places order | Derive binary creates ExecutionIntent from risk evaluation |
| Order identity | Order ID | CorrelationID + CausationID (pipeline trace) |
| Order state | Order status | `ExecutionIntent.Status` |
| Fill | Execution report | `FillRecord` appended to `ExecutionIntent.Fills` |
| Order book | Active orders collection | Not applicable — fire-and-forget |
| Position | Aggregated fills per instrument | Not applicable — per-intent only |

### 4.2 Intent vs. Order Distinction

| Aspect | Order Intent (ExecutionIntent) | Venue Order |
|--------|-------------------------------|-------------|
| Created by | Derive binary | Venue adapter (on submit) |
| Identity | CorrelationID / CausationID | VenueOrderID (Binance orderId) |
| Lifetime | Pipeline-scoped (signal → fill → persist) | Venue-scoped (submit → fill) |
| Mutable fields | Status, FilledQuantity, Fills, Final | Immutable after submit |
| Ownership | Foundry domain | Venue API |
| Persisted in | NATS KV + ClickHouse | Venue's own systems |

The **VenueOrderID** is captured in the `VenueOrderReceipt` returned by the adapter but is not the primary identity. The Foundry tracks intents by correlation chain, not by venue order ID.

### 4.3 Ownership Matrix

| Entity | Creator | Status Transitions | Persistence | Query |
|--------|---------|-------------------|-------------|-------|
| ExecutionIntent | Derive binary | Derive: `→submitted`; Execute: all others | Store binary (KV + CH) | Gateway binary (HTTP) |
| FillRecord | Execute binary (adapter) | Immutable after creation | Store binary (CH) | Gateway binary (HTTP) |
| VenueOrderReceipt | Execute binary (adapter) | N/A (ephemeral) | Not persisted | N/A |
| ControlGate | Operator / configctl | Operator toggles active/halted | NATS KV | Execute binary |

---

## 5. Scope Freeze

### 5.1 What This Charter Authorises

1. Formal recognition that `ExecutionIntent` + `FillRecord` + `Status` lifecycle constitute the Foundry's minimal OMS.
2. Documentation of lifecycle semantics, state meanings, and transition ownership (companion document).
3. Explicit non-goals that prevent scope inflation.
4. Preparation of guard rails and failure envelope concepts for S310.

### 5.2 What This Charter Does NOT Authorise

1. Creation of any new OMS module, package, or service.
2. Introduction of an order book, position tracker, or order ledger.
3. Any domain model changes to `ExecutionIntent`, `FillRecord`, or `Status`.
4. New NATS subjects, KV buckets, or ClickHouse tables.
5. Multi-venue, multi-account, or portfolio-level capabilities.
6. Operational dashboards, alerting, or monitoring infrastructure.

---

## 6. Governing Principles

| # | Principle | Rationale |
|---|----------|-----------|
| GP-1 | **Intent, not order** | The Foundry's primary abstraction is execution intent, not a tradeable order. This prevents OMS creep. |
| GP-2 | **Fire-and-forget** | Submit → receive response → record outcome. No in-flight management. |
| GP-3 | **Terminal states are absorbing** | Once filled, rejected, or cancelled, no further transitions. No reopen, no retry at OMS level. |
| GP-4 | **Venue is opaque** | The adapter translates venue responses into domain types. The domain never speaks venue protocol. |
| GP-5 | **Single source of status** | `ExecutionIntent.Status` is the authoritative status. No shadow state, no derived status. |
| GP-6 | **Correlation over identity** | Pipeline traceability uses CorrelationID/CausationID, not venue order IDs. |
| GP-7 | **Safety before submission** | Kill switch and staleness checks happen before `VenuePort.SubmitOrder()`, never inside. |

---

## 7. Relationship to Venue Readiness Wave

| Stage | Role | Depends On Charter |
|-------|------|--------------------|
| S306 | Wave charter and scope freeze | — |
| S307 | Production gap map | — |
| S308 | Contracts and invariants | — |
| **S309** | **OMS charter and lifecycle semantics** | **This document** |
| S310 | Failure envelope and containment | GP-3 (terminal states), OMS-C5 |
| S311 | Multi-symbol venue isolation | OMS-C8 (correlation), OMS-C9 (discrimination) |
| S312 | Wave gate and closure | Full charter compliance |

---

*Delivered: 2026-03-21 — Stage S309, Phase 30*
