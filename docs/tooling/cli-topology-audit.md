# CLI Topology Audit

> Defines what `raccoon-cli topology-doctor` validates for the market-foundry service topology.

---

## Expected Topology

```
                    ┌──────────┐
                    │  gateway  │ ← HTTP :8080
                    └────┬─────┘
                         │ NATS req/reply
    ┌────────────────────┼────────────────────┐
    │                    │                    │
┌───▼───┐          ┌────▼────┐          ┌────▼────┐
│configctl│         │  store   │         │  (future)│
└───┬───┘          └────┬────┘          └─────────┘
    │ events            │ consumes
    │                   │ EVIDENCE_EVENTS
┌───▼───┐          ┌────▼────┐
│ ingest │          │ derive  │
└───┬───┘          └────┬────┘
    │ publishes         │ publishes
    │ OBSERVATION_EVENTS│ EVIDENCE_EVENTS
    └───────────────────┘
         NATS + JetStream
```

---

## Service Definitions

| Service | Binary | Config | Port | Dependencies |
|---------|--------|--------|------|-------------|
| nats | (infrastructure) | deploy/nats/ | 4222, 8222 | none |
| configctl | cmd/configctl | configctl.jsonc | — | nats |
| gateway | cmd/gateway | gateway.jsonc | 8080 | nats, configctl, store |
| ingest | cmd/ingest | ingest.jsonc | 8082 | nats, configctl |
| derive | cmd/derive | derive.jsonc | 8083 | nats |
| store | cmd/store | store.jsonc | 8081 | nats, derive |

---

## Checks Performed

### Phase 1: Config Validation

For each service (configctl, gateway, ingest, derive, store):
1. Config file exists in `deploy/configs/{service}.jsonc`
2. Config contains valid JSON (with comments stripped)
3. Config has `nats` section with `enabled`, `url`, `request_timeout`
4. NATS is enabled (`enabled: true`)

### Phase 2: Compose Validation

1. All 6 services defined in docker-compose.yaml (nats + 5 binaries)
2. Service dependencies match expected graph
3. Health checks defined for all services
4. Image names follow `market-foundry/{service}:dev` pattern
5. Config volume mounts reference correct `.jsonc` files

### Phase 3: Stream and Subject Validation

Scans Go source for JetStream stream/subject definitions:

| Stream | Expected Subjects | Producer | Consumers |
|--------|------------------|----------|-----------|
| CONFIGCTL_EVENTS | `configctl.events.config.>` | configctl | ingest, derive |
| OBSERVATION_EVENTS | `observation.events.market.>` | ingest | derive |
| EVIDENCE_EVENTS | `evidence.events.candle.>` | derive | store |

| Durable Consumer | Stream | Service |
|-----------------|--------|---------|
| `derive-observation` | OBSERVATION_EVENTS | derive |
| `store-evidence` | EVIDENCE_EVENTS | store |

| Query Subject | Server | Client |
|--------------|--------|--------|
| `evidence.query.candle.latest` | store | gateway |
| `configctl.control.config.*` | configctl | gateway |

### Phase 4: Cross-Validation

1. NATS URLs consistent across all config files and compose
2. No references to removed infrastructure (Kafka brokers, old service names)
3. Config file names match compose service names

---

## Common Topology Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| Missing config file | New service added without config | Create `deploy/configs/{service}.jsonc` |
| Compose service mismatch | Binary renamed but compose not updated | Update compose service name |
| NATS URL inconsistency | Different URLs in different configs | Standardize to `nats://nats:4222` |
| Missing health check | New compose service without healthcheck | Add healthcheck block |
| Stream not found in source | Registry not yet implemented | Add stream definition in NATS adapter |

---

## Running Topology Audit

```bash
# Quick check
raccoon-cli topology-doctor

# Verbose with all findings
raccoon-cli -v topology-doctor

# JSON for CI
raccoon-cli --json topology-doctor
```
