package parameter

// Parameter represents a parameter entity in the domain layer
// This mirrors the task.Parameter but provides domain-specific operations
type Parameter struct {
	MaxValue     *float64
	MinValue     *float64
	Type         string // "requires", "given", "accepts"
	DefaultValue string
	PatternMacro string
	Pattern      string
	DataType     string
	Name         string
	Constraints  []string
	Required     bool
	HasDefault   bool
	EmailFormat  bool
	Variadic     bool
}

// NewParameter creates a new parameter
func NewParameter(name, paramType string) *Parameter {
	return &Parameter{
		Name: name,
		Type: paramType,
	}
}

// IsRequired checks if the parameter is required
func (p *Parameter) IsRequired() bool {
	return p.Required || (p.Type == "requires")
}

// HasConstraints checks if parameter has validation constraints
func (p *Parameter) HasConstraints() bool {
	return len(p.Constraints) > 0 ||
		p.MinValue != nil ||
		p.MaxValue != nil ||
		p.Pattern != "" ||
		p.PatternMacro != "" ||
		p.EmailFormat
}
