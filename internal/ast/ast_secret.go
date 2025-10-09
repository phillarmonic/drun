package ast

import (
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// SecretStatement represents secret operations (set, get, delete, exists, list)
type SecretStatement struct {
	Token     lexer.Token
	Operation string      // "set", "get", "delete", "exists", "list"
	Key       string      // Secret key name
	Value     Expression  // For "set" operation
	Namespace string      // Optional namespace (defaults to current project)
	Pattern   string      // For "list" with pattern matching
	Default   Expression  // Default value for "get" operation
}

func (ss *SecretStatement) statementNode() {}
func (ss *SecretStatement) String() string {
	var out strings.Builder

	out.WriteString("secret ")
	out.WriteString(ss.Operation)

	switch ss.Operation {
	case "set":
		out.WriteString(" \"")
		out.WriteString(ss.Key)
		out.WriteString("\" to ")
		if ss.Value != nil {
			out.WriteString(ss.Value.String())
		}
		if ss.Namespace != "" {
			out.WriteString(" in namespace \"")
			out.WriteString(ss.Namespace)
			out.WriteString("\"")
		}

	case "get":
		out.WriteString(" \"")
		out.WriteString(ss.Key)
		out.WriteString("\"")
		if ss.Namespace != "" {
			out.WriteString(" from namespace \"")
			out.WriteString(ss.Namespace)
			out.WriteString("\"")
		}
		if ss.Default != nil {
			out.WriteString(" or ")
			out.WriteString(ss.Default.String())
		}

	case "delete":
		out.WriteString(" \"")
		out.WriteString(ss.Key)
		out.WriteString("\"")
		if ss.Namespace != "" {
			out.WriteString(" from namespace \"")
			out.WriteString(ss.Namespace)
			out.WriteString("\"")
		}

	case "exists":
		out.WriteString(" \"")
		out.WriteString(ss.Key)
		out.WriteString("\"")
		if ss.Namespace != "" {
			out.WriteString(" from namespace \"")
			out.WriteString(ss.Namespace)
			out.WriteString("\"")
		}

	case "list":
		if ss.Pattern != "" {
			out.WriteString(" matching \"")
			out.WriteString(ss.Pattern)
			out.WriteString("\"")
		}
		if ss.Namespace != "" {
			out.WriteString(" from namespace \"")
			out.WriteString(ss.Namespace)
			out.WriteString("\"")
		}

	default:
		out.WriteString(" ")
		out.WriteString(ss.Key)
	}

	return out.String()
}

// SecretExpression represents a secret access in an expression context
// Example: {secret('github_token')} or {secret('api_key', 'default_value')}
type SecretExpression struct {
	Token     lexer.Token
	Key       string
	Default   Expression
	Namespace string
}

func (se *SecretExpression) expressionNode() {}
func (se *SecretExpression) String() string {
	var out strings.Builder
	out.WriteString("secret('")
	out.WriteString(se.Key)
	out.WriteString("'")

	if se.Default != nil {
		out.WriteString(", ")
		out.WriteString(se.Default.String())
	}

	if se.Namespace != "" {
		out.WriteString(", namespace='")
		out.WriteString(se.Namespace)
		out.WriteString("'")
	}

	out.WriteString(")")
	return out.String()
}

