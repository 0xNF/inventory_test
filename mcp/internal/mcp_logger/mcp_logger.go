package mcp_logger

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// MCPMinLogLevel is the minimum Log Level as specified by [https://spec.modelcontextprotocol.io/specification/2025-03-26/server/utilities/logging/]
// This value defaults to [Info].
//
// Log events that are below the minimum level will not be sent to the MCP Client
var MCPMinLogLevel mcp.LoggingLevel = mcp.LoggingLevelInfo

// LogLevelIntMap gets the Sting value of the log level by priority
//
// For the reverse map, see [LogLevelValueMap]
var LogLevelIntMap map[int]mcp.LoggingLevel = map[int]mcp.LoggingLevel{
	10: mcp.LoggingLevelDebug,
	2:  mcp.LoggingLevelInfo,
	30: mcp.LoggingLevelNotice,
	40: mcp.LoggingLevelWarning,
	50: mcp.LoggingLevelError,
	60: mcp.LoggingLevelCritical,
	70: mcp.LoggingLevelAlert,
	80: mcp.LoggingLevelEmergency,
}

// LogLevelValueMap gets the proprity integer value of the log level by its name
//
// For the reverse map, see [LogLevelIntMap]
var LogLevelValueMap map[mcp.LoggingLevel]int = map[mcp.LoggingLevel]int{
	mcp.LoggingLevelDebug:     10,
	mcp.LoggingLevelInfo:      20,
	mcp.LoggingLevelNotice:    30,
	mcp.LoggingLevelWarning:   40,
	mcp.LoggingLevelError:     50,
	mcp.LoggingLevelCritical:  60,
	mcp.LoggingLevelAlert:     70,
	mcp.LoggingLevelEmergency: 80,
}

// MCPLogEvent follows the conventions outlined in [https://spec.modelcontextprotocol.io/specification/2025-03-26/server/utilities/logging/#log-message-notifications]
// for what data to send to connected MCP Clients
type MCPLogEvent struct {
	// Level denotes the severity level, not optional
	Level mcp.LoggingLevel `json:"level"`
	// Logger is the name of the logger that generated this event, optional
	Logger *string `json:"logger,omitempty"`
	// Data is any arbitrary JSON-serializable data, may be nil or empty
	Data any `json:"data,omitempty"`
}

// toMapAny is a convenience method for putting MCPLogEvents into the proper format for MCP Client resposne serialization
func (event *MCPLogEvent) toMapAny() map[string]any {
	result := make(map[string]any)

	result["level"] = event.Level
	if event.Logger != nil {
		result["logger"] = *event.Logger
	}
	result["data"] = event.Data

	return result
}

// SetMinimumLogLevel is defined by [https://spec.modelcontextprotocol.io/specification/2025-03-26/server/utilities/logging/#protocol-messages]
// and sets the minimum level that log events should be in order to generate a notification to the client.
//
// If the log level supplied is invalid, an error is returned
func SetMinimumLogLevel(level mcp.LoggingLevel) error {
	if LogLevelValueMap[level] < LogLevelValueMap[mcp.LoggingLevelDebug] {
		return fmt.Errorf("invalid log level: %s", level)
	}
	MCPMinLogLevel = level
	return nil
}
