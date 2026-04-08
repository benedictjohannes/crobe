package report

import (
	"github.com/benedictjohannes/ComplianceProbe/executor"
	"github.com/benedictjohannes/ComplianceProbe/playbook"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var runExec executor.RunExecer

func init() {
	runExec = executor.RunExec
}

type Assertion struct {
	Timestamps struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"timestamps"`
	Passed   bool                   `json:"passed"`
	Score    int                    `json:"score"`
	MinScore int                    `json:"minScore"`
	Context  map[string]interface{} `json:"context"`
}

type Stats struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

type FinalReport struct {
	Timestamps struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"timestamps"`
	Username   string               `json:"username"`
	OS         string               `json:"os"`
	Arch       string               `json:"arch"`
	Assertions map[string]Assertion `json:"assertions"`
	Stats      Stats                `json:"stats"`
}

type FinalResult struct {
	Structured FinalReport
	Log        string
	Markdown   string
}

func GenerateReport(config playbook.ReportConfig) FinalResult {
	now := time.Now()
	var md strings.Builder
	var log strings.Builder

	log.WriteString(fmt.Sprintf(">>>>>>>>>>>> REPORT LOG: %s <<<<<<<<<<<<\n\n", now.Format("060102-150405")))
	if config.ReportFrontmatter == nil {
		config.ReportFrontmatter = make(map[string]interface{})
	}
	if _, ok := config.ReportFrontmatter["title"]; !ok {
		config.ReportFrontmatter["title"] = config.Title
	}
	if _, ok := config.ReportFrontmatter["date"]; !ok {
		config.ReportFrontmatter["date"] = now.Format("2006-01-02")
	}

	fmBytes, _ := yaml.Marshal(config.ReportFrontmatter)
	md.WriteString("---\n")
	md.Write(fmBytes)
	md.WriteString("---\n\n")

	md.WriteString(fmt.Sprintf("# %s\n\nGenerated on: %s\n\n---\n\n", config.Title, now.Format("2006-01-02 15:04:05")))

	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "mac"
	}

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	finalReport := FinalReport{
		Username:   username,
		OS:         osName,
		Arch:       runtime.GOARCH,
		Assertions: make(map[string]Assertion),
	}
	finalReport.Timestamps.Start = now.Format(time.RFC3339)

	totalPassed := 0
	totalFailed := 0

	for _, section := range config.Sections {
		fmt.Printf("  Processing Section: %s\n", section.Title)
		md.WriteString(fmt.Sprintf("## %s\n\n", section.Title))
		log.WriteString(fmt.Sprintf(">>>>>>>>> SECTION: %s <<<<<<<<<\n\n", section.Title))
		for _, desc := range section.Description {
			md.WriteString(fmt.Sprintf("%s  \n", desc))
		}
		md.WriteString("\n")

		for _, assertion := range section.Assertions {
			start := time.Now()
			context := make(map[string]interface{})
			score := 0

			log.WriteString(fmt.Sprintf(">>>>>>> ASSERTION: %s <<<<<<<\n\n", assertion.Title))

			// 1. Pre-Commands
			for _, exec := range assertion.PreCmds {
				if _, err := runExec(&exec, context); err != nil {
					fmt.Printf("      ⚠️ PreCmd Error (%s): %v\n", assertion.Code, err)
				}
			}

			// 2. Main Commands
			var outputs []string
			for _, cmd := range assertion.Cmds {
				res, err := runExec(&cmd.Exec, context)
				logExecution(&log, cmd.Exec, res, err)

				if err != nil {
					log.WriteString(fmt.Sprintf(">>>>> Error executing command: %v <<<<<\n\n", err))
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

			// 3. Post-Commands
			for _, exec := range assertion.PostCmds {
				if _, err := runExec(&exec, context); err != nil {
					fmt.Printf("      ⚠️ PostCmd Error (%s): %v\n", assertion.Code, err)
				}
			}

			passed := score >= assertion.GetMinPassingScore()
			if passed {
				totalPassed++
			} else {
				totalFailed++
			}

			report := Assertion{
				Passed:   passed,
				Score:    score,
				MinScore: assertion.GetMinPassingScore(),
				Context:  make(map[string]interface{}),
			}
			report.Timestamps.Start = start.Format(time.RFC3339)
			report.Timestamps.End = time.Now().Format(time.RFC3339)

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
					report.Context[k] = v
				}
			}
			finalReport.Assertions[assertion.Code] = report

			status := "✅ PASS"
			if !passed {
				status = "❌ FAIL"
			}
			fmt.Printf("    - %s: %s (Score: %d/%d)\n", assertion.Title, status, score, assertion.GetMinPassingScore())

			md.WriteString(fmt.Sprintf("### %s\n\n", assertion.Title))
			md.WriteString(fmt.Sprintf("%s\n\n", assertion.Description))
			shouldSkipEvidence := true
			for _, o := range outputs {
				if isEvidenceMaterial(o) {
					shouldSkipEvidence = false
					break
				}
			}
			if !shouldSkipEvidence {
				md.WriteString("**Evidence:**\n")
				md.WriteString("```\n")
				md.WriteString(strings.Join(outputs, "\n") + "\n")
				md.WriteString("```\n\n")
			}

			if passed {
				if assertion.PassDescription != "" {
					md.WriteString(fmt.Sprintf("> ✅ **Pass:** %s\n\n", assertion.PassDescription))
				}
			} else {
				if assertion.FailDescription != "" {
					md.WriteString(fmt.Sprintf("> ❌ **Fail:** %s\n\n", assertion.FailDescription))
				}
			}
		}
		md.WriteString("---\n\n")
	}

	finalReport.Timestamps.End = time.Now().Format(time.RFC3339)
	finalReport.Stats.Passed = totalPassed
	finalReport.Stats.Failed = totalFailed

	return FinalResult{
		Structured: finalReport,
		Markdown:   md.String(),
		Log:        log.String(),
	}
}

func isEvidenceMaterial(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	if s == "[REDACTED]" || s == "# --- STDOUT ---" || s == "# --- STDERR ---" {
		return false
	}
	return true
}

func logExecution(log *strings.Builder, exec playbook.Exec, res executor.ExecutionResult, err error) {
	exclude := exec.ExcludeFromReport
	script := strings.TrimSpace(exec.Script)
	cmdTitle := "COMMAND"
	isMultiline := len(strings.Split(script, "\n")) > 1
	if isMultiline {
		cmdTitle = "SCRIPT"
	} else {
		cmdTitle += ": " + script
	}

	log.WriteString(fmt.Sprintf(">>>>> %s <<<<<\n", cmdTitle))
	if isMultiline {
		log.WriteString(script)
		if !strings.HasSuffix(script, "\n") {
			log.WriteString("\n")
		}
		log.WriteString("<<<<< END SCRIPT")
		log.WriteString("\n")
	} else {
		log.WriteString("\n")
	}

	if err != nil {
		log.WriteString(fmt.Sprintf(">>> ERROR: %v <<<\n", err))
	}

	if res.Stdout != "" {
		log.WriteString(">>> STDOUT <<<\n")
		if exclude {
			log.WriteString("[REDACTED]\n")
		} else {
			log.WriteString(res.Stdout)
			if !strings.HasSuffix(res.Stdout, "\n") {
				log.WriteString("\n")
			}
		}
	}

	if res.Stderr != "" {
		log.WriteString(">>> STDERR <<<\n")
		if exclude {
			log.WriteString("[REDACTED]\n")
		} else {
			log.WriteString(res.Stderr)
			if !strings.HasSuffix(res.Stderr, "\n") {
				log.WriteString("\n")
			}
		}
	}
	log.WriteString("\n")
}
