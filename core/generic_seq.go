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

func SeqTypeIsTrue[T any](v Value) bool {
	return len((*Seq[T])(v.Ptr).Elements) > 0
}

func SeqTypeLen[T any](v Value) int64 {
	return int64(len((*Seq[T])(v.Ptr).Elements))
}

func SeqTypeNameHook(
	name string, // mutable type name
	immutableName string, // immutable type name
) func(Value) string {
	return func(v Value) string {
		if v.Immutable {
			return immutableName
		}
		return name
	}
}

func SeqAssignHook[T any](
	as func(Value) (T, bool), // Value to T convertor
	tn string, // T type name
) func(Value, Value, Value) error {
	return func(v Value, index Value, r Value) error {
		if v.Immutable {
			return errs.NewNotAssignableError(v.TypeName())
		}

		i := int64(index.Data) // optimistic scenario
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

func SeqAccessHook[T any](
	ctor func(T) Value, // T type constructor
) func(Value, *Arena, Value, bc.Opcode) (Value, error) {
	return func(v Value, _ *Arena, index Value, mode bc.Opcode) (Value, error) {
		if mode != bc.OpIndex {
			return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
		}

		i := int64(index.Data) // optimistic scenario
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

func SeqSliceHook[T any](
	ctor func(*Arena, []T, bool) Value, // T container constructor
) func(Value, *Arena, Value, Value) (Value, error) {
	return func(v Value, a *Arena, s Value, e Value) (Value, error) {
		var si, ei int64
		var ok bool

		o := (*Seq[T])(v.Ptr)
		l := int64(len(o.Elements))

		if s.Type != VT_UNDEFINED {
			si = int64(s.Data) // optimistic scenario
			if s.Type != VT_INT {
				if si, ok = s.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
				}
			}
		}

		if e.Type != VT_UNDEFINED {
			ei = int64(e.Data) // optimistic scenario
			if e.Type != VT_INT {
				if ei, ok = e.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
				}
			}
		}

		si, ei = NormalizeSliceBounds(si, s.Type != VT_UNDEFINED, ei, e.Type != VT_UNDEFINED, l)
		return ctor(a, o.Elements[si:ei], v.Immutable), nil
	}
}

func SeqSliceStepHook[T any](
	alloc func(*Arena, int, bool) []T, // T slice allocator
	ctor func(*Arena, []T, bool) Value, // T container constructor
) func(Value, *Arena, Value, Value, Value) (Value, error) {
	return func(v Value, a *Arena, s Value, e Value, stepVal Value) (Value, error) {
		var step, si, ei int64
		var ok bool

		o := (*Seq[T])(v.Ptr)
		l := int64(len(o.Elements))

		step = int64(stepVal.Data) // optimistic scenario
		if stepVal.Type != VT_INT {
			if step, ok = stepVal.AsInt(); !ok {
				return Undefined, errs.NewInvalidIndexTypeError("slice step", "int", stepVal.TypeName())
			}
		}
		if step == 0 {
			return Undefined, errs.NewSliceStepZeroError()
		}

		if s.Type != VT_UNDEFINED {
			si = int64(s.Data) // optimistic scenario
			if s.Type != VT_INT {
				if si, ok = s.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
				}
			}
		}
		if e.Type != VT_UNDEFINED {
			ei = int64(e.Data) // optimistic scenario
			if e.Type != VT_INT {
				if ei, ok = e.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
				}
			}
		}

		start, end := NormalizeSliceBoundsStep(si, s.Type != VT_UNDEFINED, ei, e.Type != VT_UNDEFINED, step, l)
		result := alloc(a, 0, false)
		if step > 0 {
			for i := start; i < end; i += step {
				result = append(result, o.Elements[i])
			}
		} else {
			for i := start; i > end; i += step {
				result = append(result, o.Elements[i])
			}
		}

		return ctor(a, result, false), nil
	}
}
