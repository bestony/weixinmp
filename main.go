package main

import (
	"context"
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
	Debug   bool             `help:"Enable debug logging."`
	Version kong.VersionFlag `help:"Print version information and quit."`

	Token           TokenCmd           `cmd:"" help:"WeChat Official Account access token helpers."`
	Signature       SignatureCmd       `cmd:"" help:"WeChat signature helpers."`
	OfficialAccount OfficialAccountCmd `cmd:"" help:"WeChat Official Account helpers powered by silenceper/wechat."`
}

type TokenCmd struct {
	Get TokenGetCmd `cmd:"" help:"Fetch access token via client_credential."`
}

type TokenGetCmd struct {
	AppID   string        `help:"WeChat appid." env:"WEIXINMP_APPID" required:""`
	Secret  string        `help:"WeChat secret." env:"WEIXINMP_SECRET" required:""`
	Timeout time.Duration `help:"HTTP timeout." default:"10s"`
	BaseURL string        `help:"WeChat API base URL." default:"https://api.weixin.qq.com"`

	Output string `help:"Output format." enum:"text,json" default:"text"`
}

func (c *TokenGetCmd) Run(cli *CLI) error {
	httpClient := &http.Client{Timeout: c.Timeout}
	client := &weixinmp.Client{
		BaseURL:    c.BaseURL,
		HTTPClient: httpClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	debugf(cli, "requesting access token from %s", c.BaseURL)
	token, err := client.GetAccessToken(ctx, c.AppID, c.Secret)
	if err != nil {
		return err
	}

	switch c.Output {
	case "text":
		fmt.Fprintln(os.Stdout, token.AccessToken)
		return nil
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		return enc.Encode(token)
	default:
		return fmt.Errorf("unsupported output format %q", c.Output)
	}
}

type SignatureCmd struct {
	Compute SignatureComputeCmd `cmd:"" help:"Compute SHA1 signature for server validation."`
	Verify  SignatureVerifyCmd  `cmd:"" help:"Verify a SHA1 signature for server validation."`
}

type OfficialAccountCmd struct {
	GetAPIDomainIP OfficialAccountGetAPIDomainIPCmd `cmd:"" help:"Fetch WeChat API domain IP addresses for the official account."`
}

type OfficialAccountGetAPIDomainIPCmd struct {
	AppID   string        `help:"WeChat appid." env:"WEIXINMP_APPID" required:""`
	Secret  string        `help:"WeChat secret." env:"WEIXINMP_SECRET" required:""`
	Timeout time.Duration `help:"HTTP timeout." default:"10s"`

	Output string `help:"Output format." enum:"text,json" default:"text"`
}

func (c *OfficialAccountGetAPIDomainIPCmd) Run(cli *CLI) error {
	httpClient := &http.Client{Timeout: c.Timeout}
	client := &weixinmp.OfficialAccountClient{
		HTTPClient: httpClient,
	}

	debugf(cli, "requesting official account API domain IP via silenceper/wechat")
	ipList, err := client.GetAPIDomainIP(c.AppID, c.Secret)
	if err != nil {
		return err
	}

	switch c.Output {
	case "text":
		for _, ip := range ipList.IPList {
			fmt.Fprintln(os.Stdout, ip)
		}
		return nil
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		return enc.Encode(ipList)
	default:
		return fmt.Errorf("unsupported output format %q", c.Output)
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
