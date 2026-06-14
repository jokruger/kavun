package kavun_test

import (
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/vm"
)

// These tests exercise the "reuse the VM across runs" pattern that production embedders use for performance.
// The VM does not zero its stack/buffers between runs (only resets indexes), so any feature that reads a slot before
// writing it could be tainted by the previous execution. The named-result slot, the in-flight-error slot of the frame,
// and the deferred-call list are the obvious risk sites.

// runReuse runs the same compiled Script `times` times on a single VM and returns the captured `out` Variables.
// It does NOT call vm.Clear() between runs, so any stale state on the stack would survive into the next run.
func runReuse(t *testing.T, rta *core.Arena, src string, times int) []any {
	t.Helper()

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(src), "out")
	c, err := s.Compile()
	require.NoError(t, err)

	results := make([]any, times)
	for i := 0; i < times; i++ {
		require.NoError(t, c.Run(rta, machine))
		results[i] = c.Get("out").Interface(rta)
	}

	return results
}

// runReuseSwitching runs script A, then script B, then script A again, etc. on the same VM. Returns the output
// Variables in the order the scripts ran.
func runReuseSwitching(t *testing.T, rta *core.Arena, scripts []string, rounds int) []any {
	t.Helper()

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	compiled := make([]*kavun.Compiled, len(scripts))
	for i, src := range scripts {
		s := kavun.NewScript([]byte(src), "out")
		c, err := s.Compile()
		require.NoError(t, err)
		compiled[i] = c
	}

	out := make([]any, 0, rounds*len(scripts))
	for range rounds {
		for _, c := range compiled {
			require.NoError(t, c.Run(rta, machine))
			out = append(out, c.Get("out").Interface(rta))
		}
	}

	return out
}

// Same compiled script run repeatedly. A function with a named result that is intentionally NOT assigned must yield
// Undefined every time, not whatever the previous run left on the stack.
func TestVMReuse_NamedResult_DefaultUndefinedAcrossRuns(t *testing.T) {
	src := `
		seed := func(x) s { s = x }
		_ = seed(12345)
		// probe declares a named result but never assigns it.
		// On every run, its named-result slot must be Undefined,
		// not 12345 from a previous slot occupant.
		probe := func() r {
			// no assignment to r
		}
		out = is_undefined(probe())
	`

	rta := core.NewArena(nil)
	results := runReuse(t, rta, src, 5)
	for i, r := range results {
		require.Equal(t, rta, true, r, "run %d", i)
	}
}

// Two different scripts on the same VM. Script A leaves a populated named-result slot; script B must not see Script
// A's value when its own named result is read before assignment.
func TestVMReuse_NamedResult_NoCrossScriptLeak(t *testing.T) {
	scriptA := `
		f := func() r { r = "from_A" }
		out = f()
	`
	scriptB := `
		g := func() r {
			before := r       // read named result before any assignment
			r = "from_B"
			out = is_undefined(before)
		}
		g()
	`

	rta := core.NewArena(nil)
	out := runReuseSwitching(t, rta, []string{scriptA, scriptB}, 3)
	for i := 0; i < len(out); i += 2 {
		require.Equal(t, rta, "from_A", out[i], "round %d script A", i/2)
		require.Equal(t, rta, true, out[i+1], "round %d script B", i/2)
	}
}

// Repeatedly call a function whose named result is conditionally assigned. A previous call must not bleed into the next
// call's "untaken branch" path.
func TestVMReuse_NamedResult_ConditionalAcrossRuns(t *testing.T) {
	src := `
		maybe := func(yes) r {
			if yes { r = "set" }
		}
		_ = maybe(true)
		out = is_undefined(maybe(false))
	`

	rta := core.NewArena(nil)
	results := runReuse(t, rta, src, 5)
	for i, r := range results {
		require.Equal(t, rta, true, r, "run %d", i)
	}
}

// Reuse a VM running a script with defers. Defer registrations live on the frame; when the frame is reused on the next
// run the defer slice must start empty, otherwise the previous run's deferred calls would fire again.
func TestVMReuse_Defer_NoLeakAcrossRuns(t *testing.T) {
	src := `
		log := []
		f := func() {
			defer func() { log = append(log, "ran") }()
		}
		f()
		out = len(log)
	`

	rta := core.NewArena(nil)
	results := runReuse(t, rta, src, 4)
	for i, r := range results {
		require.Equal(t, rta, int64(1), r, "run %d: defer should fire exactly once per run", i)
	}
}

// Multiple defers on the same VM across runs.
func TestVMReuse_Defer_MultipleAcrossRuns(t *testing.T) {
	src := `
		log := []
		f := func() {
			defer func() { log = append(log, "a") }()
			defer func() { log = append(log, "b") }()
			defer func() { log = append(log, "c") }()
		}
		f()
		out = log
	`

	rta := core.NewArena(nil)
	results := runReuse(t, rta, src, 3)
	for i, r := range results {
		res := ""
		for _, v := range r.([]any) {
			res += v.(string)
		}
		require.Equal(t, rta, "cba", res, "run %d", i)
	}
}

// Recover catches a raised error on the first call inside the script; on a subsequent call (still inside the same
// script run) the in-flight-error slot must be clean, and across whole VM runs it must remain clean too.
func TestVMReuse_Recover_NoStaleErrorAcrossRuns(t *testing.T) {
	src := `
		raised := func() res {
			defer func() {
				e := recover()
				if e != undefined { res = "caught" }
			}()
			raise(error("bang"))
		}
		clean := func() res {
			defer func() {
				e := recover()
				if e == undefined { res = "no_error" } else { res = "stale" }
			}()
			res = "ok"
		}
		_ = raised()
		out = clean()
	`

	rta := core.NewArena(nil)
	results := runReuse(t, rta, src, 4)
	for i, r := range results {
		require.Equal(t, rta, "no_error", r, "run %d", i)
	}
}

// Mixed scripts: one that always raises+recovers, one that never raises. Run them interleaved on the same VM. The clean
// script must never observe the previous script's in-flight error.
func TestVMReuse_Recover_NoCrossScriptLeak(t *testing.T) {
	scriptRaises := `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined { res = "caught" }
			}()
			raise(error("boom"))
		}
		out = f()
	`
	scriptClean := `
		check := func() res {
			defer func() {
				e := recover()
				if e == undefined { res = "ok" } else { res = "leaked" }
			}()
			res = "init"
		}
		out = check()
	`

	rta := core.NewArena(nil)
	out := runReuseSwitching(t, rta, []string{scriptRaises, scriptClean}, 4)
	for i := 0; i < len(out); i += 2 {
		require.Equal(t, rta, "caught", out[i], "round %d raises", i/2)
		require.Equal(t, rta, "ok", out[i+1], "round %d clean", i/2)
	}
}

// Stress: same script invoked many times with raise+recover paths alternating with success paths. Exercises the
// in-flight-error slot reset, the defer list reset, and the deferredFor link reset on every OpCall, repeatedly on the
// same VM.
func TestVMReuse_DeferRecover_StressRepeat(t *testing.T) {
	src := `
		safe_div := func(a, b) result {
			defer func() {
				if recover() != undefined { result = -1 }
			}()
			result = a / b
		}
		ok  := safe_div(10, 2)
		bad := safe_div(10, 0)
		out = [ok, bad]
	`

	rta := core.NewArena(nil)
	results := runReuse(t, rta, src, 50)
	for i, r := range results {
		arr := r.([]any)
		require.Equal(t, rta, 2, len(arr), "run %d", i)
		require.Equal(t, rta, int64(5), arr[0].(int64), "run %d ok", i)
		require.Equal(t, rta, int64(-1), arr[1].(int64), "run %d bad", i)
	}
}

// Tail-call optimization reuses the same frame for the recursive call. Verify the named-result slot stays correct
// across many tail-call re-entries AND across many whole VM runs.
func TestVMReuse_NamedResult_WithTailCallAcrossRuns(t *testing.T) {
	src := `
		loop := func(n) r {
			if n == 0 { r = "done"; return }
			return loop(n - 1)
		}
		out = loop(100)
	`

	rta := core.NewArena(nil)
	results := runReuse(t, rta, src, 5)
	for i, r := range results {
		require.Equal(t, rta, "done", r, "run %d", i)
	}
}

// Two scripts, one with defers + raise, the other with named result + no defers. Interleave. Each script must produce
// its own correct output regardless of execution history.
func TestVMReuse_Mixed_NamedDeferRecoverInterleaved(t *testing.T) {
	scriptDeferRaise := `
        f := func() res {
            defer func() {
                if recover() != undefined { res = "rescued" }
            }()
            res = "ok"
            raise(error("e"))
        }
        out = f()
    `
	scriptNamedOnly := `
        g := func() r { /* never assigns r */ }
        out = is_undefined(g())
    `
	scriptPlain := `
        out = 1 + 2 + 3
    `

	rta := core.NewArena(nil)
	out := runReuseSwitching(t, rta, []string{scriptDeferRaise, scriptNamedOnly, scriptPlain}, 5)
	for i := 0; i < len(out); i += 3 {
		require.Equal(t, rta, "rescued", out[i], "round %d deferRaise", i/3)
		require.Equal(t, rta, true, out[i+1], "round %d namedOnly", i/3)
		require.Equal(t, rta, int64(6), out[i+2], "round %d plain", i/3)
	}
}
