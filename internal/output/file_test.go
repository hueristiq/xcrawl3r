//nolint:testpackage // white-box tests for unexported helpers.
package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
)

const (
	testDomain        = "example.com"
	testFileNameTXT   = "out.txt"
	testFileNameJSON  = "out.json"
	testFileNameJSONL = "out.jsonl"
	testURL           = "https://example.com/path"
	testSource        = "https://example.com/"
	testVia           = "html:href"
)

func TestCreateFileExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		jsonl bool
		given string
		want  string
	}{
		{name: "txt known", given: testFileNameTXT, want: testFileNameTXT},
		{name: "txt uppercase", given: "OUT.TXT", want: "OUT.TXT"},
		{name: "txt without extension", given: "out", want: testFileNameTXT},
		{name: "txt unknown extension", given: "out.log", want: "out.log.txt"},
		{name: "txt jsonl suffix", given: testFileNameJSONL, want: "out.jsonl.txt"},
		{name: "json known", jsonl: true, given: testFileNameJSON, want: testFileNameJSON},
		{name: "jsonl known", jsonl: true, given: testFileNameJSONL, want: testFileNameJSONL},
		{name: "jsonl uppercase", jsonl: true, given: "OUT.JSONL", want: "OUT.JSONL"},
		{name: "jsonl without extension", jsonl: true, given: "out", want: testFileNameJSON},
		{name: "jsonl unknown extension", jsonl: true, given: testFileNameTXT, want: "out.txt.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			givenPath := filepath.Join(directory, tt.given)
			wantPath := filepath.Join(directory, tt.want)

			writer := NewWriter()
			if tt.jsonl {
				writer.SetFormatToJSONL()
			}

			file, err := writer.CreateFile(givenPath)
			if err != nil {
				t.Fatalf("CreateFile() error = %v", err)
			}

			name := file.Name()

			if err := file.Close(); err != nil {
				t.Fatalf("file.Close() error = %v", err)
			}

			if name != wantPath {
				t.Fatalf("file name = %q, want %q", name, wantPath)
			}
		})
	}
}

func TestCreateFileEmptyPath(t *testing.T) {
	t.Parallel()

	writer := NewWriter()

	file, err := writer.CreateFile("")
	if file != nil {
		_ = file.Close()
	}

	if !errors.Is(err, ErrNoFilePathSpecified) {
		t.Fatalf("CreateFile(\"\") error = %v, want %v", err, ErrNoFilePathSpecified)
	}
}

func TestCreateFileMode(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("file modes are not enforced on Windows")
	}

	directory := t.TempDir()
	path := filepath.Join(directory, testFileNameTXT)

	writer := NewWriter()

	file, err := writer.CreateFile(path)
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("file.Close() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}

	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("mode = %o, want 600", mode)
	}
}

func TestWriterTXT(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	writer := NewWriter()

	result := xcrawl3r.Result{
		Type:   xcrawl3r.ResultURL,
		Value:  testURL,
		Source: testSource,
		Via:    testVia,
	}

	if err := writer.Write(&buffer, testDomain, result); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := writer.Flush(&buffer); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if got, want := buffer.String(), "https://example.com/path\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWriterTXTWithTechnique(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	writer := NewWriter()

	result := xcrawl3r.Result{
		Type:  xcrawl3r.ResultURL,
		Value: testURL,
		Via:   testVia,
	}

	if err := writer.WriteWithTechnique(&buffer, testDomain, result); err != nil {
		t.Fatalf("WriteWithTechnique() error = %v", err)
	}

	if err := writer.Flush(&buffer); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if got, want := buffer.String(), "[html:href] https://example.com/path\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWriterJSONL(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	writer := NewWriter()
	writer.SetFormatToJSONL()

	result := xcrawl3r.Result{
		Type:   xcrawl3r.ResultURL,
		Value:  testURL,
		Source: testSource,
		Via:    testVia,
	}

	if err := writer.Write(&buffer, testDomain, result); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := writer.Flush(&buffer); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	var decoded resultForJSONL
	if err := json.Unmarshal(bytes.TrimSpace(buffer.Bytes()), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	want := resultForJSONL{
		Domain: testDomain,
		URL:    testURL,
		Source: testSource,
		Via:    testVia,
	}

	if decoded != want {
		t.Fatalf("decoded = %#v, want %#v", decoded, want)
	}
}

func TestWriterBuffersUntilFlush(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	writer := NewWriter()

	result := xcrawl3r.Result{
		Type:  xcrawl3r.ResultURL,
		Value: testURL,
	}

	if err := writer.Write(&buffer, testDomain, result); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if buffer.Len() != 0 {
		t.Fatalf("buffer length before flush = %d, want 0", buffer.Len())
	}

	if err := writer.Flush(&buffer); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if !strings.Contains(buffer.String(), testURL) {
		t.Fatalf("output = %q, want to contain the URL", buffer.String())
	}
}

func TestReleaseFlushesAndEvicts(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	writer := NewWriter()
	result := xcrawl3r.Result{
		Type:  xcrawl3r.ResultURL,
		Value: testURL,
	}

	if err := writer.Write(&buffer, testDomain, result); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if len(writer.writers) != 1 {
		t.Fatalf("writers = %d, want 1", len(writer.writers))
	}

	if err := writer.Release(&buffer); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	if got, want := buffer.String(), testURL+"\n"; got != want {
		t.Fatalf("released output = %q, want %q", got, want)
	}

	if len(writer.writers) != 0 {
		t.Fatalf("writers = %d after Release, want 0", len(writer.writers))
	}

	if err := writer.Release(&buffer); err != nil {
		t.Fatalf("second Release() error = %v", err)
	}
}
