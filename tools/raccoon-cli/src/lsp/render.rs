use crate::lsp::types::LspStatus;
use crate::lsp::EnrichedSymbol;

pub fn render_enriched_human(enriched: &EnrichedSymbol, verbose: bool) -> String {
    let mut out = String::new();
    out.push_str(&format!("Symbol: {}\n\n", enriched.symbol));

    match &enriched.lsp_status {
        LspStatus::Enriched => out.push_str("LSP: enriched (gopls connected)\n"),
        LspStatus::NoResults => out.push_str("LSP: connected but no additional results\n"),
        LspStatus::Unavailable { reason } => {
            out.push_str(&format!("LSP: unavailable ({reason})\n"));
        }
    }
    out.push('\n');

    if enriched.ast_definitions.is_empty() {
        out.push_str("AST definitions: none found\n");
    } else {
        out.push_str(&format!(
            "AST definitions ({}):\n",
            enriched.ast_definitions.len()
        ));
        for def in &enriched.ast_definitions {
            let name = def.qualified_name.as_deref().unwrap_or(&enriched.symbol);
            out.push_str(&format!(
                "  {} at {}:{}\n",
                name, def.location.file, def.location.line
            ));
        }
    }

    if !enriched.lsp_definitions.is_empty() {
        out.push_str(&format!(
            "\nLSP definitions ({}):\n",
            enriched.lsp_definitions.len()
        ));
        for def in &enriched.lsp_definitions {
            out.push_str(&format!("  {}:{}\n", def.location.file, def.location.line));
        }
    }

    if !enriched.lsp_references.is_empty() {
        out.push_str(&format!(
            "\nLSP references ({}):\n",
            enriched.lsp_references.len()
        ));
        let limit = if verbose {
            enriched.lsp_references.len()
        } else {
            10
        };
        for reference in enriched.lsp_references.iter().take(limit) {
            out.push_str(&format!(
                "  {}:{}\n",
                reference.location.file, reference.location.line
            ));
        }
        if !verbose && enriched.lsp_references.len() > 10 {
            out.push_str(&format!(
                "  ... and {} more (use -v to show all)\n",
                enriched.lsp_references.len() - 10
            ));
        }
    }

    if let Some(ref hover) = enriched.hover {
        out.push('\n');
        if let Some(ref sig) = hover.signature {
            out.push_str(&format!("Type: {sig}\n"));
        }
        if verbose {
            if let Some(ref doc) = hover.documentation {
                out.push_str(&format!("Doc: {doc}\n"));
            }
        }
    }

    out
}
