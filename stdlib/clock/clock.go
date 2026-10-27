package clock

import (
	"time"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

// The world's time: the current instant, the distance to one, and waiting.
//
// It is the only place a script reads the wall clock, so it needs a grant.
//
// ```go
// clock := import("clock")
// clock.now().in_zone("Europe/Kyiv").format("#datetime")
// start := clock.now()
// clock.sleep(10.milliseconds())
// clock.since(start).in_seconds()     // ≈ 0.01
// ```
//
// Durations are **int members**, not module functions: `30.seconds()`, `1.5.hours()`, `d.in_minutes()`,
// `d.duration_string()`, `"1h30m".nanoseconds()` (see [reference/types/int.md § Durations](../types/int.md#durations)).
// Construction, parsing, zones, calendar arithmetic and calendar facts are on the `time` and `date` types —
// `time(…)`, `t.in_zone(z)`, `t.add_months(n, rule)`, `t.is_leap_year()`.
//
// **Failures.** An argument that does not convert to `time` (`since`, `until`) or to an `int` duration (`sleep`)
// raises kind `invalid_argument_type`; a wrong count, `wrong_num_arguments`.
//
//kavun:module clock impure

// Answer the current instant, in UTC.
//
//kavun:fn now slot=0 impure () -> time
func clockNow(_ core.VM, args []core.Value) (core.Value, error) {
	return core.NewTimeValue(time.Now().UTC()), nil
}

// Answer the duration from `t` to now, in nanoseconds.
//
//kavun:fn since slot=1 impure (t) -> int
func clockSince(_ core.VM, args []core.Value) (core.Value, error) {
	t, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("clock.since", "first", "time(compatible)", args[0].TypeName())
	}
	return core.IntValue(int64(time.Since(t))), nil
}

// Answer the duration from now to `t`, in nanoseconds.
//
//kavun:fn until slot=2 impure (t) -> int
func clockUntil(_ core.VM, args []core.Value) (core.Value, error) {
	t, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("clock.until", "first", "time(compatible)", args[0].TypeName())
	}
	return core.IntValue(int64(time.Until(t))), nil
}

// Wait `d` nanoseconds (`clock.sleep(2.seconds())`).
//
//kavun:fn sleep slot=3 impure (d) -> undefined
func clockSleep(_ core.VM, args []core.Value) (core.Value, error) {
	d, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("clock.sleep", "first", "int(compatible)", args[0].TypeName())
	}
	time.Sleep(time.Duration(d))
	return core.Undefined, nil
}
