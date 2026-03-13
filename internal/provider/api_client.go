package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type APIClient struct {
	BaseURL       string
	APIVersion    string
	ServiceTicket string
	HTTP          *http.Client
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain,omitempty"`
}
type loginResp struct {
	ServiceTicket string `json:"serviceTicket"`
}

// LoginForServiceTicket issues the SmartZone login and returns the serviceTicket.
// It uses the preferred public API path under /wsg/api/public/{version}/serviceTicket. [3](https://docs.ruckuswireless.com/smartzone/7.1.1/vsze-public-api-reference-guide-711.html)
func LoginForServiceTicket(ctx context.Context, c *http.Client, base, ver, user, pass, domain string) (string, error) {
	url := fmt.Sprintf("%s/wsg/api/public/%s/serviceTicket", base, ver)
	payload, _ := json.Marshal(loginReq{Username: user, Password: pass, Domain: domain})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	var lr loginResp
	if err := doJSON(ctx, c, req, &lr); err != nil {
		return "", err
	}
	if lr.ServiceTicket == "" {
		return "", fmt.Errorf("missing serviceTicket in response")
	}
	return lr.ServiceTicket, nil
}
