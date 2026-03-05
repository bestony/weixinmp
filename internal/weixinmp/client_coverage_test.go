package weixinmp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type errReadCloserCoverage struct{ err error }

func (r errReadCloserCoverage) Read([]byte) (int, error) { return 0, r.err }
func (r errReadCloserCoverage) Close() error             { return nil }

func TestClientGetAccessToken_NilHTTPClient_UsesDefaultClient(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	client := &Client{
		BaseURL:    srv.URL,
		HTTPClient: nil, // cover defaulting to http.DefaultClient
	}
	got, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err != nil {
		t.Fatalf("GetAccessToken() err = %v, want nil", err)
	}
	if got.AccessToken != "abc" {
		t.Fatalf("AccessToken = %q, want %q", got.AccessToken, "abc")
	}
}

func TestClientGetAccessToken_EmptyBaseURL_DefaultsToWeChatHost(t *testing.T) {
	t.Helper()

	client := &Client{
		BaseURL: "",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Host != "api.weixin.qq.com" {
				t.Fatalf("host = %q, want %q", r.URL.Host, "api.weixin.qq.com")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"abc","expires_in":1}`)),
			}, nil
		})},
	}

	got, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err != nil {
		t.Fatalf("GetAccessToken() err = %v, want nil", err)
	}
	if got.AccessToken != "abc" {
		t.Fatalf("AccessToken = %q, want %q", got.AccessToken, "abc")
	}
}

func TestClientGetAccessToken_InvalidBaseURL(t *testing.T) {
	t.Helper()

	client := &Client{
		BaseURL:    "https://api.weixin.qq.com/%2", // invalid escape => url.Parse() error
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, nil })},
	}

	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "invalid base url") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "invalid base url")
	}
}

func TestClientGetAccessToken_NilContext(t *testing.T) {
	t.Helper()

	client := &Client{BaseURL: "https://api.weixin.qq.com"}
	_, err := client.GetAccessToken(nil, "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "create request") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "create request")
	}
}

func TestClientGetAccessToken_DoError(t *testing.T) {
	t.Helper()

	client := &Client{
		BaseURL: "https://api.weixin.qq.com",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("boom")
		})},
	}

	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "request access token") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "request access token")
	}
}

func TestClientGetAccessToken_ReadError(t *testing.T) {
	t.Helper()

	client := &Client{
		BaseURL: "https://api.weixin.qq.com",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       errReadCloserCoverage{err: errors.New("read boom")},
			}, nil
		})},
	}

	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "read response") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "read response")
	}
}

func TestClientGetAccessToken_Non2xxStatus(t *testing.T) {
	t.Helper()

	client := &Client{
		BaseURL: "https://api.weixin.qq.com",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(strings.NewReader("  nope \n")),
			}, nil
		})},
	}

	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unexpected status") || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("err = %q, want contains %q and trimmed body %q", err.Error(), "unexpected status", "nope")
	}
}

func TestClientGetAccessToken_DecodeJSONError(t *testing.T) {
	t.Helper()

	client := &Client{
		BaseURL: "https://api.weixin.qq.com",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader("{not json")),
			}, nil
		})},
	}

	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "decode json response") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "decode json response")
	}
}

func TestClientGetAccessToken_MissingAccessToken(t *testing.T) {
	t.Helper()

	client := &Client{
		BaseURL: "https://api.weixin.qq.com",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"expires_in":7200}`)),
			}, nil
		})},
	}

	_, err := client.GetAccessToken(context.Background(), "app", "secret")
	if err == nil {
		t.Fatalf("GetAccessToken() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "missing access_token") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "missing access_token")
	}
}
