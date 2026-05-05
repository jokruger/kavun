package value

import (
	"errors"
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
