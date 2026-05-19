package core

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/format"
)

const (
	dictTypeName          = "dict"
	immutableDictTypeName = "immutable-dict"
)

// DictValue creates new boxed dict value.
func DictValue(v *Dict, immutable bool) Value {
	return Value{
		Type:      VT_DICT,
		Immutable: immutable,
		Ptr:       unsafe.Pointer(v),
	}
}

// NewDictValue creates new (heap-allocated) dict value.
func NewDictValue(vals map[string]Value, immutable bool) Value {
	t := &Dict{}
	t.Set(vals)
	return DictValue(t, immutable)
}

var TypeDict = ValueType{
	Name:         SeqTypeNameHook(dictTypeName, immutableDictTypeName),
	String:       dictTypeString,
	Format:       dictTypeFormat,
	Interface:    DictInterface,
	EncodeJSON:   DictEncodeJSON,
	EncodeBinary: DictEncodeBinary,
	DecodeBinary: DictDecodeBinary,
	IsTrue:       DictIsTrue,
	IsIterable:   ConstHook(true),
	Iterator:     func(v Value, a *Arena) (Value, error) { return a.NewDictIteratorValue((*Dict)(v.Ptr).Elements), nil },
	Equal:        DictEqual,
	Copy:         dictTypeCopy,
	Len:          DictLen,
	MethodCall:   dictTypeMethodCall,
	Access:       dictTypeAccess,
	Assign:       DictAssign,
	Contains:     DictContains,
	Delete:       DictDelete,
	AsBool:       DictAsBool,
	AsString:     DictAsString,
	AsDict:       DictAsDict,
}

func dictTypeString(v Value) string {
	o := (*Dict)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("dict({%s})", strings.Join(pairs, ", "))
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

func dictTypeCopy(v Value, a *Arena) (Value, error) {
	// Deep copy the dict (and make it mutable) and its elements
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
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictTypeCopy(v, alloc)

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewRecordValue(o.Elements, v.Immutable), nil

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
		return alloc.NewStringValue(s), nil

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
		return BoolValue(DictContains(v, args[0])), nil

	case "filter":
		return dictFnFilter(v, vm, args)

	case "count":
		return dictFnCount(v, vm, args)

	case "all":
		return dictFnAll(v, vm, args)

	case "any":
		return dictFnAny(v, vm, args)

	case "for_each":
		return dictFnForEach(v, vm, args)

	case "find":
		return dictFnFind(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func dictTypeAccess(v Value, a *Arena, index Value, mode bc.Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}

	if mode == bc.OpIndex {
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
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "0 or 1", len(args))
	}

	o := (*Dict)(v.Ptr)
	alloc := vm.Allocator()
	filtered := alloc.NewDict(len(o.Elements))

	if len(args) == 0 {
		for k, v := range o.Elements {
			if v.Type != VT_UNDEFINED {
				filtered[k] = v
			}
		}
		return alloc.NewDictValue(filtered, false), nil
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value

	switch fn.Arity() {
	case 1:
		for k, v := range o.Elements {
			buf[0] = alloc.NewStringValue(k)
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
			buf[0] = alloc.NewStringValue(k)
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
			buf[0] = alloc.NewStringValue(k)
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
			buf[0] = alloc.NewStringValue(k)
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

func dictFnForEach(v Value, vm VM, args []Value) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	alloc := vm.Allocator()
	o := (*Dict)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for k := range o.Elements {
			buf[0] = alloc.NewStringValue(k)
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
			buf[0] = alloc.NewStringValue(k)
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

func dictFnFind(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	o := (*Dict)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for k := range o.Elements {
			t := alloc.NewStringValue(k)
			buf[0] = t
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return t, nil
			}
		}
		return Undefined, nil

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
				return t, nil
			}
		}
		return Undefined, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName())
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
			buf[0] = alloc.NewStringValue(k)
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
			buf[0] = alloc.NewStringValue(k)
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
			buf[0] = alloc.NewStringValue(k)
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
			buf[0] = alloc.NewStringValue(k)
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
