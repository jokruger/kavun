package reference

import (
	"regexp"
	"slices"
	"strings"
)

// Markdown answers the body as Markdown: the blocks joined with newlines, a text block verbatim, an example as a
// ```go fence long enough for its content. It is the body a reference page shows (the summary is a field of its
// own). Whether an example is run is the block's NoRun, not part of the fence: the page shows every example alike.
func (d Doc) Markdown() string {
	parts := make([]string, 0, len(d))
	for _, b := range d {
		if b.Kind == BlockExample {
			parts = append(parts, b.fenced())
			continue
		}
		parts = append(parts, b.Markdown)
	}
	return strings.Join(parts, "\n")
}

// fenced renders an example block as a ```go fenced block.
func (b Block) fenced() string {
	fence := strings.Repeat("`", fenceLen(b.Code))
	if b.Code == "" {
		return fence + "go\n" + fence
	}
	return fence + "go\n" + b.Code + "\n" + fence
}

// fenceLen answers the length of a backtick fence that no line of code closes: at least three, and longer than
// any run of backticks a line of code starts with (after its indentation).
func fenceLen(code string) int {
	n := 3
	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimLeft(line, " \t")
		if run := len(line) - len(strings.TrimLeft(line, "`")); run >= n {
			n = run + 1
		}
	}
	return n
}

// Examples answers the example blocks, in order.
func (d Doc) Examples() []Block {
	var out []Block
	for _, b := range d {
		if b.Kind == BlockExample {
			out = append(out, b)
		}
	}
	return out
}

// LinkScheme starts the target of a symbolic link: `kavun:type/array/splice`.
const LinkScheme = "kavun:"

// Link is one link destination of a Markdown text, as FindLinks finds it.
type Link struct {
	Target string // the destination as written: "kavun:type/array/splice", "#x", "https://…", "string.md"
	Offset int    // the byte offset of Target in the text
	Line   int    // the 0-based line of the text holding it
}

var (
	// inlineLink matches the destination of an inline Markdown link: [text](dest) or [text](<dest>).
	inlineLink = regexp.MustCompile(`\]\(\s*<?([^\s)>]+)`)
	// definitionLink matches a link reference definition: [label]: dest.
	definitionLink = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:\s*<?([^\s>]+)`)
	// autolink matches an autolink to an absolute URI: <scheme:…>.
	autolink = regexp.MustCompile(`<([A-Za-z][A-Za-z0-9+.-]{1,31}:[^\s<>]*)>`)
)

// FindLinks answers the link destinations of a Markdown text, in order of appearance: inline links, autolinks and
// link reference definitions, outside fenced blocks (``` or ~~~, indented up to three spaces, as in CommonMark) and
// code spans. It is the one link reader: Doc.Links, the generator's checks and the docs tool's rendering all read
// links through it.
func FindLinks(markdown string) []Link {
	var out []Link
	eachTextLine(markdown, func(n, start int, line string) {
		line = blankCodeSpans(line)
		if m := definitionLink.FindStringSubmatchIndex(line); m != nil {
			out = append(out, Link{Target: line[m[2]:m[3]], Offset: start + m[2], Line: n})
			return
		}
		var links []Link
		blanked := []byte(line)
		for _, m := range inlineLink.FindAllStringSubmatchIndex(line, -1) {
			links = append(links, Link{Target: line[m[2]:m[3]], Offset: start + m[2], Line: n})
			for i := m[0]; i < m[1]; i++ {
				blanked[i] = ' ' // an inline link's <destination> is not an autolink
			}
		}
		for _, m := range autolink.FindAllSubmatchIndex(blanked, -1) {
			links = append(links, Link{Target: line[m[2]:m[3]], Offset: start + m[2], Line: n})
		}
		slices.SortFunc(links, func(a, b Link) int { return a.Offset - b.Offset })
		out = append(out, links...)
	})
	return out
}

// ReplaceLinks answers the text with the target of every link FindLinks finds replaced by what f answers for it
// (f answering the target unchanged keeps the link as it is).
func ReplaceLinks(markdown string, f func(Link) string) string {
	var sb strings.Builder
	next := 0
	for _, l := range FindLinks(markdown) {
		sb.WriteString(markdown[next:l.Offset])
		sb.WriteString(f(l))
		next = l.Offset + len(l.Target)
	}
	sb.WriteString(markdown[next:])
	return sb.String()
}

// Links answers the kavun: targets of the links in the text blocks (FindLinks), in order of appearance, repeats
// included.
func (d Doc) Links() []string {
	var out []string
	for _, b := range d {
		if b.Kind != BlockText {
			continue
		}
		for _, l := range FindLinks(b.Markdown) {
			if strings.HasPrefix(l.Target, LinkScheme) {
				out = append(out, l.Target)
			}
		}
	}
	return out
}

// atxHeading matches an ATX heading line: up to three spaces, one to six `#`, then a space or the line's end.
var atxHeading = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?(?:[ \t]+#+)?[ \t]*$`)

// Headings answers the slugs (Slug) of the headings of the text blocks, outside fenced blocks, in order: the
// fragments a link to the block's owner may name.
func (d Doc) Headings() []string {
	var out []string
	for _, b := range d {
		if b.Kind != BlockText {
			continue
		}
		eachTextLine(b.Markdown, func(_, _ int, line string) {
			if m := atxHeading.FindStringSubmatch(line); m != nil {
				out = append(out, Slug(m[2]))
			}
		})
	}
	return out
}

// HeadingText answers the text of an ATX heading line (`## Rate and charge tables` → "Rate and charge tables"),
// false when the line is no heading. Slug turns the text into the heading's fragment.
func HeadingText(line string) (string, bool) {
	m := atxHeading.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[2], true
}

// eachTextLine calls f with every line of a Markdown text outside its fenced blocks: its 0-based number, its byte
// offset in the text and the line itself. The fence lines are not text lines.
func eachTextLine(markdown string, f func(n, start int, line string)) {
	var open fence // the open fenced block; zero outside one
	start := 0
	for n, line := range strings.Split(markdown, "\n") {
		at := start
		start += len(line) + 1
		fl := fenceOf(line)
		switch {
		case open.n > 0:
			if fl.char == open.char && fl.n >= open.n && fl.bare {
				open = fence{}
			}
			continue
		case fl.n > 0:
			open = fl
			continue
		}
		f(n, at, line)
	}
}

// fence is a fence line read by fenceOf: its character, the length of its run (0 when the line is no fence) and
// whether nothing but blanks follows the run (only such a line closes a block).
type fence struct {
	char byte
	n    int
	bare bool
}

// fenceOf reads a line as a CommonMark fence: up to three spaces, then three or more backticks or tildes (an opening
// backtick fence's info string holds no backtick).
func fenceOf(line string) fence {
	rest := strings.TrimLeft(line, " ")
	if len(line)-len(rest) > 3 || rest == "" || (rest[0] != '`' && rest[0] != '~') {
		return fence{}
	}
	c := rest[0]
	n := len(rest) - len(strings.TrimLeft(rest, string(c)))
	info := rest[n:]
	if n < 3 || c == '`' && strings.Contains(info, "`") {
		return fence{}
	}
	return fence{char: c, n: n, bare: strings.TrimSpace(info) == ""}
}

// blankCodeSpans answers a line with its code spans (a run of backticks up to the next run of the same length)
// blanked out to spaces, so the offsets of the rest stay those of the line. A run with no closing run is literal
// text and stays.
func blankCodeSpans(line string) string {
	b := []byte(line)
	for i := 0; i < len(b); {
		if b[i] != '`' {
			i++
			continue
		}
		n := i
		for n < len(b) && b[n] == '`' {
			n++
		}
		end := closingRun(line, n, n-i)
		if end < 0 {
			i = n
			continue
		}
		for j := i; j < end; j++ {
			b[j] = ' '
		}
		i = end
	}
	return string(b)
}

// closingRun answers the index just after the first run of exactly run backticks at or after from, or -1.
func closingRun(line string, from, run int) int {
	for i := from; i < len(line); {
		if line[i] != '`' {
			i++
			continue
		}
		j := i
		for j < len(line) && line[j] == '`' {
			j++
		}
		if j-i == run {
			return j
		}
		i = j
	}
	return -1
}
