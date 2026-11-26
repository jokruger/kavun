package kavungen

import (
	"go/token"
	"regexp"
	"slices"
	"strings"
)

// The Markdown of doc blocks: the checks that keep a doc block's body well-formed Markdown and stable under gofmt
// (checkMarkdown), and the one rewrite the reference's text blocks get (renderBody), which undoes what gofmt does to
// a fenced block written in its code-block form, so that gofmt on a source file never changes the reference (nor the
// pages cmd/kavun-docs renders from it).

// fenceSpan is one fenced block of a doc body: the indexes of its opening and closing fence lines (close is -1 when
// the body ends first).
type fenceSpan struct {
	open, close int
}

// fenceSpans answers the fenced blocks of a body. Only backtick fences are recognised, unindented: a line of three
// or more backticks opens one (an info string may follow), and, as in CommonMark, only a line of at least as many
// backticks and nothing else closes it — a shorter run inside is content. checkMarkdown and renderBody both read
// the body through it, so what is checked is what is rendered.
func fenceSpans(lines []string) []fenceSpan {
	var spans []fenceSpan
	for i := 0; i < len(lines); i++ {
		n := backtickRun(lines[i])
		if n < 3 {
			continue
		}
		span := fenceSpan{open: i, close: -1}
		for j := i + 1; j < len(lines); j++ {
			if m := backtickRun(lines[j]); m >= n && strings.TrimSpace(lines[j][m:]) == "" {
				span.close = j
				break
			}
		}
		spans = append(spans, span)
		if span.close < 0 {
			break
		}
		i = span.close
	}
	return spans
}

// backtickRun answers the number of backticks a line starts with.
func backtickRun(line string) int {
	return len(line) - len(strings.TrimLeft(line, "`"))
}

// topHeading matches a `#` or `##` Markdown heading line.
var topHeading = regexp.MustCompile(`^##?( |$)`)

// checkMarkdown applies the rules that keep a doc block's Markdown well-formed on the page and stable under gofmt,
// which reformats the doc comment of a declaration: no line indented outside a fenced block (gofmt would make it a
// Go code block, Markdown a code block), fenced blocks with backticks only, closed, and indented in a way gofmt
// keeps (checkFence). A `#` or `##` heading is the page's own structure: it is allowed only in a page root's block
// (allowTop), never in an entry's body. Each problem is reported at its line; ok is false when there was
// any.
func checkMarkdown(lines []string, pos []token.Position, allowTop bool, errs *errorList) bool {
	ok := true
	fenced := make([]bool, len(lines)) // the fence lines and the content of every fenced block
	for _, sp := range fenceSpans(lines) {
		end := sp.close
		if end < 0 {
			errs.add(pos[sp.open], "unclosed fenced block: the doc block ends before a closing line of at least %d backticks", backtickRun(lines[sp.open]))
			ok = false
			end = len(lines) - 1
		} else {
			ok = checkFence(lines[sp.open+1:sp.close], pos[sp.open+1:sp.close], errs) && ok
		}
		if lang, noRun, _ := fenceInfo(lines[sp.open]); lang == "go" && !noRun && strings.TrimSpace(strings.TrimLeft(lines[sp.open], "`")) != "go" {
			errs.add(pos[sp.open], "a `go` fence takes nothing after the language but the `%s` marker of an example that is not run (```go %s [reason])", NoRun, NoRun)
			ok = false
		}
		for i := sp.open; i <= end; i++ {
			fenced[i] = true
		}
	}
	for i, line := range lines {
		switch {
		case fenced[i]:
		case strings.HasPrefix(line, "~~~"):
			errs.add(pos[i], "a `~~~` fence: only backtick fences are recognised (write ``` instead)")
			ok = false
		case isIndented(line):
			errs.add(pos[i], "doc line is indented outside a fenced block: gofmt and Markdown would both read it as a code block (lists are single-level, their continuation lines unindented)")
			ok = false
		case !allowTop && topHeading.MatchString(line):
			errs.add(pos[i], "a `#` or `##` heading belongs to the page's structure (the type, module or builtins block): an entry's body uses `####` and deeper")
			ok = false
		}
	}
	return ok
}

// checkFence checks the content lines of one fenced block. Either no line is indented, or the block is in gofmt's
// code-block form: every non-blank line starts with one tab, and at least one has nothing else before its text
// (gofmt strips indentation common to every line, so a block indented further throughout would lose it).
func checkFence(lines []string, pos []token.Position, errs *errorList) bool {
	firstTab, firstPlain, flush := -1, -1, false
	for i, line := range lines {
		switch {
		case isBlank(line):
		case line[0] == ' ':
			errs.add(pos[i], "doc line inside a fenced block is indented with spaces: gofmt would turn it into a Go code block; put a tab before every line of the block instead (`//<tab>…`, gofmt's code-block form)")
			return false
		case line[0] == '\t':
			if firstTab < 0 {
				firstTab = i
			}
			flush = flush || !isIndented(line[1:])
		default:
			if firstPlain < 0 {
				firstPlain = i
			}
		}
	}
	switch {
	case firstTab >= 0 && firstPlain >= 0:
		errs.add(pos[max(firstTab, firstPlain)], "a fenced block mixes tab-indented and unindented lines: gofmt would split it; put a tab before every line of the block, or before none")
		return false
	case firstTab >= 0 && !flush:
		errs.add(pos[firstTab], "every line of a tab-indented fenced block is indented past its tab: gofmt would strip the common indentation; let the least indented line start right after the tab")
		return false
	}
	return true
}

func isIndented(line string) bool {
	return line != "" && (line[0] == ' ' || line[0] == '\t') && !isBlank(line)
}

func isBlank(line string) bool {
	return strings.TrimSpace(line) == ""
}

// renderBody answers a doc body as Markdown lines: verbatim, except that a fenced block in gofmt's code-block form
// (its lines after a tab) loses that tab and the blank lines gofmt puts between a code block and the fence lines
// around it, and that an example's `norun` marker (examples.go) is dropped from its fence (pageFence).
func renderBody(body []string) []string {
	var out []string
	next := 0 // the first line not yet copied
	for _, sp := range fenceSpans(body) {
		if sp.close < 0 { // refused by checkMarkdown; copied as it is
			break
		}
		out = append(out, body[next:sp.open]...)
		out = append(out, pageFence(body[sp.open]))
		content := body[sp.open+1 : sp.close]
		if slices.ContainsFunc(content, func(l string) bool { return strings.HasPrefix(l, "\t") }) {
			for _, l := range trimBlank(content) {
				out = append(out, strings.TrimPrefix(l, "\t"))
			}
		} else {
			out = append(out, content...)
		}
		out = append(out, body[sp.close])
		next = sp.close + 1
	}
	return append(out, body[next:]...)
}

// pageFence answers a fence line as the page shows it: a ```go norun [reason] fence, an example that is not run,
// is shown as a plain ```go fence — the marker is for TestDocExamples, not for the reader.
func pageFence(line string) string {
	if lang, noRun, _ := fenceInfo(line); lang == "go" && noRun {
		return line[:backtickRun(line)] + "go"
	}
	return line
}
