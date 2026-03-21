# Stage S243 — Risk Breadth Expansion Report

## Resumo Executivo

O S243 adicionou o segundo evaluator/type ao domínio `risk`: **`drawdown_limit`**, completando o critério de breadth da charter para os três domínios (decision, strategy, risk). A implementação segue o padrão formulaico estabelecido em S241/S242 — evaluator puro na camada de aplicação, actor wrapper, registry NATS, pipeline writer/store — sem alterar o domínio compartilhado ou mensagens existentes.

## Breadth de Risk Aplicada

### Antes do S243
| Domínio | Types |
|---------|-------|
| decision | `rsi_oversold`, `ema_crossover` |
| strategy | `mean_reversion_entry`, `trend_following_entry` |
| **risk** | **`position_exposure`** (apenas 1) |

### Depois do S243
| Domínio | Types |
|---------|-------|
| decision | `rsi_oversold`, `ema_crossover` |
| strategy | `mean_reversion_entry`, `trend_following_entry` |
| **risk** | **`position_exposure`, `drawdown_limit`** (2 — breadth atingida) |

### Semântica dos Dois Types

| Aspecto | `position_exposure` | `drawdown_limit` |
|---------|-------------------|-----------------|
| Pergunta | "Quanto capital alocar?" | "Quanto de perda é aceitável?" |
| Foco | Dimensionamento de posição | Distância de stop-loss |
| Constraints | `MaxPositionSize`, `MaxExposure` | `StopDistance`, `MaxExposure` |
| Mapeamento confidence | Linear: confidence × max_pct | Inverso: menor confiança → stop mais apertado |
| Fator de desconto | 0.95 | 0.90 |

## Arquivos Alterados

### Criados (novos)
| Arquivo | Propósito |
|---------|-----------|
| `internal/application/risk/drawdown_limit_evaluator.go` | Lógica pura de avaliação de drawdown |
| `internal/application/risk/drawdown_limit_evaluator_test.go` | 18 testes unitários |
| `internal/actors/scopes/derive/drawdown_limit_evaluator_actor.go` | Actor wrapper |
| `internal/actors/scopes/derive/drawdown_limit_evaluator_actor_test.go` | 4 testes de actor |
| `codegen/families/drawdown_limit.yaml` | Spec codegen para writer pipeline |
| `docs/architecture/risk-breadth-expansion.md` | Decisões arquiteturais de breadth |
| `docs/architecture/risk-type-02-semantics-consistency-and-boundaries.md` | Semântica e limites do segundo type |
| `docs/stages/stage-s243-risk-breadth-expansion-report.md` | Este relatório |

### Modificados
| Arquivo | Mudança |
|---------|---------|
| `internal/adapters/nats/natsrisk/registry.go` | +2 specs (event + control) + 2 consumer helpers + 1 switch case |
| `internal/adapters/nats/natsrisk/publisher.go` | +1 case em `specForType()` |
| `internal/adapters/nats/natsrisk/kv_store.go` | +1 bucket constant |
| `internal/actors/scopes/derive/derive_supervisor.go` | +1 `RiskFamilyProcessor` entry |
| `internal/actors/scopes/store/store_supervisor.go` | +1 pipeline entry (projection + consumer) |
| `cmd/writer/pipeline.go` | +1 `writerPipeline` entry |

### Não Alterados (por design)
| Arquivo | Razão |
|---------|-------|
| `internal/domain/risk/risk.go` | `RiskAssessment` struct cobre todos os types |
| `internal/actors/scopes/derive/messages.go` | Contrato de mensagens estável |
| `internal/adapters/clickhouse/risk_reader.go` | Query genérica sobre `type` column |
| HTTP handlers | Genéricos sobre risk type |

## Ganhos Semânticos e de Consistência

1. **Breadth completa**: todos os três domínios agora têm ≥2 types, satisfazendo a charter.
2. **Complementaridade semântica**: position_exposure e drawdown_limit respondem perguntas ortogonais sobre risco.
3. **Consistência de padrão**: drawdown_limit segue exatamente o mesmo padrão estrutural que position_exposure — mesmo config, mesmas mensagens, mesmo fluxo.
4. **Rastreabilidade preservada**: decision severity/rationale fluem através de ambos os risk evaluators até ClickHouse.
5. **Independência**: os dois risk types operam de forma totalmente independente — sem agregação, sem dependência cruzada.

## Cobertura de Testes

| Camada | Testes | Status |
|--------|--------|--------|
| Evaluator (application) | 18 testes (long/short/flat, invalid, timestamp, validation, multi-symbol, ownership bleed, metadata, stop floor) | PASS |
| Actor (derive) | 4 testes (long approved, flat approved, unknown direction, fan-out) | PASS |
| Testes existentes (position_exposure) | 15 evaluator + 5 actor | PASS (regressão zero) |

## Limites e Trade-offs

### Limites Explícitos
- **Sem rejeição**: drawdown_limit apenas aprova ou modifica, nunca rejeita
- **Parâmetros fixos**: max_drawdown_pct (5%) e stop_distance_pct (3%) hardcoded
- **Sem volatilidade de mercado**: stop distance baseado apenas em confidence, não em ATR/vol
- **Sem agregação cross-type**: position_exposure e drawdown_limit não se comunicam
- **Sem correlação cross-symbol**: cada avaliação é independente por symbol

### Decisões Conscientes
| Decisão | Benefício | Custo |
|---------|-----------|-------|
| Stop floor de 0.5% | Previne micro-stops irreais | Pode ser largo demais para certos instrumentos |
| Desconto 0.90 (vs 0.95) | Reflete incerteza maior em estimativas de drawdown | Desconto arbitrário |
| Sem rejection | Simplifica lógica de disposição | Não bloqueia trades com risco extremo |

## Preparação Recomendada para S244

O S244 deve ser o **gate de integração e breadth** — validando que os três domínios com breadth funcionam corretamente end-to-end. Recomendações:

1. **Validação end-to-end**: rodar actor_chain_integration_test com ambos risk types ativos.
2. **Verificação de CI**: confirmar que todos os testes passam em CI remoto.
3. **Verificação de pipeline**: confirmar que writer e store materializam ambos risk types corretamente.
4. **Gate checklist**:
   - decision: 2 types (rsi_oversold, ema_crossover) ✓
   - strategy: 2 types (mean_reversion_entry, trend_following_entry) ✓
   - risk: 2 types (position_exposure, drawdown_limit) ✓
   - Todos os testes passam ✓
   - Nenhuma regressão ✓
5. **Documentação de encerramento**: relatório de gate confirmando breadth completa.
