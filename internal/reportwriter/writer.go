package reportwriter

import (
	"compliance-probe/playbook"
	"compliance-probe/report"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var DefaultReportsDir = "reports"

// DispatchReport decides where to send the report based on the configuration.
func DispatchReport(config *playbook.ReportConfig, res report.FinalResult) error {
	switch config.ReportDestination {
	case playbook.ReportDestinationHTTPS:
		if config.ReportDestinationHTTPS == nil {
			return fmt.Errorf("reportDestination is 'https' but reportDestinationHttps is missing")
		}
		return WriteToHTTP(config.ReportDestinationHTTPS, res)
	case playbook.ReportDestinationFolder, "":
		return WriteToFolder(DefaultReportsDir, res)
	default:
		return fmt.Errorf("unknown reportDestination: %s", config.ReportDestination)
	}
}

// WriteToFolder saves the report files to a local directory.
func WriteToFolder(reportsDir string, res report.FinalResult) error {
	if reportsDir == "" {
		reportsDir = "reports"
	}
	now := time.Now()
	timestamp := now.Format("060102-150405")

	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(reportsDir, 0755); err != nil {
			return fmt.Errorf("failed to create reports directory: %w", err)
		}
	}

	reportBase := filepath.Join(reportsDir, timestamp+".report")
	logFile := reportBase + ".log"
	mdFile := reportBase + ".md"
	jsonFile := reportBase + ".json"

	if err := os.WriteFile(logFile, []byte(res.Log), 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}
	if err := os.WriteFile(mdFile, []byte(res.Markdown), 0644); err != nil {
		return fmt.Errorf("failed to write md file: %w", err)
	}
	jsonBytes, err := json.MarshalIndent(res.Structured, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON report: %w", err)
	}
	if err := os.WriteFile(jsonFile, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	fmt.Printf("\n✅ Generation Complete!\n")
	fmt.Printf("📊 PASS: %d, FAIL: %d\n", res.Structured.Stats.Passed, res.Structured.Stats.Failed)
	fmt.Printf("📝 Log: %s\n", logFile)
	fmt.Printf("📝 Markdown: %s\n", mdFile)
	fmt.Printf("📊 JSON Report: %s\n", jsonFile)
	return nil
}
