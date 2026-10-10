package core

import (
	"slices"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// bytes's members: one function per entry of TypeBytes.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf, memberSlice in tools.go).

// PURE by contract
func bytesCopy(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return bytesTypeCopy(v, true)
}

// PURE by contract
func bytesFreeze(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v.Freeze()
}

// PURE by contract
func bytesBytes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the same-type conversion constructs — a new, independent, mutable copy, exactly
	// bytes(b) / b.copy(); see the note on array's own case. b"..." literals reach a
	// writable body through this too.
	c, err := bytesTypeCopy(v, false)
	if err != nil {
		return Undefined, err
	}
	return convMember(id.String(), bytesTypeName, args, true, c)
}

// PURE by contract
func bytesArray(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	t, _ := bytesTypeAsArray(v)
	return NewArrayValue(t, false), nil
}

// PURE by contract
func bytesString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	// TOTAL: the decode never fails and never loses an octet — an undecodable one becomes its
	// reserved escape (see text_escape.go), so .string().bytes() returns these octets exactly.
	// is_valid() is how a script asks whether any escape is in there
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(string(o.Elements)), nil
}

// PURE by contract
func bytesRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	// the same decode, materialized as symbols — the mirror of .bytes()
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewRunesValue(DecodeOctets(o.Elements), false), nil
}

// PURE by contract
func bytesIsASCII(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	// bytes has no is_valid: every octet is a valid octet. The decode question is
	// b.string().is_valid(), which asks it of the type that can answer it
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(OctetsAreASCII(o.Elements)), nil
}

// PURE by contract
func bytesIsEmpty(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(len(o.Elements) == 0), nil
}

// PURE by contract
func bytesLen(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(len(o.Elements))), nil
}

// PURE by contract
func bytesFirst(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Bytes)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		return emptySeqResult(name, args)
	}
	return ByteValue(o.Elements[0]), nil
}

// PURE by contract
func bytesLast(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Bytes)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		return emptySeqResult(name, args)
	}
	return ByteValue(o.Elements[len(o.Elements)-1]), nil
}

// PURE by contract
func bytesMin(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Bytes)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		// absence is data: undefined, or the optional trailing default
		return emptySeqResult(name, args)
	}
	return ByteValue(slices.Min(o.Elements)), nil
}

// PURE by contract
func bytesMax(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Bytes)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if len(o.Elements) == 0 {
		// absence is data: undefined, or the optional trailing default
		return emptySeqResult(name, args)
	}
	return ByteValue(slices.Max(o.Elements)), nil
}

// bytesContains is contains(...): is there a match anywhere? The empty run is contained everywhere, the
// same answer `in` gives.
// PURE by contract
func bytesContains(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := bytesReadMatchArgs("contains", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements

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

// bytesCount is count(...): how many matches. Runs count non-overlapping occurrences.
// PURE by contract
func bytesCount(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := bytesReadMatchArgs("count", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements

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

// bytesKeep is keep(...): new bytes of the matches.
// PURE by contract
func bytesKeep(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesKept(vm, "keep", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesKeepInPlace is keep_in_place(...): keep the matches in the receiver itself.
func bytesKeepInPlace(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "keep_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesKept(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesRemove is remove(...): new bytes without the matches.
// PURE by contract
func bytesRemove(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesRemaining(vm, "remove", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesRemoveInPlace is remove_in_place(...): drop the matches from the receiver
// itself.
func bytesRemoveInPlace(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "remove_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesRemaining(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesAny is any(...): does some octet match? Symbols, a predicate, or nothing — never a run.
// PURE by contract
func bytesAny(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := bytesReadMatchArgs("any", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range (*Bytes)(v.Ptr).Elements {
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

// bytesAll is all(...): does every octet match? Symbols, a predicate, or nothing — never a run. True on empty
// bytes.
// PURE by contract
func bytesAll(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := bytesReadMatchArgs("all", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range (*Bytes)(v.Ptr).Elements {
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
func bytesSort(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	sorted := make([]byte, len(o.Elements))
	copy(sorted, o.Elements)
	slices.Sort(sorted)
	return NewBytesValue(sorted, false), nil
}

// IMPURE by contract (mutates the receiver)
func bytesSortInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Bytes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	slices.Sort(o.Elements)
	return v, nil
}

// bytesDedup is dedup(): a new bytes with each run of equal neighbours collapsed to one.
// PURE by contract
func bytesDedup(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dedup", "0", len(args))
	}
	return NewBytesValue(bytesDeduped((*Bytes)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. bytesDedupInPlace is dedup_in_place(): dedup applied to the receiver itself.
func bytesDedupInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "dedup_in_place"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(bytesDeduped(o.Elements))
	return v, nil
}

// bytesUnique is unique(): a new bytes without repeats, each element kept at its first occurrence.
// PURE by contract
func bytesUnique(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}
	return NewBytesValue(bytesUniqueElements((*Bytes)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. bytesUniqueInPlace is unique_in_place(): unique applied to the receiver itself.
func bytesUniqueInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "unique_in_place"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(bytesUniqueElements(o.Elements))
	return v, nil
}

// PURE by contract
func bytesReverse(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	n := len(o.Elements)
	rev := make([]byte, n)
	for i, b := range o.Elements {
		rev[n-1-i] = b
	}
	return NewBytesValue(rev, false), nil
}

// IMPURE by contract (mutates the receiver)
func bytesReverseInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Bytes)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	slices.Reverse(o.Elements)
	return v, nil
}

// bytesForEach is for_each(f): a full pass whose callback result is ignored — early exit belongs to for/break or a
// search member. Returns the receiver, so it chains.
// PURE by contract
func bytesForEach(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Bytes)(v.Ptr).Elements {
		if _, err := callElem(vm, fn, i, ByteValue(e)); err != nil {
			return Undefined, err
		}
	}
	return v, nil
}

// bytesIndex is index(...): the first match.
// PURE by contract
func bytesIndex(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return bytesLocate(vm, "index", v, args, false)
}

// bytesIndexLast is index_last(...): the last match.
// PURE by contract
func bytesIndexLast(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return bytesLocate(vm, "index_last", v, args, true)
}

// bytesChunk is chunk(size): the octets in pieces of size (the last one shorter), each an independent copy.
// PURE by contract
func bytesChunk(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	size, err := readChunkSize("chunk", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		chunks = append(chunks, NewBytesValue(slices.Clone(elems[start:min(start+step, len(elems))]), false))
	}
	return NewArrayValue(chunks, false), nil
}

// bytesChunkView is chunk_view(size): chunk's pieces sharing the receiver's storage, each marked as a view.
// PURE by contract
func bytesChunkView(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	size, err := readChunkSize("chunk_view", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		c := NewBytesValue(elems[start:min(start+step, len(elems))], v.Immutable)
		(*Bytes)(c.Ptr).IsView = true
		chunks = append(chunks, c)
	}
	return NewArrayValue(chunks, false), nil
}

// bytesSliceView is slice_view([start[, end]]): the span sharing the receiver's storage, marked as a view.
// PURE by contract
func bytesSliceView(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "slice_view"
	s, e, err := readSliceArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	si, ei, err := resolveSliceBounds(name, s, e, len(elems))
	if err != nil {
		return Undefined, err
	}
	res := NewBytesValue(elems[si:ei], v.Immutable)
	(*Bytes)(res.Ptr).IsView = true
	return res, nil
}

// PURE by contract
func bytesAppend(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return bytesTypeAppend(v, args, false)
}

// IMPURE by contract (mutates the receiver)
func bytesAppendInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return bytesTypeAppend(v, args, true)
}

// bytesPrepend is prepend(...items): a new bytes with the items (read like append's operands) in front, in
// argument order — x.prepend(a, b) is a + b + x.
// PURE by contract
func bytesPrepend(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := bytesAddItems("prepend", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPrependInPlace is prepend_in_place(...items): prepend applied to the receiver
// itself.
func bytesPrependInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "prepend_in_place"
	items, err := bytesAddItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, items...))
	return v, nil
}

// bytesPush is push(...items): a new bytes with the items — exactly one octet each — at the end.
// PURE by contract
func bytesPush(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := bytesPushItems("push", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(elems)+len(items))
	out = append(out, elems...)
	out = append(out, items...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPushInPlace is push_in_place(...items): push applied to the receiver itself.
func bytesPushInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "push_in_place"
	items, err := bytesPushItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(append(o.Elements, items...))
	return v, nil
}

// bytesPushFirst is push_first(...items): a new bytes with the items — exactly one octet each — in front,
// in argument order.
// PURE by contract
func bytesPushFirst(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := bytesPushItems("push_first", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPushFirstInPlace is push_first_in_place(...items): push_first applied to the
// receiver itself.
func bytesPushFirstInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "push_first_in_place"
	items, err := bytesPushItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, items...))
	return v, nil
}

// bytesInsert is insert(i, ...items): a new bytes with the items — one octet each — at position i, which
// raises out of [0, len].
// PURE by contract
func bytesInsert(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "insert"
	elems := (*Bytes)(v.Ptr).Elements
	at, err := readEditPos(name, args, len(elems))
	if err != nil {
		return Undefined, err
	}
	items, err := bytesPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	out := slices.Insert(slices.Clone(elems), at, items...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesInsertInPlace is insert_in_place(i, ...items): insert applied to the receiver
// itself.
func bytesInsertInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "insert_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	at, err := readEditPos(name, args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	items, err := bytesPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	o.Set(slices.Insert(o.Elements, at, items...))
	return v, nil
}

// bytesSplice is splice([start[, count[, ...items]]]): a new bytes with count octets at start replaced by
// the items (read like append's operands).
// PURE by contract
func bytesSplice(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems := (*Bytes)(v.Ptr).Elements
	start, end, err := readSpliceRange(args, len(elems))
	if err != nil {
		return Undefined, err
	}
	var items []byte
	if len(args) > 2 {
		if items, err = bytesAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	out := make([]byte, 0, start+len(items)+len(elems)-end)
	out = append(out, elems[:start]...)
	out = append(out, items...)
	out = append(out, elems[end:]...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesSpliceInPlace is splice_in_place(...): splice applied to the receiver itself.
// A side-effecting member returns the RECEIVER, so mutators chain; the removed span is x.slice(i, j) taken
// beforehand.
func bytesSpliceInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("splice_in_place", v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	start, end, err := readSpliceRange(args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	var items []byte
	if len(args) > 2 {
		if items, err = bytesAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	o.Set(append(o.Elements[:start], append(items, o.Elements[end:]...)...))
	return v, nil
}

// bytesMap is map(f): strictly 1:1, answering bytes — each callback result must be exactly one octet (an
// in-range int, byte or rune); a run or undefined raises, because widening and dropping are flat_map's job.
// PURE by contract
func bytesMap(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, len(elems))
	for i, e := range elems {
		res, err := callElem(vm, fn, i, ByteValue(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback answered undefined — map is 1:1; the dropping form is flat_map")
		}
		enc, isElement, err := bytesEncodeMatchArg(name, res)
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
	return NewBytesValue(out, false), nil
}

// bytesFlatMap is flat_map(f): map, then concatenate — each callback result is text content appended as a run (a
// single element being a run of one); undefined contributes nothing.
// PURE by contract
func bytesFlatMap(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "flat_map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(elems))
	for i, e := range elems {
		res, err := callElem(vm, fn, i, ByteValue(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			continue
		}
		enc, _, err := bytesEncodeMatchArg(name, res)
		if err != nil {
			return Undefined, err
		}
		out = append(out, enc...)
	}
	return NewBytesValue(out, false), nil
}

// bytesReduce is reduce(acc, f): folds the octets left to right.
// PURE by contract
func bytesReduce(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	acc, fn, err := readReduceArgs(args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Bytes)(v.Ptr).Elements {
		acc, err = callReduce(vm, fn, acc, i, ByteValue(e))
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
}

// PURE by contract
func bytesRepeat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*Bytes)(v.Ptr)
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
	out := make([]byte, total)
	// step by the receiver's length, never by the count: an empty receiver has total 0 and must not
	// spin n times copying nothing
	for i := 0; i < total; i += sl {
		copy(out[i:], src)
	}
	return NewBytesValue(out, false), nil
}

// bytesSplit is split(...): the pieces between the separators. Explicit separators keep the empty pieces between
// adjacent hits (n hits answer n+1 pieces); the blank form answers the maximal runs of significant octets, the
// classic whitespace split.
// PURE by contract
func bytesSplit(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "split"
	elems := (*Bytes)(v.Ptr).Elements

	if len(args) == 0 {
		var pieces []Value
		start := -1
		for i, e := range elems {
			if IsBlankByte(e) {
				if start >= 0 {
					pieces = append(pieces, bytesPiece(elems[start:i]))
					start = -1
				}
			} else if start < 0 {
				start = i
			}
		}
		if start >= 0 {
			pieces = append(pieces, bytesPiece(elems[start:]))
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
			res, err := callElem(vm, args[0], i, ByteValue(e))
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				pieces = append(pieces, bytesPiece(elems[start:i]))
				start = i + 1
			}
		}
		pieces = append(pieces, bytesPiece(elems[start:]))
		return NewArrayValue(pieces, false), nil
	}

	runs, err := bytesReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
	if err != nil {
		return Undefined, err
	}
	for i := 0; i < len(elems); {
		if k := runLengthAt(elems, i, runs); k > 0 {
			pieces = append(pieces, bytesPiece(elems[start:i]))
			i += k
			start = i
		} else {
			i++
		}
	}
	pieces = append(pieces, bytesPiece(elems[start:]))
	return NewArrayValue(pieces, false), nil
}

// PURE by contract
func bytesSplitLines(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Bytes)(v.Ptr)
	pieces := splitLinesBytes(o.Elements)
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		buf := make([]byte, len(p))
		copy(buf, p)
		arr[i] = NewBytesValue(buf, false)
	}
	return NewArrayValue(arr, false), nil
}

// bytesPartition is partition(...): the one-split form, [before, separator, after] around the first hit (the
// longest run at that position); a miss answers [receiver, empty, empty]. The blank form takes the whole run of
// blanks as the separator.
// PURE by contract
func bytesPartition(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "partition"
	elems := (*Bytes)(v.Ptr).Elements
	found, n := -1, 0

	switch {
	case len(args) == 0:
		for i, e := range elems {
			if IsBlankByte(e) {
				found, n = i, 1
				for found+n < len(elems) && IsBlankByte(elems[found+n]) {
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
			res, err := callElem(vm, args[0], i, ByteValue(e))
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
		runs, err := bytesReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
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
		return NewArrayValue([]Value{bytesPiece(elems), bytesPiece(nil), bytesPiece(nil)}, false), nil
	}
	return NewArrayValue([]Value{bytesPiece(elems[:found]), bytesPiece(elems[found : found+n]), bytesPiece(elems[found+n:])}, false), nil
}

// bytesTrim is trim(...): without the leading and trailing elements of the set.
// PURE by contract
func bytesTrim(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesTrimmed("trim", v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesTrimInPlace is trim_in_place(...): trim(...) applied to the receiver
// itself.
func bytesTrimInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "trim_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesTrimmed(name, v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesTrimStart is trim_start(...): without the leading elements of the set.
// PURE by contract
func bytesTrimStart(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesTrimmed("trim_start", v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesTrimStartInPlace is trim_start_in_place(...): trim_start(...) applied to the receiver
// itself.
func bytesTrimStartInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "trim_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesTrimmed(name, v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesTrimEnd is trim_end(...): without the trailing elements of the set.
// PURE by contract
func bytesTrimEnd(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesTrimmed("trim_end", v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesTrimEndInPlace is trim_end_in_place(...): trim_end(...) applied to the receiver
// itself.
func bytesTrimEndInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "trim_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesTrimmed(name, v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesHasPrefix is has_prefix(...): does the receiver start with one of the runs?
// PURE by contract
func bytesHasPrefix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	best, err := bytesAnchoredRun("has_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// bytesHasSuffix is has_suffix(...): does the receiver end with one of the runs?
// PURE by contract
func bytesHasSuffix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	best, err := bytesAnchoredRun("has_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// bytesRemovePrefix is remove_prefix(...): without the longest matching prefix, once.
// PURE by contract
func bytesRemovePrefix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesWithoutAnchored("remove_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesRemovePrefixInPlace is remove_prefix_in_place(...): remove_prefix(...) applied to the receiver
// itself.
func bytesRemovePrefixInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "remove_prefix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesWithoutAnchored(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesRemoveSuffix is remove_suffix(...): without the longest matching suffix, once.
// PURE by contract
func bytesRemoveSuffix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesWithoutAnchored("remove_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesRemoveSuffixInPlace is remove_suffix_in_place(...): remove_suffix(...) applied to the receiver
// itself.
func bytesRemoveSuffixInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "remove_suffix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesWithoutAnchored(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesReplace is replace(...): every occurrence of old replaced by new.
// PURE by contract
func bytesReplace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesReplaced("replace", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesReplaceInPlace is replace_in_place(...): replace(...) applied to the receiver
// itself.
func bytesReplaceInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "replace_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesReplaced(name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesPadStart is pad_start(...): filled at the front up to n elements.
// PURE by contract
func bytesPadStart(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesPadded("pad_start", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPadStartInPlace is pad_start_in_place(...): pad_start(...) applied to the receiver
// itself.
func bytesPadStartInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "pad_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesPadded(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesPadEnd is pad_end(...): filled at the end up to n elements.
// PURE by contract
func bytesPadEnd(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := bytesPadded("pad_end", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPadEndInPlace is pad_end_in_place(...): pad_end(...) applied to the receiver
// itself.
func bytesPadEndInPlace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "pad_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesPadded(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}
