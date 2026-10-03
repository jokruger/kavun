package benchmarks

// Member-function dispatch: string switch vs integer ID vs per-type table.
//
// Every builtin type resolves a member call with `switch name` on the method's string. This benchmark exists to
// keep the evidence for why that is fine (see NOTES.md). The label set below is a snapshot of core/runes.go's
// MethodCall switch (71 labels, 2026-10-03) — the exact membership does not matter, only that there are ~70 labels
// of realistic lengths. The weights are how often each member appears in kavun_test.go (+1), a proxy for usage.
//
// Three selector distributions are measured: one name only (a hot loop calling the same member — what a real
// call site looks like), names weighted by usage, and uniform random. The random rows are branch-misprediction
// bound and every strategy pays them; they are the worst case, not a program's shape.
//
// BenchmarkMemberCall measures the whole in-language call: `len(a)` is the control (builtins dispatch through an
// integer slot with no string anywhere), `a.len()` goes through Value.MethodCall and the 50-label array switch,
// and `s.has_prefix("ab")` is a structural member whose cost is its body, not its dispatch.

import (
	"math/rand"
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/vm"
)

type methodID uint16

const (
	mUnknown methodID = iota
	m_all
	m_any
	m_append
	m_append_in_place
	m_array
	m_bool
	m_bytes
	m_camel_case
	m_case_fold
	m_chunk
	m_chunk_view
	m_contains
	m_copy
	m_count
	m_date
	m_decimal
	m_dedup
	m_dedup_in_place
	m_first
	m_flat_map
	m_float
	m_for_each
	m_format
	m_freeze
	m_index
	m_index_last
	m_insert
	m_insert_in_place
	m_int
	m_is_ascii
	m_is_empty
	m_is_valid
	m_kebab_case
	m_keep
	m_keep_in_place
	m_last
	m_len
	m_lower
	m_map
	m_max
	m_min
	m_partition
	m_pascal_case
	m_prepend
	m_prepend_in_place
	m_push
	m_push_first
	m_push_first_in_place
	m_push_in_place
	m_reduce
	m_remove
	m_remove_in_place
	m_repeat
	m_reverse
	m_reverse_in_place
	m_runes
	m_slice
	m_slice_view
	m_snake_case
	m_sort
	m_sort_in_place
	m_splice
	m_splice_in_place
	m_split
	m_split_lines
	m_string
	m_time
	m_title_case
	m_unique
	m_unique_in_place
	m_upper
	mCount
)

// methodNames is the label set; methodWeights the usage proxy.
var methodNames = []string{
	"all",
	"any",
	"append",
	"append_in_place",
	"array",
	"bool",
	"bytes",
	"camel_case",
	"case_fold",
	"chunk",
	"chunk_view",
	"contains",
	"copy",
	"count",
	"date",
	"decimal",
	"dedup",
	"dedup_in_place",
	"first",
	"flat_map",
	"float",
	"for_each",
	"format",
	"freeze",
	"index",
	"index_last",
	"insert",
	"insert_in_place",
	"int",
	"is_ascii",
	"is_empty",
	"is_valid",
	"kebab_case",
	"keep",
	"keep_in_place",
	"last",
	"len",
	"lower",
	"map",
	"max",
	"min",
	"partition",
	"pascal_case",
	"prepend",
	"prepend_in_place",
	"push",
	"push_first",
	"push_first_in_place",
	"push_in_place",
	"reduce",
	"remove",
	"remove_in_place",
	"repeat",
	"reverse",
	"reverse_in_place",
	"runes",
	"slice",
	"slice_view",
	"snake_case",
	"sort",
	"sort_in_place",
	"splice",
	"splice_in_place",
	"split",
	"split_lines",
	"string",
	"time",
	"title_case",
	"unique",
	"unique_in_place",
	"upper",
}

var methodWeights = map[string]int{
	"string":              209,
	"format":              136,
	"append":              83,
	"array":               72,
	"index":               68,
	"len":                 67,
	"int":                 63,
	"repeat":              53,
	"splice_in_place":     51,
	"any":                 50,
	"contains":            46,
	"bytes":               42,
	"split":               41,
	"keep":                41,
	"append_in_place":     39,
	"map":                 38,
	"slice_view":          36,
	"sort":                34,
	"all":                 34,
	"splice":              31,
	"chunk":               29,
	"time":                28,
	"dedup":               27,
	"slice":               26,
	"reverse":             26,
	"for_each":            25,
	"push":                24,
	"bool":                23,
	"unique":              22,
	"partition":           22,
	"runes":               21,
	"remove":              21,
	"freeze":              21,
	"remove_in_place":     20,
	"reduce":              19,
	"min":                 18,
	"float":               18,
	"date":                16,
	"chunk_view":          16,
	"insert":              15,
	"first":               15,
	"max":                 14,
	"is_empty":            14,
	"copy":                14,
	"last":                13,
	"count":               13,
	"prepend":             12,
	"sort_in_place":       11,
	"is_valid":            11,
	"flat_map":            11,
	"case_fold":           11,
	"split_lines":         10,
	"reverse_in_place":    10,
	"decimal":             10,
	"lower":               9,
	"keep_in_place":       9,
	"is_ascii":            9,
	"title_case":          8,
	"snake_case":          8,
	"prepend_in_place":    8,
	"upper":               7,
	"push_in_place":       7,
	"push_first":          6,
	"insert_in_place":     6,
	"index_last":          6,
	"unique_in_place":     4,
	"push_first_in_place": 4,
	"dedup_in_place":      4,
	"kebab_case":          3,
	"pascal_case":         2,
	"camel_case":          2,
}

// methodIDOf is the load-time resolver (name -> id); it is NOT on the per-call path of any variant.
var methodIDOf = map[string]methodID{
	"all":                 m_all,
	"any":                 m_any,
	"append":              m_append,
	"append_in_place":     m_append_in_place,
	"array":               m_array,
	"bool":                m_bool,
	"bytes":               m_bytes,
	"camel_case":          m_camel_case,
	"case_fold":           m_case_fold,
	"chunk":               m_chunk,
	"chunk_view":          m_chunk_view,
	"contains":            m_contains,
	"copy":                m_copy,
	"count":               m_count,
	"date":                m_date,
	"decimal":             m_decimal,
	"dedup":               m_dedup,
	"dedup_in_place":      m_dedup_in_place,
	"first":               m_first,
	"flat_map":            m_flat_map,
	"float":               m_float,
	"for_each":            m_for_each,
	"format":              m_format,
	"freeze":              m_freeze,
	"index":               m_index,
	"index_last":          m_index_last,
	"insert":              m_insert,
	"insert_in_place":     m_insert_in_place,
	"int":                 m_int,
	"is_ascii":            m_is_ascii,
	"is_empty":            m_is_empty,
	"is_valid":            m_is_valid,
	"kebab_case":          m_kebab_case,
	"keep":                m_keep,
	"keep_in_place":       m_keep_in_place,
	"last":                m_last,
	"len":                 m_len,
	"lower":               m_lower,
	"map":                 m_map,
	"max":                 m_max,
	"min":                 m_min,
	"partition":           m_partition,
	"pascal_case":         m_pascal_case,
	"prepend":             m_prepend,
	"prepend_in_place":    m_prepend_in_place,
	"push":                m_push,
	"push_first":          m_push_first,
	"push_first_in_place": m_push_first_in_place,
	"push_in_place":       m_push_in_place,
	"reduce":              m_reduce,
	"remove":              m_remove,
	"remove_in_place":     m_remove_in_place,
	"repeat":              m_repeat,
	"reverse":             m_reverse,
	"reverse_in_place":    m_reverse_in_place,
	"runes":               m_runes,
	"slice":               m_slice,
	"slice_view":          m_slice_view,
	"snake_case":          m_snake_case,
	"sort":                m_sort,
	"sort_in_place":       m_sort_in_place,
	"splice":              m_splice,
	"splice_in_place":     m_splice_in_place,
	"split":               m_split,
	"split_lines":         m_split_lines,
	"string":              m_string,
	"time":                m_time,
	"title_case":          m_title_case,
	"unique":              m_unique,
	"unique_in_place":     m_unique_in_place,
	"upper":               m_upper,
}

// memberBody stands in for a member's implementation: tiny, so the measurement is the dispatch step.
//
//go:noinline
func memberBody(k, nargs int) int { return k + nargs }

// dispatchStringSwitch is today's shape: switch on the name, body inline.
func dispatchStringSwitch(name string, nargs int) int {
	switch name {
	case "all":
		return memberBody(1, nargs)
	case "any":
		return memberBody(2, nargs)
	case "append":
		return memberBody(3, nargs)
	case "append_in_place":
		return memberBody(4, nargs)
	case "array":
		return memberBody(5, nargs)
	case "bool":
		return memberBody(6, nargs)
	case "bytes":
		return memberBody(7, nargs)
	case "camel_case":
		return memberBody(8, nargs)
	case "case_fold":
		return memberBody(9, nargs)
	case "chunk":
		return memberBody(10, nargs)
	case "chunk_view":
		return memberBody(11, nargs)
	case "contains":
		return memberBody(12, nargs)
	case "copy":
		return memberBody(13, nargs)
	case "count":
		return memberBody(14, nargs)
	case "date":
		return memberBody(15, nargs)
	case "decimal":
		return memberBody(16, nargs)
	case "dedup":
		return memberBody(17, nargs)
	case "dedup_in_place":
		return memberBody(18, nargs)
	case "first":
		return memberBody(19, nargs)
	case "flat_map":
		return memberBody(20, nargs)
	case "float":
		return memberBody(21, nargs)
	case "for_each":
		return memberBody(22, nargs)
	case "format":
		return memberBody(23, nargs)
	case "freeze":
		return memberBody(24, nargs)
	case "index":
		return memberBody(25, nargs)
	case "index_last":
		return memberBody(26, nargs)
	case "insert":
		return memberBody(27, nargs)
	case "insert_in_place":
		return memberBody(28, nargs)
	case "int":
		return memberBody(29, nargs)
	case "is_ascii":
		return memberBody(30, nargs)
	case "is_empty":
		return memberBody(31, nargs)
	case "is_valid":
		return memberBody(32, nargs)
	case "kebab_case":
		return memberBody(33, nargs)
	case "keep":
		return memberBody(34, nargs)
	case "keep_in_place":
		return memberBody(35, nargs)
	case "last":
		return memberBody(36, nargs)
	case "len":
		return memberBody(37, nargs)
	case "lower":
		return memberBody(38, nargs)
	case "map":
		return memberBody(39, nargs)
	case "max":
		return memberBody(40, nargs)
	case "min":
		return memberBody(41, nargs)
	case "partition":
		return memberBody(42, nargs)
	case "pascal_case":
		return memberBody(43, nargs)
	case "prepend":
		return memberBody(44, nargs)
	case "prepend_in_place":
		return memberBody(45, nargs)
	case "push":
		return memberBody(46, nargs)
	case "push_first":
		return memberBody(47, nargs)
	case "push_first_in_place":
		return memberBody(48, nargs)
	case "push_in_place":
		return memberBody(49, nargs)
	case "reduce":
		return memberBody(50, nargs)
	case "remove":
		return memberBody(51, nargs)
	case "remove_in_place":
		return memberBody(52, nargs)
	case "repeat":
		return memberBody(53, nargs)
	case "reverse":
		return memberBody(54, nargs)
	case "reverse_in_place":
		return memberBody(55, nargs)
	case "runes":
		return memberBody(56, nargs)
	case "slice":
		return memberBody(57, nargs)
	case "slice_view":
		return memberBody(58, nargs)
	case "snake_case":
		return memberBody(59, nargs)
	case "sort":
		return memberBody(60, nargs)
	case "sort_in_place":
		return memberBody(61, nargs)
	case "splice":
		return memberBody(62, nargs)
	case "splice_in_place":
		return memberBody(63, nargs)
	case "split":
		return memberBody(64, nargs)
	case "split_lines":
		return memberBody(65, nargs)
	case "string":
		return memberBody(66, nargs)
	case "time":
		return memberBody(67, nargs)
	case "title_case":
		return memberBody(68, nargs)
	case "unique":
		return memberBody(69, nargs)
	case "unique_in_place":
		return memberBody(70, nargs)
	case "upper":
		return memberBody(71, nargs)
	default:
		return -1
	}
}

// dispatchIDSwitch: dense uint16 constants (a jump table), body inline.
func dispatchIDSwitch(id methodID, nargs int) int {
	switch id {
	case m_all:
		return memberBody(1, nargs)
	case m_any:
		return memberBody(2, nargs)
	case m_append:
		return memberBody(3, nargs)
	case m_append_in_place:
		return memberBody(4, nargs)
	case m_array:
		return memberBody(5, nargs)
	case m_bool:
		return memberBody(6, nargs)
	case m_bytes:
		return memberBody(7, nargs)
	case m_camel_case:
		return memberBody(8, nargs)
	case m_case_fold:
		return memberBody(9, nargs)
	case m_chunk:
		return memberBody(10, nargs)
	case m_chunk_view:
		return memberBody(11, nargs)
	case m_contains:
		return memberBody(12, nargs)
	case m_copy:
		return memberBody(13, nargs)
	case m_count:
		return memberBody(14, nargs)
	case m_date:
		return memberBody(15, nargs)
	case m_decimal:
		return memberBody(16, nargs)
	case m_dedup:
		return memberBody(17, nargs)
	case m_dedup_in_place:
		return memberBody(18, nargs)
	case m_first:
		return memberBody(19, nargs)
	case m_flat_map:
		return memberBody(20, nargs)
	case m_float:
		return memberBody(21, nargs)
	case m_for_each:
		return memberBody(22, nargs)
	case m_format:
		return memberBody(23, nargs)
	case m_freeze:
		return memberBody(24, nargs)
	case m_index:
		return memberBody(25, nargs)
	case m_index_last:
		return memberBody(26, nargs)
	case m_insert:
		return memberBody(27, nargs)
	case m_insert_in_place:
		return memberBody(28, nargs)
	case m_int:
		return memberBody(29, nargs)
	case m_is_ascii:
		return memberBody(30, nargs)
	case m_is_empty:
		return memberBody(31, nargs)
	case m_is_valid:
		return memberBody(32, nargs)
	case m_kebab_case:
		return memberBody(33, nargs)
	case m_keep:
		return memberBody(34, nargs)
	case m_keep_in_place:
		return memberBody(35, nargs)
	case m_last:
		return memberBody(36, nargs)
	case m_len:
		return memberBody(37, nargs)
	case m_lower:
		return memberBody(38, nargs)
	case m_map:
		return memberBody(39, nargs)
	case m_max:
		return memberBody(40, nargs)
	case m_min:
		return memberBody(41, nargs)
	case m_partition:
		return memberBody(42, nargs)
	case m_pascal_case:
		return memberBody(43, nargs)
	case m_prepend:
		return memberBody(44, nargs)
	case m_prepend_in_place:
		return memberBody(45, nargs)
	case m_push:
		return memberBody(46, nargs)
	case m_push_first:
		return memberBody(47, nargs)
	case m_push_first_in_place:
		return memberBody(48, nargs)
	case m_push_in_place:
		return memberBody(49, nargs)
	case m_reduce:
		return memberBody(50, nargs)
	case m_remove:
		return memberBody(51, nargs)
	case m_remove_in_place:
		return memberBody(52, nargs)
	case m_repeat:
		return memberBody(53, nargs)
	case m_reverse:
		return memberBody(54, nargs)
	case m_reverse_in_place:
		return memberBody(55, nargs)
	case m_runes:
		return memberBody(56, nargs)
	case m_slice:
		return memberBody(57, nargs)
	case m_slice_view:
		return memberBody(58, nargs)
	case m_snake_case:
		return memberBody(59, nargs)
	case m_sort:
		return memberBody(60, nargs)
	case m_sort_in_place:
		return memberBody(61, nargs)
	case m_splice:
		return memberBody(62, nargs)
	case m_splice_in_place:
		return memberBody(63, nargs)
	case m_split:
		return memberBody(64, nargs)
	case m_split_lines:
		return memberBody(65, nargs)
	case m_string:
		return memberBody(66, nargs)
	case m_time:
		return memberBody(67, nargs)
	case m_title_case:
		return memberBody(68, nargs)
	case m_unique:
		return memberBody(69, nargs)
	case m_unique_in_place:
		return memberBody(70, nargs)
	case m_upper:
		return memberBody(71, nargs)
	default:
		return -1
	}
}

// methodTable: a per-type table indexed by global ID — the "declarative table" design; one indirect call.
var methodTable = func() []func(int) int {
	t := make([]func(int) int, mCount)
	t[m_all] = func(nargs int) int { return memberBody(1, nargs) }
	t[m_any] = func(nargs int) int { return memberBody(2, nargs) }
	t[m_append] = func(nargs int) int { return memberBody(3, nargs) }
	t[m_append_in_place] = func(nargs int) int { return memberBody(4, nargs) }
	t[m_array] = func(nargs int) int { return memberBody(5, nargs) }
	t[m_bool] = func(nargs int) int { return memberBody(6, nargs) }
	t[m_bytes] = func(nargs int) int { return memberBody(7, nargs) }
	t[m_camel_case] = func(nargs int) int { return memberBody(8, nargs) }
	t[m_case_fold] = func(nargs int) int { return memberBody(9, nargs) }
	t[m_chunk] = func(nargs int) int { return memberBody(10, nargs) }
	t[m_chunk_view] = func(nargs int) int { return memberBody(11, nargs) }
	t[m_contains] = func(nargs int) int { return memberBody(12, nargs) }
	t[m_copy] = func(nargs int) int { return memberBody(13, nargs) }
	t[m_count] = func(nargs int) int { return memberBody(14, nargs) }
	t[m_date] = func(nargs int) int { return memberBody(15, nargs) }
	t[m_decimal] = func(nargs int) int { return memberBody(16, nargs) }
	t[m_dedup] = func(nargs int) int { return memberBody(17, nargs) }
	t[m_dedup_in_place] = func(nargs int) int { return memberBody(18, nargs) }
	t[m_first] = func(nargs int) int { return memberBody(19, nargs) }
	t[m_flat_map] = func(nargs int) int { return memberBody(20, nargs) }
	t[m_float] = func(nargs int) int { return memberBody(21, nargs) }
	t[m_for_each] = func(nargs int) int { return memberBody(22, nargs) }
	t[m_format] = func(nargs int) int { return memberBody(23, nargs) }
	t[m_freeze] = func(nargs int) int { return memberBody(24, nargs) }
	t[m_index] = func(nargs int) int { return memberBody(25, nargs) }
	t[m_index_last] = func(nargs int) int { return memberBody(26, nargs) }
	t[m_insert] = func(nargs int) int { return memberBody(27, nargs) }
	t[m_insert_in_place] = func(nargs int) int { return memberBody(28, nargs) }
	t[m_int] = func(nargs int) int { return memberBody(29, nargs) }
	t[m_is_ascii] = func(nargs int) int { return memberBody(30, nargs) }
	t[m_is_empty] = func(nargs int) int { return memberBody(31, nargs) }
	t[m_is_valid] = func(nargs int) int { return memberBody(32, nargs) }
	t[m_kebab_case] = func(nargs int) int { return memberBody(33, nargs) }
	t[m_keep] = func(nargs int) int { return memberBody(34, nargs) }
	t[m_keep_in_place] = func(nargs int) int { return memberBody(35, nargs) }
	t[m_last] = func(nargs int) int { return memberBody(36, nargs) }
	t[m_len] = func(nargs int) int { return memberBody(37, nargs) }
	t[m_lower] = func(nargs int) int { return memberBody(38, nargs) }
	t[m_map] = func(nargs int) int { return memberBody(39, nargs) }
	t[m_max] = func(nargs int) int { return memberBody(40, nargs) }
	t[m_min] = func(nargs int) int { return memberBody(41, nargs) }
	t[m_partition] = func(nargs int) int { return memberBody(42, nargs) }
	t[m_pascal_case] = func(nargs int) int { return memberBody(43, nargs) }
	t[m_prepend] = func(nargs int) int { return memberBody(44, nargs) }
	t[m_prepend_in_place] = func(nargs int) int { return memberBody(45, nargs) }
	t[m_push] = func(nargs int) int { return memberBody(46, nargs) }
	t[m_push_first] = func(nargs int) int { return memberBody(47, nargs) }
	t[m_push_first_in_place] = func(nargs int) int { return memberBody(48, nargs) }
	t[m_push_in_place] = func(nargs int) int { return memberBody(49, nargs) }
	t[m_reduce] = func(nargs int) int { return memberBody(50, nargs) }
	t[m_remove] = func(nargs int) int { return memberBody(51, nargs) }
	t[m_remove_in_place] = func(nargs int) int { return memberBody(52, nargs) }
	t[m_repeat] = func(nargs int) int { return memberBody(53, nargs) }
	t[m_reverse] = func(nargs int) int { return memberBody(54, nargs) }
	t[m_reverse_in_place] = func(nargs int) int { return memberBody(55, nargs) }
	t[m_runes] = func(nargs int) int { return memberBody(56, nargs) }
	t[m_slice] = func(nargs int) int { return memberBody(57, nargs) }
	t[m_slice_view] = func(nargs int) int { return memberBody(58, nargs) }
	t[m_snake_case] = func(nargs int) int { return memberBody(59, nargs) }
	t[m_sort] = func(nargs int) int { return memberBody(60, nargs) }
	t[m_sort_in_place] = func(nargs int) int { return memberBody(61, nargs) }
	t[m_splice] = func(nargs int) int { return memberBody(62, nargs) }
	t[m_splice_in_place] = func(nargs int) int { return memberBody(63, nargs) }
	t[m_split] = func(nargs int) int { return memberBody(64, nargs) }
	t[m_split_lines] = func(nargs int) int { return memberBody(65, nargs) }
	t[m_string] = func(nargs int) int { return memberBody(66, nargs) }
	t[m_time] = func(nargs int) int { return memberBody(67, nargs) }
	t[m_title_case] = func(nargs int) int { return memberBody(68, nargs) }
	t[m_unique] = func(nargs int) int { return memberBody(69, nargs) }
	t[m_unique_in_place] = func(nargs int) int { return memberBody(70, nargs) }
	t[m_upper] = func(nargs int) int { return memberBody(71, nargs) }
	return t
}()

const workMask = 4095

type dispatchWork struct {
	names []string
	ids   []methodID
}

func makeWork(pick func(*rand.Rand) string) dispatchWork {
	r := rand.New(rand.NewSource(1))
	w := dispatchWork{names: make([]string, workMask+1), ids: make([]methodID, workMask+1)}
	for i := range w.names {
		n := pick(r)
		w.names[i] = n
		w.ids[i] = methodIDOf[n]
	}
	return w
}

func weightedPool() []string {
	var pool []string
	for _, n := range methodNames {
		for k := 0; k < methodWeights[n]; k++ {
			pool = append(pool, n)
		}
	}
	return pool
}

var dispatchSink int

func BenchmarkDispatch(b *testing.B) {
	pool := weightedPool()
	workloads := []struct {
		name string
		w    dispatchWork
	}{
		{"mono", makeWork(func(*rand.Rand) string { return "len" })},
		{"weighted", makeWork(func(r *rand.Rand) string { return pool[r.Intn(len(pool))] })},
		{"uniform", makeWork(func(r *rand.Rand) string { return methodNames[r.Intn(len(methodNames))] })},
	}
	for _, wl := range workloads {
		b.Run(wl.name+"/string_switch", func(b *testing.B) {
			acc := 0
			for i := 0; i < b.N; i++ {
				acc += dispatchStringSwitch(wl.w.names[i&workMask], i&1)
			}
			dispatchSink = acc
		})
		b.Run(wl.name+"/id_switch", func(b *testing.B) {
			acc := 0
			for i := 0; i < b.N; i++ {
				acc += dispatchIDSwitch(wl.w.ids[i&workMask], i&1)
			}
			dispatchSink = acc
		})
		b.Run(wl.name+"/id_table", func(b *testing.B) {
			acc := 0
			for i := 0; i < b.N; i++ {
				acc += methodTable[wl.w.ids[i&workMask]](i & 1)
			}
			dispatchSink = acc
		})
	}
	b.Run("baseline_direct_call", func(b *testing.B) {
		acc := 0
		for i := 0; i < b.N; i++ {
			acc += memberBody(i&63, i&1)
		}
		dispatchSink = acc
	})
}

// BenchmarkMemberCall: the in-language cost. Subtract `empty` and divide by memberCallIters for ns per call.
const memberCallIters = 100_000

func BenchmarkMemberCall(b *testing.B) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	for _, sc := range []struct{ name, body string }{
		{"empty", `x = i`},
		{"len_builtin", `x = len(a)`},
		{"arr_len", `x = a.len()`},
		{"str_has_prefix", `x = s.has_prefix("ab")`},
	} {
		src := "x := 0\nfor i := 0; i < N; i++ {\n\t" + sc.body + "\n}\nout = x\n"
		compiled, err := kavun.NewScript([]byte(src), "out", "N", "a", "s").Compile()
		if err != nil {
			b.Fatalf("%s: %v", sc.name, err)
		}
		b.Run(sc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				compiled.Reset()
				compiled.Set("N", core.IntValue(memberCallIters))
				compiled.Set("a", core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false))
				compiled.Set("s", core.NewStringValue("abcdef"))
				if err := compiled.Run(machine); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
