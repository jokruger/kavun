package kavun_test

import (
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/internal/require"
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
