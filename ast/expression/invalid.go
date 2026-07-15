package expression

import "github.com/jokruger/kavun/core"

// Invalid represents invalid expression.
type Invalid struct {
	From core.Pos
	To   core.Pos
}

func (e *Invalid) Pos() core.Pos {
	return e.From
}

func (e *Invalid) End() core.Pos {
	return e.To
}

func (e *Invalid) String() string {
	return "<invalid expression>"
}

func (e *Invalid) IsUndefinedLiteral() bool {
	return false
}

func (e *Invalid) IsScalarLiteral() bool {
	return false
}

func (e *Invalid) IsCompositeLiteral() bool {
	return false
}

func (e *Invalid) IsCallExpression() bool {
	return false
}

func (e *Invalid) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
