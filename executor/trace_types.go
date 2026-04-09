package executor

import (
	"time"

	"github.com/benedictjohannes/crobe/playbook"
)

type CommandLog struct {
	Exec   playbook.Exec
	Result ExecutionResult
	Err    error
}

type AssertionContext struct {
	PlaybookAssertion playbook.Assertion
	Timestamps        struct {
		Start time.Time
		End   time.Time
	}
	Passed      bool
	Score       int
	MinScore    int
	Context     map[string]interface{}
	PreCmdLogs  []CommandLog
	CmdLogs     []CommandLog
	PostCmdLogs []CommandLog
	Outputs     []string
}

type SectionContext struct {
	PlaybookSection playbook.Section
	Assertions      []AssertionContext
}

type ExecutionTrace struct {
	Playbook    playbook.Playbook
	Sections    []SectionContext
	Timestamps  struct {
		Start time.Time
		End   time.Time
	}
	Username    string
	OS          string
	Arch        string
	TotalPassed int
	TotalFailed int
}
