package stdlib

import (
	"fmt"
	"time"

	"github.com/jokruger/fin128/civil"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

func init() {
	m := core.NewModule(value.ModuleClock, "times", false)
	m.Const(members.Nanosecond, core.IntValue(int64(time.Nanosecond)))
	m.Const(members.Microsecond, core.IntValue(int64(time.Microsecond)))
	m.Const(members.Millisecond, core.IntValue(int64(time.Millisecond)))
	m.Const(members.Second, core.IntValue(int64(time.Second)))
	m.Const(members.Minute, core.IntValue(int64(time.Minute)))
	m.Const(members.Hour, core.IntValue(int64(time.Hour)))
	m.Const(members.January, core.IntValue(int64(time.January)))
	m.Const(members.February, core.IntValue(int64(time.February)))
	m.Const(members.March, core.IntValue(int64(time.March)))
	m.Const(members.April, core.IntValue(int64(time.April)))
	m.Const(members.May, core.IntValue(int64(time.May)))
	m.Const(members.June, core.IntValue(int64(time.June)))
	m.Const(members.July, core.IntValue(int64(time.July)))
	m.Const(members.August, core.IntValue(int64(time.August)))
	m.Const(members.September, core.IntValue(int64(time.September)))
	m.Const(members.October, core.IntValue(int64(time.October)))
	m.Const(members.November, core.IntValue(int64(time.November)))
	m.Const(members.December, core.IntValue(int64(time.December)))
	// 42..127 reserved
	m.Fn(members.Sleep, 0, timesSleep, 1, core.Fixed)                                 // sleep(int)
	m.PureFn(members.ParseDuration, 1, timesParseDuration, 1, core.Fixed)             // parse_duration(str) => int
	m.Fn(members.Since, 2, timesSince, 1, core.Fixed)                                 // since(time) => int
	m.Fn(members.Until, 3, timesUntil, 1, core.Fixed)                                 // until(time) => int
	m.PureFn(members.DurationHours, 4, timesDurationHours, 1, core.Fixed)             // duration_hours(int) => float
	m.PureFn(members.DurationMinutes, 5, timesDurationMinutes, 1, core.Fixed)         // duration_minutes(int) => float
	m.PureFn(members.DurationNanoseconds, 6, timesDurationNanoseconds, 1, core.Fixed) // duration_nanoseconds(int) => int
	m.PureFn(members.DurationSeconds, 7, timesDurationSeconds, 1, core.Fixed)         // duration_seconds(int) => float
	m.PureFn(members.DurationString, 8, timesDurationString, 1, core.Fixed)           // duration_string(int) => string
	m.Fn(members.Now, 10, timesNow, 0, core.Fixed)                                    // now() => time
	m.PureFn(members.Unix, 12, timesUnix, 2, core.Fixed)                              // unix(sec, nsec) => time
	m.PureFn(members.FromUnixMs, 16, timesFromUnixMs, 1, core.Fixed)                  // from_unix_ms(msec) => time
	m.PureFn(members.FromUnixMicro, 17, timesFromUnixMicro, 1, core.Fixed)            // from_unix_micro(usec) => time
	m.PureFn(members.FromUnixNano, 18, timesFromUnixNano, 1, core.Fixed)              // from_unix_nano(nsec) => time
	m.PureFn(members.IsLeapYear, 19, timesIsLeapYear, 1, core.Fixed)                  // is_leap_year(year) => bool
	m.PureFn(members.DaysInYear, 20, timesDaysInYear, 1, core.Fixed)                  // days_in_year(year) => int
	m.PureFn(members.DaysInMonth, 21, timesDaysInMonth, 2, core.Fixed)                // days_in_month(year, month) => int
	if err := m.Register(); err != nil {
		panic(err)
	}
}

func timesSleep(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.sleep", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.sleep", "first", "int(compatible)", args[0].TypeName())
	}

	time.Sleep(time.Duration(i1))
	return core.Undefined, nil
}

func timesParseDuration(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.parse_duration", "1", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.parse_duration", "first", "string(compatible)", args[0].TypeName())
	}

	dur, err := time.ParseDuration(s1)
	if err != nil {
		return raiseGo(errs.KindConversion, "times.parse_duration", err)
	}

	return core.IntValue(int64(dur)), nil
}

func timesSince(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.since", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.since", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(time.Since(t1))), nil
}

func timesUntil(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.until", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.until", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(time.Until(t1))), nil
}

func timesDurationHours(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.duration_hours", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.duration_hours", "first", "int(compatible)", args[0].TypeName())
	}

	return core.FloatValue(time.Duration(i1).Hours()), nil
}

func timesDurationMinutes(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.duration_minutes", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.duration_minutes", "first", "int(compatible)", args[0].TypeName())
	}

	return core.FloatValue(time.Duration(i1).Minutes()), nil
}

func timesDurationNanoseconds(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.duration_nanoseconds", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.duration_nanoseconds", "first", "int(compatible)", args[0].TypeName())
	}

	return core.IntValue(time.Duration(i1).Nanoseconds()), nil
}

func timesDurationSeconds(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.duration_seconds", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.duration_seconds", "first", "int(compatible)", args[0].TypeName())
	}

	return core.FloatValue(time.Duration(i1).Seconds()), nil
}

func timesDurationString(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.duration_string", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.duration_string", "first", "int(compatible)", args[0].TypeName())
	}

	return core.NewStringValue(time.Duration(i1).String()), nil
}

// timesNow answers the current instant in UTC. The host's own zone is host state a script never sees by name
// ("Local" is not a zone); a script that wants a local wall clock names the zone: times.now().in_zone(z).
func timesNow(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.now", "0", len(args))
	}
	return core.NewTimeValue(time.Now().UTC()), nil
}

func timesUnix(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.unix", "2", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.unix", "first", "int(compatible)", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.unix", "second", "int(compatible)", args[1].TypeName())
	}

	return core.NewTimeValue(time.Unix(i1, i2).UTC()), nil
}

// The from_unix_* family: an int in the encoding each name states (the seconds encoding is the conversion
// (n).time()). Like times.unix(sec, nsec), these normalize to UTC, so the same script on two differently
// configured machines yields the same wall-clock components. Each one is the exact inverse of the time member
// accessor with the matching suffix (t.unix_ms(), t.unix_micro(), t.unix_nano()).
func timesFromUnixMs(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.from_unix_ms", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.from_unix_ms", "first", "int(compatible)", args[0].TypeName())
	}

	return core.NewTimeValue(time.UnixMilli(i1).UTC()), nil
}

func timesFromUnixMicro(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.from_unix_micro", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.from_unix_micro", "first", "int(compatible)", args[0].TypeName())
	}

	return core.NewTimeValue(time.UnixMicro(i1).UTC()), nil
}

func timesFromUnixNano(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.from_unix_nano", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.from_unix_nano", "first", "int(compatible)", args[0].TypeName())
	}

	return core.NewTimeValue(time.Unix(0, i1).UTC()), nil
}

// The calendar facts about plain numbers. A date or time answers the same questions as members
// (d.is_leap_year(), t.days_in_month(), ...); these take ints only, so an int here always means a year
// (and a month), never a timestamp.
func timesYearArg(name, pos string, a core.Value) (int, error) {
	if a.Type != value.Int {
		return 0, errs.NewInvalidArgumentTypeError(name, pos, "int", a.TypeName())
	}
	y := int64(a.Data)
	if y < 1 || y > 9999 {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) year %d out of range 1..9999", name, y))
	}
	return int(y), nil
}

func timesIsLeapYear(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.is_leap_year", "1", len(args))
	}
	y, err := timesYearArg("times.is_leap_year", "first", args[0])
	if err != nil {
		return core.Undefined, err
	}
	return core.BoolValue(civil.IsLeapYear(y)), nil
}

func timesDaysInYear(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.days_in_year", "1", len(args))
	}
	y, err := timesYearArg("times.days_in_year", "first", args[0])
	if err != nil {
		return core.Undefined, err
	}
	return core.IntValue(int64(civil.DaysInYear(y))), nil
}

func timesDaysInMonth(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.days_in_month", "2", len(args))
	}
	y, err := timesYearArg("times.days_in_month", "first", args[0])
	if err != nil {
		return core.Undefined, err
	}
	if args[1].Type != value.Int {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.days_in_month", "second", "int", args[1].TypeName())
	}
	m := int64(args[1].Data)
	if m < 1 || m > 12 {
		return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("(times.days_in_month) month %d out of range 1..12", m))
	}
	return core.IntValue(int64(civil.DaysInMonth(y, civil.Month(m)))), nil
}
