package stdlib

import (
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

var timesModule = map[string]core.Object{
	"format_ansic":        value.NewStaticString(time.ANSIC),
	"format_unix_date":    value.NewStaticString(time.UnixDate),
	"format_ruby_date":    value.NewStaticString(time.RubyDate),
	"format_rfc822":       value.NewStaticString(time.RFC822),
	"format_rfc822z":      value.NewStaticString(time.RFC822Z),
	"format_rfc850":       value.NewStaticString(time.RFC850),
	"format_rfc1123":      value.NewStaticString(time.RFC1123),
	"format_rfc1123z":     value.NewStaticString(time.RFC1123Z),
	"format_rfc3339":      value.NewStaticString(time.RFC3339),
	"format_rfc3339_nano": value.NewStaticString(time.RFC3339Nano),
	"format_kitchen":      value.NewStaticString(time.Kitchen),
	"format_stamp":        value.NewStaticString(time.Stamp),
	"format_stamp_milli":  value.NewStaticString(time.StampMilli),
	"format_stamp_micro":  value.NewStaticString(time.StampMicro),
	"format_stamp_nano":   value.NewStaticString(time.StampNano),
	"nanosecond":          value.NewStaticInt(int64(time.Nanosecond)),
	"microsecond":         value.NewStaticInt(int64(time.Microsecond)),
	"millisecond":         value.NewStaticInt(int64(time.Millisecond)),
	"second":              value.NewStaticInt(int64(time.Second)),
	"minute":              value.NewStaticInt(int64(time.Minute)),
	"hour":                value.NewStaticInt(int64(time.Hour)),
	"january":             value.NewStaticInt(int64(time.January)),
	"february":            value.NewStaticInt(int64(time.February)),
	"march":               value.NewStaticInt(int64(time.March)),
	"april":               value.NewStaticInt(int64(time.April)),
	"may":                 value.NewStaticInt(int64(time.May)),
	"june":                value.NewStaticInt(int64(time.June)),
	"july":                value.NewStaticInt(int64(time.July)),
	"august":              value.NewStaticInt(int64(time.August)),
	"september":           value.NewStaticInt(int64(time.September)),
	"october":             value.NewStaticInt(int64(time.October)),
	"november":            value.NewStaticInt(int64(time.November)),
	"december":            value.NewStaticInt(int64(time.December)),

	"sleep":                value.NewStaticBuiltinFunction("sleep", timesSleep, 1, false),                              // sleep(int)
	"parse_duration":       value.NewStaticBuiltinFunction("parse_duration", timesParseDuration, 1, false),             // parse_duration(str) => int
	"since":                value.NewStaticBuiltinFunction("since", timesSince, 1, false),                              // since(time) => int
	"until":                value.NewStaticBuiltinFunction("until", timesUntil, 1, false),                              // until(time) => int
	"duration_hours":       value.NewStaticBuiltinFunction("duration_hours", timesDurationHours, 1, false),             // duration_hours(int) => float
	"duration_minutes":     value.NewStaticBuiltinFunction("duration_minutes", timesDurationMinutes, 1, false),         // duration_minutes(int) => float
	"duration_nanoseconds": value.NewStaticBuiltinFunction("duration_nanoseconds", timesDurationNanoseconds, 1, false), // duration_nanoseconds(int) => int
	"duration_seconds":     value.NewStaticBuiltinFunction("duration_seconds", timesDurationSeconds, 1, false),         // duration_seconds(int) => float
	"duration_string":      value.NewStaticBuiltinFunction("duration_string", timesDurationString, 1, false),           // duration_string(int) => string
	"month_string":         value.NewStaticBuiltinFunction("month_string", timesMonthString, 1, false),                 // month_string(int) => string
	"date":                 value.NewStaticBuiltinFunction("date", timesDate, 7, true),                                 // date(year, month, day, hour, min, sec, nsec [,location]) => time
	"now":                  value.NewStaticBuiltinFunction("now", timesNow, 0, false),                                  // now() => time
	"parse":                value.NewStaticBuiltinFunction("parse", timesParse, 2, false),                              // parse(format, str) => time
	"unix":                 value.NewStaticBuiltinFunction("unix", timesUnix, 2, false),                                // unix(sec, nsec) => time
	"add":                  value.NewStaticBuiltinFunction("add", timesAdd, 2, false),                                  // add(time, int) => time
	"add_date":             value.NewStaticBuiltinFunction("add_date", timesAddDate, 4, false),                         // add_date(time, years, months, days) => time
	"sub":                  value.NewStaticBuiltinFunction("sub", timesSub, 2, false),                                  // sub(t time, u time) => int
	"after":                value.NewStaticBuiltinFunction("after", timesAfter, 2, false),                              // after(t time, u time) => bool
	"before":               value.NewStaticBuiltinFunction("before", timesBefore, 2, false),                            // before(t time, u time) => bool
	"time_year":            value.NewStaticBuiltinFunction("time_year", timesTimeYear, 1, false),                       // time_year(time) => int
	"time_month":           value.NewStaticBuiltinFunction("time_month", timesTimeMonth, 1, false),                     // time_month(time) => int
	"time_day":             value.NewStaticBuiltinFunction("time_day", timesTimeDay, 1, false),                         // time_day(time) => int
	"time_weekday":         value.NewStaticBuiltinFunction("time_weekday", timesTimeWeekday, 1, false),                 // time_weekday(time) => int
	"time_hour":            value.NewStaticBuiltinFunction("time_hour", timesTimeHour, 1, false),                       // time_hour(time) => int
	"time_minute":          value.NewStaticBuiltinFunction("time_minute", timesTimeMinute, 1, false),                   // time_minute(time) => int
	"time_second":          value.NewStaticBuiltinFunction("time_second", timesTimeSecond, 1, false),                   // time_second(time) => int
	"time_nanosecond":      value.NewStaticBuiltinFunction("time_nanosecond", timesTimeNanosecond, 1, false),           // time_nanosecond(time) => int
	"time_unix":            value.NewStaticBuiltinFunction("time_unix", timesTimeUnix, 1, false),                       // time_unix(time) => int
	"time_unix_nano":       value.NewStaticBuiltinFunction("time_unix_nano", timesTimeUnixNano, 1, false),              // time_unix_nano(time) => int
	"time_format":          value.NewStaticBuiltinFunction("time_format", timesTimeFormat, 2, false),                   // time_format(time, format) => string
	"time_location":        value.NewStaticBuiltinFunction("time_location", timesTimeLocation, 1, false),               // time_location(time) => string
	"time_string":          value.NewStaticBuiltinFunction("time_string", timesTimeString, 1, false),                   // time_string(time) => string
	"is_zero":              value.NewStaticBuiltinFunction("is_zero", timesIsZero, 1, false),                           // is_zero(time) => bool
	"to_local":             value.NewStaticBuiltinFunction("to_local", timesToLocal, 1, false),                         // to_local(time) => time
	"to_utc":               value.NewStaticBuiltinFunction("to_utc", timesToUTC, 1, false),                             // to_utc(time) => time
	"in_location":          value.NewStaticBuiltinFunction("in_location", timesInLocation, 2, false),                   // in_location(time, location) => time
}

func timesSleep(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.sleep", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.sleep", "first", "int(compatible)", args[0])
	}

	time.Sleep(time.Duration(i1))
	return vm.Allocator().NewUndefined(), nil
}

func timesParseDuration(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.parse_duration", "1", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.parse_duration", "first", "string(compatible)", args[0])
	}

	dur, err := time.ParseDuration(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}

	return vm.Allocator().NewInt(int64(dur)), nil
}

func timesSince(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.since", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.since", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(time.Since(t1))), nil
}

func timesUntil(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.until", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.until", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(time.Until(t1))), nil
}

func timesDurationHours(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.duration_hours", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.duration_hours", "first", "int(compatible)", args[0])
	}

	return vm.Allocator().NewFloat(time.Duration(i1).Hours()), nil
}

func timesDurationMinutes(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.duration_minutes", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.duration_minutes", "first", "int(compatible)", args[0])
	}

	return vm.Allocator().NewFloat(time.Duration(i1).Minutes()), nil
}

func timesDurationNanoseconds(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.duration_nanoseconds", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.duration_nanoseconds", "first", "int(compatible)", args[0])
	}

	return vm.Allocator().NewInt(time.Duration(i1).Nanoseconds()), nil
}

func timesDurationSeconds(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.duration_seconds", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.duration_seconds", "first", "int(compatible)", args[0])
	}

	return vm.Allocator().NewFloat(time.Duration(i1).Seconds()), nil
}

func timesDurationString(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.duration_string", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.duration_string", "first", "int(compatible)", args[0])
	}

	return vm.Allocator().NewString(time.Duration(i1).String()), nil
}

func timesMonthString(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.month_string", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.month_string", "first", "int(compatible)", args[0])
	}

	return vm.Allocator().NewString(time.Month(i1).String()), nil
}

func timesDate(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) < 7 || len(args) > 8 {
		return nil, core.NewWrongNumArgumentsError("times.date", "7 or 8", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.date", "first", "int(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.date", "second", "int(compatible)", args[1])
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.date", "third", "int(compatible)", args[2])
	}
	i4, ok := args[3].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.date", "fourth", "int(compatible)", args[3])
	}
	i5, ok := args[4].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.date", "fifth", "int(compatible)", args[4])
	}
	i6, ok := args[5].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.date", "sixth", "int(compatible)", args[5])
	}
	i7, ok := args[6].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.date", "seventh", "int(compatible)", args[6])
	}

	var loc *time.Location
	if len(args) == 8 {
		i8, ok := args[7].AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("times.date", "eighth", "string(compatible)", args[7])
		}
		loc, err = time.LoadLocation(i8)
		if err != nil {
			ret = wrapError(vm, err)
			return
		}
	} else {
		loc = time.Now().Location()
	}

	return vm.Allocator().NewTime(time.Date(int(i1), time.Month(i2), int(i3), int(i4), int(i5), int(i6), int(i7), loc)), nil
}

func timesNow(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("times.now", "0", len(args))
	}
	return vm.Allocator().NewTime(time.Now()), nil
}

func timesParse(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("times.parse", "2", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.parse", "first", "string(compatible)", args[0])
	}

	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.parse", "second", "string(compatible)", args[1])
	}

	parsed, err := time.Parse(s1, s2)
	if err != nil {
		ret = wrapError(vm, err)
		return
	}

	return vm.Allocator().NewTime(parsed), nil
}

func timesUnix(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("times.unix", "2", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.unix", "first", "int(compatible)", args[0])
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.unix", "second", "int(compatible)", args[1])
	}

	return vm.Allocator().NewTime(time.Unix(i1, i2)), nil
}

func timesAdd(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("times.add", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.add", "first", "time(compatible)", args[0])
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.add", "second", "int(compatible)", args[1])
	}

	return vm.Allocator().NewTime(t1.Add(time.Duration(i2))), nil
}

func timesSub(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("times.sub", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.sub", "first", "time(compatible)", args[0])
	}

	t2, ok := args[1].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.sub", "second", "time(compatible)", args[1])
	}

	return vm.Allocator().NewInt(int64(t1.Sub(t2))), nil
}

func timesAddDate(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 4 {
		return nil, core.NewWrongNumArgumentsError("times.add_date", "4", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.add_date", "first", "time(compatible)", args[0])
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.add_date", "second", "int(compatible)", args[1])
	}

	i3, ok := args[2].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.add_date", "third", "int(compatible)", args[2])
	}

	i4, ok := args[3].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.add_date", "fourth", "int(compatible)", args[3])
	}

	return vm.Allocator().NewTime(t1.AddDate(int(i2), int(i3), int(i4))), nil
}

func timesAfter(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("times.after", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.after", "first", "time(compatible)", args[0])
	}

	t2, ok := args[1].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.after", "second", "time(compatible)", args[1])
	}

	return vm.Allocator().NewBool(t1.After(t2)), nil
}

func timesBefore(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("times.before", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.before", "first", "time(compatible)", args[0])
	}

	t2, ok := args[1].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.before", "second", "time(compatible)", args[1])
	}

	return vm.Allocator().NewBool(t1.Before(t2)), nil
}

func timesTimeYear(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_year", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_year", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(t1.Year())), nil
}

func timesTimeMonth(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_month", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_month", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(t1.Month())), nil
}

func timesTimeDay(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_day", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_day", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(t1.Day())), nil
}

func timesTimeWeekday(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_weekday", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_weekday", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(t1.Weekday())), nil
}

func timesTimeHour(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_hour", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_hour", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(t1.Hour())), nil
}

func timesTimeMinute(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_minute", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_minute", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(t1.Minute())), nil
}

func timesTimeSecond(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_second", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_second", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(t1.Second())), nil
}

func timesTimeNanosecond(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_nanosecond", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_nanosecond", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(int64(t1.Nanosecond())), nil
}

func timesTimeUnix(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_unix", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_unix", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(t1.Unix()), nil
}

func timesTimeUnixNano(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_unix_nano", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_unix_nano", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewInt(t1.UnixNano()), nil
}

func timesTimeFormat(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("times.time_format", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_format", "first", "time(compatible)", args[0])
	}

	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_format", "second", "string(compatible)", args[1])
	}

	s := t1.Format(s2)
	if len(s) > core.MaxStringLen {

		return nil, core.NewStringLimitError("times.time_format")
	}

	return vm.Allocator().NewString(s), nil
}

func timesIsZero(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.is_zero", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.is_zero", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewBool(t1.IsZero()), nil
}

func timesToLocal(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.to_local", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.to_local", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewTime(t1.Local()), nil
}

func timesToUTC(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.to_utc", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.to_utc", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewTime(t1.UTC()), nil
}

func timesTimeLocation(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_location", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_location", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewString(t1.Location().String()), nil
}

func timesInLocation(vm core.VM, args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("times.in_location", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.in_location", "first", "time(compatible)", args[0])
	}

	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.in_location", "second", "string(compatible)", args[1])
	}

	location, err := time.LoadLocation(s2)
	if err != nil {
		ret = wrapError(vm, err)
		return
	}

	return vm.Allocator().NewTime(t1.In(location)), nil
}

func timesTimeString(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("times.time_string", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("times.time_string", "first", "time(compatible)", args[0])
	}

	return vm.Allocator().NewString(t1.String()), nil
}
