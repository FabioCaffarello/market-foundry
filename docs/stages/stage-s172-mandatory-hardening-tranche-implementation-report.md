# Stage S172 — Mandatory Hardening Tranche Implementation Report

## Resumo Executivo

S172 executou a tranche obrigatória de hardening definida no S171, implementando os três itens mandatórios que bloqueavam a entrada da Família 3 na Wave B. As mudanças são puramente estruturais — zero expansão funcional, zero mudanças de schema, zero novos endpoints.

## Tranche de Hardening Aplicada

### H-3: Helper Renaming (Fase 1)

| Item | Antes | Depois |
|---|---|---|
| Struct | `evidenceKeyParams` | `queryKeyParams` |
| Function | `parseEvidenceKeyParams()` | `parseQueryKeyParams()` |
| Scope | 7 handler files | 7 handler files (same) |

**Decisão:** Renomeado para `parseQueryKeyParams` (não `parseAnalyticalKeyParams`) porque a função é usada por **todas** as famílias de handlers (operacionais e analíticas), não apenas endpoints analíticos. Um nome neutro reflete o escopo real.

### H-1: Struct DI (Fase 2)

| Item | Antes | Depois |
|---|---|---|
| Constructor | 4 positional args | `AnalyticalHandlerDeps` struct |
| Caller changes | N/A | routes/analytical.go + 20 test calls |
| Family add cost | Edit signature + all callers | Add 1 struct field |

### H-2: Smoke Extraction (Fase 3)

| Item | Antes | Depois |
|---|---|---|
| Per-family validation | ~80 lines copy-paste | `validate_analytical_family()` call (~7 lines) |
| Error handling | ~12 lines per family | `validate_analytical_error_handling()` call (~3 lines) |
| New family cost | ~80 lines manual | ~7 lines parameterized |

## Arquivos Alterados

### Go (handlers + routes + tests)
- `internal/interfaces/http/handlers/evidence.go` — struct + function rename
- `internal/interfaces/http/handlers/analytical.go` — struct DI + rename
- `internal/interfaces/http/handlers/signal.go` — rename
- `internal/interfaces/http/handlers/decision.go` — rename
- `internal/interfaces/http/handlers/strategy.go` — rename
- `internal/interfaces/http/handlers/risk.go` — rename
- `internal/interfaces/http/handlers/execution.go` — rename
- `internal/interfaces/http/routes/analytical.go` — struct DI caller
- `internal/interfaces/http/handlers/analytical_test.go` — 20 test constructors

### Shell (smoke)
- `scripts/smoke-analytical-e2e.sh` — function extraction + call site replacement

### Documentation
- `docs/architecture/mandatory-hardening-tranche-implementation-notes.md`
- `docs/architecture/pattern-hardening-after-wave-b-family-02.md`
- `docs/stages/stage-s172-mandatory-hardening-tranche-implementation-report.md`

## Ganhos Estruturais Obtidos

1. **Repetibilidade:** Adicionar a Família 3 ao handler analítico requer 1 campo struct + 1 use case + 1 bloco de rota (sem churn de assinatura).
2. **Repetibilidade (smoke):** Adicionar a Família 3 ao smoke requer ~7 linhas chamando `validate_analytical_family()`.
3. **Clareza semântica:** `parseQueryKeyParams` reflete o escopo real (todas as famílias, não apenas evidence).
4. **Testabilidade:** Struct DI permite testes focados por família sem array posicional de nils.
5. **Governança:** Funções extraídas no smoke impõem validação uniforme (meta, Server-Timing, structure) para todas as famílias.

## Limites Mantidos

- Nenhuma família nova adicionada
- Nenhuma expansão funcional
- Nenhuma mudança de schema ou migration
- Nenhuma mudança no writer ou adapter layer
- Nenhuma abstração nova além do `AnalyticalHandlerDeps` struct
- Handlers operacionais mantêm constructors posicionais (sem pressão de expansão)
- `parseQueryKeyParams` permanece em `evidence.go` (visível ao package inteiro via Go scoping)
- Nenhuma integração CI smoke (fora de escopo S171)

## Validação

- `go build ./cmd/gateway/...` — compila limpo
- `go build ./internal/interfaces/http/...` — compila limpo
- `go test ./internal/interfaces/http/handlers/...` — todos os testes passam
- `bash -n scripts/smoke-analytical-e2e.sh` — syntax check passa
- `grep -r parseEvidenceKeyParams --include='*.go'` — 0 matches (legacy eliminado)
- `grep AnalyticalHandlerDeps` — presente em handler, routes e tests

## Preparação Recomendada para S173

S173 deve ser um **gate formal de entrada para Família 3** que:

1. Confirme que os 3 blockers do `family-03-blockers-and-hardening-success-criteria.md` estão resolvidos (verificação automática via grep/build)
2. Defina qual será a Família 3 (candidates: strategies, risk_assessments, executions)
3. Estime o escopo de schema + writer + reader + gateway + smoke para a família escolhida
4. Confirme que o padrão v2 (`wave-b-family-expansion-pattern-v2.md`) é suficiente ou precisa de ajustes
5. Avalie se codegen deve ser investigado para Família 4 (threshold do padrão v2)

**O S172 é a última etapa antes do gate. A base está pronta.**
