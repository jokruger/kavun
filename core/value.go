package core

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/fspec"
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

// PURE by contract
func (v Value) EncodeJSON() ([]byte, error) {
	b, err := ValueTypes[v.Type].EncodeJSON(v)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed for type %s: %w", v.TypeName(), err)
	}
	return b, nil
}

// PURE by contract
func (v Value) EncodeBinary() ([]byte, error) {
	i := byte(0)
	if v.Immutable {
		i = byte(1)
	}
	b, err := ValueTypes[v.Type].EncodeBinary(v)
	if err != nil {
		return nil, fmt.Errorf("binary encoding failed for type %s: %w", v.TypeName(), err)
	}
	return append([]byte{v.Type, i}, b...), nil
}

// IMPURE by contract (mutates target)
func (v *Value) DecodeBinary(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("binary decoding failed (type header): expected at least 2 bytes for type, got %d", len(data))
	}
	var t Value
	t.Type = data[0]
	t.Immutable = data[1] != 0
	if err := ValueTypes[t.Type].DecodeBinary(&t, data[2:]); err != nil {
		return fmt.Errorf("binary decoding failed for type %d: %w", t.Type, err)
	}
	*v = t
	return nil
}

// GobEncode wraps binary encoding so gob does not reflect over unsafe.Pointer field.
func (v Value) GobEncode() ([]byte, error) {
	return v.EncodeBinary()
}

// GobDecode wraps binary decoding to mirror GobEncode.
func (v *Value) GobDecode(data []byte) error {
	return v.DecodeBinary(data)
}

// LOCALISED-STATE by contract (advances iterator cursor)
func (v Value) Next() bool {
	return ValueTypes[v.Type].Next(v)
}

// LOCALISED-STATE by contract (reads iterator cursor)
func (v Value) Key() (Value, error) {
	return ValueTypes[v.Type].Key(v)
}

// LOCALISED-STATE by contract (reads iterator cursor)
func (v Value) Value() (Value, error) {
	return ValueTypes[v.Type].Value(v)
}

// PURE by contract
func (v Value) TypeName() string {
	return ValueTypes[v.Type].Name(v)
}

// PURE by contract
func (v Value) Format(sp fspec.FormatSpec) (string, error) {
	return ValueTypes[v.Type].Format(v, sp)
}

// PURE by contract
func (v Value) String() string {
	return ValueTypes[v.Type].String(v)
}

// PURE by contract
func (v Value) Interface() any {
	return ValueTypes[v.Type].Interface(v)
}

// PURE by contract
func (v Value) Arity() int {
	return ValueTypes[v.Type].Arity(v)
}

// PURE by contract
func (v Value) IsPrimitive() bool {
	return v.Type <= value.LastPrimitiveType
}

// PURE by contract
func (v Value) IsUserDefined() bool {
	return v.Type >= value.FirstUserDefinedType
}

// PURE by contract
func (v Value) IsTrue() bool {
	return ValueTypes[v.Type].IsTrue(v)
}

// PURE by contract
func (v Value) IsIterable() bool {
	return ValueTypes[v.Type].IsIterable(v)
}

// PURE by contract
func (v Value) IsCallable() bool {
	return ValueTypes[v.Type].IsCallable(v)
}

// PURE by contract
func (v Value) IsVariadic() bool {
	return ValueTypes[v.Type].IsVariadic(v)
}

// PURE by contract
func (v Value) Contains(e Value) bool {
	return ValueTypes[v.Type].Contains(v, e)
}

// PURE by contract
func (v Value) AsValue() (Value, bool) {
	return v, true
}

// PURE by contract
func (v Value) AsBool() (bool, bool) {
	return ValueTypes[v.Type].AsBool(v)
}

// PURE by contract
func (v Value) AsRune() (rune, bool) {
	return ValueTypes[v.Type].AsRune(v)
}

// PURE by contract
func (v Value) AsByte() (byte, bool) {
	return ValueTypes[v.Type].AsByte(v)
}

// PURE by contract
func (v Value) AsInt() (int64, bool) {
	return ValueTypes[v.Type].AsInt(v)
}

// PURE by contract
func (v Value) AsFloat() (float64, bool) {
	return ValueTypes[v.Type].AsFloat(v)
}

// PURE by contract
func (v Value) AsDecimal() (dec128.Dec128, bool) {
	return ValueTypes[v.Type].AsDecimal(v)
}

// PURE by contract
func (v Value) AsTime() (time.Time, bool) {
	return ValueTypes[v.Type].AsTime(v)
}

// PURE by contract
func (v Value) AsString() (string, bool) {
	return ValueTypes[v.Type].AsString(v)
}

// PURE by contract
func (v Value) AsRunes() ([]rune, bool) {
	return ValueTypes[v.Type].AsRunes(v)
}

// PURE by contract
func (v Value) AsBytes() ([]byte, bool) {
	return ValueTypes[v.Type].AsBytes(v)
}

// PURE by contract
func (v Value) AsArray() ([]Value, bool) {
	return ValueTypes[v.Type].AsArray(v)
}

// PURE by contract
func (v Value) AsDict() (map[string]Value, bool) {
	return ValueTypes[v.Type].AsDict(v)
}

// PURE by contract
func (v Value) UnaryOp(op token.Token) (Value, error) {
	return ValueTypes[v.Type].UnaryOp(v, op)
}

// PURE by contract
func (v Value) BinaryOp(op token.Token, rhs Value) (Value, error) {
	return ValueTypes[v.Type].BinaryOp(v, rhs, op)
}

// PURE by contract
func (v Value) Equal(rhs Value) bool {
	return ValueTypes[v.Type].Equal(v, rhs)
}

// PURE by contract
func (v *Value) Clone() (Value, error) {
	return ValueTypes[v.Type].Clone(*v)
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func (v Value) MethodCall(vm VM, name string, args []Value) (Value, error) {
	return ValueTypes[v.Type].MethodCall(vm, v, name, args)
}

// PURE by contract
func (v Value) Access(index Value, mode bc.Opcode) (Value, error) {
	return ValueTypes[v.Type].Access(v, index, mode)
}

// IMPURE by contract (mutates target)
func (v Value) Assign(idx Value, val Value) error {
	return ValueTypes[v.Type].Assign(v, idx, val)
}

// PURE by contract (constructs new iterator)
func (v Value) Iterator() (Value, error) {
	return ValueTypes[v.Type].Iterator(v)
}

// CALLABLE-DEPENDENT by contract
func (v Value) Call(vm VM, args []Value) (Value, error) {
	return ValueTypes[v.Type].Call(vm, v, args)
}

// PURE by contract
func (v Value) Len() int64 {
	return ValueTypes[v.Type].Len(v)
}

// GO-STYLE by contract (may share receiver storage)
func (v Value) Append(args []Value) (Value, error) {
	return ValueTypes[v.Type].Append(v, args)
}

// IMPURE by contract (mutates target)
func (v Value) Delete(key Value) (Value, error) {
	return ValueTypes[v.Type].Delete(v, key)
}

// PURE by contract
func (v Value) Slice(s Value, e Value) (Value, error) {
	return ValueTypes[v.Type].Slice(v, s, e)
}

// PURE by contract
func (v Value) SliceStep(s Value, e Value, step Value) (Value, error) {
	return ValueTypes[v.Type].SliceStep(v, s, e, step)
}

// PURE by contract
func (v Value) ToImmutable() (Value, error) {
	t := v
	t.Immutable = true
	return t, nil
}
