package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"

	"github.com/araddon/dateparse"
	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/internal/conv"
	"github.com/jokruger/gs/token"
)

type String struct {
	value []rune
}

func (o *String) Set(s string) {
	o.value = []rune(s)
}

func (o *String) Value() string {
	return string(o.value)
}

func (o *String) Len() int {
	return len(o.value)
}

func (o *String) Substring(start, end int) string {
	return string(o.value[start:end])
}

func StringValue(v *String) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_STRING,
	}
}

func NewStringValue(v string) Value {
	o := &String{}
	o.Set(v)
	return StringValue(o)
}

func stringTypeName(v Value) string {
	return "string"
}

func stringTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*String)(v.Ptr)
	var b []byte
	b = EncodeString(b, o.Value())
	return b, nil
}

func stringTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*String)(v.Ptr)
	s := string(o.value)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return nil, fmt.Errorf("string: %w", err)
	}
	return buf.Bytes(), nil
}

func stringTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("string: %w", err)
	}
	o := &String{value: []rune(s)}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func stringTypeString(v Value) string {
	o := (*String)(v.Ptr)
	return strconv.Quote(string(o.value))
}

func stringTypeInterface(v Value) any {
	o := (*String)(v.Ptr)
	return string(o.value)
}

func stringTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsString()
	if !ok {
		return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	o := (*String)(v.Ptr)
	switch op {
	case token.Add:
		return a.NewStringValue(string(o.value) + r), nil
	case token.Less:
		return BoolValue(string(o.value) < r), nil
	case token.LessEq:
		return BoolValue(string(o.value) <= r), nil
	case token.Greater:
		return BoolValue(string(o.value) > r), nil
	case token.GreaterEq:
		return BoolValue(string(o.value) >= r), nil
	}

	return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}

func stringTypeEqual(v Value, r Value) bool {
	t, ok := r.AsString()
	if !ok {
		return false
	}
	o := (*String)(v.Ptr)
	return string(o.value) == t
}

func stringTypeCopy(v Value, a Allocator) Value {
	o := (*String)(v.Ptr)
	return a.NewStringValue(string(o.value))
}

func stringTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_string":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.to_string", "0", len(args))
		}
		return v, nil

	case "to_array":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.to_array", "0", len(args))
		}
		o := (*String)(v.Ptr)
		arr := make([]Value, len(o.value))
		for i, r := range o.value {
			arr[i] = CharValue(r)
		}
		return vm.Allocator().NewArrayValue(arr, false), nil

	case "to_bool":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.to_bool", "0", len(args))
		}
		b, _ := stringTypeAsBool(v)
		return BoolValue(b), nil

	case "to_bytes":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.to_bytes", "0", len(args))
		}
		o := (*String)(v.Ptr)
		return vm.Allocator().NewBytesValue([]byte(string(o.value))), nil

	case "to_char":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.to_char", "0", len(args))
		}
		o := (*String)(v.Ptr)
		if len(o.value) == 1 {
			return CharValue(o.value[0]), nil
		}
		return CharValue(0), nil

	case "to_float":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.to_float", "0", len(args))
		}
		f, _ := stringTypeAsFloat(v)
		return FloatValue(f), nil

	case "to_int":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.to_int", "0", len(args))
		}
		i, _ := stringTypeAsInt(v)
		return IntValue(i), nil

	case "to_time":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.to_time", "0", len(args))
		}
		t, _ := stringTypeAsTime(v)
		return vm.Allocator().NewTimeValue(t), nil

	case "to_record":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.to_record", "0", len(args))
		}
		o := (*String)(v.Ptr)
		m := make(map[string]Value, len(o.value))
		for i, r := range o.value {
			m[strconv.Itoa(i)] = CharValue(r)
		}
		return vm.Allocator().NewRecordValue(m, false), nil

	case "is_empty":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.is_empty", "0", len(args))
		}
		o := (*String)(v.Ptr)
		return BoolValue(len(o.value) == 0), nil

	case "len":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.len", "0", len(args))
		}
		o := (*String)(v.Ptr)
		return IntValue(int64(len(o.value))), nil

	case "first":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.first", "0", len(args))
		}
		o := (*String)(v.Ptr)
		if len(o.value) == 0 {
			return UndefinedValue(), nil
		}
		return CharValue(o.value[0]), nil

	case "last":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.last", "0", len(args))
		}
		o := (*String)(v.Ptr)
		if len(o.value) == 0 {
			return UndefinedValue(), nil
		}
		return CharValue(o.value[len(o.value)-1]), nil

	case "lower":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.lower", "0", len(args))
		}
		o := (*String)(v.Ptr)
		t := make([]rune, len(o.value))
		for i, r := range o.value {
			t[i] = unicode.ToLower(r)
		}
		return vm.Allocator().NewStringValue(string(t)), nil

	case "upper":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("string.upper", "0", len(args))
		}
		o := (*String)(v.Ptr)
		t := make([]rune, len(o.value))
		for i, r := range o.value {
			t[i] = unicode.ToUpper(r)
		}
		return vm.Allocator().NewStringValue(string(t)), nil

	case "trim":
		return stringFnTrim(v, vm.Allocator(), "string.trim", args)

	default:
		return UndefinedValue(), errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func stringTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return UndefinedValue(), errs.NewInvalidIndexTypeError("string access", "int", index.TypeName())
		}
		o := (*String)(v.Ptr)
		if i < 0 || i >= int64(len(o.value)) {
			return UndefinedValue(), nil
		}
		return CharValue(o.value[i]), nil
	}

	k, ok := index.AsString()
	if !ok {
		return UndefinedValue(), errs.NewInvalidIndexTypeError("string selector access", "string", index.TypeName())
	}

	return UndefinedValue(), errs.NewInvalidSelectorError(v.TypeName(), k)
}

func stringTypeIsIterable(v Value) bool {
	return true
}

func stringTypeIterator(v Value, a Allocator) Value {
	o := (*String)(v.Ptr)
	return a.NewStringIteratorValue(o.value)
}

func stringTypeIsTrue(v Value) bool {
	o := (*String)(v.Ptr)
	return len(o.value) > 0
}

func stringTypeAsString(v Value) (string, bool) {
	o := (*String)(v.Ptr)
	return string(o.value), true
}

func stringTypeAsInt(v Value) (int64, bool) {
	o := (*String)(v.Ptr)
	i, err := strconv.ParseInt(string(o.value), 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func stringTypeAsFloat(v Value) (float64, bool) {
	o := (*String)(v.Ptr)
	f, err := strconv.ParseFloat(string(o.value), 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func stringTypeAsBool(v Value) (bool, bool) {
	o := (*String)(v.Ptr)
	return conv.ParseBool(string(o.value))
}

func stringTypeAsChar(v Value) (rune, bool) {
	o := (*String)(v.Ptr)
	if len(o.value) == 1 {
		return o.value[0], true
	}
	return 0, false
}

func stringTypeAsBytes(v Value) ([]byte, bool) {
	o := (*String)(v.Ptr)
	return []byte(string(o.value)), true
}

func stringTypeAsTime(v Value) (time.Time, bool) {
	o := (*String)(v.Ptr)
	val, err := dateparse.ParseAny(string(o.value))
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

func stringFnTrim(v Value, a Allocator, name string, args []Value) (Value, error) {
	if len(args) > 1 {
		return UndefinedValue(), errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}

	o := (*String)(v.Ptr)
	if len(args) == 0 {
		return a.NewStringValue(strings.Trim(string(o.value), " \t\n")), nil
	}

	s, ok := args[0].AsString()
	if !ok {
		return UndefinedValue(), errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
	}

	return a.NewStringValue(strings.Trim(string(o.value), s)), nil
}
