# Stage C27 Report: Support-Surface Sunset, Consolidation, And Retirement Strategy

## 1. Executive Summary

O Stage C27 consolidou uma estratégia explícita para lifecycle de superfícies de
suporte do `market-foundry`.

O repositório já tinha:

- entrypoints canônicos;
- disciplina de evolução de tooling;
- guard rails leves;
- rotina de review de sustentabilidade;
- cadência e gatilhos de review periódico.

O gap remanescente era de lifecycle: faltava explicitar quando uma superfície
de suporte deve continuar ativa, quando deve ser consolidada, quando deve ser
marcada como legado e quando deve ser aposentada.

O resultado foi uma camada nova, prática e proporcional, orientada a:

1. evitar acúmulo de superfícies quase duplicadas;
2. distinguir redundância saudável de redundância cara;
3. transformar sunset e retirement em decisões guiadas por critérios;
4. reforçar a sustentabilidade do ambiente sem forçar refactors amplos.

## 2. Scope Boundaries

### In scope

- estratégia de lifecycle para superfícies de suporte;
- critérios para keep, consolidate, legacy e retire;
- revisão de `Makefile`, scripts, docs operacionais, wrappers, entrypoints e
  índices sob a ótica de entropia;
- ajustes leves de docs/help/indexação;
- reforço mínimo de discoverability via guard rail documental.

### Out of scope

- remoção abrupta de superfícies úteis;
- refactors amplos em harnesses, CLI ou documentação;
- mudança da arquitetura funcional do sistema;
- reclassificação ampla de comandos ou scripts além do necessário para clareza.

### Not changed

- a arquitetura funcional dos serviços;
- o papel do `Makefile` como workflow público canônico;
- o papel de `scripts/` como harness layer;
- o papel de `docs/stages/` como evidência histórica.

## 3. Surfaces With Legacy Or Overlap Risk

O stage confirmou que os hotspots mais prováveis de acumulação de legado são:

1. aliases de `make` que podem deixar de ser discoverability helpers e virar
   duplicação cara;
2. wrappers de orquestração como `make live*`, que são úteis, mas não devem
   competir com `make smoke*` como proof-of-record;
3. invocação direta de `scripts/*.sh`, que é válida para debug e manutenção de
   harness, mas cara quando ensinada como caminho normal;
4. docs operacionais e sumários de governança que podem começar a repetir a
   mesma regra em paralelo;
5. índices e entrypoints que podem crescer até virar catálogos redundantes;
6. superfícies compatibility-only, especialmente helpers históricos da CLI.

O ponto central é que essas superfícies não são ruins por definição. O risco
surge quando deixam de ter diferenciação clara.

## 4. Criteria Established

O C27 consolidou seis dimensões de decisão:

1. uso;
2. clareza;
3. custo de manutenção;
4. discoverability;
5. sobreposição de responsabilidade;
6. risco de drift.

Com base nelas, o stage estabeleceu quatro estados explícitos:

1. active canonical;
2. active auxiliary;
3. legacy;
4. retired.

Critério estratégico central:
manter uma única resposta canônica para cada pergunta recorrente do repositório
e tolerar redundância apenas quando ela melhora usabilidade sem dividir
ownership.

## 5. Changes Applied

Foram aplicadas melhorias leves, sem remoção de superfícies:

- criação de:
  - `docs/operations/support-surface-sunset-consolidation-and-retirement-strategy.md`
  - `docs/operations/support-surface-lifecycle-signals-and-consolidation-criteria.md`
- atualização de `docs/operations/README.md` e `docs/README.md` para indexar a
  nova camada canônica;
- atualização de `Makefile` (`make docs`) para expor os docs de lifecycle;
- atualização de `docs/operations/makefile-targets-reference-and-conventions.md`
  para deixar explícito que aliases devem continuar pagando pelo custo que
  adicionam;
- atualização de `docs/operations/scripts-catalog-and-usage-guide.md` e
  `scripts/README.md` para reforçar canonical-vs-wrapper-vs-debug-only;
- atualização de `docs/stages/INDEX.md` para registrar o C27;
- atualização de `scripts/repository-consistency-check.sh` para exigir os novos
  documentos canônicos e validar cross-link mínimo entre a estratégia e seus
  critérios.

## 6. Final Lifecycle Strategy

O modelo final do C27 funciona assim:

### Remain

Uma superfície permanece ativa quando ainda tem uso recorrente, ownership claro
e custo de manutenção proporcional.

### Consolidate

Consolidação é a resposta padrão quando o problema é fragmentação e não falta
de valor.

Exemplos típicos:

- alias demais para o mesmo fluxo;
- docs ativas respondendo a mesma pergunta;
- wrappers e scripts ensinados como caminhos equivalentes;
- índices sobrepostos.

### Mark legacy

Legacy passa a ser um estado explícito, não implícito.

Ele vale quando:

- já existe substituto canônico;
- remoção abrupta ainda seria cara ou confusa;
- a superfície precisa parar de ser promovida e ficar congelada, salvo ajustes
  de compatibilidade ou segurança.

### Retire

Retirement só é recomendado quando:

- o uso atual já não justifica a existência;
- o replacement path está claro;
- docs e entrypoints ativos não dependem mais da superfície antiga;
- manter a superfície gera mais ambiguidade ou custo do que valor.

## 7. Sustainability Outcome

O ganho de sustentabilidade do stage é tornar explícito algo que antes podia
virar decisão ad hoc:

- quando manter;
- quando consolidar;
- quando rotular como legado;
- quando aposentar.

Isso reduz o risco de:

- alias inflation;
- doc sprawl;
- wrappers sem dono claro;
- compatibility paths envelhecendo sem rótulo;
- decisões de retirement feitas por impulso.

## 8. Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C27 STAGE_SLUG=support-surface-sunset-consolidation-and-retirement-strategy STAGE_REQUIRE=docs/operations/support-surface-sunset-consolidation-and-retirement-strategy.md,docs/operations/support-surface-lifecycle-signals-and-consolidation-criteria.md`

## 9. Limits And Non-Goals

- Nenhuma superfície útil foi removida.
- O stage não redefiniu a taxonomia completa do repositório.
- Não foi criada política rígida demais para um ambiente ainda em evolução.
- Não foi expandido o uso de guard rails além do mínimo para discoverability e
  coerência.

## 10. Preparation For C28

C28 deve partir do pressuposto de que o repositório agora já sabe identificar
quando uma superfície está envelhecendo, mas ainda precisa seguir tornando essa
disciplina prática em hotspots específicos.

Preparação recomendada:

1. escolher um hotspot onde o custo estrutural esteja mais ligado a manutenção
   recorrente do que a falta de capability;
2. usar os critérios de lifecycle do C27 para justificar qualquer
   consolidação;
3. preferir ajustes que reduzam fan-out de manutenção;
4. tratar aliases, wrappers, docs-ponte e índices como superfícies de custo
   real, não apenas conveniências inocentes;
5. manter o follow-through proporcional e evitar transformar sustentabilidade
   em programa de cleanup permanente.
