package rest

import (
	"os"

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

//func SetLogger(level zapcore.Level) (*zap.Logger, error) {
//
//	atomicLevel = zap.NewAtomicLevelAt(level) // Set initial level to debug
//
//	config := zap.NewProductionConfig()
//	config.Encoding = "console" // plain text logs
//	config.EncoderConfig = zap.NewProductionEncoderConfig()
//	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // Optional: set time format
//	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // Optional: set level format
//	config.Level = atomicLevel
//
//	// Create the logger with the custom configuration
//	logger, err := config.Build()
//	if err != nil {
//		return nil, err
//	}
//	Log = logger.Sugar()
//	return logger, nil
//}

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
		//consoleEncoder := zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig())
		//logger = zap.New(
		//	zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zapcore.DebugLevel),
		//	zap.AddCaller(),
		//	zap.AddCallerSkip(1),
		//)
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

func configureLogger() (*zap.SugaredLogger, error) {
	// Create console encoder for debug level
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig())

	// Create a multi-level core with different encoders
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zapcore.DebugLevel),
		zapcore.NewCore(
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
		),
	)

	// Build the logger
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar := logger.Sugar()

	return sugar, nil
}
