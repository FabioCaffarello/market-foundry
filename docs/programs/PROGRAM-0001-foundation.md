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
| **H-0** | Setup do Harvest | ADR-0016, PROGRAM-0001, CLAUDE.md → "Fase Harvest" (P1–P9; P9 added as H-1 erratum), `.claude/settings.json` (`RACCOON_REFERENCE_PATH`), RESUMPTION marcado |
| **H-1** | Práticas operacionais | TRUTH-MAP, AUTHORITY-MAP, runtime-invariants, SLOs canônicos (todos em formato nativo do foundry; sem cópia do raccoon) |
| **H-2** | Fundação ADR | Sete ADRs (0017–0023) consolidando decisões estruturais herdadas/refinadas da experiência raccoon, sem código de produto novo. Entregues com status `Proposed`; cada ADR carrega seção "Promoção para Accepted" nomeando a onda subsequente que ship o código e flipa o status. |

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

O programa Harvest opera sob o **protocolo P1–P9** documentado em
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
- P9 — Toda alteração via PR; merge em `main` pelo maintainer
  humano; próxima onda abre após merge real, não após completion
  local (P9 adicionado em H-1 como erratum; ver Changelog).

---

## Critérios de aceite da Fase

A Fase Foundation fecha quando **todos** os critérios abaixo são
verdadeiros simultaneamente:

- [ ] Ondas H-0, H-1, H-2 fechadas. Cada onda registrou
  fechamento explícito com `make verify` GREEN e RESUMPTION
  atualizado no commit de fechamento.
- [ ] ADR-0016 publicado com status `Accepted` (entregue em H-0).
- [ ] ADRs 0017–0023 publicados com status `Proposed` (entregues em
  H-2). Cada ADR é promovido a `Accepted` pela onda subsequente que
  ship o código que a implementa, conforme a seção "Promoção para
  Accepted" do próprio ADR. A Fase Foundation **não** depende da
  promoção a `Accepted` desses sete; a promoção é responsabilidade
  da onda implementadora (H-3, H-4, H-6, H-7, H-10).
- [ ] TRUTH-MAP e AUTHORITY-MAP ativos no foundry (H-1), em
  formato nativo (não copiado do raccoon).
- [ ] `runtime-invariants` documentadas com analyzer `raccoon-cli`
  associado quando aplicável (P5).
- [ ] SLOs canônicos do programa documentados em `docs/operations/`
  ou equivalente (H-1).
- [ ] PROGRAM-0001 transita para `Closed` na entrega final de H-2;
  entrada Changelog correspondente.

---

## ADRs esperados na Fase

| ADR | Escopo | Onda | Status ao fechar a Fase Foundation |
|-----|--------|------|-------------------------------------|
| 0016 | Harvest from market-raccoon — protocolo da Fase | H-0 | Accepted |
| 0017 | Event envelope and versioning | H-2 | Proposed (→ Accepted em H-3) |
| 0018 | Protobuf contract layer | H-2 | Proposed (→ Accepted em H-3) |
| 0019 | Deterministic replay and time invariants | H-2 | Proposed (→ Accepted em H-4) |
| 0020 | Sequencing and time normalization | H-2 | Proposed (→ Accepted em H-4) |
| 0021 | Canonical instrument and venue model | H-2 | Proposed (→ Accepted em H-6) |
| 0022 | Multi-venue normalization policy | H-2 | Proposed (→ Accepted em H-7) |
| 0023 | Storage tier roadmap | H-2 | Proposed (parcial: H-9; total: H-10) |

Política operativa de status: cada ADR H-2 é entregue como
`Proposed` na onda H-2, com seção "Promoção para Accepted" listando
critério explícito de promoção. A onda implementadora subsequente
flipa o status para `Accepted` no mesmo commit que ship o código.
Esta política realiza o P7 da Fase Harvest ("Status `Accepted` em
ADR exige código entregue — exceto ADRs de fundação H-2, que
aceitam decisões antes do código que as implementa, desde que o
ADR liste critérios explícitos de quando promover") via o sub-caso
mais conservador: `Proposed` em H-2; `Accepted` quando o código
existe. Se uma onda futura **não** ship o código (ex.: ADR-0023
depende de triggers empíricos), o ADR pode permanecer `Proposed`
indefinidamente — estado válido, não pendência.

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
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" (P1–P9 canônico)
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
- **2026-05-24** — **Erratum integrated with H-1**: protocolo
  ampliado de **P1–P8 → P1–P9**. P9 ("Toda alteração ao foundry
  passa por PR; maintainer humano faz o merge") adicionado ao
  CLAUDE.md → "Fase Harvest" e propagado para ADR-0016, esta
  PRD, RESUMPTION e `decisions/README.md`. O prompt H-1 já
  referenciava P1–P9 como canônico, mas o prompt H-0 entregou
  apenas P1–P8 (5 princípios do prompt expandidos para 8 via
  splitting natural — ver Changelog do ADR-0016). H-1 fecha esse
  drift no mesmo PR que entrega TRUTH-MAP / AUTHORITY /
  runtime-invariants / SLO.
- **2026-05-24** — **H-2 entregue**: sete ADRs de fundação (0017–
  0023) publicados com status `Proposed`. Política operativa de
  status clarificada: cada ADR é promovido a `Accepted` pela onda
  implementadora subsequente, no commit que ship o código; a Fase
  Foundation pode fechar enquanto os sete ADRs permanecem
  `Proposed`. ADR-0023 (storage tier) admite permanência indefinida
  em `Proposed` caso nenhum trigger empírico (T1/T2/T3) dispare.
  Lands no PR de fechamento de H-2 alongside TRUTH-MAP /
  AUTHORITY / RESUMPTION / GLOSSARY updates.
