# ADR 0015: Wave-closure discipline

## Status

Accepted. Formalization of the closure decision made at the end
of the Phase 4.1 wave.

## Context

market-foundry work proceeds in **waves** — clusters of related
sub-prompts that share a common objective. A wave starts with a
clear goal (restore CI, close a P0 backlog item, refresh
documentation) and ends with a closure decision.

The Phase 4.1 wave is the canonical case study. It started as
"restore CI to green after the P3.3 SHA-pinning migration" and
reached **18 ship-commits** (plus ~9 read-only investigations)
before closure at `P4.1.11.b`. Each fix revealed the next layer:

1. SHA-pinning the workflow actions → revealed
   `golangci-lint-action` v6→v9 argument breakage.
2. Fixing the action args → revealed 11 latent
   quality-gate-ci warnings now promoted to errors.
3. Fixing those → revealed the drift-detect const-table
   mismatch (G6.2).
4. Aligning the table → revealed the `cmd → domain` rule
   over-shoot.
5. Refining the rule → revealed the deploy-boundary
   `_test.go` exemption gap.
6. Fixing the exemption → revealed the NATS `services:`
   `command:` schema rejection.
7. Switching to `docker run` → revealed the network
   namespace issue.
8. Adding `--network host` → revealed the NATS `-m` flag
   need for the JetStream HTTP monitor.
9. Adding the flag → revealed Smoke Analytical E2E's
   external-data dependency.
10. Deferring Smoke Analytical → revealed counter-ordering
    flakes in writerpipeline tests.

At each layer the architect's natural instinct was "this is
the last one". By layer 5-6 the prediction track record was
clearly poor; by layer 8-9 the executor was explicitly
surfacing diminishing-returns signals.

The closure decision at `P4.1.11.b` deferred remaining debt to
the M-list (M1-M12 at that point) and declared the wave closed.
This unblocked Phase 4.2-4.5 (`rate_limiter`, context bounding,
ControlGate ADR, Dependabot triage) which closed the entire
Phase 4 P0 backlog.

Without explicit wave-closure discipline, the wave would have
extended indefinitely — each fix revealing the next — and Phase
4 P0 work would have remained untouched.

## Decision

The architect **MUST recognize wave-closure signals** and act on
them. The owner can prompt closure when the architect's signal
detection lags.

### Closure signals

1. **Original objective delivered**: the wave's stated goal is
   met, even if surrounding debt remains.
2. **"Each fix reveals next layer" persistence ≥5 iterations**:
   the wave has crossed into systematic debt territory, not
   isolated bugs.
3. **Wave size ≥5× original scope estimate**: a prompt that
   started with "fix 2 things" is now 10+ sub-prompts.
4. **Architect's "last layer" prediction accuracy < ~30%**: the
   architect is not in a position to estimate the remaining
   depth.
5. **Remaining debt fits documented-debt format**: the next
   layer can be captured as an M-list entry without losing
   substantive context.

### Procedure

When closure signals appear, the architect:

1. **Captures remaining debt** as design-meta (M-list candidates
   in `docs/RESUMPTION.md`).
2. **Declares wave closure** in the closing sub-prompt's commit
   narrative.
3. **Moves to the next planned phase or sub-wave**.

The owner provides direction on closure timing when the
architect's signal is ambiguous.

### Anti-pattern

Continuing to extend a wave without recognizing closure signals
is **not investigation discipline** — it is avoidance of the
harder closure decision. Phase 4.1.6.a's prime-suffix sequence
(`a → a' → a'' → a.ii`) is a within-sub-prompt recovery; it
does not extend wave length. By contrast, a 27th sub-prompt at
the end of a 10-layer wave is wave extension; closure is the
right move.

## Consequences

### Positive

- Wave delivery has bounded scope; downstream phases stay on
  the schedule the owner planned.
- Documented debt > silent debt > indefinite extension. M-list
  entries preserve substantive context for future work.
- The Phase 4.1 closure unblocked Phase 4.2-4.5 (full P0
  backlog) and Phase 5 (this work).

### Negative

- Some legitimate debt is deferred to design-meta rather than
  fixed in-flight.
- Closure can feel premature if the next layer appears "small".
- Requires the architect to overcome the sunk-cost reflex of
  "we're so close".

### Mitigation

- The M-list (`docs/RESUMPTION.md` → "design-meta candidates")
  provides explicit capture; no silent loss.
- The closing commit narrative documents the closure rationale
  so the deferral is traceable.
- The owner can reopen the wave for substantive concerns
  (M-list items are not closure decisions about the underlying
  debt — they are deferrals).
- The RESUMPTION-drift hook (P5.3,
  `scripts/check-resumption-drift.sh`) surfaces drift when
  newly-introduced M-N references aren't backfilled to the
  M-list; this reduces the cost of the capture step.

## Alternatives considered

- **Always finish the current layer before declaring closure**.
  Rejected: each layer's "current" is poorly defined when the
  next layer is revealed by the current fix; a strict rule would
  extend the wave indefinitely.
- **Time-cap waves (e.g., max N sub-prompts)**. Rejected:
  arbitrary numeric caps don't fit organic wave shapes (Phase
  4.2 was 2 commits; Phase 4.1 was 18 ship-commits). Signal-
  based recognition is more robust.
- **Defer all extension work, even when trivial**. Rejected:
  some small extensions are genuinely the right shape for the
  current wave (e.g., a one-line follow-up to a defensive-scan
  finding). Closure discipline is a recognition skill, not a
  blanket prohibition on continuation.

## References

- `.claude/agents/architect-agent.md` → "Wave-depth
  recognition" section (P5.2).
- `docs/RESUMPTION.md` → "Phase 4 design-meta candidates"
  section — the M-list (M1-M20, M19 closed) that received the
  Phase 4.1 closure debt.
- `scripts/check-resumption-drift.sh` (P5.3) — automated drift
  surfacing for new M-N references.
- Phase 4.1 closure commit: `b7eaa53d` (P4.1.11.a) — the
  closure narrative.
- ADR 0013 (pause-and-report — a wave-depth signal is a
  canonical trigger type).
- ADR 0014 (defensive-scan — accumulated scan findings can
  reveal wave-depth inflection).
