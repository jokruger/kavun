package expression

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/statement"
)

// Function represents a function literal.
type Function struct {
	Type *FunctionType
	Body *statement.Block
}

func (e *Function) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Function) Pos() core.Pos {
	return e.Type.Pos()
}

// End returns the position of first character immediately after the node.
func (e *Function) End() core.Pos {
	return e.Body.End()
}

func (e *Function) String() string {
	return "func" + e.Type.Params.String() + " " + e.Body.String()
}
