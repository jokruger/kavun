package core

import (
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/slab"
)

type Resettable interface {
	Reset()
}

type ArenaOptions struct {
	Decimals int
	Times    int

	BytesNum int
	BytesCap int

	RunesNum int
	RunesCap int

	ArraysNum int
	ArraysCap int

	BuiltinClosures   int
	CompiledFunctions int

	StringValues   int
	RunesValues    int
	BytesValues    int
	ArrayValues    int
	DictValues     int
	IntRangeValues int

	RunesIterators    int
	BytesIterators    int
	ArrayIterators    int
	DictIterators     int
	IntRangeIterators int

	Payload Resettable
}

func DefaultArenaOptions() *ArenaOptions {
	return &ArenaOptions{
		Decimals: 1024,
		Times:    1024,

		BytesNum: 1024,
		BytesCap: 64,

		RunesNum: 1024,
		RunesCap: 64,

		ArraysNum: 1024,
		ArraysCap: 64,

		BuiltinClosures:   1024,
		CompiledFunctions: 1024,

		StringValues:   1024,
		RunesValues:    1024,
		BytesValues:    1024,
		ArrayValues:    1024,
		DictValues:     1024,
		IntRangeValues: 128,

		RunesIterators:    1024,
		BytesIterators:    1024,
		ArrayIterators:    1024,
		DictIterators:     1024,
		IntRangeIterators: 1024,
	}
}

type Arena struct {
	decimals slab.Slab[dec128.Dec128]
	times    slab.Slab[time.Time]
	bytes    slab.SliceSlab[byte]
	runes    slab.SliceSlab[rune]
	arrays   slab.SliceSlab[Value]

	builtinClosures   slab.Slab[BuiltinClosure]
	compiledFunctions slab.Slab[CompiledFunction]

	stringValues   slab.Slab[string]
	runesValues    slab.Slab[Runes]
	bytesValues    slab.Slab[Bytes]
	arrayValues    slab.Slab[Array]
	dictValues     slab.Slab[Dict]
	intRangeValues slab.Slab[IntRange]

	runesIterators    slab.Slab[RunesIterator]
	bytesIterators    slab.Slab[BytesIterator]
	arrayIterators    slab.Slab[ArrayIterator]
	dictIterators     slab.Slab[DictIterator]
	intRangeIterators slab.Slab[IntRangeIterator]

	payload Resettable
}

// NewArena creates a new Arena with the given options. If opts is nil, it uses the default options.
func NewArena(opts *ArenaOptions) *Arena {
	if opts == nil {
		opts = DefaultArenaOptions()
	}

	return &Arena{
		decimals: slab.NewSlab[dec128.Dec128](opts.Decimals, nil),
		times:    slab.NewSlab[time.Time](opts.Times, nil),
		bytes:    slab.NewSliceSlab[byte](opts.BytesNum, opts.BytesCap),
		runes:    slab.NewSliceSlab[rune](opts.RunesNum, opts.RunesCap),
		arrays:   slab.NewSliceSlab[Value](opts.ArraysNum, opts.ArraysCap),

		builtinClosures:   slab.NewSlab[BuiltinClosure](opts.BuiltinClosures, nil),
		compiledFunctions: slab.NewSlab(opts.CompiledFunctions, clearCompiledFunction),

		stringValues:   slab.NewSlab[string](opts.StringValues, nil),
		runesValues:    slab.NewSlab(opts.RunesValues, clearRunesValue),
		bytesValues:    slab.NewSlab(opts.BytesValues, clearBytesValue),
		arrayValues:    slab.NewSlab(opts.ArrayValues, clearArrayValue),
		dictValues:     slab.NewSlab(opts.DictValues, clearDictValue),
		intRangeValues: slab.NewSlab[IntRange](opts.IntRangeValues, nil),

		runesIterators:    slab.NewSlab(opts.RunesIterators, clearRunesIterator),
		bytesIterators:    slab.NewSlab(opts.BytesIterators, clearBytesIterator),
		arrayIterators:    slab.NewSlab(opts.ArrayIterators, clearArrayIterator),
		dictIterators:     slab.NewSlab(opts.DictIterators, clearDictIterator),
		intRangeIterators: slab.NewSlab[IntRangeIterator](opts.IntRangeIterators, nil),

		payload: opts.Payload,
	}
}

func clearCompiledFunction(f *CompiledFunction) {
	f.Instructions = nil
	f.Free = nil
	f.SourceMap = nil
}

func clearRunesValue(r *Runes) {
	r.Elements = nil
}

func clearBytesValue(b *Bytes) {
	b.Elements = nil
}

func clearArrayValue(a *Array) {
	a.Elements = nil
}

func clearDictValue(m *Dict) {
	m.Elements = nil
}

func clearRunesIterator(i *RunesIterator) {
	i.Elements = nil
}

func clearBytesIterator(i *BytesIterator) {
	i.Elements = nil
}

func clearArrayIterator(i *ArrayIterator) {
	i.Elements = nil
}

func clearDictIterator(i *DictIterator) {
	i.Elements = nil
	i.Keys = nil
}

// Payload returns the payload associated with the arena, which can be used to store any additional data or context used
// by user-defined types and functions.
func (a *Arena) Payload() any {
	return a.payload
}

func (a *Arena) Reset() {
	a.decimals.Reset()
	a.times.Reset()
	a.bytes.Reset()
	a.runes.Reset()
	a.arrays.Reset()

	a.builtinClosures.Reset()
	a.compiledFunctions.Reset()

	a.stringValues.Reset()
	a.runesValues.Reset()
	a.bytesValues.Reset()
	a.arrayValues.Reset()
	a.dictValues.Reset()
	a.intRangeValues.Reset()

	a.runesIterators.Reset()
	a.bytesIterators.Reset()
	a.arrayIterators.Reset()
	a.dictIterators.Reset()
	a.intRangeIterators.Reset()

	if a.payload != nil {
		a.payload.Reset()
	}
}

/* Low-level resources */

func (a *Arena) NewBytes(capacity int, sized bool) []byte {
	return a.bytes.Alloc(capacity, sized)
}

func (a *Arena) NewRunes(capacity int, sized bool) []rune {
	return a.runes.Alloc(capacity, sized)
}

func (a *Arena) NewArray(capacity int, sized bool) []Value {
	return a.arrays.Alloc(capacity, sized)
}

func (a *Arena) NewDict(capacity int) map[string]Value {
	return make(map[string]Value, capacity)
}

/* Boxed Values */

func (a *Arena) NewErrorValue(payload Value, kind string, fatal bool) Value {
	return Value{
		Type:      VT_ERROR,
		Immutable: true,
		Ptr: unsafe.Pointer(&Error{
			Payload: payload,
			Kind:    KindUser,
			Fatal:   fatal,
		}),
	}
}

func (a *Arena) NewRuntimeErrorValue(kind string, fatal bool, message string) Value {
	return Value{
		Type:      VT_ERROR,
		Immutable: true,
		Ptr: unsafe.Pointer(&Error{
			Payload: a.NewStringValue(message),
			Kind:    kind,
			Fatal:   fatal,
		}),
	}
}

func (a *Arena) NewDecimalValue(d dec128.Dec128) Value {
	p := a.decimals.Alloc()
	*p = d
	return Value{
		Type:      VT_DECIMAL,
		Immutable: true,
		Ptr:       unsafe.Pointer(p),
	}
}

func (a *Arena) NewTimeValue(t time.Time) Value {
	p := a.times.Alloc()
	*p = t
	return Value{
		Type:      VT_TIME,
		Immutable: true,
		Ptr:       unsafe.Pointer(p),
	}
}

func (a *Arena) NewBuiltinClosureValue(name string, fn NativeFunc, arity int8, variadic bool) Value {
	o := a.builtinClosures.Alloc()
	o.Set(fn, name, arity, variadic)
	return BuiltinClosureValue(o)
}

func (a *Arena) NewCompiledFunctionValue(instructions []byte, free []*Value, sourceMap map[int]Pos, numLocals, maxStack int, numParameters int8, varArgs bool, namedResult int8) Value {
	o := a.compiledFunctions.Alloc()
	o.Set(instructions, free, sourceMap, numLocals, maxStack, numParameters, varArgs, namedResult)
	return CompiledFunctionValue(o)
}

func (a *Arena) NewStringValue(s string) Value {
	o := a.stringValues.Alloc()
	*o = s
	return Value{
		Type:      VT_STRING,
		Immutable: true,
		Ptr:       unsafe.Pointer(o),
	}
}

func (a *Arena) NewRunesValue(r []rune, immutable bool) Value {
	o := a.runesValues.Alloc()
	o.Set(r)
	return Value{
		Type:      VT_RUNES,
		Immutable: immutable,
		Ptr:       unsafe.Pointer(o),
	}
}

func (a *Arena) NewBytesValue(b []byte, immutable bool) Value {
	o := a.bytesValues.Alloc()
	o.Set(b)
	return Value{
		Type:      VT_BYTES,
		Immutable: immutable,
		Ptr:       unsafe.Pointer(o),
	}
}

func (a *Arena) NewArrayValue(arr []Value, immutable bool) Value {
	o := a.arrayValues.Alloc()
	o.Set(arr)
	return Value{
		Type:      VT_ARRAY,
		Immutable: immutable,
		Ptr:       unsafe.Pointer(o),
	}
}

func (a *Arena) NewDictValue(m map[string]Value, immutable bool) Value {
	o := a.dictValues.Alloc()
	o.Set(m)
	return Value{
		Type:      VT_DICT,
		Immutable: immutable,
		Ptr:       unsafe.Pointer(o),
	}
}

func (a *Arena) NewRecordValue(m map[string]Value, immutable bool) Value {
	o := a.dictValues.Alloc()
	o.Set(m)
	return Value{
		Type:      VT_RECORD,
		Immutable: immutable,
		Ptr:       unsafe.Pointer(o),
	}
}

func (a *Arena) NewIntRangeValue(start, stop, step int64) Value {
	o := a.intRangeValues.Alloc()
	o.Set(start, stop, step)
	return Value{
		Type:      VT_INT_RANGE,
		Immutable: true,
		Ptr:       unsafe.Pointer(o),
	}
}

func (a *Arena) NewRunesIteratorValue(s []rune) Value {
	o := a.runesIterators.Alloc()
	o.Set(s)
	return RunesIteratorValue(o)
}

func (a *Arena) NewBytesIteratorValue(b []byte) Value {
	o := a.bytesIterators.Alloc()
	o.Set(b)
	return BytesIteratorValue(o)
}

func (a *Arena) NewArrayIteratorValue(arr []Value) Value {
	o := a.arrayIterators.Alloc()
	o.Set(arr)
	return ArrayIteratorValue(o)
}

func (a *Arena) NewDictIteratorValue(m map[string]Value) Value {
	o := a.dictIterators.Alloc()
	o.Set(m)
	return DictIteratorValue(o)
}

func (a *Arena) NewIntRangeIteratorValue(start, stop, step int64) Value {
	o := a.intRangeIterators.Alloc()
	o.Set(start, stop, step)
	return IntRangeIteratorValue(o)
}

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

func ArenaNewRunesValue(a *Arena, r []rune, immutable bool) Value {
	return a.NewRunesValue(r, immutable)
}

func ArenaNewBytesValue(a *Arena, b []byte, immutable bool) Value {
	return a.NewBytesValue(b, immutable)
}

func ArenaNewArrayValue(a *Arena, arr []Value, immutable bool) Value {
	return a.NewArrayValue(arr, immutable)
}
