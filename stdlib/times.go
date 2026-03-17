package stdlib

import (
	"github.com/jokruger/gs/core"
)

var timesModule = map[string]core.Object{
	/*
		"format_ansic":        &value.String{Value: time.ANSIC},
		"format_unix_date":    &value.String{Value: time.UnixDate},
		"format_ruby_date":    &value.String{Value: time.RubyDate},
		"format_rfc822":       &value.String{Value: time.RFC822},
		"format_rfc822z":      &value.String{Value: time.RFC822Z},
		"format_rfc850":       &value.String{Value: time.RFC850},
		"format_rfc1123":      &value.String{Value: time.RFC1123},
		"format_rfc1123z":     &value.String{Value: time.RFC1123Z},
		"format_rfc3339":      &value.String{Value: time.RFC3339},
		"format_rfc3339_nano": &value.String{Value: time.RFC3339Nano},
		"format_kitchen":      &value.String{Value: time.Kitchen},
		"format_stamp":        &value.String{Value: time.Stamp},
		"format_stamp_milli":  &value.String{Value: time.StampMilli},
		"format_stamp_micro":  &value.String{Value: time.StampMicro},
		"format_stamp_nano":   &value.String{Value: time.StampNano},
		"nanosecond":          &value.Int{Value: int64(time.Nanosecond)},
		"microsecond":         &value.Int{Value: int64(time.Microsecond)},
		"millisecond":         &value.Int{Value: int64(time.Millisecond)},
		"second":              &value.Int{Value: int64(time.Second)},
		"minute":              &value.Int{Value: int64(time.Minute)},
		"hour":                &value.Int{Value: int64(time.Hour)},
		"january":             &value.Int{Value: int64(time.January)},
		"february":            &value.Int{Value: int64(time.February)},
		"march":               &value.Int{Value: int64(time.March)},
		"april":               &value.Int{Value: int64(time.April)},
		"may":                 &value.Int{Value: int64(time.May)},
		"june":                &value.Int{Value: int64(time.June)},
		"july":                &value.Int{Value: int64(time.July)},
		"august":              &value.Int{Value: int64(time.August)},
		"september":           &value.Int{Value: int64(time.September)},
		"october":             &value.Int{Value: int64(time.October)},
		"november":            &value.Int{Value: int64(time.November)},
		"december":            &value.Int{Value: int64(time.December)},
		"sleep": &value.BuiltinFunction{
			Name:  "sleep",
			Value: timesSleep,
		}, // sleep(int)
		"parse_duration": &value.BuiltinFunction{
			Name:  "parse_duration",
			Value: timesParseDuration,
		}, // parse_duration(str) => int
		"since": &value.BuiltinFunction{
			Name:  "since",
			Value: timesSince,
		}, // since(time) => int
		"until": &value.BuiltinFunction{
			Name:  "until",
			Value: timesUntil,
		}, // until(time) => int
		"duration_hours": &value.BuiltinFunction{
			Name:  "duration_hours",
			Value: timesDurationHours,
		}, // duration_hours(int) => float
		"duration_minutes": &value.BuiltinFunction{
			Name:  "duration_minutes",
			Value: timesDurationMinutes,
		}, // duration_minutes(int) => float
		"duration_nanoseconds": &value.BuiltinFunction{
			Name:  "duration_nanoseconds",
			Value: timesDurationNanoseconds,
		}, // duration_nanoseconds(int) => int
		"duration_seconds": &value.BuiltinFunction{
			Name:  "duration_seconds",
			Value: timesDurationSeconds,
		}, // duration_seconds(int) => float
		"duration_string": &value.BuiltinFunction{
			Name:  "duration_string",
			Value: timesDurationString,
		}, // duration_string(int) => string
		"month_string": &value.BuiltinFunction{
			Name:  "month_string",
			Value: timesMonthString,
		}, // month_string(int) => string
		"date": &value.BuiltinFunction{
			Name:  "date",
			Value: timesDate,
		}, // date(year, month, day, hour, min, sec, nsec) => time
		"now": &value.BuiltinFunction{
			Name:  "now",
			Value: timesNow,
		}, // now() => time
		"parse": &value.BuiltinFunction{
			Name:  "parse",
			Value: timesParse,
		}, // parse(format, str) => time
		"unix": &value.BuiltinFunction{
			Name:  "unix",
			Value: timesUnix,
		}, // unix(sec, nsec) => time
		"add": &value.BuiltinFunction{
			Name:  "add",
			Value: timesAdd,
		}, // add(time, int) => time
		"add_date": &value.BuiltinFunction{
			Name:  "add_date",
			Value: timesAddDate,
		}, // add_date(time, years, months, days) => time
		"sub": &value.BuiltinFunction{
			Name:  "sub",
			Value: timesSub,
		}, // sub(t time, u time) => int
		"after": &value.BuiltinFunction{
			Name:  "after",
			Value: timesAfter,
		}, // after(t time, u time) => bool
		"before": &value.BuiltinFunction{
			Name:  "before",
			Value: timesBefore,
		}, // before(t time, u time) => bool
		"time_year": &value.BuiltinFunction{
			Name:  "time_year",
			Value: timesTimeYear,
		}, // time_year(time) => int
		"time_month": &value.BuiltinFunction{
			Name:  "time_month",
			Value: timesTimeMonth,
		}, // time_month(time) => int
		"time_day": &value.BuiltinFunction{
			Name:  "time_day",
			Value: timesTimeDay,
		}, // time_day(time) => int
		"time_weekday": &value.BuiltinFunction{
			Name:  "time_weekday",
			Value: timesTimeWeekday,
		}, // time_weekday(time) => int
		"time_hour": &value.BuiltinFunction{
			Name:  "time_hour",
			Value: timesTimeHour,
		}, // time_hour(time) => int
		"time_minute": &value.BuiltinFunction{
			Name:  "time_minute",
			Value: timesTimeMinute,
		}, // time_minute(time) => int
		"time_second": &value.BuiltinFunction{
			Name:  "time_second",
			Value: timesTimeSecond,
		}, // time_second(time) => int
		"time_nanosecond": &value.BuiltinFunction{
			Name:  "time_nanosecond",
			Value: timesTimeNanosecond,
		}, // time_nanosecond(time) => int
		"time_unix": &value.BuiltinFunction{
			Name:  "time_unix",
			Value: timesTimeUnix,
		}, // time_unix(time) => int
		"time_unix_nano": &value.BuiltinFunction{
			Name:  "time_unix_nano",
			Value: timesTimeUnixNano,
		}, // time_unix_nano(time) => int
		"time_format": &value.BuiltinFunction{
			Name:  "time_format",
			Value: timesTimeFormat,
		}, // time_format(time, format) => string
		"time_location": &value.BuiltinFunction{
			Name:  "time_location",
			Value: timesTimeLocation,
		}, // time_location(time) => string
		"time_string": &value.BuiltinFunction{
			Name:  "time_string",
			Value: timesTimeString,
		}, // time_string(time) => string
		"is_zero": &value.BuiltinFunction{
			Name:  "is_zero",
			Value: timesIsZero,
		}, // is_zero(time) => bool
		"to_local": &value.BuiltinFunction{
			Name:  "to_local",
			Value: timesToLocal,
		}, // to_local(time) => time
		"to_utc": &value.BuiltinFunction{
			Name:  "to_utc",
			Value: timesToUTC,
		}, // to_utc(time) => time
		"in_location": &value.BuiltinFunction{
			Name:  "in_location",
			Value: timesInLocation,
		}, // in_location(time, location) => time
	*/
}

/*
func timesSleep(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	time.Sleep(time.Duration(i1))
	ret = value.UndefinedValue

	return
}

func timesParseDuration(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	dur, err := time.ParseDuration(s1)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = &value.Int{Value: int64(dur)}

	return
}

func timesSince(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(time.Since(t1))}

	return
}

func timesUntil(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(time.Until(t1))}

	return
}

func timesDurationHours(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Float{Value: time.Duration(i1).Hours()}

	return
}

func timesDurationMinutes(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Float{Value: time.Duration(i1).Minutes()}

	return
}

func timesDurationNanoseconds(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: time.Duration(i1).Nanoseconds()}

	return
}

func timesDurationSeconds(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Float{Value: time.Duration(i1).Seconds()}

	return
}

func timesDurationString(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.String{Value: time.Duration(i1).String()}

	return
}

func timesMonthString(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.String{Value: time.Month(i1).String()}

	return
}

func timesDate(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) < 7 || len(args) > 8 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}
	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}
	i3, ok := args[2].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}
	i4, ok := args[3].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "fourth",
			Expected: "int(compatible)",
			Found:    args[3].TypeName(),
		}
		return
	}
	i5, ok := args[4].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "fifth",
			Expected: "int(compatible)",
			Found:    args[4].TypeName(),
		}
		return
	}
	i6, ok := args[5].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "sixth",
			Expected: "int(compatible)",
			Found:    args[5].TypeName(),
		}
		return
	}
	i7, ok := args[6].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "seventh",
			Expected: "int(compatible)",
			Found:    args[6].TypeName(),
		}
		return
	}

	var loc *time.Location
	if len(args) == 8 {
		i8, ok := args[7].AsString()
		if !ok {
			err = gse.ErrInvalidArgumentType{
				Name:     "eighth",
				Expected: "string(compatible)",
				Found:    args[7].TypeName(),
			}
			return
		}
		loc, err = time.LoadLocation(i8)
		if err != nil {
			ret = wrapError(err)
			return
		}
	} else {
		loc = time.Now().Location()
	}

	ret = &value.Time{
		Value: time.Date(int(i1), time.Month(i2), int(i3), int(i4), int(i5), int(i6), int(i7), loc),
	}

	return
}

func timesNow(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		err = gse.ErrWrongNumArguments
		return
	}

	ret = &value.Time{Value: time.Now()}

	return
}

func timesParse(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	parsed, err := time.Parse(s1, s2)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = &value.Time{Value: parsed}

	return
}

func timesUnix(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	ret = &value.Time{Value: time.Unix(i1, i2)}

	return
}

func timesAdd(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	ret = &value.Time{Value: t1.Add(time.Duration(i2))}

	return
}

func timesSub(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	t2, ok := args[1].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "time(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(t1.Sub(t2))}

	return
}

func timesAddDate(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 4 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	i3, ok := args[2].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	i4, ok := args[3].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "fourth",
			Expected: "int(compatible)",
			Found:    args[3].TypeName(),
		}
		return
	}

	ret = &value.Time{Value: t1.AddDate(int(i2), int(i3), int(i4))}

	return
}

func timesAfter(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	t2, ok := args[1].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "time(compatible)",
			Found:    args[1].TypeName(),
		}
		return
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
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	t2, ok := args[1].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
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
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(t1.Year())}

	return
}

func timesTimeMonth(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(t1.Month())}

	return
}

func timesTimeDay(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(t1.Day())}

	return
}

func timesTimeWeekday(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(t1.Weekday())}

	return
}

func timesTimeHour(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(t1.Hour())}

	return
}

func timesTimeMinute(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(t1.Minute())}

	return
}

func timesTimeSecond(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(t1.Second())}

	return
}

func timesTimeNanosecond(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: int64(t1.Nanosecond())}

	return
}

func timesTimeUnix(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: t1.Unix()}

	return
}

func timesTimeUnixNano(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Int{Value: t1.UnixNano()}

	return
}

func timesTimeFormat(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	s := t1.Format(s2)
	if len(s) > core.MaxStringLen {

		return nil, gse.ErrStringLimit
	}

	ret = value.NewString(s)

	return
}

func timesIsZero(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
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
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Time{Value: t1.Local()}

	return
}

func timesToUTC(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.Time{Value: t1.UTC()}

	return
}

func timesTimeLocation(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.String{Value: t1.Location().String()}

	return
}

func timesInLocation(args ...core.Object) (
	ret core.Object,
	err error,
) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	location, err := time.LoadLocation(s2)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = &value.Time{Value: t1.In(location)}

	return
}

func timesTimeString(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	t1, ok := args[0].AsTime()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &value.String{Value: t1.String()}

	return
}
*/
