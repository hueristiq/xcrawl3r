//nolint:testpackage // white-box tests for unexported helpers.
package configuration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"
)

func fullConfigYAML(includeSettle bool) string {
	settle := ""

	if includeSettle {
		settle = "  settle: 1\n"
	}

	return fmt.Sprintf(`version: %s
request:
  delay: 0
  random_delay: 0
  timeout: 10
  retries: 0
  requests_per_minute: 0
  random_agent: false
  insecure_skip_verify: false
proxies: []
optimization:
  depth: 0
  concurrency: 5
  parallelism: 5
  max_pages: 0
  crawl_duration: ""
  ignore_query: false
  shape_dedup: false
render:
  enabled: false
  timeout: 10
%s`, Version, settle)
}

func TestCreateOrUpdateFirstRun(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")

	if err := CreateOrUpdate(path); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is under t.TempDir().
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var cfg Configuration
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if cfg.Version != Version {
		t.Fatalf("version = %q, want %q", cfg.Version, Version)
	}

	if cfg.Request.Timeout != 10 {
		t.Fatalf("request.timeout = %d, want 10", cfg.Request.Timeout)
	}

	if cfg.Request.Retries != 3 {
		t.Fatalf("request.retries = %d, want 3", cfg.Request.Retries)
	}

	if cfg.Request.InsecureSkipVerify {
		t.Fatal("request.insecure_skip_verify = true, want false")
	}

	if cfg.Optimization.Depth != 2 {
		t.Fatalf("optimization.depth = %d, want 2", cfg.Optimization.Depth)
	}

	if cfg.Render.Settle != 1 {
		t.Fatalf("render.settle = %d, want 1", cfg.Render.Settle)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("os.ReadDir() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("directory entries = %d, want 1 (no temporary file left)", len(entries))
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("os.Stat() error = %v", err)
		}

		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("mode = %o, want 600", mode)
		}
	}
}

func TestCreateOrUpdateEmptyFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")

	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if err := CreateOrUpdate(path); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is under t.TempDir().
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var cfg Configuration
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if cfg.Request.Timeout != 10 {
		t.Fatalf("request.timeout = %d, want 10", cfg.Request.Timeout)
	}

	if cfg.Request.InsecureSkipVerify {
		t.Fatal("request.insecure_skip_verify = true, want false")
	}
}

func TestCreateOrUpdatePreservesZeroValues(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")

	if err := os.WriteFile(path, []byte(fullConfigYAML(false)), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if err := CreateOrUpdate(path); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is under t.TempDir().
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var cfg Configuration
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if cfg.Request.Retries != 0 {
		t.Fatalf("request.retries = %d, want 0 preserved", cfg.Request.Retries)
	}

	if cfg.Optimization.Depth != 0 {
		t.Fatalf("optimization.depth = %d, want 0 preserved", cfg.Optimization.Depth)
	}

	if cfg.Optimization.MaxPages != 0 {
		t.Fatalf("optimization.max_pages = %d, want 0 preserved", cfg.Optimization.MaxPages)
	}

	if cfg.Optimization.IgnoreQuery {
		t.Fatal("optimization.ignore_query = true, want false preserved")
	}

	if cfg.Render.Settle != 1 {
		t.Fatalf("render.settle = %d, want 1 filled", cfg.Render.Settle)
	}
}

func TestReadPresence(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")

	body := `request:
  timeout: 0
optimization:
  depth: 0
render: {}
`

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, present, err := readWithPresence(path)
	if err != nil {
		t.Fatalf("readWithPresence() error = %v", err)
	}

	if cfg.Request.Timeout != 0 || cfg.Optimization.Depth != 0 {
		t.Fatalf("cfg = %#v, want zero timeout and depth", cfg)
	}

	wantPresent := []string{"request", "request.timeout", "optimization", "optimization.depth", "render"}
	for _, key := range wantPresent {
		if !present[key] {
			t.Fatalf("present[%q] = false, want true", key)
		}
	}

	wantAbsent := []string{"request.retries", "render.settle", "optimization.max_pages"}
	for _, key := range wantAbsent {
		if present[key] {
			t.Fatalf("present[%q] = true, want false", key)
		}
	}
}

func TestWriteIsAtomic(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")

	cfg := DefaultConfiguration

	if err := cfg.Write(path); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := cfg.Write(path); err != nil {
		t.Fatalf("Write() second error = %v", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("os.ReadDir() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("directory entries = %d, want 1 (no temporary file left)", len(entries))
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("os.Stat() error = %v", err)
		}

		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("mode = %o, want 600", mode)
		}
	}
}

func TestCreateOrUpdateVersionMismatch(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")

	body := `version: v0.0.0-old
request:
  retries: 0
optimization:
  depth: 0
`

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if err := CreateOrUpdate(path); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is under t.TempDir().
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var cfg Configuration
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if cfg.Version != Version {
		t.Fatalf("version = %q, want %q", cfg.Version, Version)
	}

	if cfg.Request.Retries != 0 {
		t.Fatalf("request.retries = %d, want 0 preserved", cfg.Request.Retries)
	}

	if cfg.Optimization.Depth != 0 {
		t.Fatalf("optimization.depth = %d, want 0 preserved", cfg.Optimization.Depth)
	}

	if cfg.Request.Timeout != 10 {
		t.Fatalf("request.timeout = %d, want default 10", cfg.Request.Timeout)
	}
}

func TestDefaultConfiguration(t *testing.T) {
	t.Parallel()

	if DefaultConfiguration.Request.InsecureSkipVerify {
		t.Fatal("DefaultConfiguration.Request.InsecureSkipVerify = true, want false")
	}

	if DefaultConfiguration.Optimization.Depth != 2 {
		t.Fatalf("DefaultConfiguration.Optimization.Depth = %d, want 2", DefaultConfiguration.Optimization.Depth)
	}

	if DefaultConfiguration.Render.Settle != 1 {
		t.Fatalf("DefaultConfiguration.Render.Settle = %d, want 1", DefaultConfiguration.Render.Settle)
	}
}

func TestReadRejectsInvalidYAML(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(path, []byte("request: [unclosed"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if _, err := Read(path); err == nil {
		t.Fatal("Read() = nil error, want parse error")
	}
}

func TestHasMissingDefaults(t *testing.T) {
	t.Parallel()

	present := make(map[string]bool, len(defaultKeys))

	for _, key := range defaultKeys {
		present[key] = true
	}

	if hasMissingDefaults(present) {
		t.Fatal("hasMissingDefaults() = true for complete configuration")
	}

	delete(present, defaultKeys[0])

	if !hasMissingDefaults(present) {
		t.Fatal("hasMissingDefaults() = false for configuration missing a key")
	}
}

func TestWriteRemovesTemporaryFileOnRenameFailure(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	target := filepath.Join(directory, "target")

	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatalf("os.Mkdir() error = %v", err)
	}

	cfg := DefaultConfiguration

	if err := cfg.Write(target); err == nil {
		t.Fatal("Write() = nil error, want rename failure")
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("os.ReadDir() error = %v", err)
	}

	if len(entries) != 1 || entries[0].Name() != "target" {
		t.Fatalf("directory entries = %v, want only %q", entries, "target")
	}
}
