package core

import (
	"slices"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// array's members: one function per entry of TypeArray.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf, memberSlice in tools.go).

// PURE by contract
func arrayCopy(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return arrayTypeCopy(v, true)
}

// PURE by contract
func arrayCopyShallow(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return arrayTypeCopy(v, false)
}

// PURE by contract
func arrayFreezeShallow(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v.ToImmutable()
}

// PURE by contract
func arrayFreeze(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v.Freeze()
}

// PURE by contract
func arrayArray(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// a conversion CONSTRUCTS, on its own type like any other: a new, independent, mutable
	// shallow copy, exactly array(a) / a.copy_shallow(). Never the receiver itself — an alias
	// handed out under a conversion spelling wrote through to the caller's array, and it was
	// invisible (is_view() reports borrowing, and this was not one). Sharing is slice_view's job.
	c, err := arrayTypeCopy(v, false)
	if err != nil {
		return Undefined, err
	}
	return convMember(id.String(), arrayTypeName, args, true, c)
}

// PURE by contract
func arrayBytes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	// element-wise, all-or-nothing: a failing element fails the conversion —
	// the silent NUL/mod-256 corruption is gone
	bs, ok := ElementsToBytes(o.Elements)
	return convMember(id.String(), arrayTypeName, args, ok, NewBytesValue(bs, false))
}

// PURE by contract
func arrayString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	// the element step is the rune conversion (string and runes are one text),
	// so ["a","b"].string() raises — join() is the concatenative spelling
	rs, ok := ElementsToRunes(o.Elements)
	return convMember(id.String(), arrayTypeName, args, ok, NewStringValue(string(rs)))
}

// PURE by contract
func arrayRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	rs, ok := ElementsToRunes(o.Elements)
	return convMember(id.String(), arrayTypeName, args, ok, NewRunesValue(rs, false))
}

// PURE by contract
func arrayRecord(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	// the ENTRIES reading: each element is exactly a 2-element array [key, value];
	// the index->element decomposition (invented keys, scrambled order) is gone
	m, ok := ElementsToEntries(o.Elements)
	return convMember(id.String(), arrayTypeName, args, ok, NewRecordValue(m, false))
}

// PURE by contract
func arrayDict(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	m, ok := ElementsToEntries(o.Elements)
	return convMember(id.String(), arrayTypeName, args, ok, NewDictValue(m, false))
}

// PURE by contract
func arrayIsEmpty(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(len(o.Elements) == 0), nil
}

// PURE by contract
func arrayLen(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(len(o.Elements))), nil
}

// PURE by contract
func arrayFirst(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Array)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		// absence is data: undefined, or the optional trailing default
		return emptySeqResult(name, args)
	}
	return o.Elements[0], nil
}

// PURE by contract
func arrayLast(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Array)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		// absence is data: undefined, or the optional trailing default
		return emptySeqResult(name, args)
	}
	return o.Elements[len(o.Elements)-1], nil
}

// arrayContains is contains(...): is there a match anywhere? The empty run is contained everywhere, the
// same answer `in` gives.
// PURE by contract
func arrayContains(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayCount(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arrayKeep is keep(...): a new array of the matches.
// PURE by contract
func arrayKeep(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayKept(vm, "keep", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayKeepInPlace is keep_in_place(...): keep the matches in the receiver itself.
func arrayKeepInPlace(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arrayRemove is remove(...): a new array without the matches.
// PURE by contract
func arrayRemove(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayRemaining(vm, "remove", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayRemoveInPlace is remove_in_place(...): drop the matches from the receiver
// itself.
func arrayRemoveInPlace(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayAny(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayAll(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// PURE by contract
func arrayMin(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// PURE by contract
func arrayMax(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// PURE by contract
func arraySum(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// PURE by contract
func arrayAvg(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// PURE by contract
func arraySort(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return arrayFnSort(v, args, false)
}

// IMPURE by contract (mutates the receiver)
func arraySortInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return arrayFnSort(v, args, true)
}

// arrayDedup is dedup(): a new array with each run of equal neighbours collapsed to one.
// PURE by contract
func arrayDedup(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dedup", "0", len(args))
	}
	return NewArrayValue(arrayDeduped((*Array)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. arrayDedupInPlace is dedup_in_place(): dedup applied to the receiver itself.
func arrayDedupInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arrayUnique is unique(): a new array without repeats, each element kept at its first occurrence.
// PURE by contract
func arrayUnique(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}
	return NewArrayValue(arrayUniqueElements((*Array)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. arrayUniqueInPlace is unique_in_place(): unique applied to the receiver itself.
func arrayUniqueInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arrayFlatMap is flat_map(f): map, then concatenate — each callback result is read like an append operand: an
// array result spreads, undefined contributes nothing, anything else is one element.
// PURE by contract
func arrayFlatMap(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// PURE by contract
func arrayReverse(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	o := (*Array)(v.Ptr)
	n := len(o.Elements)
	t := make([]Value, n)
	for i, x := range o.Elements {
		t[n-1-i] = x
	}
	return NewArrayValue(t, false), nil
}

// IMPURE by contract (mutates the receiver)
func arrayReverseInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Array)(v.Ptr)
	slices.Reverse(o.Elements)
	return v, nil
}

// arrayMap is map(f): a new array of the callback results, 1:1.
// PURE by contract
func arrayMap(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arrayReduce is reduce(acc, f): folds the elements left to right.
// PURE by contract
func arrayReduce(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arrayForEach is for_each(f): a full pass whose callback result is ignored — early exit belongs to for/break or a
// search member. Returns the receiver, so it chains.
// PURE by contract
func arrayForEach(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arrayIndex is index(...): the first match.
// PURE by contract
func arrayIndex(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return arrayLocate(vm, "index", v, args, false)
}

// arrayIndexLast is index_last(...): the last match.
// PURE by contract
func arrayIndexLast(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return arrayLocate(vm, "index_last", v, args, true)
}

// arrayTrim is trim(...): without the leading and trailing elements of the set.
// PURE by contract
func arrayTrim(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayTrimmed("trim", v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayTrimInPlace is trim_in_place(...): trim(...) applied to the receiver
// itself.
func arrayTrimInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayTrimStart(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayTrimmed("trim_start", v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayTrimStartInPlace is trim_start_in_place(...): trim_start(...) applied to the receiver
// itself.
func arrayTrimStartInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayTrimEnd(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayTrimmed("trim_end", v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayTrimEndInPlace is trim_end_in_place(...): trim_end(...) applied to the receiver
// itself.
func arrayTrimEndInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arrayHasPrefix is has_prefix(...): does the receiver start with one of the runs?
// PURE by contract
func arrayHasPrefix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	best, err := arrayAnchoredRun("has_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// arrayHasSuffix is has_suffix(...): does the receiver end with one of the runs?
// PURE by contract
func arrayHasSuffix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	best, err := arrayAnchoredRun("has_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// arrayRemovePrefix is remove_prefix(...): without the longest matching prefix, once.
// PURE by contract
func arrayRemovePrefix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayWithoutAnchored("remove_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayRemovePrefixInPlace is remove_prefix_in_place(...): remove_prefix(...) applied to the receiver
// itself.
func arrayRemovePrefixInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayRemoveSuffix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayWithoutAnchored("remove_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayRemoveSuffixInPlace is remove_suffix_in_place(...): remove_suffix(...) applied to the receiver
// itself.
func arrayRemoveSuffixInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayReplace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayReplaced("replace", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayReplaceInPlace is replace_in_place(...): replace(...) applied to the receiver
// itself.
func arrayReplaceInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayPadStart(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayPadded("pad_start", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPadStartInPlace is pad_start_in_place(...): pad_start(...) applied to the receiver
// itself.
func arrayPadStartInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayPadEnd(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := arrayPadded("pad_end", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPadEndInPlace is pad_end_in_place(...): pad_end(...) applied to the receiver
// itself.
func arrayPadEndInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arrayChunk is chunk(size): the elements in arrays of size elements (the last one shorter), each an independent
// copy.
// PURE by contract
func arrayChunk(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arrayChunkView(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func arraySliceView(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// PURE by contract
func arrayAppend(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return arrayTypeAppend(v, args, false)
}

// IMPURE by contract (mutates the receiver)
func arrayAppendInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return arrayTypeAppend(v, args, true)
}

// arrayPrepend is prepend(...items): a new array with the items (read like append's operands) in front, in
// argument order — x.prepend(a, b) is a + b + x.
// PURE by contract
func arrayPrepend(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items := arrayAddItems(args)
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPrependInPlace is prepend_in_place(...items): prepend applied to the receiver
// itself.
func arrayPrependInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("prepend_in_place", v.TypeName())
	}
	o := (*Array)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, arrayAddItems(args)...))
	return v, nil
}

// arrayPush is push(...items): a new array with the items at the end, each ONE element whatever its type — the
// spelling that never spreads, so a.push(x).last() == x even when x is an array.
// PURE by contract
func arrayPush(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(elems)+len(args))
	out = append(out, elems...)
	out = append(out, args...)
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPushInPlace is push_in_place(...items): push applied to the receiver itself.
func arrayPushInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("push_in_place", v.TypeName())
	}
	o := (*Array)(v.Ptr)
	o.Set(append(o.Elements, args...))
	return v, nil
}

// arrayPushFirst is push_first(...items): a new array with the items in front, in argument order, each ONE
// element whatever its type.
// PURE by contract
func arrayPushFirst(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(args)+len(elems))
	out = append(out, args...)
	out = append(out, elems...)
	return NewArrayValue(out, false), nil
}

// IMPURE: mutates the receiver. arrayPushFirstInPlace is push_first_in_place(...items): push_first applied to the
// receiver itself.
func arrayPushFirstInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("push_first_in_place", v.TypeName())
	}
	o := (*Array)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, args...))
	return v, nil
}

// arrayInsert is insert(i, ...items): a new array with the items — each ONE element whatever its type, never
// spread — at position i, which raises out of [0, len].
// PURE by contract
func arrayInsert(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	at, err := readEditPos("insert", args, len(elems))
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(slices.Insert(slices.Clone(elems), at, args[1:]...), false), nil
}

// IMPURE: mutates the receiver. arrayInsertInPlace is insert_in_place(i, ...items): insert applied to the receiver
// itself.
func arrayInsertInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// arraySplice is splice([start[, count[, ...items]]]): a new array with count elements at start replaced by the
// items (read like append's operands).
// PURE by contract
func arraySplice(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
func arraySpliceInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// PURE by contract
func arrayRepeat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Array)(v.Ptr)
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
}

// arrayJoin implements `array.join(sep)`.
// sep types: string | runes | byte | rune.
// Result type follows sep: string→string, runes→runes, byte→bytes, rune→runes.
// With no argument, defaults to empty string separator.
// PURE by contract
func arrayJoin(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// PURE by contract
func arrayFlatten(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
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
