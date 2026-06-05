package stdlib

import (
	"fmt"
	"testing"
	"time"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/stdlib"
	mock "github.com/jokruger/kavun/tests"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/vm"
)

type ARR = []any
type MAP = map[string]any
type IARR []any
type IMAP map[string]any

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

type callres struct {
	t *testing.T
	o any
	e error
}

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
			r := (*core.Dict)(o.Ptr)

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
	require.Equal(c.t, rta, object(a, expected), c.o, msgAndArgs...)
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
		return a.NewStringValue(v)
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
		return a.NewBytesValue(v, false)
	case MAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = object(a, v)
		}
		return a.NewRecordValue(objs, false)
	case ARR:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, object(a, e))
		}
		return a.NewArrayValue(objs, false)
	case IMAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = object(a, v)
		}
		return a.NewRecordValue(objs, true)
	case IARR:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, object(a, e))
		}
		return a.NewArrayValue(objs, true)
	case time.Time:
		return a.NewTimeValue(v)
	case []int:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, core.IntValue(int64(e)))
		}
		return a.NewArrayValue(objs, false)
	}

	panic(fmt.Errorf("unknown type: %T", v))
}

func expect(t *testing.T, input string, expected any) {
	rta := core.NewArena(nil)
	eta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	e, err := require.FromInterface(eta, expected)
	require.NoError(t, err)
	s := kavun.NewScript([]byte(input))
	c, err := s.Compile(cta)
	require.NoError(t, err)
	err = c.Run(rta, machine)
	require.NoError(t, err)
	require.NotNil(t, c)
	v := c.Get("out")
	require.NotNil(t, v)
	require.Equal(t, rta, e, v.Value())
}
