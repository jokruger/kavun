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
	Elements  map[string]Value
	Immutable bool
}

func (o *Map) Set(elements map[string]Value, immutable bool) {
	o.Elements = elements
	o.Immutable = immutable

	if o.Elements == nil {
		o.Elements = make(map[string]Value)
	}
}

// MapValue creates new boxed map value.
func MapValue(v *Map) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_MAP,
	}
}

// NewMapValue creates new (heap-allocated) map value.
func NewMapValue(vals map[string]Value, immutable bool) Value {
	t := &Map{}
	t.Set(vals, immutable)
	return MapValue(t)
}

// ToMap converts boxed map value to *Map. It is a caller's responsibility to ensure the type is correct.
func ToMap(v Value) *Map {
	return (*Map)(v.Ptr)
}

/* Map type methods */

func mapTypeName(v Value) string {
	o := (*Map)(v.Ptr)
	if o.Immutable {
		return "immutable-map"
	}
	return "map"
}

func mapTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Map)(v.Ptr)
	var b []byte
	b = append(b, '{')
	len1 := len(o.Elements) - 1
	idx := 0
	for key, value := range o.Elements {
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
	if err := enc.Encode(o.Immutable); err != nil {
		return nil, fmt.Errorf("map (immutable flag): %w", err)
	}
	if err := enc.Encode(o.Elements); err != nil {
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
		Elements:  value,
		Immutable: immutable,
	}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func mapTypeString(v Value) string {
	o := (*Map)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("map({%s})", strings.Join(pairs, ", "))
}

func mapTypeInterface(v Value) any {
	o := (*Map)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func mapTypeEqual(v Value, r Value) bool {
	switch r.Type {
	case VT_MAP:
		o := (*Map)(v.Ptr)
		x := (*Map)(r.Ptr)
		if len(o.Elements) != len(x.Elements) {
			return false
		}
		for k, v := range o.Elements {
			if !v.Equal(x.Elements[k]) {
				return false
			}
		}
		return true

	case VT_RECORD:
		o := (*Map)(v.Ptr)
		x := (*Record)(r.Ptr)
		if len(o.Elements) != len(x.Elements) {
			return false
		}
		for k, v := range o.Elements {
			if !v.Equal(x.Elements[k]) {
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
	c := make(map[string]Value, len(o.Elements))
	for k, v := range o.Elements {
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
		return BoolValue(len(o.Elements) == 0), nil

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
		return IntValue(int64(len(o.Elements))), nil

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

	case "contains":
		if len(args) != 1 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("map.contains", "1", len(args))
		}
		return BoolValue(mapTypeContains(v, args[0])), nil

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
		r, ok := o.Elements[k]
		if !ok {
			return UndefinedValue(), nil
		}
		return r, nil
	}

	return UndefinedValue(), errs.NewInvalidSelectorError(v.TypeName(), k)
}

func mapTypeAssign(v Value, index Value, r Value) error {
	o := (*Map)(v.Ptr)
	if o.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("map assignment", "string", index.TypeName())
	}
	o.Elements[k] = r

	return nil
}

func mapTypeIsIterable(v Value) bool {
	return true
}

func mapTypeIterator(v Value, a Allocator) Value {
	o := (*Map)(v.Ptr)
	return a.NewMapIteratorValue(o.Elements)
}

func mapTypeIsImmutable(v Value) bool {
	o := (*Map)(v.Ptr)
	return o.Immutable
}

func mapTypeIsTrue(v Value) bool {
	o := (*Map)(v.Ptr)
	return len(o.Elements) > 0
}

func mapTypeAsString(v Value) (string, bool) {
	return mapTypeString(v), true
}

func mapTypeAsBool(v Value) (bool, bool) {
	return mapTypeIsTrue(v), true
}

func mapKeys(v Value, a Allocator) (Value, error) {
	o := (*Map)(v.Ptr)
	keys := make([]Value, 0, len(o.Elements))
	for k := range o.Elements {
		keys = append(keys, a.NewStringValue(k))
	}
	return a.NewArrayValue(keys, false), nil
}

func mapValues(v Value, a Allocator) (Value, error) {
	o := (*Map)(v.Ptr)
	values := make([]Value, 0, len(o.Elements))
	for _, v := range o.Elements {
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
		filtered := make(map[string]Value, len(o.Elements))
		for k, v := range o.Elements {
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
		filtered := make(map[string]Value, len(o.Elements))
		for k, v := range o.Elements {
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
		for k := range o.Elements {
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
		for k, v := range o.Elements {
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
		for k := range o.Elements {
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
		for k, v := range o.Elements {
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
		for k := range o.Elements {
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
		for k, v := range o.Elements {
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

func mapTypeContains(v Value, e Value) bool {
	o := (*Record)(v.Ptr)
	s, ok := e.AsString()
	if !ok {
		return false
	}
	_, ok = o.Elements[s]
	return ok
}
