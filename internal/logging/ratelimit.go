package logging

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RateLimiter struct {
	backoffDuration time.Duration
	maxRetries      int
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		backoffDuration: 1 * time.Second,
		maxRetries:      3,
	}
}

func (r *RateLimiter) ExecuteWithBackoff(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if it's a quota exceeded error
		if !r.isQuotaExceededError(err) {
			return err
		}

		if attempt < r.maxRetries {
			// Exponential backoff with jitter
			backoffTime := r.calculateBackoff(attempt)
			time.Sleep(backoffTime)
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (r *RateLimiter) isQuotaExceededError(err error) bool {
	if st, ok := status.FromError(err); ok {
		if st.Code() == codes.ResourceExhausted {
			return true
		}
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "resource exhausted")
}

func (r *RateLimiter) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: 1s, 2s, 4s, 8s...
	backoff := r.backoffDuration * time.Duration(math.Pow(2, float64(attempt)))

	// Add jitter to avoid thundering herd
	jitter := time.Duration(float64(backoff) * 0.1)

	return backoff + jitter
}
