package scalar

import (
	"fmt"

	"github.com/jokruger/kavun/core"
)

// Range represents a folded constant range value. Unlike the other scalar literal nodes, it never comes from
// source text — it only ever replaces an eligible *expression.Range or range(...) *expression.Call subtree during
// constant folding (see compiler/optimizer_impl.go safeValueToLiteral), carrying the already-computed value.
type Range struct {
	Value    core.IntRange
	ValuePos core.Pos
}

func (e *Range) Pos() core.Pos {
	return e.ValuePos
}

func (e *Range) End() core.Pos {
	return e.ValuePos + 1
}

func (e *Range) String() string {
	if e.Value.Step == 1 {
		return fmt.Sprintf("range(%d, %d)", e.Value.Start, e.Value.Stop)
	}
	return fmt.Sprintf("range(%d, %d, %d)", e.Value.Start, e.Value.Stop, e.Value.Step)
}

func (e *Range) IsUndefinedLiteral() bool {
	return false
}

func (e *Range) IsScalarLiteral() bool {
	return true
}

func (e *Range) IsCompositeLiteral() bool {
	return false
}

func (e *Range) IsCallExpression() bool {
	return false
}

func (e *Range) LiteralToValue() (core.Value, bool) {
	return core.NewIntRangeValue(e.Value.Start, e.Value.Stop, e.Value.Step), true
}
