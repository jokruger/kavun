package core

func init() {
	// Initialize all types with defaults
	for i := range 256 {
		ValueTypes[i] = ValueTypeDefaults
	}

	// Undefined
	SetValueType(VT_UNDEFINED, ValueType{
		TypeName:         undefinedTypeName,
		TypeEncodeJSON:   undefinedTypeEncodeJSON,
		TypeEncodeBinary: undefinedTypeEncodeBinary,
		TypeDecodeBinary: undefinedTypeDecodeBinary,
		TypeString:       undefinedTypeString,
		TypeInterface:    undefinedTypeInterface,
		TypeIsIterable:   defaultTrue,
		TypeEqual:        defaultTypeEqualPrimitive,
		TypeAccess:       undefinedTypeAccess,
		TypeIsTrue:       defaultFalse, // undefined is always false
		TypeAsBool:       undefinedTypeAsBool,
	})

	// ValuePtr
	SetValueType(VT_VALUE_PTR, ValueType{
		TypeName: valuePtrTypeName,
	})

	// BuiltinFunction
	SetValueType(VT_BUILTIN_FUNCTION, ValueType{
		TypeName:         builtinFunctionTypeName,
		TypeEncodeBinary: builtinFunctionTypeEncodeBinary,
		TypeDecodeBinary: builtinFunctionTypeDecodeBinary,
		TypeString:       builtinFunctionTypeString,
		TypeArity:        builtinFunctionTypeArity,
		TypeIsTrue:       defaultTrue,
		TypeIsCallable:   defaultTrue,
		TypeIsVariadic:   builtinFunctionTypeIsVariadic,
		TypeEqual:        builtinFunctionTypeEqual,
		TypeCall:         builtinFunctionTypeCall,
	})

	// CompiledFunction
	SetValueType(VT_COMPILED_FUNCTION, ValueType{
		TypeName:         compiledFunctionTypeName,
		TypeEncodeBinary: compiledFunctionTypeEncodeBinary,
		TypeDecodeBinary: compiledFunctionTypeDecodeBinary,
		TypeString:       compiledFunctionTypeString,
		TypeArity:        compiledFunctionTypeArity,
		TypeIsTrue:       defaultTrue,
		TypeIsCallable:   defaultTrue,
		TypeIsVariadic:   compiledFunctionTypeIsVariadic,
		TypeEqual:        compiledFunctionTypeEqual,
		TypeCall:         compiledFunctionTypeCall,
	})

	// Error
	SetValueType(VT_ERROR, ValueType{
		TypeName:         errorTypeName,
		TypeEncodeJSON:   errorTypeEncodeJSON,
		TypeEncodeBinary: errorTypeEncodeBinary,
		TypeDecodeBinary: errorTypeDecodeBinary,
		TypeString:       errorTypeString,
		TypeInterface:    errorTypeInterface,
		TypeEqual:        errorTypeEqual,
		TypeCopy:         errorTypeCopy,
		TypeMethodCall:   errorTypeMethodCall,
		TypeIsTrue:       defaultFalse, // error is always false
		TypeAsString:     errorTypeAsString,
		TypeAsBool:       errorTypeAsBool,
	})

	// Bool
	SetValueType(VT_BOOL, ValueType{
		TypeName:         boolTypeName,
		TypeEncodeJSON:   boolTypeEncodeJSON,
		TypeEncodeBinary: boolTypeEncodeBinary,
		TypeDecodeBinary: boolTypeDecodeBinary,
		TypeString:       boolTypeString,
		TypeInterface:    boolTypeInterface,
		TypeIsTrue:       boolTypeIsTrue,
		TypeAsString:     boolTypeAsString,
		TypeAsInt:        boolTypeAsInt,
		TypeAsBool:       boolTypeAsBool,
		TypeEqual:        boolTypeEqual,
		TypeMethodCall:   boolTypeMethodCall,
		TypeLen:          default1,
	})

	// Char
	SetValueType(VT_CHAR, ValueType{
		TypeName:         charTypeName,
		TypeEncodeJSON:   charTypeEncodeJSON,
		TypeEncodeBinary: charTypeEncodeBinary,
		TypeDecodeBinary: charTypeDecodeBinary,
		TypeString:       charTypeString,
		TypeInterface:    charTypeInterface,
		TypeIsTrue:       charTypeIsTrue,
		TypeAsString:     charTypeAsString,
		TypeAsInt:        charTypeAsInt,
		TypeAsBool:       charTypeAsBool,
		TypeAsChar:       charTypeAsChar,
		TypeBinaryOp:     charTypeBinaryOp,
		TypeEqual:        charTypeEqual,
		TypeMethodCall:   charTypeMethodCall,
		TypeLen:          default1,
	})

	// Int
	SetValueType(VT_INT, ValueType{
		TypeName:         intTypeName,
		TypeEncodeJSON:   intTypeEncodeJSON,
		TypeEncodeBinary: intTypeEncodeBinary,
		TypeDecodeBinary: intTypeDecodeBinary,
		TypeString:       intTypeString,
		TypeInterface:    intTypeInterface,
		TypeIsTrue:       intTypeIsTrue,
		TypeAsString:     intTypeAsString,
		TypeAsInt:        intTypeAsInt,
		TypeAsFloat:      intTypeAsFloat,
		TypeAsBool:       intTypeAsBool,
		TypeAsChar:       intTypeAsChar,
		TypeAsTime:       intTypeAsTime,
		TypeBinaryOp:     intTypeBinaryOp,
		TypeEqual:        intTypeEqual,
		TypeMethodCall:   intTypeMethodCall,
		TypeLen:          default1,
	})

	// Float
	SetValueType(VT_FLOAT, ValueType{
		TypeName:         floatTypeName,
		TypeEncodeJSON:   floatTypeEncodeJSON,
		TypeEncodeBinary: floatTypeEncodeBinary,
		TypeDecodeBinary: floatTypeDecodeBinary,
		TypeString:       floatTypeString,
		TypeInterface:    floatTypeInterface,
		TypeIsTrue:       floatTypeIsTrue,
		TypeAsString:     floatTypeAsString,
		TypeAsInt:        floatTypeAsInt,
		TypeAsFloat:      floatTypeAsFloat,
		TypeAsBool:       floatTypeAsBool,
		TypeBinaryOp:     floatTypeBinaryOp,
		TypeEqual:        floatTypeEqual,
		TypeMethodCall:   floatTypeMethodCall,
		TypeLen:          default1,
	})

	// Time
	SetValueType(VT_TIME, ValueType{
		TypeName:         timeTypeName,
		TypeEncodeJSON:   timeTypeEncodeJSON,
		TypeEncodeBinary: timeTypeEncodeBinary,
		TypeDecodeBinary: timeTypeDecodeBinary,
		TypeString:       timeTypeString,
		TypeInterface:    timeTypeInterface,
		TypeBinaryOp:     timeTypeBinaryOp,
		TypeEqual:        timeTypeEqual,
		TypeCopy:         timeTypeCopy,
		TypeMethodCall:   timeTypeMethodCall,
		TypeIsTrue:       timeTypeIsTrue,
		TypeAsString:     timeTypeAsString,
		TypeAsInt:        timeTypeAsInt,
		TypeAsBool:       timeTypeAsBool,
		TypeAsTime:       timeTypeAsTime,
		TypeLen:          default1,
	})

	// String
	SetValueType(VT_STRING, ValueType{
		TypeName:         stringTypeName,
		TypeEncodeJSON:   stringTypeEncodeJSON,
		TypeEncodeBinary: stringTypeEncodeBinary,
		TypeDecodeBinary: stringTypeDecodeBinary,
		TypeString:       stringTypeString,
		TypeInterface:    stringTypeInterface,
		TypeBinaryOp:     stringTypeBinaryOp,
		TypeEqual:        stringTypeEqual,
		TypeCopy:         stringTypeCopy,
		TypeMethodCall:   stringTypeMethodCall,
		TypeAccess:       stringTypeAccess,
		TypeIsIterable:   stringTypeIsIterable,
		TypeIterator:     stringTypeIterator,
		TypeIsTrue:       stringTypeIsTrue,
		TypeAsString:     stringTypeAsString,
		TypeAsInt:        stringTypeAsInt,
		TypeAsFloat:      stringTypeAsFloat,
		TypeAsBool:       stringTypeAsBool,
		TypeAsChar:       stringTypeAsChar,
		TypeAsBytes:      stringTypeAsBytes,
		TypeAsTime:       stringTypeAsTime,
		TypeContains:     stringTypeContains,
		TypeLen:          stringTypeLen,
	})

	// Bytes
	SetValueType(VT_BYTES, ValueType{
		TypeName:         bytesTypeName,
		TypeEncodeJSON:   bytesTypeEncodeJSON,
		TypeEncodeBinary: bytesTypeEncodeBinary,
		TypeDecodeBinary: bytesTypeDecodeBinary,
		TypeString:       bytesTypeString,
		TypeInterface:    bytesTypeInterface,
		TypeBinaryOp:     bytesTypeBinaryOp,
		TypeEqual:        bytesTypeEqual,
		TypeCopy:         bytesTypeCopy,
		TypeMethodCall:   bytesTypeMethodCall,
		TypeAccess:       bytesTypeAccess,
		TypeIsIterable:   bytesTypeIsIterable,
		TypeIterator:     bytesTypeIterator,
		TypeIsTrue:       bytesTypeIsTrue,
		TypeAsString:     bytesTypeAsString,
		TypeAsBool:       bytesTypeAsBool,
		TypeAsBytes:      bytesTypeAsBytes,
		TypeContains:     bytesTypeContains,
		TypeLen:          bytesTypeLen,
	})

	// Array
	SetValueType(VT_ARRAY, ValueType{
		TypeName:         arrayTypeName,
		TypeEncodeJSON:   arrayTypeEncodeJSON,
		TypeEncodeBinary: arrayTypeEncodeBinary,
		TypeDecodeBinary: arrayTypeDecodeBinary,
		TypeString:       arrayTypeString,
		TypeInterface:    arrayTypeInterface,
		TypeBinaryOp:     arrayTypeBinaryOp,
		TypeEqual:        arrayTypeEqual,
		TypeCopy:         arrayTypeCopy,
		TypeMethodCall:   arrayTypeMethodCall,
		TypeAccess:       arrayTypeAccess,
		TypeAssign:       arrayTypeAssign,
		TypeIsIterable:   arrayTypeIsIterable,
		TypeIterator:     arrayTypeIterator,
		TypeIsImmutable:  arrayTypeIsImmutable,
		TypeIsTrue:       arrayTypeIsTrue,
		TypeAsString:     arrayTypeAsString,
		TypeAsBool:       arrayTypeAsBool,
		TypeAsBytes:      arrayTypeAsBytes,
		TypeContains:     arrayTypeContains,
		TypeLen:          arrayTypeLen,
		TypeAppend:       arrayTypeAppend,
	})

	// Record
	SetValueType(VT_RECORD, ValueType{
		TypeName:         recordTypeName,
		TypeEncodeJSON:   recordTypeEncodeJSON,
		TypeEncodeBinary: recordTypeEncodeBinary,
		TypeDecodeBinary: recordTypeDecodeBinary,
		TypeString:       recordTypeString,
		TypeInterface:    recordTypeInterface,
		TypeEqual:        recordTypeEqual,
		TypeCopy:         recordTypeCopy,
		TypeMethodCall:   recordTypeMethodCall,
		TypeAccess:       recordTypeAccess,
		TypeAssign:       recordTypeAssign,
		TypeIsIterable:   recordTypeIsIterable,
		TypeIterator:     recordTypeIterator,
		TypeIsImmutable:  recordTypeIsImmutable,
		TypeIsTrue:       recordTypeIsTrue,
		TypeAsString:     recordTypeAsString,
		TypeAsBool:       recordTypeAsBool,
		TypeContains:     recordTypeContains,
		TypeLen:          recordTypeLen,
		TypeDelete:       recordTypeDelete,
	})

	// Map
	SetValueType(VT_MAP, ValueType{
		TypeName:         mapTypeName,
		TypeEncodeJSON:   mapTypeEncodeJSON,
		TypeEncodeBinary: mapTypeEncodeBinary,
		TypeDecodeBinary: mapTypeDecodeBinary,
		TypeString:       mapTypeString,
		TypeInterface:    mapTypeInterface,
		TypeEqual:        mapTypeEqual,
		TypeCopy:         mapTypeCopy,
		TypeMethodCall:   mapTypeMethodCall,
		TypeAccess:       mapTypeAccess,
		TypeAssign:       mapTypeAssign,
		TypeIsIterable:   mapTypeIsIterable,
		TypeIterator:     mapTypeIterator,
		TypeIsImmutable:  mapTypeIsImmutable,
		TypeIsTrue:       mapTypeIsTrue,
		TypeAsString:     mapTypeAsString,
		TypeAsBool:       mapTypeAsBool,
		TypeContains:     mapTypeContains,
		TypeLen:          mapTypeLen,
		TypeDelete:       mapTypeDelete,
	})

	// IntRange
	SetValueType(VT_INT_RANGE, ValueType{
		TypeName:         intRangeTypeName,
		TypeEncodeBinary: intRangeTypeEncodeBinary,
		TypeDecodeBinary: intRangeTypeDecodeBinary,
		TypeString:       intRangeTypeString,
		TypeEqual:        intRangeTypeEqual,
		TypeCopy:         intRangeTypeCopy,
		TypeMethodCall:   intRangeTypeMethodCall,
		TypeAccess:       intRangeTypeAccess,
		TypeIsIterable:   intRangeTypeIsIterable,
		TypeIterator:     intRangeTypeIterator,
		TypeIsTrue:       intRangeTypeIsTrue,
		TypeAsBool:       intRangeTypeAsBool,
		TypeContains:     intRangeTypeContains,
		TypeLen:          intRangeTypeLen,
	})

	// StringIterator
	SetValueType(VT_STRING_ITERATOR, ValueType{
		TypeName:   stringIteratorTypeName,
		TypeString: stringIteratorTypeString,
		TypeNext:   stringIteratorTypeNext,
		TypeKey:    stringIteratorTypeKey,
		TypeValue:  stringIteratorTypeValue,
		TypeEqual:  stringIteratorTypeEqual,
	})

	// BytesIterator
	SetValueType(VT_BYTES_ITERATOR, ValueType{
		TypeName:   bytesIteratorTypeName,
		TypeString: bytesIteratorTypeString,
		TypeNext:   bytesIteratorTypeNext,
		TypeKey:    bytesIteratorTypeKey,
		TypeValue:  bytesIteratorTypeValue,
		TypeEqual:  bytesIteratorTypeEqual,
	})

	// ArrayIterator
	SetValueType(VT_ARRAY_ITERATOR, ValueType{
		TypeName:   arrayIteratorTypeName,
		TypeString: arrayIteratorTypeString,
		TypeNext:   arrayIteratorTypeNext,
		TypeKey:    arrayIteratorTypeKey,
		TypeValue:  arrayIteratorTypeValue,
		TypeEqual:  arrayIteratorTypeEqual,
	})

	// MapIterator
	SetValueType(VT_MAP_ITERATOR, ValueType{
		TypeName:   mapIteratorTypeName,
		TypeString: mapIteratorTypeString,
		TypeNext:   mapIteratorTypeNext,
		TypeKey:    mapIteratorTypeKey,
		TypeValue:  mapIteratorTypeValue,
		TypeEqual:  mapIteratorTypeEqual,
	})

	// IntRangeIterator
	SetValueType(VT_INT_RANGE_ITERATOR, ValueType{
		TypeName:   intRangeIteratorTypeName,
		TypeString: intRangeIteratorTypeString,
		TypeNext:   intRangeIteratorTypeNext,
		TypeKey:    intRangeIteratorTypeKey,
		TypeValue:  intRangeIteratorTypeValue,
		TypeEqual:  intRangeIteratorTypeEqual,
	})
}
