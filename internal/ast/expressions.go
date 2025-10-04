// Package ast defines expression-related AST nodes for the drun v2 language.
package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// BinaryExpression represents binary operations like {a} + {b}, {x} - {y}
type BinaryExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (be *BinaryExpression) expressionNode() {}
func (be *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", be.Left.String(), be.Operator, be.Right.String())
}

// IdentifierExpression represents variable references like {variable_name}
type IdentifierExpression struct {
	Token lexer.Token
	Value string
}

func (ie *IdentifierExpression) expressionNode() {}
func (ie *IdentifierExpression) String() string {
	return fmt.Sprintf("{%s}", ie.Value)
}

// LiteralExpression represents literal values like "string", 42, true
type LiteralExpression struct {
	Token lexer.Token
	Value string
}

func (le *LiteralExpression) expressionNode() {}
func (le *LiteralExpression) String() string {
	return le.Value
}

// FunctionCallExpression represents function calls like now(), current git branch
type FunctionCallExpression struct {
	Token     lexer.Token
	Function  string
	Arguments []Expression
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

// ArrayLiteral represents array literals like ["item1", "item2", "item3"]
type ArrayLiteral struct {
	Token    lexer.Token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode() {}
func (al *ArrayLiteral) String() string {
	var elements []string
	for _, elem := range al.Elements {
		elements = append(elements, elem.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}
