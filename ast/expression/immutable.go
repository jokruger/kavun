package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Immutable represents an immutable expression
type Immutable struct {
	Expr   ast.Expression
	IPos   core.Pos
	LParen core.Pos
	RParen core.Pos
}

func (e *Immutable) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Immutable) Pos() core.Pos {
	return e.IPos
}

// End returns the position of first character immediately after the node.
func (e *Immutable) End() core.Pos {
	return e.RParen
}

func (e *Immutable) String() string {
	return "immutable(" + e.Expr.String() + ")"
}
