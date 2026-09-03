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
	// No AsString hook: an array has no canonical text (its element-wise conversion is a transcoding
	// constructor, not a render), so a dict key, a join element, or any implicit to-string consumer raises
	// instead of silently keying/rendering the transcode. The explicit conversions stay: .string(), string(a).
	AsRunes: arrayTypeAsRunes,                                                        // PURE by contract
	AsBytes: arrayTypeAsBytes,                                                        // PURE by contract
	AsArray: func(v Value) ([]Value, bool) { return (*Array)(v.Ptr).Elements, true }, // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order methods
	// (keep/map/reduce/for_each/all/any/find/count) are pure in isolation — impurity can only enter via a
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
			return nil, jsonPathPrefix(fmt.Sprintf("[%d]", idx), err)
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
		// the left operand had no reading for an array and handed the operation over. Only `+` has a
		// reflected form, because only the add side has a front spelling: `x + a` is exactly
		// `a.prepend(x)`, the mirror of `a + x` = `a.append(x)`. Removal has no front member, so `-`
		// — and every other operator — raises here rather than inventing one.
		if op != token.Add {
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
		}
		// the universal contracts outrank the element reading on this side too: an error raises
		// through every operator rather than becoming the first element (undefined never reaches
		// here — it propagates without handing over)
		if other.Type == value.Error {
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
		}
		l := (*Array)(v.Ptr)
		items := arrayAddItems([]Value{other})
		t := make([]Value, 0, len(items)+len(l.Elements))
		t = append(t, items...)
		t = append(t, l.Elements...)
		return NewArrayValue(t, false), nil
	}

	// the universal contracts outrank the element reading: undefined propagates through
	// every operator and error raises through every operator — appending either as an
	// element is the member spelling (push)
	if other.Type == value.Undefined || other.Type == value.Error {
		return ValueTypes[other.Type].BinaryOp(other, v, op, true)
	}

	l := (*Array)(v.Ptr)
	switch op {
	case token.Add:
		// exactly append's reading: an operand of the receiver's OWN KIND — another array —
		// contributes its elements as a run; anything else is one element
		items := arrayAddItems([]Value{other})
		t := make([]Value, 0, len(l.Elements)+len(items))
		t = append(t, l.Elements...)
		t = append(t, items...)
		return NewArrayValue(t, false), nil

	case token.Mul:
		// exactly repeat's reading: the right operand is a COUNT, not an element — a sequence
		// times a number is that sequence n times over. There is no reflected direction:
		// `seq * n` reads as "apply n to the sequence", `n * seq` has no such reading
		if n, isCount, err := SeqRepeatOperand(other); isCount {
			if err != nil {
				return Undefined, err
			}
			src := l.Elements
			sl := len(src)
			total, terr := SeqRepeatTotal(op.String(), n, sl)
			if terr != nil {
				return Undefined, terr
			}
			t := make([]Value, total)
			// step by the receiver's length, never by the count (see the member form)
			for i := 0; i < total; i += sl {
				copy(t[i:], src)
			}
			return NewArrayValue(t, false), nil
		}

	case token.Sub:
		// exactly remove's value readings: an array operand removes every occurrence of
		// the contiguous run (never set difference), anything else every equal element
		eq := func(a, b Value) bool { return a.Equal(b) }
		switch other.Type {
		case value.Array:
			run, _ := other.AsArray()
			kept := make([]Value, 0, len(l.Elements))
			scanRuns(l.Elements, [][]Value{run}, eq,
				func(int, int) {},
				func(i int) { kept = append(kept, l.Elements[i]) })
			return NewArrayValue(kept, false), nil
		}
		kept := make([]Value, 0, len(l.Elements))
		for _, e := range l.Elements {
			if !e.Equal(other) {
				kept = append(kept, e)
			}
		}
		return NewArrayValue(kept, false), nil
	}

	if other.Type != value.Array {
		return ValueTypes[other.Type].BinaryOp(other, v, op, true)
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
		// a conversion CONSTRUCTS, on its own type like any other: a new, independent, mutable
		// shallow copy, exactly array(a) / a.copy_shallow(). Never the receiver itself — an alias
		// handed out under a conversion spelling wrote through to the caller's array, and it was
		// invisible (is_view() reports borrowing, and this was not one). Sharing is slice_view's job.
		c, err := arrayTypeCopy(v, false)
		if err != nil {
			return Undefined, err
		}
		return convMember(name, arrayTypeName, args, true, c)

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
			return Undefined, errs.FromFormatSpecError(name, err)
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
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		return o.Elements[0], nil

	case "last":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		return o.Elements[len(o.Elements)-1], nil

	case "contains", "count", "keep", "remove", "any", "all", "remove_in_place", "keep_in_place":
		// the receiver's own kind is `array` and nothing else; every other value
		// is one element — an array can hold one of anything
		mutate := strings.HasSuffix(name, "_in_place")
		if mutate && v.Immutable {
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		res, err := SeqMatchMember(vm, name, v, args, RefValue, NewArrayValue, arrayTypeResolve,
			func(a Value) (Value, bool, error) { return a, true, nil },
			func(a Value) bool { return a.Type == value.Array },
			func(a Value) ([]Value, error) {
				elems, ok := a.AsArray()
				if !ok {
					return nil, errs.NewInvalidArgumentTypeError(name, "argument", "array", a.TypeName())
				}
				return elems, nil
			},
			func(a, b Value) bool { return a.Equal(b) },
			IsBlankElement)
		if err != nil {
			return Undefined, err
		}
		if mutate {
			o.Set((*Array)(res.Ptr).Elements)
			return v, nil
		}
		return res, nil

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

	case "dedup", "dedup_in_place":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if name == "dedup_in_place" && v.Immutable {
			return Undefined, immutableTwinError(name, v.TypeName())
		}
		out := make([]Value, 0, len(o.Elements))
		for i, e := range o.Elements {
			if i == 0 || !out[len(out)-1].Equal(e) {
				out = append(out, e)
			}
		}
		if name == "dedup_in_place" {
			o.Set(out)
			return v, nil
		}
		return NewArrayValue(out, false), nil

	case "unique", "unique_in_place":
		if name == "unique_in_place" && v.Immutable {
			return Undefined, immutableTwinError(name, v.TypeName())
		}
		res, err := arrayFnUnique(v, args)
		if err != nil {
			return Undefined, err
		}
		if name == "unique_in_place" {
			o.Set((*Array)(res.Ptr).Elements)
			return v, nil
		}
		return res, nil

	case "flat_map":
		// map-then-concatenate: each callback result is read like an add-side
		// operand — an array result spreads, undefined contributes nothing,
		// anything else is one element
		call, err := seqMapCallback(name, args)
		if err != nil {
			return Undefined, err
		}
		out := make([]Value, 0, len(o.Elements))
		for i, e := range o.Elements {
			res, err := call(vm, i, e)
			if err != nil {
				return Undefined, err
			}
			switch res.Type {
			case value.Undefined:
			case value.Array:
				out = append(out, (*Array)(res.Ptr).Elements...)
			default:
				out = append(out, res)
			}
		}
		return NewArrayValue(out, false), nil

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
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		o := (*Array)(v.Ptr)
		slices.Reverse(o.Elements)
		return v, nil

	case "map":
		return SeqMap(vm, v, args, RefValue, arrayTypeResolve)

	case "reduce":
		return SeqReduce(vm, v, args, RefValue, arrayTypeResolve)

	case "for_each":
		return SeqForEach(vm, v, args, RefValue, arrayTypeResolve)

	case "index", "index_last":
		// an array's own kind is `array` and nothing else; the text triple and
		// range stay elements here
		return SeqIndex(vm, v, args, name == "index_last", RefValue, arrayTypeResolve,
			func(a Value) bool { return a.Type == value.Array },
			func(elems []Value, run Value, last bool) (int64, bool, error) {
				rs, ok := run.AsArray()
				if !ok {
					return -1, false, errs.NewInvalidArgumentTypeError(name, "first", "array", run.TypeName())
				}
				idx, found := SeqIndexRun(elems, rs, RefValue, last)
				return idx, found, nil
			},
			nil, // an array can hold one of anything — every value is an element
			IsBlankElement)

	case "trim", "trim_start", "trim_end", "has_prefix", "has_suffix",
		"remove_prefix", "remove_suffix", "replace", "pad_start", "pad_end",
		"trim_in_place", "trim_start_in_place", "trim_end_in_place",
		"remove_prefix_in_place", "remove_suffix_in_place", "replace_in_place",
		"pad_start_in_place", "pad_end_in_place":
		// sequence verbs, not text verbs: a fill element, a set of elements, a run.
		// The pads' fill is any value (undefined by default — the untyped slot's blank)
		mutate := strings.HasSuffix(name, "_in_place")
		if mutate && v.Immutable {
			return Undefined, immutableTwinError(name, v.TypeName())
		}
		res, err := SeqStructuralMember(name, v, args, NewArrayValue, arrayTypeResolve,
			arrayEncodeStructuralArg,
			func(_ string, a Value) (Value, error) { return a, nil }, Undefined,
			func(a, b Value) bool { return a.Equal(b) }, IsBlankElement)
		if err != nil {
			return Undefined, err
		}
		if mutate {
			o.Set((*Array)(res.Ptr).Elements)
			return v, nil
		}
		return res, nil

	case "chunk":
		return SeqChunk(v, args, NewArrayValue, arrayTypeResolve)

	case "chunk_view":
		return SeqChunkView(v, args, NewArrayValue, arrayTypeResolve)

	case "slice":
		return SeqSlice(v, args)

	case "slice_view":
		return SeqSliceView(v, args, NewArrayValue, arrayTypeResolve)

	case "append":
		return arrayTypeAppend(v, args, false)

	case "append_in_place":
		return arrayTypeAppend(v, args, true)

	case "prepend":
		return arrayTypeAddFront(v, args, false)

	case "prepend_in_place":
		return arrayTypeAddFront(v, args, true)

	case "push":
		return arrayTypePush(v, args, false, false)

	case "push_in_place":
		return arrayTypePush(v, args, true, false)

	case "push_first":
		return arrayTypePush(v, args, false, true)

	case "push_first_in_place":
		return arrayTypePush(v, args, true, true)

	case "insert", "insert_in_place":
		// the element-inserting sibling of splice: insert(i, ...items) — each item is
		// ONE element whatever its type, never spreads; the position is a positional
		// EDIT and raises out of range
		mutate := name == "insert_in_place"
		if mutate && v.Immutable {
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		at, err := seqEditPos(name, args, int64(len(o.Elements)))
		if err != nil {
			return Undefined, err
		}
		if mutate {
			o.Set(slices.Insert(o.Elements, int(at), args[1:]...))
			return v, nil
		}
		return NewArrayValue(slices.Insert(slices.Clone(o.Elements), int(at), args[1:]...), false), nil

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
		total, err := SeqRepeatTotal(name, n, sl)
		if err != nil {
			return Undefined, err
		}
		out := make([]Value, total)
		// step by the receiver's length, never by the count: an empty receiver has total 0 and must not
		// spin n times copying nothing
		for i := 0; i < total; i += sl {
			copy(out[i:], src)
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

// arrayTypeContains is the `in` operator: contains' VALUE readings — an operand of the receiver's own
// FAMILY (array or range) is a contiguous run, anything else one element; the empty run is contained
// everywhere. A callable raises — an operator operand is always a value.
func arrayTypeContains(v Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	o := (*Array)(v.Ptr)
	eq := func(a, b Value) bool { return a.Equal(b) }
	switch e.Type {
	case value.Array:
		run, _ := e.AsArray()
		if len(run) == 0 {
			return true, nil
		}
		for i := range o.Elements {
			if longestRunAt(o.Elements, i, [][]Value{run}, eq) > 0 {
				return true, nil
			}
		}
		return false, nil
	}
	for i := range o.Elements {
		if o.Elements[i].Equal(e) {
			return true, nil
		}
	}
	return false, nil
}

// arrayEncodeStructuralArg reads a structural member's argument on an array receiver: an argument of the
// receiver's own kind (another array) is a run; anything else is one element. Callables never reach it —
// the classifiers dispatch them first (a predicate where the member declares one, a refusal elsewhere).
func arrayEncodeStructuralArg(_ string, a Value) ([]Value, bool, error) {
	if a.Type == value.Array {
		return (*Array)(a.Ptr).Elements, false, nil
	}
	return []Value{a}, true, nil
}

// arrayAddItems flattens append/prepend's variadic operands: an argument of the receiver's OWN KIND — another
// array — contributes its elements as a run, so the member agrees with the + operator; every other value is one
// element, an array can hold one of anything. A range is one element like the rest: materializing it is spelled
// at the call site (`a + r.array()`), never inferred, so a script that never names `array` never gets one. The
// element spelling for a nested array is push(row) or the wrap append([row]).
func arrayAddItems(args []Value) []Value {
	items := make([]Value, 0, len(args))
	for _, a := range args {
		if a.Type == value.Array {
			items = append(items, (*Array)(a.Ptr).Elements...)
			continue
		}
		items = append(items, a)
	}
	return items
}

// mutate=true: IMPURE, mutates the receiver's own backing struct in place via Set (append_in_place()) — reuses
// spare capacity or reallocates exactly like Go's append, visible to every other live alias into this body.
// Rejects an immutable receiver. Not folded by the optimizer. mutate=false: PURE, returns a fresh, independent
// array with the items appended (append()) — never touches the receiver's backing storage, works regardless of
// the receiver's mutability. Both accept zero item arguments as a legal no-op. See docs/purity.md.
func arrayTypeAppend(v Value, args []Value, mutate bool) (Value, error) {
	o := (*Array)(v.Ptr)
	items := arrayAddItems(args)

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotMutableError("append_in_place", v.TypeName())
		}
		o.Set(append(o.Elements, items...))
		return v, nil
	}

	// Pure: build a fresh, independent array — never touch o's own backing storage (per docs/conventions.md's
	// variadic/slice argument immutability rule; append(o.Elements, ...) would risk writing into o's own array).
	t := make([]Value, 0, len(o.Elements)+len(items))
	t = append(t, o.Elements...)
	t = append(t, items...)
	return NewArrayValue(t, false), nil
}

// arrayTypeAddFront implements prepend/prepend_in_place: whole-operand concatenation at the FRONT, arguments
// staying in order — x.prepend(a, b) ≡ a + b + x. Same purity split as arrayTypeAppend.
func arrayTypeAddFront(v Value, args []Value, mutate bool) (Value, error) {
	o := (*Array)(v.Ptr)
	items := arrayAddItems(args)

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotMutableError("prepend_in_place", v.TypeName())
		}
		// slices.Insert reuses the receiver's backing array whenever capacity allows
		o.Set(slices.Insert(o.Elements, 0, items...))
		return v, nil
	}

	t := make([]Value, 0, len(items)+len(o.Elements))
	t = append(t, items...)
	t = append(t, o.Elements...)
	return NewArrayValue(t, false), nil
}

// arrayTypePush implements push/push_first and their _in_place twins: each argument is ONE element whatever its
// type — the spelling that never spreads, so a.push(x) ⟹ a.last() == x and a.push_first(x) ⟹ a.first() == x,
// including when x is itself an array or a range. Arguments stay in order at the front too. Same purity split
// as arrayTypeAppend.
func arrayTypePush(v Value, args []Value, mutate bool, front bool) (Value, error) {
	o := (*Array)(v.Ptr)

	if mutate {
		if v.Immutable {
			name := "push_in_place"
			if front {
				name = "push_first_in_place"
			}
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		if front {
			o.Set(slices.Insert(o.Elements, 0, args...))
		} else {
			o.Set(append(o.Elements, args...))
		}
		return v, nil
	}

	t := make([]Value, 0, len(o.Elements)+len(args))
	if front {
		t = append(t, args...)
		t = append(t, o.Elements...)
	} else {
		t = append(t, o.Elements...)
		t = append(t, args...)
	}
	return NewArrayValue(t, false), nil
}

// arraySpliceItems is array's SeqSplice convertItems function: splice's inserts take the ADD-SIDE reading,
// like every other content-adding slot — an item of the receiver's own family (array or range) contributes
// its elements as a run, anything else is one element. The element spelling for a nested array is the wrap
// ([row]) or insert, which never spreads.
func arraySpliceItems(args []Value, _ string) ([]Value, error) {
	return arrayAddItems(args), nil
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
		return Undefined, errs.NewNotMutableError("sort_in_place", v.TypeName())
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
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("min", "0 or 1", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return emptySeqResult("min", args)
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
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("max", "0 or 1", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return emptySeqResult("max", args)
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
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("sum", "0 or 1", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return emptySeqResult("sum", args)
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
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("avg", "0 or 1", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return emptySeqResult("avg", args)
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

	case value.Bytes:
		s, err := joinElementsToString(elems, string((*Bytes)(sep.Ptr).Elements))
		if err != nil {
			return Undefined, err
		}
		return NewBytesValue([]byte(s), false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string, runes, bytes, byte, or rune", sep.TypeName())
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
