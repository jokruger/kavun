package expression

import (
	"github.com/jokruger/kavun/core"
)

// Undefined represents an undefined literal.
type Undefined struct {
	TokenPos core.Pos
}

func (e *Undefined) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Undefined) Pos() core.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *Undefined) End() core.Pos {
	return e.TokenPos + 9 // len(undefined) == 9
}

func (e *Undefined) String() string {
	return "undefined"
}
