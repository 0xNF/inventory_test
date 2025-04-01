package wtlogger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogConfig struct {
	FilePath    *string
	MaxSize     int // megabytes
	MaxBackups  int
	MaxAge      int // days
	Compress    bool
	MinLogLevel zerolog.Level
	Console     bool // if true, also log to console
}

type BufferedLogger struct {
	buffer        *bytes.Buffer
	mutex         sync.Mutex
	logger        zerolog.Logger
	writer        io.Writer
	isInitialized bool
}

var (
	instance *BufferedLogger
	once     sync.Once
)

// GetLogger returns the singleton instance of BufferedLogger
func GetLogger() *BufferedLogger {
	once.Do(func() {
		instance = &BufferedLogger{
			buffer: bytes.NewBuffer(nil),
		}
		// Set up initial console logging
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
		instance.logger = zerolog.New(io.MultiWriter(instance.buffer, consoleWriter)).
			With().Timestamp().Caller().Logger()
	})
	return instance
}

// Initialize sets up the full logger with file writing capabilities
func (bl *BufferedLogger) Initialize(config LogConfig) error {
	bl.mutex.Lock()
	defer bl.mutex.Unlock()

	// Configure file rotation
	fpath := config.FilePath
	if fpath == nil {
		empty := ""
		fpath = &empty
	}
	lumberjackLogger := &lumberjack.Logger{
		Filename:   *fpath,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	// Create writers
	writers := []io.Writer{lumberjackLogger}
	if config.Console {
		writers = append(writers, zerolog.ConsoleWriter{Out: os.Stdout})
	}

	// Set up multi-writer
	multiWriter := io.MultiWriter(writers...)

	// Set global log level
	zerolog.SetGlobalLevel(config.MinLogLevel)

	// Create new logger
	bl.logger = zerolog.New(multiWriter).
		With().Timestamp().Caller().Logger()

	// Flush buffered logs
	if bl.buffer.Len() > 0 {
		if _, err := multiWriter.Write(bl.buffer.Bytes()); err != nil {
			return fmt.Errorf("failed to flush buffer: %v", err)
		}
		bl.buffer.Reset()
	}

	bl.writer = multiWriter
	bl.isInitialized = true
	return nil
}

// Log level methods
func (bl *BufferedLogger) Debug(msg string) {
	bl.logger.Debug().Msg(msg)
}

func (bl *BufferedLogger) Info(msg string) {
	bl.logger.Info().Msg(msg)
}

func (bl *BufferedLogger) Warn(msg string) {
	bl.logger.Warn().Msg(msg)
}

func (bl *BufferedLogger) Error(msg string) {
	bl.logger.Error().Msg(msg)
}

func (bl *BufferedLogger) Fatal(msg string) {
	bl.logger.Fatal().Msg(msg)
}

// Structured logging methods
func (bl *BufferedLogger) DebugWithFields(msg string, fields map[string]interface{}) {
	event := bl.logger.Debug()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (bl *BufferedLogger) InfoWithFields(msg string, fields map[string]interface{}) {
	event := bl.logger.Info()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// Add similar methods for Warn, Error, Fatal with fields...

// IsInitialized returns whether the logger has been fully initialized
func (bl *BufferedLogger) IsInitialized() bool {
	return bl.isInitialized
}
