package core

import (
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/slab"
)

type ArenaOptions struct {
	Decimals int
	Times    int

	BytesNum int
	BytesCap int

	RunesNum int
	RunesCap int

	ArraysNum int
	ArraysCap int

	BuiltinFunctions  int
	CompiledFunctions int

	ErrorValues    int
	StringValues   int
	RunesValues    int
	BytesValues    int
	ArrayValues    int
	MapValues      int
	IntRangeValues int

	RunesIterators    int
	BytesIterators    int
	ArrayIterators    int
	MapIterators      int
	IntRangeIterators int
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

		BuiltinFunctions:  1024,
		CompiledFunctions: 1024,

		ErrorValues:    128,
		StringValues:   1024,
		RunesValues:    1024,
		BytesValues:    1024,
		ArrayValues:    1024,
		MapValues:      1024,
		IntRangeValues: 128,

		RunesIterators:    1024,
		BytesIterators:    1024,
		ArrayIterators:    1024,
		MapIterators:      1024,
		IntRangeIterators: 1024,
	}
}

type Arena struct {
	decimals slab.Slab[dec128.Dec128]
	times    slab.Slab[time.Time]
	bytes    slab.SliceSlab[byte]
	runes    slab.SliceSlab[rune]
	arrays   slab.SliceSlab[Value]

	builtinFunctions  slab.Slab[BuiltinFunction]
	compiledFunctions slab.Slab[CompiledFunction]

	errorValues    slab.Slab[Error]
	stringValues   slab.Slab[String]
	runesValues    slab.Slab[Runes]
	bytesValues    slab.Slab[Bytes]
	arrayValues    slab.Slab[Array]
	mapValues      slab.Slab[Map]
	intRangeValues slab.Slab[IntRange]

	runesIterators    slab.Slab[RunesIterator]
	bytesIterators    slab.Slab[BytesIterator]
	arrayIterators    slab.Slab[ArrayIterator]
	mapIterators      slab.Slab[MapIterator]
	intRangeIterators slab.Slab[IntRangeIterator]
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

		builtinFunctions:  slab.NewSlab[BuiltinFunction](opts.BuiltinFunctions, nil),
		compiledFunctions: slab.NewSlab(opts.CompiledFunctions, clearCompiledFunction),

		errorValues:    slab.NewSlab[Error](opts.ErrorValues, nil),
		stringValues:   slab.NewSlab(opts.StringValues, clearStringValue),
		runesValues:    slab.NewSlab(opts.RunesValues, clearRunesValue),
		bytesValues:    slab.NewSlab(opts.BytesValues, clearBytesValue),
		arrayValues:    slab.NewSlab(opts.ArrayValues, clearArrayValue),
		mapValues:      slab.NewSlab(opts.MapValues, clearMapValue),
		intRangeValues: slab.NewSlab[IntRange](opts.IntRangeValues, nil),

		runesIterators:    slab.NewSlab(opts.RunesIterators, clearRunesIterator),
		bytesIterators:    slab.NewSlab(opts.BytesIterators, clearBytesIterator),
		arrayIterators:    slab.NewSlab(opts.ArrayIterators, clearArrayIterator),
		mapIterators:      slab.NewSlab(opts.MapIterators, clearMapIterator),
		intRangeIterators: slab.NewSlab[IntRangeIterator](opts.IntRangeIterators, nil),
	}
}

func clearCompiledFunction(f *CompiledFunction) {
	f.Instructions = nil
	f.Free = nil
	f.SourceMap = nil
}

func clearStringValue(s *String) {
	s.Value = ""
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

func clearMapValue(m *Map) {
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

func clearMapIterator(i *MapIterator) {
	i.Elements = nil
	i.Keys = nil
}

func (a *Arena) Stat() map[string]slab.Stats {
	return map[string]slab.Stats{
		"Decimal": a.decimals.Stats(),
		"Time":    a.times.Stats(),
		"Bytes":   a.bytes.Stats(),
		"Runes":   a.runes.Stats(),
		"Array":   a.arrays.Stats(),

		"BuiltinFunction":  a.builtinFunctions.Stats(),
		"CompiledFunction": a.compiledFunctions.Stats(),

		"ErrorValue":    a.errorValues.Stats(),
		"StringValue":   a.stringValues.Stats(),
		"RunesValue":    a.runesValues.Stats(),
		"BytesValue":    a.bytesValues.Stats(),
		"ArrayValue":    a.arrayValues.Stats(),
		"MapValue":      a.mapValues.Stats(),
		"IntRangeValue": a.intRangeValues.Stats(),

		"RunesIterator":    a.runesIterators.Stats(),
		"BytesIterator":    a.bytesIterators.Stats(),
		"ArrayIterator":    a.arrayIterators.Stats(),
		"MapIterator":      a.mapIterators.Stats(),
		"IntRangeIterator": a.intRangeIterators.Stats(),
	}
}

func (a *Arena) Reset() {
	a.decimals.Reset()
	a.times.Reset()
	a.bytes.Reset()
	a.runes.Reset()
	a.arrays.Reset()

	a.builtinFunctions.Reset()
	a.compiledFunctions.Reset()

	a.errorValues.Reset()
	a.stringValues.Reset()
	a.runesValues.Reset()
	a.bytesValues.Reset()
	a.arrayValues.Reset()
	a.mapValues.Reset()
	a.intRangeValues.Reset()

	a.runesIterators.Reset()
	a.bytesIterators.Reset()
	a.arrayIterators.Reset()
	a.mapIterators.Reset()
	a.intRangeIterators.Reset()
}

/* Low-level resources */

func (a *Arena) NewDecimal() *dec128.Dec128 {
	return a.decimals.Alloc()
}

func (a *Arena) NewTime() *time.Time {
	return a.times.Alloc()
}

func (a *Arena) NewBytes(capacity int, sized bool) []byte {
	return a.bytes.Alloc(capacity, sized)
}

func (a *Arena) NewRunes(capacity int, sized bool) []rune {
	return a.runes.Alloc(capacity, sized)
}

func (a *Arena) NewArray(capacity int, sized bool) []Value {
	return a.arrays.Alloc(capacity, sized)
}

func (a *Arena) NewMap(capacity int) map[string]Value {
	return make(map[string]Value, capacity)
}

/* Value envelopes */

func (a *Arena) NewBuiltinFunctionValue(name string, fn NativeFunc, arity int8, variadic bool) Value {
	o := a.builtinFunctions.Alloc()
	o.Set(fn, name, arity, variadic)
	return BuiltinFunctionValue(o)
}

func (a *Arena) NewCompiledFunctionValue(instructions []byte, free []*Value, sourceMap map[int]Pos, numLocals int, numParameters int8, varArgs bool) Value {
	o := a.compiledFunctions.Alloc()
	o.Set(instructions, free, sourceMap, numLocals, numParameters, varArgs)
	return CompiledFunctionValue(o)
}

func (a *Arena) NewErrorValue(e Value) Value {
	o := a.errorValues.Alloc()
	o.Set(e)
	return ErrorValue(o)
}

func (a *Arena) NewStringValue(s string) Value {
	o := a.stringValues.Alloc()
	o.Set(s)
	return StringValue(o)
}

func (a *Arena) NewRunesValue(r []rune) Value {
	o := a.runesValues.Alloc()
	o.Set(r)
	return RunesValue(o)
}

func (a *Arena) NewBytesValue(b []byte) Value {
	o := a.bytesValues.Alloc()
	o.Set(b)
	return BytesValue(o)
}

func (a *Arena) NewArrayValue(arr []Value, immutable bool) Value {
	o := a.arrayValues.Alloc()
	o.Set(arr)
	return ArrayValue(o, immutable)
}

func (a *Arena) NewMapValue(m map[string]Value, immutable bool) Value {
	o := a.mapValues.Alloc()
	o.Set(m)
	return MapValue(o, immutable)
}

func (a *Arena) NewRecordValue(m map[string]Value, immutable bool) Value {
	o := a.mapValues.Alloc()
	o.Set(m)
	return RecordValue(o, immutable)
}

func (a *Arena) NewIntRangeValue(start, stop, step int64) Value {
	o := a.intRangeValues.Alloc()
	o.Set(start, stop, step)
	return IntRangeValue(o)
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

func (a *Arena) NewMapIteratorValue(m map[string]Value) Value {
	o := a.mapIterators.Alloc()
	o.Set(m)
	return MapIteratorValue(o)
}

func (a *Arena) NewIntRangeIteratorValue(start, stop, step int64) Value {
	o := a.intRangeIterators.Alloc()
	o.Set(start, stop, step)
	return IntRangeIteratorValue(o)
}
