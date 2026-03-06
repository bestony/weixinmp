# 公众号群发接口 CLI 对照

本文档对照 `silenceper/wechat` 的公众号群发接口文档，说明 `weixinmp` 当前已经封装到 CLI 的能力、对应命令以及使用方式。

参考资料：

- `silenceper/wechat` 群发接口文档：<https://silenceper.com/wechat/officialaccount/broadcast.html>
- SDK 对应源码：<https://github.com/silenceper/wechat/blob/master/officialaccount/broadcast/broadcast.go>

## 已封装能力

| 文档能力 | CLI 命令 | 说明 |
| --- | --- | --- |
| 群发文本消息 | `weixinmp official-account broadcast send-text` | 支持按全部粉丝、标签或 OpenID 群发 |
| 群发图文消息 | `weixinmp official-account broadcast send-news` | 通过图文 `media_id` 群发 |
| 群发语音消息 | `weixinmp official-account broadcast send-voice` | 通过语音 `media_id` 群发 |
| 群发图片消息 | `weixinmp official-account broadcast send-image` | 支持一个或多个图片 `media_id` |
| 群发视频消息 | `weixinmp official-account broadcast send-video` | 通过视频 `media_id` + 标题/描述群发 |

## 凭据读取规则

所有群发命令都遵循同一套优先级：

1. 命令行参数：`--app-id`、`--secret`
2. 环境变量：`WEIXINMP_APPID`、`WEIXINMP_SECRET`
3. 配置文件：`~/.weixinmp/config.toml`

## 目标用户规则

每次群发必须且只能选择一个目标范围：

- `--to-all`：发送给全部粉丝
- `--tag-id`：发送给某个标签下的粉丝
- `--open-id`：发送给指定 OpenID，多个值用逗号分隔

例如：

```bash
weixinmp official-account broadcast send-text --to-all --content "hello all"
weixinmp official-account broadcast send-text --tag-id 2 --content "hello tag"
weixinmp official-account broadcast send-text --open-id openid-1,openid-2 --content "hello users"
```

## 命令示例

### 群发文本

```bash
weixinmp official-account broadcast send-text \
  --to-all \
  --content "hello followers"
```

### 群发图文

```bash
weixinmp official-account broadcast send-news \
  --tag-id 2 \
  --media-id MEDIA_ID \
  --ignore-reprint
```

### 群发语音

```bash
weixinmp official-account broadcast send-voice \
  --open-id openid-1,openid-2 \
  --media-id MEDIA_ID
```

### 群发图片

```bash
weixinmp official-account broadcast send-image \
  --open-id openid-1 \
  --media-id MEDIA_ID_1,MEDIA_ID_2 \
  --recommend "nice" \
  --need-open-comment 1 \
  --only-fans-can-comment 1
```

### 群发视频

```bash
weixinmp official-account broadcast send-video \
  --open-id openid-1 \
  --media-id MEDIA_ID \
  --title "视频标题" \
  --description "视频描述"
```

## 输出格式

默认文本输出会打印群发返回的 `msg_id`：

```text
1234567890
```

如需完整字段，请使用 JSON：

```bash
weixinmp official-account broadcast send-text --to-all --content "hello" --output json
```

示例输出：

```json
{
  "msg_id": 1234567890,
  "msg_data_id": 1234567890,
  "msg_status": "send success"
}
```

## 当前未封装项

`silenceper/wechat` 广播模块还提供了以下能力，但当前 CLI 还没有封装：

- 删除群发消息
- 预览群发消息
- 查询群发状态
- 查询 / 设置群发速度

如果后续要继续扩展，可以优先把这几个查询/运维型命令补上。
