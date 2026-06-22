package kavun_test

import (
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/require"
)

func testBinaryOp(t *testing.T, lhs core.Value, op token.Token, rhs core.Value, expected core.Value) {
	t.Helper()
	actual, err := lhs.BinaryOp(op, rhs)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func boolValue(b bool) core.Value {
	return core.BoolValue(b)
}

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
	require.True(t, v.Type == value.Undefined)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Undefined)
	require.Equal(t, true, v.Equal(x))

	// Bool
	v = core.True
	require.True(t, v.Type == value.Bool)
	require.Equal(t, true, v.Data != 0)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Bool)
	require.Equal(t, true, x.Data != 0)
	require.Equal(t, true, v.Equal(x))

	v = core.False
	require.True(t, v.Type == value.Bool)
	require.Equal(t, false, v.Data != 0)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Bool)
	require.Equal(t, false, x.Data != 0)
	require.Equal(t, true, v.Equal(x))

	// Byte
	v = core.ByteValue(123)
	require.True(t, v.Type == value.Byte)
	require.Equal(t, byte(123), byte(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Byte)
	require.Equal(t, byte(123), byte(x.Data))
	require.Equal(t, true, v.Equal(x))

	// Rune
	v = core.RuneValue('A')
	require.True(t, v.Type == value.Rune)
	require.Equal(t, 'A', rune(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Rune)
	require.Equal(t, 'A', rune(x.Data))
	require.Equal(t, true, v.Equal(x))

	v = core.RuneValue('₴')
	require.True(t, v.Type == value.Rune)
	require.Equal(t, '₴', rune(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Rune)
	require.Equal(t, '₴', rune(x.Data))
	require.Equal(t, true, v.Equal(x))

	// Int
	v = core.IntValue(123)
	require.True(t, v.Type == value.Int)
	require.Equal(t, int64(123), int64(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Int)
	require.Equal(t, int64(123), int64(x.Data))
	require.Equal(t, true, v.Equal(x))

	v = core.IntValue(-456)
	require.True(t, v.Type == value.Int)
	require.Equal(t, int64(-456), int64(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Int)
	require.Equal(t, int64(-456), int64(x.Data))
	require.Equal(t, true, v.Equal(x))

	// Float
	v = core.FloatValue(3.14)
	require.True(t, v.Type == value.Float)
	require.Equal(t, 3.14, math.Float64frombits(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Float)
	require.Equal(t, 3.14, math.Float64frombits(x.Data))
	require.Equal(t, true, v.Equal(x))

	v = core.FloatValue(-2.71828)
	require.True(t, v.Type == value.Float)
	require.Equal(t, -2.71828, math.Float64frombits(v.Data))
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Float)
	require.Equal(t, -2.71828, math.Float64frombits(x.Data))
	require.Equal(t, true, v.Equal(x))

	// Decimal
	v = core.NewDecimalValue(dec128.FromString("3.14"))
	require.True(t, v.Type == value.Decimal)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Decimal)
	require.Equal(t, true, v.Equal(x))

	// String
	v = core.NewStringValue("")
	require.True(t, v.Type == value.String)
	s, _ = v.AsString()
	require.Equal(t, "", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.String)
	s, _ = x.AsString()
	require.Equal(t, "", s)
	require.Equal(t, true, v.Equal(x))

	v = core.NewStringValue("hello")
	require.True(t, v.Type == value.String)
	s, _ = v.AsString()
	require.Equal(t, "hello", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.String)
	s, _ = x.AsString()
	require.Equal(t, "hello", s)
	require.Equal(t, true, v.Equal(x))

	// Runes
	v = core.NewRunesValue([]rune(""), false)
	require.True(t, v.Type == value.Runes)
	s, _ = v.AsString()
	require.Equal(t, "", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Runes)
	s, _ = x.AsString()
	require.Equal(t, "", s)
	require.Equal(t, true, v.Equal(x))

	v = core.NewRunesValue([]rune("путін хуйло"), false)
	require.True(t, v.Type == value.Runes)
	s, _ = v.AsString()
	require.Equal(t, "путін хуйло", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Runes)
	s, _ = x.AsString()
	require.Equal(t, "путін хуйло", s)
	require.Equal(t, true, v.Equal(x))

	// Bytes
	v = core.NewBytesValue([]byte{}, false)
	require.True(t, v.Type == value.Bytes)
	b, _ := v.AsBytes()
	require.Equal(t, []byte{}, b)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Bytes)
	b, _ = x.AsBytes()
	require.Equal(t, []byte{}, b)
	require.Equal(t, true, v.Equal(x))

	v = core.NewBytesValue([]byte("foo"), false)
	require.True(t, v.Type == value.Bytes)
	b, _ = v.AsBytes()
	require.Equal(t, []byte("foo"), b)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Bytes)
	b, _ = x.AsBytes()
	require.Equal(t, []byte("foo"), b)
	require.Equal(t, true, v.Equal(x))

	// Array
	v = core.NewArrayValue([]core.Value{}, false)
	require.True(t, v.Type == value.Array)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Array)
	require.Equal(t, true, v.Equal(x))

	v = core.NewArrayValue([]core.Value{}, true)
	require.True(t, v.Type == value.Array)
	require.True(t, v.Immutable)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Array)
	require.True(t, x.Immutable)
	require.Equal(t, true, v.Equal(x))

	v = core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	require.True(t, v.Type == value.Array)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Array)
	require.Equal(t, true, v.Equal(x))

	// Record
	v = core.NewRecordValue(map[string]core.Value{}, true)
	require.True(t, v.Type == value.Record)
	require.True(t, v.Immutable)
	require.True(t, v.Immutable)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Record)
	require.True(t, x.Immutable)
	require.True(t, x.Immutable)
	require.Equal(t, true, v.Equal(x))

	v = core.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.Type == value.Record)
	require.False(t, v.Immutable)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Record)
	require.False(t, x.Immutable)
	require.Equal(t, true, v.Equal(x))

	// Map
	v = core.NewDictValue(map[string]core.Value{}, true)
	require.True(t, v.Type == value.Dict)
	require.True(t, v.Immutable)
	require.True(t, v.Immutable)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Dict)
	require.True(t, x.Immutable)
	require.True(t, x.Immutable)
	require.Equal(t, true, v.Equal(x))

	v = core.NewDictValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.Type == value.Dict)
	require.False(t, v.Immutable)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Dict)
	require.False(t, x.Immutable)
	require.Equal(t, true, v.Equal(x))

	// Error
	v = core.NewErrorValue(core.Undefined, core.KindUser, false)
	require.True(t, v.Type == value.Error)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Error)
	require.Equal(t, true, v.Equal(x))

	x = core.NewStringValue("some error")
	v = core.NewErrorValue(x, core.KindUser, false)
	require.True(t, v.Type == value.Error)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Error)
	require.Equal(t, true, v.Equal(x))

	// Time
	v = core.NewTimeValue(time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC))
	require.True(t, v.Type == value.Time)
	tm, _ := v.AsTime()
	require.Equal(t, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.Time)
	tm, _ = x.AsTime()
	require.Equal(t, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	require.Equal(t, true, v.Equal(x))

	// IntRange
	v = core.NewIntRangeValue(0, 0, 1)
	require.True(t, v.Type == value.IntRange)
	rng := (*core.IntRange)(v.Ptr)
	require.True(t, rng.Empty())
	require.Equal(t, int64(0), rng.Len())
	v = core.NewIntRangeValue(0, 10, 1)
	require.NoError(t, err)
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
	require.True(t, v.Type == value.IntRange)
	rng = (*core.IntRange)(v.Ptr)
	require.Equal(t, int64(0), rng.Start)
	require.Equal(t, int64(10), rng.Stop)
	require.Equal(t, int64(2), rng.Step)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.Type == value.IntRange)
	rng = (*core.IntRange)(x.Ptr)
	require.Equal(t, int64(0), rng.Start)
	require.Equal(t, int64(10), rng.Stop)
	require.Equal(t, int64(2), rng.Step)
	require.Equal(t, true, v.Equal(x))
}

func TestObject_TypeName(t *testing.T) {
	var err error
	var o core.Value

	o = core.IntValue(0)
	require.Equal(t, "int", o.TypeName())

	o = core.FloatValue(0)
	require.Equal(t, "float", o.TypeName())

	o = core.ByteValue(0)
	require.Equal(t, "byte", o.TypeName())

	o = core.RuneValue(0)
	require.Equal(t, "rune", o.TypeName())

	o = core.NewStringValue("")
	require.NoError(t, err)
	require.Equal(t, "string", o.TypeName())

	o = core.False
	require.Equal(t, "bool", o.TypeName())

	o = core.NewArrayValue(nil, false)
	require.NoError(t, err)
	require.Equal(t, "array", o.TypeName())

	o = core.NewArrayValue(nil, true)
	require.NoError(t, err)
	require.Equal(t, "immutable-array", o.TypeName())

	o = core.NewRecordValue(nil, false)
	require.NoError(t, err)
	require.Equal(t, "record", o.TypeName())

	o = core.NewRecordValue(nil, true)
	require.NoError(t, err)
	require.Equal(t, "immutable-record", o.TypeName())

	o = core.NewDictValue(nil, false)
	require.NoError(t, err)
	require.Equal(t, "dict", o.TypeName())

	o = core.NewDictValue(nil, true)
	require.NoError(t, err)
	require.Equal(t, "immutable-dict", o.TypeName())

	o = core.NewBuiltinClosureValue("fn", nil, 0, false)
	require.NoError(t, err)
	require.Equal(t, "<builtin-closure:fn/0>", o.TypeName())

	o = core.Undefined
	require.Equal(t, "undefined", o.TypeName())

	o = core.NewErrorValue(core.Undefined, core.KindUser, false)
	require.NoError(t, err)
	require.Equal(t, "error", o.TypeName())

	o = core.NewBytesValue(nil, false)
	require.NoError(t, err)
	require.Equal(t, "bytes", o.TypeName())

	o = core.NewIntRangeValue(1, 10, 1)
	require.NoError(t, err)
	require.Equal(t, "range", o.TypeName())
}

func TestObject_IsTrue(t *testing.T) {
	var err error
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
	require.NoError(t, err)
	require.False(t, o.IsTrue())
	o = core.NewStringValue(" ")
	require.NoError(t, err)
	require.True(t, o.IsTrue())

	// empty array is false, non-empty array is true
	o = core.NewArrayValue(nil, false)
	require.NoError(t, err)
	require.False(t, o.IsTrue())
	o = core.NewArrayValue([]core.Value{core.Undefined}, false)
	require.NoError(t, err)
	require.True(t, o.IsTrue())

	// empty record is false, non-empty record is true
	o = core.NewRecordValue(nil, false)
	require.NoError(t, err)
	require.False(t, o.IsTrue())
	o = core.NewRecordValue(map[string]core.Value{"a": core.Undefined}, false)
	require.NoError(t, err)
	require.True(t, o.IsTrue())

	// undefined is false
	o = core.Undefined
	require.False(t, o.IsTrue())

	// error is false
	o = core.NewErrorValue(core.Undefined, core.KindUser, false)
	require.NoError(t, err)
	require.False(t, o.IsTrue())

	// empty bytes is false, non-empty bytes is true
	o = core.NewBytesValue(nil, false)
	require.NoError(t, err)
	require.False(t, o.IsTrue())
	o = core.NewBytesValue([]byte{1, 2}, false)
	require.NoError(t, err)
	require.True(t, o.IsTrue())

	// empty range is false, non-empty range is true
	o = core.NewIntRangeValue(0, 0, 1)
	require.NoError(t, err)
	require.False(t, o.IsTrue())
	o = core.NewIntRangeValue(0, 10, 1)
	require.NoError(t, err)
	require.True(t, o.IsTrue())

	// byte
	o = core.ByteValue(0)
	require.False(t, o.IsTrue())
	o = core.ByteValue(123)
	require.True(t, o.IsTrue())
}

func TestObject_String(t *testing.T) {
	var err error
	var o core.Value
	var x core.Value

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
	require.NoError(t, err)
	require.Equal(t, `""`, o.String())

	o = core.NewStringValue(" ")
	require.NoError(t, err)
	require.Equal(t, `" "`, o.String())

	o = core.NewArrayValue(nil, false)
	require.NoError(t, err)
	require.Equal(t, "[]", o.String())

	o = core.NewRecordValue(nil, false)
	require.NoError(t, err)
	require.Equal(t, "{}", o.String())

	o = core.NewErrorValue(core.Undefined, core.KindUser, false)
	require.NoError(t, err)
	require.Equal(t, "error()", o.String())

	x = core.NewStringValue("error 1")
	require.NoError(t, err)
	o = core.NewErrorValue(x, core.KindUser, false)
	require.NoError(t, err)
	require.Equal(t, `error("error 1")`, o.String())

	o = core.Undefined
	require.Equal(t, "undefined", o.String())

	o = core.NewBytesValue(nil, false)
	require.NoError(t, err)
	require.Equal(t, "bytes([])", o.String())

	o = core.NewBytesValue([]byte("foo"), false)
	require.NoError(t, err)
	require.Equal(t, "bytes([102, 111, 111])", o.String())

	o = core.NewIntRangeValue(0, 10, 2)
	require.NoError(t, err)
	require.Equal(t, "range(0, 10, 2)", o.String())
}

func TestObject_BinaryOp(t *testing.T) {
	var err error
	var o core.Value

	o = core.RuneValue(0)
	_, err = o.BinaryOp(token.Add, core.Undefined)
	require.Error(t, err)

	o = core.False
	_, err = o.BinaryOp(token.Add, core.Undefined)
	require.Error(t, err)

	o = core.NewRecordValue(nil, false)
	require.NoError(t, err)
	_, err = o.BinaryOp(token.Add, core.Undefined)
	require.Error(t, err)

	o = core.Undefined
	_, err = o.BinaryOp(token.Add, core.Undefined)
	require.Error(t, err)

	o = core.NewErrorValue(core.Undefined, core.KindUser, false)
	require.NoError(t, err)
	_, err = o.BinaryOp(token.Add, core.Undefined)
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
	err1 := core.NewErrorValue(core.NewStringValue("some error"), core.KindUser, false)
	err2 := err1
	require.True(t, err1.Equal(err2))
	require.True(t, err2.Equal(err1))

	err2 = core.NewErrorValue(core.NewStringValue("some error"), core.KindUser, false)
	require.True(t, err1.Equal(err2))
	require.True(t, err2.Equal(err1))

	err2 = core.NewErrorValue(core.NewStringValue("some error 2"), core.KindUser, false)
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

	res, err := m.Access(k, opcode.Index)
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

func TestFormatErrorValue(t *testing.T) {
	mkErr := func(msg string) core.Value {
		return core.NewErrorValue(core.NewStringValue(msg), core.KindUser, false)
	}

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default verb -> message text
		{"default", mkErr("boom"), "", "boom", false},
		{"default empty msg", mkErr(""), "", "", false},

		// 'v' verb -> source form
		{"v form", mkErr("boom"), "v", `error("boom")`, false},

		// 'T' universal type-name verb
		{"T", mkErr("x"), "T", "error", false},

		// generic fields with default verb (left-align by default)
		{"width left default", mkErr("err"), "10", "err       ", false},
		{"width right", mkErr("err"), ">10", "       err", false},
		{"width center", mkErr("err"), "^7", "  err  ", false},
		{"fill+align", mkErr("err"), "*<6", "err***", false},
		{"v ignores width", mkErr("x"), ">12v", `error("x")`, false},

		// no truncation: width below body length is a no-op
		{"width too small", mkErr("hello"), "3", "hello", false},

		// unsupported: any other generic verb
		{"verb d", mkErr("x"), "d", "", true},
		{"verb s", mkErr("x"), "s", "", true},
		{"verb q", mkErr("x"), "q", "", true},

		// unsupported: tail form ('#'-tail sets Verb='#')
		{"tail empty", mkErr("x"), "#", "", true},
		{"tail payload", mkErr("x"), "#anything", "", true},
		{"tail with width", mkErr("x"), "10#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatBoolValue(t *testing.T) {
	T := core.True
	F := core.False

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default verb
		{"default true", T, "", "true", false},
		{"default false", F, "", "false", false},

		// 't' / 'v' verbs == default
		{"t true", T, "t", "true", false},
		{"t false", F, "t", "false", false},
		{"v true", T, "v", "true", false},
		{"v false", F, "v", "false", false},

		// 'd'
		{"d true", T, "d", "1", false},
		{"d false", F, "d", "0", false},

		// 'T' is the universal type-name verb
		{"T true", T, "T", "bool", false},
		{"T false", F, "T", "bool", false},

		// generic width / fill / align (non-numeric defaults to left)
		{"width default left", T, "8", "true    ", false},
		{"width right", T, ">8", "    true", false},
		{"width center", F, "^7", " false ", false},
		{"fill+align", F, "*<7", "false**", false},
		{"width on T", T, ">6T", "  bool", false},
		{"width on d left", F, "3d", "0  ", false},
		{"width too small", T, "2t", "true", false},

		// unsupported verbs
		{"verb s", T, "s", "", true},
		{"verb b", T, "b", "", true},
		{"verb x", T, "x", "", true},
		{"verb y", T, "y", "", true},
		{"verb Y", T, "Y", "", true},

		// tail form unsupported
		{"tail empty", T, "#", "", true},
		{"tail payload", F, "#anything", "", true},
		{"tail with width", T, "5#x", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatByteValue(t *testing.T) {
	bv := func(b byte) core.Value { return core.ByteValue(b) }

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default / d / v
		{"default 0", bv(0), "", "0", false},
		{"default 42", bv(42), "", "42", false},
		{"default 255", bv(255), "", "255", false},
		{"d 42", bv(42), "d", "42", false},
		{"v 42", bv(42), "v", "byte(42)", false},
		{"T", bv(42), "T", "byte", false},

		// sign on non-negative
		{"+ d", bv(5), "+d", "+5", false},
		{"space d", bv(5), " d", " 5", false},
		{"- d (no-op for byte)", bv(5), "-d", "5", false},
		{"+0", bv(0), "+", "+0", false},

		// width / right-align (numeric default)
		{"width 5", bv(7), "5d", "    7", false},
		{"width <", bv(7), "<5d", "7    ", false},
		{"width ^", bv(7), "^5d", "  7  ", false},

		// zero-pad shortcut
		{"05d", bv(7), "05d", "00007", false},
		{"+05d", bv(7), "+05d", "+0007", false},
		{" 05d", bv(7), " 05d", " 0007", false},
		{"05x prefix", bv(0xab), "#06x", "", true}, // generic verb + tail forbidden by parser
		{"06x", bv(0xab), "06x", "0x00ab", false},  // sign-aware split keeps prefix

		// grouping (decimal)
		{"grouping ,", bv(255), ",d", "255", false},
		{"grouping , width", bv(255), "10,d", "       255", false},

		// hex / oct / bin
		{"x 0", bv(0), "x", "0x0", false},
		{"x 255", bv(255), "x", "0xff", false},
		{"X 255", bv(255), "X", "0xFF", false},
		{"o 8", bv(8), "o", "0o10", false},
		{"b 5", bv(5), "b", "0b101", false},
		{"b 255", bv(255), "b", "0b11111111", false},

		// '!' bare flag — suppresses the conventional prefix
		{"! x", bv(255), "!x", "ff", false},
		{"! X", bv(255), "!X", "FF", false},
		{"! o", bv(8), "!o", "10", false},
		{"! b", bv(5), "!b", "101", false},
		{"! x width", bv(255), "8!x", "      ff", false},
		{"! x zero-pad", bv(255), "08!x", "000000ff", false},

		// grouping '_' for non-decimal (every 4 digits)
		{"b _ 255", bv(255), "_b", "0b1111_1111", false},
		{"x _ 255", bv(255), "_x", "0xff", false}, // only 2 hex digits, no grouping triggered

		// 'c' verb (ASCII char)
		{"c A", bv('A'), "c", "A", false},
		{"c width", bv('A'), "3c", "A  ", false},

		// errors
		{"precision", bv(1), ".2d", "", true},
		{"~ flag", bv(1), "~d", "", true},
		{"! on d", bv(1), "!d", "", true},
		{"! on c", bv('A'), "!c", "", true},
		{"comma on hex", bv(255), ",x", "", true},
		{"sign on c", bv('A'), "+c", "", true},
		{"grouping on c", bv('A'), "_c", "", true},
		{"q byte", bv('A'), "q", "'A'", false},
		{"q byte newline", bv('\n'), "q", "'\\n'", false},
		{"unknown verb", bv(1), "z", "", true},

		// tail form unsupported (verb == '#')
		{"tail empty", bv(1), "#", "", true},
		{"tail payload", bv(1), "#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return // parser already rejected (e.g. "#06x")
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatRuneValue(t *testing.T) {
	rv := func(r rune) core.Value { return core.RuneValue(r) }

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default / 'c'
		{"default A", rv('A'), "", "A", false},
		{"default snowman", rv(0x2603), "", "\u2603", false},
		{"c A", rv('A'), "c", "A", false},
		{"c snowman", rv(0x2603), "c", "\u2603", false},

		// 'd'
		{"d A", rv('A'), "d", "65", false},
		{"d snowman", rv(0x2603), "d", "9731", false},
		{"d sign +", rv('A'), "+d", "+65", false},
		{"d zero-pad", rv('A'), "05d", "00065", false},
		{"d width right", rv('A'), "5d", "   65", false},
		{"d grouping ,", rv(0x2603), ",d", "9,731", false},
		{"d grouping _", rv(0x2603), "_d", "9_731", false},

		// 'x' / 'X' (no 0x prefix per spec)
		{"x A", rv('A'), "x", "41", false},
		{"X A", rv('A'), "X", "41", false},
		{"x snowman", rv(0x2603), "x", "2603", false},
		{"X snowman", rv(0x2603), "X", "2603", false},
		{"x lowercase ff", rv(0xff), "x", "ff", false},
		{"X uppercase FF", rv(0xff), "X", "FF", false},
		{"x grouping _", rv(0x12345), "_x", "1_2345", false},
		{"x width zero-pad", rv('A'), "06x", "000041", false},

		// 'U'
		{"U A", rv('A'), "U", "U+0041", false},
		{"U snowman", rv(0x2603), "U", "U+2603", false},
		{"U high", rv(0x1F600), "U", "U+1F600", false},
		{"U width", rv('A'), "10U", "    U+0041", false},

		// 'q' / 'v'
		{"q A", rv('A'), "q", `'A'`, false},
		{"q tab", rv('\t'), "q", `'\t'`, false},
		{"v A", rv('A'), "v", `'A'`, false},
		{"q width", rv('A'), "5q", `'A'  `, false},
		{"T", rv('A'), "T", "rune", false},

		// width / fill / align on default char
		{"c width", rv('A'), "5", "A    ", false},
		{"c right", rv('A'), ">5", "    A", false},
		{"c center", rv('A'), "*^5", "**A**", false},

		// errors
		{"precision", rv('A'), ".2c", "", true},
		{"z flag", rv('A'), "zd", "", true},
		{"comma on x", rv('A'), ",x", "", true},
		{"sign on c", rv('A'), "+c", "", true},
		{"grouping on c", rv('A'), "_c", "", true},
		{"sign on q", rv('A'), "+q", "", true},
		{"sign on U", rv('A'), "+U", "", true},
		{"zeropad on U", rv('A'), "08U", "", true},
		{"unknown verb", rv('A'), "k", "", true},

		// tail unsupported
		{"tail empty", rv('A'), "#", "", true},
		{"tail payload", rv('A'), "#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatIntValue(t *testing.T) {
	iv := func(i int64) core.Value { return core.IntValue(i) }

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default / d / v
		{"default 0", iv(0), "", "0", false},
		{"default 42", iv(42), "", "42", false},
		{"default -7", iv(-7), "", "-7", false},
		{"d 42", iv(42), "d", "42", false},
		{"v -7", iv(-7), "v", "-7", false},
		{"T", iv(0), "T", "int", false},
		{"min int64", iv(math.MinInt64), "d", "-9223372036854775808", false},
		{"max int64", iv(math.MaxInt64), "d", "9223372036854775807", false},

		// sign
		{"+ pos", iv(5), "+d", "+5", false},
		{"+ neg", iv(-5), "+d", "-5", false},
		{"space pos", iv(5), " d", " 5", false},
		{"space neg", iv(-5), " d", "-5", false},
		{"- pos", iv(5), "-d", "5", false},
		{"+ zero", iv(0), "+", "+0", false},

		// width / align
		{"width 5", iv(7), "5d", "    7", false},
		{"width 5 neg", iv(-7), "5d", "   -7", false},
		{"left", iv(7), "<5d", "7    ", false},
		{"center", iv(7), "^5d", "  7  ", false},
		{"sign-aware", iv(-7), "=5d", "-   7", false},

		// zero-pad
		{"05d pos", iv(7), "05d", "00007", false},
		{"05d neg", iv(-7), "05d", "-0007", false},
		{"+05d", iv(7), "+05d", "+0007", false},
		{"06x", iv(0xab), "06x", "0x00ab", false},
		{"06x neg", iv(-1), "06x", "-0x001", false},

		// grouping decimal
		{"comma", iv(1234567), ",d", "1,234,567", false},
		{"underscore", iv(1234567), "_d", "1_234_567", false},
		{"comma neg", iv(-1234567), ",d", "-1,234,567", false},
		{"comma width", iv(1234), "10,d", "     1,234", false},

		// hex / oct / bin
		{"x 255", iv(255), "x", "0xff", false},
		{"X 255", iv(255), "X", "0xFF", false},
		{"o 8", iv(8), "o", "0o10", false},
		{"b 5", iv(5), "b", "0b101", false},

		// '!' bare flag — suppresses the conventional prefix
		{"! x", iv(255), "!x", "ff", false},
		{"! X", iv(255), "!X", "FF", false},
		{"! o", iv(0o755), "!o", "755", false},
		{"! b", iv(5), "!b", "101", false},
		{"! x grouping", iv(0xdeadbeef), "_!x", "dead_beef", false},
		{"! x width", iv(255), "8!x", "      ff", false},
		{"! x zero-pad", iv(255), "08!x", "000000ff", false},
		{"! x neg", iv(-1), "!x", "-1", false},

		// grouping '_' on non-decimal
		{"x _", iv(0xdeadbeef), "_x", "0xdead_beef", false},
		{"b _", iv(0xff), "_b", "0b1111_1111", false},

		// 'c' verb
		{"c A", iv('A'), "c", "A", false},
		{"c snowman", iv(0x2603), "c", "\u2603", false},
		{"c width", iv('A'), "3c", "A  ", false},

		// errors
		{"precision", iv(1), ".2d", "", true},
		{"~ flag", iv(1), "~d", "", true},
		{"! on d", iv(1), "!d", "", true},
		{"! on c", iv('A'), "!c", "", true},
		{"comma on hex", iv(255), ",x", "", true},
		{"sign on c", iv('A'), "+c", "", true},
		{"grouping on c", iv('A'), "_c", "", true},
		{"c negative", iv(-1), "c", "", true},
		{"c too large", iv(0x110000), "c", "", true},
		{"q rune", iv('A'), "q", "'A'", false},
		{"q tab", iv('\t'), "q", "'\\t'", false},
		{"q negative", iv(-1), "q", "", true},
		{"q too large", iv(0x110000), "q", "", true},
		{"unknown verb", iv(1), "z", "", true},

		// tail unsupported
		{"tail empty", iv(1), "#", "", true},
		{"tail payload", iv(1), "#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatFloatValue(t *testing.T) {
	fv := func(f float64) core.Value { return core.FloatValue(f) }

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default ('g') / 'v'
		{"default 1.5", fv(1.5), "", "1.5", false},
		{"default 0", fv(0), "", "0", false},
		{"v 1.5", fv(1.5), "v", "1.5", false},
		{"default neg", fv(-2.5), "", "-2.5", false},
		{"T", fv(0), "T", "float", false},

		// 'f'
		{"f default prec", fv(1.5), "f", "1.500000", false},
		{"f prec 2", fv(1.5), ".2f", "1.50", false},
		{"f prec 0", fv(1.5), ".0f", "2", false},
		{"f neg", fv(-3.14), ".2f", "-3.14", false},

		// 'e' / 'E'
		{"e default", fv(12345.6789), "e", "1.234568e+04", false},
		{"e prec 2", fv(12345.6789), ".2e", "1.23e+04", false},
		{"E prec 2", fv(12345.6789), ".2E", "1.23E+04", false},

		// 'g' / 'G'
		{"g 1234567.89", fv(1234567.89), "g", "1.23456789e+06", false},
		{"G 1234567.89", fv(1234567.89), "G", "1.23456789E+06", false},

		// '%'
		{"% default", fv(0.5), "%", "50.000000%", false},
		{"% prec 1", fv(0.125), ".1%", "12.5%", false},
		{"% neg", fv(-0.25), ".0%", "-25%", false},

		// sign
		{"+ pos", fv(1.5), "+f", "+1.500000", false},
		{"+ neg", fv(-1.5), "+f", "-1.500000", false},
		{"space pos", fv(1.5), " f", " 1.500000", false},

		// width / align
		{"width 10", fv(1.5), "10f", "  1.500000", false},
		{"left", fv(1.5), "<10f", "1.500000  ", false},
		{"center", fv(1.5), "^10f", " 1.500000 ", false},

		// zero-pad / sign-aware
		{"0 width", fv(1.5), "010.2f", "0000001.50", false},
		{"+0 width", fv(1.5), "+010.2f", "+000001.50", false},
		{"0 width neg", fv(-1.5), "010.2f", "-000001.50", false},

		// grouping
		{"comma f", fv(1234567.89), ",.2f", "1,234,567.89", false},
		{"underscore f", fv(1234567.89), "_.2f", "1_234_567.89", false},
		{"comma neg", fv(-1234.5), ",.1f", "-1,234.5", false},
		{"comma g", fv(1234567), ",.0f", "1,234,567", false},

		// '~' coerce-zero
		{"~ neg zero f", fv(-0.0), "~f", "0.000000", false},
		{"~ rounds to zero", fv(-0.0001), ".2~f", "0.00", false},
		{"~ without -0", fv(-1.5), ".1~f", "-1.5", false},
		{"~ neg-zero g", fv(-0.0), "~g", "0", false},

		// special values
		{"NaN f", fv(math.NaN()), "f", "NaN", false},
		{"NaN F", fv(math.NaN()), "F", "NAN", false},
		{"+Inf", fv(math.Inf(1)), "f", "Inf", false},
		{"-Inf", fv(math.Inf(-1)), "f", "-Inf", false},
		{"+Inf upper", fv(math.Inf(1)), "F", "INF", false},
		{"+Inf with +", fv(math.Inf(1)), "+f", "+Inf", false},
		{"NaN with +", fv(math.NaN()), "+f", "NaN", false},
		{"NaN width", fv(math.NaN()), "5f", "  NaN", false},

		// errors
		{"unknown verb", fv(1), "x", "", true},
		{"unknown verb d", fv(1), "d", "", true},

		// tail unsupported
		{"tail empty", fv(1), "#", "", true},
		{"tail payload", fv(1), "#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatDecimalValue(t *testing.T) {
	dv := func(str string) core.Value {
		d := dec128.FromString(str)
		return core.NewDecimalValue(d)
	}

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default (canonical, trim trailing zeros)
		{"default 1.23", dv("1.23"), "", "1.23", false},
		{"default 1.230", dv("1.230"), "", "1.23", false},
		{"default 0", dv("0"), "", "0", false},
		{"default neg", dv("-2.5"), "", "-2.5", false},
		{"default 100", dv("100"), "", "100", false},

		// 'v' source form
		{"v 1.23", dv("1.23"), "v", "1.23d", false},
		{"v -2.5", dv("-2.5"), "v", "-2.5d", false},
		{"v 1.230", dv("1.230"), "v", "1.23d", false}, // canonical underneath
		{"T", dv("1.0"), "T", "decimal", false},

		// 's' preserves source scale
		{"s 1.230", dv("1.230"), "s", "1.230", false},
		{"s 1.0", dv("1.0"), "s", "1.0", false},
		{"s int", dv("100"), "s", "100", false},

		// 'f' fixed precision (default 6)
		{"f default prec", dv("1.5"), "f", "1.500000", false},
		{"f prec 2", dv("1.5"), ".2f", "1.50", false},
		{"f prec 0", dv("1.5"), ".0f", "2", false},
		{"f rounds", dv("1.235"), ".2f", "1.24", false},
		{"f neg", dv("-3.14"), ".2f", "-3.14", false},

		// '%'
		{"% default", dv("0.5"), "%", "50.000000%", false},
		{"% prec 1", dv("0.125"), ".1%", "12.5%", false},
		{"% neg", dv("-0.25"), ".0%", "-25%", false},

		// 'e' / 'E' (via float64)
		{"e", dv("1234.5"), ".2e", "1.23e+03", false},
		{"E", dv("1234.5"), ".2E", "1.23E+03", false},

		// 'g' / 'G'
		{"g 1.5", dv("1.5"), "g", "1.5", false},
		{"G 1.5", dv("1.5"), "G", "1.5", false},

		// sign
		{"+ pos", dv("1.5"), "+", "+1.5", false},
		{"+ neg", dv("-1.5"), "+", "-1.5", false},
		{"space pos", dv("1.5"), " ", " 1.5", false},
		{"+ zero", dv("0"), "+", "+0", false},

		// width / align
		{"width 8", dv("1.5"), "8", "     1.5", false},
		{"left", dv("1.5"), "<8", "1.5     ", false},
		{"center", dv("1.5"), "^8", "  1.5   ", false},

		// zero-pad / sign-aware
		{"010.2f", dv("1.5"), "010.2f", "0000001.50", false},
		{"+010.2f", dv("1.5"), "+010.2f", "+000001.50", false},
		{"010.2f neg", dv("-1.5"), "010.2f", "-000001.50", false},

		// grouping
		{"comma f", dv("1234567.89"), ",.2f", "1,234,567.89", false},
		{"underscore f", dv("1234567.89"), "_.2f", "1_234_567.89", false},
		{"comma default", dv("1234567"), ",", "1,234,567", false},
		{"comma neg", dv("-1234.5"), ",.1f", "-1,234.5", false},
		{"comma s", dv("1234.50"), ",s", "1,234.50", false},

		// '~' coerce-zero
		{"~ neg-zero", dv("-0"), "~f", "0.000000", false},
		{"~ rounds to zero", dv("-0.001"), ".2~f", "0.00", false},
		{"~ without -0", dv("-1.5"), ".1~f", "-1.5", false},

		// NaN
		{"NaN default", dv("nope"), "f", "NaN", false},
		{"NaN F", dv("nope"), "F", "NAN", false},
		{"NaN width", dv("nope"), "5f", "  NaN", false},

		// errors
		{"unknown verb d", dv("1"), "d", "", true},
		{"unknown verb x", dv("1"), "x", "", true},

		// tail unsupported
		{"tail empty", dv("1"), "#", "", true},
		{"tail payload", dv("1"), "#foo", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				if !errors.Is(ferr, errs.ErrUnsupportedFormatSpec) {
					t.Fatalf("Format(%q): expected ErrUnsupportedFormatSpec, got %v", c.spec, ferr)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatTimeValue(t *testing.T) {
	loc := time.FixedZone("UTC-5", -5*3600)
	// 2026-03-04 13:05:09.123456 -0500 (Wed)
	tm := time.Date(2026, 3, 4, 13, 5, 9, 123456000, loc)
	tv := core.NewTimeValue(tm)

	cases := []struct {
		name    string
		spec    string
		want    string
		wantErr bool
	}{
		// default RFC 3339 (seconds precision; use #isonano for nanos)
		{"default", "", "2026-03-04T13:05:09-05:00", false},

		// 'v' source form
		{"v", "v", `time("2026-03-04T13:05:09.123456-05:00")`, false},

		// 'T' universal type-name verb
		{"T", "T", "time", false},

		// named tails
		{"#", "#", "2026-03-04T13:05:09-05:00", false},
		{"#iso", "#iso", "2026-03-04T13:05:09-05:00", false},
		{"#isonano", "#isonano", "2026-03-04T13:05:09.123456-05:00", false},
		{"#date", "#date", "2026-03-04", false},
		{"#time", "#time", "13:05:09", false},
		{"#unix", "#unix", strconv.FormatInt(tm.Unix(), 10), false},
		{"#unixms", "#unixms", strconv.FormatInt(tm.UnixMilli(), 10), false},
		{"#rfc822", "#rfc822", tm.Format(time.RFC822), false},

		// strftime: simple
		{"strftime ymd", "#%Y-%m-%d", "2026-03-04", false},
		{"strftime hms 24h", "#%H:%M:%S", "13:05:09", false},
		{"strftime hms 12h", "#%I:%M:%S %p", "01:05:09 PM", false},
		{"strftime y2", "#%y", "26", false},
		{"strftime e", "#[%e]", "[ 4]", false},
		{"strftime month names", "#%B / %b", "March / Mar", false},
		{"strftime weekday", "#%A %a", "Wednesday Wed", false},
		{"strftime jday", "#%j", "063", false},
		{"strftime tz", "#%z %Z", "-0500 UTC-5", false},
		{"strftime micro", "#%f", "123456", false},
		{"strftime literal pct", "#100%%", "100%", false},
		{"strftime newline tab", "#a%nb%tc", "a\nb\tc", false},
		{"strftime unix", "#%s", strconv.FormatInt(tm.Unix(), 10), false},
		{"strftime ISO week", "#%G-W%V-%u", "2026-W10-3", false},
		{"strftime weekday num", "#%w", "3", false},
		{"strftime century", "#%C", "20", false},

		// strftime: combined like the example in the task
		{"combined", "#%Y-%m-%d %H:%M:%S", "2026-03-04 13:05:09", false},

		// width/fill/align (default left)
		{"width left", "20#date", "2026-03-04          ", false},
		{"width right", ">20#date", "          2026-03-04", false},
		{"width center", "*^12#date", "*2026-03-04*", false},

		// errors: unsupported generic fields
		{"sign", "+", "", true},
		{"precision", ".3", "", true},
		{"zeropad", "010", "", true},
		{"grouping", ",", "", true},
		{"~ flag", "~", "", true},

		// errors: unknown verb
		{"verb d", "d", "", true},
		{"verb f", "f", "", true},

		// errors: bad strftime
		{"unknown directive", "#%Q", "", true},
		{"trailing pct", "#abc%", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := tv.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatStringValue(t *testing.T) {
	sv := core.NewStringValue("hello")
	mix := core.NewStringValue("h\u00e9llo") // 5 runes, 6 bytes
	withSpec := core.NewStringValue("a b/c") // for url-encode

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		// default + s + v + q
		{"default", sv, "", "hello", false},
		{"s", sv, "s", "hello", false},
		{"v", sv, "v", `"hello"`, false},
		{"q", sv, "q", `"hello"`, false},
		{"T", sv, "T", "string", false},
		{"q with newline", core.NewStringValue("a\nb"), "q", `"a\nb"`, false},

		// base64
		{"b std", sv, "b", "aGVsbG8=", false},
		{"B url no pad", sv, "B", "aGVsbG8", false},

		// hex
		{"x lower", sv, "x", "68656c6c6f", false},
		{"X upper", sv, "X", "68656C6C6F", false},

		// url component
		{"u", withSpec, "u", "a%20b%2Fc", false},
		{"u unreserved", core.NewStringValue("A-Z.a_z~0-9"), "u", "A-Z.a_z~0-9", false},

		// precision (rune-based)
		{"prec ascii", sv, ".3", "hel", false},
		{"prec multibyte", mix, ".3", "h\u00e9l", false},
		{"prec ge len", sv, ".10", "hello", false},
		{"prec on q", sv, ".3q", `"hel"`, false},

		// width / fill / align (default left)
		{"width left", sv, "10", "hello     ", false},
		{"width right", sv, ">10", "     hello", false},
		{"width center fill", sv, "*^9", "**hello**", false},
		{"width with prec", sv, "10.3", "hel       ", false},

		// 'v' ignores width
		{"v ignores width", sv, "10v", `"hello"`, false},

		// errors
		{"sign", sv, "+", "", true},
		{"zeropad", sv, "010", "", true},
		{"grouping comma", sv, ",", "", true},
		{"z flag", sv, "z", "", true},
		{"verb d", sv, "d", "", true},
		{"v with prec ignored", sv, ".3v", `"hello"`, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatRunesValue(t *testing.T) {
	rv := core.NewRunesValue([]rune("hello"), false)
	mix := core.NewRunesValue([]rune("h\u00e9llo"), false)

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		{"default", rv, "", "hello", false},
		{"s", rv, "s", "hello", false},
		{"v source form", rv, "v", `u"hello"`, false},
		{"q", rv, "q", `"hello"`, false},
		{"T", rv, "T", "runes", false},
		{"b", rv, "b", "aGVsbG8=", false},
		{"B", rv, "B", "aGVsbG8", false},
		{"x", rv, "x", "68656c6c6f", false},
		{"X", rv, "X", "68656C6C6F", false},
		{"u", core.NewRunesValue([]rune("a b"), false), "u", "a%20b", false},

		// precision counts runes, not bytes
		{"prec multibyte", mix, ".3", "h\u00e9l", false},
		{"prec on x sees full byte hex of truncated runes", mix, ".2x", "68c3a9", false},

		// width default left
		{"width", rv, "8", "hello   ", false},
		{"width right", rv, ">8", "   hello", false},

		// errors
		{"sign", rv, "-", "", true},
		{"zeropad", rv, "08", "", true},
		{"verb f", rv, "f", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatBytesValue(t *testing.T) {
	bv := core.NewBytesValue([]byte("hello"), false)
	mix := core.NewBytesValue([]byte("h\u00e9llo"), false) // 6 bytes
	bin := core.NewBytesValue([]byte{0x00, 0xff, 0x10}, false)

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		{"default", bv, "", "hello", false},
		{"s", bv, "s", "hello", false},
		{"v source form", bv, "v", `bytes([104, 101, 108, 108, 111])`, false},
		{"q", bv, "q", `"hello"`, false},
		{"T", bv, "T", "bytes", false},
		{"b", bv, "b", "aGVsbG8=", false},
		{"B", bv, "B", "aGVsbG8", false},
		{"x", bv, "x", "68656c6c6f", false},
		{"X", bv, "X", "68656C6C6F", false},
		{"x binary", bin, "x", "00ff10", false},
		{"u", core.NewBytesValue([]byte("a b/c"), false), "u", "a%20b%2Fc", false},

		// precision counts BYTES (not runes) for bytes
		{"prec bytes", mix, ".3", "h\xc3\xa9", false},
		{"prec ge len", bv, ".10", "hello", false},
		{"prec on x", bv, ".3x", "68656c", false},

		// width
		{"width", bv, "8", "hello   ", false},
		{"width right", bv, ">8", "   hello", false},

		// errors
		{"sign", bv, "+", "", true},
		{"zeropad", bv, "08", "", true},
		{"verb d", bv, "d", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatArrayValue(t *testing.T) {
	av := core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
	}, false)
	mixed := core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.NewStringValue("hi"),
	}, false)
	empty := core.NewArrayValue(nil, false)

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		{"default", av, "", "[1, 2, 3]", false},
		{"v", av, "v", "[1, 2, 3]", false},
		{"T", av, "T", "array", false},
		{"empty", empty, "", "[]", false},
		{"nested string is quoted", mixed, "", `[1, "hi"]`, false},

		// width / align (default left)
		{"width left", av, "15", "[1, 2, 3]      ", false},
		{"width right", av, ">15", "      [1, 2, 3]", false},
		{"width center fill", av, "*^11", "*[1, 2, 3]*", false},

		// errors
		{"sign", av, "+", "", true},
		{"prec", av, ".3", "", true},
		{"zeropad", av, "010", "", true},
		{"grouping", av, ",", "", true},
		{"z", av, "z", "", true},
		{"verb d", av, "d", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}

func TestFormatRecordValue(t *testing.T) {
	rv := core.NewRecordValue(map[string]core.Value{
		"a": core.IntValue(1),
	}, false)

	// default and 'v' yield the same form
	for _, spec := range []string{"", "v"} {
		s, err := fspec.Parse(spec)
		require.NoError(t, err)
		got, ferr := rv.Format(s)
		require.NoError(t, ferr)
		require.Equal(t, `{"a": 1}`, got)
	}

	// width
	s, err := fspec.Parse("12")
	require.NoError(t, err)
	got, ferr := rv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, `{"a": 1}    `, got)

	// 'T' universal type-name verb
	s, err = fspec.Parse("T")
	require.NoError(t, err)
	got, ferr = rv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, "record", got)

	// errors
	for _, bad := range []string{"+", ".3", "010", ",", "z", "d"} {
		sp, err := fspec.Parse(bad)
		if err != nil {
			continue
		}
		_, ferr := rv.Format(sp)
		if ferr == nil {
			t.Fatalf("expected error for spec %q", bad)
		}
	}
}

func TestFormatDictValue(t *testing.T) {
	dv := core.NewDictValue(map[string]core.Value{
		"a": core.IntValue(1),
	}, false)

	// default: bare braces
	s, err := fspec.Parse("")
	require.NoError(t, err)
	got, ferr := dv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, `dict({"a": 1})`, got)

	// v: dict() wrapper
	s, err = fspec.Parse("v")
	require.NoError(t, err)
	got, ferr = dv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, `dict({"a": 1})`, got)

	// width on default
	s, err = fspec.Parse("12")
	require.NoError(t, err)
	got, ferr = dv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, `dict({"a": 1})`, got)

	// 'T' universal type-name verb
	s, err = fspec.Parse("T")
	require.NoError(t, err)
	got, ferr = dv.Format(s)
	require.NoError(t, ferr)
	require.Equal(t, "dict", got)

	// errors
	for _, bad := range []string{"+", ".3", "010", ",", "z", "d"} {
		sp, perr := fspec.Parse(bad)
		if perr != nil {
			continue
		}
		_, ferr := dv.Format(sp)
		if ferr == nil {
			t.Fatalf("expected error for spec %q", bad)
		}
	}
}

func TestFormatIntRangeValue(t *testing.T) {
	r1 := core.NewIntRangeValue(0, 10, 1)
	r2 := core.NewIntRangeValue(0, 10, 2)

	cases := []struct {
		name    string
		val     core.Value
		spec    string
		want    string
		wantErr bool
	}{
		{"default step1", r1, "", "range(0, 10)", false},
		{"default step2", r2, "", "range(0, 10, 2)", false},
		{"v step1", r1, "v", "range(0, 10)", false},
		{"v step2", r2, "v", "range(0, 10, 2)", false},
		{"T", r1, "T", "range", false},

		// width / align
		{"width left", r1, "15", "range(0, 10)   ", false},
		{"width right", r1, ">15", "   range(0, 10)", false},
		{"v ignores width fill", r2, "*^17v", "range(0, 10, 2)", false},

		// errors
		{"sign", r1, "+", "", true},
		{"prec", r1, ".3", "", true},
		{"zeropad", r1, "010", "", true},
		{"grouping", r1, ",", "", true},
		{"z", r1, "z", "", true},
		{"verb d", r1, "d", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := fspec.Parse(c.spec)
			if c.wantErr && err != nil {
				return
			}
			require.NoError(t, err)
			got, ferr := c.val.Format(s)
			if c.wantErr {
				if ferr == nil {
					t.Fatalf("Format(%q): expected error, got %q", c.spec, got)
				}
				return
			}
			require.NoError(t, ferr)
			require.Equal(t, c.want, got)
		})
	}
}
