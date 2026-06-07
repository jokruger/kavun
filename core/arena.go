package core

import (
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/refpool"
)

/* Helper functions used in combination with generics */

func ArenaNewBytes(a *Arena, capacity int, sized bool) []byte {
	if sized {
		return make([]byte, capacity)
	}
	return make([]byte, 0, capacity)
}

func ArenaNewRunes(a *Arena, capacity int, sized bool) []rune {
	if sized {
		return make([]rune, capacity)
	}
	return make([]rune, 0, capacity)
}

func ArenaNewArray(a *Arena, capacity int, sized bool) []Value {
	if sized {
		return make([]Value, capacity)
	}
	return make([]Value, 0, capacity)
}

func ArenaNewRunesValue(a *Arena, r []rune, immutable bool) (Value, bool) {
	return a.NewRunesValue(r, immutable)
}

func ArenaNewBytesValue(a *Arena, b []byte, immutable bool) (Value, bool) {
	return a.NewBytesValue(b, immutable)
}

func ArenaNewArrayValue(a *Arena, arr []Value, immutable bool) (Value, bool) {
	return a.NewArrayValue(arr, immutable)
}

type Resettable interface {
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
	Static  Static
	Payload Resettable

	DecimalBuf int
	StringBuf  int
	TimeBuf    int

	IntRangeBuf         int
	ArrayIteratorBuf    int
	BytesIteratorBuf    int
	DictIteratorBuf     int
	IntRangeIteratorBuf int
	RunesIteratorBuf    int

	ArrayBuf int
	BytesBuf int
	RunesBuf int
	DictBuf  int

	ErrorBuf            int
	FormatSpecBuf       int
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

		ArrayBuf: 64,
		BytesBuf: 32,
		RunesBuf: 256,
		DictBuf:  64,

		ErrorBuf:            0,
		FormatSpecBuf:       0,
		BuiltinClosureBuf:   64,
		CompiledFunctionBuf: 64,

		ValuePtrBuf: 256,

		ZeroOnRelease: true,
		ZeroOnReset:   true,
		ResetFull:     true,
	}
}

type Arena struct {
	resetFull bool

	static  Static
	payload Resettable

	decPool  *refpool.Pool[dec128.Dec128]
	strPool  *refpool.Pool[string]
	timePool *refpool.Pool[time.Time]

	intRangePool         *refpool.Pool[IntRange]
	arrayIteratorPool    *refpool.Pool[ArrayIterator]
	bytesIteratorPool    *refpool.Pool[BytesIterator]
	dictIteratorPool     *refpool.Pool[DictIterator]
	intRangeIteratorPool *refpool.Pool[IntRangeIterator]
	runesIteratorPool    *refpool.Pool[RunesIterator]

	arrayPool *refpool.Pool[Array]
	bytesPool *refpool.Pool[Bytes]
	runesPool *refpool.Pool[Runes]
	dictPool  *refpool.Pool[Dict]

	errorPool      *refpool.Pool[Error]
	formatSpecPool *refpool.Pool[FormatSpec]
	biPool         *refpool.Pool[BuiltinClosure]
	cfPool         *refpool.Pool[CompiledFunction]

	ptrPool *refpool.Pool[*Value]
}

// NewArena creates a new Arena with the given options. If opts is nil, it uses the default options.
func NewArena(opts *ArenaOptions) *Arena {
	if opts == nil {
		opts = DefaultArenaOptions()
	}

	poolOpts := &refpool.Options{
		ZeroOnRelease: opts.ZeroOnRelease,
		ZeroOnReset:   opts.ZeroOnReset,
	}

	return &Arena{
		resetFull: opts.ResetFull,

		static:  opts.Static,
		payload: opts.Payload,

		decPool:  refpool.New[dec128.Dec128](opts.DecimalBuf, poolOpts),
		strPool:  refpool.New[string](opts.StringBuf, poolOpts),
		timePool: refpool.New[time.Time](opts.TimeBuf, poolOpts),

		intRangePool:         refpool.New[IntRange](opts.IntRangeBuf, poolOpts),
		arrayIteratorPool:    refpool.New[ArrayIterator](opts.ArrayIteratorBuf, poolOpts),
		bytesIteratorPool:    refpool.New[BytesIterator](opts.BytesIteratorBuf, poolOpts),
		dictIteratorPool:     refpool.New[DictIterator](opts.DictIteratorBuf, poolOpts),
		intRangeIteratorPool: refpool.New[IntRangeIterator](opts.IntRangeIteratorBuf, poolOpts),
		runesIteratorPool:    refpool.New[RunesIterator](opts.RunesIteratorBuf, poolOpts),

		arrayPool: refpool.New[Array](opts.ArrayBuf, poolOpts),
		bytesPool: refpool.New[Bytes](opts.BytesBuf, poolOpts),
		runesPool: refpool.New[Runes](opts.RunesBuf, poolOpts),
		dictPool:  refpool.New[Dict](opts.DictBuf, poolOpts),

		errorPool: refpool.New[Error](opts.ErrorBuf, poolOpts),
		biPool:    refpool.New[BuiltinClosure](opts.BuiltinClosureBuf, poolOpts),
		cfPool:    refpool.New[CompiledFunction](opts.CompiledFunctionBuf, poolOpts),

		ptrPool: refpool.New[*Value](opts.ValuePtrBuf, poolOpts),
	}
}

func (a *Arena) Static() Static {
	return a.static
}

func (a *Arena) Payload() any {
	return a.payload
}

func (a *Arena) Reset() {
	if a.payload != nil {
		a.payload.Reset()
	}

	a.decPool.Reset(a.resetFull)
	a.strPool.Reset(a.resetFull)
	a.timePool.Reset(a.resetFull)

	a.intRangePool.Reset(a.resetFull)
	a.arrayIteratorPool.Reset(a.resetFull)
	a.bytesIteratorPool.Reset(a.resetFull)
	a.dictIteratorPool.Reset(a.resetFull)
	a.intRangeIteratorPool.Reset(a.resetFull)
	a.runesIteratorPool.Reset(a.resetFull)

	a.arrayPool.Reset(a.resetFull)
	a.bytesPool.Reset(a.resetFull)
	a.runesPool.Reset(a.resetFull)
	a.dictPool.Reset(a.resetFull)

	a.errorPool.Reset(a.resetFull)
	a.formatSpecPool.Reset(a.resetFull)
	a.biPool.Reset(a.resetFull)
	a.cfPool.Reset(a.resetFull)

	a.ptrPool.Reset(a.resetFull)
}

func (a *Arena) SetStatic(static Static) {
	a.static = static
}

/* FormatSpec (can be only static) */

func (a *Arena) ResolveFormatSpecValue(v Value) *FormatSpec {
	return &a.static.FormatSpecs[v.Data]
}

/* Decimal (can be static and dynamic) */

func (a *Arena) NewDecimalValue(d dec128.Dec128) (Value, bool) {
	if ref, p, ok := a.decPool.New(); ok {
		*p = d
		return Value{Type: VT_DECIMAL, Immutable: true, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinDecimalValue(v Value) {
	if !v.Static {
		a.decPool.Pin(v.Data)
	}
}

func (a *Arena) RetainDecimalValue(v Value) {
	if !v.Static {
		a.decPool.Retain(v.Data)
	}
}

func (a *Arena) ReleaseDecimalValue(v Value) {
	if !v.Static {
		a.decPool.Release(v.Data)
	}
}

func (a *Arena) ResolveDecimalValue(v Value) *dec128.Dec128 {
	if v.Static {
		return &a.static.Decimals[v.Data]
	}
	return a.decPool.Resolve(v.Data)
}

/* String (can be static and dynamic) */

func (a *Arena) NewStringValue(s string) (Value, bool) {
	if ref, p, ok := a.strPool.New(); ok {
		*p = s
		return Value{Type: VT_STRING, Immutable: true, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinStringValue(v Value) {
	if !v.Static {
		a.strPool.Pin(v.Data)
	}
}

func (a *Arena) RetainStringValue(v Value) {
	if !v.Static {
		a.strPool.Retain(v.Data)
	}
}

func (a *Arena) ReleaseStringValue(v Value) {
	if !v.Static {
		a.strPool.Release(v.Data)
	}
}

func (a *Arena) ResolveStringValue(v Value) *string {
	if v.Static {
		return &a.static.Strings[v.Data]
	}
	return a.strPool.Resolve(v.Data)
}

/* Time (can be only dynamic) */

func (a *Arena) NewTimeValue(t time.Time) (Value, bool) {
	if ref, p, ok := a.timePool.New(); ok {
		*p = t
		return Value{Type: VT_TIME, Immutable: true, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinTimeValue(v Value) {
	a.timePool.Pin(v.Data)
}

func (a *Arena) RetainTimeValue(v Value) {
	a.timePool.Retain(v.Data)
}

func (a *Arena) ReleaseTimeValue(v Value) {
	a.timePool.Release(v.Data)
}

func (a *Arena) ResolveTimeValue(v Value) *time.Time {
	return a.timePool.Resolve(v.Data)
}

/* IntRange (can be only dynamic) */

func (a *Arena) NewIntRange(start, stop, step int64) (Value, bool) {
	if ref, p, ok := a.intRangePool.New(); ok {
		p.Set(start, stop, step)
		return Value{Type: VT_INT_RANGE, Immutable: true, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinIntRangeValue(v Value) {
	a.intRangePool.Pin(v.Data)
}

func (a *Arena) RetainIntRangeValue(v Value) {
	a.intRangePool.Retain(v.Data)
}

func (a *Arena) ReleaseIntRangeValue(v Value) {
	a.intRangePool.Release(v.Data)
}

func (a *Arena) ResolveIntRangeValue(v Value) *IntRange {
	return a.intRangePool.Resolve(v.Data)
}

/* ArrayIterator (can be only dynamic) */

func (a *Arena) NewArrayIteratorValue(arr []Value) (Value, bool) {
	if ref, p, ok := a.arrayIteratorPool.New(); ok {
		p.Set(arr)
		return Value{Type: VT_ARRAY_ITERATOR, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinArrayIteratorValue(v Value) {
	a.arrayIteratorPool.Pin(v.Data)
}

func (a *Arena) RetainArrayIteratorValue(v Value) {
	a.arrayIteratorPool.Retain(v.Data)
}

func (a *Arena) ReleaseArrayIteratorValue(v Value) {
	a.arrayIteratorPool.Release(v.Data)
}

func (a *Arena) ResolveArrayIteratorValue(v Value) *ArrayIterator {
	return a.arrayIteratorPool.Resolve(v.Data)
}

/* BytesIterator (can be only dynamic) */

func (a *Arena) NewBytesIteratorValue(b []byte) (Value, bool) {
	if ref, p, ok := a.bytesIteratorPool.New(); ok {
		p.Set(b)
		return Value{Type: VT_BYTES_ITERATOR, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinBytesIteratorValue(v Value) {
	a.bytesIteratorPool.Pin(v.Data)
}

func (a *Arena) RetainBytesIteratorValue(v Value) {
	a.bytesIteratorPool.Retain(v.Data)
}

func (a *Arena) ReleaseBytesIteratorValue(v Value) {
	a.bytesIteratorPool.Release(v.Data)
}

func (a *Arena) ResolveBytesIteratorValue(v Value) *BytesIterator {
	return a.bytesIteratorPool.Resolve(v.Data)
}

/* DictIterator (can be only dynamic) */

func (a *Arena) NewDictIteratorValue(m map[string]Value) (Value, bool) {
	if ref, p, ok := a.dictIteratorPool.New(); ok {
		p.Set(m)
		return Value{Type: VT_DICT_ITERATOR, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinDictIteratorValue(v Value) {
	a.dictIteratorPool.Pin(v.Data)
}

func (a *Arena) RetainDictIteratorValue(v Value) {
	a.dictIteratorPool.Retain(v.Data)
}

func (a *Arena) ReleaseDictIteratorValue(v Value) {
	a.dictIteratorPool.Release(v.Data)
}

func (a *Arena) ResolveDictIteratorValue(v Value) *DictIterator {
	return a.dictIteratorPool.Resolve(v.Data)
}

/* IntRangeIterator (can be only dynamic) */

func (a *Arena) NewIntRangeIteratorValue(start, stop, step int64) (Value, bool) {
	if ref, p, ok := a.intRangeIteratorPool.New(); ok {
		p.Set(start, stop, step)
		return Value{Type: VT_INT_RANGE_ITERATOR, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinIntRangeIteratorValue(v Value) {
	a.intRangeIteratorPool.Pin(v.Data)
}

func (a *Arena) RetainIntRangeIteratorValue(v Value) {
	a.intRangeIteratorPool.Retain(v.Data)
}

func (a *Arena) ReleaseIntRangeIteratorValue(v Value) {
	a.intRangeIteratorPool.Release(v.Data)
}

func (a *Arena) ResolveIntRangeIteratorValue(v Value) *IntRangeIterator {
	return a.intRangeIteratorPool.Resolve(v.Data)
}

/* RunesIterator (can be only dynamic) */

func (a *Arena) NewRunesIteratorValue(s []rune) (Value, bool) {
	if ref, p, ok := a.runesIteratorPool.New(); ok {
		p.Set(s)
		return Value{Type: VT_RUNES_ITERATOR, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinRunesIteratorValue(v Value) {
	a.runesIteratorPool.Pin(v.Data)
}

func (a *Arena) RetainRunesIteratorValue(v Value) {
	a.runesIteratorPool.Retain(v.Data)
}

func (a *Arena) ReleaseRunesIteratorValue(v Value) {
	a.runesIteratorPool.Release(v.Data)
}

func (a *Arena) ResolveRunesIteratorValue(v Value) *RunesIterator {
	return a.runesIteratorPool.Resolve(v.Data)
}

/* Array (can be only dynamic) */

func (a *Arena) NewArrayValue(arr []Value, immutable bool) (Value, bool) {
	if ref, p, ok := a.arrayPool.New(); ok {
		p.Set(arr)
		return Value{Type: VT_ARRAY, Immutable: immutable, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinArrayValue(v Value) {
	a.arrayPool.Pin(v.Data)
}

func (a *Arena) RetainArrayValue(v Value) {
	a.arrayPool.Retain(v.Data)
}

func (a *Arena) ReleaseArrayValue(v Value) {
	a.arrayPool.Release(v.Data)
}

func (a *Arena) ResolveArrayValue(v Value) *Array {
	return a.arrayPool.Resolve(v.Data)
}

/* Bytes (can be only dynamic) */

func (a *Arena) NewBytesValue(b []byte, immutable bool) (Value, bool) {
	if ref, p, ok := a.bytesPool.New(); ok {
		p.Set(b)
		return Value{Type: VT_BYTES, Immutable: immutable, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinBytesValue(v Value) {
	a.bytesPool.Pin(v.Data)
}

func (a *Arena) RetainBytesValue(v Value) {
	a.bytesPool.Retain(v.Data)
}

func (a *Arena) ReleaseBytesValue(v Value) {
	a.bytesPool.Release(v.Data)
}

func (a *Arena) ResolveBytesValue(v Value) *Bytes {
	return a.bytesPool.Resolve(v.Data)
}

/* Runes (can be static and dynamic) */

func (a *Arena) NewRunesValue(r []rune, immutable bool) (Value, bool) {
	if ref, p, ok := a.runesPool.New(); ok {
		p.Set(r)
		return Value{Type: VT_RUNES, Immutable: immutable, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinRunesValue(v Value) {
	if !v.Static {
		a.runesPool.Pin(v.Data)
	}
}

func (a *Arena) RetainRunesValue(v Value) {
	if !v.Static {
		a.runesPool.Retain(v.Data)
	}
}

func (a *Arena) ReleaseRunesValue(v Value) {
	if !v.Static {
		a.runesPool.Release(v.Data)
	}
}

func (a *Arena) ResolveRunesValue(v Value) *Runes {
	if v.Static {
		return &a.static.Runes[v.Data]
	}
	return a.runesPool.Resolve(v.Data)
}

/* Dict (can be only dynamic) */

func (a *Arena) NewDictValue(m map[string]Value, immutable bool) (Value, bool) {
	if ref, p, ok := a.dictPool.New(); ok {
		p.Set(m)
		return Value{Type: VT_DICT, Immutable: immutable, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinDictValue(v Value) {
	a.dictPool.Pin(v.Data)
}

func (a *Arena) RetainDictValue(v Value) {
	a.dictPool.Retain(v.Data)
}

func (a *Arena) ReleaseDictValue(v Value) {
	a.dictPool.Release(v.Data)
}

func (a *Arena) ResolveDictValue(v Value) *Dict {
	return a.dictPool.Resolve(v.Data)
}

/* Record (can be only dynamic), based on dict pool */

func (a *Arena) NewRecordValue(m map[string]Value, immutable bool) (Value, bool) {
	if ref, p, ok := a.dictPool.New(); ok {
		p.Set(m)
		return Value{Type: VT_RECORD, Immutable: immutable, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinRecordValue(v Value) {
	a.dictPool.Pin(v.Data)
}

func (a *Arena) RetainRecordValue(v Value) {
	a.dictPool.Retain(v.Data)
}

func (a *Arena) ReleaseRecordValue(v Value) {
	a.dictPool.Release(v.Data)
}

func (a *Arena) ResolveRecordValue(v Value) *Dict {
	return a.dictPool.Resolve(v.Data)
}

/* Error (can be only dynamic) */

func (a *Arena) NewErrorValue(payload Value, kind string, fatal bool) (Value, bool) {
	if ref, p, ok := a.errorPool.New(); ok {
		payload.Pin(a) // mark payload as unmanaged because it's now also owned by the error value
		p.Payload = payload
		p.Kind = kind
		p.Fatal = fatal
		return Value{Type: VT_ERROR, Immutable: true, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) NewRuntimeErrorValue(kind string, fatal bool, message string) (Value, bool) {
	if payload, ok := a.NewStringValue(message); ok {
		return a.NewErrorValue(payload, kind, fatal)
	}
	return Undefined, false
}

func (a *Arena) PinErrorValue(v Value) {
	a.errorPool.Pin(v.Data)
}

func (a *Arena) RetainErrorValue(v Value) {
	a.errorPool.Retain(v.Data)
}

func (a *Arena) ReleaseErrorValue(v Value) {
	a.errorPool.Release(v.Data)
}

func (a *Arena) ResolveErrorValue(v Value) *Error {
	return a.errorPool.Resolve(v.Data)
}

/* BuiltinClosure (can be only dynamic) */

func (a *Arena) NewBuiltinClosureValue(name string, fn NativeFunc, arity int8, variadic bool) (Value, bool) {
	if ref, p, ok := a.biPool.New(); ok {
		p.Set(fn, name, arity, variadic)
		return Value{Type: VT_BUILTIN_CLOSURE, Immutable: true, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinBuiltinClosureValue(v Value) {
	a.biPool.Pin(v.Data)
}

func (a *Arena) RetainBuiltinClosureValue(v Value) {
	a.biPool.Retain(v.Data)
}

func (a *Arena) ReleaseBuiltinClosureValue(v Value) {
	a.biPool.Release(v.Data)
}

func (a *Arena) ResolveBuiltinClosureValue(v Value) *BuiltinClosure {
	return a.biPool.Resolve(v.Data)
}

/* CompiledFunction (can be static and dynamic) */

func (a *Arena) NewCompiledFunctionValue(instructions []byte, free []*Value, sourceMap map[int]Pos, numLocals, maxStack int, numParameters int8, varArgs bool, namedResult int8) (Value, bool) {
	if ref, p, ok := a.cfPool.New(); ok {
		p.Set(instructions, free, sourceMap, numLocals, maxStack, numParameters, varArgs, namedResult)
		return Value{Type: VT_COMPILED_FUNCTION, Immutable: true, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinCompiledFunctionValue(v Value) {
	if !v.Static {
		a.cfPool.Pin(v.Data)
	}
}

func (a *Arena) RetainCompiledFunctionValue(v Value) {
	if !v.Static {
		a.cfPool.Retain(v.Data)
	}
}

func (a *Arena) ReleaseCompiledFunctionValue(v Value) {
	if !v.Static {
		a.cfPool.Release(v.Data)
	}
}

func (a *Arena) ResolveCompiledFunctionValue(v Value) *CompiledFunction {
	if v.Static {
		return &a.static.CompiledFunctions[v.Data]
	}
	return a.cfPool.Resolve(v.Data)
}

/* ValuePtr (can be only dynamic) */

func (a *Arena) NewValuePtrValue(p *Value) (Value, bool) {
	if ref, poolPtr, ok := a.ptrPool.New(); ok {
		p.Pin(a) // mark pointed value as unmanaged because it's now also owned by the pointer value
		*poolPtr = p
		return Value{Type: VT_VALUE_PTR, Data: ref}, true
	}
	return Undefined, false
}

func (a *Arena) PinValuePtrValue(v Value) {
	a.ptrPool.Pin(v.Data)
}

func (a *Arena) RetainValuePtrValue(v Value) {
	a.ptrPool.Retain(v.Data)
}

func (a *Arena) ReleaseValuePtrValue(v Value) {
	a.ptrPool.Release(v.Data)
}

func (a *Arena) ResolveValuePtrValue(v Value) **Value {
	return a.ptrPool.Resolve(v.Data)
}
