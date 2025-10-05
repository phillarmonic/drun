package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// ConditionalStatement represents when/if statements
type ConditionalStatement struct {
	Token     lexer.Token
	Type      string
	Condition string
	Body      []Statement
	ElseBody  []Statement
}

func (cs *ConditionalStatement) statementNode() {}
func (cs *ConditionalStatement) String() string {
	var out strings.Builder
	out.WriteString(cs.Type)
	out.WriteString(" ")
	out.WriteString(cs.Condition)
	out.WriteString(":\n")

	for _, stmt := range cs.Body {
		out.WriteString("  ")
		out.WriteString(stmt.String())
		out.WriteString("\n")
	}

	if len(cs.ElseBody) > 0 {
		out.WriteString("else:\n")
		for _, stmt := range cs.ElseBody {
			out.WriteString("  ")
			out.WriteString(stmt.String())
			out.WriteString("\n")
		}
	}

	return out.String()
}

// LoopStatement represents for each loops
type LoopStatement struct {
	Token      lexer.Token
	Type       string
	Variable   string
	Iterable   string
	RangeStart string
	RangeEnd   string
	RangeStep  string
	Filter     *FilterExpression
	Parallel   bool
	MaxWorkers int
	FailFast   bool
	Body       []Statement
}

func (ls *LoopStatement) statementNode() {}
func (ls *LoopStatement) String() string {
	var out strings.Builder

	switch ls.Type {
	case "range":
		out.WriteString("for ")
		out.WriteString(ls.Variable)
		out.WriteString(" in range ")
		out.WriteString(ls.RangeStart)
		out.WriteString(" to ")
		out.WriteString(ls.RangeEnd)
		if ls.RangeStep != "" {
			out.WriteString(" step ")
			out.WriteString(ls.RangeStep)
		}
	case "line":
		out.WriteString("for each line ")
		out.WriteString(ls.Variable)
		out.WriteString(" in file ")
		out.WriteString(ls.Iterable)
	case "match":
		out.WriteString("for each match ")
		out.WriteString(ls.Variable)
		out.WriteString(" in pattern ")
		out.WriteString(ls.Iterable)
	default: // "each"
		out.WriteString("for each ")
		out.WriteString(ls.Variable)
		out.WriteString(" in ")
		out.WriteString(ls.Iterable)
	}

	if ls.Filter != nil {
		out.WriteString(" where ")
		out.WriteString(ls.Filter.Variable)
		out.WriteString(" ")
		out.WriteString(ls.Filter.Operator)
		out.WriteString(" ")
		out.WriteString(ls.Filter.Value)
	}

	if ls.Parallel {
		out.WriteString(" in parallel")
	}

	out.WriteString(":\n")

	for _, stmt := range ls.Body {
		out.WriteString("  ")
		out.WriteString(stmt.String())
		out.WriteString("\n")
	}

	return out.String()
}

// TryStatement represents try/catch/finally error handling blocks
type TryStatement struct {
	Token        lexer.Token
	TryBody      []Statement
	CatchClauses []CatchClause
	FinallyBody  []Statement
}

func (ts *TryStatement) statementNode() {}
func (ts *TryStatement) String() string {
	var out strings.Builder
	out.WriteString("try:")

	for _, stmt := range ts.TryBody {
		out.WriteString("\n  ")
		out.WriteString(stmt.String())
	}

	for _, catch := range ts.CatchClauses {
		out.WriteString("\n")
		out.WriteString(catch.String())
	}

	if len(ts.FinallyBody) > 0 {
		out.WriteString("\nfinally:")
		for _, stmt := range ts.FinallyBody {
			out.WriteString("\n  ")
			out.WriteString(stmt.String())
		}
	}

	return out.String()
}

// CatchClause represents a catch clause within a try statement
type CatchClause struct {
	Token     lexer.Token
	ErrorType string
	ErrorVar  string
	Body      []Statement
}

func (cc *CatchClause) String() string {
	var out strings.Builder
	out.WriteString("catch")

	if cc.ErrorType != "" {
		out.WriteString(" ")
		out.WriteString(cc.ErrorType)
	}

	if cc.ErrorVar != "" {
		out.WriteString(" as ")
		out.WriteString(cc.ErrorVar)
	}

	out.WriteString(":")

	for _, stmt := range cc.Body {
		out.WriteString("\n  ")
		out.WriteString(stmt.String())
	}

	return out.String()
}

// ThrowStatement represents throw and rethrow statements
type ThrowStatement struct {
	Token   lexer.Token
	Action  string
	Message string
}

func (ts *ThrowStatement) statementNode() {}
func (ts *ThrowStatement) String() string {
	switch ts.Action {
	case "throw":
		return fmt.Sprintf("throw \"%s\"", ts.Message)
	case "rethrow":
		return "rethrow"
	case "ignore":
		return "ignore"
	default:
		return ts.Action
	}
}

// BreakStatement represents break statements in loops
type BreakStatement struct {
	Token     lexer.Token
	Condition string
}

func (bs *BreakStatement) statementNode() {}
func (bs *BreakStatement) String() string {
	if bs.Condition != "" {
		return "break when " + bs.Condition
	}
	return "break"
}

// ContinueStatement represents continue statements in loops
type ContinueStatement struct {
	Token     lexer.Token
	Condition string
}

func (cs *ContinueStatement) statementNode() {}
func (cs *ContinueStatement) String() string {
	if cs.Condition != "" {
		return "continue if " + cs.Condition
	}
	return "continue"
}

// FilterExpression represents filter conditions in loops
type FilterExpression struct {
	Variable string
	Operator string
	Value    string
}
