package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"awesome-cli.com/weixinmp/internal/weixinmp"
	"github.com/alecthomas/kong"
	officialmessage "github.com/silenceper/wechat/v2/officialaccount/message"
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
	Message         MessageCmd         `cmd:"" help:"WeChat message parsing and passive reply helpers."`
	OfficialAccount OfficialAccountCmd `cmd:"" help:"WeChat Official Account helpers powered by silenceper/wechat."`

	config       *Config
	configLoaded bool
	configErr    error
}

type SignatureCmd struct {
	Compute SignatureComputeCmd `cmd:"" help:"Compute SHA1 signature for server validation."`
	Verify  SignatureVerifyCmd  `cmd:"" help:"Verify a SHA1 signature for server validation."`
}

type MessageCmd struct {
	Parse MessageParseCmd `cmd:"" help:"Parse inbound WeChat XML messages from file or stdin."`
	Reply MessageReplyCmd `cmd:"" help:"Generate passive reply XML messages."`
}

type MessageParseCmd struct {
	InputFile string `help:"Path to inbound XML message. Use '-' or omit to read stdin."`
	Output    string `help:"Output format." enum:"text,json" default:"json"`
}

func (c *MessageParseCmd) Run(cli *CLI) error {
	_ = cli

	data, err := readInputSource(c.InputFile)
	if err != nil {
		return err
	}

	msg, err := weixinmp.ParseMixMessageXML(data)
	if err != nil {
		return err
	}

	switch c.Output {
	case "text":
		return writeParsedMessageText(msg)
	case "json":
		return writeJSON(msg)
	default:
		return fmt.Errorf("unsupported output format %q", c.Output)
	}
}

type MessageReplyCmd struct {
	Text                    MessageReplyTextCmd                    `cmd:"" help:"Generate passive text reply XML."`
	Image                   MessageReplyImageCmd                   `cmd:"" help:"Generate passive image reply XML."`
	Voice                   MessageReplyVoiceCmd                   `cmd:"" help:"Generate passive voice reply XML."`
	Video                   MessageReplyVideoCmd                   `cmd:"" help:"Generate passive video reply XML."`
	Music                   MessageReplyMusicCmd                   `cmd:"" help:"Generate passive music reply XML."`
	News                    MessageReplyNewsCmd                    `cmd:"" help:"Generate passive news reply XML."`
	TransferCustomerService MessageReplyTransferCustomerServiceCmd `name:"transfer-customer-service" cmd:"" help:"Generate passive transfer-to-customer-service reply XML."`
}

type MessageReplyTextCmd struct {
	RequestFile string `help:"Path to inbound XML request to auto-fill to/from. Use '-' to read stdin."`
	ToUser      string `help:"Reply ToUserName. If --request-file is set, defaults to request FromUserName."`
	FromUser    string `help:"Reply FromUserName. If --request-file is set, defaults to request ToUserName."`
	CreateTime  int64  `help:"Reply CreateTime as Unix timestamp. Defaults to current time."`

	Content string `help:"Text reply content." required:""`
}

func (c *MessageReplyTextCmd) Run(cli *CLI) error {
	_ = cli
	reply := officialmessage.NewText(c.Content)
	return writeReplyMessageXML(officialmessage.MsgTypeText, reply, c.RequestFile, c.ToUser, c.FromUser, c.CreateTime)
}

type MessageReplyImageCmd struct {
	RequestFile string `help:"Path to inbound XML request to auto-fill to/from. Use '-' to read stdin."`
	ToUser      string `help:"Reply ToUserName. If --request-file is set, defaults to request FromUserName."`
	FromUser    string `help:"Reply FromUserName. If --request-file is set, defaults to request ToUserName."`
	CreateTime  int64  `help:"Reply CreateTime as Unix timestamp. Defaults to current time."`

	MediaID string `help:"MediaId of the image reply." required:""`
}

func (c *MessageReplyImageCmd) Run(cli *CLI) error {
	_ = cli
	reply := officialmessage.NewImage(c.MediaID)
	return writeReplyMessageXML(officialmessage.MsgTypeImage, reply, c.RequestFile, c.ToUser, c.FromUser, c.CreateTime)
}

type MessageReplyVoiceCmd struct {
	RequestFile string `help:"Path to inbound XML request to auto-fill to/from. Use '-' to read stdin."`
	ToUser      string `help:"Reply ToUserName. If --request-file is set, defaults to request FromUserName."`
	FromUser    string `help:"Reply FromUserName. If --request-file is set, defaults to request ToUserName."`
	CreateTime  int64  `help:"Reply CreateTime as Unix timestamp. Defaults to current time."`

	MediaID string `help:"MediaId of the voice reply." required:""`
}

func (c *MessageReplyVoiceCmd) Run(cli *CLI) error {
	_ = cli
	reply := officialmessage.NewVoice(c.MediaID)
	return writeReplyMessageXML(officialmessage.MsgTypeVoice, reply, c.RequestFile, c.ToUser, c.FromUser, c.CreateTime)
}

type MessageReplyVideoCmd struct {
	RequestFile string `help:"Path to inbound XML request to auto-fill to/from. Use '-' to read stdin."`
	ToUser      string `help:"Reply ToUserName. If --request-file is set, defaults to request FromUserName."`
	FromUser    string `help:"Reply FromUserName. If --request-file is set, defaults to request ToUserName."`
	CreateTime  int64  `help:"Reply CreateTime as Unix timestamp. Defaults to current time."`

	MediaID     string `help:"MediaId of the video reply." required:""`
	Title       string `help:"Video title."`
	Description string `help:"Video description."`
}

func (c *MessageReplyVideoCmd) Run(cli *CLI) error {
	_ = cli
	reply := officialmessage.NewVideo(c.MediaID, c.Title, c.Description)
	return writeReplyMessageXML(officialmessage.MsgTypeVideo, reply, c.RequestFile, c.ToUser, c.FromUser, c.CreateTime)
}

type MessageReplyMusicCmd struct {
	RequestFile string `help:"Path to inbound XML request to auto-fill to/from. Use '-' to read stdin."`
	ToUser      string `help:"Reply ToUserName. If --request-file is set, defaults to request FromUserName."`
	FromUser    string `help:"Reply FromUserName. If --request-file is set, defaults to request ToUserName."`
	CreateTime  int64  `help:"Reply CreateTime as Unix timestamp. Defaults to current time."`

	Title        string `help:"Music title."`
	Description  string `help:"Music description."`
	MusicURL     string `help:"Music URL."`
	HQMusicURL   string `help:"High-quality music URL."`
	ThumbMediaID string `help:"Thumb media ID." required:""`
}

func (c *MessageReplyMusicCmd) Run(cli *CLI) error {
	_ = cli

	reply := officialmessage.NewMusic(c.Title, c.Description, c.MusicURL, c.HQMusicURL, c.ThumbMediaID)
	reply.Music.HQMusicURL = c.HQMusicURL

	return writeReplyMessageXML(officialmessage.MsgTypeMusic, reply, c.RequestFile, c.ToUser, c.FromUser, c.CreateTime)
}

type MessageReplyNewsCmd struct {
	RequestFile string `help:"Path to inbound XML request to auto-fill to/from. Use '-' to read stdin."`
	ToUser      string `help:"Reply ToUserName. If --request-file is set, defaults to request FromUserName."`
	FromUser    string `help:"Reply FromUserName. If --request-file is set, defaults to request ToUserName."`
	CreateTime  int64  `help:"Reply CreateTime as Unix timestamp. Defaults to current time."`

	Article     []string `help:"Article JSON object. Repeat the flag for multiple items."`
	ArticleFile string   `help:"Path to a JSON file containing an array of article objects."`
}

func (c *MessageReplyNewsCmd) Run(cli *CLI) error {
	_ = cli

	articles, err := loadNewsReplyArticles(c.Article, c.ArticleFile)
	if err != nil {
		return err
	}

	reply := officialmessage.NewNews(articles)
	return writeReplyMessageXML(officialmessage.MsgTypeNews, reply, c.RequestFile, c.ToUser, c.FromUser, c.CreateTime)
}

type MessageReplyTransferCustomerServiceCmd struct {
	RequestFile string `help:"Path to inbound XML request to auto-fill to/from. Use '-' to read stdin."`
	ToUser      string `help:"Reply ToUserName. If --request-file is set, defaults to request FromUserName."`
	FromUser    string `help:"Reply FromUserName. If --request-file is set, defaults to request ToUserName."`
	CreateTime  int64  `help:"Reply CreateTime as Unix timestamp. Defaults to current time."`

	KFAccount string `name:"kf-account" help:"Optional customer service account to transfer to."`
}

func (c *MessageReplyTransferCustomerServiceCmd) Run(cli *CLI) error {
	_ = cli
	reply := officialmessage.NewTransferCustomer(c.KFAccount)
	return writeReplyMessageXML(officialmessage.MsgTypeTransfer, reply, c.RequestFile, c.ToUser, c.FromUser, c.CreateTime)
}

type OfficialAccountCmd struct {
	GetCallbackIP  OfficialAccountGetCallbackIPCmd  `cmd:"" help:"Fetch WeChat callback IP addresses for the official account."`
	GetAPIDomainIP OfficialAccountGetAPIDomainIPCmd `cmd:"" help:"Fetch WeChat API domain IP addresses for the official account."`
	ClearQuota     OfficialAccountClearQuotaCmd     `cmd:"" help:"Clear official account API call quota counters."`
	Broadcast      OfficialAccountBroadcastCmd      `cmd:"" help:"Broadcast messages for the official account."`
}

type OfficialAccountBroadcastCmd struct {
	SendText  OfficialAccountBroadcastSendTextCmd  `cmd:"" help:"Broadcast text content."`
	SendNews  OfficialAccountBroadcastSendNewsCmd  `cmd:"" help:"Broadcast mpnews content by media ID."`
	SendVoice OfficialAccountBroadcastSendVoiceCmd `cmd:"" help:"Broadcast voice content by media ID."`
	SendImage OfficialAccountBroadcastSendImageCmd `cmd:"" help:"Broadcast image content by media ID list."`
	SendVideo OfficialAccountBroadcastSendVideoCmd `cmd:"" help:"Broadcast video content by media ID."`
}

type BroadcastTargetOptions struct {
	ToAll  bool   `help:"Send to all followers."`
	TagID  int64  `help:"Send to followers under a tag ID."`
	OpenID string `name:"open-id" help:"Comma-separated OpenID recipients."`
}

func (o *BroadcastTargetOptions) resolve() (*weixinmp.BroadcastUser, error) {
	openIDs := splitCSV(o.OpenID)
	targets := 0
	if o.ToAll {
		targets++
	}
	if o.TagID != 0 {
		targets++
	}
	if len(openIDs) != 0 {
		targets++
	}
	if targets != 1 {
		return nil, errors.New("choose exactly one broadcast target: --to-all, --tag-id, or --open-id")
	}
	if o.ToAll {
		return nil, nil
	}
	if o.TagID != 0 {
		return &weixinmp.BroadcastUser{TagID: o.TagID}, nil
	}
	return &weixinmp.BroadcastUser{OpenID: openIDs}, nil
}

type BroadcastCommandOptions struct {
	AppID   string        `help:"WeChat appid (overrides env/config)."`
	Secret  string        `help:"WeChat secret (overrides env/config)."`
	Timeout time.Duration `help:"HTTP timeout." default:"10s"`
	Output  string        `help:"Output format." enum:"text,json" default:"text"`

	Target BroadcastTargetOptions `embed:""`
}

func (o *BroadcastCommandOptions) resolve(cli *CLI) (string, string, *weixinmp.BroadcastUser, *weixinmp.OfficialAccountClient, error) {
	appID, secret, err := cli.resolveCredentials(o.AppID, o.Secret)
	if err != nil {
		return "", "", nil, nil, err
	}
	user, err := o.Target.resolve()
	if err != nil {
		return "", "", nil, nil, err
	}
	return appID, secret, user, newOfficialAccountClient(o.Timeout), nil
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

type OfficialAccountBroadcastSendTextCmd struct {
	Options BroadcastCommandOptions `embed:""`
	Content string                  `help:"Text content to send." required:""`
}

func (c *OfficialAccountBroadcastSendTextCmd) Run(cli *CLI) error {
	return runBroadcastCommand(cli, &c.Options, "broadcasting text via silenceper/wechat", func(appID, secret string, user *weixinmp.BroadcastUser, client *weixinmp.OfficialAccountClient) (weixinmp.BroadcastResult, error) {
		return client.BroadcastSendText(appID, secret, user, c.Content)
	})
}

type OfficialAccountBroadcastSendNewsCmd struct {
	Options       BroadcastCommandOptions `embed:""`
	MediaID       string                  `help:"Media ID of the mpnews content." required:""`
	IgnoreReprint bool                    `help:"Ignore original-content reprint protection when supported."`
}

func (c *OfficialAccountBroadcastSendNewsCmd) Run(cli *CLI) error {
	return runBroadcastCommand(cli, &c.Options, "broadcasting mpnews via silenceper/wechat", func(appID, secret string, user *weixinmp.BroadcastUser, client *weixinmp.OfficialAccountClient) (weixinmp.BroadcastResult, error) {
		return client.BroadcastSendNews(appID, secret, user, c.MediaID, c.IgnoreReprint)
	})
}

type OfficialAccountBroadcastSendVoiceCmd struct {
	Options BroadcastCommandOptions `embed:""`
	MediaID string                  `help:"Media ID of the voice content." required:""`
}

func (c *OfficialAccountBroadcastSendVoiceCmd) Run(cli *CLI) error {
	return runBroadcastCommand(cli, &c.Options, "broadcasting voice via silenceper/wechat", func(appID, secret string, user *weixinmp.BroadcastUser, client *weixinmp.OfficialAccountClient) (weixinmp.BroadcastResult, error) {
		return client.BroadcastSendVoice(appID, secret, user, c.MediaID)
	})
}

type OfficialAccountBroadcastSendImageCmd struct {
	Options            BroadcastCommandOptions `embed:""`
	MediaID            string                  `name:"media-id" help:"Comma-separated media IDs of the image content." required:""`
	Recommend          string                  `help:"Recommendation text for image broadcast."`
	NeedOpenComment    int32                   `help:"Whether comments are enabled for the image message."`
	OnlyFansCanComment int32                   `help:"Whether only followers can comment on the image message."`
}

func (c *OfficialAccountBroadcastSendImageCmd) Run(cli *CLI) error {
	image := &weixinmp.BroadcastImage{
		MediaIDs:           splitCSV(c.MediaID),
		Recommend:          c.Recommend,
		NeedOpenComment:    c.NeedOpenComment,
		OnlyFansCanComment: c.OnlyFansCanComment,
	}
	if len(image.MediaIDs) == 0 {
		return errors.New("missing image media IDs")
	}
	return runBroadcastCommand(cli, &c.Options, "broadcasting image via silenceper/wechat", func(appID, secret string, user *weixinmp.BroadcastUser, client *weixinmp.OfficialAccountClient) (weixinmp.BroadcastResult, error) {
		return client.BroadcastSendImage(appID, secret, user, image)
	})
}

type OfficialAccountBroadcastSendVideoCmd struct {
	Options     BroadcastCommandOptions `embed:""`
	MediaID     string                  `help:"Media ID of the video content." required:""`
	Title       string                  `help:"Title of the video content." required:""`
	Description string                  `help:"Description of the video content."`
}

func (c *OfficialAccountBroadcastSendVideoCmd) Run(cli *CLI) error {
	return runBroadcastCommand(cli, &c.Options, "broadcasting video via silenceper/wechat", func(appID, secret string, user *weixinmp.BroadcastUser, client *weixinmp.OfficialAccountClient) (weixinmp.BroadcastResult, error) {
		return client.BroadcastSendVideo(appID, secret, user, c.MediaID, c.Title, c.Description)
	})
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

func writeBroadcastResult(output string, result weixinmp.BroadcastResult) error {
	switch output {
	case "text":
		if result.MsgID != 0 {
			fmt.Fprintln(os.Stdout, result.MsgID)
			return nil
		}
		fmt.Fprintln(os.Stdout, "ok")
		return nil
	case "json":
		return writeJSON(result)
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}

func runBroadcastCommand(cli *CLI, options *BroadcastCommandOptions, debugMessage string, fn func(appID, secret string, user *weixinmp.BroadcastUser, client *weixinmp.OfficialAccountClient) (weixinmp.BroadcastResult, error)) error {
	appID, secret, user, client, err := options.resolve(cli)
	if err != nil {
		return err
	}
	debugf(cli, debugMessage)
	result, err := fn(appID, secret, user, client)
	if err != nil {
		return err
	}
	return writeBroadcastResult(options.Output, result)
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
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

type newsReplyArticleInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PicURL      string `json:"pic_url"`
	URL         string `json:"url"`
}

func loadNewsReplyArticles(rawArticles []string, articleFile string) ([]*officialmessage.Article, error) {
	var inputs []newsReplyArticleInput
	for _, raw := range rawArticles {
		var article newsReplyArticleInput
		if err := json.Unmarshal([]byte(raw), &article); err != nil {
			return nil, fmt.Errorf("parse --article JSON: %w", err)
		}
		inputs = append(inputs, article)
	}

	if articleFile != "" {
		data, err := os.ReadFile(articleFile)
		if err != nil {
			return nil, fmt.Errorf("read article file %s: %w", articleFile, err)
		}

		var fileArticles []newsReplyArticleInput
		if err := json.Unmarshal(data, &fileArticles); err != nil {
			return nil, fmt.Errorf("parse article file %s: %w", articleFile, err)
		}
		inputs = append(inputs, fileArticles...)
	}

	if len(inputs) == 0 {
		return nil, errors.New("missing news articles: set --article and/or --article-file")
	}

	articles := make([]*officialmessage.Article, 0, len(inputs))
	for _, input := range inputs {
		articles = append(articles, officialmessage.NewArticle(input.Title, input.Description, input.PicURL, input.URL))
	}
	return articles, nil
}

func readInputSource(path string) ([]byte, error) {
	if path == "" || path == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return data, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read input file %s: %w", path, err)
	}
	return data, nil
}

func resolveReplyEnvelope(requestFile, toUser, fromUser string, createTime int64) (weixinmp.ReplyEnvelopeOptions, error) {
	opts := weixinmp.ReplyEnvelopeOptions{
		ToUserName:   strings.TrimSpace(toUser),
		FromUserName: strings.TrimSpace(fromUser),
		CreateTime:   createTime,
	}

	if requestFile != "" {
		data, err := readInputSource(requestFile)
		if err != nil {
			return weixinmp.ReplyEnvelopeOptions{}, err
		}

		msg, err := weixinmp.ParseMixMessageXML(data)
		if err != nil {
			return weixinmp.ReplyEnvelopeOptions{}, fmt.Errorf("parse request message: %w", err)
		}

		if opts.ToUserName == "" {
			opts.ToUserName = string(msg.FromUserName)
		}
		if opts.FromUserName == "" {
			opts.FromUserName = string(msg.ToUserName)
		}
	}

	if opts.CreateTime == 0 {
		opts.CreateTime = time.Now().Unix()
	}

	return opts, nil
}

func writeReplyMessageXML(msgType officialmessage.MsgType, reply weixinmp.ReplyMessage, requestFile, toUser, fromUser string, createTime int64) error {
	opts, err := resolveReplyEnvelope(requestFile, toUser, fromUser, createTime)
	if err != nil {
		return err
	}

	data, err := weixinmp.RenderReplyXML(msgType, reply, opts)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}

func writeParsedMessageText(msg officialmessage.MixMessage) error {
	lines := []string{
		fmt.Sprintf("msg_type=%s", msg.MsgType),
		fmt.Sprintf("from_user=%s", msg.FromUserName),
		fmt.Sprintf("to_user=%s", msg.ToUserName),
		fmt.Sprintf("create_time=%d", msg.CreateTime),
	}

	if msg.Event != "" {
		lines = append(lines, fmt.Sprintf("event=%s", msg.Event))
	}
	if msg.Content != "" {
		lines = append(lines, fmt.Sprintf("content=%s", msg.Content))
	}
	if msg.MediaID != "" {
		lines = append(lines, fmt.Sprintf("media_id=%s", msg.MediaID))
	}
	if msg.EventKey != "" {
		lines = append(lines, fmt.Sprintf("event_key=%s", msg.EventKey))
	}
	if msg.MsgID != 0 {
		lines = append(lines, fmt.Sprintf("msg_id=%d", msg.MsgID))
	}

	_, err := fmt.Fprintln(os.Stdout, strings.Join(lines, "\n"))
	return err
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
