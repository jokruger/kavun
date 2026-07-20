package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Ternary represents a ternary conditional expression.
type Ternary struct {
	Cond        ast.Expression
	True        ast.Expression
	False       ast.Expression
	QuestionPos core.Pos
	ColonPos    core.Pos
}

func (e *Ternary) Pos() core.Pos {
	return e.Cond.Pos()
}

func (e *Ternary) End() core.Pos {
	return e.False.End()
}

func (e *Ternary) String() string {
	return "(" + e.Cond.String() + " ? " + e.True.String() + " : " + e.False.String() + ")"
}

func (e *Ternary) IsUndefinedLiteral() bool {
	return false
}

func (e *Ternary) IsScalarLiteral() bool {
	return false
}

func (e *Ternary) IsCompositeLiteral() bool {
	return false
}

func (e *Ternary) IsCallExpression() bool {
	return false
}

func (e *Ternary) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
