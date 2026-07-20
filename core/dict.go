package core

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"

	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
	"github.com/jokruger/kavun/internal/format"
)

const (
	dictTypeName          = "dict"
	immutableDictTypeName = "immutable-dict"
)

type Dict struct {
	Elements map[string]Value
}

func (o *Dict) Set(elements map[string]Value) {
	o.Elements = elements
}

// sortedKeys returns the dict's keys in a deterministic (lexical) order. Go randomizes map iteration order per
// range, so any hook whose output is order-sensitive (String, EncodeJSON, EncodeBinary, the keys()/values() methods)
// must range in this order instead of ranging over o.Elements directly, or it would return a different result on
// every call for the exact same receiver — violating the purity contract (see docs/purity.md) with zero arguments
// involved.
func (o *Dict) sortedKeys() []string {
	keys := make([]string, 0, len(o.Elements))
	for k := range o.Elements {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func NewDictValue(m map[string]Value, immutable bool) Value {
	o := &Dict{Elements: m}
	return Value{Type: value.Dict, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeDict = ValueTypeDescr{
	Name:         SeqNameHook(dictTypeName, immutableDictTypeName), // PURE by contract
	String:       dictTypeString,                                   // PURE by contract
	Format:       dictTypeFormat,                                   // PURE by contract
	Interface:    dictTypeInterface,                                // PURE by contract
	EncodeJSON:   dictTypeEncodeJSON,                               // PURE by contract
	EncodeBinary: dictTypeEncodeBinary,                             // PURE by contract
	DecodeBinary: dictTypeDecodeBinary,                             // IMPURE by contract (mutates target)
	IsTrue:       dictTypeIsTrue,                                   // PURE by contract
	IsIterable:   ConstHook(true),                                  // PURE by contract
	Iterator:     dictTypeIterator,                                 // PURE by contract (constructs fresh iterator)
	Equal:        dictTypeEqual,                                    // PURE by contract
	Clone:        dictTypeClone,                                    // PURE by contract
	Len:          dictTypeLen,                                      // PURE by contract
	MethodCall:   dictTypeMethodCall,                               // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       dictTypeAccess,                                   // PURE by contract
	Assign:       dictTypeAssign,                                   // IMPURE by contract
	Contains:     dictTypeContains,                                 // PURE by contract
	Delete:       dictTypeDelete,                                   // IMPURE by contract
	AsBool:       dictTypeAsBool,                                   // PURE by contract
	AsString:     dictTypeAsString,                                 // PURE by contract
	AsDict:       dictTypeAsDict,                                   // PURE by contract

	// No _in_place methods. Higher-order methods (filter/for_each/all/any/find/count) are gated the same way as
	// array's. All methods are expected to be pure.
	IsMethodPure: func(string) bool { return true },
}

func dictTypeString(v Value) string {
	o := (*Dict)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for _, k := range o.sortedKeys() {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, o.Elements[k].String()))
	}
	return fmt.Sprintf("dict({%s})", strings.Join(pairs, ", "))
}

func dictTypeInterface(v Value) any {
	o := (*Dict)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func dictTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)
	var b []byte
	b = append(b, '{')
	keys := o.sortedKeys()
	len1 := len(keys) - 1
	for idx, key := range keys {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := o.Elements[key].EncodeJSON()
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
	}
	b = append(b, '}')
	return b, nil
}

func dictTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for _, key := range o.sortedKeys() {
		b = binary.AppendBytes(b, []byte(key))
		eb, err := o.Elements[key].EncodeBinary()
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
		}
		b = binary.AppendBytes(b, eb)
	}
	return b, nil
}

func dictTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	count, err := binary.ReadUint64(data, &offset, "dict (elements count)")
	if err != nil {
		return err
	}

	value := make(map[string]Value, int(count))
	for i := 0; i < int(count); i++ {
		kb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("dict key at index %d", i))
		if err != nil {
			return err
		}
		key := string(kb)
		eb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("dict value at key %q", key))
		if err != nil {
			return err
		}
		var element Value
		if err := element.DecodeBinary(eb); err != nil {
			return fmt.Errorf("dict value at key %q: %w", key, err)
		}
		value[key] = element
	}
	if offset != len(data) {
		return fmt.Errorf("dict: trailing %d bytes", len(data)-offset)
	}

	*v = NewDictValue(value, v.Immutable)

	return nil
}

func dictTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return dictTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(dictTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(dictTypeString(v), sp, fspec.AlignLeft), nil
}

func dictTypeClone(v Value) (Value, error) {
	// Deep copy the dict (and make it mutable) and its elements
	o := (*Dict)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Clone()
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return NewDictValue(c, false), nil
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func dictTypeIterator(v Value) (Value, error) {
	return NewDictIteratorValue((*Dict)(v.Ptr).Elements), nil
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func dictTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictTypeClone(v)

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewRecordValue(o.Elements, v.Immutable), nil

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := dictTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

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
		return dictFnKeys(v)

	case "values":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictFnValues(v)

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(dictTypeContains(v, args[0])), nil

	case "filter":
		return dictFnFilter(vm, v, args)

	case "count":
		return dictFnCount(vm, v, args)

	case "all":
		return dictFnAll(vm, v, args)

	case "any":
		return dictFnAny(vm, v, args)

	case "for_each":
		return dictFnForEach(vm, v, args)

	case "find":
		return dictFnFind(vm, v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

// PURE by contract
func dictTypeAccess(v Value, index Value, mode bc.Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}

	if mode == bc.AccessIndex {
		o := (*Dict)(v.Ptr)
		r, ok := o.Elements[k]
		if !ok {
			return Undefined, nil
		}
		return r, nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), k)
}

func dictFnKeys(v Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	sorted := o.sortedKeys()
	keys := make([]Value, 0, len(sorted))
	for _, k := range sorted {
		keys = append(keys, NewStringValue(k))
	}
	return NewArrayValue(keys, false), nil
}

func dictFnValues(v Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	sorted := o.sortedKeys()
	values := make([]Value, 0, len(sorted))
	for _, k := range sorted {
		values = append(values, o.Elements[k])
	}
	return NewArrayValue(values, false), nil
}

func dictFnFilter(vm VM, v Value, args []Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "0 or 1", len(args))
	}

	o := (*Dict)(v.Ptr)
	filtered := make(map[string]Value, len(o.Elements))

	if len(args) == 0 {
		for k, v := range o.Elements {
			if v.Type != value.Undefined {
				filtered[k] = v
			}
		}
		return NewDictValue(filtered, false), nil
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value

	switch fn.Arity() {
	case 1:
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered[k] = v
			}
		}
		return NewDictValue(filtered, false), nil

	case 2:
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered[k] = v
			}
		}
		return NewDictValue(filtered, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictFnCount(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Dict)(v.Ptr)
		var count int64
		for k := range o.Elements {
			buf[0] = NewStringValue(k)
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
			buf[0] = NewStringValue(k)
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

func dictFnForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	o := (*Dict)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for k := range o.Elements {
			buf[0] = NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return Undefined, nil
			}
		}

	case 2:
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return Undefined, nil
			}
		}
	}
	return Undefined, nil
}

func dictFnFind(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Dict)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for k := range o.Elements {
			nv := NewStringValue(k)
			buf[0] = nv
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return nv, nil
			}
		}
		return Undefined, nil

	case 2:
		for k, v := range o.Elements {
			nv := NewStringValue(k)
			buf[0] = nv
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return nv, nil
			}
		}
		return Undefined, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictFnAll(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Dict)(v.Ptr)
		for k := range o.Elements {
			buf[0] = NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return False, nil
			}
		}
		return True, nil

	case 2:
		o := (*Dict)(v.Ptr)
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return False, nil
			}
		}
		return True, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictFnAny(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Dict)(v.Ptr)
		for k := range o.Elements {
			buf[0] = NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return True, nil
			}
		}
		return False, nil

	case 2:
		o := (*Dict)(v.Ptr)
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return True, nil
			}
		}
		return False, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictTypeIsTrue(v Value) bool {
	return len((*Dict)(v.Ptr).Elements) > 0
}

func dictTypeEqual(v Value, rv Value) bool {
	var r map[string]Value
	switch rv.Type {
	case value.Dict:
		r = (*Dict)(rv.Ptr).Elements
	case value.Record:
		r = (*Record)(rv.Ptr).Elements
	default:
		return false
	}

	l := (*Dict)(v.Ptr).Elements
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
}

func dictTypeLen(v Value) int64 {
	o := (*Dict)(v.Ptr)
	return int64(len(o.Elements))
}

// IMPURE: writes into the receiver. Not folded by the optimizer. See docs/purity.md.
func dictTypeAssign(v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName())
	}

	(*Dict)(v.Ptr).Elements[k] = r

	return nil
}

func dictTypeContains(v Value, e Value) bool {
	s, ok := e.AsString()
	if !ok {
		return false
	}
	_, ok = (*Dict)(v.Ptr).Elements[s]
	return ok
}

// IMPURE: removes an entry from the receiver. Not folded by the optimizer. See docs/purity.md.
func dictTypeDelete(v Value, key Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotDeletableError(v.TypeName())
	}

	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName())
	}
	delete((*Dict)(v.Ptr).Elements, s)
	return v, nil
}

func dictTypeAsBool(v Value) (bool, bool) {
	return len((*Dict)(v.Ptr).Elements) > 0, true
}

func dictTypeAsString(v Value) (string, bool) {
	return v.String(), true
}

func dictTypeAsDict(v Value) (map[string]Value, bool) {
	return (*Dict)(v.Ptr).Elements, true
}
