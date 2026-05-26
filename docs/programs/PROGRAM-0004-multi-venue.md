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

H-6 é portanto implementada em **8 sub-ondas serializadas**
(H-6.b sub-dividida em b/b'/b'' após pré-flight de H-6.b
descobrir 15 domain types totalizando 174 test files — ver
"Refinamento H-6.b" abaixo):

| Sub-onda | Escopo | Entregas principais |
|----------|--------|---------------------|
| **H-6.a** | Domain root + Binance adapters | `internal/domain/instrument/` package (Venue, BaseAsset, QuoteAsset, ContractType, CanonicalInstrument). Refactor `ObservationTrade.Symbol` → `Instrument`. Adapters `binances` + `binancef` `Normalize` emitem `CanonicalInstrument`. Imediate readers de `ObservationTrade` migram para `.Instrument.Symbol()`. raccoon-cli `check instruments` analyzer. ADR-0021 erratum critério #4 (commit 0). PRD-0004 abertura. **ADR-0021 permanece `Proposed`.** |
| **H-6.b** | Layer 1+2: Evidence + Signal/Decision/Strategy/Risk (7 types) | Cada domain struct (`EvidenceCandle`, `EvidenceTradeBurst`, `EvidenceVolume`, `Signal`, `Decision`, `Strategy`, `RiskAssessment`) migra `Symbol string` → `Instrument CanonicalInstrument` + `VenueSymbol() string` transitório. Os 5 `PartitionKey()` composers (Signal/Decision/Strategy/Risk/Execution) compõem chave KV via `VenueSymbol()` para back-compat com bucket layout existente. raccoon-cli `check instruments` estendido via `policies/domain_types.toml` declarando migration_state per type. **ADR-0021 permanece `Proposed`.** |
| **H-6.b'** ✅ fechada | Layer 3+3': ExecutionIntent + Attribution + AuditLifecycleEntry (3 types) | Execution chain migrada. `ExecutionIntent.Symbol` → `.Instrument` + `VenueSymbol()`; PartitionKey/DeduplicationKey composers atualizados via VenueSymbol. `effectiveness.Attribution.Symbol` → `.Instrument` (derived from `intent.Instrument`). `execution.AuditLifecycleEntry.Symbol` → `.Instrument` (projection de partition key via novo `instrumentFromBinding` per-package em `executionclient`, sunset H-6.f). Triage drop: zero population sites required migration nesta sub-wave (ver Changelog). **ADR-0021 permanece `Proposed`.** |
| **H-6.b''** ✅ fechada | Layer 4: Pairing.Leg/RoundTrip + CrossSessionWindow + Triage population sites | Pairing chain migrada. `pairing.Leg.Symbol` → `.Instrument` + `VenueSymbol()` (M1 invariant via native Go struct equality `entry.Instrument != exit.Instrument` — `CanonicalInstrument` é composto de 3 string-typed components e comparable por construção; estritamente mais forte que symbol equality, pois Contract type também discrimina). `pairing.RoundTrip.Symbol` → `.Instrument` (denormalized from Leg per Decisão #3; invariant `RoundTrip.Instrument == Entry.Instrument == Exit.Instrument` enforced by MatchFIFO construction + M1). `pairing.CrossSessionWindow.Symbol` → `VenueSymbol string` **rename only** per Decisão #2 (b): pre-flight 6 confirmed the field is query metadata, never read by matching algorithm, validated only by `!= ""` — promoting to Instrument would force regression-prone source-string reconstruction at the two construction sites (`get_cross_session_pairing.go:135` + `get_continuity_review.go:178`); same regression-shape as commit 37f8ddd in H-6.b'. New `string_filter` migration_state introduced in commit 1 (analyzer schema extension) records this architectural decision permanently. Triage population site em `triageclient/get_roundtrip_triage.go:74` adopts `.VenueSymbol()` (RoundTripTriageItem.Symbol stays string per S472-style projection) — pulled forward into commit 3 by compile pressure (anonymous embedding forces co-location). Decisão #5β/γ test canaries: new unit test `get_roundtrip_triage_test.go` (β; happy-path projection + zero-Instrument observable canary) + smoke check in `smoke-analytical-e2e.sh` Phase 5 (γ; tri-state PASS/WARN/FAIL — WARN when matched-pair data unavailable in smoke window). **8 commits delivered** (plan declared 9 — see Changelog 2026-05-26 H-6.b'' entry for the consolidation rationale). **ADR-0021 permanece `Proposed`.** |
| **H-6.c** | Application layer + actors + samplers | Query types em `analyticalclient`/`triageclient` decidem caso a caso (query params vs domain values). Sampler/evaluator internal builders migram local `symbol string` → `instrument CanonicalInstrument`. Inclui DTO `analyticalclient.ReviewTransform` e DecisionTriageItem population site downstream. **ADR-0021 permanece `Proposed`.** |
| **H-6.d** | ClickHouse migration + writer back-compat read (#4b) | Nova migration adicionando columns `base`, `quote`, `contract`. Writer dual-writes (legacy `symbol` + canonical fields). Analytical client reads canonical preferred, fallback legacy. Cutover documented em runbook. Implementa **critério #4b** do ADR-0021 erratum. **ADR-0021 permanece `Proposed`.** |
| **H-6.e** | NATS subject composition decision (pause-and-report) | **Primeiro ato**: pause-and-report obrigatório. Decidir: (i) migrar NATS subject/key composition para canonical form (com window de dual-publish/dual-read se necessário), OU (ii) declarar deferral indefinido com **segundo erratum REAL ao critério #2 do ADR-0021** documentando "NATS subjects use Instrument.Symbol() (derived form) as canonical representation for routing; direct CanonicalInstrument fields not used in subjects per [justificativa]". Sem opção #2 sem erratum honesto. **ADR-0021 permanece `Proposed`.** |
| **H-6.f** | Final cleanup + ADR promotion | Remove deprecated fields/types remanescentes. Atualiza TRUTH-MAP, RESUMPTION, GLOSSARY com state final. **Promove ADR-0021 → `Accepted`** apenas se TODOS os critérios (1, 2, 3, 4a, 4b, 5) estão literalmente satisfeitos. P7 absoluto. |

### Refinamento H-6.b (introduzido em H-6.b, pós-pré-flight)

H-6.a declarou H-6.b como "Evidence + Signal + Decision + Strategy
+ Risk" assumindo cascade tractable. Pré-flight obrigatório em
H-6.b revelou:

- **15 domain types** com Symbol field em `internal/domain/`
  (não 5).
- **390 production readers** de `.Symbol` (excluindo
  `instrument/`/`observation/` já migrados).
- **128 production construction sites** com literal `Symbol:`.
- **174 test files** referenciam Symbol — top 10 com 17–37
  literais cada.
- **ExecutionIntent** sozinho tem 199 test sites; **pairing.Leg**
  101; **pairing.RoundTrip** 66. Todos individualmente acima do
  threshold de 25 arquivos declarado em decisão #4 da onda.

Sub-divisão em **H-6.b / H-6.b' / H-6.b''** segue **dependency
order** dos domain types (Layer 1+2 / Layer 3+3' / Layer 4):

| Layer | Types | Sub-onda | Justificativa |
|-------|-------|----------|---------------|
| 1 | Evidence (Candle, TradeBurst, Volume) | H-6.b | Derivam de ObservationTrade (já migrada em H-6.a) |
| 2 | Signal, Decision, Strategy, RiskAssessment | H-6.b | Derivam de Evidence; pattern uniforme `PartitionKey()` |
| 3 | ExecutionIntent | H-6.b' | Deriva de Risk/Strategy/Decision (Layer 2) |
| 3' | Attribution, AuditLifecycleEntry | H-6.b' | Derivam de ExecutionIntent |
| 4 | Pairing.Leg, RoundTrip, CrossSessionWindow | H-6.b'' | Derivam de ExecutionIntent (via Leg construção) |
| 4' | Triage population sites | H-6.b'' (RoundTrip path) / H-6.c (Decision path) | Sites migram com upstream que populam o Symbol |

Vantagens da split por dependency order:

- **Sem buracos semânticos**: cada sub-onda fecha tendo migrado
  todos os types dos quais types subsequentes derivam. Nenhum
  type não-migrado consome type migrado.
- **Cascade balanceado**: H-6.b ~62 prod + 158 test sites;
  H-6.b' ~12 prod + 226 test sites; H-6.b'' ~13 prod + 200
  test sites.
- **Semântica coerente per sub-onda**: derivative analytics em
  H-6.b; execution chain em H-6.b'; pairing chain em H-6.b''.

Estado das domain types que **NÃO migram** em H-6.b/b'/b''
(per pré-flight Decisão #2):

- `triage.DecisionTriageItem.Symbol` e
  `triage.RoundTripTriageItem.Symbol` — projections de display,
  populados via cópia from upstream. Field permanece `string`;
  population sites migram para `.VenueSymbol()` na sub-onda do
  upstream.
- `consistency.ChainSnapshot.{Decision,Strategy,Risk,Execution}Symbol`
  — package documentado (S472) com invariante "primitive types
  only para evitar coupling com domain packages". Não migra;
  consistency checks comparam strings (continuam funcionando).

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

Sub-ondas H-6.a → H-6.b → H-6.b' → H-6.b'' → H-6.c → H-6.d →
H-6.e → H-6.f → H-7 executam **estritamente serial**. Próxima
sub-onda abre branch APENAS após merge da anterior em `main`.

### Sub-wave naming convention

- Documentation/prose: H-6.b, H-6.b', H-6.b'' (apostrophes
  distinguish dependency layers within the wave H-6.b family).
- Branch names / git tags: feat/h-6-b1-…, feat/h-6-b2-…
  (numeric suffix for portability across shells/CI tools where
  apostrophes are unsafe).

Established at H-6.b' (branch feat/h-6-b1-execution-chain);
applies retroactively to existing prose references.

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

- **2026-05-26** — H-6.b'' fechada. Entregas: **pairing chain
  domain migration** — 2 domain types migrados (`pairing.Leg`,
  `pairing.RoundTrip`) com `Instrument CanonicalInstrument` +
  `VenueSymbol()` transitory accessor; 1 query-filter type
  renomeado (`pairing.CrossSessionWindow.Symbol string` →
  `VenueSymbol string`, declarado `string_filter` per Decisão
  #2 (b)); 1 triage projection site pulled-forward
  (`triageclient/get_roundtrip_triage.go:74` adopts
  `review.VenueSymbol()` por compile pressure de anonymous
  embedding). M1 invariant adota native Go struct equality
  (`entry.Instrument != exit.Instrument`) — strictly stronger
  que legacy symbol equality, Contract type also discriminates.
  Analyzer `check-instruments` ganha 3º state `string_filter`
  (commit 1) + 15 unit tests (was 14); 6/6 gate checks PASS.
  Policy entries: `pairing_leg` + `pairing_round_trip` flipam
  `pending` → `migrated`; `cross_session_window` flipa
  `pending` → `string_filter` com inline rationale block
  pointing at pre-flight 6 + commit 37f8ddd precedent.

  **Decisão #5β/γ test canaries**: new unit test
  `get_roundtrip_triage_test.go` (β; happy-path projection +
  zero-Instrument observable canary, ~133 linhas) + inline
  smoke check em `smoke-analytical-e2e.sh` Phase 5 (γ;
  tri-state PASS/WARN/FAIL — WARN preserva honestidade quando
  matched-pair data não está disponível no smoke window;
  ~62 linhas). Década de WARN consecutivo seria dívida de
  cobertura registrada — não bloqueia H-6.b''; considerado em
  H-6.f cleanup ou onda dedicada de smoke hardening.

  **8 commits delivered (plan declared 9)**: consolidação por
  compile pressure documentada nos commits 3 e 8 — commits 3
  (RoundTrip migration) e 5 (triage production line update) do
  plano original consolidados em commit 3, porque
  RoundTripReviewItem embute `pairing.RoundTrip` anonimamente e
  remover `RoundTrip.Symbol` quebra `review.Symbol` access
  imediatamente. Commit 5 do plano permaneceu como test-only
  (Decisão #5β canary coverage). Per H-6.b' precedent (PR #28):
  pull-forward by compile pressure é documentado no commit
  afetado, não escondido em renumbering. Branch
  `feat/h-6-b1-execution-chain` carrega ambas H-6.b' e
  H-6.b'' empilhadas — H-6.b'' iniciou antes do merge de
  H-6.b' em `main` (deviation observada de P4/P9 sub-onda
  sequencing policy; combinado em entrega única para o
  maintainer).

  **H-6.f scope revision (post-pré-flight 6 de H-6.b'')**:
  pré-flight 6 descobriu que o débito real de H-6.f é maior
  que "remove transitory methods". Scope revisado:
  1. Audit e remoção dos 6 helpers `instrumentFromBinding` em
     `application/{signal,decision,strategy,risk,execution,
     executionclient}/` — todos hardcoded para `binances`/
     `binancef` + `USDT` quote, retornam zero silenciosamente
     para qualquer outro input.
  2. Audit `reconstructInstrumentFromLegacy` em
     `internal/adapters/clickhouse/candle_reader.go:150` —
     retorna error mas o error é descartado em 11 chamadas em
     `composite_reader.go` e `execution_reader.go`. Either
     propagate errors or replace com Instrument pass-through
     onde upstream já carrega.
  3. Migrate callers para receber Instrument diretamente de
     upstream (pattern estabelecido por
     `NewPaperOrderEvaluatorForInstrument` em H-6.b' commit
     37f8ddd).
  4. Remover métodos `VenueSymbol()` apenas após todos os
     callers migrarem.
  5. Promover ADR-0021 a `Accepted` quando critério #2 estiver
     literalmente satisfeito: zero source-string-based
     instrument reconstruction em production code.

  Esta revisão garante que H-6.f não é "remove transitory
  methods" mas o trabalho semântico real. ADR-0021 permanece
  `Proposed` — promoção continua sendo evento atômico em
  H-6.f. Sub-onda H-6.c destravada após merge da combinada
  H-6.b' + H-6.b'' em `main`.

- **2026-05-26** — H-6.b' fechada. Entregas: três domain types
  da execution chain migrados `Symbol string` → `Instrument
  CanonicalInstrument` + `VenueSymbol()` transitory accessor:
  `execution.ExecutionIntent` (production cascade ~50 sites em
  actors/adapters/application — KV publisher dedup key e
  PartitionKey composers atualizados via VenueSymbol),
  `effectiveness.Attribution` (derived from `intent.Instrument`),
  `execution.AuditLifecycleEntry` (reconstrução de
  `LifecycleEntry` DTO via novo per-package
  `instrumentFromBinding` em `executionclient`, sunset H-6.f).
  Analyzer `check-instruments` flipou as 3 entries em
  `policies/domain_types.toml` de `pending` → `migrated`; 6/6
  checks PASS, total gate 102/102. Sub-wave naming convention
  documentada nesta PRD (acima). **Triage drop closure note**:
  Triage population sites verificados durante pré-flight; zero
  sites required migration nesta sub-wave —
  `DecisionTriageItem` é buffered pelo ReviewTransform DTO
  (application-layer; domain→DTO boundary migrado em H-6.b
  commit 4; DTO→Triage permanece string-to-string até H-6.c
  migrar ReviewTransform); `ExecutionTriageItem` não existe no
  codebase; `RoundTripTriageItem` corretamente deferido para
  H-6.b'' (upstream RoundTrip migra lá). 5 commits (ExecutionIntent
  atomic, Attribution, AuditLifecycleEntry, policy flip, docs
  closure este commit). ADR-0021 permanece `Proposed` —
  critério #2 ainda não literalmente satisfeito (restam Pairing
  chain types em H-6.b''). **Sub-onda H-6.b'' destravada apenas
  após merge desta em `main`** (sub-onda sequencing policy
  estrita).
- **2026-05-26** — H-6.b refinement: pré-flight obrigatório
  descobriu 15 domain types totalizando 174 test files,
  forçando fragmentação além do plano de H-6.a (que assumia 5
  types). Sub-divisão por dependency order em **H-6.b** (Layer
  1+2: Evidence + Signal/Decision/Strategy/Risk, 7 types),
  **H-6.b'** (Layer 3+3': ExecutionIntent + Attribution +
  AuditLifecycleEntry), e **H-6.b''** (Layer 4: Pairing chain
  + Triage RoundTrip population site). Total sub-ondas
  internas H-6 sobe de 6 para 8. Decisão fundamentada em
  numbers reais do pré-flight; análise alternativa em "Refinamento
  H-6.b" acima. Lands como **commit 1 da PR de H-6.b**, antes
  de qualquer commit de código (P3 absoluto).
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
