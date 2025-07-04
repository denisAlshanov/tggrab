package utils

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type contextKey string

type Fields = logrus.Fields

const (
	CorrelationIDKey contextKey = "correlation_id"
	RequestIDKey     contextKey = "request_id"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()

	// Set JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Set log level from environment variable
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logger.Warnf("Invalid log level %s, defaulting to info", logLevel)
		level = logrus.InfoLevel
	}

	logger.SetLevel(level)
	logger.SetOutput(os.Stdout)
}

func GetLogger() *logrus.Logger {
	return logger
}

func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

func GenerateCorrelationID() string {
	return uuid.New().String()
}

func GenerateRequestID() string {
	return "req_" + uuid.New().String()
}

func LoggerFromContext(ctx context.Context) *logrus.Entry {
	entry := logger.WithFields(logrus.Fields{})

	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		entry = entry.WithField("correlation_id", correlationID)
	}

	if requestID := GetRequestID(ctx); requestID != "" {
		entry = entry.WithField("request_id", requestID)
	}

	return entry
}

// Helper functions for common logging patterns
func LogInfo(ctx context.Context, message string, fields ...logrus.Fields) {
	entry := LoggerFromContext(ctx)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Info(message)
}

func LogError(ctx context.Context, message string, err error, fields ...logrus.Fields) {
	entry := LoggerFromContext(ctx).WithError(err)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Error(message)
}

func LogWarn(ctx context.Context, message string, fields ...logrus.Fields) {
	entry := LoggerFromContext(ctx)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Warn(message)
}

func LogDebug(ctx context.Context, message string, fields ...logrus.Fields) {
	entry := LoggerFromContext(ctx)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Debug(message)
}
