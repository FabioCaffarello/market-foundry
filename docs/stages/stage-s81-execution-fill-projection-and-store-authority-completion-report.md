# Stage S81 — Execution Fill Projection and Store Authority Completion

**Status**: complete
**Date**: 2026-03-18
**Predecessor**: S80 (first guarded venue execution step)

## 1. Resumo Executivo

O S80 abriu o primeiro passo guardado de venue execution em paper mode: o `execute` consome intents do derive, submete ao paper venue adapter, e publica `VenueOrderFilledEvent` no stream `EXECUTION_FILL_EVENTS`. Entretanto, o `store` não possuía consumer nem projection para esses fill events — a malha estava aberta no lado do read-side.

O S81 fecha essa malha. O `store` agora consome fill events do `EXECUTION_FILL_EVENTS`, materializa o resultado no bucket `EXECUTION_VENUE_MARKET_ORDER_LATEST`, e expõe via query para o gateway. A separação semântica entre intent (derive) e fill (execute) está preservada em buckets distintos.

## 2. Malha de Fill Projection Introduzida

### Fluxo Completo

```
execute binary
  ↓ publishes VenueOrderFilledEvent
EXECUTION_FILL_EVENTS stream (72h retention, 2GB)
  ↓
FillConsumer (durable: "store-execution-venue-market-order-fill")
  ↓ decodes VenueOrderFilledEvent via CBOR envelope
FillConsumerActor (store scope)
  ↓ sends fillReceivedMessage to projection PID
FillProjectionActor (store scope)
  ├─ Gate 1: finality (skip non-final)
  ├─ Gate 2: validation (reject malformed)
  └─ Gate 3: monotonicity (timestamp guard)
       ↓
EXECUTION_VENUE_MARKET_ORDER_LATEST KV bucket (64MB, FileStorage)
  ↓
QueryResponderActor serves "execution.query.venue_market_order.latest"
  ↓
Gateway HTTP: GET /execution/venue_market_order/latest?source=X&symbol=Y&timeframe=Z
```

### Componentes Criados

| Componente | Arquivo | Papel |
|-----------|---------|-------|
| `FillConsumer` | `internal/adapters/nats/fill_consumer.go` | JetStream consumer para `VenueOrderFilledEvent` |
| `FillConsumerActor` | `internal/actors/scopes/store/fill_consumer_actor.go` | Actor wrapper que encaminha fills ao projection |
| `FillProjectionActor` | `internal/actors/scopes/store/fill_projection_actor.go` | Materializa fills no KV bucket |
| `fillReceivedMessage` | `internal/actors/scopes/store/messages.go` | Mensagem interna consumer→projection |

### Componentes Modificados

| Componente | Arquivo | Mudança |
|-----------|---------|---------|
| `ExecutionRegistry` | `internal/adapters/nats/execution_registry.go` | Adicionado `StoreVenueMarketOrderFillConsumer()` spec |
| `StoreSupervisor` | `internal/actors/scopes/store/store_supervisor.go` | Pipeline `venue_market_order` registrado |
| `QueryResponderActor` | `internal/actors/scopes/store/query_responder_actor.go` | KV store e route para venue_market_order latest |
| Store `Run()` | `cmd/store/run.go` | Tracker defs para venue_market_order projection/consumer |
| Store config | `deploy/configs/store.jsonc` | `venue_market_order` adicionado a `execution_families` |

### Testes

| Arquivo | Cobertura |
|---------|-----------|
| `fill_projection_actor_test.go` | 11 testes: gates (final, validation), put results (written, stale, dedup, error), stats invariant, multi-symbol isolation, venue order ID, error tracking |

## 3. Arquivos Alterados

### Criados
- `internal/adapters/nats/fill_consumer.go`
- `internal/actors/scopes/store/fill_consumer_actor.go`
- `internal/actors/scopes/store/fill_projection_actor.go`
- `internal/actors/scopes/store/fill_projection_actor_test.go`
- `docs/architecture/execution-fill-projection-pattern.md`
- `docs/architecture/execution-read-side-authority-after-execute.md`

### Modificados
- `internal/adapters/nats/execution_registry.go` — novo consumer spec
- `internal/actors/scopes/store/messages.go` — `fillReceivedMessage`
- `internal/actors/scopes/store/store_supervisor.go` — pipeline venue_market_order
- `internal/actors/scopes/store/query_responder_actor.go` — venue_market_order KV store + query route
- `cmd/store/run.go` — tracker defs
- `deploy/configs/store.jsonc` — execution_families

## 4. Limites Remanescentes

| Limite | Justificativa |
|--------|--------------|
| **Latest-only** | Sem histórico de fills. Cada partition key retém apenas o fill mais recente. Suficiente para paper mode. |
| **Paper mode exclusivo** | O consumer e projection lidam apenas com fills simulados. Fills reais seguirão o mesmo padrão com gates adicionais. |
| **Sem agregação** | Fills não são agregados entre símbolos ou timeframes. Cada partição é independente. |
| **Sem journal** | Não há log de auditoria persistente além do stream (72h retention). Fill history amplo é explicitamente fora de escopo. |
| **VenueOrderID não no KV** | O `venue_order_id` está no evento mas não é preservado no KV value (que armazena apenas `ExecutionIntent`). Suficiente por ora. |
| **Sem cross-bucket consistency** | Os buckets `PAPER_ORDER_LATEST` e `VENUE_MARKET_ORDER_LATEST` evoluem independentemente. Sem transação atômica entre eles. |

## 5. Preparação Recomendada para S82

### Opção A: Operational Validation End-to-End
Validar que o fluxo completo `derive → execute → store → gateway` funciona corretamente em multi-symbol paper mode. Verificar:
- Fill events de execute chegam ao store sem perda
- KV bucket materializa corretamente para múltiplos símbolos
- Gateway retorna fills via HTTP
- Smoke test multi-symbol exercita ambos os buckets

### Opção B: Fill History e Auditabilidade
Se a necessidade de auditabilidade crescer, considerar adicionar um bucket de histórico para fills (análogo a `CandleHistoryBucket`). Isso não é urgente enquanto o stream tem 72h de retention.

### Opção C: Venue Integration Hardening
Preparar a transição de paper fills para real venue fills, adicionando:
- Validação de preço (price != "0")
- Validação de simulated flag (false para real)
- Gates adicionais para real venue responses

**Recomendação**: Opção A (operational validation) é o passo natural — fechar a validação antes de expandir.

## 6. Critérios de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Fill events de execute consumidos pelo store | OK — `FillConsumer` + `FillConsumerActor` |
| Read-side canônico consultável | OK — `EXECUTION_VENUE_MARKET_ORDER_LATEST` + query route |
| Store permanece authority do read-side | OK — sole writer pattern mantido |
| Gateway não assume responsabilidades indevidas | OK — gateway apenas roteia queries |
| Malha mais completa sem inflar escopo | OK — padrão idêntico às demais projections |

## 7. Guard Rails — Verificação

| Guard rail | Status |
|------------|--------|
| Não abrir venue real | OK — paper mode exclusivo |
| Não criar history/journal amplo | OK — latest-only semantics |
| Não transformar em OMS | OK — sem partial fill tracking, cancel/amend |
| Não colapsar intent e fill | OK — buckets separados, projections separadas |
| Documentar limites | OK — seção 4 acima |
