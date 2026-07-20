package expression

import (
	"github.com/jokruger/kavun/ast/statement"
	"github.com/jokruger/kavun/core"
)

// Function represents a function literal.
type Function struct {
	Type *FunctionType
	Body *statement.Block
}

func (e *Function) Pos() core.Pos {
	return e.Type.Pos()
}

func (e *Function) End() core.Pos {
	return e.Body.End()
}

func (e *Function) String() string {
	return "func" + e.Type.Params.String() + " " + e.Body.String()
}

func (e *Function) IsUndefinedLiteral() bool {
	return false
}

func (e *Function) IsScalarLiteral() bool {
	return false
}

func (e *Function) IsCompositeLiteral() bool {
	return false
}

func (e *Function) IsCallExpression() bool {
	return false
}

func (e *Function) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
