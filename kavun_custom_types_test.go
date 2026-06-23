package kavun_test

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

var (
	MyCounter             = kavun.UserDefinedType + 1
	MyCustomNumber        = kavun.UserDefinedType + 2
	MyStringArray         = kavun.UserDefinedType + 3
	MyStringCircle        = kavun.UserDefinedType + 4
	MyStringDict          = kavun.UserDefinedType + 5
	MyStringArrayIterator = kavun.UserDefinedType + 6
)

func NewCounterValue(val int64) core.Value {
	o := &Counter{value: val}
	return core.Value{Type: MyCounter, Ptr: unsafe.Pointer(o)}
}

func NewCustomNumberValue(val int64) core.Value {
	o := &CustomNumber{value: val}
	return core.Value{Type: MyCustomNumber, Ptr: unsafe.Pointer(o)}
}

func NewStringArrayValue(vals []string) core.Value {
	o := &StringArray{Value: vals}
	return core.Value{Type: MyStringArray, Ptr: unsafe.Pointer(o)}
}

func NewStringCircleValue(vals []string) core.Value {
	o := &StringCircle{Value: vals}
	return core.Value{Type: MyStringCircle, Ptr: unsafe.Pointer(o)}
}

func NewStringDictValue(vals map[string]string) core.Value {
	o := &StringDict{Value: vals}
	return core.Value{Type: MyStringDict, Ptr: unsafe.Pointer(o)}
}

func NewStringArrayIteratorValue(arr *StringArray) core.Value {
	o := &StringArrayIterator{strArr: arr, idx: 0}
	return core.Value{Type: MyStringArrayIterator, Ptr: unsafe.Pointer(o)}
}

type Counter struct {
	value int64
}

func toCounter(v core.Value) *Counter {
	if v.Type != MyCounter {
		panic(fmt.Sprintf("invalid type: expected Counter, got %s", v.TypeName()))
	}
	return (*Counter)(v.Ptr)
}

type CustomNumber struct {
	value int64
}

func toCustomNumber(v core.Value) *CustomNumber {
	if v.Type != MyCustomNumber {
		panic(fmt.Sprintf("invalid type: expected CustomNumber, got %s", v.TypeName()))
	}
	return (*CustomNumber)(v.Ptr)
}

type StringArray struct {
	Value []string
}

func toStringArray(v core.Value) *StringArray {
	if v.Type != MyStringArray {
		panic(fmt.Sprintf("invalid type: expected StringArray, got %s", v.TypeName()))
	}
	return (*StringArray)(v.Ptr)
}

type StringCircle struct {
	Value []string
}

func toStringCircle(v core.Value) *StringCircle {
	if v.Type != MyStringCircle {
		panic(fmt.Sprintf("invalid type: expected StringCircle, got %s", v.TypeName()))
	}
	return (*StringCircle)(v.Ptr)
}

type StringDict struct {
	Value map[string]string
}

func toStringDict(v core.Value) *StringDict {
	if v.Type != MyStringDict {
		panic(fmt.Sprintf("invalid type: expected StringDict, got %s", v.TypeName()))
	}
	return (*StringDict)(v.Ptr)
}

type StringArrayIterator struct {
	strArr *StringArray
	idx    int
}

func toStringArrayIterator(v core.Value) *StringArrayIterator {
	if v.Type != MyStringArrayIterator {
		panic(fmt.Sprintf("invalid type: expected StringArrayIterator, got %s", v.TypeName()))
	}
	return (*StringArrayIterator)(v.Ptr)
}

func init() {
	// Register Counter
	core.SetValueType(MyCounter, core.ValueTypeDescr{
		Interface: func(v core.Value) any { return toCounter(v) },
		Name:      func(v core.Value) string { return "counter" },
		String:    func(v core.Value) string { return fmt.Sprintf("Counter(%d)", toCounter(v).value) },
		AsString:  func(v core.Value) (string, bool) { return v.String(), true },
		BinaryOp: func(v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			if rhs.Type == value.Int {
				o := toCounter(v)
				switch op {
				case token.Add:
					return NewCounterValue(o.value + int64(rhs.Data)), nil
				case token.Sub:
					return NewCounterValue(o.value - int64(rhs.Data)), nil
				}
			}
			if rhs.Type == MyCounter {
				o := toCounter(v)
				r := toCounter(rhs)
				switch op {
				case token.Add:
					return NewCounterValue(o.value + r.value), nil
				case token.Sub:
					return NewCounterValue(o.value - r.value), nil
				}
			}
			return core.Undefined, errors.New("invalid operator")
		},
		IsTrue: func(v core.Value) bool { return toCounter(v).value != 0 },
		Equal: func(v core.Value, r core.Value) bool {
			if r.Type != MyCounter {
				return false
			}
			return toCounter(v).value == toCounter(r).value
		},
		Clone: func(v core.Value) (core.Value, error) {
			return NewCounterValue(toCounter(v).value), nil
		},
		Call: func(vm core.VM, v core.Value, args []core.Value) (core.Value, error) {
			return core.IntValue(toCounter(v).value), nil
		},
		IsCallable: func(v core.Value) bool { return true },
	})

	// Register CustomNumber
	core.SetValueType(MyCustomNumber, core.ValueTypeDescr{
		Name:   func(v core.Value) string { return "Number" },
		String: func(v core.Value) string { return strconv.FormatInt(toCustomNumber(v).value, 10) },
		BinaryOp: func(v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			r, ok := rhs.AsInt()
			if !ok {
				return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
			}
			i := toCustomNumber(v).value
			switch op {
			case token.Less:
				return core.BoolValue(i < r), nil
			case token.Greater:
				return core.BoolValue(i > r), nil
			case token.LessEq:
				return core.BoolValue(i <= r), nil
			case token.GreaterEq:
				return core.BoolValue(i >= r), nil
			}
			t := core.IntValue(i)
			return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), t.TypeName())
		},
	})

	// Register StringArray
	core.SetValueType(MyStringArray, core.ValueTypeDescr{
		Name:   func(v core.Value) string { return "string-array" },
		String: func(v core.Value) string { return strings.Join(toStringArray(v).Value, ", ") },
		BinaryOp: func(v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			if rhs.Type == MyStringArray && op == token.Add {
				l := toStringArray(v)
				r := toStringArray(rhs)
				if len(r.Value) == 0 {
					return v, nil
				}
				return NewStringArrayValue(append(l.Value, r.Value...)), nil
			}
			return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		},
		IsTrue: func(v core.Value) bool { return len(toStringArray(v).Value) != 0 },
		Equal: func(v core.Value, rhs core.Value) bool {
			if rhs.Type == MyStringArray {
				l := toStringArray(v)
				r := toStringArray(rhs)
				if len(l.Value) != len(r.Value) {
					return false
				}
				for i, v := range l.Value {
					if v != r.Value[i] {
						return false
					}
				}
				return true
			}
			return false
		},
		Clone: func(v core.Value) (core.Value, error) {
			return NewStringArrayValue(append([]string{}, toStringArray(v).Value...)), nil
		},
		Access: func(v core.Value, index core.Value, mode opcode.Opcode) (core.Value, error) {
			o := toStringArray(v)
			intIdx, ok := index.AsInt()
			if ok {
				if intIdx >= 0 && intIdx < int64(len(o.Value)) {
					return core.NewStringValue(o.Value[intIdx]), nil
				}
				return core.Undefined, errs.NewIndexOutOfBoundsError("StringArray assignment", int(intIdx), len(o.Value))
			}
			strIdx, ok := index.AsString()
			if ok {
				for vidx, str := range o.Value {
					if strIdx == str {
						return core.IntValue(int64(vidx)), nil
					}
				}
				return core.Undefined, nil
			}
			return core.Undefined, errs.NewInvalidIndexTypeError("StringArray access", "int or string", index.TypeName())
		},
		Assign: func(v core.Value, index core.Value, value core.Value) error {
			o := toStringArray(v)
			strVal, ok := value.AsString()
			if !ok {
				return errs.NewInvalidIndexTypeError("StringArray assignment", "string(compatible)", value.TypeName())
			}
			intIdx, ok := index.AsInt()
			if ok {
				if intIdx >= 0 && intIdx < int64(len(o.Value)) {
					o.Value[intIdx] = strVal
					return nil
				}
				return errs.NewIndexOutOfBoundsError("StringArray assignment", int(intIdx), len(o.Value))
			}
			return errs.NewInvalidIndexTypeError("StringArray assignment", "int", v.TypeName())
		},
		Call: func(vm core.VM, v core.Value, args []core.Value) (core.Value, error) {
			if len(args) != 1 {
				return core.Undefined, errs.NewWrongNumArgumentsError("StringArray.Call", "1", len(args))
			}
			s1, ok := args[0].AsString()
			if !ok {
				return core.Undefined, errs.NewInvalidArgumentTypeError("StringArray.Call", "first", "string(compatible)", args[0].TypeName())
			}
			o := toStringArray(v)
			for i, v := range o.Value {
				if v == s1 {
					return core.IntValue(int64(i)), nil
				}
			}
			return core.Undefined, nil
		},
		IsCallable: func(v core.Value) bool { return true },
		Iterator: func(v core.Value) (core.Value, error) {
			return NewStringArrayIteratorValue(toStringArray(v)), nil
		},
		IsIterable: func(v core.Value) bool { return true },
	})

	// Register StringCircle
	core.SetValueType(MyStringCircle, core.ValueTypeDescr{
		Name:   func(v core.Value) string { return "string-circle" },
		String: func(v core.Value) string { return "" },
		Access: func(v core.Value, index core.Value, mode opcode.Opcode) (core.Value, error) {
			intIdx, ok := index.AsInt()
			if !ok {
				return core.Undefined, errs.NewInvalidIndexTypeError("StringCircle access", "int", index.TypeName())
			}
			o := toStringCircle(v)
			r := int(intIdx) % len(o.Value)
			if r < 0 {
				r = len(o.Value) + r
			}
			return core.NewStringValue(o.Value[r]), nil
		},
		Assign: func(v core.Value, index core.Value, value core.Value) error {
			intIdx, ok := index.AsInt()
			if !ok {
				return errs.NewInvalidIndexTypeError("StringCircle assignment", "int", index.TypeName())
			}
			o := toStringCircle(v)
			r := int(intIdx) % len(o.Value)
			if r < 0 {
				r = len(o.Value) + r
			}
			strVal, ok := value.AsString()
			if !ok {
				return errs.NewInvalidIndexTypeError("StringCircle assignment", "string(compatible)", value.TypeName())
			}
			o.Value[r] = strVal
			return nil
		},
	})

	// Register StringDict
	core.SetValueType(MyStringDict, core.ValueTypeDescr{
		Name:   func(v core.Value) string { return "string-dict" },
		String: func(v core.Value) string { return "" },
		Access: func(v core.Value, index core.Value, mode opcode.Opcode) (core.Value, error) {
			strIdx, ok := index.AsString()
			if !ok {
				return core.Undefined, errs.NewInvalidIndexTypeError("StringDict access", "string", index.TypeName())
			}
			o := toStringDict(v)
			for k, v := range o.Value {
				if strings.EqualFold(strIdx, k) {
					return core.NewStringValue(v), nil
				}
			}
			return core.Undefined, nil
		},
		Assign: func(v core.Value, index core.Value, value core.Value) error {
			strIdx, ok := index.AsString()
			if !ok {
				return errs.NewInvalidIndexTypeError("StringDict assignment", "string", index.TypeName())
			}
			strVal, ok := value.AsString()
			if !ok {
				return errs.NewInvalidIndexTypeError("StringDict assignment", "string(compatible)", value.TypeName())
			}
			o := toStringDict(v)
			o.Value[strings.ToLower(strIdx)] = strVal
			return nil
		},
	})

	// Register StringArrayIterator
	core.SetValueType(MyStringArrayIterator, core.ValueTypeDescr{
		Name:   func(v core.Value) string { return "string-array-iterator" },
		String: func(v core.Value) string { return "" },
		Next: func(v core.Value) bool {
			i := toStringArrayIterator(v)
			i.idx++
			return i.idx <= len(i.strArr.Value)
		},
		Key: func(v core.Value) (core.Value, error) {
			i := toStringArrayIterator(v)
			return core.IntValue(int64(i.idx - 1)), nil
		},
		Value: func(v core.Value) (core.Value, error) {
			i := toStringArrayIterator(v)
			return core.NewStringValue(i.strArr.Value[i.idx-1]), nil
		},
	})
}

func TestIndexable(t *testing.T) {
	dict := func() core.Value {
		return NewStringDictValue(map[string]string{"a": "foo", "b": "bar"})
	}

	expectRun(t, `out = d["a"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "foo")
	expectRun(t, `out = d["B"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "bar")
	expectRun(t, `out = d["x"]`, Opts().Symbol("d", dict()).Skip2ndPass(), core.Undefined)

	strCir := func() core.Value {
		return NewStringCircleValue([]string{"one", "two", "three"})
	}

	expectRun(t, `out = cir[0]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "one")
	expectRun(t, `out = cir[1]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "two")
	expectRun(t, `out = cir[-1]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "three")
	expectRun(t, `out = cir[-2]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "two")
	expectRun(t, `out = cir[3]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "one")
	expectError(t, `cir["a"]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "invalid_index_type")

	strArr := func() core.Value {
		return NewStringArrayValue([]string{"one", "two", "three"})
	}

	expectRun(t, `out = arr["one"]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), 0)
	expectRun(t, `out = arr["three"]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), 2)
	expectRun(t, `out = arr["four"]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), core.Undefined)
	expectRun(t, `out = arr[0]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "one")
	expectRun(t, `out = arr[1]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "two")
	expectError(t, `arr[-1]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "index_out_of_bounds")
}

func TestIndexAssignable(t *testing.T) {
	dict := func() core.Value {
		return NewStringDictValue(map[string]string{"a": "foo", "b": "bar"})
	}

	expectRun(t, `d["a"] = "1984"; out = d["a"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "1984")
	expectRun(t, `d["c"] = "1984"; out = d["c"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "1984")
	expectRun(t, `d["c"] = 1984; out = d["C"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "1984")

	strCir := func() core.Value {
		return NewStringCircleValue([]string{"one", "two", "three"})
	}

	expectRun(t, `cir[0] = "ONE"; out = cir[0]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "ONE")
	expectRun(t, `cir[1] = "TWO"; out = cir[1]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "TWO")
	expectRun(t, `cir[-1] = "THREE"; out = cir[2]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "THREE")
	expectRun(t, `cir[0] = "ONE"; out = cir[3]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "ONE")
	expectError(t, `cir["a"] = "ONE"`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "invalid_index_type")

	strArr := func() core.Value {
		return NewStringArrayValue([]string{"one", "two", "three"})
	}

	expectRun(t, `arr[0] = "ONE"; out = arr[0]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "ONE")
	expectRun(t, `arr[1] = "TWO"; out = arr[1]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "TWO")
	expectError(t, `arr["one"] = "ONE"`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "invalid_index_type")
}

func TestIterable(t *testing.T) {
	strArr := func() core.Value {
		return NewStringArrayValue([]string{"one", "two", "three"})
	}

	expectRun(t, `out = 0; for i, s in arr { out += i }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), 3)
	expectRun(t, `out = ""; for i, s in arr { out += s }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "onetwothree")
	expectRun(t, `out = ""; for i, s in arr { out += s + i }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "one0two1three2")
}

func TestCompiled_CustomObject(t *testing.T) {
	c := compile(t, `r := (t<130)`, MAP{"t": NewCustomNumberValue(123)})
	compiledRun(t, c)
	compiledGet(t, c, "r", true)

	c = compile(t, `r := (t>13)`, MAP{"t": NewCustomNumberValue(123)})
	compiledRun(t, c)
	compiledGet(t, c, "r", true)
}
