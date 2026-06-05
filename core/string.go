package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/araddon/dateparse"
	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/token"
)

const stringTypeName = "string"

// String is a string type envelope.
type String struct {
	Value string
}

func (o *String) Set(s string) {
	o.Value = s
}

// StringValue creates new boxed string value.
func StringValue(v *String) Value {
	return Value{
		Type:      VT_STRING,
		Immutable: true,
		Ptr:       unsafe.Pointer(v),
	}
}

// NewStringValue creates new (heap-allocated) string value.
func NewStringValue(v string) Value {
	o := &String{}
	o.Set(v)
	return StringValue(o)
}

// TypeString is a string type descriptor.
var TypeString = ValueType{
	Name:         ConstHook(stringTypeName),
	String:       func(a *Arena, v Value) string { return strconv.Quote((*String)(v.Ptr).Value) },
	Format:       stringTypeFormat,
	Interface:    func(a *Arena, v Value) any { return (*String)(v.Ptr).Value },
	EncodeJSON:   stringTypeEncodeJSON,
	EncodeBinary: stringTypeEncodeBinary,
	DecodeBinary: stringTypeDecodeBinary,
	IsTrue:       func(a *Arena, v Value) bool { return len((*String)(v.Ptr).Value) > 0 },
	IsIterable:   ConstHook(true),
	Iterator:     stringTypeIterator,
	Equal:        stringTypeEqual,
	Len:          func(a *Arena, v Value) int64 { return int64(len((*String)(v.Ptr).Value)) },
	BinaryOp:     stringTypeBinaryOp,
	MethodCall:   stringTypeMethodCall,
	Access:       stringTypeAccess,
	Contains:     stringTypeContains,
	Slice:        stringTypeSlice,
	SliceStep:    stringTypeSliceStep,
	AsBool:       func(a *Arena, v Value) (bool, bool) { return conv.ParseBool((*String)(v.Ptr).Value) },
	AsInt:        stringTypeAsInt,
	AsByte:       stringTypeAsByte,
	AsFloat:      stringTypeAsFloat,
	AsDecimal:    stringTypeAsDecimal,
	AsTime:       stringTypeAsTime,
	AsString:     func(a *Arena, v Value) (string, bool) { return (*String)(v.Ptr).Value, true },
	AsRunes:      func(a *Arena, v Value) ([]rune, bool) { return []rune((*String)(v.Ptr).Value), true },
	AsBytes:      func(a *Arena, v Value) ([]byte, bool) { return []byte((*String)(v.Ptr).Value), true },
	AsArray:      stringTypeAsArray,
}

func stringTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := (*String)(v.Ptr)
	var b []byte
	b = EncodeString(b, o.Value)
	return b, nil
}

func stringTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := (*String)(v.Ptr)
	s := o.Value
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return nil, fmt.Errorf("string: %w", err)
	}
	return buf.Bytes(), nil
}

func stringTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("string: %w", err)
	}
	o := &String{Value: s}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func stringTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return strconv.Quote((*String)(v.Ptr).Value), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(stringTypeName, sp, fspec.AlignLeft), nil
	}
	o := (*String)(v.Ptr)
	return format.FormatStringLike(stringTypeName, sp, o.Value, false)
}

func stringTypeBinaryOp(a *Arena, v Value, rhs Value, op token.Token) (Value, error) {
	r, ok := rhs.AsString(a)
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
	}

	o := (*String)(v.Ptr)
	switch op {
	case token.Add:
		return a.NewStringValue(o.Value + r), nil
	case token.Less:
		return BoolValue(o.Value < r), nil
	case token.LessEq:
		return BoolValue(o.Value <= r), nil
	case token.Greater:
		return BoolValue(o.Value > r), nil
	case token.GreaterEq:
		return BoolValue(o.Value >= r), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
}

func stringTypeEqual(a *Arena, v Value, r Value) bool {
	t, ok := r.AsString(a)
	if !ok {
		return false
	}
	o := (*String)(v.Ptr)
	return o.Value == t
}

func stringTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*String)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewBytesValue([]byte(o.Value), false), nil

	case "runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := a.NewRunes(utf8.RuneCountInString(o.Value), true)
		for i, r := range o.Value {
			rs[i] = r
		}
		return a.NewRunesValue(rs, false), nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := stringTypeAsArray(a, v)
		return a.NewArrayValue(t, false), nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := conv.ParseBool((*String)(v.Ptr).Value)
		return BoolValue(b), nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := stringTypeAsFloat(a, v)
		return FloatValue(f), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := stringTypeAsInt(a, v)
		return IntValue(i), nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := stringTypeAsByte(a, v)
		return ByteValue(b), nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := stringTypeAsDecimal(a, v)
		r := a.NewDecimal()
		*r = d
		return DecimalValue(r), nil

	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := stringTypeAsTime(a, v)
		return a.NewTimeValue(t), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := a.NewDict(utf8.RuneCountInString(o.Value))
		for i, r := range o.Value {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return a.NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := a.NewDict(utf8.RuneCountInString(o.Value))
		for i, r := range o.Value {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return a.NewDictValue(m, false), nil

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
		s, err := stringTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s), nil

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(len(o.Value) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(len(o.Value))), nil

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(strings.ToLower(o.Value)), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(strings.ToUpper(o.Value)), nil

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(stringTypeContains(a, v, args[0])), nil

	case "trim":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(args) == 0 {
			return a.NewStringValue(strings.Trim(o.Value, " \t\n")), nil
		}
		s, ok := args[0].AsString(a)
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName(a))
		}
		return a.NewStringValue(strings.Trim(o.Value, s)), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := []rune(o.Value)
		slices.Reverse(rs)
		return a.NewStringValue(string(rs)), nil

	case "filter":
		return stringFnFilter(a, vm, v, args)

	case "count":
		return stringFnCount(a, vm, v, args)

	case "all":
		return stringFnAll(a, vm, v, args)

	case "any":
		return stringFnAny(a, vm, v, args)

	case "for_each":
		return stringFnForEach(a, vm, v, args)

	case "find":
		return stringFnFind(a, vm, v, args)

	case "repeat":
		n, err := parseRepeatCount(a, name, args)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(strings.Repeat(o.Value, n)), nil

	case "join":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return joinSeqValueWithSepString(a, args[0], o.Value, name)

	case "split":
		return stringFnSplit(a, v, args)

	case "split_lines":
		return stringFnSplitLines(a, v, args)

	case "partition":
		return stringFnPartition(a, v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}

func stringTypeAccess(a *Arena, v Value, index Value, mode bc.Opcode) (Value, error) {
	if mode == bc.OpIndex {
		i, ok := index.AsInt(a)
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName(a))
		}
		o := (*String)(v.Ptr)
		i, ok = NormalizeIndex(i, int64(len(o.Value)))
		if !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), len(o.Value))
		}
		return ByteValue(o.Value[i]), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(a), index.String(a))
}

func stringTypeIterator(a *Arena, v Value) (Value, error) {
	o := (*String)(v.Ptr)
	return a.NewRunesIteratorValue([]rune(o.Value)), nil
}

func stringTypeAsInt(a *Arena, v Value) (int64, bool) {
	o := (*String)(v.Ptr)
	i, err := strconv.ParseInt(o.Value, 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func stringTypeAsByte(a *Arena, v Value) (byte, bool) {
	o := (*String)(v.Ptr)
	i, err := strconv.ParseInt(o.Value, 10, 64)
	if err == nil {
		if i < 0 || i > 255 {
			return byte(i), false
		}
		return byte(i), true
	}
	return 0, false
}

func stringTypeAsFloat(a *Arena, v Value) (float64, bool) {
	o := (*String)(v.Ptr)
	f, err := strconv.ParseFloat(o.Value, 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func stringTypeAsDecimal(a *Arena, v Value) (dec128.Dec128, bool) {
	o := (*String)(v.Ptr)
	d := dec128.FromString(o.Value)
	return d, !d.IsNaN()
}

func stringTypeAsTime(a *Arena, v Value) (time.Time, bool) {
	o := (*String)(v.Ptr)
	val, err := dateparse.ParseAny(o.Value)
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

func stringTypeAsArray(a *Arena, v Value) ([]Value, bool) {
	o := (*String)(v.Ptr)
	arr := a.NewArray(utf8.RuneCountInString(o.Value), true)
	for i, r := range o.Value {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

func stringTypeContains(a *Arena, v Value, e Value) bool {
	o := (*String)(v.Ptr)
	switch e.Type {
	case VT_RUNE:
		c := rune(e.Data)
		return strings.ContainsRune(o.Value, c)

	case VT_STRING:
		s := (*String)(e.Ptr)
		return strings.Contains(o.Value, s.Value)

	default:
		c, ok := e.AsRune(a)
		if !ok {
			return false
		}
		return strings.ContainsRune(o.Value, c)
	}
}

func stringTypeSlice(a *Arena, v Value, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	o := (*String)(v.Ptr)
	l := int64(len(o.Value))

	if s.Type != VT_UNDEFINED {
		si, ok = s.AsInt(a)
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName(a))
		}
	}

	if e.Type != VT_UNDEFINED {
		ei, ok = e.AsInt(a)
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName(a))
		}
	}

	si, ei = NormalizeSliceBounds(si, s.Type != VT_UNDEFINED, ei, e.Type != VT_UNDEFINED, l)
	return a.NewStringValue(o.Value[si:ei]), nil
}

func stringTypeSliceStep(a *Arena, v Value, s Value, e Value, stepVal Value) (Value, error) {
	var si, ei int64
	var ok bool

	o := (*String)(v.Ptr)
	l := int64(len(o.Value))

	step, ok := stepVal.AsInt(a)
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("slice step", "int", stepVal.TypeName(a))
	}
	if step == 0 {
		return Undefined, errs.NewSliceStepZeroError()
	}

	if s.Type != VT_UNDEFINED {
		si, ok = s.AsInt(a)
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName(a))
		}
	}
	if e.Type != VT_UNDEFINED {
		ei, ok = e.AsInt(a)
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName(a))
		}
	}

	start, end := NormalizeSliceBoundsStep(si, s.Type != VT_UNDEFINED, ei, e.Type != VT_UNDEFINED, step, l)
	bs := []byte(o.Value)
	result := a.NewBytes(0, false)
	if step > 0 {
		for i := start; i < end; i += step {
			result = append(result, bs[i])
		}
	} else {
		for i := start; i > end; i += step {
			result = append(result, bs[i])
		}
	}
	return a.NewStringValue(string(result)), nil
}

func stringFnFilter(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName(a))
	}

	var buf [2]Value
	o := (*String)(v.Ptr)
	filtered := a.NewRunes(utf8.RuneCountInString(o.Value), false)

	switch fn.Arity(a) {
	case 1:
		for _, v := range o.Value {
			buf[0] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				filtered = append(filtered, v)
			}
		}
		return a.NewStringValue(string(filtered)), nil

	case 2:
		for i, v := range o.Value {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				filtered = append(filtered, v)
			}
		}
		return a.NewStringValue(string(filtered)), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName(a))
	}
}

func stringFnCount(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName(a))
	}

	o := (*String)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		var count int64
		for _, v := range o.Value {
			buf[0] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				count++
			}
		}
		return IntValue(count), nil

	case 2:
		var count int64
		for i, v := range o.Value {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:2])
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

func stringFnForEach(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(a, args)
	if err != nil {
		return Undefined, err
	}

	o := (*String)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for _, v := range o.Value {
			buf[0] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue(a) {
				return Undefined, nil
			}
		}

	case 2:
		for i, v := range o.Value {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:2])
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

func stringFnFind(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName(a))
	}

	o := (*String)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for i, v := range o.Value {
			buf[0] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	case 2:
		for i, v := range o.Value {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName(a))
	}
}

func stringFnAll(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName(a))
	}

	o := (*String)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for _, v := range o.Value {
			buf[0] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue(a) {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	case 2:
		for i, v := range o.Value {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:2])
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

func stringFnAny(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName(a))
	}

	o := (*String)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for _, v := range o.Value {
			buf[0] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue(a) {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	case 2:
		for i, v := range o.Value {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(a, vm, buf[:2])
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

func stringFnSplit(a *Arena, v Value, args []Value) (Value, error) {
	const name = "split"
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	o := (*String)(v.Ptr)
	var pieces []string
	if len(args) == 0 {
		pieces = splitStringWhitespace(o.Value)
	} else {
		sep, err := coerceSepToString(a, name, args[0])
		if err != nil {
			return Undefined, err
		}
		if sep == "" {
			return Undefined, fmt.Errorf("split separator must not be empty")
		}
		limit := -1
		if len(args) == 2 {
			limit, err = parseSplitLimit(a, name, args, 1)
			if err != nil {
				return Undefined, err
			}
		}
		pieces = splitStringByLiteral(o.Value, sep, limit)
	}
	arr := a.NewArray(len(pieces), true)
	for i, p := range pieces {
		arr[i] = a.NewStringValue(p)
	}
	return a.NewArrayValue(arr, false), nil
}

func stringFnSplitLines(a *Arena, v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*String)(v.Ptr)
	pieces := splitLinesString(o.Value)
	arr := a.NewArray(len(pieces), true)
	for i, p := range pieces {
		arr[i] = a.NewStringValue(p)
	}
	return a.NewArrayValue(arr, false), nil
}

func stringFnPartition(a *Arena, v Value, args []Value) (Value, error) {
	const name = "partition"
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	sep, err := coerceSepToString(a, name, args[0])
	if err != nil {
		return Undefined, err
	}
	if sep == "" {
		return Undefined, fmt.Errorf("partition separator must not be empty")
	}
	o := (*String)(v.Ptr)
	arr := a.NewArray(3, true)
	idx := strings.Index(o.Value, sep)
	if idx < 0 {
		arr[0] = a.NewStringValue(o.Value)
		arr[1] = a.NewStringValue("")
		arr[2] = a.NewStringValue("")
	} else {
		arr[0] = a.NewStringValue(o.Value[:idx])
		arr[1] = a.NewStringValue(o.Value[idx : idx+len(sep)])
		arr[2] = a.NewStringValue(o.Value[idx+len(sep):])
	}
	return a.NewArrayValue(arr, false), nil
}
