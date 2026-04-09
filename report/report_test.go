package report

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
)

func TestGenerateReport_Formatting(t *testing.T) {
	trace := executor.ExecutionTrace{
		Playbook: playbook.Playbook{
			Title: "Test Report",
			ReportFrontmatter: map[string]interface{}{
				"custom": "value",
			},
		},
		Username: "testuser",
		OS:       "mac",
		Arch:     "arm64",
		Sections: []executor.SectionContext{
			{
				PlaybookSection: playbook.Section{
					Title:       "Test Section",
					Description: []string{"Desc line 1", "Desc line 2"},
				},
				Assertions: []executor.AssertionContext{
					{
						PlaybookAssertion: playbook.Assertion{
							Code:            "TEST_01",
							Title:           "Test Assertion",
							Description:     "Test Description",
							PassDescription: "It passed!",
							FailDescription: "It failed!",
						},
						Passed:   true,
						Score:    2,
						MinScore: 1,
						Context: map[string]interface{}{
							"key": "val",
						},
						CmdLogs: []executor.CommandLog{
							{
								Exec:   playbook.Exec{Script: "echo 1"},
								Result: executor.ExecutionResult{Stdout: "output text", ExitCode: 0},
							},
							{
								Exec:   playbook.Exec{Script: "secret", ExcludeFromReport: true},
								Result: executor.ExecutionResult{Stdout: "secret text", ExitCode: 0},
							},
						},
						Outputs: []string{"# --- STDOUT ---", "output text", "[REDACTED]"},
					},
					{
						PlaybookAssertion: playbook.Assertion{
							Code:            "TEST_FAIL",
							Title:           "Fail Assertion",
							FailDescription: "It failed!",
						},
						Passed: false,
						Score:  0,
						CmdLogs: []executor.CommandLog{
							{
								Exec:   playbook.Exec{Script: "failcmd"},
								Result: executor.ExecutionResult{Stderr: "error text", ExitCode: 1},
							},
						},
						Outputs: []string{"# --- STDERR ---", "error text"},
					},
				},
			},
		},
		TotalPassed: 1,
		TotalFailed: 1,
	}

	tsStart, _ := time.Parse(time.RFC3339, "2026-04-10T12:00:00Z")
	tsEnd, _ := time.Parse(time.RFC3339, "2026-04-10T12:00:10Z")
	trace.Timestamps.Start = tsStart
	trace.Timestamps.End = tsEnd

	res := GenerateReport(trace)

	// JSON Checks
	report := res.Structured
	if report.OS != "mac" {
		t.Errorf("expected OS mac, got %s", report.OS)
	}
	if report.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", report.Username)
	}
	if len(report.Assertions) != 2 {
		t.Errorf("expected 2 assertions, got %d", len(report.Assertions))
	}
	ass1 := report.Assertions["TEST_01"]
	if ass1.Context["key"] != "val" {
		t.Errorf("expected context key to be val, got %v", ass1.Context["key"])
	}
	if !report.Timestamps.Start.Equal(tsStart) {
		t.Errorf("expected start timestamp %v, got %v", tsStart, report.Timestamps.Start)
	}
	if !report.Timestamps.End.Equal(tsEnd) {
		t.Errorf("expected end timestamp %v, got %v", tsEnd, report.Timestamps.End)
	}

	// Markdown Checks
	md := res.Markdown
	if !strings.Contains(md, "custom: value") {
		t.Errorf("expected custom frontmatter, not found")
	}
	if !strings.Contains(md, "Desc line 1") {
		t.Errorf("expected section description, not found")
	}
	if !strings.Contains(md, "✅ **Pass:** It passed!") {
		t.Errorf("expected pass description, not found")
	}
	if !strings.Contains(md, "❌ **Fail:** It failed!") {
		t.Errorf("expected fail description, not found")
	}
	if !strings.Contains(md, "output text") {
		t.Errorf("expected evidence output text, not found")
	}
	if strings.Contains(md, "secret text") {
		t.Errorf("secret text should not be in markdown")
	}

	// Log Checks
	logStr := res.Log
	if !strings.Contains(logStr, ">>>>> COMMAND: echo 1 <<<<<") {
		t.Errorf("expected command log, not found")
	}
	if !strings.Contains(logStr, "[REDACTED]") {
		t.Errorf("expected redacted command output, not found")
	}
}

func TestLogExecution_Extended(t *testing.T) {
	var log strings.Builder

	// Multiline script
	execMultiline := playbook.Exec{Script: "line1\nline2"}
	res := executor.ExecutionResult{Stdout: "out\n", Stderr: "err\n", ExitCode: 0}
	writeExecutionLog(&log, execMultiline, res, nil)

	logStr := log.String()
	if !strings.Contains(logStr, "SCRIPT") || !strings.Contains(logStr, "<<<<< END SCRIPT") {
		t.Errorf("Multiline script logging failed")
	}

	// Error case
	log.Reset()
	execErr := playbook.Exec{Script: "fail"}
	writeExecutionLog(&log, execErr, res, fmt.Errorf("some error"))
	logStr = log.String()
	if !strings.Contains(logStr, "ERROR: some error") {
		t.Errorf("Error logging failed")
	}

	// Redacted case
	log.Reset()
	execRedacted := playbook.Exec{Script: "secret", ExcludeFromReport: true}
	writeExecutionLog(&log, execRedacted, res, nil)
	logStr = log.String()
	if !strings.Contains(logStr, "[REDACTED]") {
		t.Errorf("Redaction in logging failed")
	}
}

func TestIsEvidenceMaterial(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{" ", false},
		{"\t", false},
		{"\n", false},
		{"\r", false},
		{" \t\n\r ", false},
		{"a", true},
		{" a ", true},
		{"[REDACTED]", false},
		{"# --- STDOUT ---", false},
		{"# --- STDERR ---", false},
	}

	for _, tt := range tests {
		if got := isEvidenceMaterial(tt.input); got != tt.expected {
			t.Errorf("isEvidenceMaterial(%q) = %v; want %v", tt.input, got, tt.expected)
		}
	}
}
