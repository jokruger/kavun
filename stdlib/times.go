package stdlib

import (
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

var timesModule = map[string]core.Object{
	"format_ansic":        value.NewString(time.ANSIC),
	"format_unix_date":    value.NewString(time.UnixDate),
	"format_ruby_date":    value.NewString(time.RubyDate),
	"format_rfc822":       value.NewString(time.RFC822),
	"format_rfc822z":      value.NewString(time.RFC822Z),
	"format_rfc850":       value.NewString(time.RFC850),
	"format_rfc1123":      value.NewString(time.RFC1123),
	"format_rfc1123z":     value.NewString(time.RFC1123Z),
	"format_rfc3339":      value.NewString(time.RFC3339),
	"format_rfc3339_nano": value.NewString(time.RFC3339Nano),
	"format_kitchen":      value.NewString(time.Kitchen),
	"format_stamp":        value.NewString(time.Stamp),
	"format_stamp_milli":  value.NewString(time.StampMilli),
	"format_stamp_micro":  value.NewString(time.StampMicro),
	"format_stamp_nano":   value.NewString(time.StampNano),
	"nanosecond":          value.NewInt(int64(time.Nanosecond)),
	"microsecond":         value.NewInt(int64(time.Microsecond)),
	"millisecond":         value.NewInt(int64(time.Millisecond)),
	"second":              value.NewInt(int64(time.Second)),
	"minute":              value.NewInt(int64(time.Minute)),
	"hour":                value.NewInt(int64(time.Hour)),
	"january":             value.NewInt(int64(time.January)),
	"february":            value.NewInt(int64(time.February)),
	"march":               value.NewInt(int64(time.March)),
	"april":               value.NewInt(int64(time.April)),
	"may":                 value.NewInt(int64(time.May)),
	"june":                value.NewInt(int64(time.June)),
	"july":                value.NewInt(int64(time.July)),
	"august":              value.NewInt(int64(time.August)),
	"september":           value.NewInt(int64(time.September)),
	"october":             value.NewInt(int64(time.October)),
	"november":            value.NewInt(int64(time.November)),
	"december":            value.NewInt(int64(time.December)),

	"sleep":                value.NewBuiltinFunction("sleep", timesSleep, 1, false),                              // sleep(int)
	"parse_duration":       value.NewBuiltinFunction("parse_duration", timesParseDuration, 1, false),             // parse_duration(str) => int
	"since":                value.NewBuiltinFunction("since", timesSince, 1, false),                              // since(time) => int
	"until":                value.NewBuiltinFunction("until", timesUntil, 1, false),                              // until(time) => int
	"duration_hours":       value.NewBuiltinFunction("duration_hours", timesDurationHours, 1, false),             // duration_hours(int) => float
	"duration_minutes":     value.NewBuiltinFunction("duration_minutes", timesDurationMinutes, 1, false),         // duration_minutes(int) => float
	"duration_nanoseconds": value.NewBuiltinFunction("duration_nanoseconds", timesDurationNanoseconds, 1, false), // duration_nanoseconds(int) => int
	"duration_seconds":     value.NewBuiltinFunction("duration_seconds", timesDurationSeconds, 1, false),         // duration_seconds(int) => float
	"duration_string":      value.NewBuiltinFunction("duration_string", timesDurationString, 1, false),           // duration_string(int) => string
	"month_string":         value.NewBuiltinFunction("month_string", timesMonthString, 1, false),                 // month_string(int) => string
	"date":                 value.NewBuiltinFunction("date", timesDate, 7, false),                                // date(year, month, day, hour, min, sec, nsec) => time
	"now":                  value.NewBuiltinFunction("now", timesNow, 0, false),                                  // now() => time
	"parse":                value.NewBuiltinFunction("parse", timesParse, 2, false),                              // parse(format, str) => time
	"unix":                 value.NewBuiltinFunction("unix", timesUnix, 2, false),                                // unix(sec, nsec) => time
	"add":                  value.NewBuiltinFunction("add", timesAdd, 2, false),                                  // add(time, int) => time
	"add_date":             value.NewBuiltinFunction("add_date", timesAddDate, 4, false),                         // add_date(time, years, months, days) => time
	"sub":                  value.NewBuiltinFunction("sub", timesSub, 2, false),                                  // sub(t time, u time) => int
	"after":                value.NewBuiltinFunction("after", timesAfter, 2, false),                              // after(t time, u time) => bool
	"before":               value.NewBuiltinFunction("before", timesBefore, 2, false),                            // before(t time, u time) => bool
	"time_year":            value.NewBuiltinFunction("time_year", timesTimeYear, 1, false),                       // time_year(time) => int
	"time_month":           value.NewBuiltinFunction("time_month", timesTimeMonth, 1, false),                     // time_month(time) => int
	"time_day":             value.NewBuiltinFunction("time_day", timesTimeDay, 1, false),                         // time_day(time) => int
	"time_weekday":         value.NewBuiltinFunction("time_weekday", timesTimeWeekday, 1, false),                 // time_weekday(time) => int
	"time_hour":            value.NewBuiltinFunction("time_hour", timesTimeHour, 1, false),                       // time_hour(time) => int
	"time_minute":          value.NewBuiltinFunction("time_minute", timesTimeMinute, 1, false),                   // time_minute(time) => int
	"time_second":          value.NewBuiltinFunction("time_second", timesTimeSecond, 1, false),                   // time_second(time) => int
	"time_nanosecond":      value.NewBuiltinFunction("time_nanosecond", timesTimeNanosecond, 1, false),           // time_nanosecond(time) => int
	"time_unix":            value.NewBuiltinFunction("time_unix", timesTimeUnix, 1, false),                       // time_unix(time) => int
	"time_unix_nano":       value.NewBuiltinFunction("time_unix_nano", timesTimeUnixNano, 1, false),              // time_unix_nano(time) => int
	"time_format":          value.NewBuiltinFunction("time_format", timesTimeFormat, 2, false),                   // time_format(time, format) => string
	"time_location":        value.NewBuiltinFunction("time_location", timesTimeLocation, 1, false),               // time_location(time) => string
	"time_string":          value.NewBuiltinFunction("time_string", timesTimeString, 1, false),                   // time_string(time) => string
	"is_zero":              value.NewBuiltinFunction("is_zero", timesIsZero, 1, false),                           // is_zero(time) => bool
	"to_local":             value.NewBuiltinFunction("to_local", timesToLocal, 1, false),                         // to_local(time) => time
	"to_utc":               value.NewBuiltinFunction("to_utc", timesToUTC, 1, false),                             // to_utc(time) => time
	"in_location":          value.NewBuiltinFunction("in_location", timesInLocation, 2, false),                   // in_location(time, location) => time
}

func timesSleep(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.sleep", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.sleep", "first", "int(compatible)", args[0])
	}

	time.Sleep(time.Duration(i1))
	ret = value.UndefinedValue

	return
}

func timesParseDuration(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.parse_duration", "1", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.InvalidArgumentType("times.parse_duration", "first", "string(compatible)", args[0])
	}

	dur, err := time.ParseDuration(s1)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = value.NewInt(int64(dur))

	return
}

func timesSince(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.since", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.since", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(time.Since(t1)))

	return
}

func timesUntil(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.until", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.until", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(time.Until(t1)))

	return
}

func timesDurationHours(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.duration_hours", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.duration_hours", "first", "int(compatible)", args[0])
	}

	ret = value.NewFloat(time.Duration(i1).Hours())

	return
}

func timesDurationMinutes(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.duration_minutes", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.duration_minutes", "first", "int(compatible)", args[0])
	}

	ret = value.NewFloat(time.Duration(i1).Minutes())

	return
}

func timesDurationNanoseconds(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.duration_nanoseconds", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.duration_nanoseconds", "first", "int(compatible)", args[0])
	}

	ret = value.NewInt(time.Duration(i1).Nanoseconds())

	return
}

func timesDurationSeconds(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.duration_seconds", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.duration_seconds", "first", "int(compatible)", args[0])
	}

	ret = value.NewFloat(time.Duration(i1).Seconds())

	return
}

func timesDurationString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.duration_string", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.duration_string", "first", "int(compatible)", args[0])
	}

	ret = value.NewString(time.Duration(i1).String())

	return
}

func timesMonthString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.month_string", "1", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.month_string", "first", "int(compatible)", args[0])
	}

	ret = value.NewString(time.Month(i1).String())

	return
}

func timesDate(args ...core.Object) (ret core.Object, err error) {
	if len(args) < 7 || len(args) > 8 {
		return nil, core.WrongNumArguments("times.date", "7 or 8", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.date", "first", "int(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.date", "second", "int(compatible)", args[1])
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.date", "third", "int(compatible)", args[2])
	}
	i4, ok := args[3].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.date", "fourth", "int(compatible)", args[3])
	}
	i5, ok := args[4].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.date", "fifth", "int(compatible)", args[4])
	}
	i6, ok := args[5].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.date", "sixth", "int(compatible)", args[5])
	}
	i7, ok := args[6].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.date", "seventh", "int(compatible)", args[6])
	}

	var loc *time.Location
	if len(args) == 8 {
		i8, ok := args[7].AsString()
		if !ok {
			return nil, core.InvalidArgumentType("times.date", "eighth", "string(compatible)", args[7])
		}
		loc, err = time.LoadLocation(i8)
		if err != nil {
			ret = wrapError(err)
			return
		}
	} else {
		loc = time.Now().Location()
	}

	ret = value.NewTime(time.Date(int(i1), time.Month(i2), int(i3), int(i4), int(i5), int(i6), int(i7), loc))

	return
}

func timesNow(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.WrongNumArguments("times.now", "0", len(args))
	}

	ret = value.NewTime(time.Now())

	return
}

func timesParse(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.WrongNumArguments("times.parse", "2", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return nil, core.InvalidArgumentType("times.parse", "first", "string(compatible)", args[0])
	}

	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.InvalidArgumentType("times.parse", "second", "string(compatible)", args[1])
	}

	parsed, err := time.Parse(s1, s2)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = value.NewTime(parsed)

	return
}

func timesUnix(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.WrongNumArguments("times.unix", "2", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.unix", "first", "int(compatible)", args[0])
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.unix", "second", "int(compatible)", args[1])
	}

	ret = value.NewTime(time.Unix(i1, i2))

	return
}

func timesAdd(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.WrongNumArguments("times.add", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.add", "first", "time(compatible)", args[0])
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.add", "second", "int(compatible)", args[1])
	}

	ret = value.NewTime(t1.Add(time.Duration(i2)))

	return
}

func timesSub(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.WrongNumArguments("times.sub", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.sub", "first", "time(compatible)", args[0])
	}

	t2, ok := args[1].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.sub", "second", "time(compatible)", args[1])
	}

	ret = value.NewInt(int64(t1.Sub(t2)))

	return
}

func timesAddDate(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 4 {
		return nil, core.WrongNumArguments("times.add_date", "4", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.add_date", "first", "time(compatible)", args[0])
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.add_date", "second", "int(compatible)", args[1])
	}

	i3, ok := args[2].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.add_date", "third", "int(compatible)", args[2])
	}

	i4, ok := args[3].AsInt()
	if !ok {
		return nil, core.InvalidArgumentType("times.add_date", "fourth", "int(compatible)", args[3])
	}

	ret = value.NewTime(t1.AddDate(int(i2), int(i3), int(i4)))

	return
}

func timesAfter(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.WrongNumArguments("times.after", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.after", "first", "time(compatible)", args[0])
	}

	t2, ok := args[1].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.after", "second", "time(compatible)", args[1])
	}

	if t1.After(t2) {
		ret = value.TrueValue
	} else {
		ret = value.FalseValue
	}

	return
}

func timesBefore(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.WrongNumArguments("times.before", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.before", "first", "time(compatible)", args[0])
	}

	t2, ok := args[1].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.before", "second", "time(compatible)", args[1])
	}

	if t1.Before(t2) {
		ret = value.TrueValue
	} else {
		ret = value.FalseValue
	}

	return
}

func timesTimeYear(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_year", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_year", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(t1.Year()))

	return
}

func timesTimeMonth(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_month", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_month", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(t1.Month()))

	return
}

func timesTimeDay(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_day", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_day", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(t1.Day()))

	return
}

func timesTimeWeekday(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_weekday", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_weekday", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(t1.Weekday()))

	return
}

func timesTimeHour(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_hour", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_hour", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(t1.Hour()))

	return
}

func timesTimeMinute(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_minute", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_minute", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(t1.Minute()))

	return
}

func timesTimeSecond(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_second", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_second", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(t1.Second()))

	return
}

func timesTimeNanosecond(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_nanosecond", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_nanosecond", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(int64(t1.Nanosecond()))

	return
}

func timesTimeUnix(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_unix", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_unix", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(t1.Unix())

	return
}

func timesTimeUnixNano(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_unix_nano", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_unix_nano", "first", "time(compatible)", args[0])
	}

	ret = value.NewInt(t1.UnixNano())

	return
}

func timesTimeFormat(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.WrongNumArguments("times.time_format", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_format", "first", "time(compatible)", args[0])
	}

	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_format", "second", "string(compatible)", args[1])
	}

	s := t1.Format(s2)
	if len(s) > core.MaxStringLen {

		return nil, core.StringLimit("times.time_format")
	}

	ret = value.NewString(s)

	return
}

func timesIsZero(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.is_zero", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.is_zero", "first", "time(compatible)", args[0])
	}

	if t1.IsZero() {
		ret = value.TrueValue
	} else {
		ret = value.FalseValue
	}

	return
}

func timesToLocal(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.to_local", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.to_local", "first", "time(compatible)", args[0])
	}

	ret = value.NewTime(t1.Local())

	return
}

func timesToUTC(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.to_utc", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.to_utc", "first", "time(compatible)", args[0])
	}

	ret = value.NewTime(t1.UTC())

	return
}

func timesTimeLocation(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_location", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_location", "first", "time(compatible)", args[0])
	}

	ret = value.NewString(t1.Location().String())

	return
}

func timesInLocation(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.WrongNumArguments("times.in_location", "2", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.in_location", "first", "time(compatible)", args[0])
	}

	s2, ok := args[1].AsString()
	if !ok {
		return nil, core.InvalidArgumentType("times.in_location", "second", "string(compatible)", args[1])
	}

	location, err := time.LoadLocation(s2)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = value.NewTime(t1.In(location))

	return
}

func timesTimeString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.WrongNumArguments("times.time_string", "1", len(args))
	}

	t1, ok := args[0].AsTime()
	if !ok {
		return nil, core.InvalidArgumentType("times.time_string", "first", "time(compatible)", args[0])
	}

	ret = value.NewString(t1.String())

	return
}
