//! check-subjects — H-6.e canonical subject-token invariant.
//!
//! Statically enforces the ADR-0009 erratum (2026-06-10, Onda H-6.e):
//! the `{symbol}` token of every published NATS subject is derived
//! exclusively via `CanonicalInstrument.SubjectToken()` — never via
//! the transitory `VenueSymbol()` helper and never hand-formatted.
//!
//! Declarative algorithm: reads `tools/raccoon-cli/policies/subjects.toml`
//! and scans production publisher files (`file_suffix`, excluding
//! `_test.go`) under `scan_root`. A subject-composition block starts
//! at a line containing `subject_marker` and ends when the call's
//! parentheses balance back to zero. Inside a block, any occurrence
//! of `forbidden_call` is an error finding.
//!
//! Scope is SUBJECTS ONLY by design (H-6.e Decisão #4): KV partition
//! keys, JetStream dedup keys, and log labels legitimately use
//! `VenueSymbol()` until sub-onda H-6.e.2 — this analyzer must not
//! flag them, which is why the scan is block-scoped rather than a
//! whole-file pattern match.
//!
//! Detection model mirrors check-metrics / check-instruments:
//! line-based scan, not AST — robust enough for the gofmt'd,
//! uniform publisher shape, and cheap to run in the gate.

use std::fs;
use std::path::{Path, PathBuf};

use serde::Deserialize;

use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};

const SUBJECTS_POLICY_PATH: &str = "tools/raccoon-cli/policies/subjects.toml";

#[derive(Debug, Deserialize)]
struct SubjectsPolicy {
    subjects: SubjectsSection,
}

#[derive(Debug, Deserialize)]
struct SubjectsSection {
    /// Directory walked for publisher files (e.g. "internal/adapters/nats").
    scan_root: String,
    /// Production file suffix that composes subjects (e.g. "publisher.go").
    file_suffix: String,
    /// Line marker that opens a subject-composition block.
    subject_marker: String,
    /// Call that must NOT appear inside a subject-composition block.
    forbidden_call: String,
}

pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("check-subjects");

    let policy_path = project_root.join(SUBJECTS_POLICY_PATH);
    let policy: SubjectsPolicy = match fs::read_to_string(&policy_path) {
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

    let scan_root = project_root.join(&policy.subjects.scan_root);
    if !scan_root.is_dir() {
        report.add(CheckResult::skip(
            "scan-root",
            format!("{} not found", scan_root.display()),
        ));
        return Ok(report);
    }
    report.add(CheckResult::pass("scan-root"));

    let mut files: Vec<PathBuf> = Vec::new();
    collect_publisher_files(&scan_root, &policy.subjects.file_suffix, &mut files)?;
    files.sort();

    let mut findings: Vec<Finding> = Vec::new();
    let mut blocks_scanned: usize = 0;

    for file in &files {
        let content = fs::read_to_string(file)?;
        let rel = file
            .strip_prefix(project_root)
            .unwrap_or(file)
            .display()
            .to_string();

        let mut depth: i64 = 0;
        let mut in_block = false;

        for (idx, line) in content.lines().enumerate() {
            let lineno = idx + 1;
            let trimmed = line.trim_start();
            if trimmed.starts_with("//") {
                continue;
            }

            if !in_block && line.contains(&policy.subjects.subject_marker) {
                in_block = true;
                blocks_scanned += 1;
                depth = 0;
            }

            if in_block {
                if line.contains(&policy.subjects.forbidden_call) {
                    findings.push(Finding::error(
                        "subject-composition",
                        format!(
                            "{rel}:{lineno}: subject composed via {} — the {{symbol}} \
                             token must come from Instrument.SubjectToken() \
                             (ADR-0009 erratum 2026-06-10, Onda H-6.e)",
                            policy.subjects.forbidden_call
                        ),
                    ));
                }
                depth += paren_delta(line);
                if depth <= 0 {
                    in_block = false;
                }
            }
        }
    }

    if findings.is_empty() {
        report.add(CheckResult::pass("subject-composition"));
        report.add(CheckResult::pass(format!(
            "subject-blocks-scanned ({} blocks in {} publisher files)",
            blocks_scanned,
            files.len()
        )));
    } else {
        report.add(CheckResult::from_findings("subject-composition", findings));
    }

    Ok(report)
}

/// Net parenthesis depth contributed by a line. The subject block
/// opens with `fmt.Sprintf(` (+1 net on the marker line) and closes
/// on the line where the call's parens balance back to zero.
fn paren_delta(line: &str) -> i64 {
    let mut delta: i64 = 0;
    for ch in line.chars() {
        match ch {
            '(' => delta += 1,
            ')' => delta -= 1,
            _ => {}
        }
    }
    delta
}

fn collect_publisher_files(
    dir: &Path,
    suffix: &str,
    out: &mut Vec<PathBuf>,
) -> std::io::Result<()> {
    for entry in fs::read_dir(dir)? {
        let entry = entry?;
        let path = entry.path();
        if path.is_dir() {
            collect_publisher_files(&path, suffix, out)?;
            continue;
        }
        let name = match path.file_name().and_then(|n| n.to_str()) {
            Some(n) => n,
            None => continue,
        };
        if !name.ends_with(suffix) || name.ends_with("_test.go") {
            continue;
        }
        out.push(path);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn write_policy(root: &Path) {
        let dir = root.join("tools/raccoon-cli/policies");
        fs::create_dir_all(&dir).unwrap();
        fs::write(
            dir.join("subjects.toml"),
            r#"[subjects]
scan_root = "internal/adapters/nats"
file_suffix = "publisher.go"
subject_marker = "subject := fmt.Sprintf("
forbidden_call = "VenueSymbol()"
"#,
        )
        .unwrap();
    }

    fn write_publisher(root: &Path, pkg: &str, body: &str) {
        let dir = root.join("internal/adapters/nats").join(pkg);
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join("publisher.go"), body).unwrap();
    }

    fn report_passed(report: &Report) -> bool {
        report.passed
    }

    #[test]
    fn clean_publisher_passes() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natssignal",
            "package natssignal\n\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s.%s.%s.%d\",\n\t\tspec.Subject,\n\t\tevent.Signal.Source,\n\t\tevent.Signal.Instrument.SubjectToken(),\n\t\tevent.Signal.Timeframe,\n\t)\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn venue_symbol_inside_subject_block_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natssignal",
            "package natssignal\n\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s.%s.%s.%d\",\n\t\tspec.Subject,\n\t\tevent.Signal.Source,\n\t\tevent.Signal.VenueSymbol(),\n\t\tevent.Signal.Timeframe,\n\t)\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report_passed(&report), "expected failure: {report:?}");
    }

    #[test]
    fn venue_symbol_outside_subject_block_is_tolerated() {
        // Dedup keys and log labels legitimately use VenueSymbol()
        // until H-6.e.2 — the analyzer must be block-scoped.
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natsevidence",
            "package natsevidence\n\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s.%s\",\n\t\tspec.Subject,\n\t\tevent.Candle.Instrument.SubjectToken(),\n\t)\n\tdedupKey := event.Candle.VenueSymbol() + \":\" + suffix\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn test_files_are_skipped() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        let dir = tmp.path().join("internal/adapters/nats/natssignal");
        fs::create_dir_all(&dir).unwrap();
        fs::write(
            dir.join("publisher.go"),
            "package natssignal\nfunc f() { subject := fmt.Sprintf(\"%s\", a.Instrument.SubjectToken()) }\n",
        )
        .unwrap();
        fs::write(
            dir.join("old_publisher_test.go"),
            "package natssignal\nfunc g() { subject := fmt.Sprintf(\"%s\", a.VenueSymbol()) }\n",
        )
        .unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn comment_lines_are_skipped() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natsrisk",
            "package natsrisk\n\n// subject := fmt.Sprintf( historically used VenueSymbol()\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s\", a.Instrument.SubjectToken())\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn missing_policy_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(!report_passed(&report), "expected failure: {report:?}");
    }

    #[test]
    fn missing_scan_root_is_skip_after_policy() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        let report = analyze(tmp.path()).unwrap();
        // policy-present passes; scan-root is a skip, not a failure.
        assert!(report_passed(&report), "expected pass/skip: {report:?}");
    }

    #[test]
    fn single_line_block_closes_same_line() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natsdecision",
            "package natsdecision\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s\", tok)\n\tx := event.D.VenueSymbol()\n\t_ = x\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }
}
