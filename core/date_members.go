package core

import (
	"fmt"

	"github.com/jokruger/fin128/civil"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// date's members: one function per entry of TypeDate.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func dateDate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), dateTypeName, args, true, v)
}

// PURE by contract
func dateInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	return convMember(id.String(), dateTypeName, args, true, IntValue(int64(d.Days())))
}

// PURE by contract
func dateString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	return convMember(id.String(), dateTypeName, args, true, NewStringValue(d.String()))
}

// PURE by contract
func dateRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	return convMember(id.String(), dateTypeName, args, true, NewRunesValue(DecodeText(d.String()), false))
}

// PURE by contract
func dateTime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	// midnight UTC — the date's own reading; time_in(zone) is the start of the day elsewhere
	return convMember(id.String(), dateTypeName, args, true, NewTimeValue(dateMidnightUTC(d)))
}

// PURE by contract
func dateComponents(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	y, m, dd := d.YMD()
	return NewRecordValue(map[string]Value{
		"year":  IntValue(int64(y)),
		"month": IntValue(int64(m)),
		"day":   IntValue(int64(dd)),
	}, false), nil
}

// PURE by contract
func dateYear(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(d.Year())), nil
}

// PURE by contract
func dateMonth(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(d.Month())), nil
}

// PURE by contract
func dateDay(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(d.Day())), nil
}

// PURE by contract
func dateWeekDay(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(d.Weekday())), nil
}

// PURE by contract
func dateYearDay(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(d.DayOfYear())), nil
}

// PURE by contract
func dateMonthName(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(monthNames[d.Month()-1]), nil
}

// PURE by contract
func dateWeekDayName(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(weekdayNames[d.Weekday()]), nil

	// month shape and calendar facts
}

// PURE by contract
func dateStartOfMonth(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return DateValue(d.StartOfMonth()), nil
}

// PURE by contract
func dateEndOfMonth(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return DateValue(d.EndOfMonth()), nil
}

// PURE by contract
func dateIsEndOfMonth(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(d.IsEndOfMonth()), nil
}

// PURE by contract
func dateIsLeapYear(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(civil.IsLeapYear(d.Year())), nil
}

// PURE by contract
func dateDaysInYear(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(civil.DaysInYear(d.Year()))), nil
}

// PURE by contract
func dateDaysInMonth(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d := dateOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(civil.DaysInMonth(d.Year(), d.Month()))), nil

	// arithmetic
}

// PURE by contract
func dateAddDays(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	d := dateOf(v)
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	return dateShiftDays(d, n)
}

// PURE by contract
func dateAddMonths(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	d := dateOf(v)
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
	if n < -1<<31 || n > 1<<31-1 {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) result out of range", name))
	}
	r, ok := d.AddMonths(int32(n), rule)
	if !ok {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) result out of range", name))
	}
	return DateValue(r), nil
}

// PURE by contract
func dateAddYears(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	d := dateOf(v)
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
	if n < -1<<31 || n > 1<<31-1 {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) result out of range", name))
	}
	r, ok := d.AddYears(int32(n), rule)
	if !ok {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) result out of range", name))
	}
	return DateValue(r), nil
}

// PURE by contract
func dateMonthsSince(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	d := dateOf(v)
	// [months, days]: whole months from other to d, and the days left over, so that
	// other.add_months(months, eom) + days == d
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	if args[0].Type != value.Date {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "date", args[0].TypeName())
	}
	rule, err := EOMRuleArg(name, "second", args[1])
	if err != nil {
		return Undefined, err
	}
	months, days, ok := d.MonthsSince(dateOf(args[0]), rule)
	if !ok {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) result out of range", name))
	}
	return NewArrayValue([]Value{IntValue(int64(months)), IntValue(int64(days))}, false), nil

	// bridge to time
}

// PURE by contract
func dateTimeIn(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	d := dateOf(v)
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	loc, err := zoneArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	return NewTimeValue(dateStartIn(d, loc)), nil
}
