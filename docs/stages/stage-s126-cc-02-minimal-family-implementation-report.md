# Stage S126 — CC-02 Minimal Family Implementation Report

**Status:** Complete
**Date:** 2025-03-19

## 1. Resumo Executivo

O Stage S126 implementou a family `ema_crossover` (EMA Crossover Signal) no escopo mínimo definido pelo S125, validando a extensibilidade do monorepo market-foundry. A implementação seguiu o Playbook 1 exatamente, produziu 3 arquivos novos e modificou 7 existentes, dentro do envelope previsto (≤4 novos, ≤8 modificados). Nenhum componente arquitetural novo foi necessário — todos os actors, publishers, projections, consumers, streams, routes e domain types existentes foram reutilizados.

## 2. Family Implementada

### EMA Crossover (`ema_crossover`)

**Domínio:** Signal
**Dependência upstream:** Evidence/candle
**Computação:** Duas médias móveis exponenciais (EMA-9 fast, EMA-21 slow) com detecção de crossover.
**Output:** `"bullish"` | `"bearish"` | `"neutral"`
**Metadata:** `fast_period`, `slow_period`, `fast_ema`, `slow_ema`, `spread`

### Ciclo completo implementado:
1. **Actor (derive):** `EMACrossoverSignalSamplerActor` recebe `candleFinalizedMessage`, delega para `EMACrossoverSampler`, emite `publishSignalMessage` + `signalGeneratedMessage`.
2. **Publisher:** `SignalPublisherActor` (reusado) roteia para subject `signal.events.ema_crossover.generated.>`.
3. **Projection (store):** `SignalProjectionActor` (reusado) materializa em bucket `SIGNAL_EMA_CROSSOVER_LATEST`.
4. **Route (gateway):** `/signal/ema_crossover/latest` via route existente type-parameterized.
5. **Wiring:** Config-driven via `pipeline.signal_families: ["ema_crossover"]`.

## 3. Arquivos Alterados

### Novos (3)
| Arquivo | Linhas | Propósito |
|---------|--------|-----------|
| `internal/application/signal/ema_crossover_sampler.go` | ~110 | Lógica pura de cálculo EMA |
| `internal/application/signal/ema_crossover_sampler_test.go` | ~130 | 6 testes unitários |
| `internal/actors/scopes/derive/ema_crossover_signal_sampler_actor.go` | ~80 | Wrapper actor |

### Modificados (7)
| Arquivo | Mudança |
|---------|---------|
| `internal/shared/settings/schema.go` | +2 linhas: registro em `knownSignalFamilies` e `signalDependsOnEvidence` |
| `internal/adapters/nats/signal_registry.go` | +30 linhas: `EMACrossoverGenerated`, `EMACrossoverLatest`, `StoreEMACrossoverSignalConsumer()` |
| `internal/adapters/nats/signal_publisher.go` | +3 linhas: `case "ema_crossover"` em `specForType()` |
| `internal/adapters/nats/signal_kv_store.go` | +1 linha: constante `SignalEMACrossoverLatestBucket` |
| `internal/actors/scopes/derive/derive_supervisor.go` | +10 linhas: registro do processor `ema_crossover` |
| `internal/actors/scopes/store/store_supervisor.go` | +15 linhas: registro do pipeline `ema_crossover` |
| `internal/shared/settings/settings_test.go` | +1 linha: atualização da contagem esperada de signal families |

### Documentação (3)
| Arquivo | Propósito |
|---------|-----------|
| `docs/architecture/cc-02-implementation-notes.md` | Decisões de implementação e simplificações |
| `docs/architecture/cc-02-runtime-activation-projection-and-route.md` | Activation, projection, route e NATS topology |
| `docs/stages/stage-s126-cc-02-minimal-family-implementation-report.md` | Este relatório |

## 4. Simplificações Adotadas

| Simplificação | Justificativa |
|---------------|---------------|
| Períodos fixos (9/21) | Mecanismo de parâmetros por-family não existe; suficiente para prova de extensibilidade |
| Sem downstream families | CC-02 valida signal-layer apenas; decision/strategy/risk/execution para ema_crossover fora de escopo |
| Sem multi-timeframe | Cada sampler opera independente por timeframe |
| Tolerância fixa (1e-8) | Previne oscilação de ruído sem configurabilidade adicional |

## 5. Critérios de Extensibilidade — Resultados

### EX — Structural Extensibility
| ID | Critério | Status |
|----|----------|--------|
| EX-01 | Domain model unchanged | PASS |
| EX-02 | No new domain types | PASS |
| EX-03 | Projection actor reused | PASS |
| EX-04 | Consumer actor reused | PASS |
| EX-05 | Publisher actor reused | PASS |
| EX-06 | HTTP route reused | PASS |
| EX-07 | Stream reused | PASS |

### RF — Registration Friction
| ID | Critério | Target | Actual | Status |
|----|----------|--------|--------|--------|
| RF-01 | New files | ≤ 4 | 3 | PASS |
| RF-02 | Modified files | ≤ 8 | 7 | PASS |
| RF-03 | Application logic | ≤ 120 lines | ~110 | PASS |
| RF-04 | Actor code | ≤ 80 lines | ~80 | PASS |
| RF-05 | Boilerplate per site | ≤ 15 lines | ≤ 15 | PASS |

### PL — Playbook Adherence
| ID | Critério | Status |
|----|----------|--------|
| PL-01 | Playbook 1 followed | PASS |
| PL-02 | Naming conventions | PASS |
| PL-03 | Dependency graph validated | PASS |
| PL-04 | Config lifecycle works | PASS |

### PP — Pipeline Proof
| ID | Critério | Status |
|----|----------|--------|
| PP-01 | Signals published to NATS | Ready (requires live validation) |
| PP-02 | Signals projected to KV | Ready (requires live validation) |
| PP-03 | Signals queryable via HTTP | Ready (requires live validation) |
| PP-04 | RSI coexistence unaffected | PASS (tests pass, no RSI code changed) |

### GV — Governance
| ID | Critério | Status |
|----|----------|--------|
| GV-01 | `make test` passes | PASS (all modules compile, tests pass) |
| GV-02 | No existing tests broken | PASS (1 test updated for new family count) |

## 6. Triggers Observados

### CF-08 — Boilerplate em Arquivos Modificados
**Evidência:** O actor `ema_crossover_signal_sampler_actor.go` é ~95% idêntico ao `signal_sampler_actor.go` (RSI). Os registration sites em `derive_supervisor.go`, `store_supervisor.go`, `signal_registry.go`, `signal_publisher.go` e `signal_kv_store.go` seguem padrão copy-paste com variação mínima (nomes e subjects).

**Observação:** Com 2 signal families, o custo é aceitável. Com 3+, uma abstração genérica (signal sampler factory + registry-driven routing) seria justificável.

### CF-03 — Correlation ID Copy-Paste
**Evidência:** O padrão `events.NewMetadata().WithCorrelationID(msg.CorrelationID)` é copiado identicamente em todo novo actor. A propagação é manual e repetitiva.

**Observação:** Um middleware de correlation ID no actor framework eliminaria essa duplicação, mas requer mudança no `actorcommon` package.

### D4 — Composition Root Test Coverage
**Evidência:** Os composition roots (`cmd/derive/run.go`, `cmd/store/run.go`) não têm testes unitários. A inclusão correta do novo processor e pipeline depende apenas de compilação e testes de integração.

## 7. Preparação Recomendada para S127

O S127 deve focar na **validação operacional live** da extensibilidade comprovada pelo S126:

1. **Live pipeline activation**: Ativar `ema_crossover` na config do pipeline live e verificar que signals são produzidos e queryáveis.
2. **Coexistence validation**: Confirmar que RSI continua operando normalmente com `ema_crossover` ativo simultaneamente.
3. **Friction capture**: Documentar qualquer atrito operacional não previsto durante a ativação.
4. **PP-01/PP-02/PP-03 gate**: Os critérios de Pipeline Proof requerem validação live para serem marcados como PASS definitivo.
5. **Healthz/Diagz**: Verificar se os componentes `ema_crossover` aparecem nos endpoints de diagnóstico.

### Decisões pendentes para o S127:
- Avaliar se CF-08 atingiu massa crítica suficiente para justificar refactor (com 2 families, provavelmente não).
- Avaliar se CF-03 merece resolução ou se permanece como dívida aceitável.
- Definir se a CC-02 precisa de downstream families (decision/strategy) para validação completa ou se a prova no signal layer é suficiente.
