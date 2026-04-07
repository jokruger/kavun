package value

import (
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Time struct {
	Object
	value time.Time
}

func (o *Time) GobDecode(b []byte) error {
	var t time.Time
	if err := t.GobDecode(b); err != nil {
		return err
	}
	o.Set(t)
	return nil
}

func (o *Time) GobEncode() ([]byte, error) {
	return o.value.GobEncode()
}

func (o *Time) Set(t time.Time) {
	o.value = t
}

func (o *Time) Value() time.Time {
	return o.value
}

func (o *Time) TypeName() string {
	return "time"
}

func (o *Time) String() string {
	return fmt.Sprintf("time(%q)", o.value.String())
}

func (o *Time) Interface() any {
	return o.value
}

func (o *Time) BinaryOp(vm core.VM, op token.Token, rhs core.Value) (core.Value, error) {
	alloc := vm.Allocator()

	if rhs.IsInt() {
		r := rhs.Int()
		switch op {
		case token.Add: // time + int => time
			return alloc.NewTimeValue(o.value.Add(time.Duration(r))), nil
		case token.Sub: // time - int => time
			return alloc.NewTimeValue(o.value.Add(time.Duration(-r))), nil
		}
	}

	v, ok := rhs.AsTime()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
	}

	switch op {
	case token.Sub: // time - time => int (duration)
		return core.IntValue(int64(o.value.Sub(v))), nil
	case token.Less: // time < time => bool
		return core.BoolValue(o.value.Before(v)), nil
	case token.Greater:
		return core.BoolValue(o.value.After(v)), nil
	case token.LessEq:
		return core.BoolValue(o.value.Equal(v) || o.value.Before(v)), nil
	case token.GreaterEq:
		return core.BoolValue(o.value.Equal(v) || o.value.After(v)), nil
	}

	return core.UndefinedValue(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
}

func (o *Time) Equals(x core.Value) bool {
	t, ok := x.AsTime()
	if !ok {
		return false
	}
	return o.value.Equal(t)
}

func (o *Time) Copy(alloc core.Allocator) core.Value {
	return alloc.NewTimeValue(o.value)
}

func (o *Time) Method(vm core.VM, name string, args []core.Value) (core.Value, error) {
	switch name {
	case "to_time":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.to_time", "0", len(args))
		}
		return core.ObjectValue(o), nil

	case "to_bool":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.to_bool", "0", len(args))
		}
		return core.BoolValue(o.IsTrue()), nil

	case "to_int":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.to_int", "0", len(args))
		}
		return core.IntValue(o.value.Unix()), nil

	case "to_string":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.to_string", "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.value.String()), nil

	case "year":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.year", "0", len(args))
		}
		return core.IntValue(int64(o.value.Year())), nil

	case "month":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.month", "0", len(args))
		}
		return core.IntValue(int64(o.value.Month())), nil

	case "day":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.day", "0", len(args))
		}
		return core.IntValue(int64(o.value.Day())), nil

	case "hour":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.hour", "0", len(args))
		}
		return core.IntValue(int64(o.value.Hour())), nil

	case "minute":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.minute", "0", len(args))
		}
		return core.IntValue(int64(o.value.Minute())), nil

	case "second":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.second", "0", len(args))
		}
		return core.IntValue(int64(o.value.Second())), nil

	case "nanosecond":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.nanosecond", "0", len(args))
		}
		return core.IntValue(int64(o.value.Nanosecond())), nil

	case "unix":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.unix", "0", len(args))
		}
		return core.IntValue(o.value.Unix()), nil

	case "unix_nano":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.unix_nano", "0", len(args))
		}
		return core.IntValue(o.value.UnixNano()), nil

	case "week_day":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.week_day", "0", len(args))
		}
		return core.IntValue(int64(o.value.Weekday())), nil

	case "year_day":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.year_day", "0", len(args))
		}
		return core.IntValue(int64(o.value.YearDay())), nil

	case "month_name":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.month_name", "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.value.Month().String()), nil

	case "week_day_name":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.week_day_name", "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.value.Weekday().String()), nil

	case "to_utc":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.to_utc", "0", len(args))
		}
		return vm.Allocator().NewTimeValue(o.value.UTC()), nil

	case "to_local":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.to_local", "0", len(args))
		}
		return vm.Allocator().NewTimeValue(o.value.Local()), nil

	case "to_date_string":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.to_date_string", "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.value.Format(time.DateOnly)), nil

	case "to_time_string":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.to_time_string", "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.value.Format(time.TimeOnly)), nil

	case "to_date_time_string":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.to_date_time_string", "0", len(args))
		}
		return vm.Allocator().NewStringValue(o.value.Format(time.DateTime)), nil

	case "zone_offset":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.zone_offset", "0", len(args))
		}
		_, offset := o.value.Zone()
		return core.IntValue(int64(offset)), nil

	case "zone_name":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("time.zone_name", "0", len(args))
		}
		name, _ := o.value.Zone()
		return vm.Allocator().NewStringValue(name), nil

	default:
		return core.UndefinedValue(), core.NewInvalidMethodError(name, o.TypeName())
	}
}

func (o *Time) Access(vm core.VM, index core.Value, op core.Opcode) (core.Value, error) {
	k, ok := index.AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidIndexTypeError("map access", "string", index.TypeName())
	}

	return core.UndefinedValue(), core.NewInvalidSelectorError(o.TypeName(), k)
}

func (o *Time) Assign(core.Value, core.Value) error {
	return core.NewNotAssignableError(o.TypeName())
}

func (o *Time) IsTime() bool {
	return true
}

func (o *Time) IsTrue() bool {
	return !o.IsFalse()
}

func (o *Time) IsFalse() bool {
	return o.value.IsZero()
}

func (o *Time) AsString() (string, bool) {
	return o.value.String(), true
}

func (o *Time) AsInt() (int64, bool) {
	return o.value.Unix(), true
}

func (o *Time) AsBool() (bool, bool) {
	return !o.IsFalse(), true
}

func (o *Time) AsTime() (time.Time, bool) {
	return o.value, true
}
