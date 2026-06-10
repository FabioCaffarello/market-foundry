#!/usr/bin/env bash
# test-guards.sh — 13-scenario decision matrix for the ADR-0026
# enforcement guards (raccoon-readonly-guard, p9-branch-guard).
#
# This is the matrix that validated the guards in the harness FASE 2
# B2 session (2026-06-09), committed so the proof is reproducible:
# 11 regression scenarios + the 2 heredoc false-positive shapes
# discovered live when the hot-loaded p9 guard denied its own
# delivery commit (message quoted "git push ... main").
#
# Run:  .claude/hooks/test-guards.sh
# Exit: 0 if all 13 scenarios produce the expected decision.
set -uo pipefail
cd "$(dirname "$0")"

export RACCOON_REFERENCE_PATH="${RACCOON_REFERENCE_PATH:-/Volumes/OWC Express 1M2/Develop/market-raccoon}"
R="$RACCOON_REFERENCE_PATH"

pass=0
fail=0

check() {
  local name="$1" script="$2" expected="$3" payload="$4"
  local out verdict
  out=$(printf '%s' "$payload" | "./$script")
  if [ -z "$out" ]; then
    verdict="allow"
  else
    verdict=$(printf '%s' "$out" | python3 -c \
      "import json,sys; print(json.load(sys.stdin)['hookSpecificOutput']['permissionDecision'])" \
      2>/dev/null || echo "unparseable")
  fi
  if [ "$verdict" = "$expected" ]; then
    echo "PASS  $name => $verdict"
    pass=$((pass + 1))
  else
    echo "FAIL  $name => expected $expected, got $verdict"
    fail=$((fail + 1))
  fi
}

# --- raccoon-readonly-guard (P2) — 6 scenarios ---------------------

check "R1 deny Edit under raccoon" raccoon-readonly-guard.sh deny \
  "{\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"$R/cmd/main.go\"}}"

check "R2 allow read-only Bash on raccoon" raccoon-readonly-guard.sh allow \
  "{\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"grep -rn foo \\\"$R/internal\\\"\"}}"

check "R3 deny rm under raccoon" raccoon-readonly-guard.sh deny \
  "{\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"rm -rf \\\"$R/tmp\\\"\"}}"

check "R4 deny cp via env-var reference (P1: no copying)" raccoon-readonly-guard.sh deny \
  '{"tool_name":"Bash","tool_input":{"command":"cp $RACCOON_REFERENCE_PATH/x.go internal/"}}'

check "R5 allow Edit on foundry file" raccoon-readonly-guard.sh allow \
  '{"tool_name":"Edit","tool_input":{"file_path":"/Volumes/OWC Express 1M2/Develop/market-foundry/CLAUDE.md"}}'

# FP shape 2 (live incident class): heredoc commit message MENTIONING
# the raccoon + write tokens in prose must not trip the guard.
check "R6 allow heredoc prose mentioning raccoon+rm (FP shape)" raccoon-readonly-guard.sh allow \
  '{"tool_name":"Bash","tool_input":{"command":"git add x && git commit -m \"$(cat <<'"'"'EOF'"'"'\ndocs: raccoon guard\n\nmentions market-raccoon and rm/cp write tokens in prose\nEOF\n)\""}}'

# --- p9-branch-guard (P9) — 7 scenarios ----------------------------

check "P1 deny git push origin main" p9-branch-guard.sh deny \
  '{"tool_name":"Bash","tool_input":{"command":"git push origin main"}}'

check "P2 deny git push HEAD:main refspec" p9-branch-guard.sh deny \
  '{"tool_name":"Bash","tool_input":{"command":"git push origin HEAD:main"}}'

check "P3 allow git push to feature branch" p9-branch-guard.sh allow \
  '{"tool_name":"Bash","tool_input":{"command":"git push -u origin feat/h-6-e-subjects"}}'

check "P4 deny git commit --no-verify" p9-branch-guard.sh deny \
  '{"tool_name":"Bash","tool_input":{"command":"git commit --no-verify -m x"}}'

check "P5 deny LEFTHOOK=0 bypass" p9-branch-guard.sh deny \
  '{"tool_name":"Bash","tool_input":{"command":"LEFTHOOK=0 git commit -m x"}}'

check "P6 ask on gh pr merge" p9-branch-guard.sh ask \
  '{"tool_name":"Bash","tool_input":{"command":"gh pr merge 37 --squash"}}'

# FP shape 1 (the live incident itself): heredoc commit message
# QUOTING push-main / --no-verify / LEFTHOOK=0 / gh pr merge in
# prose must not trip the guard — only real invocations do.
check "P7 allow heredoc prose quoting guarded patterns (FP shape)" p9-branch-guard.sh allow \
  '{"tool_name":"Bash","tool_input":{"command":"git add x && git commit -m \"$(cat <<'"'"'EOF'"'"'\nfeat: guard\n\n- deny git push targeting main\n- deny --no-verify / LEFTHOOK=0 bypass\n- ask on gh pr merge\nEOF\n)\""}}'

# -------------------------------------------------------------------
echo ""
echo "matrix: $pass/13 PASS, $fail FAIL"
[ "$fail" -eq 0 ] && [ "$pass" -eq 13 ]
