# Changelog

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
