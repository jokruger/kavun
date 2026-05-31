package value

import (
	"math"
	"testing"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/token"
	_ "github.com/jokruger/kavun/vm"
)

func TestObject_Value(t *testing.T) {
	var v core.Value
	var x core.Value
	var s string
	var bs []byte
	var err error
	var i int64
	var ok bool

	// Undefined
	v = core.Undefined
	require.True(t, v.Type == core.VT_UNDEFINED)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_UNDEFINED)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Bool
	v = core.True
	require.True(t, v.Type == core.VT_BOOL)
	require.Equal(t, alloc, true, v.Data != 0)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BOOL)
	require.Equal(t, alloc, true, x.Data != 0)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.False
	require.True(t, v.Type == core.VT_BOOL)
	require.Equal(t, alloc, false, v.Data != 0)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BOOL)
	require.Equal(t, alloc, false, x.Data != 0)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Byte
	v = core.ByteValue(123)
	require.True(t, v.Type == core.VT_BYTE)
	require.Equal(t, alloc, byte(123), byte(v.Data))
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BYTE)
	require.Equal(t, alloc, byte(123), byte(x.Data))
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Rune
	v = core.RuneValue('A')
	require.True(t, v.Type == core.VT_RUNE)
	require.Equal(t, alloc, 'A', rune(v.Data))
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNE)
	require.Equal(t, alloc, 'A', rune(x.Data))
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.RuneValue('₴')
	require.True(t, v.Type == core.VT_RUNE)
	require.Equal(t, alloc, '₴', rune(v.Data))
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNE)
	require.Equal(t, alloc, '₴', rune(x.Data))
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Int
	v = core.IntValue(123)
	require.True(t, v.Type == core.VT_INT)
	require.Equal(t, alloc, int64(123), int64(v.Data))
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_INT)
	require.Equal(t, alloc, int64(123), int64(x.Data))
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.IntValue(-456)
	require.True(t, v.Type == core.VT_INT)
	require.Equal(t, alloc, int64(-456), int64(v.Data))
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_INT)
	require.Equal(t, alloc, int64(-456), int64(x.Data))
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Float
	v = core.FloatValue(3.14)
	require.True(t, v.Type == core.VT_FLOAT)
	require.Equal(t, alloc, 3.14, math.Float64frombits(v.Data))
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_FLOAT)
	require.Equal(t, alloc, 3.14, math.Float64frombits(x.Data))
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.FloatValue(-2.71828)
	require.True(t, v.Type == core.VT_FLOAT)
	require.Equal(t, alloc, -2.71828, math.Float64frombits(v.Data))
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_FLOAT)
	require.Equal(t, alloc, -2.71828, math.Float64frombits(x.Data))
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Decimal
	v = core.NewDecimalValue(dec128.FromString("3.14"))
	require.True(t, v.Type == core.VT_DECIMAL)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_DECIMAL)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// String
	v = core.NewStringValue("")
	require.True(t, v.Type == core.VT_STRING)
	s, _ = v.AsString(alloc)
	require.Equal(t, alloc, "", s)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_STRING)
	s, _ = x.AsString(alloc)
	require.Equal(t, alloc, "", s)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.NewStringValue("hello")
	require.True(t, v.Type == core.VT_STRING)
	s, _ = v.AsString(alloc)
	require.Equal(t, alloc, "hello", s)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_STRING)
	s, _ = x.AsString(alloc)
	require.Equal(t, alloc, "hello", s)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Runes
	v = core.NewRunesValue([]rune(""), false)
	require.True(t, v.Type == core.VT_RUNES)
	s, _ = v.AsString(alloc)
	require.Equal(t, alloc, "", s)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNES)
	s, _ = x.AsString(alloc)
	require.Equal(t, alloc, "", s)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.NewRunesValue([]rune("путін хуйло"), false)
	require.True(t, v.Type == core.VT_RUNES)
	s, _ = v.AsString(alloc)
	require.Equal(t, alloc, "путін хуйло", s)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNES)
	s, _ = x.AsString(alloc)
	require.Equal(t, alloc, "путін хуйло", s)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Bytes
	v = core.NewBytesValue([]byte{}, false)
	require.True(t, v.Type == core.VT_BYTES)
	b, _ := v.AsBytes(alloc)
	require.Equal(t, alloc, []byte{}, b)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BYTES)
	b, _ = x.AsBytes(alloc)
	require.Equal(t, alloc, []byte{}, b)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.NewBytesValue([]byte("foo"), false)
	require.True(t, v.Type == core.VT_BYTES)
	b, _ = v.AsBytes(alloc)
	require.Equal(t, alloc, []byte("foo"), b)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BYTES)
	b, _ = x.AsBytes(alloc)
	require.Equal(t, alloc, []byte("foo"), b)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Array
	v = core.NewArrayValue([]core.Value{}, false)
	require.True(t, v.Type == core.VT_ARRAY)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ARRAY)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.NewArrayValue([]core.Value{}, true)
	require.True(t, v.Type == core.VT_ARRAY)
	require.True(t, v.Immutable)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ARRAY)
	require.True(t, x.Immutable)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	require.True(t, v.Type == core.VT_ARRAY)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ARRAY)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Record
	v = core.NewRecordValue(map[string]core.Value{}, true)
	require.True(t, v.Type == core.VT_RECORD)
	require.True(t, v.Immutable)
	require.True(t, v.Immutable)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RECORD)
	require.True(t, x.Immutable)
	require.True(t, x.Immutable)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.Type == core.VT_RECORD)
	require.False(t, v.Immutable)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RECORD)
	require.False(t, x.Immutable)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Map
	v = core.NewDictValue(map[string]core.Value{}, true)
	require.True(t, v.Type == core.VT_DICT)
	require.True(t, v.Immutable)
	require.True(t, v.Immutable)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_DICT)
	require.True(t, x.Immutable)
	require.True(t, x.Immutable)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.NewDictValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.Type == core.VT_DICT)
	require.False(t, v.Immutable)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_DICT)
	require.False(t, x.Immutable)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Error
	v = core.NewErrorValue(core.Undefined)
	require.True(t, v.Type == core.VT_ERROR)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ERROR)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	v = core.NewErrorValue(core.NewStringValue("some error"))
	require.True(t, v.Type == core.VT_ERROR)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ERROR)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// Time
	v = core.NewTimeValue(time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC))
	require.True(t, v.Type == core.VT_TIME)
	tm, _ := v.AsTime(alloc)
	require.Equal(t, alloc, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_TIME)
	tm, _ = x.AsTime(alloc)
	require.Equal(t, alloc, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	require.Equal(t, alloc, true, v.Equal(alloc, x))

	// IntRange
	v = core.NewIntRangeValue(0, 0, 1)
	require.True(t, v.Type == core.VT_INT_RANGE)
	rng := (*core.IntRange)(v.Ptr)
	require.True(t, rng.Empty())
	require.Equal(t, alloc, int64(0), rng.Len())
	v = core.NewIntRangeValue(0, 10, 1)
	rng = (*core.IntRange)(v.Ptr)
	require.False(t, rng.Empty())
	require.Equal(t, alloc, int64(10), rng.Len())
	i, ok = rng.Get(0)
	require.True(t, ok)
	require.Equal(t, alloc, int64(0), i)
	i, ok = rng.Get(9)
	require.True(t, ok)
	require.Equal(t, alloc, int64(9), i)
	i, ok = rng.Get(10)
	require.False(t, ok)
	v = core.NewIntRangeValue(10, 0, 1)
	rng = (*core.IntRange)(v.Ptr)
	require.False(t, rng.Empty())
	require.Equal(t, alloc, int64(10), rng.Len())
	i, ok = rng.Get(0)
	require.True(t, ok)
	require.Equal(t, alloc, int64(10), i)
	i, ok = rng.Get(9)
	require.True(t, ok)
	require.Equal(t, alloc, int64(1), i)
	i, ok = rng.Get(10)
	require.False(t, ok)

	v = core.NewIntRangeValue(0, 10, 2)
	require.True(t, v.Type == core.VT_INT_RANGE)
	rng = (*core.IntRange)(v.Ptr)
	require.Equal(t, alloc, int64(0), rng.Start)
	require.Equal(t, alloc, int64(10), rng.Stop)
	require.Equal(t, alloc, int64(2), rng.Step)
	bs, err = v.EncodeBinary(alloc)
	require.NoError(t, err)
	err = x.DecodeBinary(alloc, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_INT_RANGE)
	rng = (*core.IntRange)(x.Ptr)
	require.Equal(t, alloc, int64(0), rng.Start)
	require.Equal(t, alloc, int64(10), rng.Stop)
	require.Equal(t, alloc, int64(2), rng.Step)
	require.Equal(t, alloc, true, v.Equal(alloc, x))
}

func TestObject_TypeName(t *testing.T) {
	var o core.Value

	o = core.IntValue(0)
	require.Equal(t, alloc, "int", o.TypeName(alloc))

	o = core.FloatValue(0)
	require.Equal(t, alloc, "float", o.TypeName(alloc))

	o = core.ByteValue(0)
	require.Equal(t, alloc, "byte", o.TypeName(alloc))

	o = core.RuneValue(0)
	require.Equal(t, alloc, "rune", o.TypeName(alloc))

	o = core.NewStringValue("")
	require.Equal(t, alloc, "string", o.TypeName(alloc))

	o = core.False
	require.Equal(t, alloc, "bool", o.TypeName(alloc))

	o = core.NewArrayValue(nil, false)
	require.Equal(t, alloc, "array", o.TypeName(alloc))

	o = core.NewArrayValue(nil, true)
	require.Equal(t, alloc, "immutable-array", o.TypeName(alloc))

	o = core.NewRecordValue(nil, false)
	require.Equal(t, alloc, "record", o.TypeName(alloc))

	o = core.NewRecordValue(nil, true)
	require.Equal(t, alloc, "immutable-record", o.TypeName(alloc))

	o = core.NewDictValue(nil, false)
	require.Equal(t, alloc, "dict", o.TypeName(alloc))

	o = core.NewDictValue(nil, true)
	require.Equal(t, alloc, "immutable-dict", o.TypeName(alloc))

	o = core.NewBuiltinFunctionValue("fn", nil, 0, false)
	require.Equal(t, alloc, "<builtin-function:fn/0>", o.TypeName(alloc))

	o = core.Undefined
	require.Equal(t, alloc, "undefined", o.TypeName(alloc))

	o = core.NewErrorValue(core.Undefined)
	require.Equal(t, alloc, "error", o.TypeName(alloc))

	o = core.NewBytesValue(nil, false)
	require.Equal(t, alloc, "bytes", o.TypeName(alloc))

	o = core.NewIntRangeValue(1, 10, 1)
	require.Equal(t, alloc, "range", o.TypeName(alloc))
}

func TestObject_IsTrue(t *testing.T) {
	var o core.Value

	// 0 is false, non-zero is true
	o = core.IntValue(0)
	require.False(t, o.IsTrue(alloc))
	o = core.IntValue(1)
	require.True(t, o.IsTrue(alloc))
	o = core.IntValue(123)
	require.True(t, o.IsTrue(alloc))
	o = core.IntValue(-456)
	require.True(t, o.IsTrue(alloc))

	// NaN is false, non-NaN is true
	o = core.FloatValue(0)
	require.True(t, o.IsTrue(alloc))
	o = core.FloatValue(1)
	require.True(t, o.IsTrue(alloc))

	// non-zero char is true
	o = core.RuneValue(' ')
	require.True(t, o.IsTrue(alloc))
	o = core.RuneValue('T')
	require.True(t, o.IsTrue(alloc))

	// empty string is false, non-empty string is true
	o = core.NewStringValue("")
	require.False(t, o.IsTrue(alloc))
	o = core.NewStringValue(" ")
	require.True(t, o.IsTrue(alloc))

	// empty array is false, non-empty array is true
	o = core.NewArrayValue(nil, false)
	require.False(t, o.IsTrue(alloc))
	o = core.NewArrayValue([]core.Value{core.Undefined}, false)
	require.True(t, o.IsTrue(alloc))

	// empty record is false, non-empty record is true
	o = core.NewRecordValue(nil, false)
	require.False(t, o.IsTrue(alloc))
	o = core.NewRecordValue(map[string]core.Value{"a": core.Undefined}, false)
	require.True(t, o.IsTrue(alloc))

	// undefined is false
	o = core.Undefined
	require.False(t, o.IsTrue(alloc))

	// error is false
	o = core.NewErrorValue(core.Undefined)
	require.False(t, o.IsTrue(alloc))

	// empty bytes is false, non-empty bytes is true
	o = core.NewBytesValue(nil, false)
	require.False(t, o.IsTrue(alloc))
	o = core.NewBytesValue([]byte{1, 2}, false)
	require.True(t, o.IsTrue(alloc))

	// empty range is false, non-empty range is true
	o = core.NewIntRangeValue(0, 0, 1)
	require.False(t, o.IsTrue(alloc))
	o = core.NewIntRangeValue(0, 10, 1)
	require.True(t, o.IsTrue(alloc))

	// byte
	o = core.ByteValue(0)
	require.False(t, o.IsTrue(alloc))
	o = core.ByteValue(123)
	require.True(t, o.IsTrue(alloc))
}

func TestObject_String(t *testing.T) {
	var o core.Value

	o = core.IntValue(0)
	require.Equal(t, alloc, "0", o.String(alloc))

	o = core.IntValue(1)
	require.Equal(t, alloc, "1", o.String(alloc))

	o = core.FloatValue(0)
	require.Equal(t, alloc, "0", o.String(alloc))

	o = core.FloatValue(1)
	require.Equal(t, alloc, "1", o.String(alloc))

	o = core.RuneValue(' ')
	require.Equal(t, alloc, "' '", o.String(alloc))

	o = core.RuneValue('T')
	require.Equal(t, alloc, "'T'", o.String(alloc))

	o = core.NewStringValue("")
	require.Equal(t, alloc, `""`, o.String(alloc))

	o = core.NewStringValue(" ")
	require.Equal(t, alloc, `" "`, o.String(alloc))

	o = core.NewArrayValue(nil, false)
	require.Equal(t, alloc, "[]", o.String(alloc))

	o = core.NewRecordValue(nil, false)
	require.Equal(t, alloc, "{}", o.String(alloc))

	o = core.NewErrorValue(core.Undefined)
	require.Equal(t, alloc, "error()", o.String(alloc))

	o = core.NewErrorValue(core.NewStringValue("error 1"))
	require.Equal(t, alloc, `error("error 1")`, o.String(alloc))

	o = core.Undefined
	require.Equal(t, alloc, "undefined", o.String(alloc))

	o = core.NewBytesValue(nil, false)
	require.Equal(t, alloc, "bytes([])", o.String(alloc))

	o = core.NewBytesValue([]byte("foo"), false)
	require.Equal(t, alloc, "bytes([102, 111, 111])", o.String(alloc))

	o = core.NewIntRangeValue(0, 10, 2)
	require.Equal(t, alloc, "range(0, 10, 2)", o.String(alloc))
}

func TestObject_BinaryOp(t *testing.T) {
	var o core.Value

	o = core.RuneValue(0)
	_, err := o.BinaryOp(alloc, token.Add, core.Undefined)
	require.Error(t, err)

	o = core.False
	_, err = o.BinaryOp(alloc, token.Add, core.Undefined)
	require.Error(t, err)

	o = core.NewRecordValue(nil, false)
	_, err = o.BinaryOp(alloc, token.Add, core.Undefined)
	require.Error(t, err)

	o = core.Undefined
	_, err = o.BinaryOp(alloc, token.Add, core.Undefined)
	require.Error(t, err)

	o = core.NewErrorValue(core.Undefined)
	_, err = o.BinaryOp(alloc, token.Add, core.Undefined)
	require.Error(t, err)
}

func TestArray_BinaryOp(t *testing.T) {
	testBinaryOp(t, core.NewArrayValue(nil, false), token.Add,
		core.NewArrayValue(nil, false), core.NewArrayValue(nil, false))
	testBinaryOp(t, core.NewArrayValue(nil, false), token.Add,
		core.NewArrayValue([]core.Value{}, false), core.NewArrayValue(nil, false))
	testBinaryOp(t, core.NewArrayValue([]core.Value{}, false), token.Add,
		core.NewArrayValue(nil, false), core.NewArrayValue([]core.Value{}, false))
	testBinaryOp(t, core.NewArrayValue([]core.Value{}, false), token.Add,
		core.NewArrayValue([]core.Value{}, false),
		core.NewArrayValue([]core.Value{}, false))
	testBinaryOp(t, core.NewArrayValue(nil, false), token.Add,
		core.NewArrayValue([]core.Value{
			core.IntValue(1),
		}, false), core.NewArrayValue([]core.Value{
			core.IntValue(1),
		}, false))
	testBinaryOp(t, core.NewArrayValue(nil, false), token.Add,
		core.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.IntValue(2),
			core.IntValue(3),
		}, false), core.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.IntValue(2),
			core.IntValue(3),
		}, false))
	testBinaryOp(t, core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
	}, false), token.Add, core.NewArrayValue(nil, false),
		core.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.IntValue(2),
			core.IntValue(3),
		}, false))
	testBinaryOp(t, core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
	}, false), token.Add, core.NewArrayValue([]core.Value{
		core.IntValue(4),
		core.IntValue(5),
		core.IntValue(6),
	}, false), core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
		core.IntValue(4),
		core.IntValue(5),
		core.IntValue(6),
	}, false))
}

func TestError_Equals(t *testing.T) {
	err1 := core.NewErrorValue(core.NewStringValue("some error"))
	err2 := err1
	require.True(t, err1.Equal(alloc, err2))
	require.True(t, err2.Equal(alloc, err1))

	err2 = core.NewErrorValue(core.NewStringValue("some error"))
	require.True(t, err1.Equal(alloc, err2))
	require.True(t, err2.Equal(alloc, err1))

	err2 = core.NewErrorValue(core.NewStringValue("some error 2"))
	require.False(t, err1.Equal(alloc, err2))
	require.False(t, err2.Equal(alloc, err1))

	range1 := core.NewIntRangeValue(0, 10, 2)
	range2 := core.NewIntRangeValue(0, 10, 2)
	range3 := core.NewIntRangeValue(0, 10, 1)
	require.True(t, range1.Equal(alloc, range2))
	require.True(t, range2.Equal(alloc, range1))
	require.False(t, range1.Equal(alloc, range3))
	require.False(t, range3.Equal(alloc, range1))

	bool1 := core.True
	bool2 := core.True
	bool3 := core.False

	char1 := core.RuneValue('A')
	char2 := core.RuneValue('A')
	char3 := core.RuneValue('B')

	int1 := core.IntValue(123)
	int2 := core.IntValue(123)
	int3 := core.IntValue(456)

	float1 := core.FloatValue(3.14)
	float2 := core.FloatValue(3.14)
	float3 := core.FloatValue(2.71828)

	string1 := core.NewStringValue("hello")
	string2 := core.NewStringValue("hello")
	string3 := core.NewStringValue("world")

	bytes1 := core.NewBytesValue([]byte("foo"), false)
	bytes2 := core.NewBytesValue([]byte("foo"), false)
	bytes3 := core.NewBytesValue([]byte("bar"), false)

	array1 := core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	array2 := core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	array3 := core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(3)}, false)

	map1 := core.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	map2 := core.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	map3 := core.NewRecordValue(map[string]core.Value{"a": core.IntValue(2)}, false)

	record1 := core.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	record2 := core.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	record3 := core.NewRecordValue(map[string]core.Value{"a": core.IntValue(2)}, false)

	// compare to undefined
	require.False(t, bool1.Equal(alloc, core.Undefined))
	require.False(t, char1.Equal(alloc, core.Undefined))
	require.False(t, int1.Equal(alloc, core.Undefined))
	require.False(t, float1.Equal(alloc, core.Undefined))
	require.False(t, string1.Equal(alloc, core.Undefined))
	require.False(t, bytes1.Equal(alloc, core.Undefined))
	require.False(t, array1.Equal(alloc, core.Undefined))
	require.False(t, map1.Equal(alloc, core.Undefined))
	require.False(t, record1.Equal(alloc, core.Undefined))

	// compare to equal
	require.True(t, bool1.Equal(alloc, bool2))
	require.True(t, char1.Equal(alloc, char2))
	require.True(t, int1.Equal(alloc, int2))
	require.True(t, float1.Equal(alloc, float2))
	require.True(t, string1.Equal(alloc, string2))
	require.True(t, bytes1.Equal(alloc, bytes2))
	require.True(t, array1.Equal(alloc, array2))
	require.True(t, map1.Equal(alloc, map2))
	require.True(t, record1.Equal(alloc, record2))

	// compare to not equal
	require.False(t, bool1.Equal(alloc, bool3))
	require.False(t, char1.Equal(alloc, char3))
	require.False(t, int1.Equal(alloc, int3))
	require.False(t, float1.Equal(alloc, float3))
	require.False(t, string1.Equal(alloc, string3))
	require.False(t, bytes1.Equal(alloc, bytes3))
	require.False(t, array1.Equal(alloc, array3))
	require.False(t, map1.Equal(alloc, map3))
	require.False(t, record1.Equal(alloc, record3))
}

func TestFloat_BinaryOp(t *testing.T) {
	// float + float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Add,
				core.FloatValue(r), core.FloatValue(l+r))
		}
	}

	// float - float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Sub,
				core.FloatValue(r), core.FloatValue(l-r))
		}
	}

	// float * float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Mul,
				core.FloatValue(r), core.FloatValue(l*r))
		}
	}

	// float / float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			if r != 0 {
				testBinaryOp(t, core.FloatValue(l), token.Quo,
					core.FloatValue(r), core.FloatValue(l/r))
			}
		}
	}

	// float < float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Less,
				core.FloatValue(r), boolValue(l < r))
		}
	}

	// float > float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Greater,
				core.FloatValue(r), boolValue(l > r))
		}
	}

	// float <= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.LessEq,
				core.FloatValue(r), boolValue(l <= r))
		}
	}

	// float >= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.GreaterEq,
				core.FloatValue(r), boolValue(l >= r))
		}
	}

	// float + int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Add,
				core.IntValue(r), core.FloatValue(l+float64(r)))
		}
	}

	// float - int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Sub,
				core.IntValue(r), core.FloatValue(l-float64(r)))
		}
	}

	// float * int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Mul,
				core.IntValue(r), core.FloatValue(l*float64(r)))
		}
	}

	// float / int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, core.FloatValue(l), token.Quo,
					core.IntValue(r),
					core.FloatValue(l/float64(r)))
			}
		}
	}

	// float < int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Less,
				core.IntValue(r), boolValue(l < float64(r)))
		}
	}

	// float > int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Greater,
				core.IntValue(r), boolValue(l > float64(r)))
		}
	}

	// float <= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.LessEq,
				core.IntValue(r), boolValue(l <= float64(r)))
		}
	}

	// float >= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.GreaterEq,
				core.IntValue(r), boolValue(l >= float64(r)))
		}
	}
}

func TestInt_BinaryOp(t *testing.T) {
	// int + int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Add,
				core.IntValue(r), core.IntValue(l+r))
		}
	}

	// int - int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Sub,
				core.IntValue(r), core.IntValue(l-r))
		}
	}

	// int * int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Mul,
				core.IntValue(r), core.IntValue(l*r))
		}
	}

	// int / int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, core.IntValue(l), token.Quo,
					core.IntValue(r), core.IntValue(l/r))
			}
		}
	}

	// int % int
	for l := int64(-4); l <= 4; l++ {
		for r := -int64(-4); r <= 4; r++ {
			if r == 0 {
				testBinaryOp(t, core.IntValue(l), token.Rem,
					core.IntValue(r), core.IntValue(l%r))
			}
		}
	}

	// int & int
	testBinaryOp(t,
		core.IntValue(0), token.And, core.IntValue(0),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1), token.And, core.IntValue(0),
		core.IntValue(int64(1)&int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.And, core.IntValue(1),
		core.IntValue(int64(0)&int64(1)))
	testBinaryOp(t,
		core.IntValue(1), token.And, core.IntValue(1),
		core.IntValue(int64(1)))
	testBinaryOp(t,
		core.IntValue(0), token.And, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)&int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1), token.And, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1)&int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(int64(0xffffffff)), token.And,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1984), token.And,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1984)&int64(0xffffffff)))
	testBinaryOp(t, core.IntValue(-1984), token.And,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(-1984)&int64(0xffffffff)))

	// int | int
	testBinaryOp(t,
		core.IntValue(0), token.Or, core.IntValue(0),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1), token.Or, core.IntValue(0),
		core.IntValue(int64(1)|int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.Or, core.IntValue(1),
		core.IntValue(int64(0)|int64(1)))
	testBinaryOp(t,
		core.IntValue(1), token.Or, core.IntValue(1),
		core.IntValue(int64(1)))
	testBinaryOp(t,
		core.IntValue(0), token.Or, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)|int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1), token.Or, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1)|int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(int64(0xffffffff)), token.Or,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1984), token.Or,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1984)|int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(-1984), token.Or,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(-1984)|int64(0xffffffff)))

	// int ^ int
	testBinaryOp(t,
		core.IntValue(0), token.Xor, core.IntValue(0),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1), token.Xor, core.IntValue(0),
		core.IntValue(int64(1)^int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.Xor, core.IntValue(1),
		core.IntValue(int64(0)^int64(1)))
	testBinaryOp(t,
		core.IntValue(1), token.Xor, core.IntValue(1),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.Xor, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1), token.Xor, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1)^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(int64(0xffffffff)), token.Xor,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1984), token.Xor,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1984)^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(-1984), token.Xor,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(-1984)^int64(0xffffffff)))

	// int &^ int
	testBinaryOp(t,
		core.IntValue(0), token.AndNot, core.IntValue(0),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1), token.AndNot, core.IntValue(0),
		core.IntValue(int64(1)&^int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.AndNot,
		core.IntValue(1), core.IntValue(int64(0)&^int64(1)))
	testBinaryOp(t,
		core.IntValue(1), token.AndNot, core.IntValue(1),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)&^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1)&^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(int64(0xffffffff)), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1984), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1984)&^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(-1984), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(-1984)&^int64(0xffffffff)))

	// int << int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			core.IntValue(0), token.Shl, core.IntValue(s),
			core.IntValue(int64(0)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(1), token.Shl, core.IntValue(s),
			core.IntValue(int64(1)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(2), token.Shl, core.IntValue(s),
			core.IntValue(int64(2)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(-1), token.Shl, core.IntValue(s),
			core.IntValue(int64(-1)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(-2), token.Shl, core.IntValue(s),
			core.IntValue(int64(-2)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(int64(0xffffffff)), token.Shl,
			core.IntValue(s),
			core.IntValue(int64(0xffffffff)<<uint(s)))
	}

	// int >> int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			core.IntValue(0), token.Shr, core.IntValue(s),
			core.IntValue(int64(0)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(1), token.Shr, core.IntValue(s),
			core.IntValue(int64(1)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(2), token.Shr, core.IntValue(s),
			core.IntValue(int64(2)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(-1), token.Shr, core.IntValue(s),
			core.IntValue(int64(-1)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(-2), token.Shr, core.IntValue(s),
			core.IntValue(int64(-2)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(int64(0xffffffff)), token.Shr,
			core.IntValue(s),
			core.IntValue(int64(0xffffffff)>>uint(s)))
	}

	// int < int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Less,
				core.IntValue(r), boolValue(l < r))
		}
	}

	// int > int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Greater,
				core.IntValue(r), boolValue(l > r))
		}
	}

	// int <= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.LessEq,
				core.IntValue(r), boolValue(l <= r))
		}
	}

	// int >= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.GreaterEq,
				core.IntValue(r), boolValue(l >= r))
		}
	}

	// int + float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Add,
				core.FloatValue(r),
				core.FloatValue(float64(l)+r))
		}
	}

	// int - float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Sub,
				core.FloatValue(r),
				core.FloatValue(float64(l)-r))
		}
	}

	// int * float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Mul,
				core.FloatValue(r),
				core.FloatValue(float64(l)*r))
		}
	}

	// int / float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			if r != 0 {
				testBinaryOp(t, core.IntValue(l), token.Quo,
					core.FloatValue(r),
					core.FloatValue(float64(l)/r))
			}
		}
	}

	// int < float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Less,
				core.FloatValue(r), boolValue(float64(l) < r))
		}
	}

	// int > float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Greater,
				core.FloatValue(r), boolValue(float64(l) > r))
		}
	}

	// int <= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.LessEq,
				core.FloatValue(r), boolValue(float64(l) <= r))
		}
	}

	// int >= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.GreaterEq,
				core.FloatValue(r), boolValue(float64(l) >= r))
		}
	}
}

func TestRecord_Index(t *testing.T) {
	m := core.NewRecordValue(make(map[string]core.Value), false)
	k := core.IntValue(1)
	v := core.NewStringValue("abcdef")
	err := m.Assign(alloc, k, v)

	require.NoError(t, err)

	res, err := m.Access(alloc, k, bc.OpIndex)
	require.NoError(t, err)
	require.Equal(t, alloc, v, res)
}

func TestString_BinaryOp(t *testing.T) {
	lstr := "abcde"
	rstr := "01234"
	for l := 0; l < len(lstr); l++ {
		for r := 0; r < len(rstr); r++ {
			ls := lstr[l:]
			rs := rstr[r:]
			testBinaryOp(t, core.NewStringValue(ls), token.Add,
				core.NewStringValue(rs),
				core.NewStringValue(ls+rs))

			rc := []rune(rstr)[r]
			testBinaryOp(t, core.NewStringValue(ls), token.Add,
				core.RuneValue(rc),
				core.NewStringValue(ls+string(rc)))
		}
	}
}

func testBinaryOp(t *testing.T, lhs core.Value, op token.Token, rhs core.Value, expected core.Value) {
	t.Helper()
	actual, err := lhs.BinaryOp(alloc, op, rhs)
	require.NoError(t, err)
	require.Equal(t, alloc, expected, actual)
}

func boolValue(b bool) core.Value {
	return core.BoolValue(b)
}
