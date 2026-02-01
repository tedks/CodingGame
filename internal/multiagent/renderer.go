package multiagent

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/tedks/CodingGame/internal/ui"
)

// Renderer draws the multi-agent orchestration UI.
// Shows all active agents with their status, task, and context usage.
type Renderer struct{}

// NewRenderer creates a new multi-agent renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// Layout constants
const (
	agentCardWidth   = 200
	agentCardHeight  = 120
	agentCardSpacing = 20
	agentCardsPerRow = 4
)

// Color scheme for agent statuses
var statusColors = map[AgentStatus]color.RGBA{
	StatusIdle:      {R: 0x75, G: 0x75, B: 0x75, A: 0xFF}, // Gray
	StatusWorking:   {R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF}, // Green
	StatusPaused:    {R: 0xF5, G: 0x7F, B: 0x17, A: 0xFF}, // Orange
	StatusCompleted: {R: 0x19, G: 0x76, B: 0xD2, A: 0xFF}, // Blue
	StatusError:     {R: 0xC6, G: 0x28, B: 0x28, A: 0xFF}, // Red
}

// Draw renders the multi-agent orchestration view.
// Parameters:
//   - screen: the Ebiten image to draw to
//   - agents: the agents to display
//   - x, y: the top-left position to start drawing
//   - width, height: the available drawing area
func (r *Renderer) Draw(screen *ebiten.Image, agents []*Agent, x, y, width, height int) {
	snapshots := make([]AgentSnapshot, len(agents))
	for i, agent := range agents {
		snapshots[i] = agent.Snapshot()
	}

	headerLayout := ui.DefaultHeaderLayout()

	// Draw header with summary
	r.drawHeader(screen, snapshots, x, y, width, headerLayout)

	// If no agents, show empty state
	if len(snapshots) == 0 {
		ui.DrawEmptyState(screen, "No active agents", "Agents will appear here when active",
			x, y+headerLayout.Height, width, height-headerLayout.Height)
		return
	}

	// Draw agents as cards
	startY := y + headerLayout.Height + headerLayout.Padding
	for i, agent := range snapshots {
		row := i / agentCardsPerRow
		col := i % agentCardsPerRow

		cardX := x + headerLayout.Padding + col*(agentCardWidth+agentCardSpacing)
		cardY := startY + row*(agentCardHeight+agentCardSpacing)

		// Check if we're still in bounds
		if cardY+agentCardHeight > y+height {
			break
		}

		r.drawAgentCard(screen, agent, cardX, cardY)
	}
}

func tokenSummaryX(x, width int) (int, bool) {
	layout := ui.DefaultHeaderLayout()
	return ui.RightAlignedSummaryX(x, width, 200, layout.Padding)
}

// drawHeader renders the summary header.
func (r *Renderer) drawHeader(screen *ebiten.Image, agents []AgentSnapshot, x, y, width int, layout ui.HeaderLayout) {
	ui.DrawHeader(screen, "MULTI-AGENT ORCHESTRATOR", x, y, width, layout)

	// Calculate status summary
	statusCounts := make(map[AgentStatus]int)
	var totalTokens int64
	for _, agent := range agents {
		statusCounts[agent.Status]++
		totalTokens += agent.TokensUsed
	}

	// Draw status summary
	summaryY := y + layout.SummaryOffsetY
	working := statusCounts[StatusWorking]
	idle := statusCounts[StatusIdle]
	paused := statusCounts[StatusPaused]

	summary := fmt.Sprintf("Agents: %d | Working: %d | Idle: %d | Paused: %d",
		len(agents), working, idle, paused)
	ui.DrawHeaderSummary(screen, summary, x, y, layout)

	// Draw token usage
	tokenX, ok := tokenSummaryX(x, width)
	if ok {
		tokenSummary := fmt.Sprintf("Total tokens: %dk", totalTokens/1000)
		ebitenutil.DebugPrintAt(screen, tokenSummary, tokenX, summaryY)
	}
}

// drawAgentCard renders a single agent as a card.
func (r *Renderer) drawAgentCard(screen *ebiten.Image, agent AgentSnapshot, x, y int) {
	// Get color based on status
	statusColor := statusColors[agent.Status]

	// Draw card background
	ui.DrawCard(screen, x, y, agentCardWidth, agentCardHeight, ui.DefaultCardStyle(statusColor))

	// Draw agent name
	nameY := y + 8
	ebitenutil.DebugPrintAt(screen, agent.Name, x+8, nameY)

	// Draw status indicator
	statusY := y + 24
	statusStr := fmt.Sprintf("[%s]", agent.Status)
	ebitenutil.DebugPrintAt(screen, statusStr, x+8, statusY)

	// Draw current task (truncated)
	taskY := y + 44
	task := agent.CurrentTask
	if len(task) > 25 {
		task = task[:22] + "..."
	}
	if task == "" {
		task = "(no task assigned)"
	}
	ebitenutil.DebugPrintAt(screen, task, x+8, taskY)

	// Draw context usage bar
	usageY := y + 68
	usageWidth := agentCardWidth - 16
	usageHeight := 12
	usage := clampUnitInterval(agent.ContextUsage)

	// Fill based on usage
	usageColor := getUsageColor(usage)
	ui.DrawProgressBar(screen, x+8, usageY, usageWidth, usageHeight, usage,
		color.RGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF}, usageColor)

	// Draw usage percentage
	usageLabelY := y + 84
	usageLabel := fmt.Sprintf("Context: %.0f%% | Files: %d", usage*100, agent.FileCount)
	ebitenutil.DebugPrintAt(screen, usageLabel, x+8, usageLabelY)

	// Draw tokens used
	tokensY := y + 100
	tokensLabel := fmt.Sprintf("Tokens: %dk / %dk", agent.TokensUsed/1000, agent.TokenLimit/1000)
	ebitenutil.DebugPrintAt(screen, tokensLabel, x+8, tokensY)
}

// getUsageColor returns a color based on context usage percentage.
func getUsageColor(usage float64) color.RGBA {
	if usage < 0.5 {
		return color.RGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF} // Green
	} else if usage < 0.8 {
		return color.RGBA{R: 0xF5, G: 0x7F, B: 0x17, A: 0xFF} // Orange
	}
	return color.RGBA{R: 0xC6, G: 0x28, B: 0x28, A: 0xFF} // Red
}

func clampUnitInterval(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// GetStatusColor returns the color associated with a status.
// Exported for use by other renderers or tests.
func GetStatusColor(s AgentStatus) color.RGBA {
	if c, ok := statusColors[s]; ok {
		return c
	}
	return statusColors[StatusIdle]
}
