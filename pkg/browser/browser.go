package browser

import (
	"context"
	"log"

	"github.com/chromedp/chromedp"
	"github.com/hueristiq/hqgolog"
)

var (
	GlobalContext context.Context
	GlobalCancel  context.CancelFunc
)

func GetRenderedSource(url string) (outerHTML string) {
	// same browser, second tab
	newCtx, newCtxCancel := chromedp.NewContext(GlobalContext)

	defer newCtxCancel()

	// ensure the second tab is created
	if err := chromedp.Run(newCtx); err != nil {
		newCtxCancel()

		hqgolog.Fatal().Msg(err.Error())
	}

	// navigate to a page, and get it's entire HTML
	if err := chromedp.Run(newCtx, chromedp.Navigate(url), chromedp.OuterHTML("html", &outerHTML)); err != nil {
		hqgolog.Error().Msg(err.Error())
	}

	return
}

func GetGlobalContext(headless bool, proxy string) (ctx context.Context, cancel context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)

	if proxy != "" {
		opts = append(opts, chromedp.Flag("proxy-server", proxy))
	}

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)

	ctx, cancel = chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf), chromedp.WithBrowserOption())

	// ensure the first tab is created
	if err := chromedp.Run(ctx); err != nil {
		hqgolog.Fatal().Msg(err.Error())
	}

	return
}
