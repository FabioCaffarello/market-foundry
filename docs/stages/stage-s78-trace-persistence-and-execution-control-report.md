# Stage S78 — Trace Persistence and Execution Control Report

**Status**: complete
**Date**: 2026-03-18

## Resumo Executivo

O S78 materializa dois requisitos identificados no S74 como hard blockers para a fronteira de ação:

1. **Trace persistence**: `CorrelationID` e `CausationID` passam a ser persistidos no `ExecutionIntent` dentro do KV bucket, tornando a trilha causal consultável via query endpoint existente.

2. **Execution control gate**: Um kill switch explícito baseado em NATS KV permite halt/resume do pipeline de execução sem restart de binários, com trilha auditável de quem alterou e por quê.

Ambos foram implementados com o menor desenho aceitável, sem inflação de escopo.

## Trace Persistence

### O que foi feito

- Adicionados campos `CorrelationID` e `CausationID` ao struct `ExecutionIntent` no domínio.
- O evaluator actor no derive popula esses campos a partir do `riskAssessedMessage` upstream.
- Os campos são serializados junto com o intent no KV bucket `EXECUTION_PAPER_ORDER_LATEST`.
- A resposta do endpoint `GET /execution/:type/latest` inclui os campos de trace.

### Impacto

| Antes | Depois |
|-------|--------|
| Trace só em logs estruturados | Trace persistido no read-side |
| Requer correlação manual de logs | Query endpoint retorna correlation/causation |
| Cadeia causal perdida no KV | Cadeia preservada end-to-end |

## Execution Control Gate

### O que foi feito

- Novo struct de domínio `ControlGate` com estados `active`/`halted`.
- Novo KV bucket `EXECUTION_CONTROL` (1 MB, FileStorage).
- O publisher actor no derive lê o gate antes de cada publish.
- O query responder no store serve get/set do gate via NATS request/reply.
- Novos endpoints HTTP: `GET /execution/control` e `PUT /execution/control`.
- Novo gateway adapter para controle via NATS.

### Semântica

- **Fail-open**: Gate ausente = active. Sistema não para por falta de bucket.
- **Global**: Uma gate para todo o pipeline de execução (sem granularidade per-symbol).
- **Síncrono**: Leitura do KV a cada publish (~1ms local).
- **Auditável**: Cada mudança registra `reason`, `updated_by`, `updated_at`.

## Arquivos Alterados

### Domain
| Arquivo | Alteração |
|---------|-----------|
| `internal/domain/execution/execution.go` | +CorrelationID, +CausationID no ExecutionIntent |
| `internal/domain/execution/control.go` | **novo** — ControlGate, GateStatus |

### Adapters
| Arquivo | Alteração |
|---------|-----------|
| `internal/adapters/nats/execution_control_kv_store.go` | **novo** — KV store para EXECUTION_CONTROL |
| `internal/adapters/nats/execution_control_gateway.go` | **novo** — gateway adapter para control queries |
| `internal/adapters/nats/execution_registry.go` | +ControlGet, +ControlSet specs |

### Application
| Arquivo | Alteração |
|---------|-----------|
| `internal/application/executionclient/control_contracts.go` | **novo** — query/command contracts |
| `internal/application/executionclient/get_execution_control.go` | **novo** — get/set use cases |
| `internal/application/ports/execution.go` | +ExecutionControlGateway interface |

### Actors
| Arquivo | Alteração |
|---------|-----------|
| `internal/actors/scopes/derive/execution_evaluator_actor.go` | Set trace fields no intent |
| `internal/actors/scopes/derive/execution_publisher_actor.go` | +gate check, +controlStore, +halted counter |
| `internal/actors/scopes/store/query_responder_actor.go` | +control store, +handleExecutionControlGet/Set |

### HTTP/Gateway
| Arquivo | Alteração |
|---------|-----------|
| `internal/interfaces/http/handlers/execution_control.go` | **novo** — GET/PUT /execution/control |
| `internal/interfaces/http/routes/execution.go` | +control routes |
| `internal/interfaces/http/routes/core.go` | +ExecutionControl deps |
| `cmd/gateway/gateway.go` | +newExecutionControlGateway |
| `cmd/gateway/run.go` | +wire control use cases |

### Tests
| Arquivo | Alteração |
|---------|-----------|
| `internal/actors/scopes/store/execution_projection_actor_test.go` | +trace fields in fixtures |

### Documentation
| Arquivo | Status |
|---------|--------|
| `docs/architecture/execution-trace-persistence.md` | **novo** |
| `docs/architecture/execution-control-and-kill-switch.md` | **novo** |

## Critérios de Aceite

| Critério | Status |
|----------|--------|
| Trilha causal mínima persistida de forma consultável | OK — correlation/causation no KV |
| Read-side claro e sob authority do store | OK — store é sole writer |
| Mecanismo mínimo de controle operacional explícito | OK — gate active/halted via HTTP |
| Gateway continua limpo | OK — gateway é proxy, sem enforcement |
| Base mais segura para validação operacional | OK — halt sem restart, trace sem log mining |

## Limitações Remanescentes

1. **Granularidade do gate**: Global — não há per-source ou per-symbol control. Suficiente para paper order, requer extensão para venue real.

2. **Latência do gate check**: Leitura síncrona do KV a cada publish. Aceitável para throughput atual. Um KV watcher com atomic bool seria mais eficiente em alta frequência.

3. **Histórico do gate**: Sem histórico de mudanças de estado. A evolução para audit log requer stream dedicado.

4. **Trace read-side**: Apenas latest-only. Histórico de trace requer o JetStream stream (72h retention) ou um bucket de histórico dedicado.

5. **Cross-domain trace query**: Não há endpoint que resolva a cadeia causal completa em uma única request. Requer queries individuais por domínio.

## Preparação Recomendada para S79

1. **Execution lifecycle validation**: Com trace e control em lugar, validar o ciclo de vida execution end-to-end com smoke tests multi-symbol que verifiquem que a cadeia causal chega íntegra ao read-side.

2. **Control gate smoke test**: Incluir no script de smoke test a sequência halt → verify blocked → resume → verify publishing, confirmando que o gate funciona em cenário real.

3. **Per-source gate granularity**: Se o próximo passo for multi-venue, considerar extensão do gate para key `{source}` ao invés de `global`, permitindo halt seletivo.

4. **Gate audit stream**: Para compliance, considerar um JetStream stream `EXECUTION_CONTROL_AUDIT` que capture todas as mudanças de gate com timestamp e operador.
