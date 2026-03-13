package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain,omitempty"`
}
type loginResp struct {
	ServiceTicket string `json:"serviceTicket"`
}

func LoginForServiceTicket(ctx context.Context, c *http.Client, base, ver, user, pass, domain string) (string, error) {
	url := fmt.Sprintf("%s/wsg/api/public/%s/serviceTicket", base, ver)
	body, _ := json.Marshal(loginReq{Username: user, Password: pass, Domain: domain})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("login status %d", resp.StatusCode)
	}
	var lr loginResp
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return "", err
	}
	if lr.ServiceTicket == "" {
		return "", fmt.Errorf("no serviceTicket in response")
	}
	return lr.ServiceTicket, nil
}
