package weixinmp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client talks to WeChat Official Account (Weixin MP) HTTP APIs.
type Client struct {
	// BaseURL defaults to "https://api.weixin.qq.com".
	BaseURL string
	// HTTPClient defaults to http.DefaultClient.
	HTTPClient *http.Client
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// AccessToken is the successful response from /cgi-bin/token.
type AccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("weixinmp api error: errcode=%d errmsg=%s", e.Code, e.Message)
}

func (c *Client) GetAccessToken(ctx context.Context, appID, secret string) (AccessToken, error) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://api.weixin.qq.com"
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return AccessToken{}, fmt.Errorf("invalid base url %q: %w", baseURL, err)
	}

	endpoint, _ := url.Parse("/cgi-bin/token")
	u := base.ResolveReference(endpoint)
	q := u.Query()
	q.Set("grant_type", "client_credential")
	q.Set("appid", appID)
	q.Set("secret", secret)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return AccessToken{}, fmt.Errorf("create request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return AccessToken{}, fmt.Errorf("request access token: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return AccessToken{}, fmt.Errorf("read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return AccessToken{}, fmt.Errorf("unexpected status %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	var decoded accessTokenResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return AccessToken{}, fmt.Errorf("decode json response: %w", err)
	}
	if decoded.ErrCode != 0 {
		return AccessToken{}, &APIError{Code: decoded.ErrCode, Message: decoded.ErrMsg}
	}
	if decoded.AccessToken == "" {
		return AccessToken{}, fmt.Errorf("unexpected response: missing access_token")
	}

	return AccessToken{
		AccessToken: decoded.AccessToken,
		ExpiresIn:   decoded.ExpiresIn,
	}, nil
}

