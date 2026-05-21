# Stage S374 — Operational Smoke and Failure Isolation Multi-Binary

> Wave: S370–S375 Multi-Binary Orchestration
> Status: **Complete**
> Date: 2026-03-22

## 1. Resumo Executivo

S374 prova que falhas localizadas em binários individuais NÃO contaminam outros binários no pipeline multi-binary. Complementa S373 (prova de fluxo E2E) com prova de resiliência operacional.

O resultado central: **quando um binário falha e reinicia, os outros continuam operando corretamente, e o pipeline retoma automaticamente após a recuperação.**

Esta etapa foi deliberadamente mantida proporcional — restart determinístico por binário, sem chaos engineering ampla, sem benchmark de produção. O objetivo é aumentar confiança operacional sem inflar escopo.

## 2. Smoke e Isolamento Validados

### Cenários de Isolamento de Falhas

| Cenário | Binário Reiniciado | Binários Não Afetados | Resultado |
|---------|-------------------|----------------------|-----------|
| FI-1 | derive | execute, store, gateway | PASS — todos permanecem ready |
| FI-2 | execute | derive, store, gateway | PASS — derive continua publicando |
| FI-3 | store | derive, execute | PASS — gateway degrada e recupera |
| FI-4 | (todos após ciclo) | - | PASS — pipeline flui E2E |
| FI-5 | (integridade streams) | - | PASS — contagem non-decreasing |
| FI-6 | (isolamento trackers) | - | PASS — métricas independentes |

### Invariantes Estruturais (Go Tests)

| Teste | O Que Prova | Status |
|-------|-------------|--------|
| `DurableConsumerSpecStable` | Nome durable determinístico → resume de checkpoint | PASS |
| `IndependentTrackers` | Trackers isolados — sem contaminação cruzada | PASS |
| `ActorHandlesRedelivery` | Redelivery produz output idêntico | PASS |
| `StalenessGuardProtectsAfterRestart` | Eventos stale bloqueados pós-restart | PASS |
| `TrackerSurvivesActorRecreation` | Counters acumulam entre ciclos de vida | PASS |
| `GateSafetyOnRestart` | Safety gate funciona com KV indisponível | PASS |

### Propriedades Provadas

```
derive restart  → execute/store/gateway: NÃO AFETADOS
execute restart → derive/store/gateway:  NÃO AFETADOS
store restart   → derive/execute:        NÃO AFETADOS
                → gateway:               DEGRADA TEMPORARIAMENTE, RECUPERA

control gate    → DURABLE em NATS KV, sobrevive restart de qualquer binário
stream counts   → NON-DECREASING, zero perda com JetStream file storage
durable names   → DETERMINÍSTICOS, permitem resume de checkpoint
```

## 3. Arquivos Alterados

### Novos

| Arquivo | Tipo | Descrição |
|---------|------|-----------|
| `scripts/smoke-failure-isolation-multi-binary.sh` | Script | Smoke de isolamento de falhas (7 fases) |
| `internal/actors/scopes/execute/s374_failure_isolation_test.go` | Teste Go | 6 testes estruturais de isolamento |
| `docs/architecture/operational-smoke-and-failure-isolation-across-binaries.md` | Doc | Metodologia e design do smoke |
| `docs/architecture/multi-binary-smoke-failure-isolation-findings-and-limitations.md` | Doc | Achados e limitações |
| `docs/stages/stage-s374-operational-smoke-and-failure-isolation-report.md` | Doc | Este relatório |

### Modificados

| Arquivo | Alteração |
|---------|-----------|
| `Makefile` | Target `smoke-failure-isolation` + `.PHONY` + `smoke-help` |
| `docs/stages/INDEX.md` | Entrada S374 |

## 4. Evidências Principais

### 4.1 Testes Estruturais (Sem Stack)

```
$ go test -run "TestS374_FailureIsolation" ./internal/actors/scopes/execute/... -v
--- PASS: TestS374_FailureIsolation_DurableConsumerSpecStable (0.00s)
--- PASS: TestS374_FailureIsolation_IndependentTrackers (0.00s)
--- PASS: TestS374_FailureIsolation_ActorHandlesRedelivery (0.04s)
--- PASS: TestS374_FailureIsolation_StalenessGuardProtectsAfterRestart (0.00s)
--- PASS: TestS374_FailureIsolation_TrackerSurvivesActorRecreation (0.09s)
--- PASS: TestS374_FailureIsolation_GateSafetyOnRestart (0.00s)
PASS
ok  internal/actors/scopes/execute  0.303s
```

### 4.2 Smoke Compose (Stack Docker Completo)

```
$ make up && make seed
$ make smoke-failure-isolation
```

7 fases:
1. Pre-flight: 9 serviços healthy, baselines capturados
2. FI-1: Restart derive → execute/store/gateway permanecem ready
3. FI-2: Restart execute → derive continua publicando STRATEGY_EVENTS
4. FI-3: Restart store → derive/execute não afetados, gateway recupera
5. FI-4: Pipeline completo flui após ciclo de restarts
6. FI-5: Stream counts non-decreasing (zero perda)
7. FI-6: Trackers não degradados, testes estruturais passam

### 4.3 Mecanismo de Isolamento

O isolamento funciona porque NATS JetStream é o único ponto de acoplamento:
- Cada binário tem suas próprias conexões NATS
- Durable consumers com nomes estáveis retomam de checkpoint
- NATS KV persiste control gate e projections
- Nenhum binário depende de outro binário em memória

## 5. Limites Remanescentes

| ID | Limite | Risco | Nota |
|----|--------|-------|------|
| L1 | NATS não testado como ponto de falha | Médio | NATS é infra compartilhada — falha afeta todos |
| L2 | Apenas restarts sequenciais | Baixo | Concurrent failures → chaos engineering, fora de escopo |
| L3 | Restart graceful (SIGTERM), sem crash (SIGKILL) | Baixo | Redelivery determinístico provado estruturalmente |
| L4 | Writer buffer loss no crash | Known | Trade-off batch flush; documentado desde S280 |
| L5 | Sem endurance test | Info | Concern operacional separado |
| L6 | Gateway degradação temporária no store restart | Expected | Gateway é proxy stateless, recupera automaticamente |

Detalhamento: `docs/architecture/multi-binary-smoke-failure-isolation-findings-and-limitations.md`

## 6. Preparação Recomendada para S375 (Gate Final da Wave)

S375 deve fechar a wave S370–S375 como gate final. Recomendações:

### 6.1 Gate Checklist
- [ ] S371: Fronteiras e contratos documentados → DONE
- [ ] S372: Fiação compose validada → DONE
- [ ] S373: Pipeline E2E provado → DONE
- [ ] S374: Isolamento de falhas provado → DONE
- [ ] S375: Gate review, consolidação, wave closure

### 6.2 Consolidação de Evidências
- Agregar todos os smoke targets da wave num resumo executivo
- Confirmar que `make smoke-compose-wiring && make smoke-e2e-multi-binary && make smoke-failure-isolation` passam em sequência
- Documentar a surface de testes da wave (Go tests + smoke scripts)

### 6.3 Gaps para Próxima Wave
- Monitoramento contínuo (stream lag, consumer health dashboards)
- Multi-family expansion no execute (trend_following_entry, squeeze_breakout_entry)
- NATS cluster resilience (produção, não local)
- Endurance testing sob carga sustentada
- Real venue adapter integration

### Recomendação

S375 deve ser um **gate review curto** que:
1. Executa os 3 smoke targets da wave em sequência
2. Compila as evidências em um resumo executivo da wave
3. Declara a wave como fechada ou documenta ações pendentes
4. Não adiciona funcionalidade nova — apenas avalia e fecha
