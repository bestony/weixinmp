package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	envWeixinAppID  = "WEIXINMP_APPID"
	envWeixinSecret = "WEIXINMP_SECRET"

	envTestAppID  = "APPID"
	envTestSecret = "APPSECRET"
)

func TestMain(m *testing.M) {
	if err := loadEnvForTests(); err != nil {
		// Keep it explicit and fail-fast: tests that rely on env should not
		// silently run with wrong credentials/config.
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func loadEnvForTests() error {
	root, err := findModuleRoot()
	if err != nil {
		// In normal "go test" runs this should never fail, but don't block tests.
		root = "."
	}

	path := filepath.Join(root, ".env.test")
	envs, err := parseDotEnvFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Optional: CI shouldn't require a local .env.test file.
			envs = map[string]string{}
		} else {
			return fmt.Errorf("load %s: %w", path, err)
		}
	}

	// Precedence:
	//  1) Existing env vars (eg. CI secrets)
	//  2) Keys in .env.test
	//  3) Safe defaults (unit tests use mocked HTTP server)
	appID := firstNonEmpty(
		os.Getenv(envWeixinAppID),
		envs[envWeixinAppID],
		envs[envTestAppID],
		"test-appid",
	)
	secret := firstNonEmpty(
		os.Getenv(envWeixinSecret),
		envs[envWeixinSecret],
		envs[envTestSecret],
		"test-appsecret",
	)

	if os.Getenv(envWeixinAppID) == "" {
		_ = os.Setenv(envWeixinAppID, appID)
	}
	if os.Getenv(envWeixinSecret) == "" {
		_ = os.Setenv(envWeixinSecret, secret)
	}

	return nil
}

func findModuleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
		dir = parent
	}
}

func parseDotEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	envs := map[string]string{}
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		i := strings.IndexByte(line, '=')
		if i <= 0 {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE, got %q", lineNo, line)
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNo)
		}

		// Basic quote support: KEY="value" or KEY='value'.
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
			switch val[0] {
			case '"':
				unquoted, err := strconv.Unquote(val)
				if err != nil {
					return nil, fmt.Errorf("line %d: unquote value for %s: %w", lineNo, key, err)
				}
				val = unquoted
			case '\'':
				if val[len(val)-1] != '\'' {
					return nil, fmt.Errorf("line %d: unquote value for %s: missing closing quote", lineNo, key)
				}
				// Dotenv single quotes represent a literal string (no escapes).
				val = val[1 : len(val)-1]
			}
		}

		envs[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return envs, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func TestParseDotEnvFile(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".env.test")
	if err := os.WriteFile(path, []byte(`
# comment
export APPID="app"
APPSECRET='secret'
`), 0o600); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}
	got, err := parseDotEnvFile(path)
	if err != nil {
		t.Fatalf("parseDotEnvFile() err = %v, want nil", err)
	}
	if got["APPID"] != "app" {
		t.Fatalf("APPID = %q, want %q", got["APPID"], "app")
	}
	if got["APPSECRET"] != "secret" {
		t.Fatalf("APPSECRET = %q, want %q", got["APPSECRET"], "secret")
	}
}
