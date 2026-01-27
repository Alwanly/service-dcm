package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config holds the configuration for exponential backoff retry logic.
type Config struct {
	// MaxRetries is the maximum number of retry attempts.
	// Set to -1 for unlimited retries.
	MaxRetries int

	// InitialBackoff is the duration to wait before the first retry.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum duration to wait between retries.
	MaxBackoff time.Duration

	// Multiplier is the factor by which the backoff duration increases after each retry.
	// Default is 2.0 for exponential backoff.
	Multiplier float64

	// Jitter adds randomness to backoff duration to prevent thundering herd.
	Jitter bool
}

// Operation is a function that will be retried.
// It should return an error if the operation failed and should be retried.
// Return nil if the operation succeeded.
type Operation func(ctx context.Context) error

// WithExponentialBackoff executes the given operation with exponential backoff retry logic.
// It returns an error if all retries are exhausted or if the context is canceled.
func WithExponentialBackoff(ctx context.Context, cfg Config, op Operation) error {
	var attempt int
	var err error

	for {
		attempt++

		// Execute the operation
		err = op(ctx)
		if err == nil {
			return nil
		}

		// Check if we should retry
		if cfg.MaxRetries >= 0 && attempt > cfg.MaxRetries {
			return fmt.Errorf("operation failed after %d attempts: %w", attempt, err)
		}

		// Calculate backoff duration
		backoff := calculateBackoff(attempt, cfg)

		// Check if context is canceled before waiting
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation canceled after %d attempts: %w", attempt, ctx.Err())
		case <-time.After(backoff):
			// Continue to next retry attempt
		}
	}
}

// calculateBackoff calculates the backoff duration for the given retry attempt.
func calculateBackoff(retryNumber int, cfg Config) time.Duration {
	if retryNumber == 0 {
		return 0
	}

	// Calculate exponential backoff: initialBackoff * (multiplier ^ (retryNumber-1))
	// retryNumber==1 => initialBackoff
	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.Multiplier, float64(retryNumber-1))

	// Apply max backoff cap
	if backoff > float64(cfg.MaxBackoff) {
		backoff = float64(cfg.MaxBackoff)
	}

	duration := time.Duration(backoff)

	// Apply jitter if enabled (±25% randomness)
	if cfg.Jitter {
		jitterRange := float64(duration) * 0.25
		jitterAmount := (rand.Float64() * 2 * jitterRange) - jitterRange
		duration = time.Duration(float64(duration) + jitterAmount)

		// Ensure jitter doesn't exceed max backoff
		if duration > cfg.MaxBackoff {
			duration = cfg.MaxBackoff
		}
		if duration < 0 {
			duration = 0
		}
	}

	return duration
}
