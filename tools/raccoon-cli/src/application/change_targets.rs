use crate::io;
use serde::Serialize;
use std::path::Path;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "snake_case")]
pub(crate) enum ChangeTargetSource {
    Explicit,
    GitStatusStructural,
    GitStatusAll,
    GitStatusClean,
    NotGitRepository,
    GitUnavailable,
}

impl ChangeTargetSource {
    pub(crate) fn as_str(self) -> &'static str {
        match self {
            ChangeTargetSource::Explicit => "explicit",
            ChangeTargetSource::GitStatusStructural => "git_status_structural",
            ChangeTargetSource::GitStatusAll => "git_status_all",
            ChangeTargetSource::GitStatusClean => "git_status_clean",
            ChangeTargetSource::NotGitRepository => "not_git_repository",
            ChangeTargetSource::GitUnavailable => "git_unavailable",
        }
    }
}

#[derive(Debug, Clone, Serialize)]
pub(crate) struct ChangeTargetResolution {
    pub(crate) targets: Vec<String>,
    pub(crate) source: ChangeTargetSource,
    pub(crate) observed_changed_paths: usize,
    pub(crate) structural_target_count: usize,
    pub(crate) note: String,
}

impl ChangeTargetResolution {
    pub(crate) fn scope_note(&self, analysis_scope: &str) -> String {
        match self.source {
            ChangeTargetSource::Explicit => format!(
                "Input source: explicit {analysis_scope} provided by the caller. {}",
                self.note
            ),
            ChangeTargetSource::GitStatusStructural => format!(
                "Input source: auto-detected from git status and filtered to {} structural target(s) out of {} changed path(s). {}",
                self.structural_target_count, self.observed_changed_paths, self.note
            ),
            ChangeTargetSource::GitStatusAll => format!(
                "Input source: auto-detected from git status; no structural-only matches were found, so all {} changed path(s) were kept. {}",
                self.observed_changed_paths, self.note
            ),
            ChangeTargetSource::GitStatusClean => format!(
                "Input source: git status reported a clean worktree, so no {analysis_scope} were auto-detected. {}",
                self.note
            ),
            ChangeTargetSource::NotGitRepository => format!(
                "Input source: auto-detection unavailable because the project root is not a git repository. {}",
                self.note
            ),
            ChangeTargetSource::GitUnavailable => format!(
                "Input source: auto-detection unavailable because git could not be executed. {}",
                self.note
            ),
        }
    }
}

pub(crate) fn resolve_change_targets(
    project_root: &Path,
    explicit: &[String],
) -> ChangeTargetResolution {
    if !explicit.is_empty() {
        return ChangeTargetResolution {
            targets: explicit.to_vec(),
            source: ChangeTargetSource::Explicit,
            observed_changed_paths: explicit.len(),
            structural_target_count: explicit.len(),
            note: "Predictability is highest when critical commands receive explicit file or target arguments.".into(),
        };
    }

    resolve_from_probe(project_root, io::status_porcelain(project_root))
}

fn resolve_from_probe(_project_root: &Path, probe: io::GitStatusProbe) -> ChangeTargetResolution {
    match probe {
        io::GitStatusProbe::Changed(detected) => {
            let filtered: Vec<String> = detected
                .iter()
                .filter(|path| is_structural_change_target(path))
                .cloned()
                .collect();

            if filtered.is_empty() {
                ChangeTargetResolution {
                    targets: detected.clone(),
                    source: ChangeTargetSource::GitStatusAll,
                    observed_changed_paths: detected.len(),
                    structural_target_count: 0,
                    note: "This fallback keeps observability honest, but explicit targets are safer when you want deterministic scope.".into(),
                }
            } else {
                ChangeTargetResolution {
                    targets: filtered.clone(),
                    source: ChangeTargetSource::GitStatusStructural,
                    observed_changed_paths: detected.len(),
                    structural_target_count: filtered.len(),
                    note: "Structural filtering removes docs-only noise so change-oriented commands stay focused on development-critical paths.".into(),
                }
            }
        }
        io::GitStatusProbe::Clean => ChangeTargetResolution {
            targets: Vec::new(),
            source: ChangeTargetSource::GitStatusClean,
            observed_changed_paths: 0,
            structural_target_count: 0,
            note: "Pass targets explicitly when you need guidance before editing or when you are validating a clean checkout.".into(),
        },
        io::GitStatusProbe::NotRepository => ChangeTargetResolution {
            targets: Vec::new(),
            source: ChangeTargetSource::NotGitRepository,
            observed_changed_paths: 0,
            structural_target_count: 0,
            note: "Pass targets explicitly to preserve predictable analysis outside repository worktrees.".into(),
        },
        io::GitStatusProbe::Unavailable { detail } | io::GitStatusProbe::Failed { detail } => {
            ChangeTargetResolution {
                targets: Vec::new(),
                source: ChangeTargetSource::GitUnavailable,
                observed_changed_paths: 0,
                structural_target_count: 0,
                note: format!(
                    "git status could not be trusted ({detail}). Pass targets explicitly so the command scope stays auditable."
                ),
            }
        }
    }
}

fn is_structural_change_target(path: &str) -> bool {
    let normalized = path.trim_start_matches("./").to_ascii_lowercase();

    normalized == "makefile"
        || normalized == "go.work"
        || normalized.ends_with(".go")
        || normalized.ends_with(".rs")
        || normalized.ends_with(".json")
        || normalized.ends_with(".jsonc")
        || normalized.ends_with(".yaml")
        || normalized.ends_with(".yml")
        || normalized.ends_with(".toml")
        || normalized.ends_with(".mod")
        || normalized.ends_with(".sum")
        || normalized.starts_with("cmd/")
        || normalized.starts_with("internal/")
        || normalized.starts_with("deploy/")
        || normalized.starts_with("tools/raccoon-cli/")
        || normalized.starts_with("scripts/")
        || normalized.starts_with("tests/")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn structural_change_filter_excludes_docs_only_paths() {
        assert!(is_structural_change_target("internal/application/foo.go"));
        assert!(is_structural_change_target("Makefile"));
        assert!(!is_structural_change_target("docs/operations/guide.md"));
        assert!(!is_structural_change_target("README.md"));
    }

    #[test]
    fn explicit_targets_win_over_git_auto_detection() {
        let resolution = resolve_change_targets(
            Path::new("."),
            &["internal/domain/config.go".into(), "README.md".into()],
        );

        assert_eq!(resolution.source, ChangeTargetSource::Explicit);
        assert_eq!(
            resolution.targets,
            vec![
                "internal/domain/config.go".to_string(),
                "README.md".to_string()
            ]
        );
    }

    #[test]
    fn probe_prefers_structural_targets_when_available() {
        let resolution = resolve_from_probe(
            Path::new("."),
            io::GitStatusProbe::Changed(vec![
                "docs/notes.md".into(),
                "internal/domain/config.go".into(),
                "README.md".into(),
            ]),
        );

        assert_eq!(resolution.source, ChangeTargetSource::GitStatusStructural);
        assert_eq!(resolution.observed_changed_paths, 3);
        assert_eq!(resolution.structural_target_count, 1);
        assert_eq!(
            resolution.targets,
            vec!["internal/domain/config.go".to_string()]
        );
    }

    #[test]
    fn probe_reports_clean_worktree_explicitly() {
        let resolution = resolve_from_probe(Path::new("."), io::GitStatusProbe::Clean);
        assert_eq!(resolution.source, ChangeTargetSource::GitStatusClean);
        assert!(resolution.targets.is_empty());
        assert!(
            resolution.scope_note("files").contains("clean worktree"),
            "scope note should explain why no files were found: {}",
            resolution.scope_note("files")
        );
    }

    #[test]
    fn probe_reports_non_git_repository_explicitly() {
        let resolution = resolve_from_probe(Path::new("."), io::GitStatusProbe::NotRepository);
        assert_eq!(resolution.source, ChangeTargetSource::NotGitRepository);
        assert!(resolution.targets.is_empty());
        assert!(
            resolution
                .scope_note("targets")
                .contains("not a git repository"),
            "scope note should explain non-git roots: {}",
            resolution.scope_note("targets")
        );
    }
}
