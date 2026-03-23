package value

import (
	"strconv"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/internal/conv"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/token"
)

type String struct {
	Object
	value string
	runes []rune
}

// Should be used only for static initialization. For dynamic creation of built-in functions, use Allocator.NewString.
func NewStaticString(v string) core.Object {
	o := &String{}
	o.Set(v)
	return o
}

func (o *String) GobDecode(b []byte) error {
	o.Set(string(b))
	return nil
}

func (o *String) GobEncode() ([]byte, error) {
	return []byte(o.value), nil
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

func (o *String) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	alloc := vm.Allocator()
	switch op {
	case token.Add:
		switch rhs := rhs.(type) {
		case *String:
			if len(o.value)+len(rhs.value) > core.MaxStringLen {
				return nil, core.NewStringLimitError("string concatenation")
			}
			return alloc.NewString(o.value + rhs.value), nil
		default:
			s, ok := rhs.AsString()
			if !ok {
				return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
			}
			if len(o.value)+len(s) > core.MaxStringLen {
				return nil, core.NewStringLimitError("string concatenation")
			}
			return alloc.NewString(o.value + s), nil
		}
	case token.Less:
		switch rhs := rhs.(type) {
		case *String:
			return alloc.NewBool(o.value < rhs.value), nil
		}
	case token.LessEq:
		switch rhs := rhs.(type) {
		case *String:
			return alloc.NewBool(o.value <= rhs.value), nil
		}
	case token.Greater:
		switch rhs := rhs.(type) {
		case *String:
			return alloc.NewBool(o.value > rhs.value), nil
		}
	case token.GreaterEq:
		switch rhs := rhs.(type) {
		case *String:
			return alloc.NewBool(o.value >= rhs.value), nil
		}
	}
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *String) Equals(x core.Object) bool {
	t, ok := x.(*String)
	if !ok {
		return false
	}
	return o.value == t.value
}

func (o *String) Copy(alloc core.Allocator) core.Object {
	return alloc.NewString(o.value)
}

func (o *String) Access(vm core.VM, index core.Object, mode core.Opcode) (core.Object, error) {
	alloc := vm.Allocator()

	if mode == parser.OpSelect {
		return nil, core.NewInvalidAccessModeError("string", "select")
	}

	i, ok := index.AsInt()
	if !ok {
		return nil, core.NewInvalidIndexTypeError("string access", "int", index)
	}
	if i < 0 || i >= int64(len(o.runes)) {
		return alloc.NewUndefined(), nil
	}
	return alloc.NewChar(o.runes[i]), nil
}

func (o *String) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *String) Iterate(alloc core.Allocator) core.Iterator {
	return alloc.NewStringIterator(o.runes)
}

func (o *String) IsTrue() bool {
	return len(o.value) > 0
}

func (o *String) IsFalse() bool {
	return len(o.value) == 0
}

func (o *String) IsIterable() bool {
	return true
}

func (o *String) IsImmutable() bool {
	return true
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
	return conv.ParseBool(o.value)
}

func (o *String) AsRune() (rune, bool) {
	if len(o.runes) == 1 {
		return o.runes[0], true
	}
	return 0, false
}

func (o *String) AsBytes() ([]byte, bool) {
	return []byte(o.value), true
}

func (o *String) AsTime() (time.Time, bool) {
	// TODO: implement time parsing
	return time.Time{}, false
}
