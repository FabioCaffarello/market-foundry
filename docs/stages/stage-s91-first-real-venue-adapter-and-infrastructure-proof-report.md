# Stage S91: First Real Venue Adapter and Infrastructure Proof

**Date:** 2026-03-19
**Status:** COMPLETE
**Predecessor:** S90 (First Guarded Real-Venue Step)

---

## 1. Resumo Executivo

O S91 implementou o primeiro adapter real mínimo para o Binance Futures Testnet e fechou o hard blocker HB-S89-3 com um harness completo de testes de integração embarcados com NATS. A base está pronta para uma cerimônia formal de activation gate.

**Entregas:**
- `BinanceFuturesTestnetAdapter` implementado (VenuePort compliant, 11 unit tests)
- Embedded NATS integration harness com 11 cenários (HB-S89-3 CLOSED)
- Venue type registrado, config-driven wiring completo
- 3 documentos de arquitetura + relatório de stage

---

## 2. Venue Escolhida e Justificativa

**Binance Futures Testnet** (`binance_futures_testnet`)

| Critério | Justificativa |
|----------|--------------|
| Fit natural | Codebase usa `"binancef"` como source canônico |
| Segurança sandbox | Testnet sem exposição de capital real |
| API simples | Market orders via REST + HMAC-SHA256, sem WebSocket |
| Fills síncronos | `newOrderRespType=RESULT` retorna fill inline |
| Zero deps externas | `net/http` stdlib é suficiente |

---

## 3. Adapter Real Mínimo Implementado

### Implementação

`internal/application/execution/binance_futures_testnet_adapter.go`

- Implementa `ports.VenuePort` (SubmitOrder)
- HMAC-SHA256 signing com `crypto/hmac`
- Symbol mapping: `btcusdt` → `BTCUSDT`
- Error classification: auth, rate limit, rejection, server error
- Fill mapping: Binance response → `FillRecord{Simulated: false}`
- No-action handling: `Side=none` retorna `StatusAccepted` sem HTTP call

### 7 Invariantes do Contrato (de minimal-real-venue-adapter-contracts.md)

| # | Invariante | Status |
|---|-----------|--------|
| 1 | Context respect | OK — `http.NewRequestWithContext` |
| 2 | No gate bypass | OK — gates no actor layer |
| 3 | Credential isolation | OK — env vars only |
| 4 | Problem classification | OK — HTTP status → Problem code |
| 5 | Fill completeness | OK — Status + FilledQty + Fills |
| 6 | VenueOrderID uniqueness | OK — Binance orderId (int64→string) |
| 7 | Side-None handling | OK — no HTTP call |

### Wiring

`cmd/execute/run.go`:
```go
case settings.VenueTypeBinanceFuturesTestnet:
    creds, prob := appexec.LoadCredentials(...)
    return appexec.NewBinanceFuturesTestnetAdapter(creds, submitTimeout), nil
```

### Tests (11 unit tests — ALL PASS)

| Test | O Que Valida |
|------|-------------|
| SubmitOrder_Filled | Buy order → filled with real price |
| SubmitOrder_SellSide | Sell side mapping |
| SubmitOrder_NoAction | Side=none → no HTTP call |
| SubmitOrder_AuthError | 401 → InvalidArgument, non-retryable |
| SubmitOrder_RejectedOrder | 400 → InvalidArgument, non-retryable |
| SubmitOrder_ServerError | 503 → Unavailable, retryable |
| SubmitOrder_Timeout | Timeout → Unavailable, retryable |
| SubmitOrder_RateLimited | 429 → Unavailable, retryable |
| SymbolMapping | btcusdt → BTCUSDT |
| SignaturePresent | HMAC-SHA256 signature (64 hex chars) |
| FillNotSimulated | Simulated=false for real venue |

---

## 4. Prova de Infraestrutura/Integration Concluída

### HB-S89-3: CLOSED

Embedded NATS integration harness com 11 cenários usando `nats-server/v2` in-process.

| # | Cenário | Aspecto Provado |
|---|---------|----------------|
| 1 | PublishExecution_ConsumerReceives | Publish → consume pipeline (paper family) |
| 2 | PublishFill_FillConsumerReceives | Publish → consume pipeline (venue family) |
| 3 | ExecutionKV_PutGet | KV roundtrip para projection |
| 4 | ExecutionKV_MonotonicityGuard | Stale/duplicate rejection no KV |
| 5 | ControlGate_Lifecycle | Kill switch active → halted → active |
| 6 | PublishConsumeProject_Pipeline | Full publish → consume → project → read |
| 7 | JetStream_Deduplication | Duplicate event → exactly 1 delivery |
| 8 | MultiSymbol_Isolation | 3 symbols isolados em KV |
| 9 | FillPipeline_PublishConsumeProject | Fill event → consume → project → read |
| 10 | ControlGate_BlockAndResume | Halt → multiple reads → resume |
| 11 | ConsumerStats_Tracking | Delivery counters accuracy |

### Invariantes Provadas

- Publish/consume funciona para ambas famílias (paper + venue)
- KV projections com monotonicity guard contra JetStream real
- Kill switch (control gate) lifecycle operacional
- JetStream deduplication previne processamento duplicado
- Multi-symbol isolation preservada no pipeline completo
- Trace propagation (correlation/causation) preservada

---

## 5. Arquivos Alterados

### Novos

| Arquivo | Propósito |
|---------|----------|
| `internal/application/execution/binance_futures_testnet_adapter.go` | Adapter real Binance Futures testnet |
| `internal/application/execution/binance_futures_testnet_adapter_test.go` | 11 unit tests |
| `internal/adapters/nats/execution_integration_test.go` | 11 integration tests (embedded NATS) |
| `docs/architecture/first-real-venue-adapter-design.md` | Design do adapter |
| `docs/architecture/embedded-nats-integration-proof.md` | Prova de infraestrutura |
| `docs/architecture/real-venue-minimal-operational-scope.md` | Escopo operacional |
| `docs/stages/stage-s91-first-real-venue-adapter-and-infrastructure-proof-report.md` | Este relatório |

### Modificados

| Arquivo | Mudança |
|---------|--------|
| `internal/shared/settings/schema.go` | Adicionou `VenueTypeBinanceFuturesTestnet` + knownVenueTypes |
| `cmd/execute/run.go` | Adicionou case `binance_futures_testnet` em buildVenueAdapter |
| `internal/adapters/nats/go.mod` | Adicionou `nats-server/v2` para testes embarcados |

---

## 6. Limites Remanescentes

| Limite | Descrição |
|--------|-----------|
| Testnet keys não provisionadas | Adapter existe mas credenciais testnet não foram geradas |
| Activation gate não realizada | Cerimônia formal de ativação pendente para S92 |
| Adapter mainnet ausente | Apenas testnet; produção requer adapter separado |
| Async fills não suportados | Apenas fills síncronos (market orders) |
| Fee reconciliation ausente | `cumQuote` como proxy; comissões reais em endpoint separado |
| Retry/circuit breaker ausente | Errors classificados mas não retried automaticamente |
| Multi-venue bloqueado | Apenas uma venue por vez |
| OMS ausente | Sem state tracking beyond request/response |

---

## 7. Preparação Recomendada para S92

### Objetivo Sugerido

Cerimônia formal de activation gate para Binance Futures testnet.

### Pré-requisitos

1. **Provisionar testnet API keys** — criar conta testnet Binance Futures, gerar API key/secret.
2. **Primeiro smoke test real** — executar o adapter contra testnet com um market order mínimo (e.g., 0.001 BTCUSDT).
3. **Validar fill reconciliation** — confirmar que fills reais da testnet são corretamente mapeados para `FillRecord`.
4. **Decision: staleness/timeout tunning** — ajustar `staleness_max_age` e `submit_timeout` baseado em latência observada da testnet.
5. **Formalizar activation gate ceremony** — documento de gates, critérios, rollback plan.

### Guard Rails para S92

- Não abrir mainnet antes de testnet validada.
- Não abrir multi-venue.
- Não construir OMS.
- Não expandir para múltiplos símbolos sem necessidade.
- Manter kill switch operacional e testado.

---

## Critérios de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Primeiro adapter real mínimo implementado | PASS |
| Venue escolhida explícita e justificada | PASS |
| Sistema estreito, guardado, sem salto de escopo | PASS |
| Embedded NATS integration harness fortalecido | PASS (11 cenários, HB-S89-3 CLOSED) |
| Invariantes críticas provadas ou bloqueios explícitos | PASS |
| Base pronta para activation gate ceremony | PASS |
