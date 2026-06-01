package unit

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/token"
)

var cta = core.NewArena(nil)
var rta = core.NewArena(nil)

var (
	VT_COUNTER               = core.VT_USER_DEFINED + 1
	VT_CUSTOM_NUMBER         = core.VT_USER_DEFINED + 2
	VT_STRING_ARRAY          = core.VT_USER_DEFINED + 3
	VT_STRING_CIRCLE         = core.VT_USER_DEFINED + 4
	VT_STRING_DICT           = core.VT_USER_DEFINED + 5
	VT_STRING_ARRAY_ITERATOR = core.VT_USER_DEFINED + 6
)

type Counter struct {
	value int64
}

func NewCounterValue(val int64) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&Counter{value: val}),
		Type: VT_COUNTER,
	}
}

func toCounter(a *core.Arena, v core.Value) *Counter {
	if v.Type != VT_COUNTER {
		panic(fmt.Sprintf("invalid type: expected Counter, got %s", v.TypeName(a)))
	}
	return (*Counter)(v.Ptr)
}

type CustomNumber struct {
	value int64
}

func NewCustomNumberValue(val int64) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&CustomNumber{value: val}),
		Type: VT_CUSTOM_NUMBER,
	}
}

func toCustomNumber(a *core.Arena, v core.Value) *CustomNumber {
	if v.Type != VT_CUSTOM_NUMBER {
		panic(fmt.Sprintf("invalid type: expected CustomNumber, got %s", v.TypeName(a)))
	}
	return (*CustomNumber)(v.Ptr)
}

type StringArray struct {
	Value []string
}

func NewStringArrayValue(vals []string) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&StringArray{Value: vals}),
		Type: VT_STRING_ARRAY,
	}
}

func toStringArray(a *core.Arena, v core.Value) *StringArray {
	if v.Type != VT_STRING_ARRAY {
		panic(fmt.Sprintf("invalid type: expected StringArray, got %s", v.TypeName(a)))
	}
	return (*StringArray)(v.Ptr)
}

type StringCircle struct {
	Value []string
}

func NewStringCircleValue(vals []string) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&StringCircle{Value: vals}),
		Type: VT_STRING_CIRCLE,
	}
}

func toStringCircle(v core.Value) *StringCircle {
	if v.Type != VT_STRING_CIRCLE {
		panic(fmt.Sprintf("invalid type: expected StringCircle, got %s", v.TypeName(rta)))
	}
	return (*StringCircle)(v.Ptr)
}

type StringDict struct {
	Value map[string]string
}

func NewStringDictValue(vals map[string]string) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&StringDict{Value: vals}),
		Type: VT_STRING_DICT,
	}
}

func toStringDict(v core.Value) *StringDict {
	if v.Type != VT_STRING_DICT {
		panic(fmt.Sprintf("invalid type: expected StringDict, got %s", v.TypeName(rta)))
	}
	return (*StringDict)(v.Ptr)
}

type StringArrayIterator struct {
	strArr *StringArray
	idx    int
}

func NewStringArrayIteratorValue(arr *StringArray) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&StringArrayIterator{strArr: arr, idx: 0}),
		Type: VT_STRING_ARRAY_ITERATOR,
	}
}

func toStringArrayIterator(v core.Value) *StringArrayIterator {
	if v.Type != VT_STRING_ARRAY_ITERATOR {
		panic(fmt.Sprintf("invalid type: expected StringArrayIterator, got %s", v.TypeName(rta)))
	}
	return (*StringArrayIterator)(v.Ptr)
}

func init() {
	// Register Counter
	core.SetValueType(VT_COUNTER, core.ValueType{
		Interface: func(a *core.Arena, v core.Value) any { return toCounter(a, v) },
		Name:      func(a *core.Arena, v core.Value) string { return "counter" },
		String:    func(a *core.Arena, v core.Value) string { return fmt.Sprintf("Counter(%d)", toCounter(a, v).value) },
		AsString:  func(a *core.Arena, v core.Value) (string, bool) { return v.String(a), true },
		BinaryOp: func(a *core.Arena, v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			if rhs.Type == core.VT_INT {
				o := toCounter(a, v)
				switch op {
				case token.Add:
					return NewCounterValue(o.value + int64(rhs.Data)), nil
				case token.Sub:
					return NewCounterValue(o.value - int64(rhs.Data)), nil
				}
			}
			if rhs.Type == VT_COUNTER {
				o := toCounter(a, v)
				r := toCounter(a, rhs)
				switch op {
				case token.Add:
					return NewCounterValue(o.value + r.value), nil
				case token.Sub:
					return NewCounterValue(o.value - r.value), nil
				}
			}
			return core.Undefined, errors.New("invalid operator")
		},
		IsTrue: func(a *core.Arena, v core.Value) bool { return toCounter(a, v).value != 0 },
		Equal: func(a *core.Arena, v core.Value, r core.Value) bool {
			if r.Type != VT_COUNTER {
				return false
			}
			return toCounter(a, v).value == toCounter(a, r).value
		},
		Clone: func(a *core.Arena, v core.Value) (core.Value, error) {
			return NewCounterValue(toCounter(a, v).value), nil
		},
		Call: func(a *core.Arena, vm core.VM, v core.Value, args []core.Value) (core.Value, error) {
			return core.IntValue(toCounter(a, v).value), nil
		},
		IsCallable: func(a *core.Arena, v core.Value) bool { return true },
	})

	// Register CustomNumber
	core.SetValueType(VT_CUSTOM_NUMBER, core.ValueType{
		Name:   func(a *core.Arena, v core.Value) string { return "Number" },
		String: func(a *core.Arena, v core.Value) string { return strconv.FormatInt(toCustomNumber(a, v).value, 10) },
		BinaryOp: func(a *core.Arena, v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			r, ok := rhs.AsInt(a)
			if !ok {
				return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(rta), rhs.TypeName(rta))
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
			return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(rta), t.TypeName(rta))
		},
	})

	// Register StringArray
	core.SetValueType(VT_STRING_ARRAY, core.ValueType{
		Name:   func(a *core.Arena, v core.Value) string { return "string-array" },
		String: func(a *core.Arena, v core.Value) string { return strings.Join(toStringArray(a, v).Value, ", ") },
		BinaryOp: func(a *core.Arena, v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			if rhs.Type == VT_STRING_ARRAY && op == token.Add {
				l := toStringArray(a, v)
				r := toStringArray(a, rhs)
				if len(r.Value) == 0 {
					return v, nil
				}
				return NewStringArrayValue(append(l.Value, r.Value...)), nil
			}
			return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(rta), rhs.TypeName(rta))
		},
		IsTrue: func(a *core.Arena, v core.Value) bool { return len(toStringArray(a, v).Value) != 0 },
		Equal: func(a *core.Arena, v core.Value, rhs core.Value) bool {
			if rhs.Type == VT_STRING_ARRAY {
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
			return NewStringArrayValue(append([]string{}, toStringArray(a, v).Value...)), nil
		},
		Access: func(a *core.Arena, v core.Value, index core.Value, mode bc.Opcode) (core.Value, error) {
			o := toStringArray(a, v)
			intIdx, ok := index.AsInt(a)
			if ok {
				if intIdx >= 0 && intIdx < int64(len(o.Value)) {
					return core.NewStringValue(o.Value[intIdx]), nil
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
			return core.Undefined, errs.NewInvalidIndexTypeError("StringArray access", "int or string", index.TypeName(rta))
		},
		Assign: func(a *core.Arena, v core.Value, index core.Value, value core.Value) error {
			o := toStringArray(a, v)
			strVal, ok := value.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringArray assignment", "string(compatible)", value.TypeName(rta))
			}
			intIdx, ok := index.AsInt(a)
			if ok {
				if intIdx >= 0 && intIdx < int64(len(o.Value)) {
					o.Value[intIdx] = strVal
					return nil
				}
				return errs.NewIndexOutOfBoundsError("StringArray assignment", int(intIdx), len(o.Value))
			}
			return errs.NewInvalidIndexTypeError("StringArray assignment", "int", v.TypeName(rta))
		},
		Call: func(a *core.Arena, vm core.VM, v core.Value, args []core.Value) (core.Value, error) {
			if len(args) != 1 {
				return core.Undefined, errs.NewWrongNumArgumentsError("StringArray.Call", "1", len(args))
			}
			s1, ok := args[0].AsString(a)
			if !ok {
				return core.Undefined, errs.NewInvalidArgumentTypeError("StringArray.Call", "first", "string(compatible)", args[0].TypeName(rta))
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
			return NewStringArrayIteratorValue(toStringArray(a, v)), nil
		},
		IsIterable: func(a *core.Arena, v core.Value) bool { return true },
	})

	// Register StringCircle
	core.SetValueType(VT_STRING_CIRCLE, core.ValueType{
		Name:   func(a *core.Arena, v core.Value) string { return "string-circle" },
		String: func(a *core.Arena, v core.Value) string { return "" },
		Access: func(a *core.Arena, v core.Value, index core.Value, mode bc.Opcode) (core.Value, error) {
			intIdx, ok := index.AsInt(a)
			if !ok {
				return core.Undefined, errs.NewInvalidIndexTypeError("StringCircle access", "int", index.TypeName(rta))
			}
			o := toStringCircle(v)
			r := int(intIdx) % len(o.Value)
			if r < 0 {
				r = len(o.Value) + r
			}
			return a.NewStringValue(o.Value[r]), nil
		},
		Assign: func(a *core.Arena, v core.Value, index core.Value, value core.Value) error {
			intIdx, ok := index.AsInt(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringCircle assignment", "int", index.TypeName(rta))
			}
			o := toStringCircle(v)
			r := int(intIdx) % len(o.Value)
			if r < 0 {
				r = len(o.Value) + r
			}
			strVal, ok := value.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringCircle assignment", "string(compatible)", value.TypeName(rta))
			}
			o.Value[r] = strVal
			return nil
		},
	})

	// Register StringDict
	core.SetValueType(VT_STRING_DICT, core.ValueType{
		Name:   func(a *core.Arena, v core.Value) string { return "string-dict" },
		String: func(a *core.Arena, v core.Value) string { return "" },
		Access: func(a *core.Arena, v core.Value, index core.Value, mode bc.Opcode) (core.Value, error) {
			strIdx, ok := index.AsString(a)
			if !ok {
				return core.Undefined, errs.NewInvalidIndexTypeError("StringDict access", "string", index.TypeName(rta))
			}
			o := toStringDict(v)
			for k, v := range o.Value {
				if strings.EqualFold(strIdx, k) {
					return core.NewStringValue(v), nil
				}
			}
			return core.Undefined, nil
		},
		Assign: func(a *core.Arena, v core.Value, index core.Value, value core.Value) error {
			strIdx, ok := index.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringDict assignment", "string", index.TypeName(rta))
			}
			strVal, ok := value.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringDict assignment", "string(compatible)", value.TypeName(rta))
			}
			o := toStringDict(v)
			o.Value[strings.ToLower(strIdx)] = strVal
			return nil
		},
	})

	// Register StringArrayIterator
	core.SetValueType(VT_STRING_ARRAY_ITERATOR, core.ValueType{
		Name:   func(a *core.Arena, v core.Value) string { return "string-array-iterator" },
		String: func(a *core.Arena, v core.Value) string { return "" },
		Next: func(a *core.Arena, v core.Value) bool {
			i := toStringArrayIterator(v)
			i.idx++
			return i.idx <= len(i.strArr.Value)
		},
		Key: func(a *core.Arena, v core.Value) (core.Value, error) {
			i := toStringArrayIterator(v)
			return core.IntValue(int64(i.idx - 1)), nil
		},
		Value: func(a *core.Arena, v core.Value) (core.Value, error) {
			i := toStringArrayIterator(v)
			return core.NewStringValue(i.strArr.Value[i.idx-1]), nil
		},
	})
}
