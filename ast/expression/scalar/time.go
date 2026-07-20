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

func (e *Time) Pos() core.Pos {
	return e.ValuePos
}

func (e *Time) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal) + 1) // +1 for the 't' prefix
}

func (e *Time) String() string {
	return "t" + e.Literal
}

func (e *Time) IsUndefinedLiteral() bool {
	return false
}

func (e *Time) IsScalarLiteral() bool {
	return true
}

func (e *Time) IsCompositeLiteral() bool {
	return false
}

func (e *Time) IsCallExpression() bool {
	return false
}

func (e *Time) LiteralToValue() (core.Value, bool) {
	return core.NewTimeValue(e.Value), true
}
