package weixinmp

import (
	"fmt"
	"net/http"
	"sync"

	silenceperwechat "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	officialaccountbasic "github.com/silenceper/wechat/v2/officialaccount/basic"
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

// OfficialAccountClient talks to WeChat Official Account APIs via silenceper/wechat.
type OfficialAccountClient struct {
	HTTPClient *http.Client
}

var officialAccountSDKMu sync.Mutex

func (c *OfficialAccountClient) withBasic(appID, secret string, fn func(*officialaccountbasic.Basic) error) error {
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

	return fn(officialAccount.GetBasic())
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
