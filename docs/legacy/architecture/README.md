# Architecture Documentation

## Purpose

This directory is the deep technical reference for `market-foundry`.

Use it after you know whether your question is product-facing or
development-facing.

## Start Here

| Need | Primary document |
|---|---|
| Product/runtime owner map | [`../product/owners.md`](../product/owners.md) |
| System identity and principles | [`system-vision.md`](system-vision.md), [`system-principles.md`](system-principles.md) |
| Runtime topology and binary boundaries | [`runtime-target.md`](runtime-target.md), [`actor-ownership.md`](actor-ownership.md) |
| Evolution governance | [`market-foundry-evolution-playbook.md`](market-foundry-evolution-playbook.md) |
| Information-system governance | [`information-system-governance-and-classification.md`](information-system-governance-and-classification.md) |
| Stage completion and debt prevention | [`stage-definition-of-done.md`](stage-definition-of-done.md), [`anti-debt-checklist.md`](anti-debt-checklist.md) |
| Domain design corpus | `signal`, `decision`, `strategy`, `risk`, and `execution` design docs |
| Analytical and ClickHouse reference | [`analytical-storage-strategy.md`](analytical-storage-strategy.md), [`analytical-observability-and-runbook.md`](analytical-observability-and-runbook.md), [`cmd-migrate-and-migration-catalog.md`](cmd-migrate-and-migration-catalog.md) |
| Live exchange listening proof | [`compose-live-exchange-listening-proof.md`](compose-live-exchange-listening-proof.md) |
| Live ingress preconditions | [`live-ingress-runtime-wiring-preconditions-and-limitations.md`](live-ingress-runtime-wiring-preconditions-and-limitations.md) |
| Dry-run execution path | [`dry-run-execution-path-by-config.md`](dry-run-execution-path-by-config.md) |
| Dry-run fail-closed semantics | [`dry-run-submitter-fail-closed-semantics-and-auditability.md`](dry-run-submitter-fail-closed-semantics-and-auditability.md) |
| E2E live-listen + dry-run proof | [`end-to-end-live-listen-plus-dry-run-proof.md`](end-to-end-live-listen-plus-dry-run-proof.md) |
| Live-listen + dry-run evidence | [`live-listen-dry-run-canonical-pipeline-evidence-and-limitations.md`](live-listen-dry-run-canonical-pipeline-evidence-and-limitations.md) |
| Canonical order model & lifecycle | [`canonical-order-model-and-lifecycle-state-machine.md`](canonical-order-model-and-lifecycle-state-machine.md) |
| Lifecycle invariant coverage | [`order-lifecycle-invariant-coverage-matrix-and-price-realism-findings.md`](order-lifecycle-invariant-coverage-matrix-and-price-realism-findings.md) |
| Write-path per execution mode | [`write-path-integration-tests-by-execution-mode.md`](write-path-integration-tests-by-execution-mode.md) |
| Rejection event observability | [`rejection-event-path-and-write-path-observability.md`](rejection-event-path-and-write-path-observability.md) |
| Lifecycle persistence & PriceSource | [`lifecycle-persistence-read-path-alignment-and-pricesource-wiring.md`](lifecycle-persistence-read-path-alignment-and-pricesource-wiring.md) |
| OMS foundation evidence gate | [`oms-foundation-evidence-gate.md`](oms-foundation-evidence-gate.md) |
| Testnet venue execution proof charter | [`testnet-venue-execution-proof-wave-charter-and-scope-freeze.md`](testnet-venue-execution-proof-wave-charter-and-scope-freeze.md) |
| Testnet venue execution capabilities & non-goals | [`testnet-venue-execution-capabilities-questions-and-non-goals.md`](testnet-venue-execution-capabilities-questions-and-non-goals.md) |
| Binance segmentation wave charter | [`binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md`](binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md) |
| Binance segmentation capabilities & non-goals | [`binance-segmentation-capabilities-questions-and-non-goals.md`](binance-segmentation-capabilities-questions-and-non-goals.md) |
| Venue model refactor (4 dimensions) | [`venue-model-refactor-exchange-market-segment-environment-and-execution-mode.md`](venue-model-refactor-exchange-market-segment-environment-and-execution-mode.md) |
| Venue segmentation valid/invalid configs | [`venue-segmentation-semantics-valid-invalid-configurations-and-fail-closed-rules.md`](venue-segmentation-semantics-valid-invalid-configurations-and-fail-closed-rules.md) |
| Unified segment runtime wave charter | [`unified-segment-runtime-wave-charter-and-scope-freeze.md`](unified-segment-runtime-wave-charter-and-scope-freeze.md) |
| Unified segment runtime capabilities & non-goals | [`unified-segment-runtime-capabilities-questions-and-non-goals.md`](unified-segment-runtime-capabilities-questions-and-non-goals.md) |
| Endurance soak and persistence hardening | [`endurance-soak-and-execution-persistence-hardening.md`](endurance-soak-and-execution-persistence-hardening.md) |
| Sustained execution consistency & limitations | [`sustained-execution-state-consistency-writer-stability-and-limitations.md`](sustained-execution-state-consistency-writer-stability-and-limitations.md) |
| Unified compose E2E -- Futures proof | [`unified-compose-e2e-proof-with-futures-live-execution-path.md`](unified-compose-e2e-proof-with-futures-live-execution-path.md) |
| Futures E2E evidence, controls & limitations | [`futures-segment-e2e-compose-evidence-controls-and-limitations.md`](futures-segment-e2e-compose-evidence-controls-and-limitations.md) |
| Runtime simplification wave charter | [`runtime-simplification-and-futures-proof-prep-wave-charter-and-scope-freeze.md`](runtime-simplification-and-futures-proof-prep-wave-charter-and-scope-freeze.md) |
| Runtime simplification capabilities & non-goals | [`runtime-simplification-capabilities-questions-and-non-goals.md`](runtime-simplification-capabilities-questions-and-non-goals.md) |
| Runtime simplification evidence gate | [`runtime-simplification-evidence-gate-and-futures-proof-authorization.md`](runtime-simplification-evidence-gate-and-futures-proof-authorization.md) |
| Runtime simplification evidence matrix & gaps | [`runtime-simplification-evidence-matrix-residual-gaps-and-next-ceremony.md`](runtime-simplification-evidence-matrix-residual-gaps-and-next-ceremony.md) |

## Reading Rule

- start in [`../product/README.md`](../product/README.md) when the question is
  "what is this system?" or "which product doc owns this?";
- start in [`../development/README.md`](../development/README.md) when the
  question is "how do I work here?";
- use this directory when you need the detailed technical answer behind those
  surfaces;
- use [`../stages/INDEX.md`](../stages/INDEX.md) and [`../archive/README.md`](../archive/README.md)
  for history rather than treating transient wave docs as primary navigation.
