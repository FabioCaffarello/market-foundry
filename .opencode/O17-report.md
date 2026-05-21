# O17 Report

## Summary

O17 defines the next documentation refactor as a joint topology problem across
`.opencode` and `docs/`, not as two unrelated trees.

Current state is asymmetric:

- `.opencode` is already small, bounded, and close to its intended role.
- `docs/` still carries too many active meta-docs and too much architecture
  surface mixed with wave evidence, gates, closure narratives, and operational
  reading aids.

The target is not a total rewrite.
The target is a disciplined compression:

- `.opencode` remains the real OpenCode navigation layer;
- `docs/` is reduced to active human documentation by context;
- stage evidence and historical rationale stay traceable without competing with
  primary entrypoints.

## Current-State Diagnosis

### Verified inventory

- `.opencode`: 34 files
- `docs/`: 1302 files
- `docs/operations/`: 61 files
- `docs/tooling/`: 24 files
- `docs/architecture/`: 567 files
- `docs/stages/`: 403 files
- `docs/archive/`: 246 files

### What is already working

- `.opencode` has explicit scope control, a bounded target tree, one agent, two
  profiles, and four context areas only.
- root entrypoints already define the intended owner surfaces:
  `README.md`, `DEVELOPMENT.md`, `docs/README.md`, `docs/operations/README.md`,
  `docs/tooling/README.md`, `docs/architecture/README.md`,
  `docs/stages/INDEX.md`, and `docs/archive/README.md`.
- the repository already distinguishes owner docs, reference docs, and
  historical surfaces in multiple places.
- stage evidence and archive surfaces already exist, so traceability does not
  require keeping historical material in primary directories.

### Structural problems

#### 1. `.opencode` is healthier than `docs/`

`.opencode` already behaves like a navigation and semantic-compression layer.
The risk is not that `.opencode` is too small.
The risk is that `docs/` still tries to solve navigation, support governance,
operating model, lifecycle policy, sustainability, prioritization, and history
in too many active files.

#### 2. `docs/operations/` is acting as both owner surface and policy warehouse

`docs/operations/` currently mixes at least five classes of material:

- true owner-human docs for daily development and support workflows;
- reference-human docs that deepen those workflows;
- documentation-governance docs about the documentation system itself;
- repository-platform strategy and sustainability essays;
- historical bridge docs from prior cleanup waves.

That produces fan-out and weakens the promise that one recurring question should
have one obvious home.

#### 3. `docs/architecture/` still contains too much stage-shaped material

A significant portion of `docs/architecture/` is not durable architecture.
The filenames alone show large active clusters of charter-and-scope-freeze,
gate, evidence-matrix, next-wave, gains-tradeoffs-and-open-debts, closure, and
reconciliation documents.

Those are valuable, but they are usually historical evidence, wave governance,
or closure rationale rather than primary architecture owners.

#### 4. documentation segmentation is still repository-centric, not user-centric

The current top-level documentation split is:

- operations
- tooling
- architecture
- stages
- archive

This is better than the older tree, but it still lacks explicit human contexts
for:

- product
- development

That makes the operations surface absorb development workflow, repository governance,
runbook routing, and documentation-system governance at the same time.

#### 5. active indexes are trying to compensate for too many active docs

The repo has good indexes, but too many of them must explain:

- what is canonical;
- what is only reference;
- what is historical;
- what changed in previous cleanup waves.

When several indexes need to keep saying "this other file is the real owner",
the topology is still too broad.

## Surface Classification Rules

Use these rules before moving any file.

| Class | Definition | Keep where |
|---|---|---|
| `owner-human` | Answers a current recurring human question directly and canonically | `docs/` primary surfaces |
| `reference-human` | Deepens or operationalizes an owner doc without owning the topic | `docs/` beside owner or under same context |
| `opencode-context` | Short navigation, compression, handoff, or task routing for OpenCode | `.opencode/` only |
| `historical-evidence` | Records wave intent, proof, decision sequence, closure, or superseded rationale | `docs/stages/` or `docs/archive/` |
| `redundant` | Duplicates an active answer without unique durable value | remove or merge |

### Classification tests

Assign a file to `owner-human` only if all are true:

- a contributor would reasonably start there today;
- the question recurs outside one wave or one cleanup ceremony;
- the file describes present rules or present usage;
- the file would still be worth keeping after stage history is ignored.

Assign a file to `reference-human` when:

- it clarifies a canonical owner;
- it adds examples, decision tables, or detailed catalogs;
- removing it would not erase the primary answer.

Assign a file to `opencode-context` when:

- the content is short;
- it routes work to the real owner docs or code entrypoints;
- it compresses navigation cost for an agent or contributor;
- it does not try to own policy or long-form human guidance.

Assign a file to `historical-evidence` when any are true:

- it is tied to a numbered wave, tranche, charter, gate, closure, or matrix;
- it evaluates whether a wave passed rather than defining the lasting rule;
- it preserves a now-superseded intermediate structure;
- it exists mainly for auditability, not daily use.

Assign a file to `redundant` when:

- another active file already answers the same recurring question;
- the file survives only because an older cleanup created it;
- its unique value can be reduced to a short section in an owner doc or archive note.

## Joint Target Topology

### `.opencode`

`.opencode` should stay minimal.
No major expansion is needed.
Its current four-context model is the right shape:

```text
.opencode/
  README.md
  TARGET-TREE.md
  O-reports
  config.json
  opencode.json
  agent/core/foundry-agent.md
  profiles/{essential,developer}/profile.json
  context/
    navigation.md
    repo/
    runtime/
    change/
    intelligence/
```

Role by area:

- `repo/`: repository entrypoints, doc ownership, boundaries
- `runtime/`: stack/proof/troubleshooting routing
- `change/`: safe-change loop, impact, validation, stage handoff
- `intelligence/`: `make`/`raccoon-cli`/inspection routing

`.opencode` must not become:

- a second human documentation tree;
- a long governance catalog;
- a repository history layer;
- a substitute for `docs/development/`, `docs/tooling/`, or `docs/architecture/`.

### `docs/`

Target top-level topology:

```text
docs/
  README.md
  product/
    README.md
  development/
    README.md
  tooling/
    README.md
  architecture/
    README.md
  stages/
    INDEX.md
  archive/
    README.md
```

Rules for each surface:

- `docs/product/`: active human product docs only; create minimally now, even if
  initially small or empty, so the taxonomy matches intended growth.
- `docs/development/`: active human docs for workflow, entrypoints,
  troubleshooting, support-surface policy, stage governance, and repository
  navigation.
- `docs/tooling/`: tooling-internal reference and rule catalogs.
- `docs/architecture/`: durable system architecture, patterns, boundaries,
  runtime invariants, and a small set of canonical runbooks whose value is
  architectural rather than merely operational.
- `docs/stages/`: immutable stage reports and stage index only.
- `docs/archive/`: superseded or historical non-canonical material only.

## Target Classification By Current Surface

### Keep as `opencode-context`

- `.opencode/README.md`
- `.opencode/context/`
- `.opencode/profiles/`
- `.opencode/agent/core/foundry-agent.md`
- `.opencode` O-reports

No long-form human doc from `docs/` should migrate into `.opencode`.
Only short routers or compressed navigation belong here.

### Keep in `docs/` as `owner-human`

Root:

- `README.md`
- `DEVELOPMENT.md`
- `docs/README.md`

Target `docs/development/` owners, mostly by consolidation of current
`docs/operations/`:

- documentation ownership and taxonomy
- development lifecycle and command-surface entrypoints
- troubleshooting and onboarding
- repository navigation maps
- `make` vs `raccoon-cli` contract
- stage execution support and stage-history reading rules
- repository support-surface policy

Representative keeper set:

- `documentary-ownership-and-canonical-navigation.md`
- `documentation-governance-entrypoints-and-taxonomy.md`
- `development-lifecycle-entrypoints-and-canonical-flows.md`
- `developer-onboarding-and-troubleshooting-guide.md`
- `repository-navigation-maps-entrypoints-and-maintenance-rules.md`
- `repository-support-surface-canonical-model.md`
- `make-and-raccoon-cli-contract.md`
- `stage-tooling-and-execution-governance-support.md`

Target `docs/tooling/` owners:

- `docs/tooling/README.md`
- `docs/tooling/cli-overview.md`
- the actual CLI guardrail and drift catalogs
- tooling lifecycle and modularity docs that still govern the tool today

Target `docs/architecture/` owners:

- `system-vision.md`
- `system-principles.md`
- `market-foundry-evolution-playbook.md`
- `stage-definition-of-done.md`
- `anti-debt-checklist.md`
- `monorepo-structure-and-engineering-conventions.md`
- `prohibited-carryovers.md`
- durable domain-design, pattern, invariant, and boundary docs

### Keep in `docs/` as `reference-human`

These remain active only if they deepen a kept owner:

- detailed target catalogs
- supporting reference guides
- secondary runbooks
- localized decision tables

Representative examples:

- `makefile-targets-reference-and-conventions.md`
- `scripts-catalog-and-usage-guide.md`
- `repository-metadata-indexes-and-developer-navigation-system.md`
- selected tooling governance references
- selected architecture runbooks whose owner remains architectural

### Move to `docs/stages/` as `historical-evidence`

Any active doc that is effectively wave evidence or wave governance should move
to stage history when it maps cleanly to a stage or tranche narrative.

Default patterns:

- charter-and-scope-freeze files
- evidence-gate files
- closure files
- reconciliation files
- gains-tradeoffs-and-open-debts files
- next-wave files

This especially applies to architecture files that preserve sequence rather than
durable rules.

### Move to `docs/archive/` as `historical-evidence`

Archive when the file is historically useful but not cleanly part of the stage
narrative, or when it documents a previous documentation cleanup rather than the
current system.

Representative examples:

- `docs/operations/documentation-reorganization-and-operational-navigation.md`
- older support-surface audits and improvement matrices
- superseded topology maps
- historical bridge docs that are still useful for archaeology

### Remove or merge as `redundant`

The biggest consolidation opportunity is in current `docs/operations/`.
The following clusters should be merged into fewer owner docs:

- development-platform strategy, readiness, prioritization, checkpoint, and
  hotspot docs
- support-surface sustainability, lifecycle, and retirement docs
- multiple overlapping workflow/lifecycle/unification docs
- multiple overlapping documentation-governance and hardening docs

Representative merge candidates:

- `developer-workflow-unification.md` into the lifecycle owner set
- `development-environment-architecture-and-lifecycle.md` plus
  `development-lifecycle-entrypoints-and-canonical-flows.md` into one owner and
  one compact reference
- `documentation-system-hardening.md` into archive, with the live rules staying
  in the taxonomy/ownership owner docs
- `makefile-command-ergonomics-and-hardening.md` into
  `makefile-targets-reference-and-conventions.md`
- `scripts-normalization-and-harness-hygiene.md` into
  `scripts-catalog-and-usage-guide.md`
- multiple repository-platform strategic documents into one active owner and a
  small number of references

## Consolidation Rules

### Rule 1. Prefer context owners over topic proliferation

If a human asks "how do I work here?", "how do I validate?", "where does this
doc belong?", or "how do stages fit into current work?", the answer should live
under `docs/development/`, not across many quasi-strategic operations files.

### Rule 2. Architecture keeps durable rules, not wave paperwork

A document stays in `docs/architecture/` only if it defines:

- a durable invariant;
- a durable pattern;
- a durable domain or runtime boundary;
- a current architecture runbook whose value depends on system design, not just
  operator task routing.

Otherwise it belongs in `docs/stages/` or `docs/archive/`.

### Rule 3. `.opencode` compresses; it does not own

If a file would be useful to agents but also useful to humans as a primary
document, the owner stays in `docs/` and `.opencode` links to it.

### Rule 4. Product context is explicit even before it is large

The taxonomy should reserve `docs/product/` now.
That prevents future product material from being misplaced into architecture or
development docs.

### Rule 5. historical traceability is preserved by redirection, not by
primary-surface clutter

When moving or archiving a file:

- keep a clear successor link in the moved file or its replacement;
- preserve stage links and chronology;
- update indexes;
- do not keep the old file active in place just to preserve history.

## Migration Rules

### Phase A. Reclassify without broad moves

First produce a repository-wide classification table:

- `owner-human`
- `reference-human`
- `opencode-context`
- `historical-evidence`
- `redundant`

Do this by directory and filename pattern first, then review exceptions.

### Phase B. Establish the target human contexts

Create the new top-level surfaces:

- `docs/development/`
- `docs/product/`

Do not move everything immediately.
First create entrypoint READMEs and define the owner docs that will survive the
merge.

### Phase C. Collapse `docs/operations/` into `docs/development/`

Use `docs/operations/` as the source pool, not the permanent target.

Expected result:

- a smaller `docs/development/` owner set;
- a smaller reference set;
- archive for bridge and cleanup-history docs;
- deletion of merged duplicates.

### Phase D. Purge historical architecture pollution

Move non-durable wave artifacts out of `docs/architecture/` in batches:

1. charters and scope freezes
2. gates and evidence matrices
3. gains/tradeoffs and next-wave docs
4. reconciliation and closure narratives

Each batch should update:

- `docs/architecture/README.md`
- `docs/stages/INDEX.md` or `docs/archive/README.md`
- any owner doc that now points to the archived/staged artifact

### Phase E. Retune `.opencode` only after the human topology stabilizes

After the docs tree is reduced:

- update `.opencode/context/repo/documentation-topology.md`
- update any owner-doc links that moved from the operations surface to the
  development surface
- keep the four-context topology unless a real recurring routing gap remains

## Safe Incremental Execution Plan

### Increment 1. Classification baseline

- freeze a machine-readable inventory of current docs by class
- mark obvious historical patterns in `docs/architecture/`
- mark obvious merge clusters in `docs/operations/`

Exit criteria:

- every active directory has a classification policy;
- no file is moved yet;
- the migration backlog is explicit.

### Increment 2. Create the new human entrypoints

- add `docs/development/README.md`
- add `docs/product/README.md`
- update `docs/README.md` to route by human context

Exit criteria:

- the target taxonomy is visible before content moves;
- no current canonical answer is lost.

### Increment 3. Consolidate development docs

- merge overlapping workflow/support/governance docs into the new
  `docs/development/` owner set
- archive bridge docs that only record how prior reorganizations happened
- leave thin compatibility pointers only where needed temporarily

Exit criteria:

- daily development questions resolve through `README.md`,
  `DEVELOPMENT.md`, `docs/README.md`, and `docs/development/README.md`
- `docs/operations/` is empty, removed, or reduced to short compatibility
  stubs scheduled for deletion

### Increment 4. Drain historical architecture artifacts

- move stage-shaped docs from `docs/architecture/` into `docs/stages/` or
  `docs/archive/`
- keep only durable architecture and current architecture runbooks

Exit criteria:

- `docs/architecture/README.md` stops acting as a wave-history hub
- architecture filenames are mostly patterns, domains, rules, and runbooks

### Increment 5. Final `.opencode` reconciliation

- update `.opencode` links and owner anchors
- confirm `.opencode` remains short and bounded
- add or adjust cheap consistency checks only if objective drift remains likely

Exit criteria:

- `.opencode` points at the new human owner surfaces cleanly
- no `.opencode` file becomes a long-form human doc

## Tradeoffs

- Keeping traceability increases migration work because redirects and index
  updates must be explicit.
- Creating `docs/product/` before it is full may look premature, but it is
  cheaper than letting future product docs leak into the wrong contexts.
- Moving wave-shaped architecture docs out of `docs/architecture/` will leave
  some old links noisier in the short term, but it reduces long-term confusion.
- Consolidating operations docs means some nuanced historical distinctions will
  move from active docs into archive or stages. That is intentional.

## Non-Goals

- rewriting the architecture corpus stylistically;
- deleting useful history;
- turning `.opencode` into a long human manual;
- preserving every active filename as a first-class surface;
- treating every previous support-wave document as still deserving active status.

## Evolution Criteria

The refactor is succeeding when all are true:

- one recurring human question maps to one obvious active owner doc;
- `.opencode` points to those owners without duplicating them;
- `docs/architecture/` mostly contains durable architecture, not wave evidence;
- `docs/stages/` and `docs/archive/` carry history without competing for entry;
- root entrypoints get shorter, not longer, as the system evolves.

The refactor is failing when any are true:

- `.opencode` starts absorbing long governance prose;
- `docs/development/` simply recreates the current `docs/operations/` sprawl;
- architecture keeps stage paperwork active in place;
- multiple active indexes still need to explain which other file is the real owner.
