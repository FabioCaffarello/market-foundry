# Stage C18 Report: Development CLI Reliability, Command Testing, And Trustworthiness

## 1. Resumo Executivo

O Stage C18 endureceu o `raccoon-cli` como ferramenta de desenvolvimento do
repositório, sem expandi-lo como produto.

O ganho principal foi tornar observável e testável a resolução implícita de
alvos dos comandos orientados a mudança. Antes, `tdd`, `recommend`, `briefing`
e `impact-map` podiam cair em “nenhum alvo” sem distinguir worktree limpo,
diretório fora de git ou falha de execução do `git`. Agora esses casos têm
fonte de input explícita, semântica humana/JSON consistente e testes de
regressão.

## 2. Diagnóstico De Confiabilidade Do CLI

### Comandos mais valiosos para desenvolvimento

- `check repo`
- `check gate` / `quality-gate`
- `tdd`
- `recommend`
- `briefing`
- `impact-map`
- `snapshot`, `snapshot-diff`, `baseline-drift`

### Fragilidade principal encontrada

Os comandos orientados a mudança dependiam de auto-detecção por `git status`,
mas tratavam erros operacionais de resolução como se fossem ausência real de
mudanças. Isso reduzia:

- previsibilidade do escopo analisado;
- clareza de outputs vazios;
- testabilidade do comportamento;
- confiança prática em worktrees incompletos, limpos ou fora de git.

## 3. Gaps De Testing E Trustworthiness

Os gaps centrais eram:

- ausência de contrato observável para a origem dos alvos;
- outputs humanos vazios sem explicação operacional suficiente;
- JSON sem campo específico para validar o modo de resolução de input;
- pouca proteção de regressão para cenários “not a git repository” e “filtragem estrutural”.

## 4. Melhorias Aplicadas

### Código

- `tools/raccoon-cli/src/io/git.rs`
  - passou a distinguir `Changed`, `Clean`, `NotRepository`, `Unavailable` e `Failed`.
- `tools/raccoon-cli/src/application/change_targets.rs`
  - introduziu `ChangeTargetResolution` e `ChangeTargetSource`;
  - tornou a resolução de alvos explícita e auditável.
- `tools/raccoon-cli/src/application/mod.rs`
  - injeta semântica de origem de input nos relatórios de `tdd`, `recommend`,
    `briefing` e `impact-map`.
- `tools/raccoon-cli/src/analyzers/tdd.rs`
  - adicionou `input_source`;
  - mostra motivo explícito quando não há arquivos detectados.
- `tools/raccoon-cli/src/analyzers/recommend.rs`
  - adicionou `input.detection_mode`;
  - explicita o motivo operacional de escopo vazio.
- `tools/raccoon-cli/src/analyzers/briefing.rs`
  - adicionou `input_source`;
  - explicita o motivo operacional de ausência de targets.
- `tools/raccoon-cli/src/analyzers/impact_map.rs`
  - adicionou `input_source` ao contrato observável.

### Testes

- `tools/raccoon-cli/tests/cli_integration.rs`
  - adicionou cobertura para:
    - `briefing` fora de git;
    - `recommend` fora de git com `detection_mode`;
    - `tdd` fora de git com motivo explícito;
    - `tdd` em worktree git real com filtragem estrutural de docs-only noise.

## 5. Critérios De Confiabilidade Definidos

Foram formalizados em:

- `docs/tooling/development-cli-reliability-and-command-testing-strategy.md`
- `docs/tooling/raccoon-cli-command-trustworthiness-and-error-semantics.md`

Critérios principais:

- proveniência de input deve ser visível;
- escopo vazio deve ser explicado;
- falha de check não pode ser confundida com falha de execução;
- human output e JSON devem descrever a mesma verdade operacional;
- stderr deve ficar reservado para erros reais do comando.

## 6. Validação Executada

- `cargo fmt --manifest-path tools/raccoon-cli/Cargo.toml`
- `cargo test --manifest-path tools/raccoon-cli/Cargo.toml --test cli_integration`
- `cargo test --manifest-path tools/raccoon-cli/Cargo.toml --test validation_matrix`
- `cargo test --manifest-path tools/raccoon-cli/Cargo.toml change_targets`

## 7. Preparação Recomendada Para C19

- elevar para `validation_matrix` os novos contratos de proveniência de input
  mais críticos, caso C19 continue endurecendo comandos centrais;
- revisar se outros comandos especializados precisam do mesmo padrão de
  `input_source`;
- manter a disciplina de não introduzir novas features no CLI sem antes definir
  o contrato de saída, semântica de erro e teste de integração correspondente.
