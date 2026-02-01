package game

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/tedks/CodingGame/internal/advisor"
	"github.com/tedks/CodingGame/internal/claude"
	"github.com/tedks/CodingGame/internal/mapview"
)

func processClaudeEvent(event *claude.Event, projectPath string, mapView *mapview.MapView, advisorPool *advisor.Pool) {
	if event == nil {
		return
	}

	switch event.Type {
	case claude.EventFileRead:
		if path, ok := event.Data["file_path"].(string); ok {
			revealFile(mapView, projectPath, path)
		}

	case claude.EventFileWrite, claude.EventFileEdit:
		if path, ok := event.Data["file_path"].(string); ok {
			revealAndTrigger(mapView, advisorPool, projectPath, path)
		}

	case claude.EventBuildRun:
		// Update build status in resource tracker
		// TODO: Extract build results and update resources

	case claude.EventTestRun:
		// Update test results
		// TODO: Extract test results and update resources

	case claude.EventSubagentRun:
		advisorID, _ := event.Data["advisor_id"].(string)
		processAdvisorEvent(advisorPool, advisorID, event.Data)
	}
}

func resolveFilePath(projectPath, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(projectPath, path)
}

func revealFile(mapView *mapview.MapView, projectPath, path string) {
	if mapView == nil || path == "" {
		return
	}
	absPath := resolveFilePath(projectPath, path)
	if absPath == "" {
		return
	}
	mapView.RevealTile(absPath)
}

func revealAndTrigger(mapView *mapview.MapView, advisorPool *advisor.Pool, projectPath, path string) {
	if path == "" {
		return
	}
	revealFile(mapView, projectPath, path)
	triggerAdvisorsForFile(advisorPool, path)
}

func triggerAdvisorsForFile(advisorPool *advisor.Pool, filePath string) {
	if advisorPool == nil || filePath == "" {
		return
	}
	triggered := advisorPool.TriggerOnFileChange(filePath)
	for _, adv := range triggered {
		// In a real implementation, this would spawn the advisor subagent
		// For now, we just mark that the advisor should analyze this file
		_ = adv // TODO: Implement actual advisor execution
	}
}

func processAdvisorEvent(advisorPool *advisor.Pool, advisorID string, data map[string]interface{}) {
	if advisorPool == nil || advisorID == "" || data == nil {
		return
	}

	adv := advisorPool.Get(advisorID)
	if adv == nil {
		return
	}

	if status, ok := data["status"].(string); ok {
		switch status {
		case "started":
			adv.StartAnalysis()
		case "completed":
			duration, _ := data["duration_ms"].(float64)
			tokensIn, _ := data["tokens_in"].(float64)
			tokensOut, _ := data["tokens_out"].(float64)
			adv.CompleteAnalysis(
				durationFromMs(duration),
				int64(tokensIn),
				int64(tokensOut),
				nil,
			)
		case "error":
			errMsg, _ := data["error"].(string)
			adv.CompleteAnalysis(0, 0, 0, fmt.Errorf("%s", errMsg))
		}
	}

	if insightsData, ok := data["insights"].([]interface{}); ok {
		for _, insightData := range insightsData {
			insightMap, ok := insightData.(map[string]interface{})
			if !ok {
				continue
			}
			insight := parseInsight(advisorID, insightMap)
			if insight != nil {
				adv.AddInsight(insight)
			}
		}
	}
}

func parseInsight(advisorID string, data map[string]interface{}) *advisor.Insight {
	title, _ := data["title"].(string)
	description, _ := data["description"].(string)
	if title == "" {
		return nil
	}

	severityStr, _ := data["severity"].(string)
	severity := advisor.SeverityInfo
	switch severityStr {
	case "warning":
		severity = advisor.SeverityWarning
	case "critical":
		severity = advisor.SeverityCritical
	}

	categoryStr, _ := data["category"].(string)
	category := advisor.CategoryGeneral
	switch categoryStr {
	case "security":
		category = advisor.CategorySecurity
	case "performance":
		category = advisor.CategoryPerformance
	case "refactoring":
		category = advisor.CategoryRefactoring
	case "testing":
		category = advisor.CategoryTesting
	}

	insight := advisor.NewInsight(advisorID, title, description, severity, category)

	if filePath, ok := data["file_path"].(string); ok {
		line, _ := data["line"].(float64)
		column, _ := data["column"].(float64)
		insight.WithLocation(filePath, int(line), int(column))
	}

	if suggestion, ok := data["suggestion"].(string); ok {
		codeBefore, _ := data["code_before"].(string)
		codeAfter, _ := data["code_after"].(string)
		insight.WithSuggestion(suggestion, codeBefore, codeAfter)
	}

	return insight
}

func durationFromMs(ms float64) time.Duration {
	return time.Duration(ms * float64(time.Millisecond))
}
