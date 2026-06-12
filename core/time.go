package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

const timeTypeName = "time"

// TypeTime is a time type descriptor.
var TypeTime = ValueTypeDescr{
	Pin:          func(a *Arena, v Value) { a.PinTimeValue(v) },
	Retain:       func(a *Arena, v Value) { a.RetainTimeValue(v) },
	Release:      func(a *Arena, v Value) { a.ReleaseTimeValue(v) },
	Name:         ConstHook(timeTypeName),
	String:       timeTypeString,
	Format:       timeTypeFormat,
	Interface:    timeTypeInterface,
	EncodeJSON:   timeTypeEncodeJSON,
	EncodeBinary: timeTypeEncodeBinary,
	DecodeBinary: timeTypeDecodeBinary,
	IsTrue:       timeTypeIsTrue,
	Equal:        timeTypeEqual,
	Len:          ConstHook(int64(1)),
	BinaryOp:     timeTypeBinaryOp,
	MethodCall:   timeTypeMethodCall,
	AsString:     timeTypeAsString,
	AsInt:        timeTypeAsInt,
	AsBool:       timeTypeAsBool,
	AsTime:       timeTypeAsTime,
}

func timeTypeInterface(a *Arena, v Value) any {
	return *a.ResolveTimeValue(v)
}

func timeTypeIsTrue(a *Arena, v Value) bool {
	return !a.ResolveTimeValue(v).IsZero()
}

func timeTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveTimeValue(v)
	y, err := o.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return y, nil
}

func timeTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveTimeValue(v)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(*o); err != nil {
		return nil, fmt.Errorf("time: %w", err)
	}
	return buf.Bytes(), nil
}

func timeTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var t time.Time
	if err := dec.Decode(&t); err != nil {
		return fmt.Errorf("time: %w", err)
	}
	nv, err := a.NewTimeValue(t)
	if err != nil {
		return err
	}
	// we are not releasing old value here because it should be managed by caller Value.DecodeBinary
	*v = nv
	return nil
}

func timeTypeString(a *Arena, v Value) string {
	o := a.ResolveTimeValue(v)
	return fmt.Sprintf("time(%q)", o.Format(time.RFC3339Nano))
}

func timeTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return timeTypeString(a, v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(timeTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.HasPrec || sp.ZeroPad || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
	}

	t := a.ResolveTimeValue(v)

	var body string
	switch sp.Verb {
	case 0:
		body = t.Format(time.RFC3339)

	case '#':
		switch sp.Tail {
		case "", "iso":
			body = t.Format(time.RFC3339)
		case "isonano":
			body = t.Format(time.RFC3339Nano)
		case "date":
			body = t.Format("2006-01-02")
		case "time":
			body = t.Format("15:04:05")
		case "unix":
			body = strconv.FormatInt(t.Unix(), 10)
		case "unixms":
			body = strconv.FormatInt(t.UnixMilli(), 10)
		case "rfc822":
			body = t.Format(time.RFC822)
		default:
			out, err := strftime(*t, sp.Tail)
			if err != nil {
				return "", err
			}
			body = out
		}

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
	}

	return fspec.ApplyGenerics(body, sp, fspec.AlignLeft), nil
}

// strftime renders t using a Python-style layout containing %-directives. Supported codes:
//
//	%Y  4-digit year                    %B  full month name        %p  AM / PM
//	%y  2-digit year                    %b  abbreviated month name %P  am / pm
//	%C  century   (00-99)               %A  full weekday name      %j  day of year (001-366)
//	%m  month     (01-12)               %a  abbreviated weekday    %s  unix seconds
//	%d  day       (01-31)               %u  ISO weekday   (1-7)    %f  microseconds (000000-999999)
//	%e  day, space-padded ( 1-31)       %w  weekday       (0-6)    %Z  timezone abbreviation
//	%H  hour 24h  (00-23)               %V  ISO week      (01-53)  %z  timezone offset (-0700)
//	%I  hour 12h  (01-12)               %G  ISO week-numbering year
//	%M  minute    (00-59)               %n  literal newline
//	%S  second    (00-59)               %t  literal tab
//	%%  literal '%'
//
// An unknown directive returns an error.
func strftime(t time.Time, layout string) (string, error) {
	var b strings.Builder
	b.Grow(len(layout) + 8)
	for i := 0; i < len(layout); i++ {
		c := layout[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(layout) {
			return "", fmt.Errorf("time: trailing '%%' in format %q", layout)
		}
		i++
		switch layout[i] {
		case 'Y':
			fmt.Fprintf(&b, "%04d", t.Year())
		case 'y':
			y := t.Year() % 100
			if y < 0 {
				y = -y
			}
			fmt.Fprintf(&b, "%02d", y)
		case 'C':
			c := t.Year() / 100
			if c < 0 {
				c = -c
			}
			fmt.Fprintf(&b, "%02d", c)
		case 'm':
			fmt.Fprintf(&b, "%02d", int(t.Month()))
		case 'd':
			fmt.Fprintf(&b, "%02d", t.Day())
		case 'e':
			fmt.Fprintf(&b, "%2d", t.Day())
		case 'H':
			fmt.Fprintf(&b, "%02d", t.Hour())
		case 'I':
			h := t.Hour() % 12
			if h == 0 {
				h = 12
			}
			fmt.Fprintf(&b, "%02d", h)
		case 'M':
			fmt.Fprintf(&b, "%02d", t.Minute())
		case 'S':
			fmt.Fprintf(&b, "%02d", t.Second())
		case 'p':
			if t.Hour() < 12 {
				b.WriteString("AM")
			} else {
				b.WriteString("PM")
			}
		case 'P':
			if t.Hour() < 12 {
				b.WriteString("am")
			} else {
				b.WriteString("pm")
			}
		case 'B':
			b.WriteString(t.Month().String())
		case 'b':
			b.WriteString(t.Month().String()[:3])
		case 'A':
			b.WriteString(t.Weekday().String())
		case 'a':
			b.WriteString(t.Weekday().String()[:3])
		case 'u':
			// ISO 8601 weekday: 1=Mon … 7=Sun.
			wd := int(t.Weekday())
			if wd == 0 {
				wd = 7
			}
			fmt.Fprintf(&b, "%d", wd)
		case 'w':
			// POSIX weekday: 0=Sun … 6=Sat.
			fmt.Fprintf(&b, "%d", int(t.Weekday()))
		case 'V':
			// ISO 8601 week of year (01-53).
			_, week := t.ISOWeek()
			fmt.Fprintf(&b, "%02d", week)
		case 'G':
			// ISO 8601 week-numbering year.
			year, _ := t.ISOWeek()
			fmt.Fprintf(&b, "%04d", year)
		case 'j':
			fmt.Fprintf(&b, "%03d", t.YearDay())
		case 'Z':
			b.WriteString(t.Format("MST"))
		case 'z':
			b.WriteString(t.Format("-0700"))
		case 'f':
			fmt.Fprintf(&b, "%06d", t.Nanosecond()/1000)
		case 's':
			fmt.Fprintf(&b, "%d", t.Unix())
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case '%':
			b.WriteByte('%')
		default:
			return "", fmt.Errorf("time: unknown strftime directive %%%c in %q", layout[i], layout)
		}
	}
	return b.String(), nil
}

func timeTypeEqual(a *Arena, v Value, r Value) bool {
	t, ok := r.AsTime(a)
	if !ok {
		return false
	}
	o := a.ResolveTimeValue(v)
	return o.Equal(t)
}

func timeTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	o := a.ResolveTimeValue(v)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(!o.IsZero()), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(o.Unix()), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(o.String())

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName(a))
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := timeTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s)

	case "year":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Year())), nil

	case "month":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Month())), nil

	case "day":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Day())), nil

	case "hour":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Hour())), nil

	case "minute":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Minute())), nil

	case "second":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Second())), nil

	case "nanosecond":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Nanosecond())), nil

	case "unix":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(o.Unix()), nil

	case "unix_nano":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(o.UnixNano()), nil

	case "week_day":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Weekday())), nil

	case "year_day":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.YearDay())), nil

	case "month_name":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(o.Month().String())

	case "week_day_name":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(o.Weekday().String())

	case "utc":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewTimeValue(o.UTC())

	case "local":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewTimeValue(o.Local())

	case "format_date":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(o.Format(time.DateOnly))

	case "format_time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(o.Format(time.TimeOnly))

	case "format_datetime":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(o.Format(time.DateTime))

	case "zone_offset":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		_, offset := o.Zone()
		return IntValue(int64(offset)), nil

	case "zone_name":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		name, _ := o.Zone()
		return a.NewStringValue(name)

	case "repeat":
		return repeatScalarToArray(a, v, name, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}

func timeTypeBinaryOp(a *Arena, v Value, rhs Value, op token.Token) (Value, error) {
	o := a.ResolveTimeValue(v)

	if rhs.Type == VT_INT {
		r := int64(rhs.Data)
		switch op {
		case token.Add: // time + int => time
			return a.NewTimeValue(o.Add(time.Duration(r)))
		case token.Sub: // time - int => time
			return a.NewTimeValue(o.Add(time.Duration(-r)))
		}
	}

	r, ok := rhs.AsTime(a)
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
	}

	switch op {
	case token.Sub: // time - time => int (duration)
		return IntValue(int64(o.Sub(r))), nil
	case token.Less: // time < time => bool
		return BoolValue(o.Before(r)), nil
	case token.Greater:
		return BoolValue(o.After(r)), nil
	case token.LessEq:
		return BoolValue(o.Equal(r) || o.Before(r)), nil
	case token.GreaterEq:
		return BoolValue(o.Equal(r) || o.After(r)), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
}

func timeTypeAsString(a *Arena, v Value) (string, bool) {
	return a.ResolveTimeValue(v).String(), true
}

func timeTypeAsInt(a *Arena, v Value) (int64, bool) {
	return a.ResolveTimeValue(v).Unix(), true
}

func timeTypeAsBool(a *Arena, v Value) (bool, bool) {
	return !a.ResolveTimeValue(v).IsZero(), true
}

func timeTypeAsTime(a *Arena, v Value) (time.Time, bool) {
	return *a.ResolveTimeValue(v), true
}
