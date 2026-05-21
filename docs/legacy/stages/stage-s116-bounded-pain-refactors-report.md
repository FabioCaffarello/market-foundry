# Stage S116 — Bounded Pain Refactors After Live Pipeline

> Refatorações pequenas, cirúrgicas e de alto valor, estritamente justificadas pelos achados do S115.

## Resumo Executivo

O S116 aplicou 4 micro-refactors delimitados por dor concreta observada durante a validação operacional do S115. Todos os refactors endereçam a fricção F4 (nomes stale de "consumer"/"validator" da quality-service).

O refactor de maior impacto (R1) elimina ~260 falsos positivos do drift-detect que ocorriam em cada execução do quality gate. Os demais (R2-R4) são correções cirúrgicas em pontos isolados.

7 itens foram explicitamente avaliados e adiados com justificativa documentada.

**Resultado:** Quality gate mais limpo, zero ruído de falsos positivos, zero abstração nova introduzida.

## Base de Decisão

Todos os refactors partem da [matriz de fricções do S115](../architecture/live-pipeline-frictions-and-structural-findings.md):

| Fricção S115 | Severidade | Ação S116 |
|--------------|-----------|-----------|
| F4 — Nomes stale "consumer"/"validator" | Low | **4 refactors aplicados** (R1-R4) |
| F6 — Parser heurístico do raccoon-cli | Medium | Adiado (D1) — sem segundo falso positivo |
| F7 — Sem soak test | Low | Adiado (D4) — requer infra dedicada |
| F8 — Sem golden-file tests | Low | Adiado (D5) — concern de feature |

## Refactors Executados

### R1: Drift-Detect False Positive Suppression

- **Arquivo:** `tools/raccoon-cli/src/analyzers/drift_detect.rs`
- **Dor:** ~260 warnings por execução do quality gate — "consumer" flagged em código NATS legítimo
- **Fix:** Scan de "consumer" restrito a `cmd/` e `deploy/`; `internal/` só scanneia "emulator" e "validator"
- **Impacto:** Elimina toda a classe de falsos positivos para NATS consumer patterns

### R2: Rename `validatorRecord` → `projectionRecord`

- **Arquivo:** `internal/application/runtimecontracts/runtime_test.go`
- **Dor:** Nome de variável referenciando serviço extinto
- **Fix:** Rename + atualização da mensagem de erro

### R3: AGENTS.md Prohibited Patterns

- **Arquivo:** `AGENTS.md`
- **Dor:** Texto ambíguo que parecia proibir "consumer" como conceito NATS
- **Fix:** Texto clarificado para "Old quality-service binaries (validator, consumer, emulator)"

### R4: Test Fixture Modernization

- **Arquivo:** `tools/raccoon-cli/src/analyzers/topology/compose.rs`
- **Dor:** Fixture de teste usando `quality-service/consumer:dev` — nome extinto
- **Fix:** Alterado para `market-foundry/ingest:dev`

## Arquivos Alterados

| Arquivo | Tipo de Mudança |
|---------|----------------|
| `tools/raccoon-cli/src/analyzers/drift_detect.rs` | Lógica de scan reestruturada (R1) |
| `internal/application/runtimecontracts/runtime_test.go` | Variable rename (R2) |
| `AGENTS.md` | Texto clarificado (R3) |
| `tools/raccoon-cli/src/analyzers/topology/compose.rs` | Test fixture atualizada (R4) |
| `docs/architecture/bounded-pain-refactors-after-live-pipeline.md` | Novo — documentação dos refactors |
| `docs/architecture/refactors-deferred-after-live-pipeline.md` | Novo — documentação dos itens adiados |
| `docs/stages/stage-s116-bounded-pain-refactors-report.md` | Novo — este relatório |

## Ganhos Estruturais Obtidos

1. **Quality gate sem ruído** — drift-detect não produz mais falsos positivos para NATS consumer patterns
2. **Teste legível** — `projectionRecord` comunica o propósito real do teste
3. **Onboarding mais claro** — AGENTS.md distingue entre proibição de binários antigos e uso legítimo de "consumer"
4. **Fixtures refletem realidade** — testes do raccoon-cli usam nomes do market-foundry, não da quality-service

## Itens Adiados

7 itens foram avaliados e adiados com trigger explícito de revisão:

| ID | Item | Razão | Trigger |
|----|------|-------|---------|
| D1 | AST parsing no raccoon-cli | Over-engineering — heurística funciona após fix S115 | Segundo falso positivo |
| D2 | Cleanup de nomes em docs históricos | Sem valor operacional — são registros históricos | Novo doc com nome stale |
| D3 | Unificação de padrão de use cases | Horizontal — 20+ arquivos, sem dor comprovada | Confusão ao adicionar domínio |
| D4 | Soak test | Requer infra dedicada | Multi-symbol ou live trading |
| D5 | Golden-file tests | Concern de feature, não de arquitetura | Nova família de sinal |
| D6 | Hardening de scripts | Scripts funcionam, frequência baixa | CI/CD pipeline |
| D7 | Parametrização de configs | Sem bug causado, complexidade desnecessária | Segundo environment |

Detalhes completos em [refactors-deferred-after-live-pipeline.md](../architecture/refactors-deferred-after-live-pipeline.md).

## Validação

| Check | Resultado |
|-------|-----------|
| `cargo test` (97 tests) | PASS — 97/97 |
| `go test ./internal/application/runtimecontracts/...` | PASS |
| Refactors com base em dor observada | OK — todos linkados a F4 do S115 |
| Refatoração permanece pequena e focada | OK — 4 mudanças cirúrgicas |
| Abstração excessiva evitada | OK — zero abstração nova |
| Monorepo melhor preparado sem onda ampla | OK — quality gate mais limpo |

## Critérios de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Refactors executados têm base explícita em dor observada | OK — todos F4 |
| Refatoração permanece pequena e focada | OK — 4 arquivos de código alterados |
| Base melhora em pontos de atrito reais | OK — ~260 falsos positivos eliminados |
| Abstração excessiva evitada | OK — nenhuma abstração introduzida |
| Monorepo preparado para evolução sem onda ampla | OK — sem mudança horizontal |

## Preparação Recomendada para S117

O S116 fecha o ciclo de refactors guiados por dor do S115. As opções naturais para o próximo stage:

1. **Expansão controlada** — adicionar segundo símbolo ou segunda família de sinal, guiada pela estabilidade demonstrada nos S114-S116.

2. **Observabilidade operacional** — se o pipeline live rodar por tempo suficiente, validar estabilidade com métricas reais (goroutine count, NATS backlog, KV growth) antes de expandir.

3. **Golden-file tests** — se uma nova família de sinal for adicionada (D5), acompanhar com testes de corretude numérica.

**Recomendação:** O próximo passo natural é expansão controlada (opção 1), pois a base está operacionalmente limpa e os refactors de dor foram aplicados. A expansão para segundo símbolo validaria o playbook de expansão sem introduzir complexidade nova.
