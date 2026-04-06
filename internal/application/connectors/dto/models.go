package dto

type Connector struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Group          string            `json:"group"`
	Desc           string            `json:"desc"`
	Status         string            `json:"status"`
	CookiesCount   int               `json:"cookiesCount"`
	Cookies        []ConnectorCookie `json:"cookies"`
	LastVerifiedAt string            `json:"lastVerifiedAt"`
}

type UpsertConnectorRequest struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	CookiesPath string `json:"cookiesPath"`
}

type ClearConnectorRequest struct {
	ID string `json:"id"`
}

type ConnectConnectorRequest struct {
	ID string `json:"id"`
}

type OpenConnectorSiteRequest struct {
	ID string `json:"id"`
}

type ConnectorCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  int64  `json:"expires"`
	HttpOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
	SameSite string `json:"sameSite,omitempty"`
}
