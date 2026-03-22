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
	switch rhs := rhs.(type) {
	case *Int:
		switch op {
		case token.Add:
			return alloc.NewInt(o.value + rhs.value), nil
		case token.Sub:
			return alloc.NewInt(o.value - rhs.value), nil
		case token.Mul:
			return alloc.NewInt(o.value * rhs.value), nil
		case token.Quo:
			return alloc.NewInt(o.value / rhs.value), nil
		case token.Rem:
			return alloc.NewInt(o.value % rhs.value), nil
		case token.And:
			return alloc.NewInt(o.value & rhs.value), nil
		case token.Or:
			return alloc.NewInt(o.value | rhs.value), nil
		case token.Xor:
			return alloc.NewInt(o.value ^ rhs.value), nil
		case token.AndNot:
			return alloc.NewInt(o.value &^ rhs.value), nil
		case token.Shl:
			return alloc.NewInt(o.value << uint64(rhs.value)), nil
		case token.Shr:
			return alloc.NewInt(o.value >> uint64(rhs.value)), nil
		case token.Less:
			return alloc.NewBool(o.value < rhs.value), nil
		case token.Greater:
			return alloc.NewBool(o.value > rhs.value), nil
		case token.LessEq:
			return alloc.NewBool(o.value <= rhs.value), nil
		case token.GreaterEq:
			return alloc.NewBool(o.value >= rhs.value), nil
		}
	case *Float:
		switch op {
		case token.Add:
			return alloc.NewFloat(float64(o.value) + rhs.value), nil
		case token.Sub:
			return alloc.NewFloat(float64(o.value) - rhs.value), nil
		case token.Mul:
			return alloc.NewFloat(float64(o.value) * rhs.value), nil
		case token.Quo:
			return alloc.NewFloat(float64(o.value) / rhs.value), nil
		case token.Less:
			return alloc.NewBool(float64(o.value) < rhs.value), nil
		case token.Greater:
			return alloc.NewBool(float64(o.value) > rhs.value), nil
		case token.LessEq:
			return alloc.NewBool(float64(o.value) <= rhs.value), nil
		case token.GreaterEq:
			return alloc.NewBool(float64(o.value) >= rhs.value), nil
		}
	case *Char:
		switch op {
		case token.Add:
			return alloc.NewInt(o.value + int64(rhs.value)), nil
		case token.Sub:
			return alloc.NewInt(o.value - int64(rhs.value)), nil
		case token.Less:
			return alloc.NewBool(o.value < int64(rhs.value)), nil
		case token.Greater:
			return alloc.NewBool(o.value > int64(rhs.value)), nil
		case token.LessEq:
			return alloc.NewBool(o.value <= int64(rhs.value)), nil
		case token.GreaterEq:
			return alloc.NewBool(o.value >= int64(rhs.value)), nil
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

func (o *Int) Copy(alloc core.Allocator) core.Object {
	return alloc.NewInt(o.value)
}

func (o *Int) Access(core.VM, core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
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
