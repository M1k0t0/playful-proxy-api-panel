# Playful Proxy API Panel (PPAP)

English | [中文](README_CN.md) | [日本語](README_JA.md)

**PPAP is a self-hosted, upstream-compatible CLIProxyAPI fork with a built-in management panel, persistent usage analytics, and Codex-focused model ergonomics.**

It keeps the familiar OpenAI/Gemini/Claude/Codex-compatible proxy surface from [`router-for-me/CLIProxyAPI`](https://github.com/router-for-me/CLIProxyAPI), then adds the pieces that matter when you run it every day: usage snapshots, cost estimates, panel assets released with the backend, and safer thinking-strength aliases.

Use upstream CLIProxyAPI when you want the vanilla project. Use PPAP when you want the same proxy style with more local visibility and a tighter operations loop.

## What Makes PPAP Different

- **Usage analytics built in**: restored `/v0/management/usage`, import/export endpoints, persistent local snapshots, cache hit rate, first-byte latency, average latency, TPS, token breakdowns, and per-model/per-API rollups.
- **Panel and backend released together**: the management panel source lives in [`web/management-panel`](web/management-panel), and each release ships the matching `management.html`.
- **Opt-in full conversation logs**: operators can enable protected request/response body logs and browse them from the management panel.
- **Opt-in upstream preset prompt**: deployments can prepend an operator prompt to upstream chat-like requests without returning that prompt to API clients.
- **Codex is treated as a primary workflow**: OpenAI Codex OAuth, GPT model routing, Spark pricing estimation, and thinking-strength aliases are maintained in this fork.
- **Thinking aliases are predictable**: both `model(high)` and `model-high` work for `low`, `medium`, `high`, and `xhigh`; explicit aliases and exact model names stay higher priority.
- **Upstream compatibility is still the baseline**: upstream fixes are merged where they do not conflict with PPAP behavior. Recent Redis usage queue retention support is included.

## Core Features

- OpenAI/Gemini/Claude/Codex-compatible API endpoints for CLI models
- OAuth login for OpenAI Codex and Claude Code
- Streaming and non-streaming responses
- Function calling/tools and multimodal input
- Multi-account routing and load balancing
- Gemini CLI, AI Studio Build, Claude Code, OpenAI Codex, and Amp CLI support
- OpenAI-compatible upstream providers such as OpenRouter through config
- Protected full conversation log browsing when explicitly enabled
- Upstream-only preset prompt injection when explicitly enabled
- Reusable Go SDK for embedding the proxy

## Quick Start

Download the [latest PPAP release](https://github.com/daishuge/playful-proxy-api-panel/releases/latest), extract the archive for your platform, then start with a local config file:

```bash
cp config.example.yaml config.yaml
./cli-proxy-api -config ./config.yaml
```

The default HTTP port is `8317`.

Release archives cover the same platform families as upstream CPA: Linux, Windows, macOS, and FreeBSD on `amd64` and `aarch64`/`arm64` where Go supports them.

## Docker

The Docker image is published as `ghcr.io/daishuge/playful-proxy-api-panel` for `linux/amd64` and `linux/arm64`. The image bundles the PPAP management panel built from the same tag, so `/management.html` works without downloading a panel asset first.

From a release archive or cloned checkout:

```bash
cp config.docker.example.yaml config.yaml
mkdir -p auths logs data
# edit config.yaml: replace change-me-management-key and change-me-api-key
docker compose pull
docker compose up -d
```

To build the image locally instead of pulling GHCR:

```bash
docker compose up -d --build
```

Default persistent paths in `docker-compose.yml`:

- `./config.yaml` -> `/CLIProxyAPI/config.yaml`
- `./auths` -> `/root/.cli-proxy-api`
- `./data` -> `/CLIProxyAPI/data`
- `./logs` -> `/CLIProxyAPI/logs`

The Compose file keeps upstream-compatible default host ports, but each host port can be overridden from `.env`. For example, if host port `1455` is already in use and you do not need the Codex OAuth callback on that exact local port:

```env
CLI_PROXY_CODEX_CALLBACK_PORT=1456
```

Keep host port `1455` available when you rely on the built-in Codex OAuth callback, because the OAuth redirect URI uses `http://localhost:1455/auth/callback`.

Docker bridge requests are remote from the container's point of view, so `config.docker.example.yaml` enables `remote-management.allow-remote` and requires a management key. Replace the example keys before exposing the service beyond your own machine.

Keep `config.yaml`, `.env`, OAuth files, API keys, auth directories, logs, data snapshots, and generated stores out of git.

## Configuration Notes

Start from [`config.example.yaml`](config.example.yaml). The most useful PPAP-specific settings are:

- `usage-statistics-enabled`: enable built-in usage snapshots.
- `usage-statistics-path`: optionally move the usage snapshot away from the config directory.
- `redis-usage-queue-retention-seconds`: tune Redis usage queue retention when Redis usage queueing is enabled.
- `/v0/management/usage-queue`: pop queued usage records for integrations that consume the Redis-compatible usage stream.
- `api-key-controls`: optionally restrict individual client keys by model patterns, per-key preset prompts, and request/token/estimated USD budgets. Enable usage statistics when using budgets.
- `conversation-log`: disabled by default; enable only when you want protected full request/response body logs.
- `preset-prompt`: disabled by default; injects an operator prompt only into upstream chat-like requests. Per-key `api-key-controls[].preset-prompt` overrides this global block.
- `oauth-model-alias`: define friendly model aliases while preserving old config compatibility.

See [Conversation Logging And Preset Prompt Operations](docs/operations-conversation-logging-and-preset-prompt.md) before enabling either feature in production.

For models that declare thinking levels, PPAP can expose automatic aliases such as:

```text
gpt-5.3-codex-spark-low
gpt-5.3-codex-spark-medium
gpt-5.3-codex-spark-high
gpt-5.3-codex-spark-xhigh
```

The older parenthesized style still works:

```text
gpt-5.3-codex-spark(high)
```

## Codex Spark Pricing

`gpt-5.3-codex-spark` is included in PPAP pricing data for local usage-cost estimation. Until official preview pricing settles, PPAP temporarily estimates it with the `gpt-5.3-codex` rate.

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

The release asset `management.html` is built from the same tag as the backend binaries, so a running PPAP instance can point its panel updater at this repository.

## Compatible Ecosystem

PPAP keeps built-in usage analytics, but it also stays compatible with upstream-style Management API and usage queue integrations:

- [CPA-Manager](https://github.com/seakee/CPA-Manager): request-level monitoring, cost estimation, SQLite persistence, and Codex account-pool operations.
- [CLIProxyAPI Usage Dashboard](https://github.com/zhanglunet/cliproxyapi-usage-dashboard): local usage and quota dashboard that consumes the usage queue.
- [CLIProxy Pool Watch](https://github.com/murasame612/CLIProxyPoolWidget): macOS account quota monitor for CLIProxyAPI pools.
- [Codex Switch](https://github.com/9ycrooked/CodexSwitch): desktop account-profile switcher for OpenAI Codex auth files and quota checks.

## SDK And Docs

- SDK usage: [docs/sdk-usage.md](docs/sdk-usage.md)
- Advanced executors and translators: [docs/sdk-advanced.md](docs/sdk-advanced.md)
- Access: [docs/sdk-access.md](docs/sdk-access.md)
- Watcher: [docs/sdk-watcher.md](docs/sdk-watcher.md)
- Operations: [Conversation Logging And Preset Prompt Operations](docs/operations-conversation-logging-and-preset-prompt.md)
- Custom provider example: [`examples/custom-provider`](examples/custom-provider)

## License

MIT. See [LICENSE](LICENSE).
