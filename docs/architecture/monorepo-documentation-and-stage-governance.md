# Monorepo Documentation and Stage Governance

## Purpose

This document defines how documentation is organized in the market-foundry monorepo and how stage-based evolution governance works. It ensures documentation grows predictably and stages remain a useful evolutionary record.

---

## Documentation Structure

```
docs/
├── architecture/       # Canonical architecture decisions and patterns
├── stages/             # Stage completion reports (evolutionary record)
└── tooling/            # CLI and tooling documentation
```

### Root-Level Documents

| File | Purpose | Audience |
|------|---------|----------|
| `README.md` | Project overview and quick orientation | New contributors |
| `DEVELOPMENT.md` | Developer workflow, targets, module scoping | Active developers |
| `AGENTS.md` | AI agent operating contract | AI agents (Claude, etc.) |

These three files are the **entry points** into the repository. They must stay concise and current.

### Architecture Documents (`docs/architecture/`)

Architecture docs are the **canonical source of truth** for how the system works and how it must grow. They are binding — code must conform to them.

#### Document Categories

| Category | Purpose | Examples |
|----------|---------|---------|
| **Foundation** | Core identity and principles | `system-vision.md`, `system-principles.md` |
| **Conventions** | Structural and naming rules | `naming-conventions-*.md`, `monorepo-structure-*.md` |
| **Patterns** | Canonical implementation patterns | `gateway-pattern.md`, `derive-family-processor-pattern.md`, `projection-families-model.md` |
| **How-To** | Step-by-step guides for expansion | `how-to-introduce-new-runtimes-domains-and-families.md`, `family-runtime-registration-rules.md` |
| **Governance** | Playbook, checklists, rules | `market-foundry-evolution-playbook.md`, `anti-debt-checklist.md`, `stage-definition-of-done.md` |
| **Audit** | Sanitization records, prohibited patterns | `repository-sanitization-audit.md`, `prohibited-carryovers.md` |
| **Domain** | Domain-specific architecture | `signal-domain-architecture.md`, `evidence-domain-architecture.md` |

#### Writing Architecture Documents

Rules for new architecture docs:

1. **One concern per document** — do not combine unrelated topics.
2. **Lead with purpose** — first section explains what the document solves.
3. **Be prescriptive** — state what to do and what not to do.
4. **Include code examples** — when documenting patterns, show the canonical form.
5. **Cross-reference, don't duplicate** — link to related docs instead of repeating their content.
6. **Name descriptively** — the filename should clearly indicate the content (e.g., `dependency-injection-and-composition-roots.md`, not `di.md`).

#### When to Create vs. Update

- **Create a new doc** when formalizing a new pattern or convention that has no existing canonical reference.
- **Update an existing doc** when a pattern evolves or a convention is refined.
- **Do not create a doc** for one-time decisions or temporary state — use the stage report for that.

### Tooling Documents (`docs/tooling/`)

Tooling docs describe the raccoon-cli analyzers, guardrails, and drift rules:

- `cli-overview.md` — CLI reference and capabilities.
- `cli-architecture-guardrails.md` — what the quality gate enforces.
- `cli-{domain}-guardrails.md` — domain-specific guardrail rules.
- `cli-{domain}-drift-rules.md` — domain-specific drift detection rules.

These are updated when raccoon-cli analyzers change — they mirror the tool's capabilities.

---

## Stage Governance

### What is a Stage?

A stage is a bounded unit of architectural evolution. Each stage has:

- A clear **objective** (what it achieves).
- A defined **scope** (what it changes).
- A completion **report** (what was done, what was not, what comes next).

Stages are numbered sequentially: `S06`, `S07`, ..., `S99`, `S100`.

### Stage Report Format

Stage reports live at `docs/stages/stage-{id}-{slug}-report.md` and follow this structure:

```markdown
# Stage S{id} — {Title}

## Objective
What this stage set out to achieve.

## Changes Made
What was actually done (files, patterns, fixes).

## Conventions Established
New rules or patterns formalized (if any).

## Drift Removed
Structural or naming inconsistencies corrected.

## Trade-offs and Limitations
What was intentionally deferred or accepted.

## Preparation for Next Stage
What the next stage should consider.
```

### Stage Rules

1. **Stages do not overlap** — each stage completes before the next begins.
2. **Stage reports are immutable** — once written, a stage report is not modified (it is a historical record).
3. **Stages reference architecture docs** — when a stage formalizes a convention, the convention goes in `docs/architecture/` and the stage report references it.
4. **Not every change needs a stage** — routine bug fixes, dependency updates, and small improvements do not require stage governance.

### Stage Numbering

- Pre-numbered stages (S06–S99): Foundation hardening and consolidation.
- S100+: Reserved for the next evolutionary phase.
- Named stages (no number): Historical milestones (sanitization, first-slice, recentralization, raccoon-cli).

### Relationship Between Stages and Architecture Docs

```
Stage Report (historical)          Architecture Doc (canonical)
─────────────────────────          ──────────────────────────
"We renamed PipelineScope     →    naming-conventions-for-domains-
 to PipelineDomain because..."      families-and-runtimes.md

"We introduced composition   →    dependency-injection-and-
 roots because run.go grew..."      composition-roots.md
```

- The **stage report** records the decision and its rationale at a point in time.
- The **architecture doc** captures the resulting convention as a living, canonical reference.
- When a convention changes in a later stage, the architecture doc is updated; the original stage report remains unchanged.

---

## Documentation Maintenance

### Keeping Docs Current

1. **AGENTS.md, DEVELOPMENT.md, README.md** — update whenever services, workflows, or structure change.
2. **Architecture docs** — update when the convention they describe is refined or extended.
3. **Tooling docs** — update when raccoon-cli analyzers change.
4. **Stage reports** — never update (immutable historical record).

### Avoiding Documentation Drift

- When making structural changes, check if any architecture doc needs updating.
- The `make check` quality gate can catch some forms of structural drift.
- Cross-reference documents instead of duplicating information — duplication is the primary source of doc drift.

### Documentation as Code

- All docs are Markdown, versioned in git alongside the code they describe.
- Docs evolve with the code — they are not an afterthought.
- If a doc contradicts the code, the code is the source of truth and the doc must be updated.

## Related Documents

- `stage-definition-of-done.md` — what constitutes a complete stage
- `anti-debt-checklist.md` — practical debt prevention
- `market-foundry-evolution-playbook.md` — evolution governance playbook
- `monorepo-structure-and-engineering-conventions.md` — structural conventions
