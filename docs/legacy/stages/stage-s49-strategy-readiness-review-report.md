# Stage S49 — Strategy Readiness Review Report

> Formal readiness assessment for opening a `strategy` domain layer.
> Date: 2025-03-17

## Stage Identity

| Field | Value |
|-------|-------|
| Stage | S49 |
| Title | Strategy Readiness Review |
| Type | Review (non-implementation) |
| Objective | Determine if Market Foundry is ready to introduce `strategy` as the 5th domain layer |
| Verdict | **NOT YET READY — 3 blocking gaps must close first** |

---

## 1. Executive Summary

Market Foundry's upper layers (signal, decision) are production-ready and hardened. Governance is comprehensive (raccoon-cli covers 95%+ of drift), config-driven activation is safe with cross-layer dependency validation, the gateway is clean and stateless, and the store maintains clear projection authority. The mesh is structurally sound with all invariants enforced.

However, the foundational layers (observation, evidence) carry critical test coverage debt. Adapter encode/decode paths, projection actor gates, and the entire ingest pipeline have zero automated tests. Opening a `strategy` layer on this foundation would compound reliability risk — every strategy evaluation traces back through these untested paths.

**Recommendation**: Execute 2 focused hardening stages (S50–S51) to close test gaps, then open strategy design in S52.

---

## 2. Readiness Assessment Summary

### Domain Maturity Scores

| Domain | Score | Status |
|--------|-------|--------|
| Observation | 5/10 | Architecture sound, zero tests |
| Evidence | 6.5/10 | Core tested, adapters/actors untested |
| Signal | 8.5/10 | Hardened (S37), multi-symbol proven |
| Decision | 9/10 | Production-ready, 78 test cases |
| Gateway | 8.5/10 | Clean, stateless, conditional routes |
| Store | 8/10 | Clear authority, projection gates active |
| Config | 8.5/10 | Hardened, dependencies validated |
| Governance | 9/10 | raccoon-cli comprehensive |

### Cross-Cutting Assessment

| Dimension | Score | Notes |
|-----------|-------|-------|
| Projection authority | 9/10 | Single-writer enforced, read-only gateway |
| Query surface quality | 7/10 | All endpoints exist; 2 evidence handlers untested |
| Mesh integrity | 9.5/10 | All invariants enforced and actively monitored |
| Activation model | 8/10 | Safe; binding deactivation requires restart |
| Config dependencies | 8.5/10 | Cross-layer validation at startup + pre-deploy |

---

## 3. Explicit Answers to Review Questions

### Decision está madura o suficiente?

**Sim.** Decision é o domínio mais maduro por densidade de testes (78 casos). Implementação completa de ponta a ponta: domain → evaluator → actors → adapters → projection → query → HTTP. Hardened em S48 com projection authority, replay invariants e observability counters. Multi-symbol isolation provado em S46.

### Governança de decision está suficiente?

**Sim.** raccoon-cli tem 5 regras de drift detection (DD-1–DD-5) e 10 guardrails (DG-1–DG-10) para decision. Todos enforcement points estão ativos e integrados no quality-gate.

### Config dependencies estão seguras?

**Sim.** Cadeia de dependência `decision → signal → evidence` é validada em runtime (startup do Go) e em tempo de deploy (raccoon-cli static analysis). Nomes de família desconhecidos são rejeitados com erro explícito. Derive e store são verificados por consistência cruzada.

### A mesh continua clara e protegida pelo raccoon-cli?

**Sim.** 5 famílias ativas, todas com subject taxonomy validada, stream contracts checados, consumer specs auditados. raccoon-cli topology audit mapeia todos os binários, streams e subjects. Nenhuma violação de invariante detectada.

### O gateway continua limpo?

**Sim.** Gateway é stateless proxy puro. Zero lógica de domínio, zero event publishing, zero repository ownership. Rotas registradas condicionalmente. Acesso KV é exclusivamente read-only via QueryResponderActor.

### O store continua claro como authority?

**Sim.** Store é o único writer para todos os KV buckets. Projection actors aplicam 3 gates (final, validate, monotonicity) consistentemente. Gateway abre conexões KV como read-only. Authority model documentado em S48.

### Quais gaps impedem strategy sem gerar dívida?

Três gaps bloqueantes:

1. **BG-1**: Evidence adapters (publishers, consumers, gateways) com 0% de cobertura de testes
2. **BG-2**: Pipeline de observation/ingest com 0% de cobertura de testes
3. **BG-3**: Projection actors de evidence com 0% de cobertura de testes

Estes não são problemas de design — a arquitetura está correta. O risco é que serialização, criação de streams, durabilidade de consumers e lógica de projeção nunca foram verificados por testes automatizados.

### Menor desenho aceitável de strategy?

1. Uma família de strategy (e.g., `mean_reversion_entry`)
2. Placement em `derive` (mesmo binário que signal/decision)
3. Stream: `STRATEGY_EVENTS`
4. KV bucket: `STRATEGY_{TYPE}_LATEST` (latest-only)
5. HTTP: `GET /strategy/:type/latest`
6. Config: `strategy_families` opt-in
7. Dependency chain: strategy → decision → signal → evidence

Segue exatamente o padrão de entrada de decision (S43).

---

## 4. Blocking Gaps

| ID | Description | Severity | Effort |
|----|-------------|----------|--------|
| BG-1 | Evidence adapter tests missing | CRITICAL | 1 stage |
| BG-2 | Observation/ingest pipeline untested | CRITICAL | 0.5 stage |
| BG-3 | Evidence projection actor tests missing | HIGH | 1 stage |
| BG-4 | TradeBurst domain validation tests | MEDIUM | 0.25 stage |
| BG-5 | Evidence HTTP handler tests (tradeburst/volume) | MEDIUM | 0.25 stage |
| BG-6 | Candle dual-write atomicity undocumented | MEDIUM | 0.25 stage |

---

## 5. Non-Blocking Risks

| ID | Description | Severity | Mitigation |
|----|-------------|----------|-----------|
| NBR-1 | Binding deactivation requires restart | LOW | Operational procedure |
| NBR-2 | Single exchange adapter (binancef) | LOW | Architecture supports multi-source |
| NBR-3 | No projection lag metrics | LOW | Health trackers provide basic liveness |
| NBR-4 | QueryResponderActor not family-filtered | LOW | Zero correctness impact |
| NBR-5 | No signal/decision history projections | LOW | Intentional, add when justified |
| NBR-6 | Zero actor-level tests (systemic) | MEDIUM | BG-3 addresses evidence specifically |

---

## 6. Recommendation

**Não abrir strategy no próximo ciclo.** Executar 2 stages de hardening primeiro.

### Sequência recomendada de próximos stages:

| Stage | Título | Escopo | Resolve |
|-------|--------|--------|---------|
| **S50** | Foundation Test Coverage Sweep | Testes para adapters (evidence + observation), domain TradeBurst, HTTP handlers | BG-1, BG-2, BG-4, BG-5 |
| **S51** | Projection Hardening & Dual-Write Review | Testes para projection actors, revisão de atomicidade candle | BG-3, BG-6 |
| **S52** | Strategy Domain Design | Readiness re-run, design doc, stream/subject contracts, governance rules | Abre strategy |
| **S53** | Strategy First Slice | Implementação da primeira família de strategy | Implementação |

### Critérios para re-run da readiness review (S52):

- Todos os BG-1 a BG-6 fechados com testes passando em CI
- raccoon-cli `quality-gate fast` passa sem warnings
- Nenhum novo gap crítico introduzido

---

## 7. Deliverables Produced

| Document | Path |
|----------|------|
| Strategy Readiness Review | `docs/architecture/strategy-readiness-review.md` |
| Strategy Entry Prerequisites | `docs/architecture/strategy-entry-prerequisites.md` |
| Strategy Risks and Blockers | `docs/architecture/strategy-risks-and-blockers.md` |
| This report | `docs/stages/stage-s49-strategy-readiness-review-report.md` |

---

## 8. Acceptance Criteria Verification

| Critério | Status |
|----------|--------|
| Readiness review é específica, honesta e acionável | ✅ Cada gap tem arquivos afetados, severity e esforço estimado |
| Foundry ganha gate real antes de abrir strategy | ✅ 6 blocking gaps definidos com acceptance criteria |
| Gaps ficam claros e priorizáveis | ✅ Risk-impact matrix com P0/P1/P2/P3 |
| Próxima onda pode ser planejada com base em evidência | ✅ S50–S53 sequenciados com dependências claras |

---

## 9. Guard Rails Verification

| Guard rail | Status |
|-----------|--------|
| Não implementar strategy | ✅ Nenhum código de strategy criado |
| Não mascarar gaps reais | ✅ 3 gaps CRITICAL/HIGH identificados honestamente |
| Não inflar docs com abstrações vagas | ✅ Cada gap referencia arquivos específicos com contagem de testes |
| Registrar bloqueios e pré-condições concretas | ✅ 8 pré-requisitos com acceptance criteria |
| Manter clareza entre readiness e implementação | ✅ Review produz apenas docs de avaliação, zero código |
