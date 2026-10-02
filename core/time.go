package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128/civil"

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
// zero time's part (1970-01-01T00:00:00 UTC), so an empty map is the zero time; an UNKNOWN key raises, so a typo
// is an error rather than a silent default. The way back from t.components().
//
// `zone` names an IANA zone and makes the parts a wall clock in it; `zone_offset` alone is a fixed offset; with
// both, the offset picks between the two instants of a daylight-saving overlap and must match the zone. The
// wall clock goes through the checked builder: a part out of range, a clock a gap skips, or an overlap clock
// with no offset raises — nothing normalizes.
func TimeFromComponents(m map[string]Value) (time.Time, error) {
	get := func(key string, dflt int64) (int64, error) {
		v, ok := m[key]
		if !ok {
			return dflt, nil
		}
		i, ok := v.AsInt()
		if !ok || v.Type == value.Float || v.Type == value.Decimal {
			return 0, errs.NewInvalidArgumentTypeError("time", key, "int", v.TypeName())
		}
		return i, nil
	}
	// sorted so that a components map with SEVERAL unknown keys always names the same one: the error text
	// is part of the observable behaviour, and map order must not leak into it
	for _, k := range slices.Sorted(maps.Keys(m)) {
		switch k {
		case "year", "month", "day", "hour", "minute", "second", "nanosecond", "zone", "zone_offset":
		default:
			return time.Time{}, errs.NewInvalidValueError(fmt.Sprintf("(time) unknown component %q", k))
		}
	}
	var parts [7]int
	for i, kd := range []struct {
		key  string
		dflt int64
	}{{"year", 1970}, {"month", 1}, {"day", 1}, {"hour", 0}, {"minute", 0}, {"second", 0}, {"nanosecond", 0}} {
		n, err := get(kd.key, kd.dflt)
		if err != nil {
			return time.Time{}, err
		}
		if n < -1<<31 || n > 1<<31-1 {
			return time.Time{}, errs.NewInvalidValueError(fmt.Sprintf("(time) %s %d out of range", kd.key, n))
		}
		parts[i] = int(n)
	}
	var offset *int
	if _, ok := m["zone_offset"]; ok {
		n, err := get("zone_offset", 0)
		if err != nil {
			return time.Time{}, err
		}
		if n <= -24*3600 || n >= 24*3600 {
			return time.Time{}, errs.NewInvalidValueError(fmt.Sprintf("(time) zone_offset %d out of range", n))
		}
		o := int(n)
		offset = &o
	}
	loc := time.UTC
	if zv, ok := m["zone"]; ok {
		if zv.Type != value.String {
			return time.Time{}, errs.NewInvalidArgumentTypeError("time", "zone", "string", zv.TypeName())
		}
		z, _ := zv.AsString()
		l, err := LoadZone(z)
		if err != nil {
			return time.Time{}, errs.NewInvalidValueError("(time) " + err.Error())
		}
		loc = l
	} else if offset != nil {
		loc = fixedZone(*offset)
	}
	t, err := buildTime(parts[0], parts[1], parts[2], parts[3], parts[4], parts[5], parts[6], loc, offset)
	if err != nil {
		return time.Time{}, errs.NewInvalidValueError("(time) " + err.Error())
	}
	return t, nil
}

// Every time member is pure: zone data is read by name and is the host's tzdata, which the purity contract
// treats as fixed for a process (docs/purity.md); no member reads the host's own zone.
func timeTypeIsMethodPure(name string) bool {
	return true
}

// PURE by contract
func timeTypeInterface(v Value) any {
	return *(*time.Time)(v.Ptr)
}

// PURE by contract
func timeTypeIsTrue(v Value) (bool, error) {
	return !isZeroTime(*(*time.Time)(v.Ptr)), nil
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
		parts := map[string]Value{
			"year":        IntValue(int64(o.Year())),
			"month":       IntValue(int64(o.Month())),
			"day":         IntValue(int64(o.Day())),
			"hour":        IntValue(int64(o.Hour())),
			"minute":      IntValue(int64(o.Minute())),
			"second":      IntValue(int64(o.Second())),
			"nanosecond":  IntValue(int64(o.Nanosecond())),
			"zone_offset": IntValue(int64(off)),
		}
		// a named zone is part of the instant's identity (it decides every later wall clock); UTC and fixed
		// offsets are fully described by zone_offset
		if z := zoneNameOf(*o); z != "" {
			parts["zone"] = NewStringValue(z)
		}
		return NewRecordValue(parts, false), nil

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
			return Undefined, errs.FromFormatSpecError(name, err)
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

	case "in_zone":
		// the same instant, viewed in a named zone
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		loc, err := zoneArg(name, "first", args[0])
		if err != nil {
			return Undefined, err
		}
		return NewTimeValue(o.In(loc)), nil

	case "add_days":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		n, err := parseIntArg(name, "first", args[0])
		if err != nil {
			return Undefined, err
		}
		t, err := timeAddDays(*o, n)
		if err != nil {
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
		}
		return NewTimeValue(t), nil

	case "add_months", "add_years":
		if len(args) != 2 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
		}
		n, err := parseIntArg(name, "first", args[0])
		if err != nil {
			return Undefined, err
		}
		rule, err := EOMRuleArg(name, "second", args[1])
		if err != nil {
			return Undefined, err
		}
		if name == "add_years" {
			if n < -1<<31/12 || n > (1<<31-1)/12 {
				return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) result out of range", name))
			}
			n *= 12
		}
		t, err := timeAddMonths(*o, n, rule)
		if err != nil {
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
		}
		return NewTimeValue(t), nil

	case "is_leap_year":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(civil.IsLeapYear(o.Year())), nil

	case "days_in_year":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(civil.DaysInYear(o.Year()))), nil

	case "days_in_month":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(civil.DaysInMonth(o.Year(), civil.Month(o.Month())))), nil

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

// parseTimeText is the shared text -> time conversion behind string's, runes' and bytes' AsTime hooks: the
// canonical grammar only (ParseTimeText). Any other layout is named explicitly at the call site.
func parseTimeText(s string) (time.Time, bool) {
	t, err := ParseTimeText(s)
	return t, err == nil
}

// zoneArg reads a zone-name argument through LoadZone: a string naming an IANA zone or "UTC".
func zoneArg(name, pos string, a Value) (*time.Location, error) {
	if a.Type != value.String {
		return nil, errs.NewInvalidArgumentTypeError(name, pos, "string", a.TypeName())
	}
	z, _ := a.AsString()
	loc, err := LoadZone(z)
	if err != nil {
		return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
	}
	return loc, nil
}

// PURE by contract
func timeTypeAsTime(v Value) (time.Time, bool) {
	return *(*time.Time)(v.Ptr), true
}
