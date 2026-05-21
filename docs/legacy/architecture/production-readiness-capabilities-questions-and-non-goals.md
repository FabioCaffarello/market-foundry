# Production Readiness — Capabilities, Questions, and Non-Goals

> Companion to the Production Readiness Assessment Wave Charter (S347).
> Defines the capabilities under assessment, governing questions, evaluation
> criteria, and explicit non-goals.

## 1. What "Production Readiness" Means in This Context

Production readiness for the Foundry does **not** mean "ready to trade real money
on mainnet." It means:

> The venue activation capability can operate against a real testnet endpoint
> in a sustained, monitored, and repeatable manner, with operational procedures
> that do not depend on developer intervention.

This is a testnet-grade readiness assessment. It evaluates whether the system
can be left running against a real (but risk-free) venue for hours or days,
with confidence that:

- connectivity is real and authenticated;
- errors are classified and surfaced;
- stability is maintained without drift;
- operators can observe, halt, and recover without code changes;
- deployment is repeatable.

## 2. Capabilities Under Assessment

### C-1: Real Venue Connectivity

| Aspect | Current State | Target State |
|--------|--------------|--------------|
| Endpoint | httptest.Server mock | testnet.binancefuture.com |
| Authentication | HMAC code paths exercised | Real API key + HMAC against live endpoint |
| Network | Loopback only | Real HTTPS with TLS |
| Error modes | Simulated HTTP 400/500 | Real rate limits, auth errors, timeouts |
| Fill confirmation | Mock response parsing | Real venue fill parsing |

### C-2: Sustained Operation

| Aspect | Current State | Target State |
|--------|--------------|--------------|
| Duration | 2-minute windows | 2+ hour continuous runs |
| Counter drift | Zero over minutes | Zero over hours |
| Resource stability | Unmeasured | Measured and baselined |
| Error accumulation | Zero over minutes | Zero over hours |
| Goroutine/FD leaks | Untested at scale | Proven absent |

### C-3: Operational Observability

| Aspect | Current State | Target State |
|--------|--------------|--------------|
| Health signal | `healthz.Tracker` counters | Counters + threshold alerts |
| Gate visibility | GET /activation/surface | + gate-change notifications |
| Fill visibility | Structured logs | Structured logs + counter dashboards (defined, not built) |
| Error visibility | Structured logs | Structured logs + error rate alerts (defined, not built) |
| Latency visibility | Unmeasured | Submission latency p50/p95/p99 defined |

### C-4: Deployment Repeatability

| Aspect | Current State | Target State |
|--------|--------------|--------------|
| Deployment | Manual `go run` / binary | Scripted with pre-checks |
| Smoke | scripts/smoke-activation.sh (httptest) | Extended for testnet |
| Verification | Manual GET /activation/surface | Automated post-deploy check |
| Rollback | Manual gate-halt + restart | Scripted gate-halt + restart |

## 3. Governing Questions

Each question maps to one or more PRA blocks and has a clear evidence type.

| # | Governing Question | PRA Block | Evidence Type |
|---|-------------------|-----------|---------------|
| PQ-1 | Can the venue adapter authenticate with real Binance testnet credentials? | PRA-1 | Integration test with real endpoint |
| PQ-2 | Does a real testnet order round-trip produce a parseable fill? | PRA-1 | Fill event with venue-sourced fields |
| PQ-3 | Are testnet-specific errors (rate limits, auth failures) classified correctly? | PRA-1 | Error classification tests |
| PQ-4 | Does credential loading follow a secure, documented procedure? | PRA-1 | Procedure document + test |
| PQ-5 | Does the system maintain counter consistency over 2+ hours? | PRA-2 | Soak test output with checkpoint log |
| PQ-6 | Is resource consumption (memory, goroutines, FDs) stable over hours? | PRA-2 | Resource baseline measurements |
| PQ-7 | Does the gate remain responsive after hours of sustained operation? | PRA-2 | Gate-halt latency measurement at end of soak |
| PQ-8 | Are error rates stable (not accumulating) over hours? | PRA-2 | Error counter time series |
| PQ-9 | Is the monitoring surface defined with specific metrics and thresholds? | PRA-3 | Monitoring surface document |
| PQ-10 | Are alert rules actionable (clear trigger, clear response)? | PRA-3 | Alert rule catalog |
| PQ-11 | Can gate changes be detected without polling? | PRA-3 | Notification mechanism (log or push) |
| PQ-12 | Can the system be deployed with a single command? | PRA-4 | Deployment script/Makefile target |
| PQ-13 | Does the smoke script work against real testnet? | PRA-4 | Smoke run output against testnet |
| PQ-14 | Can rollback be performed without developer intervention? | PRA-4 | Rollback script execution |
| PQ-15 | Is the full deploy → smoke → verify cycle automated? | PRA-4 | End-to-end automation output |

## 4. Evaluation Criteria

Each PRA block is evaluated on a three-level scale:

| Level | Meaning |
|-------|---------|
| FULL | All governing questions for the block answered with HIGH confidence |
| SUBSTANTIAL | Most questions answered; remaining gaps are documented and non-blocking |
| NOT MET | Core questions unanswered or evidence insufficient |

The wave verdict aggregates block verdicts:

| Wave Verdict | Condition |
|-------------|-----------|
| READY | All blocks FULL or SUBSTANTIAL, PRA-1 and PRA-2 must be FULL |
| PARTIAL | PRA-1 FULL, at least one other block SUBSTANTIAL or above |
| NOT READY | PRA-1 NOT MET, or more than one block NOT MET |

## 5. Non-Goals

The following are explicitly out of scope for this wave. Each non-goal includes
the rationale for exclusion.

### NG-1: Mainnet Activation

**What**: Connecting to Binance mainnet or any production trading endpoint.
**Why excluded**: The assessment wave evaluates testnet-grade readiness. Mainnet
requires a separate risk assessment, capital allocation, and compliance review
that are outside the Foundry's current scope.

### NG-2: Multi-Venue Expansion

**What**: Adding adapters for exchanges beyond BinanceFuturesTestnet.
**Why excluded**: Single-venue must be production-proven before multi-venue.
The gate model (global only, DG-8) and adapter lifecycle are designed for
single-venue. Multi-venue requires per-venue gate isolation.

### NG-3: Order Management System (OMS)

**What**: Position tracking, portfolio-level risk, P&L calculation, order book management.
**Why excluded**: The Foundry's execution scope is submission-only. OMS is a
separate domain with its own lifecycle. Mixing OMS into production readiness
assessment would violate scope freeze.

### NG-4: Portfolio Risk Management

**What**: Exposure limits, margin management, drawdown controls, portfolio-level circuit breakers.
**Why excluded**: Risk management is a domain concern that sits above execution.
The production readiness assessment evaluates whether the execution pipeline
can operate safely, not whether trading decisions are sound.

### NG-5: Broad Dashboards and Visualization

**What**: Grafana dashboards, real-time charts, historical analytics views.
**Why excluded**: The assessment defines the monitoring surface (what to observe),
not the visualization layer (how to display it). Dashboard construction is a
separate deliverable that follows the monitoring surface definition.

### NG-6: New Functional Breadth

**What**: New capabilities, new domain types, new actor families, new event streams.
**Why excluded**: The wave assesses existing capabilities under realistic conditions.
Adding new capabilities would expand the assessment surface and violate the
charter's assessment-only mandate.

### NG-7: Strategy or Signal Integration

**What**: Connecting signal generation, strategy evaluation, or alpha models to the execution pipeline.
**Why excluded**: Strategy integration depends on execution being production-ready.
This wave establishes that readiness; strategy integration follows.

### NG-8: Infrastructure Platform Changes

**What**: Kubernetes, Docker orchestration, cloud deployment, service mesh.
**Why excluded**: The assessment evaluates the application's operational behavior,
not the deployment platform. Infrastructure choices follow the assessment verdict.

### NG-9: Credential Rotation Under Load

**What**: Hot-swapping API credentials without restart during sustained operation.
**Why excluded**: Credentials are process-immutable by design (DG-7). Credential
rotation is a production operations concern that follows the readiness assessment.
The assessment verifies that credentials load correctly at startup.

### NG-10: Chaos Engineering or Fault Injection

**What**: Systematic fault injection (kill processes, corrupt messages, partition networks).
**Why excluded**: Chaos engineering requires a stable operational baseline that
this wave is establishing. The assessment proves baseline stability; chaos
testing evaluates resilience beyond that baseline.

## 6. Relationship to Deferred Gaps

The table below maps each deferred gap from S346 to this wave's scope:

| Gap ID | Description | In Scope? | Addressed By |
|--------|-------------|-----------|-------------|
| DG-1 | Live testnet not exercised | YES | PRA-1 |
| DG-2 | Hours-scale soak testing | YES | PRA-2 |
| DG-3 | No automated circuit breaker | PARTIAL | PRA-3 (defines; does not implement) |
| DG-4 | No activation history endpoint | NO | Future observability wave |
| DG-5 | Full rollback requires restart | PARTIAL | PRA-4 (automates; does not redesign) |
| DG-6 | No push notifications for gate changes | PARTIAL | PRA-3 (defines mechanism) |
| DG-7 | Credentials process-immutable | NO | NG-9 |
| DG-8 | Global gate only | NO | NG-2 |
| DG-9 | Partial fills not exercised | NO | Venue protocol evolution |
| DG-10 | Post200Reconciler failure path | PARTIAL | PRA-1 (may trigger naturally with real testnet) |
| DG-11 | RetrySubmitter not triggered | PARTIAL | PRA-2 (may trigger naturally under endurance) |
| DG-12 | Binary restart during observation | NO | NG-10 |
