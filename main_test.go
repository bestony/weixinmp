package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kong"
)

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

func TestTokenGetCmdRunText(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got, want := q.Get("appid"), "app"; got != want {
			t.Fatalf("appid = %q, want %q", got, want)
		}
		if got, want := q.Get("secret"), "secret"; got != want {
			t.Fatalf("secret = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	cmd := &TokenGetCmd{
		AppID:   "app",
		Secret:  "secret",
		BaseURL: srv.URL,
		Timeout: 2 * time.Second,
		Output:  "text",
	}
	stdout, _ := captureStdoutStderr(t, func() {
		if err := cmd.Run(&CLI{}); err != nil {
			t.Fatalf("Run() err = %v, want nil", err)
		}
	})
	if got, want := stdout, "abc\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestTokenGetCmdRunJSON(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	cmd := &TokenGetCmd{
		AppID:   "app",
		Secret:  "secret",
		BaseURL: srv.URL,
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
	if got["access_token"] != "abc" {
		t.Fatalf("access_token = %v, want %q", got["access_token"], "abc")
	}
	if got["expires_in"] != float64(7200) {
		t.Fatalf("expires_in = %v, want %d", got["expires_in"], 7200)
	}
}

func TestTokenGetCmdRunUnsupportedOutput(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	cmd := &TokenGetCmd{
		AppID:   "app",
		Secret:  "secret",
		BaseURL: srv.URL,
		Timeout: 2 * time.Second,
		Output:  "yaml",
	}
	_, _ = captureStdoutStderr(t, func() {
		if err := cmd.Run(&CLI{}); err == nil {
			t.Fatalf("Run() err = nil, want non-nil")
		}
	})
}

func TestTokenGetCmdRunGetAccessTokenError(t *testing.T) {
	t.Helper()

	cmd := &TokenGetCmd{
		AppID:   "app",
		Secret:  "secret",
		BaseURL: "http://[::1", // invalid URL (missing ']')
		Timeout: 2 * time.Second,
		Output:  "text",
	}
	_, stderr := captureStdoutStderr(t, func() {
		if err := cmd.Run(&CLI{Debug: true}); err == nil {
			t.Fatalf("Run() err = nil, want non-nil")
		}
	})
	if !strings.Contains(stderr, "debug: requesting access token") {
		t.Fatalf("stderr = %q, want debug log", stderr)
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

func TestCLIUsesEnvVarsForTokenGet(t *testing.T) {
	t.Helper()

	// Ensure these are set via .env.test (or fallback defaults in TestMain).
	appID := os.Getenv(envWeixinAppID)
	secret := os.Getenv(envWeixinSecret)
	if appID == "" || secret == "" {
		t.Fatalf("%s/%s not set, want both non-empty", envWeixinAppID, envWeixinSecret)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("appid"); got != appID {
			t.Fatalf("appid = %q, want %q", got, appID)
		}
		if got := q.Get("secret"); got != secret {
			t.Fatalf("secret = %q, want %q", got, secret)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

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
		"token", "get",
		"--base-url", srv.URL,
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
	if got, want := stdout, "abc\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
