# CodingGame: Strategy Game Interface for Claude Code

A Civilization/Call to Power II/Factorio-inspired UI wrapper for Claude Code, transforming software development into an intuitive strategy game experience.

## Core Philosophy

This is a **game interface to coding**, not a game about coding. See [PHILOSOPHY.md](PHILOSOPHY.md) for detailed principles.

Key tenets:
- **Everything is real**: No fake bonuses, HP/ATK stats, or artificial progression. Build times are actual build times. Test results are actual results.
- **Descriptive, not prescriptive**: The interface shows what exists in your project, not what you've "unlocked."
- **Game metaphors for real activities**: Buildings = build targets, Units = tests, Advisors = subagents, Belts = data flow.

---

## Visual Metaphors & Mapping

### The Map: Codebase as Territory

The main view is a strategic map of your codebase.

#### Tile System

Each tile represents a file or directory. Tiles are arranged in a zoomable, pannable 2D space.


**Tile Properties:**


#### Zoom Levels

| Level | Name | Shows | Interaction |
|-------|------|-------|-------------|
| 1 | World | Repository root, top-level dirs | Click to zoom |
| 2 | Region | Package/module clusters | See building outlines |
| 3 | City | Individual packages with buildings | See unit production |
| 4 | Street | Files within a package | Double-click opens editor |
| 5 | Interior | Single file contents | Syntax-highlighted code view |


#### Fog of War Mechanics

Files start in fog until Claude analyzes them:


#### Navigation & Interaction


#### Layout Algorithms

The map can use different layout strategies:


#### Minimap

A small overview map in the corner:



Double-clicking any file tile opens it in an integrated editor.

### Buildings: Bazel Packages

Packages are your production infrastructure. Each BUILD file defines a "building" that produces units (targets).

#### Building Visualization


#### Building Data Model


#### Building Metrics

Buildings display **real metrics** from your build system:

- **Build time**: Actual duration of last build, average, trend
- **Cache hit rate**: Real cache statistics from your build system
- **Success rate**: Actual pass/fail history
- **Dependency count**: Real incoming/outgoing dependencies
- **Last built**: Timestamp of most recent build

No fake "levels" or "bonuses" - the visualization reflects actual build system state.

#### Build History

Shows actual success/failure timeline:


#### Production Queue

Buildings produce units when builds/tests run:


#### Building Upgrades

Players can invest in upgrading buildings:


#### Multi-Build-System Support

Buildings adapt to different build systems:


### Units: Tests as Combatants

Tests are visualized as units that "fight" to pass - a PvE (player vs environment) metaphor where your tests battle against bugs, regressions, and edge cases.

#### The Combat Metaphor

Running tests is visualized as combat:
- Each test "attacks" the code under test
- Passing = the test successfully validated the code
- Failing = the test found a bug (this is good! the test did its job)
- Flaky = unreliable combatant, needs attention

#### Real Metrics Only

Units display actual test metrics - no fake RPG stats:

- **Execution time**: How long the test takes to run
- **Pass rate**: Historical success percentage
- **Coverage**: What code paths this test exercises
- **Flakiness score**: Variance in results over time
- **Last run**: When it was last executed
- **Failure history**: Recent failures and their causes

#### Test Relationships

- Tests that cover the same code are "in the same squad"
- Integration tests that span multiple modules show connections
- Test dependencies are visualized (setup/teardown chains)


### Advisors: Real Subagents

Advisors are **real subagents that you write and configure** - not pre-defined characters that unlock. They appear in a council panel, Civ-style.

#### What Advisors Are

- Subagents defined in your coding harness configuration
- Each has a specific domain focus and tool access
- They execute real analysis tasks when consulted
- The panel shows what subagents you've actually configured

#### Advisor Panel

Displays your configured subagents:
- Icon and name (user-defined)
- Domain/expertise area
- Status (idle, thinking, has insights)
- Click to consult

#### Advisor Insights

Advisors can proactively surface insights based on:
- Code changes they observe
- Build/test results
- Patterns they detect in their domain

#### Consultation

Select an advisor and ask questions in their domain. The advisor executes as a real subagent with access to relevant tools.

No fake "relationship system" - advisors are tools, not characters to befriend.

### Belts: Dependency & Data Flow (Factorio-style)

Visualize relationships between files as animated flowing belts, inspired by Factorio's transport belt system.

#### Dependency Visualization

- **Belt color** = Type of relationship (import, inheritance, composition)
- **Belt width** = Coupling strength (number of symbols used)
- **Bottlenecks** = Many belts converging (high fan-in)
- **Loops** = Circular dependencies (warning!)
- **Animation speed** = How often the connection is exercised

#### Visual Debugging Mode

Belts also afford **visual debugging** - seeing data flow through functions:

- **Inputs**: What values flow into a function
- **Operations**: Step-by-step transformation
- **Outputs**: What values come out

This turns the belt metaphor into a debugging tool: watch data flow through your code like items on a Factorio belt.

#### Target Languages

Initial support for languages with good introspection:
- **Python**: Via debugger/trace hooks
- **TypeScript**: Via source maps and debugger
- **Go**: Via delve integration

#### Problem Detection

Belts highlight architectural issues:
- Circular dependencies (pulsing red)
- High coupling (thick belts)
- Orphaned modules (no connections)
- Hot paths (frequently traversed)

---

## Resource System

Resources appear in a top bar, RTS-style. The system is **extensible** - applications define their own resource types.

### Resource Bar Visualization


### Resource Data Model


### Resource Economy

Track how resources flow:


### Resource History and Trends


### Built-in Resources


### Custom Resource Examples


### Resource Actions

Resources can have associated actions:


### Resource Notifications


---

## Deployment & Usage Visualization

### The World Map: Production Environment

Beyond your codebase, visualize deployed services as a living world.

#### Production Realm Visualization


#### City (Service) Data Model


#### Trade Routes (Service Communication)


#### Threats (Barbarians/Attacks)


#### Weather System (Global Metrics)


#### Population (Users/Traffic)


#### Happiness (SLO/Performance)


#### Real-time Updates


- **Cities** = Deployed services
- **Population** = Active users/requests
- **City Happiness** = Latency, error rates
- **Trade Routes** = Service-to-service communication
- **Barbarians** = Errors, attacks, anomalies

### Metrics as Weather/Seasons

- **Clear skies** = All systems nominal
- **Storm clouds** = Elevated error rates
- **Drought** = Low traffic
- **Floods** = Traffic spikes

---

## Capability Inventory (Tech Tree View)

The tech tree is a **descriptive visualization** of what tools, MCPs, and commands you have - not a progression system to unlock.

### What It Shows

A visual clustering of your actual capabilities:
- **Tools**: Claude Code built-in tools you have access to
- **MCPs**: MCP servers you've configured
- **Commands**: Slash commands and skills available
- **Integrations**: External services connected

### Descriptive, Not Prescriptive

- No "research points" to spend
- No artificial prerequisites
- No locked capabilities to unlock
- Simply shows what you've configured in your harness

### Organization

Capabilities are clustered by domain:
- **Core**: File operations, search, edit
- **Build**: Build system integrations
- **Version Control**: Git, GitHub, etc.
- **Deployment**: CI/CD, cloud services
- **Analysis**: Linting, testing, coverage

### Discovery

When you add a new MCP server or configure a new integration, it appears in the tree. The visualization grows organically with your project's tooling.

---

## Missions System

Suggested objectives drive engagement and teach capabilities.

### Mission Data Model


### Mission Types

#### Campaign Missions (Story-driven)


#### Side Quests (Opportunistic)


#### Endless/Sandbox Missions


### Mission UI


---

## Extensibility Architecture

### Plugin System

The game interface is extensible at multiple levels through a comprehensive plugin API.

### Plugin Data Model


### Plugin Registry


### Example Plugins


### Theme System


---

## UI Layout & Interaction

### Keyboard-First Design

**No mouse required.** Everything is accessible via keyboard.

#### Input Modes (Vim-style)

- **Normal mode**: Navigate the interface, select elements, trigger actions
- **Insert mode**: Type in the prompt window
- **Visual mode**: Select multiple elements on the map

#### Key Bindings

Configurable as vim-style or emacs-style:

**Navigation:**
- `h/j/k/l` or arrows: Move focus
- `1-5`: Switch views (Map, Buildings, Units, Tech, Missions)
- `Tab`: Cycle focus areas
- `g`: Go to file (fuzzy finder)
- `/`: Search

**Actions:**
- `Enter`: Focus prompt window / submit prompt
- `Space`: Run selected build/test
- `i`: Inspect selected element
- `a`: Consult advisor panel

**Prompt Interaction:**
- Focus prompt window
- Type your request
- Press `Enter` to "end turn" and submit
- Watch visualization of the coding process

### The Turn Metaphor

Submitting a prompt is like ending your turn:
1. You give instructions
2. Claude executes (animated visualization)
3. Results appear on the map
4. Your turn again

### Main Interface Layout

- **Top**: Resource bar (real metrics: context tokens, API cost, build status)
- **Left**: Advisor panel (your configured subagents)
- **Center**: Main map view (codebase visualization)
- **Right**: Mission/objective panel
- **Bottom**: Prompt window (central interaction point)

### Responsive Layouts

Adapts to screen size while maintaining keyboard accessibility


---

## Data Model & Persistence

### Game State


### Persistence Layer


---

## Claude Code Integration

### Integration Architecture


### Tool-to-Game Mapping


### Advisor Integration with Claude


---

## Game Start Screen

When launching CodingGame, present a start screen for configuration.

### Harness Selection

For now, targets **Claude Code** harness only. Future: support other harnesses.

### Model Selection

Choose your model:
- **Claude 4.5 Opus**: More capable, higher cost
- **Claude 4.5 Sonnet**: Faster, lower cost

### Project Selection

- Open existing project
- Recent projects list
- Project-specific settings persist

---

## Multi-Agent Future

Future vision: orchestrate multiple agents working in parallel.

### The Concept

- Multiple agents visible on the map simultaneously
- Each agent has its own "fog of war" (context boundary)
- Player orchestrates which agent works on what
- Agents can hand off work to each other

### Fog of War Per Agent

Each agent only "sees" what's in its context:
- Files it has read are revealed
- Files outside context are fogged
- Different agents have different views
- Overlapping views show shared knowledge

### Orchestration UI

- See all active agents
- Assign tasks to specific agents
- Monitor agent progress
- Manage agent context windows

This extends the strategy game metaphor: you're commanding multiple units (agents) across the codebase battlefield.

---

## Open Questions

1. **Build System Agnostic?** - Started with Bazel, but should support npm, cargo, etc. *(Addressed: Multi-build-system support via adapters)*
2. **Real-time vs Turn-based?** - Continuous updates with optional "turn-end" checkpoints
3. **Multiplayer?** - Future: Team view where multiple developers are visible
4. **Save/Load?** - Implemented via persistence layer with auto-save
5. **Achievements?** - Yes, integrated with mission rewards
6. **Sound Design?** - Optional via theme system, audio feedback for key events

---

## Implementation Roadmap

### Phase 0: Foundation
- [ ] Game start screen (harness/model selection)
- [ ] Keyboard navigation framework
- [ ] Basic prompt interaction loop

### Phase 1: Core Framework
- [ ] Map visualization engine
- [ ] Tile and fog of war system
- [ ] Basic resource tracking (real metrics)
- [ ] Claude tool interception

### Phase 2: Buildings & Units
- [ ] Build system adapters (npm, Bazel, cargo)
- [ ] Real metrics display for builds
- [ ] Test visualization with actual results

### Phase 3: Advisors
- [ ] Subagent configuration loading
- [ ] Insight generation from real analysis
- [ ] Advisor consultation UI

### Phase 4: Belts & Debugging
- [ ] Dependency flow visualization
- [ ] Visual debugging mode
- [ ] Python/TypeScript/Go support

### Phase 5: Capability Inventory
- [ ] Tool/MCP discovery
- [ ] Capability clustering visualization
- [ ] Dynamic updates as config changes

### Phase 6: Production Realm
- [ ] Deployment visualization
- [ ] Service health monitoring
- [ ] Weather/metrics system

### Phase 7: Multi-Agent Future
- [ ] Multiple concurrent agents
- [ ] Per-agent fog of war (context boundaries)
- [ ] Agent orchestration UI

### Phase 8: Polish
- [ ] Plugin system
- [ ] Theme support
- [ ] Sound design (optional)

---

*"The code is dark and full of terrors... but we shall bring the light of understanding."*
— The Chronicler
