# Stage S129 — Triggered Refactors After CC-02

**Status:** Complete
**Predecessors:** S128 (Extensibility Friction Capture)
**Scope:** Execute only refactors with triggers definitively met by CC-02.

---

## 1. Resumo Executivo

O S129 analisou todas as fricções documentadas no S128 e aplicou **exatamente um refactor** — o único cuja condição de trigger foi inequivocamente atingida pela CC-02: a centralização da extração de Correlation ID no layer HTTP via middleware.

Nove fricções permaneceram conscientemente adiadas porque seus thresholds não foram atingidos (N=2 < N=3 para CF-08/CF-11, N=2 < N=5 para CF-12, etc.). Zero refactors foram executados por gosto ou estética.

A disciplina do stage foi mantida: nenhuma refatoração horizontal foi reaberta.

---

## 2. Refactors Executados

### R1: HTTP Correlation ID Middleware (CF-03)

**Trigger:** CF-03 — *"primeiro novo actor que copia o padrão `events.NewMetadata().WithCorrelationID(msg.CorrelationID)`"*. Atingido por `EMACrossoverSignalSamplerActor`. No layer HTTP, 12 handlers copiavam manualmente `requestctx.WithCorrelationID(r.Context(), r.Header.Get("X-Correlation-ID"))`.

**Implementação:**
- Novo middleware `webserver.CorrelationID` que extrai `X-Correlation-ID` do header e injeta no contexto do request.
- Middleware aplicado globalmente em `WebServer.buildHTTPServer()`.
- 12 ocorrências de extração manual removidas de 7 arquivos de handler.
- Helper privado `withCorrelationID` removido do `configctl.go`.
- Imports `requestctx` removidos dos handlers que não o utilizam mais.
- Testes do middleware adicionados; teste existente do configctl atualizado.

**Justificativa:** Cada novo handler HTTP precisava copiar a mesma linha. O middleware elimina esse boilerplate para sempre — qualquer handler futuro herda Correlation ID automaticamente.

**Risco:** Mínimo. O middleware executa exatamente a mesma lógica que era per-handler.

---

## 3. Arquivos Alterados

### Novos
| Arquivo | Propósito |
|---------|-----------|
| `internal/shared/webserver/middleware.go` | Middleware HTTP de Correlation ID |
| `internal/shared/webserver/middleware_test.go` | Testes do middleware |

### Modificados
| Arquivo | Mudança |
|---------|---------|
| `internal/shared/webserver/server.go` | Aplica middleware no `buildHTTPServer()` |
| `internal/shared/webserver/server_test.go` | Atualiza asserção de handler identity |
| `internal/interfaces/http/handlers/signal.go` | Remove extração manual + import |
| `internal/interfaces/http/handlers/decision.go` | Remove extração manual + import |
| `internal/interfaces/http/handlers/risk.go` | Remove extração manual + import |
| `internal/interfaces/http/handlers/strategy.go` | Remove extração manual + import |
| `internal/interfaces/http/handlers/execution.go` | Remove 2 extrações manuais + import |
| `internal/interfaces/http/handlers/execution_control.go` | Remove 2 extrações manuais + import |
| `internal/interfaces/http/handlers/evidence.go` | Remove 4 extrações manuais + import |
| `internal/interfaces/http/handlers/configctl.go` | Remove extração + helper + import |
| `internal/interfaces/http/handlers/configctl_test.go` | Simula middleware no contexto do teste |

### Documentação Nova
| Arquivo | Propósito |
|---------|-----------|
| `docs/architecture/triggered-refactors-after-cc-02.md` | Refactors executados com justificativa |
| `docs/architecture/refactors-still-deferred-after-cc-02.md` | Refactors adiados com triggers e thresholds |

**Total:** 2 arquivos novos de código, 11 arquivos modificados, 3 documentos.

---

## 4. Ganhos Estruturais

| Ganho | Evidência |
|-------|-----------|
| Zero boilerplate de Correlation ID por novo handler HTTP | Middleware global cobre automaticamente |
| Handlers mais limpos e focados em lógica de negócio | ~20 linhas de extração manual removidas |
| Ponto único de controle para propagação HTTP | Se a lógica de propagação mudar, um local |
| Extensibilidade validada | Próxima family/capability ganha Correlation ID grátis no HTTP |

---

## 5. Itens Adiados e Justificativa

| ID | Fricção | Threshold | Estado Atual | Adiado Até |
|----|---------|-----------|--------------|------------|
| CF-03 (actor) | Propagação manual em actors | N=3 families | N=2, 0 incidentes | CC-03 (bundled com CF-08) |
| CF-08 | Boilerplate de actor (97 LOC/family) | N=3 families | N=2, copy-paste ok | CC-03 |
| CF-11 | Switch proliferation em NATS registry | N=3 families | N=2, 0 erros de wiring | CC-03 |
| CF-12 | Pipeline boilerplate em store supervisor | N=5 families | N=2, declarativo | N=5 |
| CF-02 | Endpoint de symbols ativos | Route change ou N>5 symbols | N=2 | Oportunístico |
| CF-13 | Config per-family de algoritmo | A/B testing | Hardcoded correto | Demand-driven |
| D4 | Testes de composition root | Wiring error escape | 0 erros | Incident-driven |
| D5 | Validação de failure recovery | Produção | Paper-trading | Pré-produção |
| D6 | Soak testing infra | N>5 symbols ou 24h | N=2 | Scale-driven |

**Insight chave:** CF-08, CF-11 e CF-03 (actor) convergem no threshold N=3 families. CC-03 é o ponto natural de bundling (~5-7h estimadas).

---

## 6. Critérios de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Refactors executados têm trigger natural e justificativa explícita | **Pass** — CF-03 trigger atingido por CC-02 |
| Refatoração permanece pequena, localizada e de alto valor | **Pass** — 1 refactor, 13 arquivos, ~30 linhas net |
| Extensibilidade melhora em pontos de dor reais | **Pass** — handler HTTP friction eliminada |
| Abstração excessiva evitada | **Pass** — middleware simples, sem framework |
| Sistema mais robusto para próxima family sem reabrir refatoração ampla | **Pass** — zero mudanças horizontais |

---

## 7. Guard Rails — Verificação

| Guard Rail | Status |
|------------|--------|
| Não abrir nova refatoração horizontal | **Respeitado** |
| Não usar stage para "melhorar mais coisas" | **Respeitado** — apenas CF-03 HTTP |
| Não justificar refactor por gosto ou estética | **Respeitado** — trigger documentado |
| Não introduzir framework/padrão novo sem trigger | **Respeitado** — middleware é padrão HTTP standard |
| Documentar o que foi conscientemente mantido simples | **Respeitado** — ver `refactors-still-deferred-after-cc-02.md` |

---

## 8. Preparação para S130

### Se S130 for CC-03 (terceira signal family):

**Refactors recomendados para bundling com CC-03:**
1. **Generic `SignalSamplerActor[T]`** (CF-08) — elimina ~97 LOC/family. Effort: ~2h.
2. **Map-based NATS registry** (CF-11) — centraliza 4 touch points em 1. Effort: ~1-2h.
3. **Actor correlation ID injection** (CF-03 actor) — bundled com generic actor. Effort: ~1h.

**Total estimado:** ~5-7h, amortizado na entrega da CC-03.

**Recomendação:** Executar refactors **como parte da implementação** da CC-03, não como stage separado. O terceiro data point valida o padrão; a extração é natural e imediata.

### Se S130 for uma capability diferente:

Prosseguir sem bloqueio. O nível de fricção atual não bloqueia nenhuma capability. Revisitar refactors quando a terceira signal family for introduzida.

### Se S130 for architectural readiness review:

Avaliar se o escopo justifica um review formal ou se a maturidade atingida em S124/S128 já cobre. Recomendação: review leve focado em:
- Validação de que a middleware funciona end-to-end com a pipeline existente.
- Confirmação de que os thresholds de deferral continuam calibrados.
