# CodingGame Technical Architecture

Technical specification for implementing the CodingGame strategy interface.

---

## Technology Stack

### GUI Framework: Ebitengine + Go

**Why Ebitengine:**
- Mature, stable API (minimal breaking changes between versions)
- Pure 2D focus - lightweight, no unnecessary 3D overhead
- Extensive documentation and examples
- Cross-platform (Linux first, others later)
- Compiles to single ~10MB binary

**Why Go:**
- Goroutines + channels are ideal for streaming subprocess output without blocking the game loop
- Highly LLM-friendly language (clear syntax, explicit error handling, massive training corpus)
- Fast compilation for quick iteration
- No complex async/await integration with game loops (unlike Rust + tokio)
- Simple dependency management with Go modules

**Key advantage:** The subprocess streaming requirement (reading `claude --output-format json` in the background while the UI remains responsive) maps directly to Go's concurrency primitives. A goroutine reads stdout, sends events over a channel, and the game loop checks the channel non-blocking each frame.

**Previous consideration:** Rust + SDL2 was initially planned but async integration with game loops is complex. Go's goroutines solve this elegantly.

### Claude Integration: JSON Streaming + Hooks

```
┌─────────────────────────────────────────────────────────────────┐
│                      CodingGame Process                         │
│                                                                 │
│  ┌──────────┐    ┌──────────────┐    ┌───────────────────┐     │
│  │ Ebitengine│◄───│  Game State  │◄───│  Claude Manager   │     │
│  │ Renderer │    │   Manager    │    │   (goroutine)     │     │
│  └──────────┘    └──────────────┘    └─────────┬─────────┘     │
│                                                │               │
└────────────────────────────────────────────────┼───────────────┘
                                                 │
                                                 │ spawn + stream
                                                 ▼
                                    ┌─────────────────────────┐
                                    │  claude --output-format │
                                    │         json            │
                                    │                         │
                                    │  (subprocess per agent) │
                                    └─────────────────────────┘
```

**Integration approach:**
- Spawn `claude` CLI as subprocess with `--output-format json`
- Parse streaming JSON events in real-time
- Tool calls (`tool_use` events) drive visualizations
- Hooks for pre/post tool execution (future: interception)

---

## Core Architecture

### Process Model

```
┌─────────────────────────────────────────────────────────────────┐
│                         Main Process                            │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────┐  │
│  │   Render Loop   │  │   Event Loop    │  │  Agent Pool    │  │
│  │   (60 fps cap)  │  │   (keyboard,    │  │                │  │
│  │                 │  │    mouse)       │  │  - Main Agent  │  │
│  │  - Draw map     │  │                 │  │  - Advisor 1   │  │
│  │  - Draw UI      │  │  - Input queue  │  │  - Advisor 2   │  │
│  │  - Animations   │  │  - Keybindings  │  │  - ...         │  │
│  └─────────────────┘  └─────────────────┘  └────────────────┘  │
│           │                    │                    │          │
│           └────────────────────┼────────────────────┘          │
│                                │                               │
│                                ▼                               │
│                    ┌─────────────────────┐                     │
│                    │     Game State      │                     │
│                    │                     │                     │
│                    │  - Map state        │                     │
│                    │  - Fog of war       │                     │
│                    │  - Prompt buffer    │                     │
│                    │  - Agent states     │                     │
│                    │  - Notifications    │                     │
│                    └─────────────────────┘                     │
└─────────────────────────────────────────────────────────────────┘
```

### Threading Model

- **Main thread**: SDL2 event loop + rendering
- **Agent threads**: One per Claude subprocess (main agent + advisors)
- **Communication**: Channel-based message passing (no shared mutable state)

```rust
// Pseudocode for agent communication
enum AgentMessage {
    ToolUse { tool: String, input: Value },
    ToolResult { tool: String, output: Value },
    TextDelta { text: String },
    TurnComplete { response: String },
    Error { message: String },
}

enum GameCommand {
    StartTurn { prompt: String },
    CancelTurn,
    ConsultAdvisor { advisor_id: String, query: String },
}
```

---

## Map System

### Two Primary Views

#### 1. Directory View
Filesystem hierarchy as a navigable map.

```
┌─────────────────────────────────────────────────────────────────┐
│                        src/                                     │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐    │
│  │░░░░░░░░░░░│  │███████████│  │██████░░░░░│  │███████████│    │
│  │  auth/    │  │   api/    │  │  models/  │  │  utils/   │    │
│  │  (fogged) │  │ (in ctx)  │  │ (partial) │  │ (in ctx)  │    │
│  │           │  │  8 files  │  │  5 files  │  │  12 files │    │
│  └───────────┘  └───────────┘  └───────────┘  └───────────┘    │
│                                                                 │
│  Legend: ███ = in context (revealed)                           │
│          ░░░ = fogged (not in context)                         │
└─────────────────────────────────────────────────────────────────┘
```

#### 2. Dataflow View (Belts)
Dependencies between files as animated connections.

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│    index.ts ◄════════╗                                          │
│        │             ║                                          │
│        │             ║                                          │
│        ▼             ║                                          │
│    router.ts ───────►║───► api/endpoints.ts                     │
│        │             ║            │                             │
│        │             ║            │                             │
│        ▼             ║            ▼                             │
│    middleware.ts ◄───╝     models/user.ts                       │
│                                   │                             │
│                                   ▼                             │
│                            utils/validate.ts                    │
│                                                                 │
│  Belt width = coupling strength                                 │
│  Animation = data flow direction                                │
└─────────────────────────────────────────────────────────────────┘
```

### Fog of War = Context Window

**Core principle:** Fog represents what Claude does NOT have in context.

| File State | Visual | Meaning |
|------------|--------|---------|
| Fogged | Dark/hidden | Not in Claude's context |
| Revealed | Visible | Claude has read this file |
| Stale | Faded | Was in context, may be summarized |
| Active | Highlighted | Currently being edited/viewed |

**Context tracking:**
- Track files via `Read` tool calls in JSON stream
- Mark as revealed when file is read
- Track token usage to estimate when files get summarized
- Visual indicator when approaching context limit

### Map Navigation

- **Pan**: Arrow keys / WASD / click-drag
- **Zoom**: +/- or scroll wheel (discrete levels, not continuous)
- **Select**: Enter on tile, or click
- **View toggle**: Tab to switch Directory ↔ Dataflow

**Interacting with fogged tiles:**
- Selecting a fogged file shows metadata (path, size, last modified)
- Option to queue "read this file" for next turn
- Cannot see contents until revealed

---

## Rendering Strategy

### Efficiency First

Target: **Smooth 30fps on integrated graphics** (Intel HD 4000 era)

**Techniques:**
1. **Dirty rectangle tracking**: Only redraw changed regions
2. **Tile caching**: Pre-render tiles to textures, composite
3. **LOD (Level of Detail)**: Simplified rendering when zoomed out
4. **Frame skipping**: Skip render frames under load, keep logic running
5. **Texture atlases**: Batch similar sprites

### Render Layers (bottom to top)

```
┌─────────────────────────────────────────┐
│  Layer 5: Notifications/Tooltips        │  (UI overlay)
├─────────────────────────────────────────┤
│  Layer 4: Prompt Window                 │  (always visible)
├─────────────────────────────────────────┤
│  Layer 3: Side Panels (Advisors, Missions)│
├─────────────────────────────────────────┤
│  Layer 2: Belts/Connections             │  (animated)
├─────────────────────────────────────────┤
│  Layer 1: Tiles (files/directories)     │  (cached)
├─────────────────────────────────────────┤
│  Layer 0: Background/Grid               │  (static)
└─────────────────────────────────────────┘
```

### Animation Budget

- **Belt flow**: Simple UV scroll, very cheap
- **Fog reveal**: Dissolve shader or tile-by-tile reveal
- **Tool activity**: Pulsing highlight on affected tiles
- **Turn processing**: Progress indicator, activity log scroll

---

## Agent Architecture

### Main Agent

The primary Claude instance that executes user prompts.

```
┌────────────────────────────────────────────────────────────┐
│                       Main Agent                           │
│                                                            │
│  Context: Full project context (up to 200k tokens)         │
│  Lifecycle: Persistent across session                      │
│  Triggers: User "end turn" with prompt                     │
│                                                            │
│  Responsibilities:                                         │
│  - Execute coding tasks                                    │
│  - Read/write files                                        │
│  - Run builds/tests                                        │
│  - Answer questions                                        │
└────────────────────────────────────────────────────────────┘
```

### Advisor Agents

Specialized Claude instances with focused contexts.

```
┌────────────────────────────────────────────────────────────┐
│                    Advisor Agent Pool                       │
│                                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │  Refactoring │  │   Security   │  │    Tests     │     │
│  │   Advisor    │  │   Advisor    │  │   Advisor    │     │
│  │              │  │              │  │              │     │
│  │ Focus: code  │  │ Focus: auth, │  │ Focus: test  │     │
│  │ smells,      │  │ injection,   │  │ coverage,    │     │
│  │ patterns     │  │ secrets      │  │ flakiness    │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                            │
│  Context: Smaller, domain-specific                         │
│  Lifecycle: Spawned on-demand, may persist                 │
│  Triggers: Manual consult OR background analysis           │
│                                                            │
│  Output: Notifications to main game state                  │
│  "Security Advisor: auth.ts line 42 has hardcoded secret" │
└────────────────────────────────────────────────────────────┘
```

**Advisor configuration (JSON):**
```json
{
  "advisors": [
    {
      "id": "refactoring",
      "name": "Refactoring Advisor",
      "icon": "wrench",
      "system_prompt": "You analyze code for refactoring opportunities...",
      "trigger": "on_file_change",
      "focus_patterns": ["**/*.ts", "**/*.rs"]
    },
    {
      "id": "security",
      "name": "Security Advisor",
      "icon": "shield",
      "system_prompt": "You audit code for security vulnerabilities...",
      "trigger": "manual",
      "focus_patterns": ["**/auth/**", "**/api/**"]
    }
  ]
}
```

**Advisor trigger modes:**
- `manual`: Only runs when user clicks "Consult"
- `on_file_change`: Runs when main agent modifies matching files
- `background`: Periodic analysis (configurable interval)

---

## Turn System

### The Core Loop

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│   ┌─────────┐      ┌─────────┐      ┌─────────┐      ┌───────┐ │
│   │  IDLE   │─────►│ COMPOSE │─────►│ EXECUTE │─────►│REVIEW │ │
│   │         │      │         │      │         │      │       │ │
│   └────▲────┘      └─────────┘      └─────────┘      └───┬───┘ │
│        │                                                 │     │
│        └─────────────────────────────────────────────────┘     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

IDLE:     Navigate map, review state, consult advisors
COMPOSE:  Writing prompt in prompt window
EXECUTE:  Claude processing (tool calls streaming)
REVIEW:   Viewing results, deciding next action
```

### Turn Execution Flow

1. **User composes prompt** in bottom prompt window
2. **User hits Enter** ("End Turn")
3. **Game state transitions** to EXECUTE
4. **Main agent spawned/continued** with prompt
5. **JSON stream parsed** in real-time:
   - `tool_use` → animate affected tiles, update fog
   - `text_delta` → show in response area
   - `tool_result` → update metrics (build pass/fail, etc.)
6. **Turn completes** → transition to REVIEW
7. **Advisors triggered** (if configured for `on_file_change`)
8. **User reviews** → back to IDLE

### Visualization During Execution

While Claude works, show:
- **Active file**: Highlight tile being read/written
- **Tool log**: Scrolling list of tool calls
- **Progress**: Approximate based on typical turn length
- **Cancel button**: Interrupt if going wrong direction

---

## Data Structures

### Core Types (Rust pseudocode)

```rust
struct GameState {
    map: MapState,
    agents: AgentPool,
    turn: TurnState,
    notifications: Vec<Notification>,
    config: GameConfig,
}

struct MapState {
    view_mode: ViewMode,  // Directory | Dataflow
    camera: Camera,       // position, zoom level
    tiles: HashMap<PathBuf, TileState>,
    connections: Vec<Connection>,  // for dataflow view
}

enum TileState {
    Fogged,
    Revealed {
        read_at: Instant,
        token_estimate: usize,
    },
    Stale {
        summarized_at: Instant,
    },
}

struct AgentPool {
    main: AgentHandle,
    advisors: HashMap<String, AdvisorHandle>,
}

struct AgentHandle {
    process: Child,
    tx: Sender<GameCommand>,
    rx: Receiver<AgentMessage>,
    state: AgentState,
}

enum TurnState {
    Idle,
    Composing { buffer: String },
    Executing { started: Instant, tool_log: Vec<ToolCall> },
    Reviewing { response: String },
}
```

---

## File Structure

```
codinggame/
├── Cargo.toml
├── src/
│   ├── main.rs              # Entry point, SDL2 init
│   ├── game/
│   │   ├── mod.rs
│   │   ├── state.rs         # GameState management
│   │   ├── turn.rs          # Turn state machine
│   │   └── config.rs        # Configuration loading
│   ├── render/
│   │   ├── mod.rs
│   │   ├── map.rs           # Map rendering (both views)
│   │   ├── ui.rs            # UI panels, prompt window
│   │   ├── animations.rs    # Belt flow, highlights
│   │   └── text.rs          # Text rendering utilities
│   ├── input/
│   │   ├── mod.rs
│   │   ├── keyboard.rs      # Keybinding system
│   │   └── mouse.rs         # Click/drag handling
│   ├── agents/
│   │   ├── mod.rs
│   │   ├── manager.rs       # Agent pool management
│   │   ├── claude.rs        # Claude subprocess wrapper
│   │   ├── parser.rs        # JSON stream parsing
│   │   └── advisor.rs       # Advisor-specific logic
│   └── map/
│       ├── mod.rs
│       ├── directory.rs     # Directory view logic
│       ├── dataflow.rs      # Dataflow/belt view logic
│       └── fog.rs           # Fog of war tracking
├── assets/
│   ├── fonts/
│   ├── sprites/
│   └── themes/
└── config/
    ├── default.toml         # Default configuration
    └── advisors.json        # Advisor definitions
```

---

## Open Questions

### Resolved
- **Framework**: SDL2 + Rust
- **Claude integration**: `claude --output-format json` + hooks
- **Advisor contexts**: Separate per-advisor, smaller focused contexts
- **Map views**: Directory view + Dataflow view (toggle)
- **Tool call interception**: Observation only (no interception). May add approval gates later.
- **Advisor triggers**: Default to `manual` for Phase 0. Add `on_file_change` later.
- **Dataflow source**: LSP-first approach (see below)
- **Asset style**: Minimal geometric (rectangles, lines, text). Theming/pixel art later.
- **Rendering target**: SDL2 GUI first, TUI port planned for later

### Dataflow Implementation Notes

LSP provides accurate dependency/reference data without reimplementing parsers:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Dataflow Data Sources                        │
│                                                                 │
│  Phase 0-1: LSP Integration                                    │
│  ├── TypeScript: tsserver                                       │
│  ├── Python: pylsp / pyright                                    │
│  ├── Rust: rust-analyzer                                        │
│  ├── Go: gopls                                                  │
│  └── (extensible per-language)                                  │
│                                                                 │
│  Future options:                                                │
│  ├── Tree-sitter: Fast static parsing, language-agnostic       │
│  ├── Build system: Bazel/Buck query for target dependencies    │
│  ├── Runtime tracing: Actual execution flow (Phase 4+)         │
│  └── Custom analyzers: Domain-specific dependency extraction   │
└─────────────────────────────────────────────────────────────────┘
```

LSP queries we'll use:
- `textDocument/references` - Find all references to a symbol
- `textDocument/definition` - Find where symbol is defined
- `textDocument/documentSymbol` - List symbols in a file

This builds the dependency graph: File A imports symbol from File B → belt from B to A.

### Resolved (continued)
- **Start screen**: Minimal start screen for "new game" flow (harness → model → project). If launched with directory arg, skip directly to game.
- **Input priority**: Keyboard-first with optional mouse support. Facilitates TUI port.
- **Filesystem scanning**: Yes, scan directory structure. Cache in `.codinggame/` within project.
- **Response display**: Responsive layout (see below)

---

## Start Screen

When launched without arguments, show a "New Game" flow:

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│                      C O D I N G G A M E                        │
│                                                                 │
│                        ┌─────────────┐                          │
│                        │  NEW GAME   │                          │
│                        └─────────────┘                          │
│                        ┌─────────────┐                          │
│                        │   CONTINUE  │                          │
│                        └─────────────┘                          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### New Game Flow

**Step 1: Select Harness**
```
┌─────────────────────────────────────────┐
│  SELECT HARNESS                         │
│                                         │
│  > Claude Code                          │
│    Codex                                │
│    Gemini                               │
│    Amp                                  │
│    OpenCode                             │
└─────────────────────────────────────────┘
```

**Step 2: Select Model** (options depend on harness)
```
┌─────────────────────────────────────────┐
│  SELECT MODEL                           │
│                                         │
│  > Sonnet 4                             │
│    Opus 4                               │
│    Haiku 4                              │
└─────────────────────────────────────────┘
```

**Step 3: Select Project**
```
┌─────────────────────────────────────────┐
│  SELECT PROJECT                         │
│                                         │
│  > Open existing (file picker)          │
│    Clone from GitHub                    │
│    Start new project                    │
│                                         │
│  ─────────────────────────────────────  │
│  RECENT:                                │
│    ~/Projects/CodingGame                │
│    ~/Projects/other-project             │
└─────────────────────────────────────────┘
```

**Shortcut**: `codinggame /path/to/project` skips directly to game.

---

## Responsive Layout

Claude's response display adapts to window dimensions:

### Wide Layout (width > 1200px)
```
┌────────────────────────────────────────────────────────────────────────┐
│  Resources: [ctx: 45k/200k] [cost: $0.12] [build: ✓]                   │
├────────────┬───────────────────────────────────────────┬───────────────┤
│            │                                           │               │
│  ADVISORS  │              MAP VIEW                     │   RESPONSE    │
│            │                                           │               │
│  [Refactor]│         (directory/dataflow)              │  Claude's     │
│  [Security]│                                           │  output       │
│  [Tests]   │                                           │  streams      │
│            │                                           │  here         │
│            │                                           │               │
├────────────┴───────────────────────────────────────────┴───────────────┤
│  > Enter your prompt here...                                    [END]  │
└────────────────────────────────────────────────────────────────────────┘
```

### Narrow Layout (width < 1200px)
```
┌──────────────────────────────────────────┐
│  [ctx: 45k] [cost: $0.12] [build: ✓]     │
├──────────────────────────────────────────┤
│                                          │
│              MAP VIEW                    │
│                                          │
│         (directory/dataflow)             │
│                                          │
├──────────────────────────────────────────┤
│  Response: Claude's output here...       │
│  (scrollable, collapsed by default)      │
├──────────────────────────────────────────┤
│  > Prompt...                      [END]  │
└──────────────────────────────────────────┘
```

### Panel Behavior
- **Wide**: Response panel is right sidebar (vertical scroll)
- **Narrow**: Response panel is horizontal bar below map (expandable)
- **GUI mode**: Panels can be dragged/resized by user
- **Advisors panel**: Collapses to icons in narrow mode

---

## Filesystem Scanning

### Initial Scan
On project open, scan directory structure:

```rust
struct ProjectScan {
    root: PathBuf,
    files: Vec<FileEntry>,
    directories: Vec<DirEntry>,
    gitignore: Option<GitIgnore>,  // respect .gitignore
    scan_time: Instant,
}

struct FileEntry {
    path: PathBuf,
    size: u64,
    modified: SystemTime,
    file_type: FileType,  // source, config, asset, etc.
}
```

### Caching
Store scan results in `.codinggame/` within project:

```
project/
├── .codinggame/
│   ├── scan_cache.json      # file structure cache
│   ├── fog_state.json       # which files are revealed
│   ├── dataflow_cache.json  # dependency graph cache
│   └── session.json         # last session state
└── src/
    └── ...
```

### Incremental Updates
- Watch for filesystem changes (inotify on Linux)
- Update tile map incrementally
- Invalidate dataflow cache when source files change

---

## Phase 0 Implementation Targets

Minimal viable demo to validate the concept:

### 0.1 Window & Rendering
- [ ] SDL2 window initialization
- [ ] Basic render loop (60fps cap, dirty rect tracking)
- [ ] Text rendering (monospace font)
- [ ] Responsive layout detection (wide vs narrow)

### 0.2 Start Screen
- [ ] Title screen with NEW GAME / CONTINUE
- [ ] Harness selection menu (Claude Code only initially)
- [ ] Model selection menu
- [ ] Project selection (file picker, recent list)
- [ ] CLI arg parsing (`codinggame /path` skips to game)

### 0.3 Filesystem & Map
- [ ] Directory scanning (respect .gitignore)
- [ ] Tile generation from file structure
- [ ] Directory view rendering (rectangles + labels)
- [ ] Pan navigation (arrow keys, WASD)
- [ ] Zoom levels (at least 2: overview and detail)
- [ ] Cache to `.codinggame/`

### 0.4 Prompt & Turn System
- [ ] Prompt window (text input with cursor)
- [ ] "End Turn" button/Enter key
- [ ] Turn state machine (IDLE → COMPOSE → EXECUTE → REVIEW)
- [ ] Response display area (right panel or bottom bar)

### 0.5 Claude Integration
- [ ] Spawn `claude --output-format json` subprocess
- [ ] Parse JSON stream in real-time
- [ ] Extract tool calls from stream
- [ ] Display tool log during execution
- [ ] Basic fog reveal on `Read` tool calls

### 0.6 Polish
- [ ] Keyboard navigation (Tab between panels, arrow keys in menus)
- [ ] Basic mouse support (click to select, drag to pan)
- [ ] Error handling (subprocess failures, parse errors)

**Success criteria**:
1. Launch game, go through start screen, select project
2. See directory map with files as tiles
3. Type prompt, hit Enter
4. Watch Claude's tool calls stream in real-time
5. See tiles reveal as Claude reads files
6. See response appear in response panel

---

## Future Phases (Summary)

| Phase | Focus | Key Deliverables |
|-------|-------|------------------|
| 1 | Dataflow View | LSP integration, belt rendering, dependency graph |
| 2 | Buildings & Units | Build system adapters, test visualization |
| 3 | Advisors | Multi-agent spawning, advisor UI, insights |
| 4 | Visual Debugging | Runtime tracing, data flow animation |
| 5 | Missions | Issue tracker integration, mission panel |
| 6 | Production Realm | Deployment visualization, metrics |
| 7 | Multi-Agent | Concurrent agents, per-agent fog |
| 8 | Polish | Themes, sound, plugin system, TUI port |

---

*Next: Detailed component specs for each subsystem*
