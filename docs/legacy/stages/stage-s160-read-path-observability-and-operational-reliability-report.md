# Stage S160 — Read Path Observability and Operational Reliability

> Status: complete | Date: 2026-03-19

## 1. Resumo Executivo

O S160 instrumentou o read path analítico com sinais diagnósticos mínimos e úteis. Antes desta etapa, consultas históricas ao ClickHouse eram funcionalmente corretas mas operacionalmente opacas — sem timing, sem contagem de rows, sem visibilidade estruturada de falhas. Agora, cada resposta analítica carrega metadados de performance, cada camada emite logs estruturados com contexto operacional, e o header `Server-Timing` permite análise de latência sem parsing do corpo da resposta.

Nenhum endpoint novo foi adicionado. Nenhuma dependência externa foi introduzida. Todos os sinais usam `log/slog` e headers HTTP padrão.

## 2. Hardening Aplicado ao Read Path

### 2.1 Camada Adapter (ClickHouse CandleReader)

- Logger estruturado injetado no construtor
- Timing wall-clock medido do despacho da query ao fim da iteração de rows
- Logs de erro com contexto completo (source, symbol, timeframe, elapsed_ms) em três pontos de falha:
  - Execução da query
  - Scan de row individual
  - Iteração do result set
- Log DEBUG em query bem-sucedida com row count e elapsed_ms

### 2.2 Camada Use Case (GetCandleHistoryUseCase)

- Logger estruturado injetado no construtor
- Medição da duração da chamada ao reader
- `QueryMeta` populado no reply: `query_ms` + `row_count`
- Log INFO em toda query completada com timing e contagem
- Log WARN em toda falha com timing e erro

### 2.3 Camada HTTP Handler (AnalyticalWebHandler)

- Logger estruturado injetado no construtor
- Medição do tempo total do handler (inclui validação + serialização)
- Header `Server-Timing` em respostas 200: `total;dur=N, query;dur=M`
- Log WARN em falhas com código do problema e timing total
- Resposta enriquecida com campo `meta` contendo `query_ms` e `row_count`

### 2.4 Contrato de Resposta

Antes:
```json
{"candles": [...], "source": "clickhouse"}
```

Depois:
```json
{"candles": [...], "source": "clickhouse", "meta": {"query_ms": 12, "row_count": 50}}
```

### 2.5 Composição do Gateway

- Logger do gateway propagado por toda a cadeia: compose → adapter → use case → handler
- Sem criação de loggers avulsos; todas as camadas usam sub-loggers do mesmo root

## 3. Arquivos Alterados

| Arquivo | Tipo de Mudança |
|---------|----------------|
| `internal/adapters/clickhouse/candle_reader.go` | Timing, logging, logger no construtor |
| `internal/application/analyticalclient/contracts.go` | `QueryMeta` struct adicionada ao reply |
| `internal/application/analyticalclient/get_candle_history.go` | Logger, timing, meta population |
| `internal/interfaces/http/handlers/analytical.go` | Logger, Server-Timing header, meta na resposta |
| `internal/interfaces/http/routes/analytical.go` | Logger passado ao handler |
| `internal/interfaces/http/routes/core.go` | `Logger` adicionado ao `Dependencies` |
| `cmd/gateway/analytical_reader.go` | Logger passado ao adapter |
| `cmd/gateway/compose.go` | Logger propagado para reader e use case |
| `cmd/gateway/run.go` | Logger passado ao buildRouteDependencies |
| `internal/application/analyticalclient/get_candle_history_test.go` | Atualizado para novo construtor |
| `internal/interfaces/http/handlers/analytical_test.go` | Atualizado para novo construtor + assertions de meta/header |

## 4. Sinais e Melhorias Resultantes

### Sinais Agora Disponíveis

| Sinal | Onde | Para Quê |
|-------|------|----------|
| `query_ms` no body | Resposta HTTP | Latência da query ClickHouse visível ao consumidor |
| `row_count` no body | Resposta HTTP | Volume de dados retornado sem contar array |
| `Server-Timing` header | HTTP headers | Análise de timing sem parsing do body |
| Logs estruturados por camada | stdout/stderr | Diagnóstico operacional com contexto |
| Logs de erro com elapsed_ms | stdout/stderr | Correlação de falha com performance |

### Melhorias Operacionais

1. **Diagnóstico sem ferramentas externas**: `curl -si` agora mostra timing no header.
2. **Visibilidade de queries lentas**: `query_ms` alto aparece no log e na resposta.
3. **Detecção de dados ausentes**: `row_count: 0` quando dados são esperados é imediatamente visível.
4. **Rastreabilidade de falhas**: Cada ponto de falha no read path emite log com parâmetros da query.
5. **Propagação consistente de logger**: Um único logger root com sub-components nomeados.

## 5. Limites Remanescentes

| Limite | Razão | Impacto |
|--------|-------|---------|
| Sem métricas Prometheus/OpenTelemetry | Guard rail: sem observabilidade pesada | Requer log aggregation para tendências |
| Sem contadores de request por endpoint | Requer middleware; fora de escopo S160 | Sem visão de throughput |
| Sem connection pool visibility | Driver ClickHouse não expõe métricas simples | Degradação manifesta-se como latência alta |
| Sem trace IDs write↔read | Requer correlação cross-service | Sem rastreamento ponta-a-ponta |
| Sem query plan analysis | Requer EXPLAIN server-side | Performance analysis requer acesso direto ao ClickHouse |
| Sem alerting automatizado | Sem sistema de alertas configurado | Todos os sinais requerem revisão manual |
| Scan errors não identificam coluna | ClickHouse driver limita detalhes do scan error | Debug de schema drift requer inspeção manual |

## 6. Critérios de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Read path ganha instrumentação mínima útil | OK — timing, logging, meta em todas as camadas |
| Timing, contagem de rows e falhas ficam mais visíveis | OK — `query_ms`, `row_count`, logs estruturados |
| Consulta histórica fica mais operável | OK — Server-Timing header, meta no body, runbook |
| Camada analítica mais próxima de confiabilidade de produção controlada | OK — sinais diagnósticos sem dependências pesadas |
| Base pronta para endurecer config/startup da camada analítica | OK — logger propagado, padrão estabelecido |

## 7. Preparação Recomendada para S161

Com o read path instrumentado, os próximos passos naturais são:

1. **Hardening de config e startup da camada analítica** — validação de configuração ClickHouse, diagnóstico de startup failure mais explícito, timeout de conexão configurável.
2. **ClickHouse health check opcional no `/diagz`** — adicionar ping ao ClickHouse como check diagnóstico (não readiness) para visibilidade no endpoint de diagnóstico do gateway.
3. **Request counting middleware** — contador atômico de requests por família de rota para throughput visibility sem dependências pesadas.
4. **Writer observability alignment** — aplicar o mesmo padrão de `QueryMeta` e `Server-Timing` ao write path para consistência operacional.

Recomendação: o S161 deve focar em **hardening de config/startup + health check diagnóstico**, que são as precondições mais imediatas para operabilidade de produção controlada.
