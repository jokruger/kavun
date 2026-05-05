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
