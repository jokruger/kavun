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

// LOCALISED-STATE: returned Next hook advances the iterator's internal cursor. Iterators are expected to be held
// by a single consumer for the duration of iteration; the optimizer never speculatively evaluates iterator
// advancement. See docs/purity.md.
func SeqIterNextHook[T any](
	resolve func(v Value) *SeqIter[T],
) func(Value) bool {
	return func(v Value) bool {
		i := resolve(v)
		i.i++
		return i.i < len(i.Elements)
	}
}

// PURE: returned Key hook reads the iterator's current cursor without advancing it. See docs/purity.md.
func SeqIterKeyHook[T any](
	resolve func(v Value) *SeqIter[T],
) func(Value) (Value, error) {
	return func(v Value) (Value, error) {
		i := resolve(v)
		return IntValue(int64(i.i)), nil
	}
}

func SeqIterStringHook[T any](
	tn string,
	resolve func(v Value) *SeqIter[T],
) func(Value) string {
	return func(v Value) string {
		i := resolve(v)
		return fmt.Sprintf("%s<%d, %d>", tn, i.i, len(i.Elements))
	}
}

// PURE: returned Value hook reads the iterator's current element without advancing it. See docs/purity.md.
func SeqIterValueHook[T any](
	t2v func(T) Value,
	resolve func(v Value) *SeqIter[T],
) func(Value) (Value, error) {
	return func(v Value) (Value, error) {
		i := resolve(v)
		return t2v(i.Elements[i.i]), nil
	}
}
