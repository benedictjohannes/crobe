package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/benedictjohannes/crobe/playbook"

	"github.com/acarl005/stripansi"
	"github.com/dop251/goja"
)

type RunExecer func(e *playbook.Exec, context map[string]interface{}) (ExecutionResult, error)

type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Success  bool
}

func RunExec(e *playbook.Exec, context map[string]interface{}) (ExecutionResult, error) {
	script := e.Script

	// If Func is provided, it wins and generates the script
	if e.Func != "" {
		jsScript, err := RunJS(e.Func, context)
		if err != nil {
			return ExecutionResult{}, fmt.Errorf("JS error in Exec.Func: %v", err)
		}
		if jsScript == "" {
			return ExecutionResult{Success: true}, nil // Skip execution if JS returns empty
		}
		script = jsScript
	}
	e.Script = script

	if script == "" {
		return ExecutionResult{Success: true}, nil
	}

	res := RunShell(script, e.Shell, e.ScriptFileExtension)

	// Handle Gathering
	for _, g := range e.Gather {
		val, err := PerformGather(g, res, context)
		if err != nil {
			return res, fmt.Errorf("gather error for key %s: %v", g.Key, err)
		}
		context[g.Key] = val
	}

	return res, nil
}

func RunShell(command string, shell string, extension string) ExecutionResult {
	var name string
	var args []string

	tmpDir := os.TempDir()
	var tmpFile string

	if shell == "" {
		switch runtime.GOOS {
		case "windows":
			if _, err := exec.LookPath("pwsh"); err == nil {
				shell = "pwsh"
			} else {
				shell = "powershell"
			}
		case "darwin":
			shell = "zsh"
		default:
			shell = "bash"
		}
	}

	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	switch shell {
	case "!":
		// Direct execution: command is treated as name + args
		segments := strings.Fields(command)
		if len(segments) == 0 {
			return ExecutionResult{Success: true}
		}
		name = segments[0]
		args = segments[1:]
	case "powershell", "pwsh":
		name = shell
		ext := ".ps1"
		if extension != "" {
			ext = extension
		}
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cp_%d%s", time.Now().UnixNano(), ext))
		script := fmt.Sprintf("$ErrorActionPreference = 'Stop'\n%s\n", command)
		os.WriteFile(tmpFile, []byte(script), 0644)
		args = []string{"-ExecutionPolicy", "Bypass", "-File", tmpFile}
	case "bash", "sh", "zsh":
		name = shell
		base := filepath.Base(shell)
		
		ext := ".sh"
		if extension != "" {
			ext = extension
		} else if base == "zsh" {
			ext = ".zsh"
		}
		
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cp_%d%s", time.Now().UnixNano(), ext))
		var script string
		if base == "bash" || base == "zsh" {
			script = fmt.Sprintf("set -o pipefail\n%s\n", command)
		} else {
			script = fmt.Sprintf("%s\n", command)
		}
		os.WriteFile(tmpFile, []byte(script), 0755)
		args = []string{tmpFile}
	default:
		// Generic shell/interpreter: shell string is split into command + initial args,
		// and the script is appended as a temporary file.
		shellSegments := strings.Fields(shell)
		name = shellSegments[0]
		tmpFile = filepath.Join(tmpDir, fmt.Sprintf("cp_%d%s", time.Now().UnixNano(), extension))
		os.WriteFile(tmpFile, []byte(command), 0755)

		args = append(shellSegments[1:], tmpFile)
	}

	if tmpFile != "" {
		defer os.Remove(tmpFile)
	}

	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1", "LANG=en_US.UTF-8")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return ExecutionResult{
		Stdout:   CleanupOutput(stdout.String()),
		Stderr:   CleanupOutput(stderr.String()),
		ExitCode: exitCode,
		Success:  err == nil,
	}
}

func RunJS(code string, context map[string]interface{}) (string, error) {
	vm := goja.New()

	// Inject Context
	vm.Set("assertionContext", context)

	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "mac"
	}
	vm.Set("os", osName)
	vm.Set("arch", runtime.GOARCH)

	// Inject Env
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = pair[1]
		}
	}
	vm.Set("env", envMap)

	// Inject User/CWD
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	vm.Set("user", user)
	cwd, _ := os.Getwd()
	vm.Set("cwd", cwd)

	// Run code
	val, err := vm.RunString(code)
	if err != nil {
		return "", err
	}

	// Signature: ({ assertionContext, env, os, arch, user, cwd }) => string
	if fn, ok := goja.AssertFunction(val); ok {
		params := vm.NewObject()
		params.Set("assertionContext", context)
		params.Set("env", envMap)
		params.Set("os", osName)
		params.Set("arch", runtime.GOARCH)
		params.Set("user", user)
		params.Set("cwd", cwd)

		res, err := fn(goja.Undefined(), params)
		if err != nil {
			return "", err
		}
		return res.String(), nil
	}

	if goja.IsUndefined(val) || goja.IsNull(val) {
		return "", nil
	}

	return val.String(), nil
}

func PerformGather(g playbook.GatherSpec, res ExecutionResult, context map[string]interface{}) (string, error) {
	input := res.Stdout
	if g.GetIncludeStdErr() && input == "" {
		input = res.Stderr
	}

	// JS Function wins
	if g.Func != "" {
		vm := goja.New()
		vm.Set("stdout", res.Stdout)
		vm.Set("stderr", res.Stderr)
		vm.Set("assertionContext", context)

		val, err := vm.RunString(g.Func)
		if err != nil {
			return "", err
		}

		if fn, ok := goja.AssertFunction(val); ok {
			// Signature: (stdout, stderr, assertionContext) => string
			res, err := fn(goja.Undefined(), vm.ToValue(res.Stdout), vm.ToValue(res.Stderr), vm.ToValue(context))
			if err != nil {
				return "", err
			}
			return res.String(), nil
		}

		return val.String(), nil
	}

	// Regex
	if g.Regex != "" {
		re, err := regexp.Compile(g.Regex)
		if err != nil {
			return "", err
		}
		matches := re.FindStringSubmatch(input)
		if len(matches) > 1 {
			return matches[1], nil
		} else if len(matches) == 1 {
			return matches[0], nil
		}
	}

	return "", nil
}

func EvaluateRule(rule playbook.EvaluationRule, res ExecutionResult, context map[string]interface{}) (int, error) {
	input := res.Stdout
	if rule.GetIncludeStdErr() && input == "" {
		input = res.Stderr
	}

	if rule.Func != "" {
		vm := goja.New()
		vm.Set("stdout", res.Stdout)
		vm.Set("stderr", res.Stderr)
		vm.Set("assertionContext", context)

		val, err := vm.RunString(rule.Func)
		if err != nil {
			return 0, err
		}

		if fn, ok := goja.AssertFunction(val); ok {
			// Signature: (stdout, stderr, assertionContext) => -1 | 0 | 1
			out, err := fn(goja.Undefined(), vm.ToValue(res.Stdout), vm.ToValue(res.Stderr), vm.ToValue(context))
			if err != nil {
				return 0, err
			}
			return int(out.ToInteger()), nil
		}

		return int(val.ToInteger()), nil
	}

	if rule.Regex != "" {
		re, err := regexp.Compile(rule.Regex)
		if err != nil {
			return 0, err
		}
		if re.MatchString(input) {
			return 1, nil
		}
		return -1, nil
	}

	return 0, nil
}

func CleanupOutput(input string) string {
	// 1. Use stripansi to handle cross-platform ANSI/CSI/OSC sequences robustly
	output := stripansi.Strip(input)

	// 2. Explicitly remove BEL and other non-printable control chars
	output = strings.Map(func(r rune) rune {
		if r == '\u0007' || (r < 32 && r != '\n' && r != '\r' && r != '\t') {
			return -1 // Drop the character
		}
		return r
	}, output)

	return strings.TrimSpace(output)
}
