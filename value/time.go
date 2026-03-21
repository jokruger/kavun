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

func NewTime(t time.Time) *Time {
	o := &Time{}
	o.Set(t)
	return o
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
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
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

func (o *Time) Access(core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *Time) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Time) IsFalsy() bool {
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
	return !o.IsFalsy(), true
}

func (o *Time) AsTime() (time.Time, bool) {
	return o.value, true
}
