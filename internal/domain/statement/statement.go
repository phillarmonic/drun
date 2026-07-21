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
	TypeFile             StatementType = "file"
	TypeFileValue        StatementType = "file_value"
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
	Commands             []string
	CaptureVar           string
	Attached             bool
	StreamOutput         bool
	IsMultiline          bool
	ServiceScoped        bool
	ServiceName          string
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
	LoopType   string // "each", "range", "line", "match"
	Variable   string
	Iterable   string
	RangeStart string
	RangeEnd   string
	RangeStep  string
	Filter     *Filter
	Parallel   bool
	MaxWorkers int
	FailFast   bool
	Body       []Statement
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
	TaskName   string
	Parameters map[string]string
}

func (tc *TaskCall) Type() StatementType { return TypeTaskCall }

// TaskFromTemplate represents a task instantiated from a template
type TaskFromTemplate struct {
	Name         string
	TemplateName string
	Overrides    map[string]string
}

func (tft *TaskFromTemplate) Type() StatementType { return TypeTaskFromTemplate }

// Docker represents Docker operations
type Docker struct {
	Operation            string
	Resource             string
	Name                 string
	Options              map[string]string
	ServiceScoped        bool
	ServiceName          string
	ServiceNameIsLiteral bool
}

func (d *Docker) Type() StatementType { return TypeDocker }

// Git represents Git operations
type Git struct {
	Operation string
	Resource  string
	Name      string
	Options   map[string]string
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
	AllowFetch     bool
	CaptureVar     string
}

func (g *GitQuery) Type() StatementType { return TypeGitQuery }

// GitEnsureVersion guards a candidate against a source's latest stable version.
type GitEnsureVersion struct {
	Candidate           string
	CandidateIsVariable bool
	Source              string
	AccessMethod        string
	TagPreset           string
	TagFormat           string
	TagPattern          string
	CaptureVar          string
}

func (g *GitEnsureVersion) Type() StatementType { return TypeGitEnsureVersion }

// HTTP represents HTTP operations
type HTTP struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
	Auth    map[string]string
	Options map[string]string
}

func (h *HTTP) Type() StatementType { return TypeHTTP }

// PermissionSpec represents a permission specification for downloaded files
type PermissionSpec struct {
	Permissions []string
	Targets     []string
}

// Download represents file download operations
type Download struct {
	URL              string
	Path             string
	AllowOverwrite   bool
	AllowPermissions []PermissionSpec
	ExtractTo        string
	RemoveArchive    bool
	Headers          map[string]string
	Auth             map[string]string
	Options          map[string]string
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

// File represents file operations
type File struct {
	Action       string
	Target       string
	Source       string
	Content      string
	IsDir        bool
	CaptureVar   string
	Replacements map[string]string
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
	DefaultBranches      []string
	ProtectedBranches    []string
	BranchPattern        string
	BranchTypes          []string
	CommitPattern        string
	ExtractIdentifier    bool
	CommitMinLength      int
	CommitBans           []string
	EnforceSignedCommits bool
}

func (gp *GitPolicy) Type() StatementType { return TypeGitPolicy }

// GitValidate represents an inline git validation statement within a task.
type GitValidate struct {
	Target string // "branch_name", "commit_message", "signed_commits", "all"
	Value  string // optional explicit value to validate (e.g. commit message text)
}

func (gv *GitValidate) Type() StatementType { return TypeGitValidate }
