package core

import (
	"slices"
	"strings"

	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

type Seq[T any] struct {
	Elements []T
	// IsView reports whether Elements shares backing storage with some other Value, rather than being an
	// independently-owned allocation. Set only by the explicit `_view` constructors (slice_view/chunk_view);
	// every other constructor path leaves it at its zero value (false), including today's still-sharing
	// `slice`/`chunk` default — that default hasn't been renamed to `_view` yet (see P4-002), so it isn't
	// tagged as one. Read by the `is_view()` member predicate.
	IsView bool
}

func (o *Seq[T]) Set(elements []T) {
	o.Elements = elements
}

// SeqForEach iterates over the elements of the sequence and calls the provided callback function for each element.
func SeqForEach[T any](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value,
	resolve func(Value) *Seq[T],
) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	// a FULL pass whose callback return value is IGNORED: in a dynamically typed
	// language a control protocol on the return is indistinguishable from a
	// forgotten `return` (which yields falsy undefined), so the natural spelling
	// used to visit exactly one element and silently stop. Early exit belongs to
	// `for`/`break` or a search member. Returns the receiver, so it chains.
	o := resolve(v)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, e := range o.Elements {
			buf[0] = t2v(e)
			if _, err := fn.Call(vm, buf[:1]); err != nil {
				return Undefined, err
			}
		}

	case 2:
		for i, e := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = t2v(e)
			if _, err := fn.Call(vm, buf[:2]); err != nil {
				return Undefined, err
			}
		}
	}

	return v, nil
}

// SeqMap applies a given function to each element in the sequence and returns a new sequence containing the results.
func SeqMap[T any](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("map", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError("map", "first", "function", fn.TypeName())
	}

	var buf [2]Value
	o := resolve(v)
	mapped := make([]Value, len(o.Elements))

	switch fn.Arity() {
	case 1:
		for i, e := range o.Elements {
			buf[0] = t2v(e)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			mapped[i] = res
		}
		return NewArrayValue(mapped, false), nil

	case 2:
		for i, e := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = t2v(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			mapped[i] = res
		}
		return NewArrayValue(mapped, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("map", "first", "f/1 or f/2", fn.TypeName())
	}
}

// SeqReduce reduces the sequence to a single value by applying a given binary function cumulatively to the elements of
// the sequence, from left to right.
// The function can have arity 2 (accumulator, element) or 3 (accumulator, index, element).
func SeqReduce[T any](
	vm VM,
	v Value,
	args []Value,
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError("reduce", "2", len(args))
	}

	acc := args[0]
	fn := args[1]
	if !fn.IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError("reduce", "second", "function", fn.TypeName())
	}

	o := resolve(v)
	var buf [3]Value
	switch fn.Arity() {
	case 2:
		for _, e := range o.Elements {
			buf[0] = acc
			buf[1] = t2v(e)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	case 3:
		for i, e := range o.Elements {
			buf[0] = acc
			buf[1] = IntValue(int64(i))
			buf[2] = t2v(e)
			res, err := fn.Call(vm, buf[:3])
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("reduce", "second", "f/2 or f/3", fn.TypeName())
	}
}

// ---------------------------------------------------------------------------
// The match-member engine: contains / count / filter / remove / any / all.
// One name per operation; the ARGUMENT'S TYPE selects the reading — an
// argument of the receiver's own kind is a contiguous run, a function is a
// predicate, anything else is one element, no argument means the blank set,
// and a variadic call is a homogeneous SET (mixing readings raises naming the
// mixture; a function among several arguments always raises).
// ---------------------------------------------------------------------------

// matchPlan is a match call after dispatch: exactly one of pred/runs is set.
type matchPlan[T any] struct {
	pred func(vm VM, i int, e T) (bool, error)
	runs [][]T
}

// seqElementPred builds the two-binding element predicate every predicate-declaring member shares:
// f/1 receives the element, f/2 receives (locator, element).
func seqElementPred[T any](name string, fn Value, t2v func(T) Value) (func(vm VM, i int, e T) (bool, error), error) {
	arity := fn.Arity()
	if arity != 1 && arity != 2 {
		return nil, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
	return func(vm VM, i int, e T) (bool, error) {
		var buf [2]Value
		n := 1
		if arity >= 2 {
			buf[0] = IntValue(int64(i))
			buf[1] = t2v(e)
			n = 2
		} else {
			buf[0] = t2v(e)
		}
		res, err := fn.Call(vm, buf[:n])
		if err != nil {
			return false, err
		}
		return res.IsTrue()
	}, nil
}

// seqMatchDispatch classifies a match member's arguments. toElem returns
// (elem, ok, err): err is an acceptance error (wrong type or out of range for
// this receiver); ok=false with err=nil means "not an element" (checked as a
// run next). toRun == nil means the run reading is refused by the member.
func seqMatchDispatch[T any](
	vm VM,
	name string,
	args []Value,
	t2v func(T) Value,
	toElem func(Value) (T, bool, error),
	isRunArg func(Value) bool,
	toRun func(Value) ([]T, error),
	eq func(T, T) bool,
	isBlank func(T) bool,
	noArgMatchesBlank bool,
	allowAbsent bool,
) (*matchPlan[T], error) {
	if len(args) == 0 {
		if !allowAbsent {
			return nil, errs.NewWrongNumArgumentsError(name, "1 or more", 0)
		}
		// the blank set: these members are about separators and filler, so the
		// no-argument form acts on the type's notion of insignificant content
		return &matchPlan[T]{pred: func(_ VM, _ int, e T) (bool, error) {
			if noArgMatchesBlank {
				return isBlank(e), nil
			}
			return !isBlank(e), nil
		}}, nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return nil, errs.NewInvalidArgumentTypeError(name, "arguments", "a single predicate (a function among several arguments has no reading)", "mixed")
		}
		pred, err := seqElementPred(name, args[0], t2v)
		if err != nil {
			return nil, err
		}
		return &matchPlan[T]{pred: pred}, nil
	}

	var elems []T
	var runArgs []Value
	for _, a := range args {
		if a.IsCallable() {
			return nil, errs.NewInvalidArgumentTypeError(name, "arguments", "one reading per call (a function among several arguments always raises)", "mixed")
		}
		if isRunArg(a) {
			runArgs = append(runArgs, a)
			continue
		}
		e, ok, err := toElem(a)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", "an element, a sequence of the receiver's kind, or a predicate", a.TypeName())
		}
		elems = append(elems, e)
	}
	if len(runArgs) > 0 && len(elems) > 0 {
		return nil, errs.NewInvalidArgumentTypeError(name, "arguments", "a HOMOGENEOUS set — every argument in one call must have the same reading (all elements, or all runs)", "mixed")
	}
	if len(runArgs) > 0 {
		if toRun == nil {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a predicate (this member declares no run reading)", runArgs[0].TypeName())
		}
		runs := make([][]T, 0, len(runArgs))
		for _, a := range runArgs {
			r, err := toRun(a)
			if err != nil {
				return nil, err
			}
			runs = append(runs, r)
		}
		return &matchPlan[T]{runs: runs}, nil
	}
	return &matchPlan[T]{pred: func(_ VM, _ int, e T) (bool, error) {
		for _, x := range elems {
			if eq(e, x) {
				return true, nil
			}
		}
		return false, nil
	}}, nil
}

// scanRuns walks elems left to right applying LEFTMOST-LONGEST, NON-OVERLAPPING
// run matching over the whole set: at each position the longest run in the set
// that matches wins (longest keeps the variadic form a true SET — the answer
// cannot depend on argument order), the scan resumes after it, and unmatched
// positions go to onMiss one element at a time.
func scanRuns[T any](elems []T, runs [][]T, eq func(T, T) bool, onMatch func(start, n int), onMiss func(i int)) {
	i := 0
	for i < len(elems) {
		best := longestRunAt(elems, i, runs, eq)
		if best > 0 {
			onMatch(i, best)
			i += best
			continue
		}
		if onMiss != nil {
			onMiss(i)
		}
		i++
	}
}

// longestRunAt reports the length of the longest non-empty run in the set matching at position i (0 = none) —
// the longest-match tie-break in one place, shared by the scanning members and the single-hit ones (partition).
func longestRunAt[T any](elems []T, i int, runs [][]T, eq func(T, T) bool) int {
	best := 0
	for _, r := range runs {
		if len(r) == 0 || len(r) <= best || i+len(r) > len(elems) {
			continue
		}
		match := true
		for j := range r {
			if !eq(elems[i+j], r[j]) {
				match = false
				break
			}
		}
		if match {
			best = len(r)
		}
	}
	return best
}

// SeqMatchMember implements contains/count/filter/remove/any/all over one
// dispatch. any/all refuse a run PERMANENTLY (the contiguous-run query is
// contains's, and "every element is this subsequence" has no universal
// reading); remove's no-argument form drops the blanks (the one destructive
// no-arg cell — documented, not guarded).
func SeqMatchMember[T any](
	vm VM,
	name string,
	v Value,
	args []Value,
	t2v func(T) Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	toElem func(Value) (T, bool, error),
	isRunArg func(Value) bool,
	toRun func(Value) ([]T, error),
	eq func(T, T) bool,
	isBlank func(T) bool,
) (Value, error) {
	// the _in_place twin runs the same dispatch and verb; the caller applies the mutation.
	// The full member name stays in every error message.
	verb := strings.TrimSuffix(name, "_in_place")
	quantifier := verb == "any" || verb == "all"
	runReader := toRun
	if quantifier {
		runReader = nil
	}
	plan, err := seqMatchDispatch(vm, name, args, t2v, toElem, isRunArg, runReader, eq, isBlank,
		verb == "remove", true)
	if err != nil {
		return Undefined, err
	}
	return applyMatchVerb(vm, verb, v, resolve(v), plan, alloc, eq)
}

// applyMatchVerb executes one match member over a resolved plan.
func applyMatchVerb[T any](
	vm VM,
	name string,
	v Value,
	o *Seq[T],
	plan *matchPlan[T],
	alloc func([]T, bool) Value,
	eq func(T, T) bool,
) (Value, error) {
	if plan.runs != nil {
		switch name {
		case "contains":
			// the empty run is contained everywhere (same answer `in` gives);
			// for the counting/keeping verbs it simply matches nothing
			for _, r := range plan.runs {
				if len(r) == 0 {
					return True, nil
				}
			}
			found := false
			scanRuns(o.Elements, plan.runs, eq, func(int, int) { found = true }, nil)
			return BoolValue(found), nil
		case "count":
			n := int64(0)
			scanRuns(o.Elements, plan.runs, eq, func(int, int) { n++ }, nil)
			return IntValue(n), nil
		case "filter":
			kept := make([]T, 0, len(o.Elements))
			scanRuns(o.Elements, plan.runs, eq, func(start, n int) { kept = append(kept, o.Elements[start:start+n]...) }, nil)
			return alloc(kept, false), nil
		case "remove":
			kept := make([]T, 0, len(o.Elements))
			scanRuns(o.Elements, plan.runs, eq, func(int, int) {}, func(i int) { kept = append(kept, o.Elements[i]) })
			return alloc(kept, false), nil
		}
	}

	switch name {
	case "contains", "any":
		for i, e := range o.Elements {
			t, err := plan.pred(vm, i, e)
			if err != nil {
				return Undefined, err
			}
			if t {
				return True, nil
			}
		}
		return False, nil
	case "all":
		for i, e := range o.Elements {
			t, err := plan.pred(vm, i, e)
			if err != nil {
				return Undefined, err
			}
			if !t {
				return False, nil
			}
		}
		return True, nil
	case "count":
		n := int64(0)
		for i, e := range o.Elements {
			t, err := plan.pred(vm, i, e)
			if err != nil {
				return Undefined, err
			}
			if t {
				n++
			}
		}
		return IntValue(n), nil
	case "filter", "remove":
		keepMatches := name == "filter"
		kept := make([]T, 0, len(o.Elements))
		for i, e := range o.Elements {
			t, err := plan.pred(vm, i, e)
			if err != nil {
				return Undefined, err
			}
			if t == keepMatches {
				kept = append(kept, e)
			}
		}
		return alloc(kept, false), nil
	}
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}

// TripleMatchMember is the match engine for the text triple, where acceptance
// collapses: every accepted argument is TEXT CONTENT, encoded into the
// receiver's representation and read as a run (a length-1 run is the element
// case). The element/run classes survive only for the homogeneity check of a
// variadic set, decided by argument TYPE (byte/rune/in-range int = element
// class; string/runes/bytes = run class), never by length.
func TripleMatchMember[T any](
	vm VM,
	name string,
	v Value,
	args []Value,
	t2v func(T) Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	encode func(name string, a Value) (run []T, elementClass bool, err error),
	eq func(T, T) bool,
	isBlank func(T) bool,
) (Value, error) {
	// the _in_place twin runs the same dispatch and verb; the caller applies the mutation.
	// The full member name stays in every error message.
	verb := strings.TrimSuffix(name, "_in_place")
	quantifier := verb == "any" || verb == "all"
	o := resolve(v)

	if len(args) == 0 {
		blankMatches := verb == "remove"
		plan := &matchPlan[T]{pred: func(_ VM, _ int, e T) (bool, error) {
			if blankMatches {
				return isBlank(e), nil
			}
			return !isBlank(e), nil
		}}
		return applyMatchVerb(vm, verb, v, o, plan, alloc, eq)
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "arguments", "a single predicate (a function among several arguments has no reading)", "mixed")
		}
		pred, err := seqElementPred(name, args[0], t2v)
		if err != nil {
			return Undefined, err
		}
		return applyMatchVerb(vm, verb, v, o, &matchPlan[T]{pred: pred}, alloc, eq)
	}

	runs := make([][]T, 0, len(args))
	sawElem, sawRun := false, false
	for _, a := range args {
		if a.IsCallable() {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "arguments", "one reading per call (a function among several arguments always raises)", "mixed")
		}
		run, elementClass, err := encode(name, a)
		if err != nil {
			return Undefined, err
		}
		if elementClass {
			sawElem = true
		} else {
			sawRun = true
		}
		runs = append(runs, run)
	}
	if sawElem && sawRun {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "arguments", "a HOMOGENEOUS set — every argument in one call must have the same reading (all elements, or all runs)", "mixed")
	}
	if quantifier {
		// any/all take a value, a function, or nothing; the run query is
		// contains's, and "every element is this subsequence" has no reading
		if sawRun {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "argument", "a value, a function, or nothing (the contiguous-run query is contains's)", "sequence")
		}
		elems := make([]T, 0, len(runs))
		for _, r := range runs {
			if len(r) != 1 {
				return Undefined, errs.NewInvalidValueError("(" + name + ") the value does not fit a single element of the receiver")
			}
			elems = append(elems, r[0])
		}
		plan := &matchPlan[T]{pred: func(_ VM, _ int, e T) (bool, error) {
			for _, x := range elems {
				if eq(e, x) {
					return true, nil
				}
			}
			return false, nil
		}}
		return applyMatchVerb(vm, verb, v, o, plan, alloc, eq)
	}
	return applyMatchVerb(vm, verb, v, o, &matchPlan[T]{runs: runs}, alloc, eq)
}

// ---------------------------------------------------------------------------
// The add side: append / prepend (whole-operand concatenation, operands kept
// in order) and push / push_first (validating element add).
// ---------------------------------------------------------------------------

// tripleAddItems flattens the add side's variadic operands (append/prepend) on a text receiver into the
// receiver's representation: every accepted argument is text content — an element contributes its encoding, a
// run its content — concatenated in ARGUMENT ORDER. Mixing element and run arguments is legal here
// (x.append("ab", 'c') means exactly x + "ab" + 'c'), unlike the match side's homogeneous set.
func tripleAddItems[T any](name string, args []Value, encode func(string, Value) ([]T, bool, error)) ([]T, error) {
	items := make([]T, 0, len(args))
	for _, a := range args {
		run, _, err := encode(name, a)
		if err != nil {
			return nil, err
		}
		items = append(items, run...)
	}
	return items, nil
}

// seqEncodeElementSet reads a strict ELEMENT set: each argument must encode to exactly ONE element of the
// receiver — a run-class argument (or a callable, where no predicate reading exists) raises with refuseMsg, the
// member's own statement of its menu; an element-class argument that widens (a multi-octet rune on an octet
// receiver) raises too. Serves push/push_first, the trim family's set, and the pads' fill.
func seqEncodeElementSet[T any](name string, args []Value, encode func(string, Value) ([]T, bool, error), refuseMsg string) ([]T, error) {
	items := make([]T, 0, len(args))
	for _, a := range args {
		if a.IsCallable() {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", refuseMsg, a.TypeName())
		}
		run, elementClass, err := encode(name, a)
		if err != nil {
			return nil, err
		}
		if !elementClass {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", refuseMsg, a.TypeName())
		}
		if len(run) != 1 {
			return nil, errs.NewInvalidValueError("(" + name + ") the value does not fit a single element of the receiver")
		}
		items = append(items, run[0])
	}
	return items, nil
}

// triplePushItems validates the element-add side (push/push_first) on a text receiver: each argument must
// encode to exactly ONE element of the receiver — a sequence argument raises even at length 1, and so does an
// element-class argument that widens (a multi-octet rune pushed onto bytes). The refusal is the member's
// purpose: bytes.push(x) is how a script says "x had better be a single octet" and gets told when it is not.
func triplePushItems[T any](name string, args []Value, encode func(string, Value) ([]T, bool, error)) ([]T, error) {
	return seqEncodeElementSet(name, args, encode,
		"one element (a sequence argument never reads as an element here; append/prepend take runs)")
}

// seqEncodeRunSet reads a homogeneous RUN set (every argument in one call has the
// same reading — all element-class or all run-class — and matching is over the encoded content either way, so
// an element contributes its encoding as one run; the class survives only for the homogeneity check). A
// callable raises with callableMsg — the member's own menu statement.
func seqEncodeRunSet[T any](name string, args []Value, encode func(string, Value) ([]T, bool, error), callableMsg string) ([][]T, error) {
	runs := make([][]T, 0, len(args))
	sawElem, sawRun := false, false
	for _, a := range args {
		if a.IsCallable() {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", callableMsg, a.TypeName())
		}
		run, elementClass, err := encode(name, a)
		if err != nil {
			return nil, err
		}
		if elementClass {
			sawElem = true
		} else {
			sawRun = true
		}
		runs = append(runs, run)
	}
	if sawElem && sawRun {
		return nil, errs.NewInvalidArgumentTypeError(name, "arguments", "a HOMOGENEOUS set — every argument in one call must have the same reading (all elements, or all runs)", "mixed")
	}
	return runs, nil
}

// locatorResult applies the uniform miss contract of the locators: absence answers undefined — never an in-band
// sentinel like -1, which negative indexing would silently accept — or the optional trailing default.
func locatorResult(idx int64, found bool, dflt []Value) (Value, error) {
	if found {
		return IntValue(idx), nil
	}
	if len(dflt) == 1 {
		return dflt[0], nil
	}
	return Undefined, nil
}

// SeqIndex is the locator: index([x[, default]]) / index_last([x[, default]]). One name, and the argument's
// type selects the reading — a function is a predicate, an argument of the receiver's own kind is a contiguous
// run (the caller supplies matchRun for that; nil means the reading is not available), no argument means the
// first/last SIGNIFICANT element (the blank set), anything else is one element compared with ==. Never
// variadic: the trailing slot is the default, and no type test could tell a second needle from a fallback.
func SeqIndex[T any](
	vm VM,
	v Value,
	args []Value,
	last bool,
	t2v func(T) Value,
	resolve func(Value) *Seq[T],
	isRun func(Value) bool, // is this argument the receiver's own kind?
	matchRun func(elems []T, run Value, last bool) (int64, bool, error), // nil: no run reading
	checkElem func(name string, a Value) error, // nil: any value is an element; else the receiver's acceptance — an unreadable needle RAISES, it is never a silent miss
	isBlank func(T) bool,
) (Value, error) {
	name := "index"
	if last {
		name = "index_last"
	}
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	o := resolve(v)

	// absent: locate by the blank set
	if len(args) == 0 {
		idx, found := int64(-1), false
		for i, e := range o.Elements {
			if !isBlank(e) {
				idx, found = int64(i), true
				if !last {
					break
				}
			}
		}
		return locatorResult(idx, found, nil)
	}

	needle := args[0]
	dflt := args[1:]

	// predicate
	if needle.IsCallable() {
		if arity := needle.Arity(); arity != 1 && arity != 2 {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", needle.TypeName())
		}
		idx, found := int64(-1), false
		var buf [2]Value
		for i, e := range o.Elements {
			if needle.Arity() == 2 {
				buf[0] = IntValue(int64(i))
				buf[1] = t2v(e)
			} else {
				buf[0] = t2v(e)
			}
			res, err := needle.Call(vm, buf[:needle.Arity()])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				idx, found = int64(i), true
				if !last {
					break
				}
			}
		}
		return locatorResult(idx, found, dflt)
	}

	// contiguous run: an argument of the receiver's own kind
	if isRun != nil && isRun(needle) {
		if matchRun == nil {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "an element or a predicate", needle.TypeName())
		}
		idx, found, err := matchRun(o.Elements, needle, last)
		if err != nil {
			return Undefined, err
		}
		return locatorResult(idx, found, dflt)
	}

	// element: == comparison — but the needle must be readable as an element of this
	// receiver at all: a value the receiver's acceptance refuses raises, exactly as it
	// does on contains/count, instead of scanning to a silent miss
	if checkElem != nil {
		if err := checkElem(name, needle); err != nil {
			return Undefined, err
		}
	}
	idx, found := int64(-1), false
	for i, e := range o.Elements {
		if t2v(e).Equal(needle) {
			idx, found = int64(i), true
			if !last {
				break
			}
		}
	}
	return locatorResult(idx, found, dflt)
}

// SeqIndexRun searches for a contiguous run by element equality — leftmost (or rightmost) match of the whole
// run, non-overlapping being irrelevant for a single locator.
func SeqIndexRun[T any](elems []T, run []T, t2v func(T) Value, last bool) (int64, bool) {
	n, m := len(elems), len(run)
	if m == 0 || m > n {
		return -1, false
	}
	idx, found := int64(-1), false
	for i := 0; i+m <= n; i++ {
		match := true
		for j := 0; j < m; j++ {
			if !t2v(elems[i+j]).Equal(t2v(run[j])) {
				match = false
				break
			}
		}
		if match {
			idx, found = int64(i), true
			if !last {
				break
			}
		}
	}
	return idx, found
}

// seqChunkCore divides the sequence into chunks of the specified size and returns a new array of chunks.
// forceCopy controls whether each chunk independently owns its storage (true) or shares backing storage with
// the source (false) — shared by SeqChunk (chunk(), where the caller picks via a bool arg — retiring that in
// favor of named twins is P4-002's job, not decided here) and SeqChunkView (chunk_view(), always false).
func seqChunkCore[T any](
	v Value,
	size int64,
	forceCopy bool,
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	o := resolve(v)
	l := len(o.Elements)
	if l == 0 {
		return NewArrayValue(make([]Value, 0), false), nil
	}

	chunkCount := int((int64(l)-1)/size + 1)
	chunks := make([]Value, chunkCount)

	chunkSize := l
	if size < int64(l) {
		chunkSize = int(size)
	}

	for i, start := 0, 0; start < l; i, start = i+1, start+chunkSize {
		end := min(start+chunkSize, l)
		chunk := o.Elements[start:end]
		chunkImmutable := v.Immutable
		if forceCopy {
			chunk = make([]T, end-start)
			copy(chunk, o.Elements[start:end])
			chunkImmutable = false
		}
		chunks[i] = alloc(chunk, chunkImmutable)
	}

	return NewArrayValue(chunks, false), nil
}

// SeqChunk divides the sequence into chunks of the specified size and returns a new sequence containing the
// chunks — always independently-owned copies (P4-002: the `copy` bool parameter is retired; chunk_view() is
// the explicit opt-in for sharing now, not a second argument here).
func SeqChunk[T any](
	v Value,
	args []Value,
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("chunk", "1", len(args))
	}

	size, err := parseIntArg("chunk", "first", args[0])
	if err != nil {
		return Undefined, err
	}
	if size < 1 {
		return Undefined, errs.NewInvalidValueError("chunk size must be positive")
	}

	return seqChunkCore(v, size, true, alloc, resolve)
}

// SeqChunkView is the `_view` twin of chunk(): always shares backing storage with the source (today's
// chunk(size, false) behavior), and marks every resulting chunk as a view via IsView.
func SeqChunkView[T any](
	v Value,
	args []Value,
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("chunk_view", "1", len(args))
	}

	size, err := parseIntArg("chunk_view", "first", args[0])
	if err != nil {
		return Undefined, err
	}
	if size < 1 {
		return Undefined, errs.NewInvalidValueError("chunk size must be positive")
	}

	res, err := seqChunkCore(v, size, false, alloc, resolve)
	if err != nil {
		return Undefined, err
	}
	for _, c := range (*Array)(res.Ptr).Elements {
		resolve(c).IsView = true
	}
	return res, nil
}

// seqOptionalSliceArgs translates the member-call form's 0, 1, or 2 optional (start, end) arguments into the
// two Value arguments the Slice hook expects (the Undefined sentinel standing in for an omitted bound, exactly
// like the `a[i:j]`/`a[i:]`/`a[:j]`/`a[:]` operator forms) — shared by slice() and slice_view().
func seqOptionalSliceArgs(name string, args []Value) (Value, Value, error) {
	if len(args) > 2 {
		return Undefined, Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	s, e := Undefined, Undefined
	if len(args) > 0 {
		s = args[0]
	}
	if len(args) > 1 {
		e = args[1]
	}
	return s, e, nil
}

// seqResolveSliceBounds parses and normalizes a (start, end) bound pair against the sequence's current length,
// shared by the copying Slice hook and the sharing SeqSliceView.
func seqResolveSliceBounds[T any](
	name string,
	v Value,
	s Value,
	e Value,
	resolve func(Value) *Seq[T], // T container resolver
) (o *Seq[T], si int64, ei int64, err error) {
	var ok bool

	o = resolve(v)
	l := int64(len(o.Elements))

	if s.Type != value.Undefined {
		si = int64(s.Data) // optimistic scenario
		if s.Type != value.Int {
			if si, ok = s.AsInt(); !ok {
				return nil, 0, 0, errs.NewInvalidIndexTypeError(name, "int", s.TypeName())
			}
		}
	}

	if e.Type != value.Undefined {
		ei = int64(e.Data) // optimistic scenario
		if e.Type != value.Int {
			if ei, ok = e.AsInt(); !ok {
				return nil, 0, 0, errs.NewInvalidIndexTypeError(name, "int", e.TypeName())
			}
		}
	}

	si, ei = NormalizeSliceBounds(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, l)
	return o, si, ei, nil
}

// SeqSlice is the member-call form of two-part slicing (`x.slice(start, end)`), reusing the same (now
// always-copying, per P4-002) Slice hook the `a[i:j]` operator uses — same operation, second spelling, per
// Rule 10. Accepts 0, 1, or 2 args, mirroring the operator's own optional start/end.
func SeqSlice(v Value, args []Value) (Value, error) {
	s, e, err := seqOptionalSliceArgs("slice", args)
	if err != nil {
		return Undefined, err
	}
	return v.Slice(s, e)
}

// SeqSliceView is the `_view` twin of two-part slicing: shares backing storage with the source via a raw
// re-slice — the sharing behavior `a[i:j]` itself had before P4-002, preserved here as the explicit opt-in.
// Marked as a view via IsView. Accepts 0, 1, or 2 args, mirroring the operator's own optional start/end.
func SeqSliceView[T any](
	v Value,
	args []Value,
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
) (Value, error) {
	s, e, err := seqOptionalSliceArgs("slice_view", args)
	if err != nil {
		return Undefined, err
	}
	o, si, ei, err := seqResolveSliceBounds("slice_view", v, s, e, resolve)
	if err != nil {
		return Undefined, err
	}
	res := alloc(o.Elements[si:ei], v.Immutable)
	resolve(res).IsView = true
	return res, nil
}

// SeqSplice implements splice()/splice_in_place() for any Seq[T]-backed type (array/bytes/runes). args[0] is the
// receiver value — kept as the first positional arg (rather than a separate v Value parameter) so the
// "argument first/second/third" error wording, established when array was the only type this ran for, stays
// identical now that bytes/runes share this same function (P5-002). convertItems turns the trailing variadic
// args (from index 3 on) into []T: array's is a plain identity (elements are already Values, no conversion or
// flattening), while bytes'/runes' reuse the same flattening-and-type-checking logic append() already uses
// (bytesAppendItems/runesAppendItems), so passing a bytes/runes value as one of splice's insert items spreads it
// exactly like it does for append(), rather than erroring or nesting it as a single opaque element.
// mutate=true: IMPURE, mutates the receiver in place and returns the deleted items (splice_in_place()) — rejects
// an immutable receiver. mutate=false: PURE, returns the modified sequence instead of the deleted items
// (splice()), never touching the receiver — works regardless of the receiver's mutability. See docs/purity.md.
func SeqSplice[T any](
	args []Value,
	mutate bool,
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
	convertItems func(args []Value, methodName string) ([]T, error),
	typeName string, // used in the "mutable <typeName>" argument-type error
) (Value, error) {
	argsLen := len(args)
	if argsLen == 0 {
		return Undefined, errs.NewWrongNumArgumentsError("splice", "at least 1", argsLen)
	}
	if mutate && args[0].Immutable {
		return Undefined, errs.NewNotMutableError("splice_in_place", args[0].TypeName())
	}

	o := resolve(args[0])
	seqLen := len(o.Elements)

	var startIdx int
	if argsLen > 1 {
		arg1, ok := args[1].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError("splice", "second", "int", args[1].TypeName())
		}
		startIdx = int(arg1)
		if startIdx < 0 {
			// negative indices count from the end, like every positional slot
			startIdx += seqLen
		}
		if startIdx < 0 || startIdx > seqLen {
			return Undefined, errs.NewIndexOutOfBoundsError("splice, start index", int(arg1), seqLen)
		}
	}

	delCount := seqLen
	if argsLen > 2 {
		arg2, ok := args[2].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError("splice", "third", "int", args[2].TypeName())
		}
		if arg2 < 0 {
			return Undefined, errs.NewRecoverableError(errs.KindInvalidValue, "splice delete count must be non-negative")
		}
		// Clamp before converting to avoid signed integer overflow when computing startIdx+delCount.
		if arg2 > int64(seqLen-startIdx) {
			delCount = seqLen - startIdx
		} else {
			delCount = int(arg2)
		}
	} else if startIdx+delCount > seqLen {
		// no count given; default to "from startIdx to end"
		delCount = seqLen - startIdx
	}
	endIdx := startIdx + delCount

	var newItems []T
	if argsLen > 3 {
		var err error
		newItems, err = convertItems(args[3:], "splice")
		if err != nil {
			return Undefined, err
		}
	}

	if mutate {
		head := o.Elements[:startIdx]
		items := append(newItems, o.Elements[endIdx:]...)
		o.Set(append(head, items...))
		// a side-effecting member returns the RECEIVER — mutators chain and the
		// twins correspond (y = x.m(...) and x.m_in_place(...) leave the same
		// content in x). The removed run is x.slice(i, j) taken beforehand.
		return args[0], nil
	}

	// Pure: build a fresh, independent sequence — never touch o's own backing storage (per docs/conventions.md's
	// variadic/slice argument immutability rule; append(receiver, ...) would risk writing into o's own array).
	result := make([]T, 0, startIdx+len(newItems)+(seqLen-endIdx))
	result = append(result, o.Elements[:startIdx]...)
	result = append(result, newItems...)
	result = append(result, o.Elements[endIdx:]...)
	return alloc(result, false), nil
}

// SeqNameHook returns a hook function that provides the type name for the sequence based on its mutability.
func SeqNameHook(
	name string, // mutable type name
	immutableName string, // immutable type name
) func(Value) string {
	return func(v Value) string {
		if v.Immutable {
			return immutableName
		}
		return name
	}
}

// IMPURE: returned Assign hook writes into the receiver. Not folded by the optimizer. See docs/purity.md.
//
// SeqAssignHook returns a hook function that allows assigning a value to an element of the sequence at a specified
// index.
func SeqAssignHook[T any](
	resolve func(Value) *Seq[T], // T container resolver
	as func(Value) (T, bool), // Value to T convertor
	tn string, // T type name
) func(Value, Value, Value, bc.Opcode) error {
	return func(v Value, index Value, r Value, _ bc.Opcode) error {
		if v.Immutable {
			return errs.NewNotAssignableError(v.TypeName())
		}

		i := int64(index.Data) // optimistic scenario
		var ok bool
		if index.Type != value.Int {
			if i, ok = index.AsInt(); !ok {
				return errs.NewInvalidIndexTypeError("index assign", "int", index.TypeName())
			}
		}

		o := resolve(v)
		l := len(o.Elements)
		if i, ok = NormalizeIndex(i, int64(l)); !ok {
			return errs.NewIndexOutOfBoundsError("index assign", int(i), l)
		}

		c, ok := as(r)
		if !ok {
			return errs.NewInvalidIndexTypeError("index assign value", tn, r.TypeName())
		}

		o.Elements[i] = c

		return nil
	}
}

// SeqAccessHook returns a hook function that allows accessing an element of the sequence at a specified index.
// PURE by contract.
func SeqAccessHook[T any](
	t2v func(T) Value, // T type constructor
	resolve func(Value) *Seq[T], // T container resolver
) func(Value, Value, bc.Opcode) (Value, error) {
	return func(v Value, index Value, mode bc.Opcode) (Value, error) {
		if mode != bc.AccessIndex {
			return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
		}

		i := int64(index.Data) // optimistic scenario
		var ok bool
		if index.Type != value.Int {
			if i, ok = index.AsInt(); !ok {
				return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
			}
		}

		o := resolve(v)
		l := len(o.Elements)
		if i, ok = NormalizeIndex(i, int64(l)); !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), l)
		}

		return t2v(o.Elements[i]), nil
	}
}

// SeqSliceHook returns a hook function that allows slicing the sequence using start and end indices. Always
// returns an independently-owned copy of the selected range (P4-002, closing P01/P02) — sharing backing
// storage with the source is SeqSliceView's job now (the explicit `_view` twin), not this hook's.
// PURE by contract.
func SeqSliceHook[T any](
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
) func(Value, Value, Value) (Value, error) {
	return func(v Value, s Value, e Value) (Value, error) {
		o, si, ei, err := seqResolveSliceBounds("slice", v, s, e, resolve)
		if err != nil {
			return Undefined, err
		}
		out := make([]T, ei-si)
		copy(out, o.Elements[si:ei])
		return alloc(out, false), nil
	}
}

// SeqSliceStepHook returns a hook function that allows slicing the sequence using start and end indices with a
// specified step. PURE by contract.
func SeqSliceStepHook[T any](
	alloc func([]T, bool) Value, // T container allocator
	resolve func(Value) *Seq[T], // T container resolver
) func(Value, Value, Value, Value) (Value, error) {
	return func(v Value, s Value, e Value, stepVal Value) (Value, error) {
		var step, si, ei int64
		var ok bool

		o := resolve(v)
		l := int64(len(o.Elements))

		step = int64(stepVal.Data) // optimistic scenario
		if stepVal.Type != value.Int {
			if step, ok = stepVal.AsInt(); !ok {
				return Undefined, errs.NewInvalidIndexTypeError("slice step", "int", stepVal.TypeName())
			}
		}
		if step == 0 {
			return Undefined, errs.NewSliceStepZeroError()
		}

		if s.Type != value.Undefined {
			si = int64(s.Data) // optimistic scenario
			if s.Type != value.Int {
				if si, ok = s.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
				}
			}
		}
		if e.Type != value.Undefined {
			ei = int64(e.Data) // optimistic scenario
			if e.Type != value.Int {
				if ei, ok = e.AsInt(); !ok {
					return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
				}
			}
		}

		start, end := NormalizeSliceBoundsStep(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, step, l)
		result := make([]T, 0)
		if step > 0 {
			for i := start; i < end; i += step {
				result = append(result, o.Elements[i])
			}
		} else {
			for i := start; i > end; i += step {
				result = append(result, o.Elements[i])
			}
		}

		return alloc(result, false), nil
	}
}

// ---------------------------------------------------------------------------
// The text-structural family: split / partition, the trim family, the
// anchored pair (has_/remove_ prefix/suffix), replace, and the pads. These
// are SEQUENCE members, not text members — their definitions mention a fill
// element, an element set, and a run, never text — so array carries every one
// of them except split/partition/split_lines, which stay the text triple's.
// ---------------------------------------------------------------------------

// immutableTwinError is the uniform refusal when an _in_place twin meets an immutable receiver.
func immutableTwinError(name, typeName string) error {
	return errs.NewNotMutableError(name, typeName)
}

// SeqSplitMember implements split(...seps): separator element | run | homogeneous set | element-level
// predicate | absent = the blank set. Explicit separators keep empty pieces between adjacent hits (n matches
// answer n+1 pieces, so an empty receiver splits into one empty piece); the blank form answers the maximal runs
// of SIGNIFICANT content — the blanks are filler, and there is no piece between two of them — which is what
// preserves the classic no-argument whitespace split. Runs match leftmost-longest, non-overlapping; an empty
// separator run matches nothing.
func SeqSplitMember[T any](
	vm VM,
	name string,
	v Value,
	args []Value,
	t2v func(T) Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	encode func(name string, a Value) (run []T, elementClass bool, err error),
	eq func(T, T) bool,
	isBlank func(T) bool,
) (Value, error) {
	elems := resolve(v).Elements

	if len(args) == 0 {
		var pieces []Value
		start := -1
		for i, e := range elems {
			if isBlank(e) {
				if start >= 0 {
					pieces = append(pieces, alloc(slices.Clone(elems[start:i]), false))
					start = -1
				}
			} else if start < 0 {
				start = i
			}
		}
		if start >= 0 {
			pieces = append(pieces, alloc(slices.Clone(elems[start:]), false))
		}
		return NewArrayValue(pieces, false), nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "arguments", "a single predicate (a function among several arguments has no reading)", "mixed")
		}
		pred, err := seqElementPred(name, args[0], t2v)
		if err != nil {
			return Undefined, err
		}
		pieces := make([]Value, 0, 4)
		start := 0
		for i, e := range elems {
			hit, err := pred(vm, i, e)
			if err != nil {
				return Undefined, err
			}
			if hit {
				pieces = append(pieces, alloc(slices.Clone(elems[start:i]), false))
				start = i + 1
			}
		}
		pieces = append(pieces, alloc(slices.Clone(elems[start:]), false))
		return NewArrayValue(pieces, false), nil
	}

	runs, err := seqEncodeRunSet(name, args, encode, "one reading per call (a function among several arguments always raises)")
	if err != nil {
		return Undefined, err
	}
	pieces := make([]Value, 0, 4)
	start := 0
	scanRuns(elems, runs, eq, func(s, n int) {
		pieces = append(pieces, alloc(slices.Clone(elems[start:s]), false))
		start = s + n
	}, nil)
	pieces = append(pieces, alloc(slices.Clone(elems[start:]), false))
	return NewArrayValue(pieces, false), nil
}

// SeqPartitionMember implements partition(...seps): the one-split form — [before, separator, after], the
// separator as matched; a miss answers [receiver, empty, empty]. Same separator menu as split; the leftmost hit
// wins, the longest at that position (the set stays order-independent). The blank no-argument form takes the
// whole maximal run of filler as the separator, exactly the run split's derivation collapses on one hit.
func SeqPartitionMember[T any](
	vm VM,
	name string,
	v Value,
	args []Value,
	t2v func(T) Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	encode func(name string, a Value) (run []T, elementClass bool, err error),
	eq func(T, T) bool,
	isBlank func(T) bool,
) (Value, error) {
	elems := resolve(v).Elements
	found, n := -1, 0

	switch {
	case len(args) == 0:
		for i, e := range elems {
			if isBlank(e) {
				found, n = i, 1
				for found+n < len(elems) && isBlank(elems[found+n]) {
					n++
				}
				break
			}
		}

	case args[0].IsCallable():
		if len(args) > 1 {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "arguments", "a single predicate (a function among several arguments has no reading)", "mixed")
		}
		pred, err := seqElementPred(name, args[0], t2v)
		if err != nil {
			return Undefined, err
		}
		for i, e := range elems {
			hit, err := pred(vm, i, e)
			if err != nil {
				return Undefined, err
			}
			if hit {
				found, n = i, 1
				break
			}
		}

	default:
		runs, err := seqEncodeRunSet(name, args, encode, "one reading per call (a function among several arguments always raises)")
		if err != nil {
			return Undefined, err
		}
		for i := range elems {
			if best := longestRunAt(elems, i, runs, eq); best > 0 {
				found, n = i, best
				break
			}
		}
	}

	if found < 0 {
		return NewArrayValue([]Value{alloc(slices.Clone(elems), false), alloc([]T{}, false), alloc([]T{}, false)}, false), nil
	}
	return NewArrayValue([]Value{
		alloc(slices.Clone(elems[:found]), false),
		alloc(slices.Clone(elems[found:found+n]), false),
		alloc(slices.Clone(elems[found+n:]), false),
	}, false), nil
}

// SeqTrimMember implements trim(...set) / trim_start / trim_end: drop leading/trailing elements while they
// belong to the set — repeat-while, ELEMENTS only. A run argument raises (the anchored exact-run form is
// remove_prefix/remove_suffix — people who write trim(run) mean that), and so does a predicate; no argument
// means the blank set.
func SeqTrimMember[T any](
	name string,
	v Value,
	args []Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	encode func(name string, a Value) (run []T, elementClass bool, err error),
	eq func(T, T) bool,
	isBlank func(T) bool,
	start bool,
	end bool,
) (Value, error) {
	inSet := isBlank
	if len(args) > 0 {
		set, err := seqEncodeElementSet(name, args, encode,
			"a set of elements (the anchored run form is remove_prefix/remove_suffix; no predicate reading)")
		if err != nil {
			return Undefined, err
		}
		inSet = func(e T) bool {
			for _, x := range set {
				if eq(e, x) {
					return true
				}
			}
			return false
		}
	}
	elems := resolve(v).Elements
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
	return alloc(slices.Clone(elems[lo:hi]), false), nil
}

// SeqAnchoredMember implements has_prefix/has_suffix (a bool: is any run in the set anchored there?) and
// remove_prefix/remove_suffix (remove one exact anchored run, ONCE; absent from the receiver → unchanged; the
// longest matching run in a variadic set wins, keeping the set order-independent). Element | run | homogeneous
// set; no predicate ("the first element satisfies f" is index(f) == 0) and no no-argument form. The empty run
// is anchored everywhere: has_prefix on it answers true, remove_prefix on it removes nothing.
func SeqAnchoredMember[T any](
	name string,
	v Value,
	args []Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	encode func(name string, a Value) (run []T, elementClass bool, err error),
	eq func(T, T) bool,
	suffix bool,
	remove bool,
) (Value, error) {
	if len(args) == 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1 or more", 0)
	}
	runs, err := seqEncodeRunSet(name, args, encode,
		"an element or a run (no predicate reading — \"the first element satisfies f\" is index(f) == 0)")
	if err != nil {
		return Undefined, err
	}
	elems := resolve(v).Elements
	best := -1
	for _, r := range runs {
		if len(r) > len(elems) || len(r) <= best {
			continue
		}
		off := 0
		if suffix {
			off = len(elems) - len(r)
		}
		match := true
		for j := range r {
			if !eq(elems[off+j], r[j]) {
				match = false
				break
			}
		}
		if match {
			best = len(r)
		}
	}
	if !remove {
		return BoolValue(best >= 0), nil
	}
	if best <= 0 {
		return alloc(slices.Clone(elems), false), nil
	}
	if suffix {
		return alloc(slices.Clone(elems[:len(elems)-best]), false), nil
	}
	return alloc(slices.Clone(elems[best:]), false), nil
}

// SeqReplaceMember implements replace(old, new): element or run in BOTH positions, each argument read by its
// own type (the two slots have different roles, so no homogeneity constraint applies between them); every
// occurrence, leftmost non-overlapping. Never variadic — position 2 is the replacement — and never a predicate.
// An empty old run matches nothing: the receiver comes back unchanged.
func SeqReplaceMember[T any](
	name string,
	v Value,
	args []Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	encode func(name string, a Value) (run []T, elementClass bool, err error),
	eq func(T, T) bool,
) (Value, error) {
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	if args[0].IsCallable() || args[1].IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a run (replace is never a predicate)", "function")
	}
	old, _, err := encode(name, args[0])
	if err != nil {
		return Undefined, err
	}
	repl, _, err := encode(name, args[1])
	if err != nil {
		return Undefined, err
	}
	elems := resolve(v).Elements
	out := make([]T, 0, len(elems))
	scanRuns(elems, [][]T{old}, eq,
		func(int, int) { out = append(out, repl...) },
		func(i int) { out = append(out, elems[i]) })
	return alloc(out, false), nil
}

// SeqPadMember implements pad_start(n[, fill]) / pad_end(n[, fill]): n counts ELEMENTS, the fill is exactly one
// element — a run fill raises, because cycling a multi-element fill hides a truncation rule — and its default
// is the blank set's canonical member. A width at or below the length is a no-op.
func SeqPadMember[T any](
	name string,
	v Value,
	args []Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	fill func(name string, a Value) (T, error),
	defaultFill T,
	start bool,
) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1 or 2", len(args))
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	f := defaultFill
	if len(args) == 2 {
		var err error
		f, err = fill(name, args[1])
		if err != nil {
			return Undefined, err
		}
	}
	elems := resolve(v).Elements
	if n <= int64(len(elems)) {
		return alloc(slices.Clone(elems), false), nil
	}
	out := make([]T, 0, int(n))
	pad := int(n) - len(elems)
	if start {
		for range pad {
			out = append(out, f)
		}
		out = append(out, elems...)
	} else {
		out = append(out, elems...)
		for range pad {
			out = append(out, f)
		}
	}
	return alloc(out, false), nil
}

// tripleElemCheck adapts a text receiver's acceptance encoder into the locators' needle gate: the needle must
// be one element of the receiver (a widening int/byte raises its range error; a non-text value raises the
// acceptance error) — the run and predicate readings are dispatched before this is consulted.
func tripleElemCheck[T any](encode func(string, Value) ([]T, bool, error)) func(string, Value) error {
	return func(name string, a Value) error {
		run, elementClass, err := encode(name, a)
		if err != nil {
			return err
		}
		if !elementClass || len(run) != 1 {
			return errs.NewInvalidValueError("(" + name + ") the value does not fit a single element of the receiver")
		}
		return nil
	}
}

// tripleFillElement adapts a text receiver's acceptance encoder into the pads' one-element fill reader.
func tripleFillElement[T any](encode func(string, Value) ([]T, bool, error)) func(string, Value) (T, error) {
	return func(name string, a Value) (T, error) {
		var zero T
		items, err := seqEncodeElementSet(name, []Value{a}, encode,
			"one fill element (a run fill hides a truncation rule; build the run and append it instead)")
		if err != nil {
			return zero, err
		}
		return items[0], nil
	}
}

// SeqStructuralMember routes one structural member — or the pure half of its _in_place twin — by verb. The
// caller strips nothing: the full member name reaches every error message; only the verb switch ignores the
// suffix. has_prefix/has_suffix answer a bool and so have no twins; the caller's case list enforces that.
func SeqStructuralMember[T any](
	name string,
	v Value,
	args []Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	encode func(name string, a Value) (run []T, elementClass bool, err error),
	fill func(name string, a Value) (T, error),
	defaultFill T,
	eq func(T, T) bool,
	isBlank func(T) bool,
) (Value, error) {
	switch strings.TrimSuffix(name, "_in_place") {
	case "trim":
		return SeqTrimMember(name, v, args, alloc, resolve, encode, eq, isBlank, true, true)
	case "trim_start":
		return SeqTrimMember(name, v, args, alloc, resolve, encode, eq, isBlank, true, false)
	case "trim_end":
		return SeqTrimMember(name, v, args, alloc, resolve, encode, eq, isBlank, false, true)
	case "has_prefix":
		return SeqAnchoredMember(name, v, args, alloc, resolve, encode, eq, false, false)
	case "has_suffix":
		return SeqAnchoredMember(name, v, args, alloc, resolve, encode, eq, true, false)
	case "remove_prefix":
		return SeqAnchoredMember(name, v, args, alloc, resolve, encode, eq, false, true)
	case "remove_suffix":
		return SeqAnchoredMember(name, v, args, alloc, resolve, encode, eq, true, true)
	case "replace":
		return SeqReplaceMember(name, v, args, alloc, resolve, encode, eq)
	case "pad_start":
		return SeqPadMember(name, v, args, alloc, resolve, fill, defaultFill, true)
	case "pad_end":
		return SeqPadMember(name, v, args, alloc, resolve, fill, defaultFill, false)
	}
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}

// ---------------------------------------------------------------------------
// map / flat_map. map is strictly 1:1 and returns the RECEIVER'S type; on a
// text receiver every callback result must be exactly one element (an
// in-range int/byte/rune IS the element) — a sequence or undefined result
// raises, because widening or dropping is flat_map's job. flat_map is
// map-then-concatenate: each callback result is read like an add-side
// operand — a run concatenates, undefined contributes nothing, anything
// else is one element.
// ---------------------------------------------------------------------------

// seqMapCallback validates map/flat_map's single callback argument and returns the call closure:
// f/1 receives the element, f/2 receives (locator, element).
func seqMapCallback(name string, args []Value) (func(vm VM, i int, elem Value) (Value, error), error) {
	if len(args) != 1 {
		return nil, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	fn := args[0]
	if !fn.IsCallable() {
		return nil, errs.NewInvalidArgumentTypeError(name, "first", "function", fn.TypeName())
	}
	arity := fn.Arity()
	if arity != 1 && arity != 2 {
		return nil, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
	return func(vm VM, i int, elem Value) (Value, error) {
		var buf [2]Value
		n := 1
		if arity >= 2 {
			buf[0] = IntValue(int64(i))
			buf[1] = elem
			n = 2
		} else {
			buf[0] = elem
		}
		return fn.Call(vm, buf[:n])
	}, nil
}

// TripleMapMember: map on a text receiver — strictly 1:1, answering the receiver's type.
func TripleMapMember[T any](
	vm VM,
	name string,
	v Value,
	args []Value,
	t2v func(T) Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	encode func(name string, a Value) (run []T, elementClass bool, err error),
) (Value, error) {
	call, err := seqMapCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	o := resolve(v)
	out := make([]T, len(o.Elements))
	for i, e := range o.Elements {
		res, err := call(vm, i, t2v(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback answered undefined — map is 1:1; the dropping form is flat_map")
		}
		run, elementClass, err := encode(name, res)
		if err != nil {
			return Undefined, err
		}
		if !elementClass {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback answered a sequence — map is 1:1; the concatenating form is flat_map")
		}
		if len(run) != 1 {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback result does not fit a single element of the receiver")
		}
		out[i] = run[0]
	}
	return alloc(out, false), nil
}

// TripleFlatMapMember: flat_map on a text receiver — each callback result is text content read as a run
// (a single element is a length-1 run), undefined contributes nothing.
func TripleFlatMapMember[T any](
	vm VM,
	name string,
	v Value,
	args []Value,
	t2v func(T) Value,
	alloc func([]T, bool) Value,
	resolve func(Value) *Seq[T],
	encode func(name string, a Value) (run []T, elementClass bool, err error),
) (Value, error) {
	call, err := seqMapCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	o := resolve(v)
	out := make([]T, 0, len(o.Elements))
	for i, e := range o.Elements {
		res, err := call(vm, i, t2v(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			continue
		}
		run, _, err := encode(name, res)
		if err != nil {
			return Undefined, err
		}
		out = append(out, run...)
	}
	return alloc(out, false), nil
}

// seqEditPos reads a positional-edit slot (insert's position): int-valued and lossless, negative counting
// from the end, and — editing past the end is not harmless, unlike reading — out of [0, len] raises.
func seqEditPos(name string, args []Value, length int64) (int64, error) {
	if len(args) == 0 {
		return 0, errs.NewWrongNumArgumentsError(name, "1 or more", 0)
	}
	i, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return 0, err
	}
	orig := i
	if i < 0 {
		i += length
	}
	if i < 0 || i > length {
		return 0, errs.NewIndexOutOfBoundsError(name, int(orig), int(length))
	}
	return i, nil
}
