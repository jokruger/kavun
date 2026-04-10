package stdlib

import (
	"fmt"
	"testing"
	"time"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/stdlib"
	mock "github.com/jokruger/gs/tests"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/vm"
)

type ARR = []any
type MAP = map[string]any
type IARR []any
type IMAP map[string]any

func TestAllModuleNames(t *testing.T) {
	names := stdlib.AllModuleNames()
	require.Equal(t,
		len(stdlib.BuiltinModules)+len(stdlib.SourceModules),
		len(names))
}

func TestModulesRun(t *testing.T) {
	// os.File
	expect(t, `
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
	expect(t, `
out := ""
os := import("os")
cmd := os.exec("echo", "foo", "bar")
if !is_error(cmd) {
	out = cmd.output()
}
`, []byte("foo bar\n"))

}

func TestGetModules(t *testing.T) {
	mods := stdlib.GetModuleMap()
	require.Equal(t, 0, mods.Len())

	mods = stdlib.GetModuleMap("os")
	require.Equal(t, 1, mods.Len())
	require.NotNil(t, mods.Get("os"))

	mods = stdlib.GetModuleMap("os", "rand")
	require.Equal(t, 2, mods.Len())
	require.NotNil(t, mods.Get("os"))
	require.NotNil(t, mods.Get("rand"))

	mods = stdlib.GetModuleMap("text", "text")
	require.Equal(t, 1, mods.Len())
	require.NotNil(t, mods.Get("text"))

	mods = stdlib.GetModuleMap("nonexisting", "text")
	require.Equal(t, 1, mods.Len())
	require.NotNil(t, mods.Get("text"))
}

type callres struct {
	t *testing.T
	o any
	e error
}

func (c callres) call(funcName string, args ...any) callres {
	if c.e != nil {
		return c
	}

	var oargs []core.Value
	for _, v := range args {
		oargs = append(oargs, object(v))
	}

	v := mock.Vm

	if o, ok := c.o.(*vm.Module); ok {
		m, ok := o.Attrs[funcName]
		if !ok {
			return callres{t: c.t, e: fmt.Errorf("function not found: %s", funcName)}
		}

		if !m.IsBuiltinFunction() {
			return callres{t: c.t, e: fmt.Errorf("non-callable: %s", funcName)}
		}

		res, err := m.Call(v, oargs)
		return callres{t: c.t, o: res, e: err}
	}

	if o, ok := c.o.(core.Value); ok {
		if o.IsBuiltinFunction() {
			res, err := o.Call(v, oargs)
			return callres{t: c.t, o: res, e: err}
		}

		if o.IsRecord() {
			r := (*core.Record)(o.Ptr)

			m, ok := r.Get(funcName)
			if !ok {
				return callres{t: c.t, e: fmt.Errorf("function not found: %s", funcName)}
			}

			if !m.IsBuiltinFunction() {
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
	mod := stdlib.GetModuleMap(moduleName).GetBuiltinModule(moduleName)
	if mod == nil {
		return callres{t: t, e: fmt.Errorf("module not found: %s", moduleName)}
	}

	return callres{t: t, o: mod}
}

func object(v any) core.Value {
	switch v := v.(type) {
	case core.Value:
		return v
	case string:
		return alloc.NewStringValue(v)
	case int64:
		return core.IntValue(v)
	case int: // for convenience
		return core.IntValue(int64(v))
	case bool:
		return core.BoolValue(v)
	case rune:
		return core.CharValue(v)
	case byte: // for convenience
		return core.CharValue(rune(v))
	case float64:
		return core.FloatValue(v)
	case []byte:
		return alloc.NewBytesValue(v)
	case MAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = object(v)
		}
		return alloc.NewRecordValue(objs, false)
	case ARR:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, object(e))
		}
		return alloc.NewArrayValue(objs, false)
	case IMAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = object(v)
		}
		return alloc.NewRecordValue(objs, true)
	case IARR:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, object(e))
		}
		return alloc.NewArrayValue(objs, true)
	case time.Time:
		return alloc.NewTimeValue(v)
	case []int:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, core.IntValue(int64(e)))
		}
		return alloc.NewArrayValue(objs, false)
	}

	panic(fmt.Errorf("unknown type: %T", v))
}

func expect(t *testing.T, input string, expected any) {
	e, err := require.FromInterface(alloc, expected)
	require.NoError(t, err)
	s := gs.NewScript(alloc, []byte(input))
	s.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
	c, err := s.Run()
	require.NoError(t, err)
	require.NotNil(t, c)
	v := c.Get("out")
	require.NotNil(t, v)
	require.Equal(t, e, v.Value())
}
