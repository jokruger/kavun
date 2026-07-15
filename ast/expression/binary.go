package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
)

// Binary represents a binary operator expression.
type Binary struct {
	LHS      ast.Expression
	RHS      ast.Expression
	Token    token.Token
	TokenPos core.Pos
}

func (e *Binary) Pos() core.Pos {
	return e.LHS.Pos()
}

func (e *Binary) End() core.Pos {
	return e.RHS.End()
}

func (e *Binary) String() string {
	return "(" + e.LHS.String() + " " + e.Token.String() + " " + e.RHS.String() + ")"
}

func (e *Binary) IsUndefinedLiteral() bool {
	return false
}

func (e *Binary) IsScalarLiteral() bool {
	return false
}

func (e *Binary) IsCompositeLiteral() bool {
	return false
}

func (e *Binary) IsCallExpression() bool {
	return false
}

func (e *Binary) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
