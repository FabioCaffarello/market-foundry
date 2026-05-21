# Stage S214 — Analytical / Generated Path Consolidation Report

> **Date:** 2026-03-20
> **Type:** Consolidation (no functional expansion)
> **Predecessor:** S210 (Pre-Refactor Stabilization Gate)
> **Deliverables:** 3 architecture docs + code-level ownership annotations

---

## 1. Resumo Executivo

O S214 consolidou a convivência entre analytical path e generated path sem expandir nenhuma das duas camadas. O resultado é uma arquitetura onde ownership, boundaries e classificação de artefatos estão explícitos tanto em código quanto em documentação.

**Antes do S214:**
- Não havia distinção visual em código entre entradas manuais e codegen-governed no `pipeline.go` e nos registries.
- Múltiplos documentos descreviam boundaries com terminologia inconsistente (formato de markers S201 vs S202, status de golden snapshots para famílias não-integradas).
- A relação entre "ter spec + golden" e "ser governado pelo codegen" era ambígua — 5 famílias têm specs/goldens mas continuam manuais.
- Não existia um modelo canônico de classificação de artefatos (manual/generated/derived/operational).

**Depois do S214:**
- Todo código manualmente mantido nas zonas de convivência está anotado com `manual:owned`.
- O formato canônico de markers está consolidado (`codegen:begin/end`, formato S201).
- A ownership de cada artefato do sistema está documentada em referência definitiva.
- O modelo de 4 classificações (manual, generated, derived, operational) está formalizado.

---

## 2. Consolidação Aplicada

### 2.1 Anotações de Ownership em Código

Adicionadas anotações `manual:owned` explícitas em:

| Arquivo | Seção Anotada |
|---------|---------------|
| `cmd/writer/pipeline.go` | Evidence families (candle) — bloco manual separado dos codegen-governed |
| `cmd/writer/pipeline.go` | Decision/Strategy/Risk/Execution families — bloco manual com justificativa |
| `internal/adapters/nats/evidence_registry.go` | Writer consumer specs |
| `internal/adapters/nats/decision_registry.go` | Writer consumer specs |
| `internal/adapters/nats/strategy_registry.go` | Writer consumer specs |
| `internal/adapters/nats/risk_registry.go` | Writer consumer specs |
| `internal/adapters/nats/execution_registry.go` | Writer consumer specs |
| `internal/adapters/nats/signal_registry.go` | Clarificação: codegen governa RSI+EMA; store specs permanecem manuais |

### 2.2 Documentos Canônicos Produzidos

1. **`docs/architecture/analytical-generated-path-consolidation.md`**
   - Estado corrente da integração codegen ↔ analytical
   - Regras de markers canônicas
   - Conceitos superseded e deferrals confirmados
   - Invariantes de boundary

2. **`docs/architecture/analytical-vs-generated-ownership-and-boundaries.md`**
   - Modelo de 3 zonas (Human-Owned, Machine-Owned, Mixed)
   - Three-Condition Test para candidatos a geração
   - Escopo corrente do codegen (Tier 1: A1+A2 apenas)
   - Deferrals explícitos com blockers
   - Protocolo de integração
   - Chain de validação CI

3. **`docs/architecture/manual-generated-derived-operational-artifact-model.md`**
   - Definições formais das 4 classificações
   - Mapa completo de artefatos (domain, NATS, writer, codegen, ClickHouse, application, HTTP, gateway, store, migrations, config, CI)
   - Justificativas para o que permanece manual

---

## 3. Arquivos Alterados

### Código

| Arquivo | Tipo de Mudança |
|---------|----------------|
| `cmd/writer/pipeline.go` | Adição de comentários de ownership (`manual:owned`) |
| `internal/adapters/nats/evidence_registry.go` | Adição de comentário de ownership |
| `internal/adapters/nats/decision_registry.go` | Adição de comentário de ownership |
| `internal/adapters/nats/strategy_registry.go` | Adição de comentário de ownership |
| `internal/adapters/nats/risk_registry.go` | Adição de comentário de ownership |
| `internal/adapters/nats/execution_registry.go` | Adição de comentário de ownership |
| `internal/adapters/nats/signal_registry.go` | Clarificação de ownership mista |

### Documentação

| Arquivo | Status |
|---------|--------|
| `docs/architecture/analytical-generated-path-consolidation.md` | Novo |
| `docs/architecture/analytical-vs-generated-ownership-and-boundaries.md` | Novo |
| `docs/architecture/manual-generated-derived-operational-artifact-model.md` | Novo |
| `docs/stages/stage-s214-analytical-generated-path-consolidation-report.md` | Novo (este) |

---

## 4. Ownership/Boundaries Resultantes

### 4.1 Mapa de Ownership por Zona

| Zona | Conteúdo | Validação |
|------|----------|-----------|
| Zone 1 (Human) | ~95% do codebase | Code review + testes |
| Zone 2 (Machine) | 4 slices (2 consumer_spec + 2 pipeline_entry) | CI: `codegen-integrated-check.sh` |
| Zone 3 (Mixed) | 2 arquivos (`signal_registry.go`, `pipeline.go`) | Markers separam zones 1 e 2 |

### 4.2 Famílias por Status de Codegen

| Status | Famílias |
|--------|----------|
| Codegen-governed (markers + manifest) | rsi, ema |
| Spec exists, not integrated | candle, rsi_oversold, mean_reversion_entry, position_exposure, paper_order |
| No codegen representation | tradeburst, volume, ema_crossover, venue_market_order |

### 4.3 Paths por Ownership

| Path | Ownership | Codegen |
|------|-----------|---------|
| Writer pipeline declarations | Mixed (Zone 3) | 2 de 7 entries governed |
| NATS writer consumer specs | Mixed (Zone 3) | 2 de 7 functions governed |
| NATS store consumer specs | Human (Zone 1) | Nenhum |
| NATS registries (structs) | Human (Zone 1) | Nenhum |
| Store pipelines | Human (Zone 1) | Nenhum |
| ClickHouse readers | Human (Zone 1) | Nenhum |
| Analytical use cases | Human (Zone 1) | Nenhum |
| HTTP handlers/routes | Human (Zone 1) | Nenhum |
| Gateway composition | Human (Zone 1) | Nenhum |
| Migrations | Human (Zone 1) | Nenhum |
| Config | Human (Zone 1) | Nenhum |

---

## 5. Limites Remanescentes

### 5.1 Inconsistências Resolvidas

| Item | Antes | Depois |
|------|-------|--------|
| Formato de markers | S201 (`codegen:begin`) vs S202 (`BEGIN CODEGEN MANAGED`) | S201 é canônico; S202 é histórico |
| Golden snapshots para famílias não-integradas | Ambiguamente "machine-owned" | Clarificados como reference artifacts (human-crafted) |
| File integration status | Docs S197/S199 diziam "deferred" | S201 implementou; docs anteriores são históricos |

### 5.2 Drift Pré-Existente Documentado

O `codegen-integrated-check.sh` reporta drift em `rsi/consumer_spec` e `ema/consumer_spec`. Este drift é **pré-existente** (anterior ao S214): os golden snapshots contêm o literal `ConsumerSpec{}` expandido (output do template), mas o código alvo usa a factory `newConsumerSpec()` (introduzida para reduzir duplicação). Ambas as formas produzem valores idênticos em runtime. Pipeline entries (2 de 4 slices) passam sem drift.

**Opções para resolução futura:**
- Atualizar template para produzir factory calls (requer mudança de template)
- Atualizar golden snapshots para refletir factory pattern
- Aceitar como exceção documentada

### 5.3 Limites que Permanecem por Design

| Limite | Justificativa |
|--------|---------------|
| Apenas 2 de 7 famílias são codegen-governed | Integração das 5 restantes não foi autorizada |
| Mapper generation bloqueado | Requer extensão `domain.columns` no spec schema |
| Store path inteiramente manual | Padrão de actor diferente (projection + consumer) |
| Read path inteiramente manual | Tier 2 não autorizado; padrão estrutural diferente |
| Config entries manuais | Tooling JSONC não implementado |

### 5.4 Ruído Documental Remanescente

Os documentos de S193–S204 continuam existindo com suas descrições históricas. Os novos documentos do S214 consolidam e supersede as regras dispersas, mas os documentos históricos não foram editados (preservam o registro de decisão original).

---

## 6. Preparação Recomendada para S215

### Opção A: Integração das 5 famílias remanescentes

Se o próximo passo for migrar candle, rsi_oversold, mean_reversion_entry, position_exposure e paper_order para codegen governance:

1. Colocar markers manualmente nos 5 pontos de cada registry + pipeline.go
2. Atualizar `integrated.yaml` com os novos slices
3. Executar `codegen-integrated-check.sh` para validar equivalência
4. Nenhuma mudança funcional — o código entre markers já é estruturalmente equivalente aos goldens

**Pré-requisito:** Decisão explícita sobre se a evidence layer (candle) segue o mesmo protocolo, dada sua naming exception.

### Opção B: Refactor tranche continuation

Se o próximo passo for continuar a wave de refactor (S211–S213):

1. O S214 deixa a base limpa para refactors — ownership está explícita, boundaries estão marcadas.
2. Refactors podem tocar zonas `manual:owned` sem risco de conflito com codegen.
3. Refactors não devem tocar zonas `codegen:begin/end` sem passar pelo pipeline de regeneração.

### Opção C: Mapper generation (A3)

Se o próximo passo for habilitar geração de mappers:

1. Requer extensão do spec schema com `domain.columns`
2. Requer novo template (`mapper.go.tmpl`)
3. Requer equivalence proof (como S196 fez para A1+A2)
4. Requer novo golden snapshot set
5. Requer atualização do `integrated.yaml`

**Recomendação:** Opção A é a de menor risco e maior retorno incremental. Migra 5 famílias para governance sem mudança funcional, usando a infraestrutura que já existe.

---

## 7. Guard Rails Verificados

| Guard Rail | Status |
|------------|--------|
| Não abrir novas famílias | ✓ Nenhuma família nova |
| Não ampliar codegen | ✓ Nenhum template/spec/golden novo |
| Não abrir novos endpoints | ✓ Nenhum endpoint novo |
| Não mascarar conflitos de ownership | ✓ Ownership explicitada em código e docs |
| Documentar o que permanece manual e por quê | ✓ Artifact model completo |

---

## 8. Acceptance Criteria

| Critério | Evidência |
|----------|-----------|
| Analytical path e generated path mais coerentes | Ownership markers unificam a linguagem visual |
| Ownership e boundaries mais explícitos | 3 documentos canônicos + code annotations |
| Ruído estrutural reduzido | Inconsistências S201/S202 resolvidas; golden snapshot status clarificado |
| Base mais limpa para próxima evolução | Manual:owned + codegen:begin/end criam contratos visuais claros |
| Consolida sem expandir | Zero mudanças funcionais; apenas comentários e documentação |
