package weixinmp

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOfficialAccountClientGetCallbackIPOK(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				q := r.URL.Query()
				if got := q.Get("appid"); got != "app" {
					t.Fatalf("appid = %q, want %q", got, "app")
				}
				if got := q.Get("secret"); got != "secret" {
					t.Fatalf("secret = %q, want %q", got, "secret")
				}
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/getcallbackip":
				if got := r.URL.Query().Get("access_token"); got != "abc" {
					t.Fatalf("access_token = %q, want %q", got, "abc")
				}
				return jsonHTTPResponse(`{"ip_list":["101.1.1.1","101.1.1.2"]}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	got, err := client.GetCallbackIP("app", "secret")
	if err != nil {
		t.Fatalf("GetCallbackIP() err = %v, want nil", err)
	}
	if len(got.IPList) != 2 {
		t.Fatalf("len(IPList) = %d, want %d", len(got.IPList), 2)
	}
	if got.IPList[0] != "101.1.1.1" || got.IPList[1] != "101.1.1.2" {
		t.Fatalf("IPList = %#v, want %#v", got.IPList, []string{"101.1.1.1", "101.1.1.2"})
	}
}

func TestOfficialAccountClientGetAPIDomainIPOK(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				q := r.URL.Query()
				if got := q.Get("appid"); got != "app" {
					t.Fatalf("appid = %q, want %q", got, "app")
				}
				if got := q.Get("secret"); got != "secret" {
					t.Fatalf("secret = %q, want %q", got, "secret")
				}
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/get_api_domain_ip":
				if got := r.URL.Query().Get("access_token"); got != "abc" {
					t.Fatalf("access_token = %q, want %q", got, "abc")
				}
				return jsonHTTPResponse(`{"ip_list":["1.1.1.1","2.2.2.2"]}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	got, err := client.GetAPIDomainIP("app", "secret")
	if err != nil {
		t.Fatalf("GetAPIDomainIP() err = %v, want nil", err)
	}
	if len(got.IPList) != 2 {
		t.Fatalf("len(IPList) = %d, want %d", len(got.IPList), 2)
	}
	if got.IPList[0] != "1.1.1.1" || got.IPList[1] != "2.2.2.2" {
		t.Fatalf("IPList = %#v, want %#v", got.IPList, []string{"1.1.1.1", "2.2.2.2"})
	}
}

func TestOfficialAccountClientGetAPIDomainIPAccessTokenError(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/cgi-bin/token" {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/cgi-bin/token")
			}
			return jsonHTTPResponse(`{"errcode":40013,"errmsg":"invalid appid"}`), nil
		})},
	}

	_, err := client.GetAPIDomainIP("bad", "secret")
	if err == nil {
		t.Fatalf("GetAPIDomainIP() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "get access_token error") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "get access_token error")
	}
}

func TestOfficialAccountClientGetAPIDomainIPAPIError(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/get_api_domain_ip":
				return jsonHTTPResponse(`{"errcode":45009,"errmsg":"api freq out of limit"}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	_, err := client.GetAPIDomainIP("app", "secret")
	if err == nil {
		t.Fatalf("GetAPIDomainIP() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "GetAPIDomainIP Error") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "GetAPIDomainIP Error")
	}
}

func TestOfficialAccountClientGetAPIDomainIPTransportError(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/get_api_domain_ip":
				return nil, errors.New("boom")
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	_, err := client.GetAPIDomainIP("app", "secret")
	if err == nil {
		t.Fatalf("GetAPIDomainIP() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "boom")
	}
}

func TestOfficialAccountClientClearQuotaOK(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/clear_quota":
				if got := r.Method; got != http.MethodPost {
					t.Fatalf("method = %q, want %q", got, http.MethodPost)
				}
				if got := r.URL.Query().Get("access_token"); got != "abc" {
					t.Fatalf("access_token = %q, want %q", got, "abc")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("ReadAll(body) err = %v, want nil", err)
				}
				if got := strings.TrimSpace(string(body)); got != `{"appid":"app"}` {
					t.Fatalf("body = %q, want %q", got, `{"appid":"app"}`)
				}
				return jsonHTTPResponse(`{"errcode":0,"errmsg":"ok"}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	if err := client.ClearQuota("app", "secret"); err != nil {
		t.Fatalf("ClearQuota() err = %v, want nil", err)
	}
}

func TestOfficialAccountClientClearQuotaAPIError(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/clear_quota":
				return jsonHTTPResponse(`{"errcode":45009,"errmsg":"api freq out of limit"}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	err := client.ClearQuota("app", "secret")
	if err == nil {
		t.Fatalf("ClearQuota() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "ClearQuota Error") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "ClearQuota Error")
	}
}

func jsonHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
