# Playful Proxy API Panel (PPAP)

[English](README.md) | 中文 | [日本語](README_JA.md)

**PPAP 是一个面向自托管的 CLIProxyAPI 兼容 fork：内置管理面板、使用量统计、成本估算和更顺手的 Codex 模型别名。**

它保留 [`router-for-me/CLIProxyAPI`](https://github.com/router-for-me/CLIProxyAPI) 熟悉的 OpenAI/Gemini/Claude/Codex 兼容代理接口，同时补上长期运行时最需要的东西：可持久化的 usage 快照、请求与成本指标、和后端同 tag 发布的管理面板、以及更安全的 thinking 强度别名。

如果你只想用原版项目，用上游 CLIProxyAPI。  
如果你想要上游代理能力，同时希望本地看得见用量、延迟、缓存命中和 Codex 强度路由，用 PPAP。

## PPAP 有什么不一样

- **内置 usage 分析**：恢复 `/v0/management/usage`、导入/导出接口、本地快照持久化，并记录缓存命中率、首字响应时间、平均耗时、TPS、Token 细分、模型/API 汇总。
- **管理面板和后端同步发布**：前端源码在 [`web/management-panel`](web/management-panel)，每个 release 都带同 tag 构建的 `management.html`。
- **可选全量对话日志**：需要时可以手动启用受保护的请求/响应正文日志，并在管理面板里浏览。
- **可选上游预设 prompt**：需要时可以把 operator prompt 加到上游 chat-like 请求开头，但不会返回给 API 调用方。
- **把 Codex 当主场景维护**：支持 OpenAI Codex OAuth、GPT 模型路由、Spark 定价估算、thinking 强度别名。
- **thinking 强度写法统一**：`model(high)` 和 `model-high` 都支持，强度为 `low`、`medium`、`high`、`xhigh`；显式 alias 和真实模型名优先。
- **继续跟上游兼容**：能合的上游更新继续合；当前已纳入 Redis usage queue retention，同时保留 PPAP 自己的 usage persistence。

## 核心能力

- OpenAI/Gemini/Claude/Codex 兼容 API 端点
- OpenAI Codex 和 Claude Code OAuth 登录
- 流式与非流式响应
- 函数调用、工具调用、多模态输入
- 多账户路由和负载均衡
- Gemini CLI、AI Studio Build、Claude Code、OpenAI Codex、Amp CLI 支持
- 通过配置接入 OpenAI-compatible 上游，例如 OpenRouter
- 手动启用后可在受保护管理入口浏览全量对话日志
- 手动启用后可向上游注入不会下发给用户的预设 prompt
- 可复用 Go SDK

## 快速开始

从本仓库 [latest Release](https://github.com/daishuge/playful-proxy-api-panel/releases/latest) 下载对应平台压缩包，解压后用本地配置启动：

```bash
cp config.example.yaml config.yaml
./cli-proxy-api -config ./config.yaml
```

默认 HTTP 端口是 `8317`。

Release 压缩包覆盖与上游 CPA 相同的平台族：Linux、Windows、macOS、FreeBSD，并提供 Go 支持的 `amd64` 与 `aarch64`/`arm64` 架构。

## Docker

Docker 镜像发布在 `ghcr.io/daishuge/playful-proxy-api-panel`，支持 `linux/amd64` 和 `linux/arm64`。镜像内置同 tag 构建的 PPAP 管理面板，`/management.html` 不需要先下载面板资源。

在 release 压缩包或 clone 出来的仓库目录中：

```bash
cp config.docker.example.yaml config.yaml
mkdir -p auths logs data
# 编辑 config.yaml，替换 change-me-management-key 和 change-me-api-key
docker compose pull
docker compose up -d
```

如果要本地构建镜像：

```bash
docker compose up -d --build
```

`docker-compose.yml` 默认持久化路径：

- `./config.yaml` -> `/CLIProxyAPI/config.yaml`
- `./auths` -> `/root/.cli-proxy-api`
- `./data` -> `/CLIProxyAPI/data`
- `./logs` -> `/CLIProxyAPI/logs`

Compose 文件保持上游兼容的默认宿主机端口，但每个宿主机端口都可以从 `.env` 覆盖。例如宿主机 `1455` 已被占用，并且不需要 Codex OAuth callback 固定使用这个本机端口时：

```env
CLI_PROXY_CODEX_CALLBACK_PORT=1456
```

如果依赖内置 Codex OAuth callback，请保持宿主机 `1455` 可用，因为 OAuth redirect URI 使用 `http://localhost:1455/auth/callback`。

Docker bridge 流量在容器内会被视为非 localhost，所以 `config.docker.example.yaml` 默认开启 `remote-management.allow-remote`，并要求配置管理密钥。把服务暴露到本机之外前必须替换示例密钥。

不要把 `config.yaml`、`.env`、OAuth 文件、API key、auth 目录、日志、数据快照和生成数据提交进 git。

## 配置重点

从 [`config.example.yaml`](config.example.yaml) 开始。PPAP 里最常用的相关配置：

- `usage-statistics-enabled`：启用内置使用量快照。
- `usage-statistics-path`：可选，把快照文件放到指定路径。
- `redis-usage-queue-retention-seconds`：Redis usage queue 启用时的保留时间。
- `/v0/management/usage-queue`：弹出 Redis-compatible usage stream 中排队的使用量记录，方便外部集成消费。
- `api-key-controls`：可选，按 client key 限制模型 pattern、per-key preset prompt 和 request/token/estimated USD 用量预算；启用预算时需要同时开启 usage statistics。
- `conversation-log`：默认关闭；只有需要受保护的完整请求/响应正文日志时才启用。
- `preset-prompt`：默认关闭；只把 operator prompt 注入上游 chat-like 请求。`api-key-controls[].preset-prompt` 会覆盖这个全局配置。
- `oauth-model-alias`：配置友好模型别名，同时兼容老配置写法。

上线前先读 [Conversation Logging And Preset Prompt Operations](docs/operations-conversation-logging-and-preset-prompt.md)，尤其是日志隐私、retention 和 prompt 不外泄检查。

对于明确支持 thinking levels 的模型，PPAP 可以自动暴露：

```text
gpt-5.3-codex-spark-low
gpt-5.3-codex-spark-medium
gpt-5.3-codex-spark-high
gpt-5.3-codex-spark-xhigh
```

老写法仍然有效：

```text
gpt-5.3-codex-spark(high)
```

## Codex Spark 定价

PPAP 已把 `gpt-5.3-codex-spark` 加入本地 usage 成本估算。官方 preview 定价稳定前，暂时沿用 `gpt-5.3-codex` 估算价。

参考：

- [Introducing GPT-5.3-Codex-Spark](https://openai.com/index/introducing-gpt-5-3-codex-spark/)
- [Codex rate card](https://help.openai.com/en/articles/11369540-codex-rate-card)
- [OpenAI API pricing](https://openai.com/api/pricing/)

## 管理入口

- 管理面板源码：[`web/management-panel`](web/management-panel)
- 管理 API 文档：[help.router-for.me/cn/management/api](https://help.router-for.me/cn/management/api)
- Usage 接口：`/v0/management/usage`、`/v0/management/usage/export`、`/v0/management/usage/import`
- Usage queue 接口：`/v0/management/usage-queue?count=100`
- Conversation log 接口：`/v0/management/conversation-logs`、`/v0/management/conversation-logs/tail`、`/v0/management/conversation-logs/{id}`
- Amp CLI 指南：[help.router-for.me/cn/agent-client/amp-cli.html](https://help.router-for.me/cn/agent-client/amp-cli.html)

Release 里的 `management.html` 与后端二进制来自同一个 tag，运行中的 PPAP 可以直接把面板更新地址指向本仓库。

## 兼容生态

PPAP 保留内置 usage 分析，同时继续兼容上游风格的 Management API 和 usage queue 集成：

- [CPA-Manager](https://github.com/seakee/CPA-Manager)：请求级监控、费用估算、SQLite 持久化，以及 Codex 账号池运维。
- [CLIProxyAPI Usage Dashboard](https://github.com/zhanglunet/cliproxyapi-usage-dashboard)：消费 usage queue 的本地用量和配额看板。
- [CLIProxy Pool Watch](https://github.com/murasame612/CLIProxyPoolWidget)：面向 CLIProxyAPI 池的 macOS 账号额度监控工具。
- [Codex Switch](https://github.com/9ycrooked/CodexSwitch)：用于 OpenAI Codex auth 文件和配额检查的桌面账号切换工具。

## SDK 和文档

- SDK 使用：[docs/sdk-usage_CN.md](docs/sdk-usage_CN.md)
- 高级执行器与翻译器：[docs/sdk-advanced_CN.md](docs/sdk-advanced_CN.md)
- 认证与访问：[docs/sdk-access_CN.md](docs/sdk-access_CN.md)
- 凭据加载/更新：[docs/sdk-watcher_CN.md](docs/sdk-watcher_CN.md)
- 运维说明：[Conversation Logging And Preset Prompt Operations](docs/operations-conversation-logging-and-preset-prompt.md)
- 自定义 Provider 示例：[`examples/custom-provider`](examples/custom-provider)

## 许可证

MIT。见 [LICENSE](LICENSE)。
