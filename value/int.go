package value

import (
	"encoding/binary"
	"strconv"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Int struct {
	value int64
}

func NewInt(v int64) *Int {
	o := &Int{}
	o.Set(v)
	return o
}

func (o *Int) GobDecode(b []byte) error {
	if len(b) != 8 {
		return core.NewDecodeBinarySizeError(o, 8, len(b))
	}
	o.Set(int64(binary.BigEndian.Uint64(b)))
	return nil
}

func (o *Int) GobEncode() ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(o.value))
	return b, nil
}

func (o *Int) Set(v int64) {
	o.value = v
}

func (o *Int) Value() int64 {
	return o.value
}

func (o *Int) TypeName() string {
	return "int"
}

func (o *Int) String() string {
	return strconv.FormatInt(o.value, 10)
}

func (o *Int) Interface() any {
	return o.value
}

func (o *Int) Arity() int {
	return 0
}

func (o *Int) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	switch rhs := rhs.(type) {
	case *Int:
		switch op {
		case token.Add:
			r := o.value + rhs.value
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Sub:
			r := o.value - rhs.value
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Mul:
			r := o.value * rhs.value
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Quo:
			r := o.value / rhs.value
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Rem:
			r := o.value % rhs.value
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.And:
			r := o.value & rhs.value
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Or:
			r := o.value | rhs.value
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Xor:
			r := o.value ^ rhs.value
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.AndNot:
			r := o.value &^ rhs.value
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Shl:
			r := o.value << uint64(rhs.value)
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
		case token.Shr:
			r := o.value >> uint64(rhs.value)
			if r == o.value {
				return o, nil
			}
			return NewInt(r), nil
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
	case *Float:
		switch op {
		case token.Add:
			return NewFloat(float64(o.value) + rhs.value), nil
		case token.Sub:
			return NewFloat(float64(o.value) - rhs.value), nil
		case token.Mul:
			return NewFloat(float64(o.value) * rhs.value), nil
		case token.Quo:
			return NewFloat(float64(o.value) / rhs.value), nil
		case token.Less:
			if float64(o.value) < rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if float64(o.value) > rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if float64(o.value) <= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if float64(o.value) >= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case *Char:
		switch op {
		case token.Add:
			return NewChar(rune(o.value) + rhs.value), nil
		case token.Sub:
			return NewChar(rune(o.value) - rhs.value), nil
		case token.Less:
			if o.value < int64(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.value > int64(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.value <= int64(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.value >= int64(rhs.value) {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Int) Equals(x core.Object) bool {
	t, ok := x.AsInt()
	if !ok {
		return false
	}
	return o.value == t
}

func (o *Int) Copy() core.Object {
	return NewInt(o.value)
}

func (o *Int) Access(core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *Int) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Int) Iterate() core.Iterator {
	return nil
}

func (o *Int) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Int) IsFalsy() bool {
	return o.value == 0
}

func (o *Int) IsIterable() bool {
	return false
}

func (o *Int) IsCallable() bool {
	return false
}

func (o *Int) IsImmutable() bool {
	return true
}

func (o *Int) IsVariadic() bool {
	return false
}

func (o *Int) AsString() (string, bool) {
	return o.String(), true
}

func (o *Int) AsInt() (int64, bool) {
	return o.value, true
}

func (o *Int) AsFloat() (float64, bool) {
	return float64(o.value), true
}

func (o *Int) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Int) AsRune() (rune, bool) {
	return rune(o.value), true
}

func (o *Int) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Int) AsTime() (time.Time, bool) {
	return time.Unix(o.value, 0), true
}
