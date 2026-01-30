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

## Part 7: Implementation Phases

### Phase M1: Foundation (4-6 weeks)

**Goal**: Mobile app that can display a project map with touch navigation.

**Tasks**:
1. Create `internal/input/touch.go` with gesture recognition
2. Implement `MobileInputSource` for touch input
3. Add mobile layout system in `internal/ui/layout.go`
4. Create mobile entry point `cmd/mobile/main.go`
5. Set up gomobile build pipeline
6. Test on Android emulator and iOS simulator

**Deliverable**: APK/IPA that shows project directory as touchable map.

### Phase M2: Remote Connection (3-4 weeks)

**Goal**: Mobile app connects to relay server and receives Claude events.

**Tasks**:
1. Create relay server `cmd/relay/main.go`
2. Implement WebSocket protocol for events
3. Create `internal/harness/remote/` network harness
4. Add connection UI (QR code scanner or manual entry)
5. Stream fog of war updates from desktop to mobile

**Deliverable**: Mobile app showing real-time updates as Claude works.

### Phase M3: Interaction (3-4 weeks)

**Goal**: Send prompts from mobile, view responses.

**Tasks**:
1. Adapt prompt panel for mobile input
2. Integrate virtual keyboard
3. Implement response streaming display
4. Add haptic feedback for events
5. Polish touch targets and gesture recognition

**Deliverable**: Fully interactive mobile client.

### Phase M4: Views & Polish (4-6 weeks)

**Goal**: All major views working on mobile.

**Tasks**:
1. Port Buildings view (build targets)
2. Port Units view (tests)
3. Port Advisors panel
4. Implement mobile tab bar navigation
5. Add dark mode support
6. Performance optimization
7. App store preparation (icons, screenshots, metadata)

**Deliverable**: Production-ready mobile app.

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
