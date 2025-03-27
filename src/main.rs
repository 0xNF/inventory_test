use chrono::Local;
use clap::{Args, Parser, Subcommand};
use rusqlite::{Connection, OptionalExtension, Result as SqliteResult};
use serde::{Deserialize, Serialize};
use serde_json;
use std::io::{self, Write};
use uuid::Uuid;

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

#[derive(Args)]
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

/// Represents an inventory item
#[derive(Serialize, Deserialize)]
struct InventoryItem {
    #[serde(default)]
    id: String,
    name: String,
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

fn main() -> SqliteResult<()> {
    let cli = Cli::parse();

    // Connect to the database
    let conn = Connection::open("../inventory.db")?;

    match &cli.command {
        Commands::List(args) => {
            if args.short {
                list_short_inventory(&conn, args)?;
            } else {
                list_long_inventory(&conn, args)?;
            }
        }
        Commands::Add(args) => {
            if args.interactive {
                add_inventory_item_interactive(&conn, args.json)?;
            } else if let Some(json_input) = &args.input {
                add_inventory_item_from_json(&conn, json_input, args.json)?;
            } else {
                add_inventory_item(&conn, args.name.as_ref().unwrap(), args.json)?;
            }
        }
        Commands::Remove(args) => {
            remove_inventory_item(&conn, &args.id, args.json)?;
        }
    }

    Ok(())
}

// Data structure for short inventory items
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
    // Get total count first
    let total: u32 = conn.query_row("SELECT COUNT(*) FROM inventory", [], |row| row.get(0))?;
    let mut query = String::from("SELECT Id, Name, AcquiredDate FROM inventory");

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
    let limit = if args.all { None } else { args.limit };
    let offset = if args.all { None } else { args.offset };

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
    // Get total count first
    let total: u32 = conn.query_row("SELECT COUNT(*) FROM inventory", [], |row| row.get(0))?;
    let mut query = String::from(
        "SELECT 
            Id, Name, AcquiredDate, PurchasePrice, PurchaseCurrency, 
            IsUsed, ReceivedFrom, SerialNumber, PurchaseReference, 
            Notes, Extra, FuturePurchase 
        FROM inventory",
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
    let items_iter = stmt.query_map([], |row| {
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
    let limit = if args.all { None } else { args.limit };
    let offset = if args.all { None } else { args.offset };

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
) -> SqliteResult<()> {
    // Parse the JSON input - handle missing values
    let mut item: InventoryItem = serde_json::from_str(json_input)
        .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?;

    // Set default values for empty/missing fields
    let id = Uuid::new_v4().to_string(); // Always generate new UUID for consistency
    let today = Local::now().format("%Y-%m-%d").to_string();

    // Use default values just like interactive mode
    let acquired_date = item.acquired_date.unwrap_or_else(|| today.clone());
    let purchase_currency = item
        .purchase_currency
        .clone()
        .unwrap_or_else(|| String::from("JPY"));
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

fn add_inventory_item_interactive(conn: &Connection, json: bool) -> SqliteResult<()> {
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
    let purchase_currency = prompt_input("Purchase currency", Some("JPY"), false);
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
