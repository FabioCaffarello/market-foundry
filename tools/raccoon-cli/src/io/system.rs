use std::process::Command;

pub(crate) fn utc_timestamp() -> String {
    let output = Command::new("date")
        .args(["-u", "+%Y-%m-%dT%H:%M:%SZ"])
        .output();

    match output {
        Ok(out) if out.status.success() => String::from_utf8_lossy(&out.stdout).trim().to_string(),
        Err(_) | Ok(_) => "unknown".to_string(),
    }
}
