# 公众号消息管理 CLI 对照

本文档对照 `silenceper/wechat` 的公众号消息管理说明，整理 `weixinmp` 当前已经封装到 CLI 的消息相关能力。

参考资料：

- `silenceper/wechat` 公众号消息管理文档：<https://silenceper.com/wechat/officialaccount/message.html>
- SDK 对应目录：<https://github.com/silenceper/wechat/tree/master/officialaccount/message>

## 已封装能力

| 文档能力 | CLI 命令 | 说明 |
| --- | --- | --- |
| 接收普通消息 / 事件推送 | `weixinmp message parse` | 从 XML 解析出统一的消息结构 |
| 被动回复文本消息 | `weixinmp message reply text` | 输出符合微信格式的 XML |
| 被动回复图片消息 | `weixinmp message reply image` | 需要 `MediaId` |
| 被动回复语音消息 | `weixinmp message reply voice` | 需要 `MediaId` |
| 被动回复视频消息 | `weixinmp message reply video` | 支持标题和描述 |
| 被动回复音乐消息 | `weixinmp message reply music` | 支持普通 / 高品质音乐 URL |
| 被动回复图文消息 | `weixinmp message reply news` | 支持多个 `article` |
| 被动回复转发到客服 | `weixinmp message reply transfer-customer-service` | 支持可选指定客服账号 |

## 解析入站 XML

### 从文件读取

```bash
weixinmp message parse --input-file message.xml --output json
```

### 从标准输入读取

```bash
cat message.xml | weixinmp message parse
```

### 文本摘要输出

```bash
weixinmp message parse --input-file message.xml --output text
```

文本输出会按行展示常用字段，例如：

```text
msg_type=text
from_user=oUserOpenID
to_user=gh_xxxxx
create_time=1710000000
content=hello
```

## 生成被动回复 XML

### 复用原始请求自动交换收发双方

在被动回复场景里，最常见的需求是把原始请求中的：

- `FromUserName` → 回复 XML 的 `ToUserName`
- `ToUserName` → 回复 XML 的 `FromUserName`

CLI 支持通过 `--request-file` 自动完成这件事：

```bash
weixinmp message reply text \
  --request-file message.xml \
  --content "收到"
```

如果不提供 `--request-file`，则需要显式指定：

```bash
--to-user oUserOpenID --from-user gh_xxxxx
```

### 文本回复

```bash
weixinmp message reply text \
  --request-file message.xml \
  --content "收到，你好"
```

### 图片回复

```bash
weixinmp message reply image \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --media-id MEDIA_ID
```

### 语音回复

```bash
weixinmp message reply voice \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --media-id MEDIA_ID
```

### 视频回复

```bash
weixinmp message reply video \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --media-id MEDIA_ID \
  --title "标题" \
  --description "描述"
```

### 音乐回复

```bash
weixinmp message reply music \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --title "标题" \
  --description "描述" \
  --music-url "https://example.com/music.mp3" \
  --hq-music-url "https://example.com/music-hq.mp3" \
  --thumb-media-id THUMB_MEDIA_ID
```

### 图文回复

单篇图文：

```bash
weixinmp message reply news \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --article '{"title":"测试图文","description":"图文描述","pic_url":"https://example.com/1.png","url":"https://example.com/1"}'
```

多篇图文：

```bash
weixinmp message reply news \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --article '{"title":"第一篇","description":"描述1","pic_url":"https://example.com/1.png","url":"https://example.com/1"}' \
  --article '{"title":"第二篇","description":"描述2","pic_url":"https://example.com/2.png","url":"https://example.com/2"}'
```

也可以改用 JSON 文件：

```bash
weixinmp message reply news \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --article-file articles.json
```

`articles.json` 示例：

```json
[
  {
    "title": "第一篇",
    "description": "描述1",
    "pic_url": "https://example.com/1.png",
    "url": "https://example.com/1"
  },
  {
    "title": "第二篇",
    "description": "描述2",
    "pic_url": "https://example.com/2.png",
    "url": "https://example.com/2"
  }
]
```

### 转发到客服

```bash
weixinmp message reply transfer-customer-service \
  --request-file message.xml
```

如需指定客服账号：

```bash
weixinmp message reply transfer-customer-service \
  --request-file message.xml \
  --kf-account "test1@test"
```

## 说明

- `message parse` 和 `message reply ...` 不依赖公众号 `AppID` / `Secret`
- 回复 XML 的 `CreateTime` 默认使用当前 Unix 时间戳，也可以用 `--create-time` 显式覆盖
- 当前 CLI 聚焦被动回复与 XML 解析；消息主动发送、模板消息、订阅通知等接口暂未放到这一组命令里
