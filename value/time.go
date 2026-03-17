package value

import (
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type Time struct {
	value time.Time
}

func NewTime(t time.Time) *Time {
	o := &Time{}
	o.Set(t)
	return o
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
	return o.value.String()
}

func (o *Time) Interface() any {
	return o.value
}

func (o *Time) Arity() int {
	return 0
}

func (o *Time) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	switch rhs := rhs.(type) {
	case *Int:
		switch op {
		case token.Add: // time + int => time
			if rhs.value == 0 {
				return o, nil
			}
			return NewTime(o.value.Add(time.Duration(rhs.value))), nil
		case token.Sub: // time - int => time
			if rhs.value == 0 {
				return o, nil
			}
			return NewTime(o.value.Add(time.Duration(-rhs.value))), nil
		}
	case *Time:
		switch op {
		case token.Sub: // time - time => int (duration)
			return NewInt(int64(o.value.Sub(rhs.value))), nil
		case token.Less: // time < time => bool
			if o.value.Before(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.value.After(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.value.Equal(rhs.value) || o.value.Before(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.value.Equal(rhs.value) || o.value.After(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, gse.ErrInvalidOperator
}

func (o *Time) Equals(x core.Object) bool {
	if o == x {
		return true
	}

	t, ok := x.AsTime()
	if !ok {
		return false
	}
	return o.value.Equal(t)
}

func (o *Time) Copy() core.Object {
	return NewTime(o.value)
}

func (o *Time) IndexGet(core.Object) (core.Object, error) {
	return nil, gse.ErrNotIndexable
}

func (o *Time) IndexSet(core.Object, core.Object) error {
	return gse.ErrNotIndexAssignable
}

func (o *Time) Iterate() core.Iterator {
	return nil
}

func (o *Time) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Time) IsFalsy() bool {
	return o.value.IsZero()
}

func (o *Time) IsIterable() bool {
	return false
}

func (o *Time) IsCallable() bool {
	return false
}

func (o *Time) IsImmutable() bool {
	return false
}

func (o *Time) IsVariadic() bool {
	return false
}

func (o *Time) AsString() (string, bool) {
	return o.String(), true
}

func (o *Time) AsInt() (int64, bool) {
	return o.value.Unix(), false
}

func (o *Time) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Time) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Time) AsRune() (rune, bool) {
	return 0, false
}

func (o *Time) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Time) AsTime() (time.Time, bool) {
	return o.value, true
}
