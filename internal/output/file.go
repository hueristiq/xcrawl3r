package output

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
)

type Writer struct {
	format format
}

func (w *Writer) SetFormatToJSONL() {
	w.format = formatJSONL
}

func (w *Writer) CreateFile(path string) (file *os.File, err error) {
	if path == "" {
		err = ErrNoFilePathSpecified

		return
	}

	extension := filepath.Ext(path)

	switch w.format {
	case formatTXT:
		if extension != ".txt" {
			path += ".txt"
		}
	case formatJSONL:
		if extension != ".json" {
			path += ".json"
		}
	}

	directory := filepath.Dir(path)

	if directory != "" {
		if _, err = os.Stat(directory); os.IsNotExist(err) {
			err = os.MkdirAll(directory, 0o750)
			if err != nil {
				return
			}
		}
	}

	file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}

	return
}

func (w *Writer) Write(writer io.Writer, result xcrawl3r.Result) (err error) {
	switch w.format {
	case formatTXT:
		err = w.writeTXT(writer, result)
	case formatJSONL:
		err = w.writeJSON(writer, result)
	}

	return
}

func (w *Writer) writeTXT(writer io.Writer, result xcrawl3r.Result) (err error) {
	bw := bufio.NewWriter(writer)

	fmt.Fprintln(bw, result.Value)

	if err = bw.Flush(); err != nil {
		return
	}

	return
}

func (w *Writer) writeJSON(writer io.Writer, result xcrawl3r.Result) (err error) {
	data := resultForJSONL{
		URL: result.Value,
	}

	var dataJSONBytes []byte

	dataJSONBytes, err = json.Marshal(data)
	if err != nil {
		return
	}

	dataJSONString := string(dataJSONBytes)

	bw := bufio.NewWriter(writer)

	fmt.Fprintln(bw, dataJSONString)

	if err = bw.Flush(); err != nil {
		return
	}

	return
}

type format string

type resultForJSONL struct {
	URL string `json:"url"`
}

const (
	formatJSONL format = "JSON"
	formatTXT   format = "TXT"
)

var ErrNoFilePathSpecified = errors.New("no file path specified")

func NewWriter() (writter *Writer) {
	writter = &Writer{
		format: formatTXT,
	}

	return
}
