package xcrawl3r

import (
	"slices"
	"testing"
)

func TestTargets(t *testing.T) {
	t.Parallel()

	parsedTargetURL, err := up.Parse("example.com/path")
	if err != nil {
		t.Fatalf("up.Parse() error = %v", err)
	}

	targets := (&Crawler{}).targets(parsedTargetURL)

	if targets[0] != "https://example.com/path" {
		t.Errorf("targets[0] = %q, want seed URL", targets[0])
	}

	if !slices.Contains(targets, "https://example.com/robots.txt") {
		t.Error("targets should contain robots.txt")
	}

	if !slices.Contains(targets, "https://example.com/sitemap.xml") {
		t.Error("targets should contain sitemap.xml")
	}

	if want := 2 + len(sitemapPaths); len(targets) != want {
		t.Errorf("len(targets) = %d, want %d", len(targets), want)
	}
}

func TestTargetsMinifiedCompanion(t *testing.T) {
	t.Parallel()

	parsedTargetURL, err := up.Parse("https://example.com/app.min.js")
	if err != nil {
		t.Fatalf("up.Parse() error = %v", err)
	}

	targets := (&Crawler{}).targets(parsedTargetURL)

	if !slices.Contains(targets, "https://example.com/app.js") {
		t.Error("targets should contain the unminified companion")
	}
}
