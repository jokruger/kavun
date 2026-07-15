package scalar

import "github.com/jokruger/kavun/core"

// String represents a string literal.
type String struct {
	Value    string
	ValuePos core.Pos
	Literal  string
}

func (e *String) Pos() core.Pos {
	return e.ValuePos
}

func (e *String) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *String) String() string {
	return e.Literal
}

func (e *String) IsUndefinedLiteral() bool {
	return false
}

func (e *String) IsScalarLiteral() bool {
	return true
}

func (e *String) IsCompositeLiteral() bool {
	return false
}

func (e *String) IsCallExpression() bool {
	return false
}
