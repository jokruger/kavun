package value

import "github.com/jokruger/gs/core"

type StringIterator struct {
	Object
	v []rune
	i int
	l int
}

func NewStringIterator(v []rune) *StringIterator {
	o := &StringIterator{}
	o.Set(v)
	return o
}

func (o *StringIterator) Set(v []rune) {
	o.v = v
	o.i = 0
	o.l = len(v)
}

func (o *StringIterator) Next() bool {
	o.i++
	return o.i <= o.l
}

func (o *StringIterator) Key() core.Object {
	return NewInt(int64(o.i - 1))
}

func (o *StringIterator) Value() core.Object {
	return NewChar(o.v[o.i-1])
}

func (o *StringIterator) TypeName() string {
	return "string-iterator"
}

func (o *StringIterator) String() string {
	return "<string-iterator>"
}

func (o *StringIterator) Copy() core.Object {
	t := NewStringIterator(o.v)
	t.i = o.i
	return t
}

func (o *StringIterator) IsFalsy() bool {
	return o.v == nil || o.i > o.l
}
