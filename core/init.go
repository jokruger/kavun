package core

func init() {
	// Initialize all types with defaults
	for i := range 256 {
		ValueTypes[i] = ValueTypeDefaults
	}

	// Undefined
	setValueType(VT_UNDEFINED, ValueType{
		Name:         undefinedTypeName,
		Interface:    undefinedTypeInterface,
		String:       undefinedTypeString,
		Format:       undefinedTypeFormat,
		EncodeJSON:   undefinedTypeEncodeJSON,
		EncodeBinary: undefinedTypeEncodeBinary,
		DecodeBinary: undefinedTypeDecodeBinary,
		IsTrue:       defaultFalse, // undefined is always false
		IsIterable:   defaultTrue,
		Equal:        defaultTypeEqualPrimitive,
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
		IsTrue:       defaultTrue,
		IsCallable:   defaultTrue,
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
		IsTrue:       defaultTrue,
		IsCallable:   defaultTrue,
		IsVariadic:   compiledFunctionTypeIsVariadic,
		Equal:        compiledFunctionTypeEqual,
		Arity:        compiledFunctionTypeArity,
		Call:         compiledFunctionTypeCall,
		MethodCall:   compiledFunctionTypeMethodCall,
	})

	// Error
	setValueType(VT_ERROR, ValueType{
		Name:         errorTypeName,
		String:       errorTypeString,
		Format:       errorTypeFormat,
		Interface:    errorTypeInterface,
		EncodeJSON:   errorTypeEncodeJSON,
		EncodeBinary: errorTypeEncodeBinary,
		DecodeBinary: errorTypeDecodeBinary,
		IsTrue:       defaultFalse, // error is always false
		Equal:        errorTypeEqual,
		Copy:         errorTypeCopy,
		MethodCall:   errorTypeMethodCall,
		AsString:     errorTypeAsString,
		AsBool:       errorTypeAsBool,
	})

	// Bool
	setValueType(VT_BOOL, ValueType{
		Name:         boolTypeName,
		String:       boolTypeString,
		Format:       boolTypeFormat,
		Interface:    boolTypeInterface,
		EncodeJSON:   boolTypeEncodeJSON,
		EncodeBinary: boolTypeEncodeBinary,
		DecodeBinary: boolTypeDecodeBinary,
		IsTrue:       boolTypeIsTrue,
		Equal:        boolTypeEqual,
		MethodCall:   boolTypeMethodCall,
		Len:          default1,
		AsString:     boolTypeAsString,
		AsInt:        boolTypeAsInt,
		AsBool:       boolTypeAsBool,
		AsByte:       boolTypeAsByte,
	})

	// Byte
	setValueType(VT_BYTE, ValueType{
		Name:         byteTypeName,
		String:       byteTypeString,
		Format:       byteTypeFormat,
		Interface:    byteTypeInterface,
		EncodeJSON:   byteTypeEncodeJSON,
		EncodeBinary: byteTypeEncodeBinary,
		DecodeBinary: byteTypeDecodeBinary,
		IsTrue:       byteTypeIsTrue,
		Equal:        byteTypeEqual,
		Len:          default1,
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
		Name:         runeTypeName,
		String:       runeTypeString,
		Format:       runeTypeFormat,
		Interface:    runeTypeInterface,
		EncodeJSON:   runeTypeEncodeJSON,
		EncodeBinary: runeTypeEncodeBinary,
		DecodeBinary: runeTypeDecodeBinary,
		IsTrue:       runeTypeIsTrue,
		Equal:        runeTypeEqual,
		Len:          default1,
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
		Name:         intTypeName,
		String:       intTypeString,
		Format:       intTypeFormat,
		Interface:    intTypeInterface,
		EncodeJSON:   intTypeEncodeJSON,
		EncodeBinary: intTypeEncodeBinary,
		DecodeBinary: intTypeDecodeBinary,
		IsTrue:       intTypeIsTrue,
		Equal:        intTypeEqual,
		Len:          default1,
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
		Name:         floatTypeName,
		String:       floatTypeString,
		Format:       floatTypeFormat,
		Interface:    floatTypeInterface,
		EncodeJSON:   floatTypeEncodeJSON,
		EncodeBinary: floatTypeEncodeBinary,
		DecodeBinary: floatTypeDecodeBinary,
		IsTrue:       floatTypeIsTrue,
		Equal:        floatTypeEqual,
		Len:          default1,
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
		Name:         decimalTypeName,
		String:       decimalTypeString,
		Format:       decimalTypeFormat,
		Interface:    decimalTypeInterface,
		EncodeJSON:   decimalTypeEncodeJSON,
		EncodeBinary: decimalTypeEncodeBinary,
		DecodeBinary: decimalTypeDecodeBinary,
		IsTrue:       decimalTypeIsTrue,
		Equal:        decimalTypeEqual,
		Len:          default1,
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
		Name:         timeTypeName,
		String:       timeTypeString,
		Format:       timeTypeFormat,
		Interface:    timeTypeInterface,
		EncodeJSON:   timeTypeEncodeJSON,
		EncodeBinary: timeTypeEncodeBinary,
		DecodeBinary: timeTypeDecodeBinary,
		IsTrue:       timeTypeIsTrue,
		Equal:        timeTypeEqual,
		Len:          default1,
		BinaryOp:     timeTypeBinaryOp,
		MethodCall:   timeTypeMethodCall,
		AsString:     timeTypeAsString,
		AsInt:        timeTypeAsInt,
		AsBool:       timeTypeAsBool,
		AsTime:       timeTypeAsTime,
	})

	// String
	setValueType(VT_STRING, ValueType{
		Name:         stringTypeName,
		String:       stringTypeString,
		Format:       stringTypeFormat,
		Interface:    stringTypeInterface,
		EncodeJSON:   stringTypeEncodeJSON,
		EncodeBinary: stringTypeEncodeBinary,
		DecodeBinary: stringTypeDecodeBinary,
		IsTrue:       stringTypeIsTrue,
		IsIterable:   defaultTrue,
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
		Name:         runesTypeName,
		String:       runesTypeString,
		Format:       runesTypeFormat,
		Interface:    runesTypeInterface,
		EncodeJSON:   runesTypeEncodeJSON,
		EncodeBinary: runesTypeEncodeBinary,
		DecodeBinary: runesTypeDecodeBinary,
		IsTrue:       runesTypeIsTrue,
		IsIterable:   defaultTrue,
		Iterator:     runesTypeIterator,
		Equal:        runesTypeEqual,
		Copy:         runesTypeCopy,
		Len:          runesTypeLen,
		BinaryOp:     runesTypeBinaryOp,
		MethodCall:   runesTypeMethodCall,
		Access:       runesTypeAccess,
		Assign:       runesTypeAssign,
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
		Name:         bytesTypeName,
		String:       bytesTypeString,
		Format:       bytesTypeFormat,
		Interface:    bytesTypeInterface,
		EncodeJSON:   bytesTypeEncodeJSON,
		EncodeBinary: bytesTypeEncodeBinary,
		DecodeBinary: bytesTypeDecodeBinary,
		IsTrue:       bytesTypeIsTrue,
		IsIterable:   defaultTrue,
		Iterator:     bytesTypeIterator,
		Equal:        bytesTypeEqual,
		Copy:         bytesTypeCopy,
		Len:          bytesTypeLen,
		BinaryOp:     bytesTypeBinaryOp,
		MethodCall:   bytesTypeMethodCall,
		Access:       bytesTypeAccess,
		Assign:       bytesTypeAssign,
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
		Name:         arrayTypeName,
		String:       arrayTypeString,
		Interface:    arrayTypeInterface,
		EncodeJSON:   arrayTypeEncodeJSON,
		EncodeBinary: arrayTypeEncodeBinary,
		DecodeBinary: arrayTypeDecodeBinary,
		IsTrue:       arrayTypeIsTrue,
		IsIterable:   defaultTrue,
		Iterator:     arrayTypeIterator,
		Equal:        arrayTypeEqual,
		Copy:         arrayTypeCopy,
		Len:          arrayTypeLen,
		BinaryOp:     arrayTypeBinaryOp,
		MethodCall:   arrayTypeMethodCall,
		Access:       arrayTypeAccess,
		Assign:       arrayTypeAssign,
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
		Name:         recordTypeName,
		String:       recordTypeString,
		Interface:    genericDictTypeInterface,
		EncodeJSON:   genericDictTypeEncodeJSON,
		EncodeBinary: genericDictTypeEncodeBinary,
		DecodeBinary: genericDictTypeDecodeBinary,
		IsTrue:       genericDictTypeIsTrue,
		IsIterable:   defaultTrue,
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
		Name:         dictTypeName,
		String:       dictTypeString,
		Interface:    genericDictTypeInterface,
		EncodeJSON:   genericDictTypeEncodeJSON,
		EncodeBinary: genericDictTypeEncodeBinary,
		DecodeBinary: genericDictTypeDecodeBinary,
		IsTrue:       genericDictTypeIsTrue,
		IsIterable:   defaultTrue,
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
		Name:         intRangeTypeName,
		EncodeBinary: intRangeTypeEncodeBinary,
		DecodeBinary: intRangeTypeDecodeBinary,
		String:       intRangeTypeString,
		IsTrue:       intRangeTypeIsTrue,
		IsIterable:   defaultTrue,
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
		Name:   runesIteratorTypeName,
		String: runesIteratorTypeString,
		Equal:  runesIteratorTypeEqual,
		Next:   runesIteratorTypeNext,
		Key:    runesIteratorTypeKey,
		Value:  runesIteratorTypeValue,
	})

	// BytesIterator
	setValueType(VT_BYTES_ITERATOR, ValueType{
		Name:   bytesIteratorTypeName,
		String: bytesIteratorTypeString,
		Equal:  bytesIteratorTypeEqual,
		Next:   bytesIteratorTypeNext,
		Key:    bytesIteratorTypeKey,
		Value:  bytesIteratorTypeValue,
	})

	// ArrayIterator
	setValueType(VT_ARRAY_ITERATOR, ValueType{
		Name:   arrayIteratorTypeName,
		String: arrayIteratorTypeString,
		Equal:  arrayIteratorTypeEqual,
		Next:   arrayIteratorTypeNext,
		Key:    arrayIteratorTypeKey,
		Value:  arrayIteratorTypeValue,
	})

	// DictIterator
	setValueType(VT_DICT_ITERATOR, ValueType{
		Name:   dictIteratorTypeName,
		String: dictIteratorTypeString,
		Equal:  dictIteratorTypeEqual,
		Next:   dictIteratorTypeNext,
		Key:    dictIteratorTypeKey,
		Value:  dictIteratorTypeValue,
	})

	// IntRangeIterator
	setValueType(VT_INT_RANGE_ITERATOR, ValueType{
		Name:   intRangeIteratorTypeName,
		String: intRangeIteratorTypeString,
		Equal:  intRangeIteratorTypeEqual,
		Next:   intRangeIteratorTypeNext,
		Key:    intRangeIteratorTypeKey,
		Value:  intRangeIteratorTypeValue,
	})
}
