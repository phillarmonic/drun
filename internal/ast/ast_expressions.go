// Package ast defines expression-related AST nodes for the drun v2 language.
package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// Expression represents any expression node
type Expression interface {
	Node
	expressionNode()
}

// BinaryExpression represents binary operations like {a} + {b}, {x} - {y}
type BinaryExpression struct {
	Token    lexer.Token // the operator token
	Left     Expression  // left operand
	Operator string      // +, -, *, /, ==, !=, <, >, <=, >=
	Right    Expression  // right operand
}

func (be *BinaryExpression) expressionNode() {}
func (be *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", be.Left.String(), be.Operator, be.Right.String())
}

// IdentifierExpression represents variable references like {variable_name}
type IdentifierExpression struct {
	Token lexer.Token // the identifier token
	Value string      // the variable name
}

func (ie *IdentifierExpression) expressionNode() {}
func (ie *IdentifierExpression) String() string {
	return fmt.Sprintf("{%s}", ie.Value)
}

// LiteralExpression represents literal values like "string", 42, true
type LiteralExpression struct {
	Token lexer.Token // the literal token
	Value string      // the literal value
}

func (le *LiteralExpression) expressionNode() {}
func (le *LiteralExpression) String() string {
	return le.Value
}

// FunctionCallExpression represents function calls like now(), current git branch
type FunctionCallExpression struct {
	Token     lexer.Token  // the function name token
	Function  string       // function name
	Arguments []Expression // function arguments
}

func (fce *FunctionCallExpression) expressionNode() {}
func (fce *FunctionCallExpression) String() string {
	var args []string
	for _, arg := range fce.Arguments {
		args = append(args, arg.String())
	}
	if len(args) > 0 {
		return fmt.Sprintf("%s(%s)", fce.Function, strings.Join(args, ", "))
	}
	return fce.Function
}
