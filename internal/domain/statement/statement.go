package statement

// Statement represents a domain-level executable statement
// This abstracts away from AST details and represents what needs to be executed
type Statement interface {
	// Type returns the statement type for runtime type checking
	Type() StatementType
}

// StatementType identifies the type of statement
type StatementType string

const (
	TypeAction           StatementType = "action"
	TypeShell            StatementType = "shell"
	TypeVariable         StatementType = "variable"
	TypeConditional      StatementType = "conditional"
	TypeLoop             StatementType = "loop"
	TypeTry              StatementType = "try"
	TypeThrow            StatementType = "throw"
	TypeBreak            StatementType = "break"
	TypeContinue         StatementType = "continue"
	TypeTaskCall         StatementType = "task_call"
	TypeTaskFromTemplate StatementType = "task_from_template"
	TypeDocker           StatementType = "docker"
	TypeGit              StatementType = "git"
	TypeGitQuery         StatementType = "git_query"
	TypeGitEnsureVersion StatementType = "git_ensure_version"
	TypeHTTP             StatementType = "http"
	TypeDownload         StatementType = "download"
	TypeNetwork          StatementType = "network"
	TypeWait             StatementType = "wait"
	TypeOpen             StatementType = "open"
	TypeConfirm          StatementType = "confirm"
	TypePrompt           StatementType = "prompt"
	TypeFile             StatementType = "file"
	TypeFileValue        StatementType = "file_value"
	TypeChangelog        StatementType = "changelog"
	TypeDetection        StatementType = "detection"
	TypeUseSnippet       StatementType = "use_snippet"
	TypeSecret           StatementType = "secret"
	TypeOrchestration    StatementType = "orchestration"
	TypeChangeWorkdir    StatementType = "change_workdir"
	TypeRequiresTools    StatementType = "requires_tools"
	TypeGitPolicy        StatementType = "git_policy"
	TypeGitValidate      StatementType = "git_validate"
)

// Action represents an action statement (info, step, success, etc.)
type Action struct {
	ActionType      string
	Message         string
	LineBreakBefore bool
	LineBreakAfter  bool
}

func (a *Action) Type() StatementType { return TypeAction }

// Shell represents a shell command execution
type Shell struct {
	Action               string
	Command              string
	CaptureVar           string
	ServiceName          string
	Commands             []string
	Attached             bool
	StreamOutput         bool
	IsMultiline          bool
	ServiceScoped        bool
	ServiceNameIsLiteral bool
}

func (s *Shell) Type() StatementType { return TypeShell }

// Variable represents variable operations (let, set, transform)
type Variable struct {
	Operation string
	Name      string
	Value     string // Interpolated value as string
	Function  string
	Arguments []string
}

func (v *Variable) Type() StatementType { return TypeVariable }

// Conditional represents when/if/otherwise statements
type Conditional struct {
	ConditionType string // "when", "if", "otherwise"
	Condition     string
	Body          []Statement
	ElseBody      []Statement
}

func (c *Conditional) Type() StatementType { return TypeConditional }

// Loop represents for each loops
type Loop struct {
	Filter     *Filter
	LoopType   string // "each", "range", "line", "match"
	Variable   string
	Iterable   string
	RangeStart string
	RangeEnd   string
	RangeStep  string
	Subject    string // match loops: the $variable the pattern is matched against
	Body       []Statement
	MaxWorkers int
	Parallel   bool
	FailFast   bool
}

func (l *Loop) Type() StatementType { return TypeLoop }

// Filter represents filter conditions in loops
type Filter struct {
	Variable string
	Operator string
	Value    string
}

// Try represents try/catch/finally error handling
type Try struct {
	TryBody      []Statement
	CatchClauses []CatchClause
	FinallyBody  []Statement
}

func (t *Try) Type() StatementType { return TypeTry }

// CatchClause represents a catch clause within a try statement
type CatchClause struct {
	ErrorType string
	ErrorVar  string
	Body      []Statement
}

// Throw represents throw/rethrow/ignore statements
type Throw struct {
	Action  string // "throw", "rethrow", "ignore"
	Message string
}

func (t *Throw) Type() StatementType { return TypeThrow }

// Break represents break statements in loops
type Break struct {
	Condition string
}

func (b *Break) Type() StatementType { return TypeBreak }

// Continue represents continue statements in loops
type Continue struct {
	Condition string
}

func (c *Continue) Type() StatementType { return TypeContinue }

// TaskCall represents calling another task
type TaskCall struct {
	Parameters map[string]string
	TaskName   string
}

func (tc *TaskCall) Type() StatementType { return TypeTaskCall }

// TaskFromTemplate represents a task instantiated from a template
type TaskFromTemplate struct {
	Overrides    map[string]string
	Name         string
	TemplateName string
}

func (tft *TaskFromTemplate) Type() StatementType { return TypeTaskFromTemplate }

// Docker represents Docker operations
type Docker struct {
	Options              map[string]string
	Operation            string
	Resource             string
	Name                 string
	ServiceName          string
	ServiceScoped        bool
	ServiceNameIsLiteral bool
}

func (d *Docker) Type() StatementType { return TypeDocker }

// Git represents Git operations
type Git struct {
	Options   map[string]string
	Operation string
	Resource  string
	Name      string
}

func (g *Git) Type() StatementType { return TypeGit }

// GitQuery resolves a versioned tag from a registered project-level Git source.
type GitQuery struct {
	Result         string
	Source         string
	AccessMethod   string
	TagPreset      string
	TagFormat      string
	TagPattern     string
	Series         string
	VersionMatcher string
	OrderBy        string
	CaptureVar     string
	AllowFetch     bool
}

func (g *GitQuery) Type() StatementType { return TypeGitQuery }

// GitEnsureVersion guards a candidate against a source's latest stable version.
type GitEnsureVersion struct {
	Candidate           string
	Source              string
	AccessMethod        string
	TagPreset           string
	TagFormat           string
	TagPattern          string
	CaptureVar          string
	CandidateIsVariable bool
}

func (g *GitEnsureVersion) Type() StatementType { return TypeGitEnsureVersion }

// HTTP represents HTTP operations
type HTTP struct {
	Headers map[string]string
	Auth    map[string]string
	Options map[string]string
	Method  string
	URL     string
	Body    string
}

func (h *HTTP) Type() StatementType { return TypeHTTP }

// PermissionSpec represents a permission specification for downloaded files
type PermissionSpec struct {
	Permissions []string
	Targets     []string
}

// Download represents file download operations
type Download struct {
	Headers          map[string]string
	Auth             map[string]string
	Options          map[string]string
	URL              string
	Path             string
	ExtractTo        string
	AllowPermissions []PermissionSpec
	AllowOverwrite   bool
	RemoveArchive    bool
}

func (d *Download) Type() StatementType { return TypeDownload }

// Network represents network operations
type Network struct {
	Action    string
	Target    string
	Port      string
	Options   map[string]string
	Condition string
}

func (n *Network) Type() StatementType { return TypeNetwork }

// Wait represents a fixed-duration wait (wait 5 seconds)
type Wait struct {
	Value string // Raw value: a number literal or a {variable} interpolation
	Unit  string // Normalized singular unit: "second", "minute", "hour"
}

func (w *Wait) Type() StatementType { return TypeWait }

// Open represents opening a URL or file in the OS default handler (open url "...")
type Open struct {
	Noun string // Always "url" today
	URL  string // Raw target; may contain {variable} interpolation
}

func (o *Open) Type() StatementType { return TypeOpen }

// Confirm represents an interactive y/n confirmation gate
// (confirm "Deploy to production?"). Without a result variable the bare form
// stops the task gracefully when declined; with `as $var` the answer is stored
// using drun's string-boolean convention ("true"/"false").
type Confirm struct {
	Question     string // Prompt text; may contain {variable} interpolation
	DefaultValue string // Raw default value (e.g. "no"); empty unless HasDefault is set
	ResultVar    string // Variable name without the $ prefix; empty for a bare gate
	HasDefault   bool   // Whether a `defaults to` value was declared
}

func (c *Confirm) Type() StatementType { return TypeConfirm }

// Prompt represents a free-form interactive text prompt
// (prompt "Which environment?" as $environment). The typed answer is stored in
// the result variable; `defaults to` supplies the value used when the user
// presses enter with no input.
type Prompt struct {
	Question     string // Prompt text; may contain {variable} interpolation
	DefaultValue string // Raw default value; empty unless HasDefault is set
	ResultVar    string // Variable name without the $ prefix receiving the answer
	HasDefault   bool   // Whether a `defaults to` value was declared
}

func (p *Prompt) Type() StatementType { return TypePrompt }

// File represents file operations
type File struct {
	Replacements map[string]string
	Action       string
	Target       string
	Source       string
	Content      string
	CaptureVar   string
	IsDir        bool
}

func (f *File) Type() StatementType { return TypeFile }

// FileValue represents a format-aware scalar operation on a text file.
type FileValue struct {
	Operation     string
	Format        string
	Selector      string
	Target        string
	CaptureVar    string
	Comparison    string
	Expected      string
	Value         string
	MissingPolicy string
	ValueType     string
}

func (f *FileValue) Type() StatementType { return TypeFileValue }

// Changelog represents a Keep a Changelog promotion of the Unreleased
// section into a dated release section.
type Changelog struct {
	Path    string
	Version string
	Date    string // Optional release date override (YYYY-MM-DD), empty means today
}

func (c *Changelog) Type() StatementType { return TypeChangelog }

// Detection represents tool detection operations
type Detection struct {
	DetectionType string // "detect", "detect_available", "if_available", "when_environment", "if_version"
	Target        string
	Alternatives  []string
	Condition     string
	Value         string
	VersionOp     string
	VersionValue  string
	CaptureVar    string
	Body          []Statement
	ElseBody      []Statement
}

func (d *Detection) Type() StatementType { return TypeDetection }

// UseSnippet represents using a code snippet
type UseSnippet struct {
	SnippetName string
}

func (us *UseSnippet) Type() StatementType { return TypeUseSnippet }

// Orchestration represents orchestration action operations
type Orchestration struct {
	GroupName      string
	Action         string // start, stop, restart, health_check, status, logs, etc.
	Options        map[string]string
	ServiceFilters []string
}

func (o *Orchestration) Type() StatementType { return TypeOrchestration }

// ChangeWorkdir represents a working directory change within a task.
// Subsequent shell commands in the same task will run in this directory.
// Relative paths are resolved against the original cwd (not chained).
type ChangeWorkdir struct {
	Path string
}

func (cw *ChangeWorkdir) Type() StatementType { return TypeChangeWorkdir }

// VersionConstraint represents a single version constraint (e.g., >= "2.27")
type VersionConstraint struct {
	Operator string // ">=", ">", "<=", "<"
	Version  string // "2.27", "3.0", etc.
}

// ToolRequirement represents a tool requirement with optional version constraints
type ToolRequirement struct {
	Name          string              // tool name (e.g., "gosec", "golangci-lint")
	Constraints   []VersionConstraint // zero or more version constraints
	AutoProvision bool                // whether runtime may install or upgrade the tool automatically
}

// RequiresTools represents a "requires tools:" block that validates tool
// availability and version constraints before execution proceeds.
type RequiresTools struct {
	Tools    []ToolRequirement
	TaskRefs []string
}

func (rt *RequiresTools) Type() StatementType { return TypeRequiresTools }

// GitPolicy represents a project-level setting for git conventions.
type GitPolicy struct {
	BranchPattern        string
	CommitPattern        string
	DefaultBranches      []string
	ProtectedBranches    []string
	BranchTypes          []string
	CommitBans           []string
	CommitMinLength      int
	ExtractIdentifier    bool
	EnforceSignedCommits bool
}

func (gp *GitPolicy) Type() StatementType { return TypeGitPolicy }

// GitValidate represents an inline git validation statement within a task.
type GitValidate struct {
	Target string // "branch_name", "commit_message", "signed_commits", "all"
	Value  string // optional explicit value to validate (e.g. commit message text)
}

func (gv *GitValidate) Type() StatementType { return TypeGitValidate }
