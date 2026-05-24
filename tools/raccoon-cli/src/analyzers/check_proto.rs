//! check-proto — verify that proto/registry.json, .proto files under
//! proto/, and generated Go under internal/shared/contracts/ stay
//! in sync.
//!
//! Per ADR-0018 acceptance criterion 5 (post-2026-05-25 erratum):
//! "The raccoon-cli `check proto` analyzer statically validates that
//! proto/registry.json ↔ .proto files ↔ generated Go under
//! internal/shared/contracts/ stay in sync, and it is integrated
//! into `make verify`".
//!
//! Coverage:
//! - **Level B** (registry ↔ proto ↔ Go sync, validated):
//!   - Every registry entry references an existing .proto file.
//!   - Every .proto file in the tree has a registry entry.
//!   - Every registry entry has the corresponding generated .pb.go
//!     under internal/shared/contracts/<family>/v<n>/.
//!   - Every .proto declares `option go_package` matching the
//!     internal/shared/contracts/ path implied by its directory.
//! - **Level C smoke** (PROTO-G3 boundary, best-effort):
//!   - No Go file under internal/domain/ contains an import of
//!     `internal/shared/contracts`. This is a textual smoke check
//!     (substring match on import paths), not an AST-aware scan;
//!     deeper boundary enforcement may follow in a future onda.

use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};
use serde::Deserialize;
use std::collections::BTreeSet;
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Debug, Deserialize)]
struct Registry {
    schemas: Vec<RegistryEntry>,
}

#[derive(Debug, Deserialize)]
struct RegistryEntry {
    #[serde(rename = "type")]
    type_: String,
    version: u32,
    proto_file: String,
    #[allow(dead_code)] // exposed for future analyzer levels
    message: String,
    #[allow(dead_code)] // exposed for future analyzer levels (status taxonomy)
    status: String,
}

/// Entry point invoked from the CLI (`raccoon-cli check proto`) and
/// from the quality-gate pipeline (`gate::run`).
pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("check-proto");

    let proto_dir = project_root.join("proto");
    if !proto_dir.is_dir() {
        // Pre-H-3.a projects: proto/ does not exist. Skip cleanly
        // rather than erroring; the analyzer is harmless on projects
        // that have not yet adopted the proto contract layer.
        report.add(CheckResult::skip(
            "proto-dir",
            format!("proto/ directory not found at {}", proto_dir.display()),
        ));
        return Ok(report);
    }
    report.add(CheckResult::pass("proto-dir"));

    let registry_path = proto_dir.join("registry.json");
    let registry = match load_registry(&registry_path) {
        Ok(r) => {
            report.add(CheckResult::pass("registry-present"));
            r
        }
        Err(finding) => {
            report.add(CheckResult::from_findings("registry-present", vec![finding]));
            return Ok(report);
        }
    };

    report.add(check_registry_entries(
        &registry,
        &proto_dir,
        project_root,
    ));
    report.add(check_proto_files_registered(&registry, &proto_dir)?);
    report.add(check_go_package_options(&registry, &proto_dir));
    report.add(check_domain_boundary_smoke(project_root)?);

    Ok(report)
}

// ── Load + parse registry ───────────────────────────────────────────

fn load_registry(path: &Path) -> std::result::Result<Registry, Finding> {
    if !path.exists() {
        return Err(Finding::error(
            "check-proto",
            format!("proto/registry.json not found at {}", path.display()),
        )
        .with_why("the registry is the canonical inventory linking (type, version) to .proto files and generated Go (ADR-0018)")
        .with_help("create proto/registry.json or check --project-root"));
    }
    let raw = fs::read_to_string(path).map_err(|e| {
        Finding::error(
            "check-proto",
            format!("failed to read proto/registry.json: {e}"),
        )
    })?;
    serde_json::from_str(&raw).map_err(|e| {
        Finding::error(
            "check-proto",
            format!("proto/registry.json is not valid JSON or schema: {e}"),
        )
        .with_help("verify the file matches the schema documented in docs/GLOSSARY.md → 'Schema registry'")
    })
}

// ── Check: every registry entry has its .proto and .pb.go ───────────

fn check_registry_entries(
    registry: &Registry,
    proto_dir: &Path,
    project_root: &Path,
) -> CheckResult {
    let mut findings = Vec::new();
    for entry in &registry.schemas {
        let proto_path = proto_dir.join(&entry.proto_file);
        if !proto_path.exists() {
            findings.push(
                Finding::error(
                    "registry-entry-proto-missing",
                    format!(
                        "registry entry {} v{} references missing .proto file {}",
                        entry.type_, entry.version, entry.proto_file
                    ),
                )
                .with_why("the registry inventory must accurately reflect what exists on disk (ADR-0018 PROTO-G4)")
                .with_help(format!(
                    "create proto/{} or remove the entry from proto/registry.json",
                    entry.proto_file
                )),
            );
            continue;
        }

        // .pb.go path: strip .proto suffix from proto_file, append .pb.go.
        let pb_go_rel = entry.proto_file.trim_end_matches(".proto").to_string() + ".pb.go";
        let pb_go_path = project_root
            .join("internal/shared/contracts")
            .join(&pb_go_rel);
        if !pb_go_path.exists() {
            findings.push(
                Finding::error(
                    "registry-entry-pb-go-missing",
                    format!(
                        "registry entry {} v{} expects generated Go at internal/shared/contracts/{} but file is missing",
                        entry.type_, entry.version, pb_go_rel
                    ),
                )
                .with_why("registered schemas must have their generated Go tracked in the repo (ADR-0018 criterion 4, post-2026-05-25 erratum)")
                .with_help("run `make proto-gen` and commit the generated *.pb.go files"),
            );
        }
    }
    CheckResult::from_findings("registry-entries-have-files", findings)
}

// ── Check: every .proto in the tree is registered ───────────────────

fn check_proto_files_registered(registry: &Registry, proto_dir: &Path) -> Result<CheckResult> {
    let registered: BTreeSet<&str> = registry
        .schemas
        .iter()
        .map(|e| e.proto_file.as_str())
        .collect();

    let mut findings = Vec::new();
    let proto_files = list_proto_files(proto_dir)?;
    for path in proto_files {
        let rel = path
            .strip_prefix(proto_dir)
            .unwrap_or(&path)
            .to_string_lossy()
            .to_string();
        if !registered.contains(rel.as_str()) {
            findings.push(
                Finding::error(
                    "proto-orphan",
                    format!(
                        "proto/{} exists but has no entry in proto/registry.json",
                        rel
                    ),
                )
                .with_why("every .proto file must be registered so consumers can locate its (type, version) (ADR-0018 PROTO-G4)")
                .with_help(format!(
                    "add an entry for {} to proto/registry.json with type, version, message, status, introduced_at",
                    rel
                )),
            );
        }
    }
    Ok(CheckResult::from_findings(
        "proto-files-registered",
        findings,
    ))
}

// ── Check: option go_package matches expected internal/shared/contracts/ path ──

fn check_go_package_options(registry: &Registry, proto_dir: &Path) -> CheckResult {
    let mut findings = Vec::new();
    for entry in &registry.schemas {
        let proto_path = proto_dir.join(&entry.proto_file);
        if !proto_path.exists() {
            // already reported by check_registry_entries
            continue;
        }
        let content = match fs::read_to_string(&proto_path) {
            Ok(c) => c,
            Err(e) => {
                findings.push(Finding::error(
                    "proto-read",
                    format!("failed to read {}: {}", proto_path.display(), e),
                ));
                continue;
            }
        };

        let expected = expected_go_package(&entry.proto_file);
        let needle = format!("\"{}\"", expected);
        if !content.contains(&needle) {
            findings.push(
                Finding::error(
                    "go-package-mismatch",
                    format!(
                        "{} option go_package does not point to {}",
                        entry.proto_file, expected
                    ),
                )
                .with_why("generated Go must land under internal/shared/contracts/<family>/v<n>/ per ADR-0018 boundary (PROTO-G3)")
                .with_help(format!(
                    "set `option go_package = \"{}\";` in {}",
                    expected, entry.proto_file
                )),
            );
        }
    }
    CheckResult::from_findings("go-package-options", findings)
}

/// Compute the expected `option go_package` value for a registry entry's
/// `proto_file`. For `envelope/v1/envelope.proto` returns
/// `internal/shared/contracts/envelope/v1`.
fn expected_go_package(proto_file: &str) -> String {
    let path = Path::new(proto_file);
    let parent = path.parent().unwrap_or(Path::new(""));
    if parent.as_os_str().is_empty() {
        "internal/shared/contracts".to_string()
    } else {
        format!("internal/shared/contracts/{}", parent.display())
    }
}

// ── Level C smoke: domain layer must not import contracts ───────────

fn check_domain_boundary_smoke(project_root: &Path) -> Result<CheckResult> {
    let domain_dir = project_root.join("internal/domain");
    if !domain_dir.is_dir() {
        return Ok(CheckResult::skip(
            "domain-boundary-smoke",
            "internal/domain/ not found",
        ));
    }

    let mut findings = Vec::new();
    let go_files = list_go_files(&domain_dir)?;
    for path in go_files {
        let content = fs::read_to_string(&path).unwrap_or_default();
        // Textual smoke: import paths in Go appear inside quoted strings.
        // We match the substring `"internal/shared/contracts` to catch
        // both `"internal/shared/contracts/envelope/v1"` and aliased
        // imports like `envelopev1 "internal/shared/contracts/envelope/v1"`.
        if content.contains("\"internal/shared/contracts") {
            findings.push(
                Finding::error(
                    "domain-imports-contracts",
                    format!(
                        "{} imports internal/shared/contracts; domain layer must stay proto-free",
                        path.strip_prefix(project_root)
                            .unwrap_or(&path)
                            .display()
                    ),
                )
                .with_why("ADR-0018 PROTO-G3: internal/domain/ MUST NOT import proto-generated code")
                .with_help("introduce a domain-native type and translate at the adapter or contracts boundary"),
            );
        }
    }
    Ok(CheckResult::from_findings("domain-boundary-smoke", findings))
}

// ── Filesystem walkers ──────────────────────────────────────────────

fn list_proto_files(dir: &Path) -> Result<Vec<PathBuf>> {
    let mut out = Vec::new();
    walk_with_extension(dir, "proto", &mut out)?;
    out.sort();
    Ok(out)
}

fn list_go_files(dir: &Path) -> Result<Vec<PathBuf>> {
    let mut out = Vec::new();
    walk_with_extension(dir, "go", &mut out)?;
    out.sort();
    Ok(out)
}

fn walk_with_extension(dir: &Path, ext: &str, out: &mut Vec<PathBuf>) -> Result<()> {
    for entry in fs::read_dir(dir)? {
        let entry = entry?;
        let path = entry.path();
        if path.is_dir() {
            walk_with_extension(&path, ext, out)?;
        } else if path.extension().and_then(|e| e.to_str()) == Some(ext) {
            out.push(path);
        }
    }
    Ok(())
}

// ── Unit tests ──────────────────────────────────────────────────────

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

    fn write_valid_proto(dir: &Path, rel: &str, go_pkg: &str) {
        let proto = format!(
            "syntax = \"proto3\";\n\npackage test.v1;\n\noption go_package = \"{go_pkg}\";\n\n// Doc.\nmessage M {{}}\n"
        );
        write(&dir.join(rel), &proto);
    }

    fn write_valid_pb_go(dir: &Path, rel: &str) {
        write(&dir.join(rel), "package v1\n");
    }

    fn write_registry(dir: &Path, body: &str) {
        write(&dir.join("proto/registry.json"), body);
    }

    fn ok_project() -> TempDir {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_valid_proto(
            &root.join("proto"),
            "envelope/v1/envelope.proto",
            "internal/shared/contracts/envelope/v1",
        );
        write_valid_pb_go(
            &root.join("internal/shared/contracts"),
            "envelope/v1/envelope.pb.go",
        );
        write_registry(
            root,
            r#"{
              "schemas": [
                {
                  "type": "envelope",
                  "version": 1,
                  "proto_file": "envelope/v1/envelope.proto",
                  "message": "envelope.v1.Envelope",
                  "status": "draft",
                  "introduced_at": "2026-05-25"
                }
              ]
            }"#,
        );
        tmp
    }

    #[test]
    fn analyze_passes_on_aligned_project() {
        let tmp = ok_project();
        let report = analyze(tmp.path()).unwrap();
        assert!(
            report.passed(),
            "expected report to pass; got:\n{report}"
        );
    }

    #[test]
    fn analyze_skips_when_proto_dir_absent() {
        let tmp = TempDir::new().unwrap();
        let report = analyze(tmp.path()).unwrap();
        // Skipped check should not fail the report.
        assert!(report.passed());
    }

    #[test]
    fn analyze_fails_when_registry_entry_references_missing_proto() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_registry(
            root,
            r#"{
              "schemas": [
                {
                  "type": "envelope",
                  "version": 1,
                  "proto_file": "envelope/v1/envelope.proto",
                  "message": "envelope.v1.Envelope",
                  "status": "draft",
                  "introduced_at": "2026-05-25"
                }
              ]
            }"#,
        );
        fs::create_dir_all(root.join("proto/envelope/v1")).unwrap();
        // .proto deliberately missing
        let report = analyze(root).unwrap();
        assert!(!report.passed(), "expected report to fail");
        let serialized = format!("{report}");
        assert!(
            serialized.contains("missing .proto file"),
            "missing-proto finding not surfaced: {serialized}"
        );
    }

    #[test]
    fn analyze_fails_when_proto_has_no_registry_entry() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_valid_proto(
            &root.join("proto"),
            "envelope/v1/envelope.proto",
            "internal/shared/contracts/envelope/v1",
        );
        write_valid_proto(
            &root.join("proto"),
            "marketdata/v1/trade.proto",
            "internal/shared/contracts/marketdata/v1",
        );
        write_valid_pb_go(
            &root.join("internal/shared/contracts"),
            "envelope/v1/envelope.pb.go",
        );
        write_valid_pb_go(
            &root.join("internal/shared/contracts"),
            "marketdata/v1/trade.pb.go",
        );
        // Registry knows only about envelope; trade.proto is orphan.
        write_registry(
            root,
            r#"{
              "schemas": [
                {
                  "type": "envelope",
                  "version": 1,
                  "proto_file": "envelope/v1/envelope.proto",
                  "message": "envelope.v1.Envelope",
                  "status": "draft",
                  "introduced_at": "2026-05-25"
                }
              ]
            }"#,
        );
        let report = analyze(root).unwrap();
        assert!(!report.passed());
        let serialized = format!("{report}");
        assert!(
            serialized.contains("marketdata/v1/trade.proto") && serialized.contains("no entry"),
            "proto-orphan finding not surfaced: {serialized}"
        );
    }

    #[test]
    fn analyze_fails_when_pb_go_missing() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_valid_proto(
            &root.join("proto"),
            "envelope/v1/envelope.proto",
            "internal/shared/contracts/envelope/v1",
        );
        write_registry(
            root,
            r#"{
              "schemas": [
                {
                  "type": "envelope",
                  "version": 1,
                  "proto_file": "envelope/v1/envelope.proto",
                  "message": "envelope.v1.Envelope",
                  "status": "draft",
                  "introduced_at": "2026-05-25"
                }
              ]
            }"#,
        );
        // .pb.go deliberately missing
        let report = analyze(root).unwrap();
        assert!(!report.passed());
        let serialized = format!("{report}");
        assert!(
            serialized.contains("expects generated Go") && serialized.contains("envelope.pb.go"),
            "pb-go-missing finding not surfaced: {serialized}"
        );
    }

    #[test]
    fn analyze_fails_when_go_package_mismatches() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        write_valid_proto(
            &root.join("proto"),
            "envelope/v1/envelope.proto",
            "github.com/wrong/path/envelope/v1", // wrong go_package
        );
        write_valid_pb_go(
            &root.join("internal/shared/contracts"),
            "envelope/v1/envelope.pb.go",
        );
        write_registry(
            root,
            r#"{
              "schemas": [
                {
                  "type": "envelope",
                  "version": 1,
                  "proto_file": "envelope/v1/envelope.proto",
                  "message": "envelope.v1.Envelope",
                  "status": "draft",
                  "introduced_at": "2026-05-25"
                }
              ]
            }"#,
        );
        let report = analyze(root).unwrap();
        assert!(!report.passed());
        let serialized = format!("{report}");
        assert!(
            serialized.contains("go_package")
                && serialized.contains("internal/shared/contracts/envelope/v1"),
            "go-package finding not surfaced: {serialized}"
        );
    }

    #[test]
    fn analyze_fails_when_domain_imports_contracts() {
        let tmp = ok_project();
        let root = tmp.path();
        // Add an offending domain file.
        write(
            &root.join("internal/domain/observation/bad.go"),
            "package observation\n\nimport _ \"internal/shared/contracts/envelope/v1\"\n",
        );
        let report = analyze(root).unwrap();
        assert!(!report.passed());
        let serialized = format!("{report}");
        assert!(
            serialized.contains("imports internal/shared/contracts"),
            "boundary smoke finding not surfaced: {serialized}"
        );
    }

    #[test]
    fn expected_go_package_strips_basename() {
        assert_eq!(
            expected_go_package("envelope/v1/envelope.proto"),
            "internal/shared/contracts/envelope/v1"
        );
        assert_eq!(
            expected_go_package("marketdata/v1/trade.proto"),
            "internal/shared/contracts/marketdata/v1"
        );
    }

    #[test]
    fn analyze_fails_on_invalid_registry_json() {
        let tmp = TempDir::new().unwrap();
        let root = tmp.path();
        fs::create_dir_all(root.join("proto")).unwrap();
        write_registry(root, "{ not valid json");
        let report = analyze(root).unwrap();
        assert!(!report.passed());
    }
}
