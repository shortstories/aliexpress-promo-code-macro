package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
)

func main() {
	coupon := "AA6940"
	timeout := 6 * time.Hour

	ctx := context.Background()
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(""),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-extensions", false),
		chromedp.Flag(`enable-automation`, false),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, options...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(
		allocCtx,
		//chromedp.WithDebugf(log.Printf),
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://aliexpress.com/`),
		chromedp.WaitEnabled(`input.pl-promoCode-input`),
		chromedp.SendKeys(`input[class="pl-promoCode-input"]`, coupon, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			ticker := time.NewTicker(time.Second * 1)
			for {
				select {
				case <-ticker.C:
					var startSavedString string
					if err := chromedp.Text(`div.pl-summary__item-pc:nth-last-child(2) > div.pl-summary__item-content-pc > div`,
						&startSavedString, chromedp.ByQuery).Do(ctx); err != nil {
						log.Error().Err(err).Msg("failed to get start saved text")
						continue
					}
					startSavedString = strings.TrimPrefix(startSavedString, "-US $")
					startSaved, err := strconv.ParseFloat(startSavedString, 32)
					if err != nil {
						log.Error().Err(err).Msg("failed to parse start saved text")
						continue
					}

					log.Info().Msg("trying to apply promo code...")
					if err := chromedp.Click(`button.pl-promoCode-input-button`).Do(ctx); err != nil {
						log.Error().Err(err).Msg("failed to click button")
						continue
					}
					log.Debug().Msg("wait loading...")
					if err := chromedp.WaitNotPresent(`.comet-loading`).Do(ctx); err != nil {
						log.Error().Err(err).Msg("failed to wait loading")
						continue
					}

					log.Debug().Msg("get result...")
					var resultSavedString string
					if err := chromedp.Text(`div.pl-summary__item-pc:nth-last-child(2) > div.pl-summary__item-content-pc > div`,
						&resultSavedString, chromedp.ByQuery).Do(ctx); err != nil {
						log.Error().Err(err).Msg("failed to get result saved text")
						continue
					}
					resultSavedString = strings.TrimPrefix(resultSavedString, "-US $")
					resultSaved, err := strconv.ParseFloat(resultSavedString, 32)
					if err != nil {
						log.Error().Err(err).Msg("failed to parse result saved text")
						continue
					}

					if startSaved == resultSaved {
						var errorPrompt string
						if err := chromedp.Text(`div.promoErrorTip`, &errorPrompt).Do(ctx); err != nil {
							log.Error().Err(err).Msg("failed to get result text")
						}
						log.Info().Str("prompt", errorPrompt).Msg("error prompt")
					} else {
						log.Info().
							Str("coupon", coupon).
							Str("resultPrice", resultSavedString).
							Msg("coupon has been applied")
						return nil
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}),
	)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}
