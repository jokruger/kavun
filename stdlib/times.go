package stdlib

import (
	"time"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/module"
	"github.com/jokruger/kavun/errs"
)

func init() {
	InitModule("times", module.Times,
		map[string]core.Value{
			"nanosecond":          core.IntValue(int64(time.Nanosecond)),
			"microsecond":         core.IntValue(int64(time.Microsecond)),
			"millisecond":         core.IntValue(int64(time.Millisecond)),
			"second":              core.IntValue(int64(time.Second)),
			"minute":              core.IntValue(int64(time.Minute)),
			"hour":                core.IntValue(int64(time.Hour)),
			"january":             core.IntValue(int64(time.January)),
			"february":            core.IntValue(int64(time.February)),
			"march":               core.IntValue(int64(time.March)),
			"april":               core.IntValue(int64(time.April)),
			"may":                 core.IntValue(int64(time.May)),
			"june":                core.IntValue(int64(time.June)),
			"july":                core.IntValue(int64(time.July)),
			"august":              core.IntValue(int64(time.August)),
			"september":           core.IntValue(int64(time.September)),
			"october":             core.IntValue(int64(time.October)),
			"november":            core.IntValue(int64(time.November)),
			"december":            core.IntValue(int64(time.December)),
			"format_ansic":        core.NewStringValue(time.ANSIC),
			"format_unix_date":    core.NewStringValue(time.UnixDate),
			"format_ruby_date":    core.NewStringValue(time.RubyDate),
			"format_rfc822":       core.NewStringValue(time.RFC822),
			"format_rfc822z":      core.NewStringValue(time.RFC822Z),
			"format_rfc850":       core.NewStringValue(time.RFC850),
			"format_rfc1123":      core.NewStringValue(time.RFC1123),
			"format_rfc1123z":     core.NewStringValue(time.RFC1123Z),
			"format_rfc3339":      core.NewStringValue(time.RFC3339),
			"format_rfc3339_nano": core.NewStringValue(time.RFC3339Nano),
			"format_kitchen":      core.NewStringValue(time.Kitchen),
			"format_stamp":        core.NewStringValue(time.Stamp),
			"format_stamp_milli":  core.NewStringValue(time.StampMilli),
			"format_stamp_micro":  core.NewStringValue(time.StampMicro),
			"format_stamp_nano":   core.NewStringValue(time.StampNano),
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
			9:  core.NewBuiltinFunction("date", timesDate, 7, true, true),                                 // date(year, month, day, hour, min, sec, nsec [,location]) => time
			10: core.NewBuiltinFunction("now", timesNow, 0, false, false),                                 // now() => time
			11: core.NewBuiltinFunction("parse", timesParse, 2, false, true),                              // parse(format, str) => time
			12: core.NewBuiltinFunction("unix", timesUnix, 2, false, true),                                // unix(sec, nsec) => time
			13: core.NewBuiltinFunction("add_date", timesAddDate, 4, false, true),                         // add_date(time, years, months, days) => time
			14: core.NewBuiltinFunction("in_location", timesInLocation, 2, false, true),                   // in_location(time, location) => time
			15: core.NewBuiltinFunction("from_unix", timesFromUnix, 1, false, true),                       // from_unix(sec) => time
			16: core.NewBuiltinFunction("from_unix_ms", timesFromUnixMs, 1, false, true),                  // from_unix_ms(msec) => time
			17: core.NewBuiltinFunction("from_unix_micro", timesFromUnixMicro, 1, false, true),            // from_unix_micro(usec) => time
			18: core.NewBuiltinFunction("from_unix_nano", timesFromUnixNano, 1, false, true),              // from_unix_nano(nsec) => time
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

func timesDate(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) < 7 || len(args) > 8 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.date", "7 or 8", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.date", "first", "int(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.date", "second", "int(compatible)", args[1].TypeName())
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.date", "third", "int(compatible)", args[2].TypeName())
	}
	i4, ok := args[3].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.date", "fourth", "int(compatible)", args[3].TypeName())
	}
	i5, ok := args[4].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.date", "fifth", "int(compatible)", args[4].TypeName())
	}
	i6, ok := args[5].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.date", "sixth", "int(compatible)", args[5].TypeName())
	}
	i7, ok := args[6].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.date", "seventh", "int(compatible)", args[6].TypeName())
	}

	var err error
	var loc *time.Location
	if len(args) == 8 {
		i8, ok := args[7].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("times.date", "eighth", "string(compatible)", args[7].TypeName())
		}
		loc, err = time.LoadLocation(i8)
		if err != nil {
			return raiseGo(errs.KindConversion, "times.date", err)
		}
	} else {
		loc = time.UTC
	}

	t := time.Date(int(i1), time.Month(i2), int(i3), int(i4), int(i5), int(i6), int(i7), loc)
	return core.NewTimeValue(t), nil
}

func timesNow(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.now", "0", len(args))
	}
	return core.NewTimeValue(time.Now()), nil
}

func timesParse(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.parse", "2", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.parse", "first", "string(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.parse", "second", "string(compatible)", args[1].TypeName())
	}

	parsed, err := time.Parse(s1, s2)
	if err != nil {
		return raiseGo(errs.KindConversion, "times.parse", err)
	}

	return core.NewTimeValue(parsed), nil
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

// The from_unix* family: an int in conversion context is a unix timestamp, in the encoding each
// name states. Unlike times.unix(sec, nsec) -- which predates these and returns the host's local
// zone -- these normalize to UTC, so the same script on two differently configured machines yields
// the same wall-clock components. Each one is the exact inverse of the time member accessor with
// the matching suffix (t.unix(), t.unix_ms(), t.unix_micro(), t.unix_nano()).
func timesFromUnix(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.from_unix", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.from_unix", "first", "int(compatible)", args[0].TypeName())
	}

	return core.NewTimeValue(time.Unix(i1, 0).UTC()), nil
}

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

func timesAddDate(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 4 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.add_date", "4", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.add_date", "first", "time(compatible)", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.add_date", "second", "int(compatible)", args[1].TypeName())
	}

	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.add_date", "third", "int(compatible)", args[2].TypeName())
	}

	i4, ok := args[3].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.add_date", "fourth", "int(compatible)", args[3].TypeName())
	}

	return core.NewTimeValue(t1.AddDate(int(i2), int(i3), int(i4))), nil
}

func timesInLocation(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.in_location", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.in_location", "first", "time(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.in_location", "second", "string(compatible)", args[1].TypeName())
	}

	location, err := time.LoadLocation(s2)
	if err != nil {
		return raiseGo(errs.KindConversion, "times.in_location", err)
	}

	return core.NewTimeValue(t1.In(location)), nil
}
