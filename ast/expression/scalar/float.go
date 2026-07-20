package scalar

import "github.com/jokruger/kavun/core"

// Float represents a floating point literal.
type Float struct {
	Value    float64
	ValuePos core.Pos
	Literal  string
}

func (e *Float) Pos() core.Pos {
	return e.ValuePos
}

func (e *Float) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Float) String() string {
	return e.Literal
}

func (e *Float) IsUndefinedLiteral() bool {
	return false
}

func (e *Float) IsScalarLiteral() bool {
	return true
}

func (e *Float) IsCompositeLiteral() bool {
	return false
}

func (e *Float) IsCallExpression() bool {
	return false
}

func (e *Float) LiteralToValue() (core.Value, bool) {
	return core.FloatValue(e.Value), true
}
