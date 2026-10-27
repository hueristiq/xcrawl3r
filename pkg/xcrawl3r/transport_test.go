//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

const testRetryAfterHeader = "Retry-After"

func TestRoundTripperRetriesServerErrors(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 2 {
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	rt := &roundTripper{
		base:    &http.Transport{},
		retries: 2,
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}

	response, err := rt.RoundTrip(request)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
}

func TestRoundTripperDoesNotRetryClientErrors(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		writer.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	rt := &roundTripper{
		base:    &http.Transport{},
		retries: 5,
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}

	response, err := rt.RoundTrip(request)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}

	if got := attempts.Load(); got != 1 {
		t.Fatalf("attempts = %d, want 1", got)
	}
}

func TestRoundTripperSingleAttempt(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	rt := &roundTripper{
		base:    &http.Transport{},
		retries: 1,
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}

	response, err := rt.RoundTrip(request)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if got := attempts.Load(); got != 1 {
		t.Fatalf("attempts = %d, want 1", got)
	}
}

func TestRoundTripperRetryAfterHonorsContext(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set(testRetryAfterHeader, "60")
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	rt := &roundTripper{
		base:    &http.Transport{},
		retries: 3,
	}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}

	response, err := rt.RoundTrip(request)
	if response != nil {
		defer func() {
			_ = response.Body.Close()
		}()
	}

	if err == nil {
		t.Fatal("RoundTrip() error = nil, want context deadline error")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RoundTrip() error = %v, want context deadline exceeded", err)
	}
}

func TestRoundTripperLimiterPerHost(t *testing.T) {
	t.Parallel()

	rt := &roundTripper{requestsPerMinute: 60000}

	first := rt.limiter(testDomain)
	second := rt.limiter(testDomain)
	other := rt.limiter("other.com")

	if first != second {
		t.Fatal("limiter() returned different limiters for the same host")
	}

	if first == other {
		t.Fatal("limiter() returned the same limiter for different hosts")
	}
}

func TestRoundTripperRateLimitedRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	rt := &roundTripper{
		base:              &http.Transport{},
		retries:           1,
		requestsPerMinute: 60000,
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}

	response, err := rt.RoundTrip(request)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

func TestWaitRetryAfter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  http.Header
		cancel  bool
		wantErr bool
	}{
		{name: "no header", header: http.Header{}, wantErr: false},
		{name: "zero", header: http.Header{testRetryAfterHeader: []string{"0"}}, wantErr: false},
		{name: "invalid header", header: http.Header{testRetryAfterHeader: []string{"later"}}, wantErr: false},
		{name: "canceled", header: http.Header{testRetryAfterHeader: []string{"60"}}, cancel: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			if tt.cancel {
				cancel()
			} else {
				defer cancel()
			}

			err := waitRetryAfter(ctx, tt.header)
			if (err != nil) != tt.wantErr {
				t.Fatalf("waitRetryAfter() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.cancel && !errors.Is(err, context.Canceled) {
				t.Fatalf("waitRetryAfter() error = %v, want context canceled", err)
			}
		})
	}
}
