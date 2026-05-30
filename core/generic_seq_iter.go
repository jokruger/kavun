package core

import "fmt"

type SeqIter[T any] struct {
	Elements []T
	i        int
}

func (i *SeqIter[T]) Set(v []T) {
	i.Elements = v
	i.i = -1
}

func SeqIterEqual(a *Arena, v Value, r Value) bool {
	return v.Type == r.Type && v.Ptr == r.Ptr
}

func SeqIterNext[T any](a *Arena, v Value) bool {
	i := (*SeqIter[T])(v.Ptr)
	i.i++
	return i.i < len(i.Elements)
}

func SeqIterKey[T any](a *Arena, v Value) (Value, error) {
	i := (*SeqIter[T])(v.Ptr)
	return IntValue(int64(i.i)), nil
}

func SeqIterStringHook[T any](tn string) func(*Arena, Value) string {
	return func(a *Arena, v Value) string {
		i := (*SeqIter[T])(v.Ptr)
		return fmt.Sprintf("%s<%d, %d>", tn, i.i, len(i.Elements))
	}
}

func SeqIterValueHook[T any](
	t2v func(T) Value,
) func(*Arena, Value) (Value, error) {
	return func(a *Arena, v Value) (Value, error) {
		i := (*SeqIter[T])(v.Ptr)
		return t2v(i.Elements[i.i]), nil
	}
}
