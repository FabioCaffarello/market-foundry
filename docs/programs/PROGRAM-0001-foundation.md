# PROGRAM-0001 — Harvest Foundation

**Status:** Active
**Date:** 2026-05-24
**Owner:** Repository maintainer (Fabio Caffarello)
**Relates to:**
[`../decisions/0016-harvest-from-market-raccoon.md`](../decisions/0016-harvest-from-market-raccoon.md),
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest",
[`../RESUMPTION.md`](../RESUMPTION.md)

---

## Objetivo

Estabelecer no `market-foundry` a moldura governante do programa
Harvest: protocolo, práticas operacionais e ADRs de fundação. Ao
final desta Fase, qualquer agente Claude futuro entrando no foundry
encontra tudo o que precisa para entender o que é o Harvest, por que
ele existe, e quais são as regras — sem ter que reconstruir contexto
a partir de mensagens soltas.

Esta Fase é **exclusivamente documental e protocolar**. Nenhum
código de produto novo é entregue em H-0/H-1/H-2.

---

## Escopo (Ondas)

| Onda | Escopo resumido | Entregas principais |
|------|------------------|---------------------|
| **H-0** | Setup do Harvest | ADR-0016, PROGRAM-0001, CLAUDE.md → "Fase Harvest" (P1–P8), `.claude/settings.json` (`RACCOON_REFERENCE_PATH`), RESUMPTION marcado |
| **H-1** | Práticas operacionais | TRUTH-MAP, AUTHORITY-MAP, runtime-invariants, SLOs canônicos (todos em formato nativo do foundry; sem cópia do raccoon) |
| **H-2** | Fundação ADR | Sete ADRs (0017–0023) consolidando decisões estruturais herdadas/refinadas da experiência raccoon, sem código de produto novo |

Ondas posteriores (H-3 e além) portam capacidades específicas
(insights, replay, multi-venue, proto layer, observability,
TimescaleDB em H-10, cliente Odin em H-12+) e são escopadas no
momento em que cada uma abre. Esta PRD não congela esse roadmap;
ela congela o protocolo sob o qual ele opera.

---

## Não-Escopo

- **Nenhum código de produto novo** durante H-0/H-1/H-2. Toda
  entrega é documental ou de configuração.
- **Nenhuma cópia de arquivo do raccoon.** P1/P2 da Fase Harvest;
  capacidades são reescritas, nunca importadas.
- **Cliente Odin / WASM.** Mapeado para H-12+ dentro de `client/`
  no próprio foundry. Até lá, nada de cliente é antecipado (P8).
- **TimescaleDB.** Decisão futura, provável H-10. Não decidir
  agora; mapear.
- **5 binários extras do raccoon** (`portfolio`, `strategist`,
  `signals-separado`, `executor`, `validator`) — **permanente não-
  escopo.** O foundry consolida em 8 binários (ver
  [`../ARCHITECTURE.md`](../ARCHITECTURE.md) → "Binary boundaries");
  capacidades equivalentes serão adicionadas dentro desses owners
  existentes, não como binários novos.

---

## Princípios governantes

O programa Harvest opera sob o **protocolo P1–P8** documentado em
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest". A versão
canônica vive lá para ser carregada em toda sessão Claude; esta
PRD apenas referencia.

Resumo de uma linha por princípio (fonte canônica é CLAUDE.md):

- P1 — Foundry ground truth; raccoon referência consultiva.
- P2 — Raccoon path estritamente read-only.
- P3 — Capacidade portada passa por documento primeiro.
- P4 — Uma onda por vez; fechamento explícito antes da próxima.
- P5 — Cada onda evolui `raccoon-cli` quando adiciona invariante.
- P6 — Pause-and-report ativo durante a onda inteira.
- P7 — Sem perda de disciplina documental.
- P8 — Cliente Odin mapeado, não esquecido (H-12+).

---

## Critérios de aceite da Fase

A Fase Foundation fecha quando **todos** os critérios abaixo são
verdadeiros simultaneamente:

- [ ] Ondas H-0, H-1, H-2 fechadas. Cada onda registrou
  fechamento explícito com `make verify` GREEN e RESUMPTION
  atualizado no commit de fechamento.
- [ ] ADRs 0016 + 0017 + 0018 + 0019 + 0020 + 0021 + 0022 + 0023
  publicados com status `Accepted` (oito ADRs no total).
- [ ] TRUTH-MAP e AUTHORITY-MAP ativos no foundry (H-1), em
  formato nativo (não copiado do raccoon).
- [ ] `runtime-invariants` documentadas com analyzer `raccoon-cli`
  associado quando aplicável (P5).
- [ ] SLOs canônicos do programa documentados em `docs/operations/`
  ou equivalente (H-1).
- [ ] PROGRAM-0001 transita para `Closed` na entrega final; entrada
  Changelog correspondente.

---

## ADRs esperados na Fase

| ADR | Escopo | Onda | Status esperado ao fechar a Fase |
|-----|--------|------|-----------------------------------|
| 0016 | Harvest from market-raccoon — protocolo da Fase | H-0 | Accepted |
| 0017–0023 | Sete ADRs de fundação consolidando decisões estruturais (escopo específico definido em H-2 antes da abertura) | H-2 | Accepted |

Critério de promoção `Draft → Accepted` para ADRs de fundação H-2:
permitido aceitar antes do código que os implementa, **desde que** o
ADR liste o critério de promoção subsequente (e.g., "promove para
implementado quando onda H-N entregar o componente X com analyzer Y
verde") — P7 da Fase Harvest.

---

## Riscos

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Browsing do raccoon vira copying silencioso | Alto — corrompe P1/P2/P3 | P2 exige justificativa antes de cada leitura; revisor de PR confere `.claude/settings.json` e procura `cp -`/`rsync` em diffs |
| ADRs de fundação (0017–0023) consolidam decisões sem código que as implemente | Médio | P7 — `Accepted` antes do código é permitido **apenas** para H-2, com critério explícito de promoção declarado no próprio ADR |
| Ondas paralelizam por pressão de cronograma | Médio | P4 — uma onda por vez; fechamento explícito antes da próxima. Custo de fechamento serializado é baixo; custo de conflito documental é alto |
| TRUTH-MAP/AUTHORITY-MAP do foundry divergem do raccoon (intencional) | Baixo | Foundry é ground truth (P1); os mapas do foundry são nativos, não importados. Divergência deliberada é esperada |
| Lista de ADRs 0017–0023 não cobre todas as decisões estruturais necessárias | Médio | Escopo específico decidido **dentro de H-2**, com pause-and-report (P6) se faltar slot |

---

## Referência ao raccoon (sem cópia)

Capacidades maduras do raccoon que inspiram (mas não migram para) o
programa Harvest. Cada uma é reescrita no foundry em onda
apropriada, respeitando layer sovereignty + single-writer
invariant + configctl authority:

- Insights (heatmap, volume profile, candle/stats aggregation).
- Replay determinístico + golden tests.
- Multi-venue normalization.
- Protobuf contract layer.
- Observability surface (SLOs, metrics policy, runbooks).
- Sequencing & time normalization.
- Backpressure / overload policies.

Esta lista é informativa; a sequência exata em que cada capacidade é
portada é decidida em ondas posteriores (H-3+), não nesta PRD.

---

## Evidence

- [`../decisions/0016-harvest-from-market-raccoon.md`](../decisions/0016-harvest-from-market-raccoon.md)
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" (P1–P8 canônico)
- [`../RESUMPTION.md`](../RESUMPTION.md) → "Fase Harvest"
- [`../programs/README.md`](README.md) — convenção PRD
- [`../../.claude/settings.json`](../../.claude/settings.json) →
  `RACCOON_REFERENCE_PATH`

---

## Changelog

- **2026-05-24** — PROGRAM-0001 created. Status `Active`. Ondas
  H-0/H-1/H-2 declared. ADRs 0016 + 0017–0023 expected. Lands as
  the H-0 closure artifact alongside ADR-0016 and CLAUDE.md →
  "Fase Harvest".
