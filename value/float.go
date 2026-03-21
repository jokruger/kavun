package value

import (
	"encoding/binary"
	"math"
	"strconv"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Float struct {
	Object
	value float64
}

func NewFloat(value float64) *Float {
	o := &Float{}
	o.Set(value)
	return o
}

func (o *Float) GobDecode(b []byte) error {
	if len(b) != 8 {
		return core.NewDecodeBinarySizeError(o, 8, len(b))
	}
	o.Set(math.Float64frombits(binary.BigEndian.Uint64(b)))
	return nil
}

func (o *Float) GobEncode() ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(o.value))
	return b, nil
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
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
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

func (o *Float) Access(core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *Float) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Float) IsFalsy() bool {
	return math.IsNaN(o.value)
}

func (o *Float) IsImmutable() bool {
	return true
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
