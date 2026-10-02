package stdlib

import (
	"fmt"
	"time"

	"github.com/jokruger/fin128/civil"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/module"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

func init() {
	InitModule("times", module.Times,
		map[string]core.Value{
			"nanosecond":  core.IntValue(int64(time.Nanosecond)),
			"microsecond": core.IntValue(int64(time.Microsecond)),
			"millisecond": core.IntValue(int64(time.Millisecond)),
			"second":      core.IntValue(int64(time.Second)),
			"minute":      core.IntValue(int64(time.Minute)),
			"hour":        core.IntValue(int64(time.Hour)),
			"january":     core.IntValue(int64(time.January)),
			"february":    core.IntValue(int64(time.February)),
			"march":       core.IntValue(int64(time.March)),
			"april":       core.IntValue(int64(time.April)),
			"may":         core.IntValue(int64(time.May)),
			"june":        core.IntValue(int64(time.June)),
			"july":        core.IntValue(int64(time.July)),
			"august":      core.IntValue(int64(time.August)),
			"september":   core.IntValue(int64(time.September)),
			"october":     core.IntValue(int64(time.October)),
			"november":    core.IntValue(int64(time.November)),
			"december":    core.IntValue(int64(time.December)),
		},
		// 42..127 reserved
		map[uint64]*core.BuiltinFunction{
			0:  core.NewBuiltinFunction("sleep", timesSleep, 1, false, false),                             // sleep(int)
			1:  core.NewBuiltinFunction("parse_duration", timesParseDuration, 1, false, true),             // parse_duration(str) => int
			2:  core.NewBuiltinFunction("since", timesSince, 1, false, false),                             // since(time) => int
			3:  core.NewBuiltinFunction("until", timesUntil, 1, false, false),                             // until(time) => int
			4:  core.NewBuiltinFunction("duration_hours", timesDurationHours, 1, false, true),             // duration_hours(int) => float
			5:  core.NewBuiltinFunction("duration_minutes", timesDurationMinutes, 1, false, true),         // duration_minutes(int) => float
			6:  core.NewBuiltinFunction("duration_nanoseconds", timesDurationNanoseconds, 1, false, true), // duration_nanoseconds(int) => int
			7:  core.NewBuiltinFunction("duration_seconds", timesDurationSeconds, 1, false, true),         // duration_seconds(int) => float
			8:  core.NewBuiltinFunction("duration_string", timesDurationString, 1, false, true),           // duration_string(int) => string
			10: core.NewBuiltinFunction("now", timesNow, 0, false, false),                                 // now() => time
			12: core.NewBuiltinFunction("unix", timesUnix, 2, false, true),                                // unix(sec, nsec) => time
			16: core.NewBuiltinFunction("from_unix_ms", timesFromUnixMs, 1, false, true),                  // from_unix_ms(msec) => time
			17: core.NewBuiltinFunction("from_unix_micro", timesFromUnixMicro, 1, false, true),            // from_unix_micro(usec) => time
			18: core.NewBuiltinFunction("from_unix_nano", timesFromUnixNano, 1, false, true),              // from_unix_nano(nsec) => time
			19: core.NewBuiltinFunction("is_leap_year", timesIsLeapYear, 1, false, true),                  // is_leap_year(year) => bool
			20: core.NewBuiltinFunction("days_in_year", timesDaysInYear, 1, false, true),                  // days_in_year(year) => int
			21: core.NewBuiltinFunction("days_in_month", timesDaysInMonth, 2, false, true),                // days_in_month(year, month) => int
		},
	)
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
