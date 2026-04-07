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
	value     map[string]core.Value
	immutable bool
}

func (o *Map) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	var vals map[string]core.Value
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

func (o *Map) Set(val map[string]core.Value, immutable bool) {
	o.value = val
	if o.value == nil {
		o.value = make(map[string]core.Value)
	}
	o.immutable = immutable
}

func (o *Map) Value() map[string]core.Value {
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

func (o *Map) Get(key string) (core.Value, bool) {
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

func (o *Map) SetKey(key string, val core.Value) {
	o.value[key] = val
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

func (o *Map) BinaryOp(vm core.VM, op token.Token, rhs core.Value) (core.Value, error) {
	return core.UndefinedValue(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
}

func (o *Map) Equals(x core.Value) bool {
	if !x.IsObject() {
		return false
	}

	switch x := x.Object().(type) {
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

func (o *Map) Copy(alloc core.Allocator) core.Value {
	// perform a deep copy of the map even if it is immutable (since the values may be mutable)
	c := make(map[string]core.Value, len(o.value))
	for k, v := range o.value {
		c[k] = v.Copy(alloc)
	}
	return alloc.NewMapValue(c, false)
}

func (o *Map) Method(vm core.VM, name string, args []core.Value) (core.Value, error) {
	switch name {
	case "to_record":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("map.to_record", "0", len(args))
		}
		return vm.Allocator().NewRecordValue(o.value, o.immutable), nil

	case "is_empty":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("map.is_empty", "0", len(args))
		}
		return core.BoolValue(len(o.value) == 0), nil

	case "filter":
		return o.fnFilter(vm, "map.filter", args)

	case "count":
		return o.fnCount(vm, "map.count", args)

	case "all":
		return o.fnAll(vm, "map.all", args)

	case "any":
		return o.fnAny(vm, "map.any", args)

	case "len":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("map.len", "0", len(args))
		}
		return core.IntValue(int64(len(o.value))), nil

	case "keys":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("map.keys", "0", len(args))
		}
		return o.keys(vm)

	case "values":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("map.values", "0", len(args))
		}
		return o.values(vm)

	default:
		return core.UndefinedValue(), core.NewInvalidMethodError(name, o.TypeName())
	}
}

func (o *Map) Access(vm core.VM, index core.Value, mode core.Opcode) (core.Value, error) {
	k, ok := index.AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidIndexTypeError("map access", "string", index.TypeName())
	}

	if mode == parser.OpIndex {
		r, ok := o.value[k]
		if !ok {
			return core.UndefinedValue(), nil
		}
		return r, nil
	}

	return core.UndefinedValue(), core.NewInvalidSelectorError(o.TypeName(), k)
}

func (o *Map) Assign(index, value core.Value) error {
	if o.immutable {
		return core.NewNotAssignableError(o.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return core.NewInvalidIndexTypeError("map assignment", "string", index.TypeName())
	}
	o.value[k] = value

	return nil
}

func (o *Map) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewMapIterator(o.value)
}

func (o *Map) IsImmutable() bool {
	return o.immutable
}

func (o *Map) IsMap() bool {
	return true
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

func (o *Map) AsString() (string, bool) {
	return o.String(), true
}

func (o *Map) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

func (o *Map) keys(vm core.VM) (core.Value, error) {
	alloc := vm.Allocator()
	keys := make([]core.Value, 0, len(o.value))
	for k := range o.value {
		keys = append(keys, alloc.NewStringValue(k))
	}
	return alloc.NewArrayValue(keys, false), nil
}

func (o *Map) values(vm core.VM) (core.Value, error) {
	alloc := vm.Allocator()
	values := make([]core.Value, 0, len(o.value))
	for _, v := range o.value {
		values = append(values, v)
	}
	return alloc.NewArrayValue(values, false), nil
}

func (o *Map) fnFilter(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]core.Value
	switch fn.Arity() {
	case 1:
		filtered := make(map[string]core.Value, len(o.value))
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				filtered[k] = v
			}
		}
		return alloc.NewMapValue(filtered, false), nil

	case 2:
		filtered := make(map[string]core.Value, len(o.value))
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				filtered[k] = v
			}
		}
		return alloc.NewMapValue(filtered, false), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func (o *Map) fnCount(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]core.Value
	switch fn.Arity() {
	case 1:
		var count int64
		for k := range o.value {
			buf[0] = alloc.NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				count++
			}
		}
		return core.IntValue(count), nil

	case 2:
		var count int64
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				count++
			}
		}
		return core.IntValue(count), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func (o *Map) fnAll(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]core.Value
	switch fn.Arity() {
	case 1:
		for k := range o.value {
			buf[0] = alloc.NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsFalse() {
				return core.BoolValue(false), nil
			}
		}
		return core.BoolValue(true), nil

	case 2:
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsFalse() {
				return core.BoolValue(false), nil
			}
		}
		return core.BoolValue(true), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func (o *Map) fnAny(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]core.Value
	switch fn.Arity() {
	case 1:
		for k := range o.value {
			buf[0] = alloc.NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				return core.BoolValue(true), nil
			}
		}
		return core.BoolValue(false), nil

	case 2:
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				return core.BoolValue(true), nil
			}
		}
		return core.BoolValue(false), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}
