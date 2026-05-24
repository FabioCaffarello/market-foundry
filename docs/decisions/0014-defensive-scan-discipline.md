# ADR 0014: Defensive-scan discipline

## Status

Accepted. Formalization of practice that caught additional sites
in nearly every Phase 4 inventory-grounded fix.

## Context

Pre-fix inventories — whether produced by a read-only
investigation, a `grep` sweep, or an architect's enumeration in
the prompt — are **frequently incomplete**. Phase 4 produced
direct evidence:

| Sub-prompt | Initial inventory | After defensive scan | Catch source |
|---|---|---|---|
| P4.1.10 (`Strategy.DeduplicationKey` Unix → UnixNano) | 1 type | 4 types — `ExecutionIntent`, `Decision`, `RiskAssessment`, `Signal` siblings | sibling sweep |
| P4.1.11.a (NATS subject filter) | 4 writerpipeline sites | 9 sites — 5 in `natsexecution/restart_recovery_test.go` | post-fix grep |
| P4.2 (`rate_limiter.Close` lifecycle) | 2 production sites | 7 total — 5 test sites already `defer Close` | full-tree scan |
| P4.3.a (`context.Background()` bounding) | 14 known sites | 14 + 4 — 1 genuine `retry_submitter.isHalted` improvement, 3 `//nolint` with rationale | `contextcheck` linter sweep |
| P4.5.c.ii (ureq 2 → 3 migration) | 6 call sites | 6 confirmed accurate + 2 pre-existing pre-Phase-5 issues flagged (clippy in `parser.rs`, drift-detect tests) | post-fix scan |

Pattern: a defensive scan after the primary fix catches
**additional sites in approximately every inventory-grounded
fix**. Without the discipline, missed sites ship as latent debt
that surfaces later — often during an unrelated change, where
diagnosis costs more than the original prevention.

The "prime convention" naming (`p4.1.6.a` → `p4.1.6.a'` →
`p4.1.6.a''` → `p4.1.6.a.ii`, see ADR 0013 and
`.claude/agents/architect-agent.md`) emerged from cases where the
defensive scan revealed scope reframing was needed mid-flight.

## Decision

**After applying any fix grounded in an inventory, the executor
MUST perform a defensive scan** for additional sites or patterns
the initial inventory may have missed.

The architect **MUST require defensive scan** in fix-prompt
protocols — either via an explicit step in the Protocolo /
Execução sections or by invoking the canonical fix-prompt
template (`.claude/skills/fix-prompt-skill/SKILL.md`).

### Procedure

1. Apply the fix per the inventory.
2. Search for similar patterns / structures **beyond** the
   inventory: sibling types in the same family, adjacent files
   in the same domain, callers of the changed symbol, tests
   exercising the changed surface, pre-existing issues in the
   same area.
3. For each additional finding, decide:
   - apply the fix in the same commit (if structurally
     consistent), or
   - surface as out-of-scope (if an architectural concern, or if
     the surface is too large to absorb without reframing).
4. Document defensive-scan findings explicitly in the commit
   message ("defensive scan caught N additional sites: ...").

### Expectation calibration

| Findings | Interpretation |
|---|---|
| **0** | Either scan too narrow OR scope genuinely contained. Default to "too narrow" if confidence in the inventory was low — widen and re-scan. |
| **1-3** | Expected. Apply fix; document. |
| **4-10** | Meaningful expansion. Decide in-place: extend the current fix (bias when all are mechanical) or split into a follow-up sub-prompt (bias when any require new judgment). |
| **10+** | Pause-and-report (see ADR 0013). Pattern is probably systemic, not isolated; architect may want to reframe. |

## Consequences

### Positive

- Fix completeness improved by construction.
- Sibling-pattern detection — Phase 4.1.10's case where a
  dedup-precision fix on `Strategy` was missing in 4 sibling
  types — is structural, not happenstance.
- Linter / tooling integration opportunities surface during scan
  (P4.3.a's adoption of `contextcheck` was driven by exactly
  this).
- Out-of-scope flagging is explicit rather than silently
  deferred.

### Negative

- Per-fix overhead: each commit incurs scan time.
- False positives possible — scan finds adjacent-but-different
  patterns that look like sites but aren't.
- Tension between "extend the current fix" and "split into a
  follow-up" when 4-10 sites surface; judgment call.

### Mitigation

- The architect codifies scan scope in the prompt's Protocolo
  section (e.g., "scan also `ExecutionIntent` siblings").
- The executor exercises judgment on scope expansion and
  consults ADR 0013 (pause-and-report) when uncertain.
- The fix-prompt skill (`.claude/skills/fix-prompt-skill/SKILL.md`)
  documents what to scan, including sibling types, callers,
  tests, and pre-existing issues.

## Alternatives considered

- **Trust the inventory; don't scan**. Rejected: Phase 4
  evidence is overwhelming that inventories are incomplete.
- **Make the architect's inventory mandatory-complete**.
  Rejected: the architect cannot have the executor's depth-and-
  freshness view; this would push verification cost upstream
  without reducing it.
- **Codify in skills only, not as an ADR**. Rejected (owner
  direction, P5.5): defensive scan is a load-bearing institutional
  commitment, not a procedural preference; ADR captures the
  durability.

## References

- `.claude/agents/execution-agent.md` → "Defensive scan
  discipline" section (P5.2).
- `.claude/agents/architect-agent.md` → "Defensive scan after
  inventory" (P5.2).
- `.claude/skills/fix-prompt-skill/SKILL.md` — procedural
  knowledge.
- Phase 4 commits: `0f379ba` (P4.1.10), `b7eaa53` (P4.1.11.a),
  `d4dfab6` (P4.2), `455f02e` (P4.3.a), `68ce135` (P4.5.c.ii).
- ADR 0013 (pause-and-report — defensive scan is one of its
  canonical trigger types).
- ADR 0015 (wave-closure — accumulated 10+ scan findings can
  signal wave-depth inflection).
