# 0028 — Delivery WebSocket protocol (read-only, loopback, bounded)

## Status

Accepted (2026-06-13, H-11.a). Promovido de `Proposed` no commit de
fechamento da H-11.a, que entregou os critérios de promoção: servidor WS
no gateway (`GET /ws`), bounded context `internal/domain/delivery/`,
consumer durável `deliver-insights` sobre `INSIGHTS_EVENTS`
(`internal/adapters/nats/natsdelivery/`), fan-out por actors
(`internal/actors/scopes/delivery/`) com backpressure DropNewest bounded
(I4), e `drift-detect` ciente do durable `deliver-insights`
(enforcement estático da invariante, P5). I1–I5 entregues. **H-11.b
ampliou I3 a todas as famílias de insights** (volume_profile + TPO +
cross-venue): o durable `deliver-insights` lê `insights.events.>` e
decodifica por subject; o frame de fio passou a `{subject, event}` para
o cliente demuxar. **H-11.c fechou a Fase**: política de backpressure
configurável (DropNewest default + DropOldest; **PriorityDrop deferido** —
insights são decision-support equi-advisory, sem ordem de prioridade
natural, ADR-0027), tamanho de fila por config (`delivery.Config` via env
no gateway), e métricas Prometheus (`…delivery_frames_total{outcome}` +
`…delivery_sessions`). **H-11.d** entregou o analyzer dedicado
`check delivery` (P5): enforcement estático da fronteira read-only/
reader-only (`natsdelivery` nunca `Publish`; durable `deliver-insights`
lê `INSIGHTS_EVENTS`) — gate Step 12b, `policies/delivery.toml`.

## Context

A Fase Insights (PROGRAM-0005) entregou três capacidades de
decision-support (VPVR, TPO, cross-venue) publicadas em `INSIGHTS_EVENTS`
e materializadas em KV-latest + ClickHouse, com read endpoints REST no
gateway. Falta o **transporte push**: um cliente (o futuro Odin/WASM,
P8) precisa receber esses eventos em tempo real, não só fazer polling
REST.

O roadmap nomeia isto **H-11 (Delivery WS)**. O pré-flight (2026-06-13)
confirmou:

- O gateway já serve HTTP (httprouter + `http.Server`), que suporta o
  upgrade WebSocket; `gorilla/websocket` já está no dep tree (cliente WS
  de ingest→exchange). Servidor WS é net-new.
- O gateway é o **dono** desta capacidade (P8: sem binário novo; o
  gateway é o owner da superfície de leitura).
- `INSIGHTS_EVENTS` é a fonte; insights são decision-support read-only
  (ADR-0027), seguros para stream a clientes.
- "No HTTP authentication — loopback binding is the access control"
  (CLAUDE.md) molda o servidor WS: loopback-only.

Sem um ADR, o protocolo de delivery (escopo de eventos, backpressure,
auth, formato de frame) vira intenção dispersa. Este ADR fixa as
invariantes.

## Decision

Entregar uma camada de **delivery WebSocket no binário gateway**, que
faz bridge `INSIGHTS_EVENTS → WS clients`, governada pelas invariantes:

- **I1 — Read-only transport (sem directives inbound).** O único inbound
  do cliente são frames de controle `subscribe`/`unsubscribe`. O
  servidor NUNCA aceita injeção de evento/ordem/directive vindo do
  cliente. Delivery transporta eventos para fora; não é um canal de
  comando.
- **I2 — Loopback-only.** O servidor WS herda o binding loopback do
  gateway (CLAUDE.md "No HTTP authentication"); isolamento de rede é o
  controle de acesso. Sem token/header auth.
- **I3 — Escopo decision-support (inicial).** A H-11 entrega delivery de
  **insights events** (ADR-0027 — decision-support, sem directives,
  seguros para stream). Ampliar para outros streams read-only/
  observacionais (observation/evidence) é decisão de sub-onda futura. A
  **cadeia de directives** (decision/risk/execution) **não** é entregue
  sem decisão explícita futura — preserva a fronteira do ADR-0027/0011.
- **I4 — Backpressure bounded.** Consumidor lento não bloqueia o fan-out
  nem outros clientes: fila outbound por-sessão **bounded** com política
  de descarte configurável (default **DropNewest**; **DropOldest**
  entregue em H-11.c; PriorityDrop deferido — insights equi-advisory,
  ADR-0027). Backpressure sustentado fecha a conexão. Sem buffering
  ilimitado. **Bound de subsistema (H-11.e):** o hub limita o total de
  sessões concorrentes (`MaxSessions`, configurável) — recusa e fecha
  além do cap, bounding a memória agregada além do per-session.
- **I5 — Single-writer respeitado (ADR-0008).** Delivery é **leitor** de
  `INSIGHTS_EVENTS` (consumer durável `deliver-insights`); não escreve em
  stream nenhum. Sem binário novo (P8) — vive no gateway.

**Protocolo de fio:** frames JSON. Cliente→servidor:
`{"action":"subscribe"|"unsubscribe","subject":"<nats-subject-pattern>"}`
(ex.: `insights.events.volumeprofile.sampled.>`). Servidor→cliente (desde
H-11.b): `{"subject": "<nats-subject>", "event": <payload-json>}` — o
subject permite ao cliente demuxar quando assina mais de uma família; o
payload é o evento de insights em JSON (mesma forma do read endpoint
`/insights` correspondente). Subscription é por **padrão de subject
NATS** (nativo do foundry; sem traduzir para um Subject struct separado).

**Placement (layer sovereignty):** `domain/delivery` (Session,
Subscription — puros) → `application/delivery` (orquestração/portas) →
`adapters/nats/natsdelivery` (consumer de INSIGHTS_EVENTS) → `actors/
scopes/delivery` (RouterActor de fan-out + SessionActor por conexão) →
`interfaces/http` (endpoint de upgrade `/ws`).

## Consequences

**Positive:**
- O push real-time destrava o cliente Odin (H-12+) e fecha o gap entre
  os insights publicados e um consumidor canônico.
- A invariante I4 (backpressure bounded) é onde o backpressure de
  pipeline adiado se materializa — concreto, por-sessão, sem virar
  política genérica prematura.
- Reuso do `gorilla/websocket` já vendado; sem binário novo (P8).

**Negative / accepted costs:**
- Servidor WS é net-new (lifecycle de conexão, fan-out, backpressure) —
  superfície de concorrência nova; mitigada pelo modelo de actors
  (SessionActor por conexão isola estado) e pelo split em sub-ondas.
- Loopback-only limita o consumo a co-localizados (aceito: o Odin roda
  no mesmo host; auth de rede é Fase futura se necessário).

## Alternatives considered

- **SSE (Server-Sent Events) em vez de WS** — mais simples (HTTP
  unidirecional), mas sem canal inbound para subscribe/unsubscribe
  dinâmico; o cliente teria uma conexão por filtro. Rejeitado: o modelo
  de subscription dinâmica é central.
- **Binário `delivery` dedicado** — rejeitado por P8 (sem binários
  raccoon-style; o gateway absorve a superfície de leitura).
- **Subject struct próprio (estilo raccoon `venue/symbol/timeframe`)** —
  rejeitado: o foundry já tem subjects NATS canônicos
  (`insights.events.…`); subscription por padrão de subject nativo evita
  uma tradução redundante.
- **Backpressure por buffering ilimitado** — rejeitado (I4): risco de
  OOM com clientes lentos; bounded + drop é a escolha.

## References

- CLAUDE.md → "Fase Harvest" P8 (Delivery WS é H-11; Odin H-12+; sem
  binário novo) e "No HTTP authentication" (loopback).
- [ADR-0027](0027-insights-decision-support.md) — insights
  decision-support read-only (a fonte que a delivery transporta).
- [ADR-0008](0008-single-writer-invariant.md) — delivery é leitor, não
  writer.
- [ADR-0011](0011-no-oms-expansion-pairing.md) — a fronteira que I3
  preserva (delivery não vaza a cadeia de directives).
- [PROGRAM-0006](../programs/PROGRAM-0006-delivery.md) — a Fase que
  entrega isto.

## Changelog

- 2026-06-13 — Criado `Proposed` (abertura da Fase Delivery /
  PROGRAM-0006). Promoção a `Accepted` quando a H-11.a entregar o
  servidor WS + consumer `deliver-insights` + enforcement estático.
- 2026-06-13 — Promovido a `Accepted` no fechamento da H-11.a (servidor
  WS `GET /ws`, domain/delivery, natsdelivery `deliver-insights`, fan-out
  por actors com DropNewest bounded, drift-detect ciente do durable).
- 2026-06-13 — H-11.b ampliou I3 a todas as famílias de insights (durable
  lê `insights.events.>`, decode dispatched por subject) e introduziu o
  frame de fio `{subject, event}` (cliente demuxa multi-família).
- 2026-06-13 — H-11.c (fecha a Fase): backpressure configurável
  (DropNewest/DropOldest; PriorityDrop deferido com justificativa) +
  fila por config + métricas Prometheus de delivery. PROGRAM-0006 Closed.
- 2026-06-13 — H-11.d (endurecimento, reabre/re-fecha a Fase): analyzer
  `check delivery` (P5) — enforcement estático de I1/I5 (reader-only +
  stream-bound). PROGRAM-0006 re-fechado.
- 2026-06-15 — H-11.e (endurecimento, reabre/re-fecha a Fase): max-sessions
  cap no hub (`delivery.Config.MaxSessions`, env, default 1024) —
  completa o I4 "bounded" no nível do subsistema (além do per-session
  DropNewest/DropOldest). snapshot-then-delta deferido a H-11.f.
