

<h1 align="center">Lightcode</h1>
<p align="center">A lightweight terminal coding agent written in Go</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img alt="Bubble Tea" src="https://img.shields.io/badge/TUI-Bubble%20Tea-FFB300?style=flat-square" />
  <img alt="OpenAI Compatible" src="https://img.shields.io/badge/API-OpenAI%20Compatible-10A37F?style=flat-square" />
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/License-MIT-black?style=flat-square" /></a>
</p>

<p align="center">
  <video src="https://github.com/user-attachments/assets/af16a300-a84a-41b2-9346-e3940e087986" controls width="600"></video>
</p>
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

### Prerequisites
Before installing, make sure the required system dependencies are present. Run the following on: 
1. Debian/Ubuntu-based systems:
```bash
sudo apt install libx11-dev libxext-dev libxi-dev libsqlite3-dev
```
2. Fedora/RHEL/CentOS (dnf/yum):
```bash
sudo dnf install libX11-devel libXext-devel libXi-devel sqlite-devel
```
3. Arch Linux / Manjaro (pacman)
```bash
sudo pacman -S libx11 libxext libxi sqlite
```
4. openSUSE (zypper):
```bash
sudo zypper install libX11-devel libXext-devel libXi-devel sqlite3-devel
```
<br>

> **Note:** These libraries are required for native display and database support. Installation will fail without them.

<br> 


### <u>Final Installation</u>

**Option 1 — Install script (prebuilt binary)**
```bash
curl -fsSL https://raw.githubusercontent.com/Kartik-2239/lightcode/main/install.sh | bash
```
Pin a specific version or pick the install dir:
```bash
curl -fsSL https://raw.githubusercontent.com/Kartik-2239/lightcode/main/install.sh | bash -s -- --version v1.2.3 --bin-dir ~/.bin
```

**Option 2 — `go install` (build from source)**
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

Run `/login` in the TUI and choose `codex` to start the login flow for codex.
After login, run `/models` to use available codex models or add others in the config files.

If you have codex logged in on your coputer then you can select codex defined config in the onboarding menu.

### GitHub Copilot OAuth

Run `/login` in the TUI and choose `copilot` to start login flow for copilot.
After login, run `/models` to use available copilot models or add others in the config files.


Run `/logout` and choose `copilot` to remove the saved token and any selected Copilot model.

Copilot model access uses GitHub Copilot service endpoints and can change outside Lightcode's control. Treat `~/.lightcode/auth.json` like a password and do not commit or share it.


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


## Contributing

Contributions are welcome! See [CONTRIBUTING.md](./CONTRIBUTING.md).
