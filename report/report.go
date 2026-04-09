package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"

	"gopkg.in/yaml.v3"
)

type Assertion struct {
	Timestamps struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
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
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
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

func GenerateReport(trace executor.ExecutionTrace) FinalResult {
	config := trace.Playbook
	var md strings.Builder
	var log strings.Builder

	logName := trace.Timestamps.Start.Format(time.RFC3339)
	if trace.Timestamps.Start.IsZero() {
		logName = "execution trace"
	}

	log.WriteString(fmt.Sprintf(">>>>>>>>>>>> REPORT LOG: %s <<<<<<<<<<<<\n\n", logName))

	if config.ReportFrontmatter == nil {
		config.ReportFrontmatter = make(map[string]interface{})
	}
	if _, ok := config.ReportFrontmatter["title"]; !ok {
		config.ReportFrontmatter["title"] = config.Title
	}
	dateStr := ""
	if !trace.Timestamps.Start.IsZero() {
		dateStr = trace.Timestamps.Start.Format("2006-01-02")
	}
	if _, ok := config.ReportFrontmatter["date"]; !ok {
		config.ReportFrontmatter["date"] = dateStr
	}

	fmBytes, _ := yaml.Marshal(config.ReportFrontmatter)
	md.WriteString("---\n")
	md.Write(fmBytes)
	md.WriteString("---\n\n")

	md.WriteString(fmt.Sprintf("# %s\n\nGenerated on: %s\n\n---\n\n", config.Title, trace.Timestamps.Start.Format(time.DateTime)))

	finalReport := FinalReport{
		Username:   trace.Username,
		OS:         trace.OS,
		Arch:       trace.Arch,
		Assertions: make(map[string]Assertion),
	}
	finalReport.Timestamps.Start = trace.Timestamps.Start
	finalReport.Timestamps.End = trace.Timestamps.End

	for _, sectionCtx := range trace.Sections {
		section := sectionCtx.PlaybookSection
		md.WriteString(fmt.Sprintf("## %s\n\n", section.Title))
		log.WriteString(fmt.Sprintf(">>>>>>>>> SECTION: %s <<<<<<<<<\n\n", section.Title))
		for _, desc := range section.Description {
			md.WriteString(fmt.Sprintf("%s  \n", desc))
		}
		md.WriteString("\n")

		for _, assCtx := range sectionCtx.Assertions {
			assertion := assCtx.PlaybookAssertion

			log.WriteString(fmt.Sprintf(">>>>>>> ASSERTION: %s <<<<<<<\n\n", assertion.Title))

			for _, cmd := range assCtx.CmdLogs {
				writeExecutionLog(&log, cmd.Exec, cmd.Result, cmd.Err)
				if cmd.Err != nil {
					log.WriteString(fmt.Sprintf(">>>>> Error executing command: %v <<<<<\n\n", cmd.Err))
				}
			}

			report := Assertion{
				Passed:   assCtx.Passed,
				Score:    assCtx.Score,
				MinScore: assCtx.MinScore,
				Context:  assCtx.Context,
			}
			report.Timestamps.Start = assCtx.Timestamps.Start
			report.Timestamps.End = assCtx.Timestamps.End
			finalReport.Assertions[assertion.Code] = report

			writeAssertionMarkdown(&md, assCtx)
		}
		md.WriteString("---\n\n")
	}

	finalReport.Stats.Passed = trace.TotalPassed
	finalReport.Stats.Failed = trace.TotalFailed

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

func writeExecutionLog(log *strings.Builder, exec playbook.Exec, res executor.ExecutionResult, err error) {
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

func writeAssertionMarkdown(md *strings.Builder, a executor.AssertionContext) {
	assertion := a.PlaybookAssertion

	md.WriteString(fmt.Sprintf("### %s\n\n", assertion.Title))
	md.WriteString(fmt.Sprintf("%s\n\n", assertion.Description))

	outputs := a.Outputs
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

	if a.Passed {
		if assertion.PassDescription != "" {
			md.WriteString(fmt.Sprintf("> ✅ **Pass:** %s\n\n", assertion.PassDescription))
		} else {
			md.WriteString("> ✅ **Assertion Passed**")
		}
	} else {
		if assertion.FailDescription != "" {
			md.WriteString(fmt.Sprintf("> ❌ **Fail:** %s\n\n", assertion.FailDescription))
		} else {
			md.WriteString("> ❌ **Assertion Failed**")
		}
	}
}
