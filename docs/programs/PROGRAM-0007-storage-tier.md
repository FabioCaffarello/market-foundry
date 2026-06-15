# PROGRAM-0007 — Fase Storage Tier

**Status:** Deferred (2026-06-14 — Stage 1 entregue em H-9; Stage 2 / H-10
deferido pending triggers T1/T2/T3, ADR-0023; resumível quando disparar)
**Date:** 2026-06-14
**Owner:** Repository maintainer (Fabio Caffarello)
**Relates to:**
[`../decisions/0023-storage-tier-roadmap.md`](../decisions/0023-storage-tier-roadmap.md),
[`../decisions/0003-clickhouse-analytical.md`](../decisions/0003-clickhouse-analytical.md),
[`../decisions/0008-single-writer-invariant.md`](../decisions/0008-single-writer-invariant.md),
[`../decisions/0024-metrics-policy.md`](../decisions/0024-metrics-policy.md),
[`../operations/slo.md`](../operations/slo.md),
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest",
[`../RESUMPTION.md`](../RESUMPTION.md)

---

## Objetivo

Conduzir a **arquitetura de storage em estágios** do ADR-0023: confirmar
e formalizar o **Stage 1** (ClickHouse cold + NATS KV hot — a topologia
atual) e **instrumentar os gatilhos empíricos** (T1/T2/T3) que decidem
se/quando o **Stage 2** (TimescaleDB, Onda H-10) deve abrir. A Fase
respeita a regra central do ADR-0023: **gatilho primeiro, onda depois** —
TimescaleDB **não** é preparado preemptivamente.

## Contexto de sequenciamento

H-9/H-10 foram **reservadas** ao storage tier desde a numeração da Fase
Insights (que usou H-8.a/b/c). A Fase abre **após** a Fase Delivery
(PROGRAM-0006, H-11.a–d) fechar — ordem não-numérica por priorização do
owner (delivery primeiro). O Stage 1 já está **empiricamente validado**:
insights (H-8.a/b/c) e cross-venue persistem em ClickHouse + KV; nenhum
trabalho de storage *novo* resta em Stage 1 além de **confirmá-lo** e
**tornar os gatilhos do Stage 2 mensuráveis**.

## Escopo (Ondas / sub-ondas)

| Sub-onda | Escopo | Entregas principais |
|----------|--------|---------------------|
| **H-9** | Fechar Stage 1 + instrumentar gatilhos | Promoção parcial do ADR-0023 (**Stage 1 → Accepted**; Stage 2 → `Proposed` pending triggers — autorizado pelo ADR no fechamento de H-9). Instrumentação que torna os gatilhos **observáveis** (o ADR nota: "triggers cannot fire if instrumentation is absent"): **T1** — recording rule do p99 de query operacional do gateway (`marketfoundry_http_request_duration_seconds`) + alerta no limiar > 50 ms; **T2** — métrica de RSS do binário `store` (proxy das projeções KV) + recording rule + alerta no limiar > 4 GB. Atualização do `docs/operations/slo.md` com os SLIs dos gatilhos. **Sem TimescaleDB.** |
| **H-10** | Stage 2 — adoção de TimescaleDB | **TRIGGER-GATED — NÃO ABERTA.** Abre **somente** quando T1, T2 ou T3 disparar (e for confirmado não-transiente pelo maintainer, registrado em RESUMPTION com evidência). Entrega: TimescaleDB no stack de deploy, `internal/adapters/storage/timescale/`, migração do(s) padrão(ões) de query que dispararam, atualização de RUNTIME/ARCHITECTURE, **promoção total do ADR-0023 → Accepted**. Pode permanecer **indefinidamente não-aberta** se nenhum gatilho disparar — steady state legítimo (ADR-0023). |

Capacidades **fora** desta Fase (registradas, ADR-0023 Non-goals):
design de schema TimescaleDB (é decisão de H-10 no momento da adoção);
migração de dados históricos do ClickHouse (forward-only por padrão);
object storage para replay; Cassandra/Scylla/Dynamo; remoção do NATS KV
como hot tier; semântica dual-write.

## Decisão de design da Fase (agente + owner, 2026-06-14)

Pré-flight confirmou: o gatilho **T1 já é mensurável** — o webserver
embrulha **toda** rota com `metrics.InstrumentHTTPHandler`
(`internal/shared/webserver/server.go`), emitindo
`marketfoundry_http_request_duration_seconds{method,path,status}`; o
binário `store` expõe `/metrics` via `healthz.NewHealthServer`
(`cmd/store/run.go`). H-9 adiciona as **regras** (recording + alert) que
transformam essas métricas nos SLIs de gatilho, sem novo tier.

**Conflito resolvido (P6, 2026-06-14):** o owner escolheu "desenvolver o
storage tier"; o ADR-0023 proíbe construir o Stage 2 antes de um gatilho
disparar (e nenhum disparou — Odin/T3 nem existe). Pause-and-report ao
owner → escolha **"fechar Stage 1 + instrumentar gatilhos"**, a leitura
ADR-compliant. Nenhum gatilho é declarado disparado nesta Fase; a
instrumentação apenas os torna observáveis.

## Princípios aplicáveis (P1–P9)

Ver [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest".
Particularmente **P3** (este PRD + promoção do ADR-0023 primeiro); **P7**
(sem claim aspiracional — TimescaleDB não é declarado; Stage 1 é o que o
código entrega); **P9** (loop autônomo com self-merge re-confirmado para
a Fase Storage — ADR-0026 errata 2026-06-14). O ADR-0023 segue
[I3 forward-only](../operations/runtime-invariants.md) e
[I9 no aspirational claims](../operations/runtime-invariants.md).

## Critérios de aceite da Fase

**Stage 1 (fechado em H-9):**

- [x] ADR-0023 promovido a **Stage 1: Accepted** (Stage 2: Proposed
  pending triggers).
- [x] Gatilhos T1 e T2 **observáveis**: recording rules + alertas em
  `deploy/observability/prometheus/` (grupo `storage-triggers`,
  validado por `promtool`); SLIs documentados em `slo.md` +
  `runbooks/storage-triggers.md`.
- [x] `make verify` GREEN; RESUMPTION atualizado no commit de fechamento.

**Stage 2 (H-10 — trigger-gated, fora do controle da Fase abrir):**

- [ ] (futuro) Um gatilho registrado como disparado com evidência em
  RESUMPTION → H-10 abre → TimescaleDB + adapter + 1 query migrada →
  ADR-0023 totalmente `Accepted`.

## ADRs governantes

| ADR | Escopo | Status | Promovido por |
|-----|--------|--------|----------------|
| 0023 | Storage tier roadmap (staged + triggers) | `Proposed` → **Stage 1 Accepted** (H-9); Stage 2 pending triggers | H-9 (parcial) / H-10 (total) |

## Riscos

| Risco | Severidade | Mitigação |
|-------|-----------|-----------|
| Pressão para construir TimescaleDB preemptivamente | Alto | ADR-0023 "gatilho primeiro"; H-10 trigger-gated; este PRD não abre H-10. |
| Gatilho dispara mas instrumentação ausente (não observável) | Médio | H-9 entrega exatamente a instrumentação T1/T2 (fecha o gap que o ADR nota). |
| T2 (RSS por projeções KV) difícil de atribuir exatamente | Médio | Usar RSS total do processo `store` como proxy honesto (projeções KV são o consumidor dominante de memória do store); documentado como proxy. |
| Expectation drift (contribuidores desenham como se TimescaleDB existisse) | Médio | Stage 1/Stage 2 rotulados no ADR + RUNTIME; P7 (sem claim aspiracional). |

## Changelog

- **2026-06-14 (H-9 entregue — Stage 1 fechado; Fase Deferred)** — Stage 1
  do ADR-0023 confirmado e promovido (Stage 1 Accepted); gatilhos T1/T2
  instrumentados (recording + alert rules `storage-triggers`, runbook,
  SLIs no `slo.md`) — observáveis sem TimescaleDB. **Stage 2 (H-10)
  deferido pending triggers**: nenhum disparou; a Fase fica `Deferred`,
  resumível quando T1/T2/T3 disparar e for confirmado pelo maintainer.
- **2026-06-14 (abertura)** — Fase Storage Tier aberta após PROGRAM-0006
  (Delivery) fechar. Owner escolheu o storage tier + re-confirmou o loop
  autônomo. Pause-and-report (P6) resolveu o conflito com o trigger-gate
  do ADR-0023 → escopo ADR-compliant: **fechar Stage 1 + instrumentar
  gatilhos**, sem TimescaleDB. H-9 âncora; H-10 (Stage 2) trigger-gated,
  não aberta.
