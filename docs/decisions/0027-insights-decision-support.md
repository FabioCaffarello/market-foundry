# ADR 0027: Insights são decision-support read-only

## Status

Proposed. Promovido a `Accepted` na **Onda H-8.a** (PROGRAM-0005),
no commit que entrega o bounded context `internal/domain/insights/`
**e** o analyzer `check insights` que enforça suas invariantes
estaticamente. Não é ADR de fundação H-2; portanto permanece
`Proposed` até o código existir (P7). Critérios de promoção em
"Promoção para Accepted" abaixo.

## Date

2026-06-13.

## Context

O harvest backlog (ADR-0016) lista **"Insights domain (heatmap,
volume profile, candle/stats aggregation)"** como capacidade
madura do `market-raccoon` ainda não portada. Aberta a Fase
Insights (PROGRAM-0005) após o fechamento da Fase Multi-venue
(H-7), os insights são a primeira capacidade descritiva/analítica
do foundry além de evidence.

Insights são **decision-support**: explicam *o que foi detectado e
por que pode importar*. Eles são categoricamente diferentes da
cadeia já existente `signal → decision → strategy → risk →
execution`, que culmina em **directives** (intents de ordem). Um
volume profile, um TPO, uma fusão cross-venue de trades — todos
descrevem a estrutura do mercado; nenhum diz "compre" ou "venda".

Sem uma fronteira **declarada e enforçada**, "insights" derivaria
naturalmente para a cadeia de execução: alguém ligaria um insight
a um signal, o signal a uma decision, e em seis meses o domínio
analítico estaria acoplado ao caminho de ordem — recriando a
expansão de OMS que o foundry recusa.

Precedente direto: **ADR-0011** (no-OMS; effectiveness e pairing
são read-only, não publicam para execução, não criam OMS). Insights
é a generalização daquela postura para um domínio descritivo de
primeira classe.

Restrição factual do pré-flight (2026-06-13): o foundry **ingere
apenas trades** (aggTrade/publicTrade); order-book/depth **não é
ingerido**. Capacidades de insight que dependem de profundidade
(liquidity heatmap) não são computáveis hoje; as que dependem só de
trade prints (volume profile/VPVR, TPO, cross-venue trade fusion)
são.

## Decision

market-foundry adopta o bounded context `internal/domain/insights/`
governado por quatro invariantes:

### I1 — Insights nunca emitem directives

Um insight é decision-support: carrega tipo, evidence, janela,
instrument e (quando aplicável) confiança ∈ [0,1] e condições de
invalidação. **Nunca** carrega side/quantity/intent. Não há
caminho de um insight para uma ordem.

### I2 — Insights são read-only sobre o pipeline

Insights **consomem** `OBSERVATION_EVENTS` / `EVIDENCE_EVENTS` e
**publicam** em `INSIGHTS_EVENTS` (stream próprio, single-writer
per ADR-0008). Insights **não** publicam em `STRATEGY_EVENTS`,
`EXECUTION_EVENTS` nem em qualquer stream da cadeia de directive.
O package `internal/domain/insights/` **não importa**
`internal/domain/{strategy,execution,risk,decision}` nem os ports
de execução — fronteira verificada estaticamente.

### I3 — Forma canônica do insight

Confidence ∈ [0,1] quando presente; cada insight referencia um
`CanonicalInstrument` (ADR-0021) e uma janela. Projeções
quantitativas (volume profile) são price-bucketed de forma
determinística (binning canônico).

### I4 — Trades-only até o ingest expandir

Insights computam de trade prints. Capacidades que exigem
order-book/depth (liquidity heatmap) ficam **fora** até uma Fase
futura expandir o ingest com uma fonte de profundidade. Esta ADR
não autoriza ingestão de depth.

### Enforcement (P5)

A Onda que adiciona estas invariantes entrega o analyzer
`raccoon-cli check insights`, que valida estaticamente: (a) o
domínio `insights` não importa os packages da cadeia de directive;
(b) o publisher de insights publica apenas em `INSIGHTS_EVENTS`.
Sem isso, I1/I2 seriam intenção sem enforcement — o modo de falha
que ADR-0004 existe para prevenir.

## Consequences

### Positive

- Fronteira descritivo-vs-directive explícita e enforçada; insights
  cresce (TPO, cross-venue, mais tarde liquidity heatmap) sem risco
  de virar OMS.
- Alinhamento com ADR-0011 — uma postura única "domínios analíticos
  são read-only" cobre effectiveness, pairing e insights.
- `INSIGHTS_EVENTS` single-writer mantém o invariante mais
  importante do mesh (ADR-0008).

### Negative

- Insights não fecham o loop de trading — por design. Quem quiser
  agir sobre um insight fá-lo fora do foundry (ou via uma futura
  decisão arquitetural explícita, não por acoplamento acidental).
- Liquidity heatmap fica gated até o ingest de depth — uma das
  capacidades mais vistosas do raccoon não chega na Fase inicial.

## Alternatives considered

- **(A) Insights como mais um tipo de signal na cadeia
  derive→signal→…→execution.** Rejeitado: colapsa descritivo com
  directive; insights virariam input de execução, criando o
  acoplamento e o risco de OMS que ADR-0011 recusa.
- **(B) Sem ADR, só código.** Rejeitado: P3 (documento primeiro) e
  P7 (invariante sem enforcement é intenção); a fronteira read-only
  precisa de analyzer.
- **(C) Heatmap trades-based degradado para entregar "um heatmap"
  sem depth.** Rejeitado nesta ADR: semanticamente diverge do
  liquidity heatmap do raccoon; entregar sob o mesmo nome
  confundiria o conceito. Uma Fase futura decide explicitamente.

## Promoção para Accepted

Promovido na Onda H-8.a quando, no mesmo PR:

1. `internal/domain/insights/` existe com o modelo de Volume
   Profile respeitando I1/I3/I4.
2. O stream `INSIGHTS_EVENTS` é single-writer (I2) e o package
   insights não importa a cadeia de directive.
3. `raccoon-cli check insights` ship e roda em `make verify` via
   `quality-gate`, validando I2 estaticamente.
4. `make verify` GREEN; RESUMPTION e PROGRAM-0005 atualizados.

## References

- ADR [0011](0011-no-oms-expansion-pairing.md) — no-OMS; o
  precedente read-only que esta ADR generaliza.
- ADR [0016](0016-harvest-from-market-raccoon.md) — backlog do
  harvest; insights é item explícito.
- ADR [0008](0008-single-writer-invariant.md) — `INSIGHTS_EVENTS`
  tem exatamente um writer.
- ADR [0009](0009-subject-taxonomy.md) — subjects de insights
  seguem a taxonomia canônica.
- ADR [0021](0021-canonical-instrument-and-venue-model.md) —
  insights referenciam `CanonicalInstrument`.
- ADR [0004](0004-raccoon-cli-static-enforcement.md) — framework do
  analyzer `check insights`.
- [PROGRAM-0005](../programs/PROGRAM-0005-insights.md) — a Fase que
  implementa esta decisão.
- raccoon `internal/core/insights/` — inspiração (read-only, P2);
  o foundry diverge por (a) escopar trades-only até ter depth, (b)
  enforçar a fronteira read-only via analyzer dedicado.
