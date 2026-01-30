# CodingGame Mobile Implementation Plan

## Executive Summary

This document outlines a comprehensive plan for creating a mobile version of CodingGame, the strategy game interface for Claude Code. The mobile version will transform the keyboard-first desktop experience into a touch-optimized interface while maintaining the core visualization metaphors and real-time Claude integration.

**Key Challenge**: Claude Code runs as a subprocess on the host machine, requiring a connection strategy for mobile devices that cannot run the CLI directly.

---

## Part 1: Platform Analysis

### 1.1 Ebitengine Mobile Capabilities

Ebitengine (v2.9.7, already in use) has mature mobile support:

**Android Support**:
- Uses gomobile for building APKs
- OpenGL ES for rendering
- Full touch input support via `ebiten.TouchIDs()` and `ebiten.TouchPosition()`
- Supports Android 5.0+ (API level 21+)

**iOS Support**:
- Uses gomobile for building iOS apps
- Metal or OpenGL ES rendering
- Full touch input support
- Supports iOS 12.0+

**Mobile-specific APIs available**:
```go
// Touch input (not currently used in CodingGame)
ebiten.TouchIDs() []ebiten.TouchID
ebiten.TouchPosition(id ebiten.TouchID) (int, int)

// Virtual keyboard
ebiten.SetVirtualKeyboardVisible(visible bool)

// Screen orientation
ebiten.SetScreenClearedEveryFrame(bool)
ebiten.DeviceScaleFactor() float64
```

The project already includes `github.com/ebitengine/gomobile` as a dependency, indicating mobile builds were anticipated.

### 1.2 Current Architecture Assessment

**Components that can be reused directly**:
- `/internal/mapview/` - Core tile visualization (needs touch input)
- `/internal/tile/` - Tile data structures (no changes needed)
- `/internal/connection/` - Dependency graph (no changes needed)
- `/internal/belt/` - Belt rendering (no changes needed)
- `/internal/harness/` - Harness abstraction (needs network transport)
- `/internal/resources/` - Resource tracking (no changes needed)
- `/internal/ui/scene.go` - Scene management (no changes needed)

**Components requiring significant adaptation**:
- `/internal/input/` - Keyboard-centric, needs touch layer
- `/internal/ui/prompt.go` - Keyboard input, needs virtual keyboard
- `/internal/ui/startscreen.go` - Menu navigation, needs touch
- `/internal/game/scene.go` - Input handling, needs touch support

**Components requiring redesign for mobile**:
- Layout system (currently hardcoded 1280x720)
- Information density (screen space optimization)
- Connection to Claude Code (subprocess model does not work on mobile)

---

## Part 2: Input System Redesign

### 2.1 Touch Input Architecture

The existing `InputSource` interface in `/internal/input/source.go` provides an excellent abstraction point. We need to extend it for touch support.

**Proposed Interface Extension**:
```go
// TouchInputSource extends InputSource with touch capabilities
type TouchInputSource interface {
    InputSource

    // Touch methods
    TouchIDs() []TouchID
    TouchPosition(id TouchID) (x, y int)
    TouchPhase(id TouchID) TouchPhase // began, moved, ended, cancelled

    // Gesture recognition
    IsPinching() bool
    PinchScale() float64
    IsPanning() bool
    PanDelta() (dx, dy float64)
    IsDoubleTap() bool
    IsTapping() bool
    LongPressPosition() (x, y int, active bool)
}

type TouchPhase int
const (
    TouchPhaseBegan TouchPhase = iota
    TouchPhaseMoved
    TouchPhaseEnded
    TouchPhaseCancelled
)
```

### 2.2 Gesture Mapping

Replace keyboard navigation with intuitive touch gestures:

| Desktop Action | Touch Gesture | Implementation |
|----------------|---------------|----------------|
| hjkl/arrows (pan) | One-finger drag | Track touch movement delta |
| +/- (zoom) | Two-finger pinch | Track distance between touches |
| Tab (focus cycle) | Swipe left/right on panel edge | Edge detection + swipe |
| Enter (prompt) | Tap prompt area | Region detection |
| Escape (cancel) | Swipe down on prompt | Gesture recognition |
| i (insert mode) | Tap input field | Auto-transition to insert |
| Select tile | Single tap | Touch position to tile |
| Tile details | Long press | Hold detection (500ms) |
| Double-click | Double tap | Time-based detection |

### 2.3 Input Mode Simplification

Mobile should simplify the vim-style modes:

**Desktop Modes**:
- Normal (navigation)
- Insert (text input)
- Visual (multi-select)

**Mobile Modes**:
- Navigation (default) - panning, zooming, tapping tiles
- Input (when prompt focused) - virtual keyboard active
- Selection (long-press initiated) - multi-tile selection

The mode indicator can be smaller or hidden on mobile, with mode being implicit from context.

### 2.4 Touch Input Implementation Structure

```
internal/input/
    source.go          # Existing InputSource interface
    source_desktop.go  # EbitenInputSource (current)
    source_mobile.go   # MobileInputSource (new)
    touch.go           # Touch event types and gesture recognition
    gestures.go        # Gesture recognizers (pinch, pan, tap, etc.)
```

---

## Part 3: UI/UX Adaptations

### 3.1 Screen Layout Strategy

**Current Layout** (1280x720 desktop):
```
+----------------------------------------------------------+
|  Resources: [ctx: 45k/200k] [cost: $0.12] [build: ok]    |
+----------+--------------------------------+---------------+
| ADVISORS |          MAP VIEW              |   RESPONSE    |
|          |                                |               |
| [Refac]  |    (directory/dataflow)        |   Claude's    |
| [Secur]  |                                |   output      |
| [Tests]  |                                |               |
+----------+--------------------------------+---------------+
|  > Enter your prompt here...                      [END]  |
+----------------------------------------------------------+
```

**Mobile Layout** (portrait, ~360x800):
```
+------------------------------------+
|  [ctx: 45k] [ok] [$0.12]    [...]  |
+------------------------------------+
|                                    |
|                                    |
|            MAP VIEW                |
|       (full width, ~60% height)    |
|                                    |
|                                    |
+------------------------------------+
|  RESPONSE (collapsible, 20%)       |
|  [Claude's last response...]       |
+------------------------------------+
|  > Tap to chat...           [send] |
+------------------------------------+
|  [Map] [Bld] [Unt] [Adv] [Mis]    |  <- Tab bar
+------------------------------------+
```

### 3.2 Responsive Layout System

Create a layout system that adapts to screen dimensions:

```go
// internal/ui/layout.go
type LayoutMode int
const (
    LayoutDesktop LayoutMode = iota  // width >= 1200
    LayoutTablet                      // 600 <= width < 1200
    LayoutPhone                       // width < 600
)

type Layout struct {
    Mode         LayoutMode
    Width        int
    Height       int
    Scale        float64  // Device pixel ratio

    // Calculated regions
    ResourceBar  Rect
    MapView      Rect
    ResponseArea Rect
    PromptBar    Rect
    TabBar       Rect  // Mobile only
}

func NewLayout(width, height int, scale float64) *Layout
func (l *Layout) Update(width, height int)
```

### 3.3 Information Density Optimization

**Tile Rendering for Mobile**:
- Larger touch targets (minimum 44x44 points as per iOS HIG)
- Simplified labels at lower zoom levels
- Icon-based indicators instead of text where possible
- Progressive disclosure: tap for details

**Resource Bar**:
- Abbreviated labels: "ctx: 45k" not "Context: 45,000 tokens"
- Icons for status indicators
- Overflow into dropdown menu (...) for less critical info

**Prompt Panel**:
- Single-line input with expand option
- Send button visible (no keyboard shortcuts)
- Response area collapses when not relevant

### 3.4 Mobile Navigation Pattern

Replace keyboard-based panel navigation with:

1. **Bottom Tab Bar**: Quick switch between main views
   - Map (primary)
   - Buildings
   - Units/Tests
   - Advisors
   - Missions

2. **Swipe Gestures**:
   - Swipe up on prompt: Expand response history
   - Swipe down on prompt: Minimize
   - Swipe right on tile: Quick actions menu

3. **Floating Action Button (FAB)**:
   - Primary action: Open prompt input
   - Secondary actions: Toggle map view, center on project

---

## Part 4: Connection Strategy

### 4.1 The Core Challenge

CodingGame currently spawns `claude --output-format json` as a subprocess. This works on desktop where the CLI is installed. Mobile devices cannot:
- Install the Claude CLI
- Spawn subprocesses in the same way
- Access local project directories

### 4.2 Proposed Architecture: Remote Relay Server

```
+-------------------+        WebSocket        +-------------------+
|   Mobile App      | <--------------------> |   Relay Server    |
|   (CodingGame)    |                         |   (Desktop/Cloud) |
+-------------------+                         +-------------------+
                                                       |
                                                       | subprocess
                                                       v
                                              +-------------------+
                                              |   Claude CLI      |
                                              |   (Local Project) |
                                              +-------------------+
```

### 4.3 Relay Server Design

The relay server runs on the development machine (where the project lives) and provides:

1. **WebSocket API** for real-time event streaming
2. **Project file serving** for map visualization
3. **Claude CLI management** (start, stop, send prompts)
4. **Event forwarding** (file reads, writes, builds, tests)

**Server Implementation** (`cmd/relay/`):
```go
type RelayServer struct {
    projectPath string
    harness     harness.Harness
    clients     map[*websocket.Conn]bool
}

// WebSocket message types
type Message struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

// Messages: project_info, prompt, event, file_list, etc.
```

### 4.4 Mobile Harness Implementation

Create a network-based harness that implements the existing `harness.Harness` interface:

```go
// internal/harness/remote/remote.go
type RemoteHarness struct {
    *harness.BaseHarness
    serverURL string
    conn      *websocket.Conn
    // ...
}

func (r *RemoteHarness) Start(ctx context.Context, config harness.Config) error
func (r *RemoteHarness) SendPrompt(prompt string) error
func (r *RemoteHarness) Stop() error
```

This allows the mobile app to use the same game logic, with the harness transparently communicating over the network.

### 4.5 Connection Discovery

Options for the mobile app to find the relay server:

1. **Manual Entry**: User enters server IP/port
2. **QR Code**: Desktop displays QR code with connection URL
3. **mDNS/Bonjour**: Automatic discovery on local network
4. **Cloud Relay**: Optional cloud service for remote access

Recommended initial approach: QR code for simplicity and security.

---

## Part 5: Build System Integration

### 5.1 Mobile Build Pipeline

**Android Build**:
```bash
# Using gomobile
gomobile build -target=android -o codinggame.apk ./cmd/mobile/

# Or with Bazel (requires android_ndk_repository)
bazel build //:codinggame_android
```

**iOS Build**:
```bash
# Using gomobile
gomobile build -target=ios -o CodingGame.app ./cmd/mobile/

# Requires Xcode and Apple developer account for device deployment
```

### 5.2 Bazel Configuration for Mobile

```python
# BUILD.bazel additions
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

# Mobile entry point
go_library(
    name = "mobile_lib",
    srcs = ["cmd/mobile/main.go"],
    importpath = "github.com/tedks/CodingGame/cmd/mobile",
    deps = [
        "//internal/game",
        "//internal/input:mobile",
        "//internal/harness/remote",
        # ... other deps
    ],
)

# Android target (requires android rules)
# android_binary(
#     name = "codinggame_android",
#     ...
# )
```

### 5.3 Mobile Entry Point

```go
// cmd/mobile/main.go
package main

import (
    "github.com/hajimehoshi/ebiten/v2/mobile"
    "github.com/tedks/CodingGame/internal/game"
)

func main() {
    mobile.SetGame(game.NewMobileApp())
}
```

### 5.4 Build Considerations

1. **Asset bundling**: Embed fonts/sprites in binary or load from assets
2. **Code signing**: Required for iOS, recommended for Android
3. **Permissions**: Network access for relay communication
4. **Screen sizes**: Support various device dimensions

---

## Part 6: Feature Prioritization

### 6.1 Mobile-First Features

| Priority | Feature | Reason |
|----------|---------|--------|
| P0 | Touch navigation (pan/zoom) | Core interaction |
| P0 | Tile visualization | Core value proposition |
| P0 | Fog of war | Core metaphor |
| P0 | Prompt input | Essential interaction |
| P0 | Remote connection | Required for Claude |
| P1 | Response streaming | Real-time feedback |
| P1 | Resource bar | Project state awareness |
| P1 | View switching (tab bar) | Access all views |
| P2 | Advisors panel | Subagent interaction |
| P2 | Tile selection/details | Deeper exploration |
| P3 | Dataflow view | Complex visualization |
| P3 | Visual debugging | Advanced feature |

### 6.2 Desktop-Only Features (Not for Mobile)

| Feature | Reason |
|---------|--------|
| Vim-style keybindings | No keyboard on mobile |
| Multi-agent orchestration | Complex, desktop power-user feature |
| Plugin system | Desktop extensibility |
| Multiple windows | Mobile is single-window |
| File editing | Mobile is for monitoring/light interaction |

### 6.3 Mobile-Enhanced Features

| Feature | Mobile Enhancement |
|---------|-------------------|
| Notifications | Native push notifications for events |
| Haptic feedback | Vibration on important events |
| Dark mode | System theme integration |
| Background updates | Keep fog of war current |

---

## Part 7: Implementation Phases - Detailed Epics

This section breaks down each phase into specific epics with granular, actionable tasks.

---

### Phase M1: Foundation (4-6 weeks)

**Goal**: Mobile app that can display a project map with touch navigation.

---

#### Epic M1.1: Touch Input Types and Interfaces

**Goal**: Define core touch input abstractions that integrate with existing InputSource.

**Dependencies**: None (foundational)

**Files to create/modify**:
- `internal/input/touch.go` (new)
- `internal/input/source.go` (modify)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M1.1.1 | Define TouchID type | Create `type TouchID int` with constants for multi-touch tracking | 1h |
| M1.1.2 | Define TouchPhase enum | Create `TouchPhase` with Began/Moved/Ended/Cancelled states | 1h |
| M1.1.3 | Define TouchEvent struct | Create struct with ID, Phase, X, Y, Timestamp fields | 2h |
| M1.1.4 | Define TouchInputSource interface | Extend InputSource with TouchIDs(), TouchPosition(), TouchPhase() | 2h |
| M1.1.5 | Add touch methods to InputSource | Add optional touch methods with no-op defaults for desktop | 2h |
| M1.1.6 | Write unit tests for touch types | Test TouchPhase transitions, TouchEvent creation | 2h |
| M1.1.7 | Document touch API in code comments | Add godoc for all exported types and methods | 1h |

**Acceptance Criteria**:
- [ ] TouchInputSource interface compiles and is documented
- [ ] Existing InputSource implementations still work (no breaking changes)
- [ ] Unit tests pass for all touch types

---

#### Epic M1.2: Gesture Recognition System

**Goal**: Implement gesture recognizers for tap, pan, pinch, and long-press.

**Dependencies**: M1.1 (Touch Input Types)

**Files to create/modify**:
- `internal/input/gestures.go` (new)
- `internal/input/gestures_test.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M1.2.1 | Create GestureRecognizer interface | Define interface with Update(touches), IsActive(), Reset() | 2h |
| M1.2.2 | Implement TapRecognizer | Detect single tap within 200ms, max 10px movement | 3h |
| M1.2.3 | Implement DoubleTapRecognizer | Detect two taps within 300ms, max 20px apart | 3h |
| M1.2.4 | Implement LongPressRecognizer | Detect hold for 500ms with minimal movement | 3h |
| M1.2.5 | Implement PanRecognizer | Track single-finger drag, report delta movement | 4h |
| M1.2.6 | Implement PinchRecognizer | Track two-finger distance changes, report scale factor | 4h |
| M1.2.7 | Create GestureManager | Coordinate multiple recognizers, handle conflicts | 4h |
| M1.2.8 | Add gesture conflict resolution | Tap vs LongPress, Pan vs Pinch priority rules | 3h |
| M1.2.9 | Write comprehensive gesture tests | Test each recognizer with simulated touch sequences | 4h |
| M1.2.10 | Add gesture debugging overlay | Optional visual feedback showing recognized gestures | 2h |

**Acceptance Criteria**:
- [ ] All five gesture types recognized correctly
- [ ] Gesture conflicts resolved predictably
- [ ] Tests cover edge cases (interrupted gestures, rapid sequences)

---

#### Epic M1.3: Mobile Input Source Implementation

**Goal**: Create MobileInputSource that wraps Ebitengine touch APIs.

**Dependencies**: M1.1 (Touch Types), M1.2 (Gesture Recognition)

**Files to create/modify**:
- `internal/input/source_mobile.go` (new)
- `internal/input/source_mobile_test.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M1.3.1 | Create MobileInputSource struct | Struct with gesture manager, touch state tracking | 2h |
| M1.3.2 | Implement TouchIDs() | Wrap ebiten.TouchIDs() with our TouchID type | 1h |
| M1.3.3 | Implement TouchPosition() | Wrap ebiten.TouchPosition() | 1h |
| M1.3.4 | Implement TouchPhase() | Track phase transitions per touch ID | 3h |
| M1.3.5 | Integrate GestureManager | Call Update() each frame, expose gesture state | 2h |
| M1.3.6 | Implement IsPanning/PanDelta | Expose pan gesture state and movement | 2h |
| M1.3.7 | Implement IsPinching/PinchScale | Expose pinch gesture state and scale | 2h |
| M1.3.8 | Implement tap detection methods | IsTapping(), IsDoubleTap(), LongPressPosition() | 2h |
| M1.3.9 | Add build tag for mobile | Use `//go:build android || ios` | 1h |
| M1.3.10 | Create TestTouchSource for testing | Mock implementation for unit tests | 3h |
| M1.3.11 | Write integration tests | Test MobileInputSource with simulated Ebitengine state | 3h |

**Acceptance Criteria**:
- [ ] MobileInputSource implements TouchInputSource interface
- [ ] Gestures detected from real Ebitengine touch events
- [ ] TestTouchSource allows simulating any touch sequence

---

#### Epic M1.4: Responsive Layout System

**Goal**: Create adaptive layout system for phone, tablet, and desktop.

**Dependencies**: None (can parallel with M1.1-M1.3)

**Files to create/modify**:
- `internal/ui/layout.go` (new)
- `internal/ui/layout_test.go` (new)
- `internal/ui/rect.go` (new, if not exists)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M1.4.1 | Define LayoutMode enum | Phone (<600), Tablet (600-1200), Desktop (>1200) | 1h |
| M1.4.2 | Define Rect helper type | X, Y, Width, Height with Contains(), Intersects() | 2h |
| M1.4.3 | Create Layout struct | Mode, dimensions, scale, region rects | 2h |
| M1.4.4 | Implement NewLayout() | Auto-detect mode from dimensions | 2h |
| M1.4.5 | Implement phone layout calc | ResourceBar, MapView, Response, Prompt, TabBar | 3h |
| M1.4.6 | Implement tablet layout calc | Two-column layout with sidebar | 3h |
| M1.4.7 | Implement desktop layout calc | Current three-column layout | 2h |
| M1.4.8 | Add Update() for resize | Recalculate regions on dimension change | 2h |
| M1.4.9 | Add safe area insets | Account for notches, home indicators | 2h |
| M1.4.10 | Write layout tests | Test all three modes, edge cases | 3h |
| M1.4.11 | Add layout visualization tool | Debug overlay showing region boundaries | 2h |

**Acceptance Criteria**:
- [ ] Layout correctly calculates regions for all three modes
- [ ] Safe area insets respected on modern phones
- [ ] Layout updates correctly on orientation change

---

#### Epic M1.5: MapView Touch Integration

**Goal**: Enable touch-based pan and zoom in MapView.

**Dependencies**: M1.3 (MobileInputSource), M1.4 (Layout)

**Files to create/modify**:
- `internal/mapview/mapview.go` (modify)
- `internal/mapview/touch.go` (new)
- `internal/mapview/mapview_test.go` (modify)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M1.5.1 | Add TouchInputSource to MapView | Optional field, nil for desktop | 2h |
| M1.5.2 | Implement touch pan handling | Apply PanDelta to camera offset | 3h |
| M1.5.3 | Implement pinch zoom handling | Apply PinchScale to zoom level | 3h |
| M1.5.4 | Add zoom anchor point | Zoom toward pinch center, not screen center | 3h |
| M1.5.5 | Implement tap-to-select tile | Convert touch position to tile coordinates | 3h |
| M1.5.6 | Implement long-press tile details | Show tile info popup on long press | 3h |
| M1.5.7 | Add momentum scrolling | Continue panning with deceleration after release | 4h |
| M1.5.8 | Add zoom limits | Clamp zoom to min/max, bounce at edges | 2h |
| M1.5.9 | Add double-tap zoom | Zoom in on double-tap location | 2h |
| M1.5.10 | Write touch interaction tests | Test pan, zoom, select with TestTouchSource | 4h |

**Acceptance Criteria**:
- [ ] Map pans smoothly with one-finger drag
- [ ] Map zooms toward pinch center
- [ ] Tiles selectable via tap
- [ ] Momentum scrolling feels natural

---

#### Epic M1.6: Mobile Entry Point and Build Pipeline

**Goal**: Create mobile app entry point and build scripts.

**Dependencies**: M1.3 (MobileInputSource), M1.4 (Layout)

**Files to create/modify**:
- `cmd/mobile/main.go` (new)
- `scripts/build-android.sh` (new)
- `scripts/build-ios.sh` (new)
- `mobile/android/AndroidManifest.xml` (new)
- `mobile/ios/Info.plist` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M1.6.1 | Create cmd/mobile/main.go | Mobile entry point using ebiten/mobile | 2h |
| M1.6.2 | Create MobileApp struct | Implements ebiten.Game for mobile | 3h |
| M1.6.3 | Initialize MobileInputSource | Create and wire touch input on startup | 2h |
| M1.6.4 | Handle screen dimensions | Get device size, create Layout | 2h |
| M1.6.5 | Create AndroidManifest.xml | Package name, permissions, min SDK | 2h |
| M1.6.6 | Create Info.plist for iOS | Bundle ID, capabilities, orientation | 2h |
| M1.6.7 | Write build-android.sh | gomobile build with signing | 3h |
| M1.6.8 | Write build-ios.sh | gomobile build for simulator/device | 3h |
| M1.6.9 | Add Bazel mobile targets | go_library for mobile code | 3h |
| M1.6.10 | Test on Android emulator | Verify APK installs and runs | 2h |
| M1.6.11 | Test on iOS simulator | Verify app builds and runs | 2h |
| M1.6.12 | Document build process | README for mobile build setup | 2h |

**Acceptance Criteria**:
- [ ] APK builds and installs on Android emulator
- [ ] App runs on iOS Simulator
- [ ] Touch navigation works on both platforms
- [ ] Build scripts documented and reproducible

---

#### Epic M1.7: Offline Map Display

**Goal**: Display project structure from bundled/cached data (no server required).

**Dependencies**: M1.5 (MapView Touch), M1.6 (Build Pipeline)

**Files to create/modify**:
- `internal/project/loader.go` (new)
- `internal/project/cache.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M1.7.1 | Create ProjectLoader interface | Load project structure from various sources | 2h |
| M1.7.2 | Implement FileSystemLoader | Load from local filesystem (desktop) | 2h |
| M1.7.3 | Implement BundledLoader | Load from embedded test project | 3h |
| M1.7.4 | Create sample project for testing | Small Go project to bundle in APK | 2h |
| M1.7.5 | Implement ProjectCache | Cache project structure locally | 3h |
| M1.7.6 | Add cache serialization | JSON/protobuf for project structure | 3h |
| M1.7.7 | Wire loader to MobileApp | Load bundled project on startup | 2h |
| M1.7.8 | Test offline map display | Verify map shows without network | 2h |

**Acceptance Criteria**:
- [ ] Mobile app displays map without network connection
- [ ] Bundled sample project renders correctly
- [ ] Cache persists between app launches

---

### Phase M2: Remote Connection (3-4 weeks)

**Goal**: Mobile app connects to relay server and receives Claude events.

---

#### Epic M2.1: WebSocket Protocol Definition

**Goal**: Define the communication protocol between mobile and relay server.

**Dependencies**: None (can start in parallel with M1)

**Files to create/modify**:
- `internal/relay/protocol.go` (new)
- `internal/relay/protocol_test.go` (new)
- `docs/RELAY_PROTOCOL.md` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M2.1.1 | Define Message envelope | Type, ID, Timestamp, Payload fields | 2h |
| M2.1.2 | Define client->server messages | Connect, Prompt, Disconnect | 3h |
| M2.1.3 | Define server->client messages | ProjectInfo, Event, FogUpdate, Response | 3h |
| M2.1.4 | Define Event subtypes | FileRead, FileWrite, ToolCall, BuildStart, etc. | 3h |
| M2.1.5 | Define FogUpdate message | Tile coordinates, new fog state | 2h |
| M2.1.6 | Define error messages | ErrorResponse with code and description | 2h |
| M2.1.7 | Implement JSON serialization | Marshal/unmarshal for all message types | 3h |
| M2.1.8 | Write protocol documentation | Document all message types in markdown | 2h |
| M2.1.9 | Write serialization tests | Round-trip tests for all message types | 2h |

**Acceptance Criteria**:
- [ ] All message types defined and documented
- [ ] JSON serialization works for all types
- [ ] Protocol versioning strategy documented

---

#### Epic M2.2: Relay Server Core

**Goal**: Create the relay server that bridges mobile clients to Claude CLI.

**Dependencies**: M2.1 (Protocol)

**Files to create/modify**:
- `cmd/relay/main.go` (new)
- `internal/relay/server.go` (new)
- `internal/relay/client.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M2.2.1 | Create RelayServer struct | Project path, harness, client map, mutex | 2h |
| M2.2.2 | Implement HTTP server setup | Listen on configurable port | 2h |
| M2.2.3 | Implement WebSocket upgrade | Handle /ws endpoint for connections | 3h |
| M2.2.4 | Implement client registration | Track connected clients, assign IDs | 2h |
| M2.2.5 | Implement client cleanup | Remove on disconnect, close resources | 2h |
| M2.2.6 | Implement message routing | Parse incoming messages, dispatch handlers | 3h |
| M2.2.7 | Implement broadcast | Send message to all connected clients | 2h |
| M2.2.8 | Add connection heartbeat | Ping/pong for connection health | 2h |
| M2.2.9 | Create main.go entry point | CLI flags, graceful shutdown | 2h |
| M2.2.10 | Write server unit tests | Test registration, routing, broadcast | 3h |

**Acceptance Criteria**:
- [ ] Server accepts WebSocket connections
- [ ] Multiple clients can connect simultaneously
- [ ] Graceful shutdown on SIGTERM

---

#### Epic M2.3: Relay Server Harness Integration

**Goal**: Connect relay server to Claude CLI via existing harness system.

**Dependencies**: M2.2 (Server Core)

**Files to create/modify**:
- `internal/relay/harness_bridge.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M2.3.1 | Create HarnessBridge struct | Wraps harness.Harness, emits relay messages | 2h |
| M2.3.2 | Implement Start handling | Start harness on first client prompt | 3h |
| M2.3.3 | Implement prompt forwarding | Forward client prompts to harness | 2h |
| M2.3.4 | Implement event translation | Convert harness events to relay messages | 4h |
| M2.3.5 | Implement fog state tracking | Track which files Claude has read | 3h |
| M2.3.6 | Broadcast events to clients | Send translated events to all clients | 2h |
| M2.3.7 | Handle harness errors | Translate errors to ErrorResponse | 2h |
| M2.3.8 | Implement Stop handling | Stop harness on last client disconnect | 2h |
| M2.3.9 | Write integration tests | Test prompt -> event -> broadcast flow | 4h |

**Acceptance Criteria**:
- [ ] Prompts from mobile reach Claude CLI
- [ ] Claude events broadcast to all connected clients
- [ ] Fog state updates sent when files are read

---

#### Epic M2.4: Project File Serving

**Goal**: Serve project structure and file metadata to mobile clients.

**Dependencies**: M2.2 (Server Core)

**Files to create/modify**:
- `internal/relay/project_handler.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M2.4.1 | Implement project scan | Walk directory tree, build structure | 3h |
| M2.4.2 | Create ProjectInfo message | Directory tree, file count, languages | 2h |
| M2.4.3 | Implement /api/project endpoint | Return project structure as JSON | 2h |
| M2.4.4 | Implement file metadata endpoint | Return file size, modified time, type | 2h |
| M2.4.5 | Add file change watching | Detect filesystem changes, notify clients | 4h |
| M2.4.6 | Implement incremental updates | Send only changed files, not full tree | 3h |
| M2.4.7 | Add .gitignore filtering | Respect ignore patterns | 2h |
| M2.4.8 | Write project serving tests | Test scan, incremental updates | 3h |

**Acceptance Criteria**:
- [ ] Mobile receives full project structure on connect
- [ ] File changes detected and pushed to clients
- [ ] Large projects handled efficiently

---

#### Epic M2.5: Remote Harness Client

**Goal**: Create mobile harness that connects to relay server.

**Dependencies**: M2.1 (Protocol), M1.6 (Mobile Entry Point)

**Files to create/modify**:
- `internal/harness/remote/remote.go` (new)
- `internal/harness/remote/connection.go` (new)
- `internal/harness/remote/remote_test.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M2.5.1 | Create RemoteHarness struct | Server URL, WebSocket conn, event channel | 2h |
| M2.5.2 | Implement Connect() | Establish WebSocket connection | 3h |
| M2.5.3 | Implement reconnection logic | Auto-reconnect with exponential backoff | 4h |
| M2.5.4 | Implement SendPrompt() | Send prompt message over WebSocket | 2h |
| M2.5.5 | Implement event receiving | Parse incoming messages to events | 3h |
| M2.5.6 | Implement Events() channel | Expose events matching harness.Harness | 2h |
| M2.5.7 | Implement Stop() | Close connection cleanly | 2h |
| M2.5.8 | Add connection state tracking | Connecting, Connected, Disconnected, Error | 2h |
| M2.5.9 | Create MockRelayServer for tests | Simulate server for unit testing | 3h |
| M2.5.10 | Write remote harness tests | Test connect, send, receive, reconnect | 4h |

**Acceptance Criteria**:
- [ ] RemoteHarness implements harness.Harness interface
- [ ] Auto-reconnects on network issues
- [ ] Events flow from server to mobile app

---

#### Epic M2.6: Connection UI

**Goal**: UI for connecting mobile app to relay server.

**Dependencies**: M2.5 (Remote Harness), M1.4 (Layout)

**Files to create/modify**:
- `internal/ui/connect_screen.go` (new)
- `internal/ui/qr_scanner.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M2.6.1 | Create ConnectScreen scene | Connection UI layout | 2h |
| M2.6.2 | Implement manual URL entry | Text field for server address | 3h |
| M2.6.3 | Implement QR code display (server) | Generate QR with connection URL | 3h |
| M2.6.4 | Implement QR code scanner (mobile) | Camera-based QR scanning | 6h |
| M2.6.5 | Add recent connections list | Remember last 5 servers | 3h |
| M2.6.6 | Implement connection status display | Connecting spinner, error messages | 2h |
| M2.6.7 | Add connection timeout handling | Show error after 10s | 2h |
| M2.6.8 | Implement settings persistence | Save server URL, preferences | 2h |
| M2.6.9 | Write UI tests | Test connection flow | 3h |

**Acceptance Criteria**:
- [ ] User can connect via manual entry or QR code
- [ ] Recent connections remembered
- [ ] Clear feedback during connection process

---

#### Epic M2.7: Live Fog of War Updates

**Goal**: Update fog of war in real-time as Claude reads files.

**Dependencies**: M2.3 (Harness Integration), M2.5 (Remote Harness)

**Files to create/modify**:
- `internal/mapview/fog_sync.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M2.7.1 | Create FogSyncManager | Receives fog updates, applies to MapView | 2h |
| M2.7.2 | Parse FogUpdate messages | Extract tile coordinates and new state | 2h |
| M2.7.3 | Implement batch fog updates | Apply multiple updates efficiently | 3h |
| M2.7.4 | Add fog transition animation | Fade from foggy to visible | 3h |
| M2.7.5 | Implement initial fog sync | Get full fog state on connect | 3h |
| M2.7.6 | Handle reconnection fog sync | Resync fog state after reconnect | 2h |
| M2.7.7 | Write fog sync tests | Test update application, animations | 3h |

**Acceptance Criteria**:
- [ ] Fog clears in real-time as Claude reads files
- [ ] Fog state persists across reconnections
- [ ] Smooth visual transitions

---

### Phase M3: Interaction (3-4 weeks)

**Goal**: Send prompts from mobile, view responses.

---

#### Epic M3.1: Mobile Prompt Panel

**Goal**: Touch-optimized prompt input for mobile.

**Dependencies**: M1.4 (Layout), M2.5 (Remote Harness)

**Files to create/modify**:
- `internal/ui/prompt_mobile.go` (new)
- `internal/ui/prompt_mobile_test.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M3.1.1 | Create MobilePromptPanel struct | Touch-friendly prompt UI | 2h |
| M3.1.2 | Implement single-line input | Compact input with expand option | 3h |
| M3.1.3 | Implement expand/collapse | Tap to expand to multi-line | 3h |
| M3.1.4 | Add send button | Prominent touch target for submit | 2h |
| M3.1.5 | Implement text selection | Touch-based cursor positioning | 4h |
| M3.1.6 | Add prompt history | Swipe to access previous prompts | 3h |
| M3.1.7 | Implement @ mentions | File/function autocomplete | 4h |
| M3.1.8 | Handle keyboard appearance | Adjust layout when keyboard shows | 3h |
| M3.1.9 | Write prompt panel tests | Test input, expand, history | 3h |

**Acceptance Criteria**:
- [ ] Prompt easily accessible with one tap
- [ ] Text entry comfortable on small screens
- [ ] Keyboard doesn't obscure important UI

---

#### Epic M3.2: Virtual Keyboard Integration

**Goal**: Proper virtual keyboard handling for text input.

**Dependencies**: M3.1 (Prompt Panel)

**Files to create/modify**:
- `internal/ui/keyboard.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M3.2.1 | Create KeyboardManager | Track keyboard visibility and height | 2h |
| M3.2.2 | Implement show/hide keyboard | Use ebiten.SetVirtualKeyboardVisible | 2h |
| M3.2.3 | Track keyboard height | Adjust layout for keyboard | 3h |
| M3.2.4 | Implement keyboard avoidance | Scroll/resize to keep input visible | 4h |
| M3.2.5 | Handle keyboard type | Set appropriate keyboard for input | 2h |
| M3.2.6 | Add done/send button handling | Detect keyboard submit action | 2h |
| M3.2.7 | Implement auto-dismiss | Hide keyboard on tap outside | 2h |
| M3.2.8 | Write keyboard tests | Test show/hide, avoidance | 3h |

**Acceptance Criteria**:
- [ ] Keyboard shows when prompt tapped
- [ ] Input visible above keyboard
- [ ] Keyboard dismisses appropriately

---

#### Epic M3.3: Response Streaming Display

**Goal**: Show Claude's response as it streams in.

**Dependencies**: M2.5 (Remote Harness), M1.4 (Layout)

**Files to create/modify**:
- `internal/ui/response_panel.go` (new)
- `internal/ui/response_panel_test.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M3.3.1 | Create ResponsePanel struct | Streaming response display | 2h |
| M3.3.2 | Implement text streaming | Append text as it arrives | 3h |
| M3.3.3 | Add auto-scroll | Scroll to bottom as text streams | 2h |
| M3.3.4 | Implement scroll lock | Stop auto-scroll when user scrolls up | 2h |
| M3.3.5 | Add markdown rendering | Basic formatting (bold, code, lists) | 4h |
| M3.3.6 | Implement code block styling | Syntax highlighting for code | 4h |
| M3.3.7 | Add copy button for code | Copy code blocks to clipboard | 2h |
| M3.3.8 | Implement collapse/expand | Collapse old responses | 3h |
| M3.3.9 | Add loading indicator | Show thinking animation | 2h |
| M3.3.10 | Write response panel tests | Test streaming, scroll, formatting | 3h |

**Acceptance Criteria**:
- [ ] Response streams character by character
- [ ] Markdown renders correctly
- [ ] Code blocks are readable and copyable

---

#### Epic M3.4: Haptic Feedback System

**Goal**: Add tactile feedback for important events.

**Dependencies**: M1.6 (Mobile Entry Point)

**Files to create/modify**:
- `internal/ui/haptics.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M3.4.1 | Create HapticManager | Abstract haptic feedback | 2h |
| M3.4.2 | Implement Android haptics | Use vibration API | 3h |
| M3.4.3 | Implement iOS haptics | Use Taptic Engine | 3h |
| M3.4.4 | Define haptic patterns | Light tap, success, error, warning | 2h |
| M3.4.5 | Add haptics for tile selection | Light tap on tile tap | 1h |
| M3.4.6 | Add haptics for prompt send | Success feedback | 1h |
| M3.4.7 | Add haptics for errors | Error feedback pattern | 1h |
| M3.4.8 | Add haptics for fog clear | Subtle feedback as files revealed | 2h |
| M3.4.9 | Implement haptics settings | Toggle on/off, intensity | 2h |
| M3.4.10 | Write haptics tests | Test pattern triggering | 2h |

**Acceptance Criteria**:
- [ ] Haptics work on both platforms
- [ ] Feedback is subtle and useful, not annoying
- [ ] User can disable haptics

---

#### Epic M3.5: Touch Target Polish

**Goal**: Ensure all interactive elements meet touch target guidelines.

**Dependencies**: M1.5 (MapView Touch), M3.1 (Prompt)

**Files to create/modify**:
- Various UI files (audit and modify)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M3.5.1 | Audit all touch targets | List all interactive elements | 2h |
| M3.5.2 | Measure current target sizes | Document sizes vs 44pt minimum | 2h |
| M3.5.3 | Increase small targets | Enlarge buttons, tiles, controls | 4h |
| M3.5.4 | Add touch padding | Invisible hit area extension | 3h |
| M3.5.5 | Fix touch overlap issues | Ensure no accidental taps | 3h |
| M3.5.6 | Add visual feedback | Highlight on touch | 3h |
| M3.5.7 | Test with real users | Usability testing session | 4h |
| M3.5.8 | Document touch guidelines | Guidelines for future development | 2h |

**Acceptance Criteria**:
- [ ] All targets meet 44x44pt minimum
- [ ] Clear visual feedback on touch
- [ ] No accidental taps in usability testing

---

### Phase M4: Views & Polish (4-6 weeks)

**Goal**: All major views working on mobile.

---

#### Epic M4.1: Mobile Tab Bar Navigation

**Goal**: Bottom navigation for switching between views.

**Dependencies**: M1.4 (Layout)

**Files to create/modify**:
- `internal/ui/tabbar.go` (new)
- `internal/ui/tabbar_test.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M4.1.1 | Create TabBar struct | Bottom navigation component | 2h |
| M4.1.2 | Define tab items | Map, Buildings, Units, Advisors, More | 2h |
| M4.1.3 | Implement tab rendering | Icons with labels | 3h |
| M4.1.4 | Implement tab selection | Touch handling, active state | 3h |
| M4.1.5 | Add selection indicator | Animated indicator for active tab | 2h |
| M4.1.6 | Implement badge support | Notification badges on tabs | 2h |
| M4.1.7 | Add tab bar animation | Slide/fade view transitions | 3h |
| M4.1.8 | Handle landscape orientation | Adjust layout for wide screens | 2h |
| M4.1.9 | Implement haptic on tab change | Light tap feedback | 1h |
| M4.1.10 | Write tab bar tests | Test selection, badges, transitions | 3h |

**Acceptance Criteria**:
- [ ] Easy one-thumb access to all views
- [ ] Clear indication of active view
- [ ] Smooth transitions between views

---

#### Epic M4.2: Buildings View (Mobile)

**Goal**: Port build targets view to mobile.

**Dependencies**: M4.1 (Tab Bar), M2.5 (Remote Harness)

**Files to create/modify**:
- `internal/ui/buildings_mobile.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M4.2.1 | Create MobileBuildingsView | Touch-optimized building list | 2h |
| M4.2.2 | Implement building cards | Card UI for each build target | 3h |
| M4.2.3 | Add build status indicators | Pass/fail/building icons | 2h |
| M4.2.4 | Implement tap for details | Expand card to show details | 3h |
| M4.2.5 | Add build duration display | Time since last build | 2h |
| M4.2.6 | Implement pull-to-refresh | Trigger build status refresh | 3h |
| M4.2.7 | Add build trigger button | Start build from mobile | 3h |
| M4.2.8 | Implement build log viewer | Scrollable log output | 4h |
| M4.2.9 | Write buildings view tests | Test cards, status, logs | 3h |

**Acceptance Criteria**:
- [ ] All build targets visible
- [ ] Build status updates in real-time
- [ ] Build logs readable on mobile

---

#### Epic M4.3: Units View (Mobile)

**Goal**: Port tests view to mobile.

**Dependencies**: M4.1 (Tab Bar), M2.5 (Remote Harness)

**Files to create/modify**:
- `internal/ui/units_mobile.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M4.3.1 | Create MobileUnitsView | Touch-optimized test list | 2h |
| M4.3.2 | Implement test grouping | Group by package/file | 3h |
| M4.3.3 | Add collapsible groups | Tap to expand/collapse | 2h |
| M4.3.4 | Implement test status icons | Pass/fail/skip indicators | 2h |
| M4.3.5 | Add test duration display | Time for each test | 2h |
| M4.3.6 | Implement tap for failure details | Show failure message | 3h |
| M4.3.7 | Add coverage visualization | Simple coverage bar | 3h |
| M4.3.8 | Implement test run trigger | Run tests from mobile | 3h |
| M4.3.9 | Add filtering | Filter by status, name | 3h |
| M4.3.10 | Write units view tests | Test grouping, filtering, status | 3h |

**Acceptance Criteria**:
- [ ] Test results clearly visible
- [ ] Failure details accessible
- [ ] Can trigger test runs

---

#### Epic M4.4: Advisors Panel (Mobile)

**Goal**: Port advisors/subagents view to mobile.

**Dependencies**: M4.1 (Tab Bar), M2.5 (Remote Harness)

**Files to create/modify**:
- `internal/ui/advisors_mobile.go` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M4.4.1 | Create MobileAdvisorsView | Touch-optimized advisor list | 2h |
| M4.4.2 | Implement advisor cards | Card for each advisor type | 3h |
| M4.4.3 | Add advisor status | Active/idle/insights count | 2h |
| M4.4.4 | Implement tap for insights | Show advisor's insights | 3h |
| M4.4.5 | Add insight cards | Individual insight display | 3h |
| M4.4.6 | Implement insight actions | Dismiss, act on insight | 3h |
| M4.4.7 | Add advisor settings | Configure advisor behavior | 3h |
| M4.4.8 | Implement advisor metrics | Token usage, performance | 2h |
| M4.4.9 | Write advisors view tests | Test cards, insights, actions | 3h |

**Acceptance Criteria**:
- [ ] All advisors visible with status
- [ ] Insights accessible and actionable
- [ ] Settings adjustable from mobile

---

#### Epic M4.5: Dark Mode Support

**Goal**: System-integrated dark mode.

**Dependencies**: M1.4 (Layout)

**Files to create/modify**:
- `internal/ui/theme.go` (new)
- Various UI files (modify for theming)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M4.5.1 | Create Theme struct | Color palette for light/dark | 2h |
| M4.5.2 | Define light theme colors | Background, text, accent colors | 2h |
| M4.5.3 | Define dark theme colors | Dark mode color palette | 2h |
| M4.5.4 | Detect system theme | Read OS dark mode setting | 3h |
| M4.5.5 | Implement theme switching | Apply theme across all components | 4h |
| M4.5.6 | Update MapView for themes | Tile colors, fog, grid | 3h |
| M4.5.7 | Update panels for themes | Prompt, response, tab bar | 3h |
| M4.5.8 | Add manual override | Settings to force light/dark | 2h |
| M4.5.9 | Handle theme change at runtime | Smooth transition | 3h |
| M4.5.10 | Write theme tests | Test both themes, switching | 2h |

**Acceptance Criteria**:
- [ ] App respects system dark mode
- [ ] All UI elements themed correctly
- [ ] Manual override works

---

#### Epic M4.6: Performance Optimization

**Goal**: Ensure smooth 60fps on mid-range devices.

**Dependencies**: All previous epics

**Files to create/modify**:
- Various (based on profiling)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M4.6.1 | Profile on target devices | Identify bottlenecks | 4h |
| M4.6.2 | Optimize tile rendering | Reduce draw calls for map | 4h |
| M4.6.3 | Implement tile culling | Only render visible tiles | 4h |
| M4.6.4 | Optimize text rendering | Cache text textures | 3h |
| M4.6.5 | Reduce memory allocations | Pool frequently created objects | 4h |
| M4.6.6 | Optimize gesture processing | Efficient touch handling | 3h |
| M4.6.7 | Add frame rate limiter | Cap at 60fps, save battery | 2h |
| M4.6.8 | Implement level-of-detail | Simplify distant tiles | 4h |
| M4.6.9 | Profile memory usage | Identify leaks, reduce footprint | 4h |
| M4.6.10 | Add performance monitoring | Optional FPS/memory overlay | 2h |
| M4.6.11 | Test on low-end devices | Verify acceptable performance | 4h |

**Acceptance Criteria**:
- [ ] Consistent 60fps on mid-range devices
- [ ] No memory leaks
- [ ] Acceptable performance on low-end devices

---

#### Epic M4.7: App Store Preparation

**Goal**: Prepare for iOS App Store and Google Play submission.

**Dependencies**: All functionality complete

**Files to create/modify**:
- `mobile/android/` (various)
- `mobile/ios/` (various)
- `docs/APP_STORE.md` (new)

**Tasks**:

| ID | Task | Description | Est |
|----|------|-------------|-----|
| M4.7.1 | Create app icon | All required sizes | 4h |
| M4.7.2 | Create launch screen | Splash screen for both platforms | 3h |
| M4.7.3 | Write app description | Store listing copy | 2h |
| M4.7.4 | Create screenshots | Phone and tablet screenshots | 4h |
| M4.7.5 | Create promotional graphics | Feature graphic, banner | 3h |
| M4.7.6 | Set up privacy policy | Required for both stores | 3h |
| M4.7.7 | Configure Android signing | Production keystore | 2h |
| M4.7.8 | Configure iOS provisioning | App Store certificates | 3h |
| M4.7.9 | Set up Play Console | Developer account, app listing | 2h |
| M4.7.10 | Set up App Store Connect | Developer account, app listing | 2h |
| M4.7.11 | Implement GDPR compliance | Data handling disclosure | 3h |
| M4.7.12 | Test in-app review | Rating prompt | 2h |
| M4.7.13 | Submit for review | Initial submission | 2h |
| M4.7.14 | Address review feedback | Handle rejection issues | 4h |

**Acceptance Criteria**:
- [ ] App approved on both stores
- [ ] All store assets complete
- [ ] Privacy policy published

---

## Part 7 Summary: Epic Dependencies

```
M1.1 Touch Types ──┬──> M1.2 Gestures ──┬──> M1.3 Mobile Input ──> M1.5 MapView Touch
                   │                    │
                   └────────────────────┴──> M1.7 Offline Map
                                        │
M1.4 Layout ────────────────────────────┴──> M1.6 Build Pipeline

M2.1 Protocol ──┬──> M2.2 Server Core ──┬──> M2.3 Harness Bridge ──> M2.7 Fog Sync
                │                       │
                │                       └──> M2.4 Project Serving
                │
                └──> M2.5 Remote Harness ──> M2.6 Connection UI

M3.1 Prompt ──┬──> M3.2 Keyboard
              │
              └──> M3.3 Response ──> M3.4 Haptics ──> M3.5 Touch Polish

M4.1 Tab Bar ──┬──> M4.2 Buildings
               │
               ├──> M4.3 Units
               │
               └──> M4.4 Advisors

M4.5 Dark Mode (parallel)
M4.6 Performance (after all features)
M4.7 App Store (final)
```

---

## Part 8: Testing Strategy

### 8.1 Input Testing

The existing `TestInputSource` in `/internal/testutil/input.go` provides a pattern. Extend for touch:

```go
// internal/testutil/touch.go
type TestTouchSource struct {
    *TestInputSource
    touches map[TouchID]TouchState
}

func (t *TestTouchSource) SimulateTap(x, y int)
func (t *TestTouchSource) SimulatePinch(scale float64)
func (t *TestTouchSource) SimulatePan(dx, dy float64)
func (t *TestTouchSource) SimulateLongPress(x, y int, duration time.Duration)
```

### 8.2 Integration Testing

1. **Desktop simulation**: Test mobile layouts at small window sizes
2. **Emulator testing**: Android Studio emulator, iOS Simulator
3. **Device testing**: Real devices for performance and gesture accuracy

### 8.3 Network Testing

1. **Mock relay server**: For unit testing remote harness
2. **Latency simulation**: Test with artificial network delay
3. **Reconnection testing**: Handle network drops gracefully

---

## Part 9: Architectural Decisions

### 9.1 Decision: Extend InputSource vs New System

**Decision**: Extend the existing `InputSource` interface.

**Rationale**: The interface already abstracts input, and extending it maintains compatibility with existing code. Components that accept `InputSource` will work with both desktop and mobile implementations.

### 9.2 Decision: Single Codebase vs Separate App

**Decision**: Single codebase with build tags for platform-specific code.

**Rationale**: Maximizes code reuse. Use Go build tags (`//go:build android`) for platform-specific implementations while keeping shared logic unified.

### 9.3 Decision: Relay Server vs Cloud Service

**Decision**: Start with local relay server, design for future cloud option.

**Rationale**: Local relay is simpler, more secure, and does not require cloud infrastructure. The interface can later support a cloud relay for remote access.

### 9.4 Decision: Native UI vs Game UI

**Decision**: Use Ebitengine game UI for everything.

**Rationale**: Consistency with desktop version, single codebase, cross-platform rendering. Native UI (Android Views, iOS UIKit) would require separate implementations.

---

## Part 10: Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Touch gesture conflicts | Medium | Medium | Clear gesture zones, user testing |
| Network latency | Medium | Medium | Optimistic UI, event buffering |
| Mobile performance | Low | High | Profile early, optimize rendering |
| App store rejection | Low | High | Follow platform guidelines |
| Screen fragmentation | High | Low | Responsive layout, test many sizes |
| Virtual keyboard issues | Medium | Medium | Test on real devices |

---

## Appendix A: File Structure for Mobile Support

```
CodingGame/
├── cmd/
│   ├── codinggame/          # Desktop entry point (rename from main.go)
│   │   └── main.go
│   ├── mobile/              # Mobile entry point
│   │   └── main.go
│   └── relay/               # Relay server
│       └── main.go
├── internal/
│   ├── input/
│   │   ├── source.go        # InputSource interface
│   │   ├── source_desktop.go # EbitenInputSource
│   │   ├── source_mobile.go  # MobileInputSource (new)
│   │   ├── touch.go          # Touch types (new)
│   │   └── gestures.go       # Gesture recognition (new)
│   ├── ui/
│   │   ├── layout.go         # Responsive layout (new)
│   │   ├── layout_test.go
│   │   └── tabbar.go         # Mobile tab bar (new)
│   ├── harness/
│   │   └── remote/           # Network harness (new)
│   │       ├── remote.go
│   │       ├── protocol.go
│   │       └── remote_test.go
│   └── game/
│       ├── app_desktop.go    # Desktop-specific app code
│       └── app_mobile.go     # Mobile-specific app code (new)
├── mobile/                   # Mobile assets (new)
│   ├── android/
│   │   └── AndroidManifest.xml
│   └── ios/
│       └── Info.plist
└── scripts/
    ├── build-android.sh      # Android build script (new)
    └── build-ios.sh          # iOS build script (new)
```

---

## Appendix B: Critical Files for Implementation

The following existing files are most relevant for mobile implementation:

- `internal/input/source.go` - Core input abstraction to extend for touch support
- `internal/input/handler.go` - Input processing logic requiring touch gesture handling
- `internal/harness/harness.go` - Harness interface to implement for remote connections
- `internal/game/scene.go` - GameScene requiring mobile layout and touch input integration
- `internal/mapview/mapview.go` - Core visualization requiring touch pan/zoom/tap support

---

## Appendix C: Estimated Timeline Summary

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| M1: Foundation | 4-6 weeks | Touchable map display |
| M2: Remote Connection | 3-4 weeks | Real-time Claude events |
| M3: Interaction | 3-4 weeks | Full prompt/response flow |
| M4: Views & Polish | 4-6 weeks | Production-ready app |
| **Total** | **14-20 weeks** | **App store release** |

---

## Appendix D: Alternative Approaches Considered

### D.1 React Native / Flutter Wrapper
**Rejected**: Would require reimplementing all visualization logic. Ebitengine already supports mobile natively.

### D.2 Web-Based Mobile App (PWA)
**Rejected**: WebGL performance concerns, offline capabilities limited, and we already have a native Go codebase.

### D.3 Companion App Only (No Direct Interaction)
**Rejected**: Users expect to be able to send prompts from mobile. Read-only would limit usefulness.

### D.4 Full Local Claude on Mobile
**Rejected**: Claude CLI cannot run on mobile. Would require a completely different architecture with on-device models.
