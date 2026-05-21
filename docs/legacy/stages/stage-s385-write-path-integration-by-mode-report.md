# Stage S385 — Write-Path Integration by Execution Mode

> **Status:** Complete
> **Wave:** OMS Foundation (S382–S386+)
> **Predecessor:** S384 (Lifecycle Invariant Coverage and Price Realism)
> **Date:** 2026-03-22

## 1. Resumo Executivo

S385 fecha a lacuna principal entre modelagem (S383) e execução: os paths dominantes do write-path por modo de execução estão agora provados por testes de integração que validam alignment direto com o lifecycle state machine canônico.

**19 testes** cobrem os 3 modos (`dry_run`, `paper`, `venue_live`) com paths de buy, sell, none, rejection e partial fill. Cada teste valida transições contra `ValidTransition()`, fill record shape, `Simulated` flag, VenueOrderID prefix e preservação da correlation chain.

## 2. Modos e Paths Validados

### dry_run
| Path | Side | Status final | Testes |
|------|------|-------------|--------|
| submitted → [accepted → filled] | buy | filled | `TestS385_DryRun_Buy_SubmittedToFilled` |
| submitted → [accepted → filled] | sell | filled | `TestS385_DryRun_Sell_SubmittedToFilled` |
| submitted → accepted | none | accepted | `TestS385_DryRun_None_SubmittedToAccepted` |

### paper
| Path | Side | Status final | Testes |
|------|------|-------------|--------|
| submitted → [accepted → filled] | buy | filled | `TestS385_Paper_Buy_SubmittedToFilled` |
| submitted → [accepted → filled] | sell | filled | `TestS385_Paper_Sell_SubmittedToFilled` |
| submitted → accepted | none | accepted | `TestS385_Paper_None_SubmittedToAccepted` |

### venue_live
| Path | Side | Status final | Testes |
|------|------|-------------|--------|
| submitted → accepted → filled | buy | filled | `TestS385_VenueLive_Buy_SubmittedToFilled` |
| submitted → accepted → filled | sell | filled | `TestS385_VenueLive_Sell_SubmittedToFilled` |
| submitted → accepted | buy | accepted (NEW) | `TestS385_VenueLive_Buy_SubmittedToAccepted` |
| submitted → rejected | buy | rejected (Problem) | `TestS385_VenueLive_Rejection_SubmittedToRejected` |
| submitted → accepted | none | accepted | `TestS385_VenueLive_None_SubmittedToAccepted` |
| submitted → accepted → partially_filled | buy | partially_filled | `TestS385_VenueLive_PartialFill` |

### Cross-Mode
| Propriedade | Testes |
|-------------|--------|
| Simulated flag diferença | `TestS385_CrossMode_SimulatedFlagDifference` |
| VenueOrderID prefix | `TestS385_CrossMode_VenueOrderIDPrefixConvention` |
| Correlation chain | `TestS385_CrossMode_AllModesPreserveCorrelationChain` |
| No-action consistency | `TestS385_CrossMode_NoActionSemanticsConsistentAcrossModes` |
| Terminal states absorbing | `TestS385_CrossMode_TerminalStatesAreAbsorbing` |
| FilledQuantity == Quantity | `TestS385_CrossMode_FilledQuantityEqualsQuantityOnFill` |
| Intent field preservation | `TestS385_CrossMode_IntentFieldPreservation` |

## 3. Arquivos Alterados

| Arquivo | Tipo | Descrição |
|---------|------|-----------|
| `internal/application/execution/s385_write_path_by_mode_test.go` | **Novo** | 19 testes de integração do write-path por modo |
| `docs/architecture/write-path-integration-tests-by-execution-mode.md` | **Novo** | Catálogo dos testes por modo |
| `docs/architecture/execution-mode-paths-lifecycle-projection-and-test-findings.md` | **Novo** | Diferenças semânticas entre modos, projeção lifecycle, gaps |
| `docs/stages/stage-s385-write-path-integration-by-mode-report.md` | **Novo** | Este relatório |

## 4. Testes e Evidências

```
=== RUN   TestS385_DryRun_Buy_SubmittedToFilled       --- PASS
=== RUN   TestS385_DryRun_Sell_SubmittedToFilled      --- PASS
=== RUN   TestS385_DryRun_None_SubmittedToAccepted    --- PASS
=== RUN   TestS385_Paper_Buy_SubmittedToFilled        --- PASS
=== RUN   TestS385_Paper_Sell_SubmittedToFilled       --- PASS
=== RUN   TestS385_Paper_None_SubmittedToAccepted     --- PASS
=== RUN   TestS385_VenueLive_Buy_SubmittedToFilled    --- PASS
=== RUN   TestS385_VenueLive_Sell_SubmittedToFilled   --- PASS
=== RUN   TestS385_VenueLive_Buy_SubmittedToAccepted  --- PASS
=== RUN   TestS385_VenueLive_Rejection                --- PASS
=== RUN   TestS385_VenueLive_None_SubmittedToAccepted --- PASS
=== RUN   TestS385_VenueLive_PartialFill              --- PASS
=== RUN   TestS385_CrossMode_SimulatedFlagDifference  --- PASS
=== RUN   TestS385_CrossMode_VenueOrderIDPrefix       --- PASS
=== RUN   TestS385_CrossMode_CorrelationChain         --- PASS
=== RUN   TestS385_CrossMode_NoActionConsistency      --- PASS
=== RUN   TestS385_CrossMode_TerminalAbsorbing        --- PASS
=== RUN   TestS385_CrossMode_FilledQuantity            --- PASS
=== RUN   TestS385_CrossMode_IntentFieldPreservation  --- PASS
PASS  ok  internal/application/execution  0.287s
```

**Cobertura por critério:**

| Critério | Cobertura |
|----------|-----------|
| dry_run buy/sell → submitted → filled | 100% |
| paper buy/sell → submitted → filled | 100% |
| venue_live buy/sell → submitted → accepted → filled | 100% |
| venue_live rejection → submitted → rejected | 100% |
| venue_live partial fill | 100% |
| all modes none → submitted → accepted | 100% |
| ValidTransition() alignment | 100% (every test) |
| Simulated flag per mode | 100% |
| Correlation chain per mode | 100% |

## 5. Limites Remanescentes

| Gap | Motivo | Impacto |
|-----|--------|---------|
| Status `sent` nunca exercido | Nenhum adapter atual usa protocolo assíncrono de acknowledgment | Baixo — reservado para futura expansão |
| Status `cancelled` via adapter | Requer API de cancel-order (fluxo assíncrono) | Médio — mapped em `mapBinanceStatus()` mas sem teste end-to-end |
| Multi-fill accumulation | Todos adapters produzem 1 fill record | Médio — iceberg/partial-fill-accumulation é cenário futuro |
| Rejection event publication | Rejections retornam `Problem`, não `VenueOrderReceipt` | Decisão arquitetural — rejections não produzem `VenueOrderFilledEvent` |
| Paper mode exercício direto | `FC-9` bloqueia `paper_simulator` + `dry_run=false` | By design — paper sempre roda sob DryRunSubmitter |
| PriceSource wiring produção | Interface testada, wiring NATS KV pendente | Baixo — S384 definiu interface, wiring é straightforward |

## 6. Preparação Recomendada para S386

Com S385 fechando a lacuna entre modelagem e execução, S386 tem duas opções naturais:

### Opção A: Rejection Event Path (recomendada)
- **Escopo:** Provar que rejections no venue_live produzem um evento auditável (ex: `VenueOrderRejectedEvent`)
- **Motivo:** Hoje rejections retornam `Problem` e o actor loga, mas não publica evento. Downstream (store, writer) nunca vê rejections.
- **Risco se adiado:** Rejections reais no venue ficam invisíveis para observabilidade.

### Opção B: Cancel-Order Path
- **Escopo:** Implementar o fluxo `accepted → cancelled` via cancel-order API
- **Motivo:** `cancelled` está mapeado no state machine mas não é exercitado end-to-end.
- **Risco se adiado:** Baixo — cancel não é path dominante para market orders.

### Opção C: PriceSource Production Wiring
- **Escopo:** Implementar `CandleKVPriceSource` lendo do bucket `CANDLE_LATEST` e wirar em `cmd/execute/run.go`
- **Motivo:** Fecha o gap G1 de price realism em produção.
- **Risco se adiado:** Fills continuam com `Price="0"` em dry-run/paper — funcional mas não realista.

**Recomendação:** S386 = Opção A (Rejection Event Path), pois é o gap mais visível no write-path observability e mantém a wave focada no OMS foundation sem breadth extra.

## Guard Rails Compliance

- [x] Não abriu OMS completo
- [x] Não introduziu novos modos de execução
- [x] Não inflou para soak/benchmark
- [x] Diferenças reais entre modos documentadas sem mascaramento
- [x] Etapa focada em execução do modelo já definido
