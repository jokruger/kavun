package core_test

import (
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/internal/require"
)

// TestIsMethodPureMutatingTwins checks every type's IsMethodPure directly against the type descriptor for each
// `_in_place` mutating twin implemented anywhere in the language. This can't be observed by compiling/running a
// Kavun script — verified empirically during the original P12/P5-001 fix that neither array's composite-literal
// receivers nor bytes'/runes' Bytes/Runes-typed results ever reach the optimizer's constant-folding path today
// (isFoldableExpr's MethodCall case requires a scalar-literal receiver, which composite array literals never
// are; safeValueToLiteral has no case for Bytes/Runes/Array results either) — so this is a direct check of the
// declared contract, not a fold-behavior regression test.
//
// Found and fixed 2026-08-17: array/bytes/runes' IsMethodPure only ever excluded "append_in_place" (stale since
// splice_in_place, sort_in_place, and reverse_in_place were added later without updating it), and dict's
// IsMethodPure blanket-returned true for everything, including the already-existing "delete_in_place" — the
// same class of bug the original append_in_place fix addressed, just not carried forward consistently. All four
// mutating twins are checked together here specifically so this doesn't happen a third time.
func TestIsMethodPureMutatingTwins(t *testing.T) {
	for _, typ := range []uint8{value.Array, value.Bytes, value.Runes} {
		descr := core.ValueTypes[typ]
		for _, mutating := range []string{"append_in_place", "splice_in_place", "sort_in_place", "reverse_in_place"} {
			require.False(t, descr.IsMethodPure(mutating), "type %d: %s must be impure", typ, mutating)
		}
		for _, pure := range []string{"append", "splice", "sort", "reverse", "copy", "copy_shallow", "freeze", "freeze_shallow"} {
			require.True(t, descr.IsMethodPure(pure), "type %d: %s must be pure", typ, pure)
		}
	}

	dictDescr := core.ValueTypes[value.Dict]
	require.False(t, dictDescr.IsMethodPure("delete_in_place"), "dict: delete_in_place must be impure")
	for _, pure := range []string{"delete", "copy", "copy_shallow", "freeze", "freeze_shallow", "record", "record_view"} {
		require.True(t, dictDescr.IsMethodPure(pure), "dict: %s must be pure", pure)
	}
}
