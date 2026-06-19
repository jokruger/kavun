package core

import (
	"github.com/jokruger/kavun/core/value"
)

func init() {
	// Initialize all types with defaults
	for i := range 256 {
		ValueTypes[i] = DefaultValueType
	}

	setValueType(value.Undefined, TypeUndefined)
	setValueType(value.ValuePtr, TypeValuePtr)
	setValueType(value.BuiltinFunction, TypeBuiltinFunction)
	setValueType(value.BuiltinClosure, TypeBuiltinClosure)
	setValueType(value.CompiledFunction, TypeCompiledFunction)
	setValueType(value.Error, TypeError)
	setValueType(value.Bool, TypeBool)
	setValueType(value.Byte, TypeByte)
	setValueType(value.Rune, TypeRune)
	setValueType(value.Int, TypeInt)
	setValueType(value.Float, TypeFloat)
	setValueType(value.Decimal, TypeDecimal)
	setValueType(value.Time, TypeTime)
	setValueType(value.String, TypeString)
	setValueType(value.Runes, TypeRunes)
	setValueType(value.Bytes, TypeBytes)
	setValueType(value.Array, TypeArray)
	setValueType(value.Record, TypeRecord)
	setValueType(value.Dict, TypeDict)
	setValueType(value.IntRange, TypeIntRange)
	setValueType(value.RunesIterator, TypeRunesIterator)
	setValueType(value.BytesIterator, TypeBytesIterator)
	setValueType(value.ArrayIterator, TypeArrayIterator)
	setValueType(value.DictIterator, TypeDictIterator)
	setValueType(value.IntRangeIterator, TypeIntRangeIterator)
	setValueType(value.FormatSpec, TypeFormatSpec)
}
