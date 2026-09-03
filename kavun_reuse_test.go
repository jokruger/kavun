package kavun_test

import (
	"context"
	"testing"
	"time"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/vm"
)

// These tests exercise the "reuse the VM across runs" pattern that production embedders use for performance.
// The VM does not zero its stack/buffers between runs (only resets indexes), so any feature that reads a slot before
// writing it could be tainted by the previous execution. The named-result slot, the in-flight-error slot of the frame,
// and the deferred-call list are the obvious risk sites.

// runReuse runs the same compiled Script `times` times on a single VM and returns the captured `out` Variables.
// It does NOT call vm.Clear() between runs, so any stale state on the stack would survive into the next run.
func runReuse(t *testing.T, src string, times int) []any {
	t.Helper()

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(src), "out")
	c, err := s.Compile()
	require.NoError(t, err)

	results := make([]any, times)
	for i := 0; i < times; i++ {
		require.NoError(t, c.Run(machine))
		v, err := c.Get("out")
		require.NoError(t, err)
		results[i] = v.Interface()
	}

	return results
}

// runReuseSwitching runs script A, then script B, then script A again, etc. on the same VM. Returns the output
// Variables in the order the scripts ran.
func runReuseSwitching(t *testing.T, scripts []string, rounds int) []any {
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
			require.NoError(t, c.Run(machine))
			v, err := c.Get("out")
			require.NoError(t, err)
			out = append(out, v.Interface())
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

	results := runReuse(t, src, 5)
	for i, r := range results {
		require.Equal(t, true, r, "run %d", i)
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

	out := runReuseSwitching(t, []string{scriptA, scriptB}, 3)
	for i := 0; i < len(out); i += 2 {
		require.Equal(t, "from_A", out[i], "round %d script A", i/2)
		require.Equal(t, true, out[i+1], "round %d script B", i/2)
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

	results := runReuse(t, src, 5)
	for i, r := range results {
		require.Equal(t, true, r, "run %d", i)
	}
}

// Reuse a VM running a script with defers. Defer registrations live on the frame; when the frame is reused on the next
// run the defer slice must start empty, otherwise the previous run's deferred calls would fire again.
func TestVMReuse_Defer_NoLeakAcrossRuns(t *testing.T) {
	src := `
		log := []
		f := func() {
			defer func() { log = log.append("ran") }()
		}
		f()
		out = len(log)
	`

	results := runReuse(t, src, 4)
	for i, r := range results {
		require.Equal(t, int64(1), r, "run %d: defer should fire exactly once per run", i)
	}
}

// Multiple defers on the same VM across runs.
func TestVMReuse_Defer_MultipleAcrossRuns(t *testing.T) {
	src := `
		log := []
		f := func() {
			defer func() { log = log.append("a") }()
			defer func() { log = log.append("b") }()
			defer func() { log = log.append("c") }()
		}
		f()
		out = log
	`

	results := runReuse(t, src, 3)
	for i, r := range results {
		res := ""
		for _, v := range r.([]any) {
			res += v.(string)
		}
		require.Equal(t, "cba", res, "run %d", i)
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

	results := runReuse(t, src, 4)
	for i, r := range results {
		require.Equal(t, "no_error", r, "run %d", i)
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

	out := runReuseSwitching(t, []string{scriptRaises, scriptClean}, 4)
	for i := 0; i < len(out); i += 2 {
		require.Equal(t, "caught", out[i], "round %d raises", i/2)
		require.Equal(t, "ok", out[i+1], "round %d clean", i/2)
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

	results := runReuse(t, src, 50)
	for i, r := range results {
		arr := r.([]any)
		require.Equal(t, 2, len(arr), "run %d", i)
		require.Equal(t, int64(5), arr[0].(int64), "run %d ok", i)
		require.Equal(t, int64(-1), arr[1].(int64), "run %d bad", i)
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

	results := runReuse(t, src, 5)
	for i, r := range results {
		require.Equal(t, "done", r, "run %d", i)
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

	out := runReuseSwitching(t, []string{scriptDeferRaise, scriptNamedOnly, scriptPlain}, 5)
	for i := 0; i < len(out); i += 3 {
		require.Equal(t, "rescued", out[i], "round %d deferRaise", i/3)
		require.Equal(t, true, out[i+1], "round %d namedOnly", i/3)
		require.Equal(t, int64(6), out[i+2], "round %d plain", i/3)
	}
}

// TestVMReuse_AfterEveryOutcome pins the host-facing lifecycle contract: Compiled.Run and Compiled.RunContext both
// call VM.Reset first, so the SAME VM is usable again after every way a run can end — success, a recoverable error,
// a fatal error, a stack overflow, a contained host panic, and a cancelled context.
func TestVMReuse_AfterEveryOutcome(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	runSrc := func(t *testing.T, src string) error {
		t.Helper()
		s := kavun.NewScript([]byte(src), "out")
		c, err := s.Compile()
		require.NoError(t, err)
		return c.Run(machine)
	}
	expectOK := func(t *testing.T) {
		t.Helper()
		s := kavun.NewScript([]byte(`out = 1 + 1`), "out")
		c, err := s.Compile()
		require.NoError(t, err)
		require.NoError(t, c.Run(machine))
		v, err := c.Get("out")
		require.NoError(t, err)
		require.Equal(t, core.IntValue(2), v)
	}

	t.Run("AfterRecoverableError", func(t *testing.T) {
		require.Error(t, runSrc(t, `x := 1 / 0`))
		expectOK(t)
	})

	t.Run("AfterFatalError", func(t *testing.T) {
		require.Error(t, runSrc(t, `raise("stop", true)`))
		expectOK(t)
	})

	t.Run("AfterStackOverflow", func(t *testing.T) {
		require.Error(t, runSrc(t, `f := func() { return f() + 1 }; f()`))
		expectOK(t)
	})

	t.Run("AfterCancellation", func(t *testing.T) {
		s := kavun.NewScript([]byte(`ran = false; defer func(){ ran = true }(); for { }`), "ran")
		c, err := s.Compile()
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		err = c.RunContext(ctx, machine)
		require.Equal(t, context.DeadlineExceeded, err)

		// Cancellation aborts the VM; defers — root ones included — do not run.
		v, gerr := c.Get("ran")
		require.NoError(t, gerr)
		require.Equal(t, core.False, v)

		expectOK(t)
	})
}

// TestGlobalsReadableAfterError pins that a script's assignments up to the point of failure survive the error, which
// is what makes the root-defer outcome idiom work: the handler assigns a global and the host reads it back.
func TestGlobalsReadableAfterError(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte(`
		stage = "inputs"
		defer func() {
			e := recover()
			if e != undefined { outcome = {ok: false, stage: stage, kind: e.kind()} }
		}()
		stage = "scoring"
		x := 1 / 0
	`), "stage", "outcome")
	c, err := s.Compile()
	require.NoError(t, err)

	// A root recover() ends the script normally: the host is told the run succeeded and reads the outcome.
	require.NoError(t, c.Run(machine))

	stage, err := c.Get("stage")
	require.NoError(t, err)
	require.Equal(t, core.NewStringValue("scoring"), stage)

	outcome, err := c.Get("outcome")
	require.NoError(t, err)
	kind, err := outcome.Access(core.NewStringValue("kind"), bc.AccessIndex)
	require.NoError(t, err)
	require.Equal(t, core.NewStringValue("division_by_zero"), kind)
}

// Without a handler the same script fails, and the globals it did assign are still readable.
func TestGlobalsReadableAfterUnhandledError(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	s := kavun.NewScript([]byte("stage = \"scoring\"\nx := 1 / 0\n"), "stage")
	c, err := s.Compile()
	require.NoError(t, err)
	require.Error(t, c.Run(machine))

	stage, err := c.Get("stage")
	require.NoError(t, err)
	require.Equal(t, core.NewStringValue("scoring"), stage)
}
