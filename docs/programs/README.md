# Programs (PRDs)

This directory holds **Product Requirement Documents (PRDs)** — one
per program or phase that market-foundry runs.

A PRD frames a phase as a whole: its *objetivo*, *escopo* (broken into
ondas), *não-escopo*, governing principles, measurable acceptance
criteria, expected ADRs, and risks. PRDs complement ADRs:

- **ADRs answer "why" and "how"** — durable structural decisions
  ([`../decisions/`](../decisions/README.md)).
- **PRDs answer "what" and "when"** — scope, ondas, acceptance.

ADRs govern mechanism; PRDs govern scope. When a PRD and an ADR
reference each other, both must stay consistent — fix the divergence
immediately rather than batch later (same rule as
[`../CONTRIBUTING.md`](../CONTRIBUTING.md) → "Errata: correct
immediately").

---

## When to write a PRD vs an ADR

| You have... | → write |
|---|---|
| A phase / program with multiple ondas | PRD |
| A measurable acceptance criterion for a phase | PRD |
| A risk × mitigation table for a multi-onda effort | PRD |
| A durable structural decision (mechanism, invariant) | ADR |
| A guard rail to enforce indefinitely | ADR |
| A choice between alternatives that future readers must understand | ADR |

Trivial work (a single PR, a bug fix, a refactor) needs neither.

A phase may produce many ADRs; the PRD tracks them as expected
artifacts and stays alive until the phase closes.

---

## Status

A PRD has a single mutable `Status` field:

- **Active** — the phase is in flight; ondas are being executed or
  about to open. RESUMPTION's phase table mirrors this state.
- **Closed** — all ondas closed and acceptance criteria met. PRD
  becomes a historical record but stays in this directory.
- **Deferred** — phase paused or postponed. PRD remains for future
  resumption; a Changelog entry records the reason.

Status transitions are recorded in the PRD's Changelog and reflected
in [`../RESUMPTION.md`](../RESUMPTION.md). RESUMPTION never
duplicates PRD content — it points to the PRD.

---

## Format

Each PRD follows this shape. See
[`PROGRAM-0001-foundation.md`](PROGRAM-0001-foundation.md) for the
canonical example.

| Section | Purpose |
|---|---|
| Header (Status, Date, Owner, Relates to) | Current state + cross-refs |
| Objetivo | One paragraph: what this phase is for |
| Escopo (Ondas) | Onda list with one-line scope per onda |
| Não-Escopo | What this phase explicitly does NOT cover |
| Princípios governantes | Reference to permanent protocols (e.g. CLAUDE.md → Fase Harvest) |
| Critérios de aceite da Fase | Measurable closure criteria |
| ADRs esperados | ADRs that land during the phase |
| Riscos | Risk × impact × mitigation table |
| Evidence | Cross-refs to ADRs, CLAUDE.md, RESUMPTION |
| Changelog | Append-only, date-stamped log of PRD revisions |

PRDs are **append-only on content**. The `Status` field and the
Changelog are the mutable surfaces; all other revisions should
record the change in the Changelog rather than rewrite history in
place.

---

## How PRDs reference ADRs and RESUMPTION

- PRD lists **expected ADRs** by number; ADRs reference back to the
  parent PRD as the source of phase context.
- PRD status transitions reflect in RESUMPTION's phase table;
  RESUMPTION never duplicates PRD content — it points to the PRD.
- ADRs created during a phase's ondas link to the parent PRD in their
  References section.
- Capabilities mentioned in a PRD must not declare functionality the
  code has not yet shipped (per CLAUDE.md → Fase Harvest → P7). Use
  "Planned" for forward-looking entries; "Implemented" only when the
  code ships.
- **Wave-row update convention**: a wave's docs-closure commit
  cannot know its own merge SHA — the commit ships inside the PR.
  The RESUMPTION wave-table row for wave N is therefore flipped to
  `Fechada (PR #X mergeada em main em <sha>, <date>)` by wave N+1
  **as its first opening act** (or by the next docs PR touching
  RESUMPTION, whichever lands first). The staleness window between
  the merge of wave N and the next opening is **by design and
  declared here** — not sentinel drift. A post-merge doc-only
  commit per wave was considered and rejected (recurring PR
  overhead against P9). Formalized in FASE 3.2 (2026-06-10,
  audit finding P1-3 / Question 7).

---

## Index

| # | Title | Status | Ondas |
|---|---|---|---|
| [0001](PROGRAM-0001-foundation.md) | Harvest Foundation | Active | H-0, H-1, H-2 |
| [0002](PROGRAM-0002-wire.md) | Fase Wire — proto + replay + sequencer | Closed | H-3.a, H-3.b, H-4 |
| [0003](PROGRAM-0003-observability.md) | Fase Observability | Active | H-5 |
| [0004](PROGRAM-0004-multi-venue.md) | Fase Multi-venue | Active | H-6.a–H-6.f, H-7 |
| [0005](PROGRAM-0005-insights.md) | Fase Insights | Closed | H-8.a–H-8.c |
| [0006](PROGRAM-0006-delivery.md) | Fase Delivery (WebSocket) | Closed | H-11.a–H-11.c |
