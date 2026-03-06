# 公众号基础接口 CLI 对照

本文档对照 `silenceper/wechat` 的公众号基础接口文档，说明 `weixinmp` 当前已经封装到 CLI 的能力、对应命令以及使用方式。

参考资料：

- `silenceper/wechat` 公众号基础接口文档：<https://silenceper.com/wechat/officialaccount/basic.html>
- SDK 对应源码：<https://github.com/silenceper/wechat/blob/master/officialaccount/basic/basic.go>

## 已封装能力

| 文档能力 | CLI 命令 | 说明 |
| --- | --- | --- |
| 获取微信 callback IP | `weixinmp official-account get-callback-ip` | 返回微信服务器回调来源 IP / IP 段 |
| 获取微信 API 接口 IP | `weixinmp official-account get-api-domain-ip` | 返回公众号 API 域名对应 IP 列表 |
| 清理接口调用频次 | `weixinmp official-account clear-quota` | 适合开发/测试阶段排障，调用要谨慎 |

## 凭据读取规则

所有需要公众号凭据的命令都遵循同一套优先级：

1. 命令行参数：`--app-id`、`--secret`
2. 环境变量：`WEIXINMP_APPID`、`WEIXINMP_SECRET`
3. 配置文件：`~/.weixinmp/config.toml`

配置文件示例：

```toml
app_id = "wx1234567890abcdef"
secret = "your-wechat-secret"
```

## 命令示例

### 获取微信 callback IP

```bash
weixinmp official-account get-callback-ip
weixinmp official-account get-callback-ip --output json
```

文本输出会逐行打印 IP；JSON 输出格式为：

```json
{
  "ip_list": [
    "101.226.103.0",
    "101.226.62.77"
  ]
}
```

### 获取微信 API 域名 IP

```bash
weixinmp official-account get-api-domain-ip
weixinmp official-account get-api-domain-ip --output json
```

输出格式与 `get-callback-ip` 相同。

### 清理接口调用频次

```bash
weixinmp official-account clear-quota
weixinmp official-account clear-quota --output json
```

默认文本输出：

```text
ok
```

JSON 输出：

```json
{
  "ok": true
}
```

根据 `silenceper/wechat` 文档中的提示，这个接口有官方调用限制，建议只在确实需要时执行。

## 当前未封装项

当前不再提供直接获取 `access_token` 的独立 CLI 命令。

文档里还提到“替换 `access_token` 获取方法”的 SDK 集成方式。这个能力主要面向 Go 代码中的 `officialaccount.Options` 配置，不是一个天然的命令行操作，因此当前没有封装成单独 CLI 命令。

如果后续要继续扩展，可以优先考虑：

- 增加更多公众号基础接口的查询型命令
- 为高风险写操作增加额外确认参数
- 补充更多面向脚本集成的 JSON 输出字段
