package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"awesome-cli.com/weixinmp/internal/weixinmp"
	"github.com/alecthomas/kong"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func captureStdoutStderr(t *testing.T, fn func()) (string, string) {
	t.Helper()

	oldOut, oldErr := os.Stdout, os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdout) err = %v, want nil", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stderr) err = %v, want nil", err)
	}

	os.Stdout, os.Stderr = wOut, wErr

	outCh := make(chan []byte, 1)
	errCh := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(rOut)
		outCh <- b
	}()
	go func() {
		b, _ := io.ReadAll(rErr)
		errCh <- b
	}()

	defer func() {
		_ = wOut.Close()
		_ = wErr.Close()
		_ = rOut.Close()
		_ = rErr.Close()
		os.Stdout, os.Stderr = oldOut, oldErr
	}()

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr

	stdout := <-outCh
	stderr := <-errCh
	return string(stdout), string(stderr)
}

func TestBuildVersion(t *testing.T) {
	t.Helper()

	oldVersion, oldCommit := version, commit
	defer func() {
		version, commit = oldVersion, oldCommit
	}()

	version = "v1.2.3"

	commit = "none"
	if got, want := buildVersion(), "v1.2.3"; got != want {
		t.Fatalf("buildVersion() = %q, want %q", got, want)
	}

	commit = ""
	if got, want := buildVersion(), "v1.2.3"; got != want {
		t.Fatalf("buildVersion() = %q, want %q", got, want)
	}

	commit = "abc123"
	if got, want := buildVersion(), "v1.2.3 (abc123)"; got != want {
		t.Fatalf("buildVersion() = %q, want %q", got, want)
	}
}

func TestDebugf(t *testing.T) {
	t.Helper()

	out, errOut := captureStdoutStderr(t, func() {
		debugf(nil, "hello %s", "world")
		debugf(&CLI{Debug: false}, "hello %s", "world")
		debugf(&CLI{Debug: true}, "hello %s", "world")
	})
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
	if !strings.Contains(errOut, "debug: hello world") {
		t.Fatalf("stderr = %q, want substring %q", errOut, "debug: hello world")
	}
}

func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdin) err = %v, want nil", err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}()

	if _, err := io.WriteString(w, input); err != nil {
		t.Fatalf("WriteString(stdin) err = %v, want nil", err)
	}
	_ = w.Close()

	fn()
}

func TestSignatureComputeCmdRun(t *testing.T) {
	t.Helper()

	cmd := &SignatureComputeCmd{
		Token:     "testtoken",
		Timestamp: "1600000000",
		Nonce:     "nonce",
	}
	stdout, _ := captureStdoutStderr(t, func() {
		if err := cmd.Run(&CLI{}); err != nil {
			t.Fatalf("Run() err = %v, want nil", err)
		}
	})
	if got, want := stdout, "1282e75efd4abadbbda81cb879697196c4f90fb8\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestSignatureVerifyCmdRun(t *testing.T) {
	t.Helper()

	ok := &SignatureVerifyCmd{
		Token:     "testtoken",
		Timestamp: "1600000000",
		Nonce:     "nonce",
		Signature: "1282e75efd4abadbbda81cb879697196c4f90fb8",
	}
	if err := ok.Run(&CLI{}); err != nil {
		t.Fatalf("Run(ok) err = %v, want nil", err)
	}

	bad := &SignatureVerifyCmd{
		Token:     "testtoken",
		Timestamp: "1600000000",
		Nonce:     "nonce",
		Signature: "bad",
	}
	if err := bad.Run(&CLI{}); err == nil {
		t.Fatalf("Run(bad) err = nil, want non-nil")
	}
}

func TestMessageParseCmdRunJSON(t *testing.T) {
	t.Helper()

	cmd := &MessageParseCmd{Output: "json"}
	stdout, _ := captureStdoutStderr(t, func() {
		withStdin(t, `<xml>
  <ToUserName><![CDATA[gh_123]]></ToUserName>
  <FromUserName><![CDATA[user_456]]></FromUserName>
  <CreateTime>1710000000</CreateTime>
  <MsgType><![CDATA[text]]></MsgType>
  <Content><![CDATA[hello]]></Content>
  <MsgId>123456789</MsgId>
</xml>`, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
	})

	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal(stdout) err = %v, want nil", err)
	}
	if got["Content"] != "hello" {
		t.Fatalf("Content = %#v, want %q", got["Content"], "hello")
	}
	if got["MsgType"] != "text" {
		t.Fatalf("MsgType = %#v, want %q", got["MsgType"], "text")
	}
}

func TestMessageParseCmdRunText(t *testing.T) {
	t.Helper()

	cmd := &MessageParseCmd{Output: "text"}
	stdout, _ := captureStdoutStderr(t, func() {
		withStdin(t, `<xml>
  <ToUserName><![CDATA[gh_123]]></ToUserName>
  <FromUserName><![CDATA[user_456]]></FromUserName>
  <CreateTime>1710000000</CreateTime>
  <MsgType><![CDATA[event]]></MsgType>
  <Event><![CDATA[subscribe]]></Event>
  <EventKey><![CDATA[qrscene_42]]></EventKey>
</xml>`, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
	})

	for _, want := range []string{
		"msg_type=event",
		"from_user=user_456",
		"to_user=gh_123",
		"event=subscribe",
		"event_key=qrscene_42",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want contains %q", stdout, want)
		}
	}
}

func TestMessageReplyTextCmdRunWithRequestAutoSwap(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	requestPath := filepath.Join(dir, "request.xml")
	if err := os.WriteFile(requestPath, []byte(`<xml>
  <ToUserName><![CDATA[gh_123]]></ToUserName>
  <FromUserName><![CDATA[user_456]]></FromUserName>
  <CreateTime>1710000000</CreateTime>
  <MsgType><![CDATA[text]]></MsgType>
  <Content><![CDATA[hello]]></Content>
</xml>`), 0o600); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}

	cmd := &MessageReplyTextCmd{
		RequestFile: requestPath,
		CreateTime:  1710000001,
		Content:     "world",
	}
	stdout, _ := captureStdoutStderr(t, func() {
		if err := cmd.Run(&CLI{}); err != nil {
			t.Fatalf("Run() err = %v, want nil", err)
		}
	})

	for _, want := range []string{
		"<ToUserName><![CDATA[user_456]]></ToUserName>",
		"<FromUserName><![CDATA[gh_123]]></FromUserName>",
		"<CreateTime>1710000001</CreateTime>",
		"<MsgType>text</MsgType>",
		"<Content><![CDATA[world]]></Content>",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want contains %q", stdout, want)
		}
	}
}

func TestMessageReplyNewsCmdRun(t *testing.T) {
	t.Helper()

	cmd := &MessageReplyNewsCmd{
		ToUser:     "user_456",
		FromUser:   "gh_123",
		CreateTime: 1710000001,
		Article: []string{
			`{"title":"hello","description":"world","pic_url":"https://example.com/1.png","url":"https://example.com/1"}`,
			`{"title":"second","description":"article","pic_url":"https://example.com/2.png","url":"https://example.com/2"}`,
		},
	}
	stdout, _ := captureStdoutStderr(t, func() {
		if err := cmd.Run(&CLI{}); err != nil {
			t.Fatalf("Run() err = %v, want nil", err)
		}
	})

	for _, want := range []string{
		"<MsgType>news</MsgType>",
		"<ArticleCount>2</ArticleCount>",
		"<Title>hello</Title>",
		"<Title>second</Title>",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want contains %q", stdout, want)
		}
	}
}

func TestMessageReplyMusicCmdRun(t *testing.T) {
	t.Helper()

	cmd := &MessageReplyMusicCmd{
		ToUser:       "user_456",
		FromUser:     "gh_123",
		CreateTime:   1710000001,
		Title:        "song",
		Description:  "desc",
		MusicURL:     "https://example.com/music.mp3",
		HQMusicURL:   "https://example.com/music-hq.mp3",
		ThumbMediaID: "thumb123",
	}
	stdout, _ := captureStdoutStderr(t, func() {
		if err := cmd.Run(&CLI{}); err != nil {
			t.Fatalf("Run() err = %v, want nil", err)
		}
	})

	for _, want := range []string{
		"<MsgType>music</MsgType>",
		"<MusicUrl>https://example.com/music.mp3</MusicUrl>",
		"<HQMusicUrl>https://example.com/music-hq.mp3</HQMusicUrl>",
		"<ThumbMediaId>thumb123</ThumbMediaId>",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want contains %q", stdout, want)
		}
	}
}

func TestMessageReplyMediaCmdsRun(t *testing.T) {
	t.Helper()

	testCases := []struct {
		name string
		run  func() (string, error)
		want []string
	}{
		{
			name: "image",
			run: func() (string, error) {
				cmd := &MessageReplyImageCmd{
					ToUser:     "user_456",
					FromUser:   "gh_123",
					CreateTime: 1710000001,
					MediaID:    "img123",
				}
				var out string
				var runErr error
				out, _ = captureStdoutStderr(t, func() {
					runErr = cmd.Run(&CLI{})
				})
				return out, runErr
			},
			want: []string{"<MsgType>image</MsgType>", "<MediaId>img123</MediaId>"},
		},
		{
			name: "voice",
			run: func() (string, error) {
				cmd := &MessageReplyVoiceCmd{
					ToUser:     "user_456",
					FromUser:   "gh_123",
					CreateTime: 1710000001,
					MediaID:    "voice123",
				}
				var out string
				var runErr error
				out, _ = captureStdoutStderr(t, func() {
					runErr = cmd.Run(&CLI{})
				})
				return out, runErr
			},
			want: []string{"<MsgType>voice</MsgType>", "<MediaId>voice123</MediaId>"},
		},
		{
			name: "video",
			run: func() (string, error) {
				cmd := &MessageReplyVideoCmd{
					ToUser:      "user_456",
					FromUser:    "gh_123",
					CreateTime:  1710000001,
					MediaID:     "video123",
					Title:       "title",
					Description: "desc",
				}
				var out string
				var runErr error
				out, _ = captureStdoutStderr(t, func() {
					runErr = cmd.Run(&CLI{})
				})
				return out, runErr
			},
			want: []string{"<MsgType>video</MsgType>", "<MediaId>video123</MediaId>", "<Title>title</Title>", "<Description>desc</Description>"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, err := tc.run()
			if err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout = %q, want contains %q", stdout, want)
				}
			}
		})
	}
}

func TestMessageReplyTransferCustomerServiceCmdRun(t *testing.T) {
	t.Helper()

	cmd := &MessageReplyTransferCustomerServiceCmd{
		ToUser:     "user_456",
		FromUser:   "gh_123",
		CreateTime: 1710000001,
		KFAccount:  "test1@test",
	}
	stdout, _ := captureStdoutStderr(t, func() {
		if err := cmd.Run(&CLI{}); err != nil {
			t.Fatalf("Run() err = %v, want nil", err)
		}
	})

	for _, want := range []string{
		"<MsgType>transfer_customer_service</MsgType>",
		"<KfAccount>test1@test</KfAccount>",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want contains %q", stdout, want)
		}
	}
}

func TestOfficialAccountGetCallbackIPCmdRunText(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			q := r.URL.Query()
			if got, want := q.Get("appid"), "app"; got != want {
				t.Fatalf("appid = %q, want %q", got, want)
			}
			if got, want := q.Get("secret"), "secret"; got != want {
				t.Fatalf("secret = %q, want %q", got, want)
			}
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/getcallbackip":
			if got, want := r.URL.Query().Get("access_token"), "abc"; got != want {
				t.Fatalf("access_token = %q, want %q", got, want)
			}
			return jsonHTTPResponse(`{"ip_list":["120.1.1.1","120.1.1.2"]}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountGetCallbackIPCmd{
			AppID:   "app",
			Secret:  "secret",
			Timeout: 2 * time.Second,
			Output:  "text",
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
		if got, want := stdout, "120.1.1.1\n120.1.1.2\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestOfficialAccountGetCallbackIPCmdRunJSON(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/getcallbackip":
			return jsonHTTPResponse(`{"ip_list":["120.1.1.1","120.1.1.2"]}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountGetCallbackIPCmd{
			AppID:   "app",
			Secret:  "secret",
			Timeout: 2 * time.Second,
			Output:  "json",
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})

		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("json.Unmarshal(stdout) err = %v, want nil", err)
		}
		ipList, ok := got["ip_list"].([]any)
		if !ok {
			t.Fatalf("ip_list type = %T, want []any", got["ip_list"])
		}
		if len(ipList) != 2 || ipList[0] != "120.1.1.1" || ipList[1] != "120.1.1.2" {
			t.Fatalf("ip_list = %#v, want %#v", ipList, []any{"120.1.1.1", "120.1.1.2"})
		}
	})
}

func TestOfficialAccountGetAPIDomainIPCmdRunText(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			q := r.URL.Query()
			if got, want := q.Get("appid"), "app"; got != want {
				t.Fatalf("appid = %q, want %q", got, want)
			}
			if got, want := q.Get("secret"), "secret"; got != want {
				t.Fatalf("secret = %q, want %q", got, want)
			}
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/get_api_domain_ip":
			if got, want := r.URL.Query().Get("access_token"), "abc"; got != want {
				t.Fatalf("access_token = %q, want %q", got, want)
			}
			return jsonHTTPResponse(`{"ip_list":["1.1.1.1","2.2.2.2"]}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountGetAPIDomainIPCmd{
			AppID:   "app",
			Secret:  "secret",
			Timeout: 2 * time.Second,
			Output:  "text",
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
		if got, want := stdout, "1.1.1.1\n2.2.2.2\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestOfficialAccountGetAPIDomainIPCmdRunJSON(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/get_api_domain_ip":
			return jsonHTTPResponse(`{"ip_list":["1.1.1.1","2.2.2.2"]}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountGetAPIDomainIPCmd{
			AppID:   "app",
			Secret:  "secret",
			Timeout: 2 * time.Second,
			Output:  "json",
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})

		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("json.Unmarshal(stdout) err = %v, want nil", err)
		}
		ipList, ok := got["ip_list"].([]any)
		if !ok {
			t.Fatalf("ip_list type = %T, want []any", got["ip_list"])
		}
		if len(ipList) != 2 || ipList[0] != "1.1.1.1" || ipList[1] != "2.2.2.2" {
			t.Fatalf("ip_list = %#v, want %#v", ipList, []any{"1.1.1.1", "2.2.2.2"})
		}
	})
}

func TestOfficialAccountClearQuotaCmdRunText(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/clear_quota":
			if got, want := r.Method, http.MethodPost; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("access_token"), "abc"; got != want {
				t.Fatalf("access_token = %q, want %q", got, want)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll(body) err = %v, want nil", err)
			}
			if got, want := strings.TrimSpace(string(body)), `{"appid":"app"}`; got != want {
				t.Fatalf("body = %q, want %q", got, want)
			}
			return jsonHTTPResponse(`{"errcode":0,"errmsg":"ok"}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountClearQuotaCmd{
			AppID:   "app",
			Secret:  "secret",
			Timeout: 2 * time.Second,
			Output:  "text",
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
		if got, want := stdout, "ok\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestOfficialAccountClearQuotaCmdRunJSON(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/clear_quota":
			return jsonHTTPResponse(`{"errcode":0,"errmsg":"ok"}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountClearQuotaCmd{
			AppID:   "app",
			Secret:  "secret",
			Timeout: 2 * time.Second,
			Output:  "json",
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})

		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("json.Unmarshal(stdout) err = %v, want nil", err)
		}
		if ok, okType := got["ok"].(bool); !okType || !ok {
			t.Fatalf("ok = %#v, want true", got["ok"])
		}
	})
}

func TestBroadcastTargetOptionsResolve(t *testing.T) {
	t.Helper()

	if _, err := (&BroadcastTargetOptions{}).resolve(); err == nil {
		t.Fatalf("resolve() err = nil, want non-nil")
	}

	user, err := (&BroadcastTargetOptions{ToAll: true}).resolve()
	if err != nil {
		t.Fatalf("resolve(to-all) err = %v, want nil", err)
	}
	if user != nil {
		t.Fatalf("user = %#v, want nil for to-all", user)
	}

	user, err = (&BroadcastTargetOptions{TagID: 7}).resolve()
	if err != nil {
		t.Fatalf("resolve(tag) err = %v, want nil", err)
	}
	if user == nil || user.TagID != 7 {
		t.Fatalf("user = %#v, want tag_id=7", user)
	}

	user, err = (&BroadcastTargetOptions{OpenID: "openid-1, openid-2"}).resolve()
	if err != nil {
		t.Fatalf("resolve(openid) err = %v, want nil", err)
	}
	if user == nil || len(user.OpenID) != 2 || user.OpenID[0] != "openid-1" || user.OpenID[1] != "openid-2" {
		t.Fatalf("user = %#v, want openid list", user)
	}

	if _, err := (&BroadcastTargetOptions{ToAll: true, TagID: 1}).resolve(); err == nil {
		t.Fatalf("resolve(multiple targets) err = nil, want non-nil")
	}
}

func TestBroadcastCommandOptionsResolve(t *testing.T) {
	t.Helper()

	opts := &BroadcastCommandOptions{
		Target: BroadcastTargetOptions{ToAll: true},
	}
	if _, _, _, _, err := opts.resolve(&CLI{}); err == nil {
		t.Fatalf("resolve() err = nil, want missing credentials")
	}

	t.Setenv(envWeixinAppID, "env-app")
	t.Setenv(envWeixinSecret, "env-secret")
	opts = &BroadcastCommandOptions{
		Timeout: 3 * time.Second,
		Target:  BroadcastTargetOptions{OpenID: "openid-1"},
	}
	appID, secret, user, client, err := opts.resolve(&CLI{})
	if err != nil {
		t.Fatalf("resolve(success) err = %v, want nil", err)
	}
	if appID != "env-app" || secret != "env-secret" {
		t.Fatalf("credentials = %s/%s, want env-app/env-secret", appID, secret)
	}
	if user == nil || len(user.OpenID) != 1 || user.OpenID[0] != "openid-1" {
		t.Fatalf("user = %#v, want openid-1", user)
	}
	if client == nil || client.HTTPClient == nil || client.HTTPClient.Timeout != 3*time.Second {
		t.Fatalf("client = %#v, want timeout=3s", client)
	}
}

func TestWriteBroadcastResult(t *testing.T) {
	t.Helper()

	stdout, _ := captureStdoutStderr(t, func() {
		if err := writeBroadcastResult("text", weixinmp.BroadcastResult{MsgID: 123}); err != nil {
			t.Fatalf("writeBroadcastResult(text with msg_id) err = %v, want nil", err)
		}
	})
	if got, want := stdout, "123\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	stdout, _ = captureStdoutStderr(t, func() {
		if err := writeBroadcastResult("text", weixinmp.BroadcastResult{}); err != nil {
			t.Fatalf("writeBroadcastResult(text no msg_id) err = %v, want nil", err)
		}
	})
	if got, want := stdout, "ok\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	stdout, _ = captureStdoutStderr(t, func() {
		if err := writeBroadcastResult("json", weixinmp.BroadcastResult{MsgID: 456}); err != nil {
			t.Fatalf("writeBroadcastResult(json) err = %v, want nil", err)
		}
	})
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal(stdout) err = %v, want nil", err)
	}
	if got["msg_id"] != float64(456) {
		t.Fatalf("msg_id = %#v, want %d", got["msg_id"], 456)
	}

	if err := writeBroadcastResult("yaml", weixinmp.BroadcastResult{}); err == nil {
		t.Fatalf("writeBroadcastResult(yaml) err = nil, want non-nil")
	}
}

func TestSplitCSV(t *testing.T) {
	t.Helper()

	if got := splitCSV("  "); got != nil {
		t.Fatalf("splitCSV(empty) = %#v, want nil", got)
	}
	got := splitCSV("a, b ,, c ")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("splitCSV() = %#v, want [a b c]", got)
	}
}

func TestRunBroadcastCommandError(t *testing.T) {
	t.Helper()

	t.Setenv(envWeixinAppID, "env-app")
	t.Setenv(envWeixinSecret, "env-secret")

	err := runBroadcastCommand(&CLI{}, &BroadcastCommandOptions{
		Output: "text",
		Target: BroadcastTargetOptions{ToAll: true},
	}, "debug message", func(appID, secret string, user *weixinmp.BroadcastUser, client *weixinmp.OfficialAccountClient) (weixinmp.BroadcastResult, error) {
		return weixinmp.BroadcastResult{}, errors.New("boom")
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v, want boom", err)
	}
}

func TestOfficialAccountBroadcastSendTextCmdRunText(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/message/mass/sendall":
			payload := mustDecodeJSONRequestBody(t, r)
			if payload["msgtype"] != "text" {
				t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "text")
			}
			return jsonHTTPResponse(`{"msg_id":201}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountBroadcastSendTextCmd{
			Options: BroadcastCommandOptions{
				AppID:   "app",
				Secret:  "secret",
				Timeout: 2 * time.Second,
				Output:  "text",
				Target:  BroadcastTargetOptions{ToAll: true},
			},
			Content: "hello",
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
		if got, want := stdout, "201\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestOfficialAccountBroadcastSendNewsCmdRunJSON(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/message/mass/sendall":
			payload := mustDecodeJSONRequestBody(t, r)
			if payload["msgtype"] != "mpnews" {
				t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "mpnews")
			}
			return jsonHTTPResponse(`{"msg_id":202,"msg_data_id":302}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountBroadcastSendNewsCmd{
			Options: BroadcastCommandOptions{
				AppID:   "app",
				Secret:  "secret",
				Timeout: 2 * time.Second,
				Output:  "json",
				Target:  BroadcastTargetOptions{TagID: 2},
			},
			MediaID:       "news-media",
			IgnoreReprint: true,
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})

		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("json.Unmarshal(stdout) err = %v, want nil", err)
		}
		if got["msg_id"] != float64(202) || got["msg_data_id"] != float64(302) {
			t.Fatalf("json result = %#v, want msg_id/msg_data_id", got)
		}
	})
}

func TestOfficialAccountBroadcastSendVoiceCmdRunText(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/message/mass/send":
			payload := mustDecodeJSONRequestBody(t, r)
			if payload["msgtype"] != "voice" {
				t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "voice")
			}
			return jsonHTTPResponse(`{"msg_id":203}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountBroadcastSendVoiceCmd{
			Options: BroadcastCommandOptions{
				AppID:   "app",
				Secret:  "secret",
				Timeout: 2 * time.Second,
				Output:  "text",
				Target:  BroadcastTargetOptions{OpenID: "openid-1,openid-2"},
			},
			MediaID: "voice-media",
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
		if got, want := stdout, "203\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestOfficialAccountBroadcastSendImageCmdRunText(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/message/mass/send":
			payload := mustDecodeJSONRequestBody(t, r)
			if payload["msgtype"] != "image" {
				t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "image")
			}
			return jsonHTTPResponse(`{"msg_id":204}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountBroadcastSendImageCmd{
			Options: BroadcastCommandOptions{
				AppID:   "app",
				Secret:  "secret",
				Timeout: 2 * time.Second,
				Output:  "text",
				Target:  BroadcastTargetOptions{OpenID: "openid-1"},
			},
			MediaID:            "img-1,img-2",
			Recommend:          "nice",
			NeedOpenComment:    1,
			OnlyFansCanComment: 1,
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
		if got, want := stdout, "204\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestOfficialAccountBroadcastSendImageCmdRunMissingMedia(t *testing.T) {
	t.Helper()

	cmd := &OfficialAccountBroadcastSendImageCmd{
		Options: BroadcastCommandOptions{
			AppID:   "app",
			Secret:  "secret",
			Timeout: 2 * time.Second,
			Output:  "text",
			Target:  BroadcastTargetOptions{ToAll: true},
		},
		MediaID: " , ",
	}
	if err := cmd.Run(&CLI{}); err == nil {
		t.Fatalf("Run() err = nil, want non-nil")
	}
}

func TestOfficialAccountBroadcastSendVideoCmdRunText(t *testing.T) {
	t.Helper()

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/message/mass/send":
			payload := mustDecodeJSONRequestBody(t, r)
			if payload["msgtype"] != "mpvideo" {
				t.Fatalf("msgtype = %#v, want %q", payload["msgtype"], "mpvideo")
			}
			return jsonHTTPResponse(`{"msg_id":205}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		cmd := &OfficialAccountBroadcastSendVideoCmd{
			Options: BroadcastCommandOptions{
				AppID:   "app",
				Secret:  "secret",
				Timeout: 2 * time.Second,
				Output:  "text",
				Target:  BroadcastTargetOptions{OpenID: "openid-1"},
			},
			MediaID:     "video-media",
			Title:       "title",
			Description: "desc",
		}
		stdout, _ := captureStdoutStderr(t, func() {
			if err := cmd.Run(&CLI{}); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
		if got, want := stdout, "205\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestCLIUsesEnvVarsForOfficialAccountGetAPIDomainIP(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfigFile(t, filepath.Join(home, ".weixinmp", "config.toml"), "config-app", "config-secret")

	t.Setenv(envWeixinAppID, "env-app")
	t.Setenv(envWeixinSecret, "env-secret")

	withDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			q := r.URL.Query()
			if got := q.Get("appid"); got != "env-app" {
				t.Fatalf("appid = %q, want %q", got, "env-app")
			}
			if got := q.Get("secret"); got != "env-secret" {
				t.Fatalf("secret = %q, want %q", got, "env-secret")
			}
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/get_api_domain_ip":
			return jsonHTTPResponse(`{"ip_list":["1.1.1.1"]}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	}), func() {
		var cli CLI
		parser, err := kong.New(&cli,
			kong.Name("weixinmp"),
			kong.UsageOnError(),
			kong.Vars{"version": buildVersion()},
		)
		if err != nil {
			t.Fatalf("kong.New() err = %v, want nil", err)
		}

		ctx, err := parser.Parse([]string{
			"official-account", "get-api-domain-ip",
			"--timeout", "2s",
			"--output", "text",
		})
		if err != nil {
			t.Fatalf("Parse() err = %v, want nil", err)
		}

		stdout, _ := captureStdoutStderr(t, func() {
			if err := ctx.Run(); err != nil {
				t.Fatalf("Run() err = %v, want nil", err)
			}
		})
		if got, want := stdout, "1.1.1.1\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func withDefaultTransport(t *testing.T, transport http.RoundTripper, fn func()) {
	t.Helper()

	oldTransport := http.DefaultTransport
	http.DefaultTransport = transport
	defer func() {
		http.DefaultTransport = oldTransport
	}()

	fn()
}

func jsonHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func mustDecodeJSONRequestBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("ReadAll(body) err = %v, want nil", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json.Unmarshal(body) err = %v, want nil (body=%s)", err, string(body))
	}
	return payload
}

func TestMainRuns(t *testing.T) {
	t.Helper()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{
		"weixinmp",
		"signature",
		"compute",
		"--token=testtoken",
		"--timestamp=1600000000",
		"--nonce=nonce",
	}

	stdout, _ := captureStdoutStderr(t, func() {
		main()
	})
	if got, want := stdout, "1282e75efd4abadbbda81cb879697196c4f90fb8\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func writeConfigFile(t *testing.T, path, appID, secret string) string {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) err = %v, want nil", filepath.Dir(path), err)
	}
	content := fmt.Sprintf("app_id = %q\nsecret = %q\n", appID, secret)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) err = %v, want nil", path, err)
	}
	return path
}
