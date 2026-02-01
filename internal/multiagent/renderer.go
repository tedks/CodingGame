package multiagent

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/tedks/CodingGame/internal/theme"
)

// Renderer draws the multi-agent orchestration UI.
// Shows all active agents with their status, task, and context usage.
type Renderer struct{}

// NewRenderer creates a new multi-agent renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// Color scheme for agent statuses
var statusColors = map[AgentStatus]color.RGBA{
	StatusIdle:      theme.StatusNeutral,
	StatusWorking:   theme.StatusSuccess,
	StatusPaused:    theme.StatusWarning,
	StatusCompleted: theme.StatusInfo,
	StatusError:     theme.StatusError,
}

// Draw renders the multi-agent orchestration view.
// Parameters:
//   - screen: the Ebiten image to draw to
//   - agents: the agents to display
//   - x, y: the top-left position to start drawing
//   - width, height: the available drawing area
func (r *Renderer) Draw(screen *ebiten.Image, agents []*Agent, x, y, width, height int) {
	// Draw header with summary
	r.drawHeader(screen, agents, x, y, width)

	// If no agents, show empty state
	if len(agents) == 0 {
		r.drawEmptyState(screen, x, y+theme.PanelHeaderHeight, width, height-theme.PanelHeaderHeight)
		return
	}

	// Draw agents as cards
	startY := y + theme.PanelHeaderHeight + theme.PanelPadding
	for i, agent := range agents {
		row := i / theme.MultiAgentCardsPerRow
		col := i % theme.MultiAgentCardsPerRow

		cardX := x + theme.PanelPadding + col*(theme.MultiAgentCardWidth+theme.MultiAgentCardSpacing)
		cardY := startY + row*(theme.MultiAgentCardHeight+theme.MultiAgentCardSpacing)

		// Check if we're still in bounds
		if cardY+theme.MultiAgentCardHeight > y+height {
			break
		}

		r.drawAgentCard(screen, agent, cardX, cardY)
	}
}

func tokenSummaryX(x, width int) (int, bool) {
	return theme.RightAlignedX(x, width, theme.MultiAgentTokenSummaryWidth, theme.PanelPadding)
}

// drawHeader renders the summary header.
func (r *Renderer) drawHeader(screen *ebiten.Image, agents []*Agent, x, y, width int) {
	// Draw background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(theme.PanelHeaderHeight),
		theme.HeaderBackground, false)

	// Draw title
	ebitenutil.DebugPrintAt(screen, "MULTI-AGENT ORCHESTRATOR", x+theme.PanelPadding, y+theme.PanelHeaderTitleOffsetY)

	// Calculate status summary
	statusCounts := make(map[AgentStatus]int)
	var totalTokens int64
	for _, agent := range agents {
		statusCounts[agent.Status()]++
		totalTokens += agent.TokensUsed()
	}

	// Draw status summary
	summaryY := y + theme.PanelHeaderSummaryOffsetY
	summaryX := x + theme.PanelPadding
	working := statusCounts[StatusWorking]
	idle := statusCounts[StatusIdle]
	paused := statusCounts[StatusPaused]

	summary := fmt.Sprintf("Agents: %d | Working: %d | Idle: %d | Paused: %d",
		len(agents), working, idle, paused)
	ebitenutil.DebugPrintAt(screen, summary, summaryX, summaryY)

	// Draw token usage
	tokenX, ok := tokenSummaryX(x, width)
	if ok {
		tokenSummary := fmt.Sprintf("Total tokens: %dk", totalTokens/1000)
		ebitenutil.DebugPrintAt(screen, tokenSummary, tokenX, summaryY)
	}
}

// drawEmptyState renders a message when no agents are active.
func (r *Renderer) drawEmptyState(screen *ebiten.Image, x, y, width, height int) {
	msg := "No active agents"
	hint := "Agents will appear here when active"

	// Center the messages
	msgX := theme.CenterTextX(x, width, msg, theme.CompactTextCharWidth)
	msgY := y + height/2 - theme.EmptyStateLineOffset
	hintX := theme.CenterTextX(x, width, hint, theme.CompactTextCharWidth)
	hintY := y + height/2

	ebitenutil.DebugPrintAt(screen, msg, msgX, msgY)
	ebitenutil.DebugPrintAt(screen, hint, hintX, hintY)
}

// drawAgentCard renders a single agent as a card.
func (r *Renderer) drawAgentCard(screen *ebiten.Image, agent *Agent, x, y int) {
	// Get color based on status
	statusColor := statusColors[agent.Status()]

	// Draw card background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(theme.MultiAgentCardWidth), float32(theme.MultiAgentCardHeight),
		theme.CardBackground, false)

	// Draw status indicator border
	vector.StrokeRect(screen, float32(x), float32(y), float32(theme.MultiAgentCardWidth), float32(theme.MultiAgentCardHeight),
		theme.CardBorderWidth, statusColor, false)

	// Draw agent name
	nameY := y + theme.MultiAgentNameOffsetY
	ebitenutil.DebugPrintAt(screen, agent.Name(), x+theme.CardTextPadding, nameY)

	// Draw status indicator
	statusY := y + theme.MultiAgentStatusOffsetY
	statusStr := fmt.Sprintf("[%s]", agent.Status())
	ebitenutil.DebugPrintAt(screen, statusStr, x+theme.CardTextPadding, statusY)

	// Draw current task (truncated)
	taskY := y + theme.MultiAgentTaskOffsetY
	task := agent.CurrentTask()
	if len(task) > 25 {
		task = task[:22] + "..."
	}
	if task == "" {
		task = "(no task assigned)"
	}
	ebitenutil.DebugPrintAt(screen, task, x+theme.CardTextPadding, taskY)

	// Draw context usage bar
	usageY := y + theme.MultiAgentUsageOffsetY
	usageWidth := theme.MultiAgentCardWidth - 2*theme.CardTextPadding
	usageHeight := theme.MultiAgentUsageBarHeight
	usage := clampUnitInterval(agent.ContextUsage())

	// Background
	vector.DrawFilledRect(screen, float32(x+theme.CardTextPadding), float32(usageY), float32(usageWidth), float32(usageHeight),
		theme.UsageBarBackground, false)

	// Fill based on usage
	usageColor := getUsageColor(usage)
	fillWidth := int(float64(usageWidth) * usage)
	if fillWidth > 0 {
		vector.DrawFilledRect(screen, float32(x+theme.CardTextPadding), float32(usageY), float32(fillWidth), float32(usageHeight),
			usageColor, false)
	}

	// Draw usage percentage
	usageLabelY := y + theme.MultiAgentUsageLabelOffsetY
	usageLabel := fmt.Sprintf("Context: %.0f%% | Files: %d", usage*100, agent.FileCount())
	ebitenutil.DebugPrintAt(screen, usageLabel, x+theme.CardTextPadding, usageLabelY)

	// Draw tokens used
	tokensY := y + theme.MultiAgentTokensOffsetY
	tokensLabel := fmt.Sprintf("Tokens: %dk / %dk", agent.TokensUsed()/1000, agent.TokenLimit()/1000)
	ebitenutil.DebugPrintAt(screen, tokensLabel, x+theme.CardTextPadding, tokensY)
}

// getUsageColor returns a color based on context usage percentage.
func getUsageColor(usage float64) color.RGBA {
	if usage < 0.5 {
		return theme.StatusSuccess
	} else if usage < 0.8 {
		return theme.StatusWarning
	}
	return theme.StatusError
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
