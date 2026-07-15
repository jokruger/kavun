package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
)

// Unary represents an unary operator expression.
type Unary struct {
	Expr     ast.Expression
	Token    token.Token
	TokenPos core.Pos
}

func (e *Unary) Pos() core.Pos {
	return e.Expr.Pos()
}

func (e *Unary) End() core.Pos {
	return e.Expr.End()
}

func (e *Unary) String() string {
	return "(" + e.Token.String() + e.Expr.String() + ")"
}

func (e *Unary) IsUndefinedLiteral() bool {
	return false
}

func (e *Unary) IsScalarLiteral() bool {
	return false
}

func (e *Unary) IsCompositeLiteral() bool {
	return false
}

func (e *Unary) IsCallExpression() bool {
	return false
}

func (e *Unary) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
