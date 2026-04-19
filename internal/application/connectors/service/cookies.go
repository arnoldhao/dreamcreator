package service

import (
	"time"

	"dreamcreator/internal/application/connectors/dto"
	appcookies "dreamcreator/internal/application/cookies"
	"dreamcreator/internal/application/sitepolicy"
	"dreamcreator/internal/domain/connectors"
)

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
	policy, _ := sitepolicy.ForConnectorType(string(item.Type))
	return dto.Connector{
		ID:             item.ID,
		Type:           string(item.Type),
		Group:          connectorGroup(item.Type),
		Desc:           connectorDesc(item.Type),
		Status:         string(status),
		CookiesCount:   len(cookies),
		Cookies:        mapCookiesDTO(cookies),
		Domains:        append([]string(nil), policy.Domains...),
		PolicyKey:      policy.Key,
		Capabilities:   append([]string(nil), policy.Capabilities...),
		LastVerifiedAt: lastVerified,
	}
}

func connectorGroup(connectorType connectors.ConnectorType) string {
	switch connectorType {
	case connectors.ConnectorGoogle, connectors.ConnectorZhihu:
		return "search_engine"
	case connectors.ConnectorXiaohongshu, connectors.ConnectorReddit, connectors.ConnectorX:
		return "community"
	case connectors.ConnectorBilibili:
		return "video"
	case connectors.ConnectorGitHub:
		return "developer"
	default:
		return "other"
	}
}

func connectorDesc(connectorType connectors.ConnectorType) string {
	switch connectorType {
	case connectors.ConnectorGoogle:
		return "Global web search with broad multilingual coverage, suitable for general factual lookups."
	case connectors.ConnectorGitHub:
		return "Developer collaboration and code hosting, useful for repositories, issues, pull requests, and docs."
	case connectors.ConnectorReddit:
		return "Community discussions and long-tail troubleshooting threads, useful for niche product and user experience research."
	case connectors.ConnectorZhihu:
		return "Chinese knowledge-sharing community content, useful for answers, articles, and topic exploration."
	case connectors.ConnectorX:
		return "Real-time public conversation and creator timelines, useful for posts, threads, and timely updates."
	case connectors.ConnectorXiaohongshu:
		return "Chinese lifestyle and recommendation community content, useful for reviews and trend discovery."
	case connectors.ConnectorBilibili:
		return "Chinese video platform content, suitable for tutorials, explainers, and creator videos."
	default:
		return ""
	}
}

func mapCookiesDTO(records []appcookies.Record) []dto.ConnectorCookie {
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

func encodeCookies(records []appcookies.Record) (string, error) {
	return appcookies.EncodeJSON(records)
}

func decodeCookies(data string) []appcookies.Record {
	return appcookies.DecodeJSON(data)
}
