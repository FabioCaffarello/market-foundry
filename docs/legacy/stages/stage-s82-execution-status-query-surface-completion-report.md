# Stage S82 — Execution Status & Query Surface Completion Report

**Status:** Complete
**Date:** 2026-03-18

## Resumo Executivo

O S82 fecha a superfície de consulta de `execution` pós-`execute`, tornando status propagation explícita e consultável. A adição principal é o endpoint composto `GET /execution/status/latest` que unifica intent (paper_order), result (venue_market_order) e control gate numa única resposta com campo `propagation` derivado.

Antes do S82, um operador precisava de 3 chamadas independentes para entender o estado end-to-end de uma execução. Agora, uma única chamada retorna o panorama completo com diagnóstico de propagação embutido.

## Superfícies de Query/Status Concluídas

### Endpoints Finais

| Endpoint | Método | Descrição |
|----------|--------|-----------|
| `/execution/paper_order/latest` | GET | Latest intent (derive output) |
| `/execution/venue_market_order/latest` | GET | Latest result (execute output) |
| `/execution/status/latest` | **GET** | **Composite: intent + result + gate + propagation** |
| `/execution/control` | GET | Current control gate |
| `/execution/control` | PUT | Update control gate |

### Contratos NATS Adicionados

| Subject | Tipo Request | Tipo Reply |
|---------|-------------|------------|
| `execution.query.status.latest` | `execution.query.v1.status_latest_request` | `execution.query.v1.status_latest_reply` |

### Resposta do Status Endpoint

```json
{
  "intent": { "type": "paper_order", "status": "submitted", ... },
  "result": { "type": "venue_market_order", "status": "filled", ... },
  "gate": { "status": "active", "reason": "", "updated_at": "...", "updated_by": "" },
  "propagation": "filled"
}
```

**Derivação do `propagation`:**
- Se `result` existe -> `result.status`
- Senão se `intent` existe -> `intent.status`
- Senão -> `"none"`

## Arquivos Alterados

### Contracts & Domain
- `internal/application/executionclient/contracts.go` — `ExecutionStatusQuery`, `ExecutionStatusReply`, `DeriveEffectivePropagation()`

### Use Cases
- `internal/application/executionclient/get_execution_status.go` — **novo** `GetExecutionStatusUseCase`

### Ports
- `internal/application/ports/execution.go` — `GetExecutionStatus` adicionado a `ExecutionGateway`

### NATS Adapters
- `internal/adapters/nats/execution_registry.go` — `StatusLatest` ControlSpec
- `internal/adapters/nats/execution_gateway.go` — `GetExecutionStatus()` method

### Store (Query Responder)
- `internal/actors/scopes/store/query_responder_actor.go` — `handleExecutionStatusLatest()` (lê 2 KV stores + control store)

### HTTP Interface
- `internal/interfaces/http/handlers/execution.go` — `GetExecutionStatus` handler, `getExecutionStatusUseCase` interface
- `internal/interfaces/http/routes/execution.go` — rota `GET /execution/status/latest`
- `internal/interfaces/http/routes/core.go` — `GetExecutionStatus` em `ExecutionFamilyDeps`

### Gateway Wiring
- `cmd/gateway/run.go` — `getExecutionStatusUseCase` wired

### Architecture Docs
- `docs/architecture/execution-query-surface-after-execute.md` — **novo**
- `docs/architecture/execution-status-propagation-model.md` — **novo**

## Critérios de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Superfície de consulta coerente após `execute` | OK — intent, result, status, control endpoints completos |
| Status propagation explícita | OK — campo `propagation` derivado, documentado |
| Intent/result surfaces distinguíveis | OK — endpoints separados, types distintos, docs claros |
| Store continua authority | OK — todas as leituras via NATS KV materializado no store |
| Auditabilidade e legibilidade operacional | OK — status endpoint unifica diagnóstico em uma chamada |

## Guard Rails — Verificação

| Guard Rail | Status |
|------------|--------|
| Não abrir venue real | OK — nenhum adapter de venue real adicionado |
| Não inflar API com endpoints redundantes | OK — um endpoint composto, sem duplicação |
| Não criar superfícies ambíguas | OK — intent/result/status/control têm semântica distinta |
| Não transformar latest-only em history | OK — todas as leituras são latest-only |
| Documentar limites | OK — docs explicitam o que é out of scope |

## Limites Remanescentes

1. **Sem history queries**: Todas as superfícies são latest-only. Histórico permanece event-sourced mas não consultável.
2. **Sem causal ordering entre surfaces**: Intent e result são projeções independentes. Uma race condition teórica permite ver result antes de intent (dois consumers independentes).
3. **Sem cross-symbol aggregation**: Cada partição é independente. Não há endpoint que retorne estado de execução em múltiplos símbolos.
4. **VenueOrderID não no read model**: O ID de venue é metadata do evento, não materializado no KV. Disponível apenas via replay do stream.
5. **Sem SLA de latência**: Propagation latency é observacional (~100-500ms em paper venue), sem enforcement.

## Preparação Recomendada para S83

O S82 deixa a query surface completa e documentada. Sugestões para próximo stage:

1. **Operational validation matrix**: Validar end-to-end com smoke test multi-symbol que exercite o status endpoint e verifique propagation != "none" após fluxo completo.
2. **Staleness monitoring**: Implementar alerta quando `propagation == "submitted"` persiste além de `DefaultStalenessMaxAge` (120s).
3. **Real venue readiness gate**: Com a query surface fechada, o próximo step natural é o gate de entrada para venue real — prerequisitos, limites operacionais, e adapter skeleton.
4. **History phase planning**: Se houver demanda, planejar phase 2 do query surface com time-range queries derivadas dos streams existentes.
