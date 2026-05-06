package value

import (
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/jokruger/dec128"
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

		// 'T' universal type-name verb
		{"T", mkErr("x"), "T", "error", false},

		// generic fields with default verb (left-align by default)
		{"width left default", mkErr("err"), "10", "err       ", false},
		{"width right", mkErr("err"), ">10", "       err", false},
		{"width center", mkErr("err"), "^7", "  err  ", false},
		{"fill+align", mkErr("err"), "*<6", "err***", false},
		{"v ignores width", mkErr("x"), ">12v", `error("x")`, false},

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

		// 'd'
		{"d true", T, "d", "1", false},
		{"d false", F, "d", "0", false},

		// 'T' is the universal type-name verb
		{"T true", T, "T", "bool", false},
		{"T false", F, "T", "bool", false},

		// generic width / fill / align (non-numeric defaults to left)
		{"width default left", T, "8", "true    ", false},
		{"width right", T, ">8", "    true", false},
		{"width center", F, "^7", " false ", false},
		{"fill+align", F, "*<7", "false**", false},
		{"width on T", T, ">6T", "  bool", false},
		{"width on d left", F, "3d", "0  ", false},
		{"width too small", T, "2t", "true", false},

		// unsupported verbs
		{"verb s", T, "s", "", true},
		{"verb b", T, "b", "", true},
		{"verb x", T, "x", "", true},
		{"verb y", T, "y", "", true},
		{"verb Y", T, "Y", "", true},

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
		{"v 42", bv(42), "v", "byte(42)", false},
		{"T", bv(42), "T", "byte", false},

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
		{"T", rv('A'), "T", "rune", false},

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
		{"T", iv(0), "T", "int", false},
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
		{"T", fv(0), "T", "float", false},

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

func TestFormatDecimalValue(t *testing.T) {
	dv := func(str string) core.Value {
		d := dec128.FromString(str)
		return core.NewDecimalValue(d)
	}

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default (canonical, trim trailing zeros)
		{"default 1.23", dv("1.23"), "", "1.23", false},
		{"default 1.230", dv("1.230"), "", "1.23", false},
		{"default 0", dv("0"), "", "0", false},
		{"default neg", dv("-2.5"), "", "-2.5", false},
		{"default 100", dv("100"), "", "100", false},

		// 'v' source form
		{"v 1.23", dv("1.23"), "v", "1.23d", false},
		{"v -2.5", dv("-2.5"), "v", "-2.5d", false},
		{"v 1.230", dv("1.230"), "v", "1.23d", false}, // canonical underneath
		{"T", dv("1.0"), "T", "decimal", false},

		// 's' preserves source scale
		{"s 1.230", dv("1.230"), "s", "1.230", false},
		{"s 1.0", dv("1.0"), "s", "1.0", false},
		{"s int", dv("100"), "s", "100", false},

		// 'f' fixed precision (default 6)
		{"f default prec", dv("1.5"), "f", "1.500000", false},
		{"f prec 2", dv("1.5"), ".2f", "1.50", false},
		{"f prec 0", dv("1.5"), ".0f", "2", false},
		{"f rounds", dv("1.235"), ".2f", "1.24", false},
		{"f neg", dv("-3.14"), ".2f", "-3.14", false},

		// '%'
		{"% default", dv("0.5"), "%", "50.000000%", false},
		{"% prec 1", dv("0.125"), ".1%", "12.5%", false},
		{"% neg", dv("-0.25"), ".0%", "-25%", false},

		// 'e' / 'E' (via float64)
		{"e", dv("1234.5"), ".2e", "1.23e+03", false},
		{"E", dv("1234.5"), ".2E", "1.23E+03", false},

		// 'g' / 'G'
		{"g 1.5", dv("1.5"), "g", "1.5", false},
		{"G 1.5", dv("1.5"), "G", "1.5", false},

		// sign
		{"+ pos", dv("1.5"), "+", "+1.5", false},
		{"+ neg", dv("-1.5"), "+", "-1.5", false},
		{"space pos", dv("1.5"), " ", " 1.5", false},
		{"+ zero", dv("0"), "+", "+0", false},

		// width / align
		{"width 8", dv("1.5"), "8", "     1.5", false},
		{"left", dv("1.5"), "<8", "1.5     ", false},
		{"center", dv("1.5"), "^8", "  1.5   ", false},

		// zero-pad / sign-aware
		{"010.2f", dv("1.5"), "010.2f", "0000001.50", false},
		{"+010.2f", dv("1.5"), "+010.2f", "+000001.50", false},
		{"010.2f neg", dv("-1.5"), "010.2f", "-000001.50", false},

		// grouping
		{"comma f", dv("1234567.89"), ",.2f", "1,234,567.89", false},
		{"underscore f", dv("1234567.89"), "_.2f", "1_234_567.89", false},
		{"comma default", dv("1234567"), ",", "1,234,567", false},
		{"comma neg", dv("-1234.5"), ",.1f", "-1,234.5", false},
		{"comma s", dv("1234.50"), ",s", "1,234.50", false},

		// 'z' coerce-zero
		{"z neg-zero", dv("-0"), "zf", "0.000000", false},
		{"z rounds to zero", dv("-0.001"), ".2zf", "0.00", false},
		{"z without -0", dv("-1.5"), ".1zf", "-1.5", false},

		// NaN
		{"NaN default", dv("nope"), "f", "NaN", false},
		{"NaN F", dv("nope"), "F", "NAN", false},
		{"NaN width", dv("nope"), "5f", "  NaN", false},

		// errors
		{"unknown verb d", dv("1"), "d", "", true},
		{"unknown verb x", dv("1"), "x", "", true},

		// tail unsupported
		{"tail empty", dv("1"), "#", "", true},
		{"tail payload", dv("1"), "#foo", "", true},
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

func TestFormatTimeValue(t *testing.T) {
	loc := time.FixedZone("UTC-5", -5*3600)
	// 2026-03-04 13:05:09.123456 -0500 (Wed)
	tm := time.Date(2026, 3, 4, 13, 5, 9, 123456000, loc)
	tv := core.NewTimeValue(tm)

	cases := []struct {
		name    string
		spec    string
		want    string
		wantErr bool
	}{
		// default RFC 3339
		{"default", "", "2026-03-04T13:05:09.123456-05:00", false},

		// 'v' source form
		{"v", "v", `time("2026-03-04T13:05:09.123456-05:00")`, false},

		// 'T' universal type-name verb
		{"T", "T", "time", false},

		// named tails
		{"#", "#", "2026-03-04T13:05:09.123456-05:00", false},
		{"#iso", "#iso", "2026-03-04T13:05:09.123456-05:00", false},
		{"#date", "#date", "2026-03-04", false},
		{"#time", "#time", "13:05:09", false},
		{"#unix", "#unix", strconv.FormatInt(tm.Unix(), 10), false},
		{"#unixms", "#unixms", strconv.FormatInt(tm.UnixMilli(), 10), false},
		{"#rfc822", "#rfc822", tm.Format(time.RFC822), false},

		// strftime: simple
		{"strftime ymd", "#%Y-%m-%d", "2026-03-04", false},
		{"strftime hms 24h", "#%H:%M:%S", "13:05:09", false},
		{"strftime hms 12h", "#%I:%M:%S %p", "01:05:09 PM", false},
		{"strftime y2", "#%y", "26", false},
		{"strftime e", "#[%e]", "[ 4]", false},
		{"strftime month names", "#%B / %b", "March / Mar", false},
		{"strftime weekday", "#%A %a", "Wednesday Wed", false},
		{"strftime jday", "#%j", "063", false},
		{"strftime tz", "#%z %Z", "-0500 UTC-5", false},
		{"strftime micro", "#%f", "123456", false},
		{"strftime literal pct", "#100%%", "100%", false},
		{"strftime newline tab", "#a%nb%tc", "a\nb\tc", false},
		{"strftime unix", "#%s", strconv.FormatInt(tm.Unix(), 10), false},

		// strftime: combined like the example in the task
		{"combined", "#%Y-%m-%d %H:%M:%S", "2026-03-04 13:05:09", false},

		// width/fill/align (default left)
		{"width left", "20#date", "2026-03-04          ", false},
		{"width right", ">20#date", "          2026-03-04", false},
		{"width center", "*^12#date", "*2026-03-04*", false},

		// errors: unsupported generic fields
		{"sign", "+", "", true},
		{"precision", ".3", "", true},
		{"zeropad", "010", "", true},
		{"grouping", ",", "", true},
		{"z flag", "z", "", true},

		// errors: unknown verb
		{"verb d", "d", "", true},
		{"verb f", "f", "", true},

		// errors: bad strftime
		{"unknown directive", "#%Q", "", true},
		{"trailing pct", "#abc%", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := tv.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatStringValue(t *testing.T) {
	sv := core.NewStringValue("hello")
	mix := core.NewStringValue("h\u00e9llo") // 5 runes, 6 bytes
	withSpec := core.NewStringValue("a b/c") // for url-encode

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default + s + v + q
		{"default", sv, "", "hello", false},
		{"s", sv, "s", "hello", false},
		{"v", sv, "v", `"hello"`, false},
		{"q", sv, "q", `"hello"`, false},
		{"T", sv, "T", "string", false},
		{"q with newline", core.NewStringValue("a\nb"), "q", `"a\nb"`, false},

		// base64
		{"b std", sv, "b", "aGVsbG8=", false},
		{"B url no pad", sv, "B", "aGVsbG8", false},

		// hex
		{"x lower", sv, "x", "68656c6c6f", false},
		{"X upper", sv, "X", "68656C6C6F", false},

		// url component
		{"u", withSpec, "u", "a%20b%2Fc", false},
		{"u unreserved", core.NewStringValue("A-Z.a_z~0-9"), "u", "A-Z.a_z~0-9", false},

		// precision (rune-based)
		{"prec ascii", sv, ".3", "hel", false},
		{"prec multibyte", mix, ".3", "h\u00e9l", false},
		{"prec ge len", sv, ".10", "hello", false},
		{"prec on q", sv, ".3q", `"hel"`, false},

		// width / fill / align (default left)
		{"width left", sv, "10", "hello     ", false},
		{"width right", sv, ">10", "     hello", false},
		{"width center fill", sv, "*^9", "**hello**", false},
		{"width with prec", sv, "10.3", "hel       ", false},

		// 'v' ignores width
		{"v ignores width", sv, "10v", `"hello"`, false},

		// errors
		{"sign", sv, "+", "", true},
		{"zeropad", sv, "010", "", true},
		{"grouping comma", sv, ",", "", true},
		{"z flag", sv, "z", "", true},
		{"verb d", sv, "d", "", true},
		{"v with prec ignored", sv, ".3v", `"hello"`, false},
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
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatRunesValue(t *testing.T) {
	rv := core.NewRunesValue([]rune("hello"), false)
	mix := core.NewRunesValue([]rune("h\u00e9llo"), false)

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		{"default", rv, "", "hello", false},
		{"s", rv, "s", "hello", false},
		{"v source form", rv, "v", `u"hello"`, false},
		{"q", rv, "q", `"hello"`, false},
		{"T", rv, "T", "runes", false},
		{"b", rv, "b", "aGVsbG8=", false},
		{"B", rv, "B", "aGVsbG8", false},
		{"x", rv, "x", "68656c6c6f", false},
		{"X", rv, "X", "68656C6C6F", false},
		{"u", core.NewRunesValue([]rune("a b"), false), "u", "a%20b", false},

		// precision counts runes, not bytes
		{"prec multibyte", mix, ".3", "h\u00e9l", false},
		{"prec on x sees full byte hex of truncated runes", mix, ".2x", "68c3a9", false},

		// width default left
		{"width", rv, "8", "hello   ", false},
		{"width right", rv, ">8", "   hello", false},

		// errors
		{"sign", rv, "-", "", true},
		{"zeropad", rv, "08", "", true},
		{"verb f", rv, "f", "", true},
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
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatBytesValue(t *testing.T) {
	bv := core.NewBytesValue([]byte("hello"), false)
	mix := core.NewBytesValue([]byte("h\u00e9llo"), false) // 6 bytes
	bin := core.NewBytesValue([]byte{0x00, 0xff, 0x10}, false)

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		{"default", bv, "", "hello", false},
		{"s", bv, "s", "hello", false},
		{"v source form", bv, "v", `bytes([104, 101, 108, 108, 111])`, false},
		{"q", bv, "q", `"hello"`, false},
		{"T", bv, "T", "bytes", false},
		{"b", bv, "b", "aGVsbG8=", false},
		{"B", bv, "B", "aGVsbG8", false},
		{"x", bv, "x", "68656c6c6f", false},
		{"X", bv, "X", "68656C6C6F", false},
		{"x binary", bin, "x", "00ff10", false},
		{"u", core.NewBytesValue([]byte("a b/c"), false), "u", "a%20b%2Fc", false},

		// precision counts BYTES (not runes) for bytes
		{"prec bytes", mix, ".3", "h\xc3\xa9", false},
		{"prec ge len", bv, ".10", "hello", false},
		{"prec on x", bv, ".3x", "68656c", false},

		// width
		{"width", bv, "8", "hello   ", false},
		{"width right", bv, ">8", "   hello", false},

		// errors
		{"sign", bv, "+", "", true},
		{"zeropad", bv, "08", "", true},
		{"verb d", bv, "d", "", true},
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
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatArrayValue(t *testing.T) {
	av := core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
	}, false)
	mixed := core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.NewStringValue("hi"),
	}, false)
	empty := core.NewArrayValue(nil, false)

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		{"default", av, "", "[1, 2, 3]", false},
		{"v", av, "v", "[1, 2, 3]", false},
		{"T", av, "T", "array", false},
		{"empty", empty, "", "[]", false},
		{"nested string is quoted", mixed, "", `[1, "hi"]`, false},

		// width / align (default left)
		{"width left", av, "15", "[1, 2, 3]      ", false},
		{"width right", av, ">15", "      [1, 2, 3]", false},
		{"width center fill", av, "*^11", "*[1, 2, 3]*", false},

		// errors
		{"sign", av, "+", "", true},
		{"prec", av, ".3", "", true},
		{"zeropad", av, "010", "", true},
		{"grouping", av, ",", "", true},
		{"z", av, "z", "", true},
		{"verb d", av, "d", "", true},
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
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatRecordValue(t *testing.T) {
	rv := core.NewRecordValue(map[string]core.Value{
		"a": core.IntValue(1),
	}, false)

	// default and 'v' yield the same form
	for _, spec := range []string{"", "v"} {
		s, err := fspec.Parse(spec)
		require.NoError(t, err)
		got, ferr := rv.Format(s)
		require.NoError(t, ferr)
		require.Equal(t, `{"a": 1}`, got)
	}

	// width
	s, err := fspec.Parse("12")
	require.NoError(t, err)
	got, ferr := rv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, `{"a": 1}    `, got)

	// 'T' universal type-name verb
	s, err = fspec.Parse("T")
	require.NoError(t, err)
	got, ferr = rv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, "record", got)

	// errors
	for _, bad := range []string{"+", ".3", "010", ",", "z", "d"} {
		sp, err := fspec.Parse(bad)
		if err != nil {
			continue
		}
		_, ferr := rv.Format(sp)
		if ferr == nil {
			t.Fatalf("expected error for spec %q", bad)
		}
	}
}

func TestFormatDictValue(t *testing.T) {
	dv := core.NewDictValue(map[string]core.Value{
		"a": core.IntValue(1),
	}, false)

	// default: bare braces
	s, err := fspec.Parse("")
	require.NoError(t, err)
	got, ferr := dv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, `dict({"a": 1})`, got)

	// v: dict() wrapper
	s, err = fspec.Parse("v")
	require.NoError(t, err)
	got, ferr = dv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, `dict({"a": 1})`, got)

	// width on default
	s, err = fspec.Parse("12")
	require.NoError(t, err)
	got, ferr = dv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, `dict({"a": 1})`, got)

	// 'T' universal type-name verb
	s, err = fspec.Parse("T")
	require.NoError(t, err)
	got, ferr = dv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, "dict", got)

	// errors
	for _, bad := range []string{"+", ".3", "010", ",", "z", "d"} {
		sp, perr := fspec.Parse(bad)
		if perr != nil {
			continue
		}
		_, ferr := dv.Format(sp)
		if ferr == nil {
			t.Fatalf("expected error for spec %q", bad)
		}
	}
}

func TestFormatIntRangeValue(t *testing.T) {
	r1 := core.NewIntRangeValue(0, 10, 1)
	r2 := core.NewIntRangeValue(0, 10, 2)

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		{"default step1", r1, "", "range(0, 10)", false},
		{"default step2", r2, "", "range(0, 10, 2)", false},
		{"v step1", r1, "v", "range(0, 10)", false},
		{"v step2", r2, "v", "range(0, 10, 2)", false},
		{"T", r1, "T", "range", false},

		// width / align
		{"width left", r1, "15", "range(0, 10)   ", false},
		{"width right", r1, ">15", "   range(0, 10)", false},
		{"v ignores width fill", r2, "*^17v", "range(0, 10, 2)", false},

		// errors
		{"sign", r1, "+", "", true},
		{"prec", r1, ".3", "", true},
		{"zeropad", r1, "010", "", true},
		{"grouping", r1, ",", "", true},
		{"z", r1, "z", "", true},
		{"verb d", r1, "d", "", true},
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
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}
