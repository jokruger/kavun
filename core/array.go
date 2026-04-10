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
	value     []Value
	immutable bool
}

func (o *Array) Set(vals []Value, immutable bool) {
	o.value = vals
	o.immutable = immutable

	if o.value == nil {
		o.value = []Value{}
	}
}

func (o *Array) Value() []Value {
	return o.value
}

func (o *Array) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *Array) Len() int {
	return len(o.value)
}

func (o *Array) Slice(s, e int) []Value {
	return o.value[s:e]
}

func (o *Array) At(i int) Value {
	return o.value[i]
}

func (o *Array) Append(vals ...Value) {
	o.value = append(o.value, vals...)
}

func (o *Array) SetAt(i int, val Value) {
	o.value[i] = val
}

func ArrayValue(v *Array) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_ARRAY,
	}
}

func NewArrayValue(vals []Value, immutable bool) Value {
	t := &Array{}
	t.Set(vals, immutable)
	return ArrayValue(t)
}

func arrayTypeName(v Value) string {
	o := (*Array)(v.Ptr)
	if o.immutable {
		return "immutable-array"
	}
	return "array"
}

func arrayTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Array)(v.Ptr)
	var b []byte
	b = append(b, '[')
	len1 := o.Len() - 1
	for idx, elem := range o.Value() {
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
	if err := enc.Encode(o.immutable); err != nil {
		return nil, fmt.Errorf("array (immutable flag): %w", err)
	}
	if err := enc.Encode(o.value); err != nil {
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
		value:     arr,
		immutable: immutable,
	}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func arrayTypeString(v Value) string {
	o := (*Array)(v.Ptr)
	elements := make([]string, len(o.value))
	for i, e := range o.value {
		elements[i] = e.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

func arrayTypeInterface(v Value) any {
	o := (*Array)(v.Ptr)
	res := make([]any, len(o.value))
	for i, val := range o.value {
		res[i] = val.Interface()
	}
	return res
}

func arrayTypeBinaryOp(v Value, a Allocator, op token.Token, r Value) (Value, error) {
	if !r.IsArray() {
		return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
	}

	la := (*Array)(v.Ptr)
	ra := (*Array)(r.Ptr)
	switch op {
	case token.Add:
		return a.NewArrayValue(append(la.value, ra.value...), false), nil
	}

	return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
}

func arrayTypeEqual(v Value, r Value) bool {
	if !r.IsArray() {
		return false
	}

	la := (*Array)(v.Ptr)
	ra := (*Array)(r.Ptr)
	if len(la.value) != len(ra.value) {
		return false
	}

	for i, e := range la.value {
		if !e.Equal(ra.value[i]) {
			return false
		}
	}

	return true
}

func arrayTypeCopy(v Value, a Allocator) Value {
	// Deep copy the array and its elements even if it is immutable (since the elements themselves may be mutable)
	o := (*Array)(v.Ptr)
	c := make([]Value, len(o.value))
	for i, e := range o.value {
		c[i] = e.Copy(a)
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

	default:
		return UndefinedValue(), errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func arrayTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	o := (*Array)(v.Ptr)

	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return UndefinedValue(), errs.NewInvalidIndexTypeError("array access", "int", index.TypeName())
		}
		if i < 0 || i >= int64(len(o.value)) {
			return UndefinedValue(), nil
		}
		return o.value[i], nil
	}

	k, ok := index.AsString()
	if !ok {
		return UndefinedValue(), errs.NewInvalidIndexTypeError("array selector access", "string", index.TypeName())
	}

	return UndefinedValue(), errs.NewInvalidSelectorError(v.TypeName(), k)
}

func arrayTypeAssign(v Value, index Value, r Value) (err error) {
	o := (*Array)(v.Ptr)
	if o.immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	i, ok := index.AsInt()
	if !ok {
		return errs.NewInvalidIndexTypeError("array assignment", "int", index.TypeName())
	}
	if i < 0 || i >= int64(len(o.value)) {
		return errs.NewIndexOutOfBoundsError("array assignment", int(i), len(o.value))
	}

	o.value[i] = r

	return nil
}

func arrayTypeIsIterable(v Value) bool {
	return true
}

func arrayTypeIterator(v Value, a Allocator) Value {
	o := (*Array)(v.Ptr)
	return a.NewArrayIteratorValue(o.value)
}

func arrayTypeIsImmutable(v Value) bool {
	o := (*Array)(v.Ptr)
	return o.immutable
}

func arrayTypeIsTrue(v Value) bool {
	o := (*Array)(v.Ptr)
	return len(o.value) > 0
}

func arrayTypeAsString(v Value) (string, bool) {
	return arrayTypeString(v), true
}

func arrayTypeAsBool(v Value) (bool, bool) {
	return arrayTypeIsTrue(v), true
}

func arrayTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Array)(v.Ptr)
	bs := make([]byte, len(o.value))
	for i, e := range o.value {
		b, ok := e.AsInt()
		if !ok || b < 0 || b > 255 {
			return nil, false
		}
		bs[i] = byte(b)
	}
	return bs, true
}

func arrayFnSort(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}

	alloc := vm.Allocator()
	r := arrayTypeCopy(v, alloc)
	t := (*Array)(r.Ptr)
	var err error
	slices.SortFunc(t.value, func(a, b Value) int {
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
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		filtered := make([]Value, 0, len(o.value))
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewArrayValue(filtered, false), nil

	case 2:
		filtered := make([]Value, 0, len(o.value))
		for i, v := range o.value {
			buf[0] = IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewArrayValue(filtered, false), nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnCount(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		var count int64
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	case 2:
		var count int64
		for i, v := range o.value {
			buf[0] = IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnAll(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return UndefinedValue(), err
			}
			if !res.IsTrue() {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	case 2:
		for i, v := range o.value {
			buf[0] = IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			if !res.IsTrue() {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnAny(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	case 2:
		for i, v := range o.value {
			buf[0] = IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			if res.IsTrue() {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnMap(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 1 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		mapped := make([]Value, 0, len(o.value))
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return UndefinedValue(), err
			}
			mapped = append(mapped, res)
		}
		return alloc.NewArrayValue(mapped, false), nil

	case 2:
		mapped := make([]Value, 0, len(o.value))
		for i, v := range o.value {
			buf[0] = IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			mapped = append(mapped, res)
		}
		return alloc.NewArrayValue(mapped, false), nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func arrayFnReduce(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 2 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "2", len(args))
	}

	acc := args[0]
	fn := args[1]
	if !fn.IsCallable() || fn.IsVariadic() {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "second", "non-variadic function", fn.TypeName())
	}

	o := (*Array)(v.Ptr)
	var buf [3]Value
	switch fn.Arity() {
	case 2:
		for _, v := range o.value {
			buf[0] = acc
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return UndefinedValue(), err
			}
			acc = res
		}
		return acc, nil

	case 3:
		for i, v := range o.value {
			buf[0] = acc
			buf[1] = IntValue(int64(i))
			buf[2] = v
			res, err := fn.Call(vm, buf[:3])
			if err != nil {
				return UndefinedValue(), err
			}
			acc = res
		}
		return acc, nil

	default:
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "second", "f/2 or f/3", fn.TypeName())
	}
}

func arrayFnToArray(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return v, nil
}

func arrayFnToBytes(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	bs := make([]byte, len(o.value))
	for i, e := range o.value {
		b, ok := e.AsInt()
		if !ok || b < 0 || b > 255 {
			b = 0
		}
		bs[i] = byte(b)
	}
	return vm.Allocator().NewBytesValue(bs), nil
}

func arrayFnToString(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	r := make([]rune, len(o.value))
	for i, e := range o.value {
		rv, ok := e.AsChar()
		if !ok {
			rv = ' '
		}
		r[i] = rv
	}
	return vm.Allocator().NewStringValue(string(r)), nil
}

func arrayFnToRecord(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	r := make(map[string]Value, len(o.value))
	for i, v := range o.value {
		r[strconv.Itoa(i)] = v
	}
	return vm.Allocator().NewRecordValue(r, false), nil
}

func arrayFnIsEmpty(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	return BoolValue(len(o.value) == 0), nil
}

func arrayFnLen(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	return IntValue(int64(len(o.value))), nil
}

func arrayFnFirst(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	if len(o.value) == 0 {
		return UndefinedValue(), nil
	}
	return o.value[0], nil
}

func arrayFnLast(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Array)(v.Ptr)
	if len(o.value) == 0 {
		return UndefinedValue(), nil
	}
	return o.value[len(o.value)-1], nil
}

func arrayFnMin(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.value) == 0 {
		return UndefinedValue(), nil
	}

	alloc := vm.Allocator()
	e := o.value[0]
	for i := 1; i < len(o.value); i++ {
		less, err := o.value[i].BinaryOp(alloc, token.Less, e)
		if err != nil {
			return UndefinedValue(), err
		}
		if less.IsTrue() {
			e = o.value[i]
		}
	}

	return e, nil
}

func arrayFnMax(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.value) == 0 {
		return UndefinedValue(), nil
	}

	alloc := vm.Allocator()
	e := o.value[0]
	for i := 1; i < len(o.value); i++ {
		greater, err := o.value[i].BinaryOp(alloc, token.Greater, e)
		if err != nil {
			return UndefinedValue(), err
		}
		if greater.IsTrue() {
			e = o.value[i]
		}
	}

	return e, nil
}

func arrayFnSum(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.value) == 0 {
		return UndefinedValue(), nil
	}

	alloc := vm.Allocator()
	var err error
	s := o.value[0]
	for i := 1; i < len(o.value); i++ {
		s, err = s.BinaryOp(alloc, token.Add, o.value[i])
		if err != nil {
			return UndefinedValue(), err
		}
	}

	return s, nil
}

func arrayFnAvg(v Value, vm VM, name string, args []Value) (Value, error) {
	if len(args) != 0 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.value) == 0 {
		return UndefinedValue(), nil
	}

	alloc := vm.Allocator()
	var err error
	sum := o.value[0]
	for i := 1; i < len(o.value); i++ {
		sum, err = sum.BinaryOp(alloc, token.Add, o.value[i])
		if err != nil {
			return UndefinedValue(), err
		}
	}

	length := IntValue(int64(len(o.value)))
	avg, err := sum.BinaryOp(alloc, token.Quo, length)
	if err != nil {
		return UndefinedValue(), err
	}

	return avg, nil
}
