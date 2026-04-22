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
	Elements map[string]Value
}

func (o *Map) Set(elements map[string]Value) {
	o.Elements = elements
	if o.Elements == nil {
		o.Elements = make(map[string]Value)
	}
}

// RecordValue creates new boxed record value.
func RecordValue(v *Map, immutable bool) Value {
	if immutable {
		return Value{
			Ptr:  unsafe.Pointer(v),
			Type: VT_IMMUTABLE_RECORD,
		}
	}
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_RECORD,
	}
}

// MapValue creates new boxed map value.
func MapValue(v *Map, immutable bool) Value {
	if immutable {
		return Value{
			Ptr:  unsafe.Pointer(v),
			Type: VT_IMMUTABLE_MAP,
		}
	}
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_MAP,
	}
}

// NewRecordValue creates new (heap-allocated) record value.
func NewRecordValue(vals map[string]Value, immutable bool) Value {
	t := &Map{}
	t.Set(vals)
	return RecordValue(t, immutable)
}

// NewMapValue creates new (heap-allocated) map value.
func NewMapValue(vals map[string]Value, immutable bool) Value {
	t := &Map{}
	t.Set(vals)
	return MapValue(t, immutable)
}

/* Record type specific methods */

func recordTypeName(v Value) string {
	return "record"
}

func recordTypeString(v Value) string {
	o := (*Map)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func recordTypeCopy(v Value, a Allocator) (Value, error) {
	// perform a deep copy of the record even if it is immutable (since the values may be mutable)
	o := (*Map)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Copy(a)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return a.NewRecordValue(c, false)
}

func recordTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	// Function call on selector will be compiled as method call, so we need to process it here.
	o := (*Map)(v.Ptr)
	e, ok := o.Elements[name]
	if !ok {
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
	if !e.IsCallable() {
		return Undefined, fmt.Errorf("%s.%s is not callable, got %s", v.TypeName(), name, e.TypeName())
	}
	return e.Call(vm, args)
}

func recordTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}
	o := (*Map)(v.Ptr)
	r, ok := o.Elements[k]
	if !ok {
		return Undefined, nil
	}
	return r, nil
}

func recordTypeImmutable(v Value, a Allocator) (Value, error) {
	return a.NewRecordValue((*Map)(v.Ptr).Elements, true)
}

/* Immutable Record type specific methods */

func immutableRecordTypeName(v Value) string {
	return "immutable-record"
}

/* Map type specific methods */

func mapTypeName(v Value) string {
	return "map"
}

func mapTypeString(v Value) string {
	o := (*Map)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("map({%s})", strings.Join(pairs, ", "))
}

func mapTypeCopy(v Value, a Allocator) (Value, error) {
	// perform a deep copy of the map even if it is immutable (since the values may be mutable)
	o := (*Map)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Copy(a)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return a.NewMapValue(c, false)
}

func mapTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*Map)(v.Ptr)
	alloc := vm.Allocator()

	switch name {
	case "to_map":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "to_record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewRecordValue(o.Elements, v.Type == VT_IMMUTABLE_MAP)

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(len(o.Elements) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(len(o.Elements))), nil

	case "keys":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return mapFnKeys(v, alloc)

	case "values":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return mapFnValues(v, alloc)

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(genericMapTypeContains(v, args[0])), nil

	case "filter":
		return mapFnFilter(v, vm, args)

	case "count":
		return mapFnCount(v, vm, args)

	case "all":
		return mapFnAll(v, vm, args)

	case "any":
		return mapFnAny(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func mapTypeImmutable(v Value, a Allocator) (Value, error) {
	return a.NewMapValue((*Map)(v.Ptr).Elements, true)
}

func mapTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}

	if mode == OpIndex {
		o := (*Map)(v.Ptr)
		r, ok := o.Elements[k]
		if !ok {
			return Undefined, nil
		}
		return r, nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), k)
}

func mapFnKeys(v Value, a Allocator) (Value, error) {
	o := (*Map)(v.Ptr)
	keys := make([]Value, 0, len(o.Elements))
	for k := range o.Elements {
		t, err := a.NewStringValue(k)
		if err != nil {
			return Undefined, err
		}
		keys = append(keys, t)
	}
	return a.NewArrayValue(keys, false)
}

func mapFnValues(v Value, a Allocator) (Value, error) {
	o := (*Map)(v.Ptr)
	values := make([]Value, 0, len(o.Elements))
	for _, v := range o.Elements {
		values = append(values, v)
	}
	return a.NewArrayValue(values, false)
}

func mapFnFilter(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Map)(v.Ptr)
		filtered := make(map[string]Value, len(o.Elements))
		for k, v := range o.Elements {
			t, err := alloc.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = t
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered[k] = v
			}
		}
		return alloc.NewMapValue(filtered, false)

	case 2:
		o := (*Map)(v.Ptr)
		filtered := make(map[string]Value, len(o.Elements))
		for k, v := range o.Elements {
			t, err := alloc.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = t
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered[k] = v
			}
		}
		return alloc.NewMapValue(filtered, false)

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

func mapFnCount(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Map)(v.Ptr)
		var count int64
		for k := range o.Elements {
			t, err := alloc.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = t
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
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
			t, err := alloc.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = t
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "f/1 or f/2", fn.TypeName())
	}
}

func mapFnAll(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Map)(v.Ptr)
		for k := range o.Elements {
			t, err := alloc.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = t
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	case 2:
		o := (*Map)(v.Ptr)
		for k, v := range o.Elements {
			t, err := alloc.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = t
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "f/1 or f/2", fn.TypeName())
	}
}

func mapFnAny(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Map)(v.Ptr)
		for k := range o.Elements {
			t, err := alloc.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = t
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	case 2:
		o := (*Map)(v.Ptr)
		for k, v := range o.Elements {
			t, err := alloc.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = t
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "f/1 or f/2", fn.TypeName())
	}
}

/* Immutable Map type specific methods */

func immutableMapTypeName(v Value) string {
	return "immutable-map"
}

/* Generic Map type specific methods */

func genericMapTypeInterface(v Value) any {
	o := (*Map)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func genericMapTypeEncodeJSON(v Value) ([]byte, error) {
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

func genericMapTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Map)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Elements); err != nil {
		return nil, fmt.Errorf("map (elements): %w", err)
	}
	return buf.Bytes(), nil
}

func genericMapTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var value map[string]Value
	if err := dec.Decode(&value); err != nil {
		return fmt.Errorf("map (elements): %w", err)
	}
	if value == nil {
		value = make(map[string]Value)
	}
	o := &Map{Elements: value}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func genericMapTypeIsTrue(v Value) bool {
	return len((*Map)(v.Ptr).Elements) > 0
}

func genericMapTypeIterator(v Value, a Allocator) (Value, error) {
	return a.NewMapIteratorValue((*Map)(v.Ptr).Elements)
}

func genericMapTypeEqual(v Value, r Value) bool {
	switch r.Type {
	case VT_MAP, VT_IMMUTABLE_MAP, VT_RECORD, VT_IMMUTABLE_RECORD:
		l := (*Map)(v.Ptr).Elements
		r := (*Map)(r.Ptr).Elements
		if len(l) != len(r) {
			return false
		}
		for k, le := range l {
			re, ok := r[k]
			if !ok {
				return false
			}
			if !le.Equal(re) {
				return false
			}
		}
		return true

	default:
		return false
	}
}

func genericMapTypeLen(v Value) int64 {
	o := (*Map)(v.Ptr)
	return int64(len(o.Elements))
}

func genericMapTypeAssign(v Value, index Value, r Value) error {
	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName())
	}

	(*Map)(v.Ptr).Elements[k] = r

	return nil
}

func genericMapTypeContains(v Value, e Value) bool {
	s, ok := e.AsString()
	if !ok {
		return false
	}
	_, ok = (*Map)(v.Ptr).Elements[s]
	return ok
}

func genericMapTypeDelete(v Value, key Value) (Value, error) {
	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName())
	}
	delete((*Map)(v.Ptr).Elements, s)
	return v, nil
}

func genericMapTypeAsBool(v Value) (bool, bool) {
	return len((*Map)(v.Ptr).Elements) > 0, true
}

func genericMapTypeAsString(v Value) (string, bool) {
	return v.String(), true
}

func genericMapTypeAsMap(v Value, a Allocator) (map[string]Value, bool) {
	return (*Map)(v.Ptr).Elements, true
}
