package core

import (
	"slices"
	"unicode"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// runes's members: one function per entry of TypeRunes.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf, memberSlice in tools.go).

// PURE by contract
func runesCopy(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return runesTypeCopy(v, true)
}

// PURE by contract
func runesFreeze(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v.Freeze()
}

// PURE by contract
func runesRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the same-type conversion constructs — a new, independent, mutable copy, exactly
	// runes(r) / r.copy(); see the note on array's own case.
	c, err := runesTypeCopy(v, false)
	if err != nil {
		return Undefined, err
	}
	return convMember(id.String(), runesTypeName, args, true, c)
}

// PURE by contract
func runesString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(EncodeText(o.Elements)), nil
}

// PURE by contract
func runesArray(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	t, _ := runesTypeAsArray(v)
	return NewArrayValue(t, false), nil
}

// PURE by contract
func runesBool(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	b, ok := runesTypeAsBool(v)
	return convMember(id.String(), runesTypeName, args, ok, BoolValue(b))
}

// PURE by contract
func runesBytes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewBytesValue(EncodeOctets(o.Elements), false), nil
}

// PURE by contract
func runesFloat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f, ok := runesTypeAsFloat(v)
	return convMember(id.String(), runesTypeName, args, ok, FloatValue(f))
}

// PURE by contract
func runesInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	i, ok := runesTypeAsInt(v)
	return convMember(id.String(), runesTypeName, args, ok, IntValue(i))
}

// PURE by contract
func runesDecimal(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d, ok := runesTypeAsDecimal(v)
	return convMember(id.String(), runesTypeName, args, ok, NewDecimalValue(d))
}

// PURE by contract
func runesTime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return textTimeMember(id.String(), runesTypeName, EncodeText((*Runes)(v.Ptr).Elements), args)
}

// PURE by contract
func runesDate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return textDateMember(id.String(), runesTypeName, EncodeText((*Runes)(v.Ptr).Elements), args)
}

// PURE by contract
func runesIsValid(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	// no escapes anywhere: every element is a real symbol
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(RunesAreValid(o.Elements)), nil
}

// PURE by contract
func runesIsASCII(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(RunesAreASCII(o.Elements)), nil
}

// PURE by contract
func runesIsEmpty(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(len(o.Elements) == 0), nil
}

// PURE by contract
func runesLen(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(len(o.Elements))), nil
}

// PURE by contract
func runesFirst(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Runes)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		// absence is data: undefined, or the optional trailing default
		return emptySeqResult(name, args)
	}
	return RuneValue(o.Elements[0]), nil
}

// PURE by contract
func runesLast(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Runes)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		// absence is data: undefined, or the optional trailing default
		return emptySeqResult(name, args)
	}
	return RuneValue(o.Elements[len(o.Elements)-1]), nil
}

// PURE by contract
func runesMin(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Runes)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		// absence is data: undefined, or the optional trailing default
		return emptySeqResult(name, args)
	}
	return RuneValue(slices.Min(o.Elements)), nil
}

// PURE by contract
func runesMax(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Runes)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		// absence is data: undefined, or the optional trailing default
		return emptySeqResult(name, args)
	}
	return RuneValue(slices.Max(o.Elements)), nil
}

// PURE by contract
func runesLower(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	rs := make([]rune, len(o.Elements))
	for i, r := range o.Elements {
		rs[i] = unicode.ToLower(r)
	}
	return NewRunesValue(rs, false), nil
}

// PURE by contract
func runesUpper(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	rs := make([]rune, len(o.Elements))
	for i, r := range o.Elements {
		rs[i] = unicode.ToUpper(r)
	}
	return NewRunesValue(rs, false), nil
}

// runesContains is contains(...): is there a match anywhere? The empty run is contained everywhere, the
// same answer `in` gives.
// PURE by contract
func runesContains(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("contains", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements

	if m.runs != nil {
		for _, r := range m.runs {
			if len(r) == 0 {
				return True, nil
			}
		}
		for i := range elems {
			if runLengthAt(elems, i, m.runs) > 0 {
				return True, nil
			}
		}
		return False, nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// runesCount is count(...): how many matches. Runs count non-overlapping occurrences.
// PURE by contract
func runesCount(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("count", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements

	n := int64(0)
	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := runLengthAt(elems, i, m.runs); k > 0 {
				n++
				i += k
			} else {
				i++
			}
		}
		return IntValue(n), nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if hit {
			n++
		}
	}
	return IntValue(n), nil
}

// runesKeep is keep(...): new runes of the matches.
// PURE by contract
func runesKeep(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesKept(vm, "keep", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesKeepInPlace is keep_in_place(...): keep the matches in the receiver itself.
func runesKeepInPlace(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "keep_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesKept(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesRemove is remove(...): new runes without the matches.
// PURE by contract
func runesRemove(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesRemaining(vm, "remove", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesRemoveInPlace is remove_in_place(...): drop the matches from the receiver
// itself.
func runesRemoveInPlace(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "remove_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesRemaining(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesAny is any(...): does some symbol match? Symbols, a predicate, or nothing — never a run.
// PURE by contract
func runesAny(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("any", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range (*Runes)(v.Ptr).Elements {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// runesAll is all(...): does every symbol match? Symbols, a predicate, or nothing — never a run. True on empty
// runes.
// PURE by contract
func runesAll(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("all", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range (*Runes)(v.Ptr).Elements {
		hit, err := m.matches(vm, i, r)
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
func runesSort(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	sorted := make([]rune, len(o.Elements))
	copy(sorted, o.Elements)
	slices.Sort(sorted)
	return NewRunesValue(sorted, false), nil
}

// IMPURE by contract (mutates the receiver)
func runesSortInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	slices.Sort(o.Elements)
	return v, nil
}

// runesDedup is dedup(): a new runes with each run of equal neighbours collapsed to one.
// PURE by contract
func runesDedup(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dedup", "0", len(args))
	}
	return NewRunesValue(runesDeduped((*Runes)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. runesDedupInPlace is dedup_in_place(): dedup applied to the receiver itself.
func runesDedupInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "dedup_in_place"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(runesDeduped(o.Elements))
	return v, nil
}

// runesUnique is unique(): a new runes without repeats, each element kept at its first occurrence.
// PURE by contract
func runesUnique(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}
	return NewRunesValue(runesUniqueElements((*Runes)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. runesUniqueInPlace is unique_in_place(): unique applied to the receiver itself.
func runesUniqueInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "unique_in_place"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(runesUniqueElements(o.Elements))
	return v, nil
}

// PURE by contract
func runesReverse(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	n := len(o.Elements)
	rev := make([]rune, n)
	for i, r := range o.Elements {
		rev[n-1-i] = r
	}
	return NewRunesValue(rev, false), nil
}

// IMPURE by contract (mutates the receiver)
func runesReverseInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	slices.Reverse(o.Elements)
	return v, nil
}

// runesForEach is for_each(f): a full pass whose callback result is ignored — early exit belongs to for/break or a
// search member. Returns the receiver, so it chains.
// PURE by contract
func runesForEach(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Runes)(v.Ptr).Elements {
		if _, err := callElem(vm, fn, i, RuneValue(e)); err != nil {
			return Undefined, err
		}
	}
	return v, nil
}

// runesIndex is index(...): the first match.
// PURE by contract
func runesIndex(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return runesLocate(vm, "index", v, args, false)
}

// runesIndexLast is index_last(...): the last match.
// PURE by contract
func runesIndexLast(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return runesLocate(vm, "index_last", v, args, true)
}

// runesChunk is chunk(size): the symbols in pieces of size (the last one shorter), each an independent copy.
// PURE by contract
func runesChunk(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	size, err := readChunkSize("chunk", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		chunks = append(chunks, NewRunesValue(slices.Clone(elems[start:min(start+step, len(elems))]), false))
	}
	return NewArrayValue(chunks, false), nil
}

// runesChunkView is chunk_view(size): chunk's pieces sharing the receiver's storage, each marked as a view.
// PURE by contract
func runesChunkView(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	size, err := readChunkSize("chunk_view", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		c := NewRunesValue(elems[start:min(start+step, len(elems))], v.Immutable)
		(*Runes)(c.Ptr).IsView = true
		chunks = append(chunks, c)
	}
	return NewArrayValue(chunks, false), nil
}

// runesSliceView is slice_view([start[, end]]): the span sharing the receiver's storage, marked as a view.
// PURE by contract
func runesSliceView(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "slice_view"
	s, e, err := readSliceArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	si, ei, err := resolveSliceBounds(name, s, e, len(elems))
	if err != nil {
		return Undefined, err
	}
	res := NewRunesValue(elems[si:ei], v.Immutable)
	(*Runes)(res.Ptr).IsView = true
	return res, nil
}

// PURE by contract
func runesAppend(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return runesTypeAppend(v, args, false)
}

// IMPURE by contract (mutates the receiver)
func runesAppendInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return runesTypeAppend(v, args, true)
}

// runesPrepend is prepend(...items): a new runes with the items (read like append's operands) in front, in
// argument order — x.prepend(a, b) is a + b + x.
// PURE by contract
func runesPrepend(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := runesAddItems("prepend", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPrependInPlace is prepend_in_place(...items): prepend applied to the receiver
// itself.
func runesPrependInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "prepend_in_place"
	items, err := runesAddItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, items...))
	return v, nil
}

// runesPush is push(...items): a new runes with the items — exactly one symbol each — at the end.
// PURE by contract
func runesPush(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := runesPushItems("push", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(elems)+len(items))
	out = append(out, elems...)
	out = append(out, items...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPushInPlace is push_in_place(...items): push applied to the receiver itself.
func runesPushInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "push_in_place"
	items, err := runesPushItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(append(o.Elements, items...))
	return v, nil
}

// runesPushFirst is push_first(...items): a new runes with the items — exactly one symbol each — in front,
// in argument order.
// PURE by contract
func runesPushFirst(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := runesPushItems("push_first", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPushFirstInPlace is push_first_in_place(...items): push_first applied to the
// receiver itself.
func runesPushFirstInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "push_first_in_place"
	items, err := runesPushItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, items...))
	return v, nil
}

// runesInsert is insert(i, ...items): a new runes with the items — one symbol each — at position i, which
// raises out of [0, len].
// PURE by contract
func runesInsert(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "insert"
	elems := (*Runes)(v.Ptr).Elements
	at, err := readEditPos(name, args, len(elems))
	if err != nil {
		return Undefined, err
	}
	items, err := runesPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	out := slices.Insert(slices.Clone(elems), at, items...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesInsertInPlace is insert_in_place(i, ...items): insert applied to the receiver
// itself.
func runesInsertInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "insert_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	at, err := readEditPos(name, args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	items, err := runesPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	o.Set(slices.Insert(o.Elements, at, items...))
	return v, nil
}

// runesSplice is splice([start[, count[, ...items]]]): a new runes with count symbols at start replaced by
// the items (read like append's operands).
// PURE by contract
func runesSplice(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems := (*Runes)(v.Ptr).Elements
	start, end, err := readSpliceRange(args, len(elems))
	if err != nil {
		return Undefined, err
	}
	var items []rune
	if len(args) > 2 {
		if items, err = runesAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	out := make([]rune, 0, start+len(items)+len(elems)-end)
	out = append(out, elems[:start]...)
	out = append(out, items...)
	out = append(out, elems[end:]...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesSpliceInPlace is splice_in_place(...): splice applied to the receiver itself.
// A side-effecting member returns the RECEIVER, so mutators chain; the removed span is x.slice(i, j) taken
// beforehand.
func runesSpliceInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("splice_in_place", v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	start, end, err := readSpliceRange(args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	var items []rune
	if len(args) > 2 {
		if items, err = runesAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	o.Set(append(o.Elements[:start], append(items, o.Elements[end:]...)...))
	return v, nil
}

// runesMap is map(f): strictly 1:1, answering runes — each callback result must be exactly one symbol (an
// in-range int, byte or rune); a run or undefined raises, because widening and dropping are flat_map's job.
// PURE by contract
func runesMap(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, len(elems))
	for i, e := range elems {
		res, err := callElem(vm, fn, i, RuneValue(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback answered undefined — map is 1:1; the dropping form is flat_map")
		}
		enc, isElement, err := runesEncodeMatchArg(name, res)
		if err != nil {
			return Undefined, err
		}
		if !isElement {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback answered a sequence — map is 1:1; the concatenating form is flat_map")
		}
		if len(enc) != 1 {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback result does not fit a single element of the receiver")
		}
		out[i] = enc[0]
	}
	return NewRunesValue(out, false), nil
}

// runesFlatMap is flat_map(f): map, then concatenate — each callback result is text content appended as a run (a
// single element being a run of one); undefined contributes nothing.
// PURE by contract
func runesFlatMap(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "flat_map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(elems))
	for i, e := range elems {
		res, err := callElem(vm, fn, i, RuneValue(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			continue
		}
		enc, _, err := runesEncodeMatchArg(name, res)
		if err != nil {
			return Undefined, err
		}
		out = append(out, enc...)
	}
	return NewRunesValue(out, false), nil
}

// PURE by contract
func runesCaseFold(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	out := make([]rune, len(o.Elements))
	for i, r := range o.Elements {
		out[i] = foldRuneCanonical(r)
	}
	return NewRunesValue(out, false), nil
}

// PURE by contract
func runesTitleCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	// the label rendering segments on WRITTEN boundaries only (case transitions stay inside words);
	// the identifier renderings re-segment fully
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewRunesValue(caseJoinTitle(caseSegmentWritten(o.Elements)), false), nil
}

// PURE by contract
func runesSnakeCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewRunesValue(caseJoinLower(caseSegmentWords(o.Elements), '_'), false), nil
}

// PURE by contract
func runesKebabCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewRunesValue(caseJoinLower(caseSegmentWords(o.Elements), '-'), false), nil
}

// PURE by contract
func runesCamelCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewRunesValue(caseJoinCapitalized(caseSegmentWords(o.Elements), true), false), nil
}

// PURE by contract
func runesPascalCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewRunesValue(caseJoinCapitalized(caseSegmentWords(o.Elements), false), false), nil
}

// runesReduce is reduce(acc, f): folds the symbols left to right.
// PURE by contract
func runesReduce(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	acc, fn, err := readReduceArgs(args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Runes)(v.Ptr).Elements {
		acc, err = callReduce(vm, fn, acc, i, RuneValue(e))
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
}

// PURE by contract
func runesRepeat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Runes)(v.Ptr)
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
	out := make([]rune, total)
	// step by the receiver's length, never by the count: an empty receiver has total 0 and must not
	// spin n times copying nothing
	for i := 0; i < total; i += sl {
		copy(out[i:], src)
	}
	return NewRunesValue(out, false), nil
}

// runesSplit is split(...): the pieces between the separators. Explicit separators keep the empty pieces between
// adjacent hits (n hits answer n+1 pieces); the blank form answers the maximal runs of significant symbols, the
// classic whitespace split.
// PURE by contract
func runesSplit(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "split"
	elems := (*Runes)(v.Ptr).Elements

	if len(args) == 0 {
		var pieces []Value
		start := -1
		for i, e := range elems {
			if IsBlankRune(e) {
				if start >= 0 {
					pieces = append(pieces, runesPiece(elems[start:i]))
					start = -1
				}
			} else if start < 0 {
				start = i
			}
		}
		if start >= 0 {
			pieces = append(pieces, runesPiece(elems[start:]))
		}
		return NewArrayValue(pieces, false), nil
	}

	pieces := make([]Value, 0, 4)
	start := 0

	if args[0].IsCallable() {
		if len(args) > 1 {
			return Undefined, errPredicateAmongMany(name)
		}
		if err := checkElemCallback(name, args[0]); err != nil {
			return Undefined, err
		}
		for i, e := range elems {
			res, err := callElem(vm, args[0], i, RuneValue(e))
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				pieces = append(pieces, runesPiece(elems[start:i]))
				start = i + 1
			}
		}
		pieces = append(pieces, runesPiece(elems[start:]))
		return NewArrayValue(pieces, false), nil
	}

	runs, err := runesReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
	if err != nil {
		return Undefined, err
	}
	for i := 0; i < len(elems); {
		if k := runLengthAt(elems, i, runs); k > 0 {
			pieces = append(pieces, runesPiece(elems[start:i]))
			i += k
			start = i
		} else {
			i++
		}
	}
	pieces = append(pieces, runesPiece(elems[start:]))
	return NewArrayValue(pieces, false), nil
}

// PURE by contract
func runesSplitLines(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Runes)(v.Ptr)
	pieces := splitLinesString(EncodeText(o.Elements))
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewRunesValue(DecodeText(p), false)
	}
	return NewArrayValue(arr, false), nil
}

// runesPartition is partition(...): the one-split form, [before, separator, after] around the first hit (the
// longest run at that position); a miss answers [receiver, empty, empty]. The blank form takes the whole run of
// blanks as the separator.
// PURE by contract
func runesPartition(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "partition"
	elems := (*Runes)(v.Ptr).Elements
	found, n := -1, 0

	switch {
	case len(args) == 0:
		for i, e := range elems {
			if IsBlankRune(e) {
				found, n = i, 1
				for found+n < len(elems) && IsBlankRune(elems[found+n]) {
					n++
				}
				break
			}
		}

	case args[0].IsCallable():
		if len(args) > 1 {
			return Undefined, errPredicateAmongMany(name)
		}
		if err := checkElemCallback(name, args[0]); err != nil {
			return Undefined, err
		}
		for i, e := range elems {
			res, err := callElem(vm, args[0], i, RuneValue(e))
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				found, n = i, 1
				break
			}
		}

	default:
		runs, err := runesReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
		if err != nil {
			return Undefined, err
		}
		for i := range elems {
			if k := runLengthAt(elems, i, runs); k > 0 {
				found, n = i, k
				break
			}
		}
	}

	if found < 0 {
		return NewArrayValue([]Value{runesPiece(elems), runesPiece(nil), runesPiece(nil)}, false), nil
	}
	return NewArrayValue([]Value{runesPiece(elems[:found]), runesPiece(elems[found : found+n]), runesPiece(elems[found+n:])}, false), nil
}

// runesTrim is trim(...): without the leading and trailing elements of the set.
// PURE by contract
func runesTrim(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesTrimmed("trim", v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesTrimInPlace is trim_in_place(...): trim(...) applied to the receiver
// itself.
func runesTrimInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "trim_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesTrimmed(name, v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesTrimStart is trim_start(...): without the leading elements of the set.
// PURE by contract
func runesTrimStart(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesTrimmed("trim_start", v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesTrimStartInPlace is trim_start_in_place(...): trim_start(...) applied to the receiver
// itself.
func runesTrimStartInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "trim_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesTrimmed(name, v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesTrimEnd is trim_end(...): without the trailing elements of the set.
// PURE by contract
func runesTrimEnd(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesTrimmed("trim_end", v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesTrimEndInPlace is trim_end_in_place(...): trim_end(...) applied to the receiver
// itself.
func runesTrimEndInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "trim_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesTrimmed(name, v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesHasPrefix is has_prefix(...): does the receiver start with one of the runs?
// PURE by contract
func runesHasPrefix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	best, err := runesAnchoredRun("has_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// runesHasSuffix is has_suffix(...): does the receiver end with one of the runs?
// PURE by contract
func runesHasSuffix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	best, err := runesAnchoredRun("has_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// runesRemovePrefix is remove_prefix(...): without the longest matching prefix, once.
// PURE by contract
func runesRemovePrefix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesWithoutAnchored("remove_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesRemovePrefixInPlace is remove_prefix_in_place(...): remove_prefix(...) applied to the receiver
// itself.
func runesRemovePrefixInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "remove_prefix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesWithoutAnchored(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesRemoveSuffix is remove_suffix(...): without the longest matching suffix, once.
// PURE by contract
func runesRemoveSuffix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesWithoutAnchored("remove_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesRemoveSuffixInPlace is remove_suffix_in_place(...): remove_suffix(...) applied to the receiver
// itself.
func runesRemoveSuffixInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "remove_suffix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesWithoutAnchored(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesReplace is replace(...): every occurrence of old replaced by new.
// PURE by contract
func runesReplace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesReplaced("replace", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesReplaceInPlace is replace_in_place(...): replace(...) applied to the receiver
// itself.
func runesReplaceInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "replace_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesReplaced(name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesPadStart is pad_start(...): filled at the front up to n elements.
// PURE by contract
func runesPadStart(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesPadded("pad_start", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPadStartInPlace is pad_start_in_place(...): pad_start(...) applied to the receiver
// itself.
func runesPadStartInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "pad_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesPadded(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesPadEnd is pad_end(...): filled at the end up to n elements.
// PURE by contract
func runesPadEnd(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := runesPadded("pad_end", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPadEndInPlace is pad_end_in_place(...): pad_end(...) applied to the receiver
// itself.
func runesPadEndInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "pad_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesPadded(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}
