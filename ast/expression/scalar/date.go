package scalar

import (
	"github.com/jokruger/fin128/civil"

	"github.com/jokruger/kavun/core"
)

// Date represents a date literal (d"YYYY-MM-DD").
type Date struct {
	Value    civil.Date
	ValuePos core.Pos
	Literal  string
}

func (e *Date) Pos() core.Pos {
	return e.ValuePos
}

func (e *Date) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal) + 1) // +1 for the 'd' prefix
}

func (e *Date) String() string {
	return "d" + e.Literal
}

func (e *Date) IsUndefinedLiteral() bool {
	return false
}

func (e *Date) IsScalarLiteral() bool {
	return true
}

func (e *Date) IsCompositeLiteral() bool {
	return false
}

func (e *Date) IsCallExpression() bool {
	return false
}

func (e *Date) LiteralToValue() (core.Value, bool) {
	return core.DateValue(e.Value), true
}
