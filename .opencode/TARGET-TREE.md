# `.opencode` Target Tree

This is the approved minimal topology for `market-foundry`.

It exists to keep `.opencode` native to the repository and bounded to four
concerns: `repo`, `runtime`, `change`, and `intelligence`.

## Approved Tree

```text
.opencode/
  README.md
  TARGET-TREE.md
  O12-report.md
  O13-report.md
  O14-report.md
  O15-report.md
  O16-report.md
  O17-report.md
  O19-report.md
  O20-report.md
  config.json
  opencode.json
  agent/
    core/
      foundry-agent.md
  profiles/
    essential/
      profile.json
    developer/
      profile.json
  context/
    navigation.md
    repo/
      navigation.md
      repository-shape.md
      development-workflow.md
      architecture-boundaries.md
      tooling-contracts.md
      documentation-topology.md
    runtime/
      navigation.md
      services-topology.md
      configs-compose-streams.md
      smoke-and-live-flows.md
      troubleshooting-paths.md
    change/
      navigation.md
      impact-analysis.md
      tdd-and-validation.md
      stage-execution.md
      safe-change-rules.md
    intelligence/
      navigation.md
      raccoon-cli-usage.md
      make-target-map.md
      repo-guardrails.md
      code-intelligence-paths.md
```

## Ownership Rules

- `Makefile` remains the public command surface. `.opencode` does not grow a
  parallel command catalog.
- `AGENTS.md`, `README.md`, `DEVELOPMENT.md`, `docs/development/`,
  `docs/product/`, `docs/tooling/`, and `docs/architecture/` remain canonical
  owners.
- `raccoon-cli` remains the real intelligence layer. `.opencode` only routes to
  it and compresses the handoff context around it.
- The agent surface stays minimal: one `foundry-agent` only. Do not add durable
  subagent taxonomies here without a new owner and mission decision.
- Every context area keeps explicit owner-doc anchors and a single navigation
  file that routes to all approved local leaves.
- O-reports remain reports. They may explain why the layer changed, but they do
  not become a second active index, owner map, or policy catalog.
- `.opencode` may absorb only navigation, compression, entrypoint choice,
  operational short context, session support, and safe-change guidance.

## Not Part Of The Target Tree

- generic skills, plugins, marketplaces, or multi-domain taxonomies
- task registries, workflow engines, or command wrappers parallel to `make`
- duplicated architecture or stage-governance catalogs already owned by the
  canonical docs
- opportunistic extra context files outside the approved minimal set unless the
  target tree and the owner docs change together for a real recurring need
