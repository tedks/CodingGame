# Agent Support Implementation Plan

## Goal

Enable CodingGame to dogfood itself by integrating with real AI coding agents, starting with Claude Code and expanding to Codex, Gemini, Amp, and OpenCode.

## Implementation Status

| Phase | Name | Status | Beads Task |
|-------|------|--------|------------|
| 1 | Harness Framework | ✅ Complete | CodingGame-u70.1 |
| 2 | Claude Code Harness | ✅ Complete | CodingGame-u70.2 |
| 3 | Start Screen UI | ✅ Complete | CodingGame-u70.3 |
| 4a | OpenAI Codex CLI | Not Started | CodingGame-u70.4 |
| 4b | Google Gemini CLI | Not Started | CodingGame-u70.5 |
| 4c | Sourcegraph Amp | Not Started | CodingGame-u70.6 |
| 4d | OpenCode | Not Started | CodingGame-u70.7 |
| 5 | Dogfooding & Polish | Not Started | CodingGame-u70.8 |

**Advisor Integration**: ✅ Complete (as part of Phase 2-3)

## Research Summary

### Agent Landscape (2025-2026)

| Agent | CLI | JSON Output | MCP Support | Auth Method |
|-------|-----|-------------|-------------|-------------|
| **Claude Code** | `claude` | `--output-format json` | Yes (300+ integrations) | OAuth, API key |
| **Codex CLI** | `codex` | `--output-format json` | AGENTS.md standard | Browser auth, API key |
| **Gemini CLI** | `gemini` | `--output-format json` | Yes (Oct 2025) | Google account, API key |
| **Amp** | `amp` | AgentAPI | MCP servers | Sourcegraph auth |
| **OpenCode** | `opencode` | JSON mode | MCP + LSP | Various providers |

### Common Integration Points

1. **JSON Streaming** - All agents support JSON output for tool calls
2. **MCP Protocol** - Emerging standard for tool integration (all agents support)
3. **Hooks** - Claude Code has 8 hook events, others have similar systems
4. **File Operations** - Read/Write/Edit are universal tool types

## Architecture

### Harness Interface (Implemented)

```go
// internal/harness/harness.go

// Harness represents an interface to an AI coding agent CLI.
type Harness interface {
    // Identity
    Name() string
    Version() string

    // Lifecycle
    Start(ctx context.Context, config Config) error
    Stop() error
    IsRunning() bool

    // Communication
    SendPrompt(prompt string) error
    Events() <-chan Event

    // Capabilities
    Capabilities() Capabilities
}

// Capabilities describes what features a harness supports
type Capabilities struct {
    SupportedModels   []Model
    SupportsHooks     bool
    SupportsMCP       bool
    SupportsStreaming bool
    SupportsResume    bool
}
```

### Event Types (Implemented)

```go
// internal/harness/event.go

type EventType string

const (
    // Tool events
    EventToolUse    EventType = "tool_use"
    EventToolResult EventType = "tool_result"

    // Text events
    EventText         EventType = "text"
    EventTurnStart    EventType = "turn_start"
    EventTurnComplete EventType = "turn_complete"

    // Derived events (inferred from tool use)
    EventFileRead    EventType = "file_read"
    EventFileWrite   EventType = "file_write"
    EventFileEdit    EventType = "file_edit"
    EventBuildRun    EventType = "build_run"
    EventTestRun     EventType = "test_run"
    EventSubagentRun EventType = "subagent_run"

    // Status events
    EventError   EventType = "error"
    EventWarning EventType = "warning"
)

// Event represents a unified event from any harness
type Event struct {
    Type       EventType
    Timestamp  time.Time
    Tool       string
    ToolInput  map[string]interface{}
    ToolOutput map[string]interface{}
    Text       string
    Error      error
    Raw        map[string]interface{}
    Source     string
}
```

### Package Structure (Implemented)

```
internal/harness/
├── harness.go          # Interface, BaseHarness, Capabilities
├── harness_test.go     # BaseHarness tests
├── event.go            # Event types, EventBuilder
├── event_test.go       # Event tests (including SafeFilePath)
├── config.go           # Config with validation, defaults
├── config_test.go      # Config tests
├── registry.go         # Thread-safe harness registry
├── registry_test.go    # Registry tests
├── BUILD.bazel
└── claude/
    ├── claude.go       # ClaudeHarness implementation
    ├── claude_test.go  # Harness tests
    ├── parser.go       # JSON stream parser
    ├── parser_test.go  # Parser tests
    └── BUILD.bazel
```

### Config Schema (Implemented)

```go
// internal/harness/config.go

type Config struct {
    WorkingDir    string            // Project directory (required)
    Model         string            // Model to use
    SystemPrompt  string            // Optional system prompt
    MaxTokens     int               // Optional max tokens
    MCPServers    []MCPServer       // MCP server configs
    Env           map[string]string // Environment variables
    Verbose       bool              // Enable verbose output
    ResumeSession string            // Session to resume
    AdvisorMode   string            // Advisor ID if running as advisor
}

// Validate checks configuration and returns error for:
// - Empty WorkingDir
// - Dangerous environment variables (LD_PRELOAD, etc.)
```

## Implementation Phases

### Phase 1: Harness Framework ✅ COMPLETE

**Goal**: Create the harness abstraction and registry

**Completed**:
1. ✅ Defined `Harness` interface with context-aware Start method
2. ✅ Created unified `Event` types with builder pattern
3. ✅ Implemented thread-safe `Registry` for harness factories
4. ✅ Added `Config` schema with security validation
5. ✅ Added `SafeFilePath()` for path traversal protection
6. ✅ Created `BaseHarness` with thread-safe state management
7. ✅ Comprehensive unit tests

**Key Design Decisions**:
- Context parameter in Start() for cancellation support
- EventBuilder for fluent event construction
- Registry stores factories, not instances (supports multi-harness)
- Security: blocks dangerous env vars (LD_PRELOAD, etc.)

### Phase 2: Claude Code Harness ✅ COMPLETE

**Goal**: Fully functional Claude Code integration

**Completed**:
1. ✅ Subprocess spawning with `claude --output-format json --print`
2. ✅ JSON stream parsing with bufio.Scanner
3. ✅ Event type inference from tool names
4. ✅ Process monitoring with monitorProcess goroutine
5. ✅ Graceful shutdown with context cancellation
6. ✅ sync.Once for safe channel closing
7. ✅ WaitGroup tracking for reader goroutines
8. ✅ Version detection via `claude --version`

**Concurrency Design**:
```
Goroutines:
- readOutput: reads stdout, exits when pipe closes or done signaled
- readErrors: reads stderr, exits when pipe closes or done signaled
- monitorProcess: waits for process exit, owns cmd.Wait()

Channels:
- events: buffered(100), created by NewBaseHarness, closed by sync.Once
- done: unbuffered, signals shutdown to reader goroutines

Shutdown sequence:
1. Close done channel (unblocks select in readers)
2. Cancel context (signals subprocess)
3. Close stdin (EOF to subprocess)
4. Close stdout/stderr (unblocks scanners)
5. Wait for monitorProcess
6. Wait for reader goroutines
7. Close events channel (sync.Once)
```

**JSON Events Parsed**:
```json
{"type": "tool_use", "tool": "Read", "input": {"file_path": "/path"}}
{"type": "tool_result", "tool": "Read", "output": {"content": "..."}}
{"type": "text", "text": "Let me analyze..."}
{"type": "assistant", "content": [{"type": "tool_use", ...}]}
```

### Phase 3: Start Screen UI ✅ COMPLETE

**Goal**: New Game flow for harness/model/project selection

**Completed**:
1. ✅ Dynamic harness menu from registry
2. ✅ Dynamic model menu based on selected harness
3. ✅ Harness availability detection (IsInstalled)
4. ✅ Visual indicators for unavailable harnesses
5. ✅ Model menu updates when harness changes

**Implementation**:
- StartScreen receives Registry via SetHarnessRegistry()
- buildHarnessMenu() iterates registry.Available()
- Disabled items shown for uninstalled harnesses
- updateModelMenuForHarness() refreshes model options

### Advisor Integration ✅ COMPLETE

**Goal**: Connect advisor system to harness for subagent execution

**Completed**:
1. ✅ Added HarnessName, HarnessModel to advisor Config
2. ✅ Pool.SetHarnessRegistry() for harness access
3. ✅ Pool.RunAdvisor() creates harness, sends prompt, collects events
4. ✅ Pool.RunAdvisorAsync() with proper WaitGroup tracking
5. ✅ Token usage tracking from events
6. ✅ Error propagation to advisor state

**Key Design**: Advisors are harness-agnostic. An advisor can use:
- The same harness as the main agent (e.g., Claude advisor in Claude project)
- A different harness (e.g., Claude security advisor in a Codex project)

### Phase 4: Additional Harnesses (Not Started)

**Goal**: Support for Codex, Gemini, Amp, and OpenCode

#### 4a: OpenAI Codex CLI
- Install: `npm i -g @openai/codex`
- Auth: Browser or `OPENAI_API_KEY`
- JSON output mode
- Parse similar event structure

#### 4b: Google Gemini CLI
- Install: `npm i -g @google/gemini-cli`
- Auth: Google account or `GOOGLE_API_KEY`
- MCP server support
- Free tier: 1000 req/day

#### 4c: Sourcegraph Amp
- Install: `npm i -g @sourcegraph/amp`
- Auth: Sourcegraph token
- AgentAPI integration
- MCP server configuration

#### 4d: OpenCode
- Install: `npm i -g opencode-ai`
- Multi-provider support
- MCP + LSP integration
- Built-in agents (build, plan)

### Phase 5: Dogfooding & Polish (Not Started)

**Goal**: Use CodingGame to develop CodingGame

**Tasks**:
1. End-to-end testing with real agents
2. Performance optimization for JSON parsing
3. Error handling and recovery
4. Agent status indicators in UI
5. Context window visualization (token usage)
6. Cost tracking integration

**Success Criteria**:
- Can launch CodingGame, select Claude Code, open CodingGame project
- Can type prompts and see Claude's tool calls visualized
- Fog of war reveals as files are read
- Can switch between agents mid-session

## Technical Considerations

### JSON Parsing Performance

Claude harness uses bufio.Scanner with configurable buffer:

```go
const (
    scannerInitBufSize = 64 * 1024   // 64KB initial
    scannerMaxBufSize  = 1024 * 1024 // 1MB max
)

scanner := bufio.NewScanner(stdout)
buf := make([]byte, 0, scannerInitBufSize)
scanner.Buffer(buf, scannerMaxBufSize)
```

### MCP Integration

MCP (Model Context Protocol) is becoming the standard. Consider:
- CodingGame could expose its own MCP server
- Agents could use CodingGame as a tool source
- Enables bidirectional integration

### Error Recovery

Agents can crash, hang, or return errors:
- Process monitoring via monitorProcess goroutine
- Error events sent on crash detection
- Context cancellation for timeout handling
- sync.Once prevents channel double-close

### Multi-Agent Support

Design supports multi-agent scenarios:
- Registry creates independent harness instances
- Each advisor can have its own harness
- Concurrent execution via RunAdvisorAsync

## Dependencies

### External
- Claude Code CLI (`claude`)
- Codex CLI (`codex`) - not yet implemented
- Gemini CLI (`gemini`) - not yet implemented
- Amp CLI (`amp`) - not yet implemented
- OpenCode CLI (`opencode`) - not yet implemented

### Internal
- `internal/mapview` - Fog of war integration
- `internal/tile` - File state tracking
- `internal/ui` - Menu and scene system
- `internal/game` - Game state management
- `internal/advisor` - Subagent pool with harness integration

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Agent CLI APIs change | Abstract behind harness interface, version detection |
| Different JSON schemas | Per-harness parser with normalization |
| Auth complexity | Support multiple auth methods per harness |
| Performance with large outputs | Streaming parser, rate limiting, buffer management |
| Agent availability | Graceful degradation, clear error messages |
| Concurrency bugs | Documented goroutine/channel ownership, sync.Once |

## Sources

- [Claude Code Hooks Reference](https://docs.claude.com/en/docs/claude-code/hooks)
- [OpenAI Codex CLI](https://developers.openai.com/codex/cli/)
- [Gemini CLI](https://developers.google.com/gemini-code-assist/docs/gemini-cli)
- [Amp by Sourcegraph](https://ampcode.com/manual)
- [OpenCode](https://opencode.ai/docs/cli/)
