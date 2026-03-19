package value

import (
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

var (
	// TrueValue is the singleton instance representing the boolean value true.
	TrueValue *Bool = &Bool{value: true}

	// FalseValue is the singleton instance representing the boolean value false.
	FalseValue *Bool = &Bool{value: false}

	// UndefinedValue is the singleton instance representing the undefined value.
	UndefinedValue *Undefined = &Undefined{}
)

/* === Object (Base) === */

type Object struct {
}

func (o *Object) TypeName() string {
	return "<object>"
}

func (o *Object) String() string {
	return o.TypeName()
}

func (o *Object) Interface() any {
	return o
}

func (o *Object) Arity() int {
	return 0
}

func (o *Object) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Object) Equals(x core.Object) bool {
	return o == x
}

func (o *Object) Copy() core.Object {
	return o
}

func (o *Object) Access(core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *Object) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Object) Iterate() core.Iterator {
	return nil
}

func (o *Object) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Object) IsFalsy() bool {
	return o == nil
}

func (o *Object) IsIterable() bool {
	return false
}

func (o *Object) IsCallable() bool {
	return false
}

func (o *Object) IsImmutable() bool {
	return false
}

func (o *Object) IsVariadic() bool {
	return false
}

func (o *Object) AsString() (string, bool) {
	return "", false
}

func (o *Object) AsInt() (int64, bool) {
	return 0, false
}

func (o *Object) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Object) AsBool() (bool, bool) {
	return false, false
}

func (o *Object) AsRune() (rune, bool) {
	return 0, false
}

func (o *Object) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Object) AsTime() (time.Time, bool) {
	return time.Time{}, false
}

/* === Array Iterator === */

type ArrayIterator struct {
	Object
	v []core.Object
	i int
	l int
}

func NewArrayIterator(v []core.Object) *ArrayIterator {
	o := &ArrayIterator{}
	o.Set(v)
	return o
}

func (o *ArrayIterator) Set(v []core.Object) {
	o.v = v
	o.i = 0
	o.l = len(v)
}

func (o *ArrayIterator) Next() bool {
	o.i++
	return o.i <= o.l
}

func (o *ArrayIterator) Key() core.Object {
	return NewInt(int64(o.i - 1))
}

func (o *ArrayIterator) Value() core.Object {
	return o.v[o.i-1]
}

func (o *ArrayIterator) TypeName() string {
	return "array-iterator"
}

func (o *ArrayIterator) String() string {
	return "<array-iterator>"
}

func (o *ArrayIterator) Equals(core.Object) bool {
	return false
}

func (o *ArrayIterator) Copy() core.Object {
	t := NewArrayIterator(o.v)
	t.i = o.i
	return t
}

func (o *ArrayIterator) IsFalsy() bool {
	return true
}

/* === Map Iterator === */

type MapIterator struct {
	Object
	v map[string]core.Object
	k []string
	i int
	l int
}

func NewMapIterator(m map[string]core.Object) *MapIterator {
	o := &MapIterator{}
	o.Set(m)
	return o
}

func (o *MapIterator) Set(m map[string]core.Object) {
	o.v = m
	o.k = make([]string, 0, len(m))
	for k := range m {
		o.k = append(o.k, k)
	}
	o.i = 0
	o.l = len(o.k)
}

func (o *MapIterator) Next() bool {
	o.i++
	return o.i <= o.l
}

func (o *MapIterator) Key() core.Object {
	k := o.k[o.i-1]
	return NewString(k)
}

func (o *MapIterator) Value() core.Object {
	k := o.k[o.i-1]
	return o.v[k]
}

func (o *MapIterator) TypeName() string {
	return "map-iterator"
}

func (o *MapIterator) String() string {
	return "<map-iterator>"
}

func (o *MapIterator) Equals(core.Object) bool {
	return false
}

func (o *MapIterator) Copy() core.Object {
	t := NewMapIterator(o.v)
	t.i = o.i
	return t
}

func (o *MapIterator) IsFalsy() bool {
	return true
}
