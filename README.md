# go-retry-kit

[![CI](https://github.com/philiprehberger/go-retry-kit/actions/workflows/ci.yml/badge.svg)](https://github.com/philiprehberger/go-retry-kit/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/philiprehberger/go-retry-kit.svg)](https://pkg.go.dev/github.com/philiprehberger/go-retry-kit)
[![License](https://img.shields.io/github/license/philiprehberger/go-retry-kit)](LICENSE)

Retry with exponential backoff, circuit breaker, and context cancellation for Go.

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

## Development

```bash
go test ./...
go vet ./...
```

## License

MIT
