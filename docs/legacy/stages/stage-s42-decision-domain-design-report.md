# Stage S42 — Decision Domain Design Report

> **Type**: Design
> **Status**: Complete
> **Date**: 2026-03-17
> **Predecessor**: S41 (Signal Multi-Symbol Verification)
> **Successor**: S43 (Decision First Slice Implementation)

---

## 1. Resumo Executivo

O Stage S42 desenhou formalmente o domínio `decision` no Market Foundry. O desenho define
boundaries claros, contracts canônicos, stream families, activation model e query surface —
preparando uma implementação mínima em S43 sem improviso.

**Resultado**: Decision é um domínio próprio e independente que transforma sinais em
julgamentos categóricos (triggered/not_triggered/insufficient) sem cruzar fronteiras com
signal, evidence ou observation.

---

## 2. Entregáveis Produzidos

| # | Documento | Conteúdo |
|---|---|---|
| 1 | `docs/architecture/decision-domain-design.md` | Desenho principal: identidade, modelo, boundaries, invariantes, event contracts |
| 2 | `docs/architecture/decision-stream-families.md` | Catálogo de famílias: DF-01 (RSI Oversold), DF-02 (MACD Crossover), futuras |
| 3 | `docs/architecture/decision-activation-and-ownership.md` | Modelo de ativação, ownership matrix, actor trees, data flow |
| 4 | `docs/architecture/decision-query-surface-guidelines.md` | Regras de query HTTP, envelopes NATS, invariantes de leitura |
| 5 | `docs/stages/stage-s42-decision-domain-design-report.md` | Este relatório |

---

## 3. Decisões Principais

### 3.1 Decision é domínio próprio

Decision NÃO é extensão de signal. Tem:
- Pacote domain separado: `internal/domain/decision`
- Stream separado: `DECISION_EVENTS`
- KV buckets separados: `DECISION_{TYPE}_LATEST`
- Config separado: `pipeline.decision_families`
- Query surface separado: `decision.query.*`

### 3.2 Decision vive no binário derive

A lógica de avaliação consome sinais computados localmente no mesmo SourceScopeActor.
Usar mensagens de ator locais (não JetStream) evita latência, seguindo o padrão
evidence→signal. Binário separado foi considerado e rejeitado — pode ser reconsiderado
se o domínio crescer em complexidade.

### 3.3 Outcome é categórico, não numérico

Decisions produzem `triggered`, `not_triggered` ou `insufficient` — não scores.
O campo `confidence` fornece graduação sem tornar o outcome ambíguo.

### 3.4 RSI Oversold é a primeira família

Consome um único sinal (RSI), tem a lógica mais simples possível (comparação de threshold),
e prova o pipeline inteiro end-to-end. Espelha a escolha de RSI como primeiro sinal.

### 3.5 Sem ativação implícita

Ativar `rsi_oversold` em `decision_families` NÃO ativa automaticamente `rsi` em
`signal_families`. O operador é responsável pela consistência. Isso é intencional — evita
cadeias de ativação implícitas.

### 3.6 Latest-only em Phase 1

History para decisions é adiado para S44+. Latest-only é suficiente para provar o pipeline.

---

## 4. Limites Intencionais

### O que decision NÃO é

| Conceito | Status |
|---|---|
| Strategy | Fora de escopo — Phase 3+ |
| Risk management | Fora de escopo — Phase 3+ |
| Execution | Fora de escopo — Phase 3+ |
| Portfolio | Fora de escopo — Phase 3+ |
| Rule engine configurável | Rejeitado — famílias são code-defined |
| Alerting system | Fora de escopo — notificação é concern separado |
| Cross-symbol decisions | Rejeitado — sempre per-symbol, per-timeframe |
| Signal aggregator | Rejeitado — decision aplica lógica de avaliação, não agregação |

### O que foi adiado

| Item | Para quando | Motivo |
|---|---|---|
| Decision history (KV + query) | S44+ | Latest-only suficiente para first slice |
| Multi-signal confluence (DF-03) | S44+ | Requer single-signal families provadas |
| MACD crossover (DF-02) | S43 stretch ou S44 | Depende de MACD signal (não implementado) |
| Volume spike entry (DF-04) | S45+ | Depende de SF-03 (não implementado) |
| Raccoon-CLI decision drift rules | S43 | Governança acompanha implementação |
| ClickHouse decision projection | Não planejado | Analytical storage é concern separado |
| WebSocket streaming de decisions | Não planejado | Fora do modelo request/reply |

---

## 5. Invariantes Canônicos

9 invariantes de boundary de domínio (DBI-1 a DBI-9) e 7 invariantes operacionais (OI-1 a OI-7)
foram documentados em `decision-domain-design.md`. Destaques:

- **DBI-1**: `internal/domain/decision` não importa signal, evidence ou observation
- **DBI-7**: Decision não retroalimenta signal — grafo é estritamente unidirecional
- **DBI-9**: Decision consome dados de signal como tipos próprios, não como `signal.Signal`
- **OI-6**: Decision não consome EVIDENCE_EVENTS — apenas sinais

---

## 6. Preparação para S43

### O que S43 deve implementar (primeira fatia mínima)

1. `internal/domain/decision/` — tipos Decision, Outcome, SignalInput, eventos
2. `internal/application/decision/` — RSI oversold evaluator (lógica pura)
3. `internal/actors/scopes/derive/decision_evaluator_actor.go` — actor no scope
4. `internal/actors/scopes/derive/decision_publisher_actor.go` — publisher
5. `internal/adapters/nats/decision_publisher.go` — adapter NATS
6. `internal/adapters/nats/decision_consumer.go` — consumer no store
7. `internal/adapters/nats/decision_registry.go` — stream/bucket setup
8. `internal/adapters/nats/decision_kv_store.go` — KV read/write
9. `internal/actors/scopes/store/decision_consumer_actor.go` — store consumer
10. `internal/actors/scopes/store/decision_projection_actor.go` — store projection
11. `internal/application/decisionclient/` — query use case
12. `internal/interfaces/http/handlers/decision.go` — HTTP handler
13. `internal/interfaces/http/routes/decision.go` — route registration
14. `internal/shared/settings/schema.go` — `DecisionFamilies` config field
15. `deploy/configs/*.jsonc` — config entries
16. Tests unitários e de integração

### Pré-condições para S43

| ID | Condição | Status |
|---|---|---|
| P-1 | Signal adapter test coverage | Resolvido (S39) |
| P-2 | Signal KV store test coverage | Resolvido (S39) |
| P-3 | Trade burst/volume KV store test coverage | Resolvido (S39) |
| P-4 | Raccoon-CLI signal governance | Resolvido (S40) |
| P-5 | Signal multi-symbol verification | Resolvido (S41) |
| P-6 | Decision domain design | Resolvido (S42 — este stage) |

**Todas as pré-condições estão resolvidas. S43 pode iniciar imediatamente.**

---

## 7. Riscos Identificados

| Risco | Mitigação |
|---|---|
| Decision evaluator se torna stateful demais | Manter evaluators puros; estado de warm-up segue padrão signal |
| Tentação de acessar evidence direto | DBI invariantes + raccoon-cli drift check |
| Confusion entre decision e strategy | Boundaries explícitos documentados; strategy é Phase 3+ |
| Config explosion (3 listas de famílias) | Considerar config validation que avisa sobre famílias ativas sem dependências |

---

## 8. Verificação dos Critérios de Aceite

| Critério | Status |
|---|---|
| `decision` claramente definido como domínio próprio | ✅ Documentado em decision-domain-design.md |
| Boundaries com signal, store e gateway explícitos | ✅ 9 DBI + 7 OI invariantes |
| Stream families e ownership claros | ✅ 4 famílias catalogadas, ownership matrix completo |
| Desenho prepara implementação mínima sem improviso | ✅ 16 artefatos listados para S43 |
| Etapa evita que decision entre torto | ✅ Guard rails documentados, non-goals explícitos |

---

## 9. Conclusão

O domínio `decision` está formalmente desenhado. Nasce como domínio próprio, com boundaries
claros, sem herdar vícios de signal, e com um caminho de implementação concreto. A primeira
família (RSI Oversold) é intencionalmente simples para provar o pipeline end-to-end antes
de escalar para famílias mais complexas.

O próximo passo natural é S43: implementação da primeira fatia de decision.
