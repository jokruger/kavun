package unit

import (
	"context"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/stdlib"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/vm"
)

func add(s *gs.Script, name string, value any) error {
	v, err := require.FromInterface(alloc, value)
	if err != nil {
		return err
	}
	s.Add(name, v)
	return nil
}

func set(c *gs.Compiled, name string, value any) error {
	v, err := require.FromInterface(alloc, value)
	if err != nil {
		return err
	}
	return c.Set(name, v)
}

func TestScript_Add(t *testing.T) {
	s := gs.NewScript(alloc, []byte(`a := b; c := test(b); d := test(5)`))
	require.NoError(t, add(s, "b", 5))     // b = 5
	require.NoError(t, add(s, "b", "foo")) // b = "foo"  (re-define before compilation)
	require.NoError(t, add(s, "test",
		func(v core.VM, args []core.Value) (ret core.Value, err error) {
			if len(args) > 0 {
				if args[0].IsInt() {
					return core.IntValue(args[0].Int() + 1), nil
				}
			}
			return core.IntValue(0), nil
		}))
	c, err := s.Compile()
	require.NoError(t, err)
	require.NoError(t, c.Run())
	r := c.Get("a").Value()
	require.Equal(t, "foo", r.Interface())
	r = c.Get("b").Value()
	require.Equal(t, "foo", r.Interface())
	r = c.Get("c").Value()
	require.Equal(t, int64(0), r.Interface())
	r = c.Get("d").Value()
	require.Equal(t, int64(6), r.Interface())
}

func TestScript_Remove(t *testing.T) {
	s := gs.NewScript(alloc, []byte(`a := b`))
	err := add(s, "b", 5)
	require.NoError(t, err)
	require.True(t, s.Remove("b")) // b is removed
	_, err = s.Compile()           // should not compile because b is undefined
	require.Error(t, err)
}

func TestScript_Run(t *testing.T) {
	s := gs.NewScript(alloc, []byte(`a := b`))
	err := add(s, "b", 5)
	require.NoError(t, err)
	c, err := s.Run()
	require.NoError(t, err)
	require.NotNil(t, c)
	compiledGet(t, c, "a", int64(5))
}

func TestScript_BuiltinModules(t *testing.T) {
	s := gs.NewScript(alloc, []byte(`math := import("math"); a := math.abs(-19.84)`))
	s.SetImports(stdlib.GetModuleMap("math"))
	c, err := s.Run()
	require.NoError(t, err)
	require.NotNil(t, c)
	compiledGet(t, c, "a", 19.84)

	c, err = s.Run()
	require.NoError(t, err)
	require.NotNil(t, c)
	compiledGet(t, c, "a", 19.84)

	s.SetImports(stdlib.GetModuleMap("os"))
	_, err = s.Run()
	require.Error(t, err)

	s.SetImports(nil)
	_, err = s.Run()
	require.Error(t, err)
}

func TestScript_SourceModules(t *testing.T) {
	s := gs.NewScript(alloc, []byte(`
enum := import("enum")
a := enum.all([1,2,3], func(_, v) { 
	return v > 0 
})
`))
	s.SetImports(stdlib.GetModuleMap("enum"))
	c, err := s.Run()
	require.NoError(t, err)
	require.NotNil(t, c)
	compiledGet(t, c, "a", true)

	s.SetImports(nil)
	_, err = s.Run()
	require.Error(t, err)
}

func TestScript_SetMaxConstObjects(t *testing.T) {
	// one constant '5'
	s := gs.NewScript(alloc, []byte(`a := 5`))
	s.SetMaxConstObjects(1) // limit = 1
	_, err := s.Compile()
	require.NoError(t, err)
	s.SetMaxConstObjects(0) // limit = 0
	_, err = s.Compile()
	require.Error(t, err)
	require.Equal(t, "exceeding constant objects limit: 1", err.Error())

	// two constants '5' and '1'
	s = gs.NewScript(alloc, []byte(`a := 5 + 1`))
	s.SetMaxConstObjects(2) // limit = 2
	_, err = s.Compile()
	require.NoError(t, err)
	s.SetMaxConstObjects(1) // limit = 1
	_, err = s.Compile()
	require.Error(t, err)
	require.Equal(t, "exceeding constant objects limit: 2", err.Error())

	// duplicates will be removed
	s = gs.NewScript(alloc, []byte(`a := 5 + 5`))
	s.SetMaxConstObjects(1) // limit = 1
	_, err = s.Compile()
	require.NoError(t, err)
	s.SetMaxConstObjects(0) // limit = 0
	_, err = s.Compile()
	require.Error(t, err)
	require.Equal(t, "exceeding constant objects limit: 1", err.Error())

	// no limit set
	s = gs.NewScript(alloc, []byte(`a := 1 + 2 + 3 + 4 + 5`))
	_, err = s.Compile()
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
		"double": alloc.NewBuiltinFunctionValue(
			"unknown",
			func(v core.VM, args []core.Value) (ret core.Value, err error) {
				arg0, _ := args[0].AsInt()
				ret = core.IntValue(arg0 * 2)
				return
			},
			1,
			false,
		),
	}

	scr := gs.NewScript(alloc, code)
	_ = add(scr, "a", 0)
	_ = add(scr, "b", 0)
	_ = add(scr, "c", 0)
	mods := vm.NewModuleMap()
	mods.AddBuiltinModule("mod1", mod1)
	scr.SetImports(mods)
	compiled, err := scr.Compile()
	require.NoError(t, err)

	executeFn := func(compiled *gs.Compiled, a, b, c int) (d, e int) {
		av, _ := require.FromInterface(alloc, a)
		bv, _ := require.FromInterface(alloc, b)
		cv, _ := require.FromInterface(alloc, c)
		_ = compiled.Set("a", av)
		_ = compiled.Set("b", bv)
		_ = compiled.Set("c", cv)
		err := compiled.Run()
		require.NoError(t, err)
		d = int(compiled.Get("d").Int())
		e = int(compiled.Get("e").Int())
		return
	}

	concurrency := 500
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(compiled *gs.Compiled) {
			time.Sleep(time.Duration(rand.Int63n(50)) * time.Millisecond)
			defer wg.Done()

			a := rand.Intn(10)
			b := rand.Intn(10)
			c := rand.Intn(10)

			d, e := executeFn(compiled, a, b, c)
			expectedD, expectedE := solve(a, b, c)

			require.Equal(t, expectedD, d, "input: %d, %d, %d", a, b, c)
			require.Equal(t, expectedE, e, "input: %d, %d, %d", a, b, c)
		}(compiled.Clone())
	}
	wg.Wait()
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

func compiledGetCounter(t *testing.T, c *gs.Compiled, name string, expected *Counter) {
	v := c.Get(name)
	require.NotNil(t, v)

	val := v.Value()
	actual := toCounter(val)
	require.NotNil(t, actual)
	require.Equal(t, expected.value, actual.value)
}

func TestScriptSourceModule(t *testing.T) {
	// script1 imports "mod1"
	scr := gs.NewScript(alloc, []byte(`out := import("mod")`))
	mods := vm.NewModuleMap()
	mods.AddSourceModule("mod", []byte(`export 5`))
	scr.SetImports(mods)
	c, err := scr.Run()
	require.NoError(t, err)
	v := c.Get("out").Value()
	require.Equal(t, int64(5), v.Interface())

	// executing module function
	scr = gs.NewScript(alloc, []byte(`fn := import("mod"); out := fn()`))
	mods = vm.NewModuleMap()
	mods.AddSourceModule("mod",
		[]byte(`a := 3; export func() { return a + 5 }`))
	scr.SetImports(mods)
	c, err = scr.Run()
	require.NoError(t, err)
	v = c.Get("out").Value()
	require.Equal(t, int64(8), v.Interface())

	scr = gs.NewScript(alloc, []byte(`out := import("mod")`))
	mods = vm.NewModuleMap()
	mods.AddSourceModule("mod", []byte(`text := import("text"); export text.title("foo")`))
	mods.AddBuiltinModule("text", map[string]core.Value{
		"title": alloc.NewBuiltinFunctionValue(
			"title",
			func(v core.VM, args []core.Value) (core.Value, error) {
				s, _ := args[0].AsString()
				return alloc.NewStringValue(strings.Title(s)), nil
			},
			1,
			false,
		),
	})
	scr.SetImports(mods)
	c, err = scr.Run()
	require.NoError(t, err)
	v = c.Get("out").Value()
	require.Equal(t, "Foo", v.Interface())
	scr.SetImports(nil)
	_, err = scr.Run()
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
	s := gs.NewScript(alloc, []byte(input))
	c, err := s.Compile()
	if err != nil {
		panic(err)
	}

	for i := 0; i < n; i++ {
		if err := c.Run(); err != nil {
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

func TestCompiled_IsDefined(t *testing.T) {
	c := compile(t, `a := 5`, nil)
	compiledIsDefined(t, c, "a", false) // a is not defined before Run()
	compiledRun(t, c)
	compiledIsDefined(t, c, "a", true)
	compiledIsDefined(t, c, "b", false)
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

	// try to replace undefined variable
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
	// machine completes normally
	c := compile(t, `a := 5`, nil)
	err := c.RunContext(context.Background())
	require.NoError(t, err)
	compiledGet(t, c, "a", int64(5))

	// timeout
	c = compile(t, `for true {}`, nil)
	ctx, cancel := context.WithTimeout(context.Background(),
		1*time.Millisecond)
	defer cancel()
	err = c.RunContext(ctx)
	require.Equal(t, context.DeadlineExceeded, err)
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

	s := gs.NewScript(alloc, []byte(m))
	mods := vm.NewModuleMap()
	mods.AddSourceModule("expression", []byte(src))
	s.SetImports(mods)

	err := add(s, "ctx", map[string]any{
		"ctx": 12,
	})
	require.NoError(t, err)

	_, err = s.Run()
	require.True(t, strings.Contains(err.Error(), "expression:4:6"))
}

func compile(t *testing.T, input string, vars M) *gs.Compiled {
	s := gs.NewScript(alloc, []byte(input))
	for vn, vv := range vars {
		err := add(s, vn, vv)
		require.NoError(t, err)
	}

	c, err := s.Compile()
	require.NoError(t, err)
	require.NotNil(t, c)
	return c
}

func compileError(t *testing.T, input string, vars M) {
	s := gs.NewScript(alloc, []byte(input))
	for vn, vv := range vars {
		err := add(s, vn, vv)
		require.NoError(t, err)
	}
	_, err := s.Compile()
	require.Error(t, err)
}

func compiledRun(t *testing.T, c *gs.Compiled) {
	err := c.Run()
	require.NoError(t, err)
}

func compiledGet(t *testing.T, c *gs.Compiled, name string, expected any) {
	e, err := require.FromInterface(alloc, expected)
	require.NoError(t, err)
	v := c.Get(name)
	require.NotNil(t, v)
	require.Equal(t, e, v.Value())
}

func compiledGetAll(t *testing.T, c *gs.Compiled, expected M) {
	vars := c.GetAll()
	require.Equal(t, len(expected), len(vars))

	for k, ev := range expected {
		v, err := require.FromInterface(alloc, ev)
		require.NoError(t, err)
		var found bool
		for _, e := range vars {
			if e.Name() == k {
				require.Equal(t, v, e.Value())
				found = true
			}
		}
		require.True(t, found, "variable '%s' not found", k)
	}
}

func compiledIsDefined(t *testing.T, c *gs.Compiled, name string, expected bool) {
	require.Equal(t, expected, c.IsDefined(name))
}
func TestCompiled_Clone(t *testing.T) {
	script := gs.NewScript(alloc, []byte(`
count += 1
data["b"] = 2
`))

	err := add(script, "data", map[string]any{"a": 1})
	require.NoError(t, err)

	err = add(script, "count", 1000)
	require.NoError(t, err)

	compiled, err := script.Compile()
	require.NoError(t, err)

	clone := compiled.Clone()
	err = clone.RunContext(context.Background())
	require.NoError(t, err)

	require.Equal(t, int64(1000), compiled.Get("count").Int())
	require.Equal(t, 1, len(compiled.Get("data").Map()))

	require.Equal(t, int64(1001), clone.Get("count").Int())
	require.Equal(t, 2, len(clone.Get("data").Map()))
}
