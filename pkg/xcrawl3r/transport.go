package xcrawl3r

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	hqgolimiter "github.com/hueristiq/hq-lib-limiter-go"
	hqgoretrier "github.com/hueristiq/hq-lib-retrier-go"
)

type retryableStatusError struct {
	statusCode int
}

func (e *retryableStatusError) Error() string {
	return fmt.Sprintf("retryable status code: %d", e.statusCode)
}

type roundTripper struct {
	base              http.RoundTripper
	retries           int
	requestsPerMinute int

	mu       sync.Mutex
	limiters map[string]*hqgolimiter.Limiter
}

func (rt *roundTripper) RoundTrip(req *http.Request) (response *http.Response, err error) {
	if rt.requestsPerMinute > 0 {
		if err = rt.limiter(req.URL.Hostname()).Wait(req.Context()); err != nil {
			err = fmt.Errorf("rate limit wait: %w", err)

			return
		}
	}

	if rt.retries <= 1 {
		if response, err = rt.base.RoundTrip(req); err != nil {
			err = fmt.Errorf("base round trip: %w", err)
		}

		return
	}

	attempts := 0

	operation := func() (*http.Response, error) {
		attempts++

		response, err := rt.base.RoundTrip(req)
		if err != nil {
			return nil, fmt.Errorf("base round trip: %w", err)
		}

		if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
			if attempts >= rt.retries {
				return response, nil
			}

			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()

			return nil, &retryableStatusError{statusCode: response.StatusCode}
		}

		return response, nil
	}

	if response, err = hqgoretrier.RetryWithData(req.Context(), operation,
		hqgoretrier.WithMaxAttempts(rt.retries),
	); err != nil {
		err = fmt.Errorf("retry round trip: %w", err)
	}

	return
}

func (rt *roundTripper) limiter(host string) (limiter *hqgolimiter.Limiter) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.limiters == nil {
		rt.limiters = make(map[string]*hqgolimiter.Limiter)
	}

	limiter, ok := rt.limiters[host]
	if !ok {
		limiter = hqgolimiter.New(
			hqgolimiter.WithRequestsPerMinute(rt.requestsPerMinute),
		)

		rt.limiters[host] = limiter
	}

	return
}
