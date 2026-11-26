package kavungen

import (
	"fmt"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"github.com/jokruger/kavun/signature"
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
	KindConst    Kind = "const"
	KindNames    Kind = "names"
)

// attachedToFunc reports whether a kind must sit directly above the Go function it declares (in that function's doc
// comment); every other kind is free-standing.
func (k Kind) attachedToFunc() bool {
	return k == KindMember || k == KindBuiltin || k == KindFn
}

// attachedToVar reports whether a kind must sit directly above the package-level Go variable it declares (in that
// variable's doc comment).
func (k Kind) attachedToVar() bool {
	return k == KindConst || k == KindNames
}

// isPageRoot reports whether a kind's doc block heads a generated page, the only place a `#` or `##` heading may
// appear (an entry's heading is a generated `###`; its body uses `###` and deeper, nesting under it).
func (k Kind) isPageRoot() bool {
	return k == KindType || k == KindModule || k == KindBuiltins
}

// needsSummary reports whether a kind's doc block must open with a summary sentence.
func (k Kind) needsSummary() bool {
	return k != KindBuiltins && k != KindNames
}

// Directive is one parsed `//kavun:` line together with the doc block of its comment group.
type Directive struct {
	Kind Kind
	Pos  token.Position // the directive line

	Name     string          // type, member, builtin, module, fn or const name
	Pure     bool            // member, builtin, module, fn
	Template string          // member, builtin, fn: the signature template verbatim
	Sig      signature.Shape // parsed from Template
	// Result is the free text after ` -> ` (documentation only): what a member, builtin or fn answers, a const's type.
	Result string
	Slot   int    // builtin, fn
	GoFunc string // the Go function to call: the attached body
	GoVar  string // const: the Go variable holding the value; names: the Go variable holding the map
	// ModuleID is a module's type id, the Go expression of its id= field: an identifier of the package
	// ("myModule") or a core/value constant ("value.ModuleMath").
	ModuleID string
	// Names is what a //kavun:names declares: one member name per key of its map literal, in source order.
	Names []NameDecl

	Doc Doc

	// Sections lists a page root's headings outside fenced blocks, with their positions: a `##` heading equal to the
	// one kavun-docs adds (Members, Functions) is refused where the page gets that list, and a heading whose slug is
	// an entry's name everywhere; a link names a heading by its slug.
	Sections []Section
}

// NameDecl is one entry of a //kavun:names map literal: an embedder member name and the Go constant of its id.
type NameDecl struct {
	Key  string // the constant (member.ID) of the package keying the entry: "MemberDouble"
	Name string // the member name: "double"
	Pos  token.Position
}

// Section is one heading of a page root's doc block.
type Section struct {
	Title string // the heading's text, without its `#` markers
	Level int    // the number of `#`: 2 for `##`
	Pos   token.Position
}

// Doc is the doc block of a directive: the comment lines of its group other than directive lines, with the `// `
// prefix stripped.
type Doc struct {
	Summary    string           // the first paragraph, its lines joined with single spaces
	SummaryPos []token.Position // the position of each source line of the summary
	SummaryAt  []int            // the byte offset in Summary where each of those lines starts
	Body       []string         // the lines after the summary (markdown, kept verbatim; leading/trailing blank lines dropped)
	// BodyPos holds the source position of each line of Body (the examples' file:line, see Examples).
	BodyPos []token.Position
}

var (
	namePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
	// typeNamePattern admits a module-owned type's prefixed name, "fs.file" (CLAUDE.md "Module-owned types").
	typeNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*(\.[a-z_][a-z0-9_]*)?$`)
	// moduleIDPattern is a module's id=: a Go identifier, or an exported constant of core/value qualified by it.
	moduleIDPattern = regexp.MustCompile(`^(value\.\p{Lu}[\p{L}\p{Nd}_]*|[\p{L}_][\p{L}\p{Nd}_]*)$`)
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

	case KindNames:
		// //kavun:names
		if rest != "" {
			return d, fmt.Errorf("malformed directive: `//kavun:names` takes no fields")
		}

	case KindModule:
		// //kavun:module <name> id=<go ident> pure|impure
		fields := strings.Fields(rest)
		if len(fields) != 3 || !namePattern.MatchString(fields[0]) {
			return d, fmt.Errorf("malformed directive: want `//kavun:module <name> id=<go ident> pure|impure`")
		}
		d.Name = fields[0]
		id, ok := strings.CutPrefix(fields[1], "id=")
		if !ok {
			return d, fmt.Errorf("malformed directive: want `//kavun:module <name> id=<go ident> pure|impure` (id= names the module's type id: an identifier of the package, or value.<Ident>)")
		}
		if !moduleIDPattern.MatchString(id) {
			return d, fmt.Errorf("malformed directive: id=%s is not a Go identifier of the package or value.<Ident>", id)
		}
		d.ModuleID = id
		pure, err := parsePurity(fields[2])
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

	case KindConst:
		// //kavun:const <name> -> <type>
		name, typ, ok := strings.Cut(rest, resultArrow)
		name, typ = strings.TrimSpace(name), strings.TrimSpace(typ)
		if !ok || !isSingleWord(name) || typ == "" {
			return d, fmt.Errorf("malformed directive: want `//kavun:const <name> -> <type>`")
		}
		if !namePattern.MatchString(name) {
			return d, fmt.Errorf("malformed directive: %q is not a name ([a-z_][a-z0-9_]*)", name)
		}
		d.Name, d.Result = name, typ

	case "":
		return d, fmt.Errorf("malformed directive: missing kind after `//kavun:`")
	default:
		return d, fmt.Errorf("unknown directive kind %q", kindText)
	}
	return d, nil
}

// conversionTemplate is the template of every member named after a conversion (member.IsConversion): x.T([default]).
const conversionTemplate = "([default])"

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
