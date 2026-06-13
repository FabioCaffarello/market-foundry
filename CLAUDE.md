# CLAUDE.md

Operating instructions for Claude (and other AI agents) working on
market-foundry.

This file is read automatically by Claude when starting a session
in this repository. It is intentionally concise — for depth, follow
the references.

---

## What this repository is

market-foundry is a Go workspace foundation for cryptocurrency market
data processing. It is **not** a trading application — it is the
foundation on which trading capabilities are built.

Seven long-running binaries (configctl, gateway, ingest, derive, store,
execute, writer) plus one one-shot tool (migrate), communicating via
NATS+JetStream. ClickHouse for analytical storage. Rust raccoon-cli
for static architecture enforcement.

For higher-level orientation, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Reading order for any new session

When starting work on this repository, read in this order:

1. **The prompt or task you received** — your immediate context.
2. **[docs/RESUMPTION.md](docs/RESUMPTION.md)** — current state, known
   gaps, next concrete step. Always start here.
3. **The active PROGRAM** if one is in flight — see
   [docs/programs/](docs/programs/README.md) for the convention
   and its Index for current statuses. The wave in flight and its
   program are named in [docs/RESUMPTION.md](docs/RESUMPTION.md) →
   "Fase Harvest" — follow that pointer rather than assuming a
   hardcoded program here.
4. **[docs/AUTHORITY.md](docs/AUTHORITY.md)** — document hierarchy
   T1–T4. Tells you which doc to trust when two disagree.
5. **[docs/TRUTH-MAP.md](docs/TRUTH-MAP.md)** — capability × ADR ×
   code anchor × test anchor cross-reference. Use this to find
   where any claim is grounded in code.
6. **[docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)** — operational rules,
   PR workflow, "Specifically for AI agents" section.
7. **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** — system shape and
   structural principles.
8. **"Fase Harvest" section below** if your task touches the current
   harvest from `market-raccoon`. The nine principles P1–P9 are the
   permanent protocol for that work.
9. **Specific docs your task needs** (e.g., domain docs, HTTP-API,
   runtime, operations).

Time investment: 5-10 minutes for documents 2-3 minimum, on every
session. Skipping this leads to misaligned work.

---

## Core operating protocols

These protocols are non-negotiable. They emerged from real lessons
during Phases 0 and 1A.

### 1. Validate against code before claiming facts

Documentation can be stale. Code is the source of truth. Before
asserting a technical fact in a doc or commit message, **verify it
against the codebase** with a concrete grep, find, or read.

This rule emerged because multiple prompts during Phase 1A produced
draft content with factual divergences from code (stream counts,
consumer ownership, plane taxonomy, type lists). The "verify before
save" pattern caught them.

### 2. Pause and report on divergence

If you encounter:
- A blocker or improvement **outside the prompt's scope**,
- A **discrepancy** between expected and actual state,
- An **ambiguity** in the task that needs clarification,

**stop, report concisely, and present options (A/B/C/D).** Wait for
direction before proceeding.

This is the "authorized expansion protocol" — see
[docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) → "Authorized expansion
protocol" for the canonical 5-step procedure with examples.

Silent expansion is forbidden. Silent skipping is forbidden.

### 3. Honesty over convenience

When a failure has a convenient categorization, **investigate more, not
less**. Convenient categorization is exactly when verification is most
likely to lapse.

Concrete example: during Phase 1A, `make verify` failures were
attributed to ".opencode/ cross-refs" for 18 prompts. P1B investigation
revealed the framing was triply wrong (wrong count, wrong attribution,
missed an entire failing layer in `tools/raccoon-cli/`). The convenient
narrative cost real work.

If a report contains "and similar" or "and related" or any hedge
phrase, investigate the hedge before adopting the categorization.

### 4. Single-writer invariant

Every JetStream stream, every NATS KV bucket, every NATS query subject
has **exactly one writer**. No exceptions. This is the most important
invariant in the system; preventing it by construction is much cheaper
than debugging the race conditions it would otherwise allow.

See [docs/decisions/0008-single-writer-invariant.md](docs/decisions/0008-single-writer-invariant.md).

### 5. Adding an HTTP route requires updating boot_test.go

If your change adds a route to `internal/interfaces/http/routes/`, you
**must also** add it to `cmd/gateway/boot_test.go`'s `routes` slice.

The boot test exists as a regression guard for httprouter trie
conflicts (lesson from Phase 0 where 3 simultaneous conflicts caused
gateway CrashLoopBackoff). CI will fail your PR if you forget.

See [docs/decisions/0010-httprouter-trie-constraints.md](docs/decisions/0010-httprouter-trie-constraints.md).

### 6. Layer sovereignty is enforced

Imports flow inward only:
`domain → application → adapters → actors → interfaces → cmd`.

raccoon-cli enforces this in `make verify`. A violating import does
not ship.

See [docs/decisions/0005-layer-sovereignty.md](docs/decisions/0005-layer-sovereignty.md).

---

## Fase Harvest

market-foundry is in the **Harvest phase**: a structured, multi-onda
program that consults the sibling `market-raccoon` repository
(`$RACCOON_REFERENCE_PATH` =
`/Volumes/OWC Express 1M2/Develop/market-raccoon`, set in
`.claude/settings.json`) as a **read-only reference** while
re-implementing selected capabilities natively inside the foundry.

Umbrella program:
[PROGRAM-0001 — Harvest Foundation](docs/programs/PROGRAM-0001-foundation.md).
The wave in flight and the program it belongs to are named in
[docs/RESUMPTION.md](docs/RESUMPTION.md) → "Fase Harvest" (waves
beyond H-2 live in later programs — see the
[programs Index](docs/programs/README.md)).
Decision of record:
[ADR-0016 — Harvest from market-raccoon](docs/decisions/0016-harvest-from-market-raccoon.md).
Convention for PRDs:
[docs/programs/README.md](docs/programs/README.md).

### Princípios P1–P9

These nine principles are the **canonical, permanent protocol** for
the Harvest. They are not specific to any single onda; they govern
every onda of the program. Treat them with the same weight as the
"Core operating protocols" section above.

#### P1 — Foundry é ground truth; raccoon é referência consultiva

A estrutura atual do `market-foundry` (8 binários, ADRs existentes,
layer sovereignty enforced, RESUMPTION.md como state-truth) é o
ponto de partida. O `market-raccoon` é lido para informar decisões,
nunca para servir de base de migração. **Nenhum arquivo do raccoon
é copiado; capacidades são reescritas dentro do foundry respeitando
layer sovereignty.**

#### P2 — Raccoon path é estritamente read-only

`$RACCOON_REFERENCE_PATH` pode ser lido livremente, nunca
modificado. **Cada leitura é justificada**: declare antes de abrir
qualquer arquivo do raccoon qual decisão específica essa leitura
informa. Sem essa fricção, "browsing" vira "copying" silenciosamente.

#### P3 — Toda capacidade portada passa por documento primeiro

Sequência obrigatória, sem exceção:

```
leitura (raccoon + foundry, read-only)
  → documento (PRD da Fase, ou ADR de decisão)
  → implementação (código novo no foundry)
  → analyzer raccoon-cli (quando aplicável)
  → gate (make verify GREEN + RESUMPTION atualizado no commit)
```

Nenhum passo é pulável. A sequência é a disciplina.

#### P4 — Uma onda por vez, fechamento explícito antes da próxima

Mesmo com ritmo de revisão alto, ondas **não paralelizam**. A
próxima onda só abre quando a anterior fecha com todas as entregas
completas, `make verify` GREEN, RESUMPTION atualizado, e aprovação
explícita do maintainer. Custo de fechamento serializado é baixo;
custo de conflito documental é alto.

#### P5 — Cada onda evolui raccoon-cli quando adiciona invariante

Se a onda adiciona uma invariante arquitetural (ex.: "todo evento
tem envelope versionado"), a onda **também entrega um analyzer
`raccoon-cli`** que valida estaticamente essa invariante. Sem
isso, a invariante vira intenção sem enforcement — exatamente o
modo de falha que
[ADR-0004](docs/decisions/0004-raccoon-cli-static-enforcement.md)
existe para prevenir.

#### P6 — Pause-and-report ativo durante a onda inteira

Se encontrar blocker fora do escopo da onda atual, discrepância
entre doc e código, capacidade do raccoon que parece relevante mas
não está nesta onda, ou ambiguidade sobre como mapear conceito do
raccoon para layer do foundry — **pare, reporte concisamente,
apresente opções A/B/C/D, espere direção**. Não decida
silenciosamente. Especialização da
[ADR-0013](docs/decisions/0013-pause-and-report-protocol.md) para
a superfície do Harvest.

#### P7 — Sem perda de disciplina documental

PRDs e ADRs criados nas ondas referenciam-se entre si. RESUMPTION é
o sentinel — sempre reflete o estado real. **TRUTH-MAP (instalado
em H-1) cruzará claim × ADR/PRD × code anchor × test anchor.**
Nenhum documento criado pode declarar capacidade que o código ainda
não entregou. Status `Draft` é tolerado para PRDs em formação;
status `Accepted` em ADR exige código entregue — **exceto ADRs de
fundação H-2**, que aceitam decisões antes do código que as
implementa, **desde que o ADR liste critérios explícitos de quando
promover**.

#### P8 — Cliente Odin está mapeado, não esquecido

O cliente Odin/WASM entra no programa Harvest como **Onda H-12+**,
dentro de `client/` no próprio foundry. Até lá, **nada de cliente é
antecipado**. A Onda H-11 (Delivery WS) é desenhada considerando
que o consumidor canônico será o cliente Odin do mesmo repo, mas
sem código de cliente. Os cinco binários extras do raccoon
(`portfolio`, `strategist`, `signals-separado`, `executor`,
`validator`) **não retornam** — capacidades equivalentes são
absorvidas pelos 8 owners existentes (ver
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) → "Binary boundaries").

#### P9 — Toda alteração ao foundry passa por Pull Request

Cada onda entrega via branch dedicada (`feat/h-N-<slug>`); merge em
`main` é feito pelo **maintainer humano**, não por agentes. Agentes
(Claude Code ou outros) **não fazem self-merge nem push direto em
branches protegidas**. A próxima onda só abre após **merge** da
anterior em `main`, não apenas após "completion" local — isto
estende P4 (uma onda por vez) com requisito de incorporação real.

Travas operacionais que sustentam P9:

1. **PR humano com revisão substantiva** — maintainer lê diff antes
   de mergear (não apenas aprova checks verdes).
2. **Branch protection** em `main` — bloqueia push direto e merge
   sem PR.
3. **CI gates** — `make verify` + lefthook hooks GREEN são pré-
   requisito para merge.
4. **Pause-and-report do agente (P6)** — primeira camada; agente
   reporta ao maintainer antes de fazer algo controverso, em vez
   de descobrir no PR review.

> **Errata 2026-06-13 (delegação escopada):** o owner autorizou
> explicitamente o agente a fazer **self-merge** dos PRs do **loop
> autônomo da PROGRAM-0005** (Fase Insights: H-8.a.1 → H-8.b →
> diante). É um override **escopado** de P9 para esse loop, não uma
> revogação. O hook `p9-branch-guard.sh` **continua pedindo** (ask,
> não deny) em `gh pr merge` — o agente *tenta* o squash-merge e o
> owner *aprova o ask*; `git push origin main` e bypass
> (`--no-verify`/`LEFTHOOK=0`) seguem **negados**. Disciplina de merge
> do agente: checks verdes → **diff self-audit** → `gh pr merge
> --squash` → sync main → próxima onda. Registro completo em
> [ADR-0026](docs/decisions/0026-claude-code-hooks-enforcement.md) →
> "Errata".

---

## Essential commands

For complete daily workflow, see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
The basics:

```bash
make bootstrap           # validate prerequisites (always works)
make check               # pre-code guard rail
make tdd                 # impact-driven validation guidance
make verify              # post-change validation (green since P1D.4)
make smoke               # canonical end-to-end proof
make up                  # bring up stack
make down                # tear down stack
```

### State of make verify

`make verify` is **green** end-to-end. The historical G6 issue
(`tools/raccoon-cli/src/analyzers/drift_detect.rs` hardcoded against
the pre-reset documentation topology) was resolved in P1D.4
(commit 557a508), which realigned the const tables to the Phase 1A
topology while preserving the 27 other working drift checks. See
[docs/RESUMPTION.md](docs/RESUMPTION.md) → "Recently resolved" for
the historical detail.

A red `make verify` going forward indicates a real regression, not
historical debt. Investigate before merging.

---

## What this repository is NOT

This list is as important as what it is. Avoid assuming features
that don't exist. From [docs/RESUMPTION.md](docs/RESUMPTION.md):

- **No backtesting harness.** Strategies test in paper mode against
  live data.
- **No PnL aggregation per strategy.** Effectiveness classifies
  individual round-trips only.
- **No portfolio-level position sizing.** Decisions are local per
  symbol.
- **No multi-exchange EXECUTION surface.** Execution (paper/testnet/
  mainnet order flow, segment router, order lifecycle) is a single
  venue family: Binance Spot + Futures. The **observation plane is
  multi-venue since H-7.b** (Binance + Bybit per ADR-0022) — but
  execution adapters stay Binance-only.
- **No market-making primitives.**
- **No machine learning pipeline.**
- **No HTTP authentication.** Loopback binding is the access control.
- **No raccoon-style auxiliary binaries.** `portfolio`, `strategist`,
  `signals-separado`, `executor`, `validator` exist in
  `market-raccoon` but are **permanent non-scope** here. Capabilities
  with equivalent intent are absorbed by the existing 8 owners (see
  [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) → "Binary boundaries");
  there will be no new long-running binaries introduced to mirror the
  raccoon's `cmd/` shape.

If asked to use any of these, clarify with the user before proceeding.

---

## Boundaries by default

Unless a prompt explicitly authorizes otherwise, do not modify:

- `internal/`, `cmd/`, `tools/`, `deploy/` — code, configs, tooling.
- `docs/` files outside the scope your prompt declares —
  documentation changes ride the wave or prompt that owns them.
- `Makefile`, `.gitignore`, `go.work`, `go.mod` — repository
  infrastructure.

If your task requires modifying these, expect the prompt to call it
out. If it doesn't, ask before touching.

---

## When in doubt

Pause and ask. The cost of one extra clarification turn is much less
than the cost of an incorrect autonomous decision that requires
unwinding. Multiple prompts during Phase 1A confirmed this empirically.

For the canonical pause-and-report procedure, see
[docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) → "Authorized expansion
protocol".

---

## Reading further (canonical map)

| If you want | Go to |
|---|---|
| Current state and gaps | [docs/RESUMPTION.md](docs/RESUMPTION.md) |
| Active program (PRD) | [docs/programs/](docs/programs/README.md) |
| Document hierarchy (T1–T4) and precedence | [docs/AUTHORITY.md](docs/AUTHORITY.md) |
| Capability × code-anchor × test-anchor map | [docs/TRUTH-MAP.md](docs/TRUTH-MAP.md) |
| Operating rules and protocols | [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) |
| System architecture | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Runtime invariants (Top-10) | [docs/operations/runtime-invariants.md](docs/operations/runtime-invariants.md) |
| Service-level objectives | [docs/operations/slo.md](docs/operations/slo.md) |
| Runtime topology | [docs/RUNTIME.md](docs/RUNTIME.md) |
| HTTP endpoints | [docs/HTTP-API.md](docs/HTTP-API.md) |
| Daily workflow | [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) |
| Operations guides | [docs/operations/](docs/operations/README.md) |
| Architecture decisions | [docs/decisions/](docs/decisions/README.md) |
| Domain deep dives | [docs/domain/](docs/domain/README.md) |
| Terminology | [docs/GLOSSARY.md](docs/GLOSSARY.md) |
