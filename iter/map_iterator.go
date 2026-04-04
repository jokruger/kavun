package iter

import (
	"fmt"

	"github.com/jokruger/gs/core"
)

type MapIterator struct {
	v map[string]core.Value
	k []string
	i int
	l int
}

func (i *MapIterator) Set(m map[string]core.Value) {
	i.v = m
	i.k = make([]string, 0, len(m))
	for k := range m {
		i.k = append(i.k, k)
	}
	i.i = 0
	i.l = len(i.k)
}

func (i *MapIterator) TypeName() string {
	return "map-iterator"
}

func (i *MapIterator) String() string {
	k := "<nil>"
	if i.i > 0 && i.i <= i.l {
		k = i.k[i.i-1]
	}
	return fmt.Sprintf("MapIterator{%s, %d/%d}", k, i.i, i.l)
}

func (i *MapIterator) Next() bool {
	i.i++
	return i.i <= i.l
}

func (i *MapIterator) Key(alloc core.Allocator) core.Value {
	return alloc.NewStringValue(i.k[i.i-1])
}

func (i *MapIterator) Value(core.Allocator) core.Value {
	k := i.k[i.i-1]
	return i.v[k]
}
