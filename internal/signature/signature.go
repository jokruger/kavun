// Package signature parses signature templates — "(width[, fill])", "() | (start, stop[, step])" — into the set of
// argument counts they name. core.Signature is built on it; it lives apart from core so that tools reading
// templates (cmd/kavun-gen) do not depend on package core compiling.
package signature

import (
	"fmt"
	"slices"
	"strings"
)

// Shape is the argument-count set a template names.
type Shape struct {
	Counts   []int // the finite accepted counts besides Arity, sorted, distinct, each below RestFrom when HasRest; nil when none
	Arity    int   // the fewest arguments accepted
	RestFrom int   // with HasRest: every count >= RestFrom is accepted
	HasRest  bool  // whether the accepted set is open-ended
	Variadic bool  // whether any count other than Arity is accepted
}

// Parse parses a signature template; the grammar and the redundancy rule are documented on core.ParseSignature. A
// malformed template answers an error naming the template, the problem and its byte offset.
func Parse(template string) (Shape, error) {
	if template == "" {
		return Shape{}, fmt.Errorf("signature %q: empty template", template)
	}

	p := signatureParser{src: template}
	var readings []readingCounts // every reading parsed, for the redundancy check
	var counts []int
	restFrom, hasRest := 0, false

	for {
		start := p.pos
		rc, rest, err := p.reading()
		if err != nil {
			return Shape{}, err
		}
		readings = append(readings, readingCounts{text: template[start:p.pos], start: start, counts: rc, rest: rest})

		counts = append(counts, rc...)
		if rest >= 0 && (!hasRest || rest < restFrom) {
			restFrom, hasRest = rest, true
		}

		if p.pos == len(template) {
			break
		}
		if !p.skip(" | ") {
			return Shape{}, p.fail(p.pos, "readings must be joined by ` | `")
		}
	}

	for i, r := range readings {
		if isCoveredByOthers(readings, i) {
			return Shape{}, p.fail(r.start, "redundant reading %s: every count it accepts is accepted by another reading", r.text)
		}
	}

	return newShape(counts, restFrom, hasRest), nil
}

// readingCounts is what one reading of a template accepts, kept for the redundancy check: its finite counts
// and, when it ends in a rest, the count the rest opens at (-1 without one).
type readingCounts struct {
	text   string // the reading verbatim, for the error
	start  int    // its byte offset in the template, for the error
	counts []int  // its finite counts
	rest   int    // with a rest: every count >= rest is accepted; -1 without one
}

// isCoveredByOthers reports whether every count readings[i] accepts is also accepted by some other reading.
// A rest reading accepts every count from its start upwards, so it covers a finite count at or above its
// start, and another rest only when it opens at or below that rest's start.
func isCoveredByOthers(readings []readingCounts, i int) bool {
	acceptedByOther := func(k int) bool {
		for j, o := range readings {
			if j != i && ((o.rest >= 0 && k >= o.rest) || slices.Contains(o.counts, k)) {
				return true
			}
		}
		return false
	}

	r := readings[i]
	for _, k := range r.counts {
		if !acceptedByOther(k) {
			return false
		}
	}
	if r.rest >= 0 {
		for j, o := range readings {
			if j != i && o.rest >= 0 && o.rest <= r.rest {
				return true
			}
		}
		return false
	}
	return true
}

// newShape normalises the union of the readings' counts: it drops counts the rest already covers, takes the fewest
// as the arity and keeps the others sorted and distinct.
func newShape(counts []int, restFrom int, hasRest bool) Shape {
	slices.Sort(counts)
	counts = slices.Compact(counts)
	if hasRest {
		counts = slices.DeleteFunc(counts, func(c int) bool { return c >= restFrom })
	}

	s := Shape{RestFrom: restFrom, HasRest: hasRest}
	switch {
	case len(counts) == 0: // only a rest reading is left: it opens at its fixed count
		s.Arity = restFrom
	default: // counts are below restFrom, so the smallest is the arity
		s.Arity = counts[0]
		counts = counts[1:]
	}
	if len(counts) > 0 {
		s.Counts = slices.Clip(counts)
	}
	s.Variadic = hasRest || len(s.Counts) > 0
	return s
}

// signatureParser reads a template left to right; every method either consumes its construct or fails.
type signatureParser struct {
	src   string
	pos   int
	names []string // the names seen in the current reading
}

// fail builds the error for a problem found at offset at.
func (p *signatureParser) fail(at int, format string, args ...any) error {
	return fmt.Errorf("signature %q: %s at offset %d", p.src, fmt.Sprintf(format, args...), at)
}

// skip consumes lit when the input continues with it.
func (p *signatureParser) skip(lit string) bool {
	if strings.HasPrefix(p.src[p.pos:], lit) {
		p.pos += len(lit)
		return true
	}
	return false
}

// peek answers the next byte, or 0 at the end of the input.
func (p *signatureParser) peek() byte {
	if p.pos < len(p.src) {
		return p.src[p.pos]
	}
	return 0
}

// reading parses one "(...)" and answers its finite counts and, when it ends in a rest, the rest's fixed count
// (-1 without one).
func (p *signatureParser) reading() (counts []int, rest int, err error) {
	p.names = p.names[:0]
	if !p.skip("(") {
		return nil, -1, p.fail(p.pos, "a reading must be parenthesised")
	}
	if p.skip(")") {
		return []int{0}, -1, nil
	}
	counts, rest, err = p.list(false)
	if err != nil {
		return nil, -1, err
	}
	if !p.skip(")") {
		return nil, -1, p.unexpected("`)`")
	}
	return counts, rest, nil
}

// list parses a non-empty parameter list, at the top of a reading or inside an optional group, and answers
// the counts it accepts and its rest's fixed count (-1 without one; a group never has one).
func (p *signatureParser) list(inGroup bool) (counts []int, rest int, err error) {
	if p.peek() == '[' {
		if inGroup {
			return nil, -1, p.fail(p.pos, "an optional group must begin with a parameter")
		}
		g, err := p.group(true)
		if err != nil {
			return nil, -1, err
		}
		return append([]int{0}, g...), -1, p.groupIsLast()
	}

	n := 0 // the fixed names read so far
	for {
		if strings.HasPrefix(p.src[p.pos:], "...") {
			if inGroup {
				return nil, -1, p.fail(p.pos, "a rest cannot sit inside an optional group")
			}
			p.pos += len("...")
			if err := p.name(); err != nil {
				return nil, -1, err
			}
			if p.peek() == ',' || p.peek() == '[' {
				return nil, -1, p.fail(p.pos, "a rest must be the last parameter")
			}
			return nil, n, nil
		}

		if err := p.name(); err != nil {
			return nil, -1, err
		}
		n++

		switch {
		case p.skip(", "):
			continue
		case p.peek() == ',':
			return nil, -1, p.fail(p.pos, "parameters must be separated by `, `")
		case p.peek() == '[':
			g, err := p.group(false)
			if err != nil {
				return nil, -1, err
			}
			counts = []int{n}
			for _, c := range g {
				counts = append(counts, n+c)
			}
			return counts, -1, p.groupIsLast()
		default:
			return []int{n}, -1, nil
		}
	}
}

// group parses an optional group — "[, names]", or "[names]" at the start of an empty list — and answers the
// counts its content adds; each is at least 1.
func (p *signatureParser) group(first bool) ([]int, error) {
	open := p.pos
	p.pos++ // '['
	if first {
		if p.peek() == ',' {
			return nil, p.fail(p.pos, "an optional group at the start of the list takes no leading `, `")
		}
	} else if !p.skip(", ") {
		return nil, p.fail(p.pos, "an optional group after a parameter must continue the list with `[, `")
	}
	if p.peek() == ']' {
		return nil, p.fail(p.pos, "empty optional group")
	}
	counts, _, err := p.list(true)
	if err != nil {
		return nil, err
	}
	if !p.skip("]") {
		if p.pos == len(p.src) || p.peek() == ')' {
			return nil, p.fail(open, "unbalanced `[`")
		}
		return nil, p.unexpected("`]`")
	}
	return counts, nil
}

// groupIsLast refuses anything but a closing bracket after an optional group.
func (p *signatureParser) groupIsLast() error {
	if p.peek() == ',' {
		return p.fail(p.pos, "an optional group must be the last thing in its list")
	}
	return nil
}

// name parses one parameter name, [a-z_][a-z0-9_]*, unique within the reading.
func (p *signatureParser) name() error {
	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == '_' || (c >= 'a' && c <= 'z') || (p.pos > start && c >= '0' && c <= '9') {
			p.pos++
			continue
		}
		break
	}
	if p.pos == start {
		return p.unexpected("a parameter name ([a-z_][a-z0-9_]*)")
	}
	if c := p.peek(); (c >= 'A' && c <= 'Z') || c == '-' {
		// A name runs into a character it may not hold: report the name, not the separator after it.
		return p.fail(start, "invalid parameter name")
	}
	n := p.src[start:p.pos]
	if slices.Contains(p.names, n) {
		return p.fail(start, "duplicate parameter name %q", n)
	}
	p.names = append(p.names, n)
	return nil
}

// unexpected reports what was expected at the current offset and what was found there.
func (p *signatureParser) unexpected(want string) error {
	if p.pos == len(p.src) {
		return p.fail(p.pos, "expected %s, found end of template", want)
	}
	return p.fail(p.pos, "expected %s, found %q", want, p.src[p.pos:p.pos+1])
}
