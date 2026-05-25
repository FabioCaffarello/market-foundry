# PROGRAM-0002 — Fase Wire

**Status:** Closed
**Date:** 2026-05-25
**Owner:** Repository maintainer (Fabio Caffarello)
**Relates to:**
[`../decisions/0017-event-envelope-and-versioning.md`](../decisions/0017-event-envelope-and-versioning.md),
[`../decisions/0018-protobuf-contract-layer.md`](../decisions/0018-protobuf-contract-layer.md),
[`../decisions/0019-deterministic-replay-time-invariants.md`](../decisions/0019-deterministic-replay-time-invariants.md),
[`../decisions/0020-sequencing-and-time-normalization.md`](../decisions/0020-sequencing-and-time-normalization.md),
[`PROGRAM-0001-foundation.md`](PROGRAM-0001-foundation.md),
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest",
[`../RESUMPTION.md`](../RESUMPTION.md)

---

## Objetivo

Instalar no `market-foundry` a **camada de contrato wire** que governa
os eventos do mesh JetStream — schemas canônicos em protobuf, código
Go gerado validado por analyzer, e as invariantes determinísticas
(replay, sequenciamento, pureza do domínio) que sustentam a tese
"backtest = produção".

Esta Fase entrega **infraestrutura e contrato**, não migração runtime.
Os 11 streams continuam usando o envelope JSON legado
(`internal/shared/envelope/`) durante toda a Fase Wire; a substituição
por proto runtime é execução desta decisão arquitetural, programada
para uma fase futura (provavelmente PROGRAM-0003).

A Fase Wire é a primeira da Fase Harvest com **tooling externo**
(buf CLI + protoc-gen-go). Isso muda o bootstrap; cada onda da Fase
inclui o cuidado de manter `scripts/bootstrap-check.sh`,
`.tool-versions`, e `docs/DEVELOPMENT.md` em sync.

---

## Escopo (Ondas)

| Onda | Escopo resumido | Entregas principais |
|------|------------------|---------------------|
| **H-3.a** | Proto skeleton + buf tooling | `proto/` com `buf.yaml`, `buf.gen.yaml`, `registry.json`, `envelope/v1/envelope.proto`, `marketdata/v1/trade.proto`; `make proto-lint/proto-gen/proto-breaking`; bootstrap-check valida `buf`; `.tool-versions` adiciona `buf`. Sem código Go gerado tracked, sem analyzer raccoon-cli. ADRs 0017/0018 continuam `Proposed`. |
| **H-3.b** | Code generation + converters + analyzer | `internal/shared/contracts/envelope/v1/envelope.pb.go` + um payload piloto gerado e tracked; converters proto ↔ domain com round-trip tests; raccoon-cli `check proto` integrado em `make verify`. **Promove ADR-0017 e ADR-0018 a `Accepted`** (critérios revisados pela erratum de H-3.a). |
| **H-4** | Replay infrastructure + Sequencer | `internal/shared/replay/` recorder + player; ports `clock.Clock` + `random.Source` + migração dos call sites em `internal/domain/` que hoje usam `time.Now`; `internal/shared/sequencer/` com `seq` monotônico per stream key e persistência em NATS KV `SEQUENCER_STATE_LATEST`; raccoon-cli `check determinism` (INV-D1) integrado em `make verify`; pelo menos um golden test end-to-end por uma stream representativa; CI step rodando o golden N=50 vezes (INV-D4). **Promove ADR-0019 e ADR-0020 a `Accepted`**. |

H-3 foi dividida em sub-ondas H-3.a e H-3.b por escopo técnico
(decisão de planejamento de sessão): instalar tooling externo na
mesma onda em que se gera código Go + se escreve analyzer Rust
sobrecarrega revisão. Cada sub-onda é um PR independente; ambas
fechadas, ADR-0017 e ADR-0018 são promovidas em H-3.b.

---

## Não-Escopo

- **Migração runtime dos 11 streams para proto.** O contrato é
  instalado nesta Fase; sua adoção em produção (substituir o envelope
  JSON legado por proto nas publishes/consumes) ocorre em fase
  futura. O envelope legado (`internal/shared/envelope/`) coexiste
  com o canônico durante toda a Fase Wire e além.
- **HTTP-API wire format.** Permanece JSON; ADR-0018 explicitamente
  escopou proto para o mesh, não para a HTTP-API.
- **CBOR / MessagePack / Avro.** Deferidos. Para a surface de
  delivery do cliente Odin (H-11/H-12), a decisão pode revisitar
  CBOR; não nesta Fase.
- **Cliente Odin / WASM.** Permanece mapeado para H-12+ dentro de
  `client/` (P8 da Fase Harvest).
- **Multi-venue / canonical instrument.** ADR-0021 (`Proposed`) é
  promovida em H-6. Nesta Fase, o campo `instrument` no envelope
  proto fica como `string` com comentário declarando substituição
  futura por mensagem estruturada quando ADR-0021 promover.
- **Stream migration runbook.** Documento operacional de
  dual-publish / cutover por stream é responsabilidade da fase de
  migração, não desta Fase.
- **Composição de targets em `make proto-gate`.** Os três targets
  individuais (lint/gen/breaking) bastam para a Fase Wire; se um
  composite fizer sentido em onda futura, decide-se lá.

---

## Princípios governantes

A Fase Wire opera sob o **protocolo P1–P9** documentado em
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest". Particularmente
relevantes para esta Fase:

- **P2** — Raccoon path read-only. As capacidades validadas no
  raccoon (proto/, internal/contracts/, replay/) informam sem
  importar; cada leitura é justificada.
- **P3** — Documento primeiro. ADRs 0017–0020 (todas `Proposed`,
  Fase Harvest H-2) governam esta Fase; nenhuma implementação
  antecede a decisão arquitetural.
- **P5** — Cada onda evolui raccoon-cli quando adiciona invariante
  arquitetural. H-3.a **não** adiciona analyzer (não há invariante
  runtime — proto schemas não rodam). H-3.b adiciona `check proto`.
  H-4 adiciona `check determinism`.
- **P6** — Pause-and-report durante toda a Fase. Tooling externo
  acumula risco silencioso; pause antes de improvisar.
- **P7** — Sem perda de disciplina documental. ADRs 0017/0018
  promovem em H-3.b com critérios revisados pela erratum de
  H-3.a (commit 0). ADRs 0019/0020 promovem em H-4.
- **P9** — PR-based delivery por sub-onda. Branches:
  `feat/h-3a-proto-skeleton`, `feat/h-3b-codegen-converters`,
  `feat/h-4-replay-sequencer`. Nenhuma onda abre antes do merge
  da anterior em `main`.

---

## Critérios de aceite da Fase

A Fase Wire fecha quando **todos** os critérios abaixo são
verdadeiros simultaneamente:

- [x] Ondas H-3.a, H-3.b, H-4 fechadas. Cada onda registrou
  fechamento explícito com `make verify` GREEN e RESUMPTION
  atualizado no commit de fechamento.
- [x] `proto/` contém schemas para o envelope canônico
  (`envelope/v1/envelope.proto`) e pelo menos um piloto de payload
  (`marketdata/v1/trade.proto`), com entradas correspondentes em
  `proto/registry.json` (H-3.a).
- [x] `make proto-lint`, `make proto-gen`, `make proto-breaking`
  executam corretamente; `buf` é validado pelo
  `scripts/bootstrap-check.sh` e listado em `.tool-versions`
  (H-3.a).
- [x] Código Go gerado em
  `internal/shared/contracts/envelope/v1/envelope.pb.go` (e
  análogo para o piloto) está tracked no repo, com testes de
  round-trip serialize/deserialize (H-3.b).
- [x] Converters proto ↔ tipos de domínio existem em
  `internal/shared/contracts/<family>/v<n>/converter.go` (ou
  equivalente) com teste unitário (H-3.b).
- [x] Analyzer raccoon-cli `check proto` validando
  `proto/registry.json` ↔ `.proto` ↔ Go gerado, integrado em
  `make verify` via `quality-gate` (H-3.b).
- [x] ADR-0017 e ADR-0018 promovidos a `Accepted` no commit que
  ship H-3.b (critérios revisados pela erratum H-3.a).
- [x] `internal/shared/replay/` com recorder + player de fixtures
  determinísticas (H-4).
- [x] Ports `clock.Clock` e `random.Source` introduzidos; call
  sites em `internal/domain/` que hoje usam `time.Now`/`math.rand`
  migrados (H-4).
- [x] `internal/shared/sequencer/` com `seq` monotônico per stream
  key `(venue, instrument, event_type)`; persistência em NATS KV
  `SEQUENCER_STATE_LATEST`; testes unitários de monotonicidade
  (INV-D2); counter `marketfoundry_consumer_seq_gap_total{stream_key}`
  exposto (H-4).
- [x] Analyzer raccoon-cli `check determinism` (INV-D1) integrado
  em `make verify` via `quality-gate` (H-4).
- [x] Pelo menos um golden test end-to-end byte-stable
  (typically `observation → evidence` para `OBSERVATION_EVENTS`),
  validando INV-D3 (H-4).
- [x] CI step rodando o golden representativo N=50 vezes e
  validando byte-stability uniforme (INV-D4) (H-4).
- [x] ADR-0019 e ADR-0020 promovidos a `Accepted` no commit que
  ship H-4.
- [x] PROGRAM-0002 transita para `Closed` na entrega final de H-4;
  entrada Changelog correspondente.

---

## ADRs governantes

| ADR | Escopo | Status no início da Fase | Promovido por |
|-----|--------|--------------------------|----------------|
| 0017 | Event envelope and versioning | Proposed (entregue em H-2) | H-3.b |
| 0018 | Protobuf contract layer | Proposed (entregue em H-2) | H-3.b |
| 0019 | Deterministic replay and time invariants | Proposed (entregue em H-2) | H-4 |
| 0020 | Sequencing and time normalization | Proposed (entregue em H-2) | H-4 |

Os critérios de promoção a `Accepted` foram revisados pela erratum
de H-3.a (commit 0 da H-3.a) para separar **decisão arquitetural**
(contrato + tipos + analyzer) de **execução de decisão** (migração
runtime dos 11 streams). Os critérios revisados são H-3.b-completable
sem depender de migração; veja seção "Promoção para Accepted" em
cada ADR.

Nenhuma ADR nova é esperada nesta Fase. Se durante H-3.b ou H-4
surgir necessidade arquitetural não coberta pelas quatro acima, P6
(pause-and-report) e nova ADR sob `decisions/0024+`.

---

## Riscos

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Tooling externo (`buf`, `protoc-gen-go`) introduz dependência nova no bootstrap | Médio — onboarding e CI falham se ausente | `scripts/bootstrap-check.sh` valida presença em H-3.a; `.tool-versions` declara para asdf/mise; `docs/DEVELOPMENT.md` documenta instalação |
| Drift entre `.proto` files e `.pb.go` gerados | Alto — produção pode usar tipo Go que não corresponde ao contrato declarado | raccoon-cli `check proto` em H-3.b valida em CI; PRs que mudam `.proto` sem regerar falham `make verify` |
| Schemas evoluem durante a Fase, gerando ruído de revisão | Médio — diffs largos em `*.pb.go` confundem reviewer | Convenção de PR: `.proto` em commit separado de `.pb.go`; reviewer foca em `.proto`, skim em `.pb.go` |
| Migração de call sites de `time.Now` em H-4 toca muitos arquivos do domínio | Alto — refactor amplo, risco de regressão | H-4 entrega analyzer `check determinism` no mesmo PR que migra; CI falha em qualquer regressão futura; golden tests sealam comportamento |
| Sequencer state corrompe se NATS KV `SEQUENCER_STATE_LATEST` falhar | Médio — restart pode duplicar ou pular sequências | ADR-0020 já especifica: monotonicidade sempre, density best-effort; consumer-side dedup via `idempotency_key` absorve |
| Convivência de envelope JSON legado + canônico proto em paralelo confunde contribuidores | Médio — risco de uso incorreto em código novo | Documentado em ADR-0017 ("legacy not retired by this ADR"); GLOSSARY entries distinguem "transport envelope" (legado) de "canonical event envelope" (novo); migração runtime fora de escopo desta Fase deixa isso explícito |
| `proto-gate` mencionado em ADR-0018 body mas descopado em critérios de aceitação cria expectativa stale | Baixo | Erratum H-3.a documenta como descritivo, não normativo; futuro erratum pode revisar se composite for adotado ou descartado |
| H-4 envolve ports (`clock.Clock`) que mudam assinaturas em `internal/domain/`; quebra eventuais call sites externos | Médio | Onda H-4 audita call sites antes; migração é mecânica via go tooling; pause-and-report se houver call site que não migra trivialmente |

---

## Referência ao raccoon (sem cópia)

Capacidades validadas no raccoon que informam (sem migrar para) a
Fase Wire. Cada uma é reescrita no foundry na onda apropriada,
respeitando layer sovereignty + single-writer invariant +
configctl authority:

- `proto/` tree (envelope, marketdata, insights schemas validados em
  produção raccoon).
- `proto/registry.json` (formato de inventário de schemas).
- `buf.yaml` + `buf.gen.yaml` (configurações de lint v2 STANDARD +
  COMMENTS e breaking WIRE_JSON).
- `internal/shared/contracts/` (boundary para Go gerado de proto).
- `internal/shared/codec/` (payload codec equivalência semântica
  JSON ↔ proto).
- `internal/shared/replay/` (recorder + player + golden tests).
- Sequencer pattern (raccoon's `InstrumentStream.BuildEnvelope`
  como referência para per-stream monotonicidade).
- Domain purity guard (raccoon's `check-domain-isolation.sh` como
  referência conceitual; foundry implementa via raccoon-cli
  analyzer per P5).

Esta lista é informativa; a sequência exata em que cada capacidade
é portada é decidida em cada onda, não nesta PRD.

---

## Evidence

- [`../decisions/0017-event-envelope-and-versioning.md`](../decisions/0017-event-envelope-and-versioning.md)
- [`../decisions/0018-protobuf-contract-layer.md`](../decisions/0018-protobuf-contract-layer.md)
- [`../decisions/0019-deterministic-replay-time-invariants.md`](../decisions/0019-deterministic-replay-time-invariants.md)
- [`../decisions/0020-sequencing-and-time-normalization.md`](../decisions/0020-sequencing-and-time-normalization.md)
- [`PROGRAM-0001-foundation.md`](PROGRAM-0001-foundation.md) — Fase
  anterior; entregou as quatro ADRs governantes desta Fase como
  `Proposed`.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — protocolo
  P1–P9 canônico.
- [`../RESUMPTION.md`](../RESUMPTION.md) → "Fase Harvest" — state
  sentinel.

---

## Changelog

- **2026-05-25** — PROGRAM-0002 created. Status `Active`. Ondas
  H-3.a / H-3.b / H-4 declared. ADRs 0017 / 0018 / 0019 / 0020 são
  governantes (todas `Proposed` no início da Fase, entregues em
  H-2). Lands como entrega de H-3.a junto com `proto/` skeleton +
  buf tooling + bootstrap update.
- **2026-05-25** — **Closed**. Onda H-4 delivered the remaining
  seven acceptance criteria across fourteen commits, completing
  the Fase Wire scope: `internal/shared/replay/` (recorder +
  player + fixture format), `internal/shared/clock/` +
  `internal/shared/random/` ports, `internal/shared/sequencer/`
  package, NATS KV bucket `SEQUENCER_STATE_LATEST` + adapter,
  `marketfoundry_consumer_seq_gap_total` counter, migration of
  all five direct `time.Now` call sites in `internal/domain/`
  production code (`DefaultVerificationScope`,
  `DefaultControlGate`, `NewActivationSurface`, `Session.Close`,
  `Session.Halt`), raccoon-cli `check determinism` analyzer
  (Step 7 of the gate), end-to-end golden test +
  N=50 byte-stability validation, and finally the dual ADR
  promotion (0019 + 0020 → Accepted) plus this closure.

  The cascade analysis early in H-4 surfaced a discrepancy
  between the prompt's assumption (~10 call sites in one test
  file) and reality (5 production call sites + ~110 test call
  sites in ~24 files). The user-confirmed mitigation was
  Option (C) — production code migration plus test-file
  exemption in the analyzer — preserved by ADR-0019 References
  with the rationale documented inline. No erratum to the ADR
  texts was required.

  ADR-0020 critério 5 (writer-binary Sequencer integration in
  the running stack) was scoped to a follow-up fase: the
  architectural decision is Accepted with the package and
  primitives shipped; runtime wiring per writer is
  execution-of-decision tracked in PROGRAM-0003+.

  Migração runtime dos 11 streams para proto continues to be
  deferred (it was non-scope for Fase Wire from the outset, per
  the Non-Escopo section). The Wire layer is installed and
  enforced; its runtime adoption is a separate execution phase.
