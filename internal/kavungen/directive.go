package kavungen

import (
	"fmt"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"github.com/jokruger/kavun/internal/signature"
)

// directivePrefix starts every line the generator reads. A line with a space after `//`, or another tool's
// directive (`//go:`, `//nolint:`), is not ours.
const directivePrefix = "//kavun:"

// Kind is a directive's kind, the word after `//kavun:`.
type Kind string

// The directive kinds; see the package documentation for what each one declares.
const (
	KindType     Kind = "type"
	KindMember   Kind = "member"
	KindBuiltins Kind = "builtins"
	KindBuiltin  Kind = "builtin"
	KindModule   Kind = "module"
	KindFn       Kind = "fn"
)

// attachedToFunc reports whether a kind must sit directly above the Go function it declares (in that function's doc
// comment); every other kind is free-standing.
func (k Kind) attachedToFunc() bool {
	return k == KindMember || k == KindBuiltin || k == KindFn
}

// isPageRoot reports whether a kind's doc block heads a generated page, the only place a `#` or `##` heading may
// appear (an entry's heading is a generated `###`; its body uses `###` and deeper, nesting under it).
func (k Kind) isPageRoot() bool {
	return k == KindType || k == KindModule || k == KindBuiltins
}

// needsSummary reports whether a kind's doc block must open with a summary sentence.
func (k Kind) needsSummary() bool {
	return k != KindBuiltins
}

// Directive is one parsed `//kavun:` line together with the doc block of its comment group.
type Directive struct {
	Kind Kind
	Pos  token.Position // the directive line

	Name     string          // type, member, builtin, module or fn name
	Pure     bool            // member, builtin, module, fn
	Template string          // member, builtin, fn: the signature template verbatim
	Sig      signature.Shape // parsed from Template
	Result   string          // member, builtin, fn: what it answers, free text after ` -> ` (documentation only)
	Slot     int             // builtin, fn
	GoFunc   string          // the Go function to call: the attached body

	Doc Doc

	// Sections lists a page root's `##` headings outside fenced blocks, with their positions: a heading equal to the
	// one the generator adds (Members, Functions) is refused where the page gets that list.
	Sections []Section
}

// Section is one `##` heading of a page root's doc block.
type Section struct {
	Title string
	Pos   token.Position
}

// Doc is the doc block of a directive: the comment lines of its group other than directive lines, with the `// `
// prefix stripped.
type Doc struct {
	Summary string   // the first non-empty line
	Body    []string // the lines after the summary (markdown, kept verbatim; leading/trailing blank lines dropped)
	// BodyPos holds the source position of each line of Body (the examples' file:line, see Examples).
	BodyPos []token.Position
}

var (
	namePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
	// typeNamePattern admits a module-owned type's prefixed name, "fs.file" (CLAUDE.md "Module-owned types").
	typeNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*(\.[a-z_][a-z0-9_]*)?$`)
)

// parseDirective parses the text after `//kavun:`. The doc block, position and attached function are the caller's.
func parseDirective(text string) (Directive, error) {
	kindText, rest, _ := strings.Cut(text, " ")
	rest = strings.TrimSpace(rest)
	d := Directive{Kind: Kind(kindText)}

	switch d.Kind {
	case KindType:
		// //kavun:type <name>
		if !isSingleWord(rest) || !typeNamePattern.MatchString(rest) {
			return d, fmt.Errorf("malformed directive: want `//kavun:type <name>` (a name, or module.name for a module-owned type)")
		}
		d.Name = rest

	case KindBuiltins:
		// //kavun:builtins
		if rest != "" {
			return d, fmt.Errorf("malformed directive: `//kavun:builtins` takes no fields")
		}

	case KindModule:
		// //kavun:module <name> pure|impure
		fields := strings.Fields(rest)
		if len(fields) != 2 || !namePattern.MatchString(fields[0]) {
			return d, fmt.Errorf("malformed directive: want `//kavun:module <name> pure|impure`")
		}
		d.Name = fields[0]
		pure, err := parsePurity(fields[1])
		if err != nil {
			return d, err
		}
		d.Pure = pure

	case KindMember:
		// //kavun:member <name> pure|impure <template> -> <result>
		name, purity, tail, ok := cutWord2(rest)
		if !ok {
			return d, fmt.Errorf("malformed directive: want `//kavun:member <name> pure|impure <template> -> <result>`")
		}
		if err := d.setCommon(name, purity, tail); err != nil {
			return d, err
		}

	case KindBuiltin, KindFn:
		// //kavun:builtin <name> slot=<n> pure|impure <template> -> <result>   (fn: the same)
		name, slotText, tail, ok := cutWord2(rest)
		purity, tail, ok2 := strings.Cut(tail, " ")
		if !ok || !ok2 || !strings.HasPrefix(slotText, "slot=") {
			return d, fmt.Errorf("malformed directive: want `//kavun:%s <name> slot=<n> pure|impure <template> -> <result>`", d.Kind)
		}
		slot, err := strconv.Atoi(strings.TrimPrefix(slotText, "slot="))
		if err != nil || slot < 0 {
			return d, fmt.Errorf("malformed directive: slot %q is not a non-negative number", strings.TrimPrefix(slotText, "slot="))
		}
		d.Slot = slot
		if err := d.setCommon(name, purity, strings.TrimSpace(tail)); err != nil {
			return d, err
		}

	case "":
		return d, fmt.Errorf("malformed directive: missing kind after `//kavun:`")
	default:
		return d, fmt.Errorf("unknown directive kind %q", kindText)
	}
	return d, nil
}

// resultArrow separates a function-declaring directive's template from its result.
const resultArrow = " -> "

// keyPattern matches a `key=value` word, which no function-declaring directive takes.
var keyPattern = regexp.MustCompile(`(^| )[a-z_]+=`)

// setCommon fills and checks the name, purity, template and result every function-declaring kind carries; spec is
// `<template> -> <result>`.
func (d *Directive) setCommon(name, purity, spec string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("malformed directive: %q is not a name ([a-z_][a-z0-9_]*)", name)
	}
	d.Name = name
	pure, err := parsePurity(purity)
	if err != nil {
		return err
	}
	d.Pure = pure
	tmpl, result, ok := strings.Cut(spec, resultArrow)
	tmpl, result = strings.TrimSpace(tmpl), strings.TrimSpace(result)
	if m := keyPattern.FindString(tmpl); m != "" {
		return fmt.Errorf("malformed directive: unknown key %q (the template and ` -> <result>` are the only fields)", strings.TrimSpace(m))
	}
	if tmpl == "" {
		return fmt.Errorf("malformed directive: missing signature template")
	}
	if m := keyPattern.FindString(result); strings.TrimSpace(m) == "with=" {
		return fmt.Errorf("malformed directive: unknown key \"with=\" (every member is its own entry)")
	}
	if !ok || result == "" {
		return fmt.Errorf("malformed directive: missing ` -> <result>` after the template (what it answers: `-> array`, `-> [dir, file]`, `-> undefined`)")
	}
	d.Result = result
	sig, err := signature.Parse(tmpl)
	if err != nil {
		return err
	}
	d.Template, d.Sig = tmpl, sig
	return nil
}

func parsePurity(word string) (bool, error) {
	switch word {
	case "pure":
		return true, nil
	case "impure":
		return false, nil
	}
	return false, fmt.Errorf("malformed directive: want `pure` or `impure`, got %q", word)
}

// cutWord2 splits off the first two space-separated words; tail is the rest, trimmed.
func cutWord2(s string) (w1, w2, tail string, ok bool) {
	w1, rest, ok1 := strings.Cut(s, " ")
	w2, tail, ok2 := strings.Cut(rest, " ")
	return w1, w2, strings.TrimSpace(tail), ok1 && ok2 && w1 != "" && w2 != ""
}

func isSingleWord(s string) bool {
	return s != "" && !strings.ContainsAny(s, " \t")
}
