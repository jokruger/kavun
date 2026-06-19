package core

import (
	"fmt"
	"strings"

	"github.com/jokruger/kavun/core/opcode"
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

var TypeDict = ValueTypeDescr{
	Name:         SeqNameHook(dictTypeName, immutableDictTypeName),
	String:       dictTypeString,
	Format:       dictTypeFormat,
	Interface:    dictTypeInterface,
	EncodeJSON:   dictTypeEncodeJSON,
	EncodeBinary: dictTypeEncodeBinary,
	DecodeBinary: dictTypeDecodeBinary,
	IsTrue:       dictTypeIsTrue,
	IsIterable:   ConstHook(true),
	Iterator:     dictTypeIterator,
	Equal:        dictTypeEqual,
	Clone:        dictTypeClone,
	Len:          dictTypeLen,
	MethodCall:   dictTypeMethodCall,
	Access:       dictTypeAccess,
	Assign:       dictTypeAssign,
	Contains:     dictTypeContains,
	Delete:       dictTypeDelete,
	AsBool:       dictTypeAsBool,
	AsString:     dictTypeAsString,
	AsDict:       dictTypeAsDict,
}

func dictTypeString(a *Arena, v Value) string {
	o := a.ResolveDictValue(v)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String(a)))
	}
	return fmt.Sprintf("dict({%s})", strings.Join(pairs, ", "))
}

func dictTypeInterface(a *Arena, v Value) any {
	o := a.ResolveDictValue(v)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface(a)
	}
	return res
}

func dictTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveDictValue(v)
	var b []byte
	b = append(b, '{')
	len1 := len(o.Elements) - 1
	idx := 0
	for key, value := range o.Elements {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := value.EncodeJSON(a)
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

func dictTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveDictValue(v)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for key, value := range o.Elements {
		b = binary.AppendBytes(b, []byte(key))
		eb, err := value.EncodeBinary(a)
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
		}
		b = binary.AppendBytes(b, eb)
	}
	return b, nil
}

func dictTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
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
		if err := element.DecodeBinary(a, eb); err != nil {
			return fmt.Errorf("dict value at key %q: %w", key, err)
		}
		value[key] = element
	}
	if offset != len(data) {
		return fmt.Errorf("dict: trailing %d bytes", len(data)-offset)
	}

	o, err := a.NewDictValue(value, v.Immutable)
	if err != nil {
		return err
	}
	// we are not releasing old value here because it should be managed by caller Value.DecodeBinary
	*v = o

	return nil
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
		a.PinAny(t)
		c[k] = t
	}
	return a.NewDictValue(c, false)
}

func dictTypeIterator(a *Arena, v Value) (Value, error) {
	return a.NewDictIteratorValue(a.ResolveDictValue(v).Elements)
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
		a.RetainAny(v)
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
		return BoolValue(dictTypeContains(a, v, args[0])), nil

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
		a.PinAllocated(nv)
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
			if v.Type != value.Undefined {
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
			a.ReleaseAllocated(nv)
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
			a.ReleaseAllocated(nv)
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
			a.ReleaseAllocated(nv)
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
			a.ReleaseAllocated(nv)
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
			a.ReleaseAllocated(nv)
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
			a.ReleaseAllocated(nv)
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
				a.ReleaseAllocated(nv)
				return Undefined, err
			}
			if res.IsTrue(a) {
				return nv, nil
			}
			a.ReleaseAllocated(nv)
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
				a.ReleaseAllocated(nv)
				return Undefined, err
			}
			if res.IsTrue(a) {
				return nv, nil
			}
			a.ReleaseAllocated(nv)
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
			a.ReleaseAllocated(nv)
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
			a.ReleaseAllocated(nv)
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
			a.ReleaseAllocated(nv)
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
			a.ReleaseAllocated(nv)
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

func dictTypeIsTrue(a *Arena, v Value) bool {
	return len(a.ResolveDictValue(v).Elements) > 0
}

func dictTypeEqual(a *Arena, v Value, rv Value) bool {
	var r map[string]Value
	switch rv.Type {
	case value.Dict:
		r = a.ResolveDictValue(rv).Elements
	case value.Record:
		r = a.ResolveRecordValue(rv).Elements
	default:
		return false
	}

	l := a.ResolveDictValue(v).Elements
	if len(l) != len(r) {
		return false
	}
	for k, le := range l {
		re, ok := r[k]
		if !ok {
			return false
		}
		if !le.Equal(a, re) {
			return false
		}
	}

	return true
}

func dictTypeLen(a *Arena, v Value) int64 {
	o := a.ResolveDictValue(v)
	return int64(len(o.Elements))
}

func dictTypeAssign(a *Arena, v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName(a))
	}

	k, ok := index.AsString(a)
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName(a))
	}

	a.PinAny(r) // §5: container takes pinned ownership of the value.
	a.ResolveDictValue(v).Elements[k] = r

	return nil
}

func dictTypeContains(a *Arena, v Value, e Value) bool {
	s, ok := e.AsString(a)
	if !ok {
		return false
	}
	_, ok = a.ResolveDictValue(v).Elements[s]
	return ok
}

func dictTypeDelete(a *Arena, v Value, key Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotDeletableError(v.TypeName(a))
	}

	s, ok := key.AsString(a)
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName(a))
	}
	delete(a.ResolveDictValue(v).Elements, s)
	return v, nil
}

func dictTypeAsBool(a *Arena, v Value) (bool, bool) {
	return len(a.ResolveDictValue(v).Elements) > 0, true
}

func dictTypeAsString(a *Arena, v Value) (string, bool) {
	return v.String(a), true
}

func dictTypeAsDict(a *Arena, v Value) (map[string]Value, bool) {
	return a.ResolveDictValue(v).Elements, true
}
