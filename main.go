package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"awesome-cli.com/weixinmp/internal/weixinmp"
	"github.com/alecthomas/kong"
)

var (
	version = "dev"
	commit  = "none"
)

type CLI struct {
	Debug      bool             `help:"Enable debug logging."`
	Version    kong.VersionFlag `help:"Print version information and quit."`
	ConfigPath string           `name:"config" help:"Path to config file (default: ~/.weixinmp/config.toml)."`

	Signature       SignatureCmd       `cmd:"" help:"WeChat signature helpers."`
	OfficialAccount OfficialAccountCmd `cmd:"" help:"WeChat Official Account helpers powered by silenceper/wechat."`

	config       *Config
	configLoaded bool
	configErr    error
}

type SignatureCmd struct {
	Compute SignatureComputeCmd `cmd:"" help:"Compute SHA1 signature for server validation."`
	Verify  SignatureVerifyCmd  `cmd:"" help:"Verify a SHA1 signature for server validation."`
}

type OfficialAccountCmd struct {
	GetCallbackIP  OfficialAccountGetCallbackIPCmd  `cmd:"" help:"Fetch WeChat callback IP addresses for the official account."`
	GetAPIDomainIP OfficialAccountGetAPIDomainIPCmd `cmd:"" help:"Fetch WeChat API domain IP addresses for the official account."`
	ClearQuota     OfficialAccountClearQuotaCmd     `cmd:"" help:"Clear official account API call quota counters."`
}

type OfficialAccountGetCallbackIPCmd struct {
	AppID   string        `help:"WeChat appid (overrides env/config)."`
	Secret  string        `help:"WeChat secret (overrides env/config)."`
	Timeout time.Duration `help:"HTTP timeout." default:"10s"`

	Output string `help:"Output format." enum:"text,json" default:"text"`
}

func (c *OfficialAccountGetCallbackIPCmd) Run(cli *CLI) error {
	appID, secret, err := cli.resolveCredentials(c.AppID, c.Secret)
	if err != nil {
		return err
	}

	client := newOfficialAccountClient(c.Timeout)

	debugf(cli, "requesting official account callback IP via silenceper/wechat")
	ipList, err := client.GetCallbackIP(appID, secret)
	if err != nil {
		return err
	}

	return writeIPList(c.Output, ipList.IPList)
}

type OfficialAccountGetAPIDomainIPCmd struct {
	AppID   string        `help:"WeChat appid (overrides env/config)."`
	Secret  string        `help:"WeChat secret (overrides env/config)."`
	Timeout time.Duration `help:"HTTP timeout." default:"10s"`

	Output string `help:"Output format." enum:"text,json" default:"text"`
}

func (c *OfficialAccountGetAPIDomainIPCmd) Run(cli *CLI) error {
	appID, secret, err := cli.resolveCredentials(c.AppID, c.Secret)
	if err != nil {
		return err
	}

	client := newOfficialAccountClient(c.Timeout)

	debugf(cli, "requesting official account API domain IP via silenceper/wechat")
	ipList, err := client.GetAPIDomainIP(appID, secret)
	if err != nil {
		return err
	}

	return writeIPList(c.Output, ipList.IPList)
}

type OfficialAccountClearQuotaCmd struct {
	AppID   string        `help:"WeChat appid (overrides env/config)."`
	Secret  string        `help:"WeChat secret (overrides env/config)."`
	Timeout time.Duration `help:"HTTP timeout." default:"10s"`

	Output string `help:"Output format." enum:"text,json" default:"text"`
}

func (c *OfficialAccountClearQuotaCmd) Run(cli *CLI) error {
	appID, secret, err := cli.resolveCredentials(c.AppID, c.Secret)
	if err != nil {
		return err
	}

	client := newOfficialAccountClient(c.Timeout)

	debugf(cli, "clearing official account API quota via silenceper/wechat")
	if err := client.ClearQuota(appID, secret); err != nil {
		return err
	}

	return writeOK(c.Output)
}

func newOfficialAccountClient(timeout time.Duration) *weixinmp.OfficialAccountClient {
	return &weixinmp.OfficialAccountClient{
		HTTPClient: &http.Client{Timeout: timeout},
	}
}

func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func writeIPList(output string, ipList []string) error {
	switch output {
	case "text":
		for _, ip := range ipList {
			fmt.Fprintln(os.Stdout, ip)
		}
		return nil
	case "json":
		return writeJSON(struct {
			IPList []string `json:"ip_list"`
		}{IPList: ipList})
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}

type SignatureComputeCmd struct {
	Token     string `help:"WeChat token (the one configured in MP settings)." required:""`
	Timestamp string `help:"Timestamp from the request." required:""`
	Nonce     string `help:"Nonce from the request." required:""`
}

func (c *SignatureComputeCmd) Run(cli *CLI) error {
	_ = cli
	fmt.Fprintln(os.Stdout, weixinmp.Signature(c.Token, c.Timestamp, c.Nonce))
	return nil
}

type SignatureVerifyCmd struct {
	Token     string `help:"WeChat token (the one configured in MP settings)." required:""`
	Timestamp string `help:"Timestamp from the request." required:""`
	Nonce     string `help:"Nonce from the request." required:""`
	Signature string `help:"Signature from the request." required:""`
}

func (v *SignatureVerifyCmd) Run(cli *CLI) error {
	_ = cli
	if weixinmp.VerifySignature(v.Signature, v.Token, v.Timestamp, v.Nonce) {
		return nil
	}
	return errors.New("signature mismatch")
}

func main() {
	var cli CLI

	ctx := kong.Parse(&cli,
		kong.Name("weixinmp"),
		kong.Description("A small CLI toolbox for WeChat Official Account (Weixin MP)."),
		kong.UsageOnError(),
		kong.Vars{"version": buildVersion()},
	)

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

func buildVersion() string {
	if commit == "none" || commit == "" {
		return version
	}
	return fmt.Sprintf("%s (%s)", version, commit)
}

func debugf(cli *CLI, format string, args ...any) {
	if cli == nil || !cli.Debug {
		return
	}
	fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
}

func writeOK(output string) error {
	switch output {
	case "text":
		fmt.Fprintln(os.Stdout, "ok")
		return nil
	case "json":
		return writeJSON(struct {
			OK bool `json:"ok"`
		}{OK: true})
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}
