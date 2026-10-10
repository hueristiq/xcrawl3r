package output_test

import (
	"bytes"
	"testing"

	"github.com/hueristiq/xcrawl3r/internal/output"
	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
)

const exampleURL = "https://example.com/"

func TestWriterWriteTXT(t *testing.T) {
	t.Parallel()

	writer := output.NewWriter()

	var buffer bytes.Buffer

	err := writer.Write(&buffer, xcrawl3r.Result{
		Type:   xcrawl3r.ResultURL,
		Value:  exampleURL,
		Source: "https://example.com/robots.txt",
		Via:    "robots",
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if want := exampleURL + "\n"; buffer.String() != want {
		t.Errorf("Write() = %q, want %q", buffer.String(), want)
	}
}

func TestWriterWriteJSONL(t *testing.T) {
	t.Parallel()

	writer := output.NewWriter()
	writer.SetFormatToJSONL()

	var buffer bytes.Buffer

	err := writer.Write(&buffer, xcrawl3r.Result{
		Type:   xcrawl3r.ResultURL,
		Value:  exampleURL,
		Source: "https://example.com/robots.txt",
		Via:    "robots",
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	want := `{"url":"` + exampleURL + `","source":"https://example.com/robots.txt","via":"robots"}` + "\n"

	if buffer.String() != want {
		t.Errorf("Write() = %q, want %q", buffer.String(), want)
	}
}

func TestWriterWriteJSONLOmitsEmptyProvenance(t *testing.T) {
	t.Parallel()

	writer := output.NewWriter()
	writer.SetFormatToJSONL()

	var buffer bytes.Buffer

	err := writer.Write(&buffer, xcrawl3r.Result{
		Type:  xcrawl3r.ResultURL,
		Value: exampleURL,
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if want := `{"url":"` + exampleURL + `"}` + "\n"; buffer.String() != want {
		t.Errorf("Write() = %q, want %q", buffer.String(), want)
	}
}
