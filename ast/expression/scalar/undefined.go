package scalar

import (
	"github.com/jokruger/kavun/core"
)

// Undefined represents an undefined literal.
type Undefined struct {
	TokenPos core.Pos
}

func (e *Undefined) Pos() core.Pos {
	return e.TokenPos
}

func (e *Undefined) End() core.Pos {
	return e.TokenPos + 9 // len(undefined) == 9
}

func (e *Undefined) String() string {
	return "undefined"
}

func (e *Undefined) IsUndefinedLiteral() bool {
	return true
}

func (e *Undefined) IsScalarLiteral() bool {
	return true
}

func (e *Undefined) IsCompositeLiteral() bool {
	return false
}

func (e *Undefined) IsCallExpression() bool {
	return false
}
