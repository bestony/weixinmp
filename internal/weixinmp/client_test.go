package weixinmp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
}

