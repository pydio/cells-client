package rest

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	atomicLevel zap.AtomicLevel
	Log         *zap.SugaredLogger
)

func currentLogLevel() zapcore.Level {
	return atomicLevel.Level()
}

func IsDebugEnabled() bool { return currentLogLevel() <= zapcore.DebugLevel }

func IsInfoEnabled() bool { return currentLogLevel() <= zapcore.InfoLevel }

func SetLogger(level zapcore.Level) (logger *zap.Logger) {

	atomicLevel = zap.NewAtomicLevelAt(level) // Set initial level to debug

	if level == zapcore.DebugLevel {
		config := zap.NewProductionConfig()
		config.Encoding = "console" // plain text logs
		config.EncoderConfig = zap.NewProductionEncoderConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // Optional: set time format
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // Optional: set level format
		config.Level = atomicLevel

		// Create the logger with the custom configuration
		logger, _ = config.Build()
	} else {
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
				MessageKey:     "message",
				LevelKey:       "level",
				TimeKey:        "",
				NameKey:        "",
				CallerKey:      "",
				StacktraceKey:  "",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    customInfoLevelEncoder,
				EncodeTime:     nil,
				EncodeDuration: nil,
				EncodeCaller:   nil,
			}),
			zapcore.Lock(os.Stdout),
			zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= zapcore.InfoLevel
			}),
		)
		logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	Log = logger.Sugar()
	return
}

// Custom level encoder for INFO level
func customInfoLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	if level == zapcore.InfoLevel {
		return
	}
	if level == zapcore.FatalLevel {
		level = zapcore.ErrorLevel
	}
	enc.AppendString(level.CapitalString())
}

// S3 SDK logging:

// Map of log category strings to their corresponding aws.ClientLogMode constants
var logCategoryMap = map[string]aws.ClientLogMode{
	"signing":                aws.LogSigning,
	"retries":                aws.LogRetries,
	"request":                aws.LogRequest,
	"request_with_body":      aws.LogRequestWithBody,
	"response":               aws.LogResponse,
	"response_with_body":     aws.LogResponseWithBody,
	"deprecated_usage":       aws.LogDeprecatedUsage,
	"request_event_message":  aws.LogRequestEventMessage,
	"response_event_message": aws.LogResponseEventMessage,
}

func getLogMode(input string) aws.ClientLogMode {
	logMode := aws.ClientLogMode(0)
	categories := strings.Split(input, "|")
	for _, category := range categories {
		trimmedCategory := strings.TrimSpace(category)
		if trimmedCategory == "" {
			continue // Skip empty categories
		}
		if mode, exists := logCategoryMap[trimmedCategory]; exists {
			logMode |= mode
		} else {
			fmt.Printf("Unknown log category: %s\n", trimmedCategory)
		}
	}
	return logMode
}
