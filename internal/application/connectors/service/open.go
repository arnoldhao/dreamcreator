package service

import (
	"context"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"dreamcreator/internal/application/connectors/dto"
	"dreamcreator/internal/domain/connectors"
)

func (service *ConnectorsService) OpenConnectorSite(ctx context.Context, request dto.OpenConnectorSiteRequest) error {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return connectors.ErrInvalidConnector
	}
	connector, err := service.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	cookies := decodeCookies(connector.CookiesJSON)
	if len(cookies) == 0 {
		return connectors.ErrNoCookies
	}
	targetURL, err := connectorHomeURL(connector.Type)
	if err != nil {
		return err
	}
	return runPlaywrightOpenWithCookies(ctx, targetURL, cookies)
}

func runPlaywrightOpenWithCookies(ctx context.Context, targetURL string, cookies []cookieRecord) error {
	pw, err := playwright.Run()
	if err != nil {
		return mapPlaywrightInstallError(err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		return mapPlaywrightInstallError(err)
	}
	defer browser.Close()

	browserCtx, err := browser.NewContext()
	if err != nil {
		return err
	}
	if len(cookies) > 0 {
		if err := browserCtx.AddCookies(toPlaywrightCookies(cookies, targetURL)); err != nil {
			return err
		}
	}
	page, err := browserCtx.NewPage()
	if err != nil {
		return err
	}
	if _, err := page.Goto(targetURL); err != nil {
		return err
	}

	closeTicker := time.NewTicker(500 * time.Millisecond)
	defer closeTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-closeTicker.C:
			if page.IsClosed() {
				return nil
			}
		}
	}
}
