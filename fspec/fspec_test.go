package fspec_test

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

func TestParse(t *testing.T) {
	type want = fspec.FormatSpec
	cases := []struct {
		in   string
		ok   bool
		want want
	}{
		// empty / minimal
		{"", true, fspec.FormatSpec{}},
		{"d", true, fspec.FormatSpec{Verb: 'd'}},
		{"v", true, fspec.FormatSpec{Verb: 'v'}},

		// alignment (bare)
		{"<", true, fspec.FormatSpec{Align: fspec.AlignLeft}},
		{">", true, fspec.FormatSpec{Align: fspec.AlignRight}},
		{"^", true, fspec.FormatSpec{Align: fspec.AlignCenter}},
		{"=", true, fspec.FormatSpec{Align: fspec.AlignSign}},

		// fill + align
		{"*^10", true, fspec.FormatSpec{Fill: '*', Align: fspec.AlignCenter, Width: 10, HasWidth: true}},
		{" <5", true, fspec.FormatSpec{Fill: ' ', Align: fspec.AlignLeft, Width: 5, HasWidth: true}},
		{">10s", true, fspec.FormatSpec{Align: fspec.AlignRight, Width: 10, HasWidth: true, Verb: 's'}},
		{"<<", true, fspec.FormatSpec{Fill: '<', Align: fspec.AlignLeft}},
		{"==5", true, fspec.FormatSpec{Fill: '=', Align: fspec.AlignSign, Width: 5, HasWidth: true}},
		{"0<5", true, fspec.FormatSpec{Fill: '0', Align: fspec.AlignLeft, Width: 5, HasWidth: true}},  // '0' is fill, no ZeroPad
		{"+>5", true, fspec.FormatSpec{Fill: '+', Align: fspec.AlignRight, Width: 5, HasWidth: true}}, // '+' is fill, not sign
		{"★<5", true, fspec.FormatSpec{Fill: '★', Align: fspec.AlignLeft, Width: 5, HasWidth: true}},  // non-ASCII fill rune

		// sign
		{"+", true, fspec.FormatSpec{Sign: fspec.SignPlus}},
		{"-", true, fspec.FormatSpec{Sign: fspec.SignMinus}},
		{" ", true, fspec.FormatSpec{Sign: fspec.SignSpace}},
		{" 5d", true, fspec.FormatSpec{Sign: fspec.SignSpace, Width: 5, HasWidth: true, Verb: 'd'}},

		// width / zero-pad shortcut
		{"5d", true, fspec.FormatSpec{Width: 5, HasWidth: true, Verb: 'd'}},
		{"0", true, fspec.FormatSpec{Width: 0, HasWidth: true, ZeroPad: true, Fill: '0', Align: fspec.AlignSign}},
		{"00", true, fspec.FormatSpec{Width: 0, HasWidth: true, ZeroPad: true, Fill: '0', Align: fspec.AlignSign}},
		{"05d", true, fspec.FormatSpec{Width: 5, HasWidth: true, ZeroPad: true, Fill: '0', Align: fspec.AlignSign, Verb: 'd'}},
		{"+05d", true, fspec.FormatSpec{Sign: fspec.SignPlus, Width: 5, HasWidth: true, ZeroPad: true, Fill: '0', Align: fspec.AlignSign, Verb: 'd'}},
		{">05d", true, fspec.FormatSpec{Align: fspec.AlignRight, Width: 5, HasWidth: true, Verb: 'd'}}, // explicit align disables ZeroPad

		// grouping
		{"_", true, fspec.FormatSpec{Grouping: '_'}},
		{",", true, fspec.FormatSpec{Grouping: ','}},
		{"5,d", true, fspec.FormatSpec{Width: 5, HasWidth: true, Grouping: ',', Verb: 'd'}},
		{"5_x", true, fspec.FormatSpec{Width: 5, HasWidth: true, Grouping: '_', Verb: 'x'}},

		// precision
		{".3f", true, fspec.FormatSpec{Precision: 3, HasPrec: true, Verb: 'f'}},
		{".0f", true, fspec.FormatSpec{Precision: 0, HasPrec: true, Verb: 'f'}},
		{".5", true, fspec.FormatSpec{Precision: 5, HasPrec: true}},
		{"10,.2f", true, fspec.FormatSpec{Width: 10, HasWidth: true, Grouping: ',', Precision: 2, HasPrec: true, Verb: 'f'}},

		// '~' coerce-zero flag (formerly 'z')
		{"~", true, fspec.FormatSpec{CoerceZero: true}},
		{"~f", true, fspec.FormatSpec{CoerceZero: true, Verb: 'f'}},
		{"5~", true, fspec.FormatSpec{Width: 5, HasWidth: true, CoerceZero: true}},
		{".2~", true, fspec.FormatSpec{Precision: 2, HasPrec: true, CoerceZero: true}},
		{".2~f", true, fspec.FormatSpec{Precision: 2, HasPrec: true, CoerceZero: true, Verb: 'f'}},

		// '!' bare (no-prefix) flag
		{"!", true, fspec.FormatSpec{Bare: true}},
		{"!o", true, fspec.FormatSpec{Bare: true, Verb: 'o'}},
		{"!X", true, fspec.FormatSpec{Bare: true, Verb: 'X'}},
		{"5!o", true, fspec.FormatSpec{Width: 5, HasWidth: true, Bare: true, Verb: 'o'}},
		{"08!x", true, fspec.FormatSpec{Width: 8, HasWidth: true, ZeroPad: true, Fill: '0', Align: fspec.AlignSign, Bare: true, Verb: 'x'}},

		// flag order independence
		{"~!f", true, fspec.FormatSpec{CoerceZero: true, Bare: true, Verb: 'f'}},
		{"!~f", true, fspec.FormatSpec{CoerceZero: true, Bare: true, Verb: 'f'}},

		// duplicate flags rejected
		{"~~", false, fspec.FormatSpec{}},
		{"!!", false, fspec.FormatSpec{}},
		{"~!~", false, fspec.FormatSpec{}},

		// '%' verb
		{"%", true, fspec.FormatSpec{Verb: '%'}},
		{".2%", true, fspec.FormatSpec{Precision: 2, HasPrec: true, Verb: '%'}},
		{"10.2%", true, fspec.FormatSpec{Width: 10, HasWidth: true, Precision: 2, HasPrec: true, Verb: '%'}},

		// full stacks
		{"+010.3f", true, fspec.FormatSpec{Sign: fspec.SignPlus, Width: 10, HasWidth: true, ZeroPad: true, Fill: '0', Align: fspec.AlignSign, Precision: 3, HasPrec: true, Verb: 'f'}},
		{"*^+10,.2f", true, fspec.FormatSpec{Fill: '*', Align: fspec.AlignCenter, Sign: fspec.SignPlus, Width: 10, HasWidth: true, Grouping: ',', Precision: 2, HasPrec: true, Verb: 'f'}},

		// tail
		{"#date", true, fspec.FormatSpec{Verb: '#', Tail: "date"}},
		{"#", true, fspec.FormatSpec{Verb: '#', Tail: ""}},
		{"10#2006-01-02", true, fspec.FormatSpec{Width: 10, HasWidth: true, Verb: '#', Tail: "2006-01-02"}},
		{"##abc", true, fspec.FormatSpec{Verb: '#', Tail: "#abc"}},
		// verb + '#'-tail combinations are accepted by the parser; types decide whether to consume the tail.
		{"x#a#b", true, fspec.FormatSpec{Verb: 'x', Tail: "a#b"}},
		{"d#date", true, fspec.FormatSpec{Verb: 'd', Tail: "date"}},

		// errors
		{".f", false, fspec.FormatSpec{}},
		{".", false, fspec.FormatSpec{}},
		{"abc", false, fspec.FormatSpec{}},
		{"5dx", false, fspec.FormatSpec{}},
		{"5.", false, fspec.FormatSpec{}},
		{"{<5", false, fspec.FormatSpec{}}, // '{' not allowed as fill
		{"}<5", false, fspec.FormatSpec{}}, // '}' not allowed as fill
		{"{}", false, fspec.FormatSpec{}},
		{"  5", false, fspec.FormatSpec{}}, // sign then stray space
		{"  ", false, fspec.FormatSpec{}},
		{"_,", false, fspec.FormatSpec{}},  // double grouping
		{"5,_", false, fspec.FormatSpec{}}, // double grouping
		{"~.2", false, fspec.FormatSpec{}}, // flags must come after precision
		{"Q5", false, fspec.FormatSpec{}},  // verb not at end
		{"Q,", false, fspec.FormatSpec{}},
		{"5,5", false, fspec.FormatSpec{}},     // grouping then digits (not a verb)
		{"99999d", false, fspec.FormatSpec{}},  // width overflow (>MaxInt16)
		{".99999f", false, fspec.FormatSpec{}}, // precision overflow
	}
	for _, c := range cases {
		got, err := fspec.Parse(c.in)
		if c.ok {
			if err != nil {
				t.Errorf("Parse(%q): unexpected error: %v", c.in, err)
				continue
			}
			if got != c.want {
				t.Errorf("Parse(%q):\n got  %+v\n want %+v", c.in, got, c.want)
			}
		} else {
			if err == nil {
				t.Errorf("Parse(%q): expected error, got %+v", c.in, got)
			}
		}
	}
}

func TestParseTemplate_Literals(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain", "hello, world", "hello, world"},
		{"escaped braces", "a {{ b }} c", "a { b } c"},
		{"only escaped braces", "{{}}", "{}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tmpl, err := fspec.ParseTemplate(c.in)
			if err != nil {
				t.Fatalf("ParseTemplate(%q) error: %v", c.in, err)
			}
			if tmpl.Mode != fspec.TemplateModeUnset {
				t.Fatalf("expected mode unset, got %v", tmpl.Mode)
			}
			if c.want == "" {
				if len(tmpl.Segments) != 0 {
					t.Fatalf("expected no segments, got %+v", tmpl.Segments)
				}
				return
			}
			if len(tmpl.Segments) != 1 || tmpl.Segments[0].Kind != fspec.TemplateLiteral || tmpl.Segments[0].Literal != c.want {
				t.Fatalf("unexpected segments: %+v", tmpl.Segments)
			}
		})
	}
}

func TestParseTemplate_NamedAndIndexed(t *testing.T) {
	tmpl, err := fspec.ParseTemplate("hello {x} from {y}!")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Mode != fspec.TemplateModeNamed {
		t.Fatalf("expected Named mode, got %v", tmpl.Mode)
	}
	if len(tmpl.Segments) != 5 {
		t.Fatalf("expected 5 segments, got %d", len(tmpl.Segments))
	}
	if tmpl.Segments[1].Name != "x" || tmpl.Segments[3].Name != "y" {
		t.Fatalf("unexpected names: %+v", tmpl.Segments)
	}

	tmpl, err = fspec.ParseTemplate("hello {0} from {1}!")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Mode != fspec.TemplateModeIndexed {
		t.Fatalf("expected Indexed mode, got %v", tmpl.Mode)
	}
	if tmpl.Segments[1].Index != 0 || tmpl.Segments[3].Index != 1 {
		t.Fatalf("unexpected indices: %+v", tmpl.Segments)
	}
}

func TestParseTemplate_Spec(t *testing.T) {
	t.Run("literal spec", func(t *testing.T) {
		tmpl, err := fspec.ParseTemplate("v={x:.2f}")
		if err != nil {
			t.Fatal(err)
		}
		seg := tmpl.Segments[1]
		if !seg.HasSpec || seg.SpecIsRef {
			t.Fatalf("expected literal spec, got %+v", seg)
		}
		if !seg.Spec.HasPrec || seg.Spec.Precision != 2 || seg.Spec.Verb != 'f' {
			t.Fatalf("unexpected parsed spec: %+v", seg.Spec)
		}
	})
	t.Run("ref spec named", func(t *testing.T) {
		tmpl, err := fspec.ParseTemplate("v={x:{fmt}}")
		if err != nil {
			t.Fatal(err)
		}
		seg := tmpl.Segments[1]
		if !seg.HasSpec || !seg.SpecIsRef || seg.SpecRefName != "fmt" {
			t.Fatalf("unexpected: %+v", seg)
		}
	})
	t.Run("ref spec indexed", func(t *testing.T) {
		tmpl, err := fspec.ParseTemplate("v={0:{1}}")
		if err != nil {
			t.Fatal(err)
		}
		seg := tmpl.Segments[1]
		if !seg.HasSpec || !seg.SpecIsRef || seg.SpecRefIndex != 1 {
			t.Fatalf("unexpected: %+v", seg)
		}
	})
	t.Run("empty spec is no-op", func(t *testing.T) {
		tmpl, err := fspec.ParseTemplate("v={x:}")
		if err != nil {
			t.Fatal(err)
		}
		seg := tmpl.Segments[1]
		if !seg.HasSpec || seg.SpecIsRef {
			t.Fatalf("unexpected: %+v", seg)
		}
	})
}

func TestParseTemplate_Errors(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"bare close", "abc }"},
		{"unterminated", "abc {x"},
		{"empty placeholder", "x={}"},
		{"mixed modes", "{0} and {x}"},
		{"mixed modes reverse", "{x} and {0}"},
		{"expression inside", "{x+1}"},
		{"invalid name", "{1bad}"},
		{"nested ref with literal", "{x:>{w}}"},
		{"two refs", "{x:{a}{b}}"},
		{"ref empty", "{x:{}}"},
		{"unterminated ref", "{x:{abc"},
		{"bad fspec", "{x:zzz}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := fspec.ParseTemplate(c.in); err == nil {
				t.Fatalf("expected error for %q", c.in)
			}
		})
	}
}

func TestParseTemplate_Escapes(t *testing.T) {
	tmpl, err := fspec.ParseTemplate("set = {{ {x} }}")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Mode != fspec.TemplateModeNamed {
		t.Fatalf("mode: %v", tmpl.Mode)
	}
	if len(tmpl.Segments) != 3 {
		t.Fatalf("segs: %+v", tmpl.Segments)
	}
	if tmpl.Segments[0].Literal != "set = { " {
		t.Fatalf("first lit: %q", tmpl.Segments[0].Literal)
	}
	if tmpl.Segments[1].Name != "x" {
		t.Fatalf("placeholder: %+v", tmpl.Segments[1])
	}
	if tmpl.Segments[2].Literal != " }" {
		t.Fatalf("last lit: %q", tmpl.Segments[2].Literal)
	}
}
