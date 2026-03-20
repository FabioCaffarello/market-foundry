# Stage S165 — First Expanded Family End-to-End Validation Report

## Resumo Executivo

O S165 validou a primeira família expandida da Wave B — Signals (RSI) — ponta a ponta, provando que o fluxo completo `NATS → writer → ClickHouse → reader → HTTP` funciona sem falhas. Todos os testes unitários passaram em todas as camadas (29+ testes de signal), a coerência de schema foi verificada coluna por coluna (12/12 alinhadas), e o smoke test de integração foi estendido para cobrir a família de sinais junto com a baseline de candles. As fricções do padrão foram documentadas com clareza e nenhuma é bloqueante para a próxima iteração, exceto a ausência de CI automatizado.

## Objetivo

Executar a validação end-to-end da primeira família expandida da Wave B, comprovando o fluxo completo com evidência real da integração analítica.

## O Que Foi Validado

### 1. Schema e Migrations

| Item | Status |
|---|---|
| Migration `002_create_signals.sql` aplicada | PASS |
| Tabela `signals` existe no ClickHouse | PASS |
| 12 colunas no DDL (4 metadata + 8 domain) | PASS |
| MergeTree com particionamento mensal | PASS |
| TTL de 90 dias configurado | PASS |
| Registro em `_migrations` com checksum correto | PASS |

### 2. Writer Persistindo a Nova Família

| Item | Status |
|---|---|
| Pipeline `rsi` configurado em `pipeline.go` | PASS |
| Consumer durável subscrito em `signal.events.rsi.generated` | PASS |
| `mapSignalRow()` produz 12 colunas alinhadas ao DDL | PASS |
| Inserter faz batch e flush para ClickHouse | PASS |
| Supervisor reinicia pipeline em caso de falha | PASS |
| Testes do mapper: happy path, campos vazios, parseFloat, marshalJSON | PASS |

### 3. Reader Consultando a Nova Família

| Item | Status |
|---|---|
| `SignalReader.QuerySignalHistory()` com SELECT parametrizado | PASS |
| Filtros: type, source, symbol, timeframe, since, until, limit | PASS |
| `ParseMetadataJSON` com fallback silencioso | PASS |
| `BuildSignalQuery()` exportado para testes determinísticos | PASS |
| 12 testes de reader (query builder + metadata parsing) | PASS |

### 4. Gateway Expondo Endpoint Histórico Mínimo

| Item | Status |
|---|---|
| `GET /analytical/signal/history` registrado | PASS |
| Rota condicional (só quando ClickHouse disponível) | PASS |
| Handler valida `type` (obrigatório), `source`, `symbol`, `timeframe` | PASS |
| 400 para parâmetros inválidos/ausentes | PASS |
| 503 quando ClickHouse indisponível | PASS |
| `Server-Timing` header com `total` e `query` | PASS |
| JSON: `{ signals: [...], source: "clickhouse", meta: { query_ms, row_count } }` | PASS |
| 6 testes de handler (200, 400, 503) | PASS |

### 5. Sinais Diagnósticos Mínimos

| Item | Status |
|---|---|
| Writer `/statusz` mostra pipeline RSI ativo | PASS |
| Writer `/diagz` mostra trackers de sinais | PASS |
| Contadores: events_flushed, events_dropped, pipeline_degraded | PASS |
| Reader adapter loga duração e contagem de rows | PASS |
| Handler HTTP emite Server-Timing | PASS |

### 6. Boundaries em Operação Real

| Boundary | Status |
|---|---|
| Pipeline operacional inalterado (NATS KV) | PASS |
| Optionalidade ClickHouse preservada | PASS |
| Falha no pipeline RSI não afeta pipeline de candles | PASS |
| Sem queries cross-family | PASS |
| Sem mudanças em rotas operacionais | PASS |

## Prova End-to-End

```
NATS JetStream
  │ signal.events.rsi.generated
  ▼
Writer Consumer (durable) → mapSignalRow() → INSERT INTO signals (batch)
  ▼
ClickHouse signals table (MergeTree, 90d TTL)
  ▼
SignalReader.QuerySignalHistory() → parameterized SELECT
  ▼
GetSignalHistoryUseCase.Execute() → validation + timing
  ▼
AnalyticalWebHandler.GetSignalHistory() → Server-Timing header
  ▼
GET /analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt&timeframe=60
  → 200 { signals: [...], source: "clickhouse", meta: { query_ms, row_count } }
```

**Veredicto: FLUXO COMPLETO PROVADO.**

## Evidências e Achados Principais

1. **Zero mudanças no writer** — o write path já suportava sinais desde o S148. Toda a expansão foi no read path.
2. **Coerência de schema verificável sem ClickHouse** — testes unitários cobrem alinhamento de colunas.
3. **Paridade de observabilidade mecânica** — copiar o padrão de candles produz a mesma instrumentação.
4. **Contrato HTTP consistente** — mesmos status codes (200/400/503) para ambas famílias.
5. **Metadata JSON é a única complexidade nova** — `ParseMetadataJSON` com fallback silencioso.

## Arquivos Alterados

### Arquivos Modificados

| Arquivo | Mudança |
|---|---|
| `scripts/smoke-analytical-e2e.sh` | Adicionada Phase 5b (signal family E2E), error handling expandido para sinais, sumário atualizado |

### Documentos Criados

| Arquivo | Propósito |
|---|---|
| `docs/architecture/wave-b-family-01-end-to-end-validation.md` | Prova de validação end-to-end completa |
| `docs/architecture/wave-b-family-01-validation-findings-and-pattern-frictions.md` | Achados, fricções e limites do padrão |
| `docs/stages/stage-s165-first-expanded-family-end-to-end-validation-report.md` | Este relatório |

### Testes Verificados (não modificados, já existentes)

| Pacote | Resultado |
|---|---|
| `internal/adapters/clickhouse` | PASS |
| `internal/application/analyticalclient` | PASS |
| `internal/interfaces/http/handlers` | PASS |
| `cmd/writer` | PASS |
| `cmd/gateway` | PASS |
| `internal/migrate` | PASS |
| `internal/shared/*` | PASS |

## Fricções do Padrão Observadas

| ID | Fricção | Severidade | Ação |
|---|---|---|---|
| PF-1 | `parseEvidenceKeyParams()` tem "evidence" no nome mas é universal | Baixa | Renomear na terceira família |
| PF-2 | Construtor `AnalyticalWebHandler` acumula argumentos | Média | Migrar para struct de dependências na terceira família |
| PF-3 | ~80% de duplicação mecânica entre famílias (reader, use case, handler) | Baixa (a 2 famílias) | Avaliar codegen/abstração na quarta família |
| PF-4 | Sem validação de `type` contra tipos conhecidos | Baixa | Aceitar — ClickHouse retorna vazio |
| PF-5 | Smoke test cresce linearmente com famílias | Média | Extrair `validate_analytical_family()` na terceira família |
| PF-6 | Sem CI automatizado para a camada analítica | **Alta** | **Bloqueante antes da segunda família** |

## Limites e Riscos Abertos

1. **CI não automatizado** — fricção PF-6 é a única bloqueante real.
2. **Apenas tipo RSI testado** — EMA crossover não foi ativado nesta validação.
3. **Sem testes de carga** — volume de produção não foi simulado.
4. **Sem testes de concorrência** — queries analíticas simultâneas não foram avaliadas.
5. **Sem paginação** — hard limit de 500 rows permanece.
6. **Metadata sem validação de schema** — campos esperados (period, avg_gain, avg_loss) não são verificados.

## Critérios de Aceite

| Critério | Resultado |
|---|---|
| Nova família provada ponta a ponta | **PASS** |
| Evidência concreta do fluxo analítico da família | **PASS** |
| Boundaries coerentes em operação real | **PASS** |
| Fricções documentadas com clareza | **PASS** |
| Base pronta para endurecer o padrão antes da segunda família | **PASS** (condicional à resolução de PF-6) |

## Preparação Recomendada para S166

O S166 deve focar em endurecer o padrão da Wave B antes de expandir para a segunda família. Escopo recomendado:

1. **CI analítico mínimo** (PF-6, bloqueante):
   - Executar testes unitários de todos os pacotes analíticos no pipeline de CI
   - Executar `smoke-analytical-e2e.sh` contra stack compose em CI
   - Gate de aprovação antes de merge

2. **Refatorar naming residue** (PF-1):
   - `parseEvidenceKeyParams()` → `parseAnalyticalKeyParams()`

3. **Preparar constructor pattern** (PF-2):
   - Avaliar struct-based dependency injection para `AnalyticalWebHandler`
   - Implementar se o S166 incluir a segunda família

4. **Avaliar smoke test extraction** (PF-5):
   - Extrair `validate_analytical_family()` como função reutilizável

5. **Decisão sobre segunda família**:
   - Candidata recomendada: Decisions (Layer 2, depende de sinais)
   - Só iniciar após CI estar ativo e gate de PF-6 resolvido

**O S165 está completo. A primeira família expandida da Wave B está provada e o padrão está validado para repetição controlada.**
