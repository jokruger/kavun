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

func (o *Time) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	alloc := vm.Allocator()

	if rhs, ok := rhs.(*Int); ok {
		switch op {
		case token.Add: // time + int => time
			return alloc.NewTime(o.value.Add(time.Duration(rhs.value))), nil
		case token.Sub: // time - int => time
			return alloc.NewTime(o.value.Add(time.Duration(-rhs.value))), nil
		}
	}

	v, ok := rhs.AsTime()
	if !ok {
		return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
	}

	switch op {
	case token.Sub: // time - time => int (duration)
		return alloc.NewInt(int64(o.value.Sub(v))), nil
	case token.Less: // time < time => bool
		return alloc.NewBool(o.value.Before(v)), nil
	case token.Greater:
		return alloc.NewBool(o.value.After(v)), nil
	case token.LessEq:
		return alloc.NewBool(o.value.Equal(v) || o.value.Before(v)), nil
	case token.GreaterEq:
		return alloc.NewBool(o.value.Equal(v) || o.value.After(v)), nil
	}

	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Time) Equals(x core.Object) bool {
	t, ok := x.AsTime()
	if !ok {
		return false
	}
	return o.value.Equal(t)
}

func (o *Time) Copy(alloc core.Allocator) core.Object {
	return alloc.NewTime(o.value)
}

func (o *Time) Access(vm core.VM, index core.Object, op core.Opcode) (core.Object, error) {
	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("map access", "string", index)
	}

	alloc := vm.Allocator()
	switch k {
	case "year":
		return alloc.NewInt(int64(o.value.Year())), nil

	case "month":
		return alloc.NewInt(int64(o.value.Month())), nil

	case "day":
		return alloc.NewInt(int64(o.value.Day())), nil

	case "hour":
		return alloc.NewInt(int64(o.value.Hour())), nil

	case "minute":
		return alloc.NewInt(int64(o.value.Minute())), nil

	case "second":
		return alloc.NewInt(int64(o.value.Second())), nil

	case "nanosecond":
		return alloc.NewInt(int64(o.value.Nanosecond())), nil

	case "unix":
		return alloc.NewInt(o.value.Unix()), nil

	case "unix_nano":
		return alloc.NewInt(o.value.UnixNano()), nil

	case "week_day":
		return alloc.NewInt(int64(o.value.Weekday())), nil

	case "year_day":
		return alloc.NewInt(int64(o.value.YearDay())), nil

	case "month_name":
		return alloc.NewString(o.value.Month().String()), nil

	case "week_day_name":
		return alloc.NewString(o.value.Weekday().String()), nil

	case "utc":
		return alloc.NewTime(o.value.UTC()), nil

	case "local":
		return alloc.NewTime(o.value.Local()), nil

	case "str":
		return alloc.NewString(o.value.String()), nil

	case "date_str":
		return alloc.NewString(o.value.Format(time.DateOnly)), nil

	case "time_str":
		return alloc.NewString(o.value.Format(time.TimeOnly)), nil

	case "date_time_str":
		return alloc.NewString(o.value.Format(time.DateTime)), nil

	case "zone_offset":
		_, offset := o.value.Zone()
		return alloc.NewInt(int64(offset)), nil

	case "zone_name":
		name, _ := o.value.Zone()
		return alloc.NewString(name), nil

	default:
		return nil, core.NewInvalidSelectorError(o, k)
	}

}

func (o *Time) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Time) IsTrue() bool {
	return !o.IsFalse()
}

func (o *Time) IsFalse() bool {
	return o.value.IsZero()
}

func (o *Time) IsImmutable() bool {
	return true
}

func (o *Time) AsString() (string, bool) {
	return o.value.String(), true
}

func (o *Time) AsInt() (int64, bool) {
	return o.value.Unix(), false
}

func (o *Time) AsBool() (bool, bool) {
	return !o.IsFalse(), true
}

func (o *Time) AsTime() (time.Time, bool) {
	return o.value, true
}
