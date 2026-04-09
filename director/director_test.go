package director

import (
	"fmt"
	"os"
	"testing"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
)

func TestDirectorScoring(t *testing.T) {
	config := playbook.Playbook{
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
	trace := Run(config)
	
	ass := trace.Sections[0].Assertions[0]
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
	trace2 := Run(config)
	if !trace2.Sections[0].Assertions[0].Passed {
		t.Errorf("Assertion failed with score %d; expected pass (min 2)", trace2.Sections[0].Assertions[0].Score)
	}
}

func TestExcludeFromReport(t *testing.T) {
	config := playbook.Playbook{
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
	trace := Run(config)

	ass := trace.Sections[0].Assertions[0]
	if _, exists := ass.Context["sensitive"]; exists {
		t.Errorf("expected 'sensitive' key to be excluded from report context")
	}
	if _, exists := ass.Context["public"]; !exists {
		t.Errorf("expected 'public' key to be included in report context")
	}
}

func TestDirector_Advanced(t *testing.T) {
	minScore := 1
	config := playbook.Playbook{
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
	trace := Run(config)
	
	ass := trace.Sections[0].Assertions[0]
	if ass.Context["pre"] != "pre-val" {
		t.Errorf("Context missing pre-command value: %v", ass.Context)
	}
}

func TestDirector_ErrorCases(t *testing.T) {
	config := playbook.Playbook{
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
	trace := Run(config)
	ass := trace.Sections[0].Assertions[0]
	if ass.Score != -1 {
		t.Errorf("Score = %d; want -1 (from main error)", ass.Score)
	}
}

func TestDirector_EnvUsage(t *testing.T) {
	os.Setenv("USER", "")
	os.Setenv("USERNAME", "testuser")
	defer os.Unsetenv("USERNAME")

	config := playbook.Playbook{
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
	trace := Run(config)
	
	if trace.Username != "testuser" {
		t.Errorf("Username = %s; want testuser", trace.Username)
	}
}

func TestDirector_DefaultExitCode(t *testing.T) {
	config := playbook.Playbook{
		Title: "Default Exit Code",
		Sections: []playbook.Section{
			{
				Title: "S1",
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
	trace := Run(config)
	
	if !trace.Sections[0].Assertions[0].Passed {
		t.Errorf("E_PASS should have passed")
	}
	if trace.Sections[0].Assertions[1].Passed {
		t.Errorf("E_FAIL should have failed")
	}
}

func TestDirector_CoverageBoost(t *testing.T) {
	oldOS := goos
	goos = "darwin"
	defer func() { goos = oldOS }()

	config := playbook.Playbook{
		Title: "Boost Test",
		Sections: []playbook.Section{
			{
				Title: "S1",
				Assertions: []playbook.Assertion{
					{
						Code: "B_01",
						PreCmds: []playbook.Exec{
							{
								Script: "pre-gather",
								Gather: []playbook.GatherSpec{
									{Key: "pre_secret", Regex: "(.*)", ExcludeFromReport: true},
								},
							},
						},
						Cmds: []playbook.Cmd{
							{
								Exec: playbook.Exec{Script: "cmd-mixed"},
								StdErrRule: playbook.EvaluationRule{Regex: "ERROR_MATCH"},
							},
							{
								Exec: playbook.Exec{Script: "cmd-stderr-only"},
							},
						},
						PostCmds: []playbook.Exec{
							{
								Script: "post-gather",
								Gather: []playbook.GatherSpec{
									{Key: "post_secret", Regex: "(.*)", ExcludeFromReport: true},
								},
							},
						},
					},
				},
			},
		},
	}

	mockExec := func(e *playbook.Exec, context map[string]interface{}) (executor.ExecutionResult, error) {
		if e.Script == "pre-gather" {
			context["pre_secret"] = "pre-secret-val"
			return executor.ExecutionResult{Success: true}, nil
		}
		if e.Script == "cmd-mixed" {
			return executor.ExecutionResult{
				Stdout:   "some stdout",
				Stderr:   "ERROR_MATCH",
				ExitCode: 0,
				Success:  true,
			}, nil
		}
		if e.Script == "cmd-stderr-only" {
			return executor.ExecutionResult{
				Stdout:   "",
				Stderr:   "just stderr",
				ExitCode: 0,
				Success:  true,
			}, nil
		}
		if e.Script == "post-gather" {
			context["post_secret"] = "secret-val"
			return executor.ExecutionResult{Success: true}, nil
		}
		return executor.ExecutionResult{Success: true}, nil
	}

	runExec = mockExec
	trace := Run(config)
	
	if trace.OS != "mac" {
		t.Errorf("OS = %s; want mac", trace.OS)
	}

	ass := trace.Sections[0].Assertions[0]
	if _, exists := ass.Context["pre_secret"]; exists {
		t.Errorf("pre_secret should be excluded from report context")
	}
	if _, exists := ass.Context["post_secret"]; exists {
		t.Errorf("post_secret should be excluded from report context")
	}
}
