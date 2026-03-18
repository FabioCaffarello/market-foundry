use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};
use std::collections::HashMap;
use std::path::Path;

pub mod compose;
pub mod configs;
pub mod source;

pub use compose::ComposeTopology;
pub use configs::ServiceConfig;
pub use source::SourceTopology;

// ── Discovered topology ─────────────────────────────────────────────

/// A stage in the market-foundry pipeline (nats, configctl, gateway, ingest, derive, store).
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
#[allow(dead_code)]
pub enum Stage {
    Nats,
    ConfigCtl,
    Gateway,
    Ingest,
    Derive,
    Store,
}

impl std::fmt::Display for Stage {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Stage::Nats => write!(f, "nats"),
            Stage::ConfigCtl => write!(f, "configctl"),
            Stage::Gateway => write!(f, "gateway"),
            Stage::Ingest => write!(f, "ingest"),
            Stage::Derive => write!(f, "derive"),
            Stage::Store => write!(f, "store"),
        }
    }
}

/// An edge connecting two stages via a named transport.
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct Edge {
    pub from: Stage,
    pub to: Stage,
    pub transport: String,
    pub detail: String,
}

/// Full discovered topology.
#[derive(Debug, Default)]
pub struct Topology {
    pub configs: HashMap<String, ServiceConfig>,
    pub compose: Option<ComposeTopology>,
    pub source: Option<SourceTopology>,
    #[allow(dead_code)]
    pub edges: Vec<Edge>,
}

// ── Constants ────────────────────────────────────────────────────────

const EXPECTED_CONFIG_SERVICES: &[&str] = &["configctl", "gateway", "ingest", "derive", "store"];

const EXPECTED_COMPOSE_SERVICES: &[&str] =
    &["nats", "configctl", "gateway", "ingest", "derive", "store"];

const EXPECTED_STREAMS: &[&str] = &["CONFIGCTL_EVENTS", "OBSERVATION_EVENTS", "EVIDENCE_EVENTS"];

const EXPECTED_DURABLES: &[&str] = &[
    "derive-observation",
    "store-candle",
    "store-trade-burst",
    "store-volume",
];

const EXPECTED_SUBJECT_PREFIXES: &[&str] = &[
    "configctl.events.config",
    "configctl.control.config",
    "observation.events.market.trade",
    "evidence.events.candle.sampled",
    "evidence.events.tradeburst.sampled",
    "evidence.events.volume.sampled",
    "evidence.query.candle",
    "evidence.query.tradeburst",
    "evidence.query.volume",
];

// ── Main analysis entry point ────────────────────────────────────────

pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("topology-doctor");
    let mut topo = Topology::default();

    // Phase 1: Parse config files
    let configs_dir = project_root.join("deploy/configs");
    if configs_dir.is_dir() {
        report.add(CheckResult::pass("configs-dir-exists"));
        topo.configs = configs::parse_all_configs(&configs_dir)?;
        report.add(check_configs(&topo.configs));
    } else {
        report.add(CheckResult::from_findings(
            "configs-dir-exists",
            vec![Finding::error(
                "configs-dir",
                "deploy/configs directory not found",
            )
            .with_why("all topology checks depend on service configs to validate NATS consistency")
            .with_help("run `raccoon-cli doctor` to verify project structure first")],
        ));
        return Ok(report);
    }

    // Phase 2: Parse docker-compose
    let compose_path = project_root.join("deploy/compose/docker-compose.yaml");
    if compose_path.is_file() {
        match compose::parse_compose(&compose_path) {
            Ok(ct) => {
                report.add(check_compose(&ct));
                report.add(check_compose_dependencies(&ct));
                report.add(check_compose_runtime_contract(&ct));
                topo.compose = Some(ct);
            }
            Err(e) => {
                report.add(CheckResult::from_findings(
                    "compose-parse",
                    vec![Finding::error("compose", format!("failed to parse: {e}"))],
                ));
            }
        }
    } else {
        report.add(CheckResult::from_findings(
            "compose-exists",
            vec![Finding::warning("compose", "docker-compose.yaml not found")],
        ));
    }

    // Phase 3: Scan source for topology constants
    let internal_dir = project_root.join("internal");
    if internal_dir.is_dir() {
        match source::scan_source(&internal_dir) {
            Ok(st) => {
                report.add(check_source_streams(&st));
                report.add(check_source_durables(&st));
                report.add(check_source_subjects(&st));
                topo.source = Some(st);
            }
            Err(e) => {
                report.add(CheckResult::from_findings(
                    "source-scan",
                    vec![Finding::error("source", format!("failed to scan: {e}"))],
                ));
            }
        }
    }

    // Phase 4: Cross-validate
    report.add(check_nats_url_consistency(&topo));
    report.add(check_nats_compose_alignment(&topo));
    report.add(check_stream_subject_alignment(&topo));
    report.add(check_durable_stream_alignment(&topo));
    report.add(check_pipeline_continuity(&topo));

    Ok(report)
}

// ── Individual checks ────────────────────────────────────────────────

fn check_configs(configs: &HashMap<String, ServiceConfig>) -> CheckResult {
    let mut findings = Vec::new();

    for name in EXPECTED_CONFIG_SERVICES {
        if !configs.contains_key(*name) {
            findings.push(
                Finding::warning(
                    "config-present",
                    format!("config for '{name}' not found in deploy/configs/"),
                )
                .with_help(format!("create deploy/configs/{name}.jsonc")),
            );
        }
    }

    // Every service must have NATS config with url and enabled
    for name in EXPECTED_CONFIG_SERVICES {
        if let Some(cfg) = configs.get(*name) {
            if cfg.nats_url.is_none() {
                findings.push(
                    Finding::error(
                        "nats-config",
                        format!("'{name}' config has no nats.url"),
                    )
                    .with_why("all services communicate via NATS; without a URL the service cannot connect")
                    .with_help(format!("add nats.url to deploy/configs/{name}.jsonc")),
                );
            }
            if cfg.nats_enabled.is_none() {
                findings.push(
                    Finding::warning(
                        "nats-enabled",
                        format!("'{name}' config does not declare nats.enabled"),
                    )
                    .with_help(format!(
                        "add nats.enabled to deploy/configs/{name}.jsonc"
                    )),
                );
            }
        }
    }

    CheckResult::from_findings("config-completeness", findings)
}

fn check_compose(ct: &ComposeTopology) -> CheckResult {
    let mut findings = Vec::new();

    for name in EXPECTED_COMPOSE_SERVICES {
        if !ct.services.contains_key(*name) {
            findings.push(
                Finding::error(
                    "compose-service",
                    format!("service '{name}' not found in docker-compose"),
                )
                .with_why("local development requires all services; missing ones break `make up`")
                .with_help(format!(
                    "add '{name}' service to deploy/compose/docker-compose.yaml"
                )),
            );
        }
    }

    CheckResult::from_findings("compose-services", findings)
}

fn check_compose_dependencies(ct: &ComposeTopology) -> CheckResult {
    let mut findings = Vec::new();

    let expected_deps: &[(&str, &[&str])] = &[
        ("gateway", &["nats", "configctl", "store"]),
        ("ingest", &["nats", "configctl"]),
        ("derive", &["nats"]),
        ("store", &["nats", "derive"]),
        ("configctl", &["nats"]),
    ];

    for (service, deps) in expected_deps {
        if let Some(svc) = ct.services.get(*service) {
            for dep in *deps {
                if !svc.depends_on.contains(&dep.to_string()) {
                    findings.push(
                        Finding::warning(
                            "compose-dependency",
                            format!("'{service}' should depend on '{dep}'"),
                        )
                        .with_location(format!("docker-compose.yaml:{service}")),
                    );
                }
            }
        }
    }

    CheckResult::from_findings("compose-dependencies", findings)
}

fn check_compose_runtime_contract(ct: &ComposeTopology) -> CheckResult {
    let mut findings = Vec::new();

    // Verify expected port mappings (only gateway and nats expose host ports)
    let expected_ports: &[(&str, &str)] = &[
        ("nats", "4222:4222"),
        ("nats", "8222:8222"),
        ("gateway", "8080:8080"),
    ];

    for (service, port_fragment) in expected_ports {
        if let Some(svc) = ct.services.get(*service) {
            if !svc.ports.iter().any(|port| port.contains(port_fragment)) {
                findings.push(
                    Finding::error(
                        "compose-port",
                        format!(
                            "'{service}' must expose local port mapping containing '{port_fragment}'"
                        ),
                    )
                    .with_location(format!("docker-compose.yaml:{service}"))
                    .with_why("local smoke tests and operator workflows depend on stable host port mappings")
                    .with_help(format!(
                        "restore a port mapping containing '{port_fragment}' for '{service}'"
                    )),
                );
            }
        }
    }

    // Verify Go services use correct docker image pattern: market-foundry/{service}:dev
    let go_services = ["configctl", "gateway", "ingest", "derive", "store"];
    for service in &go_services {
        if let Some(svc) = ct.services.get(*service) {
            let expected_image = format!("market-foundry/{service}:dev");
            if let Some(image) = &svc.image {
                if !image.contains(&expected_image) {
                    findings.push(
                        Finding::warning(
                            "compose-image",
                            format!(
                                "'{service}' image '{image}' does not match expected '{expected_image}'"
                            ),
                        )
                        .with_location(format!("docker-compose.yaml:{service}"))
                        .with_help(format!("set service '{service}' image to '{expected_image}'")),
                    );
                }
            }
        }
    }

    // Verify NATS image is pinned
    if let Some(nats) = ct.services.get("nats") {
        if let Some(image) = &nats.image {
            if !image.starts_with("nats:") {
                findings.push(
                    Finding::error(
                        "compose-image",
                        format!("nats image '{image}' does not look like a standard nats image"),
                    )
                    .with_location("docker-compose.yaml:nats"),
                );
            }
        }
    }

    // Verify healthchecks: all services with depends_on should be depended on via condition
    // This is a lightweight heuristic — compose parser already extracts depends_on
    for service in EXPECTED_COMPOSE_SERVICES {
        if *service == "nats" {
            continue; // nats is infrastructure, not a Go service
        }
        if let Some(svc) = ct.services.get(*service) {
            if svc.image.is_none() {
                findings.push(
                    Finding::warning(
                        "compose-image",
                        format!("'{service}' has no image defined"),
                    )
                    .with_location(format!("docker-compose.yaml:{service}")),
                );
            }
        }
    }

    CheckResult::from_findings("compose-runtime-contract", findings)
}

fn check_source_streams(st: &SourceTopology) -> CheckResult {
    let mut findings = Vec::new();

    for stream in EXPECTED_STREAMS {
        if !st.streams.contains_key(*stream) {
            findings.push(
                Finding::error(
                    "stream-defined",
                    format!("expected stream '{stream}' not found in source"),
                )
                .with_why("JetStream streams are required for durable message delivery in the pipeline")
                .with_help(
                    "verify the stream constant is defined in the NATS adapter registry code",
                ),
            );
        }
    }

    CheckResult::from_findings("source-streams", findings)
}

fn check_source_durables(st: &SourceTopology) -> CheckResult {
    let mut findings = Vec::new();

    for durable in EXPECTED_DURABLES {
        if !st.durables.contains_key(*durable) {
            findings.push(Finding::error(
                "durable-defined",
                format!("expected durable consumer '{durable}' not found in source"),
            ));
        }
    }

    CheckResult::from_findings("source-durables", findings)
}

fn check_source_subjects(st: &SourceTopology) -> CheckResult {
    let mut findings = Vec::new();

    for prefix in EXPECTED_SUBJECT_PREFIXES {
        let found = st.subjects.iter().any(|s| s.starts_with(prefix));
        if !found {
            findings.push(Finding::warning(
                "subject-prefix",
                format!("no subjects with prefix '{prefix}' found in source"),
            ));
        }
    }

    CheckResult::from_findings("source-subjects", findings)
}

fn check_nats_url_consistency(topo: &Topology) -> CheckResult {
    let mut findings = Vec::new();
    let mut urls: Vec<(String, String)> = Vec::new();

    for (name, cfg) in &topo.configs {
        if let Some(url) = &cfg.nats_url {
            urls.push((name.clone(), url.clone()));
        }
    }

    if urls.len() >= 2 {
        let reference = &urls[0].1;
        for (name, url) in &urls[1..] {
            if url != reference {
                findings.push(Finding::warning(
                    "nats-url",
                    format!(
                        "NATS URL differs between '{}' ({}) and '{}' ({})",
                        urls[0].0, reference, name, url
                    ),
                ));
            }
        }
    }

    CheckResult::from_findings("nats-url-consistency", findings)
}

fn check_nats_compose_alignment(topo: &Topology) -> CheckResult {
    let mut findings = Vec::new();

    if let Some(compose) = &topo.compose {
        if compose.services.contains_key("nats") {
            for (name, cfg) in &topo.configs {
                if let Some(url) = &cfg.nats_url {
                    let host = url
                        .trim_start_matches("nats://")
                        .trim_start_matches("tls://");
                    let hostname = host.split(':').next().unwrap_or(host);
                    if hostname != "nats" && hostname != "localhost" && hostname != "127.0.0.1" {
                        findings.push(
                            Finding::warning(
                                "nats-host",
                                format!(
                                    "'{name}' NATS URL hostname '{hostname}' doesn't match compose service 'nats'"
                                ),
                            )
                            .with_help(format!(
                                "set nats.url to 'nats://nats:4222' in deploy/configs/{name}.jsonc"
                            )),
                        );
                    }
                }
            }
        }
    }

    CheckResult::from_findings("nats-compose-alignment", findings)
}

fn check_stream_subject_alignment(topo: &Topology) -> CheckResult {
    let mut findings = Vec::new();
    let source = match &topo.source {
        Some(s) => s,
        None => return CheckResult::skip("stream-subject-alignment", "source not scanned"),
    };

    // For each stream, verify its subjects appear in the global subject list
    for (stream_name, stream_subjects) in &source.streams {
        for subject_pattern in stream_subjects {
            let prefix = subject_pattern.trim_end_matches(".>");
            let has_matching = source
                .subjects
                .iter()
                .any(|s| s.starts_with(prefix) || s == subject_pattern);
            if !has_matching {
                findings.push(Finding::warning(
                    "stream-subject",
                    format!(
                        "stream '{stream_name}' declares subject '{subject_pattern}' but no matching subject found in source"
                    ),
                ));
            }
        }
    }

    CheckResult::from_findings("stream-subject-alignment", findings)
}

fn check_durable_stream_alignment(topo: &Topology) -> CheckResult {
    let mut findings = Vec::new();
    let source = match &topo.source {
        Some(s) => s,
        None => return CheckResult::skip("durable-stream-alignment", "source not scanned"),
    };

    for (durable_name, durable_stream) in &source.durables {
        if !source.streams.contains_key(durable_stream.as_str()) {
            findings.push(Finding::error(
                "durable-stream",
                format!(
                    "durable '{durable_name}' references stream '{durable_stream}' which was not found"
                ),
            ));
        }
    }

    CheckResult::from_findings("durable-stream-alignment", findings)
}

fn check_pipeline_continuity(topo: &Topology) -> CheckResult {
    let mut findings = Vec::new();

    // All services must have NATS config (the sole transport)
    let pipeline_services = ["ingest", "derive", "store", "gateway", "configctl"];
    for service in &pipeline_services {
        let has_nats = topo
            .configs
            .get(*service)
            .map_or(false, |cfg| cfg.nats_url.is_some());

        if !has_nats {
            findings.push(Finding::error(
                "pipeline-continuity",
                format!("'{service}' has no NATS config — cannot participate in the pipeline"),
            ));
        }
    }

    // Verify the observation stream exists and has a durable consumer (derive-observation)
    if let Some(source) = &topo.source {
        let has_observation_stream = source.streams.contains_key("OBSERVATION_EVENTS");
        let has_derive_durable = source.durables.contains_key("derive-observation");

        if has_observation_stream && !has_derive_durable {
            findings.push(Finding::error(
                "pipeline-subscriber",
                "OBSERVATION_EVENTS stream exists but derive-observation durable consumer not found",
            ));
        }
        if !has_observation_stream && has_derive_durable {
            findings.push(Finding::error(
                "pipeline-stream",
                "derive-observation durable exists but OBSERVATION_EVENTS stream not found",
            ));
        }

        // Verify the evidence stream exists and has at least one store durable consumer
        let has_evidence_stream = source.streams.contains_key("EVIDENCE_EVENTS");
        let has_store_durable = source.durables.contains_key("store-candle")
            || source.durables.contains_key("store-trade-burst")
            || source.durables.contains_key("store-volume");

        if has_evidence_stream && !has_store_durable {
            findings.push(Finding::error(
                "pipeline-subscriber",
                "EVIDENCE_EVENTS stream exists but no store durable consumers found (expected store-candle, store-trade-burst, store-volume)",
            ));
        }
        if !has_evidence_stream && has_store_durable {
            findings.push(Finding::error(
                "pipeline-stream",
                "store durable consumers exist but EVIDENCE_EVENTS stream not found",
            ));
        }

        // Signal pipeline continuity: SIGNAL_EVENTS ↔ store-signal-rsi
        let has_signal_stream = source.streams.contains_key("SIGNAL_EVENTS");
        let has_signal_durable = source.durables.contains_key("store-signal-rsi");

        if has_signal_stream && !has_signal_durable {
            findings.push(Finding::error(
                "pipeline-subscriber",
                "SIGNAL_EVENTS stream exists but no store-signal-rsi durable consumer found",
            ).with_why("signal events will accumulate with no consumer projecting them")
             .with_help("add store-signal-rsi consumer spec in internal/adapters/nats/signal_registry.go"));
        }
        if !has_signal_stream && has_signal_durable {
            findings.push(Finding::error(
                "pipeline-stream",
                "store-signal-rsi durable consumer exists but SIGNAL_EVENTS stream not found",
            ).with_why("consumer will fail to bind at runtime")
             .with_help("add SIGNAL_EVENTS stream spec in internal/adapters/nats/signal_registry.go"));
        }

        // Strategy pipeline continuity: STRATEGY_EVENTS ↔ store-strategy-mean-reversion-entry
        let has_strategy_stream = source.streams.contains_key("STRATEGY_EVENTS");
        let has_strategy_durable = source.durables.contains_key("store-strategy-mean-reversion-entry");

        if has_strategy_stream && !has_strategy_durable {
            findings.push(Finding::error(
                "pipeline-subscriber",
                "STRATEGY_EVENTS stream exists but no store-strategy-mean-reversion-entry durable consumer found",
            ).with_why("strategy events will accumulate with no consumer projecting them")
             .with_help("add store-strategy-mean-reversion-entry consumer spec in internal/adapters/nats/strategy_registry.go"));
        }
        if !has_strategy_stream && has_strategy_durable {
            findings.push(Finding::error(
                "pipeline-stream",
                "store-strategy-mean-reversion-entry durable consumer exists but STRATEGY_EVENTS stream not found",
            ).with_why("consumer will fail to bind at runtime")
             .with_help("add STRATEGY_EVENTS stream spec in internal/adapters/nats/strategy_registry.go"));
        }

        // Guard: premature projection events entry
        let has_projection_stream = source.streams.contains_key("PROJECTION_EVENTS");
        if has_projection_stream {
            findings.push(Finding::error(
                "premature-projection-entry",
                "PROJECTION_EVENTS stream found in source — projection notification family is not yet approved for entry",
            ).with_why("projection events are planned but not yet needed; gateway is stateless")
             .with_help("remove PROJECTION_EVENTS references until the projection family is formally approved"));
        }
    }

    CheckResult::from_findings("pipeline-continuity", findings)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::Severity;

    fn make_service_config(name: &str) -> ServiceConfig {
        ServiceConfig {
            name: name.into(),
            nats_url: Some("nats://nats:4222".into()),
            nats_enabled: Some(true),
            nats_request_timeout: Some("5s".into()),
        }
    }

    fn make_all_configs() -> HashMap<String, ServiceConfig> {
        let mut configs = HashMap::new();
        for name in EXPECTED_CONFIG_SERVICES {
            configs.insert(name.to_string(), make_service_config(name));
        }
        configs
    }

    fn make_source_topology() -> SourceTopology {
        let mut streams = HashMap::new();
        streams.insert(
            "CONFIGCTL_EVENTS".into(),
            vec!["configctl.events.config.>".into()],
        );
        streams.insert(
            "OBSERVATION_EVENTS".into(),
            vec!["observation.events.market.trade.>".into()],
        );
        streams.insert(
            "EVIDENCE_EVENTS".into(),
            vec![
                "evidence.events.candle.sampled.>".into(),
                "evidence.events.tradeburst.sampled.>".into(),
                "evidence.events.volume.sampled.>".into(),
            ],
        );
        streams.insert(
            "SIGNAL_EVENTS".into(),
            vec!["signal.events.>".into()],
        );
        streams.insert(
            "DECISION_EVENTS".into(),
            vec!["decision.events.>".into()],
        );
        streams.insert(
            "STRATEGY_EVENTS".into(),
            vec!["strategy.events.>".into()],
        );

        let mut durables = HashMap::new();
        durables.insert(
            "derive-observation".into(),
            "OBSERVATION_EVENTS".into(),
        );
        durables.insert(
            "store-candle".into(),
            "EVIDENCE_EVENTS".into(),
        );
        durables.insert(
            "store-trade-burst".into(),
            "EVIDENCE_EVENTS".into(),
        );
        durables.insert(
            "store-volume".into(),
            "EVIDENCE_EVENTS".into(),
        );
        durables.insert(
            "store-signal-rsi".into(),
            "SIGNAL_EVENTS".into(),
        );
        durables.insert(
            "store-decision-rsi-oversold".into(),
            "DECISION_EVENTS".into(),
        );
        durables.insert(
            "store-strategy-mean-reversion-entry".into(),
            "STRATEGY_EVENTS".into(),
        );

        let subjects = vec![
            "configctl.control.config.>".into(),
            "configctl.events.config.>".into(),
            "configctl.events.config.activated".into(),
            "evidence.events.candle.sampled.>".into(),
            "evidence.events.tradeburst.sampled.>".into(),
            "evidence.events.volume.sampled.>".into(),
            "evidence.query.candle.latest".into(),
            "evidence.query.tradeburst.latest".into(),
            "evidence.query.volume.latest".into(),
            "observation.events.market.trade.>".into(),
            "signal.events.>".into(),
            "signal.events.rsi.generated.>".into(),
            "signal.query.rsi.latest".into(),
            "decision.events.>".into(),
            "decision.events.rsi_oversold.evaluated.>".into(),
            "decision.query.rsi_oversold.latest".into(),
            "strategy.events.>".into(),
            "strategy.events.mean_reversion_entry.resolved.>".into(),
            "strategy.query.mean_reversion_entry.latest".into(),
        ];

        SourceTopology {
            streams,
            durables,
            subjects,
        }
    }

    fn make_compose_topology() -> ComposeTopology {
        let mut services = HashMap::new();

        services.insert(
            "nats".into(),
            compose::ComposeService {
                name: "nats".into(),
                image: Some("nats:2.10.18-alpine".into()),
                depends_on: vec![],
                profiles: vec![],
                ports: vec!["127.0.0.1:4222:4222".into(), "127.0.0.1:8222:8222".into()],
                internal_port: None,
            },
        );
        services.insert(
            "configctl".into(),
            compose::ComposeService {
                name: "configctl".into(),
                image: Some("market-foundry/configctl:dev".into()),
                depends_on: vec!["nats".into()],
                profiles: vec![],
                ports: vec![],
                internal_port: None,
            },
        );
        services.insert(
            "gateway".into(),
            compose::ComposeService {
                name: "gateway".into(),
                image: Some("market-foundry/gateway:dev".into()),
                depends_on: vec!["nats".into(), "configctl".into(), "store".into()],
                profiles: vec![],
                ports: vec!["127.0.0.1:8080:8080".into()],
                internal_port: None,
            },
        );
        services.insert(
            "ingest".into(),
            compose::ComposeService {
                name: "ingest".into(),
                image: Some("market-foundry/ingest:dev".into()),
                depends_on: vec!["nats".into(), "configctl".into()],
                profiles: vec![],
                ports: vec!["127.0.0.1:8082:8082".into()],
                internal_port: None,
            },
        );
        services.insert(
            "derive".into(),
            compose::ComposeService {
                name: "derive".into(),
                image: Some("market-foundry/derive:dev".into()),
                depends_on: vec!["nats".into()],
                profiles: vec![],
                ports: vec!["127.0.0.1:8083:8083".into()],
                internal_port: None,
            },
        );
        services.insert(
            "store".into(),
            compose::ComposeService {
                name: "store".into(),
                image: Some("market-foundry/store:dev".into()),
                depends_on: vec!["nats".into(), "derive".into()],
                profiles: vec![],
                ports: vec!["127.0.0.1:8081:8081".into()],
                internal_port: None,
            },
        );

        ComposeTopology { services }
    }

    #[test]
    fn config_check_passes_with_all_services() {
        let configs = make_all_configs();
        let result = check_configs(&configs);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn config_check_warns_missing_service() {
        let mut configs = make_all_configs();
        configs.remove("ingest");
        configs.remove("derive");

        let result = check_configs(&configs);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Warning && f.message.contains("ingest")));
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Warning && f.message.contains("derive")));
    }

    #[test]
    fn config_check_errors_service_without_nats_url() {
        let mut configs = make_all_configs();
        configs.get_mut("ingest").unwrap().nats_url = None;

        let result = check_configs(&configs);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("ingest")));
    }

    #[test]
    fn config_check_warns_service_without_nats_enabled() {
        let mut configs = make_all_configs();
        configs.get_mut("store").unwrap().nats_enabled = None;

        let result = check_configs(&configs);
        assert!(result
            .findings
            .iter()
            .any(|f| f.check == "nats-enabled" && f.message.contains("store")));
    }

    #[test]
    fn compose_check_passes_with_all_services() {
        let ct = make_compose_topology();
        let result = check_compose(&ct);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn compose_check_errors_missing_service() {
        let mut ct = make_compose_topology();
        ct.services.remove("derive");

        let result = check_compose(&ct);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("derive")));
    }

    #[test]
    fn compose_dependencies_passes_when_correct() {
        let ct = make_compose_topology();
        let result = check_compose_dependencies(&ct);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn compose_dependencies_warns_on_missing_dep() {
        let mut ct = make_compose_topology();
        ct.services
            .get_mut("gateway")
            .unwrap()
            .depends_on
            .retain(|d| d != "store");

        let result = check_compose_dependencies(&ct);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("gateway") && f.message.contains("store")));
    }

    #[test]
    fn compose_runtime_contract_passes_when_correct() {
        let ct = make_compose_topology();
        let result = check_compose_runtime_contract(&ct);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn compose_runtime_contract_fails_on_missing_port() {
        let mut ct = make_compose_topology();
        ct.services.get_mut("gateway").unwrap().ports.clear();

        let result = check_compose_runtime_contract(&ct);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
        assert!(result
            .findings
            .iter()
            .any(|f| f.check == "compose-port" && f.message.contains("gateway")));
    }

    #[test]
    fn compose_runtime_contract_warns_on_wrong_image() {
        let mut ct = make_compose_topology();
        ct.services.get_mut("ingest").unwrap().image =
            Some("wrong-project/ingest:dev".into());

        let result = check_compose_runtime_contract(&ct);
        assert!(result
            .findings
            .iter()
            .any(|f| f.check == "compose-image" && f.message.contains("ingest")));
    }

    #[test]
    fn nats_url_consistency_ok_when_matching() {
        let mut topo = Topology::default();
        topo.configs = make_all_configs();

        let result = check_nats_url_consistency(&topo);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn nats_url_consistency_warns_on_mismatch() {
        let mut topo = Topology::default();
        topo.configs = make_all_configs();
        topo.configs.get_mut("derive").unwrap().nats_url =
            Some("nats://other-host:4222".into());

        let result = check_nats_url_consistency(&topo);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Warning));
    }

    #[test]
    fn nats_compose_alignment_warns_on_bad_hostname() {
        let mut topo = Topology::default();
        topo.configs = make_all_configs();
        topo.configs.get_mut("ingest").unwrap().nats_url =
            Some("nats://wrong-host:4222".into());
        topo.compose = Some(make_compose_topology());

        let result = check_nats_compose_alignment(&topo);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("wrong-host")));
    }

    #[test]
    fn source_streams_passes_with_all_streams() {
        let st = make_source_topology();
        let result = check_source_streams(&st);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn source_streams_errors_on_missing_stream() {
        let mut st = make_source_topology();
        st.streams.remove("EVIDENCE_EVENTS");

        let result = check_source_streams(&st);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("EVIDENCE_EVENTS")));
    }

    #[test]
    fn source_durables_passes_with_all_durables() {
        let st = make_source_topology();
        let result = check_source_durables(&st);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn source_durables_errors_on_missing_durable() {
        let mut st = make_source_topology();
        st.durables.remove("store-candle");

        let result = check_source_durables(&st);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("store-candle")));
    }

    #[test]
    fn source_subjects_passes_with_all_prefixes() {
        let st = make_source_topology();
        let result = check_source_subjects(&st);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn source_subjects_warns_on_missing_prefix() {
        let mut st = make_source_topology();
        st.subjects.retain(|s| !s.starts_with("evidence.query"));

        let result = check_source_subjects(&st);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("evidence.query.candle")));
    }

    #[test]
    fn durable_stream_alignment_fails_on_orphan() {
        let mut source = make_source_topology();
        source
            .durables
            .insert("orphan-durable".into(), "NONEXISTENT_STREAM".into());
        let mut topo = Topology::default();
        topo.source = Some(source);

        let result = check_durable_stream_alignment(&topo);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("NONEXISTENT_STREAM")));
    }

    #[test]
    fn durable_stream_alignment_passes_when_all_match() {
        let mut topo = Topology::default();
        topo.source = Some(make_source_topology());
        let result = check_durable_stream_alignment(&topo);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn durable_stream_alignment_skips_without_source() {
        let topo = Topology::default();
        let result = check_durable_stream_alignment(&topo);
        assert_eq!(result.status, crate::models::CheckStatus::Skip);
    }

    #[test]
    fn stream_subject_alignment_passes_when_matched() {
        let mut topo = Topology::default();
        topo.source = Some(make_source_topology());

        let result = check_stream_subject_alignment(&topo);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn stream_subject_alignment_warns_orphan_stream_subject() {
        let mut source = make_source_topology();
        source
            .streams
            .insert("ORPHAN_STREAM".into(), vec!["orphan.events.>".into()]);
        let mut topo = Topology::default();
        topo.source = Some(source);

        let result = check_stream_subject_alignment(&topo);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("ORPHAN_STREAM")));
    }

    #[test]
    fn stream_subject_alignment_skips_without_source() {
        let topo = Topology::default();
        let result = check_stream_subject_alignment(&topo);
        assert_eq!(result.status, crate::models::CheckStatus::Skip);
    }

    #[test]
    fn pipeline_continuity_passes_with_complete_config() {
        let mut topo = Topology::default();
        topo.configs = make_all_configs();
        topo.source = Some(make_source_topology());

        let result = check_pipeline_continuity(&topo);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn pipeline_continuity_fails_without_nats() {
        let mut topo = Topology::default();
        topo.configs = make_all_configs();
        topo.configs.get_mut("derive").unwrap().nats_url = None;

        let result = check_pipeline_continuity(&topo);
        assert!(result
            .findings
            .iter()
            .any(|f| f.severity == Severity::Error && f.message.contains("derive")));
    }

    #[test]
    fn pipeline_continuity_errors_on_orphan_durable() {
        let mut topo = Topology::default();
        topo.configs = make_all_configs();
        let mut source = make_source_topology();
        source.streams.remove("OBSERVATION_EVENTS");
        // derive-observation durable still present but stream is gone
        topo.source = Some(source);

        let result = check_pipeline_continuity(&topo);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("OBSERVATION_EVENTS")));
    }

    #[test]
    fn pipeline_continuity_errors_on_orphan_stream() {
        let mut topo = Topology::default();
        topo.configs = make_all_configs();
        let mut source = make_source_topology();
        source.durables.remove("store-candle");
        source.durables.remove("store-trade-burst");
        source.durables.remove("store-volume");
        // EVIDENCE_EVENTS stream still present but all store durables are gone
        topo.source = Some(source);

        let result = check_pipeline_continuity(&topo);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("store durable consumers")));
    }

    #[test]
    fn analyze_returns_report_on_empty_configs_dir() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::create_dir_all(dir.path().join("deploy/configs")).unwrap();

        let report = analyze(dir.path()).unwrap();
        assert_eq!(report.title, "topology-doctor");
    }

    #[test]
    fn analyze_fails_when_no_configs_dir() {
        let dir = tempfile::tempdir().unwrap();
        let report = analyze(dir.path()).unwrap();
        assert!(!report.passed());
        assert!(report.checks.iter().any(|c| c.name == "configs-dir-exists"));
    }
}
