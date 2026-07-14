package scalar

import (
	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
)

// Decimal represents a decimal literal.
type Decimal struct {
	Value    dec128.Dec128
	ValuePos core.Pos
	Literal  string
}

func (e *Decimal) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Decimal) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Decimal) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Decimal) String() string {
	return e.Literal
}
