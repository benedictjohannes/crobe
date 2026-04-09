package director

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
)

var (
	runExec = executor.RunExec
	goos    = runtime.GOOS
)

func Run(config playbook.Playbook) executor.ExecutionTrace {
	now := time.Now()

	osName := goos
	if osName == "darwin" {
		osName = "mac"
	}

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	trace := executor.ExecutionTrace{
		Playbook: config,
		Username: username,
		OS:       osName,
		Arch:     runtime.GOARCH,
	}
	trace.Timestamps.Start = now

	totalPassed := 0
	totalFailed := 0

	for _, section := range config.Sections {
		fmt.Printf("  Processing Section: %s\n", section.Title)
		sectionCtx := executor.SectionContext{
			PlaybookSection: section,
		}

		for _, assertion := range section.Assertions {
			start := time.Now()
			context := make(map[string]interface{})
			score := 0

			assCtx := executor.AssertionContext{
				PlaybookAssertion: assertion,
				Context:           make(map[string]interface{}),
			}

			// 1. Pre-Commands
			for _, exec := range assertion.PreCmds {
				res, err := runExec(&exec, context)
				assCtx.PreCmdLogs = append(assCtx.PreCmdLogs, executor.CommandLog{
					Exec:   exec,
					Result: res,
					Err:    err,
				})
				if err != nil {
					fmt.Printf("      ⚠️ PreCmd Error (%s): %v\n", assertion.Code, err)
				}
			}

			// 2. Main Commands
			var outputs []string
			for _, cmd := range assertion.Cmds {
				res, err := runExec(&cmd.Exec, context)
				assCtx.CmdLogs = append(assCtx.CmdLogs, executor.CommandLog{
					Exec:   cmd.Exec,
					Result: res,
					Err:    err,
				})

				if err != nil {
					score += cmd.GetFailScore()
					continue
				}

				if cmd.Exec.ExcludeFromReport {
					outputs = append(outputs, "[REDACTED]")
				} else {
					if res.Stderr != "" {
						if len(res.Stdout) > 0 {
							outputs = append(outputs, "# --- STDOUT ---")
							outputs = append(outputs, res.Stdout)
						}
						outputs = append(outputs, "# --- STDERR ---")
						outputs = append(outputs, res.Stderr)
					} else {
						outputs = append(outputs, res.Stdout)
					}
				}

				// Evaluation
				result := 0
				foundRule := false
				for _, rule := range cmd.ExitCodeRules {
					match := true
					if rule.Min != nil && res.ExitCode < *rule.Min {
						match = false
					}
					if rule.Max != nil && res.ExitCode > *rule.Max {
						match = false
					}
					if match {
						result = rule.Result
						foundRule = true
						break
					}
				}

				if !foundRule {
					if res.ExitCode == 0 {
						result = 1
					} else {
						result = -1
					}
				}

				if cmd.StdOutRule.Regex != "" || cmd.StdOutRule.Func != "" {
					verdict, _ := executor.EvaluateRule(cmd.StdOutRule, res, context)
					if verdict != 0 {
						result = verdict
					}
				}
				if cmd.StdErrRule.Regex != "" || cmd.StdErrRule.Func != "" {
					verdict, _ := executor.EvaluateRule(cmd.StdErrRule, res, context)
					if verdict != 0 {
						result = verdict
					}
				}

				switch result {
				case 1:
					score += cmd.GetPassScore()
				case -1:
					score += cmd.GetFailScore()
				}
			}
			assCtx.Outputs = outputs

			// 3. Post-Commands
			for _, exec := range assertion.PostCmds {
				res, err := runExec(&exec, context)
				assCtx.PostCmdLogs = append(assCtx.PostCmdLogs, executor.CommandLog{
					Exec:   exec,
					Result: res,
					Err:    err,
				})
				if err != nil {
					fmt.Printf("      ⚠️ PostCmd Error (%s): %v\n", assertion.Code, err)
				}
			}

			passed := score >= assertion.GetMinPassingScore()
			if passed {
				totalPassed++
			} else {
				totalFailed++
			}

			assCtx.Passed = passed
			assCtx.Score = score
			assCtx.MinScore = assertion.GetMinPassingScore()
			assCtx.Timestamps.Start = start
			assCtx.Timestamps.End = time.Now()

			// Determine which keys to exclude from report
			excludedKeys := make(map[string]bool)
			for _, exec := range assertion.PreCmds {
				for _, g := range exec.Gather {
					if g.ExcludeFromReport {
						excludedKeys[g.Key] = true
					}
				}
			}
			for _, cmd := range assertion.Cmds {
				for _, g := range cmd.Exec.Gather {
					if g.ExcludeFromReport {
						excludedKeys[g.Key] = true
					}
				}
			}
			for _, exec := range assertion.PostCmds {
				for _, g := range exec.Gather {
					if g.ExcludeFromReport {
						excludedKeys[g.Key] = true
					}
				}
			}

			for k, v := range context {
				if !excludedKeys[k] {
					assCtx.Context[k] = v
				}
			}

			status := "✅ PASS"
			if !passed {
				status = "❌ FAIL"
			}
			fmt.Printf("    - %s: %s (Score: %d/%d)\n", assertion.Title, status, score, assertion.GetMinPassingScore())

			sectionCtx.Assertions = append(sectionCtx.Assertions, assCtx)
		}
		trace.Sections = append(trace.Sections, sectionCtx)
	}

	trace.Timestamps.End = time.Now()
	trace.TotalPassed = totalPassed
	trace.TotalFailed = totalFailed

	return trace
}
