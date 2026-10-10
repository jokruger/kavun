package core

import (
	"slices"
	"strings"
	"unicode"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/conv"
)

// string's members: one function per entry of TypeString.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf, memberSlice in tools.go).

// PURE by contract
func stringString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// immutable and identity-less, so there is nothing to construct: the receiver IS the
	// independent value. It still takes the trailing default like every other conversion
	// cell, so generic x.string(fallback) code works on a string receiver too.
	return convMember(id.String(), stringTypeName, args, true, v)
}

// PURE by contract
func stringBytes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewBytesValue([]byte(*o), false), nil
}

// PURE by contract
func stringRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewRunesValue(DecodeText(*o), false), nil
}

// PURE by contract
func stringArray(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	t, _ := stringTypeAsArray(v)
	return NewArrayValue(t, false), nil
}

// PURE by contract
func stringBool(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	b, ok := conv.ParseBool(*(*string)(v.Ptr))
	return convMember(id.String(), stringTypeName, args, ok, BoolValue(b))
}

// PURE by contract
func stringFloat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f, ok := stringTypeAsFloat(v)
	return convMember(id.String(), stringTypeName, args, ok, FloatValue(f))
}

// PURE by contract
func stringInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	i, ok := stringTypeAsInt(v)
	return convMember(id.String(), stringTypeName, args, ok, IntValue(i))
}

// PURE by contract
func stringDecimal(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	d, ok := stringTypeAsDecimal(v)
	return convMember(id.String(), stringTypeName, args, ok, NewDecimalValue(d))
}

// PURE by contract
func stringTime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return textTimeMember(id.String(), stringTypeName, *(*string)(v.Ptr), args)
}

// PURE by contract
func stringDate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return textDateMember(id.String(), stringTypeName, *(*string)(v.Ptr), args)
}

// PURE by contract
func stringIsValid(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	// no escapes anywhere: the text is well-formed UTF-8 end to end
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(TextIsValid(*o)), nil
}

// PURE by contract
func stringIsASCII(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(IsASCIIText(*o)), nil
}

// PURE by contract
func stringIsEmpty(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(len(*o) == 0), nil
}

// PURE by contract
func stringLen(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(v.Data)), nil // symbols, not bytes
}

// PURE by contract
func stringLower(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(EncodeText(mapRunesCase(DecodeText(*o), unicode.ToLower))), nil
}

// PURE by contract
func stringUpper(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(EncodeText(mapRunesCase(DecodeText(*o), unicode.ToUpper))), nil
}

// stringContains is contains(...): is there a match anywhere? The empty run is contained everywhere, the
// same answer `in` gives.
// PURE by contract
func stringContains(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("contains", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))

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

// stringCount is count(...): how many matches. Runs count non-overlapping occurrences.
// PURE by contract
func stringCount(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("count", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))

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

// stringKeep is keep(...): a new string of the matches, in order.
// PURE by contract
func stringKeep(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("keep", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
	out := make([]rune, 0, len(elems))

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := runLengthAt(elems, i, m.runs); k > 0 {
				out = append(out, elems[i:i+k]...)
				i += k
			} else {
				i++
			}
		}
		return NewStringValue(EncodeText(out)), nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if hit {
			out = append(out, r)
		}
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringRemove is remove(...): a new string without the matches, in order. With no argument remove drops the
// BLANK symbols — so it keeps the significant ones, landing on keep()'s answer by the opposite action.
// PURE by contract
func stringRemove(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("remove", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
	out := make([]rune, 0, len(elems))

	if m.significant {
		for _, r := range elems {
			if !IsBlankRune(r) {
				out = append(out, r)
			}
		}
		return NewStringValue(EncodeText(out)), nil
	}

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := runLengthAt(elems, i, m.runs); k > 0 {
				i += k
			} else {
				out = append(out, elems[i])
				i++
			}
		}
		return NewStringValue(EncodeText(out)), nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if !hit {
			out = append(out, r)
		}
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringAny is any(...): does some symbol match? Symbols, a predicate, or nothing — never a run.
// PURE by contract
func stringAny(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("any", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range DecodeText(*(*string)(v.Ptr)) {
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

// stringAll is all(...): does every symbol match? Symbols, a predicate, or nothing — never a run. True on the empty
// string.
// PURE by contract
func stringAll(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("all", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range DecodeText(*(*string)(v.Ptr)) {
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

// stringAppend is append(...items): x.append(a, b) is x + a + b.
// PURE by contract
func stringAppend(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := stringAddItems("append", args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(*(*string)(v.Ptr) + EncodeText(items)), nil
}

// stringPrepend is prepend(...items): x.prepend(a, b) is a + b + x.
// PURE by contract
func stringPrepend(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := stringAddItems("prepend", args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(items) + *(*string)(v.Ptr)), nil
}

// stringPush is push(...items): the items — exactly one symbol each — appended.
// PURE by contract
func stringPush(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := stringPushItems("push", args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(*(*string)(v.Ptr) + EncodeText(items)), nil
}

// stringPushFirst is push_first(...items): the items — exactly one symbol each — in front, in argument order.
// PURE by contract
func stringPushFirst(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	items, err := stringPushItems("push_first", args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(items) + *(*string)(v.Ptr)), nil
}

// stringTrim is trim(...): without the leading and trailing elements of the set.
// PURE by contract
func stringTrim(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := stringTrimmed("trim", v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringTrimStart is trim_start(...): without the leading elements of the set.
// PURE by contract
func stringTrimStart(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := stringTrimmed("trim_start", v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringTrimEnd is trim_end(...): without the trailing elements of the set.
// PURE by contract
func stringTrimEnd(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := stringTrimmed("trim_end", v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringHasPrefix is has_prefix(...): does the receiver start with one of the runs?
// PURE by contract
func stringHasPrefix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	best, err := stringAnchoredRun("has_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// stringHasSuffix is has_suffix(...): does the receiver end with one of the runs?
// PURE by contract
func stringHasSuffix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	best, err := stringAnchoredRun("has_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// stringRemovePrefix is remove_prefix(...): without the longest matching prefix, once.
// PURE by contract
func stringRemovePrefix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := stringWithoutAnchored("remove_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringRemoveSuffix is remove_suffix(...): without the longest matching suffix, once.
// PURE by contract
func stringRemoveSuffix(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := stringWithoutAnchored("remove_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringReplace is replace(...): every occurrence of old replaced by new.
// PURE by contract
func stringReplace(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := stringReplaced("replace", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringPadStart is pad_start(...): filled at the front up to n elements.
// PURE by contract
func stringPadStart(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := stringPadded("pad_start", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringPadEnd is pad_end(...): filled at the end up to n elements.
// PURE by contract
func stringPadEnd(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := stringPadded("pad_end", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// PURE by contract
func stringReverse(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	rs := DecodeText(*o)
	slices.Reverse(rs)
	return NewStringValue(EncodeText(rs)), nil
}

// PURE by contract
func stringFirst(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*string)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	rs := DecodeText(*o)
	if len(rs) == 0 {
		// absence is data: undefined, or the optional trailing default
		return emptySeqResult(name, args)
	}
	return RuneValue(rs[0]), nil
}

// PURE by contract
func stringLast(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*string)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	rs := DecodeText(*o)
	if len(rs) == 0 {
		return emptySeqResult(name, args)
	}
	return RuneValue(rs[len(rs)-1]), nil
}

// PURE by contract
func stringMin(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*string)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	rs := DecodeText(*o)
	if len(rs) == 0 {
		return emptySeqResult(name, args)
	}
	return RuneValue(slices.Min(rs)), nil
}

// PURE by contract
func stringMax(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*string)(v.Ptr)
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	rs := DecodeText(*o)
	if len(rs) == 0 {
		return emptySeqResult(name, args)
	}
	return RuneValue(slices.Max(rs)), nil
}

// PURE by contract
func stringSort(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	rs := DecodeText(*o)
	slices.Sort(rs)
	return NewStringValue(EncodeText(rs)), nil
}

// PURE by contract
func stringDedup(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	rs := DecodeText(*o)
	out := make([]rune, 0, len(rs))
	for i, r := range rs {
		if i == 0 || r != rs[i-1] {
			out = append(out, r)
		}
	}
	return NewStringValue(EncodeText(out)), nil
}

// PURE by contract
func stringUnique(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	rs := DecodeText(*o)
	out := make([]rune, 0, len(rs))
	seen := make(map[rune]struct{}, len(rs))
	for _, r := range rs {
		if _, ok := seen[r]; !ok {
			seen[r] = struct{}{}
			out = append(out, r)
		}
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringChunk is chunk(size): the symbols in strings of size symbols (the last one shorter).
// PURE by contract
func stringChunk(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	size, err := readChunkSize("chunk", args)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		chunks = append(chunks, NewStringValue(EncodeText(elems[start:min(start+step, len(elems))])))
	}
	return NewArrayValue(chunks, false), nil
}

// stringInsert is insert(i, ...items): a new string with the items — one symbol each — at symbol position i, which
// raises out of [0, len].
// PURE by contract
func stringInsert(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "insert"
	elems := DecodeText(*(*string)(v.Ptr))
	at, err := readEditPos(name, args, len(elems))
	if err != nil {
		return Undefined, err
	}
	items, err := stringPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(slices.Insert(elems, at, items...))), nil
}

// stringSplice is splice([start[, count[, ...items]]]): a new string with count symbols at start replaced by the
// items (read like append's operands).
// PURE by contract
func stringSplice(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems := DecodeText(*(*string)(v.Ptr))
	start, end, err := readSpliceRange(args, len(elems))
	if err != nil {
		return Undefined, err
	}
	var items []rune
	if len(args) > 2 {
		if items, err = stringAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	out := make([]rune, 0, start+len(items)+len(elems)-end)
	out = append(out, elems[:start]...)
	out = append(out, items...)
	out = append(out, elems[end:]...)
	return NewStringValue(EncodeText(out)), nil
}

// stringMap is map(f): strictly 1:1, answering a string — each callback result must be exactly one symbol (an
// in-range int, byte or rune); a run or undefined raises, because widening and dropping are flat_map's job.
// PURE by contract
func stringMap(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
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
	return NewStringValue(EncodeText(out)), nil
}

// stringFlatMap is flat_map(f): map, then concatenate — each callback result is text content appended as a run (a
// single element being a run of one); undefined contributes nothing.
// PURE by contract
func stringFlatMap(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "flat_map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
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
	return NewStringValue(EncodeText(out)), nil
}

// stringReduce is reduce(acc, f): folds the symbols left to right.
// PURE by contract
func stringReduce(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	acc, fn, err := readReduceArgs(args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range DecodeText(*(*string)(v.Ptr)) {
		acc, err = callReduce(vm, fn, acc, i, RuneValue(e))
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
}

// PURE by contract
func stringCaseFold(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	rs := DecodeText(*o)
	for i, r := range rs {
		rs[i] = foldRuneCanonical(r)
	}
	return NewStringValue(EncodeText(rs)), nil
}

// PURE by contract
func stringTitleCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	// the label rendering segments on WRITTEN boundaries only (case transitions stay inside words);
	// the identifier renderings re-segment fully
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(EncodeText(caseJoinTitle(caseSegmentWritten(DecodeText(*o))))), nil
}

// PURE by contract
func stringSnakeCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(EncodeText(caseJoinLower(caseSegmentWords(DecodeText(*o)), '_'))), nil
}

// PURE by contract
func stringKebabCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(EncodeText(caseJoinLower(caseSegmentWords(DecodeText(*o)), '-'))), nil
}

// PURE by contract
func stringCamelCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(EncodeText(caseJoinCapitalized(caseSegmentWords(DecodeText(*o)), true))), nil
}

// PURE by contract
func stringPascalCase(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*string)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return NewStringValue(EncodeText(caseJoinCapitalized(caseSegmentWords(DecodeText(*o)), false))), nil
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
//
// stringForEach is for_each(f): a full pass whose callback result is ignored; returns the receiver, so it chains.
// It walks the Go string directly rather than decoding it first.
func stringForEach(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}
	i := 0
	for _, r := range *(*string)(v.Ptr) {
		if _, err := callElem(vm, fn, i, RuneValue(r)); err != nil {
			return Undefined, err
		}
		i++
	}
	return v, nil
}

// stringIndex is index(...): the first match.
// PURE by contract
func stringIndex(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return stringLocate(vm, "index", v, args, false)
}

// stringIndexLast is index_last(...): the last match.
// PURE by contract
func stringIndexLast(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return stringLocate(vm, "index_last", v, args, true)
}

// PURE by contract
func stringRepeat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	o := (*string)(v.Ptr)
	n, err := parseRepeatCount(name, args)
	if err != nil {
		return Undefined, err
	}
	if _, err := SeqRepeatTotal(name, n, len(*o)); err != nil {
		return Undefined, err
	}
	return NewStringValue(strings.Repeat(*o, n)), nil
}

// stringSplit is split(...): the pieces between the separators. Explicit separators keep the empty pieces between
// adjacent hits (n hits answer n+1 pieces); the blank form answers the maximal runs of significant symbols, the
// classic whitespace split.
// PURE by contract
func stringSplit(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "split"
	elems := DecodeText(*(*string)(v.Ptr))

	if len(args) == 0 {
		var pieces []Value
		start := -1
		for i, e := range elems {
			if IsBlankRune(e) {
				if start >= 0 {
					pieces = append(pieces, stringPiece(elems[start:i]))
					start = -1
				}
			} else if start < 0 {
				start = i
			}
		}
		if start >= 0 {
			pieces = append(pieces, stringPiece(elems[start:]))
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
				pieces = append(pieces, stringPiece(elems[start:i]))
				start = i + 1
			}
		}
		pieces = append(pieces, stringPiece(elems[start:]))
		return NewArrayValue(pieces, false), nil
	}

	runs, err := stringReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
	if err != nil {
		return Undefined, err
	}
	for i := 0; i < len(elems); {
		if k := runLengthAt(elems, i, runs); k > 0 {
			pieces = append(pieces, stringPiece(elems[start:i]))
			i += k
			start = i
		} else {
			i++
		}
	}
	pieces = append(pieces, stringPiece(elems[start:]))
	return NewArrayValue(pieces, false), nil
}

// stringPartition is partition(...): the one-split form, [before, separator, after] around the first hit (the
// longest run at that position); a miss answers [receiver, empty, empty]. The blank form takes the whole run of
// blanks as the separator.
// PURE by contract
func stringPartition(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "partition"
	elems := DecodeText(*(*string)(v.Ptr))
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
		runs, err := stringReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
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
		return NewArrayValue([]Value{stringPiece(elems), stringPiece(nil), stringPiece(nil)}, false), nil
	}
	return NewArrayValue([]Value{stringPiece(elems[:found]), stringPiece(elems[found : found+n]), stringPiece(elems[found+n:])}, false), nil
}

// PURE by contract
func stringSplitLines(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*string)(v.Ptr)
	pieces := splitLinesString(*o)
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewStringValue(p)
	}
	return NewArrayValue(arr, false), nil
}
