package core

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/token"
)

// The minimum required fields for Value are ptr, d64 and kind. This allow to store primitive types such as int, float, rune; and heap allocated objects.
// Due to padding, the size of such structure will be 24 bytes on 64-bit architectures. So we can add some d32, d16 and d8 extra fields for free.
type Value struct {
	Type  uint8
	Const bool
	Data  uint64
	Ptr   unsafe.Pointer
}

func (v *Value) Set(val Value) {
	*v = val
}

func (v Value) EncodeJSON() ([]byte, error) {
	b, err := ValueTypes[v.Type].EncodeJSON(v)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed for type %s: %w", v.TypeName(), err)
	}
	return b, nil
}

func (v Value) EncodeBinary() ([]byte, error) {
	b, err := ValueTypes[v.Type].EncodeBinary(v)
	if err != nil {
		return nil, fmt.Errorf("binary encoding failed for type %s: %w", v.TypeName(), err)
	}
	i := byte(0)
	if v.Const {
		i = byte(1)
	}
	return append([]byte{v.Type, i}, b...), nil
}

func (v Value) GobEncode() ([]byte, error) {
	return v.EncodeBinary()
}

func (v *Value) DecodeBinary(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("binary decoding failed (type header): expected at least 2 bytes for type, got %d", len(data))
	}

	var t Value
	t.Type = data[0]
	t.Const = data[1] != 0
	if err := ValueTypes[t.Type].DecodeBinary(&t, data[2:]); err != nil {
		return fmt.Errorf("binary decoding failed for type %d: %w", t.Type, err)
	}
	*v = t

	return nil
}

func (v *Value) GobDecode(data []byte) error {
	return v.DecodeBinary(data)
}

func (v Value) Next() bool {
	return ValueTypes[v.Type].Next(v)
}

func (v Value) Key(a *Arena) (Value, error) {
	return ValueTypes[v.Type].Key(v, a)
}

func (v Value) Value(a *Arena) (Value, error) {
	return ValueTypes[v.Type].Value(v, a)
}

func (v Value) TypeName() string {
	return ValueTypes[v.Type].Name(v)
}

func (v Value) String() string {
	return ValueTypes[v.Type].String(v)
}

func (v Value) Interface() any {
	return ValueTypes[v.Type].Interface(v)
}

func (v Value) Arity() int8 {
	return ValueTypes[v.Type].Arity(v)
}

func (v Value) IsUserDefined() bool {
	return v.Type >= VT_USER_DEFINED
}

func (v Value) IsTrue() bool {
	return ValueTypes[v.Type].IsTrue(v)
}

func (v Value) IsIterable() bool {
	return ValueTypes[v.Type].IsIterable(v)
}

func (v Value) IsCallable() bool {
	return ValueTypes[v.Type].IsCallable(v)
}

func (v Value) IsVariadic() bool {
	return ValueTypes[v.Type].IsVariadic(v)
}

func (v Value) IsImmutable() bool {
	return v.Const
}

func (v Value) Contains(e Value) bool {
	return ValueTypes[v.Type].Contains(v, e)
}

func (v Value) AsBool() (bool, bool) {
	return ValueTypes[v.Type].AsBool(v)
}

func (v Value) AsRune() (rune, bool) {
	return ValueTypes[v.Type].AsRune(v)
}

func (v Value) AsByte() (byte, bool) {
	i, ok := ValueTypes[v.Type].AsInt(v)
	if !ok || i < 0 || i > 255 {
		return 0, false
	}
	return byte(i), true
}

func (v Value) AsInt() (int64, bool) {
	return ValueTypes[v.Type].AsInt(v)
}

func (v Value) AsFloat() (float64, bool) {
	return ValueTypes[v.Type].AsFloat(v)
}

func (v Value) AsDecimal() (dec128.Dec128, bool) {
	return ValueTypes[v.Type].AsDecimal(v)
}

func (v Value) AsTime() (time.Time, bool) {
	return ValueTypes[v.Type].AsTime(v)
}

func (v Value) AsString() (string, bool) {
	return ValueTypes[v.Type].AsString(v)
}

func (v Value) AsRunes() ([]rune, bool) {
	return ValueTypes[v.Type].AsRunes(v)
}

func (v Value) AsBytes() ([]byte, bool) {
	return ValueTypes[v.Type].AsBytes(v)
}

func (v Value) AsArray(a *Arena) ([]Value, bool) {
	return ValueTypes[v.Type].AsArray(v, a)
}

func (v Value) AsMap(a *Arena) (map[string]Value, bool) {
	return ValueTypes[v.Type].AsMap(v, a)
}

func (v Value) UnaryOp(a *Arena, op token.Token) (Value, error) {
	return ValueTypes[v.Type].UnaryOp(v, a, op)
}

func (v Value) BinaryOp(a *Arena, op token.Token, rhs Value) (Value, error) {
	return ValueTypes[v.Type].BinaryOp(v, a, op, rhs)
}

func (v Value) Equal(rhs Value) bool {
	return ValueTypes[v.Type].Equal(v, rhs)
}

func (v *Value) Copy(a *Arena) (Value, error) {
	return ValueTypes[v.Type].Copy(*v, a)
}

func (v Value) MethodCall(vm VM, name string, args []Value) (Value, error) {
	return ValueTypes[v.Type].MethodCall(v, vm, name, args)
}

func (v Value) Access(vm VM, index Value, mode Opcode) (Value, error) {
	return ValueTypes[v.Type].Access(v, vm.Allocator(), index, mode)
}

func (v Value) Assign(idx Value, val Value) error {
	return ValueTypes[v.Type].Assign(v, idx, val)
}

func (v Value) Iterator(a *Arena) (Value, error) {
	return ValueTypes[v.Type].Iterator(v, a)
}

func (v Value) Call(vm VM, args []Value) (Value, error) {
	return ValueTypes[v.Type].Call(v, vm, args)
}

func (v Value) Len() int64 {
	return ValueTypes[v.Type].Len(v)
}

func (v Value) Append(a *Arena, args []Value) (Value, error) {
	return ValueTypes[v.Type].Append(v, a, args)
}

func (v Value) Delete(key Value) (Value, error) {
	return ValueTypes[v.Type].Delete(v, key)
}

func (v Value) Slice(a *Arena, s Value, e Value) (Value, error) {
	return ValueTypes[v.Type].Slice(v, a, s, e)
}

func (v Value) Immutable(a *Arena) (Value, error) {
	t := v
	t.Const = true
	return t, nil
}
