package value

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
)

type Bytes struct {
	Object
	value []byte
}

func (o *Bytes) GobDecode(b []byte) error {
	decoded := make([]byte, len(b))
	copy(decoded, b)
	o.Set(decoded)
	return nil
}

func (o *Bytes) GobEncode() ([]byte, error) {
	encoded := make([]byte, len(o.value))
	copy(encoded, o.value)
	return encoded, nil
}

func (o *Bytes) Set(v []byte) {
	o.value = v
	if o.value == nil {
		o.value = make([]byte, 0)
	}
}

func (o *Bytes) Value() []byte {
	return o.value
}

func (o *Bytes) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *Bytes) Len() int {
	return len(o.value)
}

func (o *Bytes) Append(v []byte) {
	o.value = append(o.value, v...)
}

func (o *Bytes) At(i int) byte {
	return o.value[i]
}

func (o *Bytes) Clear() {
	o.value = o.value[:0]
}

func (o *Bytes) Slice(start, end int) []byte {
	return o.value[start:end]
}

func (o *Bytes) TypeName() string {
	return "bytes"
}

func (o *Bytes) String() string {
	es := make([]string, len(o.value))
	for i, b := range o.value {
		es[i] = fmt.Sprintf("%d", b)
	}
	return fmt.Sprintf("bytes([%s])", strings.Join(es, ", "))
}

func (o *Bytes) Interface() any {
	return o.value
}

func (o *Bytes) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	alloc := vm.Allocator()
	switch op {
	case token.Add:
		switch rhs := rhs.(type) {
		case *Bytes:
			if len(o.value)+len(rhs.value) > core.MaxBytesLen {
				return nil, core.NewBytesLimitError("bytes concatenation")
			}
			return alloc.NewBytes(append(o.value, rhs.value...)), nil
		}
	}
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Bytes) Equals(x core.Object) bool {
	t, ok := x.AsBytes()
	if !ok {
		return false
	}
	return bytes.Equal(o.value, t)
}

func (o *Bytes) Copy(alloc core.Allocator) core.Object {
	t := make([]byte, len(o.value))
	copy(t, o.value)
	return alloc.NewBytes(t)
}

func (o *Bytes) Access(vm core.VM, index core.Object, mode core.Opcode) (core.Object, error) {
	alloc := vm.Allocator()

	if mode == parser.OpSelect {
		return nil, core.NewInvalidAccessModeError("bytes", "select")
	}

	i, ok := index.AsInt()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("bytes index", "int", index)
	}

	if i < 0 || i >= int64(len(o.value)) {
		return alloc.NewUndefined(), nil
	}

	return alloc.NewInt(int64(o.value[i])), nil
}

func (o *Bytes) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Bytes) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewBytesIterator(o.value)
}

func (o *Bytes) IsTrue() bool {
	return len(o.value) > 0
}

func (o *Bytes) IsFalse() bool {
	return len(o.value) == 0
}

func (o *Bytes) IsIterable() bool {
	return true
}

func (o *Bytes) IsImmutable() bool {
	return true
}

func (o *Bytes) AsString() (string, bool) {
	return string(o.value), true
}

func (o *Bytes) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

func (o *Bytes) AsBytes() ([]byte, bool) {
	return o.value, true
}
