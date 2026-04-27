package stdlib

import (
	"time"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

var timesModule = map[string]core.Value{
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

	"sleep":                core.NewBuiltinFunctionValue("sleep", timesSleep, 1, false),                              // sleep(int)
	"parse_duration":       core.NewBuiltinFunctionValue("parse_duration", timesParseDuration, 1, false),             // parse_duration(str) => int
	"since":                core.NewBuiltinFunctionValue("since", timesSince, 1, false),                              // since(time) => int
	"until":                core.NewBuiltinFunctionValue("until", timesUntil, 1, false),                              // until(time) => int
	"duration_hours":       core.NewBuiltinFunctionValue("duration_hours", timesDurationHours, 1, false),             // duration_hours(int) => float
	"duration_minutes":     core.NewBuiltinFunctionValue("duration_minutes", timesDurationMinutes, 1, false),         // duration_minutes(int) => float
	"duration_nanoseconds": core.NewBuiltinFunctionValue("duration_nanoseconds", timesDurationNanoseconds, 1, false), // duration_nanoseconds(int) => int
	"duration_seconds":     core.NewBuiltinFunctionValue("duration_seconds", timesDurationSeconds, 1, false),         // duration_seconds(int) => float
	"duration_string":      core.NewBuiltinFunctionValue("duration_string", timesDurationString, 1, false),           // duration_string(int) => string
	"month_string":         core.NewBuiltinFunctionValue("month_string", timesMonthString, 1, false),                 // month_string(int) => string
	"date":                 core.NewBuiltinFunctionValue("date", timesDate, 7, true),                                 // date(year, month, day, hour, min, sec, nsec [,location]) => time
	"now":                  core.NewBuiltinFunctionValue("now", timesNow, 0, false),                                  // now() => time
	"parse":                core.NewBuiltinFunctionValue("parse", timesParse, 2, false),                              // parse(format, str) => time
	"unix":                 core.NewBuiltinFunctionValue("unix", timesUnix, 2, false),                                // unix(sec, nsec) => time
	"add":                  core.NewBuiltinFunctionValue("add", timesAdd, 2, false),                                  // add(time, int) => time
	"add_date":             core.NewBuiltinFunctionValue("add_date", timesAddDate, 4, false),                         // add_date(time, years, months, days) => time
	"sub":                  core.NewBuiltinFunctionValue("sub", timesSub, 2, false),                                  // sub(t time, u time) => int
	"after":                core.NewBuiltinFunctionValue("after", timesAfter, 2, false),                              // after(t time, u time) => bool
	"before":               core.NewBuiltinFunctionValue("before", timesBefore, 2, false),                            // before(t time, u time) => bool
	"time_year":            core.NewBuiltinFunctionValue("time_year", timesTimeYear, 1, false),                       // time_year(time) => int
	"time_month":           core.NewBuiltinFunctionValue("time_month", timesTimeMonth, 1, false),                     // time_month(time) => int
	"time_day":             core.NewBuiltinFunctionValue("time_day", timesTimeDay, 1, false),                         // time_day(time) => int
	"time_weekday":         core.NewBuiltinFunctionValue("time_weekday", timesTimeWeekday, 1, false),                 // time_weekday(time) => int
	"time_hour":            core.NewBuiltinFunctionValue("time_hour", timesTimeHour, 1, false),                       // time_hour(time) => int
	"time_minute":          core.NewBuiltinFunctionValue("time_minute", timesTimeMinute, 1, false),                   // time_minute(time) => int
	"time_second":          core.NewBuiltinFunctionValue("time_second", timesTimeSecond, 1, false),                   // time_second(time) => int
	"time_nanosecond":      core.NewBuiltinFunctionValue("time_nanosecond", timesTimeNanosecond, 1, false),           // time_nanosecond(time) => int
	"time_unix":            core.NewBuiltinFunctionValue("time_unix", timesTimeUnix, 1, false),                       // time_unix(time) => int
	"time_unix_nano":       core.NewBuiltinFunctionValue("time_unix_nano", timesTimeUnixNano, 1, false),              // time_unix_nano(time) => int
	"time_format":          core.NewBuiltinFunctionValue("time_format", timesTimeFormat, 2, false),                   // time_format(time, format) => string
	"time_location":        core.NewBuiltinFunctionValue("time_location", timesTimeLocation, 1, false),               // time_location(time) => string
	"time_string":          core.NewBuiltinFunctionValue("time_string", timesTimeString, 1, false),                   // time_string(time) => string
	"is_zero":              core.NewBuiltinFunctionValue("is_zero", timesIsZero, 1, false),                           // is_zero(time) => bool
	"to_local":             core.NewBuiltinFunctionValue("to_local", timesToLocal, 1, false),                         // to_local(time) => time
	"to_utc":               core.NewBuiltinFunctionValue("to_utc", timesToUTC, 1, false),                             // to_utc(time) => time
	"in_location":          core.NewBuiltinFunctionValue("in_location", timesInLocation, 2, false),                   // in_location(time, location) => time
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
		return wrapError(vm, err)
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

	return vm.Allocator().NewStringValue(time.Duration(i1).String()), nil
}

func timesMonthString(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.month_string", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.month_string", "first", "int(compatible)", args[0].TypeName())
	}

	return vm.Allocator().NewStringValue(time.Month(i1).String()), nil
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
			return wrapError(vm, err)
		}
	} else {
		loc = time.Now().Location()
	}

	d := vm.Allocator().NewTime()
	*d = time.Date(int(i1), time.Month(i2), int(i3), int(i4), int(i5), int(i6), int(i7), loc)
	return core.TimeValue(d), nil
}

func timesNow(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.now", "0", len(args))
	}

	d := vm.Allocator().NewTime()
	*d = time.Now()
	return core.TimeValue(d), nil
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
		return wrapError(vm, err)
	}

	d := vm.Allocator().NewTime()
	*d = parsed
	return core.TimeValue(d), nil
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

	d := vm.Allocator().NewTime()
	*d = time.Unix(i1, i2)
	return core.TimeValue(d), nil
}

func timesAdd(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.add", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.add", "first", "time(compatible)", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.add", "second", "int(compatible)", args[1].TypeName())
	}

	d := vm.Allocator().NewTime()
	*d = t1.Add(time.Duration(i2))
	return core.TimeValue(d), nil
}

func timesSub(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.sub", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.sub", "first", "time(compatible)", args[0].TypeName())
	}

	t2, ok := args[1].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.sub", "second", "time(compatible)", args[1].TypeName())
	}

	return core.IntValue(int64(t1.Sub(t2))), nil
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

	d := vm.Allocator().NewTime()
	*d = t1.AddDate(int(i2), int(i3), int(i4))
	return core.TimeValue(d), nil
}

func timesAfter(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.after", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.after", "first", "time(compatible)", args[0].TypeName())
	}

	t2, ok := args[1].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.after", "second", "time(compatible)", args[1].TypeName())
	}

	return core.BoolValue(t1.After(t2)), nil
}

func timesBefore(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.before", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.before", "first", "time(compatible)", args[0].TypeName())
	}

	t2, ok := args[1].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.before", "second", "time(compatible)", args[1].TypeName())
	}

	return core.BoolValue(t1.Before(t2)), nil
}

func timesTimeYear(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_year", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_year", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(t1.Year())), nil
}

func timesTimeMonth(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_month", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_month", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(t1.Month())), nil
}

func timesTimeDay(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_day", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_day", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(t1.Day())), nil
}

func timesTimeWeekday(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_weekday", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_weekday", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(t1.Weekday())), nil
}

func timesTimeHour(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_hour", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_hour", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(t1.Hour())), nil
}

func timesTimeMinute(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_minute", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_minute", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(t1.Minute())), nil
}

func timesTimeSecond(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_second", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_second", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(t1.Second())), nil
}

func timesTimeNanosecond(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_nanosecond", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_nanosecond", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(int64(t1.Nanosecond())), nil
}

func timesTimeUnix(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_unix", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_unix", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(t1.Unix()), nil
}

func timesTimeUnixNano(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_unix_nano", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_unix_nano", "first", "time(compatible)", args[0].TypeName())
	}

	return core.IntValue(t1.UnixNano()), nil
}

func timesTimeFormat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_format", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_format", "first", "time(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_format", "second", "string(compatible)", args[1].TypeName())
	}

	s := t1.Format(s2)
	return vm.Allocator().NewStringValue(s), nil
}

func timesIsZero(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.is_zero", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.is_zero", "first", "time(compatible)", args[0].TypeName())
	}

	return core.BoolValue(t1.IsZero()), nil
}

func timesToLocal(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.to_local", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.to_local", "first", "time(compatible)", args[0].TypeName())
	}

	d := vm.Allocator().NewTime()
	*d = t1.Local()
	return core.TimeValue(d), nil
}

func timesToUTC(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.to_utc", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.to_utc", "first", "time(compatible)", args[0].TypeName())
	}

	d := vm.Allocator().NewTime()
	*d = t1.UTC()
	return core.TimeValue(d), nil
}

func timesTimeLocation(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_location", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_location", "first", "time(compatible)", args[0].TypeName())
	}

	return vm.Allocator().NewStringValue(t1.Location().String()), nil
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
		return wrapError(vm, err)
	}

	d := vm.Allocator().NewTime()
	*d = t1.In(location)
	return core.TimeValue(d), nil
}

func timesTimeString(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("times.time_string", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("times.time_string", "first", "time(compatible)", args[0].TypeName())
	}

	return vm.Allocator().NewStringValue(t1.String()), nil
}
