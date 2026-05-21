# MarketMonkey → Market Foundry: Sequenciamento de Adoção

> Documento complementar à matriz de tradução.
> Define prioridades, proibições e sequência de incorporação futura.

---

## 1. Prioridades de Adoção

### Prioridade 1 — Fundação de Domínio (pré-requisito para tudo)

Antes de qualquer código de ingestão ou derivação, o Market Foundry precisa de:

| Item | Descrição | Justificativa |
|------|-----------|---------------|
| **Domínio `observation`** | Tipos puros: `MarketInstrument`, `ObservationTrade`, `ObservationBookUpdate`, `ObservationBookSnapshot`, `ObservationExchangeStat` | Sem domínio, não há contrato. Sem contrato, não há actor. |
| **Domínio `evidence`** | Tipos puros: `EvidenceCandle`, `EvidenceVolumeProfile`, `EvidenceStatAggregate`, `ProjectionOrderbook`, `ProjectionHeatmap` | Idem — domínio antes de infra |
| **Subject registry** | Registro centralizado de todos os subjects NATS dos novos domínios, validável pelo raccoon-cli | Evita drift de naming desde o início |
| **Stream definitions** | JetStream streams: `OBSERVATION_MARKET_EVENTS`, `EVIDENCE_REALTIME_EVENTS`, `EVIDENCE_FINALIZED_EVENTS`, `EVIDENCE_PROJECTION_EVENTS` | Streams devem existir antes dos publishers |
| **Envelope contracts** | `Envelope[ObservationTrade]`, `Envelope[EvidenceCandle]`, etc. — tipagem forte para cada mensagem | Contratos são a fronteira entre domínios |

**Referência MM:** Tipos em `event/event.go`. Traduzir semanticamente, não copiar.

### Prioridade 2 — Ingestão (`ingest` binary)

| Item | Descrição | Referência MM |
|------|-----------|---------------|
| **IngestSupervisor** | Actor raiz do binário `ingest` | Não existe no MM (era flat) — criar com padrão MF |
| **ExchangeConnectorActor** | 1 actor por exchange configurada, gerencia pool de WebSocket workers | `actor/consumer/` — adaptar, não copiar |
| **Exchange adapters** | Parsers específicos por exchange (Binance, Bybit, etc.) | `actor/consumer/binancef/`, etc. |
| **NATS publisher** | Publica `ObservationTrade`, `ObservationBookUpdate`, etc. com `Envelope[T]` | `pkg/nats/producer.go` |
| **Configuração via configctl** | Exchanges, symbols e tickSizes vêm do configctl, não de config.yml | `config.yml` → substituir por ConfigSet |

**Dependências:** Prioridade 1 completa + configctl operacional (já existe).

### Prioridade 3 — Derivação core (`derive` binary, candles apenas)

| Item | Descrição | Referência MM |
|------|-----------|---------------|
| **DeriveSupervisor** | Actor raiz do binário `derive` | Não existe no MM — criar com padrão MF |
| **ExchangeScopeActor** | 1 actor por exchange, roteia para symbols | Similar ao Processor do MM |
| **SymbolScopeActor** | 1 actor por symbol, supervisiona samplers | `actor/symbol/` |
| **CandleSamplerActor** | Amostragem OHLCV multi-timeframe | `actor/trade/` — algoritmo é referência forte |
| **Lógica de Final/Sampled** | Separação de candles realtime vs finalizados em subjects distintos | `Final` flag do MM → subject split no MF |

**Por que candles primeiro:** É o derivado mais simples, mais testável, e mais valioso. Valida todo o pipeline observation → evidence sem complexidade de orderbook.

### Prioridade 4 — Derivação estendida (volume, stats)

| Item | Descrição | Referência MM |
|------|-----------|---------------|
| **VolumeProfilerActor** | Volume profile por nível de preço | `actor/volume/` |
| **StatAggregatorActor** | Agregação de métricas (trades/s, etc.) | `actor/stat/` |
| **PriceGroup calculation** | Binning por tick size | Lógica em `actor/volume/` |
| **Throttled publishing** | Publicação a cada 250ms (volume) | Timer-based publish no actor |

**Dependências:** Prioridade 3 completa e validada.

### Prioridade 5 — Orderbook e Heatmap

| Item | Descrição | Referência MM |
|------|-----------|---------------|
| **OrderbookProjectorActor** | Manutenção de livro com B-tree, 2048 níveis, depth management | `actor/orderbook/` |
| **HeatmapProjectorActor** | Agregação do orderbook em bins para visualização | `actor/orderbook/` (mesma package no MM) |
| **Snapshot vs Delta** | Tratamento distinto de `book.snapshot` e `book.delta` | `BookUpdate.Snapshot` flag no MM |

**Dependências:** Prioridade 4 completa. Orderbook é o componente mais complexo e estado-pesado.

### Prioridade 6 — Persistência (`store` binary)

| Item | Descrição | Referência MM |
|------|-----------|---------------|
| **StoreSupervisor** | Actor raiz do binário `store` | Não existe como supervisor no MM |
| **Store actors por tipo** | CandleStoreActor, VolumeStoreActor, etc. | `actor/store/` — flat no MM |
| **Repository ports** | Interfaces de persistência (port, não implementação) | `pkg/db/db.go` → ports/adapters MF |
| **DB adapter** | Implementação específica (ClickHouse ou TimescaleDB) | `pkg/db/clickhouse/`, `pkg/db/timescale/` |

**Dependências:** Prioridades 3-5 publicando eventos finalizados.

### Prioridade 7 — WebSocket (extensão do gateway)

| Item | Descrição | Referência MM |
|------|-----------|---------------|
| **WebSocket handler** | Upgrade HTTP → WS no gateway existente | `actor/server/` |
| **Subscription manager** | Gerenciar consumers NATS efêmeros por client | `actor/server_router/` |
| **Session actor** | 1 actor por conexão WebSocket | `actor/server_session/` |
| **Client protocol** | Subscribe/unsubscribe/getrange via WS | Protocolo JSON do MM — adaptar |

**Dependências:** Pipeline completo (ingest → derive → streams populados).

---

## 2. Itens Proibidos de Absorção Ingênua

### 2.1 Proibições Absolutas (não importar sob nenhuma circunstância)

| Item | Razão | Risco |
|------|-------|-------|
| **`config.yml` como arquivo estático** | MF tem configctl como domínio. Configuração é cidadão de primeira classe com lifecycle (draft → validate → compile → activate). Importar config.yml é regredir ao modelo MM onde config é um blob sem versionamento. | Viola princípio 5 (Configuration as Domain Object) |
| **Consul service discovery** | MF é NATS-native. Não há necessidade de service discovery externo quando actors se comunicam via NATS subjects. | Dependência externa sem valor |
| **1 binary por exchange** | MM usa processos separados por exchange por limitação do modelo (sem supervisor tree robusto). MF usa Hollywood actors com supervisão hierárquica — a exchange é uma dimensão de roteamento, não de deployment. | Proliferação de binários, complexidade operacional desnecessária |
| **`gorilla/websocket` direto em actors** | Actors do MM misturam WebSocket I/O com lógica de negócio. No MF, WebSocket é concern de interface (`internal/interfaces/`), actors são concern de orquestração (`internal/actors/`). | Violação de layer sovereignty |
| **Echo HTTP framework** | O gateway MF já possui seu stack HTTP. Importar Echo cria duplicação e conflito de routing. | Duplicação de infraestrutura |
| **Supabase JWT auth inline** | Auth é concern transversal. No MM está hardcoded no server actor. No MF, deve ser middleware de interface, não lógica de actor. | Acoplamento de auth no lugar errado |
| **`pkg/db` como client interface direto** | No MM, `db.Client` é interface com métodos `Insert*`/`Get*` chamados diretamente pelos actors. No MF, persistência é port no application layer, implementada por adapter — actors nunca tocam DB diretamente. | Violação de ports/adapters |
| **Métricas Prometheus inline** | No MM, `prometheus.NewHistogram()` aparece dentro dos actors. No MF, observabilidade deve ser aspecto, não código inline em actors de domínio. | Poluição do domínio com infra |
| **`cmd/backfill`, `cmd/history`, `cmd/sync`** | Tooling específico do produto MM, com assunções sobre schema de DB e formato de dados que não existem no MF. | Importar complexidade e assunções falsas |
| **Subject naming em UPPER case** | `trades.BINANCEF.BTCUSDT` — o MF usa lowercase canônico em toda a taxonomia de subjects. | Inconsistência |
| **CBOR encoding sem envelope** | MM codifica structs diretamente em CBOR. MF exige `Envelope[T]` com metadata de rastreabilidade (correlation, causation, timestamp, source). | Perda de observabilidade |
| **`event/encoding.go` helpers** | Helpers de encoding do MM assumem encoding direto. No MF, encoding é responsabilidade do codec no adapter NATS. | Acoplamento de serialização no domínio |

### 2.2 Proibições de Padrão (não reproduzir o pattern)

| Padrão MM | Por quê proibido no MF |
|----------|----------------------|
| **Actor com dependência direta de NATS** | No MM, actors importam `pkg/nats` diretamente. No MF, actors recebem ports injetados (não sabem que estão falando com NATS). |
| **Flat struct como evento de domínio** | No MM, `Trade` é struct plana sem metadata. No MF, todo evento de domínio implementa `events.Event` com `EventName()` e `EventMetadata()`. |
| **Boolean para estado de lifecycle** | `Final=true/false` mistura dado com estado. No MF, usar subjects distintos (`.sampled.` vs `.finalized.`). |
| **Snapshot/Delta como flag** | `BookUpdate.Snapshot=true/false` é ambíguo. No MF, usar eventos distintos: `book.snapshot` e `book.delta`. |
| **Timer-based publish sem backpressure** | Volume actor publica a cada 250ms independente de consumers. No MF, considerar backpressure via NATS flow control. |
| **Pool de goroutines para WebSocket** | MM usa goroutine pools para WS connections. No MF, cada conexão é um actor com lifecycle gerenciado pelo supervisor. |

### 2.3 Zona Cinzenta (avaliar caso a caso)

| Item | Consideração |
|------|-------------|
| **B-tree para orderbook** | A biblioteca `tidwall/btree` é boa, mas é dependência externa. Avaliar se `sync.Map` ou sorted slice não atende para os volumes do MF. |
| **2048 níveis de orderbook** | Heurística prática do MM. Pode ser parâmetro configurável no MF em vez de constante. |
| **Depth management ±10%** | Heurística validada. Absorver o conceito mas tornar os percentuais configuráveis. |
| **Publish intervals (200ms, 250ms)** | Valores validados em produção no MM. Usar como defaults, mas tornar configuráveis via configctl. |
| **CBOR como formato** | MF já usa CBOR — confirmar se é a melhor escolha para os volumes de dados de mercado (vs protobuf). |

---

## 3. Sequência Recomendada de Incorporação

```
Phase 0 (atual)
│ configctl + gateway operacionais
│ ✅ Completo
│
├─ Phase 2a: Domínios puros
│  │ Criar internal/domain/observation/ e internal/domain/evidence/
│  │ Definir tipos, eventos, contratos — ZERO infraestrutura
│  │ Validar com raccoon-cli (arch-guard, contract-audit)
│  │
│  │ Entregáveis:
│  │  - internal/domain/observation/instrument.go (MarketInstrument)
│  │  - internal/domain/observation/events.go (ObservationTrade, ObservationBookUpdate, etc.)
│  │  - internal/domain/evidence/events.go (EvidenceCandle, EvidenceVolumeProfile, etc.)
│  │  - internal/domain/evidence/projections.go (ProjectionOrderbook, ProjectionHeatmap)
│  │
│  │ Critério de saída: domain code compila, testes passam, quality-gate green
│  │
├─ Phase 2b: Stream infrastructure
│  │ Registrar subjects NATS para observation e evidence
│  │ Definir streams JetStream
│  │ Criar adapters NATS para os novos domínios
│  │
│  │ Entregáveis:
│  │  - internal/adapters/nats/observation_registry.go
│  │  - internal/adapters/nats/evidence_registry.go
│  │  - internal/adapters/nats/observation_publisher.go
│  │  - Stream definitions em deploy/nats/ ou bootstrap code
│  │
│  │ Critério de saída: streams criados, contract-audit valida subjects
│  │
├─ Phase 2c: Ingest binary (observation capture)
│  │ cmd/ingest com IngestSupervisor → ExchangeConnectorActors
│  │ Começar com 1 exchange (binancef) e 1 symbol (btcusdt)
│  │ Validar que trades chegam no stream observation corretamente
│  │
│  │ Referência MM: actor/consumer/binancef/ — traduzir, não copiar
│  │
│  │ Entregáveis:
│  │  - cmd/ingest/main.go
│  │  - internal/actors/scopes/ingest/supervisor.go
│  │  - internal/actors/scopes/ingest/exchange_connector.go
│  │  - internal/application/ingest/ (use cases de bootstrap)
│  │
│  │ Critério de saída: `nats sub "observation.events.market.trade.binancef.btcusdt"`
│  │  retorna trades em tempo real com envelope correto
│  │
├─ Phase 2d: Derive binary (candles only)
│  │ cmd/derive com DeriveSupervisor → ExchangeScope → SymbolScope → CandleSampler
│  │ Consumir de observation, publicar evidence candles
│  │ Multi-timeframe: pelo menos 1s, 60s, 5m
│  │
│  │ Referência MM: actor/trade/ (CandleSampler logic) — traduzir algoritmo
│  │
│  │ Entregáveis:
│  │  - cmd/derive/main.go
│  │  - internal/actors/scopes/derive/supervisor.go
│  │  - internal/actors/scopes/derive/exchange_scope.go
│  │  - internal/actors/scopes/derive/symbol_scope.go
│  │  - internal/actors/scopes/derive/candle_sampler.go
│  │  - internal/application/derive/ (use cases)
│  │
│  │ Critério de saída: `nats sub "evidence.events.candle.sampled.binancef.btcusdt.60"`
│  │  retorna candles realtime; `evidence.events.candle.finalized.*` retorna candles finais
│  │
├─ Phase 3a: Derive extensions (volume, stats)
│  │ Adicionar VolumeProfilerActor e StatAggregatorActor ao derive
│  │
│  │ Referência MM: actor/volume/, actor/stat/
│  │
├─ Phase 3b: Derive orderbook + heatmap
│  │ Componente mais complexo — orderbook com B-tree e depth management
│  │
│  │ Referência MM: actor/orderbook/
│  │
├─ Phase 3c: Multi-exchange
│  │ Expandir ingest e derive para Bybit, Coinbase, etc.
│  │ 1 exchange por iteração, validar end-to-end antes da próxima
│  │
├─ Phase 3d: Store binary
│  │ Persistência de eventos finalizados em time-series DB
│  │ Ports no application layer, adapter para DB escolhido
│  │
│  │ Referência MM: actor/store/, pkg/db/ — padrão ports/adapters do MF
│  │
└─ Phase 4: WebSocket + Client API
   │ Extensão do gateway com WebSocket
   │ Subscription management actors
   │ Protocolo de comunicação client ↔ server
   │
   │ Referência MM: actor/server/, actor/server_router/, actor/server_session/
```

---

## 4. Critérios de Validação por Phase

| Phase | Quality Gate | Validação |
|-------|-------------|-----------|
| 2a | `make check` green | Domínios compilam, sem imports de infra, arch-guard passa |
| 2b | `make check` + contract-audit | Subjects registrados, streams definidos, sem drift |
| 2c | `make verify` + teste manual | Trades fluem de exchange → NATS com envelope correto |
| 2d | `make verify` + teste manual | Candles derivados aparecem nos subjects corretos |
| 3a-b | `make verify` | Volumes/stats/orderbook nos subjects corretos |
| 3c | `make verify` por exchange | Cada exchange validada end-to-end |
| 3d | `make verify` + teste de persistência | Dados finalizados persistidos e consultáveis |
| 4 | `make check-deep` + teste manual WS | Client conecta, subscreve, recebe dados em tempo real |

---

## 5. Riscos e Mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Copiar código do MM por conveniência | Alta | Alto — polui arquitetura | Code review rigoroso; raccoon-cli arch-guard; proibição explícita em CLAUDE.md |
| Envelope overhead em alta frequência | Média | Médio — latência | Benchmark cedo (Phase 2c); envelope é non-negotiable, otimizar encoding se necessário |
| Orderbook state explosion | Média | Alto — OOM | Depth management do MM é referência; adicionar limites configuráveis |
| Multi-exchange complexity | Alta | Médio | 1 exchange por iteração; não paralelizar exchanges |
| Config via configctl overhead | Baixa | Baixo | ConfigSet é cache-friendly; actors carregam config no boot |
| NATS stream consolidation issues | Baixa | Médio | Testar throughput com subjects consolidados vs separados em Phase 2b |

---

## 6. Métricas de Sucesso

A absorção do MarketMonkey será considerada bem-sucedida quando:

1. **Pipeline mínimo funcional**: `exchange → ingest → observation stream → derive → evidence stream` operacional para pelo menos 1 exchange e 1 symbol
2. **Zero código copiado**: Nenhum arquivo, função ou struct do MM existe verbatim no MF
3. **Zero dependências importadas do MM**: Nenhum `import` referencia packages do MM
4. **Todos os quality gates passam**: `make check-deep` green em todas as phases
5. **Subject naming canônico**: Todos os subjects seguem `{domain}.{plane}.{aggregate}.{verb}.{dimensions}`
6. **Envelope universal**: Todo evento nos streams NATS é `Envelope[T]` com metadata completa
7. **Actors sem infra direta**: Nenhum actor importa `pkg/nats` ou qualquer adapter diretamente
8. **Configuração via configctl**: Exchanges, symbols, intervals vêm de ConfigSet, não de arquivo estático
9. **Documentação atualizada**: `stream-taxonomy.md`, `actor-ownership.md`, `runtime-target.md` refletem a nova topologia
