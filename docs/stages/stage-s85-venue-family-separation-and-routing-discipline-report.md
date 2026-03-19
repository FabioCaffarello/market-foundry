# Stage S85 — Venue Family Separation and Routing Discipline Report

> Refines the architectural boundary between the paper execution family and the venue execution family, documenting transitional bridges, ownership splits, and the migration path for clean venue-specific routing.

## Resumo Executivo

O S85 separa com disciplina a modelagem, routing e contracts entre a paper family e a venue family dentro do domínio de execução. O passo principal é documentar explicitamente o transitional bridge (o consumidor do execute que lê subjects do paper_order) e anotar todo o código envolvido com marcadores `TRANSITIONAL BRIDGE` para prevenir drift estrutural. Nenhuma mudança quebra a malha paper atual.

## Separação Arquitetural Aplicada

### 1. Registry Reorganizado (`execution_registry.go`)

O `ExecutionRegistry` foi reorganizado com seções explícitas:
- **Paper Family**: `PaperOrderSubmitted`, `PaperOrderLatest`
- **Venue Family**: `VenueMarketOrderFilled`, `VenueMarketOrderLatest`
- **Cross-Family**: `StatusLatest`, `ControlGet`, `ControlSet`

Streams agora documentam ownership:
- `EXECUTION_EVENTS` — paper family stream, com nota de uso transitional pelo execute
- `EXECUTION_FILL_EVENTS` — venue family stream, owner exclusivo do execute binary

O `ExecuteVenueMarketOrderIntakeConsumer` agora documenta explicitamente o transitional bridge e o caminho de migração.

### 2. Events com Family Annotations (`events.go`)

Cada família de eventos agora tem bloco de documentação com:
- Owner (binary responsável)
- Stream e subject pattern
- Consumers autorizados
- Semântica do evento

### 3. Settings com Family Documentation (`schema.go`)

O mapa `knownExecutionFamilies` agora documenta:
- Ownership de cada família (derive vs execute)
- Streams e KV buckets correspondentes
- Coexistência independente (habilitar venue não desabilita paper)

### 4. Transitional Bridge Annotations

Todos os pontos de acoplamento cross-family foram anotados com `TRANSITIONAL BRIDGE`:
- `execution_registry.go` — consumer spec e subject filter
- `execute_supervisor.go` — spawn do consumer
- `execute/messages.go` — tipo da mensagem interna

### 5. Drift Rules Atualizados

- `EXECUTION_DOCS` no raccoon-cli: expandido de 9 para 11 documentos
- `ED-1` atualizado nos dois docs de drift rules
- Descrição do durable `execute-venue-market-order-intake` anotada como transitional bridge

## Arquivos Alterados

### Código (Go)

| Arquivo | Tipo de Mudança |
|---------|----------------|
| `internal/adapters/nats/execution_registry.go` | Reorganização com seções de família, documentação de ownership, annotations de bridge |
| `internal/domain/execution/events.go` | Blocos de documentação por família com owner, stream, consumers |
| `internal/shared/settings/schema.go` | Documentação de family stages e coexistência |
| `internal/actors/scopes/execute/execute_supervisor.go` | Annotation de transitional bridge no spawn do consumer |
| `internal/actors/scopes/execute/messages.go` | Annotation de transitional bridge no tipo de mensagem |
| `internal/actors/scopes/store/store_supervisor.go` | Comentários de família nos pipelines de execução |

### Tooling (Rust)

| Arquivo | Tipo de Mudança |
|---------|----------------|
| `tools/raccoon-cli/src/analyzers/drift_detect.rs` | `EXECUTION_DOCS` expandido para 11, description de durable atualizada |

### Documentação de Drift Rules

| Arquivo | Tipo de Mudança |
|---------|----------------|
| `docs/tooling/cli-execution-drift-rules.md` | ED-1 atualizado (11 docs), family annotations em durables |
| `docs/tooling/cli-execute-drift-rules.md` | `EXECUTION_DOCS` count atualizado |

### Documentos de Arquitetura (Novos)

| Arquivo | Conteúdo |
|---------|----------|
| `docs/architecture/execution-family-separation-after-paper-step.md` | Family definitions, transitional bridge, migration path, invariants |
| `docs/architecture/venue-routing-and-ownership-split.md` | Subject hierarchy, stream/KV ownership, binary responsibility matrix, routing rules |

## Riscos e Compatibilidades

### Compatibilidade

- **Zero breaking changes**: Nenhum subject, consumer, bucket ou event type foi alterado.
- **Paper flow intacto**: Derive → EXECUTION_EVENTS → store projection continua idêntico.
- **Venue flow intacto**: Execute → EXECUTION_FILL_EVENTS → store fill projection continua idêntico.
- **Transitional bridge preservado**: O execute continua consumindo paper_order subjects em paper mode.

### Riscos Documentados

| Risco | Mitigação |
|-------|-----------|
| Bridge pode persistir além do necessário | Annotations `TRANSITIONAL BRIDGE` em 4 arquivos + doc dedicado com migration path |
| Novo consumer pode subscrever paper subjects por engano | Invariant documentado: nenhum novo consumer deve usar paper subjects para processamento venue |
| Drift entre docs e código | ED-1 agora verifica 11 docs, drift rules rastreiam ambos documentos de separação |

## Preparação Recomendada para S86

### Opção A: Venue Intent Event Type

Introduzir `VenueOrderIntentEvent` como evento dedicado para intake do execute binary:
- Novo subject: `execution.events.venue_market_order.submitted.{s}.{sym}.{tf}`
- Derive produz ambos eventos quando `venue_market_order` está habilitado
- Execute migra intake consumer para o novo subject
- Bridge removido

### Opção B: Hardening Operacional

Antes de introduzir venue intent events, validar a malha atual com:
- Smoke tests multi-symbol com execute binary ativo
- Validação de trace chain end-to-end (derive → execute → store)
- Verificação de staleness guard em cenários de atraso

### Recomendação

**Opção B primeiro, depois A.** O hardening operacional valida que a topologia atual é estável antes de introduzir mudanças de routing. A separação conceitual do S85 já protege contra drift — o passo seguinte pode ser pragmático.

## Decisões Tomadas

| Decisão | Justificativa |
|---------|---------------|
| Não introduzir venue intent event type neste estágio | O bridge funciona, está documentado, e o risco é controlado. Introduzir prematuramente criaria churn desnecessário. |
| Anotar código em vez de refatorar | Annotations são reversíveis e não criam risco de regressão. A refatoração virá com o venue intent event. |
| Expandir EXECUTION_DOCS para 11 | Drift rules são a melhor proteção contra documentação que desaparece silenciosamente. |
| Documentar migration path em 5 passos | Passos atômicos e verificáveis previnem migração incompleta. |
