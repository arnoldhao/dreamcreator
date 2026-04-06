package service

import (
	"context"
	"errors"
	"strings"

	"github.com/playwright-community/playwright-go"
)

var ErrPlaywrightNotInstalled = errors.New("playwright not installed")

func (service *ConnectorsService) InstallPlaywright(_ context.Context) error {
	return playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
	})
}

func mapPlaywrightInstallError(err error) error {
	if err == nil {
		return nil
	}
	if isPlaywrightInstallError(err.Error()) {
		return ErrPlaywrightNotInstalled
	}
	return err
}

func isPlaywrightInstallError(message string) bool {
	lowered := strings.ToLower(message)
	if strings.Contains(lowered, "playwright not installed") {
		return true
	}
	if strings.Contains(lowered, "please install") && (strings.Contains(lowered, "playwright") || strings.Contains(lowered, "driver")) {
		return true
	}
	if strings.Contains(lowered, "driver exists but version not") {
		return true
	}
	if strings.Contains(lowered, "executable doesn't exist") {
		return true
	}
	return false
}
