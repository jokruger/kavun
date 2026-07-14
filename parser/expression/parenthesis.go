package expression

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/ast"
)

// Parenthesis represents a parenthesis wrapped expression.
type Parenthesis struct {
	Expr   ast.Expression
	LParen core.Pos
	RParen core.Pos
}

func (e *Parenthesis) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Parenthesis) Pos() core.Pos {
	return e.LParen
}

// End returns the position of first character immediately after the node.
func (e *Parenthesis) End() core.Pos {
	return e.RParen + 1
}

func (e *Parenthesis) String() string {
	return "(" + e.Expr.String() + ")"
}
