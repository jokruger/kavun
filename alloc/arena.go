package alloc

import (
	"time"

	"github.com/jokruger/kavun/core"
)

type Arena struct {
	decimals slab[core.Decimal]
	times    slab[time.Time]
	bytes    sliceSlab[byte]
	runes    sliceSlab[rune]
	arrays   sliceSlab[core.Value]

	builtinFunctions  slab[core.BuiltinFunction]
	compiledFunctions slab[core.CompiledFunction]

	errorValues    slab[core.Error]
	stringValues   slab[core.String]
	runesValues    slab[core.Runes]
	bytesValues    slab[core.Bytes]
	arrayValues    slab[core.Array]
	mapValues      slab[core.Map]
	intRangeValues slab[core.IntRange]

	runesIterators    slab[core.RunesIterator]
	bytesIterators    slab[core.BytesIterator]
	arrayIterators    slab[core.ArrayIterator]
	mapIterators      slab[core.MapIterator]
	intRangeIterators slab[core.IntRangeIterator]
}

func NewArena(opts ...ArenaOption) *Arena {
	o := &ArenaOptions{
		decimals: 1024,
		times:    1024,

		bytesNum: 1024,
		bytesCap: 64,

		runesNum: 1024,
		runesCap: 64,

		arraysNum: 1024,
		arraysCap: 64,

		builtinFunctions:  1024,
		compiledFunctions: 1024,

		errorValues:    128,
		stringValues:   1024,
		runesValues:    1024,
		bytesValues:    1024,
		arrayValues:    1024,
		mapValues:      1024,
		intRangeValues: 128,

		runesIterators:    1024,
		bytesIterators:    1024,
		arrayIterators:    1024,
		mapIterators:      1024,
		intRangeIterators: 1024,
	}

	for _, opt := range opts {
		opt(o)
	}

	return &Arena{
		decimals: newSlab[core.Decimal](o.decimals, nil),
		times:    newSlab[time.Time](o.times, nil),
		bytes:    newSliceSlab[byte](o.bytesNum, o.bytesCap),
		runes:    newSliceSlab[rune](o.runesNum, o.runesCap),
		arrays:   newSliceSlab[core.Value](o.arraysNum, o.arraysCap),

		builtinFunctions:  newSlab[core.BuiltinFunction](o.builtinFunctions, nil),
		compiledFunctions: newSlab[core.CompiledFunction](o.compiledFunctions, clearCompiledFunction),

		errorValues:    newSlab[core.Error](o.errorValues, nil),
		stringValues:   newSlab[core.String](o.stringValues, clearStringValue),
		runesValues:    newSlab[core.Runes](o.runesValues, clearRunesValue),
		bytesValues:    newSlab[core.Bytes](o.bytesValues, clearBytesValue),
		arrayValues:    newSlab[core.Array](o.arrayValues, clearArrayValue),
		mapValues:      newSlab[core.Map](o.mapValues, clearMapValue),
		intRangeValues: newSlab[core.IntRange](o.intRangeValues, nil),

		runesIterators:    newSlab[core.RunesIterator](o.runesIterators, clearRunesIterator),
		bytesIterators:    newSlab[core.BytesIterator](o.bytesIterators, clearBytesIterator),
		arrayIterators:    newSlab[core.ArrayIterator](o.arrayIterators, clearArrayIterator),
		mapIterators:      newSlab[core.MapIterator](o.mapIterators, clearMapIterator),
		intRangeIterators: newSlab[core.IntRangeIterator](o.intRangeIterators, nil),
	}
}

func clearCompiledFunction(f *core.CompiledFunction) {
	f.Instructions = nil
	f.Free = nil
	f.SourceMap = nil
}

func clearStringValue(s *core.String) {
	s.Value = ""
}

func clearRunesValue(r *core.Runes) {
	r.Elements = nil
}

func clearBytesValue(b *core.Bytes) {
	b.Elements = nil
}

func clearArrayValue(a *core.Array) {
	a.Elements = nil
}

func clearMapValue(m *core.Map) {
	m.Elements = nil
}

func clearRunesIterator(i *core.RunesIterator) {
	i.Elements = nil
}

func clearBytesIterator(i *core.BytesIterator) {
	i.Elements = nil
}

func clearArrayIterator(i *core.ArrayIterator) {
	i.Elements = nil
}

func clearMapIterator(i *core.MapIterator) {
	i.Elements = nil
	i.Keys = nil
}

func (a *Arena) Stat() map[string]TypeStat {
	return map[string]TypeStat{
		"Decimal": {Pool: a.decimals.used, Heap: a.decimals.fallback},
		"Time":    {Pool: a.times.used, Heap: a.times.fallback},
		"Bytes":   {Pool: a.bytes.used, Heap: a.bytes.fallback},
		"Runes":   {Pool: a.runes.used, Heap: a.runes.fallback},
		"Array":   {Pool: a.arrays.used, Heap: a.arrays.fallback},

		"BuiltinFunction":  {Pool: a.builtinFunctions.used, Heap: a.builtinFunctions.fallback},
		"CompiledFunction": {Pool: a.compiledFunctions.used, Heap: a.compiledFunctions.fallback},

		"ErrorValue":    {Pool: a.errorValues.used, Heap: a.errorValues.fallback},
		"StringValue":   {Pool: a.stringValues.used, Heap: a.stringValues.fallback},
		"RunesValue":    {Pool: a.runesValues.used, Heap: a.runesValues.fallback},
		"BytesValue":    {Pool: a.bytesValues.used, Heap: a.bytesValues.fallback},
		"ArrayValue":    {Pool: a.arrayValues.used, Heap: a.arrayValues.fallback},
		"MapValue":      {Pool: a.mapValues.used, Heap: a.mapValues.fallback},
		"IntRangeValue": {Pool: a.intRangeValues.used, Heap: a.intRangeValues.fallback},

		"RunesIterator":    {Pool: a.runesIterators.used, Heap: a.runesIterators.fallback},
		"BytesIterator":    {Pool: a.bytesIterators.used, Heap: a.bytesIterators.fallback},
		"ArrayIterator":    {Pool: a.arrayIterators.used, Heap: a.arrayIterators.fallback},
		"MapIterator":      {Pool: a.mapIterators.used, Heap: a.mapIterators.fallback},
		"IntRangeIterator": {Pool: a.intRangeIterators.used, Heap: a.intRangeIterators.fallback},
	}
}

func (a *Arena) Reset() {
	a.decimals.reset()
	a.times.reset()
	a.bytes.reset()
	a.runes.reset()
	a.arrays.reset()

	a.builtinFunctions.reset()
	a.compiledFunctions.reset()

	a.errorValues.reset()
	a.stringValues.reset()
	a.runesValues.reset()
	a.bytesValues.reset()
	a.arrayValues.reset()
	a.mapValues.reset()
	a.intRangeValues.reset()

	a.runesIterators.reset()
	a.bytesIterators.reset()
	a.arrayIterators.reset()
	a.mapIterators.reset()
	a.intRangeIterators.reset()
}

/* Low-level resources */

func (a *Arena) NewDecimal() (*core.Decimal, error) {
	return a.decimals.alloc(), nil
}

func (a *Arena) NewTime() (*time.Time, error) {
	return a.times.alloc(), nil
}

func (a *Arena) NewBytes(capacity int, resize bool) ([]byte, error) {
	return a.bytes.alloc(capacity, resize), nil
}

func (a *Arena) NewRunes(capacity int, resize bool) ([]rune, error) {
	return a.runes.alloc(capacity, resize), nil
}

func (a *Arena) NewArray(capacity int, resize bool) ([]core.Value, error) {
	return a.arrays.alloc(capacity, resize), nil
}

func (a *Arena) NewMap(capacity int) (map[string]core.Value, error) {
	o := make(map[string]core.Value, capacity)
	return o, nil
}

/* Value envelopes */

func (a *Arena) NewBuiltinFunctionValue(name string, fn core.NativeFunc, arity int8, variadic bool) (core.Value, error) {
	o := a.builtinFunctions.alloc()
	o.Set(fn, name, arity, variadic)
	return core.BuiltinFunctionValue(o), nil
}

func (a *Arena) NewCompiledFunctionValue(instructions []byte, free []*core.Value, sourceMap map[int]core.Pos, numLocals int, numParameters int8, varArgs bool) (core.Value, error) {
	o := a.compiledFunctions.alloc()
	o.Set(instructions, free, sourceMap, numLocals, numParameters, varArgs)
	return core.CompiledFunctionValue(o), nil
}

func (a *Arena) NewErrorValue(e core.Value) (core.Value, error) {
	o := a.errorValues.alloc()
	o.Set(e)
	return core.ErrorValue(o), nil
}

func (a *Arena) NewStringValue(s string) (core.Value, error) {
	o := a.stringValues.alloc()
	o.Set(s)
	return core.StringValue(o), nil
}

func (a *Arena) NewRunesValue(r []rune) (core.Value, error) {
	o := a.runesValues.alloc()
	o.Set(r)
	return core.RunesValue(o), nil
}

func (a *Arena) NewBytesValue(b []byte) (core.Value, error) {
	o := a.bytesValues.alloc()
	o.Set(b)
	return core.BytesValue(o), nil
}

func (a *Arena) NewArrayValue(arr []core.Value, immutable bool) (core.Value, error) {
	o := a.arrayValues.alloc()
	o.Set(arr)
	return core.ArrayValue(o, immutable), nil
}

func (a *Arena) NewMapValue(m map[string]core.Value, immutable bool) (core.Value, error) {
	o := a.mapValues.alloc()
	o.Set(m)
	return core.MapValue(o, immutable), nil
}

func (a *Arena) NewRecordValue(m map[string]core.Value, immutable bool) (core.Value, error) {
	o := a.mapValues.alloc()
	o.Set(m)
	return core.RecordValue(o, immutable), nil
}

func (a *Arena) NewIntRangeValue(start, stop, step int64) (core.Value, error) {
	o := a.intRangeValues.alloc()
	o.Set(start, stop, step)
	return core.IntRangeValue(o), nil
}

func (a *Arena) NewRunesIteratorValue(s []rune) (core.Value, error) {
	o := a.runesIterators.alloc()
	o.Set(s)
	return core.RunesIteratorValue(o), nil
}

func (a *Arena) NewBytesIteratorValue(b []byte) (core.Value, error) {
	o := a.bytesIterators.alloc()
	o.Set(b)
	return core.BytesIteratorValue(o), nil
}

func (a *Arena) NewArrayIteratorValue(arr []core.Value) (core.Value, error) {
	o := a.arrayIterators.alloc()
	o.Set(arr)
	return core.ArrayIteratorValue(o), nil
}

func (a *Arena) NewMapIteratorValue(m map[string]core.Value) (core.Value, error) {
	o := a.mapIterators.alloc()
	o.Set(m)
	return core.MapIteratorValue(o), nil
}

func (a *Arena) NewIntRangeIteratorValue(start, stop, step int64) (core.Value, error) {
	o := a.intRangeIterators.alloc()
	o.Set(start, stop, step)
	return core.IntRangeIteratorValue(o), nil
}
