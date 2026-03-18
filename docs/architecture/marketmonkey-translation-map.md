# MarketMonkey → Market Foundry: Matriz de Tradução Canônica

> Documento de referência arquitetural. Não é um plano de implementação.
> Cada elemento do MarketMonkey é analisado sob a ótica do Market Foundry:
> o que é, que papel exerce, que problema resolve, que valor entrega,
> qual equivalente canônico deve existir no MF, e qual veredicto de absorção se aplica.

---

## 1. Eventos e Envelopes

### 1.1 Trade (evento contínuo, timeframe=0)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Struct `Trade{ID, Pair, Unix, Price, Qty, IsBuy}` | Evento de domínio `observation` |
| Papel | Unidade atômica de dado de mercado | Mesmo — é o fato observável primordial |
| Problema que resolve | Captura execução individual de exchange | Idem |
| Valor arquitetural | Alta — é a fonte de todos os derivados (candle, volume, stat) | Altíssima — alimenta todo o pipeline |
| Equivalente MF | `ObservationTrade` dentro de `Envelope[T]` no stream `observation.events.market.trade.{exchange}.{symbol}` |
| Veredicto | **ADAPTAR** — a semântica é canônica, mas deve usar `Envelope[T]`, CBOR, subject naming do MF, e viver no domínio `observation` |

### 1.2 BookUpdate (delta/snapshot de orderbook)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Struct `BookUpdate{Unix, Pair, Asks[], Bids[], Snapshot}` | Evento de domínio `observation` |
| Papel | Delta ou snapshot de livro de ofertas | Mesmo |
| Problema que resolve | Manter orderbook local atualizado | Idem |
| Valor arquitetural | Alta — base para heatmap e profundidade | Alta |
| Equivalente MF | `ObservationBookUpdate` em `observation.events.market.book.{exchange}.{symbol}` |
| Veredicto | **ADAPTAR** — semântica preservada, envelope e subject reescritos |

### 1.3 LiquidationUpdate

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Struct `LiquidationUpdate{Pair, Unix, Price, Size, IsBuy}` | Evento de domínio `observation` |
| Papel | Notificação de liquidação forçada | Mesmo |
| Problema que resolve | Rastrear liquidações para análise de risco | Idem |
| Valor arquitetural | Média — relevante apenas para derivados futuros | Média |
| Equivalente MF | `ObservationLiquidation` em `observation.events.market.liquidation.{exchange}.{symbol}` |
| Veredicto | **POSTERGAR** para Phase 3+ — não é necessário para pipeline mínimo |

### 1.4 Stat (pre-aggregated: funding, mark price)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Struct `Stat{Pair, Unix, MarkPrice, Funding, LiqV*, Tbuy, Tsell, Final}` | Evento de domínio `observation` |
| Papel | Dado pré-agregado da exchange (não derivado internamente) | Mesmo |
| Problema que resolve | Capturar métricas que a exchange publica (funding rate, mark price) | Idem |
| Valor arquitetural | Média-alta — necessário para derivados de futuros | Média |
| Equivalente MF | `ObservationExchangeStat` em `observation.events.market.stat.{exchange}.{symbol}` |
| Veredicto | **ADAPTAR** — separar campos de liquidação (que são derivados) dos campos nativos da exchange |

### 1.5 Candle (derivado, com timeframe)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Struct `Candle{Unix, OHLC, Vbuy, Vsell, Tbuy, Tsell, Final}` em wrapper `Candles{Pair, Timeframe, []*Candle}` | Evento de domínio `evidence` |
| Papel | OHLCV amostrado de trades em intervalos configuráveis | Mesmo — é evidência derivada |
| Problema que resolve | Transformar stream contínuo de trades em séries temporais agregadas | Idem |
| Valor arquitetural | Altíssima — artefato primário de consumo | Altíssima |
| Equivalente MF | `EvidenceCandle` em `evidence.events.candle.sampled.{exchange}.{symbol}.{timeframe}` |
| Veredicto | **ADAPTAR** — semântica perfeita, mas pertence ao domínio `evidence`, não `observation`. A distinção `Final=true/false` deve virar dois subjects distintos ou um campo no envelope |

### 1.6 Volume Profile (derivado, com timeframe)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Struct `Volume{Pair, Unix, Timeframe, Prices[], Buys[], Sells[], PriceGroup, Final}` | Evento de domínio `evidence` |
| Papel | Distribuição de volume por nível de preço | Mesmo |
| Problema que resolve | Visualizar concentração de volume (footprint) | Idem |
| Valor arquitetural | Alta — diferencial de produto | Alta |
| Equivalente MF | `EvidenceVolumeProfile` em `evidence.events.volume.profiled.{exchange}.{symbol}.{timeframe}` |
| Veredicto | **ADAPTAR** — domínio `evidence`. O cálculo de `PriceGroup` a partir de `tickSize` é padrão reutilizável |

### 1.7 Orderbook Snapshot

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Struct `Orderbook{Unix, Pair, AskPrices/Sizes[], BidPrices/Sizes[], LastPrice}` — até 2048 níveis | Projeção materializada |
| Papel | Snapshot consolidado do livro | Mesmo |
| Problema que resolve | Fornecer profundidade atual para clientes | Idem |
| Valor arquitetural | Alta — base para visualização de depth | Alta |
| Equivalente MF | `ProjectionOrderbook` — é uma projeção, não um evento de domínio |
| Veredicto | **ADAPTAR** — não é evento canônico, é projeção do domínio `evidence`. Publicar em `evidence.events.projection.orderbook.{exchange}.{symbol}` |

### 1.8 Heatmap (derivado do orderbook)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Struct `Heatmap{Pair, PriceGroup, Min/MaxPrice, Min/MaxSize, Prices[], Sizes[], Unix}` | Projeção materializada |
| Papel | Agregação do orderbook em bins de preço para visualização | Mesmo |
| Problema que resolve | Reduzir 2048 níveis para bins visualizáveis | Idem |
| Valor arquitetural | Média-alta — produto visual | Média-alta |
| Equivalente MF | `ProjectionHeatmap` em `evidence.events.projection.heatmap.{exchange}.{symbol}` |
| Veredicto | **ADAPTAR** — projeção do domínio `evidence` |

### 1.9 Envelope/Encoding

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | CBOR encoding direto em structs, MsgID como string | `Envelope[T]` com ID, Kind, Type, Source, Subject, CorrelationID, CausationID, Headers, Payload, Problem |
| Papel | Serialização e deduplicação | Serialização, rastreabilidade, causalidade |
| Problema que resolve | MM: serialização eficiente + dedup | MF: serialização + observabilidade + correlação |
| Valor arquitetural | MM envelope é primitivo demais para MF | MF envelope é canônico e não-negociável |
| Equivalente MF | `Envelope[T]` existente — já definido e operacional |
| Veredicto | **DESCARTAR** o padrão MM de encoding direto. Todos os eventos MM devem ser encapsulados em `Envelope[T]` do MF. O MsgID do MM pode ser mapeado para `Envelope.ID` |

---

## 2. Fluxos por Exchange/Symbol/Timeframe

### 2.1 Dimensão Exchange

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Exchange como prefixo de processo (1 consumer binary por exchange) | Exchange como dimensão de roteamento no subject |
| Papel | Isolamento de processo por exchange | Isolamento lógico por subject naming |
| Problema que resolve | Falha de uma exchange não afeta outra | Idem, mas via topologia de actors |
| Valor arquitetural | Alta — isolamento é correto | Alta — mas por actors, não por binários |
| Equivalente MF | `{domain}.events.market.{type}.{exchange}.{symbol}` — exchange como segmento de subject |
| Veredicto | **ADAPTAR** — o conceito de isolamento por exchange permanece, mas a granularidade é por actor (ExchangeConnectorActor), não por binário separado. Um único binário `ingest` pode gerenciar múltiplas exchanges via supervisão de actors |

### 2.2 Dimensão Symbol

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Symbol como actor child do Processor (1 SymbolActor por par) | Symbol como dimensão de roteamento |
| Papel | Paralelismo por par de negociação | Idem |
| Problema que resolve | Processamento paralelo, isolamento de estado por símbolo | Idem |
| Valor arquitetural | Alta — padrão correto e escalável | Alta |
| Equivalente MF | Actor por symbol dentro do scope de cada exchange — ownership claro no supervisor |
| Veredicto | **ABSORVER** — o padrão 1-actor-por-symbol é canônico. Traduzir para a topologia MF com supervisão Hollywood |

### 2.3 Dimensão Timeframe

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Sampler interno por interval (1s, 5s, 60s, 5m, 15m, 30m, 1h, 1d) | Timeframe como segmento final do subject |
| Papel | Produzir derivados em múltiplas janelas temporais | Idem |
| Problema que resolve | Múltiplas granularidades para análise | Idem |
| Valor arquitetural | Alta — multi-timeframe é requisito de produto | Alta |
| Equivalente MF | Samplers como actors filhos do SymbolActor, publicando com timeframe no subject |
| Veredicto | **ABSORVER** — o padrão multi-sampler é canônico. Traduzir lista de intervals para configuração do domínio `observation` |

---

## 3. Consumer / Processor / Store / Server

### 3.1 Consumer (Exchange WebSocket → NATS)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Binary `cmd/consumer` — 1 processo por exchange com pool de WebSocket workers | Binary `ingest` (Phase 2) |
| Papel | Conectar-se a exchanges e publicar raw events | Idem — é o "observation capture" |
| Problema que resolve | Ingestão de dados em tempo real | Idem |
| Valor arquitetural | Altíssima — é o entry point de todo dado | Altíssima |
| Equivalente MF | `cmd/ingest` com `IngestSupervisor` → `ExchangeConnectorActor` por exchange → pool de workers |
| Veredicto | **ADAPTAR** — a responsabilidade é a mesma, mas: (1) subject naming MF, (2) envelope MF, (3) configuração via configctl, não config.yml estático, (4) um único binário com múltiplos actors por exchange, não N binários |

### 3.2 Processor (NATS raw → NATS derivado)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Binary `cmd/processor` — consome raw events, roteia para SymbolActors, que produzem candles/volumes/stats/heatmaps | Binary `derive` (Phase 3) |
| Papel | Transformar observações em evidências | Mesmo papel, vocabulário diferente |
| Problema que resolve | Derivar OHLCV, volume profile, stats de trades brutos | Idem |
| Valor arquitetural | Altíssima — é o core analítico | Altíssima |
| Equivalente MF | `cmd/derive` com `DeriveSupervisor` → actors por exchange → actors por symbol → samplers (candle, volume, stat, orderbook) |
| Veredicto | **ADAPTAR** — topologia de actors é referência forte, mas: (1) pertence ao domínio `evidence`, (2) consome de `observation.events.*`, (3) publica em `evidence.events.*`, (4) configuração via configctl |

### 3.3 Store (NATS → Database)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Binary `cmd/store` — consome de `store_*` streams, persiste em ClickHouse/TimescaleDB | Binary `store` (Phase 3) |
| Papel | Persistência de dados finalizados | Mesmo |
| Problema que resolve | Histórico consultável | Idem |
| Valor arquitetural | Alta — necessário para backfill e queries históricas | Alta |
| Equivalente MF | `cmd/store` como consumidor de eventos finalizados dos domínios `evidence` e `observation` |
| Veredicto | **POSTERGAR** — Phase 3. Quando implementado, deve usar ports/adapters do MF, não o client interface direto do MM |

### 3.4 Server (WebSocket API para clientes)

| Dimensão | MarketMonkey | Market Foundry |
|----------|-------------|----------------|
| O que é | Binary `cmd/server` — Echo HTTP + WebSocket, com Router e Session actors | Extensão do `gateway` existente |
| Papel | Distribuir dados realtime para clientes | Mesmo |
| Problema que resolve | API de consumo para frontends | Idem |
| Valor arquitetural | Alta — interface com produto | Alta |
| Equivalente MF | Extensão do `cmd/server` (gateway) com WebSocket handler + subscription management actors |
| Veredicto | **POSTERGAR** — Phase 3+. O gateway já existe; WebSocket é extensão futura. Não criar binário separado |

---

## 4. Publishers e Subscribers

### 4.1 Mapa de Publicação

| Publisher MM | Stream MM | Publisher MF | Subject MF |
|-------------|-----------|-------------|------------|
| Consumer (exchange) | `trades` | IngestExchangeActor | `observation.events.market.trade.{exchange}.{symbol}` |
| Consumer (exchange) | `bookupdates` | IngestExchangeActor | `observation.events.market.book.{exchange}.{symbol}` |
| Consumer (exchange) | `prestats` | IngestExchangeActor | `observation.events.market.stat.{exchange}.{symbol}` |
| Consumer (exchange) | `liquidations` | IngestExchangeActor | `observation.events.market.liquidation.{exchange}.{symbol}` |
| Trade sampler | `rt_candles` | DeriveCandleSampler | `evidence.events.candle.sampled.{exchange}.{symbol}.{tf}` |
| Trade sampler | `store_candles` | DeriveCandleSampler | `evidence.events.candle.finalized.{exchange}.{symbol}.{tf}` |
| Volume sampler | `rt_volumes` | DeriveVolumeProfiler | `evidence.events.volume.profiled.{exchange}.{symbol}.{tf}` |
| Volume sampler | `store_volumes` | DeriveVolumeProfiler | `evidence.events.volume.finalized.{exchange}.{symbol}.{tf}` |
| Stat sampler | `rt_stats` | DeriveStatAggregator | `evidence.events.stat.aggregated.{exchange}.{symbol}.{tf}` |
| Stat sampler | `store_stats` | DeriveStatAggregator | `evidence.events.stat.finalized.{exchange}.{symbol}.{tf}` |
| Orderbook actor | `rt_orderbooks` | DeriveOrderbookProjector | `evidence.events.projection.orderbook.{exchange}.{symbol}` |
| Orderbook actor | `store_heatmaps` | DeriveHeatmapProjector | `evidence.events.projection.heatmap.{exchange}.{symbol}` |

### 4.2 Mapa de Consumo

| Subscriber MM | Consome de | Subscriber MF | Consome de |
|-------------|-----------|-------------|------------|
| Processor | `trades.*.*` | DeriveTradeRouter | `observation.events.market.trade.>` |
| Processor | `bookupdates.*.*` | DeriveBookRouter | `observation.events.market.book.>` |
| Processor | `prestats.*.*` | DeriveStatRouter | `observation.events.market.stat.>` |
| Store | `store_*.*.*.*` | StoreConsumer (Phase 3) | `evidence.events.*.finalized.>` |
| Server Router | `rt_*.*.*.*` | Gateway WS (Phase 3+) | `evidence.events.*.sampled.>` + `evidence.events.projection.>` |

---

## 5. Ownership de Actors

### 5.1 Topologia MM → Topologia MF

```
MarketMonkey                          Market Foundry
──────────────                        ──────────────

Consumer Binary (1 por exchange)      cmd/ingest (único binary)
└── WebSocketManager                  └── IngestSupervisor
    └── WebSocketConsumer[]               ├── ExchangeConnectorActor (binancef)
                                          │   └── WebSocketWorkerPool
                                          ├── ExchangeConnectorActor (bybit)
                                          │   └── WebSocketWorkerPool
                                          └── ... (1 actor por exchange configurada)

Processor Binary (1 por exchange)     cmd/derive (único binary)
└── SymbolActor (1 por symbol)        └── DeriveSupervisor
    ├── TradeActor                        ├── ExchangeScopeActor (binancef)
    │   └── CandleSampler[]               │   └── SymbolActor (btcusdt)
    ├── VolumeActor                       │       ├── CandleSamplerActor
    │   └── VolumeSampler[]               │       ├── VolumeProfilerActor
    ├── StatActor                         │       ├── StatAggregatorActor
    │   └── StatSampler[]                 │       └── OrderbookActor
    └── OrderbookActor                    ├── ExchangeScopeActor (bybit)
                                          │   └── ...
                                          └── ...

Store Binary (singleton)              cmd/store (Phase 3)
└── (flat consumer)                   └── StoreSupervisor
                                          ├── CandleStoreActor
                                          ├── VolumeStoreActor
                                          ├── StatStoreActor
                                          └── HeatmapStoreActor

Server Binary                         cmd/server (gateway, já existe)
├── ServerRouter                      └── ServerActor (já existe)
└── ServerSession[]                       ├── (HTTP routes existentes)
                                          └── (WebSocket extension — Phase 3+)
```

### 5.2 Regras de Ownership Traduzidas

| Regra MM | Tradução MF |
|----------|-------------|
| 1 consumer binary por exchange | 1 ExchangeConnectorActor por exchange dentro de `ingest` |
| 1 processor binary por exchange | 1 ExchangeScopeActor por exchange dentro de `derive` |
| 1 symbol actor por par | 1 SymbolActor por par dentro do ExchangeScope |
| N samplers por tipo × timeframe | N sampler actors como filhos do SymbolActor |
| Store como singleton flat | StoreSupervisor com actors por tipo de dado |
| Server router como singleton | Extensão futura do gateway |

---

## 6. Subject Naming

### 6.1 Comparação de Convenções

| Aspecto | MarketMonkey | Market Foundry |
|---------|-------------|----------------|
| Formato | `{stream}.{EXCHANGE}.{SYMBOL}[.{TIMEFRAME}]` | `{domain}.{plane}.{aggregate}.{verb}.{exchange}.{symbol}[.{timeframe}]` |
| Case | UPPER para exchange/symbol | lowercase para tudo |
| Exemplos | `trades.BINANCEF.BTCUSDT` | `observation.events.market.trade.binancef.btcusdt` |
| | `rt_candles.BINANCEF.BTCUSDT.60` | `evidence.events.candle.sampled.binancef.btcusdt.60` |
| | `store_candles.BINANCEF.BTCUSDT.60` | `evidence.events.candle.finalized.binancef.btcusdt.60` |
| Wildcard | `trades.BINANCEF.>` | `observation.events.market.trade.binancef.>` |
| Dedup ID | `trade:{exchange}:{symbol}:{tradeID}` | Via `Envelope.ID` (UUID) + NATS MsgID |

### 6.2 Veredicto sobre Subject Naming

**DESCARTAR** completamente o padrão MM. O MF já possui taxonomia canônica definida em `stream-taxonomy.md`:
- Domain como primeiro segmento
- Plane (events/control) como segundo
- Aggregate como terceiro
- Verb/noun como quarto
- Dimensões (exchange, symbol, timeframe) como sufixos

O padrão MM é funcional mas não escala para múltiplos domínios. O padrão MF já suporta `observation`, `evidence`, `signal`, `configctl` etc.

---

## 7. Uso de NATS

### 7.1 Streams JetStream

| Stream MM | Propósito | Storage | Retention | Equivalente MF |
|-----------|----------|---------|-----------|----------------|
| `trades` | Raw trades | File, 4GB | 12h | `OBSERVATION_MARKET_EVENTS` (subjects: `observation.events.market.>`) |
| `bookupdates` | Book deltas | File, 4GB | 12h | Mesmo stream acima (multi-subject) |
| `prestats` | Exchange stats | File, 4GB | 12h | Mesmo stream acima |
| `liquidations` | Liquidações | File, 4GB | 12h | Mesmo stream acima |
| `rt_candles` | Candles realtime | Memory, 128MB | 5m | `EVIDENCE_REALTIME_EVENTS` (subjects: `evidence.events.*.sampled.>`) |
| `rt_volumes` | Volumes realtime | Memory, 128MB | 5m | Mesmo stream acima |
| `rt_stats` | Stats realtime | Memory, 128MB | 5m | Mesmo stream acima |
| `rt_orderbooks` | Orderbook realtime | Memory, 128MB | 5m | `EVIDENCE_PROJECTION_EVENTS` (subjects: `evidence.events.projection.>`) |
| `rt_heatmaps` | Heatmap realtime | Memory, 128MB | 5m | Mesmo stream acima |
| `store_candles` | Candles finais | File, 2GB | 12h | `EVIDENCE_FINALIZED_EVENTS` (subjects: `evidence.events.*.finalized.>`) |
| `store_volumes` | Volumes finais | File, 2GB | 12h | Mesmo stream acima |
| `store_stats` | Stats finais | File, 2GB | 12h | Mesmo stream acima |
| `store_heatmaps` | Heatmaps finais | File, 2GB | 12h | Mesmo stream acima |

**Nota:** O MM usa 12+ streams separados. O MF deve consolidar em ~4 streams com subject filtering. Isso é mais idiomático para NATS JetStream e simplifica administração.

### 7.2 Consumer Patterns

| Padrão MM | Equivalente MF |
|----------|----------------|
| Durable consumer nomeado `{stream}:{exchange}` | Durable consumer `{domain}.{binary}.{exchange}` |
| Ephemeral consumer para clientes WS | Ephemeral consumer gerenciado por subscription actor |
| Explicit ACK | Explicit ACK (mantido) |
| DeliverNewPolicy para realtime | DeliverNewPolicy para realtime (mantido) |
| 24h inactive threshold | Configurável via configctl |

### 7.3 Deduplicação

| Padrão MM | Equivalente MF |
|----------|----------------|
| MsgID como string concatenada | `Envelope.ID` (UUID v7 — ordenável por tempo) como NATS MsgID |
| Formato: `{type}:{exchange}:{symbol}:{id}` | Formato: UUID v7 gerado pelo publisher |
| Window de dedup: padrão NATS (2min) | Mesmo — dedup window padrão NATS |

---

## 8. Estruturas Úteis vs Acoplamentos Históricos

### 8.1 Estruturas a PRESERVAR (traduzidas)

| Estrutura MM | Por quê preservar | Como traduzir |
|-------------|-------------------|---------------|
| `Pair{Exchange, Symbol}` | Identificador natural de instrumento | `MarketInstrument{Exchange, Symbol}` no domínio `observation` |
| `StreamData` interface | Polimorfismo por timeframe | Interface similar no domínio `evidence` |
| `CandleSampler` lógica | Algoritmo de amostragem correto e testado | Reimplementar no domínio `evidence`, mesmo algoritmo |
| `VolumeSampler` com PriceGroup | Aggregation por tick correto | Reimplementar no domínio `evidence` |
| B-tree para orderbook | Estrutura eficiente para range queries | Reutilizar conceito (btree/skiplist) |
| `Final` flag em derivados | Distinção realtime vs finalizado é essencial | Mapear para subject (`.sampled.` vs `.finalized.`) |
| Multi-timeframe samplers | Padrão correto de fan-out temporal | Actors por timeframe no MF |
| Depth management (2048 níveis, ±10%) | Heurística prática e validada | Preservar no OrderbookActor |
| Publish intervals (200ms orderbook, 250ms volume) | Throttling necessário para não saturar | Configurável via configctl |

### 8.2 Acoplamentos a DESCARTAR

| Acoplamento MM | Por quê descartar | Risco se absorvido |
|---------------|-------------------|---------------------|
| `config.yml` estático | MF usa configctl como domínio de configuração | Bypass do lifecycle de config — viola principio 5 |
| Consul para service discovery | MF é NATS-native, não precisa de Consul | Dependência externa desnecessária |
| Echo HTTP framework no server | MF gateway já usa seu próprio HTTP stack | Duplicação de infra HTTP |
| Supabase JWT auth hardcoded | Auth é concern transversal, não do domínio de mercado | Acoplamento de auth no lugar errado |
| ClickHouse/TimescaleDB client direto | MF usa ports/adapters — DB é detail de infra | Violação de layer sovereignty |
| 1 binary por exchange | MF usa actors para isolamento, não processos | Proliferação desnecessária de binários |
| `gorilla/websocket` direto no actor | WebSocket é concern de interface, não de actor | Violação de separação de camadas |
| Subject naming UPPER case | MF convenção é lowercase | Inconsistência |
| CBOR direto sem envelope | MF exige `Envelope[T]` para rastreabilidade | Perda de correlação e causalidade |
| `hollywood/actor` como dependência direta em domain code | Domain deve ser puro, actor é infra | Violação de layer sovereignty |
| Métricas Prometheus hardcoded nos actors | Observabilidade deve ser aspect, não inline | Acoplamento de infra no domínio |
| `backfill`/`history`/`sync` commands | Tooling específico do MM, não traduzível | Importar complexidade desnecessária |
| Version server (`marketmonkeyterminal.com`) | Específico do produto MM | Acoplamento a serviço externo irrelevante |

### 8.3 Padrões a REPENSAR

| Padrão MM | Problema | Alternativa MF |
|----------|----------|---------------|
| Consumer → Processor como binários separados | No MM, necessário por escala; no MF, actors resolvem | `ingest` e `derive` são binários separados por responsabilidade de domínio (observation vs evidence), não por escala |
| `store_*` streams com retention 12h | Assumption de que store é rápido | Retention configurável; store deve ter backpressure |
| `rt_*` em memory storage | Perda de dados em restart | Aceitável para realtime, mas documentar trade-off |
| `Snapshot` bool em BookUpdate | Semântica implícita | Usar events distintos: `book.snapshot` vs `book.delta` |
| `Final` como bool em structs | Mistura estado de lifecycle com dado | Usar subjects distintos (`.sampled.` vs `.finalized.`) para separar streams |

---

## 9. Glossário de Tradução

| Termo MarketMonkey | Termo Market Foundry | Domínio MF |
|-------------------|---------------------|------------|
| Trade | ObservationTrade | observation |
| BookUpdate | ObservationBookUpdate / ObservationBookSnapshot | observation |
| LiquidationUpdate | ObservationLiquidation | observation |
| Stat (pre-aggregated) | ObservationExchangeStat | observation |
| Candle | EvidenceCandle | evidence |
| Volume | EvidenceVolumeProfile | evidence |
| Stat (derived) | EvidenceStatAggregate | evidence |
| Orderbook | ProjectionOrderbook | evidence |
| Heatmap | ProjectionHeatmap | evidence |
| Pair | MarketInstrument | observation (shared) |
| Consumer | IngestExchangeConnector | observation |
| Processor | DeriveProcessor | evidence |
| Store | StoreProjector | store |
| Server | Gateway (já existe) | — |
| SymbolActor | SymbolScopeActor | evidence |
| CandleSampler | CandleSamplerActor | evidence |
| VolumeSampler | VolumeProfilerActor | evidence |
| StatSampler | StatAggregatorActor | evidence |
| OrderbookActor | OrderbookProjectorActor | evidence |
| Stream (trades, rt_candles, etc.) | Subject pattern + JetStream stream | — |
| config.yml | ConfigSet via configctl | configctl |
| Final=true | `.finalized.` subject | — |
| Final=false | `.sampled.` subject | — |
