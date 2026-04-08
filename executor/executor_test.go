package executor

import (
	"github.com/benedictjohannes/ComplianceProbe/playbook"
	"testing"
)

func TestCleanupOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ANSI and BEL",
			input:    "\x1b[31mError:\x1b[0m \u0007Something went wrong\n",
			expected: "Error: Something went wrong",
		},
		{
			name:     "Mixed Newlines and Tabs",
			input:    "Line 1\r\nLine 2\tTabbed",
			expected: "Line 1\r\nLine 2\tTabbed",
		},
		{
			name:     "Control Characters",
			input:    "Hello\u0001World",
			expected: "HelloWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanupOutput(tt.input)
			if got != tt.expected {
				t.Errorf("CleanupOutput() = %q; want %q", got, tt.expected)
			}
		})
	}
}

func TestPerformGather(t *testing.T) {
	tests := []struct {
		name     string
		spec     playbook.GatherSpec
		res      ExecutionResult
		expected string
	}{
		{
			name: "Regex Capture",
			spec: playbook.GatherSpec{
				Key:   "v",
				Regex: "v(\\d+)",
			},
			res:      ExecutionResult{Stdout: "Product v123"},
			expected: "123",
		},
		{
			name: "JS Function",
			spec: playbook.GatherSpec{
				Key:  "v",
				Func: "(stdout) => stdout.split(' ')[1].substring(1)",
			},
			res:      ExecutionResult{Stdout: "Product v123"},
			expected: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PerformGather(tt.spec, tt.res, make(map[string]interface{}))
			if err != nil {
				t.Fatalf("PerformGather() error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("PerformGather() = %q; want %q", got, tt.expected)
			}
		})
	}

	// Test IncludeStdErr in Gather
	isTrue := true
	spec := playbook.GatherSpec{Key: "err", Regex: "FATAL: (.*)", IncludeStdErr: &isTrue}
	res := ExecutionResult{Stdout: "", Stderr: "FATAL: system crash"}
	got, _ := PerformGather(spec, res, nil)
	if got != "system crash" {
		t.Errorf("PerformGather(stderr) = %q; want \"system crash\"", got)
	}

	// Test Regex with no capture group
	spec2 := playbook.GatherSpec{Key: "all", Regex: "Product v123"}
	res2 := ExecutionResult{Stdout: "Product v123"}
	got, _ = PerformGather(spec2, res2, nil)
	if got != "Product v123" {
		t.Errorf("PerformGather(no-group) = %q; want \"Product v123\"", got)
	}

	// Test No match
	spec3 := playbook.GatherSpec{Key: "none", Regex: "MISSING"}
	res3 := ExecutionResult{Stdout: "nothing here"}
	got, _ = PerformGather(spec3, res3, nil)
	if got != "" {
		t.Errorf("PerformGather(no-match) = %q; want \"\"", got)
	}

	// Test invalid regex
	spec4 := playbook.GatherSpec{Key: "err", Regex: "[["}
	_, err := PerformGather(spec4, res3, nil)
	if err == nil {
		t.Error("PerformGather should return error on invalid regex")
	}

	// Test JS error
	spec5 := playbook.GatherSpec{Key: "jserr", Func: "() => { throw new Error('ops') }"}
	_, err = PerformGather(spec5, res3, nil)
	if err == nil {
		t.Error("PerformGather should return error on JS error")
	}
}

func TestRunJS(t *testing.T) {
	context := map[string]interface{}{"foo": "bar"}
	code := "({ assertionContext }) => assertionContext.foo + 'baz'"
	got, err := RunJS(code, context)
	if err != nil {
		t.Fatalf("RunJS() error: %v", err)
	}
	if got != "barbaz" {
		t.Errorf("RunJS() = %q; want %q", got, "barbaz")
	}

	// Test direct code returning value
	got, err = RunJS("'hello ' + os", context)
	if err != nil {
		t.Fatalf("RunJS(direct) error: %v", err)
	}
	if got == "" {
		t.Error("RunJS(direct) should not be empty")
	}

	// Test returning empty/null
	got, err = RunJS("null", context)
	if err != nil || got != "" {
		t.Errorf("RunJS(null) = %q, %v; want \"\", nil", got, err)
	}

	// Test syntax error
	_, err = RunJS("this is not valid js", context)
	if err == nil {
		t.Error("RunJS should return error on syntax error")
	}
}

func TestRunExec(t *testing.T) {
	context := make(map[string]interface{})

	// 1. Simple Script
	e := &playbook.Exec{Script: "echo hello world"}
	res, err := RunExec(e, context)
	if err != nil {
		t.Fatalf("RunExec(simple) error: %v", err)
	}
	if res.Stdout != "hello world" {
		t.Errorf("RunExec(simple) stdout = %q; want \"hello world\"", res.Stdout)
	}

	// 2. JS Func generates script
	e2 := &playbook.Exec{Func: "() => 'echo js script'"}
	res, err = RunExec(e2, context)
	if err != nil {
		t.Fatalf("RunExec(js-func) error: %v", err)
	}
	if res.Stdout != "js script" {
		t.Errorf("RunExec(js-func) stdout = %q; want \"js script\"", res.Stdout)
	}

	// 3. Gathering in RunExec
	e3 := &playbook.Exec{
		Script: "echo 12345",
		Gather: []playbook.GatherSpec{
			{Key: "test_key", Regex: "(\\d+)"},
		},
	}
	res, err = RunExec(e3, context)
	if err != nil {
		t.Fatalf("RunExec(gather) error: %v", err)
	}
	if context["test_key"] != "12345" {
		t.Errorf("RunExec(gather) context[\"test_key\"] = %v; want \"12345\"", context["test_key"])
	}

	// 4. Empty script
	e4 := &playbook.Exec{Script: ""}
	res, err = RunExec(e4, context)
	if err != nil || !res.Success {
		t.Errorf("RunExec(empty) = %+v, %v; want Success: true, nil err", res, err)
	}

	// 5. JS error
	e5 := &playbook.Exec{Func: "() => { throw new Error('ops') }"}
	_, err = RunExec(e5, context)
	if err == nil {
		t.Error("RunExec should fail on JS error")
	}

	// 6. JS returns empty string
	e6 := &playbook.Exec{Func: "() => ''"}
	res, err = RunExec(e6, context)
	if err != nil || !res.Success {
		t.Errorf("RunExec(js-empty) = %+v, %v; want Success: true, nil err", res, err)
	}
}

func TestEvaluateRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     playbook.EvaluationRule
		res      ExecutionResult
		expected int
	}{
		{
			name:     "Regex Match",
			rule:     playbook.EvaluationRule{Regex: "PASS"},
			res:      ExecutionResult{Stdout: "Result: PASS"},
			expected: 1,
		},
		{
			name:     "Regex No Match",
			rule:     playbook.EvaluationRule{Regex: "PASS"},
			res:      ExecutionResult{Stdout: "Result: FAIL"},
			expected: -1,
		},
		{
			name:     "JS Function Pass",
			rule:     playbook.EvaluationRule{Func: "(stdout) => stdout.includes('OK') ? 1 : -1"},
			res:      ExecutionResult{Stdout: "Status is OK"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateRule(tt.rule, tt.res, make(map[string]interface{}))
			if err != nil {
				t.Fatalf("EvaluateRule() error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("EvaluateRule() = %d; want %d", got, tt.expected)
			}
		})
	}

	// Test IncludeStdErr
	rule := playbook.EvaluationRule{Regex: "ERROR"}
	res := ExecutionResult{Stdout: "", Stderr: "ERROR: something happened"}
	// Without IncludeStdErr, it shouldn't find it if Stdout is empty
	got, _ := EvaluateRule(rule, res, nil)
	if got == 1 {
		t.Error("EvaluateRule should not match stderr by default if stdout is empty")
	}

	isTrue := true
	rule.IncludeStdErr = &isTrue
	got, _ = EvaluateRule(rule, res, nil)
	if got != 1 {
		t.Error("EvaluateRule should match stderr when IncludeStdErr is true")
	}

	// Test invalid regex
	rule2 := playbook.EvaluationRule{Regex: "[["}
	_, err := EvaluateRule(rule2, res, nil)
	if err == nil {
		t.Error("EvaluateRule should return error on invalid regex")
	}

	// Test no rule
	rule3 := playbook.EvaluationRule{}
	got, _ = EvaluateRule(rule3, res, nil)
	if got != 0 {
		t.Errorf("EvaluateRule(no-rule) = %d; want 0", got)
	}

	// Test JS error
	rule4 := playbook.EvaluationRule{Func: "() => { throw new Error('ops') }"}
	_, err = EvaluateRule(rule4, res, nil)
	if err == nil {
		t.Error("EvaluateRule should return error on JS error")
	}
}

func TestRunShell(t *testing.T) {
	// Simple success
	res := RunShell("echo hello", "")
	if !res.Success || res.Stdout != "hello" {
		t.Errorf("RunShell(echo) = %+v; want Success: true, Stdout: hello", res)
	}

	// Failure
	res = RunShell("ls non_existent_file_12345", "")
	if res.Success || res.ExitCode == 0 {
		t.Errorf("RunShell(ls fail) = %+v; want Success: false, ExitCode != 0", res)
	}

	// Specific shell
	res = RunShell("echo hello", "sh")
	if !res.Success || res.Stdout != "hello" {
		t.Errorf("RunShell(sh) = %+v; want Success: true", res)
	}

	// Default shell (not sh or bash)
	res = RunShell("echo hello world", "bash") // wait bash IS a case
	res = RunShell("echo hello world", "echo")
	if !res.Success {
		t.Errorf("RunShell(default) = %+v; want Success: true", res)
	}

	// Non-existent shell (should trigger exit code -1)
	res = RunShell("echo hello", "/non/existent/shell/cp")
	if res.Success || res.ExitCode != -1 {
		t.Errorf("RunShell(missing) = %+v; want Success: false, ExitCode: -1", res)
	}
}
