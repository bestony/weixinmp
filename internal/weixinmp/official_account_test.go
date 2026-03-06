package weixinmp

import (
	"encoding/json"
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

func TestOfficialAccountClientBroadcastSendTextOK(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/message/mass/sendall":
				var payload map[string]any
				mustDecodeJSONBody(t, r, &payload)
				if payload["msgtype"] != "text" {
					t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "text")
				}
				filter, ok := payload["filter"].(map[string]any)
				if !ok || filter["is_to_all"] != true {
					t.Fatalf("filter = %#v, want is_to_all=true", payload["filter"])
				}
				text, ok := payload["text"].(map[string]any)
				if !ok || text["content"] != "hello" {
					t.Fatalf("text = %#v, want content=hello", payload["text"])
				}
				return jsonHTTPResponse(`{"msg_id":101,"msg_data_id":202,"msg_status":"send success"}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	got, err := client.BroadcastSendText("app", "secret", nil, "hello")
	if err != nil {
		t.Fatalf("BroadcastSendText() err = %v, want nil", err)
	}
	if got.MsgID != 101 || got.MsgDataID != 202 || got.MsgStatus != "send success" {
		t.Fatalf("result = %#v, want msg_id/msg_data_id/msg_status populated", got)
	}
}

func TestOfficialAccountClientBroadcastSendNewsOK(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/message/mass/sendall":
				var payload map[string]any
				mustDecodeJSONBody(t, r, &payload)
				if payload["msgtype"] != "mpnews" {
					t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "mpnews")
				}
				filter, ok := payload["filter"].(map[string]any)
				if !ok || filter["is_to_all"] != false || filter["tag_id"] != float64(2) {
					t.Fatalf("filter = %#v, want tag_id=2", payload["filter"])
				}
				mpnews, ok := payload["mpnews"].(map[string]any)
				if !ok || mpnews["media_id"] != "news-media" {
					t.Fatalf("mpnews = %#v, want media_id=news-media", payload["mpnews"])
				}
				if payload["send_ignore_reprint"] != float64(1) {
					t.Fatalf("send_ignore_reprint = %#v, want 1", payload["send_ignore_reprint"])
				}
				return jsonHTTPResponse(`{"msg_id":102}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	got, err := client.BroadcastSendNews("app", "secret", &BroadcastUser{TagID: 2}, "news-media", true)
	if err != nil {
		t.Fatalf("BroadcastSendNews() err = %v, want nil", err)
	}
	if got.MsgID != 102 {
		t.Fatalf("MsgID = %d, want %d", got.MsgID, 102)
	}
}

func TestOfficialAccountClientBroadcastSendVoiceOK(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/message/mass/send":
				var payload map[string]any
				mustDecodeJSONBody(t, r, &payload)
				if payload["msgtype"] != "voice" {
					t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "voice")
				}
				toUser, ok := payload["touser"].([]any)
				if !ok || len(toUser) != 2 || toUser[0] != "openid-1" || toUser[1] != "openid-2" {
					t.Fatalf("touser = %#v, want openid list", payload["touser"])
				}
				voice, ok := payload["voice"].(map[string]any)
				if !ok || voice["media_id"] != "voice-media" {
					t.Fatalf("voice = %#v, want media_id=voice-media", payload["voice"])
				}
				return jsonHTTPResponse(`{"msg_id":103}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	got, err := client.BroadcastSendVoice("app", "secret", &BroadcastUser{OpenID: []string{"openid-1", "openid-2"}}, "voice-media")
	if err != nil {
		t.Fatalf("BroadcastSendVoice() err = %v, want nil", err)
	}
	if got.MsgID != 103 {
		t.Fatalf("MsgID = %d, want %d", got.MsgID, 103)
	}
}

func TestOfficialAccountClientBroadcastSendImageOK(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/message/mass/send":
				var payload map[string]any
				mustDecodeJSONBody(t, r, &payload)
				if payload["msgtype"] != "image" {
					t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "image")
				}
				images, ok := payload["images"].(map[string]any)
				if !ok {
					t.Fatalf("images = %#v, want object", payload["images"])
				}
				mediaIDs, ok := images["media_ids"].([]any)
				if !ok || len(mediaIDs) != 2 || mediaIDs[0] != "img-1" || mediaIDs[1] != "img-2" {
					t.Fatalf("media_ids = %#v, want [img-1 img-2]", images["media_ids"])
				}
				if images["recommend"] != "nice" || images["need_open_comment"] != float64(1) || images["only_fans_can_comment"] != float64(1) {
					t.Fatalf("images = %#v, want recommend/comment fields", images)
				}
				return jsonHTTPResponse(`{"msg_id":104}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	got, err := client.BroadcastSendImage("app", "secret", &BroadcastUser{OpenID: []string{"openid-1"}}, &BroadcastImage{
		MediaIDs:           []string{"img-1", "img-2"},
		Recommend:          "nice",
		NeedOpenComment:    1,
		OnlyFansCanComment: 1,
	})
	if err != nil {
		t.Fatalf("BroadcastSendImage() err = %v, want nil", err)
	}
	if got.MsgID != 104 {
		t.Fatalf("MsgID = %d, want %d", got.MsgID, 104)
	}
}

func TestOfficialAccountClientBroadcastSendVideoOK(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/message/mass/send":
				var payload map[string]any
				mustDecodeJSONBody(t, r, &payload)
				if payload["msgtype"] != "mpvideo" {
					t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "mpvideo")
				}
				mpvideo, ok := payload["mpvideo"].(map[string]any)
				if !ok || mpvideo["media_id"] != "video-media" || mpvideo["title"] != "title" || mpvideo["description"] != "desc" {
					t.Fatalf("mpvideo = %#v, want media_id/title/description", payload["mpvideo"])
				}
				return jsonHTTPResponse(`{"msg_id":105}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	got, err := client.BroadcastSendVideo("app", "secret", &BroadcastUser{OpenID: []string{"openid-1"}}, "video-media", "title", "desc")
	if err != nil {
		t.Fatalf("BroadcastSendVideo() err = %v, want nil", err)
	}
	if got.MsgID != 105 {
		t.Fatalf("MsgID = %d, want %d", got.MsgID, 105)
	}
}

func TestOfficialAccountClientBroadcastSendTextAPIError(t *testing.T) {
	t.Helper()

	client := &OfficialAccountClient{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/cgi-bin/token":
				return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
			case "/cgi-bin/message/mass/sendall":
				return jsonHTTPResponse(`{"errcode":45009,"errmsg":"api freq out of limit"}`), nil
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			return nil, nil
		})},
	}

	_, err := client.BroadcastSendText("app", "secret", nil, "hello")
	if err == nil {
		t.Fatalf("BroadcastSendText() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "broadcast send text") || !strings.Contains(err.Error(), "SendText Error") {
		t.Fatalf("err = %q, want wrapped broadcast/SDK error", err.Error())
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

func mustDecodeJSONBody(t *testing.T, r *http.Request, dst any) {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("ReadAll(body) err = %v, want nil", err)
	}
	if err := json.Unmarshal(body, dst); err != nil {
		t.Fatalf("json.Unmarshal(body) err = %v, want nil (body=%s)", err, string(body))
	}
}
