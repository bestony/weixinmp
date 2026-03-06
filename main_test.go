package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
