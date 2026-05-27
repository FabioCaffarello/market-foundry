# Resumption

> This document is the first thing to read when returning to
> market-foundry after a pause. It captures the current state, known
> gaps, and the next concrete step.
>
> It is **honest, not aspirational.** If a capability is missing or
> partial, it says so. If a feature is broken, it says where.

Last meaningful state change: **Phase 4 CLOSED (2026-05-23)** —
P0 backlog FULLY CLOSED (5/5 items: CI restoration via the P4.1
wave, `rate_limiter` + `Close` lifecycle, `context.Background()`
bounding, ControlGate fail-open posture formalized as ADR-0012,
and the Dependabot triage wave). 12 ADRs total; 20 design-meta
candidates queued (M1–M20, M19 closed during P4.5.c verification);
~9 errata observations accumulated.

`make verify` GREEN locally; CI 7/7 GREEN at `main` HEAD, sustained
since P4.1.1's SHA-pinning migration. Some intermediate Dependabot
merges show the documented `TestControlledActivation_FullLifecycle`
/ `TestRealVenueActivation_FullLifecycle` Integration Tests timing
flake; these are non-required and non-blocking per branch protection
(see Phase 4.5 narrative for full posture).

Phase 5 OPENED (2026-05-23) — environment work, distinct from
Phase 4's code/CI delivery. P5.0 audit (read-only, ~7 min wall-clock)
categorized 12 findings across `.claude/`, prompt templates,
operational tooling, and process debt; produced an 8-slot P5.x
candidate roadmap pending owner direction. See "Phase 4 outlook"
below for retrospective detail; Phase 5 narrative pending P5.1+.

---

## Fase Harvest

**Fase Harvest aberta (2026-05-24)** sob protocolo P1–P9 — ver
[`../CLAUDE.md`](../CLAUDE.md) → "Fase Harvest" para a versão
canônica dos princípios. P9 ("Toda alteração ao foundry passa por
PR; maintainer humano faz o merge") foi adicionado durante H-1 como
erratum do prompt H-0 que entregou apenas P1–P8. Programa de Fundação tracked em
[`programs/PROGRAM-0001-foundation.md`](programs/PROGRAM-0001-foundation.md)
(Status: `Active`); decisão de adoção em
[`decisions/0016-harvest-from-market-raccoon.md`](decisions/0016-harvest-from-market-raccoon.md)
(Status: `Accepted`).

Wave protocol — uma onda por vez (P4); próxima onda abre após
**merge** real em `main` (P9), não apenas após completion local.

| Onda | Estado | Escopo |
|------|--------|--------|
| **H-0** | Fechada (PR #19 mergeada em `main` em `c762b8f`, 2026-05-24) | Setup do Harvest: ADR-0016, PROGRAM-0001, CLAUDE.md → "Fase Harvest" (P1–P8), `.claude/settings.json` (`RACCOON_REFERENCE_PATH`). |
| **H-1** | Fechada (PR #20 mergeada em `main` em `65f4c3f`, 2026-05-24) | Práticas operacionais: [`TRUTH-MAP`](TRUTH-MAP.md), [`AUTHORITY`](AUTHORITY.md), [`runtime-invariants`](operations/runtime-invariants.md), [`slo.md`](operations/slo.md). Erratum integrado: P9 adicionado a CLAUDE.md → "Fase Harvest" + propagado para ADR-0016, PROGRAM-0001, e este documento. |
| **H-2** | Fechada (PR #21 mergeada em `main` em `a93f3d8`, 2026-05-24) | Sete ADRs de fundação (0017–0023) em status `Proposed`. Sem código de produto novo. Cada ADR carrega seção "Promoção para Accepted" nomeando a onda implementadora. |
| **H-3.a** | Fechada (PR #22 mergeada em `main` em `387811b`, 2026-05-25) | Proto skeleton + buf tooling. Abre [PROGRAM-0002](programs/PROGRAM-0002-wire.md) (Fase Wire). Entrega `proto/` com `buf.yaml`/`buf.gen.yaml`/`registry.json`/`envelope/v1/envelope.proto`/`marketdata/v1/trade.proto`; `make proto-{lint,gen,breaking}`; bootstrap-check valida buf; `.tool-versions` adiciona buf; **erratum a ADRs 0017/0018** separando decisão arquitetural de execução de rollout. Sem código Go gerado tracked, sem analyzer raccoon-cli. ADRs 0017/0018 continuam `Proposed`. |
| **H-3.b** | Fechada (PR #23 mergeada em `main` em `32d1792`, 2026-05-25) | Code generation + converters + analyzer. `internal/shared/contracts/envelope/v1/envelope.pb.go` + `marketdata/v1/trade.pb.go` tracked (gitignore G removed); `CanonicalEvent` foundry-native domain projection + converter; raccoon-cli `check proto` analyzer integrado em `make verify` (via quality-gate); `make proto-lint` adicionado a verify; bootstrap valida `protoc-gen-go v1.36.8` (pinned matching runtime). Promove ADR-0017 e ADR-0018 a `Accepted` — primeira promoção de ADR Proposed→Accepted da Fase Harvest. |
| **H-4** | Fechada (PR #24 mergeada em `main` em `218a010`, 2026-05-25) | Replay + Sequencer + determinism analyzer + dual ADR promotion + PRD closure. 14 commits: clock/random ports, replay recorder+player, sequencer, KV bucket+Store, gap counter, Clock plumbing through cmd/* + actor configs, 5 domain migrations (DefaultVerificationScope, DefaultControlGate, NewActivationSurface, Session.Close, Session.Halt), check determinism analyzer + gate integration, golden test + N=50 byte-stability, ADR-0019 + ADR-0020 → Accepted, PROGRAM-0002 → Closed. **Fase Wire fechada.** |
| **H-5** | Fechada (PR #25 mergeada em `main` em `6df8e66`, 2026-05-25) | PROGRAM-0003 (Observability) opening + delivery. 11 commits: PRD-0003, ADR-0024 metrics-policy, ADR-0025 alerting-strategy, refactor `consumer_seq_gap_total` (drop instrument label per ADR-0024 MP-2), prometheus+grafana opt-in compose profile, prometheus scrape + recording rules (44 rules, 4 SLO groups + runtime-aggregates), burn-rate alerts (13 rules — 8 SLO at ticket severity per Observing taxonomy + 5 runtime-safety), 5 Grafana dashboards provisioning (ingest/derive/store/gateway/determinism-health), raccoon-cli `check metrics` analyzer with declarative `tools/raccoon-cli/policies/binaries.toml` allowlist, SLOs F1–F4 flipped `Proposed`→`Observing`, `docs/operations/observability.md` operator guide, ADR-0024 + ADR-0025 → Accepted, PROGRAM-0003 opened Active. **Observability stack ativo.** |
| **H-6** | Sub-dividida em H-6.a/b/b'/b''/c/d/e/f por cascade discovery (pré-flight H-6.a: 342 `.Symbol` refs em 106 production files em 31 packages; pré-flight H-6.b: 15 domain types em 174 test files → split por dependency order em b/b'/b''). Ver [PROGRAM-0004](programs/PROGRAM-0004-multi-venue.md). Sub-onda sequencing policy estrita: próxima abre APENAS após merge da anterior em `main`. | PROGRAM-0004 (Multi-venue) implementation. ADR-0021 promotion é atômica em H-6.f. |
| **H-6.a** | Fechada (PR #26 mergeada em `main` em `ac7fb8f`, 2026-05-26) | PROGRAM-0004 opening + canonical instrument domain root. 8 commits incl. ADR-0021 erratum (criterion #4 split em #4a/#4b), PRD-0004, `internal/domain/instrument/` package, atomic `ObservationTrade.Symbol` → `Instrument` + `VenueSymbol()`, ambos Binance adapters com regex `_\d{6}$` para delivery futures, raccoon-cli `check instruments` analyzer (4 checks). ADR-0021 permanece `Proposed`. |
| **H-6.b** | Fechada (PR #27 mergeada em `main` em `d7fae4c`, 2026-05-26) | Layer 1+2 dependency order: 7 domain types migrados Symbol → Instrument + VenueSymbol() per ADR-0021. 7 commits: PRD-0004 sub-onda b/b'/b'' refinement, EvidenceCandle atomic, EvidenceTradeBurst+Volume consolidado, Signal+Decision pair (PartitionKey via VenueSymbol), Strategy+Risk pair, check-instruments analyzer estendido via `policies/domain_types.toml` declarando migration_state per type (6 checks total, +2 do domain-type check), docs closure. 6 application samplers + 3 decision evaluators + 3 strategy resolvers + 2 risk evaluators gain `instrumentFromBinding` transitory helper (sunset H-6.c). ClickHouse readers reuse `reconstructInstrumentFromLegacy` da H-6.a. ADR-0021 permanece `Proposed`. |
| **H-6.b'** | Fechada (PR #28 mergeada em `main` em `6b62d89`, 2026-05-26) | Layer 3+3' dependency order: 3 domain types da execution chain migrados Symbol → Instrument + VenueSymbol() per ADR-0021. 5 commits + fix(execute) pull-forward 37f8ddd (descoberto via CI Integration Tests em PR #28: silent zero Instrument por reconstrução source-string em `instrumentFromBinding`; fix via `NewPaperOrderEvaluatorForInstrument` passthrough). check-instruments analyzer 6 checks PASS. **Triage drop closure note** (zero migration sites nesta sub-wave): DecisionTriageItem buffered pelo ReviewTransform DTO; ExecutionTriageItem não existe; RoundTripTriageItem deferido para H-6.b''. ADR-0021 permanece `Proposed`. |
| **H-6.b''** | Fechada (PR #29 mergeada em `main` em `54a2706`, 2026-05-26) | Layer 4: Pairing chain migrada — 2 domain types Symbol → Instrument + VenueSymbol() (pairing.Leg + pairing.RoundTrip) + 1 rename (pairing.CrossSessionWindow.Symbol → VenueSymbol string, declarado `string_filter` per Decisão #2) + 1 triage population site (`get_roundtrip_triage.go:74` adopts `review.VenueSymbol()` por compile pressure pull-forward). 8 commits (plano declarava 9 — consolidação por compile pressure documentada em commit 3 e commit 8) + 1 follow-up commit (G6 flake registry + pre-push lesson). check-instruments analyzer estendido com 3º state `string_filter` (commit 1) e 15 unit tests (was 14). ADR-0021 permanece `Proposed`. |
| **H-6.c.1** | Fechada (PR #30 mergeada em `main` em `8125e6c`, 2026-05-27) | Application-layer pass-through migration: derive scope. 13 commits eliminating source-string Instrument reconstruction in 4 application packages (signal/decision/strategy/risk). Commit 1 installs declarative `policies/anti_patterns.toml` + analyzer scan extension. Commits 2-5 add `NewXxxForInstrument` pass-through constructors (14 total). Commit 6 wires derive actors to compute Instrument once at the `BindingTarget.Instrument()` boundary (new helper in `internal/application/ingest/binding.go` with error-returning signature — eliminates the H-6.b' commit 37f8ddd silent-zero regression-shape at its source). Commits 7a-7d delete the legacy `NewXxx` wrappers + per-package `instrumentFromBinding` helpers + dead `symbol` struct field, migrating ~250 test sites. Commit 8 adds derive-scope canary integration tests for the 6 synthetic sources from commit 6. Commit 9 records migration progress in `anti_patterns.toml`. Commit 10 closes docs. **CI fixup (commits 11-13)**: commit 11 populates anti_patterns exception list with all 11 deferred call sites (the `--profile ci` gate promotes warning→error; the policy installed with severity=warning was incompatible until exceptions covered all remaining callers); commit 12 fixes a pre-existing topology analyzer fragility (whole-file fallback in `find_stream_name_near` picked unrelated SCREAMING_SNAKE_CASE constants on Linux file iteration); commit 13 records the `--profile ci` pre-push validation requirement permanently in CONTRIBUTING.md. ADR-0021 permanece `Proposed`. |
| **H-6.c.2** | **Atual** (esta entrega — branch `feat/h-6-c-2-execute-clickhouse` aberta após merge de H-6.c.1 em PR #30) | Application-layer pass-through migration: execute scope + ClickHouse composite_reader treatment. 8 commits eliminating the last source-string reconstruction sites in the execution package + uniformizing the ClickHouse `reconstructInstrumentFromLegacy` error-handling pattern. Commit 1 migrates ~28 paper_order_evaluator test sites to ForInstrument constructor. Commit 2 converts the 5 silent error-discard sites in `composite_reader.go` to warn-and-emit-zero (matches the 8 existing sister sites' pattern; all 13 ClickHouse readers now uniform). Commit 3 declares `ReviewTransform` as `string_filter` per Decisão Q2 of H-6.c.1 + adds inline godoc to ReviewTransform.Symbol + DecisionTriageItem.Symbol fields. Commit 4 migrates the 2 testnet adapters (`binance_spot/futures_testnet_adapter.go`) to use `BindingTarget.Instrument()` boundary helper with warn-and-emit-zero fallback per Decisão #2 — option (b) confirmed after option (a) port-signature refactor cascade analysis revealed 12 files exceeding the sub-onda threshold (a) recorded as H-6.f candidate. Commit 5 deletes the legacy `NewPaperOrderEvaluator` ctor + `instrumentFromBinding` helper file + dead `symbol` field + migrates 8 cross-scope stragglers (1 derive test + 2 risk_scaling test + 6 integration-tagged: 1 writerpipeline + 4 natsexecution + 1 live_consumer_flow). Commit 6 adds the explicit 37f8ddd canary in execute scope (`execute_venue_adapter_canary_test.go` — 2 tests / 2 passes locking the contract that "execute.venue-adapter" synthetic source preserves canonical Instrument through PaperOrderEvaluator + strategy_consumer_actor). Commit 7 shrinks `anti_patterns.toml` exception list 11→8 + uniformizes prose. Commit 8 (este commit) closes docs. ADR-0021 permanece `Proposed`; **execução agora 5 dos 6 helpers eliminados (signal/decision/strategy/risk/execution); apenas executionclient remanesce para H-6.f**. |

**Nota sobre divisão H-3**: H-3 foi dividida em sub-ondas
**H-3.a** (proto skeleton + tooling) e **H-3.b** (code generation +
converters + analyzer) por escopo técnico — instalar tooling
externo na mesma onda em que se gera código Go + se escreve
analyzer Rust sobrecarrega revisão. Cada sub-onda é PR
independente; ambas fechadas, ADR-0017 e ADR-0018 promovem em
H-3.b. Divisão registrada em [PROGRAM-0002](programs/PROGRAM-0002-wire.md).

**Erratum H-3.a**: como **commit 0** do PR H-3.a, ambas ADR-0017
e ADR-0018 receberam erratum reescrevendo suas seções "Promoção
para Accepted" para separar **decisão arquitetural** (contrato +
tipos + analyzer, completável em H-3.b) de **execução de decisão**
(migração runtime dos 11 streams, fase futura). Sem o erratum,
H-3.b seria literal-incompatível com os critérios originais. O
erratum também removeu `make proto-gate` dos critérios de
aceitação de ADR-0018 — composição de targets é tooling, não
arquitetura.

Entregas H-3.b (esta sessão):

- `internal/shared/contracts/doc.go` — scaffold do package boundary
  per ADR-0018 (commit 1).
- `internal/shared/contracts/envelope/v1/envelope.pb.go` +
  `internal/shared/contracts/marketdata/v1/trade.pb.go` — código Go
  gerado de `.proto` via `make proto-gen` com `protoc-gen-go v1.36.8`
  (matching runtime). Tracked no repo; `.gitignore` section G removida
  (commit 2).
- `internal/shared/contracts/envelope/v1/envelope_test.go` — 3 testes
  (round-trip, ts_exchange absent, byte-stability N=50 per INV-D4)
  (commit 3).
- `internal/shared/contracts/marketdata/v1/trade_test.go` — 2 testes
  (round-trip, byte-stability) (commit 4).
- `internal/shared/contracts/envelope/v1/converter.go` — `CanonicalEvent`
  foundry-native domain projection do envelope canônico; `ToProto` +
  `FromProto` com validation explícita dos 6 campos obrigatórios
  (commit 5).
- `internal/shared/contracts/envelope/v1/converter_test.go` — 4 testes
  top-level + 13 sub-tests cobrindo round-trip, absence semantics,
  validation bidirecional (commit 5).
- `tools/raccoon-cli/src/analyzers/check_proto.rs` — novo analyzer
  Rust (595 LoC). Level B + Level C smoke (sync registry/proto/Go +
  PROTO-G3 domain boundary). 9 unit tests (commit 6).
- `tools/raccoon-cli/src/cli/mod.rs` + `application/mod.rs` +
  `gate/mod.rs` — wire do analyzer no CLI dispatch e no quality-gate
  pipeline. Subcommand `raccoon-cli check proto` disponível (commit 7).
- `Makefile` — `make verify` agora invoca `proto-lint`; novo target
  `make proto-check`; `make proto-gen` prepended PATH com
  `$(go env GOPATH)/bin` para encontrar `protoc-gen-go` (commit 7).
- `scripts/bootstrap-check.sh` — valida `protoc-gen-go` presence +
  versão exata v1.36.8 (pin matching runtime). Mensagem clara de
  install em caso de mismatch (commit 7).
- `docs/DEVELOPMENT.md` — entry para `protoc-gen-go` em External
  tooling table; nova subsection com install command + pin rationale
  (commit 7).
- `internal/shared/go.mod` — `google.golang.org/protobuf v1.36.8`
  promovido de indirect para direct dep (`go mod tidy` após adicionar
  primeiro consumer em `envelope_test.go`).
- `docs/decisions/0017-event-envelope-and-versioning.md` — Status
  `Proposed → Accepted`; Changelog entry "Promoted to Accepted"
  (commit 8).
- `docs/decisions/0018-protobuf-contract-layer.md` — Status
  `Proposed → Accepted`; Changelog entry (commit 8).
- `docs/TRUTH-MAP.md` — rows de ADR-0017/0018 atualizadas para
  `Implemented` com anchors reais (zero TODOs); seção
  "Planned capabilities — Foundation ADRs (Proposed)" renomeada para
  "Foundation ADRs — delivery state (mixed)" refletindo divisão entre
  Accepted (0017, 0018) e Proposed (0019, 0020, 0021, 0022, 0023);
  Summary count revisado (0001–0018 Accepted; 0019–0023 Proposed).
- `docs/GLOSSARY.md` — novo termo `Converter` no Tooling section
  documentando o pattern proto ↔ domain.

**Marco**: H-3.b é a **primeira promoção de ADR Proposed→Accepted da
Fase Harvest**. Estabelece o pattern operacional de "promover no mesmo
commit que entrega o último critério" — verificado: ADRs 0017/0018
flipam status no commit 8, no mesmo PR que os critérios 3/4 (0017) e
4/5 (0018) são entregues nos commits 2-7.

Entregas H-4 (esta sessão):

- `internal/shared/clock/{clock,clock_test}.go` — `Clock` interface
  + `SystemClock` + `FixedClock` (commit 1).
- `internal/shared/random/{random,random_test}.go` — `Source`
  interface + `SystemSource` (seeded from crypto/rand) +
  `SeededSource` (commit 1).
- `internal/shared/replay/{doc,fixture,recorder,player,replay_test}.go`
  — record/replay infrastructure com JSONL fixture format, stdlib
  encoder (não protojson — instabilidade documentada), payload
  normalize empty→[]byte{} (commit 2).
- `internal/shared/sequencer/{sequencer,sequencer_test}.go` —
  per-StreamKey monotonic counter com Snapshot/Restore/Peek,
  concurrent-safe (-race verified) (commit 3).
- `internal/adapters/nats/natssequencer/{doc,store,store_unit_test,store_roundtrip_test}.go`
  — KV adapter para `SEQUENCER_STATE_LATEST`, key format por
  ADR-0020, owner-isolation no LoadSnapshot (commit 4).
- `internal/shared/metrics/{sequencer_metrics,sequencer_metrics_test}.go`
  — Counter `marketfoundry_consumer_seq_gap_total{stream_key}`
  (commit 5).
- `cmd/{execute,store}/run.go`, `cmd/gateway/compose.go`,
  `internal/actors/scopes/{execute,store}/*supervisor*`,
  `internal/actors/scopes/execute/venue_adapter_actor.go`,
  `internal/actors/scopes/store/query_responder_actor.go`,
  `internal/adapters/nats/natsexecution/control_kv_store.go`,
  `internal/application/executionclient/verify_session.go` —
  Clock plumbing aditivo (campos + WithClock setters/options),
  cmd/* instanciam `clock.SystemClock{}` (commit 6.0).
- `internal/domain/execution/{verification,control,activation,session}.go`
  — 5 production call sites de `time.Now` migrados para
  `clock.Clock` parameter (commits 6a/6b/6c/6d). Cascade incluiu
  ControlKVStore.Get split de nil-receiver vs nil-bucket guard
  para preservar ADR-0012 fail-open posture.
- `tools/raccoon-cli/src/analyzers/check_determinism.rs` —
  novo analyzer (~370 LoC, 12 unit tests). Scope: `internal/domain/*.go`
  excluding `*_test.go`. Detecta banned imports + banned
  symbols com 3 safeguards (skip comments, skip string literals,
  skip identifier substrings) (commit 7).
- `tools/raccoon-cli/src/{cli,application,gate}/mod.rs` — CLI
  variant `Determinism`, dispatch handler, gate Step 7 (drift-detect
  renumbered to Step 8) (commit 8).
- `Makefile` — `make determinism-check` target + ##@ Goldens
  section com `make golden-regen SCOPE=<scope>|all` (refuse without
  SCOPE) (commits 8 e 9).
- `internal/shared/replay/golden_test.go` +
  `golden_data_test.go` + `golden_regen_test.go` (build tag
  `goldenregen`) — golden test + N=50 byte-stability + regen
  helper. Fixture: `testdata/golden/replay-cycle/synthetic-100.jsonl`
  (100 events, distribuição agreed em PAUSE #5) (commit 9).
- `docs/decisions/0019-deterministic-replay-time-invariants.md`,
  `docs/decisions/0020-sequencing-and-time-normalization.md` —
  Status `Proposed → Accepted`; Changelog "Promoted to Accepted";
  criterion-by-criterion mapping section. References section em
  ADR-0019 inclui rationale do test-file exemption + protojson
  instability (commit 10).
- `docs/programs/PROGRAM-0002-wire.md` — Status `Active → Closed`;
  todos os 15 critérios de aceite marcados [x]; Changelog entry de
  closure (commit 10).
- `docs/TRUTH-MAP.md`, `docs/RUNTIME.md`, `docs/GLOSSARY.md` —
  rows de ADR-0019/0020 movidas para Implemented com anchors
  reais; bucket `SEQUENCER_STATE_LATEST` adicionado a RUNTIME;
  7 novos termos no GLOSSARY (Clock, Random, Replay, Recorder,
  Player, Golden test, Determinism); summary counts atualizados
  (23 ADRs, 17 KV buckets, 93 verify checks, 2 PRDs) (commit 10).
- `docs/RESUMPTION.md` — esta seção (commit 10).

**Marco**: H-4 fecha a Fase Wire (PROGRAM-0002 Closed). Dual ADR
promotion (0019 + 0020). `internal/domain/` production code agora
**mecanicamente livre** de `time.Now` direto via raccoon-cli
analyzer integrado no gate. Próxima fase: PROGRAM-0003
(Observability) começa em H-5.

**Cascade discovery em H-4**: análise prévia ao commit 1 identificou
5 production call sites de `time.Now` em `internal/domain/execution/`
(vs 1 arquivo de teste assumido no prompt). User-confirmed mitigation:
Option (C) — migração de production code + test-file exemption no
analyzer. Sem erratum a ADR-0019; critério 2 cumprido literalmente
("existing direct time.Now call sites in `internal/domain/` migrated").

---

Entregas H-6.c.2 (esta sessão):

- **Commit 1** (`df5ea36`):
  **paper_order_evaluator test migration** (5 files, +40/-28).
  ~28 test sites across 4 _test.go files migrated from
  `appexec.NewPaperOrderEvaluator(...)` to
  `appexec.NewPaperOrderEvaluatorForInstrument(...)` via uniform
  sed pattern. New `solUSDTPerp(t)` fixture added to external
  test helpers (third base alongside btcUSDTPerp/ethUSDTPerp).
  Production code untouched — dual-API coexists during migration
  window. Pre-existing out-of-scope gofmt drift discovered in
  `live_mainnet_dryrun_test.go` (untouched, not bundled — 6th
  instance of H-6.c.1 retrospective gofmt pattern).
- **Commit 2** (`7e3c6b8`):
  **composite_reader silent sites → warn-and-emit-zero**
  (1 file, +43/-5). The 5 silent error-discard sites in
  `composite_reader.go` (signal/decision/strategy/risk/execution
  composite queries) converted to match the warn-and-emit-zero
  pattern used by the 8 sister sites in candle/decision/
  execution/risk/signal/strategy readers. Partial-chain-assembly
  contract preserved (zero Instrument still propagates to
  maintain stage population); the only behavior change is
  structured Warn log emission for operator visibility. All 13
  `reconstructInstrumentFromLegacy` callers now uniform.
  File docstring extended with TODO(H-6.d) pointer for the
  deferred log-emission unit canary (gap symmetric with the
  8 sister sites that also lack such canaries).
- **Commit 3** (`3168a76`):
  **ReviewTransform string_filter + DecisionTriageItem godoc**
  (3 files, +81/-23). Zero production code change — the DTO
  symbol projection chain (decision.Decision.Instrument →
  d.VenueSymbol() → ReviewTransform.Symbol →
  DecisionTriageItem.Symbol) is in the post-canonical state
  since H-6.b. Work reduces to policy declaration + inline
  godoc. New `[domain_types.review_transform]` entry in
  `domain_types.toml` with `migration_state = "string_filter"`
  + full rationale comment. Inline godoc added to
  ReviewTransform struct + Symbol field and to
  DecisionTriageItem struct + Symbol field documenting the
  string-filter semantics. In-scope gofmt drift bundled (7th
  retrospective instance — pre-existing alignment drift in
  ReviewInputs/ReviewTransform/Finding/SessionTriageItem
  struct field blocks within the touched files).
- **Commit 4** (`70457f5`):
  **testnet adapters use BindingTarget.Instrument()**
  (2 files, +41/-2). Per Decisão #2 — option (b) confirmed
  after pre-flight verification showed option (a) port-signature
  refactor cascade = 12 files (5 prod + 7 test), exceeding the
  H-6.c.2 threshold. Both testnet adapter call sites
  (`binance_spot_testnet_adapter.go:391`,
  `binance_futures_testnet_adapter.go:395`) replace
  `instrumentFromBinding(SOURCE, symbol)` with
  `appingest.BindingTarget{Source: SOURCE, Symbol:
  symbol}.Instrument()` + warn-and-emit-zero log emission.
  Uniform pattern with commit 2. Architectural debt recorded
  for H-6.f (see PROGRAM-0004 scope notes).
- **Commit 5** (`789dfdb`):
  **delete legacy NewPaperOrderEvaluator + helper + symbol
  field** (11 files, +25/-65). Deletes
  `internal/application/execution/instrument_binding.go`
  (34 LoC); removes legacy ctor + dead `symbol` field +
  unused `strings` import from
  `paper_order_evaluator.go`. Updates inline comment in
  `strategy_consumer_actor.go`. **8 cross-scope stragglers**
  migrated: 1 derive (s470_lineage_causality_test) + 2 risk
  (risk_scaling_test) + 6 integration-tagged (writerpipeline +
  4 natsexecution + 1 live_consumer_flow). Discovery: make
  verify masks integration-tagged build failures — the
  explicit `go test -tags=integration -run DOES_NOT_EXIST`
  check surfaced the 6 integration stragglers that would
  have shipped broken to CI. Lesson reinforces H-6.c.1
  commit 13 pre-push discipline.
- **Commit 6** (`db0d5f1`):
  **execute.venue-adapter canary** (1 file, +150). New file
  `execute_venue_adapter_canary_test.go` (no integration tag,
  runs in regular `make verify`). 2 tests / 2 passes lock the
  37f8ddd contract:
    * TestPaperOrderEvaluator_PreservesInstrument_WithSyntheticSource
      (unit shape): direct ctor + Evaluate + assert
      `intent.Instrument == input && !intent.Instrument.IsZero()`.
    * TestStrategyConsumerActor_PreservesInstrument_WithSyntheticSource
      (actor shape): spawn strategy_consumer_actor + send
      strategyReceivedMessage with synthetic Source +
      canonical Instrument + assert intent.Instrument matches.
  Uses existing fixtures (`btcUSDTPerpExec`, `spawnTestStrategy`,
  `waitForIntent`).
- **Commit 7** (`e337be3`):
  **anti_patterns.toml exception list 11 → 8** (1 file,
  +11/-16). Net -3: removed the 3 execution package entries
  (paper_order_evaluator + 2 testnet adapters). Kept: 1
  executionclient entry (H-6.f scope) + 7 ClickHouse readers
  (H-6.d scope). composite_reader.go re-tagged from "H-6.c.2
  treatment" to "H-6.d helper removal". Prose updates:
  instrumentFromBinding migration progress reflects execute
  scope completion; reconstructInstrumentFromLegacy why text
  drops the "8 warn + 5 silent" split (all 13 now uniform).
- **Commit 8** (este commit): TRUTH-MAP / RESUMPTION /
  PROGRAM-0004 closure.

**Marco**: H-6.c.2 fecha a migração application-layer pass-
through para execute scope + uniformiza a ClickHouse adapter
error-handling. **5 dos 6 `instrumentFromBinding` helpers
eliminados** (signal/decision/strategy/risk/execution); apenas
`executionclient/instrument_binding.go` permanece para H-6.f
(blocked by LifecycleEntry contract migration). Todos os 13
`reconstructInstrumentFromLegacy` callers em ClickHouse readers
agora uniformes (warn-and-emit-zero), pending helper removal
em H-6.d schema migration. ADR-0021 critério #2 **ainda não
literalmente satisfeito** — restam 1 helper em executionclient
+ 13 reconstruction callers em ClickHouse. **ADR-0021 permanece
`Proposed`**; promoção é evento atómico em H-6.f.

**Métricas H-6.c.2**: 8 commits, ~30 test sites migrated, 1
helper file deleted (34 LoC), 2 testnet adapter sites converted
to BindingTarget.Instrument(), 5 composite_reader silent sites
treated, 1 new policy entry (ReviewTransform string_filter), 8
cross-scope stragglers fixed (1 derive + 2 risk + 6 integration),
1 canary test suite added (2 tests / 2 passes), anti_patterns
exception list reduced 11 → 8. make verify GREEN every commit;
raccoon-cli --profile ci PASSED 10/10 every commit (lesson 13
of H-6.c.1 enforced).

**H-6.f architectural debt — QueryOrder port refactor candidate
(recorded during H-6.c.2 Decisão #2 verification)**:

Option (a) cascade analysis: changing the `ports.VenueQueryPort`
interface signature from `QueryOrder(ctx, clientOrderID, symbol
string)` to `QueryOrder(ctx, clientOrderID string, instrument
instrument.CanonicalInstrument)` would eliminate the residual
source-string reconstruction in the testnet adapters entirely
— the caller (`post200_reconciler.go:66`) already holds the
canonical Intent.Instrument and would pass it directly.
Architecturally cleaner than the (b) BindingTarget.Instrument()
approach used at H-6.c.2 (reconstruction in adapter layer is
semantically the wrong layer).

Cascade size — 12 files / >8 threshold (sub-onda H-6.c.2
exceeded):

- Production (5): `internal/application/ports/venue.go` (port
  signature), `binance_spot_testnet_adapter.go` (impl line 353),
  `binance_futures_testnet_adapter.go` (impl line 355),
  `segment_router.go:69-83` (wrapper), `post200_reconciler.go:66`
  (single non-test caller).
- Tests (7): `post200_reconciler_test.go` (1 fakeQueryVenue impl
  + 3 callback signatures), `s405_spot_venue_acceptance_fill_test.go`
  (2 sites), `s422_futures_venue_connectivity_fill_test.go`
  (2 sites), `s423_futures_rejection_partial_fill_test.go`
  (3 sites), `s416_futures_venue_acceptance_fill_test.go`
  (2 sites), `s416_futures_venue_lifecycle_test.go` (1 site),
  `s405_spot_venue_lifecycle_test.go` (1 site).

Current state (post-H-6.c.2): testnet adapters use
`BindingTarget.Instrument()` with warn-and-emit-zero fallback.
Eliminates the `instrumentFromBinding` helper file in the
execution package, but the reconstruction itself remains in
adapter layer. H-6.f candidate refactor: port signature
migration to Instrument. Tractable as dedicated H-6.f sub-task
alongside executionclient + LifecycleEntry migration.

**Próxima sub-onda destravada após merge**: H-6.d — ClickHouse
schema migration with canonical `base`/`quote`/`contract`
columns + back-compat read window. Sub-onda sequencing policy
estrita: H-6.d abre branch APENAS após merge desta PR (H-6.c.2)
em `main`.

---

Entregas H-6.c.1 (sessão anterior — PR #30 mergeada em `main` em `8125e6c`, 2026-05-27):

- **Commit 1** (`9c14ac2`):
  **Anti-patterns analyzer + policy installation**. New file
  `tools/raccoon-cli/policies/anti_patterns.toml` declares the
  forbidden source-string Instrument reconstruction functions
  (`instrumentFromBinding` + `reconstructInstrumentFromLegacy`).
  `check_instruments` analyzer gains
  `load_anti_patterns_policy` + `scan_anti_pattern` +
  `collect_production_go_files` + 5 unit tests (~458 LoC Rust).
  Severity is `warning` during the migration window; flips to
  `error` in H-6.f once helpers are eliminated. Rationale: the
  pre-flight 5 of H-6.c documented that production Source values
  include synthetic strings outside the hardcoded
  binances/binancef mapping (`"binance"`, `"binance_spot"`,
  `"derive"`, `"clickhouse"`, `"unknown_exchange"`,
  `"execute.venue-adapter"`); each call site is a potential
  silent-zero regression analogous to commit 37f8ddd in H-6.b'.
- **Commits 2-5** (`03f32a4`, `09e0537`, `24fd400`, `d147456`):
  **NewXxxForInstrument pass-through constructors** added across
  4 application packages (signal/decision/strategy/risk).
  14 constructors total: 6 in signal (RSI, ATR, Bollinger,
  EMACrossover, MACD, VWAP), 3 in decision (RSIOversold,
  EMACrossover, BollingerSqueeze), 3 in strategy
  (MeanReversion, SqueezeBreakout, TrendFollowing), 2 in risk
  (DrawdownLimit, PositionExposure). Each constructor accepts
  `(source string, inst CanonicalInstrument, timeframe int)`
  bypassing `instrumentFromBinding`. Legacy `NewXxx(source,
  symbol, timeframe)` wrappers retained transitorily, delegating
  via the existing helper. 4 new `instrument_passthrough_test.go`
  files document the pass-through contract per package.
- **Commit 6** (`849768b`):
  **Boundary helper + derive actor cascade** (32 files, +490/-123).
  New `(BindingTarget).Instrument() (CanonicalInstrument, error)`
  method in `internal/application/ingest/binding.go` with a
  declarative `venueSourceContract` registry (binances→Spot,
  binancef→Perpetual). Returns explicit error for unknown
  sources — synthetic sources are *intentionally absent* from
  the registry, surfacing the 37f8ddd failure mode rather than
  hiding it. 5 derive Config structs gain canonical Instrument
  field (Signal/Decision/Strategy/Risk/Execution Evaluator
  configs). 10 derive actor files switch the application
  constructor call to `NewXxxForInstrument(cfg.Source,
  cfg.Instrument, ...)`. `source_scope_actor.onActivateSampler`
  computes Instrument once at the boundary via
  `msg.Target.Instrument()` and skips activation with a
  structured Error log on failure. `derive_supervisor` cascades
  the inst parameter through 12 factory NewActor callbacks.
  15 derive test files gain `Instrument: btcUSDTPerp()` on
  Config literals (Python-script-driven migration). **(P1)
  commit-as-is discipline applied**: fragmenting into
  6a-production + 6b-tests was rejected because it would
  produce a semantically invalid intermediate state (actors
  compile but instantiate evaluators with zero Instrument).
  R2 cleanup applied during landing — collateral gofmt drift
  from a `gofmt -w internal/` overreach was soft-reset + scoped
  re-stage to the 32 intentional files.
- **Commit 7a** (`8fb781e`):
  **Signal package legacy cleanup** (15 files, +108/-184).
  Deletes `internal/application/signal/instrument_binding.go`
  (45 LoC), removes `symbol string` field + legacy `NewXxxSampler`
  wrapper from all 6 sampler.go files. 52 test sites migrated
  via uniform sed pattern to `NewXxxSamplerForInstrument` with
  `btcUSDTPerp`/`ethUSDTPerp` fixtures. New
  `instrument_fixtures_test.go` (package signal) + extended
  `instrument_passthrough_test.go` (package signal_test) provide
  the fixtures.
- **Commit 7b** (`df04a94`):
  **Decision package legacy cleanup** (9 files, +95/-144). Same
  pattern as 7a. Deletes `decision/instrument_binding.go` (36 LoC).
  6 test sites migrated. Discovery: 1 legacy caller in
  `derive/s470_lineage_causality_test.go` missed by commit 6's
  Python script (caller is a test, not production — pulled into
  7b as single-line fix).
- **Commit 7c** (`aa9ce66`):
  **Strategy package legacy cleanup** (12 files, +113/-134).
  Same pattern. Deletes `strategy/instrument_binding.go` (34 LoC).
  6 strategy test sites + 4 cross-scope stragglers migrated:
  1 in `derive/s470_lineage_causality_test.go`, 1 in
  `execute/s373_structural_test.go`
  (using existing `btcUSDTPerpExec(t)` fixture), and 2 in
  `execute/e2e_derive_to_execution_test.go` +
  `store/e2e_derive_to_store_test.go` (added parameterless
  `btcUSDTPerpDerive` IIFE fixtures since the derive event
  helpers have no testing.T threaded through 13 call sites).
  In-scope gofmt drift bundled in `s373_structural_test.go`
  (~5 LoC struct field alignment, documented in commit body
  per "honesty over convenience").
- **Commit 7d** (`5ac42df`):
  **Risk package legacy cleanup** (9 files, +117/-146). Same
  pattern. Deletes `risk/instrument_binding.go` (34 LoC). ~50
  risk test sites migrated (largest count of any 7x commit:
  16 drawdown + 16 position + 17 risk_scaling + 8
  multi_symbol_concurrency). Fixture file extended with
  `btcUSDTPerp`/`ethUSDTPerp`/`solUSDTPerp` + `mustPerpOrSpot`
  helper + `instrumentForSymbol(sym)` mapper for parameterized
  struct cases. Final derive scope straggler in
  `s470_lineage_causality_test.go` migrated (3 total across
  7b/7c/7d). In-scope drift bundled in `risk_scaling_test.go`
  (7c precedent); out-of-scope drift in `risk_scaling.go`
  (production, 1 LoC trailing newline, untouched by migration)
  reported but NOT bundled.
- **Commit 8** (`cef879b`):
  **Synthetic-source canary integration tests** (1 new file,
  +287 LoC). New `internal/actors/scopes/derive/
  synthetic_source_canary_integration_test.go` adds derive-scope
  canary tests that fix the regression-shape canary established
  by commit 6's `BindingTarget.Instrument()` at the wiring
  level. 3 tests / 15 subtests: rejection-at-boundary
  (6 synthetic sources), full activation flow with
  `canaryActivator` stand-in (verifies log emission +
  rejection counters), legitimate-activation-proceeds
  (binances spot + binancef perpetual must NOT be
  over-rejected). Avoids full SourceScopeActor instantiation
  (NATS publisher dependency); end-to-end NATS-bound coverage
  is deferred to make smoke / live integration runs.
- **Commit 9** (`f1f961c`):
  **Policy progress documentation**. Updates
  `tools/raccoon-cli/policies/anti_patterns.toml`
  `instrumentFromBinding` entry's `why` text with per-package
  migration progress (4 eliminated + 2 remaining → H-6.c.2 / H-6.f)
  and `help` text references commits 2-7d as the migration
  pattern reference. **Schema unchanged** (function-based, not
  per-package) per the architectural decision that filesystem
  is the source of truth for migration status — adding a
  per-package `status` field would duplicate filesystem reality
  and create drift risk. `reconstructInstrumentFromLegacy`
  entry unchanged (13 call sites in clickhouse, H-6.c.2 scope).
- **Commit 10** (este commit): TRUTH-MAP / RESUMPTION /
  PROGRAM-0004 closure + gofmt drift retrospective + per-package
  schema consideration note.

**Marco**: H-6.c.1 fecha a migração application-layer
pass-through para derive scope — `instrumentFromBinding`
helper **completamente eliminado** de signal/decision/strategy/
risk (4 packages). `BindingTarget.Instrument()` (com signature
error-returning) é estabelecido como o canonical
reconstruction point para legítimo boundary
(source, symbol) → CanonicalInstrument. Derive actors agora
computam Instrument uma única vez na entrada da activação
(`source_scope_actor.onActivateSampler`) e fazem pass-through
em todo o cascade signal/decision/strategy/risk/execution.
Synthetic sources (`"binance"`, `"binance_spot"`, `"derive"`,
`"clickhouse"`, `"unknown_exchange"`,
`"execute.venue-adapter"`) são rejeitados explicitamente com
log estruturado — NÃO mais silent-zero.

**Métricas H-6.c.1**: 10 commits, ~250 test sites migrated,
4 helper files deleted (~150 LoC), 14 NewXxxForInstrument
constructors added (4 packages × 2-6 constructors each),
1 new boundary helper (`BindingTarget.Instrument()`),
1 anti-patterns policy file + Rust analyzer extension
(~458 LoC), 1 canary integration test suite (15 subtests).
make verify GREEN every commit; lefthook hooks GREEN
(pre-commit gofmt + commit-msg format + post-commit drift).

**Próxima sub-onda destravada após merge**: H-6.c.2 —
execute scope migration: 3 remaining `instrumentFromBinding`
callers in `application/execution` (paper_order_evaluator +
2 testnet adapters), ClickHouse `reconstructInstrumentFromLegacy`
treatment (8 warn-and-emit-zero + 5 silent discard in
composite_reader), ReviewTransform string_filter migration,
DecisionTriageItem cascade, and the 37f8ddd integration test
(now an explicit canary against the regression-shape).
Sub-onda sequencing policy estrita: H-6.c.2 abre branch
APENAS após merge desta PR (H-6.c.1) em `main`.

**Pattern observed — gofmt drift accumulation (H-6.c.1
retrospective)**:

H-6.c.1 encountered gofmt drift in **5 instances across the
10 commits**:
- Commit 4 (strategy ForInstrument constructors): in-scope
  drift detected during landing.
- Commit 6 (boundary helper + actor cascade): scope-expansion
  drift caught by R2 cleanup — `gofmt -w internal/`
  inadvertently captured 48 unrelated files; soft-reset +
  scoped re-stage applied.
- Commit 7a (signal cleanup): in-scope drift in touched files.
- Commit 7c (strategy cleanup): in-scope drift in
  `s373_structural_test.go` (file touched by migration).
- Commit 7d (risk cleanup): in-scope drift in
  `risk_scaling_test.go` (touched) + out-of-scope drift
  detected in `risk_scaling.go` (production, untouched by
  migration; not bundled per scope discipline).

This frequency suggests **systematic gofmt drift accumulated
in the codebase that was previously invisible** — either not
enforced by CI consistently, or enforced historically but
bypassed at some point. The pre-commit hook
(`gofmt -l {staged_files}`) only catches drift in staged
files, so untouched files with accumulated drift remain
hidden until an unrelated commit happens to touch them.

Candidate mitigations (deferred to H-6.f or dedicated
audit wave; **decision pending owner**):

1. Add pre-commit hook that runs `gofmt -d` (detect, don't
   modify) on full repo; fails if drift detected anywhere.
   Forces explicit cleanup decision per commit.
2. Dedicated commit `chore(gofmt): repository-wide drift
   audit + cleanup` running `gofmt -w internal/` once,
   committed as cosmetic-only with no semantic changes.
3. CI step in `make verify` that validates zero drift in
   entire repo, not just modified files.

Recommendation order: option 2 first (one-shot cleanup),
then option 1 or 3 to prevent recurrence.

**Future consideration — anti_patterns.toml schema
(H-6.c.1 retrospective)**:

The current `anti_patterns.toml` schema is function-based
(one entry per forbidden function name). H-6.c.1 commit 9
documented migration progress in prose within the existing
`why`/`help` text rather than refactoring to per-package
status entries. The function-based schema is appropriate
because:

1. Filesystem is source of truth — helper file deletion
   means migrated; a `status` field would duplicate
   filesystem reality.
2. Anti-patterns are function names that may exist in N
   packages; per-package decomposition is unnecessary when
   the scanner already finds zero callers in deleted-helper
   packages.

If drift ever appears between policy declaration and
filesystem reality (e.g., helper exists but policy says
migrated, or vice-versa), refactoring to a per-package
schema with enforceable `status` field becomes justified.
This is **not justified at H-6.c.1 closure**; recorded
here to prevent the same discussion in a future onda.

---

Entregas H-6.b'' (sessão anterior — PR #29 mergeada em `main` em `54a2706`, 2026-05-26):

- **Commit 1** (`888b162`):
  **Analyzer: `string_filter` migration_state** added to
  `tools/raccoon-cli/policies/domain_types.toml` schema + analyzer
  acceptance. New state documents the architectural choice that a
  type's venue-native string field is canonical by design (no
  Instrument upgrade applies). Tolerated like `pending` — no
  enforcement — but conveys permanence rather than transience.
  Helps prevent the H-6.b' regression-shape (commit 37f8ddd: silent
  zero Instrument from source-string reconstruction at a query
  boundary) by capturing the decision in policy. Analyzer gains
  +1 unit test (15 total). Rationale fully documented in the
  analyzer rustdoc header and the policy file's header comment.
- **Commit 2** (`3a40536`):
  **pairing.Leg migration** (1 prod file + 5 test files; net
  +148/-66). `Leg.Symbol string` → `Leg.Instrument
  instrument.CanonicalInstrument` + `VenueSymbol() string`
  transitory accessor. M1 invariant adopts native Go struct equality
  (`entry.Instrument != exit.Instrument`); CanonicalInstrument is
  composed of three string-typed components and is comparable by
  construction (no Equal() method needed). IntentToLeg passthrough
  on `intent.Instrument` — zero source-string reconstruction.
  MatchFIFO RoundTrip{} construction uses S472-style projection
  bridge `Symbol: leg.VenueSymbol()` to keep compile-green while
  RoundTrip.Symbol still exists (it migrates in commit 3). Three
  IIFE-vars `btcUSDTPerp`/`btcUSDTSpot`/`ethUSDTSpot` replace the
  prior `func btcUSDTPerp(t)` helper following the
  get_composite_chain_test.go precedent; makeIntent/makeLeg
  fixtures consistently use Instrument: btcUSDTSpot to match their
  Source: "binance_spot" semantically.
- **Commit 3** (`2675d99`):
  **pairing.RoundTrip migration + triage pull-forward** (1 prod
  file in pairing/ + 1 prod file in triageclient/ + 6 test files;
  net +93/-65). `RoundTrip.Symbol string` →
  `RoundTrip.Instrument instrument.CanonicalInstrument` +
  `VenueSymbol()` accessor. Denormalization invariant per Decisão
  #3 documented inline:
  `RoundTrip.Instrument == Entry.Instrument == Exit.Instrument`
  (enforced by MatchFIFO construction + M1). MatchFIFO sites
  switch from S472 bridge to clean passthrough
  `Instrument: leg.Instrument`. **Pull-forward**: triage population
  site at `triageclient/get_roundtrip_triage.go:74` (Symbol:
  review.Symbol → Symbol: review.VenueSymbol()) traveled into
  commit 3 by compile pressure — RoundTripReviewItem embeds
  pairing.RoundTrip anonymously, so removing RoundTrip.Symbol
  immediately breaks the promoted field access. Pattern matches
  H-6.b' commit 1 precedent (ExecutionIntent pulled
  venue_adapter_actor.go forward by compile pressure). The
  semantically corresponding commit (commit 5 in the plan) was
  retained as test-only.
- **Commit 4** (`0236315`):
  **CrossSessionWindow rename per Decisão #2 (b)** (1 prod file
  in pairing/ + 2 prod files in analyticalclient/ + 2 test files +
  policy entry; net +59/-26). `Symbol string` → `VenueSymbol string`
  (rename only — no Instrument upgrade). JSON tag `"symbol"` →
  `"venue_symbol"`. Validate() reads `w.VenueSymbol != ""`. The
  two construction sites in analyticalclient pass `query.Symbol`
  through verbatim with no canonical reconstruction. Struct godoc
  documents the architectural choice inline: VenueSymbol is
  metadata only, NOT consulted by matching algorithm, only
  validated for non-emptiness — promoting would force regression-
  prone source-string reconstruction (commit 37f8ddd precedent).
  Policy entry `cross_session_window` flips `pending` →
  `string_filter` with inline rationale block.
- **Commit 5** (`17c0628`):
  **test(triage): get_roundtrip_triage projection coverage**
  (1 new test file +133; Decisão #5β canary). Closes the test-
  coverage gap flagged in pre-flight 7. Two tests:
  TestGetRoundTripTriage_ProjectsVenueSymbolFromInstrument asserts
  the happy-path projection (BTC/USDT-spot Instrument → "btcusdt"
  Symbol via the embedded RoundTrip.VenueSymbol()). The second
  test, TestGetRoundTripTriage_ZeroInstrumentProducesEmptyString,
  is the regression-detection canary: a zero-valued
  pairing.RoundTrip.Instrument MUST surface as empty Symbol in
  the wire shape, observable rather than silently defaulted.
  stubRoundTripReviewer + btcUSDTSpotForTriage(t) helper provide
  full fixture control over the embedded RoundTrip.
- **Commit 6** (`97d8f21`):
  **test(smoke): pairing/review instrument projection canary**
  (1 file +62; Decisão #5γ). Inline section in
  `scripts/smoke-analytical-e2e.sh` Phase 5 (after Executions
  filter validations). Tri-state semantics — HTTP 200 + reviews
  populated + instrument.base populated → PASS; reviews empty
  → WARN (canary inapplicable since smoke setup does not
  explicitly seed matched buy+sell within FLUSH_WAIT); reviews
  populated + instrument.base empty → FAIL (regression-shape).
  WARN keeps the canary honest under data scarcity while
  preserving FAIL semantics when paired data exists. Tri-state
  validated offline via python snippet simulation (no live stack
  needed for syntax verification).
- **Commit 7** (`96475df`):
  **chore(policy): flip pairing_leg + pairing_round_trip to
  migrated** in `tools/raccoon-cli/policies/domain_types.toml`.
  Both types already carry `Instrument CanonicalInstrument` field
  + `VenueSymbol() string` method (added in commits 2 and 3); this
  commit activates the analyzer enforcement going forward.
  cross_session_window stays `string_filter` (set in commit 4).
- **Commit 8** (este commit): TRUTH-MAP / RESUMPTION /
  PROGRAM-0004 closure + H-6.f scope revision note.

**Marco**: H-6.b'' fecha a migração da Pairing chain — Leg e
RoundTrip carregam `Instrument CanonicalInstrument` +
`VenueSymbol()` transitory accessor; CrossSessionWindow renomeia
o field para refletir sua semântica de query filter (Decisão
#2 (b)). Total agora: **12 dos 15 domain types iniciais** com
Symbol field migrados (3 de H-6.a/H-6.b + 7 de H-6.b + 3 de
H-6.b' + 2 nesta sub-onda) **+ 1 type formalmente declarado
`string_filter`** (CrossSessionWindow). ADR-0021 critério #2
**ainda não literalmente satisfeito** — restam os ~6
`instrumentFromBinding` helpers em application/* e o
`reconstructInstrumentFromLegacy` em adapters/clickhouse cujos
errors são descartados em 11 chamadas. **ADR-0021 permanece
`Proposed`**; promoção é evento atômico em H-6.f.

**H-6.f scope revision (post-pré-flight 6 de H-6.b'')**:
pré-flight 6 descobriu que o débito real de H-6.f é maior que
"remove transitory methods". Scope revisado:
1. Audit e remoção dos 6 helpers `instrumentFromBinding` em
   `application/{signal,decision,strategy,risk,execution,
   executionclient}/` — todos hardcoded para `binances`/`binancef`
   + `USDT` quote, retornam zero silenciosamente para qualquer
   outro input.
2. Audit `reconstructInstrumentFromLegacy` em
   `internal/adapters/clickhouse/candle_reader.go:150` —
   retorna error mas o error é descartado em 11 chamadas em
   `composite_reader.go` e `execution_reader.go`. Either
   propagate errors or replace with Instrument pass-through
   where upstream carries it.
3. Migrate callers para receber Instrument diretamente de
   upstream (pattern estabelecido por
   `NewPaperOrderEvaluatorForInstrument` em H-6.b' commit
   37f8ddd).
4. Remover métodos `VenueSymbol()` apenas após todos os callers
   migrarem.
5. Promover ADR-0021 a `Accepted` quando critério #2 estiver
   literalmente satisfeito: zero source-string-based instrument
   reconstruction em production code.

Esta revisão é capturada também em PROGRAM-0004 Changelog.

**8 commits (plano declarava 9 — consolidação documentada)**:
o plano original tinha commits 3 (RoundTrip migration) e 5
(triage production line update) separados. Compile pressure
forçou consolidação: removing RoundTrip.Symbol immediately
breaks `review.Symbol` access in `get_roundtrip_triage.go` via
anonymous embedding. Commit 3 absorveu o 1-line touch em
triage; commit 5 do plano permaneceu como test-only (canary
coverage per Decisão #5β). Per H-6.b' precedent (PR #28):
pull-forward by compile pressure é documentado no commit
afetado, não escondido em renumbering.

**Próxima sub-onda destravada após merge**: H-6.c — migration
de Application-layer query types em `analyticalclient`/
`triageclient` (DecisionTriageItem population site downstream;
ReviewTransform DTO; etc.) e início do sunset transitorio
`instrumentFromBinding` per H-6.f scope revision acima.
Sub-onda sequencing policy estrita: H-6.c abre branch APENAS
após merge desta PR (H-6.b'') em `main`.

---

Entregas H-6.b' (sessão anterior — PR #28 mergeada em `main` em `6b62d89`):

- **Commit 1** (`234193e`):
  **ExecutionIntent atomic migration** (~50 production sites + ~85
  test files). Domain type `execution.ExecutionIntent` migra
  `Symbol string` → `Instrument CanonicalInstrument` +
  `VenueSymbol()` transitory accessor. PartitionKey e
  DeduplicationKey composers em `internal/adapters/nats/natsexecution/`
  reescritos via `VenueSymbol()` para preservar back-compat de KV
  bucket layout. Production cascade abrange: actors
  (`derive/execution_publisher_actor.go`, `execute/venue_adapter_actor.go`,
  3 `store/*_projection_actor.go`), adapters (`nats/natsexecution/publisher.go`,
  `clickhouse/{execution,composite}_reader.go` e `writerpipeline/support.go`),
  application (`paper_order_evaluator.go`, `dry_run_submitter.go`,
  `paper_venue_adapter.go`, `post200_reconciler.go`, ambos
  `binance_*_testnet_adapter.go`), domain (`pairing.IntentToLeg`),
  cmd (`gateway/session_reader.go`), e analyticalclient
  (`contracts.go`, `get_decision_review.go`). Per-package
  `instrumentFromBinding(source, venueNative)` transitory helper
  adicionado em `internal/application/execution/`. ClickHouse readers
  reusam `reconstructInstrumentFromLegacy` de H-6.a.
- **Commit 2** (`4cccaf7`):
  **Attribution migration** (3 files).
  `effectiveness.Attribution.Symbol` → `.Instrument` (derived from
  `intent.Instrument` em `Classify`/`ClassifyPair`); `VenueSymbol()`
  transitory accessor adicionado. `Explain()` usa `.VenueSymbol()`
  em todos os fmt.Sprintf sites. effectiveness_test.go atualizado
  via helper `btcUSDTPerp(t)` já existente.
- **Commit 2.1** (`0e18664`):
  **chore**: remoção de `cmd/gateway/gateway` binário acidentalmente
  committado em commit 2 via `git add -A`. `.gitignore` line 163
  tem `/gateway` (root-only) mas não cobre nested
  `cmd/<name>/<name>`. Removido via `git rm --cached`.
- **Commit 3** (`4707ef7`):
  **AuditLifecycleEntry migration** (3 files).
  `execution.AuditLifecycleEntry.Symbol` → `.Instrument` +
  `VenueSymbol()`. `convertLifecycleEntries` em
  `executionclient/audit_session.go` reconstrói Instrument do
  `(source, symbol)` do LifecycleEntry DTO via novo
  per-package `instrumentFromBinding` em
  `internal/application/executionclient/instrument_binding.go`
  (sunset H-6.f — LifecycleEntry permanece string-based até read-path
  migration na mesma onda).
- **Commit 4** (`e8be08c`):
  **policy flip** em `tools/raccoon-cli/policies/domain_types.toml`:
  `execution_intent`, `attribution`, `audit_lifecycle_entry`
  flipam de `pending` → `migrated`. check-instruments analyzer
  re-run reporta 6/6 PASS; full make verify 10/10 analyzers,
  102/102 checks GREEN.
- **Commit 5** (este commit): TRUTH-MAP / RESUMPTION /
  PROGRAM-0004 closure.

**Marco**: H-6.b' fecha a migração da camada execution chain — 3
dos 8 domain types restantes pós-H-6.b agora carregam
`Instrument CanonicalInstrument` + `VenueSymbol()` transitory
accessor (ExecutionIntent + Attribution + AuditLifecycleEntry).
Total agora: **10 dos 15 domain types** com Symbol field
migrados (3 de H-6.a/H-6.b + 7 de H-6.b + 3 desta sub-onda).
ADR-0021 critério #2 ainda **não** literalmente satisfeito —
restam Pairing chain types (Leg, RoundTrip, CrossSessionWindow)
para H-6.b''. **ADR-0021 permanece `Proposed`**; promoção é
evento atômico em H-6.f.

**Triage drop closure note** (verbatim do user em pré-flight
Decisão #1): Triage population sites verified during pre-flight.
Zero sites required migration in this sub-wave:
- `DecisionTriageItem`: buffered by ReviewTransform DTO
  (application-layer); domain→DTO boundary migrated in H-6.b
  commit 4. DTO→Triage remains string-to-string until H-6.c
  migrates ReviewTransform.
- `ExecutionTriageItem`: type does not exist in codebase.
- `RoundTripTriageItem`: correctly deferred to H-6.b''
  (upstream RoundTrip migrates there).

**Sub-wave naming convention** (estabelecida nesta sub-onda):
- Documentation/prose: H-6.b, H-6.b', H-6.b'' (apostrophes
  distinguish dependency layers within the wave H-6.b family).
- Branch names / git tags: feat/h-6-b1-…, feat/h-6-b2-…
  (numeric suffix for portability across shells/CI tools where
  apostrophes are unsafe).

Established at H-6.b' (branch `feat/h-6-b1-execution-chain`);
applies retroactively to existing prose references. Documentada
em PROGRAM-0004 → "Sub-wave naming convention".

**Próxima sub-onda destravada após merge**: H-6.b'' — migration
de Pairing.Leg + RoundTrip + CrossSessionWindow + Triage
RoundTrip population site. Sub-onda sequencing policy estrita:
H-6.b'' abre branch APENAS após merge de H-6.b' em `main`.

---

Entregas H-6.b (sessão anterior):

- **Commit 1** ([`e303202`](https://github.com/FabioCaffarello/market-foundry/commit/e303202)):
  [`docs/programs/PROGRAM-0004-multi-venue.md`](programs/PROGRAM-0004-multi-venue.md)
  refined. H-6.b pre-flight discovered 15 domain types totaling
  174 test files (5× the master plan estimate). Sub-divided into
  **H-6.b** (Layer 1+2: Evidence + Signal/Decision/Strategy/Risk),
  **H-6.b'** (Layer 3+3': ExecutionIntent + Attribution +
  AuditLifecycleEntry), **H-6.b''** (Layer 4: Pairing chain +
  Triage population sites). Decision driven by dependency order
  to avoid semantic gaps (no type-not-migrated consuming a
  type-migrated). PRD updated with new sub-onda table and
  rationale before any code change.
- **Commit 2** ([`86fa59e`](https://github.com/FabioCaffarello/market-foundry/commit/86fa59e)):
  **EvidenceCandle atomic migration** (19 files). Domain type
  + CandleSampler (captures trade.Instrument) + KV/projection
  actors + ClickHouse reader (with new
  `reconstructInstrumentFromLegacy` transitory helper for the
  H-6.b→H-6.d window) + writer mapper + 9 test files. KV bucket
  layout preserved via `VenueSymbol()`.
- **Commit 3** ([`167dd76`](https://github.com/FabioCaffarello/market-foundry/commit/167dd76)):
  **EvidenceTradeBurst + EvidenceVolume consolidated** (20 files).
  Same atomic pattern, trivially-analogous types per user
  allowance.
- **Commit 4** ([`e021761`](https://github.com/FabioCaffarello/market-foundry/commit/e021761)):
  **Signal + Decision pair** (60 files). Domain types with
  `PartitionKey()` composer now compose via `VenueSymbol()` —
  bucket layout `{source}.{venuesymbol}.{timeframe}` stays
  identical. 6 signal samplers (`rsi`, `bollinger`, `ema_crossover`,
  `macd`, `vwap`, `atr`) + 3 decision evaluators
  (`rsi_oversold`, `bollinger_squeeze`, `ema_crossover`) gain
  internal `instrument CanonicalInstrument` field populated via
  package-local `instrumentFromBinding(source, venueNative)`
  TRANSITORY helper. Public sampler/evaluator constructor
  signatures unchanged (sunset H-6.c). 30 test files migrated
  via subagent (multi_symbol partition-key isolation tests
  added).
- **Commit 5** ([`de372f5`](https://github.com/FabioCaffarello/market-foundry/commit/de372f5)):
  **Strategy + RiskAssessment pair** (55 files). Same shape: 3
  strategy resolvers (`mean_reversion`, `squeeze_breakout`,
  `trend_following`) + 2 risk evaluators (`drawdown_limit`,
  `position_exposure`). `analyticalclient.get_decision_review`
  ChainSnapshot projections use `.VenueSymbol()`; consistency
  ChainSnapshot fields stay string per S472 invariant. 31 test
  files migrated.
- **Commit 6** ([`4e5aeb7`](https://github.com/FabioCaffarello/market-foundry/commit/4e5aeb7)):
  **`check-instruments` analyzer extended** with
  `policies/domain_types.toml` declarative migration-state per
  type. `migrated` types must have both
  `instrument.CanonicalInstrument` reference and
  `VenueSymbol() string` method; `pending` types tolerated.
  Pre-H-6.b deployments without the policy file get a skip (no
  hard fail). Analyzer grew 4 → 6 checks; total gate from 100
  → 102.
- **Commit 7** (este commit, este sessão): TRUTH-MAP / RESUMPTION
  / GLOSSARY closure.

**Marco**: H-6.b fecha a migração da camada derivative
analytics — 7 dos 15 domain types restantes agora carregam
`Instrument CanonicalInstrument` + `VenueSymbol()` transitory
accessor. KV bucket layout back-compat preservada via VenueSymbol
nos 5 `PartitionKey()` composers (Signal/Decision/Strategy/Risk +
o ExecutionIntent que continua em H-6.b'). ADR-0021 critério #2
("all domain-layer call sites migrated") ainda **não** literalmente
satisfeito — restam ExecutionIntent + Attribution + Pairing chain
para H-6.b'/b''. **ADR-0021 permanece `Proposed`**; promoção é
evento atômico em H-6.f.

**Mid-development discovery em H-6.b**: pré-flight em 5 passos
descobriu 15 domain types (não 5 conforme master plan original)
totalizando 174 test files (top 10 com 17–37 literais de Symbol
cada). Cascade ExecutionIntent sozinho tem 199 test sites;
pairing.Leg 101; pairing.RoundTrip 66 — todos individualmente
acima do threshold de 25 do prompt. Após pause-and-report, user
aceitou opção (D) — split por **dependency order**, garantindo
que cada sub-onda fecha sem buracos semânticos. Refinement
documentado em PRD-0004 ANTES de qualquer commit de código (P3).

**Pattern reuse**: o `VenueSymbol()` transitory accessor
introduzido em H-6.a foi reaplicado mecanicamente nos 7 types
desta sub-onda. Cada package-de-domain repete: `Symbol` field
removido, `Instrument CanonicalInstrument` adicionado, método
`VenueSymbol() string` derivando lowercase `base+quote`.
Adicionalmente os 5 types com `PartitionKey()` composer
(Signal/Decision/Strategy/Risk/Execution) preservam o shape do
KV bucket layout — `{source}.{venuesymbol}.{timeframe}` — via
`VenueSymbol()`, sem mudança de wire-format na chave de partição.

**Próxima sub-onda destravada após merge**: H-6.b' — migration
de ExecutionIntent + effectiveness.Attribution +
execution.AuditLifecycleEntry. Sub-onda sequencing policy
estrita: H-6.b' abre branch APENAS após merge de H-6.b em
`main`.

Entregas H-6.a (sessão anterior):

- **Commit 0 (erratum)**:
  [`docs/decisions/0021-canonical-instrument-and-venue-model.md`](decisions/0021-canonical-instrument-and-venue-model.md)
  — criterion #4 split into #4a (writer-side adapt; H-6.a, zero
  schema change) and #4b (ClickHouse migration; H-6.d). ADR stays
  `Proposed`; Changelog entry documents the erratum trigger.
  Criterion #2 (all domain-layer call sites migrated) stays
  literal — no erratum loophole.
- **Commit 1 (PRD-0004 opening)**:
  [`docs/programs/PROGRAM-0004-multi-venue.md`](programs/PROGRAM-0004-multi-venue.md)
  — Fase Multi-venue PRD. Six sub-ondas H-6.a–H-6.f + Onda H-7
  declared. Sub-onda sequencing policy stricter than P4: next
  sub-onda only opens after the previous merges to `main`.
  ADR-0021 promotes only in H-6.f when criterion #2 is literally
  satisfied. Transitory-method pattern documented for reuse by
  H-6.b–H-6.e.
- **Commit 2 (`internal/domain/instrument/` package)** — 4
  production files + 21 unit tests:
  - `asset.go` — `BaseAsset`, `QuoteAsset` types with
    `NewBaseAsset` / `NewQuoteAsset` constructors (trim +
    uppercase + ASCII A–Z 0–9 1–16-char validation).
  - `venue.go` — `Venue` enum restricted to `VenueBinance`,
    `VenueBinanceFutures` (only shipping adapters; new venues
    add entries when adapters ship, mirroring H-5 check-metrics
    discipline).
  - `contract_type.go` — `ContractType` enum (`spot`,
    `usdtfutures`, `coinfutures`, `perpetual`) per ADR-0021.
  - `canonical.go` — `CanonicalInstrument{Base, Quote, Contract}`
    with `New(base, quote, contract)`, `Symbol()` →
    `"{BASE}/{QUOTE}-{CONTRACT}"`, `FromSymbol(s)` parser,
    `Validate()`, `IsZero()`. JSON tags (`base`, `quote`,
    `contract`) for wire-format stability of embedding domain
    types.
- **Commit 3 (atomic — ObservationTrade + adapters + readers)** —
  13 files (4 production + 9 test). One commit because removing
  `Symbol string` breaks every reader; user explicitly rejected
  dual-write as toxic debt:
  - `internal/domain/observation/trade.go` — `Symbol string` →
    `Instrument CanonicalInstrument` (with JSON tag). New method
    `VenueSymbol() string` (option C — semantically distinct
    name, sunset H-6.f) returns lowercase `base+quote` derived
    form. Docstring documents lossy behavior for delivery
    contracts and H-6.e deferral.
  - `internal/adapters/exchanges/binances/aggtrade.go` —
    `Normalize` calls `parseSpotSymbol` which uppercases, asserts
    USDT suffix, splits base/quote, calls
    `instrument.New(base, "USDT", instrument.ContractSpot)`.
    Non-USDT quotes rejected at the boundary.
  - `internal/adapters/exchanges/binancef/aggtrade.go` —
    `Normalize` calls `parseFuturesSymbol` with package-level
    regex `var deliverySuffix = regexp.MustCompile(`_\d{6}$`)`.
    Suffix present → `ContractUSDTFutures` (suffix stripped);
    absent → `ContractPerpetual`. Non-USDT quotes rejected
    (binancef is the USDT-margined family by definition).
  - Reader migrations:
    `internal/actors/scopes/derive/source_scope_actor.go:routeTrade`
    and `internal/actors/scopes/ingest/publisher_actor.go` both
    now call `.VenueSymbol()` instead of reading `.Symbol`.
  - Test updates across the consumption surface
    (`trade_test.go`, both `aggtrade_test.go`, `sampler_test.go`,
    `trade_burst_sampler_test.go`, `volume_sampler_test.go`,
    `test_helpers_test.go`). New tests added: binancef
    delivery-vs-perpetual classification (2), binances/binancef
    non-USDT rejection (2), VenueSymbol behavior in the
    observation package (2).
- **Commit 7 (raccoon-cli analyzer + policy)** — P5 enforcement:
  - `tools/raccoon-cli/policies/adapters.toml` — declarative
    allowlist `["binances", "binancef"]`. Future venues require
    a policy edit before the analyzer accepts the adapter
    directory.
  - `tools/raccoon-cli/src/analyzers/check_instruments.rs` — 9
    unit tests. Three checks: adapter-allowlisted (directory
    appears in policy), adapter-uses-canonical-constructor
    (production code imports `internal/domain/instrument` AND
    calls `instrument.New(` or `instrument.FromSymbol(`).
    Struct-literal bypass is rejected by check 3; `*_test.go`
    files excluded from the production scan.
  - CLI wiring: new subcommand `raccoon-cli check instruments`
    (visible alias `check-instruments`). Quality-gate pipeline:
    new Step 9 (between `check-metrics` and `drift-detect`).
- **Commit 8 (closure — this commit)**:
  - [`docs/programs/PROGRAM-0004-multi-venue.md`](programs/PROGRAM-0004-multi-venue.md)
    — H-6.a Changelog entry; transitory-method pattern section
    added between escopo and sub-onda sequencing policy.
  - [`docs/TRUTH-MAP.md`](TRUTH-MAP.md) — Observation row updated
    (Instrument field + VenueSymbol method); ADR-0021 row moved
    from `Planned` to `Partially Implemented` with full anchors;
    new `check-instruments` analyzer row; summary counts updated
    (100 verify checks, 9 static analyzers, 4 PRDs, H-6.a
    closure Changelog entry).
  - [`docs/GLOSSARY.md`](GLOSSARY.md) — `Canonical instrument`
    entry updated to point to shipping code; new entries:
    `Canonical symbol`, `Venue symbol`, `Transitory adapter
    method`.
  - This RESUMPTION.md entry.

**Marco**: H-6.a abre PROGRAM-0004 (Multi-venue) com modelo
canônico de instrument + 2 adapters Binance migrados + analyzer
P5. Padrão de promoção difere de PROGRAM-0003 (que promoveu ADRs
na mesma onda em que os introduziu): **ADR-0021 permanece
`Proposed`** durante toda a migração; promoção é evento atômico
em H-6.f apenas se critérios #1–#5 (incluindo #4a + #4b após
split) estão literalmente satisfeitos. P7 absoluto.

**Mid-development discovery em H-6.a**: pré-flight de H-6
revelou cascade de 342 `.Symbol` references em 106 production
files em 31 packages — escopo incompatível com onda única
revisável. Após pause-and-report, user aceitou re-escopo em
sub-ondas H-6.a–H-6.f com sub-onda sequencing policy estrita.
Dual-write rejeitado como "débito tóxico permanente"; opção (C)
selecionada — transitory method com nome distinto (`VenueSymbol()`,
não `Symbol()`) elimina classe de bug latente
(`t.Symbol()` esperando canonical recebendo venue-native em 6
meses). Pattern documentado em PRD-0004 para reuso nas próximas
sub-ondas.

**Próxima sub-onda destravada após merge**: H-6.b — migration
de Evidence + Signal + Decision + Strategy + Risk domain types
para `Instrument CanonicalInstrument`. Sub-onda sequencing
policy: H-6.b abre branch APENAS após merge de H-6.a em `main`.

Entregas H-5 (sessão anterior):

- `docs/programs/PROGRAM-0003-observability.md` — PRD opening
  Fase Observability with single-onda scope (H-5). Includes
  pre-onda audit confirming 7/7 long-running binaries already
  expose `/metrics` (via HealthServer or via gateway routes),
  so entrega-4 "audit + gap-fill" becomes documentation, not
  code (commit 1).
- `docs/decisions/0024-metrics-policy.md` — ADR-0024 codifying
  naming convention (MP-1), label budget (MP-2, prohibits
  `instrument`/`symbol`/`request_id`/composite labels),
  histogram buckets (MP-3), per-subsystem cardinality budget
  (MP-4), **log compensation pattern** (MP-5 — when a
  high-cardinality dimension is diagnostically valuable but
  operationally expensive as a label, emit a structured log
  alongside the metric increment), and migration of existing
  `consumer_seq_gap_total` (MP-6) (commit 2).
- `docs/decisions/0025-alerting-strategy.md` — ADR-0025 codifying
  **SLO status taxonomy** (Proposed/Observing/Committed; AS-1),
  multi-window multi-burn-rate per Google SRE (AS-2), severity
  tiers with Observing SLOs CAPPED at `ticket` regardless of
  burn (AS-3), label conventions (AS-4), silence conventions
  (AS-5), runtime-safety alerts as distinct category (AS-6)
  (commit 3).
- `internal/shared/metrics/sequencer_metrics.go` —
  `marketfoundry_consumer_seq_gap_total` refactored from
  `{stream_key}` (composite encoding instrument) to
  `{venue, event_type}` per ADR-0024 MP-2. `IncSeqGap` signature
  changes; inline doc shows MP-5 log compensation pattern
  callers MUST follow (commit 4).
- `deploy/observability/prometheus/{prometheus,recording.rules,alerts.rules}.yml` —
  scrape config (7 binaries + self-scrape), 44 recording rules
  (4 SLO groups + runtime-aggregates), 13 alert rules (8 SLO
  burn-rate at ticket severity + 5 runtime-safety per ADR-0025
  AS-6) (commits 5/6/7).
- `deploy/observability/grafana/{provisioning,dashboards}/` —
  datasource (`uid: marketfoundry-prometheus`) + filesystem
  dashboard provisioning + 5 dashboards (ingest-health,
  derive-health, store-health, gateway-health,
  determinism-health) each with 5 panels (commit 8).
- `deploy/compose/docker-compose.yaml` — `observability` opt-in
  profile adds `prometheus` (image `prom/prometheus:v2.54.1`,
  :9090, 30d retention) and `grafana` (image
  `grafana/grafana:11.2.2`, :3000, admin/admin default).
  Persistent volumes `market-foundry-prometheus-data` /
  `market-foundry-grafana-data` (commit 5).
- `Makefile` — new `##@ Observability` section (`obs-up`,
  `obs-down`, `obs-reload`) + new `metrics-check` target under
  `##@ Determinism` (commits 5/9).
- `tools/raccoon-cli/policies/binaries.toml` — declarative
  allowlist with two categories: `one_shot = ["migrate"]` for
  CLI tools without HTTP; `transitive_registration = ["gateway"]`
  for binaries whose `/metrics` registration lives in an imported
  package. Tech debt documented inline: future refactor may
  replace this list with transitive import-closure scanning
  (commit 9).
- `tools/raccoon-cli/src/analyzers/check_metrics.rs` — new
  analyzer (~370 LoC, 10 unit tests). Reads policy file; scans
  `cmd/*/main.go` directories; flags long-running binaries
  missing `healthz.NewHealthServer` / `metrics.HandlerFunc` /
  `mux.Handle("GET /metrics", ...)` in their own package. CLI
  variant + dispatch + gate Step 8 integration (commit 9).
- `docs/operations/slo.md` — `Status: Active — all four SLOs in
  Observing`. New SLO status taxonomy section. Per-SLO `Status`
  field flipped from "Not yet measured" to "Observing" with
  details on whether the underlying counter is wired (F2 + F3)
  or canonical-name-reserved (F1 + F4). Targets summary table
  gains `Status` column. "How to promote Observing →
  Committed" section replaces the old "How to evolve" section
  (commit 10).
- `docs/operations/observability.md` — new operator guide.
  Quick-start make commands; architecture diagram; per-binary
  `/metrics` inventory; provisioned dashboards table; alert
  summary; common workflows; layout map; persistence; known
  limitations (cross-linking PROGRAM-0003 non-goals) (commit
  10).
- `docs/decisions/0024-metrics-policy.md`,
  `docs/decisions/0025-alerting-strategy.md` — `Status` flipped
  `Proposed` → `Accepted`; per-ADR Changelog entries; criterion-
  by-criterion mapping sections referencing the H-5 commits
  that delivered each criterion (commit 11).
- `docs/TRUTH-MAP.md` — 6 new rows under Foundation ADRs +
  Architectural invariants sections (metrics-policy +
  alerting-strategy + observability-stack + counter-refactor +
  check-metrics-analyzer). Summary counts updated: 25 ADRs (added
  0024 + 0025 both Accepted); 96 verify checks (+3 from
  `check metrics`); 3 PRDs (added PROGRAM-0003 Active) (commit
  11).
- `docs/GLOSSARY.md` — new `## Observability` section with 5
  terms: **SLI**, **SLO** (with status taxonomy reference),
  **Error budget**, **Burn-rate alert**, **Recording rule**
  (commit 11).

**Marco**: H-5 abre PROGRAM-0003 (Observability) com primeira
fase entregue. **Dois ADRs introduzidos e promovidos na mesma
onda** (ADR-0024 + ADR-0025) — pattern diferente de PROGRAM-0002
que herdou ADRs Proposed de PROGRAM-0001. SLOs F1–F4 saem do
estado "template — not yet measured" para `Observing` —
infraestrutura mensurando, baseline em coleta, promoção para
`Committed` é decisão de onda futura per ADR-0025 (7 dias de
compliance). Stack via opt-in profile (`make obs-up`); padrão
`make up` permanece lean.

**Mid-development discovery em H-5**: análise pós-commit-9
revelou que o detector spec original (`healthz.NewHealthServer
|| mux.Handle.*"/metrics" || metrics.HandlerFunc` no package
do main) não passava no gateway, que registra `/metrics`
transitivamente via `routes.DefaultRoutes(deps)`. User-confirmed
mitigation: estender `policies/binaries.toml` com
`transitive_registration` allowlist (declarativo); analyzer
trata listed binaries como compliant. Tech debt documentado:
futuro refactor via `go list -deps` ou AST closure scan
substituiria a lista. **Sem erratum em ADR-0024**; o ADR
References gained an analyzer-scope note pointing at the
known-debt path.

**Próxima onda destravada após merge**: H-6 — PROGRAM-0004
(Canonical instrument + venue normalization, ADR-0021/0022)
opens.

Entregas H-3.a (sessão anterior):

- **Commit 0 (erratum)**: [`docs/decisions/0017-event-envelope-and-versioning.md`](decisions/0017-event-envelope-and-versioning.md)
  e [`docs/decisions/0018-protobuf-contract-layer.md`](decisions/0018-protobuf-contract-layer.md)
  — seção "Promoção para Accepted" de ambas reescrita; Changelog
  adicionado. Sem mudança no status (ambas continuam `Proposed`).
- [`docs/programs/PROGRAM-0002-wire.md`](programs/PROGRAM-0002-wire.md)
  — PRD da Fase Wire (H-3.a / H-3.b / H-4), status `Active`.
- `proto/buf.yaml`, `proto/buf.gen.yaml`, `proto/registry.json`,
  `proto/envelope/v1/envelope.proto`,
  `proto/marketdata/v1/trade.proto` — skeleton proto. `buf lint`
  e `buf build` PASS sobre os dois schemas.
- `Makefile` — três targets novos: `make proto-lint`,
  `make proto-gen`, `make proto-breaking`. **NÃO** estão em
  `make verify` ainda (composição arriva em H-3.b com analyzer
  `check proto`).
- `scripts/proto-breaking.sh` — wrapper que trata "baseline empty"
  como PASS (relevante só na primeira introdução de `proto/`).
- `scripts/bootstrap-check.sh` — valida presença de `buf` + versão
  mínima 1.50.0 (foundry usa schema v2 que requer >= 1.32.0;
  pinned 1.50.0 conservador).
- `.tool-versions` — `buf 1.68.4` (validado localmente).
- `docs/DEVELOPMENT.md` — Prerequisites lista `buf` + nova seção
  "External tooling" com versões e referência a `protoc-gen-go`
  (validação CI deferida a H-3.b).
- `.gitignore` — nova seção G `internal/shared/contracts/**/*.pb.go`
  TEMPORARY (H-3.a only; removida em H-3.b).
- [`docs/TRUTH-MAP.md`](TRUTH-MAP.md) — rows de ADR-0017 e ADR-0018
  parcialmente populadas (anchor real para skeleton; H-3.b
  preenche generated + analyzer). Status: Planned (partial).
- [`docs/GLOSSARY.md`](GLOSSARY.md) — termos novos: Proto schema,
  buf, Schema registry, Schema status.

Entregas H-2 (sessão anterior):

- [`docs/decisions/0017-event-envelope-and-versioning.md`](decisions/0017-event-envelope-and-versioning.md)
  — envelope canônico de nove campos (type, version, venue,
  instrument, ts_exchange, ts_ingest, seq, idempotency_key,
  payload) para eventos no mesh JetStream. **Status: Proposed**;
  promovido por H-3.
- [`docs/decisions/0018-protobuf-contract-layer.md`](decisions/0018-protobuf-contract-layer.md)
  — proto como wire format primário do mesh; JSON fallback; buf
  tooling; boundary `internal/shared/contracts/`. **Status:
  Proposed**; promovido por H-3.
- [`docs/decisions/0019-deterministic-replay-time-invariants.md`](decisions/0019-deterministic-replay-time-invariants.md)
  — quatro invariantes determinísticas (INV-D1 pureza domínio,
  INV-D2 ordering por seq, INV-D3 replay byte-idêntico, INV-D4
  byte-stability em N=50 runs). **Status: Proposed**; promovido
  por H-4.
- [`docs/decisions/0020-sequencing-and-time-normalization.md`](decisions/0020-sequencing-and-time-normalization.md)
  — Sequencer com seq monotônico per stream_key
  `(venue, instrument, event_type)`; persistência em NATS KV
  `SEQUENCER_STATE_LATEST`; gap detection via counter
  `consumer_seq_gap_total`. **Status: Proposed**; promovido por
  H-4.
- [`docs/decisions/0021-canonical-instrument-and-venue-model.md`](decisions/0021-canonical-instrument-and-venue-model.md)
  — `internal/domain/instrument/` com Venue / BaseAsset /
  QuoteAsset / ContractType / CanonicalInstrument; adapter
  ToCanonical/FromCanonical. **Status: Proposed**; promovido por
  H-6.
- [`docs/decisions/0022-multi-venue-normalization-policy.md`](decisions/0022-multi-venue-normalization-policy.md)
  — política operacional R1–R4 (Capabilities, /venues/capabilities,
  silent-reject, raccoon-cli check venue-parity). **Status:
  Proposed**; promovido por H-7.
- [`docs/decisions/0023-storage-tier-roadmap.md`](decisions/0023-storage-tier-roadmap.md)
  — Stage 1 (ClickHouse + KV, atual) → Stage 2 (TimescaleDB,
  H-10) com triggers empíricos T1/T2/T3. **Status: Proposed**;
  pode permanecer indefinidamente se nenhum trigger disparar.
- [`docs/decisions/README.md`](decisions/README.md) — nova seção
  "Fase Harvest — Foundation ADRs (Proposed)" indexa as 7 ADRs.
- [`docs/programs/PROGRAM-0001-foundation.md`](programs/PROGRAM-0001-foundation.md)
  — política operativa de status clarificada; tabela de ADRs
  esperados expande para sete linhas com critérios de promoção;
  Changelog 2026-05-24 H-2 anexado.
- [`docs/TRUTH-MAP.md`](TRUTH-MAP.md) — nova seção "Planned
  capabilities — Foundation ADRs (Proposed)" com 7 rows;
  Summary count 16 → 23 ADRs.
- [`docs/AUTHORITY.md`](AUTHORITY.md) — nota T3 atualizada
  (zero → sete Proposed ADRs); file-to-tier inventory expandido;
  Changelog 2026-05-24 H-2 anexado.
- [`docs/GLOSSARY.md`](GLOSSARY.md) — termos novos: Canonical
  event envelope, Canonical instrument, Sequencer, Stream key,
  Wire format, Storage tier, Venue. Entrada existente
  `Envelope` reclassificada como "transport envelope" com
  pointer para o canônico.

Capacidades futuras (H-3+) — cliente Odin (H-12+, em `client/`),
TimescaleDB (provável H-10 se trigger ADR-0023 disparar),
insights/replay/multi-venue/proto layer/observability — são
escopadas no momento em que cada onda abre. Esta seção registra
apenas o estado atual do programa Foundation; o roadmap detalhado
vive em PROGRAM-0001.

`market-raccoon` (em `$RACCOON_REFERENCE_PATH`) permanece read-only
referência consultiva; nenhum arquivo é copiado, capacidades são
reescritas no foundry.

---

## Current functional state

The system runs end-to-end in paper (dry-run) mode against Binance
WebSocket data. Specifically:

- **All eight binaries build and start cleanly** (`make build`, `make up`).
- **Stack health passes** (`make ps` shows all services healthy).
- **Smoke test passes** (`make smoke` runs the canonical end-to-end
  proof against a real compose stack).
- **Gateway boot is verified at CI time** by
  `cmd/gateway/boot_test.go`, which hermetically registers all routes
  and would fail before container boot if a route trie conflict were
  reintroduced.
- **60 HTTP endpoints are catalogued and reachable** through the
  gateway (subject to conditional registration — see below).
- **ClickHouse persistence is operational**: events from the stream
  mesh land in the analytical tables via the `writer` binary, and
  read endpoints serve them back through the gateway.
- **Forward-only migrations are tracked** in `_migrations` and
  applied via `make migrate-up`.

What was verified concretely during Phase 0 closure (May 2026):

| Verification | Status |
|---|---|
| `make bootstrap` | PASS |
| `make verify` | PASS (since P1D.4 — G6 resolved, see "Recently resolved"). All 6 active quality-gate analyzers green; 84 checks, 0 errors. |
| `make build` | PASS for all services |
| `make up` → 9 services healthy | PASS |
| `make smoke` | PASS |
| Gateway boot test | PASS (introduced after P0.6) |
| Three route trie conflicts | FIXED (P0.6 removed lifecycle/list, renamed source-explain and session statics) |
| `cmd/gateway/boot_test.go` regression guard | IN PLACE |

---

## Known gaps in operational state

These are real gaps in the running system. They are not blockers for
development but operators should know they exist.

### G1 — `/execution-source-explain` is unreachable in any environment

The endpoint exists in code (`internal/interfaces/http/routes/source_explain.go`)
and registers conditionally on `deps.GetSourceExplanation != nil`. However,
**no code path in `cmd/gateway/` ever constructs a `GetSourceExplanation`
use case** — `NewGetSourceExplanationUseCase` (defined in
`internal/application/executionclient/get_source_explanation.go`) has no
caller in the gateway composition root. The dep is therefore always
`nil`, the route never registers, and the endpoint returns 404 in any
deployment, not just local default.

The handler also requires a `SourcePathConfigProvider` implementation;
no concrete implementation exists in the repository today.

**Source:** originally documented as gap WG-1 in the pre-reset
strategy-signal integration evidence matrix (retired in P2.Y; recoverable
via `git log`). The gap itself is still real.

**Fix:** in `cmd/gateway/` (likely `compose.go`), provide a
`SourcePathConfigProvider` implementation and call
`executionclient.NewGetSourceExplanationUseCase(gateway, configProvider)`,
then pass the result into the `SourceExplainFamilyDeps.GetSourceExplanation`
slot. Until then, expect 404.

### G2 — KV bucket coverage gaps for signals and strategies

Not every signal type and not every strategy type has a corresponding
`_LATEST` KV bucket. Verified against the codebase:

- **Signal:** 2 of 6 types have a bucket (`SIGNAL_RSI_LATEST`,
  `SIGNAL_EMA_CROSSOVER_LATEST`). The remaining 4 (bollinger, macd,
  vwap, atr) flow through `SIGNAL_EVENTS` and persist in ClickHouse
  but have no operational projection.
- **Strategy:** 2 of 3 types have a bucket
  (`STRATEGY_MEAN_REVERSION_ENTRY_LATEST`,
  `STRATEGY_TREND_FOLLOWING_ENTRY_LATEST`). The missing one is
  `squeeze_breakout_entry`.

What this means: events flow through the JetStream mesh and persist
in ClickHouse, but **operational read** via gateway returns nothing
for the uncovered types. Analytical reads (via writer + ClickHouse)
do work.

**Source:** discovered during P1A.4b runtime inventory.
**Status:** unclear whether this is intentional design (some signals
are analytical-only) or oversight. No documented decision either way.

### G4 — HTTP authentication

There is **no authentication** on any gateway endpoint. The default
local deployment binds gateway to `127.0.0.1` only, making loopback
isolation the primary access control. Live deployments are expected
to add a reverse proxy with auth in front.

**Status:** deliberate gap for the local single-operator phase.
Needs to be addressed before any non-loopback deployment.

### G5 — Conditional registration is universal

This is more a documentation gap than a system gap, but operators
need to know: **almost every endpoint is conditionally registered**
based on whether its backing dependency is wired in the gateway
composition root. If a dep is absent, the endpoint silently returns
404 with no indication it would exist when wired.

The conditional endpoints table in [`HTTP-API.md`](HTTP-API.md)
lists each dep gate.

**Status:** by design — allows gateway to start with partial
dependencies. But the silent 404 is operator-hostile and could be
improved (e.g., a `/debug/routes` endpoint listing actually-registered
routes). Future enhancement.

### G6 — `TestS460_SessionLifecycleTransitions` time-resolution flake

`internal/application/execution/s460_session_metadata_test.go:104`
asserts `Session.Duration() != 0` after `Session.Close()`.
`Duration() = ClosedAt.Sub(StartedAt)` returns zero when both
timestamps fall in the same nanosecond — the test sets
`StartedAt: time.Now()` and immediately calls
`Close(clock.SystemClock{}, ...)` which does `now := clk.Now()`.
Under batch test load (`make test-integration`), the two
`time.Now()` calls occasionally land on the same nanosecond and
the assertion trips. Isolated re-run (`go test -count=3 -run
TestS460_SessionLifecycleTransitions`) consistently PASS.

Observed during H-6.b'' pre-push validation (2026-05-26). Not a
regression of H-6.b''; the test file dates from commit `218a010`
(H-4, 2026-05-25) and has zero overlap with files modified by
H-6.b''.

**Fix candidates** (any of):

1. Inject a `FixedClock` with an explicit time delta into the
   test setup, instead of relying on `clock.SystemClock{}`.
2. Have `Session.Close()` enforce `ClosedAt >= StartedAt + 1ns`
   (or use a monotonic counter for ordering).
3. Change the assertion to `Duration() >= 0` if zero is
   semantically valid (probably not — duration of a closed
   session should be strictly positive).

**Deferred to:** H-6.f cleanup wave or a dedicated test-
hardening sub-wave. Workaround: re-run the failing test
isolated via `go test -count=3 -run TestS460_…` to confirm
flake before treating as a regression.

---

## Known surface debt

These are quirks that don't block usage but are visible debt that a
future cleanup wave should address.

### D1 — Hyphenated HTTP paths from P0.6

Three paths use hyphens for naming, an unusual choice forced by
httprouter trie limitations:

- `/session-list` (was `/session/list`)
- `/session-batch-audit` (was `/session/batch-audit`)
- `/execution-source-explain` (was `/execution/source-explain`)

These coexist with non-hyphenated wildcard paths like `/session/:id`
which couldn't move. The result is a mildly inconsistent URL surface.

**Resolution path:** a future API redesign wave. Not urgent — the
endpoints work fine; only aesthetic.

### D2 — `execute` config sprawl + `s449` namespace residue

Seven of twelve config files under `deploy/configs/` are variants of
`execute`:

- `execute.jsonc`
- `execute-mainnet-dry-run.jsonc`
- `execute-mainnet-live.jsonc`
- `execute-mainnet-live-s449.jsonc`  ← stage-tagged
- `execute-unified.jsonc`
- `execute-venue-live.jsonc`
- `execute.env.example`

At least one (`execute-mainnet-live-s449.jsonc`) carries a stage
reference in its filename. Since stage-based governance was retired
(decision Y of the reset), the `s449` namespace is dead weight.

**Resolution path:** a config consolidation pass. Either flatten
into one execute config with environment-variant overlays, or at
minimum rename to drop `s449`.

### D3 — configctl subject namespace ambiguity (singular vs plural)

The configctl family currently uses **both** singular
(`configctl.event.config.*`) and plural (`configctl.events.config.*`)
subject patterns in parallel. This is a transitional surface — one
was being migrated to the other, but the migration never completed.

**Resolution path:** pick one, audit all publishers and consumers,
deprecate the other. Coordinated change required across multiple
files in `internal/adapters/nats/natsconfigctl/`.

### D4 — Stage-tagged smoke targets in Makefile

The Makefile has ~23 smoke targets in total, of which ~14 are
stage-tagged (`smoke-compose-wiring` (S372), `smoke-failure-isolation`
(S374), `smoke-live-listening` (S378), `smoke-live-dry-run` (S380),
`smoke-segmented-compose` (S394), and similar). These were used
during the previous evolution model where each stage produced a
dedicated smoke. They still exist but no longer fit the operational
model.

**Resolution path:** prune in a cleanup wave. Most likely keep the
~9 functional smoke targets (smoke, smoke-multi, smoke-analytical,
smoke-round-trip, smoke-composed, smoke-live-stack, smoke-operational,
smoke-restart-recovery, smoke-help) and move the stage-tagged ones
out — either delete, or relocate to `scripts/historical/` for
archaeology.

---

## Recently resolved

### Phase 4.1 wave — CI restoration + quality gate cleanup

**Resolved** by 9 sub-prompts that took CI from red to fully green
on the quality-gate-ci job, clearing all 11 ci-profile warnings
surfaced after the Phase 4.1 SHA pinning migration lifted the
workflow-rejection layer that had masked latent failures since P3.3.

Sub-prompt summary:

- **P4.0** — documental hygiene sweep (DOC-1 through DOC-5) plus the
  P0-6 `SC2206` fix in `scripts/utils/lib.sh` that P3.5.safety had
  missed (scope was `scripts/*.sh`, not the `utils/` subtree).
- **P4.1** — CI workflow SHA pinning migration. 6 actions converted
  from tag refs (`@v4`, `@v5`) to commit SHAs. Branch protection
  rule `sha_pinning_required` (enabled in P3.3) became enforceable.
  Commit `4b5f14c`.
- **P4.1.1** — `golangci-lint-action` v6 → v9. The v9 binary takes
  `install-only` instead of the v6 `args` form; the v6 args were
  silently ignored on v9 (latent CI red). Commits `83e222e`,
  `899f4b5`.
- **P4.1.2** — Read-only investigation of `make quality-gate-ci`.
  Surfaced 11 pre-existing warnings now severity-promoted to errors
  by the `ci` profile (`tools/raccoon-cli/src/gate/mod.rs`). No
  fixes; categorisation only.
- **P4.1.3.a** — `drift-detect` `CANONICAL_STREAMS` aligned with
  the current `internal/adapters/nats/natsexecution/registry.go`
  set. G6.2: same pattern as the G6 fix at `557a508`, for streams
  added later. Commit `7ea24cd`.
- **P4.1.3.a'** — `contract-audit` alignment for the
  SessionLifecycle event: subject pattern widening, move from the
  ad-hoc `session_lifecycle_event.go` into the canonical
  `events.go`, addition of the `Metadata` field required by the
  domain event convention. Commit `41966a7`.
- **P4.1.3.b** — `_test.go` exemption added to the `deploy-boundary`
  check in `tools/raccoon-cli/src/analyzers/arch_guard.rs`. Tests
  asserting on canonical deploy paths is legitimate behaviour;
  extracting to constants would create indirection just to satisfy
  a scanner. Commit `6f9efd5`.
- **P4.1.3.c.i** — Read-only `cmd-boundary` mini-investigation.
  3 of 4 violations were TYPE-ONLY (composition wiring), 1 was
  MIXED (a single `execution.ComputeEffectiveMode` call from
  `cmd/execute/run.go` used for startup logging). Verdict: rule
  overshoots ADR-0005's "cmd sees everything" and is inconsistent
  with the application-client public contracts.
- **P4.1.3.c.ii** — `cmd-boundary` rule refined to flag domain
  function invocations only, permitting type/constant/struct-literal
  references. Implementation: text-pattern detection seeded by the
  codeintel `ProjectIndex` (functions known from the parsed AST).
  Go side adds `internal/application/executionclient/compute_effective_mode.go`
  wrapping the domain function; `cmd/execute/run.go` routes through
  the wrapper. Commit `25839ea`.
- **P4.1.5 / P4.1.6.a*** — NATS+JetStream infrastructure restoration
  for the Integration Tests job. Services-block startup was unreliable
  on the GitHub runner; switched to `docker run --network host` with
  the NATS monitor bound on port 8222 (`-m 8222`). Commits `d2238a0`,
  `5c8d0ff`.
- **P4.1.7** — Domain failure triage on the integration suite once
  NATS came up. Surfaced a P3 counter race: tests asserted on
  `tracker.Counter("filled")` immediately after the actor published
  the fill, but the counter was incremented after publish, leaving
  a sub-microsecond window for the read to miss the increment.
- **P4.1.8** — `eventuallyAtLeast` poll helper introduced and applied
  across 11 test sites that read execute-scope counters synchronously
  after a publish. Commit `81a2319`.
- **P4.1.8.a** — Suite timeout extension. The newly-polling tests
  pushed the suite above the 10-min default; bumped `-timeout 18m`
  in the Makefile target and the CI workflow timeout to 20 min.
  Commit `a5fff7c`.
- **P4.1.8.b** — Defensive completion: 5 additional counter-read
  sites identified during the scan-and-catch-up pass were converted
  to the helper. Commit `a378117`.
- **P4.1.8.c** — Read-only investigation of the counter-ordering
  question raised in the architect META review ("is the helper a
  band-aid for an actor-ordering bug?"). Findings: 11 non-test
  counter readers, all intra-actor self-reads (race-free by
  Hollywood single-threaded mailbox); only external surface is HTTP
  `/statusz`, whose multi-ms timing dominates the ~500µs race
  window; Prometheus uses a separate counter set. No current
  production consumer can observe the invariant violation. Owner
  decision: **Option (C)** — accept helper, defer actor reorder,
  document the trade-off.
- **P4.1.8.d** — P4.1.8 wave closure. Counter-ordering decision
  documented in `internal/actors/scopes/execute/venue_adapter_actor.go`;
  M7 ("dual-semantic counter for pre-publish vs post-publish
  observability") added to the design-meta queue; `-short` flag
  added to the Makefile `test-integration` target so the existing
  `testing.Short()` guards on 6 endurance/extended-observation
  tests become active in PR CI, dropping the suite from ~18m to
  ~1-2m. Long-running tests remain runnable locally without
  `-short`, or in a future nightly schedule.
- **P4.1.6.b** — Smoke Analytical E2E moved out of PR CI to a
  dedicated workflow (`.github/workflows/smoke-analytical.yml`)
  with `workflow_dispatch` (manual via `gh workflow run
  smoke-analytical.yml`) and `schedule: cron '0 6 * * *'` (daily
  06:00 UTC) triggers. Architectural rationale: PR CI is a
  fast-feedback loop; integration tests against external services
  (live Binance WSS) don't belong there. Job definition preserved
  verbatim (same steps, SHA pins, env vars, timeout); only the
  trigger surface changed. M8 (synthetic seeder pre-requisite for
  restoring smoke-analytical to PR CI) and M9 (log-error scan
  robustness — current warn-vs-error grep missed the silent failure
  mode) added to the design-meta queue.
- **P4.1.10** — Strategy dedup key precision fix. P4.1.9
  investigation (read-only) diagnosed three persistently-failing
  rapid-publish family tests (S380-DR-4, S373-MB-2/phase-2,
  E2E-2/phase-2) as a domain-layer bug: `Strategy.DeduplicationKey()`
  used `Timestamp.Unix()` (whole-second precision), so multiple
  publishes within a single wall-clock second produced identical
  `Nats-Msg-Id` values and were silently dropped by JetStream's
  2-minute Duplicate Window. Production was unaffected (kline cadence
  ≥1s never exercises this); tests tripped the bug because they
  publish siblings in tight loops. Fix: switch to `Timestamp.UnixNano()`.
  Also added `PubAck.Duplicate` warn-log surfacing in
  `internal/adapters/nats/natsstrategy/publisher.go` so future
  similar bugs are not silent (the operational blind spot P4.1.9
  noted as surpresa #2). Bug introduced in commit `fa8f04a5`
  ("initial quick start") and was latent through Phases 1–3 and
  most of Phase 4 — surfaced only after P4.1 lifted CI SHA-pinning
  rejection. Counter increment for `dedup_dropped` was intentionally
  omitted to keep blast radius to a single file (Publisher has no
  tracker field; wiring one would change the constructor signature
  across 15 callsites).
- **P4.1.11** — Time-capped abbreviated investigation (5:20 min
  finish vs 20-min cap) of a newly-visible writerpipeline failure
  that surfaced once P4.1.10 unmasked the prior layer. Found the
  same Subject-as-prefix mismatch pattern as P4.1.3.a'
  SessionLifecycle: 9 test sites across two files
  (`writerpipeline/restart_recovery_test.go` and
  `natsexecution/restart_recovery_test.go`) build `ConsumerSpec` by
  hand using the bare `registry.PaperOrderSubmitted` EventSpec
  (subject `execution.events.paper_order.submitted`, no wildcard).
  The consumer fallback at `natsexecution/consumer.go:79` then sets
  `FilterSubject` to that bare value, which does not match
  publishers' qualified subjects
  (`execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}`).
  Production paths use helper specs
  (`ExecuteStrategyMeanReversionEntryConsumer`,
  `ExecuteVenueMarketOrderIntakeConsumer`) that supply the `.>`
  wildcard form, which is why production and family tests were
  unaffected. Investigation report captured at
  `/tmp/p4.1.11-writerpipeline-investigation.md` (173 lines).
- **P4.1.11.a** — Bundled three-part fix that closes the Phase 4.1
  wave. Initial scope (subject-filter helper) discovered two more
  pre-existing layers during local repro; each was the **same bug
  class as an earlier wave fix**, not a genuinely new architectural
  concern, so they were folded into the same commit rather than
  spawning further sub-prompts:

  1. **Subject filter** — new
     `WriterPaperOrderExecutionConsumerForTest(durable string)`
     helper in `internal/adapters/nats/natsexecution/registry.go`
     mirroring the codegen-managed `WriterPaperOrderExecutionConsumer()`
     but accepting a caller-supplied durable. 9 spec construction
     sites updated across `writerpipeline/restart_recovery_test.go`
     (4) and `natsexecution/restart_recovery_test.go` (5);
     `natskit` import dropped from the writerpipeline test (no
     longer referenced). Same root-cause class as P4.1.3.a'
     SessionLifecycle subject mismatch.
  2. **Test-isolation reset** — new
     `ResetExecutionEventsStreamForTest(url string)` helper in the
     same registry file. Best-effort `js.DeleteStream` of
     `EXECUTION_EVENTS` at the top of each affected test so the
     shared NATS container (re-used across tests in the integration
     suite) does not replay one test's events into a later test's
     fresh durable. 9 reset calls inserted. The same
     `JSErrCodeStreamNotFound` swallow pattern as production
     `consumer.go` is used for the "first run, nothing to delete"
     case.
  3. **DeduplicationKey precision (completion of P4.1.10)** —
     P4.1.10 fixed `Strategy.DeduplicationKey()` (Unix → UnixNano)
     because the family tests it targeted only published
     strategies. The same `Timestamp.Unix()` precision bug existed
     in `ExecutionIntent`, `Decision`, `RiskAssessment`, and
     `Signal` (4 sibling timestamp-keyed types in
     `internal/domain/`). The restart_recovery tests publish
     `PaperOrderSubmittedEvent` which embeds `ExecutionIntent` —
     so the same silent JetStream Duplicate-Window drop reappeared
     for tests that publish siblings within a wall-clock second.
     All 4 sibling impls switched to `UnixNano()`; 4 unit-test
     format assertions updated (`execution_test.go`,
     `decision_test.go`, `risk_test.go`,
     `signal_test.go` — the last required adding `fmt` to its
     imports since the previous hardcoded literal was replaced
     with `fmt.Sprintf`). Production cadence (kline ≥1s) keeps
     this latent for all four types in prod; the latency surfaces
     only under tight-loop test publishes.

  Cumulative effect: Phase 4.1 wave fully closes; PR CI returns
  to 7/7 GREEN. M11 (subject-filter validation in `consumer.go:79`
  fallback) added to the design-meta queue as the architectural
  follow-up — the test-side helper prevents the manifestation but
  the fallback path remains a quiet footgun for any future test
  that bypasses production helpers. M12 (audit all timestamp-keyed
  `DeduplicationKey` impls in one pass when new domain types are
  added) is the systemic lesson — patching one type at a time
  cost three sub-prompts when the recipe was identical.

Quality-gate-ci error count across the wave:
**11 → 9 → 7 → 4 → 0**.

First fully-green `make quality-gate-ci` since P3.3 (`5830fc9`).

Process notes:

- The 11 errors were process debt (latent failures surfacing as the
  workflow-rejection layer cleared), not regressions. The same
  warnings had been present and unreported for many commits; only
  the `ci` profile severity promotion made them visible.
- Both formerly-red CI jobs are now resolved end-to-end:
  **Smoke Analytical E2E** moved off PR CI by P4.1.6.b (now on
  schedule/manual); **Integration Tests** restored to GREEN by the
  chain P4.1.5 → P4.1.6.a* → P4.1.7 → P4.1.8.* → P4.1.10 → P4.1.11.a.
  The wave revealed three layered, pre-existing failure classes
  (counter-ordering races, rapid-publish dedup precision, and
  subject-filter wildcard mismatch) that had been masked by earlier
  workflow-rejection layers. Each layer surfaced only when the layer
  above it cleared — see the per-P4.1.x entries above for the
  per-class root causes.

Institutional knowledge captured in `docs/CONTRIBUTING.md` →
"Audit and investigation patterns" (P4.1.4).

### CONTRIBUTING.md expansion + README refresh (Phase 3.9)

**Resolved** by codifying Phase 1+2+3 institutional knowledge in
`docs/CONTRIBUTING.md` and refining `README.md` for a public-visitor
audience. Closes P3.0 audit P1 findings "CONTRIBUTING missing AI
agent protocols (depth)" and "README gaps for public visitor".

`docs/CONTRIBUTING.md` expansion (existing "Specifically for AI
agents" section renamed to **"For AI agents (institutional
knowledge)"** and substantially extended):

- Preamble framing the section as "cumulative knowledge base — what
  we've learned the hard way" complementary to `CLAUDE.md`.
- Existing 4 subsections preserved (Read these documents first;
  Apply the protocols rigorously; Commit messages: explicit about
  provenance; When in doubt).
- New subsections added:
  - **Operating philosophy** (3 priority-ordered principles).
  - **Pause-and-report protocol (5 steps)** with a table of 5 worked
    examples from P2.3, P2.Y, P3.3, P3.5, P3.7.
  - **Common patterns** (working-tree verification, cross-ref search,
    inventory-first, atomic commits per concern; each cross-linked
    to its `.claude/commands/` slash command).
  - **Validation discipline** (project-vs-tool versions; audit-
    heuristic validation; format pre-commit checks).
  - **Cross-platform quirks** (shell quoting; `sed -i` macOS vs Linux).
  - **Lessons learned (Phase 1+2+3 errata)** — 5 specific mistakes
    documented to avoid repetition.
  - **Anti-patterns to avoid** (reframe-to-fit; aggregate concerns;
    trust narrative reference; skip validation; bypass safety hooks).

`README.md` refresh (conservative — no full rewrite):

- "Current state" section now leads with "Early-stage personal
  project. Active development by a single maintainer. Not
  production-ready; no API stability guarantees." plus an explicit
  "External contributions are not accepted at this stage" note with
  SECURITY.md pointer.
- "Contributing" section reframed for maintainers and AI agents
  with explicit pointers to `CLAUDE.md`, `docs/CONTRIBUTING.md`,
  and `.claude/`.
- "License" section refined with explicit permitted/not-permitted
  bullets (personal use vs commercial use).

`CLAUDE.md` unchanged — already robust post-P1C; `CONTRIBUTING.md`
expansion complements rather than duplicates.

### `.claude/` automation surfaces populated (Phase 3.8)

**Resolved** by populating `.claude/commands/` and `.claude/agents/`
with content codifying Phase 1+2 patterns. Closes P3.0 audit P1
finding "`.claude/` commands/agents/hooks empty".

Commands added (5 slash commands in `.claude/commands/`):

- **`/check-clean`** — pre-action verification (working tree clean +
  `make verify` / `make bootstrap` PASS). Used at session start.
- **`/check-refs <path>`** — cross-reference search across source,
  config, docs, Makefile, CI before deletion or rename. Prevents the
  stale-infrastructure-post-restructure pattern that surfaced
  repeatedly in Phase 1+2.
- **`/inventory <area>`** — structured inventory production (files,
  sizes, last-modified dates, subdirs). Used as foundation for
  fact-dense work in P1A, P2.X, P3.0.
- **`/audit <area>`** — read-only investigation skeleton with
  P0/P1/P2/P3 severity buckets and explicit "stop at recommendations"
  rule. Template for P3.0-style audits.
- **`/version-check`** — version consistency across `go.work`,
  `tools/raccoon-cli/rust-toolchain.toml`, `.tool-versions`, and CI.

Agent templates added (2 in `.claude/agents/`):

- **`investigation-agent`** — read-only investigator with structured
  output and severity categorization.
- **`execution-agent`** — scoped executor with explicit 5-step
  pause-and-report protocol (codifies lessons from P2.3, P2.Y, P3.3,
  P3.5.safety where pause-and-report caught factual divergence
  between premise and reality).

Hooks (`.claude/hooks/`) **not** added in P3.8: Claude Code hooks
feature remains exploratory; populated only when concrete repeated
needs surface. Possible follow-up as P3.8.1 or Phase 4.

Updated:
- `.claude/README.md`: added "Available commands" and "Available
  agent templates" sections; updated philosophy paragraph.
- `docs/CONTRIBUTING.md`: added "Claude Code automation" section
  between "Git hooks (lefthook)" and "Authorized expansion protocol".

`CLAUDE.md` (repo root) is unchanged — already robust post-P1C; the
new automation complements rather than replaces it.

### Editor configs and tool-versions added (Phase 3.7)

**Resolved** by adding three universal config files at the repo root.
Closes P3.0 audit P1 finding "editor/IDE configs absent".

- **`.editorconfig`**: cross-editor formatting standard. Go uses tabs
  (gofmt convention) and Makefiles use tabs (POSIX requirement); most
  other file types use 2-space indent with LF line endings, UTF-8,
  trailing-whitespace trim, and final newline. Markdown intentionally
  keeps trailing whitespace (line-break syntax). Editors with native
  or plugin EditorConfig support (VS Code, GoLand, vim, emacs, etc.)
  pick it up automatically.

- **`.gitattributes`**: git-level file handling. Forces LF line
  endings for tracked text files (cross-platform consistency); marks
  common binary extensions to prevent accidental diff/merge
  corruption; flags `go.sum` and `Cargo.lock` as
  `linguist-generated=true` so GitHub's language stats exclude them;
  marks `docs/**` and `*.md` as `linguist-documentation`. Pre-adoption
  CRLF audit confirmed zero tracked text files had CRLF endings — no
  re-checkout churn expected.

- **`.tool-versions`**: version manifest for [asdf](https://asdf-vm.com)
  and [mise](https://mise.jdx.dev). Currently pins:
  - `golang 1.25.7` (sourced from `go.work`)
  - `rust 1.90.0` (sourced from `tools/raccoon-cli/rust-toolchain.toml`)
  - `golangci-lint 2.12.2` (pinned in `.github/workflows/ci.yml` via
    `golangci-lint-action@v6` with explicit `version: v2.12.2`; the
    v2.x major series is also pinned in `.golangci.yml`'s
    `version: "2"`. Keep this manifest in sync with the CI pin.)

  Tools without asdf/mise plugins (`lefthook`, `shellcheck`) install
  separately via `brew` or `go install`.

**Not included (deferred)**:
- `.vscode/` — per-user IDE choice. Can be added in P3.7.1 if a VS
  Code workspace is desired.
- `.idea/` — same rationale.

### Shellcheck safety fixes + P3.0 audit retraction (Phase 3.5.safety)

**Resolved** by re-investigating P3.0's "scripts safety" finding via
`shellcheck` and applying targeted fixes for the real issues surfaced.
Closes P3.0 audit P0 finding "scripts safety" with corrected scope.

P3.0 audit had claimed **"39/39 scripts MISSING `set -e`"**. That
finding is **retracted**: re-investigation found all 41 scripts already
have `set -euo pipefail` (the audit's heuristic `head -10 | grep`
missed the directive which appears after the header comment block,
typically lines 7–49). Real safety state is broadly safe.

Shellcheck 0.11.0 across all 41 scripts surfaced 106 issues:
- **71 (67%) false positives**: SC2015 (`A && B || C` used for logging),
  SC1091 (dynamic `source` paths shellcheck can't statically resolve).
- **28 (26%) minor cleanups**: SC2034 (unused vars), SC2329 (dead
  functions), SC2155 (declare+assign), SC2064 (trap quoting), SC2012/
  SC2010 (`ls` vs `find`), SC2153, SC2001. Cosmetic — not safety risks.
  Deferred to optional P3.5.cleanup.
- **7 (7%) real safety issues**: 5 × SC2086 (word splitting on
  unquoted variables) + 2 × SC2206 (array assignment via word
  splitting). **Fixed in this phase**:
  - `scripts/diag-check.sh:183` — `exit "$ERRORS"`
  - `scripts/live-pipeline-activate.sh:116` — `sleep "$POLL_INTERVAL"`
  - `scripts/live-pipeline-activate.sh:402` — `exit "$ERRORS"`
  - `scripts/smoke-compose-wiring.sh:492` — `exit "$ERRORS"`
  - `scripts/smoke-first-slice.sh:98` — `sleep "$POLL_INTERVAL"`
  - `scripts/smoke-multi-symbol.sh:77–78` — `read -ra` instead of
    `ARRAY=($VAR)` for `SYMBOLS` and `TIMEFRAMES`.

Total post-fix shellcheck issue count: 99 (= 106 − 7), all warnings
or notes, zero errors.

P3.6 (scripts safety — group 2) is **retired** as no-op: it was based
on the same incorrect "missing set -e" premise.

Lesson institutionalized: audit heuristics like `head -N | grep` can
miss content beyond the first N lines. For findings about widely-
adopted conventions (`set -e`, `gofmt`, etc.), validate with a
dedicated tool (`shellcheck`, `gofmt -l`, `cargo clippy`, etc.) before
planning remediation. Pause-and-report on audit divergence caught this
before any unnecessary work shipped.

### lefthook adopted for pre-commit and commit-msg validation (Phase 3.4)

**Resolved** by introducing [lefthook](https://lefthook.dev/) as the
pre-commit framework. Closes P3.0 audit P1 finding "no pre-commit
framework" plus the related "no commitlint" finding without a Node.js
dependency.

Stages configured in the new `lefthook.yml`:

- **pre-commit**: `gofmt` check on staged `.go` files, trailing
  whitespace, and YAML/JSON/TOML validity. Fast (sub-2-second typical).
- **commit-msg**: conventional commit format
  (`type(scope?): description`) via the new
  `scripts/validate-commit-msg.sh`, which accepts `feat`, `fix`,
  `chore`, `docs`, `ci`, `refactor`, `test`, `style`, `perf`, `build`,
  `revert`. Tested against the last 10 commits — all pass.
- **pre-push**: `make lint-go` and `make verify` available but
  `skip: true` by default. Opt in by removing the skip lines when
  ready for stricter local push gating.

Activation is per-developer (hooks are NOT auto-installed by the
commit): `brew install lefthook` (macOS) or `go install
github.com/evilmartians/lefthook@latest`, then `make install-hooks`.
Bypass for emergencies via `LEFTHOOK=0 git commit ...` or
`git commit --no-verify`.

`docs/CONTRIBUTING.md` gained a "Git hooks (lefthook)" section
between "PR workflow" and "Authorized expansion protocol".
`scripts/README.md` table updated with the new validator.

### GitHub settings lockdown applied (Phase 3.3)

**Resolved** by applying remote settings via `gh CLI`. Closes P3.0
audit P0 findings #3, #4, #6. Finding #2 (fork lockdown) partially
deferred — see below.

Changes via `gh api`:

- **Branch protection on `main`**: required status checks (Unit Tests,
  Repository Consistency & Quality Gate, Go Lint (golangci-lint)),
  strict (branch up to date), linear history required, no force-push,
  no deletions. PR review NOT required (solo-dev workflow).
- **Security & Analysis**: `secret_scanning`,
  `secret_scanning_push_protection`, `dependabot_security_updates`,
  `private_vulnerability_reporting` all enabled.
- **Actions**: `sha_pinning_required: true` (allowed_actions kept at
  "all"). May surface tag-pinned actions in the next CI run; P3.3.1
  will migrate the workflow to SHA pins if so.

**Finding #2 (fork lockdown) — deferred**: GitHub rejects
`allow_forking: false` on personal-owned public repositories (HTTP
422 — "Allow forks setting can only be changed on org-owned private
repositories"). The repo still publishes `allow_forking: true`, but
`pull_request_creation_policy: collaborators_only` already blocks
external PRs, which was the underlying intent. Manual fallbacks
(transfer to a GitHub org, or accept the fork-able state) documented
in `docs/operations/github-settings.md`.

Canonical reference of all remote settings now lives in
`docs/operations/github-settings.md` (remote settings have no git
history; this file is the source of truth going forward).

### `.gitignore` hardened (Phase 3.2)

**Resolved** by expanding `.gitignore` from 17 lines (minimal) to ~180
lines organized in six categories with explanatory comments. Closes
P3.0 audit P0 finding #5 (.gitignore missing critical patterns for
a public repository).

The new file groups patterns by intent:

- **A. Secrets and credentials** (P0 for public repo): `*.env`,
  `.env.local`, `.env.*.local`, `.env.production`, etc.; `*.key`,
  `*.pem`, `*.p12`, `*.pfx`, `*.crt`; `credentials`, `credentials.json`,
  `credentials.yml`; SSH keys (`id_rsa`, `id_ed25519`, ...); cloud
  configs (`.aws/`, `.gcp/`, `.azure/`); generic stores (`.secrets/`,
  `secrets/`, `*.token`, `*.secret`); `.netrc`, `.npmrc`.
- **B. Build artifacts**: `bin/`, `build/`, `dist/`, `out/`, coverage
  outputs, tmp/, archives. Preserves project-specific patterns
  `trace-pack-*` and `references/`.
- **C. Editor/OS metadata**: vim swap, backup files, `.DS_Store`,
  `Thumbs.db`. `.vscode/` and `.idea/` intentionally NOT excluded
  (per-developer choice).
- **D. Runtime**: `*.log`, core dumps, `*.test`, `*.prof`.
- **E. Tool-specific**: Rust `target/`, Node `node_modules/`, Python
  caches.
- **F. Compiled service binaries at repo root**: the original
  `/configctl`, `/derive`, `/execute`, `/gateway`, `/ingest`, `/store`,
  `/writer`, `/migrate` guards preserved verbatim — `go build ./cmd/<x>`
  drops the binary in the repo root by default.

Audit before modification confirmed zero existing tracked files match
new secret patterns, and the tracked file count (979) is preserved.
The previous `*.env` pattern was retained so `deploy/envs/local.env`
remains ignored.

### LICENSE adopted + SECURITY.md added (Phase 3.1)

**Resolved** by creating `LICENSE` and `SECURITY.md` in the repository
root. Closes P0 finding #1 from the P3.0 environment audit (LICENSE
absent) and finding #11 (no SECURITY.md).

The license is **PolyForm Noncommercial 1.0.0** — designed for solo
developers wanting to forbid commercial use while keeping source
visible. Permits personal use, research, education, hobby projects, and
evaluation. Compatible with the Go module proxy, no impact on
dependency tooling. Reference:
<https://polyformproject.org/licenses/noncommercial/1.0.0/>.

`SECURITY.md` documents how to report vulnerabilities to a personal
project: out-of-band via the maintainer email, no SLA, no bounty, scope
limited to this repository's own code.

`README.md` gained a final "License" section linking both files.
Source files were intentionally **not** annotated with per-file headers
— the `LICENSE` file alone is legally sufficient and a 400+ file diff
was not justified by the cosmetic gain. May be revisited later.

### `docs/legacy/` removed definitively (Phase 2.Y)

**Resolved** by `git rm -rf docs/legacy/` and updating active
cross-references. The 1712 files preserved under the original
"C+Y+Q — preserve legacy in-repo" decision were deleted; owner chose
no tag and no archive branch, trusting `git log` for recovery.

Cumulative consultation rate of legacy material during Phases 1A
through 2.X.1 was zero, demonstrating documental sufficiency of the
new topology. Removing the tree also takes ~17 MB off git operations,
IDE indexing, and GitHub web UI.

Cross-references corrected in the same commit:

- `scripts/bootstrap-check.sh` — `required_paths` array realigned from
  15 legacy entries to the current Phase 1A topology (root docs + the
  three subdir READMEs). The "Next Steps" tail message also updated.
  This was the **10th instance** of the stale-validation-infrastructure
  pattern observed since the reset (`.opencode/`, the original 500-line
  `repository-consistency-check.sh`, `AGENTS.md`, root `DEVELOPMENT.md`,
  root `README.md`, CI workflow blast-radius visibility,
  `raccoon-cli drift_detect.rs` const tables, `scripts/stage-tooling.sh`,
  the 4 orphan P2.X.1 smokes, and now `bootstrap-check.sh`).
- `scripts/repository-consistency-check.sh` — narrative comment.
- `tools/raccoon-cli/src/analyzers/drift_detect.rs` — 2 rustdoc comments.
- `deploy/configs/execute-mainnet-live.jsonc` — removed dangling
  `// See: docs/legacy/...` pointer (authorized scope expansion).
- `docs/RESUMPTION.md`, `docs/DEVELOPMENT.md`, `CLAUDE.md`, `AGENTS.md`,
  `README.md` — narrative refs and reading-map rows.

For any future need to inspect pre-reset material, use
`git log -- docs/legacy/<path>` or `git show <SHA>:docs/legacy/<path>`
against the parent of the P2.Y commit.

### G6 — `raccoon-cli drift-detect` against old topology (Phase 1D.4)

**Resolved** by rewriting 6 const tables in
`tools/raccoon-cli/src/analyzers/drift_detect.rs` to align with the
Phase 1A topology:

- `SIGNAL_DOCS`, `DECISION_DOCS`, `STRATEGY_DOCS`, `RISK_DOCS`,
  `EXECUTION_DOCS` collapsed from 7–30 paths each (pre-reset granular
  family architecture design docs, retired in P1A.1) to
  1 path each (`docs/domain/<x>.md`).
- `ARCH_DOCS` rewritten from 27 pre-reset arch docs to 8 canonical
  root docs (`docs/ARCHITECTURE.md`, `docs/RUNTIME.md`,
  `docs/HTTP-API.md`, `docs/DEVELOPMENT.md`, `docs/RESUMPTION.md`,
  `docs/CONTRIBUTING.md`, `docs/GLOSSARY.md`, `docs/decisions/README.md`).
- The "runtime-target.md mentions all services" sub-check rewired to
  read `docs/RUNTIME.md` (was previously silently skipping because the
  hardcoded path didn't exist).

The 27 other checks in `drift_detect.rs` (per-domain adapter alignment,
domain Go files, NATS subjects/durables/buckets, contracts,
naming-identity guard against `DEFUNCT_NAMES = ["emulator", "validator"]`,
actor-scope, stream-registry, premature-domain-entry, etc.) preserved
unchanged. They were already passing; this change only touched the
6 constants and one sub-check path.

**Effect:** `make quality-gate` PASS (6/6 active analyzers, 84 checks,
0 errors, was 61 errors). `make verify` PASS **for the first time since
P1A.1** (18+ prompts ago). CI workflow `repository-checks` job will run
green on the next push.

**Pattern note:** this was the 7th instance of the
"stale-infrastructure-post-restructure" pattern observed across Phases
1A–1D (`.opencode/`, `scripts/repository-consistency-check.sh`,
`AGENTS.md`, root `DEVELOPMENT.md`, root `README.md`, the CI workflow's
silent G6 propagation, and finally `drift_detect.rs` itself). The
discipline now lives in `docs/CONTRIBUTING.md` "Rules for documentation
changes" and the `make` verification surface, with the analyzer itself
enforcing the new topology going forward.

## Earlier resolutions

### G3 — `make verify` cross-references (originally framed as 9 failures from `.opencode/`)

The original framing of G3 ("9 failures, all from `.opencode/`
cross-refs") was inaccurate. P1B uncovered the truth in three layers:

1. **`.opencode/` directory** existed and had 1 cross-reference check
   failing. **Resolved** by deletion in P1B.

2. **`scripts/repository-consistency-check.sh`** had ~7 checks failing
   because the script was hardcoded against the pre-reset docs topology
   (`docs/product/`, `docs/architecture/`, `docs/development/`,
   `docs/stages/`, `docs/archive/`, `docs/tooling/`) which was
   restructured in P1A.1. The script was never updated during Phase 1A
   because the failure was misattributed to `.opencode/`.
   **Resolved** in P1B by replacement with a minimal stub aligned with
   the current Phase 1A topology (`scripts/repository-consistency-check.sh`,
   ~100 lines).

3. **`tools/raccoon-cli/src/analyzers/drift_detect.rs`** is a separate
   failing layer (61 errors) that was invisible in the original
   framing. **Escalated as G6**, not resolved in P1B (out of scope —
   `tools/` was off-limits).

**Net effect:** P1B resolved two of the three underlying layers.
`make verify` is still red because of G6. The "9 failures from
`.opencode/`" narrative was triply wrong (count of root causes,
attribution of the root, and missing an entirely separate failing
layer) and is corrected here so future readers learn from the error
rather than inherit it.

### D5 — `.opencode/` directory still present

**Resolved** by P1B. The directory was the navigation layer for an
external agent tool (OpenCode CLI). It has been deleted in its
entirety (37 files). The agentic layer will be rebuilt from scratch
in P1C using the Anthropic ecosystem (CLAUDE.md root + `.claude/`).

---

## Deliberate non-features

This section is as important as the gaps section. Each item below
is **intentionally not implemented**. Adding any of them requires a
deliberate design decision (an ADR), not an opportunistic PR.

### N1 — No backtesting harness

There is no mechanism to replay historical ClickHouse data through
the pipeline deterministically. Strategies must currently be tested
in paper mode against live WebSocket data.

This is the most-likely **next major feature**. The infrastructure
exists (PaperVenueAdapter, ClickHouse history, deterministic event
deduplication), but the runner that pulls history and replays it is
absent.

### N2 — No PnL aggregation per strategy

The `effectiveness` domain classifies individual round-trips into
win/loss/breakeven/unresolved. There is no aggregator that produces
"strategy X earned Y net over period Z, with max drawdown W".
Without this, you cannot quantitatively rank strategies or decide
when to retire one.

### N3 — No portfolio-level position sizing

Decisions are local per symbol. The `risk` domain checks
position-exposure and drawdown limits per assessment, but there is
no central model managing aggregate exposure across the portfolio.

### N4 — No multi-exchange surface

A single venue family (Binance, with Spot and Futures sub-segments).
Adding OKX, Bybit, or any other exchange would require a new adapter
under `internal/adapters/exchanges/` and corresponding execution
adapters. This is not currently scoped.

### N5 — No market-making primitives

No order book depth tracking, no queue position estimation, no
inventory risk model. The system is currently designed for momentum
and mean-reversion strategies, not market making.

### N6 — No machine learning pipeline

Signals are deterministic indicators (RSI, EMA, MACD, Bollinger,
ATR, VWAP). There is no training loop, no model registry, no
inference service.

### N7 — No HTTP authentication

Already mentioned in G4. Restated here for completeness — this is a
deliberate gap for the local single-operator phase, not a missing
feature in the usual sense.

---

## Where we are in the resumption cycle

The resumption from a 2-month pause is being executed in phases.
Each phase has a clear exit criterion.

| Phase | Goal | Status |
|---|---|---|
| **Phase 0** | Unblock — fix git limbo, align Go version, get smoke passing | **CLOSED** (commits up to 8900694, mid-May 2026) |
| **Phase 1A** | Documentation reset — move legacy, write new docs | **CLOSED** (17 sub-prompts, 36 docs, May 2026) |
| **Phase 1B** | Exterminate `.opencode/` | **CLOSED** (G6 escalated; see Recently resolved) |
| **Phase 1C** | Build `.claude/` agentic layer | **CLOSED** (CLAUDE.md + .claude/ structure built) |
| **Phase 1D** | PR-based governance + G6 resolution | **CLOSED** (root files consolidated, .github/ templates, drift_detect.rs realigned) |
| **Phase 2** | Environment hardening (CI, Docker, scripts, Makefile cleanup) | **CLOSED** (11 sub-prompts; golangci-lint baseline, Dependabot, CI hardening, Docker contexts, Rust toolchain pinning) |
| **Phase 3** | Public-repo hygiene (license, security, hooks, editor configs, AI agent automation) | **CLOSED** (2026-05-22; 10 sub-prompts executed, 2 deferred. See "Phase 3 — closed summary" below.) |
| Phase 4 | CI restoration + P0 follow-through deferred from Phase 3 | **CLOSED** (2026-05-23; P0 backlog 5/5 closed across P4.0–P4.5.c.ii; 12 ADRs, 20 M-candidates queued, 0 open Dependabot PRs, 0 open security advisories) |
| Phase 5 | Environment work — `.claude/`, prompt templates, operational tooling, process-debt mitigation (distinct from feature work) | **IN PROGRESS** (P5.0 audit complete 2026-05-23; P5.1+ pending owner direction) |
| Phase 6+ | Subsequent waves (feature work; first capabilities likely include backtesting) | Future |

Phase 1A subdivision (status at time of this doc):

| Sub-phase | Goal | Status |
|---|---|---|
| P1A.1 | Restructure docs/ topology + new scaffolding (legacy tree later retired in P2.Y) | Done |
| P1A.2 | docs/README, docs/GLOSSARY | Done |
| P1A.3 | docs/ARCHITECTURE.md | Done |
| P1A.4a | Runtime inventory (read-only, /tmp) | Done |
| P1A.4b | docs/RUNTIME.md | Done |
| P1A.4b.1 | Errata correcting ARCHITECTURE.md and GLOSSARY.md | Done |
| P1A.4c | docs/HTTP-API.md | Done |
| P1A.5a | docs/DEVELOPMENT.md | Done |
| P1A.5b | docs/RESUMPTION.md (this document) | Done |
| P1A.6 | Domain docs under docs/domain/ | Done |
| P1A.7 | Operations docs under docs/operations/ | Done |
| P1A.8 | Initial ADRs under docs/decisions/ | Done |
| P1A.9 | docs/CONTRIBUTING.md | Done |

### Phase 3 — closed summary

Goal: engineering excellence for solo dev + Claude Code in a public
repo.

| Sub-phase | Status | Outcome |
|---|---|---|
| P3.0 | ✓ | Environment audit (1345 lines, 13 sections) |
| P3.1 | ✓ | LICENSE (PolyForm NC 1.0.0) + SECURITY.md |
| P3.2 | ✓ | .gitignore hardened (17 → 184 lines, 6 categories) |
| P3.3 | ✓ partial | Branch protection + security toggles; fork lockdown blocked by GitHub personal-tier policy (mitigated by `collaborators_only` PR policy + LICENSE) |
| P3.4 | ✓ | lefthook + custom shell `commit-msg` validator |
| P3.5.safety | ✓ | 7 shellcheck SC2086/SC2206 fixes; P3.0 "missing set -e" finding retracted (all 41 scripts already had `set -euo pipefail`) |
| P3.7 | ✓ | `.editorconfig`, `.gitattributes`, `.tool-versions` |
| P3.8 | ✓ | `.claude/` commands (5) + agent templates (2) |
| P3.9 | ✓ | `docs/CONTRIBUTING.md` expansion ("For AI agents — institutional knowledge"); README refresh |
| P3.10 | ✓ | Closing audit + this RESUMPTION refresh |
| P3.5.cleanup | ⏿ | Deferred (cosmetic — 28 minor shellcheck issues + 32 SC1091) |
| P3.6 | ⏿ | Retired (audit premise was wrong) |

Key lessons institutionalized (in `docs/CONTRIBUTING.md` under
"For AI agents — institutional knowledge"):

- 5-step pause-and-report protocol with 5 worked examples.
- Project-declared vs tool-environment version distinction.
- Audit-heuristic validation (heuristics like `head -N | grep` miss
  content beyond the inspection window — validate with dedicated
  tools).
- Cross-platform shell quirks (quoting, `sed -i` macOS vs Linux).
- Atomic commits per concern.

Surprises caught during Phase 3 via pause-and-report:

- **P3.3**: GitHub personal-tier doesn't allow fork disable; mitigated
  by collaborator-only PR policy + LICENSE.
- **P3.5**: P3.0 finding "39/39 scripts missing `set -e`" was wrong
  — audit grepped only `head -10`; all 41 scripts already had
  `set -euo pipefail` declared after the header comment block.
  Retracted in P3.5.safety; replaced by shellcheck-based audit that
  surfaced 7 real safety issues (SC2086/SC2206), all fixed.
- **P3.7**: original claim that golangci-lint was "not pinned in CI"
  was wrong — `.github/workflows/ci.yml:179-182` explicitly pins
  `version: v2.12.2` on `golangci-lint-action@v6`, matching
  `.tool-versions`. Drift is zero. Corrected in P4.0 (see DOC-3
  erratum below); ongoing task is **monitoring drift** when
  Dependabot bumps the action wrapper (e.g., `@v6 → @v9`) without
  necessarily bumping the underlying lint binary.
- **P3.5.safety scope omission** (caught by P4.0 pre-audit, DOC-4):
  shellcheck audit covered `scripts/*.sh` but not `scripts/utils/*.sh`.
  One real SC2206 in `scripts/utils/lib.sh` was missed at the time
  and fixed in P4.0 alongside the documental sweep. Methodology drift,
  not a new pattern — the same rule should have been applied to all
  `.sh` files, not the top-level only.

---

## Phase 4 outlook

Phase 4 essential delivery complete (2026-05-23). The 4.1 wave (CI
restoration + quality gate cleanup) closed on 2026-05-22 with
quality-gate-ci green (commit `25839ea`); P4.2–P4.5 closed all five
P0 items deferred from Phase 3, with read-only investigation
interleaved before each fix.

**Phase 4 P0 backlog FULLY CLOSED** (5/5):

| P0 item | Phase 4 prompt | Closure commit / date |
|---|---|---|
| P0-1 (CI restoration) | P4.0 + P4.1 wave (24+ sub-prompts) | b7eaa53, 2026-05-22 |
| P0-2 (rate_limiter + Close) | P4.2 / P4.2.a | a6f0175, 2026-05-23 |
| P0-3 (context bounding) | P4.3 / P4.3.a | 455f02e, 2026-05-23 |
| P0-4 (Dependabot triage) | P4.5 / P4.5.a-c.ii | this commit, 2026-05-23 |
| P0-5 (ControlGate fail-open) | P4.4 / P4.4.a | 7c2f09e, 2026-05-23 |

**Cumulative artifacts shipped during Phase 4**:

- 12 ADRs (added ADR-0012 for ControlGate fail-open posture).
- 20 design-meta candidates (M1–M20, with M10 reserved gap;
  M19 closed during P4.5.c verification).
- ~9 errata observations across CONTRIBUTING/investigation patterns.
- 7/7 CI consistently green on main.
- 0 open Dependabot PRs.
- 0 open security advisories.

**Remaining Phase 4 work** (all discretionary):

- **P4.X** — Tier E quality enhancements, opt-in (e.g., the ~60
  hardcoded timeout literals deferred from P4.3.a — operational
  tunability gap, not a bug).
- **Phase 4 design-meta discussion** — full conversation across M1–M20
  when momentum permits. Not blocking.

The existing "Outstanding work" section below records each item's
closure narrative for handoff context; preserved verbatim as
historical record.

### Outstanding work (post P4.1)

1. ✓ **Integration Tests + Smoke Analytical E2E** (P4.1.5 → P4.1.6
   scope). Closed across the Phase 4.1 wave. Smoke Analytical E2E
   deferred to a scheduled/manual workflow (P4.1.6.b, commit
   `e91b863`); Integration Tests stabilized via the NATS docker-run
   switch (P4.1.6.a..a.ii) and counter-race helpers (P4.1.8.a..d).
   The documented `TestControlledActivation_FullLifecycle` /
   `TestRealVenueActivation_FullLifecycle` 200 ms timing flake
   remains visible on some intermediate Dependabot merges (per the
   P4.5.a/b/c.ii closure narrative); non-required and non-blocking
   per branch protection. CI 7/7 GREEN at `main` HEAD.
2. ✓ **`rate_limiter` test + `Close` lifecycle** (P0-2 / P4.2).
   Closed 2026-05-23. 10 unit tests added (`rate_limiter_test.go`);
   `Close()` lifecycle wired at the 2 cmd/execute mainnet sites via
   a `closers []func()` field on `venueAdapterResult`. P4.2.a fixed
   a downstream goroutine-assertion flake. CI 7/7 green.
3. ✓ **`context.Background()` propagation in actors** (P0-3 / P4.3).
   Closed 2026-05-23. Reframed: Hollywood deliberately drops context
   at the mailbox boundary, so the right shape was "bound fresh
   Background with WithTimeout + config", not "propagate caller ctx".
   P4.3.a bounded 14 unbounded sites + enabled the `contextcheck`
   linter for prevention. Surfaced M13/M14/M15 (see design-meta).
4. ✓ **Kill switch fail-open decision** (P0-5 / P4.4 + P4.4.a).
   Closed 2026-05-23. Investigation reframed P0-5 as documentation +
   observability gap, not semantic gap — the audit's "kill switch" is
   the codebase's ControlGate, with fail-open intentionally chosen
   and protected by 8-layer defense-in-depth. P4.4.a formalized the
   posture as ADR-0012 and added `gate_read_failures_total`
   counter with 5 reason labels so the silent failure mode is
   monitorable. No semantic change. Future hybrid strategies
   deferred as M16/M17/M18 pending counter data.
5. ◐ **Dependabot security PRs** (P0-4 / P4.5). Triage closed 2026-05-23;
   security wave closed same day:
   - P4.5 investigation: 17 open PRs identified, all 1 day old. Six
     open security advisories cluster cleanly to 3 PRs (#16/#17/#18).
     All 17 PRs share one root cause — bases predate the P4.1
     SHA-pinning migration. Triage shape is 3 archetype waves, not
     17 individual reviews.
   - P4.5.a (closed 2026-05-23): closed obsolete PR #5
     (golangci-lint-action — already applied via P4.1.1); rebased +
     merged security PRs #16 (otel /clickhouse), #18 (otel /migrate),
     #17 (rustls-webpki /raccoon-cli). All 6 security advisories
     closed. Required CI checks (Unit Tests, Quality Gate, Go Lint)
     green for all three; Integration Tests flake (the documented
     `TestControlledActivation_FullLifecycle` / `TestRealVenueActivation_FullLifecycle`
     timing flake) ignored as non-required, non-regression.
   - P4.5.b (closed 2026-05-23): minor/patch batch — 8 PRs (#7, #9,
     #11, #10, #13, #15, #12, #14) rebased + merged sequentially.
     Order grouped 3 cargo singletons → 3 standalone gomod →
     in-module pair (#12/#14 share `internal/adapters/nats/go.mod`).
     All 8 cleared required CI (Unit Tests + Quality Gate + Go Lint);
     Integration Tests flake non-blocking per P4.5.a posture. No
     genuine test failures; no mirror-pair conflicts (Dependabot
     rebase-on-trigger handles each PR against current main).
     `go.work.sum` picked up transitive checksums for the
     `golang.org/x/{net,sync,term,text,tools,mod}` and otel/metric
     families pulled in by the nats.go/clickhouse-go bumps.
   - P4.5.c (closed 2026-05-23): 5 majors — 4 GitHub Actions (#6,
     #2, #3, #4) + ureq 2→3 (#8). Two phases:
       * **Phase 1** (verification + investigation, ~10 min):
         rebased PR #6 to test M19 hypothesis. Result: post-rebase
         diff was SHA-style with version comment
         (`actions/checkout@de0fac2e... # v6.0.2`); 8 sites in ci.yml +
         1 in smoke-analytical.yml all updated. M19 verified
         **self-correcting**; closed. ureq surface inventory: 1
         file (`tools/raccoon-cli/src/smoke/api.rs`), 6 call sites,
         3 patterns, ~25 LOC. Recommendation: migrate.
       * **Phase 2** (execution): merged #6 as validation; sequential
         rebase + merge for #2 (actions/cache 4→5), #3 (actions/setup-go
         5→6), #4 (actions/upload-artifact 4→7) — all four landed
         SHA-pinned with version comments. ureq 2→3 migrated in api.rs
         (header/StatusCode/Agent.config_builder().timeout_global/body_mut
         .read_json) preserving ApiClient public interface. PR #8 closed
         in favor of combined Cargo.toml + source commit.
   - **Phase 4.5 wave fully closed.** Final state: 0 open Dependabot
     PRs, 0 open security advisories.
   - **Phase 4 P0 backlog FULLY CLOSED** (5/5 items: P4.2 rate_limiter,
     P4.3 context bounding, P4.4 ControlGate ADR-0012, P4.5 Dependabot
     wave, and P4.0/P4.1 wave covering CI infrastructure restoration).

### Phase 4 design-meta candidates (deferred)

Twenty architectural questions surfaced across the Phase 4 wave
(M1–M20; M19 closed during P4.5.c verification). Captured here so
context isn't lost; not blocking. Each deserves a dedicated
discussion session — Phase 4 P0 work has now closed, so the
strategic view is informed.

The queue is the artifact; resolution is future work.

#### M1 — Auto-derive `CANONICAL_STREAMS` from Go AST

`tools/raccoon-cli/src/analyzers/drift_detect.rs` mirrors the stream
catalogue declared in `internal/adapters/nats/natsexecution/registry.go`.
Drift has hit twice (G6, G6.2) when new streams shipped without the
mirror being updated. A codegen step deriving `CANONICAL_STREAMS`
from the Go AST would eliminate the G-class drift surface
permanently.

#### M2 — `EventSpec.Subject` "prefix as published subject" convention

The `contract-audit` `event-stream-coverage` check treats
`EventSpec.Subject` as the literal published subject. Several
publishers (e.g., `PublishExecution`) append context tokens to the
spec prefix at publish time, so `Subject` is in practice a prefix.
3 of 4 execution publishers happen to align with their stream
wildcards by coincidence of prefix lengths; the SessionLifecycle
event surfaced because it did not. Extend the scanner to understand
prefix-then-context, removing the latent risk in EventSpecs that
pass only by happenstance.

#### M3 — Document raccoon-cli profile semantics

The `fast`, `ci`, and `deep` profiles run the same check set; `ci`
promotes warnings to errors and prefixes them with `[ci]`. The
mapping is hardcoded in `tools/raccoon-cli/src/gate/mod.rs` with no
external config and no user-facing documentation. Surface this in
`tools/raccoon-cli/README.md` or `docs/operations/` so the
promotion rule is discoverable rather than discovered.

#### M4 — `walk_go_files` doc-vs-reality cleanup

The doc comment on `walk_go_files` in `arch_guard.rs` claims
"non-test, non-vendor", but the function filters only `vendor/`.
The test-file filter lives inside `check_deploy_boundary`'s closure
(P4.1.3.b). Not a bug today (only deploy-boundary calls
`walk_go_files`), but a trap for future callers. Either align the
doc with the behaviour or move the filter into `walk_go_files`
and remove it from the closure.

#### M5 — Application clients exposing domain types in public contracts

`executionclient` and `monitoringclient` return and accept domain
types directly in their public APIs (e.g.,
`SessionListReply.Sessions []execution.Session`). This is why `cmd/`
must import `internal/domain/*` for composition wiring — the
clients don't hide domain behind DTOs. ADR-0005's "cmd sees
everything" makes the current state defensible; the question is
whether an anti-corruption boundary between application and its
consumers would be net positive (more isolation, more boilerplate,
more test surface). May spawn a sub-ADR.

#### M6 — ADR-0005 clarification: composition vs invocation

ADR-0005 says "cmd sees everything". P4.1.3.c.ii clarified what
that means in practice: cmd may reference domain types for
composition, but should not invoke domain functions directly
(those are routed through application clients). Add a companion
note to ADR-0005, or amend it in place, articulating the
composition-vs-invocation distinction so the refined raccoon-cli
rule and the ADR speak the same language.

#### M7 — Dual-semantic counter for pre-publish vs post-publish observability

`Counter("filled")` (and analogous counters in execute-scope actors)
is incremented AFTER the NATS publish that signals the same event.
This creates a sub-microsecond observability window where subscribers
see the published event before the counter reflects it.

Current consumers tolerate this: HTTP `/statusz` timing dominates the
race; intra-actor `logStats()` reads are race-free by Hollywood's
single-threaded mailbox; Prometheus `/metrics` uses a separate counter
set. P4.1.8 added an `eventuallyAtLeast` helper for test consumers.

**Candidate work**: introduce dual-semantic counters — e.g.,
`submit_attempted` (incremented before publish) and `submit_succeeded`
(incremented after publish ack). Tests synchronize on
`submit_attempted` for pre-publish observability; production `/statusz`
keeps `submit_succeeded` for honest post-publish accounting.

**When to revisit**: if new production observability surfaces require
sub-millisecond timing (real-time dashboards, alerting on counter
rates), or if cross-actor counter reads emerge.

Decision context: P4.1.8.c investigation; Option (C) accepted in
P4.1.8.d (keep eventually-poll helper, skip actor reorder).

#### M8 — Synthetic data path for analytical surface

The analytical pipeline (`writer` → ClickHouse → `gateway` queries)
depends on live Binance Futures WSS data via `ingest`. CI runners
typically cannot reach Binance (network egress / geo-blocking on
GitHub Actions). Smoke Composed Pipeline works around this with
Go-level synthetic data, but Smoke Analytical needs the full stack
plus the live feed.

**Candidate work**: introduce a synthetic ingester (or synthetic
data injection point upstream of `writer`) that emits the same
downstream events `ingest` would produce. Would unblock smoke-
analytical for PR CI.

**Status**: smoke-analytical deferred to scheduled/manual workflow
(P4.1.6.b) until the synthetic seeder exists.

#### M9 — CI log-error scan robustness

The "Scan for error-level logs" step in the smoke-analytical
workflow greps for `"level":"error"` only. Warn-level logs (e.g.,
`ingest` unable to reach an external service surfacing as `warn`,
not `error`) escape detection. The step PASSED even when the
end-to-end pipeline produced no data, contributing to false
confidence in the CI signal.

**Candidate work**: extend the scan to flag warn-level too, OR add
a health-endpoint assertion (writer `active_trackers`, gateway
readiness counters), OR fail-fast when upstream produces no events
within a fixed window.

**When to revisit**: pre-requisite for restoring smoke-analytical
to PR CI alongside M8 (synthetic seeder). Without M9, the restored
job would silently pass on broken pipelines again.

#### M11 — Subject-filter validation in NATS consumer fallback

`internal/adapters/nats/natsexecution/consumer.go:79` falls back to
`c.spec.Event.Subject` (bare base subject) when `FilterSubjects` is
not supplied on the `ConsumerSpec`. If the bare base subject has no
`.>` (or `.*` etc.) wildcard suffix and the publisher writes at
qualified sub-subjects, NATS JetStream silently delivers zero
messages to the consumer — the producer side is the same channel,
but the subscription pattern never matches.

P4.1.11 found 9 test sites across `writerpipeline` and
`natsexecution` integration tests that hit exactly this fallback
with the bare `registry.PaperOrderSubmitted` EventSpec
(`execution.events.paper_order.submitted`) while publishers wrote to
`execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}`.
The tests were latent through the entire project history until
P4.1.10 unmasked them; production paths were safe because they go
through helper specs that supply the wildcard form.

This is the consumer-side counterpart to **M2**
(`EventSpec.Subject` "prefix as published subject" convention).
M2 closes the publisher-side scanner gap; M11 closes the consumer-
side runtime gap. Same underlying convention; different enforcement
surface.

**Candidate work**: at `consumer.go:79`, validate the subject
pattern before binding the consumer. If the spec's `Event.Subject`
contains no wildcard segment AND the publisher's known
`Event.Subject` convention is "prefix-then-context" (M2-aware), emit
a startup-time warning (or a panic in `_test.go`-compiled builds)
so future tests bypassing production helpers cannot silently miss
events. The check could also live in `natskit.NewConsumerSpec`
factory or in a separate static-analysis check; design choice is
open.

**Why deferred**: P4.1.11.a fix (single helper, 9 call sites)
prevents the specific manifestation. Defensive runtime validation
is broader architectural work that needs the M2 scanner design
finalised first (so both sides apply the same prefix-vs-context
heuristic).

#### M12 — Sweep all timestamp-keyed `DeduplicationKey` impls atomically

P4.1.10 fixed `Strategy.DeduplicationKey()` (Unix → UnixNano) when
the family tests it targeted surfaced the silent JetStream Duplicate-
Window drop. P4.1.11.a then had to extend the same recipe to
`ExecutionIntent`, `Decision`, `RiskAssessment`, and `Signal` once
the writerpipeline + natsexecution restart_recovery tests exercised
those types. Each was the *identical* one-line fix. The pattern is
clear: any `DeduplicationKey` method that interpolates
`Timestamp.Unix()` is latent — production cadence (kline ≥1s) hides
it for current callers, but any new tight-loop producer (test or
future code) will re-discover the bug.

Two `DeduplicationKey` impls are exempt because they don't use a
timestamp suffix: `SessionLifecycleEvent.DeduplicationKey()`
(session-id + status) and `ObservationTrade.DeduplicationKey()`
(source + tradeID).

**Candidate work**: add a quality-gate / raccoon-cli check that
flags any `DeduplicationKey` implementation containing
`Timestamp.Unix()` (without `Nano`). Alternatively, a domain test
that asserts no `DeduplicationKey` collides for two distinct
sub-second siblings. Either prevents the recipe from being
re-discovered piecemeal in future waves.

**Why deferred**: the four sibling impls were fixed in P4.1.11.a;
the check would be a guard, not a hotfix. Bundle with M2/M11 when
the broader "publish-side / consume-side contract validation" work
is scoped.

#### M13 — NATS header-extracted deadlines (responder layer)

P4.3.a fixed the unbounded `context.Background()` in
`natskit/request_reply_responder.go` by adding a configurable
`requestTimeout` field (default 5s). The alternative considered but
deferred was extracting the deadline from NATS request headers
(e.g., a `Nats-Expected-Deadline` header), allowing callers to
specify per-request bounds. More honest deadline propagation from
HTTP through gateway through actor handler.

**Candidate work**: define a header convention; update
`RequestReplyResponder` to honor the header if present, falling
back to its configured default otherwise; update gateway emitters
to populate the header from the HTTP request's deadline.

**Why deferred**: the configurable default is sufficient for current
operations; per-request deadline propagation matters more for
externally-driven load patterns we don't have yet. Single timeout
field in `ControlResponderConfig` adequate today.

#### M14 — Per-use-case ControlRouter timeouts

`ControlRouterActor` uses a single `RequestTimeout` for every
use-case dispatch (P4.3.a `handlerContext` helper). Some use cases
are heavier than others — `compileConfig` involves JSON Schema
validation; `getConfig` is a single KV read. A single timeout
forces compromise: large enough for the slow case, looser than
needed for the fast case.

**Candidate work**: extend `ControlRouterConfig` with optional
per-use-case overrides; `handlerContext` accepts a use-case
identifier and applies the appropriate timeout per operation.

**Why deferred**: single timeout adequate for current operations
(none yet exhibit measurable timeout-driven friction). Pull into
scope only when a specific use case routinely hits the global cap.

#### M15 — Custom raccoon-cli context analyzer

P4.3.a enabled the standard `contextcheck` golangci-lint linter
to flag bare `context.Background()` patterns. `contextcheck` catches
generic Go patterns but doesn't understand the Hollywood mailbox
boundary: it can't distinguish a legitimate fresh-context creation
(actor handler that has no caller context) from an accidental one
(handler that has a context but ignores it). The 3 `//nolint:contextcheck`
suppressions added in P4.3.a are project-specific rules that
contextcheck cannot express.

**Candidate work**: extend `tools/raccoon-cli`'s `arch-guard`
analyzer with project-specific context flow rules — e.g., "inside
a `Receive(c *actor.Context)` method, fresh `context.Background()`
is allowed; inside a function that takes `ctx context.Context`,
deriving from `Background()` requires a justification annotation".

**Why deferred**: `contextcheck` + `//nolint` comments are
sufficient today (3 known suppressions, all with rationale). The
custom analyzer earns its keep only if more Hollywood-boundary
patterns surface that contextcheck cannot classify correctly.

#### M16 — ControlGate cached state with staleness threshold (H1)

P4.4 design discussion option H1, deferred at P4.4.a in favour of
documenting the current fail-open posture (ADR-0012) and adding
observability (`gate_read_failures_total`). The cached-state variant
would memoize the last successful gate read in process memory and
serve it during transient KV failures, falling back to fail-closed
only after a configured staleness threshold (e.g., 30s). Combines
availability of pure fail-open with the safety of fail-closed
during sustained outages.

**Why deferred**: requires operational data from the new counter
before the threshold can be chosen non-arbitrarily. A non-zero
`kv_error` or `ctx_timeout` rate at scale would make this concrete;
a flat-zero rate confirms M16 is unnecessary.

#### M17 — ControlGate conditional fail-closed (H2)

P4.4 H2: bifurcate the IsHalted contract by adapter mode.
`AdapterVenue` + `CredentialPresent` callers would fail-closed
(safety prioritized on the live path); paper / dry-run callers
would keep the current fail-open. Matches the risk surface (only
the live + creds path can cause real harm) to the safety posture.

**Why deferred**: adds a second code path with mode-conditional
semantics; subtle bugs possible around mode transitions. Need
M16's operational data first to judge whether the simpler
single-path posture has a real cost.

#### M18 — ControlGate `ErrKeyNotFound` distinction (H3)

P4.4 H3: split today's "any read failure = fail-open" into
"first-boot (no operator write yet) = fail-open by design" vs
"real read failure = different posture". With the counter in place,
operators can already see `key_not_found` separately from
`kv_error` / `ctx_timeout` / `unmarshal_error`. M18 would change
behaviour on the latter three categories independently of the
first.

**Why deferred**: composes with M16/M17. Choosing M18 alone (e.g.,
strict fail-closed only on `kv_error` + `ctx_timeout` + `unmarshal_error`,
keeping `nil_bucket` and `key_not_found` fail-open) would be the
smallest semantic step away from current posture; worth considering
once counter data exists.

#### M19 — Dependabot SHA-pinning behavior verification — CLOSED

P4.5 investigation flagged "GitHub Actions in Dependabot" as a
potential structural friction: PRs reference v-tags (`@v5`) which the
SHA-pinning policy rejects. P4.5.a's deeper inspection hypothesized
the friction is largely self-correcting on rebase — Dependabot
preserves the existing workflow file's pin style, so once the PR
is rebased onto a base where actions are SHA-pinned (post-P4.1),
the regenerated diff is SHA-style and passes CI.

**Verified in P4.5.c Phase 1 (2026-05-23) via PR #6 rebase test**:
- Pre-rebase diff: `-uses: actions/checkout@v4` / `+uses: actions/checkout@v6`
  (tag-style, generated against pre-pinning base).
- Post-rebase diff: `-uses: actions/checkout@34e1148... # v4.3.1` /
  `+uses: actions/checkout@de0fac2e... # v6.0.2` (SHA-style with version
  comment, generated against current SHA-pinned base).
- 8 sites in `ci.yml` + 1 in `smoke-analytical.yml` updated automatically.
- All 4 Actions PRs (#6/#2/#3/#4) merged in P4.5.c with the same
  rebase pattern; each preserved SHA-pinning.

**Outcome**: no config change required. Future Dependabot Actions PRs
will be auto-mergeable after a single `@dependabot rebase` comment.
M19 closes.

#### M20 — Dependabot dedup for manually-applied upgrades

P4.5 surpresa #2: when a dependency is upgraded manually (e.g.,
P4.1.1 bumped `golangci-lint-action` 6→9 via direct workflow edit),
Dependabot does not auto-close the corresponding open PR. Manual
close required (done in P4.5.a for PR #5). With weekly Dependabot
cadence + 17 PRs from a single sync, a similar drift on multiple
PRs is plausible.

**Candidate work**: investigate whether Dependabot has a config
option to detect "target version already at or beyond what is on
the default branch" and auto-close. Alternatively, a small post-merge
GitHub Action that closes any open Dependabot PR whose target
version is now ≤ main's current pin.

**Why deferred**: low frequency to date (1 known instance, #5).
Worth revisiting if the pattern recurs after the routine batch
(P4.5.b) or after future manual upgrades.

#### M21 — TRUTH-MAP anchor validator (`raccoon-cli check truth-map`)

**Origem:** avaliação em H-1 (2026-05-24).
**Status:** Deferido com triggers de reabertura.

**Contexto:** TRUTH-MAP entregue em H-1 com **~37 capability rows**
e **~80 anchor strings** (code anchors + test anchors). Um analyzer
estático que valida cada anchor (arquivo existe + símbolo presente)
foi estimado em **~700 LoC Rust + ~150 LoC de testes + integração
ao gate profile**, com risco médio-alto de false positives
(sub-tests `t.Run(...)`, generics, build-tag-exclusive symbols).
**Zero amostras empíricas de drift** existiam no momento da
avaliação.

**Decisão:** **P7** (sem perda de disciplina documental — TRUTH-MAP
atualizado no commit que move o anchor) e **P9** (maintainer review
em todo PR) são a **primeira camada**. Adiar o analyzer até evidência
empírica de que a primeira camada é insuficiente.

**Triggers de reabertura** (qualquer um basta):

- **Quantitativo** — 3+ correções de anchor drift em PRs subsequentes
  dentro de **6 ondas consecutivas**. Sinal: P7 manual +
  maintainer review estão consistentemente falhando em catch.
- **Qualitativo** — **1 incidente** onde TRUTH-MAP declarou
  capability cujo code anchor já não existia em `main` (drift
  silencioso passou maintainer review). Sinal: human review não
  escala com complexidade crescente.
- **Contextual** — TRUTH-MAP cresce acima de **~60 capability
  rows** (atualmente ~37). Sinal: escala manual atingiu limite
  cognitivo razoável de revisão.

**Quando reaberto:** avaliar **versão completa** (~700 LoC,
validação de file + symbol) versus **versão minimalista** (~200 LoC,
apenas "file exists"; ~30% do valor por ~30% do custo) com dados
reais sobre quais drift patterns ocorreram. A escolha do flavor
depende de qual trigger disparou: quantitativo → minimal pode
bastar; qualitativo ou contextual → versão completa provavelmente
necessária.

**Captura única:** este design-meta é a fonte canônica do
deferimento. PROGRAM-0001, ADR-0016 e CLAUDE.md **não** repetem
a entrada — sobrevive ao fechamento de PRD/Fase em um único lugar.

### Available work (P1/P2, opt-in)

- Code-side audit of `internal/` and `cmd/` (Phase 3 was
  environment-only).
- Test coverage analysis (current ratio ≈ 0.71; ~288 test files vs
  ~402 prod files).
- Security deep dive post the P3.3 toggles (real residual exposure?).
- Performance audit (compose stack startup, smoke duration trends).
- **P3.5.cleanup**: 28 minor shellcheck issues + 32 SC1091 (source-
  path directives). Cosmetic.
- **P3.7.1**: `.vscode/` configs if owner uses VS Code.
- **P3.8.1**: `.claude/hooks/` if a concrete pattern surfaces.

### Architectural decisions registry (Phase 3)

For session orientation; full ADRs are in `docs/decisions/`.

| Decision | Source |
|---|---|
| License: PolyForm Noncommercial 1.0.0 | P3.1; LICENSE + SECURITY.md |
| Pre-commit framework: lefthook (Go-based, no Node) | P3.4; `lefthook.yml` |
| Commit message convention: custom shell validator (no commitlint) | P3.4; `scripts/validate-commit-msg.sh` |
| Editor formatting standard: EditorConfig | P3.7; `.editorconfig` |
| Line-ending normalization: LF everywhere via `.gitattributes` | P3.7 |
| Tool-version manifest: `.tool-versions` (asdf/mise compatible) | P3.7 |
| Branch protection: 3 required status checks, linear history, no force-push | P3.3; `docs/operations/github-settings.md` |
| Security toggles: secret scanning + push protection + dependabot + private-vuln-reporting | P3.3 |
| Actions: SHA pinning required (workflow migration pending) | P3.3 |
| Issue/PR templates: kept (4 templates from pre-Phase 3) | Pre-P3 |
| AI agent automation: `.claude/commands/` (5) + `.claude/agents/` (2) | P3.8 |
| Institutional knowledge: `docs/CONTRIBUTING.md` "For AI agents" section | P3.9 |

`make verify` is green locally. Any new red on `make verify` is a
real regression — not historical debt.

---

## How to keep this document fresh

`RESUMPTION.md` only earns its keep if it stays current. The trigger
for updating it is:

- **Phase transition** (e.g., when Phase 1A closes, update the phase
  table to show 1B in progress).
- **New known gap discovered** (add to G section).
- **Gap resolved** (move from G section to a "Recently resolved"
  appendix, or just remove).
- **Significant feature shipped** (add to "Current functional
  state", remove from "Deliberate non-features" if applicable).

If you find yourself wondering whether this doc reflects reality,
**that itself is the trigger to update it**.

---

## Reading further

| If you want | Go to |
|---|---|
| System overview | [`README.md`](README.md) |
| Architecture | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| Topology, ports, streams | [`RUNTIME.md`](RUNTIME.md) |
| HTTP endpoints | [`HTTP-API.md`](HTTP-API.md) |
| Daily workflow | [`DEVELOPMENT.md`](DEVELOPMENT.md) |
| PR rules | [`CONTRIBUTING.md`](CONTRIBUTING.md) |
| Domain deep dives | [`domain/`](domain/README.md) |
| Operational procedures | [`operations/`](operations/README.md) |
| Architecture decision records | [`decisions/`](decisions/README.md) |
| Historical material | [`legacy/`](legacy/README.md) |
| Terminology | [`GLOSSARY.md`](GLOSSARY.md) |
