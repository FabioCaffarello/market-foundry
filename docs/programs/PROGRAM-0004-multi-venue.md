# PROGRAM-0004 — Fase Multi-venue

**Status:** Active
**Date:** 2026-05-25
**Owner:** Repository maintainer (Fabio Caffarello)
**Relates to:**
[`../decisions/0021-canonical-instrument-and-venue-model.md`](../decisions/0021-canonical-instrument-and-venue-model.md)
(Proposed),
[`../decisions/0022-multi-venue-normalization-policy.md`](../decisions/0022-multi-venue-normalization-policy.md)
(Proposed),
[`../decisions/0017-event-envelope-and-versioning.md`](../decisions/0017-event-envelope-and-versioning.md)
(Accepted, defines envelope `venue` + `instrument`),
[`PROGRAM-0001-foundation.md`](PROGRAM-0001-foundation.md) (closed),
[`PROGRAM-0002-wire.md`](PROGRAM-0002-wire.md) (closed),
[`PROGRAM-0003-observability.md`](PROGRAM-0003-observability.md)
(Active),
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest",
[`../RESUMPTION.md`](../RESUMPTION.md)

---

## Objetivo

Tornar o `market-foundry` **estruturalmente multi-venue** —
introduzir o modelo canônico de instrumento (`CanonicalInstrument`
per ADR-0021), refatorar os dois adapters Binance existentes
(`binances`, `binancef`) para emitirem o modelo canônico, expandir
para um segundo venue (Bybit) para provar paridade cross-venue, e
codificar a política de normalização cross-venue (ADR-0022).

Esta Fase entrega **infraestrutura de cross-venue** — o modelo, a
política, os dois primeiros adapters compliant. Não entrega
delivery futures discrimination além do binancef, options, ou
cross-listed asset alias resolution (todos non-goals explícitos,
ver abaixo).

PROGRAM-0001 (Foundation, closed) entregou ADR-0021 + ADR-0022
como `Proposed` em H-2. PROGRAM-0002 (Wire, closed) adicionou o
envelope `instrument` field (string) per ADR-0017. PROGRAM-0003
(Observability, Active) instrumentou o stack de metrics que
multi-venue dashboards consumirão. PROGRAM-0004 implementa o
modelo + adapters que validam ADR-0021 e prepara o terreno para
ADR-0022.

---

## Escopo (Ondas)

| Onda | Escopo resumido | Entregas principais |
|------|------------------|---------------------|
| **H-6** | Canonical instrument model + Binance refactor | Distribuído em **sub-ondas H-6.a–H-6.f** por questão de cascade (descoberto em pré-flight de H-6.a: 342 `.Symbol` references em 106 production files em 31 packages). Ver "Sub-ondas H-6" abaixo. |
| **H-7** | Bybit adapter + multi-venue parity policy | Adapter Bybit (3º venue), implementando `ToCanonical`/`FromCanonical` per ADR-0021. Promove **ADR-0022** (multi-venue normalization policy) — primeira prova real de paridade cross-venue. raccoon-cli `check venue-parity` analyzer (P5). |

H-6 e H-7 são **sequenciais e estritas** — H-7 só abre após
**H-6.f mergeada em main** (P9 + sub-onda sequencing policy
abaixo).

---

## Sub-ondas H-6 — sequenciamento estrito

A descoberta do pré-flight de H-6.a (342 `.Symbol` references em
106 production files em 31 packages) tornou impossível executar
H-6 como onda única honesta — a refatoração tocaria todo o
foundry simultaneamente e produziria PR irrevisável.

H-6 é portanto implementada em **6 sub-ondas serializadas**:

| Sub-onda | Escopo | Entregas principais |
|----------|--------|---------------------|
| **H-6.a** | Domain root + Binance adapters | `internal/domain/instrument/` package (Venue, BaseAsset, QuoteAsset, ContractType, CanonicalInstrument). Refactor `ObservationTrade.Symbol` → `Instrument`. Adapters `binances` + `binancef` `Normalize` emitem `CanonicalInstrument`. Imediate readers de `ObservationTrade` migram para `.Instrument.Symbol()`. raccoon-cli `check instruments` analyzer. ADR-0021 erratum critério #4 (commit 0). PRD-0004 abertura (esta sessão). **ADR-0021 permanece `Proposed`.** |
| **H-6.b** | Evidence + Signal + Decision + Strategy + Risk domain types | Cada domain struct (`EvidenceCandle`, `Signal`, `Decision`, `Strategy`, `Risk`) migra `Symbol string` → `Instrument CanonicalInstrument`. ~30 originating struct literals atualizados. NATS KV partition keys ainda usam derived-symbol (`Instrument.Symbol()`) para back-compat. **ADR-0021 permanece `Proposed`.** |
| **H-6.c** | Application layer + actors + samplers | Query types em `analyticalclient`/`triageclient` decidem caso a caso (query params vs domain values). Sampler/evaluator internal builders migram local `symbol string` → `instrument CanonicalInstrument`. ~40 files. **ADR-0021 permanece `Proposed`.** |
| **H-6.d** | ClickHouse migration + writer back-compat read (#4b) | Nova migration adicionando columns `base`, `quote`, `contract`. Writer dual-writes (legacy `symbol` + canonical fields). Analytical client reads canonical preferred, fallback legacy. Cutover documented em runbook. Implementa **critério #4b** do ADR-0021 erratum. **ADR-0021 permanece `Proposed`.** |
| **H-6.e** | NATS subject composition decision (pause-and-report) | **Primeiro ato**: pause-and-report obrigatório. Decidir: (i) migrar NATS subject/key composition para canonical form (com window de dual-publish/dual-read se necessário), OU (ii) declarar deferral indefinido com **segundo erratum REAL ao critério #2 do ADR-0021** documentando "NATS subjects use Instrument.Symbol() (derived form) as canonical representation for routing; direct CanonicalInstrument fields not used in subjects per [justificativa]". Sem opção #2 sem erratum honesto. **ADR-0021 permanece `Proposed`.** |
| **H-6.f** | Final cleanup + ADR promotion | Remove deprecated fields/types remanescentes. Atualiza TRUTH-MAP, RESUMPTION, GLOSSARY com state final. **Promove ADR-0021 → `Accepted`** apenas se TODOS os critérios (1, 2, 3, 4a, 4b, 5) estão literalmente satisfeitos. P7 absoluto. |

### Transitory-method pattern (introduzido em H-6.a)

H-6.a precisou consumir a cascade de `.Symbol` references sem
forçar migração simultânea de todos os 31 packages. A solução —
adotada após pause-and-report e descartando dual-write — foi
introduzir um **accessor transitório com nome semanticamente
distinto** no domain type recém-migrado:

- O field `Symbol string` foi removido e substituído por
  `Instrument CanonicalInstrument` (verdade canônica).
- Um método **`VenueSymbol() string`** foi adicionado, derivando
  forma venue-native (lowercase `base+quote`) da identidade
  canônica.

Por que **`VenueSymbol`** e não `Symbol()`:

- `CanonicalInstrument.Symbol()` retorna a forma canônica
  `"BTC/USDT-spot"`.
- `ObservationTrade.VenueSymbol()` retorna a forma venue-native
  `"btcusdt"`.
- Se ambos fossem chamados `Symbol()`, em 6 meses um caller
  escreveria `t.Symbol()` esperando a forma canônica e receberia
  silenciosamente a venue-native — bug latente clássico.

Atributos da disciplina:

1. **Sunset declarado no docstring**: cada método transitório lista
   a sub-onda que o remove. `VenueSymbol()` é removida em H-6.f
   quando o último reader venue-native sai do código.
2. **Limitações documentadas**: `VenueSymbol()` é lossy para
   delivery futures (`BTCUSDT_240329` colapsa para `"btcusdt"`).
   Aceitável em H-6.a porque nenhum contrato delivery rida o
   routing path atual; H-6.e revisita o shape da subject NATS.
3. **Nome distinto do canônico**: nunca substituir um símbolo
   canônico por um símbolo transitório com o mesmo identificador.

Sub-ondas H-6.b–H-6.e podem reusar o pattern conforme novos
domain types migram — cada um define seu próprio `VenueSymbol()`
ou equivalente, todos com sunset documentado para H-6.f.

---

## Sub-onda sequencing policy

Sub-ondas H-6.a → H-6.b → H-6.c → H-6.d → H-6.e → H-6.f → H-7
executam **estritamente serial**. Próxima sub-onda abre branch
APENAS após merge da anterior em `main`.

Razões:

- **Sobreposição de código**: cada sub-onda migra mais domain
  types ou layers; paralelo garante merge conflicts no mesmo
  arquivo entre sub-ondas distintas.
- **Disciplina cognitiva**: cada sub-onda é um contexto que
  exige revisão completa. Múltiplas em flight diluem foco do
  reviewer e fragmentam o entendimento do estado parcial da
  migração.
- **TRUTH-MAP reflete estado real**: múltiplas sub-ondas
  in-flight produzem `TRUTH-MAP` inconsistente até a última
  fechar.

Esta política é **mais estrita que P4** (uma onda por vez, no
top-level) porque sub-ondas dentro de PROGRAM-0004 manipulam o
mesmo subsystema (canonical instrument migration), enquanto
ondas top-level frequentemente tocam subsystemas distintos.

---

## Não-Escopo

- **Coinbase, Hyperliquid, Kraken adapters.** H-6 + H-7 entregam
  Binance (refactor) + Bybit (novo). Os outros venues do ADR-0021
  `Venue` enum são reservados (a enum declara para
  forward-compatibility) mas adapters concretos são ondas futuras.
- **Bybit COIN-margined futures.** H-7 entrega Bybit USDT-perpetual
  como o primeiro caso. COIN-margined Bybit (se necessário) é
  onda futura.
- **Delivery futures discrimination além do binancef pattern.** H-6.a
  inclui pattern detection para `binancef` (suffix `_YYMMDD` →
  `ContractUSDTFutures`; sem suffix → `ContractPerpetual`). Outros
  venues replicam pattern quando o caso aparecer no dataset; sem
  generalização premature.
- **Options.** Strike, expiry per leg, multi-leg structures
  permanecem non-goal per ADR-0021 (linha 174-176). Foundry não
  trata options no roadmap previsível.
- **Cross-listed asset alias resolution** (BTC vs WBTC vs cbBTC).
  Per ADR-0021, são distintos `BaseAsset` values. Consumer que
  quer cross-token analytics aplica policy própria.
- **Stablecoin equivalence** (USDT vs USDC vs FDUSD). Per ADR-0021,
  distintos `QuoteAsset` values. Per-venue liquidity differs.
- **HTTP-API symbol format change.** Gateway pode expor canonical
  ou venue-friendly alias; H-6 toca apenas internal mesh.
  HTTP-API.md decisão futura.
- **Tracing / log aggregation / per-strategy SLOs.** Todos
  herdados de PROGRAM-0003 non-goals.

---

## Princípios governantes

A Fase Multi-venue opera sob o **protocolo P1–P9** documentado em
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest". Particularmente
relevantes para PROGRAM-0004:

- **P3** — Documento primeiro. ADR-0021 (canonical instrument)
  governa H-6; ADR-0022 (multi-venue normalization) governa H-7.
  Ambas já em main como `Proposed` desde H-2.
- **P4** — Uma onda por vez. **Estendida via sub-onda sequencing
  policy acima**: sub-ondas dentro de PROGRAM-0004 também são
  estritamente seriais, não apenas as top-level.
- **P5** — Cada onda evolui raccoon-cli quando adiciona invariante
  arquitetural. H-6.a adiciona `check instruments` (todo adapter
  declarado em policies/adapters.toml emite `CanonicalInstrument`).
  H-7 adiciona `check venue-parity` (cross-venue event-type
  coverage).
- **P6** — Pause-and-report ativo. H-6.e protocolo (NATS subject
  composition decision) é o caso explícito; mais surgirão durante
  as sub-ondas.
- **P7** — Sem perda de disciplina documental. ADR-0021 permanece
  `Proposed` até H-6.f satisfazer **literalmente** todos os
  critérios. Erratum REAL ao critério #4 (split #4a/#4b) lands
  em commit 0 de H-6.a; criterion #2 ("all domain-layer call
  sites migrated") **não** ganha erratum loophole.

---

## Critérios de aceite da Fase

A Fase Multi-venue fecha quando **todos** os critérios abaixo são
verdadeiros simultaneamente:

- [ ] Sub-ondas H-6.a, H-6.b, H-6.c, H-6.d, H-6.e, H-6.f, e Onda
  H-7 fechadas. Cada uma registrou fechamento explícito com
  `make verify` GREEN e RESUMPTION atualizado no commit de
  fechamento.
- [ ] `internal/domain/instrument/` package compliant com
  ADR-0021 spec (Venue, BaseAsset, QuoteAsset, ContractType,
  CanonicalInstrument, validation, `Symbol()` method).
- [ ] Adapters `binances` + `binancef` emitem `CanonicalInstrument`
  via `Normalize`. Pattern detection no `binancef` para discriminar
  `ContractPerpetual` vs `ContractUSDTFutures` via symbol suffix.
- [ ] Adapter Bybit (`bybit`) implementa `ToCanonical`/`FromCanonical`
  per ADR-0021; emite `CanonicalInstrument` via `Normalize`.
- [ ] Todos os domain types migrados de `Symbol string` para
  `Instrument CanonicalInstrument` (ObservationTrade,
  EvidenceCandle, Signal, Decision, Strategy, Risk, Pairing's
  RoundTrip/Leg, Effectiveness's Attribution, etc.).
- [ ] Application layer + actors + samplers consomem
  `CanonicalInstrument`; internal builders migram local
  `symbol string` → `instrument CanonicalInstrument`.
- [ ] ClickHouse migration nova adicionando canonical columns;
  writer dual-writes; analytical client reads canonical-preferred
  com fallback legacy; cutover runbook em
  `docs/operations/runbooks/clickhouse-canonical-migration.md`
  (criação prevista em H-6.d).
- [ ] NATS subject composition: ou migrada para canonical form
  (H-6.e opção (i)), ou deferral indefinido documentado em
  segundo erratum REAL ao ADR-0021 critério #2 (H-6.e opção (ii)).
- [ ] raccoon-cli `check instruments` (H-6.a) e `check venue-parity`
  (H-7) integrados em `make verify`.
- [ ] ADR-0021 promovido a `Accepted` no commit final de H-6.f
  (todos critérios literais satisfeitos).
- [ ] ADR-0022 promovido a `Accepted` no commit final de H-7
  (cross-venue parity provada com Binance + Bybit).
- [ ] PROGRAM-0004 transita para `Closed` na entrega final de
  H-7; entrada Changelog correspondente.

---

## ADRs governantes

| ADR | Escopo | Status no início da Fase | Promovido por |
|-----|--------|--------------------------|----------------|
| 0021 | Canonical instrument & venue model | Proposed (entregue em H-2) | H-6.f (após todos critérios literais) |
| 0022 | Multi-venue normalization policy | Proposed (entregue em H-2) | H-7 |

Nenhuma ADR nova esperada nesta Fase. Se durante as sub-ondas
surgir necessidade arquitetural não coberta, P6 (pause-and-report)
e nova ADR sob `decisions/0026+`.

---

## Riscos

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Cascade de `.Symbol` migration explode (342 refs em 106 files) e produz PRs irrevisáveis | Alto — onda gigante invertida pelo H-6.a pré-flight | Re-escopo em 6 sub-ondas serializadas (H-6.a–H-6.f). Cada sub-onda toca subsystema bounded. PRD declara sub-onda sequencing policy estrita. |
| Sub-ondas paralelas geram merge conflicts no mesmo arquivo | Alto — fragmentação custosa de revisão | Sub-onda sequencing policy estrita: próxima abre apenas após merge da anterior em `main`. |
| ADR-0021 critério #2 ("all domain-layer call sites migrated") sob pressão de aceitar erratum loophole | Alto — perda de disciplina P7 | Decisão firme: critério #2 fica literal. Sem erratum até H-6.f. ADR permanece `Proposed` durante toda a migração; promoção é evento atômico no final. |
| ClickHouse schema migration introduz data ordering risk se bundled com domain refactor | Médio — rollback complexo | Erratum REAL ao critério #4 (split #4a/#4b). #4a é writer-side adapt em H-6.a (zero schema change). #4b é migration dedicada em H-6.d, sequenced after H-6.c e before H-6.f. |
| NATS subject composition decision ad-hoc em H-6.e sem critério explícito | Médio — deferral silencioso | H-6.e protocolo: pause-and-report como primeiro ato. Opção (i) migrar OU (ii) deferral com segundo erratum REAL. Sem decisão silenciosa. |
| `binancef.Normalize` assume todos os símbolos são perpetual; quebra quando delivery dataset entrar | Médio (não imediato; futuro silencioso) | Pattern detection em H-6.a commit 5: regex `_YYMMDD` suffix discrimina `ContractUSDTFutures` vs `ContractPerpetual`. 2 testes cobrindo ambos casos. |
| `check instruments` analyzer over-restrictive — bloqueia adapter legítimo que não declarou policy | Baixo | Declarative via `policies/adapters.toml`; reviewer adiciona explicitamente. Mesmo pattern de `check metrics` de H-5. |
| H-7 Bybit adapter implementa parity-by-default que não generaliza para Coinbase/Hyperliquid futuros | Médio | ADR-0022 define a política; H-7 prova com Bybit. Generalization para outros venues é onda futura per-venue, não em H-7. |
| Promoção de ADR-0021 em H-6.f tentando rush últimos critérios | Médio | P7 absoluto: critérios literais. Reviewer + agente verificam um a um antes do commit que flip Status. |

---

## Referência ao raccoon (sem cópia)

Capacidades validadas no raccoon que informam (sem migrar para)
PROGRAM-0004. Lidas em pré-flight de H-6.a com justificativa
explícita:

- `internal/core/marketdata/domain/instrument_identity.go` +
  `instrument_metadata.go` + `market_type.go` — pattern
  `InstrumentMetadata{VenueSymbol, CanonicalSymbol, BaseAsset,
  QuoteAsset, MarketType}`. Foundry diverges: (a) `CanonicalInstrument`
  é struct rooted em `Base + Quote + Contract` (Venue carrega no
  envelope, não dentro da instrument identity); (b) canonical
  format é `BASE/QUOTE-CONTRACT` (4-way contract discrimination,
  raccoon faz 3-way `SPOT/USD_M/COIN_M` sem distinguir perpetual);
  (c) tipos fortes `BaseAsset`/`QuoteAsset` (raccoon usa `string`
  plano); (d) lowercase enum naming (raccoon uppercase).
- `internal/adapters/exchange/binance/parser.go` — pattern
  `ParseMessage(data, recvAt) (IngestRequest, bool, *Problem)`.
  Foundry mantém signature simpler (`Normalize(raw, symbol) (Event,
  *Problem)`); H-6.a só muda tipo do field interno do output, não
  signature externa.
- `internal/adapters/exchange/common/common.go` — helpers
  `NormalizeSide`, `NormalizeMarketType`. Foundry **não cria**
  `common/` package em H-6.a — single venue family currently;
  premature abstraction. Quando H-7 (Bybit) revelar duplicação
  real, refactor para shared package.
- `docs/adrs/ADR-0011-marketdata-binance-canonical-instrument-and-event-mapping.md`
  — single-venue-Binance scope; foundry ADR-0021 generaliza para
  multi-venue desde início.

Anti-padrão: copiar `instrument_identity.go` literal. Reescrever
no foundry com tipos fortes per ADR-0021 spec.

---

## Evidence

- [`../decisions/0021-canonical-instrument-and-venue-model.md`](../decisions/0021-canonical-instrument-and-venue-model.md)
  — ADR governing instrument model; erratum landed in H-6.a
  commit 0.
- [`../decisions/0022-multi-venue-normalization-policy.md`](../decisions/0022-multi-venue-normalization-policy.md)
  — ADR governing cross-venue parity policy; promotes in H-7.
- [`PROGRAM-0001-foundation.md`](PROGRAM-0001-foundation.md) — Fase
  anterior; entregou ADRs 0021 + 0022 em H-2 como `Proposed`.
- [`PROGRAM-0002-wire.md`](PROGRAM-0002-wire.md) — Fase anterior;
  entregou envelope `instrument` field (string per ADR-0017).
- [`PROGRAM-0003-observability.md`](PROGRAM-0003-observability.md)
  — Fase paralela ativa; metrics instrumentadas que multi-venue
  dashboards consumirão.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — protocolo
  P1–P9 canônico.
- [`../RESUMPTION.md`](../RESUMPTION.md) → "Fase Harvest" — state
  sentinel.

---

## Changelog

- **2026-05-25** — PROGRAM-0004 created. Status `Active`. Ondas
  H-6 (sub-divididas em H-6.a–H-6.f por questão de cascade) +
  H-7 declared. ADRs 0021 + 0022 são governantes (ambas
  `Proposed` no início da Fase, entregues em H-2). Sub-onda
  sequencing policy declarada (mais estrita que P4 top-level).
  Lands como entrega de **commit 1 da H-6.a**, alongside ADR-0021
  erratum (criterion #4 split) em commit 0.
- **2026-05-25** — H-6.a fechada. Entregas: erratum ADR-0021
  (criterion #4 split em #4a/#4b), abertura PROGRAM-0004,
  `internal/domain/instrument/` package (Venue, BaseAsset,
  QuoteAsset, ContractType, CanonicalInstrument com JSON tags +
  21 testes), `ObservationTrade.Symbol string` → `Instrument
  CanonicalInstrument` com método transitório
  `VenueSymbol()` (sunset H-6.f), ambos Binance adapters migrados
  com pattern detection `_\d{6}$` para discriminar delivery
  futures, raccoon-cli `check instruments` analyzer + policy
  `policies/adapters.toml` (allowlist binances/binancef) integrado
  como Step 9 do quality-gate. Transitory-method pattern
  documentado nesta PRD para reuso em sub-ondas seguintes.
  ADR-0021 permanece `Proposed` — promoção é evento atômico em
  H-6.f após critério #2 literal. **Sub-onda H-6.b destravada
  apenas após merge desta em `main`** (sub-onda sequencing
  policy estrita).
