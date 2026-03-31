package value

import "github.com/jokruger/gs/core"

type StringIterator struct {
	Object
	v []rune
	i int
	l int
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

func (o *StringIterator) Key(alloc core.Allocator) core.Object {
	return alloc.NewInt(int64(o.i - 1))
}

func (o *StringIterator) Value(alloc core.Allocator) core.Object {
	return alloc.NewChar(o.v[o.i-1])
}

func (o *StringIterator) TypeName() string {
	return "string-iterator"
}

func (o *StringIterator) String() string {
	return "<string-iterator>"
}

func (o *StringIterator) Copy(alloc core.Allocator) core.Object {
	t := alloc.NewStringIterator(o.v).(*StringIterator)
	t.i = o.i
	return t
}

func (o *StringIterator) IsTrue() bool {
	return o.v != nil && o.i <= o.l
}

func (o *StringIterator) IsFalse() bool {
	return !o.IsTrue()
}

func (o *StringIterator) AsBool() (bool, bool) {
	return o.IsTrue(), true
}
