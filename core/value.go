package core

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/jokruger/gs/token"
)

// The minimum required fields for Value are ptr, d64 and kind. This allow to store primitive types such as int, float, rune; and heap allocated objects.
// Due to padding, the size of such structure will be 24 bytes on 64-bit architectures. So we can add some d32, d16 and d8 extra fields for free.
type Value struct {
	Type uint8
	Data uint64
	Ptr  unsafe.Pointer
}

func (v *Value) Set(val Value) {
	*v = val
}

func (v Value) EncodeJSON() ([]byte, error) {
	b, err := TypeEncodeJSON[v.Type](v)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed for type %s: %w", v.TypeName(), err)
	}
	return b, nil
}

func (v Value) EncodeBinary() ([]byte, error) {
	b, err := TypeEncodeBinary[v.Type](v)
	if err != nil {
		return nil, fmt.Errorf("binary encoding failed for type %s: %w", v.TypeName(), err)
	}
	return append([]byte{v.Type}, b...), nil
}

func (v Value) GobEncode() ([]byte, error) {
	return v.EncodeBinary()
}

func (v *Value) DecodeBinary(data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("binary decoding failed (type header): expected at least 1 byte for type, got %d", len(data))
	}

	var t Value
	t.Type = data[0]
	if err := TypeDecodeBinary[t.Type](&t, data[1:]); err != nil {
		return fmt.Errorf("binary decoding failed for type %d: %w", t.Type, err)
	}
	*v = t

	return nil
}

func (v *Value) GobDecode(data []byte) error {
	return v.DecodeBinary(data)
}

func (v *Value) Next() bool {
	return TypeNext[v.Type](v)
}

func (v Value) Key(alloc Allocator) Value {
	return TypeKey[v.Type](v, alloc)
}

func (v Value) Value(alloc Allocator) Value {
	return TypeValue[v.Type](v, alloc)
}

func (v Value) TypeName() string {
	return TypeName[v.Type](v)
}

func (v Value) String() string {
	return TypeString[v.Type](v)
}

func (v Value) Interface() any {
	return TypeInterface[v.Type](v)
}

func (v Value) Arity() int {
	return TypeArity[v.Type](v)
}

func (v Value) IsUndefined() bool {
	return v.Type == VT_UNDEFINED
}

func (v Value) IsValuePtr() bool {
	return v.Type == VT_VALUE_PTR
}

func (v Value) IsBuiltinFunction() bool {
	return v.Type == VT_BUILTIN_FUNCTION
}

func (v Value) IsCompiledFunction() bool {
	return v.Type == VT_COMPILED_FUNCTION
}

func (v Value) IsError() bool {
	return v.Type == VT_ERROR
}

func (v Value) IsBool() bool {
	return v.Type == VT_BOOL
}

func (v Value) IsChar() bool {
	return v.Type == VT_CHAR
}

func (v Value) IsInt() bool {
	return v.Type == VT_INT
}

func (v Value) IsFloat() bool {
	return v.Type == VT_FLOAT
}

func (v Value) IsTime() bool {
	return v.Type == VT_TIME
}

func (v Value) IsString() bool {
	return v.Type == VT_STRING
}

func (v Value) IsBytes() bool {
	return v.Type == VT_BYTES
}

func (v Value) IsArray() bool {
	return v.Type == VT_ARRAY
}

func (v Value) IsRecord() bool {
	return v.Type == VT_RECORD
}

func (v Value) IsMap() bool {
	return v.Type == VT_MAP
}

func (v Value) IsStringIterator() bool {
	return v.Type == VT_STRING_ITERATOR
}

func (v Value) IsBytesIterator() bool {
	return v.Type == VT_BYTES_ITERATOR
}

func (v Value) IsArrayIterator() bool {
	return v.Type == VT_ARRAY_ITERATOR
}

func (v Value) IsMapIterator() bool {
	return v.Type == VT_MAP_ITERATOR
}

func (v Value) IsIntRange() bool {
	return v.Type == VT_INT_RANGE
}

func (v Value) IsIntRangeIterator() bool {
	return v.Type == VT_INT_RANGE_ITERATOR
}

func (v Value) IsUserDefined() bool {
	return v.Type >= VT_USER_DEFINED
}

func (v Value) IsTrue() bool {
	return TypeIsTrue[v.Type](v)
}

func (v Value) IsIterable() bool {
	return TypeIsIterable[v.Type](v)
}

func (v Value) IsCallable() bool {
	return TypeIsCallable[v.Type](v)
}

func (v Value) IsVariadic() bool {
	return TypeIsVariadic[v.Type](v)
}

func (v Value) IsImmutable() bool {
	return TypeIsImmutable[v.Type](v)
}

func (v Value) Contains(e Value) bool {
	return TypeContains[v.Type](v, e)
}

func (v Value) AsString() (string, bool) {
	return TypeAsString[v.Type](v)
}

func (v Value) AsInt() (int64, bool) {
	return TypeAsInt[v.Type](v)
}

func (v Value) AsFloat() (float64, bool) {
	return TypeAsFloat[v.Type](v)
}

func (v Value) AsBool() (bool, bool) {
	return TypeAsBool[v.Type](v)
}

func (v Value) AsChar() (rune, bool) {
	return TypeAsChar[v.Type](v)
}

func (v Value) AsBytes() ([]byte, bool) {
	return TypeAsBytes[v.Type](v)
}

func (v Value) AsTime() (time.Time, bool) {
	return TypeAsTime[v.Type](v)
}

func (v Value) BinaryOp(a Allocator, op token.Token, rhs Value) (Value, error) {
	return TypeBinaryOp[v.Type](v, a, op, rhs)
}

func (v Value) Equal(rhs Value) bool {
	return TypeEqual[v.Type](v, rhs)
}

func (v *Value) Copy(alloc Allocator) Value {
	return TypeCopy[v.Type](*v, alloc)
}

func (v Value) MethodCall(vm VM, name string, args []Value) (Value, error) {
	return TypeMethodCall[v.Type](v, vm, name, args)
}

func (v Value) Access(vm VM, index Value, mode Opcode) (Value, error) {
	return TypeAccess[v.Type](v, vm.Allocator(), index, mode)
}

func (v Value) Assign(idx Value, val Value) error {
	return TypeAssign[v.Type](v, idx, val)
}

func (v Value) Iterator(alloc Allocator) Value {
	return TypeIterator[v.Type](v, alloc)
}

func (v Value) Call(vm VM, args []Value) (Value, error) {
	return TypeCall[v.Type](v, vm, args)
}
