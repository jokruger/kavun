package value

import (
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Bool struct {
	Object
	value bool
}

// Should be used only for static initialization. For dynamic creation use Allocator.NewBool.
func NewStaticBool(v bool) *Bool {
	return &Bool{value: v}
}

func (o *Bool) GobDecode(b []byte) error {
	if len(b) != 1 {
		return core.NewDecodeBinarySizeError(o, 1, len(b))
	}
	o.value = b[0] == 1
	return nil
}

func (o *Bool) GobEncode() ([]byte, error) {
	if o.value {
		return []byte{1}, nil
	}
	return []byte{0}, nil
}

func (o *Bool) Value() bool {
	return o.value
}

func (o *Bool) TypeName() string {
	return "bool"
}

func (o *Bool) String() string {
	if o.value {
		return "true"
	}
	return "false"
}

func (o *Bool) Interface() any {
	return o.value
}

func (o *Bool) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Bool) Equals(x core.Object) bool {
	t, ok := x.AsBool()
	if !ok {
		return false
	}
	return o.value == t
}

func (o *Bool) Copy(alloc core.Allocator) core.Object {
	return alloc.NewBool(o.value)
}

func (o *Bool) Access(vm core.VM, index core.Object, op core.Opcode) (core.Object, error) {
	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidSelectorError(o, k)
	}

	alloc := vm.Allocator()
	switch k {
	case "bool":
		return o, nil

	case "int":
		if o.value {
			return alloc.NewInt(1), nil
		}
		return alloc.NewInt(0), nil

	case "string":
		return alloc.NewString(o.String()), nil

	default:
		return nil, core.NewInvalidSelectorError(o, k)
	}
}

func (o *Bool) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Bool) IsTrue() bool {
	return o.value
}

func (o *Bool) IsFalse() bool {
	return !o.value
}

func (o *Bool) IsImmutable() bool {
	return true
}

func (o *Bool) IsBool() bool {
	return true
}

func (o *Bool) AsString() (string, bool) {
	return o.String(), true
}

func (o *Bool) AsInt() (int64, bool) {
	if o.value {
		return 1, true
	}
	return 0, true
}

func (o *Bool) AsBool() (bool, bool) {
	return o.value, true
}
