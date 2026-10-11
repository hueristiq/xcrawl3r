package output_test

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestWriterCreateFileExtensions(t *testing.T) {
	t.Parallel()

	const (
		txtExt  = "out.txt"
		csvExt  = "out.csv"
		jsonExt = "out.json"
	)

	toJSONL := func(writer *output.Writer) {
		writer.SetFormatToJSONL()
	}

	tests := []struct {
		name  string
		setup func(*output.Writer)
		path  string
		want  string
	}{
		{"txt appends extension when missing", nil, "out", txtExt},
		{"txt keeps .txt", nil, txtExt, txtExt},
		{"txt keeps foreign extension", nil, csvExt, csvExt},
		{"jsonl appends extension when missing", toJSONL, "out", jsonExt},
		{"jsonl keeps .json", toJSONL, jsonExt, jsonExt},
		{"jsonl keeps .jsonl", toJSONL, "out.jsonl", "out.jsonl"},
		{"jsonl keeps foreign extension", toJSONL, csvExt, csvExt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()

			path := filepath.Join(directory, tt.path)

			writer := output.NewWriter()

			if tt.setup != nil {
				tt.setup(writer)
			}

			file, err := writer.CreateFile(path)
			if err != nil {
				t.Fatalf("CreateFile() error = %v", err)
			}

			_ = file.Close()

			want := filepath.Join(directory, tt.want)

			if _, err := os.Stat(want); err != nil {
				t.Errorf("expected output file %s to exist: %v", want, err)
			}
		})
	}
}
