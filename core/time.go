package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/araddon/dateparse"
	"github.com/jokruger/dec128"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const timeTypeName = "time"

func NewStaticTimeValue(t *time.Time) Value {
	return Value{Type: value.Time, Immutable: true, Ptr: unsafe.Pointer(t)}
}

func NewTimeValue(t time.Time) Value {
	return Value{Type: value.Time, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

// TypeTime is a time type descriptor.
var TypeTime = ValueTypeDescr{
	Name:         ConstHook(timeTypeName), // PURE by contract
	String:       timeTypeString,          // PURE by contract
	Format:       timeTypeFormat,          // PURE by contract
	Interface:    timeTypeInterface,       // PURE by contract
	EncodeJSON:   timeTypeEncodeJSON,      // PURE by contract
	EncodeBinary: timeTypeEncodeBinary,    // PURE by contract
	DecodeBinary: timeTypeDecodeBinary,    // IMPURE by contract (mutates target)
	IsTrue:       timeTypeIsTrue,          // PURE by contract
	Len:          ConstHook(int64(1)),     // PURE by contract
	Equal:        timeTypeEqual,           // PURE by contract
	BinaryOp:     timeTypeBinaryOp,        // PURE by contract
	MethodCall:   timeTypeMethodCall,      // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	AsString:     timeTypeAsString,        // PURE by contract
	AsInt:        timeTypeAsInt,           // PURE by contract
	AsFloat:      timeTypeAsFloat,         // PURE by contract
	AsDecimal:    timeTypeAsDecimal,       // PURE by contract
	AsTime:       timeTypeAsTime,          // PURE by contract
	IsMethodPure: timeTypeIsMethodPure,
}

// TimeFromComponents rebuilds an instant from its constitutive parts. Every key is optional and defaults to the
// zero time's part, so an empty map is the zero time; an UNKNOWN key raises, so a typo is an error rather than
// silently year 1. The way back from t.components().
func TimeFromComponents(m map[string]Value) (time.Time, error) {
	get := func(key string, dflt int64) (int64, error) {
		v, ok := m[key]
		if !ok {
			return dflt, nil
		}
		i, ok := v.AsInt()
		if !ok {
			return 0, errs.NewInvalidArgumentTypeError("time", key, "int", v.TypeName())
		}
		return i, nil
	}
	for k := range m {
		switch k {
		case "year", "month", "day", "hour", "minute", "second", "nanosecond", "zone_offset":
		default:
			return time.Time{}, errs.NewInvalidValueError(fmt.Sprintf("(time) unknown component %q", k))
		}
	}
	year, err := get("year", 1)
	if err != nil {
		return time.Time{}, err
	}
	month, err := get("month", 1)
	if err != nil {
		return time.Time{}, err
	}
	day, err := get("day", 1)
	if err != nil {
		return time.Time{}, err
	}
	hour, err := get("hour", 0)
	if err != nil {
		return time.Time{}, err
	}
	minute, err := get("minute", 0)
	if err != nil {
		return time.Time{}, err
	}
	second, err := get("second", 0)
	if err != nil {
		return time.Time{}, err
	}
	nanosecond, err := get("nanosecond", 0)
	if err != nil {
		return time.Time{}, err
	}
	zoneOffset, err := get("zone_offset", 0)
	if err != nil {
		return time.Time{}, err
	}
	loc := time.UTC
	if zoneOffset != 0 {
		loc = time.FixedZone("", int(zoneOffset))
	}
	return time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(second), int(nanosecond), loc), nil
}

func timeTypeIsMethodPure(name string) bool {
	switch name {
	case "local": // IMPURE because it depends on the system's local timezone
		return false
	default:
		return true
	}
}

// PURE by contract
func timeTypeInterface(v Value) any {
	return *(*time.Time)(v.Ptr)
}

// PURE by contract
func timeTypeIsTrue(v Value) (bool, error) {
	return !(*time.Time)(v.Ptr).IsZero(), nil
}

// PURE by contract
func timeTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*time.Time)(v.Ptr)
	y, err := o.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return y, nil
}

// PURE by contract
func timeTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*time.Time)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(*o); err != nil {
		return nil, fmt.Errorf("time: %w", err)
	}
	return buf.Bytes(), nil
}

// IMPURE by contract (mutates target)
func timeTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var t time.Time
	if err := dec.Decode(&t); err != nil {
		return fmt.Errorf("time: %w", err)
	}
	*v = NewTimeValue(t)
	return nil
}

// PURE by contract
func timeTypeString(v Value) string {
	o := (*time.Time)(v.Ptr)
	return fmt.Sprintf("time(%q)", o.Format(time.RFC3339Nano))
}

// PURE by contract
func timeTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return timeTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(timeTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.HasPrec || sp.ZeroPad || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	t := (*time.Time)(v.Ptr)

	var body string
	switch sp.Verb {
	case 0:
		// the default render is precision-preserving: the fraction appears iff
		// the instant carries one; #iso is the explicitly seconds-truncating spec
		body = t.Format(time.RFC3339Nano)

	case '#':
		switch sp.Tail {
		case "":
			body = t.Format(time.RFC3339Nano)
		case "iso":
			body = t.Format(time.RFC3339)
		case "isonano":
			body = t.Format(time.RFC3339Nano)
		case "date":
			body = t.Format("2006-01-02")
		case "time":
			body = t.Format("15:04:05")
		case "datetime":
			body = t.Format(time.DateTime)
		case "unix":
			body = strconv.FormatInt(t.Unix(), 10)
		case "unixms":
			body = strconv.FormatInt(t.UnixMilli(), 10)
		case "unixmicro":
			body = strconv.FormatInt(t.UnixMicro(), 10)
		case "unixnano":
			body = strconv.FormatInt(t.UnixNano(), 10)
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
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
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
			return "", errs.NewFormattingError(fmt.Sprintf("time: trailing '%%' in format %q", layout))
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
			return "", errs.NewFormattingError(fmt.Sprintf("time: unknown strftime directive %%%c in %q", layout[i], layout))
		}
	}
	return b.String(), nil
}

// PURE by contract.
func timeTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Time:
		o := (*time.Time)(v.Ptr)
		r := (*time.Time)(other.Ptr)
		return o.Equal(*r)
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

func timeTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Int:
			switch op {
			case token.Add:
				l := int64(other.Data)
				r := (*time.Time)(v.Ptr)
				return NewTimeValue(r.Add(time.Duration(l))), nil
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Time:
		l := *(*time.Time)(v.Ptr)
		r := *(*time.Time)(other.Ptr)
		switch op {
		case token.Sub:
			return IntValue(int64(l.Sub(r))), nil
		case token.Less:
			return BoolValue(l.Before(r)), nil
		case token.Greater:
			return BoolValue(l.After(r)), nil
		case token.LessEq:
			return BoolValue(l.Equal(r) || l.Before(r)), nil
		case token.GreaterEq:
			return BoolValue(l.Equal(r) || l.After(r)), nil
		}

	case value.Int:
		switch op {
		case token.Add:
			l := (*time.Time)(v.Ptr)
			r := int64(other.Data)
			return NewTimeValue(l.Add(time.Duration(r))), nil
		case token.Sub:
			l := (*time.Time)(v.Ptr)
			r := int64(other.Data)
			return NewTimeValue(l.Add(time.Duration(-r))), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func timeTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value regardless of copy depth
		return v, nil

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable already, so freeze/freeze_shallow are no-ops
		return v, nil

	case "time":
		return convMember(name, timeTypeName, args, true, v)

	case "int":
		return convMember(name, timeTypeName, args, true, IntValue(o.Unix()))

	case "float":
		f, ok := timeTypeAsFloat(v)
		return convMember(name, timeTypeName, args, ok, FloatValue(f))

	case "components":
		// the constitutive parts only — the minimal set the instant can be rebuilt
		// from; computed accessors (week_day, month_name, zone_name) stay their own
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		_, off := o.Zone()
		return NewRecordValue(map[string]Value{
			"year":        IntValue(int64(o.Year())),
			"month":       IntValue(int64(o.Month())),
			"day":         IntValue(int64(o.Day())),
			"hour":        IntValue(int64(o.Hour())),
			"minute":      IntValue(int64(o.Minute())),
			"second":      IntValue(int64(o.Second())),
			"nanosecond":  IntValue(int64(o.Nanosecond())),
			"zone_offset": IntValue(int64(off)),
		}, false), nil

	case "decimal":
		d, ok := timeTypeAsDecimal(v)
		return convMember(name, timeTypeName, args, ok, NewDecimalValue(d))

	case "string":
		// the ONE text form: RFC3339 with the fraction the instant carries
		s, ok := v.AsString()
		return convMember(name, timeTypeName, args, ok, NewStringValue(s))

	case "runes":
		s, ok := v.AsString()
		return convMember(name, timeTypeName, args, ok, NewRunesValue([]rune(s), false))

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
		return NewStringValue(s), nil

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

	case "unix_ms":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(o.UnixMilli()), nil

	case "unix_micro":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(o.UnixMicro()), nil

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
		return NewStringValue(o.Month().String()), nil

	case "week_day_name":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(o.Weekday().String()), nil

	case "utc":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewTimeValue(o.UTC()), nil

	case "local":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewTimeValue(o.Local()), nil

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
		return NewStringValue(name), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

// PURE by contract
func timeTypeAsString(v Value) (string, bool) {
	// ONE text form: RFC3339 with the fraction the instant carries — the same
	// text format() and f-strings produce, and it round-trips through the parse
	return (*time.Time)(v.Ptr).Format(time.RFC3339Nano), true
}

// PURE by contract
func timeTypeAsInt(v Value) (int64, bool) {
	return (*time.Time)(v.Ptr).Unix(), true
}

// an instant as a number is unix sec.frac; the float form is approximate
// (~100ns at present-day magnitudes), the decimal form exact to the nanosecond
func timeTypeAsFloat(v Value) (float64, bool) {
	o := (*time.Time)(v.Ptr)
	return float64(o.UnixNano()) / 1e9, true
}

func timeTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	o := (*time.Time)(v.Ptr)
	d := dec128.FromInt64(o.UnixNano()).Div(dec128.FromInt64(1e9))
	return d, !d.IsNaN()
}

// parseTimeText is the shared text -> time parse used by string's and runes' AsTime hooks.
//
// dateparse resolves a bare numeric string (a unix timestamp, whose unit it infers from the digit
// count) through time.Unix, which yields the host's LOCAL zone -- the only construction path in the
// language that does. The instant is correct either way, but the wall-clock accessors are not:
// time("1704067200").hour() would differ per machine. Normalizing that one case to UTC keeps every
// int-shaped conversion host-independent, matching int/float/decimal's own AsTime hooks.
//
// Textual forms are deliberately left alone: dateparse already returns UTC for a zoneless one, and
// the stated offset for a zoned one -- that offset is data the caller wrote, not a default to
// normalize away.
func parseTimeText(s string) (time.Time, bool) {
	t, err := dateparse.ParseAny(s)
	if err != nil {
		return time.Time{}, false
	}
	if isAllDigits(s) {
		return t.UTC(), true
	}
	return t, true
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// PURE by contract
func timeTypeAsTime(v Value) (time.Time, bool) {
	return *(*time.Time)(v.Ptr), true
}
