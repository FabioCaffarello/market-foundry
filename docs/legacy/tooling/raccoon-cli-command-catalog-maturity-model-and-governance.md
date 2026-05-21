# Raccoon CLI Command Catalog Maturity Model And Governance

## Purpose

This document maps the current `raccoon-cli` command catalog by role,
criticality, and maturity, then states the governance implications for each
surface.

## Current Catalog Summary

| Surface | Commands | Criticality | Maturity | Governance note |
|---|---|---|---|---|
| `check` | `repo`, `topology`, `contracts`, `bindings`, `arch`, `drift`, `gate` | high | `stable core` | canonical guard-rail surface |
| `inspect` | `symbol`, `lsp`, `contract-usage`, `coverage` | medium-high | `stable core` | canonical expert inspection surface |
| `change` | `impact`, `tdd`, `briefing`, `recommend`, `rename` | high | `stable core` | canonical change-planning surface |
| utility | `snapshot`, `snapshot-diff`, `baseline-drift` | medium | `stable utility` | durable specialist tools |
| legacy | `legacy runtime-smoke` | low | `legacy` | retained only for compatibility |
| compatibility aliases | `doctor`, `quality-gate`, `symbol-trace`, `impact-map`, `tdd`, and peers | low-medium | `legacy` | hidden dispatch compatibility only |

## Catalog By Command

| Command | Role | Criticality | Maturity | Notes |
|---|---|---|---|---|
| `check repo` | repository structure sanity | high | `stable core` | first-line guard rail |
| `check topology` | runtime topology audit | high | `stable core` | central to service/config/compose alignment |
| `check contracts` | contract and transport audit | high | `stable core` | central to NATS contract discipline |
| `check bindings` | runtime binding validation | high | `stable core` | central to config-to-runtime alignment |
| `check arch` | architecture boundary enforcement | high | `stable core` | critical governance surface |
| `check drift` | docs/config/source drift detection | high | `stable core` | keeps support artifacts honest |
| `check gate` | consolidated guard-rail orchestration | high | `stable core` | key umbrella command |
| `inspect symbol` | symbol trace | medium-high | `stable core` | primary codeintel inspection flow |
| `inspect lsp` | semantic enrichment | medium | `stable core` | specialist but actively governed |
| `inspect contract-usage` | contract usage mapping | medium-high | `stable core` | distinct inspection value |
| `inspect coverage` | coverage and quality-map visibility | medium | `stable core` | decision support for change validation |
| `change impact` | blast-radius mapping | high | `stable core` | central to safe change work |
| `change tdd` | validation planning | high | `stable core` | canonical pre-change guidance |
| `change briefing` | concise auditable summary | medium | `stable core` | supports disciplined context handoff |
| `change recommend` | validation recommendations | medium | `stable core` | useful composition surface |
| `change rename` | rename safety | medium | `stable core` | specialized but durable |
| `snapshot` | baseline generation | medium | `stable utility` | specialist, durable, not daily-driver |
| `snapshot-diff` | baseline comparison | medium | `stable utility` | useful but narrower than grouped flows |
| `baseline-drift` | semantic drift from baseline | medium | `stable utility` | complements snapshots, not a core workflow |
| `legacy runtime-smoke` | historical runtime wrapper | low | `legacy` | superseded by `make smoke*` |

## Core Diagnosis

### Central and mature surfaces

The strongest and clearest command families are:

- `check`, because it maps directly to repository governance and validation;
- `change`, because it supports the default development loop;
- most of `inspect`, because it offers structural visibility without competing
  with runtime workflows.

These commands are central because they reinforce the repository contract rather
than bypassing it.

### Peripheral but justified surfaces

The snapshot family is peripheral in frequency, but justified in value.

It should remain available because:

- it serves drift analysis and baseline comparison cleanly;
- the commands are conceptually coherent together;
- folding them into `change` or `inspect` would make those groups noisier.

This is why they fit `stable utility`, not `stable core`.

### Fragile or legacy surfaces

The genuinely fragile surface is runtime-oriented historical compatibility:

- `legacy runtime-smoke`;
- the deep gate profile insofar as it invokes the legacy helper;
- hidden flat aliases that preserve pre-taxonomy command habits.

These surfaces should stay contained and clearly labeled so contributors do not
confuse them with the canonical workflow.

### Redundancy and overlap signals

Current overlap signals are manageable, not severe:

- grouped commands and flat aliases share the same semantics by design;
- `inspect lsp` and `inspect symbol --lsp` are adjacent but still distinct
  enough because one is explicit enrichment and the other is full tracing;
- `change recommend`, `change tdd`, and `change briefing` all work on change
  guidance, but they answer different levels of decision support.

The main governance risk is documentation drift, not implementation conflict.

## Maturity Model

The catalog uses two axes:

- `criticality`: how important the command is to the repository workflow;
- `maturity`: how stable and promotable the surface is.

### Criticality levels

| Level | Meaning |
|---|---|
| high | repository workflow or governance depends on it |
| medium | recurring expert use, but not required on every change |
| low | compatibility or narrow fallback only |

### Maturity levels

| Level | Meaning |
|---|---|
| `stable core` | default, canonical, recurring surface |
| `stable utility` | durable specialist surface |
| `experimental` | bounded proving surface |
| `legacy` | deprecated or compatibility-only surface |

Governance consequence:

- high criticality commands need the strongest help text, test confidence, and
  documentation clarity;
- low criticality commands must not dominate the public narrative.

## Governance Rules For The Catalog

### Stable core governance

- keep examples current and canonical;
- prefer subcommand growth over creating parallel top-level commands;
- treat naming changes as migrations, not casual churn;
- maintain reliable exit and output semantics.

### Stable utility governance

- keep documented, but do not over-promote;
- resist moving niche flows into core groups unless repetition proves it helps;
- keep behavior crisp and specialist.

### Experimental governance

- require explicit label in help and docs;
- set promotion or retirement criteria when introduced;
- avoid creating compatibility debt around experimental naming.

### Legacy governance

- freeze semantics unless safety requires adjustment;
- keep public docs minimal and replacement-oriented;
- remove only when compatibility value materially drops.

## Recommended Catalog Direction

Near-term direction for the current catalog:

1. Preserve the grouped canonical taxonomy as the dominant surface.
2. Keep snapshot commands as stable utility, not as a new major group.
3. Keep runtime-smoke and flat aliases explicitly outside the promoted surface.
4. Avoid adding new top-level commands unless they are clearly durable utility
   commands with distinct value.

## Review Checklist For Future Changes

Before merging a command-surface change, verify:

1. The command has an obvious lifecycle state.
2. The command does not duplicate an existing surface.
3. The canonical docs use the preferred invocation form.
4. Any alias remains hidden and behaviorally identical.
5. The change strengthens the CLI as a development tool rather than widening it
   into a parallel platform.
