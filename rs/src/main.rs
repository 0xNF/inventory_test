mod cli;
mod commands;
mod config;
mod regex_rust;
mod structs;
use clap::Parser;
use cli::*;
use commands::*;
use rusqlite::{Connection, Result as SqliteResult};

fn main() -> SqliteResult<()> {
    let cli = cli::Cli::parse();

    // Load configuration using XDG conventions
    let config = config::Config::load().unwrap_or_else(|_| config::Config::default());

    // Connect to the database
    let db_path = config
        .database_path
        .to_owned()
        .unwrap_or_else(|| "../inventory.db".to_string());
    let conn = Connection::open(db_path)?;

    // Add the REGEXP function
    regex_rust::add_regexp_function(&conn)?;

    // Apply config to command args if needed
    match &cli.command {
        Commands::List(args) => {
            // Apply config defaults if args are not explicitly provided
            let mut args_with_defaults = args.clone();

            // Apply defaults from config
            if args_with_defaults.limit.is_none() && !args_with_defaults.all {
                args_with_defaults.limit = config.default_page_limit;
            }

            if args_with_defaults.sort_by.is_none() {
                args_with_defaults.sort_by = config.default_sort_by.clone();
            }

            if args_with_defaults.order_by.is_none() {
                args_with_defaults.order_by = config.default_order_by.clone();
            }

            if args_with_defaults.short {
                list_short_inventory(&conn, &args_with_defaults)?;
            } else {
                list_long_inventory(&conn, &args_with_defaults)?;
            }
        }

        cli::Commands::Add(args) => {
            if args.interactive {
                // Use default currency from config if available
                let default_currency =
                    &config.default_currency.unwrap_or_else(|| "JPY".to_string());
                add_inventory_item_interactive(&conn, args.json, &default_currency)?;
            } else if let Some(json_input) = &args.input {
                add_inventory_item_from_json(&conn, json_input, args.json, &config)?;
            } else {
                add_inventory_item(&conn, args.name.as_ref().unwrap(), args.json)?;
            }
        }
        Commands::Remove(args) => {
            remove_inventory_item(&conn, &args.id, args.json)?;
        }
        Commands::Edit(args) => {
            edit_inventory_item(&conn, &args.id, &args.input, args.json)?;
        }
    }

    Ok(())
}
