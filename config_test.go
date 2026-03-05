package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaultMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cli := &CLI{}
	cfg, err := cli.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() err = %v, want nil", err)
	}
	if cfg.AppID != "" || cfg.Secret != "" {
		t.Fatalf("cfg = %#v, want zero values", cfg)
	}
}

func TestLoadConfigExplicitMissing(t *testing.T) {
	tmp := t.TempDir()
	cli := &CLI{ConfigPath: filepath.Join(tmp, "config.toml")}
	if _, err := cli.loadConfig(); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("loadConfig() err = %v, want wrapped os.ErrNotExist", err)
	}
}

func TestLoadConfigMalformed(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".weixinmp")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() err = %v, want nil", err)
	}
	path := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(path, []byte("not toml"), 0o600); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}
	t.Setenv("HOME", tmp)

	cli := &CLI{}
	if _, err := cli.loadConfig(); err == nil {
		t.Fatalf("loadConfig() err = nil, want non-nil")
	}
}

func TestResolveCredentialsPrecedence(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.toml")
	if err := os.WriteFile(configPath, []byte("app_id=\"config-app\"\nsecret=\"config-secret\""), 0o600); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}

	cli := &CLI{ConfigPath: configPath}

	appID, secret, err := cli.resolveCredentials("", "")
	if err != nil {
		t.Fatalf("resolveCredentials(config) err = %v, want nil", err)
	}
	if appID != "config-app" || secret != "config-secret" {
		t.Fatalf("config values = %s/%s, want config-app/config-secret", appID, secret)
	}

	t.Setenv(envWeixinAppID, "env-app")
	t.Setenv(envWeixinSecret, "env-secret")
	appID, secret, err = cli.resolveCredentials("", "")
	if err != nil {
		t.Fatalf("resolveCredentials(env) err = %v, want nil", err)
	}
	if appID != "env-app" || secret != "env-secret" {
		t.Fatalf("env values = %s/%s, want env-app/env-secret", appID, secret)
	}

	appID, secret, err = cli.resolveCredentials("flag-app", "flag-secret")
	if err != nil {
		t.Fatalf("resolveCredentials(flags) err = %v, want nil", err)
	}
	if appID != "flag-app" || secret != "flag-secret" {
		t.Fatalf("flag values = %s/%s, want flag-app/flag-secret", appID, secret)
	}
}

func TestResolveCredentialsMissing(t *testing.T) {
	cli := &CLI{}
	if _, _, err := cli.resolveCredentials("", ""); err == nil {
		t.Fatalf("resolveCredentials() err = nil, want non-nil")
	}
}
