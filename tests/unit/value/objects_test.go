package value

import (
	"math"
	"testing"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
	mock "github.com/jokruger/kavun/tests"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/token"
	_ "github.com/jokruger/kavun/vm"
)

var vm = mock.Vm
var alloc = core.NewArena(nil)

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
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_UNDEFINED)
	require.Equal(t, true, v.Equal(x))

	// Bool
	v = core.True
	require.True(t, v.Type == core.VT_BOOL)
	require.Equal(t, true, v.Data != 0)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BOOL)
	require.Equal(t, true, x.Data != 0)
	require.Equal(t, true, v.Equal(x))

	v = core.False
	require.True(t, v.Type == core.VT_BOOL)
	require.Equal(t, false, v.Data != 0)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BOOL)
	require.Equal(t, false, x.Data != 0)
	require.Equal(t, true, v.Equal(x))

	// Rune
	v = core.RuneValue('A')
	require.True(t, v.Type == core.VT_RUNE)
	require.Equal(t, 'A', rune(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNE)
	require.Equal(t, 'A', rune(x.Data))
	require.Equal(t, true, v.Equal(x))

	v = core.RuneValue('₴')
	require.True(t, v.Type == core.VT_RUNE)
	require.Equal(t, '₴', rune(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNE)
	require.Equal(t, '₴', rune(x.Data))
	require.Equal(t, true, v.Equal(x))

	// Int
	v = core.IntValue(123)
	require.True(t, v.Type == core.VT_INT)
	require.Equal(t, int64(123), int64(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_INT)
	require.Equal(t, int64(123), int64(x.Data))
	require.Equal(t, true, v.Equal(x))

	v = core.IntValue(-456)
	require.True(t, v.Type == core.VT_INT)
	require.Equal(t, int64(-456), int64(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_INT)
	require.Equal(t, int64(-456), int64(x.Data))
	require.Equal(t, true, v.Equal(x))

	// Float
	v = core.FloatValue(3.14)
	require.True(t, v.Type == core.VT_FLOAT)
	require.Equal(t, 3.14, math.Float64frombits(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_FLOAT)
	require.Equal(t, 3.14, math.Float64frombits(x.Data))
	require.Equal(t, true, v.Equal(x))

	v = core.FloatValue(-2.71828)
	require.True(t, v.Type == core.VT_FLOAT)
	require.Equal(t, -2.71828, math.Float64frombits(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_FLOAT)
	require.Equal(t, -2.71828, math.Float64frombits(x.Data))
	require.Equal(t, true, v.Equal(x))

	// Decimal
	v = core.NewDecimalValue(dec128.FromString("3.14"))
	require.True(t, v.Type == core.VT_DECIMAL)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_DECIMAL)
	require.Equal(t, true, v.Equal(x))

	// String
	v = core.NewStringValue("")
	require.True(t, v.Type == core.VT_STRING)
	s, _ = v.AsString()
	require.Equal(t, "", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_STRING)
	s, _ = x.AsString()
	require.Equal(t, "", s)
	require.Equal(t, true, v.Equal(x))

	v = core.NewStringValue("hello")
	require.True(t, v.Type == core.VT_STRING)
	s, _ = v.AsString()
	require.Equal(t, "hello", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_STRING)
	s, _ = x.AsString()
	require.Equal(t, "hello", s)
	require.Equal(t, true, v.Equal(x))

	// Runes
	v = core.NewRunesValue([]rune(""))
	require.True(t, v.Type == core.VT_RUNES)
	s, _ = v.AsString()
	require.Equal(t, "", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNES)
	s, _ = x.AsString()
	require.Equal(t, "", s)
	require.Equal(t, true, v.Equal(x))

	v = core.NewRunesValue([]rune("путін хуйло"))
	require.True(t, v.Type == core.VT_RUNES)
	s, _ = v.AsString()
	require.Equal(t, "путін хуйло", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RUNES)
	s, _ = x.AsString()
	require.Equal(t, "путін хуйло", s)
	require.Equal(t, true, v.Equal(x))

	// Bytes
	v = core.NewBytesValue([]byte{})
	require.True(t, v.Type == core.VT_BYTES)
	b, _ := v.AsBytes()
	require.Equal(t, []byte{}, b)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BYTES)
	b, _ = x.AsBytes()
	require.Equal(t, []byte{}, b)
	require.Equal(t, true, v.Equal(x))

	v = core.NewBytesValue([]byte("foo"))
	require.True(t, v.Type == core.VT_BYTES)
	b, _ = v.AsBytes()
	require.Equal(t, []byte("foo"), b)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_BYTES)
	b, _ = x.AsBytes()
	require.Equal(t, []byte("foo"), b)
	require.Equal(t, true, v.Equal(x))

	// Array
	v = core.NewArrayValue([]core.Value{}, false)
	require.True(t, v.Type == core.VT_ARRAY)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ARRAY)
	require.Equal(t, true, v.Equal(x))

	v = core.NewArrayValue([]core.Value{}, true)
	require.True(t, v.Type == core.VT_ARRAY)
	require.True(t, v.Const)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ARRAY)
	require.True(t, x.Const)
	require.Equal(t, true, v.Equal(x))

	v = core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	require.True(t, v.Type == core.VT_ARRAY)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ARRAY)
	require.Equal(t, true, v.Equal(x))

	// Record
	v = core.NewRecordValue(map[string]core.Value{}, true)
	require.True(t, v.Type == core.VT_RECORD)
	require.True(t, v.Const)
	require.True(t, v.IsImmutable())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RECORD)
	require.True(t, x.Const)
	require.True(t, x.IsImmutable())
	require.Equal(t, true, v.Equal(x))

	v = core.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.Type == core.VT_RECORD)
	require.False(t, v.IsImmutable())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_RECORD)
	require.False(t, x.IsImmutable())
	require.Equal(t, true, v.Equal(x))

	// Map
	v = core.NewDictValue(map[string]core.Value{}, true)
	require.True(t, v.Type == core.VT_DICT)
	require.True(t, v.Const)
	require.True(t, v.IsImmutable())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_DICT)
	require.True(t, x.Const)
	require.True(t, x.IsImmutable())
	require.Equal(t, true, v.Equal(x))

	v = core.NewDictValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.Type == core.VT_DICT)
	require.False(t, v.IsImmutable())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_DICT)
	require.False(t, x.IsImmutable())
	require.Equal(t, true, v.Equal(x))

	// Error
	v = core.NewErrorValue(core.Undefined)
	require.True(t, v.Type == core.VT_ERROR)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ERROR)
	require.Equal(t, true, v.Equal(x))

	v = core.NewErrorValue(core.NewStringValue("some error"))
	require.True(t, v.Type == core.VT_ERROR)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_ERROR)
	require.Equal(t, true, v.Equal(x))

	// Time
	v = core.NewTimeValue(time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC))
	require.True(t, v.Type == core.VT_TIME)
	tm, _ := v.AsTime()
	require.Equal(t, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_TIME)
	tm, _ = x.AsTime()
	require.Equal(t, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	require.Equal(t, true, v.Equal(x))

	// IntRange
	v = core.NewIntRangeValue(0, 0, 1)
	require.True(t, v.Type == core.VT_INT_RANGE)
	rng := (*core.IntRange)(v.Ptr)
	require.True(t, rng.Empty())
	require.Equal(t, int64(0), rng.Len())
	v = core.NewIntRangeValue(0, 10, 1)
	rng = (*core.IntRange)(v.Ptr)
	require.False(t, rng.Empty())
	require.Equal(t, int64(10), rng.Len())
	i, ok = rng.Get(0)
	require.True(t, ok)
	require.Equal(t, int64(0), i)
	i, ok = rng.Get(9)
	require.True(t, ok)
	require.Equal(t, int64(9), i)
	i, ok = rng.Get(10)
	require.False(t, ok)
	v = core.NewIntRangeValue(10, 0, 1)
	rng = (*core.IntRange)(v.Ptr)
	require.False(t, rng.Empty())
	require.Equal(t, int64(10), rng.Len())
	i, ok = rng.Get(0)
	require.True(t, ok)
	require.Equal(t, int64(10), i)
	i, ok = rng.Get(9)
	require.True(t, ok)
	require.Equal(t, int64(1), i)
	i, ok = rng.Get(10)
	require.False(t, ok)

	v = core.NewIntRangeValue(0, 10, 2)
	require.True(t, v.Type == core.VT_INT_RANGE)
	rng = (*core.IntRange)(v.Ptr)
	require.Equal(t, int64(0), rng.Start)
	require.Equal(t, int64(10), rng.Stop)
	require.Equal(t, int64(2), rng.Step)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == core.VT_INT_RANGE)
	rng = (*core.IntRange)(x.Ptr)
	require.Equal(t, int64(0), rng.Start)
	require.Equal(t, int64(10), rng.Stop)
	require.Equal(t, int64(2), rng.Step)
	require.Equal(t, true, v.Equal(x))
}

func TestObject_TypeName(t *testing.T) {
	var o core.Value

	o = core.IntValue(0)
	require.Equal(t, "int", o.TypeName())

	o = core.FloatValue(0)
	require.Equal(t, "float", o.TypeName())

	o = core.RuneValue(0)
	require.Equal(t, "rune", o.TypeName())

	o = core.NewStringValue("")
	require.Equal(t, "string", o.TypeName())

	o = core.False
	require.Equal(t, "bool", o.TypeName())

	o = core.NewArrayValue(nil, false)
	require.Equal(t, "array", o.TypeName())

	o = core.NewArrayValue(nil, true)
	require.Equal(t, "immutable-array", o.TypeName())

	o = core.NewRecordValue(nil, false)
	require.Equal(t, "record", o.TypeName())

	o = core.NewRecordValue(nil, true)
	require.Equal(t, "immutable-record", o.TypeName())

	o = core.NewDictValue(nil, false)
	require.Equal(t, "dict", o.TypeName())

	o = core.NewDictValue(nil, true)
	require.Equal(t, "immutable-dict", o.TypeName())

	o = core.NewBuiltinFunctionValue("fn", nil, 0, false)
	require.Equal(t, "<builtin-function:fn/0>", o.TypeName())

	o = core.Undefined
	require.Equal(t, "undefined", o.TypeName())

	o = core.NewErrorValue(core.Undefined)
	require.Equal(t, "error", o.TypeName())

	o = core.NewBytesValue(nil)
	require.Equal(t, "bytes", o.TypeName())

	o = core.NewIntRangeValue(1, 10, 1)
	require.Equal(t, "range", o.TypeName())
}

func TestObject_IsTrue(t *testing.T) {
	var o core.Value

	// 0 is false, non-zero is true
	o = core.IntValue(0)
	require.False(t, o.IsTrue())
	o = core.IntValue(1)
	require.True(t, o.IsTrue())
	o = core.IntValue(123)
	require.True(t, o.IsTrue())
	o = core.IntValue(-456)
	require.True(t, o.IsTrue())

	// NaN is false, non-NaN is true
	o = core.FloatValue(0)
	require.True(t, o.IsTrue())
	o = core.FloatValue(1)
	require.True(t, o.IsTrue())

	// non-zero char is true
	o = core.RuneValue(' ')
	require.True(t, o.IsTrue())
	o = core.RuneValue('T')
	require.True(t, o.IsTrue())

	// empty string is false, non-empty string is true
	o = core.NewStringValue("")
	require.False(t, o.IsTrue())
	o = core.NewStringValue(" ")
	require.True(t, o.IsTrue())

	// empty array is false, non-empty array is true
	o = core.NewArrayValue(nil, false)
	require.False(t, o.IsTrue())
	o = core.NewArrayValue([]core.Value{core.Undefined}, false)
	require.True(t, o.IsTrue())

	// empty record is false, non-empty record is true
	o = core.NewRecordValue(nil, false)
	require.False(t, o.IsTrue())
	o = core.NewRecordValue(map[string]core.Value{"a": core.Undefined}, false)
	require.True(t, o.IsTrue())

	// undefined is false
	o = core.Undefined
	require.False(t, o.IsTrue())

	// error is false
	o = core.NewErrorValue(core.Undefined)
	require.False(t, o.IsTrue())

	// empty bytes is false, non-empty bytes is true
	o = core.NewBytesValue(nil)
	require.False(t, o.IsTrue())
	o = core.NewBytesValue([]byte{1, 2})
	require.True(t, o.IsTrue())

	// empty range is false, non-empty range is true
	o = core.NewIntRangeValue(0, 0, 1)
	require.False(t, o.IsTrue())
	o = core.NewIntRangeValue(0, 10, 1)
	require.True(t, o.IsTrue())
}

func TestObject_String(t *testing.T) {
	var o core.Value

	o = core.IntValue(0)
	require.Equal(t, "0", o.String())

	o = core.IntValue(1)
	require.Equal(t, "1", o.String())

	o = core.FloatValue(0)
	require.Equal(t, "0", o.String())

	o = core.FloatValue(1)
	require.Equal(t, "1", o.String())

	o = core.RuneValue(' ')
	require.Equal(t, "' '", o.String())

	o = core.RuneValue('T')
	require.Equal(t, "'T'", o.String())

	o = core.NewStringValue("")
	require.Equal(t, `""`, o.String())

	o = core.NewStringValue(" ")
	require.Equal(t, `" "`, o.String())

	o = core.NewArrayValue(nil, false)
	require.Equal(t, "[]", o.String())

	o = core.NewRecordValue(nil, false)
	require.Equal(t, "{}", o.String())

	o = core.NewErrorValue(core.Undefined)
	require.Equal(t, "error(undefined)", o.String())

	o = core.NewErrorValue(core.NewStringValue("error 1"))
	require.Equal(t, `error("error 1")`, o.String())

	o = core.Undefined
	require.Equal(t, "undefined", o.String())

	o = core.NewBytesValue(nil)
	require.Equal(t, "bytes([])", o.String())

	o = core.NewBytesValue([]byte("foo"))
	require.Equal(t, "bytes([102, 111, 111])", o.String())

	o = core.NewIntRangeValue(0, 10, 2)
	require.Equal(t, "range(0, 10, 2)", o.String())
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
	require.True(t, err1.Equal(err2))
	require.True(t, err2.Equal(err1))

	err2 = core.NewErrorValue(core.NewStringValue("some error"))
	require.True(t, err1.Equal(err2))
	require.True(t, err2.Equal(err1))

	err2 = core.NewErrorValue(core.NewStringValue("some error 2"))
	require.False(t, err1.Equal(err2))
	require.False(t, err2.Equal(err1))

	range1 := core.NewIntRangeValue(0, 10, 2)
	range2 := core.NewIntRangeValue(0, 10, 2)
	range3 := core.NewIntRangeValue(0, 10, 1)
	require.True(t, range1.Equal(range2))
	require.True(t, range2.Equal(range1))
	require.False(t, range1.Equal(range3))
	require.False(t, range3.Equal(range1))

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

	bytes1 := core.NewBytesValue([]byte("foo"))
	bytes2 := core.NewBytesValue([]byte("foo"))
	bytes3 := core.NewBytesValue([]byte("bar"))

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
	require.False(t, bool1.Equal(core.Undefined))
	require.False(t, char1.Equal(core.Undefined))
	require.False(t, int1.Equal(core.Undefined))
	require.False(t, float1.Equal(core.Undefined))
	require.False(t, string1.Equal(core.Undefined))
	require.False(t, bytes1.Equal(core.Undefined))
	require.False(t, array1.Equal(core.Undefined))
	require.False(t, map1.Equal(core.Undefined))
	require.False(t, record1.Equal(core.Undefined))

	// compare to equal
	require.True(t, bool1.Equal(bool2))
	require.True(t, char1.Equal(char2))
	require.True(t, int1.Equal(int2))
	require.True(t, float1.Equal(float2))
	require.True(t, string1.Equal(string2))
	require.True(t, bytes1.Equal(bytes2))
	require.True(t, array1.Equal(array2))
	require.True(t, map1.Equal(map2))
	require.True(t, record1.Equal(record2))

	// compare to not equal
	require.False(t, bool1.Equal(bool3))
	require.False(t, char1.Equal(char3))
	require.False(t, int1.Equal(int3))
	require.False(t, float1.Equal(float3))
	require.False(t, string1.Equal(string3))
	require.False(t, bytes1.Equal(bytes3))
	require.False(t, array1.Equal(array3))
	require.False(t, map1.Equal(map3))
	require.False(t, record1.Equal(record3))
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
	err := m.Assign(k, v)

	require.NoError(t, err)

	res, err := m.Access(vm, k, core.OpIndex)
	require.NoError(t, err)
	require.Equal(t, v, res)
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
	require.Equal(t, expected, actual)
}

func boolValue(b bool) core.Value {
	return core.BoolValue(b)
}
