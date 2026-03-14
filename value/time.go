package value

import (
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type Time struct {
	ObjectImpl
	Value time.Time
}

func (o *Time) TypeName() string {
	return "time"
}

func (o *Time) String() string {
	return o.Value.String()
}

func (o *Time) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	switch rhs := rhs.(type) {
	case *Int:
		switch op {
		case token.Add: // time + int => time
			if rhs.Value == 0 {
				return o, nil
			}
			return &Time{Value: o.Value.Add(time.Duration(rhs.Value))}, nil
		case token.Sub: // time - int => time
			if rhs.Value == 0 {
				return o, nil
			}
			return &Time{Value: o.Value.Add(time.Duration(-rhs.Value))}, nil
		}
	case *Time:
		switch op {
		case token.Sub: // time - time => int (duration)
			return &Int{Value: int64(o.Value.Sub(rhs.Value))}, nil
		case token.Less: // time < time => bool
			if o.Value.Before(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.Value.After(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.Value.Equal(rhs.Value) || o.Value.Before(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.Value.Equal(rhs.Value) || o.Value.After(rhs.Value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, gse.ErrInvalidOperator
}

func (o *Time) Copy() core.Object {
	return &Time{Value: o.Value}
}

func (o *Time) IsFalsy() bool {
	return o.Value.IsZero()
}

func (o *Time) Equals(x core.Object) bool {
	t, ok := x.(*Time)
	if !ok {
		return false
	}
	return o.Value.Equal(t.Value)
}

func (o *Time) ToString() (string, bool) {
	return o.String(), true
}

func (o *Time) ToBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Time) ToInt() (int, bool) {
	return int(o.Value.Unix()), false
}

func (o *Time) ToInt64() (int64, bool) {
	return o.Value.Unix(), false
}

func (o *Time) ToTime() (time.Time, bool) {
	return o.Value, true
}

func (o *Time) ToInterface() any {
	return o.Value
}
