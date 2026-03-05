package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func buildBinary(t *testing.T, ldflags string) string {
	t.Helper()

	tmpDir := t.TempDir()
	binName := "weixinmp"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(tmpDir, binName)

	args := []string{"build", "-trimpath"}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, "-o", binPath, ".")

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}

	return binPath
}

func zipFile(t *testing.T, srcPath, zipPath string) {
	t.Helper()

	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer func() { _ = zf.Close() }()

	w := zip.NewWriter(zf)
	defer func() { _ = w.Close() }()

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		t.Fatalf("stat src: %v", err)
	}

	h, err := zip.FileInfoHeader(srcInfo)
	if err != nil {
		t.Fatalf("zip header: %v", err)
	}
	h.Name = filepath.Base(srcPath)
	h.Method = zip.Deflate

	fw, err := w.CreateHeader(h)
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		t.Fatalf("open src: %v", err)
	}
	defer func() { _ = src.Close() }()

	if _, err := io.Copy(fw, src); err != nil {
		t.Fatalf("zip write: %v", err)
	}
}

func unzipSingle(t *testing.T, zipPath, dstDir string) string {
	t.Helper()

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer func() { _ = r.Close() }()

	if len(r.File) != 1 {
		t.Fatalf("zip contains %d files, want 1", len(r.File))
	}

	f := r.File[0]
	rc, err := f.Open()
	if err != nil {
		t.Fatalf("open zipped file: %v", err)
	}
	defer func() { _ = rc.Close() }()

	dstPath := filepath.Join(dstDir, filepath.Base(f.Name))
	out, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatalf("create dst: %v", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, rc); err != nil {
		t.Fatalf("unzip copy: %v", err)
	}

	// Some unzip tools on Unix preserve mode bits; zip library doesn't always. Ensure executable.
	if runtime.GOOS != "windows" {
		_ = os.Chmod(dstPath, 0o755)
	}

	return dstPath
}

func runBin(t *testing.T, binPath string, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(binPath, args...)
	// Default to a sanitized environment so CLI invocations don't accidentally
	// inherit WEIXINMP_* from the unit-test process (which would hit the real
	// network when flags are omitted).
	cmd.Env = env
	if cmd.Env == nil {
		cmd.Env = envWithoutKeys(os.Environ(), envWeixinAppID, envWeixinSecret)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err == nil {
		return stdout, stderr, 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdout, stderr, exitErr.ExitCode()
	}
	t.Fatalf("run %q err = %v (stdout=%q stderr=%q)", strings.Join(cmd.Args, " "), err, stdout, stderr)
	return "", "", 0
}

func envWithoutKeys(base []string, keys ...string) []string {
	if len(keys) == 0 {
		return base
	}
	keySet := map[string]struct{}{}
	for _, k := range keys {
		keySet[normalizeEnvKey(k)] = struct{}{}
	}

	out := make([]string, 0, len(base))
	for _, kv := range base {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			out = append(out, kv)
			continue
		}
		k := normalizeEnvKey(kv[:i])
		if _, drop := keySet[k]; drop {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// envSet removes any existing keys from base and appends the provided KEY=VALUE entries.
func envSet(base []string, kv ...string) []string {
	if len(kv) == 0 {
		return base
	}
	keys := make([]string, 0, len(kv))
	for _, e := range kv {
		if i := strings.IndexByte(e, '='); i > 0 {
			keys = append(keys, e[:i])
		}
	}
	out := envWithoutKeys(base, keys...)
	out = append(out, kv...)
	return out
}

func normalizeEnvKey(k string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(k)
	}
	return k
}

func TestCLI_HelpFromPackagedBinary(t *testing.T) {
	t.Helper()

	// Build -> zip -> unzip -> execute. This mirrors the release pipeline enough to
	// catch accidental runtime/packaging issues (missing execute bit, wrong output, etc.).
	binPath := buildBinary(t, `-X main.version=v0.0.0 -X main.commit=deadbeef`)

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "weixinmp.zip")
	zipFile(t, binPath, zipPath)
	unzippedBin := unzipSingle(t, zipPath, tmpDir)

	stdout, stderr, code := runBin(t, unzippedBin, nil, "--help")
	if code != 0 {
		t.Fatalf("--help exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	out := stdout + stderr

	wantSubstrings := []string{
		"Usage: weixinmp <command>",
		"A small CLI toolbox for WeChat Official Account (Weixin MP).",
		"--debug",
		"--version",
		"token get",
		"Fetch access token via client_credential.",
		"signature compute",
		"Compute SHA1 signature for server validation.",
		"signature verify",
		"Verify a SHA1 signature for server validation.",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q\n%s", want, out)
		}
	}
}

func TestCLI_VersionFromPackagedBinary(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, `-X main.version=v9.9.9 -X main.commit=feedface`)
	stdout, stderr, code := runBin(t, binPath, nil, "--version")
	if code != 0 {
		t.Fatalf("--version exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	if !strings.Contains(stdout, "v9.9.9") || !strings.Contains(stdout, "feedface") {
		t.Fatalf("version output = %q, want contains version+commit", stdout)
	}
}

func TestCLI_TokenGet_JSON_NoHTMLEscape(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/token" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/cgi-bin/token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"<tag>","expires_in":7200}`))
	}))
	defer srv.Close()

	binPath := buildBinary(t, "")

	stdout, stderr, code := runBin(t, binPath, nil,
		"token", "get",
		"--app-id", "app",
		"--secret", "secret",
		"--base-url", srv.URL,
		"--output", "json",
	)
	if code != 0 {
		t.Fatalf("token get exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	if strings.Contains(stdout, "\\u003c") {
		t.Fatalf("stdout = %q, want '<' not HTML-escaped", stdout)
	}

	var decoded struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("stdout is not json: %v\n%s", err, stdout)
	}
	if decoded.AccessToken != "<tag>" {
		t.Fatalf("access_token = %q, want %q", decoded.AccessToken, "<tag>")
	}
	if decoded.ExpiresIn != 7200 {
		t.Fatalf("expires_in = %d, want %d", decoded.ExpiresIn, 7200)
	}
}

func TestCLI_TokenGet_Text(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil,
		"token", "get",
		"--app-id", "app",
		"--secret", "secret",
		"--base-url", srv.URL,
		"--output", "text",
	)
	if code != 0 {
		t.Fatalf("token get exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	if got := strings.TrimSpace(stdout); got != "abc" {
		t.Fatalf("stdout = %q, want %q", stdout, "abc\\n")
	}
}

func TestCLI_SignatureVerifyMismatch_ExitNonZero(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil,
		"signature", "verify",
		"--token", "testtoken",
		"--timestamp", "1600000000",
		"--nonce", "nonce",
		"--signature", "bad",
	)
	if code == 0 {
		t.Fatalf("exitCode=%d, want non-zero (stdout=%q stderr=%q)", code, stdout, stderr)
	}
	if !strings.Contains(stdout+stderr, "signature mismatch") {
		t.Fatalf("output = %q, want contains %q", stdout+stderr, "signature mismatch")
	}
}

func TestCLI_TokenGet_UsesEnvVars(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("appid"); got != "app" {
			t.Fatalf("appid = %q, want %q", got, "app")
		}
		if got := q.Get("secret"); got != "secret" {
			t.Fatalf("secret = %q, want %q", got, "secret")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	binPath := buildBinary(t, "")
	env := envSet(os.Environ(),
		envWeixinAppID+"=app",
		envWeixinSecret+"=secret",
	)
	stdout, stderr, code := runBin(t, binPath, env,
		"token", "get",
		"--base-url", srv.URL,
		"--output", "text",
	)
	if code != 0 {
		t.Fatalf("token get exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	if got := strings.TrimSpace(stdout); got != "abc" {
		t.Fatalf("stdout = %q, want %q", stdout, "abc\\n")
	}
}

func TestCLI_HelpForSubcommandContainsFlags(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil, "token", "get", "--help")
	if code != 0 {
		t.Fatalf("token get --help exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	out := stdout + stderr
	for _, want := range []string{
		"Usage: weixinmp token get",
		"--app-id=STRING",
		"--secret=STRING",
		"--timeout",
		"--base-url",
		"--output",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q\n%s", want, out)
		}
	}
}

func TestCLI_SignatureCompute_Output(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil,
		"signature", "compute",
		"--token", "testtoken",
		"--timestamp", "1600000000",
		"--nonce", "nonce",
	)
	if code != 0 {
		t.Fatalf("signature compute exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	if got := strings.TrimSpace(stdout); got == "" {
		t.Fatalf("stdout is empty, want signature (stderr=%q)", stderr)
	}
	// Basic sanity: sha1 hex length is 40.
	if len(strings.TrimSpace(stdout)) != 40 {
		t.Fatalf("stdout = %q, want sha1 hex digest length 40", stdout)
	}
}

func TestCLI_TokenGet_UnexpectedStatus_ShowsStatus(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	binPath := buildBinary(t, "")
	_, stderr, code := runBin(t, binPath, nil,
		"token", "get",
		"--app-id", "app",
		"--secret", "secret",
		"--base-url", srv.URL,
	)
	if code == 0 {
		t.Fatalf("exitCode=%d, want non-zero (stderr=%q)", code, stderr)
	}
	if !strings.Contains(stderr, "unexpected status") {
		t.Fatalf("stderr = %q, want contains %q", stderr, "unexpected status")
	}
}

func TestCLI_TokenGet_InvalidOutput_ShowsError(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	binPath := buildBinary(t, "")
	_, stderr, code := runBin(t, binPath, nil,
		"token", "get",
		"--app-id", "app",
		"--secret", "secret",
		"--base-url", srv.URL,
		"--output", "yaml",
	)
	if code == 0 {
		t.Fatalf("exitCode=%d, want non-zero (stderr=%q)", code, stderr)
	}
	// CLI-level parsing rejects invalid enum values before reaching runtime validation.
	if !strings.Contains(stderr, "--output") || !strings.Contains(stderr, "must be one of") {
		t.Fatalf("stderr = %q, want enum validation error mentioning --output", stderr)
	}
}

func TestCLI_TokenGet_MissingRequiredFlags(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	_, stderr, code := runBin(t, binPath, nil, "token", "get")
	if code == 0 {
		t.Fatalf("exitCode=%d, want non-zero (stderr=%q)", code, stderr)
	}
	// Kong shows field name flags on validation errors; keep this check fairly loose.
	if !strings.Contains(stderr, "--app-id") || !strings.Contains(stderr, "--secret") {
		t.Fatalf("stderr = %q, want mentions required flags", stderr)
	}
}

func TestCLI_HelpDoesNotHitNetwork(t *testing.T) {
	t.Helper()

	// A tiny canary test: help should be purely local and should not require any env vars.
	binPath := buildBinary(t, "")
	_, stderr, code := runBin(t, binPath, nil, "token", "get", "--help")
	if code != 0 {
		t.Fatalf("exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
}

func TestCLI_TokenGet_JSON_PrintsNewline(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil,
		"token", "get",
		"--app-id", "app",
		"--secret", "secret",
		"--base-url", srv.URL,
		"--output", "json",
	)
	if code != 0 {
		t.Fatalf("exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Fatalf("stdout = %q, want trailing newline", stdout)
	}
}

func TestCLI_BinaryRunsFromDifferentCWD(t *testing.T) {
	t.Helper()

	// Ensure the binary doesn't depend on being run from the repo root.
	binPath := buildBinary(t, "")
	tmpDir := t.TempDir()

	cmd := exec.Command(binPath, "--help")
	cmd.Dir = tmpDir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		t.Fatalf("run err=%v stderr=%q", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Usage: weixinmp") {
		t.Fatalf("stdout = %q, want usage", outBuf.String())
	}
}

func TestCLI_BinaryTokenGet_WithTimeoutFlag(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":7200}`))
	}))
	defer srv.Close()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil,
		"token", "get",
		"--app-id", "app",
		"--secret", "secret",
		"--base-url", srv.URL,
		"--timeout", "2s",
		"--output", "text",
	)
	if code != 0 {
		t.Fatalf("exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	if got := strings.TrimSpace(stdout); got != "abc" {
		t.Fatalf("stdout = %q, want %q", stdout, "abc\\n")
	}
}

func TestCLI_HelpMentionsHowToGetCommandHelp(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil, "--help")
	if code != 0 {
		t.Fatalf("exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	out := stdout + stderr
	want := fmt.Sprintf("Run %q for more information on a command.", `weixinmp <command> --help`)
	if !strings.Contains(out, want) {
		t.Fatalf("help output missing %q\n%s", want, out)
	}
}
