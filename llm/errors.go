package llm

import (
	"fmt"
	"net/http"
	"strings"
)

// ErrorType represents the type of LLM error
type ErrorType string

const (
	ErrorTypeUnknown        ErrorType = "unknown"
	ErrorTypeInvalidRequest ErrorType = "invalid_request"
	ErrorTypeAuthentication ErrorType = "authentication_error"
	ErrorTypePermission     ErrorType = "permission_error"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeRateLimit      ErrorType = "rate_limit_exceeded"
	// Backwards-compat alias used by some tests
	ErrorTypeRateLimited       ErrorType = ErrorTypeRateLimit
	ErrorTypeInsufficientQuota ErrorType = "insufficient_quota"
	ErrorTypeInvalidModel      ErrorType = "invalid_model"
	ErrorTypeContextLength     ErrorType = "context_length_exceeded"
	ErrorTypeContentFilter     ErrorType = "content_filter"
	ErrorTypeServerError       ErrorType = "server_error"
	ErrorTypeTimeout           ErrorType = "timeout"
	ErrorTypeConnectionError   ErrorType = "connection_error"
	ErrorTypeValidationError   ErrorType = "validation_error"
	ErrorTypeJSONParsingError  ErrorType = "json_parsing_error"
)

// LLMError represents an error from an LLM provider
type LLMError struct {
	Type       ErrorType         `json:"type"`
	Message    string            `json:"message"`
	Code       string            `json:"code,omitempty"`
	Provider   Provider          `json:"provider"`
	Model      string            `json:"model,omitempty"`
	HTTPStatus int               `json:"http_status,omitempty"`
	Retryable  bool              `json:"retryable"`
	RetryAfter int               `json:"retry_after,omitempty"` // Seconds to wait before retry
	Details    map[string]string `json:"details,omitempty"`
	Cause      error             `json:"-"`
}

// Error implements the error interface
func (e *LLMError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s [%s]: %s", e.Provider, e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Provider, e.Message)
}

// Unwrap returns the underlying error
func (e *LLMError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns true if the error is retryable
func (e *LLMError) IsRetryable() bool {
	return e.Retryable
}

// NewLLMError creates a new LLM error
func NewLLMError(provider Provider, errorType ErrorType, message string) *LLMError {
	return &LLMError{
		Type:      errorType,
		Message:   message,
		Provider:  provider,
		Retryable: isRetryableError(errorType),
	}
}

// NewLLMErrorWithCause creates a new LLM error with an underlying cause
func NewLLMErrorWithCause(provider Provider, errorType ErrorType, message string, cause error) *LLMError {
	err := NewLLMError(provider, errorType, message)
	err.Cause = cause
	return err
}

// isRetryableError determines if an error type is retryable
func isRetryableError(errorType ErrorType) bool {
	switch errorType {
	case ErrorTypeRateLimit, ErrorTypeServerError, ErrorTypeTimeout, ErrorTypeConnectionError:
		return true
	default:
		return false
	}
}

// ParseHTTPError parses HTTP status codes into appropriate LLM errors
func ParseHTTPError(provider Provider, statusCode int, body string) *LLMError {
	var errorType ErrorType
	var message string
	retryable := false

	switch statusCode {
	case http.StatusBadRequest:
		errorType = ErrorTypeInvalidRequest
		message = "Invalid request parameters"
	case http.StatusUnauthorized:
		errorType = ErrorTypeAuthentication
		message = "Invalid API key or authentication failed"
	case http.StatusForbidden:
		errorType = ErrorTypePermission
		message = "Permission denied"
	case http.StatusNotFound:
		errorType = ErrorTypeNotFound
		message = "Resource not found"
	case http.StatusTooManyRequests:
		errorType = ErrorTypeRateLimit
		message = "Rate limit exceeded"
		retryable = true
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		errorType = ErrorTypeServerError
		message = "Server error occurred"
		retryable = true
	default:
		errorType = ErrorTypeUnknown
		message = fmt.Sprintf("HTTP %d error", statusCode)
	}

	// Try to extract more specific error information from response body
	if body != "" {
		if specificError := extractSpecificError(provider, body); specificError != nil {
			specificError.HTTPStatus = statusCode
			return specificError
		}
		message = fmt.Sprintf("%s: %s", message, truncateBody(body, 200))
	}

	return &LLMError{
		Type:       errorType,
		Message:    message,
		Provider:   provider,
		HTTPStatus: statusCode,
		Retryable:  retryable,
	}
}

// extractSpecificError extracts provider-specific error information
func extractSpecificError(provider Provider, body string) *LLMError {
	lowerBody := strings.ToLower(body)

	// Common error patterns
	if strings.Contains(lowerBody, "rate limit") || strings.Contains(lowerBody, "too many requests") {
		return &LLMError{
			Type:      ErrorTypeRateLimit,
			Message:   "Rate limit exceeded",
			Provider:  provider,
			Retryable: true,
		}
	}

	if strings.Contains(lowerBody, "insufficient quota") || strings.Contains(lowerBody, "quota exceeded") {
		return &LLMError{
			Type:     ErrorTypeInsufficientQuota,
			Message:  "Insufficient quota or credits",
			Provider: provider,
		}
	}

	if strings.Contains(lowerBody, "context length") || strings.Contains(lowerBody, "token limit") {
		return &LLMError{
			Type:     ErrorTypeContextLength,
			Message:  "Context length exceeded",
			Provider: provider,
		}
	}

	if strings.Contains(lowerBody, "content filter") || strings.Contains(lowerBody, "safety") {
		return &LLMError{
			Type:     ErrorTypeContentFilter,
			Message:  "Content filtered by safety system",
			Provider: provider,
		}
	}

	if strings.Contains(lowerBody, "model") && (strings.Contains(lowerBody, "not found") || strings.Contains(lowerBody, "invalid")) {
		return &LLMError{
			Type:     ErrorTypeInvalidModel,
			Message:  "Invalid or unavailable model",
			Provider: provider,
		}
	}

	return nil
}

// truncateBody truncates response body for error messages
func truncateBody(body string, maxLength int) string {
	if len(body) <= maxLength {
		return body
	}
	return body[:maxLength] + "..."
}

// IsLLMError checks if an error is an LLMError
func IsLLMError(err error) (*LLMError, bool) {
	if llmErr, ok := err.(*LLMError); ok {
		return llmErr, true
	}
	return nil, false
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if llmErr, ok := IsLLMError(err); ok {
		// Compute retryability from the error type to be robust even if the
		// struct was constructed without using the constructor.
		return isRetryableError(llmErr.Type)
	}
	return false
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	if llmErr, ok := IsLLMError(err); ok {
		return llmErr.Type == ErrorTypeRateLimit
	}
	return false
}

// IsContextLengthError checks if an error is a context length error
func IsContextLengthError(err error) bool {
	if llmErr, ok := IsLLMError(err); ok {
		return llmErr.Type == ErrorTypeContextLength
	}
	return false
}

// IsAuthenticationError checks if an error is an authentication error
func IsAuthenticationError(err error) bool {
	if llmErr, ok := IsLLMError(err); ok {
		return llmErr.Type == ErrorTypeAuthentication
	}
	return false
}

// ValidationError represents a validation error for structured outputs
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
	Code    string      `json:"code,omitempty"`
}

// Error implements the error interface
func (v *ValidationError) Error() string {
	if v.Field != "" {
		return fmt.Sprintf("validation error on field '%s': %s", v.Field, v.Message)
	}
	return fmt.Sprintf("validation error: %s", v.Message)
}

// MultiValidationError represents multiple validation errors
type MultiValidationError struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (m *MultiValidationError) Error() string {
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}
	return fmt.Sprintf("%d validation errors occurred", len(m.Errors))
}

// Add adds a validation error
func (m *MultiValidationError) Add(field string, value interface{}, message string) {
	m.Errors = append(m.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors
func (m *MultiValidationError) HasErrors() bool {
	return len(m.Errors) > 0
}

// ErrorOrNil returns the error if there are validation errors, otherwise nil
func (m *MultiValidationError) ErrorOrNil() error {
	if m.HasErrors() {
		return m
	}
	return nil
}
