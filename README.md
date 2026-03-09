# go-retry-kit

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
```

## License

MIT
