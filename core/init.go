package core

import (
	"github.com/jokruger/kavun/internal/hook"
	"github.com/jokruger/kavun/internal/seq"
)

func init() {
	// Initialize all types with defaults
	for i := range 256 {
		ValueTypes[i] = ValueTypeDefaults
	}

	// Undefined
	setValueType(VT_UNDEFINED, ValueType{
		Name:         hook.Const[Value](undefinedTypeName),
		Interface:    undefinedTypeInterface,
		String:       undefinedTypeString,
		Format:       undefinedTypeFormat,
		EncodeJSON:   undefinedTypeEncodeJSON,
		EncodeBinary: undefinedTypeEncodeBinary,
		DecodeBinary: undefinedTypeDecodeBinary,
		IsTrue:       hook.Const[Value, bool](false), // undefined is always false
		IsIterable:   hook.Const[Value, bool](true),
		Equal:        func(v Value, r Value) bool { return v.Type == r.Type && v.Data == r.Data && v.Ptr == r.Ptr },
		MethodCall:   undefinedTypeMethodCall,
		Access:       undefinedTypeAccess,
		AsBool:       undefinedTypeAsBool,
	})

	// ValuePtr
	setValueType(VT_VALUE_PTR, ValueType{
		Name: valuePtrTypeName,
	})

	// BuiltinFunction
	setValueType(VT_BUILTIN_FUNCTION, ValueType{
		Name:         builtinFunctionTypeName,
		String:       builtinFunctionTypeString,
		EncodeBinary: builtinFunctionTypeEncodeBinary,
		DecodeBinary: builtinFunctionTypeDecodeBinary,
		IsTrue:       hook.Const[Value, bool](true),
		IsCallable:   hook.Const[Value, bool](true),
		IsVariadic:   builtinFunctionTypeIsVariadic,
		Equal:        builtinFunctionTypeEqual,
		Arity:        builtinFunctionTypeArity,
		Call:         builtinFunctionTypeCall,
		MethodCall:   builtinFunctionTypeMethodCall,
	})

	// CompiledFunction
	setValueType(VT_COMPILED_FUNCTION, ValueType{
		Name:         compiledFunctionTypeName,
		String:       compiledFunctionTypeString,
		EncodeBinary: compiledFunctionTypeEncodeBinary,
		DecodeBinary: compiledFunctionTypeDecodeBinary,
		IsTrue:       hook.Const[Value, bool](true),
		IsCallable:   hook.Const[Value, bool](true),
		IsVariadic:   compiledFunctionTypeIsVariadic,
		Equal:        compiledFunctionTypeEqual,
		Arity:        compiledFunctionTypeArity,
		Call:         compiledFunctionTypeCall,
		MethodCall:   compiledFunctionTypeMethodCall,
	})

	// Error
	setValueType(VT_ERROR, ValueType{
		Name:         hook.Const[Value](errorTypeName),
		String:       errorTypeString,
		Format:       errorTypeFormat,
		Interface:    errorTypeInterface,
		EncodeJSON:   errorTypeEncodeJSON,
		EncodeBinary: errorTypeEncodeBinary,
		DecodeBinary: errorTypeDecodeBinary,
		IsTrue:       hook.Const[Value, bool](false), // error is always false
		Equal:        errorTypeEqual,
		Copy:         errorTypeCopy,
		MethodCall:   errorTypeMethodCall,
		AsString:     errorTypeAsString,
		AsBool:       errorTypeAsBool,
	})

	// Bool
	setValueType(VT_BOOL, ValueType{
		Name:         hook.Const[Value](boolTypeName),
		String:       boolTypeString,
		Format:       boolTypeFormat,
		Interface:    boolTypeInterface,
		EncodeJSON:   boolTypeEncodeJSON,
		EncodeBinary: boolTypeEncodeBinary,
		DecodeBinary: boolTypeDecodeBinary,
		IsTrue:       boolTypeIsTrue,
		Equal:        boolTypeEqual,
		MethodCall:   boolTypeMethodCall,
		Len:          hook.Const[Value, int64](1),
		AsString:     boolTypeAsString,
		AsInt:        boolTypeAsInt,
		AsBool:       boolTypeAsBool,
		AsByte:       boolTypeAsByte,
	})

	// Byte
	setValueType(VT_BYTE, ValueType{
		Name:         hook.Const[Value](byteTypeName),
		String:       byteTypeString,
		Format:       byteTypeFormat,
		Interface:    byteTypeInterface,
		EncodeJSON:   byteTypeEncodeJSON,
		EncodeBinary: byteTypeEncodeBinary,
		DecodeBinary: byteTypeDecodeBinary,
		IsTrue:       byteTypeIsTrue,
		Equal:        byteTypeEqual,
		Len:          hook.Const[Value, int64](1),
		BinaryOp:     byteTypeBinaryOp,
		MethodCall:   byteTypeMethodCall,
		AsString:     byteTypeAsString,
		AsInt:        byteTypeAsInt,
		AsBool:       byteTypeAsBool,
		AsRune:       byteTypeAsRune,
		AsByte:       byteTypeAsByte,
		AsFloat:      byteTypeAsFloat,
		AsDecimal:    byteTypeAsDecimal,
	})

	// Rune
	setValueType(VT_RUNE, ValueType{
		Name:         hook.Const[Value](runeTypeName),
		String:       runeTypeString,
		Format:       runeTypeFormat,
		Interface:    runeTypeInterface,
		EncodeJSON:   runeTypeEncodeJSON,
		EncodeBinary: runeTypeEncodeBinary,
		DecodeBinary: runeTypeDecodeBinary,
		IsTrue:       runeTypeIsTrue,
		Equal:        runeTypeEqual,
		Len:          hook.Const[Value, int64](1),
		BinaryOp:     runeTypeBinaryOp,
		MethodCall:   runeTypeMethodCall,
		AsString:     runeTypeAsString,
		AsInt:        runeTypeAsInt,
		AsBool:       runeTypeAsBool,
		AsRune:       runeTypeAsRune,
		AsByte:       runeTypeAsByte,
	})

	// Int
	setValueType(VT_INT, ValueType{
		Name:         hook.Const[Value](intTypeName),
		String:       intTypeString,
		Format:       intTypeFormat,
		Interface:    intTypeInterface,
		EncodeJSON:   intTypeEncodeJSON,
		EncodeBinary: intTypeEncodeBinary,
		DecodeBinary: intTypeDecodeBinary,
		IsTrue:       intTypeIsTrue,
		Equal:        intTypeEqual,
		Len:          hook.Const[Value, int64](1),
		UnaryOp:      intTypeUnaryOp,
		BinaryOp:     intTypeBinaryOp,
		MethodCall:   intTypeMethodCall,
		AsString:     intTypeAsString,
		AsInt:        intTypeAsInt,
		AsFloat:      intTypeAsFloat,
		AsDecimal:    intTypeAsDecimal,
		AsBool:       intTypeAsBool,
		AsRune:       intTypeAsRune,
		AsTime:       intTypeAsTime,
		AsByte:       intTypeAsByte,
	})

	// Float
	setValueType(VT_FLOAT, ValueType{
		Name:         hook.Const[Value](floatTypeName),
		String:       floatTypeString,
		Format:       floatTypeFormat,
		Interface:    floatTypeInterface,
		EncodeJSON:   floatTypeEncodeJSON,
		EncodeBinary: floatTypeEncodeBinary,
		DecodeBinary: floatTypeDecodeBinary,
		IsTrue:       floatTypeIsTrue,
		Equal:        floatTypeEqual,
		Len:          hook.Const[Value, int64](1),
		UnaryOp:      floatTypeUnaryOp,
		BinaryOp:     floatTypeBinaryOp,
		MethodCall:   floatTypeMethodCall,
		AsString:     floatTypeAsString,
		AsInt:        floatTypeAsInt,
		AsFloat:      floatTypeAsFloat,
		AsDecimal:    floatTypeAsDecimal,
		AsBool:       floatTypeAsBool,
	})

	// Decimal
	setValueType(VT_DECIMAL, ValueType{
		Name:         hook.Const[Value](decimalTypeName),
		String:       decimalTypeString,
		Format:       decimalTypeFormat,
		Interface:    decimalTypeInterface,
		EncodeJSON:   decimalTypeEncodeJSON,
		EncodeBinary: decimalTypeEncodeBinary,
		DecodeBinary: decimalTypeDecodeBinary,
		IsTrue:       decimalTypeIsTrue,
		Equal:        decimalTypeEqual,
		Len:          hook.Const[Value, int64](1),
		UnaryOp:      decimalTypeUnaryOp,
		BinaryOp:     decimalTypeBinaryOp,
		MethodCall:   decimalTypeMethodCall,
		AsString:     decimalTypeAsString,
		AsInt:        decimalTypeAsInt,
		AsFloat:      decimalTypeAsFloat,
		AsDecimal:    decimalTypeAsDecimal,
		AsBool:       decimalTypeAsBool,
	})

	// Time
	setValueType(VT_TIME, ValueType{
		Name:         hook.Const[Value](timeTypeName),
		String:       timeTypeString,
		Format:       timeTypeFormat,
		Interface:    timeTypeInterface,
		EncodeJSON:   timeTypeEncodeJSON,
		EncodeBinary: timeTypeEncodeBinary,
		DecodeBinary: timeTypeDecodeBinary,
		IsTrue:       timeTypeIsTrue,
		Equal:        timeTypeEqual,
		Len:          hook.Const[Value, int64](1),
		BinaryOp:     timeTypeBinaryOp,
		MethodCall:   timeTypeMethodCall,
		AsString:     timeTypeAsString,
		AsInt:        timeTypeAsInt,
		AsBool:       timeTypeAsBool,
		AsTime:       timeTypeAsTime,
	})

	// String
	setValueType(VT_STRING, ValueType{
		Name:         hook.Const[Value](stringTypeName),
		String:       stringTypeString,
		Format:       stringTypeFormat,
		Interface:    stringTypeInterface,
		EncodeJSON:   stringTypeEncodeJSON,
		EncodeBinary: stringTypeEncodeBinary,
		DecodeBinary: stringTypeDecodeBinary,
		IsTrue:       stringTypeIsTrue,
		IsIterable:   hook.Const[Value, bool](true),
		Iterator:     stringTypeIterator,
		Equal:        stringTypeEqual,
		Len:          stringTypeLen,
		BinaryOp:     stringTypeBinaryOp,
		MethodCall:   stringTypeMethodCall,
		Access:       stringTypeAccess,
		Contains:     stringTypeContains,
		Slice:        stringTypeSlice,
		SliceStep:    stringTypeSliceStep,
		AsBool:       stringTypeAsBool,
		AsInt:        stringTypeAsInt,
		AsByte:       stringTypeAsByte,
		AsFloat:      stringTypeAsFloat,
		AsDecimal:    stringTypeAsDecimal,
		AsTime:       stringTypeAsTime,
		AsString:     stringTypeAsString,
		AsRunes:      stringTypeAsRunes,
		AsBytes:      stringTypeAsBytes,
		AsArray:      stringTypeAsArray,
	})

	// Runes
	setValueType(VT_RUNES, ValueType{
		Name:         hook.ContainerTypeName[Value](runesTypeName, immutableRunesTypeName),
		String:       runesTypeString,
		Format:       runesTypeFormat,
		Interface:    runesTypeInterface,
		EncodeJSON:   runesTypeEncodeJSON,
		EncodeBinary: runesTypeEncodeBinary,
		DecodeBinary: runesTypeDecodeBinary,
		IsTrue:       runesTypeIsTrue,
		IsIterable:   hook.Const[Value, bool](true),
		Iterator:     runesTypeIterator,
		Equal:        runesTypeEqual,
		Copy:         runesTypeCopy,
		Len:          runesTypeLen,
		BinaryOp:     runesTypeBinaryOp,
		MethodCall:   runesTypeMethodCall,
		Access:       seq.AccessHook[Value, Arena](RuneValue),
		Assign:       seq.AssignHook[Value](Value.AsRune, runeTypeName),
		Append:       runesTypeAppend,
		Contains:     runesTypeContains,
		Slice:        runesTypeSlice,
		SliceStep:    runesTypeSliceStep,
		AsBool:       runesTypeAsBool,
		AsInt:        runesTypeAsInt,
		AsByte:       runesTypeAsByte,
		AsFloat:      runesTypeAsFloat,
		AsDecimal:    runesTypeAsDecimal,
		AsTime:       runesTypeAsTime,
		AsString:     runesTypeAsString,
		AsRunes:      runesTypeAsRunes,
		AsBytes:      runesTypeAsBytes,
		AsArray:      runesTypeAsArray,
	})

	// Bytes
	setValueType(VT_BYTES, ValueType{
		Name:         hook.ContainerTypeName[Value](bytesTypeName, immutableBytesTypeName),
		String:       bytesTypeString,
		Format:       bytesTypeFormat,
		Interface:    bytesTypeInterface,
		EncodeJSON:   bytesTypeEncodeJSON,
		EncodeBinary: bytesTypeEncodeBinary,
		DecodeBinary: bytesTypeDecodeBinary,
		IsTrue:       bytesTypeIsTrue,
		IsIterable:   hook.Const[Value, bool](true),
		Iterator:     bytesTypeIterator,
		Equal:        bytesTypeEqual,
		Copy:         bytesTypeCopy,
		Len:          bytesTypeLen,
		BinaryOp:     bytesTypeBinaryOp,
		MethodCall:   bytesTypeMethodCall,
		Access:       seq.AccessHook[Value, Arena](ByteValue),
		Assign:       seq.AssignHook[Value](Value.AsByte, byteTypeName),
		Append:       bytesTypeAppend,
		Contains:     bytesTypeContains,
		Slice:        bytesTypeSlice,
		SliceStep:    bytesTypeSliceStep,
		AsBool:       bytesTypeAsBool,
		AsString:     bytesTypeAsString,
		AsBytes:      bytesTypeAsBytes,
		AsArray:      bytesTypeAsArray,
	})

	// Array
	setValueType(VT_ARRAY, ValueType{
		Name:         hook.ContainerTypeName[Value](arrayTypeName, immutableArrayTypeName),
		String:       arrayTypeString,
		Format:       arrayTypeFormat,
		Interface:    arrayTypeInterface,
		EncodeJSON:   arrayTypeEncodeJSON,
		EncodeBinary: arrayTypeEncodeBinary,
		DecodeBinary: arrayTypeDecodeBinary,
		IsTrue:       arrayTypeIsTrue,
		IsIterable:   hook.Const[Value, bool](true),
		Iterator:     arrayTypeIterator,
		Equal:        arrayTypeEqual,
		Copy:         arrayTypeCopy,
		Len:          arrayTypeLen,
		BinaryOp:     arrayTypeBinaryOp,
		MethodCall:   arrayTypeMethodCall,
		Access:       seq.AccessHook[Value, Arena](RefValue),
		Assign:       seq.AssignHook[Value](Value.AsValue, anyTypeName),
		Contains:     arrayTypeContains,
		Append:       arrayTypeAppend,
		Slice:        arrayTypeSlice,
		SliceStep:    arrayTypeSliceStep,
		AsBool:       arrayTypeAsBool,
		AsString:     arrayTypeAsString,
		AsRunes:      arrayTypeAsRunes,
		AsBytes:      arrayTypeAsBytes,
		AsArray:      arrayTypeAsArray,
	})

	// Record
	setValueType(VT_RECORD, ValueType{
		Name:         hook.ContainerTypeName[Value](recordTypeName, immutableRecordTypeName),
		String:       recordTypeString,
		Format:       recordTypeFormat,
		Interface:    genericDictTypeInterface,
		EncodeJSON:   genericDictTypeEncodeJSON,
		EncodeBinary: genericDictTypeEncodeBinary,
		DecodeBinary: genericDictTypeDecodeBinary,
		IsTrue:       genericDictTypeIsTrue,
		IsIterable:   hook.Const[Value, bool](true),
		Iterator:     genericDictTypeIterator,
		Equal:        genericDictTypeEqual,
		Copy:         recordTypeCopy,
		Len:          genericDictTypeLen,
		MethodCall:   recordTypeMethodCall,
		Access:       recordTypeAccess,
		Assign:       genericDictTypeAssign,
		Contains:     genericDictTypeContains,
		Delete:       genericDictTypeDelete,
		AsBool:       genericDictTypeAsBool,
		AsString:     genericDictTypeAsString,
		AsDict:       genericDictTypeAsDict,
	})

	// Dict
	setValueType(VT_DICT, ValueType{
		Name:         hook.ContainerTypeName[Value](dictTypeName, immutableDictTypeName),
		String:       dictTypeString,
		Format:       dictTypeFormat,
		Interface:    genericDictTypeInterface,
		EncodeJSON:   genericDictTypeEncodeJSON,
		EncodeBinary: genericDictTypeEncodeBinary,
		DecodeBinary: genericDictTypeDecodeBinary,
		IsTrue:       genericDictTypeIsTrue,
		IsIterable:   hook.Const[Value, bool](true),
		Iterator:     genericDictTypeIterator,
		Equal:        genericDictTypeEqual,
		Copy:         dictTypeCopy,
		Len:          genericDictTypeLen,
		MethodCall:   dictTypeMethodCall,
		Access:       dictTypeAccess,
		Assign:       genericDictTypeAssign,
		Contains:     genericDictTypeContains,
		Delete:       genericDictTypeDelete,
		AsBool:       genericDictTypeAsBool,
		AsString:     genericDictTypeAsString,
		AsDict:       genericDictTypeAsDict,
	})

	// IntRange
	setValueType(VT_INT_RANGE, ValueType{
		Name:         hook.Const[Value](intRangeTypeName),
		EncodeBinary: intRangeTypeEncodeBinary,
		DecodeBinary: intRangeTypeDecodeBinary,
		String:       intRangeTypeString,
		Format:       intRangeTypeFormat,
		IsTrue:       intRangeTypeIsTrue,
		IsIterable:   hook.Const[Value, bool](true),
		Iterator:     intRangeTypeIterator,
		Equal:        intRangeTypeEqual,
		Len:          intRangeTypeLen,
		MethodCall:   intRangeTypeMethodCall,
		Access:       intRangeTypeAccess,
		Contains:     intRangeTypeContains,
		AsBool:       intRangeTypeAsBool,
		AsArray:      intRangeTypeAsArray,
	})

	// RunesIterator
	setValueType(VT_RUNES_ITERATOR, ValueType{
		Name:   hook.Const[Value](runesIteratorTypeName),
		String: runesIteratorTypeString,
		Equal:  runesIteratorTypeEqual,
		Next:   runesIteratorTypeNext,
		Key:    runesIteratorTypeKey,
		Value:  runesIteratorTypeValue,
	})

	// BytesIterator
	setValueType(VT_BYTES_ITERATOR, ValueType{
		Name:   hook.Const[Value](bytesIteratorTypeName),
		String: bytesIteratorTypeString,
		Equal:  bytesIteratorTypeEqual,
		Next:   bytesIteratorTypeNext,
		Key:    bytesIteratorTypeKey,
		Value:  bytesIteratorTypeValue,
	})

	// ArrayIterator
	setValueType(VT_ARRAY_ITERATOR, ValueType{
		Name:   hook.Const[Value](arrayIteratorTypeName),
		String: arrayIteratorTypeString,
		Equal:  arrayIteratorTypeEqual,
		Next:   arrayIteratorTypeNext,
		Key:    arrayIteratorTypeKey,
		Value:  arrayIteratorTypeValue,
	})

	// DictIterator
	setValueType(VT_DICT_ITERATOR, ValueType{
		Name:   hook.Const[Value](dictIteratorTypeName),
		String: dictIteratorTypeString,
		Equal:  dictIteratorTypeEqual,
		Next:   dictIteratorTypeNext,
		Key:    dictIteratorTypeKey,
		Value:  dictIteratorTypeValue,
	})

	// IntRangeIterator
	setValueType(VT_INT_RANGE_ITERATOR, ValueType{
		Name:   hook.Const[Value](intRangeIteratorTypeName),
		String: intRangeIteratorTypeString,
		Equal:  intRangeIteratorTypeEqual,
		Next:   intRangeIteratorTypeNext,
		Key:    intRangeIteratorTypeKey,
		Value:  intRangeIteratorTypeValue,
	})

	// FormatSpec (internal: only ever lives in the constant pool, referenced by OpFormat; never visible on the
	// user-facing value stack)
	setValueType(VT_FORMAT_SPEC, ValueType{
		Name:         hook.Const[Value](formatSpecTypeName),
		String:       formatSpecTypeString,
		EncodeBinary: formatSpecTypeEncodeBinary,
		DecodeBinary: formatSpecTypeDecodeBinary,
		Equal:        formatSpecTypeEqual,
	})
}
