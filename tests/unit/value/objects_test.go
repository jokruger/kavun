package value

import (
	"testing"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	mock "github.com/jokruger/gs/tests"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/token"
	"github.com/jokruger/gs/value"
)

var vm = mock.Vm
var alloc = mock.Alloc

func TestObject_TypeName(t *testing.T) {
	var o core.Object = alloc.NewInt(0)
	require.Equal(t, "int", o.TypeName())

	o = alloc.NewFloat(0)
	require.Equal(t, "float", o.TypeName())

	o = alloc.NewChar(0)
	require.Equal(t, "char", o.TypeName())

	o = alloc.NewString("")
	require.Equal(t, "string", o.TypeName())

	o = alloc.NewBool(false)
	require.Equal(t, "bool", o.TypeName())

	o = alloc.NewArray(nil, false)
	require.Equal(t, "array", o.TypeName())

	o = alloc.NewRecord(nil, false)
	require.Equal(t, "record", o.TypeName())

	o = alloc.NewArrayIterator(nil)
	require.Equal(t, "array-iterator", o.TypeName())

	o = alloc.NewStringIterator(nil)
	require.Equal(t, "string-iterator", o.TypeName())

	o = alloc.NewMapIterator(nil)
	require.Equal(t, "map-iterator", o.TypeName())

	o = alloc.NewBuiltinFunction("fn", nil, 0, false)
	require.Equal(t, "<builtin-function:fn/0>", o.TypeName())

	o = &value.CompiledFunction{}
	require.Equal(t, "<compiled-function/0>", o.TypeName())

	o = alloc.NewUndefined()
	require.Equal(t, "undefined", o.TypeName())

	o = alloc.NewError(nil)
	require.Equal(t, "error", o.TypeName())

	o = alloc.NewBytes(nil)
	require.Equal(t, "bytes", o.TypeName())
}

func TestObject_IsFalsy(t *testing.T) {
	var o core.Object = alloc.NewInt(0)
	require.True(t, o.IsFalse())

	o = alloc.NewInt(1)
	require.False(t, o.IsFalse())

	o = alloc.NewFloat(0)
	require.False(t, o.IsFalse())

	o = alloc.NewFloat(1)
	require.False(t, o.IsFalse())

	o = alloc.NewChar(' ')
	require.False(t, o.IsFalse())

	o = alloc.NewChar('T')
	require.False(t, o.IsFalse())

	o = alloc.NewString("")
	require.True(t, o.IsFalse())

	o = alloc.NewString(" ")
	require.False(t, o.IsFalse())

	o = alloc.NewArray(nil, false)
	require.True(t, o.IsFalse())

	o = alloc.NewArray([]core.Object{nil}, false) // nil is not valid but still count as 1 element
	require.False(t, o.IsFalse())

	o = alloc.NewRecord(nil, false)
	require.True(t, o.IsFalse())

	o = alloc.NewRecord(map[string]core.Object{"a": nil}, false) // nil is not valid but still count as 1 element
	require.False(t, o.IsFalse())

	o = alloc.NewStringIterator(nil)
	require.True(t, o.IsFalse())

	o = alloc.NewArrayIterator(nil)
	require.True(t, o.IsFalse())

	o = alloc.NewMapIterator(nil)
	require.True(t, o.IsFalse())

	o = alloc.NewBuiltinFunction("fn", nil, 0, false)
	require.False(t, o.IsFalse())

	o = &value.CompiledFunction{}
	require.False(t, o.IsFalse())

	o = alloc.NewUndefined()
	require.True(t, o.IsFalse())

	o = alloc.NewError(nil)
	require.True(t, o.IsFalse())

	o = alloc.NewBytes(nil)
	require.True(t, o.IsFalse())

	o = alloc.NewBytes([]byte{1, 2})
	require.False(t, o.IsFalse())
}

func TestObject_String(t *testing.T) {
	var o core.Object = alloc.NewInt(0)
	require.Equal(t, "0", o.String())

	o = alloc.NewInt(1)
	require.Equal(t, "1", o.String())

	o = alloc.NewFloat(0)
	require.Equal(t, "0", o.String())

	o = alloc.NewFloat(1)
	require.Equal(t, "1", o.String())

	o = alloc.NewChar(' ')
	require.Equal(t, "' '", o.String())

	o = alloc.NewChar('T')
	require.Equal(t, "'T'", o.String())

	o = alloc.NewString("")
	require.Equal(t, `""`, o.String())

	o = alloc.NewString(" ")
	require.Equal(t, `" "`, o.String())

	o = alloc.NewArray(nil, false)
	require.Equal(t, "[]", o.String())

	o = alloc.NewRecord(nil, false)
	require.Equal(t, "{}", o.String())

	o = alloc.NewError(nil)
	require.Equal(t, "error(undefined)", o.String())

	o = alloc.NewError(alloc.NewString("error 1"))
	require.Equal(t, `error("error 1")`, o.String())

	o = alloc.NewStringIterator(nil)
	require.Equal(t, "<string-iterator>", o.String())

	o = alloc.NewArrayIterator(nil)
	require.Equal(t, "<array-iterator>", o.String())

	o = alloc.NewMapIterator(nil)
	require.Equal(t, "<map-iterator>", o.String())

	o = alloc.NewUndefined()
	require.Equal(t, "undefined", o.String())

	o = alloc.NewBytes(nil)
	require.Equal(t, "bytes([])", o.String())

	o = alloc.NewBytes([]byte("foo"))
	require.Equal(t, "bytes([102, 111, 111])", o.String())
}

func TestObject_BinaryOp(t *testing.T) {
	var o core.Object = alloc.NewChar(0)
	_, err := o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)

	o = alloc.NewBool(false)
	_, err = o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)

	o = alloc.NewRecord(nil, false)
	_, err = o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)

	o = alloc.NewArrayIterator(nil)
	_, err = o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)

	o = alloc.NewStringIterator(nil)
	_, err = o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)

	o = alloc.NewMapIterator(nil)
	_, err = o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)

	o = alloc.NewBuiltinFunction("fn", nil, 0, false)
	_, err = o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)

	o = &value.CompiledFunction{}
	_, err = o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)

	o = alloc.NewUndefined()
	_, err = o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)

	o = alloc.NewError(nil)
	_, err = o.BinaryOp(vm, token.Add, alloc.NewUndefined())
	require.Error(t, err)
}

func TestArray_BinaryOp(t *testing.T) {
	testBinaryOp(t, alloc.NewArray(nil, false), token.Add,
		alloc.NewArray(nil, false), alloc.NewArray(nil, false))
	testBinaryOp(t, alloc.NewArray(nil, false), token.Add,
		alloc.NewArray([]core.Object{}, false), alloc.NewArray(nil, false))
	testBinaryOp(t, alloc.NewArray([]core.Object{}, false), token.Add,
		alloc.NewArray(nil, false), alloc.NewArray([]core.Object{}, false))
	testBinaryOp(t, alloc.NewArray([]core.Object{}, false), token.Add,
		alloc.NewArray([]core.Object{}, false),
		alloc.NewArray([]core.Object{}, false))
	testBinaryOp(t, alloc.NewArray(nil, false), token.Add,
		alloc.NewArray([]core.Object{
			alloc.NewInt(1),
		}, false), alloc.NewArray([]core.Object{
			alloc.NewInt(1),
		}, false))
	testBinaryOp(t, alloc.NewArray(nil, false), token.Add,
		alloc.NewArray([]core.Object{
			alloc.NewInt(1),
			alloc.NewInt(2),
			alloc.NewInt(3),
		}, false), alloc.NewArray([]core.Object{
			alloc.NewInt(1),
			alloc.NewInt(2),
			alloc.NewInt(3),
		}, false))
	testBinaryOp(t, alloc.NewArray([]core.Object{
		alloc.NewInt(1),
		alloc.NewInt(2),
		alloc.NewInt(3),
	}, false), token.Add, alloc.NewArray(nil, false),
		alloc.NewArray([]core.Object{
			alloc.NewInt(1),
			alloc.NewInt(2),
			alloc.NewInt(3),
		}, false))
	testBinaryOp(t, alloc.NewArray([]core.Object{
		alloc.NewInt(1),
		alloc.NewInt(2),
		alloc.NewInt(3),
	}, false), token.Add, alloc.NewArray([]core.Object{
		alloc.NewInt(4),
		alloc.NewInt(5),
		alloc.NewInt(6),
	}, false), alloc.NewArray([]core.Object{
		alloc.NewInt(1),
		alloc.NewInt(2),
		alloc.NewInt(3),
		alloc.NewInt(4),
		alloc.NewInt(5),
		alloc.NewInt(6),
	}, false))
}

func TestError_Equals(t *testing.T) {
	err1 := alloc.NewError(alloc.NewString("some error"))
	err2 := err1
	require.True(t, err1.Equals(err2))
	require.True(t, err2.Equals(err1))

	err2 = alloc.NewError(alloc.NewString("some error"))
	require.True(t, err1.Equals(err2))
	require.True(t, err2.Equals(err1))

	err2 = alloc.NewError(alloc.NewString("some error 2"))
	require.False(t, err1.Equals(err2))
	require.False(t, err2.Equals(err1))
}

func TestFloat_BinaryOp(t *testing.T) {
	// float + float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, alloc.NewFloat(l), token.Add,
				alloc.NewFloat(r), alloc.NewFloat(l+r))
		}
	}

	// float - float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, alloc.NewFloat(l), token.Sub,
				alloc.NewFloat(r), alloc.NewFloat(l-r))
		}
	}

	// float * float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, alloc.NewFloat(l), token.Mul,
				alloc.NewFloat(r), alloc.NewFloat(l*r))
		}
	}

	// float / float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			if r != 0 {
				testBinaryOp(t, alloc.NewFloat(l), token.Quo,
					alloc.NewFloat(r), alloc.NewFloat(l/r))
			}
		}
	}

	// float < float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, alloc.NewFloat(l), token.Less,
				alloc.NewFloat(r), boolValue(l < r))
		}
	}

	// float > float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, alloc.NewFloat(l), token.Greater,
				alloc.NewFloat(r), boolValue(l > r))
		}
	}

	// float <= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, alloc.NewFloat(l), token.LessEq,
				alloc.NewFloat(r), boolValue(l <= r))
		}
	}

	// float >= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, alloc.NewFloat(l), token.GreaterEq,
				alloc.NewFloat(r), boolValue(l >= r))
		}
	}

	// float + int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewFloat(l), token.Add,
				alloc.NewInt(r), alloc.NewFloat(l+float64(r)))
		}
	}

	// float - int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewFloat(l), token.Sub,
				alloc.NewInt(r), alloc.NewFloat(l-float64(r)))
		}
	}

	// float * int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewFloat(l), token.Mul,
				alloc.NewInt(r), alloc.NewFloat(l*float64(r)))
		}
	}

	// float / int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, alloc.NewFloat(l), token.Quo,
					alloc.NewInt(r),
					alloc.NewFloat(l/float64(r)))
			}
		}
	}

	// float < int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewFloat(l), token.Less,
				alloc.NewInt(r), boolValue(l < float64(r)))
		}
	}

	// float > int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewFloat(l), token.Greater,
				alloc.NewInt(r), boolValue(l > float64(r)))
		}
	}

	// float <= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewFloat(l), token.LessEq,
				alloc.NewInt(r), boolValue(l <= float64(r)))
		}
	}

	// float >= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewFloat(l), token.GreaterEq,
				alloc.NewInt(r), boolValue(l >= float64(r)))
		}
	}
}

func TestInt_BinaryOp(t *testing.T) {
	// int + int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewInt(l), token.Add,
				alloc.NewInt(r), alloc.NewInt(l+r))
		}
	}

	// int - int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewInt(l), token.Sub,
				alloc.NewInt(r), alloc.NewInt(l-r))
		}
	}

	// int * int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewInt(l), token.Mul,
				alloc.NewInt(r), alloc.NewInt(l*r))
		}
	}

	// int / int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, alloc.NewInt(l), token.Quo,
					alloc.NewInt(r), alloc.NewInt(l/r))
			}
		}
	}

	// int % int
	for l := int64(-4); l <= 4; l++ {
		for r := -int64(-4); r <= 4; r++ {
			if r == 0 {
				testBinaryOp(t, alloc.NewInt(l), token.Rem,
					alloc.NewInt(r), alloc.NewInt(l%r))
			}
		}
	}

	// int & int
	testBinaryOp(t,
		alloc.NewInt(0), token.And, alloc.NewInt(0),
		alloc.NewInt(int64(0)))
	testBinaryOp(t,
		alloc.NewInt(1), token.And, alloc.NewInt(0),
		alloc.NewInt(int64(1)&int64(0)))
	testBinaryOp(t,
		alloc.NewInt(0), token.And, alloc.NewInt(1),
		alloc.NewInt(int64(0)&int64(1)))
	testBinaryOp(t,
		alloc.NewInt(1), token.And, alloc.NewInt(1),
		alloc.NewInt(int64(1)))
	testBinaryOp(t,
		alloc.NewInt(0), token.And, alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(0)&int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(1), token.And, alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(1)&int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(int64(0xffffffff)), token.And,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(1984), token.And,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(1984)&int64(0xffffffff)))
	testBinaryOp(t, alloc.NewInt(-1984), token.And,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(-1984)&int64(0xffffffff)))

	// int | int
	testBinaryOp(t,
		alloc.NewInt(0), token.Or, alloc.NewInt(0),
		alloc.NewInt(int64(0)))
	testBinaryOp(t,
		alloc.NewInt(1), token.Or, alloc.NewInt(0),
		alloc.NewInt(int64(1)|int64(0)))
	testBinaryOp(t,
		alloc.NewInt(0), token.Or, alloc.NewInt(1),
		alloc.NewInt(int64(0)|int64(1)))
	testBinaryOp(t,
		alloc.NewInt(1), token.Or, alloc.NewInt(1),
		alloc.NewInt(int64(1)))
	testBinaryOp(t,
		alloc.NewInt(0), token.Or, alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(0)|int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(1), token.Or, alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(1)|int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(int64(0xffffffff)), token.Or,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(1984), token.Or,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(1984)|int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(-1984), token.Or,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(-1984)|int64(0xffffffff)))

	// int ^ int
	testBinaryOp(t,
		alloc.NewInt(0), token.Xor, alloc.NewInt(0),
		alloc.NewInt(int64(0)))
	testBinaryOp(t,
		alloc.NewInt(1), token.Xor, alloc.NewInt(0),
		alloc.NewInt(int64(1)^int64(0)))
	testBinaryOp(t,
		alloc.NewInt(0), token.Xor, alloc.NewInt(1),
		alloc.NewInt(int64(0)^int64(1)))
	testBinaryOp(t,
		alloc.NewInt(1), token.Xor, alloc.NewInt(1),
		alloc.NewInt(int64(0)))
	testBinaryOp(t,
		alloc.NewInt(0), token.Xor, alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(0)^int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(1), token.Xor, alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(1)^int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(int64(0xffffffff)), token.Xor,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(0)))
	testBinaryOp(t,
		alloc.NewInt(1984), token.Xor,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(1984)^int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(-1984), token.Xor,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(-1984)^int64(0xffffffff)))

	// int &^ int
	testBinaryOp(t,
		alloc.NewInt(0), token.AndNot, alloc.NewInt(0),
		alloc.NewInt(int64(0)))
	testBinaryOp(t,
		alloc.NewInt(1), token.AndNot, alloc.NewInt(0),
		alloc.NewInt(int64(1)&^int64(0)))
	testBinaryOp(t,
		alloc.NewInt(0), token.AndNot,
		alloc.NewInt(1), alloc.NewInt(int64(0)&^int64(1)))
	testBinaryOp(t,
		alloc.NewInt(1), token.AndNot, alloc.NewInt(1),
		alloc.NewInt(int64(0)))
	testBinaryOp(t,
		alloc.NewInt(0), token.AndNot,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(0)&^int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(1), token.AndNot,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(1)&^int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(int64(0xffffffff)), token.AndNot,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(0)))
	testBinaryOp(t,
		alloc.NewInt(1984), token.AndNot,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(1984)&^int64(0xffffffff)))
	testBinaryOp(t,
		alloc.NewInt(-1984), token.AndNot,
		alloc.NewInt(int64(0xffffffff)),
		alloc.NewInt(int64(-1984)&^int64(0xffffffff)))

	// int << int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			alloc.NewInt(0), token.Shl, alloc.NewInt(s),
			alloc.NewInt(int64(0)<<uint(s)))
		testBinaryOp(t,
			alloc.NewInt(1), token.Shl, alloc.NewInt(s),
			alloc.NewInt(int64(1)<<uint(s)))
		testBinaryOp(t,
			alloc.NewInt(2), token.Shl, alloc.NewInt(s),
			alloc.NewInt(int64(2)<<uint(s)))
		testBinaryOp(t,
			alloc.NewInt(-1), token.Shl, alloc.NewInt(s),
			alloc.NewInt(int64(-1)<<uint(s)))
		testBinaryOp(t,
			alloc.NewInt(-2), token.Shl, alloc.NewInt(s),
			alloc.NewInt(int64(-2)<<uint(s)))
		testBinaryOp(t,
			alloc.NewInt(int64(0xffffffff)), token.Shl,
			alloc.NewInt(s),
			alloc.NewInt(int64(0xffffffff)<<uint(s)))
	}

	// int >> int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			alloc.NewInt(0), token.Shr, alloc.NewInt(s),
			alloc.NewInt(int64(0)>>uint(s)))
		testBinaryOp(t,
			alloc.NewInt(1), token.Shr, alloc.NewInt(s),
			alloc.NewInt(int64(1)>>uint(s)))
		testBinaryOp(t,
			alloc.NewInt(2), token.Shr, alloc.NewInt(s),
			alloc.NewInt(int64(2)>>uint(s)))
		testBinaryOp(t,
			alloc.NewInt(-1), token.Shr, alloc.NewInt(s),
			alloc.NewInt(int64(-1)>>uint(s)))
		testBinaryOp(t,
			alloc.NewInt(-2), token.Shr, alloc.NewInt(s),
			alloc.NewInt(int64(-2)>>uint(s)))
		testBinaryOp(t,
			alloc.NewInt(int64(0xffffffff)), token.Shr,
			alloc.NewInt(s),
			alloc.NewInt(int64(0xffffffff)>>uint(s)))
	}

	// int < int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewInt(l), token.Less,
				alloc.NewInt(r), boolValue(l < r))
		}
	}

	// int > int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewInt(l), token.Greater,
				alloc.NewInt(r), boolValue(l > r))
		}
	}

	// int <= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewInt(l), token.LessEq,
				alloc.NewInt(r), boolValue(l <= r))
		}
	}

	// int >= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, alloc.NewInt(l), token.GreaterEq,
				alloc.NewInt(r), boolValue(l >= r))
		}
	}

	// int + float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, alloc.NewInt(l), token.Add,
				alloc.NewFloat(r),
				alloc.NewFloat(float64(l)+r))
		}
	}

	// int - float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, alloc.NewInt(l), token.Sub,
				alloc.NewFloat(r),
				alloc.NewFloat(float64(l)-r))
		}
	}

	// int * float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, alloc.NewInt(l), token.Mul,
				alloc.NewFloat(r),
				alloc.NewFloat(float64(l)*r))
		}
	}

	// int / float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			if r != 0 {
				testBinaryOp(t, alloc.NewInt(l), token.Quo,
					alloc.NewFloat(r),
					alloc.NewFloat(float64(l)/r))
			}
		}
	}

	// int < float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, alloc.NewInt(l), token.Less,
				alloc.NewFloat(r), boolValue(float64(l) < r))
		}
	}

	// int > float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, alloc.NewInt(l), token.Greater,
				alloc.NewFloat(r), boolValue(float64(l) > r))
		}
	}

	// int <= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, alloc.NewInt(l), token.LessEq,
				alloc.NewFloat(r), boolValue(float64(l) <= r))
		}
	}

	// int >= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, alloc.NewInt(l), token.GreaterEq,
				alloc.NewFloat(r), boolValue(float64(l) >= r))
		}
	}
}

func TestRecord_Index(t *testing.T) {
	m := alloc.NewRecord(make(map[string]core.Object), false)
	k := alloc.NewInt(1)
	v := alloc.NewString("abcdef")
	err := m.Assign(k, v)

	require.NoError(t, err)

	res, err := m.Access(vm, k, parser.OpIndex)
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
			testBinaryOp(t, alloc.NewString(ls), token.Add,
				alloc.NewString(rs),
				alloc.NewString(ls+rs))

			rc := []rune(rstr)[r]
			testBinaryOp(t, alloc.NewString(ls), token.Add,
				alloc.NewChar(rc),
				alloc.NewString(ls+string(rc)))
		}
	}
}

func testBinaryOp(t *testing.T, lhs core.Object, op token.Token, rhs core.Object, expected core.Object) {
	t.Helper()
	actual, err := lhs.BinaryOp(vm, op, rhs)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func boolValue(b bool) core.Object {
	return alloc.NewBool(b)
}
