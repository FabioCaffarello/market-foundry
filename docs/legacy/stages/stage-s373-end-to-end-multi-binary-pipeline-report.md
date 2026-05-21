# Stage S373 — End-to-End Multi-Binary Pipeline Proof

> Wave: S370–S373 Multi-Binary Orchestration
> Status: **Complete**
> Date: 2026-03-22

## 1. Resumo Executivo

S373 fecha o objetivo principal da wave de orquestração multi-binary: provar que o pipeline canônico funciona corretamente quando separado por processos reais comunicando-se exclusivamente via NATS JetStream.

Após S371 documentar as fronteiras e S372 validar a fiação estrutural, S373 demonstra que **dados fluem corretamente ponta a ponta** — do derive produzindo `StrategyResolvedEvent`, atravessando NATS real, sendo consumido pelo execute em processo separado, avaliado, submetido ao venue, persistido e lido via gateway HTTP.

A prova é sustentada por três camadas complementares:
- **Testes Go estruturais** (sem stack) — invariantes do pipeline verificados
- **Testes Go de integração** (NATS real) — pipeline cross-binary com conexões NATS separadas
- **Smoke script compose** (stack completo) — 12 fases validando binários reais em containers Docker

## 2. Pipeline Multi-Binary Validado

```
derive (binary)
  │ MeanReversionEntryResolver → StrategyResolvedEvent
  │ NATS publish: strategy.events.mean_reversion_entry.resolved.binancef.btcusdt.60
  ▼
NATS JetStream — STRATEGY_EVENTS stream (FileStorage, 72h, 256MB)
  │ Durable consumer: execute-strategy-mean-reversion-entry
  ▼
execute (binary)
  │ StrategyConsumerActor → confidence check → staleness guard → direction→side
  │ VenueAdapterActor → control gate (NATS KV) → paper venue → fill
  │ NATS publish: execution.fill.venue_market_order_filled.*
  ▼
NATS JetStream — EXECUTION_FILL_EVENTS stream
  │
  ├──→ store (binary) → NATS KV materialization
  │       └──→ gateway (binary) → HTTP API (request/reply)
  │
  └──→ writer (binary) → ClickHouse persistence
          └──→ gateway (binary) → HTTP analytical endpoints
```

### Invariantes Verificados

| ID | Invariante | Status |
|----|-----------|--------|
| INV-1 | Identidade do strategy type preservada | PASS |
| INV-2 | Mapeamento direction→side determinístico | PASS (3 direções) |
| INV-3 | Cadeia de correlação intacta derive→fill | PASS |
| INV-4 | Risk type/disposition explícitos | PASS |
| INV-5 | Timestamp do strategy usado (não time.Now) | PASS |
| INV-6 | Tipo errado não entregue ao consumer | PASS (filtro NATS subject) |
| INV-7 | Flat → side=none, qty=0 | PASS |
| CTRL-1 | Gate halt bloqueia fills cross-binary | PASS |
| CTRL-2 | Gate resume habilita fills | PASS |
| MB-1 | Sem estado Go compartilhado entre binários | PASS (conexões NATS separadas) |

## 3. Arquivos Alterados

### Novos

| Arquivo | Tipo | Descrição |
|---------|------|-----------|
| `scripts/smoke-e2e-multi-binary.sh` | Script | Smoke E2E multi-binary (12 fases) |
| `internal/actors/scopes/execute/s373_multi_binary_pipeline_test.go` | Teste Go | Testes de integração multi-binary (4 testes, requer NATS) |
| `internal/actors/scopes/execute/s373_structural_test.go` | Teste Go | Testes estruturais (4 testes, sem stack) |
| `docs/architecture/end-to-end-multi-binary-pipeline-proof.md` | Doc | Documentação da prova E2E |
| `docs/architecture/multi-binary-canonical-pipeline-evidence-and-limitations.md` | Doc | Evidências e limitações |
| `docs/stages/stage-s373-end-to-end-multi-binary-pipeline-report.md` | Doc | Este relatório |

### Modificados

| Arquivo | Alteração |
|---------|-----------|
| `Makefile` | Target `smoke-e2e-multi-binary` + `.PHONY` + `smoke-help` |

## 4. Evidências Principais

### 4.1 Testes Estruturais (Sem Stack)

```
$ go test -run "TestS373_MultiBinaryPipeline_Structural" ./internal/actors/scopes/execute/...
ok  internal/actors/scopes/execute  0.287s
```

4 testes passam validando: derive→execute pipeline, todas as direções, safety gate, tracker metrics.

### 4.2 Testes de Integração (NATS Real)

```
$ go test -tags=integration -run "TestS373_MultiBinaryPipeline" ./internal/actors/scopes/execute/...
```

4 testes com NATS real em `localhost:4222`:
- **S373-MB-1:** Pipeline completo derive→fill com correlação preservada
- **S373-MB-2:** Control gate halt→resume bloqueia e habilita fills
- **S373-MB-3:** KV store legível de conexão NATS separada
- **S373-MB-4:** 3 direções (long/short/flat) corretas cross-binary

### 4.3 Smoke Compose (Stack Docker Completo)

```
$ make up && make seed
$ make smoke-e2e-multi-binary
```

12 fases validando 9 serviços em containers isolados:
- Stream deltas provam que derive está ativo
- Tracker counters do execute confirmam consumo cross-binary
- Gateway HTTP retorna dados materializados pelo store
- ClickHouse contém dados persistidos pelo writer
- Control gate acessível e coerente

### 4.4 Isolamento Binário

Cada teste de integração usa conexões NATS separadas para simular processos isolados:
- Publisher "derive" → conexão NATS A
- Supervisor "execute" → conexão NATS B (via `ExecuteSupervisor`)
- Verificação "store" → conexão NATS C (via `ControlKVStore`)

Nenhum estado Go é compartilhado entre eles. A comunicação é exclusivamente via NATS.

## 5. Limites Remanescentes

| ID | Limite | Risco | Mitigação |
|----|--------|-------|-----------|
| L1 | Apenas `mean_reversion_entry` exercitado | Baixo — arquitetura é family-generic | Futuras famílias testadas quando adicionadas ao execute |
| L2 | Apenas `binancef/btcusdt/60` | Baixo — routing por subject paramétrico | Multi-symbol provado em S220 |
| L3 | Apenas paper venue | Baixo — venue é leaf node | Real venue é concern separado |
| L4 | Writer flush timing não garantido | Info — smoke usa warn, não fail | Testes de integração do writer são separados |
| L5 | Composite chains dependem de pipeline ativo | Info — smoke usa warn | Testes de integração são determinísticos |
| L6 | Sem prova de restart/recovery multi-binary | Médio — JetStream durable garante replay | smoke-restart-recovery.sh cobre cenário isolado |
| L7 | Sem teste de carga/back-pressure | Info — concern operacional | Fora do escopo da wave de correção |

Detalhamento completo: `docs/architecture/multi-binary-canonical-pipeline-evidence-and-limitations.md`

## 6. Preparação Recomendada para S374

Com o pipeline multi-binary provado ponta a ponta, as próximas etapas naturais são:

### 6.1 Monitoramento e Observabilidade Operacional
- Dashboard de stream lag (STRATEGY_EVENTS consumer lag)
- Alertas em `skipped_halt` > 0 quando gate deveria estar ativo
- Métricas de latência derive→fill (correlation_id timestamp delta)

### 6.2 Resiliência e Recovery
- Prova de recovery multi-binary: restart derive, verificar execute retoma
- Prova de consumer replay: execute reinicia, reprocessa pending
- Teste de split-brain: gate halt durante restart

### 6.3 Multi-Family Expansion
- Habilitar `trend_following_entry` no execute consumer
- E2E proof para segunda família
- Verificar isolamento entre famílias (sem cross-contamination)

### 6.4 Hardening Operacional
- Smoke de longa duração (endurance)
- Verificação de memory/goroutine leak sob carga sustentada
- Prova de graceful shutdown com events in-flight

### Recomendação

A próxima etapa de maior valor é **monitoramento e observabilidade operacional** (6.1), pois transforma as evidências pontuais de S373 em vigilância contínua. Isso prepara o terreno para operação confiável e permite detectar regressões antes que impactem o pipeline.
