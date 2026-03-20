use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};
use std::collections::HashMap;
use std::path::Path;

mod dataplane;
mod envelope;
mod events;
mod registry;

// ── Discovered contracts ────────────────────────────────────────────

/// Full contract index built from scanning the source.
#[derive(Debug)]
pub struct ContractIndex {
    pub registry: registry::RegistryIndex,
    pub envelope: Option<envelope::EnvelopeContract>,
    pub codec: Option<envelope::CodecUsage>,
    pub events: events::DomainEventIndex,
    pub dataplane: Option<dataplane::DataPlaneContract>,
}

// ── Main entry point ────────────────────────────────────────────────

pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("contract-audit");

    let internal_dir = project_root.join("internal");
    if !internal_dir.is_dir() {
        report.add(CheckResult::from_findings(
            "internal-dir",
            vec![Finding::error("internal-dir", "internal/ directory not found")
                .with_why("contract-audit scans Go source in internal/ for registries, envelopes, and domain events")
                .with_help("run `raccoon-cli doctor` to verify project structure first")],
        ));
        return Ok(report);
    }

    // Phase 1: Scan all contract sources
    let registry = registry::scan_registries(&internal_dir)?;
    let envelope_contract = envelope::scan_envelope(&internal_dir)?;
    let codec = envelope::scan_codec(&internal_dir)?;
    let domain_events = events::scan_domain_events(&internal_dir)?;
    let dp_contract = dataplane::scan_dataplane(&internal_dir)?;

    let index = ContractIndex {
        registry,
        envelope: envelope_contract,
        codec,
        events: domain_events,
        dataplane: dp_contract,
    };

    // Phase 2: Run checks
    report.add(check_registry_control_completeness(&index));
    report.add(check_registry_event_completeness(&index));
    report.add(check_subject_type_convention(&index));
    report.add(check_reply_type_symmetry(&index));
    report.add(check_queue_group_convention(&index));
    report.add(check_event_stream_coverage(&index));
    report.add(check_consumer_filter_validity(&index));
    report.add(check_envelope_required_fields(&index));
    report.add(check_codec_consistency(&index));
    report.add(check_dataplane_field_completeness(&index));
    report.add(check_dataplane_content_type_default(&index));
    report.add(check_event_metadata_presence(&index));
    report.add(check_event_registry_alignment(&index));

    Ok(report)
}

// ── Checks ──────────────────────────────────────────────────────────

/// Verify all control specs have Subject, RequestType, ReplyType, QueueGroup.
fn check_registry_control_completeness(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    if index.registry.control_specs.is_empty() {
        findings.push(
            Finding::error("control-specs", "no control specs found in registry files")
                .with_why("control specs define the request-reply contract for NATS communication")
                .with_help(
                    "add ControlSpec definitions to the registry files in internal/adapters/nats/",
                ),
        );
        return CheckResult::from_findings("registry-control-completeness", findings);
    }

    for spec in &index.registry.control_specs {
        if spec.subject.is_empty() {
            findings.push(
                Finding::error(
                    "control-subject",
                    format!("'{}' has empty Subject", spec.name),
                )
                .with_location(&spec.file)
                .with_why(
                    "Subject is the NATS routing address; without it requests cannot be dispatched",
                ),
            );
        }
        if spec.request_type.is_empty() {
            findings.push(
                Finding::error(
                    "control-request-type",
                    format!("'{}' has empty RequestType", spec.name),
                )
                .with_location(&spec.file)
                .with_why("RequestType is used by the codec to serialize/deserialize the envelope correctly"),
            );
        }
        if spec.reply_type.is_empty() {
            findings.push(
                Finding::error(
                    "control-reply-type",
                    format!("'{}' has empty ReplyType", spec.name),
                )
                .with_location(&spec.file)
                .with_why("ReplyType is used by the codec to deserialize the response envelope"),
            );
        }
        if spec.queue_group.is_empty() {
            findings.push(
                Finding::warning(
                    "control-queue-group",
                    format!("'{}' has empty QueueGroup", spec.name),
                )
                .with_location(&spec.file)
                .with_why("without a queue group, all subscribers receive every message instead of load-balancing"),
            );
        }
    }

    CheckResult::from_findings("registry-control-completeness", findings)
}

/// Verify all event specs have Subject and Type.
fn check_registry_event_completeness(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    if index.registry.event_specs.is_empty() {
        findings.push(Finding::error(
            "event-specs",
            "no event specs found in registry files",
        )
        .with_why("event specs define the JetStream event contract; downstream consumers depend on them")
        .with_help("add EventSpec definitions to the registry files in internal/adapters/nats/"));
        return CheckResult::from_findings("registry-event-completeness", findings);
    }

    for spec in &index.registry.event_specs {
        if spec.subject.is_empty() {
            findings.push(
                Finding::error("event-subject", format!("'{}' has empty Subject", spec.name))
                    .with_location(&spec.file)
                    .with_why("Subject is the JetStream publishing address; events cannot be stored without it"),
            );
        }
        if spec.event_type.is_empty() {
            findings.push(
                Finding::error("event-type", format!("'{}' has empty Type", spec.name))
                    .with_location(&spec.file)
                    .with_why(
                        "Type is used for envelope kind routing and consumer deserialization",
                    ),
            );
        }
    }

    CheckResult::from_findings("registry-event-completeness", findings)
}

/// Verify subject-to-type naming convention (heuristic — warnings only).
/// Market-foundry uses versioned types ({domain}.{plane}.{version}.{name}),
/// so subject-to-type mapping is not 1:1. This check flags potential mismatches
/// as warnings for review, not hard errors.
fn check_subject_type_convention(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    for spec in &index.registry.control_specs {
        if spec.subject.contains(".control.") {
            // Heuristic: "x.control.y" → request type should reference the same domain
            let _expected_command = spec.subject.replace(".control.", ".command.");
            let _expected_query = spec.subject.replace(".control.", ".query.");

            // Accept versioned types (e.g., configctl.control.v1.create_draft_command)
            // Only check that domains match between subject and type
            let domain = spec.subject.split('.').next().unwrap_or("");
            let type_domain = spec.request_type.split('.').next().unwrap_or("");
            if domain != type_domain {
                findings.push(
                    Finding::warning(
                        "subject-request-type",
                        format!(
                            "'{}': subject domain '{}' doesn't match request type domain '{}'",
                            spec.name, domain, type_domain
                        ),
                    )
                    .with_location(&spec.file),
                );
            }

            let reply_domain = spec.reply_type.split('.').next().unwrap_or("");
            if domain != reply_domain {
                findings.push(
                    Finding::warning(
                        "subject-reply-type",
                        format!(
                            "'{}': subject domain '{}' doesn't match reply type domain '{}'",
                            spec.name, domain, reply_domain
                        ),
                    )
                    .with_location(&spec.file),
                );
            }
        } else {
            // Non-control subjects: just verify domain consistency
            let domain = spec.subject.split('.').next().unwrap_or("");
            let type_domain = spec.request_type.split('.').next().unwrap_or("");
            if domain != type_domain {
                findings.push(
                    Finding::warning(
                        "subject-request-type",
                        format!(
                            "'{}': subject domain '{}' doesn't match request type domain '{}'",
                            spec.name, domain, type_domain
                        ),
                    )
                    .with_location(&spec.file),
                );
            }
        }
    }

    for spec in &index.registry.event_specs {
        // Skip wildcard patterns (used for stream subscriptions, not individual events)
        if spec.subject.ends_with(".>") || spec.subject.ends_with(".*") {
            continue;
        }

        // Verify domain consistency: subject domain should match type domain
        // Market-foundry uses versioned types (e.g., evidence.events.v1.candle_sampled)
        // so we only check domain, not exact mapping
        let subject_domain = spec.subject.split('.').next().unwrap_or("");
        let type_domain = spec.event_type.split('.').next().unwrap_or("");
        if subject_domain != type_domain {
            findings.push(
                Finding::warning(
                    "event-subject-type",
                    format!(
                        "'{}': subject domain '{}' doesn't match type domain '{}'",
                        spec.name, subject_domain, type_domain
                    ),
                )
                .with_location(&spec.file),
            );
        }
    }

    CheckResult::from_findings("subject-type-convention", findings)
}

/// Verify every control spec has both request and reply types that are paired.
fn check_reply_type_symmetry(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    for spec in &index.registry.control_specs {
        let req_suffix = control_operation_suffix(&spec.request_type);
        let reply_suffix = control_operation_suffix(&spec.reply_type);

        if req_suffix != reply_suffix {
            findings.push(
                Finding::warning(
                    "reply-symmetry",
                    format!(
                        "'{}': request type suffix '{}' doesn't match reply type suffix '{}'",
                        spec.name, req_suffix, reply_suffix
                    ),
                )
                .with_location(&spec.file),
            );
        }
    }

    CheckResult::from_findings("reply-type-symmetry", findings)
}

/// Verify queue group naming follows `{domain}.{scope}` convention.
fn check_queue_group_convention(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    for spec in &index.registry.control_specs {
        if spec.queue_group.is_empty() {
            continue;
        }

        // Queue group should be a dot-separated identifier
        let parts: Vec<&str> = spec.queue_group.split('.').collect();
        if parts.len() < 2 {
            findings.push(
                Finding::warning(
                    "queue-group",
                    format!(
                        "'{}': queue group '{}' should follow 'domain.scope' convention",
                        spec.name, spec.queue_group
                    ),
                )
                .with_location(&spec.file),
            );
        }

        // Queue group domain should match the subject domain
        let subject_domain = spec.subject.split('.').next().unwrap_or("");
        let qg_domain = parts.first().unwrap_or(&"");
        if subject_domain != *qg_domain {
            findings.push(
                Finding::warning(
                    "queue-group-domain",
                    format!(
                        "'{}': queue group domain '{}' doesn't match subject domain '{}'",
                        spec.name, qg_domain, subject_domain
                    ),
                )
                .with_location(&spec.file),
            );
        }
    }

    CheckResult::from_findings("queue-group-convention", findings)
}

/// Verify all event subjects are covered by a JetStream stream's subject pattern.
fn check_event_stream_coverage(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    if index.registry.streams.is_empty() {
        return CheckResult::skip("event-stream-coverage", "no stream specs found");
    }

    // Build set of all stream subject patterns
    let stream_patterns: Vec<(&str, &str)> = index
        .registry
        .streams
        .iter()
        .flat_map(|s| {
            s.subjects
                .iter()
                .map(move |subj| (s.name.as_str(), subj.as_str()))
        })
        .collect();

    for event in &index.registry.event_specs {
        // Skip events that are wildcard patterns themselves (stream subscriptions)
        if event.subject.ends_with(".>") || event.subject.ends_with(".*") {
            continue;
        }

        let covered = stream_patterns
            .iter()
            .any(|(_, pattern)| subject_matches_pattern(&event.subject, pattern));

        if !covered && !stream_patterns.is_empty() {
            findings.push(
                Finding::warning(
                    "event-stream",
                    format!(
                        "event '{}' with subject '{}' may not be covered by any stream",
                        event.name, event.subject
                    ),
                )
                .with_location(&event.file),
            );
        }
    }

    CheckResult::from_findings("event-stream-coverage", findings)
}

/// Verify consumer filter subjects are valid subsets of their stream's subject patterns.
fn check_consumer_filter_validity(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    for consumer in &index.registry.consumers {
        // Find the stream this consumer belongs to
        let stream = index
            .registry
            .streams
            .iter()
            .find(|s| s.name == consumer.stream_name);

        if let Some(stream) = stream {
            for filter in &consumer.filter_subjects {
                let valid = stream
                    .subjects
                    .iter()
                    .any(|stream_subj| subject_matches_pattern(filter, stream_subj));

                if !valid && !stream.subjects.is_empty() {
                    findings.push(
                        Finding::warning(
                            "consumer-filter",
                            format!(
                                "consumer '{}' filter '{}' may not match stream '{}' subjects {:?}",
                                consumer.durable, filter, stream.name, stream.subjects
                            ),
                        )
                        .with_location(&consumer.file),
                    );
                }
            }
        } else if !consumer.stream_name.is_empty() {
            findings.push(
                Finding::error(
                    "consumer-stream",
                    format!(
                        "consumer '{}' references stream '{}' which was not found",
                        consumer.durable, consumer.stream_name
                    ),
                )
                .with_location(&consumer.file),
            );
        }
    }

    CheckResult::from_findings("consumer-filter-validity", findings)
}

/// Verify envelope contract defines required fields and validates them.
fn check_envelope_required_fields(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    let envelope = match &index.envelope {
        Some(e) => e,
        None => {
            return CheckResult::skip("envelope-required-fields", "envelope.go not found");
        }
    };

    // Expected required fields that must be validated
    let expected_required = ["id", "kind", "type", "timestamp", "content_type"];

    for field in &expected_required {
        if !envelope.required_fields.iter().any(|f| f == field) {
            findings.push(
                Finding::error(
                    "envelope-required",
                    format!("envelope field '{}' is not validated as required", field),
                )
                .with_location(&envelope.file),
            );
        }
    }

    // Verify default content type is set
    match &envelope.default_content_type {
        Some(ct) if ct == "application/json" => {
            findings.push(Finding::info(
                "envelope-content-type",
                format!("default content type is '{}'", ct),
            ));
        }
        Some(ct) => {
            findings.push(
                Finding::warning(
                    "envelope-content-type",
                    format!("unexpected default content type: '{}'", ct),
                )
                .with_location(&envelope.file),
            );
        }
        None => {
            findings.push(
                Finding::warning("envelope-content-type", "no default content type found")
                    .with_location(&envelope.file),
            );
        }
    }

    // Verify valid kinds
    let expected_kinds = ["command", "event", "request", "reply"];
    for kind in &expected_kinds {
        if !envelope.valid_kinds.iter().any(|k| k == kind) {
            findings.push(
                Finding::error(
                    "envelope-kind",
                    format!("expected Kind '{}' not defined", kind),
                )
                .with_location(&envelope.file),
            );
        }
    }

    CheckResult::from_findings("envelope-required-fields", findings)
}

/// Verify NATS codec uses CBOR and performs kind/type validation.
fn check_codec_consistency(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    let codec = match &index.codec {
        Some(c) => c,
        None => {
            return CheckResult::skip("codec-consistency", "codec.go not found");
        }
    };

    // Verify CBOR serialization
    if codec.serialization_format != "cbor" {
        findings.push(
            Finding::error(
                "codec-format",
                format!(
                    "NATS codec uses '{}' instead of expected 'cbor'",
                    codec.serialization_format
                ),
            )
            .with_location(&codec.file),
        );
    } else {
        findings.push(Finding::info(
            "codec-format",
            "NATS codec uses CBOR serialization",
        ));
    }

    // Verify encode functions set correct Kind
    let expected_encode = [
        ("encodeControlRequest", "command"),
        ("encodeControlReply", "reply"),
        ("encodeEvent", "event"),
    ];

    for (func, expected_kind) in &expected_encode {
        match codec
            .encode_kind_checks
            .iter()
            .find(|c| c.function == *func)
        {
            Some(check) => {
                if check.expected_kind != *expected_kind {
                    findings.push(
                        Finding::error(
                            "codec-kind",
                            format!(
                                "{} uses Kind '{}' but expected '{}'",
                                func, check.expected_kind, expected_kind
                            ),
                        )
                        .with_location(&codec.file),
                    );
                }
            }
            None => {
                findings.push(
                    Finding::warning(
                        "codec-encode",
                        format!("encode function '{}' not found", func),
                    )
                    .with_location(&codec.file),
                );
            }
        }
    }

    // Verify decode functions validate correct Kind
    let expected_decode = [
        ("decodeControlRequest", "command"),
        ("decodeControlReply", "reply"),
        ("decodeEvent", "event"),
    ];

    for (func, expected_kind) in &expected_decode {
        match codec
            .decode_kind_checks
            .iter()
            .find(|c| c.function == *func)
        {
            Some(check) => {
                if check.expected_kind != *expected_kind {
                    findings.push(
                        Finding::error(
                            "codec-kind",
                            format!(
                                "{} validates Kind '{}' but expected '{}'",
                                func, check.expected_kind, expected_kind
                            ),
                        )
                        .with_location(&codec.file),
                    );
                }
            }
            None => {
                findings.push(
                    Finding::warning(
                        "codec-decode",
                        format!("decode function '{}' not found", func),
                    )
                    .with_location(&codec.file),
                );
            }
        }
    }

    CheckResult::from_findings("codec-consistency", findings)
}

/// Verify DataPlane message validates all critical fields.
fn check_dataplane_field_completeness(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    let dp = match &index.dataplane {
        Some(d) => d,
        None => {
            return CheckResult::skip(
                "dataplane-field-completeness",
                "dataplane contracts.go not found",
            );
        }
    };

    // Expected required fields in the Message struct
    let expected_message_fields = ["binding", "origin", "payload", "metadata"];
    for field in &expected_message_fields {
        if !dp.message_fields.contains(&field.to_string()) {
            findings.push(
                Finding::error(
                    "dataplane-field",
                    format!("Message struct missing expected field '{}'", field),
                )
                .with_location(&dp.file),
            );
        }
    }

    // Expected validated fields
    let expected_validated = [
        "binding.name",
        "binding.topic",
        "binding.scope.kind",
        "binding.scope.key",
        "origin.source",
        "origin.topic",
        "metadata.message_id",
        "metadata.ingested_at",
        "metadata.content_type",
        "payload",
    ];

    for field in &expected_validated {
        if !dp.validated_fields.iter().any(|f| f.path == *field) {
            findings.push(
                Finding::error(
                    "dataplane-validation",
                    format!("DataPlane Message.Validate() doesn't check '{}'", field),
                )
                .with_location(&dp.file),
            );
        }
    }

    // Verify message ID format exists
    if dp.message_id_format.is_some() {
        findings.push(Finding::info(
            "dataplane-message-id",
            "MessageID format for Kafka records is defined",
        ));
    } else {
        findings.push(
            Finding::warning(
                "dataplane-message-id",
                "MessageIDForKafkaRecord format not found",
            )
            .with_location(&dp.file),
        );
    }

    CheckResult::from_findings("dataplane-field-completeness", findings)
}

/// Verify DataPlane defaults content-type to application/json.
fn check_dataplane_content_type_default(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    let dp = match &index.dataplane {
        Some(d) => d,
        None => {
            return CheckResult::skip("dataplane-content-type", "dataplane contracts.go not found");
        }
    };

    match &dp.default_content_type {
        Some(ct) if ct == "application/json" => {
            findings.push(Finding::info(
                "dataplane-content-type",
                "DataPlane defaults content_type to 'application/json'",
            ));
        }
        Some(ct) => {
            findings.push(
                Finding::warning(
                    "dataplane-content-type",
                    format!(
                        "DataPlane default content_type is '{}', expected 'application/json'",
                        ct
                    ),
                )
                .with_location(&dp.file),
            );
        }
        None => {
            findings.push(
                Finding::warning(
                    "dataplane-content-type",
                    "DataPlane ContentTypeJSON constant not found",
                )
                .with_location(&dp.file),
            );
        }
    }

    // Verify source default
    match &dp.default_source {
        Some(src) if src == "kafka" => {
            findings.push(Finding::info(
                "dataplane-source",
                "DataPlane defaults source to 'kafka'",
            ));
        }
        Some(src) => {
            findings.push(
                Finding::warning(
                    "dataplane-source",
                    format!("DataPlane default source is '{}', expected 'kafka'", src),
                )
                .with_location(&dp.file),
            );
        }
        None => {
            findings.push(
                Finding::warning(
                    "dataplane-source",
                    "DataPlane SourceKafka constant not found",
                )
                .with_location(&dp.file),
            );
        }
    }

    CheckResult::from_findings("dataplane-content-type", findings)
}

/// Verify all domain events have a Metadata field.
fn check_event_metadata_presence(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    if index.events.events.is_empty() {
        return CheckResult::skip("event-metadata-presence", "no domain events found");
    }

    for event in &index.events.events {
        if !event.has_metadata {
            findings.push(
                Finding::error(
                    "event-metadata",
                    format!(
                        "domain event '{}' ({}) is missing events.Metadata field",
                        event.struct_name, event.event_name
                    ),
                )
                .with_location(&event.file),
            );
        }
    }

    if findings.is_empty() {
        findings.push(Finding::info(
            "event-metadata",
            format!(
                "all {} domain events have Metadata field",
                index.events.events.len()
            ),
        ));
    }

    CheckResult::from_findings("event-metadata-presence", findings)
}

/// Verify domain event names align with registry event spec subjects.
fn check_event_registry_alignment(index: &ContractIndex) -> CheckResult {
    let mut findings = Vec::new();

    if index.events.events.is_empty() || index.registry.event_specs.is_empty() {
        return CheckResult::skip(
            "event-registry-alignment",
            "domain events or registry event specs not found",
        );
    }

    let mut domain_events: HashMap<&str, Vec<&events::DomainEventDef>> = HashMap::new();
    for event in &index.events.events {
        domain_events
            .entry(event.domain.as_str())
            .or_default()
            .push(event);
    }

    let registry_specs: Vec<_> = index
        .registry
        .event_specs
        .iter()
        .filter(|spec| !spec.subject.contains("dataplane."))
        .filter(|spec| !spec.subject.contains('>') && !spec.subject.contains('*'))
        .collect();

    if registry_specs.is_empty() {
        return CheckResult::skip(
            "event-registry-alignment",
            "registry event specs not found",
        );
    }

    for spec in &registry_specs {
        let domain = spec.subject.split('.').next().unwrap_or("");
        let Some(domain_defs) = domain_events.get(domain) else {
            findings.push(
                Finding::warning(
                    "event-alignment",
                    format!(
                        "registry event '{}' (subject '{}') has no domain events in internal/domain/{domain}/events.go",
                        spec.name, spec.subject
                    ),
                )
                .with_location(&spec.file),
            );
            continue;
        };

        if !registry_event_matches_domain(spec, domain_defs) {
            findings.push(
                Finding::warning(
                    "event-alignment",
                    format!(
                        "registry event '{}' (subject '{}', type '{}') has no matching domain event in '{domain}'",
                        spec.name, spec.subject, spec.event_type
                    ),
                )
                .with_location(&spec.file),
            );
        }
    }

    for event in &index.events.events {
        let has_registry = registry_specs.iter().any(|spec| {
            spec.subject.split('.').next().unwrap_or("") == event.domain
                && domain_event_matches_registry(event, spec)
        });

        if !has_registry {
            findings.push(
                Finding::warning(
                    "event-alignment",
                    format!(
                        "domain event '{}' ({}) in '{}' has no matching registry event spec",
                        event.event_name, event.struct_name, event.domain
                    ),
                )
                .with_location(&event.file),
            );
        }
    }

    CheckResult::from_findings("event-registry-alignment", findings)
}

// ── Helpers ─────────────────────────────────────────────────────────

fn control_operation_suffix(type_name: &str) -> String {
    let suffix = type_name.split('.').last().unwrap_or("");
    suffix
        .trim_end_matches("_request")
        .trim_end_matches("_reply")
        .to_string()
}

fn registry_event_matches_domain(
    spec: &registry::EventSpecRecord,
    domain_events: &[&events::DomainEventDef],
) -> bool {
    domain_events
        .iter()
        .any(|event| domain_event_matches_registry(event, spec))
}

fn domain_event_matches_registry(
    event: &events::DomainEventDef,
    spec: &registry::EventSpecRecord,
) -> bool {
    let subject_suffix = subject_suffix(&spec.subject);
    let type_suffix = event_type_suffix(&spec.event_type);
    let event_tokens = tokenize_event_name(&event.event_name);
    let subject_tokens = tokenize_event_name(&subject_suffix);
    let type_tokens = tokenize_event_name(&type_suffix);

    event_tokens == subject_tokens
        || event_tokens == type_tokens
        || is_subsequence(&event_tokens, &subject_tokens)
        || is_subsequence(&subject_tokens, &event_tokens)
        || is_subsequence(&event_tokens, &type_tokens)
        || is_subsequence(&type_tokens, &event_tokens)
        || (event.domain == spec.subject.split('.').next().unwrap_or("")
            && event_tokens.last() == type_tokens.last())
}

fn subject_suffix(subject: &str) -> String {
    let parts: Vec<&str> = subject.split('.').collect();
    if parts.len() <= 2 {
        return subject.to_string();
    }
    parts[2..].join(".")
}

fn event_type_suffix(event_type: &str) -> String {
    let parts: Vec<&str> = event_type.split('.').collect();
    if let Some(version_idx) = parts.iter().position(|part| {
        part.starts_with('v') && part[1..].chars().all(|c| c.is_ascii_digit())
    }) {
        if version_idx + 1 < parts.len() {
            return parts[version_idx + 1..].join(".");
        }
    }

    parts.last().copied().unwrap_or(event_type).to_string()
}

fn tokenize_event_name(value: &str) -> Vec<String> {
    value
        .split(['.', '_'])
        .filter(|part| !part.is_empty())
        .map(|part| part.to_lowercase())
        .collect()
}

fn is_subsequence(needle: &[String], haystack: &[String]) -> bool {
    if needle.is_empty() {
        return true;
    }
    if needle.len() > haystack.len() {
        return false;
    }

    let mut needle_idx = 0usize;
    for token in haystack {
        if token == &needle[needle_idx] {
            needle_idx += 1;
            if needle_idx == needle.len() {
                return true;
            }
        }
    }

    false
}

/// Check if a concrete subject matches a NATS subject pattern.
/// Supports ">" (multi-level wildcard) and "*" (single-level wildcard).
fn subject_matches_pattern(subject: &str, pattern: &str) -> bool {
    if subject == pattern {
        return true;
    }

    let subj_parts: Vec<&str> = subject.split('.').collect();
    let pat_parts: Vec<&str> = pattern.split('.').collect();

    let mut si = 0;
    let mut pi = 0;

    while pi < pat_parts.len() && si < subj_parts.len() {
        match pat_parts[pi] {
            ">" => return true, // matches everything after
            "*" => {
                si += 1;
                pi += 1;
            }
            seg => {
                if seg != subj_parts[si] {
                    return false;
                }
                si += 1;
                pi += 1;
            }
        }
    }

    si == subj_parts.len() && pi == pat_parts.len()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn subject_matches_exact() {
        assert!(subject_matches_pattern("foo.bar", "foo.bar"));
        assert!(!subject_matches_pattern("foo.bar", "foo.baz"));
    }

    #[test]
    fn subject_matches_wildcard_gt() {
        assert!(subject_matches_pattern(
            "configctl.events.config.activated",
            "configctl.events.config.>"
        ));
        assert!(subject_matches_pattern(
            "dataplane.ingestion.received.global.default.orders",
            "dataplane.ingestion.received.>"
        ));
        assert!(!subject_matches_pattern(
            "other.events.foo",
            "configctl.events.config.>"
        ));
    }

    #[test]
    fn subject_matches_star() {
        assert!(subject_matches_pattern("foo.bar.baz", "foo.*.baz"));
        assert!(!subject_matches_pattern("foo.bar.qux", "foo.*.baz"));
    }

    #[test]
    fn check_convention_passes_correct_specs() {
        let index = make_test_index();
        let result = check_subject_type_convention(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn check_convention_warns_bad_request_type() {
        let mut index = make_test_index();
        // Different domain triggers a warning (not error — convention is heuristic)
        index.registry.control_specs[0].request_type = "wrong.type.here".into();
        let result = check_subject_type_convention(&index);
        assert!(
            result
                .findings
                .iter()
                .any(|f| f.check == "subject-request-type"),
            "should warn about domain mismatch"
        );
    }

    #[test]
    fn check_convention_warns_bad_reply_type() {
        let mut index = make_test_index();
        index.registry.control_specs[0].reply_type = "wrong.type.here".into();
        let result = check_subject_type_convention(&index);
        assert!(
            result
                .findings
                .iter()
                .any(|f| f.check == "subject-reply-type"),
            "should warn about domain mismatch"
        );
    }

    #[test]
    fn check_event_convention_passes() {
        let index = make_test_index();
        let result = check_subject_type_convention(&index);
        // Should pass since events follow the convention
        let event_findings: Vec<_> = result
            .findings
            .iter()
            .filter(|f| f.check.contains("event-subject-type"))
            .collect();
        assert!(event_findings.is_empty());
    }

    #[test]
    fn check_reply_symmetry_passes() {
        let index = make_test_index();
        let result = check_reply_type_symmetry(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn check_reply_symmetry_accepts_versioned_request_reply_pairs() {
        let mut index = make_test_index();
        index.registry.control_specs[0].request_type = "signal.query.v1.rsi_latest_request".into();
        index.registry.control_specs[0].reply_type = "signal.query.v1.rsi_latest_reply".into();
        let result = check_reply_type_symmetry(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn check_event_stream_coverage_passes() {
        let index = make_test_index();
        let result = check_event_stream_coverage(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn check_event_stream_coverage_warns_uncovered() {
        let mut index = make_test_index();
        index.registry.event_specs.push(registry::EventSpecRecord {
            name: "Orphan".into(),
            subject: "orphan.events.something".into(),
            event_type: "orphan.event.something".into(),
            stream_name: None,
            file: "test.go".into(),
        });
        let result = check_event_stream_coverage(&index);
        // Uncovered events are warnings now (streams may not be parseable)
        assert!(
            result.findings.iter().any(|f| f.check == "event-stream"),
            "should warn about uncovered event"
        );
    }

    #[test]
    fn check_consumer_filter_passes() {
        let index = make_test_index();
        let result = check_consumer_filter_validity(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn check_consumer_filter_warns_invalid() {
        let mut index = make_test_index();
        index.registry.consumers.push(registry::ConsumerSpecRecord {
            durable: "bad-consumer".into(),
            stream_name: "CONFIGCTL_EVENTS".into(),
            filter_subjects: vec!["other.subject.nope".into()],
            file: "test.go".into(),
        });
        let result = check_consumer_filter_validity(&index);
        // Consumer filter mismatches are warnings (stream subjects may not be parsed)
        assert!(
            result.findings.iter().any(|f| f.check == "consumer-filter"),
            "should warn about filter mismatch"
        );
    }

    #[test]
    fn check_queue_group_convention_passes() {
        let index = make_test_index();
        let result = check_queue_group_convention(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    // ── subject_matches_pattern edge cases ──────────────────────────

    #[test]
    fn subject_matches_empty_strings() {
        assert!(subject_matches_pattern("", ""));
        assert!(!subject_matches_pattern("foo.bar", ""));
        assert!(!subject_matches_pattern("", "foo.bar"));
    }

    #[test]
    fn subject_matches_single_segment_no_match() {
        assert!(!subject_matches_pattern("foo", "bar"));
    }

    #[test]
    fn subject_matches_gt_at_root() {
        assert!(subject_matches_pattern("anything.here", ">"));
        assert!(subject_matches_pattern("a.b.c.d.e", ">"));
    }

    #[test]
    fn subject_matches_star_single_level_only() {
        assert!(!subject_matches_pattern("foo.bar.baz.qux", "foo.*.baz"));
    }

    #[test]
    fn subject_matches_gt_only_at_tail() {
        // ">" consumes all remaining segments
        assert!(subject_matches_pattern("a.b.c.d", "a.>"));
        assert!(!subject_matches_pattern("x.b.c.d", "a.>"));
    }

    #[test]
    fn subject_matches_multiple_stars() {
        assert!(subject_matches_pattern("a.b.c", "*.*.c"));
        assert!(!subject_matches_pattern("a.b.d", "*.*.c"));
    }

    #[test]
    fn subject_matches_length_mismatch() {
        assert!(!subject_matches_pattern("a.b", "a.b.c"));
        assert!(!subject_matches_pattern("a.b.c", "a.b"));
    }

    // ── check edge cases ────────────────────────────────────────────

    #[test]
    fn check_registry_control_completeness_empty_specs() {
        let mut index = make_test_index();
        index.registry.control_specs.clear();
        let result = check_registry_control_completeness(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    #[test]
    fn check_registry_event_completeness_empty_specs() {
        let mut index = make_test_index();
        index.registry.event_specs.clear();
        let result = check_registry_event_completeness(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Fail);
    }

    #[test]
    fn check_event_stream_coverage_no_streams() {
        let mut index = make_test_index();
        index.registry.streams.clear();
        let result = check_event_stream_coverage(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Skip);
    }

    #[test]
    fn check_envelope_required_fields_skips_when_missing() {
        let index = make_test_index();
        // envelope is None in make_test_index
        let result = check_envelope_required_fields(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Skip);
    }

    #[test]
    fn check_codec_consistency_skips_when_missing() {
        let index = make_test_index();
        let result = check_codec_consistency(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Skip);
    }

    #[test]
    fn check_dataplane_field_completeness_skips_when_missing() {
        let index = make_test_index();
        let result = check_dataplane_field_completeness(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Skip);
    }

    #[test]
    fn check_event_metadata_presence_skips_when_no_events() {
        let index = make_test_index();
        let result = check_event_metadata_presence(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Skip);
    }

    #[test]
    fn check_event_registry_alignment_skips_when_no_events() {
        let index = make_test_index();
        let result = check_event_registry_alignment(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Skip);
    }

    #[test]
    fn check_event_registry_alignment_accepts_family_specific_registry_subjects() {
        let mut index = make_test_index();
        index.registry.event_specs = vec![
            registry::EventSpecRecord {
                name: "TradeReceived".into(),
                subject: "observation.events.market.trade".into(),
                event_type: "observation.events.v1.trade_received".into(),
                stream_name: Some("OBSERVATION_EVENTS".into()),
                file: "observation_registry.go".into(),
            },
            registry::EventSpecRecord {
                name: "RSIGenerated".into(),
                subject: "signal.events.rsi.generated".into(),
                event_type: "signal.events.v1.rsi_generated".into(),
                stream_name: Some("SIGNAL_EVENTS".into()),
                file: "signal_registry.go".into(),
            },
            registry::EventSpecRecord {
                name: "VenueOrderFilled".into(),
                subject: "execution.fill.venue_market_order".into(),
                event_type: "execution.fill.v1.venue_market_order_filled".into(),
                stream_name: Some("EXECUTION_FILL_EVENTS".into()),
                file: "execution_registry.go".into(),
            },
        ];
        index.events.events = vec![
            events::DomainEventDef {
                domain: "observation".into(),
                const_name: "EventTradeReceived".into(),
                event_name: "market.trade_received".into(),
                struct_name: "TradeReceivedEvent".into(),
                has_metadata: true,
                file: "internal/domain/observation/events.go".into(),
            },
            events::DomainEventDef {
                domain: "signal".into(),
                const_name: "EventSignalGenerated".into(),
                event_name: "signal_generated".into(),
                struct_name: "SignalGeneratedEvent".into(),
                has_metadata: true,
                file: "internal/domain/signal/events.go".into(),
            },
            events::DomainEventDef {
                domain: "execution".into(),
                const_name: "EventVenueOrderFilled".into(),
                event_name: "venue_order_filled".into(),
                struct_name: "VenueOrderFilledEvent".into(),
                has_metadata: true,
                file: "internal/domain/execution/events.go".into(),
            },
        ];

        let result = check_event_registry_alignment(&index);
        assert_eq!(result.status, crate::models::CheckStatus::Pass);
    }

    #[test]
    fn check_queue_group_warns_on_single_segment() {
        let mut index = make_test_index();
        index.registry.control_specs[0].queue_group = "singleword".into();
        let result = check_queue_group_convention(&index);
        assert!(result
            .findings
            .iter()
            .any(|f| f.message.contains("domain.scope")));
    }

    #[test]
    fn check_reply_symmetry_warns_on_mismatch() {
        let mut index = make_test_index();
        index.registry.control_specs[0].reply_type = "configctl.reply.other_thing".into();
        let result = check_reply_type_symmetry(&index);
        assert!(result.findings.iter().any(|f| f.check == "reply-symmetry"));
    }

    fn make_test_index() -> ContractIndex {
        ContractIndex {
            registry: registry::RegistryIndex {
                control_specs: vec![
                    registry::ControlSpec {
                        name: "CreateDraft".into(),
                        subject: "configctl.control.create_draft".into(),
                        request_type: "configctl.command.create_draft".into(),
                        reply_type: "configctl.reply.create_draft".into(),
                        queue_group: "configctl.control".into(),
                        file: "configctl_registry.go".into(),
                    },
                    registry::ControlSpec {
                        name: "GetConfig".into(),
                        subject: "configctl.control.get_config".into(),
                        request_type: "configctl.query.get_config".into(),
                        reply_type: "configctl.reply.get_config".into(),
                        queue_group: "configctl.control".into(),
                        file: "configctl_registry.go".into(),
                    },
                ],
                event_specs: vec![registry::EventSpecRecord {
                    name: "DraftCreated".into(),
                    subject: "configctl.events.config.draft_created".into(),
                    event_type: "configctl.event.config.draft_created".into(),
                    stream_name: Some("CONFIGCTL_EVENTS".into()),
                    file: "configctl_registry.go".into(),
                }],
                streams: vec![registry::StreamSpecRecord {
                    name: "CONFIGCTL_EVENTS".into(),
                    subjects: vec!["configctl.events.config.>".into()],
                    file: "configctl_registry.go".into(),
                }],
                consumers: vec![registry::ConsumerSpecRecord {
                    durable: "validator-runtime-cache-v1".into(),
                    stream_name: "CONFIGCTL_EVENTS".into(),
                    filter_subjects: vec!["configctl.events.config.activated".into()],
                    file: "configctl_registry.go".into(),
                }],
            },
            envelope: None,
            codec: None,
            events: events::DomainEventIndex::default(),
            dataplane: None,
        }
    }
}
