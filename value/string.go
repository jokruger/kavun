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

func NewStaticString(v string) core.Value {
	o := &String{}
	o.Set(v)
	return core.ObjectValue(o)
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

func (o *String) BinaryOp(vm core.VM, op token.Token, rhs core.Value) (core.Value, error) {
	alloc := vm.Allocator()
	v, ok := rhs.AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
	}

	switch op {
	case token.Add:
		return alloc.NewStringValue(string(o.value) + v), nil
	case token.Less:
		return core.BoolValue(string(o.value) < v), nil
	case token.LessEq:
		return core.BoolValue(string(o.value) <= v), nil
	case token.Greater:
		return core.BoolValue(string(o.value) > v), nil
	case token.GreaterEq:
		return core.BoolValue(string(o.value) >= v), nil
	}

	return core.UndefinedValue(), core.NewInvalidBinaryOperatorError(op.String(), o.TypeName(), rhs.TypeName())
}

func (o *String) Equals(x core.Value) bool {
	t, ok := x.AsString()
	if !ok {
		return false
	}
	return string(o.value) == t
}

func (o *String) Copy(alloc core.Allocator) core.Value {
	return alloc.NewStringValue(string(o.value))
}

func (o *String) Method(vm core.VM, name string, args []core.Value) (core.Value, error) {
	switch name {
	case "to_string":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.to_string", "0", len(args))
		}
		return core.ObjectValue(o), nil

	case "to_array":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.to_array", "0", len(args))
		}
		arr := make([]core.Value, len(o.value))
		for i, r := range o.value {
			arr[i] = core.CharValue(r)
		}
		return vm.Allocator().NewArrayValue(arr, false), nil

	case "to_bool":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.to_bool", "0", len(args))
		}
		b, _ := o.AsBool()
		return core.BoolValue(b), nil

	case "to_bytes":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.to_bytes", "0", len(args))
		}
		return vm.Allocator().NewBytesValue([]byte(string(o.value))), nil

	case "to_char":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.to_char", "0", len(args))
		}
		if len(o.value) == 1 {
			return core.CharValue(o.value[0]), nil
		}
		return core.CharValue(0), nil

	case "to_float":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.to_float", "0", len(args))
		}
		f, _ := o.AsFloat()
		return core.FloatValue(f), nil

	case "to_int":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.to_int", "0", len(args))
		}
		i, _ := o.AsInt()
		return core.IntValue(i), nil

	case "to_time":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.to_time", "0", len(args))
		}
		t, _ := o.AsTime()
		return vm.Allocator().NewTimeValue(t), nil

	case "to_record":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.to_record", "0", len(args))
		}
		m := make(map[string]core.Value, len(o.value))
		for i, r := range o.value {
			m[strconv.Itoa(i)] = core.CharValue(r)
		}
		return vm.Allocator().NewRecordValue(m, false), nil

	case "is_empty":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.is_empty", "0", len(args))
		}
		return core.BoolValue(len(o.value) == 0), nil

	case "len":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.len", "0", len(args))
		}
		return core.IntValue(int64(len(o.value))), nil

	case "first":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.first", "0", len(args))
		}
		if len(o.value) == 0 {
			return core.UndefinedValue(), nil
		}
		return core.CharValue(o.value[0]), nil

	case "last":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.last", "0", len(args))
		}
		if len(o.value) == 0 {
			return core.UndefinedValue(), nil
		}
		return core.CharValue(o.value[len(o.value)-1]), nil

	case "lower":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.lower", "0", len(args))
		}
		t := make([]rune, len(o.value))
		for i, r := range o.value {
			t[i] = unicode.ToLower(r)
		}
		return vm.Allocator().NewStringValue(string(t)), nil

	case "upper":
		if len(args) != 0 {
			return core.UndefinedValue(), core.NewWrongNumArgumentsError("string.upper", "0", len(args))
		}
		t := make([]rune, len(o.value))
		for i, r := range o.value {
			t[i] = unicode.ToUpper(r)
		}
		return vm.Allocator().NewStringValue(string(t)), nil

	case "trim":
		return o.fnTrim(vm, "string.trim", args)

	default:
		return core.UndefinedValue(), core.NewInvalidMethodError(name, o.TypeName())
	}
}

func (o *String) Access(vm core.VM, index core.Value, mode core.Opcode) (core.Value, error) {
	if mode == parser.OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return core.UndefinedValue(), core.NewInvalidIndexTypeError("string access", "int", index.TypeName())
		}
		if i < 0 || i >= int64(len(o.value)) {
			return core.UndefinedValue(), nil
		}
		return core.CharValue(o.value[i]), nil
	}

	k, ok := index.AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidSelectorError(o.TypeName(), k)
	}
	return core.UndefinedValue(), core.NewInvalidSelectorError(o.TypeName(), k)
}

func (o *String) Assign(core.Value, core.Value) error {
	return core.NewNotAssignableError(o.TypeName())
}

func (o *String) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewStringIterator(o.value)
}

func (o *String) IsString() bool {
	return true
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

func (o *String) AsChar() (rune, bool) {
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

func (o *String) fnTrim(vm core.VM, name string, args []core.Value) (core.Value, error) {
	if len(args) > 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}

	if len(args) == 0 {
		return vm.Allocator().NewStringValue(strings.Trim(string(o.value), " \t\n")), nil
	}

	s, ok := args[0].AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
	}

	return vm.Allocator().NewStringValue(strings.Trim(string(o.value), s)), nil
}
