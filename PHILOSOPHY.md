# CodingGame Philosophy

## Core Principle: Game Interface to Coding, Not Game About Coding

This is not a gamification layer that adds fake rewards to coding. This is a **game interface** - using game UI patterns and metaphors to visualize and interact with real coding activities.

## Everything Must Be Real

- **No fake bonuses**: There is no "+10% build speed" upgrade. The build takes however long it takes.
- **No fake stats**: Units don't have HP/ATK/DEF. They have real metrics - execution time, pass rate, coverage.
- **No fake progression**: You don't "unlock" tools. The tech tree shows what you've actually added to your project.
- **No fake resources**: Context tokens are real. API costs are real. Build times are real.

## Descriptive, Not Prescriptive

The interface **describes** what exists in your project, it doesn't **prescribe** a progression path.

- **Tech Tree**: Shows tools, MCPs, and commands you've added - clustered by domain. Not a skill tree to unlock.
- **Advisors**: Real subagents you've written and configured. Not pre-defined characters that unlock.
- **Buildings**: Real build targets with real metrics. Not RPG structures with levels.

## The Metaphors

### Buildings = Build Targets
A Bazel package, npm script, or Makefile target. Shows real build times, dependencies, success history.

### Units = Tests
Tests "fight" to pass. This is PvE (player vs environment) - your tests battling against bugs and regressions. Shows real pass/fail status, execution time, flakiness.

### Belts = Data Flow & Debugging
Factorio-style belts show dependency flow AND afford visual debugging - seeing inputs flow into a function, the operations performed, and outputs produced.

### Advisors = Subagents
Real Claude subagents that you write and deploy from your coding harness. They execute actual analysis tasks.

### Fog of War = Context/Knowledge Boundary
Areas Claude hasn't analyzed are fogged. Multi-agent future: each agent has its own fog representing its context window.

### Tech Tree = Capability Inventory
A visual clustering of what tools, MCPs, slash commands, and integrations you have. Descriptive, not a progression system.

## Interface Principles

### Keyboard-First
Everything accessible via keyboard. No mouse required.
- Vim-style or emacs-style bindings
- Navigation mode vs entry mode
- Focus areas with single keys

### Central Interaction: The Prompt
- Focus the prompt window
- Type your request
- Press Enter to "end turn"
- Watch visualization of the coding process

### Future: Multi-Agent Orchestration
- Multiple agents working in parallel
- Player orchestrates through the interface
- Each agent's context is its own "fog of war"

## Target Platform

For now: Claude Code harness with Claude 4.5 Opus or Sonnet.

Future: Game start screen to select harness and model.

## Target Languages for Visual Debugging (Belts)

Initial targets for showing function inputs/operations/outputs:
- Python
- TypeScript
- Go

Expand to others based on debugger/introspection support.
