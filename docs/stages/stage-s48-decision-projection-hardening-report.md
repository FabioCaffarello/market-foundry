# Stage S48: Decision Projection Hardening Report

> Endurecer o dominio `decision` com foco em projection authority, idempotencia/replay,
> invariantes latest-only, health/readiness, e query path.

## Resumo Executivo

O dominio `decision` foi construido apos o hardening de `signal` (S37) e ja incorporava
as licoes aprendidas: nil guards, monotonicity guard, health trackers registrados,
structured logging com family context, e contracts sem omitempty. Este stage formalizou
os invariantes que estavam implicitos no codigo, adicionou observabilidade incremental,
e produziu a documentacao canonica necessaria para que `decision` seja tratado como
camada madura.

**Conclusao principal:** `decision` esta estruturalmente pronto para ser avaliado
como dependencia de `strategy`. As limitacoes remanescentes sao conhecidas, documentadas,
e nenhuma bloqueia o uso atual.

## Arquivos Alterados

### Codigo

| Arquivo | Alteracao |
|---------|-----------|
| `internal/actors/scopes/store/decision_projection_actor.go` | Adicionado counter `received` ao `decisionProjectionStats` para total explicito de eventos processados |

### Documentacao (novos)

| Arquivo | Conteudo |
|---------|---------|
| `docs/architecture/decision-projection-pattern.md` | Padrao canonico de projection: pipeline, single-writer, gates, counters, bucket ownership, query path, latest-only rationale |
| `docs/architecture/decision-replay-idempotency-rules.md` | Cinco invariantes de replay (INV-1 a INV-5), safety matrix, limitacoes aceitas, partition/dedup key contracts |
| `docs/stages/stage-s48-decision-projection-hardening-report.md` | Este relatorio |

## Hardening Aplicado

### H-1: Counter `received` no Projection Actor

Adicionado `received atomic.Int64` ao `decisionProjectionStats`. O counter e
incrementado no inicio de `onDecision()`, antes de qualquer gate. Isso permite
verificar a invariante:

```
received = materialized + skipped_stale + skipped_dedup + skipped_non_final + rejected + errors
```

O counter aparece no log de stats no shutdown do actor. Antes, o total de eventos
processados so podia ser inferido somando os outros contadores.

### H-2: Projection Authority Documentada

O documento `decision-projection-pattern.md` formaliza quatro regras de authority:

1. **Store owns the read model** — `DecisionProjectionActor` e o unico writer.
2. **Gateway reads, never writes** — `QueryResponderActor` abre conexao read-only.
3. **No cross-domain writes** — signal/evidence nunca tocam buckets de decision.
4. **Validation is the projection's responsibility** — gate 2 rejeita decisions malformadas
   independentemente do que derive publicou.

### H-3: Replay/Idempotency Invariantes Formalizados

O documento `decision-replay-idempotency-rules.md` formaliza cinco invariantes:

| Invariante | Descricao | Enforcement |
|-----------|-----------|-------------|
| INV-1 | Somente decisions finalizadas entram no read model | Gate 1: `Final == false` -> skip |
| INV-2 | Toda decision materializada passa domain validation | Gate 2: `Validate()` -> reject |
| INV-3 | Latest nunca regride (monotonicity guard) | KV read-before-write |
| INV-4 | JetStream dedup previne double-publish | `DeduplicationKey()` como MsgID |
| INV-5 | Durable consumer retoma do ultimo ack | Consumer `store-decision-rsi-oversold` |

Inclui replay safety matrix com 8 cenarios e outcome esperado.

### H-4: Latest-Only Documentado como Escolha Intencional

O `decision-projection-pattern.md` documenta explicitamente que latest-only e uma
decisao de design, nao uma feature faltante:

1. Decisions sao avaliacoes efemeras — podem ser re-derivadas dos signals.
2. Simplicidade sobre completude — bucket unico e mais facil de operar e replay-safe.
3. No history until proven necessary — abrir history bucket so com caso de uso concreto.

### H-5: Validacao Ja Presente (Confirmacao)

A auditoria confirmou que o codigo de `decision` ja incorpora todos os patterns
aplicados no S37 para `signal`:

| Pattern | Status | Referencia |
|---------|--------|------------|
| Nil guards no KV store (Put + Get) | Presente | `decision_kv_store.go:62,97` |
| Monotonicity guard com read-before-write | Presente | `decision_kv_store.go:68-79` |
| ErrKeyNotFound tratado explicitamente | Presente | `decision_kv_store.go:104` |
| Health trackers registrados em `cmd/store/run.go` | Presente | `run.go:88-96` |
| Response contract sem omitempty | Presente | `contracts.go:16` |
| Structured logging com family context | Presente | `decision_projection_actor.go:48`, `decision_consumer_actor.go:39` |
| 3 gates no projection actor | Presente | `decision_projection_actor.go:92-108` |
| 6+ counters atomicos | Presente | `decision_projection_actor.go:24-31` |
| Stats log no shutdown | Presente | `decision_projection_actor.go:56` |
| Domain validation com outcome enum | Presente | `decision.go:57-64` |
| DeduplicationKey para JetStream MsgID | Presente | `decision_publisher.go:80` |
| Durable consumer com AckWait + MaxDeliver | Presente | `decision_registry.go:52-64` |

## Health/Readiness

O decision pipeline tem trackers dedicados:

- `decision-rsi-oversold-projection` — registrado em `cmd/store/run.go:89`
- `decision-rsi-oversold-consumer` — registrado em `cmd/store/run.go:89`

Ambos visveis via `/statusz`. Idle warnings apos 2 minutos de inatividade.

## Query Path

Fluxo completo verificado:

```
GET /decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60
  -> DecisionWebHandler.GetLatestDecision()
  -> GetLatestDecisionUseCase.Execute() [valida type, source, symbol, timeframe]
  -> DecisionGateway.GetLatestDecision() [NATS request/reply]
  -> decision.query.rsi_oversold.latest [queue group: decision.query]
  -> QueryResponderActor.handleDecisionRSIOversoldLatest()
  -> DecisionKVStore.Get() [read-only]
  -> DECISION_RSI_OVERSOLD_LATEST [bucket]
  -> DecisionLatestReply{Decision: *decision.Decision}
```

O gateway nunca tem acesso de escrita ao bucket. O use case valida todos os
parametros antes de chegar ao NATS.

## Testes

Todos os testes existentes passam sem alteracao:

```
ok  internal/domain/decision
ok  internal/adapters/nats
ok  internal/application/decisionclient
ok  internal/application/decision
ok  internal/interfaces/http/handlers
ok  internal/interfaces/http/routes
```

## Limitacoes Remanescentes

| Limitacao | Severidade | Impacto |
|-----------|-----------|---------|
| Latest-only (sem history bucket) | Baixa | Intencional; re-derive se necessario |
| Ack-before-projection window | Baixa | Bounded a 1 decision por partition key |
| Single-writer assumption | Baixa | Multi-writer converge via monotonicity guard |
| Somente `rsi_oversold` implementado | Media | Registry pronto para extensao (`LatestSpecByType`) |
| Sem cross-type atomicity | Baixa | Cada family independente; aceitavel |
| Confidence como string, sem validacao numerica | Baixa | Validacao apenas de presenca, nao de range |

## Preparacao Recomendada para S49

1. **Decision readiness review para `strategy`** — Avaliar se `decision` sustenta
   os requisitos de `strategy` como dependencia. Verificar se o contrato de
   `DecisionLatestReply` e suficiente para o que `strategy` precisa consumir.

2. **Segundo decision family (opcional)** — Se `strategy` precisar de mais de um
   tipo de decision, implementar um segundo family (e.g., `macd_crossover`) para
   validar a extensibilidade do registry e do pipeline pattern.

3. **Confidence range validation** — Considerar adicionar validacao de range
   (0.0 a 1.0) no `Decision.Validate()` se `strategy` depender desse valor
   como input numerico.

4. **Raccoon CLI governance** — Adicionar regras de drift para `decision` no
   raccoon-cli, analogas as existentes para `signal`.

5. **Config dependency hardening** — Garantir que `decision_families` so ativa
   se `signal_families` tambem estiver ativa (decision depende de signal no runtime).
