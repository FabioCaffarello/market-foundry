# Stage C31 Report: Criteria For Opening, Containing, Or Rejecting New Support Surfaces

## 1. Executive Summary

O Stage C31 consolidou o modelo executivo para decidir quando o
`market-foundry` deve abrir novas superfícies de suporte e quando deve conter,
consolidar ou rejeitar esse tipo de expansão.

As waves C20 a C30 já tinham definido automação leve, custo estrutural,
disciplina de extensão, lifecycle, health, checkpoints e readiness. O gap
remanescente era decisório e operacional ao mesmo tempo: faltava um modelo
curto, concreto e ancorado em exemplos reais do repositório para responder
quando uma nova superfície de suporte é realmente necessária.

O resultado do C31 foi:

- um documento canônico de critérios executivos para abertura, contenção,
  consolidação e rejeição;
- um documento aplicado com regras e exemplos reais de CLI, scripts, Make
  targets, docs e checks;
- pequenos ajustes em índices, help e guard rails para tornar essa disciplina
  visível e usável no repositório real.

## 2. Scope Boundaries

### In scope

- analisar o crescimento recente do ambiente de desenvolvimento;
- distinguir abertura útil de crescimento entrópico;
- consolidar critérios executivos de decisão para support surfaces;
- aplicar ajustes leves em docs, índices, `make docs` e guard rails;
- documentar o modelo final de decisão sobre expansão do ambiente de
  desenvolvimento.

### Out of scope

- mudança da arquitetura funcional do sistema;
- criação de processo formal pesado de aprovação;
- abertura de novas superfícies operacionais sem necessidade comprovada;
- refactors amplos de CLI, scripts, runtime ou docs fora do necessário para
  publicar e proteger o modelo.

### Not changed

- o papel de `make` como entrypoint público canônico;
- o papel de `scripts/` como harness layer;
- o papel do `raccoon-cli` como tooling de análise estrutural e governança;
- o papel de `docs/stages/` como evidência histórica;
- a arquitetura funcional do Foundry.

## 3. Historical Patterns Found

O histórico recente mostrou quatro padrões claros.

### 1. Aberturas que agregaram valor real

Essas superfícies nasceram para fechar lacunas recorrentes e objetivas.

Exemplos:

- `make stage-status` abriu uma superfície intermediária legítima entre
  scaffolding e check formal;
- `make smoke-restart-recovery` e `make codegen-equivalence` promoveram fluxos
  já reais, mas escondidos;
- `docs/operations/` criou um owner canônico para regras ativas de suporte que
  antes estavam dispersas.

### 2. Necessidades que deveriam ser absorvidas por superfícies existentes

Exemplos:

- aliases como `lint`, `test-unit` e `stack-*` melhoraram discoverability sem
  criar novas superfícies canônicas;
- `make docs` melhorou a orientação, mas como ponte para owners já existentes;
- a evolução do CLI em C23 privilegiou taxonomia agrupada e contenção de
  compatibilidade em vez de novos catálogos paralelos.

### 3. Demandas melhores resolvidas por docs ou convenção

Exemplos:

- contenção explícita de `runtime-smoke` como `legacy`;
- reforço documental de que `scripts/*.sh` são harness/debug, não segunda API
  pública;
- reforço recorrente de que stage reports são evidência histórica, não source
  of truth ativo.

### 4. Casos em que expansão tenderia a aumentar entropia

Exemplos:

- novas scripts muito próximas de harnesses já existentes;
- novos checks para preferências subjetivas ou temporárias;
- novas famílias públicas quando um owner atual já consegue absorver a
  necessidade.

O padrão final foi nítido:
quando a proposta melhorava ownership e reduzia ambiguidade, o ganho era real;
quando apenas multiplicava conveniências, o repositório acumulava entropia.

## 4. Criteria Defined

O C31 consolidou sete critérios executivos:

1. recorrência real do problema;
2. clareza de owner;
3. valor diferenciado;
4. custo estrutural e fan-out de manutenção;
5. efeito sobre o caminho canônico;
6. risco de drift;
7. reversibilidade do movimento.

Com esses critérios, toda proposta deve terminar em um dos quatro estados:

1. open;
2. contain;
3. consolidate;
4. reject.

O viés deliberado do modelo ficou:

1. clarify;
2. contain;
3. consolidate;
4. open.

Ou seja:
abrir nova superfície deixou de ser a resposta padrão para desconforto de
workflow.

## 5. Changes Applied

Foram criados:

- `docs/operations/criteria-for-opening-containing-or-rejecting-new-support-surfaces.md`
- `docs/operations/support-surface-expansion-decision-rules-and-examples.md`
- `docs/stages/stage-c31-criteria-for-opening-containing-or-rejecting-new-support-surfaces-report.md`

Foram atualizados:

- `docs/operations/README.md`
- `docs/README.md`
- `Makefile`
- `scripts/repository-consistency-check.sh`
- `docs/stages/INDEX.md`

Os ajustes leves fizeram quatro coisas:

1. indexaram o modelo do C31 nos entrypoints canônicos;
2. tornaram os novos docs visíveis no `make docs`;
3. adicionaram proteção leve contra drift silencioso entre os dois docs do C31;
4. registraram a stage na trilha histórica oficial.

## 6. Final Executive Decision Model

O modelo final do C31 opera assim:

1. declarar o problema recorrente em linguagem operacional;
2. identificar o owner atual que deveria responder a ele;
3. testar primeiro se docs, naming ou alinhamento resolvem;
4. testar depois se o owner atual consegue absorver a necessidade;
5. consolidar se o problema real for overlap;
6. abrir nova superfície apenas se a lacuna continuar sem owner adequado;
7. rejeitar quando a proposta criar mais burden do que clareza.

A regra executiva final ficou:

- abrir quando falta capacidade durável e recorrente;
- conter quando o owner já existe;
- consolidar quando overlap é o problema real;
- rejeitar quando docs, convenção ou nenhuma mudança bastam.

Isso reforça a plataforma de desenvolvimento do repositório sem transformar
governança em bloqueio burocrático.

## 7. Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C31 STAGE_SLUG=criteria-for-opening-containing-or-rejecting-new-support-surfaces STAGE_REQUIRE=docs/operations/criteria-for-opening-containing-or-rejecting-new-support-surfaces.md,docs/operations/support-surface-expansion-decision-rules-and-examples.md`

## 8. Preparation For C32

C32 deve aplicar o modelo do C31 sobre um hotspot concreto, não expandir a
camada conceitual.

Preparação recomendada:

1. escolher um hotspot real onde haja risco de multiplicação de superfícies,
   especialmente em docs operacionais, wrappers ou checks leves;
2. usar o modelo do C31 para decidir entre conter, consolidar ou rejeitar antes
   de qualquer nova abertura;
3. priorizar um ajuste pequeno com redução clara de fan-out de manutenção;
4. evitar waves mistas em que crescimento funcional venha acompanhado de
   expansão oportunista da repository platform.
