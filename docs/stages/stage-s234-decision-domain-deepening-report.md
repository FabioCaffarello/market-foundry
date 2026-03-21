# Stage S234 — Decision Domain Deepening Report

## Resumo Executivo

S234 aprofundou o domínio de decisão adicionando **Severity** (classificação graduada de intensidade) e **Rationale** (explicação estruturada legível) ao modelo de decisão, além de enriquecer os metadados do evaluator com zonas RSI e percentual de distância do threshold. A evolução manteve escopo mínimo: nenhuma nova família, nenhum novo subsistema, nenhuma regra adicional de decisão.

## Aprofundamento Aplicado

### Campos Semânticos Adicionados

| Campo       | Tipo     | Valores                          | Propósito                                    |
|-------------|----------|----------------------------------|----------------------------------------------|
| `Severity`  | enum     | `none`, `low`, `moderate`, `high`| Classificação de intensidade da condição     |
| `Rationale` | string   | Sentença estruturada             | Explicação legível da avaliação              |

### Zonas de Severidade (RSI Oversold)

| Severidade | Faixa RSI  | Distância do Threshold |
|------------|------------|------------------------|
| `none`     | >= 30      | — (not triggered)      |
| `low`      | 20–30      | 0–10 pontos            |
| `moderate` | 10–20      | 10–20 pontos           |
| `high`     | < 10       | 20+ pontos             |

### Metadados Enriquecidos

| Chave          | Exemplo   | Descrição                                |
|----------------|-----------|------------------------------------------|
| `threshold`    | `"30.0"`  | Threshold utilizado (existente)          |
| `rsi_zone`     | `"low"`   | Zona de severidade (novo)                |
| `distance_pct` | `"16.7"`  | Distância percentual do threshold (novo) |

## Arquivos Alterados

### Domínio
| Arquivo | Alteração |
|---------|-----------|
| `internal/domain/decision/decision.go` | Adicionados `Severity` type+consts, campos `Severity` e `Rationale` na struct, validação de Severity |
| `internal/domain/decision/decision_test.go` | Testes para validação de severidade, todos os valores do enum, severidade vazia permitida |

### Aplicação
| Arquivo | Alteração |
|---------|-----------|
| `internal/application/decision/rsi_oversold_evaluator.go` | Classificação de severidade por zonas, geração de rationale, metadados enriquecidos |
| `internal/application/decision/rsi_oversold_evaluator_test.go` | 10 novos testes: severidade por zona, monotonicity, rationale triggered/not_triggered, metadados rsi_zone e distance_pct |

### Adaptadores
| Arquivo | Alteração |
|---------|-----------|
| `internal/adapters/clickhouse/decision_reader.go` | SELECT atualizado com severity/rationale, Scan com novos campos |
| `internal/adapters/clickhouse/decision_reader_test.go` | Colunas esperadas atualizadas de 10 para 12 |
| `internal/adapters/clickhouse/writerpipeline/support.go` | `mapDecisionRow` emite severity e rationale |
| `internal/adapters/clickhouse/writerpipeline/support_test.go` | Column count 14→16, novos assertions |

### Infraestrutura
| Arquivo | Alteração |
|---------|-----------|
| `deploy/migrations/007_add_decision_severity_rationale.sql` | ALTER TABLE ADD COLUMN severity, rationale |
| `cmd/writer/pipeline.go` | INSERT SQL atualizado |
| `codegen/families/rsi_oversold.yaml` | Colunas atualizadas |
| `codegen/golden-snapshots/rsi_oversold/pipeline_entry.go.golden` | Golden snapshot atualizado |
| `codegen/render_test.go` | Spec inline atualizada |

### Testes de Integração
| Arquivo | Alteração |
|---------|-----------|
| `internal/actors/scopes/store/decision_projection_actor_test.go` | `validDecision()` com severity/rationale |
| `internal/interfaces/http/handlers/decision_test.go` | Decision struct com severity/rationale |
| `internal/interfaces/http/handlers/analytical_test.go` | Decision struct com severity/rationale |
| `internal/interfaces/http/routes/decision_test.go` | Decision struct com severity/rationale |
| `internal/application/decisionclient/get_latest_decision_test.go` | Decision struct com severity/rationale |

### Documentação
| Arquivo | Conteúdo |
|---------|----------|
| `docs/architecture/decision-domain-deepening.md` | Visão geral das mudanças, impacto downstream, precedentes |
| `docs/architecture/decision-semantics-thresholds-and-rationale.md` | Modelo semântico, zonas, formato de rationale, limites |

## Ganhos Semânticos Obtidos

1. **Decisão auto-explicativa**: O campo `rationale` permite que qualquer consumidor (humano ou máquina) entenda por que uma decisão foi tomada, sem consultar documentação.
2. **Graduação de intensidade**: `severity` permite que consumers downstream distingam entre um RSI de 29 (levemente oversold) e um RSI de 5 (extremamente oversold), sem precisar reinterpretar o valor bruto.
3. **Observabilidade enriquecida**: `rsi_zone` e `distance_pct` nos metadados fornecem diagnóstico imediato em dashboards e logs.
4. **Consistência arquitetural**: O pattern `Rationale` é compartilhado com `risk.RiskAssessment`, criando coerência entre domínios.

## Limites e Trade-offs

### Decisões Tomadas
- **Zonas fixas de 10 pontos** em vez de percentis dinâmicos: prioriza reprodutibilidade e debugabilidade sobre adaptatividade.
- **Severity não é forwarded para strategy**: strategy consome outcome/confidence; severity é para observabilidade e consumers externos.
- **Rationale é para leitura, não para parsing downstream**: resolvers devem usar campos tipados, não parsear strings.

### Não-Objetivos Explícitos
- Não foi criada nenhuma nova família de decisão.
- Não foi criado um motor genérico de regras.
- Não foram adicionadas múltiplas regras ao espaço de decisão.
- Severity não implica recomendação de ação (isso pertence a strategy/risk).
- Não há classificação de "quão não-oversold" para decisões not_triggered.

### Dívidas Técnicas
- Nenhuma introduzida. As mudanças são backward-compatible via defaults no ClickHouse e zero-values no Go.

## Evidência de Testes

```
ok  internal/domain/decision         0.252s
ok  internal/application/decision    1.351s
ok  internal/adapters/clickhouse     0.158s
ok  internal/adapters/clickhouse/writerpipeline  0.271s
ok  internal/actors/scopes/store     1.146s
ok  internal/interfaces/http/handlers  0.515s
ok  internal/interfaces/http/routes  0.618s
ok  internal/application/decisionclient  1.315s
ok  codegen                          0.733s
```

Todos os testes passam. Nenhum teste existente foi removido; testes foram adicionados e atualizados.

## Preparação Recomendada para S235

O S235 deve alinhar o domínio `strategy` com a semântica enriquecida de `decision`:

1. **Strategy pode consumir severity**: O `MeanReversionEntryResolver` pode opcionalmente usar severity para ajustar confidence da estratégia (e.g., severidade high → confidence boost).
2. **Strategy rationale**: Seguindo o pattern, strategy pode adotar seu próprio campo `Rationale` (atualmente não possui).
3. **Analytical queries por severity**: Com a coluna severity no ClickHouse, queries analíticas podem filtrar por `severity = 'high'` para encontrar decisões de alta intensidade.
4. **Dashboard observability**: severity e rationale podem alimentar painéis de monitoramento para visualização de distribuição de decisões por zona.

### Pré-condições para S235
- Migração 007 aplicada no ClickHouse.
- Deploy com nova versão do evaluator ativo e produzindo severity/rationale.
- Pipeline derive → store → read validado com os novos campos em ambiente staging.
