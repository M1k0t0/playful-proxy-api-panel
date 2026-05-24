# Playful Proxy API Panel (PPAP)

[English](README.md) | [中文](README_CN.md) | 日本語

**PPAP は、自ホスト向けの CLIProxyAPI 互換 fork です。管理パネル、永続化される使用量分析、コスト推定、Codex 向け model alias をまとめて提供します。**

[`router-for-me/CLIProxyAPI`](https://github.com/router-for-me/CLIProxyAPI) の OpenAI/Gemini/Claude/Codex 互換 proxy surface を保ちながら、日常運用で必要になる usage snapshot、request/cost metrics、backend と同じ tag で配布される管理パネル、より安全な thinking strength alias を追加しています。

素の CLIProxyAPI が必要なら上流を使ってください。  
使用量、latency、cache hit、Codex strength routing をローカルで見たいなら PPAP を使ってください。

## PPAP の違い

- **使用量分析を内蔵**: `/v0/management/usage`、import/export、local snapshot persistence、cache hit rate、first-byte latency、average latency、TPS、token breakdown、model/API rollup。
- **パネルと backend を同時 release**: frontend source は [`web/management-panel`](web/management-panel) にあり、各 release に同じ tag で build された `management.html` が含まれます。
- **任意の full conversation logs**: operator が明示的に有効化した場合だけ、保護された request/response body logs を management panel で閲覧できます。
- **任意の upstream preset prompt**: operator prompt を upstream chat-like request の先頭に追加できますが、その prompt は API client に返しません。
- **Codex を主要 workflow として扱う**: OpenAI Codex OAuth、GPT model routing、Spark pricing estimate、thinking strength alias をこの fork で保守します。
- **thinking strength の書き方を統一**: `model(high)` と `model-high` の両方に対応し、`low`、`medium`、`high`、`xhigh` を扱います。explicit alias と exact model name が優先されます。
- **上流互換を維持**: 競合しない上流更新は取り込みます。Redis usage queue retention も含まれ、PPAP の usage persistence は維持されます。

## Core Features

- OpenAI/Gemini/Claude/Codex-compatible API endpoints
- OAuth login for OpenAI Codex and Claude Code
- Streaming and non-streaming responses
- Function calling, tools, and multimodal input
- Multi-account routing and load balancing
- Gemini CLI, AI Studio Build, Claude Code, OpenAI Codex, and Amp CLI support
- OpenAI-compatible upstream providers such as OpenRouter through config
- Protected full conversation log browsing when explicitly enabled
- Upstream-only preset prompt injection when explicitly enabled
- Reusable Go SDK

## Quick Start

この repository の [latest Release](https://github.com/daishuge/playful-proxy-api-panel/releases/latest) から platform に合う archive をダウンロードし、展開して local config で起動します。

```bash
cp config.example.yaml config.yaml
./cli-proxy-api -config ./config.yaml
```

Default HTTP port は `8317` です。

Release archives は upstream CPA と同じ platform family を対象にします。Linux、Windows、macOS、FreeBSD に対して、Go が対応する `amd64` と `aarch64`/`arm64` を提供します。

## Docker

Docker image は `ghcr.io/daishuge/playful-proxy-api-panel` に公開され、`linux/amd64` と `linux/arm64` をサポートします。Image には同じ tag から build された PPAP management panel が含まれるため、`/management.html` は初回 download なしで使えます。

Release archive または cloned checkout の中で:

```bash
cp config.docker.example.yaml config.yaml
mkdir -p auths logs data
# edit config.yaml: replace change-me-management-key and change-me-api-key
docker compose pull
docker compose up -d
```

Local build を使う場合:

```bash
docker compose up -d --build
```

Default persistent paths in `docker-compose.yml`:

- `./config.yaml` -> `/CLIProxyAPI/config.yaml`
- `./auths` -> `/root/.cli-proxy-api`
- `./data` -> `/CLIProxyAPI/data`
- `./logs` -> `/CLIProxyAPI/logs`

Compose file は upstream-compatible default host ports を保ちますが、各 host port は `.env` で上書きできます。たとえば host port `1455` がすでに使われていて、Codex OAuth callback をその local port に固定する必要がない場合:

```env
CLI_PROXY_CODEX_CALLBACK_PORT=1456
```

Built-in Codex OAuth callback を使う場合は、OAuth redirect URI が `http://localhost:1455/auth/callback` を使うため、host port `1455` を空けてください。

Docker bridge traffic は container 内では non-localhost として見えるため、`config.docker.example.yaml` は `remote-management.allow-remote` を有効化し、management key を必須にしています。Service を自分の machine 外へ公開する前に example key を置き換えてください。

`config.yaml`、`.env`、OAuth files、API keys、auth directories、logs、data snapshots、generated stores は git に commit しないでください。

## Configuration Notes

[`config.example.yaml`](config.example.yaml) から始めてください。PPAP でよく使う設定:

- `usage-statistics-enabled`: built-in usage snapshot を有効化。
- `usage-statistics-path`: snapshot file の保存先を指定。
- `redis-usage-queue-retention-seconds`: Redis usage queue retention を調整。
- `/v0/management/usage-queue`: Redis-compatible usage stream の queued records を external integrations 向けに pop。
- `api-key-controls`: client key ごとに model patterns、per-key preset prompts、request/token/estimated USD budgets を制御。budgets を使う場合は usage statistics を有効化します。
- `conversation-log`: default は disabled。保護された full request/response body logs が必要な場合だけ有効化します。
- `preset-prompt`: default は disabled。operator prompt は upstream chat-like request だけに注入します。`api-key-controls[].preset-prompt` はこの global block を上書きします。
- `oauth-model-alias`: friendly model alias を定義し、legacy config style も維持。

Production で有効化する前に [Conversation Logging And Preset Prompt Operations](docs/operations-conversation-logging-and-preset-prompt.md) を読み、log privacy、retention、prompt non-leak checks を確認してください。

thinking levels を宣言している model では、PPAP は次のような aliases を自動で公開できます。

```text
gpt-5.3-codex-spark-low
gpt-5.3-codex-spark-medium
gpt-5.3-codex-spark-high
gpt-5.3-codex-spark-xhigh
```

従来の parentheses style も使えます。

```text
gpt-5.3-codex-spark(high)
```

## Codex Spark Pricing

PPAP pricing data には `gpt-5.3-codex-spark` が含まれています。公式 preview pricing が安定するまでは、`gpt-5.3-codex` の推定 rate を使います。

References:

- [Introducing GPT-5.3-Codex-Spark](https://openai.com/index/introducing-gpt-5-3-codex-spark/)
- [Codex rate card](https://help.openai.com/en/articles/11369540-codex-rate-card)
- [OpenAI API pricing](https://openai.com/api/pricing/)

## Management

- Management panel source: [`web/management-panel`](web/management-panel)
- Management API docs: [help.router-for.me/management/api](https://help.router-for.me/management/api)
- Usage endpoints: `/v0/management/usage`, `/v0/management/usage/export`, `/v0/management/usage/import`
- Usage queue endpoint: `/v0/management/usage-queue?count=100`
- Conversation log endpoints: `/v0/management/conversation-logs`, `/v0/management/conversation-logs/tail`, `/v0/management/conversation-logs/{id}`
- Amp CLI guide: [help.router-for.me/agent-client/amp-cli.html](https://help.router-for.me/agent-client/amp-cli.html)

Release asset の `management.html` は backend binaries と同じ tag から build されます。

## Compatible Ecosystem

PPAP は built-in usage analytics を維持しつつ、upstream-style Management API と usage queue integrations と互換性を保ちます。

- [CPA-Manager](https://github.com/seakee/CPA-Manager): request-level monitoring、cost estimation、SQLite persistence、Codex account-pool operations。
- [CLIProxyAPI Usage Dashboard](https://github.com/zhanglunet/cliproxyapi-usage-dashboard): usage queue を消費する local usage/quota dashboard。
- [CLIProxy Pool Watch](https://github.com/murasame612/CLIProxyPoolWidget): CLIProxyAPI pool 向け macOS account quota monitor。
- [Codex Switch](https://github.com/9ycrooked/CodexSwitch): OpenAI Codex auth files と quota checks 向け desktop account-profile switcher。

## SDK And Docs

- SDK usage: [docs/sdk-usage.md](docs/sdk-usage.md)
- Advanced executors and translators: [docs/sdk-advanced.md](docs/sdk-advanced.md)
- Access: [docs/sdk-access.md](docs/sdk-access.md)
- Watcher: [docs/sdk-watcher.md](docs/sdk-watcher.md)
- Operations: [Conversation Logging And Preset Prompt Operations](docs/operations-conversation-logging-and-preset-prompt.md)
- Custom provider example: [`examples/custom-provider`](examples/custom-provider)

## License

MIT. See [LICENSE](LICENSE).
