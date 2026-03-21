# Changelog

## 0.3.2

- Consolidate README badges onto single line

## 0.3.1

- Add badges and Development section to README

## 0.3.0

- Add `CircuitBreaker.Reset()` to manually reset to Closed state
- Add input validation for `NewCircuitBreaker` (clamp threshold, timeout, max attempts)

## 0.2.1

- Fix exponential backoff overflow at high attempt counts
- Add tests for `RetryError.Unwrap()` chain
- Add jitter bounds validation test

## 0.2.0

- Add `WithOnSuccess` and `WithOnFailure` option setters
- Add input validation (clamp `MaxAttempts`, `InitialDelay`, `MaxDelay` to valid ranges)
- Add comprehensive test suite for retry and circuit breaker

## 0.1.0

- Initial release
