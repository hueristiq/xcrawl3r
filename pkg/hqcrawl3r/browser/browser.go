package browser

import (
	"context"
	"fmt"
	"os"

	"github.com/chromedp/chromedp"
)

var GlobalContext context.Context
var GlobalCancel context.CancelFunc

func GetRenderedSource(url string) (source string) {
	// same browser, second tab
	newCtx, newCtxCancel := chromedp.NewContext(GlobalContext)
	defer newCtxCancel()

	// ensure the second tab is created
	if err := chromedp.Run(newCtx); err != nil {
		newCtxCancel()
		fmt.Fprint(os.Stderr, err)
		return
	}

	// navigate to a page, and get it's entire HTML
	if err := chromedp.Run(newCtx,
		chromedp.Navigate(url),
		chromedp.OuterHTML("html", &source),
	); err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}

	return
}

func GetGlobalContext(headless bool, proxy string) (ctx context.Context, cancel context.CancelFunc) {
	var (
		allocCtx context.Context
	)

	if proxy == "" {
		allocCtx, cancel = chromedp.NewExecAllocator(context.Background(),
			chromedp.Flag("headless", headless),
			chromedp.Flag("ignore-certificate-errors", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("no-first-run", true),
			chromedp.Flag("no-default-browser-check", true),
		)
	} else {
		allocCtx, cancel = chromedp.NewExecAllocator(context.Background(),
			chromedp.Flag("headless", headless),
			chromedp.Flag("ignore-certificate-errors", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("no-first-run", true),
			chromedp.Flag("no-default-browser-check", true),
			chromedp.Flag("no-default-browser-check", true),
			chromedp.Flag("proxy-server", proxy),
		)
	}

	// create chrome instance
	ctx, cancel = chromedp.NewContext(allocCtx,
		chromedp.WithBrowserOption(),
	)

	// ensure the first tab is created
	if err := chromedp.Run(ctx); err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}

	return
}
