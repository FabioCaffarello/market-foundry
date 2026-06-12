# PROGRAM-0004 — Fase Multi-venue

**Status:** Active
**Date:** 2026-05-25
**Owner:** Repository maintainer (Fabio Caffarello)
**Relates to:**
[`../decisions/0021-canonical-instrument-and-venue-model.md`](../decisions/0021-canonical-instrument-and-venue-model.md)
(Proposed),
[`../decisions/0022-multi-venue-normalization-policy.md`](../decisions/0022-multi-venue-normalization-policy.md)
(Proposed),
[`../decisions/0017-event-envelope-and-versioning.md`](../decisions/0017-event-envelope-and-versioning.md)
(Accepted, defines envelope `venue` + `instrument`),
[`PROGRAM-0001-foundation.md`](PROGRAM-0001-foundation.md) (closed),
[`PROGRAM-0002-wire.md`](PROGRAM-0002-wire.md) (closed),
[`PROGRAM-0003-observability.md`](PROGRAM-0003-observability.md)
(Active),
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest",
[`../RESUMPTION.md`](../RESUMPTION.md)

---

## Objetivo

Tornar o `market-foundry` **estruturalmente multi-venue** —
introduzir o modelo canônico de instrumento (`CanonicalInstrument`
per ADR-0021), refatorar os dois adapters Binance existentes
(`binances`, `binancef`) para emitirem o modelo canônico, expandir
para um segundo venue (Bybit) para provar paridade cross-venue, e
codificar a política de normalização cross-venue (ADR-0022).

Esta Fase entrega **infraestrutura de cross-venue** — o modelo, a
política, os dois primeiros adapters compliant. Não entrega
delivery futures discrimination além do binancef, options, ou
cross-listed asset alias resolution (todos non-goals explícitos,
ver abaixo).

PROGRAM-0001 (Foundation, closed) entregou ADR-0021 + ADR-0022
como `Proposed` em H-2. PROGRAM-0002 (Wire, closed) adicionou o
envelope `instrument` field (string) per ADR-0017. PROGRAM-0003
(Observability, Active) instrumentou o stack de metrics que
multi-venue dashboards consumirão. PROGRAM-0004 implementa o
modelo + adapters que validam ADR-0021 e prepara o terreno para
ADR-0022.

---

## Escopo (Ondas)

| Onda | Escopo resumido | Entregas principais |
|------|------------------|---------------------|
| **H-6** | Canonical instrument model + Binance refactor | Distribuído em **sub-ondas H-6.a–H-6.f** por questão de cascade (descoberto em pré-flight de H-6.a: 342 `.Symbol` references em 106 production files em 31 packages). Ver "Sub-ondas H-6" abaixo. |
| **H-7** | Bybit adapter + multi-venue parity policy + expiry (G10) | Dividida em **sub-ondas H-7.a/b/c** na abertura (owner 2026-06-12, Decisão #1 (B) — mesmo rationale do split de H-6: escopo combinado produziria PR irrevisável). **H-7.a**: capabilities framework (ADR-0022 R1–R4 sem venue novo — tipo `Capabilities` + retrofit binances/binancef + gateway `/venues/capabilities` + counter + analyzer `check venue-parity`). **H-7.b**: adapter Bybit (spot + linear perpetual, plano de observação apenas — Decisão #2 (A)) + **promoção ADR-0022 → Accepted** (fecha os 6 critérios; atômica no commit final). **H-7.c**: modelagem do expiry (G10) — campo opcional + ativação do slot `[_expiry]` do token (Decisão #4 (A); coluna ClickHouse deferida até habilitar delivery futures no ingest). Ver "Sub-ondas H-7" abaixo. |

H-6 e H-7 são **sequenciais e estritas** — H-7 só abre após
**H-6.f.1 mergeada em main** (P9 + sub-onda sequencing policy
abaixo). Erratum 2026-06-11: o texto original dizia "após H-6.f";
com o split f.1/f.2 (Decisão #1 da abertura de H-6.f.1), H-7
destrava em f.1 e **H-6.f.2 fecha a promoção do ADR-0021** — ver
"Erratum de sequenciamento" na sequencing policy abaixo.

## Sub-ondas H-7 — decisões da abertura (owner, 2026-06-12)

Pré-flight da abertura (main@5195f8e) grounded as cinco decisões;
wave prompt auditado pelo owner em 2026-06-12:

- **Decisão #1 (B) — split serial a/b/c.** H-7.a (capabilities
  framework) → H-7.b (Bybit + promoção ADR-0022) → H-7.c (expiry).
  Próxima sub-onda abre APENAS após merge da anterior (P4/P9).
- **Decisão #2 (A) — escopo Bybit**: spot + linear perpetual,
  **plano de observação apenas** (ingest → evidence/derive).
  Inverse (coinfutures) e delivery futures FORA (delivery segue
  gated por G10 até H-7.c). **Execução segue Binance-only** — o
  segment model do execute (`settings.sourceForSegment`, S400) não
  é tocado; Bybit na execução é non-scope desta Fase.
- **Decisão #3 (A) — sources `bybits`/`bybitf`**, espelhando
  binances/binancef. Rationale grounded no pré-flight: o registry
  `venueSourceContract` (`application/ingest/binding.go`) é uma
  bijeção source→contract; um source único "bybit" cobrindo
  spot+linear quebraria a fundação do `BindingTarget.Instrument()`.
- **Decisão #4 (A) — expiry como campo opcional** (`Expiry string`,
  vazio = sem expiry; zero impacto nos 4 contract types atuais);
  quando não-vazio ativa o 4º componente do SubjectToken (slot
  dormente do erratum ADR-0009) e estende FromSubjectToken (o
  lock-in test da f.1 tem pause trigger armado exatamente para
  isso) + Symbol()/FromSymbol; errata ADR-0009/0021 na sub-onda.
  **Coluna ClickHouse `expiry` deferida** até a onda que habilitar
  delivery futures no ingest (nenhum circula hoje; a cascade de
  codegen/goldens/positional da d.1 não se paga agora) — gap
  sucessor do G10 registrado no closure da c.
- **Decisão #5 (A) — house pattern no adapter Bybit**
  (`parseBybitSymbol` + `Normalize`, coerente com os 2 adapters
  existentes). O naming literal `ToCanonical`/`FromCanonical` do
  ADR-0021 ganha **erratum de equivalência** na revisão da H-6.f.2
  (criterion #2 já foi aceito com o shape house pattern na e.2).

---

## Sub-ondas H-6 — sequenciamento estrito

A descoberta do pré-flight de H-6.a (342 `.Symbol` references em
106 production files em 31 packages) tornou impossível executar
H-6 como onda única honesta — a refatoração tocaria todo o
foundry simultaneamente e produziria PR irrevisável.

H-6 é portanto implementada em **8 sub-ondas serializadas**
(H-6.b sub-dividida em b/b'/b'' após pré-flight de H-6.b
descobrir 15 domain types totalizando 174 test files — ver
"Refinamento H-6.b" abaixo):

| Sub-onda | Escopo | Entregas principais |
|----------|--------|---------------------|
| **H-6.a** | Domain root + Binance adapters | `internal/domain/instrument/` package (Venue, BaseAsset, QuoteAsset, ContractType, CanonicalInstrument). Refactor `ObservationTrade.Symbol` → `Instrument`. Adapters `binances` + `binancef` `Normalize` emitem `CanonicalInstrument`. Imediate readers de `ObservationTrade` migram para `.Instrument.Symbol()`. raccoon-cli `check instruments` analyzer. ADR-0021 erratum critério #4 (commit 0). PRD-0004 abertura. **ADR-0021 permanece `Proposed`.** |
| **H-6.b** | Layer 1+2: Evidence + Signal/Decision/Strategy/Risk (7 types) | Cada domain struct (`EvidenceCandle`, `EvidenceTradeBurst`, `EvidenceVolume`, `Signal`, `Decision`, `Strategy`, `RiskAssessment`) migra `Symbol string` → `Instrument CanonicalInstrument` + `VenueSymbol() string` transitório. Os 5 `PartitionKey()` composers (Signal/Decision/Strategy/Risk/Execution) compõem chave KV via `VenueSymbol()` para back-compat com bucket layout existente. raccoon-cli `check instruments` estendido via `policies/domain_types.toml` declarando migration_state per type. **ADR-0021 permanece `Proposed`.** |
| **H-6.b'** ✅ fechada | Layer 3+3': ExecutionIntent + Attribution + AuditLifecycleEntry (3 types) | Execution chain migrada. `ExecutionIntent.Symbol` → `.Instrument` + `VenueSymbol()`; PartitionKey/DeduplicationKey composers atualizados via VenueSymbol. `effectiveness.Attribution.Symbol` → `.Instrument` (derived from `intent.Instrument`). `execution.AuditLifecycleEntry.Symbol` → `.Instrument` (projection de partition key via novo `instrumentFromBinding` per-package em `executionclient`, sunset H-6.f). Triage drop: zero population sites required migration nesta sub-wave (ver Changelog). **ADR-0021 permanece `Proposed`.** |
| **H-6.b''** ✅ fechada | Layer 4: Pairing.Leg/RoundTrip + CrossSessionWindow + Triage population sites | Pairing chain migrada. `pairing.Leg.Symbol` → `.Instrument` + `VenueSymbol()` (M1 invariant via native Go struct equality `entry.Instrument != exit.Instrument` — `CanonicalInstrument` é composto de 3 string-typed components e comparable por construção; estritamente mais forte que symbol equality, pois Contract type também discrimina). `pairing.RoundTrip.Symbol` → `.Instrument` (denormalized from Leg per Decisão #3; invariant `RoundTrip.Instrument == Entry.Instrument == Exit.Instrument` enforced by MatchFIFO construction + M1). `pairing.CrossSessionWindow.Symbol` → `VenueSymbol string` **rename only** per Decisão #2 (b): pre-flight 6 confirmed the field is query metadata, never read by matching algorithm, validated only by `!= ""` — promoting to Instrument would force regression-prone source-string reconstruction at the two construction sites (`get_cross_session_pairing.go:135` + `get_continuity_review.go:178`); same regression-shape as commit 37f8ddd in H-6.b'. New `string_filter` migration_state introduced in commit 1 (analyzer schema extension) records this architectural decision permanently. Triage population site em `triageclient/get_roundtrip_triage.go:74` adopts `.VenueSymbol()` (RoundTripTriageItem.Symbol stays string per S472-style projection) — pulled forward into commit 3 by compile pressure (anonymous embedding forces co-location). Decisão #5β/γ test canaries: new unit test `get_roundtrip_triage_test.go` (β; happy-path projection + zero-Instrument observable canary) + smoke check in `smoke-analytical-e2e.sh` Phase 5 (γ; tri-state PASS/WARN/FAIL — WARN when matched-pair data unavailable in smoke window). **8 commits delivered** (plan declared 9 — see Changelog 2026-05-26 H-6.b'' entry for the consolidation rationale). **ADR-0021 permanece `Proposed`.** |
| **H-6.c** | Application layer + actors + samplers | Sub-dividida em **H-6.c.1** (derive scope) e **H-6.c.2** (execute scope + ClickHouse readers) post-pré-flight 5 de H-6.c (descoberta de 6 helpers `instrumentFromBinding` + 13 callers `reconstructInstrumentFromLegacy` + DTO migration cascade). **ADR-0021 permanece `Proposed`** em ambas. |
| **H-6.c.1** ✅ fechada | Application pass-through: derive scope | `instrumentFromBinding` helper **eliminado** de 4 application packages (signal/decision/strategy/risk) — commits 7a-7d. Novo canonical boundary helper `internal/application/ingest/binding.go:BindingTarget.Instrument()` (commit 6) com signature error-returning — synthetic sources (`"binance"`, `"binance_spot"`, `"derive"`, `"clickhouse"`, `"unknown_exchange"`, `"execute.venue-adapter"`) intencionalmente ausentes do registry, surfacing o H-6.b' 37f8ddd silent-zero regression-shape rather than hiding it. Derive actors computam Instrument uma vez em `source_scope_actor.onActivateSampler` e fazem pass-through pelo cascade signal/decision/strategy/risk/execution. 14 `NewXxxForInstrument` constructors (commits 2-5); 5 derive Config structs gain canonical Instrument field; `derive_supervisor` cascades inst por 12 factory NewActor callbacks. ~250 application test sites migrados. Derive-scope canary integration tests (commit 8: 3 tests / 15 subtests). Nova `tools/raccoon-cli/policies/anti_patterns.toml` + analyzer scan extension (commit 1). **10 commits delivered**. **ADR-0021 permanece `Proposed`.** |
| **H-6.c.2** ✅ fechada | Application pass-through: execute scope + ClickHouse composite_reader uniformization + ReviewTransform string_filter | `instrumentFromBinding` helper **eliminado** do execution package (5º de 6 — commits 4 + 5). Testnet adapters (binance_spot/futures) usam `BindingTarget.Instrument()` boundary helper com warn-and-emit-zero fallback (commit 4 — per Decisão #2 após cascade analysis revelou option (a) port-signature refactor = 12 files, excedendo o threshold da sub-onda). ClickHouse `composite_reader.go` 5 silent sites (linhas 188/243/302/360/423) convertidos para warn-and-emit-zero (commit 2), uniformizando os 13 `reconstructInstrumentFromLegacy` callers. `ReviewTransform` DTO declarado como `string_filter` em `domain_types.toml` + godoc inline em ReviewTransform.Symbol + DecisionTriageItem.Symbol (commit 3 — zero production code change; cascade já completo desde H-6.b). 37f8ddd canary explícito em execute scope: `execute_venue_adapter_canary_test.go` com 2 tests / 2 passes (commit 6). 8 cross-scope test stragglers migrated (1 derive + 2 risk + 6 integration-tagged; commit 5 — surfaced pelo explicit integration build check). anti_patterns exception list reduzida 11 → 8 (commit 7 — net -3 execution entries). **8 commits delivered**. **ADR-0021 permanece `Proposed`.** |
| **H-6.d** | ClickHouse migration + writer canonical population + reader cutover (#4b) | Sub-dividida em **H-6.d.1** (schema migration + writer canonical-column population) e **H-6.d.2** (reader cutover canonical-preferred) post-pré-flight (descoberta de positional-INSERT cascade em integration tests + tagged-build drift de 3 meses). **ADR-0021 permanece `Proposed`** em ambas. |
| **H-6.d.1** ✅ fechada | ClickHouse schema migration + writer canonical column population | 6 migrations adicionadas (`008_add_canonical_columns_evidence_candles.sql` → `013_add_canonical_columns_executions.sql`) — split per-table after ClickHouse Go driver multi-statement constraint surfaced (Decisão #1 (A); runner enhancement deferred to H-6.f scope expansion). Cada migration adiciona `base`/`quote`/`contract LowCardinality(String) DEFAULT '' AFTER symbol/base/quote` idempotently (`ADD COLUMN IF NOT EXISTS`). Writer population end-to-end: 14 YAML specs + 14 golden snapshots regenerados via codegen, 17 INSERT SQL strings em `cmd/writer/pipeline.go`, 8 mappers em `writerpipeline/support.go` (cada um appends 3 canonical values após `VenueSymbol()`), ~120 test row position shifts em `support_test.go` + `behavioral_roundtrip_test.go` (codegen self-consistency invariant — bundle atômico). Integration fixture migration (commit 3a): 34 positional INSERTs em `composite_reader_integration_test.go` convertidos para explicit column lists (5 unique templates per table) + 20 pre-H-6.b `.Symbol` references migrados para `.VenueSymbol()` em `composite_reader_integration_test.go` + `live_execution_analytical_test.go` — descoberta de drift de 3 meses não capturada pelo default `make verify` (tagged-build invisibility lesson). Writer canary (commit 3b): `Client.Exec()` adicionado para DDL via native protocol (clickhouse-go/v2 Query returns EOF on DDL), novo `canonical_columns_integration_test.go` com 6 tests / 1 per table verificando population end-to-end. Helper retention strategy (Resolução 1): 5 `composite_reader.go` callers + 8 sister-site readers de `reconstructInstrumentFromLegacy` MANTÊM warn-and-emit-zero fallback até H-6.f (TTL window 90 dias retire legacy rows; H-6.d.2 reader cutover preferred-canonical-with-fallback). **4 commits delivered**. **ADR-0021 permanece `Proposed`.** |
| **H-6.d.2** ✅ fechada | Reader canonical-preferred cutover | Analytical client readers migram para dual-path: `instrumentFromCanonicalColumns(base, quote, contract)` primary com sentinel `ErrLegacyRow`, fallback para `reconstructInstrumentFromLegacy(src, sym)` quando canonical columns empty, warn-and-emit-zero quando ambos falham. **7 reader files / 13 instrument-resolution sites / 13 SELECT column lists** atualizados uniformemente (8 query builders + 5 composite inline SELECTs); pattern uniforme através dos 13 sites validado em pré-flight 3. Novo helper `internal/adapters/clickhouse/canonical_instrument_columns.go` com `ErrLegacyRow` sentinel exportada — discriminates "expected legacy row → fallback" from "validation regression → propagate". 4 unit tests / 9 sub-cases lock-in o contrato do helper. Reader canary integration test `canonical_columns_reader_integration_test.go` (~714 LoC, `//go:build requireclickhouse`) com 6 tests / 18 subtests (canonical_path / fallback_path / mixed_state per table) — mixed_state subtest é a prova literal da Resolução 1 (ambas shapes coexistem durante 90-day TTL window). Per-reader test files (candle/signal/decision/strategy/risk/execution + s453a + s454a) atualizados: expectedCols slices estendidas com `base/quote/contract`, comentários column counts bumped (e.g. candle 12→15, execution 16→19). `reconstructInstrumentFromLegacy` **RETAINED** per Resolução 1; deletion deferida para H-6.f post-TTL operational verification. **Critério #4b reader-side LANDED** — completa ADR-0021 erratum #4b end-to-end (writer-side H-6.d.1 + reader-side H-6.d.2). **4 commits delivered**. **ADR-0021 permanece `Proposed`** (promotion gated em literal critério #2 satisfaction — executionclient helper deletion + operational verification em H-6.f). |
| **H-6.e** ✅ fechada | NATS subject canonical cutover (subjects only) | Pause-and-report executado como primeiro ato (2026-06-10). **Decisão do owner: (i)** — migrar subjects para forma canônica; (ii) descartada (o "deferral indefinido" apoiaria subjects num helper com sunset planejado em H-6.f e a lossiness de delivery futures colide token+PartitionKey exatamente na superfície que H-7 expande). Enumeração D (cap 30 min): **zero parsers do token de symbol** em qualquer superfície (consumers logam `msg.Subject()` apenas; fixtures de replay embedam `instrument` canônico, não subjects; Prometheus/Grafana omitem instrument per ADR-0024 MP-2) → **cutover atômico**, sem dual-publish; mixed-state nos streams até TTL 72h (precedente H-6.d). Token canônico `{base}_{quote}_{contract}` com slot `[_expiry]` dormente via helper único `SubjectToken()` (Decisão #1; débito de modelagem do expiry registrado — ver H-6.e.2). 10 builder sites com symbol migram (11º é session-lifecycle, sem symbol). Analyzer `check subjects` no mesmo PR, **escopo subjects-only** (Decisão #4 — `PartitionKey()` consome `VenueSymbol()` legitimamente até e.2). Errata: ADR-0009 (gramática) + ADR-0021 critério #2 (fechamento literal desloca para e.2). Nota: o texto original desta linha descrevia o estado como "Instrument.Symbol() (derived form)" — impreciso; era VenueSymbol-derived (corrigido no erratum ao ADR-0009). **ADR-0021 permanece `Proposed`.** |
| **H-6.e.2** ✅ fechada | KV partition keys + contrato HTTP de leitura (split da Decisão #2 de H-6.e) | Migra os 5 `PartitionKey()` composers (Signal/Decision/Strategy/Risk/Execution) e o contrato HTTP `(source, symbol, timeframe)` (`parseQueryKeyParams`) para forma canônica, coerentemente — as chaves são parser-free, mas o read path as constrói a partir do contrato HTTP: migrá-las sem mudar o contrato exigiria inferência venue→canonical no boundary, o anti-pattern eliminado em H-6.c. **Decisões da abertura (owner, 2026-06-11 — pacote B)**: (a) contrato → trio explícito `base/quote/contract` validado por `instrument.New` (sem dual-accept — exigiria a inferência banida; sem versionamento — zero consumidores externos, loopback/sem auth, Odin é H-12+ via WS+proto); (b) KV keys write+read → `{source}.{SubjectToken()}.{timeframe}` no mesmo commit (órfãos inertes com purge manual opcional; janela de miss por tipo até a primeira escrita pós-deploy — documentado); (c) ClickHouse `WHERE … symbol = ?` **inalterado**, valor derivado via helper transitório único `LegacyFilterValue()` = `lower(base+quote)` (direção legítima canonical→venue; a coluna legacy armazena exatamente esse valor) — **flip do WHERE para colunas canônicas em H-6.f pós-TTL** (~2026-08-26), atômico com as deleções de helper; (d) extensão do `check subjects` com seção `[keys]` (PartitionKey deve usar SubjectToken, proibido VenueSymbol) no mesmo PR per P5; (e) **expiry (G10) deferido a H-7** — e.2 não modela; G10 segue bloqueando delivery futures no ingest. **Dependências escritas**: e.2 abre APENAS após merge de H-6.e; **H-6.f abre APENAS após merge de H-6.e.2** (o sunset do `VenueSymbol()` em f bloqueia em e.2 — `PartitionKey()` o consome até lá). Critério #2 do ADR-0021 fecha literalmente aqui (erratum 2026-06-10). **ADR-0021 permanece `Proposed`.** |
| **H-6.f.1** | Cleanup não-TTL-gated + fix da regressão de auditoria (split da Decisão #1 da abertura de H-6.f, owner 2026-06-11, opção A) | Fix da **regressão silent-zero da auditoria** descoberta na abertura (audit bundles com `Instrument` zerado desde o merge de H-6.e.2: `audit_session.go` reconstrói via `instrumentFromBinding(e.Source, e.Symbol)`, que exige sufixo `USDT` venue-native — o token canônico `base_quote_contract` que `e.Symbol` passou a carregar não casa → `CanonicalInstrument{}`). Fix via novo parser `instrument.FromSubjectToken(token)` (direção legítima canonical→canonical; premissa "contract types não contêm underscore" com lock-in test); **deleção do 6º e último `instrumentFromBinding`** (`executionclient/instrument_binding.go`) + exception retirada de `anti_patterns.toml` + canário unit `AuditLifecycleEntry.Instrument` não-zero (a ausência desse canário foi o que deixou a regressão passar). Dedup keys canonicalizam (Decisão #4): 7 `DeduplicationKey()` de domínio + 4 builders inline nos publishers migram `VenueSymbol()` → `SubjectToken()`; analyzer `check subjects` estendido com seção `[dedup]` (P5); janela de dedup JetStream quebrada na transição — documentada, risco aceito single-operator. Migration runner multi-statement (deferral da H-6.d.1). Test-hardening: G8 fix obrigatório (FixedClock no TestS460); G7/G9 só se mecânico. **ADR-0021 permanece `Proposed`.** |
| **H-6.f.2** | Cleanup TTL-gated + ADR promotion — **abre SOMENTE pós-TTL (~2026-08-26)**, com verificação operacional como pré-condição da promoção | Flip do ClickHouse `WHERE` para colunas canônicas; deleções de `reconstructInstrumentFromLegacy`, `LegacyFilterValue()` (introduzido em H-6.e.2) e `VenueSymbol()` (133 sites); postura da coluna legacy `symbol` nos writers; exception list ClickHouse (7 entries) do `anti_patterns.toml`; verificação operacional pós-TTL. Atualiza TRUTH-MAP, RESUMPTION, GLOSSARY com state final. **Promove ADR-0021 → `Accepted`** apenas se TODOS os critérios (1, 2, 3, 4a, 4b, 5) estão literalmente satisfeitos. P7 absoluto. |

### Refinamento H-6.b (introduzido em H-6.b, pós-pré-flight)

H-6.a declarou H-6.b como "Evidence + Signal + Decision + Strategy
+ Risk" assumindo cascade tractable. Pré-flight obrigatório em
H-6.b revelou:

- **15 domain types** com Symbol field em `internal/domain/`
  (não 5).
- **390 production readers** de `.Symbol` (excluindo
  `instrument/`/`observation/` já migrados).
- **128 production construction sites** com literal `Symbol:`.
- **174 test files** referenciam Symbol — top 10 com 17–37
  literais cada.
- **ExecutionIntent** sozinho tem 199 test sites; **pairing.Leg**
  101; **pairing.RoundTrip** 66. Todos individualmente acima do
  threshold de 25 arquivos declarado em decisão #4 da onda.

Sub-divisão em **H-6.b / H-6.b' / H-6.b''** segue **dependency
order** dos domain types (Layer 1+2 / Layer 3+3' / Layer 4):

| Layer | Types | Sub-onda | Justificativa |
|-------|-------|----------|---------------|
| 1 | Evidence (Candle, TradeBurst, Volume) | H-6.b | Derivam de ObservationTrade (já migrada em H-6.a) |
| 2 | Signal, Decision, Strategy, RiskAssessment | H-6.b | Derivam de Evidence; pattern uniforme `PartitionKey()` |
| 3 | ExecutionIntent | H-6.b' | Deriva de Risk/Strategy/Decision (Layer 2) |
| 3' | Attribution, AuditLifecycleEntry | H-6.b' | Derivam de ExecutionIntent |
| 4 | Pairing.Leg, RoundTrip, CrossSessionWindow | H-6.b'' | Derivam de ExecutionIntent (via Leg construção) |
| 4' | Triage population sites | H-6.b'' (RoundTrip path) / H-6.c (Decision path) | Sites migram com upstream que populam o Symbol |

Vantagens da split por dependency order:

- **Sem buracos semânticos**: cada sub-onda fecha tendo migrado
  todos os types dos quais types subsequentes derivam. Nenhum
  type não-migrado consome type migrado.
- **Cascade balanceado**: H-6.b ~62 prod + 158 test sites;
  H-6.b' ~12 prod + 226 test sites; H-6.b'' ~13 prod + 200
  test sites.
- **Semântica coerente per sub-onda**: derivative analytics em
  H-6.b; execution chain em H-6.b'; pairing chain em H-6.b''.

Estado das domain types que **NÃO migram** em H-6.b/b'/b''
(per pré-flight Decisão #2):

- `triage.DecisionTriageItem.Symbol` e
  `triage.RoundTripTriageItem.Symbol` — projections de display,
  populados via cópia from upstream. Field permanece `string`;
  population sites migram para `.VenueSymbol()` na sub-onda do
  upstream.
- `consistency.ChainSnapshot.{Decision,Strategy,Risk,Execution}Symbol`
  — package documentado (S472) com invariante "primitive types
  only para evitar coupling com domain packages". Não migra;
  consistency checks comparam strings (continuam funcionando).

### Transitory-method pattern (introduzido em H-6.a)

H-6.a precisou consumir a cascade de `.Symbol` references sem
forçar migração simultânea de todos os 31 packages. A solução —
adotada após pause-and-report e descartando dual-write — foi
introduzir um **accessor transitório com nome semanticamente
distinto** no domain type recém-migrado:

- O field `Symbol string` foi removido e substituído por
  `Instrument CanonicalInstrument` (verdade canônica).
- Um método **`VenueSymbol() string`** foi adicionado, derivando
  forma venue-native (lowercase `base+quote`) da identidade
  canônica.

Por que **`VenueSymbol`** e não `Symbol()`:

- `CanonicalInstrument.Symbol()` retorna a forma canônica
  `"BTC/USDT-spot"`.
- `ObservationTrade.VenueSymbol()` retorna a forma venue-native
  `"btcusdt"`.
- Se ambos fossem chamados `Symbol()`, em 6 meses um caller
  escreveria `t.Symbol()` esperando a forma canônica e receberia
  silenciosamente a venue-native — bug latente clássico.

Atributos da disciplina:

1. **Sunset declarado no docstring**: cada método transitório lista
   a sub-onda que o remove. `VenueSymbol()` é removida em H-6.f.2
   (pós-split 2026-06-11; originalmente "H-6.f") quando o último
   reader venue-native sai do código.
2. **Limitações documentadas**: `VenueSymbol()` é lossy para
   delivery futures (`BTCUSDT_240329` colapsa para `"btcusdt"`).
   Aceitável em H-6.a porque nenhum contrato delivery rida o
   routing path atual; H-6.e revisita o shape da subject NATS.
3. **Nome distinto do canônico**: nunca substituir um símbolo
   canônico por um símbolo transitório com o mesmo identificador.

Sub-ondas H-6.b–H-6.e podem reusar o pattern conforme novos
domain types migram — cada um define seu próprio `VenueSymbol()`
ou equivalente, todos com sunset documentado para H-6.f.2
(pós-split 2026-06-11; docstrings escritos antes do split dizem
"H-6.f").

---

## Sub-onda sequencing policy

Sub-ondas H-6.a → H-6.b → H-6.b' → H-6.b'' → H-6.c → H-6.d →
H-6.e → **H-6.e.2** → **H-6.f.1** → **{H-7 ∥ H-6.f.2}** executam
serial até f.1. Próxima sub-onda abre branch APENAS após merge da
anterior em `main`. Após o merge de f.1: H-7 destrava
imediatamente; H-6.f.2 aguarda adicionalmente o **gate temporal
pós-TTL (~2026-08-26)** e abre com verificação operacional como
pré-condição da promoção. (H-6.e.2 inserida 2026-06-10 pelo split
da Decisão #2 de H-6.e; H-6.f dividida em f.1/f.2 em 2026-06-11
pela Decisão #1 da abertura de H-6.f.1.)

Dentro de H-7: **H-7.a → H-7.b → H-7.c** executam estritamente
serial entre si (split da Decisão #1 (B) da abertura de H-7,
2026-06-12), independentes de H-6.f.2 (que corre em paralelo no
gate temporal próprio).

### Erratum de sequenciamento (2026-06-11, Decisão #2 da abertura de H-6.f.1)

O texto original desta policy — e o gate "H-7 só abre após H-6.f
mergeada" acima — declarava a cadeia **estritamente serial** até
H-7. O split de H-6.f em f.1 (cleanup não-TTL-gated, agora) e f.2
(TTL-gated, ~2026-08-26) muda isso honestamente: a cadeia vira
`e → e.2 → f.1 → {H-7 ∥ f.2}`, com **f.2 fechando a promoção do
ADR-0021**. Rationale: o modelo canônico está entregue e provado
end-to-end; a promoção é formalização gated em **verificação
operacional pós-TTL**, não pré-requisito técnico do adapter Bybit
(H-7). P7 permanece intacto: ADR-0021 segue `Proposed` até f.2
cumprir os critérios literais — nenhuma claim antecipada.

### Sub-wave naming convention

- Documentation/prose: H-6.b, H-6.b', H-6.b'' (apostrophes
  distinguish dependency layers within the wave H-6.b family).
- Branch names / git tags: feat/h-6-b1-…, feat/h-6-b2-…
  (numeric suffix for portability across shells/CI tools where
  apostrophes are unsafe).

Established at H-6.b' (branch feat/h-6-b1-execution-chain);
applies retroactively to existing prose references.

Razões:

- **Sobreposição de código**: cada sub-onda migra mais domain
  types ou layers; paralelo garante merge conflicts no mesmo
  arquivo entre sub-ondas distintas.
- **Disciplina cognitiva**: cada sub-onda é um contexto que
  exige revisão completa. Múltiplas em flight diluem foco do
  reviewer e fragmentam o entendimento do estado parcial da
  migração.
- **TRUTH-MAP reflete estado real**: múltiplas sub-ondas
  in-flight produzem `TRUTH-MAP` inconsistente até a última
  fechar.

Esta política é **mais estrita que P4** (uma onda por vez, no
top-level) porque sub-ondas dentro de PROGRAM-0004 manipulam o
mesmo subsystema (canonical instrument migration), enquanto
ondas top-level frequentemente tocam subsystemas distintos.

---

## Não-Escopo

- **Coinbase, Hyperliquid, Kraken adapters.** H-6 + H-7 entregam
  Binance (refactor) + Bybit (novo). Os outros venues do ADR-0021
  `Venue` enum são reservados (a enum declara para
  forward-compatibility) mas adapters concretos são ondas futuras.
- **Bybit COIN-margined futures.** H-7 entrega Bybit USDT-perpetual
  como o primeiro caso. COIN-margined Bybit (se necessário) é
  onda futura.
- **Delivery futures discrimination além do binancef pattern.** H-6.a
  inclui pattern detection para `binancef` (suffix `_YYMMDD` →
  `ContractUSDTFutures`; sem suffix → `ContractPerpetual`). Outros
  venues replicam pattern quando o caso aparecer no dataset; sem
  generalização premature.
- **Options.** Strike, expiry per leg, multi-leg structures
  permanecem non-goal per ADR-0021 (linha 174-176). Foundry não
  trata options no roadmap previsível.
- **Cross-listed asset alias resolution** (BTC vs WBTC vs cbBTC).
  Per ADR-0021, são distintos `BaseAsset` values. Consumer que
  quer cross-token analytics aplica policy própria.
- **Stablecoin equivalence** (USDT vs USDC vs FDUSD). Per ADR-0021,
  distintos `QuoteAsset` values. Per-venue liquidity differs.
- **HTTP-API symbol format change.** Gateway pode expor canonical
  ou venue-friendly alias; H-6 toca apenas internal mesh.
  HTTP-API.md decisão futura.
- **Tracing / log aggregation / per-strategy SLOs.** Todos
  herdados de PROGRAM-0003 non-goals.

---

## Princípios governantes

A Fase Multi-venue opera sob o **protocolo P1–P9** documentado em
[`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest". Particularmente
relevantes para PROGRAM-0004:

- **P3** — Documento primeiro. ADR-0021 (canonical instrument)
  governa H-6; ADR-0022 (multi-venue normalization) governa H-7.
  Ambas já em main como `Proposed` desde H-2.
- **P4** — Uma onda por vez. **Estendida via sub-onda sequencing
  policy acima**: sub-ondas dentro de PROGRAM-0004 também são
  estritamente seriais, não apenas as top-level.
- **P5** — Cada onda evolui raccoon-cli quando adiciona invariante
  arquitetural. H-6.a adiciona `check instruments` (todo adapter
  declarado em policies/adapters.toml emite `CanonicalInstrument`).
  H-7 adiciona `check venue-parity` (cross-venue event-type
  coverage).
- **P6** — Pause-and-report ativo. H-6.e protocolo (NATS subject
  composition decision) é o caso explícito; mais surgirão durante
  as sub-ondas.
- **P7** — Sem perda de disciplina documental. ADR-0021 permanece
  `Proposed` até H-6.f satisfazer **literalmente** todos os
  critérios. Erratum REAL ao critério #4 (split #4a/#4b) lands
  em commit 0 de H-6.a; criterion #2 ("all domain-layer call
  sites migrated") **não** ganha erratum loophole.

---

## Critérios de aceite da Fase

A Fase Multi-venue fecha quando **todos** os critérios abaixo são
verdadeiros simultaneamente:

- [ ] Sub-ondas H-6.a, H-6.b, H-6.c, H-6.d, H-6.e, H-6.e.2,
  H-6.f.1, H-6.f.2, e Onda H-7 fechadas. Cada uma registrou
  fechamento explícito com `make verify` GREEN e RESUMPTION
  atualizado no commit de fechamento.
- [ ] `internal/domain/instrument/` package compliant com
  ADR-0021 spec (Venue, BaseAsset, QuoteAsset, ContractType,
  CanonicalInstrument, validation, `Symbol()` method).
- [ ] Adapters `binances` + `binancef` emitem `CanonicalInstrument`
  via `Normalize`. Pattern detection no `binancef` para discriminar
  `ContractPerpetual` vs `ContractUSDTFutures` via symbol suffix.
- [x] Adapter Bybit implementa a normalização canônica per ADR-0021
  e emite `CanonicalInstrument` via `Normalize` — entregue em H-7.b
  como packages `bybits`/`bybitf` no house pattern
  (`parseBybit*Symbol` + `Normalize`, Decisão #5 (A); o naming
  literal `ToCanonical`/`FromCanonical` ganha erratum de
  equivalência na revisão da H-6.f.2, junto com os adapters
  Binance que usam o mesmo shape desde H-6.a).
- [ ] Todos os domain types migrados de `Symbol string` para
  `Instrument CanonicalInstrument` (ObservationTrade,
  EvidenceCandle, Signal, Decision, Strategy, Risk, Pairing's
  RoundTrip/Leg, Effectiveness's Attribution, etc.).
- [ ] Application layer + actors + samplers consomem
  `CanonicalInstrument`; internal builders migram local
  `symbol string` → `instrument CanonicalInstrument`.
- [ ] ClickHouse migration nova adicionando canonical columns;
  writer dual-writes; analytical client reads canonical-preferred
  com fallback legacy; cutover runbook em
  `docs/operations/runbooks/clickhouse-canonical-migration.md`
  (criação prevista em H-6.d).
- [ ] NATS subject composition migrada para canonical form
  (H-6.e, opção (i) decidida 2026-06-10 — token via
  `SubjectToken()` único, analyzer `check subjects` no gate);
  KV partition keys + contrato HTTP de leitura migrados em
  H-6.e.2 (critério #2 do ADR-0021 fecha literalmente em e.2,
  per erratum 2026-06-10).
- [x] raccoon-cli `check instruments` (H-6.a) e `check venue-parity`
  (H-7.a, gate step 11) integrados em `make verify`.
- [ ] ADR-0021 promovido a `Accepted` no commit final de H-6.f.2
  (todos critérios literais satisfeitos; gate temporal pós-TTL
  ~2026-08-26 + verificação operacional — erratum 2026-06-11).
- [x] ADR-0022 promovido a `Accepted` no commit final de H-7.b
  (2026-06-12; cross-venue parity provada com Binance + Bybit — 4
  declarações de capabilities introspectáveis, analyzer no gate,
  canário integration vs NATS vivo).
- [ ] PROGRAM-0004 transita para `Closed` na entrega final de
  H-7; entrada Changelog correspondente.

---

## ADRs governantes

| ADR | Escopo | Status no início da Fase | Promovido por |
|-----|--------|--------------------------|----------------|
| 0021 | Canonical instrument & venue model | Proposed (entregue em H-2) | H-6.f.2 (após todos critérios literais; gate pós-TTL ~2026-08-26) |
| 0022 | Multi-venue normalization policy | Proposed (entregue em H-2) | **H-7.b ✓ (Accepted 2026-06-12** — framework em H-7.a, Bybit + promoção em H-7.b; 6 critérios verificados na seção Status do ADR) |

Nenhuma ADR nova esperada nesta Fase. Se durante as sub-ondas
surgir necessidade arquitetural não coberta, P6 (pause-and-report)
e nova ADR sob `decisions/0026+`.

---

## Riscos

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Cascade de `.Symbol` migration explode (342 refs em 106 files) e produz PRs irrevisáveis | Alto — onda gigante invertida pelo H-6.a pré-flight | Re-escopo em 6 sub-ondas serializadas (H-6.a–H-6.f). Cada sub-onda toca subsystema bounded. PRD declara sub-onda sequencing policy estrita. |
| Sub-ondas paralelas geram merge conflicts no mesmo arquivo | Alto — fragmentação custosa de revisão | Sub-onda sequencing policy estrita: próxima abre apenas após merge da anterior em `main`. |
| ADR-0021 critério #2 ("all domain-layer call sites migrated") sob pressão de aceitar erratum loophole | Alto — perda de disciplina P7 | Decisão firme: critério #2 fica literal. Sem erratum até H-6.f. ADR permanece `Proposed` durante toda a migração; promoção é evento atômico no final. |
| ClickHouse schema migration introduz data ordering risk se bundled com domain refactor | Médio — rollback complexo | Erratum REAL ao critério #4 (split #4a/#4b). #4a é writer-side adapt em H-6.a (zero schema change). #4b é migration dedicada em H-6.d, sequenced after H-6.c e before H-6.f. |
| NATS subject composition decision ad-hoc em H-6.e sem critério explícito | Médio — deferral silencioso | H-6.e protocolo: pause-and-report como primeiro ato. Opção (i) migrar OU (ii) deferral com segundo erratum REAL. Sem decisão silenciosa. |
| `binancef.Normalize` assume todos os símbolos são perpetual; quebra quando delivery dataset entrar | Médio (não imediato; futuro silencioso) | Pattern detection em H-6.a commit 5: regex `_YYMMDD` suffix discrimina `ContractUSDTFutures` vs `ContractPerpetual`. 2 testes cobrindo ambos casos. |
| `check instruments` analyzer over-restrictive — bloqueia adapter legítimo que não declarou policy | Baixo | Declarative via `policies/adapters.toml`; reviewer adiciona explicitamente. Mesmo pattern de `check metrics` de H-5. |
| H-7 Bybit adapter implementa parity-by-default que não generaliza para Coinbase/Hyperliquid futuros | Médio | ADR-0022 define a política; H-7 prova com Bybit. Generalization para outros venues é onda futura per-venue, não em H-7. |
| Promoção de ADR-0021 em H-6.f tentando rush últimos critérios | Médio | P7 absoluto: critérios literais. Reviewer + agente verificam um a um antes do commit que flip Status. |

---

## Referência ao raccoon (sem cópia)

Capacidades validadas no raccoon que informam (sem migrar para)
PROGRAM-0004. Lidas em pré-flight de H-6.a com justificativa
explícita:

- `internal/core/marketdata/domain/instrument_identity.go` +
  `instrument_metadata.go` + `market_type.go` — pattern
  `InstrumentMetadata{VenueSymbol, CanonicalSymbol, BaseAsset,
  QuoteAsset, MarketType}`. Foundry diverges: (a) `CanonicalInstrument`
  é struct rooted em `Base + Quote + Contract` (Venue carrega no
  envelope, não dentro da instrument identity); (b) canonical
  format é `BASE/QUOTE-CONTRACT` (4-way contract discrimination,
  raccoon faz 3-way `SPOT/USD_M/COIN_M` sem distinguir perpetual);
  (c) tipos fortes `BaseAsset`/`QuoteAsset` (raccoon usa `string`
  plano); (d) lowercase enum naming (raccoon uppercase).
- `internal/adapters/exchange/binance/parser.go` — pattern
  `ParseMessage(data, recvAt) (IngestRequest, bool, *Problem)`.
  Foundry mantém signature simpler (`Normalize(raw, symbol) (Event,
  *Problem)`); H-6.a só muda tipo do field interno do output, não
  signature externa.
- `internal/adapters/exchange/common/common.go` — helpers
  `NormalizeSide`, `NormalizeMarketType`. Foundry **não cria**
  `common/` package em H-6.a — single venue family currently;
  premature abstraction. Quando H-7 (Bybit) revelar duplicação
  real, refactor para shared package.
- `docs/adrs/ADR-0011-marketdata-binance-canonical-instrument-and-event-mapping.md`
  — single-venue-Binance scope; foundry ADR-0021 generaliza para
  multi-venue desde início.

Anti-padrão: copiar `instrument_identity.go` literal. Reescrever
no foundry com tipos fortes per ADR-0021 spec.

---

## Evidence

- [`../decisions/0021-canonical-instrument-and-venue-model.md`](../decisions/0021-canonical-instrument-and-venue-model.md)
  — ADR governing instrument model; erratum landed in H-6.a
  commit 0.
- [`../decisions/0022-multi-venue-normalization-policy.md`](../decisions/0022-multi-venue-normalization-policy.md)
  — ADR governing cross-venue parity policy; promotes in H-7.
- [`PROGRAM-0001-foundation.md`](PROGRAM-0001-foundation.md) — Fase
  anterior; entregou ADRs 0021 + 0022 em H-2 como `Proposed`.
- [`PROGRAM-0002-wire.md`](PROGRAM-0002-wire.md) — Fase anterior;
  entregou envelope `instrument` field (string per ADR-0017).
- [`PROGRAM-0003-observability.md`](PROGRAM-0003-observability.md)
  — Fase paralela ativa; metrics instrumentadas que multi-venue
  dashboards consumirão.
- [`../../CLAUDE.md`](../../CLAUDE.md) → "Fase Harvest" — protocolo
  P1–P9 canônico.
- [`../RESUMPTION.md`](../RESUMPTION.md) → "Fase Harvest" — state
  sentinel.

---

## Changelog

- **2026-06-12 (closure H-7.b)** — Adapter Bybit entregue em 7
  commits; **ADR-0022 → `Accepted`** no commit final (6 critérios
  verificados um a um na seção Status do ADR; divergência de layout
  bybits/bybitf vs o path único "bybit/" esboçado registrada lá —
  o split preserva a bijeção do venueSourceContract, Decisão #3).
  Packages bybits (spot) + bybitf (linear perpetual): parser
  tri-state (frames de controle v5 skipados), Normalize em batch
  (data[]; BuyerMaker = taker S=="Sell"), delivery rejeitado no
  parser (gate G10), WSClient subscribe-frame + ping app-level.
  Wiring completo (+Venue enum, +switch ingest, +registry,
  +allowlist, +união gateway 4 venues). Canário integration vs
  NATS vivo prova batch-não-colapsa-no-dedup e roteamento dos dois
  sources — duas lições do draft corrigidas e comentadas (payload
  é CBOR; TradeIDs fixos eram deduplicados no rerun dentro da
  janela de 2min). RUNTIME.md ganha "Venue ingest sources" + fix
  do exemplo stale de partition key (pré-e.2); CLAUDE.md e
  RESUMPTION N4 re-escopados para "no multi-exchange EXECUTION
  surface" (observação é multi-venue desde aqui). **ADR-0021
  permanece `Proposed`** (promoção em H-6.f.2 pós-TTL). Próxima:
  **H-7.c (expiry/G10)** após merge.

- **2026-06-12 (abertura H-7.b)** — H-7.a mergeada (PR #45 em
  `main` em `8d5bedd`) destrava H-7.b: adapter Bybit per Decisões
  #2/#3/#5 da abertura de H-7. Pré-flight da sub-onda confirmou:
  spawn do ingest é config-driven (supervisor cria scope por
  `Target.Source`; único dispatch hardcoded é o switch do
  websocket_actor) e os subjects do observation stream são
  wildcard (`observation.events.market.>`) — sem mudança de
  registry NATS. Surface enumerada: Venue enum (+bybit/+bybitf),
  2 packages novos (bybits/bybitf, espelho da família Binance),
  switch do websocket_actor (+2 cases com loop por `data[]`),
  `venueSourceContract` (+2), `adapters.toml` (+2), união do
  gateway (+2), RUNTIME.md, CLAUDE.md non-features. Achado: o
  RUNTIME.md carrega exemplo stale de partition key
  (`binance_spot.btcusdt.60`, shape pré-e.2) — fix no closure.
  **Promoção ADR-0022 → Accepted no commit final desta sub-onda**
  se os 6 critérios literais fecharem (verificação um a um).

- **2026-06-12 (closure H-7.a)** — Capabilities framework entregue
  em 8 commits (0–6, com 5a/5b). Contrato `Capabilities` em
  `application/ports` (mea culpa estrutural registrado: o pré-flight
  assumiu interfaces→adapters permitido; arch-guard acusou e o
  contrato moveu para o home dos ports — mesmo package do
  VenuePort); retrofit binances/binancef; guard R3 + counter no
  ingest; `GET /venues/capabilities` (boot_test 60→61); analyzer
  `check venue-parity` (gate step 11, 8 unit tests, live 6/6).
  Segundo mea culpa menor: binaries.toml não precisa de entry para
  counter (allowlist é de exposição /metrics). **ADR-0022 permanece
  `Proposed`** — promoção atômica em H-7.b com o adapter Bybit
  (critério #1, único pendente dos 6).

- **2026-06-12 (abertura H-7 / H-7.a)** — H-6.f.1 mergeada (PR #44
  em `main` em `5195f8e`, 2026-06-12) destrava H-7 per erratum.
  Pré-flight da abertura produziu wave prompt auditado pelo owner;
  **cinco decisões registradas** (ver "Sub-ondas H-7" acima):
  split a/b/c; Bybit spot+linear perpetual em observação apenas;
  sources `bybits`/`bybitf` (bijeção do venueSourceContract);
  expiry como campo opcional com coluna CH deferida; house pattern
  no adapter com erratum de equivalência ToCanonical/FromCanonical
  agendado para a revisão da f.2. Achados do pré-flight que
  moldaram as decisões: bijeção source→contract no binding
  registry; segment model do execute é Binance-only por construção
  (S400); Bybit v5 usa subscribe-frames + `data[]` array + taker
  side `S` (shape de WSClient distinto do modelo URL-stream da
  Binance); CLAUDE.md "No multi-exchange surface" precisa de
  update quando o Bybit ship (H-7.b). **H-7.a aberta** (branch
  `feat/h-7-a-capabilities-framework`): capabilities framework
  ADR-0022 R1–R4 sem venue novo. **ADR-0022 permanece `Proposed`**
  (promoção atômica em H-7.b).

- **2026-06-11 (closure H-6.f.1)** — Entrega completa em 7 commits.
  Regressão da auditoria FIXADA: `instrument.FromSubjectToken`
  (premissa de não-ambiguidade verificada mais forte que a
  declarada — assets só admitem A-Z0-9, além de contract types sem
  underscore; lock-in dos dois lados) + **6º/último
  `instrumentFromBinding` deletado** (grep: zero call sites/
  definições) + canários unit não-zero + entry do anti_patterns
  flipped severity=error com exception list vazia. Dedup keys:
  recontagem confirmou 11 sites declarados / **9 com token de
  instrument** (caveat previsto na Decisão #4) — todos migrados;
  janela de dedup verificada = **2min** (default JetStream, sem
  `Duplicates` explícito); analyzer `[dedup]` (P5) com 6 unit
  tests, 7 composers + 12 blocks varridos no live run. Migration
  runner: `SplitStatements` ;-aware + 14 shapes reais pinned.
  Test-hardening: **G8 resolvido** (FixedClock, -count=20 PASS,
  entrada movida para Recently resolved); G7/G9 investigados e NÃO
  absorvidos (G7 = refactor de infra de teste, pause trigger da
  onda; G9 = ambiental sob carga de CI) — rationale no registry.
  **ADR-0021 permanece `Proposed`** (promoção em f.2 pós-TTL).
  Pós-merge: H-7 destravada ∥ f.2 agendada ~2026-08-26.

- **2026-06-11 (abertura H-6.f.1)** — Pause-and-report da abertura
  de H-6.f revelou **regressão silent-zero na auditoria**: audit
  bundles com `Instrument` zerado desde o merge de H-6.e.2 —
  `audit_session.go:170` reconstrói via
  `instrumentFromBinding(e.Source, e.Symbol)`, que exige sufixo
  `USDT` venue-native, mas `e.Symbol` passou a carregar o token
  canônico `base_quote_contract` pós-cutover das KV keys.
  **Decisão #1 (owner, opção A)**: H-6.f dividida em **f.1**
  (agora — fix da regressão + deferrals não-TTL-gated: dedup keys,
  migration runner multi-statement, test-hardening G8) e **f.2**
  (TTL-gated ~2026-08-26 — flip do WHERE, deleções de
  helpers/VenueSymbol, postura da coluna legacy, verificação
  operacional, **promoção ADR-0021**). **Decisão #2**: erratum de
  sequenciamento — H-7 destrava após merge de f.1, cadeia
  `e → e.2 → f.1 → {H-7 ∥ f.2}` (ver "Erratum de sequenciamento"
  na sequencing policy). Decisões #3–#7 da abertura registradas no
  wave prompt: fix via `instrument.FromSubjectToken` + deleção do
  último `instrumentFromBinding`; dedup keys canonicalizam na f.1
  com analyzer `[dedup]` (P5); migration runner statement-split;
  G8 fix obrigatório via FixedClock; convenção wave-row (commit 0
  fecha a linha anterior). **ADR-0021 permanece `Proposed`.**

- **2026-06-11 (closure H-6.e.2)** — Entrega completa do pacote B em
  6 commits (bundle atômico de 231 arquivos no commit 2). **Critério
  #2 do ADR-0021 literalmente satisfeito** per erratum (subjects em
  H-6.e + keys/contrato aqui); promoção segue atômica em H-6.f.
  Mea culpa do executor registrado: a enumeração da abertura
  declarou as chaves KV "parser-free"; `parsePartitionKey`
  (`query_responder_actor.go`) é um parser que o sweep não viu —
  formato-compatível com o token novo (sem pontos), pacote B
  inalterado, claim corrigida. `DefaultVerificationScope` migrou
  para source real (`binances`/`btcusdt`) — o default antigo
  ("BTCUSDT", sem venue) era case-mismatched contra a coluna
  ClickHouse lowercase e tornaria os checks Skip. Canários: key
  shape literal vs NATS vivo PASS; readers d.2 via ports canônicos
  6/6 PASS vs ClickHouse vivo.

- **2026-06-11** — H-6.e fechada (PR #42 mergeada em `main` em
  `f8543b7`, 2026-06-10); **H-6.e.2 aberta** com decisões do owner
  registradas (**pacote B**): contrato HTTP → trio canônico
  `base/quote/contract`; clients → `CanonicalInstrument`; KV keys
  write+read → `{source}.{SubjectToken()}.{timeframe}`; ClickHouse
  WHERE inalterado com valor derivado (`LegacyFilterValue()`
  transitório, sunset em f com o flip do WHERE pós-TTL); analyzer
  `[keys]` no mesmo PR; **expiry deferido a H-7**. Pré-flight da
  abertura recontou a superfície (ground truth via grep): 31 sites
  `parseQueryKeyParams` + 9 extrações diretas de symbol, **8** client
  packages (não 7), 9 builders ClickHouse com param symbol, 5
  PartitionKey — divergências vs. a enumeração da abertura eram
  unidades de contagem do agente, não escopo novo. Cross-check
  corrigiu pointer documental: `parseQueryKeyParams` vive em
  `handlers/evidence.go`, não `handlers/common.go` (HTTP-API.md
  corrigido na onda).

- **2026-06-10** — H-6.e aberta; pause-and-report executado como
  primeiro ato; **owner decide opção (i)** (migrar subjects para
  forma canônica). Enumeração D confirma zero parsers do token de
  symbol → cutover atômico sem dual-publish. **Sub-onda H-6.e.2
  criada** (split da Decisão #2): KV partition keys + contrato
  HTTP de leitura, com extensão do analyzer e o débito de
  modelagem do expiry (CanonicalInstrument sem campo expiry —
  delivery futures de expiries distintos colidem em identidade
  canônica; mea culpa do arquiteto registrado: a prescrição
  original assumia expiry no modelo). Sequenciamento atualizado:
  e → e.2 → f, com H-6.f bloqueando em e.2. Errata da mesma data:
  ADR-0009 (gramática do token canônico) e ADR-0021 critério #2
  (fechamento literal desloca para e.2). Imprecisão da linha
  original de H-6.e corrigida ("Instrument.Symbol() (derived
  form)" → era VenueSymbol-derived).

- **2026-05-28** — H-6.d.2 fechada. **Sub-onda H-6.d encerrada**
  (H-6.d.1 + H-6.d.2 ambas mergeadas em `main`). Entregas H-6.d.2:
  **ClickHouse reader-side cutover para canonical columns com
  legacy fallback** via 4 commits. Critério #4b do ADR-0021
  erratum agora completo end-to-end (writer-side em H-6.d.1 +
  reader-side em H-6.d.2).

  **Commit 1** — Novo helper
  `internal/adapters/clickhouse/canonical_instrument_columns.go`
  com `ErrLegacyRow` sentinel exportada +
  `instrumentFromCanonicalColumns(base, quote, contract) →
  (CanonicalInstrument, error)`. Sentinel pattern (`errors.Is`)
  per Decisão #3: idiomatic Go discrimination entre
  expected-legacy-row case e validation regressions em rows
  com canonical populados mas inválidos (e.g. unknown contract
  type — devem propagar, não cair silentemente em fallback).
  Validação delegada a `instrument.New` (gate autoritativa
  per ADR-0021). 4 unit tests / 9 sub-cases:
  - All-empty triple → ErrLegacyRow.
  - Cada single empty field → ErrLegacyRow.
  - Valid (spot / perpetual / usdtfutures) → CanonicalInstrument
    com matching Contract e identity round-trip.
  - Invalid contract type em populated row → non-ErrLegacyRow
    error (regression-shape guard).

  **Commit 2** — Reader dual-path migration. 7 reader files / 13
  instrument-resolution sites / 13 SELECT column lists
  atualizados uniformemente. Pattern uniform através dos 13
  sites (validated em pré-flight 3):
  ```go
  inst, instErr := instrumentFromCanonicalColumns(base, quote, contract)
  if instErr != nil {
      inst, instErr = reconstructInstrumentFromLegacy(src, sym)
      if instErr != nil {
          r.logger.Warn(...)
      }
  }
  ```
  Per-table query builders (8 builders): BuildCandleQuery /
  BuildSignalQuery / BuildDecisionQuery / BuildStrategyQuery /
  BuildRiskQuery (1 cada) + BuildExecutionQuery /
  BuildLifecycleHistoryQuery / BuildExecutionListQuery (3 em
  execution_reader.go). Composite reader inline SELECTs (5):
  querySignalByCorrelation / queryDecisionByCorrelation /
  queryStrategyByCorrelation / queryRiskByCorrelation /
  queryExecutionByCorrelation. Cada SELECT insere `base, quote,
  contract` após `symbol`, alinhando com o column ordering
  emitido pelos H-6.d.1 writer mappers. Scan signatures ganham
  &base, &quote, &contract pointers. 8 test files atualizados:
  expectedCols slices estendidas, column counts bumped (candle
  12→15, signal 8→11, decision 12→15, strategy 11→14, risk
  13→16, execution 16→19) + s453a_lifecycle_history_test +
  s454a_operational_list_queries_test.

  **Commit 3** — Reader canary integration test
  `canonical_columns_reader_integration_test.go` (~714 LoC,
  `//go:build requireclickhouse`, package `clickhouse_test`).
  6 tests / 18 subtests (canonical_path / fallback_path /
  mixed_state per table). Per-table DDL constants
  duplicated from writerpipeline canary (Go _test packages não
  podem cross-import). Helper `skipUnlessClickHouseReader`
  mirrors `skipUnlessClickHouseCanonical`. **mixed_state subtest
  é a prova literal da Resolução 1**: insere uma row canonical-
  populada (ETH/USDT/spot) + uma legacy-shape (`binances ethusdt`,
  canonical columns vazias) na mesma tabela, query única retorna
  ambas, cada uma resolve via path próprio, ambas produzem
  CanonicalInstrument equivalente (ETH/USDT/spot). Fixture
  ETH/USDT/spot vs. binances→BTC/USDT/spot default disambiguates
  o canonical path do fallback (silent regression em
  instrumentFromCanonicalColumns surge como canonical row
  voltando BTC/USDT em vez de ETH/USDT).

  **Resolução 1 — Helper retention through 90-day TTL preserved**:
  `reconstructInstrumentFromLegacy` permanece em
  `candle_reader.go:150` per Resolução 1 documentada em H-6.d.1
  Changelog. **NÃO** é deletado em H-6.d.2 — deletion deferida
  para H-6.f post-TTL operational verification.
  Correctness-driven: legacy rows persistem até MergeTree TTL
  expirar (~90 dias post-2026-05-27 H-6.d.1 merge → ~2026-08-25);
  reader DEVE reconstructar Instrument durante esse window. O
  mixed_state subtest é a prova permanente de que durante o
  window ambas shapes coexistem corretamente.

  **H-6.f scope expansion preserved** (registered durante
  H-6.d.1 closure, atualizado para refletir H-6.d.2 progress):
  1. **Helper deletion**: `reconstructInstrumentFromLegacy`
     + `executionclient/instrument_binding.go` (post 90-day TTL
     window).
  2. Migration runner multi-statement support (deferred from
     H-6.d.1 Decisão #1).
  3. Exception list shrinking: 7 ClickHouse entries em
     `anti_patterns.toml` (currently tagged "H-6.d helper
     removal") removed após cutover + TTL window passar.
  4. Operational verification post-TTL: confirmar legacy-only
     rows expired; canonical-only reads PASS sem fallback;
     promover ADR-0021 → `Accepted` per critério #2 + #4b
     literal satisfaction.

  **Métricas H-6.d.2**: 4 commits, 1 new helper + 1 sentinel
  error + 7 readers migrated + 13 SELECTs + 13 Scan sites + 1
  new test file (714 LoC, 6/18 subtests) + 8 test files
  updated. Pre-push validation: `make verify` GREEN +
  `raccoon-cli --profile ci` GREEN + reader canary 18/18 PASS
  contra live ClickHouse.

  **Marco**: H-6.d.2 fecha **critério #4b end-to-end do
  ADR-0021 erratum** — writer populates canonical columns
  (H-6.d.1) + reader prefers canonical com legacy fallback
  (H-6.d.2). ADR-0021 critério #2 (zero source-string-based
  reconstruction em production) **ainda não literalmente
  satisfeito** — `reconstructInstrumentFromLegacy` retained
  através do TTL window, `executionclient/instrument_binding.go`
  remanesce. Helper deletion + ADR-0021 promotion atómicos em
  H-6.f post-TTL.

  **Próxima sub-onda destravada após merge**: H-6.e — NATS
  subject composition decision (primeiro ato: pause-and-report
  obrigatório). Sub-onda sequencing policy estrita: H-6.e abre
  branch APENAS após merge desta PR (H-6.d.2) em `main`.

- **2026-05-27** — H-6.d.1 fechada. **Sub-onda H-6.d introduzida**
  (sub-divisão de H-6.d em H-6.d.1 + H-6.d.2 post-pré-flight —
  positional-INSERT cascade em integration tests + tagged-build
  invisibility de 3 meses tornou monolithic H-6.d impraticável).
  Entregas H-6.d.1: **ClickHouse schema migration + writer
  canonical column population end-to-end** via 4 commits.

  **Commit 1** — 6 migrations adicionadas
  (`008_add_canonical_columns_evidence_candles.sql` →
  `013_add_canonical_columns_executions.sql`), uma por
  Instrument-bearing table. Cada migration: `ADD COLUMN IF NOT
  EXISTS base/quote/contract LowCardinality(String) DEFAULT ''
  AFTER symbol/base/quote`. Idempotent + reversible per header
  contract. Split per-table after Decisão #1 — initial
  `008_add_canonical_columns.sql` multi-statement FAILED contra
  ClickHouse (code 62, "Multi-statements are not allowed").
  Opção (A) chosen: 6 separate files. Opção (B) (migration runner
  enhancement para parse-and-execute statement-by-statement)
  declared scope creep e **deferred para H-6.f scope expansion**
  alongside helper deletion + exception list shrinking.

  **Commit 2** — codegen self-consistency atomic bundle. 14 YAML
  family specs ganham 3 canonical columns na string `writer.columns`
  (sed-driven uniform `base, quote, contract` inserts post-symbol).
  14 golden snapshots regenerados via `codegen generate <spec>
  pipeline_entry`. `codegen/render_test.go` 6 inline `Columns:`
  strings updated. `cmd/writer/pipeline.go` 17 INSERT SQL strings
  updated (14 codegen + 3 manual: squeeze_breakout_entry,
  venue_fill, venue_rejection). `writerpipeline/support.go` 8
  mappers (`mapCandleRow`/`mapSignalRow`/`mapDecisionRow`/
  `mapStrategyRow`/`mapRiskRow`/`mapExecutionRow`/`mapVenueFillRow`/
  `mapVenueRejectionRow`) each appends
  `string(x.Instrument.Base), string(x.Instrument.Quote),
  string(x.Instrument.Contract)` after `VenueSymbol()`. Test row
  position shift cascade: ~41 row[N] + 6 column count updates em
  `support_test.go`, 70 bare row[N] + 43 multi-letter Row variable
  shifts em `behavioral_roundtrip_test.go` (highRow/lowRow/ctRow/
  ptRow/decRow/stratRow/riskRow regex pass). Atomic bundle pattern
  — codegen YAML/golden/pipeline.go/mappers/tests **must move
  together** by self-consistency invariant (golden snapshot
  regen would fail if YAML/pipeline.go diverged).

  **Commit 3a** — Integration fixture pre-flight migration. 34
  positional INSERTs em `composite_reader_integration_test.go`
  convertidos para explicit column lists (5 unique templates per
  table: candle/signal/decision/strategy/risk/execution). Sem
  explicit columns, schema migration teria quebrado fixture
  inserts silenciosamente. Pulled-forward into commit 3a por
  cascade analysis durante commit 2 review. 20 pre-H-6.b drift
  fixes: `.Symbol` → `.VenueSymbol()` em
  `composite_reader_integration_test.go` (Signal/Decision/Strategy/
  Risk/Execution) + 3 em `live_execution_analytical_test.go`
  (results[i].Symbol + r.Symbol). **Tagged-build drift discovery**:
  files com `//go:build requireclickhouse` são invisíveis ao
  default `make verify` — pre-H-6.b drift survived 3 months
  undetected. Decisão #2 (A): explicit column list é arquiteturalmente
  superior independent of schema migration.

  **Commit 3b** — Writer canonical population canary.
  `Client.Exec(ctx, query, args)` adicionado em
  `internal/adapters/clickhouse/client.go` para DDL via native
  protocol (clickhouse-go/v2 `Query` returns EOF on DDL como
  CREATE/DROP/ALTER). Novo
  `internal/adapters/clickhouse/writerpipeline/canonical_columns_integration_test.go`
  (~527 LoC, `//go:build requireclickhouse`, package
  writerpipeline) com 6 tests / 1 per table:
  `TestWriter_PopulatesCanonicalColumns_EvidenceCandles/Signals/
  Decisions/Strategies/RiskAssessments/Executions`. Cada test
  reseta tabela (DROP + CREATE com schema post-H-6.d.1 inline),
  insere 1 row via writer mapper, queries
  `SELECT base, quote, contract FROM <table>`, asserts
  canonical values are populated (não vazios). Helpers:
  `skipUnlessClickHouseCanonical` + `resetTable` +
  `queryCanonicalColumns` + `assertCanonicalColumns`.

  **Resolução 1 — Helper retention through 90-day TTL**:
  `composite_reader.go` 5 callers + 8 sister-site readers de
  `reconstructInstrumentFromLegacy` MANTÊM warn-and-emit-zero
  fallback até H-6.f. Razões: (i) MergeTree TTL de 90 dias
  retire legacy rows (rows pre-H-6.d.1) gradualmente; durante
  TTL window readers DEVEM aceitar both shapes
  (canonical-populated AND legacy-only); (ii) H-6.d.2 reader
  cutover é canonical-preferred-with-fallback, não helper
  removal; (iii) helper deletion + exception list shrinking
  (~7 ClickHouse entries em `anti_patterns.toml`) consolidated
  em H-6.f post-TTL operational verification. **Helper retention
  é correctness-driven, não convenience**: deletion durante TTL
  window quebraria reads de legacy rows.

  **H-6.f scope expansion** (registrado durante H-6.d.1
  closure):
  1. Helper deletion: `executionclient/instrument_binding.go`
     + `reconstructInstrumentFromLegacy` (post 90-day TTL).
  2. Migration runner multi-statement support (deferred from
     H-6.d.1 Decisão #1 — parse-and-execute statement-by-
     statement em `cmd/migrate`).
  3. Exception list shrinking: 7 ClickHouse entries em
     `anti_patterns.toml` (currently tagged "H-6.d helper
     removal") removed após cutover + TTL window passar.
  4. Operational verification post-TTL: confirm legacy-only
     rows expired; canonical-only reads PASS sem fallback;
     promote ADR-0021 → `Accepted` per critério #2 + #4b
     literal satisfaction.

  **Lessons registered**:

  - *Positional INSERT pre-flight discipline*: schema migrations
    must scan for positional INSERTs em integration fixtures
    BEFORE migration commits. Standard pre-flights (production
    code grep, .Symbol audit) miss tagged-build test files.
    Pré-flight checklist for schema changes future-onward:
    `grep -r "INSERT INTO <table> VALUES" --include="*_test.go"`
    + `grep -r "//go:build requireclickhouse"` enumeration.

  - *Tagged-build drift detection*: files com
    `//go:build requireclickhouse` (e similar tags) are
    invisible to default `make verify`. Pre-H-6.b drift survived
    3 months undetected. Mitigation candidates (registered as
    H-6.f deferral candidate): (a) `make verify-tagged` step
    explicitly building each tag enumeração, (b) CI matrix
    expansion, (c) raccoon-cli analyzer scanning tagged files
    against domain types policy.

  - *Codegen self-consistency invariant*: YAML specs + golden
    snapshots + stamped artifacts em pipeline.go + mappers +
    tests **must move atomically**. Splitting commit 2 into
    "codegen-only" + "writer-only" would produce intermediate
    state where regen would FAIL (golden snapshot diff vs.
    pipeline.go INSERT shape). Bundle pattern reaffirmed
    (precedent: H-6.c.1 commit 6 atomic actor-cascade bundle).

  ADR-0021 row em TRUTH-MAP atualizada: canonical columns
  populated end-to-end pelo writer; reader migration deferred
  para H-6.d.2; helper retention through TTL declared.
  ADR-0021 permanece `Proposed`; promotion gated em literal
  critério #4b satisfaction (reader cutover + helper deletion
  + operational verification), atómico em H-6.f.

  **Próxima sub-onda destravada após merge**: H-6.d.2 — reader
  canonical-preferred cutover com fallback window through
  90-day TTL.

- **2026-05-27** — H-6.c.2 fechada. **Sub-onda H-6.c
  encerrada** (H-6.c.1 + H-6.c.2 ambas mergeadas em `main`).
  Entregas H-6.c.2: application-layer pass-through migration
  para execute scope + uniformização da error-handling em
  ClickHouse `reconstructInstrumentFromLegacy` callers + DTO
  string_filter declaration. 8 commits.

  Helper migration progress:
  - `instrumentFromBinding`: 5 de 6 packages eliminated
    (signal/decision/strategy/risk em H-6.c.1 commits 7a-7d +
    execution em H-6.c.2 commit 5). Apenas
    `executionclient/instrument_binding.go` remanesce para
    H-6.f (blocked by LifecycleEntry contract migration).
  - Testnet adapters (`binance_spot_testnet_adapter.go:391`,
    `binance_futures_testnet_adapter.go:395`) migrados em
    commit 4 para usar `BindingTarget.Instrument()` boundary
    helper com warn-and-emit-zero fallback.

  ClickHouse uniformization (commit 2): 5 silent error-discard
  sites em `composite_reader.go` (linhas 188/243/302/360/423)
  convertidos para warn-and-emit-zero pattern, matching os 8
  existing sister sites em
  `candle/decision/execution/risk/signal/strategy_reader.go`.
  All 13 `reconstructInstrumentFromLegacy` callers agora
  uniformes; helper removal scheduled para H-6.d via canonical
  column schema migration. Partial-chain-assembly contract
  preservado (zero Instrument continua propagando para
  manter stage population).

  Application-layer DTO declaration (commit 3): `ReviewTransform`
  declared como `string_filter` em `policies/domain_types.toml`
  via novo entry `[domain_types.review_transform]` com package
  path `internal/application/analyticalclient` (schema accepts
  arbitrary package paths beyond `internal/domain/*`). Zero
  production code change — o cascade
  decision.Decision.Instrument → d.VenueSymbol() →
  ReviewTransform.Symbol → DecisionTriageItem.Symbol já estava
  no post-canonical state desde H-6.b. Inline godoc adicionado
  em ambos os structs documentando a string-filter semantics.

  37f8ddd canary explícito (commit 6): novo arquivo
  `internal/actors/scopes/execute/execute_venue_adapter_canary_test.go`
  com 2 tests / 2 passes lockando o contract:
  - `TestPaperOrderEvaluator_PreservesInstrument_WithSyntheticSource`
    (unit shape, 0.00s).
  - `TestStrategyConsumerActor_PreservesInstrument_WithSyntheticSource`
    (actor shape, 0.02s).
  Sem dependência de NATS, sem integration tag — runs em todo
  `make verify` para fast feedback.

  Cross-scope test stragglers (commit 5): 8 sites missed by
  H-6.c.1 commit 6's Python migration script — pulled into
  H-6.c.2 cleanup commit per the straggler-fix pattern
  established at H-6.c.1 7b/7c/7d. Distribution:
  - 1 derive: `s470_lineage_causality_test.go:318`.
  - 2 risk: `risk_scaling_test.go:720, 762`.
  - 6 integration-tagged: writerpipeline (1) +
    natsexecution (4) + execute live_consumer_flow (1).
  **Discovery pattern reinforced** (H-6.c.1 lesson 13):
  make verify masks integration-tagged build failures. Without
  explicit `go test -tags=integration -run DOES_NOT_EXIST`
  check, the 6 integration stragglers would have shipped
  broken to CI.

  anti_patterns.toml exception list shrunk 11 → 8 entries
  (commit 7 — net -3 execution package entries). Kept: 1
  executionclient (H-6.f scope) + 7 ClickHouse readers (H-6.d
  scope; composite_reader.go re-tagged from "H-6.c.2 treatment"
  to "H-6.d helper removal").

  ADR-0021 row em TRUTH-MAP atualizada: 5/6 helper-elimination
  state + new boundary-helper wiring em execute scope + new
  canary test anchor + uniform ClickHouse pattern. ADR-0021
  permanece `Proposed`; promotion gated em literal critério #2
  satisfaction, atómico em H-6.f.

  ## H-6.f Architectural debt — QueryOrder port refactor candidate

  Option (a) cascade analysis (recorded during H-6.c.2 Decisão
  #2 verification):

  - 5 production files (port + segment_router +
    binance_spot_testnet_adapter + binance_futures_testnet_adapter
    + post200_reconciler).
  - 7 test files / ~15 sites (post200_reconciler_test +
    s405_spot_venue_acceptance_fill + s422_futures_venue_connectivity_fill +
    s423_futures_rejection_partial_fill +
    s416_futures_venue_acceptance_fill + 2 lifecycle tests).
  - Total: 12 files / >8 threshold (sub-onda H-6.c.2
    exceeded).

  Architectural rationale: `QueryOrder(ctx, clientOrderID,
  symbol string)` takes symbol string, forcing testnet adapters
  to reconstruct Instrument via boundary helper at adapter
  layer. The architecturally correct shape is `QueryOrder(ctx,
  clientOrderID string, instrument instrument.CanonicalInstrument)`
  — caller (which holds Intent.Instrument) passes Instrument
  directly; adapter receives canonical type and uses
  `inst.Symbol()` for venue-side mapping. Eliminates the
  residual reconstruction in adapter layer entirely.

  Current state (post-H-6.c.2): testnet adapters use
  `BindingTarget.Instrument()` with warn-and-emit-zero fallback.
  Removes the `instrumentFromBinding` helper file from
  execution package but keeps reconstruction in adapter layer.
  Eight-or-fewer test sites would migrate cleanly per uniform
  pattern; 12 total files exceeded the sub-onda threshold for
  H-6.c.2 scope.

  H-6.f candidate refactor: port signature migration to
  Instrument. Cascade out of H-6.c.2 scope but tractable as
  dedicated H-6.f sub-task alongside `executionclient` +
  LifecycleEntry migration. Recorded here to prevent
  "rediscover this in H-6.f" when the time comes.

  **Próxima sub-onda destravada após merge**: H-6.d —
  ClickHouse schema migration with canonical
  `base`/`quote`/`contract` columns + back-compat read window.
  Eliminates all 13 `reconstructInstrumentFromLegacy` callers
  + the helper itself.

- **2026-05-27** — H-6.c.1 fechada. **Sub-onda H-6.c
  introduzida** (sub-divisão de H-6.c em H-6.c.1 +
  H-6.c.2 post-pré-flight 5 — descoberta de 6 helpers
  `instrumentFromBinding` + 13 callers
  `reconstructInstrumentFromLegacy` + DTO migration cascade
  inviabilizou monolithic H-6.c). Entregas H-6.c.1:
  **application pass-through migration para derive scope**
  via 10 commits.

  Eliminação completa: `instrumentFromBinding` helper
  deletado de 4 application packages
  (`internal/application/{signal,decision,strategy,risk}/instrument_binding.go`
  + dead `symbol string` field + legacy `NewXxx` wrappers
  removidos das 14 evaluator/sampler/resolver structs). 2
  packages remanescentes (`application/execution`,
  `application/executionclient`) — escopo H-6.c.2 e H-6.f
  respectivamente.

  Novo canonical boundary helper (commit 6):
  `internal/application/ingest/binding.go:BindingTarget.Instrument()`
  com signature `(CanonicalInstrument, error)`. Registry
  declarativa `venueSourceContract` reconhece apenas
  `binances`→Spot e `binancef`→Perpetual. **Synthetic
  sources** (`"binance"`, `"binance_spot"`, `"derive"`,
  `"clickhouse"`, `"unknown_exchange"`,
  `"execute.venue-adapter"`) **intencionalmente ausentes** do
  registry, surfacing o H-6.b' 37f8ddd silent-zero regression-
  shape rather than hiding it. Callers MUST propagate the
  error (a `Finding::with_why` em anti_patterns.toml documenta
  o contrato).

  Derive actor cascade (commit 6):
  `source_scope_actor.onActivateSampler` computa Instrument
  uma única vez via `msg.Target.Instrument()` no boundary;
  skip activation com structured `Error` log on failure (não
  silent drop). 5 derive Config structs gain
  `Instrument CanonicalInstrument` field. 10 derive actor
  files migram para `NewXxxForInstrument(cfg.Source,
  cfg.Instrument, ...)`. `derive_supervisor` cascades inst
  por 12 factory `NewActor` callbacks. **(P1) commit-as-is
  discipline aplicada** — fragmentação em 6a-production +
  6b-tests rejeitada porque produziria estado intermediate
  semantically invalid (actors compilam mas instanciam
  evaluators com zero Instrument).

  14 `NewXxxForInstrument` constructors (commits 2-5):
  signal (RSI, ATR, Bollinger, EMACrossover, MACD, VWAP =
  6), decision (RSIOversold, EMACrossover, BollingerSqueeze
  = 3), strategy (MeanReversion, SqueezeBreakout,
  TrendFollowing = 3), risk (DrawdownLimit, PositionExposure
  = 2). 4 novos `instrument_passthrough_test.go` documentam
  o pass-through contract per package. ~250 application test
  sites migrados via sed/Python-driven uniform pattern
  (commits 7a-7d).

  Cross-scope stragglers (~4 sites missed by commit 6
  Python script, migrados em 7b/7c/7d como single-line fixes
  per package): 3 em `derive/s470_lineage_causality_test.go`
  (1 per package: RSIOversold, MeanReversion,
  PositionExposure), 1 em `execute/s373_structural_test.go`
  (`btcUSDTPerpExec(t)` existente), 2 em
  `execute/e2e_derive_to_execution_test.go` +
  `store/e2e_derive_to_store_test.go` (parameterless
  `btcUSDTPerpDerive` IIFE fixtures adicionados — derive
  event helpers não têm `testing.T` thread-able through 13
  call sites).

  Derive-scope canary integration tests (commit 8): novo
  arquivo
  `internal/actors/scopes/derive/synthetic_source_canary_integration_test.go`
  (~287 LoC) com 3 tests / 15 subtests — rejection at
  boundary (6 synthetic sources), full activation flow with
  `canaryActivator` stand-in (verifica log emission +
  rejection counters + structured fields), legitimate-
  activation-proceeds inverse canary (binances spot + binancef
  perpetual devem NÃO ser over-rejected). Stand-in mirrors
  `source_scope_actor.onActivateSampler` decision shape sem
  spawnar SourceScopeActor full (NATS publisher dependency
  evitada); end-to-end NATS-bound coverage deferida para
  smoke / live integration runs.

  Nova policy file `tools/raccoon-cli/policies/anti_patterns.toml`
  (commit 1) declara forbidden source-string reconstruction
  function names (`instrumentFromBinding` +
  `reconstructInstrumentFromLegacy`) com schema
  `(name, function, severity, why, help, exceptions)`.
  `check_instruments` analyzer ganha
  `load_anti_patterns_policy` + `scan_anti_pattern` +
  `collect_production_go_files` + 5 unit tests (~458 LoC
  Rust). Severity `warning` durante migration window; flips
  para `error` em H-6.f. Commit 9 atualiza `why` prose com
  per-package migration progress (4 eliminated + 2 remaining)
  preservando schema function-based (decisão arquitetural:
  filesystem é source of truth para migration status; adicionar
  `status` field duplicaria filesystem reality).

  **Pattern observado — gofmt drift accumulation**: 5
  instâncias em H-6.c.1 (commits 4, 6, 7a, 7c, 7d) — pre-
  existing drift surfaced opportunistically pelo pre-commit
  hook em touched files. Mitigations registered no
  `RESUMPTION.md` retrospective section (decisão deferida —
  candidate options: full-repo gofmt audit + cleanup,
  pre-commit hook on whole repo, ou CI step validating zero
  drift).

  ADR-0021 row em TRUTH-MAP atualizada para refletir 4
  helpers eliminados + new boundary helper + derive actor
  cascade + canary tests. ADR-0021 permanece `Proposed`;
  promotion gated em literal critério #2 satisfaction
  (zero source-string-based instrument reconstruction em
  production code), atómico em H-6.f.

  **Próxima sub-onda destravada após merge**: H-6.c.2
  (execute scope migration). Sub-onda sequencing policy
  estrita: H-6.c.2 abre branch APENAS após merge desta PR
  (H-6.c.1) em `main`.

- **2026-05-26** — H-6.b'' fechada. Entregas: **pairing chain
  domain migration** — 2 domain types migrados (`pairing.Leg`,
  `pairing.RoundTrip`) com `Instrument CanonicalInstrument` +
  `VenueSymbol()` transitory accessor; 1 query-filter type
  renomeado (`pairing.CrossSessionWindow.Symbol string` →
  `VenueSymbol string`, declarado `string_filter` per Decisão
  #2 (b)); 1 triage projection site pulled-forward
  (`triageclient/get_roundtrip_triage.go:74` adopts
  `review.VenueSymbol()` por compile pressure de anonymous
  embedding). M1 invariant adota native Go struct equality
  (`entry.Instrument != exit.Instrument`) — strictly stronger
  que legacy symbol equality, Contract type also discriminates.
  Analyzer `check-instruments` ganha 3º state `string_filter`
  (commit 1) + 15 unit tests (was 14); 6/6 gate checks PASS.
  Policy entries: `pairing_leg` + `pairing_round_trip` flipam
  `pending` → `migrated`; `cross_session_window` flipa
  `pending` → `string_filter` com inline rationale block
  pointing at pre-flight 6 + commit 37f8ddd precedent.

  **Decisão #5β/γ test canaries**: new unit test
  `get_roundtrip_triage_test.go` (β; happy-path projection +
  zero-Instrument observable canary, ~133 linhas) + inline
  smoke check em `smoke-analytical-e2e.sh` Phase 5 (γ;
  tri-state PASS/WARN/FAIL — WARN preserva honestidade quando
  matched-pair data não está disponível no smoke window;
  ~62 linhas). Década de WARN consecutivo seria dívida de
  cobertura registrada — não bloqueia H-6.b''; considerado em
  H-6.f cleanup ou onda dedicada de smoke hardening.

  **8 commits delivered (plan declared 9)**: consolidação por
  compile pressure documentada nos commits 3 e 8 — commits 3
  (RoundTrip migration) e 5 (triage production line update) do
  plano original consolidados em commit 3, porque
  RoundTripReviewItem embute `pairing.RoundTrip` anonimamente e
  remover `RoundTrip.Symbol` quebra `review.Symbol` access
  imediatamente. Commit 5 do plano permaneceu como test-only
  (Decisão #5β canary coverage). Per H-6.b' precedent (PR #28):
  pull-forward by compile pressure é documentado no commit
  afetado, não escondido em renumbering. **P4/P9 deviation
  observada**: branch `feat/h-6-b1-execution-chain` iniciou
  trabalho de H-6.b'' antes do merge de H-6.b' em `main`
  (PR #28). Pós-merge de H-6.b', branch foi rebased em
  `origin/main` (commit `6b62d89`) para reconciliar histórico
  e produzir uma PR limpa contendo apenas os 9 commits de
  H-6.b''. Lesson registrada em PR description (link no
  Changelog) e em `CONTRIBUTING.md` pre-push validation
  discipline.


  **H-6.f scope revision (post-pré-flight 6 de H-6.b'')**:
  pré-flight 6 descobriu que o débito real de H-6.f é maior
  que "remove transitory methods". Scope revisado:
  1. Audit e remoção dos 6 helpers `instrumentFromBinding` em
     `application/{signal,decision,strategy,risk,execution,
     executionclient}/` — todos hardcoded para `binances`/
     `binancef` + `USDT` quote, retornam zero silenciosamente
     para qualquer outro input.
  2. Audit `reconstructInstrumentFromLegacy` em
     `internal/adapters/clickhouse/candle_reader.go:150` —
     retorna error mas o error é descartado em 11 chamadas em
     `composite_reader.go` e `execution_reader.go`. Either
     propagate errors or replace com Instrument pass-through
     onde upstream já carrega.
  3. Migrate callers para receber Instrument diretamente de
     upstream (pattern estabelecido por
     `NewPaperOrderEvaluatorForInstrument` em H-6.b' commit
     37f8ddd).
  4. Remover métodos `VenueSymbol()` apenas após todos os
     callers migrarem.
  5. Promover ADR-0021 a `Accepted` quando critério #2 estiver
     literalmente satisfeito: zero source-string-based
     instrument reconstruction em production code.

  Esta revisão garante que H-6.f não é "remove transitory
  methods" mas o trabalho semântico real. ADR-0021 permanece
  `Proposed` — promoção continua sendo evento atômico em
  H-6.f. Sub-onda H-6.c destravada após merge desta PR
  (H-6.b'') em `main`.

- **2026-05-26** — H-6.b' fechada. Entregas: três domain types
  da execution chain migrados `Symbol string` → `Instrument
  CanonicalInstrument` + `VenueSymbol()` transitory accessor:
  `execution.ExecutionIntent` (production cascade ~50 sites em
  actors/adapters/application — KV publisher dedup key e
  PartitionKey composers atualizados via VenueSymbol),
  `effectiveness.Attribution` (derived from `intent.Instrument`),
  `execution.AuditLifecycleEntry` (reconstrução de
  `LifecycleEntry` DTO via novo per-package
  `instrumentFromBinding` em `executionclient`, sunset H-6.f).
  Analyzer `check-instruments` flipou as 3 entries em
  `policies/domain_types.toml` de `pending` → `migrated`; 6/6
  checks PASS, total gate 102/102. Sub-wave naming convention
  documentada nesta PRD (acima). **Triage drop closure note**:
  Triage population sites verificados durante pré-flight; zero
  sites required migration nesta sub-wave —
  `DecisionTriageItem` é buffered pelo ReviewTransform DTO
  (application-layer; domain→DTO boundary migrado em H-6.b
  commit 4; DTO→Triage permanece string-to-string até H-6.c
  migrar ReviewTransform); `ExecutionTriageItem` não existe no
  codebase; `RoundTripTriageItem` corretamente deferido para
  H-6.b'' (upstream RoundTrip migra lá). 5 commits (ExecutionIntent
  atomic, Attribution, AuditLifecycleEntry, policy flip, docs
  closure este commit). ADR-0021 permanece `Proposed` —
  critério #2 ainda não literalmente satisfeito (restam Pairing
  chain types em H-6.b''). **Sub-onda H-6.b'' destravada apenas
  após merge desta em `main`** (sub-onda sequencing policy
  estrita).
- **2026-05-26** — H-6.b refinement: pré-flight obrigatório
  descobriu 15 domain types totalizando 174 test files,
  forçando fragmentação além do plano de H-6.a (que assumia 5
  types). Sub-divisão por dependency order em **H-6.b** (Layer
  1+2: Evidence + Signal/Decision/Strategy/Risk, 7 types),
  **H-6.b'** (Layer 3+3': ExecutionIntent + Attribution +
  AuditLifecycleEntry), e **H-6.b''** (Layer 4: Pairing chain
  + Triage RoundTrip population site). Total sub-ondas
  internas H-6 sobe de 6 para 8. Decisão fundamentada em
  numbers reais do pré-flight; análise alternativa em "Refinamento
  H-6.b" acima. Lands como **commit 1 da PR de H-6.b**, antes
  de qualquer commit de código (P3 absoluto).
- **2026-05-25** — PROGRAM-0004 created. Status `Active`. Ondas
  H-6 (sub-divididas em H-6.a–H-6.f por questão de cascade) +
  H-7 declared. ADRs 0021 + 0022 são governantes (ambas
  `Proposed` no início da Fase, entregues em H-2). Sub-onda
  sequencing policy declarada (mais estrita que P4 top-level).
  Lands como entrega de **commit 1 da H-6.a**, alongside ADR-0021
  erratum (criterion #4 split) em commit 0.
- **2026-05-25** — H-6.a fechada. Entregas: erratum ADR-0021
  (criterion #4 split em #4a/#4b), abertura PROGRAM-0004,
  `internal/domain/instrument/` package (Venue, BaseAsset,
  QuoteAsset, ContractType, CanonicalInstrument com JSON tags +
  21 testes), `ObservationTrade.Symbol string` → `Instrument
  CanonicalInstrument` com método transitório
  `VenueSymbol()` (sunset H-6.f), ambos Binance adapters migrados
  com pattern detection `_\d{6}$` para discriminar delivery
  futures, raccoon-cli `check instruments` analyzer + policy
  `policies/adapters.toml` (allowlist binances/binancef) integrado
  como Step 9 do quality-gate. Transitory-method pattern
  documentado nesta PRD para reuso em sub-ondas seguintes.
  ADR-0021 permanece `Proposed` — promoção é evento atômico em
  H-6.f após critério #2 literal. **Sub-onda H-6.b destravada
  apenas após merge desta em `main`** (sub-onda sequencing
  policy estrita).
