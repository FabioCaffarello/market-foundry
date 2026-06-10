#!/usr/bin/env bash
# p9-branch-guard.sh — PreToolUse (Bash) hook enforcing P9 locally.
# See docs/decisions/0026-claude-code-hooks-enforcement.md.
#
# Posture (owner decision, 2026-06-09 — "misto"):
#   deny — git push targeting main (any refspec form);
#   deny — --no-verify / LEFTHOOK=0 bypass of git hooks;
#   ask  — gh pr merge (human may authorize an agent-driven merge
#          interactively; default remains maintainer-merges).
#
# Heredoc bodies and quoted strings are stripped before matching so
# that commit messages MENTIONING these patterns (e.g. docs about
# this very guard) do not trip it — only actual invocations do.
# Known accepted false-negative: quoting the branch name evades the
# match; remote branch protection is the backstop (defense-in-depth).
#
# These are local mirrors of remote branch protection: the harness
# must not teach (or silently attempt) what the remote gate blocks.
# Emergency escape hatch: the human runs the command directly with
# the `!` prefix in the prompt — hooks only gate the agent's tools.
set -uo pipefail

HOOK_INPUT="$(cat)"
export HOOK_INPUT

python3 - <<'PY'
import json
import os
import re
import sys

try:
    data = json.loads(os.environ.get("HOOK_INPUT", "{}"))
except Exception:
    sys.exit(0)

if data.get("tool_name") != "Bash":
    sys.exit(0)

cmd = (data.get("tool_input", {}) or {}).get("command", "")

# Strip heredoc bodies and quoted strings: match invocations, not prose.
sanitized = re.sub(r"<<-?\s*'?(\w+)'?\s*\n.*?\n\1\b", " ", cmd, flags=re.S)
sanitized = re.sub(r"'[^']*'", "''", sanitized)
sanitized = re.sub(r'"[^"]*"', '""', sanitized)
segments = re.split(r"[|;&\n]", sanitized)


def decide(decision, reason):
    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": decision,
            "permissionDecisionReason": reason,
        }
    }))
    sys.exit(0)


for seg in segments:
    is_git_push = re.search(r"\bgit\b.*\bpush\b", seg)
    is_git_commit = re.search(r"\bgit\b.*\bcommit\b", seg)

    # P9: agents never push to main — delivery is feature branch + PR,
    # merge by the human maintainer.
    if is_git_push and re.search(r"(\bmain\b|HEAD:main)", seg):
        decide(
            "deny",
            "P9: agentes nao fazem push em main. Entregue via branch "
            "dedicada + PR; o merge em main e do maintainer humano "
            "(CLAUDE.md - Fase Harvest - P9, ADR-0026).",
        )

    # P9 trava operacional 3: CI gates sao pre-requisito — bypass de
    # hooks locais e proibido para agentes.
    if (is_git_push or is_git_commit) and re.search(r"--no-verify\b", seg):
        decide(
            "deny",
            "P9: bypass de hooks (--no-verify) nao autorizado para "
            "agentes (execution-agent anti-patterns, ADR-0026).",
        )
    if re.search(r"\bLEFTHOOK=0\b", seg):
        decide(
            "deny",
            "P9: bypass de hooks (LEFTHOOK=0) nao autorizado para "
            "agentes (ADR-0026).",
        )

    # Merge de PR: ask — preserva a agencia do owner na hora.
    if re.search(r"\bgh\s+pr\s+merge\b", seg):
        decide(
            "ask",
            "P9: merge de PR e do maintainer humano por padrao. "
            "Confirme para autorizar este merge via agente (ADR-0026).",
        )

sys.exit(0)
PY
