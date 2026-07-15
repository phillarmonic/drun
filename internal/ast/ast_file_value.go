package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// FileValueStatement represents a format-aware scalar read, check, or update.
type FileValueStatement struct {
	Token         lexer.Token
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

func (fs *FileValueStatement) statementNode() {}

func (fs *FileValueStatement) String() string {
	if fs.Format == "drun" && fs.Selector == "project.version" {
		switch fs.Operation {
		case "check":
			operator := fs.Comparison
			if operator == "differs" {
				operator = "differs from"
			}
			return fmt.Sprintf("check project version %s %q", operator, fs.Expected)
		case "update":
			return fmt.Sprintf("update project version to %q", fs.Value)
		}
	}
	switch fs.Operation {
	case "get":
		return fmt.Sprintf("get %s %q from %q as $%s", fs.Format, fs.Selector, fs.Target, fs.CaptureVar)
	case "check":
		operator := fs.Comparison
		if operator == "differs" {
			operator = "differs from"
		}
		return fmt.Sprintf("check %s %q in %q %s %q", fs.Format, fs.Selector, fs.Target, operator, fs.Expected)
	case "update":
		result := fmt.Sprintf("update %s %q in %q to %q or %s", fs.Format, fs.Selector, fs.Target, fs.Value, fs.MissingPolicy)
		if fs.ValueType != "" {
			result += " as " + fs.ValueType
		}
		return result
	default:
		return fmt.Sprintf("%s %s %q in %q", fs.Operation, fs.Format, fs.Selector, fs.Target)
	}
}
