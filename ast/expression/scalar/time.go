package scalar

import (
	"time"

	"github.com/jokruger/kavun/core"
)

// Time represents a time literal (t"...").
type Time struct {
	Value    time.Time
	ValuePos core.Pos
	Literal  string
}

func (e *Time) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Time) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Time) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal) + 1) // +1 for the 't' prefix
}

func (e *Time) String() string {
	return "t" + e.Literal
}
