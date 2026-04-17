package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
	"unsafe"

	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/token"
)

// TimeValue creates new boxed time value.
func TimeValue(v *time.Time) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_TIME,
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
	return fmt.Sprintf("time(%q)", o.String())
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

func timeTypeCopy(v Value, a Allocator) (Value, error) {
	o := (*time.Time)(v.Ptr)
	return a.NewTimeValue(*o)
}

func timeTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.to_time", "0", len(args))
		}
		return v, nil

	case "to_bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.to_bool", "0", len(args))
		}
		b, _ := timeTypeAsBool(v)
		return BoolValue(b), nil

	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.to_int", "0", len(args))
		}
		i, _ := timeTypeAsInt(v)
		return IntValue(i), nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.to_string", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return vm.Allocator().NewStringValue(o.String())

	case "year":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.year", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.Year())), nil

	case "month":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.month", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.Month())), nil

	case "day":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.day", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.Day())), nil

	case "hour":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.hour", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.Hour())), nil

	case "minute":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.minute", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.Minute())), nil

	case "second":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.second", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.Second())), nil

	case "nanosecond":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.nanosecond", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.Nanosecond())), nil

	case "unix":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.unix", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(o.Unix()), nil

	case "unix_nano":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.unix_nano", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(o.UnixNano()), nil

	case "week_day":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.week_day", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.Weekday())), nil

	case "year_day":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.year_day", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.YearDay())), nil

	case "month_name":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.month_name", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return vm.Allocator().NewStringValue(o.Month().String())

	case "week_day_name":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.week_day_name", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return vm.Allocator().NewStringValue(o.Weekday().String())

	case "to_utc":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.to_utc", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return vm.Allocator().NewTimeValue(o.UTC())

	case "to_local":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.to_local", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return vm.Allocator().NewTimeValue(o.Local())

	case "to_date_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.to_date_string", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return vm.Allocator().NewStringValue(o.Format(time.DateOnly))

	case "to_time_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.to_time_string", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return vm.Allocator().NewStringValue(o.Format(time.TimeOnly))

	case "to_date_time_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.to_date_time_string", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		return vm.Allocator().NewStringValue(o.Format(time.DateTime))

	case "zone_offset":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.zone_offset", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		_, offset := o.Zone()
		return IntValue(int64(offset)), nil

	case "zone_name":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("time.zone_name", "0", len(args))
		}
		o := (*time.Time)(v.Ptr)
		name, _ := o.Zone()
		return vm.Allocator().NewStringValue(name)

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

func timeTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	if rhs.Type == VT_INT {
		r := int64(rhs.Data)
		switch op {
		case token.Add: // time + int => time
			o := (*time.Time)(v.Ptr)
			return a.NewTimeValue(o.Add(time.Duration(r)))
		case token.Sub: // time - int => time
			o := (*time.Time)(v.Ptr)
			return a.NewTimeValue(o.Add(time.Duration(-r)))
		}
	}

	r, ok := rhs.AsTime()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	switch op {
	case token.Sub: // time - time => int (duration)
		o := (*time.Time)(v.Ptr)
		return IntValue(int64(o.Sub(r))), nil
	case token.Less: // time < time => bool
		o := (*time.Time)(v.Ptr)
		return BoolValue(o.Before(r)), nil
	case token.Greater:
		o := (*time.Time)(v.Ptr)
		return BoolValue(o.After(r)), nil
	case token.LessEq:
		o := (*time.Time)(v.Ptr)
		return BoolValue(o.Equal(r) || o.Before(r)), nil
	case token.GreaterEq:
		o := (*time.Time)(v.Ptr)
		return BoolValue(o.Equal(r) || o.After(r)), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}
