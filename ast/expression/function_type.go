package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// FunctionType represents a function type definition.
type FunctionType struct {
	FuncPos core.Pos
	Params  *ast.Identifiers

	// Result is the optional named return identifier:
	//   func(a, b) name { ... }
	// When non-nil, `name` is allocated as a local pre-initialized to undefined; bare `return` and exit-after-recover
	// return its current value.
	Result *ast.Identifier
}

func (e *FunctionType) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FunctionType) Pos() core.Pos {
	return e.FuncPos
}

// End returns the position of first character immediately after the node.
func (e *FunctionType) End() core.Pos {
	if e.Result != nil {
		return e.Result.End()
	}
	return e.Params.End()
}

func (e *FunctionType) String() string {
	if e.Result != nil {
		return "func" + e.Params.String() + " " + e.Result.Name
	}
	return "func" + e.Params.String()
}
