//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"strings"
	"testing"
	"time"
)

func validConfiguration() Configuration {
	return Configuration{
		Timeout:     time.Second,
		Parallelism: 1,
		Depth:       2,
	}
}

func TestConfigurationValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Configuration)
		wantErr string
	}{
		{name: "valid", mutate: func(*Configuration) {}, wantErr: ""},
		{name: "zero timeout", mutate: func(cfg *Configuration) { cfg.Timeout = 0 }, wantErr: "timeout"},
		{name: "zero parallelism", mutate: func(cfg *Configuration) { cfg.Parallelism = 0 }, wantErr: "parallelism"},
		{name: "negative delay", mutate: func(cfg *Configuration) { cfg.Delay = -time.Second }, wantErr: "delay"},
		{name: "negative random delay", mutate: func(cfg *Configuration) { cfg.RandomDelay = -time.Second }, wantErr: "random delay"},
		{name: "negative crawl duration", mutate: func(cfg *Configuration) { cfg.CrawlDuration = -time.Second }, wantErr: "crawl duration"},
		{name: "negative depth", mutate: func(cfg *Configuration) { cfg.Depth = -1 }, wantErr: "depth"},
		{name: "negative max pages", mutate: func(cfg *Configuration) { cfg.MaxPages = -1 }, wantErr: "max pages"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfiguration()
			tt.mutate(&cfg)

			err := cfg.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() error = %v, want nil", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("validate() error = nil, want substring %q", tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validate() error = %q, want substring %q", err, tt.wantErr)
			}
		})
	}
}
