# Stage S282 — CI Enforcement and Non-Skipping Test Baseline

Status: **Complete**
Date: 2026-03-21
Gate: Pre-requisite for Signal Evolution Wave (S283–S287)
Predecessor: S281 (Post-Operational Proof Feature Gate)

## 1. Resumo Executivo

S282 eliminou o estado em que 40 testes auto-skipavam silenciosamente na CI,
inflando a contagem de testes sem exercitar código. Através de build tags e
uma NATS service container na CI, o baseline agora é honesto: todo teste que
compila ou passa ou falha. Zero auto-skips.

## 2. Estado Anterior da Suíte

### Problema central

A suíte continha 40 testes com lógica `t.Skip`/`t.Skipf` que dependia de
infraestrutura externa (NATS, ClickHouse). Como a CI não provisionava essa
infraestrutura, esses testes auto-skipavam em cada run, aparecendo como
"passed" nos relatórios do Go test runner.

### Impacto

| Aspecto | Estado Pré-S282 |
|---------|----------------|
| Testes auto-skipping na CI | 40 |
| Job `integration-tests` efetivo | No-op (tag `integration` sem arquivos tagueados) |
| Confiança no baseline | Falsa — 40 testes nunca executados |
| Build tags na suíte | 0 arquivos |

## 3. Skips Classificados por Causa

### Causa 1: Dependência de NATS externo (39 testes, 6 arquivos)

| Arquivo | Testes | Stage |
|---------|--------|-------|
| `natsexecution/kv_store_roundtrip_test.go` | 8 | S271 |
| `natsexecution/control_gate_runtime_test.go` | 6 | S273 |
| `natsexecution/control_plane_full_path_test.go` | 5 | S275 |
| `natsexecution/multi_binary_integration_test.go` | 6 | S276 |
| `natsexecution/restart_recovery_test.go` | 10 | S280 |
| `writerpipeline/restart_recovery_test.go` | 4 | S280 |

**Mecanismo de skip:** Helper `natsURL()` / `wrNATSURL()` faz TCP dial a
`localhost:4222`. Se falha, chama `t.Skipf()`.

**Classificação:** Skip legítimo por dependência externa, mas evitável com
provisionamento correto na CI.

**Remediação:** Build tag `//go:build integration` + NATS service container.

### Causa 2: Dependência de ClickHouse externo (1 teste, 1 arquivo)

| Arquivo | Testes | Stage |
|---------|--------|-------|
| `clickhouse/live_execution_analytical_test.go` | 1 | S277 |

**Mecanismo de skip:** Checa `CLICKHOUSE_DSN` env var e faz ping ao server.

**Classificação:** Skip legítimo — ClickHouse é pesado demais para service
container em CI genérica. O path é validado pelo job `smoke-analytical`.

**Remediação:** Build tag `//go:build requireclickhouse` + target `make test-clickhouse`.

### Causa 3: Fragilidade de harness — Nenhum encontrado

### Causa 4: Dívida estrutural corrigível — Nenhum encontrado

## 4. Correções Realizadas

### 4.1 Build Tags Adicionadas (7 arquivos)

```
//go:build integration
  internal/adapters/nats/natsexecution/kv_store_roundtrip_test.go
  internal/adapters/nats/natsexecution/control_gate_runtime_test.go
  internal/adapters/nats/natsexecution/control_plane_full_path_test.go
  internal/adapters/nats/natsexecution/multi_binary_integration_test.go
  internal/adapters/nats/natsexecution/restart_recovery_test.go
  internal/adapters/clickhouse/writerpipeline/restart_recovery_test.go

//go:build requireclickhouse
  internal/adapters/clickhouse/live_execution_analytical_test.go
```

### 4.2 CI Workflow (`ci.yml`)

- Job `integration-tests` agora provisiona NATS como service container
- Imagem: `nats:2.10.18-alpine` (mesma versão do compose de produção)
- Health check via `wget http://127.0.0.1:8222/healthz`
- `NATS_URL=nats://localhost:4222` injetado como env var

### 4.3 Makefile

- Target `test-clickhouse` adicionado (build tag `requireclickhouse`)
- Help text atualizado para refletir requisitos reais
- `.PHONY` atualizado

## 5. Baseline Final e Enforcement

### Contagem de Auto-Skips por Job

| Job CI | Antes | Depois |
|--------|-------|--------|
| `unit-tests` | 40 | **0** |
| `integration-tests` | 39 (no-op) | **0** (39 executam de verdade) |
| `codegen-golden` | 0 | 0 |
| `behavioral-scenarios` | 0 | 0 |
| `smoke-analytical` | 0 | 0 |

### Regras de Enforcement

1. `make test` → zero `t.Skip` em arquivos sem build tag
2. Novos testes NATS → obrigatório `//go:build integration`
3. Novos testes ClickHouse → obrigatório `//go:build requireclickhouse`
4. Todo job CI deve estar green para merge

### Taxonomia de Build Tags

| Tag | Infra | CI Job |
|-----|-------|--------|
| *(nenhuma)* | Nenhuma | `unit-tests` |
| `integration` | NATS | `integration-tests` |
| `requireclickhouse` | ClickHouse | Local / `smoke-analytical` (system-level) |

## 6. Preparação Recomendada para S283

1. **Novos signal samplers** (Bollinger, etc.) devem ser unit tests puros
   sem build tags — seguem o padrão de `bollinger_sampler_test.go`.

2. **Se S283+ precisar de novos testes NATS** (e.g., stream de novos signal
   events), usar `//go:build integration` e garantir que o service container
   NATS na CI suporta o cenário.

3. **O baseline honesto de S282** serve como gate: qualquer regressão em
   testes que antes eram skipped agora será visível imediatamente como
   falha, não como skip silencioso.

4. **Não é necessário expandir a CI** para S283–S287 a menos que surjam
   novos tipos de dependência externa além de NATS e ClickHouse.

## Entregáveis

| Entregável | Status |
|-----------|--------|
| Ajustes em código/testes/scripts | Completo (7 test files + ci.yml + Makefile) |
| `docs/architecture/ci-enforcement-and-non-skipping-test-baseline.md` | Completo |
| `docs/architecture/test-skip-cause-matrix-and-remediation.md` | Completo |
| `docs/stages/stage-s282-ci-enforcement-and-non-skipping-test-baseline-report.md` | Este documento |

## Critérios de Aceite

| Critério | Resultado |
|----------|-----------|
| Testes auto-skip mapeados e classificados | 40 testes, 2 causas raiz, rigor completo |
| Redução real de skips evitáveis | 40 → 0 auto-skips na CI |
| Baseline mais honesto e útil como gate | Sim — todo teste que compila executa |
| CI apta como pré-requisito para Signal Evolution Wave | Sim — enforcement contract definido |
