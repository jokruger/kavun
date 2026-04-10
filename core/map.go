package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
	"unsafe"

	"github.com/jokruger/gs/errs"
)

type Map struct {
	value     map[string]Value
	immutable bool
}

func (o *Map) Set(vals map[string]Value, immutable bool) {
	o.value = vals
	o.immutable = immutable

	if o.value == nil {
		o.value = make(map[string]Value)
	}
}

func (o *Map) Value() map[string]Value {
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

func (o *Map) Get(key string) (Value, bool) {
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

func (o *Map) SetKey(key string, val Value) {
	o.value[key] = val
}

func MapValue(v *Map) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_MAP,
	}
}

func NewMapValue(vals map[string]Value, immutable bool) Value {
	t := &Map{}
	t.Set(vals, immutable)
	return MapValue(t)
}

func mapTypeName(v Value) string {
	o := (*Map)(v.Ptr)
	if o.immutable {
		return "immutable-map"
	}
	return "map"
}

func mapTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Map)(v.Ptr)
	var b []byte
	b = append(b, '{')
	len1 := o.Len() - 1
	idx := 0
	for key, value := range o.Value() {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := value.EncodeJSON()
		if err != nil {
			return nil, fmt.Errorf("map value at key %q: %w", key, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
		idx++
	}
	b = append(b, '}')
	return b, nil
}

func mapTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Map)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.immutable); err != nil {
		return nil, fmt.Errorf("map (immutable flag): %w", err)
	}
	if err := enc.Encode(o.value); err != nil {
		return nil, fmt.Errorf("map (elements): %w", err)
	}
	return buf.Bytes(), nil
}

func mapTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var immutable bool
	if err := dec.Decode(&immutable); err != nil {
		return fmt.Errorf("map (immutable flag): %w", err)
	}
	var value map[string]Value
	if err := dec.Decode(&value); err != nil {
		return fmt.Errorf("map (elements): %w", err)
	}
	if value == nil {
		value = make(map[string]Value)
	}
	o := &Map{
		value:     value,
		immutable: immutable,
	}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func mapTypeString(v Value) string {
	o := (*Map)(v.Ptr)
	pairs := make([]string, 0, len(o.value))
	for k, v := range o.value {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("map({%s})", strings.Join(pairs, ", "))
}

func mapTypeInterface(v Value) any {
	o := (*Map)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.value {
		res[key] = v.Interface()
	}
	return res
}

func mapTypeEqual(v Value, r Value) bool {
	switch {
	case r.IsMap():
		o := (*Map)(v.Ptr)
		x := (*Map)(r.Ptr)
		if len(o.value) != len(x.value) {
			return false
		}
		for k, v := range o.value {
			if !v.Equal(x.value[k]) {
				return false
			}
		}
		return true

	case r.IsRecord():
		o := (*Map)(v.Ptr)
		x := (*Record)(r.Ptr)
		if len(o.value) != len(x.value) {
			return false
		}
		for k, v := range o.value {
			if !v.Equal(x.value[k]) {
				return false
			}
		}
		return true

	default:
		return false
	}
}

func mapTypeCopy(v Value, a Allocator) Value {
	// perform a deep copy of the map even if it is immutable (since the values may be mutable)
	o := (*Map)(v.Ptr)
	c := make(map[string]Value, len(o.value))
	for k, v := range o.value {
		c[k] = v.Copy(a)
	}
	return a.NewMapValue(c, false)
}

func mapTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_record":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("map.to_record", "0", len(args))
		}
		return v, nil

	case "is_empty":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("map.is_empty", "0", len(args))
		}
		o := (*Map)(v.Ptr)
		return BoolValue(len(o.value) == 0), nil

	case "filter":
		return mapFnFilter(v, vm, "map.filter", args)

	case "count":
		return mapFnCount(v, vm, "map.count", args)

	case "all":
		return mapFnAll(v, vm, "map.all", args)

	case "any":
		return mapFnAny(v, vm, "map.any", args)

	case "len":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("map.len", "0", len(args))
		}
		o := (*Map)(v.Ptr)
		return IntValue(int64(len(o.value))), nil

	case "keys":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("map.keys", "0", len(args))
		}
		return mapKeys(v, vm.Allocator())

	case "values":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("map.values", "0", len(args))
		}
		return mapValues(v, vm.Allocator())

	default:
		return UndefinedValue(), errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func mapTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return UndefinedValue(), errs.NewInvalidIndexTypeError("map access", "string", index.TypeName())
	}

	if mode == OpIndex {
		o := (*Map)(v.Ptr)
		r, ok := o.value[k]
		if !ok {
			return UndefinedValue(), nil
		}
		return r, nil
	}

	return UndefinedValue(), errs.NewInvalidSelectorError(v.TypeName(), k)
}

func mapTypeAssign(v Value, index Value, r Value) error {
	o := (*Map)(v.Ptr)
	if o.immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("map assignment", "string", index.TypeName())
	}
	o.value[k] = r

	return nil
}

func mapTypeIsIterable(v Value) bool {
	return true
}

func mapTypeIterator(v Value, a Allocator) Value {
	o := (*Map)(v.Ptr)
	return a.NewMapIteratorValue(o.value)
}

func mapTypeIsImmutable(v Value) bool {
	o := (*Map)(v.Ptr)
	return o.immutable
}

func mapTypeIsTrue(v Value) bool {
	o := (*Map)(v.Ptr)
	return len(o.value) > 0
}

func mapTypeAsString(v Value) (string, bool) {
	return mapTypeString(v), true
}

func mapTypeAsBool(v Value) (bool, bool) {
	return mapTypeIsTrue(v), true
}

func mapKeys(v Value, a Allocator) (Value, error) {
	o := (*Map)(v.Ptr)
	keys := make([]Value, 0, len(o.value))
	for k := range o.value {
		keys = append(keys, a.NewStringValue(k))
	}
	return a.NewArrayValue(keys, false), nil
}

func mapValues(v Value, a Allocator) (Value, error) {
	o := (*Map)(v.Ptr)
	values := make([]Value, 0, len(o.value))
	for _, v := range o.value {
		values = append(values, v)
	}
	return a.NewArrayValue(values, false), nil
}

func mapFnFilter(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Map)(v.Ptr)
		filtered := make(map[string]Value, len(o.value))
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				filtered[k] = v
			}
		}
		return alloc.NewMapValue(filtered, false), nil

	case 2:
		o := (*Map)(v.Ptr)
		filtered := make(map[string]Value, len(o.value))
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				filtered[k] = v
			}
		}
		return alloc.NewMapValue(filtered, false), nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func mapFnCount(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Map)(v.Ptr)
		var count int64
		for k := range o.value {
			buf[0] = alloc.NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	case 2:
		o := (*Map)(v.Ptr)
		var count int64
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func mapFnAll(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Map)(v.Ptr)
		for k := range o.value {
			buf[0] = alloc.NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return UndefinedValue(), err
			}
			if !res.IsTrue() {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	case 2:
		o := (*Map)(v.Ptr)
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			if !res.IsTrue() {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func mapFnAny(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Map)(v.Ptr)
		for k := range o.value {
			buf[0] = alloc.NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	case 2:
		o := (*Map)(v.Ptr)
		for k, v := range o.value {
			buf[0] = alloc.NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}
