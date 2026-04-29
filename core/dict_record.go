package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
	"unsafe"

	"github.com/jokruger/kavun/errs"
)

type Dict struct {
	Elements map[string]Value
}

func (o *Dict) Set(elements map[string]Value) {
	o.Elements = elements
}

// RecordValue creates new boxed record value.
func RecordValue(v *Dict, immutable bool) Value {
	return Value{
		Type:  VT_RECORD,
		Const: immutable,
		Ptr:   unsafe.Pointer(v),
	}
}

// DictValue creates new boxed dict value.
func DictValue(v *Dict, immutable bool) Value {
	return Value{
		Type:  VT_DICT,
		Const: immutable,
		Ptr:   unsafe.Pointer(v),
	}
}

// NewRecordValue creates new (heap-allocated) record value.
func NewRecordValue(vals map[string]Value, immutable bool) Value {
	t := &Dict{}
	t.Set(vals)
	return RecordValue(t, immutable)
}

// NewDictValue creates new (heap-allocated) dict value.
func NewDictValue(vals map[string]Value, immutable bool) Value {
	t := &Dict{}
	t.Set(vals)
	return DictValue(t, immutable)
}

/* Record type specific methods */

func recordTypeName(v Value) string {
	if v.Const {
		return "immutable-record"
	}
	return "record"
}

func recordTypeString(v Value) string {
	o := (*Dict)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func recordTypeCopy(v Value, a *Arena) (Value, error) {
	// perform a deep copy of the record even if it is immutable (since the values may be mutable)
	o := (*Dict)(v.Ptr)
	c := a.NewDict(len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Copy(a)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return a.NewRecordValue(c, false), nil
}

func recordTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	// Function call on selector will be compiled as method call, so we need to process it here.
	o := (*Dict)(v.Ptr)
	e, ok := o.Elements[name]
	if !ok {
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
	if !e.IsCallable() {
		return Undefined, fmt.Errorf("%s.%s is not callable, got %s", v.TypeName(), name, e.TypeName())
	}
	return e.Call(vm, args)
}

func recordTypeAccess(v Value, a *Arena, index Value, mode Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}
	o := (*Dict)(v.Ptr)
	r, ok := o.Elements[k]
	if !ok {
		return Undefined, nil
	}
	return r, nil
}

/* Dict type specific methods */

func dictTypeName(v Value) string {
	if v.Const {
		return "immutable-dict"
	}
	return "dict"
}

func dictTypeString(v Value) string {
	o := (*Dict)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("dict({%s})", strings.Join(pairs, ", "))
}

func dictTypeCopy(v Value, a *Arena) (Value, error) {
	// perform a deep copy of the dict even if it is immutable (since the values may be mutable)
	o := (*Dict)(v.Ptr)
	c := a.NewDict(len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Copy(a)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return a.NewDictValue(c, false), nil
}

func dictTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	alloc := vm.Allocator()

	switch name {
	case "to_dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "to_record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewRecordValue(o.Elements, v.Const), nil

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
		return dictFnKeys(v, alloc)

	case "values":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictFnValues(v, alloc)

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(genericDictTypeContains(v, args[0])), nil

	case "filter":
		return dictFnFilter(v, vm, args)

	case "count":
		return dictFnCount(v, vm, args)

	case "all":
		return dictFnAll(v, vm, args)

	case "any":
		return dictFnAny(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func dictTypeAccess(v Value, a *Arena, index Value, mode Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}

	if mode == OpIndex {
		o := (*Dict)(v.Ptr)
		r, ok := o.Elements[k]
		if !ok {
			return Undefined, nil
		}
		return r, nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), k)
}

func dictFnKeys(v Value, a *Arena) (Value, error) {
	o := (*Dict)(v.Ptr)
	keys := a.NewArray(len(o.Elements), false)
	for k := range o.Elements {
		t := a.NewStringValue(k)
		keys = append(keys, t)
	}
	return a.NewArrayValue(keys, false), nil
}

func dictFnValues(v Value, a *Arena) (Value, error) {
	o := (*Dict)(v.Ptr)
	values := a.NewArray(len(o.Elements), false)
	for _, v := range o.Elements {
		values = append(values, v)
	}
	return a.NewArrayValue(values, false), nil
}

func dictFnFilter(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	alloc := vm.Allocator()
	o := (*Dict)(v.Ptr)
	filtered := alloc.NewDict(len(o.Elements))

	switch fn.Arity() {
	case 1:
		for k, v := range o.Elements {
			t := alloc.NewStringValue(k)
			buf[0] = t
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered[k] = v
			}
		}
		return alloc.NewDictValue(filtered, false), nil

	case 2:
		for k, v := range o.Elements {
			t := alloc.NewStringValue(k)
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
		return alloc.NewDictValue(filtered, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictFnCount(v Value, vm VM, args []Value) (Value, error) {
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
		o := (*Dict)(v.Ptr)
		var count int64
		for k := range o.Elements {
			t := alloc.NewStringValue(k)
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
		o := (*Dict)(v.Ptr)
		var count int64
		for k, v := range o.Elements {
			t := alloc.NewStringValue(k)
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

func dictFnAll(v Value, vm VM, args []Value) (Value, error) {
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
		o := (*Dict)(v.Ptr)
		for k := range o.Elements {
			t := alloc.NewStringValue(k)
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
		o := (*Dict)(v.Ptr)
		for k, v := range o.Elements {
			t := alloc.NewStringValue(k)
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

func dictFnAny(v Value, vm VM, args []Value) (Value, error) {
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
		o := (*Dict)(v.Ptr)
		for k := range o.Elements {
			t := alloc.NewStringValue(k)
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
		o := (*Dict)(v.Ptr)
		for k, v := range o.Elements {
			t := alloc.NewStringValue(k)
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

/* Generic Dict type specific methods */

func genericDictTypeInterface(v Value) any {
	o := (*Dict)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func genericDictTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)
	var b []byte
	b = append(b, '{')
	len1 := len(o.Elements) - 1
	idx := 0
	for key, value := range o.Elements {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := value.EncodeJSON()
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
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

func genericDictTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Elements); err != nil {
		return nil, fmt.Errorf("dict (elements): %w", err)
	}
	return buf.Bytes(), nil
}

func genericDictTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var value map[string]Value
	if err := dec.Decode(&value); err != nil {
		return fmt.Errorf("dict (elements): %w", err)
	}
	if value == nil {
		value = make(map[string]Value)
	}
	o := &Dict{Elements: value}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func genericDictTypeIsTrue(v Value) bool {
	return len((*Dict)(v.Ptr).Elements) > 0
}

func genericDictTypeIterator(v Value, a *Arena) (Value, error) {
	return a.NewDictIteratorValue((*Dict)(v.Ptr).Elements), nil
}

func genericDictTypeEqual(v Value, r Value) bool {
	switch r.Type {
	case VT_DICT, VT_RECORD:
		l := (*Dict)(v.Ptr).Elements
		r := (*Dict)(r.Ptr).Elements
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

func genericDictTypeLen(v Value) int64 {
	o := (*Dict)(v.Ptr)
	return int64(len(o.Elements))
}

func genericDictTypeAssign(v Value, index Value, r Value) error {
	if v.Const {
		return errs.NewNotAssignableError(v.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName())
	}

	(*Dict)(v.Ptr).Elements[k] = r

	return nil
}

func genericDictTypeContains(v Value, e Value) bool {
	s, ok := e.AsString()
	if !ok {
		return false
	}
	_, ok = (*Dict)(v.Ptr).Elements[s]
	return ok
}

func genericDictTypeDelete(v Value, key Value) (Value, error) {
	if v.Const {
		return Undefined, errs.NewInvalidDeleteError(v.TypeName())
	}

	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName())
	}
	delete((*Dict)(v.Ptr).Elements, s)
	return v, nil
}

func genericDictTypeAsBool(v Value) (bool, bool) {
	return len((*Dict)(v.Ptr).Elements) > 0, true
}

func genericDictTypeAsString(v Value) (string, bool) {
	return v.String(), true
}

func genericDictTypeAsDict(v Value, a *Arena) (map[string]Value, bool) {
	return (*Dict)(v.Ptr).Elements, true
}
