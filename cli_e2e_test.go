package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
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
		"message parse",
		"Parse inbound WeChat XML messages from file or stdin.",
		"message reply text",
		"Generate passive text reply XML.",
		"official-account get-callback-ip",
		"Fetch WeChat callback IP addresses for the official account.",
		"official-account clear-quota",
		"Clear official account API call quota counters.",
		"official-account broadcast send-text",
		"Broadcast text content.",
		"official-account get-api-domain-ip",
		"Fetch WeChat API domain IP addresses for the official account.",
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

func TestCLI_HelpForOfficialAccountSubcommandContainsFlags(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil, "official-account", "get-api-domain-ip", "--help")
	if code != 0 {
		t.Fatalf("official-account get-api-domain-ip --help exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	out := stdout + stderr
	for _, want := range []string{
		"Usage: weixinmp official-account get-api-domain-ip",
		"--app-id=STRING",
		"--secret=STRING",
		"--timeout",
		"--output",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q\n%s", want, out)
		}
	}
}

func TestCLI_HelpForMessageParseContainsFlags(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil, "message", "parse", "--help")
	if code != 0 {
		t.Fatalf("message parse --help exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	out := stdout + stderr
	for _, want := range []string{
		"Usage: weixinmp message parse",
		"--input-file=STRING",
		"--output",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q\n%s", want, out)
		}
	}
}

func TestCLI_HelpForMessageReplyTextContainsFlags(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil, "message", "reply", "text", "--help")
	if code != 0 {
		t.Fatalf("message reply text --help exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	out := stdout + stderr
	for _, want := range []string{
		"Usage: weixinmp message reply text",
		"--request-file=STRING",
		"--to-user=STRING",
		"--from-user=STRING",
		"--content=STRING",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q\n%s", want, out)
		}
	}
}

func TestCLI_HelpForOfficialAccountGetCallbackIPContainsFlags(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil, "official-account", "get-callback-ip", "--help")
	if code != 0 {
		t.Fatalf("official-account get-callback-ip --help exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	out := stdout + stderr
	for _, want := range []string{
		"Usage: weixinmp official-account get-callback-ip",
		"--app-id=STRING",
		"--secret=STRING",
		"--timeout",
		"--output",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q\n%s", want, out)
		}
	}
}

func TestCLI_MessageParse_JSON(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "message.xml")
	if err := os.WriteFile(inputPath, []byte(`<xml>
  <ToUserName><![CDATA[gh_123]]></ToUserName>
  <FromUserName><![CDATA[user_456]]></FromUserName>
  <CreateTime>1710000000</CreateTime>
  <MsgType><![CDATA[text]]></MsgType>
  <Content><![CDATA[hello]]></Content>
</xml>`), 0o600); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}

	stdout, stderr, code := runBin(t, binPath, nil,
		"message", "parse",
		"--input-file", inputPath,
		"--output", "json",
	)
	if code != 0 {
		t.Fatalf("message parse exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	if !strings.Contains(stdout, `"Content":"hello"`) {
		t.Fatalf("stdout = %q, want contains %q", stdout, `"Content":"hello"`)
	}
}

func TestCLI_MessageReplyText_WithRequestFile(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
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

	stdout, stderr, code := runBin(t, binPath, nil,
		"message", "reply", "text",
		"--request-file", requestPath,
		"--create-time", "1710000001",
		"--content", "world",
	)
	if code != 0 {
		t.Fatalf("message reply text exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	for _, want := range []string{
		"<ToUserName><![CDATA[user_456]]></ToUserName>",
		"<FromUserName><![CDATA[gh_123]]></FromUserName>",
		"<Content><![CDATA[world]]></Content>",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want contains %q", stdout, want)
		}
	}
}

func TestCLI_HelpForOfficialAccountClearQuotaContainsFlags(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil, "official-account", "clear-quota", "--help")
	if code != 0 {
		t.Fatalf("official-account clear-quota --help exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	out := stdout + stderr
	for _, want := range []string{
		"Usage: weixinmp official-account clear-quota",
		"--app-id=STRING",
		"--secret=STRING",
		"--timeout",
		"--output",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q\n%s", want, out)
		}
	}
}

func TestCLI_HelpForOfficialAccountBroadcastSendTextContainsFlags(t *testing.T) {
	t.Helper()

	binPath := buildBinary(t, "")
	stdout, stderr, code := runBin(t, binPath, nil, "official-account", "broadcast", "send-text", "--help")
	if code != 0 {
		t.Fatalf("official-account broadcast send-text --help exitCode=%d, want 0 (stderr=%q)", code, stderr)
	}
	out := stdout + stderr
	for _, want := range []string{
		"Usage: weixinmp official-account broadcast send-text",
		"--app-id=STRING",
		"--secret=STRING",
		"--timeout",
		"--output",
		"--to-all",
		"--tag-id=INT",
		"--open-id=STRING",
		"--content=STRING",
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

func TestCLI_HelpDoesNotHitNetwork(t *testing.T) {
	t.Helper()

	// A tiny canary test: help should be purely local and should not require any env vars.
	binPath := buildBinary(t, "")
	_, stderr, code := runBin(t, binPath, nil, "official-account", "get-api-domain-ip", "--help")
	if code != 0 {
		t.Fatalf("exitCode=%d, want 0 (stderr=%q)", code, stderr)
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
