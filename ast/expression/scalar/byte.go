package scalar

import "github.com/jokruger/kavun/core"

// Byte represents a byte literal.
type Byte struct {
	Value    byte
	ValuePos core.Pos
	Literal  string
}

func (e *Byte) Pos() core.Pos {
	return e.ValuePos
}

func (e *Byte) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal) + 1) // +1 for the 'b' prefix
}

func (e *Byte) String() string {
	return "b" + e.Literal
}

func (e *Byte) IsUndefinedLiteral() bool {
	return false
}

func (e *Byte) IsScalarLiteral() bool {
	return true
}

func (e *Byte) IsCompositeLiteral() bool {
	return false
}

func (e *Byte) IsCallExpression() bool {
	return false
}

func (e *Byte) LiteralToValue() (core.Value, bool) {
	return core.ByteValue(e.Value), true
}
