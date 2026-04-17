package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/token"
)

type Array struct {
	Elements  []Value
	Immutable bool
}

func (o *Array) Set(elements []Value, immutable bool) {
	o.Elements = elements
	o.Immutable = immutable

	if o.Elements == nil {
		o.Elements = []Value{}
	}
}

// ArrayValue creates boxed array value.
func ArrayValue(v *Array) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_ARRAY,
	}
}

// NewArrayValue creates a new (heap-allocated) array value.
func NewArrayValue(vals []Value, immutable bool) Value {
	t := &Array{}
	t.Set(vals, immutable)
	return ArrayValue(t)
}

/* Array type methods */

func arrayTypeName(v Value) string {
	o := (*Array)(v.Ptr)
	if o.Immutable {
		return "immutable-array"
	}
	return "array"
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
	if err := enc.Encode(o.Immutable); err != nil {
		return nil, fmt.Errorf("array (immutable flag): %w", err)
	}
	if err := enc.Encode(o.Elements); err != nil {
		return nil, fmt.Errorf("array (elements): %w", err)
	}
	return buf.Bytes(), nil
}

func arrayTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var immutable bool
	if err := dec.Decode(&immutable); err != nil {
		return fmt.Errorf("array (immutable flag): %w", err)
	}
	var arr []Value
	if err := dec.Decode(&arr); err != nil {
		return fmt.Errorf("array (elements): %w", err)
	}
	if arr == nil {
		arr = []Value{}
	}
	o := &Array{
		Elements:  arr,
		Immutable: immutable,
	}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func arrayTypeString(v Value) string {
	o := (*Array)(v.Ptr)
	elements := make([]string, len(o.Elements))
	for i, e := range o.Elements {
		elements[i] = e.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

func arrayTypeInterface(v Value) any {
	o := (*Array)(v.Ptr)
	res := make([]any, len(o.Elements))
	for i, val := range o.Elements {
		res[i] = val.Interface()
	}
	return res
}

func arrayTypeBinaryOp(v Value, a Allocator, op token.Token, r Value) (Value, error) {
	if r.Type != VT_ARRAY {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
	}

	la := (*Array)(v.Ptr)
	ra := (*Array)(r.Ptr)
	switch op {
	case token.Add:
		return a.NewArrayValue(append(la.Elements, ra.Elements...), false)
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
}

func arrayTypeEqual(v Value, r Value) bool {
	if r.Type != VT_ARRAY {
		return false
	}

	la := (*Array)(v.Ptr)
	ra := (*Array)(r.Ptr)
	if len(la.Elements) != len(ra.Elements) {
		return false
	}

	for i, e := range la.Elements {
		if !e.Equal(ra.Elements[i]) {
			return false
		}
	}

	return true
}

func arrayTypeCopy(v Value, a Allocator) (Value, error) {
	// Deep copy the array and its elements even if it is immutable (since the elements themselves may be mutable)
	o := (*Array)(v.Ptr)
	c := make([]Value, len(o.Elements))
	for i, e := range o.Elements {
		t, err := e.Copy(a)
		if err != nil {
			return Undefined, err
		}
		c[i] = t
	}
	return a.NewArrayValue(c, false)
}

func arrayTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_array":
		return arrayFnToArray(v, vm, "array.to_array", args)

	case "to_bytes":
		return arrayFnToBytes(v, vm, "array.to_bytes", args)

	case "to_string":
		return arrayFnToString(v, vm, "array.to_string", args)

	case "to_record":
		return arrayFnToRecord(v, vm, "array.to_record", args)

	case "sort":
		return arrayFnSort(v, vm, "array.sort", args)

	case "filter":
		return arrayFnFilter(v, vm, "array.filter", args)

	case "count":
		return arrayFnCount(v, vm, "array.count", args)

	case "all":
		return arrayFnAll(v, vm, "array.all", args)

	case "any":
		return arrayFnAny(v, vm, "array.any", args)

	case "map":
		return arrayFnMap(v, vm, "array.map", args)

	case "reduce":
		return arrayFnReduce(v, vm, "array.reduce", args)

	case "is_empty":
		return arrayFnIsEmpty(v, vm, "array.is_empty", args)

	case "len":
		return arrayFnLen(v, vm, "array.len", args)

	case "first":
		return arrayFnFirst(v, vm, "array.first", args)

	case "last":
		return arrayFnLast(v, vm, "array.last", args)

	case "min":
		return arrayFnMin(v, vm, "array.min", args)

	case "max":
		return arrayFnMax(v, vm, "array.max", args)

	case "sum":
		return arrayFnSum(v, vm, "array.sum", args)

	case "avg":
		return arrayFnAvg(v, vm, "array.avg", args)

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError("array.contains", "1", len(args))
		}
		return BoolValue(arrayTypeContains(v, args[0])), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func arrayTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	o := (*Array)(v.Ptr)

	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("array access", "int", index.TypeName())
		}
		if i < 0 || i >= int64(len(o.Elements)) {
			return Undefined, nil
		}
		return o.Elements[i], nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func arrayTypeAssign(v Value, index Value, r Value) (err error) {
	o := (*Array)(v.Ptr)
	if o.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	i, ok := index.AsInt()
	if !ok {
		return errs.NewInvalidIndexTypeError("array assignment", "int", index.TypeName())
	}
	if i < 0 || i >= int64(len(o.Elements)) {
		return errs.NewIndexOutOfBoundsError("array assignment", int(i), len(o.Elements))
	}

	o.Elements[i] = r

	return nil
}

func arrayTypeIsIterable(v Value) bool {
	return true
}

func arrayTypeIterator(v Value, a Allocator) (Value, error) {
	o := (*Array)(v.Ptr)
	return a.NewArrayIteratorValue(o.Elements)
}

func arrayTypeIsImmutable(v Value) bool {
	o := (*Array)(v.Ptr)
	return o.Immutable
}

func arrayTypeIsTrue(v Value) bool {
	o := (*Array)(v.Ptr)
	return len(o.Elements) > 0
}

func arrayTypeAsString(v Value) (string, bool) {
	return arrayTypeString(v), true
}

func arrayTypeAsBool(v Value) (bool, bool) {
	return arrayTypeIsTrue(v), true
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

func arrayTypeAsArray(v Value, a Allocator) ([]Value, bool) {
	o := (*Array)(v.Ptr)
	return o.Elements, true
}

func arrayFnSort(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}

	alloc := vm.Allocator()
	r, err := arrayTypeCopy(v, alloc)
	if err != nil {
		return Undefined, err
	}
	t := (*Array)(r.Ptr)
	slices.SortFunc(t.Elements, func(a, b Value) int {
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
	return r, err
}

func arrayFnFilter(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		filtered := make([]Value, 0, len(o.Elements))
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
		return alloc.NewArrayValue(filtered, false)

	case 2:
		filtered := make([]Value, 0, len(o.Elements))
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
		return alloc.NewArrayValue(filtered, false)

	default:
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnCount(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		var count int64
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
		var count int64
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
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnAll(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
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
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnAny(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
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
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnMap(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		mapped := make([]Value, 0, len(o.Elements))
		for _, v := range o.Elements {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			mapped = append(mapped, res)
		}
		return alloc.NewArrayValue(mapped, false)

	case 2:
		mapped := make([]Value, 0, len(o.Elements))
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			mapped = append(mapped, res)
		}
		return alloc.NewArrayValue(mapped, false)

	default:
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnReduce(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}

	acc := args[0]
	fn := args[1]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "second", "non-variadic function", fn.TypeName())
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
		return Undefined, errs.NewInvalidArgumentTypeError(name, "second", "f/2 or f/3", fn.TypeName())
	}
}

func arrayFnToArray(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return v, nil
}

func arrayFnToBytes(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	bs := make([]byte, len(o.Elements))
	for i, e := range o.Elements {
		b, ok := e.AsInt()
		if !ok || b < 0 || b > 255 {
			b = 0
		}
		bs[i] = byte(b)
	}
	return vm.Allocator().NewBytesValue(bs)
}

func arrayFnToString(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	r := make([]rune, len(o.Elements))
	for i, e := range o.Elements {
		rv, ok := e.AsChar()
		if !ok {
			rv = ' '
		}
		r[i] = rv
	}
	return vm.Allocator().NewStringValue(string(r))
}

func arrayFnToRecord(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	r := make(map[string]Value, len(o.Elements))
	for i, v := range o.Elements {
		r[strconv.Itoa(i)] = v
	}
	return vm.Allocator().NewRecordValue(r, false)
}

func arrayFnIsEmpty(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	return BoolValue(len(o.Elements) == 0), nil
}

func arrayFnLen(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	return IntValue(int64(len(o.Elements))), nil
}

func arrayFnFirst(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}
	return o.Elements[0], nil
}

func arrayFnLast(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}
	return o.Elements[len(o.Elements)-1], nil
}

func arrayFnMin(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
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

func arrayFnMax(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
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

func arrayFnSum(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
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

func arrayFnAvg(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
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

func arrayTypeLen(v Value) int64 {
	o := (*Array)(v.Ptr)
	return int64(len(o.Elements))
}

func arrayTypeAppend(v Value, a Allocator, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	return a.NewArrayValue(append(o.Elements, args...), false)
}

func arrayTypeSlice(v Value, a Allocator, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	o := (*Array)(v.Ptr)
	l := int64(len(o.Elements))

	if s.Type != VT_UNDEFINED {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("array slice", "int", s.TypeName())
		}
	}

	if e.Type == VT_UNDEFINED {
		ei = l
	} else {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("array slice", "int", e.TypeName())
		}
	}

	if si > ei {
		return Undefined, fmt.Errorf("invalid slice index: %d > %d", si, ei)
	}

	if si < 0 {
		si = 0
	} else if si > l {
		si = l
	}

	if ei < 0 {
		ei = 0
	} else if ei > l {
		ei = l
	}

	return a.NewArrayValue(o.Elements[si:ei], false)
}

func arrayTypeImmutable(v Value, a Allocator) (Value, error) {
	o := (*Array)(v.Ptr)
	if o.Immutable {
		return v, nil
	}
	return a.NewArrayValue(o.Elements, true)
}
