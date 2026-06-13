# PROGRAM-0005 — Fase Insights

**Status:** Closed (2026-06-13 — H-8.c.1 entregue; ver Changelog)
**Date:** 2026-06-13
**Owner:** Repository maintainer (Fabio Caffarello)
**Relates to:**
[`../decisions/0027-insights-decision-support.md`](../decisions/0027-insights-decision-support.md),
[`../decisions/0011-no-oms-expansion-pairing.md`](../decisions/0011-no-oms-expansion-pairing.md),
[`../decisions/0016-harvest-from-market-raccoon.md`](../decisions/0016-harvest-from-market-raccoon.md),
[`PROGRAM-0004-multi-venue.md`](PROGRAM-0004-multi-venue.md),
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest",
[`../RESUMPTION.md`](../RESUMPTION.md)

---

## Objetivo

Portar do `market-raccoon` o **domínio de insights** — análise
descritiva de estrutura de mercado (volume profile / VPVR, TPO,
fusão cross-venue de trades) — como capacidade de primeira classe
do foundry, governada por ADR-0027: **insights são decision-support,
nunca directives**.

Esta Fase entrega insights **trades-only** (o foundry ingere apenas
trade prints; depth/order-book não é ingerido — restrição do
pré-flight de 2026-06-13). O **liquidity heatmap** do raccoon
(depende de `bookdelta`) fica **fora** da Fase até uma Fase futura
expandir o ingest com profundidade.

Insights é a capacidade-âncora do harvest (ADR-0016) ainda não
portada, e **desbloqueia** capacidades futuras: o backpressure
adiado (VPVR overload policy entra aqui, junto com o VPVR) e a
delivery WS (que entrega insights events).

## Contexto de sequenciamento

A Fase abre após H-7 (Multi-venue) fechar. Roda **em paralelo** ao
gate temporal de H-6.f.2 (~2026-08-26, que fecha PROGRAM-0004) —
são independentes: insights consome OBSERVATION/EVIDENCE, não toca
o cutover ClickHouse pendente.

Numeração das ondas: insights usa **sub-ondas H-8.a/b/c** (padrão
H-6), **não** H-9/H-10 — estas estão reservadas para o storage tier
(ADR-0023: "Partial: H-9; full: H-10") e H-11/H-12+ para delivery/
Odin (CLAUDE.md P8). Evitar colisão de numeração é decisão de
abertura desta Fase.

## Escopo (Ondas / sub-ondas)

| Sub-onda | Escopo | Entregas principais |
|----------|--------|---------------------|
| **H-8.a** | Volume Profile (VPVR) + overload policy | Bounded context `internal/domain/insights/` (VolumeProfile price-bucketed buy/sell notional por janela, binning canônico, overload levels L0–L3 com bounded buckets); sampler no derive scope consumindo `ObservationTrade`; stream `INSIGHTS_EVENTS` (single-writer) + publisher; KV `INSIGHTS_VOLUME_PROFILE_LATEST`; read endpoint no gateway; analyzer `check insights` (P5); **promove ADR-0027 → Accepted**. (Persistência ClickHouse **deferida** — ver H-8.a.1 / G12.) |
| **H-8.a.1** | Persistência ClickHouse do VolumeProfile (completa G12) | Tabela `insights_volume_profile` com **Array-columns** (`bucket_price_level/buy_volume/sell_volume Array(String)`, 1 linha/janela — Decisão #6 Opção B) + colunas canônicas base/quote/contract; **extensão do codegen** p/ reconhecer o layer `insights` evidence-style (Decisão #7 Opção A); consumer writer-side `writer-volume-profile` no `INSIGHTS_EVENTS` + mapper `mapVolumeProfileRow`; canário `requireclickhouse`; drift-detect ciente da tabela/consumer. Resolve **G12**. |
| **H-8.b** | TPO profile (Time-Price Opportunity) — compute + KV + read | Market profile / TPO **timeframe-anchored** (Decisão T1 — não session-anchored; o foundry não tem conceito de sessão), trades-only (Decisão T2 — períodos derivados de trades, não candles). A janela de timeframe é subdividida em **períodos** (letras A–X, cap 24 — Decisão T3); cada trade marca seu nível de preço (reusa `BucketLevel`) com a letra do período. `TPOProfile{Periods[], Levels[]}` com `TPOLevel{PriceLevel, Letters, Count}`; POC/value-area/initial-balance/range no snapshot (Decisão T4). Sampler no derive + stream `INSIGHTS_EVENTS` + **KV-latest** `INSIGHTS_TPO_LATEST` + read `GET /insights/tpo/latest` + drift-detect ciente do durable `store-tpo`. (Escopo espelha a H-8.a; persistência ClickHouse **deferida** à H-8.b.1 — ver Decisão T5.) |
| **H-8.b.1** | Persistência ClickHouse do TPO | Tabela `insights_tpo` com **Array-columns** (períodos + níveis paralelos — Decisão T5, padrão H-8.a.1); reusa o layer codegen `insights`; consumer writer-side `writer-tpo`; canário `requireclickhouse`. Espelha a H-8.a.1. |
| **H-8.c** | Cross-venue trade fusion (compute + KV + read) | Fusão de trades multi-venue (Binance + Bybit, pós-H-7) por **janela de timeframe** num `CrossVenueSnapshot` por canonical instrument: linhas por-venue (trade_count, notional, last/high/low) + spread consolidado (max−min dos last prices), mid, venue dominante por notional. **Topologia nova** (C1): a fusão NÃO é um FamilyProcessor per-source (cada SourceScopeActor só vê seu source) — é um **actor único no nível do DeriveSupervisor**, que recebe todo trade e funde por canonical instrument (venue = dimensão fundida; `CanonicalInstrument` exclui venue, ADR-0021). Sampler windowed (C2), key = canonical instrument (C3). Stream `INSIGHTS_EVENTS` + KV `INSIGHTS_CROSS_VENUE_LATEST` + read `GET /insights/cross-venue/latest`. Persistência ClickHouse → **H-8.c.1** (C5; espelha a/a.1, b/b.1). **Fecha a Fase** (com H-8.c.1). |

Capacidades fora desta Fase (registradas para Fases futuras):
**liquidity heatmap** (exige ingestão de depth/`bookdelta` — Fase
própria); microstructure evidence; session-emit policy de delivery.

## Decisões da abertura (owner, 2026-06-13)

Wave prompt auditado em 2026-06-13 (`/tmp/program-0005-insights-wave-prompt.md`);
pré-flight read-only (foundry Explore + raccoon P2 justificado) fundamentou:

- **Decisão #1 (A)** — Nova Fase PROGRAM-0005 + **ADR-0027**
  (insights decision-support read-only); analyzer `check insights`
  per P5.
- **Decisão #2 (A)** — **Volume Profile (VPVR) como H-8.a âncora**,
  trades-only, auto-contida; TPO (H-8.b) e cross-venue (H-8.c)
  serializam depois.
- **Decisão #3 (A)** — **Liquidity heatmap FORA** da Fase inicial
  (liquidity-based, exige ingerir depth); registrado como Fase
  futura. Não entregar heatmap trades-based degradado sob o mesmo
  nome.
- **Decisão #4 (A)** — Cômputo vive como **sampler/FamilyProcessor
  no derive scope** (espelha candle/volume/trade_burst). Binário
  novo proibido por P8.
- **Decisão #5 (incluir)** — O **VPVR overload policy** (L0–L3)
  entra **junto** com o VPVR na H-8.a (o sujeito é o próprio VPVR:
  bounded buckets + degradação). Fecha parcialmente o gap de
  backpressure adiado, **sem** expandir para backpressure genérico
  de pipeline (isso fica para onda própria, pós delivery/insights).

### Decisões da H-8.a.1 (owner, 2026-06-13)

Pré-flight read-only do pipeline codegen→writer→ClickHouse fundamentou:

- **Decisão #6 (Opção B)** — Schema **Array-columns, 1 linha/janela**:
  `bucket_price_level/buy_volume/sell_volume Array(String)`. Preserva
  o contrato **1-evento→1-row** do codegen (a linha tem células Array);
  idiomático p/ analytics ClickHouse (arrayJoin/agregações). Rejeitadas:
  JSON String (analytics por bucket exigiria JSONExtract) e multi-row
  (quebraria o `RowEmitter`).
- **Decisão #7 (Opção A)** — **Estender o codegen** p/ reconhecer o
  layer `insights` no estilo **evidence** (naming family-specific:
  `WriterVolumeProfileConsumer`/`NewVolumeProfileStarter`/
  `NewVolumeProfileConsumer`), com namespace de config próprio
  (`IsInsightsFamilyEnabled`). Mantém a invariante "writer→ClickHouse é
  codegen-governed" (golden self-consistency cobre insights) e
  TPO/cross-venue reusam. Rejeitado hand-write (criaria snowflake fora
  do codegen-integrated).

**Mea culpa (correção do framing H-8.a + do plano da H-8.a.1):** (1) o
closure da H-8.a disse que `buckets[]` "não mapeiam o codegen
1-evento→1-row" — isso vale **só** p/ multi-row; Array-columns mantêm
1-row (Decisão #6). (2) O plano inicial da H-8.a.1 assumiu que o codegen
aceitaria um `volume_profile.yaml` direto; o cross-check de
`codegen/spec.go` revelou que `validLayers` é hardcoded aos 6 layers da
cadeia evidence→execution (`insights` ausente) e que o modelo
family-como-discriminador (signal/decision/…) não encaixa em insights
(event types distintos por family) — o layer `evidence` (family-specific)
é o molde correto. Daí a Decisão #7.

### Decisões de design da H-8.b (agente, pré-flight 2026-06-13)

Planejadas pelo agente sob o mandato de autonomia do owner (loop
PROGRAM-0005), fundamentadas no pré-flight (foundry insights + leitura
read-only justificada P2 do TPO no market-raccoon; nada copiado, P1).
Reversíveis no PR review.

- **T1 — timeframe-anchored, não session-anchored.** O TPO do raccoon é
  ancorado em sessão de exchange (CME_RTH etc.); o foundry **não tem
  conceito de sessão** no domínio. A H-8.b usa a **janela de timeframe**
  (mesma `WindowFor` do VolumeProfile) e subdivide em períodos. TPO
  session-anchored fica para Fase futura (exige infra de sessão/config).
- **T2 — trades-only (períodos derivados de trades).** O raccoon ingere
  candles (high/low por período); o foundry é trades-only. Cada trade
  marca seu nível de preço (`BucketLevel`) com a **letra** do seu
  período. O high/low de cada período é derivado dos trades.
- **T3 — períodos com cap 24 (A–X).** Período = `(ts - openTime) /
  periodDuration`; letra = índice. `periodDuration` é config do sampler
  (derive), validado p/ não exceder 24 períodos por janela. Overload
  (`OverloadLevel`) keyed na contagem de **níveis** (não períodos, que
  são bounded por construção).
- **T4 — POC / VAH / VAL / IB / range no snapshot.** Funções puras sobre
  Levels/Periods: POC = nível com mais letras; value area = ~70% da
  contagem de letras ao redor do POC; initial balance = high/low dos 2
  primeiros períodos; range = high/low globais.
- **T5 — persistência Array-columns (padrão H-8.a.1), em sub-onda
  própria.** Períodos e níveis em colunas `Array` paralelas
  index-aligned na tabela `insights_tpo`; 1-evento→1-row preservado;
  reusa o layer codegen `insights`. `level_letters` como `Array(String)`
  (letras concatenadas por nível, ex. "ACF"). **Escopo: a H-8.b entrega
  compute→publish→KV→read (espelhando a H-8.a); a persistência ClickHouse
  fica na H-8.b.1 (espelhando a H-8.a.1).** Decisão de split tomada em
  implementação p/ manter o PR revisável (P4; precedente H-8.a/a.1) —
  mea culpa: o commit 0 da H-8.b foi otimista ao agrupar CH na mesma onda.

### Decisões de design da H-8.c (agente + owner, pré-flight 2026-06-13)

Pré-flight (foundry multi-venue + leitura read-only justificada P2 do
cross-venue join no raccoon; nada copiado, P1):

- **C1 — fusion actor no nível do DeriveSupervisor, não FamilyProcessor
  per-source.** Cada `SourceScopeActor` só vê seu próprio source; a fusão
  precisa do MESMO canonical instrument através de sources distintos. O
  supervisor já recebe todo trade (antes de rotear per-source) — faneia
  cada trade a um **único** `CrossVenueFusionActor`. (Risco load-bearing
  do pré-flight: tentar como FamilyProcessor per-source falharia em
  silêncio.)
- **C2 — windowed snapshot (owner, escolhido).** Por janela de timeframe
  (mesmo `WindowFor`): acumula por-venue {trade_count, notional,
  last/high/low}; no fechamento emite `CrossVenueSnapshot` com spread
  consolidado/mid/venue dominante. Consistente com VPVR/TPO (vs. o
  per-trade spread monitor do raccoon — rejeitado por volume + divergência
  de windowing).
- **C3 — key = canonical instrument; venues = sources.** `binancef`+`bybitf`
  (perpetual) ou `binances`+`bybits` (spot). Venue é a dimensão fundida;
  `CanonicalInstrument` exclui venue (ADR-0021).
- **C5 — escopo compute→publish→KV→read; ClickHouse → H-8.c.1** (precedente
  a/a.1, b/b.1; mantém o PR focado na topologia nova). A Fase fecha com
  H-8.c.1.

## Princípios aplicáveis (P1–P9)

Ver [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest".
Particularmente:

- **P1/P2** — raccoon `internal/core/insights/` é referência
  consultiva read-only; nada copiado. O foundry diverge: trades-only
  até ter depth, fronteira read-only enforçada por analyzer.
- **P3** — ADR-0027 + este PRD primeiro (este commit); código
  depois.
- **P5** — H-8.a adiciona a invariante "insights são read-only" e
  entrega o analyzer `check insights` que a enforça.
- **P8** — sem binário novo; insights absorvido pelo derive (sampler)
  + store/writer (persistência) + gateway (read).

## Critérios de aceite da Fase

A Fase Insights fecha quando **todos** abaixo forem verdadeiros:

- [x] Sub-ondas H-8.a, H-8.b, H-8.c fechadas (cada uma — e seus splits
  ClickHouse H-8.a.1/b.1/c.1 — com `make verify` GREEN + RESUMPTION
  atualizado no commit de fechamento). PRs #49–#53 + H-8.c.1.
- [x] `internal/domain/insights/` modela VolumeProfile, TPO e
  cross-venue snapshot respeitando ADR-0027 (decision-support,
  trades-only).
- [x] `INSIGHTS_EVENTS` stream single-writer; insights publicados
  e persistidos (CH `insights_volume_profile`/`insights_tpo`/
  `insights_cross_venue` + KV latest); read endpoints no gateway.
- [x] `raccoon-cli check insights` integrado em `make verify`,
  enforçando a fronteira read-only (+ `insights-contracts-drift`).
- [x] ADR-0027 promovido a `Accepted` (na H-8.a).
- [x] PROGRAM-0005 transita para `Closed` na entrega final de
  H-8.c.1; entrada Changelog correspondente.

## ADRs governantes

| ADR | Escopo | Status no início da Fase | Promovido por |
|-----|--------|--------------------------|----------------|
| 0027 | Insights são decision-support read-only | Proposed (abertura) → **Accepted (2026-06-13, H-8.a)** | H-8.a (commit do analyzer `check insights`) |

## Riscos

| Risco | Severidade | Mitigação |
|-------|-----------|-----------|
| Cascade de insights (domínio amplo no raccoon: heatmap/VPVR/TPO/cross-venue/overload) gera onda gigante | Alto (lição H-6) | Split em sub-ondas H-8.a/b/c; escopo trades-only; heatmap fora. Pré-flight por sub-onda reconta sites. |
| Insights derivam para a cadeia de execução (vira OMS) | Alto | ADR-0027 I1/I2 + analyzer `check insights` (fronteira estática). |
| "Heatmap" degradado trades-based confunde o conceito | Médio | Decisão #3: heatmap fora até depth; não entregar sob o mesmo nome sem decisão explícita. |
| Overload scope creep para backpressure genérico de pipeline | Médio | Decisão #5: VPVR overload só (sujeito real); backpressure de pipeline fica para onda própria. |

## Changelog

- **2026-06-13 (closure H-8.c.1 — FECHA A FASE)** — Persistência
  ClickHouse do cross-venue entregue (5 commits): migration 016
  `insights_cross_venue` (Array-columns das venue rows), family codegen
  `cross_venue`, consumer `writer-cross-venue`, `mapCrossVenueRow`,
  canário `requireclickhouse` PASS, drift-detect `writer-cross-venue` +
  `insights_cross_venue`. **PROGRAM-0005 → `Closed`.** A Fase Insights
  entregou 3 capacidades (VPVR, TPO, cross-venue), cada uma
  compute→KV→read + persistência ClickHouse Array-columns, em 6
  sub-ondas (H-8.a/a.1/b/b.1/c/c.1) no loop autônomo; layer codegen
  `insights`, analyzer `check insights` + `insights-contracts-drift`,
  ADR-0027 `Accepted`. Critérios de aceite todos satisfeitos.

- **2026-06-13 (abertura H-8.c.1)** — Persistência ClickHouse do
  cross-venue aberta após H-8.c fechar (PR #53). Espelha a/a.1, b/b.1:
  tabela `insights_cross_venue` com Array-columns das venue rows, family
  codegen `cross_venue`, consumer `writer-cross-venue`, canário
  `requireclickhouse`. **É a última sub-onda — seu merge fecha a Fase
  Insights.** Loop autônomo (self-merge escopado — ADR-0026).

- **2026-06-13 (closure H-8.c)** — Cross-venue trade fusion entregue (6
  commits; escopo compute→publish→KV→read). Domínio `CrossVenueSnapshot`
  (spread/mid/dominant puros) + **topologia nova C1**: `CrossVenueFusion`
  windowed + `CrossVenueFusionActor` único no `DeriveSupervisor` (não
  per-source), que faneia todo trade e funde por canonical instrument;
  store-side `store-cross-venue` → `INSIGHTS_CROSS_VENUE_LATEST`; read
  `GET /insights/cross-venue/latest` (sem source); drift-detect
  `store-cross-venue`; canário integration PASS. Loop autônomo
  (self-merge escopado — ADR-0026). **Próxima: H-8.c.1** (cross-venue
  ClickHouse), a **última** sub-onda — sua entrega transita esta Fase a
  `Closed`.

- **2026-06-13 (abertura H-8.c)** — Cross-venue trade fusion aberta após
  H-8.b.1 fechar (PR #52). Pré-flight expôs a topologia: fusão é
  cross-source, logo NÃO cabe num FamilyProcessor per-source — vive como
  actor único no DeriveSupervisor (C1). Owner escolheu o modelo
  **windowed snapshot** (C2, consistente com VPVR/TPO). Escopo
  compute→publish→KV→read; ClickHouse → H-8.c.1 (C5). Última capacidade da
  Fase Insights. Loop autônomo (self-merge escopado — ADR-0026).

- **2026-06-13 (closure H-8.b.1)** — Persistência ClickHouse do TPO
  entregue (5 commits), espelhando a H-8.a.1. Migration 015
  `insights_tpo` (Array-columns paralelas períodos+níveis, `level_count
  Int32`); family codegen `tpo` (knownAbbreviations `tpo→TPO`);
  consumer `writer-tpo`; `mapTPOProfileRow` + `NewTPOStarter`; canário
  `requireclickhouse` (Array round-trip vs CH vivo) PASS; drift-detect
  `writer-tpo` + `insights_tpo`. Loop autônomo (self-merge escopado —
  ADR-0026). Próxima: H-8.c (cross-venue fusion).

- **2026-06-13 (abertura H-8.b.1)** — Persistência ClickHouse do TPO
  aberta após H-8.b fechar (PR #51). Espelha a H-8.a.1: tabela
  `insights_tpo` com Array-columns paralelas (períodos + níveis), family
  codegen `tpo` no layer `insights`, consumer `writer-tpo`, canário
  `requireclickhouse`. Loop autônomo (self-merge escopado — ADR-0026).
  Próxima após merge: H-8.c (cross-venue fusion).

- **2026-06-13 (closure H-8.b)** — TPO profile entregue (escopo
  compute→publish→KV→read; ClickHouse split p/ H-8.b.1). 6 commits:
  domínio `tpo.go` (POC/value-area/IB/range puros) + sampler
  timeframe-anchored trades-only (períodos A–X, níveis com set de
  letras, overload por nível) + publish path; store-side
  `store-tpo`→`INSIGHTS_TPO_LATEST`; read `GET /insights/tpo/latest`
  (gateway KV-direct, ambos os KV stores); drift-detect `store-tpo`;
  canário integration (publish→consume→KV→read vs NATS vivo) PASS.
  Decisões T1–T5 (agente, pré-flight). Loop autônomo (self-merge
  escopado — ADR-0026). Próxima: H-8.b.1 (TPO ClickHouse), depois
  H-8.c (cross-venue).

- **2026-06-13 (abertura H-8.b)** — TPO profile aberta após H-8.a.1
  fechar (PR #50). Pré-flight (foundry insights reuse + leitura P2 do
  TPO no raccoon) fundamentou as Decisões T1–T5 (timeframe-anchored,
  trades-only, cap 24 períodos, POC/VA/IB, Array-columns). Reusa o
  domínio insights, o layer codegen `insights`, o stream
  `INSIGHTS_EVENTS` e o padrão da H-8.a. Entregue no loop autônomo
  (self-merge escopado — ADR-0026). **Split em implementação**: a H-8.b
  entrega compute→publish→KV→read; a persistência ClickHouse foi para a
  **H-8.b.1** (mea culpa do commit 0; precedente H-8.a/a.1). Próxima
  após merge: H-8.b.1 (TPO ClickHouse), depois H-8.c (cross-venue).

- **2026-06-13 (closure H-8.a.1)** — Persistência ClickHouse do
  VolumeProfile entregue em 6 commits; **G12 resolvido** (write-path).
  Migration 014 `insights_volume_profile` (Array-columns paralelas,
  Decisão #6); codegen estendido p/ o layer `insights` evidence-style
  (Decisão #7 — `validLayers` + `usesFamilySpecificNaming`, family
  `volume_profile`, goldens, integrated.yaml); consumer writer-side
  `writer-volume-profile` + `mapVolumeProfileRow` (1-evento→1-row
  preservado); `IsInsightsFamilyEnabled` (backward-compat); canário
  `requireclickhouse` (Array round-trip vs CH vivo) PASS; drift-detect
  `insights-contracts-drift` (P5). Read de history CH fica fora (sem
  consumidor; KV-latest atende). **Gotcha**: bloco codegen consumer_spec
  deve vir após `DefaultRegistry` p/ o event-stream-coverage do
  contract-audit (profile ci). Entregue no **loop autônomo** (self-merge
  escopado — ADR-0026 errata). `make verify` GREEN (drift-detect 33 /
  123 checks); `--profile ci` GREEN; `raccoon-test` GREEN. Próxima:
  H-8.b (TPO).

- **2026-06-13 (abertura H-8.a.1)** — Persistência ClickHouse do
  VolumeProfile aberta p/ completar G12 (deferido na H-8.a). Owner
  escolheu Opção B (Array-columns, 1 linha/janela — Decisão #6) e
  Opção A (estender o codegen p/ o layer `insights` evidence-style —
  Decisão #7). Pré-flight do pipeline codegen→writer→ClickHouse
  fundamentou; mea culpa do framing 1-row registrado. Esta sub-onda
  roda no **loop autônomo** autorizado pelo owner (self-merge escopado
  — ADR-0026 errata). Próxima após merge: H-8.b (TPO).

- **2026-06-13 (closure H-8.a)** — Volume Profile (VPVR) + overload
  entregue em 7 commits; **ADR-0027 → Accepted**. Domínio
  `insights` (VolumeProfile/binning/overload) + sampler no derive +
  família `INSIGHTS_EVENTS` + KV-latest + read endpoint + analyzer
  `check insights` (gate step 12). Canário integration
  publish→consume→KV→read vs NATS vivo PASS. **Escopo ajustado
  (mea culpa)**: o commit 0 declarou tabela ClickHouse na H-8.a; o
  pré-flight do codegen revelou que os `buckets[]` aninhados não
  mapeiam o codegen 1-evento→1-row — persistência ClickHouse movida
  para sub-onda própria (gap G12 no RESUMPTION); a H-8.a entrega via
  KV-latest, que prova o pipeline end-to-end. Read-path KV-direct
  no gateway (reader livre, ADR-0008). Próxima: H-8.b (TPO) ou a
  persistência ClickHouse — sequenciamento na abertura.

- **2026-06-13 (abertura)** — Fase Insights aberta após H-7 fechar
  (PROGRAM-0004 segue Active aguardando H-6.f.2 no gate temporal).
  Capacidade escolhida pelo owner após reconsiderar backpressure
  (pré-flight mostrou-o acoplado a delivery/insights ausentes;
  insights é o desbloqueador). Decisões #1–#5 registradas acima.
  ADR-0027 criado `Proposed`. Sub-onda âncora H-8.a (Volume Profile
  + overload) destravada.
