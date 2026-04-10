package core

import (
	"fmt"
	"time"

	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/token"
)

func defaultTypeName(v Value) string {
	return fmt.Sprintf("<unknown:%d>", v.Type)
}

func defaultTypeEncodeJSON(v Value) ([]byte, error) {
	return nil, fmt.Errorf("value type %s does not support JSON encoding", v.TypeName())
}

func defaultTypeEncodeBinary(v Value) ([]byte, error) {
	return nil, fmt.Errorf("value type %s does not support binary encoding", v.TypeName())
}

func defaultTypeDecodeBinary(v *Value, data []byte) error {
	return fmt.Errorf("value type %s does not support binary decoding", v.TypeName())
}

func defaultTypeString(v Value) string {
	return v.TypeName()
}

func defaultTypeInterface(v Value) any {
	return nil
}

func defaultTypeIsTrue(v Value) bool {
	return false
}

func defaultTypeIsImmutable(v Value) bool {
	return false
}

func defaultTypeIsIterable(v Value) bool {
	return false
}

func defaultTypeIsCallable(v Value) bool {
	return false
}

func defaultTypeAsBool(v Value) (bool, bool) {
	return false, false
}

func defaultTypeAsChar(v Value) (rune, bool) {
	return 0, false
}

func defaultTypeAsInt(v Value) (int64, bool) {
	return 0, false
}

func defaultTypeAsFloat(v Value) (float64, bool) {
	return 0, false
}

func defaultTypeAsTime(v Value) (time.Time, bool) {
	return time.Time{}, false
}

func defaultTypeAsString(v Value) (string, bool) {
	return "", false
}

func defaultTypeAsBytes(v Value) ([]byte, bool) {
	return nil, false
}

func defaultTypeCopy(v Value, a Allocator) Value {
	// by default copy as primitive value (used by Int, Float, etc)
	return v
}

func defaultTypeEqual(v Value, r Value) bool {
	return v == r
}

func defaultTypeBinaryOp(v Value, a Allocator, op token.Token, r Value) (Value, error) {
	return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
}

func defaultTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	return UndefinedValue(), errs.NewInvalidMethodError(name, v.TypeName())
}

func defaultTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	return UndefinedValue(), errs.NewNotAccessibleError(v.TypeName())
}

func defaultTypeAssign(v Value, index Value, r Value) error {
	return errs.NewNotAssignableError(v.TypeName())
}

func defaultTypeIterator(v Value, a Allocator) Value {
	return UndefinedValue()
}

func defaultTypeNext(v *Value) bool {
	return false
}

func defaultTypeKey(v Value, a Allocator) Value {
	return UndefinedValue()
}

func defaultTypeValue(v Value, a Allocator) Value {
	return UndefinedValue()
}

func defaultTypeArity(v Value) int {
	return 0
}

func defaultTypeIsVariadic(v Value) bool {
	return false
}

func defaultTypeCall(v Value, vm VM, args []Value) (Value, error) {
	return UndefinedValue(), errs.NewNotCallableError(v.TypeName())
}

func init() {
	for i := 0; i < 256; i++ {
		TypeName[i] = defaultTypeName
		TypeEncodeJSON[i] = defaultTypeEncodeJSON
		TypeEncodeBinary[i] = defaultTypeEncodeBinary
		TypeDecodeBinary[i] = defaultTypeDecodeBinary
		TypeString[i] = defaultTypeString
		TypeInterface[i] = defaultTypeInterface

		TypeIsTrue[i] = defaultTypeIsTrue
		TypeIsImmutable[i] = defaultTypeIsImmutable
		TypeIsIterable[i] = defaultTypeIsIterable
		TypeIsCallable[i] = defaultTypeIsCallable

		TypeAsBool[i] = defaultTypeAsBool
		TypeAsChar[i] = defaultTypeAsChar
		TypeAsInt[i] = defaultTypeAsInt
		TypeAsFloat[i] = defaultTypeAsFloat
		TypeAsTime[i] = defaultTypeAsTime
		TypeAsString[i] = defaultTypeAsString
		TypeAsBytes[i] = defaultTypeAsBytes

		TypeCopy[i] = defaultTypeCopy
		TypeEqual[i] = defaultTypeEqual
		TypeBinaryOp[i] = defaultTypeBinaryOp
		TypeMethodCall[i] = defaultTypeMethodCall

		TypeAccess[i] = defaultTypeAccess
		TypeAssign[i] = defaultTypeAssign
		TypeIterator[i] = defaultTypeIterator

		TypeNext[i] = defaultTypeNext
		TypeKey[i] = defaultTypeKey
		TypeValue[i] = defaultTypeValue

		TypeArity[i] = defaultTypeArity
		TypeIsVariadic[i] = defaultTypeIsVariadic
		TypeCall[i] = defaultTypeCall
	}

	// Undefined
	TypeName[VT_UNDEFINED] = undefinedTypeName
	TypeEncodeJSON[VT_UNDEFINED] = undefinedTypeEncodeJSON
	TypeEncodeBinary[VT_UNDEFINED] = undefinedTypeEncodeBinary
	TypeDecodeBinary[VT_UNDEFINED] = undefinedTypeDecodeBinary
	TypeString[VT_UNDEFINED] = undefinedTypeString
	TypeInterface[VT_UNDEFINED] = undefinedTypeInterface
	TypeIsIterable[VT_UNDEFINED] = undefinedTypeIsIterable
	TypeEqual[VT_UNDEFINED] = undefinedTypeEqual
	TypeAccess[VT_UNDEFINED] = undefinedTypeAccess
	TypeAsBool[VT_UNDEFINED] = undefinedTypeAsBool

	// ValuePtr
	TypeName[VT_VALUE_PTR] = valuePtrTypeName

	// BuiltinFunction
	TypeName[VT_BUILTIN_FUNCTION] = builtinFunctionTypeName
	TypeEncodeBinary[VT_BUILTIN_FUNCTION] = builtinFunctionTypeEncodeBinary
	TypeDecodeBinary[VT_BUILTIN_FUNCTION] = builtinFunctionTypeDecodeBinary
	TypeString[VT_BUILTIN_FUNCTION] = builtinFunctionTypeString
	TypeArity[VT_BUILTIN_FUNCTION] = builtinFunctionTypeArity
	TypeIsTrue[VT_BUILTIN_FUNCTION] = builtinFunctionTypeIsTrue
	TypeIsCallable[VT_BUILTIN_FUNCTION] = builtinFunctionTypeIsCallable
	TypeIsVariadic[VT_BUILTIN_FUNCTION] = builtinFunctionTypeIsVariadic
	TypeEqual[VT_BUILTIN_FUNCTION] = builtinFunctionTypeEqual
	TypeCall[VT_BUILTIN_FUNCTION] = builtinFunctionTypeCall

	// CompiledFunction
	TypeName[VT_COMPILED_FUNCTION] = compiledFunctionTypeName
	TypeEncodeBinary[VT_COMPILED_FUNCTION] = compiledFunctionTypeEncodeBinary
	TypeDecodeBinary[VT_COMPILED_FUNCTION] = compiledFunctionTypeDecodeBinary
	TypeString[VT_COMPILED_FUNCTION] = compiledFunctionTypeString
	TypeArity[VT_COMPILED_FUNCTION] = compiledFunctionTypeArity
	TypeIsTrue[VT_COMPILED_FUNCTION] = compiledFunctionTypeIsTrue
	TypeIsCallable[VT_COMPILED_FUNCTION] = compiledFunctionTypeIsCallable
	TypeIsVariadic[VT_COMPILED_FUNCTION] = compiledFunctionTypeIsVariadic
	TypeEqual[VT_COMPILED_FUNCTION] = compiledFunctionTypeEqual
	TypeCall[VT_COMPILED_FUNCTION] = compiledFunctionTypeCall

	// Error
	TypeName[VT_ERROR] = errorTypeName
	TypeEncodeJSON[VT_ERROR] = errorTypeEncodeJSON
	TypeEncodeBinary[VT_ERROR] = errorTypeEncodeBinary
	TypeDecodeBinary[VT_ERROR] = errorTypeDecodeBinary
	TypeString[VT_ERROR] = errorTypeString
	TypeInterface[VT_ERROR] = errorTypeInterface
	TypeEqual[VT_ERROR] = errorTypeEqual
	TypeCopy[VT_ERROR] = errorTypeCopy
	TypeMethodCall[VT_ERROR] = errorTypeMethodCall
	TypeIsTrue[VT_ERROR] = errorTypeIsTrue
	TypeAsString[VT_ERROR] = errorTypeAsString
	TypeAsBool[VT_ERROR] = errorTypeAsBool

	// Bool
	TypeName[VT_BOOL] = boolTypeName
	TypeEncodeJSON[VT_BOOL] = boolTypeEncodeJSON
	TypeEncodeBinary[VT_BOOL] = boolTypeEncodeBinary
	TypeDecodeBinary[VT_BOOL] = boolTypeDecodeBinary
	TypeString[VT_BOOL] = boolTypeString
	TypeInterface[VT_BOOL] = boolTypeInterface
	TypeIsTrue[VT_BOOL] = boolTypeIsTrue
	TypeAsString[VT_BOOL] = boolTypeAsString
	TypeAsInt[VT_BOOL] = boolTypeAsInt
	TypeAsBool[VT_BOOL] = boolTypeAsBool
	TypeEqual[VT_BOOL] = boolTypeEqual
	TypeMethodCall[VT_BOOL] = boolTypeMethodCall

	// Char
	TypeName[VT_CHAR] = charTypeName
	TypeEncodeJSON[VT_CHAR] = charTypeEncodeJSON
	TypeEncodeBinary[VT_CHAR] = charTypeEncodeBinary
	TypeDecodeBinary[VT_CHAR] = charTypeDecodeBinary
	TypeString[VT_CHAR] = charTypeString
	TypeInterface[VT_CHAR] = charTypeInterface
	TypeIsTrue[VT_CHAR] = charTypeIsTrue
	TypeAsString[VT_CHAR] = charTypeAsString
	TypeAsInt[VT_CHAR] = charTypeAsInt
	TypeAsBool[VT_CHAR] = charTypeAsBool
	TypeAsChar[VT_CHAR] = charTypeAsChar
	TypeBinaryOp[VT_CHAR] = charTypeBinaryOp
	TypeEqual[VT_CHAR] = charTypeEqual
	TypeMethodCall[VT_CHAR] = charTypeMethodCall

	// Int
	TypeName[VT_INT] = intTypeName
	TypeEncodeJSON[VT_INT] = intTypeEncodeJSON
	TypeEncodeBinary[VT_INT] = intTypeEncodeBinary
	TypeDecodeBinary[VT_INT] = intTypeDecodeBinary
	TypeString[VT_INT] = intTypeString
	TypeInterface[VT_INT] = intTypeInterface
	TypeIsTrue[VT_INT] = intTypeIsTrue
	TypeAsString[VT_INT] = intTypeAsString
	TypeAsInt[VT_INT] = intTypeAsInt
	TypeAsFloat[VT_INT] = intTypeAsFloat
	TypeAsBool[VT_INT] = intTypeAsBool
	TypeAsChar[VT_INT] = intTypeAsChar
	TypeAsTime[VT_INT] = intTypeAsTime
	TypeBinaryOp[VT_INT] = intTypeBinaryOp
	TypeEqual[VT_INT] = intTypeEqual
	TypeMethodCall[VT_INT] = intTypeMethodCall

	// Float
	TypeName[VT_FLOAT] = floatTypeName
	TypeEncodeJSON[VT_FLOAT] = floatTypeEncodeJSON
	TypeEncodeBinary[VT_FLOAT] = floatTypeEncodeBinary
	TypeDecodeBinary[VT_FLOAT] = floatTypeDecodeBinary
	TypeString[VT_FLOAT] = floatTypeString
	TypeInterface[VT_FLOAT] = floatTypeInterface
	TypeIsTrue[VT_FLOAT] = floatTypeIsTrue
	TypeAsString[VT_FLOAT] = floatTypeAsString
	TypeAsInt[VT_FLOAT] = floatTypeAsInt
	TypeAsFloat[VT_FLOAT] = floatTypeAsFloat
	TypeAsBool[VT_FLOAT] = floatTypeAsBool
	TypeBinaryOp[VT_FLOAT] = floatTypeBinaryOp
	TypeEqual[VT_FLOAT] = floatTypeEqual
	TypeMethodCall[VT_FLOAT] = floatTypeMethodCall

	// Time
	TypeName[VT_TIME] = timeTypeName
	TypeEncodeJSON[VT_TIME] = timeTypeEncodeJSON
	TypeEncodeBinary[VT_TIME] = timeTypeEncodeBinary
	TypeDecodeBinary[VT_TIME] = timeTypeDecodeBinary
	TypeString[VT_TIME] = timeTypeString
	TypeInterface[VT_TIME] = timeTypeInterface
	TypeBinaryOp[VT_TIME] = timeTypeBinaryOp
	TypeEqual[VT_TIME] = timeTypeEqual
	TypeCopy[VT_TIME] = timeTypeCopy
	TypeMethodCall[VT_TIME] = timeTypeMethodCall
	TypeIsTrue[VT_TIME] = timeTypeIsTrue
	TypeAsString[VT_TIME] = timeTypeAsString
	TypeAsInt[VT_TIME] = timeTypeAsInt
	TypeAsBool[VT_TIME] = timeTypeAsBool
	TypeAsTime[VT_TIME] = timeTypeAsTime

	// String
	TypeName[VT_STRING] = stringTypeName
	TypeEncodeJSON[VT_STRING] = stringTypeEncodeJSON
	TypeEncodeBinary[VT_STRING] = stringTypeEncodeBinary
	TypeDecodeBinary[VT_STRING] = stringTypeDecodeBinary
	TypeString[VT_STRING] = stringTypeString
	TypeInterface[VT_STRING] = stringTypeInterface
	TypeBinaryOp[VT_STRING] = stringTypeBinaryOp
	TypeEqual[VT_STRING] = stringTypeEqual
	TypeCopy[VT_STRING] = stringTypeCopy
	TypeMethodCall[VT_STRING] = stringTypeMethodCall
	TypeAccess[VT_STRING] = stringTypeAccess
	TypeIsIterable[VT_STRING] = stringTypeIsIterable
	TypeIterator[VT_STRING] = stringTypeIterator
	TypeIsTrue[VT_STRING] = stringTypeIsTrue
	TypeAsString[VT_STRING] = stringTypeAsString
	TypeAsInt[VT_STRING] = stringTypeAsInt
	TypeAsFloat[VT_STRING] = stringTypeAsFloat
	TypeAsBool[VT_STRING] = stringTypeAsBool
	TypeAsChar[VT_STRING] = stringTypeAsChar
	TypeAsBytes[VT_STRING] = stringTypeAsBytes
	TypeAsTime[VT_STRING] = stringTypeAsTime

	// Bytes
	TypeName[VT_BYTES] = bytesTypeName
	TypeEncodeJSON[VT_BYTES] = bytesTypeEncodeJSON
	TypeEncodeBinary[VT_BYTES] = bytesTypeEncodeBinary
	TypeDecodeBinary[VT_BYTES] = bytesTypeDecodeBinary
	TypeString[VT_BYTES] = bytesTypeString
	TypeInterface[VT_BYTES] = bytesTypeInterface
	TypeBinaryOp[VT_BYTES] = bytesTypeBinaryOp
	TypeEqual[VT_BYTES] = bytesTypeEqual
	TypeCopy[VT_BYTES] = bytesTypeCopy
	TypeMethodCall[VT_BYTES] = bytesTypeMethodCall
	TypeAccess[VT_BYTES] = bytesTypeAccess
	TypeIsIterable[VT_BYTES] = bytesTypeIsIterable
	TypeIterator[VT_BYTES] = bytesTypeIterator
	TypeIsTrue[VT_BYTES] = bytesTypeIsTrue
	TypeAsString[VT_BYTES] = bytesTypeAsString
	TypeAsBool[VT_BYTES] = bytesTypeAsBool
	TypeAsBytes[VT_BYTES] = bytesTypeAsBytes

	// Array
	TypeName[VT_ARRAY] = arrayTypeName
	TypeEncodeJSON[VT_ARRAY] = arrayTypeEncodeJSON
	TypeEncodeBinary[VT_ARRAY] = arrayTypeEncodeBinary
	TypeDecodeBinary[VT_ARRAY] = arrayTypeDecodeBinary
	TypeString[VT_ARRAY] = arrayTypeString
	TypeInterface[VT_ARRAY] = arrayTypeInterface
	TypeBinaryOp[VT_ARRAY] = arrayTypeBinaryOp
	TypeEqual[VT_ARRAY] = arrayTypeEqual
	TypeCopy[VT_ARRAY] = arrayTypeCopy
	TypeMethodCall[VT_ARRAY] = arrayTypeMethodCall
	TypeAccess[VT_ARRAY] = arrayTypeAccess
	TypeAssign[VT_ARRAY] = arrayTypeAssign
	TypeIsIterable[VT_ARRAY] = arrayTypeIsIterable
	TypeIterator[VT_ARRAY] = arrayTypeIterator
	TypeIsImmutable[VT_ARRAY] = arrayTypeIsImmutable
	TypeIsTrue[VT_ARRAY] = arrayTypeIsTrue
	TypeAsString[VT_ARRAY] = arrayTypeAsString
	TypeAsBool[VT_ARRAY] = arrayTypeAsBool
	TypeAsBytes[VT_ARRAY] = arrayTypeAsBytes

	// Record
	TypeName[VT_RECORD] = recordTypeName
	TypeEncodeJSON[VT_RECORD] = recordTypeEncodeJSON
	TypeEncodeBinary[VT_RECORD] = recordTypeEncodeBinary
	TypeDecodeBinary[VT_RECORD] = recordTypeDecodeBinary
	TypeString[VT_RECORD] = recordTypeString
	TypeInterface[VT_RECORD] = recordTypeInterface
	TypeEqual[VT_RECORD] = recordTypeEqual
	TypeCopy[VT_RECORD] = recordTypeCopy
	TypeMethodCall[VT_RECORD] = recordTypeMethodCall
	TypeAccess[VT_RECORD] = recordTypeAccess
	TypeAssign[VT_RECORD] = recordTypeAssign
	TypeIsIterable[VT_RECORD] = recordTypeIsIterable
	TypeIterator[VT_RECORD] = recordTypeIterator
	TypeIsImmutable[VT_RECORD] = recordTypeIsImmutable
	TypeIsTrue[VT_RECORD] = recordTypeIsTrue
	TypeAsString[VT_RECORD] = recordTypeAsString
	TypeAsBool[VT_RECORD] = recordTypeAsBool

	// Map
	TypeName[VT_MAP] = mapTypeName
	TypeEncodeJSON[VT_MAP] = mapTypeEncodeJSON
	TypeEncodeBinary[VT_MAP] = mapTypeEncodeBinary
	TypeDecodeBinary[VT_MAP] = mapTypeDecodeBinary
	TypeString[VT_MAP] = mapTypeString
	TypeInterface[VT_MAP] = mapTypeInterface
	TypeEqual[VT_MAP] = mapTypeEqual
	TypeCopy[VT_MAP] = mapTypeCopy
	TypeMethodCall[VT_MAP] = mapTypeMethodCall
	TypeAccess[VT_MAP] = mapTypeAccess
	TypeAssign[VT_MAP] = mapTypeAssign
	TypeIsIterable[VT_MAP] = mapTypeIsIterable
	TypeIterator[VT_MAP] = mapTypeIterator
	TypeIsImmutable[VT_MAP] = mapTypeIsImmutable
	TypeIsTrue[VT_MAP] = mapTypeIsTrue
	TypeAsString[VT_MAP] = mapTypeAsString
	TypeAsBool[VT_MAP] = mapTypeAsBool

	// StringIterator
	TypeName[VT_STRING_ITERATOR] = stringIteratorTypeName
	TypeString[VT_STRING_ITERATOR] = stringIteratorTypeString
	TypeNext[VT_STRING_ITERATOR] = stringIteratorTypeNext
	TypeKey[VT_STRING_ITERATOR] = stringIteratorTypeKey
	TypeValue[VT_STRING_ITERATOR] = stringIteratorTypeValue
	TypeEqual[VT_STRING_ITERATOR] = stringIteratorTypeEqual

	// BytesIterator
	TypeName[VT_BYTES_ITERATOR] = bytesIteratorTypeName
	TypeString[VT_BYTES_ITERATOR] = bytesIteratorTypeString
	TypeNext[VT_BYTES_ITERATOR] = bytesIteratorTypeNext
	TypeKey[VT_BYTES_ITERATOR] = bytesIteratorTypeKey
	TypeValue[VT_BYTES_ITERATOR] = bytesIteratorTypeValue
	TypeEqual[VT_BYTES_ITERATOR] = bytesIteratorTypeEqual

	// ArrayIterator
	TypeName[VT_ARRAY_ITERATOR] = arrayIteratorTypeName
	TypeString[VT_ARRAY_ITERATOR] = arrayIteratorTypeString
	TypeNext[VT_ARRAY_ITERATOR] = arrayIteratorTypeNext
	TypeKey[VT_ARRAY_ITERATOR] = arrayIteratorTypeKey
	TypeValue[VT_ARRAY_ITERATOR] = arrayIteratorTypeValue
	TypeEqual[VT_ARRAY_ITERATOR] = arrayIteratorTypeEqual

	// MapIterator
	TypeName[VT_MAP_ITERATOR] = mapIteratorTypeName
	TypeString[VT_MAP_ITERATOR] = mapIteratorTypeString
	TypeNext[VT_MAP_ITERATOR] = mapIteratorTypeNext
	TypeKey[VT_MAP_ITERATOR] = mapIteratorTypeKey
	TypeValue[VT_MAP_ITERATOR] = mapIteratorTypeValue
	TypeEqual[VT_MAP_ITERATOR] = mapIteratorTypeEqual
}
