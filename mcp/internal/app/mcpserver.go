package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"inventory_shared"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type InventoryMCPServer struct {
	Mcp     *server.MCPServer
	Program *inventory_shared.InventoryProg
	Config  Config
}

func NewInventoryMCPServer() *server.MCPServer {
	const serverName = "0xNFWT Inventory Manager"
	const version = "0.1.0"
	/* This server supports a Resource (rs cli).
	* It subscribes to changes to the resource (true),
	* and does not support notifications about when the list of resources changes (false),
	* because this server only has one resource, the cli.
	 */
	withRes := server.WithResourceCapabilities(true, false)

	/** This STDIO server will log errors to this value */
	// withErrorLogger := server.WithErrorLogger(log.Default())
	s := server.NewMCPServer(serverName, version, withRes)

	return s
}

func LoadServer(config Config) (*InventoryMCPServer, error) {
	/* One or the other must be set, but not neither and not both */
	if config.CLIPath == nil && config.WebServerAddress == nil {
		return nil, errors.New("neither CLIPath nor WebServerAddress were set. Either one or the other must be specified")
	} else if config.CLIPath != nil && config.WebServerAddress != nil {
		return nil, errors.New("both CLIPath and WebServerAddrss were not. Only one may be specified")
	}

	var prog *inventory_shared.InventoryProg
	if config.CLIPath != nil {
		p := inventory_shared.NewInventoryProg(*config.CLIPath)
		prog = &p
		err := prog.Test()
		if err != nil {
			return nil, fmt.Errorf("binary specified by CLIPath did not respond to validation: %w", err)
		}
	}

	server := InventoryMCPServer{
		NewInventoryMCPServer(),
		prog,
		config,
	}
	addToolAddItem(server)
	addToolEditItem(server)
	addToolListItems(server)

	return &server, nil
}

func addToolListItems(s InventoryMCPServer) {
	toolName := "listInventoryItems"
	toolDescription := "Returns a paged list of inventory item. If a filter is supplied, only items matching the filter are returned. Otherwise, it lists all items. The paging can be controlled with the limit and offset fields."
	listItemsTool := mcp.NewTool(
		toolName,
		mcp.WithDescription(toolDescription),
		mcp.WithString("limit", mcp.Description("How many items to return at once. Use with offset to page through the results. May be omitted, but not negative."), mcp.Min(0)),
		mcp.WithString("offset", mcp.Description("Paging offset to start returning items from. Use with limit to page through the results. Maybe omitted, but not negative."), mcp.Min(0)),
		mcp.WithString("sortBy", mcp.Description("Field to sort the returned paged items by. Defaults to 'Name'."), mcp.Enum(inventory_shared.FIELDS_ARR...)),
		mcp.WithString("orderBy", mcp.Description("Whether to sort returned page items in ascending or descending order. Defaults to asc."), mcp.Enum("asc", "desc")),
		mcp.WithNumber("filter", mcp.Description("If present, this text will be used to perform a regex search over the inventory items looking for matches.")),
	)

	s.Mcp.AddTool(listItemsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var limit *uint32
		var offset *uint32
		var sortBy string
		var orderBy string
		var filter string
		/* Parse limit */
		raw_limit := request.Params.Arguments["limit"]
		if raw_limit != nil {
			lim, ok := raw_limit.(uint32)
			if ok {
				limit = &lim
			} else {
				return nil, fmt.Errorf("list: limit was supplied, but was not a non-negative integer")
			}
		}

		/* Parse offset */
		raw_offset := request.Params.Arguments["offset"]
		if raw_offset != nil {
			off, ok := raw_offset.(uint32)
			if ok {
				offset = &off
			} else {
				return nil, fmt.Errorf("list: offset was supplied, but was not a non-negative integer")
			}
		}

		/* Parse sortBy */
		raw_sortBy := request.Params.Arguments["sortBy"]
		if raw_sortBy != nil {
			sort, ok := raw_sortBy.(string)
			if ok {
				/* find in field arr */
				found := false
				for _, field := range inventory_shared.FIELDS_ARR {
					if field == sort {
						found = true
						sortBy = field
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("list: sortBy was supplied but not was a valid value")
				}
			} else {
				return nil, fmt.Errorf("list: sortBy was supplied, but was not a string")
			}
		}

		/* Parse orderBy */
		raw_orderBy := request.Params.Arguments["orderBy"]
		if raw_orderBy != nil {
			order, ok := raw_orderBy.(string)
			if ok {
				/* find in field arr */
				found := false
				for _, field := range []string{"asc", "desc"} {
					if field == strings.ToLower(order) {
						found = true
						orderBy = order
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("list: orderBy was supplied but not was a valid value")
				}
			} else {
				return nil, fmt.Errorf("list: orderBy was supplied, but was not a string")
			}
		}

		/* Parse filter */
		raw_filter := request.Params.Arguments["filter"]
		if raw_filter != nil {
			fil := strings.TrimSpace(fmt.Sprint(raw_filter))
			filter = fil
		}

		page, err := s.Program.List(limit, offset, sortBy, orderBy, filter, []string{})
		if err != nil {
			return nil, fmt.Errorf("list: failed to run list command: %w", err)
		}

		m, err := json.Marshal(page)
		if err != nil {
			return nil, fmt.Errorf("list: failed to marshal: %w", err)
		}
		s := string(m)
		return mcp.NewToolResultText(s), nil

	})
}

func addToolEditItem(s InventoryMCPServer) {
	toolName := "editInventoryItem"
	toolDescription := "Edits an existing inventory item with the supplied fields. Only Id is required, everything else is optional."
	editItemTool := mcp.NewTool(
		toolName,
		mcp.WithDescription(toolDescription),
		mcp.WithString("id", mcp.Required(), mcp.Description("UUID of the inventory item to edit"), mcp.MinLength(36), mcp.MaxLength(36)),
		mcp.WithString("name", mcp.Description("Name or short description of the inventory name"), mcp.MinLength(1)),
		mcp.WithString("acquired_date", mcp.Description("RFC3339 full-date formatted string of the date this item was acquired. Acquisition is defined by either purchase, or by through any other means, such as donation. A full-date is formatted like YYYY-mm-DD (e.g, 2025-03-07). If not supplied, will default to today's date."), mcp.MinLength(10), mcp.MaxLength(10)),
		mcp.WithString("purchase_currency", mcp.Description("ISO-4217 3-letter Currency Code of the currency this item was purchased in, for example JPY, USD, EUR, etc."), mcp.MinLength(3), mcp.MaxLength(3)),
		mcp.WithNumber("purchase_price", mcp.Description("Price item was purchased for, in the currency defined by PurchaseCurrency. If PurchaseCurrency is not supplied, this value is assumed to be in Japanese Yen (JPY). Value may be null, but may not be negative or fractional. Only full integers are accepted. (i >= 0)"), mcp.Min(0)),
		mcp.WithBoolean("is_used", mcp.Description("Whether this item is a Used Item, i.e., whether it was received second-hand or purchased via auction")),
		mcp.WithString("received_from", mcp.Description("Source of acquisition for this item. If purchased from a website (e.g., 'Amazon.com'), then it is the website, if received from a friend, then it is the friends name (e.g., Takumi), etc."), mcp.MinLength(2)),
		mcp.WithString("serial_number", mcp.Description("Serial or Model Number of the item, if available."), mcp.MinLength(1)),
		mcp.WithString("purchase_reference", mcp.Description("If avaialble, a reference to a specific order number, or receipt code for this item"), mcp.MinLength(1)),
		mcp.WithString("notes", mcp.Description("Free-form entry of any other information that relates to this item. Try to keep brief."), mcp.MinLength(1), mcp.MaxLength(1028)),
	)

	s.Mcp.AddTool(editItemTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var id string
		item := inventory_shared.EditItemRequest{}
		/* Parse Id */
		raw_id := request.Params.Arguments["id"]
		if raw_id != nil {
			id2, ok := raw_id.(string)
			if ok {
				if len(id2) != 36 {
					return nil, fmt.Errorf("edit: id was supplied, but mismatched length. Must be a 36 character uuid")
				}
				id = id2
			} else {
				return nil, fmt.Errorf("edit: id was supplied, but was not a string")
			}
		} else {
			return nil, fmt.Errorf("edit: id must be supplied")
		}
		/* Parse name */
		raw_name := request.Params.Arguments["name"]
		if raw_name != nil {
			name, ok := raw_name.(string)
			if ok {
				if len(name) < 1 {
					return nil, fmt.Errorf("edit: name must be at least 1 character ong")
				}
				item.Name = name
			} else {
				return nil, fmt.Errorf("edit: name was supplied, but was not a string")
			}
		}

		/* Parse Date */
		raw_date := request.Params.Arguments["acquired_date"]
		if raw_date != nil {
			acquired_date, ok := raw_date.(string)
			if ok {
				_, err := time.Parse("2006-01-02", acquired_date)
				if err != nil {
					return nil, fmt.Errorf("edit: acquired_date must be in RFC3339 format")
				}
				item.AcquiredDate = acquired_date
			} else {
				return nil, fmt.Errorf("edit: acquired_date was supplied but was not a string")
			}
		}

		/* Parse Purchase Currency */
		raw_currency := request.Params.Arguments["purchase_currency"]
		if raw_currency != nil {
			currency, ok := raw_currency.(string)
			if ok {
				if len(currency) != 3 {
					return nil, fmt.Errorf("edit: purchase_currency must be a 3 letter currency code in ISO-4217 format")
				}
				item.PurchaseCurrency = currency
			} else {
				return nil, fmt.Errorf("edit: purchase_currency was supplied, but was not a string")
			}
		}

		/* Parse Purchase Price */
		raw_price := request.Params.Arguments["purchase_price"]
		if raw_price != nil {
			price, ok := raw_price.(float64)
			if ok {
				if price < 0 {
					return nil, fmt.Errorf("edit: purchase_price must be non-negative")
				}
				item.PurchasePrice = &price
			} else {
				return nil, fmt.Errorf("edit: purchase_price was supplied, but was not an integer")
			}
		}

		/* Parse Is Used */
		raw_used := request.Params.Arguments["is_used"]
		if raw_used != nil {
			used, ok := raw_used.(bool)
			if !ok {
				return nil, fmt.Errorf("edit: is_used was supplied, but was not a boolean")
			}
			item.IsUsed = &used
		}

		/* Parse Purchase Currency */
		raw_received_from := request.Params.Arguments["received_from"]
		if raw_received_from != nil {
			received_from, ok := raw_received_from.(string)
			if ok {
				if len(received_from) < 1 {
					return nil, fmt.Errorf("edit: received_from must be at least 1 character long")
				}
				item.ReceivedFrom = received_from
			} else {
				return nil, fmt.Errorf("edit: received_from was supplied, but was not a string")
			}
		}

		/* Parse Serial Number */
		raw_serial := request.Params.Arguments["serial_number"]
		if raw_serial != nil {
			serial, ok := raw_serial.(string)
			if ok {
				if len(serial) < 1 {
					return nil, fmt.Errorf("edit: serial_number must be at least 1 character long")
				}
				item.SerialNumber = serial
			} else {
				return nil, fmt.Errorf("edit: serial_number was supplied, but was not a string")
			}
		}

		/* Parse Serial Number */
		raw_reference := request.Params.Arguments["purchase_reference"]
		if raw_reference != nil {
			reference, ok := raw_reference.(string)
			if ok {
				if len(reference) < 1 {
					return nil, fmt.Errorf("edit: purchase_reference must be at least 1 character long")
				}
				item.PurchaseReference = reference
			} else {
				return nil, fmt.Errorf("edit: purchase_reference was supplied, but was not a string")
			}
		}

		/* Parse Notes */
		raw_notes := request.Params.Arguments["notes"]
		if raw_notes != nil {
			notes, ok := raw_notes.(string)
			if ok {
				if len(notes) < 1 {
					return nil, fmt.Errorf("edit: notes must be at least 1 character long")
				}
				item.Notes = notes
			} else {
				return nil, fmt.Errorf("edit: notes was supplied, but was not a string")
			}
		}

		edited, err := s.Program.Edit(id, item)
		if err != nil {
			return nil, err
		}

		m, err := json.Marshal(edited)
		if err != nil {
			return nil, fmt.Errorf("edit: failed to marshal: %w", err)
		}
		s := string(m)
		return mcp.NewToolResultText(s), nil
	})
}

func addToolAddItem(s InventoryMCPServer) {
	toolName := "addInventoryItem"
	toolDescription := "Registers a new item to be added to the inventory database"
	addItemTool := mcp.NewTool(
		toolName,
		mcp.WithDescription(toolDescription),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name or short description of the inventory name"), mcp.MinLength(1)),
		mcp.WithString("acquired_date", mcp.Description("RFC3339 full-date formatted string of the date this item was acquired. Acquisition is defined by either purchase, or by through any other means, such as donation. A full-date is formatted like YYYY-mm-DD (e.g, 2025-03-07). If not supplied, will default to today's date."), mcp.MinLength(10), mcp.MaxLength(10)),
		mcp.WithString("purchase_currency", mcp.Description("ISO-4217 3-letter Currency Code of the currency this item was purchased in, for example JPY, USD, EUR, etc."), mcp.MinLength(3), mcp.MaxLength(3)),
		mcp.WithNumber("purchase_price", mcp.Description("Price item was purchased for, in the currency defined by PurchaseCurrency. If PurchaseCurrency is not supplied, this value is assumed to be in Japanese Yen (JPY). Value may be null, but may not be negative or fractional. Only full integers are accepted. (i >= 0)"), mcp.Min(0)),
		mcp.WithBoolean("is_used", mcp.Description("Whether this item is a Used Item, i.e., whether it was received second-hand or purchased via auction")),
		mcp.WithString("received_from", mcp.Description("Source of acquisition for this item. If purchased from a website (e.g., 'Amazon.com'), then it is the website, if received from a friend, then it is the friends name (e.g., Takumi), etc."), mcp.MinLength(2)),
		mcp.WithString("serial_number", mcp.Description("Serial or Model Number of the item, if available."), mcp.MinLength(1)),
		mcp.WithString("purchase_reference", mcp.Description("If avaialble, a reference to a specific order number, or receipt code for this item"), mcp.MinLength(1)),
		mcp.WithString("notes", mcp.Description("Free-form entry of any other information that relates to this item. Try to keep brief."), mcp.MinLength(1), mcp.MaxLength(1028)),
	)

	s.Mcp.AddTool(addItemTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		item := inventory_shared.InventoryItem{}
		/* Parse name */
		raw_name := request.Params.Arguments["name"]
		if raw_name != nil {
			name, ok := raw_name.(string)
			if ok {
				if len(name) < 1 {
					return nil, fmt.Errorf("add: name must be at least 1 character ong")
				}
				item.Name = name
			} else {
				return nil, fmt.Errorf("add: name was supplied, but was not a string")
			}
		} else {
			return nil, fmt.Errorf("add: name was not present, but is required")
		}

		/* Parse Date */
		raw_date := request.Params.Arguments["acquired_date"]
		if raw_date != nil {
			acquired_date, ok := raw_date.(string)
			if ok {
				_, err := time.Parse("2006-01-02", acquired_date)
				if err != nil {
					return nil, fmt.Errorf("add: acquired_date must be in RFC3339 format")
				}
				item.AcquiredDate = acquired_date
			} else {
				return nil, fmt.Errorf("add: acquired_date was supplied but was not a string")
			}
		}

		/* Parse Purchase Currency */
		raw_currency := request.Params.Arguments["purchase_currency"]
		if raw_currency != nil {
			currency, ok := raw_currency.(string)
			if ok {
				if len(currency) != 3 {
					return nil, fmt.Errorf("add: purchase_currency must be a 3 letter currency code in ISO-4217 format")
				}
				item.PurchaseCurrency = currency
			} else {
				return nil, fmt.Errorf("add: purchase_currency was supplied, but was not a string")
			}
		}

		/* Parse Purchase Price */
		raw_price := request.Params.Arguments["purchase_price"]
		if raw_price != nil {
			price, ok := raw_price.(int64)
			if ok {
				if price < 0 {
					return nil, fmt.Errorf("add: purchase_price must be non-negative")
				}
				item.PurchasePrice = price
			} else {
				return nil, fmt.Errorf("add: purchase_price was supplied, but was not an integer")
			}
		}

		/* Parse Is Used */
		raw_used := request.Params.Arguments["is_used"]
		if raw_used != nil {
			used, ok := raw_used.(bool)
			if !ok {
				return nil, fmt.Errorf("add: is_used was supplied, but was not a boolean")
			}
			item.IsUsed = &used
		}

		/* Parse Purchase Currency */
		raw_received_from := request.Params.Arguments["received_from"]
		if raw_received_from != nil {
			received_from, ok := raw_received_from.(string)
			if ok {
				if len(received_from) < 1 {
					return nil, fmt.Errorf("add: received_from must be at least 1 character long")
				}
				item.ReceivedFrom = received_from
			} else {
				return nil, fmt.Errorf("add: received_from was supplied, but was not a string")
			}
		}

		/* Parse Serial Number */
		raw_serial := request.Params.Arguments["serial_number"]
		if raw_serial != nil {
			serial, ok := raw_serial.(string)
			if ok {
				if len(serial) < 1 {
					return nil, fmt.Errorf("add: serial_number must be at least 1 character long")
				}
				item.SerialNumber = serial
			} else {
				return nil, fmt.Errorf("add: serial_number was supplied, but was not a string")
			}
		}

		/* Parse Serial Number */
		raw_reference := request.Params.Arguments["purchase_reference"]
		if raw_reference != nil {
			reference, ok := raw_reference.(string)
			if ok {
				if len(reference) < 1 {
					return nil, fmt.Errorf("add: purchase_reference must be at least 1 character long")
				}
				item.PurchaseReference = reference
			} else {
				return nil, fmt.Errorf("add: purchase_reference was supplied, but was not a string")
			}
		}

		/* Parse Notes */
		raw_notes := request.Params.Arguments["notes"]
		if raw_notes != nil {
			notes, ok := raw_notes.(string)
			if ok {
				if len(notes) < 1 {
					return nil, fmt.Errorf("add: notes must be at least 1 character long")
				}
				item.Notes = notes
			} else {
				return nil, fmt.Errorf("add: notes was supplied, but was not a string")
			}
		}

		added, err := s.Program.Add(item)
		if err != nil {
			return nil, err
		}

		m, err := json.Marshal(added)
		if err != nil {
			return nil, fmt.Errorf("add: failed to marshal: %w", err)
		}
		s := string(m)
		return mcp.NewToolResultText(s), nil
	},
	)

}
