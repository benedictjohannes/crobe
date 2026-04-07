package report

import (
	"compliance-probe/executor"
	"compliance-probe/playbook"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestReporterScoring(t *testing.T) {
	config := playbook.ReportConfig{
		Title: "Test Report",
		Sections: []playbook.Section{
			{
				Title:       "Test Section",
				Description: []string{"Desc"},
				Assertions: []playbook.Assertion{
					{
						Code:            "TEST_01",
						Title:           "Test Assertion",
						Description:     "Test Description",
						MinPassingScore: func(i int) *int { return &i }(2),
						Cmds: []playbook.Cmd{
							{
								Exec:      playbook.Exec{Script: "echo 1"},
								PassScore: func(i int) *int { return &i }(1),
							},
							{
								Exec:      playbook.Exec{Script: "echo 2"},
								PassScore: func(i int) *int { return &i }(1),
							},
						},
					},
				},
			},
		},
	}

	// Mock execution: first succeeds, second fails
	callIdx := 0
	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		callIdx++
		if callIdx == 1 {
			return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "ok"}, nil
		}
		return executor.ExecutionResult{ExitCode: 1, Success: false, Stdout: "fail"}, nil
	}
	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured

	// Score: 1 (cmd1 pass) + -1 (cmd2 fail default) = 0
	// MinScore: 2
	// Expect: Passed = false
	ass := report.Assertions["TEST_01"]
	if ass.Passed {
		t.Errorf("Assertion passed with score %d; expected fail (min 2)", ass.Score)
	}
	if ass.Score != 0 {
		t.Errorf("Assertion score = %d; want 0", ass.Score)
	}

	// Now try a passing case
	mockExecPass := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
	}
	runExec = mockExecPass
	res2 := GenerateReport(config)
	report2 := res2.Structured
	if !report2.Assertions["TEST_01"].Passed {
		t.Errorf("Assertion failed with score %d; expected pass (min 2)", report2.Assertions["TEST_01"].Score)
	}
}

func TestExcludeFromReport(t *testing.T) {
	config := playbook.ReportConfig{
		Title: "Test Exclude",
		Sections: []playbook.Section{
			{
				Title: "Section 1",
				Assertions: []playbook.Assertion{
					{
						Code:  "EXCL_01",
						Title: "Exclusion Test",
						Cmds: []playbook.Cmd{
							{
								Exec: playbook.Exec{
									Script:            "echo sensitive_data",
									ExcludeFromReport: true,
									Gather: []playbook.GatherSpec{
										{
											Key:               "sensitive",
											Regex:             "(.*)",
											ExcludeFromReport: true,
										},
										{
											Key:               "public",
											Regex:             "(.*)",
											ExcludeFromReport: false,
										},
									},
								},
							},
						},
						MinPassingScore: func(i int) *int { return &i }(1),
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		out := "sensitive_data"
		res := executor.ExecutionResult{Stdout: out, Success: true, ExitCode: 0}
		for _, g := range e.Gather {
			context[g.Key] = out
		}
		return res, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	md := res.Markdown
	logStr := res.Log

	ass1 := report.Assertions["EXCL_01"]
	if _, exists := ass1.Context["sensitive"]; exists {
		t.Errorf("expected 'sensitive' key to be excluded from report context")
	}
	if _, exists := ass1.Context["public"]; !exists {
		t.Errorf("expected 'public' key to be included in report context")
	}
	if strings.Contains(md, "sensitive_data") {
		t.Errorf("expected sensitive command output to be excluded from markdown report")
	}
	if !strings.Contains(logStr, "[REDACTED]") {
		t.Errorf("expected [REDACTED] to be present in log for STDOUT")
	}
}

func TestGenerateReport_Advanced(t *testing.T) {
	minScore := 1
	config := playbook.ReportConfig{
		Title: "Advanced Test",
		ReportFrontmatter: map[string]interface{}{
			"custom": "value",
		},
		Sections: []playbook.Section{
			{
				Title: "Advanced Section",
				Assertions: []playbook.Assertion{
					{
						Code:            "ADV_01",
						Title:           "Advanced Assertion",
						MinPassingScore: &minScore,
						PreCmds: []playbook.Exec{
							{Script: "pre-cmd", Gather: []playbook.GatherSpec{{Key: "pre", Regex: "(.*)"}}},
						},
						Cmds: []playbook.Cmd{
							{
								Exec: playbook.Exec{Script: "main-cmd"},
								ExitCodeRules: []playbook.ExitCodeRule{
									{Min: func(i int) *int { return &i }(0), Max: func(i int) *int { return &i }(0), Result: 1},
									{Min: func(i int) *int { return &i }(1), Max: func(i int) *int { return &i }(10), Result: -1},
								},
								StdOutRule: playbook.EvaluationRule{Regex: "SUCCESS"},
							},
						},
						PostCmds: []playbook.Exec{
							{Script: "post-cmd"},
						},
						PassDescription: "Passed!",
						FailDescription: "Failed!",
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "pre-cmd" {
			context["pre"] = "pre-val"
			return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
		}
		if e.Script == "main-cmd" {
			return executor.ExecutionResult{ExitCode: 0, Success: true, Stdout: "SUCCESS"}, nil
		}
		return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	md := res.Markdown

	ass := report.Assertions["ADV_01"]
	// main-cmd: ExitCode 0 -> Result 1 (via rule). Score += PassScore (default 1? wait)
	// Actually, if PassScore is nil, it uses 1?
	// Let's check what Cmd.GetPassScore returns.

	if !strings.Contains(md, "custom: value") {
		t.Errorf("Markdown frontmatter missing custom value")
	}
	if !strings.Contains(md, "Passed!") {
		t.Errorf("Markdown missing pass description")
	}
	if ass.Context["pre"] != "pre-val" {
		t.Errorf("Context missing pre-command value: %v", ass.Context)
	}
}

func TestGenerateReport_ErrorCases(t *testing.T) {
	config := playbook.ReportConfig{
		Title: "Error Test",
		Sections: []playbook.Section{
			{
				Title: "Error Section",
				Assertions: []playbook.Assertion{
					{
						Code:  "ERR_01",
						Title: "Error Assertion",
						PreCmds: []playbook.Exec{
							{Script: "pre-fail"},
						},
						Cmds: []playbook.Cmd{
							{Exec: playbook.Exec{Script: "main-fail"}},
						},
						PostCmds: []playbook.Exec{
							{Script: "post-fail"},
						},
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "pre-fail" || e.Script == "post-fail" {
			return executor.ExecutionResult{}, fmt.Errorf("error in command")
		}
		if e.Script == "main-fail" {
			return executor.ExecutionResult{}, fmt.Errorf("main error")
		}
		return executor.ExecutionResult{Success: true}, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	ass := report.Assertions["ERR_01"]
	// Main fail will have FailScore (-1 default)
	// pre/post fail just print warnings but don't affect score directly in GenerateReport logic as it stands.
	if ass.Score != -1 {
		t.Errorf("Score = %d; want -1 (from main error)", ass.Score)
	}
}

func TestGenerateReport_EnvUsage(t *testing.T) {
	// Cover USER vs USERNAME
	os.Setenv("USER", "")
	os.Setenv("USERNAME", "testuser")
	defer os.Unsetenv("USERNAME")

	config := playbook.ReportConfig{
		Title: "Env Test",
		Sections: []playbook.Section{
			{
				Title: "S1",
				Assertions: []playbook.Assertion{
					{
						Code: "E_01",
						Cmds: []playbook.Cmd{{Exec: playbook.Exec{Script: "echo 1"}}},
					},
				},
			},
		},
	}
	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		return executor.ExecutionResult{Stdout: "ok", Stderr: "some error"}, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	if report.Username != "testuser" {
		t.Errorf("Username = %s; want testuser", report.Username)
	}
}

func TestGenerateReport_DefaultExitCode(t *testing.T) {
	config := playbook.ReportConfig{
		Title: "Default Exit Code",
		Sections: []playbook.Section{
			{
				Title: "S1",
				// Two assertions: one success, one fail by exit code
				Assertions: []playbook.Assertion{
					{
						Code:  "E_PASS",
						Title: "Should Pass Assertion",
						Cmds:  []playbook.Cmd{{Exec: playbook.Exec{Script: "ok"}}},
					},
					{
						Code:  "E_FAIL",
						Title: "Should Fail Assertion",
						Cmds:  []playbook.Cmd{{Exec: playbook.Exec{Script: "fail"}}},
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "ok" {
			return executor.ExecutionResult{ExitCode: 0, Success: true}, nil
		}
		return executor.ExecutionResult{ExitCode: 1, Success: false}, nil
	}

	runExec = mockExec
	res := GenerateReport(config)
	report := res.Structured
	if !report.Assertions["E_PASS"].Passed {
		t.Errorf("E_PASS should have passed")
	}
	if report.Assertions["E_FAIL"].Passed {
		t.Errorf("E_FAIL should have failed")
	}
}

func TestLogExecution_Extended(t *testing.T) {
	var log strings.Builder

	// Multiline script
	execMultiline := playbook.Exec{Script: "line1\nline2"}
	res := executor.ExecutionResult{Stdout: "out\n", Stderr: "err\n", ExitCode: 0}
	logExecution(&log, execMultiline, res, nil)

	logStr := log.String()
	if !strings.Contains(logStr, "SCRIPT") || !strings.Contains(logStr, "<<<<< END SCRIPT") {
		t.Errorf("Multiline script logging failed")
	}

	// Error case
	log.Reset()
	execErr := playbook.Exec{Script: "fail"}
	logExecution(&log, execErr, res, fmt.Errorf("some error"))
	logStr = log.String()
	if !strings.Contains(logStr, "ERROR: some error") {
		t.Errorf("Error logging failed")
	}

	// Redacted case
	log.Reset()
	execRedacted := playbook.Exec{Script: "secret", ExcludeFromReport: true}
	logExecution(&log, execRedacted, res, nil)
	logStr = log.String()
	if !strings.Contains(logStr, "[REDACTED]") {
		t.Errorf("Redaction in logging failed")
	}
}
