package core

import (
	"fmt"
	"strings"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/opcode"
)

const (
	dictTypeName          = "dict"
	immutableDictTypeName = "immutable-dict"
)

var TypeDict = ValueTypeDescr{
	Pin:          func(a *Arena, v Value) { a.PinDictValue(v) },
	Retain:       func(a *Arena, v Value) { a.RetainDictValue(v) },
	Release:      func(a *Arena, v Value) { a.ReleaseDictValue(v) },
	Name:         SeqNameHook(dictTypeName, immutableDictTypeName),
	String:       dictTypeString,
	Format:       dictTypeFormat,
	Interface:    DictInterface,
	EncodeJSON:   DictEncodeJSON,
	EncodeBinary: DictEncodeBinary,
	DecodeBinary: DictDecodeBinary,
	IsTrue:       DictIsTrue,
	IsIterable:   ConstHook(true),
	Iterator:     func(a *Arena, v Value) (Value, error) { return a.NewDictIteratorValue(a.ResolveDictValue(v).Elements) },
	Equal:        DictEqual,
	Clone:        dictTypeClone,
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

func dictTypeString(a *Arena, v Value) string {
	o := a.ResolveDictValue(v)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String(a)))
	}
	return fmt.Sprintf("dict({%s})", strings.Join(pairs, ", "))
}

func dictTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return dictTypeString(a, v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(a), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(dictTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(dictTypeString(a, v), sp, fspec.AlignLeft), nil
}

func dictTypeClone(a *Arena, v Value) (Value, error) {
	// Deep copy the dict (and make it mutable) and its elements
	o := a.ResolveDictValue(v)
	c := a.NewDict(len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Clone(a)
		if err != nil {
			return Undefined, err
		}
		t.Pin(a)
		c[k] = t
	}
	return a.NewDictValue(c, false)
}

func dictTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	o := a.ResolveDictValue(v)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictTypeClone(a, v)

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewRecordValue(o.Elements, v.Immutable)

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName(a))
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := dictTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s)

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
		return dictFnKeys(a, v)

	case "values":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictFnValues(a, v)

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(DictContains(a, v, args[0])), nil

	case "filter":
		return dictFnFilter(a, vm, v, args)

	case "count":
		return dictFnCount(a, vm, v, args)

	case "all":
		return dictFnAll(a, vm, v, args)

	case "any":
		return dictFnAny(a, vm, v, args)

	case "for_each":
		return dictFnForEach(a, vm, v, args)

	case "find":
		return dictFnFind(a, vm, v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}

func dictTypeAccess(a *Arena, v Value, index Value, mode opcode.Opcode) (Value, error) {
	k, ok := index.AsString(a)
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName(a))
	}

	if mode == opcode.Index {
		o := a.ResolveDictValue(v)
		r, ok := o.Elements[k]
		if !ok {
			return Undefined, nil
		}
		return r, nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(a), k)
}

func dictFnKeys(a *Arena, v Value) (Value, error) {
	o := a.ResolveDictValue(v)
	keys := a.NewArray(len(o.Elements), false)
	for k := range o.Elements {
		nv, err := a.NewStringValue(k)
		if err != nil {
			return Undefined, err
		}
		nv.Pin(a)
		keys = append(keys, nv)
	}
	return a.NewArrayValue(keys, false)
}

func dictFnValues(a *Arena, v Value) (Value, error) {
	o := a.ResolveDictValue(v)
	values := a.NewArray(len(o.Elements), false)
	for _, v := range o.Elements {
		values = append(values, v)
	}
	return a.NewArrayValue(values, false)
}

func dictFnFilter(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "0 or 1", len(args))
	}

	o := a.ResolveDictValue(v)
	filtered := a.NewDict(len(o.Elements))

	if len(args) == 0 {
		for k, v := range o.Elements {
			if v.Type != VT_UNDEFINED {
				filtered[k] = v
			}
		}
		return a.NewDictValue(filtered, false)
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName(a))
	}

	var buf [2]Value

	switch fn.Arity(a) {
	case 1:
		for k, v := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			res, err := fn.Call(a, vm, buf[:1])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				filtered[k] = v
			}
		}
		return a.NewDictValue(filtered, false)

	case 2:
		for k, v := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			buf[1] = v
			res, err := fn.Call(a, vm, buf[:2])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				filtered[k] = v
			}
		}
		return a.NewDictValue(filtered, false)

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName(a))
	}
}

func dictFnCount(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName(a))
	}

	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		o := a.ResolveDictValue(v)
		var count int64
		for k := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			res, err := fn.Call(a, vm, buf[:1])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				count++
			}
		}
		return IntValue(count), nil

	case 2:
		o := a.ResolveDictValue(v)
		var count int64
		for k, v := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			buf[1] = v
			res, err := fn.Call(a, vm, buf[:2])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				count++
			}
		}
		return IntValue(count), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "f/1 or f/2", fn.TypeName(a))
	}
}

func dictFnForEach(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(a, args)
	if err != nil {
		return Undefined, err
	}

	o := a.ResolveDictValue(v)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for k := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			res, err := fn.Call(a, vm, buf[:1])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue(a) {
				return Undefined, nil
			}
		}

	case 2:
		for k, v := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			buf[1] = v
			res, err := fn.Call(a, vm, buf[:2])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue(a) {
				return Undefined, nil
			}
		}
	}
	return Undefined, nil
}

func dictFnFind(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName(a))
	}

	o := a.ResolveDictValue(v)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for k := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			res, err := fn.Call(a, vm, buf[:1])
			if err != nil {
				a.ReleaseStringValue(nv)
				return Undefined, err
			}
			if res.IsTrue(a) {
				return nv, nil
			}
			a.ReleaseStringValue(nv)
		}
		return Undefined, nil

	case 2:
		for k, v := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			buf[1] = v
			res, err := fn.Call(a, vm, buf[:2])
			if err != nil {
				a.ReleaseStringValue(nv)
				return Undefined, err
			}
			if res.IsTrue(a) {
				return nv, nil
			}
			a.ReleaseStringValue(nv)
		}
		return Undefined, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName(a))
	}
}

func dictFnAll(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName(a))
	}

	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		o := a.ResolveDictValue(v)
		for k := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			res, err := fn.Call(a, vm, buf[:1])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue(a) {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	case 2:
		o := a.ResolveDictValue(v)
		for k, v := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			buf[1] = v
			res, err := fn.Call(a, vm, buf[:2])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue(a) {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "f/1 or f/2", fn.TypeName(a))
	}
}

func dictFnAny(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName(a))
	}

	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		o := a.ResolveDictValue(v)
		for k := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			res, err := fn.Call(a, vm, buf[:1])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	case 2:
		o := a.ResolveDictValue(v)
		for k, v := range o.Elements {
			nv, err := a.NewStringValue(k)
			if err != nil {
				return Undefined, err
			}
			buf[0] = nv
			buf[1] = v
			res, err := fn.Call(a, vm, buf[:2])
			a.ReleaseStringValue(nv)
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "f/1 or f/2", fn.TypeName(a))
	}
}
