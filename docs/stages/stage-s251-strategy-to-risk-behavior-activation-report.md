# Stage S251 — Strategy-to-Risk Behavior Activation Report

> Date: 2026-03-21
> Wave: Behavioral Wave 1 (S249–S253)
> Predecessor: S250 (Decision-to-Strategy Behavior Activation)

## 1. Resumo Executivo

S251 ativa comportamento real na ligação `strategy → risk`, fazendo com que ambos os
avaliadores de risco (`position_exposure` e `drawdown_limit`) passem a responder
semanticamente ao tipo de estratégia e à severidade da decisão.

Antes do S251, risk aplicava multiplicadores fixos independentes do contexto
(×0.95 para position, ×0.90 para drawdown). Agora, risk diferencia:
- **Counter-trend (mean_reversion)** → avaliação mais conservadora (confidence ×0.90/×0.85,
  stop mais apertado)
- **Pro-trend (trend_following)** → avaliação menos conservadora (confidence ×0.95/×0.92,
  stop mais largo)
- **Severidade alta** → limites de posição e tolerância a drawdown +15%
- **Severidade baixa** → limites de posição e tolerância a drawdown −20%

Zero novos tipos de risco, zero novas mensagens, zero novos atores. Apenas ativação
comportamental sobre dados que já fluíam pelo pipeline.

## 2. Comportamento `strategy → risk` Ativado

### Position Exposure Evaluator

| Dimensão | Antes (S250) | Depois (S251) |
|---|---|---|
| Risk confidence multiplier | Fixo ×0.95 | mean_reversion→×0.90, trend_following→×0.95, default→×0.92 |
| Position limit | Fixo `maxPositionPct` | Ajustado por severity: high→×1.15, low→×0.80 |
| Strategy type in metadata | Ausente | Presente (`strategy_type`) |
| Effective params in output | Ausente | `effective_max_position_pct`, `confidence_factor`, `severity_limit_factor` |
| Rationale | `"Position size X within limits; decision severity Y"` | `"Position size X within limits; mean_reversion_entry (confidence ×0.90); decision severity Y (limit ×0.80)"` |

### Drawdown Limit Evaluator

| Dimensão | Antes (S250) | Depois (S251) |
|---|---|---|
| Risk confidence multiplier | Fixo ×0.90 | mean_reversion→×0.85, trend_following→×0.92, default→×0.88 |
| Stop distance base | Fixo `stopDistancePct` | Ajustado por strategy type: mean_reversion→×0.85, trend_following→×1.15 |
| Max drawdown tolerance | Fixo `maxDrawdownPct` | Ajustado por severity: high→×1.15, low→×0.80 |
| Strategy type in metadata | Ausente | Presente (`strategy_type`) |
| Effective params in output | Ausente | `effective_stop_distance_pct`, `effective_max_drawdown_pct`, etc. |
| Rationale | `"Stop distance X within limits for Y; decision severity Z"` | `"Stop distance X within limits for Y; trend_following_entry (confidence ×0.92, stop ×1.15); decision severity Z (tolerance ×1.15)"` |

## 3. Arquivos Alterados

### Novos

| Arquivo | Propósito |
|---|---|
| `internal/application/risk/risk_scaling.go` | Mapas de scaling e funções puras para ajuste por strategy type e severity |
| `internal/application/risk/risk_scaling_test.go` | 11 testes: confidence por tipo, position limit por severity, stop base por tipo, drawdown tolerance por severity, metadata, rationale, cenários combinados |
| `docs/architecture/strategy-to-risk-behavior-activation.md` | Especificação comportamental S251 |
| `docs/architecture/strategy-context-consumption-by-risk.md` | Contrato de consumo de contexto strategy→risk |
| `docs/stages/stage-s251-strategy-to-risk-behavior-activation-report.md` | Este relatório |

### Modificados

| Arquivo | Mudança |
|---|---|
| `internal/application/risk/position_exposure_evaluator.go` | Strategy-type confidence factor, severity position limit, metadata enriquecida, rationale contextualizado, effective params |
| `internal/application/risk/drawdown_limit_evaluator.go` | Strategy-type confidence + stop factor, severity drawdown tolerance, metadata enriquecida, rationale contextualizado, effective params |
| `internal/application/risk/position_exposure_evaluator_test.go` | Assertions atualizadas para novos valores, novos testes de metadata/params |
| `internal/application/risk/drawdown_limit_evaluator_test.go` | Assertions atualizadas para novos valores, novos testes de metadata/params |
| `internal/domain/risk/risk.go` | Doc comment atualizado em StrategyInput |

### Não alterados (confirmação)

- `internal/actors/scopes/derive/risk_evaluator_actor.go` — sem mudanças no wiring
- `internal/actors/scopes/derive/messages.go` — sem novos campos de mensagem
- `internal/adapters/nats/natsrisk/*` — sem mudanças em publishers/stores
- `internal/adapters/clickhouse/risk_reader.go` — sem mudanças no read path

## 4. Cobertura de Testes

### Testes Unitários — Risk Scaling (risk_scaling_test.go)

| Teste | Subtestes | O que verifica |
|---|---|---|
| `TestPositionExposure_StrategyTypeConfidence` | 3 | mean_reversion ×0.90, trend_following ×0.95, unknown ×0.92 |
| `TestPositionExposure_SeverityAdjustsPositionLimit` | 5 | high/moderate/low/empty/none → effective_max_position_pct |
| `TestPositionExposure_StrategyTypeInMetadata` | 1 | strategy_type presente |
| `TestPositionExposure_RationaleIncludesStrategyType` | 1 | rationale contém tipo e fatores |
| `TestPositionExposure_CombinedStrategyAndSeverity` | 1 | mean_reversion+high vs trend_following+low end-to-end |
| `TestDrawdown_StrategyTypeConfidence` | 3 | mean_reversion ×0.85, trend_following ×0.92, unknown ×0.88 |
| `TestDrawdown_StrategyTypeAdjustsStopBase` | 3 | mean_reversion ×0.85, trend_following ×1.15, unknown ×1.00 |
| `TestDrawdown_SeverityAdjustsDrawdownTolerance` | 5 | high/moderate/low/empty/none → effective_max_drawdown_pct |
| `TestDrawdown_StrategyTypeInMetadata` | 1 | strategy_type presente |
| `TestDrawdown_RationaleIncludesStrategyType` | 1 | rationale contém tipo e fatores |
| `TestDrawdown_CombinedStrategyAndSeverity` | 1 | mean_reversion+high vs trend_following+low end-to-end |

### Testes Existentes Atualizados

Todos os testes de `position_exposure_evaluator_test.go` e `drawdown_limit_evaluator_test.go`
atualizados para refletir os novos valores de confidence e a presença de `strategy_type`
em metadata.

### Testes de Integração (actor_chain_integration_test.go)

Todos os 7 testes de integração passam sem alteração:
- `TestActorChain_Signal_To_Decision_To_Strategy_To_Risk`
- `TestActorChain_NotTriggered_FlowsThrough`
- `TestActorChain_EMACrossover_Bullish_Triggered`
- `TestActorChain_EMACrossover_Bearish_NotTriggered`
- `TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk`
- `TestActorChain_EMACrossover_TrendFollowingEntry_To_DrawdownLimitRisk`
- `TestActorChain_CorrelationID_PreservedEndToEnd`

## 5. Ganhos Semânticos e Operacionais

1. **Risk diferencia estratégias** — counter-trend recebe avaliação mais conservadora que
   pro-trend, refletindo o perfil de risco real de cada família.
2. **Severidade influencia limites** — sinais fortes justificam posições maiores e mais
   tolerância a drawdown; sinais fracos restringem.
3. **Outputs explicáveis** — rationale detalha exatamente quais fatores foram aplicados,
   incluindo os multiplicadores numéricos.
4. **Audit trail completo** — `strategy_type` em metadata + effective params permitem
   reconstruir exatamente como risk chegou à sua decisão.
5. **Sem aumento de complexidade topológica** — zero novos atores, mensagens ou streams.

## 6. Limites e Trade-offs

### O que S251 NÃO fez

- **Não abriu nova breadth** — risk continua com dois tipos: `position_exposure` e
  `drawdown_limit`.
- **Não criou policy engine** — fatores são mapas estáticos em código, não regras
  configuráveis.
- **Não introduziu rejection por tipo** — strategy type influencia scaling, nunca causa
  rejection direto.
- **Não alterou topologia de atores** — nenhum novo ator ou mensagem.
- **Não modificou o read path** — ClickHouse reader, KV store, HTTP handlers inalterados.

### Trade-offs aceitos

- Fatores de scaling são hardcoded como package-level maps. Se futuramente for necessário
  torná-los configuráveis, será necessário refatorar para injeção via construtor.
- Unknown strategy types recebem defaults neutros. Se um novo tipo exigir scaling
  específico, precisa ser adicionado aos mapas.

## 7. Invariantes Preservadas

| Invariante | Status |
|---|---|
| Domain isolation (DBI-9) | ✅ Zero imports cruzados |
| Pure application logic | ✅ Sem I/O nos evaluators |
| Single-writer per stream | ✅ Inalterado |
| Acyclic data flow | ✅ Inalterado |
| Envelope uniformity | ✅ Inalterado |
| Backward compatibility | ✅ Unknown type/severity → ×1.00 neutral |

## 8. Preparação Recomendada para S252

S252 deve validar o pipeline ponta a ponta por cenários explícitos:

1. **Cenário A**: RSI 10.0 (high severity) → mean_reversion_entry → position_exposure
   - Verificar: confidence ×0.90, position limit ×1.15
2. **Cenário B**: RSI 28.0 (low severity) → mean_reversion_entry → drawdown_limit
   - Verificar: confidence ×0.85, stop ×0.85, tolerance ×0.80
3. **Cenário C**: EMA bullish (moderate severity) → trend_following_entry → position_exposure
   - Verificar: confidence ×0.95, position limit ×1.00
4. **Cenário D**: EMA bullish (moderate severity) → trend_following_entry → drawdown_limit
   - Verificar: confidence ×0.92, stop ×1.15, tolerance ×1.00
5. **Cenário E**: Cross-chain com ambos risk types para o mesmo strategy
   - Verificar: isolamento entre position_exposure e drawdown_limit
6. **Cenário F**: Multi-symbol → confirmar que scaling não vaza entre símbolos

Esses cenários devem ser testes de integração no `actor_chain_integration_test.go`,
validando valores numéricos explícitos de confidence, position size, stop distance e
rationale em cada estágio.
