# ADR 0016: Harvest from market-raccoon — selective wave protocol

## Status

Accepted.

## Date

2026-05-24.

## Context

A sibling repository, `market-raccoon`, exists at
`$RACCOON_REFERENCE_PATH`
(`/Volumes/OWC Express 1M2/Develop/market-raccoon`). It is the product
of an earlier evolution model under the same maintainer and carries
mature capabilities that market-foundry does not yet have:

- Insights domain (heatmap, volume profile, candle/stats aggregation).
- Deterministic replay + golden tests.
- Multi-venue normalization.
- Protobuf contract layer.
- Observability surface (SLOs, metrics policy, runbooks).
- Sequencing and time-normalization model.
- Backpressure / overload policies.

It also carries explicit technical debt: documented drift between
docs and runtime, layer sovereignty only partially enforced, `cmd/`
sprawl across many auxiliary binaries (portfolio, strategist,
signals-separado, executor, validator) that the foundry has
consolidated into 8 owners (see
[`../ARCHITECTURE.md`](../ARCHITECTURE.md) → "Binary boundaries").

market-foundry, by contrast, has stronger architectural discipline:
layer sovereignty enforced statically by raccoon-cli (ADR
[0004](0004-raccoon-cli-static-enforcement.md) /
[0005](0005-layer-sovereignty.md)), single-writer invariant per
stream and KV bucket (ADR
[0008](0008-single-writer-invariant.md)), configctl as the lone
lifecycle authority (ADR
[0006](0006-configctl-lifecycle-authority.md)), but narrower
functional coverage. It is the right base; the raccoon is the right
catalogue of validated capabilities to inform what to build next.

The open question this ADR answers is **how the two relate**.

## Decision

market-foundry adopts a **selective harvest** of market-raccoon
capabilities, executed via a **wave protocol**. The harvest operates
in **Mode (B): foundry is ground truth, raccoon is a read-only
consultative reference.** No file is copied; capabilities are
re-implemented inside the foundry respecting layer sovereignty and
existing invariants.

The protocol is governed by nine principles (P1–P9), summarised
below. The canonical full version lives in
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" so it is loaded
into every Claude session by default; this ADR captures the durable
decision, not the operational expansion.

### Princípios P1–P9 (resumo)

- **P1 — Foundry é ground truth; raccoon é referência consultiva.**
  Nenhum arquivo do raccoon é copiado; capacidades são reescritas
  no foundry respeitando layer sovereignty.
- **P2 — Raccoon path é estritamente read-only.** Cada leitura é
  justificada antes da abertura.
- **P3 — Toda capacidade portada passa por documento primeiro.**
  Sequência: leitura → documento (PRD/ADR) → implementação →
  analyzer raccoon-cli (se aplicável) → gate.
- **P4 — Uma onda por vez.** Ondas não paralelizam; fechamento
  explícito antes da próxima.
- **P5 — Cada onda evolui raccoon-cli quando adiciona invariante.**
  Sem analyzer, invariante vira intenção sem enforcement.
- **P6 — Pause-and-report ativo durante a onda inteira.** Blocker /
  discrepância / ambiguidade → pare, opções A/B/C/D, espere direção.
  Especialização da ADR [0013](0013-pause-and-report-protocol.md).
- **P7 — Sem perda de disciplina documental.** PRDs e ADRs referem-
  se entre si; RESUMPTION é o sentinel; TRUTH-MAP (instalado em
  H-1) cruzará claim × ADR/PRD × code anchor × test anchor.
- **P8 — Cliente Odin está mapeado, não esquecido.** Entra como
  H-12+ dentro de `client/` no próprio foundry. Até lá, nada de
  cliente é antecipado.
- **P9 — Toda alteração ao foundry passa por Pull Request.** Cada
  onda entrega via branch dedicada; merge em `main` é feito pelo
  maintainer humano, não por agentes. Próxima onda só abre após
  merge da anterior (estende P4 com requisito de incorporação real).
  Sustentado por branch protection, CI gates, lefthook hooks, e
  pause-and-report (P6) como primeira camada.

A versão completa (incluindo critérios de promoção de "Draft" →
"Accepted", tratamento explícito de ADRs de fundação H-2, e as
quatro travas operacionais de P9) vive em
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest".

### Programa Foundation (PROGRAM-0001)

The first phase of the harvest is "Foundation", covering ondas
H-0, H-1, and H-2, and is tracked under
[PROGRAM-0001](../programs/PROGRAM-0001-foundation.md):

- **H-0** — Setup do Harvest (this ADR + PROGRAM-0001 + CLAUDE.md
  protocol installation).
- **H-1** — Operational practices (TRUTH-MAP, AUTHORITY-MAP,
  runtime-invariants, SLOs documented in foundry idiom).
- **H-2** — Foundation ADRs (0017–0023, seven decisions consolidating
  structural choices for the Harvest program; no production code).

Subsequent ondas (H-3 …) port specific capabilities (insights,
replay, multi-venue, proto layer, observability, etc.) and are
scoped at the time each one opens. Each onda lands one or more
ADRs and may evolve raccoon-cli with new analyzers per P5.

## Consequences

### Positive

- **Foundry's architectural discipline is preserved.** Layer
  sovereignty, single-writer invariant, configctl authority, and
  raccoon-cli static enforcement remain unaltered. No drift is
  imported.
- **Raccoon's mature catalogue is available as input.** Years of
  validated capabilities, naming, and operational patterns inform
  foundry design without forcing a migration cost.
- **Wave protocol provides natural stop points.** Each onda has an
  explicit gate; the maintainer can pause, reprioritise, or close
  the program after any onda without leaving in-flight debt.
- **Cliente Odin returns to the roadmap.** Mapped as H-12+ inside
  `client/` of this repo (P8), instead of being permanently absent.
- **TimescaleDB is now an explicit future decision.** Likely H-10
  rather than an undocumented gap; the dual-database strategy
  question can be addressed in context.
- **raccoon-cli evolves wave-by-wave.** New invariants land with
  their enforcement (P5) rather than as deferred intent.

### Negative

- **Wave serialisation slows wall-clock delivery.** P4 forbids
  parallel ondas; total program duration is the sum of waves, not
  the maximum. Mitigated by maintaining short ondas and by the
  asymmetric cost of doc/protocol drift if waves overlap.
- **Maintainer attention split between two repositories.** Each
  raccoon read costs context; cumulative cost is non-trivial.
  Mitigated by P2 (justified reads only) and by P6 (pause on
  ambiguous mapping rather than improvising).
- **Some capabilities will be re-implemented from scratch.**
  Re-writing instead of copying forfeits raccoon-implementation
  effort. Accepted: the foundry's idioms (Hollywood actors, NATS
  KV projection model, ClickHouse analytical store) demand
  rewriting anyway.

### Mitigation summary

- P2 justification gate against silent "browsing → copying" drift.
- P3 doc-first sequencing against premature implementation.
- P5 analyzer-per-invariant against unenforced intent.
- P7 cross-reference discipline against doc rot.

## Alternatives considered

- **(A) Migrate everything from raccoon to foundry.** Rejected.
  Would import raccoon's documented drift (mid-migration `cmd/`
  sprawl, partial layer sovereignty) and discard the foundry's
  enforced disciplines — net regression in architectural quality.
- **(C) Ignore raccoon and rebuild from scratch.** Rejected.
  Would waste years of validated capability knowledge (insights,
  replay, multi-venue, proto layer, observability). The raccoon's
  reference value is the catalogued experience, not the source.
- **(D) Replace foundry with raccoon.** Rejected. Same root reason
  as (A): foundry's disciplines (layer sovereignty enforced,
  single-writer invariant, configctl authority, raccoon-cli
  guards) are not transferable by file-copy; abandoning them is a
  net loss even if functional coverage temporarily widens.

## References

- [PROGRAM-0001 — Harvest Foundation](../programs/PROGRAM-0001-foundation.md)
  — phase tracker for ondas H-0, H-1, H-2.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" —
  canonical P1–P9 expansion (loaded into every Claude session).
- [`../RESUMPTION.md`](../RESUMPTION.md) → "Fase Harvest" — state
  sentinel; current onda and unblocked-next pointer.
- [`../programs/README.md`](../programs/README.md) — PRD convention
  introduced alongside this ADR.
- ADR [0004](0004-raccoon-cli-static-enforcement.md) — raccoon-cli
  as the enforcement tool that P5 evolves wave by wave.
- ADR [0005](0005-layer-sovereignty.md) — invariant that no
  harvested capability may violate.
- ADR [0008](0008-single-writer-invariant.md) — invariant that no
  harvested capability may violate.
- ADR [0013](0013-pause-and-report-protocol.md) — generalised
  pause-and-report protocol that P6 specialises for the Harvest
  surface.
- `.claude/settings.json` → `RACCOON_REFERENCE_PATH` (read-only
  reference checkout).
