package xcrawl3r

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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

	//nolint:contextcheck // invoked synchronously by RetryWithData under req.Context().
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

			if err := waitRetryAfter(req.Context(), response.Header); err != nil {
				return nil, err
			}

			return nil, &retryableStatusError{statusCode: response.StatusCode}
		}

		return response, nil
	}

	if response, err = hqgoretrier.RetryWithData(req.Context(), operation,
		hqgoretrier.WithMaxAttempts(rt.retries),
		hqgoretrier.WithRetryOn(isTransientError),
	); err != nil {
		err = fmt.Errorf("retry round trip: %w", err)
	}

	return
}

// isTransientError reports whether err is worth retrying: retryable HTTP
// statuses, unexpected EOFs, and network timeouts. Permanent failures (DNS
// resolution, TLS handshakes, connection refusals) retry nothing.
func isTransientError(err error) (transient bool) {
	var statusError *retryableStatusError
	if errors.As(err, &statusError) {
		return true
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error

	return errors.As(err, &netErr) && netErr.Timeout()
}

// waitRetryAfter sleeps for the delay requested by the server's Retry-After
// header (delta-seconds or HTTP-date), or until ctx is done.
func waitRetryAfter(ctx context.Context, header http.Header) (err error) {
	raw := header.Get("Retry-After")

	if raw == "" {
		return
	}

	var delay time.Duration

	if seconds, parseErr := strconv.Atoi(strings.TrimSpace(raw)); parseErr == nil {
		delay = time.Duration(seconds) * time.Second
	} else if retryAfter, parseErr := http.ParseTime(raw); parseErr == nil {
		delay = time.Until(retryAfter)
	}

	if delay <= 0 {
		return
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return
	case <-ctx.Done():
		return fmt.Errorf("retry-after wait interrupted: %w", ctx.Err())
	}
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
