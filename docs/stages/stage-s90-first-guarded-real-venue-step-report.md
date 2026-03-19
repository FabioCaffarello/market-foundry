# Stage S90: First Guarded Real-Venue Step Report

> **Stage:** S90
> **Date:** 2026-03-19
> **Type:** Activation/Secrets/Wiring Implementation
> **Gate:** S89 post-hardening action boundary gate (GO CONDICIONAL)
> **Option Selected:** Option 2 — Activation/secrets/wiring-only

---

## 1. Resumo Executivo

O S90 validou o veredito do S89 (GO CONDICIONAL) e selecionou **Opção 2: Activation/secrets/wiring-only** — implementando os pré-requisitos de infraestrutura que convertem designs do S88 em código de runtime, sem acionar execução real de venue.

**Resultado:** 3 dos 3 hard blockers resolvidos ou parcialmente resolvidos. 2 de 2 PRE-A items implementados. Nenhuma execução real foi acionada. O sistema permanece em paper mode com a infraestrutura de ativação pronta.

### Blockers Resolvidos

| Blocker | Status | Evidência |
|---------|--------|-----------|
| HB-S89-1: Credential infrastructure | **RESOLVIDO** | `LoadCredentials()`, `CredentialSet`, env_file template, .gitignore, fail-fast em run.go |
| HB-S89-2: Reconciliation enforcement | **RESOLVIDO** | RC-1 (fill-to-intent), RC-2 (quantity boundary), RC-4 (orphan handling) em FillProjectionActor |
| HB-S89-3: NATS integration tests | **PARCIAL** | `make test-integration` target criado; embedded NATS test harness pendente |

### PRE-A Items Implementados

| Item | Status | Evidência |
|------|--------|-----------|
| PRE-A5: Venue submit timeout | **RESOLVIDO** | Config-driven timeout (1s–60s, default 10s) em VenueAdapterActor |
| PRE-A6: Staleness configurable | **RESOLVIDO** | Config-driven staleness (30s–600s, default 120s) em VenueConfig |

---

## 2. Arquivos Alterados

### Novos Arquivos

| Arquivo | Propósito |
|---------|-----------|
| `internal/application/execution/credentials.go` | LoadCredentials, CredentialSet (HB-S89-1) |
| `internal/application/execution/credentials_test.go` | 5 unit tests para credential loading |
| `deploy/configs/execute.env.example` | Template de credenciais para Docker Compose |
| `docs/architecture/first-guarded-real-venue-step.md` | Escopo e decisões do S90 |
| `docs/architecture/real-venue-activation-and-secret-handling.md` | Credential delivery, activation flow, kill switch |
| `docs/architecture/minimal-real-venue-adapter-contracts.md` | VenuePort contract, adapter invariants, registration |
| `docs/stages/stage-s90-first-guarded-real-venue-step-report.md` | Este relatório |

### Arquivos Modificados

| Arquivo | Mudança |
|---------|---------|
| `internal/shared/settings/schema.go` | VenueConfig: `staleness_max_age`, `submit_timeout` fields + validation + duration methods |
| `internal/actors/scopes/execute/venue_adapter_actor.go` | `SubmitTimeout` em config; `context.WithTimeout()` no submit; log atualizado |
| `internal/actors/scopes/execute/execute_supervisor.go` | Lê staleness/timeout do config; log atualizado com venue_type/staleness/timeout |
| `internal/actors/scopes/store/fill_projection_actor.go` | RC-1/RC-2/RC-4 gates; `intentStore`; `orphaned`/`overflowed` counters; stats invariant atualizado |
| `internal/actors/scopes/store/fill_projection_actor_test.go` | Stats invariant atualizado; teste RC-1 sem intent store |
| `internal/actors/scopes/store/store_supervisor.go` | Passa `IntentBucket` para FillProjectionConfig |
| `cmd/execute/run.go` | `buildVenueAdapter` com credential loading para venue types futuros |
| `deploy/configs/execute.jsonc` | `staleness_max_age` e `submit_timeout` fields |
| `.gitignore` | `*.env` pattern adicionado |
| `Makefile` | `test-integration` target adicionado |

---

## 3. Testes

### Testes Novos (5)

| Teste | Arquivo | Cobertura |
|-------|---------|-----------|
| `TestLoadCredentials_AllPresent` | `credentials_test.go` | Carregamento completo de credenciais |
| `TestLoadCredentials_MissingRequired_FailsFast` | `credentials_test.go` | Fail-fast quando credencial ausente |
| `TestLoadCredentials_NoRequiredKeys_Succeeds` | `credentials_test.go` | Paper mode (sem credenciais necessárias) |
| `TestLoadCredentials_HasKey` | `credentials_test.go` | Verificação de presença de chave |
| `TestLoadCredentials_NilCredentialSet` | `credentials_test.go` | Segurança com nil receiver |

### Testes Atualizados (2)

| Teste | Mudança |
|-------|---------|
| `TestFillProjection_StatsInvariant_ReceivedEqualsSum` | Inclui `orphaned` e `overflowed` na soma |
| `TestFillProjection_RC1_OrphanFill_NoIntentStore` | Verifica que RC-1 não interfere quando desabilitado |

### Resultado

```
ok  internal/application/execution    0.156s
ok  internal/actors/scopes/store      0.168s
ok  internal/shared/settings          0.156s
```

Todos os testes passam. Falhas pré-existentes em `cmd/gateway` e `internal/application/configctl` não são relacionadas ao S90.

---

## 4. Limites Encontrados

### HB-S89-3 Parcialmente Resolvido

O `make test-integration` target está implementado, mas o embedded NATS test harness com 8 cenários de teste requer adição da dependência `nats-server` embedded. Esta é uma tarefa de infraestrutura de teste que não altera o código de produção.

**Impacto:** A cerimônia de ativação AG-1..AG-17 exige os 8 cenários de integração. Este item deve ser completado no S91.

### Reconciliation com IntentStore Indisponível

Quando o intent KV store está indisponível, os gates RC-1 e RC-2 são desabilitados (degradação graceful). Em paper mode, isso é aceitável porque fills sempre correspondem a intents. Em real venue, a indisponibilidade do intent store deve ser tratada como erro.

---

## 5. Itens Explicitamente Adiados para S91+

| Item | Motivo | Stage Alvo |
|------|--------|------------|
| Real venue adapter implementation | Requer escolha de exchange + API study | S91 |
| Embedded NATS test harness (8 cenários) | Dependência de infraestrutura de teste | S91 |
| Venue architecture doc | Depende de exchange alvo | S91 |
| Drift rules para novo venue type | Depende de venue adapter | S91 |
| Activation gate ceremony (AG-1..AG-17) | Requer S91 completo | S92 |
| FillTrackerActor (async polling) | Design-only no S88; não necessário até venue real | S91+ |
| Background reconciliation actor (RC-5) | PRE-O3 | S95 |
| Prometheus /metrics | PRE-O1 | S95 |
| CI pipeline automation | PRE-O2 | S95 |
| Consumer-projection coupling fix | PRE-O4 | S95 |
| Transitional bridge migration | Requer venue-specific intent events | S91+ |

---

## 6. Verificação de Guard Rails

| Guard Rail | Status |
|-----------|--------|
| Não construir OMS | ✅ Respeitado |
| Não abrir multi-venue | ✅ Respeitado |
| Não abrir portfolio | ✅ Respeitado |
| Não contradizer S89 | ✅ Respeitado — implementou exatamente os blockers identificados |
| Não permitir caminho real sem secrets/activation/kill switch | ✅ `buildVenueAdapter()` falha para venue types sem credenciais |
| Kill switch, authority, governança, traceability válidos | ✅ Sem alteração nos mecanismos existentes |
| Readiness provada antes de avançar | ✅ Nenhuma execução real acionada |

---

## 7. Recomendação para S91

O S90 concluiu a camada de wiring. O próximo stage deve:

1. **Escolher a exchange alvo** (testnet first).
2. **Implementar o venue adapter** com os 7 invariants definidos em `minimal-real-venue-adapter-contracts.md`.
3. **Implementar embedded NATS test harness** com os 8 cenários.
4. **Criar venue architecture doc** e atualizar drift rules.
5. **Após S91 completo**, executar a cerimônia de ativação (S92).

---

## 8. Artefatos Produzidos

| Artefato | Caminho |
|----------|---------|
| Escopo e decisões | `docs/architecture/first-guarded-real-venue-step.md` |
| Activation e secrets | `docs/architecture/real-venue-activation-and-secret-handling.md` |
| Adapter contracts | `docs/architecture/minimal-real-venue-adapter-contracts.md` |
| Stage report | `docs/stages/stage-s90-first-guarded-real-venue-step-report.md` |
