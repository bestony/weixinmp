# weixinmp

[![Unit Test](https://github.com/bestony/weixinmp/actions/workflows/unit-test.yml/badge.svg)](https://github.com/bestony/weixinmp/actions/workflows/unit-test.yml)
[![Release](https://github.com/bestony/weixinmp/actions/workflows/release.yml/badge.svg)](https://github.com/bestony/weixinmp/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/bestony/weixinmp)](https://github.com/bestony/weixinmp/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bestony/weixinmp)](https://github.com/bestony/weixinmp/blob/main/go.mod)
[![GitHub Stars](https://img.shields.io/github/stars/bestony/weixinmp?style=social)](https://github.com/bestony/weixinmp/stargazers)
[![GitHub Issues](https://img.shields.io/github/issues/bestony/weixinmp)](https://github.com/bestony/weixinmp/issues)
[![License](https://img.shields.io/github/license/bestony/weixinmp)](https://github.com/bestony/weixinmp/blob/main/LICENSE)

`weixinmp` 是一个面向微信公众号（Weixin MP / Official Account）的轻量级命令行工具，适合在本地调试、脚本自动化、CI 流程或服务接入排障时使用。

当前版本聚焦微信公众号基础接口里的几个高频场景：

- 解析微信服务器推送的 XML 消息 / 事件
- 生成被动回复 XML（文本、图片、图文、语音、视频、音乐、转客服）
- 查询微信服务器 callback IP / IP 段
- 按文本、图文、语音、图片、视频类型群发消息
- 计算与校验开发者接入所需的 SHA1 签名
- 查询公众号接口调用所使用的 API 域名 IP 列表
- 清理接口调用频次

项目使用 Go 编写，命令行基于 `kong`，并对关键行为提供了完整测试覆盖。

## 功能特性

- 支持通过命令行、环境变量或配置文件读取公众号凭据
- 支持文本与 JSON 两种输出格式，方便脚本集成
- 支持解析入站微信 XML 消息为文本或 JSON
- 支持生成多种被动回复 XML
- 支持计算和校验微信服务器接入签名
- 支持查询微信服务器 callback IP 列表
- 支持按多种消息类型执行公众号群发
- 支持查询公众号 API 域名 IP 列表
- 支持清理公众号接口调用频次
- 内置 `--debug`、`--version`、`--config` 等通用选项
- CI 约束总测试覆盖率保持在 `100%`

## 安装与构建

### 方式一：使用 npm 安装

发布到 npm 后，可以直接安装当前平台对应的预编译 binary，并注册 `weixinmp` 命令：

```bash
npm install -g weixinmp
```

安装完成后即可直接执行：

```bash
weixinmp --version
weixinmp --help
```

也可以不全局安装，直接运行：

```bash
npx weixinmp --help
```

npm 包会根据当前平台自动分发对应 binary，目前支持：

- Linux `amd64` / `arm64`
- macOS `amd64` / `arm64`
- Windows `amd64`

### 方式二：使用 Homebrew 安装

当前仓库名不是 `homebrew-*`，添加 tap 时需要显式指定仓库 URL：

```bash
brew tap --custom-remote bestony/weixinmp https://github.com/bestony/weixinmp
```

然后安装：

```bash
brew install bestony/weixinmp/weixinmp
```

如果已经 tap 成功，也可以使用短命令：

```bash
brew install weixinmp
```

升级到最新 release：

```bash
brew update && brew upgrade weixinmp
```

Homebrew formula 会在每次发布 `v*` GitHub Release 后自动更新，安装包来源于对应 release 里的预编译压缩包。

### 方式三：本地构建

```bash
go build .
```

构建完成后会生成当前平台可执行文件：

```bash
./weixinmp --help
```

### 方式四：源码直接运行

```bash
go run . --help
```

### 方式五：下载预编译版本

仓库的 GitHub Actions 在打 `v*` 标签时会自动构建并发布以下平台的压缩包：

- Linux `amd64` / `arm64`
- macOS `amd64` / `arm64`
- Windows `amd64`

## 快速开始

### 1. 配置公众号凭据

`weixinmp` 读取凭据的优先级如下：

1. 命令行参数：`--app-id`、`--secret`
2. 环境变量：`WEIXINMP_APPID`、`WEIXINMP_SECRET`
3. 配置文件：`~/.weixinmp/config.toml`

配置文件示例：

```toml
app_id = "wx1234567890abcdef"
secret = "your-wechat-secret"
```

如果你不想使用默认路径，也可以显式指定：

```bash
weixinmp --config /path/to/config.toml official-account get-api-domain-ip
```

### 2. 查询微信服务器 callback IP

```bash
weixinmp official-account get-callback-ip
```

默认按文本逐行输出：

```text
101.226.103.0
101.226.62.77
...
```

如需 JSON 输出：

```bash
weixinmp official-account get-callback-ip --output json
```

示例输出：

```json
{
  "ip_list": [
    "101.226.103.0",
    "101.226.62.77"
  ]
}
```

### 3. 查询 API 域名 IP 列表

```bash
weixinmp official-account get-api-domain-ip
```

默认按文本逐行输出：

```text
101.226.103.0
101.226.62.77
...
```

如需 JSON 输出：

```bash
weixinmp official-account get-api-domain-ip --output json
```

示例输出：

```json
{
  "ip_list": [
    "101.226.103.0",
    "101.226.62.77"
  ]
}
```

这两个 IP 查询命令都支持 `--timeout`，默认值为 `10s`。

### 4. 清理接口调用频次

```bash
weixinmp official-account clear-quota
```

默认输出：

```text
ok
```

如需 JSON 输出：

```bash
weixinmp official-account clear-quota --output json
```

示例输出：

```json
{
  "ok": true
}
```

参考 `silenceper/wechat` 文档，这个接口存在官方调用限制，不建议随意执行。

### 5. 群发消息

群发文本消息：

```bash
weixinmp official-account broadcast send-text \
  --to-all \
  --content "hello followers"
```

按标签群发图文消息：

```bash
weixinmp official-account broadcast send-news \
  --tag-id 2 \
  --media-id MEDIA_ID \
  --ignore-reprint
```

按 OpenID 群发语音 / 图片 / 视频消息：

```bash
weixinmp official-account broadcast send-voice \
  --open-id openid-1,openid-2 \
  --media-id MEDIA_ID

weixinmp official-account broadcast send-image \
  --open-id openid-1 \
  --media-id MEDIA_ID_1,MEDIA_ID_2

weixinmp official-account broadcast send-video \
  --open-id openid-1 \
  --media-id MEDIA_ID \
  --title "视频标题" \
  --description "视频描述"
```

更完整的参数说明见 `docs/official-account-broadcast.md`。

### 6. 解析入站消息 XML

```bash
weixinmp message parse --input-file message.xml --output json
```

如果不指定 `--input-file`，会从标准输入读取 XML：

```bash
cat message.xml | weixinmp message parse
```

示例输出：

```json
{
  "ToUserName": "gh_xxxxx",
  "FromUserName": "oUserOpenID",
  "CreateTime": 1710000000,
  "MsgType": "text",
  "Content": "hello"
}
```

如果只想看简要信息：

```bash
weixinmp message parse --input-file message.xml --output text
```

### 7. 生成被动文本回复

最方便的方式是直接复用原始请求 XML，让 CLI 自动交换 `ToUserName` / `FromUserName`：

```bash
weixinmp message reply text \
  --request-file message.xml \
  --content "收到，你好"
```

如果你已经知道双方账号，也可以直接指定：

```bash
weixinmp message reply text \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --content "收到，你好"
```

### 8. 生成其他被动回复

图片回复：

```bash
weixinmp message reply image \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --media-id MEDIA_ID
```

图文回复：

```bash
weixinmp message reply news \
  --to-user oUserOpenID \
  --from-user gh_xxxxx \
  --article '{"title":"测试图文","description":"图文描述","pic_url":"https://example.com/1.png","url":"https://example.com/1"}'
```

音乐回复：

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

转发到客服：

```bash
weixinmp message reply transfer-customer-service \
  --request-file message.xml \
  --kf-account "test1@test"
```

同一组命令还支持：

- `message reply voice`
- `message reply video`

其中 `news` 支持重复传入多个 `--article`，也支持用 `--article-file` 指向 JSON 数组文件。

### 9. 计算服务器接入签名

```bash
weixinmp signature compute \
  --token your-token \
  --timestamp 1710000000 \
  --nonce 123456
```

该命令会输出根据微信规则排序并进行 SHA1 计算后的签名。

签名相关命令不依赖 `AppID` 或 `Secret`。

### 10. 校验服务器接入签名

```bash
weixinmp signature verify \
  --token your-token \
  --timestamp 1710000000 \
  --nonce 123456 \
  --signature calculated-signature
```

校验成功时命令直接退出；校验失败时返回错误：

```text
signature mismatch
```

## 文档对照

当前 CLI 对齐了 `silenceper/wechat` 公众号基础接口文档中的以下能力：

- 获取微信 callback IP → `weixinmp official-account get-callback-ip`
- 获取微信 API 接口 IP → `weixinmp official-account get-api-domain-ip`
- 清理接口调用频次 → `weixinmp official-account clear-quota`
- 群发消息 → `weixinmp official-account broadcast ...`
- 解析入站消息 / 事件 XML → `weixinmp message parse`
- 生成被动回复 XML → `weixinmp message reply ...`

更完整的命令示例与接口映射见：

- `docs/official-account-basic.md`
- `docs/official-account-broadcast.md`
- `docs/official-account-message.md`

参考文档：

- `silenceper/wechat` 公众号基础接口说明：<https://silenceper.com/wechat/officialaccount/basic.html>
- `silenceper/wechat` 对应实现：<https://github.com/silenceper/wechat/blob/master/officialaccount/basic/basic.go>
- `silenceper/wechat` 群发接口说明：<https://silenceper.com/wechat/officialaccount/broadcast.html>
- `silenceper/wechat` 消息管理说明：<https://silenceper.com/wechat/officialaccount/message.html>
- 更完整的消息命令示例：`docs/official-account-message.md`

## 命令总览

```bash
weixinmp <command> [flags]
```

当前提供的命令包括：

- `message parse`：解析微信服务器推送 XML
- `message reply text|image|news|voice|video|music|transfer-customer-service`：生成被动回复 XML
- `signature compute`：计算微信公众号开发者接入签名
- `signature verify`：校验微信公众号开发者接入签名
- `official-account get-callback-ip`：查询微信服务器 callback IP 列表
- `official-account get-api-domain-ip`：查询公众号接口调用 IP 列表
- `official-account clear-quota`：清理公众号接口调用频次
- `official-account broadcast send-text|send-news|send-voice|send-image|send-video`：按不同消息类型执行群发

常用全局参数：

- `--debug`：输出调试日志
- `--version`：打印版本信息
- `--config`：指定配置文件路径

## 配置说明

### 环境变量

```bash
export WEIXINMP_APPID="wx1234567890abcdef"
export WEIXINMP_SECRET="your-wechat-secret"
```

### 配置文件路径

默认配置文件路径为：

```text
~/.weixinmp/config.toml
```

如果默认配置文件不存在，程序会继续尝试从环境变量和命令行读取配置；如果显式通过 `--config` 指定了路径但文件不存在，则会直接报错。

### 凭据缺失时的错误

如果最终仍未提供完整的 `AppID` 和 `Secret`，程序会返回：

```text
missing WeChat credentials: set --app-id/--secret, env vars, or a config file
```

## 开发与测试

### 常用命令

```bash
go build .
go run . --help
go test ./...
go test ./... -run TestCLI
go test ./... -coverprofile=coverage.out -covermode=atomic
make package-all VERSION=v0.0.0 TAG=v0.0.0
make pack-npm NPM_PACKAGE_VERSION=0.0.0 NPM_RELEASE_TAG=v0.0.0
make verify-npm-install
```

其中 `go test ./... -coverprofile=coverage.out -covermode=atomic` 与 CI 的覆盖率门禁保持一致，并要求总覆盖率达到 `100%`。后两条命令分别用于生成 npm 发布包和验证本地安装链路。

### 本地测试凭据

项目提供了 `.env.test.example`，可作为本地测试配置模板：

```bash
cp .env.test.example .env.test
```

然后填入：

```dotenv
APPID=your_appid_here
APPSECRET=your_app_secret_here
```

当前仓库内的大多数单元测试都使用 mocked HTTP 服务，不依赖真实公众号凭据；`.env.test` 主要用于保留一份本地测试配置模板，并在测试启动时校验其格式是否正确。

## 项目结构

```text
.
├── main.go                     # CLI 入口与命令定义
├── config.go                   # 配置文件、环境变量与参数优先级处理
├── docs/                       # 补充文档与接口对照说明
├── *_test.go                   # CLI、配置、dotenv、端到端测试
├── internal/weixinmp/
│   ├── official_account.go     # 公众号基础接口封装（callback IP / API Domain IP / ClearQuota / Broadcast）
│   ├── signature.go            # SHA1 签名计算与校验
│   └── *_test.go               # 对应单元测试
└── .github/workflows/          # 测试、覆盖率与发布流程
```

## 设计说明

- `official-account get-callback-ip`、`official-account get-api-domain-ip`、`official-account clear-quota`、`official-account broadcast ...` 基于 `silenceper/wechat` SDK 实现
- 输出格式尽量保持简单，便于 shell 脚本、CI 或其他程序消费

## 适用场景

- 本地调试微信公众号接口接入
- 对全量、标签或指定 OpenID 执行群发测试
- 验证微信服务器回调参数中的签名是否正确
- 获取微信服务器回调来源 IP 或 IP 段，用于白名单配置
- 排查公众号 API 域名解析、网络访问或白名单相关问题
- 在开发/测试阶段按需清理接口调用频次

## License

如仓库后续补充 License 文件，请以该文件为准。
