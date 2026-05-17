package core

import (
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/errs"
)

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

func HookConst[C any](c C) func(Value) C {
	return func(Value) C {
		return c
	}
}

func HookConst2[C1 any, C2 any](c1 C1, c2 C2) func(Value) (C1, C2) {
	return func(Value) (C1, C2) {
		return c1, c2
	}
}

func HookValue(v Value, e error) func(Value, *Arena) (Value, error) {
	return func(Value, *Arena) (Value, error) {
		return v, e
	}
}

func HookSelf(v Value, _ *Arena) (Value, error) {
	return v, nil
}

func HookSeqTypeName(name string, immutableName string) func(Value) string {
	return func(v Value) string {
		if v.Immutable {
			return immutableName
		}
		return name
	}
}

func HookSeqAssign[T any](as func(Value) (T, bool), tn string) func(Value, Value, Value) error {
	return func(v Value, index Value, r Value) error {
		if v.Immutable {
			return errs.NewNotAssignableError(v.TypeName())
		}

		i := int64(index.Data)
		var ok bool
		if index.Type != VT_INT {
			if i, ok = index.AsInt(); !ok {
				return errs.NewInvalidIndexTypeError("index assign", "int", index.TypeName())
			}
		}

		o := (*Seq[T])(v.Ptr)
		l := len(o.Elements)
		if i, ok = NormalizeIndex(i, int64(l)); !ok {
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

func HookSeqAccess[T any](ctor func(T) Value) func(Value, *Arena, Value, bc.Opcode) (Value, error) {
	return func(v Value, _ *Arena, index Value, mode bc.Opcode) (Value, error) {
		if mode != bc.OpIndex {
			return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
		}

		i := int64(index.Data)
		var ok bool
		if index.Type != VT_INT {
			if i, ok = index.AsInt(); !ok {
				return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
			}
		}

		o := (*Seq[T])(v.Ptr)
		l := len(o.Elements)
		if i, ok = NormalizeIndex(i, int64(l)); !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), l)
		}

		return ctor(o.Elements[i]), nil
	}
}
