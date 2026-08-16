package core_test

import (
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/internal/require"
)

// TestIsMethodPureAppend checks the P12/P5-001 fix directly against the type descriptor: "append_in_place" must
// be reported impure (it mutates the receiver via Seq.Set) while "append" itself must be reported pure (it's now
// an unconditional copy on all three Seq-shaped types). This can't be observed by compiling/running a Kavun
// script — verified empirically during this fix that neither array's composite-literal receivers nor bytes'/
// runes' Bytes/Runes-typed results ever reach the optimizer's constant-folding path today (isFoldableExpr's
// MethodCall case requires a scalar-literal receiver, which composite array literals never are; safeValueToLiteral
// has no case for Bytes/Runes/Array results either) — so this is a direct check of the declared contract, not a
// fold-behavior regression test.
func TestIsMethodPureAppend(t *testing.T) {
	for _, typ := range []uint8{value.Array, value.Bytes, value.Runes} {
		descr := core.ValueTypes[typ]
		require.False(t, descr.IsMethodPure("append_in_place"), "type %d: append_in_place must be impure", typ)
		require.True(t, descr.IsMethodPure("append"), "type %d: append must be pure", typ)
	}
}
