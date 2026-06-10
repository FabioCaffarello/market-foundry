# AUTHORITY — Document hierarchy and precedence

**Status:** Active
**Date:** 2026-05-24
**Owner:** Repository maintainer
**Authority tier:** T1 — Canonical (this file defines the tiering;
it is itself T1)
**Relates to:** [`TRUTH-MAP.md`](TRUTH-MAP.md),
[`decisions/`](decisions/README.md),
[`programs/`](programs/README.md),
[`RESUMPTION.md`](RESUMPTION.md)

---

## Purpose

Define **which document wins** when two parts of the foundry's
documentation disagree, and **how documents move between tiers**
over their lifecycle.

Without an explicit hierarchy, every documentation conflict becomes
an ad-hoc negotiation: "this old stage report says X, but the new
ADR says Y, which one is authoritative?" This file answers that
question categorically and saves the negotiation.

---

## Hierarchy

```
┌──────────────────────────────────────────────────────────┐
│  T1 — Canonical      governs code, architecture          │
│  ──────────────                                          │
│  ADRs (Accepted), PRDs (Active), CLAUDE.md, TRUTH-MAP,   │
│  AUTHORITY, runtime-invariants, ARCHITECTURE             │
└──────────────────────────────────────────────────────────┘
                       ▲ promotes when accepted
┌──────────────────────────────────────────────────────────┐
│  T2 — Operational    how to run / develop / test         │
│  ────────────────                                        │
│  RESUMPTION, DEVELOPMENT, RUNTIME, HTTP-API,             │
│  CONTRIBUTING, GLOSSARY, operations/*.md,                │
│  domain/*.md, programs/README                            │
└──────────────────────────────────────────────────────────┘
                       ▲ promotes when accepted
┌──────────────────────────────────────────────────────────┐
│  T3 — Evolutionary   proposals, drafts (not binding)     │
│  ────────────────                                        │
│  ADRs (Proposed), PRDs (in-flight not closed),           │
│  in-flight prompt scaffolds, design-meta candidates      │
└──────────────────────────────────────────────────────────┘
                       ▲ demotes after closure
┌──────────────────────────────────────────────────────────┐
│  T4 — Historical     past records (never govern)         │
│  ────────────────                                        │
│  PRDs (Closed), ADRs (Superseded), RESUMPTION             │
│  sections marked as past phases, prior                    │
│  RESUMPTION snapshots in git                              │
└──────────────────────────────────────────────────────────┘
```

**Precedence**, top to bottom: **T1 > T2 > T3 > T4.** Within a tier,
the **more specific document wins** (ADR-0008 on single-writer beats
a generic ARCHITECTURE.md paragraph).

---

## T1 — Canonical

T1 documents **govern code**. Code must conform to them; if code
and T1 diverge, the code is the bug (or the T1 doc needs an explicit
revision through the proper process — a new ADR superseding the old
one, or a PRD update).

| Family | Path | Update process |
|---|---|---|
| ADRs (Accepted) | [`decisions/0001…0016*.md`](decisions/) | Append-only. To change, write a new ADR superseding the old; do not edit history (except typos / broken links). |
| PRDs (Active) | [`programs/PROGRAM-*.md`](programs/) (status `Active`) | Status field + Changelog are mutable; content is append-or-correct. Closure transition is a Changelog entry. |
| `CLAUDE.md` | [`../CLAUDE.md`](../CLAUDE.md) | Edits land via PR (P9). "Fase Harvest" section is the canonical home of P1–P9; changes there require ADR linkage. |
| `TRUTH-MAP.md` | [`TRUTH-MAP.md`](TRUTH-MAP.md) | Updated in the same commit that ships the underlying code change. Anchor drift is corrected immediately (P7). |
| `AUTHORITY.md` | this file | Update by PR; changes to the tier definitions or precedence rules deserve a brief commit-message rationale. |
| `operations/runtime-invariants.md` | [`operations/runtime-invariants.md`](operations/runtime-invariants.md) | Updated whenever an invariant is added, removed, or the enforcement mechanism changes. |
| `ARCHITECTURE.md` | [`ARCHITECTURE.md`](ARCHITECTURE.md) | T1 because it carries durable structural principles. Updates land alongside the ADR that justifies the change. |

**ADR vs PRD scope split:**

- **ADRs answer "why" / "how"** — mechanism, durable structural
  decisions, alternatives rejected.
- **PRDs answer "what" / "when"** — scope, ondas, acceptance
  criteria, expected ADRs.

PRDs cite ADRs as the mechanism backing a scope claim; ADRs cite
PRDs as the program context that motivated the decision.

---

## T2 — Operational

T2 documents describe **how to run, develop, test** the foundry.
They are updated in-place by any contributor — no ADR required
for routine edits (a typo fix in `DEVELOPMENT.md` does not need
a meta-decision). T2 docs **do not define architecture**; if they
appear to, that content belongs in T1 instead.

| Document | Scope | Updated when |
|---|---|---|
| [`RESUMPTION.md`](RESUMPTION.md) | Current state, known gaps, next concrete step | Phase transitions, gap discoveries, gap resolutions, significant features shipped |
| [`DEVELOPMENT.md`](DEVELOPMENT.md) | Daily workflow, commands, conventions | Workflow changes, new make targets, tool upgrades |
| [`RUNTIME.md`](RUNTIME.md) | Binaries, streams, ports, KV buckets, subjects (concrete catalogue) | Stream added, port reassigned, KV bucket added |
| [`HTTP-API.md`](HTTP-API.md) | HTTP endpoints, conditional registration table | Endpoint added or its dep gate changes |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | PR workflow, AI-agent protocols, code rules | New rule, refined protocol, lessons learned |
| [`GLOSSARY.md`](GLOSSARY.md) | Terminology with system-specific meaning | Term added, term redefined, term retired |
| [`operations/*.md`](operations/) | Backups, deployment, smoke tests, troubleshooting, GitHub settings | Operational reality changes (new procedure, new tool) |
| [`domain/*.md`](domain/) | Per-domain deep dives | Domain logic evolves enough that the dive needs to follow |
| [`programs/README.md`](programs/) | PRD convention, status taxonomy, format | Convention itself changes (rare) |
| [`README.md`](README.md) (this dir's, and root) | Index pages | Files added/removed in the directory |

---

## T3 — Evolutionary

T3 documents are **proposals not yet binding**. Consult them for
direction; **do not treat them as canonical**. Code does not have
to conform to them; reviewers can reject changes that lean on T3
content as if it were T1.

| Family | Path / pattern | Promotion path |
|---|---|---|
| ADRs (Proposed) | `decisions/NNNN-*.md` with `Status: Proposed` | Promote to T1 when accepted by the maintainer (status flipped to `Accepted` in the same commit that ships the implementing code, or for foundation ADRs per Harvest P7 with explicit promotion criteria) |
| PRDs (in-flight) | `programs/PROGRAM-*.md` with `Status: Active` and unmet acceptance criteria | Remain T2/T3 hybrid until all acceptance criteria are met and Status flips to `Closed` |
| Design-meta candidates | RESUMPTION → "design-meta candidates" (M-list) | Promote to ADR when discussion crystallises, otherwise stay deferred |
| In-flight prompt scaffolds | Not currently in `docs/` (live in chat / `/tmp`) | Promote to PRD if they grow into a multi-onda effort |

Note: as of Onda H-2 closure (2026-05-24), the foundry has **seven
`Status: Proposed` ADRs** on `main`: ADRs 0017–0023, the Foundation
ADRs of the Fase Harvest. They are T3 until promoted to `Accepted`
by their implementing ondas (H-3, H-4, H-6, H-7, H-10 per each
ADR's "Promoção para Accepted" section). ADR-0023 may legitimately
remain `Proposed` indefinitely if its empirical triggers do not
fire. ADRs 0001–0016 remain `Accepted` (T1).

---

## T4 — Historical

T4 documents are **past records**. They are kept for traceability
but **never govern future decisions**. Citing a T4 document as
justification is invalid.

| Family | Where | Examples in current foundry |
|---|---|---|
| PRDs (Closed) | `programs/PROGRAM-*.md` with `Status: Closed` | None yet — PROGRAM-0001 still Active |
| ADRs (Superseded / Deprecated) | `decisions/NNNN-*.md` with `Status: Superseded by N` | None yet |
| RESUMPTION past-phase sections | The "Phase N — closed summary" subsections in `RESUMPTION.md` | Phase 3 summary, Phase 4 outlook tables, P5.0 audit narrative |
| Prior commit snapshots | `git log` / `git show <SHA>:docs/<file>` | Any retired content; not a path in the live tree |

T4 content occasionally lives in the same file as T1/T2 content
(e.g., RESUMPTION.md has both current state and closed-phase
summaries). The relevant tier is determined by the **section** of
the document, not the file as a whole.

---

## Precedence rules

1. **T1 always wins.** A closed RESUMPTION section (T4) cannot
   override an Accepted ADR (T1) — even if the RESUMPTION section
   is more recently dated.
2. **More specific wins within a tier.** ADR-0008 on single-writer
   beats a generic claim in ARCHITECTURE.md; a domain doc beats a
   generic GLOSSARY entry; PRD acceptance criteria beat PRD
   narrative.
3. **PRD vs ADR within T1:** PRDs win on **scope** ("what is in
   this phase"); ADRs win on **mechanism** ("how do we do it").
4. **Code vs T1 doc:** if they disagree, **the code is reality** —
   either the doc needs updating (correct path) or the code is the
   bug (fix path). Pick deliberately. (CLAUDE.md Core operating
   protocols #1: "Code is the source of truth".)
5. **No document-chain citations.** A change should cite the T1
   document directly, not "section X of RESUMPTION which references
   ADR-Y". Cite ADR-Y.

---

## When to promote a document between tiers

| From → To | Trigger |
|---|---|
| T3 → T1 | Proposed ADR is accepted by maintainer (Status flips to `Accepted`) |
| T1 → T4 | ADR is superseded (write the superseding ADR; mark old as `Superseded by ADR-N`) |
| T3 → T4 | PRD acceptance criteria all met → Status flips to `Closed` |
| Audit refresh | New RESUMPTION audit produced → previous narrative becomes T4 historical detail |
| Stays T2 | Operational docs are edited in place; no tier change |

Promotion is a **deliberate** act. A commit that promotes a
document should say so explicitly in the commit message (e.g.,
"promote ADR-0023 from Proposed to Accepted; implementing code at
internal/foo/bar.go").

---

## Classifying a new document

Before adding a new doc, answer one question:

| Question | → Tier |
|---|---|
| Does code / architecture **must** conform to it? | **T1** — requires ADR / PRD process |
| Does it describe **how** to operate / develop / test? | **T2** — update in-place, no ADR |
| Is it a **proposal** for future change? | **T3** — track status; promote via ADR when accepted |
| Is it a **record** of completed work? | **T4** — append-only; never governs |

If unsure, default to **T4** (or T2 for operational notes). It is
easy to promote later; hard to demote without confusion.

---

## File-to-tier inventory (2026-05-24)

Every file currently in `docs/` (and the canonical `CLAUDE.md`),
mapped to its tier.

### Root-level docs (foundry)

| File | Tier | Reason |
|---|---|---|
| [`../CLAUDE.md`](../CLAUDE.md) | T1 | Operating instructions + canonical Fase Harvest P1–P9 |
| [`README.md`](README.md) | T2 | Index page |
| [`ARCHITECTURE.md`](ARCHITECTURE.md) | T1 | Durable structural principles |
| [`RUNTIME.md`](RUNTIME.md) | T2 | Concrete operational catalogue (changes as the system grows) |
| [`HTTP-API.md`](HTTP-API.md) | T2 | Endpoint catalogue (operational) |
| [`DEVELOPMENT.md`](DEVELOPMENT.md) | T2 | Daily workflow |
| [`RESUMPTION.md`](RESUMPTION.md) | T2 (current sections) / T4 (closed-phase sections) | State sentinel |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | T2 | PR workflow + AI agent institutional knowledge |
| [`GLOSSARY.md`](GLOSSARY.md) | T2 | Terminology |
| [`TRUTH-MAP.md`](TRUTH-MAP.md) | T1 | Capability × evidence cross-reference |
| [`AUTHORITY.md`](AUTHORITY.md) (this file) | T1 | Tiering definition |

### `decisions/`

| File | Tier | Reason |
|---|---|---|
| [`decisions/README.md`](decisions/README.md) | T2 | Index page |
| `decisions/0001…0016-*.md` (16 files) | T1 | All Accepted ADRs |
| `decisions/0017…0023-*.md` (7 files) | T3 | Foundation ADRs (Proposed) — promoted to T1 by implementing ondas H-3/H-4/H-6/H-7/H-10 |

### `programs/`

| File | Tier | Reason |
|---|---|---|
| [`programs/README.md`](programs/README.md) | T2 | PRD convention |
| [`programs/PROGRAM-0001-foundation.md`](programs/PROGRAM-0001-foundation.md) | T1 (Status: Active) | First PRD |

### `operations/`

| File | Tier | Reason |
|---|---|---|
| [`operations/README.md`](operations/README.md) | T2 | Index |
| [`operations/backups.md`](operations/backups.md) | T2 | Operational procedure |
| [`operations/deployment.md`](operations/deployment.md) | T2 | Operational procedure |
| [`operations/github-settings.md`](operations/github-settings.md) | T2 | Operational reference |
| [`operations/smoke-tests.md`](operations/smoke-tests.md) | T2 | Operational procedure |
| [`operations/troubleshooting.md`](operations/troubleshooting.md) | T2 | Operational procedure |
| [`operations/runtime-invariants.md`](operations/runtime-invariants.md) | T1 | Top-N invariants enforced by gates |
| [`operations/slo.md`](operations/slo.md) | T1 (template) / T2 (per-target) | SLI/SLO definitions are T1; concrete targets become T2 once measured |

### `domain/`

| File | Tier | Reason |
|---|---|---|
| [`domain/README.md`](domain/README.md) | T2 | Index |
| `domain/{configctl,observation,evidence,signal,decision,strategy,risk,execution,effectiveness,pairing}.md` (10 files) | T2 | Per-domain deep dives (concrete, operational) |

---

## Changelog

- **2026-05-24** — Initial version, shipped as H-1 deliverable.
  T1–T4 hierarchy declared with concrete `docs/` inventory.
  T3 currently has zero entries (no Proposed ADRs on `main`);
  T4 has past-phase RESUMPTION sections and that is all.
  Companion to TRUTH-MAP (T1) and runtime-invariants (T1) in
  the same wave.
- **2026-05-24** — Onda H-2 closure: seven new ADRs (0017–0023,
  Foundation ADRs of the Fase Harvest) landed with `Status:
  Proposed`. T3 now holds these seven entries; the file-to-tier
  inventory and the T3-section note are updated to reflect this.
  ADRs are promoted T3 → T1 by their implementing ondas (H-3,
  H-4, H-6, H-7, H-10) in the commit that ships the supporting
  code; ADR-0023 may legitimately remain T3 indefinitely if its
  empirical triggers do not fire.
