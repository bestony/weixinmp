package weixinmp

import (
	"fmt"
	"net/http"
	"sync"

	silenceperwechat "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	officialaccount "github.com/silenceper/wechat/v2/officialaccount"
	officialaccountbasic "github.com/silenceper/wechat/v2/officialaccount/basic"
	officialaccountbroadcast "github.com/silenceper/wechat/v2/officialaccount/broadcast"
	officialaccountconfig "github.com/silenceper/wechat/v2/officialaccount/config"
	wechatutil "github.com/silenceper/wechat/v2/util"
)

// IPList is the successful response from official account IP-list APIs.
type IPList struct {
	IPList []string `json:"ip_list"`
}

// CallbackIPList is the successful response from getcallbackip.
type CallbackIPList = IPList

// APIDomainIPList is the successful response from get_api_domain_ip.
type APIDomainIPList = IPList

// BroadcastUser identifies who should receive a broadcast.
type BroadcastUser = officialaccountbroadcast.User

// BroadcastImage is the image payload for image broadcasts.
type BroadcastImage = officialaccountbroadcast.Image

// BroadcastResult is the successful response from a broadcast send operation.
type BroadcastResult = officialaccountbroadcast.Result

// OfficialAccountClient talks to WeChat Official Account APIs via silenceper/wechat.
type OfficialAccountClient struct {
	HTTPClient *http.Client
}

var officialAccountSDKMu sync.Mutex

func (c *OfficialAccountClient) withOfficialAccount(appID, secret string, fn func(*officialaccount.OfficialAccount) error) error {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	officialAccountSDKMu.Lock()
	oldHTTPClient := wechatutil.DefaultHTTPClient
	wechatutil.DefaultHTTPClient = httpClient
	defer func() {
		wechatutil.DefaultHTTPClient = oldHTTPClient
		officialAccountSDKMu.Unlock()
	}()

	wc := silenceperwechat.NewWechat()
	wc.SetCache(cache.NewMemory())
	wc.SetHTTPClient(httpClient)

	officialAccount := wc.GetOfficialAccount(&officialaccountconfig.Config{
		AppID:     appID,
		AppSecret: secret,
	})

	return fn(officialAccount)
}

func (c *OfficialAccountClient) withBasic(appID, secret string, fn func(*officialaccountbasic.Basic) error) error {
	return c.withOfficialAccount(appID, secret, func(officialAccount *officialaccount.OfficialAccount) error {
		return fn(officialAccount.GetBasic())
	})
}

func (c *OfficialAccountClient) withBroadcast(appID, secret string, fn func(*officialaccountbroadcast.Broadcast) error) error {
	return c.withOfficialAccount(appID, secret, func(officialAccount *officialaccount.OfficialAccount) error {
		return fn(officialAccount.GetBroadcast())
	})
}

func (c *OfficialAccountClient) withBroadcastResult(appID, secret, action string, fn func(*officialaccountbroadcast.Broadcast) (*officialaccountbroadcast.Result, error)) (BroadcastResult, error) {
	var result *officialaccountbroadcast.Result
	err := c.withBroadcast(appID, secret, func(broadcast *officialaccountbroadcast.Broadcast) error {
		var err error
		result, err = fn(broadcast)
		if err != nil {
			return fmt.Errorf("%s: %w", action, err)
		}
		return nil
	})
	if err != nil {
		return BroadcastResult{}, err
	}
	return *result, nil
}

func (c *OfficialAccountClient) GetCallbackIP(appID, secret string) (CallbackIPList, error) {
	var ipList []string
	if err := c.withBasic(appID, secret, func(basic *officialaccountbasic.Basic) error {
		var err error
		ipList, err = basic.GetCallbackIP()
		if err != nil {
			return fmt.Errorf("get official account callback ip: %w", err)
		}
		return nil
	}); err != nil {
		return CallbackIPList{}, err
	}

	return CallbackIPList{IPList: ipList}, nil
}

func (c *OfficialAccountClient) GetAPIDomainIP(appID, secret string) (APIDomainIPList, error) {
	var ipList []string
	if err := c.withBasic(appID, secret, func(basic *officialaccountbasic.Basic) error {
		var err error
		ipList, err = basic.GetAPIDomainIP()
		if err != nil {
			return fmt.Errorf("get official account api domain ip: %w", err)
		}
		return nil
	}); err != nil {
		return APIDomainIPList{}, err
	}

	return APIDomainIPList{IPList: ipList}, nil
}

func (c *OfficialAccountClient) ClearQuota(appID, secret string) error {
	return c.withBasic(appID, secret, func(basic *officialaccountbasic.Basic) error {
		if err := basic.ClearQuota(); err != nil {
			return fmt.Errorf("clear official account quota: %w", err)
		}
		return nil
	})
}

func (c *OfficialAccountClient) BroadcastSendText(appID, secret string, user *BroadcastUser, content string) (BroadcastResult, error) {
	return c.withBroadcastResult(appID, secret, "broadcast send text", func(broadcast *officialaccountbroadcast.Broadcast) (*officialaccountbroadcast.Result, error) {
		return broadcast.SendText(user, content)
	})
}

func (c *OfficialAccountClient) BroadcastSendNews(appID, secret string, user *BroadcastUser, mediaID string, ignoreReprint bool) (BroadcastResult, error) {
	return c.withBroadcastResult(appID, secret, "broadcast send news", func(broadcast *officialaccountbroadcast.Broadcast) (*officialaccountbroadcast.Result, error) {
		return broadcast.SendNews(user, mediaID, ignoreReprint)
	})
}

func (c *OfficialAccountClient) BroadcastSendVoice(appID, secret string, user *BroadcastUser, mediaID string) (BroadcastResult, error) {
	return c.withBroadcastResult(appID, secret, "broadcast send voice", func(broadcast *officialaccountbroadcast.Broadcast) (*officialaccountbroadcast.Result, error) {
		return broadcast.SendVoice(user, mediaID)
	})
}

func (c *OfficialAccountClient) BroadcastSendImage(appID, secret string, user *BroadcastUser, image *BroadcastImage) (BroadcastResult, error) {
	return c.withBroadcastResult(appID, secret, "broadcast send image", func(broadcast *officialaccountbroadcast.Broadcast) (*officialaccountbroadcast.Result, error) {
		return broadcast.SendImage(user, image)
	})
}

func (c *OfficialAccountClient) BroadcastSendVideo(appID, secret string, user *BroadcastUser, mediaID, title, description string) (BroadcastResult, error) {
	return c.withBroadcastResult(appID, secret, "broadcast send video", func(broadcast *officialaccountbroadcast.Broadcast) (*officialaccountbroadcast.Result, error) {
		return broadcast.SendVideo(user, mediaID, title, description)
	})
}
