#!/usr/bin/env bash
# session-start-reminder.sh — SessionStart hook. Emits a compact
# orientation line so every session starts anchored on the state
# sentinel and the two mechanically-enforced invariants.
# See docs/decisions/0026-claude-code-hooks-enforcement.md.
set -uo pipefail

cat <<'EOF'
[market-foundry] Antes de qualquer trabalho: leia docs/RESUMPTION.md
-> "Fase Harvest" (onda em voo + programa ativo; CLAUDE.md reading
order itens 2-3). Invariantes com enforcement mecanico nesta sessao:
P2 (RACCOON_REFERENCE_PATH e read-only) e P9 (sem push em main, sem
--no-verify; merge de PR e do maintainer). Ver ADR-0026.
EOF
