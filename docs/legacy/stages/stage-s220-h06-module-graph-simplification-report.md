# S220: H-06 Module Graph Simplification — Stage Report

**Status:** complete
**Scope:** H-06 — module graph simplification
**Baseline:** all 17 modules build and test clean

## 1. Resumo Executivo

O S220 simplificou o module graph do market-foundry de 19 para 17 módulos Go, eliminando dois módulos sem justificativa de isolamento (`internal/migrate` e `internal/adapters/repositories`). A simplificação não criou novas dependências externas, não alterou nenhuma API pública e preservou integralmente a baseline de build e testes.

## 2. Simplificações Aplicadas

### 2.1 `internal/migrate` → `cmd/migrate/migrate`

- **Motivação:** módulo de 478 LOC sem dependências externas, consumido exclusivamente por `cmd/migrate/main.go`. O boundary do módulo existia sem benefício de isolamento.
- **Ação:** moveu 6 arquivos (4 source + 2 test) para sub-package `cmd/migrate/migrate/`. Atualizou import em `main.go`. Removeu entry do `go.work` e deletou diretório original.
- **Risco:** zero — relação 1:1 consumidor, sem dependências transitivas.

### 2.2 `internal/adapters/repositories` → `internal/application/configctl/memoryrepo`

- **Motivação:** módulo de 1.434 LOC sem dependências externas, consumido por apenas 2 arquivos. O módulo `internal/application` já define as ports (interfaces) que o repositório implementa, tornando-o o destino natural.
- **Ação:** moveu 3 arquivos para `internal/application/configctl/memoryrepo/`, atualizou package name para `memoryrepo`, atualizou imports nos 2 consumidores. Removeu entry do `go.work` e deletou diretório original.
- **Risco:** zero — ambos consumidores já dependiam de `internal/application`.

## 3. Arquivos Alterados

### Criados
- `cmd/migrate/migrate/migration.go`
- `cmd/migrate/migrate/catalog.go`
- `cmd/migrate/migrate/runner.go`
- `cmd/migrate/migrate/checksum.go`
- `cmd/migrate/migrate/catalog_test.go`
- `cmd/migrate/migrate/checksum_test.go`
- `internal/application/configctl/memoryrepo/repository.go`
- `internal/application/configctl/memoryrepo/records.go`
- `internal/application/configctl/memoryrepo/repository_test.go`

### Modificados
- `go.work` — removidos 2 entries (`./internal/migrate`, `./internal/adapters/repositories`)
- `cmd/migrate/main.go` — import path atualizado
- `internal/actors/scopes/configctl/control_router.go` — import path atualizado
- `internal/application/configctl/usecases_test.go` — import path atualizado

### Removidos
- `internal/migrate/` — diretório completo (6 arquivos + go.mod)
- `internal/adapters/repositories/` — diretório completo (3 arquivos + go.mod)

### Documentação
- `docs/architecture/h06-module-graph-simplification.md`
- `docs/architecture/module-graph-before-and-after.md`
- `docs/stages/stage-s220-h06-module-graph-simplification-report.md`

## 4. Before/After

| Métrica | Antes | Depois | Delta |
|---------|-------|--------|-------|
| Módulos no workspace | 19 | 17 | −2 (−10.5%) |
| Arquivos `go.mod` | 19 | 17 | −2 |
| Dependências externas | inalterado | inalterado | 0 |
| Testes | inalterado | inalterado | 0 |
| Build baseline | green | green | — |

## 5. Limites e Trade-offs

### O que NÃO foi simplificado (e por quê)

| Módulo | Razão para manter separado |
|--------|---------------------------|
| `internal/interfaces/http` | 5.568 LOC, 37 arquivos — massa crítica justifica módulo próprio |
| `internal/adapters/clickhouse` | Dep externa (clickhouse-go) — isolamento evita poluir consumers não-analíticos |
| `internal/adapters/exchanges` | Dep externa (gorilla/websocket) — isolamento mantém WebSocket fora de non-ingest |
| `internal/adapters/nats` | Deps externas (nats-go, cbor) — maior módulo adaptador, 9 sub-packages |
| Merge de todos `adapters/*` | Combinaria clickhouse-go + websocket + nats, aumentando audit surface |
| `internal/domain` + `internal/shared` | Perfis de dependência diferentes, ambos fundacionais e grandes |

### Trade-offs aceitos

- **`cmd/migrate/migrate/` package name**: o nome `migrate` dentro de `cmd/migrate` é auto-referencial, mas mantém paridade com o import path anterior e não gera conflito.
- **`memoryrepo` dentro de `application`**: pode-se argumentar que a implementação de repositório é um "adapter" e não pertence a `application`. No entanto, como é uma implementação in-memory sem deps externas e `application` define as ports, a colocalização é pragmaticamente correta.

## 6. Preparação para S221

O module graph está agora em 17 módulos com topologia mais clara:
- **Camada foundation**: `shared`, `domain` (0 deps internas)
- **Camada adapter**: `clickhouse`, `exchanges`, `nats` (deps externas isoladas)
- **Camada application**: `application` (ports + use cases + memoryrepo)
- **Camada orchestration**: `actors`, `interfaces/http`
- **Camada binary**: 8 `cmd/*` modules
- **Standalone**: `codegen`

Recomendações para próxima etapa:
1. **Reconciliação documental** — atualizar documentation-canonical-map e technical-debt-registry para refletir o novo estado pós-S220.
2. **Avaliar consolidação de `cmd/*` binários** — cmd/configctl, cmd/store e cmd/execute são muito finos (< 200 LOC cada) e poderiam ser avaliados como candidatos a merge em um futuro stage, se houver ganho operacional.
3. **Revisitar `internal/interfaces/http`** — se o gateway continuar sendo o único consumidor, avaliar absorção em S222+.
