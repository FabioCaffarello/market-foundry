use std::collections::HashSet;
use std::path::Path;
use std::process::Command;

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum GitStatusProbe {
    Changed(Vec<String>),
    Clean,
    NotRepository,
    Unavailable { detail: String },
    Failed { detail: String },
}

pub(crate) fn status_porcelain(project_root: &Path) -> GitStatusProbe {
    let output = Command::new("git")
        .args(["status", "--porcelain", "-u"])
        .current_dir(project_root)
        .output();

    match output {
        Ok(out) if out.status.success() => {
            let paths = parse_status_paths(&String::from_utf8_lossy(&out.stdout));
            if paths.is_empty() {
                GitStatusProbe::Clean
            } else {
                GitStatusProbe::Changed(paths)
            }
        }
        Ok(out) => {
            let stderr = String::from_utf8_lossy(&out.stderr).trim().to_string();
            if stderr.contains("not a git repository") {
                GitStatusProbe::NotRepository
            } else {
                GitStatusProbe::Failed {
                    detail: if stderr.is_empty() {
                        format!("git exited with status {}", out.status)
                    } else {
                        stderr
                    },
                }
            }
        }
        Err(error) => GitStatusProbe::Unavailable {
            detail: error.to_string(),
        },
    }
}

fn parse_status_paths(output: &str) -> Vec<String> {
    let mut seen = HashSet::new();
    let mut paths = Vec::new();

    for line in output.lines() {
        if line.len() < 4 {
            continue;
        }

        let status = &line[..2];
        if status.contains('D') {
            continue;
        }

        let raw_path = line[3..].trim();
        if raw_path.is_empty() {
            continue;
        }

        let path = raw_path
            .split_once(" -> ")
            .map(|(_, new_path)| new_path)
            .unwrap_or(raw_path)
            .trim_matches('"')
            .trim();

        if path.is_empty() {
            continue;
        }

        let owned = path.to_string();
        if seen.insert(owned.clone()) {
            paths.push(owned);
        }
    }

    paths
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_status_paths_handles_renames_and_deletes() {
        let parsed = parse_status_paths(
            " M internal/application/foo.go\nR  old/name.go -> internal/application/bar.go\nD  removed/file.go\n?? docs/note.md\n",
        );

        assert_eq!(
            parsed,
            vec![
                "internal/application/foo.go".to_string(),
                "internal/application/bar.go".to_string(),
                "docs/note.md".to_string()
            ]
        );
    }

    #[test]
    fn failed_probe_classifies_non_git_repository() {
        let probe = match Command::new("git")
            .args(["status", "--porcelain", "-u"])
            .current_dir("/nonexistent")
            .output()
        {
            Ok(out) if !out.status.success() => {
                let stderr = String::from_utf8_lossy(&out.stderr).trim().to_string();
                if stderr.contains("not a git repository") {
                    GitStatusProbe::NotRepository
                } else {
                    GitStatusProbe::Failed { detail: stderr }
                }
            }
            Ok(_) => GitStatusProbe::Clean,
            Err(error) => GitStatusProbe::Unavailable {
                detail: error.to_string(),
            },
        };

        assert!(
            matches!(
                probe,
                GitStatusProbe::NotRepository
                    | GitStatusProbe::Failed { .. }
                    | GitStatusProbe::Unavailable { .. }
            ),
            "unexpected probe variant: {probe:?}"
        );
    }
}
