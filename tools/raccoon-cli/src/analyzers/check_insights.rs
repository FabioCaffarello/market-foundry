//! check-insights — ADR-0027 (insights são decision-support
//! read-only, Onda H-8.a). Static enforcement of invariant I2:
//!
//! - **read-only domain**: no production `.go` file under
//!   `domain_root` imports any path in `forbidden_domain_imports`
//!   (the directive chain). Insights describe; they never direct.
//! - **stream-bound publisher**: `publisher_file` references
//!   `required_stream` and none of `forbidden_streams` — insights
//!   publish only to INSIGHTS_EVENTS.
//!
//! Line-based scan (mirrors check-venue-parity / check-subjects):
//! robust for gofmt'd imports, cheap in the gate. Per P5 of the
//! Harvest protocol, the onda that adds the invariant ships its
//! analyzer.

use std::fs;
use std::path::Path;

use serde::Deserialize;

use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};

const INSIGHTS_POLICY_PATH: &str = "tools/raccoon-cli/policies/insights.toml";

#[derive(Debug, Deserialize)]
struct InsightsPolicy {
    insights: InsightsSection,
}

#[derive(Debug, Deserialize)]
struct InsightsSection {
    domain_root: String,
    forbidden_domain_imports: Vec<String>,
    publisher_file: String,
    required_stream: String,
    forbidden_streams: Vec<String>,
}

pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("check-insights");

    let policy_path = project_root.join(INSIGHTS_POLICY_PATH);
    let policy: InsightsPolicy = match fs::read_to_string(&policy_path) {
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
    let policy = policy.insights;

    check_domain_read_only(project_root, &policy, &mut report)?;
    check_publisher_stream_bound(project_root, &policy, &mut report);

    Ok(report)
}

/// I2a: the insights domain imports none of the directive-chain
/// packages.
fn check_domain_read_only(project_root: &Path, policy: &InsightsSection, report: &mut Report) -> Result<()> {
    let root = project_root.join(&policy.domain_root);
    if !root.is_dir() {
        report.add(CheckResult::skip("domain-read-only", format!("{} not found", root.display())));
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
            for forbidden in &policy.forbidden_domain_imports {
                // Match the quoted import path.
                if line.contains(&format!("\"{forbidden}\"")) {
                    findings.push(Finding::error(
                        "domain-read-only",
                        format!(
                            "{rel}:{}: insights domain imports `{forbidden}` — insights are \
                             decision-support and MUST NOT import the directive chain (ADR-0027 I2)",
                            idx + 1
                        ),
                    ));
                }
            }
        }
    }

    if findings.is_empty() {
        report.add(CheckResult::pass("domain-read-only"));
        report.add(CheckResult::pass(format!("domain-files-scanned ({files_scanned} files)")));
    } else {
        report.add(CheckResult::from_findings("domain-read-only", findings));
    }
    Ok(())
}

/// I2b: the insights publisher binds to INSIGHTS_EVENTS only.
fn check_publisher_stream_bound(project_root: &Path, policy: &InsightsSection, report: &mut Report) {
    let path = project_root.join(&policy.publisher_file);
    let content = match fs::read_to_string(&path) {
        Ok(c) => c,
        Err(e) => {
            report.add(CheckResult::from_findings(
                "publisher-stream-bound",
                vec![Finding::error("publisher-stream-bound", format!("{}: {e}", path.display()))],
            ));
            return;
        }
    };

    let mut findings: Vec<Finding> = Vec::new();
    if !content.contains(&policy.required_stream) {
        findings.push(Finding::error(
            "publisher-stream-bound",
            format!(
                "{}: does not reference the required stream `{}` (ADR-0027 I2)",
                policy.publisher_file, policy.required_stream
            ),
        ));
    }
    for forbidden in &policy.forbidden_streams {
        if content.contains(forbidden) {
            findings.push(Finding::error(
                "publisher-stream-bound",
                format!(
                    "{}: references directive-chain stream `{forbidden}` — insights publish only \
                     to {} (ADR-0027 I2)",
                    policy.publisher_file, policy.required_stream
                ),
            ));
        }
    }

    if findings.is_empty() {
        report.add(CheckResult::pass("publisher-stream-bound"));
    } else {
        report.add(CheckResult::from_findings("publisher-stream-bound", findings));
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
            dir.join("insights.toml"),
            r#"[insights]
domain_root = "internal/domain/insights"
forbidden_domain_imports = ["internal/domain/strategy", "internal/domain/execution"]
publisher_file = "internal/adapters/nats/natsinsights/publisher.go"
required_stream = "INSIGHTS_EVENTS"
forbidden_streams = ["STRATEGY_EVENTS", "EXECUTION_EVENTS"]
"#,
        )
        .unwrap();
    }

    fn write_domain(root: &Path, file: &str, body: &str) {
        let dir = root.join("internal/domain/insights");
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join(file), body).unwrap();
    }

    fn write_publisher(root: &Path, body: &str) {
        let dir = root.join("internal/adapters/nats/natsinsights");
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join("publisher.go"), body).unwrap();
    }

    fn write_full_fixture(root: &Path) {
        write_policy(root);
        write_domain(root, "volume_profile.go", "package insights\n\nimport \"internal/domain/instrument\"\n\ntype VolumeProfile struct{ I instrument.CanonicalInstrument }\n");
        write_publisher(root, "package natsinsights\n\nfunc f() { _ = \"INSIGHTS_EVENTS\" }\n");
    }

    #[test]
    fn clean_surface_passes() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed, "expected pass: {report:?}");
    }

    #[test]
    fn domain_importing_directive_chain_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_domain(tmp.path(), "bad.go", "package insights\n\nimport \"internal/domain/strategy\"\n");
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure: {report:?}");
    }

    #[test]
    fn publisher_missing_required_stream_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_publisher(tmp.path(), "package natsinsights\nfunc f() {}\n");
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure: {report:?}");
    }

    #[test]
    fn publisher_referencing_directive_stream_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_publisher(tmp.path(), "package natsinsights\nfunc f() { _ = \"INSIGHTS_EVENTS\"; _ = \"EXECUTION_EVENTS\" }\n");
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure: {report:?}");
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
        // A _test.go file may import anything (e.g. test helpers).
        write_domain(tmp.path(), "x_test.go", "package insights_test\n\nimport \"internal/domain/strategy\"\n");
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed, "expected pass (test files ignored): {report:?}");
    }
}
