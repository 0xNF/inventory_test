use chrono::Local;
use clap::{Args, Parser, Subcommand};
use dirs::{config_dir, home_dir};
use rusqlite::{params_from_iter, Connection, OptionalExtension, Result as SqliteResult};
use serde::{Deserialize, Serialize};
use serde_json;
use std::fs::File;
use std::io::BufReader;
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use uuid::Uuid;
mod regex_rust;

/// Configuration structure for the inventory manager
#[derive(Debug, Serialize, Deserialize)]
struct Config {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub default_currency: Option<String>,

    #[serde(skip_serializing_if = "Option::is_none")]
    pub database_path: Option<String>,

    #[serde(skip_serializing_if = "Option::is_none")]
    pub default_page_limit: Option<u32>,

    #[serde(skip_serializing_if = "Option::is_none")]
    pub default_sort_by: Option<Vec<String>>,

    #[serde(skip_serializing_if = "Option::is_none")]
    pub default_order_by: Option<String>,
}

impl Config {
    /// Load configuration using XDG conventions
    pub fn load() -> Result<Self, Box<dyn std::error::Error>> {
        // Try multiple locations in order of priority
        let config_paths = get_config_paths();

        for path in config_paths {
            println!("Searching: {}", &path.to_string_lossy());
            if path.exists() {
                return Self::from_file(&path);
            }
        }

        // Return default if no config file found
        Ok(Self::default())
    }

    /// Load configuration from a specific file path
    pub fn from_file(path: &Path) -> Result<Self, Box<dyn std::error::Error>> {
        let file = File::open(path)?;
        let reader = BufReader::new(file);
        let config = serde_json::from_reader(reader)?;
        Ok(config)
    }

    /// Create a new default configuration
    pub fn default() -> Self {
        Config {
            default_currency: None,
            database_path: None,
            default_page_limit: None,
            default_sort_by: None,
            default_order_by: None,
        }
    }
}

/// Inventory Manager - A CLI tool to manage inventory items
#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
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
struct PagedResponse<T> {
    items: Vec<T>,
    paging: PagingInfo,
}

#[derive(Debug, Serialize)]
struct PagingInfo {
    limit: Option<u32>,
    offset: Option<u32>,
    total: u32,
}

#[derive(Args, Clone)]
struct ListArgs {
    /// Display only ID, Name, and Date Purchased
    #[arg(short, long, default_value_t = false)]
    short: bool,

    /// Display all item details (default)
    #[arg(long, default_value_t = false)]
    long: bool,

    /// Output in JSON format
    #[arg(long, default_value_t = false)]
    json: bool,

    /// Return all results without paging
    #[arg(long, default_value_t = false)]
    all: bool,

    /// Number of items per page
    #[arg(long)]
    limit: Option<u32>,

    /// Number of items to skip
    #[arg(long)]
    offset: Option<u32>,

    /// Sort direction (asc or desc)
    #[arg(long, value_parser = ["asc", "desc"])]
    order_by: Option<String>,

    /// Fields to sort by, in order of priority (can be specified multiple times)
    #[arg(long, value_delimiter = ',')]
    sort_by: Option<Vec<String>>,

    /// Regular expression to filter results by
    #[arg(long)]
    filter: Option<String>,

    /// Comma-separated list of fields to filter on
    #[arg(long, value_delimiter = ',')]
    fields: Option<Vec<String>>,
}

#[derive(Args)]
struct AddArgs {
    /// Name of the item
    #[arg(required_unless_present_any = ["interactive", "input"])]
    name: Option<String>,

    /// Interactive mode - prompts for all fields
    #[arg(short = 'i', long = "interactive")]
    interactive: bool,

    /// JSON string containing item details
    #[arg(long = "input")]
    input: Option<String>,

    /// Output in JSON format
    #[arg(long, default_value_t = false)]
    json: bool,
}

#[derive(Args)]
struct RemoveArgs {
    /// ID of the item to remove
    #[arg(required = true)]
    id: String,

    /// Output in JSON format
    #[arg(long, default_value_t = false)]
    json: bool,
}

#[derive(Args)]
struct EditArgs {
    /// ID of the item to edit
    #[arg(required = true)]
    id: String,

    /// JSON string containing fields to update
    #[arg(long = "input", required = true)]
    input: String,

    /// Output in JSON format
    #[arg(long, default_value_t = false)]
    json: bool,
}

/// Represents a new inventory item with required name
#[derive(Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct InventoryItem {
    #[serde(default)]
    id: String,
    name: String, // Required for new items
    acquired_date: Option<String>,
    purchase_price: Option<i64>,
    purchase_currency: Option<String>,
    is_used: Option<bool>,
    received_from: Option<String>,
    serial_number: Option<String>,
    purchase_reference: Option<String>,
    notes: Option<String>,
    extra: Option<String>,
    future_purchase: Option<bool>,
}

const FIELDS_ARR: &[&str] = &[
    "Name",
    "AcquiredDate",
    "PurchaseCurrency",
    "PurchasePrice",
    "ReceivedFrom",
    "SerialNumber",
    "PurchaseReference",
    "Notes",
    "Extra",
];

/// Get the list of possible config file paths following XDG convention
fn get_config_paths() -> Vec<PathBuf> {
    let mut paths = Vec::new();

    // First check for environment variable
    if let Ok(path) = std::env::var("0XNFWT_INVENTORY_CONFIG") {
        paths.push(PathBuf::from(path));
    }

    // Then check XDG_CONFIG_HOME or ~/.config
    if let Some(config_dir) = config_dir() {
        let xdg_path = config_dir
            .join("0xnfwt_inventory")
            .join("0xnfwt_inventory.json");
        paths.push(xdg_path);
    }

    // Check home directory
    if let Some(home) = home_dir() {
        paths.push(home.join(".0xnfwt_inventory.json"));
    }

    // Check current directory
    paths.push(PathBuf::from("0xnfwt_inventory.json"));

    // Check relative to executable
    paths.push(PathBuf::from("../0xnfwt_inventory.json"));

    paths
}

fn main() -> SqliteResult<()> {
    let cli = Cli::parse();

    // Load configuration using XDG conventions
    let config = Config::load().unwrap_or_else(|_| Config::default());

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

        Commands::Add(args) => {
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

// Data structure for short inventory items
/// Represents an editable item where all fields are optional
#[derive(Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct EditableItem {
    #[serde(default)]
    #[serde(skip_serializing)]
    id: String,
    #[serde(default)]
    name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    acquired_date: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    purchase_price: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    purchase_currency: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    is_used: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    received_from: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    serial_number: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    purchase_reference: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    notes: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    extra: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    future_purchase: Option<bool>,
}

#[derive(Serialize)]
struct ShortInventoryItem {
    id: String,
    name: String,
    acquired_date: Option<String>,
}

fn build_sort_clause(args: &ListArgs) -> String {
    if let Some(sort_fields) = &args.sort_by {
        if !sort_fields.is_empty() {
            let direction = args.order_by.as_deref().unwrap_or("asc");
            let sort_terms: Vec<String> = sort_fields
                .iter()
                .map(|field| format!("{} {}", field, direction))
                .collect();
            format!(" ORDER BY {}", sort_terms.join(", "))
        } else {
            String::new()
        }
    } else {
        String::new()
    }
}

// Function to retrieve short inventory data
fn get_short_inventory(
    conn: &Connection,
    args: &ListArgs,
) -> SqliteResult<PagedResponse<ShortInventoryItem>> {
    use regex::Regex;

    // Build the WHERE clause for filtering if needed
    let mut where_conditions = Vec::new();
    let mut params: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();

    if let Some(filter_pattern) = &args.filter {
        if let Ok(_) = Regex::new(filter_pattern) {
            let filter_fields = args.fields.as_ref().map(|f| f.to_owned()).unwrap_or(
                FIELDS_ARR
                    .iter()
                    .map(|y| y.to_string())
                    .collect::<Vec<String>>(),
            );

            let field_conditions: Vec<String> = filter_fields
                .iter()
                .map(|field| {
                    params.push(Box::new(filter_pattern));
                    format!("{} REGEXP ?", field)
                })
                .collect();

            if !field_conditions.is_empty() {
                where_conditions.push(format!("({})", field_conditions.join(" OR ")));
            }
        }
    }

    // Build the complete query
    let where_clause = if !where_conditions.is_empty() {
        format!(" WHERE {}", where_conditions.join(" AND "))
    } else {
        String::new()
    };

    // Get total count with filters applied
    let count_query = format!("SELECT COUNT(*) FROM inventory{}", where_clause);

    let param_refs: Vec<&dyn rusqlite::ToSql> = params.iter().map(|p| p.as_ref()).collect();
    let total: u32 = conn.query_row(&count_query, param_refs.as_slice(), |row| row.get(0))?;

    let mut query = format!(
        "SELECT Id, Name, AcquiredDate FROM inventory{}",
        where_clause
    );

    // Add sorting
    query.push_str(&build_sort_clause(args));

    if let Some(limit_val) = args.limit {
        query.push_str(&format!(" LIMIT {}", limit_val));
        if let Some(offset_val) = args.offset {
            query.push_str(&format!(" OFFSET {}", offset_val));
        }
    }

    let mut stmt = conn.prepare(&query)?;
    let items = stmt.query_map([], |row| {
        Ok(ShortInventoryItem {
            id: row.get(0)?,
            name: row.get(1)?,
            acquired_date: row.get(2)?,
        })
    })?;

    let mut results = Vec::new();
    for item in items {
        results.push(item?);
    }
    Ok(PagedResponse {
        items: results,
        paging: PagingInfo {
            limit: args.limit,
            offset: args.offset,
            total,
        },
    })
}

// Function to print short inventory
fn print_short_inventory(response: &PagedResponse<ShortInventoryItem>) {
    println!("{:<36} | {:<30} | {:<10}", "ID", "Name", "Acquired Date");
    println!("{:-<36}-+-{:-<30}-+-{:-<10}", "", "", "");

    if response.paging.total > 0 {
        if let Some(limit) = response.paging.limit {
            let start = response.paging.offset.unwrap_or(0) + 1;
            let end = (start + limit - 1).min(response.paging.total);
            println!(
                "Showing items {}-{} of {}",
                start, end, response.paging.total
            );
        } else {
            println!("Showing all {} items", response.paging.total);
        }
    } else {
        println!("No items found");
    }

    for item in &response.items {
        let date_str = item.acquired_date.as_deref().unwrap_or("N/A");
        println!("{:<36} | {:<30} | {:<10}", item.id, item.name, date_str);
    }
}

// Main function that combines retrieval and display
fn list_short_inventory(conn: &Connection, args: &ListArgs) -> SqliteResult<()> {
    let response = get_short_inventory(conn, args)?;
    if args.json {
        println!(
            "{}",
            serde_json::to_string_pretty(&response)
                .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?
        );
    } else {
        print_short_inventory(&response);
    }
    Ok(())
}

// Function to retrieve full inventory data
fn get_long_inventory(
    conn: &Connection,
    args: &ListArgs,
) -> SqliteResult<PagedResponse<InventoryItem>> {
    use regex::Regex;

    // Build the WHERE clause for filtering if needed
    let mut where_conditions = Vec::new();
    let mut params: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();

    let mut filter_field_count: usize = 0;
    if let Some(filter_pattern) = &args.filter {
        if let Ok(_) = Regex::new(filter_pattern) {
            let filter_fields = args.fields.as_ref().map(|f| f.to_owned()).unwrap_or(
                FIELDS_ARR
                    .iter()
                    .map(|y| y.to_string())
                    .collect::<Vec<String>>(),
            );
            filter_field_count = filter_fields.len();

            let field_conditions: Vec<String> = filter_fields
                .iter()
                .map(|field| {
                    params.push(Box::new(filter_pattern));
                    format!("{} REGEXP ?", field)
                })
                .collect();

            if !field_conditions.is_empty() {
                where_conditions.push(format!("({})", field_conditions.join(" OR ")));
            }
        }
    }

    // Build the complete query
    let where_clause = if !where_conditions.is_empty() {
        format!(" WHERE {}", where_conditions.join(" AND "))
    } else {
        String::new()
    };

    // Get total count with filters applied
    let count_query = format!("SELECT COUNT(*) FROM inventory{}", where_clause);

    let param_refs: Vec<&dyn rusqlite::ToSql> = params.iter().map(|p| p.as_ref()).collect();
    let total: u32 = conn.query_row(&count_query, param_refs.as_slice(), |row| row.get(0))?;

    let mut query = format!(
        "SELECT 
            Id, Name, AcquiredDate, PurchasePrice, PurchaseCurrency, 
            IsUsed, ReceivedFrom, SerialNumber, PurchaseReference, 
            Notes, Extra, FuturePurchase 
        FROM inventory{}",
        where_clause
    );

    // Add sorting
    query.push_str(&build_sort_clause(args));

    if let Some(limit_val) = args.limit {
        query.push_str(&format!(" LIMIT {}", limit_val));
        if let Some(offset_val) = args.offset {
            query.push_str(&format!(" OFFSET {}", offset_val));
        }
    }

    let mut stmt = conn.prepare(&query)?;

    let params = match args.filter.as_ref() {
        Some(s) => {
            let mut v: Vec<String> = Vec::with_capacity(filter_field_count);
            for _ in 0..filter_field_count {
                v.push(s.to_owned());
            }
            v
        }
        None => Vec::new(),
    };
    let items_iter = stmt.query_map(params_from_iter(params.iter()), |row| {
        let is_used: Option<i64> = row.get(5)?;
        let future_purchase: Option<i64> = row.get(11)?;

        Ok(InventoryItem {
            id: row.get(0)?,
            name: row.get(1)?,
            acquired_date: row.get(2)?,
            purchase_price: row.get(3)?,
            purchase_currency: row.get(4)?,
            is_used: is_used.map(|v| v != 0),
            received_from: row.get(6)?,
            serial_number: row.get(7)?,
            purchase_reference: row.get(8)?,
            notes: row.get(9)?,
            extra: row.get(10)?,
            future_purchase: future_purchase.map(|v| v != 0),
        })
    })?;

    let mut items = Vec::new();
    for item_result in items_iter {
        items.push(item_result?);
    }
    Ok(PagedResponse {
        items,
        paging: PagingInfo {
            limit: args.limit,
            offset: args.offset,
            total,
        },
    })
}

// Function to print full inventory details
fn print_long_inventory(response: &PagedResponse<InventoryItem>, json: bool) -> SqliteResult<()> {
    if json {
        println!(
            "{}",
            serde_json::to_string_pretty(&response)
                .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?
        );
    } else {
        if response.paging.total > 0 {
            if let Some(limit) = response.paging.limit {
                let start = response.paging.offset.unwrap_or(0) + 1;
                let end = (start + limit - 1).min(response.paging.total);
                println!(
                    "Showing items {}-{} of {}",
                    start, end, response.paging.total
                );
            } else {
                println!("Showing all {} items", response.paging.total);
            }
        } else {
            println!("No items found");
        }
        println!();
        for item in &response.items {
            println!("ID: {}", item.id);
            println!("Name: {}", item.name);

            if let Some(date) = &item.acquired_date {
                println!("Acquired Date: {}", date);
            }

            if let Some(price) = item.purchase_price {
                println!("Purchase Price: {}", price);
            }

            if let Some(currency) = &item.purchase_currency {
                println!("Purchase Currency: {}", currency);
            }

            println!("Is Used: {}", item.is_used.unwrap_or(false));

            if let Some(from) = &item.received_from {
                println!("Received From: {}", from);
            }

            if let Some(serial) = &item.serial_number {
                println!("Serial Number: {}", serial);
            }

            if let Some(reference) = &item.purchase_reference {
                println!("Purchase Reference: {}", reference);
            }

            if let Some(notes) = &item.notes {
                println!("Notes: {}", notes);
            }

            if let Some(extra) = &item.extra {
                println!("Extra: {}", extra);
            }

            println!("Future Purchase: {}", item.future_purchase.unwrap_or(false));
            println!("----------------------------------------");
        }
    }
    Ok(())
}

// Main function that combines retrieval and display
fn list_long_inventory(conn: &Connection, args: &ListArgs) -> SqliteResult<()> {
    let response = get_long_inventory(conn, args)?;
    print_long_inventory(&response, args.json)
}

// Data structure for a newly added inventory item
#[derive(Serialize)]
struct NewInventoryItem {
    id: String,
    name: String,
    acquired_date: String,
}

// Function to add a new inventory item to the database
fn create_inventory_item(conn: &Connection, name: &str) -> SqliteResult<NewInventoryItem> {
    let id = Uuid::new_v4().to_string();
    let today = Local::now().format("%Y-%m-%d").to_string();

    conn.execute(
        "INSERT INTO inventory (Id, Name, AcquiredDate) VALUES (?1, ?2, ?3)",
        [&id, name, &today],
    )?;

    Ok(NewInventoryItem {
        id,
        name: name.to_string(),
        acquired_date: today,
    })
}

// Function to print a newly added inventory item
fn print_new_inventory_item(item: &NewInventoryItem, json: bool) -> SqliteResult<()> {
    if json {
        println!(
            "{}",
            serde_json::to_string_pretty(&item)
                .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?
        );
    } else {
        println!("Added new inventory item:");
        println!("ID: {}", item.id);
        println!("Name: {}", item.name);
        println!("Acquired Date: {}", item.acquired_date);
    }
    Ok(())
}

// Main function that combines creation and display
fn add_inventory_item(conn: &Connection, name: &str, json: bool) -> SqliteResult<()> {
    let item = create_inventory_item(conn, name)?;
    print_new_inventory_item(&item, json)
}

fn add_inventory_item_from_json(
    conn: &Connection,
    json_input: &str,
    json_output: bool,
    config: &Config,
) -> SqliteResult<()> {
    // Parse the JSON input - handle missing values
    let item: InventoryItem = serde_json::from_str(json_input)
        .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?;

    // Set default values for empty/missing fields
    let id = Uuid::new_v4().to_string(); // Always generate new UUID for consistency
    let today = Local::now().format("%Y-%m-%d").to_string();

    // Use default values just like interactive mode
    let acquired_date = item.acquired_date.unwrap_or_else(|| today.clone());
    // Use default currency from config if available
    let purchase_currency = item.purchase_currency.clone().unwrap_or_else(|| {
        config
            .default_currency
            .clone()
            .unwrap_or_else(|| String::from("JPY"))
    });
    let is_used = item.is_used.unwrap_or(false);
    let future_purchase = item.future_purchase.unwrap_or(false);

    // Insert the new item into the database
    conn.execute(
        "INSERT INTO inventory (
            Id, Name, AcquiredDate, PurchasePrice, PurchaseCurrency, 
            IsUsed, ReceivedFrom, SerialNumber, PurchaseReference, 
            Notes, Extra, FuturePurchase
        ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)",
        rusqlite::params![
            id,
            item.name,
            acquired_date,
            item.purchase_price,
            purchase_currency,
            is_used as i64,
            item.received_from,
            item.serial_number,
            item.purchase_reference,
            item.notes,
            item.extra,
            future_purchase as i64
        ],
    )?;

    // Create response structure
    let new_item = NewInventoryItem {
        id,
        name: item.name,
        acquired_date,
    };

    // Print the result
    print_new_inventory_item(&new_item, json_output)
}

fn add_inventory_item_interactive(
    conn: &Connection,
    json: bool,
    default_currency: &str,
) -> SqliteResult<()> {
    // Create a new inventory item interactively
    let id = Uuid::new_v4().to_string();
    let today = Local::now().format("%Y-%m-%d").to_string();

    // Name - required field
    let name = prompt_input("Name of item", None, true);

    // Acquired Date
    let acquired_date = prompt_input("Date of purchase (YYYY-MM-DD)", Some(&today), false);

    // Purchase Price
    let price_str = prompt_input("Purchase price (leave empty if unknown)", None, false);
    let purchase_price = if price_str.is_empty() {
        None
    } else {
        price_str.parse::<i64>().ok()
    };

    // Purchase Currency
    let purchase_currency = prompt_input("Purchase currency", Some(default_currency), false);
    let purchase_currency = if purchase_currency.is_empty() {
        None
    } else {
        Some(purchase_currency)
    };

    // Is Used
    let is_used_str = prompt_input("Is this a used item? (y/n)", Some("n"), false).to_lowercase();
    let is_used = is_used_str.starts_with('y');

    // Received From
    let received_from = prompt_input("Received from", None, false);
    let received_from = if received_from.is_empty() {
        None
    } else {
        Some(received_from)
    };

    // Serial Number
    let serial_number = prompt_input("Serial number", None, false);
    let serial_number = if serial_number.is_empty() {
        None
    } else {
        Some(serial_number)
    };

    // Purchase Reference
    let purchase_reference = prompt_input("Purchase reference", None, false);
    let purchase_reference = if purchase_reference.is_empty() {
        None
    } else {
        Some(purchase_reference)
    };

    // Notes
    let notes = prompt_input("Notes", None, false);
    let notes = if notes.is_empty() { None } else { Some(notes) };

    // Extra
    let extra = prompt_input("Extra information", None, false);
    let extra = if extra.is_empty() { None } else { Some(extra) };

    // Future Purchase
    let future_purchase_str =
        prompt_input("Is this a future purchase? (y/n)", Some("n"), false).to_lowercase();
    let future_purchase = future_purchase_str.starts_with('y');

    // Insert the new item into the database
    conn.execute(
        "INSERT INTO inventory (
            Id, Name, AcquiredDate, PurchasePrice, PurchaseCurrency, 
            IsUsed, ReceivedFrom, SerialNumber, PurchaseReference, 
            Notes, Extra, FuturePurchase
        ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)",
        rusqlite::params![
            id,
            name,
            acquired_date,
            purchase_price,
            purchase_currency,
            is_used as i64,
            received_from,
            serial_number,
            purchase_reference,
            notes,
            extra,
            future_purchase as i64
        ],
    )?;

    let new_item = NewInventoryItem {
        id,
        name,
        acquired_date,
    };

    print_new_inventory_item(&new_item, json)
}

/// Helper function to prompt for user input with an optional default value
fn prompt_input(prompt: &str, default: Option<&str>, required: bool) -> String {
    loop {
        print!("{}", prompt);

        // Show default if available
        if let Some(default_value) = default {
            print!(" [{}]", default_value);
        }

        print!(":");
        if required {
            print!(" (required)")
        }
        print!(" ");
        io::stdout().flush().unwrap();

        let mut input = String::new();
        io::stdin()
            .read_line(&mut input)
            .expect("Failed to read input");
        let input = input.trim().to_string();

        // Return the default if input is empty and a default exists
        if input.is_empty() {
            if let Some(default_value) = default {
                return default_value.to_string();
            } else if !required {
                return input;
            }
            println!("This field is required. Please provide a value.");
            continue;
        }

        return input;
    }
}

// Data structure for removal result
#[derive(Serialize)]
struct RemovalResult {
    success: bool,
    item_id: String,
    item_name: Option<String>,
    message: String,
}

// Function to remove an inventory item from the database
fn delete_inventory_item(conn: &Connection, id: &str) -> SqliteResult<RemovalResult> {
    // First verify the item exists
    let mut stmt = conn.prepare("SELECT Name FROM inventory WHERE Id = ?1")?;
    let name: Option<String> = stmt.query_row([id], |row| row.get(0)).optional()?;

    let (success, message) = match name {
        Some(ref name) => {
            let deleted = conn.execute("DELETE FROM inventory WHERE Id = ?1", [id])?;
            if deleted > 0 {
                (
                    true,
                    format!("Successfully removed item '{}' with ID: {}", name, id),
                )
            } else {
                (false, format!("No item found with ID: {}", id))
            }
        }
        None => (false, format!("No item found with ID: {}", id)),
    };

    Ok(RemovalResult {
        success,
        item_id: id.to_string(),
        item_name: name,
        message,
    })
}

// Function to print removal result
fn print_removal_result(result: &RemovalResult, json: bool) -> SqliteResult<()> {
    if json {
        println!(
            "{}",
            serde_json::to_string_pretty(&result)
                .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?
        );
    } else {
        println!("{}", result.message);
    }
    Ok(())
}

// Main function that combines removal and display
fn remove_inventory_item(conn: &Connection, id: &str, json: bool) -> SqliteResult<()> {
    let result = delete_inventory_item(conn, id)?;
    print_removal_result(&result, json)
}

#[derive(Serialize)]
struct EditResult {
    success: bool,
    item_id: String,
    message: String,
}

fn edit_inventory_item(
    conn: &Connection,
    id: &str,
    json_input: &str,
    json_output: bool,
) -> SqliteResult<()> {
    // First verify the item exists
    let mut stmt = conn.prepare("SELECT Name FROM inventory WHERE Id = ?1")?;
    let name: Option<String> = stmt.query_row([id], |row| row.get(0)).optional()?;

    if name.is_none() {
        let result = EditResult {
            success: false,
            item_id: id.to_string(),
            message: format!("No item found with ID: {}", id),
        };
        return print_edit_result(&result, json_output);
    }

    // Parse the JSON input for editable item
    let updates: EditableItem = serde_json::from_str(json_input)
        .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?;

    // Build the UPDATE query dynamically based on which fields are present
    let mut query = String::from("UPDATE inventory SET ");
    let mut params: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();
    let mut set_clauses = Vec::new();

    if !updates.name.is_empty() {
        set_clauses.push("Name = ?");
        params.push(Box::new(updates.name.clone()));
    }
    if updates.acquired_date.is_some() {
        set_clauses.push("AcquiredDate = ?");
        params.push(Box::new(updates.acquired_date));
    }
    if updates.purchase_price.is_some() {
        set_clauses.push("PurchasePrice = ?");
        params.push(Box::new(updates.purchase_price));
    }
    if updates.purchase_currency.is_some() {
        set_clauses.push("PurchaseCurrency = ?");
        params.push(Box::new(updates.purchase_currency));
    }
    if updates.is_used.is_some() {
        set_clauses.push("IsUsed = ?");
        params.push(Box::new(updates.is_used.map(|v| v as i64)));
    }
    if updates.received_from.is_some() {
        set_clauses.push("ReceivedFrom = ?");
        params.push(Box::new(updates.received_from));
    }
    if updates.serial_number.is_some() {
        set_clauses.push("SerialNumber = ?");
        params.push(Box::new(updates.serial_number));
    }
    if updates.purchase_reference.is_some() {
        set_clauses.push("PurchaseReference = ?");
        params.push(Box::new(updates.purchase_reference));
    }
    if updates.notes.is_some() {
        set_clauses.push("Notes = ?");
        params.push(Box::new(updates.notes));
    }
    if updates.extra.is_some() {
        set_clauses.push("Extra = ?");
        params.push(Box::new(updates.extra));
    }
    if updates.future_purchase.is_some() {
        set_clauses.push("FuturePurchase = ?");
        params.push(Box::new(updates.future_purchase.map(|v| v as i64)));
    }

    if set_clauses.is_empty() {
        let result = EditResult {
            success: false,
            item_id: id.to_string(),
            message: "No fields to update".to_string(),
        };
        return print_edit_result(&result, json_output);
    }

    query.push_str(&set_clauses.join(", "));
    query.push_str(" WHERE Id = ?");
    params.push(Box::new(id));

    // Execute the update
    let mut stmt = conn.prepare(&query)?;
    let param_refs: Vec<&dyn rusqlite::ToSql> = params.iter().map(|p| p.as_ref()).collect();
    let updated = stmt.execute(param_refs.as_slice())?;

    let result = EditResult {
        success: updated > 0,
        item_id: id.to_string(),
        message: if updated > 0 {
            format!("Successfully updated item with ID: {}", id)
        } else {
            format!("Failed to update item with ID: {}", id)
        },
    };

    print_edit_result(&result, json_output)
}

fn print_edit_result(result: &EditResult, json: bool) -> SqliteResult<()> {
    if json {
        println!(
            "{}",
            serde_json::to_string_pretty(&result)
                .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?
        );
    } else {
        println!("{}", result.message);
    }
    Ok(())
}
