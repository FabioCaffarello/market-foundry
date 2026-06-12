//! check-venue-parity — ADR-0022 multi-venue normalization policy
//! (Onda H-7.a). Statically validates rules R1–R3 of the policy; the
//! analyzer itself is rule R4.
//!
//! Declarative algorithm: reads
//! `tools/raccoon-cli/policies/venue_parity.toml` and checks:
//!
//! - **R1** — every venue adapter package directly under
//!   `adapters_root` ships a `Capabilities()` declaration
//!   (`capabilities_marker` found in a production `.go` file).
//!   Shared non-venue packages are listed in `exempt_packages`.
//!   A declaration with zero event types (`event_type_marker`
//!   occurrences inside the function body) is permitted only when
//!   the body carries an explicit justifying comment containing
//!   the word "empty" (ADR-0022 R4).
//! - **R2** — the gateway route file (`route_file`) registers the
//!   introspection path (`route_path`).
//! - **R3** — the producer-boundary guard file (`guard_file`)
//!   contains every call in `guard_calls` (the Allows() check and
//!   the undeclared-event counter increment), so gaps are rejected
//!   observably at the producer.
//!
//! Detection model mirrors check-subjects / check-instruments:
//! line-based scan, not AST — robust for the gofmt'd adapter shape
//! and cheap to run in the gate.

use std::fs;
use std::path::Path;

use serde::Deserialize;

use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};

const VENUE_PARITY_POLICY_PATH: &str = "tools/raccoon-cli/policies/venue_parity.toml";

#[derive(Debug, Deserialize)]
struct VenueParityPolicy {
    venue_parity: VenueParitySection,
}

#[derive(Debug, Deserialize)]
struct VenueParitySection {
    /// Directory whose immediate sub-packages are venue adapters.
    adapters_root: String,
    /// Line marker that opens the Capabilities() declaration.
    capabilities_marker: String,
    /// Sub-packages under adapters_root that are NOT venue adapters
    /// (shared types/helpers) and skip the R1 requirement.
    #[serde(default)]
    exempt_packages: Vec<String>,
    /// Marker counted inside the declaration body — one per declared
    /// event type (e.g. "Type:").
    event_type_marker: String,
    /// Production file that hosts the R3 producer-boundary guard.
    guard_file: String,
    /// Calls that must all appear in guard_file.
    guard_calls: Vec<String>,
    /// Route file that must register the R2 introspection path.
    route_file: String,
    /// The introspection path literal.
    route_path: String,
}

pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("check-venue-parity");

    let policy_path = project_root.join(VENUE_PARITY_POLICY_PATH);
    let policy: VenueParityPolicy = match fs::read_to_string(&policy_path) {
        Ok(raw) => match toml::from_str(&raw) {
            Ok(p) => {
                report.add(CheckResult::pass("policy-present"));
                p
            }
            Err(e) => {
                report.add(CheckResult::from_findings(
                    "policy-present",
                    vec![Finding::error(
                        "policy-parse",
                        format!("{}: {e}", policy_path.display()),
                    )],
                ));
                return Ok(report);
            }
        },
        Err(e) => {
            report.add(CheckResult::from_findings(
                "policy-present",
                vec![Finding::error(
                    "policy-missing",
                    format!("{}: {e}", policy_path.display()),
                )],
            ));
            return Ok(report);
        }
    };
    let policy = policy.venue_parity;

    check_capabilities_declared(project_root, &policy, &mut report)?;
    check_guard_wired(project_root, &policy, &mut report);
    check_route_registered(project_root, &policy, &mut report);

    Ok(report)
}

/// R1: every venue adapter package ships a Capabilities()
/// declaration; empty declarations need a justifying comment.
fn check_capabilities_declared(
    project_root: &Path,
    policy: &VenueParitySection,
    report: &mut Report,
) -> Result<()> {
    let root = project_root.join(&policy.adapters_root);
    if !root.is_dir() {
        report.add(CheckResult::skip(
            "adapters-root",
            format!("{} not found", root.display()),
        ));
        return Ok(());
    }
    report.add(CheckResult::pass("adapters-root"));

    let mut findings: Vec<Finding> = Vec::new();
    let mut adapters_checked: usize = 0;

    let mut entries: Vec<_> = fs::read_dir(&root)?
        .filter_map(|e| e.ok())
        .map(|e| e.path())
        .filter(|p| p.is_dir())
        .collect();
    entries.sort();

    for dir in entries {
        let pkg = match dir.file_name().and_then(|n| n.to_str()) {
            Some(n) => n.to_string(),
            None => continue,
        };
        if policy.exempt_packages.iter().any(|e| e == &pkg) {
            continue;
        }

        // A venue adapter package = directory with at least one
        // production .go file.
        let mut go_files: Vec<_> = Vec::new();
        for f in fs::read_dir(&dir)? {
            let f = f?;
            let name = f.file_name().to_string_lossy().to_string();
            if name.ends_with(".go") && !name.ends_with("_test.go") {
                go_files.push(f.path());
            }
        }
        if go_files.is_empty() {
            continue;
        }
        go_files.sort();
        adapters_checked += 1;

        let mut found_marker = false;
        for file in &go_files {
            let content = fs::read_to_string(file)?;
            if !content.contains(&policy.capabilities_marker) {
                continue;
            }
            found_marker = true;

            // Scan the declaration body: count event types; an empty
            // declaration requires a justifying comment.
            let mut depth: i64 = 0;
            let mut in_func = false;
            let mut event_types = 0usize;
            let mut has_empty_justification = false;
            for line in content.lines() {
                if !in_func && line.contains(&policy.capabilities_marker) {
                    in_func = true;
                    depth = 1;
                    continue;
                }
                if in_func {
                    if line.contains(&policy.event_type_marker) {
                        event_types += 1;
                    }
                    if line.trim_start().starts_with("//")
                        && line.to_lowercase().contains("empty")
                    {
                        has_empty_justification = true;
                    }
                    depth += brace_delta(line);
                    if depth <= 0 {
                        break;
                    }
                }
            }
            if event_types == 0 && !has_empty_justification {
                let rel = file
                    .strip_prefix(project_root)
                    .unwrap_or(file)
                    .display()
                    .to_string();
                findings.push(Finding::error(
                    "capabilities-declared",
                    format!(
                        "{rel}: Capabilities() declares zero event types without an \
                         explicit justifying comment (ADR-0022 R4 — empty declarations \
                         are permitted only with a comment explaining why)"
                    ),
                ));
            }
            break;
        }

        if !found_marker {
            findings.push(Finding::error(
                "capabilities-declared",
                format!(
                    "{}/{pkg}: venue adapter package has no Capabilities() declaration \
                     ({}) — every adapter ships one per ADR-0022 R1",
                    policy.adapters_root, policy.capabilities_marker
                ),
            ));
        }
    }

    if findings.is_empty() {
        report.add(CheckResult::pass("capabilities-declared"));
        report.add(CheckResult::pass(format!(
            "adapters-scanned ({adapters_checked} venue adapter packages)"
        )));
    } else {
        report.add(CheckResult::from_findings("capabilities-declared", findings));
    }
    Ok(())
}

/// R3: the producer-boundary guard file contains the Allows() check
/// and the undeclared-event counter increment.
fn check_guard_wired(project_root: &Path, policy: &VenueParitySection, report: &mut Report) {
    let path = project_root.join(&policy.guard_file);
    let content = match fs::read_to_string(&path) {
        Ok(c) => c,
        Err(e) => {
            report.add(CheckResult::from_findings(
                "producer-guard-wired",
                vec![Finding::error(
                    "producer-guard-wired",
                    format!("{}: {e}", path.display()),
                )],
            ));
            return;
        }
    };

    let mut findings: Vec<Finding> = Vec::new();
    for call in &policy.guard_calls {
        if !content.contains(call) {
            findings.push(Finding::error(
                "producer-guard-wired",
                format!(
                    "{}: missing `{call}` — undeclared events must be rejected \
                     observably at the producer (ADR-0022 R3)",
                    policy.guard_file
                ),
            ));
        }
    }
    if findings.is_empty() {
        report.add(CheckResult::pass("producer-guard-wired"));
    } else {
        report.add(CheckResult::from_findings("producer-guard-wired", findings));
    }
}

/// R2: the gateway route file registers the introspection path.
fn check_route_registered(project_root: &Path, policy: &VenueParitySection, report: &mut Report) {
    let path = project_root.join(&policy.route_file);
    match fs::read_to_string(&path) {
        Ok(content) if content.contains(&policy.route_path) => {
            report.add(CheckResult::pass("introspection-route-registered"));
        }
        Ok(_) => {
            report.add(CheckResult::from_findings(
                "introspection-route-registered",
                vec![Finding::error(
                    "introspection-route-registered",
                    format!(
                        "{}: does not register `{}` (ADR-0022 R2)",
                        policy.route_file, policy.route_path
                    ),
                )],
            ));
        }
        Err(e) => {
            report.add(CheckResult::from_findings(
                "introspection-route-registered",
                vec![Finding::error(
                    "introspection-route-registered",
                    format!("{}: {e}", path.display()),
                )],
            ));
        }
    }
}

/// Net brace depth contributed by a line (function-body tracking).
fn brace_delta(line: &str) -> i64 {
    let mut delta: i64 = 0;
    for ch in line.chars() {
        match ch {
            '{' => delta += 1,
            '}' => delta -= 1,
            _ => {}
        }
    }
    delta
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::path::Path;

    const ADAPTER_OK: &str = r#"package binances

func Capabilities() ports.Capabilities {
	return ports.Capabilities{
		Venue: instrument.VenueBinance,
		EventTypes: []ports.EventTypeSupport{
			{Type: "observation.trade", Contracts: []instrument.ContractType{instrument.ContractSpot}},
		},
		Contracts: []instrument.ContractType{instrument.ContractSpot},
	}
}
"#;

    fn write_policy(root: &Path) {
        let dir = root.join("tools/raccoon-cli/policies");
        fs::create_dir_all(&dir).unwrap();
        fs::write(
            dir.join("venue_parity.toml"),
            r#"[venue_parity]
adapters_root = "internal/adapters/exchanges"
capabilities_marker = "func Capabilities() ports.Capabilities {"
exempt_packages = []
event_type_marker = "Type:"
guard_file = "internal/actors/scopes/ingest/websocket_actor.go"
guard_calls = ["Allows(", "IncAdapterUndeclaredEvent("]
route_file = "internal/interfaces/http/routes/venues.go"
route_path = "/venues/capabilities"
"#,
        )
        .unwrap();
    }

    fn write_adapter(root: &Path, pkg: &str, body: &str) {
        let dir = root.join("internal/adapters/exchanges").join(pkg);
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join("capabilities.go"), body).unwrap();
    }

    fn write_guard(root: &Path, body: &str) {
        let dir = root.join("internal/actors/scopes/ingest");
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join("websocket_actor.go"), body).unwrap();
    }

    fn write_route(root: &Path, body: &str) {
        let dir = root.join("internal/interfaces/http/routes");
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join("venues.go"), body).unwrap();
    }

    fn write_full_fixture(root: &Path) {
        write_policy(root);
        write_adapter(root, "binances", ADAPTER_OK);
        write_guard(
            root,
            "package ingest\nfunc g() { if !caps.Allows(et, ct) { metrics.IncAdapterUndeclaredEvent(v, et, c) } }\n",
        );
        write_route(
            root,
            "package routes\nvar p = \"/venues/capabilities\"\n",
        );
    }

    #[test]
    fn coherent_surface_passes() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed, "expected pass: {report:?}");
    }

    #[test]
    fn adapter_without_capabilities_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        // A second adapter package with production code but no
        // Capabilities() declaration.
        let dir = tmp.path().join("internal/adapters/exchanges/bybit");
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join("parser.go"), "package bybit\nfunc f() {}\n").unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure: {report:?}");
    }

    #[test]
    fn exempt_package_is_skipped() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        // Override the policy with an exemption: a shared helpers
        // package under adapters_root has no Capabilities() function
        // and is skipped when declared in exempt_packages.
        let dir = tmp.path().join("tools/raccoon-cli/policies");
        fs::write(
            dir.join("venue_parity.toml"),
            r#"[venue_parity]
adapters_root = "internal/adapters/exchanges"
capabilities_marker = "func Capabilities() ports.Capabilities {"
exempt_packages = ["common"]
event_type_marker = "Type:"
guard_file = "internal/actors/scopes/ingest/websocket_actor.go"
guard_calls = ["Allows(", "IncAdapterUndeclaredEvent("]
route_file = "internal/interfaces/http/routes/venues.go"
route_path = "/venues/capabilities"
"#,
        )
        .unwrap();
        let shared = tmp.path().join("internal/adapters/exchanges/common");
        fs::create_dir_all(&shared).unwrap();
        fs::write(shared.join("helpers.go"), "package common\n").unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed, "expected pass: {report:?}");
    }

    #[test]
    fn empty_declaration_without_justification_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_adapter(
            tmp.path(),
            "coinbase",
            "package coinbase\n\nfunc Capabilities() ports.Capabilities {\n\treturn ports.Capabilities{Venue: instrument.VenueCoinbase}\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure: {report:?}");
    }

    #[test]
    fn empty_declaration_with_justifying_comment_passes() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_adapter(
            tmp.path(),
            "coinbase",
            "package coinbase\n\nfunc Capabilities() ports.Capabilities {\n\t// Intentionally empty declaration: parser not yet shipped;\n\t// adapter package holds shared metadata only.\n\treturn ports.Capabilities{Venue: instrument.VenueCoinbase}\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed, "expected pass: {report:?}");
    }

    #[test]
    fn missing_guard_call_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_guard(
            tmp.path(),
            "package ingest\nfunc g() { if !caps.Allows(et, ct) { return } }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure: {report:?}");
    }

    #[test]
    fn missing_route_path_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_full_fixture(tmp.path());
        write_route(tmp.path(), "package routes\n");
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure: {report:?}");
    }

    #[test]
    fn missing_policy_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed, "expected failure: {report:?}");
    }
}
