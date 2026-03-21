mod analyzers;
mod application;
mod cli;
mod codeintel;
mod error;
mod gate;
mod io;
mod lsp;
mod models;
mod output;
#[allow(dead_code)]
mod smoke;

use clap::Parser;

fn main() {
    let cli = cli::Cli::parse();
    std::process::exit(application::run(cli));
}
