# Stage S164 — First Controlled Analytical Family Expansion Report

## Resumo Executivo

O Stage S164 implementou a primeira expansão controlada da camada analítica sob o padrão Wave B definido no S163. A família escolhida foi **Signals (RSI)** — a candidata de menor risco com write path já ativo e domain type mais simples após candles. A implementação seguiu integralmente o checklist de 9 artefatos, o fluxo Schema→Writer→Reader→Gateway→Tests, e todas as constraints do S162/S163.

**Resultado:** exatamente uma família expandida, exatamente um endpoint histórico adicionado, schema/writer/reader/gateway coerentes, padrão aplicado com disciplina.

## Família Escolhida e Justificativa

| Critério | Signal (RSI) | Próxima alternativa (Decision) |
|---|---|---|
| Write path ativo | SIM — pipeline `rsi` operacional | SIM — pipeline `rsi_oversold` operacional |
| Complexidade do domain type | 8 campos, 1 JSON field | 14 campos, 2 JSON fields |
| Schema existente | SIM — migration 002 | SIM — migration 003 |
| Recomendação S163 | Explícita | Não mencionada |
| Posição na dependency chain | Layer 1 (depende de evidence) | Layer 2 (depende de signal) |

**Decisão:** Signals minimiza risco e maximiza aprendizado sobre o padrão antes de famílias mais complexas.

## Artefatos Implementados

### Wave B Checklist — 9 Artefatos Obrigatórios

| # | Artefato | Status | Arquivo |
|---|---|---|---|
| 1 | Schema DDL | PRE-EXISTING | `deploy/migrations/002_create_signals.sql` |
| 2 | Writer mapper | PRE-EXISTING | `cmd/writer/mappers.go:mapSignalRow` |
| 3 | Writer pipeline entry | PRE-EXISTING | `cmd/writer/pipeline.go` (pipeline `rsi`) |
| 4 | Reader adapter | NEW | `internal/adapters/clickhouse/signal_reader.go` |
| 5 | Contracts | EXTENDED | `internal/application/analyticalclient/contracts.go` |
| 6 | Use case | NEW | `internal/application/analyticalclient/get_signal_history.go` |
| 7 | HTTP handler | EXTENDED | `internal/interfaces/http/handlers/analytical.go` |
| 8 | HTTP route | EXTENDED | `internal/interfaces/http/routes/analytical.go` |
| 9 | Tests | NEW | 4 test files, 28+ test cases total |

### Arquivos Criados (4)

| File | LOC | Purpose |
|---|---|---|
| `internal/adapters/clickhouse/signal_reader.go` | ~120 | Reader adapter: parameterized SELECT, row→domain |
| `internal/adapters/clickhouse/signal_reader_test.go` | ~140 | Query builder + metadata parsing tests |
| `internal/application/analyticalclient/get_signal_history.go` | ~90 | Use case: validation, delegation, timing |
| `internal/application/analyticalclient/get_signal_history_test.go` | ~140 | 10 use case tests |

### Arquivos Modificados (8)

| File | Change |
|---|---|
| `internal/application/analyticalclient/contracts.go` | +`SignalHistoryQuery`, +`SignalHistoryReply` |
| `internal/interfaces/http/handlers/analytical.go` | +`GetSignalHistory` handler method |
| `internal/interfaces/http/handlers/analytical_test.go` | +6 handler tests |
| `internal/interfaces/http/routes/analytical.go` | +`GetSignalHistory` dep, +route `/analytical/signal/history` |
| `cmd/gateway/analytical_reader.go` | +`newAnalyticalSignalReader()` factory |
| `cmd/gateway/analytical_reader_test.go` | +compile-time interface assertion |
| `cmd/gateway/compose.go` | Wired signal reader into analytical deps |
| `tests/http/analytical.http` | +7 signal HTTP test requests |

### Documentação Criada (4)

| File | Content |
|---|---|
| `docs/architecture/wave-b-family-01-implementation-notes.md` | Escolha da família, decisões de design, simplificações |
| `docs/architecture/wave-b-family-01-schema-writer-reader-gateway-path.md` | Verificação de coerência DDL↔writer↔reader, data flow, endpoint spec |
| `docs/architecture/wave-b-family-01-runbook-and-operability-notes.md` | Verificação de saúde, cenários de falha, paridade de observabilidade |
| `docs/stages/stage-s164-...report.md` | Este relatório |

## Verificação de Coerência Schema

| Column | DDL Type | Writer | Reader | Aligned |
|---|---|---|---|---|
| type | LowCardinality(String) | string | string | YES |
| source | LowCardinality(String) | string | string | YES |
| symbol | LowCardinality(String) | string | string | YES |
| timeframe | UInt32 | uint32 | uint32 | YES |
| value | Float64 | parseFloat→float64 | float64→FormatFloat | YES |
| metadata | String | marshalJSON→string | string→ParseMetadataJSON | YES |
| final | Bool | bool | bool | YES |
| timestamp | DateTime64(3) | time.Time | time.Time | YES |

**Resultado: 8/8 colunas alinhadas — APROVADO.**

## Testes

| Pacote | Testes Adicionados | Resultado |
|---|---|---|
| `internal/adapters/clickhouse` | 12 (8 query builder + 4 metadata parsing) | PASS |
| `internal/application/analyticalclient` | 10 (use case validation/execution) | PASS |
| `internal/interfaces/http/handlers` | 6 (handler happy/error paths) | PASS |
| `cmd/gateway` | 1 (compile-time interface assertion) | PASS |

**Total: 29 novos testes, todos passando.**

## Verificação dos Critérios de Aceite

| Critério | Status | Evidência |
|---|---|---|
| Exatamente uma nova família expandida | SATISFEITO | Apenas signals (RSI) |
| Exatamente um novo endpoint histórico | SATISFEITO | `GET /analytical/signal/history` |
| Schema/writer/reader/gateway coerentes | SATISFEITO | Tabela de coerência 8/8 |
| Padrão Wave B aplicado com disciplina | SATISFEITO | 9 artefatos, checklist completo |
| Base pronta para prova end-to-end | SATISFEITO | Writer pipeline ativo, reader implementado, endpoint registrado |

## Verificação dos Guard Rails

| Guard Rail | Status | Evidência |
|---|---|---|
| Não expandir múltiplas famílias | RESPEITADO | Apenas signals |
| Não adicionar endpoints extras | RESPEITADO | Apenas `/analytical/signal/history` |
| Não enfraquecer restrições S162 | RESPEITADO | Additive only, no operational changes |
| Não introduzir abstrações novas | RESPEITADO | Seguiu padrão candle sem abstrações genéricas |
| Documentar limites e atritos | RESPEITADO | Seção abaixo |

## Atritos Observados na Primeira Aplicação do Padrão

### Atrito 1: Nome da função `parseEvidenceKeyParams`

O handler de signals reutiliza `parseEvidenceKeyParams()` para extrair source/symbol/timeframe. O nome sugere acoplamento a evidence, mas os parâmetros são universais. Renomear seria uma refatoração horizontal que viola C-9 (additive only). **Decisão:** aceitar o nome e documentar.

### Atrito 2: Construtor `NewAnalyticalWebHandler` acumulando argumentos

Com a adição de `getSignalHistory`, o construtor agora aceita 3 argumentos posicionais. A terceira família vai empurrar para 4. **Risco:** error-prone em uso. **Mitigação futura:** considerar um `AnalyticalHandlerDeps` struct quando família 3 entrar. Não abstrair agora (NG-1: no generic framework).

### Atrito 3: Ausência de validação de `type` contra `knownSignalFamilies`

O reader aceita qualquer `type` string e retorna resultados vazios para tipos desconhecidos. Validação exigiria acoplar o read path ao settings registry. **Decisão:** aceitar — ClickHouse faz a filtragem, resultado vazio é semanticamente correto.

### Atrito 4: Duplicação mecânica entre candle e signal paths

O use case, handler, e reader seguem a mesma estrutura com substituição de tipos. ~80% do código é mecânico. Isso é intencional (cada família é independente), mas se torna tedioso na 4ª+ iteração. **Mitigação futura:** considerar code generation se o atrito escalar. Não abstrair agora.

## Open Debts

### Herdados (carry-forward do S162)

| Debt | Priority | Status |
|---|---|---|
| CI integration de smoke-analytical | Medium | Pendente |
| Backoff jitter no writer retry | Low | Pendente |

### Novos (descobertos nesta iteração)

| Debt | Priority | Justification |
|---|---|---|
| `parseEvidenceKeyParams` naming | Low | Cosmético, não bloqueia |
| Handler constructor ergonomics | Low | Aceitável até família 3 |
| Signal type validation at read path | Low | ClickHouse filtra corretamente |

## Preparação para S165

### Recomendações

1. **Próxima família Wave B:** Decisions (RSI Oversold) — Layer 2, complexidade intermediária (2 JSON fields), pipeline ativo.
2. **CI integration:** antes da terceira família, integrar `smoke-analytical-e2e.sh` no CI (constraint C-3 do S162).
3. **Handler constructor refactor:** considerar `AnalyticalHandlerDeps` struct na próxima família para evitar construtor de 4+ args.
4. **Extend smoke-analytical:** adicionar verificação de signal read path ao script existente.

### Sequência sugerida

```
S165: Decisions (RSI Oversold) — segunda família Wave B
S166: CI integration de smoke-analytical (constraint C-3 deadline)
S167: Strategies (Mean Reversion Entry) — terceira família
...
```

## Conclusão

O Stage S164 validou com sucesso o padrão Wave B na primeira aplicação prática. O fluxo Schema→Writer→Reader→Gateway→Tests foi executado sem bloqueios. Os atritos observados são todos de baixa severidade e não comprometem a disciplina do padrão. A base está pronta para a prova end-to-end da família signals e para a próxima iteração Wave B.
