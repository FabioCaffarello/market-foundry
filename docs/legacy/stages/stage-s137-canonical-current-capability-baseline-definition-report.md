# Stage S137 — Canonical Current Capability Baseline Definition

> **Stage:** S137
> **Type:** Consolidation / Baseline Definition
> **Status:** Complete
> **Predecessor:** S136 (Post-Timeframe-Expansion Readiness Review)

---

## 1. Resumo Executivo

O S137 define formalmente a **baseline operacional canônica** do Market Foundry com base exclusivamente no que já existe e já foi provado após TC-01 (S131–S136).

Nenhuma feature nova foi aberta. Nenhuma family foi adicionada. Nenhuma refatoração foi executada. O escopo é inteiramente de **consolidação e formalização**.

O resultado é um conjunto de documentos que responde:
- O que o Foundry faz hoje?
- Sob quais condições ele é considerado saudável?
- O que está dentro e fora da baseline?
- O que essa baseline implica para uma futura integração com ClickHouse?

---

## 2. Baseline Escolhida e Justificativa

### O Loop Canônico

O loop canônico é o **vertical slice completo**: da captura de dados de mercado até a execução simulada.

```
Binance WS → ingest → derive → store → gateway (query)
                                  ↓
                              execute → fill
```

### Por que este loop?

1. **Já existe e funciona** — provado por smoke tests, validação TC-01, e operação multi-símbolo
2. **É representativo** — exercita todos os 7 runtimes, todos os layers, e toda a cadeia causal
3. **É repetível** — scripts de ativação e validação já existem (`live-pipeline-activate.sh`, `smoke-first-slice.sh`, `smoke-multi-symbol.sh`)
4. **É observável** — query surfaces HTTP e NATS KV cobrem todo o pipeline
5. **É mínimo** — usa apenas o que está ativo em config, sem dependências opcionais

### Composição

| Dimensão | Valores na Baseline |
|----------|-------------------|
| **Runtimes** | NATS, configctl, gateway, ingest, derive, store, execute (7) |
| **Símbolos** | btcusdt (primário), ethusdt (secundário) |
| **Source** | binancef |
| **Timeframes** | 60s, 300s, 900s, 3600s |
| **Evidence families** | candle, tradeburst, volume |
| **Signal families** | rsi |
| **Decision families** | rsi_oversold |
| **Strategy families** | mean_reversion_entry |
| **Risk families** | position_exposure |
| **Execution families** | paper_order (derive), venue_market_order (execute/store) |
| **Venue adapter** | paper_simulator |

---

## 3. Loop e Escopo Definidos

### Cadeia Causal Completa

```
observation → candle/tradeburst/volume → rsi → rsi_oversold → mean_reversion_entry → position_exposure → paper_order → venue_market_order (fill)
```

Cada elo da cadeia é exercitado em todos os 4 timeframes e para cada símbolo ativo.

### Fases Operacionais

| Fase | Duração Esperada | O que acontece |
|------|-----------------|----------------|
| **Startup** | < 30s | Todos os serviços saudáveis, streams criados |
| **Activation** | 60–75s | Primeiro candle 60s emitido |
| **Warm-up** | ~15min | RSI ativo no TF de 60s (15 candles) |
| **Steady-state** | Indefinido | Todas as query surfaces retornam dados frescos |
| **Shutdown** | < 10s | Graceful; janelas em andamento são perdidas |

### Automação Existente

| Script | O que valida |
|--------|-------------|
| `live-pipeline-activate.sh` | Build → start → health → seed → validate |
| `smoke-first-slice.sh` | Evidence em 4 TFs + error handling |
| `smoke-multi-symbol.sh` | Multi-symbol × multi-family × multi-TF |
| `tests/http/*.http` | Queries manuais para cada domínio |

---

## 4. Critérios de Sucesso

Definidos em `current-capability-baseline-success-criteria.md` com 30 critérios organizados em 5 tiers + error conditions:

| Tier | Critérios | Tempo | O que prova |
|------|-----------|-------|-------------|
| 1: Infraestrutura | B-01 → B-05 | < 2min | Plataforma viva |
| 2: Pipeline | B-06 → B-12 | < 5min | Dados fluindo |
| 3: Cadeia completa | B-13 → B-18 | < 20min | Vertical slice inteiro |
| 4: Multi-símbolo | B-19 → B-23 | < 25min | Escala horizontal |
| 5: TC-01 coverage | B-24 → B-27 | 15–75min | Todos os timeframes |
| Erros | B-28 → B-30 | < 1min | Error handling |

**Decisão de pass/fail:**
- **BASELINE PASS:** Tiers 1–3 passam (B-01 a B-18)
- **BASELINE PASS (full):** Todos os 30 critérios passam
- **BASELINE FAIL:** Qualquer critério de Tier 1 ou 2 falha

---

## 5. Limites e Não-Objetivos

### Limitações Aceitas

| ID | Limitação | Aceita Porque |
|----|-----------|---------------|
| L-01 | Estado de janela apenas em memória | Adequado para TC-01; persistence é D-01 |
| L-02 | Lista global de timeframes | Sem necessidade heterogênea demonstrada |
| L-03 | Sem snapshots parciais de candle | Simplifica o publisher |
| L-04 | Tracking apenas agregado | Adequado a 4 TFs |
| L-05 | RSI warm-up longo em 900s/3600s | Restrição física |
| L-06 | Representação de TF apenas em inteiros | Unambíguo |
| L-07 | Sem endpoint de discovery de TFs | Sem consumidores externos |
| L-08 | Volume de logs escala linearmente | Inerente ao design |

### Explicitamente Fora de Escopo

- TC-02 (timeframes adicionais)
- Novas families (ema_crossover, MACD, Bollinger)
- Novos símbolos além de btcusdt/ethusdt
- Novas sources além de binancef
- ClickHouse como dependência de runtime
- Execução real em venue (binance_futures_testnet)
- Backtest, alertas, P&L tracking
- State persistence / WAL
- Per-binding timeframe config
- Dashboard ou views agregadas

---

## 6. Preparação para ClickHouse

Documentada em `current-baseline-and-future-clickhouse-preparation-notes.md`.

### Principais conclusões:

1. **Os event streams da baseline são a fonte natural de ingestão** para ClickHouse
2. **O padrão de consumer (store)** é o template arquitetural para um writer de ClickHouse
3. **Os gaps de query** (histórico profundo, análise cross-dimensional, backtesting) definem o valor do ClickHouse
4. **A cardinalidade é conhecida** e trivial para ClickHouse na escala atual
5. **ClickHouse deve permanecer opcional** — o pipeline não pode depender dele

### Sequência de preparação recomendada:

| Fase | O que fazer | Quando |
|------|-------------|--------|
| A: Schema Design | Desenhar tabelas CH baseadas nos event schemas | Pode começar agora |
| B: Writer Design | Projetar serviço writer (padrão store) | Após schema |
| C: Query Surface | Projetar rotas gateway para queries históricas | Após writer |

---

## 7. Preparação Recomendada para S138

O S137 consolida. O S138 deve **escolher a próxima frente** com base nesta baseline.

### Opções candidatas (já identificadas em S136):

| Opção | Tipo | Pré-requisitos da Baseline |
|-------|------|---------------------------|
| **Product wave** (paper trading P&L, alertas) | Produto | Baseline PASS (B-01 a B-18) |
| **Nova signal family** (ema_crossover ativação) | Expansão incremental | Baseline PASS; schema.go já tem registro |
| **ClickHouse Phase A** (schema design) | Infra analítica | Baseline PASS; event schemas estáveis |
| **TC-02** (mais timeframes) | Expansão temporal | D-01 resolvido (state persistence) |

### Recomendação:

O S138 deveria ser um **stage de decisão** que avalia estas opções contra critérios de valor de produto e viabilidade técnica, usando a baseline S137 como referência operacional.

A baseline canônica definida neste stage garante que qualquer direção escolhida parte de uma fundação **conhecida, validada e repetível**.

---

## 8. Entregáveis

| Documento | Caminho | Conteúdo |
|-----------|---------|----------|
| Baseline Definition | `docs/architecture/current-capability-baseline-definition.md` | Definição completa da baseline: runtimes, símbolos, TFs, families, query surfaces, fases operacionais, limitações |
| Success Criteria | `docs/architecture/current-capability-baseline-success-criteria.md` | 30 critérios objetivos em 5 tiers + error conditions |
| ClickHouse Notes | `docs/architecture/current-baseline-and-future-clickhouse-preparation-notes.md` | Mapeamento baseline → ClickHouse, pré-condições, sequência de preparação |
| Stage Report | `docs/stages/stage-s137-canonical-current-capability-baseline-definition-report.md` | Este documento |

---

## 9. Critérios de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Baseline do Foundry claramente definida | **Atendido** — 10 seções cobrindo todos os aspectos |
| Escopo focado em capacidade existente | **Atendido** — zero novas features, families ou expansões |
| Critérios de sucesso objetivos | **Atendido** — 30 critérios pass/fail verificáveis |
| Baseline serve como referência operacional real | **Atendido** — mapeada a scripts e endpoints existentes |
| ClickHouse como direção estratégica, não implementação | **Atendido** — apenas análise e preparação documental |
| Guard rails respeitados | **Atendido** — sem TC-02, sem nova family, sem ClickHouse runtime |
