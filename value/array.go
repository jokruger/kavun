package value

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
)

/* === Array === */

type Array struct {
	value     []core.Object
	immutable bool
}

func NewArray(val []core.Object, immutable bool) *Array {
	o := &Array{}
	o.Set(val, immutable)
	return o
}

func (o *Array) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	var vals []core.Object
	if err := dec.Decode(&vals); err != nil {
		return err
	}

	var immutable bool
	if err := dec.Decode(&immutable); err != nil {
		return err
	}

	o.Set(vals, immutable)
	return nil
}

func (o *Array) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(o.value); err != nil {
		return nil, err
	}
	if err := enc.Encode(o.immutable); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (o *Array) Set(val []core.Object, immutable bool) {
	o.value = val
	if o.value == nil {
		o.value = []core.Object{}
	}

	o.immutable = immutable
}

func (o *Array) Value() []core.Object {
	return o.value
}

func (o *Array) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *Array) Len() int {
	return len(o.value)
}

func (o *Array) Slice(s, e int) []core.Object {
	return o.value[s:e]
}

func (o *Array) At(i int) core.Object {
	return o.value[i]
}

func (o *Array) Get(i int) (core.Object, bool) {
	if i < 0 || i >= len(o.value) {
		return UndefinedValue, false
	}
	return o.value[i], true
}

func (o *Array) Append(vals ...core.Object) {
	o.value = append(o.value, vals...)
}

func (o *Array) SetAt(i int, val core.Object) {
	o.value[i] = val
}

func (o *Array) TypeName() string {
	if o.immutable {
		return "immutable-array"
	}
	return "array"
}

func (o *Array) String() string {
	var elements []string
	for _, e := range o.value {
		elements = append(elements, e.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

func (o *Array) Interface() any {
	res := make([]any, len(o.value))
	for i, val := range o.value {
		res[i] = val.Interface()
	}
	return res
}

func (o *Array) Arity() int {
	return 0
}

func (o *Array) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	if rhs, ok := rhs.(*Array); ok {
		switch op {
		case token.Add:
			if len(rhs.value) == 0 {
				return o, nil
			}
			return NewArray(append(o.value, rhs.value...), false), nil
		}
	}
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Array) Equals(x core.Object) bool {
	if o == x {
		return true
	}

	switch x := x.(type) {
	case *Array:
		if len(o.value) != len(x.value) {
			return false
		}
		for i, e := range o.value {
			if !e.Equals(x.value[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (o *Array) Copy() core.Object {
	// Deep copy the array and its elements even if it is immutable (since the elements themselves may be mutable)
	c := make([]core.Object, len(o.value))
	for i, e := range o.value {
		c[i] = e.Copy()
	}
	return NewArray(c, false) // copy always returns a mutable array
}

func (o *Array) Access(index core.Object, mode core.Opcode) (core.Object, error) {
	if mode == parser.OpSelect {
		return nil, core.NewInvalidAccessModeError("array", "select")
	}

	i, ok := index.AsInt()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("array access", "int", index)
	}

	if i < 0 || i >= int64(len(o.value)) {
		return UndefinedValue, nil
	}

	return o.value[i], nil
}

func (o *Array) Assign(index, value core.Object) (err error) {
	if o.immutable {
		return core.NewNotAssignableError(o)
	}

	i, ok := index.AsInt()
	if !ok {
		return core.NewInvalidIndexTypeError("array assignment", "int", index)
	}
	if i < 0 || i >= int64(len(o.value)) {
		return core.NewIndexOutOfBoundsError("array assignment", int(i), len(o.value))
	}

	o.value[i] = value
	return nil
}

func (o *Array) Iterate() core.Iterator {
	return NewArrayIterator(o.value)
}

func (o *Array) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Array) IsFalsy() bool {
	return len(o.value) == 0
}

func (o *Array) IsIterable() bool {
	return true
}

func (o *Array) IsCallable() bool {
	return false
}

func (o *Array) IsImmutable() bool {
	return o.immutable
}

func (o *Array) IsVariadic() bool {
	return false
}

func (o *Array) AsString() (string, bool) {
	return o.String(), true
}

func (o *Array) AsInt() (int64, bool) {
	return 0, false
}

func (o *Array) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Array) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Array) AsRune() (rune, bool) {
	return 0, false
}

func (o *Array) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Array) AsTime() (time.Time, bool) {
	return time.Time{}, false
}
