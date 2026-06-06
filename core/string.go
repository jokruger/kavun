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

// TypeString is a string type descriptor.
var TypeString = ValueTypeDescr{
	Name:         ConstHook(stringTypeName),
	String:       func(a *Arena, v Value) string { return strconv.Quote(*(*string)(v.Ptr)) },
	Format:       stringTypeFormat,
	Interface:    func(a *Arena, v Value) any { return *(*string)(v.Ptr) },
	EncodeJSON:   stringTypeEncodeJSON,
	EncodeBinary: stringTypeEncodeBinary,
	DecodeBinary: stringTypeDecodeBinary,
	IsTrue:       func(a *Arena, v Value) bool { return len(*(*string)(v.Ptr)) > 0 },
	IsIterable:   ConstHook(true),
	Iterator:     stringTypeIterator,
	Equal:        stringTypeEqual,
	Len:          func(a *Arena, v Value) int64 { return int64(len(*(*string)(v.Ptr))) },
	BinaryOp:     stringTypeBinaryOp,
	MethodCall:   stringTypeMethodCall,
	Access:       stringTypeAccess,
	Contains:     stringTypeContains,
	Slice:        stringTypeSlice,
	SliceStep:    stringTypeSliceStep,
	AsBool:       func(a *Arena, v Value) (bool, bool) { return conv.ParseBool(*(*string)(v.Ptr)) },
	AsInt:        stringTypeAsInt,
	AsByte:       stringTypeAsByte,
	AsFloat:      stringTypeAsFloat,
	AsDecimal:    stringTypeAsDecimal,
	AsTime:       stringTypeAsTime,
	AsString:     func(a *Arena, v Value) (string, bool) { return *(*string)(v.Ptr), true },
	AsRunes:      func(a *Arena, v Value) ([]rune, bool) { return []rune(*(*string)(v.Ptr)), true },
	AsBytes:      func(a *Arena, v Value) ([]byte, bool) { return []byte(*(*string)(v.Ptr)), true },
	AsArray:      stringTypeAsArray,
}

func stringTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := (*string)(v.Ptr)
	var b []byte
	b = EncodeString(b, *o)
	return b, nil
}

func stringTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := (*string)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(*o); err != nil {
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
	*v = a.NewStringValue(s)
	return nil
}

func stringTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	o := (*string)(v.Ptr)
	if sp.Verb == 'v' {
		return strconv.Quote(*o), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(stringTypeName, sp, fspec.AlignLeft), nil
	}
	return format.FormatStringLike(stringTypeName, sp, *o, false)
}

func stringTypeBinaryOp(a *Arena, v Value, rhs Value, op token.Token) (Value, error) {
	r, ok := rhs.AsString(a)
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
	}

	o := (*string)(v.Ptr)
	switch op {
	case token.Add:
		return a.NewStringValue(*o + r), nil
	case token.Less:
		return BoolValue(*o < r), nil
	case token.LessEq:
		return BoolValue(*o <= r), nil
	case token.Greater:
		return BoolValue(*o > r), nil
	case token.GreaterEq:
		return BoolValue(*o >= r), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
}

func stringTypeEqual(a *Arena, v Value, r Value) bool {
	t, ok := r.AsString(a)
	if !ok {
		return false
	}
	o := (*string)(v.Ptr)
	return *o == t
}

func stringTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*string)(v.Ptr)

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
		return a.NewBytesValue([]byte(*o), false), nil

	case "runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := a.NewRunes(utf8.RuneCountInString(*o), true)
		for i, r := range *o {
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
		b, _ := conv.ParseBool(*(*string)(v.Ptr))
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
		return a.NewDecimalValue(d), nil

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
		m := a.NewDict(utf8.RuneCountInString(*o))
		for i, r := range *o {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return a.NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := a.NewDict(utf8.RuneCountInString(*o))
		for i, r := range *o {
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
		return BoolValue(len(*o) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(len(*o))), nil

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(strings.ToLower(*o)), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(strings.ToUpper(*o)), nil

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
			return a.NewStringValue(strings.Trim(*o, " \t\n")), nil
		}
		s, ok := args[0].AsString(a)
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName(a))
		}
		return a.NewStringValue(strings.Trim(*o, s)), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := []rune(*o)
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
		return a.NewStringValue(strings.Repeat(*o, n)), nil

	case "join":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return joinSeqValueWithSepString(a, args[0], *o, name)

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
		s := *(*string)(v.Ptr)
		i, ok = NormalizeIndex(i, int64(len(s)))
		if !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), len(s))
		}
		return ByteValue(s[i]), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(a), index.String(a))
}

func stringTypeIterator(a *Arena, v Value) (Value, error) {
	o := (*string)(v.Ptr)
	return a.NewRunesIteratorValue([]rune(*o)), nil
}

func stringTypeAsInt(a *Arena, v Value) (int64, bool) {
	o := (*string)(v.Ptr)
	i, err := strconv.ParseInt(*o, 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func stringTypeAsByte(a *Arena, v Value) (byte, bool) {
	o := (*string)(v.Ptr)
	i, err := strconv.ParseInt(*o, 10, 64)
	if err == nil {
		if i < 0 || i > 255 {
			return byte(i), false
		}
		return byte(i), true
	}
	return 0, false
}

func stringTypeAsFloat(a *Arena, v Value) (float64, bool) {
	o := (*string)(v.Ptr)
	f, err := strconv.ParseFloat(*o, 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func stringTypeAsDecimal(a *Arena, v Value) (dec128.Dec128, bool) {
	o := (*string)(v.Ptr)
	d := dec128.FromString(*o)
	return d, !d.IsNaN()
}

func stringTypeAsTime(a *Arena, v Value) (time.Time, bool) {
	o := (*string)(v.Ptr)
	val, err := dateparse.ParseAny(*o)
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

func stringTypeAsArray(a *Arena, v Value) ([]Value, bool) {
	o := (*string)(v.Ptr)
	arr := a.NewArray(utf8.RuneCountInString(*o), true)
	for i, r := range *o {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

func stringTypeContains(a *Arena, v Value, e Value) bool {
	o := (*string)(v.Ptr)
	switch e.Type {
	case VT_RUNE:
		c := rune(e.Data)
		return strings.ContainsRune(*o, c)

	case VT_STRING:
		s := (*string)(e.Ptr)
		return strings.Contains(*o, *s)

	default:
		c, ok := e.AsRune(a)
		if !ok {
			return false
		}
		return strings.ContainsRune(*o, c)
	}
}

func stringTypeSlice(a *Arena, v Value, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	str := *(*string)(v.Ptr)
	l := int64(len(str))

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
	return a.NewStringValue(str[si:ei]), nil
}

func stringTypeSliceStep(a *Arena, v Value, s Value, e Value, stepVal Value) (Value, error) {
	var si, ei int64
	var ok bool

	str := *(*string)(v.Ptr)
	l := int64(len(str))

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
	bs := []byte(str)
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
	o := (*string)(v.Ptr)
	filtered := a.NewRunes(utf8.RuneCountInString(*o), false)

	switch fn.Arity(a) {
	case 1:
		for _, v := range *o {
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
		for i, v := range *o {
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

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		var count int64
		for _, v := range *o {
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
		for i, v := range *o {
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

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for _, v := range *o {
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
		for i, v := range *o {
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

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for i, v := range *o {
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
		for i, v := range *o {
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

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for _, v := range *o {
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
		for i, v := range *o {
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

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity(a) {
	case 1:
		for _, v := range *o {
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
		for i, v := range *o {
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
	o := (*string)(v.Ptr)
	var pieces []string
	if len(args) == 0 {
		pieces = splitStringWhitespace(*o)
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
		pieces = splitStringByLiteral(*o, sep, limit)
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
	o := (*string)(v.Ptr)
	pieces := splitLinesString(*o)
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
	s := *(*string)(v.Ptr)
	arr := a.NewArray(3, true)
	idx := strings.Index(s, sep)
	if idx < 0 {
		arr[0] = a.NewStringValue(s)
		arr[1] = a.NewStringValue("")
		arr[2] = a.NewStringValue("")
	} else {
		arr[0] = a.NewStringValue(s[:idx])
		arr[1] = a.NewStringValue(s[idx : idx+len(sep)])
		arr[2] = a.NewStringValue(s[idx+len(sep):])
	}
	return a.NewArrayValue(arr, false), nil
}
