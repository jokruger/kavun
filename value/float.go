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

// Should be used only for static initialization. For dynamic creation of built-in functions, use Allocator.NewFloat.
func NewStaticFloat(v float64) core.Object {
	o := &Float{}
	o.Set(v)
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

func (o *Float) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	alloc := vm.Allocator()
	switch rhs := rhs.(type) {
	case *Float:
		switch op {
		case token.Add:
			return alloc.NewFloat(o.value + rhs.value), nil
		case token.Sub:
			return alloc.NewFloat(o.value - rhs.value), nil
		case token.Mul:
			return alloc.NewFloat(o.value * rhs.value), nil
		case token.Quo:
			return alloc.NewFloat(o.value / rhs.value), nil
		case token.Less:
			return alloc.NewBool(o.value < rhs.value), nil
		case token.Greater:
			return alloc.NewBool(o.value > rhs.value), nil
		case token.LessEq:
			return alloc.NewBool(o.value <= rhs.value), nil
		case token.GreaterEq:
			return alloc.NewBool(o.value >= rhs.value), nil
		}
	case *Int:
		switch op {
		case token.Add:
			return alloc.NewFloat(o.value + float64(rhs.value)), nil
		case token.Sub:
			return alloc.NewFloat(o.value - float64(rhs.value)), nil
		case token.Mul:
			return alloc.NewFloat(o.value * float64(rhs.value)), nil
		case token.Quo:
			return alloc.NewFloat(o.value / float64(rhs.value)), nil
		case token.Less:
			return alloc.NewBool(o.value < float64(rhs.value)), nil
		case token.Greater:
			return alloc.NewBool(o.value > float64(rhs.value)), nil
		case token.LessEq:
			return alloc.NewBool(o.value <= float64(rhs.value)), nil
		case token.GreaterEq:
			return alloc.NewBool(o.value >= float64(rhs.value)), nil
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

func (o *Float) Copy(alloc core.Allocator) core.Object {
	return alloc.NewFloat(o.value)
}

func (o *Float) Access(core.VM, core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *Float) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Float) IsTrue() bool {
	return !o.IsFalse()
}

func (o *Float) IsFalse() bool {
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
	return !o.IsFalse(), true
}
