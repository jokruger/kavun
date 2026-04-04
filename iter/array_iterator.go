package iter

import (
	"fmt"

	"github.com/jokruger/gs/core"
)

type ArrayIterator struct {
	v []core.Value
	i int
	l int
}

func (i *ArrayIterator) Set(v []core.Value) {
	i.v = v
	i.i = 0
	i.l = len(v)
}

func (i *ArrayIterator) TypeName() string {
	return "array-iterator"
}

func (i *ArrayIterator) String() string {
	return fmt.Sprintf("ArrayIterator{%d/%d}", i.i, i.l)
}

func (i *ArrayIterator) Next() bool {
	i.i++
	return i.i <= i.l
}

func (i *ArrayIterator) Key(core.Allocator) core.Value {
	return core.NewInt(int64(i.i - 1))
}

func (i *ArrayIterator) Value(core.Allocator) core.Value {
	return i.v[i.i-1]
}
