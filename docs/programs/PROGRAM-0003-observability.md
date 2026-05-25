# PROGRAM-0003 — Fase Observability

**Status:** Active
**Date:** 2026-05-25
**Owner:** Repository maintainer (Fabio Caffarello)
**Relates to:**
[`../decisions/0024-metrics-policy.md`](../decisions/0024-metrics-policy.md),
[`../decisions/0025-alerting-strategy.md`](../decisions/0025-alerting-strategy.md),
[`../operations/slo.md`](../operations/slo.md),
[`../operations/runtime-invariants.md`](../operations/runtime-invariants.md),
[`PROGRAM-0001-foundation.md`](PROGRAM-0001-foundation.md) (closed),
[`PROGRAM-0002-wire.md`](PROGRAM-0002-wire.md) (closed),
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest",
[`../RESUMPTION.md`](../RESUMPTION.md)

---

## Objetivo

Tornar visível o estado operacional do `market-foundry` em runtime
via **Prometheus + Grafana**, com **alerts mensuráveis contra SLOs**
e dashboards mínimos cobrindo os fluxos críticos do mesh JetStream.

Esta Fase entrega **observabilidade primária** — counters, scrape,
recording rules, alerts, dashboards, SLOs ativos saindo de
template. Não entrega tracing distribuído, log aggregation, nem
incident-response automation (todos non-goals explícitos, ver
abaixo).

PROGRAM-0001 (Foundation, closed) deixou os counters declarados
em `internal/shared/metrics/` e o template de
[`../operations/slo.md`](../operations/slo.md) com 4 SLIs
definidos. PROGRAM-0002 (Wire, closed) adicionou o
`marketfoundry_consumer_seq_gap_total` per ADR-0020.
PROGRAM-0003 instala o stack que **consome** essas instrumentações.

---

## Escopo (Ondas)

| Onda | Escopo resumido | Entregas principais |
|------|------------------|---------------------|
| **H-5** | Stack Prometheus + Grafana + recording/alerts/dashboards + SLOs ativos | `deploy/observability/{prometheus,grafana}/` completo via compose profile; ADR-0024 (metrics-policy); ADR-0025 (alerting-strategy); refactor `consumer_seq_gap_total` para cardinalidade compliant; `make obs-up`/`make obs-down`/`make obs-reload`; raccoon-cli `check metrics` analyzer; SLOs F1–F4 flipam `Proposed` → `Observing`; `docs/operations/observability.md` guia operacional; 5 dashboards (ingest/derive/store/gateway/determinism-health). |

H-5 é a única onda planejada no PRD inicial. Promoção dos 4 SLOs
de `Observing` para `Committed` (ver ADR-0025 SLO status taxonomy)
é critério para uma onda futura — provavelmente alinhada com H-9
quando insights/baseline-data acumular o suficiente.

---

## Auditoria pré-onda: estado vigente de `/metrics`

Pré-flight executado em 2026-05-25 antes de abrir a Fase. Mapeamento
completo dos 8 binários em `cmd/*/main.go`:

| Binário | `/metrics` exposto? | Mecanismo | Categoria |
|---|---|---|---|
| `configctl` | ✅ | `healthz.NewHealthServer` auto-route | long-running |
| `derive` | ✅ | `healthz.NewHealthServer` auto-route | long-running |
| `execute` | ✅ | `healthz.NewHealthServer` auto-route | long-running |
| `gateway` | ✅ | `routes/core.go:364` (`metrics.HandlerFunc()`) | long-running |
| `ingest` | ✅ | `healthz.NewHealthServer` auto-route | long-running |
| `store` | ✅ | `healthz.NewHealthServer` auto-route | long-running |
| `writer` | ✅ | `healthz.NewHealthServer` auto-route | long-running |
| `migrate` | n/a | sem HTTP (CLI one-shot) | one-shot |

**Mecânica universal:** `internal/shared/healthz/healthz.go:222`
faz `mux.Handle("GET /metrics", metrics.Handler())` no
`HealthServer`. Todo binário long-running que usa
`healthz.NewHealthServer` ganha `/metrics` automaticamente; o
gateway, que tem seu próprio HTTP server, registra `/metrics` via
route table.

**Resultado: 7/7 binários long-running compliant. 0 gaps de
gap-fill.** A entrega 4 originalmente planejada (auditoria +
gap-fill) virou esta seção — registro do estado vigente, não
código.

A invariante "todo binário long-running expõe `/metrics`" é
codificada estaticamente em H-5 commit 9 via raccoon-cli
`check metrics` analyzer com allowlist declarativa
(`tools/raccoon-cli/policies/binaries.toml`) listando o único
one-shot (`migrate`); qualquer novo `cmd/*/main.go` que não
declare `one_shot` no policies file e não registre `/metrics` falha
em `make verify`.

---

## Não-Escopo

- **Distributed tracing (OpenTelemetry, Jaeger, Zipkin).** Future
  phase. Spans cross-binary exigem propagação via NATS headers +
  storage backend; trade-off não justificado para a fase
  single-operator atual onde o mesh é pequeno e logs estruturados
  já correlacionam por `correlation_id`.
- **Log aggregation (Loki, Elasticsearch, OpenSearch).** Future
  phase. Operacionalmente compensado em H-5 pelo *log compensation
  pattern* documentado em ADR-0024: counters com cardinalidade
  reduzida + log estruturado co-emitido carregando dimensões
  finas. Operator usa `docker logs <binary> | grep ...` até
  agregação chegar.
- **Incident-response automation (PagerDuty, Opsgenie, on-call
  rotation).** Future phase. H-5 entrega alerts com labels
  `severity: page|ticket` mas não wired contra paging service.
  Single-operator phase suporta alerts via stdout/log inspection;
  paging só faz sentido com mais de um operator.
- **Per-strategy effectiveness SLOs.** Effectiveness é read-side
  classifier (ADR-0011); não há aggregator por estratégia.
  Per-strategy SLO virá com PnL aggregation, fase futura.
- **Multi-venue parity SLOs.** Single venue family (Binance Spot +
  Futures). Cross-venue SLOs entram quando ADR-0021/ADR-0022 (canonical
  instrument + venue normalization) promoverem em PROGRAM-0004+.
- **Promoção `Observing` → `Committed` em H-5.** Os 4 SLOs F1–F4
  flipam `Proposed` → `Observing` (stack medindo, baseline em
  coleta). `Committed` requer baseline de 7-14 dias mínimo + revisão
  contra dados observados, decisão de uma onda futura.

---

## Princípios governantes

A Fase Observability opera sob o **protocolo P1–P9** documentado em
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest".
Particularmente relevantes para PROGRAM-0003:

- **P3** — Documento primeiro. ADR-0024 (metrics-policy) e
  ADR-0025 (alerting-strategy) precedem qualquer arquivo em
  `deploy/observability/`.
- **P5** — Cada onda evolui raccoon-cli quando adiciona invariante
  arquitetural. H-5 adiciona invariante "todo binário long-running
  expõe `/metrics`" via `check metrics` analyzer com allowlist
  declarativa.
- **P7** — Sem perda de disciplina documental. SLOs declarados
  `Observing` em `slo.md` são SLOs **medidos**, não aspiracionais.
  Status taxonomy explícita (`Proposed`/`Observing`/`Committed`) per
  ADR-0025 evita ambiguidade.

---

## Critérios de aceite da Fase

A Fase Observability fecha quando **todos** os critérios abaixo são
verdadeiros simultaneamente (H-5 é a única onda):

- [ ] H-5 fechada. `make verify` GREEN, RESUMPTION atualizado no
  commit de fechamento.
- [ ] ADR-0024 (metrics-policy) promovido a `Accepted`. Inclui
  naming convention, label budget (proibindo `instrument`,
  `symbol`, `request_id`, `subject` como labels), histogram bucket
  guidance, cardinality budget per subsystem, e log compensation
  pattern documentado.
- [ ] ADR-0025 (alerting-strategy) promovido a `Accepted`. Inclui
  SLO status taxonomy (`Proposed`/`Observing`/`Committed`),
  burn-rate windows (multi-window multi-burn-rate per Google SRE),
  severity tiers (Observing → ticket, Committed → page), e
  silence conventions.
- [ ] `marketfoundry_consumer_seq_gap_total` refactored:
  composite `stream_key` label substituído por
  `{venue, event_type}` per ADR-0024. Log compensation pattern
  documentado inline no metric definition.
- [ ] `deploy/observability/prometheus/prometheus.yml` configurado
  com scrape targets para os 7 binários long-running + Prometheus
  self-scrape. Scrape interval declarado.
- [ ] `deploy/observability/prometheus/recording.rules.yml`
  contém recording rules para os 4 SLOs F1–F4 (error_ratio /
  burn_rate por janelas).
- [ ] `deploy/observability/prometheus/alerts.rules.yml` contém
  burn-rate alerts contra os 4 SLOs, mais alerts runtime
  (consumer lag stall, gap rate non-zero sustained). Alerts
  contra SLOs `Observing` carregam `severity: ticket` per ADR-0025.
- [ ] `deploy/observability/grafana/provisioning/` declara
  datasource Prometheus + dashboards provider. 5 dashboards
  provisionados sob `deploy/observability/grafana/dashboards/`:
  **ingest-health**, **derive-health**, **store-health**,
  **gateway-health**, **determinism-health**.
- [ ] Compose profile `observability` adicionado a
  `deploy/compose/docker-compose.yaml`. `make obs-up`,
  `make obs-down`, `make obs-reload` funcionando. Profile
  opt-in via `--profile observability`.
- [ ] `raccoon-cli check metrics` analyzer integrado em
  `make verify` (gate Step 8). Allowlist declarativa em
  `tools/raccoon-cli/policies/binaries.toml` listando one-shots
  exemptos. PASS no estado vigente; bloquearia adicionar binário
  long-running sem `/metrics`.
- [ ] `docs/operations/slo.md` sai de "template — targets not yet
  measured" para `Status: Active — Observing`. Os 4 SLOs (F1
  ingest 99.5%, F2 derive p99<500ms, F3 gateway-read p99<200ms,
  F4 writer 99.9%) declarados como `Observing` com status taxonomy
  explícito. Promoção para `Committed` declarada como onda
  futura.
- [ ] `docs/operations/observability.md` criado cobrindo: como
  subir stack (`make obs-up`), portas (Prometheus :9090, Grafana
  :3000), credenciais default Grafana (admin/admin documentado),
  onde estão dashboards, como adicionar nova métrica, e ponteiro
  para ADR-0024/0025.
- [ ] `docs/TRUTH-MAP.md` linhas novas: ADR-0024, ADR-0025,
  observability-stack (compose profile + recording rules +
  alerts), check-metrics-analyzer. Summary counts atualizados.
- [ ] `docs/GLOSSARY.md` 5 novos termos: SLI, SLO, error budget,
  burn-rate alert, recording rule.

---

## ADRs governantes

| ADR | Escopo | Status no início da Fase | Promovido por |
|-----|--------|--------------------------|----------------|
| 0024 | Metrics policy (naming, labels, cardinality, log compensation) | Proposed (entregue em H-5 commit 2) | H-5 commit 11 |
| 0025 | Alerting strategy (SLO taxonomy, burn-rate windows, severity tiers) | Proposed (entregue em H-5 commit 3) | H-5 commit 11 |

Diferente de PROGRAM-0002 (que herdou ADRs já em main como Proposed
de PROGRAM-0001 H-2), PROGRAM-0003 **declara e promove** seus dois
ADRs governantes na mesma onda (H-5). Ambos saem `Accepted` quando
H-5 ship.

Nenhuma ADR adicional esperada nesta Fase. Se durante H-5 surgir
necessidade arquitetural não coberta, P6 (pause-and-report) e
nova ADR sob `decisions/0026+`.

---

## Riscos

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Cardinality explosion via label drift | Alto — scrape custoso, Prometheus OOM | ADR-0024 declara label budget + proibidos; raccoon-cli analyzer estende em onda futura para validar labels declarados (deferred — H-5 só valida `/metrics` presence) |
| Targets SLO incorretos (muito agressivos OU muito frouxos) sem baseline | Médio — alertas ruidosos OU falsa complacência | Status taxonomy `Observing` em ADR-0025: alerts ativos com severity `ticket` (não `page`) durante coleta de baseline; promoção para `Committed` exige revisão de target contra dados |
| Compose profile não detectado / esquecido | Baixo — dev sobe stack sem obs | `make obs-up` é wrapper documentado em `observability.md`; `docs/DEVELOPMENT.md` referencia no fluxo padrão |
| Counter `consumer_seq_gap_total` refactor quebra caller existente | Baixo — H-4 declarou counter sem caller produção (cf. caller surface report pré-H-5) | Refactor isolado em commit 4 dedicado; 2 arquivos tocados (metric def + test); pause-and-report ativada se cascade exceder |
| Log compensation pattern não absorvido pelos callers futuros | Médio — dimensão fina perdida sem percepção | ADR-0024 documenta padrão explicitamente; comentário inline no metric definition com exemplo de código; reviewer responsabilidade até log aggregation chegar |
| Stack inicial em CI tem custo de tempo | Baixo — CI mais lento se sempre rodar obs | Profile opt-in (não sobe por default em `make up`); CI smoke targets não dependem de observability |
| Dashboards JSON desviam do schema Grafana ao longo de versões | Médio — provisioning falha após upgrade | Dashboards committed como JSON estável; versão Grafana pinada em compose; upgrade dashboards é onda futura dedicada |

---

## Referência ao raccoon (sem cópia)

Capacidades validadas no raccoon que informam (sem migrar para)
PROGRAM-0003. Lidos em pré-flight de H-5 com justificativa
explícita:

- `deploy/observability/prometheus/{prometheus.yml,recording.rules.yml,alerts.rules.yml}` —
  pattern de scrape minimal + grupos de recording rules
  por SLO (rate, error_ratio per window, burn_rate per window)
  + alerts AND de duas janelas com `for:` distintos por severity.
- `deploy/observability/grafana/provisioning/{datasources,dashboards}/*.yml` —
  datasource declarativo + filesystem provider para dashboards.
- `docs/observability/metrics-policy.md` — label budget
  conservador (proíbe `instrument`, `symbol`, `request_id`,
  `subject`); foundry adota e estende com log compensation pattern.
- `docs/observability/slo.md` — 3 SLOs do raccoon (Ingest 99.9%,
  Delivery Latency 99% < 250ms, Data Loss 99.99%) inspiram mas
  não copiam; foundry mantém 4 SLOs F1–F4 já declarados em
  template H-1.

Anti-padrão: copiar JSON de dashboard ou regex de recording rule
sem adaptar para counters/labels do foundry. Cada artefato em
`deploy/observability/` é escrito do zero referenciando counters
do foundry (`marketfoundry_*`).

---

## Evidence

- [`../decisions/0024-metrics-policy.md`](../decisions/0024-metrics-policy.md)
  (H-5 commit 2)
- [`../decisions/0025-alerting-strategy.md`](../decisions/0025-alerting-strategy.md)
  (H-5 commit 3)
- [`PROGRAM-0001-foundation.md`](PROGRAM-0001-foundation.md) — Fase
  anterior; entregou o template de `slo.md` (T1 definitions) e os
  counters em `internal/shared/metrics/`.
- [`PROGRAM-0002-wire.md`](PROGRAM-0002-wire.md) — Fase anterior;
  entregou `marketfoundry_consumer_seq_gap_total` em H-4 commit 5.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — protocolo
  P1–P9 canônico.
- [`../RESUMPTION.md`](../RESUMPTION.md) → "Fase Harvest" — state
  sentinel.

---

## Changelog

- **2026-05-25** — PROGRAM-0003 created. Status `Active`. H-5
  declared as the (only) onda. ADRs 0024 / 0025 são governantes
  (ambos a serem entregues e promovidos em H-5 — pattern diferente
  de PROGRAM-0002, que herdou ADRs já em main). Pré-flight da onda
  mapeou: 7/7 binários long-running já expondo `/metrics` via
  `healthz.NewHealthServer` + gateway route table; 11 counters
  declarados em `internal/shared/metrics/` compliant com naming
  convention proposta; única discrepância identificada é o label
  composto `stream_key` em `marketfoundry_consumer_seq_gap_total`
  (refactor planejado em H-5 commit 4). Lands como entrega de
  H-5 commit 1.
