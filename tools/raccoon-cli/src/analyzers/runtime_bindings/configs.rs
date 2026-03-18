use crate::error::Result;
use std::path::Path;

/// A service config entry extracted from deploy/configs/*.jsonc files.
#[derive(Debug, Clone)]
pub struct ServiceConfig {
    /// Service name derived from the file name (e.g., "ingest", "derive", "store").
    pub service: String,
    /// NATS URL configured for this service.
    pub nats_url: Option<String>,
    /// Source file path.
    pub source_file: String,
    /// Evidence families configured in pipeline.families (empty = all enabled).
    pub families: Vec<String>,
    /// Signal families configured in pipeline.signal_families (empty = none).
    pub signal_families: Vec<String>,
    /// Decision families configured in pipeline.decision_families (empty = none).
    pub decision_families: Vec<String>,
    /// Strategy families configured in pipeline.strategy_families (empty = none).
    pub strategy_families: Vec<String>,
}

/// Scan deploy/configs/ for service config files and extract NATS connection info.
pub fn scan_service_configs(configs_dir: &Path) -> Result<Vec<ServiceConfig>> {
    let mut configs = Vec::new();

    let entries = match std::fs::read_dir(configs_dir) {
        Ok(e) => e,
        Err(_) => return Ok(configs),
    };

    for entry in entries {
        let entry = entry?;
        let path = entry.path();
        if path.extension().and_then(|e| e.to_str()) != Some("jsonc") {
            continue;
        }

        let file_name = path
            .file_stem()
            .and_then(|n| n.to_str())
            .unwrap_or("unknown")
            .to_string();

        let raw = std::fs::read_to_string(&path)?;
        let cleaned = strip_jsonc_comments(&raw);

        let (nats_url, families, signal_families, decision_families, strategy_families) =
            if let Ok(value) = serde_json::from_str::<serde_json::Value>(&cleaned) {
                let nats = value
                    .get("nats")
                    .and_then(|n| n.get("url"))
                    .and_then(|u| u.as_str())
                    .map(String::from)
                    .or_else(|| {
                        value
                            .get("nats_url")
                            .and_then(|u| u.as_str())
                            .map(String::from)
                    });
                let pipeline = value.get("pipeline");
                let fam = extract_string_array(pipeline, "families");
                let sig = extract_string_array(pipeline, "signal_families");
                let dec = extract_string_array(pipeline, "decision_families");
                let strat = extract_string_array(pipeline, "strategy_families");
                (nats, fam, sig, dec, strat)
            } else {
                (None, Vec::new(), Vec::new(), Vec::new(), Vec::new())
            };

        configs.push(ServiceConfig {
            service: file_name,
            nats_url,
            source_file: path.to_string_lossy().to_string(),
            families,
            signal_families,
            decision_families,
            strategy_families,
        });
    }

    configs.sort_by(|a, b| a.service.cmp(&b.service));
    Ok(configs)
}

/// Extract a string array from a JSON object field.
fn extract_string_array(parent: Option<&serde_json::Value>, key: &str) -> Vec<String> {
    parent
        .and_then(|p| p.get(key))
        .and_then(|v| v.as_array())
        .map(|arr| {
            arr.iter()
                .filter_map(|v| v.as_str().map(String::from))
                .collect()
        })
        .unwrap_or_default()
}

fn strip_jsonc_comments(input: &str) -> String {
    let mut result = String::with_capacity(input.len());
    let mut in_string = false;
    let mut escape_next = false;
    let chars: Vec<char> = input.chars().collect();
    let len = chars.len();
    let mut i = 0;

    while i < len {
        if escape_next {
            result.push(chars[i]);
            escape_next = false;
            i += 1;
            continue;
        }
        if chars[i] == '\\' && in_string {
            result.push(chars[i]);
            escape_next = true;
            i += 1;
            continue;
        }
        if chars[i] == '"' {
            in_string = !in_string;
            result.push(chars[i]);
            i += 1;
            continue;
        }
        if !in_string && i + 1 < len && chars[i] == '/' && chars[i + 1] == '/' {
            while i < len && chars[i] != '\n' {
                i += 1;
            }
            continue;
        }
        result.push(chars[i]);
        i += 1;
    }

    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn scan_service_configs_returns_empty_on_empty_dir() {
        let dir = tempfile::tempdir().unwrap();
        let result = scan_service_configs(dir.path()).unwrap();
        assert!(result.is_empty());
    }

    #[test]
    fn scan_service_configs_skips_non_jsonc() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::write(dir.path().join("readme.md"), "# Not a config").unwrap();
        let result = scan_service_configs(dir.path()).unwrap();
        assert!(result.is_empty());
    }

    #[test]
    fn scan_service_configs_extracts_nats_url() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::write(
            dir.path().join("ingest.jsonc"),
            r#"{
  // ingest service config
  "nats": {
    "url": "nats://nats:4222"
  }
}"#,
        )
        .unwrap();

        let configs = scan_service_configs(dir.path()).unwrap();
        assert_eq!(configs.len(), 1);
        assert_eq!(configs[0].service, "ingest");
        assert_eq!(configs[0].nats_url.as_deref(), Some("nats://nats:4222"));
    }

    #[test]
    fn scan_service_configs_extracts_pipeline_families() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::write(
            dir.path().join("derive.jsonc"),
            r#"{
  "nats": { "url": "nats://nats:4222" },
  "pipeline": {
    "families": ["candle", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"]
  }
}"#,
        )
        .unwrap();

        let configs = scan_service_configs(dir.path()).unwrap();
        assert_eq!(configs[0].families, vec!["candle", "volume"]);
        assert_eq!(configs[0].signal_families, vec!["rsi"]);
        assert_eq!(configs[0].decision_families, vec!["rsi_oversold"]);
    }

    #[test]
    fn scan_service_configs_handles_missing_nats() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::write(
            dir.path().join("gateway.jsonc"),
            r#"{ "http": { "port": 8080 } }"#,
        )
        .unwrap();

        let configs = scan_service_configs(dir.path()).unwrap();
        assert_eq!(configs.len(), 1);
        assert_eq!(configs[0].service, "gateway");
        assert!(configs[0].nats_url.is_none());
    }

    #[test]
    fn scan_service_configs_multiple_files_sorted() {
        let dir = tempfile::tempdir().unwrap();
        std::fs::write(dir.path().join("store.jsonc"), "{}").unwrap();
        std::fs::write(dir.path().join("derive.jsonc"), "{}").unwrap();
        std::fs::write(dir.path().join("ingest.jsonc"), "{}").unwrap();

        let configs = scan_service_configs(dir.path()).unwrap();
        assert_eq!(configs.len(), 3);
        assert_eq!(configs[0].service, "derive");
        assert_eq!(configs[1].service, "ingest");
        assert_eq!(configs[2].service, "store");
    }

    #[test]
    fn strip_jsonc_comments_removes_line_comments() {
        let input = r#"{
  // this is a comment
  "key": "value"
}"#;
        let cleaned = strip_jsonc_comments(input);
        assert!(!cleaned.contains("//"));
        assert!(cleaned.contains("\"key\""));
    }

    #[test]
    fn strip_jsonc_comments_preserves_strings_with_slashes() {
        let input = r#"{ "url": "nats://host:4222" }"#;
        let cleaned = strip_jsonc_comments(input);
        assert!(cleaned.contains("nats://host:4222"));
    }
}
