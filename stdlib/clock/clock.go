package clock

import (
	"time"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// The clock module is the world's time: the current instant, the distance to one, and waiting. It is the only
// place a script reads the wall clock, so it needs a grant. Durations are not here — they are int members
// (30.seconds(), d.in_hours(), "1h30m".nanoseconds()) — and neither are calendar facts (date/time members).
func init() {
	m := core.NewModule(value.ModuleClock, "clock", false)
	m.Fn(members.Now, 0, clockNow, 0, core.Fixed)
	m.Fn(members.Since, 1, clockSince, 1, core.Fixed)
	m.Fn(members.Until, 2, clockUntil, 1, core.Fixed)
	m.Fn(members.Sleep, 3, clockSleep, 1, core.Fixed)
	// 4..127 reserved
	if err := m.Register(); err != nil {
		panic(err)
	}
}

// now(): the current instant, in UTC.
func clockNow(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("clock.now", "0", len(args))
	}
	return core.NewTimeValue(time.Now().UTC()), nil
}

// since(t): the duration from t to now, in nanoseconds.
func clockSince(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("clock.since", "1", len(args))
	}
	t, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("clock.since", "first", "time(compatible)", args[0].TypeName())
	}
	return core.IntValue(int64(time.Since(t))), nil
}

// until(t): the duration from now to t, in nanoseconds.
func clockUntil(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("clock.until", "1", len(args))
	}
	t, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("clock.until", "first", "time(compatible)", args[0].TypeName())
	}
	return core.IntValue(int64(time.Until(t))), nil
}

// sleep(d): wait d nanoseconds (2.seconds(), …); undefined.
func clockSleep(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("clock.sleep", "1", len(args))
	}
	d, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("clock.sleep", "first", "int(compatible)", args[0].TypeName())
	}
	time.Sleep(time.Duration(d))
	return core.Undefined, nil
}
