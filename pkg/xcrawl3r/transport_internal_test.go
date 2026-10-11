package xcrawl3r

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newStatusServer(t *testing.T, status int, requests *atomic.Int32) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)

		w.WriteHeader(status)
	}))

	t.Cleanup(server.Close)

	return server
}

func roundTrip(t *testing.T, rt http.RoundTripper, rawURL string) *http.Response {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}

	response, err := rt.RoundTrip(request)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	return response
}

func TestRoundTripperRetriesOnRetryableStatus(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rt := &roundTripper{base: http.DefaultTransport, retries: 3}

	response := roundTrip(t, rt, server.URL)
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		t.Errorf("RoundTrip() status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	if got := requests.Load(); got != 2 {
		t.Errorf("requests = %d, want 2", got)
	}
}

func TestRoundTripperReturnsLastResponseWhenRetriesExhausted(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := newStatusServer(t, http.StatusServiceUnavailable, &requests)
	rt := &roundTripper{base: http.DefaultTransport, retries: 3}

	response := roundTrip(t, rt, server.URL)
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("RoundTrip() status = %d, want %d", response.StatusCode, http.StatusServiceUnavailable)
	}

	if got := requests.Load(); got != 3 {
		t.Errorf("requests = %d, want 3", got)
	}
}

func TestRoundTripperRetriesDisabled(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := newStatusServer(t, http.StatusServiceUnavailable, &requests)
	rt := &roundTripper{base: http.DefaultTransport, retries: 1}

	response := roundTrip(t, rt, server.URL)
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("RoundTrip() status = %d, want %d", response.StatusCode, http.StatusServiceUnavailable)
	}

	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
}

func TestRoundTripperRateLimitSpacesRequests(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := newStatusServer(t, http.StatusOK, &requests)
	rt := &roundTripper{base: http.DefaultTransport, retries: 1, requestsPerMinute: 600}

	start := time.Now()

	for range 2 {
		response := roundTrip(t, rt, server.URL)

		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}

	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Errorf("two requests at 600 rpm took %v, want >= 50ms", elapsed)
	}
}
