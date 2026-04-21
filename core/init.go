package core

func init() {
	// Initialize all types with defaults
	for i := range 256 {
		ValueTypes[i] = ValueTypeDefaults
	}

	// Undefined
	SetValueType(VT_UNDEFINED, ValueType{
		Name:         undefinedTypeName,
		String:       undefinedTypeString,
		Interface:    undefinedTypeInterface,
		EncodeJSON:   undefinedTypeEncodeJSON,
		EncodeBinary: undefinedTypeEncodeBinary,
		DecodeBinary: undefinedTypeDecodeBinary,
		IsTrue:       defaultFalse, // undefined is always false
		IsIterable:   defaultTrue,
		Equal:        defaultTypeEqualPrimitive,
		Access:       undefinedTypeAccess,
		AsBool:       undefinedTypeAsBool,
	})

	// ValuePtr
	SetValueType(VT_VALUE_PTR, ValueType{
		Name: valuePtrTypeName,
	})

	// BuiltinFunction
	SetValueType(VT_BUILTIN_FUNCTION, ValueType{
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
	})

	// CompiledFunction
	SetValueType(VT_COMPILED_FUNCTION, ValueType{
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
	})

	// Error
	SetValueType(VT_ERROR, ValueType{
		Name:         errorTypeName,
		String:       errorTypeString,
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
	SetValueType(VT_BOOL, ValueType{
		Name:         boolTypeName,
		String:       boolTypeString,
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
	})

	// Char
	SetValueType(VT_CHAR, ValueType{
		Name:         charTypeName,
		String:       charTypeString,
		Interface:    charTypeInterface,
		EncodeJSON:   charTypeEncodeJSON,
		EncodeBinary: charTypeEncodeBinary,
		DecodeBinary: charTypeDecodeBinary,
		IsTrue:       charTypeIsTrue,
		Equal:        charTypeEqual,
		Len:          default1,
		BinaryOp:     charTypeBinaryOp,
		MethodCall:   charTypeMethodCall,
		AsString:     charTypeAsString,
		AsInt:        charTypeAsInt,
		AsBool:       charTypeAsBool,
		AsChar:       charTypeAsChar,
	})

	// Int
	SetValueType(VT_INT, ValueType{
		Name:         intTypeName,
		String:       intTypeString,
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
		AsChar:       intTypeAsChar,
		AsTime:       intTypeAsTime,
	})

	// Float
	SetValueType(VT_FLOAT, ValueType{
		Name:         floatTypeName,
		String:       floatTypeString,
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
	SetValueType(VT_DECIMAL, ValueType{
		Name:         decimalTypeName,
		String:       decimalTypeString,
		Interface:    decimalTypeInterface,
		EncodeJSON:   decimalTypeEncodeJSON,
		EncodeBinary: decimalTypeEncodeBinary,
		DecodeBinary: decimalTypeDecodeBinary,
		IsTrue:       decimalTypeIsTrue,
		Equal:        decimalTypeEqual,
		Copy:         decimalTypeCopy,
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
	SetValueType(VT_TIME, ValueType{
		Name:         timeTypeName,
		String:       timeTypeString,
		Interface:    timeTypeInterface,
		EncodeJSON:   timeTypeEncodeJSON,
		EncodeBinary: timeTypeEncodeBinary,
		DecodeBinary: timeTypeDecodeBinary,
		IsTrue:       timeTypeIsTrue,
		Equal:        timeTypeEqual,
		Copy:         timeTypeCopy,
		Len:          default1,
		BinaryOp:     timeTypeBinaryOp,
		MethodCall:   timeTypeMethodCall,
		AsString:     timeTypeAsString,
		AsInt:        timeTypeAsInt,
		AsBool:       timeTypeAsBool,
		AsTime:       timeTypeAsTime,
	})

	// String
	SetValueType(VT_STRING, ValueType{
		Name:         stringTypeName,
		String:       stringTypeString,
		Interface:    stringTypeInterface,
		EncodeJSON:   stringTypeEncodeJSON,
		EncodeBinary: stringTypeEncodeBinary,
		DecodeBinary: stringTypeDecodeBinary,
		IsTrue:       stringTypeIsTrue,
		IsIterable:   stringTypeIsIterable,
		Iterator:     stringTypeIterator,
		Equal:        stringTypeEqual,
		Copy:         stringTypeCopy,
		Len:          stringTypeLen,
		BinaryOp:     stringTypeBinaryOp,
		MethodCall:   stringTypeMethodCall,
		Access:       stringTypeAccess,
		Contains:     stringTypeContains,
		Slice:        stringTypeSlice,
		AsBool:       stringTypeAsBool,
		AsChar:       stringTypeAsChar,
		AsInt:        stringTypeAsInt,
		AsFloat:      stringTypeAsFloat,
		AsDecimal:    stringTypeAsDecimal,
		AsTime:       stringTypeAsTime,
		AsString:     stringTypeAsString,
		AsBytes:      stringTypeAsBytes,
		AsArray:      stringTypeAsArray,
	})

	// Bytes
	SetValueType(VT_BYTES, ValueType{
		Name:         bytesTypeName,
		String:       bytesTypeString,
		Interface:    bytesTypeInterface,
		EncodeJSON:   bytesTypeEncodeJSON,
		EncodeBinary: bytesTypeEncodeBinary,
		DecodeBinary: bytesTypeDecodeBinary,
		IsTrue:       bytesTypeIsTrue,
		IsIterable:   bytesTypeIsIterable,
		Iterator:     bytesTypeIterator,
		Equal:        bytesTypeEqual,
		Copy:         bytesTypeCopy,
		Len:          bytesTypeLen,
		BinaryOp:     bytesTypeBinaryOp,
		MethodCall:   bytesTypeMethodCall,
		Access:       bytesTypeAccess,
		Contains:     bytesTypeContains,
		Slice:        bytesTypeSlice,
		AsBool:       bytesTypeAsBool,
		AsString:     bytesTypeAsString,
		AsBytes:      bytesTypeAsBytes,
		AsArray:      bytesTypeAsArray,
	})

	// Array
	SetValueType(VT_ARRAY, ValueType{
		Name:         arrayTypeName,
		String:       arrayTypeString,
		Interface:    arrayTypeInterface,
		EncodeJSON:   arrayTypeEncodeJSON,
		EncodeBinary: arrayTypeEncodeBinary,
		DecodeBinary: arrayTypeDecodeBinary,
		IsTrue:       arrayTypeIsTrue,
		IsIterable:   arrayTypeIsIterable,
		IsImmutable:  arrayTypeIsImmutable,
		Iterator:     arrayTypeIterator,
		Immutable:    arrayTypeImmutable,
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
		AsBool:       arrayTypeAsBool,
		AsString:     arrayTypeAsString,
		AsBytes:      arrayTypeAsBytes,
		AsArray:      arrayTypeAsArray,
	})

	// Record
	SetValueType(VT_RECORD, ValueType{
		Name:         recordTypeName,
		String:       recordTypeString,
		Interface:    recordTypeInterface,
		EncodeJSON:   recordTypeEncodeJSON,
		EncodeBinary: recordTypeEncodeBinary,
		DecodeBinary: recordTypeDecodeBinary,
		IsTrue:       recordTypeIsTrue,
		IsIterable:   recordTypeIsIterable,
		IsImmutable:  recordTypeIsImmutable,
		Iterator:     recordTypeIterator,
		Immutable:    recordTypeImmutable,
		Equal:        recordTypeEqual,
		Copy:         recordTypeCopy,
		Len:          recordTypeLen,
		MethodCall:   recordTypeMethodCall,
		Access:       recordTypeAccess,
		Assign:       recordTypeAssign,
		Contains:     recordTypeContains,
		Delete:       recordTypeDelete,
		AsBool:       recordTypeAsBool,
		AsString:     recordTypeAsString,
		AsMap:        recordTypeAsMap,
	})

	// Map
	SetValueType(VT_MAP, ValueType{
		Name:         mapTypeName,
		String:       mapTypeString,
		Interface:    mapTypeInterface,
		EncodeJSON:   mapTypeEncodeJSON,
		EncodeBinary: mapTypeEncodeBinary,
		DecodeBinary: mapTypeDecodeBinary,
		IsTrue:       mapTypeIsTrue,
		IsIterable:   mapTypeIsIterable,
		IsImmutable:  mapTypeIsImmutable,
		Iterator:     mapTypeIterator,
		Immutable:    mapTypeImmutable,
		Equal:        mapTypeEqual,
		Copy:         mapTypeCopy,
		Len:          mapTypeLen,
		MethodCall:   mapTypeMethodCall,
		Access:       mapTypeAccess,
		Assign:       mapTypeAssign,
		Contains:     mapTypeContains,
		Delete:       mapTypeDelete,
		AsBool:       mapTypeAsBool,
		AsString:     mapTypeAsString,
		AsMap:        mapTypeAsMap,
	})

	// IntRange
	SetValueType(VT_INT_RANGE, ValueType{
		Name:         intRangeTypeName,
		EncodeBinary: intRangeTypeEncodeBinary,
		DecodeBinary: intRangeTypeDecodeBinary,
		String:       intRangeTypeString,
		IsTrue:       intRangeTypeIsTrue,
		IsIterable:   intRangeTypeIsIterable,
		Iterator:     intRangeTypeIterator,
		Equal:        intRangeTypeEqual,
		Copy:         intRangeTypeCopy,
		Len:          intRangeTypeLen,
		MethodCall:   intRangeTypeMethodCall,
		Access:       intRangeTypeAccess,
		Contains:     intRangeTypeContains,
		AsBool:       intRangeTypeAsBool,
		AsArray:      intRangeTypeAsArray,
	})

	// StringIterator
	SetValueType(VT_STRING_ITERATOR, ValueType{
		Name:   stringIteratorTypeName,
		String: stringIteratorTypeString,
		Equal:  stringIteratorTypeEqual,
		Next:   stringIteratorTypeNext,
		Key:    stringIteratorTypeKey,
		Value:  stringIteratorTypeValue,
	})

	// BytesIterator
	SetValueType(VT_BYTES_ITERATOR, ValueType{
		Name:   bytesIteratorTypeName,
		String: bytesIteratorTypeString,
		Equal:  bytesIteratorTypeEqual,
		Next:   bytesIteratorTypeNext,
		Key:    bytesIteratorTypeKey,
		Value:  bytesIteratorTypeValue,
	})

	// ArrayIterator
	SetValueType(VT_ARRAY_ITERATOR, ValueType{
		Name:   arrayIteratorTypeName,
		String: arrayIteratorTypeString,
		Equal:  arrayIteratorTypeEqual,
		Next:   arrayIteratorTypeNext,
		Key:    arrayIteratorTypeKey,
		Value:  arrayIteratorTypeValue,
	})

	// MapIterator
	SetValueType(VT_MAP_ITERATOR, ValueType{
		Name:   mapIteratorTypeName,
		String: mapIteratorTypeString,
		Equal:  mapIteratorTypeEqual,
		Next:   mapIteratorTypeNext,
		Key:    mapIteratorTypeKey,
		Value:  mapIteratorTypeValue,
	})

	// IntRangeIterator
	SetValueType(VT_INT_RANGE_ITERATOR, ValueType{
		Name:   intRangeIteratorTypeName,
		String: intRangeIteratorTypeString,
		Equal:  intRangeIteratorTypeEqual,
		Next:   intRangeIteratorTypeNext,
		Key:    intRangeIteratorTypeKey,
		Value:  intRangeIteratorTypeValue,
	})
}
