use clap::{Args, Parser, Subcommand};
use serde::Serialize;

/// Inventory Manager - A CLI tool to manage inventory items
#[derive(Parser)]
#[command(author, version, about, long_about = None)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,
}

#[derive(Subcommand)]
pub enum Commands {
    /// List all inventory items
    List(ListArgs),

    /// Add a new inventory item
    Add(AddArgs),

    /// Remove an inventory item by ID
    Remove(RemoveArgs),

    /// Edit an existing inventory item
    Edit(EditArgs),
}

#[derive(Debug, Serialize)]
pub struct PagedResponse<T> {
    pub items: Vec<T>,
    pub paging: PagingInfo,
}

#[derive(Debug, Serialize)]
pub struct PagingInfo {
    pub limit: Option<u32>,
    pub offset: Option<u32>,
    pub total: u32,
}

#[derive(Args, Clone)]
pub struct ListArgs {
    /// Display only ID, Name, and Date Purchased
    #[arg(short, long, default_value_t = false)]
    pub short: bool,

    /// Display all item details (default)
    #[arg(long, default_value_t = false)]
    pub long: bool,

    /// Output in JSON format
    #[arg(long, default_value_t = false)]
    pub json: bool,

    /// Return all results without paging
    #[arg(long, default_value_t = false)]
    pub all: bool,

    /// Number of items per page
    #[arg(long)]
    pub limit: Option<u32>,

    /// Number of items to skip
    #[arg(long)]
    pub offset: Option<u32>,

    /// Sort direction (asc or desc)
    #[arg(long, value_parser = ["asc", "desc"])]
    pub order_by: Option<String>,

    /// Fields to sort by, in order of priority (can be specified multiple times)
    #[arg(long, value_delimiter = ',')]
    pub sort_by: Option<Vec<String>>,

    /// Regular expression to filter results by
    #[arg(long)]
    pub filter: Option<String>,

    /// Comma-separated list of fields to filter on
    #[arg(long, value_delimiter = ',')]
    pub fields: Option<Vec<String>>,
}

#[derive(Args)]
pub struct AddArgs {
    /// Name of the item
    #[arg(required_unless_present_any = ["interactive", "input"])]
    pub name: Option<String>,

    /// Interactive mode - prompts for all fields
    #[arg(short = 'i', long = "interactive")]
    pub interactive: bool,

    /// JSON string containing item details
    #[arg(long = "input")]
    pub input: Option<String>,

    /// Output in JSON format
    #[arg(long, default_value_t = false)]
    pub json: bool,
}

#[derive(Args)]
pub struct RemoveArgs {
    /// ID of the item to remove
    #[arg(required = true)]
    pub id: String,

    /// Output in JSON format
    #[arg(long, default_value_t = false)]
    pub json: bool,
}

#[derive(Args)]
pub struct EditArgs {
    /// ID of the item to edit
    #[arg(required = true)]
    pub id: String,

    /// JSON string containing fields to update
    #[arg(long = "input", required = true)]
    pub input: String,

    /// Output in JSON format
    #[arg(long, default_value_t = false)]
    pub json: bool,
}
