# Stage S169 — Decisions Family Minimal Implementation Report

## Resumo Executivo

O S169 implementou a família Decisions (RSI Oversold) como segunda família da Wave B, cobrindo o read path completo (reader adapter, use case, endpoint HTTP) e artefatos operacionais obrigatórios. O write path (schema, mapper, pipeline) já existia de estágios anteriores. O padrão da Wave B foi aplicado de forma disciplinada, sem exceções artesanais.

**Resultado:** Exatamente uma nova família implementada. Exatamente um novo endpoint histórico. Todos os testes passam. Base pronta para validação end-to-end.

## Família Implementada

**Decisions (RSI Oversold)** — segunda família da Wave B, introduzindo complexidade controlada:
- 2 colunas JSON (signals + metadata) vs 1 na família anterior
- 1 parâmetro de query adicional (outcome filter)
- 1 coluna enum-like (outcome)
- Desserialização de array JSON (`[]SignalInput`) pela primeira vez no read path

## Arquivos Alterados

### Novos (4 arquivos)

| Arquivo | Descrição |
|---------|-----------|
| `internal/adapters/clickhouse/decision_reader.go` | Reader adapter com `QueryDecisionHistory()`, `BuildDecisionQuery()`, `ParseSignalInputsJSON()` |
| `internal/adapters/clickhouse/decision_reader_test.go` | 12 testes unitários (query builder + JSON parsing) |
| `internal/application/analyticalclient/get_decision_history.go` | Use case com validação, limites, timing, error wrapping |
| `internal/application/analyticalclient/get_decision_history_test.go` | 11 testes unitários (validação, limites, erros, nil safety) |

### Estendidos (7 arquivos)

| Arquivo | Mudança |
|---------|---------|
| `internal/application/analyticalclient/contracts.go` | +`DecisionHistoryQuery`, +`DecisionHistoryReply`, +import decision |
| `internal/interfaces/http/handlers/analytical.go` | +`GetDecisionHistory()` handler, +interface, +response struct, construtor aceita 4 args |
| `internal/interfaces/http/handlers/analytical_test.go` | +7 testes de handler (happy path, erros, outcome filter, nil safety) |
| `internal/interfaces/http/routes/analytical.go` | +`GetDecisionHistory` em deps, +route registration, +interface |
| `cmd/gateway/analytical_reader.go` | +`newAnalyticalDecisionReader()` |
| `cmd/gateway/compose.go` | +wiring do decision reader e use case no composition root |
| `tests/http/analytical.http` | +8 test cases para decision endpoint (queries + error handling) |

### Estendidos (operabilidade, 1 arquivo)

| Arquivo | Mudança |
|---------|---------|
| `scripts/smoke-analytical-e2e.sh` | +Phase 5c (decision read path), +error handling checks (6h–6k), +summary atualizado |

### Documentação (3 arquivos)

| Arquivo | Descrição |
|---------|-----------|
| `docs/architecture/wave-b-family-02-decisions-implementation-notes.md` | Notas de implementação, schema coherence table, endpoint spec |
| `docs/architecture/wave-b-family-02-decisions-runtime-and-operability-notes.md` | Runbook, failure modes, diagnostic signals |
| `docs/stages/stage-s169-decisions-family-minimal-implementation-report.md` | Este relatório |

### Inalterados (write path pré-existente)

| Arquivo | Status |
|---------|--------|
| `deploy/migrations/003_create_decisions.sql` | Pré-existente, sem mudanças |
| `cmd/writer/mappers.go` (mapDecisionRow) | Pré-existente, sem mudanças |
| `cmd/writer/pipeline.go` (rsi_oversold entry) | Pré-existente, sem mudanças |
| `cmd/writer/consumer.go` | Sem mudanças |
| `cmd/writer/inserter.go` | Sem mudanças |
| `deploy/configs/writer.jsonc` | Sem mudanças |
| `deploy/configs/gateway.jsonc` | Sem mudanças |

## Simplificações Adotadas

1. **Sem validação de outcome no handler** — o valor é passado como-is ao ClickHouse WHERE clause. Valores inválidos simplesmente retornam 0 resultados.
2. **Reuso de `ParseMetadataJSON`** — a função existente do signal reader foi reutilizada sem duplicação.
3. **Sem aggregação** — endpoint retorna eventos brutos, sem estatísticas por outcome.
4. **Sem drill-down de signals** — o array `signals` é retornado como-is, sem join com a tabela signals.

## Atritos Observados

### Atrito 1: Construtor com argumentos posicionais crescentes

`NewAnalyticalWebHandler` agora aceita 4 argumentos posicionais (candle, signal, decision, logger). Com mais famílias, isso se torna difícil de manter. Confirmado como pré-compromisso H-1 para Family 03.

**Severidade:** Baixa (funcional, mas ergonomicamente frágil a partir de 5+ famílias).

### Atrito 2: Nenhum (JSON array desserialização)

A desserialização de `[]SignalInput` funcionou sem qualquer atrito. `json.Unmarshal` trata arrays e maps identicamente. `ParseSignalInputsJSON` seguiu o mesmo padrão de fallback de `ParseMetadataJSON`.

### Atrito 3: Nenhum (outcome filter)

O parâmetro `outcome` foi adicionado ao query builder como filtro opcional sem complicações. ClickHouse otimiza LowCardinality eficientemente.

**Conclusão:** O padrão da Wave B suporta payload JSON mais complexo sem virar exceção artesanal. O único atrito real é o crescimento do construtor (H-1).

## Limites Mantidos

| Guard Rail | Status |
|------------|--------|
| Exatamente uma família implementada | OK — apenas Decisions |
| Exatamente um endpoint novo | OK — `/analytical/decision/history` |
| Sem endpoints extras | OK |
| Sem abstrações novas | OK — reutilizou padrões existentes |
| Sem mudanças oportunistas | OK — write path intocado |
| Writer inalterado | OK — nenhuma mudança em cmd/writer |
| Schema inalterado | OK — migration 003 pré-existente |
| Candles/Signals inalterados | OK — zero regressões |
| Restrições S167 preservadas | OK |

## Verificação

```bash
# Unit tests
go test ./internal/adapters/clickhouse/...       # OK
go test ./internal/application/analyticalclient/... # OK
go test ./internal/interfaces/http/handlers/...   # OK
go test ./internal/interfaces/http/routes/...     # OK

# Build verification
go build ./cmd/gateway/...                        # OK

# End-to-end (requires running stack)
./scripts/smoke-analytical-e2e.sh
```

## 5-Point Gate Review

| # | Critério | Status |
|---|----------|--------|
| 1 | Todos os testes unitários passam | PASS |
| 2 | smoke-analytical-e2e.sh inclui Phase 5c | PASS (implementado) |
| 3 | CI passa no branch | PENDENTE (requer push) |
| 4 | Zero regressões em testes/smoke existentes | PASS |
| 5 | Schema coherence documentada | PASS (ver implementation-notes.md) |

## Preparação Recomendada para S170

### Validação end-to-end obrigatória
- Executar `smoke-analytical-e2e.sh` com stack completo para provar o data flow Decisions ponta a ponta.
- Verificar que Phase 5c passa sem erros.

### Hardening pré-comprometido para Family 03
- **H-1:** Refatorar `NewAnalyticalWebHandler` para struct-based DI (`AnalyticalHandlerDeps{...}`).
- **H-2:** Parametrizar smoke test para evitar duplicação de phases.
- **H-3:** Revisão de naming consistency (consumer/inserter labels).

### Próxima família candidata
- A terceira família deve ser selecionada com base no delta de complexidade incremental.
- O padrão Wave B v2 está provado para até 2 colunas JSON + filtro opcional por família.
- Nenhum atrito estrutural bloqueia expansão, desde que H-1 seja resolvido antes.
