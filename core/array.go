package core

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unsafe"

	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
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

// Array is an array's body: its elements.
type Array struct {
	Elements []Value
	// IsView reports whether Elements shares backing storage with another value. Only the _view members
	// (slice_view, chunk_view) set it; every other constructor answers an independently-owned body.
	IsView bool
}

// Set replaces the elements — how every _in_place member writes its result back.
func (o *Array) Set(elements []Value) {
	o.Elements = elements
}

func NewArrayValue(arr []Value, immutable bool) Value {
	o := &Array{}
	o.Set(arr)
	return Value{Type: value.Array, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeArray = ValueTypeDescr{
	Name:         MutabilityNameHook(arrayTypeName, immutableArrayTypeName),                     // PURE by contract
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
	Contains:     arrayTypeContains,                                                             // PURE by contract
	Append:       arrayTypeAppend,                                                               // MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append)
	Slice:        arrayTypeSlice,                                                                // PURE by contract
	SliceStep:    arrayTypeSliceStep,                                                            // PURE by contract

	CallNamedMethod:     arrayTypeCallNamedMethod, // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
	AccessIndex:         arrayTypeAccessIndex,     // PURE by contract
	AccessNamedProperty: noNamedProperty,          // PURE by contract
	AssignIndex:         arrayTypeAssignIndex,     // IMPURE by contract

	AsBool: func(v Value) (bool, bool) { return len((*Array)(v.Ptr).Elements) > 0, true }, // PURE by contract
	// No AsString hook: an array has no canonical text (its element-wise conversion is a transcoding
	// constructor, not a render), so a dict key, a join element, or any implicit to-string consumer raises
	// instead of silently keying/rendering the transcode. The explicit conversions stay: .string(), string(a).
	AsRunes: arrayTypeAsRunes,                                                        // PURE by contract
	AsBytes: arrayTypeAsBytes,                                                        // PURE by contract
	AsArray: func(v Value) ([]Value, bool) { return (*Array)(v.Ptr).Elements, true }, // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order methods
	// (keep/map/reduce/for_each/all/any/find/count) are pure in isolation — impurity can only enter via a
	// function-valued argument.
	IsNamedMethodPure: func(name string) bool { return !strings.HasSuffix(name, "_in_place") },
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
		if op != tokens.Add {
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
	case tokens.Add:
		// exactly append's reading: an operand of the receiver's OWN KIND — another array —
		// contributes its elements as a run; anything else is one element
		items := arrayAddItems([]Value{other})
		t := make([]Value, 0, len(l.Elements)+len(items))
		t = append(t, l.Elements...)
		t = append(t, items...)
		return NewArrayValue(t, false), nil

	case tokens.Mul:
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

	case tokens.Sub:
		// exactly remove's value readings: an array operand removes every occurrence of
		// the contiguous run (never set difference), anything else every equal element
		switch other.Type {
		case value.Array:
			runs := [][]Value{(*Array)(other.Ptr).Elements}
			kept := make([]Value, 0, len(l.Elements))
			for i := 0; i < len(l.Elements); {
				if k := arrayRunLengthAt(l.Elements, i, runs); k > 0 {
					i += k
				} else {
					kept = append(kept, l.Elements[i])
					i++
				}
			}
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

// PURE by contract
func arrayTypeAccessIndex(v Value, index Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	i, err := resolveIndex("index access", index, len(elems))
	if err != nil {
		return Undefined, err
	}
	return (elems[i]), nil
}

// IMPURE by contract: writes into the receiver. Not folded by the optimizer. See docs/purity.md.
func arrayTypeAssignIndex(v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}
	elems := (*Array)(v.Ptr).Elements
	i, err := resolveIndex("index assign", index, len(elems))
	if err != nil {
		return err
	}
	elems[i] = r
	return nil
}

// PURE by contract. a[i:j] always answers an independently-owned copy; sharing is slice_view's job.
func arrayTypeSlice(v Value, s Value, e Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	si, ei, err := resolveSliceBounds("slice", s, e, len(elems))
	if err != nil {
		return Undefined, err
	}
	out := make([]Value, ei-si)
	copy(out, elems[si:ei])
	return NewArrayValue(out, false), nil
}

// PURE by contract
func arrayTypeSliceStep(v Value, s Value, e Value, step Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	start, end, st, err := resolveSliceStep(s, e, step, len(elems))
	if err != nil {
		return Undefined, err
	}
	out := make([]Value, 0)
	if st > 0 {
		for i := start; i < end; i += st {
			out = append(out, elems[i])
		}
	} else {
		for i := start; i > end; i += st {
			out = append(out, elems[i])
		}
	}
	return NewArrayValue(out, false), nil
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
func arrayTypeCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
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
		return memberFormat(vm, v, members.Format, args)

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

	case "contains":
		return arrayContainsMember(vm, v, args)

	case "count":
		return arrayCount(vm, v, args)

	case "keep":
		return arrayKeep(vm, v, args)

	case "keep_in_place":
		return arrayKeepInPlace(vm, v, args)

	case "remove":
		return arrayRemove(vm, v, args)

	case "remove_in_place":
		return arrayRemoveInPlace(vm, v, args)

	case "any":
		return arrayAny(vm, v, args)

	case "all":
		return arrayAll(vm, v, args)

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
		return arrayDedup(vm, v, args)

	case "dedup_in_place":
		return arrayDedupInPlace(vm, v, args)

	case "unique":
		return arrayUnique(vm, v, args)

	case "unique_in_place":
		return arrayUniqueInPlace(vm, v, args)

	case "flat_map":
		return arrayFlatMap(vm, v, args)

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
		return arrayMap(vm, v, args)

	case "reduce":
		return arrayReduce(vm, v, args)

	case "for_each":
		return arrayForEach(vm, v, args)

	case "index":
		return arrayIndex(vm, v, args)

	case "index_last":
		return arrayIndexLast(vm, v, args)

	case "trim":
		return arrayTrim(vm, v, args)

	case "trim_in_place":
		return arrayTrimInPlace(vm, v, args)

	case "trim_start":
		return arrayTrimStart(vm, v, args)

	case "trim_start_in_place":
		return arrayTrimStartInPlace(vm, v, args)

	case "trim_end":
		return arrayTrimEnd(vm, v, args)

	case "trim_end_in_place":
		return arrayTrimEndInPlace(vm, v, args)

	case "has_prefix":
		return arrayHasPrefix(vm, v, args)

	case "has_suffix":
		return arrayHasSuffix(vm, v, args)

	case "remove_prefix":
		return arrayRemovePrefix(vm, v, args)

	case "remove_prefix_in_place":
		return arrayRemovePrefixInPlace(vm, v, args)

	case "remove_suffix":
		return arrayRemoveSuffix(vm, v, args)

	case "remove_suffix_in_place":
		return arrayRemoveSuffixInPlace(vm, v, args)

	case "replace":
		return arrayReplace(vm, v, args)

	case "replace_in_place":
		return arrayReplaceInPlace(vm, v, args)

	case "pad_start":
		return arrayPadStart(vm, v, args)

	case "pad_start_in_place":
		return arrayPadStartInPlace(vm, v, args)

	case "pad_end":
		return arrayPadEnd(vm, v, args)

	case "pad_end_in_place":
		return arrayPadEndInPlace(vm, v, args)

	case "chunk":
		return arrayChunk(vm, v, args)

	case "chunk_view":
		return arrayChunkView(vm, v, args)

	case "slice":
		return sliceMember(v, args)

	case "slice_view":
		return arraySliceView(vm, v, args)

	case "append":
		return arrayTypeAppend(v, args, false)

	case "append_in_place":
		return arrayTypeAppend(v, args, true)

	case "prepend":
		return arrayPrepend(vm, v, args)

	case "prepend_in_place":
		return arrayPrependInPlace(vm, v, args)

	case "push":
		return arrayPush(vm, v, args)

	case "push_in_place":
		return arrayPushInPlace(vm, v, args)

	case "push_first":
		return arrayPushFirst(vm, v, args)

	case "push_first_in_place":
		return arrayPushFirstInPlace(vm, v, args)

	case "insert":
		return arrayInsert(vm, v, args)

	case "insert_in_place":
		return arrayInsertInPlace(vm, v, args)

	case "splice":
		return arraySplice(vm, v, args)

	case "splice_in_place":
		return arraySpliceInPlace(vm, v, args)

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
		return CallMemberByLookup(vm, v, name, args)
	}
}

// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all.
//
// One name per operation; the ARGUMENT'S TYPE selects the reading:
//   - no argument      the significant elements (not IsBlankElement)
//   - a function       a predicate, f/1(element) or f/2(index, element)
//   - an array         a contiguous run (not for any/all)
//   - anything else    one element, compared with ==
//
// Several arguments form a SET of one reading (all elements, or all runs); mixing the two, or putting a function
// among them, raises. Runs match leftmost-longest and never overlap.
// ---------------------------------------------------------------------------

// arrayMatch is a match member's argument list after reading; exactly one reading is set.
type arrayMatch struct {
	significant bool      // no argument
	pred        Value     // a predicate
	elems       []Value   // a set of elements
	runs        [][]Value // a set of runs
}

// arrayReadMatchArgs reads a match member's arguments. allowRuns is false for any/all: "some element is this
// subsequence" is contains' question, not theirs.
func arrayReadMatchArgs(name string, args []Value, allowRuns bool) (arrayMatch, error) {
	if len(args) == 0 {
		return arrayMatch{significant: true}, nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return arrayMatch{}, errPredicateAmongMany(name)
		}
		if err := checkElemCallback(name, args[0]); err != nil {
			return arrayMatch{}, err
		}
		return arrayMatch{pred: args[0]}, nil
	}

	var m arrayMatch
	firstRun := -1
	for i, a := range args {
		switch {
		case a.IsCallable():
			return arrayMatch{}, errFunctionInSet(name)
		case a.Type == value.Array:
			if firstRun < 0 {
				firstRun = i
			}
			m.runs = append(m.runs, (*Array)(a.Ptr).Elements)
		default:
			m.elems = append(m.elems, a)
		}
	}
	if m.runs != nil && m.elems != nil {
		return arrayMatch{}, errMixedSet(name)
	}
	if m.runs != nil && !allowRuns {
		return arrayMatch{}, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a predicate (this member declares no run reading)", args[firstRun].TypeName())
	}
	return m, nil
}

// matches reports whether element e at index i matches an element-wise reading (anything but runs).
func (m *arrayMatch) matches(vm VM, i int, e Value) (bool, error) {
	switch {
	case m.significant:
		return !IsBlankElement(e), nil
	case m.pred.IsCallable():
		res, err := callElem(vm, m.pred, i, e)
		if err != nil {
			return false, err
		}
		return res.IsTrue()
	}
	for _, x := range m.elems {
		if e.Equal(x) {
			return true, nil
		}
	}
	return false, nil
}

// arrayRunLengthAt reports the length of the longest non-empty run in runs that matches elems at position i
// (0 when none does) — runLengthAt with Value equality.
func arrayRunLengthAt(elems []Value, i int, runs [][]Value) int {
	best := 0
	for _, r := range runs {
		if len(r) <= best || i+len(r) > len(elems) {
			continue
		}
		if arrayEqualRun(elems[i:i+len(r)], r) {
			best = len(r)
		}
	}
	return best
}

// arrayContainsMember is contains(...): is there a match anywhere? The empty run is contained everywhere, the
// same answer `in` gives.
func arrayContainsMember(vm VM, v Value, args []Value) (Value, error) {
	m, err := arrayReadMatchArgs("contains", args, true)
	if err != nil {
		return Undefined, err
	}
	elems := (*Array)(v.Ptr).Elements

	if m.runs != nil {
		for _, r := range m.runs {
			if len(r) == 0 {
				return True, nil
			}
		}
		for i := range elems {
			if arrayRunLengthAt(elems, i, m.runs) > 0 {
				return True, nil
			}
		}
		return False, nil
	}

	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// arrayCount is count(...): how many matches. Runs count non-overlapping occurrences.
func arrayCount(vm VM, v Value, args []Value) (Value, error) {
	m, err := arrayReadMatchArgs("count", args, true)
	if err != nil {
		return Undefined, err
	}
	elems := (*Array)(v.Ptr).Elements

	n := int64(0)
	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := arrayRunLengthAt(elems, i, m.runs); k > 0 {
				n++
				i += k
			} else {
				i++
			}
		}
		return IntValue(n), nil
	}

	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if hit {
			n++
		}
	}
	return IntValue(n), nil
}

// arrayKept answers the elements keep(...) keeps: the matches, in order. Shared by keep and keep_in_place; name
// is the member called, for the errors.
func arrayKept(vm VM, name string, v Value, args []Value) ([]Value, error) {
	m, err := arrayReadMatchArgs(name, args, true)
	if err != nil {
		return nil, err
	}
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(elems))

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := arrayRunLengthAt(elems, i, m.runs); k > 0 {
				out = append(out, elems[i:i+k]...)
				i += k
			} else {
				i++
			}
		}
		return out, nil
	}

	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return nil, err
		}
		if hit {
			out = append(out, e)
		}
	}
	return out, nil
}

// arrayKeep is keep(...): a new array of the matches.
func arrayKeep(vm VM, v Value, args []Value) (Value, error) {
	out, err := arrayKept(vm, "keep", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayKeepInPlace is keep_in_place(...): keep the matches in the receiver itself.
func arrayKeepInPlace(vm VM, v Value, args []Value) (Value, error) {
	const name = "keep_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayKept(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayRemaining answers the elements remove(...) leaves: everything but the matches, in order. With no argument
// remove drops the BLANK elements — so it keeps the significant ones, landing on keep()'s answer by the opposite
// action. Shared by remove and remove_in_place; name is the member called, for the errors.
func arrayRemaining(vm VM, name string, v Value, args []Value) ([]Value, error) {
	m, err := arrayReadMatchArgs(name, args, true)
	if err != nil {
		return nil, err
	}
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(elems))

	if m.significant {
		for _, e := range elems {
			if !IsBlankElement(e) {
				out = append(out, e)
			}
		}
		return out, nil
	}

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := arrayRunLengthAt(elems, i, m.runs); k > 0 {
				i += k
			} else {
				out = append(out, elems[i])
				i++
			}
		}
		return out, nil
	}

	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return nil, err
		}
		if !hit {
			out = append(out, e)
		}
	}
	return out, nil
}

// arrayRemove is remove(...): a new array without the matches.
func arrayRemove(vm VM, v Value, args []Value) (Value, error) {
	out, err := arrayRemaining(vm, "remove", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayRemoveInPlace is remove_in_place(...): drop the matches from the receiver
// itself.
func arrayRemoveInPlace(vm VM, v Value, args []Value) (Value, error) {
	const name = "remove_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayRemaining(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayAny is any(...): does some element match? Elements, a predicate, or nothing — never a run.
func arrayAny(vm VM, v Value, args []Value) (Value, error) {
	m, err := arrayReadMatchArgs("any", args, false)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Array)(v.Ptr).Elements {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// arrayAll is all(...): does every element match? Elements, a predicate, or nothing — never a run. True on an
// empty array.
func arrayAll(vm VM, v Value, args []Value) (Value, error) {
	m, err := arrayReadMatchArgs("all", args, false)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Array)(v.Ptr).Elements {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if !hit {
			return False, nil
		}
	}
	return True, nil
}

// ---------------------------------------------------------------------------
// The locators: index([x[, default]]) / index_last([x[, default]]) — the position of the first / last match. The
// argument's type selects the reading:
//   - no argument      the first / last significant element (not IsBlankElement)
//   - a function       a predicate, f/1(element) or f/2(index, element)
//   - an array         a contiguous run
//   - anything else    one element, compared with == (an array can hold one of anything)
// A miss answers undefined, or the trailing default. Never variadic: the second slot is the default.
// ---------------------------------------------------------------------------

// arrayIndex is index(...): the first match.
func arrayIndex(vm VM, v Value, args []Value) (Value, error) {
	return arrayLocate(vm, "index", v, args, false)
}

// arrayIndexLast is index_last(...): the last match.
func arrayIndexLast(vm VM, v Value, args []Value) (Value, error) {
	return arrayLocate(vm, "index_last", v, args, true)
}

// arrayLocate is the body of index and index_last; name is the member called, for the errors.
func arrayLocate(vm VM, name string, v Value, args []Value, last bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	elems := (*Array)(v.Ptr).Elements

	if len(args) == 0 {
		idx := -1
		for i, e := range elems {
			if !IsBlankElement(e) {
				idx = i
				if !last {
					break
				}
			}
		}
		return locatorAnswer(idx, nil)
	}

	needle, dflt := args[0], args[1:]

	if needle.IsCallable() {
		if err := checkElemCallback(name, needle); err != nil {
			return Undefined, err
		}
		idx := -1
		for i, e := range elems {
			res, err := callElem(vm, needle, i, e)
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				idx = i
				if !last {
					break
				}
			}
		}
		return locatorAnswer(idx, dflt)
	}

	if needle.Type == value.Array {
		return locatorAnswer(arrayIndexRun(elems, (*Array)(needle.Ptr).Elements, last), dflt)
	}

	idx := -1
	for i, e := range elems {
		if e.Equal(needle) {
			idx = i
			if !last {
				break
			}
		}
	}
	return locatorAnswer(idx, dflt)
}

// arrayIndexRun is indexRun with Value equality.
func arrayIndexRun(elems []Value, run []Value, last bool) int {
	n, m := len(elems), len(run)
	if m == 0 || m > n {
		return -1
	}
	idx := -1
	for i := 0; i+m <= n; i++ {
		if arrayEqualRun(elems[i:i+m], run) {
			idx = i
			if !last {
				break
			}
		}
	}
	return idx
}

// ---------------------------------------------------------------------------
// The structural members: the trim family, the anchored pair (has_/remove_ prefix/suffix), replace, and the pads.
// They are sequence verbs, not text verbs: an array argument is a run, anything else is one element; none of
// them takes a predicate.
//   - trim*            a set of single elements, stripped repeat-while; no argument = the blank set (IsBlankElement)
//   - *_prefix/suffix  a homogeneous set of runs; the longest anchored run wins; remove_* removes it once
//   - replace(old, new) every occurrence of the run old, leftmost and non-overlapping; an empty old matches nothing
//   - pad_*(n[, fill]) n counts elements; the fill is any one value, undefined by default
// ---------------------------------------------------------------------------

// arrayReadElementSet reads the trim family's set of single elements: an array argument or a function raises with
// refuse, the member's own statement of what it takes.
func arrayReadElementSet(name string, args []Value, refuse string) ([]Value, error) {
	for _, a := range args {
		if a.IsCallable() || a.Type == value.Array {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", refuse, a.TypeName())
		}
	}
	return args, nil
}

// arrayReadRunSet reads the anchored pair's homogeneous set of runs: an array argument is a run, anything else a
// run of one element, but one call may not mix the two. A function raises with onFunction.
func arrayReadRunSet(name string, args []Value, onFunction string) ([][]Value, error) {
	runs := make([][]Value, 0, len(args))
	sawElement, sawRun := false, false
	for _, a := range args {
		switch {
		case a.IsCallable():
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", onFunction, a.TypeName())
		case a.Type == value.Array:
			sawRun = true
			runs = append(runs, (*Array)(a.Ptr).Elements)
		default:
			sawElement = true
			runs = append(runs, []Value{a})
		}
	}
	if sawElement && sawRun {
		return nil, errMixedSet(name)
	}
	return runs, nil
}

// arrayEqualRun reports whether two runs hold equal elements, in order.
func arrayEqualRun(a, b []Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

// arrayTrimmed answers the receiver without its leading (start) and/or trailing (end) elements that belong to the
// set. The name is the member called, for the errors.
func arrayTrimmed(name string, v Value, args []Value, start, end bool) ([]Value, error) {
	set, err := arrayReadElementSet(name, args, "a set of elements (the anchored run form is remove_prefix/remove_suffix; no predicate reading)")
	if err != nil {
		return nil, err
	}
	inSet := func(e Value) bool {
		if len(args) == 0 {
			return IsBlankElement(e)
		}
		for _, x := range set {
			if e.Equal(x) {
				return true
			}
		}
		return false
	}
	elems := (*Array)(v.Ptr).Elements
	lo, hi := 0, len(elems)
	if start {
		for lo < hi && inSet(elems[lo]) {
			lo++
		}
	}
	if end {
		for hi > lo && inSet(elems[hi-1]) {
			hi--
		}
	}
	return slices.Clone(elems[lo:hi]), nil
}

// arrayAnchoredRun answers the length of the longest run of the argument set found at the start (or, with suffix,
// the end) of the receiver, or -1 when none is. The empty run is anchored everywhere. The name is the member
// called, for the errors.
func arrayAnchoredRun(name string, v Value, args []Value, suffix bool) (int, error) {
	if len(args) == 0 {
		return 0, errs.NewWrongNumArgumentsError(name, "1 or more", 0)
	}
	runs, err := arrayReadRunSet(name, args, "an element or a run (no predicate reading — \"the first element satisfies f\" is index(f) == 0)")
	if err != nil {
		return 0, err
	}
	elems := (*Array)(v.Ptr).Elements
	best := -1
	for _, r := range runs {
		if len(r) > len(elems) || len(r) <= best {
			continue
		}
		at := elems[:len(r)]
		if suffix {
			at = elems[len(elems)-len(r):]
		}
		if arrayEqualRun(at, r) {
			best = len(r)
		}
	}
	return best, nil
}

// arrayWithoutAnchored answers the receiver without its longest matching prefix (or suffix) — removed once;
// unchanged when nothing matches. The name is the member called, for the errors.
func arrayWithoutAnchored(name string, v Value, args []Value, suffix bool) ([]Value, error) {
	best, err := arrayAnchoredRun(name, v, args, suffix)
	if err != nil {
		return nil, err
	}
	elems := (*Array)(v.Ptr).Elements
	switch {
	case best <= 0:
		return slices.Clone(elems), nil
	case suffix:
		return slices.Clone(elems[:len(elems)-best]), nil
	default:
		return slices.Clone(elems[best:]), nil
	}
}

// arrayReplaced answers the receiver with every occurrence of the run old (args[0]) replaced by new (args[1]); an
// array argument is a run, anything else a run of one element. The name is the member called, for the errors.
func arrayReplaced(name string, v Value, args []Value) ([]Value, error) {
	if len(args) != 2 {
		return nil, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	if args[0].IsCallable() || args[1].IsCallable() {
		return nil, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a run (replace is never a predicate)", "function")
	}
	asRun := func(a Value) []Value {
		if a.Type == value.Array {
			return (*Array)(a.Ptr).Elements
		}
		return []Value{a}
	}
	old, repl := asRun(args[0]), asRun(args[1])
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(elems))
	for i := 0; i < len(elems); {
		if len(old) > 0 && i+len(old) <= len(elems) && arrayEqualRun(elems[i:i+len(old)], old) {
			out = append(out, repl...)
			i += len(old)
		} else {
			out = append(out, elems[i])
			i++
		}
	}
	return out, nil
}

// arrayPadded answers the receiver filled at the front (start) or the end up to n elements; a width at or below
// the length leaves it unchanged. The fill is any one value. The name is the member called, for the errors.
func arrayPadded(name string, v Value, args []Value, start bool) ([]Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, errs.NewWrongNumArgumentsError(name, "1 or 2", len(args))
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return nil, err
	}
	fill := Undefined
	if len(args) == 2 {
		fill = args[1]
	}
	elems := (*Array)(v.Ptr).Elements
	if n <= int64(len(elems)) {
		return slices.Clone(elems), nil
	}
	width, err := SeqPadWidth(name, n)
	if err != nil {
		return nil, err
	}
	pad := slices.Repeat([]Value{fill}, width-len(elems))
	if start {
		return slices.Concat(pad, elems), nil
	}
	return slices.Concat(elems, pad), nil
}

// arrayTrim is trim(...): without the leading and trailing elements of the set.
func arrayTrim(_ VM, v Value, args []Value) (Value, error) {
	out, err := arrayTrimmed("trim", v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayTrimInPlace is trim_in_place(...): trim(...) applied to the receiver
// itself.
func arrayTrimInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "trim_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayTrimmed(name, v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayTrimStart is trim_start(...): without the leading elements of the set.
func arrayTrimStart(_ VM, v Value, args []Value) (Value, error) {
	out, err := arrayTrimmed("trim_start", v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayTrimStartInPlace is trim_start_in_place(...): trim_start(...) applied to the receiver
// itself.
func arrayTrimStartInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "trim_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayTrimmed(name, v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayTrimEnd is trim_end(...): without the trailing elements of the set.
func arrayTrimEnd(_ VM, v Value, args []Value) (Value, error) {
	out, err := arrayTrimmed("trim_end", v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayTrimEndInPlace is trim_end_in_place(...): trim_end(...) applied to the receiver
// itself.
func arrayTrimEndInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "trim_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayTrimmed(name, v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayRemovePrefix is remove_prefix(...): without the longest matching prefix, once.
func arrayRemovePrefix(_ VM, v Value, args []Value) (Value, error) {
	out, err := arrayWithoutAnchored("remove_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayRemovePrefixInPlace is remove_prefix_in_place(...): remove_prefix(...) applied to the receiver
// itself.
func arrayRemovePrefixInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "remove_prefix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayWithoutAnchored(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayRemoveSuffix is remove_suffix(...): without the longest matching suffix, once.
func arrayRemoveSuffix(_ VM, v Value, args []Value) (Value, error) {
	out, err := arrayWithoutAnchored("remove_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayRemoveSuffixInPlace is remove_suffix_in_place(...): remove_suffix(...) applied to the receiver
// itself.
func arrayRemoveSuffixInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "remove_suffix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayWithoutAnchored(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayReplace is replace(...): every occurrence of old replaced by new.
func arrayReplace(_ VM, v Value, args []Value) (Value, error) {
	out, err := arrayReplaced("replace", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayReplaceInPlace is replace_in_place(...): replace(...) applied to the receiver
// itself.
func arrayReplaceInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "replace_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayReplaced(name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayPadStart is pad_start(...): filled at the front up to n elements.
func arrayPadStart(_ VM, v Value, args []Value) (Value, error) {
	out, err := arrayPadded("pad_start", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPadStartInPlace is pad_start_in_place(...): pad_start(...) applied to the receiver
// itself.
func arrayPadStartInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "pad_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayPadded(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayPadEnd is pad_end(...): filled at the end up to n elements.
func arrayPadEnd(_ VM, v Value, args []Value) (Value, error) {
	out, err := arrayPadded("pad_end", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPadEndInPlace is pad_end_in_place(...): pad_end(...) applied to the receiver
// itself.
func arrayPadEndInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "pad_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := arrayPadded(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Array)(v.Ptr).Set(out)
	return v, nil
}

// arrayHasPrefix is has_prefix(...): does the receiver start with one of the runs?
func arrayHasPrefix(_ VM, v Value, args []Value) (Value, error) {
	best, err := arrayAnchoredRun("has_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// arrayHasSuffix is has_suffix(...): does the receiver end with one of the runs?
func arrayHasSuffix(_ VM, v Value, args []Value) (Value, error) {
	best, err := arrayAnchoredRun("has_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// ---------------------------------------------------------------------------
// The callback members: for_each, map, flat_map, reduce. A per-element callback is f/1(element) or
// f/2(index, element); reduce's is f/2(acc, element) or f/3(acc, index, element).
// ---------------------------------------------------------------------------

// arrayForEach is for_each(f): a full pass whose callback result is ignored — early exit belongs to for/break or a
// search member. Returns the receiver, so it chains.
func arrayForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Array)(v.Ptr).Elements {
		if _, err := callElem(vm, fn, i, e); err != nil {
			return Undefined, err
		}
	}
	return v, nil
}

// arrayMap is map(f): a new array of the callback results, 1:1.
func arrayMap(vm VM, v Value, args []Value) (Value, error) {
	fn, err := readElemCallback("map", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, len(elems))
	for i, e := range elems {
		out[i], err = callElem(vm, fn, i, e)
		if err != nil {
			return Undefined, err
		}
	}
	return NewArrayValue(out, false), nil
}

// arrayFlatMap is flat_map(f): map, then concatenate — each callback result is read like an append operand: an
// array result spreads, undefined contributes nothing, anything else is one element.
func arrayFlatMap(vm VM, v Value, args []Value) (Value, error) {
	fn, err := readElemCallback("flat_map", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(elems))
	for i, e := range elems {
		res, err := callElem(vm, fn, i, e)
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
}

// arrayReduce is reduce(acc, f): folds the elements left to right.
func arrayReduce(vm VM, v Value, args []Value) (Value, error) {
	acc, fn, err := readReduceArgs(args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Array)(v.Ptr).Elements {
		acc, err = callReduce(vm, fn, acc, i, e)
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
}

// ---------------------------------------------------------------------------
// The add and edit members: chunk, slice_view, splice, insert, prepend, push, push_first (append is the Append
// hook, arrayTypeAppend; slice is the Slice hook's member spelling). append/prepend/splice's inserts read like
// the + operator (an array operand spreads, see arrayAddItems); push/push_first/insert never spread.
// ---------------------------------------------------------------------------

// arrayChunk is chunk(size): the elements in arrays of size elements (the last one shorter), each an independent
// copy.
func arrayChunk(_ VM, v Value, args []Value) (Value, error) {
	size, err := readChunkSize("chunk", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Array)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		chunks = append(chunks, NewArrayValue(slices.Clone(elems[start:min(start+step, len(elems))]), false))
	}
	return NewArrayValue(chunks, false), nil
}

// arrayChunkView is chunk_view(size): chunk's pieces sharing the receiver's storage, each marked as a view.
func arrayChunkView(_ VM, v Value, args []Value) (Value, error) {
	size, err := readChunkSize("chunk_view", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Array)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		c := NewArrayValue(elems[start:min(start+step, len(elems))], v.Immutable)
		(*Array)(c.Ptr).IsView = true
		chunks = append(chunks, c)
	}
	return NewArrayValue(chunks, false), nil
}

// arraySliceView is slice_view([start[, end]]): the span sharing the receiver's storage, marked as a view.
func arraySliceView(_ VM, v Value, args []Value) (Value, error) {
	const name = "slice_view"
	s, e, err := readSliceArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Array)(v.Ptr).Elements
	si, ei, err := resolveSliceBounds(name, s, e, len(elems))
	if err != nil {
		return Undefined, err
	}
	res := NewArrayValue(elems[si:ei], v.Immutable)
	(*Array)(res.Ptr).IsView = true
	return res, nil
}

// arraySplice is splice([start[, count[, ...items]]]): a new array with count elements at start replaced by the
// items (read like append's operands).
func arraySplice(_ VM, v Value, args []Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	start, end, err := readSpliceRange(args, len(elems))
	if err != nil {
		return Undefined, err
	}
	var items []Value
	if len(args) > 2 {
		items = arrayAddItems(args[2:])
	}
	out := make([]Value, 0, start+len(items)+len(elems)-end)
	out = append(out, elems[:start]...)
	out = append(out, items...)
	out = append(out, elems[end:]...)
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arraySpliceInPlace is splice_in_place(...): splice applied to the receiver itself.
// A side-effecting member returns the RECEIVER, so mutators chain; the removed span is x.slice(i, j) taken
// beforehand.
func arraySpliceInPlace(_ VM, v Value, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("splice_in_place", v.TypeName())
	}
	o := (*Array)(v.Ptr)
	start, end, err := readSpliceRange(args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	var items []Value
	if len(args) > 2 {
		items = arrayAddItems(args[2:])
	}
	o.Set(append(o.Elements[:start], append(items, o.Elements[end:]...)...))
	return v, nil
}

// arrayInsert is insert(i, ...items): a new array with the items — each ONE element whatever its type, never
// spread — at position i, which raises out of [0, len].
func arrayInsert(_ VM, v Value, args []Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	at, err := readEditPos("insert", args, len(elems))
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(slices.Insert(slices.Clone(elems), at, args[1:]...), false), nil
}

// IMPURE: mutates the receiver. arrayInsertInPlace is insert_in_place(i, ...items): insert applied to the receiver
// itself.
func arrayInsertInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "insert_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Array)(v.Ptr)
	at, err := readEditPos(name, args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	o.Set(slices.Insert(o.Elements, at, args[1:]...))
	return v, nil
}

// arrayPrepend is prepend(...items): a new array with the items (read like append's operands) in front, in
// argument order — x.prepend(a, b) is a + b + x.
func arrayPrepend(_ VM, v Value, args []Value) (Value, error) {
	items := arrayAddItems(args)
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPrependInPlace is prepend_in_place(...items): prepend applied to the receiver
// itself.
func arrayPrependInPlace(_ VM, v Value, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("prepend_in_place", v.TypeName())
	}
	o := (*Array)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, arrayAddItems(args)...))
	return v, nil
}

// arrayPush is push(...items): a new array with the items at the end, each ONE element whatever its type — the
// spelling that never spreads, so a.push(x).last() == x even when x is an array.
func arrayPush(_ VM, v Value, args []Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(elems)+len(args))
	out = append(out, elems...)
	out = append(out, args...)
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPushInPlace is push_in_place(...items): push applied to the receiver itself.
func arrayPushInPlace(_ VM, v Value, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("push_in_place", v.TypeName())
	}
	o := (*Array)(v.Ptr)
	o.Set(append(o.Elements, args...))
	return v, nil
}

// arrayPushFirst is push_first(...items): a new array with the items in front, in argument order, each ONE
// element whatever its type.
func arrayPushFirst(_ VM, v Value, args []Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(args)+len(elems))
	out = append(out, args...)
	out = append(out, elems...)
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPushFirstInPlace is push_first_in_place(...items): push_first applied to the
// receiver itself.
func arrayPushFirstInPlace(_ VM, v Value, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("push_first_in_place", v.TypeName())
	}
	o := (*Array)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, args...))
	return v, nil
}

// arrayDeduped answers the elements with each run of equal neighbours collapsed to one.
func arrayDeduped(elems []Value) []Value {
	out := make([]Value, 0, len(elems))
	for i, e := range elems {
		if i == 0 || !out[len(out)-1].Equal(e) {
			out = append(out, e)
		}
	}
	return out
}

// arrayDedup is dedup(): a new array with each run of equal neighbours collapsed to one.
func arrayDedup(_ VM, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dedup", "0", len(args))
	}
	return NewArrayValue(arrayDeduped((*Array)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. arrayDedupInPlace is dedup_in_place(): dedup applied to the receiver itself.
func arrayDedupInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "dedup_in_place"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Array)(v.Ptr)
	o.Set(arrayDeduped(o.Elements))
	return v, nil
}

// arrayUniqueElements answers the elements without repeats, each kept at its first occurrence.
func arrayUniqueElements(elems []Value) []Value {
	out := make([]Value, 0, len(elems))
	for _, e := range elems {
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
	return out
}

// arrayUnique is unique(): a new array without repeats, each element kept at its first occurrence.
func arrayUnique(_ VM, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}
	return NewArrayValue(arrayUniqueElements((*Array)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. arrayUniqueInPlace is unique_in_place(): unique applied to the receiver itself.
func arrayUniqueInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "unique_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}
	o := (*Array)(v.Ptr)
	o.Set(arrayUniqueElements(o.Elements))
	return v, nil
}

// arrayTypeContains is the `in` operator: contains' VALUE readings — an operand of the receiver's own
// FAMILY (array or range) is a contiguous run, anything else one element; the empty run is contained
// everywhere. A callable raises — an operator operand is always a value.
func arrayTypeContains(v Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	o := (*Array)(v.Ptr)
	switch e.Type {
	case value.Array:
		runs := [][]Value{(*Array)(e.Ptr).Elements}
		if len(runs[0]) == 0 {
			return true, nil
		}
		for i := range o.Elements {
			if arrayRunLengthAt(o.Elements, i, runs) > 0 {
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
		less, e := x.BinaryOp(tokens.Less, y)
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
		less, err := o.Elements[i].BinaryOp(tokens.Less, e)
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
		greater, err := o.Elements[i].BinaryOp(tokens.Greater, e)
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
		s, err = s.BinaryOp(tokens.Add, o.Elements[i])
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
		sum, err = sum.BinaryOp(tokens.Add, o.Elements[i])
		if err != nil {
			return Undefined, err
		}
	}

	length := IntValue(int64(len(o.Elements)))
	avg, err := sum.BinaryOp(tokens.Quo, length)
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
