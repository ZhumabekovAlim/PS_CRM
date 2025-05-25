package utils

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// Standardized APIError response
type APIError struct {
	StatusCode int    `json:"-"` // HTTP status code, not included in JSON response body for error itself
	Code       string `json:"code,omitempty"` // Application-specific error code
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
}

// NewAPIError creates a new APIError instance
func NewAPIError(statusCode int, code string, message string, details string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Details:    details,
	}
}

// RespondWithError sends a standardized JSON error response
func RespondWithError(c *gin.Context, err *APIError) {
	c.JSON(err.StatusCode, gin.H{"error": err})
	c.Abort() // Abort further processing if it's a middleware or critical error
}

// Common Error Constants (examples)
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeInternalServerError = "INTERNAL_SERVER_ERROR"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
)

// Validation functions

// IsEmpty checks if a string is empty after trimming whitespace.
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsValidEmail checks if a string is a valid email format.
var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

func IsValidEmail(email string) bool {
	return emailRegex.MatchString(strings.ToLower(email))
}

// IsValidPasswordLength checks if password meets minimum length requirement.
func IsValidPasswordLength(password string, minLength int) bool {
	return len(password) >= minLength
}

// ValidatePayloadWithRules is a generic helper to validate a payload based on a map of rules.
// Rules map: fieldName -> validationFunction (func(value interface{}) (bool, string))
// This is a conceptual example; a more robust solution would use struct tags and a validation library like validator/v10.
func ValidatePayloadWithRules(payload interface{}, rules map[string]func(interface{}) (bool, string)) (bool, map[string]string) {
	// This function would require reflection to iterate over payload fields and apply rules.
	// For simplicity in this context, direct validation in handlers is often clearer without a complex generic validator.
	// Or, use a library like "github.com/go-playground/validator/v10"
	// Example usage with validator/v10:
	// validate := validator.New()
	// err := validate.Struct(payload)
	// if err != nil { ... handle validation errors ... }
	panic("ValidatePayloadWithRules is conceptual and not fully implemented here. Use a library like go-playground/validator.")
}

// Helper to return a standard validation error
func RespondValidationFailed(c *gin.Context, details string) {
	RespondWithError(c, NewAPIError(http.StatusBadRequest, ErrCodeValidationFailed, "Input validation failed", details))
}

