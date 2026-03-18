# Stage S56 — Strategy First Slice

## Resumo executivo

O primeiro vertical slice do domínio `strategy` foi implementado com sucesso no Market Foundry. A family `mean_reversion_entry` (STF-01) está funcional ponta a ponta: derive resolve strategies a partir de decisions, store materializa no read model, e gateway expõe via HTTP.

## Pré-condições validadas

| Stage | Status | Verificação |
|---|---|---|
| S53 — Strategy Domain Design | Completo | Contratos, stream families, e rules definidos |
| S54 — Strategy Governance Activation | Completo | raccoon-cli com drift rules e guardrails ativos |
| S55 — Strategy Implementation Readiness | Completo | `strategy_families` em schema.go com validação e dependências |

## Family escolhida

**`mean_reversion_entry`** (STF-01) — conforme S53.

**Justificativa:**
- Resolução mais simples: single-decision threshold → directional intent
- Prova o pipeline strategy ponta a ponta com risco mínimo
- Espelha o padrão onde `rsi_oversold` foi a primeira decision family
- Dependência única e madura: `rsi_oversold`

## Arquivos criados

### Domain layer
- `internal/domain/strategy/strategy.go` — Strategy entity com Direction, Validate, PartitionKey, DeduplicationKey
- `internal/domain/strategy/events.go` — StrategyResolvedEvent
- `internal/domain/strategy/strategy_test.go` — 13 testes

### Application layer
- `internal/application/strategy/mean_reversion_entry_resolver.go` — Resolver puro: decision → strategy
- `internal/application/strategy/mean_reversion_entry_resolver_test.go` — 9 testes
- `internal/application/strategyclient/contracts.go` — StrategyLatestQuery, StrategyLatestReply
- `internal/application/strategyclient/get_latest_strategy.go` — GetLatestStrategyUseCase
- `internal/application/strategyclient/get_latest_strategy_test.go` — 3 testes
- `internal/application/ports/strategy.go` — StrategyGateway interface

### NATS adapters
- `internal/adapters/nats/strategy_registry.go` — STRATEGY_EVENTS stream, subjects, consumer specs
- `internal/adapters/nats/strategy_registry_test.go` — 6 testes
- `internal/adapters/nats/strategy_publisher.go` — Publica no STRATEGY_EVENTS
- `internal/adapters/nats/strategy_consumer.go` — Consome com durable consumer
- `internal/adapters/nats/strategy_gateway.go` — NATS request/reply para gateway
- `internal/adapters/nats/strategy_kv_store.go` — KV store com monotonicity guard
- `internal/adapters/nats/strategy_kv_store_test.go` — 8 testes

### Actors — derive
- `internal/actors/scopes/derive/strategy_resolver_actor.go` — MeanReversionEntryResolverActor
- `internal/actors/scopes/derive/strategy_publisher_actor.go` — StrategyPublisherActor

### Actors — store
- `internal/actors/scopes/store/strategy_consumer_actor.go` — StrategyConsumerActor
- `internal/actors/scopes/store/strategy_projection_actor.go` — StrategyProjectionActor (three-gate)

### HTTP interface
- `internal/interfaces/http/handlers/strategy.go` — StrategyWebHandler
- `internal/interfaces/http/handlers/strategy_test.go` — 4 testes
- `internal/interfaces/http/routes/strategy.go` — Route registration
- `internal/interfaces/http/routes/strategy_test.go` — 3 testes

### Tests & docs
- `tests/http/strategy.http` — Manual test file
- `docs/architecture/strategy-first-slice.md` — Documentação arquitetural

## Arquivos modificados

### Derive supervisor & scope
- `internal/actors/scopes/derive/derive_supervisor.go` — StrategyFamilyProcessor, strategy registry, strategy processors
- `internal/actors/scopes/derive/source_scope_actor.go` — StrategyFamilyProcessor type, strategy publisher, strategy resolvers, routeDecisionToStrategy
- `internal/actors/scopes/derive/decision_evaluator_actor.go` — ScopePID para fan-out de decisions para strategy
- `internal/actors/scopes/derive/messages.go` — decisionEvaluatedMessage, publishStrategyMessage

### Store supervisor & query responder
- `internal/actors/scopes/store/store_supervisor.go` — StrategyPipeline type, strategy pipeline registration
- `internal/actors/scopes/store/query_responder_actor.go` — Strategy registry, KV store, query handler
- `internal/actors/scopes/store/messages.go` — strategyReceivedMessage
- `internal/actors/scopes/store/projection_store.go` — strategyProjectionStore interface

### HTTP routes
- `internal/interfaces/http/routes/core.go` — StrategyFamilyDeps, strategy route wiring

### Binary wiring
- `cmd/gateway/run.go` — Strategy gateway initialization e use case wiring
- `cmd/gateway/gateway.go` — newStrategyGateway factory
- `cmd/store/run.go` — Strategy pipeline trackers

### Configuration
- `deploy/configs/derive.jsonc` — `strategy_families: ["mean_reversion_entry"]` ativado
- `deploy/configs/store.jsonc` — `strategy_families: ["mean_reversion_entry"]` ativado

## Stream contracts implementados

| Contrato | Valor |
|---|---|
| Stream | `STRATEGY_EVENTS` (72h, 2GB, file-backed) |
| Event subject | `strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}` |
| Query subject | `strategy.query.mean_reversion_entry.latest` |
| KV bucket | `STRATEGY_MEAN_REVERSION_ENTRY_LATEST` |
| Durable consumer | `store-strategy-mean-reversion-entry` |
| HTTP endpoint | `GET /strategy/:type/latest?source=X&symbol=Y&timeframe=Z` |
| Dedup key prefix | `strat:` |

## Resultados dos testes

| Pacote | Testes | Status |
|---|---|---|
| `internal/domain/strategy` | 13 | PASS |
| `internal/application/strategy` | 9 | PASS |
| `internal/application/strategyclient` | 3 | PASS |
| `internal/adapters/nats` (strategy) | 14 | PASS |
| `internal/interfaces/http/handlers` (strategy) | 4 | PASS |
| `internal/interfaces/http/routes` (strategy) | 3 | PASS |
| **Total** | **46** | **PASS** |

Zero regressões em testes existentes (nota: `configctl` tem falha pré-existente não relacionada).

## Limites encontrados

1. **DecisionFamilyProcessor signature change**: O `NewActor` de `DecisionFamilyProcessor` recebeu um parâmetro adicional `scopePID` para permitir fan-out de decisions para strategy resolvers. Essa mudança é backward-compatible pois o `ScopePID` é nilável.

2. **Flat as valid output**: `DirectionFlat` é um output válido de strategy (significa "sem recomendação de trade"), conforme S53. Não é um erro.

## Itens explicitamente adiados para S57+

1. **Strategy history queries** — Apenas latest-only em Phase 1
2. **Famílias adicionais** — `macd_momentum_entry` (STF-02), `confluence_entry` (STF-03)
3. **Multi-decision patterns** — Estratégias que combinam múltiplas decisions
4. **Domínios upstream** — `risk`, `execution`, `portfolio` não foram abertos
5. **Strategy-specific smoke test** — Integração em `scripts/smoke-first-slice.sh`
6. **Projection actor tests** — Unit tests para `StrategyProjectionActor` com mock store
7. **ClickHouse integration** — Analytical storage para strategy events

## Critérios de aceite — verificação

| Critério | Status |
|---|---|
| Strategy family mínima funcionando ponta a ponta | OK |
| `strategy` claramente separada de `decision` | OK — domínio não importa decision |
| `store` continua como authority do read-side | OK |
| `gateway` permanece limpo | OK — sem lógica de domínio |
| First slice prova o novo domínio sem inflar complexidade | OK |
| Implementação coerente com docs do S53 | OK |
