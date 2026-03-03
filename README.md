# Mortis

**A personal AI gateway written in Go. Multi-provider LLM support with a simple, self-contained binary.**

Inspired by [Moltis](https://github.com/moltis-org/moltis) — a local-first AI gateway that sits between you and multiple LLM providers.

## Features

- **Multi-provider LLM support** — OpenAI, Anthropic (Claude), and Local LLMs (Ollama, LM Studio) through a provider interface
- **Streaming responses** — Real-time token streaming for a responsive user experience
- **Web gateway** — HTTP and WebSocket server with a built-in web UI
- **Session persistence** — JSONL-backed conversation history and session management
- **Configuration** — TOML-based configuration with sensible defaults
- **Provider abstraction** — Easy to add new LLM providers via a simple interface

## Installation

```bash
# Clone and build
git clone https://github.com/Inokinoki/mortis.git
cd mortis
go build -o mortis ./cmd/mortis

# Run
./mortis
```

On first run, Mortis creates a default configuration file at `~/.mortis/mortis.toml`.

## Quick Start

1. **Install and run**
   ```bash
   go build -o mortis ./cmd/mortis
   ./mortis
   ```

2. **Open the web UI**
   Navigate to `http://localhost:13131`

3. **Configure providers**
   Edit `~/.mortis/mortis.toml` to add your API keys:
   ```toml
   [providers.openai]
   type = "openai"
   apiKey = "sk-..."
   enabled = true

   [providers.anthropic]
   type = "anthropic"
   apiKey = "sk-ant-..."
   enabled = true
   ```

4. **Start chatting**
   Refresh the web UI and start chatting!

## Configuration

Mortis uses a TOML configuration file at `~/.mortis/mortis.toml` (customizable via `--config-dir` or `MORTIS_CONFIG_DIR`).

### Server Configuration

```toml
[server]
host = "127.0.0.1"
port = 13131
tlsDisabled = false
readTimeout = 30
writeTimeout = 30
```

### Gateway Configuration

```toml
[gateway]
name = "Mortis"
emoji = "🤖"
defaultSessionId = "default"
defaultProvider = "openai"
defaultModel = "gpt-4o"
agentTimeout = 600
maxToolIterations = 25
messageQueueMode = "followup"
toolResultSanitizationLimit = 52428
```

### Provider Configuration

```toml
[providers.openai]
type = "openai"
apiKey = "sk-..."
model = "gpt-4o"
models = ["gpt-4o", "gpt-4o-mini", "o1-preview"]
enabled = true

[providers.anthropic]
type = "anthropic"
apiKey = "sk-ant-..."
model = "claude-sonnet-4-20250514"
models = ["claude-sonnet-4-20250514", "claude-3-haiku-20250318", "claude-3-opus-20250229"]
enabled = true

[providers.local]
type = "local"
enabled = true
```

### Session Configuration

```toml
[session]
dataDir = "~/.mortis/sessions"
autoCompactEnabled = true
compactThreshold = 0.95
historyLimit = 200
```

## Architecture

Mortis is organized as a Go module with the following packages:

| Package | Description |
|---------|-------------|
| `cmd/mortis` | Command-line entry point |
| `pkg/protocol` | WebSocket/RPC protocol definitions |
| `pkg/config` | Configuration management |
| `pkg/provider` | LLM provider integrations |
| `pkg/gateway` | HTTP/WebSocket server |
| `pkg/session` | Session persistence |

## Project Structure

```
mortis/
├── cmd/
│   └── mortis/
│       └── main.go              # Entry point
├── pkg/
│   ├── protocol/               # Protocol types (frames, events, RPC)
│   ├── config/                 # Configuration management
│   ├── provider/               # LLM provider interfaces
│   │   ├── openai.go          # OpenAI provider
│   │   ├── anthropic.go        # Anthropic provider
│   │   ├── local.go            # Local LLM provider
│   │   └── types.go            # Provider interfaces
│   ├── gateway/                 # Gateway server
│   │   └── server.go            # HTTP/WebSocket server
│   └── session/                 # Session management
│       └── types.go            # Session types
├── web/
│   └── ui/
│       ├── index.html            # Web UI
│       └── assets/
│           └── app.js              # Web UI logic
├── go.mod                        # Go module definition
└── README.md                     # This file
```

## Protocol

Mortis uses a WebSocket-based protocol compatible with Moltis protocol version 3.

### Frame Types

- **Request** (`type: "req"`) — Client → gateway RPC call
- **Response** (`type: "res"`) — Gateway → client RPC result
- **Event** (`type: "event"`) — Gateway → client server-push

### RPC Methods

| Method | Description |
|--------|-------------|
| `chat.send` | Send a chat message |
| `chat.history` | Get session history |
| `session.list` | List all sessions |
| `session.create` | Create a new session |
| `session.delete` | Delete a session |
| `provider.list` | List available providers |
| `provider.configure` | Configure a provider |
| `config.get` | Get configuration |
| `config.set` | Set configuration |

### Events

| Event | Description |
|-------|-------------|
| `gateway.start` | Gateway started |
| `chat.thinking` | Agent is thinking |
| `chat.textDelta` | Text delta (streaming) |
| `chat.toolStart` | Tool call started |
| `chat.toolEnd` | Tool call ended |
| `chat.done` | Chat response complete |

## Providers

### OpenAI

Supports GPT-4o, GPT-4o Mini, o1, and more.

Features:
- Completion (non-streaming)
- Streaming
- Tool calling

### Anthropic

Supports Claude Sonnet 4, Claude 3.5 Haiku, and more.

Features:
- Completion (non-streaming)
- Streaming
- Tool calling
- Vision

### Local LLM

Supports Ollama, LM Studio, and other local LLM servers.

Features:
- Completion (non-streaming)
- Streaming

## License

MIT

## Acknowledgments

- [Moltis](https://github.com/moltis-org/moltis) — Inspiration and protocol reference
- [Gin](https://github.com/gin-gonic/gin) — HTTP framework
- [Gorilla WebSocket](https://github.com/gorilla/websocket) — WebSocket implementation
