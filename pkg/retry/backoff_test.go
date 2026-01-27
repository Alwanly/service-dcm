package retry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestWithExponentialBackoff_SuccessFirstAttempt(t *testing.T) {
	cfg := Config{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		Jitter:         false,
	}

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		return nil
	}

	err := WithExponentialBackoff(context.Background(), cfg, op)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestWithExponentialBackoff_SuccessAfterRetries(t *testing.T) {
	cfg := Config{
		MaxRetries:     5,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false,
	}

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	start := time.Now()
	err := WithExponentialBackoff(context.Background(), cfg, op)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	// First attempt: 0ms
	// Second attempt: 10ms backoff
	// Third attempt: 20ms backoff
	// Total: ~30ms minimum
	if elapsed < 30*time.Millisecond {
		t.Errorf("expected at least 30ms elapsed, got %v", elapsed)
	}
}

func TestWithExponentialBackoff_ExhaustsRetries(t *testing.T) {
	cfg := Config{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false,
	}

	attempts := 0
	expectedErr := errors.New("permanent failure")
	op := func(ctx context.Context) error {
		attempts++
		return expectedErr
	}

	err := WithExponentialBackoff(context.Background(), cfg, op)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if attempts != 4 {
		t.Errorf("expected 4 attempts (1 initial + 3 retries), got %d", attempts)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error to be %v, got %v", expectedErr, err)
	}
}

func TestWithExponentialBackoff_ContextCancellation(t *testing.T) {
	cfg := Config{
		MaxRetries:     10,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		Jitter:         false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		return errors.New("always fails")
	}

	err := WithExponentialBackoff(ctx, cfg, op)
	if err == nil {
		t.Error("expected error due to context cancellation, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}

	// Should have attempted at least once, but not all 10 retries
	if attempts == 0 {
		t.Error("expected at least one attempt")
	}
	if attempts > 5 {
		t.Errorf("expected fewer attempts due to context timeout, got %d", attempts)
	}
}

func TestCalculateBackoff_ExponentialGrowth(t *testing.T) {
	cfg := Config{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
		Jitter:         false,
	}

	tests := []struct {
		retryNumber int
		want        time.Duration
	}{
		{0, 0},                 // First attempt (no backoff)
		{1, 1 * time.Second},   // 1 * 2^1 = 1s
		{2, 2 * time.Second},   // 1 * 2^2 = 2s
		{3, 4 * time.Second},   // 1 * 2^3 = 4s
		{4, 8 * time.Second},   // 1 * 2^4 = 8s
		{5, 16 * time.Second},  // 1 * 2^5 = 16s
		{6, 30 * time.Second},  // 1 * 2^6 = 32s -> capped at 30s
		{7, 30 * time.Second},  // Capped at max
		{10, 30 * time.Second}, // Capped at max
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("retry_%d", tt.retryNumber), func(t *testing.T) {
			got := calculateBackoff(tt.retryNumber, cfg)
			if got != tt.want {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.retryNumber, got, tt.want)
			}
		})
	}
}

func TestCalculateBackoff_WithJitter(t *testing.T) {
	cfg := Config{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     10 * time.Second,
		Multiplier:     2.0,
		Jitter:         true,
	}

	retryNumber := 3
	expectedBase := 4 * time.Second

	// Run multiple times to ensure jitter produces different values
	results := make(map[time.Duration]bool)
	for i := 0; i < 20; i++ {
		backoff := calculateBackoff(retryNumber, cfg)

		// Jitter should be within ±25% of base
		minExpected := time.Duration(float64(expectedBase) * 0.75)
		maxExpected := time.Duration(float64(expectedBase) * 1.25)

		if backoff < minExpected || backoff > maxExpected {
			t.Errorf("backoff %v outside expected range [%v, %v]", backoff, minExpected, maxExpected)
		}

		// Should not exceed max backoff
		if backoff > cfg.MaxBackoff {
			t.Errorf("backoff %v exceeds max backoff %v", backoff, cfg.MaxBackoff)
		}

		results[backoff] = true
	}

	// With jitter enabled, we should see variety in results (not all the same)
	if len(results) < 5 {
		t.Error("jitter not producing enough variation in backoff durations")
	}
}

func TestWithExponentialBackoff_UnlimitedRetries(t *testing.T) {
	cfg := Config{
		MaxRetries:     -1, // Unlimited
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	op := func(ctx context.Context) error {
		attempts++
		if attempts == 10 {
			return nil
		}
		return errors.New("keep retrying")
	}

	err := WithExponentialBackoff(ctx, cfg, op)
	if err != nil {
		t.Errorf("expected success after 10 attempts, got error: %v", err)
	}
	if attempts != 10 {
		t.Errorf("expected 10 attempts, got %d", attempts)
	}
}
