package xcrawl3r

import "testing"

func TestWithRetriesSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero means no retries", 0, 1},
		{"negative means no retries", -1, 1},
		{"positive value kept", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			crawler := &Crawler{opts: defaultOptions()}

			WithRetries(tt.in)(crawler.opts)

			if crawler.opts.retries != tt.want {
				t.Errorf("retries = %d, want %d", crawler.opts.retries, tt.want)
			}
		})
	}
}
