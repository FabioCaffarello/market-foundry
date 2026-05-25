//! check-determinism — enforce ADR-0019 INV-D1 (domain purity).
//!
//! Scans `internal/domain/*.go` (excluding `*_test.go`) for direct
//! use of non-deterministic stdlib facilities. Production code in
//! the domain layer MUST source time via `clock.Clock`, randomness
//! via `random.Source`, and `context.Context` explicitly through
//! function parameters.
//!
//! # Scope: production code only (test files exempt)
//!
//! `*_test.go` files are excluded from this analyzer. Rationale:
//! the real enforcement for tests is the determinism gates
//! INV-D3/INV-D4 (golden tests + N=50 byte-stability). A test
//! using `time.Now` incorrectly flaps goldens — no static check
//! needed. This is a foundry-specific divergence from the raccoon
//! reference (whose `check-domain-isolation.sh` applies to all
//! `.go` files); rationale documented in ADR-0019 References
//! section.
//!
//! # Detection model
//!
//! The analyzer is line-based, not AST-aware. It catches obvious
//! violations cheaply; it does not attempt to disambiguate every
//! possible Go syntax form. Coverage:
//!
//! - **Banned imports**: any import whose canonical path matches
//!   a banned package (`math/rand`, `crypto/rand`).
//! - **Banned symbols**: any call site of `time.Now`, `time.Since`,
//!   `time.Until`, `time.Tick`, `os.Getenv`, `os.Args`,
//!   `context.Background`, `context.TODO`. The match is the
//!   substring `<symbol>(` outside of `//` line comments and
//!   string literals.
//!
//! False positive risk: a string literal containing the symbol
//! text would be flagged if the heuristic-based string-detector
//! misses it. False negatives risk: clever syntactic obfuscation
//! (e.g., aliasing imports, building call expressions
//! dynamically) is not detected. Both are acceptable for a
//! convention enforcement gate; reviewer + commit message
//! discipline backs the analyzer.

use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};
use std::fs;
use std::path::{Path, PathBuf};

const BANNED_IMPORTS: &[&str] = &["math/rand", "crypto/rand", "math/rand/v2"];

struct BannedSymbol {
    name: &'static str,
    hint: &'static str,
}

const BANNED_SYMBOLS: &[BannedSymbol] = &[
    BannedSymbol {
        name: "time.Now",
        hint: "accept clock.Clock via constructor and call clk.Now()",
    },
    BannedSymbol {
        name: "time.Since",
        hint: "compute via clock.Clock.Now().Sub(t)",
    },
    BannedSymbol {
        name: "time.Until",
        hint: "compute via t.Sub(clock.Clock.Now())",
    },
    BannedSymbol {
        name: "time.Tick",
        hint: "drive time externally; do not poll wall clock from domain",
    },
    BannedSymbol {
        name: "os.Getenv",
        hint: "inject configuration via constructor; do not read env from domain",
    },
    BannedSymbol {
        name: "os.Args",
        hint: "inject configuration via constructor",
    },
    BannedSymbol {
        name: "context.Background",
        hint: "accept context.Context as a function parameter",
    },
    BannedSymbol {
        name: "context.TODO",
        hint: "accept context.Context as a function parameter",
    },
];

/// Entry point invoked from the CLI (`raccoon-cli check determinism`)
/// and from the quality-gate pipeline (`gate::run`).
pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("check-determinism");

    let domain_dir = project_root.join("internal/domain");
    if !domain_dir.is_dir() {
        report.add(CheckResult::skip(
            "domain-dir",
            format!("internal/domain/ not found at {}", domain_dir.display()),
        ));
        return Ok(report);
    }
    report.add(CheckResult::pass("domain-dir"));

    let go_files = list_production_go_files(&domain_dir)?;

    let mut import_findings: Vec<Finding> = Vec::new();
    let mut symbol_findings: Vec<Finding> = Vec::new();

    for path in &go_files {
        let rel = path.strip_prefix(project_root).unwrap_or(path);
        let content = match fs::read_to_string(path) {
            Ok(c) => c,
            Err(_) => continue,
        };
        scan_imports(rel, &content, &mut import_findings);
        scan_symbols(rel, &content, &mut symbol_findings);
    }

    report.add(CheckResult::from_findings(
        "banned-imports",
        import_findings,
    ));
    report.add(CheckResult::from_findings(
        "banned-symbols",
        symbol_findings,
    ));

    Ok(report)
}

// ── File enumeration ────────────────────────────────────────────────

fn list_production_go_files(dir: &Path) -> Result<Vec<PathBuf>> {
    let mut out = Vec::new();
    walk(dir, &mut out)?;
    out.retain(|p| !is_test_file(p));
    out.sort();
    Ok(out)
}

fn is_test_file(path: &Path) -> bool {
    path.file_name()
        .and_then(|n| n.to_str())
        .map(|s| s.ends_with("_test.go"))
        .unwrap_or(false)
}

fn walk(dir: &Path, out: &mut Vec<PathBuf>) -> Result<()> {
    for entry in fs::read_dir(dir)? {
        let entry = entry?;
        let path = entry.path();
        if path.is_dir() {
            walk(&path, out)?;
        } else if path.extension().and_then(|e| e.to_str()) == Some("go") {
            out.push(path);
        }
    }
    Ok(())
}

// ── Scan: banned imports ────────────────────────────────────────────

fn scan_imports(path: &Path, content: &str, findings: &mut Vec<Finding>) {
    let mut in_import_block = false;
    for (lineno, line) in content.lines().enumerate() {
        let trimmed = line.trim();
        if trimmed.starts_with("//") {
            continue;
        }
        if trimmed.starts_with("import (") {
            in_import_block = true;
            continue;
        }
        if in_import_block && trimmed == ")" {
            in_import_block = false;
            continue;
        }
        let is_import_line =
            in_import_block || trimmed.starts_with("import \"") || trimmed.starts_with("import _ \"");
        if !is_import_line {
            continue;
        }
        for banned in BANNED_IMPORTS {
            let needle = format!("\"{banned}\"");
            if line.contains(&needle) {
                findings.push(banned_import_finding(path, lineno + 1, banned));
            }
        }
    }
}

fn banned_import_finding(path: &Path, line: usize, import: &str) -> Finding {
    let help = match import {
        "math/rand" | "math/rand/v2" | "crypto/rand" => {
            "inject internal/shared/random.Source via constructor"
        }
        _ => "inject the dependency via a port",
    };
    Finding::error(
        "banned-import",
        format!(
            "{}:{} imports {} in production code",
            path.display(),
            line,
            import
        ),
    )
    .with_why("ADR-0019 INV-D1: internal/domain/ must source non-determinism via ports (clock.Clock, random.Source)")
    .with_help(help)
}

// ── Scan: banned symbols ────────────────────────────────────────────

fn scan_symbols(path: &Path, content: &str, findings: &mut Vec<Finding>) {
    for (lineno, line) in content.lines().enumerate() {
        let stripped = strip_line_comment(line);
        if stripped.trim().is_empty() {
            continue;
        }
        for banned in BANNED_SYMBOLS {
            let needle = format!("{}(", banned.name);
            let mut search_from = 0usize;
            while let Some(found) = stripped[search_from..].find(&needle) {
                let idx = search_from + found;
                if !is_in_string(stripped, idx) && is_fresh_identifier(stripped, idx) {
                    findings.push(banned_symbol_finding(
                        path,
                        lineno + 1,
                        banned.name,
                        banned.hint,
                    ));
                    break;
                }
                search_from = idx + needle.len();
            }
        }
    }
}

fn banned_symbol_finding(path: &Path, line: usize, symbol: &str, hint: &str) -> Finding {
    Finding::error(
        "banned-symbol",
        format!(
            "{}:{} uses {} in production code",
            path.display(),
            line,
            symbol
        ),
    )
    .with_why("ADR-0019 INV-D1: domain purity forbids direct stdlib non-determinism")
    .with_help(hint)
}

/// Returns the prefix of `line` up to the first `//` that is not
/// inside a string literal, or the whole line if no such comment
/// is present.
fn strip_line_comment(line: &str) -> &str {
    let bytes = line.as_bytes();
    let mut in_string = false;
    let mut i = 0;
    while i + 1 < bytes.len() {
        let c = bytes[i] as char;
        if c == '\\' && in_string {
            i += 2;
            continue;
        }
        if c == '"' {
            in_string = !in_string;
        }
        if !in_string && c == '/' && bytes[i + 1] as char == '/' {
            return &line[..i];
        }
        i += 1;
    }
    line
}

/// Returns true if `idx` in `line` is inside a Go string literal
/// (between double quotes). Tracks escape sequences.
fn is_in_string(line: &str, idx: usize) -> bool {
    let bytes = line.as_bytes();
    let mut in_string = false;
    let mut i = 0;
    while i < idx && i < bytes.len() {
        let c = bytes[i] as char;
        if c == '\\' && in_string && i + 1 < bytes.len() {
            i += 2;
            continue;
        }
        if c == '"' {
            in_string = !in_string;
        }
        i += 1;
    }
    in_string
}

/// Returns true if the character immediately preceding `idx` in
/// `line` is NOT an identifier-continuation character (alphanumeric
/// or underscore). This filters matches like `mytime.Now(` where
/// `time.Now(` is actually a substring of a longer identifier.
fn is_fresh_identifier(line: &str, idx: usize) -> bool {
    if idx == 0 {
        return true;
    }
    let prev_byte = line.as_bytes()[idx - 1];
    let c = prev_byte as char;
    !(c.is_alphanumeric() || c == '_')
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

    fn write_domain_file(root: &Path, rel: &str, content: &str) {
        write(&root.join("internal/domain").join(rel), content);
    }

    fn project_root() -> TempDir {
        let tmp = TempDir::new().unwrap();
        // Pure domain file that passes; provides baseline.
        write_domain_file(
            tmp.path(),
            "pure/pure.go",
            "package pure\n\nfunc Greet() string { return \"hello\" }\n",
        );
        tmp
    }

    #[test]
    fn analyze_passes_on_pure_domain() {
        let tmp = project_root();
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed(), "expected pass; got:\n{report}");
    }

    #[test]
    fn analyze_skips_when_domain_dir_absent() {
        let tmp = TempDir::new().unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(report.passed());
    }

    #[test]
    fn analyze_detects_time_now_in_production() {
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "evil/evil.go",
            "package evil\n\nimport \"time\"\n\nfunc Stamp() time.Time { return time.Now() }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed(), "expected fail; got:\n{report}");
        let s = format!("{report}");
        assert!(s.contains("time.Now"), "expected time.Now finding; got:\n{s}");
    }

    #[test]
    fn analyze_ignores_time_now_in_test_files() {
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "evil/evil_test.go",
            "package evil\n\nimport \"time\"\n\nfunc TestStamp(t T) { _ = time.Now() }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(
            report.passed(),
            "test files must be exempt; got fail:\n{report}"
        );
    }

    #[test]
    fn analyze_detects_math_rand_import() {
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "evil/evil.go",
            "package evil\n\nimport \"math/rand\"\n\nfunc R() int { return rand.Intn(10) }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed());
        let s = format!("{report}");
        assert!(s.contains("math/rand"), "expected math/rand finding; got:\n{s}");
    }

    #[test]
    fn analyze_detects_crypto_rand_import_in_block() {
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "evil/evil.go",
            "package evil\n\nimport (\n\t\"crypto/rand\"\n)\n\nfunc R() {}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed());
        let s = format!("{report}");
        assert!(s.contains("crypto/rand"));
    }

    #[test]
    fn analyze_detects_os_getenv() {
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "evil/evil.go",
            "package evil\n\nimport \"os\"\n\nfunc Env() string { return os.Getenv(\"FOO\") }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed());
        let s = format!("{report}");
        assert!(s.contains("os.Getenv"));
    }

    #[test]
    fn analyze_detects_context_background() {
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "evil/evil.go",
            "package evil\n\nimport \"context\"\n\nfunc C() context.Context { return context.Background() }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report.passed());
        let s = format!("{report}");
        assert!(s.contains("context.Background"));
    }

    #[test]
    fn analyze_ignores_time_now_in_line_comment() {
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "ok/ok.go",
            "package ok\n\n// avoid time.Now() in this code\nfunc F() int { return 42 }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(
            report.passed(),
            "time.Now in a // comment must not flag; got:\n{report}"
        );
    }

    #[test]
    fn analyze_ignores_time_now_in_string_literal() {
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "ok/ok.go",
            "package ok\n\nfunc F() string { return \"use time.Now() carefully\" }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(
            report.passed(),
            "time.Now inside string literal must not flag; got:\n{report}"
        );
    }

    #[test]
    fn analyze_ignores_substring_in_longer_identifier() {
        // `mytime.Now(` contains `time.Now(` as a substring; the
        // analyzer must not flag it because `time.Now` is part of
        // a longer identifier.
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "ok/ok.go",
            "package ok\n\ntype T struct{}\nfunc (t T) Now() int { return 0 }\nvar mytime = T{}\nfunc F() int { return mytime.Now() }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(
            report.passed(),
            "mytime.Now must not flag; got:\n{report}"
        );
    }

    #[test]
    fn analyze_allows_time_type_usage_without_call() {
        // Using time.Time, time.Duration as types or in arithmetic
        // is permitted (purely value operations).
        let tmp = project_root();
        write_domain_file(
            tmp.path(),
            "ok/ok.go",
            "package ok\n\nimport \"time\"\n\ntype S struct { Ts time.Time; D time.Duration }\nfunc F(s S) bool { return s.Ts.IsZero() }\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(
            report.passed(),
            "time.Time as field type must not flag; got:\n{report}"
        );
    }
}
