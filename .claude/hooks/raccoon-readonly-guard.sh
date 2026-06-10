#!/usr/bin/env bash
# raccoon-readonly-guard.sh — PreToolUse hook enforcing P2 (raccoon
# read-only). See docs/decisions/0026-claude-code-hooks-enforcement.md.
#
# Denies:
#   - Write / Edit / NotebookEdit targeting $RACCOON_REFERENCE_PATH
#     (defense-in-depth alongside settings.json permissions.deny).
#   - Bash commands that mention the raccoon checkout together with a
#     write-capable token (rm, mv, cp, tee, sed -i, redirection, ...).
#     Note cp IS denied: P1 forbids copying raccoon files into the
#     foundry — capabilities are rewritten, never copied.
#
# Read-only access (grep, cat, less, diff ...) passes through.
set -uo pipefail

RACCOON_PATH="${RACCOON_REFERENCE_PATH:-/Volumes/OWC Express 1M2/Develop/market-raccoon}"
HOOK_INPUT="$(cat)"
export HOOK_INPUT

python3 - "$RACCOON_PATH" <<'PY'
import json
import os
import re
import sys

raccoon = sys.argv[1]
try:
    data = json.loads(os.environ.get("HOOK_INPUT", "{}"))
except Exception:
    sys.exit(0)  # malformed input: do not block, permission system still applies

tool = data.get("tool_name", "")
tool_input = data.get("tool_input", {}) or {}


def deny(reason):
    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": reason,
        }
    }))
    sys.exit(0)


if tool in ("Write", "Edit", "NotebookEdit"):
    path = tool_input.get("file_path", "") or tool_input.get("notebook_path", "")
    if path.startswith(raccoon):
        deny(
            "P2: o market-raccoon e estritamente read-only "
            f"({raccoon}). Nenhum arquivo do raccoon pode ser "
            "modificado (CLAUDE.md - Fase Harvest, ADR-0016, ADR-0026)."
        )

elif tool == "Bash":
    cmd = tool_input.get("command", "")
    # Strip heredoc bodies and single-quoted strings so that prose
    # MENTIONING the raccoon (commit messages, docs being committed)
    # does not trip the guard — only actual command segments do.
    # Double-quoted strings are kept: the raccoon path contains a
    # space and is normally double-quoted in real invocations.
    sanitized = re.sub(r"<<-?\s*'?(\w+)'?\s*\n.*?\n\1\b", " ", cmd, flags=re.S)
    sanitized = re.sub(r"'[^']*'", "''", sanitized)
    write_tokens = re.compile(
        r"(\b(rm|mv|cp|touch|mkdir|rmdir|ln|chmod|chown|truncate|dd|tee)\b"
        r"|\bsed\b[^|;&\n]*\s-i"
        r"|>>?"
        r"|\bgit\b[^|;&\n]*\b(commit|push|checkout|reset|clean|stash|rebase|merge|rm|mv)\b)"
    )
    for seg in re.split(r"[|;&\n]", sanitized):
        mentions_raccoon = (
            raccoon in seg
            or "market-raccoon" in seg
            or "RACCOON_REFERENCE_PATH" in seg
        )
        if mentions_raccoon and write_tokens.search(seg):
            deny(
                "P2: comando Bash com token de escrita referenciando o "
                "market-raccoon (read-only). Leitura (grep/cat/diff) e "
                "permitida; escrita e copia nao (P1: nada do raccoon e "
                "copiado). Ver ADR-0026."
            )

sys.exit(0)
PY
