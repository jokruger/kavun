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

func (e *Decimal) Pos() core.Pos {
	return e.ValuePos
}

func (e *Decimal) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Decimal) String() string {
	return e.Literal
}

func (e *Decimal) IsUndefinedLiteral() bool {
	return false
}

func (e *Decimal) IsScalarLiteral() bool {
	return true
}

func (e *Decimal) IsCompositeLiteral() bool {
	return false
}

func (e *Decimal) IsCallExpression() bool {
	return false
}

func (e *Decimal) LiteralToValue() (core.Value, bool) {
	return core.NewDecimalValue(e.Value), true
}
