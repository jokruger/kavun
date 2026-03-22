package value

import (
	"encoding/binary"
	"fmt"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Char struct {
	Object
	value rune
}

// Should be used only for static initialization. For dynamic creation of built-in functions, use Allocator.NewChar.
func NewStaticChar(v rune) core.Object {
	o := &Char{}
	o.Set(v)
	return o
}

func (o *Char) GobDecode(b []byte) error {
	if len(b) != 4 {
		return core.NewDecodeBinarySizeError(o, 4, len(b))
	}
	o.Set(rune(int32(binary.BigEndian.Uint32(b))))
	return nil
}

func (o *Char) GobEncode() ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(int32(o.value)))
	return b, nil
}

func (o *Char) Set(v rune) {
	o.value = v
}

func (o *Char) Value() rune {
	return o.value
}

func (o *Char) TypeName() string {
	return "char"
}

func (o *Char) String() string {
	return fmt.Sprintf("%q", o.value)
}

func (o *Char) Interface() any {
	return o.value
}

func (o *Char) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	alloc := vm.Allocator()
	switch rhs := rhs.(type) {
	case *Char:
		switch op {
		case token.Add:
			return alloc.NewChar(o.value + rhs.value), nil
		case token.Sub:
			return alloc.NewChar(o.value - rhs.value), nil
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
			return alloc.NewInt(int64(o.value) + rhs.value), nil
		case token.Sub:
			return alloc.NewInt(int64(o.value) - rhs.value), nil
		case token.Less:
			return alloc.NewBool(int64(o.value) < rhs.value), nil
		case token.Greater:
			return alloc.NewBool(int64(o.value) > rhs.value), nil
		case token.LessEq:
			return alloc.NewBool(int64(o.value) <= rhs.value), nil
		case token.GreaterEq:
			return alloc.NewBool(int64(o.value) >= rhs.value), nil
		}
	}
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Char) Equals(x core.Object) bool {
	t, ok := x.AsRune()
	if !ok {
		return false
	}
	return o.value == t
}

func (o *Char) Copy(alloc core.Allocator) core.Object {
	return alloc.NewChar(o.value)
}

func (o *Char) Access(core.VM, core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *Char) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Char) IsTrue() bool {
	return o.value != 0
}

func (o *Char) IsFalse() bool {
	return o.value == 0
}

func (o *Char) IsImmutable() bool {
	return true
}

func (o *Char) AsString() (string, bool) {
	return string(o.value), true
}

func (o *Char) AsInt() (int64, bool) {
	return int64(o.value), true
}

func (o *Char) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

func (o *Char) AsRune() (rune, bool) {
	return o.value, true
}
