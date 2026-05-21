# Stage C23 Report: Raccoon CLI Command Lifecycle And Deprecation Strategy

## 1. Resumo Executivo

O Stage C23 consolidou uma política explícita de lifecycle para os comandos do
`raccoon-cli`. A taxonomia agrupada existente permaneceu intacta, mas agora a
superfície do CLI distingue com mais clareza comandos `stable core`,
`stable utility`, `experimental` e `legacy`.

O resultado principal não foi uma reestruturação grande do binário. Foi a
formalização de critérios de nascimento, promoção, consolidação e depreciação,
com pequenos ajustes reais em help e docs para evitar que aliases históricos e
helpers legados pareçam superfícies canônicas.

## 2. Diagnóstico Do Catálogo Atual

### Estado encontrado

O catálogo atual já tinha uma taxonomia funcional:

- `check` para guard rails e auditorias;
- `inspect` para análise estrutural somente leitura;
- `change` para impacto e orientação de validação;
- família `snapshot` para baseline e drift;
- `legacy` para helper histórico frágil.

Também já existiam aliases flat ocultos para compatibilidade, como `doctor`,
`quality-gate`, `symbol-trace`, `impact-map`, `tdd` e `runtime-smoke`.

### Problemas observados

Os principais problemas não estavam no dispatcher, mas na clareza de superfície:

1. O lifecycle dos comandos estava implícito, não explícito.
2. Alguns documentos ainda apresentavam aliases históricos como se fossem parte
   normal do catálogo atual.
3. A família `snapshot` era útil e durável, mas sem classificação formal de
   maturidade.
4. O helper `runtime-smoke` já era legado, porém faltava uma política mais
   completa para quando manter, consolidar ou retirar esse tipo de superfície.

### Classificação do catálogo

| Grupo | Maturidade | Observação |
|---|---|---|
| `check` | `stable core` | núcleo de governança e validação |
| `inspect` | `stable core` | superfície de inspeção especializada mas madura |
| `change` | `stable core` | núcleo do loop de mudança segura |
| `snapshot`, `snapshot-diff`, `baseline-drift` | `stable utility` | especialistas e duráveis, não centrais no dia a dia |
| `legacy runtime-smoke` | `legacy` | helper compatível, não canônico |
| aliases flat ocultos | `legacy` | compatibilidade de naming, não segunda taxonomia |

Nenhum comando precisou ser promovido como `experimental` no estado atual.
O diagnóstico correto foi: hoje o risco maior é ambiguidade documental, não
explosão real de comandos experimentais.

## 3. Maturidade E Lifecycle Dos Comandos

Foi definido o seguinte modelo:

- `stable core`: superfície padrão, recorrente e oficialmente suportada;
- `stable utility`: superfície menor, porém durável e justificada;
- `experimental`: superfície de prova, ainda não promovida;
- `legacy`: superfície mantida apenas por compatibilidade, migração ou fallback.

Também foram definidos critérios para:

- introdução de novos comandos, priorizando extensão de subcomandos existentes
  antes de criar novos entrypoints;
- promoção de experimentais apenas com uso repetido, naming estável e valor
  claro;
- consolidação quando houver sobreposição semântica;
- depreciação gradual com rotulagem explícita, aliases ocultos e retirada só
  depois de baixo valor residual.

## 4. Ajustes Aplicados

### Help do CLI

Foi ajustado o help principal do `raccoon-cli` para explicitar:

- a distinção entre `stable core`, `stable utility`, `experimental` e `legacy`;
- que não há comandos experimentais promovidos no help público atual;
- que `legacy runtime-smoke` e aliases flat são superfícies de compatibilidade.

### Documentação

Foram atualizados os seguintes documentos existentes:

- `tools/raccoon-cli/README.md`
- `docs/tooling/cli-overview.md`
- `docs/operations/raccoon-cli-command-reference.md`
- `docs/tooling/README.md`
- `docs/stages/INDEX.md`

Também foram criados:

- `docs/tooling/raccoon-cli-command-lifecycle-and-deprecation-strategy.md`
- `docs/tooling/raccoon-cli-command-catalog-maturity-model-and-governance.md`

### Correção de superfície canônica

O `tools/raccoon-cli/README.md` deixou de apresentar o catálogo principal em
forma flat e passou a priorizar a taxonomia agrupada atual, com seção explícita
de lifecycle e uma separação mais clara entre comandos canônicos, utilitários e
legados.

## 5. Estratégia Final De Depreciação/Evolução

### Estratégia aprovada

1. Manter `check`, `inspect` e `change` como núcleo estável da superfície.
2. Manter `snapshot*` e `baseline-drift` como utilitários estáveis, não como
   novo núcleo de workflow.
3. Tratar aliases flat como camada de compatibilidade oculta, não como catálogo
   paralelo.
4. Conter `legacy runtime-smoke` como helper congelado e replacement-oriented,
   sem revalorizá-lo como prova operacional.
5. Exigir avaliação de sobreposição antes de qualquer novo comando.

### Regra de evolução

Novos comandos devem nascer raramente. A ordem preferida é:

1. expandir comando existente;
2. adicionar flag ou modo de saída;
3. adicionar subcomando agrupado;
4. criar utilitário estável claramente distinto;
5. só então abrir uma superfície experimental.

Essa regra mantém o CLI útil sem torná-lo inchado ou burocrático.

## 6. Preparação Recomendada Para C24

Para o próximo stage, a preparação recomendada é:

1. Auditar se existe algum caso real em que aliases flat ainda apareçam em docs,
   scripts ou hábitos de equipe como se fossem superfície principal.
2. Verificar se o perfil `deep` do `check gate` merece mais isolamento semântico
   adicional por ainda tocar o helper legado.
3. Definir, se C24 tocar o CLI novamente, um critério leve de revisão de
   nomenclatura para impedir a criação de subcomandos sobrepostos em `inspect`
   e `change`.

## Evidência

- Help do CLI atualizado em `tools/raccoon-cli/src/cli/mod.rs`
- Política de lifecycle em
  `docs/tooling/raccoon-cli-command-lifecycle-and-deprecation-strategy.md`
- Catálogo e maturidade em
  `docs/tooling/raccoon-cli-command-catalog-maturity-model-and-governance.md`

## Validação

- `cargo test --manifest-path tools/raccoon-cli/Cargo.toml`
