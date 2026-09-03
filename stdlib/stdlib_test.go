package stdlib_test

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/mock"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
)

type ARR = []any
type MAP = map[string]any
type IARR []any
type IMAP map[string]any

type callres struct {
	t *testing.T
	o any
	e error
}

var (
	base64Bytes1 = []byte{0x06, 0xAC, 0x76, 0x1B, 0x1D, 0x6A, 0xFA, 0x9D, 0xB1, 0xA0}
	hexBytes1    = []byte{0x06, 0xAC, 0x76, 0x1B, 0x1D, 0x6A, 0xFA, 0x9D, 0xB1, 0xA0}
)

const (
	base64Std    = "Bqx2Gx1q+p2xoA=="
	base64URL    = "Bqx2Gx1q-p2xoA=="
	base64RawStd = "Bqx2Gx1q+p2xoA"
	base64RawURL = "Bqx2Gx1q-p2xoA"
	hex1         = "06ac761b1d6afa9db1a0"
)

func (c callres) call(funcName string, args ...any) callres {
	if c.e != nil {
		return c
	}

	var oargs []core.Value
	for _, v := range args {
		oargs = append(oargs, object(v))
	}

	v := mock.Vm

	if o, ok := c.o.(*stdlib.Module); ok {
		m, ok := (*core.Record)(o.Body.Ptr).Elements[funcName]
		if !ok {
			return callres{t: c.t, e: fmt.Errorf("function not found: %s", funcName)}
		}

		if m.Type != value.BuiltinFunction && m.Type != value.BuiltinClosure {
			return callres{t: c.t, e: fmt.Errorf("non-callable: %s", funcName)}
		}

		res, err := m.Call(v, oargs)
		return callres{t: c.t, o: res, e: err}
	}

	if o, ok := c.o.(core.Value); ok {
		if o.Type == value.BuiltinFunction || o.Type == value.BuiltinClosure {
			res, err := o.Call(v, oargs)
			return callres{t: c.t, o: res, e: err}
		}

		if o.Type == value.Record {
			r := (*core.Record)(o.Ptr)

			m, ok := r.Elements[funcName]
			if !ok {
				return callres{t: c.t, e: fmt.Errorf("function not found: %s", funcName)}
			}

			if m.Type != value.BuiltinFunction && m.Type != value.BuiltinClosure {
				return callres{t: c.t, e: fmt.Errorf("non-callable: %s", funcName)}
			}

			res, err := m.Call(v, oargs)
			return callres{t: c.t, o: res, e: err}
		}
	}

	panic(fmt.Errorf("unexpected object: %+v (%T)", c.o, c.o))
}

func (c callres) expect(expected any, msgAndArgs ...any) {
	require.NoError(c.t, c.e, msgAndArgs...)
	require.Equal(c.t, object(expected), c.o, msgAndArgs...)
}

func (c callres) expectError() {
	require.Error(c.t, c.e)
}

func module(t *testing.T, moduleName string) callres {
	mod, ok := stdlib.GetModuleDefinition(moduleName)
	if !ok {
		return callres{t: t, e: fmt.Errorf("module_not_found: %s", moduleName)}
	}

	return callres{t: t, o: mod}
}

func object(v any) core.Value {
	switch v := v.(type) {
	case core.Value:
		return v

	case string:
		return core.NewStringValue(v)

	case int64:
		return core.IntValue(v)

	case int: // for convenience
		return core.IntValue(int64(v))

	case bool:
		return core.BoolValue(v)

	case rune:
		return core.RuneValue(v)

	case byte: // for convenience
		return core.RuneValue(rune(v))

	case float64:
		return core.FloatValue(v)

	case []byte:
		return core.NewBytesValue(v, false)

	case MAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = object(v)
		}
		return core.NewRecordValue(objs, false)

	case ARR:
		var objs []core.Value
		for _, e := range v {
			t := object(e)
			objs = append(objs, t)
		}
		return core.NewArrayValue(objs, false)

	case IMAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = object(v)
		}
		return core.NewRecordValue(objs, true)

	case IARR:
		var objs []core.Value
		for _, e := range v {
			t := object(e)
			objs = append(objs, t)
		}
		return core.NewArrayValue(objs, true)

	case time.Time:
		return core.NewTimeValue(v)

	case []int:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, core.IntValue(int64(e)))
		}
		return core.NewArrayValue(objs, false)
	}

	panic(fmt.Errorf("unknown type: %T", v))
}

func expect(t *testing.T, input string, expected any) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	e, err := kavun.ValueOf(expected)
	require.NoError(t, err)
	s := kavun.NewScript([]byte(input), "out")
	c, err := s.Compile()
	require.NoError(t, err)
	err = c.Run(machine)
	require.NoError(t, err)
	require.NotNil(t, c)
	v, err := c.Get("out")
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, e, v)
}

func TestModulesRun(t *testing.T) {
	// os.File. Nothing is checked after a call: a failure raises, and the statement-like members
	// (write, close, remove) answer undefined.
	expect(t, `
os := import("os")

write_file := func(filename, data) {
	file := os.create(filename)
	defer file.close()
	file.write(bytes(data))
}

read_file := func(filename) res {
	file := os.open(filename)
	defer file.close()
	data := bytes(b'\x00', 100)
	cnt := file.read(data)
	res = data[:cnt]
}

write_file("./temp", "foobar")
out = string(read_file("./temp"))
os.remove("./temp")
`, "foobar")

	// exec.command
	expect(t, `
os := import("os")
out = os.exec("echo", "foo", "bar").output()
`, []byte("foo bar\n"))
}

// Every module translates its library's Go errors into a raise of that module's kind, exactly once, at the
// module boundary. Nothing answers an error value any more.
func TestModuleFailuresRaise(t *testing.T) {
	cases := []struct {
		name string
		src  string
		kind string
		msg  string
	}{
		{"os.remove", `os := import("os"); os.remove("/nonexistent/definitely/not/here")`, "io", "(os.remove)"},
		{"os.read_file", `os := import("os"); os.read_file("/nonexistent/definitely/not/here")`, "io", "(os.read_file)"},
		{"os.chdir", `os := import("os"); os.chdir("/nonexistent/definitely/not/here")`, "io", "(os.chdir)"},
		{"base64.decode", `b := import("base64"); b.decode("!!!!")`, "conversion", "(base64.decode)"},
		{"hex.decode", `h := import("hex"); h.decode("zz")`, "conversion", "(hex.decode)"},
		{"regexp.re_compile", `r := import("regexp"); r.re_compile("(")`, "invalid_value", "(regexp.re_compile)"},
		{"regexp.re_match", `r := import("regexp"); r.re_match("(", "x")`, "invalid_value", "(regexp.re_match)"},
		{"times.parse", `t := import("times"); t.parse("nonsense layout", "nope")`, "conversion", "(times.parse)"},
		{"times.parse_duration", `t := import("times"); t.parse_duration("not a duration")`, "conversion", "(times.parse_duration)"},
		{"times.in_location", `t := import("times"); t.in_location(t.now(), "Nowhere/Nothing")`, "conversion", "(times.in_location)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
			c, err := kavun.NewScript([]byte(tc.src)).Compile()
			require.NoError(t, err)
			err = c.Run(machine)
			require.Error(t, err)

			var re *kavun.RuntimeError
			require.True(t, errors.As(err, &re), "expected a RuntimeError, got %T: %v", err, err)
			require.Equal(t, tc.kind, re.Kind)
			require.Equal(t, errs.CategoryRuntime.String(), re.Category.String())
			require.False(t, re.Fatal)
			require.True(t, strings.HasPrefix(re.Message, tc.msg), "message %q should start with %q", re.Message, tc.msg)
		})
	}
}

// json failures raise like every other module's, and a nested encoding failure names its path ONCE rather than
// re-prefixing at each level.
func TestJSONFailuresRaise(t *testing.T) {
	cases := []struct{ name, src, kind, msg string }{
		{"decode/truncated", `j := import("json"); j.decode("{")`, "json_decoding", "(json.decode) unexpected end of JSON input"},
		{"indent/truncated", `j := import("json"); j.indent("{", "", "  ")`, "json_decoding", "(json.indent) unexpected end of JSON input"},
		{"encode/unsupported", `j := import("json"); j.encode(func(){})`, "json_encoding", "value type <compiled-function/0> does not support JSON encoding"},
		{"encode/nested-path", `j := import("json"); j.encode({items: [{price: func(){}}]})`, "json_encoding", ".items[0].price: value type <compiled-function/0> does not support JSON encoding"},
		{"encode/array-path", `j := import("json"); j.encode([1, [2, func(){}]])`, "json_encoding", "[1][1]: value type <compiled-function/0> does not support JSON encoding"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
			c, err := kavun.NewScript([]byte(tc.src)).Compile()
			require.NoError(t, err)
			err = c.Run(machine)
			require.Error(t, err)

			var re *kavun.RuntimeError
			require.True(t, errors.As(err, &re), "expected a RuntimeError, got %T: %v", err, err)
			require.Equal(t, tc.kind, re.Kind)
			require.False(t, re.Fatal)
			require.Equal(t, tc.msg, re.Message)
		})
	}
}

// The regexp expansion ceiling is a function of the script's own arguments, so it is recoverable like every
// other declared limit — it used to be a fatal resource_limit, which no recover() could catch.
func TestRegexpExpansionLimitIsRecoverable(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	// 200k matches x a 40k-reference template: far past the 4294967296-element ceiling, and cheap to reject.
	src := `r := import("regexp"); r.re_replace("a", "a".repeat(200000), "$0".repeat(40000))`
	c, err := kavun.NewScript([]byte(src)).Compile()
	require.NoError(t, err)
	err = c.Run(machine)
	require.Error(t, err)

	var re *kavun.RuntimeError
	require.True(t, errors.As(err, &re), "got %T: %v", err, err)
	require.Equal(t, "invalid_value", re.Kind)
	require.False(t, re.Fatal)
	require.True(t, strings.Contains(re.Message, "(regexp.re_replace)"), "got %q", re.Message)
}

// A statement-like module function answers undefined on success — "no result", not a `true` to branch on.
func TestModuleStatementLikeAnswerUndefined(t *testing.T) {
	expect(t, `
os := import("os")
d := os.temp_dir() + "/kavun_stdlib_test_dir"
os.remove_all(d)
out = [
	is_undefined(os.mkdir(d, 0o755)),
	is_undefined(os.chmod(d, 0o700)),
	is_undefined(os.remove_all(d)),
]
`, []any{true, true, true})
}

func TestBase64(t *testing.T) {
	module(t, `base64`).call("encode", base64Bytes1).expect(base64Std)
	module(t, `base64`).call("decode", base64Std).expect(base64Bytes1)
	module(t, `base64`).call("url_encode", base64Bytes1).expect(base64URL)
	module(t, `base64`).call("url_decode", base64URL).expect(base64Bytes1)
	module(t, `base64`).call("raw_encode", base64Bytes1).expect(base64RawStd)
	module(t, `base64`).call("raw_decode", base64RawStd).expect(base64Bytes1)
	module(t, `base64`).call("raw_url_encode", base64Bytes1).expect(base64RawURL)
	module(t, `base64`).call("raw_url_decode", base64RawURL).expect(base64Bytes1)
}

func TestHex(t *testing.T) {
	module(t, `hex`).call("encode", hexBytes1).expect(hex1)
	module(t, `hex`).call("decode", hex1).expect(hexBytes1)
}

func TestJSON(t *testing.T) {
	module(t, "json").call("encode", 5).expect([]byte("5"))
	module(t, "json").call("encode", "foobar").expect([]byte(`"foobar"`))
	module(t, "json").call("encode", MAP{"foo": 5}).expect([]byte("{\"foo\":5}"))
	module(t, "json").call("encode", IMAP{"foo": 5}).expect([]byte("{\"foo\":5}"))
	module(t, "json").call("encode", ARR{1, 2, 3}).expect([]byte("[1,2,3]"))
	module(t, "json").call("encode", IARR{1, 2, 3}).expect([]byte("[1,2,3]"))
	module(t, "json").call("encode", MAP{"foo": "bar"}).expect([]byte("{\"foo\":\"bar\"}"))
	module(t, "json").call("encode", MAP{"foo": 1.8}).expect([]byte("{\"foo\":1.8}"))
	module(t, "json").call("encode", MAP{"foo": true}).expect([]byte("{\"foo\":true}"))
	module(t, "json").call("encode", MAP{"foo": '8'}).expect([]byte("{\"foo\":56}"))
	module(t, "json").call("encode", MAP{"foo": []byte("foo")}).expect([]byte("{\"foo\":\"Zm9v\"}")) // json encoding returns []byte as base64 encoded string
	module(t, "json").call("encode", MAP{"foo": ARR{"bar", 1, 1.8, '8', true}}).expect([]byte("{\"foo\":[\"bar\",1,1.8,56,true]}"))
	module(t, "json").call("encode", MAP{"foo": IARR{"bar", 1, 1.8, '8', true}}).expect([]byte("{\"foo\":[\"bar\",1,1.8,56,true]}"))
	module(t, "json").call("encode", MAP{"foo": ARR{ARR{"bar", 1}, ARR{"bar", 1}}}).expect([]byte("{\"foo\":[[\"bar\",1],[\"bar\",1]]}"))
	module(t, "json").call("encode", MAP{"foo": MAP{"string": "bar"}}).expect([]byte("{\"foo\":{\"string\":\"bar\"}}"))
	module(t, "json").call("encode", MAP{"foo": IMAP{"string": "bar"}}).expect([]byte("{\"foo\":{\"string\":\"bar\"}}"))
	module(t, "json").call("encode", MAP{"foo": MAP{"map1": MAP{"string": "bar"}}}).expect([]byte("{\"foo\":{\"map1\":{\"string\":\"bar\"}}}"))
	module(t, "json").call("encode", ARR{ARR{"bar", 1}, ARR{"bar", 1}}).expect([]byte("[[\"bar\",1],[\"bar\",1]]"))

	module(t, "json").call("decode", `5`).expect(5)
	module(t, "json").call("decode", `"foo"`).expect("foo")
	module(t, "json").call("decode", `[1,2,3,"bar"]`).expect(ARR{1, 2, 3, "bar"})
	module(t, "json").call("decode", `{"foo":5}`).expect(MAP{"foo": 5})
	module(t, "json").call("decode", `{"foo":2.5}`).expect(MAP{"foo": 2.5})
	module(t, "json").call("decode", `{"foo":true}`).expect(MAP{"foo": true})
	module(t, "json").call("decode", `{"foo":"bar"}`).expect(MAP{"foo": "bar"})
	module(t, "json").call("decode", `{"foo":[1,2,3,"bar"]}`).expect(MAP{"foo": ARR{1, 2, 3, "bar"}})

	module(t, "json").call("indent", []byte("{\"foo\":[\"bar\",1,1.8,56,true]}"), "", "  ").expect([]byte(`{
  "foo": [
    "bar",
    1,
    1.8,
    56,
    true
  ]
}`))

	module(t, "json").call("html_escape", []byte(`{"M":"<html>foo &`+"\xe2\x80\xa8 \xe2\x80\xa9"+`</html>"}`)).
		expect([]byte(`{"M":"\u003chtml\u003efoo \u0026\u2028 \u2029\u003c/html\u003e"}`))
}

func TestReadFile(t *testing.T) {
	content := []byte("the quick brown fox jumps over the lazy dog")
	tf, err := os.CreateTemp("", "test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tf.Name()) }()

	_, err = tf.Write(content)
	require.NoError(t, err)
	_ = tf.Close()

	module(t, "os").call("read_file", tf.Name()).expect(core.NewBytesValue(content, false))
}

func TestReadFileArgs(t *testing.T) {
	module(t, "os").call("read_file").expectError()
}
func TestFileStatArgs(t *testing.T) {
	module(t, "os").call("stat").expectError()
}

func TestFileStatFile(t *testing.T) {
	content := []byte("the quick brown fox jumps over the lazy dog")
	tf, err := os.CreateTemp("", "test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tf.Name()) }()

	_, err = tf.Write(content)
	require.NoError(t, err)
	_ = tf.Close()

	stat, err := os.Stat(tf.Name())
	if err != nil {
		t.Logf("could not get tmp file stat: %s", err)
		return
	}

	rec := core.NewRecordValue(map[string]core.Value{
		"name":      core.NewStringValue(stat.Name()),
		"mtime":     core.NewTimeValue(stat.ModTime()),
		"size":      core.IntValue(stat.Size()),
		"mode":      core.IntValue(int64(stat.Mode())),
		"directory": core.False,
	}, true)
	module(t, "os").call("stat", tf.Name()).expect(rec)
}

func TestFileStatDir(t *testing.T) {
	td, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(td) }()

	stat, err := os.Stat(td)
	require.NoError(t, err)

	rec := core.NewRecordValue(map[string]core.Value{
		"name":      core.NewStringValue(stat.Name()),
		"mtime":     core.NewTimeValue(stat.ModTime()),
		"size":      core.IntValue(stat.Size()),
		"mode":      core.IntValue(int64(stat.Mode())),
		"directory": core.True,
	}, true)
	module(t, "os").call("stat", td).expect(rec)
}

func TestOSExpandEnv(t *testing.T) {
	_ = os.Setenv("KAVUN", "FOO BAR")
	module(t, "os").call("expand_env", "$KAVUN").expect("FOO BAR")

	_ = os.Setenv("KAVUN", "FOO")
	module(t, "os").call("expand_env", "$KAVUN $KAVUN").expect("FOO FOO")

	_ = os.Setenv("KAVUN", "123456789012")
	module(t, "os").call("expand_env", "$KAVUN").expect("123456789012")

	_ = os.Setenv("KAVUN", "123456")
	module(t, "os").call("expand_env", "$KAVUN$KAVUN").expect("123456123456")

	_ = os.Setenv("KAVUN", "123456")
	module(t, "os").call("expand_env", "${KAVUN}${KAVUN}").expect("123456123456")
}

func TestRegexpREAlternation(t *testing.T) {
	module(t, "regexp").call("re_find", "([a-zA-Z])|([0-9])", "a").expect(ARR{
		ARR{
			IMAP{"text": "a", "begin": 0, "end": 1},
			IMAP{"text": "a", "begin": 0, "end": 1},
		},
	}, "alternation with letter")

	module(t, "regexp").call("re_find", "([a-zA-Z])|([0-9])", "5").expect(ARR{
		ARR{
			IMAP{"text": "5", "begin": 0, "end": 1},
			IMAP{"text": "5", "begin": 0, "end": 1},
		},
	}, "alternation with number")

	module(t, "regexp").call("re_find", "([a-zA-Z])|([0-9])", "").expect(core.Undefined, "empty input")

	module(t, "regexp").call("re_find", "([a-zA-Z])|([0-9])", "!").expect(core.Undefined, "non-matching input")

	module(t, "regexp").call("re_find", "(?:([a-zA-Z])|([0-9]))+", "a5b").expect(ARR{
		ARR{
			IMAP{"text": "a5b", "begin": 0, "end": 3},
			IMAP{"text": "b", "begin": 2, "end": 3},
			IMAP{"text": "5", "begin": 1, "end": 2},
		},
	}, "multiple alternations")

	module(t, "regexp").call("re_find", "(foo)|(bar)|(baz)", "foo").expect(ARR{
		ARR{
			IMAP{"text": "foo", "begin": 0, "end": 3},
			IMAP{"text": "foo", "begin": 0, "end": 3},
		},
	}, "multiple groups with non-matches")

	module(t, "regexp").call("re_find", "((cat)|(dog))((run)|(walk))", "catrun").expect(ARR{
		ARR{
			IMAP{"text": "catrun", "begin": 0, "end": 6},
			IMAP{"text": "cat", "begin": 0, "end": 3},
			IMAP{"text": "cat", "begin": 0, "end": 3},
			IMAP{"text": "run", "begin": 3, "end": 6},
			IMAP{"text": "run", "begin": 3, "end": 6},
		},
	}, "nested groups with alternation")
}

func TestRegexpRE(t *testing.T) {
	// re_match(pattern, text)
	for _, d := range []struct {
		pattern string
		text    string
	}{
		{"abc", ""},
		{"abc", "abc"},
		{"a", "abc"},
		{"b", "abc"},
		{"^a", "abc"},
		{"^b", "abc"},
	} {
		expected := regexp.MustCompile(d.pattern).MatchString(d.text)
		module(t, "regexp").call("re_match", d.pattern, d.text).expect(expected, "pattern: %q, src: %q", d.pattern, d.text)
		module(t, "regexp").call("re_compile", d.pattern).call("match", d.text).expect(expected, "patter: %q, src: %q", d.pattern, d.text)
	}

	// re_find(pattern, text)
	for _, d := range []struct {
		pattern  string
		text     string
		expected any
	}{
		{"a(b)", "", core.Undefined},
		{"a(b)", "ab", ARR{
			ARR{
				IMAP{"text": "ab", "begin": 0, "end": 2},
				IMAP{"text": "b", "begin": 1, "end": 2},
			},
		}},
		{"a(bc)d", "abcdefgabcd", ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "bc", "begin": 1, "end": 3},
			},
		}},
		{"(a)b(c)d", "abcdefgabcd", ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "a", "begin": 0, "end": 1},
				IMAP{"text": "c", "begin": 2, "end": 3},
			},
		}},
	} {
		module(t, "regexp").call("re_find", d.pattern, d.text).expect(d.expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "regexp").call("re_compile", d.pattern).call("find", d.text).expect(d.expected, "pattern: %q, text: %q", d.pattern, d.text)
	}

	// re_find(pattern, text, count))
	for _, d := range []struct {
		pattern  string
		text     string
		count    int
		expected any
	}{
		{"a(b)", "", -1, core.Undefined},
		{"a(b)", "ab", -1, ARR{
			ARR{
				IMAP{"text": "ab", "begin": 0, "end": 2},
				IMAP{"text": "b", "begin": 1, "end": 2},
			},
		}},
		{"a(bc)d", "abcdefgabcd", -1, ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "bc", "begin": 1, "end": 3},
			},
			ARR{
				IMAP{"text": "abcd", "begin": 7, "end": 11},
				IMAP{"text": "bc", "begin": 8, "end": 10},
			},
		}},
		{"(a)b(c)d", "abcdefgabcd", -1, ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "a", "begin": 0, "end": 1},
				IMAP{"text": "c", "begin": 2, "end": 3},
			},
			ARR{
				IMAP{"text": "abcd", "begin": 7, "end": 11},
				IMAP{"text": "a", "begin": 7, "end": 8},
				IMAP{"text": "c", "begin": 9, "end": 10},
			},
		}},
		{"(a)b(c)d", "abcdefgabcd", 0, core.Undefined},
		{"(a)b(c)d", "abcdefgabcd", 1, ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "a", "begin": 0, "end": 1},
				IMAP{"text": "c", "begin": 2, "end": 3},
			},
		}},
	} {
		module(t, "regexp").call("re_find", d.pattern, d.text, d.count).expect(d.expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "regexp").call("re_compile", d.pattern).call("find", d.text, d.count).expect(d.expected, "pattern: %q, text: %q", d.pattern, d.text)
	}

	// re_replace(pattern, text, repl)
	for _, d := range []struct {
		pattern string
		text    string
		repl    string
	}{
		{"a", "", "b"},
		{"a", "a", "b"},
		{"a", "acac", "b"},
		{"b", "acac", "x"},
		{"a", "acac", "123"},
		{"ac", "acac", "99"},
		{"ac$", "acac", "foo"},
		{"a(b)", "ababab", "$1"},
		{"a(b)(c)", "abcabcabc", "$2$1"},
		{"(a(b)c)", "abcabcabc", "$1$2"},
		{"(일(2)삼)", "일2삼12삼일23", "$1$2"},
		{"((일)(2)3)", "일23\n일이3\n일23", "$1$2$3"},
		{"(a(b)c)", "abc\nabc\nabc", "$1$2"},
	} {
		expected := regexp.MustCompile(d.pattern).
			ReplaceAllString(d.text, d.repl)
		module(t, "regexp").call("re_replace", d.pattern, d.text, d.repl).expect(expected, "pattern: %q, text: %q, repl: %q", d.pattern, d.text, d.repl)
		module(t, "regexp").call("re_compile", d.pattern).call("replace", d.text, d.repl).expect(expected, "pattern: %q, text: %q, repl: %q", d.pattern, d.text, d.repl)
	}

	// re_split(pattern, text)
	for _, d := range []struct {
		pattern string
		text    string
	}{
		{"a", ""},
		{"a", "abcabc"},
		{"ab", "abcabc"},
		{"^a", "abcabc"},
	} {
		var expected []any
		for _, ex := range regexp.MustCompile(d.pattern).Split(d.text, -1) {
			expected = append(expected, ex)
		}
		module(t, "regexp").call("re_split", d.pattern, d.text).expect(expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "regexp").call("re_compile", d.pattern).call("split", d.text).expect(expected, "pattern: %q, text: %q", d.pattern, d.text)
	}

	// re_split(pattern, text, count))
	for _, d := range []struct {
		pattern string
		text    string
		count   int
	}{
		{"a", "", -1},
		{"a", "abcabc", -1},
		{"ab", "abcabc", -1},
		{"^a", "abcabc", -1},
		{"a", "abcabc", 0},
		{"a", "abcabc", 1},
		{"a", "abcabc", 2},
		{"a", "abcabc", 3},
		{"b", "abcabc", 1},
		{"b", "abcabc", 2},
		{"b", "abcabc", 3},
	} {
		var expected []any
		for _, ex := range regexp.MustCompile(d.pattern).Split(d.text, d.count) {
			expected = append(expected, ex)
		}
		module(t, "regexp").call("re_split", d.pattern, d.text, d.count).expect(expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "regexp").call("re_compile", d.pattern).call("split", d.text, d.count).expect(expected, "pattern: %q, text: %q", d.pattern, d.text)
	}
}

func TestRegexpReplace(t *testing.T) {
	module(t, "regexp").call("re_replace", "1", "123456789012", "x").expect("x234567890x2")
	module(t, "regexp").call("re_replace", "12", "123456789012", "x").expect("x34567890x")
	module(t, "regexp").call("re_replace", "1(2)", "123456789012", "x$1").expect("x234567890x2")
	module(t, "regexp").call("re_replace", "(1)(2)", "123456789012", "$2$1").expect("213456789021")
}

func TestTimes(t *testing.T) {
	// UTC, not the host's zone: times.date(...) with no location argument now builds in UTC so the
	// same script is reproducible on every machine, and this fixture has to match it.
	time1 := time.Date(1982, 9, 28, 19, 21, 44, 999, time.UTC)
	time2 := time.Now()
	location, _ := time.LoadLocation("Pacific/Auckland")
	time3 := time.Date(1982, 9, 28, 19, 21, 44, 999, location)

	module(t, "times").call("sleep", 1).expect(core.Undefined)

	r := module(t, "times").call("since", time.Now().Add(-time.Hour)).o.(core.Value)
	require.True(t, r.Type == value.Int)
	require.True(t, int64(r.Data) > 3600000000000)

	r = module(t, "times").call("until", time.Now().Add(time.Hour)).o.(core.Value)
	require.True(t, r.Type == value.Int)
	require.True(t, int64(r.Data) < 3600000000000)

	module(t, "times").call("parse_duration", "1ns").expect(1)
	module(t, "times").call("parse_duration", "1ms").expect(1000000)
	module(t, "times").call("parse_duration", "1h").expect(3600000000000)
	module(t, "times").call("duration_hours", 1800000000000).expect(0.5)
	module(t, "times").call("duration_minutes", 1800000000000).expect(30.0)
	module(t, "times").call("duration_nanoseconds", 100).expect(100)
	module(t, "times").call("duration_seconds", 1000000).expect(0.001)
	module(t, "times").call("duration_string", 1800000000000).expect("30m0s")

	module(t, "times").call("date", 1982, 9, 28, 19, 21, 44, 999).expect(time1)
	module(t, "times").call("date", 1982, 9, 28, 19, 21, 44, 999, "Pacific/Auckland").expect(time3)

	r = module(t, "times").call("now").o.(core.Value)
	rt, _ := r.AsTime()
	nowD := time.Until(rt).Nanoseconds()
	require.True(t, 0 > nowD && nowD > -100000000) // within 100ms

	parsed, _ := time.Parse(time.RFC3339, "1982-09-28T19:21:44+07:00")
	module(t, "times").call("parse", time.RFC3339, "1982-09-28T19:21:44+07:00").expect(parsed)
	module(t, "times").call("unix", 1234325, 94493).expect(time.Unix(1234325, 94493).UTC())

	module(t, "times").call("add_date", time2, 1, 2, 3).expect(time2.AddDate(1, 2, 3))

	// int -> time constructors: the int is a unix timestamp in the encoding the name states, and the
	// result is UTC (unlike times.unix(sec, nsec), which returns the host's local zone).
	module(t, "times").call("from_unix", time1.Unix()).expect(time.Unix(time1.Unix(), 0).UTC())
	module(t, "times").call("from_unix_ms", time1.UnixMilli()).expect(time.UnixMilli(time1.UnixMilli()).UTC())
	module(t, "times").call("from_unix_micro", time1.UnixMicro()).expect(time.UnixMicro(time1.UnixMicro()).UTC())
	module(t, "times").call("from_unix_nano", time1.UnixNano()).expect(time.Unix(0, time1.UnixNano()).UTC())
	module(t, "times").call("in_location", time1, location.String()).expect(time1.In(location))
}

// TestRetiredModuleFunctions pins the module-surface deletions: the text module is gone entirely —
// its five regex functions live on in the regexp module and everything string-shaped became member
// functions — times keeps only what has no member or operator spelling, and min/max selection is
// the free variadic builtins' and the members' job. A retired name reads as undefined through the
// module record (selector semantics), so the loud failure is at the call: undefined is not callable.
func TestRetiredModuleFunctions(t *testing.T) {
	expect(t, `times := import("times")
out = times.add == undefined && times.sub == undefined && times.after == undefined &&
      times.before == undefined && times.is_zero == undefined && times.time_year == undefined &&
      times.to_utc == undefined && times.month_string == undefined && times.time_format == undefined`, true)
	expect(t, `math := import("math"); out = math.min == undefined && math.max == undefined`, true)
	expect(t, `re := import("regexp"); out = re.re_match("[0-9]", "a1")`, true)
}
