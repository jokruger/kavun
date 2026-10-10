package core

import (
	"fmt"
	"time"

	"github.com/jokruger/fin128/civil"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// time's members: one function per entry of TypeTime.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func timeTime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), timeTypeName, args, true, v)
}

// PURE by contract
func timeInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	return convMember(id.String(), timeTypeName, args, true, IntValue(o.Unix()))
}

// PURE by contract
func timeFloat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f, ok := timeTypeAsFloat(v)
	return convMember(id.String(), timeTypeName, args, ok, FloatValue(f))
}

// PURE by contract
func timeComponents(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	// the constitutive parts only — the minimal set the instant can be rebuilt
	// from; computed accessors (week_day, month_name, zone_name) stay their own
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
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
}

// PURE by contract
func timeDecimal(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d, ok := timeTypeAsDecimal(v)
	return convMember(id.String(), timeTypeName, args, ok, NewDecimalValue(d))
}

// PURE by contract
func timeString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the ONE text form: RFC3339 with the fraction the instant carries
	s, ok := v.AsString()
	return convMember(id.String(), timeTypeName, args, ok, NewStringValue(s))
}

// PURE by contract
func timeRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	s, ok := v.AsString()
	return convMember(id.String(), timeTypeName, args, ok, NewRunesValue(DecodeText(s), false))
}

// PURE by contract
func timeYear(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Year())), nil
}

// PURE by contract
func timeMonth(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Month())), nil
}

// PURE by contract
func timeDay(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Day())), nil
}

// PURE by contract
func timeHour(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Hour())), nil
}

// PURE by contract
func timeMinute(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Minute())), nil
}

// PURE by contract
func timeSecond(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Second())), nil
}

// PURE by contract
func timeNanosecond(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Nanosecond())), nil
}

// PURE by contract
func timeUnix(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(o.Unix()), nil
}

// PURE by contract
func timeUnixMs(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(o.UnixMilli()), nil
}

// PURE by contract
func timeUnixMicro(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(o.UnixMicro()), nil
}

// PURE by contract
func timeUnixNano(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(o.UnixNano()), nil
}

// PURE by contract
func timeWeekDay(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.Weekday())), nil
}

// PURE by contract
func timeYearDay(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(o.YearDay())), nil
}

// PURE by contract
func timeMonthName(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(o.Month().String()), nil
}

// PURE by contract
func timeWeekDayName(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(o.Weekday().String()), nil
}

// PURE by contract
func timeUTC(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewTimeValue(o.UTC()), nil
}

// PURE by contract
func timeDate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the civil day the time reads in its own zone
	d, ok := timeTypeAsDate(v)
	return convMember(id.String(), timeTypeName, args, ok, DateValue(d))
}

// PURE by contract
func timeDateIn(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*time.Time)(v.Ptr)
	// the civil day in a named zone: ≡ t.in_zone(z).date()
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	loc, err := zoneArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	y, m, dd := o.In(loc).Date()
	d, err := newDate(y, int(m), dd)
	if err != nil {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
	}
	return DateValue(d), nil
}

// PURE by contract
func timeInZone(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*time.Time)(v.Ptr)
	// the same instant, viewed in a named zone
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	loc, err := zoneArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	return NewTimeValue(o.In(loc)), nil
}

// PURE by contract
func timeAddDays(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*time.Time)(v.Ptr)
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	t, err := timeShiftDays(*o, n)
	if err != nil {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
	}
	return NewTimeValue(t), nil
}

// PURE by contract
func timeAddMonths(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*time.Time)(v.Ptr)
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
	t, err := timeShiftMonths(*o, n, rule)
	if err != nil {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
	}
	return NewTimeValue(t), nil
}

// PURE by contract
func timeAddYears(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*time.Time)(v.Ptr)
	// n years is 12n months — checked first, so the multiplication cannot overflow
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
	if n < -1<<31/12 || n > (1<<31-1)/12 {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) result out of range", name))
	}
	t, err := timeShiftMonths(*o, n*12, rule)
	if err != nil {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, err))
	}
	return NewTimeValue(t), nil
}

// PURE by contract
func timeIsLeapYear(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(civil.IsLeapYear(o.Year())), nil
}

// PURE by contract
func timeDaysInYear(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(civil.DaysInYear(o.Year()))), nil
}

// PURE by contract
func timeDaysInMonth(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(civil.DaysInMonth(o.Year(), civil.Month(o.Month())))), nil
}

// PURE by contract
func timeZoneOffset(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	_, offset := o.Zone()
	return IntValue(int64(offset)), nil
}

// PURE by contract
func timeZoneName(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*time.Time)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	zone, _ := o.Zone()
	return NewStringValue(zone), nil
}
