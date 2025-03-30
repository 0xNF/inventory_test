use dirs::{config_dir, home_dir};
use serde::{Deserialize, Serialize};
use serde_json;
use std::fs::File;
use std::io::BufReader;
use std::path::{Path, PathBuf};

/// Configuration structure for the inventory manager
#[derive(Debug, Serialize, Deserialize)]
pub struct Config {
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

/// Get the list of possible config file paths following XDG convention
fn get_config_paths() -> Vec<PathBuf> {
    const ENV_KEY_CONFIG: &'static str = "0XNFWT_INVENTORY_CONFIG";
    const CONFIG_JSON_NAME: &'static str = "0xnfwt_inventory.json";
    const XDG_CONFIG_DIR: &'static str = "0xnfwt_inventory";

    let mut paths = Vec::new();

    // First check for environment variable
    if let Ok(path) = std::env::var(ENV_KEY_CONFIG) {
        paths.push(PathBuf::from(path));
    }

    // Then check XDG_CONFIG_HOME or ~/.config
    if let Some(config_dir) = config_dir() {
        let xdg_path = config_dir.join(XDG_CONFIG_DIR).join(CONFIG_JSON_NAME);
        paths.push(xdg_path);
    }

    // Check home directory
    if let Some(home) = home_dir() {
        paths.push(home.join(CONFIG_JSON_NAME));
    }

    // Check current directory
    paths.push(PathBuf::from(CONFIG_JSON_NAME));

    // Check relative to executable
    paths.push(PathBuf::from(CONFIG_JSON_NAME));

    paths
}
