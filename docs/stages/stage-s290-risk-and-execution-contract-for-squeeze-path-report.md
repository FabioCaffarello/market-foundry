# S290: Risk and Execution Contract for Squeeze Path — Stage Report

Status: **Complete**
Date: 2026-03-21

---

## 1. Resumo Executivo

O S290 fecha o contrato de risk→execution para o slice de squeeze breakout, garantindo que o caminho completo `bollinger_squeeze decision → squeeze_breakout_entry strategy → position_exposure/drawdown_limit risk → paper_order execution` funciona de ponta a ponta no Foundry, reaproveitando integralmente o loop paper maduro.

**Achados principais:**
- A arquitetura de actors já era genérica — nenhum ator, mensagem ou publisher novo foi necessário.
- O gap real estava nos **scaling factors** de risk (que caíam para defaults) e no **grafo de dependências** de settings (incompleto para drawdown_limit e para o novo slice).
- A validação de dependências risk→strategy e execution→risk foi corrigida de "all required" para "at least one", permitindo configurações com um único slice ativo.

---

## 2. Design dos Contracts

### Risk Scaling Factors (novo)

| Evaluator | Factor | squeeze_breakout_entry | Rationale |
|-----------|--------|----------------------|-----------|
| Position Exposure | Confidence multiplier | 0.93 | Momentum strategy — between counter-trend (0.90) and pro-trend (0.95) |
| Drawdown Limit | Confidence multiplier | 0.90 | Between counter-trend (0.85) and pro-trend (0.92) |
| Drawdown Limit | Stop distance multiplier | 1.05 | Slightly wider ceiling for breakout development |

### Dependency Graph (corrigido)

| Dependency | Before S290 | After S290 |
|------------|-------------|------------|
| `knownRiskFamilies` | `[position_exposure]` | `[position_exposure, drawdown_limit]` |
| `riskDependsOnStrategy[position_exposure]` | `[mean_reversion_entry]` | `[mean_reversion_entry, trend_following_entry, squeeze_breakout_entry]` |
| `riskDependsOnStrategy[drawdown_limit]` | _(missing)_ | `[mean_reversion_entry, trend_following_entry, squeeze_breakout_entry]` |
| `executionDependsOnRisk[paper_order]` | `[position_exposure]` | `[position_exposure, drawdown_limit]` |
| `executionDependsOnRisk[venue_market_order]` | `[position_exposure]` | `[position_exposure, drawdown_limit]` |
| Validation semantics | "all required" | "at least one" |

---

## 3. Arquivos Alterados

### Production Code

| File | Change |
|------|--------|
| `internal/application/risk/risk_scaling.go` | Added `squeeze_breakout_entry` to `positionExposureConfidenceFactor` (0.93), `drawdownConfidenceFactor` (0.90), `drawdownStopFactor` (1.05) |
| `internal/shared/settings/schema.go` | Added `drawdown_limit` to `knownRiskFamilies`; expanded `riskDependsOnStrategy` and `executionDependsOnRisk` to cover all families; changed validation to "at least one" semantics |

### Tests

| File | Tests Added |
|------|-------------|
| `internal/application/risk/risk_scaling_test.go` | 8 new tests: `TestPositionExposure_SqueezeBreakoutConfidence`, `TestDrawdown_SqueezeBreakoutConfidence`, `TestDrawdown_SqueezeBreakoutStopFactor`, `TestPositionExposure_SqueezeBreakoutCombinedHighSeverity`, `TestDrawdown_SqueezeBreakoutCombinedHighSeverity`, `TestPositionExposure_SqueezeBreakoutFlat`, `TestPaperOrder_SqueezeBreakoutApproved`, `TestPaperOrder_SqueezeBreakoutRejected` |
| `internal/shared/settings/settings_test.go` | 4 new tests: `TestValidatePipelineAcceptsSqueezeBreakoutFullChain`, `TestValidatePipelineAcceptsDualRiskWithSingleStrategy`, `TestValidatePipelineRejectsRiskWithoutAnyStrategy`, `TestValidatePipelineAcceptsDrawdownLimit` |

### Documentation

| File | Purpose |
|------|---------|
| `docs/architecture/squeeze-path-risk-and-execution-contracts.md` | Risk and execution contract specification |
| `docs/architecture/squeeze-path-paper-execution-integration.md` | Paper execution integration architecture |
| `docs/stages/stage-s290-risk-and-execution-contract-for-squeeze-path-report.md` | This report |

---

## 4. Testes e Validações

### Unit Tests (12 novos, todos passando)

**Risk scaling:**
- Position exposure confidence for squeeze_breakout_entry = 0.7905 (0.85 x 0.93)
- Drawdown confidence for squeeze_breakout_entry = 0.7650 (0.85 x 0.90)
- Drawdown stop factor = 0.0315 (0.03 x 1.05)
- Combined high-severity position exposure: confidence=0.8370, limit=0.0230, size=0.0207, approved
- Combined high-severity drawdown: confidence=0.8100, stop base=0.0315, max drawdown=0.0575, approved
- Flat squeeze breakout: approved, confidence=1.0

**End-to-end risk→execution:**
- Approved squeeze breakout → paper order buy with correct quantity and causal trace
- Rejected squeeze breakout (zero confidence) → paper order none with quantity 0

**Settings validation:**
- Full squeeze breakout chain (candle→bollinger→bollinger_squeeze→squeeze_breakout_entry→position_exposure→paper_order): valid
- Dual risk (position_exposure + drawdown_limit) with single strategy: valid
- Risk without any strategy: rejected
- drawdown_limit as known risk family: accepted

### Regression

All existing tests in `internal/application/risk`, `internal/shared/settings`, and `internal/application/execution` continue to pass.

---

## 5. Limites Explícitos

### Dentro do escopo (fechado)
- Scaling factors explícitos para squeeze_breakout_entry em ambos os evaluators de risk
- Grafo de dependências corrigido para cobrir todos os risk families e strategy families
- Semântica de validação de dependências corrigida para "at least one"
- Prova end-to-end de risk→execution em paper mode para o slice de squeeze breakout
- drawdown_limit registrado como known risk family

### Fora do escopo (não aberto)
- Venue real (OMS/router/portfolio) — guard rail respeitado
- Refatoração da camada de risk — não necessária (design genérico já acomoda)
- Codegen integration para squeeze_breakout_entry nos layers de risk/execution
- Multi-strategy aggregation (risk avalia cada estratégia independentemente)
- Garantias operacionais de restart/recovery para o slice de squeeze breakout
- Ajuste fino dos scaling factors (os valores escolhidos são semanticamente coerentes mas podem ser recalibrados com dados reais)

---

## 6. Preparação Recomendada para S291

### Opções estratégicas

1. **Codegen reentry para risk/execution** — Integrar squeeze_breakout_entry, drawdown_limit, e os demais families ao manifesto codegen (`integrated.yaml`). Isso fecha o gap entre artifacts manuais e gerados para os layers abaixo de strategy.

2. **Operational proof para squeeze breakout** — Smoke test end-to-end do slice completo (bollinger→squeeze→risk→paper) em processo real com NATS embeddado, validando restart/recovery e KV monotonicity.

3. **Signal evolution continuation** — Prosseguir com novos signal families ou decision families que alimentem os strategies existentes, ampliando a cobertura de sinais sem alterar os contracts de risk/execution.

4. **Scaling factor calibration** — Com dados de backtesting ou paper trading, recalibrar os fatores de confidence, stop, e position limit para squeeze_breakout_entry.

### Recomendação

A opção mais valiosa é a **operational proof** (opção 2), pois fecha o loop completo do slice em condições reais de runtime sem abrir novo escopo de codegen. A alternativa de codegen reentry (opção 1) é igualmente válida se o foco for consolidação de artifacts.
