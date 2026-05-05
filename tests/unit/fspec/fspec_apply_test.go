package fspec

import (
	"testing"

	"github.com/jokruger/kavun/fspec"
)

func TestApplyGenerics(t *testing.T) {
	cases := []struct {
		name         string
		body         string
		spec         fspec.FormatSpec
		defaultAlign fspec.Align
		want         string
	}{
		// no width → identity
		{"no width", "abc", fspec.FormatSpec{}, fspec.AlignLeft, "abc"},
		{"no width but align set", "abc", fspec.FormatSpec{Align: fspec.AlignRight}, fspec.AlignLeft, "abc"},

		// body already meets/exceeds width
		{"exact fit", "abc", fspec.FormatSpec{Width: 3, HasWidth: true}, fspec.AlignLeft, "abc"},
		{"oversized body", "abcdef", fspec.FormatSpec{Width: 3, HasWidth: true}, fspec.AlignLeft, "abcdef"},
		{"width zero", "abc", fspec.FormatSpec{Width: 0, HasWidth: true}, fspec.AlignLeft, "abc"},

		// alignment - default fill (space)
		{"left default", "abc", fspec.FormatSpec{Width: 6, HasWidth: true, Align: fspec.AlignLeft}, fspec.AlignLeft, "abc   "},
		{"right default", "abc", fspec.FormatSpec{Width: 6, HasWidth: true, Align: fspec.AlignRight}, fspec.AlignLeft, "   abc"},
		{"center even", "ab", fspec.FormatSpec{Width: 6, HasWidth: true, Align: fspec.AlignCenter}, fspec.AlignLeft, "  ab  "},
		{"center odd extra-on-right", "a", fspec.FormatSpec{Width: 4, HasWidth: true, Align: fspec.AlignCenter}, fspec.AlignLeft, " a  "},

		// alignment - custom fill
		{"left star", "abc", fspec.FormatSpec{Width: 6, HasWidth: true, Align: fspec.AlignLeft, Fill: '*'}, fspec.AlignLeft, "abc***"},
		{"right star", "abc", fspec.FormatSpec{Width: 6, HasWidth: true, Align: fspec.AlignRight, Fill: '*'}, fspec.AlignLeft, "***abc"},
		{"center star", "ab", fspec.FormatSpec{Width: 6, HasWidth: true, Align: fspec.AlignCenter, Fill: '*'}, fspec.AlignLeft, "**ab**"},
		{"non-ASCII fill", "ab", fspec.FormatSpec{Width: 5, HasWidth: true, Align: fspec.AlignCenter, Fill: '★'}, fspec.AlignLeft, "★ab★★"},

		// default alignment fallback
		{"unset align uses defaultAlign=Left", "x", fspec.FormatSpec{Width: 4, HasWidth: true}, fspec.AlignLeft, "x   "},
		{"unset align uses defaultAlign=Right", "42", fspec.FormatSpec{Width: 5, HasWidth: true}, fspec.AlignRight, "   42"},
		{"unset align defaultAlign=None ⇒ Left", "x", fspec.FormatSpec{Width: 3, HasWidth: true}, fspec.AlignNone, "x  "},

		// sign-aware (AlignSign)
		{"sign-aware no sign", "123", fspec.FormatSpec{Width: 6, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, "000123"},
		{"sign-aware plus", "+123", fspec.FormatSpec{Width: 7, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, "+000123"},
		{"sign-aware minus", "-42", fspec.FormatSpec{Width: 6, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, "-00042"},
		{"sign-aware space", " 7", fspec.FormatSpec{Width: 5, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, " 0007"},
		{"sign-aware hex prefix", "0x2A", fspec.FormatSpec{Width: 8, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, "0x00002A"},
		{"sign-aware sign+hex prefix", "-0x2A", fspec.FormatSpec{Width: 9, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, "-0x00002A"},
		{"sign-aware bin prefix", "0b101", fspec.FormatSpec{Width: 8, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, "0b000101"},
		{"sign-aware oct prefix", "0o17", fspec.FormatSpec{Width: 7, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, "0o00017"},
		{"sign-aware uppercase hex prefix", "0XFF", fspec.FormatSpec{Width: 6, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, "0X00FF"},

		// ZeroPad shortcut already encoded as Fill='0', Align='='
		{"zero-pad shortcut", "42", fspec.FormatSpec{Width: 5, HasWidth: true, ZeroPad: true, Fill: '0', Align: fspec.AlignSign}, fspec.AlignRight, "00042"},
		{"zero-pad with sign", "-7", fspec.FormatSpec{Width: 5, HasWidth: true, ZeroPad: true, Fill: '0', Align: fspec.AlignSign}, fspec.AlignRight, "-0007"},

		// runes counted, not bytes
		{"multibyte body", "héllo", fspec.FormatSpec{Width: 7, HasWidth: true, Align: fspec.AlignLeft}, fspec.AlignLeft, "héllo  "},
		{"multibyte body right", "héllo", fspec.FormatSpec{Width: 7, HasWidth: true, Align: fspec.AlignRight}, fspec.AlignLeft, "  héllo"},
		{"multibyte body already wider", "héllo", fspec.FormatSpec{Width: 5, HasWidth: true, Align: fspec.AlignLeft}, fspec.AlignLeft, "héllo"},

		// empty body
		{"empty body", "", fspec.FormatSpec{Width: 3, HasWidth: true, Align: fspec.AlignLeft}, fspec.AlignLeft, "   "},
		{"empty body sign-aware", "", fspec.FormatSpec{Width: 3, HasWidth: true, Align: fspec.AlignSign, Fill: '0'}, fspec.AlignRight, "000"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := fspec.ApplyGenerics(c.body, c.spec, c.defaultAlign)
			if got != c.want {
				t.Errorf("ApplyGenerics(%q) = %q, want %q", c.body, got, c.want)
			}
		})
	}
}
