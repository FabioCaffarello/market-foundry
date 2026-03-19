# Stage S83: Execute Runtime Governance and Activation Hardening

**Status**: Complete
**Date**: 2026-03-18

## Resumo Executivo

O S83 endureceu a governança do runtime `execute`, eliminando drift entre o estado real do código (pós-S80/S82) e o que o tooling/docs/config refletiam. O adapter selection passou de hardcoded para config-driven, o raccoon-cli foi atualizado com todos os artefatos canônicos, e o dead code de pre-implementation guard foi removido.

Nenhuma funcionalidade nova foi adicionada. Nenhum venue real foi introduzido. O escopo foi estritamente governança e alinhamento.

## Hardening de Governança Aplicado

### 1. Config-Driven Venue Adapter Selection

**Antes (S80):** `PaperVenueAdapter` hardcoded em `cmd/execute/run.go` linha 33.

**Depois (S83):**
- `VenueConfig` struct adicionado em `settings/schema.go` com validação de tipo
- `venue.type` seção adicionada em `execute.jsonc`
- `buildVenueAdapter()` factory function em `run.go` seleciona adapter via config
- Apenas `paper_simulator` é permitido; qualquer outro tipo é rejeitado no boot
- Backward compatible: `venue.type` vazio faz fallback para `paper_simulator`

### 2. Schema Validation Expansion

- `VenueConfig.Validate()` rejeita venue types desconhecidos
- `AppConfig.Validate()` agora inclui venue validation
- `knownVenueTypes` registry — ponto único para venue types aprovados

### 3. Raccoon-CLI Complete Alignment

**Constantes atualizadas:**

| Constante | Antes | Depois | Delta |
|-----------|-------|--------|-------|
| `APP_BINARIES` | 5 | 6 | +execute |
| `CANONICAL_STREAMS` | 8 | 9 | +EXECUTION_FILL_EVENTS |
| `EXECUTION_DOCS` | 7 | 9 | +2 docs de governance |
| `EXECUTION_EXPECTED_SUBJECTS` | 2 | 6 | +fill, +status, +control |
| `EXECUTION_EXPECTED_DURABLES` | 1 | 3 | +execute intake, +store fill |
| `EXECUTION_EXPECTED_BUCKETS` | 1 | 3 | +venue latest, +control |
| `EXECUTION_ADAPTER_FILES` | 5 | 7 | +control gateway/kv |
| `EXECUTION_DOMAIN_FILES` | 6 | 13 | +control, venue, staleness, use cases |

**Checks aprimorados:**
- `check_execution_domain_drift` agora verifica execute scope actors
- `check_execution_config_drift` agora verifica execute.jsonc + venue config

### 4. Dead Code Removal

- `check_execution_premature_implementation` removido — era o guard S70 pre-implementation que gerava false positives dado que execution já está completamente implementado

## Arquivos Alterados

### Runtime/Config
| Arquivo | Mudança |
|---------|---------|
| `deploy/configs/execute.jsonc` | Adicionado seção `venue` com `type: "paper_simulator"` |
| `internal/shared/settings/schema.go` | Adicionado `VenueConfig`, `VenueType`, `knownVenueTypes`, validação |
| `cmd/execute/run.go` | Substituído adapter hardcoded por `buildVenueAdapter()` factory |

### Tooling
| Arquivo | Mudança |
|---------|---------|
| `tools/raccoon-cli/src/analyzers/drift_detect.rs` | Atualizado constants, checks, removido dead code |

### Documentação
| Arquivo | Tipo |
|---------|------|
| `docs/architecture/execute-governance-and-activation-model.md` | Novo — modelo de governança consolidado |
| `docs/tooling/cli-execute-drift-rules.md` | Novo — regras de drift específicas do execute binary |
| `docs/tooling/cli-execution-drift-rules.md` | Atualizado — reflete estado ativo de ED-1..ED-6 |
| `docs/stages/stage-s83-execute-runtime-governance-and-activation-hardening-report.md` | Novo — este relatório |

## Legado/Drift Removidos

| Item | Tipo | Ação |
|------|------|------|
| Adapter hardcoded em run.go | Drift de governança | Substituído por factory config-driven |
| `check_execution_premature_implementation` | Dead code | Removido (era guard S70, agora obsoleto) |
| ED-3..ED-6 marcados como "prepared" nos docs | Drift documental | Atualizados para "active" |
| `EXECUTION_FILL_EVENTS` ausente de `CANONICAL_STREAMS` | Drift de tooling | Adicionado |
| `execute` ausente de `APP_BINARIES` | Drift de tooling | Adicionado |
| Control/fill subjects/durables/buckets ausentes | Drift de contracts | Adicionados todos os artefatos canônicos |

## Gates que Seguem Bloqueando Venue Real

| Gate | Status | Descrição |
|------|--------|-----------|
| `knownVenueTypes` | Bloqueado | Apenas `paper_simulator` registrado |
| `buildVenueAdapter()` | Bloqueado | Switch rejeita qualquer type não-paper |
| Schema validation | Bloqueado | `VenueConfig.Validate()` rejeita types desconhecidos |
| Activation gate ceremony | Não iniciada | Requer: implementation, tests, gates G-S1..G-O4 |
| Credential infrastructure | Não existe | Sem gerenciamento de credenciais de exchange |
| Multi-venue routing | Não existe | Binary suporta apenas um venue type |

## Verificação

- `go build ./cmd/execute/...` — compila com sucesso
- `go test ./internal/shared/settings/...` — todos os testes passam
- `cargo build` (raccoon-cli) — compila com sucesso

## Preparação Recomendada para S84

1. **NATS Integration Testing**: Rodar `check_execution_contracts_drift` com um cluster NATS real e verificar que todos os 6 subjects, 3 durables e 3 buckets são criados corretamente.

2. **Fill Projection Completion**: Store-side fill consumer + projection actor (`fill_consumer_actor.go`, `fill_projection_actor.go`) ainda são untracked files. Integrá-los formalmente e adicionar à governança.

3. **End-to-End Smoke Test**: Estender `scripts/smoke-multi-symbol.sh` para incluir o execute binary no pipeline e verificar que fills são publicados e projetados.

4. **Operational Validation Matrix**: Usar `docs/architecture/execution-operational-validation-matrix.md` (se existir) para validar os checks operacionais com o pipeline completo.

5. **Docker Compose Integration**: Adicionar `execute` como service no docker-compose.yaml (atualmente apenas warning do raccoon-cli).
