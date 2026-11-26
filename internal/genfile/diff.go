package genfile

import (
	"fmt"
	"strings"
)

// diffContext is the number of unchanged lines shown around each change.
const diffContext = 3

// UnifiedDiff renders the line difference between a and b in unified format. It is meant for a human reading a
// test failure or a -check report, not for patch(1): it uses a plain longest-common-subsequence table over the
// lines between the common head and tail, which is fine for a few thousand changed lines in a file of any length.
func UnifiedDiff(nameA, nameB, a, b string) string {
	x, y := splitLines(a), splitLines(b)

	// the unchanged head and tail stay out of the table: lo lines in front, hi lines at the end
	lo := 0
	for lo < len(x) && lo < len(y) && x[lo] == y[lo] {
		lo++
	}
	hi := 0
	for hi < len(x)-lo && hi < len(y)-lo && x[len(x)-1-hi] == y[len(y)-1-hi] {
		hi++
	}
	mx, my := len(x)-hi, len(y)-hi // the changed middle is x[lo:mx], y[lo:my]

	// lcs[i-lo][j-lo] is the length of the longest common subsequence of x[i:mx] and y[j:my].
	lcs := make([][]int, mx-lo+1)
	for i := range lcs {
		lcs[i] = make([]int, my-lo+1)
	}
	for i := mx - 1; i >= lo; i-- {
		for j := my - 1; j >= lo; j-- {
			if x[i] == y[j] {
				lcs[i-lo][j-lo] = lcs[i-lo+1][j-lo+1] + 1
			} else {
				lcs[i-lo][j-lo] = max(lcs[i-lo+1][j-lo], lcs[i-lo][j-lo+1])
			}
		}
	}

	// ops is the edit script: ' ' keep, '-' only in a, '+' only in b, with each line's numbers in a and b.
	type op struct {
		kind   byte
		text   string
		ai, bi int
	}
	var ops []op
	i, j := 0, 0
	for i < len(x) || j < len(y) {
		switch {
		case i < lo || i >= mx && j >= my: // the common head or tail
			ops = append(ops, op{' ', x[i], i, j})
			i, j = i+1, j+1
		case i >= mx: // a is through its middle: the rest of b's changed
			ops = append(ops, op{'+', y[j], i, j})
			j++
		case j >= my:
			ops = append(ops, op{'-', x[i], i, j})
			i++
		case x[i] == y[j]:
			ops = append(ops, op{' ', x[i], i, j})
			i, j = i+1, j+1
		case lcs[i-lo][j-lo+1] >= lcs[i-lo+1][j-lo]:
			ops = append(ops, op{'+', y[j], i, j})
			j++
		default:
			ops = append(ops, op{'-', x[i], i, j})
			i++
		}
	}

	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", nameA, nameB)
	for k := 0; k < len(ops); {
		if ops[k].kind == ' ' {
			k++
			continue
		}
		// a hunk: from diffContext lines before this change to diffContext lines after the last change that is
		// no more than 2*diffContext unchanged lines away
		start := max(0, k-diffContext)
		end := k
		for end < len(ops) {
			if ops[end].kind != ' ' {
				end++
				continue
			}
			run := end
			for run < len(ops) && ops[run].kind == ' ' {
				run++
			}
			if run == len(ops) || run-end > 2*diffContext {
				end = min(len(ops), end+diffContext)
				break
			}
			end = run
		}
		na, nb := 0, 0
		for _, o := range ops[start:end] {
			if o.kind != '+' {
				na++
			}
			if o.kind != '-' {
				nb++
			}
		}
		fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n", ops[start].ai+1, na, ops[start].bi+1, nb)
		for _, o := range ops[start:end] {
			fmt.Fprintf(&out, "%c%s\n", o.kind, o.text)
		}
		k = end
	}
	return out.String()
}

// splitLines splits text into lines without their newlines; empty text has no lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
