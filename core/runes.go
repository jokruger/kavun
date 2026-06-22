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
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
)

const (
	runesTypeName          = "runes"
	immutableRunesTypeName = "immutable-runes"
)

type Runes = Seq[rune]

func NewStaticRunesValue(r *Runes) Value {
	return Value{Type: value.Runes, Immutable: true, Ptr: unsafe.Pointer(r)}
}

func NewRunesValue(r []rune, immutable bool) Value {
	o := &Runes{}
	o.Set(r)
	return Value{Type: value.Runes, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeRunes = ValueTypeDescr{
	Name:         SeqNameHook(runesTypeName, immutableRunesTypeName),
	String:       func(v Value) string { return "u" + strconv.Quote(string((*Runes)(v.Ptr).Elements)) },
	Format:       runesTypeFormat,
	Interface:    func(v Value) any { return (*Runes)(v.Ptr).Elements },
	EncodeJSON:   runesTypeEncodeJSON,
	EncodeBinary: runesTypeEncodeBinary,
	DecodeBinary: runesTypeDecodeBinary,
	IsTrue:       func(v Value) bool { return len((*Runes)(v.Ptr).Elements) > 0 },
	IsIterable:   ConstHook(true),
	Iterator:     runesTypeIterator,
	Equal:        runesTypeEqual,
	Clone:        runesTypeClone,
	Len:          func(v Value) int64 { return int64(len((*Runes)(v.Ptr).Elements)) },
	BinaryOp:     runesTypeBinaryOp,
	MethodCall:   runesTypeMethodCall,
	Access:       SeqAccessHook(RuneValue, runesTypeResolve),
	Assign:       SeqAssignHook(runesTypeResolve, Value.AsRune, runeTypeName),
	Append:       runesTypeAppend,
	Contains:     runesTypeContains,
	Slice:        SeqSliceHook(NewRunesValue, runesTypeResolve),
	SliceStep:    SeqSliceStepHook(NewRunesValue, runesTypeResolve),
	AsBool:       runesTypeAsBool,
	AsInt:        runesTypeAsInt,
	AsByte:       runesTypeAsByte,
	AsFloat:      runesTypeAsFloat,
	AsDecimal:    runesTypeAsDecimal,
	AsTime:       runesTypeAsTime,
	AsString:     func(v Value) (string, bool) { return string((*Runes)(v.Ptr).Elements), true },
	AsRunes:      func(v Value) ([]rune, bool) { return (*Runes)(v.Ptr).Elements, true },
	AsBytes:      runesTypeAsBytes,
	AsArray:      runesTypeAsArray,
}

func runesTypeResolve(v Value) *Runes {
	return (*Runes)(v.Ptr)
}

func runesTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	var b []byte
	b = EncodeString(b, string(o.Elements))
	return b, nil
}

func runesTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	s := string(o.Elements)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return nil, fmt.Errorf("runes: %w", err)
	}
	return buf.Bytes(), nil
}

func runesTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("runes: %w", err)
	}
	*v = NewRunesValue([]rune(s), v.Immutable)
	return nil
}

func runesTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return "u" + strconv.Quote(string((*Runes)(v.Ptr).Elements)), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	o := (*Runes)(v.Ptr)
	return format.FormatStringLike("runes", sp, string(o.Elements), false)
}

func runesTypeAppend(v Value, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	res := append([]rune{}, o.Elements...)
	for i, arg := range args {
		switch arg.Type {
		case value.Runes:
			res = append(res, (*Runes)(arg.Ptr).Elements...)
		default:
			c, ok := arg.AsRune()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError("append", fmt.Sprintf("%d", i+1), "rune or runes", arg.TypeName())
			}
			res = append(res, c)
		}
	}
	return NewRunesValue(res, false), nil
}

func runesTypeBinaryOp(v Value, rhs Value, op token.Token) (Value, error) {
	r, ok := rhs.AsRunes()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	o := (*Runes)(v.Ptr)
	switch op {
	case token.Add:
		return NewRunesValue(append(o.Elements, r...), false), nil
	case token.Less:
		return BoolValue(string(o.Elements) < string(r)), nil
	case token.LessEq:
		return BoolValue(string(o.Elements) <= string(r)), nil
	case token.Greater:
		return BoolValue(string(o.Elements) > string(r)), nil
	case token.GreaterEq:
		return BoolValue(string(o.Elements) >= string(r)), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}

func runesTypeEqual(v Value, r Value) bool {
	t, ok := r.AsRunes()
	if !ok {
		return false
	}
	o := (*Runes)(v.Ptr)
	return slices.Equal(o.Elements, t)
}

func runesTypeClone(v Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	rs := make([]rune, len(o.Elements))
	copy(rs, o.Elements)
	return NewRunesValue(rs, false), nil
}

func runesTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return runesTypeClone(v)

	case "runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(string(o.Elements)), nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsArray(v)
		return NewArrayValue(t, false), nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := runesTypeAsBool(v)
		return BoolValue(b), nil

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewBytesValue([]byte(string(o.Elements)), false), nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := runesTypeAsFloat(v)
		return FloatValue(f), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := runesTypeAsInt(v)
		return IntValue(i), nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := runesTypeAsByte(v)
		return ByteValue(b), nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := runesTypeAsDecimal(v)
		return NewDecimalValue(d), nil

	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsTime(v)
		return NewTimeValue(t), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := make(map[string]Value, len(o.Elements))
		for i, r := range o.Elements {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := make(map[string]Value, len(o.Elements))
		for i, r := range o.Elements {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return NewDictValue(m, false), nil

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
		s, err := runesTypeFormat(v, sp)
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
		rs := make([]rune, len(o.Elements))
		for i, r := range o.Elements {
			rs[i] = unicode.ToLower(r)
		}
		return NewRunesValue(rs, false), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := make([]rune, len(o.Elements))
		for i, r := range o.Elements {
			rs[i] = unicode.ToUpper(r)
		}
		return NewRunesValue(rs, false), nil

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(runesTypeContains(v, args[0])), nil

	case "trim":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(args) == 0 {
			return NewRunesValue([]rune(strings.Trim(string(o.Elements), " \t\n")), false), nil
		}
		s, ok := args[0].AsString()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string or runes", args[0].TypeName())
		}
		return NewRunesValue([]rune(strings.Trim(string(o.Elements), s)), false), nil

	case "sort":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		sorted := make([]rune, len(o.Elements))
		copy(sorted, o.Elements)
		slices.Sort(sorted)
		return NewRunesValue(sorted, false), nil

	case "dedup":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := make([]rune, 0, len(o.Elements))
		for i, r := range o.Elements {
			if i == 0 || r != o.Elements[i-1] {
				out = append(out, r)
			}
		}
		return NewRunesValue(out, false), nil

	case "unique":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := make([]rune, 0, len(o.Elements))
		seen := make(map[rune]struct{}, len(o.Elements))
		for _, r := range o.Elements {
			if _, ok := seen[r]; !ok {
				seen[r] = struct{}{}
				out = append(out, r)
			}
		}
		return NewRunesValue(out, false), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		n := len(o.Elements)
		rev := make([]rune, n)
		for i, r := range o.Elements {
			rev[n-1-i] = r
		}
		return NewRunesValue(rev, false), nil

	case "filter":
		return SeqFilter(vm, v, args, RuneValue, NewRunesValue, runesTypeResolve)

	case "count":
		return SeqCount(vm, v, args, RuneValue, runesTypeResolve)

	case "all":
		return SeqAll(vm, v, args, RuneValue, runesTypeResolve)

	case "any":
		return SeqAny(vm, v, args, RuneValue, runesTypeResolve)

	case "for_each":
		return SeqForEach(vm, v, args, RuneValue, runesTypeResolve)

	case "find":
		return SeqFind(vm, v, args, RuneValue, runesTypeResolve)

	case "chunk":
		return SeqChunk(v, args, NewRunesValue, runesTypeResolve)

	case "sum":
		return runesFnSum(v, args)

	case "avg":
		return runesFnAvg(v, args)

	case "map":
		return SeqMap(vm, v, args, RuneValue, runesTypeResolve)

	case "reduce":
		return SeqReduce(vm, v, args, RuneValue, runesTypeResolve)

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		src := o.Elements
		sl := len(src)
		out := make([]rune, n*sl)
		for i := range n {
			copy(out[i*sl:], src)
		}
		return NewRunesValue(out, false), nil

	case "join":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		elems, err := resolveJoinSeq(args[0], name)
		if err != nil {
			return Undefined, err
		}
		s, err := joinElementsToString(elems, string(o.Elements))
		if err != nil {
			return Undefined, err
		}
		return NewRunesValue([]rune(s), false), nil

	case "split":
		return runesFnSplit(v, args)

	case "split_lines":
		return runesFnSplitLines(v, args)

	case "partition":
		return runesFnPartition(v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func runesTypeIterator(v Value) (Value, error) {
	return NewRunesIteratorValue((*Runes)(v.Ptr).Elements), nil
}

func runesTypeAsByte(v Value) (byte, bool) {
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

func runesTypeAsInt(v Value) (int64, bool) {
	o := (*Runes)(v.Ptr)
	i, err := strconv.ParseInt(string(o.Elements), 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func runesTypeAsFloat(v Value) (float64, bool) {
	o := (*Runes)(v.Ptr)
	f, err := strconv.ParseFloat(string(o.Elements), 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func runesTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	o := (*Runes)(v.Ptr)
	d := dec128.FromString(string(o.Elements))
	return d, !d.IsNaN()
}

func runesTypeAsBool(v Value) (bool, bool) {
	o := (*Runes)(v.Ptr)
	return conv.ParseBool(string(o.Elements))
}

func runesTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Runes)(v.Ptr)
	return []byte(string(o.Elements)), true
}

func runesTypeAsTime(v Value) (time.Time, bool) {
	o := (*Runes)(v.Ptr)
	val, err := dateparse.ParseAny(string(o.Elements))
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

func runesTypeAsArray(v Value) ([]Value, bool) {
	o := (*Runes)(v.Ptr)
	arr := make([]Value, len(o.Elements))
	for i, r := range o.Elements {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

func runesTypeContains(v Value, e Value) bool {
	o := (*Runes)(v.Ptr)
	switch e.Type {
	case value.Rune:
		c := rune(e.Data)
		return slices.Contains(o.Elements, c)

	case value.String:
		return strings.Contains(string(o.Elements), *(*string)(e.Ptr))

	case value.Runes:
		return strings.Contains(string(o.Elements), string((*Runes)(e.Ptr).Elements))

	default:
		c, ok := e.AsRune()
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

func runesFnSplit(v Value, args []Value) (Value, error) {
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
		sep, err := coerceSepToString(name, args[0])
		if err != nil {
			return Undefined, err
		}
		if sep == "" {
			return Undefined, fmt.Errorf("split separator must not be empty")
		}
		limit := -1
		if len(args) == 2 {
			limit, err = parseSplitLimit(name, args, 1)
			if err != nil {
				return Undefined, err
			}
		}
		pieces = splitStringByLiteral(src, sep, limit)
	}
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewRunesValue([]rune(p), false)
	}
	return NewArrayValue(arr, false), nil
}

func runesFnSplitLines(v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Runes)(v.Ptr)
	pieces := splitLinesString(string(o.Elements))
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewRunesValue([]rune(p), false)
	}
	return NewArrayValue(arr, false), nil
}

func runesFnPartition(v Value, args []Value) (Value, error) {
	const name = "partition"
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	sep, err := coerceSepToString(name, args[0])
	if err != nil {
		return Undefined, err
	}
	if sep == "" {
		return Undefined, fmt.Errorf("partition separator must not be empty")
	}
	o := (*Runes)(v.Ptr)
	src := string(o.Elements)
	arr := make([]Value, 3)
	idx := strings.Index(src, sep)
	if idx < 0 {
		arr[0] = NewRunesValue([]rune(src), false)
		arr[1] = NewRunesValue(nil, false)
		arr[2] = NewRunesValue(nil, false)
	} else {
		arr[0] = NewRunesValue([]rune(src[:idx]), false)
		arr[1] = NewRunesValue([]rune(src[idx:idx+len(sep)]), false)
		arr[2] = NewRunesValue([]rune(src[idx+len(sep):]), false)
	}
	return NewArrayValue(arr, false), nil
}
