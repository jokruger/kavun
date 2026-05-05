package value

import (
	"errors"
	"math"
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/tests/require"
)

func TestFormatErrorValue(t *testing.T) {
	mkErr := func(msg string) core.Value {
		return alloc.NewErrorValue(alloc.NewStringValue(msg))
	}

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default verb -> message text
		{"default", mkErr("boom"), "", "boom", false},
		{"default empty msg", mkErr(""), "", "", false},

		// 'v' verb -> source form
		{"v form", mkErr("boom"), "v", `error("boom")`, false},

		// generic fields with default verb (left-align by default)
		{"width left default", mkErr("err"), "10", "err       ", false},
		{"width right", mkErr("err"), ">10", "       err", false},
		{"width center", mkErr("err"), "^7", "  err  ", false},
		{"fill+align", mkErr("err"), "*<6", "err***", false},
		{"width on v", mkErr("x"), ">12v", `  error("x")`, false},

		// no truncation: width below body length is a no-op
		{"width too small", mkErr("hello"), "3", "hello", false},

		// unsupported: any other generic verb
		{"verb d", mkErr("x"), "d", "", true},
		{"verb s", mkErr("x"), "s", "", true},
		{"verb q", mkErr("x"), "q", "", true},

		// unsupported: tail form ('#'-tail sets Verb='#')
		{"tail empty", mkErr("x"), "#", "", true},
		{"tail payload", mkErr("x"), "#anything", "", true},
		{"tail with width", mkErr("x"), "10#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatBoolValue(t *testing.T) {
	T := core.True
	F := core.False

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default verb
		{"default true", T, "", "true", false},
		{"default false", F, "", "false", false},

		// 't' / 'v' verbs == default
		{"t true", T, "t", "true", false},
		{"t false", F, "t", "false", false},
		{"v true", T, "v", "true", false},
		{"v false", F, "v", "false", false},

		// 'T'
		{"T true", T, "T", "TRUE", false},
		{"T false", F, "T", "FALSE", false},

		// 'y' / 'Y'
		{"y true", T, "y", "yes", false},
		{"y false", F, "y", "no", false},
		{"Y true", T, "Y", "YES", false},
		{"Y false", F, "Y", "NO", false},

		// 'd'
		{"d true", T, "d", "1", false},
		{"d false", F, "d", "0", false},

		// generic width / fill / align (non-numeric defaults to left)
		{"width default left", T, "8", "true    ", false},
		{"width right", T, ">8", "    true", false},
		{"width center", F, "^7", " false ", false},
		{"fill+align", F, "*<7", "false**", false},
		{"width on Y", T, ">5Y", "  YES", false},
		{"width on d left", F, "3d", "0  ", false},
		{"width too small", T, "2T", "TRUE", false},

		// unsupported verbs
		{"verb s", T, "s", "", true},
		{"verb b", T, "b", "", true},
		{"verb x", T, "x", "", true},

		// tail form unsupported
		{"tail empty", T, "#", "", true},
		{"tail payload", F, "#anything", "", true},
		{"tail with width", T, "5#x", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatByteValue(t *testing.T) {
	bv := func(b byte) core.Value { return core.ByteValue(b) }

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default / d / v
		{"default 0", bv(0), "", "0", false},
		{"default 42", bv(42), "", "42", false},
		{"default 255", bv(255), "", "255", false},
		{"d 42", bv(42), "d", "42", false},
		{"v 42", bv(42), "v", "42", false},

		// sign on non-negative
		{"+ d", bv(5), "+d", "+5", false},
		{"space d", bv(5), " d", " 5", false},
		{"- d (no-op for byte)", bv(5), "-d", "5", false},
		{"+0", bv(0), "+", "+0", false},

		// width / right-align (numeric default)
		{"width 5", bv(7), "5d", "    7", false},
		{"width <", bv(7), "<5d", "7    ", false},
		{"width ^", bv(7), "^5d", "  7  ", false},

		// zero-pad shortcut
		{"05d", bv(7), "05d", "00007", false},
		{"+05d", bv(7), "+05d", "+0007", false},
		{" 05d", bv(7), " 05d", " 0007", false},
		{"05x prefix", bv(0xab), "#06x", "", true}, // generic verb + tail forbidden by parser
		{"06x", bv(0xab), "06x", "0x00ab", false},  // sign-aware split keeps prefix

		// grouping (decimal)
		{"grouping ,", bv(255), ",d", "255", false},
		{"grouping , width", bv(255), "10,d", "       255", false},

		// hex / oct / bin
		{"x 0", bv(0), "x", "0x0", false},
		{"x 255", bv(255), "x", "0xff", false},
		{"X 255", bv(255), "X", "0XFF", false},
		{"o 8", bv(8), "o", "0o10", false},
		{"b 5", bv(5), "b", "0b101", false},
		{"b 255", bv(255), "b", "0b11111111", false},

		// grouping '_' for non-decimal (every 4 digits)
		{"b _ 255", bv(255), "_b", "0b1111_1111", false},
		{"x _ 255", bv(255), "_x", "0xff", false}, // only 2 hex digits, no grouping triggered

		// 'c' verb (ASCII char)
		{"c A", bv('A'), "c", "A", false},
		{"c width", bv('A'), "3c", "A  ", false},

		// errors
		{"precision", bv(1), ".2d", "", true},
		{"z flag", bv(1), "zd", "", true},
		{"comma on hex", bv(255), ",x", "", true},
		{"sign on c", bv('A'), "+c", "", true},
		{"grouping on c", bv('A'), "_c", "", true},
		{"unknown verb", bv(1), "q", "", true},

		// tail form unsupported (verb == '#')
		{"tail empty", bv(1), "#", "", true},
		{"tail payload", bv(1), "#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return // parser already rejected (e.g. "#06x")
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatRuneValue(t *testing.T) {
	rv := func(r rune) core.Value { return core.RuneValue(r) }

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default / 'c'
		{"default A", rv('A'), "", "A", false},
		{"default snowman", rv(0x2603), "", "\u2603", false},
		{"c A", rv('A'), "c", "A", false},
		{"c snowman", rv(0x2603), "c", "\u2603", false},

		// 'd'
		{"d A", rv('A'), "d", "65", false},
		{"d snowman", rv(0x2603), "d", "9731", false},
		{"d sign +", rv('A'), "+d", "+65", false},
		{"d zero-pad", rv('A'), "05d", "00065", false},
		{"d width right", rv('A'), "5d", "   65", false},
		{"d grouping ,", rv(0x2603), ",d", "9,731", false},
		{"d grouping _", rv(0x2603), "_d", "9_731", false},

		// 'x' / 'X' (no 0x prefix per spec)
		{"x A", rv('A'), "x", "41", false},
		{"X A", rv('A'), "X", "41", false},
		{"x snowman", rv(0x2603), "x", "2603", false},
		{"X snowman", rv(0x2603), "X", "2603", false},
		{"x lowercase ff", rv(0xff), "x", "ff", false},
		{"X uppercase FF", rv(0xff), "X", "FF", false},
		{"x grouping _", rv(0x12345), "_x", "1_2345", false},
		{"x width zero-pad", rv('A'), "06x", "000041", false},

		// 'U'
		{"U A", rv('A'), "U", "U+0041", false},
		{"U snowman", rv(0x2603), "U", "U+2603", false},
		{"U high", rv(0x1F600), "U", "U+1F600", false},
		{"U width", rv('A'), "10U", "    U+0041", false},

		// 'q' / 'v'
		{"q A", rv('A'), "q", `'A'`, false},
		{"q tab", rv('\t'), "q", `'\t'`, false},
		{"v A", rv('A'), "v", `'A'`, false},
		{"q width", rv('A'), "5q", `'A'  `, false},

		// width / fill / align on default char
		{"c width", rv('A'), "5", "A    ", false},
		{"c right", rv('A'), ">5", "    A", false},
		{"c center", rv('A'), "*^5", "**A**", false},

		// errors
		{"precision", rv('A'), ".2c", "", true},
		{"z flag", rv('A'), "zd", "", true},
		{"comma on x", rv('A'), ",x", "", true},
		{"sign on c", rv('A'), "+c", "", true},
		{"grouping on c", rv('A'), "_c", "", true},
		{"sign on q", rv('A'), "+q", "", true},
		{"sign on U", rv('A'), "+U", "", true},
		{"zeropad on U", rv('A'), "08U", "", true},
		{"unknown verb", rv('A'), "k", "", true},

		// tail unsupported
		{"tail empty", rv('A'), "#", "", true},
		{"tail payload", rv('A'), "#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatIntValue(t *testing.T) {
	iv := func(i int64) core.Value { return core.IntValue(i) }

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default / d / v
		{"default 0", iv(0), "", "0", false},
		{"default 42", iv(42), "", "42", false},
		{"default -7", iv(-7), "", "-7", false},
		{"d 42", iv(42), "d", "42", false},
		{"v -7", iv(-7), "v", "-7", false},
		{"min int64", iv(math.MinInt64), "d", "-9223372036854775808", false},
		{"max int64", iv(math.MaxInt64), "d", "9223372036854775807", false},

		// sign
		{"+ pos", iv(5), "+d", "+5", false},
		{"+ neg", iv(-5), "+d", "-5", false},
		{"space pos", iv(5), " d", " 5", false},
		{"space neg", iv(-5), " d", "-5", false},
		{"- pos", iv(5), "-d", "5", false},
		{"+ zero", iv(0), "+", "+0", false},

		// width / align
		{"width 5", iv(7), "5d", "    7", false},
		{"width 5 neg", iv(-7), "5d", "   -7", false},
		{"left", iv(7), "<5d", "7    ", false},
		{"center", iv(7), "^5d", "  7  ", false},
		{"sign-aware", iv(-7), "=5d", "-   7", false},

		// zero-pad
		{"05d pos", iv(7), "05d", "00007", false},
		{"05d neg", iv(-7), "05d", "-0007", false},
		{"+05d", iv(7), "+05d", "+0007", false},
		{"06x", iv(0xab), "06x", "0x00ab", false},
		{"06x neg", iv(-1), "06x", "-0x001", false},

		// grouping decimal
		{"comma", iv(1234567), ",d", "1,234,567", false},
		{"underscore", iv(1234567), "_d", "1_234_567", false},
		{"comma neg", iv(-1234567), ",d", "-1,234,567", false},
		{"comma width", iv(1234), "10,d", "     1,234", false},

		// hex / oct / bin
		{"x 255", iv(255), "x", "0xff", false},
		{"X 255", iv(255), "X", "0XFF", false},
		{"o 8", iv(8), "o", "0o10", false},
		{"b 5", iv(5), "b", "0b101", false},

		// grouping '_' on non-decimal
		{"x _", iv(0xdeadbeef), "_x", "0xdead_beef", false},
		{"b _", iv(0xff), "_b", "0b1111_1111", false},

		// 'c' verb
		{"c A", iv('A'), "c", "A", false},
		{"c snowman", iv(0x2603), "c", "\u2603", false},
		{"c width", iv('A'), "3c", "A  ", false},

		// errors
		{"precision", iv(1), ".2d", "", true},
		{"z flag", iv(1), "zd", "", true},
		{"comma on hex", iv(255), ",x", "", true},
		{"sign on c", iv('A'), "+c", "", true},
		{"grouping on c", iv('A'), "_c", "", true},
		{"c negative", iv(-1), "c", "", true},
		{"c too large", iv(0x110000), "c", "", true},
		{"unknown verb", iv(1), "q", "", true},

		// tail unsupported
		{"tail empty", iv(1), "#", "", true},
		{"tail payload", iv(1), "#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatFloatValue(t *testing.T) {
	fv := func(f float64) core.Value { return core.FloatValue(f) }

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default ('g') / 'v'
		{"default 1.5", fv(1.5), "", "1.5", false},
		{"default 0", fv(0), "", "0", false},
		{"v 1.5", fv(1.5), "v", "1.5", false},
		{"default neg", fv(-2.5), "", "-2.5", false},

		// 'f'
		{"f default prec", fv(1.5), "f", "1.500000", false},
		{"f prec 2", fv(1.5), ".2f", "1.50", false},
		{"f prec 0", fv(1.5), ".0f", "2", false},
		{"f neg", fv(-3.14), ".2f", "-3.14", false},

		// 'e' / 'E'
		{"e default", fv(12345.6789), "e", "1.234568e+04", false},
		{"e prec 2", fv(12345.6789), ".2e", "1.23e+04", false},
		{"E prec 2", fv(12345.6789), ".2E", "1.23E+04", false},

		// 'g' / 'G'
		{"g 1234567.89", fv(1234567.89), "g", "1.23456789e+06", false},
		{"G 1234567.89", fv(1234567.89), "G", "1.23456789E+06", false},

		// '%'
		{"% default", fv(0.5), "%", "50.000000%", false},
		{"% prec 1", fv(0.125), ".1%", "12.5%", false},
		{"% neg", fv(-0.25), ".0%", "-25%", false},

		// sign
		{"+ pos", fv(1.5), "+f", "+1.500000", false},
		{"+ neg", fv(-1.5), "+f", "-1.500000", false},
		{"space pos", fv(1.5), " f", " 1.500000", false},

		// width / align
		{"width 10", fv(1.5), "10f", "  1.500000", false},
		{"left", fv(1.5), "<10f", "1.500000  ", false},
		{"center", fv(1.5), "^10f", " 1.500000 ", false},

		// zero-pad / sign-aware
		{"0 width", fv(1.5), "010.2f", "0000001.50", false},
		{"+0 width", fv(1.5), "+010.2f", "+000001.50", false},
		{"0 width neg", fv(-1.5), "010.2f", "-000001.50", false},

		// grouping
		{"comma f", fv(1234567.89), ",.2f", "1,234,567.89", false},
		{"underscore f", fv(1234567.89), "_.2f", "1_234_567.89", false},
		{"comma neg", fv(-1234.5), ",.1f", "-1,234.5", false},
		{"comma g", fv(1234567), ",.0f", "1,234,567", false},

		// 'z' coerce-zero
		{"z neg zero f", fv(-0.0), "zf", "0.000000", false},
		{"z rounds to zero", fv(-0.0001), ".2zf", "0.00", false},
		{"z without -0", fv(-1.5), ".1zf", "-1.5", false},
		{"z neg-zero g", fv(-0.0), "zg", "0", false},

		// special values
		{"NaN f", fv(math.NaN()), "f", "NaN", false},
		{"NaN F", fv(math.NaN()), "F", "NAN", false},
		{"+Inf", fv(math.Inf(1)), "f", "Inf", false},
		{"-Inf", fv(math.Inf(-1)), "f", "-Inf", false},
		{"+Inf upper", fv(math.Inf(1)), "F", "INF", false},
		{"+Inf with +", fv(math.Inf(1)), "+f", "+Inf", false},
		{"NaN with +", fv(math.NaN()), "+f", "NaN", false},
		{"NaN width", fv(math.NaN()), "5f", "  NaN", false},

		// errors
		{"unknown verb", fv(1), "x", "", true},
		{"unknown verb d", fv(1), "d", "", true},

		// tail unsupported
		{"tail empty", fv(1), "#", "", true},
		{"tail payload", fv(1), "#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}
