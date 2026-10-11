package input

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrEmptyFile          = errors.New("file is empty")
	ErrInvalidRequestLine = errors.New("invalid request line")
)

func ParseBurpRequest(path string) (headers []string, err error) {
	var file *os.File

	//nolint:gosec // path is the user-provided burp request file path.
	file, err = os.Open(path)
	if err != nil {
		err = fmt.Errorf("failed to open burp request file: %w", err)

		return
	}

	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	if !scanner.Scan() {
		err = fmt.Errorf("failed to read burp request file: %w", ErrEmptyFile)

		return
	}

	requestLine := scanner.Text()

	if len(strings.Fields(requestLine)) < 3 {
		err = fmt.Errorf("failed to parse burp request file: %w: %s", ErrInvalidRequestLine, requestLine)

		return
	}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			break
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch {
		case strings.EqualFold(key, "Host"),
			strings.EqualFold(key, "Content-Length"),
			strings.EqualFold(key, "Accept-Encoding"),
			strings.EqualFold(key, "Connection"):
			continue
		}

		headers = append(headers, fmt.Sprintf("%s: %s", key, value))
	}

	if err = scanner.Err(); err != nil {
		err = fmt.Errorf("failed to read burp request file: %w", err)

		return
	}

	return
}
