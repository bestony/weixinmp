package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"awesome-cli.com/weixinmp/internal/weixinmp"
	officialmessage "github.com/silenceper/wechat/v2/officialaccount/message"
)

func TestCoverage_ConfigNilAndEmptyHomeBranches(t *testing.T) {
	t.Helper()

	var nilCLI *CLI

	if got := nilCLI.configPath(); got != "" {
		t.Fatalf("configPath(nil) = %q, want empty", got)
	}

	cfg, err := nilCLI.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig(nil) err = %v, want nil", err)
	}
	if cfg == nil || cfg.AppID != "" || cfg.Secret != "" {
		t.Fatalf("loadConfig(nil) = %#v, want zero config", cfg)
	}

	cfg, err = nilCLI.readConfig()
	if err != nil {
		t.Fatalf("readConfig(nil) err = %v, want nil", err)
	}
	if cfg == nil || cfg.AppID != "" || cfg.Secret != "" {
		t.Fatalf("readConfig(nil) = %#v, want zero config", cfg)
	}

	t.Setenv("HOME", "")
	if got := (&CLI{}).configPath(); got != "" {
		t.Fatalf("configPath() with empty HOME = %q, want empty", got)
	}
}

func TestCoverage_ResolveCredentialsLoadConfigError(t *testing.T) {
	t.Helper()

	t.Setenv(envWeixinAppID, "")
	t.Setenv(envWeixinSecret, "")

	cli := &CLI{ConfigPath: filepath.Join(t.TempDir(), "missing.toml")}
	if _, _, err := cli.resolveCredentials("", ""); err == nil {
		t.Fatalf("resolveCredentials() err = nil, want non-nil")
	}
}

func TestCoverage_MessageParseCmdErrorPaths(t *testing.T) {
	t.Helper()

	t.Run("missing input file", func(t *testing.T) {
		cmd := &MessageParseCmd{
			InputFile: filepath.Join(t.TempDir(), "missing.xml"),
			Output:    "json",
		}
		if err := cmd.Run(&CLI{}); err == nil {
			t.Fatalf("Run() err = nil, want non-nil")
		}
	})

	t.Run("invalid xml", func(t *testing.T) {
		cmd := &MessageParseCmd{Output: "json"}
		withStdin(t, "<xml>", func() {
			if err := cmd.Run(&CLI{}); err == nil {
				t.Fatalf("Run() err = nil, want non-nil")
			}
		})
	})

	t.Run("unsupported output", func(t *testing.T) {
		cmd := &MessageParseCmd{Output: "yaml"}
		withStdin(t, `<xml>
  <ToUserName><![CDATA[to]]></ToUserName>
  <FromUserName><![CDATA[from]]></FromUserName>
  <CreateTime>1</CreateTime>
  <MsgType><![CDATA[text]]></MsgType>
  <Content><![CDATA[hello]]></Content>
</xml>`, func() {
			if err := cmd.Run(&CLI{}); err == nil || !strings.Contains(err.Error(), "unsupported output format") {
				t.Fatalf("Run() err = %v, want unsupported output format error", err)
			}
		})
	})
}

func TestCoverage_MessageReplyNewsCmdRunError(t *testing.T) {
	t.Helper()

	cmd := &MessageReplyNewsCmd{
		ToUser:   "user_456",
		FromUser: "gh_123",
		Article:  []string{"{bad json"},
	}
	if err := cmd.Run(&CLI{}); err == nil {
		t.Fatalf("Run() err = nil, want non-nil")
	}
}

func TestCoverage_LoadNewsReplyArticles(t *testing.T) {
	t.Helper()

	t.Run("invalid raw article json", func(t *testing.T) {
		if _, err := loadNewsReplyArticles([]string{"{bad json"}, ""); err == nil {
			t.Fatalf("loadNewsReplyArticles() err = nil, want non-nil")
		}
	})

	t.Run("missing article file", func(t *testing.T) {
		if _, err := loadNewsReplyArticles(nil, filepath.Join(t.TempDir(), "missing.json")); err == nil {
			t.Fatalf("loadNewsReplyArticles() err = nil, want non-nil")
		}
	})

	t.Run("invalid article file json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "articles.json")
		if err := os.WriteFile(path, []byte("{bad json"), 0o600); err != nil {
			t.Fatalf("WriteFile() err = %v, want nil", err)
		}
		if _, err := loadNewsReplyArticles(nil, path); err == nil {
			t.Fatalf("loadNewsReplyArticles() err = nil, want non-nil")
		}
	})

	t.Run("valid article file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "articles.json")
		if err := os.WriteFile(path, []byte(`[{"title":"from-file","description":"desc","pic_url":"https://example.com/p.png","url":"https://example.com"}]`), 0o600); err != nil {
			t.Fatalf("WriteFile() err = %v, want nil", err)
		}
		articles, err := loadNewsReplyArticles(nil, path)
		if err != nil {
			t.Fatalf("loadNewsReplyArticles() err = %v, want nil", err)
		}
		if len(articles) != 1 || articles[0] == nil || articles[0].Title != "from-file" {
			t.Fatalf("articles = %#v, want one parsed file article", articles)
		}
	})

	t.Run("missing all inputs", func(t *testing.T) {
		if _, err := loadNewsReplyArticles(nil, ""); err == nil {
			t.Fatalf("loadNewsReplyArticles() err = nil, want non-nil")
		}
	})
}

func TestCoverage_ReadInputSourceStdinError(t *testing.T) {
	t.Helper()

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() err = %v, want nil", err)
	}
	os.Stdin = r
	_ = r.Close()
	_ = w.Close()
	defer func() {
		os.Stdin = oldStdin
	}()

	if _, err := readInputSource(""); err == nil {
		t.Fatalf("readInputSource(\"\") err = nil, want non-nil")
	}
}

func TestCoverage_ResolveReplyEnvelopeBranches(t *testing.T) {
	t.Helper()

	t.Run("missing request file", func(t *testing.T) {
		if _, err := resolveReplyEnvelope(filepath.Join(t.TempDir(), "missing.xml"), "", "", 1); err == nil {
			t.Fatalf("resolveReplyEnvelope() err = nil, want non-nil")
		}
	})

	t.Run("invalid request xml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "request.xml")
		if err := os.WriteFile(path, []byte("<xml>"), 0o600); err != nil {
			t.Fatalf("WriteFile() err = %v, want nil", err)
		}
		if _, err := resolveReplyEnvelope(path, "", "", 1); err == nil || !strings.Contains(err.Error(), "parse request message") {
			t.Fatalf("resolveReplyEnvelope() err = %v, want parse request message error", err)
		}
	})

	t.Run("default create time", func(t *testing.T) {
		before := time.Now().Unix()
		opts, err := resolveReplyEnvelope("", " user ", " bot ", 0)
		if err != nil {
			t.Fatalf("resolveReplyEnvelope() err = %v, want nil", err)
		}
		after := time.Now().Unix()
		if opts.ToUserName != "user" || opts.FromUserName != "bot" {
			t.Fatalf("opts = %#v, want trimmed names", opts)
		}
		if opts.CreateTime < before || opts.CreateTime > after {
			t.Fatalf("CreateTime = %d, want in [%d, %d]", opts.CreateTime, before, after)
		}
	})
}

func TestCoverage_WriteReplyMessageXMLErrors(t *testing.T) {
	t.Helper()

	t.Run("resolve envelope error", func(t *testing.T) {
		reply := officialmessage.NewText("hello")
		if err := writeReplyMessageXML(officialmessage.MsgTypeText, reply, filepath.Join(t.TempDir(), "missing.xml"), "", "", 1); err == nil {
			t.Fatalf("writeReplyMessageXML() err = nil, want non-nil")
		}
	})

	t.Run("render error", func(t *testing.T) {
		if err := writeReplyMessageXML(officialmessage.MsgTypeText, nil, "", "to", "from", 1); err == nil {
			t.Fatalf("writeReplyMessageXML() err = nil, want non-nil")
		}
	})
}

func TestCoverage_WriteParsedMessageTextOptionalFields(t *testing.T) {
	t.Helper()

	stdout, _ := captureStdoutStderr(t, func() {
		err := writeParsedMessageText(officialmessage.MixMessage{
			CommonToken: officialmessage.CommonToken{
				ToUserName:   officialmessage.CDATA("to"),
				FromUserName: officialmessage.CDATA("from"),
				CreateTime:   1,
				MsgType:      officialmessage.MsgType("text"),
			},
			Content: "hello",
			MediaID: "media-1",
			MsgID:   99,
		})
		if err != nil {
			t.Fatalf("writeParsedMessageText() err = %v, want nil", err)
		}
	})

	for _, want := range []string{
		"content=hello",
		"media_id=media-1",
		"msg_id=99",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want contains %q", stdout, want)
		}
	}
}

func TestCoverage_WriteIPListAndWriteOKInvalidOutput(t *testing.T) {
	t.Helper()

	if err := writeIPList("yaml", []string{"1.1.1.1"}); err == nil {
		t.Fatalf("writeIPList() err = nil, want non-nil")
	}
	if err := writeOK("yaml"); err == nil {
		t.Fatalf("writeOK() err = nil, want non-nil")
	}
}

func TestCoverage_RunBroadcastCommandResolveError(t *testing.T) {
	t.Helper()

	options := &BroadcastCommandOptions{
		AppID:  "app",
		Secret: "secret",
		Output: "text",
		Target: BroadcastTargetOptions{
			ToAll: true,
			TagID: 1,
		},
	}
	err := runBroadcastCommand(&CLI{}, options, "debug", func(appID, secret string, user *weixinmp.BroadcastUser, client *weixinmp.OfficialAccountClient) (weixinmp.BroadcastResult, error) {
		t.Fatalf("broadcast fn should not be called")
		return weixinmp.BroadcastResult{}, nil
	})
	if err == nil {
		t.Fatalf("runBroadcastCommand() err = nil, want non-nil")
	}
}

func TestCoverage_OfficialAccountCommandResolveCredentialErrors(t *testing.T) {
	t.Helper()

	t.Setenv(envWeixinAppID, "")
	t.Setenv(envWeixinSecret, "")
	t.Setenv("HOME", t.TempDir())

	testCases := []struct {
		name string
		run  func(*CLI) error
	}{
		{
			name: "get callback ip",
			run: func(cli *CLI) error {
				return (&OfficialAccountGetCallbackIPCmd{}).Run(cli)
			},
		},
		{
			name: "get api domain ip",
			run: func(cli *CLI) error {
				return (&OfficialAccountGetAPIDomainIPCmd{}).Run(cli)
			},
		},
		{
			name: "clear quota",
			run: func(cli *CLI) error {
				return (&OfficialAccountClearQuotaCmd{}).Run(cli)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(&CLI{}); err == nil || !strings.Contains(err.Error(), "missing WeChat credentials") {
				t.Fatalf("%s err = %v, want missing credentials error", tc.name, err)
			}
		})
	}
}

func TestCoverage_OfficialAccountCommandClientErrors(t *testing.T) {
	t.Helper()

	testCases := []struct {
		name      string
		transport roundTripFunc
		run       func(*CLI) error
	}{
		{
			name: "get callback ip",
			transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				switch r.URL.Path {
				case "/cgi-bin/token":
					return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
				case "/cgi-bin/getcallbackip":
					return jsonHTTPResponse(`{"errcode":45009,"errmsg":"api freq out of limit"}`), nil
				default:
					t.Fatalf("unexpected path %q", r.URL.Path)
				}
				return nil, nil
			}),
			run: func(cli *CLI) error {
				return (&OfficialAccountGetCallbackIPCmd{
					AppID:   "app",
					Secret:  "secret",
					Timeout: time.Second,
					Output:  "text",
				}).Run(cli)
			},
		},
		{
			name: "get api domain ip",
			transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				switch r.URL.Path {
				case "/cgi-bin/token":
					return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
				case "/cgi-bin/get_api_domain_ip":
					return jsonHTTPResponse(`{"errcode":45009,"errmsg":"api freq out of limit"}`), nil
				default:
					t.Fatalf("unexpected path %q", r.URL.Path)
				}
				return nil, nil
			}),
			run: func(cli *CLI) error {
				return (&OfficialAccountGetAPIDomainIPCmd{
					AppID:   "app",
					Secret:  "secret",
					Timeout: time.Second,
					Output:  "text",
				}).Run(cli)
			},
		},
		{
			name: "clear quota",
			transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				switch r.URL.Path {
				case "/cgi-bin/token":
					return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
				case "/cgi-bin/clear_quota":
					return jsonHTTPResponse(`{"errcode":45009,"errmsg":"api freq out of limit"}`), nil
				default:
					t.Fatalf("unexpected path %q", r.URL.Path)
				}
				return nil, nil
			}),
			run: func(cli *CLI) error {
				return (&OfficialAccountClearQuotaCmd{
					AppID:   "app",
					Secret:  "secret",
					Timeout: time.Second,
					Output:  "text",
				}).Run(cli)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withDefaultTransport(t, tc.transport, func() {
				if err := tc.run(&CLI{}); err == nil {
					t.Fatalf("%s err = nil, want non-nil", tc.name)
				}
			})
		})
	}
}
