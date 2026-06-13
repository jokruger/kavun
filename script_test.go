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
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
)

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
	require.Equal(t, rta, int64(5), r.Interface(rta))
}

func TestScript_SetGet(t *testing.T) {
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`a := b; c := test(b); d := test(5)`), "b", "test")
	c, err := s.Compile()
	require.NoError(t, err)

	require.NoError(t, c.Set("b", core.IntValue(5)))              // b = 5
	require.NoError(t, c.Set("b", rta.MustNewStringValue("foo"))) // b = "foo"  (re-define before compilation)
	require.NoError(t, err)

	require.NoError(t, c.Set("test", rta.MustNewBuiltinClosureValue("test", func(a *core.Arena, v core.VM, args []core.Value) (core.Value, error) {
		if len(args) > 0 {
			if args[0].Type == core.VT_INT {
				return core.IntValue(int64(args[0].Data) + 1), nil
			}
		}
		return core.IntValue(0), nil
	}, 1, false)))

	require.NoError(t, c.Run(rta, machine))

	r := c.Get("a")
	require.Equal(t, rta, "foo", r.Interface(rta))

	r = c.Get("b")
	require.Equal(t, rta, "foo", r.Interface(rta))

	r = c.Get("c")
	require.Equal(t, rta, int64(0), r.Interface(rta))

	r = c.Get("d")
	require.Equal(t, rta, int64(6), r.Interface(rta))
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
	require.Equal(t, rta, core.IntValue(1), c.Get("count"))
	require.Equal(t, rta, rta.MustNewArrayValue([]core.Value{core.IntValue(11)}, false), c.Get("arr"))
	require.Equal(t, rta, core.IntValue(12), c.Get("out"))

	// Run #2: uses updated globals from previous run.
	require.NoError(t, c.Run(rta, machine))
	require.Equal(t, rta, core.IntValue(2), c.Get("count"))
	require.Equal(t, rta, rta.MustNewArrayValue([]core.Value{core.IntValue(12)}, false), c.Get("arr"))
	require.Equal(t, rta, core.IntValue(14), c.Get("out"))

	// Update globals and verify recurrent runs use updated values.
	require.NoError(t, c.Set("count", core.IntValue(100)))
	require.NoError(t, c.Set("arr", rta.MustNewArrayValue([]core.Value{core.IntValue(1)}, false)))
	require.NoError(t, c.Set("step", core.IntValue(2)))

	require.NoError(t, c.Run(rta, machine))
	require.Equal(t, rta, core.IntValue(101), c.Get("count"))
	require.Equal(t, rta, rta.MustNewArrayValue([]core.Value{core.IntValue(3)}, false), c.Get("arr"))
	require.Equal(t, rta, core.IntValue(104), c.Get("out"))
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
	require.Equal(t, rta, 19.84, c.Get("a").Interface(rta))

	c, err = s.Compile()
	require.NoError(t, err)
	err = c.Run(rta, machine)
	require.NoError(t, err)
	require.Equal(t, rta, 19.84, c.Get("a").Interface(rta))

	s.SetAllowedModules("os")
	_, err = s.Compile()
	require.Error(t, err)

	s.SetAllowedModules("qqqq")
	_, err = s.Compile()
	require.Error(t, err)
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

	stdlib.InitModule("mod1", core.BI_MOD_USER_DEFINED, nil, nil, map[uint64]*core.BuiltinFunction{
		0: core.NewBuiltinFunction(
			"double",
			func(a *core.Arena, v core.VM, args []core.Value) (ret core.Value, err error) {
				arg0, _ := args[0].AsInt(a)
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

			require.Equal(t, rta, int64(expectedD), d, "input: %d, %d, %d", a, b, c)
			require.Equal(t, rta, int64(expectedE), e, "input: %d, %d, %d", a, b, c)
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
			require.Equal(t, rta, int64(expectedD), d, "input: %d, %d, %d", a, b, c)
			require.Equal(t, rta, int64(expectedE), e, "input: %d, %d, %d", a, b, c)
			lock.Unlock()
		}(c)
	}
	wg2.Wait()
}
