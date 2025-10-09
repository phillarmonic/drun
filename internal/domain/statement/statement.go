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
	TypeHTTP             StatementType = "http"
	TypeDownload         StatementType = "download"
	TypeNetwork          StatementType = "network"
	TypeFile             StatementType = "file"
	TypeDetection        StatementType = "detection"
	TypeUseSnippet       StatementType = "use_snippet"
	TypeSecret           StatementType = "secret"
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
	Action       string
	Command      string
	Commands     []string
	CaptureVar   string
	StreamOutput bool
	IsMultiline  bool
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
	Operation string
	Resource  string
	Name      string
	Options   map[string]string
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
	Action     string
	Target     string
	Source     string
	Content    string
	IsDir      bool
	CaptureVar string
}

func (f *File) Type() StatementType { return TypeFile }

// Detection represents tool detection operations
type Detection struct {
	DetectionType string // "detect", "detect_available", "if_available", "when_environment", "if_version"
	Target        string
	Alternatives  []string
	Condition     string
	Value         string
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
