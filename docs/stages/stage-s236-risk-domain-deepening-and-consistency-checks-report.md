# Stage S236 — Risk Domain Deepening and Consistency Checks

## Resumo Executivo

S236 aprofundou o domínio risk para que ele reflita a densidade semântica produzida por decision (S234) e strategy (S235). A severidade e rationale da decisão agora fluem ponta a ponta até o risk assessment, enriquecendo rastreabilidade, rationale de risco e observabilidade analítica — sem inflar regras ou abrir um subsistema complexo.

## Aprofundamento Aplicado

### 1. Domain: StrategyInput Enriquecido
`risk.StrategyInput` ganhou dois campos opcionais:
- `DecisionSeverity` — classifica a extremidade da condição que originou o assessment
- `DecisionRationale` — carrega a explicação semântica da decisão original

Ambos usam `omitempty` para compatibilidade retroativa com dados existentes.

### 2. Messages: Contexto de Decisão na Pipeline de Atores
- `strategyResolvedMessage` agora carrega `DecisionSeverity` e `DecisionRationale` do strategy resolver para o risk evaluator
- `riskAssessedMessage` agora carrega `DecisionSeverity` do risk evaluator para o execution evaluator
- O strategy resolver extrai esses valores do primeiro `DecisionInput` da estratégia

### 3. Application: Evaluator com Rationale Rico
`PositionExposureEvaluator.Evaluate()` agora:
- Aceita `decisionSeverity` e `decisionRationale` como parâmetros
- Gera rationale contextual: `"Position size 0.0170 within exposure limits; decision severity high"`
- Propaga contexto de decisão no `Metadata` do risk assessment para queries analíticas
- Preserva contexto de decisão no `StrategyInput` para rastreabilidade estruturada

### 4. Actor Layer: Propagação Consistente
- Strategy resolver extrai severity/rationale do `DecisionInput` e encaminha no `strategyResolvedMessage`
- Risk evaluator passa os novos campos para o evaluator e inclui severity no fan-out para execution

## Arquivos Alterados

| Arquivo | Tipo de Mudança |
|---------|----------------|
| `internal/domain/risk/risk.go` | `StrategyInput` enriquecido com `DecisionSeverity`, `DecisionRationale` |
| `internal/domain/risk/risk_test.go` | Testes de contexto de decisão no StrategyInput |
| `internal/application/risk/position_exposure_evaluator.go` | Nova assinatura, rationale rico, metadata de decisão |
| `internal/application/risk/position_exposure_evaluator_test.go` | 17 testes (5 novos para contexto de decisão) |
| `internal/actors/scopes/derive/messages.go` | Novos campos em `strategyResolvedMessage` e `riskAssessedMessage` |
| `internal/actors/scopes/derive/strategy_resolver_actor.go` | Encaminha severity/rationale no fan-out para risk |
| `internal/actors/scopes/derive/risk_evaluator_actor.go` | Passa contexto de decisão ao evaluator; inclui no fan-out |
| `internal/actors/scopes/derive/risk_evaluator_actor_test.go` | 5 testes (1 novo para fan-out com DecisionSeverity) |
| `internal/adapters/clickhouse/risk_reader_test.go` | Testes de round-trip JSON com e sem contexto de decisão |
| `internal/adapters/clickhouse/writerpipeline/support_test.go` | Testes de serialização JSON com contexto de decisão |
| `docs/architecture/risk-domain-deepening-and-consistency-checks.md` | Documento de arquitetura |
| `docs/architecture/decision-strategy-risk-consistency-model.md` | Modelo de consistência ponta a ponta |

## Ganhos Semânticos e de Consistência

1. **Rastreabilidade ponta a ponta**: Todo risk assessment agora carrega o severity e rationale da decisão que o originou, sem joins necessários.
2. **Rationale contextual**: O rationale de risco agora referencia a severidade da decisão, tornando os outputs mais explicáveis.
3. **Observabilidade analítica**: `Metadata["decision_severity"]` e `Metadata["decision_rationale"]` permitem queries analíticas ricas no ClickHouse.
4. **Compatibilidade retroativa**: Campos `omitempty` garantem que dados existentes desserializam sem problemas.
5. **Isolamento de domínio preservado**: Nenhum import cruzado entre domínios. Valores cruzam fronteiras como primitivos.

## Limites e Trade-offs

| Decisão | Justificativa |
|---------|--------------|
| Severity como rastreabilidade, não lógica | Mantém risk determinístico e testável; evita acoplar regras de risco a heurísticas de decisão |
| Sem path de rejeição baseado em severity | Rejeição por severity é decisão de política que deve ser deliberada, não efeito colateral |
| Metadata duplicada (StrategyInput + Metadata) | Habilita acesso estruturado (StrategyInput) e otimizado para query (Metadata) |
| Sem motor complexo de risco | Escopo controlado conforme charter — aprofundamento de consistência, não expansão de sistema |

## Não-Objetivos Explícitos

- Não abrir subsistema amplo de risk management
- Não inflar regras de risco baseadas em severity
- Não criar risk temporal (e.g., "muitas decisões high-severity em 5 minutos")
- Não criar risk agregado cross-symbol
- Não alterar position sizing ou disposition baseado em severity

## Evidência de Testes

```
ok  internal/domain/risk              — 18 testes (2 novos)
ok  internal/application/risk         — 17 testes (5 novos)
ok  internal/actors/scopes/derive     — todos passam (1 teste novo de fan-out)
ok  internal/actors/scopes/store      — todos passam
ok  internal/adapters/clickhouse      — todos passam (2 testes novos)
ok  internal/adapters/clickhouse/writerpipeline — todos passam (1 teste novo)
```

Full suite: **0 falhas**.

## Preparação Recomendada para S237

O S237 pode focar em **hardening leve e integração**, considerando:

1. **Risk → Execution alignment**: O `riskAssessedMessage` agora carrega `DecisionSeverity` — o execution evaluator pode usar isso para logging/metadata sem mudança de lógica.
2. **Analytical views**: Criar views ClickHouse que unem decision, strategy e risk por correlation_id para dashboards de observabilidade ponta a ponta.
3. **End-to-end integration test**: Validar o fluxo completo signal→decision→strategy→risk→execution com severity "high" e verificar rastreabilidade.
4. **Documentation consolidation**: Unificar os documentos de arquitetura S234/S235/S236 em um diagrama de fluxo único.
5. **Clean pass gate**: Rodar CI remota e marcar milestone de charter completion.

## Conclusão

S236 fecha a trilha de aprofundamento de domínio da charter. Os três domínios derive — decision, strategy, risk — agora produzem outputs semanticamente ricos, rastreáveis ponta a ponta, e compatíveis retroativamente. O escopo permaneceu controlado: sem motor complexo de risco, sem regras infladas, sem opacidade. A base está pronta para hardening e integração no S237.
