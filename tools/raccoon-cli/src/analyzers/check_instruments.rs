//! check-instruments — enforce ADR-0021 / H-6.a invariant
//! "every exchange adapter normalizes venue-native symbols to
//! CanonicalInstrument at the layer boundary".
//!
//! Algorithm is **declarative**, not inferred from code patterns:
//!
//! 1. Read `tools/raccoon-cli/policies/adapters.toml` listing the
//!    recognized adapter packages.
//! 2. Walk `internal/adapters/exchanges/*/` — each subdirectory is
//!    an adapter package.
//! 3. For each adapter directory:
//!    - If NOT in the allowlist: emit a finding (unknown adapter
//!      must be declared before it can ship).
//!    - Else: scan the package's production `.go` files (excluding
//!      `*_test.go`) for:
//!        - an import of `internal/domain/instrument`, AND
//!        - a call to `instrument.New(` or `instrument.FromSymbol(`.
//!      If either is missing, emit a finding pointing to the
//!      offending package.
//!
//! Declarative-via-allowlist is preferred over pure inference
//! because (a) a new adapter dropped under
//! `internal/adapters/exchanges/` without going through ADR-0021
//! should not silently pass — it must be declared, which makes
//! the policy change auditable; and (b) refactor of an existing
//! adapter that drops the canonical constructor is a regression
//! the analyzer must catch even if the file still compiles.
//!
//! Detection model mirrors check-metrics: line-based scan, not
//! AST-aware. False positive surface is small: the matched needles
//! are distinctive enough (`internal/domain/instrument`,
//! `instrument.New(`, `instrument.FromSymbol(`) that incidental
//! collisions in unrelated code are essentially nil. False
//! negative surface (renamed import alias) is acceptable for a
//! convention-enforcement gate; reviewer + commit discipline
//! backs it up.

use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};
use serde::Deserialize;
use std::collections::BTreeSet;
use std::fs;
use std::path::{Path, PathBuf};

const POLICY_PATH: &str = "tools/raccoon-cli/policies/adapters.toml";
const ADAPTERS_DIR: &str = "internal/adapters/exchanges";
const INSTRUMENT_IMPORT: &str = "internal/domain/instrument";
const CONSTRUCTOR_NEW: &str = "instrument.New(";
const CONSTRUCTOR_FROM_SYMBOL: &str = "instrument.FromSymbol(";

#[derive(Debug, Deserialize, Default)]
struct AdapterPolicy {
    adapters: AdaptersSection,
}

#[derive(Debug, Deserialize, Default)]
struct AdaptersSection {
    /// Adapter package names recognized as canonical-instrument-aware
    /// per ADR-0021. An adapter directory not appearing here is a
    /// fail-stop — the reviewer must explicitly add it.
    #[serde(default)]
    allowlist: Vec<String>,
}

/// Entry point invoked from the CLI (`raccoon-cli check instruments`)
/// and from the quality-gate pipeline (`gate::run`).
pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("check-instruments");

    let adapters_dir = project_root.join(ADAPTERS_DIR);
    if !adapters_dir.is_dir() {
        report.add(CheckResult::skip(
            "adapters-dir",
            format!(
                "{} not found at {}",
                ADAPTERS_DIR,
                adapters_dir.display()
            ),
        ));
        return Ok(report);
    }
    report.add(CheckResult::pass("adapters-dir"));

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

    let allowlist: BTreeSet<&str> =
        policy.adapters.allowlist.iter().map(String::as_str).collect();

    let adapters = match list_adapters(&adapters_dir) {
        Ok(a) => a,
        Err(e) => return Err(e),
    };

    let mut allowlist_findings: Vec<Finding> = Vec::new();
    let mut canonical_findings: Vec<Finding> = Vec::new();

    for adapter in &adapters {
        if !allowlist.contains(adapter.name.as_str()) {
            allowlist_findings.push(
                Finding::error(
                    "unknown-adapter",
                    format!(
                        "{}: adapter package is not declared in adapters.toml allowlist",
                        adapter
                            .dir
                            .strip_prefix(project_root)
                            .unwrap_or(&adapter.dir)
                            .display()
                    ),
                )
                .with_why(
                    "ADR-0021 / H-6.a: a new exchange adapter must be declared in tools/raccoon-cli/policies/adapters.toml before it can ship — the declaration is the auditable step that proves the adapter normalizes venue-native symbols to CanonicalInstrument",
                )
                .with_help(
                    "add the package name to [adapters] allowlist in tools/raccoon-cli/policies/adapters.toml, and ensure the adapter imports internal/domain/instrument and calls instrument.New(...) or instrument.FromSymbol(...)",
                ),
            );
            continue;
        }
        match adapter_uses_canonical_constructor(&adapter.dir) {
            Ok(true) => {}
            Ok(false) => canonical_findings.push(
                Finding::error(
                    "missing-canonical-constructor",
                    format!(
                        "{}: adapter does not normalize via the canonical instrument constructor. Expected an import of {} AND a call to {} or {}",
                        adapter
                            .dir
                            .strip_prefix(project_root)
                            .unwrap_or(&adapter.dir)
                            .display(),
                        INSTRUMENT_IMPORT,
                        CONSTRUCTOR_NEW,
                        CONSTRUCTOR_FROM_SYMBOL,
                    ),
                )
                .with_why(
                    "ADR-0021 invariant: adapters translate venue-native symbol shapes into CanonicalInstrument at the adapter / domain boundary. Bare struct-literal construction bypasses validation and is forbidden",
                )
                .with_help(
                    "in the adapter's Normalize (or equivalent) entry point, call instrument.New(base, quote, contract) — or instrument.FromSymbol(...) if you already hold a canonical symbol string — rather than constructing CanonicalInstrument{...} directly",
                ),
            ),
            Err(e) => canonical_findings.push(Finding::error(
                "scan-error",
                format!(
                    "{}: scan failed: {}",
                    adapter
                        .dir
                        .strip_prefix(project_root)
                        .unwrap_or(&adapter.dir)
                        .display(),
                    e
                ),
            )),
        }
    }

    report.add(CheckResult::from_findings(
        "adapter-allowlisted",
        allowlist_findings,
    ));
    report.add(CheckResult::from_findings(
        "adapter-uses-canonical-constructor",
        canonical_findings,
    ));

    Ok(report)
}

// ── Policy loading ──────────────────────────────────────────────────

fn load_policy(project_root: &Path) -> std::result::Result<AdapterPolicy, Finding> {
    let path = project_root.join(POLICY_PATH);
    if !path.exists() {
        return Err(Finding::error(
            "check-instruments",
            format!("policy file not found at {}", path.display()),
        )
        .with_why(
            "tools/raccoon-cli/policies/adapters.toml declares which exchange adapter packages are recognized as canonical-instrument-aware",
        )
        .with_help("create the policy file or check --project-root"));
    }
    let raw = fs::read_to_string(&path).map_err(|e| {
        Finding::error(
            "check-instruments",
            format!("failed to read {}: {e}", path.display()),
        )
    })?;
    toml::from_str(&raw).map_err(|e| {
        Finding::error(
            "check-instruments",
            format!("{} is not valid TOML or schema: {e}", path.display()),
        )
        .with_help("verify the file matches the documented schema (see comments in adapters.toml)")
    })
}

// ── Adapter enumeration ────────────────────────────────────────────

struct Adapter {
    name: String,
    dir: PathBuf,
}

fn list_adapters(adapters_dir: &Path) -> Result<Vec<Adapter>> {
    let mut out = Vec::new();
    for entry in fs::read_dir(adapters_dir)? {
        let entry = entry?;
        let path = entry.path();
        if !path.is_dir() {
            continue;
        }
        if let Some(name) = path.file_name().and_then(|n| n.to_str()) {
            out.push(Adapter {
                name: name.to_string(),
                dir: path,
            });
        }
    }
    out.sort_by(|a, b| a.name.cmp(&b.name));
    Ok(out)
}

// ── Canonical-constructor detection ────────────────────────────────

fn adapter_uses_canonical_constructor(
    adapter_dir: &Path,
) -> std::result::Result<bool, String> {
    let entries = match fs::read_dir(adapter_dir) {
        Ok(e) => e,
        Err(e) => return Err(format!("read_dir: {e}")),
    };

    let mut imports_instrument = false;
    let mut calls_constructor = false;

    for entry in entries.flatten() {
        let path = entry.path();
        if path.extension().and_then(|e| e.to_str()) != Some("go") {
            continue;
        }
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
        if content.contains(INSTRUMENT_IMPORT) {
            imports_instrument = true;
        }
        if content.contains(CONSTRUCTOR_NEW) || content.contains(CONSTRUCTOR_FROM_SYMBOL) {
            calls_constructor = true;
        }
        if imports_instrument && calls_constructor {
            return Ok(true);
        }
    }
    Ok(imports_instrument && calls_constructor)
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

    fn write_policy(root: &Path, allowlist: &[&str]) {
        let items = allowlist
            .iter()
            .map(|s| format!("\"{}\"", s))
            .collect::<Vec<_>>()
            .join(", ");
        let body = format!("[adapters]\nallowlist = [{}]\n", items);
        write(&root.join(POLICY_PATH), &body);
    }

    fn write_adapter(root: &Path, name: &str, file_contents: &str) {
        let dir = root.join(ADAPTERS_DIR).join(name);
        write(&dir.join("aggtrade.go"), file_contents);
    }

    const COMPLIANT_SOURCE: &str = "package x\n\nimport (\n\t\"internal/domain/instrument\"\n)\n\nfunc Normalize() { _ = instrument.New(\"BTC\", \"USDT\", instrument.ContractSpot) }\n";

    #[test]
    fn analyze_passes_when_adapter_uses_canonical_constructor() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        let report = analyze(root).unwrap();
        assert!(report.passed(), "expected pass; got:\n{report}");
    }

    #[test]
    fn analyze_passes_when_adapter_uses_from_symbol() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(
            root,
            "binances",
            "package x\n\nimport \"internal/domain/instrument\"\n\nfunc N() { _, _ = instrument.FromSymbol(\"BTC/USDT-spot\") }\n",
        );
        let report = analyze(root).unwrap();
        assert!(report.passed(), "expected pass; got:\n{report}");
    }

    #[test]
    fn analyze_fails_when_adapter_missing_import() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        // Calls instrument.New but doesn't import internal/domain/instrument
        // (synthetic test case — would not compile in Go).
        write_adapter(
            root,
            "binances",
            "package x\n\nfunc N() { _ = instrument.New(\"BTC\", \"USDT\", x) }\n",
        );
        let report = analyze(root).unwrap();
        assert!(!report.passed(), "expected fail; got:\n{report}");
        let s = format!("{report}");
        assert!(
            s.contains("missing-canonical-constructor"),
            "expected missing-canonical-constructor finding; got:\n{s}"
        );
    }

    #[test]
    fn analyze_fails_when_adapter_missing_constructor_call() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        // Imports instrument package but never calls New / FromSymbol.
        write_adapter(
            root,
            "binances",
            "package x\n\nimport \"internal/domain/instrument\"\n\nvar _ instrument.CanonicalInstrument\n",
        );
        let report = analyze(root).unwrap();
        assert!(!report.passed(), "expected fail; got:\n{report}");
        let s = format!("{report}");
        assert!(s.contains("missing-canonical-constructor"));
    }

    #[test]
    fn analyze_fails_when_adapter_not_in_allowlist() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        // Allowlist names binances, but a 'rogue' adapter directory
        // exists and is not declared — fail-stop.
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        write_adapter(root, "rogue", COMPLIANT_SOURCE);
        let report = analyze(root).unwrap();
        assert!(!report.passed(), "expected fail; got:\n{report}");
        let s = format!("{report}");
        assert!(
            s.contains("rogue") && s.contains("unknown-adapter"),
            "expected unknown-adapter finding for rogue; got:\n{s}"
        );
    }

    #[test]
    fn analyze_ignores_test_files() {
        // A canonical constructor call living only in _test.go does
        // not satisfy the invariant — production code must adopt it.
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        let dir = root.join(ADAPTERS_DIR).join("binances");
        write(
            &dir.join("aggtrade.go"),
            "package x\n\nfunc N() {}\n",
        );
        write(
            &dir.join("aggtrade_test.go"),
            "package x\n\nimport \"internal/domain/instrument\"\n\nfunc TestX() { _ = instrument.New(\"a\", \"b\", c) }\n",
        );
        let report = analyze(root).unwrap();
        assert!(
            !report.passed(),
            "expected fail — _test.go adoption must not count; got:\n{report}"
        );
    }

    #[test]
    fn analyze_skips_when_exchanges_dir_absent() {
        let tmp = TempDir::new().unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed());
    }

    #[test]
    fn analyze_fails_when_policy_file_missing() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        // No policy file written.
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        let report = analyze(root).unwrap();
        assert!(!report.passed(), "expected fail when policy missing");
        let s = format!("{report}");
        assert!(s.contains("policy file not found"));
    }

    #[test]
    fn analyze_handles_multiple_adapters_mixed() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances", "binancef"]);
        // 2 OK + 1 violation (in allowlist but no constructor) +
        // 1 unknown.
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        write_adapter(root, "binancef", COMPLIANT_SOURCE);
        write_adapter(
            root,
            "bybit",
            COMPLIANT_SOURCE, // compliant code but adapter not declared
        );
        write_adapter(
            root,
            "binances",
            COMPLIANT_SOURCE,
        );
        let report = analyze(root).unwrap();
        assert!(!report.passed());
        let s = format!("{report}");
        assert!(s.contains("bybit"), "expected bybit unknown-adapter; got:\n{s}");
        // Compliant adapters should not produce findings.
        assert!(!s.contains("binances:"));
        assert!(!s.contains("binancef:"));
    }
}
