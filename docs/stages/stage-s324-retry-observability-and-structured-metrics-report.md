# Stage S324: Retry Observability and Structured Metrics — Report

> Venue closure tranche — S324 execution report.

## Resumo Executivo

S324 adicionou observabilidade estruturada mínima ao retry path do venue
submitter, fechando o gap R-S320-5 (métricas de retry ausentes). A
implementação usa `slog` para logs estruturados e `healthz.Tracker` para
contadores atômicos, ambos opcionais e nil-safe. Todos os 17 testes
pré-existentes passam sem modificação; 6 novos testes validam os sinais
de observabilidade.

## Objetivo

Tornar o retry path auditável e legível operacionalmente sem inflar escopo,
criar dashboards ou instrumentar indiscriminadamente.

## Observabilidade Entregue

### Sinais Estruturados

| Sinal | Tipo | Nível | Quando |
|-------|------|-------|--------|
| `retry attempt failed` | Log | Warn | Cada falha retryable não-terminal |
| `retry succeeded` | Log | Info | Sucesso após retry (attempt > 1) |
| `retry exhausted` | Log | Warn | MaxAttempts consumidos |
| `retry halted by kill switch` | Log | Warn | Kill switch abortou entre tentativas |
| `retry deadline exceeded` | Log | Warn | Budget global excedido |

### Contadores

| Contador | Semântica |
|----------|-----------|
| `retry_attempts` | Tentativas individuais não-terminais |
| `retry_success_after_retry` | Sucesso após pelo menos 1 retry |
| `retry_exhausted` | Sequências que esgotaram tentativas |
| `retry_halted` | Sequências abortadas por kill switch |
| `retry_deadline_exceeded` | Sequências abortadas por deadline |

### Enriquecimento no Actor

O `VenueAdapterActor` agora inclui metadata de retry (`retry_attempts`,
`retry_exhausted`, `retry_halted`, `retry_deadline_exceeded`) no log de
erro `venue submit failed` quando presente em `Problem.Details`.

## Arquivos Alterados

| Arquivo | Ação | Descrição |
|---------|------|-----------|
| `internal/application/execution/retry_submitter.go` | Modificado | Logger + tracker opcionais, logs e contadores em cada ponto de decisão |
| `internal/application/execution/retry_submitter_test.go` | Modificado | +6 testes de observabilidade (S324) |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | Modificado | Log de erro enriquecido com retry metadata |
| `docs/architecture/retry-observability-and-structured-metrics.md` | Novo | Design e invariantes da observabilidade |
| `docs/architecture/retry-metrics-logging-semantics-and-usage.md` | Novo | Referência operacional de métricas e logs |
| `docs/stages/stage-s324-retry-observability-and-structured-metrics-report.md` | Novo | Este relatório |

## Evidências de Teste

### Testes S324 (6 novos)

| Teste | Cenário | Resultado |
|-------|---------|-----------|
| `TestRetryObservability_SuccessAfterRetry_LogsAndCounts` | Sucesso no 2o attempt → verifica logs e contadores | PASS |
| `TestRetryObservability_Exhaustion_LogsAndCounts` | 3 tentativas esgotadas → verifica log e contadores | PASS |
| `TestRetryObservability_Halt_LogsAndCounts` | Kill switch ativo → verifica log e contador | PASS |
| `TestRetryObservability_Deadline_LogsAndCounts` | Deadline excedido → verifica log e contador | PASS |
| `TestRetryObservability_FirstAttemptSuccess_NoRetryLogs` | Sucesso na 1a tentativa → zero logs, zero contadores | PASS |
| `TestRetryObservability_NilLoggerAndTracker_NoPanic` | Sem logger/tracker → não causa panic | PASS |

### Regressão

Todos os 17 testes pré-existentes passam sem modificação:

- 9 testes originais (S320)
- 8 testes S323 (deadline, halt, abort)

**Total: 23/23 PASS.**

## Invariantes

- **INV-OBS-1**: Sucesso na primeira tentativa emite zero sinais.
- **INV-OBS-2**: Logger e tracker nil nunca causam panic.
- **INV-OBS-3**: Nomes de contadores são estáveis (renomear é breaking change).
- **INV-OBS-4**: Mensagens de log são identificadores estáveis para grep/alertas.
- **INV-OBS-5**: Logs retry-level e actor-level são complementares, não duplicativos.

## Gaps Residuais

| ID | Gap | Risco | Nota |
|----|-----|-------|------|
| R-S320-4 | Venue error codes não usados para classificação | Baixo | Escopo separado |
| R-S320-6 | Retry-After header não parseado | Baixo | Escopo separado |
| R-S323-3 | Wiring de produção de WithHaltChecker no actor pipeline | Médio | Integração pendente |
| R-S324-1 | WithLogger/WithTracker não wired no bootstrap de produção | Baixo | Composição no main; capability pronta |
| R-S324-2 | Sem breakdown per-symbol nos contadores de retry | Baixo | Alta cardinalidade; não justificado agora |

## Preparação Recomendada para S325

S325 pode focar em:

1. **Wiring de produção** — conectar `WithLogger`, `WithTracker`, e
   `WithHaltChecker` no bootstrap do venue adapter actor (fecha R-S323-3 e
   R-S324-1).
2. **Venue error code classification** (R-S320-4) — mapear códigos Binance
   para classificação mais granular.
3. **Closure gate preparation** (S326) — se os gaps restantes forem
   suficientemente baixo risco, S325 pode preparar a evidência de fechamento
   da tranche.

## Critérios de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Venue path ganha visibilidade mínima útil sobre retries | Atendido — 5 sinais de log + 5 contadores |
| Métricas e logs têm semântica clara | Atendido — doc de referência operacional criado |
| Melhora explainability operacional sem inflar escopo | Atendido — zero sinais em first-attempt success |
| Closure tranche fica mais auditável | Atendido — retry outcomes visíveis via /statusz e logs |

## Guard Rails — Verificação

| Guard rail | Status |
|------------|--------|
| Não abrir dashboards amplos | Respeitado |
| Não criar nova observability wave | Respeitado |
| Não instrumentar tudo indiscriminadamente | Respeitado — apenas retry path |
| Não aumentar ruído sem valor operacional | Respeitado — INV-OBS-1 garante zero noise em happy path |
