package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const (
	envWeixinAppID  = "WEIXINMP_APPID"
	envWeixinSecret = "WEIXINMP_SECRET"
)

type Config struct {
	AppID  string `toml:"app_id"`
	Secret string `toml:"secret"`
}

func (cli *CLI) configPath() string {
	if cli == nil {
		return ""
	}
	if cli.ConfigPath != "" {
		return cli.ConfigPath
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".weixinmp", "config.toml")
}

func (cli *CLI) loadConfig() (*Config, error) {
	if cli == nil {
		return &Config{}, nil
	}
	if cli.configLoaded {
		return cli.config, cli.configErr
	}
	cfg, err := cli.readConfig()
	if err != nil {
		cli.configErr = err
		cli.configLoaded = true
		return nil, err
	}

	cli.config = cfg
	cli.configLoaded = true
	return cfg, nil
}

func (cli *CLI) readConfig() (*Config, error) {
	path := cli.configPath()
	if path == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && cli.ConfigPath == "" {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}
	return &cfg, nil
}

func (cli *CLI) resolveCredentials(flagAppID, flagSecret string) (string, string, error) {
	cfg, err := cli.loadConfig()
	if err != nil {
		return "", "", err
	}

	appID := cfg.AppID
	secret := cfg.Secret

	if env := os.Getenv(envWeixinAppID); env != "" {
		appID = env
	}
	if env := os.Getenv(envWeixinSecret); env != "" {
		secret = env
	}
	if flagAppID != "" {
		appID = flagAppID
	}
	if flagSecret != "" {
		secret = flagSecret
	}

	if appID == "" || secret == "" {
		return "", "", errors.New("missing WeChat credentials: set --app-id/--secret, env vars, or a config file")
	}

	return appID, secret, nil
}
