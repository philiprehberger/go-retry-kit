// Package retrykit provides retry with backoff and circuit breaker for Go.
package retrykit

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// Backoff defines the backoff strategy.
type Backoff int

const (
	Exponential Backoff = iota
	Linear
	Fixed
)

// Options configures retry behavior.
type Options struct {
	MaxAttempts  int
	Backoff      Backoff
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Jitter       bool
	RetryOn      func(error) bool
	OnRetry      func(err error, attempt int)
	OnSuccess    func(attempt int)
	OnFailure    func(err error, attempts int)
}

// DefaultOptions returns sensible default retry options.
func DefaultOptions() Options {
	return Options{
		MaxAttempts:  3,
		Backoff:      Exponential,
		InitialDelay: time.Second,
		MaxDelay:     30 * time.Second,
		Jitter:       true,
	}
}

// Option is a functional option for configuring retry behavior.
type Option func(*Options)

// WithMaxAttempts sets the maximum number of attempts.
func WithMaxAttempts(n int) Option {
	return func(o *Options) { o.MaxAttempts = n }
}

// WithBackoff sets the backoff strategy.
func WithBackoff(b Backoff) Option {
	return func(o *Options) { o.Backoff = b }
}

// WithInitialDelay sets the initial delay between retries.
func WithInitialDelay(d time.Duration) Option {
	return func(o *Options) { o.InitialDelay = d }
}

// WithMaxDelay sets the maximum delay between retries.
func WithMaxDelay(d time.Duration) Option {
	return func(o *Options) { o.MaxDelay = d }
}

// WithJitter enables or disables jitter.
func WithJitter(j bool) Option {
	return func(o *Options) { o.Jitter = j }
}

// WithRetryOn sets a function to determine if an error should be retried.
func WithRetryOn(fn func(error) bool) Option {
	return func(o *Options) { o.RetryOn = fn }
}

// WithOnRetry sets a callback invoked before each retry.
func WithOnRetry(fn func(error, int)) Option {
	return func(o *Options) { o.OnRetry = fn }
}

// WithOnSuccess sets a callback invoked when the operation succeeds.
func WithOnSuccess(fn func(attempt int)) Option {
	return func(o *Options) { o.OnSuccess = fn }
}

// WithOnFailure sets a callback invoked when all attempts have been exhausted.
func WithOnFailure(fn func(err error, attempts int)) Option {
	return func(o *Options) { o.OnFailure = fn }
}

// RetryError is returned when all attempts have been exhausted.
type RetryError struct {
	Attempts int
	Last     error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("all %d attempts failed: %v", e.Attempts, e.Last)
}

func (e *RetryError) Unwrap() error {
	return e.Last
}

func calculateDelay(attempt int, opts Options) time.Duration {
	var delay time.Duration
	switch opts.Backoff {
	case Exponential:
		delay = opts.InitialDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	case Linear:
		delay = opts.InitialDelay * time.Duration(attempt)
	default:
		delay = opts.InitialDelay
	}

	if delay > opts.MaxDelay {
		delay = opts.MaxDelay
	}

	if opts.Jitter {
		delay = time.Duration(float64(delay) * (0.5 + rand.Float64()*0.5))
	}

	return delay
}

// Do executes fn with retry logic according to the provided options.
func Do[T any](ctx context.Context, fn func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	cfg := DefaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.InitialDelay < 0 {
		cfg.InitialDelay = 0
	}
	if cfg.MaxDelay < cfg.InitialDelay {
		cfg.MaxDelay = cfg.InitialDelay
	}

	var lastErr error
	var zero T

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		result, err := fn(ctx)
		if err == nil {
			if cfg.OnSuccess != nil {
				cfg.OnSuccess(attempt)
			}
			return result, nil
		}

		lastErr = err

		if cfg.RetryOn != nil && !cfg.RetryOn(err) {
			return zero, err
		}

		if attempt < cfg.MaxAttempts {
			if cfg.OnRetry != nil {
				cfg.OnRetry(err, attempt)
			}

			delay := calculateDelay(attempt, cfg)
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	if cfg.OnFailure != nil {
		cfg.OnFailure(lastErr, cfg.MaxAttempts)
	}

	return zero, &RetryError{Attempts: cfg.MaxAttempts, Last: lastErr}
}

// Presets

// Aggressive returns options for aggressive retry (5 attempts, fast backoff).
func Aggressive() []Option {
	return []Option{
		WithMaxAttempts(5),
		WithBackoff(Exponential),
		WithInitialDelay(500 * time.Millisecond),
		WithMaxDelay(5 * time.Second),
		WithJitter(true),
	}
}

// Gentle returns options for gentle retry (3 attempts, slow backoff).
func Gentle() []Option {
	return []Option{
		WithMaxAttempts(3),
		WithBackoff(Exponential),
		WithInitialDelay(2 * time.Second),
		WithMaxDelay(30 * time.Second),
		WithJitter(true),
	}
}

// NetworkRequest returns options suited for network requests.
func NetworkRequest() []Option {
	return []Option{
		WithMaxAttempts(3),
		WithBackoff(Exponential),
		WithInitialDelay(time.Second),
		WithMaxDelay(10 * time.Second),
		WithJitter(true),
	}
}

// DatabaseQuery returns options suited for database queries.
func DatabaseQuery() []Option {
	return []Option{
		WithMaxAttempts(3),
		WithBackoff(Linear),
		WithInitialDelay(500 * time.Millisecond),
		WithMaxDelay(5 * time.Second),
		WithJitter(false),
	}
}
