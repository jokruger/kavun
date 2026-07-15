package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Parenthesis represents a parenthesis wrapped expression.
type Parenthesis struct {
	Expr   ast.Expression
	LParen core.Pos
	RParen core.Pos
}

func (e *Parenthesis) Pos() core.Pos {
	return e.LParen
}

func (e *Parenthesis) End() core.Pos {
	return e.RParen + 1
}

func (e *Parenthesis) String() string {
	return "(" + e.Expr.String() + ")"
}

func (e *Parenthesis) IsUndefinedLiteral() bool {
	return false
}

func (e *Parenthesis) IsScalarLiteral() bool {
	return false
}

func (e *Parenthesis) IsCompositeLiteral() bool {
	return false
}

func (e *Parenthesis) IsCallExpression() bool {
	return false
}

func (e *Parenthesis) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
