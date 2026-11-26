package core_test

import (
	"testing"
	"unsafe"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/signature"
)

// TestParseSignature pins what ParseSignature keeps of a template: the template verbatim and the pair package
// signature derives from it. The count sets and the parse errors themselves are signature's tests.
func TestParseSignature(t *testing.T) {
	templates := []string{
		"()", "(x)", "(width, fill)", "([sep])", "(width[, fill])", "(start, stop[, step])", "(a[, b, c])",
		"(a[, b[, c]])", "([a[, b]])", "(...parts)", "(first, ...rest)", "() | (components) | (start, stop[, step])",
		"(text) | (bands, min, max)", "(a) | (a, b, c, ...rest)", "(x_1, _y)", "(a, a_[, b]) | (b)", "(a) | (a, b)",
	}
	for _, tmpl := range templates {
		sh, err := signature.Parse(tmpl)
		require.NoError(t, err, tmpl)
		s, err := core.ParseSignature(tmpl)
		require.NoError(t, err, tmpl)
		require.Equal(t, sh.Arity, s.Arity(), tmpl)
		require.Equal(t, sh.Variadic, s.IsVariadic(), tmpl)
		require.Equal(t, tmpl, s.Template(), tmpl)
		require.True(t, s == core.MustSignature(tmpl), tmpl)
	}

	// a malformed template answers signature.Parse's error, naming the template
	_, err := core.ParseSignature("(a,b)")
	require.Error(t, err)
	_, want := signature.Parse("(a,b)")
	require.Equal(t, want.Error(), err.Error())
}

func TestMustSignaturePanics(t *testing.T) {
	defer func() {
		require.NotNil(t, recover(), "MustSignature on a malformed template must panic")
	}()
	core.MustSignature("(a,b)")
}

func TestSignatureOf(t *testing.T) {
	fixed := core.SignatureOf(2, false)
	require.Equal(t, 2, fixed.Arity())
	require.False(t, fixed.IsVariadic())
	require.Equal(t, "", fixed.Template())

	variadic := core.SignatureOf(1, true)
	require.Equal(t, 1, variadic.Arity())
	require.True(t, variadic.IsVariadic())
	require.Equal(t, "", variadic.Template())

	// the pair alone decides equality; a template makes a different Signature
	require.True(t, core.SignatureOf(1, true) == variadic)
	require.False(t, core.SignatureOf(1, false) == variadic)
	require.False(t, core.MustSignature("(x)") == core.SignatureOf(1, false))
}

func TestSignatureOfNegativeArityPanics(t *testing.T) {
	defer func() {
		require.NotNil(t, recover(), "SignatureOf with a negative arity must panic")
	}()
	core.SignatureOf(-1, false)
}

func TestSignatureZeroValue(t *testing.T) {
	var zero core.Signature
	require.True(t, zero == core.SignatureOf(0, false))
	require.Equal(t, 0, zero.Arity())
	require.False(t, zero.IsVariadic())
	require.Equal(t, "", zero.Template())
}

// TestSignatureSize pins the runtime entry's footprint on a 64-bit platform: the template's string header (16), the
// arity (8) and the floor span (8) — nothing parsed is retained (the count set lives in package signature).
func TestSignatureSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the size is pinned for 64-bit platforms")
	}
	require.Equal(t, 32, int(unsafe.Sizeof(core.Signature{})))
}

func TestArgCountError(t *testing.T) {
	cases := []struct {
		sig  core.Signature
		name string
		n    int
		want string
	}{
		{core.MustSignature("([sep])"), "join", 2, "wrong_num_arguments: (join) expected ([sep]), got 2 arguments"},
		{core.MustSignature("() | (components) | (start, stop[, step])"), "range", 4, "wrong_num_arguments: (range) expected () | (components) | (start, stop[, step]), got 4 arguments"},
		{core.MustSignature("(x)"), "len", 0, "wrong_num_arguments: (len) expected (x), got 0 arguments"},
		{core.MustSignature("()"), "upper", 1, "wrong_num_arguments: (upper) expected (), got 1 argument"},
		{core.SignatureOf(2, false), "f", 1, "wrong_num_arguments: (f) expected 2 arguments, got 1 argument"},
		{core.SignatureOf(1, false), "f", 0, "wrong_num_arguments: (f) expected 1 argument, got 0 arguments"},
		{core.SignatureOf(2, true), "f", 1, "wrong_num_arguments: (f) expected 2 or more arguments, got 1 argument"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, core.ArgCountError(tc.sig, tc.name, tc.n).Error())
	}
	// the dispatcher's floor check: exact for a fixed signature, at least the arity for a variadic one; the rest is
	// the body's to bound
	call := func(sig core.Signature, n int) error {
		f := core.NewBuiltinClosureValue("f", sig, func(core.VM, []core.Value) (core.Value, error) { return core.Undefined, nil })
		_, err := f.Call(nil, make([]core.Value, n))
		return err
	}
	require.NoError(t, call(core.MustSignature("(x)"), 1))
	require.Error(t, call(core.MustSignature("(x)"), 0))
	require.Error(t, call(core.MustSignature("(x)"), 2))
	require.NoError(t, call(core.MustSignature("()"), 0))
	require.Error(t, call(core.MustSignature("()"), 1))
	require.NoError(t, call(core.MustSignature("(x[, y])"), 5)) // a bound is the body's
	require.Error(t, call(core.MustSignature("(x[, y])"), 0))
	require.NoError(t, call(core.MustSignature("(...xs)"), 0))
	require.NoError(t, call(core.MustSignature("(...xs)"), 1000))
	require.NoError(t, call(core.SignatureOf(2, true), 2))
	require.Error(t, call(core.SignatureOf(2, true), 1))
	require.NoError(t, call(core.Signature{}, 0)) // the zero value is SignatureOf(0, false)
	require.Error(t, call(core.Signature{}, 1))
}
