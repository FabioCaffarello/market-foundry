# Stage C28 Report: Strategic Operating Model For The Repository As A Development Platform

## 1. Executive Summary

O Stage C28 consolidou o operating model estratégico do repositório como
plataforma de desenvolvimento do `market-foundry`.

As waves C20 a C27 já tinham produzido uma base forte de automação leve,
controle de custo estrutural, disciplina de extensão, lifecycle do CLI,
health, review periódico e sunset de superfícies. O gap remanescente era
estratégico-operacional: faltava uma visão única que explicasse como esses
artefatos funcionam juntos como sistema de operação do repositório.

O resultado do C28 foi fechar essa camada em dois documentos complementares:

- um operating model estratégico para o repositório como development platform;
- um modelo aplicado de governança, health, review e sustentabilidade.

Também foram feitos ajustes leves em índices, governança documental,
`make docs`, guard rails e stage history para que essa camada passe a ser parte
ativa do sistema de suporte do repositório.

## 2. Scope Boundaries

### In scope

- consolidação do operating model estratégico do repositório como development
  platform;
- integração entre health, review cadence, tooling governance, support-surface
  lifecycle, documentação e automação leve;
- ajustes leves de índices, governança documental e proteção mínima contra
  drift.

### Out of scope

- mudança da arquitetura funcional do sistema;
- criação de workflow engine, scorecard ou programa pesado de governança;
- refactors amplos de harnesses, CLI ou superfícies operacionais sem hotspot
  comprovado.

### Not changed

- o papel de `make` como entrypoint público canônico;
- o papel de `scripts/` como harness layer;
- o papel do `raccoon-cli` como superfície de análise estrutural e governança;
- o papel de `docs/stages/` como evidência histórica.

## 3. Síntese Das Dimensões Estratégicas Acumuladas

O C28 parte explicitamente dos aprendizados consolidados nas waves anteriores:

- C20: automação leve só onde a rotina é frequente, objetiva e transparente;
- C21: custo estrutural é dominado por fan-out de manutenção, não por volume
  bruto;
- C22: novas superfícies devem disputar espaço com as existentes antes de
  nascer;
- C23: o `raccoon-cli` precisa de lifecycle explícito para não virar catálogo
  paralelo;
- C24: sustentabilidade depende de entrypoints, índices e promotion path
  consistentes;
- C25: health do ambiente é multidimensional e deve orientar decisão, não
  scoring;
- C26: revisão periódica precisa ser leve, gatilhada por sinais e acompanhada
  de follow-through proporcional;
- C27: superfícies de suporte precisam de estados explícitos de lifecycle.

O C28 unifica isso sob a ideia de que o repositório é uma development platform
do Foundry, e portanto precisa de um operating model de longo prazo para si.

## 4. Changes Applied

Foram criados:

- `docs/operations/strategic-operating-model-for-the-repository-as-a-development-platform.md`
- `docs/operations/repository-platform-governance-health-review-and-sustainability-model.md`

Foram atualizados:

- `docs/operations/README.md`
- `docs/README.md`
- `docs/operations/documentation-governance-entrypoints-and-taxonomy.md`
- `Makefile` (`make docs`)
- `scripts/repository-consistency-check.sh`
- `docs/stages/INDEX.md`

Os ajustes fizeram três coisas:

1. indexaram o C28 como camada canônica;
2. tornaram o C28 parte do conjunto protegido pelo guard rail leve;
3. ligaram a nova camada ao modelo de taxonomia e maintenance triggers já
   existente.

## 5. Modelo Estratégico Consolidado

O modelo final diferencia três camadas:

### 1. Canônico

- `make` como workflow público principal;
- pilha de entrypoints em `README.md`, `DEVELOPMENT.md`, `docs/README.md` e
  `docs/operations/README.md`;
- `scripts/` como harness layer;
- `raccoon-cli` como tooling de análise estrutural e governança;
- `docs/operations/`, `docs/tooling/`, `docs/architecture/` e `docs/stages/`
  com ownership explícito.

### 2. Revisado periodicamente

- health do repositório;
- cadência de review;
- CLI maturity/lifecycle;
- aliases, wrappers, scripts e harnesses;
- docs ativas, índices e navegação;
- checks leves e automação de baixo volume.

### 3. Flexível

- wording e UX local de helpers;
- pequenos ajustes de automação advisory;
- flags, caminhos de debug e follow-ups táticos;
- seleção do próximo hotspot de sustentabilidade.

O operating model também explicita:

- ownership por superfície;
- três ritmos leves de operação: contínuo, periódico e estratégico;
- um decision model baseado em problema, owner surface, menor resposta válida e
  custo de manutenção;
- um protection model orientado a guard rails objetivos e baratos.

## 6. Final Operating Model

O repositório agora passa a operar com o seguinte modelo final:

1. tratar o repositório como development platform do Foundry, não apenas como
   árvore de código;
2. preservar uma superfície pública pequena, canônica e claramente indexada;
3. revisar periodicamente apenas as superfícies com risco real de entropia;
4. manter flexibilidade tática sem abrir mão de ownership estrutural;
5. usar health, review cadence, lifecycle e cost control como um único sistema
   de decisão;
6. preferir clarify, align, consolidate e guard antes de abrir nova wave;
7. manter stage reports como evidência histórica e não como owner implícito de
   regra ativa.

Esse modelo fecha a camada estratégica com baixo peso operacional e valor real
para a evolução do ambiente de desenvolvimento do Market Foundry.

## 7. Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C28 STAGE_SLUG=strategic-operating-model-for-the-repository-as-a-development-platform STAGE_REQUIRE=docs/operations/strategic-operating-model-for-the-repository-as-a-development-platform.md,docs/operations/repository-platform-governance-health-review-and-sustainability-model.md`

## 8. Limits And Non-Goals

- O C28 não altera a arquitetura funcional do sistema.
- O C28 não cria um produto paralelo de tooling/governança.
- O C28 não transforma review periódico em cerimônia obrigatória.
- O C28 não cria workflow engine, ownership roster externo ou scorecards.
- O C28 não tenta substituir julgamento técnico por automação.

## 9. Preparation For Next Stage

A próxima wave recomendada é uma wave curta de aplicação seletiva do operating
model em hotspots concretos da repository platform.

Prioridade sugerida:

1. revisar se existe algum hotspot real de convergência entre `make`,
   harnesses grandes e docs operacionais onde consolidação renderia redução
   clara de fan-out;
2. auditar compatibilidade residual e superfícies auxiliares que ainda estejam
   muito próximas de competir com a camada canônica;
3. preferir follow-ups pequenos e baseados em sinais recorrentes do que abrir
   uma nova frente abstrata de governança.

Recomendação formal:
abrir uma próxima wave Codex apenas se ela atacar um hotspot comprovado da
development platform com ganho mensurável de coerência, discoverability ou
sustentabilidade operacional.
