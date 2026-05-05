package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unsafe"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

type Array struct {
	Elements []Value
}

func (o *Array) Set(elements []Value) {
	o.Elements = elements
}

// ArrayValue creates boxed array value.
func ArrayValue(v *Array, immutable bool) Value {
	return Value{
		Type:  VT_ARRAY,
		Const: immutable,
		Ptr:   unsafe.Pointer(v),
	}
}

// NewArrayValue creates a new (heap-allocated) array value.
func NewArrayValue(vals []Value, immutable bool) Value {
	t := &Array{}
	t.Set(vals)
	return ArrayValue(t, immutable)
}

/* Array type methods */

func arrayTypeName(v Value) string {
	if v.Const {
		return "immutable-array"
	}
	return "array"
}

func arrayTypeAssign(v Value, index Value, r Value) (err error) {
	if v.Const {
		return errs.NewNotAssignableError("immutable-array")
	}

	o := (*Array)(v.Ptr)
	i, ok := index.AsInt()
	if !ok {
		return errs.NewInvalidIndexTypeError("index assign", "int", index.TypeName())
	}
	i, ok = normalizeSequenceIndex(i, int64(len(o.Elements)))
	if !ok {
		return errs.NewIndexOutOfBoundsError("index assign", int(i), len(o.Elements))
	}

	o.Elements[i] = r

	return nil
}

func arrayTypeString(v Value) string {
	o := (*Array)(v.Ptr)
	parts := make([]string, len(o.Elements))
	for i, e := range o.Elements {
		parts[i] = e.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func arrayTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return arrayTypeString(v), nil
	}
	if err := validateContainerSpec(v, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(arrayTypeString(v), sp, fspec.AlignLeft), nil
}

func arrayTypeInterface(v Value) any {
	o := (*Array)(v.Ptr)
	res := make([]any, len(o.Elements))
	for i, val := range o.Elements {
		res[i] = val.Interface()
	}
	return res
}

func arrayTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Array)(v.Ptr)
	var b []byte
	b = append(b, '[')
	len1 := len(o.Elements) - 1
	for idx, elem := range o.Elements {
		eb, err := elem.EncodeJSON()
		if err != nil {
			return nil, fmt.Errorf("array element at index %d: %w", idx, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
	}
	b = append(b, ']')
	return b, nil
}

func arrayTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Array)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Elements); err != nil {
		return nil, fmt.Errorf("array (elements): %w", err)
	}
	return buf.Bytes(), nil
}

func arrayTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var arr []Value
	if err := dec.Decode(&arr); err != nil {
		return fmt.Errorf("array (elements): %w", err)
	}
	if arr == nil {
		arr = []Value{}
	}
	o := &Array{Elements: arr}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func arrayTypeIsTrue(v Value) bool {
	o := (*Array)(v.Ptr)
	return len(o.Elements) > 0
}

func arrayTypeIterator(v Value, a *Arena) (Value, error) {
	o := (*Array)(v.Ptr)
	return a.NewArrayIteratorValue(o.Elements), nil
}

func arrayTypeEqual(v Value, r Value) bool {
	if r.Type != VT_ARRAY {
		return false
	}

	la := (*Array)(v.Ptr).Elements
	ra := (*Array)(r.Ptr).Elements

	if len(la) != len(ra) {
		return false
	}

	for i, e := range la {
		if !e.Equal(ra[i]) {
			return false
		}
	}

	return true
}

func arrayTypeCopy(v Value, a *Arena) (Value, error) {
	// Deep copy the array (and make it mutable) and its elements
	o := (*Array)(v.Ptr)
	c := a.NewArray(len(o.Elements), true)
	for i, e := range o.Elements {
		t, err := e.Copy(a)
		if err != nil {
			return Undefined, err
		}
		c[i] = t
	}
	return a.NewArrayValue(c, false), nil
}

func arrayTypeLen(v Value) int64 {
	o := (*Array)(v.Ptr)
	return int64(len(o.Elements))
}

func arrayTypeBinaryOp(v Value, a *Arena, op token.Token, r Value) (Value, error) {
	if r.Type != VT_ARRAY {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
	}

	la := (*Array)(v.Ptr)
	ra := (*Array)(r.Ptr)
	switch op {
	case token.Add:
		return a.NewArrayValue(append(la.Elements, ra.Elements...), false), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
}

func arrayTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	alloc := vm.Allocator()

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return arrayTypeCopy(v, alloc)

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		bs := alloc.NewBytes(len(o.Elements), true)
		for i, e := range o.Elements {
			bs[i], _ = e.AsByte()
		}
		return alloc.NewBytesValue(bs, false), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		r := alloc.NewRunes(len(o.Elements), true)
		for i, e := range o.Elements {
			r[i], _ = e.AsRune()
		}
		return alloc.NewStringValue(string(r)), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		r := alloc.NewDict(len(o.Elements))
		for i, v := range o.Elements {
			r[strconv.Itoa(i)] = v
		}
		return alloc.NewRecordValue(r, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		r := alloc.NewDict(len(o.Elements))
		for i, v := range o.Elements {
			r[strconv.Itoa(i)] = v
		}
		return alloc.NewDictValue(r, false), nil

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
		return o.Elements[0], nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return o.Elements[len(o.Elements)-1], nil

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(arrayTypeContains(v, args[0])), nil

	case "min":
		return arrayFnMin(v, vm, args)

	case "max":
		return arrayFnMax(v, vm, args)

	case "sum":
		return arrayFnSum(v, vm, args)

	case "avg":
		return arrayFnAvg(v, vm, args)

	case "sort":
		return arrayFnSort(v, vm, args)

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		alloc := vm.Allocator()
		o := (*Array)(v.Ptr)
		n := len(o.Elements)
		t := alloc.NewArray(n, true)
		for i, x := range o.Elements {
			t[n-1-i] = x
		}
		return alloc.NewArrayValue(t, false), nil

	case "filter":
		return arrayFnFilter(v, vm, args)

	case "count":
		return arrayFnCount(v, vm, args)

	case "all":
		return arrayFnAll(v, vm, args)

	case "any":
		return arrayFnAny(v, vm, args)

	case "map":
		return arrayFnMap(v, vm, args)

	case "reduce":
		return arrayFnReduce(v, vm, args)

	case "for_each":
		return arrayFnForEach(v, vm, args)

	case "find":
		return arrayFnFind(v, vm, args)

	case "chunk":
		return arrayFnChunk(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func arrayTypeAccess(v Value, a *Arena, index Value, mode Opcode) (Value, error) {
	o := (*Array)(v.Ptr)

	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}
		i, ok = normalizeSequenceIndex(i, int64(len(o.Elements)))
		if !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), len(o.Elements))
		}
		return o.Elements[i], nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func arrayTypeContains(v Value, e Value) bool {
	o := (*Array)(v.Ptr)
	switch e.Type {
	case VT_ARRAY:
		t := (*Array)(e.Ptr)
		if len(t.Elements) == 0 {
			return true
		}
		if len(o.Elements) < len(t.Elements) {
			return false
		}
		for i := range o.Elements {
			if o.Elements[i].Equal(t.Elements[0]) {
				match := true
				for j := 1; j < len(t.Elements); j++ {
					if i+j >= len(o.Elements) || !o.Elements[i+j].Equal(t.Elements[j]) {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
		return false

	default:
		for i := range o.Elements {
			if o.Elements[i].Equal(e) {
				return true
			}
		}
		return false
	}
}

func arrayTypeAppend(v Value, a *Arena, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	return a.NewArrayValue(append(o.Elements, args...), false), nil
}

func arrayTypeSlice(v Value, a *Arena, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	o := (*Array)(v.Ptr)
	l := int64(len(o.Elements))

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
	return a.NewArrayValue(o.Elements[si:ei], v.Const), nil
}

func arrayTypeSliceStep(v Value, a *Arena, s Value, e Value, stepVal Value) (Value, error) {
	var si, ei int64
	var ok bool

	o := (*Array)(v.Ptr)
	l := int64(len(o.Elements))

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
	result := a.NewArray(0, false)
	if step > 0 {
		for i := start; i < end; i += step {
			result = append(result, o.Elements[i])
		}
	} else {
		for i := start; i > end; i += step {
			result = append(result, o.Elements[i])
		}
	}

	return a.NewArrayValue(result, false), nil
}

func arrayTypeAsBool(v Value) (bool, bool) {
	o := (*Array)(v.Ptr)
	return len(o.Elements) > 0, true
}

func arrayTypeAsString(v Value) (string, bool) {
	rs, ok := arrayTypeAsRunes(v)
	if !ok {
		return "", false
	}
	return string(rs), true
}

func arrayTypeAsRunes(v Value) ([]rune, bool) {
	o := (*Array)(v.Ptr)
	rs := make([]rune, len(o.Elements))
	for i, e := range o.Elements {
		r, ok := e.AsInt()
		if !ok || r < 0 || r > unicode.MaxRune {
			return nil, false
		}
		rs[i] = rune(r)
	}
	return rs, true
}

func arrayTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Array)(v.Ptr)
	bs := make([]byte, len(o.Elements))
	for i, e := range o.Elements {
		b, ok := e.AsInt()
		if !ok || b < 0 || b > 255 {
			return nil, false
		}
		bs[i] = byte(b)
	}
	return bs, true
}

func arrayTypeAsArray(v Value, a *Arena) ([]Value, bool) {
	o := (*Array)(v.Ptr)
	return o.Elements, true
}

func arrayFnChunk(v Value, vm VM, args []Value) (Value, error) {
	size, copyChunks, err := chunkArgs("chunk", args)
	if err != nil {
		return Undefined, err
	}

	o := (*Array)(v.Ptr)
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
			chunk = alloc.NewArray(end-start, true)
			copy(chunk, o.Elements[start:end])
			chunkConst = false
		}
		chunks[i] = alloc.NewArrayValue(chunk, chunkConst)
	}

	return alloc.NewArrayValue(chunks, false), nil
}

func arrayFnForEach(v Value, vm VM, args []Value) (Value, error) {
	fn, err := forEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	o := (*Array)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = v
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

func arrayFnSort(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sort", "0", len(args))
	}

	alloc := vm.Allocator()
	var err error

	o := (*Array)(v.Ptr)
	t := alloc.NewArray(len(o.Elements), true)
	copy(t, o.Elements)
	slices.SortFunc(t, func(a, b Value) int {
		less, e := a.BinaryOp(alloc, token.Less, b)
		if e != nil {
			err = e
			return 0
		}
		if !less.IsTrue() {
			if a.Equal(b) {
				return 0
			}
			return 1
		}
		return -1
	})
	if err != nil {
		return Undefined, err
	}

	return alloc.NewArrayValue(t, false), nil
}

func arrayFnFilter(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	o := (*Array)(v.Ptr)
	alloc := vm.Allocator()
	filtered := alloc.NewArray(len(o.Elements), false)

	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewArrayValue(filtered, false), nil

	case 2:
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewArrayValue(filtered, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnCount(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [2]Value
	var count int64

	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = v
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
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
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

func arrayFnAll(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = v
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

func arrayFnAny(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = v
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

func arrayFnFind(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for i, v := range o.Elements {
			buf[0] = v
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
			buf[1] = v
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

func arrayFnMap(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("map", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("map", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	o := (*Array)(v.Ptr)
	alloc := vm.Allocator()
	mapped := alloc.NewArray(len(o.Elements), true)

	switch fn.Arity() {
	case 1:
		for i, v := range o.Elements {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			mapped[i] = res
		}
		return alloc.NewArrayValue(mapped, false), nil

	case 2:
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = v
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

func arrayFnReduce(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError("reduce", "2", len(args))
	}

	acc := args[0]
	fn := args[1]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("reduce", "second", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [3]Value
	switch fn.Arity() {
	case 2:
		for _, v := range o.Elements {
			buf[0] = acc
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	case 3:
		for i, v := range o.Elements {
			buf[0] = acc
			buf[1] = IntValue(int64(i))
			buf[2] = v
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

func arrayFnMin(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("min", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	alloc := vm.Allocator()
	e := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		less, err := o.Elements[i].BinaryOp(alloc, token.Less, e)
		if err != nil {
			return Undefined, err
		}
		if less.IsTrue() {
			e = o.Elements[i]
		}
	}

	return e, nil
}

func arrayFnMax(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("max", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	alloc := vm.Allocator()
	e := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		greater, err := o.Elements[i].BinaryOp(alloc, token.Greater, e)
		if err != nil {
			return Undefined, err
		}
		if greater.IsTrue() {
			e = o.Elements[i]
		}
	}

	return e, nil
}

func arrayFnSum(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sum", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	alloc := vm.Allocator()
	var err error
	s := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		s, err = s.BinaryOp(alloc, token.Add, o.Elements[i])
		if err != nil {
			return Undefined, err
		}
	}

	return s, nil
}

func arrayFnAvg(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("avg", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	alloc := vm.Allocator()
	var err error
	sum := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		sum, err = sum.BinaryOp(alloc, token.Add, o.Elements[i])
		if err != nil {
			return Undefined, err
		}
	}

	length := IntValue(int64(len(o.Elements)))
	avg, err := sum.BinaryOp(alloc, token.Quo, length)
	if err != nil {
		return Undefined, err
	}

	return avg, nil
}
