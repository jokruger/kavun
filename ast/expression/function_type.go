package expression

import (
	"github.com/jokruger/kavun/core"
)

// FunctionType represents a function type definition.
type FunctionType struct {
	FuncPos core.Pos
	Params  *Identifiers

	// Result is the optional named return identifier:
	//   func(a, b) name { ... }
	// When non-nil, `name` is allocated as a local pre-initialized to undefined; bare `return` and exit-after-recover
	// return its current value.
	Result *Identifier
}

func (e *FunctionType) Pos() core.Pos {
	return e.FuncPos
}

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

func (e *FunctionType) IsUndefinedLiteral() bool {
	return false
}

func (e *FunctionType) IsScalarLiteral() bool {
	return false
}

func (e *FunctionType) IsCompositeLiteral() bool {
	return false
}

func (e *FunctionType) IsCallExpression() bool {
	return false
}
