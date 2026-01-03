# CodingGame: Strategy Game Interface for Claude Code

A Civilization/Call to Power II/Factorio-inspired UI wrapper for Claude Code, transforming software development into an intuitive strategy game experience.

## Core Philosophy

Software development *is* strategy. You manage resources, build infrastructure, deploy units, research new capabilities, and pursue objectives. This interface makes those metaphors explicit and interactive.

---

## Visual Metaphors & Mapping

### The Map: Codebase as Territory

The main view is a strategic map of your codebase:

- **Tiles/Regions** = Directories and file clusters
- **Fog of War** = Unread/unexplored code
- **Revealed Territory** = Code Claude has analyzed
- **Terrain Types** = File types (source, test, config, docs)
- **Trade Routes** = Import/export relationships between modules

Double-clicking any file tile opens it in an integrated editor.

### Buildings: Bazel Packages

Packages are your production infrastructure:

```
┌─────────────────────┐
│  //src/auth:lib     │
│  ━━━━━━━━━━━━━━━━   │
│  Level: 3           │
│  Health: 87%        │
│  Producing: ⚔️ x2   │
│  Dependencies: 5    │
└─────────────────────┘
```

- **Building Level** = Package maturity/complexity
- **Health** = Build reliability, test pass rate
- **Production Queue** = Targets being built
- **Upgrade Path** = Refactoring opportunities

### Units: Build & Test Targets

Each build/test target is a unit with RPG-style stats:

```
┌─────────────────────────────┐
│ ⚔️ auth_service_test        │
├─────────────────────────────┤
│ HP:  ████████░░  80/100     │  ← Reliability (flaky = low HP)
│ ATK: ██████████  47         │  ← Coverage / Bug detection
│ DEF: ███████░░░  34         │  ← Resilience to changes
│ SPD: ████░░░░░░  1.2s       │  ← Execution time
├─────────────────────────────┤
│ Status: ✓ Passing           │
│ Last Run: 2m ago            │
│ XP: 1,247 (Level 8)         │  ← Time since last failure
└─────────────────────────────┘
```

Units are "produced" by buildings when builds/tests run. Stats update with results.

### Advisors: Subagents

Specialized AI advisors appear in a council panel, Civ-style:

| Advisor | Domain | Personality |
|---------|--------|-------------|
| **Architect** | Codebase structure, dependencies | "Our eastern modules grow tangled, sire" |
| **Sentinel** | Security, vulnerabilities | "I've detected weaknesses in our defenses" |
| **Chronicler** | Documentation, code clarity | "The people cannot understand the auth scrolls" |
| **Quartermaster** | Performance, optimization | "Build times drain our resources" |
| **Inquisitor** | Testing, coverage, quality | "42% of our realm remains unverified" |
| **Diplomat** | Dependencies, external APIs | "The npm kingdoms demand updates" |

Advisors proactively surface insights and can be consulted for their domain.

### Belts: Dependency & Data Flow (Factorio-style)

Visualize relationships between files as flowing belts:

```
┌──────────┐    ┌──────────┐    ┌──────────┐
│ types.ts │═══▶│ auth.ts  │═══▶│ api.ts   │
└──────────┘    └──────────┘    └──────────┘
                    ║
                    ▼
               ┌──────────┐
               │ utils.ts │
               └──────────┘
```

- **Belt color** = Type of relationship (import, inheritance, composition)
- **Belt width** = Coupling strength
- **Bottlenecks** = Many belts converging (high fan-in)
- **Loops** = Circular dependencies (warning!)
- **Throughput** = How often the connection is exercised

---

## Resource System

Resources appear in a top bar, RTS-style. The system is **extensible** - applications define their own resource types.

### Core Resource Types

```
┌────────────────────────────────────────────────────────────┐
│ 🪙 Context: 42,000/100,000  │ 😊 Coverage: 73%  │ ⚡ CI: 3  │
└────────────────────────────────────────────────────────────┘
```

#### Built-in Resources

| Resource | Metaphor | Meaning |
|----------|----------|---------|
| **Context** | Gold | Token budget - scarce, spend wisely |
| **Coverage** | Happiness | Test coverage - affects stability |
| **CI Capacity** | Production | Parallel build/test slots |

#### Extensible Resource Framework

Resources are defined declaratively:

```typescript
interface Resource {
  id: string;
  name: string;
  icon: string;
  metaphor: "currency" | "satisfaction" | "capacity" | "mana";

  // How to compute current value
  compute: () => number | Promise<number>;

  // Optional: max value (for bar display)
  max?: number | (() => number);

  // Optional: thresholds for warnings
  thresholds?: {
    critical?: number;
    warning?: number;
    good?: number;
  };

  // How it's consumed/produced
  economy?: {
    producers?: string[];  // What generates this
    consumers?: string[];  // What spends this
  };
}
```

#### Example Custom Resources

```typescript
// API rate limits as mana
const apiMana: Resource = {
  id: "github_api",
  name: "GitHub API",
  icon: "🐙",
  metaphor: "mana",
  compute: () => getRateLimitRemaining(),
  max: 5000,
  thresholds: { critical: 100, warning: 500 }
};

// Deploy slots as capacity
const deploySlots: Resource = {
  id: "deploy_capacity",
  name: "Deploy Slots",
  icon: "🚀",
  metaphor: "capacity",
  compute: () => getAvailableDeploySlots(),
  max: 3
};

// Error budget as currency
const errorBudget: Resource = {
  id: "error_budget",
  name: "Error Budget",
  icon: "💔",
  metaphor: "currency",
  compute: () => getSLOErrorBudgetRemaining(),
  max: 100,
  thresholds: { critical: 10, warning: 30 }
};
```

---

## Deployment & Usage Visualization

### The World Map: Production Environment

Beyond your codebase, visualize deployed services:

```
┌─────────────────────────────────────────────────────────┐
│                    PRODUCTION REALM                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   ┌─────────┐         ┌─────────┐         ┌─────────┐   │
│   │ Web App │────────▶│   API   │────────▶│   DB    │   │
│   │ ██████░ │         │ ████████│         │ ███████ │   │
│   │ 2.1k rps│         │ 847 rps │         │ 99.2%   │   │
│   └─────────┘         └─────────┘         └─────────┘   │
│        │                   │                             │
│        │              ┌────┴────┐                        │
│        └─────────────▶│  Cache  │                        │
│                       │ HIT:94% │                        │
│                       └─────────┘                        │
│                                                          │
│   Population: 12,847 active users                        │
│   Happiness: 98.2% (P99: 142ms)                         │
└─────────────────────────────────────────────────────────┘
```

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

## Tech Tree: Tools, MCP, Commands

Research and unlock capabilities:

```
                         ┌─────────────┐
                         │   Claude    │
                         │    Code     │
                         └──────┬──────┘
                                │
          ┌─────────────────────┼─────────────────────┐
          │                     │                     │
    ┌─────┴─────┐        ┌─────┴─────┐        ┌─────┴─────┐
    │   Tools   │        │    MCP    │        │ Commands  │
    └─────┬─────┘        └─────┬─────┘        └─────┬─────┘
          │                    │                    │
    ┌─────┼─────┐        ┌─────┼─────┐        ┌─────┼─────┐
    │     │     │        │     │     │        │     │     │
   Edit  Bash  Web    GitHub  DB   Custom   /review /fix  ...
          │              │
        ┌─┴─┐          ┌─┴─┐
       npm  git      Issues PRs
```

- **Researched** = Configured and available
- **Locked** = Available but not set up
- **Undiscovered** = User hasn't encountered yet
- **Prerequisites** = Some tools require others first

---

## Missions System

Suggested objectives drive engagement and teach capabilities:

### Mission Types

#### Campaign Missions (Story-driven)
Sequential missions that teach the system:
1. "Explore the Realm" - Read and understand the codebase
2. "Fortify the Walls" - Improve test coverage
3. "Optimize the Roads" - Speed up build times

#### Side Quests (Opportunistic)
Triggered by detected opportunities:
- "The Flaky Fortress" - Fix that test that fails 10% of the time
- "Technical Debt Collection" - Address that TODO from 2019
- "The Deprecated Dependency" - Upgrade that old package

#### Endless/Sandbox
Ongoing objectives with leaderboards:
- Build time speedrun
- Coverage maximization
- Zero-bug streak

### Mission UI

```
┌─────────────────────────────────────────────────────────┐
│ 📜 ACTIVE MISSION: Ship the Authentication Feature      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│ The kingdom requires secure gates. Implement OAuth      │
│ integration and prove its strength with tests.          │
│                                                          │
│ Objectives:                                              │
│ ✓ Create auth module structure                          │
│ ✓ Implement OAuth flow                                  │
│ ○ Add session management                                │
│ ○ Write integration tests (0/5)                         │
│ ○ Pass security review                                  │
│                                                          │
│ Rewards: +500 XP, Unlock "Security Advisor"             │
│                                                          │
│ [Abandon] [Get Hint] [Mark Complete]                    │
└─────────────────────────────────────────────────────────┘
```

---

## Extensibility Architecture

### Plugin System

The game interface is extensible at multiple levels:

```typescript
interface CodingGamePlugin {
  id: string;
  name: string;

  // Add new resource types
  resources?: Resource[];

  // Add new advisor personalities
  advisors?: Advisor[];

  // Add mission templates
  missions?: MissionTemplate[];

  // Add visualization layers
  visualizations?: Visualization[];

  // Add building types (beyond Bazel)
  buildingSystems?: BuildingSystem[];

  // Add unit types (beyond build targets)
  unitTypes?: UnitType[];
}
```

### Example: Kubernetes Plugin

```typescript
const k8sPlugin: CodingGamePlugin = {
  id: "kubernetes",
  name: "Kubernetes Realm",

  resources: [
    { id: "pods", name: "Pod Capacity", icon: "🫛", ... },
    { id: "cpu", name: "CPU Quota", icon: "🔥", ... },
  ],

  advisors: [
    { id: "helmsman", name: "The Helmsman", domain: "k8s" }
  ],

  visualizations: [
    { id: "cluster_map", name: "Cluster Map", ... }
  ],

  buildingSystems: [
    { id: "deployment", name: "Deployments", ... },
    { id: "service", name: "Services", ... }
  ]
};
```

---

## UI Layout Concept

```
┌──────────────────────────────────────────────────────────────────┐
│ 🪙 Context: 42k │ 😊 Coverage: 73% │ ⚡ CI: 3 │ 🐙 API: 4.2k     │
├──────────────────────────────────────────────────────────────────┤
│                    │                                    │        │
│   ┌─────────────┐  │         [MAIN MAP VIEW]           │ 📜     │
│   │  Advisors   │  │                                    │        │
│   │             │  │   Codebase / Belts / Deployment   │ Active │
│   │ 🏛️ Architect│  │                                    │ Mission│
│   │ 🛡️ Sentinel │  │                                    │        │
│   │ 📚 Chronicler│ │                                    │        │
│   │ ⚖️ Quarter. │  │                                    │        │
│   │             │  │                                    │        │
│   └─────────────┘  │                                    │        │
│                    │                                    │        │
├────────────────────┴────────────────────────────────────┴────────┤
│ [Tech Tree] [Buildings] [Units] [Missions]    🔍 Search  ⚙️ Set. │
├──────────────────────────────────────────────────────────────────┤
│ > Claude: Analyzing the auth module...                           │
│ > Build complete: //src/auth:lib ✓                              │
│                                                                  │
│ ▌                                                                │
└──────────────────────────────────────────────────────────────────┘
```

---

## Open Questions

1. **Build System Agnostic?** - Started with Bazel, but should support npm, cargo, etc.
2. **Real-time vs Turn-based?** - Continuous updates or discrete "turns"?
3. **Multiplayer?** - Team view where multiple developers are visible?
4. **Save/Load?** - Persist game state across sessions?
5. **Achievements?** - Long-term progression system?
6. **Sound Design?** - Audio feedback for events?

---

## Next Steps

- [ ] Prototype the map visualization
- [ ] Define the core resource API
- [ ] Implement first advisor (Architect)
- [ ] Create belt visualization for a sample project
- [ ] Design the mission system schema
- [ ] Build plugin architecture

---

*"The code is dark and full of terrors... but we shall bring the light of understanding."*
— The Chronicler
