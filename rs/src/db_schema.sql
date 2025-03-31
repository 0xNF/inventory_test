BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "Audit" (
	"AuditId"	TEXT NOT NULL,
	"TableName"	TEXT NOT NULL,
	"RecordId"	TEXT NOT NULL,
	"Action"	TEXT NOT NULL,
	"ChangedFields"	TEXT,
	"OldValues"	TEXT,
	"NewValues"	TEXT,
	"Timestamp"	TEXT NOT NULL,
	"UserId"	TEXT,
	"ClientInfo"	TEXT,
	PRIMARY KEY("AuditId")
) STRICT;

CREATE TABLE IF NOT EXISTS "Inventory" (
	"Id"	TEXT NOT NULL,
	"Name"	TEXT NOT NULL,
	"AcquiredDate"	TEXT,
	"PurchasePrice"	INTEGER,
	"PurchaseCurrency"	TEXT,
	"IsUsed"	INTEGER DEFAULT 0,
	"ReceivedFrom"	TEXT,
	"SerialNumber"	TEXT,
	"PurchaseReference"	TEXT,
	"Notes"	TEXT,
	"Extra"	TEXT,
	"FuturePurchase"	INTEGER DEFAULT 0,
	PRIMARY KEY("Id")
) STRICT;

CREATE TRIGGER inventory_after_delete
AFTER DELETE ON Inventory
BEGIN
    INSERT INTO Audit (AuditId, TableName, RecordId, Action, ChangedFields, OldValues, NewValues, Timestamp, UserId, ClientInfo)
    VALUES (
        lower(hex(randomblob(16))),
        'Inventory',
        OLD.Id,
        'DELETE',
        NULL,
        json_object(
            'Id', OLD.Id,
            'Name', OLD.Name,
            'AcquiredDate', OLD.AcquiredDate,
            'PurchasePrice', OLD.PurchasePrice,
            'PurchaseCurrency', OLD.PurchaseCurrency,
            'IsUsed', OLD.IsUsed,
            'ReceivedFrom', OLD.ReceivedFrom,
            'SerialNumber', OLD.SerialNumber,
            'PurchaseReference', OLD.PurchaseReference,
            'Notes', OLD.Notes,
            'Extra', OLD.Extra,
            'FuturePurchase', OLD.FuturePurchase
        ), -- JSON with values being deleted
        NULL, -- No new values for deletes
        datetime('now'),
        NULL,
        NULL
    );
END;
CREATE TRIGGER inventory_after_insert 
AFTER INSERT ON Inventory
BEGIN
    INSERT INTO Audit (AuditId, TableName, RecordId, Action, ChangedFields, OldValues, NewValues, Timestamp, UserId, ClientInfo)
    VALUES (
        lower(hex(randomblob(16))), -- Generates a random UUID 
        'Inventory',
        NEW.Id,
        'INSERT',
        NULL, -- No changed fields for inserts
        NULL, -- No old values for inserts
             json_object(
            'Id', NEW.Id,
            'Name', NEW.Name,
            'AcquiredDate', NEW.AcquiredDate,
            'PurchasePrice', NEW.PurchasePrice,
            'PurchaseCurrency', NEW.PurchaseCurrency,
            'IsUsed', NEW.IsUsed,
            'ReceivedFrom', NEW.ReceivedFrom,
            'SerialNumber', NEW.SerialNumber,
            'PurchaseReference', NEW.PurchaseReference,
            'Notes', NEW.Notes,
            'Extra', NEW.Extra,
            'FuturePurchase', NEW.FuturePurchase
        ), -- JSON with all new values
        datetime('now'),
        NULL,
        NULL
    );
END;
CREATE TRIGGER inventory_after_update
AFTER UPDATE ON Inventory
BEGIN
    INSERT INTO Audit (AuditId, TableName, RecordId, Action, ChangedFields, OldValues, NewValues, Timestamp, UserId, ClientInfo)
    VALUES (
        lower(hex(randomblob(16))),
        'Inventory',
        NEW.Id,
        'UPDATE',
        -- Detects and records which fields changed as a JSON array
        (WITH changes(fields) AS (
            SELECT json_group_array(field)
            FROM (
                SELECT 'Name' AS field WHERE OLD.Name IS NOT NEW.Name
                UNION ALL SELECT 'AcquiredDate' WHERE OLD.AcquiredDate IS NOT NEW.AcquiredDate
                UNION ALL SELECT 'PurchasePrice' WHERE OLD.PurchasePrice IS NOT NEW.PurchasePrice
                UNION ALL SELECT 'PurchaseCurrency' WHERE OLD.PurchaseCurrency IS NOT NEW.PurchaseCurrency
                UNION ALL SELECT 'IsUsed' WHERE OLD.IsUsed IS NOT NEW.IsUsed
                UNION ALL SELECT 'ReceivedFrom' WHERE OLD.ReceivedFrom IS NOT NEW.ReceivedFrom
                UNION ALL SELECT 'SerialNumber' WHERE OLD.SerialNumber IS NOT NEW.SerialNumber
                UNION ALL SELECT 'PurchaseReference' WHERE OLD.PurchaseReference IS NOT NEW.PurchaseReference
                UNION ALL SELECT 'Notes' WHERE OLD.Notes IS NOT NEW.Notes
                UNION ALL SELECT 'Extra' WHERE OLD.Extra IS NOT NEW.Extra
                UNION ALL SELECT 'FuturePurchase' WHERE OLD.FuturePurchase IS NOT NEW.FuturePurchase
            )
        ) SELECT fields FROM changes),
        json_object(
            'Id', OLD.Id,
            'Name', OLD.Name,
            'AcquiredDate', OLD.AcquiredDate,
            'PurchasePrice', OLD.PurchasePrice,
            'PurchaseCurrency', OLD.PurchaseCurrency,
            'IsUsed', OLD.IsUsed,
            'ReceivedFrom', OLD.ReceivedFrom,
            'SerialNumber', OLD.SerialNumber,
            'PurchaseReference', OLD.PurchaseReference,
            'Notes', OLD.Notes,
            'Extra', OLD.Extra,
            'FuturePurchase', OLD.FuturePurchase
        ), -- JSON with old values
        json_object(
            'Id', NEW.Id,
            'Name', NEW.Name,
            'AcquiredDate', NEW.AcquiredDate,
            'PurchasePrice', NEW.PurchasePrice,
            'PurchaseCurrency', NEW.PurchaseCurrency,
            'IsUsed', NEW.IsUsed,
            'ReceivedFrom', NEW.ReceivedFrom,
            'SerialNumber', NEW.SerialNumber,
            'PurchaseReference', NEW.PurchaseReference,
            'Notes', NEW.Notes,
            'Extra', NEW.Extra,
            'FuturePurchase', NEW.FuturePurchase
        ), -- JSON with new values
        datetime('now'),
        NULL,
        NULL
    );
END;
COMMIT;
