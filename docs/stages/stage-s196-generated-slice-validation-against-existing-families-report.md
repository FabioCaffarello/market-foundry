# Stage S196: Generated Slice Validation Against Existing Families

**Status**: COMPLETE
**Date**: 2026-03-20
**Predecessor**: S195 (Minimal Codegen Engine for a Narrow Slice)
**Gate**: S194 (Manual-to-Generated Equivalence Baseline)

---

## 1. Resumo Executivo

O Stage S196 validou o slice gerado (A1: consumer spec + A2: pipeline entry) contra **todas as 6 famílias manuais existentes**, não apenas as 2 famílias baseline (RSI, Paper Order) usadas no S195. **12 de 12 comparações passam com equivalência estrutural.**

O engine de codegen produz artefatos que, após normalização S194 (strip comments, normalize whitespace, remove blank lines), são idênticos byte-a-byte ao código escrito manualmente. Três instâncias de drift cosmético foram detectadas (diferenças de comentário), todas classificadas como INFO — zero impacto comportamental.

A validação de codegen agora entra no CI como um job independente (`codegen-golden`) que bloqueia merge em caso de regressão.

---

## 2. Resultados de Equivalência

### Matriz Completa

| Família | Camada | consumer_spec | pipeline_entry | Baseline S195? |
|---------|--------|:------------:|:--------------:|:--------------:|
| candle | evidence | PASS | PASS | Não |
| rsi | signal | PASS | PASS | Sim |
| rsi_oversold | decision | PASS | PASS | Não |
| mean_reversion_entry | strategy | PASS | PASS | Não |
| position_exposure | risk | PASS | PASS | Não |
| paper_order | execution | PASS | PASS | Sim |

**Total: 12/12 PASS (100%)**

### Cobertura de Validação por Dimensão

| Dimensão | Cobertura | Evidência |
|----------|-----------|-----------|
| Todas as 6 camadas (L1-L6) | ✓ | Cada camada tem pelo menos 1 família |
| Exceções evidence layer | ✓ | candle valida 3 exceções documentadas |
| Abreviações conhecidas | ✓ | RSI, RSIOversold exercitam mapa de abreviações |
| Nomes multi-word | ✓ | mean_reversion_entry, position_exposure, paper_order |
| Mínima complexidade | ✓ | candle (evidence, 1 palavra) |
| Máxima complexidade | ✓ | paper_order (execution, compound names) |

---

## 3. Drift e Limites Encontrados

### Drift Cosmético (Aceitável)

| ID | Tipo | Impacto | Severidade |
|----|------|---------|-----------|
| D1 | Phrasing de comentário (template vs manual) | Zero | INFO |
| D2 | Comprimento de dashes decorativos em seção | Zero | INFO |
| D3 | Evidence layer omite "evidence" no comentário live | Zero | INFO |

Todos os 3 são diferenças em texto de comentário, removidos pela normalização. Nenhum afeta estrutura ou comportamento.

### Drift Perigoso Detectado

**Nenhum.** Zero divergência estrutural em 12 pontos de comparação.

### Limites Atuais do Engine

| Limite | Descrição | Bloqueador Para |
|--------|-----------|-----------------|
| Cobertura de artefatos | 2 de 6 artefatos Tier 1 | Geração autônoma de família completa |
| Mapper (A3) | Requer extensão `domain.columns` no spec | Escrita completa no write-path |
| Mapper tests (A4) | Depende de A3 | Testes do mapper |
| Config entry (A5) | Requer manipulação JSONC | Config writer.jsonc |
| Smoke phase (A6) | Requer geração de shell script | Teste E2E |
| File integration | Engine produz fragmentos, não escreve em arquivos | Automação completa |
| Tier 2 | Não autorizado | Read-path completo |

### O Que Merece Ser Gerado vs O Que Ainda Não

| Artefato | Merece gerar? | Justificativa |
|----------|:------------:|---------------|
| A1: Consumer spec | ✓ Sim | 100% equivalência, 6/6 famílias, zero decisão criativa |
| A2: Pipeline entry | ✓ Sim | 100% equivalência, 6/6 famílias, zero decisão criativa |
| A3: Mapper | ✗ Ainda não | Requer spec extension + validação de column-order contra DDL |
| A4: Mapper tests | ✗ Ainda não | Depende de A3 |
| A5: Config entry | ✗ Ainda não | JSONC tooling não implementado |
| A6: Smoke phase | ✗ Ainda não | Shell template não implementado |

---

## 4. Arquivos Alterados

### Novos (Codegen)

| Arquivo | Propósito |
|---------|-----------|
| `codegen/families/candle.yaml` | Spec para família candle (evidence) |
| `codegen/families/rsi_oversold.yaml` | Spec para família rsi_oversold (decision) |
| `codegen/families/mean_reversion_entry.yaml` | Spec para família mean_reversion_entry (strategy) |
| `codegen/families/position_exposure.yaml` | Spec para família position_exposure (risk) |
| `codegen/golden-snapshots/candle/*.go.golden` | Golden snapshots para candle |
| `codegen/golden-snapshots/rsi_oversold/*.go.golden` | Golden snapshots para rsi_oversold |
| `codegen/golden-snapshots/mean_reversion_entry/*.go.golden` | Golden snapshots para mean_reversion_entry |
| `codegen/golden-snapshots/position_exposure/*.go.golden` | Golden snapshots para position_exposure |

### Modificados

| Arquivo | Mudança |
|---------|---------|
| `codegen/render_test.go` | +8 golden tests por família, +1 TestCheckAllFamilies cross-validation gate, +4 fixture functions |
| `Makefile` | +2 targets (codegen-check, codegen-test), +help text |
| `.github/workflows/ci.yml` | +codegen-golden CI job |

### Novos (Docs)

| Arquivo | Propósito |
|---------|-----------|
| `docs/architecture/generated-slice-validation-against-existing-families.md` | Validação cross-family detalhada |
| `docs/architecture/codegen-drift-findings-and-equivalence-results.md` | Drift e equivalência medidos |
| `docs/architecture/codegen-slice-01-ci-validation-strategy.md` | Estratégia CI para codegen |

---

## 5. Estratégia de Validação no CI

### Job Adicionado: `codegen-golden`

```yaml
codegen-golden:
  name: Codegen Golden Equivalence
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - run: make codegen-check    # 12 comparações golden
    - run: make codegen-test     # 26 unit tests
```

- Executa em paralelo com `unit-tests` (sem dependência)
- Duração estimada: ~3s
- Bloqueante: falha impede merge
- Auto-extensível: novos artefatos em `SupportedArtifacts()` são automaticamente cobertos

### Targets Makefile

```bash
make codegen-check   # Roda check-all (12 golden comparisons)
make codegen-test    # Roda go test ./... no módulo codegen
```

---

## 6. Preparação Recomendada para S197

### Opção A: Ampliar Slice (Slice 02 — Mapper Generation)
Se o objetivo é progredir o codegen para cobrir mais artefatos:
1. Implementar extensão `domain.columns` no schema do spec
2. Criar template `mapper.go.tmpl`
3. Extrair golden snapshots dos mappers manuais existentes
4. Validar equivalência column-order contra DDL
5. Adicionar `mapper_test.go.tmpl` (A4)
6. Estender `SupportedArtifacts()` e golden comparisons

**Risco**: Column-order é estrutural, não cosmético (ClickHouse positional binding). Essa validação é mais difícil que A1+A2.

### Opção B: Primeira Família Gerada (Within-Layer Expansion)
Se o objetivo é provar valor de produção com o slice atual:
1. Escolher uma nova família dentro de uma camada existente (e.g., novo signal type)
2. Criar spec para a nova família
3. Gerar A1 + A2 a partir do spec
4. Escrever A3-A6 manualmente
5. Integrar gerado + manual no codebase
6. Validar via CI + smoke-analytical

**Risco menor**: A1+A2 são validados. Os artefatos manuais (A3-A6) seguem padrão provado.

### Opção C: Hardening e Automação
Se o objetivo é solidificar antes de expandir:
1. Implementar file integration (marker sections)
2. Adicionar `codegen-drift` CI job (regenerate → diff vs committed)
3. Adicionar spec linting (validate uniqueness, referential integrity)
4. Documentar workflow completo para primeira família gerada

### Recomendação

**Opção B** é a de maior valor incremental: prova que o codegen produz valor real (reduz trabalho manual de ~45min para ~15min por família) mesmo cobrindo apenas 2 de 6 artefatos. As 4 famílias restantes por artefato serão escritas manualmente, mas isso é aceitável como primeiro passo.

**Opção A** é o próximo passo técnico lógico, mas o mapper é o artefato mais complexo do Tier 1 e merece um stage dedicado.

---

## 7. Decisões Registradas

| ID | Decisão | Justificativa |
|----|---------|---------------|
| S196-D1 | Expandir cobertura de famílias de 2 para 6 | Provar que equivalência não é acidente dos 2 baselines |
| S196-D2 | Criar CI job independente (não nested em unit-tests) | Codegen é módulo standalone; falha isolada é mais clara |
| S196-D3 | Classificar drift cosmético como INFO | Comentários não afetam comportamento; normalização os trata corretamente |
| S196-D4 | Não expandir escopo do codegen para A3-A6 | Fora do escopo deste stage; mapper requer spec extension |
| S196-D5 | TestCheckAllFamilies como gate test | Um teste que cobre todas as famílias × artefatos automaticamente |

---

## 8. Métricas

| Métrica | Valor |
|---------|-------|
| Testes no S195 | 17 |
| Testes adicionados no S196 | 9 (8 golden + 1 cross-validation) |
| Total de testes codegen | 26 |
| Golden comparisons S195 | 4 (2 famílias × 2 artefatos) |
| Golden comparisons S196 | 12 (6 famílias × 2 artefatos) |
| Specs YAML | 6 (2 existentes + 4 novos) |
| Golden snapshot files | 12 (2 existentes × 2 = 4, + 4 novos × 2 = 8) |
| CI time adicionado | ~3s |

---

## 9. Critérios de Aceite — Checklist

- [x] O slice gerado é validado contra a baseline existente
- [x] Equivalência e drift ficam explícitos (3 INFO, 0 WARNING, 0 CRITICAL)
- [x] Os limites atuais do engine ficam claros (2 de 6 artefatos Tier 1)
- [x] A entrada no CI fica definida (codegen-golden job + make targets)
- [x] A base fica pronta para decidir sobre primeira família gerada (sim para A1+A2; não para A3-A6)
