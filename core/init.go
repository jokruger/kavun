package core

func init() {
	// Initialize all types with defaults
	for i := range 256 {
		ValueTypes[i] = ValueTypeDefaults
	}

	// Undefined
	SetValueType(VT_UNDEFINED, ValueType{
		Name:         undefinedTypeName,
		EncodeJSON:   undefinedTypeEncodeJSON,
		EncodeBinary: undefinedTypeEncodeBinary,
		DecodeBinary: undefinedTypeDecodeBinary,
		String:       undefinedTypeString,
		Interface:    undefinedTypeInterface,
		IsIterable:   defaultTrue,
		Equal:        defaultTypeEqualPrimitive,
		Access:       undefinedTypeAccess,
		IsTrue:       defaultFalse, // undefined is always false
		AsBool:       undefinedTypeAsBool,
	})

	// ValuePtr
	SetValueType(VT_VALUE_PTR, ValueType{
		Name: valuePtrTypeName,
	})

	// BuiltinFunction
	SetValueType(VT_BUILTIN_FUNCTION, ValueType{
		Name:         builtinFunctionTypeName,
		EncodeBinary: builtinFunctionTypeEncodeBinary,
		DecodeBinary: builtinFunctionTypeDecodeBinary,
		String:       builtinFunctionTypeString,
		Arity:        builtinFunctionTypeArity,
		IsTrue:       defaultTrue,
		IsCallable:   defaultTrue,
		IsVariadic:   builtinFunctionTypeIsVariadic,
		Equal:        builtinFunctionTypeEqual,
		Call:         builtinFunctionTypeCall,
	})

	// CompiledFunction
	SetValueType(VT_COMPILED_FUNCTION, ValueType{
		Name:         compiledFunctionTypeName,
		EncodeBinary: compiledFunctionTypeEncodeBinary,
		DecodeBinary: compiledFunctionTypeDecodeBinary,
		String:       compiledFunctionTypeString,
		Arity:        compiledFunctionTypeArity,
		IsTrue:       defaultTrue,
		IsCallable:   defaultTrue,
		IsVariadic:   compiledFunctionTypeIsVariadic,
		Equal:        compiledFunctionTypeEqual,
		Call:         compiledFunctionTypeCall,
	})

	// Error
	SetValueType(VT_ERROR, ValueType{
		Name:         errorTypeName,
		EncodeJSON:   errorTypeEncodeJSON,
		EncodeBinary: errorTypeEncodeBinary,
		DecodeBinary: errorTypeDecodeBinary,
		String:       errorTypeString,
		Interface:    errorTypeInterface,
		Equal:        errorTypeEqual,
		Copy:         errorTypeCopy,
		MethodCall:   errorTypeMethodCall,
		IsTrue:       defaultFalse, // error is always false
		AsString:     errorTypeAsString,
		AsBool:       errorTypeAsBool,
	})

	// Bool
	SetValueType(VT_BOOL, ValueType{
		Name:         boolTypeName,
		EncodeJSON:   boolTypeEncodeJSON,
		EncodeBinary: boolTypeEncodeBinary,
		DecodeBinary: boolTypeDecodeBinary,
		String:       boolTypeString,
		Interface:    boolTypeInterface,
		IsTrue:       boolTypeIsTrue,
		AsString:     boolTypeAsString,
		AsInt:        boolTypeAsInt,
		AsBool:       boolTypeAsBool,
		Equal:        boolTypeEqual,
		MethodCall:   boolTypeMethodCall,
		Len:          default1,
	})

	// Char
	SetValueType(VT_CHAR, ValueType{
		Name:         charTypeName,
		EncodeJSON:   charTypeEncodeJSON,
		EncodeBinary: charTypeEncodeBinary,
		DecodeBinary: charTypeDecodeBinary,
		String:       charTypeString,
		Interface:    charTypeInterface,
		IsTrue:       charTypeIsTrue,
		AsString:     charTypeAsString,
		AsInt:        charTypeAsInt,
		AsBool:       charTypeAsBool,
		AsChar:       charTypeAsChar,
		BinaryOp:     charTypeBinaryOp,
		Equal:        charTypeEqual,
		MethodCall:   charTypeMethodCall,
		Len:          default1,
	})

	// Int
	SetValueType(VT_INT, ValueType{
		Name:         intTypeName,
		EncodeJSON:   intTypeEncodeJSON,
		EncodeBinary: intTypeEncodeBinary,
		DecodeBinary: intTypeDecodeBinary,
		String:       intTypeString,
		Interface:    intTypeInterface,
		IsTrue:       intTypeIsTrue,
		AsString:     intTypeAsString,
		AsInt:        intTypeAsInt,
		AsFloat:      intTypeAsFloat,
		AsBool:       intTypeAsBool,
		AsChar:       intTypeAsChar,
		AsTime:       intTypeAsTime,
		BinaryOp:     intTypeBinaryOp,
		Equal:        intTypeEqual,
		MethodCall:   intTypeMethodCall,
		Len:          default1,
	})

	// Float
	SetValueType(VT_FLOAT, ValueType{
		Name:         floatTypeName,
		EncodeJSON:   floatTypeEncodeJSON,
		EncodeBinary: floatTypeEncodeBinary,
		DecodeBinary: floatTypeDecodeBinary,
		String:       floatTypeString,
		Interface:    floatTypeInterface,
		IsTrue:       floatTypeIsTrue,
		AsString:     floatTypeAsString,
		AsInt:        floatTypeAsInt,
		AsFloat:      floatTypeAsFloat,
		AsBool:       floatTypeAsBool,
		BinaryOp:     floatTypeBinaryOp,
		Equal:        floatTypeEqual,
		MethodCall:   floatTypeMethodCall,
		Len:          default1,
	})

	// Time
	SetValueType(VT_TIME, ValueType{
		Name:         timeTypeName,
		EncodeJSON:   timeTypeEncodeJSON,
		EncodeBinary: timeTypeEncodeBinary,
		DecodeBinary: timeTypeDecodeBinary,
		String:       timeTypeString,
		Interface:    timeTypeInterface,
		BinaryOp:     timeTypeBinaryOp,
		Equal:        timeTypeEqual,
		Copy:         timeTypeCopy,
		MethodCall:   timeTypeMethodCall,
		IsTrue:       timeTypeIsTrue,
		AsString:     timeTypeAsString,
		AsInt:        timeTypeAsInt,
		AsBool:       timeTypeAsBool,
		AsTime:       timeTypeAsTime,
		Len:          default1,
	})

	// String
	SetValueType(VT_STRING, ValueType{
		Name:         stringTypeName,
		EncodeJSON:   stringTypeEncodeJSON,
		EncodeBinary: stringTypeEncodeBinary,
		DecodeBinary: stringTypeDecodeBinary,
		String:       stringTypeString,
		Interface:    stringTypeInterface,
		BinaryOp:     stringTypeBinaryOp,
		Equal:        stringTypeEqual,
		Copy:         stringTypeCopy,
		MethodCall:   stringTypeMethodCall,
		Access:       stringTypeAccess,
		IsIterable:   stringTypeIsIterable,
		Iterator:     stringTypeIterator,
		IsTrue:       stringTypeIsTrue,
		AsString:     stringTypeAsString,
		AsInt:        stringTypeAsInt,
		AsFloat:      stringTypeAsFloat,
		AsBool:       stringTypeAsBool,
		AsChar:       stringTypeAsChar,
		AsBytes:      stringTypeAsBytes,
		AsTime:       stringTypeAsTime,
		Contains:     stringTypeContains,
		Len:          stringTypeLen,
	})

	// Bytes
	SetValueType(VT_BYTES, ValueType{
		Name:         bytesTypeName,
		EncodeJSON:   bytesTypeEncodeJSON,
		EncodeBinary: bytesTypeEncodeBinary,
		DecodeBinary: bytesTypeDecodeBinary,
		String:       bytesTypeString,
		Interface:    bytesTypeInterface,
		BinaryOp:     bytesTypeBinaryOp,
		Equal:        bytesTypeEqual,
		Copy:         bytesTypeCopy,
		MethodCall:   bytesTypeMethodCall,
		Access:       bytesTypeAccess,
		IsIterable:   bytesTypeIsIterable,
		Iterator:     bytesTypeIterator,
		IsTrue:       bytesTypeIsTrue,
		AsString:     bytesTypeAsString,
		AsBool:       bytesTypeAsBool,
		AsBytes:      bytesTypeAsBytes,
		Contains:     bytesTypeContains,
		Len:          bytesTypeLen,
	})

	// Array
	SetValueType(VT_ARRAY, ValueType{
		Name:         arrayTypeName,
		EncodeJSON:   arrayTypeEncodeJSON,
		EncodeBinary: arrayTypeEncodeBinary,
		DecodeBinary: arrayTypeDecodeBinary,
		String:       arrayTypeString,
		Interface:    arrayTypeInterface,
		BinaryOp:     arrayTypeBinaryOp,
		Equal:        arrayTypeEqual,
		Copy:         arrayTypeCopy,
		MethodCall:   arrayTypeMethodCall,
		Access:       arrayTypeAccess,
		Assign:       arrayTypeAssign,
		IsIterable:   arrayTypeIsIterable,
		Iterator:     arrayTypeIterator,
		IsImmutable:  arrayTypeIsImmutable,
		IsTrue:       arrayTypeIsTrue,
		AsString:     arrayTypeAsString,
		AsBool:       arrayTypeAsBool,
		AsBytes:      arrayTypeAsBytes,
		Contains:     arrayTypeContains,
		Len:          arrayTypeLen,
		Append:       arrayTypeAppend,
	})

	// Record
	SetValueType(VT_RECORD, ValueType{
		Name:         recordTypeName,
		EncodeJSON:   recordTypeEncodeJSON,
		EncodeBinary: recordTypeEncodeBinary,
		DecodeBinary: recordTypeDecodeBinary,
		String:       recordTypeString,
		Interface:    recordTypeInterface,
		Equal:        recordTypeEqual,
		Copy:         recordTypeCopy,
		MethodCall:   recordTypeMethodCall,
		Access:       recordTypeAccess,
		Assign:       recordTypeAssign,
		IsIterable:   recordTypeIsIterable,
		Iterator:     recordTypeIterator,
		IsImmutable:  recordTypeIsImmutable,
		IsTrue:       recordTypeIsTrue,
		AsString:     recordTypeAsString,
		AsBool:       recordTypeAsBool,
		Contains:     recordTypeContains,
		Len:          recordTypeLen,
		Delete:       recordTypeDelete,
	})

	// Map
	SetValueType(VT_MAP, ValueType{
		Name:         mapTypeName,
		EncodeJSON:   mapTypeEncodeJSON,
		EncodeBinary: mapTypeEncodeBinary,
		DecodeBinary: mapTypeDecodeBinary,
		String:       mapTypeString,
		Interface:    mapTypeInterface,
		Equal:        mapTypeEqual,
		Copy:         mapTypeCopy,
		MethodCall:   mapTypeMethodCall,
		Access:       mapTypeAccess,
		Assign:       mapTypeAssign,
		IsIterable:   mapTypeIsIterable,
		Iterator:     mapTypeIterator,
		IsImmutable:  mapTypeIsImmutable,
		IsTrue:       mapTypeIsTrue,
		AsString:     mapTypeAsString,
		AsBool:       mapTypeAsBool,
		Contains:     mapTypeContains,
		Len:          mapTypeLen,
		Delete:       mapTypeDelete,
	})

	// IntRange
	SetValueType(VT_INT_RANGE, ValueType{
		Name:         intRangeTypeName,
		EncodeBinary: intRangeTypeEncodeBinary,
		DecodeBinary: intRangeTypeDecodeBinary,
		String:       intRangeTypeString,
		Equal:        intRangeTypeEqual,
		Copy:         intRangeTypeCopy,
		MethodCall:   intRangeTypeMethodCall,
		Access:       intRangeTypeAccess,
		IsIterable:   intRangeTypeIsIterable,
		Iterator:     intRangeTypeIterator,
		IsTrue:       intRangeTypeIsTrue,
		AsBool:       intRangeTypeAsBool,
		Contains:     intRangeTypeContains,
		Len:          intRangeTypeLen,
	})

	// StringIterator
	SetValueType(VT_STRING_ITERATOR, ValueType{
		Name:   stringIteratorTypeName,
		String: stringIteratorTypeString,
		Next:   stringIteratorTypeNext,
		Key:    stringIteratorTypeKey,
		Value:  stringIteratorTypeValue,
		Equal:  stringIteratorTypeEqual,
	})

	// BytesIterator
	SetValueType(VT_BYTES_ITERATOR, ValueType{
		Name:   bytesIteratorTypeName,
		String: bytesIteratorTypeString,
		Next:   bytesIteratorTypeNext,
		Key:    bytesIteratorTypeKey,
		Value:  bytesIteratorTypeValue,
		Equal:  bytesIteratorTypeEqual,
	})

	// ArrayIterator
	SetValueType(VT_ARRAY_ITERATOR, ValueType{
		Name:   arrayIteratorTypeName,
		String: arrayIteratorTypeString,
		Next:   arrayIteratorTypeNext,
		Key:    arrayIteratorTypeKey,
		Value:  arrayIteratorTypeValue,
		Equal:  arrayIteratorTypeEqual,
	})

	// MapIterator
	SetValueType(VT_MAP_ITERATOR, ValueType{
		Name:   mapIteratorTypeName,
		String: mapIteratorTypeString,
		Next:   mapIteratorTypeNext,
		Key:    mapIteratorTypeKey,
		Value:  mapIteratorTypeValue,
		Equal:  mapIteratorTypeEqual,
	})

	// IntRangeIterator
	SetValueType(VT_INT_RANGE_ITERATOR, ValueType{
		Name:   intRangeIteratorTypeName,
		String: intRangeIteratorTypeString,
		Next:   intRangeIteratorTypeNext,
		Key:    intRangeIteratorTypeKey,
		Value:  intRangeIteratorTypeValue,
		Equal:  intRangeIteratorTypeEqual,
	})
}
