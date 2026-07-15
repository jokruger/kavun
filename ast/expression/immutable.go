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

func (e *Immutable) Pos() core.Pos {
	return e.IPos
}

func (e *Immutable) End() core.Pos {
	return e.RParen
}

func (e *Immutable) String() string {
	return "immutable(" + e.Expr.String() + ")"
}

func (e *Immutable) IsUndefinedLiteral() bool {
	return false
}

func (e *Immutable) IsScalarLiteral() bool {
	return false
}

func (e *Immutable) IsCompositeLiteral() bool {
	return false
}

func (e *Immutable) IsCallExpression() bool {
	return false
}
