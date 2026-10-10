package input

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBurpRequest(t *testing.T) {
	t.Parallel()

	rawRequest := "GET /admin HTTP/1.1\r\n" +
		"Host: target.example.com\r\n" +
		"User-Agent: Mozilla/5.0 Test\r\n" +
		"Cookie: session=abc123\r\n" +
		"Authorization: Bearer token\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n"

	directory := t.TempDir()

	path := filepath.Join(directory, "request.txt")

	if err := os.WriteFile(path, []byte(rawRequest), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	headers, err := ParseBurpRequest(path)
	if err != nil {
		t.Fatalf("ParseBurpRequest() error = %v", err)
	}

	want := []string{
		"User-Agent: Mozilla/5.0 Test",
		"Cookie: session=abc123",
		"Authorization: Bearer token",
	}

	if len(headers) != len(want) {
		t.Fatalf("ParseBurpRequest() returned %d headers, want %d: %v", len(headers), len(want), headers)
	}

	for i := range want {
		if headers[i] != want[i] {
			t.Errorf("headers[%d] = %q, want %q", i, headers[i], want[i])
		}
	}
}

func TestParseBurpRequestMissingFile(t *testing.T) {
	t.Parallel()

	if _, err := ParseBurpRequest(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Error("ParseBurpRequest() expected error for missing file")
	}
}

func TestParseBurpRequestInvalidRequestLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "request.txt")

	if err := os.WriteFile(path, []byte("garbage\n"), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	if _, err := ParseBurpRequest(path); err == nil {
		t.Error("ParseBurpRequest() expected error for invalid request line")
	}
}
