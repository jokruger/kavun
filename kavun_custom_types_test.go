package kavun_test

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/refpool"
)

var (
	MyCounter             = kavun.UserDefinedType + 1
	MyCustomNumber        = kavun.UserDefinedType + 2
	MyStringArray         = kavun.UserDefinedType + 3
	MyStringCircle        = kavun.UserDefinedType + 4
	MyStringDict          = kavun.UserDefinedType + 5
	MyStringArrayIterator = kavun.UserDefinedType + 6
)

type MyArena struct {
	arena *refpool.Arena
}

func NewMyArena() *MyArena {
	return &MyArena{arena: refpool.NewArena(
		true,
		true,
		refpool.With[Counter](MyCounter, 0),
		refpool.With[CustomNumber](MyCustomNumber, 0),
		refpool.With[StringArray](MyStringArray, 0),
		refpool.With[StringCircle](MyStringCircle, 0),
		refpool.With[StringDict](MyStringDict, 0),
		refpool.With[StringArrayIterator](MyStringArrayIterator, 0),
	)}
}

func (a *MyArena) Pin(v core.Value) {
	a.arena.Pin(v.Type, v.Data)
}

func (a *MyArena) Retain(v core.Value) {
	a.arena.Retain(v.Type, v.Data)
}

func (a *MyArena) Release(v core.Value) {
	a.arena.Release(v.Type, v.Data)
}

func (a *MyArena) Reset() {
	a.arena.Reset(MyCounter, true)
	a.arena.Reset(MyCustomNumber, true)
	a.arena.Reset(MyStringArray, true)
	a.arena.Reset(MyStringCircle, true)
	a.arena.Reset(MyStringDict, true)
	a.arena.Reset(MyStringArrayIterator, true)
}

func (a *MyArena) NewCounterValue(val int64) core.Value {
	r, p, ok := a.arena.New(MyCounter)
	if !ok {
		panic("failed to allocate Counter")
	}
	(*Counter)(p).value = val
	return core.Value{Type: MyCounter, Data: r}
}

func (a *MyArena) NewCustomNumberValue(val int64) core.Value {
	r, p, ok := a.arena.New(MyCustomNumber)
	if !ok {
		panic("failed to allocate CustomNumber")
	}
	(*CustomNumber)(p).value = val
	return core.Value{Type: MyCustomNumber, Data: r}
}

func (a *MyArena) NewStringArrayValue(vals []string) core.Value {
	r, p, ok := a.arena.New(MyStringArray)
	if !ok {
		panic("failed to allocate StringArray")
	}
	(*StringArray)(p).Value = vals
	return core.Value{Type: MyStringArray, Data: r}
}

func (a *MyArena) NewStringCircleValue(vals []string) core.Value {
	r, p, ok := a.arena.New(MyStringCircle)
	if !ok {
		panic("failed to allocate StringCircle")
	}
	(*StringCircle)(p).Value = vals
	return core.Value{Type: MyStringCircle, Data: r}
}

func (a *MyArena) NewStringDictValue(vals map[string]string) core.Value {
	r, p, ok := a.arena.New(MyStringDict)
	if !ok {
		panic("failed to allocate StringDict")
	}
	(*StringDict)(p).Value = vals
	return core.Value{Type: MyStringDict, Data: r}
}

func (a *MyArena) NewStringArrayIteratorValue(arr *StringArray) core.Value {
	r, p, ok := a.arena.New(MyStringArrayIterator)
	if !ok {
		panic("failed to allocate StringArrayIterator")
	}
	(*StringArrayIterator)(p).strArr = arr
	(*StringArrayIterator)(p).idx = 0
	return core.Value{Type: MyStringArrayIterator, Data: r}
}

type Counter struct {
	value int64
}

func toCounter(a *core.Arena, v core.Value) *Counter {
	if v.Type != MyCounter {
		panic(fmt.Sprintf("invalid type: expected Counter, got %s", v.TypeName(a)))
	}
	return (*Counter)(a.Payload().(*MyArena).arena.Resolve(MyCounter, v.Data))
}

type CustomNumber struct {
	value int64
}

func toCustomNumber(a *core.Arena, v core.Value) *CustomNumber {
	if v.Type != MyCustomNumber {
		panic(fmt.Sprintf("invalid type: expected CustomNumber, got %s", v.TypeName(a)))
	}
	return (*CustomNumber)(a.Payload().(*MyArena).arena.Resolve(MyCustomNumber, v.Data))
}

type StringArray struct {
	Value []string
}

func toStringArray(a *core.Arena, v core.Value) *StringArray {
	if v.Type != MyStringArray {
		panic(fmt.Sprintf("invalid type: expected StringArray, got %s", v.TypeName(a)))
	}
	return (*StringArray)(a.Payload().(*MyArena).arena.Resolve(MyStringArray, v.Data))
}

type StringCircle struct {
	Value []string
}

func toStringCircle(a *core.Arena, v core.Value) *StringCircle {
	if v.Type != MyStringCircle {
		panic(fmt.Sprintf("invalid type: expected StringCircle, got %s", v.TypeName(a)))
	}
	return (*StringCircle)(a.Payload().(*MyArena).arena.Resolve(MyStringCircle, v.Data))
}

type StringDict struct {
	Value map[string]string
}

func toStringDict(a *core.Arena, v core.Value) *StringDict {
	if v.Type != MyStringDict {
		panic(fmt.Sprintf("invalid type: expected StringDict, got %s", v.TypeName(a)))
	}
	return (*StringDict)(a.Payload().(*MyArena).arena.Resolve(MyStringDict, v.Data))
}

type StringArrayIterator struct {
	strArr *StringArray
	idx    int
}

func toStringArrayIterator(a *core.Arena, v core.Value) *StringArrayIterator {
	if v.Type != MyStringArrayIterator {
		panic(fmt.Sprintf("invalid type: expected StringArrayIterator, got %s", v.TypeName(a)))
	}
	return (*StringArrayIterator)(a.Payload().(*MyArena).arena.Resolve(MyStringArrayIterator, v.Data))
}

func init() {
	// Register Counter
	core.SetValueType(MyCounter, core.ValueTypeDescr{
		Interface: func(a *core.Arena, v core.Value) any { return toCounter(a, v) },
		Name:      func(a *core.Arena, v core.Value) string { return "counter" },
		String:    func(a *core.Arena, v core.Value) string { return fmt.Sprintf("Counter(%d)", toCounter(a, v).value) },
		AsString:  func(a *core.Arena, v core.Value) (string, bool) { return v.String(a), true },
		BinaryOp: func(a *core.Arena, v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			ma := a.Payload().(*MyArena)
			if rhs.Type == value.Int {
				o := toCounter(a, v)
				switch op {
				case token.Add:
					return ma.NewCounterValue(o.value + int64(rhs.Data)), nil
				case token.Sub:
					return ma.NewCounterValue(o.value - int64(rhs.Data)), nil
				}
			}
			if rhs.Type == MyCounter {
				o := toCounter(a, v)
				r := toCounter(a, rhs)
				switch op {
				case token.Add:
					return ma.NewCounterValue(o.value + r.value), nil
				case token.Sub:
					return ma.NewCounterValue(o.value - r.value), nil
				}
			}
			return core.Undefined, errors.New("invalid operator")
		},
		IsTrue: func(a *core.Arena, v core.Value) bool { return toCounter(a, v).value != 0 },
		Equal: func(a *core.Arena, v core.Value, r core.Value) bool {
			if r.Type != MyCounter {
				return false
			}
			return toCounter(a, v).value == toCounter(a, r).value
		},
		Clone: func(a *core.Arena, v core.Value) (core.Value, error) {
			ma := a.Payload().(*MyArena)
			return ma.NewCounterValue(toCounter(a, v).value), nil
		},
		Call: func(a *core.Arena, vm core.VM, v core.Value, args []core.Value) (core.Value, error) {
			return core.IntValue(toCounter(a, v).value), nil
		},
		IsCallable: func(a *core.Arena, v core.Value) bool { return true },
	})

	// Register CustomNumber
	core.SetValueType(MyCustomNumber, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "Number" },
		String: func(a *core.Arena, v core.Value) string { return strconv.FormatInt(toCustomNumber(a, v).value, 10) },
		BinaryOp: func(a *core.Arena, v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			r, ok := rhs.AsInt(a)
			if !ok {
				return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
			}
			i := toCustomNumber(a, v).value
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
			return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), t.TypeName(a))
		},
	})

	// Register StringArray
	core.SetValueType(MyStringArray, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "string-array" },
		String: func(a *core.Arena, v core.Value) string { return strings.Join(toStringArray(a, v).Value, ", ") },
		BinaryOp: func(a *core.Arena, v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			if rhs.Type == MyStringArray && op == token.Add {
				l := toStringArray(a, v)
				r := toStringArray(a, rhs)
				if len(r.Value) == 0 {
					return v, nil
				}
				ma := a.Payload().(*MyArena)
				return ma.NewStringArrayValue(append(l.Value, r.Value...)), nil
			}
			return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
		},
		IsTrue: func(a *core.Arena, v core.Value) bool { return len(toStringArray(a, v).Value) != 0 },
		Equal: func(a *core.Arena, v core.Value, rhs core.Value) bool {
			if rhs.Type == MyStringArray {
				l := toStringArray(a, v)
				r := toStringArray(a, rhs)
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
		Clone: func(a *core.Arena, v core.Value) (core.Value, error) {
			ma := a.Payload().(*MyArena)
			return ma.NewStringArrayValue(append([]string{}, toStringArray(a, v).Value...)), nil
		},
		Access: func(a *core.Arena, v core.Value, index core.Value, mode opcode.Opcode) (core.Value, error) {
			o := toStringArray(a, v)
			intIdx, ok := index.AsInt(a)
			if ok {
				if intIdx >= 0 && intIdx < int64(len(o.Value)) {
					return a.NewStringValue(o.Value[intIdx])
				}
				return core.Undefined, errs.NewIndexOutOfBoundsError("StringArray assignment", int(intIdx), len(o.Value))
			}
			strIdx, ok := index.AsString(a)
			if ok {
				for vidx, str := range o.Value {
					if strIdx == str {
						return core.IntValue(int64(vidx)), nil
					}
				}
				return core.Undefined, nil
			}
			return core.Undefined, errs.NewInvalidIndexTypeError("StringArray access", "int or string", index.TypeName(a))
		},
		Assign: func(a *core.Arena, v core.Value, index core.Value, value core.Value) error {
			o := toStringArray(a, v)
			strVal, ok := value.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringArray assignment", "string(compatible)", value.TypeName(a))
			}
			intIdx, ok := index.AsInt(a)
			if ok {
				if intIdx >= 0 && intIdx < int64(len(o.Value)) {
					o.Value[intIdx] = strVal
					return nil
				}
				return errs.NewIndexOutOfBoundsError("StringArray assignment", int(intIdx), len(o.Value))
			}
			return errs.NewInvalidIndexTypeError("StringArray assignment", "int", v.TypeName(a))
		},
		Call: func(a *core.Arena, vm core.VM, v core.Value, args []core.Value) (core.Value, error) {
			if len(args) != 1 {
				return core.Undefined, errs.NewWrongNumArgumentsError("StringArray.Call", "1", len(args))
			}
			s1, ok := args[0].AsString(a)
			if !ok {
				return core.Undefined, errs.NewInvalidArgumentTypeError("StringArray.Call", "first", "string(compatible)", args[0].TypeName(a))
			}
			o := toStringArray(a, v)
			for i, v := range o.Value {
				if v == s1 {
					return core.IntValue(int64(i)), nil
				}
			}
			return core.Undefined, nil
		},
		IsCallable: func(a *core.Arena, v core.Value) bool { return true },
		Iterator: func(a *core.Arena, v core.Value) (core.Value, error) {
			ma := a.Payload().(*MyArena)
			return ma.NewStringArrayIteratorValue(toStringArray(a, v)), nil
		},
		IsIterable: func(a *core.Arena, v core.Value) bool { return true },
	})

	// Register StringCircle
	core.SetValueType(MyStringCircle, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "string-circle" },
		String: func(a *core.Arena, v core.Value) string { return "" },
		Access: func(a *core.Arena, v core.Value, index core.Value, mode opcode.Opcode) (core.Value, error) {
			intIdx, ok := index.AsInt(a)
			if !ok {
				return core.Undefined, errs.NewInvalidIndexTypeError("StringCircle access", "int", index.TypeName(a))
			}
			o := toStringCircle(a, v)
			r := int(intIdx) % len(o.Value)
			if r < 0 {
				r = len(o.Value) + r
			}
			return a.NewStringValue(o.Value[r])
		},
		Assign: func(a *core.Arena, v core.Value, index core.Value, value core.Value) error {
			intIdx, ok := index.AsInt(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringCircle assignment", "int", index.TypeName(a))
			}
			o := toStringCircle(a, v)
			r := int(intIdx) % len(o.Value)
			if r < 0 {
				r = len(o.Value) + r
			}
			strVal, ok := value.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringCircle assignment", "string(compatible)", value.TypeName(a))
			}
			o.Value[r] = strVal
			return nil
		},
	})

	// Register StringDict
	core.SetValueType(MyStringDict, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "string-dict" },
		String: func(a *core.Arena, v core.Value) string { return "" },
		Access: func(a *core.Arena, v core.Value, index core.Value, mode opcode.Opcode) (core.Value, error) {
			strIdx, ok := index.AsString(a)
			if !ok {
				return core.Undefined, errs.NewInvalidIndexTypeError("StringDict access", "string", index.TypeName(a))
			}
			o := toStringDict(a, v)
			for k, v := range o.Value {
				if strings.EqualFold(strIdx, k) {
					return a.NewStringValue(v)
				}
			}
			return core.Undefined, nil
		},
		Assign: func(a *core.Arena, v core.Value, index core.Value, value core.Value) error {
			strIdx, ok := index.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringDict assignment", "string", index.TypeName(a))
			}
			strVal, ok := value.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringDict assignment", "string(compatible)", value.TypeName(a))
			}
			o := toStringDict(a, v)
			o.Value[strings.ToLower(strIdx)] = strVal
			return nil
		},
	})

	// Register StringArrayIterator
	core.SetValueType(MyStringArrayIterator, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "string-array-iterator" },
		String: func(a *core.Arena, v core.Value) string { return "" },
		Next: func(a *core.Arena, v core.Value) bool {
			i := toStringArrayIterator(a, v)
			i.idx++
			return i.idx <= len(i.strArr.Value)
		},
		Key: func(a *core.Arena, v core.Value) (core.Value, error) {
			i := toStringArrayIterator(a, v)
			return core.IntValue(int64(i.idx - 1)), nil
		},
		Value: func(a *core.Arena, v core.Value) (core.Value, error) {
			i := toStringArrayIterator(a, v)
			return a.NewStringValue(i.strArr.Value[i.idx-1])
		},
	})
}

func TestIndexable(t *testing.T) {
	ma := NewMyArena()
	opts := core.DefaultArenaOptions()
	opts.Payload = ma
	rta := core.NewArena(opts)

	dict := func() core.Value {
		return ma.NewStringDictValue(map[string]string{"a": "foo", "b": "bar"})
	}

	expectRun(t, rta, `out = d["a"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "foo")
	expectRun(t, rta, `out = d["B"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "bar")
	expectRun(t, rta, `out = d["x"]`, Opts().Symbol("d", dict()).Skip2ndPass(), core.Undefined)

	strCir := func() core.Value {
		return ma.NewStringCircleValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `out = cir[0]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "one")
	expectRun(t, rta, `out = cir[1]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "two")
	expectRun(t, rta, `out = cir[-1]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "three")
	expectRun(t, rta, `out = cir[-2]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "two")
	expectRun(t, rta, `out = cir[3]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "one")
	expectError(t, rta, `cir["a"]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "invalid_index_type")

	strArr := func() core.Value {
		return ma.NewStringArrayValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `out = arr["one"]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), 0)
	expectRun(t, rta, `out = arr["three"]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), 2)
	expectRun(t, rta, `out = arr["four"]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), core.Undefined)
	expectRun(t, rta, `out = arr[0]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "one")
	expectRun(t, rta, `out = arr[1]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "two")
	expectError(t, rta, `arr[-1]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "index_out_of_bounds")
}

func TestIndexAssignable(t *testing.T) {
	ma := NewMyArena()
	opts := core.DefaultArenaOptions()
	opts.Payload = ma
	rta := core.NewArena(opts)

	dict := func() core.Value {
		return ma.NewStringDictValue(map[string]string{"a": "foo", "b": "bar"})
	}

	expectRun(t, rta, `d["a"] = "1984"; out = d["a"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "1984")
	expectRun(t, rta, `d["c"] = "1984"; out = d["c"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "1984")
	expectRun(t, rta, `d["c"] = 1984; out = d["C"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "1984")

	strCir := func() core.Value {
		return ma.NewStringCircleValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `cir[0] = "ONE"; out = cir[0]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "ONE")
	expectRun(t, rta, `cir[1] = "TWO"; out = cir[1]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "TWO")
	expectRun(t, rta, `cir[-1] = "THREE"; out = cir[2]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "THREE")
	expectRun(t, rta, `cir[0] = "ONE"; out = cir[3]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "ONE")
	expectError(t, rta, `cir["a"] = "ONE"`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "invalid_index_type")

	strArr := func() core.Value {
		return ma.NewStringArrayValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `arr[0] = "ONE"; out = arr[0]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "ONE")
	expectRun(t, rta, `arr[1] = "TWO"; out = arr[1]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "TWO")
	expectError(t, rta, `arr["one"] = "ONE"`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "invalid_index_type")
}

func TestIterable(t *testing.T) {
	ma := NewMyArena()
	opts := core.DefaultArenaOptions()
	opts.Payload = ma
	rta := core.NewArena(opts)

	strArr := func() core.Value {
		return ma.NewStringArrayValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `out = 0; for i, s in arr { out += i }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), 3)
	expectRun(t, rta, `out = ""; for i, s in arr { out += s }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "onetwothree")
	expectRun(t, rta, `out = ""; for i, s in arr { out += s + i }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "one0two1three2")
}

func TestCompiled_CustomObject(t *testing.T) {
	ma := NewMyArena()
	opts := core.DefaultArenaOptions()
	opts.Payload = ma
	rta := core.NewArena(opts)

	c := compile(t, rta, `r := (t<130)`, MAP{"t": ma.NewCustomNumberValue(123)})
	compiledRun(t, rta, c)
	compiledGet(t, rta, c, "r", true)

	c = compile(t, rta, `r := (t>13)`, MAP{"t": ma.NewCustomNumberValue(123)})
	compiledRun(t, rta, c)
	compiledGet(t, rta, c, "r", true)
}
