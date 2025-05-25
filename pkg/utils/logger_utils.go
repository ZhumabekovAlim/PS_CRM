package utils

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger initializes the global zerolog logger with a structured format.
func InitLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs // Unix Milliseconds for time
	// zerolog.SetGlobalLevel(zerolog.InfoLevel) // Default level
	// if gin.IsDebugging() {
	// 	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// }

	// Use ConsoleWriter for more human-readable output during development
	// For production, you might want to remove this or use a JSON logger directly.
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()

	log.Info().Msg("Logger initialized")
}

// GinLogger is a middleware for Gin that logs requests using zerolog.
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		t_start := time.Now()

		// Process request
		c.Next()

		// Fill log fields
		var event *zerolog.Event
		latency := time.Since(t_start)
		statusCode := c.Writer.Status()

		if statusCode >= 500 {
			event = log.Error()
		} else if statusCode >= 400 {
			event = log.Warn()
		} else {
			event = log.Info()
		}

		// Log request details
		event.Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status_code", statusCode).
			Str("client_ip", c.ClientIP()).
			Str("latency", latency.String()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("Request processed")
	}
}

// LogError is a helper to log an error with zerolog.
func LogError(err error, message string) {
	if err != nil {
		log.Error().Err(err).Msg(message)
	}
}

// LogInfo is a helper to log an informational message.
func LogInfo(message string, fields ...map[string]interface{}) {
	event := log.Info()
	if len(fields) > 0 {
		for _, f := range fields {
			event = event.Fields(f)
		}
	}
	event.Msg(message)
}

// LogDebug is a helper to log a debug message.
func LogDebug(message string, fields ...map[string]interface{}) {
	event := log.Debug()
	if len(fields) > 0 {
		for _, f := range fields {
			event = event.Fields(f)
		}
	}
	event.Msg(message)
}

