//! check-metrics — enforce PROGRAM-0003 / ADR-0024 invariant
//! "every long-running cmd/*/main.go binary exposes /metrics".
//!
//! Algorithm is **declarative**, not inferred from code patterns:
//!
//! 1. Read `tools/raccoon-cli/policies/binaries.toml` listing the
//!    one-shot binaries that are exempt from the invariant.
//! 2. Walk `cmd/*/main.go` and identify every binary directory.
//! 3. For each binary:
//!    - If listed in `one_shot`: skip (exempt).
//!    - Else: scan all `.go` files in the binary's package for
//!      at least one of:
//!         - `healthz.NewHealthServer` (long-running pattern that
//!           auto-routes `/metrics` per
//!           `internal/shared/healthz/healthz.go:222`).
//!         - `mux.Handle(...)/metrics(...)` (explicit ServeMux
//!           registration of the metrics path).
//!         - `metrics.HandlerFunc` (the gateway pattern, registered
//!           via the routes table).
//!      If none match, emit a finding pointing to the offending
//!      binary directory.
//!
//! Declarative-via-allowlist is preferred over inference-by-pattern
//! because (a) refactor of a long-running binary that drops
//! `healthz.NewHealthServer` accidentally would otherwise be a
//! silent regression, and (b) introducing a future one-shot binary
//! forces a documented policy edit that the reviewer questions
//! rather than letting the analyzer "magically" pass because the
//! author skipped a code pattern.
//!
//! Future extension (not in H-5): validate labels declared in
//! `internal/shared/metrics/` against ADR-0024 MP-2's
//! permitted/prohibited list. The framework added here (read
//! policy file, walk binaries, emit findings) is the substrate for
//! that extension.

use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};
use serde::Deserialize;
use std::collections::BTreeSet;
use std::fs;
use std::path::{Path, PathBuf};

const POLICY_PATH: &str = "tools/raccoon-cli/policies/binaries.toml";

#[derive(Debug, Deserialize, Default)]
struct BinaryPolicy {
    binaries: BinariesSection,
}

#[derive(Debug, Deserialize, Default)]
struct BinariesSection {
    /// Binaries with no HTTP server; /metrics is not applicable.
    #[serde(default)]
    one_shot: Vec<String>,

    /// Binaries whose /metrics registration happens in an
    /// imported package, not in their own main package. The
    /// analyzer scans only the main package's source files;
    /// these binaries are trusted to register /metrics
    /// transitively. A future refactor may replace this list
    /// with transitive import closure scanning. See ADR-0024
    /// References.
    #[serde(default)]
    transitive_registration: Vec<String>,
}

/// Entry point invoked from the CLI (`raccoon-cli check metrics`)
/// and from the quality-gate pipeline (`gate::run`).
pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("check-metrics");

    let cmd_dir = project_root.join("cmd");
    if !cmd_dir.is_dir() {
        report.add(CheckResult::skip(
            "cmd-dir",
            format!("cmd/ directory not found at {}", cmd_dir.display()),
        ));
        return Ok(report);
    }
    report.add(CheckResult::pass("cmd-dir"));

    let policy = match load_policy(project_root) {
        Ok(p) => {
            report.add(CheckResult::pass("policy-present"));
            p
        }
        Err(finding) => {
            report.add(CheckResult::from_findings(
                "policy-present",
                vec![finding],
            ));
            return Ok(report);
        }
    };

    let one_shot: BTreeSet<&str> =
        policy.binaries.one_shot.iter().map(String::as_str).collect();
    let transitive: BTreeSet<&str> = policy
        .binaries
        .transitive_registration
        .iter()
        .map(String::as_str)
        .collect();

    let binaries = match list_binaries(&cmd_dir) {
        Ok(b) => b,
        Err(e) => {
            return Err(e);
        }
    };

    let mut findings: Vec<Finding> = Vec::new();
    for bin in &binaries {
        if one_shot.contains(bin.name.as_str()) {
            continue;
        }
        if transitive.contains(bin.name.as_str()) {
            // Declared as registering /metrics transitively via an
            // imported package; reviewer audits the listing.
            continue;
        }
        match binary_exposes_metrics(&bin.dir) {
            Ok(true) => {}
            Ok(false) => findings.push(
                Finding::error(
                    "missing-metrics-endpoint",
                    format!(
                        "{}: long-running binary does not expose /metrics. Expected one of: healthz.NewHealthServer, mux.Handle(\"GET /metrics\", ...), metrics.HandlerFunc",
                        bin.dir
                            .strip_prefix(project_root)
                            .unwrap_or(&bin.dir)
                            .display()
                    ),
                )
                .with_why("ADR-0024 metrics policy + PROGRAM-0003 acceptance: every long-running binary MUST expose /metrics")
                .with_help(
                    "either add `healthz.NewHealthServer` to wire the standard health/metrics server; or, if /metrics is registered in an imported package, list this binary in tools/raccoon-cli/policies/binaries.toml under [binaries] transitive_registration; or, if the binary is genuinely one-shot (no HTTP surface), list it under [binaries] one_shot",
                ),
            ),
            Err(e) => findings.push(
                Finding::error(
                    "scan-error",
                    format!(
                        "{}: scan failed: {}",
                        bin.dir
                            .strip_prefix(project_root)
                            .unwrap_or(&bin.dir)
                            .display(),
                        e
                    ),
                ),
            ),
        }
    }

    report.add(CheckResult::from_findings(
        "binary-exposes-metrics",
        findings,
    ));

    Ok(report)
}

// ── Policy loading ──────────────────────────────────────────────────

fn load_policy(project_root: &Path) -> std::result::Result<BinaryPolicy, Finding> {
    let path = project_root.join(POLICY_PATH);
    if !path.exists() {
        return Err(Finding::error(
            "check-metrics",
            format!("policy file not found at {}", path.display()),
        )
        .with_why("tools/raccoon-cli/policies/binaries.toml declares which binaries are exempt from the /metrics invariant")
        .with_help("create the policy file or check --project-root"));
    }
    let raw = fs::read_to_string(&path).map_err(|e| {
        Finding::error(
            "check-metrics",
            format!("failed to read {}: {e}", path.display()),
        )
    })?;
    toml::from_str(&raw).map_err(|e| {
        Finding::error(
            "check-metrics",
            format!("{} is not valid TOML or schema: {e}", path.display()),
        )
        .with_help("verify the file matches the documented schema (see comments in binaries.toml)")
    })
}

// ── Binary enumeration ─────────────────────────────────────────────

struct Binary {
    name: String,
    dir: PathBuf,
}

fn list_binaries(cmd_dir: &Path) -> Result<Vec<Binary>> {
    let mut out = Vec::new();
    for entry in fs::read_dir(cmd_dir)? {
        let entry = entry?;
        let path = entry.path();
        if !path.is_dir() {
            continue;
        }
        let main_go = path.join("main.go");
        if !main_go.exists() {
            continue;
        }
        if let Some(name) = path.file_name().and_then(|n| n.to_str()) {
            out.push(Binary {
                name: name.to_string(),
                dir: path,
            });
        }
    }
    out.sort_by(|a, b| a.name.cmp(&b.name));
    Ok(out)
}

// ── Metric-exposure detection ──────────────────────────────────────

fn binary_exposes_metrics(binary_dir: &Path) -> std::result::Result<bool, String> {
    // Scan all .go files in the binary's own package (cmd/<name>/*.go)
    // for any of the canonical registration patterns. The .go file
    // walk does NOT recurse — only the binary's own package is the
    // composition root.
    let entries = match fs::read_dir(binary_dir) {
        Ok(e) => e,
        Err(e) => return Err(format!("read_dir: {e}")),
    };

    for entry in entries.flatten() {
        let path = entry.path();
        if path.extension().and_then(|e| e.to_str()) != Some("go") {
            continue;
        }
        // _test.go files do not count as production composition.
        if path
            .file_name()
            .and_then(|n| n.to_str())
            .map(|s| s.ends_with("_test.go"))
            .unwrap_or(false)
        {
            continue;
        }
        let content = match fs::read_to_string(&path) {
            Ok(c) => c,
            Err(_) => continue,
        };
        if content.contains("healthz.NewHealthServer")
            || content.contains("metrics.HandlerFunc")
            || (content.contains("/metrics") && content.contains("mux.Handle"))
        {
            return Ok(true);
        }
    }
    Ok(false)
}

// ── Unit tests ─────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    fn write(path: &Path, content: &str) {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).unwrap();
        }
        fs::write(path, content).unwrap();
    }

    fn write_policy(root: &Path, one_shot: &[&str]) {
        write_full_policy(root, one_shot, &[]);
    }

    fn write_full_policy(root: &Path, one_shot: &[&str], transitive: &[&str]) {
        let one_shot_items = one_shot
            .iter()
            .map(|s| format!("\"{}\"", s))
            .collect::<Vec<_>>()
            .join(", ");
        let transitive_items = transitive
            .iter()
            .map(|s| format!("\"{}\"", s))
            .collect::<Vec<_>>()
            .join(", ");
        let body = format!(
            "[binaries]\none_shot = [{}]\ntransitive_registration = [{}]\n",
            one_shot_items, transitive_items,
        );
        write(&root.join(POLICY_PATH), &body);
    }

    fn write_binary(root: &Path, name: &str, file_contents: &str) {
        let dir = root.join("cmd").join(name);
        write(&dir.join("main.go"), "package main\n\nfunc main() {}\n");
        write(&dir.join("run.go"), file_contents);
    }

    #[test]
    fn analyze_passes_when_binary_uses_health_server() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &[]);
        write_binary(
            root,
            "ingest",
            "package main\nfunc Run() { healthz.NewHealthServer(addr, nil, nil) }\n",
        );
        let report = analyze(root).unwrap();
        assert!(report.passed(), "expected pass; got:\n{report}");
    }

    #[test]
    fn analyze_passes_when_binary_registers_metrics_route() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &[]);
        write_binary(
            root,
            "gateway",
            "package main\n// route table includes metrics.HandlerFunc()\nfunc Run() { _ = metrics.HandlerFunc }\n",
        );
        let report = analyze(root).unwrap();
        assert!(report.passed(), "expected pass; got:\n{report}");
    }

    #[test]
    fn analyze_passes_when_binary_uses_mux_handle_metrics() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &[]);
        write_binary(
            root,
            "custom",
            "package main\nfunc Run() { mux.Handle(\"GET /metrics\", h) }\n",
        );
        let report = analyze(root).unwrap();
        assert!(report.passed(), "expected pass; got:\n{report}");
    }

    #[test]
    fn analyze_fails_when_long_running_binary_missing_metrics() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &[]);
        write_binary(
            root,
            "rogue",
            "package main\nfunc Run() { /* no /metrics anywhere */ }\n",
        );
        let report = analyze(root).unwrap();
        assert!(!report.passed());
        let s = format!("{report}");
        assert!(
            s.contains("rogue") && s.contains("missing-metrics-endpoint"),
            "expected missing-metrics-endpoint finding for rogue; got:\n{s}"
        );
    }

    #[test]
    fn analyze_exempts_one_shot_binary() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["migrate"]);
        write_binary(
            root,
            "migrate",
            "package main\nfunc Run() { /* one-shot CLI, no HTTP */ }\n",
        );
        let report = analyze(root).unwrap();
        assert!(
            report.passed(),
            "expected pass — migrate listed as one_shot; got:\n{report}"
        );
    }

    #[test]
    fn analyze_fails_when_policy_file_missing() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        // No policy file written.
        write_binary(
            root,
            "ingest",
            "package main\nfunc Run() { healthz.NewHealthServer(addr, nil, nil) }\n",
        );
        let report = analyze(root).unwrap();
        assert!(!report.passed(), "expected fail when policy missing");
        let s = format!("{report}");
        assert!(s.contains("policy file not found"));
    }

    #[test]
    fn analyze_skips_when_cmd_dir_absent() {
        let tmp = TempDir::new().unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed());
    }

    #[test]
    fn analyze_handles_multiple_binaries_mixed() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["migrate"]);
        // 3 OK + 1 exempt + 1 violation.
        write_binary(
            root,
            "ingest",
            "package main\nfunc Run() { healthz.NewHealthServer(addr, nil, nil) }\n",
        );
        write_binary(
            root,
            "derive",
            "package main\nfunc Run() { healthz.NewHealthServer(addr, nil, nil) }\n",
        );
        write_binary(
            root,
            "gateway",
            "package main\nfunc Run() { _ = metrics.HandlerFunc }\n",
        );
        write_binary(
            root,
            "migrate",
            "package main\nfunc Run() { /* one-shot */ }\n",
        );
        write_binary(
            root,
            "rogue",
            "package main\nfunc Run() { /* no /metrics */ }\n",
        );
        let report = analyze(root).unwrap();
        assert!(!report.passed());
        let s = format!("{report}");
        assert!(s.contains("rogue"), "expected rogue violation; got:\n{s}");
        // Other binaries should not appear as failures.
        assert!(!s.contains("ingest: long-running binary does not"));
        assert!(!s.contains("gateway: long-running binary does not"));
    }

    #[test]
    fn analyze_exempts_transitive_registration_binary() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_full_policy(root, &[], &["gateway"]);
        // gateway's main package contains no metrics references —
        // the route registration lives in an imported package.
        write_binary(
            root,
            "gateway",
            "package main\n// routes.DefaultRoutes(deps) is the entry point\nfunc Run() {}\n",
        );
        let report = analyze(root).unwrap();
        assert!(
            report.passed(),
            "expected pass — gateway listed as transitive_registration; got:\n{report}"
        );
    }

    #[test]
    fn analyze_ignores_test_go_files() {
        // If only a _test.go file contains healthz.NewHealthServer
        // but no production .go file does, the binary is in
        // violation. _test.go references do not satisfy the
        // production exposure invariant.
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &[]);
        let dir = root.join("cmd").join("trick");
        write(&dir.join("main.go"), "package main\nfunc main() {}\n");
        write(
            &dir.join("trick_test.go"),
            "package main\n// healthz.NewHealthServer present only in test\n",
        );
        let report = analyze(root).unwrap();
        assert!(
            !report.passed(),
            "expected fail — test file references do not count; got:\n{report}"
        );
    }
}
