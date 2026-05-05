package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

// TimeValue creates new boxed time value.
func TimeValue(v *time.Time) Value {
	return Value{
		Type:  VT_TIME,
		Const: true,
		Ptr:   unsafe.Pointer(v),
	}
}

// NewTimeValue creates new (heap-allocated) boxed time value.
func NewTimeValue(t time.Time) Value {
	o := &t
	return TimeValue(o)
}

/* Time type methods */

func timeTypeName(v Value) string {
	return "time"
}

func timeTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*time.Time)(v.Ptr)
	y, err := o.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return y, nil
}

func timeTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*time.Time)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(*o); err != nil {
		return nil, fmt.Errorf("time: %w", err)
	}
	return buf.Bytes(), nil
}

func timeTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var t time.Time
	if err := dec.Decode(&t); err != nil {
		return fmt.Errorf("time: %w", err)
	}
	v.Ptr = unsafe.Pointer(&t)
	return nil
}

func timeTypeString(v Value) string {
	o := (*time.Time)(v.Ptr)
	return fmt.Sprintf("time(%q)", o.Format(time.RFC3339Nano))
}

func timeTypeFormat(v Value, s fspec.FormatSpec) (string, error) {
	if s.Verb == 'v' {
		return timeTypeString(v), nil
	}
	if s.Sign != fspec.SignDefault || s.Grouping != 0 || s.HasPrec || s.ZeroPad || s.CoerceZero {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
	}
	t := *(*time.Time)(v.Ptr)

	var body string
	switch s.Verb {
	case 0:
		body = t.Format(time.RFC3339Nano)

	case '#':
		switch s.Tail {
		case "", "iso":
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
			out, err := strftime(t, s.Tail)
			if err != nil {
				return "", err
			}
			body = out
		}

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
	}

	return fspec.ApplyGenerics(body, s, fspec.AlignLeft), nil
}

// strftime renders t using a Python-style layout containing %-directives. Supported codes:
//
//	%Y  4-digit year                    %B  full month name        %p  AM / PM
//	%y  2-digit year                    %b  abbreviated month name %P  am / pm
//	%m  month     (01-12)               %A  full weekday name      %j  day of year (001-366)
//	%d  day       (01-31)               %a  abbreviated weekday    %s  unix seconds
//	%e  day, space-padded ( 1-31)       %Z  timezone abbreviation  %f  microseconds (000000-999999)
//	%H  hour 24h  (00-23)               %z  timezone offset (-0700)
//	%I  hour 12h  (01-12)               %n  literal newline
//	%M  minute    (00-59)               %t  literal tab
//	%S  second    (00-59)               %%  literal '%'
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

func timeTypeInterface(v Value) any {
	o := (*time.Time)(v.Ptr)
	return *o
}

func timeTypeEqual(v Value, r Value) bool {
	t, ok := r.AsTime()
	if !ok {
		return false
	}
	o := (*time.Time)(v.Ptr)
	return o.Equal(t)
}

func timeTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)

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
		b, _ := timeTypeAsBool(v)
		return BoolValue(b), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := timeTypeAsInt(v)
		return IntValue(i), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.String()), nil

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := timeTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return vm.Allocator().NewStringValue(s), nil

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
		return vm.Allocator().NewStringValue(o.Month().String()), nil

	case "week_day_name":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.Weekday().String()), nil

	case "utc":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d := vm.Allocator().NewTime()
		*d = o.UTC()
		return TimeValue(d), nil

	case "local":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d := vm.Allocator().NewTime()
		*d = o.Local()
		return TimeValue(d), nil

	case "format_date":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.Format(time.DateOnly)), nil

	case "format_time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.Format(time.TimeOnly)), nil

	case "format_datetime":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.Format(time.DateTime)), nil

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
		return vm.Allocator().NewStringValue(name), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func timeTypeIsTrue(v Value) bool {
	o := (*time.Time)(v.Ptr)
	return !o.IsZero()
}

func timeTypeAsString(v Value) (string, bool) {
	o := (*time.Time)(v.Ptr)
	return o.String(), true
}

func timeTypeAsInt(v Value) (int64, bool) {
	o := (*time.Time)(v.Ptr)
	return o.Unix(), true
}

func timeTypeAsBool(v Value) (bool, bool) {
	return timeTypeIsTrue(v), true
}

func timeTypeAsTime(v Value) (time.Time, bool) {
	o := (*time.Time)(v.Ptr)
	return *o, true
}

func timeTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	o := (*time.Time)(v.Ptr)

	if rhs.Type == VT_INT {
		r := int64(rhs.Data)
		switch op {
		case token.Add: // time + int => time
			d := a.NewTime()
			*d = o.Add(time.Duration(r))
			return TimeValue(d), nil
		case token.Sub: // time - int => time
			d := a.NewTime()
			*d = o.Add(time.Duration(-r))
			return TimeValue(d), nil
		}
	}

	r, ok := rhs.AsTime()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
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

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}
