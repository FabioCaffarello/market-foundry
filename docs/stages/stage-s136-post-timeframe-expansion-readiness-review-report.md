# Stage S136 — Post-Timeframe-Expansion Readiness Review

> **Status**: Complete
> **Date**: 2026-03-19
> **Scope**: Formal readiness review after TC-01 wave (S131–S135)
> **Depends on**: S131, S132, S133, S134, S135

---

## 1. Resumo Executivo

O Stage S136 encerra a onda de expansão temporal (TC-01) com uma avaliação formal de prontidão do Market Foundry. A revisão é baseada em evidência concreta produzida pelos estágios S131–S135, não em projeções ou desejos de refatoração.

**Conclusão principal**: A expansão de 2→4 timeframes provou a tese arquitetural de scaling config-driven. O Foundry está robusto para o escopo atual. A próxima onda deve ser orientada a produto, não a mais expansão temporal.

---

## 2. Avaliação Formal Pós-Expansão Temporal

### 2.1 A expansão provou robustez?

**Sim.** Zero alterações de código Go foram necessárias. Uma única mudança em `derive.jsonc` propagou corretamente por todos os 6 estágios do pipeline (evidence → signal → decision → strategy → risk → execution). Crescimento linear confirmado:

- Atores: 2× (6→12 evidence samplers/symbol)
- Subjects NATS: 2× (~32→~64)
- Chaves KV: 2× em todos os buckets
- Carga de escrita: <30% de aumento (TFs longos escrevem menos)

### 2.2 A expansão provou escalabilidade saudável?

**Sim, com qualificação.** Seis problemas antecipados não se materializaram (pressão NATS, latência de fan-out, contensão KV, colisão de dedup, interferência cross-TF, acumulação de memória). Porém, estado de janela in-memory é um gate hard para TFs de 4h+.

### 2.3 Critérios de sucesso

| Critério | Status |
|----------|--------|
| M1–M6 (Tier 1: ativação) | PASS |
| M9, M11–M13 (Tier 2: materialização) | PASS |
| M10 (Tier 3: finalização 3600s) | PASS |
| M7, M8 (convergência RSI 900s/3600s) | Deferred (requer 6–15h de runtime) |

**11/13 critérios obrigatórios aprovados. 2 deferidos por restrição física, não por falha arquitetural.**

---

## 3. Ganhos e Trade-offs

### Ganhos Concretos

| # | Ganho | Valor |
|---|-------|-------|
| G1 | Prova de scaling config-driven | Permanente — tese arquitetural validada |
| G2 | Crescimento linear confirmado | Elimina medo de explosão combinatória |
| G3 | 6 riscos antecipados descartados | Reduz superfície de risco para expansões futuras |
| G4 | Validação de config no startup | Previne classe inteira de bugs de misconfiguration |
| G5 | Runbook de recuperação operacional | Operadores informados sobre perda de dados por TF |
| G6 | Cobertura de testes completa em 4 TFs | Segurança de regressão para qualquer mudança futura |

### Trade-offs Aceitos

| # | Trade-off | Custo | Benefício |
|---|-----------|-------|-----------|
| T1 | Lista global de TFs | Sem customização por symbol | Config simples, spawn simples |
| T2 | Sem snapshots parciais | Sem sinal parcial em janelas longas | Publisher simples, sem conceito "in-progress" |
| T3 | Estado in-memory only | Até 60min de perda no crash | Sem WAL, sem overhead de persistência |
| T4 | Tracking agregado only | Sem visibilidade per-TF | Menos métricas, tracker simples |
| T5 | M7/M8 deferidos | Sem prova de RSI em janelas longas | TC-01 concluído sem runs de 15h |

---

## 4. Débitos Abertos e Refactors que NÃO Valem o Custo Agora

### Débitos Abertos

| ID | Débito | Prioridade | Trigger |
|----|--------|-----------|---------|
| D1 | Persistência de estado de janela | P2 | TC-02 com TFs 4h+ |
| D2 | Detecção de idle per-TF | P2 | TFs 4h+ em produção |
| D3 | Prova de convergência RSI 900s/3600s | P3 | Viabilidade operacional |
| D4 | Config de TFs per-binding | P3 | Necessidade heterogênea por symbol |
| D5 | Observabilidade da query surface | P3 | Consumidores externos |

### Refactors que NÃO Valem o Custo Agora

| Item | Por que não |
|------|-----------|
| D1 (state persistence) antes de TC-02 | Resolve problema que só existe com TFs 4h+; premature engineering |
| Endpoint de listagem de TFs (F-07) | Sem consumidores externos; config é source of truth |
| Disambiguação de null response (F-08) | Só afeta operadores não-experts que não existem ainda |
| View agregada no gateway (F-19) | Setup single-symbol; prematura até 5+ symbols |
| Framework genérico de evaluators | Zero evidência de necessidade; RSI pattern é suficiente |
| Sistema de plugins | Arquitetura não demanda; code-driven é adequado |
| Per-binding timeframes agora | Todos symbols usam mesmos TFs; complexidade sem benefício |

### Refactors que Valeram o Custo (S135)

| Item | Custo | Payoff |
|------|-------|--------|
| R-01: ValidateTimeframes() | ~25 linhas prod + ~35 linhas teste | Previne misconfiguration permanentemente |
| R-02: Recovery runbook | Documentação only | Operadores informados, zero custo de código |

---

## 5. Recomendação Objetiva para a Próxima Onda

### Decisão: Product Wave

A infraestrutura está provada. O pipeline processa trades → candles → signals → decisions → strategies → risk → execution across 4 timeframes. A próxima prova significativa é: **isso produz output útil para um caso de uso real?**

### Por que não as outras opções

| Opção | Veredicto | Razão |
|-------|-----------|-------|
| TC-02 (mais TFs) | Esperar | Tese já provada; D1 é hard gate; sem demanda de produto |
| Nova família/capacidade | Secundária | Boa se escoped tightly (ex: MACD signal) |
| Hardening específico | Incorporar | Não é onda própria; items individuais conforme necessário |
| **Product wave** | **Primária** | **Infraestrutura provada; hora de provar valor** |

### Cenários candidatos para product wave (escolher um)

1. **Paper trading loop**: Executar paper orders baseados em mean_reversion_entry, rastrear P&L
2. **Alert/notification output**: Surfacer sinais de strategy como notificações acionáveis
3. **Backtest harness**: Replay histórico pelo pipeline, medir performance da strategy

### Critérios de entrada para a product wave

- [ ] Definir cenário de produto específico
- [ ] Definir critérios de sucesso mensuráveis (como M1–M13 de TC-01)
- [ ] Definir exit condition explícita
- [ ] Escopo máximo de 2 semanas

### Condições para revisitar TC-02

- Demanda real de produto por TFs 4h+ ou daily
- D1 (state persistence) resolvido
- M7/M8 (RSI convergence) verificados

---

## 6. Entregáveis Produzidos

| # | Documento | Caminho |
|---|-----------|---------|
| 1 | Readiness Review | `docs/architecture/post-timeframe-coverage-01-readiness-review.md` |
| 2 | Gains, Trade-offs & Open Debts | `docs/architecture/timeframe-coverage-01-gains-tradeoffs-and-open-debts.md` |
| 3 | Next Wave Recommendations | `docs/architecture/next-wave-recommendations-after-timeframe-coverage-01.md` |
| 4 | Stage Report (este documento) | `docs/stages/stage-s136-post-timeframe-expansion-readiness-review-report.md` |

---

## 7. Critérios de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Review é específica, honesta e baseada em evidência real | PASS — baseada em 11 docs de arquitetura + código + testes |
| Ganhos, atritos e trade-offs ficam claros | PASS — tabelados com prioridade e triggers |
| Foundry ganha melhor critério para próxima onda | PASS — framework de decisão com 4 perguntas-chave |
| Decisão deixa de depender de refatoração por impulso | PASS — cada refactor requer trigger explícito |
| Etapa fecha a onda com clareza estratégica | PASS — recomendação product wave com critérios de entrada |

---

## 8. Conclusão

TC-01 fechou com sucesso. A expansão temporal provou que a arquitetura escala por config, sem código novo, com crescimento linear e sem explosão de complexidade. Os dois refactors executados (validação + runbook) tiveram payoff real com custo mínimo. Cinco débitos permanecem abertos com triggers explícitos — nenhum requer ação imediata.

**O Foundry provou que pode escalar. Agora precisa provar que pode entregar valor.**

A próxima onda deve ser orientada a produto: exercitar o pipeline completo com um cenário concreto e mensurável. Expansão temporal adicional, hardening especulativo e abstrações horizontais ficam explicitamente fora de escopo até que evidência de produto demande.
