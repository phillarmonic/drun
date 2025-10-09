package statement

// Secret represents secret management operations
type Secret struct {
	Operation string // "set", "get", "delete", "exists", "list"
	Key       string
	Value     string // For "set" operation (interpolated)
	Namespace string // Optional namespace (defaults to current project)
	Pattern   string // For "list" with pattern matching
	Default   string // Default value for "get" operation (interpolated)
}

func (s *Secret) Type() StatementType { return TypeSecret }
