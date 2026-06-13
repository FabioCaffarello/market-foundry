# PROGRAM-0005 — Fase Insights

**Status:** Active
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
| **H-8.b** | TPO profile (Time-Price Opportunity) | Market profile / TPO por janela de sessão, trades-only. Reutiliza binning + stream + persistência da H-8.a. |
| **H-8.c** | Cross-venue trade fusion | Fusão de trades multi-venue (Binance + Bybit, pós-H-7) em snapshots cross-venue; encaixe direto na superfície multi-venue. |

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

- [ ] Sub-ondas H-8.a, H-8.b, H-8.c fechadas (cada uma com
  `make verify` GREEN + RESUMPTION atualizado no commit de
  fechamento).
- [ ] `internal/domain/insights/` modela VolumeProfile, TPO e
  cross-venue snapshot respeitando ADR-0027 (decision-support,
  trades-only).
- [ ] `INSIGHTS_EVENTS` stream single-writer; insights publicados
  e persistidos (CH + KV); read endpoints no gateway.
- [ ] `raccoon-cli check insights` integrado em `make verify`,
  enforçando a fronteira read-only.
- [ ] ADR-0027 promovido a `Accepted` (na H-8.a).
- [ ] PROGRAM-0005 transita para `Closed` na entrega final de
  H-8.c; entrada Changelog correspondente.

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
