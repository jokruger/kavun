package kavungen

import (
	"fmt"
	"strings"
)

// diffContext is the number of unchanged lines shown around each change.
const diffContext = 3

// unifiedDiff renders the line difference between a and b in unified format. It is meant for a human reading a
// test failure or a -check report, not for patch(1): it uses a plain longest-common-subsequence table, which is
// fine for files of a few thousand lines.
func unifiedDiff(nameA, nameB, a, b string) string {
	x, y := splitLines(a), splitLines(b)

	// lcs[i][j] is the length of the longest common subsequence of x[i:] and y[j:].
	lcs := make([][]int, len(x)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
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
		case i < len(x) && j < len(y) && x[i] == y[j]:
			ops = append(ops, op{' ', x[i], i, j})
			i, j = i+1, j+1
		case j < len(y) && (i == len(x) || lcs[i][j+1] >= lcs[i+1][j]):
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
