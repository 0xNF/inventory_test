package mcp_logger

import (
	"context"
	"inventory_shared/wtlogger"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPLogger struct {
	// logger is the underlying logger that provides writing to stdout and to the filesystem
	logger *wtlogger.BufferedLogger
	// postToServerFunc is responsible for sending events back to the a conneted MCP Client
	postToServerFunc func(ctx context.Context, mcpLogEvent MCPLogEvent) error
}

// GetLogger returns the singleton instance of BufferedLogger
func GetLogger(s *server.MCPServer, name *string) *MCPLogger {
	wtlogger := wtlogger.GetLogger()

	// TODO(nf, 04/01/25): MCP-Go does not yet easily support the Logging responses, refactor this away when it does

	// Wrap the `SendNotificationToClient` function to make it easy to send MCPLogEvents
	f := func(ctx context.Context, mcpEvent MCPLogEvent) error {
		if LogLevelValueMap[mcpEvent.Level] >= LogLevelValueMap[MCPMinLogLevel] {
			return s.SendNotificationToClient(ctx, "notifications/message", mcpEvent.toMapAny())
		}
		return nil
	}

	return &MCPLogger{
		logger:           wtlogger,
		postToServerFunc: f,
	}
}

// Log level methods
func (bl *MCPLogger) Debug(ctx context.Context, msg string) {
	bl.logger.Debug(msg)
	mcpEvent := MCPLogEvent{
		Level:  mcp.LoggingLevelDebug,
		Logger: nil,
		Data:   msg,
	}
	bl.postToServerFunc(ctx, mcpEvent)
}

func (bl *MCPLogger) Info(ctx context.Context, msg string) {
	bl.logger.Info(msg)
	mcpEvent := MCPLogEvent{
		Level:  mcp.LoggingLevelInfo,
		Logger: nil,
		Data:   msg,
	}
	bl.postToServerFunc(ctx, mcpEvent)
}

func (bl *MCPLogger) Warn(ctx context.Context, msg string) {
	bl.logger.Warn(msg)
	mcpEvent := MCPLogEvent{
		Level:  mcp.LoggingLevelWarning,
		Logger: nil,
		Data:   msg,
	}
	bl.postToServerFunc(ctx, mcpEvent)
}

func (bl *MCPLogger) Error(ctx context.Context, msg string) {
	bl.logger.Error(msg)
	mcpEvent := MCPLogEvent{
		Level:  mcp.LoggingLevelError,
		Logger: nil,
		Data:   msg,
	}
	bl.postToServerFunc(ctx, mcpEvent)
}

func (bl *MCPLogger) Fatal(ctx context.Context, msg string) {
	bl.logger.Fatal(msg)
	mcpEvent := MCPLogEvent{
		Level:  mcp.LoggingLevelEmergency,
		Logger: nil,
		Data:   msg,
	}
	bl.postToServerFunc(ctx, mcpEvent)
}

// Structured logging methods
func (bl *MCPLogger) DebugWithFields(ctx context.Context, msg string, fields map[string]interface{}) {
	bl.logger.DebugWithFields(msg, fields)
	mcpEvent := MCPLogEvent{
		Level:  mcp.LoggingLevelDebug,
		Logger: nil,
		Data:   msg,
	}
	bl.postToServerFunc(ctx, mcpEvent)
}

func (bl *MCPLogger) InfoWithFields(ctx context.Context, msg string, fields map[string]interface{}) {
	bl.logger.InfoWithFields(msg, fields)
	mcpEvent := MCPLogEvent{
		Level:  mcp.LoggingLevelInfo,
		Logger: nil,
		Data:   msg,
	}
	bl.postToServerFunc(ctx, mcpEvent)
}
