package llm

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// Retrier handles retry logic for LLM operations
type Retrier struct {
	config RetryConfig
	rand   *rand.Rand
}

// NewRetrier creates a new retrier with the given configuration
func NewRetrier(config RetryConfig) *Retrier {
	return &Retrier{
		config: config,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RetryOperation represents an operation that can be retried
type RetryOperation[T any] func(ctx context.Context, attempt int) (T, error)

// Execute executes an operation with retry logic
func Execute[T any](r *Retrier, ctx context.Context, operation RetryOperation[T]) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		// Execute the operation
		result, err := operation(ctx, attempt)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if !r.shouldRetry(err, attempt) {
			// If we've reached max retries, return an exhaustion error to signal retry policy completed
			if attempt >= r.config.MaxRetries {
				return zero, fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxRetries+1, err)
			}
			return zero, err
		}

		// Calculate delay
		delay := r.calculateDelay(attempt, err)

		// Wait before retry
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return zero, fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxRetries+1, lastErr)
}

// ExecuteSimple executes a simple operation without generics
func (r *Retrier) ExecuteSimple(ctx context.Context, operation func(context.Context, int) error) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := operation(ctx, attempt)
		if err == nil {
			return nil
		}

		lastErr = err

		if !r.shouldRetry(err, attempt) {
			return err
		}

		delay := r.calculateDelay(attempt, err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxRetries+1, lastErr)
}

// shouldRetry determines if an operation should be retried
func (r *Retrier) shouldRetry(err error, attempt int) bool {
	// Don't retry if we've exceeded max attempts
	if attempt >= r.config.MaxRetries {
		return false
	}

	// Check if it's an LLM error and retryable
	if llmErr, ok := IsLLMError(err); ok {
		return llmErr.IsRetryable()
	}

	// Check against configured retryable error types/messages
	errStr := err.Error()
	for _, retryableErr := range r.config.RetryableErrors {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(retryableErr)) {
			return true
		}
	}

	return false
}

// calculateDelay calculates the delay before the next retry
func (r *Retrier) calculateDelay(attempt int, err error) time.Duration {
	// Check if the error specifies a retry-after value
	if llmErr, ok := IsLLMError(err); ok && llmErr.RetryAfter > 0 {
		return time.Duration(llmErr.RetryAfter) * time.Second
	}

	// Exponential backoff with jitter
	base := float64(r.config.InitialDelay)
	delay := base * math.Pow(r.config.BackoffFactor, float64(attempt))

	// Add jitter (±25%)
	jitter := 0.25 * delay * (r.rand.Float64()*2 - 1)
	delay += jitter

	// Ensure we don't exceed max delay
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Ensure minimum delay
	if delay < float64(r.config.InitialDelay) {
		delay = float64(r.config.InitialDelay)
	}

	return time.Duration(delay)
}

// RetryStats tracks retry statistics
type RetryStats struct {
	TotalAttempts int            `json:"total_attempts"`
	Successful    int            `json:"successful"`
	Failed        int            `json:"failed"`
	TotalDelay    time.Duration  `json:"total_delay"`
	LastError     string         `json:"last_error,omitempty"`
	ErrorTypes    map[string]int `json:"error_types,omitempty"`
}

// StatTrackingRetrier wraps a retrier with statistics tracking
type StatTrackingRetrier struct {
	*Retrier
	stats RetryStats
}

// NewStatTrackingRetrier creates a new retry with stats tracking
func NewStatTrackingRetrier(config RetryConfig) *StatTrackingRetrier {
	return &StatTrackingRetrier{
		Retrier: NewRetrier(config),
		stats: RetryStats{
			ErrorTypes: make(map[string]int),
		},
	}
}

// Execute executes an operation with retry logic and stats tracking
func ExecuteWithStats[T any](s *StatTrackingRetrier, ctx context.Context, operation RetryOperation[T]) (T, error) {
	start := time.Now()
	var zero T
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		s.stats.TotalAttempts++

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			s.stats.Failed++
			return zero, ctx.Err()
		default:
		}

		// Execute the operation
		result, err := operation(ctx, attempt)
		if err == nil {
			s.stats.Successful++
			s.stats.TotalDelay = time.Since(start)
			return result, nil
		}

		lastErr = err
		s.stats.LastError = err.Error()

		// Track error type
		if llmErr, ok := IsLLMError(err); ok {
			s.stats.ErrorTypes[string(llmErr.Type)]++
		} else {
			s.stats.ErrorTypes["unknown"]++
		}

		// Check if we should retry
		if !s.shouldRetry(err, attempt) {
			s.stats.Failed++
			s.stats.TotalDelay = time.Since(start)
			if attempt >= s.config.MaxRetries {
				return zero, fmt.Errorf("operation failed after %d attempts: %w", s.config.MaxRetries+1, err)
			}
			return zero, err
		}

		// Calculate delay
		delay := s.calculateDelay(attempt, err)

		// Wait before retry
		select {
		case <-ctx.Done():
			s.stats.Failed++
			s.stats.TotalDelay = time.Since(start)
			return zero, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	s.stats.Failed++
	s.stats.TotalDelay = time.Since(start)
	return zero, fmt.Errorf("operation failed after %d attempts: %w", s.config.MaxRetries+1, lastErr)
}

// GetStats returns the current retry statistics
func (s *StatTrackingRetrier) GetStats() RetryStats {
	return s.stats
}

// ResetStats resets the retry statistics
func (s *StatTrackingRetrier) ResetStats() {
	s.stats = RetryStats{
		ErrorTypes: make(map[string]int),
	}
}
