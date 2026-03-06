# weixinmp

[![Unit Test](https://github.com/bestony/weixinmp/actions/workflows/unit-test.yml/badge.svg)](https://github.com/bestony/weixinmp/actions/workflows/unit-test.yml)
[![Release](https://github.com/bestony/weixinmp/actions/workflows/release.yml/badge.svg)](https://github.com/bestony/weixinmp/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/bestony/weixinmp)](https://github.com/bestony/weixinmp/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bestony/weixinmp)](https://github.com/bestony/weixinmp/blob/main/go.mod)
[![GitHub Stars](https://img.shields.io/github/stars/bestony/weixinmp?style=social)](https://github.com/bestony/weixinmp/stargazers)
[![GitHub Issues](https://img.shields.io/github/issues/bestony/weixinmp)](https://github.com/bestony/weixinmp/issues)
[![License](https://img.shields.io/github/license/bestony/weixinmp)](https://github.com/bestony/weixinmp/blob/main/LICENSE)

`weixinmp` 是一个面向微信公众号（Weixin MP / Official Account）的轻量级命令行工具，适合在本地调试、脚本自动化、CI 流程或服务接入排障时使用。

当前版本聚焦三个高频场景：

- 获取公众号 `access_token`
- 计算与校验开发者接入所需的 SHA1 签名
- 查询公众号接口调用所使用的 API 域名 IP 列表

项目使用 Go 编写，命令行基于 `kong`，并对关键行为提供了完整测试覆盖。

## 功能特性

- 支持通过命令行、环境变量或配置文件读取公众号凭据
- 支持文本与 JSON 两种输出格式，方便脚本集成
- 支持计算和校验微信服务器接入签名
- 支持查询公众号 API 域名 IP 列表
- 内置 `--debug`、`--version`、`--config` 等通用选项
- CI 约束总测试覆盖率保持在 `100%`

## 安装与构建

### 方式一：本地构建

```bash
go build .
```

构建完成后会生成当前平台可执行文件：

```bash
./weixinmp --help
```

### 方式二：源码直接运行

```bash
go run . --help
```

### 方式三：下载预编译版本

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
weixinmp --config /path/to/config.toml token get
```

### 2. 获取 Access Token

```bash
weixinmp token get
```

默认输出纯文本 token：

```text
ACCESS_TOKEN_VALUE
```

如需 JSON 输出：

```bash
weixinmp token get --output json
```

示例输出：

```json
{
  "access_token": "ACCESS_TOKEN_VALUE",
  "expires_in": 7200
}
```

你也可以在调试或测试时直接通过参数覆盖凭据：

```bash
weixinmp token get \
  --app-id wx1234567890abcdef \
  --secret your-wechat-secret
```

如需控制请求行为，还可以使用：

- `--timeout`：设置 HTTP 超时时间，默认 `10s`
- `--base-url`：覆盖微信 API 基地址，便于联调或测试

### 3. 计算服务器接入签名

```bash
weixinmp signature compute \
  --token your-token \
  --timestamp 1710000000 \
  --nonce 123456
```

该命令会输出根据微信规则排序并进行 SHA1 计算后的签名。

签名相关命令不依赖 `AppID` 或 `Secret`。

### 4. 校验服务器接入签名

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

### 5. 查询 API 域名 IP 列表

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

该命令同样支持 `--timeout`，默认值为 `10s`。

## 命令总览

```bash
weixinmp <command> [flags]
```

当前提供的命令包括：

- `token get`：通过 `client_credential` 模式获取公众号 `access_token`
- `signature compute`：计算微信公众号开发者接入签名
- `signature verify`：校验微信公众号开发者接入签名
- `official-account get-api-domain-ip`：查询公众号接口调用 IP 列表

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
```

其中最后一条命令与 CI 保持一致，并要求总覆盖率达到 `100%`。

### 本地测试凭据

项目提供了 `.env.test.example`，可用于本地测试：

```bash
cp .env.test.example .env.test
```

然后填入：

```dotenv
APPID=your_appid_here
APPSECRET=your_app_secret_here
```

测试会将这些值映射到：

- `WEIXINMP_APPID`
- `WEIXINMP_SECRET`

如果没有提供本地测试凭据，单元测试默认也可以使用安全的占位值配合 mocked HTTP 服务运行。

## 项目结构

```text
.
├── main.go                     # CLI 入口与命令定义
├── config.go                   # 配置文件、环境变量与参数优先级处理
├── *_test.go                   # CLI、配置、dotenv、端到端测试
├── internal/weixinmp/
│   ├── client.go               # access_token HTTP 客户端
│   ├── official_account.go     # 公众号 API 域名 IP 查询
│   ├── signature.go            # SHA1 签名计算与校验
│   └── *_test.go               # 对应单元测试
└── .github/workflows/          # 测试、覆盖率与发布流程
```

## 设计说明

- `token get` 直接调用微信 `/cgi-bin/token` 接口
- `official-account get-api-domain-ip` 基于 `silenceper/wechat` SDK 实现
- 输出格式尽量保持简单，便于 shell 脚本、CI 或其他程序消费

## 适用场景

- 本地调试微信公众号接口接入
- 编写 shell 脚本批量获取 `access_token`
- 验证微信服务器回调参数中的签名是否正确
- 排查公众号 API 域名解析、网络访问或白名单相关问题

## License

如仓库后续补充 License 文件，请以该文件为准。
