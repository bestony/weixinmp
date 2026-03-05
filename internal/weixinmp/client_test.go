package weixinmp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type errReadCloser struct{ err error }

func (e errReadCloser) Read(p []byte) (int, error) { return 0, e.err }
func (e errReadCloser) Close() error               { return nil }

func TestClientGetAccessTokenOK(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/cgi-bin/token" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/cgi-bin/token")
		}
		q := r.URL.Query()
		if got := q.Get("grant_type"); got != "client_credential" {
			t.Fatalf("grant_type = %q, want %q", got, "client_credential")
		}
		if got := q.Get("appid"); got != "app" {
			t.Fatalf("appid = %q, want %q", got, "app")
		}
		if got := q.Get("secret"); got != "secret" {
			t.Fatalf("secret = %q, want %q", got, "secret")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	client := &Client{
		BaseURL:    srv.URL,
		HTTPClient: &http.Client{Timeout: 2 * time.Second},
	}
	got, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err != nil {
		t.Fatalf("GetAccessToken() err = %v, want nil", err)
	}
	if got.AccessToken != "abc" {
		t.Fatalf("AccessToken = %q, want %q", got.AccessToken, "abc")
	}
	if got.ExpiresIn != 7200 {
		t.Fatalf("ExpiresIn = %d, want %d", got.ExpiresIn, 7200)
	}
}

func TestClientGetAccessTokenUsesDefaultHTTPClientWhenNil(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	client := &Client{
		BaseURL: srv.URL,
		// HTTPClient intentionally nil.
	}
	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err != nil {
		t.Fatalf("GetAccessToken() err = %v, want nil", err)
	}
}

func TestClientGetAccessTokenAPIError(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"invalid appid"}`))
	}))
	defer srv.Close()

	client := &Client{
		BaseURL:    srv.URL,
		HTTPClient: &http.Client{Timeout: 2 * time.Second},
	}
	_, err := client.GetAccessToken(context.Background(), "bad", "bad")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("GetAccessToken() err = %T, want *APIError", err)
	}
	if apiErr.Code != 40013 {
		t.Fatalf("Code = %d, want %d", apiErr.Code, 40013)
	}
	if apiErr.Message == "" {
		t.Fatalf("Message is empty, want non-empty")
	}
	if got := apiErr.Error(); !strings.Contains(got, "errcode=40013") {
		t.Fatalf("Error() = %q, want substring %q", got, "errcode=40013")
	}
}

func TestClientGetAccessTokenUsesDefaultBaseURLWhenEmpty(t *testing.T) {
	t.Helper()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Scheme != "https" {
				t.Fatalf("scheme = %q, want %q", r.URL.Scheme, "https")
			}
			if r.URL.Host != "api.weixin.qq.com" {
				t.Fatalf("host = %q, want %q", r.URL.Host, "api.weixin.qq.com")
			}
			if r.URL.Path != "/cgi-bin/token" {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/cgi-bin/token")
			}
			w := io.NopCloser(strings.NewReader(`{"access_token":"abc","expires_in":7200}`))
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       w,
				Request:    r,
			}, nil
		}),
	}

	client := &Client{
		BaseURL:    "", // trigger default base URL
		HTTPClient: httpClient,
	}
	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err != nil {
		t.Fatalf("GetAccessToken() err = %v, want nil", err)
	}
}

func TestClientGetAccessTokenInvalidBaseURL(t *testing.T) {
	t.Helper()

	client := &Client{
		BaseURL:    "http://[::1", // missing closing bracket
		HTTPClient: &http.Client{Timeout: 1 * time.Second},
	}
	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "invalid base url") {
		t.Fatalf("err = %q, want substring %q", err.Error(), "invalid base url")
	}
}

func TestClientGetAccessTokenCreateRequestError(t *testing.T) {
	t.Helper()

	// NewRequestWithContext rejects nil contexts.
	client := &Client{
		BaseURL: "https://example.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				t.Fatalf("RoundTrip() called, want not called")
				return nil, nil
			}),
		},
	}
	_, err := client.GetAccessToken(nil, "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "create request") {
		t.Fatalf("err = %q, want substring %q", err.Error(), "create request")
	}
}

func TestClientGetAccessTokenRequestError(t *testing.T) {
	t.Helper()

	sent := errors.New("dial tcp: test error")
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, sent
		}),
	}
	client := &Client{
		BaseURL:    "https://example.com",
		HTTPClient: httpClient,
	}
	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "request access token") {
		t.Fatalf("err = %q, want substring %q", err.Error(), "request access token")
	}
}

func TestClientGetAccessTokenReadResponseError(t *testing.T) {
	t.Helper()

	sent := errors.New("read: test error")
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       errReadCloser{err: sent},
				Request:    r,
			}, nil
		}),
	}
	client := &Client{
		BaseURL:    "https://example.com",
		HTTPClient: httpClient,
	}
	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "read response") {
		t.Fatalf("err = %q, want substring %q", err.Error(), "read response")
	}
}

func TestClientGetAccessTokenUnexpectedStatus(t *testing.T) {
	t.Helper()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(strings.NewReader("boom\n")),
				Request:    r,
			}, nil
		}),
	}
	client := &Client{
		BaseURL:    "https://example.com",
		HTTPClient: httpClient,
	}
	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 500 Internal Server Error") {
		t.Fatalf("err = %q, want substring %q", err.Error(), "unexpected status 500 Internal Server Error")
	}
}

func TestClientGetAccessTokenDecodeJSONError(t *testing.T) {
	t.Helper()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader("{not json")),
				Request:    r,
			}, nil
		}),
	}
	client := &Client{
		BaseURL:    "https://example.com",
		HTTPClient: httpClient,
	}
	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "decode json response") {
		t.Fatalf("err = %q, want substring %q", err.Error(), "decode json response")
	}
}

func TestClientGetAccessTokenMissingAccessToken(t *testing.T) {
	t.Helper()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"expires_in":7200}`)),
				Request:    r,
			}, nil
		}),
	}
	client := &Client{
		BaseURL:    "https://example.com",
		HTTPClient: httpClient,
	}
	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if got, want := err.Error(), "missing access_token"; !strings.Contains(got, want) {
		t.Fatalf("err = %q, want substring %q", got, want)
	}
}

func TestAPIErrorError(t *testing.T) {
	t.Helper()

	err := (&APIError{Code: 123, Message: "oops"}).Error()
	if got, want := err, "weixinmp api error: errcode=123 errmsg=oops"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestClientGetAccessTokenOKIncludesRequestQueryParams(t *testing.T) {
	t.Helper()

	// Coverage for the request's query parameter encoding (appid/secret/grant_type).
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	client := &Client{BaseURL: srv.URL, HTTPClient: &http.Client{Timeout: 2 * time.Second}}
	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err != nil {
		t.Fatalf("GetAccessToken() err = %v, want nil", err)
	}
	if gotRawQuery == "" {
		t.Fatalf("RawQuery is empty, want non-empty")
	}
	if !strings.Contains(gotRawQuery, "grant_type=client_credential") {
		t.Fatalf("RawQuery = %q, want substring %q", gotRawQuery, "grant_type=client_credential")
	}
}
