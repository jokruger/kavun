package core_test

import (
	"reflect"
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/internal/require"
)

// An array's runes reading takes every element through its own rune conversion: a byte is read as text (ASCII as
// itself, a high octet as its escape U+DC80–U+DCFF, never its Latin-1 code point), an int must be in the rune
// domain, and one element without a rune reading refuses the whole array.
func TestArrayAsRunes(t *testing.T) {
	arr := func(vs ...core.Value) core.Value { return core.NewArrayValue(vs, false) }

	rs, ok := arr(core.ByteValue(0xE9), core.ByteValue('A'), core.RuneValue(0xE9), core.IntValue(0x41)).AsRunes()
	require.True(t, ok)
	require.True(t, reflect.DeepEqual([]rune{0xDCE9, 'A', 0xE9, 'A'}, rs), "%U", rs)

	for _, bad := range []core.Value{core.IntValue(0xD800), core.IntValue(-1), core.IntValue(0x110000), core.NewStringValue("a")} {
		_, ok := arr(core.RuneValue('a'), bad).AsRunes()
		require.False(t, ok, bad.String())
	}
	rs, ok = arr(core.IntValue(0xDCFF)).AsRunes() // an escape is in the rune domain
	require.True(t, ok)
	require.True(t, reflect.DeepEqual([]rune{0xDCFF}, rs))
}
