package weixinmp

import (
	"encoding/xml"
	"strings"
	"testing"

	officialmessage "github.com/silenceper/wechat/v2/officialaccount/message"
)

func TestParseMixMessageXML(t *testing.T) {
	t.Helper()

	msg, err := ParseMixMessageXML([]byte(`
<xml>
  <ToUserName><![CDATA[gh_123]]></ToUserName>
  <FromUserName><![CDATA[user_456]]></FromUserName>
  <CreateTime>1710000000</CreateTime>
  <MsgType><![CDATA[text]]></MsgType>
  <Content><![CDATA[hello]]></Content>
  <MsgId>123456789</MsgId>
</xml>`))
	if err != nil {
		t.Fatalf("ParseMixMessageXML() err = %v, want nil", err)
	}
	if got, want := string(msg.ToUserName), "gh_123"; got != want {
		t.Fatalf("ToUserName = %q, want %q", got, want)
	}
	if got, want := string(msg.FromUserName), "user_456"; got != want {
		t.Fatalf("FromUserName = %q, want %q", got, want)
	}
	if got, want := string(msg.MsgType), "text"; got != want {
		t.Fatalf("MsgType = %q, want %q", got, want)
	}
	if got, want := msg.Content, "hello"; got != want {
		t.Fatalf("Content = %q, want %q", got, want)
	}
	if got, want := msg.MsgID, int64(123456789); got != want {
		t.Fatalf("MsgID = %d, want %d", got, want)
	}
}

func TestParseMixMessageXMLInvalid(t *testing.T) {
	t.Helper()

	_, err := ParseMixMessageXML([]byte(`<xml><MsgType>`))
	if err == nil {
		t.Fatalf("ParseMixMessageXML() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse WeChat message XML") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "parse WeChat message XML")
	}
}

func TestRenderReplyXMLText(t *testing.T) {
	t.Helper()

	data, err := RenderReplyXML(officialmessage.MsgTypeText, officialmessage.NewText("你好，世界"), ReplyEnvelopeOptions{
		ToUserName:   "user_456",
		FromUserName: "gh_123",
		CreateTime:   1710000001,
	})
	if err != nil {
		t.Fatalf("RenderReplyXML() err = %v, want nil", err)
	}

	var got officialmessage.Text
	if err := xml.Unmarshal(data, &got); err != nil {
		t.Fatalf("xml.Unmarshal(reply) err = %v, want nil", err)
	}
	if string(got.ToUserName) != "user_456" || string(got.FromUserName) != "gh_123" {
		t.Fatalf("routing = %q/%q, want %q/%q", got.ToUserName, got.FromUserName, "user_456", "gh_123")
	}
	if got.CreateTime != 1710000001 {
		t.Fatalf("CreateTime = %d, want %d", got.CreateTime, 1710000001)
	}
	if got.MsgType != officialmessage.MsgTypeText {
		t.Fatalf("MsgType = %q, want %q", got.MsgType, officialmessage.MsgTypeText)
	}
	if string(got.Content) != "你好，世界" {
		t.Fatalf("Content = %q, want %q", got.Content, "你好，世界")
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatalf("reply XML = %q, want trailing newline", string(data))
	}
}

func TestRenderReplyXMLMissingRouting(t *testing.T) {
	t.Helper()

	_, err := RenderReplyXML(officialmessage.MsgTypeText, officialmessage.NewText("hi"), ReplyEnvelopeOptions{
		CreateTime: 1710000001,
	})
	if err == nil {
		t.Fatalf("RenderReplyXML() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "missing reply routing") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "missing reply routing")
	}
}

func TestRenderReplyXMLMissingPayload(t *testing.T) {
	t.Helper()

	_, err := RenderReplyXML(officialmessage.MsgTypeText, nil, ReplyEnvelopeOptions{
		ToUserName:   "user_456",
		FromUserName: "gh_123",
		CreateTime:   1710000001,
	})
	if err == nil {
		t.Fatalf("RenderReplyXML() err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "missing reply payload") {
		t.Fatalf("err = %q, want contains %q", err.Error(), "missing reply payload")
	}
}
