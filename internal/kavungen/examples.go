package kavungen

import (
	"fmt"
	"go/token"
	"strings"
)

// The examples of the doc blocks. A fenced ```go block in a doc block is a standalone Kavun script, run by
// TestDocExamples (kavun_test.go) and checked line by line through its comments:
//
//	expr   // => <render>        the expression's value renders as <render>: format(value)
//	stmt   // Error: <kind>[: …] the statement raises an error of that kind
//	…      // anything else     prose, not checked
//
// A render may be followed by a note after " — " (`[1, 2] — spreads`), which is prose. A block that is not a
// runnable script (a fragment, pseudo-code, placeholders) is fenced ```go norun, optionally followed by the
// reason; it is not run, and the page shows a plain ```go fence (renderBody).

// NoRun is the fence marker of an example that is not run: ```go norun [reason].
const NoRun = "norun"

// Example is one fenced `go` block of a doc block.
type Example struct {
	Owner  string         // what the doc block declares, for messages: "member pad_start", "type array"
	Pos    token.Position // the opening fence line
	NoRun  bool           // fenced ```go norun: not run
	Reason string         // the text after `norun` on the fence, its leading "—" dropped; "" when none
	Lines  []ExampleLine  // the content lines, gofmt's code-block tab removed
}

// ExampleLine is one line of an example with its source position.
type ExampleLine struct {
	Text string
	Pos  token.Position
}

// LineKind is what a line of an example claims (ClassifyLine).
type LineKind int

const (
	// LineCode is a line with no check: no comment, or a prose comment.
	LineCode LineKind = iota
	// LineResult is `expr // => <render>`: the expression's format() render is Render.
	LineResult
	// LineError is `stmt // Error: <kind>…`: the statement raises an error of kind ErrKind.
	LineError
)

// LineCheck is the reading of one example line.
type LineCheck struct {
	Kind    LineKind
	Code    string // the line without its trailing comment, right-trimmed
	Comment string // the comment text after `//`, trimmed; "" when there is none
	Render  string // LineResult: the expected render (a " — " note removed)
	ErrKind string // LineError: the expected error kind
}

// resultMarker and errorMarker open the two checked comments.
const (
	resultMarker = "=>"
	errorMarker  = "Error:"
	noteSep      = " — "
)

// ClassifyLine reads one example line: its code, its trailing `//` comment (found outside string and character
// literals) and what the comment claims.
func ClassifyLine(line string) LineCheck {
	at := commentStart(line)
	if at < 0 {
		return LineCheck{Kind: LineCode, Code: strings.TrimRight(line, " \t")}
	}
	c := LineCheck{Code: strings.TrimRight(line[:at], " \t"), Comment: strings.TrimSpace(line[at+2:])}
	switch {
	case strings.HasPrefix(c.Comment, resultMarker+" "), c.Comment == resultMarker:
		c.Kind = LineResult
		c.Render = strings.TrimSpace(strings.TrimPrefix(c.Comment, resultMarker))
		if i := strings.Index(c.Render, noteSep); i >= 0 {
			c.Render = strings.TrimSpace(c.Render[:i])
		}
	case strings.HasPrefix(c.Comment, errorMarker):
		c.Kind = LineError
		rest := strings.TrimSpace(strings.TrimPrefix(c.Comment, errorMarker))
		end := strings.IndexAny(rest, ": \t")
		if end < 0 {
			end = len(rest)
		}
		c.ErrKind = rest[:end]
	default:
		c.Kind = LineCode
	}
	return c
}

// commentStart answers the byte offset of the `//` that opens the line's trailing comment, or -1. It skips string
// literals ("…", u"…", b"…", f"…" — the expressions inside an f-string's braces included), raw strings (`…`) and
// character literals ('…').
func commentStart(line string) int {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '/':
			if i+1 < len(line) && line[i+1] == '/' {
				return i
			}
		case '"', '\'':
			i = skipQuoted(line, i)
		case '`':
			if j := strings.IndexByte(line[i+1:], '`'); j >= 0 {
				i += j + 1
			} else {
				return -1
			}
		case 'f':
			if i+1 < len(line) && line[i+1] == '"' && (i == 0 || !isIdentByte(line[i-1])) {
				i = skipFString(line, i+1)
			}
		}
	}
	return -1
}

// skipQuoted answers the index of the quote closing the literal opened at line[open] (backslash escapes honoured),
// or the last index when the line ends first.
func skipQuoted(line string, open int) int {
	q := line[open]
	for i := open + 1; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case q:
			return i
		}
	}
	return len(line) - 1
}

// skipFString answers the index of the quote closing the f-string opened at line[open]: inside its `{…}`
// replacement fields, nested string literals are skipped whole.
func skipFString(line string, open int) int {
	depth := 0
	for i := open + 1; i < len(line); i++ {
		switch c := line[i]; {
		case c == '\\' && depth == 0:
			i++
		case c == '{':
			depth++
		case c == '}' && depth > 0:
			depth--
		case c == '"' && depth == 0:
			return i
		case (c == '"' || c == '\'') && depth > 0:
			i = skipQuoted(line, i)
		}
	}
	return len(line) - 1
}

func isIdentByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// fenceInfo reads a fence line's info string: lang is its first word ("go"), noRun reports the `norun` marker
// after it, and reason is the text after the marker, its leading dash dropped (```go norun — a fragment → "a
// fragment").
func fenceInfo(line string) (lang string, noRun bool, reason string) {
	info := strings.TrimSpace(strings.TrimLeft(line, "`"))
	lang, rest, _ := strings.Cut(info, " ")
	rest = strings.TrimSpace(rest)
	if word, after, _ := strings.Cut(rest, " "); word == NoRun {
		return lang, true, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(after), "—"))
	}
	return lang, false, ""
}

// Examples reads the package directories (as kavun-gen does: the files the build includes, tests excluded) and
// answers every fenced ```go block of their doc blocks, in directory, file and line order. A package whose
// directives do not parse is reported as kavun-gen would report it.
func Examples(dirs []string) ([]Example, error) {
	var out []Example
	var errs errorList
	for _, dir := range dirs {
		files, _, problems := readPackage(dir)
		if len(problems) > 0 {
			errs = append(errs, problems...)
			continue
		}
		scanned, problems := scanPackage(dir, files)
		if len(problems) > 0 {
			errs = append(errs, problems...)
			continue
		}
		for _, f := range scanned.files {
			for _, d := range f.directives {
				out = append(out, docExamples(d)...)
			}
		}
	}
	if err := errs.err(); err != nil {
		return nil, err
	}
	return out, nil
}

// docExamples answers the fenced go blocks of one directive's doc body.
func docExamples(d Directive) []Example {
	owner := string(d.Kind)
	if d.Name != "" {
		owner += " " + d.Name
	}
	body, pos := d.Doc.Body, d.Doc.BodyPos
	var out []Example
	for _, sp := range fenceSpans(body) {
		if sp.close < 0 {
			break
		}
		lang, noRun, reason := fenceInfo(body[sp.open])
		if lang != "go" {
			continue
		}
		ex := Example{Owner: owner, Pos: pos[sp.open], NoRun: noRun, Reason: reason}
		content := body[sp.open+1 : sp.close]
		tabbed := containsTabbed(content)
		for i, l := range content {
			if tabbed {
				l = strings.TrimPrefix(l, "\t")
			}
			ex.Lines = append(ex.Lines, ExampleLine{Text: l, Pos: pos[sp.open+1+i]})
		}
		out = append(out, ex)
	}
	return out
}

// containsTabbed reports whether a fenced block is in gofmt's code-block form (a tab before its lines).
func containsTabbed(content []string) bool {
	for _, l := range content {
		if strings.HasPrefix(l, "\t") {
			return true
		}
	}
	return false
}

// String renders an example's place for messages: file:line (owner).
func (e Example) String() string {
	return fmt.Sprintf("%s (%s)", where(e.Pos), e.Owner)
}
