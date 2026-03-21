use crate::io;

pub(crate) fn resolve_change_targets(
    project_root: &std::path::Path,
    explicit: &[String],
) -> Vec<String> {
    if !explicit.is_empty() {
        return explicit.to_vec();
    }

    let detected = detect_changed_files(project_root);
    let filtered: Vec<String> = detected
        .iter()
        .filter(|path| is_structural_change_target(path))
        .cloned()
        .collect();

    if filtered.is_empty() {
        detected
    } else {
        filtered
    }
}

fn detect_changed_files(project_root: &std::path::Path) -> Vec<String> {
    io::status_porcelain_paths(project_root)
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
}
