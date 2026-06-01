package core

func init() {
	// Initialize all types with defaults
	for i := range 256 {
		ValueTypes[i] = ValueTypeDefaults
	}

	setValueType(VT_UNDEFINED, TypeUndefined)
	setValueType(VT_VALUE_PTR, TypeValuePtr)
	setValueType(VT_BUILTIN_FUNCTION, TypeBuiltinFunction)
	setValueType(VT_BUILTIN_CLOSURE, TypeBuiltinClosure)
	setValueType(VT_COMPILED_FUNCTION, TypeCompiledFunction)
	setValueType(VT_ERROR, TypeError)
	setValueType(VT_BOOL, TypeBool)
	setValueType(VT_BYTE, TypeByte)
	setValueType(VT_RUNE, TypeRune)
	setValueType(VT_INT, TypeInt)
	setValueType(VT_FLOAT, TypeFloat)
	setValueType(VT_DECIMAL, TypeDecimal)
	setValueType(VT_TIME, TypeTime)
	setValueType(VT_STRING, TypeString)
	setValueType(VT_RUNES, TypeRunes)
	setValueType(VT_BYTES, TypeBytes)
	setValueType(VT_ARRAY, TypeArray)
	setValueType(VT_RECORD, TypeRecord)
	setValueType(VT_DICT, TypeDict)
	setValueType(VT_INT_RANGE, TypeIntRange)
	setValueType(VT_RUNES_ITERATOR, TypeRunesIterator)
	setValueType(VT_BYTES_ITERATOR, TypeBytesIterator)
	setValueType(VT_ARRAY_ITERATOR, TypeArrayIterator)
	setValueType(VT_DICT_ITERATOR, TypeDictIterator)
	setValueType(VT_INT_RANGE_ITERATOR, TypeIntRangeIterator)
	setValueType(VT_FORMAT_SPEC, TypeFormatSpec)
}
