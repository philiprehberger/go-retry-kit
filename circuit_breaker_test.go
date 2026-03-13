package retrykit

import (
	"errors"
	"testing"
	"time"
)

var errCB = errors.New("circuit breaker test error")

func TestCircuitBreakerClosedSuccess(t *testing.T) {
	cb := NewCircuitBreaker()
	result, err := Call(cb, func() (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected 'ok', got %q", result)
	}
	if cb.State() != Closed {
		t.Fatalf("expected Closed state, got %v", cb.State())
	}
}

func TestCircuitBreakerOpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(WithFailureThreshold(3))

	for i := 0; i < 3; i++ {
		Call(cb, func() (int, error) {
			return 0, errCB
		})
	}

	if cb.State() != Open {
		t.Fatalf("expected Open state after %d failures, got %v", 3, cb.State())
	}

	// Subsequent calls should be rejected
	_, err := Call(cb, func() (int, error) {
		t.Fatal("function should not be called when circuit is open")
		return 0, nil
	})
	var openErr *CircuitOpenError
	if !errors.As(err, &openErr) {
		t.Fatalf("expected CircuitOpenError, got %v", err)
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(
		WithFailureThreshold(2),
		WithResetTimeout(50*time.Millisecond),
		WithHalfOpenMaxAttempts(1),
	)

	// Trip the breaker
	for i := 0; i < 2; i++ {
		Call(cb, func() (int, error) {
			return 0, errCB
		})
	}
	if cb.State() != Open {
		t.Fatalf("expected Open, got %v", cb.State())
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Next call should transition to HalfOpen and execute
	result, err := Call(cb, func() (string, error) {
		return "recovered", nil
	})
	if err != nil {
		t.Fatalf("unexpected error in half-open: %v", err)
	}
	if result != "recovered" {
		t.Fatalf("expected 'recovered', got %q", result)
	}
	if cb.State() != Closed {
		t.Fatalf("expected Closed after half-open success, got %v", cb.State())
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(
		WithFailureThreshold(2),
		WithResetTimeout(50*time.Millisecond),
		WithHalfOpenMaxAttempts(1),
	)

	// Trip the breaker
	for i := 0; i < 2; i++ {
		Call(cb, func() (int, error) {
			return 0, errCB
		})
	}

	time.Sleep(60 * time.Millisecond)

	// Fail in half-open — should go back to Open
	Call(cb, func() (int, error) {
		return 0, errCB
	})
	if cb.State() != Open {
		t.Fatalf("expected Open after half-open failure, got %v", cb.State())
	}
}

func TestCircuitBreakerOnStateChange(t *testing.T) {
	var transitions []struct{ from, to CircuitState }
	cb := NewCircuitBreaker(
		WithFailureThreshold(2),
		WithResetTimeout(50*time.Millisecond),
		WithOnStateChange(func(from, to CircuitState) {
			transitions = append(transitions, struct{ from, to CircuitState }{from, to})
		}),
	)

	// Trip: Closed -> Open
	for i := 0; i < 2; i++ {
		Call(cb, func() (int, error) {
			return 0, errCB
		})
	}

	if len(transitions) != 1 || transitions[0].from != Closed || transitions[0].to != Open {
		t.Fatalf("expected [Closed->Open], got %v", transitions)
	}
}

func TestCircuitBreakerOnCircuitOpen(t *testing.T) {
	var openFailures int
	cb := NewCircuitBreaker(
		WithFailureThreshold(2),
		WithOnCircuitOpen(func(failures int) {
			openFailures = failures
		}),
	)

	for i := 0; i < 2; i++ {
		Call(cb, func() (int, error) {
			return 0, errCB
		})
	}

	if openFailures != 2 {
		t.Fatalf("expected 2 failures in callback, got %d", openFailures)
	}
}

func TestCircuitBreakerSuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker(WithFailureThreshold(3))

	// 2 failures then a success
	Call(cb, func() (int, error) { return 0, errCB })
	Call(cb, func() (int, error) { return 0, errCB })
	Call(cb, func() (int, error) { return 1, nil })

	// Should be back to 0 failures — 2 more failures shouldn't trip
	Call(cb, func() (int, error) { return 0, errCB })
	Call(cb, func() (int, error) { return 0, errCB })

	if cb.State() != Closed {
		t.Fatal("expected Closed — success should have reset failure count")
	}
}

func TestCircuitBreakerString(t *testing.T) {
	cb := NewCircuitBreaker()
	s := cb.String()
	if s == "" {
		t.Fatal("expected non-empty string")
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker(WithFailureThreshold(2))

	// Trip the breaker
	for i := 0; i < 2; i++ {
		Call(cb, func() (int, error) { return 0, errCB })
	}
	if cb.State() != Open {
		t.Fatalf("expected Open, got %v", cb.State())
	}

	// Reset manually
	cb.Reset()
	if cb.State() != Closed {
		t.Fatalf("expected Closed after Reset, got %v", cb.State())
	}

	// Should work normally again
	result, err := Call(cb, func() (string, error) { return "ok", nil })
	if err != nil {
		t.Fatalf("unexpected error after Reset: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected 'ok', got %q", result)
	}
}

func TestCircuitBreakerResetFromHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(
		WithFailureThreshold(2),
		WithResetTimeout(50*time.Millisecond),
	)
	for i := 0; i < 2; i++ {
		Call(cb, func() (int, error) { return 0, errCB })
	}
	time.Sleep(60 * time.Millisecond)

	// Trigger half-open by calling
	Call(cb, func() (int, error) { return 0, errCB })

	cb.Reset()
	if cb.State() != Closed {
		t.Fatalf("expected Closed after Reset from re-opened, got %v", cb.State())
	}
}

func TestCircuitBreakerInputValidation(t *testing.T) {
	cb := NewCircuitBreaker(
		WithFailureThreshold(0),
		WithResetTimeout(-time.Second),
		WithHalfOpenMaxAttempts(-1),
	)
	// Should clamp to safe values
	if cb.failureThreshold < 1 {
		t.Fatalf("expected failureThreshold >= 1, got %d", cb.failureThreshold)
	}
	if cb.resetTimeout < 0 {
		t.Fatalf("expected resetTimeout >= 0, got %v", cb.resetTimeout)
	}
	if cb.halfOpenMaxAttempts < 1 {
		t.Fatalf("expected halfOpenMaxAttempts >= 1, got %d", cb.halfOpenMaxAttempts)
	}
}

func TestCircuitStateString(t *testing.T) {
	if Closed.String() != "closed" {
		t.Errorf("expected 'closed', got %q", Closed.String())
	}
	if Open.String() != "open" {
		t.Errorf("expected 'open', got %q", Open.String())
	}
	if HalfOpen.String() != "half_open" {
		t.Errorf("expected 'half_open', got %q", HalfOpen.String())
	}
}
