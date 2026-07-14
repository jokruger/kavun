package expression

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/ast"
)

// Ternary represents a ternary conditional expression.
type Ternary struct {
	Cond        ast.Expression
	True        ast.Expression
	False       ast.Expression
	QuestionPos core.Pos
	ColonPos    core.Pos
}

func (e *Ternary) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Ternary) Pos() core.Pos {
	return e.Cond.Pos()
}

// End returns the position of first character immediately after the node.
func (e *Ternary) End() core.Pos {
	return e.False.End()
}

func (e *Ternary) String() string {
	return "(" + e.Cond.String() + " ? " + e.True.String() + " : " + e.False.String() + ")"
}
