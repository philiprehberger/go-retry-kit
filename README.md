# go-retry-kit

[![CI](https://github.com/philiprehberger/go-retry-kit/actions/workflows/ci.yml/badge.svg)](https://github.com/philiprehberger/go-retry-kit/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/philiprehberger/go-retry-kit.svg)](https://pkg.go.dev/github.com/philiprehberger/go-retry-kit) [![License](https://img.shields.io/github/license/philiprehberger/go-retry-kit)](LICENSE)

Retry with exponential backoff, circuit breaker, and context cancellation for Go

## Installation

```bash
go get github.com/philiprehberger/go-retry-kit
```

## Usage

### Basic Retry

```go
import "github.com/philiprehberger/go-retry-kit"

data, err := retrykit.Do(ctx, func(ctx context.Context) (string, error) {
    return fetchData(ctx)
})
```

### With Options

```go
data, err := retrykit.Do(ctx, fetchData,
    retrykit.WithMaxAttempts(5),
    retrykit.WithBackoff(retrykit.Exponential),
    retrykit.WithInitialDelay(time.Second),
    retrykit.WithMaxDelay(30*time.Second),
    retrykit.WithJitter(true),
    retrykit.WithRetryOn(func(err error) bool {
        return errors.Is(err, ErrTemporary)
    }),
    retrykit.WithOnSuccess(func(attempt int) {
        log.Printf("succeeded on attempt %d", attempt)
    }),
    retrykit.WithOnFailure(func(err error, attempts int) {
        log.Printf("all %d attempts failed: %v", attempts, err)
    }),
)
```

### Presets

```go
data, err := retrykit.Do(ctx, fetchData, retrykit.NetworkRequest()...)
data, err := retrykit.Do(ctx, queryDB, retrykit.DatabaseQuery()...)
data, err := retrykit.Do(ctx, criticalOp, retrykit.Aggressive()...)
```

### Circuit Breaker

```go
cb := retrykit.NewCircuitBreaker(
    retrykit.WithFailureThreshold(5),
    retrykit.WithResetTimeout(30*time.Second),
    retrykit.WithOnStateChange(func(from, to retrykit.CircuitState) {
        log.Printf("circuit: %s → %s", from, to)
    }),
)

result, err := retrykit.Call(cb, func() (string, error) {
    return fetchData()
})

// Manually reset the circuit breaker
cb.Reset()
```

## API

| Function / Method | Description |
|---|---|
| `Backoff` | Backoff strategy type (Exponential, Linear, Fixed) |
| `Options` | Struct configuring retry behavior |
| `Option` | Functional option for configuring retries |
| `DefaultOptions() Options` | Return sensible default retry options |
| `Do[T](ctx, fn, opts ...Option) (T, error)` | Execute fn with retry logic |
| `WithMaxAttempts(n int) Option` | Set maximum number of attempts |
| `WithBackoff(b Backoff) Option` | Set the backoff strategy |
| `WithInitialDelay(d time.Duration) Option` | Set initial delay between retries |
| `WithMaxDelay(d time.Duration) Option` | Set maximum delay between retries |
| `WithJitter(j bool) Option` | Enable or disable jitter |
| `WithRetryOn(fn func(error) bool) Option` | Filter which errors trigger retries |
| `WithOnRetry(fn func(error, int)) Option` | Set callback invoked before each retry |
| `WithOnSuccess(fn func(int)) Option` | Set callback invoked on success |
| `WithOnFailure(fn func(error, int)) Option` | Set callback invoked when all attempts fail |
| `Aggressive() []Option` | Preset: 5 attempts, fast exponential backoff |
| `Gentle() []Option` | Preset: 3 attempts, slow exponential backoff |
| `NetworkRequest() []Option` | Preset: tuned for network requests |
| `DatabaseQuery() []Option` | Preset: tuned for database queries |
| `RetryError` | Error returned when all attempts are exhausted |
| `(*RetryError) Error() string` | Format the retry error message |
| `(*RetryError) Unwrap() error` | Return the last underlying error |
| `CircuitState` | Circuit breaker state (Closed, Open, HalfOpen) |
| `(CircuitState) String() string` | Human-readable state name |
| `CircuitBreaker` | Circuit breaker implementation |
| `CircuitBreakerOption` | Functional option for configuring circuit breaker |
| `NewCircuitBreaker(opts ...CircuitBreakerOption) *CircuitBreaker` | Create a new circuit breaker |
| `(*CircuitBreaker) State() CircuitState` | Return the current circuit state |
| `(*CircuitBreaker) Reset()` | Manually reset to Closed state |
| `(*CircuitBreaker) String() string` | Human-readable breaker description |
| `Call[T](cb *CircuitBreaker, fn func() (T, error)) (T, error)` | Execute fn through the circuit breaker |
| `WithFailureThreshold(n int) CircuitBreakerOption` | Set failures before opening the circuit |
| `WithResetTimeout(d time.Duration) CircuitBreakerOption` | Set open-to-half-open timeout |
| `WithHalfOpenMaxAttempts(n int) CircuitBreakerOption` | Set max attempts in half-open state |
| `WithOnStateChange(fn) CircuitBreakerOption` | Set callback for state transitions |
| `WithOnCircuitOpen(fn func(int)) CircuitBreakerOption` | Set callback when circuit opens |
| `CircuitOpenError` | Error returned when circuit is open |
| `(*CircuitOpenError) Error() string` | Format the circuit open error |

## Development

```bash
go test ./...
go vet ./...
```

## License

MIT
