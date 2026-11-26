package signature_test

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/signature"
)

// req and opt build a Param: required, or at optional depth d.
func req(name string) signature.Param        { return signature.Param{Name: name} }
func opt(name string, d int) signature.Param { return signature.Param{Name: name, Optional: d} }

// rd builds a Reading without a rest; rdRest one with.
func rd(params ...signature.Param) signature.Reading {
	return signature.Reading{Params: append([]signature.Param{}, params...)}
}
func rdRest(rest string, params ...signature.Param) signature.Reading {
	r := rd(params...)
	r.Rest = rest
	return r
}

func TestParseReadings(t *testing.T) {
	cases := []struct {
		template string
		readings []signature.Reading
	}{
		{"()", []signature.Reading{rd()}},
		{"(x)", []signature.Reading{rd(req("x"))}},
		{"(...args)", []signature.Reading{rdRest("args")}},
		{"([sep])", []signature.Reading{rd(opt("sep", 1))}},
		{"(width[, fill])", []signature.Reading{rd(req("width"), opt("fill", 1))}},
		{"(start, stop[, step])", []signature.Reading{rd(req("start"), req("stop"), opt("step", 1))}},
		{"(a, ...rest)", []signature.Reading{rdRest("rest", req("a"))}},
		{"(a[, b, c])", []signature.Reading{rd(req("a"), opt("b", 1), opt("c", 1))}},
		{"(a[, b[, c]])", []signature.Reading{rd(req("a"), opt("b", 1), opt("c", 2))}},
		{"([start[, delete_count]])", []signature.Reading{rd(opt("start", 1), opt("delete_count", 2))}},
		{"([a, b[, c[, d]]])", []signature.Reading{rd(opt("a", 1), opt("b", 1), opt("c", 2), opt("d", 3))}},
		{"() | (start, stop[, step])", []signature.Reading{rd(), rd(req("start"), req("stop"), opt("step", 1))}},
		{"(a) | (a, b, c, ...rest)", []signature.Reading{rd(req("a")), rdRest("rest", req("a"), req("b"), req("c"))}},
	}
	for _, tc := range cases {
		s, err := signature.Parse(tc.template)
		require.NoError(t, err, tc.template)
		require.Equal(t, tc.template, s.Template, tc.template)
		require.True(t, reflect.DeepEqual(tc.readings, s.Readings), "%s: readings %+v", tc.template, s.Readings)
		for _, r := range s.Readings {
			require.True(t, r.Params != nil, "%s: Params must never be nil", tc.template)
		}
	}
}

func TestReadingCounts(t *testing.T) {
	cases := []struct {
		template string
		counts   []int
		restFrom int
		hasRest  bool
	}{
		{"()", []int{0}, 0, false},
		{"(x)", []int{1}, 0, false},
		{"(width, fill)", []int{2}, 0, false},
		{"([sep])", []int{0, 1}, 0, false},
		{"(width[, fill])", []int{1, 2}, 0, false},
		{"(start, stop[, step])", []int{2, 3}, 0, false},
		{"(a[, b, c])", []int{1, 3}, 0, false},
		{"(a[, b[, c]])", []int{1, 2, 3}, 0, false},
		{"([a[, b]])", []int{0, 1, 2}, 0, false},
		{"([a, b[, c]])", []int{0, 2, 3}, 0, false},
		{"(...args)", nil, 0, true},
		{"(first, ...rest)", nil, 1, true},
		{"(a, b, c, ...rest)", nil, 3, true},
	}
	for _, tc := range cases {
		s, err := signature.Parse(tc.template)
		require.NoError(t, err, tc.template)
		require.Equal(t, 1, len(s.Readings), tc.template)
		counts, restFrom, hasRest := s.Readings[0].Counts()
		require.True(t, slices.Equal(tc.counts, counts), "%s: counts %v", tc.template, counts)
		require.Equal(t, tc.restFrom, restFrom, tc.template)
		require.Equal(t, tc.hasRest, hasRest, tc.template)
	}
}

func TestShapeAcceptsAndMax(t *testing.T) {
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
		accepts  []int // within 0..limit
		max      int
		open     bool
	}{
		{"()", 0, false, []int{0}, 0, false},
		{"(x)", 1, false, []int{1}, 1, false},
		{"(width, fill)", 2, false, []int{2}, 2, false},
		{"([sep])", 0, true, []int{0, 1}, 1, false},
		{"(width[, fill])", 1, true, []int{1, 2}, 2, false},
		{"(start, stop[, step])", 2, true, []int{2, 3}, 3, false},
		{"(a[, b, c])", 1, true, []int{1, 3}, 3, false},
		{"(a[, b[, c]])", 1, true, []int{1, 2, 3}, 3, false},
		{"([a[, b]])", 0, true, []int{0, 1, 2}, 2, false},
		{"(...parts)", 0, true, open(0), 0, true},
		{"(first, ...rest)", 1, true, open(1), 1, true},
		{"() | (components) | (start, stop[, step])", 0, true, []int{0, 1, 2, 3}, 3, false},
		{"(text) | (bands, min, max)", 1, true, []int{1, 3}, 3, false},
		{"(a) | (a, b, c, ...rest)", 1, true, append([]int{1}, open(3)...), 3, true},
		{"(x_1, _y)", 2, false, []int{2}, 2, false},
		{"(a, a_[, b]) | (b)", 1, true, []int{1, 2, 3}, 3, false},
		{"(a) | (a, b)", 1, true, []int{1, 2}, 2, false},
	}
	for _, tc := range cases {
		s, err := signature.Parse(tc.template)
		require.NoError(t, err, tc.template)
		require.Equal(t, tc.arity, s.Arity, tc.template)
		require.Equal(t, tc.variadic, s.Variadic, tc.template)
		require.False(t, s.Accepts(-1), tc.template)
		var got []int
		for k := 0; k <= limit; k++ {
			if s.Accepts(k) {
				got = append(got, k)
			}
		}
		require.True(t, slices.Equal(tc.accepts, got), "%s: accepts %v", tc.template, got)
		n, open := s.Max()
		require.Equal(t, tc.max, n, tc.template)
		require.Equal(t, tc.open, open, tc.template)
		// Max is consistent with Accepts: max is accepted, and above it only an open set accepts
		require.True(t, s.Accepts(n), tc.template)
		require.Equal(t, open, s.Accepts(n+1), tc.template)
	}

	accepts := func(template string, k int) bool {
		s, err := signature.Parse(template)
		require.NoError(t, err, template)
		return s.Accepts(k)
	}
	require.True(t, accepts("(...parts)", 1000))
	require.True(t, accepts("(first, ...rest)", 1000))
	require.False(t, accepts("(a[, b[, c]])", 1000))
}

func TestParseErrors(t *testing.T) {
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
		_, err := signature.Parse(tc.template)
		require.Error(t, err, tc.name)
		msg := err.Error()
		require.True(t, strings.Contains(msg, tc.problem), tc.name+": "+msg)
		require.True(t, strings.Contains(msg, `"`+tc.template+`"`), tc.name+": the error names the template: "+msg)
	}
}

// multiForm holds every distinct multi-form template ("... ) | ( ...") declared in the repository's sources, with
// the readings it must parse to. TestMultiFormTemplates walks the sources both ways: a declared template missing
// here fails, and so does a row no declaration uses any more.
var multiForm = map[string][]signature.Reading{
	"([start[, delete_count]]) | (start, delete_count, ...items)": {
		rd(opt("start", 1), opt("delete_count", 2)),
		rdRest("items", req("start"), req("delete_count")),
	},
	"() | (components) | (start, stop[, step])": {
		rd(), rd(req("components")), rd(req("start"), req("stop"), opt("step", 1)),
	},
	"(x) | (template, args)": {
		rd(req("x")), rd(req("template"), req("args")),
	},
	"([x]) | (text, layout)": {
		rd(opt("x", 1)), rd(req("text"), req("layout")),
	},
	"(amount, on, scale, mode) | (amount, on, fallback, scale, mode)": {
		rd(req("amount"), req("on"), req("scale"), req("mode")),
		rd(req("amount"), req("on"), req("fallback"), req("scale"), req("mode")),
	},
	"(n, pmt, pv, fv, timing, scale, mode) | (n, pmt, pv, fv, timing, solver, scale, mode)": {
		rd(req("n"), req("pmt"), req("pv"), req("fv"), req("timing"), req("scale"), req("mode")),
		rd(req("n"), req("pmt"), req("pv"), req("fv"), req("timing"), req("solver"), req("scale"), req("mode")),
	},
	"(cf, scale, mode) | (cf, solver, scale, mode)": {
		rd(req("cf"), req("scale"), req("mode")),
		rd(req("cf"), req("solver"), req("scale"), req("mode")),
	},
	"() | (n1, d1[, n2, d2])": {
		rd(), rd(req("n1"), req("d1"), opt("n2", 1), opt("d2", 1)),
	},
	"(flows, convention, scale, mode) | (flows, convention, solver, scale, mode)": {
		rd(req("flows"), req("convention"), req("scale"), req("mode")),
		rd(req("flows"), req("convention"), req("solver"), req("scale"), req("mode")),
	},
	"() | (text) | (bands, min, max)": {
		rd(), rd(req("text")), rd(req("bands"), req("min"), req("max")),
	},
	"(f, scale, mode) | (f, solver, scale, mode)": {
		rd(req("f"), req("scale"), req("mode")),
		rd(req("f"), req("solver"), req("scale"), req("mode")),
	},
	"() | (hi) | (lo, hi)": {
		rd(), rd(req("hi")), rd(req("lo"), req("hi")),
	},
}

// declaredTemplate extracts the template of a //kavun:member, //kavun:builtin or //kavun:fn directive.
var declaredTemplate = regexp.MustCompile(`^//kavun:(?:member|builtin|fn) .*? (?:pure|impure) (\(.*\)) -> `)

// declaredMultiForm answers every multi-form template declared under core, vm and stdlib, one entry per directive.
func declaredMultiForm(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, dir := range []string{"../core", "../vm", "../stdlib"} {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				if m := declaredTemplate.FindStringSubmatch(sc.Text()); m != nil && strings.Contains(m[1], ") | (") {
					out = append(out, m[1])
				}
			}
			return sc.Err()
		})
		require.NoError(t, err, dir)
	}
	return out
}

func TestMultiFormTemplates(t *testing.T) {
	declared := declaredMultiForm(t)
	require.True(t, len(declared) >= 20, "multi-form declarations found: %d", len(declared))
	for _, tmpl := range declared {
		_, ok := multiForm[tmpl]
		require.True(t, ok, "declared multi-form template missing from the table: %s", tmpl)
	}
	for tmpl := range multiForm {
		require.True(t, slices.Contains(declared, tmpl), "table row no declaration uses: %s", tmpl)
	}

	for tmpl, want := range multiForm {
		s, err := signature.Parse(tmpl)
		require.NoError(t, err, tmpl)
		require.True(t, reflect.DeepEqual(want, s.Readings), "%s: readings %+v", tmpl, s.Readings)

		// the union of the readings' own counts is exactly what the Shape accepts
		for k := 0; k <= 10; k++ {
			union := false
			for _, r := range s.Readings {
				counts, restFrom, hasRest := r.Counts()
				if slices.Contains(counts, k) || (hasRest && k >= restFrom) {
					union = true
				}
			}
			require.Equal(t, union, s.Accepts(k), "%s: count %d", tmpl, k)
		}
	}
}
