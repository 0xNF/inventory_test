use serde::{Deserialize, Serialize};

/// Represents a new inventory item with required name
#[derive(Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct InventoryItem {
    #[serde(default)]
    pub(crate) id: String,
    pub(crate) name: String, // Required for new items
    pub(crate) acquired_date: Option<String>,
    pub(crate) purchase_price: Option<i64>,
    pub(crate) purchase_currency: Option<String>,
    pub(crate) is_used: Option<bool>,
    pub(crate) received_from: Option<String>,
    pub(crate) model_number: Option<String>,
    pub(crate) serial_number: Option<String>,
    pub(crate) purchase_reference: Option<String>,
    pub(crate) notes: Option<String>,
    pub(crate) extra: Option<String>,
    pub(crate) future_purchase: Option<bool>,
}

/// Data structure for short inventory items
/// Represents an editable item where all fields are optional
#[derive(Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct EditableItem {
    #[serde(default)]
    #[serde(skip_serializing)]
    pub(crate) id: String,
    #[serde(default)]
    pub(crate) name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) acquired_date: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) purchase_price: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) purchase_currency: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) is_used: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) received_from: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) model_number: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) serial_number: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) purchase_reference: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) notes: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) extra: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) future_purchase: Option<bool>,
}

#[derive(Serialize)]
pub struct ShortInventoryItem {
    pub(crate) id: String,
    pub(crate) name: String,
    pub(crate) acquired_date: Option<String>,
}

/// Data structure for a newly added inventory item
#[derive(Serialize)]
pub(crate) struct NewInventoryItem {
    pub(crate) id: String,
    pub(crate) name: String,
    pub(crate) acquired_date: String,
}

/// Data structure for removal result
#[derive(Serialize)]
pub(crate) struct RemovalResult {
    pub(crate) success: bool,
    pub(crate) item_id: String,
    pub(crate) item_name: Option<String>,
    pub(crate) message: String,
}

#[derive(Serialize)]
pub(crate) struct EditResult {
    pub(crate) success: bool,
    pub(crate) item_id: String,
    pub(crate) message: String,
}
