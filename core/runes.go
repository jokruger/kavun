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
	"github.com/jokruger/kavun/token"
)

type Runes struct {
	Elements []rune
}

func (o *Runes) Set(r []rune) {
	o.Elements = r
}

// RunesValue creates new boxed runes value.
func RunesValue(v *Runes, immutable bool) Value {
	return Value{
		Type:  VT_RUNES,
		Const: immutable,
		Ptr:   unsafe.Pointer(v),
	}
}

// NewRunesValue creates new (heap-allocated) runes value.
func NewRunesValue(v []rune, immutable bool) Value {
	o := &Runes{}
	o.Set(v)
	return RunesValue(o, immutable)
}

/* Runes type methods */

func runesTypeName(v Value) string {
	if v.Const {
		return "immutable-runes"
	}
	return "runes"
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
	o := &Runes{Elements: []rune(s)}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func runesTypeString(v Value) string {
	o := (*Runes)(v.Ptr)
	return "u" + strconv.Quote(string(o.Elements))
}

func runesTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return runesTypeString(v), nil
	}
	o := (*Runes)(v.Ptr)
	return formatStringLike(v, sp, string(o.Elements), false)
}

func runesTypeInterface(v Value) any {
	o := (*Runes)(v.Ptr)
	return o.Elements
}

func runesTypeAssign(v Value, index Value, r Value) error {
	if v.Const {
		return errs.NewNotAssignableError("immutable-runes")
	}

	o := (*Runes)(v.Ptr)
	i, ok := index.AsInt()
	if !ok {
		return errs.NewInvalidIndexTypeError("index assign", "int", index.TypeName())
	}
	i, ok = normalizeSequenceIndex(i, int64(len(o.Elements)))
	if !ok {
		return errs.NewIndexOutOfBoundsError("index assign", int(i), len(o.Elements))
	}

	c, ok := r.AsRune()
	if !ok {
		return errs.NewInvalidIndexTypeError("index assign value", "rune", r.TypeName())
	}
	o.Elements[i] = c

	return nil
}

func runesTypeAppend(v Value, a *Arena, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	res := append([]rune{}, o.Elements...)
	for i, arg := range args {
		switch arg.Type {
		case VT_RUNES:
			t := (*Runes)(arg.Ptr)
			res = append(res, t.Elements...)
		default:
			c, ok := arg.AsRune()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError("append", fmt.Sprintf("%d", i+1), "rune or runes", arg.TypeName())
			}
			res = append(res, c)
		}
	}
	return a.NewRunesValue(res, false), nil
}

func runesTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsRunes()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
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

func runesTypeCopy(v Value, a *Arena) (Value, error) {
	o := (*Runes)(v.Ptr)
	rs := a.NewRunes(len(o.Elements), true)
	copy(rs, o.Elements)
	return a.NewRunesValue(rs, false), nil
}

func runesTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	alloc := vm.Allocator()

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return runesTypeCopy(v, alloc)

	case "runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewStringValue(string(o.Elements)), nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsArray(v, alloc)
		return alloc.NewArrayValue(t, false), nil

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
		return alloc.NewBytesValue([]byte(string(o.Elements)), false), nil

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
		r := alloc.NewDecimal()
		*r = d
		return DecimalValue(r), nil

	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsTime(v)
		d := alloc.NewTime()
		*d = t
		return TimeValue(d), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := alloc.NewDict(len(o.Elements))
		for i, r := range o.Elements {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return alloc.NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := alloc.NewDict(len(o.Elements))
		for i, r := range o.Elements {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return alloc.NewDictValue(m, false), nil

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
		rs := alloc.NewRunes(len(o.Elements), true)
		for i, r := range o.Elements {
			rs[i] = unicode.ToLower(r)
		}
		return alloc.NewRunesValue(rs, false), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := alloc.NewRunes(len(o.Elements), true)
		for i, r := range o.Elements {
			rs[i] = unicode.ToUpper(r)
		}
		return alloc.NewRunesValue(rs, false), nil

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
			return alloc.NewRunesValue([]rune(strings.Trim(string(o.Elements), " \t\n")), false), nil
		}
		s, ok := args[0].AsString()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string or runes", args[0].TypeName())
		}
		return alloc.NewRunesValue([]rune(strings.Trim(string(o.Elements), s)), false), nil

	case "sort":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		sorted := alloc.NewRunes(len(o.Elements), true)
		copy(sorted, o.Elements)
		slices.Sort(sorted)
		return alloc.NewRunesValue(sorted, false), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		n := len(o.Elements)
		rev := alloc.NewRunes(n, true)
		for i, r := range o.Elements {
			rev[n-1-i] = r
		}
		return alloc.NewRunesValue(rev, false), nil

	case "filter":
		return runesFnFilter(v, vm, args)

	case "count":
		return runesFnCount(v, vm, args)

	case "all":
		return runesFnAll(v, vm, args)

	case "any":
		return runesFnAny(v, vm, args)

	case "for_each":
		return runesFnForEach(v, vm, args)

	case "find":
		return runesFnFind(v, vm, args)

	case "chunk":
		return runesFnChunk(v, vm, args)

	case "sum":
		return runesFnSum(v, vm, args)

	case "avg":
		return runesFnAvg(v, vm, args)

	case "map":
		return runesFnMap(v, vm, args)

	case "reduce":
		return runesFnReduce(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func runesTypeAccess(v Value, a *Arena, index Value, mode Opcode) (Value, error) {
	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}
		o := (*Runes)(v.Ptr)
		rs := o.Elements
		i, ok = normalizeSequenceIndex(i, int64(len(rs)))
		if !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), len(rs))
		}
		return RuneValue(rs[i]), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func runesTypeIterator(v Value, a *Arena) (Value, error) {
	o := (*Runes)(v.Ptr)
	return a.NewRunesIteratorValue(o.Elements), nil
}

func runesTypeIsTrue(v Value) bool {
	o := (*Runes)(v.Ptr)
	return len(o.Elements) > 0
}

func runesTypeAsString(v Value) (string, bool) {
	o := (*Runes)(v.Ptr)
	return string(o.Elements), true
}

func runesTypeAsRunes(v Value) ([]rune, bool) {
	o := (*Runes)(v.Ptr)
	return o.Elements, true
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

func runesTypeAsArray(v Value, a *Arena) ([]Value, bool) {
	o := (*Runes)(v.Ptr)
	arr := a.NewArray(len(o.Elements), true)
	for i, r := range o.Elements {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

func runesTypeContains(v Value, e Value) bool {
	o := (*Runes)(v.Ptr)
	switch e.Type {
	case VT_RUNE:
		c := rune(e.Data)
		return slices.Contains(o.Elements, c)

	case VT_STRING:
		s := (*String)(e.Ptr)
		return strings.Contains(string(o.Elements), s.Value)

	case VT_RUNES:
		runes := (*Runes)(e.Ptr)
		return strings.Contains(string(o.Elements), string(runes.Elements))

	default:
		c, ok := e.AsRune()
		if !ok {
			return false
		}
		return slices.Contains(o.Elements, c)
	}
}

func runesTypeLen(v Value) int64 {
	o := (*Runes)(v.Ptr)
	return int64(len(o.Elements))
}

func runesTypeSlice(v Value, a *Arena, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	o := (*Runes)(v.Ptr)
	rs := o.Elements
	l := int64(len(rs))

	if s.Type != VT_UNDEFINED {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
		}
	}

	if e.Type != VT_UNDEFINED {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	si, ei = normalizeSliceBounds(si, s.Type != VT_UNDEFINED, ei, e.Type != VT_UNDEFINED, l)
	return a.NewRunesValue(rs[si:ei], v.Const), nil
}

func runesTypeSliceStep(v Value, a *Arena, s Value, e Value, stepVal Value) (Value, error) {
	var si, ei int64
	var ok bool

	o := (*Runes)(v.Ptr)
	rs := o.Elements
	l := int64(len(rs))

	step, ok := stepVal.AsInt()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("slice step", "int", stepVal.TypeName())
	}
	if step == 0 {
		return Undefined, errs.NewSliceStepZeroError()
	}

	if s.Type != VT_UNDEFINED {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
		}
	}
	if e.Type != VT_UNDEFINED {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	start, end := normalizeSliceBoundsStep(si, s.Type != VT_UNDEFINED, ei, e.Type != VT_UNDEFINED, step, l)
	result := a.NewRunes(0, false)
	if step > 0 {
		for i := start; i < end; i += step {
			result = append(result, rs[i])
		}
	} else {
		for i := start; i > end; i += step {
			result = append(result, rs[i])
		}
	}
	return a.NewRunesValue(result, false), nil
}

func runesFnChunk(v Value, vm VM, args []Value) (Value, error) {
	size, copyChunks, err := chunkArgs("chunk", args)
	if err != nil {
		return Undefined, err
	}

	o := (*Runes)(v.Ptr)
	length := len(o.Elements)
	alloc := vm.Allocator()
	chunks := alloc.NewArray(chunkCount(length, size), true)

	if length == 0 {
		return alloc.NewArrayValue(chunks, false), nil
	}

	chunkSize := length
	if size < int64(length) {
		chunkSize = int(size)
	}

	for i, start := 0, 0; start < length; i, start = i+1, start+chunkSize {
		end := min(start+chunkSize, length)
		chunk := o.Elements[start:end]
		chunkConst := v.Const
		if copyChunks {
			chunk = alloc.NewRunes(end-start, true)
			copy(chunk, o.Elements[start:end])
			chunkConst = false
		}
		chunks[i] = alloc.NewRunesValue(chunk, chunkConst)
	}

	return alloc.NewArrayValue(chunks, false), nil
}

func runesFnForEach(v Value, vm VM, args []Value) (Value, error) {
	fn, err := forEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	o := (*Runes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return Undefined, nil
			}
		}

	case 2:
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
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

func runesFnFind(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for i, v := range o.Elements {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	case 2:
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName())
	}
}

func runesFnFilter(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		filtered := alloc.NewRunes(len(o.Elements), false)
		for _, v := range o.Elements {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewRunesValue(filtered, false), nil

	case 2:
		filtered := alloc.NewRunes(len(o.Elements), false)
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewRunesValue(filtered, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

func runesFnCount(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		var count int64
		for _, v := range o.Elements {
			buf[0] = RuneValue(v)
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
		var count int64
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
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

func runesFnAll(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = RuneValue(v)
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
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
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

func runesFnAny(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = RuneValue(v)
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
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
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

func runesFnSum(v Value, vm VM, args []Value) (Value, error) {
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

func runesFnAvg(v Value, vm VM, args []Value) (Value, error) {
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

func runesFnMap(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("map", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("map", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	o := (*Runes)(v.Ptr)
	alloc := vm.Allocator()
	mapped := alloc.NewArray(len(o.Elements), true)

	switch fn.Arity() {
	case 1:
		for i, r := range o.Elements {
			buf[0] = RuneValue(r)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			mapped[i] = res
		}
		return alloc.NewArrayValue(mapped, false), nil

	case 2:
		for i, r := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(r)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			mapped[i] = res
		}
		return alloc.NewArrayValue(mapped, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("map", "first", "f/1 or f/2", fn.TypeName())
	}
}

func runesFnReduce(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError("reduce", "2", len(args))
	}

	acc := args[0]
	fn := args[1]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("reduce", "second", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	var buf [3]Value
	switch fn.Arity() {
	case 2:
		for _, r := range o.Elements {
			buf[0] = acc
			buf[1] = RuneValue(r)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	case 3:
		for i, r := range o.Elements {
			buf[0] = acc
			buf[1] = IntValue(int64(i))
			buf[2] = RuneValue(r)
			res, err := fn.Call(vm, buf[:3])
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("reduce", "second", "f/2 or f/3", fn.TypeName())
	}
}
