use crate::error::Result;
use std::collections::{HashMap, HashSet};
use std::path::Path;

/// Runtime binding source-level constants extracted from Go source files.
/// Reflects the market-foundry NATS/JetStream architecture:
/// - Streams: CONFIGCTL_EVENTS, OBSERVATION_EVENTS, EVIDENCE_EVENTS, SIGNAL_EVENTS, DECISION_EVENTS, STRATEGY_EVENTS
/// - Durable consumers: derive-observation, store-candle, store-trade-burst, store-volume, store-signal-rsi, store-decision-rsi_oversold, store-strategy-mean-reversion-entry
/// - Query subjects: evidence.query.{candle,tradeburst,volume}.latest, evidence.query.candle.history, signal.query.rsi.latest, decision.query.rsi_oversold.latest, strategy.query.mean_reversion_entry.latest, configctl.control.config.*
/// - Service binaries: configctl, gateway, ingest, derive, store
#[derive(Debug, Clone, Default)]
pub struct RuntimeBindingSource {
    /// Stream name -> list of subject patterns found in source.
    pub stream_subjects: HashMap<String, Vec<String>>,
    /// Durable consumer name -> stream name.
    pub durable_consumers: HashMap<String, String>,
    /// Query/request subjects found in source (e.g., "evidence.query.candle.latest").
    pub query_subjects: HashSet<String>,
    /// Publish subjects found in source (patterns like "observation.events.market.trade.*").
    pub publish_subjects: HashSet<String>,
    /// Service binary -> set of adapter types found (publisher, consumer, responder, kv).
    pub service_adapters: HashMap<String, HashSet<String>>,
    /// Lifecycle event names found in source (e.g., "config.activated").
    pub lifecycle_events: HashSet<String>,
    /// Whether each key adapter file exists.
    pub has_observation_publisher: bool,
    pub has_observation_consumer: bool,
    pub has_evidence_publisher: bool,
    pub has_evidence_consumer: bool,
    pub has_evidence_gateway: bool,
    pub has_candle_kv_store: bool,
    pub has_binding_watcher: bool,
    pub has_signal_publisher: bool,
    pub has_signal_consumer: bool,
    pub has_signal_gateway: bool,
    pub has_signal_kv_store: bool,
    pub has_signal_registry: bool,
    pub has_decision_publisher: bool,
    pub has_decision_consumer: bool,
    pub has_decision_gateway: bool,
    pub has_decision_kv_store: bool,
    pub has_decision_registry: bool,
    pub has_strategy_publisher: bool,
    pub has_strategy_consumer: bool,
    pub has_strategy_gateway: bool,
    pub has_strategy_kv_store: bool,
    pub has_strategy_registry: bool,
    // Risk domain adapter files (prepared for S64)
    pub has_risk_publisher: bool,
    pub has_risk_consumer: bool,
    pub has_risk_gateway: bool,
    pub has_risk_kv_store: bool,
    pub has_risk_registry: bool,
}

/// Scan Go source files under `internal/` for market-foundry runtime constants.
pub fn scan_runtime_bindings(internal_dir: &Path) -> Result<RuntimeBindingSource> {
    let mut src = RuntimeBindingSource::default();

    if !internal_dir.is_dir() {
        return Ok(src);
    }

    scan_dir(internal_dir, &mut src)?;

    // Check for key adapter files
    let adapters = internal_dir.join("adapters/nats");
    src.has_observation_publisher = adapters.join("observation_publisher.go").is_file();
    src.has_observation_consumer = adapters.join("observation_consumer.go").is_file();
    src.has_evidence_publisher = adapters.join("evidence_publisher.go").is_file();
    src.has_evidence_consumer = adapters.join("evidence_consumer.go").is_file();
    src.has_evidence_gateway = adapters.join("evidence_gateway.go").is_file();
    src.has_candle_kv_store = adapters.join("candle_kv_store.go").is_file();
    src.has_binding_watcher = adapters.join("binding_event_consumer.go").is_file();
    src.has_signal_publisher = adapters.join("signal_publisher.go").is_file();
    src.has_signal_consumer = adapters.join("signal_consumer.go").is_file();
    src.has_signal_gateway = adapters.join("signal_gateway.go").is_file();
    src.has_signal_kv_store = adapters.join("signal_kv_store.go").is_file();
    src.has_signal_registry = adapters.join("signal_registry.go").is_file();
    src.has_decision_publisher = adapters.join("decision_publisher.go").is_file();
    src.has_decision_consumer = adapters.join("decision_consumer.go").is_file();
    src.has_decision_gateway = adapters.join("decision_gateway.go").is_file();
    src.has_decision_kv_store = adapters.join("decision_kv_store.go").is_file();
    src.has_decision_registry = adapters.join("decision_registry.go").is_file();
    src.has_strategy_publisher = adapters.join("strategy_publisher.go").is_file();
    src.has_strategy_consumer = adapters.join("strategy_consumer.go").is_file();
    src.has_strategy_gateway = adapters.join("strategy_gateway.go").is_file();
    src.has_strategy_kv_store = adapters.join("strategy_kv_store.go").is_file();
    src.has_strategy_registry = adapters.join("strategy_registry.go").is_file();
    // Risk domain adapter files (prepared for S64)
    src.has_risk_publisher = adapters.join("risk_publisher.go").is_file();
    src.has_risk_consumer = adapters.join("risk_consumer.go").is_file();
    src.has_risk_gateway = adapters.join("risk_gateway.go").is_file();
    src.has_risk_kv_store = adapters.join("risk_kv_store.go").is_file();
    src.has_risk_registry = adapters.join("risk_registry.go").is_file();

    // Derive service adapter presence from actor scopes
    let scopes = internal_dir.join("actors/scopes");
    for service in &["ingest", "derive", "store"] {
        let scope_dir = scopes.join(service);
        if scope_dir.is_dir() {
            let adapters_set = src.service_adapters.entry(service.to_string()).or_default();
            scan_scope_for_adapters(&scope_dir, adapters_set)?;
        }
    }

    // Deduplicate subjects per stream
    for subjects in src.stream_subjects.values_mut() {
        subjects.sort();
        subjects.dedup();
    }

    Ok(src)
}

fn scan_scope_for_adapters(dir: &Path, adapters: &mut HashSet<String>) -> Result<()> {
    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return Ok(()),
    };

    for entry in entries {
        let entry = entry?;
        let name = entry.file_name().to_string_lossy().to_string();
        if !name.ends_with(".go") || name.ends_with("_test.go") {
            continue;
        }
        if name.contains("publisher") {
            adapters.insert("publisher".to_string());
        }
        if name.contains("consumer") {
            adapters.insert("consumer".to_string());
        }
        if name.contains("responder") || name.contains("query") {
            adapters.insert("responder".to_string());
        }
        if name.contains("websocket") {
            adapters.insert("websocket".to_string());
        }
        if name.contains("sampler") {
            adapters.insert("sampler".to_string());
        }
        if name.contains("projection") {
            adapters.insert("projection".to_string());
        }
        if name.contains("binding_watcher") || name.contains("binding_event") {
            adapters.insert("binding_watcher".to_string());
        }
    }

    Ok(())
}

fn scan_dir(dir: &Path, src: &mut RuntimeBindingSource) -> Result<()> {
    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return Ok(()),
    };

    for entry in entries {
        let entry = entry?;
        let path = entry.path();
        if path.is_dir() {
            let name = path.file_name().and_then(|n| n.to_str()).unwrap_or("");
            if name.starts_with('.') || name == "vendor" || name == "testdata" {
                continue;
            }
            scan_dir(&path, src)?;
        } else if path.extension().and_then(|e| e.to_str()) == Some("go") {
            if let Ok(content) = std::fs::read_to_string(&path) {
                scan_go_file(&content, src);
            }
        }
    }

    Ok(())
}

fn scan_go_file(content: &str, src: &mut RuntimeBindingSource) {
    extract_streams(content, src);
    extract_durables(content, src);
    extract_subjects(content, src);
    extract_lifecycle_events(content, src);
}

fn extract_streams(content: &str, src: &mut RuntimeBindingSource) {
    let lines: Vec<&str> = content.lines().collect();

    for (i, line) in lines.iter().enumerate() {
        let trimmed = line.trim();
        if trimmed.starts_with("//") {
            continue;
        }

        let is_stream_context =
            trimmed.contains("Name:") || trimmed.contains("Stream") || trimmed.contains("stream");

        if !is_stream_context {
            continue;
        }

        for word in extract_all_quoted(trimmed) {
            if is_stream_name(&word) {
                let subjects = find_subjects_near(&lines, i, 10);
                src.stream_subjects
                    .entry(word)
                    .or_default()
                    .extend(subjects);
            }
        }
    }
}

fn extract_durables(content: &str, src: &mut RuntimeBindingSource) {
    let lines: Vec<&str> = content.lines().collect();

    for (i, line) in lines.iter().enumerate() {
        let trimmed = line.trim();
        if trimmed.starts_with("//") || !trimmed.contains("Durable") {
            continue;
        }

        for val in extract_all_quoted(trimmed) {
            if val.contains('-') && val.chars().all(|c| c.is_alphanumeric() || c == '-') {
                let stream = find_stream_name_near(&lines, i, 15)
                    .or_else(|| find_stream_name_near(&lines, lines.len() / 2, lines.len()));
                if let Some(stream_name) = stream {
                    src.durable_consumers.insert(val, stream_name);
                }
            }
        }
    }
}

fn extract_subjects(content: &str, src: &mut RuntimeBindingSource) {
    for line in content.lines() {
        let trimmed = line.trim();
        if trimmed.starts_with("//") {
            continue;
        }

        for val in extract_all_quoted(trimmed) {
            if !is_nats_subject(&val) {
                continue;
            }

            if val.starts_with("observation.events.") {
                src.publish_subjects.insert(val);
            } else if val.starts_with("evidence.events.") {
                src.publish_subjects.insert(val);
            } else if val.starts_with("evidence.query.") {
                src.query_subjects.insert(val);
            } else if val.starts_with("signal.events.") {
                src.publish_subjects.insert(val);
            } else if val.starts_with("signal.query.") {
                src.query_subjects.insert(val);
            } else if val.starts_with("decision.events.") {
                src.publish_subjects.insert(val);
            } else if val.starts_with("decision.query.") {
                src.query_subjects.insert(val);
            } else if val.starts_with("strategy.events.") {
                src.publish_subjects.insert(val);
            } else if val.starts_with("strategy.query.") {
                src.query_subjects.insert(val);
            } else if val.starts_with("risk.events.") {
                src.publish_subjects.insert(val);
            } else if val.starts_with("risk.query.") {
                src.query_subjects.insert(val);
            } else if val.starts_with("configctl.control.") {
                src.query_subjects.insert(val);
            } else if val.starts_with("configctl.events.") {
                src.publish_subjects.insert(val);
            }
        }
    }
}

fn extract_lifecycle_events(content: &str, src: &mut RuntimeBindingSource) {
    for line in content.lines() {
        let trimmed = line.trim();
        if trimmed.starts_with("//") {
            continue;
        }

        if trimmed.contains("events.Name") || trimmed.contains("EventName") {
            for val in extract_all_quoted(trimmed) {
                if val.starts_with("config.") {
                    src.lifecycle_events.insert(val);
                }
            }
        }
    }
}

// ── Helpers ─────────────────────────────────────────────────────────

fn extract_all_quoted(s: &str) -> Vec<String> {
    let mut results = Vec::new();
    let mut rest = s;

    while let Some(start) = rest.find('"') {
        let after_quote = &rest[start + 1..];
        if let Some(end) = after_quote.find('"') {
            let value = &after_quote[..end];
            if !value.is_empty() {
                results.push(value.to_string());
            }
            rest = &after_quote[end + 1..];
        } else {
            break;
        }
    }

    results
}

fn is_stream_name(s: &str) -> bool {
    s.len() >= 3
        && s.chars()
            .all(|c| c.is_ascii_uppercase() || c == '_' || c.is_ascii_digit())
        && s.contains('_')
        && s.chars().next().map_or(false, |c| c.is_ascii_uppercase())
}

fn is_nats_subject(s: &str) -> bool {
    if s.is_empty() || s.len() < 3 {
        return false;
    }
    let segments: Vec<&str> = s.split('.').collect();
    if segments.len() < 2 {
        return false;
    }
    segments.iter().all(|seg| {
        !seg.is_empty()
            && seg
                .chars()
                .all(|c| c.is_alphanumeric() || c == '_' || c == '-' || c == '>' || c == '*')
    })
}

fn find_subjects_near(lines: &[&str], center: usize, radius: usize) -> Vec<String> {
    let start = center.saturating_sub(radius);
    let end = (center + radius).min(lines.len());

    for i in start..end {
        let trimmed = lines[i].trim();
        if trimmed.contains("Subjects") && !trimmed.starts_with("//") {
            let mut subjects = Vec::new();
            for val in extract_all_quoted(trimmed) {
                if is_nats_subject(&val) {
                    subjects.push(val);
                }
            }
            for j in (i + 1)..((i + 5).min(lines.len())) {
                for val in extract_all_quoted(lines[j]) {
                    if is_nats_subject(&val) {
                        subjects.push(val);
                    }
                }
                if lines[j].trim().contains('}') || lines[j].trim().contains(']') {
                    break;
                }
            }
            if !subjects.is_empty() {
                return subjects;
            }
        }
    }

    Vec::new()
}

fn find_stream_name_near(lines: &[&str], center: usize, radius: usize) -> Option<String> {
    // Search outward from center so the closest match wins. This prevents
    // picking up a KV bucket constant that happens to be above the durable
    // when the actual stream name is below it (closer).
    for offset in 0..=radius {
        for &dir in &[0isize, -1, 1] {
            let idx = if dir == 0 && offset == 0 {
                center
            } else if dir < 0 {
                match center.checked_sub(offset) {
                    Some(i) => i,
                    None => continue,
                }
            } else if dir > 0 {
                let i = center + offset;
                if i >= lines.len() {
                    continue;
                }
                i
            } else {
                continue;
            };

            if idx >= lines.len() {
                continue;
            }

            let trimmed = lines[idx].trim();
            if trimmed.starts_with("//") {
                continue;
            }
            for val in extract_all_quoted(trimmed) {
                if is_stream_name(&val) {
                    return Some(val);
                }
            }
        }
    }

    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extract_streams_from_source() {
        let content = r#"
func DefaultObservationRegistry() ObservationRegistry {
    return ObservationRegistry{
        Stream: StreamSpec{
            Name:     "OBSERVATION_EVENTS",
            Subjects: []string{"observation.events.>"},
        },
    }
}
"#;
        let mut src = RuntimeBindingSource::default();
        extract_streams(content, &mut src);
        assert!(src.stream_subjects.contains_key("OBSERVATION_EVENTS"));
    }

    #[test]
    fn extract_durables_from_source() {
        let content = r#"
    DeriveObservation: ConsumerSpec{
        Durable: "derive-observation",
        Event: EventSpec{
            Stream: StreamSpec{
                Name: "OBSERVATION_EVENTS",
            },
        },
    },
"#;
        let mut src = RuntimeBindingSource::default();
        extract_durables(content, &mut src);
        assert_eq!(
            src.durable_consumers.get("derive-observation"),
            Some(&"OBSERVATION_EVENTS".to_string())
        );
    }

    #[test]
    fn extract_subjects_classifies_observation() {
        let content = r#"
    subject := "observation.events.market.trade.binancef"
"#;
        let mut src = RuntimeBindingSource::default();
        extract_subjects(content, &mut src);
        assert!(src
            .publish_subjects
            .contains("observation.events.market.trade.binancef"));
    }

    #[test]
    fn extract_subjects_classifies_query() {
        let content = r#"
    subject := "evidence.query.candle.latest"
"#;
        let mut src = RuntimeBindingSource::default();
        extract_subjects(content, &mut src);
        assert!(src.query_subjects.contains("evidence.query.candle.latest"));
    }

    #[test]
    fn extract_subjects_classifies_evidence_events() {
        let content = r#"
    subject := "evidence.events.candle.sampled.BTCUSDT.1m"
"#;
        let mut src = RuntimeBindingSource::default();
        extract_subjects(content, &mut src);
        assert!(src
            .publish_subjects
            .contains("evidence.events.candle.sampled.BTCUSDT.1m"));
    }

    #[test]
    fn extract_subjects_classifies_configctl_control() {
        let content = r#"
    subject := "configctl.control.config.compile"
"#;
        let mut src = RuntimeBindingSource::default();
        extract_subjects(content, &mut src);
        assert!(src
            .query_subjects
            .contains("configctl.control.config.compile"));
    }

    #[test]
    fn extract_lifecycle_events_from_source() {
        let content = r#"
    EventActivated               events.Name = "config.activated"
    EventDeactivated             events.Name = "config.deactivated"
"#;
        let mut src = RuntimeBindingSource::default();
        extract_lifecycle_events(content, &mut src);
        assert!(src.lifecycle_events.contains("config.activated"));
        assert!(src.lifecycle_events.contains("config.deactivated"));
    }

    #[test]
    fn extract_lifecycle_events_skips_comments() {
        let content = r#"
    // EventActivated events.Name = "config.activated"
    EventDeactivated events.Name = "config.deactivated"
"#;
        let mut src = RuntimeBindingSource::default();
        extract_lifecycle_events(content, &mut src);
        assert!(!src.lifecycle_events.contains("config.activated"));
        assert!(src.lifecycle_events.contains("config.deactivated"));
    }

    #[test]
    fn scan_runtime_bindings_on_empty_dir() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::create_dir_all(dir.path().join("internal")).unwrap();
        let result = scan_runtime_bindings(&dir.path().join("internal")).unwrap();
        assert!(result.stream_subjects.is_empty());
        assert!(!result.has_observation_publisher);
    }

    #[test]
    fn scan_runtime_bindings_returns_default_on_nonexistent() {
        let result = scan_runtime_bindings(Path::new("/nonexistent")).unwrap();
        assert!(result.stream_subjects.is_empty());
        assert!(result.durable_consumers.is_empty());
    }

    #[test]
    fn scan_runtime_bindings_detects_adapter_files() {
        let dir = tempfile::tempdir().unwrap();
        let internal = dir.path().join("internal");
        let adapters = internal.join("adapters/nats");
        std::fs::create_dir_all(&adapters).unwrap();

        std::fs::write(
            adapters.join("observation_publisher.go"),
            "package nats\n",
        )
        .unwrap();
        std::fs::write(
            adapters.join("observation_consumer.go"),
            "package nats\n",
        )
        .unwrap();
        std::fs::write(adapters.join("evidence_publisher.go"), "package nats\n").unwrap();
        std::fs::write(adapters.join("evidence_consumer.go"), "package nats\n").unwrap();
        std::fs::write(adapters.join("evidence_gateway.go"), "package nats\n").unwrap();
        std::fs::write(adapters.join("candle_kv_store.go"), "package nats\n").unwrap();
        std::fs::write(
            adapters.join("binding_event_consumer.go"),
            "package nats\n",
        )
        .unwrap();

        let result = scan_runtime_bindings(&internal).unwrap();
        assert!(result.has_observation_publisher);
        assert!(result.has_observation_consumer);
        assert!(result.has_evidence_publisher);
        assert!(result.has_evidence_consumer);
        assert!(result.has_evidence_gateway);
        assert!(result.has_candle_kv_store);
        assert!(result.has_binding_watcher);
    }

    #[test]
    fn scan_scope_for_adapters_detects_actors() {
        let dir = tempfile::tempdir().unwrap();
        let scope = dir.path().join("ingest");
        std::fs::create_dir_all(&scope).unwrap();

        std::fs::write(scope.join("publisher_actor.go"), "package ingest\n").unwrap();
        std::fs::write(scope.join("websocket_actor.go"), "package ingest\n").unwrap();
        std::fs::write(
            scope.join("binding_watcher_actor.go"),
            "package ingest\n",
        )
        .unwrap();

        let mut adapters = HashSet::new();
        scan_scope_for_adapters(&scope, &mut adapters).unwrap();
        assert!(adapters.contains("publisher"));
        assert!(adapters.contains("websocket"));
        assert!(adapters.contains("binding_watcher"));
    }

    #[test]
    fn extract_subjects_classifies_signal_events() {
        let content = r#"
    subject := "signal.events.rsi.generated.binancef.btcusdt.60"
"#;
        let mut src = RuntimeBindingSource::default();
        extract_subjects(content, &mut src);
        assert!(src
            .publish_subjects
            .contains("signal.events.rsi.generated.binancef.btcusdt.60"));
    }

    #[test]
    fn extract_subjects_classifies_signal_query() {
        let content = r#"
    subject := "signal.query.rsi.latest"
"#;
        let mut src = RuntimeBindingSource::default();
        extract_subjects(content, &mut src);
        assert!(src.query_subjects.contains("signal.query.rsi.latest"));
    }

    #[test]
    fn scan_runtime_bindings_detects_signal_adapter_files() {
        let dir = tempfile::tempdir().unwrap();
        let internal = dir.path().join("internal");
        let adapters = internal.join("adapters/nats");
        std::fs::create_dir_all(&adapters).unwrap();

        std::fs::write(adapters.join("signal_publisher.go"), "package nats\n").unwrap();
        std::fs::write(adapters.join("signal_consumer.go"), "package nats\n").unwrap();
        std::fs::write(adapters.join("signal_gateway.go"), "package nats\n").unwrap();
        std::fs::write(adapters.join("signal_kv_store.go"), "package nats\n").unwrap();
        std::fs::write(adapters.join("signal_registry.go"), "package nats\n").unwrap();

        let result = scan_runtime_bindings(&internal).unwrap();
        assert!(result.has_signal_publisher);
        assert!(result.has_signal_consumer);
        assert!(result.has_signal_gateway);
        assert!(result.has_signal_kv_store);
        assert!(result.has_signal_registry);
    }

    #[test]
    fn scan_runtime_bindings_with_go_source() {
        let dir = tempfile::tempdir().unwrap();
        let internal = dir.path().join("internal");
        let nats_adapters = internal.join("adapters/nats");
        std::fs::create_dir_all(&nats_adapters).unwrap();

        std::fs::write(
            nats_adapters.join("observation_registry.go"),
            r#"package nats
func DefaultObservationRegistry() ObservationRegistry {
    return ObservationRegistry{
        Stream: StreamSpec{
            Name:     "OBSERVATION_EVENTS",
            Subjects: []string{"observation.events.>"},
        },
        Consumer: ConsumerSpec{
            Durable: "derive-observation",
        },
    }
}
"#,
        )
        .unwrap();

        std::fs::write(
            nats_adapters.join("observation_publisher.go"),
            "package nats\n",
        )
        .unwrap();

        let result = scan_runtime_bindings(&internal).unwrap();
        assert!(result.stream_subjects.contains_key("OBSERVATION_EVENTS"));
        assert!(result.has_observation_publisher);
    }
}
