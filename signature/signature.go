// Package signature is the signature-template grammar: it parses a template — "(width[, fill])",
// "() | (start, stop[, step])" — into its readings, their parameters and the argument counts they accept.
// package core builds core.Signature on it, cmd/kavun-gen validates declarations with it, and tools reading
// templates (the reference, an editor) use it directly; it does not import core.
//
// The grammar is strict and admits only its canonical spelling:
//
//	template := reading { " | " reading }
//	reading  := "(" [ list ] ")"
//	list     := group | rest | name { ", " name } [ ", " rest | group ]
//	rest     := "..." name
//	group    := "[" ( ", " if a name precedes it ) name { ", " name } [ group ] "]"
//	name     := [a-z_][a-z0-9_]*
//
// An optional group continues the list it closes, so it opens with ", " inside the bracket ("(width[, fill])"),
// except at the start of an empty list ("([sep])"). A group is the last thing in its list; a rest
// ("...name") is the last item of a reading and never sits inside a group. A name appears once per reading.
// Every reading must accept at least one argument count that no other reading of the template accepts: a
// reading whose counts the others already cover (an identical reading included) is refused as redundant.
//
// Groups nest and never sit side by side, so the parameters of one nesting depth form one group and the
// brackets are recoverable from Param.Optional exactly: "([start[, delete_count]])" is start 1, delete_count 2.
//
// Arity is the fewest arguments any reading accepts; the template is variadic when it accepts any other count.
package signature

import (
	"fmt"
	"slices"
	"strings"
)

// Shape is what a template names: its readings and the argument-count set their union accepts.
type Shape struct {
	Template string    // the template verbatim
	Readings []Reading // in template order
	Counts   []int     // the finite accepted counts besides Arity, sorted, distinct, each below RestFrom when HasRest; nil when none
	Arity    int       // the fewest arguments accepted
	RestFrom int       // with HasRest: every count >= RestFrom is accepted
	HasRest  bool      // whether the accepted set is open-ended
	Variadic bool      // whether any count other than Arity is accepted
}

// Reading is one parenthesised alternative of a template.
type Reading struct {
	Params []Param `json:"params"`         // the named parameters in order; never nil: "()" is an empty list
	Rest   string  `json:"rest,omitempty"` // the name after "..."; "" without a rest
}

// Param is one named parameter of a reading.
type Param struct {
	Name     string `json:"name"`
	Optional int    `json:"optional,omitempty"` // optional-group nesting depth; 0 = required
}

// Parse parses a signature template (the grammar is in the package doc). A malformed template answers an error
// naming the template, the problem and its byte offset.
func Parse(template string) (Shape, error) {
	if template == "" {
		return Shape{}, fmt.Errorf("signature %q: empty template", template)
	}

	p := signatureParser{src: template}
	var readings []Reading
	var spans [][2]int // each reading's byte offsets, for the redundancy error

	for {
		start := p.pos
		r, err := p.reading()
		if err != nil {
			return Shape{}, err
		}
		readings = append(readings, r)
		spans = append(spans, [2]int{start, p.pos})

		if p.pos == len(template) {
			break
		}
		if !p.skip(" | ") {
			return Shape{}, p.fail(p.pos, "readings must be joined by ` | `")
		}
	}

	var counts []int
	restFrom, hasRest := 0, false
	for _, r := range readings {
		rc, rf, hr := r.Counts()
		counts = append(counts, rc...)
		if hr && (!hasRest || rf < restFrom) {
			restFrom, hasRest = rf, true
		}
	}

	for i := range readings {
		if isCoveredByOthers(readings, i) {
			sp := spans[i]
			return Shape{}, p.fail(sp[0], "redundant reading %s: every count it accepts is accepted by another reading",
				template[sp[0]:sp[1]])
		}
	}

	s := newShape(counts, restFrom, hasRest)
	s.Template = template
	s.Readings = readings
	return s, nil
}

// Accepts reports whether the template accepts exactly k arguments. A negative k is never accepted.
func (s Shape) Accepts(k int) bool {
	if k < 0 {
		return false
	}
	return k == s.Arity || (s.HasRest && k >= s.RestFrom) || slices.Contains(s.Counts, k)
}

// Max answers the largest argument count the template names and whether the accepted set is open above it. For a
// closed set n is its largest member; for an open set (a "...rest") n is the count the open run starts at — every
// k >= n is accepted.
func (s Shape) Max() (n int, open bool) {
	if s.HasRest {
		return s.RestFrom, true
	}
	if len(s.Counts) > 0 {
		return s.Counts[len(s.Counts)-1], false
	}
	return s.Arity, false
}

// Counts answers the argument counts this reading alone accepts: the finite ones, sorted and distinct, and with a
// rest the count it opens at (the number of fixed parameters — every count >= restFrom is accepted). A rest
// reading has no optional group, so its finite counts are empty: every count it accepts starts at restFrom.
func (r Reading) Counts() (counts []int, restFrom int, hasRest bool) {
	if r.Rest != "" {
		return nil, len(r.Params), true
	}
	// Groups nest and are last, so each depth closes a count: the parameters at that depth or shallower.
	n := 0
	for i, prm := range r.Params {
		if i == 0 || prm.Optional != r.Params[i-1].Optional {
			counts = append(counts, n)
		}
		n++
	}
	counts = append(counts, n)
	if len(r.Params) > 0 && r.Params[0].Optional == 0 {
		counts = counts[1:] // the leading 0 is a count only when the list starts with a group
	}
	return counts, 0, false
}

// isCoveredByOthers reports whether every count readings[i] accepts is also accepted by some other reading.
// A rest reading accepts every count from its start upwards, so it covers a finite count at or above its
// start, and another rest only when it opens at or below that rest's start.
func isCoveredByOthers(readings []Reading, i int) bool {
	type accepted struct {
		counts   []int
		restFrom int
		hasRest  bool
	}
	all := make([]accepted, len(readings))
	for j, r := range readings {
		all[j].counts, all[j].restFrom, all[j].hasRest = r.Counts()
	}

	acceptedByOther := func(k int) bool {
		for j, o := range all {
			if j != i && ((o.hasRest && k >= o.restFrom) || slices.Contains(o.counts, k)) {
				return true
			}
		}
		return false
	}

	r := all[i]
	for _, k := range r.counts {
		if !acceptedByOther(k) {
			return false
		}
	}
	if r.hasRest {
		for j, o := range all {
			if j != i && o.hasRest && o.restFrom <= r.restFrom {
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
	src    string
	pos    int
	params []Param // the parameters of the current reading
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

// reading parses one "(...)" into its parameters and rest.
func (p *signatureParser) reading() (Reading, error) {
	p.params = []Param{}
	if !p.skip("(") {
		return Reading{}, p.fail(p.pos, "a reading must be parenthesised")
	}
	if p.skip(")") {
		return Reading{Params: p.params}, nil
	}
	rest, err := p.list(0)
	if err != nil {
		return Reading{}, err
	}
	if !p.skip(")") {
		return Reading{}, p.unexpected("`)`")
	}
	return Reading{Params: p.params, Rest: rest}, nil
}

// list parses a non-empty parameter list at optional depth depth — the top of a reading (0) or inside an optional
// group — recording its parameters, and answers its rest's name ("" without one; a group never has one).
func (p *signatureParser) list(depth int) (rest string, err error) {
	inGroup := depth > 0
	if p.peek() == '[' {
		if inGroup {
			return "", p.fail(p.pos, "an optional group must begin with a parameter")
		}
		if err := p.group(true, depth+1); err != nil {
			return "", err
		}
		return "", p.groupIsLast()
	}

	for {
		if strings.HasPrefix(p.src[p.pos:], "...") {
			if inGroup {
				return "", p.fail(p.pos, "a rest cannot sit inside an optional group")
			}
			p.pos += len("...")
			name, err := p.name()
			if err != nil {
				return "", err
			}
			if p.peek() == ',' || p.peek() == '[' {
				return "", p.fail(p.pos, "a rest must be the last parameter")
			}
			return name, nil
		}

		name, err := p.name()
		if err != nil {
			return "", err
		}
		p.params = append(p.params, Param{Name: name, Optional: depth})

		switch {
		case p.skip(", "):
			continue
		case p.peek() == ',':
			return "", p.fail(p.pos, "parameters must be separated by `, `")
		case p.peek() == '[':
			if err := p.group(false, depth+1); err != nil {
				return "", err
			}
			return "", p.groupIsLast()
		default:
			return "", nil
		}
	}
}

// group parses an optional group — "[, names]", or "[names]" at the start of an empty list — whose parameters sit
// at optional depth depth.
func (p *signatureParser) group(first bool, depth int) error {
	open := p.pos
	p.pos++ // '['
	if first {
		if p.peek() == ',' {
			return p.fail(p.pos, "an optional group at the start of the list takes no leading `, `")
		}
	} else if !p.skip(", ") {
		return p.fail(p.pos, "an optional group after a parameter must continue the list with `[, `")
	}
	if p.peek() == ']' {
		return p.fail(p.pos, "empty optional group")
	}
	if _, err := p.list(depth); err != nil {
		return err
	}
	if !p.skip("]") {
		if p.pos == len(p.src) || p.peek() == ')' {
			return p.fail(open, "unbalanced `[`")
		}
		return p.unexpected("`]`")
	}
	return nil
}

// groupIsLast refuses anything but a closing bracket after an optional group.
func (p *signatureParser) groupIsLast() error {
	if p.peek() == ',' {
		return p.fail(p.pos, "an optional group must be the last thing in its list")
	}
	return nil
}

// name parses one parameter name, [a-z_][a-z0-9_]*, unique within the reading (its rest included), and answers it.
func (p *signatureParser) name() (string, error) {
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
		return "", p.unexpected("a parameter name ([a-z_][a-z0-9_]*)")
	}
	if c := p.peek(); (c >= 'A' && c <= 'Z') || c == '-' {
		// A name runs into a character it may not hold: report the name, not the separator after it.
		return "", p.fail(start, "invalid parameter name")
	}
	n := p.src[start:p.pos]
	if slices.ContainsFunc(p.params, func(prm Param) bool { return prm.Name == n }) {
		return "", p.fail(start, "duplicate parameter name %q", n)
	}
	return n, nil
}

// unexpected reports what was expected at the current offset and what was found there.
func (p *signatureParser) unexpected(want string) error {
	if p.pos == len(p.src) {
		return p.fail(p.pos, "expected %s, found end of template", want)
	}
	return p.fail(p.pos, "expected %s, found %q", want, p.src[p.pos:p.pos+1])
}
