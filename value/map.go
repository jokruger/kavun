package value

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
)

type Map struct {
	Object
	value     map[string]core.Object
	immutable bool
}

func (o *Map) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	var vals map[string]core.Object
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

func (o *Map) GobEncode() ([]byte, error) {
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

func (o *Map) Set(val map[string]core.Object, immutable bool) {
	o.value = val
	if o.value == nil {
		o.value = make(map[string]core.Object)
	}

	o.immutable = immutable
}

func (o *Map) Value() map[string]core.Object {
	return o.value
}

func (o *Map) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *Map) Len() int {
	return len(o.value)
}

func (o *Map) Delete(key string) {
	delete(o.value, key)
}

func (o *Map) Has(key string) bool {
	_, ok := o.value[key]
	return ok
}

func (o *Map) Get(key string) (core.Object, bool) {
	v, ok := o.value[key]
	return v, ok
}

func (o *Map) Keys() []string {
	keys := make([]string, 0, len(o.value))
	for k := range o.value {
		keys = append(keys, k)
	}
	return keys
}

func (o *Map) SetKey(key string, value core.Object) {
	o.value[key] = value
}

func (o *Map) TypeName() string {
	if o.immutable {
		return "immutable-map"
	}
	return "map"
}

func (o *Map) String() string {
	pairs := make([]string, 0, len(o.value))
	for k, v := range o.value {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("map({%s})", strings.Join(pairs, ", "))
}

func (o *Map) Interface() any {
	res := make(map[string]any)
	for key, v := range o.value {
		res[key] = v.Interface()
	}
	return res
}

func (o *Map) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Map) Equals(x core.Object) bool {
	if o == x {
		return true
	}

	switch x := x.(type) {
	case *Map:
		if len(o.value) != len(x.value) {
			return false
		}
		for k, v := range o.value {
			if !v.Equals(x.value[k]) {
				return false
			}
		}
		return true
	case *Record:
		if len(o.value) != len(x.value) {
			return false
		}
		for k, v := range o.value {
			if !v.Equals(x.value[k]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (o *Map) Copy(alloc core.Allocator) core.Object {
	// perform a deep copy of the map even if it is immutable (since the values may be mutable)
	c := make(map[string]core.Object, len(o.value))
	for k, v := range o.value {
		c[k] = v.Copy(alloc)
	}
	return alloc.NewMap(c, false) // copy always returns a mutable map
}

func (o *Map) Access(vm core.VM, index core.Object, mode core.Opcode) (core.Object, error) {
	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("map access", "string", index)
	}

	if mode == parser.OpIndex {
		r, ok := o.value[k]
		if !ok {
			return vm.Allocator().NewUndefined(), nil
		}
		return r, nil
	}

	switch k {
	case "empty":
		return vm.Allocator().NewBool(len(o.value) == 0), nil

	case "len":
		return vm.Allocator().NewInt(int64(len(o.value))), nil

	case "keys":
		return o.keys(vm)

	case "values":
		return o.values(vm)

	case "filter":
		return o.fnFilter(vm, "map.filter")

	case "count":
		return o.fnCount(vm, "map.count")

	case "all":
		return o.fnAll(vm, "map.all")

	case "any":
		return o.fnAny(vm, "map.any")

	default:
		return nil, core.NewInvalidSelectorError(o, k)
	}
}

func (o *Map) Assign(index, value core.Object) error {
	if o.immutable {
		return core.NewNotAssignableError(o)
	}

	k, ok := index.AsString()
	if !ok {
		return core.NewInvalidIndexTypeError("map assignment", "string", index)
	}
	o.value[k] = value

	return nil
}

func (o *Map) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewMapIterator(o.value)
}

func (o *Map) IsTrue() bool {
	return len(o.value) > 0
}

func (o *Map) IsFalse() bool {
	return len(o.value) == 0
}

func (o *Map) IsIterable() bool {
	return true
}

func (o *Map) IsImmutable() bool {
	return o.immutable
}

func (o *Map) AsString() (string, bool) {
	return o.String(), true
}

func (o *Map) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

func (o *Map) keys(vm core.VM) (core.Object, error) {
	alloc := vm.Allocator()
	keys := make([]core.Object, 0, len(o.value))
	for k := range o.value {
		keys = append(keys, alloc.NewString(k))
	}
	return alloc.NewArray(keys, false), nil
}

func (o *Map) values(vm core.VM) (core.Object, error) {
	alloc := vm.Allocator()
	values := make([]core.Object, 0, len(o.value))
	for _, v := range o.value {
		values = append(values, v)
	}
	return alloc.NewArray(values, false), nil
}

func (o *Map) fnFilter(vm core.VM, name string) (core.Object, error) {
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
			filtered := make(map[string]core.Object, len(o.value))
			for k, v := range o.value {
				res, err := fn.Call(vm, alloc.NewString(k))
				if err != nil {
					return nil, err
				}
				if res.IsTrue() {
					filtered[k] = v
				}
			}
			return alloc.NewMap(filtered, false), nil

		case 2:
			filtered := make(map[string]core.Object, len(o.value))
			for k, v := range o.value {
				res, err := fn.Call(vm, alloc.NewString(k), v)
				if err != nil {
					return nil, err
				}
				if res.IsTrue() {
					filtered[k] = v
				}
			}
			return alloc.NewMap(filtered, false), nil

		default:
			return nil, core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn)
		}
	}, 1, false), nil
}

func (o *Map) fnCount(vm core.VM, name string) (core.Object, error) {
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
			var count int64
			for k := range o.value {
				res, err := fn.Call(vm, alloc.NewString(k))
				if err != nil {
					return nil, err
				}
				if res.IsTrue() {
					count++
				}
			}
			return alloc.NewInt(count), nil

		case 2:
			var count int64
			for k, v := range o.value {
				res, err := fn.Call(vm, alloc.NewString(k), v)
				if err != nil {
					return nil, err
				}
				if res.IsTrue() {
					count++
				}
			}
			return alloc.NewInt(count), nil

		default:
			return nil, core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn)
		}
	}, 1, false), nil
}

func (o *Map) fnAll(vm core.VM, name string) (core.Object, error) {
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
			for k := range o.value {
				res, err := fn.Call(vm, alloc.NewString(k))
				if err != nil {
					return nil, err
				}
				if res.IsFalse() {
					return alloc.NewBool(false), nil
				}
			}
			return alloc.NewBool(true), nil

		case 2:
			for k, v := range o.value {
				res, err := fn.Call(vm, alloc.NewString(k), v)
				if err != nil {
					return nil, err
				}
				if res.IsFalse() {
					return alloc.NewBool(false), nil
				}
			}
			return alloc.NewBool(true), nil

		default:
			return nil, core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn)
		}
	}, 1, false), nil
}

func (o *Map) fnAny(vm core.VM, name string) (core.Object, error) {
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
			for k := range o.value {
				res, err := fn.Call(vm, alloc.NewString(k))
				if err != nil {
					return nil, err
				}
				if res.IsTrue() {
					return alloc.NewBool(true), nil
				}
			}
			return alloc.NewBool(false), nil

		case 2:
			for k, v := range o.value {
				res, err := fn.Call(vm, alloc.NewString(k), v)
				if err != nil {
					return nil, err
				}
				if res.IsTrue() {
					return alloc.NewBool(true), nil
				}
			}
			return alloc.NewBool(false), nil

		default:
			return nil, core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn)
		}
	}, 1, false), nil
}
