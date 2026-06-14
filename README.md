<h1 align="center">Lightcode</h1>
<p align="center">A lightweight terminal coding agent written in Go</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img alt="Bubble Tea" src="https://img.shields.io/badge/TUI-Bubble%20Tea-FFB300?style=flat-square" />
  <img alt="OpenAI Compatible" src="https://img.shields.io/badge/API-OpenAI%20Compatible-10A37F?style=flat-square" />
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/License-MIT-black?style=flat-square" /></a>
</p>

![Lightcode demo](assets/lightcode.gif)

---

## What is Lightcode?

Lightcode is a terminal-based coding agent for developers. It connects to any OpenAI-compatible model provider.

## Features

- **OpenAI-compatible** — works with any provider that speaks the OpenAI Chat Completions API (OpenAI, Anthropic via proxy, Ollama, LM Studio, etc.)
- **OAuth providers** — sign in from the TUI with Codex ChatGPT auth or GitHub Copilot device login
- **Low memory usage** — uses less ram around 20-30 mb
- **Skills** — uses specialized [agent-skills](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview).
- **Multi-model support** — configure multiple providers and switch between models with `/models` inside the TUI


## Requirements

- [Go](https://go.dev/dl/) **1.25+**
- At least one OpenAI-compatible endpoint configured


## Install

```bash
go install github.com/Kartik-2239/lightcode/cmd/lightcode@latest
```


## Quick Start

Run the **TUI** and **API server** together (defaults to `:8080`):

```bash
lightcode
```

Or run directly from source:

```bash
go run ./cmd/lightcode/main.go
```

On first run, Lightcode creates `~/.lightcode/` with a default `config.json`. If no provider is configured yet, the TUI still opens and prompts you to run `/login`.

## TUI commands

| Command | Description |
|---------|-------------|
| `/login` | Open the provider login picker. Supports `codex` and `copilot`. |
| `/logout` | Open the provider logout picker and clear saved auth for the selected provider. |
| `/models` | Switch the active model. OAuth-backed models appear as `codex auth` or `copilot auth`. |
| `/effort` | Set Codex reasoning effort when the selected model supports it. |

### Codex OAuth

Lightcode can use your existing Codex ChatGPT login instead of an OpenAI API key.

1. Sign in with the Codex CLI:

```bash
codex login
```

2. Make sure Codex is using file-based credential storage if your machine does not have `~/.codex/auth.json`:

```toml
# ~/.codex/config.toml
cli_auth_credentials_store = "file"
```

3. Start Lightcode and select `Codex` during onboarding, or run `/login` in the TUI and choose `codex` to launch `codex login`. After login, open `/models` and choose a model from the `codex auth` provider.

Use `/effort` with Codex auth models to set reasoning effort: `low`, `medium`, `high`, or `extra high` (`xhigh` on the wire).

Lightcode imports ChatGPT OAuth tokens from `${CODEX_HOME:-~/.codex}/auth.json` into `~/.lightcode/auth.json` under the `codex` provider. Treat both files like passwords and do not commit or share them.

If a request fails with `token_invalidated` or `refresh_token_invalidated`, run `/login` and choose `codex` again. Lightcode clears its imported Codex token, runs `codex logout`, launches a fresh `codex login`, and re-imports the refreshed auth cache.

### GitHub Copilot OAuth

Run `/login` in the TUI and choose `copilot` to start GitHub's device login flow. Lightcode opens `https://github.com/login/device`, shows the user code in the chat, polls until GitHub accepts the authorization, then saves the access token under the `copilot` provider.

After login, open `/models` and choose a model from the `copilot auth` provider. Lightcode shows the same picker labels used by Copilot and maps them to the underlying model IDs internally:

- `Auto`
- `GPT-5.4 mini (default)`
- `GPT-5 mini`
- `Claude Haiku 4.5`
- `Gemini 3.1 Pro (Preview)`

Run `/logout` and choose `copilot` to remove the saved token and any selected Copilot model.

Copilot OAuth uses GitHub's documented device flow. Copilot model access uses GitHub Copilot service endpoints and can change outside Lightcode's control. Treat `~/.lightcode/auth.json` like a password and do not commit or share it.


## Configuration

All settings live under **`~/.lightcode/config.json`**. The file is created automatically on first run.

### Full example

```json
{
  "theme": "light",
  "skills_path": "~/.lightcode/skills",
  "port": "8080",
  "providers": [
    {
      "models": ["gpt-5.5"],
      "base_url": "https://api.openai.com/v1",
      "api_key": "sk-..."
    },
    {
      "models": ["some-ai-800b"],
      "base_url": "https://your-gateway.example/v1",
      "api_key": "your-api-key"
    }
  ]
}
```

OAuth models are stored separately in `~/.lightcode/auth.json`:

```json
{
  "codex": {
    "type": "oauth",
    "access_token": "...",
    "refresh_token": "...",
    "expires": 2000000000,
    "account_id": "...",
    "models": ["gpt-5.5", "gpt-5.4-mini", "gpt-5.3-codex-spark"]
  },
  "copilot": {
    "type": "oauth",
    "access_token": "...",
    "refresh_token": "",
    "expires": 0,
    "account_id": "",
    "models": ["Auto", "GPT-5.4 mini (default)", "GPT-5 mini", "Claude Haiku 4.5", "Gemini 3.1 Pro (Preview)"]
  }
}
```

### Config reference

| Key | Default | Description |
|-----|---------|-------------|
| `theme` | `"light"` | UI theme — `"light"` or `"dark"` |
| `skills_path`| `~/.lightcode/skills` | Path to your skills directory (or change it to another skill path) |
| `port` | `"8080"` | Port for the local HTTP API server |
| `providers` | `[]` | List of model providers (see below) |

### Providers

Each entry in the `providers` array requires:

| Key | Description |
|-----|-------------|
| `models` | List of model IDs available at this endpoint |
| `base_url` | Base URL of the OpenAI-compatible API |
| `api_key` | API key for authentication |

Once configured, run `/models` inside the TUI to select your active model.


## Skills

Skills give the agent domain-specific context and significantly improve response quality for specialized tasks.

**To add a skill:**

1. Create a subdirectory under `~/.lightcode/skills/`
2. Add a `SKILL.md` file inside it describing the context or instructions

```
~/.lightcode/skills/
├── golang/
│   └── SKILL.md
└── docker/
    └── SKILL.md
```

You can also point `skills_path` in `config.json` to any other directory on your system.
