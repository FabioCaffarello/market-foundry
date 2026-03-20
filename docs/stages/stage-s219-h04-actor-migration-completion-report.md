# Stage S219 — H-04 Actor Migration Completion Report

> Completing the migration of store consumer actors to the generic infrastructure.

## Resumo Executivo

O Stage S219 concluiu a migração dos 9 consumer actors do store scope para a infraestrutura genérica `GenericConsumerActor` que havia sido introduzida mas nunca adotada. O resultado é a eliminação de 8 arquivos, 8 tipos Config, 8 tipos Actor e ~510 linhas de código duplicado, sem alteração de comportamento. O investimento na infraestrutura genérica agora captura valor real.

## Migração Concluída

### Consumer actors migrados (9 → 1)

Todos os 9 consumer actors domain-specific foram substituídos por `ConsumerStartFn` closures declarados em `declarePipelines()`, delegando lifecycle management ao `GenericConsumerActor`:

| Antes (arquivo deletado) | Domínio | Closure em declarePipelines() |
|---|---|---|
| `evidence_consumer_actor.go` | evidence/candle | `startConsumer("candle", ...)` |
| `trade_burst_consumer_actor.go` | evidence/tradeburst | `startConsumer("tradeburst", ...)` |
| `volume_consumer_actor.go` | evidence/volume | `startConsumer("volume", ...)` |
| `signal_consumer_actor.go` | signal/rsi + ema_crossover | `startConsumer("rsi", ...)` / `startConsumer("ema_crossover", ...)` |
| `decision_consumer_actor.go` | decision/rsi_oversold | `startConsumer("rsi_oversold", ...)` |
| `strategy_consumer_actor.go` | strategy/mean_reversion_entry | `startConsumer("mean_reversion_entry", ...)` |
| `risk_consumer_actor.go` | risk/position_exposure | `startConsumer("position_exposure", ...)` |
| `execution_consumer_actor.go` | execution/paper_order | `startConsumer("paper_order", ...)` |
| `fill_consumer_actor.go` | execution/venue_market_order | `startConsumer("venue_market_order", ...)` |

### O que conscientemente permaneceu fora

| Componente | Razão |
|---|---|
| 9 projection actors (store) | Lógica de domínio diverge: validation gates, dual-bucket writes (candle), intent cross-reference (fill), log fields específicos |
| 5 publisher actors (derive) | Fora do escopo — candidatos para consolidação futura se o padrão se estabilizar |
| Supervisors, watchers, evaluators | Não seguem o padrão consumer; não são candidatos à mesma infraestrutura |

## Arquivos Alterados

### Deletados (9 arquivos)
```
internal/actors/scopes/store/evidence_consumer_actor.go
internal/actors/scopes/store/trade_burst_consumer_actor.go
internal/actors/scopes/store/volume_consumer_actor.go
internal/actors/scopes/store/signal_consumer_actor.go
internal/actors/scopes/store/decision_consumer_actor.go
internal/actors/scopes/store/strategy_consumer_actor.go
internal/actors/scopes/store/risk_consumer_actor.go
internal/actors/scopes/store/execution_consumer_actor.go
internal/actors/scopes/store/fill_consumer_actor.go
```

### Modificados (1 arquivo)
```
internal/actors/scopes/store/store_supervisor.go
  — imports: adicionados io, domain types
  — declarePipelines(): todos os NewConsumer fields agora usam startConsumer() helper
  — adicionado startConsumer() helper local
```

### Criados (3 documentos)
```
docs/architecture/h04-actor-migration-completion.md
docs/architecture/actor-infrastructure-adoption-before-and-after.md
docs/stages/stage-s219-h04-actor-migration-completion-report.md
```

## Ganhos Arquiteturais

1. **Duplicação eliminada**: 9 actor types + 9 config types → 1 generic type. ~510 linhas de código mecânico removidas.

2. **Topologia mais previsível**: todos os consumer actors no store scope agora são instâncias de `GenericConsumerActor`. Não há variações estruturais — apenas variações de comportamento capturadas em closures.

3. **Custo de adição de pipeline reduzido**: adicionar um novo pipeline requer apenas 1 closure de ~8 linhas em `declarePipelines()`, vs. criar arquivo + Config + Actor + Constructor + Receive + start (~90 linhas).

4. **Investimento na infraestrutura captura valor**: `GenericConsumerActor` deixou de ser código morto e passou a ser o único caminho de criação de consumer actors no store.

5. **Module graph simplificado**: `store_supervisor.go` agora importa os domain types diretamente (para construir os closures), mas não depende mais de 9 tipos intermediários que existiam apenas como cola.

## Limites e Trade-offs

1. **Closures são opacos para tooling**: ferramentas de análise estática que procuram tipos concretos de actor não encontrarão `EvidenceConsumerActor` etc. — encontrarão apenas `GenericConsumerActor`. Para observabilidade em runtime, o campo `Family` no config preserva a identidade.

2. **Projection actors não foram consolidados**: a decisão de não generalizar projection actors é consciente — a lógica de domínio diverge o suficiente para justificar implementações separadas. Revisitar se o número de projection actors crescer significativamente.

3. **Derive scope não foi tocado**: publisher actors no derive scope seguem padrão similar mas têm diferenças suficientes (fan-out, publication semantics) que justificam análise separada.

## Verificação

| Check | Resultado |
|---|---|
| `go build internal/actors/scopes/store` | Clean |
| `go vet internal/actors/scopes/store/...` | Clean |
| `go test internal/actors/scopes/store/...` | All pass |

## Preparação para S220

O S219 deixa a base pronta para a simplificação do module graph no S220:

1. **Consumer actor types eliminados**: 9 tipos que criavam dependências cruzadas desnecessárias foram removidos. O module graph do store scope agora tem menos nós.

2. **`declarePipelines()` é o único ponto de wiring**: toda a configuração de consumer → projection está centralizada. Qualquer reorganização de módulos no S220 tem um único ponto de impacto.

3. **Projection actors são auto-contidos**: cada projection actor depende apenas do seu domain type e do respectivo NATS KV store adapter. Não há dependências laterais entre projection actors.

4. **Candidatos para S220**: a consolidação dos publisher actors no derive scope é um candidato natural para a mesma técnica (closure-captured `PublisherStartFn`), e pode ser incluída na simplificação do module graph.
