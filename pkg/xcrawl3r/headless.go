package xcrawl3r

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/chromedp"
)

const defaultHeadlessTimeout = 10 * time.Second

// headlessLinksJS collects the absolute URLs of every element carrying an href
// or src attribute from the rendered DOM. e.href/e.src resolve relative URLs
// against document.baseURI, so links injected by client-side routing are
// captured.
const headlessLinksJS = `(() => {
	const links = new Set();

	for (const e of document.querySelectorAll('[href],[src]')) {
		const v = e.href || e.src;

		if (v) {
			links.add(v);
		}
	}

	return [...links];
})()`

type headless struct {
	timeout time.Duration

	mu          sync.Mutex
	browser     context.Context //nolint:containedctx // browser lifecycle handle, not a request scope.
	cancel      context.CancelFunc
	cancelAlloc context.CancelFunc
	started     atomic.Bool
	err         error
}

func (h *headless) start() (err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started.Load() {
		return h.err
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)

	h.browser, h.cancel = chromedp.NewContext(allocCtx)

	h.err = chromedp.Do(h.browser, chromedp.Evaluate[chromedp.Void]("1"))
	if h.err != nil {
		h.cancel()
		cancelAlloc()

		h.browser = nil
		h.cancel = nil

		return
	}

	h.cancelAlloc = cancelAlloc
	h.started.Store(true)

	return
}

func (h *headless) close() {
	if !h.started.Load() {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.started.Load() {
		return
	}

	h.cancel()
	h.cancelAlloc()

	h.browser = nil
	h.cancel = nil
	h.cancelAlloc = nil
	h.err = nil
	h.started.Store(false)
}

// Validate starts the headless browser early when headless mode is enabled, so
// a missing Chrome/Chromium installation surfaces immediately instead of
// silently degrading every render.
func (c *Crawler) Validate() (err error) {
	if c.headless == nil {
		return
	}

	err = c.headless.start()

	return
}

// Close shuts the headless browser down. It must be called when the crawler is
// no longer needed and headless mode was enabled.
func (c *Crawler) Close() {
	if c.headless != nil {
		c.headless.close()
	}
}

// render loads rawURL in a headless browser tab and returns the absolute URLs
// found in the rendered DOM: attribute links plus absolute URLs matched by the
// extractor regex in the rendered HTML.
func (c *Crawler) render(ctx context.Context, rawURL string) (urls []string, err error) {
	if err = c.headless.start(); err != nil {
		return
	}

	tabCtx, cancelTab := chromedp.NewContext(c.headless.browser)
	defer cancelTab()

	tabCtx, cancelTimeout := context.WithTimeout(tabCtx, c.headless.timeout)
	defer cancelTimeout()

	stop := context.AfterFunc(ctx, cancelTimeout)
	defer stop()

	if err = chromedp.Do(tabCtx, //nolint:contextcheck // tabCtx is a chromedp tab context, derived from the browser context.
		chromedp.Navigate(rawURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(500*time.Millisecond),
	); err != nil {
		return
	}

	var renderedHTML string

	renderedHTML, err = chromedp.Run(tabCtx, chromedp.OuterHTML(chromedp.CSS("html"))) //nolint:contextcheck // tabCtx is a chromedp tab context.
	if err != nil {
		return
	}

	var links []string

	links, err = chromedp.Run(tabCtx, chromedp.Evaluate[[]string](headlessLinksJS)) //nolint:contextcheck // tabCtx is a chromedp tab context.
	if err != nil {
		return
	}

	urls = append(urls, links...)

	for _, link := range c.urlExtractorRegex.FindAllString(renderedHTML, -1) {
		if strings.Contains(link, "://") {
			urls = append(urls, link)
		}
	}

	return
}
