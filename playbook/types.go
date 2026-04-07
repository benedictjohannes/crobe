package playbook

type Assertion struct {
	Code            string `yaml:"code" json:"code" jsonschema:"description=Unique code for the assertion,minLength=3"`
	Title           string `yaml:"title" json:"title" jsonschema:"description=Title of the assertion,minLength=3"`
	Description     string `yaml:"description" json:"description" jsonschema:"description=Detailed description of what is being checked,minLength=3"`
	PreCmds         []Exec `yaml:"preCmds,omitempty" json:"preCmds,omitempty" jsonschema:"description=Executions before main commands"`
	Cmds            []Cmd  `yaml:"cmds" json:"cmds" jsonschema:"description=Main command units to execute,minItems=1"`
	PostCmds        []Exec `yaml:"postCmds,omitempty" json:"postCmds,omitempty" jsonschema:"description=Executions after main commands"`
	MinPassingScore *int   `yaml:"minPassingScore,omitempty" json:"minPassingScore,omitempty" jsonschema:"description=Minimum score to consider assertion as passed,default=1"`
	PassDescription string `yaml:"passDescription" json:"passDescription" jsonschema:"description=Message shown if passed,minLength=3"`
	FailDescription string `yaml:"failDescription" json:"failDescription" jsonschema:"description=Message shown if failed,minLength=3"`
}

func (a Assertion) GetMinPassingScore() int {
	if a.MinPassingScore == nil {
		return 1
	}
	return *a.MinPassingScore
}

type Cmd struct {
	Exec          Exec           `yaml:"exec" json:"exec" jsonschema:"description=Execution details"`
	PassScore     *int           `yaml:"passScore,omitempty" json:"passScore,omitempty" jsonschema:"description=Score added if passed,default=1"`
	FailScore     *int           `yaml:"failScore,omitempty" json:"failScore,omitempty" jsonschema:"description=Score added if failed,default=-1"`
	StdOutRule    EvaluationRule `yaml:"stdOutRule,omitempty" json:"stdOutRule,omitempty" jsonschema:"description=Rule for stdout evaluation"`
	StdErrRule    EvaluationRule `yaml:"stdErrRule,omitempty" json:"stdErrRule,omitempty" jsonschema:"description=Rule for stderr evaluation"`
	ExitCodeRules []ExitCodeRule `yaml:"exitCodeRules,omitempty" json:"exitCodeRules,omitempty" jsonschema:"description=Rules for exit code evaluation"`
}

func (c Cmd) GetPassScore() int {
	if c.PassScore == nil {
		return 1
	}
	return *c.PassScore
}

func (c Cmd) GetFailScore() int {
	if c.FailScore == nil {
		return -1
	}
	return *c.FailScore
}

type Exec struct {
	Shell             string       `yaml:"shell,omitempty" json:"shell,omitempty" jsonschema:"description=Shell to use (eg: bash\\, powershell\\, sh)"`
	Script            string       `yaml:"script,omitempty" json:"script,omitempty" jsonschema:"description=Script to execute"`
	Func              string       `yaml:"func,omitempty" json:"func,omitempty" jsonschema:"description=Embedded JS code. Signature: ({ assertionContext\\, env\\, os\\, arch\\, user\\, cwd }) => string"`
	FuncFile          string       `yaml:"funcFile,omitempty" json:"funcFile,omitempty" jsonschema:"description=Path to JS/TS file. BUILDER ONLY: using this in real playbook will cause error."`
	Gather            []GatherSpec `yaml:"gather,omitempty" json:"gather,omitempty" jsonschema:"description=Data extraction specs"`
	ExcludeFromReport bool         `yaml:"excludeFromReport,omitempty" json:"excludeFromReport,omitempty" jsonschema:"description=Hide stdout/stderr results from log and markdown report"`
}

type EvaluationRule struct {
	Regex         string `yaml:"regex,omitempty" json:"regex,omitempty" jsonschema:"description=Regex to match against output"`
	IncludeStdErr *bool  `yaml:"includeStdErr,omitempty" json:"includeStdErr,omitempty" jsonschema:"description=Include stderr in regex evaluation,default=false"`
	Func          string `yaml:"func,omitempty" json:"func,omitempty" jsonschema:"description=JS function for evaluation. Signature: (stdout\\, stderr\\, assertionContext) => -1 | 0 | 1"`
	FuncFile      string `yaml:"funcFile,omitempty" json:"funcFile,omitempty" jsonschema:"description=Path to JS/TS file. BUILDER ONLY: using this in real playbook will cause error."`
}

func (r EvaluationRule) GetIncludeStdErr() bool {
	if r.IncludeStdErr == nil {
		return false
	}
	return *r.IncludeStdErr
}

type GatherSpec struct {
	Key               string `yaml:"key" json:"key" jsonschema:"description=Key in context"`
	ExcludeFromReport bool   `yaml:"excludeFromReport,omitempty" json:"excludeFromReport,omitempty" jsonschema:"description=Hide key from JSON report"`
	Regex             string `yaml:"regex,omitempty" json:"regex,omitempty" jsonschema:"description=Regex to extract data"`
	IncludeStdErr     *bool  `yaml:"includeStdErr,omitempty" json:"includeStdErr,omitempty" jsonschema:"description=Include stderr in regex evaluation,default=false"`
	Func              string `yaml:"func,omitempty" json:"func,omitempty" jsonschema:"description=JS function for extraction. Signature: (stdout\\, stderr\\, assertionContext) => string"`
	FuncFile          string `yaml:"funcFile,omitempty" json:"funcFile,omitempty" jsonschema:"description=Path to JS/TS file. BUILDER ONLY: using this in real playbook will cause error."`
}

func (g GatherSpec) GetIncludeStdErr() bool {
	if g.IncludeStdErr == nil {
		return false
	}
	return *g.IncludeStdErr
}

type ExitCodeRule struct {
	Min    *int `yaml:"min,omitempty" json:"min,omitempty" jsonschema:"description=Minimum exit code"`
	Max    *int `yaml:"max,omitempty" json:"max,omitempty" jsonschema:"description=Maximum exit code"`
	Result int  `yaml:"result" json:"result" jsonschema:"description=Score result (-1\\, 0\\, 1)"`
}

type ReportDestination string

const (
	ReportDestinationFolder ReportDestination = "folder"
	ReportDestinationHTTPS  ReportDestination = "https"
)

type Section struct {
	Title       string      `yaml:"title" json:"title" jsonschema:"description=Title of the section,minLength=3"`
	Description []string    `yaml:"description" json:"description" jsonschema:"description=List of descriptions for the section,minItems=1"`
	Assertions  []Assertion `yaml:"assertions" json:"assertions" jsonschema:"description=List of assertions within this section,minItems=1"`
}

type ReportFormat string

const (
	ReportFormatMultipart ReportFormat = "multipart"
	ReportFormatJSON      ReportFormat = "json"
)

type ReportDestinationConfig struct {
	URL               string            `yaml:"url" json:"url" jsonschema:"description=URL to post report content to"`
	Format            ReportFormat      `yaml:"format,omitempty" json:"format,omitempty" jsonschema:"description=Format of the report (json|multipart),default=multipart,enum=multipart,enum=json"`
	SignatureSecret   string            `yaml:"signatureSecret,omitempty" json:"signatureSecret,omitempty" jsonschema:"description=Secret for HMAC-SHA256 signature"`
	AdditionalHeaders map[string]string `yaml:"additionalHeaders,omitempty" json:"additionalHeaders,omitempty" jsonschema:"description=Custom headers for the request"`
}

type ReportConfig struct {
	Title                  string                   `yaml:"title" json:"title" jsonschema:"description=Title of the report,minLength=3"`
	ReportFrontmatter      map[string]interface{}   `yaml:"reportFrontmatter,omitempty" json:"reportFrontmatter,omitempty" jsonschema:"description=Custom YAML frontmatter for markdown reports"`
	Sections               []Section                `yaml:"sections" json:"sections" jsonschema:"description=List of sections in the report,minItems=1"`
	ReportDestination      ReportDestination        `yaml:"reportDestination,omitempty" json:"reportDestination,omitempty" jsonschema:"description=Destination for the report,default=folder,enum=folder,enum=https"`
	ReportDestinationHTTPS *ReportDestinationConfig `yaml:"reportDestinationHttps,omitempty" json:"reportDestinationHttps,omitempty" jsonschema:"description=Configuration for HTTPS report destination"`
}
