package retrykit

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestDoSuccess(t *testing.T) {
	result, err := Do(context.Background(), func(ctx context.Context) (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected 'ok', got %q", result)
	}
}

func TestDoRetryThenSuccess(t *testing.T) {
	attempt := 0
	result, err := Do(context.Background(), func(ctx context.Context) (int, error) {
		attempt++
		if attempt < 3 {
			return 0, errTest
		}
		return 42, nil
	}, WithMaxAttempts(5), WithBackoff(Fixed), WithInitialDelay(time.Millisecond), WithJitter(false))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
	if attempt != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempt)
	}
}

func TestDoAllRetriesExhausted(t *testing.T) {
	_, err := Do(context.Background(), func(ctx context.Context) (int, error) {
		return 0, errTest
	}, WithMaxAttempts(3), WithBackoff(Fixed), WithInitialDelay(time.Millisecond), WithJitter(false))

	var retryErr *RetryError
	if !errors.As(err, &retryErr) {
		t.Fatalf("expected RetryError, got %v", err)
	}
	if retryErr.Attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", retryErr.Attempts)
	}
	if !errors.Is(retryErr.Last, errTest) {
		t.Fatalf("expected wrapped errTest, got %v", retryErr.Last)
	}
}

func TestDoContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempt := 0

	_, err := Do(ctx, func(ctx context.Context) (int, error) {
		attempt++
		if attempt == 1 {
			cancel()
		}
		return 0, errTest
	}, WithMaxAttempts(10), WithBackoff(Fixed), WithInitialDelay(time.Millisecond), WithJitter(false))

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestDoWithRetryOn(t *testing.T) {
	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("non-retryable")

	attempt := 0
	_, err := Do(context.Background(), func(ctx context.Context) (int, error) {
		attempt++
		if attempt == 1 {
			return 0, nonRetryableErr
		}
		return 42, nil
	}, WithRetryOn(func(err error) bool {
		return errors.Is(err, retryableErr)
	}), WithMaxAttempts(5), WithInitialDelay(time.Millisecond))

	if !errors.Is(err, nonRetryableErr) {
		t.Fatalf("expected nonRetryableErr, got %v", err)
	}
	if attempt != 1 {
		t.Fatalf("expected 1 attempt (no retry), got %d", attempt)
	}
}

func TestDoOnRetryCallback(t *testing.T) {
	var retryAttempts []int
	_, err := Do(context.Background(), func(ctx context.Context) (int, error) {
		return 0, errTest
	},
		WithMaxAttempts(3),
		WithBackoff(Fixed),
		WithInitialDelay(time.Millisecond),
		WithJitter(false),
		WithOnRetry(func(err error, attempt int) {
			retryAttempts = append(retryAttempts, attempt)
		}),
	)

	if err == nil {
		t.Fatal("expected error")
	}
	// OnRetry should be called for attempts 1 and 2 (not the last attempt)
	if len(retryAttempts) != 2 || retryAttempts[0] != 1 || retryAttempts[1] != 2 {
		t.Fatalf("expected OnRetry calls [1,2], got %v", retryAttempts)
	}
}

func TestDoOnSuccessCallback(t *testing.T) {
	var successAttempt int
	attempt := 0
	_, err := Do(context.Background(), func(ctx context.Context) (int, error) {
		attempt++
		if attempt < 2 {
			return 0, errTest
		}
		return 1, nil
	},
		WithMaxAttempts(3),
		WithBackoff(Fixed),
		WithInitialDelay(time.Millisecond),
		WithJitter(false),
		WithOnSuccess(func(a int) {
			successAttempt = a
		}),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if successAttempt != 2 {
		t.Fatalf("expected OnSuccess at attempt 2, got %d", successAttempt)
	}
}

func TestDoOnFailureCallback(t *testing.T) {
	var failErr error
	var failAttempts int
	_, err := Do(context.Background(), func(ctx context.Context) (int, error) {
		return 0, errTest
	},
		WithMaxAttempts(2),
		WithBackoff(Fixed),
		WithInitialDelay(time.Millisecond),
		WithJitter(false),
		WithOnFailure(func(err error, attempts int) {
			failErr = err
			failAttempts = attempts
		}),
	)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(failErr, errTest) {
		t.Fatalf("expected errTest in callback, got %v", failErr)
	}
	if failAttempts != 2 {
		t.Fatalf("expected 2 attempts in callback, got %d", failAttempts)
	}
}

func TestDoBackoffStrategies(t *testing.T) {
	strategies := []Backoff{Exponential, Linear, Fixed}
	for _, b := range strategies {
		_, err := Do(context.Background(), func(ctx context.Context) (int, error) {
			return 0, errTest
		}, WithMaxAttempts(2), WithBackoff(b), WithInitialDelay(time.Millisecond), WithJitter(false))
		if err == nil {
			t.Fatalf("expected error for backoff %v", b)
		}
	}
}

func TestDoInputValidation(t *testing.T) {
	// MaxAttempts < 1 should be clamped to 1
	attempt := 0
	_, err := Do(context.Background(), func(ctx context.Context) (int, error) {
		attempt++
		return 0, errTest
	}, WithMaxAttempts(-1), WithInitialDelay(time.Millisecond))

	var retryErr *RetryError
	if !errors.As(err, &retryErr) {
		t.Fatalf("expected RetryError, got %v", err)
	}
	if attempt != 1 {
		t.Fatalf("expected 1 attempt with clamped MaxAttempts, got %d", attempt)
	}
}

func TestPresets(t *testing.T) {
	presets := []struct {
		name string
		opts []Option
	}{
		{"Aggressive", Aggressive()},
		{"Gentle", Gentle()},
		{"NetworkRequest", NetworkRequest()},
		{"DatabaseQuery", DatabaseQuery()},
	}
	for _, p := range presets {
		cfg := DefaultOptions()
		for _, opt := range p.opts {
			opt(&cfg)
		}
		if cfg.MaxAttempts < 1 {
			t.Errorf("%s: invalid MaxAttempts %d", p.name, cfg.MaxAttempts)
		}
	}
}

func TestRetryErrorUnwrap(t *testing.T) {
	_, err := Do(context.Background(), func(ctx context.Context) (int, error) {
		return 0, errTest
	}, WithMaxAttempts(1), WithInitialDelay(time.Millisecond))

	if !errors.Is(err, errTest) {
		t.Fatal("expected errors.Is to find errTest through Unwrap chain")
	}

	var retryErr *RetryError
	if !errors.As(err, &retryErr) {
		t.Fatal("expected errors.As to find RetryError")
	}
	if retryErr.Last != errTest {
		t.Fatal("expected Last to be errTest")
	}
}

func TestJitterBounds(t *testing.T) {
	opts := Options{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Backoff:      Fixed,
		Jitter:       true,
	}

	for i := 0; i < 100; i++ {
		d := calculateDelay(1, opts)
		// Jitter formula: delay * (0.5 + rand*0.5), so range is [50ms, 100ms)
		if d < 50*time.Millisecond || d >= 100*time.Millisecond {
			t.Fatalf("jitter delay %v out of expected range [50ms, 100ms)", d)
		}
	}
}

func TestExponentialBackoffLargeAttemptCapped(t *testing.T) {
	opts := Options{
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Backoff:      Exponential,
		Jitter:       false,
	}

	d := calculateDelay(50, opts) // 2^49 ms would be huge
	if d != time.Second {
		t.Fatalf("expected delay capped at 1s for large attempt, got %v", d)
	}
}

func TestCalculateDelay(t *testing.T) {
	opts := Options{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Jitter:       false,
	}

	// Exponential
	opts.Backoff = Exponential
	d := calculateDelay(1, opts)
	if d != 100*time.Millisecond {
		t.Errorf("expected 100ms for exp attempt 1, got %v", d)
	}
	d = calculateDelay(2, opts)
	if d != 200*time.Millisecond {
		t.Errorf("expected 200ms for exp attempt 2, got %v", d)
	}

	// Linear
	opts.Backoff = Linear
	d = calculateDelay(3, opts)
	if d != 300*time.Millisecond {
		t.Errorf("expected 300ms for linear attempt 3, got %v", d)
	}

	// Fixed
	opts.Backoff = Fixed
	d = calculateDelay(5, opts)
	if d != 100*time.Millisecond {
		t.Errorf("expected 100ms for fixed attempt 5, got %v", d)
	}

	// MaxDelay cap
	opts.Backoff = Exponential
	d = calculateDelay(10, opts)
	if d != 1*time.Second {
		t.Errorf("expected delay capped at 1s, got %v", d)
	}
}
