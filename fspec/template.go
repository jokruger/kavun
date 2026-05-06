package fspec

import (
	"fmt"
	"strings"
)

// TemplateMode indicates whether a template's placeholders use named or indexed look-ups. The mode is locked by the
// first non-empty placeholder and every subsequent placeholder must agree.
type TemplateMode byte

const (
	TemplateModeUnset   TemplateMode = 0
	TemplateModeIndexed TemplateMode = 1 // {0}, {1}, ...
	TemplateModeNamed   TemplateMode = 2 // {name}, {x}, ...
)

// TemplateSegmentKind discriminates literal segments from interpolation placeholders inside a parsed template.
type TemplateSegmentKind byte

const (
	TemplateLiteral     TemplateSegmentKind = 0
	TemplatePlaceholder TemplateSegmentKind = 1
)

// TemplateSegment is one piece of a parsed template: either a literal run of text or a placeholder that references a
// value (and optionally a format spec) from the args supplied at runtime.
type TemplateSegment struct {
	Kind TemplateSegmentKind

	// Literal segment payload.
	Literal string

	// Placeholder fields.
	Name  string // for named mode
	Index int    // for indexed mode

	HasSpec bool

	// When HasSpec && !SpecIsRef the literal spec text was parsed at template parse time and stored in Spec.
	Spec FormatSpec

	// When HasSpec && SpecIsRef the spec is provided at runtime via a single {ref} placeholder inside the spec body.
	SpecIsRef    bool
	SpecRefName  string
	SpecRefIndex int
}

// Template is a parsed runtime format template ready to be rendered against a runtime args container (array for indexed
// mode, dict/record for named mode).
type Template struct {
	Mode     TemplateMode
	Segments []TemplateSegment
}

// ParseTemplate parses a runtime format template and returns the segment list.
//
// Grammar (informal):
//
//	template     := { text_char | '{{' | '}}' | placeholder }
//	placeholder  := '{' name_or_index [ ':' spec_body ] '}'
//	spec_body    := { spec_char | '{' name_or_index '}' }
//	name_or_index := identifier | non-negative-integer
//
// A bare '}' is an error. Empty '{}' is an error. Mixing named and indexed placeholders in the same template is an
// error. Inside the spec body the only `{...}` form allowed is a single name_or_index (no nesting, no expressions).
// A '#'-tail inside the literal portion of a spec body works as in fspec.Parse — no separate handling is needed here
// because the literal spec text is forwarded to fspec.Parse verbatim.
func ParseTemplate(s string) (Template, error) {
	var t Template
	var lit strings.Builder
	flushLit := func() {
		if lit.Len() == 0 {
			return
		}
		t.Segments = append(t.Segments, TemplateSegment{
			Kind:    TemplateLiteral,
			Literal: lit.String(),
		})
		lit.Reset()
	}

	i := 0
	n := len(s)
	for i < n {
		c := s[i]
		switch c {
		case '{':
			if i+1 < n && s[i+1] == '{' {
				lit.WriteByte('{')
				i += 2
				continue
			}
			flushLit()
			seg, next, err := parsePlaceholder(s, i)
			if err != nil {
				return Template{}, err
			}
			mode := placeholderMode(seg)
			if t.Mode == TemplateModeUnset {
				t.Mode = mode
			} else if mode != t.Mode {
				return Template{}, fmt.Errorf("format: cannot mix named and indexed placeholders at offset %d", i)
			}
			t.Segments = append(t.Segments, seg)
			i = next

		case '}':
			if i+1 < n && s[i+1] == '}' {
				lit.WriteByte('}')
				i += 2
				continue
			}
			return Template{}, fmt.Errorf("format: unmatched '}' at offset %d (use '}}' for a literal '}')", i)

		default:
			lit.WriteByte(c)
			i++
		}
	}
	flushLit()
	return t, nil
}

// placeholderMode reports the mode implied by a single parsed placeholder.
func placeholderMode(seg TemplateSegment) TemplateMode {
	if seg.Name != "" {
		return TemplateModeNamed
	}
	return TemplateModeIndexed
}

// parsePlaceholder parses a placeholder beginning at s[start] (the '{').
// It returns the parsed segment and the index just past the closing '}'.
func parsePlaceholder(s string, start int) (TemplateSegment, int, error) {
	n := len(s)
	i := start + 1 // skip '{'

	// Read name_or_index up to ':' or '}'.
	keyStart := i
	for i < n && s[i] != ':' && s[i] != '}' && s[i] != '{' {
		i++
	}
	if i == n {
		return TemplateSegment{}, 0, fmt.Errorf("format: unterminated placeholder starting at offset %d", start)
	}
	if s[i] == '{' {
		return TemplateSegment{}, 0, fmt.Errorf("format: unexpected '{' at offset %d (expressions are not allowed inside '{...}')", i)
	}
	keyText := s[keyStart:i]
	if keyText == "" {
		return TemplateSegment{}, 0, fmt.Errorf("format: empty placeholder '{}' at offset %d (auto-numbering is not supported)", start)
	}

	var seg TemplateSegment
	seg.Kind = TemplatePlaceholder
	if err := setSegmentKey(&seg, keyText, start); err != nil {
		return TemplateSegment{}, 0, err
	}

	if s[i] == '}' {
		return seg, i + 1, nil
	}

	// s[i] == ':'  — parse spec body.
	i++
	specStart := i
	hasInnerRef := false
	var refName string
	var refIndex int
	for i < n && s[i] != '}' {
		if s[i] == '{' {
			if hasInnerRef {
				return TemplateSegment{}, 0, fmt.Errorf("format: only one '{ref}' is allowed inside a format spec (offset %d)", i)
			}
			// the spec must be exactly a single {ref} with no surrounding literal characters, mirroring the user-facing
			// rule "fspec is in-place literal OR a single {ref}".
			if i != specStart {
				return TemplateSegment{}, 0, fmt.Errorf("format: '{ref}' inside a format spec must stand alone (offset %d)", i)
			}
			refStart := i + 1
			j := refStart
			for j < n && s[j] != '}' && s[j] != '{' && s[j] != ':' {
				j++
			}
			if j == n || s[j] != '}' {
				return TemplateSegment{}, 0, fmt.Errorf("format: unterminated '{ref}' inside format spec at offset %d", i)
			}
			ref := s[refStart:j]
			if ref == "" {
				return TemplateSegment{}, 0, fmt.Errorf("format: empty '{}' inside format spec at offset %d", i)
			}
			if isAllDigits(ref) {
				idx, ok := parseNonNegInt(ref)
				if !ok {
					return TemplateSegment{}, 0, fmt.Errorf("format: invalid index %q inside format spec at offset %d", ref, i)
				}
				refIndex = idx
				refName = ""
			} else {
				if !isIdent(ref) {
					return TemplateSegment{}, 0, fmt.Errorf("format: invalid name %q inside format spec at offset %d", ref, i)
				}
				refName = ref
				refIndex = 0
			}
			hasInnerRef = true
			i = j + 1
			// the spec must end immediately after the '}' of {ref}
			if i < n && s[i] != '}' {
				return TemplateSegment{}, 0, fmt.Errorf("format: '{ref}' inside a format spec must stand alone (offset %d)", i)
			}
			continue
		}
		i++
	}
	if i == n {
		return TemplateSegment{}, 0, fmt.Errorf("format: unterminated placeholder starting at offset %d", start)
	}

	seg.HasSpec = true
	if hasInnerRef {
		seg.SpecIsRef = true
		seg.SpecRefName = refName
		seg.SpecRefIndex = refIndex
	} else {
		spec, err := Parse(s[specStart:i])
		if err != nil {
			return TemplateSegment{}, 0, fmt.Errorf("format: %v", err)
		}
		seg.Spec = spec
	}
	return seg, i + 1, nil
}

// setSegmentKey fills Name or Index on seg based on keyText, validating it.
func setSegmentKey(seg *TemplateSegment, keyText string, start int) error {
	if isAllDigits(keyText) {
		idx, ok := parseNonNegInt(keyText)
		if !ok {
			return fmt.Errorf("format: invalid index %q at offset %d", keyText, start)
		}
		seg.Index = idx
		return nil
	}
	if !isIdent(keyText) {
		return fmt.Errorf("format: invalid placeholder %q at offset %d", keyText, start)
	}
	seg.Name = keyText
	return nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func parseNonNegInt(s string) (int, bool) {
	const max = int(^uint(0) >> 1)
	x := 0
	for i := 0; i < len(s); i++ {
		d := int(s[i] - '0')
		if x > (max-d)/10 {
			return 0, false
		}
		x = x*10 + d
	}
	return x, true
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}
