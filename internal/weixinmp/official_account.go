package weixinmp

import (
	"fmt"
	"net/http"
	"sync"

	silenceperwechat "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	officialaccountconfig "github.com/silenceper/wechat/v2/officialaccount/config"
	wechatutil "github.com/silenceper/wechat/v2/util"
)

// APIDomainIPList is the successful response from get_api_domain_ip.
type APIDomainIPList struct {
	IPList []string `json:"ip_list"`
}

// OfficialAccountClient talks to WeChat Official Account APIs via silenceper/wechat.
type OfficialAccountClient struct {
	HTTPClient *http.Client
}

var officialAccountSDKMu sync.Mutex

func (c *OfficialAccountClient) GetAPIDomainIP(appID, secret string) (APIDomainIPList, error) {
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

	ipList, err := officialAccount.GetBasic().GetAPIDomainIP()
	if err != nil {
		return APIDomainIPList{}, fmt.Errorf("get official account api domain ip: %w", err)
	}

	return APIDomainIPList{IPList: ipList}, nil
}
