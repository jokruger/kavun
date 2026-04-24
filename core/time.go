package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
	"unsafe"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/token"
)

// TimeValue creates new boxed time value.
func TimeValue(v *Time) Value {
	return Value{
		Type:  VT_TIME,
		Const: true,
		Ptr:   unsafe.Pointer(v),
	}
}

// NewTimeValue creates new (heap-allocated) boxed time value.
func NewTimeValue(t Time) Value {
	o := &t
	return TimeValue(o)
}

/* Time type methods */

func timeTypeName(v Value) string {
	return "time"
}

func timeTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Time)(v.Ptr)
	y, err := o.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return y, nil
}

func timeTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Time)(v.Ptr)
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
	var t Time
	if err := dec.Decode(&t); err != nil {
		return fmt.Errorf("time: %w", err)
	}
	v.Ptr = unsafe.Pointer(&t)
	return nil
}

func timeTypeString(v Value) string {
	o := (*Time)(v.Ptr)
	return fmt.Sprintf("time(%q)", o.String())
}

func timeTypeInterface(v Value) any {
	o := (*Time)(v.Ptr)
	return *o
}

func timeTypeEqual(v Value, r Value) bool {
	t, ok := r.AsTime()
	if !ok {
		return false
	}
	o := (*Time)(v.Ptr)
	return o.Equal(t)
}

func timeTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*Time)(v.Ptr)

	switch name {
	case "to_time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "to_bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := timeTypeAsBool(v)
		return BoolValue(b), nil

	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := timeTypeAsInt(v)
		return IntValue(i), nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.String())

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
		return vm.Allocator().NewStringValue(o.Month().String())

	case "week_day_name":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.Weekday().String())

	case "to_utc":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, err := vm.Allocator().NewTime()
		if err != nil {
			return Undefined, err
		}
		*d = o.UTC()
		return TimeValue(d), nil

	case "to_local":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, err := vm.Allocator().NewTime()
		if err != nil {
			return Undefined, err
		}
		*d = o.Local()
		return TimeValue(d), nil

	case "format_date":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.Format(time.DateOnly))

	case "format_time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.Format(time.TimeOnly))

	case "format_datetime":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.Format(time.DateTime))

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
		return vm.Allocator().NewStringValue(name)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func timeTypeIsTrue(v Value) bool {
	o := (*Time)(v.Ptr)
	return !o.IsZero()
}

func timeTypeAsString(v Value) (string, bool) {
	o := (*Time)(v.Ptr)
	return o.String(), true
}

func timeTypeAsInt(v Value) (int64, bool) {
	o := (*Time)(v.Ptr)
	return o.Unix(), true
}

func timeTypeAsBool(v Value) (bool, bool) {
	return timeTypeIsTrue(v), true
}

func timeTypeAsTime(v Value) (Time, bool) {
	o := (*Time)(v.Ptr)
	return *o, true
}

func timeTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	o := (*Time)(v.Ptr)

	if rhs.Type == VT_INT {
		r := int64(rhs.Data)
		switch op {
		case token.Add: // time + int => time
			d, err := a.NewTime()
			if err != nil {
				return Undefined, err
			}
			*d = o.Add(time.Duration(r))
			return TimeValue(d), nil
		case token.Sub: // time - int => time
			d, err := a.NewTime()
			if err != nil {
				return Undefined, err
			}
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
