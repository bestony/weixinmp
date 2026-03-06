package weixinmp

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	officialmessage "github.com/silenceper/wechat/v2/officialaccount/message"
)

type ReplyEnvelopeOptions struct {
	ToUserName   string
	FromUserName string
	CreateTime   int64
}

type ReplyMessage interface {
	SetToUserName(officialmessage.CDATA)
	SetFromUserName(officialmessage.CDATA)
	SetCreateTime(int64)
	SetMsgType(officialmessage.MsgType)
}

func ParseMixMessageXML(data []byte) (officialmessage.MixMessage, error) {
	var msg officialmessage.MixMessage
	if err := xml.Unmarshal(data, &msg); err != nil {
		return officialmessage.MixMessage{}, fmt.Errorf("parse WeChat message XML: %w", err)
	}
	return msg, nil
}

func RenderReplyXML(msgType officialmessage.MsgType, payload ReplyMessage, opts ReplyEnvelopeOptions) ([]byte, error) {
	if payload == nil {
		return nil, errors.New("missing reply payload")
	}

	toUserName := strings.TrimSpace(opts.ToUserName)
	fromUserName := strings.TrimSpace(opts.FromUserName)
	if toUserName == "" || fromUserName == "" {
		return nil, errors.New("missing reply routing: set --to-user/--from-user or provide --request-file")
	}

	payload.SetToUserName(officialmessage.CDATA(toUserName))
	payload.SetFromUserName(officialmessage.CDATA(fromUserName))
	payload.SetCreateTime(opts.CreateTime)
	payload.SetMsgType(msgType)

	data, err := xml.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal reply XML: %w", err)
	}

	return append(data, '\n'), nil
}
