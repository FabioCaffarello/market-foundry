# Stage C24 Report: Long-Term Documentation And Operational Sustainability Model

## 1. Executive Summary

O Stage C24 consolidou um modelo leve de sustentabilidade documental e
operacional de longo prazo para o `market-foundry`.

O foco não foi criar uma nova camada de processo. O foco foi explicitar como
docs, índices, entrypoints, scripts, `raccoon-cli`, checks leves e stage
reports devem continuar coerentes quando o repositório crescer em novas waves.

O resultado principal foi a formalização de um modelo único de sustentabilidade
que conecta governança documental, governança de tooling e saúde estrutural do
ambiente de desenvolvimento, com melhorias proporcionais aplicadas onde o risco
de entropia já era concreto.

## 2. Sustainability Diagnosis

### Estado encontrado

O repositório já possuía uma base forte:

- entrypoints canônicos claros em `README.md`, `DEVELOPMENT.md`, `docs/README.md`
  e `docs/operations/README.md`;
- modelo de navegação por metadados leves vindo do C19;
- automação de continuidade e governança leve vindas do C20;
- controle de custo estrutural definido no C21;
- disciplina de evolução de tooling e de lifecycle do `raccoon-cli` vindas de
  C22 e C23;
- guard rail leve em `make repo-consistency-check`.

### Lacunas ainda presentes

Mesmo com essa base, a sustentabilidade futura ainda dependia demais de
disciplina implícita em três pontos:

1. não havia um documento único conectando docs, tooling, índices e saúde
   operacional de longo prazo;
2. faltava um conjunto explícito de rotinas curtas de revisão para conter
   entropia sem criar burocracia;
3. havia risco real de docs ativas ficarem fora dos índices canônicos,
   especialmente em `docs/tooling/`.

### Riscos de entropia identificados

- docs ativas surgirem fora dos READMEs canônicos da área;
- stage reports continuarem sendo usados como dono implícito de regra ativa;
- novos helpers crescerem em `make`, `scripts/` e CLI para o mesmo problema;
- scripts raros seguirem existindo sem owner surface claro;
- root docs crescerem novamente como catálogos paralelos;
- checks leves acumularem volume histórico em vez de invariantes ativas.

## 3. Scope Boundaries

### In scope

- sustentabilidade documental e operacional de longo prazo;
- ligação entre docs, tooling, índices, entrypoints e checks leves;
- rotinas leves de revisão e controle de entropia;
- ajustes proporcionais em índices e guard rails.

### Out of scope

- refatores arquiteturais do runtime funcional;
- engines de workflow ou governança pesada;
- reestruturação ampla de `raccoon-cli` ou dos scripts de smoke;
- inventários externos de ownership.

### Not changed

- a arquitetura funcional do sistema;
- os bounded contexts e a topologia operacional;
- o papel de `make` como entrypoint público canônico;
- o papel de `docs/stages/` como trilha histórica.

## 4. Changes Applied

### Novos documentos canônicos

Foram criados:

- `docs/operations/long-term-documentation-and-operational-sustainability-model.md`
- `docs/operations/repository-sustainability-review-routines-and-entropy-control.md`

O primeiro define o modelo final de sustentabilidade do repositório.
O segundo transforma esse modelo em rotinas curtas de revisão e controle de
entropia.

### Ajustes leves em entrypoints e índices

Foram atualizados:

- `docs/operations/README.md`
- `docs/README.md`
- `Makefile` (`make docs`)
- `docs/stages/INDEX.md`

Esses ajustes garantem que o C24 entre no mapa canônico de navegação sem
reintroduzir catálogo excessivo em superfícies raiz.

### Correção proporcional de risco já materializado

Durante o diagnóstico, foi encontrado um sinal concreto de entropia:
`docs/tooling/README.md` não indexava dois documentos ativos já presentes no
diretório:

- `docs/tooling/raccoon-cli-advanced-architecture-refinement.md`
- `docs/tooling/raccoon-cli-internal-refactor-rules-and-extension-guidelines.md`

O índice de tooling foi atualizado para reincorporar esses documentos, reduzindo
o risco de docs boas se tornarem órfãs por omissão de navegação.

### Reforço proporcional do guard rail leve

`scripts/repository-consistency-check.sh` foi estendido em dois pontos:

1. o conjunto de docs canônicas obrigatórias agora inclui os dois documentos do
   C24;
2. foi adicionado um check leve para validar que docs ativas em
   `docs/operations/` e `docs/tooling/` estejam indexadas nos READMEs canônicos
   dessas áreas.

Esse reforço é objetivo, barato e diretamente ligado ao risco identificado.
Ele evita entropia silenciosa sem transformar o check em um sistema pesado.

## 5. Final Operating Model

O modelo final do C24 ficou apoiado em seis pilares:

1. disciplina de entrypoints canônicos;
2. disciplina de promoção de regras duráveis para docs ativas;
3. separação clara entre `make`, scripts, CLI, docs ativas e stage evidence;
4. obrigação de indexação para docs ativas de `operations` e `tooling`;
5. rotinas leves de revisão anexadas ao fluxo normal de mudança;
6. manutenção de baixo fan-out, evitando que uma pequena mudança force ampla
   reconciliação manual.

Na prática, isso significa:

- uma pergunta recorrente deve continuar tendo um ponto de partida óbvio;
- uma regra durável não deve ficar presa ao stage report;
- uma doc ativa não deve depender de memória ou busca textual para ser achada;
- novos helpers devem disputar espaço com superfícies existentes antes de
  nascerem como novos entrypoints;
- checks leves devem proteger só o que é ativo, objetivo e silenciosamente
  sujeito a drift.

## 6. Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C24 STAGE_SLUG=long-term-documentation-and-operational-sustainability-model STAGE_REQUIRE=docs/operations/long-term-documentation-and-operational-sustainability-model.md,docs/operations/repository-sustainability-review-routines-and-entropy-control.md`

## 7. Limits And Deferred Follow-Ups

C24 deliberadamente não fez o seguinte:

- não criou workflow engine para docs ou tooling;
- não adicionou aprovação obrigatória para pequenas mudanças;
- não criou inventário externo de ownership;
- não transformou o guard rail leve em política totalizante;
- não alterou arquitetura funcional, bounded contexts ou runtime topology.

Também não tentou eliminar toda duplicação residual do repositório.
O objetivo foi fortalecer o modelo de sustentabilidade, não abrir uma wave de
reescrita documental.

## 8. Preparation For Next Stage

A próxima wave Codex recomendada é uma wave curta de consolidação orientada a
hotspots de suporte, com prioridade para:

1. revisar scripts de smoke e wrappers que continuam grandes ou com alto custo
   de manutenção;
2. identificar aliases, rotas compatíveis e entrypoints pouco usados que ainda
   mereçam consolidação ou despromoção;
3. continuar preferindo correções proporcionais guiadas por hotspots reais, e
   não expansão abstrata de governança.

Em termos práticos, a próxima wave deve tratar a sustentabilidade do repositório
como disciplina contínua de contenção de entropia, não como criação de novas
camadas processuais.
