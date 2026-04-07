package value

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
)

type Array struct {
	Object
	value     []core.Value
	immutable bool
}

func (o *Array) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	var vals []core.Value
	if err := dec.Decode(&vals); err != nil {
		return err
	}

	var immutable bool
	if err := dec.Decode(&immutable); err != nil {
		return err
	}

	o.Set(vals, immutable)
	return nil
}

func (o *Array) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(o.value); err != nil {
		return nil, err
	}

	if err := enc.Encode(o.immutable); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (o *Array) Set(val []core.Value, immutable bool) {
	o.value = val
	if o.value == nil {
		o.value = []core.Value{}
	}
	o.immutable = immutable
}

func (o *Array) Value() []core.Value {
	return o.value
}

func (o *Array) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *Array) Len() int {
	return len(o.value)
}

func (o *Array) Slice(s, e int) []core.Value {
	return o.value[s:e]
}

func (o *Array) At(i int) core.Value {
	return o.value[i]
}

func (o *Array) Append(vals ...core.Value) {
	o.value = append(o.value, vals...)
}

func (o *Array) SetAt(i int, val core.Value) {
	o.value[i] = val
}

func (o *Array) TypeName() string {
	if o.immutable {
		return "immutable-array"
	}
	return "array"
}

func (o *Array) String() string {
	elements := make([]string, len(o.value))
	for i, e := range o.value {
		elements[i] = e.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

func (o *Array) Interface() any {
	res := make([]any, len(o.value))
	for i, val := range o.value {
		res[i] = val.Interface()
	}
	return res
}

func (o *Array) BinaryOp(vm core.VM, op token.Token, rhs core.Value) (core.Value, error) {
	if !rhs.IsObject() {
		return core.UndefinedValue(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
	}

	alloc := vm.Allocator()
	if rhs, ok := rhs.Object().(*Array); ok {
		switch op {
		case token.Add:
			return alloc.NewArrayValue(append(o.value, rhs.value...), false), nil
		}
	}

	return core.UndefinedValue(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
}

func (o *Array) Equals(x core.Value) bool {
	if !x.IsObject() {
		return false
	}

	switch x := x.Object().(type) {
	case *Array:
		if len(o.value) != len(x.value) {
			return false
		}
		for i, e := range o.value {
			if !e.Equals(x.value[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (o *Array) Copy(alloc core.Allocator) core.Value {
	// Deep copy the array and its elements even if it is immutable (since the elements themselves may be mutable)
	c := make([]core.Value, len(o.value))
	for i, e := range o.value {
		c[i] = e.Copy(alloc)
	}
	return alloc.NewArrayValue(c, false)
}

func (o *Array) Method(vm core.VM, name string, args []core.Value) (core.Value, error) {
	switch name {
	case "to_array":
		return o.fnToArray(vm, "array.to_array", args)

	case "to_bytes":
		return o.fnToBytes(vm, "array.to_bytes", args)

	case "to_string":
		return o.fnToString(vm, "array.to_string", args)

	case "to_record":
		return o.fnToRecord(vm, "array.to_record", args)

	case "sort":
		return o.fnSort(vm, "array.sort", args)

	case "filter":
		return o.fnFilter(vm, "array.filter", args)

	case "count":
		return o.fnCount(vm, "array.count", args)

	case "all":
		return o.fnAll(vm, "array.all", args)

	case "any":
		return o.fnAny(vm, "array.any", args)

	case "map":
		return o.fnMap(vm, "array.map", args)

	case "reduce":
		return o.fnReduce(vm, "array.reduce", args)

	case "is_empty":
		return o.fnIsEmpty(vm, "array.is_empty", args)

	case "len":
		return o.fnLen(vm, "array.len", args)

	case "first":
		return o.fnFirst(vm, "array.first", args)

	case "last":
		return o.fnLast(vm, "array.last", args)

	case "min":
		return o.fnMin(vm, "array.min", args)

	case "max":
		return o.fnMax(vm, "array.max", args)

	case "sum":
		return o.fnSum(vm, "array.sum", args)

	case "avg":
		return o.fnAvg(vm, "array.avg", args)

	default:
		return core.UndefinedValue(), core.NewInvalidMethodError(name, o.TypeName())
	}
}

func (o *Array) Access(vm core.VM, index core.Value, mode core.Opcode) (core.Value, error) {
	if mode == parser.OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return core.UndefinedValue(), core.NewInvalidIndexTypeError("array access", "int", index.TypeName())
		}
		if i < 0 || i >= int64(len(o.value)) {
			return core.UndefinedValue(), nil
		}
		return o.value[i], nil
	}

	k, ok := index.AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidSelectorError(o.TypeName(), k)
	}
	return core.UndefinedValue(), core.NewInvalidSelectorError(o.TypeName(), k)
}

func (o *Array) Assign(index, value core.Value) (err error) {
	if o.immutable {
		return core.NewNotAssignableError(o.TypeName())
	}

	i, ok := index.AsInt()
	if !ok {
		return core.NewInvalidIndexTypeError("array assignment", "int", index.TypeName())
	}
	if i < 0 || i >= int64(len(o.value)) {
		return core.NewIndexOutOfBoundsError("array assignment", int(i), len(o.value))
	}
	o.value[i] = value

	return nil
}

func (o *Array) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewArrayIterator(o.value)
}

func (o *Array) IsImmutable() bool {
	return o.immutable
}

func (o *Array) IsArray() bool {
	return true
}

func (o *Array) IsTrue() bool {
	return len(o.value) > 0
}

func (o *Array) IsFalse() bool {
	return len(o.value) == 0
}

func (o *Array) IsIterable() bool {
	return true
}

func (o *Array) AsString() (string, bool) {
	return o.String(), true
}

func (o *Array) AsBool() (bool, bool) {
	return o.IsTrue(), true
}

func (o *Array) AsBytes() ([]byte, bool) {
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

func (o *Array) fnSort(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}

	r := o.Copy(vm.Allocator())
	t := r.Object().(*Array)
	var err error
	slices.SortFunc(t.value, func(a, b core.Value) int {
		less, e := a.BinaryOp(vm, token.Less, b)
		if e != nil {
			err = e
			return 0
		}
		if less.IsFalse() {
			if a.Equals(b) {
				return 0
			}
			return 1
		}
		return -1
	})
	return r, err
}

func (o *Array) fnFilter(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]core.Value
	switch fn.Arity() {
	case 1:
		filtered := make([]core.Value, 0, len(o.value))
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewArrayValue(filtered, false), nil

	case 2:
		filtered := make([]core.Value, 0, len(o.value))
		for i, v := range o.value {
			buf[0] = core.IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewArrayValue(filtered, false), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func (o *Array) fnCount(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]core.Value
	var count int64
	switch fn.Arity() {
	case 1:
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				count++
			}
		}

	case 2:
		for i, v := range o.value {
			buf[0] = core.IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				count++
			}
		}

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}

	return core.IntValue(count), nil
}

func (o *Array) fnAll(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]core.Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsFalse() {
				return core.BoolValue(false), nil
			}
		}
		return core.BoolValue(true), nil

	case 2:
		for i, v := range o.value {
			buf[0] = core.IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsFalse() {
				return core.BoolValue(false), nil
			}
		}
		return core.BoolValue(true), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func (o *Array) fnAny(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]core.Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				return core.BoolValue(true), nil
			}
		}
		return core.BoolValue(false), nil

	case 2:
		for i, v := range o.value {
			buf[0] = core.IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			if res.IsTrue() {
				return core.BoolValue(true), nil
			}
		}
		return core.BoolValue(false), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func (o *Array) fnMap(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "non-variadic function", fn.TypeName())
	}

	alloc := vm.Allocator()
	var buf [2]core.Value
	switch fn.Arity() {
	case 1:
		mapped := make([]core.Value, 0, len(o.value))
		for _, v := range o.value {
			buf[0] = v
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return core.UndefinedValue(), err
			}
			mapped = append(mapped, res)
		}
		return alloc.NewArrayValue(mapped, false), nil

	case 2:
		mapped := make([]core.Value, 0, len(o.value))
		for i, v := range o.value {
			buf[0] = core.IntValue(int64(i))
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			mapped = append(mapped, res)
		}
		return alloc.NewArrayValue(mapped, false), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
}

func (o *Array) fnReduce(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "2", len(args))
	}

	acc := args[0]
	fn := args[1]
	if !fn.IsCallable() || fn.IsVariadic() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "second", "non-variadic function", fn.TypeName())
	}

	var buf [3]core.Value
	switch fn.Arity() {
	case 2:
		for _, v := range o.value {
			buf[0] = acc
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return core.UndefinedValue(), err
			}
			acc = res
		}
		return acc, nil

	case 3:
		for i, v := range o.value {
			buf[0] = acc
			buf[1] = core.IntValue(int64(i))
			buf[2] = v
			res, err := fn.Call(vm, buf[:3])
			if err != nil {
				return core.UndefinedValue(), err
			}
			acc = res
		}
		return acc, nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "second", "f/2 or f/3", fn.TypeName())
	}
}

func (o *Array) fnToArray(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return core.ObjectValue(o), nil
}

func (o *Array) fnToBytes(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
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

func (o *Array) fnToString(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
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

func (o *Array) fnToRecord(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	r := make(map[string]core.Value, len(o.value))
	for i, v := range o.value {
		r[strconv.Itoa(i)] = v
	}
	return vm.Allocator().NewRecordValue(r, false), nil
}

func (o *Array) fnIsEmpty(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return core.BoolValue(len(o.value) == 0), nil
}

func (o *Array) fnLen(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return core.IntValue(int64(len(o.value))), nil
}

func (o *Array) fnFirst(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if len(o.value) == 0 {
		return core.UndefinedValue(), nil
	}
	return o.value[0], nil
}

func (o *Array) fnLast(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if len(o.value) == 0 {
		return core.UndefinedValue(), nil
	}
	return o.value[len(o.value)-1], nil
}

func (o *Array) fnMin(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if len(o.value) == 0 {
		return core.UndefinedValue(), nil
	}

	v := o.value[0]
	for i := 1; i < len(o.value); i++ {
		less, err := o.value[i].BinaryOp(vm, token.Less, v)
		if err != nil {
			return core.UndefinedValue(), err
		}
		if less.IsTrue() {
			v = o.value[i]
		}
	}

	return v, nil
}

func (o *Array) fnMax(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if len(o.value) == 0 {
		return core.UndefinedValue(), nil
	}

	v := o.value[0]
	for i := 1; i < len(o.value); i++ {
		greater, err := o.value[i].BinaryOp(vm, token.Greater, v)
		if err != nil {
			return core.UndefinedValue(), err
		}
		if greater.IsTrue() {
			v = o.value[i]
		}
	}

	return v, nil
}

func (o *Array) fnSum(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if len(o.value) == 0 {
		return core.UndefinedValue(), nil
	}

	var err error
	v := o.value[0]
	for i := 1; i < len(o.value); i++ {
		v, err = v.BinaryOp(vm, token.Add, o.value[i])
		if err != nil {
			return core.UndefinedValue(), err
		}
	}

	return v, nil
}

func (o *Array) fnAvg(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if len(o.value) == 0 {
		return core.UndefinedValue(), nil
	}

	var err error
	sum := o.value[0]
	for i := 1; i < len(o.value); i++ {
		sum, err = sum.BinaryOp(vm, token.Add, o.value[i])
		if err != nil {
			return core.UndefinedValue(), err
		}
	}

	length := core.IntValue(int64(len(o.value)))
	avg, err := sum.BinaryOp(vm, token.Quo, length)
	if err != nil {
		return core.UndefinedValue(), err
	}

	return avg, nil
}
