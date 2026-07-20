package scalar

import "github.com/jokruger/kavun/core"

// Int represents an integer literal.
type Int struct {
	Value    int64
	ValuePos core.Pos
	Literal  string
}

func (e *Int) Pos() core.Pos {
	return e.ValuePos
}

func (e *Int) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Int) String() string {
	return e.Literal
}

func (e *Int) IsUndefinedLiteral() bool {
	return false
}

func (e *Int) IsScalarLiteral() bool {
	return true
}

func (e *Int) IsCompositeLiteral() bool {
	return false
}

func (e *Int) IsCallExpression() bool {
	return false
}

func (e *Int) LiteralToValue() (core.Value, bool) {
	return core.IntValue(e.Value), true
}
