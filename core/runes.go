package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"

	"github.com/araddon/dateparse"
	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/token"
)

const (
	runesTypeName          = "runes"
	immutableRunesTypeName = "immutable-runes"
)

type Runes = Seq[rune]

var TypeRunes = ValueTypeDescr{
	Pin:          func(a *Arena, v Value) { a.PinRunesValue(v) },
	Retain:       func(a *Arena, v Value) { a.RetainRunesValue(v) },
	Release:      func(a *Arena, v Value) { a.ReleaseRunesValue(v) },
	Name:         SeqNameHook(runesTypeName, immutableRunesTypeName),
	String:       func(a *Arena, v Value) string { return "u" + strconv.Quote(string((*Runes)(v.Ptr).Elements)) },
	Format:       runesTypeFormat,
	Interface:    func(a *Arena, v Value) any { return (*Runes)(v.Ptr).Elements },
	EncodeJSON:   runesTypeEncodeJSON,
	EncodeBinary: runesTypeEncodeBinary,
	DecodeBinary: runesTypeDecodeBinary,
	IsTrue:       SeqIsTrue[rune],
	IsIterable:   ConstHook(true),
	Iterator:     runesTypeIterator,
	Equal:        runesTypeEqual,
	Clone:        runesTypeClone,
	Len:          SeqLen[rune],
	BinaryOp:     runesTypeBinaryOp,
	MethodCall:   runesTypeMethodCall,
	Access:       SeqAccessHook(RuneValue),
	Assign:       SeqAssignHook(Value.AsRune, runeTypeName),
	Append:       runesTypeAppend,
	Contains:     runesTypeContains,
	Slice:        SeqSliceHook(ArenaNewRunesValue),
	SliceStep:    SeqSliceStepHook(ArenaNewRunes, ArenaNewRunesValue),
	AsBool:       runesTypeAsBool,
	AsInt:        runesTypeAsInt,
	AsByte:       runesTypeAsByte,
	AsFloat:      runesTypeAsFloat,
	AsDecimal:    runesTypeAsDecimal,
	AsTime:       runesTypeAsTime,
	AsString:     func(a *Arena, v Value) (string, bool) { return string((*Runes)(v.Ptr).Elements), true },
	AsRunes:      func(a *Arena, v Value) ([]rune, bool) { return (*Runes)(v.Ptr).Elements, true },
	AsBytes:      runesTypeAsBytes,
	AsArray:      runesTypeAsArray,
}

func runesTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	var b []byte
	b = EncodeString(b, string(o.Elements))
	return b, nil
}

func runesTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	s := string(o.Elements)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return nil, fmt.Errorf("runes: %w", err)
	}
	return buf.Bytes(), nil
}

func runesTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("runes: %w", err)
	}
	o := &Runes{Elements: []rune(s)}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func runesTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return "u" + strconv.Quote(string((*Runes)(v.Ptr).Elements)), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(a), sp, fspec.AlignLeft), nil
	}
	o := (*Runes)(v.Ptr)
	return format.FormatStringLike("runes", sp, string(o.Elements), false)
}

func runesTypeAppend(a *Arena, v Value, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	res := append([]rune{}, o.Elements...)
	for i, arg := range args {
		switch arg.Type {
		case VT_RUNES:
			t := (*Runes)(arg.Ptr)
			res = append(res, t.Elements...)
		default:
			c, ok := arg.AsRune(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError("append", fmt.Sprintf("%d", i+1), "rune or runes", arg.TypeName(a))
			}
			res = append(res, c)
		}
	}
	return a.NewRunesValue(res, false), nil
}

func runesTypeBinaryOp(a *Arena, v Value, rhs Value, op token.Token) (Value, error) {
	r, ok := rhs.AsRunes(a)
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
	}

	o := (*Runes)(v.Ptr)
	switch op {
	case token.Add:
		return a.NewRunesValue(append(o.Elements, r...), false), nil
	case token.Less:
		return BoolValue(string(o.Elements) < string(r)), nil
	case token.LessEq:
		return BoolValue(string(o.Elements) <= string(r)), nil
	case token.Greater:
		return BoolValue(string(o.Elements) > string(r)), nil
	case token.GreaterEq:
		return BoolValue(string(o.Elements) >= string(r)), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
}

func runesTypeEqual(a *Arena, v Value, r Value) bool {
	t, ok := r.AsRunes(a)
	if !ok {
		return false
	}
	o := (*Runes)(v.Ptr)
	return slices.Equal(o.Elements, t)
}

func runesTypeClone(a *Arena, v Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	rs := a.NewRunes(len(o.Elements), true)
	copy(rs, o.Elements)
	return a.NewRunesValue(rs, false), nil
}

func runesTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return runesTypeClone(a, v)

	case "runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(string(o.Elements)), nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsArray(a, v)
		return a.NewArrayValue(t, false), nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := runesTypeAsBool(a, v)
		return BoolValue(b), nil

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewBytesValue([]byte(string(o.Elements)), false), nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := runesTypeAsFloat(a, v)
		return FloatValue(f), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := runesTypeAsInt(a, v)
		return IntValue(i), nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := runesTypeAsByte(a, v)
		return ByteValue(b), nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := runesTypeAsDecimal(a, v)
		return a.NewDecimalValue(d), nil

	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsTime(a, v)
		return a.NewTimeValue(t), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := a.NewDict(len(o.Elements))
		for i, r := range o.Elements {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return a.NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := a.NewDict(len(o.Elements))
		for i, r := range o.Elements {
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
		s, err := runesTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s), nil

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

	case "first":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return RuneValue(o.Elements[0]), nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return RuneValue(o.Elements[len(o.Elements)-1]), nil

	case "min":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return RuneValue(slices.Min(o.Elements)), nil

	case "max":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return RuneValue(slices.Max(o.Elements)), nil

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := a.NewRunes(len(o.Elements), true)
		for i, r := range o.Elements {
			rs[i] = unicode.ToLower(r)
		}
		return a.NewRunesValue(rs, false), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := a.NewRunes(len(o.Elements), true)
		for i, r := range o.Elements {
			rs[i] = unicode.ToUpper(r)
		}
		return a.NewRunesValue(rs, false), nil

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(runesTypeContains(a, v, args[0])), nil

	case "trim":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(args) == 0 {
			return a.NewRunesValue([]rune(strings.Trim(string(o.Elements), " \t\n")), false), nil
		}
		s, ok := args[0].AsString(a)
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string or runes", args[0].TypeName(a))
		}
		return a.NewRunesValue([]rune(strings.Trim(string(o.Elements), s)), false), nil

	case "sort":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		sorted := a.NewRunes(len(o.Elements), true)
		copy(sorted, o.Elements)
		slices.Sort(sorted)
		return a.NewRunesValue(sorted, false), nil

	case "dedup":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := a.NewRunes(len(o.Elements), false)
		for i, r := range o.Elements {
			if i == 0 || r != o.Elements[i-1] {
				out = append(out, r)
			}
		}
		return a.NewRunesValue(out, false), nil

	case "unique":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := a.NewRunes(len(o.Elements), false)
		seen := make(map[rune]struct{}, len(o.Elements))
		for _, r := range o.Elements {
			if _, ok := seen[r]; !ok {
				seen[r] = struct{}{}
				out = append(out, r)
			}
		}
		return a.NewRunesValue(out, false), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		n := len(o.Elements)
		rev := a.NewRunes(n, true)
		for i, r := range o.Elements {
			rev[n-1-i] = r
		}
		return a.NewRunesValue(rev, false), nil

	case "filter":
		return SeqFilter(a, vm, v, args, RuneValue, ArenaNewRunes, ArenaNewRunesValue)

	case "count":
		return SeqCount(a, vm, v, args, RuneValue)

	case "all":
		return SeqAll(a, vm, v, args, RuneValue)

	case "any":
		return SeqAny(a, vm, v, args, RuneValue)

	case "for_each":
		return SeqForEach(a, vm, v, args, RuneValue)

	case "find":
		return SeqFind(a, vm, v, args, RuneValue)

	case "chunk":
		return SeqChunk(a, v, args, ArenaNewRunes, ArenaNewRunesValue)

	case "sum":
		return runesFnSum(v, args)

	case "avg":
		return runesFnAvg(v, args)

	case "map":
		return SeqMap(a, vm, v, args, RuneValue)

	case "reduce":
		return SeqReduce(a, vm, v, args, RuneValue)

	case "repeat":
		n, err := parseRepeatCount(a, name, args)
		if err != nil {
			return Undefined, err
		}
		src := o.Elements
		sl := len(src)
		out := a.NewRunes(n*sl, true)
		for i := range n {
			copy(out[i*sl:], src)
		}
		return a.NewRunesValue(out, false), nil

	case "join":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		elems, err := resolveJoinSeq(a, args[0], name)
		if err != nil {
			return Undefined, err
		}
		s, err := joinElementsToString(a, elems, string(o.Elements))
		if err != nil {
			return Undefined, err
		}
		return a.NewRunesValue([]rune(s), false), nil

	case "split":
		return runesFnSplit(a, v, args)

	case "split_lines":
		return runesFnSplitLines(a, v, args)

	case "partition":
		return runesFnPartition(a, v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}

func runesTypeIterator(a *Arena, v Value) (Value, error) {
	return a.NewRunesIteratorValue((*Runes)(v.Ptr).Elements), nil
}

func runesTypeAsByte(a *Arena, v Value) (byte, bool) {
	o := (*Runes)(v.Ptr)
	i, err := strconv.ParseInt(string(o.Elements), 10, 64)
	if err == nil {
		if i < 0 || i > 255 {
			return byte(i), false
		}
		return byte(i), true
	}
	return 0, false
}

func runesTypeAsInt(a *Arena, v Value) (int64, bool) {
	o := (*Runes)(v.Ptr)
	i, err := strconv.ParseInt(string(o.Elements), 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func runesTypeAsFloat(a *Arena, v Value) (float64, bool) {
	o := (*Runes)(v.Ptr)
	f, err := strconv.ParseFloat(string(o.Elements), 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func runesTypeAsDecimal(a *Arena, v Value) (dec128.Dec128, bool) {
	o := (*Runes)(v.Ptr)
	d := dec128.FromString(string(o.Elements))
	return d, !d.IsNaN()
}

func runesTypeAsBool(a *Arena, v Value) (bool, bool) {
	o := (*Runes)(v.Ptr)
	return conv.ParseBool(string(o.Elements))
}

func runesTypeAsBytes(a *Arena, v Value) ([]byte, bool) {
	o := (*Runes)(v.Ptr)
	return []byte(string(o.Elements)), true
}

func runesTypeAsTime(a *Arena, v Value) (time.Time, bool) {
	o := (*Runes)(v.Ptr)
	val, err := dateparse.ParseAny(string(o.Elements))
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

func runesTypeAsArray(a *Arena, v Value) ([]Value, bool) {
	o := (*Runes)(v.Ptr)
	arr := a.NewArray(len(o.Elements), true)
	for i, r := range o.Elements {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

func runesTypeContains(a *Arena, v Value, e Value) bool {
	o := (*Runes)(v.Ptr)
	switch e.Type {
	case VT_RUNE:
		c := rune(e.Data)
		return slices.Contains(o.Elements, c)

	case VT_STRING:
		return strings.Contains(string(o.Elements), *(*string)(e.Ptr))

	case VT_RUNES:
		runes := (*Runes)(e.Ptr)
		return strings.Contains(string(o.Elements), string(runes.Elements))

	default:
		c, ok := e.AsRune(a)
		if !ok {
			return false
		}
		return slices.Contains(o.Elements, c)
	}
}

func runesFnSum(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sum", "0", len(args))
	}
	o := (*Runes)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}
	var s int64
	for _, r := range o.Elements {
		s += int64(r)
	}
	return IntValue(s), nil
}

func runesFnAvg(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("avg", "0", len(args))
	}
	o := (*Runes)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}
	var s int64
	for _, r := range o.Elements {
		s += int64(r)
	}
	return IntValue(s / int64(len(o.Elements))), nil
}

func runesFnSplit(a *Arena, v Value, args []Value) (Value, error) {
	const name = "split"
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	o := (*Runes)(v.Ptr)
	src := string(o.Elements)
	var pieces []string
	if len(args) == 0 {
		pieces = splitStringWhitespace(src)
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
		pieces = splitStringByLiteral(src, sep, limit)
	}
	arr := a.NewArray(len(pieces), true)
	for i, p := range pieces {
		arr[i] = a.NewRunesValue([]rune(p), false)
	}
	return a.NewArrayValue(arr, false), nil
}

func runesFnSplitLines(a *Arena, v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Runes)(v.Ptr)
	pieces := splitLinesString(string(o.Elements))
	arr := a.NewArray(len(pieces), true)
	for i, p := range pieces {
		arr[i] = a.NewRunesValue([]rune(p), false)
	}
	return a.NewArrayValue(arr, false), nil
}

func runesFnPartition(a *Arena, v Value, args []Value) (Value, error) {
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
	o := (*Runes)(v.Ptr)
	src := string(o.Elements)
	arr := a.NewArray(3, true)
	idx := strings.Index(src, sep)
	if idx < 0 {
		arr[0] = a.NewRunesValue([]rune(src), false)
		arr[1] = a.NewRunesValue(nil, false)
		arr[2] = a.NewRunesValue(nil, false)
	} else {
		arr[0] = a.NewRunesValue([]rune(src[:idx]), false)
		arr[1] = a.NewRunesValue([]rune(src[idx:idx+len(sep)]), false)
		arr[2] = a.NewRunesValue([]rune(src[idx+len(sep):]), false)
	}
	return a.NewArrayValue(arr, false), nil
}
