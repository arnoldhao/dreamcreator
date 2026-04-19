package browsercdp

import (
	"context"
	"errors"
	"testing"

	connectorsdto "dreamcreator/internal/application/connectors/dto"
)

type connectorsReaderStub struct {
	items []connectorsdto.Connector
	err   error
}

func (stub connectorsReaderStub) ListConnectors(context.Context) ([]connectorsdto.Connector, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return append([]connectorsdto.Connector(nil), stub.items...), nil
}

func TestResolveConnectorCookiesForURL_MatchesConnectorPolicy(t *testing.T) {
	t.Parallel()

	cookies, err := ResolveConnectorCookiesForURL(context.Background(), connectorsReaderStub{
		items: []connectorsdto.Connector{
			{
				Type: "google",
				Cookies: []connectorsdto.ConnectorCookie{
					{Name: "SID", Value: "google-cookie", Domain: ".google.com", Path: "/"},
				},
			},
			{
				Type: "github",
				Cookies: []connectorsdto.ConnectorCookie{
					{Name: "logged_in", Value: "yes", Domain: ".github.com", Path: "/"},
				},
			},
		},
	}, "https://www.youtube.com/watch?v=test")
	if err != nil {
		t.Fatalf("resolve cookies: %v", err)
	}
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "SID" || cookies[0].Value != "google-cookie" {
		t.Fatalf("unexpected cookie: %#v", cookies[0])
	}
}

func TestResolveConnectorCookiesForURL_ReturnsNilWhenNoMatch(t *testing.T) {
	t.Parallel()

	cookies, err := ResolveConnectorCookiesForURL(context.Background(), connectorsReaderStub{
		items: []connectorsdto.Connector{
			{
				Type: "github",
				Cookies: []connectorsdto.ConnectorCookie{
					{Name: "logged_in", Value: "yes", Domain: ".github.com", Path: "/"},
				},
			},
		},
	}, "https://example.com/")
	if err != nil {
		t.Fatalf("resolve cookies: %v", err)
	}
	if len(cookies) != 0 {
		t.Fatalf("expected no cookies, got %#v", cookies)
	}
}

func TestResolveConnectorCookiesForURL_PropagatesReaderError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("connectors unavailable")
	_, err := ResolveConnectorCookiesForURL(context.Background(), connectorsReaderStub{
		err: expectedErr,
	}, "https://github.com/openai")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}
