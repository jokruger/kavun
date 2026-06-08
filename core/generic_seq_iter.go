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

func SeqIterNextHook[T any](
	resolve func(a *Arena, v Value) *SeqIter[T],
) func(*Arena, Value) bool {
	return func(a *Arena, v Value) bool {
		i := resolve(a, v)
		i.i++
		return i.i < len(i.Elements)
	}
}

func SeqIterKeyHook[T any](
	resolve func(a *Arena, v Value) *SeqIter[T],
) func(*Arena, Value) (Value, error) {
	return func(a *Arena, v Value) (Value, error) {
		i := resolve(a, v)
		return IntValue(int64(i.i)), nil
	}
}

func SeqIterStringHook[T any](
	tn string,
	resolve func(a *Arena, v Value) *SeqIter[T],
) func(*Arena, Value) string {
	return func(a *Arena, v Value) string {
		i := resolve(a, v)
		return fmt.Sprintf("%s<%d, %d>", tn, i.i, len(i.Elements))
	}
}

func SeqIterValueHook[T any](
	t2v func(T) Value,
	resolve func(a *Arena, v Value) *SeqIter[T],
) func(*Arena, Value) (Value, error) {
	return func(a *Arena, v Value) (Value, error) {
		i := resolve(a, v)
		return t2v(i.Elements[i.i]), nil
	}
}
