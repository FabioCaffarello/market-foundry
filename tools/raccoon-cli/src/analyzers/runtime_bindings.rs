use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};
use std::collections::HashSet;
use std::path::Path;

mod configs;
mod source;

#[allow(unused_imports)]
pub use configs::ServiceConfig as BindingDefinition;
pub use source::RuntimeBindingSource;

// ── Architecture constants ──────────────────────────────────────────

/// The five JetStream streams in the market-foundry pipeline.
const EXPECTED_STREAMS: &[(&str, &str)] = &[
    (
        "CONFIGCTL_EVENTS",
        "carries config lifecycle events (activated, deactivated)",
    ),
    (
        "OBSERVATION_EVENTS",
        "carries raw market observations from ingest",
    ),
    (
        "EVIDENCE_EVENTS",
        "carries derived evidence (candles, trade bursts, volume) from derive",
    ),
    (
        "SIGNAL_EVENTS",
        "carries derived signals (RSI, MACD) from derive",
    ),
    (
        "DECISION_EVENTS",
        "carries derived decisions (RSI oversold) from derive",
    ),
    (
        "STRATEGY_EVENTS",
        "carries resolved strategies (mean reversion entry) from derive",
    ),
];

/// Durable consumers and the streams they must be bound to.
const EXPECTED_DURABLES: &[(&str, &str, &str)] = &[
    (
        "derive-observation",
        "OBSERVATION_EVENTS",
        "derive consumes raw observations to produce evidence",
    ),
    (
        "store-candle",
        "EVIDENCE_EVENTS",
        "store consumes candle evidence for projection",
    ),
    (
        "store-trade-burst",
        "EVIDENCE_EVENTS",
        "store consumes trade burst evidence for projection",
    ),
    (
        "store-volume",
        "EVIDENCE_EVENTS",
        "store consumes volume evidence for projection",
    ),
    (
        "store-signal-rsi",
        "SIGNAL_EVENTS",
        "store consumes RSI signal events for projection",
    ),
    (
        "store-decision-rsi-oversold",
        "DECISION_EVENTS",
        "store consumes RSI oversold decision events for projection",
    ),
    (
        "store-strategy-mean-reversion-entry",
        "STRATEGY_EVENTS",
        "store consumes mean reversion entry strategy events for projection",
    ),
];

/// Expected query/request-reply subjects.
const EXPECTED_QUERY_SUBJECTS: &[(&str, &str)] = &[
    (
        "evidence.query.candle.latest",
        "store serves latest candle queries from gateway",
    ),
    (
        "evidence.query.candle.history",
        "store serves candle history queries from gateway",
    ),
    (
        "evidence.query.tradeburst.latest",
        "store serves latest trade burst queries from gateway",
    ),
    (
        "evidence.query.volume.latest",
        "store serves latest volume queries from gateway",
    ),
    (
        "signal.query.rsi.latest",
        "store serves latest RSI signal queries from gateway",
    ),
    (
        "decision.query.rsi_oversold.latest",
        "store serves latest RSI oversold decision queries from gateway",
    ),
    (
        "strategy.query.mean_reversion_entry.latest",
        "store serves latest mean reversion entry strategy queries from gateway",
    ),
];

/// Expected service binaries and their required adapter capabilities.
const EXPECTED_SERVICE_ADAPTERS: &[(&str, &[&str])] = &[
    ("ingest", &["publisher", "websocket", "binding_watcher"]),
    ("derive", &["consumer", "publisher", "binding_watcher"]),
    ("store", &["consumer", "projection", "responder"]),
];

// ── Main analysis entry point ───────────────────────────────────────

pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("runtime-bindings");

    // Phase 1: Scan Go source for stream/subject/durable patterns
    let internal_dir = project_root.join("internal");
    if !internal_dir.is_dir() {
        report.add(CheckResult::from_findings(
            "internal-dir",
            vec![
                Finding::error("internal-dir", "internal/ directory not found")
                    .with_why(
                        "runtime-bindings scans Go source for NATS stream and subject definitions",
                    )
                    .with_help("run `raccoon-cli doctor` to verify project structure first"),
            ],
        ));
        return Ok(report);
    }

    let src = match source::scan_runtime_bindings(&internal_dir) {
        Ok(s) => s,
        Err(e) => {
            report.add(CheckResult::from_findings(
                "source-scan",
                vec![Finding::error("source", format!("failed to scan: {e}"))],
            ));
            return Ok(report);
        }
    };

    // Phase 2: Scan deploy configs
    let configs_dir = project_root.join("deploy/configs");
    let service_configs = if configs_dir.is_dir() {
        match configs::scan_service_configs(&configs_dir) {
            Ok(c) => c,
            Err(_) => Vec::new(),
        }
    } else {
        Vec::new()
    };

    // Phase 3: Run all checks
    report.add(check_stream_ownership(&src));
    report.add(check_consumer_binding(&src));
    report.add(check_query_routing(&src));
    report.add(check_config_source_alignment(&service_configs));
    report.add(check_adapter_presence(&src));
    report.add(check_adapter_files(&src));
    report.add(check_lifecycle_events(&src));
    report.add(check_cross_config_family_consistency(&service_configs));

    Ok(report)
}

// ── Check 1: Stream ownership ───────────────────────────────────────

/// Verify each expected JetStream stream is declared in source.
fn check_stream_ownership(src: &RuntimeBindingSource) -> CheckResult {
    let mut findings = Vec::new();

    for (stream, purpose) in EXPECTED_STREAMS {
        if src.stream_subjects.contains_key(*stream) {
            let subjects = &src.stream_subjects[*stream];
            findings.push(Finding::info(
                "stream-present",
                format!(
                    "stream {stream} found with {} subject pattern(s)",
                    subjects.len()
                ),
            ));
        } else {
            findings.push(
                Finding::error(
                    "stream-missing",
                    format!("stream {stream} not found in source"),
                )
                .with_why(*purpose)
                .with_help("check internal/adapters/nats/ registry files"),
            );
        }
    }

    // Check for unexpected streams (not necessarily an error, but worth noting)
    let expected: HashSet<&str> = EXPECTED_STREAMS.iter().map(|(s, _)| *s).collect();
    for stream in src.stream_subjects.keys() {
        if !expected.contains(stream.as_str()) {
            findings.push(Finding::info(
                "extra-stream",
                format!("additional stream found: {stream}"),
            ));
        }
    }

    CheckResult::from_findings("stream-ownership", findings)
}

// ── Check 2: Consumer binding ───────────────────────────────────────

/// Verify each durable consumer is declared and bound to the correct stream.
fn check_consumer_binding(src: &RuntimeBindingSource) -> CheckResult {
    let mut findings = Vec::new();

    for (durable, expected_stream, purpose) in EXPECTED_DURABLES {
        match src.durable_consumers.get(*durable) {
            Some(actual_stream) => {
                if actual_stream == *expected_stream {
                    findings.push(Finding::info(
                        "durable-correct",
                        format!("durable {durable} correctly bound to {expected_stream}"),
                    ));
                } else {
                    findings.push(
                        Finding::error(
                            "durable-wrong-stream",
                            format!(
                                "durable {durable} bound to '{actual_stream}' instead of '{expected_stream}'"
                            ),
                        )
                        .with_why(*purpose),
                    );
                }
            }
            None => {
                findings.push(
                    Finding::error(
                        "durable-missing",
                        format!("durable consumer {durable} not found in source"),
                    )
                    .with_why(*purpose)
                    .with_help("check the relevant registry file in internal/adapters/nats/"),
                );
            }
        }
    }

    // Warn about unexpected durables
    let expected: HashSet<&str> = EXPECTED_DURABLES.iter().map(|(d, _, _)| *d).collect();
    for durable in src.durable_consumers.keys() {
        if !expected.contains(durable.as_str()) {
            findings.push(Finding::info(
                "extra-durable",
                format!("additional durable consumer found: {durable}"),
            ));
        }
    }

    CheckResult::from_findings("consumer-binding", findings)
}

// ── Check 3: Query routing ──────────────────────────────────────────

/// Verify request/reply subjects are present in source.
fn check_query_routing(src: &RuntimeBindingSource) -> CheckResult {
    let mut findings = Vec::new();

    for (subject, purpose) in EXPECTED_QUERY_SUBJECTS {
        if src.query_subjects.contains(*subject) {
            findings.push(Finding::info(
                "query-present",
                format!("query subject {subject} found in source"),
            ));
        } else {
            // Check with prefix matching
            let found = src
                .query_subjects
                .iter()
                .any(|q| q.starts_with(&subject[..subject.rfind('.').unwrap_or(subject.len())]));
            if found {
                findings.push(Finding::info(
                    "query-pattern",
                    format!("query subject pattern matching {subject} found in source"),
                ));
            } else {
                findings.push(
                    Finding::error(
                        "query-missing",
                        format!("query subject {subject} not found in source"),
                    )
                    .with_why(*purpose)
                    .with_help(
                        "check internal/adapters/nats/ for responder/gateway implementations",
                    ),
                );
            }
        }
    }

    // Report any additional query subjects
    let expected: HashSet<&str> = EXPECTED_QUERY_SUBJECTS.iter().map(|(s, _)| *s).collect();
    for subject in &src.query_subjects {
        if !expected.contains(subject.as_str()) {
            findings.push(Finding::info(
                "extra-query",
                format!("additional query subject found: {subject}"),
            ));
        }
    }

    CheckResult::from_findings("query-routing", findings)
}

// ── Check 4: Config-source alignment ────────────────────────────────

/// Verify deploy configs exist for expected services and reference NATS.
fn check_config_source_alignment(service_configs: &[configs::ServiceConfig]) -> CheckResult {
    let mut findings = Vec::new();

    let expected_services = ["ingest", "derive", "store"];
    let configured: HashSet<&str> = service_configs.iter().map(|c| c.service.as_str()).collect();

    for svc in &expected_services {
        if configured.contains(svc) {
            let cfg = service_configs.iter().find(|c| c.service == *svc).unwrap();
            if cfg.nats_url.is_some() {
                findings.push(Finding::info(
                    "config-nats",
                    format!("service {svc} has NATS configuration"),
                ));
            } else {
                findings.push(
                    Finding::warning(
                        "config-no-nats",
                        format!("service {svc} config exists but has no NATS URL"),
                    )
                    .with_why(format!(
                        "{svc} needs NATS connectivity for stream publishing/consuming"
                    ))
                    .with_help(format!("add nats.url to deploy/configs/{svc}.jsonc")),
                );
            }
        } else {
            findings.push(
                Finding::warning(
                    "config-missing",
                    format!("no deploy config found for service {svc}"),
                )
                .with_help(format!("create deploy/configs/{svc}.jsonc")),
            );
        }
    }

    CheckResult::from_findings("config-source-alignment", findings)
}

// ── Check 5: Adapter presence ───────────────────────────────────────

/// Verify each service binary has the required actor/adapter files.
fn check_adapter_presence(src: &RuntimeBindingSource) -> CheckResult {
    let mut findings = Vec::new();

    for (service, required_adapters) in EXPECTED_SERVICE_ADAPTERS {
        match src.service_adapters.get(*service) {
            Some(actual) => {
                for adapter in *required_adapters {
                    if actual.contains(*adapter) {
                        findings.push(Finding::info(
                            "adapter-present",
                            format!("{service} has {adapter} adapter"),
                        ));
                    } else {
                        findings.push(
                            Finding::error(
                                "adapter-missing",
                                format!("{service} is missing required {adapter} adapter"),
                            )
                            .with_help(format!(
                                "check internal/actors/scopes/{service}/ for {adapter} actor file"
                            )),
                        );
                    }
                }
            }
            None => {
                findings.push(
                    Finding::error(
                        "scope-missing",
                        format!("actor scope directory for {service} not found"),
                    )
                    .with_why(format!(
                        "{service} needs an actor scope to participate in the pipeline"
                    ))
                    .with_help(format!(
                        "create internal/actors/scopes/{service}/ with required actors"
                    )),
                );
            }
        }
    }

    CheckResult::from_findings("adapter-presence", findings)
}

// ── Check 6: NATS adapter files ─────────────────────────────────────

/// Verify key NATS adapter implementation files exist.
fn check_adapter_files(src: &RuntimeBindingSource) -> CheckResult {
    let mut findings = Vec::new();

    let checks = [
        (
            src.has_observation_publisher,
            "observation_publisher",
            "ingest publishes raw trades to OBSERVATION_EVENTS",
        ),
        (
            src.has_observation_consumer,
            "observation_consumer",
            "derive consumes observations from OBSERVATION_EVENTS",
        ),
        (
            src.has_evidence_publisher,
            "evidence_publisher",
            "derive publishes sampled candles to EVIDENCE_EVENTS",
        ),
        (
            src.has_evidence_consumer,
            "evidence_consumer",
            "store consumes candles from EVIDENCE_EVENTS",
        ),
        (
            src.has_evidence_gateway,
            "evidence_gateway",
            "gateway sends NATS requests to store for candle queries",
        ),
        (
            src.has_candle_kv_store,
            "candle_kv_store",
            "store persists candle state in NATS KV",
        ),
        (
            src.has_binding_watcher,
            "binding_event_consumer",
            "services watch for config binding changes to activate/deactivate streams",
        ),
        (
            src.has_signal_publisher,
            "signal_publisher",
            "derive publishes signal events to SIGNAL_EVENTS",
        ),
        (
            src.has_signal_consumer,
            "signal_consumer",
            "store consumes signal events from SIGNAL_EVENTS",
        ),
        (
            src.has_signal_gateway,
            "signal_gateway",
            "gateway sends NATS requests to store for signal queries",
        ),
        (
            src.has_signal_kv_store,
            "signal_kv_store",
            "store persists signal state in NATS KV",
        ),
        (
            src.has_signal_registry,
            "signal_registry",
            "defines SIGNAL_EVENTS stream, consumer, and query specs",
        ),
        (
            src.has_decision_publisher,
            "decision_publisher",
            "derive publishes decision events to DECISION_EVENTS",
        ),
        (
            src.has_decision_consumer,
            "decision_consumer",
            "store consumes decision events from DECISION_EVENTS",
        ),
        (
            src.has_decision_gateway,
            "decision_gateway",
            "gateway sends NATS requests to store for decision queries",
        ),
        (
            src.has_decision_kv_store,
            "decision_kv_store",
            "store persists decision state in NATS KV",
        ),
        (
            src.has_decision_registry,
            "decision_registry",
            "defines DECISION_EVENTS stream, consumer, and query specs",
        ),
        (
            src.has_strategy_publisher,
            "strategy_publisher",
            "derive publishes strategy events to STRATEGY_EVENTS",
        ),
        (
            src.has_strategy_consumer,
            "strategy_consumer",
            "store consumes strategy events from STRATEGY_EVENTS",
        ),
        (
            src.has_strategy_gateway,
            "strategy_gateway",
            "gateway sends NATS requests to store for strategy queries",
        ),
        (
            src.has_strategy_kv_store,
            "strategy_kv_store",
            "store persists strategy state in NATS KV",
        ),
        (
            src.has_strategy_registry,
            "strategy_registry",
            "defines STRATEGY_EVENTS stream, consumer, and query specs",
        ),
    ];

    for (present, name, purpose) in &checks {
        if *present {
            findings.push(Finding::info("adapter-file", format!("{name}.go present")));
        } else {
            findings.push(
                Finding::warning(
                    "adapter-file-missing",
                    format!("{name}.go not found in internal/adapters/nats/"),
                )
                .with_why(*purpose),
            );
        }
    }

    CheckResult::from_findings("adapter-files", findings)
}

// ── Check 7: Lifecycle events ───────────────────────────────────────

/// Verify config lifecycle events are declared in source.
fn check_lifecycle_events(src: &RuntimeBindingSource) -> CheckResult {
    let mut findings = Vec::new();

    let expected = [
        (
            "config.activated",
            "triggers binding activation across services",
        ),
        (
            "config.deactivated",
            "triggers binding deactivation across services",
        ),
    ];

    for (event, purpose) in &expected {
        if src.lifecycle_events.contains(*event) {
            findings.push(Finding::info(
                "lifecycle-event",
                format!("lifecycle event '{event}' found"),
            ));
        } else {
            findings.push(
                Finding::warning(
                    "lifecycle-event-missing",
                    format!("lifecycle event '{event}' not found in source"),
                )
                .with_why(*purpose),
            );
        }
    }

    CheckResult::from_findings("lifecycle-events", findings)
}

// ── Check 8: Cross-config family consistency ────────────────────────

/// Verify that derive and store configs enable the same families.
/// A family enabled in derive but missing in store (or vice-versa) means
/// events are produced but never projected, or projections wait for
/// events that never arrive.
fn check_cross_config_family_consistency(
    service_configs: &[configs::ServiceConfig],
) -> CheckResult {
    let mut findings = Vec::new();

    let derive_cfg = service_configs.iter().find(|c| c.service == "derive");
    let store_cfg = service_configs.iter().find(|c| c.service == "store");

    let (derive, store) = match (derive_cfg, store_cfg) {
        (Some(d), Some(s)) => (d, s),
        _ => {
            findings.push(Finding::info(
                "cross-config-skip",
                "skipping cross-config check: derive or store config not found".to_string(),
            ));
            return CheckResult::from_findings("cross-config-family-consistency", findings);
        }
    };

    // Evidence families — empty list means "all enabled" (backward compatible).
    // Only flag a mismatch when both services have explicit lists.
    if !derive.families.is_empty() && !store.families.is_empty() {
        let derive_set: HashSet<&str> = derive.families.iter().map(|s| s.as_str()).collect();
        let store_set: HashSet<&str> = store.families.iter().map(|s| s.as_str()).collect();

        for f in derive_set.difference(&store_set) {
            findings.push(
                Finding::error(
                    "evidence-derive-only",
                    format!("evidence family '{f}' enabled in derive but not in store"),
                )
                .with_why("derive will publish events that store never projects")
                .with_help(format!(
                    "add '{f}' to pipeline.families in deploy/configs/store.jsonc"
                )),
            );
        }
        for f in store_set.difference(&derive_set) {
            findings.push(
                Finding::error(
                    "evidence-store-only",
                    format!("evidence family '{f}' enabled in store but not in derive"),
                )
                .with_why("store expects events that derive never produces")
                .with_help(format!(
                    "add '{f}' to pipeline.families in deploy/configs/derive.jsonc"
                )),
            );
        }
        if derive_set == store_set {
            findings.push(Finding::info(
                "evidence-consistent",
                "evidence families are consistent between derive and store".to_string(),
            ));
        }
    } else {
        findings.push(Finding::info(
            "evidence-default",
            "evidence families use default (all enabled) in at least one service".to_string(),
        ));
    }

    // Signal families — both are opt-in; empty = none.
    let derive_sig: HashSet<&str> = derive.signal_families.iter().map(|s| s.as_str()).collect();
    let store_sig: HashSet<&str> = store.signal_families.iter().map(|s| s.as_str()).collect();

    for f in derive_sig.difference(&store_sig) {
        findings.push(
            Finding::error(
                "signal-derive-only",
                format!("signal family '{f}' enabled in derive but not in store"),
            )
            .with_why("derive will publish signal events that store never projects")
            .with_help(format!(
                "add '{f}' to pipeline.signal_families in deploy/configs/store.jsonc"
            )),
        );
    }
    for f in store_sig.difference(&derive_sig) {
        findings.push(
            Finding::error(
                "signal-store-only",
                format!("signal family '{f}' enabled in store but not in derive"),
            )
            .with_why("store expects signal events that derive never produces")
            .with_help(format!(
                "add '{f}' to pipeline.signal_families in deploy/configs/derive.jsonc"
            )),
        );
    }
    if derive_sig == store_sig && !derive_sig.is_empty() {
        findings.push(Finding::info(
            "signal-consistent",
            "signal families are consistent between derive and store".to_string(),
        ));
    }

    // Decision families — both are opt-in; empty = none.
    let derive_dec: HashSet<&str> = derive
        .decision_families
        .iter()
        .map(|s| s.as_str())
        .collect();
    let store_dec: HashSet<&str> = store.decision_families.iter().map(|s| s.as_str()).collect();

    for f in derive_dec.difference(&store_dec) {
        findings.push(
            Finding::error(
                "decision-derive-only",
                format!("decision family '{f}' enabled in derive but not in store"),
            )
            .with_why("derive will publish decision events that store never projects")
            .with_help(format!(
                "add '{f}' to pipeline.decision_families in deploy/configs/store.jsonc"
            )),
        );
    }
    for f in store_dec.difference(&derive_dec) {
        findings.push(
            Finding::error(
                "decision-store-only",
                format!("decision family '{f}' enabled in store but not in derive"),
            )
            .with_why("store expects decision events that derive never produces")
            .with_help(format!(
                "add '{f}' to pipeline.decision_families in deploy/configs/derive.jsonc"
            )),
        );
    }
    if derive_dec == store_dec && !derive_dec.is_empty() {
        findings.push(Finding::info(
            "decision-consistent",
            "decision families are consistent between derive and store".to_string(),
        ));
    }

    // Strategy families — both are opt-in; empty = none.
    let derive_strat: HashSet<&str> = derive
        .strategy_families
        .iter()
        .map(|s| s.as_str())
        .collect();
    let store_strat: HashSet<&str> = store.strategy_families.iter().map(|s| s.as_str()).collect();

    for f in derive_strat.difference(&store_strat) {
        findings.push(
            Finding::error(
                "strategy-derive-only",
                format!("strategy family '{f}' enabled in derive but not in store"),
            )
            .with_why("derive will publish strategy events that store never projects")
            .with_help(format!(
                "add '{f}' to pipeline.strategy_families in deploy/configs/store.jsonc"
            )),
        );
    }
    for f in store_strat.difference(&derive_strat) {
        findings.push(
            Finding::error(
                "strategy-store-only",
                format!("strategy family '{f}' enabled in store but not in derive"),
            )
            .with_why("store expects strategy events that derive never produces")
            .with_help(format!(
                "add '{f}' to pipeline.strategy_families in deploy/configs/derive.jsonc"
            )),
        );
    }
    if derive_strat == store_strat && !derive_strat.is_empty() {
        findings.push(Finding::info(
            "strategy-consistent",
            "strategy families are consistent between derive and store".to_string(),
        ));
    }

    CheckResult::from_findings("cross-config-family-consistency", findings)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;

    fn make_source() -> RuntimeBindingSource {
        let mut stream_subjects = HashMap::new();
        stream_subjects.insert("CONFIGCTL_EVENTS".into(), vec!["configctl.events.>".into()]);
        stream_subjects.insert(
            "OBSERVATION_EVENTS".into(),
            vec!["observation.events.>".into()],
        );
        stream_subjects.insert("EVIDENCE_EVENTS".into(), vec!["evidence.events.>".into()]);
        stream_subjects.insert("SIGNAL_EVENTS".into(), vec!["signal.events.>".into()]);
        stream_subjects.insert("DECISION_EVENTS".into(), vec!["decision.events.>".into()]);
        stream_subjects.insert("STRATEGY_EVENTS".into(), vec!["strategy.events.>".into()]);

        let mut durable_consumers = HashMap::new();
        durable_consumers.insert("derive-observation".into(), "OBSERVATION_EVENTS".into());
        durable_consumers.insert("store-candle".into(), "EVIDENCE_EVENTS".into());
        durable_consumers.insert("store-trade-burst".into(), "EVIDENCE_EVENTS".into());
        durable_consumers.insert("store-volume".into(), "EVIDENCE_EVENTS".into());
        durable_consumers.insert("store-signal-rsi".into(), "SIGNAL_EVENTS".into());
        durable_consumers.insert(
            "store-decision-rsi-oversold".into(),
            "DECISION_EVENTS".into(),
        );
        durable_consumers.insert(
            "store-strategy-mean-reversion-entry".into(),
            "STRATEGY_EVENTS".into(),
        );

        let mut query_subjects = HashSet::new();
        query_subjects.insert("evidence.query.candle.latest".into());
        query_subjects.insert("evidence.query.candle.history".into());
        query_subjects.insert("evidence.query.tradeburst.latest".into());
        query_subjects.insert("evidence.query.volume.latest".into());
        query_subjects.insert("signal.query.rsi.latest".into());
        query_subjects.insert("decision.query.rsi_oversold.latest".into());
        query_subjects.insert("strategy.query.mean_reversion_entry.latest".into());

        let mut publish_subjects = HashSet::new();
        publish_subjects.insert("observation.events.market.trade.binancef".into());
        publish_subjects.insert("evidence.events.candle.sampled.BTCUSDT.1m".into());

        let mut service_adapters = HashMap::new();
        let mut ingest_adapters = HashSet::new();
        ingest_adapters.insert("publisher".into());
        ingest_adapters.insert("websocket".into());
        ingest_adapters.insert("binding_watcher".into());
        service_adapters.insert("ingest".into(), ingest_adapters);

        let mut derive_adapters = HashSet::new();
        derive_adapters.insert("consumer".into());
        derive_adapters.insert("publisher".into());
        derive_adapters.insert("binding_watcher".into());
        service_adapters.insert("derive".into(), derive_adapters);

        let mut store_adapters = HashSet::new();
        store_adapters.insert("consumer".into());
        store_adapters.insert("projection".into());
        store_adapters.insert("responder".into());
        service_adapters.insert("store".into(), store_adapters);

        let mut lifecycle_events = HashSet::new();
        lifecycle_events.insert("config.activated".into());
        lifecycle_events.insert("config.deactivated".into());

        RuntimeBindingSource {
            stream_subjects,
            durable_consumers,
            query_subjects,
            publish_subjects,
            service_adapters,
            lifecycle_events,
            has_observation_publisher: true,
            has_observation_consumer: true,
            has_evidence_publisher: true,
            has_evidence_consumer: true,
            has_evidence_gateway: true,
            has_candle_kv_store: true,
            has_binding_watcher: true,
            has_signal_publisher: true,
            has_signal_consumer: true,
            has_signal_gateway: true,
            has_signal_kv_store: true,
            has_signal_registry: true,
            has_decision_publisher: true,
            has_decision_consumer: true,
            has_decision_gateway: true,
            has_decision_kv_store: true,
            has_decision_registry: true,
            has_strategy_publisher: true,
            has_strategy_consumer: true,
            has_strategy_gateway: true,
            has_strategy_kv_store: true,
            has_strategy_registry: true,
            has_risk_publisher: false,
            has_risk_consumer: false,
            has_risk_gateway: false,
            has_risk_kv_store: false,
            has_risk_registry: false,
        }
    }

    // ── check_stream_ownership ────────────────────────────────────────

    #[test]
    fn stream_ownership_passes_all_present() {
        let src = make_source();
        let result = check_stream_ownership(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn stream_ownership_fails_missing_observation() {
        let mut src = make_source();
        src.stream_subjects.remove("OBSERVATION_EVENTS");
        let result = check_stream_ownership(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    #[test]
    fn stream_ownership_fails_missing_evidence() {
        let mut src = make_source();
        src.stream_subjects.remove("EVIDENCE_EVENTS");
        let result = check_stream_ownership(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    #[test]
    fn stream_ownership_fails_missing_configctl() {
        let mut src = make_source();
        src.stream_subjects.remove("CONFIGCTL_EVENTS");
        let result = check_stream_ownership(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    // ── check_consumer_binding ────────────────────────────────────────

    #[test]
    fn consumer_binding_passes_correct() {
        let src = make_source();
        let result = check_consumer_binding(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn consumer_binding_fails_missing_derive() {
        let mut src = make_source();
        src.durable_consumers.remove("derive-observation");
        let result = check_consumer_binding(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    #[test]
    fn consumer_binding_fails_missing_store() {
        let mut src = make_source();
        src.durable_consumers.remove("store-candle");
        let result = check_consumer_binding(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    #[test]
    fn consumer_binding_fails_wrong_stream() {
        let mut src = make_source();
        src.durable_consumers
            .insert("derive-observation".into(), "WRONG_STREAM".into());
        let result = check_consumer_binding(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    // ── check_query_routing ───────────────────────────────────────────

    #[test]
    fn query_routing_passes_present() {
        let src = make_source();
        let result = check_query_routing(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn query_routing_fails_missing() {
        let mut src = make_source();
        src.query_subjects.clear();
        let result = check_query_routing(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    // ── check_config_source_alignment ─────────────────────────────────

    fn make_service_config(service: &str, nats: bool) -> configs::ServiceConfig {
        configs::ServiceConfig {
            service: service.into(),
            nats_url: if nats {
                Some("nats://nats:4222".into())
            } else {
                None
            },
            source_file: format!("deploy/configs/{service}.jsonc"),
            families: Vec::new(),
            signal_families: Vec::new(),
            decision_families: Vec::new(),
            strategy_families: Vec::new(),
        }
    }

    fn config_alignment_passes_all_present() {
        let configs = vec![
            make_service_config("ingest", true),
            make_service_config("derive", true),
            make_service_config("store", true),
        ];
        let result = check_config_source_alignment(&configs);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn config_alignment_warns_missing_service() {
        let configs = vec![make_service_config("ingest", true)];
        let result = check_config_source_alignment(&configs);
        // Missing configs are warnings, not errors
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("no deploy config")));
    }

    #[test]
    fn config_alignment_warns_no_nats_url() {
        let configs = vec![
            make_service_config("ingest", false),
            make_service_config("derive", true),
            make_service_config("store", true),
        ];
        let result = check_config_source_alignment(&configs);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("no NATS URL")));
    }

    // ── check_adapter_presence ────────────────────────────────────────

    #[test]
    fn adapter_presence_passes_all_present() {
        let src = make_source();
        let result = check_adapter_presence(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn adapter_presence_fails_missing_scope() {
        let mut src = make_source();
        src.service_adapters.remove("ingest");
        let result = check_adapter_presence(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    #[test]
    fn adapter_presence_fails_missing_adapter() {
        let mut src = make_source();
        if let Some(adapters) = src.service_adapters.get_mut("derive") {
            adapters.remove("consumer");
        }
        let result = check_adapter_presence(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    // ── check_adapter_files ───────────────────────────────────────────

    #[test]
    fn adapter_files_passes_all_present() {
        let src = make_source();
        let result = check_adapter_files(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn adapter_files_warns_missing() {
        let mut src = make_source();
        src.has_observation_publisher = false;
        let result = check_adapter_files(&src);
        // Missing adapter files are warnings
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("observation_publisher")));
    }

    // ── check_lifecycle_events ────────────────────────────────────────

    #[test]
    fn lifecycle_events_passes_present() {
        let src = make_source();
        let result = check_lifecycle_events(&src);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn lifecycle_events_warns_missing() {
        let mut src = make_source();
        src.lifecycle_events.clear();
        let result = check_lifecycle_events(&src);
        // Only warnings, still passes
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("config.activated")));
    }

    // ── check_cross_config_family_consistency ──────────────────────────

    #[test]
    fn cross_config_passes_when_families_match() {
        let mut derive = make_service_config("derive", true);
        derive.families = vec!["candle".into(), "volume".into()];
        derive.signal_families = vec!["rsi".into()];
        derive.decision_families = vec!["rsi_oversold".into()];

        let mut store = make_service_config("store", true);
        store.families = vec!["candle".into(), "volume".into()];
        store.signal_families = vec!["rsi".into()];
        store.decision_families = vec!["rsi_oversold".into()];

        let result = check_cross_config_family_consistency(&[derive, store]);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn cross_config_fails_evidence_derive_only() {
        let mut derive = make_service_config("derive", true);
        derive.families = vec!["candle".into(), "volume".into()];

        let mut store = make_service_config("store", true);
        store.families = vec!["candle".into()]; // volume missing

        let result = check_cross_config_family_consistency(&[derive, store]);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
        assert!(
            result
                .findings
                .iter()
                .any(|f| f.message.contains("volume")
                    && f.message.contains("derive but not in store"))
        );
    }

    #[test]
    fn cross_config_fails_signal_mismatch() {
        let mut derive = make_service_config("derive", true);
        derive.signal_families = vec!["rsi".into()];

        let store = make_service_config("store", true);
        // store has no signal families

        let result = check_cross_config_family_consistency(&[derive, store]);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("rsi") && f.message.contains("derive but not in store")));
    }

    #[test]
    fn cross_config_fails_decision_mismatch() {
        let mut derive = make_service_config("derive", true);
        derive.decision_families = vec!["rsi_oversold".into()];

        let store = make_service_config("store", true);

        let result = check_cross_config_family_consistency(&[derive, store]);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    #[test]
    fn cross_config_skips_without_both_services() {
        let derive = make_service_config("derive", true);
        let result = check_cross_config_family_consistency(&[derive]);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    // ── analyze integration ───────────────────────────────────────────

    #[test]
    fn analyze_fails_without_internal_dir() {
        let dir = tempfile::tempdir().unwrap();
        let report = analyze(dir.path()).unwrap();
        assert!(!report.passed());
    }

    #[test]
    fn analyze_succeeds_on_empty_internal() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::create_dir_all(dir.path().join("internal")).unwrap();
        let report = analyze(dir.path()).unwrap();
        assert_eq!(report.title, "runtime-bindings");
    }

    #[test]
    fn analyze_reports_have_correct_title() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::create_dir_all(dir.path().join("internal")).unwrap();
        let report = analyze(dir.path()).unwrap();
        assert_eq!(report.title, "runtime-bindings");
        // Should have multiple check results
        assert!(!report.checks.is_empty());
    }
}
