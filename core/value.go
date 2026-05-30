package core

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

const anyTypeName = "value"

// Value represents a boxed Kavun value.
type Value struct {
	Type      uint8
	Immutable bool
	Data      uint64
	Ptr       unsafe.Pointer
}

// RefValue is a dummy constructor used in internal generics.
func RefValue(v Value) Value {
	return v
}

func (v *Value) Set(val Value) {
	*v = val
}

func (v Value) EncodeJSON(a *Arena) ([]byte, error) {
	b, err := ValueTypes[v.Type].EncodeJSON(a, v)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed for type %s: %w", v.TypeName(a), err)
	}
	return b, nil
}

func (v Value) EncodeBinary(a *Arena) ([]byte, error) {
	b, err := ValueTypes[v.Type].EncodeBinary(a, v)
	if err != nil {
		return nil, fmt.Errorf("binary encoding failed for type %s: %w", v.TypeName(a), err)
	}
	i := byte(0)
	if v.Immutable {
		i = byte(1)
	}
	return append([]byte{v.Type, i}, b...), nil
}

func (v *Value) DecodeBinary(a *Arena, data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("binary decoding failed (type header): expected at least 2 bytes for type, got %d", len(data))
	}

	var t Value
	t.Type = data[0]
	t.Immutable = data[1] != 0
	if err := ValueTypes[t.Type].DecodeBinary(a, &t, data[2:]); err != nil {
		return fmt.Errorf("binary decoding failed for type %d: %w", t.Type, err)
	}
	*v = t

	return nil
}

func (v Value) Next(a *Arena) bool {
	return ValueTypes[v.Type].Next(a, v)
}

func (v Value) Key(a *Arena) (Value, error) {
	return ValueTypes[v.Type].Key(a, v)
}

func (v Value) Value(a *Arena) (Value, error) {
	return ValueTypes[v.Type].Value(a, v)
}

func (v Value) TypeName(a *Arena) string {
	return ValueTypes[v.Type].Name(a, v)
}

func (v Value) Format(a *Arena, sp fspec.FormatSpec) (string, error) {
	return ValueTypes[v.Type].Format(a, v, sp)
}

func (v Value) String(a *Arena) string {
	return ValueTypes[v.Type].String(a, v)
}

func (v Value) Interface(a *Arena) any {
	return ValueTypes[v.Type].Interface(a, v)
}

func (v Value) Arity(a *Arena) int8 {
	return ValueTypes[v.Type].Arity(a, v)
}

func (v Value) IsUserDefined() bool {
	return v.Type >= VT_USER_DEFINED
}

func (v Value) IsTrue(a *Arena) bool {
	return ValueTypes[v.Type].IsTrue(a, v)
}

func (v Value) IsIterable(a *Arena) bool {
	return ValueTypes[v.Type].IsIterable(a, v)
}

func (v Value) IsCallable(a *Arena) bool {
	return ValueTypes[v.Type].IsCallable(a, v)
}

func (v Value) IsVariadic(a *Arena) bool {
	return ValueTypes[v.Type].IsVariadic(a, v)
}

func (v Value) Contains(a *Arena, e Value) bool {
	return ValueTypes[v.Type].Contains(a, v, e)
}

func (v Value) AsValue() (Value, bool) {
	return v, true
}

func (v Value) AsBool(a *Arena) (bool, bool) {
	return ValueTypes[v.Type].AsBool(a, v)
}

func (v Value) AsRune(a *Arena) (rune, bool) {
	return ValueTypes[v.Type].AsRune(a, v)
}

func (v Value) AsByte(a *Arena) (byte, bool) {
	return ValueTypes[v.Type].AsByte(a, v)
}

func (v Value) AsInt(a *Arena) (int64, bool) {
	return ValueTypes[v.Type].AsInt(a, v)
}

func (v Value) AsFloat(a *Arena) (float64, bool) {
	return ValueTypes[v.Type].AsFloat(a, v)
}

func (v Value) AsDecimal(a *Arena) (dec128.Dec128, bool) {
	return ValueTypes[v.Type].AsDecimal(a, v)
}

func (v Value) AsTime(a *Arena) (time.Time, bool) {
	return ValueTypes[v.Type].AsTime(a, v)
}

func (v Value) AsString(a *Arena) (string, bool) {
	return ValueTypes[v.Type].AsString(a, v)
}

func (v Value) AsRunes(a *Arena) ([]rune, bool) {
	return ValueTypes[v.Type].AsRunes(a, v)
}

func (v Value) AsBytes(a *Arena) ([]byte, bool) {
	return ValueTypes[v.Type].AsBytes(a, v)
}

func (v Value) AsArray(a *Arena) ([]Value, bool) {
	return ValueTypes[v.Type].AsArray(a, v)
}

func (v Value) AsDict(a *Arena) (map[string]Value, bool) {
	return ValueTypes[v.Type].AsDict(a, v)
}

func (v Value) UnaryOp(a *Arena, op token.Token) (Value, error) {
	return ValueTypes[v.Type].UnaryOp(a, v, op)
}

func (v Value) BinaryOp(a *Arena, op token.Token, rhs Value) (Value, error) {
	return ValueTypes[v.Type].BinaryOp(a, v, rhs, op)
}

func (v Value) Equal(a *Arena, rhs Value) bool {
	return ValueTypes[v.Type].Equal(a, v, rhs)
}

func (v *Value) Clone(a *Arena) (Value, error) {
	return ValueTypes[v.Type].Clone(a, *v)
}

func (v Value) MethodCall(a *Arena, vm VM, name string, args []Value) (Value, error) {
	return ValueTypes[v.Type].MethodCall(a, vm, v, name, args)
}

func (v Value) Access(a *Arena, index Value, mode bc.Opcode) (Value, error) {
	return ValueTypes[v.Type].Access(a, v, index, mode)
}

func (v Value) Assign(a *Arena, idx Value, val Value) error {
	return ValueTypes[v.Type].Assign(a, v, idx, val)
}

func (v Value) Iterator(a *Arena) (Value, error) {
	return ValueTypes[v.Type].Iterator(a, v)
}

func (v Value) Call(a *Arena, vm VM, args []Value) (Value, error) {
	return ValueTypes[v.Type].Call(a, vm, v, args)
}

func (v Value) Len(a *Arena) int64 {
	return ValueTypes[v.Type].Len(a, v)
}

func (v Value) Append(a *Arena, args []Value) (Value, error) {
	return ValueTypes[v.Type].Append(a, v, args)
}

func (v Value) Delete(a *Arena, key Value) (Value, error) {
	return ValueTypes[v.Type].Delete(a, v, key)
}

func (v Value) Slice(a *Arena, s Value, e Value) (Value, error) {
	return ValueTypes[v.Type].Slice(a, v, s, e)
}

func (v Value) SliceStep(a *Arena, s Value, e Value, step Value) (Value, error) {
	return ValueTypes[v.Type].SliceStep(a, v, s, e, step)
}

func (v Value) ToImmutable(a *Arena) (Value, error) {
	t := v
	t.Immutable = true
	return t, nil
}
