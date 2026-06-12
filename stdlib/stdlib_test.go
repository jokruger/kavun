package stdlib_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
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

func (c callres) call(rta *core.Arena, funcName string, args ...any) callres {
	if c.e != nil {
		return c
	}

	var oargs []core.Value
	for _, v := range args {
		oargs = append(oargs, object(rta, v))
	}

	v := mock.Vm

	if o, ok := c.o.(*stdlib.Module); ok {
		m, ok := o.Attrs[funcName]
		if !ok {
			return callres{t: c.t, e: fmt.Errorf("function not found: %s", funcName)}
		}

		if m.Type != core.VT_BUILTIN_FUNCTION && m.Type != core.VT_BUILTIN_CLOSURE {
			return callres{t: c.t, e: fmt.Errorf("non-callable: %s", funcName)}
		}

		res, err := m.Call(rta, v, oargs)
		return callres{t: c.t, o: res, e: err}
	}

	if o, ok := c.o.(core.Value); ok {
		if o.Type == core.VT_BUILTIN_FUNCTION || o.Type == core.VT_BUILTIN_CLOSURE {
			res, err := o.Call(rta, v, oargs)
			return callres{t: c.t, o: res, e: err}
		}

		if o.Type == core.VT_RECORD {
			r := rta.ResolveDictValue(o)

			m, ok := r.Elements[funcName]
			if !ok {
				return callres{t: c.t, e: fmt.Errorf("function not found: %s", funcName)}
			}

			if m.Type != core.VT_BUILTIN_FUNCTION && m.Type != core.VT_BUILTIN_CLOSURE {
				return callres{t: c.t, e: fmt.Errorf("non-callable: %s", funcName)}
			}

			res, err := m.Call(rta, v, oargs)
			return callres{t: c.t, o: res, e: err}
		}
	}

	panic(fmt.Errorf("unexpected object: %+v (%T)", c.o, c.o))
}

func (c callres) expect(a *core.Arena, expected any, msgAndArgs ...any) {
	require.NoError(c.t, c.e, msgAndArgs...)
	require.Equal(c.t, a, object(a, expected), c.o, msgAndArgs...)
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

func object(a *core.Arena, v any) core.Value {
	switch v := v.(type) {
	case core.Value:
		return v

	case string:
		nv, err := a.NewStringValue(v)
		if err != nil {
			panic(fmt.Errorf("failed to create string value: %w", err))
		}
		return nv

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
		nv, err := a.NewBytesValue(v, false)
		if err != nil {
			panic(fmt.Errorf("failed to create bytes value: %w", err))
		}
		return nv

	case MAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = object(a, v)
			objs[k].Pin(a)
		}
		nv, err := a.NewRecordValue(objs, false)
		if err != nil {
			panic(fmt.Errorf("failed to create record value: %w", err))
		}
		return nv

	case ARR:
		var objs []core.Value
		for _, e := range v {
			t := object(a, e)
			t.Pin(a)
			objs = append(objs, t)
		}
		nv, err := a.NewArrayValue(objs, false)
		if err != nil {
			panic(fmt.Errorf("failed to create array value: %w", err))
		}
		return nv

	case IMAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = object(a, v)
			objs[k].Pin(a)
		}
		nv, err := a.NewRecordValue(objs, true)
		if err != nil {
			panic(fmt.Errorf("failed to create record value: %w", err))
		}
		return nv

	case IARR:
		var objs []core.Value
		for _, e := range v {
			t := object(a, e)
			t.Pin(a)
			objs = append(objs, t)
		}
		nv, err := a.NewArrayValue(objs, true)
		if err != nil {
			panic(fmt.Errorf("failed to create array value: %w", err))
		}
		return nv

	case time.Time:
		nv, err := a.NewTimeValue(v)
		if err != nil {
			panic(fmt.Errorf("failed to create time value: %w", err))
		}
		return nv

	case []int:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, core.IntValue(int64(e)))
		}
		nv, err := a.NewArrayValue(objs, false)
		if err != nil {
			panic(fmt.Errorf("failed to create array value: %w", err))
		}
		return nv
	}

	panic(fmt.Errorf("unknown type: %T", v))
}

func expect(t *testing.T, a *core.Arena, input string, expected any) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	e, err := kavun.ValueOf(a, expected)
	require.NoError(t, err)
	s := kavun.NewScript([]byte(input))
	c, err := s.Compile()
	require.NoError(t, err)
	err = c.Run(a, machine)
	require.NoError(t, err)
	require.NotNil(t, c)
	v := c.Get("out")
	require.NotNil(t, v)
	require.Equal(t, a, e, v)
}

func TestModulesRun(t *testing.T) {
	rta := core.NewArena(nil)

	// os.File
	expect(t, rta, `
os := import("os")
out := ""

write_file := func(filename, data) {
	file := os.create(filename)
	if !file { return file }

	if res := file.write(bytes(data)); is_error(res) {
		return res
	}

	return file.close()
}

read_file := func(filename) {
	file := os.open(filename)
	if !file { return file }

	data := bytes(100)
	cnt := file.read(data)
	if  is_error(cnt) {
		return cnt
	}

	file.close()
	return data[:cnt]
}

if write_file("./temp", "foobar") {
	out = string(read_file("./temp"))
}

os.remove("./temp")
`, "foobar")

	// exec.command
	expect(t, rta, `
out := ""
os := import("os")
cmd := os.exec("echo", "foo", "bar")
if !is_error(cmd) {
	out = cmd.output()
}
`, []byte("foo bar\n"))

}

func TestBase64(t *testing.T) {
	rta := core.NewArena(nil)
	module(t, `base64`).call(rta, "encode", base64Bytes1).expect(rta, base64Std)
	module(t, `base64`).call(rta, "decode", base64Std).expect(rta, base64Bytes1)
	module(t, `base64`).call(rta, "url_encode", base64Bytes1).expect(rta, base64URL)
	module(t, `base64`).call(rta, "url_decode", base64URL).expect(rta, base64Bytes1)
	module(t, `base64`).call(rta, "raw_encode", base64Bytes1).expect(rta, base64RawStd)
	module(t, `base64`).call(rta, "raw_decode", base64RawStd).expect(rta, base64Bytes1)
	module(t, `base64`).call(rta, "raw_url_encode", base64Bytes1).expect(rta, base64RawURL)
	module(t, `base64`).call(rta, "raw_url_decode", base64RawURL).expect(rta, base64Bytes1)
}

func TestHex(t *testing.T) {
	rta := core.NewArena(nil)
	module(t, `hex`).call(rta, "encode", hexBytes1).expect(rta, hex1)
	module(t, `hex`).call(rta, "decode", hex1).expect(rta, hexBytes1)
}

func TestJSON(t *testing.T) {
	rta := core.NewArena(nil)

	module(t, "json").call(rta, "encode", 5).expect(rta, []byte("5"))
	module(t, "json").call(rta, "encode", "foobar").expect(rta, []byte(`"foobar"`))
	module(t, "json").call(rta, "encode", MAP{"foo": 5}).expect(rta, []byte("{\"foo\":5}"))
	module(t, "json").call(rta, "encode", IMAP{"foo": 5}).expect(rta, []byte("{\"foo\":5}"))
	module(t, "json").call(rta, "encode", ARR{1, 2, 3}).expect(rta, []byte("[1,2,3]"))
	module(t, "json").call(rta, "encode", IARR{1, 2, 3}).expect(rta, []byte("[1,2,3]"))
	module(t, "json").call(rta, "encode", MAP{"foo": "bar"}).expect(rta, []byte("{\"foo\":\"bar\"}"))
	module(t, "json").call(rta, "encode", MAP{"foo": 1.8}).expect(rta, []byte("{\"foo\":1.8}"))
	module(t, "json").call(rta, "encode", MAP{"foo": true}).expect(rta, []byte("{\"foo\":true}"))
	module(t, "json").call(rta, "encode", MAP{"foo": '8'}).expect(rta, []byte("{\"foo\":56}"))
	module(t, "json").call(rta, "encode", MAP{"foo": []byte("foo")}).expect(rta, []byte("{\"foo\":\"Zm9v\"}")) // json encoding returns []byte as base64 encoded string
	module(t, "json").call(rta, "encode", MAP{"foo": ARR{"bar", 1, 1.8, '8', true}}).expect(rta, []byte("{\"foo\":[\"bar\",1,1.8,56,true]}"))
	module(t, "json").call(rta, "encode", MAP{"foo": IARR{"bar", 1, 1.8, '8', true}}).expect(rta, []byte("{\"foo\":[\"bar\",1,1.8,56,true]}"))
	module(t, "json").call(rta, "encode", MAP{"foo": ARR{ARR{"bar", 1}, ARR{"bar", 1}}}).expect(rta, []byte("{\"foo\":[[\"bar\",1],[\"bar\",1]]}"))
	module(t, "json").call(rta, "encode", MAP{"foo": MAP{"string": "bar"}}).expect(rta, []byte("{\"foo\":{\"string\":\"bar\"}}"))
	module(t, "json").call(rta, "encode", MAP{"foo": IMAP{"string": "bar"}}).expect(rta, []byte("{\"foo\":{\"string\":\"bar\"}}"))
	module(t, "json").call(rta, "encode", MAP{"foo": MAP{"map1": MAP{"string": "bar"}}}).expect(rta, []byte("{\"foo\":{\"map1\":{\"string\":\"bar\"}}}"))
	module(t, "json").call(rta, "encode", ARR{ARR{"bar", 1}, ARR{"bar", 1}}).expect(rta, []byte("[[\"bar\",1],[\"bar\",1]]"))

	module(t, "json").call(rta, "decode", `5`).expect(rta, 5)
	module(t, "json").call(rta, "decode", `"foo"`).expect(rta, "foo")
	module(t, "json").call(rta, "decode", `[1,2,3,"bar"]`).expect(rta, ARR{1, 2, 3, "bar"})
	module(t, "json").call(rta, "decode", `{"foo":5}`).expect(rta, MAP{"foo": 5})
	module(t, "json").call(rta, "decode", `{"foo":2.5}`).expect(rta, MAP{"foo": 2.5})
	module(t, "json").call(rta, "decode", `{"foo":true}`).expect(rta, MAP{"foo": true})
	module(t, "json").call(rta, "decode", `{"foo":"bar"}`).expect(rta, MAP{"foo": "bar"})
	module(t, "json").call(rta, "decode", `{"foo":[1,2,3,"bar"]}`).expect(rta, MAP{"foo": ARR{1, 2, 3, "bar"}})

	module(t, "json").call(rta, "indent", []byte("{\"foo\":[\"bar\",1,1.8,56,true]}"), "", "  ").expect(rta, []byte(`{
  "foo": [
    "bar",
    1,
    1.8,
    56,
    true
  ]
}`))

	module(t, "json").call(rta, "html_escape", []byte(`{"M":"<html>foo &`+"\xe2\x80\xa8 \xe2\x80\xa9"+`</html>"}`)).
		expect(rta, []byte(`{"M":"\u003chtml\u003efoo \u0026\u2028 \u2029\u003c/html\u003e"}`))
}

func TestReadFile(t *testing.T) {
	rta := core.NewArena(nil)

	content := []byte("the quick brown fox jumps over the lazy dog")
	tf, err := os.CreateTemp("", "test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tf.Name()) }()

	_, err = tf.Write(content)
	require.NoError(t, err)
	_ = tf.Close()

	bs, err := rta.NewBytesValue(content, false)
	require.NoError(t, err)
	module(t, "os").call(rta, "read_file", tf.Name()).expect(rta, bs)
}

func TestReadFileArgs(t *testing.T) {
	rta := core.NewArena(nil)
	module(t, "os").call(rta, "read_file").expectError()
}
func TestFileStatArgs(t *testing.T) {
	rta := core.NewArena(nil)
	module(t, "os").call(rta, "stat").expectError()
}

func TestFileStatFile(t *testing.T) {
	rta := core.NewArena(nil)

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

	name, err := rta.NewStringValue(stat.Name())
	require.NoError(t, err)
	mt, err := rta.NewTimeValue(stat.ModTime())
	require.NoError(t, err)

	rec, err := rta.NewRecordValue(map[string]core.Value{
		"name":      name,
		"mtime":     mt,
		"size":      core.IntValue(stat.Size()),
		"mode":      core.IntValue(int64(stat.Mode())),
		"directory": core.False,
	}, true)
	require.NoError(t, err)
	module(t, "os").call(rta, "stat", tf.Name()).expect(rta, rec)
}

func TestFileStatDir(t *testing.T) {
	rta := core.NewArena(nil)

	td, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(td) }()

	stat, err := os.Stat(td)
	require.NoError(t, err)

	name, err := rta.NewStringValue(stat.Name())
	require.NoError(t, err)
	mt, err := rta.NewTimeValue(stat.ModTime())
	require.NoError(t, err)

	rec, err := rta.NewRecordValue(map[string]core.Value{
		"name":      name,
		"mtime":     mt,
		"size":      core.IntValue(stat.Size()),
		"mode":      core.IntValue(int64(stat.Mode())),
		"directory": core.True,
	}, true)
	require.NoError(t, err)
	module(t, "os").call(rta, "stat", td).expect(rta, rec)
}

func TestOSExpandEnv(t *testing.T) {
	rta := core.NewArena(nil)

	_ = os.Setenv("KAVUN", "FOO BAR")
	module(t, "os").call(rta, "expand_env", "$KAVUN").expect(rta, "FOO BAR")

	_ = os.Setenv("KAVUN", "FOO")
	module(t, "os").call(rta, "expand_env", "$KAVUN $KAVUN").expect(rta, "FOO FOO")

	_ = os.Setenv("KAVUN", "123456789012")
	module(t, "os").call(rta, "expand_env", "$KAVUN").expect(rta, "123456789012")

	_ = os.Setenv("KAVUN", "123456")
	module(t, "os").call(rta, "expand_env", "$KAVUN$KAVUN").expect(rta, "123456123456")

	_ = os.Setenv("KAVUN", "123456")
	module(t, "os").call(rta, "expand_env", "${KAVUN}${KAVUN}").expect(rta, "123456123456")
}

func TestTextREAlternation(t *testing.T) {
	rta := core.NewArena(nil)

	module(t, "text").call(rta, "re_find", "([a-zA-Z])|([0-9])", "a").expect(rta, ARR{
		ARR{
			IMAP{"text": "a", "begin": 0, "end": 1},
			IMAP{"text": "a", "begin": 0, "end": 1},
		},
	}, "alternation with letter")

	module(t, "text").call(rta, "re_find", "([a-zA-Z])|([0-9])", "5").expect(rta, ARR{
		ARR{
			IMAP{"text": "5", "begin": 0, "end": 1},
			IMAP{"text": "5", "begin": 0, "end": 1},
		},
	}, "alternation with number")

	module(t, "text").call(rta, "re_find", "([a-zA-Z])|([0-9])", "").expect(rta, core.Undefined, "empty input")

	module(t, "text").call(rta, "re_find", "([a-zA-Z])|([0-9])", "!").expect(rta, core.Undefined, "non-matching input")

	module(t, "text").call(rta, "re_find", "(?:([a-zA-Z])|([0-9]))+", "a5b").expect(rta, ARR{
		ARR{
			IMAP{"text": "a5b", "begin": 0, "end": 3},
			IMAP{"text": "b", "begin": 2, "end": 3},
			IMAP{"text": "5", "begin": 1, "end": 2},
		},
	}, "multiple alternations")

	module(t, "text").call(rta, "re_find", "(foo)|(bar)|(baz)", "foo").expect(rta, ARR{
		ARR{
			IMAP{"text": "foo", "begin": 0, "end": 3},
			IMAP{"text": "foo", "begin": 0, "end": 3},
		},
	}, "multiple groups with non-matches")

	module(t, "text").call(rta, "re_find", "((cat)|(dog))((run)|(walk))", "catrun").expect(rta, ARR{
		ARR{
			IMAP{"text": "catrun", "begin": 0, "end": 6},
			IMAP{"text": "cat", "begin": 0, "end": 3},
			IMAP{"text": "cat", "begin": 0, "end": 3},
			IMAP{"text": "run", "begin": 3, "end": 6},
			IMAP{"text": "run", "begin": 3, "end": 6},
		},
	}, "nested groups with alternation")
}

func TestTextRE(t *testing.T) {
	rta := core.NewArena(nil)

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
		module(t, "text").call(rta, "re_match", d.pattern, d.text).expect(rta, expected, "pattern: %q, src: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "match", d.text).expect(rta, expected, "patter: %q, src: %q", d.pattern, d.text)
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
		module(t, "text").call(rta, "re_find", d.pattern, d.text).expect(rta, d.expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "find", d.text).expect(rta, d.expected, "pattern: %q, text: %q", d.pattern, d.text)
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
		module(t, "text").call(rta, "re_find", d.pattern, d.text, d.count).expect(rta, d.expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "find", d.text, d.count).expect(rta, d.expected, "pattern: %q, text: %q", d.pattern, d.text)
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
		module(t, "text").call(rta, "re_replace", d.pattern, d.text, d.repl).expect(rta, expected, "pattern: %q, text: %q, repl: %q", d.pattern, d.text, d.repl)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "replace", d.text, d.repl).expect(rta, expected, "pattern: %q, text: %q, repl: %q", d.pattern, d.text, d.repl)
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
		module(t, "text").call(rta, "re_split", d.pattern, d.text).expect(rta, expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "split", d.text).expect(rta, expected, "pattern: %q, text: %q", d.pattern, d.text)
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
		module(t, "text").call(rta, "re_split", d.pattern, d.text, d.count).expect(rta, expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "split", d.text, d.count).expect(rta, expected, "pattern: %q, text: %q", d.pattern, d.text)
	}
}

func TestText(t *testing.T) {
	rta := core.NewArena(nil)

	module(t, "text").call(rta, "compare", "", "").expect(rta, 0)
	module(t, "text").call(rta, "compare", "", "a").expect(rta, -1)
	module(t, "text").call(rta, "compare", "a", "").expect(rta, 1)
	module(t, "text").call(rta, "compare", "a", "a").expect(rta, 0)
	module(t, "text").call(rta, "compare", "a", "b").expect(rta, -1)
	module(t, "text").call(rta, "compare", "b", "a").expect(rta, 1)
	module(t, "text").call(rta, "compare", "abcde", "abcde").expect(rta, 0)
	module(t, "text").call(rta, "compare", "abcde", "abcdf").expect(rta, -1)
	module(t, "text").call(rta, "compare", "abcdf", "abcde").expect(rta, 1)

	module(t, "text").call(rta, "contains", "", "").expect(rta, true)
	module(t, "text").call(rta, "contains", "", "a").expect(rta, false)
	module(t, "text").call(rta, "contains", "a", "").expect(rta, true)
	module(t, "text").call(rta, "contains", "a", "a").expect(rta, true)
	module(t, "text").call(rta, "contains", "abcde", "a").expect(rta, true)
	module(t, "text").call(rta, "contains", "abcde", "abcde").expect(rta, true)
	module(t, "text").call(rta, "contains", "abc", "abcde").expect(rta, false)
	module(t, "text").call(rta, "contains", "ab cd", "bc").expect(rta, false)

	module(t, "text").call(rta, "replace", "", "", "", -1).expect(rta, "")
	module(t, "text").call(rta, "replace", "abcd", "a", "x", -1).expect(rta, "xbcd")
	module(t, "text").call(rta, "replace", "aaaa", "a", "x", -1).expect(rta, "xxxx")
	module(t, "text").call(rta, "replace", "aaaa", "a", "x", 0).expect(rta, "aaaa")
	module(t, "text").call(rta, "replace", "aaaa", "a", "x", 2).expect(rta, "xxaa")
	module(t, "text").call(rta, "replace", "abcd", "bc", "x", -1).expect(rta, "axd")

	module(t, "text").call(rta, "format_bool", true).expect(rta, "true")
	module(t, "text").call(rta, "format_bool", false).expect(rta, "false")
	module(t, "text").call(rta, "format_float", -19.84, "f", -1, 64).expect(rta, "-19.84")
	module(t, "text").call(rta, "format_int", -1984, 10).expect(rta, "-1984")
	module(t, "text").call(rta, "format_int", 1984, 8).expect(rta, "3700")
	module(t, "text").call(rta, "parse_bool", "true").expect(rta, true)
	module(t, "text").call(rta, "parse_bool", "0").expect(rta, false)
	module(t, "text").call(rta, "parse_float", "-19.84", 64).expect(rta, -19.84)
	module(t, "text").call(rta, "parse_int", "-1984", 10, 64).expect(rta, -1984)
}

func TestReplace(t *testing.T) {
	rta := core.NewArena(nil)

	module(t, "text").call(rta, "replace", "123456789012", "1", "x", -1).expect(rta, "x234567890x2")
	module(t, "text").call(rta, "replace", "123456789012", "12", "x", -1).expect(rta, "x34567890x")
	module(t, "text").call(rta, "replace", "123456789012", "012", "xyz", -1).expect(rta, "123456789xyz")
	module(t, "text").call(rta, "re_replace", "1", "123456789012", "x").expect(rta, "x234567890x2")
	module(t, "text").call(rta, "re_replace", "12", "123456789012", "x").expect(rta, "x34567890x")
	module(t, "text").call(rta, "re_replace", "1(2)", "123456789012", "x$1").expect(rta, "x234567890x2")
	module(t, "text").call(rta, "re_replace", "(1)(2)", "123456789012", "$2$1").expect(rta, "213456789021")
}

func TestTextRepeat(t *testing.T) {
	rta := core.NewArena(nil)

	module(t, "text").call(rta, "repeat", "1234", "3").expect(rta, "123412341234")
	module(t, "text").call(rta, "repeat", "1", "12").expect(rta, "111111111111")
}

func TestSubstr(t *testing.T) {
	rta := core.NewArena(nil)

	module(t, "text").call(rta, "substr", "", 0, 0).expect(rta, "")
	module(t, "text").call(rta, "substr", "abcdef", 0, 3).expect(rta, "abc")
	module(t, "text").call(rta, "substr", "abcdef", 0, 6).expect(rta, "abcdef")
	module(t, "text").call(rta, "substr", "abcdef", 0, 10).expect(rta, "abcdef")
	module(t, "text").call(rta, "substr", "abcdef", -10, 10).expect(rta, "abcdef")
	module(t, "text").call(rta, "substr", "abcdef", 0).expect(rta, "abcdef")
	module(t, "text").call(rta, "substr", "abcdef", 3).expect(rta, "def")

	module(t, "text").call(rta, "substr", "", 10, 0).expectError()
	module(t, "text").call(rta, "substr", "", "10", 0).expectError()
	module(t, "text").call(rta, "substr", "", 10, "0").expectError()
	module(t, "text").call(rta, "substr", "", "10", "0").expectError()

	module(t, "text").call(rta, "substr", 0, 0, 1).expect(rta, "0")
	module(t, "text").call(rta, "substr", 123, 0, 1).expect(rta, "1")
	module(t, "text").call(rta, "substr", 123.456, 4, 7).expect(rta, "456")
}

func TestPadLeft(t *testing.T) {
	rta := core.NewArena(nil)

	module(t, "text").call(rta, "pad_left", "ab", 7, 0).expect(rta, "00000ab")
	module(t, "text").call(rta, "pad_right", "ab", 7, 0).expect(rta, "ab00000")
	module(t, "text").call(rta, "pad_left", "ab", 7, "+-").expect(rta, "-+-+-ab")
	module(t, "text").call(rta, "pad_right", "ab", 7, "+-").expect(rta, "ab+-+-+")
}

func TestTimes(t *testing.T) {
	rta := core.NewArena(nil)

	time1 := time.Date(1982, 9, 28, 19, 21, 44, 999, time.Now().Location())
	time2 := time.Now()
	location, _ := time.LoadLocation("Pacific/Auckland")
	time3 := time.Date(1982, 9, 28, 19, 21, 44, 999, location)

	module(t, "times").call(rta, "sleep", 1).expect(rta, core.Undefined)

	r := module(t, "times").call(rta, "since", time.Now().Add(-time.Hour)).o.(core.Value)
	require.True(t, r.Type == core.VT_INT)
	require.True(t, int64(r.Data) > 3600000000000)

	r = module(t, "times").call(rta, "until", time.Now().Add(time.Hour)).o.(core.Value)
	require.True(t, r.Type == core.VT_INT)
	require.True(t, int64(r.Data) < 3600000000000)

	module(t, "times").call(rta, "parse_duration", "1ns").expect(rta, 1)
	module(t, "times").call(rta, "parse_duration", "1ms").expect(rta, 1000000)
	module(t, "times").call(rta, "parse_duration", "1h").expect(rta, 3600000000000)
	module(t, "times").call(rta, "duration_hours", 1800000000000).expect(rta, 0.5)
	module(t, "times").call(rta, "duration_minutes", 1800000000000).expect(rta, 30.0)
	module(t, "times").call(rta, "duration_nanoseconds", 100).expect(rta, 100)
	module(t, "times").call(rta, "duration_seconds", 1000000).expect(rta, 0.001)
	module(t, "times").call(rta, "duration_string", 1800000000000).expect(rta, "30m0s")

	module(t, "times").call(rta, "month_string", 1).expect(rta, "January")
	module(t, "times").call(rta, "month_string", 12).expect(rta, "December")

	module(t, "times").call(rta, "date", 1982, 9, 28, 19, 21, 44, 999).expect(rta, time1)
	module(t, "times").call(rta, "date", 1982, 9, 28, 19, 21, 44, 999, "Pacific/Auckland").expect(rta, time3)

	r = module(t, "times").call(rta, "now").o.(core.Value)
	rt, _ := r.AsTime(rta)
	nowD := time.Until(rt).Nanoseconds()
	require.True(t, 0 > nowD && nowD > -100000000) // within 100ms

	parsed, _ := time.Parse(time.RFC3339, "1982-09-28T19:21:44+07:00")
	module(t, "times").call(rta, "parse", time.RFC3339, "1982-09-28T19:21:44+07:00").expect(rta, parsed)
	module(t, "times").call(rta, "unix", 1234325, 94493).expect(rta, time.Unix(1234325, 94493))

	module(t, "times").call(rta, "add", time2, 3600000000000).expect(rta, time2.Add(time.Duration(3600000000000)))
	module(t, "times").call(rta, "sub", time2, time2.Add(-time.Hour)).expect(rta, 3600000000000)
	module(t, "times").call(rta, "add_date", time2, 1, 2, 3).expect(rta, time2.AddDate(1, 2, 3))
	module(t, "times").call(rta, "after", time2, time2.Add(time.Hour)).expect(rta, false)
	module(t, "times").call(rta, "after", time2, time2.Add(-time.Hour)).expect(rta, true)
	module(t, "times").call(rta, "before", time2, time2.Add(time.Hour)).expect(rta, true)
	module(t, "times").call(rta, "before", time2, time2.Add(-time.Hour)).expect(rta, false)

	module(t, "times").call(rta, "time_year", time1).expect(rta, time1.Year())
	module(t, "times").call(rta, "time_month", time1).expect(rta, int(time1.Month()))
	module(t, "times").call(rta, "time_day", time1).expect(rta, time1.Day())
	module(t, "times").call(rta, "time_hour", time1).expect(rta, time1.Hour())
	module(t, "times").call(rta, "time_minute", time1).expect(rta, time1.Minute())
	module(t, "times").call(rta, "time_second", time1).expect(rta, time1.Second())
	module(t, "times").call(rta, "time_nanosecond", time1).expect(rta, time1.Nanosecond())
	module(t, "times").call(rta, "time_unix", time1).expect(rta, time1.Unix())
	module(t, "times").call(rta, "time_unix_nano", time1).expect(rta, time1.UnixNano())
	module(t, "times").call(rta, "time_format", time1, time.RFC3339).expect(rta, time1.Format(time.RFC3339))
	module(t, "times").call(rta, "is_zero", time1).expect(rta, false)
	module(t, "times").call(rta, "is_zero", time.Time{}).expect(rta, true)
	module(t, "times").call(rta, "to_local", time1).expect(rta, time1.Local())
	module(t, "times").call(rta, "to_utc", time1).expect(rta, time1.UTC())
	module(t, "times").call(rta, "time_location", time1).expect(rta, time1.Location().String())
	module(t, "times").call(rta, "time_string", time1).expect(rta, time1.String())
	module(t, "times").call(rta, "in_location", time1, location.String()).expect(rta, time1.In(location))
}
