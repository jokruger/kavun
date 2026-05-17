package seq

import (
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal"
)

// Seq is a generic sequence type that can hold elements of any type T.
type Seq[T any] struct {
	Elements []T
}

func (o *Seq[T]) Set(elements []T) {
	o.Elements = elements
}

// NormalizeIndex normalizes index for Python-style indexing (-1 = last element, -2 = second to last, etc.) and checks
// if it's within bounds.
func NormalizeIndex(index int64, length int64) (int64, bool) {
	if index < 0 {
		index += length
	}
	if index < 0 || index >= length {
		return index, false
	}
	return index, true
}

// Assign creates a sequence type assign hook.
func AssignHook[V internal.Value, T any](as func(V) (T, bool), tn string) func(V, V, V) error {
	return func(v V, index V, r V) error {
		if v.IsImmutable() {
			return errs.NewNotAssignableError(v.TypeName())
		}

		i, ok := index.AsInt()
		if !ok {
			return errs.NewInvalidIndexTypeError("index assign", "int", index.TypeName())
		}

		o := (*Seq[T])(v.GetPtr())
		l := len(o.Elements)
		i, ok = NormalizeIndex(i, int64(l))
		if !ok {
			return errs.NewIndexOutOfBoundsError("index assign", int(i), l)
		}

		c, ok := as(r)
		if !ok {
			return errs.NewInvalidIndexTypeError("index assign value", tn, r.TypeName())
		}

		o.Elements[i] = c

		return nil
	}
}

// Access creates a sequence type access hook.
func AccessHook[V internal.Value, A any, T any](ctor func(T) V) func(V, *A, V, bc.Opcode) (V, error) {
	return func(v V, _ *A, index V, mode bc.Opcode) (V, error) {
		if mode != bc.OpIndex {
			var zero V
			return zero, errs.NewInvalidSelectorError(v.TypeName(), index.String())
		}

		i, ok := index.AsInt()
		if !ok {
			var zero V
			return zero, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}

		o := (*Seq[T])(v.GetPtr())
		l := len(o.Elements)
		i, ok = NormalizeIndex(i, int64(l))
		if !ok {
			var zero V
			return zero, errs.NewIndexOutOfBoundsError("index access", int(i), l)
		}

		return ctor(o.Elements[i]), nil
	}
}
