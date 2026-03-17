package value

import (
	"math"
	"strconv"
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type Float struct {
	value float64
}

func NewFloat(value float64) *Float {
	o := &Float{}
	o.Set(value)
	return o
}

func (o *Float) Set(value float64) {
	o.value = value
}

func (o *Float) Value() float64 {
	return o.value
}

func (o *Float) TypeName() string {
	return "float"
}

func (o *Float) String() string {
	return strconv.FormatFloat(o.value, 'f', -1, 64)
}

func (o *Float) Interface() any {
	return o.value
}

func (o *Float) Arity() int {
	return 0
}

func (o *Float) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	switch rhs := rhs.(type) {
	case *Float:
		switch op {
		case token.Add:
			r := o.value + rhs.value
			if r == o.value {
				return o, nil
			}
			return NewFloat(r), nil
		case token.Sub:
			r := o.value - rhs.value
			if r == o.value {
				return o, nil
			}
			return NewFloat(r), nil
		case token.Mul:
			r := o.value * rhs.value
			if r == o.value {
				return o, nil
			}
			return NewFloat(r), nil
		case token.Quo:
			r := o.value / rhs.value
			if r == o.value {
				return o, nil
			}
			return NewFloat(r), nil
		case token.Less:
			if o.value < rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.value > rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.value <= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.value >= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case *Int:
		switch op {
		case token.Add:
			r := o.value + float64(rhs.value)
			if r == o.value {
				return o, nil
			}
			return NewFloat(r), nil
		case token.Sub:
			r := o.value - float64(rhs.value)
			if r == o.value {
				return o, nil
			}
			return NewFloat(r), nil
		case token.Mul:
			r := o.value * float64(rhs.value)
			if r == o.value {
				return o, nil
			}
			return NewFloat(r), nil
		case token.Quo:
			r := o.value / float64(rhs.value)
			if r == o.value {
				return o, nil
			}
			return NewFloat(r), nil
		case token.Less:
			if o.value < float64(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.value > float64(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.value <= float64(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.value >= float64(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, gse.ErrInvalidOperator
}

func (o *Float) Equals(x core.Object) bool {
	t, ok := x.AsFloat()
	if !ok {
		return false
	}
	return o.value == t
}

func (o *Float) Copy() core.Object {
	return NewFloat(o.value)
}

func (o *Float) IndexGet(core.Object) (core.Object, error) {
	return nil, gse.ErrNotIndexable
}

func (o *Float) IndexSet(core.Object, core.Object) error {
	return gse.ErrNotIndexAssignable
}

func (o *Float) Iterate() core.Iterator {
	return nil
}

func (o *Float) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Float) IsFalsy() bool {
	return math.IsNaN(o.value)
}

func (o *Float) IsIterable() bool {
	return false
}

func (o *Float) IsCallable() bool {
	return false
}

func (o *Float) IsImmutable() bool {
	return false
}

func (o *Float) IsVariadic() bool {
	return false
}

func (o *Float) AsString() (string, bool) {
	return o.String(), true
}

func (o *Float) AsInt() (int64, bool) {
	return int64(o.value), true
}

func (o *Float) AsFloat() (float64, bool) {
	return o.value, true
}

func (o *Float) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Float) AsRune() (rune, bool) {
	return 0, false
}

func (o *Float) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Float) AsTime() (time.Time, bool) {
	return time.Time{}, false
}
