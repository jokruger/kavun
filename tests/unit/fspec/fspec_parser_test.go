package fspec

import (
	"testing"

	"github.com/jokruger/kavun/fspec"
)

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
		{"0<5", true, fspec.FormatSpec{Fill: '0', Align: fspec.AlignLeft, Width: 5, HasWidth: true}},        // '0' is fill, no ZeroPad
		{"+>5", true, fspec.FormatSpec{Fill: '+', Align: fspec.AlignRight, Width: 5, HasWidth: true}},      // '+' is fill, not sign
		{"★<5", true, fspec.FormatSpec{Fill: '★', Align: fspec.AlignLeft, Width: 5, HasWidth: true}},       // non-ASCII fill rune

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

		// z flag
		{"z", true, fspec.FormatSpec{CoerceZero: true}},
		{"zf", true, fspec.FormatSpec{CoerceZero: true, Verb: 'f'}},
		{"5z", true, fspec.FormatSpec{Width: 5, HasWidth: true, CoerceZero: true}},
		{".2z", true, fspec.FormatSpec{Precision: 2, HasPrec: true, CoerceZero: true}},
		{".2zf", true, fspec.FormatSpec{Precision: 2, HasPrec: true, CoerceZero: true, Verb: 'f'}},

		// full stacks
		{"+010.3f", true, fspec.FormatSpec{Sign: fspec.SignPlus, Width: 10, HasWidth: true, ZeroPad: true, Fill: '0', Align: fspec.AlignSign, Precision: 3, HasPrec: true, Verb: 'f'}},
		{"*^+10,.2f", true, fspec.FormatSpec{Fill: '*', Align: fspec.AlignCenter, Sign: fspec.SignPlus, Width: 10, HasWidth: true, Grouping: ',', Precision: 2, HasPrec: true, Verb: 'f'}},

		// tail
		{"#date", true, fspec.FormatSpec{Tail: "date"}},
		{"#", true, fspec.FormatSpec{Tail: ""}},
		{"10#2006-01-02", true, fspec.FormatSpec{Width: 10, HasWidth: true, Tail: "2006-01-02"}},
		{"x#a#b", true, fspec.FormatSpec{Verb: 'x', Tail: "a#b"}},
		{"##abc", true, fspec.FormatSpec{Tail: "#abc"}},

		// errors
		{".f", false, fspec.FormatSpec{}},
		{".", false, fspec.FormatSpec{}},
		{"abc", false, fspec.FormatSpec{}},
		{"5dx", false, fspec.FormatSpec{}},
		{"5.", false, fspec.FormatSpec{}},
		{"{<5", false, fspec.FormatSpec{}},  // '{' not allowed as fill
		{"}<5", false, fspec.FormatSpec{}},  // '}' not allowed as fill
		{"{}", false, fspec.FormatSpec{}},
		{"  5", false, fspec.FormatSpec{}},  // sign then stray space
		{"  ", false, fspec.FormatSpec{}},
		{"_,", false, fspec.FormatSpec{}},   // double grouping
		{"5,_", false, fspec.FormatSpec{}},  // double grouping
		{"z.2", false, fspec.FormatSpec{}},  // z must come after precision
		{"Q5", false, fspec.FormatSpec{}},   // verb not at end
		{"Q,", false, fspec.FormatSpec{}},
		{"5,5", false, fspec.FormatSpec{}},  // grouping then digits (not a verb)
		{"99999d", false, fspec.FormatSpec{}}, // width overflow (>MaxInt16)
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
