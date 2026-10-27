package xcrawl3r

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	defaultRenderTimeout = 10 * time.Second
	defaultRenderSettle  = time.Second
	renderStartTimeout   = 30 * time.Second
)

type renderer struct {
	cfg *Configuration

	timeout time.Duration
	settle  time.Duration

	mu          sync.Mutex
	browser     context.Context //nolint:containedctx // browser lifecycle handle, not a request scope.
	cancel      context.CancelFunc
	cancelAlloc context.CancelFunc
	started     atomic.Bool
	failed      bool
	err         error
}

//nolint:contextcheck // the browser must outlive the per-crawl context; startup is linked to it with a watchdog.
func (r *renderer) start(ctx context.Context) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.failed {
		return r.err
	}

	if r.started.Load() {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), renderOptions(r.cfg)...)
	browser, cancel := chromedp.NewContext(allocCtx)

	// chromedp binds the browser process to the context of its first Run, so
	// a timeout context would kill a successfully started browser when it is
	// canceled. Bound startup with watchdogs instead: one for the timeout and
	// one for the crawl context.
	startupTimer := time.AfterFunc(renderStartTimeout, cancel)
	stopStartup := context.AfterFunc(ctx, cancel)

	err = chromedp.Do(browser, chromedp.Evaluate[chromedp.Void]("1"))

	timerStopped := startupTimer.Stop()

	stopStartup()

	switch {
	case err != nil:
	case !timerStopped:
		err = fmt.Errorf("browser startup timed out after %s", renderStartTimeout)
	case ctx.Err() != nil:
		err = fmt.Errorf("browser startup interrupted: %w", ctx.Err())
	}

	if err != nil {
		cancel()
		cancelAlloc()

		r.browser = nil
		r.cancel = nil
		r.cancelAlloc = nil
		r.err = err
		r.failed = true

		return
	}

	r.browser = browser
	r.cancel = cancel
	r.cancelAlloc = cancelAlloc
	r.err = nil
	r.started.Store(true)

	return
}

func (r *renderer) close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started.Load() {
		r.cancel()
		r.cancelAlloc()
	}

	r.browser = nil
	r.cancel = nil
	r.cancelAlloc = nil
	r.err = nil
	r.failed = false
	r.started.Store(false)
}

// Validate checks whether the browser needed for rendering can be started.
func (c *Crawler) Validate(ctx context.Context) (err error) {
	if c.renderer == nil {
		return
	}

	err = c.renderer.start(ctx)

	return
}

// DisableRender turns rendering off for this crawler. It is used when the
// browser cannot be started so that crawling can continue without it.
func (c *Crawler) DisableRender() {
	c.cfg.Render = false

	if c.renderer != nil {
		c.renderer.close()
		c.renderer = nil
	}
}

func (c *Crawler) Close() {
	if c.renderer != nil {
		c.renderer.close()
	}
}

// render loads rawURL in a headless Chrome tab and returns the rendered
// document.
func (c *Crawler) render(ctx context.Context, rawURL string, scope *scope) (renderedHTML string, err error) {
	if err = c.renderer.start(ctx); err != nil {
		return
	}

	tabCtx, cancelTab := chromedp.NewContext(c.renderer.browser)
	defer cancelTab()

	tabCtx, cancelTimeout := context.WithTimeout(tabCtx, c.renderer.timeout)
	defer cancelTimeout()

	stop := context.AfterFunc(ctx, cancelTimeout)
	defer stop()

	if _, err = chromedp.Call(tabCtx, network.Enable, network.EnableParams{}); err != nil { //nolint:contextcheck // tabCtx is a chromedp tab context.
		return
	}

	if headers := parseHeaderEntries(c.cfg.Headers); len(headers) > 0 {
		if _, err = chromedp.Call(tabCtx, network.SetExtraHTTPHeaders, network.SetExtraHTTPHeadersParams{Headers: headers}); err != nil { //nolint:contextcheck // tabCtx is a chromedp tab context.
			return
		}
	}

	cookies := cookieParams(c.cfg.Cookies, rawURL)

	for index := range cookies {
		if _, err = chromedp.Call(tabCtx, network.SetCookie, cookies[index]); err != nil { //nolint:contextcheck // tabCtx is a chromedp tab context.
			return
		}
	}

	settle := c.renderer.settle
	if settle > c.renderer.timeout {
		settle = c.renderer.timeout
	}

	if err = chromedp.Do(tabCtx, //nolint:contextcheck // tabCtx is a chromedp tab context, derived from the browser context.
		chromedp.Navigate(rawURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(settle),
	); err != nil {
		return
	}

	var location string

	if location, err = chromedp.Run(tabCtx, chromedp.Location()); err != nil { //nolint:contextcheck // tabCtx is a chromedp tab context.
		return
	}

	// Chrome follows redirects the HTTP client would have blocked, so reject
	// a rendered page that left the configured scope.
	if !scope.allows(location) {
		err = fmt.Errorf("rendered page navigated outside the crawl scope: %s", location)

		return
	}

	renderedHTML, err = chromedp.Run(tabCtx, chromedp.OuterHTML(chromedp.CSS("html"))) //nolint:contextcheck // tabCtx is a chromedp tab context.

	return
}

// renderOptions returns the browser allocator options derived from cfg. The
// defaults come first; configuration-specific options are appended.
func renderOptions(cfg *Configuration) (options []chromedp.ExecAllocatorOption) {
	options = append(options, chromedp.DefaultExecAllocatorOptions[:]...)

	if cfg == nil {
		return
	}

	if len(cfg.Proxies) > 0 {
		// Chrome supports a single proxy; use the first configured one.
		options = append(options, chromedp.ProxyServer(cfg.Proxies[0]))
	}

	if cfg.InsecureSkipVerify {
		options = append(options, chromedp.IgnoreCertErrors)
	}

	if cfg.Client != nil && cfg.Client.UserAgent != "" {
		options = append(options, chromedp.UserAgent(cfg.Client.UserAgent))
	}

	return
}

// parseHeaderEntries converts "Name: value" entries into CDP headers. Entries
// without a colon or with an empty name are ignored; values are trimmed.
func parseHeaderEntries(entries []string) (headers network.Headers) {
	headers = make(network.Headers)

	for _, entry := range entries {
		name, value, found := strings.Cut(entry, ":")
		if !found {
			continue
		}

		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		headers[name] = strings.TrimSpace(value)
	}

	return
}

// cookieParams converts "name=value" entries into CDP cookies bound to
// rawURL. Cookie attributes after the first ';' are ignored for the CDP
// cookie; entries without '=' or with an empty name are skipped.
func cookieParams(entries []string, rawURL string) (cookies []network.SetCookieParams) {
	for _, entry := range entries {
		pair, _, _ := strings.Cut(entry, ";")

		name, value, found := strings.Cut(pair, "=")

		name = strings.TrimSpace(name)
		if !found || name == "" {
			continue
		}

		cookies = append(cookies, network.SetCookieParams{
			Name:  name,
			Value: strings.TrimSpace(value),
			URL:   rawURL,
		})
	}

	return
}
