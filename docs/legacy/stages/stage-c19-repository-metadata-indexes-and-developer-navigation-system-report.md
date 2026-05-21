# Stage C19 Report: Repository Metadata, Indexes, And Developer Navigation System

## 1. Executive Summary

O Stage C19 fortaleceu a navegabilidade prática do `market-foundry` sem criar
uma camada pesada de gestão de documentação.

O foco foi simples: ligar melhor a taxonomia documental já existente à
estrutura física real do repositório. O resultado é um sistema leve de
entrypoints locais, mapas de navegação por tarefa e guard rails mínimos para
evitar regressão de discoverability.

## 2. Navigation Diagnosis

Antes de C19, o repositório já tinha bons índices documentais em `README.md`,
`DEVELOPMENT.md`, `docs/README.md` e `docs/operations/README.md`. O problema é
que esses índices respondiam melhor à pergunta “qual documento ler” do que à
pergunta “para qual área do repositório eu devo ir”.

Os sintomas principais eram:

- ausência de entrypoints em áreas top-level importantes como `cmd/`,
  `internal/`, `deploy/`, `scripts/` e `tests/`;
- dependência de conhecimento implícito para saber onde começa a leitura de uma
  concern real;
- discoverability boa para taxonomia de docs, mas fraca para navegação da árvore;
- necessidade de abrir vários documentos até inferir ownership estrutural.

## 3. Scope Boundaries

### In scope

- metadados leves, índices e mapas de navegação ligados à estrutura real do repo;
- entrypoints locais para áreas top-level importantes;
- reforço dos índices raiz e guard rails mínimos para sustain da navegação.

### Out of scope

- criação de um sistema pesado de metadata management;
- duplicação extensa de arquitetura, workflow ou evidência histórica;
- inventários artificiais desconectados do tree real.

### Not changed

- semântica arquitetural do sistema;
- workflow canônico baseado em `make`;
- papel de `docs/stages/` como trilha histórica e não fonte canônica de operação.

## 4. Metadata And Index Gaps Found

As lacunas centrais foram:

- faltava uma camada explícita de metadados leves para a estrutura física do repo;
- faltavam mapas orientados a “task -> diretório -> primeiro arquivo”;
- os docs raiz não apontavam de forma suficiente para entrypoints locais do tree;
- o consistency check protegia docs canônicos, mas não os principais mapas
  físicos do repositório.

## 5. Changes Applied

### Entry points físicos

Foram criados:

- `cmd/README.md`
- `internal/README.md`
- `deploy/README.md`
- `scripts/README.md`
- `tests/README.md`

Esses arquivos funcionam como mapas locais de ownership e leitura inicial, sem
repetir arquitetura ou workflow em excesso.

### Documentos canônicos novos

Foram adicionados:

- `docs/operations/repository-metadata-indexes-and-developer-navigation-system.md`
- `docs/operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md`

O primeiro define o modelo leve de metadados e sustentabilidade. O segundo é o
mapa prático que conecta tarefas às áreas físicas do repo.

### Ajustes nos índices existentes

Foram atualizados:

- `README.md`
- `docs/README.md`
- `docs/operations/README.md`
- `DEVELOPMENT.md`
- `docs/stages/INDEX.md`
- `Makefile`

Esses ajustes promovem a nova camada de navegação sem deslocar os entrypoints
já canônicos.

### Sustentabilidade

O script `scripts/repository-consistency-check.sh` passou a exigir a presença
dos novos documentos e dos entrypoints top-level mais importantes. Isso mantém
o sistema de navegação leve, mas não descartável.

## 6. Final Navigation System

O modelo final fica assim:

1. `README.md`, `DEVELOPMENT.md` e `docs/README.md` continuam como entrypoints
   raiz.
2. `docs/operations/repository-navigation-maps-entrypoints-and-maintenance-rules.md`
   conecta tarefas a áreas do tree.
3. Cada área estrutural importante passa a ter um `README.md` local para
   orientar leitura inicial.
4. O consistency check protege a presença desse núcleo de navegação.

Isso melhora a inteligibilidade do repositório sem criar inventário artificial
ou taxonomia paralela.

## 7. Validation

- `make repo-consistency-check`
- `make stage-check STAGE_ID=C19 STAGE_SLUG=repository-metadata-indexes-and-developer-navigation-system`

## 8. Preparation For Next Stage

- usar os novos entrypoints físicos como critério normal de revisão sempre que
  uma área top-level ganhar responsabilidade nova;
- revisar em C20 se há mais uma ou duas superfícies que merecem entrypoint
  próprio, mas somente se a navegação real justificar;
- manter a regra de promover apenas mapas com uso operacional claro, evitando
  inflar a documentação com inventários decorativos.
