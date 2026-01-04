# CodingGame: Strategy Game Interface for Claude Code

A Civilization/Call to Power II/Factorio-inspired UI wrapper for Claude Code, transforming software development into an intuitive strategy game experience.

## Core Philosophy

Software development *is* strategy. You manage resources, build infrastructure, deploy units, research new capabilities, and pursue objectives. This interface makes those metaphors explicit and interactive.

---

## Visual Metaphors & Mapping

### The Map: Codebase as Territory

The main view is a strategic map of your codebase.

#### Tile System

Each tile represents a file or directory. Tiles are arranged in a zoomable, pannable 2D space.

```
┌─────────────────────────────────────────────────────────────────┐
│                         src/                                     │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐    │
│  │   auth/   │  │   api/    │  │  models/  │  │  utils/   │    │
│  │  ░░░░░░░  │  │  ████████ │  │  ██████░░ │  │  ████████ │    │
│  │  Level 2  │  │  Level 4  │  │  Level 3  │  │  Level 5  │    │
│  └───────────┘  └───────────┘  └───────────┘  └───────────┘    │
│        │              │              │              │           │
│        └──────────────┼──────────────┘              │           │
│                       │                              │           │
│                       ▼                              │           │
│                 ┌───────────┐                        │           │
│                 │  index.ts │◄───────────────────────┘           │
│                 │  ████████ │                                    │
│                 └───────────┘                                    │
└─────────────────────────────────────────────────────────────────┘
```

**Tile Properties:**

```typescript
interface MapTile {
  id: string;
  path: string;                    // File or directory path
  type: TileType;
  position: { x: number; y: number };
  size: { width: number; height: number };

  // Visibility state
  visibility: "fog" | "revealed" | "explored";
  lastExplored?: Date;
  exploredBy?: string;             // Which agent/action revealed it

  // Visual properties
  terrain: TerrainType;
  elevation: number;               // Based on directory depth
  fertility: number;               // Code quality score

  // Interaction
  selected: boolean;
  highlighted: boolean;
  clickAction: "open" | "expand" | "select";
}

type TileType = "file" | "directory" | "package" | "module";

type TerrainType =
  | "source"      // .ts, .js, .py, etc. - grassland
  | "test"        // *.test.*, *.spec.* - fortified
  | "config"      // .json, .yaml, .toml - mountain
  | "docs"        // .md, .txt - plains
  | "asset"       // images, fonts - forest
  | "generated"   // build output - wasteland
  | "vendored";   // node_modules, vendor - foreign territory
```

#### Zoom Levels

| Level | Name | Shows | Interaction |
|-------|------|-------|-------------|
| 1 | World | Repository root, top-level dirs | Click to zoom |
| 2 | Region | Package/module clusters | See building outlines |
| 3 | City | Individual packages with buildings | See unit production |
| 4 | Street | Files within a package | Double-click opens editor |
| 5 | Interior | Single file contents | Syntax-highlighted code view |

```typescript
interface ZoomLevel {
  level: number;
  name: string;
  minScale: number;
  maxScale: number;

  // What's visible at this level
  showFiles: boolean;
  showDirectories: boolean;
  showBuildings: boolean;
  showUnits: boolean;
  showBelts: boolean;
  showLabels: "none" | "major" | "all";

  // Clustering behavior
  clusterThreshold: number;        // Files per cluster before collapsing
  clusterStrategy: "grid" | "treemap" | "force";
}
```

#### Fog of War Mechanics

Files start in fog until Claude analyzes them:

```typescript
interface FogOfWar {
  // Visibility states
  states: {
    fog: {
      opacity: 0.8;
      color: "#1a1a2e";
      interaction: "click to explore";
    };
    revealed: {
      opacity: 0.2;
      color: "#16213e";
      interaction: "full access";
      decayTime?: number;          // Optional: fog returns after N hours
    };
    explored: {
      opacity: 0;
      interaction: "full access";
    };
  };

  // What reveals fog
  revealTriggers: [
    "file_read",                   // Claude reads the file
    "file_edit",                   // Claude edits the file
    "grep_match",                  // File appears in search results
    "import_traced",               // Discovered via dependency
    "user_click",                  // User manually explores
  ];

  // Reveal radius - exploring one file reveals neighbors
  revealRadius: number;            // Tiles around the explored file
  revealDepth: number;             // Directory levels to reveal
}
```

#### Navigation & Interaction

```typescript
interface MapNavigation {
  // Pan controls
  pan: {
    mouse: "drag" | "edge-scroll";
    keyboard: ["arrow keys", "WASD"];
    touch: "two-finger-drag";
  };

  // Zoom controls
  zoom: {
    mouse: "scroll-wheel";
    keyboard: ["+/-", "ctrl+scroll"];
    touch: "pinch";
    doubleClick: "zoom-to-fit-selection";
  };

  // Selection
  select: {
    single: "click";
    multi: "ctrl+click" | "shift+click";
    area: "drag-rectangle";
    all: "ctrl+a";
  };

  // Quick navigation
  shortcuts: {
    "g": "goto-file-dialog";
    "h": "go-home";                // Jump to root
    "b": "go-back";                // Previous location
    "/": "search";
    "f": "find-in-map";
    "1-5": "zoom-to-level";
  };

  // File interaction
  fileActions: {
    "click": "select";
    "double-click": "open-in-editor";
    "right-click": "context-menu";
    "hover": "show-tooltip";
    "drag": "n/a";                 // Files don't move
  };
}
```

#### Layout Algorithms

The map can use different layout strategies:

```typescript
type LayoutAlgorithm =
  | "treemap"      // Space-filling, size = lines of code
  | "force"        // Force-directed based on dependencies
  | "grid"         // Simple grid layout
  | "hierarchical" // Tree structure, directories as containers
  | "radial";      // Root in center, depth = radius

interface LayoutConfig {
  algorithm: LayoutAlgorithm;

  // Treemap options
  treemap?: {
    sizeMetric: "lines" | "bytes" | "complexity" | "changes";
    aspectRatio: number;
    padding: number;
  };

  // Force-directed options
  force?: {
    linkStrength: number;          // Pull between connected files
    repulsion: number;             // Push between all files
    gravity: number;               // Pull toward center
    iterations: number;
  };

  // Common options
  sortBy: "name" | "type" | "size" | "modified" | "importance";
  groupBy: "directory" | "type" | "module" | "none";
}
```

#### Minimap

A small overview map in the corner:

```
┌──────────────┐
│ ▪▪▪  ▫▫▫ ▪▪ │
│ ▪▪▪▪ ▫▫  ▪▪ │
│   ┌────┐     │
│   │view│ ▪▪  │
│   └────┘     │
│ ▪▪  ▫▫▫  ▪▪ │
└──────────────┘
```

```typescript
interface Minimap {
  position: "top-right" | "bottom-right" | "bottom-left";
  size: { width: number; height: number };

  // What to show
  showFog: boolean;
  showBuildings: boolean;
  showViewport: boolean;
  showHotspots: boolean;           // Recently changed areas

  // Interaction
  clickToNavigate: boolean;
  dragViewport: boolean;
}
```

Double-clicking any file tile opens it in an integrated editor.

### Buildings: Bazel Packages

Packages are your production infrastructure. Each BUILD file defines a "building" that produces units (targets).

#### Building Visualization

```
┌─────────────────────────────────────────┐
│  🏭 //src/auth:lib                      │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
├─────────────────────────────────────────┤
│  Level: 3 ★★★☆☆     │  Health: 87%     │
│  ──────────────────────────────────────│
│  📦 Targets: 4       │  ⚡ Build: 2.3s  │
│  🔗 Deps: 5 in / 12 out                 │
├─────────────────────────────────────────┤
│  PRODUCTION QUEUE                       │
│  ┌─────┐ ┌─────┐ ┌─────┐               │
│  │⚔️ 1 │→│⚔️ 2 │→│🛡️ 1 │→ [output]    │
│  └─────┘ └─────┘ └─────┘               │
├─────────────────────────────────────────┤
│  [Build] [Test] [Inspect] [Upgrade]    │
└─────────────────────────────────────────┘
```

#### Building Data Model

```typescript
interface Building {
  id: string;
  label: string;                   // Bazel label: //path/to:target
  path: string;                    // Filesystem path to BUILD file
  buildSystem: BuildSystem;

  // Building stats
  level: number;                   // 1-5, computed from metrics
  health: number;                  // 0-100%, based on build success rate
  experience: number;              // Accumulated from successful builds

  // Targets this building can produce
  targets: Target[];
  defaultTarget?: string;

  // Dependencies
  dependencies: {
    incoming: Dependency[];        // What this building needs
    outgoing: Dependency[];        // What depends on this building
  };

  // Production metrics
  production: {
    lastBuildTime: number;         // ms
    averageBuildTime: number;
    successRate: number;           // Last N builds
    totalBuilds: number;
    lastBuilt?: Date;
  };

  // Visual state
  position: { x: number; y: number };
  size: BuildingSize;
  style: BuildingStyle;
  animations: BuildingAnimation[];
}

type BuildSystem =
  | { type: "bazel"; workspace: string }
  | { type: "npm"; packageJson: string }
  | { type: "cargo"; cargoToml: string }
  | { type: "gradle"; buildGradle: string }
  | { type: "make"; makefile: string }
  | { type: "cmake"; cmakeLists: string };

type BuildingSize = "small" | "medium" | "large" | "mega";

interface BuildingStyle {
  sprite: string;                  // Building graphic
  color: string;                   // Tint based on health/status
  effects: VisualEffect[];         // Smoke, sparks, glow
  damaged: boolean;                // Show cracks if health < 50%
}
```

#### Building Levels

Buildings level up based on accumulated metrics:

| Level | Name | Requirements | Bonuses |
|-------|------|--------------|---------|
| 1 | Hut | New package | Basic production |
| 2 | Workshop | 10 successful builds | +10% build speed |
| 3 | Factory | 50 builds, 80% success | Parallel production |
| 4 | Foundry | 200 builds, 90% success | +25% speed, priority queue |
| 5 | Citadel | 500 builds, 95% success | Instant cache hits |

```typescript
interface BuildingLevel {
  level: number;
  name: string;
  sprite: string;

  // Requirements to reach this level
  requirements: {
    totalBuilds: number;
    successRate: number;
    minTargets?: number;
    minCoverage?: number;
  };

  // Bonuses at this level
  bonuses: {
    buildSpeedMultiplier: number;
    parallelSlots: number;
    cacheEfficiency: number;
    priorityBoost: number;
  };

  // Visual upgrades
  visualUpgrades: string[];        // Additional sprites/effects
}
```

#### Building Health

Health reflects reliability and is affected by:

```typescript
interface BuildingHealth {
  current: number;                 // 0-100

  // Factors that affect health
  factors: {
    buildSuccessRate: number;      // Weight: 40%
    testPassRate: number;          // Weight: 30%
    lastBuildAge: number;          // Weight: 10% (stale = unhealthy)
    dependencyHealth: number;      // Weight: 10% (avg of deps)
    codeChurn: number;             // Weight: 10% (high churn = lower)
  };

  // Health states
  state:
    | "thriving"    // 90-100%
    | "healthy"     // 70-89%
    | "struggling"  // 50-69%
    | "critical"    // 25-49%
    | "ruined";     // 0-24%

  // Visual indicators
  effects: {
    thriving: ["golden glow", "birds"];
    healthy: ["normal"];
    struggling: ["smoke", "cracks"];
    critical: ["fire", "heavy smoke"];
    ruined: ["rubble", "abandoned"];
  };
}
```

#### Production Queue

Buildings produce units when builds/tests run:

```typescript
interface ProductionQueue {
  building: string;                // Building ID
  queue: ProductionJob[];
  maxParallel: number;             // Based on building level
  currentlyProducing: ProductionJob[];

  // Queue management
  add(target: Target, priority?: number): void;
  cancel(jobId: string): void;
  prioritize(jobId: string): void;
  pause(): void;
  resume(): void;
}

interface ProductionJob {
  id: string;
  target: Target;
  priority: number;
  status: "queued" | "producing" | "complete" | "failed";

  // Timing
  queuedAt: Date;
  startedAt?: Date;
  completedAt?: Date;
  estimatedTime: number;           // Based on historical data

  // Progress
  progress: number;                // 0-100%
  currentPhase: string;            // "compiling", "linking", "testing"

  // Result
  result?: {
    success: boolean;
    unit?: Unit;                   // The produced unit
    errors?: BuildError[];
    warnings?: BuildWarning[];
  };
}
```

#### Building Upgrades

Players can invest in upgrading buildings:

```typescript
interface BuildingUpgrade {
  id: string;
  name: string;
  description: string;
  icon: string;

  // Cost to upgrade
  cost: {
    context?: number;              // Token cost
    time?: number;                 // Time investment
    prerequisites?: string[];      // Other upgrades needed
  };

  // Effects
  effects: {
    healthBonus?: number;
    speedBonus?: number;
    parallelSlots?: number;
    unlockTargets?: string[];
    special?: string;              // Special ability
  };
}

// Example upgrades
const buildingUpgrades: BuildingUpgrade[] = [
  {
    id: "cache_optimization",
    name: "Cache Optimization",
    description: "Improve build caching for faster rebuilds",
    icon: "💾",
    cost: { context: 1000 },
    effects: { speedBonus: 0.3 }
  },
  {
    id: "parallel_testing",
    name: "Parallel Testing",
    description: "Run tests in parallel",
    icon: "⚡",
    cost: { context: 2000, prerequisites: ["cache_optimization"] },
    effects: { parallelSlots: 4 }
  },
  {
    id: "auto_repair",
    name: "Auto-Repair",
    description: "Automatically fix common build issues",
    icon: "🔧",
    cost: { context: 5000 },
    effects: { special: "auto_fix_imports" }
  }
];
```

#### Multi-Build-System Support

Buildings adapt to different build systems:

```typescript
interface BuildSystemAdapter {
  type: string;
  name: string;
  icon: string;

  // Detection
  detectFiles: string[];           // Files that indicate this system
  detectPatterns: RegExp[];

  // Building operations
  operations: {
    list(): Promise<Target[]>;
    build(target: string): Promise<BuildResult>;
    test(target: string): Promise<TestResult>;
    clean(): Promise<void>;
    dependencies(target: string): Promise<Dependency[]>;
  };

  // Mapping to game concepts
  mapping: {
    packageToBuilding: (pkg: any) => Building;
    targetToUnit: (target: any) => Unit;
    dependencyToBelt: (dep: any) => Belt;
  };
}

// Registered adapters
const buildSystems: BuildSystemAdapter[] = [
  bazelAdapter,
  npmAdapter,
  cargoAdapter,
  gradleAdapter,
  makeAdapter,
  cmakeAdapter,
  poetryAdapter,
  goModAdapter,
];
```

### Units: Build & Test Targets

Each build/test target is a unit with RPG-style stats. Units are "produced" by buildings when builds/tests run.

#### Unit Visualization

```
┌─────────────────────────────────────────┐
│ ⚔️ auth_service_test          Level 8   │
├─────────────────────────────────────────┤
│                                          │
│  ┌──────┐  HP:  ████████░░  80/100      │
│  │      │  ATK: ██████████  47          │
│  │ 🗡️   │  DEF: ███████░░░  34          │
│  │      │  SPD: ████░░░░░░  1.2s        │
│  └──────┘  LCK: █████░░░░░  23          │
│                                          │
├─────────────────────────────────────────┤
│ Class: Test Knight    │ Rank: Veteran   │
├─────────────────────────────────────────┤
│ Status: ✓ Passing                       │
│ Last Run: 2m ago                        │
│ XP: 1,247 / 2,000     ████████░░        │
├─────────────────────────────────────────┤
│ Abilities:                              │
│  • Integration Slash (tests 3 services) │
│  • Mock Shield (isolates dependencies)  │
├─────────────────────────────────────────┤
│ [Run] [Debug] [Inspect] [Retire]        │
└─────────────────────────────────────────┘
```

#### Unit Data Model

```typescript
interface Unit {
  id: string;
  name: string;                    // Target name
  label: string;                   // Full build label
  building: string;                // Parent building ID

  // Classification
  class: UnitClass;
  rank: UnitRank;
  type: UnitType;

  // Core stats
  stats: {
    hp: Stat;                      // Reliability
    atk: Stat;                     // Bug detection power
    def: Stat;                     // Resilience to changes
    spd: Stat;                     // Execution time
    lck: Stat;                     // Flakiness (inverse)
  };

  // Progression
  level: number;
  experience: number;
  experienceToNext: number;

  // State
  status: UnitStatus;
  lastRun?: Date;
  lastResult?: RunResult;
  history: RunResult[];

  // Abilities
  abilities: Ability[];
  passives: Passive[];

  // Equipment (configuration)
  equipment: Equipment[];

  // Visual
  sprite: string;
  animations: Animation[];
  effects: VisualEffect[];
}

interface Stat {
  base: number;                    // Computed from metrics
  current: number;                 // After modifiers
  max: number;
  modifiers: Modifier[];
}

type UnitClass =
  | "warrior"      // Binary/executable targets
  | "knight"       // Test targets
  | "mage"         // Library targets
  | "archer"       // Benchmark targets
  | "healer"       // Lint/format targets
  | "scout";       // Type-check targets

type UnitRank =
  | "recruit"      // Level 1-2
  | "soldier"      // Level 3-4
  | "veteran"      // Level 5-7
  | "elite"        // Level 8-9
  | "champion";    // Level 10+

type UnitType = "build" | "test" | "lint" | "format" | "typecheck" | "benchmark";

type UnitStatus =
  | "idle"
  | "queued"
  | "running"
  | "passing"
  | "failing"
  | "flaky"
  | "disabled"
  | "retired";
```

#### Stat Calculations

Each stat maps to real build/test metrics:

```typescript
interface StatCalculation {
  // HP: Reliability (0-100)
  hp: {
    formula: "successRate * 100";
    factors: [
      { metric: "pass_rate_last_100", weight: 0.6 },
      { metric: "consecutive_passes", weight: 0.2 },
      { metric: "time_since_last_fail_days", weight: 0.2 }
    ];
    // HP regenerates when tests pass
    regen: {
      onPass: 5;
      onFail: -20;
      passive: 1; // per hour of no failures
    };
  };

  // ATK: Bug Detection Power
  atk: {
    formula: "coverage * complexity_tested";
    factors: [
      { metric: "line_coverage", weight: 0.3 },
      { metric: "branch_coverage", weight: 0.3 },
      { metric: "mutation_score", weight: 0.2 },
      { metric: "assertion_density", weight: 0.2 }
    ];
    // Higher ATK = better at finding bugs
  };

  // DEF: Resilience to Changes
  def: {
    formula: "stability_score";
    factors: [
      { metric: "change_frequency_inverse", weight: 0.3 },
      { metric: "dependency_stability", weight: 0.3 },
      { metric: "api_surface_area_inverse", weight: 0.2 },
      { metric: "test_isolation_score", weight: 0.2 }
    ];
    // High DEF = doesn't break when other code changes
  };

  // SPD: Execution Speed
  spd: {
    formula: "time_score";
    factors: [
      { metric: "execution_time_percentile", weight: 0.5 },
      { metric: "startup_time", weight: 0.2 },
      { metric: "parallelization_factor", weight: 0.3 }
    ];
    // Displayed as actual time, internally normalized 0-100
    display: (value) => `${value}ms`;
  };

  // LCK: Anti-Flakiness
  lck: {
    formula: "consistency_score";
    factors: [
      { metric: "variance_in_results", weight: 0.4 },
      { metric: "environment_independence", weight: 0.3 },
      { metric: "determinism_score", weight: 0.3 }
    ];
    // Low LCK = flaky test, high LCK = rock solid
  };
}
```

#### Unit Leveling

Units gain XP and level up:

```typescript
interface LevelingSystem {
  // XP sources
  xpSources: {
    pass: 10;
    passAfterFail: 50;             // Bonus for fixing
    catchBug: 100;                 // Test caught a real bug
    stayGreen: 1;                  // Per hour of continuous passing
    surviveRefactor: 25;           // Passed after related code changed
  };

  // XP curve (exponential)
  xpForLevel: (level: number) => Math.floor(100 * Math.pow(1.5, level - 1));

  // Level up bonuses
  levelUpBonus: {
    statIncrease: 2;               // +2 to all stats
    abilityUnlock: [3, 5, 7, 10];  // Levels that unlock abilities
    rankUp: [3, 5, 8, 10];         // Levels that increase rank
  };

  // Max level
  maxLevel: 20;

  // Prestige system (optional)
  prestige: {
    enabled: boolean;
    requirement: { level: 20, xp: 50000 };
    bonus: "permanent +10% to all stats";
    resetLevel: true;
    keepAbilities: true;
  };
}
```

#### Abilities

Units unlock abilities as they level:

```typescript
interface Ability {
  id: string;
  name: string;
  description: string;
  icon: string;
  unlockLevel: number;

  // Ability type
  type: "active" | "passive" | "triggered";

  // For active abilities
  activation?: {
    cost?: { context?: number; time?: number };
    cooldown?: number;             // Runs before can use again
    duration?: number;
  };

  // Effect
  effect: AbilityEffect;
}

// Example abilities
const unitAbilities: Ability[] = [
  // Test abilities
  {
    id: "integration_slash",
    name: "Integration Slash",
    description: "Test multiple services in one sweep",
    icon: "⚔️",
    unlockLevel: 3,
    type: "passive",
    effect: {
      type: "coverage_boost",
      scope: "integration",
      bonus: 0.2
    }
  },
  {
    id: "mock_shield",
    name: "Mock Shield",
    description: "Isolate from external dependencies",
    icon: "🛡️",
    unlockLevel: 5,
    type: "passive",
    effect: {
      type: "isolation",
      reduceFlakinessBy: 0.3
    }
  },
  {
    id: "parallel_strike",
    name: "Parallel Strike",
    description: "Execute test cases in parallel",
    icon: "⚡",
    unlockLevel: 7,
    type: "passive",
    effect: {
      type: "speed_boost",
      multiplier: 0.5
    }
  },
  {
    id: "regression_hunter",
    name: "Regression Hunter",
    description: "Automatically runs when related code changes",
    icon: "🎯",
    unlockLevel: 10,
    type: "triggered",
    effect: {
      type: "auto_trigger",
      on: "related_file_change"
    }
  },

  // Build abilities
  {
    id: "incremental_build",
    name: "Incremental Build",
    description: "Only rebuild changed components",
    icon: "🔄",
    unlockLevel: 3,
    type: "passive",
    effect: {
      type: "speed_boost",
      condition: "partial_change",
      multiplier: 0.7
    }
  },
  {
    id: "cache_mastery",
    name: "Cache Mastery",
    description: "Perfect cache utilization",
    icon: "💾",
    unlockLevel: 7,
    type: "passive",
    effect: {
      type: "cache_hit_boost",
      bonus: 0.25
    }
  }
];
```

#### Unit Interactions

Units can interact with each other:

```typescript
interface UnitInteraction {
  // Synergies - units that work well together
  synergies: {
    // Test + corresponding lib = coverage bonus
    testWithLib: {
      condition: (test, lib) => test.tests(lib);
      bonus: { atk: 10 };
    };

    // Multiple tests for same code = defense bonus
    testRedundancy: {
      condition: (tests) => tests.length >= 3;
      bonus: { def: 5, perExtra: 2 };
    };

    // Integration + unit tests = full coverage
    layeredTesting: {
      condition: (unit, integration) => bothPass;
      bonus: { lck: 10 };
    };
  };

  // Combat (debugging/fixing)
  combat: {
    // When tests find bugs
    bugEncounter: {
      attacker: Unit;              // The test
      defender: Bug;               // The bug found
      damage: (atk, bugSeverity) => atk * (1 - bugSeverity * 0.1);
      rewards: { xp: 100, gold: 50 };
    };

    // When tests fail
    testFailure: {
      unit: Unit;
      damage: 20;                  // HP loss
      debuff: "wounded";           // Status effect
      recovery: "fix_and_pass";
    };
  };
}
```

#### Unit Lifecycle

```typescript
interface UnitLifecycle {
  states: {
    // Birth: Unit created from build target
    created: {
      trigger: "build_target_defined";
      initialStats: "computed_from_first_run";
      initialLevel: 1;
    };

    // Active: Unit regularly runs
    active: {
      trigger: "scheduled_or_manual_runs";
      statUpdates: "after_each_run";
      xpGain: "based_on_results";
    };

    // Dormant: Unit hasn't run in a while
    dormant: {
      trigger: "no_runs_for_30_days";
      effects: ["stats_decay", "fog_returns"];
      reactivation: "next_run";
    };

    // Retired: Unit deleted or deprecated
    retired: {
      trigger: "target_removed_or_manual";
      effects: ["remove_from_map", "archive_history"];
      memorial: "hall_of_fame_if_level_10+";
    };
  };

  // Stat decay for inactive units
  decay: {
    startAfter: "7_days_no_run";
    rate: "1_point_per_day";
    minimum: "50%_of_max";
    prevention: "run_unit";
  };
}
```

### Advisors: Subagents

Specialized AI advisors appear in a council panel, Civ-style. Each advisor is a specialized Claude subagent with domain expertise.

#### Advisor Panel

```
┌─────────────────────────────────────────────────────────────────┐
│                      ROYAL COUNCIL                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            │
│  │  🏛️     │  │  🛡️     │  │  📚     │  │  ⚖️     │            │
│  │Architect│  │Sentinel │  │Chroniclr│  │Quarter. │            │
│  │  ●●●○○  │  │  ●●○○○  │  │  ●○○○○  │  │  ●●●●○  │            │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘            │
│       │            │            │            │                  │
│  ┌────┴────────────┴────────────┴────────────┴────┐            │
│  │                                                 │            │
│  │  🏛️ Architect:                                 │            │
│  │  "My liege, I've observed troubling patterns   │            │
│  │   in the eastern modules. The auth package     │            │
│  │   now depends on 47 other packages - this      │            │
│  │   coupling threatens our realm's stability."   │            │
│  │                                                 │            │
│  │  [Show Details] [Propose Solution] [Dismiss]   │            │
│  │                                                 │            │
│  └─────────────────────────────────────────────────┘            │
│                                                                  │
│  ┌─────────┐  ┌─────────┐                                       │
│  │  🔍     │  │  🤝     │  ← More advisors unlock via tech tree │
│  │Inquistr│  │Diplomat │                                       │
│  │  ●●●●● │  │  ●●●○○  │                                       │
│  └─────────┘  └─────────┘                                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### Advisor Data Model

```typescript
interface Advisor {
  id: string;
  name: string;
  title: string;
  icon: string;
  portrait: string;                // Character art

  // Domain expertise
  domain: AdvisorDomain;
  expertise: string[];             // Specific topics
  tools: string[];                 // Claude tools this advisor uses

  // Personality
  personality: AdvisorPersonality;
  voiceStyle: VoiceStyle;
  catchphrases: string[];

  // State
  mood: AdvisorMood;
  urgency: number;                 // 0-100, affects notification priority
  lastConsulted?: Date;
  relationship: number;            // 0-100, improves with interaction

  // Insights queue
  pendingInsights: Insight[];
  deliveredInsights: Insight[];

  // Unlock status
  unlocked: boolean;
  unlockRequirement?: UnlockRequirement;
}

type AdvisorDomain =
  | "architecture"
  | "security"
  | "documentation"
  | "performance"
  | "testing"
  | "dependencies"
  | "devops"
  | "accessibility"
  | "internationalization";

interface AdvisorPersonality {
  formality: "casual" | "formal" | "archaic";
  enthusiasm: "reserved" | "moderate" | "enthusiastic";
  directness: "diplomatic" | "balanced" | "blunt";
  humor: "serious" | "occasional" | "frequent";
  metaphorStyle: "medieval" | "military" | "nautical" | "mystical";
}

interface VoiceStyle {
  greeting: string[];
  concern: string[];
  praise: string[];
  urgent: string[];
  suggestion: string[];
}

type AdvisorMood =
  | "pleased"      // Domain is in good shape
  | "concerned"    // Issues detected
  | "alarmed"      // Critical issues
  | "neutral"      // Nothing notable
  | "excited";     // Good news to share
```

#### Core Advisors

```typescript
const coreAdvisors: Advisor[] = [
  {
    id: "architect",
    name: "The Architect",
    title: "Master of Structures",
    icon: "🏛️",
    domain: "architecture",
    expertise: [
      "dependency_analysis",
      "module_structure",
      "circular_dependencies",
      "coupling_metrics",
      "code_organization"
    ],
    tools: ["Glob", "Grep", "Read", "Task"],
    personality: {
      formality: "formal",
      enthusiasm: "moderate",
      directness: "diplomatic",
      humor: "occasional",
      metaphorStyle: "medieval"
    },
    voiceStyle: {
      greeting: [
        "My liege, I've been studying the realm's foundations.",
        "The structures of our domain require your attention."
      ],
      concern: [
        "I've observed troubling patterns in the eastern modules.",
        "Our foundations show signs of strain."
      ],
      praise: [
        "The new architecture is sound and elegant.",
        "Our structures grow stronger with each improvement."
      ],
      urgent: [
        "Sire! A circular dependency threatens to collapse the eastern wing!",
        "Critical structural failure imminent!"
      ],
      suggestion: [
        "Perhaps we might consider extracting a shared module?",
        "I propose we establish clearer boundaries between domains."
      ]
    },
    unlocked: true
  },

  {
    id: "sentinel",
    name: "The Sentinel",
    title: "Guardian of the Gates",
    icon: "🛡️",
    domain: "security",
    expertise: [
      "vulnerability_scanning",
      "dependency_audit",
      "secret_detection",
      "auth_patterns",
      "input_validation"
    ],
    tools: ["Bash", "Grep", "Read", "WebFetch"],
    personality: {
      formality: "formal",
      enthusiasm: "reserved",
      directness: "blunt",
      humor: "serious",
      metaphorStyle: "military"
    },
    voiceStyle: {
      greeting: [
        "The watch continues. I have my report.",
        "I've completed my patrol of the perimeter."
      ],
      concern: [
        "Vulnerabilities detected in our defenses.",
        "The enemy may exploit these weaknesses."
      ],
      praise: [
        "Our defenses hold strong.",
        "The realm is secure... for now."
      ],
      urgent: [
        "BREACH! Critical vulnerability in production!",
        "We are under attack! Immediate action required!"
      ],
      suggestion: [
        "We must patch this vulnerability immediately.",
        "I recommend implementing additional safeguards."
      ]
    },
    unlocked: true
  },

  {
    id: "chronicler",
    name: "The Chronicler",
    title: "Keeper of Knowledge",
    icon: "📚",
    domain: "documentation",
    expertise: [
      "documentation_coverage",
      "api_docs",
      "readme_quality",
      "code_comments",
      "changelog"
    ],
    tools: ["Read", "Glob", "Grep"],
    personality: {
      formality: "archaic",
      enthusiasm: "enthusiastic",
      directness: "diplomatic",
      humor: "occasional",
      metaphorStyle: "mystical"
    },
    voiceStyle: {
      greeting: [
        "The scrolls speak of many changes, my liege.",
        "I've been cataloging the wisdom of our realm."
      ],
      concern: [
        "Much knowledge remains unrecorded.",
        "The people cannot understand the ancient scrolls."
      ],
      praise: [
        "Our documentation illuminates even the darkest corners.",
        "Future generations will thank us for these records."
      ],
      urgent: [
        "The API has changed but the scrolls lie!",
        "Outdated documentation leads our people astray!"
      ],
      suggestion: [
        "Perhaps we should document this arcane function?",
        "A README would guide travelers through this module."
      ]
    },
    unlocked: true
  },

  {
    id: "quartermaster",
    name: "The Quartermaster",
    title: "Master of Resources",
    icon: "⚖️",
    domain: "performance",
    expertise: [
      "build_time",
      "bundle_size",
      "memory_usage",
      "cpu_profiling",
      "cache_efficiency"
    ],
    tools: ["Bash", "Read", "Task"],
    personality: {
      formality: "casual",
      enthusiasm: "moderate",
      directness: "blunt",
      humor: "frequent",
      metaphorStyle: "medieval"
    },
    voiceStyle: {
      greeting: [
        "Let's talk numbers, shall we?",
        "I've been counting the coins... er, milliseconds."
      ],
      concern: [
        "We're burning through resources like there's no tomorrow.",
        "Build times are eating into our productivity."
      ],
      praise: [
        "Efficiency is up! The troops are pleased.",
        "We're running lean and mean."
      ],
      urgent: [
        "The build is taking FOREVER! We're losing daylight!",
        "Memory usage is through the roof!"
      ],
      suggestion: [
        "Have you considered lazy loading this module?",
        "We could shave 30% off build time with better caching."
      ]
    },
    unlocked: true
  },

  {
    id: "inquisitor",
    name: "The Inquisitor",
    title: "Seeker of Truth",
    icon: "🔍",
    domain: "testing",
    expertise: [
      "test_coverage",
      "flaky_tests",
      "test_quality",
      "mutation_testing",
      "integration_testing"
    ],
    tools: ["Bash", "Read", "Grep", "Task"],
    personality: {
      formality: "formal",
      enthusiasm: "enthusiastic",
      directness: "blunt",
      humor: "serious",
      metaphorStyle: "mystical"
    },
    voiceStyle: {
      greeting: [
        "The truth shall be revealed through testing.",
        "I've been examining the evidence."
      ],
      concern: [
        "42% of our realm remains unverified.",
        "These tests speak lies - they are flaky."
      ],
      praise: [
        "Our tests stand vigilant and true.",
        "Full coverage! No bug shall escape our gaze."
      ],
      urgent: [
        "The tests have failed! Something is terribly wrong!",
        "A regression has been detected!"
      ],
      suggestion: [
        "This function lacks proper interrogation... er, testing.",
        "We should question this code more thoroughly."
      ]
    },
    unlocked: false,
    unlockRequirement: {
      type: "coverage",
      threshold: 50
    }
  },

  {
    id: "diplomat",
    name: "The Diplomat",
    title: "Ambassador to Foreign Realms",
    icon: "🤝",
    domain: "dependencies",
    expertise: [
      "dependency_updates",
      "breaking_changes",
      "license_compliance",
      "vulnerability_advisories",
      "api_compatibility"
    ],
    tools: ["Bash", "WebFetch", "Read"],
    personality: {
      formality: "formal",
      enthusiasm: "moderate",
      directness: "diplomatic",
      humor: "occasional",
      metaphorStyle: "nautical"
    },
    voiceStyle: {
      greeting: [
        "News from the foreign kingdoms, my liege.",
        "I've been in contact with our allies."
      ],
      concern: [
        "The npm kingdoms demand tribute... er, updates.",
        "Our allies have deprecated their old treaties."
      ],
      praise: [
        "All dependencies are current and compatible.",
        "Our foreign relations are strong."
      ],
      urgent: [
        "A critical vulnerability in our ally's code!",
        "Breaking changes incoming! Prepare the defenses!"
      ],
      suggestion: [
        "Perhaps we should upgrade to the latest version?",
        "A new alliance with this library could benefit us."
      ]
    },
    unlocked: false,
    unlockRequirement: {
      type: "dependencies",
      threshold: 10
    }
  }
];
```

#### Advisor Insights System

Advisors proactively generate insights:

```typescript
interface Insight {
  id: string;
  advisor: string;                 // Advisor ID
  type: InsightType;
  severity: InsightSeverity;
  title: string;
  message: string;                 // In advisor's voice
  technicalDetails: string;        // Actual technical info

  // Context
  relatedFiles: string[];
  relatedUnits: string[];
  relatedBuildings: string[];

  // Actions
  suggestedActions: SuggestedAction[];

  // State
  createdAt: Date;
  expiresAt?: Date;
  dismissed: boolean;
  actedUpon: boolean;

  // Priority
  priority: number;                // Computed from severity + urgency
}

type InsightType =
  | "warning"
  | "opportunity"
  | "achievement"
  | "suggestion"
  | "alert"
  | "news";

type InsightSeverity =
  | "info"
  | "low"
  | "medium"
  | "high"
  | "critical";

interface SuggestedAction {
  id: string;
  label: string;
  description: string;
  action: () => Promise<void>;

  // Cost/benefit
  estimatedTime?: number;
  estimatedImpact?: string;

  // Can Claude do this automatically?
  automatable: boolean;
  requiresConfirmation: boolean;
}

// Insight generation triggers
interface InsightTrigger {
  advisor: string;
  condition: (state: GameState) => boolean;
  generate: (state: GameState) => Insight;
  cooldown: number;                // Don't repeat for N hours
}

const insightTriggers: InsightTrigger[] = [
  // Architect: Circular dependency detected
  {
    advisor: "architect",
    condition: (s) => s.hasCircularDependency(),
    generate: (s) => ({
      type: "warning",
      severity: "high",
      title: "Circular Dependency Detected",
      message: "A serpent eating its own tail! The modules form a forbidden circle.",
      technicalDetails: `Cycle: ${s.getCircularPath().join(" → ")}`,
      suggestedActions: [
        {
          label: "Break the Cycle",
          description: "Extract shared code to a new module",
          automatable: true,
          requiresConfirmation: true
        }
      ]
    }),
    cooldown: 24
  },

  // Sentinel: Vulnerability found
  {
    advisor: "sentinel",
    condition: (s) => s.hasVulnerabilities(),
    generate: (s) => ({
      type: "alert",
      severity: "critical",
      title: "Security Vulnerability",
      message: "The walls have been breached! A known vulnerability lurks within.",
      technicalDetails: s.getVulnerabilityReport(),
      suggestedActions: [
        {
          label: "Patch Now",
          description: "Update affected dependency",
          automatable: true,
          requiresConfirmation: false
        }
      ]
    }),
    cooldown: 1
  },

  // Inquisitor: Coverage dropped
  {
    advisor: "inquisitor",
    condition: (s) => s.coverageDropped(5),
    generate: (s) => ({
      type: "warning",
      severity: "medium",
      title: "Coverage Declining",
      message: "The darkness spreads. 5% more of our realm now lies unexamined.",
      technicalDetails: `Coverage: ${s.previousCoverage}% → ${s.currentCoverage}%`,
      suggestedActions: [
        {
          label: "Write Tests",
          description: "Add tests for uncovered code",
          automatable: false,
          requiresConfirmation: false
        }
      ]
    }),
    cooldown: 4
  }
];
```

#### Advisor Consultation

Players can actively consult advisors:

```typescript
interface AdvisorConsultation {
  // Start a consultation
  consult(advisorId: string, topic?: string): Promise<ConsultationResult>;

  // Ask follow-up questions
  askFollowUp(consultationId: string, question: string): Promise<Response>;

  // Request specific analysis
  requestAnalysis(advisorId: string, target: AnalysisTarget): Promise<Analysis>;

  // Get recommendation
  getRecommendation(advisorId: string, situation: Situation): Promise<Recommendation>;
}

interface ConsultationResult {
  advisor: Advisor;
  greeting: string;
  currentAssessment: string;
  topConcerns: Insight[];
  opportunities: Insight[];

  // Conversation state
  conversationId: string;
  canAskFollowUp: boolean;
}

// Example consultation flow:
// User clicks on Architect
// → "My liege, I've been studying the realm's foundations.
//    Currently, I'm most concerned about the coupling in src/auth.
//    Would you like me to elaborate?"
// User: "Tell me more about the auth coupling"
// → "The auth module has grown to depend on 47 other packages.
//    This creates fragility - a change anywhere can break auth.
//    I suggest we extract the core auth logic into a separate,
//    more isolated module. Shall I draw up the plans?"
```

#### Advisor Relationship System

Advisors remember your interactions:

```typescript
interface AdvisorRelationship {
  advisorId: string;
  relationshipLevel: number;       // 0-100

  // History
  totalConsultations: number;
  insightsActedUpon: number;
  insightsDismissed: number;

  // Unlocks at relationship levels
  perks: {
    10: "More detailed insights";
    25: "Proactive suggestions";
    50: "Instant analysis";
    75: "Priority notifications";
    100: "Advisor becomes mentor";
  };

  // Relationship affects behavior
  effects: {
    moreTrusting: boolean;         // Shares more concerns
    moreHelpful: boolean;          // Better suggestions
    lessFormal: boolean;           // Relaxed personality
    shareSecrets: boolean;         // Hidden insights
  };
}
```

Advisors proactively surface insights and can be consulted for their domain.

### Belts: Dependency & Data Flow (Factorio-style)

Visualize relationships between files as animated flowing belts, inspired by Factorio's transport belt system.

#### Belt Visualization

```
┌─────────────────────────────────────────────────────────────────────┐
│                        DEPENDENCY FLOW                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────┐                                                        │
│  │ types.ts │                                                        │
│  └────┬─────┘                                                        │
│       │                                                              │
│       ▼ ════════════════════╗                                        │
│  ┌──────────┐               ║                                        │
│  │ config.ts│               ║                                        │
│  └────┬─────┘               ║                                        │
│       │                     ║                                        │
│       ▼ ══════════╗         ║                                        │
│  ┌──────────┐     ║         ║                                        │
│  │ utils.ts │     ║         ║                                        │
│  └────┬─────┘     ║         ║                                        │
│       │           ║         ║                                        │
│       ▼           ▼         ▼                                        │
│  ┌──────────┐═══════════════════▶┌──────────┐                        │
│  │ auth.ts  │════════════════════│  api.ts  │═══▶ [OUTPUT]           │
│  └──────────┘◀═══════════════════└──────────┘                        │
│       ▲                               │                              │
│       │                               │                              │
│       └───────────⚠️ CYCLE ───────────┘                              │
│                                                                      │
│  Legend: ═══ Import  ─── Inheritance  ··· Composition               │
│          ⚠️  Problem   ⚡ Hot path      💤 Cold path                  │
└─────────────────────────────────────────────────────────────────────┘
```

#### Belt Data Model

```typescript
interface Belt {
  id: string;
  source: string;                  // Source file/module ID
  target: string;                  // Target file/module ID

  // Relationship type
  type: BeltType;
  subtype?: string;                // More specific (e.g., "default import")

  // Visual properties
  style: BeltStyle;
  path: BeltPath;                  // Routing information
  animation: BeltAnimation;

  // Metrics
  metrics: BeltMetrics;

  // State
  highlighted: boolean;
  selected: boolean;
  problematic: boolean;
}

type BeltType =
  | "import"           // ES/CommonJS import
  | "inheritance"      // extends/implements
  | "composition"      // contains/uses
  | "dependency"       // Build dependency
  | "runtime"          // Runtime dependency
  | "test"             // Test depends on
  | "type_only";       // Type import (no runtime)

interface BeltStyle {
  color: string;                   // Based on type
  width: number;                   // 1-5, based on coupling
  pattern: "solid" | "dashed" | "dotted";
  glow: boolean;                   // For hot paths
  opacity: number;

  // Animated particles on belt
  particles: {
    enabled: boolean;
    speed: number;                 // Pixels per second
    density: number;               // Particles per 100px
    sprite: string;                // What flows on the belt
  };
}

interface BeltPath {
  // Bezier curve control points
  points: Point[];
  curveType: "straight" | "bezier" | "orthogonal";

  // Routing
  avoidance: string[];             // IDs of nodes to route around
  preferredSide: "left" | "right" | "top" | "bottom";

  // Junctions
  junctions: Junction[];           // Where belts merge/split
}

interface BeltMetrics {
  // Static analysis
  couplingStrength: number;        // 0-100
  importCount: number;             // Number of imports
  symbolCount: number;             // Number of symbols used

  // Runtime/usage
  callFrequency: number;           // How often this path is used
  lastUsed?: Date;
  hotness: number;                 // 0-100, based on recent activity

  // Quality
  isCircular: boolean;
  isBidirectional: boolean;
  complexity: number;              // Relationship complexity
}

interface BeltAnimation {
  enabled: boolean;
  direction: "forward" | "reverse" | "bidirectional";
  speed: number;                   // Based on usage frequency

  // Particle types (what flows on the belt)
  particleType: ParticleType;
  particleColor: string;
}

type ParticleType =
  | "data"             // Generic data flow
  | "type"             // Type information
  | "function"         // Function calls
  | "event"            // Events/callbacks
  | "error";           // Error propagation
```

#### Belt Colors and Types

```typescript
const beltStyles: Record<BeltType, BeltStyle> = {
  import: {
    color: "#4CAF50",              // Green
    pattern: "solid",
    particles: { sprite: "📦" }
  },
  inheritance: {
    color: "#2196F3",              // Blue
    pattern: "solid",
    particles: { sprite: "🧬" }
  },
  composition: {
    color: "#9C27B0",              // Purple
    pattern: "dashed",
    particles: { sprite: "🔗" }
  },
  dependency: {
    color: "#FF9800",              // Orange
    pattern: "solid",
    particles: { sprite: "⚙️" }
  },
  runtime: {
    color: "#F44336",              // Red
    pattern: "solid",
    particles: { sprite: "⚡" }
  },
  test: {
    color: "#00BCD4",              // Cyan
    pattern: "dotted",
    particles: { sprite: "🧪" }
  },
  type_only: {
    color: "#607D8B",              // Gray
    pattern: "dotted",
    particles: { sprite: "📝" }
  }
};
```

#### Belt Width (Coupling Strength)

```typescript
interface CouplingCalculation {
  // Width levels
  levels: {
    1: { coupling: "0-10", description: "Loose coupling" },
    2: { coupling: "11-25", description: "Light coupling" },
    3: { coupling: "26-50", description: "Moderate coupling" },
    4: { coupling: "51-75", description: "Tight coupling" },
    5: { coupling: "76-100", description: "Very tight coupling" }
  };

  // Factors that increase coupling
  factors: {
    importCount: number;           // More imports = tighter
    deepImports: number;           // Importing internals
    bidirectional: number;         // Mutual dependencies
    symbolSpread: number;          // Using many different things
    changeCorrelation: number;     // Changed together historically
  };

  // Calculate coupling score
  calculate(source: string, target: string): number;
}
```

#### Belt Interactions

```typescript
interface BeltInteraction {
  // Mouse events
  onHover(belt: Belt): void {
    // Show tooltip with relationship details
    // Highlight connected nodes
    // Show flow animation faster
  }

  onClick(belt: Belt): void {
    // Open relationship detail panel
    // Show all symbols flowing through
    // Option to navigate to source/target
  }

  onRightClick(belt: Belt): void {
    // Context menu:
    // - Go to source
    // - Go to target
    // - Show in editor
    // - Find usages
    // - Analyze coupling
  }

  // Filtering
  filters: {
    byType: BeltType[];
    byMinCoupling: number;
    byHotness: number;
    showCircular: boolean;
    showBidirectional: boolean;
  };

  // Highlighting modes
  highlightModes: {
    "none": "No special highlighting",
    "hot-paths": "Highlight frequently used connections",
    "problems": "Highlight circular/tight coupling",
    "selected-node": "Highlight all connections to selected",
    "trace": "Trace dependency chain"
  };
}
```

#### Problem Detection

Belts visualize architectural problems:

```typescript
interface BeltProblem {
  type: BeltProblemType;
  severity: "info" | "warning" | "error";
  belts: Belt[];
  description: string;
  suggestion: string;
}

type BeltProblemType =
  | "circular"           // A → B → C → A
  | "bidirectional"      // A ↔ B
  | "hub"                // One node with too many connections
  | "chain"              // Deep dependency chain
  | "orphan"             // Isolated node
  | "tight_coupling"     // Coupling score > 75
  | "layer_violation";   // Breaks architectural layers

const problemVisuals: Record<BeltProblemType, Visual> = {
  circular: {
    color: "#FF0000",
    animation: "pulse",
    icon: "⚠️",
    label: "CYCLE"
  },
  bidirectional: {
    color: "#FF6600",
    animation: "bidirectional-flow",
    icon: "↔️",
    label: "MUTUAL"
  },
  hub: {
    color: "#FFCC00",
    animation: "overload",
    icon: "🕸️",
    label: "HUB"
  },
  tight_coupling: {
    color: "#FF3300",
    animation: "strain",
    icon: "🔒",
    label: "COUPLED"
  }
};
```

#### Belt Layouts

Different layout algorithms for belt routing:

```typescript
interface BeltLayout {
  algorithm: BeltLayoutAlgorithm;

  // Orthogonal routing (like circuit boards)
  orthogonal: {
    preferHorizontal: boolean;
    cornerRadius: number;
    minSpacing: number;
    avoidCrossings: boolean;
  };

  // Curved routing (like Factorio)
  curved: {
    curveTension: number;
    bundleParallel: boolean;       // Group parallel belts
    minimizeCrossings: boolean;
  };

  // Hierarchical (tree-like)
  hierarchical: {
    direction: "TB" | "BT" | "LR" | "RL";
    levelSeparation: number;
    nodeSeparation: number;
  };
}

type BeltLayoutAlgorithm =
  | "orthogonal"       // Right angles only
  | "curved"           // Smooth curves
  | "hierarchical"     // Tree layout
  | "force"            // Force-directed
  | "radial";          // Radial from center
```

#### Animated Flow

Belts show animated flow to indicate activity:

```typescript
interface FlowAnimation {
  // Particle settings
  particle: {
    shape: "circle" | "square" | "icon";
    size: number;
    color: string;
    icon?: string;
  };

  // Movement
  speed: {
    base: number;                  // Pixels per second
    hotMultiplier: number;         // Speed up for hot paths
    coldMultiplier: number;        // Slow down for cold paths
  };

  // Density
  density: {
    base: number;                  // Particles per 100px
    scaleWithUsage: boolean;       // More particles = more used
  };

  // Effects
  effects: {
    trail: boolean;                // Leave fade trail
    glow: boolean;                 // Glow effect
    pulse: boolean;                // Periodic pulse
  };
}

// Real-time updates based on runtime data
interface LiveFlowData {
  belt: string;
  callsPerSecond: number;
  dataVolume: number;
  latency: number;

  // Update animation
  updateAnimation(): void {
    this.belt.animation.speed = this.callsPerSecond * 10;
    this.belt.animation.density = Math.min(this.dataVolume / 100, 10);
  }
}
```

- **Belt color** = Type of relationship (import, inheritance, composition)
- **Belt width** = Coupling strength
- **Bottlenecks** = Many belts converging (high fan-in)
- **Loops** = Circular dependencies (warning!)
- **Throughput** = How often the connection is exercised

---

## Resource System

Resources appear in a top bar, RTS-style. The system is **extensible** - applications define their own resource types.

### Resource Bar Visualization

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ 🪙 Context    │ 😊 Coverage │ ⚡ CI     │ 🐙 API     │ 💔 Errors  │ 🚀 Deploy │
│ 42k/100k      │ 73%         │ 2/4 slots │ 4.2k/5k    │ 12% left   │ Ready     │
│ ████████░░░░  │ ████████░░  │ ██░░      │ █████████░ │ ██░░░░░░░░ │ ●●●○      │
│ ▼ 2.1k/hr     │ ▲ +2%       │           │ ▼ 100/min  │ ▼ 0.3%/hr  │           │
└──────────────────────────────────────────────────────────────────────────────┘
     │                                          │
     ▼                                          ▼
  [Click to expand                         [Hover shows
   spending details]                        regeneration]
```

### Resource Data Model

```typescript
interface Resource {
  id: string;
  name: string;
  shortName: string;               // For compact display
  icon: string;
  description: string;

  // Classification
  metaphor: ResourceMetaphor;
  category: ResourceCategory;

  // Current state
  current: number;
  max: number | (() => number);
  min: number;

  // Display
  displayFormat: DisplayFormat;
  barColor: string;
  barColorThresholds: ColorThreshold[];

  // Computation
  compute: () => number | Promise<number>;
  computeMax?: () => number | Promise<number>;
  refreshInterval: number;         // ms between updates

  // Thresholds
  thresholds: ResourceThresholds;

  // Economy
  economy: ResourceEconomy;

  // Trends
  history: ResourceHistory;
  trend: ResourceTrend;

  // Actions
  actions: ResourceAction[];

  // Notifications
  notifications: NotificationConfig;
}

type ResourceMetaphor =
  | "currency"       // Spent and earned (gold)
  | "satisfaction"   // Percentage-based (happiness)
  | "capacity"       // Slots/units (production)
  | "mana"           // Regenerating pool
  | "health"         // Depletable, regenerates
  | "reputation";    // Grows over time

type ResourceCategory =
  | "core"           // Always visible
  | "development"    // Dev-related
  | "production"     // Deployment/runtime
  | "external"       // Third-party APIs
  | "custom";        // User-defined

interface DisplayFormat {
  type: "number" | "percentage" | "ratio" | "time" | "custom";
  precision: number;
  suffix?: string;
  formatter?: (value: number) => string;
}

interface ResourceThresholds {
  critical: number;                // Red alert
  warning: number;                 // Yellow warning
  good: number;                    // Green/normal
  excellent?: number;              // Blue/bonus

  // Threshold direction
  direction: "higher_is_better" | "lower_is_better";
}

interface ColorThreshold {
  threshold: number;
  color: string;
  pulseWhenBelow?: boolean;
}
```

### Resource Economy

Track how resources flow:

```typescript
interface ResourceEconomy {
  // What produces this resource
  producers: ResourceProducer[];

  // What consumes this resource
  consumers: ResourceConsumer[];

  // Flow rates
  productionRate: number;          // Units per hour
  consumptionRate: number;         // Units per hour
  netFlow: number;                 // Production - consumption

  // Regeneration
  regeneration?: {
    rate: number;                  // Units per hour
    cap: number;                   // Max regeneration target
    conditions: string[];          // When regeneration applies
  };

  // Transactions
  transactions: Transaction[];
  transactionLimit: number;        // Keep last N
}

interface ResourceProducer {
  id: string;
  name: string;
  type: "building" | "unit" | "action" | "passive" | "external";
  rate: number;                    // Units per hour
  conditions?: string[];           // When this produces

  // For display
  icon: string;
  description: string;
}

interface ResourceConsumer {
  id: string;
  name: string;
  type: "building" | "unit" | "action" | "passive";
  rate: number;                    // Units per hour (negative)
  conditions?: string[];

  // Cost per action (for non-passive)
  costPerUse?: number;

  icon: string;
  description: string;
}

interface Transaction {
  id: string;
  timestamp: Date;
  amount: number;                  // Positive = gain, negative = spend
  source: string;                  // Producer/consumer ID
  description: string;
  category: string;
}
```

### Resource History and Trends

```typescript
interface ResourceHistory {
  // Time series data
  dataPoints: DataPoint[];
  resolution: "minute" | "hour" | "day";
  retention: number;               // How long to keep

  // Aggregations
  hourlyAverage: number[];
  dailyAverage: number[];
  weeklyAverage: number[];
}

interface DataPoint {
  timestamp: Date;
  value: number;
  max: number;
  events: string[];                // Notable events at this time
}

interface ResourceTrend {
  direction: "up" | "down" | "stable";
  rate: number;                    // Change per hour
  prediction: {
    depletionTime?: Date;          // When will it run out?
    fullTime?: Date;               // When will it be full?
    confidence: number;
  };

  // Display
  arrow: "▲" | "▼" | "─";
  color: string;
}
```

### Built-in Resources

```typescript
const coreResources: Resource[] = [
  // Context (Gold)
  {
    id: "context",
    name: "Context Window",
    shortName: "Context",
    icon: "🪙",
    metaphor: "currency",
    category: "core",
    description: "Token budget for Claude conversations",

    compute: () => claudeSession.tokensRemaining,
    computeMax: () => claudeSession.maxTokens,

    displayFormat: {
      type: "number",
      precision: 0,
      formatter: (v) => v >= 1000 ? `${(v/1000).toFixed(1)}k` : v.toString()
    },

    thresholds: {
      critical: 5000,
      warning: 20000,
      good: 50000,
      direction: "higher_is_better"
    },

    economy: {
      producers: [
        { id: "new_session", name: "New Session", rate: 0, type: "action" }
      ],
      consumers: [
        { id: "tool_calls", name: "Tool Calls", type: "passive" },
        { id: "responses", name: "Responses", type: "passive" },
        { id: "file_reads", name: "File Reads", type: "passive" }
      ]
    },

    notifications: {
      onCritical: "Context running low! Consider summarizing.",
      onWarning: "Context at 20% - plan your remaining queries."
    }
  },

  // Coverage (Happiness)
  {
    id: "coverage",
    name: "Test Coverage",
    shortName: "Coverage",
    icon: "😊",
    metaphor: "satisfaction",
    category: "core",
    description: "Percentage of code covered by tests",

    compute: async () => {
      const report = await getCoverageReport();
      return report.lineCoverage;
    },
    max: 100,

    displayFormat: {
      type: "percentage",
      precision: 0
    },

    thresholds: {
      critical: 30,
      warning: 60,
      good: 80,
      excellent: 95,
      direction: "higher_is_better"
    },

    economy: {
      producers: [
        { id: "add_tests", name: "Adding Tests", type: "action" }
      ],
      consumers: [
        { id: "new_code", name: "New Untested Code", type: "passive" }
      ]
    }
  },

  // CI Capacity (Production)
  {
    id: "ci_capacity",
    name: "CI Capacity",
    shortName: "CI",
    icon: "⚡",
    metaphor: "capacity",
    category: "core",
    description: "Available CI/CD runner slots",

    compute: () => ciSystem.availableSlots,
    computeMax: () => ciSystem.totalSlots,

    displayFormat: {
      type: "ratio"
    },

    thresholds: {
      critical: 0,
      warning: 1,
      good: 2,
      direction: "higher_is_better"
    },

    economy: {
      producers: [
        { id: "build_complete", name: "Build Completes", type: "passive" }
      ],
      consumers: [
        { id: "build_start", name: "Build Starts", type: "action", costPerUse: 1 }
      ]
    }
  }
];
```

### Custom Resource Examples

```typescript
// API Rate Limits (Mana)
const githubApiResource: Resource = {
  id: "github_api",
  name: "GitHub API",
  shortName: "API",
  icon: "🐙",
  metaphor: "mana",
  category: "external",
  description: "GitHub API rate limit remaining",

  compute: async () => {
    const limits = await github.getRateLimit();
    return limits.remaining;
  },
  max: 5000,

  displayFormat: {
    type: "number",
    formatter: (v) => `${(v/1000).toFixed(1)}k`
  },

  thresholds: {
    critical: 100,
    warning: 500,
    good: 2000,
    direction: "higher_is_better"
  },

  economy: {
    regeneration: {
      rate: 5000,                  // Resets hourly
      cap: 5000,
      conditions: ["hourly_reset"]
    },
    consumers: [
      { id: "api_call", name: "API Calls", type: "action", costPerUse: 1 }
    ]
  }
};

// Error Budget (Currency)
const errorBudgetResource: Resource = {
  id: "error_budget",
  name: "Error Budget",
  shortName: "Errors",
  icon: "💔",
  metaphor: "currency",
  category: "production",
  description: "SLO error budget remaining this month",

  compute: async () => {
    const slo = await monitoring.getSLOStatus();
    return slo.errorBudgetRemaining;
  },
  max: 100,

  displayFormat: {
    type: "percentage",
    precision: 1,
    suffix: " left"
  },

  thresholds: {
    critical: 10,
    warning: 30,
    good: 70,
    direction: "higher_is_better"
  },

  notifications: {
    onCritical: "Error budget nearly exhausted! Freeze deployments.",
    onWarning: "Error budget below 30% - be cautious with changes."
  }
};

// Technical Debt (Inverse health)
const techDebtResource: Resource = {
  id: "tech_debt",
  name: "Technical Debt",
  shortName: "Debt",
  icon: "🏚️",
  metaphor: "health",
  category: "development",
  description: "Accumulated technical debt score",

  compute: async () => {
    return await analyzer.getTechDebtScore();
  },
  max: 100,

  displayFormat: {
    type: "number",
    suffix: " issues"
  },

  thresholds: {
    critical: 80,
    warning: 50,
    good: 20,
    direction: "lower_is_better"      // Unlike most, lower is better
  },

  economy: {
    producers: [
      { id: "quick_fixes", name: "Quick Fixes", type: "passive" },
      { id: "skipped_tests", name: "Skipped Tests", type: "passive" }
    ],
    consumers: [
      { id: "refactoring", name: "Refactoring", type: "action" },
      { id: "debt_paydown", name: "Debt Paydown", type: "action" }
    ]
  }
};
```

### Resource Actions

Resources can have associated actions:

```typescript
interface ResourceAction {
  id: string;
  name: string;
  icon: string;
  description: string;

  // When available
  available: (resource: Resource) => boolean;

  // What it does
  execute: () => Promise<void>;

  // Cost
  cost?: {
    resource: string;
    amount: number;
  };

  // Cooldown
  cooldown?: number;
  lastUsed?: Date;
}

// Example actions
const resourceActions: ResourceAction[] = [
  // For Context
  {
    id: "summarize_context",
    name: "Summarize",
    icon: "📝",
    description: "Summarize conversation to free up context",
    available: (r) => r.current < r.max * 0.3,
    execute: async () => await claudeSession.summarize()
  },

  // For Coverage
  {
    id: "generate_tests",
    name: "Generate Tests",
    icon: "🧪",
    description: "Auto-generate tests for uncovered code",
    available: (r) => r.current < 80,
    execute: async () => await testGenerator.generateForUncovered(),
    cost: { resource: "context", amount: 5000 }
  },

  // For Error Budget
  {
    id: "freeze_deploys",
    name: "Freeze Deploys",
    icon: "🥶",
    description: "Halt all deployments to preserve error budget",
    available: (r) => r.current < 20,
    execute: async () => await deploySystem.freeze()
  }
];
```

### Resource Notifications

```typescript
interface ResourceNotification {
  resource: string;
  type: "critical" | "warning" | "recovery" | "milestone";
  message: string;
  timestamp: Date;
  acknowledged: boolean;

  // Advisor integration
  advisor?: string;                // Which advisor reports this
  advisorMessage?: string;         // In advisor's voice
}

// Example notifications
const notificationTemplates = {
  context: {
    critical: {
      message: "Context nearly exhausted!",
      advisorMessage: "My liege, our scrolls grow too long! We must summarize or start anew."
    },
    recovery: {
      message: "Context refreshed",
      advisorMessage: "A new session begins. Our memory is clear once more."
    }
  },
  coverage: {
    milestone: {
      message: "Coverage reached 80%!",
      advisorMessage: "Excellent! 80% of our realm now stands verified. The Inquisitor is pleased."
    }
  }
};
```

---

## Deployment & Usage Visualization

### The World Map: Production Environment

Beyond your codebase, visualize deployed services as a living world.

#### Production Realm Visualization

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         PRODUCTION REALM                                 │
│                    ☀️ Clear Skies - All Systems Nominal                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│     🌲🌲           ┌──────────────┐           🌲🌲                       │
│    🌲🌲🌲         │   🏰 Web App  │          🌲🌲🌲                      │
│                    │  ████████░░  │                                      │
│                    │  2.1k rps    │                                      │
│                    │  P99: 45ms   │                                      │
│                    │  👥 12,847   │                                      │
│                    └──────┬───────┘                                      │
│                           │                                              │
│            ═══════════════╪═══════════════                               │
│           ║               ▼               ║                              │
│     ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                   │
│     │  🏛️ Auth    │  │  🏭 API     │  │  📊 Analytics│                   │
│     │  ████████   │  │  ████████   │  │  ██████░░   │                   │
│     │  142 rps    │  │  847 rps    │  │  234 rps    │                   │
│     │  P99: 12ms  │  │  P99: 89ms  │  │  P99: 156ms │                   │
│     └──────┬──────┘  └──────┬──────┘  └─────────────┘                   │
│            │                │                                            │
│            ▼                ▼                                            │
│     ┌─────────────┐  ┌─────────────┐                                    │
│     │  🗄️ DB      │  │  💨 Cache   │     ⚔️ 3 Errors (Barbarians)       │
│     │  ████████░  │  │  █████████  │     🛡️ 0 Attacks                   │
│     │  99.2% up   │  │  HIT: 94%   │                                    │
│     │  234 conn   │  │  1.2M keys  │                                    │
│     └─────────────┘  └─────────────┘                                    │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │ 👥 Population: 12,847 active │ 😊 Happiness: 98.2% │ 📈 +12% hr  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

#### City (Service) Data Model

```typescript
interface City {
  id: string;
  name: string;
  type: CityType;
  icon: string;

  // Location in the realm
  position: { x: number; y: number };
  region: string;                  // Logical grouping

  // Health and status
  health: CityHealth;
  status: CityStatus;

  // Population (traffic)
  population: {
    current: number;               // Active connections/requests
    peak: number;                  // Highest today
    trend: "growing" | "stable" | "declining";
  };

  // Happiness (performance)
  happiness: {
    score: number;                 // 0-100
    factors: HappinessFactor[];
  };

  // Infrastructure
  infrastructure: {
    replicas: number;
    cpu: ResourceUsage;
    memory: ResourceUsage;
    storage?: ResourceUsage;
  };

  // Connections to other cities
  tradeRoutes: TradeRoute[];

  // Threats
  threats: Threat[];

  // Visual
  sprite: CitySprite;
  animation: CityAnimation;
}

type CityType =
  | "castle"       // Main application/frontend
  | "temple"       // Auth/identity service
  | "market"       // API gateway
  | "factory"      // Worker/processor
  | "warehouse"    // Database
  | "tower"        // Cache
  | "outpost"      // Edge/CDN
  | "harbor";      // External integration

interface CityHealth {
  current: number;                 // 0-100
  uptime: number;                  // Percentage
  lastIncident?: Date;

  indicators: {
    errorRate: number;
    latencyP50: number;
    latencyP99: number;
    saturation: number;            // Resource usage
  };
}

type CityStatus =
  | "thriving"     // All good, growing
  | "stable"       // Normal operations
  | "strained"     // Under pressure
  | "degraded"     // Partial issues
  | "critical"     // Major problems
  | "offline";     // Down
```

#### Trade Routes (Service Communication)

```typescript
interface TradeRoute {
  id: string;
  source: string;                  // City ID
  target: string;                  // City ID

  // Traffic
  traffic: {
    requestsPerSecond: number;
    bytesPerSecond: number;
    activeConnections: number;
  };

  // Health
  health: {
    successRate: number;
    averageLatency: number;
    errorRate: number;
  };

  // Visual
  style: TradeRouteStyle;
  animation: FlowAnimation;
}

interface TradeRouteStyle {
  width: number;                   // Based on traffic volume
  color: string;                   // Based on health
  pattern: "solid" | "dashed";
  particles: {
    enabled: boolean;
    type: "request" | "data" | "error";
    density: number;
    speed: number;
  };
}

// Color coding
const tradeRouteColors = {
  healthy: "#4CAF50",              // Green
  degraded: "#FF9800",             // Orange
  failing: "#F44336",              // Red
  idle: "#9E9E9E"                  // Gray
};
```

#### Threats (Barbarians/Attacks)

```typescript
interface Threat {
  id: string;
  type: ThreatType;
  severity: "low" | "medium" | "high" | "critical";

  // Location
  target: string;                  // City ID
  position: { x: number; y: number };

  // Details
  description: string;
  count: number;                   // For aggregated threats
  firstSeen: Date;
  lastSeen: Date;

  // Visual
  icon: string;
  animation: "idle" | "attacking" | "retreating";
  sprite: string;
}

type ThreatType =
  | "error"            // Application errors
  | "timeout"          // Request timeouts
  | "rate_limit"       // Rate limit hits
  | "attack"           // Security threat
  | "overload"         // Resource exhaustion
  | "dependency"       // External service issue
  | "data_corruption"; // Data integrity issue

// Threat visuals
const threatSprites = {
  error: { icon: "⚔️", color: "#F44336", label: "Error" },
  timeout: { icon: "⏱️", color: "#FF9800", label: "Timeout" },
  attack: { icon: "🏴‍☠️", color: "#9C27B0", label: "Attack" },
  overload: { icon: "🔥", color: "#FF5722", label: "Overload" }
};
```

#### Weather System (Global Metrics)

```typescript
interface Weather {
  current: WeatherType;
  forecast: WeatherForecast[];

  // Factors that affect weather
  factors: {
    errorRate: number;             // High = storms
    traffic: number;               // High = floods, low = drought
    latency: number;               // High = fog
    threats: number;               // Many = dark clouds
  };

  // Visual effects
  effects: WeatherEffect[];
}

type WeatherType =
  | "clear"            // ☀️ All systems nominal
  | "cloudy"           // ☁️ Minor issues
  | "rain"             // 🌧️ Elevated errors
  | "storm"            // ⛈️ Major issues
  | "fog"              // 🌫️ High latency
  | "drought"          // 🏜️ Low traffic
  | "flood"            // 🌊 Traffic spike
  | "snow";            // ❄️ Frozen/slow

interface WeatherForecast {
  time: Date;
  predicted: WeatherType;
  confidence: number;
  reason: string;
}

// Weather effects on map
const weatherEffects: Record<WeatherType, WeatherEffect> = {
  clear: {
    sky: "gradient-blue",
    particles: ["birds", "butterflies"],
    lighting: "bright"
  },
  storm: {
    sky: "gradient-dark",
    particles: ["rain", "lightning"],
    lighting: "dim",
    sound: "thunder"
  },
  flood: {
    sky: "gradient-gray",
    particles: ["heavy-rain"],
    lighting: "normal",
    overlay: "water-rising"
  }
};
```

#### Population (Users/Traffic)

```typescript
interface Population {
  // Global stats
  total: {
    activeUsers: number;
    requestsPerSecond: number;
    sessionsActive: number;
  };

  // Per-city breakdown
  distribution: Map<string, number>;

  // Segments
  segments: PopulationSegment[];

  // Trends
  trend: {
    direction: "up" | "down" | "stable";
    rate: number;                  // Change per hour
    comparison: {
      vsLastHour: number;
      vsYesterday: number;
      vsLastWeek: number;
    };
  };

  // Geographic (if available)
  geographic?: {
    regions: Map<string, number>;
    topCountries: string[];
  };
}

interface PopulationSegment {
  id: string;
  name: string;
  count: number;
  characteristics: string[];
  color: string;
}
```

#### Happiness (SLO/Performance)

```typescript
interface Happiness {
  overall: number;                 // 0-100

  // Individual factors
  factors: {
    latency: {
      score: number;
      p50: number;
      p99: number;
      target: number;
    };
    availability: {
      score: number;
      current: number;
      target: number;
    };
    errorRate: {
      score: number;
      current: number;
      target: number;
    };
    throughput: {
      score: number;
      current: number;
      capacity: number;
    };
  };

  // Visual indicator
  display: {
    icon: "😊" | "😐" | "😟" | "😢" | "😱";
    color: string;
    description: string;
  };

  // Alerts
  alerts: HappinessAlert[];
}

// Happiness thresholds
const happinessLevels = {
  ecstatic: { min: 95, icon: "😊", color: "#4CAF50" },
  happy: { min: 80, icon: "🙂", color: "#8BC34A" },
  neutral: { min: 60, icon: "😐", color: "#FFC107" },
  unhappy: { min: 40, icon: "😟", color: "#FF9800" },
  miserable: { min: 0, icon: "😢", color: "#F44336" }
};
```

#### Real-time Updates

```typescript
interface RealmUpdates {
  // WebSocket connection for live updates
  subscribe(callback: (update: RealmUpdate) => void): Unsubscribe;

  // Update types
  types: {
    cityHealth: "City health changed";
    tradeRouteTraffic: "Traffic pattern changed";
    threatAppeared: "New threat detected";
    threatResolved: "Threat eliminated";
    weatherChanged: "Weather conditions changed";
    populationShift: "User population changed";
    incidentStarted: "Incident began";
    incidentResolved: "Incident resolved";
  };
}

interface RealmUpdate {
  type: string;
  timestamp: Date;
  data: any;
  affectedCities: string[];
  severity: "info" | "warning" | "critical";
}
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

Research and unlock capabilities through a technology tree system.

### Tech Tree Visualization

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           TECHNOLOGY TREE                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ERA I: FOUNDATIONS                ERA II: EXPANSION                        │
│  ══════════════════                ═══════════════════                      │
│                                                                              │
│  ┌─────────┐                       ┌─────────┐                              │
│  │ ✓ Read  │─────────────────────▶│ ✓ Grep  │                              │
│  │ [DONE]  │                       │ [DONE]  │                              │
│  └────┬────┘                       └────┬────┘                              │
│       │                                 │                                    │
│       ▼                                 ▼                                    │
│  ┌─────────┐     ┌─────────┐      ┌─────────┐      ┌─────────┐             │
│  │ ✓ Edit  │────▶│ ✓ Write │      │ ✓ Glob  │─────▶│ 🔒 Task │             │
│  │ [DONE]  │     │ [DONE]  │      │ [DONE]  │      │ [LOCK]  │             │
│  └────┬────┘     └─────────┘      └─────────┘      └────┬────┘             │
│       │                                                  │                   │
│       ▼                                                  ▼                   │
│  ┌─────────┐                                       ┌─────────┐              │
│  │ ✓ Bash  │                                       │ ? Agent │              │
│  │ [DONE]  │                                       │ [???]   │              │
│  └────┬────┘                                       └─────────┘              │
│       │                                                                      │
│       ├────────────────────┬────────────────────┐                           │
│       ▼                    ▼                    ▼                           │
│  ┌─────────┐          ┌─────────┐          ┌─────────┐                     │
│  │ ✓ Git   │          │ 🔒 npm  │          │ 🔒Docker│                     │
│  │ [DONE]  │          │ [LOCK]  │          │ [LOCK]  │                     │
│  └─────────┘          └─────────┘          └─────────┘                     │
│                                                                              │
│  ERA III: INTEGRATION              ERA IV: MASTERY                          │
│  ════════════════════              ════════════════                         │
│                                                                              │
│  ┌─────────┐      ┌─────────┐      ┌─────────┐      ┌─────────┐           │
│  │🔒GitHub │─────▶│ 🔒 PR   │      │ ? CI/CD │─────▶│ ? Deploy│           │
│  │  MCP    │      │ Review  │      │  [???]  │      │  [???]  │           │
│  └─────────┘      └─────────┘      └─────────┘      └─────────┘           │
│                                                                              │
│  Legend: ✓ Unlocked  🔒 Locked (available)  ? Undiscovered                  │
│                                                                              │
│  Research Points: 1,247 / Next Era: 2,000                                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Tech Tree Data Model

```typescript
interface TechTree {
  id: string;
  name: string;
  description: string;

  // Structure
  eras: Era[];
  nodes: TechNode[];
  connections: TechConnection[];

  // Progress
  researchPoints: number;
  unlockedNodes: string[];
  activeResearch?: string;
}

interface Era {
  id: string;
  name: string;
  order: number;
  color: string;
  requiredPoints: number;          // Points to unlock this era
  nodes: string[];                 // Node IDs in this era
}

interface TechNode {
  id: string;
  name: string;
  description: string;
  icon: string;
  era: string;

  // Type
  type: TechNodeType;
  category: TechCategory;

  // State
  state: TechState;

  // Requirements
  prerequisites: string[];         // Other nodes required
  cost: TechCost;

  // Effects when unlocked
  unlocks: TechUnlock[];

  // Position in tree
  position: { x: number; y: number };
}

type TechNodeType =
  | "tool"           // Built-in Claude tool
  | "mcp"            // MCP server
  | "command"        // Slash command
  | "advisor"        // Advisor unlock
  | "feature"        // Game feature
  | "integration";   // External integration

type TechCategory =
  | "core"           // Essential tools
  | "development"    // Dev tools
  | "collaboration"  // Team features
  | "automation"     // CI/CD, bots
  | "monitoring"     // Observability
  | "security";      // Security tools

type TechState =
  | "unlocked"       // Available to use
  | "locked"         // Prerequisites not met
  | "available"      // Can be researched
  | "researching"    // Currently being unlocked
  | "undiscovered";  // Not yet revealed

interface TechCost {
  researchPoints?: number;
  contextCost?: number;            // One-time context spend
  timeCost?: number;               // Cooldown after unlocking
  prerequisites?: string[];
}

interface TechUnlock {
  type: "tool" | "mcp" | "command" | "advisor" | "feature" | "bonus";
  id: string;
  name: string;
  description: string;
}
```

### Research System

```typescript
interface ResearchSystem {
  // Points
  points: {
    current: number;
    lifetime: number;
    rate: number;                  // Per hour
  };

  // How to earn research points
  sources: ResearchSource[];

  // Active research
  activeResearch?: {
    node: string;
    progress: number;              // 0-100
    estimatedCompletion: Date;
  };

  // Queue
  researchQueue: string[];
}

interface ResearchSource {
  id: string;
  name: string;
  description: string;
  pointsPerAction: number;
  frequency: "once" | "repeatable" | "passive";
}

const researchSources: ResearchSource[] = [
  {
    id: "first_build",
    name: "First Successful Build",
    description: "Complete your first build",
    pointsPerAction: 100,
    frequency: "once"
  },
  {
    id: "test_pass",
    name: "Tests Pass",
    description: "All tests pass after a change",
    pointsPerAction: 10,
    frequency: "repeatable"
  },
  {
    id: "code_review",
    name: "Code Review",
    description: "Complete a code review",
    pointsPerAction: 25,
    frequency: "repeatable"
  },
  {
    id: "bug_fix",
    name: "Bug Fixed",
    description: "Resolve a bug from tests",
    pointsPerAction: 50,
    frequency: "repeatable"
  },
  {
    id: "coverage_increase",
    name: "Coverage Increase",
    description: "Increase test coverage",
    pointsPerAction: 5,            // Per percentage point
    frequency: "repeatable"
  }
];
```

### Tech Tree Nodes

```typescript
const techNodes: TechNode[] = [
  // ERA I: Foundations
  {
    id: "read",
    name: "Read",
    description: "Read files from the codebase",
    icon: "📖",
    era: "foundations",
    type: "tool",
    category: "core",
    state: "unlocked",             // Available by default
    prerequisites: [],
    cost: { researchPoints: 0 },
    unlocks: [{ type: "tool", id: "Read", name: "Read Tool" }]
  },
  {
    id: "edit",
    name: "Edit",
    description: "Make changes to files",
    icon: "✏️",
    era: "foundations",
    type: "tool",
    category: "core",
    prerequisites: ["read"],
    cost: { researchPoints: 50 },
    unlocks: [{ type: "tool", id: "Edit", name: "Edit Tool" }]
  },
  {
    id: "bash",
    name: "Bash",
    description: "Execute shell commands",
    icon: "💻",
    era: "foundations",
    type: "tool",
    category: "core",
    prerequisites: ["edit"],
    cost: { researchPoints: 100 },
    unlocks: [{ type: "tool", id: "Bash", name: "Bash Tool" }]
  },

  // ERA II: Expansion
  {
    id: "git",
    name: "Git",
    description: "Version control operations",
    icon: "🔀",
    era: "expansion",
    type: "tool",
    category: "development",
    prerequisites: ["bash"],
    cost: { researchPoints: 150 },
    unlocks: [
      { type: "feature", id: "git_status", name: "Git Status Panel" },
      { type: "feature", id: "commit_history", name: "Commit History View" }
    ]
  },
  {
    id: "task_agents",
    name: "Task Agents",
    description: "Spawn specialized sub-agents",
    icon: "🤖",
    era: "expansion",
    type: "feature",
    category: "automation",
    prerequisites: ["glob", "grep"],
    cost: { researchPoints: 300 },
    unlocks: [
      { type: "tool", id: "Task", name: "Task Tool" },
      { type: "advisor", id: "architect", name: "The Architect" }
    ]
  },

  // ERA III: Integration
  {
    id: "github_mcp",
    name: "GitHub MCP",
    description: "GitHub API integration",
    icon: "🐙",
    era: "integration",
    type: "mcp",
    category: "collaboration",
    prerequisites: ["git"],
    cost: { researchPoints: 400 },
    unlocks: [
      { type: "mcp", id: "github", name: "GitHub MCP Server" },
      { type: "feature", id: "pr_view", name: "PR Visualization" }
    ]
  },
  {
    id: "pr_review",
    name: "PR Review",
    description: "Automated PR review capabilities",
    icon: "👀",
    era: "integration",
    type: "command",
    category: "collaboration",
    prerequisites: ["github_mcp"],
    cost: { researchPoints: 500 },
    unlocks: [
      { type: "command", id: "/review", name: "Review Command" },
      { type: "advisor", id: "inquisitor", name: "The Inquisitor" }
    ]
  },

  // ERA IV: Mastery
  {
    id: "ci_cd",
    name: "CI/CD Integration",
    description: "Continuous integration and deployment",
    icon: "🔄",
    era: "mastery",
    type: "integration",
    category: "automation",
    prerequisites: ["github_mcp", "bash"],
    cost: { researchPoints: 800 },
    unlocks: [
      { type: "feature", id: "ci_dashboard", name: "CI Dashboard" },
      { type: "feature", id: "auto_fix", name: "Auto-Fix Failed Builds" }
    ]
  },
  {
    id: "deploy",
    name: "Deployment",
    description: "Deploy to production environments",
    icon: "🚀",
    era: "mastery",
    type: "integration",
    category: "automation",
    prerequisites: ["ci_cd"],
    cost: { researchPoints: 1000 },
    unlocks: [
      { type: "feature", id: "deploy_view", name: "Deployment Realm View" },
      { type: "command", id: "/deploy", name: "Deploy Command" }
    ]
  }
];
```

### Discovery System

Some nodes start hidden:

```typescript
interface DiscoverySystem {
  // Conditions that reveal hidden nodes
  triggers: DiscoveryTrigger[];

  // Recently discovered
  recentDiscoveries: Discovery[];
}

interface DiscoveryTrigger {
  nodeId: string;
  conditions: DiscoveryCondition[];
  discoveryMessage: string;
}

type DiscoveryCondition =
  | { type: "use_tool"; tool: string; count: number }
  | { type: "era_reached"; era: string }
  | { type: "node_unlocked"; node: string }
  | { type: "resource_threshold"; resource: string; value: number }
  | { type: "time_played"; hours: number };

// Example triggers
const discoveryTriggers: DiscoveryTrigger[] = [
  {
    nodeId: "task_agents",
    conditions: [
      { type: "use_tool", tool: "Grep", count: 10 },
      { type: "use_tool", tool: "Glob", count: 10 }
    ],
    discoveryMessage: "You've discovered the power of Task Agents!"
  },
  {
    nodeId: "advanced_mcp",
    conditions: [
      { type: "node_unlocked", node: "github_mcp" },
      { type: "era_reached", era: "integration" }
    ],
    discoveryMessage: "New MCP possibilities have been revealed!"
  }
];
```

- **Researched** = Configured and available
- **Locked** = Available but not set up
- **Undiscovered** = User hasn't encountered yet
- **Prerequisites** = Some tools require others first

---

## Missions System

Suggested objectives drive engagement and teach capabilities.

### Mission Data Model

```typescript
interface Mission {
  id: string;
  name: string;
  title: string;                   // Display title
  description: string;             // Flavor text
  technicalDescription: string;    // What it actually does

  // Classification
  type: MissionType;
  category: MissionCategory;
  difficulty: MissionDifficulty;

  // Structure
  objectives: Objective[];
  phases?: MissionPhase[];         // For multi-phase missions

  // Requirements
  prerequisites: MissionPrerequisite[];
  unlockConditions: UnlockCondition[];

  // Rewards
  rewards: MissionReward[];

  // State
  state: MissionState;
  progress: MissionProgress;
  startedAt?: Date;
  completedAt?: Date;

  // Triggers
  triggers: MissionTrigger[];      // What activates this mission

  // UI
  icon: string;
  banner?: string;
  advisorNarration?: AdvisorNarration[];
}

type MissionType =
  | "campaign"        // Story missions
  | "side_quest"      // Opportunistic
  | "daily"           // Reset daily
  | "weekly"          // Reset weekly
  | "endless"         // Ongoing with leaderboards
  | "tutorial"        // Teaching missions
  | "challenge";      // Special difficulty

type MissionCategory =
  | "exploration"     // Understanding codebase
  | "construction"    // Building features
  | "fortification"   // Testing/coverage
  | "optimization"    // Performance
  | "diplomacy"       // Dependencies
  | "conquest"        // Bug fixing
  | "research";       // Learning new tools

type MissionDifficulty =
  | "trivial"         // 1 star
  | "easy"            // 2 stars
  | "medium"          // 3 stars
  | "hard"            // 4 stars
  | "legendary";      // 5 stars

type MissionState =
  | "locked"
  | "available"
  | "active"
  | "completed"
  | "failed"
  | "abandoned";

interface Objective {
  id: string;
  description: string;
  type: ObjectiveType;
  target: ObjectiveTarget;
  progress: number;
  required: number;
  completed: boolean;
  optional: boolean;
}

type ObjectiveType =
  | "create_file"
  | "edit_file"
  | "run_tests"
  | "pass_tests"
  | "increase_coverage"
  | "fix_bug"
  | "review_code"
  | "deploy"
  | "reach_metric"
  | "consult_advisor"
  | "unlock_tech"
  | "custom";

interface MissionReward {
  type: RewardType;
  amount?: number;
  id?: string;
  name: string;
  description: string;
}

type RewardType =
  | "xp"
  | "research_points"
  | "unlock_advisor"
  | "unlock_tech"
  | "unlock_mission"
  | "title"
  | "achievement"
  | "cosmetic";
```

### Mission Types

#### Campaign Missions (Story-driven)

```typescript
const campaignMissions: Mission[] = [
  {
    id: "campaign_01",
    name: "explore_realm",
    title: "Explore the Realm",
    description: "A new kingdom awaits. Survey your lands and understand what you've inherited.",
    type: "campaign",
    category: "exploration",
    difficulty: "trivial",
    objectives: [
      {
        id: "read_readme",
        description: "Read the sacred README",
        type: "read_file",
        target: { pattern: "**/README.md" },
        required: 1
      },
      {
        id: "explore_structure",
        description: "Explore 5 different directories",
        type: "explore",
        target: { type: "directory" },
        required: 5
      },
      {
        id: "find_entry",
        description: "Locate the main entry point",
        type: "find_file",
        target: { pattern: ["**/index.ts", "**/main.ts", "**/app.ts"] },
        required: 1
      }
    ],
    rewards: [
      { type: "xp", amount: 100, name: "Explorer XP" },
      { type: "unlock_advisor", id: "architect", name: "The Architect joins your council" }
    ]
  },
  {
    id: "campaign_02",
    name: "fortify_walls",
    title: "Fortify the Walls",
    description: "A kingdom without tests is a kingdom waiting to fall. Build your defenses.",
    type: "campaign",
    category: "fortification",
    difficulty: "easy",
    prerequisites: [{ type: "mission_complete", missionId: "campaign_01" }],
    objectives: [
      {
        id: "run_tests",
        description: "Run the test suite",
        type: "run_tests",
        required: 1
      },
      {
        id: "check_coverage",
        description: "View coverage report",
        type: "view_coverage",
        required: 1
      },
      {
        id: "add_test",
        description: "Add a new test",
        type: "create_file",
        target: { pattern: "**/*.test.*" },
        required: 1
      },
      {
        id: "increase_coverage",
        description: "Increase coverage by 5%",
        type: "increase_coverage",
        required: 5
      }
    ],
    rewards: [
      { type: "xp", amount: 250, name: "Defender XP" },
      { type: "unlock_advisor", id: "inquisitor", name: "The Inquisitor joins your council" },
      { type: "research_points", amount: 100, name: "Research Points" }
    ]
  }
];
```

#### Side Quests (Opportunistic)

```typescript
interface SideQuestTrigger {
  id: string;
  condition: (state: GameState) => boolean;
  generate: (state: GameState) => Mission;
  cooldown: number;                // Hours before can trigger again
  maxActive: number;               // Max simultaneous of this type
}

const sideQuestTriggers: SideQuestTrigger[] = [
  {
    id: "flaky_test",
    condition: (s) => s.units.some(u => u.stats.lck.current < 50),
    generate: (s) => {
      const flakyTest = s.units.find(u => u.stats.lck.current < 50);
      return {
        id: `flaky_${flakyTest.id}`,
        name: "flaky_fortress",
        title: "The Flaky Fortress",
        description: `The test "${flakyTest.name}" is unreliable. Stabilize it.`,
        type: "side_quest",
        category: "fortification",
        difficulty: "medium",
        objectives: [
          {
            id: "fix_flakiness",
            description: "Make the test reliable (LCK > 80)",
            type: "reach_metric",
            target: { unit: flakyTest.id, stat: "lck", value: 80 },
            required: 1
          }
        ],
        rewards: [
          { type: "xp", amount: 100 },
          { type: "research_points", amount: 50 }
        ]
      };
    },
    cooldown: 24,
    maxActive: 3
  },
  {
    id: "old_todo",
    condition: (s) => s.todos.some(t => t.age > 365),
    generate: (s) => {
      const oldTodo = s.todos.find(t => t.age > 365);
      return {
        id: `todo_${oldTodo.id}`,
        title: "Technical Debt Collection",
        description: `A TODO from ${oldTodo.date.getFullYear()} haunts the codebase.`,
        type: "side_quest",
        objectives: [
          {
            id: "resolve_todo",
            description: "Address the ancient TODO",
            type: "edit_file",
            target: { file: oldTodo.file, line: oldTodo.line },
            required: 1
          }
        ],
        rewards: [
          { type: "xp", amount: 75 },
          { type: "title", id: "debt_collector", name: "Debt Collector" }
        ]
      };
    },
    cooldown: 48,
    maxActive: 5
  }
];
```

#### Endless/Sandbox Missions

```typescript
interface EndlessMission extends Mission {
  scoring: ScoringSystem;
  leaderboard: Leaderboard;
  tiers: EndlessTier[];
}

interface ScoringSystem {
  metric: string;
  direction: "higher" | "lower";
  formula: (state: GameState) => number;
}

const endlessMissions: EndlessMission[] = [
  {
    id: "endless_build_speed",
    title: "Speed Demon",
    description: "How fast can you build?",
    type: "endless",
    scoring: {
      metric: "Build Time",
      direction: "lower",
      formula: (s) => s.lastBuildTime
    },
    tiers: [
      { name: "Bronze", threshold: 60000, reward: { type: "title", id: "speedy" } },
      { name: "Silver", threshold: 30000, reward: { type: "cosmetic", id: "silver_belt" } },
      { name: "Gold", threshold: 15000, reward: { type: "cosmetic", id: "gold_belt" } },
      { name: "Platinum", threshold: 5000, reward: { type: "title", id: "lightning" } }
    ]
  },
  {
    id: "endless_coverage",
    title: "The Perfect Kingdom",
    description: "Achieve maximum test coverage",
    type: "endless",
    scoring: {
      metric: "Coverage",
      direction: "higher",
      formula: (s) => s.coverage
    },
    tiers: [
      { name: "Bronze", threshold: 60, reward: { type: "xp", amount: 100 } },
      { name: "Silver", threshold: 80, reward: { type: "xp", amount: 250 } },
      { name: "Gold", threshold: 95, reward: { type: "title", id: "perfectionist" } },
      { name: "Platinum", threshold: 100, reward: { type: "achievement", id: "100_percent" } }
    ]
  }
];
```

### Mission UI

```
┌─────────────────────────────────────────────────────────────────────┐
│ 📜 ACTIVE MISSION: Ship the Authentication Feature                  │
│    ★★★☆☆ Medium | Campaign                                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  🏛️ Architect: "The kingdom requires secure gates. Implement       │
│     OAuth integration and prove its strength with tests."           │
│                                                                      │
│  ═══════════════════════════════════════════════════                │
│  OBJECTIVES                                      Progress            │
│  ═══════════════════════════════════════════════════                │
│                                                                      │
│  ✓ Create auth module structure                 [████████████] 100% │
│  ✓ Implement OAuth flow                         [████████████] 100% │
│  ○ Add session management                       [████░░░░░░░░]  33% │
│  ○ Write integration tests (2/5)                [████████░░░░]  40% │
│  ○ Pass security review                         [░░░░░░░░░░░░]   0% │
│                                                                      │
│  ───────────────────────────────────────────────────────────────    │
│  OVERALL: [████████████████░░░░░░░░░░░░░░] 55%                      │
│                                                                      │
│  ═══════════════════════════════════════════════════                │
│  REWARDS                                                             │
│  ═══════════════════════════════════════════════════                │
│  • +500 XP                                                          │
│  • 🛡️ Unlock "Security Advisor"                                     │
│  • 🔓 Unlock Tech: "Auth Patterns"                                  │
│                                                                      │
│  ───────────────────────────────────────────────────────────────    │
│  [💡 Get Hint] [⏸️ Pause] [🚫 Abandon]           Est: 2h remaining  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Extensibility Architecture

### Plugin System

The game interface is extensible at multiple levels through a comprehensive plugin API.

### Plugin Data Model

```typescript
interface CodingGamePlugin {
  // Identity
  id: string;
  name: string;
  version: string;
  description: string;
  author: string;

  // Dependencies
  dependencies?: PluginDependency[];
  peerDependencies?: string[];

  // Lifecycle hooks
  hooks: PluginHooks;

  // Extensions
  resources?: Resource[];
  advisors?: Advisor[];
  missions?: MissionTemplate[];
  visualizations?: Visualization[];
  buildingSystems?: BuildingSystem[];
  unitTypes?: UnitType[];
  techNodes?: TechNode[];
  commands?: Command[];
  themes?: Theme[];

  // Settings
  settings?: PluginSetting[];
  defaultConfig?: Record<string, any>;
}

interface PluginDependency {
  id: string;
  version: string;
  optional: boolean;
}

interface PluginHooks {
  // Lifecycle
  onLoad?: () => Promise<void>;
  onUnload?: () => Promise<void>;
  onEnable?: () => Promise<void>;
  onDisable?: () => Promise<void>;

  // Game events
  onGameStateChange?: (state: GameState) => void;
  onBuildComplete?: (result: BuildResult) => void;
  onTestComplete?: (result: TestResult) => void;
  onFileChange?: (file: FileChange) => void;
  onAdvisorConsult?: (advisor: Advisor, query: string) => void;
  onMissionComplete?: (mission: Mission) => void;

  // UI events
  onMapClick?: (tile: MapTile) => void;
  onUnitSelect?: (unit: Unit) => void;
  onBuildingSelect?: (building: Building) => void;

  // Custom events
  onCustomEvent?: (event: CustomEvent) => void;
}

interface PluginSetting {
  id: string;
  name: string;
  description: string;
  type: "boolean" | "string" | "number" | "select" | "color";
  default: any;
  options?: { value: any; label: string }[];
  validate?: (value: any) => boolean;
}
```

### Plugin Registry

```typescript
interface PluginRegistry {
  // Registration
  register(plugin: CodingGamePlugin): void;
  unregister(pluginId: string): void;

  // Discovery
  list(): PluginInfo[];
  get(id: string): CodingGamePlugin | null;
  search(query: string): PluginInfo[];

  // Management
  enable(id: string): Promise<void>;
  disable(id: string): Promise<void>;
  update(id: string): Promise<void>;

  // Installation
  install(source: string): Promise<void>;
  uninstall(id: string): Promise<void>;

  // Events
  on(event: PluginEvent, handler: Function): void;
  off(event: PluginEvent, handler: Function): void;
}

type PluginEvent =
  | "plugin:registered"
  | "plugin:unregistered"
  | "plugin:enabled"
  | "plugin:disabled"
  | "plugin:error";
```

### Example Plugins

```typescript
// Kubernetes Plugin
const k8sPlugin: CodingGamePlugin = {
  id: "kubernetes",
  name: "Kubernetes Realm",
  version: "1.0.0",
  description: "Visualize and manage Kubernetes clusters",
  author: "CodingGame Team",

  resources: [
    {
      id: "pods",
      name: "Pod Capacity",
      icon: "🫛",
      metaphor: "capacity",
      compute: () => k8s.getAvailablePods(),
      max: () => k8s.getPodLimit()
    },
    {
      id: "cpu",
      name: "CPU Quota",
      icon: "🔥",
      metaphor: "mana",
      compute: () => k8s.getCpuRemaining(),
      max: () => k8s.getCpuQuota()
    }
  ],

  advisors: [
    {
      id: "helmsman",
      name: "The Helmsman",
      title: "Master of the Fleet",
      icon: "⚓",
      domain: "kubernetes",
      personality: {
        formality: "formal",
        metaphorStyle: "nautical"
      },
      voiceStyle: {
        greeting: ["The fleet awaits your orders, Captain."],
        concern: ["Rough seas ahead - pods are failing."]
      }
    }
  ],

  visualizations: [
    {
      id: "cluster_map",
      name: "Cluster Map",
      type: "map_layer",
      render: (ctx) => renderClusterMap(ctx)
    }
  ],

  buildingSystems: [
    {
      id: "deployment",
      name: "Deployments",
      icon: "🚀",
      detector: (path) => isKubernetesDeployment(path),
      parser: parseDeployment,
      toBuilding: deploymentToBuilding
    }
  ],

  hooks: {
    onLoad: async () => {
      await k8s.connect();
    },
    onBuildComplete: (result) => {
      if (result.target.includes("k8s")) {
        k8s.triggerRollout(result.target);
      }
    }
  }
};

// GitHub Plugin
const githubPlugin: CodingGamePlugin = {
  id: "github",
  name: "GitHub Integration",
  version: "1.0.0",
  description: "Full GitHub integration with PR visualization",

  resources: [
    {
      id: "github_api",
      name: "GitHub API",
      icon: "🐙",
      metaphor: "mana",
      compute: () => github.getRateLimit().remaining,
      max: 5000
    }
  ],

  advisors: [
    {
      id: "diplomat",
      name: "The Diplomat",
      domain: "dependencies",
      // ... full advisor config
    }
  ],

  techNodes: [
    {
      id: "github_mcp",
      name: "GitHub MCP",
      era: "integration",
      prerequisites: ["git"],
      unlocks: [
        { type: "mcp", id: "github" },
        { type: "feature", id: "pr_visualization" }
      ]
    }
  ],

  commands: [
    {
      id: "/pr",
      name: "Pull Request",
      description: "Create or view pull requests",
      execute: async (args) => await handlePRCommand(args)
    }
  ]
};

// Metrics/Observability Plugin
const metricsPlugin: CodingGamePlugin = {
  id: "metrics",
  name: "Metrics & Observability",
  version: "1.0.0",

  resources: [
    {
      id: "error_budget",
      name: "Error Budget",
      icon: "💔",
      metaphor: "currency",
      compute: () => metrics.getErrorBudgetRemaining()
    },
    {
      id: "p99_latency",
      name: "P99 Latency",
      icon: "⏱️",
      metaphor: "health",
      compute: () => metrics.getP99Latency()
    }
  ],

  visualizations: [
    {
      id: "production_realm",
      name: "Production Realm",
      type: "map_view",
      render: renderProductionRealm
    }
  ]
};
```

### Theme System

```typescript
interface Theme {
  id: string;
  name: string;
  description: string;

  // Colors
  colors: {
    primary: string;
    secondary: string;
    background: string;
    surface: string;
    text: string;
    textSecondary: string;
    success: string;
    warning: string;
    error: string;
    info: string;
  };

  // Map styles
  map: {
    tileColors: Record<TerrainType, string>;
    fogColor: string;
    gridColor: string;
    selectionColor: string;
  };

  // Unit sprites
  sprites: {
    units: Record<UnitClass, string>;
    buildings: Record<BuildingLevel, string>;
    advisors: Record<string, string>;
  };

  // Sounds
  sounds?: {
    buildComplete: string;
    testPass: string;
    testFail: string;
    levelUp: string;
    notification: string;
  };
}
```

---

## UI Layout & Interaction

### Main Interface Layout

```
┌────────────────────────────────────────────────────────────────────────────────┐
│ RESOURCE BAR                                                                    │
│ 🪙 Context: 42k │ 😊 Coverage: 73% │ ⚡ CI: 2/4 │ 🐙 API: 4.2k │ 💔 Budget: 87% │
├────────────────────────────────────────────────────────────────────────────────┤
│ ┌──────────────┐ ┌──────────────────────────────────────────┐ ┌──────────────┐ │
│ │   ADVISORS   │ │                                          │ │   MISSION    │ │
│ │              │ │                                          │ │              │ │
│ │ ┌──────────┐ │ │                                          │ │ 📜 Ship Auth │ │
│ │ │ 🏛️ Arch  │ │ │                                          │ │ ─────────── │ │
│ │ │ ●●●○○    │ │ │             MAIN MAP VIEW                │ │ ✓ Structure │ │
│ │ └──────────┘ │ │                                          │ │ ✓ OAuth     │ │
│ │ ┌──────────┐ │ │     Codebase / Belts / Deployment        │ │ ○ Session   │ │
│ │ │ 🛡️ Sent  │ │ │                                          │ │ ○ Tests 2/5 │ │
│ │ │ ●●○○○    │ │ │                                          │ │             │ │
│ │ └──────────┘ │ │                                          │ │ ────────── │ │
│ │ ┌──────────┐ │ │                                          │ │ 55% ████░░ │ │
│ │ │ 📚 Chron │ │ │                                          │ │             │ │
│ │ │ ●○○○○    │ │ │                                          │ │ [Details]  │ │
│ │ └──────────┘ │ │                                          │ └──────────────┘ │
│ │ ┌──────────┐ │ │                                          │ ┌──────────────┐ │
│ │ │ ⚖️ Quart │ │ │                                          │ │  MINIMAP     │ │
│ │ │ ●●●●○    │ │ │                                          │ │ ┌──────────┐ │ │
│ │ └──────────┘ │ │                                          │ │ │▪▪  ▫▫ ▪▪ │ │ │
│ │              │ │                                          │ │ │ ┌──┐  ▪▪ │ │ │
│ │ [Council]    │ │                                          │ │ │ └──┘     │ │ │
│ └──────────────┘ │                                          │ │ └──────────┘ │ │
│                  └──────────────────────────────────────────┘ └──────────────┘ │
├────────────────────────────────────────────────────────────────────────────────┤
│ NAVIGATION                                                                      │
│ [🗺️ Map] [🏛️ Buildings] [⚔️ Units] [🔬 Tech Tree] [📜 Missions]  🔍    ⚙️     │
├────────────────────────────────────────────────────────────────────────────────┤
│ CLAUDE PANEL                                                                    │
│ ┌────────────────────────────────────────────────────────────────────────────┐ │
│ │ > Claude: I've analyzed the auth module. The OAuth flow looks good, but   │ │
│ │   I noticed the session management is missing token refresh logic.        │ │
│ │ > Build complete: //src/auth:lib ✓ (2.3s)                                │ │
│ │ > Test: auth_test passed (14/14 assertions)                              │ │
│ │                                                                           │ │
│ │ ▌ Type a message or command...                                    [Send] │ │
│ └────────────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────────┘
```

### Keyboard Shortcuts

```typescript
const keyboardShortcuts: Record<string, ShortcutAction> = {
  // Navigation
  "1": { action: "switch_view", view: "map" },
  "2": { action: "switch_view", view: "buildings" },
  "3": { action: "switch_view", view: "units" },
  "4": { action: "switch_view", view: "tech_tree" },
  "5": { action: "switch_view", view: "missions" },

  // Map controls
  "w": { action: "pan", direction: "up" },
  "a": { action: "pan", direction: "left" },
  "s": { action: "pan", direction: "down" },
  "d": { action: "pan", direction: "right" },
  "+": { action: "zoom", direction: "in" },
  "-": { action: "zoom", direction: "out" },
  "h": { action: "go_home" },
  "b": { action: "go_back" },

  // Quick actions
  "g": { action: "goto_file" },
  "/": { action: "search" },
  "f": { action: "find_in_map" },
  "space": { action: "pause_resume" },
  "escape": { action: "deselect" },

  // Advisors
  "ctrl+1": { action: "consult_advisor", advisor: "architect" },
  "ctrl+2": { action: "consult_advisor", advisor: "sentinel" },
  "ctrl+3": { action: "consult_advisor", advisor: "chronicler" },
  "ctrl+4": { action: "consult_advisor", advisor: "quartermaster" },

  // Building/Units
  "r": { action: "run_selected" },
  "t": { action: "test_selected" },
  "i": { action: "inspect_selected" },

  // Claude
  "enter": { action: "focus_input" },
  "ctrl+enter": { action: "send_message" }
};
```

### Responsive Layouts

```typescript
interface ResponsiveLayout {
  breakpoints: {
    mobile: 0;
    tablet: 768;
    desktop: 1024;
    wide: 1440;
  };

  layouts: {
    mobile: {
      advisors: "collapsed",
      mission: "collapsed",
      minimap: "hidden",
      claudePanel: "bottom_sheet"
    };
    tablet: {
      advisors: "sidebar_left",
      mission: "collapsed",
      minimap: "overlay",
      claudePanel: "bottom_fixed"
    };
    desktop: {
      advisors: "sidebar_left",
      mission: "sidebar_right",
      minimap: "sidebar_right",
      claudePanel: "bottom_fixed"
    };
    wide: {
      advisors: "sidebar_left_expanded",
      mission: "sidebar_right",
      minimap: "sidebar_right",
      claudePanel: "bottom_expanded"
    };
  };
}
```

---

## Data Model & Persistence

### Game State

```typescript
interface GameState {
  // Identity
  id: string;
  projectPath: string;
  createdAt: Date;
  lastSavedAt: Date;

  // Map state
  map: {
    tiles: MapTile[];
    viewState: ViewState;
    fogOfWar: FogState;
  };

  // Entities
  buildings: Building[];
  units: Unit[];
  belts: Belt[];

  // Resources
  resources: ResourceState[];

  // Progress
  techTree: TechTreeState;
  missions: MissionState[];
  achievements: Achievement[];

  // Advisors
  advisors: AdvisorState[];

  // Statistics
  stats: GameStats;

  // Settings
  settings: GameSettings;
}

interface GameStats {
  totalPlayTime: number;
  totalBuilds: number;
  totalTests: number;
  totalBugsFixes: number;
  filesExplored: number;
  linesWritten: number;
  coverageHighWatermark: number;
}
```

### Persistence Layer

```typescript
interface PersistenceService {
  // Save/Load
  save(state: GameState): Promise<void>;
  load(projectPath: string): Promise<GameState | null>;

  // Auto-save
  enableAutoSave(interval: number): void;
  disableAutoSave(): void;

  // Export/Import
  export(format: "json" | "binary"): Promise<Blob>;
  import(data: Blob): Promise<GameState>;

  // History
  getHistory(): SavePoint[];
  restoreFromHistory(savePointId: string): Promise<GameState>;
}

// Storage locations
interface StorageConfig {
  // Local storage for quick access
  localStorage: {
    key: "codinggame_state";
    maxSize: 5 * 1024 * 1024;  // 5MB
  };

  // File system for full saves
  fileSystem: {
    path: ".codinggame/saves/";
    format: "json.gz";
  };

  // Cloud sync (optional)
  cloudSync?: {
    provider: "github_gist" | "custom";
    syncInterval: number;
  };
}
```

---

## Claude Code Integration

### Integration Architecture

```typescript
interface ClaudeCodeIntegration {
  // Tool interception
  tools: ToolInterceptor;

  // Session management
  session: SessionManager;

  // Event streaming
  events: EventStream;

  // State synchronization
  sync: StateSynchronizer;
}

interface ToolInterceptor {
  // Intercept and track tool calls
  onToolCall(tool: string, params: any): void;
  onToolResult(tool: string, result: any): void;

  // Map tools to game actions
  toolMappings: {
    Read: (params) => revealTile(params.file_path);
    Edit: (params) => updateBuilding(params.file_path);
    Bash: (params) => trackCommand(params.command);
    Glob: (params) => explorePath(params.pattern);
    Grep: (params) => highlightMatches(params.results);
    Task: (params) => spawnAgent(params.subagent_type);
  };
}

interface SessionManager {
  // Track context usage
  contextUsed: number;
  contextMax: number;

  // Conversation tracking
  messages: Message[];
  turns: number;

  // Cost tracking
  tokensIn: number;
  tokensOut: number;
  estimatedCost: number;
}

interface EventStream {
  // Subscribe to Claude events
  subscribe(event: ClaudeEvent, handler: Function): void;

  // Event types
  events: [
    "message_start",
    "message_end",
    "tool_use_start",
    "tool_use_end",
    "file_read",
    "file_write",
    "command_run",
    "error"
  ];
}
```

### Tool-to-Game Mapping

```typescript
const toolGameMappings: Record<string, GameAction[]> = {
  Read: [
    { action: "reveal_tile", params: (p) => ({ path: p.file_path }) },
    { action: "gain_xp", params: () => ({ amount: 5, source: "exploration" }) },
    { action: "update_fog", params: (p) => ({ path: p.file_path, state: "explored" }) }
  ],

  Edit: [
    { action: "update_tile", params: (p) => ({ path: p.file_path }) },
    { action: "mark_changed", params: (p) => ({ path: p.file_path }) },
    { action: "recalculate_belts", params: (p) => ({ affected: p.file_path }) }
  ],

  Bash: [
    {
      condition: (p) => p.command.startsWith("git"),
      action: "git_operation",
      params: (p) => parseGitCommand(p.command)
    },
    {
      condition: (p) => isBuildCommand(p.command),
      action: "build_triggered",
      params: (p) => parseBuildCommand(p.command)
    },
    {
      condition: (p) => isTestCommand(p.command),
      action: "test_triggered",
      params: (p) => parseTestCommand(p.command)
    }
  ],

  Task: [
    { action: "spawn_agent", params: (p) => ({ type: p.subagent_type }) },
    { action: "advisor_activity", params: (p) => ({ advisor: mapAgentToAdvisor(p.subagent_type) }) }
  ]
};
```

### Advisor Integration with Claude

```typescript
interface AdvisorClaudeIntegration {
  // Advisor consultations spawn specialized prompts
  consult(advisorId: string, topic?: string): Promise<ConsultationResult> {
    const advisor = getAdvisor(advisorId);
    const systemPrompt = generateAdvisorPrompt(advisor);

    return claude.query({
      system: systemPrompt,
      user: topic || "What is your current assessment?",
      tools: advisor.tools,
      personality: advisor.personality
    });
  }

  // Generate advisor-appropriate system prompt
  generateAdvisorPrompt(advisor: Advisor): string {
    return `You are ${advisor.name}, ${advisor.title}.

Your domain of expertise is: ${advisor.domain}
Your personality: ${JSON.stringify(advisor.personality)}

Speak in character using these voice patterns:
- Greetings: ${advisor.voiceStyle.greeting.join(", ")}
- Concerns: ${advisor.voiceStyle.concern.join(", ")}
- Praise: ${advisor.voiceStyle.praise.join(", ")}

Focus your analysis on: ${advisor.expertise.join(", ")}`;
  }
}
```

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

### Phase 1: Core Framework
- [ ] Map visualization engine
- [ ] Tile and fog of war system
- [ ] Basic resource tracking (context)
- [ ] Claude tool interception

### Phase 2: Buildings & Units
- [ ] Build system adapters (npm, Bazel, cargo)
- [ ] Unit stat calculation from build/test results
- [ ] Building health and production queue

### Phase 3: Advisors
- [ ] Advisor prompt engineering
- [ ] Insight generation system
- [ ] Advisor consultation UI

### Phase 4: Visualization
- [ ] Belt/dependency visualization
- [ ] Animated flow rendering
- [ ] Problem detection highlighting

### Phase 5: Gamification
- [ ] Tech tree implementation
- [ ] Mission system
- [ ] XP and leveling

### Phase 6: Production Realm
- [ ] Deployment visualization
- [ ] Service health monitoring
- [ ] Weather/metrics system

### Phase 7: Polish
- [ ] Plugin system
- [ ] Theme support
- [ ] Sound design
- [ ] Achievements

---

*"The code is dark and full of terrors... but we shall bring the light of understanding."*
— The Chronicler
