package weixinmp

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strings"
	"testing"

	officialmessage "github.com/silenceper/wechat/v2/officialaccount/message"
)

type marshalErrorReply struct {
	officialmessage.CommonToken
}

func (*marshalErrorReply) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return errors.New("boom")
}

func TestRenderReplyXMLMarshalError(t *testing.T) {
	t.Helper()

	_, err := RenderReplyXML(officialmessage.MsgTypeText, &marshalErrorReply{}, ReplyEnvelopeOptions{
		ToUserName:   "to",
		FromUserName: "from",
		CreateTime:   1,
	})
	if err == nil || !strings.Contains(err.Error(), "marshal reply XML") {
		t.Fatalf("RenderReplyXML() err = %v, want marshal reply XML error", err)
	}
}

func TestOfficialAccountClientGetCallbackIPErrorUsesDefaultHTTPClient(t *testing.T) {
	t.Helper()

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			return jsonHTTPResponse(`{"access_token":"abc","expires_in":7200}`), nil
		case "/cgi-bin/getcallbackip":
			return jsonHTTPResponse(`{"errcode":45009,"errmsg":"api freq out of limit"}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		return nil, nil
	})
	defer func() {
		http.DefaultTransport = oldTransport
	}()

	client := &OfficialAccountClient{}
	if _, err := client.GetCallbackIP("app", "secret"); err == nil || !strings.Contains(err.Error(), "get official account callback ip") {
		t.Fatalf("GetCallbackIP() err = %v, want wrapped callback ip error", err)
	}
}
