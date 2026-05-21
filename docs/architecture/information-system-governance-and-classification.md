# Information System Governance And Classification

## Purpose

This document defines the lightweight governance for the joint repository
information system formed by `.opencode/` and `docs/`.

The goal is to keep the split durable:

- `.opencode/` stays the real OpenCode configuration and compression layer;
- `docs/` stays the canonical human surface;
- `docs/stages/` and `docs/archive/` hold history without competing with active
  navigation;
- new documents are classified consistently before they create drift or
  duplicate ownership.

## Operating Model

The information system has four classes:

| Class | Meaning | Canonical location |
|---|---|---|
| `opencode-context` | short routing, compression, handoff, or safe-change context for OpenCode | `.opencode/` |
| `owner-human` | current recurring human answer with clear ownership | `docs/product/`, `docs/development/`, selected `docs/tooling/`, selected `docs/architecture/` |
| `reference-human` | deeper technical or supporting material that clarifies an owner doc | `docs/tooling/`, `docs/architecture/` |
| `historical-evidence` | stage proof, superseded rationale, reconciliation trail, or retired guidance | `docs/stages/`, `docs/archive/` |

Anything that does not earn one of these roles should be consolidated or
removed.

## Invariants

### `.opencode/`

`.opencode/` may contain only:

- navigation into owner docs and repository entrypoints;
- semantic compression of recurring repo context;
- short handoff context between sessions and agents;
- safe-change and validation routing tied to the real workflow;
- thin `make` and `raccoon-cli` guidance in support of canonical owners.

`.opencode/` must not contain:

- long-form policy or rationale;
- human owner maps;
- historical evidence, closure logs, or stage narratives as active guidance;
- generic taxonomies, marketplaces, workflow engines, or parallel command
  catalogs;
- any file whose real mission is better handled by a human doc under `docs/`.

### `docs/` active

Active `docs/` may contain only:

- current owner docs for recurring human questions;
- current reference docs that deepen those owners without competing with them;
- concise navigation surfaces that route readers into those owners.

Active `docs/` must not contain:

- superseded guidance left in place "just in case";
- stage-shaped evidence files competing with current answers;
- broad generic governance essays without a clear current owner mission;
- duplicate surfaces answering the same recurring question at the same level.

### `docs/stages/`

A document belongs in `docs/stages/` when it:

- records a numbered stage, wave, charter, implementation step, proof, gate, or
  closure;
- exists primarily for auditability and delivery traceability;
- would lose most of its value if current readers only needed the present rule.

Stage reports are evidence, not the owner of recurring workflow or architecture
guidance.

### `docs/archive/`

A document belongs in `docs/archive/` when it:

- was once useful but is no longer the current owner;
- preserves superseded structure, transitional rationale, or consolidation
  provenance;
- still has archeological value but should not compete with active navigation.

Archive material stays readable, but it is never the first-stop source of
truth.

### Consolidate Or Remove

Consolidate a document when:

- two active files answer the same recurring question;
- one file can absorb the durable answer without losing clarity;
- the weaker file adds only examples, history, or restated policy.

Remove a document when:

- it has no current owner mission;
- its only value is duplicated elsewhere;
- an archive note or short section in another owner doc fully preserves the
  durable signal.

## Classification Rules

Use these tests before creating or moving any document.

### Assign to `.opencode/`

Use `opencode-context` only when all are true:

- the content is short and task-shaped;
- it reduces navigation or handoff cost for OpenCode;
- it routes outward to real owner docs or code entrypoints;
- it does not need to become a human-owned source of truth.

### Keep in active `docs/`

Use `owner-human` only when all are true:

- the question recurs outside a single stage or cleanup ceremony;
- a contributor or operator should reasonably start there today;
- the answer describes present rules, present topology, or present usage;
- the file remains valuable even if stage history is ignored.

Use `reference-human` when:

- the file deepens a current owner;
- it adds technical detail, examples, or catalogs;
- removing it would not erase the primary answer.

### Send to `docs/stages/`

Use `historical-evidence` in `docs/stages/` when the document is tied to a
numbered delivery event and its main value is "what happened and what proved
it."

### Send to `docs/archive/`

Archive when the file is no longer active but still useful for rationale,
provenance, or archaeology.

## Anti-Duplication Rules

Use these rules to avoid competition between human owners and OpenCode context:

- Human ownership lives in `docs/`, never in `.opencode/`.
- `.opencode/` may summarize owner routes, but it must not clone owner maps,
  long policy, or broad narrative structure.
- When a current human answer changes, update the owner doc first and only then
  adjust `.opencode/` routing if navigation or handoff changed.
- If a new recurring question appears, first ask whether an existing owner doc
  can absorb it. Opening a new active surface is the last resort.
- Historical evidence must point outward to promoted docs; promoted docs should
  point back to history only when the historical trail materially helps.

## Lightweight Checks

The repository should detect only drift that changes ownership or navigation in
material ways.

Required checks:

- broken links across active support surfaces and `.opencode/`;
- orphaned active docs that are not reachable from their area README or owner
  map;
- competing owner-map entries for the same recurring subject;
- inconsistent navigation between root docs, area indexes, owner maps, and
  `.opencode/` routes;
- historical evidence reappearing in active primary surfaces;
- reintroduction of broad generic support taxonomies into active surfaces;
- `.opencode/` growth that starts mirroring human owner docs.

Non-goals for checks:

- cosmetic markdown linting;
- blocking on minor wording or formatting issues;
- forcing every architecture doc into one giant index;
- preventing legitimate growth when the owner docs and checks evolve together.

## Evolution And Maintenance Rules

Use this sequence for future changes:

1. Classify the new information before choosing a path.
2. Prefer updating an existing owner or reference doc over creating a new one.
3. If history is the main value, send it directly to `docs/stages/` or
   `docs/archive/`.
4. If OpenCode needs shorter routing, update `.opencode/` after the owner doc is
   correct.
5. Keep the checks aligned in the same change whenever the active topology
   changes.

Follow-through rules:

- When adding a new active owner doc, update the relevant area README, owner
  map, and consistency checks in the same change.
- When retiring an active doc, move or merge it before removing its links.
- When adding a new `.opencode` context file, update `TARGET-TREE.md`,
  navigation entrypoints, and `.opencode` consistency checks together.
- When a document stops owning a current answer, archive it instead of leaving
  it to decay in place.

## Expansion Criteria

Open a new active document only when all are true:

- the question genuinely recurs;
- no current owner can absorb it cleanly;
- the document has a stable owner mission;
- the navigation cost of not creating it is materially higher than the
  maintenance cost;
- the new file does not recreate a generic taxonomy or duplicate `.opencode/`
  compression.

Reject or defer expansion when any are true:

- the need is stage-specific or cleanup-specific;
- the document would mostly restate an existing owner;
- the value is mostly historical;
- the proposal widens navigation more than it reduces confusion.
