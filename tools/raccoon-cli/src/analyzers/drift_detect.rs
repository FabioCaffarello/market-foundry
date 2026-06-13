use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};
use std::collections::{HashMap, HashSet};
use std::path::Path;

use super::topology::{self, ComposeTopology, ServiceConfig, SourceTopology};

// ── Constants ───────────────────────────────────────────────────────

/// The five application binaries in market-foundry.
const APP_BINARIES: &[&str] = &[
    "configctl",
    "gateway",
    "ingest",
    "derive",
    "store",
    "execute",
];

/// Infrastructure services (no config file expected).
const INFRA_SERVICES: &[&str] = &["nats"];

/// Canonical JetStream stream names.
const CANONICAL_STREAMS: &[&str] = &[
    "CONFIGCTL_EVENTS",
    "OBSERVATION_EVENTS",
    "EVIDENCE_EVENTS",
    "SIGNAL_EVENTS",
    "DECISION_EVENTS",
    "STRATEGY_EVENTS",
    "RISK_EVENTS",
    "EXECUTION_EVENTS",
    "EXECUTION_FILL_EVENTS",
    "EXECUTION_REJECTION_EVENTS",
    "SESSION_LIFECYCLE_EVENTS",
    "INSIGHTS_EVENTS", // PROGRAM-0005 / H-8.a — insights decision-support stream
];

/// Old service names that should no longer appear in active code.
const DEFUNCT_NAMES: &[&str] = &["emulator", "validator"];

/// Canonical root-level docs that must exist (Phase 1A topology).
/// Pre-reset docs/architecture/* design docs were retired in P1A.1
/// (commit 407c723); active surface is the consolidated root files.
const ARCH_DOCS: &[&str] = &[
    "docs/ARCHITECTURE.md",
    "docs/RUNTIME.md",
    "docs/HTTP-API.md",
    "docs/DEVELOPMENT.md",
    "docs/RESUMPTION.md",
    "docs/CONTRIBUTING.md",
    "docs/GLOSSARY.md",
    "docs/decisions/README.md",
];

/// Stream names that must NOT appear in source yet (premature domain entry guard).
const PROHIBITED_STREAMS: &[(&str, &str)] = &[(
    "PROJECTION_EVENTS",
    "projection notification family is planned but not yet approved for implementation",
)];

/// Active signal-domain doc (Phase 1A topology). Per-family design docs
/// were retired in P1A.1 and consolidated into one domain doc per
/// Phase 1A.6.
const SIGNAL_DOCS: &[&str] = &["docs/domain/signal.md"];

/// Expected signal NATS subjects that must exist in source.
const SIGNAL_EXPECTED_SUBJECTS: &[(&str, &str)] = &[
    (
        "signal.events.rsi.generated",
        "RSI signal event subject — derive publishes finalized RSI signals",
    ),
    (
        "signal.query.rsi.latest",
        "RSI latest query subject — gateway queries store for latest RSI",
    ),
];

/// Expected signal durable consumers.
const SIGNAL_EXPECTED_DURABLES: &[(&str, &str)] = &[(
    "store-signal-rsi",
    "store consumes RSI signal events from SIGNAL_EVENTS for projection",
)];

/// Expected signal KV bucket names that must appear in source.
const SIGNAL_EXPECTED_BUCKETS: &[(&str, &str)] = &[(
    "SIGNAL_RSI_LATEST",
    "stores latest finalized RSI signal per partition key",
)];

/// Expected signal adapter files in internal/adapters/nats/.
const SIGNAL_ADAPTER_FILES: &[(&str, &str)] = &[
    (
        "natssignal/registry.go",
        "defines SIGNAL_EVENTS stream, consumer, and query specs",
    ),
    (
        "natssignal/publisher.go",
        "publishes signal events to SIGNAL_EVENTS stream",
    ),
    (
        "natssignal/consumer.go",
        "durable consumer for signal events in store",
    ),
    (
        "natssignal/gateway.go",
        "gateway adapter for signal NATS request/reply queries",
    ),
    (
        "natssignal/kv_store.go",
        "KV bucket store for latest signal projections",
    ),
];

/// Expected signal domain/application files.
const SIGNAL_DOMAIN_FILES: &[(&str, &str)] = &[
    (
        "internal/domain/signal/signal.go",
        "signal domain entity with Validate(), PartitionKey(), DeduplicationKey()",
    ),
    (
        "internal/domain/signal/events.go",
        "SignalGeneratedEvent definition",
    ),
    (
        "internal/application/signal/rsi_sampler.go",
        "RSI sampler (Wilder's smoothed moving average)",
    ),
    (
        "internal/application/signalclient/contracts.go",
        "signal query/reply contracts",
    ),
    (
        "internal/application/signalclient/get_latest_signal.go",
        "GetLatestSignal use case",
    ),
    (
        "internal/application/ports/signal.go",
        "SignalGateway port interface",
    ),
];

// ── Decision domain governance constants ─────────────────────────────

/// Active decision-domain doc (Phase 1A topology).
const DECISION_DOCS: &[&str] = &["docs/domain/decision.md"];

/// Expected decision NATS subjects that must exist in source.
const DECISION_EXPECTED_SUBJECTS: &[(&str, &str)] = &[
    (
        "decision.events.rsi_oversold.evaluated",
        "decision event subject — derive publishes finalized RSI oversold decisions",
    ),
    (
        "decision.query.rsi_oversold.latest",
        "decision latest query subject — gateway queries store for latest RSI oversold decision",
    ),
];

/// Expected decision durable consumers.
const DECISION_EXPECTED_DURABLES: &[(&str, &str)] = &[(
    "store-decision-rsi-oversold",
    "store consumes RSI oversold decision events from DECISION_EVENTS for projection",
)];

/// Expected decision KV bucket names that must appear in source.
const DECISION_EXPECTED_BUCKETS: &[(&str, &str)] = &[(
    "DECISION_RSI_OVERSOLD_LATEST",
    "stores latest finalized RSI oversold decision per partition key",
)];

/// Expected decision adapter files in internal/adapters/nats/.
const DECISION_ADAPTER_FILES: &[(&str, &str)] = &[
    (
        "natsdecision/registry.go",
        "defines DECISION_EVENTS stream, consumer, and query specs",
    ),
    (
        "natsdecision/publisher.go",
        "publishes decision events to DECISION_EVENTS stream",
    ),
    (
        "natsdecision/consumer.go",
        "durable consumer for decision events in store",
    ),
    (
        "natsdecision/gateway.go",
        "gateway adapter for decision NATS request/reply queries",
    ),
    (
        "natsdecision/kv_store.go",
        "KV bucket store for latest decision projections",
    ),
];

/// Expected decision domain/application files.
const DECISION_DOMAIN_FILES: &[(&str, &str)] = &[
    (
        "internal/domain/decision/decision.go",
        "decision domain entity with Validate(), PartitionKey(), DeduplicationKey()",
    ),
    (
        "internal/domain/decision/events.go",
        "DecisionEvaluatedEvent definition",
    ),
    (
        "internal/application/decision/rsi_oversold_evaluator.go",
        "RSI oversold evaluator (threshold-based)",
    ),
    (
        "internal/application/decisionclient/contracts.go",
        "decision query/reply contracts",
    ),
    (
        "internal/application/decisionclient/get_latest_decision.go",
        "GetLatestDecision use case",
    ),
    (
        "internal/application/ports/decision.go",
        "DecisionGateway port interface",
    ),
];

// ── Strategy domain governance constants ─────────────────────────────

/// Active strategy-domain doc (Phase 1A topology).
const STRATEGY_DOCS: &[&str] = &["docs/domain/strategy.md"];

/// Expected strategy NATS subjects that must exist in source.
const STRATEGY_EXPECTED_SUBJECTS: &[(&str, &str)] = &[
    (
        "strategy.events.mean_reversion_entry.resolved",
        "strategy event subject — derive publishes finalized mean reversion entry strategies",
    ),
    (
        "strategy.query.mean_reversion_entry.latest",
        "strategy latest query subject — gateway queries store for latest mean reversion entry",
    ),
];

/// Expected strategy durable consumers.
const STRATEGY_EXPECTED_DURABLES: &[(&str, &str)] = &[(
    "store-strategy-mean-reversion-entry",
    "store consumes mean reversion entry strategy events from STRATEGY_EVENTS for projection",
)];

/// Expected strategy KV bucket names that must appear in source.
const STRATEGY_EXPECTED_BUCKETS: &[(&str, &str)] = &[(
    "STRATEGY_MEAN_REVERSION_ENTRY_LATEST",
    "stores latest finalized mean reversion entry strategy per partition key",
)];

/// Expected strategy adapter files in internal/adapters/nats/.
const STRATEGY_ADAPTER_FILES: &[(&str, &str)] = &[
    (
        "natsstrategy/registry.go",
        "defines STRATEGY_EVENTS stream, consumer, and query specs",
    ),
    (
        "natsstrategy/publisher.go",
        "publishes strategy events to STRATEGY_EVENTS stream",
    ),
    (
        "natsstrategy/consumer.go",
        "durable consumer for strategy events in store",
    ),
    (
        "natsstrategy/gateway.go",
        "gateway adapter for strategy NATS request/reply queries",
    ),
    (
        "natsstrategy/kv_store.go",
        "KV bucket store for latest strategy projections",
    ),
];

/// Expected strategy domain/application files.
const STRATEGY_DOMAIN_FILES: &[(&str, &str)] = &[
    (
        "internal/domain/strategy/strategy.go",
        "strategy domain entity with Validate(), PartitionKey(), DeduplicationKey()",
    ),
    (
        "internal/domain/strategy/events.go",
        "StrategyResolvedEvent definition",
    ),
    (
        "internal/application/strategy/mean_reversion_entry_resolver.go",
        "mean reversion entry resolver (decision-to-strategy)",
    ),
    (
        "internal/application/strategyclient/contracts.go",
        "strategy query/reply contracts",
    ),
    (
        "internal/application/strategyclient/get_latest_strategy.go",
        "GetLatestStrategy use case",
    ),
    (
        "internal/application/ports/strategy.go",
        "StrategyGateway port interface",
    ),
];

// ── Risk domain governance constants ─────────────────────────────────
// Risk governance is active from S63. Implementation begins in S64.
// These constants define the expected artifacts once risk is implemented.
// Until S64 opens, RISK_EVENTS is in PROHIBITED_STREAMS.

/// Active risk-domain doc (Phase 1A topology).
const RISK_DOCS: &[&str] = &["docs/domain/risk.md"];

/// Expected risk NATS subjects (activate after S64 opens implementation).
const RISK_EXPECTED_SUBJECTS: &[(&str, &str)] = &[
    (
        "risk.events.position_exposure.assessed",
        "risk event subject — derive publishes finalized position exposure assessments",
    ),
    (
        "risk.query.position_exposure.latest",
        "risk latest query subject — gateway queries store for latest position exposure",
    ),
];

/// Expected risk durable consumers (activate after S64 opens implementation).
const RISK_EXPECTED_DURABLES: &[(&str, &str)] = &[(
    "store-risk-position-exposure",
    "store consumes position exposure risk events from RISK_EVENTS for projection",
)];

/// Expected risk KV bucket names (activate after S64 opens implementation).
const RISK_EXPECTED_BUCKETS: &[(&str, &str)] = &[(
    "RISK_POSITION_EXPOSURE_LATEST",
    "stores latest finalized position exposure risk assessment per partition key",
)];

/// Expected risk adapter files in internal/adapters/nats/ (activate after S64).
const RISK_ADAPTER_FILES: &[(&str, &str)] = &[
    (
        "natsrisk/registry.go",
        "defines RISK_EVENTS stream, consumer, and query specs",
    ),
    (
        "natsrisk/publisher.go",
        "publishes risk events to RISK_EVENTS stream",
    ),
    (
        "natsrisk/consumer.go",
        "durable consumer for risk events in store",
    ),
    (
        "natsrisk/gateway.go",
        "gateway adapter for risk NATS request/reply queries",
    ),
    (
        "natsrisk/kv_store.go",
        "KV bucket store for latest risk projections",
    ),
];

/// Expected risk domain/application files (activate after S64 opens implementation).
const RISK_DOMAIN_FILES: &[(&str, &str)] = &[
    (
        "internal/domain/risk/risk.go",
        "risk domain entity with Validate(), PartitionKey(), DeduplicationKey()",
    ),
    (
        "internal/domain/risk/events.go",
        "RiskAssessedEvent definition",
    ),
    (
        "internal/application/risk/position_exposure_evaluator.go",
        "position exposure evaluator (strategy-to-risk)",
    ),
    (
        "internal/application/riskclient/contracts.go",
        "risk query/reply contracts",
    ),
    (
        "internal/application/riskclient/get_latest_risk.go",
        "GetLatestRisk use case",
    ),
    (
        "internal/application/ports/risk.go",
        "RiskGateway port interface",
    ),
];

// ── Execution domain governance constants (S70→S83) ──────────────────
// Execution governance active from S70. Implementation completed S71→S82.
// Execute binary governance hardened in S83.

/// Active execution-domain doc (Phase 1A topology).
const EXECUTION_DOCS: &[&str] = &["docs/domain/execution.md"];

/// Expected execution NATS subjects — all active post-S80.
const EXECUTION_EXPECTED_SUBJECTS: &[(&str, &str)] = &[
    (
        "execution.events.paper_order.submitted",
        "execution event subject — derive publishes finalized paper order intents",
    ),
    (
        "execution.query.paper_order.latest",
        "execution latest query subject — gateway queries store for latest paper order intent",
    ),
    (
        "execution.fill.venue_market_order",
        "fill event subject — execute publishes venue order fill confirmations",
    ),
    (
        "execution.query.status.latest",
        "execution status composite query — gateway queries for combined status",
    ),
    (
        "execution.control.get",
        "execution control gate read — gateway/execute reads kill switch state",
    ),
    (
        "execution.control.set",
        "execution control gate write — gateway sets kill switch state",
    ),
];

/// Expected execution durable consumers — all active post-S80.
const EXECUTION_EXPECTED_DURABLES: &[(&str, &str)] = &[
    ("store-execution-paper-order", "store consumes paper order execution events from EXECUTION_EVENTS for projection"),
    ("execute-venue-market-order-intake", "execute consumes paper order intents from EXECUTION_EVENTS for venue submission (transitional bridge — paper mode)"),
    ("store-execution-venue-market-order-fill", "store consumes fill events from EXECUTION_FILL_EVENTS for projection"),
];

/// Expected execution KV bucket names — all active post-S80.
const EXECUTION_EXPECTED_BUCKETS: &[(&str, &str)] = &[
    (
        "EXECUTION_PAPER_ORDER_LATEST",
        "stores latest finalized paper order execution intent per partition key",
    ),
    (
        "EXECUTION_VENUE_MARKET_ORDER_LATEST",
        "stores latest venue market order fill per partition key",
    ),
    (
        "EXECUTION_CONTROL",
        "stores global execution control gate (kill switch)",
    ),
];

/// Expected execution adapter files in internal/adapters/nats/ — all active post-S80.
const EXECUTION_ADAPTER_FILES: &[(&str, &str)] = &[
    (
        "natsexecution/registry.go",
        "defines EXECUTION_EVENTS and EXECUTION_FILL_EVENTS streams, consumers, and query specs",
    ),
    (
        "natsexecution/publisher.go",
        "publishes execution events and fill events to JetStream",
    ),
    (
        "natsexecution/consumer.go",
        "durable consumer for execution events in store/execute",
    ),
    (
        "natsexecution/gateway.go",
        "gateway adapter for execution NATS request/reply queries",
    ),
    (
        "natsexecution/kv_store.go",
        "KV bucket store for latest execution projections",
    ),
    (
        "natsexecution/control_gateway.go",
        "gateway adapter for execution control gate NATS request/reply",
    ),
    (
        "natsexecution/control_kv_store.go",
        "KV bucket store for execution control gate (kill switch)",
    ),
];

/// Expected execution domain/application files — all active post-S80.
const EXECUTION_DOMAIN_FILES: &[(&str, &str)] = &[
    ("internal/domain/execution/execution.go", "execution domain entity (ExecutionIntent) with Validate(), PartitionKey(), DeduplicationKey()"),
    ("internal/domain/execution/events.go", "PaperOrderSubmittedEvent and VenueOrderFilledEvent definitions"),
    ("internal/domain/execution/control.go", "ControlGate domain entity for execution kill switch"),
    ("internal/application/execution/paper_order_evaluator.go", "paper order evaluator (risk-to-execution)"),
    ("internal/application/execution/paper_venue_adapter.go", "paper venue adapter (simulated order execution)"),
    ("internal/application/execution/staleness_guard.go", "staleness guard for execution intent temporal validation"),
    ("internal/application/executionclient/contracts.go", "execution query/reply contracts"),
    ("internal/application/executionclient/control_contracts.go", "execution control gate query/reply contracts"),
    ("internal/application/executionclient/get_latest_execution.go", "GetLatestExecution use case"),
    ("internal/application/executionclient/get_execution_status.go", "GetExecutionStatus use case"),
    ("internal/application/executionclient/get_execution_control.go", "GetExecutionControl use case"),
    ("internal/application/ports/execution.go", "ExecutionGateway and ExecutionControlGateway port interfaces"),
    ("internal/application/ports/venue.go", "VenuePort interface for venue adapter boundary"),
];

/// Expected insights durable consumers (H-8.a.1). Insights persistence is
/// a single-writer split (ADR-0008): the writer owns the ClickHouse table,
/// the store owns the KV-latest bucket — two distinct durables on
/// INSIGHTS_EVENTS.
const INSIGHTS_EXPECTED_DURABLES: &[(&str, &str)] = &[
    (
        "writer-volume-profile",
        "writer persists volume profile events from INSIGHTS_EVENTS into the insights_volume_profile ClickHouse table (codegen-governed)",
    ),
    (
        "store-volume-profile",
        "store projects volume profile events from INSIGHTS_EVENTS into the KV latest bucket",
    ),
    (
        "store-tpo",
        "store projects TPO profile events from INSIGHTS_EVENTS into the KV latest bucket (H-8.b)",
    ),
    (
        "writer-tpo",
        "writer persists TPO profile events from INSIGHTS_EVENTS into the insights_tpo ClickHouse table (codegen-governed, H-8.b.1)",
    ),
    (
        "store-cross-venue",
        "store projects cross-venue snapshots from INSIGHTS_EVENTS into the KV latest bucket (H-8.c)",
    ),
    (
        "writer-cross-venue",
        "writer persists cross-venue snapshots from INSIGHTS_EVENTS into the insights_cross_venue ClickHouse table (codegen-governed, H-8.c.1)",
    ),
    (
        "deliver-insights",
        "delivery (gateway) reads insights events from INSIGHTS_EVENTS and pushes them to subscribed WebSocket clients — read-only transport, ADR-0028/PROGRAM-0006 (H-11.a)",
    ),
];

/// Expected insights ClickHouse history tables that must appear in
/// deploy/migrations (H-8.a.1 volume profile, H-8.b.1 TPO, H-8.c.1
/// cross-venue).
const INSIGHTS_EXPECTED_TABLES: &[(&str, &str)] = &[
    (
        "insights_volume_profile",
        "stores per-window volume profile (VPVR) history with parallel Array(String) bucket columns",
    ),
    (
        "insights_tpo",
        "stores per-window TPO history with parallel Array columns for periods and price levels",
    ),
    (
        "insights_cross_venue",
        "stores per-window cross-venue snapshots with parallel Array columns for the per-venue rows",
    ),
];

// ── Public API ──────────────────────────────────────────────────────

pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("drift-detect");

    // Phase 1: Gather all evidence
    let evidence = gather_evidence(project_root)?;

    // Phase 2: Run drift checks
    report.add(check_config_compose_drift(&evidence));
    report.add(check_binary_compose_drift(&evidence));
    report.add(check_naming_identity_drift(&evidence));
    report.add(check_docs_reality_drift(&evidence));
    report.add(check_actor_scope_drift(&evidence));
    report.add(check_stream_registry_drift(&evidence));
    report.add(check_premature_domain_entry(&evidence));

    // Phase 3: Signal domain governance checks
    report.add(check_signal_docs_drift(&evidence));
    report.add(check_signal_adapter_drift(&evidence));
    report.add(check_signal_domain_drift(&evidence));
    report.add(check_signal_config_drift(&evidence));
    report.add(check_signal_contracts_drift(&evidence));

    // Phase 4: Decision domain governance checks
    report.add(check_decision_docs_drift(&evidence));
    report.add(check_decision_adapter_drift(&evidence));
    report.add(check_decision_domain_drift(&evidence));
    report.add(check_decision_config_drift(&evidence));
    report.add(check_decision_contracts_drift(&evidence));

    // Phase 5: Strategy domain governance checks
    report.add(check_strategy_docs_drift(&evidence));
    report.add(check_strategy_adapter_drift(&evidence));
    report.add(check_strategy_domain_drift(&evidence));
    report.add(check_strategy_config_drift(&evidence));
    report.add(check_strategy_contracts_drift(&evidence));

    // Phase 6: Risk domain governance checks (S64 — active)
    // Risk is now implemented. Full drift checks are active.
    report.add(check_risk_docs_drift(&evidence));
    report.add(check_risk_adapter_drift(&evidence));
    report.add(check_risk_domain_drift(&evidence));
    report.add(check_risk_config_drift(&evidence));
    report.add(check_risk_contracts_drift(&evidence));

    // Phase 7: Execution domain governance checks (S71 — active)
    // Execution is now implemented. Full drift checks are active.
    report.add(check_execution_docs_drift(&evidence));
    report.add(check_execution_adapter_drift(&evidence));
    report.add(check_execution_domain_drift(&evidence));
    report.add(check_execution_config_drift(&evidence));
    report.add(check_execution_contracts_drift(&evidence));

    // Phase 8: Insights domain governance checks (H-8.a.1 — VPVR persistence)
    report.add(check_insights_contracts_drift(&evidence));

    Ok(report)
}

/// Check insights-contracts-drift: verify the insights durable consumers and
/// the ClickHouse history table exist in source (H-8.a.1). The writer durable
/// is codegen-governed (volume_profile family); the store-side durable feeds
/// the KV latest bucket. Single-writer split per ADR-0008.
fn check_insights_contracts_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let source = match &evidence.source {
        Some(s) => s,
        None => return CheckResult::skip("insights-contracts-drift", "source not scanned"),
    };

    // Insights durable consumers (writer + store).
    for (durable, purpose) in INSIGHTS_EXPECTED_DURABLES {
        if source.durables.contains_key(*durable) {
            findings.push(Finding::info(
                "insights-durable-present",
                format!("insights durable consumer found: {durable}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "insights-durable-missing",
                    format!("insights durable consumer not found: {durable}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsinsights/registry.go for consumer spec",
                ),
            );
        }
    }

    // Insights ClickHouse history tables (declared in migrations).
    let migrations_dir = evidence.project_root.join("deploy/migrations");
    for (table, purpose) in INSIGHTS_EXPECTED_TABLES {
        if scan_sql_dir_for_string(&migrations_dir, table) {
            findings.push(Finding::info(
                "insights-table-present",
                format!("insights ClickHouse table found in migrations: {table}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "insights-table-missing",
                    format!("insights ClickHouse table not found in migrations: {table}"),
                )
                .with_why(*purpose)
                .with_help("check deploy/migrations for the CREATE TABLE statement"),
            );
        }
    }

    CheckResult::from_findings("insights-contracts-drift", findings)
}

// ── Evidence gathering ──────────────────────────────────────────────

struct Evidence {
    project_root: std::path::PathBuf,
    configs: HashMap<String, ServiceConfig>,
    compose: Option<ComposeTopology>,
    source: Option<SourceTopology>,
    makefile_targets: HashSet<String>,
    dev_doc_targets: HashSet<String>,
    existing_cmd_dirs: HashSet<String>,
    existing_actor_scopes: HashSet<String>,
    stale_references: Vec<StaleReference>,
}

#[derive(Debug, Clone)]
struct StaleReference {
    file: String,
    line_num: usize,
    pattern: String,
    #[allow(dead_code)]
    context: String,
}

fn gather_evidence(project_root: &Path) -> Result<Evidence> {
    let mut evidence = Evidence {
        project_root: project_root.to_path_buf(),
        configs: HashMap::new(),
        compose: None,
        source: None,
        makefile_targets: HashSet::new(),
        dev_doc_targets: HashSet::new(),
        existing_cmd_dirs: HashSet::new(),
        existing_actor_scopes: HashSet::new(),
        stale_references: Vec::new(),
    };

    // Configs
    let configs_dir = project_root.join("deploy/configs");
    if configs_dir.is_dir() {
        evidence.configs = topology::configs::parse_all_configs(&configs_dir)?;
    }

    // Compose
    let compose_path = project_root.join("deploy/compose/docker-compose.yaml");
    if compose_path.is_file() {
        evidence.compose = topology::compose::parse_compose(&compose_path).ok();
    }

    // Source topology
    let internal_dir = project_root.join("internal");
    if internal_dir.is_dir() {
        evidence.source = topology::source::scan_source(&internal_dir).ok();
    }

    // Binary directories
    let cmd_dir = project_root.join("cmd");
    if cmd_dir.is_dir() {
        if let Ok(entries) = std::fs::read_dir(&cmd_dir) {
            for entry in entries.flatten() {
                if entry.path().is_dir() {
                    if let Some(name) = entry.file_name().to_str() {
                        evidence.existing_cmd_dirs.insert(name.to_string());
                    }
                }
            }
        }
    }

    // Actor scope directories
    let scopes_dir = project_root.join("internal/actors/scopes");
    if scopes_dir.is_dir() {
        if let Ok(entries) = std::fs::read_dir(&scopes_dir) {
            for entry in entries.flatten() {
                if entry.path().is_dir() {
                    if let Some(name) = entry.file_name().to_str() {
                        evidence.existing_actor_scopes.insert(name.to_string());
                    }
                }
            }
        }
    }

    // Stale name references
    scan_stale_references(project_root, &mut evidence.stale_references);

    // Makefile
    let makefile_path = project_root.join("Makefile");
    if makefile_path.is_file() {
        evidence.makefile_targets = extract_makefile_targets(&makefile_path);
    }

    // DEVELOPMENT.md
    let dev_doc_path = project_root.join("DEVELOPMENT.md");
    if dev_doc_path.is_file() {
        evidence.dev_doc_targets = extract_dev_doc_make_targets(&dev_doc_path);
    }

    Ok(evidence)
}

// ── Scanners ────────────────────────────────────────────────────────

fn scan_stale_references(project_root: &Path, refs: &mut Vec<StaleReference>) {
    // Scan Go source and config files for residual old names.
    //
    // "consumer" is now legitimate across cmd/, internal/, and deploy/ as part of the
    // current JetStream/durable-consumer topology. Only truly defunct service identities
    // remain actionable here.

    let universal_patterns: Vec<(&str, &str)> = vec![
        ("emulator", "old service name 'emulator'"),
        ("validator", "old service name 'validator'"),
    ];

    let internal_dir = project_root.join("internal");
    if internal_dir.is_dir() {
        scan_dir_for_patterns(&internal_dir, &universal_patterns, refs, project_root);
    }

    for dir in &[project_root.join("cmd"), project_root.join("deploy")] {
        if dir.is_dir() {
            scan_dir_for_patterns(dir, &universal_patterns, refs, project_root);
        }
    }
}

fn scan_dir_for_patterns(
    dir: &Path,
    patterns: &[(&str, &str)],
    refs: &mut Vec<StaleReference>,
    project_root: &Path,
) {
    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return,
    };
    for entry in entries.flatten() {
        let path = entry.path();
        if path.is_dir() {
            // Skip directories named after old services (they are tracked by other checks)
            let dirname = path.file_name().and_then(|n| n.to_str()).unwrap_or("");
            if DEFUNCT_NAMES.contains(&dirname) {
                continue;
            }
            scan_dir_for_patterns(&path, patterns, refs, project_root);
        } else {
            let ext = path.extension().and_then(|e| e.to_str()).unwrap_or("");
            if ext == "go" || ext == "jsonc" || ext == "yaml" || ext == "yml" {
                scan_file_for_patterns(&path, patterns, refs, project_root);
            }
        }
    }
}

fn scan_file_for_patterns(
    path: &Path,
    patterns: &[(&str, &str)],
    refs: &mut Vec<StaleReference>,
    project_root: &Path,
) {
    let content = match std::fs::read_to_string(path) {
        Ok(c) => c,
        Err(_) => return,
    };

    let relative = path
        .strip_prefix(project_root)
        .unwrap_or(path)
        .display()
        .to_string();

    for (line_num, line) in content.lines().enumerate() {
        let trimmed = line.trim();
        if trimmed.starts_with("//") || trimmed.starts_with("#") {
            continue;
        }
        let lower = trimmed.to_lowercase();
        for (pattern, description) in patterns {
            // Match as whole word boundary: check chars before and after
            let pat_lower = pattern.to_lowercase();
            let mut search = lower.as_str();
            while let Some(pos) = search.find(&pat_lower) {
                let before_ok = pos == 0 || !search.as_bytes()[pos - 1].is_ascii_alphanumeric();
                let after_pos = pos + pat_lower.len();
                let after_ok = after_pos >= search.len()
                    || !search.as_bytes()[after_pos].is_ascii_alphanumeric();

                if before_ok && after_ok {
                    refs.push(StaleReference {
                        file: relative.clone(),
                        line_num: line_num + 1,
                        pattern: description.to_string(),
                        context: trimmed.chars().take(120).collect(),
                    });
                    break; // one finding per line per pattern
                }
                search = &search[pos + pat_lower.len()..];
            }
        }
    }
}

fn extract_makefile_targets(path: &Path) -> HashSet<String> {
    let mut targets = HashSet::new();
    let content = match std::fs::read_to_string(path) {
        Ok(c) => c,
        Err(_) => return targets,
    };

    for line in content.lines() {
        let trimmed = line.trim();
        if trimmed.starts_with('#')
            || trimmed.starts_with('\t')
            || trimmed.starts_with(' ')
            || trimmed.starts_with('@')
            || trimmed.starts_with("define ")
            || trimmed.starts_with("endef")
            || trimmed.starts_with("ifeq")
            || trimmed.starts_with("ifneq")
            || trimmed.starts_with("endif")
            || trimmed.starts_with("else")
            || trimmed.is_empty()
        {
            continue;
        }
        if trimmed.contains("?=")
            || trimmed.contains(":=")
            || trimmed.contains("+=")
            || (trimmed.contains('=') && !trimmed.contains(':'))
        {
            continue;
        }
        if trimmed.starts_with('.') || trimmed.starts_with("$(") {
            continue;
        }
        if let Some(colon_pos) = trimmed.find(':') {
            let target = trimmed[..colon_pos].trim();
            if !target.is_empty()
                && !target.contains(' ')
                && !target.contains('$')
                && !target.contains('/')
            {
                targets.insert(target.to_string());
            }
        }
    }

    targets
}

fn extract_dev_doc_make_targets(path: &Path) -> HashSet<String> {
    let mut targets = HashSet::new();
    let content = match std::fs::read_to_string(path) {
        Ok(c) => c,
        Err(_) => return targets,
    };

    let make_stopwords: HashSet<&str> = [
        "a", "an", "the", "it", "is", "to", "your", "sure", "changes", "sense", "this", "that",
        "any", "no", "not", "use", "certain",
    ]
    .iter()
    .copied()
    .collect();

    for line in content.lines() {
        let mut rest = line;
        while let Some(pos) = rest.find("make ") {
            let after = &rest[pos + 5..];
            let end = after
                .find(|c: char| !c.is_alphanumeric() && c != '-' && c != '_')
                .unwrap_or(after.len());
            let target = &after[..end];
            if !target.is_empty() && !make_stopwords.contains(target) {
                targets.insert(target.to_string());
            }
            rest = &after[end..];
        }
    }

    targets
}

// ── Drift checks ────────────────────────────────────────────────────

/// Check 1: For each service, verify config file exists AND compose service exists.
/// Check NATS URLs are consistent.
fn check_config_compose_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let compose = match &evidence.compose {
        Some(c) => c,
        None => {
            return CheckResult::skip("config-compose-drift", "docker-compose.yaml not available")
        }
    };

    if evidence.configs.is_empty() {
        return CheckResult::skip(
            "config-compose-drift",
            "no configs found in deploy/configs/",
        );
    }

    let compose_services: HashSet<&str> = compose.services.keys().map(|s| s.as_str()).collect();
    let config_services: HashSet<&str> = evidence.configs.keys().map(|s| s.as_str()).collect();
    let app_set: HashSet<&str> = APP_BINARIES.iter().copied().collect();

    // Configs without matching compose service
    for svc in config_services.difference(&compose_services) {
        if app_set.contains(*svc) {
            findings.push(
                Finding::warning(
                    "config-without-compose",
                    format!("config '{svc}' exists but no matching compose service"),
                )
                .with_why("config declares runtime settings for a service that doesn't exist in compose -- the config is dead weight")
                .with_help(format!("add '{svc}' service to deploy/compose/docker-compose.yaml or remove deploy/configs/{svc}.jsonc")),
            );
        }
    }

    // App services in compose without config
    for svc in compose_services.intersection(&app_set) {
        if !config_services.contains(*svc) {
            findings.push(
                Finding::warning(
                    "compose-without-config",
                    format!("compose service '{svc}' has no deploy/configs/{svc}.jsonc"),
                )
                .with_why("service runs with default/hardcoded settings -- explicit config makes behavior visible and auditable")
                .with_help(format!("create deploy/configs/{svc}.jsonc with at minimum the NATS transport settings")),
            );
        }
    }

    // NATS URL consistency: all configs should point to the same NATS URL
    let nats_urls: Vec<(&str, &str)> = evidence
        .configs
        .iter()
        .filter_map(|(name, cfg)| cfg.nats_url.as_deref().map(|url| (name.as_str(), url)))
        .collect();

    if nats_urls.len() > 1 {
        let first_url = nats_urls[0].1;
        for (name, url) in &nats_urls[1..] {
            if *url != first_url {
                findings.push(
                    Finding::error(
                        "nats-url-drift",
                        format!(
                            "service '{name}' uses NATS URL '{url}' but '{}' uses '{first_url}'",
                            nats_urls[0].0
                        ),
                    )
                    .with_why(
                        "inconsistent NATS URLs cause services to connect to different clusters",
                    )
                    .with_help("ensure all configs use the same nats.url value"),
                );
            }
        }
    }

    // NATS dependency: configs declaring nats_url must depend on nats in compose
    for (name, cfg) in &evidence.configs {
        if let Some(svc) = compose.services.get(name.as_str()) {
            if cfg.nats_url.is_some() && !svc.depends_on.contains(&"nats".to_string()) {
                findings.push(
                    Finding::error(
                        "transport-drift",
                        format!("'{name}' config declares nats url but compose service doesn't depend on nats"),
                    )
                    .with_why("service will fail to connect if nats isn't running")
                    .with_help(format!("add 'nats' to depends_on of '{name}' in docker-compose.yaml")),
                );
            }
        }
    }

    CheckResult::from_findings("config-compose-drift", findings)
}

/// Check 2: For each expected binary, verify cmd/{name}/ exists AND compose service exists.
fn check_binary_compose_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let compose_services: HashSet<&str> = evidence
        .compose
        .as_ref()
        .map(|c| c.services.keys().map(|s| s.as_str()).collect())
        .unwrap_or_default();

    for binary in APP_BINARIES {
        // Check cmd directory
        if !evidence.existing_cmd_dirs.contains(*binary) {
            findings.push(
                Finding::error(
                    "missing-binary-dir",
                    format!("expected binary directory cmd/{binary}/ does not exist"),
                )
                .with_why(format!(
                    "'{binary}' is an expected service but has no entry point"
                ))
                .with_help(format!("create cmd/{binary}/main.go")),
            );
        }

        // Check compose service
        if evidence.compose.is_some() && !compose_services.contains(binary) {
            // gateway maps to "server" in compose for backward compat -- skip if server exists
            if *binary == "gateway" && compose_services.contains("server") {
                findings.push(
                    Finding::info(
                        "gateway-server-alias",
                        "compose uses 'server' instead of 'gateway' -- consider renaming for consistency",
                    ),
                );
            } else {
                findings.push(
                    Finding::warning(
                        "binary-without-compose",
                        format!("binary '{binary}' exists but no matching compose service"),
                    )
                    .with_why(
                        "binary cannot be deployed via docker-compose without a service definition",
                    )
                    .with_help(format!(
                        "add '{binary}' service to deploy/compose/docker-compose.yaml"
                    )),
                );
            }
        }
    }

    // Check for compose services that are not expected binaries or infra
    let all_known: HashSet<&str> = APP_BINARIES
        .iter()
        .chain(INFRA_SERVICES.iter())
        .copied()
        .collect();

    for svc_name in &compose_services {
        // Allow "server" as gateway alias
        if *svc_name == "server" {
            continue;
        }
        if !all_known.contains(svc_name) {
            findings.push(Finding::info(
                "unknown-compose-service",
                format!("compose service '{svc_name}' is not a recognized market-foundry binary"),
            ));
        }
    }

    CheckResult::from_findings("binary-compose-drift", findings)
}

/// Check 3: Scan for residual old service names in active code.
fn check_naming_identity_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    if evidence.stale_references.is_empty() {
        return CheckResult::from_findings("naming-identity-drift", findings);
    }

    // Group by pattern for summary
    let mut by_pattern: HashMap<&str, Vec<&StaleReference>> = HashMap::new();
    for r in &evidence.stale_references {
        by_pattern.entry(r.pattern.as_str()).or_default().push(r);
    }

    for (pattern, refs) in &by_pattern {
        if refs.len() <= 3 {
            // Report individually
            for r in refs {
                findings.push(
                    Finding::warning(
                        "stale-name",
                        format!("{pattern} found in {}", r.file),
                    )
                    .with_location(format!("{}:{}", r.file, r.line_num))
                    .with_why("residual old names cause confusion and may break tooling that expects the new architecture")
                    .with_help("rename to the current service name or remove the reference"),
                );
            }
        } else {
            // Summarize
            let files: HashSet<&str> = refs.iter().map(|r| r.file.as_str()).collect();
            findings.push(
                Finding::warning(
                    "stale-name",
                    format!(
                        "{pattern}: {} occurrences across {} file(s)",
                        refs.len(),
                        files.len()
                    ),
                )
                .with_why("residual old names cause confusion and may break tooling that expects the new architecture")
                .with_help("grep for the pattern and update all references"),
            );
        }
    }

    CheckResult::from_findings("naming-identity-drift", findings)
}

/// Check 4: Verify docs and Makefile are consistent with reality.
fn check_docs_reality_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    // Check architecture docs exist
    for doc_path in ARCH_DOCS {
        let full = evidence.project_root.join(doc_path);
        if !full.is_file() {
            findings.push(
                Finding::warning(
                    "missing-arch-doc",
                    format!("architecture doc '{doc_path}' does not exist"),
                )
                .with_why("missing documentation leaves the architecture undocumented and hard to onboard")
                .with_help(format!("create {doc_path}")),
            );
        }
    }

    // Check docs/RUNTIME.md mentions all services
    let runtime_doc_path = evidence.project_root.join("docs/RUNTIME.md");
    if runtime_doc_path.is_file() {
        if let Ok(content) = std::fs::read_to_string(&runtime_doc_path) {
            let lower = content.to_lowercase();
            for binary in APP_BINARIES {
                if !lower.contains(&binary.to_lowercase()) {
                    findings.push(
                        Finding::warning(
                            "doc-service-missing",
                            format!("docs/RUNTIME.md does not mention service '{binary}'"),
                        )
                        .with_location("docs/RUNTIME.md")
                        .with_why("runtime doc is incomplete -- developers won't know this service exists")
                        .with_help(format!("add '{binary}' to the docs/RUNTIME.md service inventory")),
                    );
                }
            }
        }
    }

    // DEVELOPMENT.md / Makefile consistency
    if evidence.makefile_targets.is_empty() && evidence.dev_doc_targets.is_empty() {
        return CheckResult::from_findings("docs-reality-drift", findings);
    }

    // Targets referenced in DEVELOPMENT.md but not in Makefile
    for target in &evidence.dev_doc_targets {
        if !evidence.makefile_targets.contains(target) {
            if ["test", "build", "docker-build"].contains(&target.as_str()) {
                continue;
            }
            findings.push(
                Finding::error(
                    "doc-target-drift",
                    format!("DEVELOPMENT.md references `make {target}` but target not found in Makefile"),
                )
                .with_why("developers following the documented workflow will get 'No rule to make target' errors")
                .with_help(format!("add '{target}' target to Makefile or update DEVELOPMENT.md")),
            );
        }
    }

    // Key workflow targets that should be documented
    let workflow_targets = ["check", "verify", "smoke", "up-all", "down", "logs"];
    for target in &workflow_targets {
        if evidence.makefile_targets.contains(*target)
            && !evidence.dev_doc_targets.contains(*target)
        {
            findings.push(Finding::info(
                "undocumented-target",
                format!("Makefile has workflow target '{target}' not referenced in DEVELOPMENT.md"),
            ));
        }
    }

    CheckResult::from_findings("docs-reality-drift", findings)
}

/// Check 5: Verify each binary has a corresponding actor scope.
fn check_actor_scope_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    if evidence.existing_actor_scopes.is_empty() {
        return CheckResult::skip(
            "actor-scope-drift",
            "no actor scope directories found in internal/actors/scopes/",
        );
    }

    // Services that should have actor scopes (gateway/configctl may share server scope)
    let scope_expected: &[&str] = &["ingest", "derive", "store"];

    for scope in scope_expected {
        if !evidence.existing_actor_scopes.contains(*scope) {
            findings.push(
                Finding::error(
                    "missing-actor-scope",
                    format!("binary '{scope}' has no actor scope directory internal/actors/scopes/{scope}/"),
                )
                .with_why(format!("'{scope}' service has no actor hierarchy -- it cannot process messages"))
                .with_help(format!("create internal/actors/scopes/{scope}/ with at minimum a supervisor actor")),
            );
        }
    }

    // Actor scopes that don't correspond to any binary
    let binary_set: HashSet<&str> = APP_BINARIES.iter().copied().collect();
    for scope in &evidence.existing_actor_scopes {
        // Allow scopes that match binaries or are known shared scopes
        if !binary_set.contains(scope.as_str()) && scope != "server" {
            findings.push(Finding::info(
                "orphan-actor-scope",
                format!("actor scope '{scope}' does not correspond to any expected binary"),
            ));
        }
    }

    CheckResult::from_findings("actor-scope-drift", findings)
}

/// Check 6: Scan Go source for JetStream stream names and verify they match canonical streams.
fn check_stream_registry_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let source = match &evidence.source {
        Some(s) => s,
        None => return CheckResult::skip("stream-registry-drift", "source not scanned"),
    };

    let canonical: HashSet<&str> = CANONICAL_STREAMS.iter().copied().collect();

    // Check that all canonical streams exist in source
    for stream in CANONICAL_STREAMS {
        if !source.streams.contains_key(*stream) {
            findings.push(
                Finding::error(
                    "missing-canonical-stream",
                    format!("canonical stream '{stream}' not found in source"),
                )
                .with_why("services depend on this stream for message routing -- missing definition breaks the pipeline")
                .with_help("verify the registry adapter defines this stream"),
            );
        }
    }

    // Check for non-canonical streams (old or unexpected)
    for stream_name in source.streams.keys() {
        if !canonical.contains(stream_name.as_str()) {
            findings.push(
                Finding::warning(
                    "non-canonical-stream",
                    format!("stream '{stream_name}' is not in the canonical stream set"),
                )
                .with_why("unexpected streams may be leftovers from the old architecture or undocumented additions")
                .with_help(format!("if '{stream_name}' is intentional, add it to the canonical stream list in docs/architecture/stream-taxonomy.md")),
            );
        }
    }

    // Check that durable consumers target existing streams
    for (durable, stream) in &source.durables {
        if !source.streams.contains_key(stream.as_str()) {
            findings.push(
                Finding::error(
                    "durable-target-drift",
                    format!("durable '{durable}' targets stream '{stream}' which doesn't exist"),
                )
                .with_why("durable consumer will fail to bind at runtime -- messages will not be delivered")
                .with_help(format!("update durable '{durable}' to reference an existing stream")),
            );
        }
    }

    // Verify stream-subject alignment
    for (stream_name, stream_subjects) in &source.streams {
        for pattern in stream_subjects {
            let prefix = pattern.trim_end_matches(".>");
            let has_matching = source
                .subjects
                .iter()
                .any(|s| s.starts_with(prefix) || s == pattern);
            if !has_matching {
                findings.push(
                    Finding::warning(
                        "stream-subject-drift",
                        format!("stream '{stream_name}' declares subject pattern '{pattern}' but no matching concrete subjects found"),
                    )
                    .with_why("stream may be capturing zero messages if no publisher uses this subject pattern")
                    .with_help(format!("verify publishers emit to subjects matching '{pattern}'")),
                );
            }
        }
    }

    CheckResult::from_findings("stream-registry-drift", findings)
}

/// Check 7: Detect premature domain entry — streams/subjects that must not appear yet.
fn check_premature_domain_entry(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let source = match &evidence.source {
        Some(s) => s,
        None => return CheckResult::skip("premature-domain-entry", "source not scanned"),
    };

    for (stream, reason) in PROHIBITED_STREAMS {
        if source.streams.contains_key(*stream) {
            findings.push(
                Finding::error(
                    "premature-stream",
                    format!("prohibited stream '{stream}' found in source"),
                )
                .with_why(*reason)
                .with_help(format!(
                    "remove all references to '{stream}' until the domain is formally approved for entry"
                )),
            );
        }

        // Also check subjects that would belong to this stream
        let prefix = stream.trim_end_matches("_EVENTS").to_lowercase();
        for subject in &source.subjects {
            if subject.starts_with(&format!("{prefix}.events."))
                || subject.starts_with(&format!("{prefix}.query."))
            {
                findings.push(
                    Finding::error(
                        "premature-subject",
                        format!("subject '{subject}' belongs to prohibited domain '{prefix}'"),
                    )
                    .with_why(*reason),
                );
            }
        }
    }

    CheckResult::from_findings("premature-domain-entry", findings)
}

// ── Signal domain governance checks ─────────────────────────────────

/// Check signal-docs-drift: verify all required signal architecture docs exist.
fn check_signal_docs_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for doc_path in SIGNAL_DOCS {
        let full = evidence.project_root.join(doc_path);
        if full.is_file() {
            findings.push(Finding::info(
                "signal-doc-present",
                format!("signal doc present: {doc_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "signal-doc-missing",
                    format!("required signal architecture doc not found: {doc_path}"),
                )
                .with_why("signal governance requires canonical design docs to prevent drift between architecture and implementation")
                .with_help(format!("create {doc_path} following the pattern of existing evidence/stream docs")),
            );
        }
    }

    CheckResult::from_findings("signal-docs-drift", findings)
}

/// Check signal-adapter-drift: verify all expected signal NATS adapter files exist.
fn check_signal_adapter_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();
    let adapters_dir = evidence.project_root.join("internal/adapters/nats");

    for (file, purpose) in SIGNAL_ADAPTER_FILES {
        if adapters_dir.join(file).is_file() {
            findings.push(Finding::info(
                "signal-adapter-present",
                format!("signal adapter present: {file}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "signal-adapter-missing",
                    format!("signal adapter file missing: internal/adapters/nats/{file}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "restore internal/adapters/nats/{file} to match the post-S218 adapter layout"
                )),
            );
        }
    }

    CheckResult::from_findings("signal-adapter-drift", findings)
}

/// Check signal-domain-drift: verify signal domain and application layer files exist.
fn check_signal_domain_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for (file_path, purpose) in SIGNAL_DOMAIN_FILES {
        let full = evidence.project_root.join(file_path);
        if full.is_file() {
            findings.push(Finding::info(
                "signal-domain-present",
                format!("signal domain file present: {file_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "signal-domain-missing",
                    format!("signal domain file missing: {file_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {file_path} following the evidence domain pattern"
                )),
            );
        }
    }

    // Verify signal actors exist in derive and store scopes
    let signal_actors: &[(&str, &str)] = &[
        (
            "internal/actors/scopes/derive/signal_sampler_actor.go",
            "derive computes signal values from evidence",
        ),
        (
            "internal/actors/scopes/derive/signal_publisher_actor.go",
            "derive publishes signals to SIGNAL_EVENTS",
        ),
        (
            "internal/actors/scopes/store/signal_projection_actor.go",
            "store projects signals to KV buckets",
        ),
    ];

    for (actor_path, purpose) in signal_actors {
        let full = evidence.project_root.join(actor_path);
        if full.is_file() {
            findings.push(Finding::info(
                "signal-actor-present",
                format!("signal actor present: {actor_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "signal-actor-missing",
                    format!("signal actor file missing: {actor_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {actor_path} following the evidence actor pattern"
                )),
            );
        }
    }

    push_store_consumer_wiring_finding(
        &mut findings,
        evidence,
        "signal",
        &[
            "natssignal.StoreRSISignalConsumer()",
            "natssignal.StoreEMACrossoverSignalConsumer()",
        ],
        "store consumes signal events for projection via GenericConsumerActor wiring",
    );

    // Verify signal HTTP interface exists
    let signal_http: &[(&str, &str)] = &[
        (
            "internal/interfaces/http/handlers/signal.go",
            "HTTP handler for signal queries",
        ),
        (
            "internal/interfaces/http/routes/signal.go",
            "HTTP route registration for signal endpoints",
        ),
    ];

    for (http_path, purpose) in signal_http {
        let full = evidence.project_root.join(http_path);
        if full.is_file() {
            findings.push(Finding::info(
                "signal-http-present",
                format!("signal HTTP file present: {http_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "signal-http-missing",
                    format!("signal HTTP file missing: {http_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {http_path} following the evidence HTTP pattern"
                )),
            );
        }
    }

    CheckResult::from_findings("signal-domain-drift", findings)
}

/// Check signal-config-drift: verify signal_families config alignment between derive and store.
fn check_signal_config_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let configs_dir = evidence.project_root.join("deploy/configs");

    let derive_config = configs_dir.join("derive.jsonc");
    let store_config = configs_dir.join("store.jsonc");

    let derive_has_signal = if derive_config.is_file() {
        match std::fs::read_to_string(&derive_config) {
            Ok(content) => content.contains("signal_families"),
            Err(_) => false,
        }
    } else {
        false
    };

    let store_has_signal = if store_config.is_file() {
        match std::fs::read_to_string(&store_config) {
            Ok(content) => content.contains("signal_families"),
            Err(_) => false,
        }
    } else {
        false
    };

    match (derive_has_signal, store_has_signal) {
        (true, true) => {
            findings.push(Finding::info(
                "signal-config-aligned",
                "both derive.jsonc and store.jsonc declare signal_families",
            ));
        }
        (true, false) => {
            findings.push(
                Finding::error(
                    "signal-config-asymmetry",
                    "derive.jsonc has signal_families but store.jsonc does not",
                )
                .with_why("derive will produce signal events but store won't consume them — events accumulate with no projection")
                .with_help("add pipeline.signal_families to deploy/configs/store.jsonc"),
            );
        }
        (false, true) => {
            findings.push(
                Finding::error(
                    "signal-config-asymmetry",
                    "store.jsonc has signal_families but derive.jsonc does not",
                )
                .with_why("store consumer will idle because derive isn't producing signal events")
                .with_help("add pipeline.signal_families to deploy/configs/derive.jsonc"),
            );
        }
        (false, false) => {
            findings.push(
                Finding::warning(
                    "signal-config-absent",
                    "neither derive.jsonc nor store.jsonc declare signal_families",
                )
                .with_why("signal pipeline is inactive — this is safe but means no signal processing occurs")
                .with_help("add pipeline.signal_families: [\"rsi\"] to both derive.jsonc and store.jsonc to activate"),
            );
        }
    }

    CheckResult::from_findings("signal-config-drift", findings)
}

/// Check signal-contracts-drift: verify signal subjects, durables, and KV buckets exist in source.
fn check_signal_contracts_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let source = match &evidence.source {
        Some(s) => s,
        None => return CheckResult::skip("signal-contracts-drift", "source not scanned"),
    };

    // Check signal subjects exist in source
    for (subject, purpose) in SIGNAL_EXPECTED_SUBJECTS {
        let found = source.subjects.iter().any(|s| s.contains(subject));
        if found {
            findings.push(Finding::info(
                "signal-subject-present",
                format!("signal subject found: {subject}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "signal-subject-missing",
                    format!("signal subject not found in source: {subject}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natssignal/registry.go for subject definitions",
                ),
            );
        }
    }

    // Check signal durable consumers
    for (durable, purpose) in SIGNAL_EXPECTED_DURABLES {
        if source.durables.contains_key(*durable) {
            findings.push(Finding::info(
                "signal-durable-present",
                format!("signal durable consumer found: {durable}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "signal-durable-missing",
                    format!("signal durable consumer not found: {durable}"),
                )
                .with_why(*purpose)
                .with_help("check internal/adapters/nats/natssignal/registry.go for consumer spec"),
            );
        }
    }

    // Check signal KV bucket names in source
    let nats_dir = evidence.project_root.join("internal/adapters/nats");
    for (bucket, purpose) in SIGNAL_EXPECTED_BUCKETS {
        let found = scan_dir_for_string(&nats_dir, bucket);
        if found {
            findings.push(Finding::info(
                "signal-bucket-present",
                format!("signal KV bucket found in source: {bucket}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "signal-bucket-missing",
                    format!("signal KV bucket name not found in source: {bucket}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natssignal/kv_store.go for bucket definition",
                ),
            );
        }
    }

    CheckResult::from_findings("signal-contracts-drift", findings)
}

// ── Decision domain governance checks ────────────────────────────────

/// Check decision-docs-drift: verify all required decision architecture docs exist.
fn check_decision_docs_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for doc_path in DECISION_DOCS {
        let full = evidence.project_root.join(doc_path);
        if full.is_file() {
            findings.push(Finding::info(
                "decision-doc-present",
                format!("decision doc present: {doc_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "decision-doc-missing",
                    format!("required decision architecture doc not found: {doc_path}"),
                )
                .with_why("decision governance requires canonical design docs to prevent drift between architecture and implementation")
                .with_help(format!("create {doc_path} following the pattern of existing signal/evidence docs")),
            );
        }
    }

    CheckResult::from_findings("decision-docs-drift", findings)
}

/// Check decision-adapter-drift: verify all expected decision NATS adapter files exist.
fn check_decision_adapter_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();
    let adapters_dir = evidence.project_root.join("internal/adapters/nats");

    for (file, purpose) in DECISION_ADAPTER_FILES {
        if adapters_dir.join(file).is_file() {
            findings.push(Finding::info(
                "decision-adapter-present",
                format!("decision adapter present: {file}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "decision-adapter-missing",
                    format!("decision adapter file missing: internal/adapters/nats/{file}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "restore internal/adapters/nats/{file} to match the post-S218 adapter layout"
                )),
            );
        }
    }

    CheckResult::from_findings("decision-adapter-drift", findings)
}

/// Check decision-domain-drift: verify decision domain and application layer files exist.
fn check_decision_domain_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for (file_path, purpose) in DECISION_DOMAIN_FILES {
        let full = evidence.project_root.join(file_path);
        if full.is_file() {
            findings.push(Finding::info(
                "decision-domain-present",
                format!("decision domain file present: {file_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "decision-domain-missing",
                    format!("decision domain file missing: {file_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {file_path} following the signal domain pattern"
                )),
            );
        }
    }

    // Verify decision actors exist in derive and store scopes
    let decision_actors: &[(&str, &str)] = &[
        (
            "internal/actors/scopes/derive/decision_evaluator_actor.go",
            "derive evaluates signal data to produce decisions",
        ),
        (
            "internal/actors/scopes/derive/decision_publisher_actor.go",
            "derive publishes decisions to DECISION_EVENTS",
        ),
        (
            "internal/actors/scopes/store/decision_projection_actor.go",
            "store projects decisions to KV buckets",
        ),
    ];

    for (actor_path, purpose) in decision_actors {
        let full = evidence.project_root.join(actor_path);
        if full.is_file() {
            findings.push(Finding::info(
                "decision-actor-present",
                format!("decision actor present: {actor_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "decision-actor-missing",
                    format!("decision actor file missing: {actor_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {actor_path} following the signal actor pattern"
                )),
            );
        }
    }

    push_store_consumer_wiring_finding(
        &mut findings,
        evidence,
        "decision",
        &["natsdecision.StoreRSIOversoldDecisionConsumer()"],
        "store consumes decision events for projection via GenericConsumerActor wiring",
    );

    // Verify decision HTTP interface exists
    let decision_http: &[(&str, &str)] = &[
        (
            "internal/interfaces/http/handlers/decision.go",
            "HTTP handler for decision queries",
        ),
        (
            "internal/interfaces/http/routes/decision.go",
            "HTTP route registration for decision endpoints",
        ),
    ];

    for (http_path, purpose) in decision_http {
        let full = evidence.project_root.join(http_path);
        if full.is_file() {
            findings.push(Finding::info(
                "decision-http-present",
                format!("decision HTTP file present: {http_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "decision-http-missing",
                    format!("decision HTTP file missing: {http_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {http_path} following the signal HTTP pattern"
                )),
            );
        }
    }

    CheckResult::from_findings("decision-domain-drift", findings)
}

/// Check decision-config-drift: verify decision_families config alignment between derive and store.
fn check_decision_config_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let configs_dir = evidence.project_root.join("deploy/configs");

    let derive_config = configs_dir.join("derive.jsonc");
    let store_config = configs_dir.join("store.jsonc");

    let derive_has_decision = if derive_config.is_file() {
        match std::fs::read_to_string(&derive_config) {
            Ok(content) => content.contains("decision_families"),
            Err(_) => false,
        }
    } else {
        false
    };

    let store_has_decision = if store_config.is_file() {
        match std::fs::read_to_string(&store_config) {
            Ok(content) => content.contains("decision_families"),
            Err(_) => false,
        }
    } else {
        false
    };

    match (derive_has_decision, store_has_decision) {
        (true, true) => {
            findings.push(Finding::info(
                "decision-config-aligned",
                "both derive.jsonc and store.jsonc declare decision_families",
            ));
        }
        (true, false) => {
            findings.push(
                Finding::error(
                    "decision-config-asymmetry",
                    "derive.jsonc has decision_families but store.jsonc does not",
                )
                .with_why("derive will produce decision events but store won't consume them — events accumulate with no projection")
                .with_help("add pipeline.decision_families to deploy/configs/store.jsonc"),
            );
        }
        (false, true) => {
            findings.push(
                Finding::error(
                    "decision-config-asymmetry",
                    "store.jsonc has decision_families but derive.jsonc does not",
                )
                .with_why("store consumer will idle because derive isn't producing decision events")
                .with_help("add pipeline.decision_families to deploy/configs/derive.jsonc"),
            );
        }
        (false, false) => {
            findings.push(
                Finding::warning(
                    "decision-config-absent",
                    "neither derive.jsonc nor store.jsonc declare decision_families",
                )
                .with_why("decision pipeline is inactive — this is safe but means no decision processing occurs")
                .with_help("add pipeline.decision_families: [\"rsi_oversold\"] to both derive.jsonc and store.jsonc to activate"),
            );
        }
    }

    CheckResult::from_findings("decision-config-drift", findings)
}

/// Check decision-contracts-drift: verify decision subjects, durables, and KV buckets exist in source.
fn check_decision_contracts_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let source = match &evidence.source {
        Some(s) => s,
        None => return CheckResult::skip("decision-contracts-drift", "source not scanned"),
    };

    // Check decision subjects exist in source
    for (subject, purpose) in DECISION_EXPECTED_SUBJECTS {
        let found = source.subjects.iter().any(|s| s.contains(subject));
        if found {
            findings.push(Finding::info(
                "decision-subject-present",
                format!("decision subject found: {subject}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "decision-subject-missing",
                    format!("decision subject not found in source: {subject}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsdecision/registry.go for subject definitions",
                ),
            );
        }
    }

    // Check decision durable consumers
    for (durable, purpose) in DECISION_EXPECTED_DURABLES {
        if source.durables.contains_key(*durable) {
            findings.push(Finding::info(
                "decision-durable-present",
                format!("decision durable consumer found: {durable}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "decision-durable-missing",
                    format!("decision durable consumer not found: {durable}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsdecision/registry.go for consumer spec",
                ),
            );
        }
    }

    // Check decision KV bucket names in source
    let nats_dir = evidence.project_root.join("internal/adapters/nats");
    for (bucket, purpose) in DECISION_EXPECTED_BUCKETS {
        let found = scan_dir_for_string(&nats_dir, bucket);
        if found {
            findings.push(Finding::info(
                "decision-bucket-present",
                format!("decision KV bucket found in source: {bucket}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "decision-bucket-missing",
                    format!("decision KV bucket name not found in source: {bucket}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsdecision/kv_store.go for bucket definition",
                ),
            );
        }
    }

    CheckResult::from_findings("decision-contracts-drift", findings)
}

// ── Strategy domain governance checks ─────────────────────────────────

/// Check strategy-docs-drift: verify all required strategy architecture docs exist.
fn check_strategy_docs_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for doc_path in STRATEGY_DOCS {
        let full = evidence.project_root.join(doc_path);
        if full.is_file() {
            findings.push(Finding::info(
                "strategy-doc-present",
                format!("strategy doc present: {doc_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "strategy-doc-missing",
                    format!("required strategy architecture doc not found: {doc_path}"),
                )
                .with_why("strategy governance requires canonical design docs to prevent drift between architecture and implementation")
                .with_help(format!("create {doc_path} following the pattern of existing decision/signal docs")),
            );
        }
    }

    CheckResult::from_findings("strategy-docs-drift", findings)
}

/// Check strategy-adapter-drift: verify all expected strategy NATS adapter files exist.
fn check_strategy_adapter_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();
    let adapters_dir = evidence.project_root.join("internal/adapters/nats");

    for (file, purpose) in STRATEGY_ADAPTER_FILES {
        if adapters_dir.join(file).is_file() {
            findings.push(Finding::info(
                "strategy-adapter-present",
                format!("strategy adapter present: {file}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "strategy-adapter-missing",
                    format!("strategy adapter file missing: internal/adapters/nats/{file}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "restore internal/adapters/nats/{file} to match the post-S218 adapter layout"
                )),
            );
        }
    }

    CheckResult::from_findings("strategy-adapter-drift", findings)
}

/// Check strategy-domain-drift: verify strategy domain and application layer files exist.
fn check_strategy_domain_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for (file_path, purpose) in STRATEGY_DOMAIN_FILES {
        let full = evidence.project_root.join(file_path);
        if full.is_file() {
            findings.push(Finding::info(
                "strategy-domain-present",
                format!("strategy domain file present: {file_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "strategy-domain-missing",
                    format!("strategy domain file missing: {file_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {file_path} following the decision domain pattern"
                )),
            );
        }
    }

    // Verify strategy actors exist in derive and store scopes
    let strategy_actors: &[(&str, &str)] = &[
        (
            "internal/actors/scopes/derive/strategy_resolver_actor.go",
            "derive resolves decision data into strategy output",
        ),
        (
            "internal/actors/scopes/derive/strategy_publisher_actor.go",
            "derive publishes strategies to STRATEGY_EVENTS",
        ),
        (
            "internal/actors/scopes/store/strategy_projection_actor.go",
            "store projects strategies to KV buckets",
        ),
    ];

    for (actor_path, purpose) in strategy_actors {
        let full = evidence.project_root.join(actor_path);
        if full.is_file() {
            findings.push(Finding::info(
                "strategy-actor-present",
                format!("strategy actor present: {actor_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "strategy-actor-missing",
                    format!("strategy actor file missing: {actor_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {actor_path} following the decision actor pattern"
                )),
            );
        }
    }

    push_store_consumer_wiring_finding(
        &mut findings,
        evidence,
        "strategy",
        &["natsstrategy.StoreMeanReversionEntryStrategyConsumer()"],
        "store consumes strategy events for projection via GenericConsumerActor wiring",
    );

    // Verify strategy HTTP interface exists
    let strategy_http: &[(&str, &str)] = &[
        (
            "internal/interfaces/http/handlers/strategy.go",
            "HTTP handler for strategy queries",
        ),
        (
            "internal/interfaces/http/routes/strategy.go",
            "HTTP route registration for strategy endpoints",
        ),
    ];

    for (http_path, purpose) in strategy_http {
        let full = evidence.project_root.join(http_path);
        if full.is_file() {
            findings.push(Finding::info(
                "strategy-http-present",
                format!("strategy HTTP file present: {http_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "strategy-http-missing",
                    format!("strategy HTTP file missing: {http_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {http_path} following the decision HTTP pattern"
                )),
            );
        }
    }

    CheckResult::from_findings("strategy-domain-drift", findings)
}

/// Check strategy-config-drift: verify strategy_families config alignment between derive and store.
fn check_strategy_config_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let configs_dir = evidence.project_root.join("deploy/configs");

    let derive_config = configs_dir.join("derive.jsonc");
    let store_config = configs_dir.join("store.jsonc");

    let derive_has_strategy = if derive_config.is_file() {
        match std::fs::read_to_string(&derive_config) {
            Ok(content) => content.contains("strategy_families"),
            Err(_) => false,
        }
    } else {
        false
    };

    let store_has_strategy = if store_config.is_file() {
        match std::fs::read_to_string(&store_config) {
            Ok(content) => content.contains("strategy_families"),
            Err(_) => false,
        }
    } else {
        false
    };

    match (derive_has_strategy, store_has_strategy) {
        (true, true) => {
            findings.push(Finding::info(
                "strategy-config-aligned",
                "both derive.jsonc and store.jsonc declare strategy_families",
            ));
        }
        (true, false) => {
            findings.push(
                Finding::error(
                    "strategy-config-asymmetry",
                    "derive.jsonc has strategy_families but store.jsonc does not",
                )
                .with_why("derive will produce strategy events but store won't consume them — events accumulate with no projection")
                .with_help("add pipeline.strategy_families to deploy/configs/store.jsonc"),
            );
        }
        (false, true) => {
            findings.push(
                Finding::error(
                    "strategy-config-asymmetry",
                    "store.jsonc has strategy_families but derive.jsonc does not",
                )
                .with_why("store consumer will idle because derive isn't producing strategy events")
                .with_help("add pipeline.strategy_families to deploy/configs/derive.jsonc"),
            );
        }
        (false, false) => {
            findings.push(
                Finding::warning(
                    "strategy-config-absent",
                    "neither derive.jsonc nor store.jsonc declare strategy_families",
                )
                .with_why("strategy pipeline is inactive — this is expected before strategy implementation begins")
                .with_help("add pipeline.strategy_families: [\"mean_reversion_entry\"] to both derive.jsonc and store.jsonc when ready to activate"),
            );
        }
    }

    CheckResult::from_findings("strategy-config-drift", findings)
}

/// Check strategy-contracts-drift: verify strategy subjects, durables, and KV buckets exist in source.
fn check_strategy_contracts_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let source = match &evidence.source {
        Some(s) => s,
        None => return CheckResult::skip("strategy-contracts-drift", "source not scanned"),
    };

    // Check strategy subjects exist in source
    for (subject, purpose) in STRATEGY_EXPECTED_SUBJECTS {
        let found = source.subjects.iter().any(|s| s.contains(subject));
        if found {
            findings.push(Finding::info(
                "strategy-subject-present",
                format!("strategy subject found: {subject}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "strategy-subject-missing",
                    format!("strategy subject not found in source: {subject}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsstrategy/registry.go for subject definitions",
                ),
            );
        }
    }

    // Check strategy durable consumers
    for (durable, purpose) in STRATEGY_EXPECTED_DURABLES {
        if source.durables.contains_key(*durable) {
            findings.push(Finding::info(
                "strategy-durable-present",
                format!("strategy durable consumer found: {durable}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "strategy-durable-missing",
                    format!("strategy durable consumer not found: {durable}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsstrategy/registry.go for consumer spec",
                ),
            );
        }
    }

    // Check strategy KV bucket names in source
    let nats_dir = evidence.project_root.join("internal/adapters/nats");
    for (bucket, purpose) in STRATEGY_EXPECTED_BUCKETS {
        let found = scan_dir_for_string(&nats_dir, bucket);
        if found {
            findings.push(Finding::info(
                "strategy-bucket-present",
                format!("strategy KV bucket found in source: {bucket}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "strategy-bucket-missing",
                    format!("strategy KV bucket name not found in source: {bucket}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsstrategy/kv_store.go for bucket definition",
                ),
            );
        }
    }

    CheckResult::from_findings("strategy-contracts-drift", findings)
}

/// Scan a directory recursively for a string pattern in .go files.
fn scan_dir_for_string(dir: &Path, pattern: &str) -> bool {
    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return false,
    };
    for entry in entries.flatten() {
        let path = entry.path();
        if path.is_dir() {
            if scan_dir_for_string(&path, pattern) {
                return true;
            }
        } else if path.extension().and_then(|e| e.to_str()) == Some("go") {
            if let Ok(content) = std::fs::read_to_string(&path) {
                if content.contains(pattern) {
                    return true;
                }
            }
        }
    }
    false
}

/// Like scan_dir_for_string but scans .sql files — used to verify a table is
/// declared in deploy/migrations (the authoritative source for table DDL).
fn scan_sql_dir_for_string(dir: &Path, pattern: &str) -> bool {
    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return false,
    };
    for entry in entries.flatten() {
        let path = entry.path();
        if path.is_dir() {
            if scan_sql_dir_for_string(&path, pattern) {
                return true;
            }
        } else if path.extension().and_then(|e| e.to_str()) == Some("sql") {
            if let Ok(content) = std::fs::read_to_string(&path) {
                if content.contains(pattern) {
                    return true;
                }
            }
        }
    }
    false
}

fn push_store_consumer_wiring_finding(
    findings: &mut Vec<Finding>,
    evidence: &Evidence,
    domain: &str,
    markers: &[&str],
    purpose: &str,
) {
    let generic_consumer = evidence
        .project_root
        .join("internal/actors/scopes/store/generic_consumer_actor.go");
    let supervisor = evidence
        .project_root
        .join("internal/actors/scopes/store/store_supervisor.go");

    let generic_exists = generic_consumer.is_file();
    let supervisor_content = std::fs::read_to_string(&supervisor).ok();
    let markers_present = supervisor_content.as_ref().map_or(false, |content| {
        markers.iter().all(|marker| content.contains(marker))
    });

    if generic_exists && supervisor.is_file() && markers_present {
        findings.push(Finding::info(
            "store-consumer-wiring-present",
            format!(
                "{domain} store consumer wiring present via generic_consumer_actor.go + store_supervisor.go"
            ),
        ));
    } else {
        findings.push(
            Finding::error(
                "store-consumer-wiring-missing",
                format!(
                    "{domain} store consumer wiring missing from internal/actors/scopes/store/generic_consumer_actor.go or store_supervisor.go"
                ),
            )
            .with_why(purpose)
            .with_help(format!(
                "restore {domain} pipeline wiring in internal/actors/scopes/store/store_supervisor.go using the GenericConsumerActor pattern"
            )),
        );
    }
}

// ── Risk domain governance checks (S63) ──────────────────────────────

/// Check risk-docs-drift: verify all required risk architecture docs exist (S62 output).
fn check_risk_docs_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for doc_path in RISK_DOCS {
        let full = evidence.project_root.join(doc_path);
        if full.is_file() {
            findings.push(Finding::info(
                "risk-doc-present",
                format!("risk doc present: {doc_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "risk-doc-missing",
                    format!("required risk architecture doc not found: {doc_path}"),
                )
                .with_why("risk governance requires canonical design docs (S62) to prevent drift between architecture and future implementation")
                .with_help(format!("create {doc_path} following the pattern of existing strategy/decision docs")),
            );
        }
    }

    CheckResult::from_findings("risk-docs-drift", findings)
}

// ── Execution domain governance checks (S70) ─────────────────────────

/// Check execution-docs-drift: verify all required execution architecture docs exist (S69 output).
fn check_execution_docs_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for doc_path in EXECUTION_DOCS {
        let full = evidence.project_root.join(doc_path);
        if full.is_file() {
            findings.push(Finding::info(
                "execution-doc-present",
                format!("execution doc present: {doc_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "execution-doc-missing",
                    format!("required execution architecture doc not found: {doc_path}"),
                )
                .with_why("execution governance requires canonical design docs (S69) to prevent drift between architecture and future implementation")
                .with_help(format!("create {doc_path} following the pattern of existing risk/strategy docs")),
            );
        }
    }

    CheckResult::from_findings("execution-docs-drift", findings)
}

// ── Risk domain drift checks (activated S70, risk implemented S64) ────
// These functions follow the exact same pattern as signal/decision/strategy.
// Activated in S70 after risk implementation was completed in S64.

/// Check risk-adapter-drift: verify all expected risk NATS adapter files exist.
/// Activated in S70 (risk implementation opened in S64).
fn check_risk_adapter_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();
    let adapters_dir = evidence.project_root.join("internal/adapters/nats");

    for (file, purpose) in RISK_ADAPTER_FILES {
        if adapters_dir.join(file).is_file() {
            findings.push(Finding::info(
                "risk-adapter-present",
                format!("risk adapter present: {file}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "risk-adapter-missing",
                    format!("risk adapter file missing: internal/adapters/nats/{file}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "restore internal/adapters/nats/{file} to match the post-S218 adapter layout"
                )),
            );
        }
    }

    CheckResult::from_findings("risk-adapter-drift", findings)
}

/// Check risk-domain-drift: verify risk domain and application layer files exist.
/// Activated in S70 (risk implementation opened in S64).
fn check_risk_domain_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for (file_path, purpose) in RISK_DOMAIN_FILES {
        let full = evidence.project_root.join(file_path);
        if full.is_file() {
            findings.push(Finding::info(
                "risk-domain-present",
                format!("risk domain file present: {file_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "risk-domain-missing",
                    format!("risk domain file missing: {file_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {file_path} following the strategy domain pattern"
                )),
            );
        }
    }

    // Verify risk actors exist in derive and store scopes
    let risk_actors: &[(&str, &str)] = &[
        (
            "internal/actors/scopes/derive/risk_evaluator_actor.go",
            "derive evaluates strategy data to produce risk assessments",
        ),
        (
            "internal/actors/scopes/derive/risk_publisher_actor.go",
            "derive publishes risk assessments to RISK_EVENTS",
        ),
        (
            "internal/actors/scopes/store/risk_projection_actor.go",
            "store projects risk assessments to KV buckets",
        ),
    ];

    for (actor_path, purpose) in risk_actors {
        let full = evidence.project_root.join(actor_path);
        if full.is_file() {
            findings.push(Finding::info(
                "risk-actor-present",
                format!("risk actor present: {actor_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "risk-actor-missing",
                    format!("risk actor file missing: {actor_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {actor_path} following the strategy actor pattern"
                )),
            );
        }
    }

    push_store_consumer_wiring_finding(
        &mut findings,
        evidence,
        "risk",
        &["natsrisk.StorePositionExposureRiskConsumer()"],
        "store consumes risk events for projection via GenericConsumerActor wiring",
    );

    // Verify risk HTTP interface exists
    let risk_http: &[(&str, &str)] = &[
        (
            "internal/interfaces/http/handlers/risk.go",
            "HTTP handler for risk queries",
        ),
        (
            "internal/interfaces/http/routes/risk.go",
            "HTTP route registration for risk endpoints",
        ),
    ];

    for (http_path, purpose) in risk_http {
        let full = evidence.project_root.join(http_path);
        if full.is_file() {
            findings.push(Finding::info(
                "risk-http-present",
                format!("risk HTTP file present: {http_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "risk-http-missing",
                    format!("risk HTTP file missing: {http_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {http_path} following the strategy HTTP pattern"
                )),
            );
        }
    }

    CheckResult::from_findings("risk-domain-drift", findings)
}

/// Check risk-config-drift: verify risk_families config alignment between derive and store.
/// Activated in S70 (risk implementation opened in S64).
fn check_risk_config_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let configs_dir = evidence.project_root.join("deploy/configs");

    let derive_config = configs_dir.join("derive.jsonc");
    let store_config = configs_dir.join("store.jsonc");

    let derive_has_risk = if derive_config.is_file() {
        match std::fs::read_to_string(&derive_config) {
            Ok(content) => content.contains("risk_families"),
            Err(_) => false,
        }
    } else {
        false
    };

    let store_has_risk = if store_config.is_file() {
        match std::fs::read_to_string(&store_config) {
            Ok(content) => content.contains("risk_families"),
            Err(_) => false,
        }
    } else {
        false
    };

    match (derive_has_risk, store_has_risk) {
        (true, true) => {
            findings.push(Finding::info(
                "risk-config-aligned",
                "both derive.jsonc and store.jsonc declare risk_families",
            ));
        }
        (true, false) => {
            findings.push(
                Finding::error(
                    "risk-config-asymmetry",
                    "derive.jsonc has risk_families but store.jsonc does not",
                )
                .with_why("derive will produce risk events but store won't consume them — events accumulate with no projection")
                .with_help("add pipeline.risk_families to deploy/configs/store.jsonc"),
            );
        }
        (false, true) => {
            findings.push(
                Finding::error(
                    "risk-config-asymmetry",
                    "store.jsonc has risk_families but derive.jsonc does not",
                )
                .with_why("store consumer will idle because derive isn't producing risk events")
                .with_help("add pipeline.risk_families to deploy/configs/derive.jsonc"),
            );
        }
        (false, false) => {
            findings.push(
                Finding::warning(
                    "risk-config-absent",
                    "neither derive.jsonc nor store.jsonc declare risk_families",
                )
                .with_why("risk pipeline is inactive — this is expected before risk implementation begins in S64")
                .with_help("add pipeline.risk_families: [\"position_exposure\"] to both derive.jsonc and store.jsonc when S64 activates"),
            );
        }
    }

    CheckResult::from_findings("risk-config-drift", findings)
}

/// Check risk-contracts-drift: verify risk subjects, durables, and KV buckets exist in source.
/// Activated in S70 (risk implementation opened in S64).
fn check_risk_contracts_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let source = match &evidence.source {
        Some(s) => s,
        None => return CheckResult::skip("risk-contracts-drift", "source not scanned"),
    };

    // Check risk subjects exist in source
    for (subject, purpose) in RISK_EXPECTED_SUBJECTS {
        let found = source.subjects.iter().any(|s| s.contains(subject));
        if found {
            findings.push(Finding::info(
                "risk-subject-present",
                format!("risk subject found: {subject}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "risk-subject-missing",
                    format!("risk subject not found in source: {subject}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsrisk/registry.go for subject definitions",
                ),
            );
        }
    }

    // Check risk durable consumers
    for (durable, purpose) in RISK_EXPECTED_DURABLES {
        if source.durables.contains_key(*durable) {
            findings.push(Finding::info(
                "risk-durable-present",
                format!("risk durable consumer found: {durable}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "risk-durable-missing",
                    format!("risk durable consumer not found: {durable}"),
                )
                .with_why(*purpose)
                .with_help("check internal/adapters/nats/natsrisk/registry.go for consumer spec"),
            );
        }
    }

    // Check risk KV bucket names in source
    let nats_dir = evidence.project_root.join("internal/adapters/nats");
    for (bucket, purpose) in RISK_EXPECTED_BUCKETS {
        let found = scan_dir_for_string(&nats_dir, bucket);
        if found {
            findings.push(Finding::info(
                "risk-bucket-present",
                format!("risk KV bucket found in source: {bucket}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "risk-bucket-missing",
                    format!("risk KV bucket name not found in source: {bucket}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsrisk/kv_store.go for bucket definition",
                ),
            );
        }
    }

    CheckResult::from_findings("risk-contracts-drift", findings)
}

// ── Execution domain drift checks (activated S71, execution implemented S71) ────
// These functions follow the exact same pattern as signal/decision/strategy/risk.
// Activated in S71 after execution implementation was completed.

/// Check execution-adapter-drift: verify all expected execution NATS adapter files exist.
/// Activated in S71 (execution implementation opened in S71).
fn check_execution_adapter_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();
    let adapters_dir = evidence.project_root.join("internal/adapters/nats");

    for (file, purpose) in EXECUTION_ADAPTER_FILES {
        if adapters_dir.join(file).is_file() {
            findings.push(Finding::info(
                "execution-adapter-present",
                format!("execution adapter present: {file}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "execution-adapter-missing",
                    format!("execution adapter file missing: internal/adapters/nats/{file}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "restore internal/adapters/nats/{file} to match the post-S218 adapter layout"
                )),
            );
        }
    }

    CheckResult::from_findings("execution-adapter-drift", findings)
}

/// Check execution-domain-drift: verify execution domain and application layer files exist.
/// Activated in S71, hardened in S83 (execute scope actors + control + venue files added).
fn check_execution_domain_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    for (file_path, purpose) in EXECUTION_DOMAIN_FILES {
        let full = evidence.project_root.join(file_path);
        if full.is_file() {
            findings.push(Finding::info(
                "execution-domain-present",
                format!("execution domain file present: {file_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "execution-domain-missing",
                    format!("execution domain file missing: {file_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {file_path} following the risk domain pattern"
                )),
            );
        }
    }

    // Verify execution actors exist in derive, store, and execute scopes
    let execution_actors: &[(&str, &str)] = &[
        (
            "internal/actors/scopes/derive/execution_evaluator_actor.go",
            "derive evaluates risk data to produce execution intents",
        ),
        (
            "internal/actors/scopes/derive/execution_publisher_actor.go",
            "derive publishes execution intents to EXECUTION_EVENTS",
        ),
        (
            "internal/actors/scopes/store/execution_projection_actor.go",
            "store projects execution intents to KV buckets",
        ),
        (
            "internal/actors/scopes/execute/execute_supervisor.go",
            "execute binary root actor — venue adapter lifecycle and consumer wiring",
        ),
        (
            "internal/actors/scopes/execute/venue_adapter_actor.go",
            "execute processes intents through kill switch, staleness guard, and venue port",
        ),
    ];

    for (actor_path, purpose) in execution_actors {
        let full = evidence.project_root.join(actor_path);
        if full.is_file() {
            findings.push(Finding::info(
                "execution-actor-present",
                format!("execution actor present: {actor_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "execution-actor-missing",
                    format!("execution actor file missing: {actor_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {actor_path} following the risk actor pattern"
                )),
            );
        }
    }

    push_store_consumer_wiring_finding(
        &mut findings,
        evidence,
        "execution",
        &[
            "natsexecution.StorePaperOrderExecutionConsumer()",
            "natsexecution.StoreVenueMarketOrderFillConsumer()",
        ],
        "store consumes execution and fill events for projection via GenericConsumerActor wiring",
    );

    // Verify execution HTTP interface exists
    let execution_http: &[(&str, &str)] = &[
        (
            "internal/interfaces/http/handlers/execution.go",
            "HTTP handler for execution queries",
        ),
        (
            "internal/interfaces/http/routes/execution.go",
            "HTTP route registration for execution endpoints",
        ),
    ];

    for (http_path, purpose) in execution_http {
        let full = evidence.project_root.join(http_path);
        if full.is_file() {
            findings.push(Finding::info(
                "execution-http-present",
                format!("execution HTTP file present: {http_path}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "execution-http-missing",
                    format!("execution HTTP file missing: {http_path}"),
                )
                .with_why(*purpose)
                .with_help(format!(
                    "create {http_path} following the risk HTTP pattern"
                )),
            );
        }
    }

    CheckResult::from_findings("execution-domain-drift", findings)
}

/// Check execution-config-drift: verify execution_families config alignment between derive, store, and execute.
/// Activated in S71, hardened in S83 to include execute.jsonc and venue config.
fn check_execution_config_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let configs_dir = evidence.project_root.join("deploy/configs");

    let has_execution = |name: &str| -> bool {
        let path = configs_dir.join(name);
        if path.is_file() {
            match std::fs::read_to_string(&path) {
                Ok(content) => content.contains("execution_families"),
                Err(_) => false,
            }
        } else {
            false
        }
    };

    let derive_has = has_execution("derive.jsonc");
    let store_has = has_execution("store.jsonc");
    let execute_has = has_execution("execute.jsonc");

    // derive ↔ store symmetry
    match (derive_has, store_has) {
        (true, true) => {
            findings.push(Finding::info(
                "execution-config-aligned",
                "both derive.jsonc and store.jsonc declare execution_families",
            ));
        }
        (true, false) => {
            findings.push(
                Finding::error(
                    "execution-config-asymmetry",
                    "derive.jsonc has execution_families but store.jsonc does not",
                )
                .with_why("derive will produce execution events but store won't consume them — events accumulate with no projection")
                .with_help("add pipeline.execution_families to deploy/configs/store.jsonc"),
            );
        }
        (false, true) => {
            findings.push(
                Finding::error(
                    "execution-config-asymmetry",
                    "store.jsonc has execution_families but derive.jsonc does not",
                )
                .with_why(
                    "store consumer will idle because derive isn't producing execution events",
                )
                .with_help("add pipeline.execution_families to deploy/configs/derive.jsonc"),
            );
        }
        (false, false) => {
            findings.push(
                Finding::warning(
                    "execution-config-absent",
                    "neither derive.jsonc nor store.jsonc declare execution_families",
                )
                .with_why("execution pipeline is inactive")
                .with_help("add pipeline.execution_families to derive.jsonc and store.jsonc"),
            );
        }
    }

    // execute.jsonc must exist and declare execution_families
    if !execute_has {
        let execute_path = configs_dir.join("execute.jsonc");
        if !execute_path.is_file() {
            findings.push(
                Finding::error(
                    "execution-config-missing-execute",
                    "deploy/configs/execute.jsonc does not exist",
                )
                .with_why("execute binary requires its own config for venue adapter selection and pipeline validation")
                .with_help("create deploy/configs/execute.jsonc with venue and pipeline sections"),
            );
        } else {
            findings.push(
                Finding::error(
                    "execution-config-asymmetry",
                    "execute.jsonc exists but does not declare execution_families",
                )
                .with_why("execute binary will not know which execution families to consume")
                .with_help("add pipeline.execution_families to deploy/configs/execute.jsonc"),
            );
        }
    } else {
        findings.push(Finding::info(
            "execution-config-execute-present",
            "execute.jsonc declares execution_families",
        ));
    }

    // Venue config check in execute.jsonc
    let execute_path = configs_dir.join("execute.jsonc");
    if execute_path.is_file() {
        if let Ok(content) = std::fs::read_to_string(&execute_path) {
            if content.contains("\"venue\"") {
                findings.push(Finding::info(
                    "execution-venue-config-present",
                    "execute.jsonc declares venue adapter configuration",
                ));
            } else {
                findings.push(
                    Finding::warning(
                        "execution-venue-config-absent",
                        "execute.jsonc does not declare venue adapter configuration",
                    )
                    .with_why("execute binary uses config-driven venue adapter selection (S83) — absent venue section falls back to paper_simulator")
                    .with_help("add venue section with type field to deploy/configs/execute.jsonc"),
                );
            }
        }
    }

    CheckResult::from_findings("execution-config-drift", findings)
}

/// Check execution-contracts-drift: verify execution subjects, durables, and KV buckets exist in source.
/// Activated in S71 (execution implementation opened in S71).
fn check_execution_contracts_drift(evidence: &Evidence) -> CheckResult {
    let mut findings = Vec::new();

    let source = match &evidence.source {
        Some(s) => s,
        None => return CheckResult::skip("execution-contracts-drift", "source not scanned"),
    };

    // Check execution subjects exist in source
    for (subject, purpose) in EXECUTION_EXPECTED_SUBJECTS {
        let found = source.subjects.iter().any(|s| s.contains(subject));
        if found {
            findings.push(Finding::info(
                "execution-subject-present",
                format!("execution subject found: {subject}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "execution-subject-missing",
                    format!("execution subject not found in source: {subject}"),
                )
                .with_why(*purpose)
                .with_help("check internal/adapters/nats/natsexecution/registry.go for subject definitions"),
            );
        }
    }

    // Check execution durable consumers
    for (durable, purpose) in EXECUTION_EXPECTED_DURABLES {
        if source.durables.contains_key(*durable) {
            findings.push(Finding::info(
                "execution-durable-present",
                format!("execution durable consumer found: {durable}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "execution-durable-missing",
                    format!("execution durable consumer not found: {durable}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsexecution/registry.go for consumer spec",
                ),
            );
        }
    }

    // Check execution KV bucket names in source
    let nats_dir = evidence.project_root.join("internal/adapters/nats");
    for (bucket, purpose) in EXECUTION_EXPECTED_BUCKETS {
        let found = scan_dir_for_string(&nats_dir, bucket);
        if found {
            findings.push(Finding::info(
                "execution-bucket-present",
                format!("execution KV bucket found in source: {bucket}"),
            ));
        } else {
            findings.push(
                Finding::error(
                    "execution-bucket-missing",
                    format!("execution KV bucket name not found in source: {bucket}"),
                )
                .with_why(*purpose)
                .with_help(
                    "check internal/adapters/nats/natsexecution/kv_store.go for bucket definition",
                ),
            );
        }
    }

    CheckResult::from_findings("execution-contracts-drift", findings)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::{CheckStatus, Severity};
    use topology::compose::ComposeService;

    // ── Helper builders ─────────────────────────────────────────────

    fn make_service_config(name: &str) -> ServiceConfig {
        ServiceConfig {
            name: name.into(),
            nats_url: Some("nats://nats:4222".into()),
            ..Default::default()
        }
    }

    fn make_compose_topology() -> ComposeTopology {
        let mut services = HashMap::new();

        services.insert(
            "nats".into(),
            ComposeService {
                name: "nats".into(),
                image: Some("nats:2.10-alpine".into()),
                depends_on: vec![],
                profiles: vec![],
                ports: vec!["4222:4222".into()],
                internal_port: None,
            },
        );

        for svc in &["configctl", "gateway", "ingest", "derive", "store"] {
            services.insert(
                svc.to_string(),
                ComposeService {
                    name: svc.to_string(),
                    image: Some(format!("market-foundry/{svc}:dev")),
                    depends_on: vec!["nats".into()],
                    profiles: vec![],
                    ports: vec![],
                    internal_port: None,
                },
            );
        }

        ComposeTopology { services }
    }

    fn make_source_topology() -> SourceTopology {
        let mut streams = HashMap::new();
        streams.insert("CONFIGCTL_EVENTS".into(), vec!["configctl.events.>".into()]);
        streams.insert(
            "OBSERVATION_EVENTS".into(),
            vec!["observation.events.>".into()],
        );
        streams.insert("EVIDENCE_EVENTS".into(), vec!["evidence.events.>".into()]);
        streams.insert("SIGNAL_EVENTS".into(), vec!["signal.events.>".into()]);
        streams.insert("DECISION_EVENTS".into(), vec!["decision.events.>".into()]);
        streams.insert("STRATEGY_EVENTS".into(), vec!["strategy.events.>".into()]);
        streams.insert("RISK_EVENTS".into(), vec!["risk.events.>".into()]);
        streams.insert("EXECUTION_EVENTS".into(), vec!["execution.events.>".into()]);
        streams.insert(
            "EXECUTION_FILL_EVENTS".into(),
            vec!["execution.fill.>".into()],
        );
        // Aligned with the full CANONICAL_STREAMS const. REJECTION +
        // SESSION_LIFECYCLE were absent here (pre-existing fixture
        // drift — cargo test for raccoon-cli is not in make verify
        // nor CI, so it went unnoticed); INSIGHTS_EVENTS added in
        // H-8.a.
        streams.insert(
            "EXECUTION_REJECTION_EVENTS".into(),
            vec!["execution.rejection.>".into()],
        );
        streams.insert(
            "SESSION_LIFECYCLE_EVENTS".into(),
            vec!["execution.session.>".into()],
        );
        streams.insert("INSIGHTS_EVENTS".into(), vec!["insights.events.>".into()]);

        let mut durables = HashMap::new();
        durables.insert("derive-observation-v1".into(), "OBSERVATION_EVENTS".into());
        durables.insert("store-evidence-v1".into(), "EVIDENCE_EVENTS".into());
        durables.insert("store-signal-rsi".into(), "SIGNAL_EVENTS".into());
        durables.insert(
            "store-decision-rsi-oversold".into(),
            "DECISION_EVENTS".into(),
        );
        durables.insert(
            "store-strategy-mean-reversion-entry".into(),
            "STRATEGY_EVENTS".into(),
        );
        durables.insert("store-risk-position-exposure".into(), "RISK_EVENTS".into());
        durables.insert(
            "store-execution-paper-order".into(),
            "EXECUTION_EVENTS".into(),
        );

        let subjects = vec![
            "configctl.events.>".into(),
            "configctl.events.config.activated".into(),
            "observation.events.>".into(),
            "observation.events.trade.received".into(),
            "evidence.events.>".into(),
            "evidence.events.candle.sampled".into(),
            "signal.events.>".into(),
            "signal.events.rsi.generated".into(),
            "signal.query.rsi.latest".into(),
            "decision.events.>".into(),
            "decision.events.rsi_oversold.evaluated".into(),
            "decision.query.rsi_oversold.latest".into(),
            "strategy.events.>".into(),
            "strategy.events.mean_reversion_entry.resolved".into(),
            "strategy.query.mean_reversion_entry.latest".into(),
            "risk.events.>".into(),
            "risk.events.position_exposure.assessed".into(),
            "risk.query.position_exposure.latest".into(),
            "execution.events.>".into(),
            "execution.events.paper_order.submitted".into(),
            "execution.query.paper_order.latest".into(),
        ];

        SourceTopology {
            streams,
            durables,
            subjects,
        }
    }

    fn make_evidence() -> Evidence {
        let mut configs = HashMap::new();
        for svc in &["configctl", "gateway", "ingest", "derive", "store"] {
            configs.insert(svc.to_string(), make_service_config(svc));
        }

        let existing_cmd_dirs: HashSet<String> =
            APP_BINARIES.iter().map(|s| s.to_string()).collect();
        let existing_actor_scopes: HashSet<String> = ["ingest", "derive", "store"]
            .iter()
            .map(|s| s.to_string())
            .collect();

        let mut makefile_targets = HashSet::new();
        for t in &[
            "help", "tidy", "test", "build", "up-all", "down", "logs", "check", "verify", "smoke",
        ] {
            makefile_targets.insert(t.to_string());
        }

        let mut dev_doc_targets = HashSet::new();
        for t in &["check", "verify", "smoke", "up-all", "down", "logs"] {
            dev_doc_targets.insert(t.to_string());
        }

        Evidence {
            project_root: std::path::PathBuf::from("/tmp/market-foundry"),
            configs,
            compose: Some(make_compose_topology()),
            source: Some(make_source_topology()),
            makefile_targets,
            dev_doc_targets,
            existing_cmd_dirs,
            existing_actor_scopes,
            stale_references: Vec::new(),
        }
    }

    // ── config-compose-drift ────────────────────────────────────────

    #[test]
    fn config_compose_drift_passes_when_aligned() {
        let evidence = make_evidence();
        let result = check_config_compose_drift(&evidence);
        assert_eq!(result.status, CheckStatus::Pass);
    }

    #[test]
    fn config_compose_drift_skips_without_compose() {
        let mut evidence = make_evidence();
        evidence.compose = None;
        let result = check_config_compose_drift(&evidence);
        assert_eq!(result.status, CheckStatus::Skip);
    }

    #[test]
    fn config_compose_drift_warns_config_without_service() {
        let mut evidence = make_evidence();
        // Remove gateway from compose but keep its config
        let compose = evidence.compose.as_mut().unwrap();
        compose.services.remove("gateway");
        let result = check_config_compose_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("gateway") && f.message.contains("no matching compose")));
    }

    #[test]
    fn config_compose_drift_warns_compose_without_config() {
        let mut evidence = make_evidence();
        evidence.configs.remove("ingest");
        let result = check_config_compose_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("ingest") && f.message.contains("no deploy/configs")));
    }

    #[test]
    fn config_compose_drift_detects_nats_without_dependency() {
        let mut evidence = make_evidence();
        let compose = evidence.compose.as_mut().unwrap();
        let ingest = compose.services.get_mut("ingest").unwrap();
        ingest.depends_on.retain(|d| d != "nats");

        let result = check_config_compose_drift(&evidence);
        assert!(result.findings.iter().any(|f| f.severity == Severity::Error
            && f.message.contains("nats")
            && f.message.contains("ingest")));
    }

    #[test]
    fn config_compose_drift_detects_nats_url_inconsistency() {
        let mut evidence = make_evidence();
        evidence.configs.get_mut("derive").unwrap().nats_url =
            Some("nats://other-nats:4222".into());

        let result = check_config_compose_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.check == "nats-url-drift"));
    }

    // ── binary-compose-drift ────────────────────────────────────────

    #[test]
    fn binary_compose_drift_passes_when_aligned() {
        let evidence = make_evidence();
        let result = check_binary_compose_drift(&evidence);
        assert_eq!(result.status, CheckStatus::Pass);
    }

    #[test]
    fn binary_compose_drift_detects_missing_cmd_dir() {
        let mut evidence = make_evidence();
        evidence.existing_cmd_dirs.remove("derive");
        let result = check_binary_compose_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("cmd/derive")));
    }

    #[test]
    fn binary_compose_drift_detects_missing_compose_service() {
        let mut evidence = make_evidence();
        let compose = evidence.compose.as_mut().unwrap();
        compose.services.remove("store");
        let result = check_binary_compose_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("store") && f.message.contains("no matching compose")));
    }

    // ── naming-identity-drift ───────────────────────────────────────

    #[test]
    fn naming_identity_drift_passes_when_clean() {
        let evidence = make_evidence();
        let result = check_naming_identity_drift(&evidence);
        assert_eq!(result.status, CheckStatus::Pass);
        assert!(result.findings.is_empty());
    }

    #[test]
    fn naming_identity_drift_reports_stale_references() {
        let mut evidence = make_evidence();
        evidence.stale_references.push(StaleReference {
            file: "internal/foo/bar.go".into(),
            line_num: 42,
            pattern: "old service name 'validator'".into(),
            context: "// references validator service".into(),
        });
        let result = check_naming_identity_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.check == "stale-name" && f.message.contains("validator")));
    }

    #[test]
    fn naming_identity_drift_summarizes_many_refs() {
        let mut evidence = make_evidence();
        for i in 0..5 {
            evidence.stale_references.push(StaleReference {
                file: format!("internal/file{i}.go"),
                line_num: i + 1,
                pattern: "old service name 'validator'".into(),
                context: "validator reference".into(),
            });
        }
        let result = check_naming_identity_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("5 occurrences")));
    }

    // ── docs-reality-drift ──────────────────────────────────────────

    #[test]
    fn docs_reality_drift_detects_makefile_target_gap() {
        let mut evidence = make_evidence();
        evidence.dev_doc_targets.insert("nonexistent".into());
        let result = check_docs_reality_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("nonexistent")));
    }

    #[test]
    fn docs_reality_drift_passes_when_aligned() {
        let evidence = make_evidence();
        let result = check_docs_reality_drift(&evidence);
        // May have info-level findings for missing arch docs but should not fail
        let has_errors = result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error);
        assert!(!has_errors);
    }

    // ── actor-scope-drift ───────────────────────────────────────────

    #[test]
    fn actor_scope_drift_passes_when_aligned() {
        let evidence = make_evidence();
        let result = check_actor_scope_drift(&evidence);
        assert_eq!(result.status, CheckStatus::Pass);
    }

    #[test]
    fn actor_scope_drift_skips_when_empty() {
        let mut evidence = make_evidence();
        evidence.existing_actor_scopes.clear();
        let result = check_actor_scope_drift(&evidence);
        assert_eq!(result.status, CheckStatus::Skip);
    }

    #[test]
    fn actor_scope_drift_detects_missing_scope() {
        let mut evidence = make_evidence();
        evidence.existing_actor_scopes.remove("derive");
        let result = check_actor_scope_drift(&evidence);
        assert!(result.findings.iter().any(|f| f.severity == Severity::Error
            && f.message.contains("derive")
            && f.message.contains("actor scope")));
    }

    #[test]
    fn actor_scope_drift_flags_orphan_scope() {
        let mut evidence = make_evidence();
        evidence
            .existing_actor_scopes
            .insert("unknown_scope".into());
        let result = check_actor_scope_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("unknown_scope")));
    }

    // ── stream-registry-drift ───────────────────────────────────────

    #[test]
    fn stream_registry_drift_passes_when_aligned() {
        let evidence = make_evidence();
        let result = check_stream_registry_drift(&evidence);
        assert_eq!(result.status, CheckStatus::Pass);
    }

    #[test]
    fn stream_registry_drift_skips_without_source() {
        let mut evidence = make_evidence();
        evidence.source = None;
        let result = check_stream_registry_drift(&evidence);
        assert_eq!(result.status, CheckStatus::Skip);
    }

    #[test]
    fn stream_registry_drift_detects_missing_canonical_stream() {
        let mut evidence = make_evidence();
        let source = evidence.source.as_mut().unwrap();
        source.streams.remove("EVIDENCE_EVENTS");

        let result = check_stream_registry_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("EVIDENCE_EVENTS")));
    }

    #[test]
    fn stream_registry_drift_warns_non_canonical_stream() {
        let mut evidence = make_evidence();
        let source = evidence.source.as_mut().unwrap();
        source.streams.insert("DATA_PLANE_INGESTION".into(), vec![]);

        let result = check_stream_registry_drift(&evidence);
        assert!(result.findings.iter().any(
            |f| f.check == "non-canonical-stream" && f.message.contains("DATA_PLANE_INGESTION")
        ));
    }

    #[test]
    fn stream_registry_drift_detects_durable_orphan() {
        let mut evidence = make_evidence();
        let source = evidence.source.as_mut().unwrap();
        source
            .durables
            .insert("orphan-durable".into(), "NONEXISTENT_STREAM".into());

        let result = check_stream_registry_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("NONEXISTENT_STREAM")));
    }

    #[test]
    fn stream_registry_drift_warns_unmatched_subject_pattern() {
        let mut evidence = make_evidence();
        let source = evidence.source.as_mut().unwrap();
        source
            .streams
            .insert("EVIDENCE_EVENTS".into(), vec!["unmatched.pattern.>".into()]);

        let result = check_stream_registry_drift(&evidence);
        assert!(result
            .findings
            .iter()
            .any(|f| f.check == "stream-subject-drift" && f.message.contains("unmatched.pattern")));
    }

    // ── Integration ─────────────────────────────────────────────────

    #[test]
    fn full_report_passes_when_aligned() {
        let evidence = make_evidence();
        let mut report = Report::new("drift-detect");
        report.add(check_config_compose_drift(&evidence));
        report.add(check_binary_compose_drift(&evidence));
        report.add(check_naming_identity_drift(&evidence));
        report.add(check_actor_scope_drift(&evidence));
        report.add(check_stream_registry_drift(&evidence));
        assert!(report.passed());
    }

    #[test]
    fn full_report_fails_on_missing_stream() {
        let mut evidence = make_evidence();
        let source = evidence.source.as_mut().unwrap();
        source.streams.remove("OBSERVATION_EVENTS");

        let mut report = Report::new("drift-detect");
        report.add(check_stream_registry_drift(&evidence));
        assert!(!report.passed());
    }
}
