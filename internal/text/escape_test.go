package text_test

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/internal/text"
)

// TextFirst walked from the left and TextLast walked from the right must both reproduce DecodeText exactly, on
// every kind of octet sequence: valid text, a literal U+FFFD, stray continuation and lead octets, truncated and
// overlong sequences, encoded surrogates. This is what makes the string members' symbol-at-a-time reading exact —
// a fact about the toolchain in use, not reasoning about it.
func TestTextFirstLastAgreeWithDecodeText(t *testing.T) {
	samples := []string{
		"", "a", "héllo", "�", "a�b", "😀",
		"\xff", "\x80", "\xc3", "\xe2\x82", "a\xe2\x82", "\xe2\x82\xac\xac", "\xc3\xa9\xa9",
		"\xed\xa0\x80", "\xf0\x9f\x98", "\xf0\xf0\x9f\x98\x80", "\xc0\xaf", "\xe0\x80\x80", "\xf4\x90\x80\x80",
	}
	atoms := []byte{'a', ' ', 0x00, 0x80, 0xa9, 0xac, 0xbf, 0xc3, 0xe2, 0x82, 0xed, 0xa0, 0xf0, 0x9f, 0x98, 0xff}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 3000; i++ {
		b := make([]byte, rng.Intn(10))
		for j := range b {
			b[j] = atoms[rng.Intn(len(atoms))]
		}
		samples = append(samples, string(b))
	}

	for _, s := range samples {
		want := text.DecodeText(s)

		var fwd []rune
		for rest := s; len(rest) > 0; {
			r, w := text.TextFirst(rest)
			require.True(t, w > 0 && w <= len(rest), "TextFirst width %d on %q", w, rest)
			fwd = append(fwd, r)
			rest = rest[w:]
		}
		require.True(t, slices.Equal(want, fwd), "forward walk of %q: want %U, got %U", s, want, fwd)

		var bwd []rune
		for rest := s; len(rest) > 0; {
			r, w := text.TextLast(rest)
			require.True(t, w > 0 && w <= len(rest), "TextLast width %d on %q", w, rest)
			bwd = append([]rune{r}, bwd...)
			rest = rest[:len(rest)-w]
		}
		require.True(t, slices.Equal(want, bwd), "backward walk of %q: want %U, got %U", s, want, bwd)
	}

	r, w := text.TextFirst("")
	require.Equal(t, rune(0), r)
	require.Equal(t, 0, w)
	r, w = text.TextLast("")
	require.Equal(t, rune(0), r)
	require.Equal(t, 0, w)
}
