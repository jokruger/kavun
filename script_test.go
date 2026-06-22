package kavun_test

import (
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
)

func compileError(t *testing.T, input string, vars map[string]any) {
	s := kavun.NewScript([]byte(input))
	for n := range vars {
		s.AddGlobals(n)
	}
	_, err := s.Compile()
	require.Error(t, err)
}

func compile(t *testing.T, input string, vars map[string]any) *kavun.Compiled {
	s := kavun.NewScript([]byte(input))
	for n := range vars {
		s.AddGlobals(n)
	}

	c, err := s.Compile()
	require.NoError(t, err)
	for n, v := range vars {
		err := c.Set(n, kavun.MustValueOf(v))
		require.NoError(t, err)
	}

	return c
}

func compiledRun(t *testing.T, c *kavun.Compiled) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	err := c.Run(a, machine)
	require.NoError(t, err)
}

func compiledGet(t *testing.T, c *kavun.Compiled, name string, expected any) {
	e, err := kavun.ValueOf(expected)
	require.NoError(t, err)
	v := c.Get(name)
	require.NotNil(t, v)
	require.Equal(t, a, e, v)
}

func TestScript_Run(t *testing.T) {
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`a := b`), "b")
	c, err := s.Compile()
	require.NoError(t, err)

	err = c.Set("b", core.IntValue(5))
	require.NoError(t, err)
	err = c.Run(rta, machine)
	require.NoError(t, err)

	r := c.Get("a")
	require.Equal(t, int64(5), r.Interface(rta))
}

func TestScript_SetGet(t *testing.T) {
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`a := b; c := test(b); d := test(5)`), "b", "test")
	c, err := s.Compile()
	require.NoError(t, err)

	require.NoError(t, c.Set("b", core.IntValue(5)))           // b = 5
	require.NoError(t, c.Set("b", core.NewStringValue("foo"))) // b = "foo"  (re-define before compilation)
	require.NoError(t, err)

	require.NoError(t, c.Set("test", rta.MustNewBuiltinClosureValue("test", func(v core.VM, args []core.Value) (core.Value, error) {
		if len(args) > 0 {
			if args[0].Type == value.Int {
				return core.IntValue(int64(args[0].Data) + 1), nil
			}
		}
		return core.IntValue(0), nil
	}, 1, false)))

	require.NoError(t, c.Run(rta, machine))

	r := c.Get("a")
	require.Equal(t, "foo", r.Interface(rta))

	r = c.Get("b")
	require.Equal(t, "foo", r.Interface(rta))

	r = c.Get("c")
	require.Equal(t, int64(0), r.Interface(rta))

	r = c.Get("d")
	require.Equal(t, int64(6), r.Interface(rta))
}

func TestScript_RecurrentRun(t *testing.T) {
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`
count += 1
arr[0] += step
out = count + arr[0]
`), "count", "arr", "step", "out")

	c, err := s.Compile()
	require.NoError(t, err)

	require.NoError(t, c.Set("count", core.IntValue(0)))
	require.NoError(t, c.Set("arr", rta.MustNewArrayValue([]core.Value{core.IntValue(10)}, false)))
	require.NoError(t, c.Set("step", core.IntValue(1)))
	require.NoError(t, c.Set("out", core.Undefined))

	// Run #1: uses initial globals.
	require.NoError(t, c.Run(rta, machine))
	require.Equal(t, core.IntValue(1), c.Get("count"))
	require.Equal(t, rta.MustNewArrayValue([]core.Value{core.IntValue(11)}, false), c.Get("arr"))
	require.Equal(t, core.IntValue(12), c.Get("out"))

	// Run #2: uses updated globals from previous run.
	require.NoError(t, c.Run(rta, machine))
	require.Equal(t, core.IntValue(2), c.Get("count"))
	require.Equal(t, rta.MustNewArrayValue([]core.Value{core.IntValue(12)}, false), c.Get("arr"))
	require.Equal(t, core.IntValue(14), c.Get("out"))

	// Update globals and verify recurrent runs use updated values.
	require.NoError(t, c.Set("count", core.IntValue(100)))
	require.NoError(t, c.Set("arr", rta.MustNewArrayValue([]core.Value{core.IntValue(1)}, false)))
	require.NoError(t, c.Set("step", core.IntValue(2)))

	require.NoError(t, c.Run(rta, machine))
	require.Equal(t, core.IntValue(101), c.Get("count"))
	require.Equal(t, rta.MustNewArrayValue([]core.Value{core.IntValue(3)}, false), c.Get("arr"))
	require.Equal(t, core.IntValue(104), c.Get("out"))
}

func TestScript_SetAssignmentMode(t *testing.T) {
	s := kavun.NewScript([]byte(`a = 1`))
	_, err := s.Compile()
	require.NoError(t, err)

	s = kavun.NewScript([]byte(`a = 1`))
	s.SetAssignmentMode(compiler.AssignmentModeStrict)
	_, err = s.Compile()
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unresolved reference 'a'"))

	s = kavun.NewScript([]byte(`a += 1`))
	_, err = s.Compile()
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unresolved reference 'a'"))
}

func TestScript_BuiltinModules(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	rta := core.NewArena(nil)

	s := kavun.NewScript([]byte(`math := import("math"); a := math.abs(-19.84)`))
	s.SetAllowedModules("math")
	c, err := s.Compile()
	require.NoError(t, err)
	err = c.Run(rta, machine)
	require.NoError(t, err)
	require.Equal(t, 19.84, c.Get("a").Interface(rta))

	c, err = s.Compile()
	require.NoError(t, err)
	err = c.Run(rta, machine)
	require.NoError(t, err)
	require.Equal(t, 19.84, c.Get("a").Interface(rta))

	s.SetAllowedModules("os")
	_, err = s.Compile()
	require.Error(t, err)

	s.SetAllowedModules("qqqq")
	_, err = s.Compile()
	require.Error(t, err)
}

func TestCompiled_Get(t *testing.T) {
	rta := core.NewArena(nil)

	// simple script
	c := compile(t, `a := 5`, nil)
	compiledRun(t, c)
	compiledGet(t, c, "a", int64(5))

	// user-defined variables
	compileError(t, `a := b`, nil)            // compile error because "b" is not defined
	c = compile(t, `a := b`, MAP{"b": "foo"}) // now compile with b = "foo" defined
	compiledGet(t, c, "a", nil)               // a = undefined; because it's before Compiled.Run()
	compiledRun(t, c)                         // Compiled.Run()
	compiledGet(t, c, "a", "foo")             // a = "foo"
}

func Test_IsDefined(t *testing.T) {
	rta := core.NewArena(nil)
	c := compile(t, `a := 5`, nil)
	compiledRun(t, c)
	v := c.Get("a")
	require.Equal(t, value.Int, v.Type)
	require.Equal(t, int(5), int(v.Data))
	v = c.Get("b")
	require.Equal(t, value.Undefined, v.Type)
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

	s := kavun.NewScript([]byte(m), "ctx")
	s.AddCustomModule("expression", []byte(src))

	c, err := s.Compile()
	require.NoError(t, err)

	rta := core.NewArena(nil)
	require.NoError(t, c.Set("ctx", kavun.MustValueOf(rta, MAP{})))

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	err = c.Run(rta, machine)
	require.True(t, strings.Contains(err.Error(), "expression:4:6"))
}

// Verifies that reassigning a builtin in a script does not leak across independently compiled scripts that share the
// same VM, and that the builtin remains accessible by name in scripts that do not reassign it.
func TestScript_BuiltinReassign_VMReuse(t *testing.T) {
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	// Script A reassigns the builtin `len` to a constant.
	sA := kavun.NewScript([]byte(`len = 42; out = len`), "out")
	cA, err := sA.Compile()
	require.NoError(t, err)
	require.NoError(t, cA.Run(rta, machine))
	require.Equal(t, int64(42), cA.Get("out").Interface(rta))

	// Script B uses the builtin `len` on the same VM. It must see the original builtin, unaffected by Script A's
	// reassignment.
	sB := kavun.NewScript([]byte(`out = len("hello")`), "out")
	cB, err := sB.Compile()
	require.NoError(t, err)
	require.NoError(t, cB.Run(rta, machine))
	require.Equal(t, int64(5), cB.Get("out").Interface(rta))

	// Re-running Script A again on the same VM still reassigns to 42.
	require.NoError(t, cA.Run(rta, machine))
	require.Equal(t, int64(42), cA.Get("out").Interface(rta))

	// Re-running Script B again still uses the builtin.
	require.NoError(t, cB.Run(rta, machine))
	require.Equal(t, int64(5), cB.Get("out").Interface(rta))
}

// Verifies that re-running the same Compiled script multiple times restores the global slot that backs the reassigned
// builtin from compile-time globals on every run.
func TestScript_BuiltinReassign_RecurrentRun(t *testing.T) {
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`
before := len("abc")
len = 100
after := len
out = before + after
`), "out")

	c, err := s.Compile()
	require.NoError(t, err)

	for range 3 {
		require.NoError(t, c.Run(rta, machine))
		require.Equal(t, int64(103), c.Get("out").Interface(rta))
	}
}

// Verifies that builtins shadowed in function-local scopes do not leak to outer scopes.
func TestScript_BuiltinShadow_Scopes(t *testing.T) {
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`
inner := func() {
    len := 99
    return len
}
shadowed := inner()
builtin_after := len("ab")
out = shadowed + builtin_after
`), "out")

	c, err := s.Compile()
	require.NoError(t, err)
	require.NoError(t, c.Run(rta, machine))
	require.Equal(t, int64(101), c.Get("out").Interface(rta))
}

// Verifies that reassigning a builtin in the main script does not affect imported modules: the module compiles with its
// own fresh symbol table seeded from the original builtins.
func TestScript_BuiltinReassign_ModuleIsolation(t *testing.T) {
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	scr := kavun.NewScript([]byte(`
len = 999
fn := import("mod")
out = fn("abcd")
`), "out")
	scr.AddCustomModule("mod", []byte(`export func(s) { return len(s) }`))

	c, err := scr.Compile()
	require.NoError(t, err)
	require.NoError(t, c.Run(rta, machine))
	require.Equal(t, int64(4), c.Get("out").Interface(rta))
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

	stdlib.InitModule("mod1", kavun.UsedDefinedModule, nil, nil, map[uint64]*core.BuiltinFunction{
		0: core.NewBuiltinFunction(
			"double",
			func(v core.VM, args []core.Value) (ret core.Value, err error) {
				arg0, _ := args[0].AsInt()
				ret = core.IntValue(arg0 * 2)
				return
			},
			1,
			false,
		),
	})
	defer stdlib.RemoveModule("mod1")

	concurrency := 500

	// own vm and allocator
	var wg1 sync.WaitGroup
	wg1.Add(concurrency)
	for range concurrency {
		rta := core.NewArena(nil)
		scr := kavun.NewScript(code, "a", "b", "c")
		c, err := scr.Compile()
		require.NoError(t, err)
		c.Set("a", core.IntValue(0))
		c.Set("b", core.IntValue(0))
		c.Set("c", core.IntValue(0))

		go func(compiled *kavun.Compiled) {
			machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

			time.Sleep(time.Duration(rand.Int63n(50)) * time.Millisecond)
			defer wg1.Done()

			a := rand.Intn(10)
			b := rand.Intn(10)
			c := rand.Intn(10)

			_ = compiled.Set("a", kavun.MustValueOf(rta, a))
			_ = compiled.Set("b", kavun.MustValueOf(rta, b))
			_ = compiled.Set("c", kavun.MustValueOf(rta, c))
			err := compiled.Run(rta, machine)
			require.NoError(t, err)
			d, _ := compiled.Get("d").AsInt(rta)
			e, _ := compiled.Get("e").AsInt(rta)

			expectedD, expectedE := solve(a, b, c)

			require.Equal(t, int64(expectedD), d, "input: %d, %d, %d", a, b, c)
			require.Equal(t, int64(expectedE), e, "input: %d, %d, %d", a, b, c)
		}(c)
	}
	wg1.Wait()

	// shared vm and allocator
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	var lock sync.RWMutex
	var wg2 sync.WaitGroup
	wg2.Add(concurrency)
	for range concurrency {
		scr := kavun.NewScript(code, "a", "b", "c")
		c, err := scr.Compile()
		require.NoError(t, err)
		c.Set("a", core.IntValue(0))
		c.Set("b", core.IntValue(0))
		c.Set("c", core.IntValue(0))
		require.NoError(t, err)
		go func(compiled *kavun.Compiled) {
			time.Sleep(time.Duration(rand.Int63n(50)) * time.Millisecond)
			defer wg2.Done()

			a := rand.Intn(10)
			b := rand.Intn(10)
			c := rand.Intn(10)
			expectedD, expectedE := solve(a, b, c)

			lock.Lock()
			_ = compiled.Set("a", kavun.MustValueOf(rta, a))
			_ = compiled.Set("b", kavun.MustValueOf(rta, b))
			_ = compiled.Set("c", kavun.MustValueOf(rta, c))
			err := compiled.Run(rta, machine)
			require.NoError(t, err)
			d, _ := compiled.Get("d").AsInt(rta)
			e, _ := compiled.Get("e").AsInt(rta)
			require.Equal(t, int64(expectedD), d, "input: %d, %d, %d", a, b, c)
			require.Equal(t, int64(expectedE), e, "input: %d, %d, %d", a, b, c)
			lock.Unlock()
		}(c)
	}
	wg2.Wait()
}

func TestScript_CustomObjects(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	ma := NewMyArena()
	opts := core.DefaultArenaOptions()
	opts.Payload = ma
	rta := core.NewArena(opts)

	s := kavun.NewScript([]byte(`a := c1(); s := string(c1); c2 := c1; c2++`), "c1")
	c, err := s.Compile()
	require.NoError(t, err)
	require.NoError(t, c.Set("c1", ma.NewCounterValue(5)))
	require.NoError(t, c.Run(rta, machine))
	require.Equal(t, int64(5), c.Get("a").Interface(rta))
	require.Equal(t, "Counter(5)", c.Get("s").Interface(rta))
	r := c.Get("c2").Interface(rta).(*Counter)
	require.NotNil(t, r)
	require.Equal(t, int64(6), r.value)

	s = kavun.NewScript([]byte(`
arr := [1, 2, 3, 4]
for x in arr {
	c1 += x
}
out := c1()
`), "c1")
	c, err = s.Compile()
	require.NoError(t, err)
	require.NoError(t, c.Set("c1", ma.NewCounterValue(5)))
	require.NoError(t, c.Run(rta, machine))
	require.Equal(t, int64(15), c.Get("out").Interface(rta))
}

func TestScriptCustomModule(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	rta := core.NewArena(nil)

	// script1 imports "mod1"
	scr := kavun.NewScript([]byte(`out := import("mod1")`))
	scr.AddCustomModule("mod1", []byte(`export 5`))
	c, err := scr.Compile()
	require.NoError(t, err)
	err = c.Run(rta, machine)
	require.NoError(t, err)
	v := c.Get("out")
	require.Equal(t, int64(5), v.Interface(rta))

	// executing module function
	scr = kavun.NewScript([]byte(`fn := import("mod1"); out := fn()`))
	scr.AddCustomModule("mod1", []byte(`a := 3; export func() { return a + 5 }`))
	c, err = scr.Compile()
	require.NoError(t, err)
	err = c.Run(rta, machine)
	require.NoError(t, err)
	v = c.Get("out")
	require.Equal(t, int64(8), v.Interface(rta))

	stdlib.InitModule("text1", kavun.UsedDefinedModule, nil, nil, map[uint64]*core.BuiltinFunction{
		0: core.NewBuiltinFunction(
			"title",
			func(v core.VM, args []core.Value) (core.Value, error) {
				s, _ := args[0].AsString()
				return rta.NewStringValue(strings.Title(s))
			},
			1,
			false,
		),
	})
	defer stdlib.RemoveModule("text1")

	scr = kavun.NewScript([]byte(`out := import("mod1")`))
	scr.AddCustomModule("mod1", []byte(`text := import("text1"); export text.title("foo")`))
	c, err = scr.Compile()
	require.NoError(t, err)
	err = c.Run(rta, machine)
	require.NoError(t, err)
	v = c.Get("out")
	require.Equal(t, "Foo", v.Interface(rta))
}
