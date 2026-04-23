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

	// Rune
	SetValueType(VT_RUNE, ValueType{
		Name:         runeTypeName,
		String:       runeTypeString,
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
		AsRune:       intTypeAsRune,
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
		IsIterable:   defaultTrue,
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
		AsRune:       stringTypeAsRune,
		AsInt:        stringTypeAsInt,
		AsFloat:      stringTypeAsFloat,
		AsDecimal:    stringTypeAsDecimal,
		AsTime:       stringTypeAsTime,
		AsString:     stringTypeAsString,
		AsRunes:      stringTypeAsRunes,
		AsBytes:      stringTypeAsBytes,
		AsArray:      stringTypeAsArray,
	})

	// Runes
	SetValueType(VT_RUNES, ValueType{
		Name:         runesTypeName,
		String:       runesTypeString,
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
		Contains:     runesTypeContains,
		Slice:        runesTypeSlice,
		AsBool:       runesTypeAsBool,
		AsInt:        runesTypeAsInt,
		AsFloat:      runesTypeAsFloat,
		AsDecimal:    runesTypeAsDecimal,
		AsTime:       runesTypeAsTime,
		AsString:     runesTypeAsString,
		AsRunes:      runesTypeAsRunes,
		AsBytes:      runesTypeAsBytes,
		AsArray:      runesTypeAsArray,
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
		IsIterable:   defaultTrue,
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
		String:       genericArrayTypeString,
		Interface:    genericArrayTypeInterface,
		EncodeJSON:   genericArrayTypeEncodeJSON,
		EncodeBinary: genericArrayTypeEncodeBinary,
		DecodeBinary: genericArrayTypeDecodeBinary,
		IsTrue:       genericArrayTypeIsTrue,
		IsIterable:   defaultTrue,
		IsImmutable:  defaultFalse,
		Iterator:     genericArrayTypeIterator,
		Immutable:    arrayTypeImmutable,
		Equal:        genericArrayTypeEqual,
		Copy:         genericArrayTypeCopy,
		Len:          genericArrayTypeLen,
		BinaryOp:     genericArrayTypeBinaryOp,
		MethodCall:   genericArrayTypeMethodCall,
		Access:       genericArrayTypeAccess,
		Assign:       arrayTypeAssign,
		Contains:     genericArrayTypeContains,
		Append:       genericArrayTypeAppend,
		Slice:        genericArrayTypeSlice,
		AsBool:       genericArrayTypeAsBool,
		AsString:     genericArrayTypeAsString,
		AsRunes:      genericArrayTypeAsRunes,
		AsBytes:      genericArrayTypeAsBytes,
		AsArray:      genericArrayTypeAsArray,
	})

	// Immutable Array
	SetValueType(VT_IMMUTABLE_ARRAY, ValueType{
		Name:         immutableArrayTypeName,
		String:       genericArrayTypeString,
		Interface:    genericArrayTypeInterface,
		EncodeJSON:   genericArrayTypeEncodeJSON,
		EncodeBinary: genericArrayTypeEncodeBinary,
		DecodeBinary: genericArrayTypeDecodeBinary,
		IsTrue:       genericArrayTypeIsTrue,
		IsIterable:   defaultTrue,
		IsImmutable:  defaultTrue,
		Iterator:     genericArrayTypeIterator,
		Immutable:    defaultSelf,
		Equal:        genericArrayTypeEqual,
		Copy:         genericArrayTypeCopy,
		Len:          genericArrayTypeLen,
		BinaryOp:     genericArrayTypeBinaryOp,
		MethodCall:   genericArrayTypeMethodCall,
		Access:       genericArrayTypeAccess,
		Contains:     genericArrayTypeContains,
		Append:       genericArrayTypeAppend,
		Slice:        genericArrayTypeSlice,
		AsBool:       genericArrayTypeAsBool,
		AsString:     genericArrayTypeAsString,
		AsRunes:      genericArrayTypeAsRunes,
		AsBytes:      genericArrayTypeAsBytes,
		AsArray:      genericArrayTypeAsArray,
	})

	// Record
	SetValueType(VT_RECORD, ValueType{
		Name:         recordTypeName,
		String:       recordTypeString,
		Interface:    genericMapTypeInterface,
		EncodeJSON:   genericMapTypeEncodeJSON,
		EncodeBinary: genericMapTypeEncodeBinary,
		DecodeBinary: genericMapTypeDecodeBinary,
		IsTrue:       genericMapTypeIsTrue,
		IsIterable:   defaultTrue,
		IsImmutable:  defaultFalse,
		Iterator:     genericMapTypeIterator,
		Immutable:    recordTypeImmutable,
		Equal:        genericMapTypeEqual,
		Copy:         recordTypeCopy,
		Len:          genericMapTypeLen,
		MethodCall:   recordTypeMethodCall,
		Access:       recordTypeAccess,
		Assign:       genericMapTypeAssign,
		Contains:     genericMapTypeContains,
		Delete:       genericMapTypeDelete,
		AsBool:       genericMapTypeAsBool,
		AsString:     genericMapTypeAsString,
		AsMap:        genericMapTypeAsMap,
	})

	// Immutable Record
	SetValueType(VT_IMMUTABLE_RECORD, ValueType{
		Name:         immutableRecordTypeName,
		String:       recordTypeString,
		Interface:    genericMapTypeInterface,
		EncodeJSON:   genericMapTypeEncodeJSON,
		EncodeBinary: genericMapTypeEncodeBinary,
		DecodeBinary: genericMapTypeDecodeBinary,
		IsTrue:       genericMapTypeIsTrue,
		IsIterable:   defaultTrue,
		IsImmutable:  defaultTrue,
		Iterator:     genericMapTypeIterator,
		Immutable:    defaultSelf,
		Equal:        genericMapTypeEqual,
		Copy:         recordTypeCopy,
		Len:          genericMapTypeLen,
		MethodCall:   recordTypeMethodCall,
		Access:       recordTypeAccess,
		Contains:     genericMapTypeContains,
		AsBool:       genericMapTypeAsBool,
		AsString:     genericMapTypeAsString,
		AsMap:        genericMapTypeAsMap,
	})

	// Map
	SetValueType(VT_MAP, ValueType{
		Name:         mapTypeName,
		String:       mapTypeString,
		Interface:    genericMapTypeInterface,
		EncodeJSON:   genericMapTypeEncodeJSON,
		EncodeBinary: genericMapTypeEncodeBinary,
		DecodeBinary: genericMapTypeDecodeBinary,
		IsTrue:       genericMapTypeIsTrue,
		IsIterable:   defaultTrue,
		IsImmutable:  defaultFalse,
		Iterator:     genericMapTypeIterator,
		Immutable:    mapTypeImmutable,
		Equal:        genericMapTypeEqual,
		Copy:         mapTypeCopy,
		Len:          genericMapTypeLen,
		MethodCall:   mapTypeMethodCall,
		Access:       mapTypeAccess,
		Assign:       genericMapTypeAssign,
		Contains:     genericMapTypeContains,
		Delete:       genericMapTypeDelete,
		AsBool:       genericMapTypeAsBool,
		AsString:     genericMapTypeAsString,
		AsMap:        genericMapTypeAsMap,
	})

	// Immutable Map
	SetValueType(VT_IMMUTABLE_MAP, ValueType{
		Name:         immutableMapTypeName,
		String:       mapTypeString,
		Interface:    genericMapTypeInterface,
		EncodeJSON:   genericMapTypeEncodeJSON,
		EncodeBinary: genericMapTypeEncodeBinary,
		DecodeBinary: genericMapTypeDecodeBinary,
		IsTrue:       genericMapTypeIsTrue,
		IsIterable:   defaultTrue,
		IsImmutable:  defaultTrue,
		Iterator:     genericMapTypeIterator,
		Immutable:    defaultSelf,
		Equal:        genericMapTypeEqual,
		Copy:         mapTypeCopy,
		Len:          genericMapTypeLen,
		MethodCall:   mapTypeMethodCall,
		Access:       mapTypeAccess,
		Contains:     genericMapTypeContains,
		AsBool:       genericMapTypeAsBool,
		AsString:     genericMapTypeAsString,
		AsMap:        genericMapTypeAsMap,
	})

	// IntRange
	SetValueType(VT_INT_RANGE, ValueType{
		Name:         intRangeTypeName,
		EncodeBinary: intRangeTypeEncodeBinary,
		DecodeBinary: intRangeTypeDecodeBinary,
		String:       intRangeTypeString,
		IsTrue:       intRangeTypeIsTrue,
		IsIterable:   defaultTrue,
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

	// RunesIterator
	SetValueType(VT_RUNES_ITERATOR, ValueType{
		Name:   runesIteratorTypeName,
		String: runesIteratorTypeString,
		Equal:  runesIteratorTypeEqual,
		Next:   runesIteratorTypeNext,
		Key:    runesIteratorTypeKey,
		Value:  runesIteratorTypeValue,
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
