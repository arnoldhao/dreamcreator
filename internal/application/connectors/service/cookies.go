package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"dreamcreator/internal/application/connectors/dto"
	"dreamcreator/internal/domain/connectors"
)

type cookieRecord struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  int64  `json:"expires"`
	HttpOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
	SameSite string `json:"sameSite,omitempty"`
}

func mapConnectorDTO(item connectors.Connector) dto.Connector {
	cookies := decodeCookies(item.CookiesJSON)
	status := item.Status
	if len(cookies) == 0 {
		status = connectors.StatusDisconnected
	} else if status == "" || status == connectors.StatusDisconnected {
		status = connectors.StatusConnected
	}
	lastVerified := ""
	if item.LastVerifiedAt != nil {
		lastVerified = item.LastVerifiedAt.Format(time.RFC3339)
	}
	return dto.Connector{
		ID:             item.ID,
		Type:           string(item.Type),
		Group:          connectorGroup(item.Type),
		Desc:           connectorDesc(item.Type),
		Status:         string(status),
		CookiesCount:   len(cookies),
		Cookies:        mapCookiesDTO(cookies),
		LastVerifiedAt: lastVerified,
	}
}

func connectorGroup(connectorType connectors.ConnectorType) string {
	switch connectorType {
	case connectors.ConnectorGoogle, connectors.ConnectorXiaohongshu:
		return "search_engine"
	case connectors.ConnectorBilibili:
		return "video"
	default:
		return "other"
	}
}

func connectorDesc(connectorType connectors.ConnectorType) string {
	switch connectorType {
	case connectors.ConnectorGoogle:
		return "Global web search with broad multilingual coverage, suitable for general factual lookups."
	case connectors.ConnectorXiaohongshu:
		return "Chinese lifestyle and recommendation community content, useful for reviews and trend discovery."
	case connectors.ConnectorBilibili:
		return "Chinese video platform content, suitable for tutorials, explainers, and creator videos."
	default:
		return ""
	}
}

func mapCookiesDTO(records []cookieRecord) []dto.ConnectorCookie {
	if len(records) == 0 {
		return nil
	}
	result := make([]dto.ConnectorCookie, 0, len(records))
	for _, record := range records {
		result = append(result, dto.ConnectorCookie{
			Name:     record.Name,
			Value:    record.Value,
			Domain:   record.Domain,
			Path:     record.Path,
			Expires:  record.Expires,
			HttpOnly: record.HttpOnly,
			Secure:   record.Secure,
			SameSite: record.SameSite,
		})
	}
	return result
}

func encodeCookies(records []cookieRecord) (string, error) {
	if len(records) == 0 {
		return "", nil
	}
	data, err := json.Marshal(records)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeCookies(data string) []cookieRecord {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return nil
	}
	var records []cookieRecord
	if err := json.Unmarshal([]byte(trimmed), &records); err != nil {
		return nil
	}
	return records
}

func cookiesFromPlaywright(cookies []playwright.Cookie) []cookieRecord {
	if len(cookies) == 0 {
		return nil
	}
	result := make([]cookieRecord, 0, len(cookies))
	for _, cookie := range cookies {
		sameSite := ""
		if cookie.SameSite != nil {
			sameSite = strings.ToLower(string(*cookie.SameSite))
		}
		result = append(result, cookieRecord{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  int64(cookie.Expires),
			HttpOnly: cookie.HttpOnly,
			Secure:   cookie.Secure,
			SameSite: sameSite,
		})
	}
	return result
}

func toPlaywrightCookies(records []cookieRecord, targetURL string) []playwright.OptionalCookie {
	if len(records) == 0 {
		return nil
	}
	result := make([]playwright.OptionalCookie, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.Name) == "" {
			continue
		}
		cookie := playwright.OptionalCookie{
			Name:  record.Name,
			Value: record.Value,
		}
		domain := strings.TrimSpace(record.Domain)
		path := strings.TrimSpace(record.Path)
		if domain != "" {
			cookie.Domain = playwright.String(domain)
		} else if strings.TrimSpace(targetURL) != "" {
			cookie.URL = playwright.String(strings.TrimSpace(targetURL))
		}
		if path != "" {
			cookie.Path = playwright.String(path)
		}
		if record.Expires > 0 {
			cookie.Expires = playwright.Float(float64(record.Expires))
		}
		cookie.HttpOnly = playwright.Bool(record.HttpOnly)
		cookie.Secure = playwright.Bool(record.Secure)
		if sameSite := mapSameSite(record.SameSite); sameSite != nil {
			cookie.SameSite = sameSite
		}
		result = append(result, cookie)
	}
	return result
}

func mapSameSite(value string) *playwright.SameSiteAttribute {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "lax":
		return playwright.SameSiteAttributeLax
	case "strict":
		return playwright.SameSiteAttributeStrict
	case "none":
		return playwright.SameSiteAttributeNone
	default:
		return nil
	}
}
