package scalar

import "github.com/jokruger/kavun/core"

// Bool represents a boolean literal.
type Bool struct {
	Value    bool
	ValuePos core.Pos
	Literal  string
}

func (e *Bool) Pos() core.Pos {
	return e.ValuePos
}

func (e *Bool) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *Bool) String() string {
	return e.Literal
}

func (e *Bool) IsUndefinedLiteral() bool {
	return false
}

func (e *Bool) IsScalarLiteral() bool {
	return true
}

func (e *Bool) IsCompositeLiteral() bool {
	return false
}

func (e *Bool) IsCallExpression() bool {
	return false
}

func (e *Bool) LiteralToValue() (core.Value, bool) {
	if e.Value {
		return core.True, true
	}
	return core.False, true
}
