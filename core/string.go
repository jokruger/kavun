package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/araddon/dateparse"
	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/internal/conv"
	"github.com/jokruger/gs/token"
)

type String struct {
	Value string
	runes []rune
}

func (o *String) Set(s string) {
	o.Value = s
	o.runes = nil
}

func (o *String) Runes() []rune {
	if o.runes == nil {
		o.runes = []rune(o.Value)
	}
	return o.runes
}

func (o *String) Len() int {
	if o.runes == nil {
		o.runes = []rune(o.Value)
	}
	return len(o.runes)
}

func (o *String) At(i int) rune {
	if o.runes == nil {
		o.runes = []rune(o.Value)
	}
	return o.runes[i]
}

// StringValue creates new boxed string value.
func StringValue(v *String) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_STRING,
	}
}

// NewStringValue creates new (heap-allocated) string value.
func NewStringValue(v string) Value {
	o := &String{}
	o.Set(v)
	return StringValue(o)
}

/* String type methods */

func stringTypeName(v Value) string {
	return "string"
}

func stringTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*String)(v.Ptr)
	var b []byte
	b = EncodeString(b, o.Value)
	return b, nil
}

func stringTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*String)(v.Ptr)
	s := o.Value
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
	o := &String{Value: s}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func stringTypeString(v Value) string {
	o := (*String)(v.Ptr)
	return strconv.Quote(o.Value)
}

func stringTypeInterface(v Value) any {
	o := (*String)(v.Ptr)
	return o.Value
}

func stringTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsString()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	o := (*String)(v.Ptr)
	switch op {
	case token.Add:
		return a.NewStringValue(o.Value + r)
	case token.Less:
		return BoolValue(o.Value < r), nil
	case token.LessEq:
		return BoolValue(o.Value <= r), nil
	case token.Greater:
		return BoolValue(o.Value > r), nil
	case token.GreaterEq:
		return BoolValue(o.Value >= r), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}

func stringTypeEqual(v Value, r Value) bool {
	t, ok := r.AsString()
	if !ok {
		return false
	}
	o := (*String)(v.Ptr)
	return o.Value == t
}

func stringTypeCopy(v Value, a Allocator) (Value, error) {
	o := (*String)(v.Ptr)
	return a.NewStringValue(o.Value)
}

func stringTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.to_string", "0", len(args))
		}
		return v, nil

	case "to_array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.to_array", "0", len(args))
		}
		a := vm.Allocator()
		t, _ := stringTypeAsArray(v, a)
		return a.NewArrayValue(t, false)

	case "to_bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.to_bool", "0", len(args))
		}
		b, _ := stringTypeAsBool(v)
		return BoolValue(b), nil

	case "to_bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.to_bytes", "0", len(args))
		}
		o := (*String)(v.Ptr)
		return vm.Allocator().NewBytesValue([]byte(o.Value))

	case "to_char":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.to_char", "0", len(args))
		}
		o := (*String)(v.Ptr)
		rs := o.Runes()
		if len(rs) == 1 {
			return CharValue(rs[0]), nil
		}
		return CharValue(0), nil

	case "to_float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.to_float", "0", len(args))
		}
		f, _ := stringTypeAsFloat(v)
		return FloatValue(f), nil

	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.to_int", "0", len(args))
		}
		i, _ := stringTypeAsInt(v)
		return IntValue(i), nil

	case "to_time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.to_time", "0", len(args))
		}
		t, _ := stringTypeAsTime(v)
		return vm.Allocator().NewTimeValue(t)

	case "to_record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.to_record", "0", len(args))
		}
		o := (*String)(v.Ptr)
		rs := o.Runes()
		m := make(map[string]Value, len(rs))
		for i, r := range rs {
			m[strconv.Itoa(i)] = CharValue(r)
		}
		return vm.Allocator().NewRecordValue(m, false)

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.is_empty", "0", len(args))
		}
		o := (*String)(v.Ptr)
		return BoolValue(len(o.Value) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.len", "0", len(args))
		}
		o := (*String)(v.Ptr)
		return IntValue(int64(o.Len())), nil

	case "first":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.first", "0", len(args))
		}
		o := (*String)(v.Ptr)
		if len(o.Value) == 0 {
			return Undefined, nil
		}
		return CharValue(o.At(0)), nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.last", "0", len(args))
		}
		o := (*String)(v.Ptr)
		if len(o.Value) == 0 {
			return Undefined, nil
		}
		return CharValue(o.At(o.Len() - 1)), nil

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.lower", "0", len(args))
		}
		o := (*String)(v.Ptr)
		return vm.Allocator().NewStringValue(strings.ToLower(o.Value))

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("string.upper", "0", len(args))
		}
		o := (*String)(v.Ptr)
		return vm.Allocator().NewStringValue(strings.ToUpper(o.Value))

	case "trim":
		return stringFnTrim(v, vm.Allocator(), "string.trim", args)

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError("string.contains", "1", len(args))
		}
		return BoolValue(stringTypeContains(v, args[0])), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func stringTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("string access", "int", index.TypeName())
		}
		o := (*String)(v.Ptr)
		rs := o.Runes()
		if i < 0 || i >= int64(len(rs)) {
			return Undefined, nil
		}
		return CharValue(rs[i]), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func stringTypeIsIterable(v Value) bool {
	return true
}

func stringTypeIterator(v Value, a Allocator) (Value, error) {
	o := (*String)(v.Ptr)
	return a.NewStringIteratorValue(o.Runes())
}

func stringTypeIsTrue(v Value) bool {
	o := (*String)(v.Ptr)
	return len(o.Value) > 0
}

func stringTypeAsString(v Value) (string, bool) {
	o := (*String)(v.Ptr)
	return o.Value, true
}

func stringTypeAsInt(v Value) (int64, bool) {
	o := (*String)(v.Ptr)
	i, err := strconv.ParseInt(o.Value, 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func stringTypeAsFloat(v Value) (float64, bool) {
	o := (*String)(v.Ptr)
	f, err := strconv.ParseFloat(o.Value, 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func stringTypeAsBool(v Value) (bool, bool) {
	o := (*String)(v.Ptr)
	return conv.ParseBool(o.Value)
}

func stringTypeAsChar(v Value) (rune, bool) {
	o := (*String)(v.Ptr)
	rs := o.Runes()
	if len(rs) == 1 {
		return rs[0], true
	}
	return 0, false
}

func stringTypeAsBytes(v Value) ([]byte, bool) {
	o := (*String)(v.Ptr)
	return []byte(o.Value), true
}

func stringTypeAsTime(v Value) (time.Time, bool) {
	o := (*String)(v.Ptr)
	val, err := dateparse.ParseAny(o.Value)
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

func stringTypeAsArray(v Value, a Allocator) ([]Value, bool) {
	o := (*String)(v.Ptr)
	rs := o.Runes()
	arr := make([]Value, len(rs))
	for i, r := range rs {
		arr[i] = CharValue(r)
	}
	return arr, true
}

func stringFnTrim(v Value, a Allocator, name string, args []Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}

	o := (*String)(v.Ptr)
	if len(args) == 0 {
		return a.NewStringValue(strings.Trim(o.Value, " \t\n"))
	}

	s, ok := args[0].AsString()
	if !ok {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
	}

	return a.NewStringValue(strings.Trim(o.Value, s))
}

func stringTypeContains(v Value, e Value) bool {
	o := (*String)(v.Ptr)
	switch e.Type {
	case VT_CHAR:
		c := ToChar(e)
		return strings.ContainsRune(o.Value, c)

	case VT_STRING:
		s := (*String)(e.Ptr)
		return strings.Contains(o.Value, s.Value)

	default:
		c, ok := e.AsChar()
		if !ok {
			return false
		}
		return slices.Contains(o.Runes(), c)
	}
}

func stringTypeLen(v Value) int64 {
	o := (*String)(v.Ptr)
	return int64(o.Len())
}

func stringTypeSlice(v Value, a Allocator, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	o := (*String)(v.Ptr)
	rs := o.Runes()
	l := int64(len(rs))

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

	return a.NewStringValue(string(rs[si:ei]))
}
