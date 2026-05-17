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

func SeqTypeNameHook(name string, immutableName string) func(Value) string {
	return func(v Value) string {
		if v.Immutable {
			return immutableName
		}
		return name
	}
}

func SeqAssignHook[T any](as func(Value) (T, bool), tn string) func(Value, Value, Value) error {
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

func SeqAccessHook[T any](ctor func(T) Value) func(Value, *Arena, Value, bc.Opcode) (Value, error) {
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
