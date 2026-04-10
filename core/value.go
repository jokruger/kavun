package core

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/jokruger/gs/errs"
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

func (v Value) ValuePtr() *Value {
	return toValuePtr(v)
}

func (v Value) BuiltinFunction() *BuiltinFunction {
	return toBuiltinFunction(v)
}

func (v Value) CompiledFunction() *CompiledFunction {
	return toCompiledFunction(v)
}

func (v Value) Int() int64 {
	return toInt(v)
}

func (v Value) Float() float64 {
	return toFloat(v)
}

func (v Value) Char() rune {
	return toChar(v)
}

func (v Value) Bool() bool {
	return toBool(v)
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
	switch v.Type {
	case VT_STRING_ITERATOR:
		i := (*StringIterator)(v.Ptr)
		i.i++
		return i.i <= i.l

	case VT_BYTES_ITERATOR:
		i := (*BytesIterator)(v.Ptr)
		i.i++
		return i.i <= i.l

	case VT_ARRAY_ITERATOR:
		i := (*ArrayIterator)(v.Ptr)
		i.i++
		return i.i <= i.l

	case VT_MAP_ITERATOR:
		i := (*MapIterator)(v.Ptr)
		i.i++
		return i.i <= i.l

	default:
		return TypeNext[v.Type](v)
	}
}

func (v Value) Key(alloc Allocator) Value {
	switch v.Type {
	case VT_STRING_ITERATOR:
		i := (*StringIterator)(v.Ptr)
		return IntValue(int64(i.i - 1))

	case VT_BYTES_ITERATOR:
		i := (*BytesIterator)(v.Ptr)
		return IntValue(int64(i.i - 1))

	case VT_ARRAY_ITERATOR:
		i := (*ArrayIterator)(v.Ptr)
		return IntValue(int64(i.i - 1))

	case VT_MAP_ITERATOR:
		i := (*MapIterator)(v.Ptr)
		return alloc.NewStringValue(i.k[i.i-1])

	default:
		return TypeKey[v.Type](v, alloc)
	}
}

func (v Value) Value(alloc Allocator) Value {
	switch v.Type {
	case VT_STRING_ITERATOR:
		i := (*StringIterator)(v.Ptr)
		return CharValue(i.v[i.i-1])

	case VT_BYTES_ITERATOR:
		i := (*BytesIterator)(v.Ptr)
		return IntValue(int64(i.v[i.i-1]))

	case VT_ARRAY_ITERATOR:
		i := (*ArrayIterator)(v.Ptr)
		return i.v[i.i-1]

	case VT_MAP_ITERATOR:
		i := (*MapIterator)(v.Ptr)
		k := i.k[i.i-1]
		return i.v[k]

	default:
		return TypeValue[v.Type](v, alloc)
	}
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

func (v Value) Arity() uint8 {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_BUILTIN_FUNCTION:
		return builtinFunctionTypeArity(v)
	case VT_COMPILED_FUNCTION:
		return compiledFunctionTypeArity(v)
	default:
		return TypeArity[v.Type](v)
	}
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

func (v Value) IsUserDefined() bool {
	return v.Type >= VT_USER_DEFINED
}

func (v Value) IsTrue() bool {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_UNDEFINED:
		return false
	case VT_ERROR:
		return errorTypeIsTrue(v)
	case VT_BOOL:
		return boolTypeIsTrue(v)
	default:
		return TypeIsTrue[v.Type](v)
	}
}

func (v Value) IsIterable() bool {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_STRING:
		return stringTypeIsIterable(v)
	case VT_BYTES:
		return bytesTypeIsIterable(v)
	case VT_ARRAY:
		return arrayTypeIsIterable(v)
	case VT_RECORD:
		return recordTypeIsIterable(v)
	case VT_MAP:
		return mapTypeIsIterable(v)
	default:
		return TypeIsIterable[v.Type](v)
	}
}

func (v Value) IsCallable() bool {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_BUILTIN_FUNCTION:
		return builtinFunctionTypeIsCallable(v)
	case VT_COMPILED_FUNCTION:
		return compiledFunctionTypeIsCallable(v)
	default:
		return TypeIsCallable[v.Type](v)
	}
}

func (v Value) IsVariadic() bool {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_BUILTIN_FUNCTION:
		return builtinFunctionTypeIsVariadic(v)
	case VT_COMPILED_FUNCTION:
		return compiledFunctionTypeIsVariadic(v)
	default:
		return TypeIsVariadic[v.Type](v)
	}
}

func (v Value) IsImmutable() bool {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_ARRAY:
		return arrayTypeIsImmutable(v)
	case VT_RECORD:
		return recordTypeIsImmutable(v)
	case VT_MAP:
		return mapTypeIsImmutable(v)
	default:
		return TypeIsImmutable[v.Type](v)
	}
}

func (v Value) AsString() (string, bool) {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_ERROR:
		return errorTypeAsString(v)
	case VT_STRING:
		return stringTypeAsString(v)
	case VT_BYTES:
		return bytesTypeAsString(v)
	default:
		return TypeAsString[v.Type](v)
	}
}

func (v Value) AsInt() (int64, bool) {
	switch v.Type {
	case VT_CHAR:
		return int64(toChar(v)), true

	case VT_INT:
		return toInt(v), true

	case VT_FLOAT:
		return int64(toFloat(v)), true

	default:
		return TypeAsInt[v.Type](v)
	}
}

func (v Value) AsFloat() (float64, bool) {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_INT:
		return intTypeAsFloat(v)
	case VT_FLOAT:
		return floatTypeAsFloat(v)
	default:
		return TypeAsFloat[v.Type](v)
	}
}

func (v Value) AsBool() (bool, bool) {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_UNDEFINED:
		return undefinedTypeAsBool(v)
	case VT_ERROR:
		return errorTypeAsBool(v)
	case VT_BOOL:
		return boolTypeAsBool(v)
	default:
		return TypeAsBool[v.Type](v)
	}
}

func (v Value) AsChar() (rune, bool) {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_CHAR:
		return charTypeAsChar(v)
	case VT_INT:
		return intTypeAsChar(v)
	default:
		return TypeAsChar[v.Type](v)
	}
}

func (v Value) AsBytes() ([]byte, bool) {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_STRING:
		return stringTypeAsBytes(v)
	case VT_BYTES:
		return bytesTypeAsBytes(v)
	default:
		return TypeAsBytes[v.Type](v)
	}
}

func (v Value) AsTime() (time.Time, bool) {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_INT:
		return intTypeAsTime(v)
	case VT_TIME:
		return timeTypeAsTime(v)
	case VT_STRING:
		return stringTypeAsTime(v)
	default:
		return TypeAsTime[v.Type](v)
	}
}

func (v Value) BinaryOp(a Allocator, op token.Token, rhs Value) (Value, error) {
	switch v.Type {
	case VT_CHAR:
		switch rhs.Type {
		case VT_INT: // char op int => int
			l := int64(toChar(v))
			r := toInt(rhs)
			switch op {
			case token.Add:
				return IntValue(l + r), nil
			case token.Sub:
				return IntValue(l - r), nil
			case token.Less:
				return BoolValue(l < r), nil
			case token.Greater:
				return BoolValue(l > r), nil
			case token.LessEq:
				return BoolValue(l <= r), nil
			case token.GreaterEq:
				return BoolValue(l >= r), nil
			default:
				return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
			}

		case VT_STRING: // char op string => string
			l := string(toChar(v))
			r, _ := stringTypeAsString(rhs)
			switch op {
			case token.Add:
				return a.NewStringValue(l + r), nil
			default:
				return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
			}

		default:
			// char op any => char
			r, ok := rhs.AsChar()
			if !ok {
				return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
			}

			l := toChar(v)
			switch op {
			case token.Add:
				return CharValue(l + r), nil
			case token.Sub:
				return CharValue(l - r), nil
			case token.Less:
				return BoolValue(l < r), nil
			case token.Greater:
				return BoolValue(l > r), nil
			case token.LessEq:
				return BoolValue(l <= r), nil
			case token.GreaterEq:
				return BoolValue(l >= r), nil
			default:
				return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
			}
		}

	case VT_INT:
		switch rhs.Type {
		case VT_INT: // int op int => int
			l := toInt(v)
			r := toInt(rhs)
			switch op {
			case token.Add:
				return IntValue(l + r), nil
			case token.Sub:
				return IntValue(l - r), nil
			case token.Mul:
				return IntValue(l * r), nil
			case token.Quo:
				return IntValue(l / r), nil
			case token.Rem:
				return IntValue(l % r), nil
			case token.And:
				return IntValue(l & r), nil
			case token.Or:
				return IntValue(l | r), nil
			case token.Xor:
				return IntValue(l ^ r), nil
			case token.AndNot:
				return IntValue(l &^ r), nil
			case token.Shl:
				return IntValue(l << uint64(r)), nil
			case token.Shr:
				return IntValue(l >> uint64(r)), nil
			case token.Less:
				return BoolValue(l < r), nil
			case token.Greater:
				return BoolValue(l > r), nil
			case token.LessEq:
				return BoolValue(l <= r), nil
			case token.GreaterEq:
				return BoolValue(l >= r), nil
			default:
				return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
			}

		case VT_FLOAT: // int op float => float
			l := float64(toInt(v))
			r := toFloat(rhs)
			switch op {
			case token.Add:
				return FloatValue(l + r), nil
			case token.Sub:
				return FloatValue(l - r), nil
			case token.Mul:
				return FloatValue(l * r), nil
			case token.Quo:
				return FloatValue(l / r), nil
			case token.Less:
				return BoolValue(l < r), nil
			case token.Greater:
				return BoolValue(l > r), nil
			case token.LessEq:
				return BoolValue(l <= r), nil
			case token.GreaterEq:
				return BoolValue(l >= r), nil
			default:
				return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
			}

		default:
			// int op any => int
			r, ok := rhs.AsInt()
			if !ok {
				return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
			}

			l := v.Int()
			switch op {
			case token.Add:
				return IntValue(l + r), nil
			case token.Sub:
				return IntValue(l - r), nil
			case token.Mul:
				return IntValue(l * r), nil
			case token.Quo:
				return IntValue(l / r), nil
			case token.Rem:
				return IntValue(l % r), nil
			case token.And:
				return IntValue(l & r), nil
			case token.Or:
				return IntValue(l | r), nil
			case token.Xor:
				return IntValue(l ^ r), nil
			case token.AndNot:
				return IntValue(l &^ r), nil
			case token.Shl:
				return IntValue(l << uint64(r)), nil
			case token.Shr:
				return IntValue(l >> uint64(r)), nil
			case token.Less:
				return BoolValue(l < r), nil
			case token.Greater:
				return BoolValue(l > r), nil
			case token.LessEq:
				return BoolValue(l <= r), nil
			case token.GreaterEq:
				return BoolValue(l >= r), nil
			default:
				return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
			}
		}

	case VT_FLOAT:
		r, ok := rhs.AsFloat()
		if !ok {
			return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

		l := toFloat(v)
		switch op {
		case token.Add:
			return FloatValue(l + r), nil
		case token.Sub:
			return FloatValue(l - r), nil
		case token.Mul:
			return FloatValue(l * r), nil
		case token.Quo:
			return FloatValue(l / r), nil
		case token.Less:
			return BoolValue(l < r), nil
		case token.Greater:
			return BoolValue(l > r), nil
		case token.LessEq:
			return BoolValue(l <= r), nil
		case token.GreaterEq:
			return BoolValue(l >= r), nil
		default:
			return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

	case VT_TIME:
		if rhs.IsInt() {
			r := rhs.Int()
			switch op {
			case token.Add: // time + int => time
				o := (*time.Time)(v.Ptr)
				return a.NewTimeValue(o.Add(time.Duration(r))), nil
			case token.Sub: // time - int => time
				o := (*time.Time)(v.Ptr)
				return a.NewTimeValue(o.Add(time.Duration(-r))), nil
			}
		}

		r, ok := rhs.AsTime()
		if !ok {
			return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

		switch op {
		case token.Sub: // time - time => int (duration)
			o := (*time.Time)(v.Ptr)
			return IntValue(int64(o.Sub(r))), nil
		case token.Less: // time < time => bool
			o := (*time.Time)(v.Ptr)
			return BoolValue(o.Before(r)), nil
		case token.Greater:
			o := (*time.Time)(v.Ptr)
			return BoolValue(o.After(r)), nil
		case token.LessEq:
			o := (*time.Time)(v.Ptr)
			return BoolValue(o.Equal(r) || o.Before(r)), nil
		case token.GreaterEq:
			o := (*time.Time)(v.Ptr)
			return BoolValue(o.Equal(r) || o.After(r)), nil
		}

		return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	default:
		return TypeBinaryOp[v.Type](v, a, op, rhs)
	}
}

func (v Value) Equal(rhs Value) bool {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_BOOL:
		return boolTypeEqual(v, rhs)
	case VT_CHAR:
		return charTypeEqual(v, rhs)
	case VT_INT:
		return intTypeEqual(v, rhs)
	case VT_FLOAT:
		return floatTypeEqual(v, rhs)
	default:
		return TypeEqual[v.Type](v, rhs)
	}
}

func (v *Value) Copy(alloc Allocator) Value {
	return TypeCopy[v.Type](*v, alloc)
}

func (v Value) MethodCall(vm VM, name string, args []Value) (Value, error) {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_TIME:
		return timeTypeMethodCall(v, vm, name, args)
	case VT_STRING:
		return stringTypeMethodCall(v, vm, name, args)
	case VT_BYTES:
		return bytesTypeMethodCall(v, vm, name, args)
	case VT_ARRAY:
		return arrayTypeMethodCall(v, vm, name, args)
	case VT_RECORD:
		return recordTypeMethodCall(v, vm, name, args)
	case VT_MAP:
		return mapTypeMethodCall(v, vm, name, args)
	default:
		return TypeMethodCall[v.Type](v, vm, name, args)
	}
}

func (v Value) Access(vm VM, index Value, mode Opcode) (Value, error) {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_STRING:
		return stringTypeAccess(v, vm.Allocator(), index, mode)
	case VT_BYTES:
		return bytesTypeAccess(v, vm.Allocator(), index, mode)
	case VT_ARRAY:
		return arrayTypeAccess(v, vm.Allocator(), index, mode)
	case VT_RECORD:
		return recordTypeAccess(v, vm.Allocator(), index, mode)
	case VT_MAP:
		return mapTypeAccess(v, vm.Allocator(), index, mode)
	default:
		return TypeAccess[v.Type](v, vm.Allocator(), index, mode)
	}
}

func (v Value) Assign(idx Value, val Value) error {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_ARRAY:
		return arrayTypeAssign(v, idx, val)
	case VT_RECORD:
		return recordTypeAssign(v, idx, val)
	case VT_MAP:
		return mapTypeAssign(v, idx, val)
	default:
		return TypeAssign[v.Type](v, idx, val)
	}
}

func (v Value) Iterator(alloc Allocator) Value {
	// Fast path, must be in sync with Function tables in init.go
	switch v.Type {
	case VT_STRING:
		return stringTypeIterator(v, alloc)
	case VT_BYTES:
		return bytesTypeIterator(v, alloc)
	case VT_ARRAY:
		return arrayTypeIterator(v, alloc)
	case VT_RECORD:
		return recordTypeIterator(v, alloc)
	case VT_MAP:
		return mapTypeIterator(v, alloc)
	default:
		return TypeIterator[v.Type](v, alloc)
	}
}

func (v Value) Call(vm VM, args []Value) (Value, error) {
	switch v.Type {
	case VT_BUILTIN_FUNCTION:
		return (*BuiltinFunction)(v.Ptr).Func(vm, args)

	case VT_COMPILED_FUNCTION:
		return vm.Call((*CompiledFunction)(v.Ptr), args)

	default:
		return TypeCall[v.Type](v, vm, args)
	}
}
