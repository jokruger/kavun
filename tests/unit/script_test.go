package unit

import (
	"context"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/vm"
)

func add(s *kavun.Script, name string, value any) error {
	v, err := require.FromInterface(alloc, value)
	if err != nil {
		return err
	}
	s.Add(name, v)
	return nil
}

func set(c *kavun.Compiled, name string, value any) error {
	v, err := require.FromInterface(alloc, value)
	if err != nil {
		return err
	}
	return c.Set(name, v)
}

func TestScript_Add(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`a := b; c := test(b); d := test(5)`))
	require.NoError(t, add(s, "b", 5))     // b = 5
	require.NoError(t, add(s, "b", "foo")) // b = "foo"  (re-define before compilation)
	require.NoError(t, add(s, "test",
		func(v core.VM, args []core.Value) (ret core.Value, err error) {
			if len(args) > 0 {
				if args[0].Type == core.VT_INT {
					return core.IntValue(int64(args[0].Data) + 1), nil
				}
			}
			return core.IntValue(0), nil
		}))
	c, err := s.Compile(alloc)
	require.NoError(t, err)
	require.NoError(t, c.Run(alloc, machine))
	r := c.Get("a").Value()
	require.Equal(t, alloc, "foo", r.Interface(alloc))
	r = c.Get("b").Value()
	require.Equal(t, alloc, "foo", r.Interface(alloc))
	r = c.Get("c").Value()
	require.Equal(t, alloc, int64(0), r.Interface(alloc))
	r = c.Get("d").Value()
	require.Equal(t, alloc, int64(6), r.Interface(alloc))
}

func TestScript_Remove(t *testing.T) {
	s := kavun.NewScript([]byte(`a := b`))
	err := add(s, "b", 5)
	require.NoError(t, err)
	require.True(t, s.Remove("b")) // b is removed
	_, err = s.Compile(nil)        // should not compile because b is undefined
	require.Error(t, err)
}

func TestScript_Run(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`a := b`))
	err := add(s, "b", 5)
	require.NoError(t, err)
	c, err := s.Compile(alloc)
	require.NoError(t, err)
	err = c.Run(alloc, machine)
	require.NoError(t, err)
	require.NotNil(t, c)
	compiledGet(t, c, "a", int64(5))
}

func TestScript_RecurrentRun(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`
count += 1
arr[0] += step
out = count + arr[0]
`))

	require.NoError(t, add(s, "count", 0))
	require.NoError(t, add(s, "arr", []any{10}))
	require.NoError(t, add(s, "step", 1))
	require.NoError(t, add(s, "out", nil))

	c, err := s.Compile(alloc)
	require.NoError(t, err)

	// Run #1: uses initial globals.
	require.NoError(t, c.Run(alloc, machine))
	require.Equal(t, alloc, int64(1), c.Get("count").Int(alloc))
	require.Equal(t, alloc, int64(11), c.Get("arr").Array(alloc)[0])
	require.Equal(t, alloc, int64(12), c.Get("out").Int(alloc))

	// Run #2 without Set: runtime must reset from compile-time globals.
	require.NoError(t, c.Run(alloc, machine))
	require.Equal(t, alloc, int64(1), c.Get("count").Int(alloc))
	require.Equal(t, alloc, int64(11), c.Get("arr").Array(alloc)[0])
	require.Equal(t, alloc, int64(12), c.Get("out").Int(alloc))

	// Update compile-time globals and verify recurrent runs use updated values.
	require.NoError(t, set(c, "count", 100))
	require.NoError(t, set(c, "arr", []any{1}))
	require.NoError(t, set(c, "step", 2))

	require.NoError(t, c.Run(alloc, machine))
	require.Equal(t, alloc, int64(101), c.Get("count").Int(alloc))
	require.Equal(t, alloc, int64(3), c.Get("arr").Array(alloc)[0])
	require.Equal(t, alloc, int64(104), c.Get("out").Int(alloc))

	// Run again without Set: should repeat the same result.
	require.NoError(t, c.Run(alloc, machine))
	require.Equal(t, alloc, int64(101), c.Get("count").Int(alloc))
	require.Equal(t, alloc, int64(3), c.Get("arr").Array(alloc)[0])
	require.Equal(t, alloc, int64(104), c.Get("out").Int(alloc))
}

func TestScript_SetAssignmentMode(t *testing.T) {
	s := kavun.NewScript([]byte(`a = 1`))
	_, err := s.Compile(alloc)
	require.NoError(t, err)

	s = kavun.NewScript([]byte(`a = 1`))
	s.SetAssignmentMode(compiler.AssignmentModeStrict)
	_, err = s.Compile(alloc)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unresolved reference 'a'"))

	s = kavun.NewScript([]byte(`a += 1`))
	_, err = s.Compile(alloc)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unresolved reference 'a'"))
}

func TestScript_BuiltinModules(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`math := import("math"); a := math.abs(-19.84)`))
	s.SetImports(stdlib.GetModuleMap("math"))
	c, err := s.Compile(alloc)
	require.NoError(t, err)
	err = c.Run(alloc, machine)
	require.NoError(t, err)
	require.NotNil(t, c)
	compiledGet(t, c, "a", 19.84)

	c, err = s.Compile(alloc)
	require.NoError(t, err)
	err = c.Run(alloc, machine)
	require.NoError(t, err)
	require.NotNil(t, c)
	compiledGet(t, c, "a", 19.84)

	s.SetImports(stdlib.GetModuleMap("os"))
	_, err = s.Compile(alloc)
	require.Error(t, err)

	s.SetImports(nil)
	_, err = s.Compile(alloc)
	require.Error(t, err)
}

/*
func TestScript_SourceModules(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`
enum := import("enum")
a := enum.all([1,2,3], func(_, v) {
	return v > 0
})
`))
	s.SetImports(stdlib.GetModuleMap("enum"))
	c, err := s.Compile(alloc)
	require.NoError(t, err)
	err = c.Run(alloc, machine)
	require.NoError(t, err)
	require.NotNil(t, c)
	compiledGet(t, c, "a", true)

	s.SetImports(nil)
	_, err = s.Compile(alloc)
	require.Error(t, err)
}
*/

func TestScript_SetMaxConstObjects(t *testing.T) {
	// one constant '5'
	s := kavun.NewScript([]byte(`a := 5`))
	s.SetMaxConstObjects(1) // limit = 1
	_, err := s.Compile(alloc)
	require.NoError(t, err)
	s.SetMaxConstObjects(0) // limit = 0
	_, err = s.Compile(alloc)
	require.Error(t, err)
	require.Equal(t, alloc, "exceeding constant objects limit: 1", err.Error())

	// two constants '5' and '1'
	s = kavun.NewScript([]byte(`a := 5 + 1`))
	s.SetMaxConstObjects(2) // limit = 2
	_, err = s.Compile(alloc)
	require.NoError(t, err)
	s.SetMaxConstObjects(1) // limit = 1
	_, err = s.Compile(alloc)
	require.Error(t, err)
	require.Equal(t, alloc, "exceeding constant objects limit: 2", err.Error())

	// duplicates will be removed
	s = kavun.NewScript([]byte(`a := 5 + 5`))
	s.SetMaxConstObjects(1) // limit = 1
	_, err = s.Compile(alloc)
	require.NoError(t, err)
	s.SetMaxConstObjects(0) // limit = 0
	_, err = s.Compile(alloc)
	require.Error(t, err)
	require.Equal(t, alloc, "exceeding constant objects limit: 1", err.Error())

	// no limit set
	s = kavun.NewScript([]byte(`a := 1 + 2 + 3 + 4 + 5`))
	_, err = s.Compile(alloc)
	require.NoError(t, err)
}

func TestScriptConcurrency(t *testing.T) {
	solve := func(a, b, c int) (d, e int) {
		a += 2
		b += c
		a += b * 2
		d = a + b + c
		e = 0
		for i := 1; i <= d; i++ {
			e += i
		}
		e *= 2
		return
	}

	code := []byte(`
mod1 := import("mod1")

a += 2
b += c
a += b * 2

arr := [a, b, c]
arrstr := string(arr)
map1 := {a: a, b: b, c: c}

d := a + b + c
s := 0

for i:=1; i<=d; i++ {
	s += i
}

e := mod1.double(s)
`)
	mod1 := map[string]core.Value{
		"double": core.NewBuiltinFunctionValue(
			"unknown",
			func(a *core.Arena, v core.VM, args []core.Value) (ret core.Value, err error) {
				arg0, _ := args[0].AsInt(a)
				ret = core.IntValue(arg0 * 2)
				return
			},
			1,
			false,
		),
	}

	scr := kavun.NewScript(code)
	_ = add(scr, "a", 0)
	_ = add(scr, "b", 0)
	_ = add(scr, "c", 0)
	mods := vm.NewModuleMap()
	mods.AddBuiltinModule("mod1", mod1)
	scr.SetImports(mods)
	compiled, err := scr.Compile(nil)
	require.NoError(t, err)

	concurrency := 500

	// own vm and allocator
	var wg1 sync.WaitGroup
	wg1.Add(concurrency)
	for range concurrency {
		cln, err := compiled.Clone(alloc)
		require.NoError(t, err)
		go func(compiled *kavun.Compiled) {
			machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

			time.Sleep(time.Duration(rand.Int63n(50)) * time.Millisecond)
			defer wg1.Done()

			a := rand.Intn(10)
			b := rand.Intn(10)
			c := rand.Intn(10)

			av, _ := require.FromInterface(alloc, a)
			bv, _ := require.FromInterface(alloc, b)
			cv, _ := require.FromInterface(alloc, c)
			_ = compiled.Set("a", av)
			_ = compiled.Set("b", bv)
			_ = compiled.Set("c", cv)
			err := compiled.Run(alloc, machine)
			require.NoError(t, err)
			d := int(compiled.Get("d").Int(alloc))
			e := int(compiled.Get("e").Int(alloc))

			expectedD, expectedE := solve(a, b, c)

			require.Equal(t, alloc, expectedD, d, "input: %d, %d, %d", a, b, c)
			require.Equal(t, alloc, expectedE, e, "input: %d, %d, %d", a, b, c)
		}(cln)
	}
	wg1.Wait()

	// shared vm and allocator
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	var lock sync.RWMutex
	var wg2 sync.WaitGroup
	wg2.Add(concurrency)
	for range concurrency {
		cln, err := compiled.Clone(alloc)
		require.NoError(t, err)
		go func(compiled *kavun.Compiled) {
			time.Sleep(time.Duration(rand.Int63n(50)) * time.Millisecond)
			defer wg2.Done()

			a := rand.Intn(10)
			b := rand.Intn(10)
			c := rand.Intn(10)

			av, _ := require.FromInterface(alloc, a)
			bv, _ := require.FromInterface(alloc, b)
			cv, _ := require.FromInterface(alloc, c)
			_ = compiled.Set("a", av)
			_ = compiled.Set("b", bv)
			_ = compiled.Set("c", cv)

			lock.Lock()
			err := compiled.Run(alloc, machine)
			lock.Unlock()

			require.NoError(t, err)
			d := int(compiled.Get("d").Int(alloc))
			e := int(compiled.Get("e").Int(alloc))

			expectedD, expectedE := solve(a, b, c)

			require.Equal(t, alloc, expectedD, d, "input: %d, %d, %d", a, b, c)
			require.Equal(t, alloc, expectedE, e, "input: %d, %d, %d", a, b, c)
		}(cln)
	}
	wg2.Wait()
}

func TestScript_CustomObjects(t *testing.T) {
	c := compile(t, `a := c1(); s := string(c1); c2 := c1; c2++`, M{"c1": NewCounterValue(5)})
	compiledRun(t, c)
	compiledGet(t, c, "a", int64(5))
	compiledGet(t, c, "s", "Counter(5)")
	compiledGetCounter(t, c, "c2", &Counter{value: 6})

	c = compile(t, `
arr := [1, 2, 3, 4]
for x in arr {
	c1 += x
}
out := c1()
`, M{
		"c1": NewCounterValue(5),
	})
	compiledRun(t, c)
	compiledGet(t, c, "out", int64(15))
}

func compiledGetCounter(t *testing.T, c *kavun.Compiled, name string, expected *Counter) {
	v := c.Get(name)
	require.NotNil(t, v)

	val := v.Value()
	actual := toCounter(alloc, val)
	require.NotNil(t, actual)
	require.Equal(t, alloc, expected.value, actual.value)
}

func TestScriptSourceModule(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	// script1 imports "mod1"
	scr := kavun.NewScript([]byte(`out := import("mod")`))
	mods := vm.NewModuleMap()
	mods.AddSourceModule("mod", []byte(`export 5`))
	scr.SetImports(mods)
	c, err := scr.Compile(alloc)
	require.NoError(t, err)
	err = c.Run(alloc, machine)
	require.NoError(t, err)
	v := c.Get("out").Value()
	require.Equal(t, alloc, int64(5), v.Interface(alloc))

	// executing module function
	scr = kavun.NewScript([]byte(`fn := import("mod"); out := fn()`))
	mods = vm.NewModuleMap()
	mods.AddSourceModule("mod", []byte(`a := 3; export func() { return a + 5 }`))
	scr.SetImports(mods)
	c, err = scr.Compile(alloc)
	require.NoError(t, err)
	err = c.Run(alloc, machine)
	require.NoError(t, err)
	v = c.Get("out").Value()
	require.Equal(t, alloc, int64(8), v.Interface(alloc))

	scr = kavun.NewScript([]byte(`out := import("mod")`))
	mods = vm.NewModuleMap()
	mods.AddSourceModule("mod", []byte(`text := import("text"); export text.title("foo")`))
	mods.AddBuiltinModule("text", map[string]core.Value{
		"title": core.NewBuiltinFunctionValue(
			"title",
			func(a *core.Arena, v core.VM, args []core.Value) (core.Value, error) {
				s, _ := args[0].AsString(a)
				return core.NewStringValue(strings.Title(s)), nil
			},
			1,
			false,
		),
	})
	scr.SetImports(mods)
	c, err = scr.Compile(alloc)
	require.NoError(t, err)
	err = c.Run(alloc, machine)
	require.NoError(t, err)
	v = c.Get("out").Value()
	require.Equal(t, alloc, "Foo", v.Interface(alloc))
	scr.SetImports(nil)
	_, err = scr.Compile(alloc)
	require.Error(t, err)
}

func BenchmarkArrayIndex(b *testing.B) {
	bench(b.N, `a := [1, 2, 3, 4, 5, 6, 7, 8, 9];
        for i := 0; i < 1000; i++ {
            a[0]; a[1]; a[2]; a[3]; a[4]; a[5]; a[6]; a[7]; a[7];
        }
    `)
}

func BenchmarkArrayIndexCompare(b *testing.B) {
	bench(b.N, `a := [1, 2, 3, 4, 5, 6, 7, 8, 9];
        for i := 0; i < 1000; i++ {
            1; 2; 3; 4; 5; 6; 7; 8; 9;
        }
    `)
}

func bench(n int, input string) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(input))
	c, err := s.Compile(alloc)
	if err != nil {
		panic(err)
	}

	for range n {
		if err := c.Run(alloc, machine); err != nil {
			panic(err)
		}
	}
}

type M map[string]any

func TestCompiled_Get(t *testing.T) {
	// simple script
	c := compile(t, `a := 5`, nil)
	compiledRun(t, c)
	compiledGet(t, c, "a", int64(5))

	// user-defined variables
	compileError(t, `a := b`, nil)          // compile error because "b" is not defined
	c = compile(t, `a := b`, M{"b": "foo"}) // now compile with b = "foo" defined
	compiledGet(t, c, "a", nil)             // a = undefined; because it's before Compiled.Run()
	compiledRun(t, c)                       // Compiled.Run()
	compiledGet(t, c, "a", "foo")           // a = "foo"
}

func TestCompiled_GetAll(t *testing.T) {
	c := compile(t, `a := 5`, nil)
	compiledRun(t, c)
	compiledGetAll(t, c, M{"a": int64(5)})

	c = compile(t, `a := b`, M{"b": "foo"})
	compiledRun(t, c)
	compiledGetAll(t, c, M{"a": "foo", "b": "foo"})

	c = compile(t, `a := b; b = 5`, M{"b": "foo"})
	compiledRun(t, c)
	compiledGetAll(t, c, M{"a": "foo", "b": int64(5)})
}

func Test_IsDefined(t *testing.T) {
	c := compile(t, `a := 5`, nil)
	compiledRun(t, c)
	v := c.GetValue("a")
	require.Equal(t, alloc, core.VT_INT, v.Type)
	require.Equal(t, alloc, int(5), int(v.Data))
	v = c.GetValue("b")
	require.Equal(t, alloc, core.VT_UNDEFINED, v.Type)
}

func TestCompiled_Set(t *testing.T) {
	c := compile(t, `a := b`, M{"b": "foo"})
	compiledRun(t, c)
	compiledGet(t, c, "a", "foo")

	// replace value of 'b'
	err := set(c, "b", "bar")
	require.NoError(t, err)
	compiledRun(t, c)
	compiledGet(t, c, "a", "bar")

	// try to replace undefined_variable
	err = set(c, "c", 1984)
	require.Error(t, err) // 'c' is not defined

	// case #2
	c = compile(t, `
a := func() {
	return func() {
		return b + 5
	}()
}()`, M{"b": 5})
	compiledRun(t, c)
	compiledGet(t, c, "a", int64(10))
	err = set(c, "b", 10)
	require.NoError(t, err)
	compiledRun(t, c)
	compiledGet(t, c, "a", int64(15))
}

func TestCompiled_RunContext(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	// machine completes normally
	c := compile(t, `a := 5`, nil)
	err := c.RunContext(context.Background(), alloc, machine)
	require.NoError(t, err)
	compiledGet(t, c, "a", int64(5))

	// timeout
	c = compile(t, `for true {}`, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	err = c.RunContext(ctx, alloc, machine)
	require.Equal(t, alloc, context.DeadlineExceeded, err)
}

func TestCompiled_CustomObject(t *testing.T) {
	c := compile(t, `r := (t<130)`, M{"t": NewCustomNumberValue(123)})
	compiledRun(t, c)
	compiledGet(t, c, "r", true)

	c = compile(t, `r := (t>13)`, M{"t": NewCustomNumberValue(123)})
	compiledRun(t, c)
	compiledGet(t, c, "r", true)
}

func TestScript_ImportError(t *testing.T) {
	m := `
	exp := import("expression")
	r := exp(ctx)
`

	src := `
export func(ctx) {
	closure := func() {
		if ctx.actiontimes < 0 { // an error is thrown here because actiontimes is undefined
			return true
		}
		return false
	}

	return closure()
}`

	s := kavun.NewScript([]byte(m))
	mods := vm.NewModuleMap()
	mods.AddSourceModule("expression", []byte(src))
	s.SetImports(mods)

	err := add(s, "ctx", map[string]any{
		"ctx": 12,
	})
	require.NoError(t, err)

	err = run(s)
	require.True(t, strings.Contains(err.Error(), "expression:4:6"))
}

func run(s *kavun.Script) error {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	c, err := s.Compile(alloc)
	if err != nil {
		return err
	}
	err = c.Run(alloc, machine)
	if err != nil {
		return err
	}
	return nil
}

func compile(t *testing.T, input string, vars M) *kavun.Compiled {
	s := kavun.NewScript([]byte(input))
	for vn, vv := range vars {
		err := add(s, vn, vv)
		require.NoError(t, err)
	}

	c, err := s.Compile(nil)
	require.NoError(t, err)
	require.NotNil(t, c)
	return c
}

func compileError(t *testing.T, input string, vars M) {
	s := kavun.NewScript([]byte(input))
	for vn, vv := range vars {
		err := add(s, vn, vv)
		require.NoError(t, err)
	}
	_, err := s.Compile(nil)
	require.Error(t, err)
}

func compiledRun(t *testing.T, c *kavun.Compiled) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	err := c.Run(alloc, machine)
	require.NoError(t, err)
}

func compiledGet(t *testing.T, c *kavun.Compiled, name string, expected any) {
	e, err := require.FromInterface(alloc, expected)
	require.NoError(t, err)
	v := c.Get(name)
	require.NotNil(t, v)
	require.Equal(t, alloc, e, v.Value())
}

func compiledGetAll(t *testing.T, c *kavun.Compiled, expected M) {
	vars := c.GetAll()
	require.Equal(t, alloc, len(expected), len(vars))

	for k, ev := range expected {
		v, err := require.FromInterface(alloc, ev)
		require.NoError(t, err)
		var found bool
		for _, e := range vars {
			if e.Name() == k {
				require.Equal(t, alloc, v, e.Value())
				found = true
			}
		}
		require.True(t, found, "variable '%s' not found", k)
	}
}

func TestCompiled_Clone1(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	script := kavun.NewScript([]byte(`
count += 1
data["b"] = 2
`))

	err := add(script, "data", map[string]any{"a": 1})
	require.NoError(t, err)
	err = add(script, "count", 1000)
	require.NoError(t, err)

	compiled, err := script.Compile(alloc)
	require.NoError(t, err)
	err = compiled.Run(alloc, machine)
	require.NoError(t, err)

	clone, err := compiled.Clone(alloc)
	require.NoError(t, err)
	err = clone.Run(alloc, machine)
	require.NoError(t, err)

	require.Equal(t, alloc, int64(1001), compiled.Get("count").Int(alloc))
	require.Equal(t, alloc, 2, len(compiled.Get("data").Map(alloc)))

	require.Equal(t, alloc, int64(1001), clone.Get("count").Int(alloc))
	require.Equal(t, alloc, 2, len(clone.Get("data").Map(alloc)))
}

func TestCompiled_Clone2(t *testing.T) {
	script := kavun.NewScript([]byte(`
count += 1
data["b"] = 2
`))

	err := add(script, "data", map[string]any{"a": 1})
	require.NoError(t, err)
	err = add(script, "count", 1000)
	require.NoError(t, err)

	compiled, err := script.Compile(nil)
	require.NoError(t, err)
	vm1 := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	err = compiled.Run(alloc, vm1)
	require.NoError(t, err)

	clone, err := compiled.Clone(nil)
	require.NoError(t, err)
	vm2 := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	err = clone.Run(alloc, vm2)
	require.NoError(t, err)

	require.Equal(t, alloc, int64(1001), compiled.Get("count").Int(alloc))
	require.Equal(t, alloc, 2, len(compiled.Get("data").Map(alloc)))

	require.Equal(t, alloc, int64(1001), clone.Get("count").Int(alloc))
	require.Equal(t, alloc, 2, len(clone.Get("data").Map(alloc)))
}

// Verifies that reassigning a builtin in a script does not leak across independently compiled scripts that share the
// same VM, and that the builtin remains accessible by name in scripts that do not reassign it.
func TestScript_BuiltinReassign_VMReuse(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	// Script A reassigns the builtin `len` to a constant.
	sA := kavun.NewScript([]byte(`len = 42; out = len`))
	require.NoError(t, add(sA, "out", nil))
	cA, err := sA.Compile(alloc)
	require.NoError(t, err)
	require.NoError(t, cA.Run(alloc, machine))
	require.Equal(t, alloc, int64(42), cA.Get("out").Int(alloc))

	// Script B uses the builtin `len` on the same VM. It must see the original builtin, unaffected by Script A's
	// reassignment.
	sB := kavun.NewScript([]byte(`out = len("hello")`))
	require.NoError(t, add(sB, "out", nil))
	cB, err := sB.Compile(alloc)
	require.NoError(t, err)
	require.NoError(t, cB.Run(alloc, machine))
	require.Equal(t, alloc, int64(5), cB.Get("out").Int(alloc))

	// Re-running Script A again on the same VM still reassigns to 42.
	require.NoError(t, cA.Run(alloc, machine))
	require.Equal(t, alloc, int64(42), cA.Get("out").Int(alloc))

	// Re-running Script B again still uses the builtin.
	require.NoError(t, cB.Run(alloc, machine))
	require.Equal(t, alloc, int64(5), cB.Get("out").Int(alloc))
}

// Verifies that re-running the same Compiled script multiple times restores the global slot that backs the reassigned
// builtin from compile-time globals on every run.
func TestScript_BuiltinReassign_RecurrentRun(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`
before := len("abc")
len = 100
after := len
out = before + after
`))
	require.NoError(t, add(s, "out", nil))

	c, err := s.Compile(alloc)
	require.NoError(t, err)

	for range 3 {
		require.NoError(t, c.Run(alloc, machine))
		require.Equal(t, alloc, int64(103), c.Get("out").Int(alloc))
	}
}

// Verifies that builtins shadowed in function-local scopes do not leak to outer scopes.
func TestScript_BuiltinShadow_Scopes(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`
inner := func() {
    len := 99
    return len
}
shadowed := inner()
builtin_after := len("ab")
out = shadowed + builtin_after
`))
	require.NoError(t, add(s, "out", nil))

	c, err := s.Compile(alloc)
	require.NoError(t, err)
	require.NoError(t, c.Run(alloc, machine))
	require.Equal(t, alloc, int64(101), c.Get("out").Int(alloc))
}

// Verifies that reassigning a builtin in the main script does not affect imported modules: the module compiles with its
// own fresh symbol table seeded from the original builtins.
func TestScript_BuiltinReassign_ModuleIsolation(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	scr := kavun.NewScript([]byte(`
len = 999
fn := import("mod")
out = fn("abcd")
`))
	mods := vm.NewModuleMap()
	mods.AddSourceModule("mod", []byte(`export func(s) { return len(s) }`))
	scr.SetImports(mods)
	require.NoError(t, add(scr, "out", nil))

	c, err := scr.Compile(alloc)
	require.NoError(t, err)
	require.NoError(t, c.Run(alloc, machine))
	require.Equal(t, alloc, int64(4), c.Get("out").Int(alloc))
}
