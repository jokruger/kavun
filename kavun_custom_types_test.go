package kavun_test

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/require"
)

var (
	MyCounter             = kavun.UserDefinedType + 1
	MyCustomNumber        = kavun.UserDefinedType + 2
	MyStringArray         = kavun.UserDefinedType + 3
	MyStringCircle        = kavun.UserDefinedType + 4
	MyStringDict          = kavun.UserDefinedType + 5
	MyStringArrayIterator = kavun.UserDefinedType + 6
	MyBox                 = kavun.UserDefinedType + 7
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

func NewBoxValue(inner core.Value) core.Value {
	o := &Box{inner: inner}
	return core.Value{Type: MyBox, Ptr: unsafe.Pointer(o)}
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

// Box wraps any single Value and is deliberately "compatible" with every other type: it compares
// equal to whatever it wraps and applies operators to the wrapped value. Its whole reason to exist is
// to exercise the delegate paths from the *builtin* side — nothing in core/*.go can name MyBox, so
// every `builtin == box` / `builtin op box` row can only ever work if the builtin's own hook
// delegates to the type it doesn't recognize. See docs/extending-types.md.
type Box struct {
	inner core.Value
}

func toBox(v core.Value) *Box {
	if v.Type != MyBox {
		panic(fmt.Sprintf("invalid type: expected Box, got %s", v.TypeName()))
	}
	return (*Box)(v.Ptr)
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
		BinaryOp: func(v core.Value, rhs core.Value, op token.Token, _ bool) (core.Value, error) {
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
		IsTrue: func(v core.Value) (bool, error) { return toCounter(v).value != 0, nil },
		Equal: func(v core.Value, r core.Value, _ bool) bool {
			if r.Type != MyCounter {
				return false
			}
			return toCounter(v).value == toCounter(r).value
		},
		Copy: func(v core.Value, _ bool) (core.Value, error) {
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
		BinaryOp: func(v core.Value, rhs core.Value, op token.Token, _ bool) (core.Value, error) {
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

	// Register Box
	core.SetValueType(MyBox, core.ValueTypeDescr{
		Name:   func(v core.Value) string { return "box" },
		String: func(v core.Value) string { return "box(" + toBox(v).inner.String() + ")" },
		// Recognizes every type, so it never needs the delegate itself and `final` is irrelevant —
		// the unrecognized-type obligation in docs/extending-types.md doesn't arise here.
		Equal: func(v core.Value, other core.Value, _ bool) bool {
			inner := toBox(v).inner
			if other.Type == MyBox {
				return inner.Equal(toBox(other).inner)
			}
			return inner.Equal(other)
		},
		BinaryOp: func(v core.Value, other core.Value, op token.Token, reflected bool) (core.Value, error) {
			inner := toBox(v).inner
			if other.Type == MyBox {
				other = toBox(other).inner
			}
			if reflected {
				// v is the ORIGINAL rhs, so the expression written was "other op inner".
				return other.BinaryOp(op, inner)
			}
			return inner.BinaryOp(op, other)
		},
	})

	// Register StringArray
	core.SetValueType(MyStringArray, core.ValueTypeDescr{
		Name:   func(v core.Value) string { return "string-array" },
		String: func(v core.Value) string { return strings.Join(toStringArray(v).Value, ", ") },
		BinaryOp: func(v core.Value, rhs core.Value, op token.Token, _ bool) (core.Value, error) {
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
		IsTrue: func(v core.Value) (bool, error) { return len(toStringArray(v).Value) != 0, nil },
		Equal: func(v core.Value, rhs core.Value, _ bool) bool {
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
		Copy: func(v core.Value, _ bool) (core.Value, error) {
			return NewStringArrayValue(append([]string{}, toStringArray(v).Value...)), nil
		},
		Access: func(v core.Value, index core.Value, mode bc.Opcode) (core.Value, error) {
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
		Assign: func(v core.Value, index core.Value, value core.Value, _ bc.Opcode) error {
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
		Access: func(v core.Value, index core.Value, mode bc.Opcode) (core.Value, error) {
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
		Assign: func(v core.Value, index core.Value, value core.Value, _ bc.Opcode) error {
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
		Access: func(v core.Value, index core.Value, mode bc.Opcode) (core.Value, error) {
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
		Assign: func(v core.Value, index core.Value, value core.Value, _ bc.Opcode) error {
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

// Every builtin type's Equal must delegate to an unrecognized type's own hook, so that an embedder
// type can own its pairings with builtins from its own side and have them work in BOTH written
// orders. `builtin == box` is the direction that only works because the builtin delegates: before
// that was made unconditional, array/dict/record/bytes/time/error/int_range/the iterators/
// compiled_function/format_spec and the DefaultValueType fallback all returned a flat false here,
// which made an embedder type structurally unable to be equal to any of them. undefined is the one
// sanctioned exception — see core/undefined.go's undefinedTypeEqual.
func TestEmbedderTypeEqualityDelegation(t *testing.T) {
	fs := &core.FormatSpec{Text: "d"}
	builtins := []struct {
		name string
		v    core.Value
	}{
		{"bool", core.True},
		{"byte", core.ByteValue(7)},
		{"rune", core.RuneValue('x')},
		{"int", core.IntValue(7)},
		{"float", core.FloatValue(7.5)},
		{"decimal", core.NewDecimalValue(dec128.FromInt64(7))},
		{"string", core.NewStringValue("ab")},
		{"bytes", core.NewBytesValue([]byte("ab"), false)},
		{"runes", core.NewRunesValue([]rune("ab"), false)},
		{"array", core.NewArrayValue([]core.Value{core.IntValue(1)}, false)},
		{"dict", core.NewDictValue(map[string]core.Value{"a": core.IntValue(1)}, false)},
		{"record", core.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)},
		{"time", core.NewTimeValue(time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC))},
		{"error", core.NewErrorValue(core.Undefined, core.KindUser, false)},
		{"int-range", core.NewIntRangeValue(0, 10, 1)},
		{"int-range-iterator", core.NewIntRangeIteratorValue(0, 10, 1)},
		{"dict-iterator", core.NewDictIteratorValue(map[string]core.Value{"a": core.IntValue(1)})},
		{"array-iterator", core.NewArrayIteratorValue([]core.Value{core.IntValue(1)})},
		{"bytes-iterator", core.NewBytesIteratorValue([]byte("ab"))},
		{"runes-iterator", core.NewRunesIteratorValue([]rune("ab"))},
		{"format-spec", core.NewStaticFormatSpecValue(fs)},
		{"builtin-closure", core.NewBuiltinClosureValue("f", func(core.VM, []core.Value) (core.Value, error) { return core.Undefined, nil }, 0, false)},
	}

	for _, c := range builtins {
		t.Run("box("+c.name+") == "+c.name, func(t *testing.T) {
			require.True(t, NewBoxValue(c.v).Equal(c.v))
		})
		t.Run(c.name+" == box("+c.name+") -- only works if the builtin delegates", func(t *testing.T) {
			require.True(t, c.v.Equal(NewBoxValue(c.v)))
		})
		t.Run(c.name+" == box(unrelated) -> false", func(t *testing.T) {
			require.False(t, c.v.Equal(NewBoxValue(core.NewStringValue("no-such-value"))))
		})
	}

	// undefined: the one type that deliberately never delegates, so the pairing is asymmetric BY
	// DESIGN. Pinned here so the exception stays a decision rather than an accident.
	t.Run("box(undefined) == undefined -> true (box recognizes it directly)", func(t *testing.T) {
		require.True(t, NewBoxValue(core.Undefined).Equal(core.Undefined))
	})
	t.Run("undefined == box(undefined) -> false (undefined never delegates)", func(t *testing.T) {
		require.False(t, core.Undefined.Equal(NewBoxValue(core.Undefined)))
	})
}

// The BinaryOp counterpart: an embedder type's reflected branch is reachable only through a
// builtin's rule-3 delegate, so `builtin op box` exercises exactly that path — including the
// non-commutative operators, where the reflected branch has to compute "other op v", not "v op
// other".
func TestEmbedderTypeBinaryOpDelegation(t *testing.T) {
	box := func(v core.Value) core.Value { return NewBoxValue(v) }

	cases := []struct {
		name string
		lhs  core.Value
		op   token.Token
		rhs  core.Value
		want core.Value
	}{
		{"box(int) + int", box(core.IntValue(10)), token.Add, core.IntValue(4), core.IntValue(14)},
		{"int + box(int)", core.IntValue(10), token.Add, box(core.IntValue(4)), core.IntValue(14)},
		{"box(int) - int", box(core.IntValue(10)), token.Sub, core.IntValue(4), core.IntValue(6)},
		{"int - box(int) -- reflected, operand order matters", core.IntValue(10), token.Sub, box(core.IntValue(4)), core.IntValue(6)},
		{"box(int) / int", box(core.IntValue(10)), token.Quo, core.IntValue(4), core.IntValue(2)},
		{"int / box(int) -- reflected, operand order matters", core.IntValue(10), token.Quo, box(core.IntValue(4)), core.IntValue(2)},
		{"box(string) + string", box(core.NewStringValue("a")), token.Add, core.NewStringValue("b"), core.NewStringValue("ab")},
		{"string + box(string) -- reflected, content order matters", core.NewStringValue("a"), token.Add, box(core.NewStringValue("b")), core.NewStringValue("ab")},
		{"box(int) < int", box(core.IntValue(1)), token.Less, core.IntValue(2), core.True},
		{"int < box(int) -- reflected", core.IntValue(1), token.Less, box(core.IntValue(2)), core.True},
		{"int > box(int) -- reflected", core.IntValue(1), token.Greater, box(core.IntValue(2)), core.False},
		{"box(decimal) + int", box(core.NewDecimalValue(dec128.FromInt64(10))), token.Add, core.IntValue(4), core.NewDecimalValue(dec128.FromInt64(14))},
		{"decimal + box(int) -- reflected", core.NewDecimalValue(dec128.FromInt64(10)), token.Add, box(core.IntValue(4)), core.NewDecimalValue(dec128.FromInt64(14))},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.lhs.BinaryOp(c.op, c.rhs)
			require.NoError(t, err)
			require.Equal(t, c.want, got)
		})
	}

	// end-to-end through the VM, the way an embedder actually sees it
	expectRun(t, `out = b == 42`, Opts().Symbol("b", NewBoxValue(core.IntValue(42))).Skip2ndPass(), true)
	expectRun(t, `out = 42 == b`, Opts().Symbol("b", NewBoxValue(core.IntValue(42))).Skip2ndPass(), true)
	expectRun(t, `out = 50 - b`, Opts().Symbol("b", NewBoxValue(core.IntValue(8))).Skip2ndPass(), 42)
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
	expectRun(t, `out = ""; for i, s in arr { out += s + i.string() }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "one0two1three2")
}

func TestCompiled_CustomObject(t *testing.T) {
	c := compile(t, `r = (t<130)`, MAP{"r": core.Undefined, "t": NewCustomNumberValue(123)})
	compiledRun(t, c)
	compiledGet(t, c, "r", true)

	c = compile(t, `r = (t>13)`, MAP{"r": core.Undefined, "t": NewCustomNumberValue(123)})
	compiledRun(t, c)
	compiledGet(t, c, "r", true)
}
