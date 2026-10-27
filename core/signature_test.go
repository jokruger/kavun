package core_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/internal/require"
)

// acceptedUpTo lists the counts in [0, limit] a Signature accepts.
func acceptedUpTo(s core.Signature, limit int) []int {
	var out []int
	for k := 0; k <= limit; k++ {
		if s.Accepts(k) {
			out = append(out, k)
		}
	}
	return out
}

func TestParseSignature(t *testing.T) {
	const limit = 12
	open := func(from int) []int { // every count from `from` up to the probe limit
		var out []int
		for k := from; k <= limit; k++ {
			out = append(out, k)
		}
		return out
	}

	cases := []struct {
		template string
		arity    int
		variadic bool
		accepts  []int
	}{
		{"()", 0, false, []int{0}},
		{"(x)", 1, false, []int{1}},
		{"(width, fill)", 2, false, []int{2}},
		{"([sep])", 0, true, []int{0, 1}},
		{"(width[, fill])", 1, true, []int{1, 2}},
		{"(start, stop[, step])", 2, true, []int{2, 3}},
		{"(a[, b, c])", 1, true, []int{1, 3}},
		{"(a[, b[, c]])", 1, true, []int{1, 2, 3}},
		{"([a[, b]])", 0, true, []int{0, 1, 2}},
		{"(...parts)", 0, true, open(0)},
		{"(first, ...rest)", 1, true, open(1)},
		{"() | (components) | (start, stop[, step])", 0, true, []int{0, 1, 2, 3}},
		{"(text) | (bands, min, max)", 1, true, []int{1, 3}},
		{"(a) | (a, b, c, ...rest)", 1, true, append([]int{1}, open(3)...)},
		{"(x_1, _y)", 2, false, []int{2}},
		{"(a, a_[, b]) | (b)", 1, true, []int{1, 2, 3}},
		{"(a) | (a, b)", 1, true, []int{1, 2}},
	}
	for _, tc := range cases {
		s, err := core.ParseSignature(tc.template)
		require.NoError(t, err, tc.template)
		require.Equal(t, tc.arity, s.Arity(), tc.template)
		require.Equal(t, tc.variadic, s.IsVariadic(), tc.template)
		require.Equal(t, tc.template, s.Template(), tc.template)
		got := acceptedUpTo(s, limit)
		require.True(t, slices.Equal(tc.accepts, got), "%s accepts %v", tc.template, got)
		require.False(t, s.Accepts(-1), tc.template)

		m := core.MustSignature(tc.template)
		require.Equal(t, s.Arity(), m.Arity(), tc.template)
		require.Equal(t, s.IsVariadic(), m.IsVariadic(), tc.template)
	}

	require.True(t, core.MustSignature("(...parts)").Accepts(1000))
	require.True(t, core.MustSignature("(first, ...rest)").Accepts(1000))
	require.False(t, core.MustSignature("(a[, b[, c]])").Accepts(1000))
}

func TestParseSignatureErrors(t *testing.T) {
	cases := []struct {
		name     string
		template string
		problem  string // a fragment the error must carry
	}{
		{"empty template", "", "empty template"},
		{"no parentheses", "a, b", "parenthesised"},
		{"no closing parenthesis", "(a", "expected `)`"},
		{"no opening parenthesis", "a)", "parenthesised"},
		{"comma without space", "(a,b)", "separated by `, `"},
		{"space inside parentheses", "( a )", "parameter name"},
		{"trailing space", "(a )", "expected `)`"},
		{"space before comma", "(a ,b)", "expected `)`"},
		{"two spaces after comma", "(a,  b)", "parameter name"},
		{"pipe without space after", "(a) |(b)", "` | `"},
		{"pipe without space before", "(a)| (b)", "` | `"},
		{"pipe without spaces", "(a)|(b)", "` | `"},
		{"trailing pipe", "(a) | ", "parenthesised"},
		{"upper-case name", "(A)", "parameter name"},
		{"upper-case in name", "(aB)", "invalid parameter name"},
		{"name starts with a digit", "(1a)", "parameter name"},
		{"hyphen in name", "(a-b)", "invalid parameter name"},
		{"trailing comma", "(a, )", "parameter name"},
		{"trailing comma, no space", "(a,)", "separated by `, `"},
		{"leading comma", "(, a)", "parameter name"},
		{"empty optional group", "([])", "empty optional group"},
		{"empty continued group", "(a[, ])", "empty optional group"},
		{"group without continuation", "(a[b])", "`[, `"},
		{"group at start with leading comma", "([, a])", "no leading `, `"},
		{"group after a comma", "(a, [b])", "parameter name"},
		{"group nested at group start", "([[a]])", "must begin with a parameter"},
		{"item after a group", "(a[, b], c)", "last thing in its list"},
		{"item after a start group", "([a], b)", "last thing in its list"},
		{"item after a nested group", "(a[, b[, c], d])", "last thing in its list"},
		{"rest not last", "(...a, b)", "rest must be the last"},
		{"rest before a group", "(...a[, b])", "rest must be the last"},
		{"rest inside a group", "(a[, ...b])", "inside an optional group"},
		{"rest without a name", "(...)", "parameter name"},
		{"duplicate name", "(a, a)", "duplicate parameter name"},
		{"duplicate name in group", "(a[, a])", "duplicate parameter name"},
		{"duplicate rest name", "(a, ...a)", "duplicate parameter name"},
		{"duplicate reading", "(a) | (a)", "redundant reading (a): every count it accepts is accepted by another reading at offset 0"},
		{"duplicate reading, not adjacent", "(a) | () | (a)", "redundant reading (a): every count it accepts is accepted by another reading at offset 0"},
		{"same count, other names", "(a) | (b)", "redundant reading (a)"},
		{"same count, swapped names", "(a, b) | (b, a)", "redundant reading (a, b)"},
		{"two rests from the same start", "(...a) | (...b)", "redundant reading (...a)"},
		{"fixed count under a rest", "(a, b, c) | (...rest)", "redundant reading (a, b, c)"},
		{"fixed count under a later rest", "(...rest) | (a, b, c)", "redundant reading (a, b, c): every count it accepts is accepted by another reading at offset 12"},
		{"count under a rest of the same start", "(x) | (x, ...rest)", "redundant reading (x)"},
		{"empty reading under an optional group", "([a]) | ()", "redundant reading ()"},
		{"optional group spelled out", "(a[, b]) | (a, b)", "redundant reading (a, b)"},
		{"fixed count under a lower rest", "(a, ...r) | (a, b)", "redundant reading (a, b)"},
		{"unclosed group", "(a[, b)", "unbalanced `[`"},
		{"unclosed group at end", "(a[, b", "unbalanced `[`"},
		{"extra closing bracket", "(a])", "expected `)`"},
		{"extra closing bracket after group", "(a[, b]])", "expected `)`"},
		{"extra closing parenthesis", "(a))", "` | `"},
		{"reading not parenthesised", "(a) | b", "parenthesised"},
		{"bare names", "a", "parenthesised"},
	}
	for _, tc := range cases {
		_, err := core.ParseSignature(tc.template)
		require.Error(t, err, tc.name)
		msg := err.Error()
		require.True(t, strings.Contains(msg, tc.problem), tc.name+": "+msg)
		require.True(t, strings.Contains(msg, `"`+tc.template+`"`), tc.name+": the error names the template: "+msg)
	}
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
	require.True(t, slices.Equal([]int{2}, acceptedUpTo(fixed, 10)))
	require.False(t, fixed.Accepts(-1))

	variadic := core.SignatureOf(1, true)
	require.Equal(t, 1, variadic.Arity())
	require.True(t, variadic.IsVariadic())
	require.Equal(t, "", variadic.Template())
	require.True(t, slices.Equal([]int{1, 2, 3, 4}, acceptedUpTo(variadic, 4)))
	require.True(t, variadic.Accepts(1000))
	require.False(t, variadic.Accepts(-1))

	none := core.SignatureOf(0, true)
	require.True(t, none.Accepts(0))
	require.True(t, none.Accepts(1000))
	require.False(t, none.Accepts(-1))
}

func TestSignatureOfNegativeArityPanics(t *testing.T) {
	defer func() {
		require.NotNil(t, recover(), "SignatureOf with a negative arity must panic")
	}()
	core.SignatureOf(-1, false)
}

func TestSignatureZeroValue(t *testing.T) {
	var zero core.Signature
	want := core.SignatureOf(0, false)
	require.Equal(t, want.Arity(), zero.Arity())
	require.Equal(t, want.IsVariadic(), zero.IsVariadic())
	require.Equal(t, "", zero.Template())
	require.True(t, slices.Equal([]int{0}, acceptedUpTo(zero, 10)))
	require.False(t, zero.Accepts(-1))
}

func TestSignatureMax(t *testing.T) {
	cases := []struct {
		sig  core.Signature
		max  int
		open bool
	}{
		{core.MustSignature("()"), 0, false},
		{core.MustSignature("(x)"), 1, false},
		{core.MustSignature("(width, fill)"), 2, false},
		{core.MustSignature("([sep])"), 1, false},
		{core.MustSignature("(width[, fill])"), 2, false},
		{core.MustSignature("(a[, b, c])"), 3, false},
		{core.MustSignature("(a[, b[, c]])"), 3, false},
		{core.MustSignature("() | (components) | (start, stop[, step])"), 3, false},
		{core.MustSignature("(text) | (bands, min, max)"), 3, false},
		{core.MustSignature("(...parts)"), 0, true},
		{core.MustSignature("(first, ...rest)"), 1, true},
		{core.MustSignature("(a) | (a, b, c, ...rest)"), 3, true},
		{core.SignatureOf(0, false), 0, false},
		{core.SignatureOf(2, false), 2, false},
		{core.SignatureOf(2, true), 2, true},
	}
	for _, tc := range cases {
		max, open := tc.sig.Max()
		label := tc.sig.Template()
		require.Equal(t, tc.max, max, label)
		require.Equal(t, tc.open, open, label)
		// Max is consistent with Accepts: max is accepted, and above it only an open set accepts
		require.True(t, tc.sig.Accepts(max), label)
		require.Equal(t, open, tc.sig.Accepts(max+1), label)
	}
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
