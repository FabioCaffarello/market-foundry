# Stage S72 — Execution Projection Hardening Report

## Resumo Executivo

O S72 endureceu o domínio `execution` consolidando os padrões de projection authority, idempotência/replay, invariantes latest-only, observabilidade mínima e query path confiável. Todas as mudanças são internas ao domínio — nenhuma integração com venue real foi tocada.

O foco foi alinhar `execution` com a maturidade estrutural já alcançada por `risk`, `strategy`, `decision` e demais famílias, tornando-o explicável, auditável e seguro para replay.

## Arquivos Alterados

### Runtime

| Arquivo | Mudança |
|---------|---------|
| `internal/actors/scopes/store/execution_projection_actor.go` | Extração de `start()`, `checkStatsInvariant()`, `logStats()`. Logging de projection authority e materialização com traceability completa. Remoção de interface não utilizada. Alinhamento com padrão risk. |
| `internal/actors/scopes/derive/execution_publisher_actor.go` | `defer cancel()` no contexto de publicação (alinhamento com padrão canônico). |
| `internal/domain/execution/execution.go` | `ValidSide()`, `ValidStatus()` helpers. Validação de status desconhecido no `Validate()`. |

### Documentação

| Arquivo | Conteúdo |
|---------|----------|
| `docs/architecture/execution-projection-pattern.md` | Projection authority, three-gate pipeline, stats invariant, query path, ownership boundaries. |
| `docs/architecture/execution-replay-idempotency-rules.md` | Três camadas de idempotência (JetStream dedup, durable consumer, KV monotonicity), replay safety, limitações operacionais. |
| `docs/stages/stage-s72-execution-projection-hardening-report.md` | Este relatório. |

## Hardening Aplicado

### 1. Projection Authority (explícita)

O `ExecutionProjectionActor` agora loga no startup:
- `projection_authority: execution-paper_order-projection`
- `semantics: latest-only`
- `bucket_latest: EXECUTION_PAPER_ORDER_LATEST`

Isso torna inequívoco quem é o sole writer do bucket e qual é a semântica de persistência.

### 2. Stats Invariant (safeguard dedicado)

Extraído `checkStatsInvariant()` como método dedicado que loga em nível ERROR se `received != sum of outcomes`. Anteriormente, o invariante era verificado inline no handler de Stopped sem log explícito de violação.

### 3. Materialization Logging (traceability)

Cada materialização bem-sucedida agora loga:
- `type`, `source`, `symbol`, `timeframe`
- `side`, `quantity`, `status`
- `risk_disposition`
- `timestamp`, `correlation_id`, `causation_id`

Isso permite rastrear a cadeia causal completa: observation → evidence → signal → decision → strategy → risk → execution.

### 4. Domain Validation (endurecida)

- `ValidSide()` e `ValidStatus()` como funções públicas reutilizáveis.
- `Validate()` agora rejeita status desconhecidos (proteção contra extensão não controlada).

### 5. Context Handling (canônico)

Publisher e projection agora usam `defer cancel()` consistentemente, alinhados com o padrão de risk e demais famílias.

### 6. Replay/Idempotência (documentada)

As três camadas de idempotência foram formalizadas:
1. **JetStream MsgID** — dedup na publicação
2. **Durable consumer** — ACK persistente, redelivery limitado
3. **KV monotonicity guard** — timestamp-based stale/dedup rejection

## Limitações Restantes

| Limitação | Status | Nota |
|-----------|--------|------|
| Sem history bucket | Intencional | Latest-only é escolha de design. History pode ser adicionado em stage futuro. |
| Sem snapshot/rebuild automatizado | Conhecido | Rebuild requer replay do stream (72h retention). |
| Sem métricas Prometheus | Pendente | Stats são logadas em structured logging; métricas exportáveis podem vir em stage futuro. |
| Sem circuit breaker no KV write | Conhecido | Erros são contados, mas não há backoff automático. Volume atual não justifica. |
| Sem validação cruzada com risk | Intencional | Execution recebe dados primitivos do risk, não importa structs — domain isolation preservada. |

## Preparação Recomendada para S73/S74

### S73 — Execution Observability & Diagnostics
- Adicionar endpoint `/execution/paper_order/stats` para exposição das stats de projeção via HTTP.
- Considerar métricas Prometheus para materialização, stale, dedup, rejected.
- Avaliar healthcheck dedicado para a projeção de execution (bucket writeable + consumer lag).

### S74 — Execution Query Surface Expansion
- Avaliar necessidade de history bucket para execution (paper_order history por símbolo).
- Considerar query de "last N executions" para auditoria operacional.
- Avaliar se o gateway precisa de um endpoint de status do pipeline de execution.

### Pré-requisito para qualquer integração com venue
- Execution precisa ter stats observáveis externamente (não apenas em logs).
- Execution precisa ter readiness probe que confirma: consumer ativo + bucket acessível.
- Execution precisa ter documentação de failure modes antes de tocar qualquer venue real.
