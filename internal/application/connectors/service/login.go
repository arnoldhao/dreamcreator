package service

import (
	"context"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"dreamcreator/internal/application/connectors/dto"
	"dreamcreator/internal/domain/connectors"
)

func (service *ConnectorsService) ConnectConnector(ctx context.Context, request dto.ConnectConnectorRequest) (dto.Connector, error) {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return dto.Connector{}, connectors.ErrInvalidConnector
	}
	connector, err := service.repo.Get(ctx, id)
	if err != nil {
		return dto.Connector{}, err
	}
	targetURL, err := connectorHomeURL(connector.Type)
	if err != nil {
		return dto.Connector{}, err
	}
	cookies, err := runPlaywrightLogin(ctx, targetURL)
	if err != nil {
		return dto.Connector{}, err
	}
	cookiesJSON, err := encodeCookies(cookies)
	if err != nil {
		return dto.Connector{}, err
	}

	now := service.now()
	status := connectors.StatusDisconnected
	var lastVerifiedAt *time.Time
	if len(cookies) > 0 {
		status = connectors.StatusConnected
		lastVerifiedAt = &now
	}
	updated, err := connectors.NewConnector(connectors.ConnectorParams{
		ID:             connector.ID,
		Type:           string(connector.Type),
		Status:         string(status),
		CookiesJSON:    cookiesJSON,
		LastVerifiedAt: lastVerifiedAt,
		CreatedAt:      &connector.CreatedAt,
		UpdatedAt:      &now,
	})
	if err != nil {
		return dto.Connector{}, err
	}
	if err := service.repo.Save(ctx, updated); err != nil {
		return dto.Connector{}, err
	}
	return mapConnectorDTO(updated), nil
}

func connectorHomeURL(connectorType connectors.ConnectorType) (string, error) {
	switch connectorType {
	case connectors.ConnectorGoogle:
		return "https://www.google.com/", nil
	case connectors.ConnectorXiaohongshu:
		return "https://www.xiaohongshu.com/", nil
	case connectors.ConnectorBilibili:
		return "https://www.bilibili.com/", nil
	default:
		return "", connectors.ErrInvalidConnector
	}
}

func runPlaywrightLogin(ctx context.Context, targetURL string) ([]cookieRecord, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, mapPlaywrightInstallError(err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		return nil, mapPlaywrightInstallError(err)
	}
	defer browser.Close()

	browserCtx, err := browser.NewContext()
	if err != nil {
		return nil, err
	}
	page, err := browserCtx.NewPage()
	if err != nil {
		return nil, err
	}
	if _, err := page.Goto(targetURL); err != nil {
		return nil, err
	}

	closeTicker := time.NewTicker(500 * time.Millisecond)
	defer closeTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-closeTicker.C:
			if page.IsClosed() {
				cookies, err := browserCtx.Cookies()
				if err != nil {
					return nil, err
				}
				return cookiesFromPlaywright(cookies), nil
			}
		}
	}
}
