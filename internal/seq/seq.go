package seq

import (
	"fmt"
	"slices"

	"github.com/jokruger/kavun/errs"
)

// NormalizeIndex normalizes index (-1 = last element, -2 = second to last, etc.) and checks if it's within bounds.
func NormalizeIndex(index int64, length int64) (int64, bool) {
	if index < 0 {
		index += length
	}
	if index < 0 || index >= length {
		return index, false
	}
	return index, true
}

// NormalizeSliceBounds normalizes slice bounds (negative values count from the end, missing start defaults to 0,
// missing end defaults to length) and clamps them to [0, length]. If start > end after normalization, start is set to
// end.
func NormalizeSliceBounds(start int64, hasStart bool, end int64, hasEnd bool, length int64) (int64, int64) {
	if !hasStart {
		start = 0
	} else if start < 0 {
		start += length
	}

	if !hasEnd {
		end = length
	} else if end < 0 {
		end += length
	}

	if start < 0 {
		start = 0
	} else if start > length {
		start = length
	}

	if end < 0 {
		end = 0
	} else if end > length {
		end = length
	}

	if start > end {
		start = end
	}

	return start, end
}

// NormalizeSliceBoundsStep returns the effective start and end for a step-based slice.
// Caller must ensure step != 0. For step > 0 the iteration is start..end (exclusive).
// For step < 0 the iteration is start..end (exclusive, with end possibly -1 to include index 0).
func NormalizeSliceBoundsStep(si int64, hasStart bool, ei int64, hasEnd bool, step int64, length int64) (int64, int64) {
	var start, end int64
	if step > 0 {
		if !hasStart {
			start = 0
		} else {
			start = si
			if start < 0 {
				start += length
			}
			if start < 0 {
				start = 0
			} else if start > length {
				start = length
			}
		}
		if !hasEnd {
			end = length
		} else {
			end = ei
			if end < 0 {
				end += length
			}
			if end < 0 {
				end = 0
			} else if end > length {
				end = length
			}
		}
	} else {
		// step < 0: lower bound is -1, upper bound is length-1
		if !hasStart {
			start = length - 1
		} else {
			start = si
			if start < 0 {
				start += length
			}
			if start < -1 {
				start = -1
			} else if start >= length {
				start = length - 1
			}
		}
		if !hasEnd {
			end = -1
		} else {
			end = ei
			if end < 0 {
				end += length
			}
			if end < -1 {
				end = -1
			} else if end >= length {
				end = length - 1
			}
		}
	}
	return start, end
}

// ArgOrdinal names the argument at zero-based position i the way every argument error does ("first",
// "second", …); past the tenth, where a variadic member can still be, it falls back to "11th", "22nd", ….
// PURE by contract.
func ArgOrdinal(i int) string {
	words := [...]string{"first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth"}
	if i >= 0 && i < len(words) {
		return words[i]
	}
	n := i + 1
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

// RunLengthAt reports the length of the longest non-empty run in runs that matches elems at position i (0 when
// none does). Longest wins so that a set of runs is a true SET: the answer never depends on argument order.
func RunLengthAt[T comparable](elems []T, i int, runs [][]T) int {
	best := 0
	for _, r := range runs {
		if len(r) <= best || i+len(r) > len(elems) {
			continue
		}
		if slices.Equal(elems[i:i+len(r)], r) {
			best = len(r)
		}
	}
	return best
}

// IndexRun answers the position of the first (or, with last, the final) occurrence of run in elems, or -1. The
// empty run is never found: a locator needs a position, and "everywhere" is not one.
func IndexRun[T comparable](elems []T, run []T, last bool) int {
	n, m := len(elems), len(run)
	if m == 0 || m > n {
		return -1
	}
	idx := -1
	for i := 0; i+m <= n; i++ {
		if slices.Equal(elems[i:i+m], run) {
			idx = i
			if !last {
				break
			}
		}
	}
	return idx
}

// RemoveRun answers a new slice holding elems with every occurrence of run removed, leftmost and non-overlapping.
// The empty run removes nothing. elems is never modified.
func RemoveRun[T comparable](elems []T, run []T) []T {
	out := make([]T, 0, len(elems))
	m := len(run)
	for i := 0; i < len(elems); {
		if m > 0 && i+m <= len(elems) && slices.Equal(elems[i:i+m], run) {
			i += m
			continue
		}
		out = append(out, elems[i])
		i++
	}
	return out
}

// ErrPredicateAmongMany: a function was passed together with other arguments to a member that reads a single
// function as a predicate.
func ErrPredicateAmongMany(name string) error {
	return errs.NewInvalidArgumentTypeError(name, "arguments", "a single predicate (a function among several arguments has no reading)", "mixed")
}

// ErrFunctionInSet: a function appeared later in a variadic set of values.
func ErrFunctionInSet(name string) error {
	return errs.NewInvalidArgumentTypeError(name, "arguments", "one reading per call (a function among several arguments always raises)", "mixed")
}

// ErrMixedSet: a variadic set mixed elements and runs.
func ErrMixedSet(name string) error {
	return errs.NewInvalidArgumentTypeError(name, "arguments", "a HOMOGENEOUS set — every argument in one call must have the same reading (all elements, or all runs)", "mixed")
}

// ErrNotOneElement: a value that had to be exactly one element of the receiver encodes to more (or fewer).
func ErrNotOneElement(name string) error {
	return errs.NewInvalidValueError("(" + name + ") the value does not fit a single element of the receiver")
}

// MaxSequenceLen bounds every count-driven sequence allocation (`repeat`, the `*` operator, `pad_start` and
// `pad_end`). Go's makeslice PANICS — not raises — for a length it cannot represent, and a Go panic escaping
// into the host is exactly what the error model forbids, so the count is checked against this ceiling first
// and answers a catchable error instead. The value is far past any real script and below makeslice's own
// limit for every element type Kavun has (byte, rune, Value), which makes that panic unreachable through
// these paths.
const MaxSequenceLen = 1 << 32

// RepeatTotal computes len(receiver) * count for a repeat, raising rather than overflowing or panicking.
// PURE by contract.
func RepeatTotal(name string, n, elems int) (int, error) {
	if n != 0 && elems > MaxSequenceLen/n {
		// argument validation must be catchable by recover()
		return 0, errs.NewInvalidValueError(fmt.Sprintf(
			"(%s) result would be %d × %d elements, past the %d limit", name, elems, n, MaxSequenceLen))
	}
	return n * elems, nil
}

// PadWidth reads a pad's target WIDTH as an allocation size, raising rather than panicking. It is the same
// ceiling as `repeat`'s, checked directly instead of as a product: a pad's width IS the resulting element
// count. The caller must have handled the no-op case (a width at or below the length) first, so what reaches
// here is always a real allocation. PURE by contract.
func PadWidth(name string, n int64) (int, error) {
	return AllocLen(name, n)
}

// AllocLen reads an element count as an allocation size, raising rather than panicking in makeslice when it
// is past MaxSequenceLen. It is the guard for every path that materialises a lazily-described sequence (a range's
// elements) as well as for the pad width above. PURE by contract.
func AllocLen(name string, n int64) (int, error) {
	if n > MaxSequenceLen {
		// argument validation must be catchable by recover()
		return 0, errs.NewInvalidValueError(fmt.Sprintf(
			"(%s) result would be %d elements, past the %d limit", name, n, MaxSequenceLen))
	}
	return int(n), nil
}
