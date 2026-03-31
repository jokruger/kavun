package value

import (
	"encoding/binary"
	"strconv"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Int struct {
	Object
	value int64
}

// Should be used only for static initialization. For dynamic creation of built-in functions, use Allocator.NewInt.
func NewStaticInt(v int64) core.Object {
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

func (o *Int) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	alloc := vm.Allocator()

	// int op float => float
	if rhs, ok := rhs.(*Float); ok {
		v := float64(o.value)
		switch op {
		case token.Add:
			return alloc.NewFloat(v + rhs.value), nil
		case token.Sub:
			return alloc.NewFloat(v - rhs.value), nil
		case token.Mul:
			return alloc.NewFloat(v * rhs.value), nil
		case token.Quo:
			return alloc.NewFloat(v / rhs.value), nil
		case token.Less:
			return alloc.NewBool(v < rhs.value), nil
		case token.Greater:
			return alloc.NewBool(v > rhs.value), nil
		case token.LessEq:
			return alloc.NewBool(v <= rhs.value), nil
		case token.GreaterEq:
			return alloc.NewBool(v >= rhs.value), nil
		}
	}

	// int op any => int
	v, ok := rhs.AsInt()
	if !ok {
		return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
	}

	switch op {
	case token.Add:
		return alloc.NewInt(o.value + v), nil
	case token.Sub:
		return alloc.NewInt(o.value - v), nil
	case token.Mul:
		return alloc.NewInt(o.value * v), nil
	case token.Quo:
		return alloc.NewInt(o.value / v), nil
	case token.Rem:
		return alloc.NewInt(o.value % v), nil
	case token.And:
		return alloc.NewInt(o.value & v), nil
	case token.Or:
		return alloc.NewInt(o.value | v), nil
	case token.Xor:
		return alloc.NewInt(o.value ^ v), nil
	case token.AndNot:
		return alloc.NewInt(o.value &^ v), nil
	case token.Shl:
		return alloc.NewInt(o.value << uint64(v)), nil
	case token.Shr:
		return alloc.NewInt(o.value >> uint64(v)), nil
	case token.Less:
		return alloc.NewBool(o.value < v), nil
	case token.Greater:
		return alloc.NewBool(o.value > v), nil
	case token.LessEq:
		return alloc.NewBool(o.value <= v), nil
	case token.GreaterEq:
		return alloc.NewBool(o.value >= v), nil
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

func (o *Int) Copy(alloc core.Allocator) core.Object {
	return alloc.NewInt(o.value)
}

func (o *Int) Access(vm core.VM, index core.Object, op core.Opcode) (core.Object, error) {
	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidSelectorError(o, k)
	}

	alloc := vm.Allocator()
	switch k {
	case "int":
		return o, nil

	case "float":
		return alloc.NewFloat(float64(o.value)), nil

	case "bool":
		return alloc.NewBool(o.IsTrue()), nil

	case "char":
		return alloc.NewChar(rune(o.value)), nil

	case "string":
		return alloc.NewString(strconv.FormatInt(o.value, 10)), nil

	case "time":
		return alloc.NewTime(time.Unix(o.value, 0)), nil

	default:
		return nil, core.NewInvalidSelectorError(o, k)
	}
}

func (o *Int) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Int) IsTrue() bool {
	return o.value != 0
}

func (o *Int) IsFalse() bool {
	return o.value == 0
}

func (o *Int) IsImmutable() bool {
	return true
}

func (o *Int) IsInt() bool {
	return true
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
	return o.IsTrue(), true
}

func (o *Int) AsRune() (rune, bool) {
	return rune(o.value), true
}

func (o *Int) AsTime() (time.Time, bool) {
	return time.Unix(o.value, 0), true
}
