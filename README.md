# Lightcode

Lightcode is a **coding agent** written in Go. It supports all the llm providers that support OpenAi Api format.

## Requirements

- [Go](https://go.dev/dl/) **1.25+**
- An API key of any provider and their Base Url

## Configuration

Create a `.env` in the root of the project and set the values.

```bash
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://...
SKILL_PATH=path/to/skill/folder
API_URL=http://localhost:8080
```

## Quick start

Run the **API server** (by default listens on **`:8080`**):

```bash
go run ./cmd/server/main.go
```

In another terminal, run the Bubble tea **TUI client**:

```bash
go run ./cmd/tui/main.go
```

The agent streams responses over Server-Sent Events while tool calls and file operations run on the server side.

## What’s inside

| Piece | Role |
|--------|------|
| `cmd/server` | HTTP API: sessions, messages, streaming chat completion |
| `cmd/tui` | Bubble Tea frontend that calls the API |
| `internal/server/agent` | Agent loop, message history, tool execution |
| `internal/server/tools` | `read_file`, `write_file`, `edit`, `bash`, `grep`, `glob`, `list_dir`, `web_fetch`, `skill` |
| `internal/server/db` | GORM + SQLite (`lightcode.db`) for sessions and messages |

---

## Todo

- [x] copy paste multiple lines into a [ paste #1 13 lines ]
- [x] better tool and thinking formating
- [x] Skills
- [x] grep tool
- [x] arrow key should bring previous text // scrapped
- [x] first make the ui work
- [x] UI upgrades
- [ ] MCP
- [ ] todo tool
- [ ] question tool # in line 105 of agent.go. add a new role like question when the tc.name = question_tool and handle that in the tui/client.go
