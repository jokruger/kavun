package core

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unsafe"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
	"github.com/jokruger/kavun/internal/format"
)

const (
	arrayTypeName          = "array"
	immutableArrayTypeName = "immutable-array"
)

type Array = Seq[Value]

func NewArrayValue(arr []Value, immutable bool) Value {
	o := &Array{}
	o.Set(arr)
	return Value{Type: value.Array, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeArray = ValueTypeDescr{
	Name:         SeqNameHook(arrayTypeName, immutableArrayTypeName),                            // PURE by contract
	String:       arrayTypeString,                                                               // PURE by contract
	Format:       arrayTypeFormat,                                                               // PURE by contract
	Interface:    arrayTypeInterface,                                                            // PURE by contract
	EncodeJSON:   arrayTypeEncodeJSON,                                                           // PURE by contract
	EncodeBinary: arrayTypeEncodeBinary,                                                         // PURE by contract
	DecodeBinary: arrayTypeDecodeBinary,                                                         // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return len((*Array)(v.Ptr).Elements) > 0, nil }, // PURE by contract
	IsIterable:   ConstHook(true),                                                               // PURE by contract
	Iterator:     arrayTypeIterator,                                                             // PURE by contract (constructs fresh iterator)
	Equal:        arrayTypeEqual,                                                                // PURE by contract
	BinaryOp:     arrayTypeBinaryOp,                                                             // PURE by contract
	Copy:         arrayTypeCopy,                                                                 // PURE by contract
	Len:          func(v Value) int64 { return int64(len((*Array)(v.Ptr).Elements)) },           // PURE by contract
	MethodCall:   arrayTypeMethodCall,                                                           // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       SeqAccessHook(RefValue, arrayTypeResolve),                                     // PURE by contract
	Assign:       SeqAssignHook(arrayTypeResolve, Value.AsValue, anyTypeName),                   // IMPURE by contract
	Contains:     arrayTypeContains,                                                             // PURE by contract
	Append:       arrayTypeAppend,                                                               // MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append)
	Slice:        SeqSliceHook(NewArrayValue, arrayTypeResolve),                                 // PURE by contract
	SliceStep:    SeqSliceStepHook(NewArrayValue, arrayTypeResolve),                             // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return len((*Array)(v.Ptr).Elements) > 0, true }, // PURE by contract
	AsString:     arrayTypeAsString,                                                             // PURE by contract
	AsRunes:      arrayTypeAsRunes,                                                              // PURE by contract
	AsBytes:      arrayTypeAsBytes,                                                              // PURE by contract
	AsArray:      func(v Value) ([]Value, bool) { return (*Array)(v.Ptr).Elements, true },       // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order methods
	// (filter/map/reduce/for_each/all/any/find/count) are pure in isolation — impurity can only enter via a
	// function-valued argument.
	IsMethodPure: func(name string) bool { return !strings.HasSuffix(name, "_in_place") },
}

func arrayTypeResolve(v Value) *Array {
	return (*Array)(v.Ptr)
}

func arrayTypeString(v Value) string {
	o := (*Array)(v.Ptr)
	parts := make([]string, len(o.Elements))
	for i, e := range o.Elements {
		parts[i] = e.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func arrayTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return arrayTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(arrayTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(arrayTypeString(v), sp, fspec.AlignLeft), nil
}

func arrayTypeInterface(v Value) any {
	o := (*Array)(v.Ptr)
	res := make([]any, len(o.Elements))
	for i, val := range o.Elements {
		res[i] = val.Interface()
	}
	return res
}

func arrayTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Array)(v.Ptr)
	var b []byte
	b = append(b, '[')
	len1 := len(o.Elements) - 1
	for idx, elem := range o.Elements {
		eb, err := elem.EncodeJSON()
		if err != nil {
			return nil, fmt.Errorf("array element at index %d: %w", idx, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
	}
	b = append(b, ']')
	return b, nil
}

func arrayTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Array)(v.Ptr)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for i, elem := range o.Elements {
		eb, err := elem.EncodeBinary()
		if err != nil {
			return nil, fmt.Errorf("array element at index %d: %w", i, err)
		}
		b = binary.AppendBytes(b, eb)
	}

	return b, nil
}

func arrayTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	count, err := binary.ReadUint64(data, &offset, "array (elements count)")
	if err != nil {
		return err
	}

	arr := make([]Value, int(count))
	for i := range arr {
		eb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("array element at index %d", i))
		if err != nil {
			return err
		}
		if err := arr[i].DecodeBinary(eb); err != nil {
			return fmt.Errorf("array element at index %d: %w", i, err)
		}
	}
	if offset != len(data) {
		return fmt.Errorf("array: trailing %d bytes", len(data)-offset)
	}

	*v = NewArrayValue(arr, v.Immutable)
	return nil
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func arrayTypeIterator(v Value) (Value, error) {
	return NewArrayIteratorValue((*Array)(v.Ptr).Elements), nil
}

func arrayTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Array:
		l := (*Array)(v.Ptr).Elements
		r := (*Array)(other.Ptr).Elements
		if len(l) != len(r) {
			return false
		}
		for i, e := range l {
			if !e.Equal(r[i]) {
				return false
			}
		}
		return true
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func arrayTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		// array has no cross-type relationship with anything — same-type only, always resolved non-reflected, so the
		// reflected branch always declines
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	if other.Type != value.Array {
		return ValueTypes[other.Type].BinaryOp(other, v, op, true)
	}

	l := (*Array)(v.Ptr)
	r := (*Array)(other.Ptr)
	switch op {
	case token.Add:
		t := make([]Value, len(l.Elements)+len(r.Elements))
		copy(t, l.Elements)
		copy(t[len(l.Elements):], r.Elements)
		return NewArrayValue(t, false), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), other.TypeName())
}

// deep=true recursively copies every element (today's copy() semantics); deep=false only clones the top-level
// slice header, leaving nested containers sharing the source (copy_shallow()).
func arrayTypeCopy(v Value, deep bool) (Value, error) {
	o := (*Array)(v.Ptr)
	c := make([]Value, len(o.Elements))
	if !deep {
		copy(c, o.Elements)
		return NewArrayValue(c, false), nil
	}
	for i, e := range o.Elements {
		t, err := e.Copy(true)
		if err != nil {
			return Undefined, err
		}
		c[i] = t
	}
	return NewArrayValue(c, false), nil
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func arrayTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return arrayTypeCopy(v, true)

	case "copy_shallow":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return arrayTypeCopy(v, false)

	case "freeze_shallow":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v.ToImmutable()

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v.Freeze()

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "bytes":
		// element-wise, all-or-nothing: a failing element fails the conversion —
		// the silent NUL/mod-256 corruption is gone
		bs, ok := ElementsToBytes(o.Elements)
		return convMember(name, arrayTypeName, args, ok, NewBytesValue(bs, false))

	case "string":
		// the element step is the rune conversion (string and runes are one text),
		// so ["a","b"].string() raises — join() is the concatenative spelling
		rs, ok := ElementsToRunes(o.Elements)
		return convMember(name, arrayTypeName, args, ok, NewStringValue(string(rs)))

	case "runes":
		rs, ok := ElementsToRunes(o.Elements)
		return convMember(name, arrayTypeName, args, ok, NewRunesValue(rs, false))

	case "record":
		// the ENTRIES reading: each element is exactly a 2-element array [key, value];
		// the index->element decomposition (invented keys, scrambled order) is gone
		m, ok := ElementsToEntries(o.Elements)
		return convMember(name, arrayTypeName, args, ok, NewRecordValue(m, false))

	case "dict":
		m, ok := ElementsToEntries(o.Elements)
		return convMember(name, arrayTypeName, args, ok, NewDictValue(m, false))

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := arrayTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(len(o.Elements) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(len(o.Elements))), nil

	case "first":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return o.Elements[0], nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return o.Elements[len(o.Elements)-1], nil

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(arrayTypeContains(v, args[0])), nil

	case "min":
		return arrayFnMin(v, args)

	case "max":
		return arrayFnMax(v, args)

	case "sum":
		return arrayFnSum(v, args)

	case "avg":
		return arrayFnAvg(v, args)

	case "sort":
		return arrayFnSort(v, args, false)

	case "sort_in_place":
		return arrayFnSort(v, args, true)

	case "dedup":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("dedup", "0", len(args))
		}
		o := (*Array)(v.Ptr)
		out := make([]Value, 0, len(o.Elements))
		for i, e := range o.Elements {
			if i == 0 || !out[len(out)-1].Equal(e) {
				out = append(out, e)
			}
		}
		return NewArrayValue(out, false), nil

	case "unique":
		return arrayFnUnique(v, args)

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Array)(v.Ptr)
		n := len(o.Elements)
		t := make([]Value, n)
		for i, x := range o.Elements {
			t[n-1-i] = x
		}
		return NewArrayValue(t, false), nil

	case "reverse_in_place":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if v.Immutable {
			return Undefined, errs.NewNotReversibleError(v.TypeName())
		}
		o := (*Array)(v.Ptr)
		slices.Reverse(o.Elements)
		return v, nil

	case "filter":
		return SeqFilter(vm, v, args, RefValue, NewArrayValue, arrayTypeResolve)

	case "count":
		return SeqCount(vm, v, args, RefValue, arrayTypeResolve)

	case "all":
		return SeqAll(vm, v, args, RefValue, arrayTypeResolve)

	case "any":
		return SeqAny(vm, v, args, RefValue, arrayTypeResolve)

	case "map":
		return SeqMap(vm, v, args, RefValue, arrayTypeResolve)

	case "reduce":
		return SeqReduce(vm, v, args, RefValue, arrayTypeResolve)

	case "for_each":
		return SeqForEach(vm, v, args, RefValue, arrayTypeResolve)

	case "find":
		return SeqFind(vm, v, args, RefValue, arrayTypeResolve)

	case "chunk":
		return SeqChunk(v, args, NewArrayValue, arrayTypeResolve)

	case "chunk_view":
		return SeqChunkView(v, args, NewArrayValue, arrayTypeResolve)

	case "slice":
		return SeqSlice(v, args)

	case "slice_view":
		return SeqSliceView(v, args, NewArrayValue, arrayTypeResolve)

	case "is_view":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsView), nil

	case "append":
		return arrayTypeAppend(v, args, false)

	case "append_in_place":
		return arrayTypeAppend(v, args, true)

	case "splice_in_place":
		return SeqSplice(append([]Value{v}, args...), true, NewArrayValue, arrayTypeResolve, arraySpliceItems, arrayTypeName)

	case "splice":
		return SeqSplice(append([]Value{v}, args...), false, NewArrayValue, arrayTypeResolve, arraySpliceItems, arrayTypeName)

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		src := o.Elements
		sl := len(src)
		out := make([]Value, n*sl)
		for i := range n {
			copy(out[i*sl:], src)
		}
		return NewArrayValue(out, false), nil

	case "join":
		return arrayFnJoin(v, args)

	case "flatten":
		return arrayFnFlatten(v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func arrayTypeContains(v Value, e Value) bool {
	o := (*Array)(v.Ptr)
	switch e.Type {
	case value.Array:
		t := (*Array)(e.Ptr)
		if len(t.Elements) == 0 {
			return true
		}
		if len(o.Elements) < len(t.Elements) {
			return false
		}
		for i := range o.Elements {
			if o.Elements[i].Equal(t.Elements[0]) {
				match := true
				for j := 1; j < len(t.Elements); j++ {
					if i+j >= len(o.Elements) || !o.Elements[i+j].Equal(t.Elements[j]) {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
		return false

	default:
		for i := range o.Elements {
			if o.Elements[i].Equal(e) {
				return true
			}
		}
		return false
	}
}

// mutate=true: IMPURE, mutates the receiver's own backing struct in place via Set (append_in_place()) — reuses
// spare capacity or reallocates exactly like Go's append, visible to every other live alias into this body.
// Rejects an immutable receiver. Not folded by the optimizer. mutate=false: PURE, returns a fresh, independent
// array with the items appended (append()) — never touches the receiver's backing storage, works regardless of
// the receiver's mutability. Both accept zero item arguments as a legal no-op. See docs/purity.md.
func arrayTypeAppend(v Value, args []Value, mutate bool) (Value, error) {
	o := (*Array)(v.Ptr)

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotAppendableError(v.TypeName())
		}
		o.Set(append(o.Elements, args...))
		return v, nil
	}

	// Pure: build a fresh, independent array — never touch o's own backing storage (per docs/conventions.md's
	// variadic/slice argument immutability rule; append(o.Elements, ...) would risk writing into o's own array).
	t := make([]Value, 0, len(o.Elements)+len(args))
	t = append(t, o.Elements...)
	t = append(t, args...)
	return NewArrayValue(t, false), nil
}

// arraySpliceItems is array's SeqSplice convertItems function: elements are already Values, so splice's insert
// items need no conversion or flattening — unlike bytes'/runes', which reuse their append-item conversion here
// (see bytesAppendItems/runesAppendItems).
func arraySpliceItems(args []Value, _ string) ([]Value, error) {
	return args, nil
}

func arrayTypeAsString(v Value) (string, bool) {
	rs, ok := arrayTypeAsRunes(v)
	if !ok {
		return "", false
	}
	return string(rs), true
}

func arrayTypeAsRunes(v Value) ([]rune, bool) {
	o := (*Array)(v.Ptr)
	rs := make([]rune, len(o.Elements))
	for i, e := range o.Elements {
		r, ok := e.AsInt()
		if !ok || r < 0 || r > unicode.MaxRune {
			return nil, false
		}
		rs[i] = rune(r)
	}
	return rs, true
}

func arrayTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Array)(v.Ptr)
	bs := make([]byte, len(o.Elements))
	for i, e := range o.Elements {
		b, ok := e.AsInt()
		if !ok || b < 0 || b > 255 {
			return nil, false
		}
		bs[i] = byte(b)
	}
	return bs, true
}

// mutate=false: PURE, returns a fresh, independently-owned sorted array, source untouched. mutate=true: sorts
// the receiver's own backing storage directly, visible to every other alias sharing it; rejects an immutable
// receiver.
func arrayFnSort(v Value, args []Value, mutate bool) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sort", "0", len(args))
	}
	if mutate && v.Immutable {
		return Undefined, errs.NewNotSortableError(v.TypeName())
	}

	var err error
	o := (*Array)(v.Ptr)
	var t []Value
	if mutate {
		t = o.Elements
	} else {
		t = make([]Value, len(o.Elements))
		copy(t, o.Elements)
	}
	slices.SortFunc(t, func(x, y Value) int {
		less, e := x.BinaryOp(token.Less, y)
		if e != nil {
			err = e
			return 0
		}
		lt, e2 := less.IsTrue()
		if e2 != nil {
			err = e2
			return 0
		}
		if !lt {
			if x.Equal(y) {
				return 0
			}
			return 1
		}
		return -1
	})
	if err != nil {
		return Undefined, err
	}
	if mutate {
		return v, nil
	}

	return NewArrayValue(t, false), nil
}

// unique returns a new array with duplicate elements removed, regardless of their position in the array. This is less
// efficient than dedup, but does not require the input array to be sorted.
func arrayFnUnique(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	out := make([]Value, 0, len(o.Elements))
	for _, e := range o.Elements {
		seen := false
		for _, u := range out {
			if u.Equal(e) {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, e)
		}
	}

	return NewArrayValue(out, false), nil
}

func arrayFnMin(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("min", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	e := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		less, err := o.Elements[i].BinaryOp(token.Less, e)
		if err != nil {
			return Undefined, err
		}
		lt, terr := less.IsTrue()
		if terr != nil {
			return Undefined, terr
		}
		if lt {
			e = o.Elements[i]
		}
	}

	return e, nil
}

func arrayFnMax(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("max", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	e := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		greater, err := o.Elements[i].BinaryOp(token.Greater, e)
		if err != nil {
			return Undefined, err
		}
		gt, terr := greater.IsTrue()
		if terr != nil {
			return Undefined, terr
		}
		if gt {
			e = o.Elements[i]
		}
	}

	return e, nil
}

func arrayFnSum(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sum", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	var err error
	s := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		s, err = s.BinaryOp(token.Add, o.Elements[i])
		if err != nil {
			return Undefined, err
		}
	}

	return s, nil
}

func arrayFnAvg(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("avg", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	var err error
	sum := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		sum, err = sum.BinaryOp(token.Add, o.Elements[i])
		if err != nil {
			return Undefined, err
		}
	}

	length := IntValue(int64(len(o.Elements)))
	avg, err := sum.BinaryOp(token.Quo, length)
	if err != nil {
		return Undefined, err
	}

	return avg, nil
}

// arrayFnJoin implements `array.join(sep)`.
// sep types: string | runes | byte | rune.
// Result type follows sep: string→string, runes→runes, byte→bytes, rune→runes.
// With no argument, defaults to empty string separator.
func arrayFnJoin(v Value, args []Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("join", "0 or 1", len(args))
	}
	o := (*Array)(v.Ptr)
	if len(args) == 0 {
		s, err := joinElementsToString(o.Elements, "")
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil
	}
	return joinSeqWithSep(o.Elements, args[0], "join")
}

// joinSeqWithSep performs the join given pre-resolved seq elements and a separator value.
// Returns a value whose type is determined by the sep type.
func joinSeqWithSep(elems []Value, sep Value, name string) (Value, error) {
	switch sep.Type {
	case value.String:
		s, err := joinElementsToString(elems, *(*string)(sep.Ptr))
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case value.Runes:
		s, err := joinElementsToString(elems, string((*Runes)(sep.Ptr).Elements))
		if err != nil {
			return Undefined, err
		}
		return NewRunesValue([]rune(s), false), nil

	case value.Rune:
		s, err := joinElementsToString(elems, string(rune(sep.Data)))
		if err != nil {
			return Undefined, err
		}
		return NewRunesValue([]rune(s), false), nil

	case value.Byte:
		s, err := joinElementsToString(elems, string([]byte{byte(sep.Data)}))
		if err != nil {
			return Undefined, err
		}
		return NewBytesValue([]byte(s), false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string, runes, byte, or rune", sep.TypeName())
	}
}

func arrayFnFlatten(v Value, args []Value) (Value, error) {
	const name = "flatten"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	depth := 1
	if len(args) == 1 {
		d, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "int", args[0].TypeName())
		}
		if d < 0 {
			depth = -1
		} else {
			depth = int(d)
		}
	}
	o := (*Array)(v.Ptr)
	out := make([]Value, 0, len(o.Elements))
	out = flattenAppend(out, o.Elements, depth)
	arr := make([]Value, len(out))
	copy(arr, out)
	return NewArrayValue(arr, false), nil
}

// flattenAppend appends each element of src to dst, unwrapping nested arrays up to `depth` levels.
// depth == 0 means no unwrapping (shallow copy).
// depth < 0 means unbounded (fully recursive).
func flattenAppend(dst []Value, src []Value, depth int) []Value {
	if depth == 0 {
		return append(dst, src...)
	}
	next := depth
	if next > 0 {
		next--
	}
	for _, e := range src {
		if e.Type == value.Array {
			inner := (*Array)(e.Ptr).Elements
			dst = flattenAppend(dst, inner, next)
		} else {
			dst = append(dst, e)
		}
	}
	return dst
}
