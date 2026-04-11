package unit

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
	mock "github.com/jokruger/gs/tests"
	"github.com/jokruger/gs/token"
)

var alloc = mock.Alloc

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

func toCounter(v core.Value) *Counter {
	if v.Type != VT_COUNTER {
		panic(fmt.Sprintf("invalid type: expected Counter, got %s", v.TypeName()))
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

func toCustomNumber(v core.Value) *CustomNumber {
	if v.Type != VT_CUSTOM_NUMBER {
		panic(fmt.Sprintf("invalid type: expected CustomNumber, got %s", v.TypeName()))
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

func toStringArray(v core.Value) *StringArray {
	if v.Type != VT_STRING_ARRAY {
		panic(fmt.Sprintf("invalid type: expected StringArray, got %s", v.TypeName()))
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
		panic(fmt.Sprintf("invalid type: expected StringCircle, got %s", v.TypeName()))
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
		panic(fmt.Sprintf("invalid type: expected StringDict, got %s", v.TypeName()))
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
		panic(fmt.Sprintf("invalid type: expected StringArrayIterator, got %s", v.TypeName()))
	}
	return (*StringArrayIterator)(v.Ptr)
}

func init() {
	// Register Counter
	core.TypeInterface[VT_COUNTER] = func(v core.Value) any {
		return toCounter(v)
	}
	core.TypeName[VT_COUNTER] = func(v core.Value) string {
		return "counter"
	}
	core.TypeString[VT_COUNTER] = func(v core.Value) string {
		return fmt.Sprintf("Counter(%d)", toCounter(v).value)
	}
	core.TypeAsString[VT_COUNTER] = func(v core.Value) (string, bool) {
		return v.String(), true
	}
	core.TypeBinaryOp[VT_COUNTER] = func(v core.Value, a core.Allocator, op token.Token, rhs core.Value) (core.Value, error) {
		if rhs.IsInt() {
			o := toCounter(v)
			switch op {
			case token.Add:
				return NewCounterValue(o.value + core.ToInt(rhs)), nil
			case token.Sub:
				return NewCounterValue(o.value - core.ToInt(rhs)), nil
			}
		}
		if rhs.Type == VT_COUNTER {
			o := toCounter(v)
			r := toCounter(rhs)
			switch op {
			case token.Add:
				return NewCounterValue(o.value + r.value), nil
			case token.Sub:
				return NewCounterValue(o.value - r.value), nil
			}
		}
		return core.UndefinedValue(), errors.New("invalid operator")
	}
	core.TypeIsTrue[VT_COUNTER] = func(v core.Value) bool {
		return toCounter(v).value != 0
	}
	core.TypeEqual[VT_COUNTER] = func(v core.Value, r core.Value) bool {
		if r.Type != VT_COUNTER {
			return false
		}
		return toCounter(v).value == toCounter(r).value
	}
	core.TypeCopy[VT_COUNTER] = func(v core.Value, alloc core.Allocator) core.Value {
		return NewCounterValue(toCounter(v).value)
	}
	core.TypeCall[VT_COUNTER] = func(v core.Value, vm core.VM, args []core.Value) (core.Value, error) {
		return core.IntValue(toCounter(v).value), nil
	}
	core.TypeIsCallable[VT_COUNTER] = func(v core.Value) bool {
		return true
	}

	// Register CustomNumber
	core.TypeName[VT_CUSTOM_NUMBER] = func(v core.Value) string {
		return "Number"
	}
	core.TypeString[VT_CUSTOM_NUMBER] = func(v core.Value) string {
		return strconv.FormatInt(toCustomNumber(v).value, 10)
	}
	core.TypeBinaryOp[VT_CUSTOM_NUMBER] = func(v core.Value, a core.Allocator, op token.Token, rhs core.Value) (core.Value, error) {
		r, ok := rhs.AsInt()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
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
		return core.UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), t.TypeName())
	}

	// Register StringArray
	core.TypeName[VT_STRING_ARRAY] = func(v core.Value) string {
		return "string-array"
	}
	core.TypeString[VT_STRING_ARRAY] = func(v core.Value) string {
		return strings.Join(toStringArray(v).Value, ", ")
	}
	core.TypeBinaryOp[VT_STRING_ARRAY] = func(v core.Value, a core.Allocator, op token.Token, rhs core.Value) (core.Value, error) {
		if rhs.Type == VT_STRING_ARRAY && op == token.Add {
			l := toStringArray(v)
			r := toStringArray(rhs)
			if len(r.Value) == 0 {
				return v, nil
			}
			return NewStringArrayValue(append(l.Value, r.Value...)), nil
		}
		return core.UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}
	core.TypeIsTrue[VT_STRING_ARRAY] = func(v core.Value) bool {
		return len(toStringArray(v).Value) != 0
	}
	core.TypeEqual[VT_STRING_ARRAY] = func(v core.Value, rhs core.Value) bool {
		if rhs.Type == VT_STRING_ARRAY {
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
	}
	core.TypeCopy[VT_STRING_ARRAY] = func(v core.Value, alloc core.Allocator) core.Value {
		return NewStringArrayValue(append([]string{}, toStringArray(v).Value...))
	}
	core.TypeAccess[VT_STRING_ARRAY] = func(v core.Value, a core.Allocator, index core.Value, mode core.Opcode) (core.Value, error) {
		o := toStringArray(v)
		intIdx, ok := index.AsInt()
		if ok {
			if intIdx >= 0 && intIdx < int64(len(o.Value)) {
				return alloc.NewStringValue(o.Value[intIdx]), nil
			}
			return core.UndefinedValue(), errs.NewIndexOutOfBoundsError("StringArray assignment", int(intIdx), len(o.Value))
		}
		strIdx, ok := index.AsString()
		if ok {
			for vidx, str := range o.Value {
				if strIdx == str {
					return core.IntValue(int64(vidx)), nil
				}
			}
			return core.UndefinedValue(), nil
		}
		return core.UndefinedValue(), errs.NewInvalidIndexTypeError("StringArray access", "int or string", index.TypeName())
	}
	core.TypeAssign[VT_STRING_ARRAY] = func(v core.Value, index core.Value, value core.Value) error {
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
	}
	core.TypeCall[VT_STRING_ARRAY] = func(v core.Value, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.UndefinedValue(), errs.NewWrongNumArgumentsError("StringArray.Call", "1", len(args))
		}
		s1, ok := args[0].AsString()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("StringArray.Call", "first", "string(compatible)", args[0].TypeName())
		}
		o := toStringArray(v)
		for i, v := range o.Value {
			if v == s1 {
				return core.IntValue(int64(i)), nil
			}
		}
		return core.UndefinedValue(), nil
	}
	core.TypeIsCallable[VT_STRING_ARRAY] = func(v core.Value) bool {
		return true
	}
	core.TypeIterator[VT_STRING_ARRAY] = func(v core.Value, alloc core.Allocator) core.Value {
		return NewStringArrayIteratorValue(toStringArray(v))
	}
	core.TypeIsIterable[VT_STRING_ARRAY] = func(v core.Value) bool {
		return true
	}

	// Register StringCircle
	core.TypeName[VT_STRING_CIRCLE] = func(v core.Value) string {
		return "string-circle"
	}
	core.TypeString[VT_STRING_CIRCLE] = func(v core.Value) string {
		return ""
	}
	core.TypeAccess[VT_STRING_CIRCLE] = func(v core.Value, a core.Allocator, index core.Value, mode core.Opcode) (core.Value, error) {
		intIdx, ok := index.AsInt()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidIndexTypeError("StringCircle access", "int", index.TypeName())
		}
		o := toStringCircle(v)
		r := int(intIdx) % len(o.Value)
		if r < 0 {
			r = len(o.Value) + r
		}
		return a.NewStringValue(o.Value[r]), nil
	}
	core.TypeAssign[VT_STRING_CIRCLE] = func(v core.Value, index core.Value, value core.Value) error {
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
	}

	// Register StringDict
	core.TypeName[VT_STRING_DICT] = func(v core.Value) string {
		return "string-dict"
	}
	core.TypeString[VT_STRING_DICT] = func(v core.Value) string {
		return ""
	}
	core.TypeAccess[VT_STRING_DICT] = func(v core.Value, a core.Allocator, index core.Value, mode core.Opcode) (core.Value, error) {
		strIdx, ok := index.AsString()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidIndexTypeError("StringDict access", "string", index.TypeName())
		}
		o := toStringDict(v)
		for k, v := range o.Value {
			if strings.EqualFold(strIdx, k) {
				return alloc.NewStringValue(v), nil
			}
		}
		return core.UndefinedValue(), nil
	}
	core.TypeAssign[VT_STRING_DICT] = func(v core.Value, index core.Value, value core.Value) error {
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
	}

	// Register StringArrayIterator
	core.TypeName[VT_STRING_ARRAY_ITERATOR] = func(v core.Value) string {
		return "string-array-iterator"
	}
	core.TypeString[VT_STRING_ARRAY_ITERATOR] = func(v core.Value) string {
		return ""
	}
	core.TypeNext[VT_STRING_ARRAY_ITERATOR] = func(v *core.Value) bool {
		i := toStringArrayIterator(*v)
		i.idx++
		return i.idx <= len(i.strArr.Value)
	}
	core.TypeKey[VT_STRING_ARRAY_ITERATOR] = func(v core.Value, alloc core.Allocator) core.Value {
		i := toStringArrayIterator(v)
		return core.IntValue(int64(i.idx - 1))
	}
	core.TypeValue[VT_STRING_ARRAY_ITERATOR] = func(v core.Value, alloc core.Allocator) core.Value {
		i := toStringArrayIterator(v)
		return alloc.NewStringValue(i.strArr.Value[i.idx-1])
	}
}
