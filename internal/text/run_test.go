package text_test

import (
	"testing"

	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/internal/text"
)

func TestTextSpellsRunes(t *testing.T) {
	cases := []struct {
		rs   []rune
		want bool
	}{
		{nil, true},
		{[]rune("héllo"), true},
		{[]rune{'a', 0xDCFF, 'b'}, true},        // a lone escape reads back as itself
		{[]rune{0xDCE2, 'x'}, true},             // a lead octet followed by ASCII stays an escape
		{[]rune{0xDCE2, 0xDC82}, true},          // a truncated sequence stays two escapes
		{[]rune{0xDCE2, 0xDC82, 0xDCA9}, false}, // E2 82 A9 reads as "₩"
		{[]rune{0xDCC3, 0xDCA9}, false},         // C3 A9 reads as "é"
		{[]rune{'a', 0xDCC3, 0xDCA9, 'b'}, false},
	}
	for _, c := range cases {
		require.True(t, text.TextSpellsRunes(text.EncodeText(c.rs), c.rs) == c.want, "%U", c.rs)
	}
	require.False(t, text.TextSpellsRunes("ab", []rune("a")))
	require.False(t, text.TextSpellsRunes("a", []rune("ab")))
}
