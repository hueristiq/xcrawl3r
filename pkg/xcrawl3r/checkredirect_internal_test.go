package xcrawl3r

import (
	"errors"
	"net/http"
	"testing"
)

func TestCheckRedirect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		domains    []string
		target     string
		preWiden   string
		wantFollow bool
		wantWiden  bool
		wantWarn   bool
	}{
		{
			name:       "in-scope redirect followed",
			domains:    []string{exampleDomain},
			target:     "https://example.com/deep",
			wantFollow: true,
		},
		{
			name:       "same-registrable-domain redirect followed and widens scope",
			domains:    []string{exampleDomain},
			target:     "https://www.example.com/",
			wantFollow: true,
			wantWiden:  true,
			wantWarn:   true,
		},
		{
			name:       "subdomain redirect followed and widens scope",
			domains:    []string{exampleDomain},
			target:     "https://api.example.com/v1",
			wantFollow: true,
			wantWiden:  true,
			wantWarn:   true,
		},
		{
			name:    "different registrable domain refused",
			domains: []string{exampleDomain},
			target:  "https://evil.com/",
		},
		{
			name:    "sibling registrable domain refused",
			domains: []string{exampleDomain},
			target:  "https://example.org/",
		},
		{
			name:       "multi-label public suffix boundary respected",
			domains:    []string{"example.co.uk"},
			target:     "https://www.example.co.uk/",
			wantFollow: true,
			wantWiden:  true,
			wantWarn:   true,
		},
		{
			name:    "public suffix boundary refused",
			domains: []string{"example.co.uk"},
			target:  "https://example.uk/",
		},
		{
			name:    "ip redirect refused",
			domains: []string{"127.0.0.1"},
			target:  "http://127.0.0.2/",
		},
		{
			name:       "redirect to an already-widened host does not warn again",
			domains:    []string{exampleDomain},
			target:     "https://www.example.com/other",
			preWiden:   "www.example.com",
			wantFollow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &scope{domains: tt.domains}

			if tt.preWiden != "" {
				s.widen(tt.preWiden)
			}

			results := make(chan Result, 2)

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, tt.target, http.NoBody)
			if err != nil {
				t.Fatalf("http.NewRequestWithContext() error = %v", err)
			}

			err = checkRedirect(t.Context(), results, s, request)

			if tt.wantFollow && err != nil {
				t.Errorf("checkRedirect() error = %v, want redirect followed", err)
			}

			if !tt.wantFollow && !errors.Is(err, http.ErrUseLastResponse) {
				t.Errorf("checkRedirect() error = %v, want %v", err, http.ErrUseLastResponse)
			}

			if tt.wantWiden && !s.allows(tt.target) {
				t.Errorf("scope was not widened to %s", tt.target)
			}

			if tt.wantWarn {
				select {
				case result := <-results:
					if result.Type != ResultWarning {
						t.Errorf("result type = %v, want %v", result.Type, ResultWarning)
					}
				default:
					t.Error("expected a scope-widening warning")
				}
			}
		})
	}
}
