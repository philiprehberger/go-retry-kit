package retrykit

import (
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker.
type CircuitState int

const (
	Closed   CircuitState = iota
	Open
	HalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// CircuitOpenError is returned when the circuit breaker is open.
type CircuitOpenError struct{}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker is open — request rejected"
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu                  sync.Mutex
	failureThreshold    int
	resetTimeout        time.Duration
	halfOpenMaxAttempts int
	onStateChange       func(from, to CircuitState)
	onCircuitOpen       func(failures int)

	state            CircuitState
	failures         int
	lastFailureTime  time.Time
	halfOpenAttempts int
}

// CircuitBreakerOption configures a CircuitBreaker.
type CircuitBreakerOption func(*CircuitBreaker)

// WithFailureThreshold sets the number of failures before opening the circuit.
func WithFailureThreshold(n int) CircuitBreakerOption {
	return func(cb *CircuitBreaker) { cb.failureThreshold = n }
}

// WithResetTimeout sets how long to wait before transitioning from open to half-open.
func WithResetTimeout(d time.Duration) CircuitBreakerOption {
	return func(cb *CircuitBreaker) { cb.resetTimeout = d }
}

// WithHalfOpenMaxAttempts sets the max attempts allowed in half-open state.
func WithHalfOpenMaxAttempts(n int) CircuitBreakerOption {
	return func(cb *CircuitBreaker) { cb.halfOpenMaxAttempts = n }
}

// WithOnStateChange sets a callback for state transitions.
func WithOnStateChange(fn func(from, to CircuitState)) CircuitBreakerOption {
	return func(cb *CircuitBreaker) { cb.onStateChange = fn }
}

// WithOnCircuitOpen sets a callback when the circuit opens.
func WithOnCircuitOpen(fn func(failures int)) CircuitBreakerOption {
	return func(cb *CircuitBreaker) { cb.onCircuitOpen = fn }
}

// NewCircuitBreaker creates a new CircuitBreaker with the given options.
func NewCircuitBreaker(opts ...CircuitBreakerOption) *CircuitBreaker {
	cb := &CircuitBreaker{
		failureThreshold:    5,
		resetTimeout:        30 * time.Second,
		halfOpenMaxAttempts: 1,
		state:               Closed,
	}
	for _, opt := range opts {
		opt(cb)
	}
	if cb.failureThreshold < 1 {
		cb.failureThreshold = 1
	}
	if cb.resetTimeout < 0 {
		cb.resetTimeout = 0
	}
	if cb.halfOpenMaxAttempts < 1 {
		cb.halfOpenMaxAttempts = 1
	}
	return cb
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset manually resets the circuit breaker to the Closed state with zero failures.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transition(Closed)
	cb.failures = 0
	cb.halfOpenAttempts = 0
}

func (cb *CircuitBreaker) transition(to CircuitState) {
	if cb.state != to {
		from := cb.state
		cb.state = to
		if cb.onStateChange != nil {
			cb.onStateChange(from, to)
		}
	}
}

// Call executes fn through the circuit breaker.
func Call[T any](cb *CircuitBreaker, fn func() (T, error)) (T, error) {
	cb.mu.Lock()
	var zero T

	if cb.state == Open {
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.transition(HalfOpen)
			cb.halfOpenAttempts = 0
		} else {
			cb.mu.Unlock()
			return zero, &CircuitOpenError{}
		}
	}

	if cb.state == HalfOpen && cb.halfOpenAttempts >= cb.halfOpenMaxAttempts {
		cb.mu.Unlock()
		return zero, &CircuitOpenError{}
	}

	if cb.state == HalfOpen {
		cb.halfOpenAttempts++
	}
	cb.mu.Unlock()

	result, err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()

		if cb.state == HalfOpen {
			cb.transition(Open)
			if cb.onCircuitOpen != nil {
				cb.onCircuitOpen(cb.failures)
			}
		} else if cb.failures >= cb.failureThreshold {
			cb.transition(Open)
			if cb.onCircuitOpen != nil {
				cb.onCircuitOpen(cb.failures)
			}
		}
		return zero, err
	}

	if cb.state == HalfOpen {
		cb.transition(Closed)
	}
	cb.failures = 0
	return result, nil
}

// String returns a human-readable description of the circuit breaker state.
func (cb *CircuitBreaker) String() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return fmt.Sprintf("CircuitBreaker{state: %s, failures: %d}", cb.state, cb.failures)
}
