# Stage S166 — Wave B Pattern Hardening and CI Smoke Integration Report

> Hardened the Wave B expansion pattern based on family-01 learnings and integrated smoke-analytical into CI.

## Resumo Executivo

O S166 cumpre a restrição obrigatória do S162 (C-3): integrar o smoke-analytical ao CI antes da segunda família. Além da integração CI, esta etapa endureceu o padrão da Wave B com base nas fricções reais encontradas na primeira iteração (Signals/RSI) e produziu a versão v2 do padrão de expansão.

**Resultado: PASS — segunda família desbloqueada para expansão.**

## Objetivos e Resultados

| Objetivo | Status |
|---|---|
| Integrar smoke-analytical ao CI | **Feito** — GitHub Actions workflow criado |
| Endurecer padrão Wave B com base em family-01 | **Feito** — v2 do padrão publicada |
| Ajustar checklist e critérios de gate | **Feito** — CI obrigatório, gate com 6 critérios explícitos |
| Documentar lições da primeira iteração | **Feito** — hardening doc com 6 PFs e disposições |
| Não abrir segunda família | **Cumprido** — nenhuma família iniciada |

## O Que Foi Feito

### 1. Integração CI — GitHub Actions

Criado `.github/workflows/ci.yml` com dois jobs:

- **unit-tests**: Executa `make test` em todos os módulos Go do workspace. Rápido (~30s). Gate obrigatório.
- **smoke-analytical**: Depende de `unit-tests`. Sobe stack completa via compose, executa seed, aguarda flush do writer (120s), roda `make smoke-analytical`. Coleta logs em caso de falha.

Triggers: push no `main` e pull requests para `main`.

### 2. Makefile — Target CI

Adicionado `make ci-analytical` que combina `make test` + `make smoke-analytical` em uma única invocação. Útil tanto para CI quanto para validação local completa.

### 3. Padrão Wave B v2

Publicado `wave-b-family-expansion-pattern-v2.md` que substitui o v1 como referência canônica. Mudanças principais:

- CI gate obrigatório (era "recommended").
- Gate review com 5 critérios explícitos (era self-review informal).
- Artefato de documentação expandido para 4 seções obrigatórias.
- Thresholds hard para família 3: smoke parameterization, constructor refactor, naming cleanup.
- Threshold para família 4: avaliação de codegen.

### 4. Hardening Document

Publicado `wave-b-pattern-hardening-after-family-01.md` documentando:

- 6 confirmações do que funcionou no padrão.
- 6 fricções (PF-1 a PF-6) com disposição clara.
- 5 ajustes aplicados ao padrão (A-1 a A-5).
- Débitos carry-forward com prioridade e trigger.

### 5. Checklist Atualizado

Ajustes no `wave-b-family-checklist-schema-writer-reader-gateway-tests-runbook.md`:

- Entry condition: CI passa de "required before second family; recommended before first" para "required — no family merges without green CI".
- Gate review: expandido de 4 para 6+ critérios explícitos incluindo CI pipeline green, schema coherence table, e friction log.

### 6. CI Integration Document

Publicado `ci-smoke-analytical-integration.md` descrevendo:

- Arquitetura do workflow (2 jobs, dependência sequencial).
- O que cada fase do smoke valida (7 fases tabuladas).
- Budget de timeout (~5 min total).
- Como estender para novas famílias.
- Paridade local vs CI.
- 5 regras de manutenção.

## Arquivos Alterados

### Novos

| Arquivo | Descrição |
|---|---|
| `.github/workflows/ci.yml` | GitHub Actions workflow com unit tests + smoke-analytical |
| `docs/architecture/wave-b-pattern-hardening-after-family-01.md` | Lições e ajustes da primeira iteração |
| `docs/architecture/ci-smoke-analytical-integration.md` | Arquitetura e operação da integração CI |
| `docs/architecture/wave-b-family-expansion-pattern-v2.md` | Padrão endurecido v2 |
| `docs/stages/stage-s166-wave-b-pattern-hardening-and-ci-smoke-integration-report.md` | Este relatório |

### Modificados

| Arquivo | Mudança |
|---|---|
| `Makefile` | Adicionado target `ci-analytical` (test + smoke-analytical) |
| `docs/architecture/wave-b-family-checklist-schema-writer-reader-gateway-tests-runbook.md` | CI obrigatório no entry, gate review expandido |

## Ganhos

1. **C-3 cumprida.** O blocker obrigatório do S162 está resolvido. A segunda família pode ser iniciada.
2. **Padrão repetível e verificável.** A expansão não depende mais de processo manual implícito.
3. **Regressions catch automático.** CI roda unit tests + smoke em cada PR, impedindo merge de código que quebre famílias existentes.
4. **Thresholds formalizados.** Família 3 tem compromissos explícitos (smoke extraction, constructor refactor, naming). Não são sugestões — são blocking.
5. **Gate review com critérios claros.** 6 condições explícitas substituem self-review informal.

## Trade-offs

1. **CI é lento (~5 min).** O smoke-analytical precisa do compose stack rodando e do writer flush (120s). Não tem como acelerar significativamente sem mudar o batch interval do writer.
2. **GitHub Actions como plataforma.** O workflow assume GitHub Actions. Se a plataforma mudar, o workflow precisa ser adaptado. O `make ci-analytical` target é portável.
3. **Sem cache de Go modules no CI.** O workflow não configura cache. Pode ser adicionado se o tempo de build se tornar problema.
4. **Smoke script continua monolítico.** A extração da função `validate_analytical_family()` foi adiada para família 3 — o script atual é gerenciável com 2 famílias.

## Débitos Residuais

| Débito | Prioridade | Trigger |
|---|---|---|
| Rename `parseEvidenceKeyParams()` | Low | Family 3 |
| Constructor → struct DI | Medium | Family 3 |
| Smoke test extraction | Medium | Family 3 |
| Codegen evaluation | Low | Family 4 |
| Go module cache no CI | Low | Se build time > 2min |
| Backoff jitter no writer retry | Low | Não agendado |
| Consumer lag visibility | Medium | Não agendado |

## Critérios de Aceite — Verificação

| Critério | Status |
|---|---|
| Padrão da Wave B mais robusto após primeira iteração | **PASS** — v2 publicada com thresholds, gate expandido |
| smoke-analytical integrado ao CI | **PASS** — GitHub Actions workflow criado |
| Segunda família não depende de processo manual implícito | **PASS** — CI gate, checklist, critérios explícitos |
| Expansão futura mais segura e repetível | **PASS** — thresholds formalizados, v2 canônica |
| Base pronta para gate entre iterações | **PASS** — gate review com 6 critérios |

## Guard Rails — Verificação

| Guard Rail | Status |
|---|---|
| Não abrir segunda família | **OK** — nenhuma família iniciada |
| Não expandir funcionalidade | **OK** — zero mudanças em código Go |
| Não inflar com burocracia | **OK** — checklist cresceu em 2 itens no gate, não em categorias inteiras |
| CI não desnecessariamente complexo | **OK** — 2 jobs, sem matrix, sem cache, sem secrets |
| Limites documentados | **OK** — trade-offs e débitos explícitos |

## Preparação Recomendada para S167

O S167 pode seguir dois caminhos:

### Opção A: Segunda Família (Decisions)

Se o objetivo é expandir a camada analítica:

1. Escolher a família Decisions (RSI Oversold) — Layer 2, 14 campos, próxima na cadeia causal.
2. Seguir o padrão v2 exatamente.
3. CI deve passar antes do merge.
4. Documentar as 4 seções obrigatórias.

### Opção B: Terceira Família (com Refactors)

Se duas famílias já foram feitas antes do S167:

1. Família 3 carrega os thresholds obrigatórios: smoke extraction, constructor refactor, naming cleanup.
2. Esses refactors são parte do escopo da iteração, não extras.
3. O gate review inclui verificação dos thresholds.

### Recomendação

Prosseguir com a segunda família (Decisions) no S167. O padrão está endurecido, o CI está ativo, e a família Decisions é a próxima na cadeia de causalidade do pipeline analítico.
