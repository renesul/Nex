# Nex

**Intelligent WhatsApp secretary** — an AI agent on your personal WhatsApp number, powered by any OpenAI-compatible LLM, with a full web UI for configuration and monitoring.

---

## Highlights

- **WhatsApp via QR code** — uses your personal number, no Business API needed
- **Any OpenAI-compatible LLM** — OpenAI, Groq, Ollama, Together, LM Studio, etc.
- **16 built-in tools** + custom API tools + MCP protocol support
- **Knowledge base** with FTS5 full-text search + embeddings (RAG)
- **Up to 10 agents** with independent personality, model, and provider + agent chaining
- **Guardrails** — whitelist/blacklist, anti-prompt-injection, PII filtering
- **Message debounce** — groups rapid messages before sending to AI
- **Session management** with automatic summaries (prevents hallucination)
- **Full web UI** — config, conversations, logs, knowledge base, reports, chat
- **Multi-user auth** — admin and user roles, bcrypt, persistent sessions
- **WhatsApp groups** (optional, configurable per group)
- **Scheduled messages** — hourly, daily, weekly, monthly, cron expressions
- **MCP server** — exposes all tools via SSE protocol
- **External databases** — query MySQL and PostgreSQL from the AI

---

## Quick Start

```bash
git clone <repo-url>
cd Nex
./build.sh
./nex
```

1. Open **http://localhost:8080**
2. Login with `admin` / `admin123`
3. Set your AI provider API key and base URL
4. Scan the WhatsApp QR code
5. Done — start chatting

---

## Requirements

| Requirement | Details |
|---|---|
| Go | 1.25+ |
| CGO | Enabled (`gcc` / `build-essential` must be installed) |
| OS | Linux, macOS (Windows via WSL) |

---

## Configuration

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `~/.nex/` | Data directory (databases, logs) |

Both are optional. Set them in `.env` or as environment variables.

**Everything else** (AI provider, system prompt, tools, guardrails, agents, etc.) is configured through the web UI and stored in SQLite.

---

## Built-in Tools

| Tool | Description |
|---|---|
| `get_datetime` | Current date, time, and day of week |
| `create_task` | Create a task/reminder for the contact |
| `list_tasks` | List tasks, optionally filtered by status |
| `complete_task` | Mark a task as completed |
| `search_knowledge` | Search the knowledge base (RAG) |
| `calculate` | Evaluate math expressions (sqrt, abs, round, etc.) |
| `search_web` | Web search via DuckDuckGo |
| `fetch_url` | Fetch and extract text from a URL |
| `weather` | Current weather for any location |
| `save_note` | Save persistent notes about a contact |
| `get_notes` | Retrieve saved notes |
| `currency` | Currency conversion with live rates |
| `schedule_message` | Schedule a message with optional recurrence |
| `list_scheduled` | List pending scheduled messages |
| `cancel_scheduled` | Cancel scheduled messages |
| `query_database` | Run read-only SQL queries (local + external DBs) |

You can also add **custom API tools** (any REST endpoint) and **MCP tools** (via SSE transport) through the web UI.

---

## Architecture

### Message Flow

```
WhatsApp Message
    |
    v
 Filter (private/group, text-only, not from self)
    |
    v
 Debounce (group rapid messages)
    |
    v
 Guardrails Pre-filter
    |
    v
 Agent Lookup (per-contact routing)
    |
    v
 Session (create/resume + auto-summary)
    |
    v
 RAG (FTS5 + embeddings hybrid search)
    |
    v
 AI (LLM call + tool execution loop)
    |
    v
 Guardrails Post-filter
    |
    v
 WhatsApp Response
```

### Project Structure

```
cmd/nex/main.go              Entry point, wiring, signal handling, migrations

app/                          Business logic
  types/types.go              Shared types (Message, Agent, Contact, etc.)
  ai/ai.go                   OpenAI-compatible client (reply, summarize, embed)
  memory/memory.go            Sessions, history, summaries, token budget
  rag/rag.go                  Knowledge base with FTS5 + embeddings
  guardrails/guardrails.go    Pre/post-AI message filtering
  tools/tools.go              Function calling registry, built-in + custom tools
  tools/mcp.go                MCP client + MCP server
  pipeline/pipeline.go        Message processing pipeline

internal/                     Infrastructure
  config/config.go            Config struct + SQLite key-value store
  logger/logger.go            JSON file (always) + SQLite (debug mode)
  auth/auth.go                Multi-user auth (bcrypt, sessions, middleware)
  whatsapp/whatsapp.go        WhatsApp connection (QR, send, receive)
  debounce/debounce.go        Message grouping per contact
  web/web.go                  HTTP routes and API handlers

templates/                    HTML pages (inline CSS/JS, no build step)
```

---

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go |
| Database | SQLite with FTS5 |
| WhatsApp | whatsmeow (Web multidevice protocol) |
| AI Client | go-openai (any OpenAI-compatible API) |
| MCP | mcp-go (SSE transport) |
| Auth | bcrypt (golang.org/x/crypto) |
| External DBs | MySQL (go-sql-driver), PostgreSQL (lib/pq) |

---

## Development

```bash
./build.sh               # Build binary
./run.sh                  # Build and run
./test.sh                 # Run all tests
./test.sh Config          # Run tests matching "Config"
```

### Databases

Stored in `~/.nex/` (or `DB_PATH`):

| File | Purpose |
|---|---|
| `nex.db` | App data (config, messages, summaries, knowledge, tasks, tools, agents, users, sessions, logs) |
| `whatsapp.db` | WhatsApp session store (managed by whatsmeow) |

---

## License

Private project.
