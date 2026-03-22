package value

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strings"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
)

type Array struct {
	Object
	value     []core.Object
	immutable bool
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
	elements := make([]string, len(o.value))
	for i, e := range o.value {
		elements[i] = e.String()
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

func (o *Array) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	alloc := vm.Allocator()
	if rhs, ok := rhs.(*Array); ok {
		switch op {
		case token.Add:
			if len(rhs.value) == 0 {
				return o, nil
			}
			return alloc.NewArray(append(o.value, rhs.value...), false), nil
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

func (o *Array) Copy(alloc core.Allocator) core.Object {
	// Deep copy the array and its elements even if it is immutable (since the elements themselves may be mutable)
	c := make([]core.Object, len(o.value))
	for i, e := range o.value {
		c[i] = e.Copy(alloc)
	}
	return alloc.NewArray(c, false) // copy always returns a mutable array
}

func (o *Array) Access(vm core.VM, index core.Object, mode core.Opcode) (core.Object, error) {
	if mode == parser.OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return nil, core.NewInvalidIndexTypeError("array access", "int", index)
		}

		if i < 0 || i >= int64(len(o.value)) {
			return vm.Allocator().NewUndefined(), nil
		}

		return o.value[i], nil
	}

	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidSelectorError(o, k)
	}

	switch k {
	case "empty":
		return vm.Allocator().NewBool(len(o.value) == 0), nil

	case "len":
		return vm.Allocator().NewInt(int64(len(o.value))), nil

	case "first":
		if len(o.value) == 0 {
			return vm.Allocator().NewUndefined(), nil
		}
		return o.value[0], nil

	case "last":
		if len(o.value) == 0 {
			return vm.Allocator().NewUndefined(), nil
		}
		return o.value[len(o.value)-1], nil

	case "min":
		return o.min(vm)

	case "max":
		return o.max(vm)

	case "sort":
		return o.fnSort(vm, "array.sort")

	case "filter":
		return o.fnFilter(vm, "array.filter")

	default:
		return nil, core.NewInvalidSelectorError(o, k)
	}
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

func (o *Array) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewArrayIterator(o.value)
}

func (o *Array) IsTrue() bool {
	return len(o.value) > 0
}

func (o *Array) IsFalse() bool {
	return len(o.value) == 0
}

func (o *Array) IsIterable() bool {
	return true
}

func (o *Array) IsImmutable() bool {
	return o.immutable
}

func (o *Array) AsString() (string, bool) {
	return o.String(), true
}

func (o *Array) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

func (o *Array) AsBytes() ([]byte, bool) {
	bs := make([]byte, len(o.value))
	for i, e := range o.value {
		b, ok := e.AsInt()
		if !ok || b < 0 || b > 255 {
			return nil, false
		}
		bs[i] = byte(b)
	}
	return bs, true
}

func (o *Array) min(vm core.VM) (core.Object, error) {
	if len(o.value) == 0 {
		return vm.Allocator().NewUndefined(), nil
	}

	v := o.value[0]
	for i := 1; i < len(o.value); i++ {
		less, err := o.value[i].BinaryOp(vm, token.Less, v)
		if err != nil {
			return nil, err
		}
		if less.IsTrue() {
			v = o.value[i]
		}
	}

	return v, nil
}

func (o *Array) max(vm core.VM) (core.Object, error) {
	if len(o.value) == 0 {
		return vm.Allocator().NewUndefined(), nil
	}

	v := o.value[0]
	for i := 1; i < len(o.value); i++ {
		greater, err := o.value[i].BinaryOp(vm, token.Greater, v)
		if err != nil {
			return nil, err
		}
		if greater.IsTrue() {
			v = o.value[i]
		}
	}

	return v, nil
}

func (o *Array) fnSort(vm core.VM, name string) (core.Object, error) {
	return vm.Allocator().NewBuiltinFunction(name, func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) != 0 {
			return nil, core.NewWrongNumArgumentsError(name, "0", len(args))
		}

		r := o.Copy(vm.Allocator()).(*Array)
		var err error
		slices.SortFunc(r.value, func(a, b core.Object) int {
			less, e := a.BinaryOp(vm, token.Less, b)
			if e != nil {
				err = e
				return 0
			}
			if less.IsFalse() {
				if a.Equals(b) {
					return 0
				}
				return 1
			}
			return -1
		})
		return r, err
	}, 0, false), nil
}

func (o *Array) fnFilter(vm core.VM, name string) (core.Object, error) {
	return vm.Allocator().NewBuiltinFunction(name, func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError(name, "1", len(args))
		}

		fn := args[0]
		if !fn.IsCallable() || fn.IsVariadic() {
			return nil, core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn)
		}

		alloc := vm.Allocator()
		switch fn.Arity() {
		case 1:
			filtered := make([]core.Object, 0, len(o.value))
			for _, v := range o.value {
				res, err := fn.Call(vm, v)
				if err != nil {
					return nil, err
				}
				if res.IsTrue() {
					filtered = append(filtered, v)
				}
			}
			return alloc.NewArray(filtered, false), nil

		case 2:
			filtered := make([]core.Object, 0, len(o.value))
			for i, v := range o.value {
				res, err := fn.Call(vm, alloc.NewInt(int64(i)), v)
				if err != nil {
					return nil, err
				}
				if res.IsTrue() {
					filtered = append(filtered, v)
				}
			}
			return alloc.NewArray(filtered, false), nil

		default:
			return nil, core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn)
		}
	}, 1, false), nil
}
