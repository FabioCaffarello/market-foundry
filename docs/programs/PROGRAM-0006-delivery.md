# PROGRAM-0006 — Fase Delivery (WebSocket)

**Status:** Closed (2026-06-15 — H-11.a–c + endurecimento H-11.d
(`check delivery`) + H-11.e (max-sessions cap); Fase Delivery completa)
**Date:** 2026-06-13
**Owner:** Repository maintainer (Fabio Caffarello)
**Relates to:**
[`../decisions/0028-delivery-websocket-protocol.md`](../decisions/0028-delivery-websocket-protocol.md),
[`../decisions/0027-insights-decision-support.md`](../decisions/0027-insights-decision-support.md),
[`../decisions/0008-single-writer-invariant.md`](../decisions/0008-single-writer-invariant.md),
[`PROGRAM-0005-insights.md`](PROGRAM-0005-insights.md),
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest",
[`../RESUMPTION.md`](../RESUMPTION.md)

---

## Objetivo

Entregar o **transporte push** dos eventos do foundry a clientes via
**WebSocket** — começando pelos insights events (VPVR/TPO/cross-venue)
que a PROGRAM-0005 produziu. Governada por **ADR-0028**: delivery é
read-only transport, loopback-only, com backpressure bounded; vive no
binário **gateway** (P8 — sem binário novo). É o **H-11** do roadmap,
desenhado considerando o futuro cliente Odin (H-12+), **sem código de
cliente** nesta Fase.

## Contexto de sequenciamento

Abre após PROGRAM-0005 (Insights) fechar. É o próximo item **ativo** do
roadmap: o storage tier (H-9/H-10, ADR-0023) é trigger-gated
(empírico — pode ficar `Planned`); o backpressure genérico estava
acoplado ao consumidor de delivery (ausente até agora). Delivery
destrava ambos e é o caminho canônico ao Odin (P8).

## Escopo (Ondas / sub-ondas)

| Sub-onda | Escopo | Entregas principais |
|----------|--------|---------------------|
| **H-11.a** | WS server skeleton + delivery de volume profile | Bounded context `internal/domain/delivery/` (Session, Subscription por padrão de subject NATS); consumer durável `deliver-insights` (`internal/adapters/nats/natsdelivery/`) sobre `INSIGHTS_EVENTS`; `RouterActor` (fan-out sessions↔subjects) + `SessionActor` (1 por conexão; backpressure DropNewest bounded) em `internal/actors/scopes/delivery/`; endpoint `GET /ws` (`interfaces/http`, upgrade gorilla); canário integration (connect→subscribe→receber 1 volume profile); drift-detect ciente do durable. **Promove ADR-0028 → Accepted.** |
| **H-11.b** | Modelo de subscription multi-evento + todos os insights | Generaliza subscription a wildcards de subject (`insights.events.>`, por instrument/venue); adiciona TPO + cross-venue; filtragem por subject. |
| **H-11.c** | Políticas de backpressure + métricas de sessão | Backpressure configurável (DropNewest/DropOldest; PriorityDrop deferido — ADR-0027); tamanho de fila outbound por config; métricas (frames delivered/dropped, client count) em Prometheus. |
| **H-11.d** | Endurecimento — analyzer `check delivery` (P5) | Incremento pós-fechamento (reabre/re-fecha a Fase). Analyzer estático `check delivery` da fronteira read-only/reader-only (ADR-0028 I1/I5): `natsdelivery` nunca `Publish`; durable `deliver-insights` lê `INSIGHTS_EVENTS`. Mirror de `check_insights` (policy + 4 sites de registro + gate step). **Fora**: auth (contradiz loopback non-feature), PriorityDrop (deferido), snapshot-then-delta + max-sessions (→ H-11.e). |
| **H-11.e** | Endurecimento — max-sessions cap | Incremento pós-fechamento (reabre/re-fecha a Fase). Bound configurável do total de sessões WS concorrentes (`delivery.Config.MaxSessions`, default 1024, env `MARKETFOUNDRY_DELIVERY_MAX_SESSIONS`, 0=ilimitado); `Hub.Admit` rejeita acima do cap; métrica `…delivery_sessions_rejected_total`. Completa o "bounded" do ADR-0028 I4 no nível do subsistema. **Fora / → H-11.f**: snapshot-then-delta (port `SnapshotProvider` + subject→KV-key); auth e PriorityDrop seguem fora. |

Capacidades fora desta Fase (registradas): delivery de streams
observacionais (observation/evidence) — sub-onda futura; delivery da
cadeia de directives (decision/risk/execution) — **fora** sem decisão
explícita (ADR-0028 I3); snapshot-then-delta + backfill histórico
(H-11.d futura); auth de rede (loopback é o controle hoje).

## Decisões de design da Fase (agente + owner, pré-flight 2026-06-13)

Pré-flight read-only (foundry serving infra + leitura justificada P2 do
delivery no market-raccoon; nada copiado, P1):

- **D1 — owner = gateway.** Sem binário novo (P8); o gateway já é dono da
  superfície de leitura HTTP e faz binding loopback.
- **D2 — `gorilla/websocket`** (já vendado no módulo de exchanges p/ o
  cliente WS de ingest) como lib do servidor.
- **D3 — subscription por padrão de subject NATS** (nativo; sem Subject
  struct separado estilo raccoon).
- **D4 — backpressure DropNewest bounded** como default (H-11.a
  hardcoded; configurável em H-11.c). Sem buffering ilimitado (ADR-0028
  I4).
- **D5 — escopo inicial insights-only** (ADR-0028 I3); ampliar a
  observacionais é sub-onda futura; directives nunca sem decisão
  explícita.
- **D6 — protocolo JSON** (subscribe/unsubscribe inbound; envelope de
  evento outbound).
- **D7 — placement layer-sovereign**: domain/delivery → application →
  adapters/nats/natsdelivery → actors/scopes/delivery → interfaces/http.

## Princípios aplicáveis (P1–P9)

Ver [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest".
Particularmente: **P1/P2** (raccoon `internal/core/delivery/` é
referência consultiva; nada copiado — o foundry usa subjects NATS
nativos); **P3** (este PRD + ADR-0028 primeiro); **P5** (H-11.a entrega
a invariante "delivery é read-only/loopback" e o enforcement estático —
drift-detect do durable `deliver-insights`, e/ou `check delivery`);
**P8** (sem binário novo — gateway; sem código de cliente Odin);
**P9** (loop autônomo com self-merge re-confirmado — ADR-0026 errata
2026-06-13).

## Critérios de aceite da Fase

A Fase Delivery fecha quando **todos** abaixo forem verdadeiros — todos
satisfeitos no fechamento de H-11.c:

- [x] Sub-ondas H-11.a, H-11.b, H-11.c fechadas (cada uma com
  `make verify` GREEN + RESUMPTION atualizado no commit de fechamento).
- [x] `internal/domain/delivery/` modela Session/Subscription;
  servidor WS no gateway faz bridge `INSIGHTS_EVENTS → clients` (read-
  only, loopback — ADR-0028 I1/I2).
- [x] Backpressure bounded por sessão (I4); consumidor lento não bloqueia
  o fan-out; política configurável (H-11.c — DropNewest/DropOldest).
- [x] Enforcement estático da fronteira delivery (drift-detect do durable
  `deliver-insights`) integrado em `make verify`.
- [x] ADR-0028 promovido a `Accepted` (na H-11.a).
- [x] PROGRAM-0006 transita para `Closed` na entrega final de H-11.c;
  entrada Changelog correspondente.

## ADRs governantes

| ADR | Escopo | Status | Promovido por |
|-----|--------|--------|----------------|
| 0028 | Delivery WS protocol (read-only/loopback/bounded) | **Accepted** (2026-06-13, H-11.a) | H-11.a (servidor WS + consumer `deliver-insights` + enforcement) |

## Riscos

| Risco | Severidade | Mitigação |
|-------|-----------|-----------|
| Servidor WS net-new (concorrência: lifecycle de conexão, fan-out, backpressure) | Alto | Modelo de actors (SessionActor por conexão isola estado); split em sub-ondas; canário integration vivo. |
| Backpressure mal-feito derruba o fan-out / vaza memória | Alto | ADR-0028 I4 (bounded + DropNewest); H-11.c endurece com políticas + métricas. |
| Escopo vaza p/ a cadeia de directives (delivery vira canal de comando) | Alto | ADR-0028 I1/I3 + enforcement estático; insights-only no início. |
| Delivery scope creep (todos os streams de uma vez) | Médio | D5: insights-only na H-11; observacionais e além são sub-ondas futuras. |

## Changelog

- **2026-06-15 (H-11.e entregue — Fase RE-FECHADA)** — max-sessions cap
  entregue: bound configurável do total de sessões WS no hub
  (`delivery.Config.MaxSessions`, default 1024, env, 0=ilimitado);
  `Hub.Admit` rejeita acima do cap (contador atômico; conexão fechada
  com `CloseTryAgainLater`); métrica `…delivery_sessions_rejected_total`.
  Completa o "bounded" do ADR-0028 I4 no nível do subsistema. PROGRAM-0006
  re-fechado. **snapshot-then-delta → H-11.f** (não aberta).
- **2026-06-15 (H-11.e aberta — reabertura)** — após o fechamento de
  H-11.d, o owner escolheu mais endurecimento (H-11.e) e re-confirmou o
  loop. A Fase reabre (Active) para o **max-sessions cap** (bound do total
  de sessões WS, completando o "bounded" do ADR-0028 I4 no nível do
  subsistema). **snapshot-then-delta** fica para H-11.f (maior valor
  cliente; precisa de port `SnapshotProvider` + subject→KV-key). Re-fecha
  ao merge.
- **2026-06-13 (H-11.d entregue — Fase RE-FECHADA)** — analyzer
  `check delivery` entregue: enforcement estático da fronteira
  read-only/reader-only (ADR-0028 I1/I5 — `natsdelivery` nunca `Publish`;
  durable `deliver-insights` lê `INSIGHTS_EVENTS`); `policies/delivery.toml`
  + 6 testes Rust + gate Step 12b. Preenche a lacuna P5. PROGRAM-0006
  re-fechado (Closed). H-11.e (snapshot-then-delta + max-sessions) fica
  como candidato futuro, não aberto.
- **2026-06-13 (H-11.d aberta — reabertura)** — após o fechamento de
  H-11.c, o owner escolheu um incremento de **endurecimento** (H-11.d) e
  re-confirmou o loop autônomo. A Fase reabre (Active) para entregar o
  analyzer `check delivery` (P5 — enforcement estático da fronteira
  read-only/reader-only, ADR-0028 I1/I5), preenchendo a lacuna P5 que
  H-11.a–c deixaram (a invariante existia sem analyzer dedicado;
  drift-detect cobria só o durable). Exclusões registradas: auth de rede
  (contradiz o não-feature loopback), PriorityDrop (deferido por design),
  snapshot-then-delta + max-sessions (→ H-11.e). Re-fecha ao merge.
- **2026-06-13 (H-11.c entregue — Fase FECHADA)** — última sub-onda:
  política de backpressure **configurável** (`BackpressurePolicy`
  DropNewest/DropOldest no domínio; SessionActor evicta o mais antigo no
  DropOldest) + `delivery.Config{QueueSize,Policy}` via `ConfigFromEnv`
  no gateway (sem tocar settings schema); métricas Prometheus
  `marketfoundry_delivery_frames_total{outcome}` +
  `marketfoundry_delivery_sessions`. **PriorityDrop deferido** com
  justificativa (insights são decision-support equi-advisory, ADR-0027).
  `check delivery` dedicado permanece opcional (drift-detect cobre o
  durable). **PROGRAM-0006 → `Closed`** — todos os critérios satisfeitos.
- **2026-06-13 (H-11.b entregue)** — delivery generalizada a **todas as
  famílias de insights**: durable `deliver-insights` lê
  `insights.events.>` com decode dispatched por subject (volume_profile /
  tpo / cross-venue → JSON tipado); frame de fio `{subject, event}` para
  o cliente demuxar multi-família; filtragem por subject (o matcher do
  domínio já suporta wildcards). Canários integration TPO + cross-venue +
  multi-família/1-sessão. Sem novo ADR (ADR-0028 I3 já cobria insights;
  nota de ampliação + wire frame registrada no ADR). Backpressure
  configurável + métricas → H-11.c (fecha a Fase).
- **2026-06-13 (H-11.a entregue)** — primeira sub-onda fechada: bounded
  context `internal/domain/delivery/` (Session/Subscription + matcher de
  subject NATS puro); consumer durável `deliver-insights`
  (`internal/adapters/nats/natsdelivery/`); `RouterActor` + `SessionActor`
  (`internal/actors/scopes/delivery/`) com backpressure DropNewest
  bounded; port `internal/application/ports/delivery.go` (mantém
  interfaces/ sem importar actors/ — ADR-0005); endpoint `GET /ws`
  (gorilla) + wiring no gateway; canário integration
  (publish→subscribe→receive 1 volume profile, real NATS); drift-detect
  ciente do durable `deliver-insights` (P5). **ADR-0028 promovido →
  Accepted.** Subscription multi-evento + filtragem → H-11.b; políticas
  de backpressure + métricas → H-11.c.
- **2026-06-13 (abertura)** — Fase Delivery aberta após PROGRAM-0005
  (Insights) fechar. Owner escolheu Delivery como próxima etapa +
  re-confirmou o loop autônomo (self-merge — ADR-0026 errata). Pré-flight
  fundamentou as Decisões D1–D7 (gateway owner, gorilla, subscription por
  subject NATS, backpressure DropNewest bounded, insights-only inicial,
  JSON, placement layer-sovereign). ADR-0028 criado `Proposed`. Sub-onda
  âncora H-11.a (WS skeleton + volume profile) destravada.
