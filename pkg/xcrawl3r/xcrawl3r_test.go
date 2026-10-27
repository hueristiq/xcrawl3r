//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cfg := validConfiguration()

	crawler := newTestCrawler(t, &cfg)
	if crawler.renderer != nil {
		t.Fatal("New() with Render=false created a renderer")
	}

	withRender := validConfiguration()
	withRender.Render = true
	withRender.RenderTimeout = time.Second
	withRender.RenderSettle = time.Second

	crawler = newTestCrawler(t, &withRender)
	if crawler.renderer == nil {
		t.Fatal("New() with Render=true did not create a renderer")
	}
}
