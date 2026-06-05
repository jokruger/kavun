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
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_UNDEFINED)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Bool
	v = core.True
	require.True(t, v.Type == core.VT_BOOL)
	require.Equal(t, rta, true, v.Data != 0)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BOOL)
	require.Equal(t, rta, true, x.Data != 0)
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = core.False
	require.True(t, v.Type == core.VT_BOOL)
	require.Equal(t, rta, false, v.Data != 0)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BOOL)
	require.Equal(t, rta, false, x.Data != 0)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Byte
	v = core.ByteValue(123)
	require.True(t, v.Type == core.VT_BYTE)
	require.Equal(t, rta, byte(123), byte(v.Data))
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BYTE)
	require.Equal(t, rta, byte(123), byte(x.Data))
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Rune
	v = core.RuneValue('A')
	require.True(t, v.Type == core.VT_RUNE)
	require.Equal(t, rta, 'A', rune(v.Data))
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNE)
	require.Equal(t, rta, 'A', rune(x.Data))
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = core.RuneValue('₴')
	require.True(t, v.Type == core.VT_RUNE)
	require.Equal(t, rta, '₴', rune(v.Data))
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNE)
	require.Equal(t, rta, '₴', rune(x.Data))
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Int
	v = core.IntValue(123)
	require.True(t, v.Type == core.VT_INT)
	require.Equal(t, rta, int64(123), int64(v.Data))
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_INT)
	require.Equal(t, rta, int64(123), int64(x.Data))
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = core.IntValue(-456)
	require.True(t, v.Type == core.VT_INT)
	require.Equal(t, rta, int64(-456), int64(v.Data))
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_INT)
	require.Equal(t, rta, int64(-456), int64(x.Data))
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Float
	v = core.FloatValue(3.14)
	require.True(t, v.Type == core.VT_FLOAT)
	require.Equal(t, rta, 3.14, math.Float64frombits(v.Data))
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_FLOAT)
	require.Equal(t, rta, 3.14, math.Float64frombits(x.Data))
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = core.FloatValue(-2.71828)
	require.True(t, v.Type == core.VT_FLOAT)
	require.Equal(t, rta, -2.71828, math.Float64frombits(v.Data))
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_FLOAT)
	require.Equal(t, rta, -2.71828, math.Float64frombits(x.Data))
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Decimal
	v = rta.NewDecimalValue(dec128.FromString("3.14"))
	require.True(t, v.Type == core.VT_DECIMAL)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_DECIMAL)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// String
	v = rta.NewStringValue("")
	require.True(t, v.Type == core.VT_STRING)
	s, _ = v.AsString(rta)
	require.Equal(t, rta, "", s)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_STRING)
	s, _ = x.AsString(rta)
	require.Equal(t, rta, "", s)
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = rta.NewStringValue("hello")
	require.True(t, v.Type == core.VT_STRING)
	s, _ = v.AsString(rta)
	require.Equal(t, rta, "hello", s)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_STRING)
	s, _ = x.AsString(rta)
	require.Equal(t, rta, "hello", s)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Runes
	v = rta.NewRunesValue([]rune(""), false)
	require.True(t, v.Type == core.VT_RUNES)
	s, _ = v.AsString(rta)
	require.Equal(t, rta, "", s)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNES)
	s, _ = x.AsString(rta)
	require.Equal(t, rta, "", s)
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = rta.NewRunesValue([]rune("путін хуйло"), false)
	require.True(t, v.Type == core.VT_RUNES)
	s, _ = v.AsString(rta)
	require.Equal(t, rta, "путін хуйло", s)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNES)
	s, _ = x.AsString(rta)
	require.Equal(t, rta, "путін хуйло", s)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Bytes
	v = rta.NewBytesValue([]byte{}, false)
	require.True(t, v.Type == core.VT_BYTES)
	b, _ := v.AsBytes(rta)
	require.Equal(t, rta, []byte{}, b)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BYTES)
	b, _ = x.AsBytes(rta)
	require.Equal(t, rta, []byte{}, b)
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = rta.NewBytesValue([]byte("foo"), false)
	require.True(t, v.Type == core.VT_BYTES)
	b, _ = v.AsBytes(rta)
	require.Equal(t, rta, []byte("foo"), b)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BYTES)
	b, _ = x.AsBytes(rta)
	require.Equal(t, rta, []byte("foo"), b)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Array
	v = rta.NewArrayValue([]core.Value{}, false)
	require.True(t, v.Type == core.VT_ARRAY)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ARRAY)
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = rta.NewArrayValue([]core.Value{}, true)
	require.True(t, v.Type == core.VT_ARRAY)
	require.True(t, v.Immutable)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ARRAY)
	require.True(t, x.Immutable)
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = rta.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	require.True(t, v.Type == core.VT_ARRAY)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ARRAY)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Record
	v = rta.NewRecordValue(map[string]core.Value{}, true)
	require.True(t, v.Type == core.VT_RECORD)
	require.True(t, v.Immutable)
	require.True(t, v.Immutable)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RECORD)
	require.True(t, x.Immutable)
	require.True(t, x.Immutable)
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = rta.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.Type == core.VT_RECORD)
	require.False(t, v.Immutable)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RECORD)
	require.False(t, x.Immutable)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Map
	v = rta.NewDictValue(map[string]core.Value{}, true)
	require.True(t, v.Type == core.VT_DICT)
	require.True(t, v.Immutable)
	require.True(t, v.Immutable)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_DICT)
	require.True(t, x.Immutable)
	require.True(t, x.Immutable)
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = rta.NewDictValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.Type == core.VT_DICT)
	require.False(t, v.Immutable)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_DICT)
	require.False(t, x.Immutable)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Error
	v = rta.NewErrorValue(core.Undefined, core.KindUser, false)
	require.True(t, v.Type == core.VT_ERROR)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ERROR)
	require.Equal(t, rta, true, v.Equal(rta, x))

	v = rta.NewErrorValue(rta.NewStringValue("some error"), core.KindUser, false)
	require.True(t, v.Type == core.VT_ERROR)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ERROR)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// Time
	v = rta.NewTimeValue(time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC))
	require.True(t, v.Type == core.VT_TIME)
	tm, _ := v.AsTime(rta)
	require.Equal(t, rta, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_TIME)
	tm, _ = x.AsTime(rta)
	require.Equal(t, rta, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	require.Equal(t, rta, true, v.Equal(rta, x))

	// IntRange
	v = rta.NewIntRangeValue(0, 0, 1)
	require.True(t, v.Type == core.VT_INT_RANGE)
	rng := (*core.IntRange)(v.Ptr)
	require.True(t, rng.Empty())
	require.Equal(t, rta, int64(0), rng.Len())
	v = rta.NewIntRangeValue(0, 10, 1)
	rng = (*core.IntRange)(v.Ptr)
	require.False(t, rng.Empty())
	require.Equal(t, rta, int64(10), rng.Len())
	i, ok = rng.Get(0)
	require.True(t, ok)
	require.Equal(t, rta, int64(0), i)
	i, ok = rng.Get(9)
	require.True(t, ok)
	require.Equal(t, rta, int64(9), i)
	i, ok = rng.Get(10)
	require.False(t, ok)
	v = rta.NewIntRangeValue(10, 0, 1)
	rng = (*core.IntRange)(v.Ptr)
	require.False(t, rng.Empty())
	require.Equal(t, rta, int64(10), rng.Len())
	i, ok = rng.Get(0)
	require.True(t, ok)
	require.Equal(t, rta, int64(10), i)
	i, ok = rng.Get(9)
	require.True(t, ok)
	require.Equal(t, rta, int64(1), i)
	i, ok = rng.Get(10)
	require.False(t, ok)

	v = rta.NewIntRangeValue(0, 10, 2)
	require.True(t, v.Type == core.VT_INT_RANGE)
	rng = (*core.IntRange)(v.Ptr)
	require.Equal(t, rta, int64(0), rng.Start)
	require.Equal(t, rta, int64(10), rng.Stop)
	require.Equal(t, rta, int64(2), rng.Step)
	bs, err = v.EncodeBinary(rta)
	require.NoError(t, err)
	err = x.DecodeBinary(rta, bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_INT_RANGE)
	rng = (*core.IntRange)(x.Ptr)
	require.Equal(t, rta, int64(0), rng.Start)
	require.Equal(t, rta, int64(10), rng.Stop)
	require.Equal(t, rta, int64(2), rng.Step)
	require.Equal(t, rta, true, v.Equal(rta, x))
}

func TestObject_TypeName(t *testing.T) {
	var o core.Value

	o = core.IntValue(0)
	require.Equal(t, rta, "int", o.TypeName(rta))

	o = core.FloatValue(0)
	require.Equal(t, rta, "float", o.TypeName(rta))

	o = core.ByteValue(0)
	require.Equal(t, rta, "byte", o.TypeName(rta))

	o = core.RuneValue(0)
	require.Equal(t, rta, "rune", o.TypeName(rta))

	o = rta.NewStringValue("")
	require.Equal(t, rta, "string", o.TypeName(rta))

	o = core.False
	require.Equal(t, rta, "bool", o.TypeName(rta))

	o = rta.NewArrayValue(nil, false)
	require.Equal(t, rta, "array", o.TypeName(rta))

	o = rta.NewArrayValue(nil, true)
	require.Equal(t, rta, "immutable-array", o.TypeName(rta))

	o = rta.NewRecordValue(nil, false)
	require.Equal(t, rta, "record", o.TypeName(rta))

	o = rta.NewRecordValue(nil, true)
	require.Equal(t, rta, "immutable-record", o.TypeName(rta))

	o = rta.NewDictValue(nil, false)
	require.Equal(t, rta, "dict", o.TypeName(rta))

	o = rta.NewDictValue(nil, true)
	require.Equal(t, rta, "immutable-dict", o.TypeName(rta))

	o = core.NewBuiltinClosureValue("fn", nil, 0, false)
	require.Equal(t, rta, "<builtin-closure:fn/0>", o.TypeName(rta))

	o = core.Undefined
	require.Equal(t, rta, "undefined", o.TypeName(rta))

	o = rta.NewErrorValue(core.Undefined, core.KindUser, false)
	require.Equal(t, rta, "error", o.TypeName(rta))

	o = rta.NewBytesValue(nil, false)
	require.Equal(t, rta, "bytes", o.TypeName(rta))

	o = core.NewIntRangeValue(1, 10, 1)
	require.Equal(t, rta, "range", o.TypeName(rta))
}

func TestObject_IsTrue(t *testing.T) {
	var o core.Value

	// 0 is false, non-zero is true
	o = core.IntValue(0)
	require.False(t, o.IsTrue(rta))
	o = core.IntValue(1)
	require.True(t, o.IsTrue(rta))
	o = core.IntValue(123)
	require.True(t, o.IsTrue(rta))
	o = core.IntValue(-456)
	require.True(t, o.IsTrue(rta))

	// NaN is false, non-NaN is true
	o = core.FloatValue(0)
	require.True(t, o.IsTrue(rta))
	o = core.FloatValue(1)
	require.True(t, o.IsTrue(rta))

	// non-zero char is true
	o = core.RuneValue(' ')
	require.True(t, o.IsTrue(rta))
	o = core.RuneValue('T')
	require.True(t, o.IsTrue(rta))

	// empty string is false, non-empty string is true
	o = rta.NewStringValue("")
	require.False(t, o.IsTrue(rta))
	o = rta.NewStringValue(" ")
	require.True(t, o.IsTrue(rta))

	// empty array is false, non-empty array is true
	o = rta.NewArrayValue(nil, false)
	require.False(t, o.IsTrue(rta))
	o = rta.NewArrayValue([]core.Value{core.Undefined}, false)
	require.True(t, o.IsTrue(rta))

	// empty record is false, non-empty record is true
	o = rta.NewRecordValue(nil, false)
	require.False(t, o.IsTrue(rta))
	o = rta.NewRecordValue(map[string]core.Value{"a": core.Undefined}, false)
	require.True(t, o.IsTrue(rta))

	// undefined is false
	o = core.Undefined
	require.False(t, o.IsTrue(rta))

	// error is false
	o = rta.NewErrorValue(core.Undefined, core.KindUser, false)
	require.False(t, o.IsTrue(rta))

	// empty bytes is false, non-empty bytes is true
	o = rta.NewBytesValue(nil, false)
	require.False(t, o.IsTrue(rta))
	o = rta.NewBytesValue([]byte{1, 2}, false)
	require.True(t, o.IsTrue(rta))

	// empty range is false, non-empty range is true
	o = core.NewIntRangeValue(0, 0, 1)
	require.False(t, o.IsTrue(rta))
	o = core.NewIntRangeValue(0, 10, 1)
	require.True(t, o.IsTrue(rta))

	// byte
	o = core.ByteValue(0)
	require.False(t, o.IsTrue(rta))
	o = core.ByteValue(123)
	require.True(t, o.IsTrue(rta))
}

func TestObject_String(t *testing.T) {
	var o core.Value

	o = core.IntValue(0)
	require.Equal(t, rta, "0", o.String(rta))

	o = core.IntValue(1)
	require.Equal(t, rta, "1", o.String(rta))

	o = core.FloatValue(0)
	require.Equal(t, rta, "0", o.String(rta))

	o = core.FloatValue(1)
	require.Equal(t, rta, "1", o.String(rta))

	o = core.RuneValue(' ')
	require.Equal(t, rta, "' '", o.String(rta))

	o = core.RuneValue('T')
	require.Equal(t, rta, "'T'", o.String(rta))

	o = rta.NewStringValue("")
	require.Equal(t, rta, `""`, o.String(rta))

	o = rta.NewStringValue(" ")
	require.Equal(t, rta, `" "`, o.String(rta))

	o = rta.NewArrayValue(nil, false)
	require.Equal(t, rta, "[]", o.String(rta))

	o = rta.NewRecordValue(nil, false)
	require.Equal(t, rta, "{}", o.String(rta))

	o = rta.NewErrorValue(core.Undefined, core.KindUser, false)
	require.Equal(t, rta, "error()", o.String(rta))

	o = rta.NewErrorValue(rta.NewStringValue("error 1"), core.KindUser, false)
	require.Equal(t, rta, `error("error 1")`, o.String(rta))

	o = core.Undefined
	require.Equal(t, rta, "undefined", o.String(rta))

	o = rta.NewBytesValue(nil, false)
	require.Equal(t, rta, "bytes([])", o.String(rta))

	o = rta.NewBytesValue([]byte("foo"), false)
	require.Equal(t, rta, "bytes([102, 111, 111])", o.String(rta))

	o = rta.NewIntRangeValue(0, 10, 2)
	require.Equal(t, rta, "range(0, 10, 2)", o.String(rta))
}

func TestObject_BinaryOp(t *testing.T) {
	var o core.Value

	o = core.RuneValue(0)
	_, err := o.BinaryOp(rta, token.Add, core.Undefined)
	require.Error(t, err)

	o = core.False
	_, err = o.BinaryOp(rta, token.Add, core.Undefined)
	require.Error(t, err)

	o = rta.NewRecordValue(nil, false)
	_, err = o.BinaryOp(rta, token.Add, core.Undefined)
	require.Error(t, err)

	o = core.Undefined
	_, err = o.BinaryOp(rta, token.Add, core.Undefined)
	require.Error(t, err)

	o = rta.NewErrorValue(core.Undefined, core.KindUser, false)
	_, err = o.BinaryOp(rta, token.Add, core.Undefined)
	require.Error(t, err)
}

func TestArray_BinaryOp(t *testing.T) {
	testBinaryOp(t, rta.NewArrayValue(nil, false), token.Add,
		rta.NewArrayValue(nil, false), rta.NewArrayValue(nil, false))
	testBinaryOp(t, rta.NewArrayValue(nil, false), token.Add,
		rta.NewArrayValue([]core.Value{}, false), rta.NewArrayValue(nil, false))
	testBinaryOp(t, rta.NewArrayValue([]core.Value{}, false), token.Add,
		rta.NewArrayValue(nil, false), rta.NewArrayValue([]core.Value{}, false))
	testBinaryOp(t, rta.NewArrayValue([]core.Value{}, false), token.Add,
		rta.NewArrayValue([]core.Value{}, false),
		rta.NewArrayValue([]core.Value{}, false))
	testBinaryOp(t, rta.NewArrayValue(nil, false), token.Add,
		rta.NewArrayValue([]core.Value{
			core.IntValue(1),
		}, false), rta.NewArrayValue([]core.Value{
			core.IntValue(1),
		}, false))
	testBinaryOp(t, rta.NewArrayValue(nil, false), token.Add,
		rta.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.IntValue(2),
			core.IntValue(3),
		}, false), rta.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.IntValue(2),
			core.IntValue(3),
		}, false))
	testBinaryOp(t, rta.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
	}, false), token.Add, rta.NewArrayValue(nil, false),
		rta.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.IntValue(2),
			core.IntValue(3),
		}, false))
	testBinaryOp(t, rta.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
	}, false), token.Add, rta.NewArrayValue([]core.Value{
		core.IntValue(4),
		core.IntValue(5),
		core.IntValue(6),
	}, false), rta.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
		core.IntValue(4),
		core.IntValue(5),
		core.IntValue(6),
	}, false))
}

func TestError_Equals(t *testing.T) {
	err1 := rta.NewErrorValue(rta.NewStringValue("some error"), core.KindUser, false)
	err2 := err1
	require.True(t, err1.Equal(rta, err2))
	require.True(t, err2.Equal(rta, err1))

	err2 = rta.NewErrorValue(rta.NewStringValue("some error"), core.KindUser, false)
	require.True(t, err1.Equal(rta, err2))
	require.True(t, err2.Equal(rta, err1))

	err2 = rta.NewErrorValue(rta.NewStringValue("some error 2"), core.KindUser, false)
	require.False(t, err1.Equal(rta, err2))
	require.False(t, err2.Equal(rta, err1))

	range1 := core.NewIntRangeValue(0, 10, 2)
	range2 := core.NewIntRangeValue(0, 10, 2)
	range3 := core.NewIntRangeValue(0, 10, 1)
	require.True(t, range1.Equal(rta, range2))
	require.True(t, range2.Equal(rta, range1))
	require.False(t, range1.Equal(rta, range3))
	require.False(t, range3.Equal(rta, range1))

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

	string1 := rta.NewStringValue("hello")
	string2 := rta.NewStringValue("hello")
	string3 := rta.NewStringValue("world")

	bytes1 := rta.NewBytesValue([]byte("foo"), false)
	bytes2 := rta.NewBytesValue([]byte("foo"), false)
	bytes3 := rta.NewBytesValue([]byte("bar"), false)

	array1 := rta.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	array2 := rta.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	array3 := rta.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(3)}, false)

	map1 := rta.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	map2 := rta.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	map3 := rta.NewRecordValue(map[string]core.Value{"a": core.IntValue(2)}, false)

	record1 := rta.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	record2 := rta.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	record3 := rta.NewRecordValue(map[string]core.Value{"a": core.IntValue(2)}, false)

	// compare to undefined
	require.False(t, bool1.Equal(rta, core.Undefined))
	require.False(t, char1.Equal(rta, core.Undefined))
	require.False(t, int1.Equal(rta, core.Undefined))
	require.False(t, float1.Equal(rta, core.Undefined))
	require.False(t, string1.Equal(rta, core.Undefined))
	require.False(t, bytes1.Equal(rta, core.Undefined))
	require.False(t, array1.Equal(rta, core.Undefined))
	require.False(t, map1.Equal(rta, core.Undefined))
	require.False(t, record1.Equal(rta, core.Undefined))

	// compare to equal
	require.True(t, bool1.Equal(rta, bool2))
	require.True(t, char1.Equal(rta, char2))
	require.True(t, int1.Equal(rta, int2))
	require.True(t, float1.Equal(rta, float2))
	require.True(t, string1.Equal(rta, string2))
	require.True(t, bytes1.Equal(rta, bytes2))
	require.True(t, array1.Equal(rta, array2))
	require.True(t, map1.Equal(rta, map2))
	require.True(t, record1.Equal(rta, record2))

	// compare to not equal
	require.False(t, bool1.Equal(rta, bool3))
	require.False(t, char1.Equal(rta, char3))
	require.False(t, int1.Equal(rta, int3))
	require.False(t, float1.Equal(rta, float3))
	require.False(t, string1.Equal(rta, string3))
	require.False(t, bytes1.Equal(rta, bytes3))
	require.False(t, array1.Equal(rta, array3))
	require.False(t, map1.Equal(rta, map3))
	require.False(t, record1.Equal(rta, record3))
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
	m := rta.NewRecordValue(make(map[string]core.Value), false)
	k := core.IntValue(1)
	v := rta.NewStringValue("abcdef")
	err := m.Assign(rta, k, v)

	require.NoError(t, err)

	res, err := m.Access(rta, k, bc.OpIndex)
	require.NoError(t, err)
	require.Equal(t, rta, v, res)
}

func TestString_BinaryOp(t *testing.T) {
	lstr := "abcde"
	rstr := "01234"
	for l := 0; l < len(lstr); l++ {
		for r := 0; r < len(rstr); r++ {
			ls := lstr[l:]
			rs := rstr[r:]
			testBinaryOp(t, rta.NewStringValue(ls), token.Add,
				rta.NewStringValue(rs),
				rta.NewStringValue(ls+rs))

			rc := []rune(rstr)[r]
			testBinaryOp(t, rta.NewStringValue(ls), token.Add,
				core.RuneValue(rc),
				rta.NewStringValue(ls+string(rc)))
		}
	}
}

func testBinaryOp(t *testing.T, lhs core.Value, op token.Token, rhs core.Value, expected core.Value) {
	t.Helper()
	actual, err := lhs.BinaryOp(rta, op, rhs)
	require.NoError(t, err)
	require.Equal(t, rta, expected, actual)
}

func boolValue(b bool) core.Value {
	return core.BoolValue(b)
}
