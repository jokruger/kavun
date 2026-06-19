package core

import (
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/refpool"
)

/* Helper functions used in combination with generics */

func ArenaNewBytes(a *Arena, capacity int, sized bool) []byte {
	return a.NewBytes(capacity, sized)
}

func ArenaNewRunes(a *Arena, capacity int, sized bool) []rune {
	return a.NewRunes(capacity, sized)
}

func ArenaNewArray(a *Arena, capacity int, sized bool) []Value {
	return a.NewArray(capacity, sized)
}

func ArenaNewRunesValue(a *Arena, r []rune, immutable bool) (Value, error) {
	return a.NewRunesValue(r, immutable)
}

func ArenaNewBytesValue(a *Arena, b []byte, immutable bool) (Value, error) {
	return a.NewBytesValue(b, immutable)
}

func ArenaNewArrayValue(a *Arena, arr []Value, immutable bool) (Value, error) {
	return a.NewArrayValue(arr, immutable)
}

type UserTypeArena interface {
	Pin(Value)
	Retain(Value)
	Release(Value)
	Reset()
}

type Static struct {
	Primitives        []Value
	Decimals          []dec128.Dec128
	Strings           []string
	Runes             []Runes
	FormatSpecs       []FormatSpec
	CompiledFunctions []CompiledFunction
}

type ArenaOptions struct {
	Static  *Static
	Payload UserTypeArena

	DecimalBuf int
	StringBuf  int
	TimeBuf    int

	IntRangeBuf         int
	ArrayIteratorBuf    int
	BytesIteratorBuf    int
	DictIteratorBuf     int
	IntRangeIteratorBuf int
	RunesIteratorBuf    int

	ArrayBuf  int
	BytesBuf  int
	RunesBuf  int
	RecordBuf int
	DictBuf   int

	ErrorBuf            int
	BuiltinClosureBuf   int
	CompiledFunctionBuf int

	ValuePtrBuf int

	ZeroOnRelease bool // whether Release should reset value to zero
	ZeroOnReset   bool // whether Reset should reset values to zero
	ResetFull     bool // whether Reset should reset all pools to the initial capacity
}

func DefaultArenaOptions() *ArenaOptions {
	return &ArenaOptions{
		DecimalBuf: 256,
		StringBuf:  256,
		TimeBuf:    256,

		IntRangeBuf:         32,
		ArrayIteratorBuf:    32,
		BytesIteratorBuf:    32,
		DictIteratorBuf:     32,
		IntRangeIteratorBuf: 32,
		RunesIteratorBuf:    32,

		ArrayBuf:  64,
		BytesBuf:  32,
		RunesBuf:  256,
		RecordBuf: 64,
		DictBuf:   64,

		ErrorBuf:            0,
		BuiltinClosureBuf:   64,
		CompiledFunctionBuf: 64,

		ValuePtrBuf: 256,

		ZeroOnRelease: true,
		ZeroOnReset:   true,
		ResetFull:     true,
	}
}

type Arena struct {
	static    *Static
	arena     *refpool.Arena
	payload   UserTypeArena
	resetFull bool
}

// NewArena creates a new Arena with the given options. If opts is nil, it uses the default options.
func NewArena(opts *ArenaOptions) *Arena {
	if opts == nil {
		opts = DefaultArenaOptions()
	}

	return &Arena{
		static:    opts.Static,
		payload:   opts.Payload,
		resetFull: opts.ResetFull,
		arena: refpool.NewArena(
			opts.ZeroOnRelease,
			opts.ZeroOnReset,
			refpool.With[dec128.Dec128](value.Decimal, opts.DecimalBuf),
			refpool.With[time.Time](value.Time, opts.TimeBuf),
			refpool.With[string](value.String, opts.StringBuf),
			refpool.With[Record](value.Record, opts.RecordBuf),
			refpool.With[Dict](value.Dict, opts.DictBuf),
			refpool.With[Array](value.Array, opts.ArrayBuf),
			refpool.With[Bytes](value.Bytes, opts.BytesBuf),
			refpool.With[Runes](value.Runes, opts.RunesBuf),
			refpool.With[IntRange](value.IntRange, opts.IntRangeBuf),
			refpool.With[DictIterator](value.DictIterator, opts.DictIteratorBuf),
			refpool.With[ArrayIterator](value.ArrayIterator, opts.ArrayIteratorBuf),
			refpool.With[BytesIterator](value.BytesIterator, opts.BytesIteratorBuf),
			refpool.With[RunesIterator](value.RunesIterator, opts.RunesIteratorBuf),
			refpool.With[IntRangeIterator](value.IntRangeIterator, opts.IntRangeIteratorBuf),
			refpool.With[Error](value.Error, opts.ErrorBuf),
			refpool.With[BuiltinClosure](value.BuiltinClosure, opts.BuiltinClosureBuf),
			refpool.With[CompiledFunction](value.CompiledFunction, opts.CompiledFunctionBuf),
			refpool.With[*Value](value.ValuePtr, opts.ValuePtrBuf),
		),
	}
}

func (a *Arena) Stats() (allocated, used, free int) {
	s := func(a, u, f int) {
		allocated += a
		used += u
		free += f
	}

	s(a.arena.Stats(value.Decimal))
	s(a.arena.Stats(value.Time))
	s(a.arena.Stats(value.String))
	s(a.arena.Stats(value.Record))
	s(a.arena.Stats(value.Dict))
	s(a.arena.Stats(value.Array))
	s(a.arena.Stats(value.Bytes))
	s(a.arena.Stats(value.Runes))
	s(a.arena.Stats(value.IntRange))
	s(a.arena.Stats(value.DictIterator))
	s(a.arena.Stats(value.ArrayIterator))
	s(a.arena.Stats(value.BytesIterator))
	s(a.arena.Stats(value.RunesIterator))
	s(a.arena.Stats(value.IntRangeIterator))
	s(a.arena.Stats(value.Error))
	s(a.arena.Stats(value.BuiltinClosure))
	s(a.arena.Stats(value.CompiledFunction))
	s(a.arena.Stats(value.ValuePtr))

	return
}

func (a *Arena) SetStatic(static *Static) {
	a.static = static
}

func (a *Arena) Static() *Static {
	return a.static
}

func (a *Arena) Payload() any {
	return a.payload
}

func (a *Arena) Reset() {
	if a.payload != nil {
		a.payload.Reset()
	}

	a.arena.Reset(value.Decimal, a.resetFull)
	a.arena.Reset(value.Time, a.resetFull)
	a.arena.Reset(value.String, a.resetFull)
	a.arena.Reset(value.Record, a.resetFull)
	a.arena.Reset(value.Dict, a.resetFull)
	a.arena.Reset(value.Array, a.resetFull)
	a.arena.Reset(value.Bytes, a.resetFull)
	a.arena.Reset(value.Runes, a.resetFull)
	a.arena.Reset(value.IntRange, a.resetFull)
	a.arena.Reset(value.DictIterator, a.resetFull)
	a.arena.Reset(value.ArrayIterator, a.resetFull)
	a.arena.Reset(value.BytesIterator, a.resetFull)
	a.arena.Reset(value.RunesIterator, a.resetFull)
	a.arena.Reset(value.IntRangeIterator, a.resetFull)
	a.arena.Reset(value.Error, a.resetFull)
	a.arena.Reset(value.BuiltinClosure, a.resetFull)
	a.arena.Reset(value.CompiledFunction, a.resetFull)
	a.arena.Reset(value.ValuePtr, a.resetFull)
}

/* Low-level helpers */

func (a *Arena) NewBytes(capacity int, sized bool) []byte {
	if sized {
		return make([]byte, capacity)
	}
	return make([]byte, 0, capacity)
}

func (a *Arena) NewRunes(capacity int, sized bool) []rune {
	if sized {
		return make([]rune, capacity)
	}
	return make([]rune, 0, capacity)
}

func (a *Arena) NewArray(capacity int, sized bool) []Value {
	if sized {
		return make([]Value, capacity)
	}
	return make([]Value, 0, capacity)
}

func (a *Arena) NewDict(capacity int) map[string]Value {
	return make(map[string]Value, capacity)
}

/* Common ref-counting helpers */

// Pin value if it is not static and is allocated (arena or user type).
func (a *Arena) PinAny(v Value) {
	if !v.Static && v.Type >= value.FirstArenaType {
		a.PinAllocated(v)
	}
}

// Pin allocated or user value.
func (a *Arena) PinAllocated(v Value) {
	if v.Type <= value.LastArenaType {
		a.arena.Pin(v.Type, v.Data)
		return
	}
	a.payload.Pin(v)
}

// Retain value if it is not static and is allocated (arena or user type).
func (a *Arena) RetainAny(v Value) {
	if !v.Static && v.Type >= value.FirstArenaType {
		a.RetainAllocated(v)
	}
}

// Retain allocated or user value.
func (a *Arena) RetainAllocated(v Value) {
	if v.Type <= value.LastArenaType {
		a.arena.Retain(v.Type, v.Data)
		return
	}
	a.payload.Retain(v)
}

// Release value if it is not static and is allocated (arena or user type).
func (a *Arena) ReleaseAny(v Value) {
	if !v.Static && v.Type >= value.FirstArenaType {
		a.ReleaseAllocated(v)
	}
}

// Release allocated or user value.
func (a *Arena) ReleaseAllocated(v Value) {
	if v.Type <= value.LastArenaType {
		a.arena.Release(v.Type, v.Data)
		return
	}
	a.payload.Release(v)
}

/* FormatSpec (can be only static) */

func (a *Arena) ResolveFormatSpecValue(v Value) *FormatSpec {
	return &a.static.FormatSpecs[v.Data]
}

/* Decimal (can be static and dynamic) */

func (a *Arena) MustNewDecimalValue(d dec128.Dec128) Value {
	v, err := a.NewDecimalValue(d)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewDecimalValue(d dec128.Dec128) (Value, error) {
	if ref, p, ok := a.arena.New(value.Decimal); ok {
		*(*dec128.Dec128)(p) = d
		return Value{Type: value.Decimal, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(decimalTypeName)
}

func (a *Arena) ResolveDecimalValue(v Value) *dec128.Dec128 {
	if v.Static {
		return &a.static.Decimals[v.Data]
	}
	return (*dec128.Dec128)(a.arena.Resolve(value.Decimal, v.Data))
}

/* String (can be static and dynamic) */

func (a *Arena) MustNewStringValue(s string) Value {
	v, err := a.NewStringValue(s)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewStringValue(s string) (Value, error) {
	if ref, p, ok := a.arena.New(value.String); ok {
		*(*string)(p) = s
		return Value{Type: value.String, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(stringTypeName)
}

func (a *Arena) ResolveStringValue(v Value) *string {
	if v.Static {
		return &a.static.Strings[v.Data]
	}
	return (*string)(a.arena.Resolve(value.String, v.Data))
}

/* Time (can be only dynamic) */

func (a *Arena) MustNewTimeValue(t time.Time) Value {
	v, err := a.NewTimeValue(t)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewTimeValue(t time.Time) (Value, error) {
	if ref, p, ok := a.arena.New(value.Time); ok {
		*(*time.Time)(p) = t
		return Value{Type: value.Time, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(timeTypeName)
}

func (a *Arena) ResolveTimeValue(v Value) *time.Time {
	return (*time.Time)(a.arena.Resolve(value.Time, v.Data))
}

/* IntRange (can be only dynamic) */

func (a *Arena) MustNewIntRangeValue(start, stop, step int64) Value {
	v, err := a.NewIntRangeValue(start, stop, step)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewIntRangeValue(start, stop, step int64) (Value, error) {
	if ref, p, ok := a.arena.New(value.IntRange); ok {
		(*IntRange)(p).Set(start, stop, step)
		return Value{Type: value.IntRange, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(intRangeTypeName)
}

func (a *Arena) ResolveIntRangeValue(v Value) *IntRange {
	return (*IntRange)(a.arena.Resolve(value.IntRange, v.Data))
}

/* ArrayIterator (can be only dynamic) */

func (a *Arena) MustNewArrayIteratorValue(arr []Value) Value {
	v, err := a.NewArrayIteratorValue(arr)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewArrayIteratorValue(arr []Value) (Value, error) {
	if ref, p, ok := a.arena.New(value.ArrayIterator); ok {
		(*ArrayIterator)(p).Set(arr)
		return Value{Type: value.ArrayIterator, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(arrayIteratorTypeName)
}

func (a *Arena) ResolveArrayIteratorValue(v Value) *ArrayIterator {
	return (*ArrayIterator)(a.arena.Resolve(value.ArrayIterator, v.Data))
}

/* BytesIterator (can be only dynamic) */

func (a *Arena) MustNewBytesIteratorValue(b []byte) Value {
	v, err := a.NewBytesIteratorValue(b)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewBytesIteratorValue(b []byte) (Value, error) {
	if ref, p, ok := a.arena.New(value.BytesIterator); ok {
		(*BytesIterator)(p).Set(b)
		return Value{Type: value.BytesIterator, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(bytesIteratorTypeName)
}

func (a *Arena) ResolveBytesIteratorValue(v Value) *BytesIterator {
	return (*BytesIterator)(a.arena.Resolve(value.BytesIterator, v.Data))
}

/* DictIterator (can be only dynamic) */

func (a *Arena) MustNewDictIteratorValue(m map[string]Value) Value {
	v, err := a.NewDictIteratorValue(m)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewDictIteratorValue(m map[string]Value) (Value, error) {
	if ref, p, ok := a.arena.New(value.DictIterator); ok {
		(*DictIterator)(p).Set(m)
		return Value{Type: value.DictIterator, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(dictIteratorTypeName)
}

func (a *Arena) ResolveDictIteratorValue(v Value) *DictIterator {
	return (*DictIterator)(a.arena.Resolve(value.DictIterator, v.Data))
}

/* IntRangeIterator (can be only dynamic) */

func (a *Arena) MustNewIntRangeIteratorValue(start, stop, step int64) Value {
	v, err := a.NewIntRangeIteratorValue(start, stop, step)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewIntRangeIteratorValue(start, stop, step int64) (Value, error) {
	if ref, p, ok := a.arena.New(value.IntRangeIterator); ok {
		(*IntRangeIterator)(p).Set(start, stop, step)
		return Value{Type: value.IntRangeIterator, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(intRangeIteratorTypeName)
}

func (a *Arena) ResolveIntRangeIteratorValue(v Value) *IntRangeIterator {
	return (*IntRangeIterator)(a.arena.Resolve(value.IntRangeIterator, v.Data))
}

/* RunesIterator (can be only dynamic) */

func (a *Arena) MustNewRunesIteratorValue(s []rune) Value {
	v, err := a.NewRunesIteratorValue(s)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewRunesIteratorValue(s []rune) (Value, error) {
	if ref, p, ok := a.arena.New(value.RunesIterator); ok {
		(*RunesIterator)(p).Set(s)
		return Value{Type: value.RunesIterator, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(runesIteratorTypeName)
}

func (a *Arena) ResolveRunesIteratorValue(v Value) *RunesIterator {
	return (*RunesIterator)(a.arena.Resolve(value.RunesIterator, v.Data))
}

/* Array (can be only dynamic) */

func (a *Arena) MustNewArrayValue(arr []Value, immutable bool) Value {
	v, err := a.NewArrayValue(arr, immutable)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewArrayValue(arr []Value, immutable bool) (Value, error) {
	if ref, p, ok := a.arena.New(value.Array); ok {
		(*Array)(p).Set(arr)
		return Value{Type: value.Array, Immutable: immutable, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(arrayTypeName)
}

func (a *Arena) ResolveArrayValue(v Value) *Array {
	return (*Array)(a.arena.Resolve(value.Array, v.Data))
}

/* Bytes (can be only dynamic) */

func (a *Arena) MustNewBytesValue(b []byte, immutable bool) Value {
	v, err := a.NewBytesValue(b, immutable)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewBytesValue(b []byte, immutable bool) (Value, error) {
	if ref, p, ok := a.arena.New(value.Bytes); ok {
		(*Bytes)(p).Set(b)
		return Value{Type: value.Bytes, Immutable: immutable, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(bytesTypeName)
}

func (a *Arena) ResolveBytesValue(v Value) *Bytes {
	return (*Bytes)(a.arena.Resolve(value.Bytes, v.Data))
}

/* Runes (can be static and dynamic) */

func (a *Arena) MustNewRunesValue(r []rune, immutable bool) Value {
	v, err := a.NewRunesValue(r, immutable)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewRunesValue(r []rune, immutable bool) (Value, error) {
	if ref, p, ok := a.arena.New(value.Runes); ok {
		(*Runes)(p).Set(r)
		return Value{Type: value.Runes, Immutable: immutable, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(runesTypeName)
}

func (a *Arena) ResolveRunesValue(v Value) *Runes {
	if v.Static {
		return &a.static.Runes[v.Data]
	}
	return (*Runes)(a.arena.Resolve(value.Runes, v.Data))
}

/* Dict (can be only dynamic) */

func (a *Arena) MustNewDictValue(m map[string]Value, immutable bool) Value {
	v, err := a.NewDictValue(m, immutable)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewDictValue(m map[string]Value, immutable bool) (Value, error) {
	if ref, p, ok := a.arena.New(value.Dict); ok {
		(*Dict)(p).Set(m)
		return Value{Type: value.Dict, Immutable: immutable, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(dictTypeName)
}

func (a *Arena) ResolveDictValue(v Value) *Dict {
	return (*Dict)(a.arena.Resolve(value.Dict, v.Data))
}

/* Record (can be only dynamic), based on dict pool */

func (a *Arena) MustNewRecordValue(m map[string]Value, immutable bool) Value {
	v, err := a.NewRecordValue(m, immutable)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewRecordValue(m map[string]Value, immutable bool) (Value, error) {
	if ref, p, ok := a.arena.New(value.Record); ok {
		(*Record)(p).Set(m)
		return Value{Type: value.Record, Immutable: immutable, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(recordTypeName)
}

func (a *Arena) ResolveRecordValue(v Value) *Record {
	return (*Record)(a.arena.Resolve(value.Record, v.Data))
}

/* Error (can be only dynamic) */

func (a *Arena) MustNewErrorValue(payload Value, kind string, fatal bool) Value {
	v, err := a.NewErrorValue(payload, kind, fatal)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewErrorValue(payload Value, kind string, fatal bool) (Value, error) {
	if ref, p, ok := a.arena.New(value.Error); ok {
		a.PinAny(payload) // mark payload as unmanaged because it's now also owned by the error value
		(*Error)(p).Set(payload, kind, fatal)
		return Value{Type: value.Error, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(errorTypeName)
}

func (a *Arena) NewRuntimeErrorValue(kind string, fatal bool, message string) (Value, error) {
	payload, err := a.NewStringValue(message)
	if err != nil {
		return Undefined, err
	}
	return a.NewErrorValue(payload, kind, fatal)
}

func (a *Arena) ResolveErrorValue(v Value) *Error {
	return (*Error)(a.arena.Resolve(value.Error, v.Data))
}

/* BuiltinClosure (can be only dynamic) */

func (a *Arena) MustNewBuiltinClosureValue(name string, fn NativeFunc, arity int8, variadic bool) Value {
	v, err := a.NewBuiltinClosureValue(name, fn, arity, variadic)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewBuiltinClosureValue(name string, fn NativeFunc, arity int8, variadic bool) (Value, error) {
	if ref, p, ok := a.arena.New(value.BuiltinClosure); ok {
		(*BuiltinClosure)(p).Set(fn, name, arity, variadic)
		return Value{Type: value.BuiltinClosure, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError("builtin-closure")
}

func (a *Arena) ResolveBuiltinClosureValue(v Value) *BuiltinClosure {
	return (*BuiltinClosure)(a.arena.Resolve(value.BuiltinClosure, v.Data))
}

/* CompiledFunction (can be static and dynamic) */

func (a *Arena) MustNewCompiledFunctionValue(
	instructions []byte,
	free []*Value,
	sourceMap map[int]Pos,
	numLocals int,
	maxStack int,
	numParameters int8,
	varArgs bool,
	namedResult int8,
) Value {
	v, err := a.NewCompiledFunctionValue(instructions, free, sourceMap, numLocals, maxStack, numParameters, varArgs, namedResult)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewCompiledFunctionValue(
	instructions []byte,
	free []*Value,
	sourceMap map[int]Pos,
	numLocals int,
	maxStack int,
	numParameters int8,
	varArgs bool,
	namedResult int8,
) (Value, error) {
	if ref, p, ok := a.arena.New(value.CompiledFunction); ok {
		(*CompiledFunction)(p).Set(instructions, free, sourceMap, numLocals, maxStack, numParameters, varArgs, namedResult)
		return Value{Type: value.CompiledFunction, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError("compiled-function")
}

func (a *Arena) ResolveCompiledFunctionValue(v Value) *CompiledFunction {
	if v.Static {
		return &a.static.CompiledFunctions[v.Data]
	}
	return (*CompiledFunction)(a.arena.Resolve(value.CompiledFunction, v.Data))
}

/* ValuePtr (can be only dynamic) */

func (a *Arena) MustNewValuePtrValue(p *Value) Value {
	v, err := a.NewValuePtrValue(p)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewValuePtrValue(p *Value) (Value, error) {
	if ref, poolPtr, ok := a.arena.New(value.ValuePtr); ok {
		a.PinAny(*p) // mark pointed value as unmanaged because it's now also owned by the pointer value
		*(**Value)(poolPtr) = p
		return Value{Type: value.ValuePtr, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(valuePtrTypeName)
}

func (a *Arena) ResolveValuePtrValue(v Value) **Value {
	return (**Value)(a.arena.Resolve(value.ValuePtr, v.Data))
}
