# O13 Report

## Diagnosis

`.opencode` is already close to the right shape for `market-foundry`, because
its real value is concentrated in `context/repo`, `context/runtime`,
`context/change`, and `context/intelligence`.

The misalignment is structural, not thematic:

1. the removed `.opencode` command area duplicates command discovery already owned by
   `Makefile`, `AGENTS.md`, `README.md`, `DEVELOPMENT.md`, and
   `docs/operations/`.
2. the removed subagent area under `.opencode/agent` introduces a durable delegation taxonomy that
   is not a repository owner surface and is not part of the Foundry mission.
3. The guard rail previously enforced thinness and reachability, but not the
   minimal target topology that keeps `.opencode` native to Foundry.
4. The codebase source of truth lives outside `.opencode`: the official
   workflow is `make check -> make tdd -> change -> make verify`, architecture
   stays `domain -> application -> adapters -> actors -> interfaces -> cmd`,
   and `raccoon-cli` is the real intelligence layer.

## Classification

| Area | Decision | Rationale |
|---|---|---|
| `.opencode/README.md` | adapt | Make the mission and minimal topology explicit |
| `.opencode/TARGET-TREE.md` | preserve | Formalize the approved target architecture |
| `.opencode/O12-report.md` | preserve | Historical evidence of the previous hardening stage |
| `.opencode/O13-report.md` | preserve | Records the reconciliation, tradeoffs, and rollout order |
| `.opencode/config.json` | preserve | Keep a single root agent binding |
| `.opencode/opencode.json` | preserve | Keep profile registration and context root |
| `.opencode/agent/core/` | adapt | Preserve one root Foundry agent, remove extra role taxonomy |
| removed subagent area under `.opencode/agent` | merge then remove | Move concern guidance into the bounded `context/intelligence/` navigation files and drop the durable surface |
| removed `.opencode` command area | remove | Redundant with `Makefile` and canonical docs |
| `.opencode/context/navigation.md` | preserve | Correct top-level router into owner docs |
| `.opencode/context/repo/` | preserve | Native repository-shape compression layer |
| `.opencode/context/runtime/` | preserve | Native runtime/proof routing layer |
| `.opencode/context/change/` | preserve | Correct place for the official change loop reminder |
| `.opencode/context/intelligence/navigation.md` | adapt | Keep focus on `raccoon-cli` and owner docs, not agent taxonomy |
| `.opencode/context/intelligence/` | adapt | Keep `raccoon-cli` and guard-rail routing without named subagent dependency |
| `.opencode/profiles/` | preserve | Minimal bundles remain useful and thin |
| `scripts/opencode-consistency-check.sh` | adapt | Enforce the approved minimal topology and concern map |
| `docs/operations/*opencode invariants*` | adapt | Reflect the new native-topology invariant in canonical docs |

## Rationale

The Foundry already has clear owners:

- `AGENTS.md` and `Makefile` define the official workflow contract.
- `README.md`, `DEVELOPMENT.md`, `docs/operations/`, `docs/tooling/`, and
  `docs/architecture/` define identity, workflow, operations, tooling, and
  governance.
- `raccoon-cli` owns structural inspection, impact analysis, and architecture
  safety.

Given that shape, `.opencode` should not be another platform. It should be a
thin, repository-native compression layer that makes those owners easier to
reach without creating alternate commands, alternate governance, or durable
agent taxonomies.

## Tradeoffs

- Removing the command area reduces OpenCode-local shortcuts, but eliminates a parallel
  command surface that would drift from `Makefile`.
- Removing durable subagents reduces explicit role labels, but keeps the tree
  closer to the real repository ownership model.
- Tightening the guard rail increases structural strictness, but keeps future
  `.opencode` growth honest and cheap to review.

## Non-Goals

- redefining the canonical developer workflow outside `AGENTS.md` and
  `Makefile`
- turning `.opencode` into a plugin, skill, or marketplace surface
- replacing `docs/stages/` with `.opencode` evidence tracking
- promoting `raccoon-cli` into a runtime orchestrator or proof-of-record layer
- cloning architecture or operations catalogs that already exist in canonical
  docs

## Minimal Target Architecture

`.opencode` should stay bounded to:

- `repo` for repository shape and canonical-owner routing
- `runtime` for stack/proof/troubleshooting routing
- `change` for the official change loop and stage-touch reminders
- `intelligence` for `raccoon-cli` navigation and short handoff compression

The formal tree lives in [`TARGET-TREE.md`](./TARGET-TREE.md).

## Incremental Plan

1. Remove the extra surfaces now: delete the `.opencode` command area and the
   subagent area under `.opencode/agent`.
2. Anchor the target shape now: update `.opencode/README.md`,
   `.opencode/agent/core/foundry-agent.md`,
   `.opencode/context/intelligence/navigation.md`, and add `TARGET-TREE.md`.
3. Harden enforcement now: extend `scripts/opencode-consistency-check.sh` so
   unexpected `.opencode` topology drift fails in `make check` and
   `make verify`.
4. Align canonical explanation now: update the lightweight guard-rail docs in
   `docs/operations/`.
5. Keep future changes narrow: only add new `.opencode` areas after an explicit
   mission and ownership decision tied back to the real repository.
