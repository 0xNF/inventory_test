BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "Inventory" (
	-- UUID (v4, but v7 better) of the item. Assigned by the rs cli on insert
	"Id"	TEXT NOT NULL,
	-- Name of the item. Mandatory on insert
	"Name"	TEXT NOT NULL,
	-- RFC3339 format date of acquisition, which is either purchase or receipt by other source
	"AcquiredDate"	TEXT,
	-- Amount, in the following PurchaseCurrency field, paid for this item. Uses integers, not floats. For USD, use values denominated in pennies.
	"PurchasePrice"	INTEGER,
	-- ISO 4217 Currency Code of the purchase price (JPY, USX (pennies))
	"PurchaseCurrency"	TEXT,
	-- Whether this item is second-hand
	"IsUsed"	INTEGER DEFAULT 0,
	-- Source of receipt / place of purchase (Amazon, a friend, found on the street, etc)
	"ReceivedFrom"	TEXT,
	-- Serial Number of the item. This is currently conflated with Model Number. Should be separated out eventually.
	"SerialNumber"	TEXT,
	-- Reference to a purchase order associated with ReceivedFrom, such as an amazon number, or an electronics store receipt
	"PurchaseReference"	TEXT,
	-- Freeform textual notes on this item
	"Notes"	TEXT,
	-- A metadata field in JSON format, specifying extra data that should eventually be moved into the concrete schema
	"Extra"	TEXT,
	-- Whether this registration is for an item that not yet purchased
	"FuturePurchase"	INTEGER DEFAULT 0,
	PRIMARY KEY("Id")
) STRICT;
COMMIT;
