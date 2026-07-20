package composite

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// RecordElement represents a record element.
type RecordElement struct {
	Key      string
	KeyPos   core.Pos
	ColonPos core.Pos
	Value    ast.Expression
}

func (e *RecordElement) Pos() core.Pos {
	return e.KeyPos
}

func (e *RecordElement) End() core.Pos {
	return e.Value.End()
}

func (e *RecordElement) String() string {
	return e.Key + ": " + e.Value.String()
}

func (e *RecordElement) IsUndefinedLiteral() bool {
	return false
}

func (e *RecordElement) IsScalarLiteral() bool {
	return false
}

func (e *RecordElement) IsCompositeLiteral() bool {
	return false
}

func (e *RecordElement) IsCallExpression() bool {
	return false
}

func (e *RecordElement) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
