//! check-instruments — enforce ADR-0021 invariants:
//!
//! **H-6.a layer (adapter side):** every exchange adapter normalizes
//! venue-native symbols to CanonicalInstrument via the canonical
//! constructor at the adapter / domain boundary. Algorithm:
//!
//! 1. Read `tools/raccoon-cli/policies/adapters.toml` listing the
//!    recognized adapter packages.
//! 2. Walk `internal/adapters/exchanges/*/` — each subdirectory is
//!    an adapter package.
//! 3. For each adapter directory:
//!    - If NOT in the allowlist: emit a finding (unknown adapter
//!      must be declared before it can ship).
//!    - Else: scan the package's production `.go` files (excluding
//!      `*_test.go`) for an import of `internal/domain/instrument`
//!      AND a call to `instrument.New(` or `instrument.FromSymbol(`.
//!
//! **H-6.b layer (domain side):** every domain type undergoing the
//! Symbol → Instrument migration is declared in
//! `policies/domain_types.toml` with a `migration_state`. The analyzer
//! enforces that types marked `migrated` have both the canonical
//! Instrument field (`instrument.CanonicalInstrument` referenced in
//! the type file) and a transitory accessor `VenueSymbol() string`
//! (per the H-6.b sunset pattern, removed in H-6.f). Types marked
//! `pending` are tolerated — the legacy `Symbol string` field stays
//! until its own sub-onda migrates them. Types marked `string_filter`
//! are documented as query/filter DTOs whose venue-native string
//! field is canonical by design and intentionally not promoted to
//! CanonicalInstrument; declaration is the auditable record of that
//! architectural decision (pre-flight 6 of H-6.b'' confirmed the
//! pattern for `CrossSessionWindow`, where promoting would force
//! source-string reconstruction at the boundary — the exact regression
//! shape that caused commit 37f8ddd).
//!
//! **H-6.c layer (application side, anti-pattern detection):** every
//! function in production code that reconstructs CanonicalInstrument
//! from a source-string + symbol-string pair is declared in
//! `policies/anti_patterns.toml`. The analyzer scans production .go
//! files (recursively under `internal/`, excluding `_test.go`) for
//! call sites of each declared function and emits a finding per call
//! site. Severity is intentionally `warning` during H-6.c migration
//! so the gate does not fail while callers are migrated to Instrument
//! pass-through; H-6.f flips known patterns to `error` once the
//! helpers are eliminated. Pre-flight 5 of H-6.c discovered that
//! production Source values include `"binance"`, `"binance_spot"`,
//! `"derive"`, `"clickhouse"`, `"unknown_exchange"`, and
//! `"execute.venue-adapter"` — all unrecognized by the hardcoded
//! `binances`/`binancef` mapping in the helpers, so every call site
//! is a potential silent-zero regression analogous to commit
//! 37f8ddd. Per-call-site exceptions can be declared in the policy
//! file for legitimate boundary cases (helper that serves a true
//! HTTP-DTO boundary, etc.).
//!
//! Declarative-via-allowlist (adapter layer) and declarative-via-state
//! (domain layer) are preferred over pure inference because (a) new
//! migrations should not silently pass — they must be declared, which
//! makes the policy change auditable; and (b) regressions that drop
//! the canonical constructor or the VenueSymbol() accessor get caught
//! even if the file still compiles.
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
use crate::models::{CheckResult, CheckStatus, Finding, Report, Severity};
use serde::Deserialize;
use std::collections::{BTreeMap, BTreeSet};
use std::fs;
use std::path::{Path, PathBuf};

const POLICY_PATH: &str = "tools/raccoon-cli/policies/adapters.toml";
const DOMAIN_TYPES_POLICY_PATH: &str = "tools/raccoon-cli/policies/domain_types.toml";
const ANTI_PATTERNS_POLICY_PATH: &str = "tools/raccoon-cli/policies/anti_patterns.toml";
/// Root directory walked by the anti-patterns scan. Production Go
/// code lives under `internal/`; cmd/ is intentionally included via
/// the recursive walk below. Tests (`*_test.go`) are excluded inline.
const ANTI_PATTERNS_SCAN_ROOT: &str = "internal";
const ADAPTERS_DIR: &str = "internal/adapters/exchanges";
const INSTRUMENT_IMPORT: &str = "internal/domain/instrument";
const CONSTRUCTOR_NEW: &str = "instrument.New(";
const CONSTRUCTOR_FROM_SYMBOL: &str = "instrument.FromSymbol(";

const MIGRATION_STATE_MIGRATED: &str = "migrated";
const MIGRATION_STATE_PENDING: &str = "pending";
/// Documented architectural choice: the declared type is a query
/// filter / DTO whose Symbol or VenueSymbol field is venue-native by
/// design and intentionally not promoted to CanonicalInstrument.
/// Tolerated like `pending` (no enforcement) but conveys a permanent
/// decision rather than a transient state. See H-6.b'' / pre-flight 6.
const MIGRATION_STATE_STRING_FILTER: &str = "string_filter";
/// Substring that signals a CanonicalInstrument field in the type
/// file. Robust against gofmt column alignment because it matches
/// the type reference only, not the field name + whitespace.
const INSTRUMENT_FIELD_NEEDLE: &str = "instrument.CanonicalInstrument";
const VENUE_SYMBOL_METHOD_NEEDLE: &str = ") VenueSymbol() string";

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

/// Domain types policy schema. Each entry under `[domain_types.<key>]`
/// declares one domain type's migration state per ADR-0021. Loaded
/// from `tools/raccoon-cli/policies/domain_types.toml`.
#[derive(Debug, Deserialize, Default)]
struct DomainTypesPolicy {
    #[serde(default)]
    domain_types: BTreeMap<String, DomainTypeEntry>,
}

#[derive(Debug, Deserialize)]
struct DomainTypeEntry {
    /// Package path under internal/ (e.g., "internal/domain/observation").
    package: String,
    /// File name within the package (e.g., "trade.go").
    file: String,
    /// Go type name (e.g., "ObservationTrade").
    type_name: String,
    /// Migration state: "migrated" → enforced (must have Instrument
    /// field + VenueSymbol() method); "pending" → tolerated, no
    /// enforcement (transient); "string_filter" → tolerated, no
    /// enforcement (permanent — type is a query/filter DTO whose
    /// venue-native string is canonical by design).
    migration_state: String,
}

/// Anti-patterns policy schema. Each entry under `[[anti_patterns]]`
/// declares one forbidden source-string reconstruction function.
/// Loaded from `tools/raccoon-cli/policies/anti_patterns.toml`. The
/// analyzer scans production `.go` files (recursively under `internal/`,
/// excluding `_test.go`) for call sites of each declared function and
/// emits findings per call site.
#[derive(Debug, Deserialize, Default)]
struct AntiPatternsPolicy {
    #[serde(default)]
    anti_patterns: Vec<AntiPattern>,
}

#[derive(Debug, Deserialize)]
struct AntiPattern {
    /// Unique identifier surfaced in findings (e.g.,
    /// "source-string-instrument-reconstructor").
    name: String,
    /// Bare Go function name (no receiver). Used as the substring
    /// `"<function>("` in the line-based scan.
    function: String,
    /// Severity for emitted findings: "warning" (does not fail the
    /// gate) or "error" (fails the gate). H-6.c uses "warning"; H-6.f
    /// flips to "error" once the helpers are eliminated.
    severity: String,
    /// Rationale for the prohibition. Surfaced via `Finding::with_why`.
    #[serde(default)]
    why: String,
    /// Recommended migration path. Surfaced via `Finding::with_help`.
    #[serde(default)]
    help: String,
    /// Per-call-site exceptions — file paths (relative to project
    /// root) where the function call is allowed legitimately. Each
    /// exception is auditable: the policy file change shows up in PR
    /// review.
    #[serde(default)]
    exceptions: Vec<String>,
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

    // ── Domain types policy check (H-6.b extension) ────────────────
    //
    // Per ADR-0021 / H-6.b, each domain type undergoing canonical
    // instrument migration is declared in policies/domain_types.toml
    // with its migration_state. The analyzer enforces that "migrated"
    // types have both the canonical Instrument field and the
    // transitory VenueSymbol() method. "pending" types are tolerated.
    match load_domain_types_policy(project_root) {
        Ok(types_policy) => {
            report.add(CheckResult::pass("domain-types-policy-present"));
            let mut type_findings: Vec<Finding> = Vec::new();
            for (key, entry) in &types_policy.domain_types {
                match check_domain_type(project_root, key, entry) {
                    Ok(findings) => type_findings.extend(findings),
                    Err(e) => type_findings.push(Finding::error(
                        "domain-type-scan-error",
                        format!("{}: {}", key, e),
                    )),
                }
            }
            report.add(CheckResult::from_findings(
                "domain-type-migration-state",
                type_findings,
            ));
        }
        Err(finding) => {
            // Missing policy file is allowed (back-compat with
            // pre-H-6.b installs that have only adapters.toml).
            // But malformed TOML is an error.
            if finding.check == "policy-missing-tolerable" {
                report.add(CheckResult::skip(
                    "domain-types-policy-present",
                    "policies/domain_types.toml not present — skipping domain-type migration check (pre-H-6.b deployment)".to_string(),
                ));
            } else {
                report.add(CheckResult::from_findings(
                    "domain-types-policy-present",
                    vec![finding],
                ));
            }
        }
    }

    // ── Anti-patterns policy check (H-6.c.1 extension) ─────────────
    //
    // Per H-6.c.1 / Decision #4(c), each source-string instrument
    // reconstruction function is declared in policies/anti_patterns.toml
    // with a severity (warning during H-6.c migration; error after
    // H-6.f). The analyzer scans production .go files (recursively
    // under internal/, excluding _test.go) for call sites of each
    // declared function and emits a finding per call site. Per-site
    // exceptions are auditable via the policy file. Soft warnings
    // do not fail the gate; the visible findings drive the migration
    // effort.
    match load_anti_patterns_policy(project_root) {
        Ok(policy) => {
            report.add(CheckResult::pass("anti-patterns-policy-present"));
            let mut findings: Vec<Finding> = Vec::new();
            match collect_production_go_files(project_root) {
                Ok(files) => {
                    for pattern in &policy.anti_patterns {
                        findings.extend(scan_anti_pattern(project_root, &files, pattern));
                    }
                }
                Err(e) => findings.push(Finding::error(
                    "anti-patterns-scan-error",
                    format!("failed to walk {}: {}", ANTI_PATTERNS_SCAN_ROOT, e),
                )),
            }
            report.add(CheckResult::from_findings("anti-patterns-detected", findings));
        }
        Err(finding) => {
            if finding.check == "policy-missing-tolerable" {
                report.add(CheckResult::skip(
                    "anti-patterns-policy-present",
                    "policies/anti_patterns.toml not present — skipping anti-pattern scan (pre-H-6.c deployment)".to_string(),
                ));
            } else {
                report.add(CheckResult::from_findings(
                    "anti-patterns-policy-present",
                    vec![finding],
                ));
            }
        }
    }

    Ok(report)
}

// ── Domain types policy loading ────────────────────────────────────

fn load_domain_types_policy(
    project_root: &Path,
) -> std::result::Result<DomainTypesPolicy, Finding> {
    let path = project_root.join(DOMAIN_TYPES_POLICY_PATH);
    if !path.exists() {
        // Tolerable absence: pre-H-6.b deployments without the file.
        return Err(Finding::error(
            "policy-missing-tolerable",
            format!("{} not present", path.display()),
        ));
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
        .with_help("verify the file matches the documented schema (see comments in domain_types.toml)")
    })
}

// ── Per-type migration-state enforcement ───────────────────────────

fn check_domain_type(
    project_root: &Path,
    key: &str,
    entry: &DomainTypeEntry,
) -> std::result::Result<Vec<Finding>, String> {
    let mut findings = Vec::new();

    // Validate migration_state is a recognized value.
    let state = entry.migration_state.as_str();
    if state != MIGRATION_STATE_MIGRATED
        && state != MIGRATION_STATE_PENDING
        && state != MIGRATION_STATE_STRING_FILTER
    {
        findings.push(
            Finding::error(
                "unknown-migration-state",
                format!(
                    "{}: migration_state {:?} is not recognized (expected \"migrated\", \"pending\", or \"string_filter\")",
                    key, entry.migration_state
                ),
            )
            .with_help("update the entry to use a known migration_state"),
        );
        return Ok(findings);
    }

    // For "pending" we tolerate the legacy Symbol string; no
    // further enforcement applies. For "string_filter" we tolerate
    // a venue-native string field as a permanent architectural
    // choice (the type is a query/filter DTO — see H-6.b''
    // CrossSessionWindow); no further enforcement applies.
    if state == MIGRATION_STATE_PENDING || state == MIGRATION_STATE_STRING_FILTER {
        return Ok(findings);
    }

    // "migrated" path: the file must exist and contain both the
    // Instrument field and a VenueSymbol() string method.
    let file_path = project_root.join(&entry.package).join(&entry.file);
    if !file_path.exists() {
        findings.push(Finding::error(
            "domain-type-file-missing",
            format!(
                "{}: declared file {} does not exist",
                key,
                file_path.display()
            ),
        ));
        return Ok(findings);
    }
    let content = fs::read_to_string(&file_path)
        .map_err(|e| format!("read {}: {e}", file_path.display()))?;

    if !content.contains(INSTRUMENT_FIELD_NEEDLE) {
        findings.push(
            Finding::error(
                "missing-instrument-field",
                format!(
                    "{}: type {} declared migrated but {} does not reference {}",
                    key,
                    entry.type_name,
                    file_path.display(),
                    INSTRUMENT_FIELD_NEEDLE
                ),
            )
            .with_why("ADR-0021 / H-6.b: migrated domain types carry CanonicalInstrument as a first-class field")
            .with_help("add `Instrument instrument.CanonicalInstrument` to the struct, or mark the entry as \"pending\" until the migration lands"),
        );
    }

    if !content.contains(VENUE_SYMBOL_METHOD_NEEDLE) {
        findings.push(
            Finding::error(
                "missing-venue-symbol-method",
                format!(
                    "{}: type {} declared migrated but {} does not contain method `VenueSymbol() string`",
                    key, entry.type_name, file_path.display()
                ),
            )
            .with_why("ADR-0021 / H-6.b: migrated domain types expose VenueSymbol() — the transitory accessor that keeps venue-native readers compiling until H-6.f sunset")
            .with_help("add `func (x T) VenueSymbol() string { ... }` to the type, or mark the entry as \"pending\" until the migration lands"),
        );
    }

    Ok(findings)
}

// ── Anti-patterns policy loading + scanning (H-6.c.1) ───────────────

fn load_anti_patterns_policy(
    project_root: &Path,
) -> std::result::Result<AntiPatternsPolicy, Finding> {
    let path = project_root.join(ANTI_PATTERNS_POLICY_PATH);
    if !path.exists() {
        // Tolerable absence: pre-H-6.c deployments without the file.
        return Err(Finding::error(
            "policy-missing-tolerable",
            format!("{} not present", path.display()),
        ));
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
        .with_help(
            "verify the file matches the documented schema (see comments in anti_patterns.toml)",
        )
    })
}

/// Recursively collects all `.go` files under `internal/` (relative
/// to the project root), excluding `*_test.go`. Returns the absolute
/// paths so they can be opened directly; the project-relative paths
/// (computed during scan emission) are used in findings + exception
/// matching.
fn collect_production_go_files(project_root: &Path) -> std::result::Result<Vec<PathBuf>, String> {
    let root = project_root.join(ANTI_PATTERNS_SCAN_ROOT);
    if !root.is_dir() {
        return Ok(Vec::new());
    }
    let mut out: Vec<PathBuf> = Vec::new();
    walk_collect_go(&root, &mut out).map_err(|e| format!("walk: {e}"))?;
    out.sort();
    Ok(out)
}

fn walk_collect_go(dir: &Path, out: &mut Vec<PathBuf>) -> std::io::Result<()> {
    for entry in fs::read_dir(dir)? {
        let entry = entry?;
        let path = entry.path();
        if path.is_dir() {
            walk_collect_go(&path, out)?;
            continue;
        }
        let name = match path.file_name().and_then(|n| n.to_str()) {
            Some(n) => n,
            None => continue,
        };
        if !name.ends_with(".go") || name.ends_with("_test.go") {
            continue;
        }
        out.push(path);
    }
    Ok(())
}

/// Scans the collected production `.go` files for call sites of the
/// declared anti-pattern function. Skips function declaration lines
/// (`func ... funcName(...)`), single-line `//` comments, and any
/// file matching an entry in the `exceptions` list (path is matched
/// relative to project_root). Emits one `Finding` per call site at
/// the declared severity.
fn scan_anti_pattern(
    project_root: &Path,
    files: &[PathBuf],
    pattern: &AntiPattern,
) -> Vec<Finding> {
    let mut findings: Vec<Finding> = Vec::new();
    let needle = format!("{}(", pattern.function);
    let func_decl_prefix = format!("func {}(", pattern.function);
    let func_decl_with_receiver = format!(") {}(", pattern.function);

    for path in files {
        let rel_path = path
            .strip_prefix(project_root)
            .unwrap_or(path)
            .to_string_lossy()
            .to_string();
        if pattern.exceptions.iter().any(|ex| ex == &rel_path) {
            continue;
        }
        let content = match fs::read_to_string(path) {
            Ok(c) => c,
            Err(_) => continue,
        };
        for (lineno, line) in content.lines().enumerate() {
            let trimmed = line.trim_start();
            if trimmed.starts_with("//") {
                continue;
            }
            // Skip the function's own definition line (with or
            // without a receiver). The bare `func instrumentFromBinding(`
            // catches the no-receiver form (which is how the helpers
            // are declared in their respective packages).
            if trimmed.starts_with(&func_decl_prefix) {
                continue;
            }
            if trimmed.contains(&func_decl_with_receiver) {
                continue;
            }
            if !line.contains(&needle) {
                continue;
            }
            // Reduce false positives: require a non-identifier char
            // immediately before the function name so that a longer
            // identifier ending in `instrumentFromBinding` does not
            // trip the scan. (Conservative: bare match works for
            // exported & unexported names in current scope.)
            let needle_idx = match line.find(&needle) {
                Some(i) => i,
                None => continue,
            };
            if needle_idx > 0 {
                let prev = line.as_bytes()[needle_idx - 1] as char;
                if prev.is_alphanumeric() || prev == '_' {
                    continue;
                }
            }
            let location = format!("{}:{}", rel_path, lineno + 1);
            let mut finding = match pattern.severity.as_str() {
                "error" => Finding::error(pattern.name.as_str(), pattern.function.clone()),
                _ => Finding::warning(pattern.name.as_str(), pattern.function.clone()),
            };
            finding = finding.with_location(location);
            if !pattern.why.is_empty() {
                finding = finding.with_why(pattern.why.clone());
            }
            if !pattern.help.is_empty() {
                finding = finding.with_help(pattern.help.clone());
            }
            findings.push(finding);
        }
    }
    findings
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

    // ── Domain types policy tests (H-6.b extension) ─────────────

    fn write_domain_types_policy(root: &Path, body: &str) {
        write(&root.join(DOMAIN_TYPES_POLICY_PATH), body);
    }

    fn write_domain_type_file(root: &Path, pkg: &str, file: &str, contents: &str) {
        write(&root.join(pkg).join(file), contents);
    }

    #[test]
    fn analyze_domain_types_passes_when_migrated_type_has_field_and_method() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);

        write_domain_types_policy(
            root,
            r#"
[domain_types.observation_trade]
package = "internal/domain/observation"
file = "trade.go"
type_name = "ObservationTrade"
migration_state = "migrated"
"#,
        );
        write_domain_type_file(
            root,
            "internal/domain/observation",
            "trade.go",
            "package observation\n\ntype ObservationTrade struct { Instrument instrument.CanonicalInstrument }\n\nfunc (t ObservationTrade) VenueSymbol() string { return \"\" }\n",
        );

        let report = analyze(root).unwrap();
        assert!(report.passed(), "expected pass; got:\n{report}");
    }

    #[test]
    fn analyze_domain_types_fails_when_migrated_type_missing_venue_symbol() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);

        write_domain_types_policy(
            root,
            r#"
[domain_types.observation_trade]
package = "internal/domain/observation"
file = "trade.go"
type_name = "ObservationTrade"
migration_state = "migrated"
"#,
        );
        // Has Instrument field but missing VenueSymbol method.
        write_domain_type_file(
            root,
            "internal/domain/observation",
            "trade.go",
            "package observation\n\ntype ObservationTrade struct { Instrument instrument.CanonicalInstrument }\n",
        );

        let report = analyze(root).unwrap();
        assert!(!report.passed(), "expected fail; got:\n{report}");
        let s = format!("{report}");
        assert!(
            s.contains("missing-venue-symbol-method"),
            "expected missing-venue-symbol-method finding; got:\n{s}"
        );
    }

    #[test]
    fn analyze_domain_types_pending_state_tolerates_legacy_symbol() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);

        write_domain_types_policy(
            root,
            r#"
[domain_types.execution_intent]
package = "internal/domain/execution"
file = "execution.go"
type_name = "ExecutionIntent"
migration_state = "pending"
"#,
        );
        // Pending type: still has Symbol string, no Instrument field.
        write_domain_type_file(
            root,
            "internal/domain/execution",
            "execution.go",
            "package execution\n\ntype ExecutionIntent struct { Symbol string }\n",
        );

        let report = analyze(root).unwrap();
        assert!(report.passed(), "expected pass; pending tolerated. Got:\n{report}");
    }

    #[test]
    fn analyze_domain_types_unknown_state_fails() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);

        write_domain_types_policy(
            root,
            r#"
[domain_types.weird]
package = "internal/domain/weird"
file = "weird.go"
type_name = "Weird"
migration_state = "kind_of_migrated"
"#,
        );

        let report = analyze(root).unwrap();
        assert!(!report.passed());
        let s = format!("{report}");
        assert!(s.contains("unknown-migration-state"), "got:\n{s}");
        assert!(
            s.contains("string_filter"),
            "error help should list string_filter as a recognized state; got:\n{s}"
        );
    }

    #[test]
    fn analyze_domain_types_string_filter_state_tolerates_string_field() {
        // H-6.b'' / Decisão #2: query-filter / DTO types whose
        // VenueSymbol stays as a venue-native string are declared
        // with migration_state = "string_filter". The analyzer
        // tolerates them without requiring Instrument field or
        // VenueSymbol() method.
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);

        write_domain_types_policy(
            root,
            r#"
[domain_types.cross_session_window]
package = "internal/domain/pairing"
file = "continuity.go"
type_name = "CrossSessionWindow"
migration_state = "string_filter"
"#,
        );
        // string_filter type: carries a venue-native string field,
        // no Instrument, no VenueSymbol() method. Analyzer must
        // accept this shape without finding.
        write_domain_type_file(
            root,
            "internal/domain/pairing",
            "continuity.go",
            "package pairing\n\ntype CrossSessionWindow struct { VenueSymbol string }\n",
        );

        let report = analyze(root).unwrap();
        assert!(
            report.passed(),
            "expected pass (string_filter tolerated). Got:\n{report}"
        );
    }

    #[test]
    fn analyze_domain_types_tolerates_missing_policy_file() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        // No domain_types.toml: pre-H-6.b deployment shape.

        let report = analyze(root).unwrap();
        assert!(report.passed(), "expected pass (skip), got:\n{report}");
        let s = format!("{report}");
        assert!(
            s.contains("domain-types-policy-present") && s.contains("skip"),
            "expected skip on missing policy; got:\n{s}"
        );
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

    // ── Anti-patterns policy tests (H-6.c.1 extension) ─────────────

    fn write_anti_patterns_policy(root: &Path, body: &str) {
        write(&root.join(ANTI_PATTERNS_POLICY_PATH), body);
    }

    fn write_internal_go(root: &Path, rel_path: &str, contents: &str) {
        write(&root.join("internal").join(rel_path), contents);
    }

    /// Locate findings from the anti-patterns scan check in a report,
    /// counting warning/error severities separately.
    fn anti_pattern_findings(report: &Report) -> (usize, usize) {
        let check = report
            .checks
            .iter()
            .find(|c| c.name == "anti-patterns-detected")
            .expect("anti-patterns-detected check missing");
        let warnings = check
            .findings
            .iter()
            .filter(|f| f.severity == Severity::Warning)
            .count();
        let errors = check
            .findings
            .iter()
            .filter(|f| f.severity == Severity::Error)
            .count();
        (warnings, errors)
    }

    #[test]
    fn analyze_anti_patterns_emits_warning_per_call_site_and_does_not_fail_gate() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        write_anti_patterns_policy(
            root,
            r#"
[[anti_patterns]]
name = "source-string-instrument-reconstructor"
function = "instrumentFromBinding"
severity = "warning"
why = "Reconstructs CanonicalInstrument from source-string mapping."
help = "Migrate caller to receive Instrument from upstream."
exceptions = []
"#,
        );
        // 3 production call sites + 1 declaration (must be ignored) +
        // 1 test-file caller (must be ignored).
        write_internal_go(
            root,
            "application/signal/instrument_binding.go",
            "package signal\n\nfunc instrumentFromBinding(source, sym string) struct{} { return struct{}{} }\n",
        );
        write_internal_go(
            root,
            "application/signal/rsi_sampler.go",
            "package signal\n\nfunc NewRSI(source, sym string) {\n    _ = instrumentFromBinding(source, sym)\n}\n",
        );
        write_internal_go(
            root,
            "application/signal/atr_sampler.go",
            "package signal\n\nfunc NewATR(source, sym string) {\n    _ = instrumentFromBinding(source, sym)\n}\n",
        );
        write_internal_go(
            root,
            "application/signal/macd_sampler.go",
            "package signal\n\nfunc NewMACD(source, sym string) {\n    _ = instrumentFromBinding(source, sym)\n}\n",
        );
        // _test.go file with the same call — must be excluded.
        write(
            &root.join("internal/application/signal/rsi_sampler_test.go"),
            "package signal\n\nfunc TestFoo() {\n    _ = instrumentFromBinding(\"binances\", \"btcusdt\")\n}\n",
        );

        let report = analyze(root).unwrap();
        assert!(
            report.passed(),
            "soft-warning anti-patterns must not fail the gate. Got:\n{report}"
        );
        let (warnings, errors) = anti_pattern_findings(&report);
        assert_eq!(
            warnings, 3,
            "expected 3 warnings (one per call site, excluding declaration + _test.go). Got:\n{report}"
        );
        assert_eq!(errors, 0, "soft-warning mode must not emit errors");
    }

    #[test]
    fn analyze_anti_patterns_skips_function_declaration_line() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        write_anti_patterns_policy(
            root,
            r#"
[[anti_patterns]]
name = "test-pattern"
function = "instrumentFromBinding"
severity = "warning"
exceptions = []
"#,
        );
        // File contains ONLY the declaration. Should emit zero findings.
        write_internal_go(
            root,
            "application/signal/instrument_binding.go",
            "package signal\n\nfunc instrumentFromBinding(source, sym string) struct{} { return struct{}{} }\n",
        );

        let report = analyze(root).unwrap();
        let (warnings, errors) = anti_pattern_findings(&report);
        assert_eq!(warnings, 0, "declaration-only file must not produce findings");
        assert_eq!(errors, 0);
    }

    #[test]
    fn analyze_anti_patterns_exception_suppresses_warning() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        write_anti_patterns_policy(
            root,
            r#"
[[anti_patterns]]
name = "test-pattern"
function = "instrumentFromBinding"
severity = "warning"
exceptions = ["internal/application/execution/binance_spot_testnet_adapter.go"]
"#,
        );
        write_internal_go(
            root,
            "application/execution/binance_spot_testnet_adapter.go",
            "package execution\n\nfunc N() {\n    _ = instrumentFromBinding(\"binances\", \"btcusdt\")\n}\n",
        );
        write_internal_go(
            root,
            "application/signal/rsi_sampler.go",
            "package signal\n\nfunc N() {\n    _ = instrumentFromBinding(s, v)\n}\n",
        );

        let report = analyze(root).unwrap();
        let (warnings, errors) = anti_pattern_findings(&report);
        // Exception suppresses the testnet adapter caller; the signal
        // caller still emits a finding.
        assert_eq!(warnings, 1, "exception must suppress only the declared file");
        assert_eq!(errors, 0);
    }

    #[test]
    fn analyze_anti_patterns_error_severity_fails_gate() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        write_anti_patterns_policy(
            root,
            r#"
[[anti_patterns]]
name = "strict-pattern"
function = "instrumentFromBinding"
severity = "error"
exceptions = []
"#,
        );
        write_internal_go(
            root,
            "application/signal/rsi_sampler.go",
            "package signal\n\nfunc N() {\n    _ = instrumentFromBinding(s, v)\n}\n",
        );

        let report = analyze(root).unwrap();
        assert!(
            !report.passed(),
            "error-severity findings must fail the gate (H-6.f strict mode). Got:\n{report}"
        );
        let (warnings, errors) = anti_pattern_findings(&report);
        assert_eq!(warnings, 0);
        assert_eq!(errors, 1);
    }

    #[test]
    fn analyze_anti_patterns_tolerates_missing_policy_file() {
        // Pre-H-6.c deployments have no anti_patterns.toml. Skip with
        // an info message rather than failing.
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_policy(root, &["binances"]);
        write_adapter(root, "binances", COMPLIANT_SOURCE);
        // No anti_patterns.toml written.

        let report = analyze(root).unwrap();
        assert!(report.passed(), "missing policy must skip, not fail");
        let check = report
            .checks
            .iter()
            .find(|c| c.name == "anti-patterns-policy-present")
            .expect("anti-patterns-policy-present check missing");
        assert_eq!(check.status, CheckStatus::Skip);
    }
}
