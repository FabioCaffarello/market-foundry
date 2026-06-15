//! check-delivery — ADR-0028 (delivery is read-only transport, Onda
//! H-11.d). Static enforcement of invariants I1/I5:
//!
//! - **reader-only adapter**: no production `.go` file under
//!   `adapter_dir` publishes to a stream (none of
//!   `forbidden_publish_tokens`) — delivery consumes INSIGHTS_EVENTS and
//!   never writes back (ADR-0008 single-writer; ADR-0028 I5).
//! - **stream-bound consumer**: `consumer_file` declares the durable
//!   `required_durable` and references `required_stream` — delivery
//!   reads only INSIGHTS_EVENTS via the deliver-insights durable (I1).
//!
//! Line-based scan (mirrors check-insights): robust for gofmt'd source,
//! cheap in the gate. Per P5 of the Harvest protocol, the onda that
//! hardens the invariant ships its analyzer.

use std::fs;
use std::path::Path;

use serde::Deserialize;

use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};

const DELIVERY_POLICY_PATH: &str = "tools/raccoon-cli/policies/delivery.toml";

#[derive(Debug, Deserialize)]
struct DeliveryPolicy {
    delivery: DeliverySection,
}

#[derive(Debug, Deserialize)]
struct DeliverySection {
    adapter_dir: String,
    consumer_file: String,
    required_durable: String,
    required_stream: String,
    forbidden_publish_tokens: Vec<String>,
}

pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("check-delivery");

    let policy_path = project_root.join(DELIVERY_POLICY_PATH);
    let policy: DeliveryPolicy = match fs::read_to_string(&policy_path) {
        Ok(raw) => match toml::from_str(&raw) {
            Ok(p) => {
                report.add(CheckResult::pass("policy-present"));
                p
            }
            Err(e) => {
                report.add(CheckResult::from_findings(
                    "policy-present",
                    vec![Finding::error("policy-parse", format!("{}: {e}", policy_path.display()))],
                ));
                return Ok(report);
            }
        },
        Err(e) => {
            report.add(CheckResult::from_findings(
                "policy-present",
                vec![Finding::error("policy-missing", format!("{}: {e}", policy_path.display()))],
            ));
            return Ok(report);
        }
    };
    let policy = policy.delivery;

    check_reader_only(project_root, &policy, &mut report)?;
    check_consumer_stream_bound(project_root, &policy, &mut report);

    Ok(report)
}

/// I5: the delivery adapter is reader-only — it never publishes events.
fn check_reader_only(project_root: &Path, policy: &DeliverySection, report: &mut Report) -> Result<()> {
    let root = project_root.join(&policy.adapter_dir);
    if !root.is_dir() {
        report.add(CheckResult::skip("reader-only", format!("{} not found", root.display())));
        return Ok(());
    }

    let mut findings: Vec<Finding> = Vec::new();
    let mut files_scanned = 0usize;

    let mut entries: Vec<_> = fs::read_dir(&root)?.filter_map(|e| e.ok()).map(|e| e.path()).collect();
    entries.sort();
    for path in entries {
        let name = match path.file_name().and_then(|n| n.to_str()) {
            Some(n) => n,
            None => continue,
        };
        if !name.ends_with(".go") || name.ends_with("_test.go") {
            continue;
        }
        files_scanned += 1;
        let content = fs::read_to_string(&path)?;
        let rel = path.strip_prefix(project_root).unwrap_or(&path).display().to_string();
        for (idx, line) in content.lines().enumerate() {
            if line.trim_start().starts_with("//") {
                continue;
            }
            for token in &policy.forbidden_publish_tokens {
                if line.contains(token.as_str()) {
                    findings.push(Finding::error(
                        "reader-only",
                        format!(
                            "{rel}:{}: delivery adapter calls `{token}` — delivery is \
                             read-only transport and MUST NOT publish to a stream \
                             (ADR-0028 I5 / ADR-0008 single-writer)",
                            idx + 1
                        ),
                    ));
                }
            }
        }
    }

    if findings.is_empty() {
        report.add(CheckResult::pass("reader-only"));
        report.add(CheckResult::pass(format!("adapter-files-scanned ({files_scanned} files)")));
    } else {
        report.add(CheckResult::from_findings("reader-only", findings));
    }
    Ok(())
}

/// I1: the delivery consumer binds the deliver-insights durable on
/// INSIGHTS_EVENTS.
fn check_consumer_stream_bound(project_root: &Path, policy: &DeliverySection, report: &mut Report) {
    let path = project_root.join(&policy.consumer_file);
    let content = match fs::read_to_string(&path) {
        Ok(c) => c,
        Err(e) => {
            report.add(CheckResult::from_findings(
                "consumer-stream-bound",
                vec![Finding::error("consumer-stream-bound", format!("{}: {e}", path.display()))],
            ));
            return;
        }
    };

    let mut findings: Vec<Finding> = Vec::new();
    if !content.contains(&policy.required_durable) {
        findings.push(Finding::error(
            "consumer-stream-bound",
            format!(
                "{}: does not declare the required durable `{}` (ADR-0028 I1)",
                policy.consumer_file, policy.required_durable
            ),
        ));
    }
    if !content.contains(&policy.required_stream) {
        findings.push(Finding::error(
            "consumer-stream-bound",
            format!(
                "{}: does not reference the required stream `{}` (ADR-0028 I1)",
                policy.consumer_file, policy.required_stream
            ),
        ));
    }

    if findings.is_empty() {
        report.add(CheckResult::pass("consumer-stream-bound"));
    } else {
        report.add(CheckResult::from_findings("consumer-stream-bound", findings));
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::path::Path;

    fn write_policy(root: &Path) {
        let dir = root.join("tools/raccoon-cli/policies");
        fs::create_dir_all(&dir).unwrap();
        fs::write(
            dir.join("delivery.toml"),
            r#"[delivery]
adapter_dir = "internal/adapters/nats/natsdelivery"
consumer_file = "internal/adapters/nats/natsdelivery/consumer.go"
required_durable = "deliver-insights"
required_stream = "INSIGHTS_EVENTS"
forbidden_publish_tokens = [".Publish(", ".PublishMsg(", ".PublishAsync("]
"#,
        )
        .unwrap();
    }

    fn write_adapter(root: &Path, file: &str, body: &str) {
        let dir = root.join("internal/adapters/nats/natsdelivery");
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join(file), body).unwrap();
    }

    fn write_full_fixture(root: &Path) {
        write_policy(root);
        write_adapter(
            root,
            "consumer.go",
            "package natsdelivery\n\nfunc spec() { _ = \"deliver-insights\"; _ = \"INSIGHTS_EVENTS\" }\n\nfunc consume() { _ = cons.Consume(onMessage) }\n",
        );
    }

    #[test]
    fn clean_surface_passes() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed, "expected pass: {report:?}");
    }

    #[test]
    fn adapter_publishing_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_adapter(tmp.path(), "bad.go", "package natsdelivery\n\nfunc leak() { _, _ = js.Publish(ctx, subj, data) }\n");
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure (adapter publishes): {report:?}");
    }

    #[test]
    fn consumer_missing_durable_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_adapter(tmp.path(), "consumer.go", "package natsdelivery\n\nfunc f() { _ = \"INSIGHTS_EVENTS\" }\n");
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure (no durable): {report:?}");
    }

    #[test]
    fn consumer_missing_stream_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_adapter(tmp.path(), "consumer.go", "package natsdelivery\n\nfunc f() { _ = \"deliver-insights\" }\n");
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure (no stream): {report:?}");
    }

    #[test]
    fn missing_policy_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure: {report:?}");
    }

    #[test]
    fn test_files_are_ignored() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        // A _test.go file may construct a publisher for fixtures.
        write_adapter(tmp.path(), "x_test.go", "package natsdelivery\n\nfunc t() { _, _ = js.Publish(ctx, s, d) }\n");
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed, "expected pass (test files ignored): {report:?}");
    }
}
