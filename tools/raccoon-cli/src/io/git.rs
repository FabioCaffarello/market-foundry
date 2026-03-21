use std::collections::HashSet;
use std::path::Path;
use std::process::Command;

pub(crate) fn status_porcelain_paths(project_root: &Path) -> Vec<String> {
    let output = Command::new("git")
        .args(["status", "--porcelain", "-u"])
        .current_dir(project_root)
        .output();

    match output {
        Ok(out) if out.status.success() => {
            parse_status_paths(&String::from_utf8_lossy(&out.stdout))
        }
        Err(_) | Ok(_) => Vec::new(),
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
}
