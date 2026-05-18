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

// SeqTypeIsTrue returns true if the sequence is not empty.
func SeqTypeIsTrue[T any](v Value) bool {
	return len((*Seq[T])(v.Ptr).Elements) > 0
}

// SeqTypeLen returns the length of the sequence.
func SeqTypeLen[T any](v Value) int64 {
	return int64(len((*Seq[T])(v.Ptr).Elements))
}

// SeqForEach iterates over the elements of the sequence and calls the provided callback function for each element.
func SeqForEach[T any](
	v Value,
	vm VM,
	args []Value,
	ctor func(T) Value,
) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	o := (*Seq[T])(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, e := range o.Elements {
			buf[0] = ctor(e)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return Undefined, nil
			}
		}

	case 2:
		for i, e := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = ctor(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return Undefined, nil
			}
		}
	}

	return Undefined, nil
}

// SeqChunk divides the sequence into chunks of the specified size and returns a new sequence containing the chunks.
func SeqChunk[T any](
	v Value,
	vm VM,
	args []Value,
	alloc func(*Arena, int, bool) []T, // T slice allocator
	ctor func(*Arena, []T, bool) Value, // T container allocator
) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError("chunk", "1 or 2", len(args))
	}

	size, ok := args[0].AsInt()
	if !ok {
		return Undefined, errs.NewInvalidArgumentTypeError("chunk", "first", "int", args[0].TypeName())
	}
	if size < 1 {
		return Undefined, errs.NewInvalidValueError("chunk size must be positive")
	}

	copyChunks := false
	if len(args) == 2 {
		if args[1].Type != VT_BOOL {
			return Undefined, errs.NewInvalidArgumentTypeError("chunk", "second", "bool", args[1].TypeName())
		}
		copyChunks = args[1].IsTrue()
	}

	a := vm.Allocator()
	o := (*Seq[T])(v.Ptr)
	l := len(o.Elements)
	if l == 0 {
		return a.NewArrayValue(a.NewArray(0, true), false), nil
	}

	chunkCount := int((int64(l)-1)/size + 1)
	chunks := a.NewArray(chunkCount, true)

	chunkSize := l
	if size < int64(l) {
		chunkSize = int(size)
	}

	for i, start := 0, 0; start < l; i, start = i+1, start+chunkSize {
		end := min(start+chunkSize, l)
		chunk := o.Elements[start:end]
		chunkImmutable := v.Immutable
		if copyChunks {
			chunk = alloc(a, end-start, true)
			copy(chunk, o.Elements[start:end])
			chunkImmutable = false
		}
		chunks[i] = ctor(a, chunk, chunkImmutable)
	}

	return a.NewArrayValue(chunks, false), nil
}

// SeqTypeNameHook returns a hook function that provides the type name for the sequence based on its mutability.
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

// SeqAssignHook returns a hook function that allows assigning a value to an element of the sequence at a specified
// index.
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

// SeqAccessHook returns a hook function that allows accessing an element of the sequence at a specified index.
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

// SeqSliceHook returns a hook function that allows slicing the sequence using start and end indices.
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

// SeqSliceStepHook returns a hook function that allows slicing the sequence using start and end indices with a
// specified step.
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
