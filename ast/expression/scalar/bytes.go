package scalar

import "github.com/jokruger/kavun/core"

// Bytes represents a bytes string literal (b"...").
type Bytes struct {
	Value    []byte
	ValuePos core.Pos
	Literal  string
}

func (e *Bytes) Pos() core.Pos {
	return e.ValuePos
}

func (e *Bytes) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal) + 1) // +1 for the 'b' prefix
}

func (e *Bytes) String() string {
	return "b" + e.Literal
}

func (e *Bytes) IsUndefinedLiteral() bool {
	return false
}

func (e *Bytes) IsScalarLiteral() bool {
	return true
}

func (e *Bytes) IsCompositeLiteral() bool {
	return false
}

func (e *Bytes) IsCallExpression() bool {
	return false
}

func (e *Bytes) LiteralToValue() (core.Value, bool) {
	return core.NewBytesValue(e.Value, true), true
}
