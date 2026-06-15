use crate::output::OutputFormat;
use clap::{Args, Parser, Subcommand, ValueEnum};

#[derive(Parser)]
#[command(
    name = "raccoon-cli",
    about = "Repository support CLI for market-foundry",
    long_about = "Repository support CLI for market-foundry.\n\n\
        Raccoon CLI is the repository intelligence layer for maintenance, validation,\n\
        inspection, and safe-change analysis. It is not a product control plane,\n\
        should remain isolated from the live Go runtime, and must not become a\n\
        parallel runtime orchestrator.\n\n\
        Contract with `make`:\n  \
          `make` owns the canonical public workflow and runtime/proof entrypoints\n  \
          `raccoon-cli` owns expert inspection, impact analysis, tdd guidance,\n  \
          drift detection, and architecture safety\n\n\
        Canonical taxonomy:\n  \
          check    repository guard rails and audits\n  \
          inspect  read-only structural and contract analysis\n  \
          change   impact mapping and validation guidance\n  \
          snapshot stable utility baseline and drift commands\n  \
          legacy   fragile or deprecated helper flows\n\n\
        Lifecycle states:\n  \
          stable       default supported command surface for everyday use\n  \
          utility      narrower but durable support surface for focused tasks\n  \
          experimental limited-scope proving surface; promote only after repeated use\n  \
          legacy       retained only for compatibility, migration, or bounded fallback",
    version,
    propagate_version = true,
    after_help = "Surface maturity:\n  \
        stable core     `check`, `inspect`, `change`\n  \
        stable utility  `snapshot`, `snapshot-diff`, `baseline-drift`\n  \
        legacy          `legacy runtime-smoke` and hidden flat compatibility aliases\n  \
        experimental    none currently promoted in the public help surface\n\n\
        Canonical usage:\n  \
        raccoon-cli check repo\n  \
        raccoon-cli check gate --profile ci --json\n  \
        raccoon-cli inspect symbol ConfigSet --lsp\n  \
        raccoon-cli change tdd\n  \
        raccoon-cli snapshot --output baseline.json\n  \
        raccoon-cli legacy runtime-smoke\n\n\
        Compatibility:\n  \
        Existing flat commands such as `doctor`, `quality-gate`, `symbol-trace`,\n  \
        `tdd`, and `runtime-smoke` remain supported as hidden compatibility aliases.\n\
        New guidance should prefer grouped commands and Makefile wrappers.\n\n\
        Operational proof rule:\n  \
        Prefer `make smoke*` for runtime proof. The CLI deep profile and runtime-smoke\n  \
        helper are compatibility surfaces, not the proof-of-record operator entrypoint."
)]
pub(crate) struct Cli {
    /// Output as JSON instead of human-readable text
    #[arg(long, global = true)]
    pub(crate) json: bool,

    /// Show detailed findings for all checks, not just failures
    #[arg(long, short, global = true)]
    pub(crate) verbose: bool,

    /// Path to the project root (defaults to current directory)
    #[arg(long, global = true, default_value = ".")]
    pub(crate) project_root: std::path::PathBuf,

    #[command(subcommand)]
    pub(crate) command: Commands,
}

impl Cli {
    pub(crate) fn output_format(&self) -> OutputFormat {
        if self.json {
            OutputFormat::Json
        } else if self.verbose {
            OutputFormat::HumanVerbose
        } else {
            OutputFormat::Human
        }
    }
}

/// Execution profile for quality-gate
#[derive(Debug, Clone, Copy, PartialEq, Eq, ValueEnum)]
pub(crate) enum GateProfile {
    /// Static checks only (topology-doctor + contract-audit), no infra needed
    Fast,
    /// Same as fast, but warnings become failures (strict for CI)
    Ci,
    /// All checks including the legacy runtime-smoke helper (requires running environment)
    Deep,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct RuntimeSmokeArgs {
    /// Base URL for the HTTP API
    #[arg(long, default_value = "http://127.0.0.1:8080", value_name = "URL")]
    pub(crate) base_url: String,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct SymbolTraceArgs {
    /// The symbol name to trace (type, function, constant, or variable)
    #[arg(value_name = "SYMBOL")]
    pub(crate) symbol: String,
    /// Enrich with gopls definitions, references, and hover info
    #[arg(long, conflicts_with = "no_lsp")]
    pub(crate) lsp: bool,
    /// Skip gopls even if --lsp is set (useful for benchmarking AST-only)
    #[arg(long, conflicts_with = "lsp")]
    pub(crate) no_lsp: bool,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct ImpactMapArgs {
    /// Enrich with gopls references for exported symbols
    #[arg(long, conflicts_with = "no_lsp")]
    pub(crate) lsp: bool,
    /// Skip gopls even if --lsp is set
    #[arg(long, conflicts_with = "lsp")]
    pub(crate) no_lsp: bool,
    /// Targets to analyze (files, packages, or symbols). If omitted, uses filtered git status changes.
    #[arg(trailing_var_arg = true, value_name = "TARGET")]
    pub(crate) targets: Vec<String>,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct TddArgs {
    /// Files you plan to change (if omitted, uses filtered git status changes)
    #[arg(trailing_var_arg = true, value_name = "TARGET")]
    pub(crate) files: Vec<String>,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct QualityGateArgs {
    /// Execution profile (default: fast)
    #[arg(long, value_enum, default_value_t = GateProfile::Fast)]
    pub(crate) profile: GateProfile,
    /// Base URL for runtime-smoke (only used with --profile deep)
    #[arg(long, default_value = "http://127.0.0.1:8080", value_name = "URL")]
    pub(crate) base_url: String,
    /// Stop after the first failing step (skip remaining steps)
    #[arg(long)]
    pub(crate) fail_fast: bool,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct LspEnrichArgs {
    /// The symbol name to enrich
    #[arg(value_name = "SYMBOL")]
    pub(crate) symbol: String,
    /// Skip gopls and return AST-only results
    #[arg(long)]
    pub(crate) no_lsp: bool,
    /// Timeout in seconds for gopls requests (default: 30)
    #[arg(long, default_value_t = 30)]
    pub(crate) timeout: u64,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct RenameSafetyArgs {
    /// The symbol name to evaluate for renaming
    #[arg(value_name = "SYMBOL")]
    pub(crate) symbol: String,
    /// Optional new name (checks for conflicts)
    #[arg(long = "to", value_name = "NEW_NAME")]
    pub(crate) new_name: Option<String>,
    /// Enrich with gopls references for deeper coverage
    #[arg(long, conflicts_with = "no_lsp")]
    pub(crate) lsp: bool,
    /// Skip gopls even if --lsp is set
    #[arg(long, conflicts_with = "lsp")]
    pub(crate) no_lsp: bool,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct ContractUsageMapArgs {
    /// Enrich with gopls references for deeper coverage
    #[arg(long, conflicts_with = "no_lsp")]
    pub(crate) lsp: bool,
    /// Skip gopls even if --lsp is set
    #[arg(long, conflicts_with = "lsp")]
    pub(crate) no_lsp: bool,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct BriefingArgs {
    /// Enrich with gopls references for deeper coverage
    #[arg(long, conflicts_with = "no_lsp")]
    pub(crate) lsp: bool,
    /// Skip gopls even if --lsp is set
    #[arg(long, conflicts_with = "lsp")]
    pub(crate) no_lsp: bool,
    /// Targets to analyze (files, packages, or symbols). If omitted, uses filtered git status changes.
    #[arg(trailing_var_arg = true, value_name = "TARGET")]
    pub(crate) targets: Vec<String>,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct BaselineDriftArgs {
    /// Path to the baseline snapshot JSON file
    #[arg(value_name = "BASELINE_JSON")]
    pub(crate) baseline: std::path::PathBuf,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct SnapshotArgs {
    /// Save JSON output to a file instead of stdout
    #[arg(long, short, value_name = "OUTPUT_JSON")]
    pub(crate) output: Option<std::path::PathBuf>,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct SnapshotDiffArgs {
    /// Path to the 'before' snapshot JSON file
    #[arg(value_name = "BEFORE_JSON")]
    pub(crate) before: std::path::PathBuf,
    /// Path to the 'after' snapshot JSON file (omit with --after-live to use current project)
    #[arg(required_unless_present = "after_live", value_name = "AFTER_JSON")]
    pub(crate) after: Option<std::path::PathBuf>,
    /// Use a live snapshot of the current project as 'after' instead of a file
    #[arg(long)]
    pub(crate) after_live: bool,
}

#[derive(Args, Debug, Clone)]
pub(crate) struct RecommendArgs {
    /// Optional baseline snapshot for drift-aware recommendations
    #[arg(long, value_name = "BASELINE_JSON")]
    pub(crate) baseline: Option<std::path::PathBuf>,
    /// Files to analyze (if omitted, uses filtered git status changes)
    #[arg(trailing_var_arg = true, value_name = "TARGET")]
    pub(crate) files: Vec<String>,
}

#[derive(Subcommand)]
pub(crate) enum CheckCommands {
    /// Validate repository structure and required support paths
    #[command(
        name = "repo",
        visible_alias = "doctor",
        after_help = "Examples:\n  \
            raccoon-cli check repo\n  \
            raccoon-cli doctor"
    )]
    Repo,
    /// Validate runtime topology declarations across compose, configs, and source
    #[command(
        name = "topology",
        visible_alias = "topology-doctor",
        after_help = "Examples:\n  \
            raccoon-cli check topology\n  \
            raccoon-cli check topology"
    )]
    Topology,
    /// Audit messaging contracts and transport invariants
    #[command(
        name = "contracts",
        visible_alias = "contract-audit",
        after_help = "Examples:\n  \
            raccoon-cli check contracts\n  \
            raccoon-cli check contracts"
    )]
    Contracts,
    /// Validate runtime binding declarations and routing alignment
    #[command(
        name = "bindings",
        visible_alias = "runtime-bindings",
        after_help = "Examples:\n  \
            raccoon-cli check bindings\n  \
            raccoon-cli check bindings"
    )]
    Bindings,
    /// Enforce clean-architecture layer boundaries
    #[command(
        name = "arch",
        visible_alias = "arch-guard",
        long_about = "Detect violations of the clean architecture layer rules using structural \
            analysis (AST-based codeintel index). Goes beyond import-path checking to inspect \
            type definitions, struct fields, interface signatures, and function parameters.\n\n\
            Rules enforced:\n  \
              1.  Layer dependency direction (domain → application → adapters → actors → interfaces)\n  \
              2.  Domain purity (no infrastructure imports in domain/)\n  \
              3.  Application isolation (no direct adapter imports)\n  \
              4.  Interfaces isolation (no adapter/actor imports in HTTP handlers)\n  \
              5.  Cmd boundary (AST type counting — cmd/ wires, does not define models)\n  \
              6.  Tooling boundary (tools/ must not contain Go modules)\n  \
              7.  No cross-cmd imports (binaries are independently deployable)\n  \
              8.  Deploy boundary (no hardcoded deploy/ paths in Go source)\n  \
              9.  Port contract leaks (port interfaces must not reference infra types)\n  \
              10. Domain type contamination (struct fields must not embed infra types)\n  \
              11. Exported signature leaks (domain/application funcs must not expose infra types)",
        after_help = "Examples:\n  \
            raccoon-cli check arch\n  \
            raccoon-cli check arch"
    )]
    Arch,
    /// Detect drift between docs, configs, compose, and source wiring
    #[command(
        name = "drift",
        visible_alias = "drift-detect",
        long_about = "Detect drift between what the system declares, what it configures, what the source \
            wires, and what the documentation says.\n\n\
            Drift classes:\n  \
              1. Config ↔ Compose: services in configs vs compose, transport dependency alignment\n  \
              2. Config ↔ Source: stream/durable/subject constants vs config declarations\n  \
              3. Binding ↔ Topology: declared bindings vs routing infrastructure\n  \
              4. Workflow ↔ Reality: DEVELOPMENT.md targets vs actual Makefile targets\n  \
              5. Contract ↔ Domain: registry event specs vs domain event definitions\n  \
              6. Compose ↔ Profiles: profile assignments vs Makefile up-* targets",
        after_help = "Examples:\n  \
            raccoon-cli check drift\n  \
            raccoon-cli check drift"
    )]
    Drift,
    /// Validate proto/registry.json ↔ .proto files ↔ generated Go sync
    #[command(
        name = "proto",
        visible_alias = "check-proto",
        long_about = "Statically validate that proto/registry.json, .proto files under proto/, and \
            generated Go under internal/shared/contracts/ stay in sync. Enforces ADR-0018 \
            (protobuf contract layer) acceptance criterion 5 (post-2026-05-25 erratum).\n\n\
            Checks performed:\n  \
              1. Every registry entry references an existing .proto file (PROTO-G4).\n  \
              2. Every .proto under proto/ has a registry entry (PROTO-G4).\n  \
              3. Every registry entry has the corresponding *.pb.go under internal/shared/contracts/.\n  \
              4. Every .proto declares option go_package matching its internal/shared/contracts/ path.\n  \
              5. Smoke boundary: no Go file under internal/domain/ imports internal/shared/contracts (PROTO-G3).",
        after_help = "Examples:\n  \
            raccoon-cli check proto\n  \
            raccoon-cli check-proto"
    )]
    Proto,
    /// Enforce ADR-0019 INV-D1 (domain purity)
    #[command(
        name = "determinism",
        visible_alias = "check-determinism",
        long_about = "Statically enforce ADR-0019 INV-D1 (domain purity) on \
            internal/domain/ production code. Scans .go files (excluding \
            *_test.go) for direct use of non-deterministic stdlib facilities. \
            Production code MUST source time via clock.Clock, randomness via \
            random.Source, and context.Context explicitly through function \
            parameters.\n\n\
            Checks performed:\n  \
              1. Banned imports: math/rand, math/rand/v2, crypto/rand.\n  \
              2. Banned symbols: time.Now, time.Since, time.Until, time.Tick, \
            os.Getenv, os.Args, context.Background, context.TODO.\n\n\
            Test files (*_test.go) are EXEMPT — the real enforcement for \
            tests is the determinism gates INV-D3/INV-D4 (golden tests + N=50 \
            byte-stability).",
        after_help = "Examples:\n  \
            raccoon-cli check determinism\n  \
            raccoon-cli check-determinism"
    )]
    Determinism,
    /// Enforce PROGRAM-0003 / ADR-0024 metrics invariant
    #[command(
        name = "metrics",
        visible_alias = "check-metrics",
        long_about = "Statically enforce PROGRAM-0003 / ADR-0024 invariant: every \
            long-running cmd/*/main.go binary exposes Prometheus /metrics. \
            Declarative algorithm: reads tools/raccoon-cli/policies/binaries.toml \
            for the one_shot allowlist; every other binary must register /metrics \
            via healthz.NewHealthServer, mux.Handle(\"GET /metrics\", ...), or \
            metrics.HandlerFunc.\n\n\
            Adding a one-shot binary requires editing the policy file — the \
            analyzer cannot pass on a long-running binary that drops the \
            /metrics endpoint accidentally.",
        after_help = "Examples:\n  \
            raccoon-cli check metrics\n  \
            raccoon-cli check-metrics"
    )]
    Metrics,
    /// Enforce ADR-0021 / H-6.a canonical-instrument invariant
    #[command(
        name = "instruments",
        visible_alias = "check-instruments",
        long_about = "Statically enforce ADR-0021 / H-6.a invariant: every \
            exchange adapter under internal/adapters/exchanges/<pkg>/ \
            normalizes venue-native symbols to CanonicalInstrument via \
            the canonical constructor at the adapter / domain boundary.\n\n\
            Declarative algorithm: reads tools/raccoon-cli/policies/adapters.toml \
            for the allowlist of recognized adapter packages; every \
            adapter directory must (a) be declared in the allowlist, \
            (b) import internal/domain/instrument, and (c) call \
            instrument.New(...) or instrument.FromSymbol(...) — the \
            constructors that validate the asset / contract shape.\n\n\
            Adding a new venue requires editing the policy file — the \
            analyzer fail-stops on an unrecognized adapter directory, so \
            an adapter cannot silently ship without canonical-instrument \
            adoption.",
        after_help = "Examples:\n  \
            raccoon-cli check instruments\n  \
            raccoon-cli check-instruments"
    )]
    Instruments,
    /// Enforce ADR-0009 erratum / H-6.e canonical subject-token invariant
    #[command(
        name = "subjects",
        visible_alias = "check-subjects",
        long_about = "Statically enforce the ADR-0009 erratum (2026-06-10, \
            Onda H-6.e): the {symbol} token of every published NATS subject \
            is derived exclusively via CanonicalInstrument.SubjectToken() — \
            never via the transitory VenueSymbol() helper.\n\n\
            Declarative algorithm: reads tools/raccoon-cli/policies/subjects.toml \
            and scans production publisher files under the declared scan_root. \
            The scan is BLOCK-scoped to `subject := fmt.Sprintf(` calls: KV \
            partition keys, dedup keys, and log labels legitimately use \
            VenueSymbol() until sub-onda H-6.e.2 and are not flagged.",
        after_help = "Examples:\n  \
            raccoon-cli check subjects\n  \
            raccoon-cli check-subjects"
    )]
    Subjects,
    /// Enforce ADR-0022 multi-venue normalization policy (R1–R3)
    #[command(
        name = "venue-parity",
        visible_alias = "check-venue-parity",
        long_about = "Statically enforce ADR-0022 (multi-venue \
            normalization policy, Onda H-7.a) rules R1-R3; this analyzer \
            is rule R4.\n\n\
            R1: every venue adapter package under the declared \
            adapters_root ships a static Capabilities() declaration \
            (empty declarations require an explicit justifying comment). \
            R2: the gateway registers GET /venues/capabilities. \
            R3: the ingest producer-boundary guard rejects undeclared \
            (event_type, contract) pairs observably — Allows() check + \
            undeclared-event counter increment.\n\n\
            Declarative algorithm: reads \
            tools/raccoon-cli/policies/venue_parity.toml. A new venue \
            adapter directory fail-stops until Capabilities() ships.",
        after_help = "Examples:\n  \
            raccoon-cli check venue-parity\n  \
            raccoon-cli check-venue-parity"
    )]
    VenueParity,
    /// Enforce ADR-0027 insights decision-support read-only invariant
    #[command(
        name = "insights",
        visible_alias = "check-insights",
        long_about = "Statically enforce ADR-0027 (insights são \
            decision-support read-only, Onda H-8.a) invariant I2.\n\n\
            Read-only domain: no production file under \
            internal/domain/insights imports the directive chain \
            (strategy/decision/risk/execution). Stream-bound \
            publisher: the insights publisher references INSIGHTS_EVENTS \
            and no directive-chain stream.\n\n\
            Declarative algorithm: reads \
            tools/raccoon-cli/policies/insights.toml.",
        after_help = "Examples:\n  \
            raccoon-cli check insights\n  \
            raccoon-cli check-insights"
    )]
    Insights,
    /// Enforce ADR-0028 delivery read-only / reader-only invariant
    #[command(
        name = "delivery",
        visible_alias = "check-delivery",
        long_about = "Statically enforce ADR-0028 (delivery is read-only \
            transport, Onda H-11.d) invariants I1/I5.\n\n\
            Reader-only adapter: no production file under \
            internal/adapters/nats/natsdelivery publishes to a stream \
            (delivery consumes INSIGHTS_EVENTS and never writes back — \
            single-writer, ADR-0008). Stream-bound consumer: the delivery \
            consumer declares the deliver-insights durable on \
            INSIGHTS_EVENTS.\n\n\
            Declarative algorithm: reads \
            tools/raccoon-cli/policies/delivery.toml.",
        after_help = "Examples:\n  \
            raccoon-cli check delivery\n  \
            raccoon-cli check-delivery"
    )]
    Delivery,
    /// Run the consolidated repository guard-rail profile
    #[command(
        name = "gate",
        visible_alias = "quality-gate",
        long_about = "Run consolidated repository guard rails: doctor + topology + contracts + \
            runtime-bindings + arch-guard + drift-detect.\n\n\
            Profiles:\n  \
              fast  — static repository checks only (default)\n  \
              ci    — same as fast, warnings become failures\n  \
              deep  — includes the legacy runtime-smoke helper; use `make smoke*` for the canonical operational-proof surface",
        after_help = "Examples:\n  \
            raccoon-cli check gate\n  \
            raccoon-cli check gate --profile ci --json\n  \
            raccoon-cli check gate --profile deep"
    )]
    Gate(QualityGateArgs),
}

#[derive(Subcommand)]
pub(crate) enum InspectCommands {
    /// Trace a symbol through definitions, references, and contracts
    #[command(
        name = "symbol",
        visible_alias = "symbol-trace",
        long_about = "Trace a symbol (type, function, constant, variable) across the Go codebase.\n\n\
            Uses the codeintel AST index to find:\n  \
              - Where the symbol is defined (type, file, line, visibility)\n  \
              - Where it is structurally referenced (struct fields, function params/returns,\n    \
                receivers, interface embeds, type aliases, const/var type hints)\n  \
              - Which packages are involved\n  \
              - Contract connections (ports, message types, interfaces)\n  \
              - Recommended raccoon-cli checks\n\n\
            With --lsp, enriches results via gopls. Use --no-lsp to force AST-only output.",
        after_help = "Examples:\n  \
            raccoon-cli inspect symbol ConfigSet\n  \
            raccoon-cli inspect symbol ConfigSet --lsp\n  \
            raccoon-cli inspect symbol ConfigSet"
    )]
    Symbol(SymbolTraceArgs),
    /// Show gopls-backed enrichment for a symbol
    #[command(
        name = "lsp",
        visible_alias = "lsp-enrich",
        long_about = "Enrich a symbol with semantic information using the gopls LSP bridge.\n\n\
            Combines deterministic AST facts from codeintel with type-resolved definitions,\n\
            cross-package references, and hover/type info from gopls.\n\n\
            If gopls is not available, returns AST-only results with a clear indication that\n\
            LSP enrichment was unavailable.",
        after_help = "Examples:\n  \
            raccoon-cli inspect lsp ConfigSet\n  \
            raccoon-cli inspect lsp ConfigSet"
    )]
    Lsp(LspEnrichArgs),
    /// Map where contracts are defined, propagated, consumed, and validated
    #[command(
        name = "contract-usage",
        visible_aliases = ["contracts", "contract-usage-map"],
        long_about = "Map real contract usage across the repository using AST structural analysis.\n\n\
            For each contract type (envelopes, commands, queries, replies, events, records, bindings, etc.):\n  \
              - Definition: where the type is declared\n  \
              - Construction: factory functions, builder methods, struct literals\n  \
              - Propagation: parameters, returns, embeddings, interface methods\n  \
              - Consumption: handlers, decoders, field access\n  \
              - Validation: Validate/Normalize methods",
        after_help = "Examples:\n  \
            raccoon-cli inspect contract-usage\n  \
            raccoon-cli inspect contract-usage --lsp\n  \
            raccoon-cli contract-usage-map"
    )]
    ContractUsage(ContractUsageMapArgs),
    /// Show which guard rails and Go tests cover sensitive repository areas
    #[command(
        name = "coverage",
        visible_alias = "coverage-map",
        long_about = "Show which quality dimensions and scenarios cover each sensitive area of the codebase.\n\n\
            Reports:\n  \
              - All quality dimensions (static and runtime)\n  \
              - Sensitive areas with their coverage status\n  \
              - Go test file distribution\n  \
              - Coverage gaps that need attention",
        after_help = "Examples:\n  \
            raccoon-cli inspect coverage\n  \
            raccoon-cli inspect coverage"
    )]
    Coverage,
}

#[derive(Subcommand)]
pub(crate) enum ChangeCommands {
    /// Map the structural impact of pending or explicit targets
    #[command(
        name = "impact",
        visible_alias = "impact-map",
        long_about = "Map the potential impact of changes to files, packages, or symbols.\n\n\
            Uses the codeintel AST index to trace import relationships, exported symbols,\n\
            and contract surface. Differentiates observed facts from inferred risks.\n\n\
            If no targets are given, uses `git status` and ignores documentation-only changes when possible.",
        after_help = "Examples:\n  \
            raccoon-cli change impact internal/domain/configctl/config.go\n  \
            raccoon-cli change impact ConfigSet --lsp\n  \
            raccoon-cli change impact"
    )]
    Impact(ImpactMapArgs),
    /// Generate impact-driven validation guidance for a change set
    #[command(
        name = "tdd",
        long_about = "Impact-driven TDD guidance using AST analysis and structural impact tracing.\n\n\
            Given a list of files you plan to change (or auto-detected via `git status`):\n  \
              1. Traces exported symbols, dependents, and contract surface\n  \
              2. Identifies affected sensitive areas and coverage gaps\n  \
              3. Finds existing tests near the changed code\n  \
              4. Recommends specific checks, scenarios, and gate profile\n  \
              5. Shows BEFORE/AFTER commands for disciplined TDD flow\n\n\
            Auto-detection ignores documentation-only changes when possible.",
        after_help = "Examples:\n  \
            raccoon-cli change tdd internal/adapters/nats/codec.go\n  \
            raccoon-cli change tdd\n  \
            raccoon-cli change tdd"
    )]
    Tdd(TddArgs),
    /// Generate a concise, auditable briefing for an area or change set
    #[command(
        name = "briefing",
        long_about = "Generate a short, dense briefing combining impact analysis, architecture checks,\n\
            contract health, and TDD guidance for a given set of targets.\n\n\
            Designed for pasting into agent context or reading during development.\n\
            Auto-detection ignores documentation-only changes when possible.",
        after_help = "Examples:\n  \
            raccoon-cli change briefing internal/domain/configctl/config.go\n  \
            raccoon-cli change briefing\n  \
            raccoon-cli change briefing"
    )]
    Briefing(BriefingArgs),
    /// Recommend what to validate after a change
    #[command(
        name = "recommend",
        long_about = "Generate prioritized, actionable recommendations for what to validate after a change.\n\n\
            Composes signals from impact-map, tdd guidance, and optional baseline drift.\n\
            Auto-detection ignores documentation-only changes when possible.",
        after_help = "Examples:\n  \
            raccoon-cli change recommend\n  \
            raccoon-cli change recommend --baseline snapshot.json\n  \
            raccoon-cli change recommend"
    )]
    Recommend(RecommendArgs),
    /// Evaluate rename safety before touching shared contracts or symbols
    #[command(
        name = "rename",
        visible_alias = "rename-safety",
        long_about = "Evaluate the safety of renaming a Go symbol before performing the rename.\n\n\
            Uses the codeintel AST index to trace definitions, structural references,\n\
            contract surface, and sensitive areas. Does not execute the rename.",
        after_help = "Examples:\n  \
            raccoon-cli change rename ConfigSet\n  \
            raccoon-cli change rename ConfigSet --to QualityConfigSet --lsp\n  \
            raccoon-cli change rename ConfigSet"
    )]
    Rename(RenameSafetyArgs),
}

#[derive(Subcommand)]
pub(crate) enum LegacyCommands {
    /// Deprecated runtime smoke helper kept for compatibility
    #[command(
        name = "runtime-smoke",
        long_about = "Run the legacy runtime smoke helper kept for compatibility.\n\n\
            Prefer Makefile operational flows instead:\n  \
              - `make smoke`\n  \
              - `make smoke-multi`\n  \
              - `make smoke-analytical`\n  \
              - `make smoke-operational`\n  \
              - `make smoke-restart-recovery`\n\n\
            Use this command only when you specifically need the historical CLI wrapper.",
        after_help = "Examples:\n  \
            raccoon-cli legacy runtime-smoke\n  \
            raccoon-cli runtime-smoke"
    )]
    RuntimeSmoke(RuntimeSmokeArgs),
}

#[derive(Subcommand)]
pub(crate) enum Commands {
    /// Repository guard rails and audits
    #[command(
        subcommand,
        after_help = "Examples:\n  \
            raccoon-cli check repo\n  \
            raccoon-cli check gate --profile ci\n  \
            raccoon-cli check arch"
    )]
    Check(CheckCommands),
    /// Read-only structural and contract analysis
    #[command(
        subcommand,
        after_help = "Examples:\n  \
            raccoon-cli inspect symbol ConfigSet\n  \
            raccoon-cli inspect contract-usage --lsp\n  \
            raccoon-cli inspect coverage"
    )]
    Inspect(InspectCommands),
    /// Change-focused impact analysis and validation guidance
    #[command(
        subcommand,
        after_help = "Examples:\n  \
            raccoon-cli change impact\n  \
            raccoon-cli change tdd\n  \
            raccoon-cli change recommend"
    )]
    Change(ChangeCommands),
    /// Fragile or deprecated helper flows
    #[command(
        subcommand,
        after_help = "Examples:\n  \
            raccoon-cli legacy runtime-smoke\n\n\
            Prefer Makefile operational flows for runtime validation."
    )]
    Legacy(LegacyCommands),
    #[command(hide = true)]
    #[command(after_help = "Examples:\n  \
        raccoon-cli doctor\n  \
        raccoon-cli --project-root /path/to/market-foundry doctor")]
    Doctor,
    #[command(hide = true)]
    #[command(after_help = "Examples:\n  \
        raccoon-cli topology-doctor\n  \
        raccoon-cli --json topology-doctor")]
    TopologyDoctor,
    #[command(hide = true)]
    #[command(after_help = "Examples:\n  \
        raccoon-cli contract-audit\n  \
        raccoon-cli --json contract-audit | jq '.checks[] | select(.status == \"fail\")'")]
    ContractAudit,
    #[command(hide = true)]
    #[command(after_help = "Examples:\n  \
        raccoon-cli runtime-bindings\n  \
        raccoon-cli --json runtime-bindings")]
    RuntimeBindings,
    #[command(hide = true)]
    #[command(
        long_about = "Detect drift between what the system declares, what it configures, what the source \
            wires, and what the documentation says.\n\n\
            Drift classes:\n  \
              1. Config ↔ Compose: services in configs vs compose, transport dependency alignment\n  \
              2. Config ↔ Source: stream/durable/subject constants vs config declarations\n  \
              3. Binding ↔ Topology: declared bindings vs routing infrastructure\n  \
              4. Workflow ↔ Reality: DEVELOPMENT.md targets vs actual Makefile targets\n  \
              5. Contract ↔ Domain: registry event specs vs domain event definitions\n  \
              6. Compose ↔ Profiles: profile assignments vs Makefile up-* targets",
        after_help = "Examples:\n  \
            raccoon-cli drift-detect\n  \
            raccoon-cli --json drift-detect\n  \
            raccoon-cli -v drift-detect"
    )]
    DriftDetect,
    #[command(hide = true)]
    #[command(
        long_about = "Detect violations of the clean architecture layer rules using structural \
            analysis (AST-based codeintel index). Goes beyond import-path checking to inspect \
            type definitions, struct fields, interface signatures, and function parameters.\n\n\
            Rules enforced:\n  \
              1.  Layer dependency direction (domain → application → adapters → actors → interfaces)\n  \
              2.  Domain purity (no infrastructure imports in domain/)\n  \
              3.  Application isolation (no direct adapter imports)\n  \
              4.  Interfaces isolation (no adapter/actor imports in HTTP handlers)\n  \
              5.  Cmd boundary (AST type counting — cmd/ wires, does not define models)\n  \
              6.  Tooling boundary (tools/ must not contain Go modules)\n  \
              7.  No cross-cmd imports (binaries are independently deployable)\n  \
              8.  Deploy boundary (no hardcoded deploy/ paths in Go source)\n  \
              9.  Port contract leaks (port interfaces must not reference infra types)\n  \
              10. Domain type contamination (struct fields must not embed infra types)\n  \
              11. Exported signature leaks (domain/application funcs must not expose infra types)",
        after_help = "Examples:\n  \
            raccoon-cli arch-guard\n  \
            raccoon-cli --json arch-guard\n  \
            raccoon-cli arch-guard --project-root /path/to/market-foundry"
    )]
    ArchGuard,
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli symbol-trace ConfigSet\n  \
            raccoon-cli symbol-trace --lsp ConfigSet"
    )]
    SymbolTrace(SymbolTraceArgs),
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli runtime-smoke\n  \
            raccoon-cli legacy runtime-smoke"
    )]
    RuntimeSmoke(RuntimeSmokeArgs),
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli impact-map internal/domain/configctl/config.go\n  \
            raccoon-cli change impact"
    )]
    ImpactMap(ImpactMapArgs),
    /// Show quality coverage map: which dimensions, scenarios, and Go tests cover each sensitive area
    #[command(hide = true)]
    #[command(
        long_about = "Show which quality dimensions and scenarios cover each sensitive area of the codebase.\n\n\
            Reports:\n  \
              - All quality dimensions (static and runtime)\n  \
              - Sensitive areas with their coverage status\n  \
              - Go test file distribution\n  \
              - Coverage gaps that need attention",
        after_help = "Examples:\n  \
            raccoon-cli coverage-map\n  \
            raccoon-cli --json coverage-map\n  \
            raccoon-cli -v coverage-map"
    )]
    CoverageMap,
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli tdd internal/adapters/nats/codec.go\n  \
            raccoon-cli change tdd"
    )]
    Tdd(TddArgs),
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli quality-gate\n  \
            raccoon-cli check gate --profile ci"
    )]
    QualityGate(QualityGateArgs),
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli lsp-enrich ConfigSet\n  \
            raccoon-cli inspect lsp ConfigSet"
    )]
    LspEnrich(LspEnrichArgs),
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli rename-safety ConfigSet\n  \
            raccoon-cli change rename ConfigSet"
    )]
    RenameSafety(RenameSafetyArgs),
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli contract-usage-map\n  \
            raccoon-cli inspect contract-usage"
    )]
    ContractUsageMap(ContractUsageMapArgs),
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli briefing internal/domain/configctl/config.go\n  \
            raccoon-cli change briefing"
    )]
    Briefing(BriefingArgs),
    /// Detect semantic drift between a baseline snapshot and the current repository state
    #[command(
        long_about = "Compare the current repository against a previously saved baseline snapshot\n\
            to detect semantic drift — structural changes that may indicate divergence\n\
            from expected architecture, contracts, or invariants.\n\n\
            Drift classes detected:\n  \
              1. Contract surface drift: removed/modified/added contracts\n  \
              2. Interface breaking: removed interface methods\n  \
              3. Interface expansion: added interface methods\n  \
              4. Layer boundary drift: architecture layer changes\n  \
              5. Type breaking: removed fields, type changes\n  \
              6. API signature drift: exported function signature changes\n  \
              7. Coupling increase: new cross-layer imports\n  \
              8. Isolation loss: domain/application importing infrastructure\n  \
              9. Contract proliferation: rapid growth without validation\n  \
              10. Structural scale shift: large-scale code changes\n\n\
            Every finding is tagged with its evidence basis:\n  \
              - observed: directly from the snapshot diff\n  \
              - inferred: derived from combining multiple facts\n  \
              - heuristic: statistical or pattern-based",
        after_help = "Examples:\n  \
            raccoon-cli baseline-drift baseline.json\n  \
            raccoon-cli --json baseline-drift baseline.json\n  \
            raccoon-cli -v baseline-drift baseline.json\n  \
            raccoon-cli --json snapshot -o baseline.json   # save baseline first"
    )]
    BaselineDrift(BaselineDriftArgs),
    /// Generate a golden snapshot of the repository's code intelligence
    #[command(
        long_about = "Generate a deterministic, auditable snapshot of the repository's structural\n\
            and semantic state as observed by the codeintel layer.\n\n\
            The snapshot captures:\n  \
              - Packages, imports, types, functions, constants, interfaces\n  \
              - Architecture layer classification per package\n  \
              - Detected contract types and families\n  \
              - Aggregate statistics\n\n\
            Every fact is tagged with its provenance: ast, lsp, inferred, or runtime.\n\
            Output is sorted and deterministic — same source tree produces the same\n\
            snapshot (modulo metadata.generated_at).\n\n\
            Use for baseline comparison, drift detection, and debugging.",
        after_help = "Examples:\n  \
            raccoon-cli snapshot\n  \
            raccoon-cli --json snapshot\n  \
            raccoon-cli --json snapshot --output snapshot.json\n  \
            raccoon-cli -v snapshot                              # show types, functions, imports\n  \
            diff <(raccoon-cli --json snapshot) baseline.json    # detect drift"
    )]
    Snapshot(SnapshotArgs),
    /// Compare two snapshots and produce a semantic diff report
    #[command(
        long_about = "Compare two code intelligence snapshots and produce a structured diff.\n\n\
            Highlights additions, removals, and modifications across all snapshot sections:\n\
            packages, imports, types, functions, constants, interfaces, arch layers, and contracts.\n\n\
            Changes are reported semantically (field added, signature changed, method removed)\n\
            rather than as raw text diffs. The report separates observed facts from derived\n\
            inferences about impact and risk.\n\n\
            Both snapshots must have the same format version. Corrupted or incompatible\n\
            snapshots are detected and reported clearly.",
        after_help = "Examples:\n  \
            raccoon-cli snapshot-diff before.json after.json\n  \
            raccoon-cli --json snapshot-diff baseline.json current.json\n  \
            raccoon-cli -v snapshot-diff old.json new.json\n  \
            raccoon-cli snapshot-diff before.json --after-live     # compare file vs live project"
    )]
    SnapshotDiff(SnapshotDiffArgs),
    #[command(
        hide = true,
        after_help = "Examples:\n  \
            raccoon-cli recommend\n  \
            raccoon-cli change recommend --baseline snapshot.json"
    )]
    Recommend(RecommendArgs),
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::Parser;

    #[test]
    fn cli_parses_doctor() {
        let cli = Cli::try_parse_from(["raccoon", "doctor"]).unwrap();
        assert!(!cli.json);
        assert!(!cli.verbose);
        assert!(matches!(cli.command, Commands::Doctor));
    }

    #[test]
    fn cli_parses_topology_doctor() {
        let cli = Cli::try_parse_from(["raccoon", "topology-doctor"]).unwrap();
        assert!(matches!(cli.command, Commands::TopologyDoctor));
    }

    #[test]
    fn cli_parses_grouped_check_repo() {
        let cli = Cli::try_parse_from(["raccoon", "check", "repo"]).unwrap();
        assert!(matches!(cli.command, Commands::Check(CheckCommands::Repo)));
    }

    #[test]
    fn cli_parses_grouped_inspect_symbol() {
        let cli = Cli::try_parse_from(["raccoon", "inspect", "symbol", "ConfigSet"]).unwrap();
        assert!(matches!(
            cli.command,
            Commands::Inspect(InspectCommands::Symbol(SymbolTraceArgs { ref symbol, .. }))
                if symbol == "ConfigSet"
        ));
    }

    #[test]
    fn cli_parses_grouped_change_tdd() {
        let cli = Cli::try_parse_from(["raccoon", "change", "tdd"]).unwrap();
        assert!(matches!(
            cli.command,
            Commands::Change(ChangeCommands::Tdd(TddArgs { ref files })) if files.is_empty()
        ));
    }

    #[test]
    fn cli_parses_grouped_legacy_runtime_smoke() {
        let cli = Cli::try_parse_from(["raccoon", "legacy", "runtime-smoke"]).unwrap();
        assert!(matches!(
            cli.command,
            Commands::Legacy(LegacyCommands::RuntimeSmoke(_))
        ));
    }

    #[test]
    fn cli_parses_json_flag() {
        let cli = Cli::try_parse_from(["raccoon", "--json", "doctor"]).unwrap();
        assert!(cli.json);
    }

    #[test]
    fn cli_parses_verbose_long() {
        let cli = Cli::try_parse_from(["raccoon", "--verbose", "doctor"]).unwrap();
        assert!(cli.verbose);
    }

    #[test]
    fn cli_parses_verbose_short() {
        let cli = Cli::try_parse_from(["raccoon", "-v", "doctor"]).unwrap();
        assert!(cli.verbose);
    }

    #[test]
    fn cli_parses_lsp_enrich() {
        let cli = Cli::try_parse_from(["raccoon", "lsp-enrich", "ConfigSet"]).unwrap();
        assert!(matches!(
            cli.command,
            Commands::LspEnrich(LspEnrichArgs { ref symbol, no_lsp, .. }) if symbol == "ConfigSet" && !no_lsp
        ));
    }

    #[test]
    fn cli_parses_lsp_enrich_no_lsp() {
        let cli = Cli::try_parse_from(["raccoon", "lsp-enrich", "--no-lsp", "Foo"]).unwrap();
        assert!(matches!(
            cli.command,
            Commands::LspEnrich(LspEnrichArgs { no_lsp: true, .. })
        ));
    }

    #[test]
    fn cli_parses_lsp_enrich_json() {
        let cli = Cli::try_parse_from(["raccoon", "--json", "lsp-enrich", "Bar"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::LspEnrich(_)));
    }

    #[test]
    fn cli_parses_project_root() {
        let cli = Cli::try_parse_from(["raccoon", "--project-root", "/tmp", "doctor"]).unwrap();
        assert_eq!(cli.project_root, std::path::PathBuf::from("/tmp"));
    }

    #[test]
    fn cli_parses_contract_audit() {
        let cli = Cli::try_parse_from(["raccoon", "contract-audit"]).unwrap();
        assert!(matches!(cli.command, Commands::ContractAudit));
    }

    #[test]
    fn cli_parses_runtime_bindings() {
        let cli = Cli::try_parse_from(["raccoon", "runtime-bindings"]).unwrap();
        assert!(matches!(cli.command, Commands::RuntimeBindings));
    }

    #[test]
    fn cli_parses_runtime_bindings_json() {
        let cli = Cli::try_parse_from(["raccoon", "--json", "runtime-bindings"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::RuntimeBindings));
    }

    #[test]
    fn cli_parses_runtime_smoke() {
        let cli = Cli::try_parse_from(["raccoon", "runtime-smoke"]).unwrap();
        assert!(matches!(cli.command, Commands::RuntimeSmoke(_)));
    }

    #[test]
    fn cli_parses_runtime_smoke_with_base_url() {
        let cli = Cli::try_parse_from([
            "raccoon",
            "runtime-smoke",
            "--base-url",
            "http://localhost:9090",
        ])
        .unwrap();
        match cli.command {
            Commands::RuntimeSmoke(RuntimeSmokeArgs { ref base_url }) => {
                assert_eq!(base_url, "http://localhost:9090");
            }
            _ => panic!("expected RuntimeSmoke"),
        }
    }

    #[test]
    fn cli_parses_runtime_smoke_json() {
        let cli = Cli::try_parse_from(["raccoon", "--json", "runtime-smoke"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::RuntimeSmoke(_)));
    }

    #[test]
    fn cli_parses_quality_gate_default() {
        let cli = Cli::try_parse_from(["raccoon-cli", "quality-gate"]).unwrap();
        match cli.command {
            Commands::QualityGate(QualityGateArgs { profile, .. }) => {
                assert_eq!(profile, GateProfile::Fast);
            }
            _ => panic!("expected QualityGate"),
        }
    }

    #[test]
    fn cli_parses_quality_gate_fast() {
        let cli =
            Cli::try_parse_from(["raccoon-cli", "quality-gate", "--profile", "fast"]).unwrap();
        match cli.command {
            Commands::QualityGate(QualityGateArgs { profile, .. }) => {
                assert_eq!(profile, GateProfile::Fast)
            }
            _ => panic!("expected QualityGate"),
        }
    }

    #[test]
    fn cli_parses_quality_gate_ci() {
        let cli = Cli::try_parse_from(["raccoon-cli", "quality-gate", "--profile", "ci"]).unwrap();
        match cli.command {
            Commands::QualityGate(QualityGateArgs { profile, .. }) => {
                assert_eq!(profile, GateProfile::Ci)
            }
            _ => panic!("expected QualityGate"),
        }
    }

    #[test]
    fn cli_parses_quality_gate_deep() {
        let cli =
            Cli::try_parse_from(["raccoon-cli", "quality-gate", "--profile", "deep"]).unwrap();
        match cli.command {
            Commands::QualityGate(QualityGateArgs { profile, .. }) => {
                assert_eq!(profile, GateProfile::Deep)
            }
            _ => panic!("expected QualityGate"),
        }
    }

    #[test]
    fn cli_rejects_quality_gate_invalid_profile() {
        assert!(
            Cli::try_parse_from(["raccoon-cli", "quality-gate", "--profile", "turbo"]).is_err()
        );
    }

    #[test]
    fn cli_parses_quality_gate_json() {
        let cli =
            Cli::try_parse_from(["raccoon-cli", "--json", "quality-gate", "--profile", "fast"])
                .unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::QualityGate(_)));
    }

    #[test]
    fn cli_parses_drift_detect() {
        let cli = Cli::try_parse_from(["raccoon", "drift-detect"]).unwrap();
        assert!(matches!(cli.command, Commands::DriftDetect));
    }

    #[test]
    fn cli_parses_drift_detect_json() {
        let cli = Cli::try_parse_from(["raccoon", "--json", "drift-detect"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::DriftDetect));
    }

    #[test]
    fn cli_parses_arch_guard() {
        let cli = Cli::try_parse_from(["raccoon", "arch-guard"]).unwrap();
        assert!(matches!(cli.command, Commands::ArchGuard));
    }

    #[test]
    fn cli_parses_arch_guard_json() {
        let cli = Cli::try_parse_from(["raccoon", "--json", "arch-guard"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::ArchGuard));
    }

    #[test]
    fn cli_parses_symbol_trace() {
        let cli = Cli::try_parse_from(["raccoon", "symbol-trace", "ConfigSet"]).unwrap();
        match cli.command {
            Commands::SymbolTrace(SymbolTraceArgs {
                ref symbol,
                lsp,
                no_lsp,
            }) => {
                assert_eq!(symbol, "ConfigSet");
                assert!(!lsp);
                assert!(!no_lsp);
            }
            _ => panic!("expected SymbolTrace"),
        }
    }

    #[test]
    fn cli_parses_symbol_trace_with_lsp() {
        let cli = Cli::try_parse_from(["raccoon", "symbol-trace", "--lsp", "ConfigSet"]).unwrap();
        match cli.command {
            Commands::SymbolTrace(SymbolTraceArgs {
                ref symbol,
                lsp,
                no_lsp,
            }) => {
                assert_eq!(symbol, "ConfigSet");
                assert!(lsp);
                assert!(!no_lsp);
            }
            _ => panic!("expected SymbolTrace"),
        }
    }

    #[test]
    fn cli_parses_symbol_trace_json() {
        let cli = Cli::try_parse_from(["raccoon", "--json", "symbol-trace", "Foo"]).unwrap();
        assert!(cli.json);
        match cli.command {
            Commands::SymbolTrace(SymbolTraceArgs { ref symbol, .. }) => assert_eq!(symbol, "Foo"),
            _ => panic!("expected SymbolTrace"),
        }
    }

    #[test]
    fn cli_symbol_trace_requires_symbol() {
        assert!(Cli::try_parse_from(["raccoon", "symbol-trace"]).is_err());
    }

    #[test]
    fn cli_rejects_unknown_command() {
        assert!(Cli::try_parse_from(["raccoon", "foobar"]).is_err());
    }

    #[test]
    fn cli_project_root_defaults_to_current_dir() {
        let cli = Cli::try_parse_from(["raccoon-cli", "doctor"]).unwrap();
        assert_eq!(cli.project_root, std::path::PathBuf::from("."));
    }

    #[test]
    fn cli_json_flag_before_subcommand() {
        let cli = Cli::try_parse_from(["raccoon-cli", "--json", "topology-doctor"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::TopologyDoctor));
    }

    #[test]
    fn cli_quality_gate_base_url_with_deep() {
        let cli = Cli::try_parse_from([
            "raccoon-cli",
            "quality-gate",
            "--profile",
            "deep",
            "--base-url",
            "http://localhost:9090",
        ])
        .unwrap();
        match cli.command {
            Commands::QualityGate(QualityGateArgs {
                profile,
                base_url,
                fail_fast,
            }) => {
                assert_eq!(profile, GateProfile::Deep);
                assert_eq!(base_url, "http://localhost:9090");
                assert!(!fail_fast);
            }
            _ => panic!("expected QualityGate"),
        }
    }

    #[test]
    fn cli_parses_quality_gate_fail_fast() {
        let cli = Cli::try_parse_from(["raccoon-cli", "quality-gate", "--fail-fast"]).unwrap();
        match cli.command {
            Commands::QualityGate(QualityGateArgs { fail_fast, .. }) => assert!(fail_fast),
            _ => panic!("expected QualityGate"),
        }
    }

    #[test]
    fn cli_quality_gate_fail_fast_default_is_false() {
        let cli = Cli::try_parse_from(["raccoon-cli", "quality-gate"]).unwrap();
        match cli.command {
            Commands::QualityGate(QualityGateArgs { fail_fast, .. }) => assert!(!fail_fast),
            _ => panic!("expected QualityGate"),
        }
    }

    #[test]
    fn json_and_verbose_are_independent() {
        let cli = Cli::try_parse_from(["raccoon-cli", "--json", "-v", "doctor"]).unwrap();
        assert!(cli.json);
        assert!(cli.verbose);
    }

    #[test]
    fn verbose_flag_global_with_quality_gate() {
        let cli = Cli::try_parse_from(["raccoon-cli", "-v", "quality-gate"]).unwrap();
        assert!(cli.verbose);
        assert!(matches!(cli.command, Commands::QualityGate(_)));
    }

    #[test]
    fn cli_parses_snapshot() {
        let cli = Cli::try_parse_from(["raccoon-cli", "snapshot"]).unwrap();
        assert!(matches!(
            cli.command,
            Commands::Snapshot(SnapshotArgs { output: None })
        ));
    }

    #[test]
    fn cli_parses_snapshot_json() {
        let cli = Cli::try_parse_from(["raccoon-cli", "--json", "snapshot"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::Snapshot(_)));
    }

    #[test]
    fn cli_parses_snapshot_with_output() {
        let cli =
            Cli::try_parse_from(["raccoon-cli", "snapshot", "--output", "snap.json"]).unwrap();
        match cli.command {
            Commands::Snapshot(SnapshotArgs { output }) => {
                assert_eq!(output, Some(std::path::PathBuf::from("snap.json")));
            }
            _ => panic!("expected Snapshot"),
        }
    }

    #[test]
    fn cli_parses_snapshot_with_short_output() {
        let cli = Cli::try_parse_from(["raccoon-cli", "snapshot", "-o", "out.json"]).unwrap();
        match cli.command {
            Commands::Snapshot(SnapshotArgs { output }) => {
                assert_eq!(output, Some(std::path::PathBuf::from("out.json")));
            }
            _ => panic!("expected Snapshot"),
        }
    }

    #[test]
    fn cli_parses_rename_safety() {
        let cli = Cli::try_parse_from(["raccoon-cli", "rename-safety", "ConfigSet"]).unwrap();
        match cli.command {
            Commands::RenameSafety(RenameSafetyArgs {
                ref symbol,
                ref new_name,
                lsp,
                no_lsp,
            }) => {
                assert_eq!(symbol, "ConfigSet");
                assert!(new_name.is_none());
                assert!(!lsp);
                assert!(!no_lsp);
            }
            _ => panic!("expected RenameSafety"),
        }
    }

    #[test]
    fn cli_parses_rename_safety_with_new_name() {
        let cli = Cli::try_parse_from([
            "raccoon-cli",
            "rename-safety",
            "ConfigSet",
            "--to",
            "QualityConfigSet",
        ])
        .unwrap();
        match cli.command {
            Commands::RenameSafety(RenameSafetyArgs {
                ref symbol,
                ref new_name,
                ..
            }) => {
                assert_eq!(symbol, "ConfigSet");
                assert_eq!(new_name.as_deref(), Some("QualityConfigSet"));
            }
            _ => panic!("expected RenameSafety"),
        }
    }

    #[test]
    fn cli_parses_rename_safety_with_lsp() {
        let cli =
            Cli::try_parse_from(["raccoon-cli", "rename-safety", "--lsp", "ConfigSet"]).unwrap();
        match cli.command {
            Commands::RenameSafety(RenameSafetyArgs { lsp, no_lsp, .. }) => {
                assert!(lsp);
                assert!(!no_lsp);
            }
            _ => panic!("expected RenameSafety"),
        }
    }

    #[test]
    fn cli_parses_rename_safety_json() {
        let cli = Cli::try_parse_from(["raccoon-cli", "--json", "rename-safety", "Foo"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::RenameSafety(_)));
    }

    #[test]
    fn cli_rename_safety_requires_symbol() {
        assert!(Cli::try_parse_from(["raccoon-cli", "rename-safety"]).is_err());
    }

    #[test]
    fn cli_parses_contract_usage_map() {
        let cli = Cli::try_parse_from(["raccoon-cli", "contract-usage-map"]).unwrap();
        assert!(matches!(
            cli.command,
            Commands::ContractUsageMap(ContractUsageMapArgs {
                lsp: false,
                no_lsp: false
            })
        ));
    }

    #[test]
    fn cli_parses_contract_usage_map_with_lsp() {
        let cli = Cli::try_parse_from(["raccoon-cli", "contract-usage-map", "--lsp"]).unwrap();
        match cli.command {
            Commands::ContractUsageMap(ContractUsageMapArgs { lsp, no_lsp }) => {
                assert!(lsp);
                assert!(!no_lsp);
            }
            _ => panic!("expected ContractUsageMap"),
        }
    }

    #[test]
    fn cli_parses_contract_usage_map_json() {
        let cli = Cli::try_parse_from(["raccoon-cli", "--json", "contract-usage-map"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::ContractUsageMap(_)));
    }

    #[test]
    fn cli_parses_contract_usage_map_verbose() {
        let cli = Cli::try_parse_from(["raccoon-cli", "-v", "contract-usage-map"]).unwrap();
        assert!(cli.verbose);
        assert!(matches!(cli.command, Commands::ContractUsageMap(_)));
    }

    #[test]
    fn cli_parses_coverage_map() {
        let cli = Cli::try_parse_from(["raccoon-cli", "coverage-map"]).unwrap();
        assert!(matches!(cli.command, Commands::CoverageMap));
    }

    #[test]
    fn cli_parses_coverage_map_json() {
        let cli = Cli::try_parse_from(["raccoon-cli", "--json", "coverage-map"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::CoverageMap));
    }

    #[test]
    fn cli_parses_coverage_map_verbose() {
        let cli = Cli::try_parse_from(["raccoon-cli", "-v", "coverage-map"]).unwrap();
        assert!(cli.verbose);
        assert!(matches!(cli.command, Commands::CoverageMap));
    }

    #[test]
    fn cli_parses_tdd_no_files() {
        let cli = Cli::try_parse_from(["raccoon-cli", "tdd"]).unwrap();
        match cli.command {
            Commands::Tdd(TddArgs { files }) => assert!(files.is_empty()),
            _ => panic!("expected Tdd"),
        }
    }

    #[test]
    fn cli_parses_tdd_with_files() {
        let cli = Cli::try_parse_from([
            "raccoon-cli",
            "tdd",
            "internal/adapters/nats/codec.go",
            "deploy/configs/consumer.jsonc",
        ])
        .unwrap();
        match cli.command {
            Commands::Tdd(TddArgs { files }) => {
                assert_eq!(files.len(), 2);
                assert_eq!(files[0], "internal/adapters/nats/codec.go");
                assert_eq!(files[1], "deploy/configs/consumer.jsonc");
            }
            _ => panic!("expected Tdd"),
        }
    }

    #[test]
    fn cli_parses_tdd_json() {
        let cli = Cli::try_parse_from(["raccoon-cli", "--json", "tdd", "file.go"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::Tdd(_)));
    }

    #[test]
    fn cli_parses_impact_map_no_targets() {
        let cli = Cli::try_parse_from(["raccoon-cli", "impact-map"]).unwrap();
        match cli.command {
            Commands::ImpactMap(ImpactMapArgs { targets, .. }) => assert!(targets.is_empty()),
            _ => panic!("expected ImpactMap"),
        }
    }

    #[test]
    fn cli_parses_impact_map_with_targets() {
        let cli = Cli::try_parse_from([
            "raccoon-cli",
            "impact-map",
            "internal/domain/configctl/config.go",
            "ConfigSet",
        ])
        .unwrap();
        match cli.command {
            Commands::ImpactMap(ImpactMapArgs { targets, .. }) => {
                assert_eq!(targets.len(), 2);
                assert_eq!(targets[0], "internal/domain/configctl/config.go");
                assert_eq!(targets[1], "ConfigSet");
            }
            _ => panic!("expected ImpactMap"),
        }
    }

    #[test]
    fn cli_parses_impact_map_json() {
        let cli = Cli::try_parse_from(["raccoon-cli", "--json", "impact-map", "file.go"]).unwrap();
        assert!(cli.json);
        assert!(matches!(cli.command, Commands::ImpactMap(_)));
    }

    #[test]
    fn cli_parses_impact_map_with_lsp() {
        let cli = Cli::try_parse_from(["raccoon-cli", "impact-map", "--lsp", "ConfigSet"]).unwrap();
        match cli.command {
            Commands::ImpactMap(ImpactMapArgs {
                lsp,
                no_lsp,
                targets,
            }) => {
                assert!(lsp);
                assert!(!no_lsp);
                assert_eq!(targets, vec!["ConfigSet"]);
            }
            _ => panic!("expected ImpactMap"),
        }
    }

    #[test]
    fn cli_rejects_conflicting_lsp_flags() {
        assert!(Cli::try_parse_from([
            "raccoon-cli",
            "impact-map",
            "--lsp",
            "--no-lsp",
            "ConfigSet"
        ])
        .is_err());
    }
}
