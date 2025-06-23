package mcpserver

import (
	"0xnfwtiventory/internal/mcp_logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"inventory_shared"
	"strings"
	"time"

	"inventory_shared/wtlogger"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type InventoryMCPServer struct {
	Mcp          *server.MCPServer
	InventoryCLI *inventory_shared.InventoryProg
	Config       WTServerConfig
}

func NewInventoryMCPServer(serverName string, version string) *server.MCPServer {
	const resourceCanSubscribe = true
	const resourceWillNotifyChanged = true
	const promptWillNotifyChanged = true

	options := []server.ServerOption{
		server.WithResourceCapabilities(resourceCanSubscribe, resourceWillNotifyChanged),
		server.WithPromptCapabilities(promptWillNotifyChanged),
		server.WithLogging(),
	}
	/** This STDIO server will log errors to this value */
	s := server.NewMCPServer(serverName, version, options...)

	/** Register handler for the Logging Min Level function by MCP Clients */
	s.AddNotificationHandler("logging/setLevel", func(ctx context.Context, notification mcp.JSONRPCNotification) {
		handleSetMinimumLogLevel(ctx, notification, s)
	})

	return s
}

func handleSetMinimumLogLevel(ctx context.Context, notification mcp.JSONRPCNotification, s *server.MCPServer) {
	// TODO(nf, 04/01/25): MCP-Go does not yet support easily responding to this Minimum Log Level message, refactor this method when it does
	// including getting rid of the `s` parameter and its SendNotificationToClient responses
	level, ok := notification.Params.AdditionalFields["level"].(string)
	if !ok {
		s.SendNotificationToClient(ctx, "-32602", map[string]any{"message": "level not supplied, or not a string"})
		return
	} else {
		mcpLevel := mcp.LoggingLevel(level)
		mcp_logger.GetLogger(s, nil).InfoWithFields(ctx, "Setting Minimum Log Level", map[string]any{"level": mcpLevel})
		err := mcp_logger.SetMinimumLogLevel(mcpLevel)
		if err != nil {
			s.SendNotificationToClient(ctx, "-32602", map[string]any{"message": fmt.Sprintf("invalid level: %v", err.Error())})
			return
		}
	}
}

func LoadServer(serverName string, version string, config WTServerConfig) (*InventoryMCPServer, error) {
	wtlogger.GetLogger().Info("Loading new MCP Server")
	/* One or the other must be set, but not neither and not both */
	if config.CLIPath == nil && config.WebServerAddress == nil {
		return nil, errors.New("neither CLIPath nor WebServerAddress were set. Either one or the other must be specified")
	} else if config.CLIPath != nil && config.WebServerAddress != nil {
		return nil, errors.New("both CLIPath and WebServerAddrss were not. Only one may be specified")
	}

	var prog *inventory_shared.InventoryProg
	if config.CLIPath != nil {
		wtlogger.GetLogger().Debug("CLI Path was set, will make a new CLI Program path, and test for correctness")
		p := inventory_shared.NewInventoryProg(*config.CLIPath)
		prog = &p
		err := prog.Test()
		if err != nil {
			return nil, fmt.Errorf("binary specified by CLIPath did not respond to validation: %w", err)
		}
	}

	server := InventoryMCPServer{
		NewInventoryMCPServer(serverName, version),
		prog,
		config,
	}

	/* Add Tools */
	mcp_logger.GetLogger(server.Mcp, nil)
	wtlogger.GetLogger().Info("Adding AddItem tool")
	addToolAddItem(server)
	wtlogger.GetLogger().Info("Adding EditItem tool")
	addToolEditItem(server)
	wtlogger.GetLogger().Info("Adding ListItems tool")
	addToolListItems(server)

	/* Add Prompts */
	wtlogger.GetLogger().Info("Adding Upgrades prompt")
	AddPromptUpgrades(server)
	wtlogger.GetLogger().Info("Adding Query By Name prompt")
	AddPromptQueryByName(server)

	/* Add Resources */

	return &server, nil
}

func AddPromptQueryByName(s InventoryMCPServer) {
	logger := mcp_logger.GetLogger(s.Mcp, nil)
	promptName := "inventoryQueryByName"
	promptDescription := "Using the information in the inventory db, returns any items that have a matching or similar name"

	const promptArgName = "name"
	prompt := mcp.NewPrompt(promptName, mcp.WithPromptDescription(promptDescription), mcp.WithArgument(promptArgName,
		mcp.ArgumentDescription("The name of the item to look for in the inventory"),
		mcp.RequiredArgument(),
	))

	s.Mcp.AddPrompt(prompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		logger.Info(ctx, fmt.Sprintf("Invoking %s Prompt", promptName))

		name := strings.TrimSpace(request.Params.Arguments[promptArgName])
		if len(name) == 0 {
			return nil, errors.New("promptQuery: name not supplied, or was empty")
		}

		return &mcp.GetPromptResult{
			Description: promptDescription,
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Find all items in my inventory with a name like '%s'", name),
					},
				},
			},
		}, nil
	})

}

func AddPromptUpgrades(s InventoryMCPServer) {
	logger := mcp_logger.GetLogger(s.Mcp, nil)
	promptName := "inventorySuggestedUpgrades"
	promptDescription := "Using the information in the inventory db, asks for items that could be upgraded, either because some items are old, or broken, or slow."

	prompt := mcp.NewPrompt(promptName, mcp.WithPromptDescription(promptDescription))

	s.Mcp.AddPrompt(prompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		logger.Info(ctx, fmt.Sprintf("Invoking %s Prompt", promptName))

		return &mcp.GetPromptResult{
			Description: promptDescription,
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: "From my inventory, what items should be considered for upgrading or retirement?",
					},
				},
			},
		}, nil
	})
}

func addToolListItems(s InventoryMCPServer) {
	logger := mcp_logger.GetLogger(s.Mcp, nil)
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
		logger.Info(ctx, fmt.Sprintf("Invoking %s tool", toolName))
		logger.DebugWithFields(ctx, "ListItems params", map[string]any{"fields": request.Params})
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

		page, err := s.InventoryCLI.List(limit, offset, sortBy, orderBy, filter, []string{})
		if err != nil {
			return nil, fmt.Errorf("list: failed to run list command: %w", err)
		}

		m, err := json.Marshal(page)
		if err != nil {
			return nil, fmt.Errorf("list: failed to marshal: %w", err)
		}
		s := string(m)

		logger.Info(ctx, "ListItems success")
		return mcp.NewToolResultText(s), nil

	})
}

func addToolEditItem(s InventoryMCPServer) {
	logger := mcp_logger.GetLogger(s.Mcp, nil)
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
		mcp.WithString("model_number", mcp.Description("Model Number of the item, if available"), mcp.MinLength(1)),
		mcp.WithString("serial_number", mcp.Description("Serial Number of the item, if available."), mcp.MinLength(1)),
		mcp.WithString("purchase_reference", mcp.Description("If avaialble, a reference to a specific order number, or receipt code for this item"), mcp.MinLength(1)),
		mcp.WithString("notes", mcp.Description("Free-form entry of any other information that relates to this item. Try to keep brief."), mcp.MinLength(1), mcp.MaxLength(1028)),
	)

	s.Mcp.AddTool(editItemTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info(ctx, fmt.Sprintf("Invoking %s tool", toolName))
		logger.DebugWithFields(ctx, "EditItem params", map[string]any{"fields": request.Params})

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

		/* Parse Model Number */
		raw_model := request.Params.Arguments["model_number"]
		if raw_model != nil {
			model, ok := raw_model.(string)
			if ok {
				if len(model) < 1 {
					return nil, fmt.Errorf("edit: model_number must be at least 1 character long")
				}
				item.ModelNumber = model
			} else {
				return nil, fmt.Errorf("edit: model_number was supplied, but was not a string")
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

		/* Parse Purchae Reference */
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

		edited, err := s.InventoryCLI.Edit(id, item)
		if err != nil {
			return nil, err
		}

		m, err := json.Marshal(edited)
		if err != nil {
			return nil, fmt.Errorf("edit: failed to marshal: %w", err)
		}
		s := string(m)

		logger.Info(ctx, "EditItem success")
		return mcp.NewToolResultText(s), nil
	})
}

func addToolAddItem(s InventoryMCPServer) {
	logger := mcp_logger.GetLogger(s.Mcp, nil)
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
		mcp.WithString("model_number", mcp.Description("Model Number of the item, if available."), mcp.MinLength(1)),
		mcp.WithString("serial_number", mcp.Description("Serial Number of the item, if available."), mcp.MinLength(1)),
		mcp.WithString("purchase_reference", mcp.Description("If avaialble, a reference to a specific order number, or receipt code for this item"), mcp.MinLength(1)),
		mcp.WithString("notes", mcp.Description("Free-form entry of any other information that relates to this item. Try to keep brief."), mcp.MinLength(1), mcp.MaxLength(1028)),
	)

	s.Mcp.AddTool(addItemTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info(ctx, fmt.Sprintf("Invoking %s tool", toolName))
		logger.DebugWithFields(ctx, "AddItem params", map[string]any{"fields": request.Params})

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

		/* Parse Model Number */
		raw_model := request.Params.Arguments["model_number"]
		if raw_model != nil {
			model, ok := raw_model.(string)
			if ok {
				if len(model) < 1 {
					return nil, fmt.Errorf("edit: model_number must be at least 1 character long")
				}
				item.ModelNumber = model
			} else {
				return nil, fmt.Errorf("edit: model_number was supplied, but was not a string")
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

		/* Parse Purchase Reference */
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

		added, err := s.InventoryCLI.Add(item)
		if err != nil {
			return nil, err
		}

		m, err := json.Marshal(added)
		if err != nil {
			return nil, fmt.Errorf("add: failed to marshal: %w", err)
		}
		s := string(m)

		logger.Info(ctx, "AddItem success")
		return mcp.NewToolResultText(s), nil
	},
	)

}
