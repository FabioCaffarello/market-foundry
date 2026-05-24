#!/usr/bin/env bash
#
# scripts/check-resumption-drift.sh
#
# Post-commit hook: warn (stderr) if a commit's message references
# new M-N design-meta identifiers but docs/RESUMPTION.md was not
# modified in the same commit. Codifies P5.0 audit finding F4.
#
# Background (F4): the P4.3.a commit body listed M13, M14, M15 as
# newly surfaced design-meta candidates. RESUMPTION was not touched
# in that commit. Backfill happened in P4.5.a (2 sub-prompt latency).
# This hook surfaces such drift at commit time so the architect or
# executor can amend or follow up immediately.
#
# Detection (heuristic, conservative; warn-only):
#   1. Extract M-N tokens (\bM[0-9]+\b) from the commit message body.
#   2. For each M-N, check whether `#### M<N>` exists in
#      docs/RESUMPTION.md AS OF the commit being checked
#      (via `git show <sha>:docs/RESUMPTION.md`).
#   3. M-N not found in that snapshot → likely an introduction.
#   4. If any introductions exist AND RESUMPTION was not changed
#      in the commit → warn.
#
# Exits 0 in all cases (warn-only, non-blocking).
#
# Usage:
#   scripts/check-resumption-drift.sh             # check HEAD
#   scripts/check-resumption-drift.sh <commit>    # check a specific
#                                                 # commit (testing)
#
# Adopted: P5.3, 2026-05-24.

set -uo pipefail

SHA_ARG="${1:-HEAD}"
COMMIT_SHA=$(git rev-parse "$SHA_ARG" 2>/dev/null || true)

if [ -z "$COMMIT_SHA" ]; then
    # No commit yet (e.g., empty repo on first install); nothing to do.
    exit 0
fi

COMMIT_MSG=$(git log -1 --format=%B "$COMMIT_SHA")

# Extract M-N references. \bM[0-9]+\b matches M1..M99999. -o emits
# each match on its own line. sort -u dedupes case-sensitively.
M_REFS=$(printf '%s\n' "$COMMIT_MSG" | grep -oE '\bM[0-9]+\b' | sort -uV || true)

if [ -z "$M_REFS" ]; then
    # No M-N references; nothing to check.
    exit 0
fi

# Did this commit modify docs/RESUMPTION.md?
RESUMPTION_MODIFIED=false
CHANGED_FILES=$(git show --name-only --format='' "$COMMIT_SHA")
if grep -qx 'docs/RESUMPTION.md' <<< "$CHANGED_FILES"; then
    RESUMPTION_MODIFIED=true
fi

# Fetch RESUMPTION as it stood at this commit (post-commit state).
# If the commit modified RESUMPTION, this reflects the post-modification
# version (so legitimate introductions land here). If the commit didn't
# modify RESUMPTION, this equals the parent's version.
RESUMPTION_AT_COMMIT=$(git show "${COMMIT_SHA}:docs/RESUMPTION.md" 2>/dev/null || true)

if [ -z "$RESUMPTION_AT_COMMIT" ]; then
    # File doesn't exist at this commit (e.g., commit predates
    # RESUMPTION). Nothing to compare against.
    exit 0
fi

# Find M-N refs that don't appear as a `#### M<N>` section in the
# commit-time RESUMPTION. The trailing `\b` excludes false matches
# like M1 finding M10/M11/M12.
#
# A here-string (<<<) is used rather than `printf | grep -q` because
# `grep -q` exits early on first match, closing the pipe; with
# `pipefail` set, the upstream `printf` gets SIGPIPE and the
# pipeline reports failure even when grep actually matched.
LIKELY_INTRODUCTIONS=()
for M_REF in $M_REFS; do
    if ! grep -qE "^#{2,4} ${M_REF}\b" <<< "$RESUMPTION_AT_COMMIT"; then
        LIKELY_INTRODUCTIONS+=("$M_REF")
    fi
done

if [ "${#LIKELY_INTRODUCTIONS[@]}" -eq 0 ]; then
    # All M-N references are pre-existing in RESUMPTION (just
    # mentions, not introductions); no drift.
    exit 0
fi

if [ "$RESUMPTION_MODIFIED" = "true" ]; then
    # RESUMPTION was modified in this commit; if the introductions
    # haven't landed as sections yet that's a more subtle drift, but
    # erring on the side of fewer false positives — assume the
    # author is mid-flight and chose this shape deliberately.
    exit 0
fi

# Drift detected: commit introduces new M-N references but
# RESUMPTION.md is untouched.
INTROS_FORMATTED=$(printf '%s ' "${LIKELY_INTRODUCTIONS[@]}")

cat <<EOF >&2

⚠ RESUMPTION-drift warning (P5.3 / F4):

This commit (${COMMIT_SHA:0:7}) references the following M-N
design-meta identifier(s) that don't yet have a section in
docs/RESUMPTION.md:

  ${INTROS_FORMATTED}

If these are new candidates, consider:
  - amending this commit to add the #### M-N section(s), OR
  - producing a follow-up docs commit that backfills RESUMPTION.

If these are typos or references to closed items, this warning
can be ignored.

(Codifies P5.0 audit finding F4: M13/M14/M15 surfaced in the
P4.3.a commit body but persisted to RESUMPTION only at P4.5.a —
2 sub-prompt latency. This hook surfaces such drift at commit
time. Warn-only; non-blocking.)

EOF

# Warn-only; exit 0 to allow workflow to proceed.
exit 0
