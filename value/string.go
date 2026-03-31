package value

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/araddon/dateparse"
	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/internal/conv"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
)

type String struct {
	Object
	value []rune
}

// Should be used only for static initialization. For dynamic creation of built-in functions, use Allocator.NewString.
func NewStaticString(v string) core.Object {
	o := &String{}
	o.Set(v)
	return o
}

func (o *String) GobDecode(b []byte) error {
	o.Set(string(b))
	return nil
}

func (o *String) GobEncode() ([]byte, error) {
	return []byte(string(o.value)), nil
}

func (o *String) Set(s string) {
	o.value = []rune(s)
}

func (o *String) Value() string {
	return string(o.value)
}

func (o *String) Runes() []rune {
	return o.value
}

func (o *String) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *String) Len() int {
	return len(o.value)
}

func (o *String) At(i int) rune {
	return o.value[i]
}

func (o *String) Get(i int) (rune, bool) {
	if i < 0 || i >= len(o.value) {
		return 0, false
	}
	return o.value[i], true
}

func (o *String) Substring(start, end int) string {
	return string(o.value[start:end])
}

func (o *String) Append(s string) {
	o.value = append(o.value, []rune(s)...)
}

func (o *String) TypeName() string {
	return "string"
}

func (o *String) String() string {
	return strconv.Quote(string(o.value))
}

func (o *String) Interface() any {
	return string(o.value)
}

func (o *String) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	alloc := vm.Allocator()
	v, ok := rhs.AsString()
	if !ok {
		return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
	}

	switch op {
	case token.Add:
		return alloc.NewString(string(o.value) + v), nil
	case token.Less:
		return alloc.NewBool(string(o.value) < v), nil
	case token.LessEq:
		return alloc.NewBool(string(o.value) <= v), nil
	case token.Greater:
		return alloc.NewBool(string(o.value) > v), nil
	case token.GreaterEq:
		return alloc.NewBool(string(o.value) >= v), nil
	}

	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *String) Equals(x core.Object) bool {
	t, ok := x.AsString()
	if !ok {
		return false
	}
	return string(o.value) == t
}

func (o *String) Copy(alloc core.Allocator) core.Object {
	return alloc.NewString(string(o.value))
}

func (o *String) Access(vm core.VM, index core.Object, mode core.Opcode) (core.Object, error) {
	alloc := vm.Allocator()

	if mode == parser.OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return nil, core.NewInvalidIndexTypeError("string access", "int", index)
		}
		if i < 0 || i >= int64(len(o.value)) {
			return alloc.NewUndefined(), nil
		}
		return alloc.NewChar(o.value[i]), nil
	}

	k, ok := index.AsString()
	if !ok {
		return nil, core.NewInvalidSelectorError(o, k)
	}

	switch k {
	case "string":
		return o, nil

	case "array":
		arr := make([]core.Object, len(o.value))
		for i, r := range o.value {
			arr[i] = alloc.NewChar(r)
		}
		return alloc.NewArray(arr, false), nil

	case "bool":
		b, _ := o.AsBool()
		return alloc.NewBool(b), nil

	case "bytes":
		return alloc.NewBytes([]byte(string(o.value))), nil

	case "char":
		if len(o.value) == 1 {
			return alloc.NewChar(o.value[0]), nil
		}
		return alloc.NewChar(0), nil

	case "float":
		f, _ := o.AsFloat()
		return alloc.NewFloat(f), nil

	case "int":
		i, _ := o.AsInt()
		return alloc.NewInt(i), nil

	case "time":
		t, _ := o.AsTime()
		return alloc.NewTime(t), nil

	case "record":
		m := make(map[string]core.Object, len(o.value))
		for i, r := range o.value {
			m[strconv.Itoa(i)] = alloc.NewChar(r)
		}
		return alloc.NewRecord(m, false), nil

	case "empty":
		return alloc.NewBool(len(o.value) == 0), nil

	case "len":
		return alloc.NewInt(int64(len(o.value))), nil

	case "first":
		if len(o.value) == 0 {
			return alloc.NewUndefined(), nil
		}
		return alloc.NewChar(o.value[0]), nil

	case "last":
		if len(o.value) == 0 {
			return alloc.NewUndefined(), nil
		}
		return alloc.NewChar(o.value[len(o.value)-1]), nil

	case "lower":
		t := make([]rune, len(o.value))
		for i, r := range o.value {
			t[i] = unicode.ToLower(r)
		}
		return alloc.NewString(string(t)), nil

	case "upper":
		t := make([]rune, len(o.value))
		for i, r := range o.value {
			t[i] = unicode.ToUpper(r)
		}
		return alloc.NewString(string(t)), nil

	case "trim":
		return o.fnTrim(vm, "string.trim")

	default:
		return nil, core.NewInvalidSelectorError(o, k)
	}
}

func (o *String) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *String) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewStringIterator(o.value)
}

func (o *String) IsTrue() bool {
	return len(o.value) > 0
}

func (o *String) IsFalse() bool {
	return len(o.value) == 0
}

func (o *String) IsIterable() bool {
	return true
}

func (o *String) IsImmutable() bool {
	return true
}

func (o *String) AsString() (string, bool) {
	return string(o.value), true
}

func (o *String) AsInt() (int64, bool) {
	i, err := strconv.ParseInt(string(o.value), 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func (o *String) AsFloat() (float64, bool) {
	f, err := strconv.ParseFloat(string(o.value), 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func (o *String) AsBool() (bool, bool) {
	return conv.ParseBool(string(o.value))
}

func (o *String) AsRune() (rune, bool) {
	if len(o.value) == 1 {
		return o.value[0], true
	}
	return 0, false
}

func (o *String) AsBytes() ([]byte, bool) {
	return []byte(string(o.value)), true
}

func (o *String) AsTime() (time.Time, bool) {
	val, err := dateparse.ParseAny(string(o.value))
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

func (o *String) fnTrim(vm core.VM, name string) (core.Object, error) {
	return vm.Allocator().NewBuiltinFunction(name, func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) > 1 {
			return nil, core.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}

		if len(args) == 0 {
			return vm.Allocator().NewString(strings.Trim(string(o.value), " \t\n")), nil
		}

		s, ok := args[0].AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError(name, "first", "string", args[0])
		}

		return vm.Allocator().NewString(strings.Trim(string(o.value), s)), nil
	}, 0, true), nil
}
