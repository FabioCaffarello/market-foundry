# Stage S280 — Durable Restart and Consumer Recovery Proof

**Date:** 2026-03-21
**Status:** Complete
**Predecessor:** S279 (OS-process compose-level operational smoke)

## 1. Resumo Executivo

S280 prova que o market-foundry mantém continuidade operacional após restart de componentes críticos no fluxo paper. Foram validados 20 cenários de restart/recovery distribuídos em três camadas: adapter-level (10 testes), writer pipeline (4 testes) e compose-level (6 fases no smoke script).

**Conclusão principal:** O sistema se recupera corretamente de restarts graças a consumers duráveis JetStream com ACK explícito, KV file-backed persistente, e binários stateless. O único gap material identificado é a janela de buffer loss do writer (até 1000 rows ou 5 segundos entre o último ACK e o próximo flush).

## 2. Cenários de Restart/Recovery Validados

### Camada 1: Adapter-Level (NATS real, sem compose)

| ID | Cenário | Resultado |
|----|---------|-----------|
| RR-1 | Consumer durável retoma do último ACK após restart | Eventos durante downtime entregues; ACKed não reentregues |
| RR-2 | Control gate KV sobrevive reconexão | Estado halted persiste; write após reconexão funciona |
| RR-3 | KV projection persiste entre restarts | Dados sobrevivem; monotonicity guard continua operando |
| RR-4 | Publisher reconectado entrega a consumer existente | Novo publisher publica no mesmo stream com sucesso |
| RR-5 | Ciclo completo: publish → restart → resume → zero loss | 10 eventos, 2 batches, restart mid-stream, zero perda |
| RR-6 | Execute restart: safety gate relê KV corretamente | Novo SafetyGate lê halt/active correto do KV |
| RR-7 | Dedup boundary: republicação idempotente | Mesmo MsgID na janela → delivery única |
| RR-8 | Multi-binary: derive restart, execute sobrevive | Dois derive instances, execute (nunca reiniciado) recebe de ambos |
| RR-9 | Writer consumer durável retoma posição no stream | ACKed não reentregue; novos eventos entregues no resume |
| RR-10 | Control gate cross-binary restart coerente | Store restart → write active → derive lê → execute lê |

### Camada 2: Writer Pipeline (ConsumerStarter real)

| ID | Cenário | Resultado |
|----|---------|-----------|
| WR-1 | ConsumerStarter stop/restart retoma posição durável | Eventos durante downtime entregues via restart |
| WR-2 | Row mapping consistente entre restarts | mapExecutionRow produz 20 colunas idênticas |
| WR-3 | Buffer loss boundary documentado | ACKed events NÃO reentregues (buffer loss é o gap) |
| WR-4 | Múltiplos ciclos de restart convergem ao total correto | 3 ciclos × batches variados = total exato (9 eventos) |

### Camada 3: Compose-Level (Docker, stack completo)

| ID | Cenário | Resultado |
|----|---------|-----------|
| RC-1 | Writer restart: consumer durável retoma | Contagem analítica não-decrescente |
| RC-2 | Execute restart: safety gate reinicializado | Readyz saudável; control gate inalterado |
| RC-3 | Store restart: KV projections queryable | Endpoints KV respondem; gate write funciona |
| RC-4 | Gateway restart: HTTP endpoints recuperam | Readyz saudável; analytical e gate funcionais |
| RC-5 | Control gate sobrevive store+gateway restart | Status, reason, updated_by persistem |
| RC-6 | Projeção analítica continuidade | Contagens non-decreasing em todas as fases |

## 3. Arquivos Alterados

### Novos testes
- `internal/adapters/nats/natsexecution/restart_recovery_test.go` — 10 cenários adapter-level
- `internal/adapters/clickhouse/writerpipeline/restart_recovery_test.go` — 4 cenários writer pipeline

### Novo script
- `scripts/smoke-restart-recovery.sh` — Smoke compose-level (6 fases, 6 cenários RC-*)

### Nova documentação
- `docs/architecture/durable-restart-and-consumer-recovery-proof.md` — Prova de restart/recovery
- `docs/architecture/restart-recovery-semantics-and-operational-limits.md` — Semântica e limites
- `docs/stages/stage-s280-durable-restart-and-consumer-recovery-proof-report.md` — Este relatório

## 4. Evidências Principais

### Propriedade 1: Durable Consumer Resume
O JetStream durable consumer mantém a posição do último ACK no servidor NATS. Ao reiniciar o consumer com o mesmo `Durable` name, o consumo retoma exatamente do ponto onde parou. Eventos ACKed nunca são reentregues; eventos não-ACKed são reentregues automaticamente.

**Teste:** RR-1, RR-5, RR-9, WR-1, WR-4
**Mecanismo:** `jetstream.ConsumerConfig{Durable: name, AckPolicy: AckExplicitPolicy}`

### Propriedade 2: KV File-Backed Persistence
Ambos os buckets KV (`EXECUTION_CONTROL` e `EXECUTION_PAPER_ORDER_LATEST`) usam `FileStorage`. Os dados persistem em disco (volume Docker `nats_data`) e sobrevivem a qualquer restart de componente, incluindo o próprio NATS server.

**Teste:** RR-2, RR-3, RR-6, RR-10
**Mecanismo:** `jetstream.KeyValueConfig{Storage: jetstream.FileStorage}`

### Propriedade 3: Stateless Binaries
Nenhum binário Go carrega estado que não possa ser reconstruído a partir de NATS/ClickHouse. Restart é equivalente a um fresh start com os mesmos nomes de consumer durável.

**Teste:** RR-8 (derive restart, execute sobrevive), RC-1 a RC-4 (cada serviço reiniciado)
**Mecanismo:** Architectural decision — all state externalized

### Propriedade 4: Dedup Prevents Amplification
JetStream MsgID dedup impede que o mesmo evento seja armazenado duas vezes no stream dentro da janela de ~2 minutos.

**Teste:** RR-7
**Mecanismo:** `jetstream.WithMsgID(dedupKey)` on publish

### Propriedade 5: Control Gate Durability
O control gate (halt/active) sobrevive a restarts de store, gateway, derive e execute. O estado é lido do KV em cada operação, não cacheado em memória.

**Teste:** RR-2, RR-6, RR-10, RC-5
**Mecanismo:** Poll-based KV read + file-backed bucket

## 5. Limites Explícitos

### Limites conhecidos e aceitos

| Limite | Impacto | Bound | Mitigação futura |
|--------|---------|-------|-----------------|
| **Buffer loss window** | Rows ACKed mas não flushadas perdem-se no crash | ≤1000 rows ou ≤5s de dados | WAL ou disk buffer |
| **ClickHouse duplicatas** | INSERT bem-sucedido + ACK falha = rows duplicadas no restart | ≤1 batch (1000 rows) por restart | Dedup em query (`DISTINCT`, `argMax`) |
| **Dedup window** | Janela JetStream ~2min; fora dela, eventos podem duplicar | Relevante apenas em outages longos | Dedup applicativo ou idempotent INSERT |
| **Sem reconnect automático** | NATS connection loss → processo morre | Docker restart 5-15s | Client-side reconnect (complexidade) |
| **Polling latency** | Gate change → 1 cycle de latência de propagação | Depende da taxa de eventos | NATS KV watch (push-based) |
| **Sem WAL no inserter** | Buffer em memória é o único staging | Ver buffer loss | Disk-backed buffer |

### O que S280 NÃO cobre

- Crash do NATS server (infraestrutura)
- Crash do ClickHouse (infraestrutura)
- Network partition (requer framework distribuído)
- Multi-service crash simultâneo (explosão combinatória)
- Exactly-once delivery para ClickHouse (requer WAL transacional)
- Degradação graceful sob carga (performance testing)

## 6. Preparação Recomendada para S281

Com restart/recovery provado, os próximos ganhos de confiança operacional podem seguir estas direções:

### Opção A: Observability e Alerting Operacional
- Expor métricas de restart (pipeline_restarts, pipeline_degraded) via `/statusz`
- Criar alertas para buffer depth próximo de maxPending
- Monitorar redelivery rate como proxy de saúde do consumer
- **Risco:** baixo; ganho de visibilidade sem mudança de código

### Opção B: Graceful Shutdown Hardening
- Validar que SIGTERM drena inserter buffer antes de exit
- Provar que o supervisor cancela todos os children cleanly
- Verificar que o stop_grace_period (15s) é suficiente para flush
- **Risco:** baixo; melhora a qualidade do shutdown sem ampliar escopo

### Opção C: Multi-Symbol Flow Validation
- Provar que restart/recovery funciona igualmente para múltiplos símbolos
- Verificar isolamento de consumer durable por família e símbolo
- **Risco:** médio; amplia superfície mas mantém foco operacional

### Recomendação
**Opção A** (observability) é o próximo passo mais seguro. Aumenta confiança operacional sem introduzir nova complexidade de código. Opção B é candidata imediata se houver evidência de shutdown incompleto em logs de produção.

---

**Guard rails respeitados:**
- Nenhuma garantia assumida que o sistema não oferece
- Gaps de replay, ordering e dedup documentados com honestidade
- Foco mantido em continuidade operacional mínima do fluxo paper
- Nenhum programa amplo de fault tolerance aberto
