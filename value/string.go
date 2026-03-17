package value

import (
	"strconv"
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type String struct {
	value string
	runes []rune
}

func NewString(s string) *String {
	o := &String{}
	o.Set(s)
	return o
}

func (o *String) Set(s string) {
	o.value = s
	o.runes = []rune(o.value)
}

func (o *String) Value() string {
	return o.value
}

func (o *String) Runes() []rune {
	return o.runes
}

func (o *String) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *String) Len() int {
	return len(o.runes)
}

func (o *String) At(i int) rune {
	return o.runes[i]
}

func (o *String) Get(i int) (rune, bool) {
	if i < 0 || i >= len(o.runes) {
		return 0, false
	}
	return o.runes[i], true
}

func (o *String) Slice(start, end int) string {
	return o.value[start:end]
}

func (o *String) Substring(start, end int) string {
	return string(o.runes[start:end])
}

func (o *String) Append(s string) {
	o.value += s
	o.runes = []rune(o.value)
}

func (o *String) TypeName() string {
	return "string"
}

func (o *String) String() string {
	return strconv.Quote(o.value)
}

func (o *String) Interface() any {
	return o.value
}

func (o *String) Arity() int {
	return 0
}

func (o *String) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	switch op {
	case token.Add:
		switch rhs := rhs.(type) {
		case *String:
			if len(o.value)+len(rhs.value) > core.MaxStringLen {
				return nil, gse.ErrStringLimit
			}
			return NewString(o.value + rhs.value), nil
		default:
			s := rhs.String()
			if len(o.value)+len(s) > core.MaxStringLen {
				return nil, gse.ErrStringLimit
			}
			return NewString(o.value + s), nil
		}
	case token.Less:
		switch rhs := rhs.(type) {
		case *String:
			if o.value < rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case token.LessEq:
		switch rhs := rhs.(type) {
		case *String:
			if o.value <= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case token.Greater:
		switch rhs := rhs.(type) {
		case *String:
			if o.value > rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case token.GreaterEq:
		switch rhs := rhs.(type) {
		case *String:
			if o.value >= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, gse.ErrInvalidOperator
}

func (o *String) Equals(x core.Object) bool {
	t, ok := x.(*String)
	if !ok {
		return false
	}
	return o.value == t.value
}

func (o *String) Copy() core.Object {
	return NewString(o.value)
}

func (o *String) IndexGet(index core.Object) (res core.Object, err error) {
	i, ok := index.AsInt()
	if !ok {
		err = gse.ErrInvalidIndexType
		return
	}
	if i < 0 || i >= int64(len(o.runes)) {
		res = UndefinedValue
		return
	}
	res = NewChar(o.runes[i])
	return
}

func (o *String) IndexSet(core.Object, core.Object) error {
	return gse.ErrNotIndexAssignable
}

func (o *String) Iterate() core.Iterator {
	return NewStringIterator(o.runes)
}

func (o *String) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *String) IsFalsy() bool {
	return len(o.value) == 0
}

func (o *String) IsIterable() bool {
	return true
}

func (o *String) IsCallable() bool {
	return false
}

func (o *String) IsImmutable() bool {
	return false
}

func (o *String) IsVariadic() bool {
	return false
}

func (o *String) AsString() (string, bool) {
	return o.value, true
}

func (o *String) AsInt() (int64, bool) {
	i, err := strconv.ParseInt(o.value, 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func (o *String) AsFloat() (float64, bool) {
	f, err := strconv.ParseFloat(o.value, 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func (o *String) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *String) AsRune() (rune, bool) {
	return 0, false
}

func (o *String) AsByteSlice() ([]byte, bool) {
	return []byte(o.value), true
}

func (o *String) AsTime() (time.Time, bool) {
	return time.Time{}, false
}
