package reference

import (
	"fmt"
	"slices"
	"strings"
)

// Links in doc blocks are ordinary Markdown links with a kavun: target, naming what they link to rather than a page:
//
//	kavun:type/<type>[/<member>]           a type, one of its members
//	kavun:module/<module>[/<entry>]        a module, one of its functions or constants
//	kavun:builtins                         the builtin functions' block
//	kavun:builtin/<function>               a builtin function
//	kavun:doc/<page>                       a hand-written guide page: <docs root>/<page>.md of the linking origin
//
// A link to a type, a module, the builtins or a guide page may name a heading of it by its slug (Slug) as a
// fragment: kavun:type/string#undecodable-octets, kavun:doc/embedding#handling-errors. The heading of a type, a
// module or the builtins is one of its root block's (Doc.Headings); an entry's link takes no fragment.

// Target is what a kavun: link names (Set.Resolve).
type Target struct {
	Kind     OwnerKind // OwnerType, OwnerModule, OwnerBuiltins or OwnerDoc
	Owner    string    // the type's or module's name; the page for OwnerDoc ("embedding"); "" for the builtins
	Entry    string    // the member, the module's function or constant, the builtin function; "" for the owner
	Fragment string    // the slug of a heading; ""
	// Origin is the origin of the reference declaring the owner (for a builtin function: the function); "" for
	// OwnerDoc, a page of the origin whose reference holds the link.
	Origin string
}

// ParseLink parses a kavun: link into its target without resolving it: the form only (Target.Origin is "").
func ParseLink(link string) (Target, error) {
	rest, ok := strings.CutPrefix(link, LinkScheme)
	if !ok {
		return Target{}, fmt.Errorf("%s: not a %s link", link, LinkScheme)
	}
	rest, fragment, hasFragment := strings.Cut(rest, "#")
	if hasFragment && fragment == "" {
		return Target{}, fmt.Errorf("%s: an empty fragment", link)
	}
	kind, path, _ := strings.Cut(rest, "/")
	parts := strings.Split(path, "/")
	if path == "" {
		parts = nil
	}
	for _, p := range parts {
		if p == "" {
			return Target{}, fmt.Errorf("%s: an empty path segment", link)
		}
	}
	t := Target{Kind: OwnerKind(kind), Fragment: fragment}
	switch t.Kind {
	case OwnerType, OwnerModule:
		if len(parts) < 1 || len(parts) > 2 {
			return Target{}, fmt.Errorf("%s: want %s%s/<name>[/<entry>]", link, LinkScheme, kind)
		}
		t.Owner = parts[0]
		if len(parts) == 2 {
			t.Entry = parts[1]
		}
	case OwnerBuiltins:
		if len(parts) != 0 {
			return Target{}, fmt.Errorf("%s: want %sbuiltins (a builtin function is %sbuiltin/<name>)", link, LinkScheme, LinkScheme)
		}
	case "builtin":
		if len(parts) != 1 {
			return Target{}, fmt.Errorf("%s: want %sbuiltin/<name>", link, LinkScheme)
		}
		t.Kind, t.Entry = OwnerBuiltins, parts[0]
	case OwnerDoc:
		if len(parts) == 0 {
			return Target{}, fmt.Errorf("%s: want %sdoc/<page>", link, LinkScheme)
		}
		t.Owner = path
		if strings.HasSuffix(path, ".md") || strings.Contains("/"+path+"/", "/../") || strings.Contains("/"+path+"/", "/./") {
			return Target{}, fmt.Errorf("%s: a guide page is named by its path below the docs root, without \".md\" or \"..\"", link)
		}
	default:
		return Target{}, fmt.Errorf("%s: unknown kind %q (type, module, builtins, builtin or doc)", link, kind)
	}
	if t.Entry != "" && hasFragment {
		return Target{}, fmt.Errorf("%s: a link to an entry takes no fragment", link)
	}
	return t, nil
}

// String answers the target as a kavun: link.
func (t Target) String() string {
	var s string
	switch {
	case t.Kind == OwnerBuiltins && t.Entry != "":
		s = LinkScheme + "builtin/" + t.Entry
	case t.Kind == OwnerBuiltins:
		s = LinkScheme + "builtins"
	case t.Entry != "":
		s = LinkScheme + string(t.Kind) + "/" + t.Owner + "/" + t.Entry
	default:
		s = LinkScheme + string(t.Kind) + "/" + t.Owner
	}
	if t.Fragment != "" {
		s += "#" + t.Fragment
	}
	return s
}

// Resolve resolves a kavun: link against the set: its target, with the origin declaring it. The owner and the entry
// must exist, and a fragment must be the slug of a heading of the owner's root block. A kavun:doc/ link is parsed
// only: its page is a file of the linking origin's docs, which the set does not hold (Target.Origin is "").
func (s *Set) Resolve(link string) (Target, error) {
	t, err := ParseLink(link)
	if err != nil {
		return Target{}, err
	}
	var doc Doc
	var ref *Reference
	switch t.Kind {
	case OwnerDoc:
		return t, nil
	case OwnerType:
		e, ok := s.types[t.Owner]
		if !ok {
			return Target{}, fmt.Errorf("%s: no type %s", link, t.Owner)
		}
		if _, ok := e.v.Member(t.Entry); t.Entry != "" && !ok {
			return Target{}, fmt.Errorf("%s: type %s has no member %s", link, t.Owner, t.Entry)
		}
		doc, ref = e.v.Doc, e.ref
	case OwnerModule:
		e, ok := s.modules[t.Owner]
		if !ok {
			return Target{}, fmt.Errorf("%s: no module %s", link, t.Owner)
		}
		if t.Entry != "" {
			_, fn := e.v.Function(t.Entry)
			_, c := e.v.Constant(t.Entry)
			if !fn && !c {
				return Target{}, fmt.Errorf("%s: module %s has no function or constant %s", link, t.Owner, t.Entry)
			}
		}
		doc, ref = e.v.Doc, e.ref
	case OwnerBuiltins:
		if t.Entry != "" {
			e, ok := s.builtins[t.Entry]
			if !ok {
				return Target{}, fmt.Errorf("%s: no builtin function %s", link, t.Entry)
			}
			t.Origin = e.ref.Origin
			return t, nil
		}
		ref = s.builtinsRef()
		if ref == nil {
			return Target{}, fmt.Errorf("%s: no reference declares builtin functions", link)
		}
		doc = ref.Builtins.Doc
	}
	if t.Fragment != "" && !slices.Contains(doc.Headings(), t.Fragment) {
		return Target{}, fmt.Errorf("%s: no heading %q in the block of %s", link, t.Fragment, t.ownerName())
	}
	t.Origin = ref.Origin
	return t, nil
}

// builtinsRef answers the first reference (in Merge order) with a builtins block; nil when none has one.
func (s *Set) builtinsRef() *Reference {
	for _, r := range s.refs {
		if r.Builtins != nil {
			return r
		}
	}
	return nil
}

// ownerName names the target's owner in a message: "type array", "module fin", "the builtins".
func (t Target) ownerName() string {
	if t.Kind == OwnerBuiltins {
		return "the builtins"
	}
	return string(t.Kind) + " " + t.Owner
}
