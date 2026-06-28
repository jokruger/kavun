package core

import (
	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

type Seq[T any] struct {
	Elements []T
}

func (o *Seq[T]) Set(elements []T) {
	o.Elements = elements
}

// SeqForEach iterates over the elements of the sequence and calls the provided callback function for each element.
func SeqForEach[T any](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value,
	resolve func(Value) *Seq[T],
) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	o := resolve(v)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, e := range o.Elements {
			buf[0] = t2v(e)
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
			buf[1] = t2v(e)
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

// SeqFilter filters the elements of the sequence and returns a new sequence.
// If no arguments provided, it filters out zero values. If a function is provided, it filters out elements for which
// the function returns false. The function can have arity 1 (element) or 2 (index, element).
func SeqFilter[T comparable](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value, // T type constructor
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "0 or 1", len(args))
	}

	o := resolve(v)
	filtered := make([]T, 0, len(o.Elements))

	if len(args) == 0 {
		var zero T
		for _, e := range o.Elements {
			if e != zero {
				filtered = append(filtered, e)
			}
		}
		return alloc(filtered, false), nil
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value

	switch fn.Arity() {
	case 1:
		for _, e := range o.Elements {
			buf[0] = t2v(e)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, e)
			}
		}
		return alloc(filtered, false), nil

	case 2:
		for i, e := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = t2v(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, e)
			}
		}
		return alloc(filtered, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

// SeqCount counts the number of elements in the sequence that satisfy a given condition.
func SeqCount[T comparable](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "0 or 1", len(args))
	}

	o := resolve(v)
	var count int64

	if len(args) == 0 {
		var zero T
		for _, e := range o.Elements {
			if e != zero {
				count++
			}
		}
		return IntValue(count), nil
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value

	switch fn.Arity() {
	case 1:
		for _, e := range o.Elements {
			buf[0] = t2v(e)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	case 2:
		for i, e := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = t2v(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "f/1 or f/2", fn.TypeName())
	}
}

// SeqAll checks if all elements in the sequence satisfy a given condition.
func SeqAll[T any](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName())
	}

	o := resolve(v)
	var buf [2]Value

	switch fn.Arity() {
	case 1:
		for _, e := range o.Elements {
			buf[0] = t2v(e)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return False, nil
			}
		}
		return True, nil

	case 2:
		for i, e := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = t2v(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return False, nil
			}
		}
		return True, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "f/1 or f/2", fn.TypeName())
	}
}

// SeqAny checks if any element in the sequence satisfy a given condition.
func SeqAny[T any](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName())
	}

	o := resolve(v)
	var buf [2]Value

	switch fn.Arity() {
	case 1:
		for _, e := range o.Elements {
			buf[0] = t2v(e)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return True, nil
			}
		}
		return False, nil

	case 2:
		for i, e := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = t2v(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return True, nil
			}
		}
		return False, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "f/1 or f/2", fn.TypeName())
	}
}

// SeqMap applies a given function to each element in the sequence and returns a new sequence containing the results.
func SeqMap[T any](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("map", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("map", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	o := resolve(v)
	mapped := make([]Value, len(o.Elements))

	switch fn.Arity() {
	case 1:
		for i, e := range o.Elements {
			buf[0] = t2v(e)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			mapped[i] = res
		}
		return NewArrayValue(mapped, false), nil

	case 2:
		for i, e := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = t2v(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			mapped[i] = res
		}
		return NewArrayValue(mapped, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("map", "first", "f/1 or f/2", fn.TypeName())
	}
}

// SeqReduce reduces the sequence to a single value by applying a given binary function cumulatively to the elements of
// the sequence, from left to right.
// The function can have arity 2 (accumulator, element) or 3 (accumulator, index, element).
func SeqReduce[T any](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError("reduce", "2", len(args))
	}

	acc := args[0]
	fn := args[1]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("reduce", "second", "non-variadic function", fn.TypeName())
	}

	o := resolve(v)
	var buf [3]Value
	switch fn.Arity() {
	case 2:
		for _, e := range o.Elements {
			buf[0] = acc
			buf[1] = t2v(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	case 3:
		for i, e := range o.Elements {
			buf[0] = acc
			buf[1] = IntValue(int64(i))
			buf[2] = t2v(e)
			res, err := fn.Call(vm, buf[:3])
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("reduce", "second", "f/2 or f/3", fn.TypeName())
	}
}

// SeqFind searches for the first element in the sequence that satisfies a given condition and returns its index.
func SeqFind[T any](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	o := resolve(v)
	switch fn.Arity() {
	case 1:
		for i, e := range o.Elements {
			buf[0] = t2v(e)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	case 2:
		for i, e := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = t2v(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName())
	}
}

// SeqChunk divides the sequence into chunks of the specified size and returns a new sequence containing the chunks.
func SeqChunk[T any](
	v Value,
	args []Value,
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
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
		if args[1].Type != value.Bool {
			return Undefined, errs.NewInvalidArgumentTypeError("chunk", "second", "bool", args[1].TypeName())
		}
		copyChunks = args[1].IsTrue()
	}

	o := resolve(v)
	l := len(o.Elements)
	if l == 0 {
		return NewArrayValue(make([]Value, 0), false), nil
	}

	chunkCount := int((int64(l)-1)/size + 1)
	chunks := make([]Value, chunkCount)

	chunkSize := l
	if size < int64(l) {
		chunkSize = int(size)
	}

	for i, start := 0, 0; start < l; i, start = i+1, start+chunkSize {
		end := min(start+chunkSize, l)
		chunk := o.Elements[start:end]
		chunkImmutable := v.Immutable
		if copyChunks {
			chunk = make([]T, end-start)
			copy(chunk, o.Elements[start:end])
			chunkImmutable = false
		}
		chunks[i] = alloc(chunk, chunkImmutable)
	}

	return NewArrayValue(chunks, false), nil
}

// SeqNameHook returns a hook function that provides the type name for the sequence based on its mutability.
func SeqNameHook(
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
	resolve func(Value) *Seq[T], // T container resolver
	as func(Value) (T, bool), // Value to T convertor
	tn string, // T type name
) func(Value, Value, Value) error {
	return func(v Value, index Value, r Value) error {
		if v.Immutable {
			return errs.NewNotAssignableError(v.TypeName())
		}

		i := int64(index.Data) // optimistic scenario
		var ok bool
		if index.Type != value.Int {
			if i, ok = index.AsInt(); !ok {
				return errs.NewInvalidIndexTypeError("index assign", "int", index.TypeName())
			}
		}

		o := resolve(v)
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
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) func(Value, Value, opcode.Opcode) (Value, error) {
	return func(v Value, index Value, mode opcode.Opcode) (Value, error) {
		if mode != opcode.AccessIndex {
			return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
		}

		i := int64(index.Data) // optimistic scenario
		var ok bool
		if index.Type != value.Int {
			if i, ok = index.AsInt(); !ok {
				return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
			}
		}

		o := resolve(v)
		l := len(o.Elements)
		if i, ok = NormalizeIndex(i, int64(l)); !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), l)
		}

		return t2v(o.Elements[i]), nil
	}
}

// SeqSliceHook returns a hook function that allows slicing the sequence using start and end indices.
func SeqSliceHook[T any](
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
) func(Value, Value, Value) (Value, error) {
	return func(v Value, s Value, e Value) (Value, error) {
		var si, ei int64
		var ok bool

		o := resolve(v)
		l := int64(len(o.Elements))

		if s.Type != value.Undefined {
			si = int64(s.Data) // optimistic scenario
			if s.Type != value.Int {
				if si, ok = s.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
				}
			}
		}

		if e.Type != value.Undefined {
			ei = int64(e.Data) // optimistic scenario
			if e.Type != value.Int {
				if ei, ok = e.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
				}
			}
		}

		si, ei = NormalizeSliceBounds(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, l)
		return alloc(o.Elements[si:ei], v.Immutable), nil
	}
}

// SeqSliceStepHook returns a hook function that allows slicing the sequence using start and end indices with a
// specified step.
func SeqSliceStepHook[T any](
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
) func(Value, Value, Value, Value) (Value, error) {
	return func(v Value, s Value, e Value, stepVal Value) (Value, error) {
		var step, si, ei int64
		var ok bool

		o := resolve(v)
		l := int64(len(o.Elements))

		step = int64(stepVal.Data) // optimistic scenario
		if stepVal.Type != value.Int {
			if step, ok = stepVal.AsInt(); !ok {
				return Undefined, errs.NewInvalidIndexTypeError("slice step", "int", stepVal.TypeName())
			}
		}
		if step == 0 {
			return Undefined, errs.NewSliceStepZeroError()
		}

		if s.Type != value.Undefined {
			si = int64(s.Data) // optimistic scenario
			if s.Type != value.Int {
				if si, ok = s.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
				}
			}
		}
		if e.Type != value.Undefined {
			ei = int64(e.Data) // optimistic scenario
			if e.Type != value.Int {
				if ei, ok = e.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
				}
			}
		}

		start, end := NormalizeSliceBoundsStep(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, step, l)
		result := make([]T, 0)
		if step > 0 {
			for i := start; i < end; i += step {
				result = append(result, o.Elements[i])
			}
		} else {
			for i := start; i > end; i += step {
				result = append(result, o.Elements[i])
			}
		}

		return alloc(result, false), nil
	}
}
