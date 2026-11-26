package kavungen

import (
	"go/token"
	"regexp"
	"slices"
	"strings"

	"github.com/jokruger/kavun/reference"
)

// The links of doc blocks. A link names what it links to, never a page (reference.ParseLink): a kavun: target, a
// same-page shorthand #x, or an absolute URL. The links are read by reference.FindLinks — the one link reader —
// from every summary and every body.
//
// The checks run in two places. Per package (checkLinks, from buildModel): a relative target is refused, a kavun:
// target must be well-formed, a shorthand must name an entry of its owner or a heading of the owner's root block,
// and a root-block heading must not take an entry's anchor. With the reference (buildReference, built in memory by
// every run; only a run over the whole module writes it): every shorthand is written in its absolute kavun: form,
// and every kavun: target but kavun:doc/ (a page of the rendered site, checked by kavun-docs) is resolved against
// the declarations of the run (reference.Set.Resolve). A run over part of the module checks a target only when the
// run declares its owner — the type, the module, or (for kavun:builtins and kavun:builtin/) a builtins block: then
// the entry or heading it names must exist. A target whose owner the run does not declare passes, since another
// package of the module may declare it; the whole-module run (go generate) checks it.

// linkOwner is the owner of a page's doc blocks — a type, the builtins or a module — as its links see it.
type linkOwner struct {
	desc     string          // names it in a message: "type array"
	link     string          // its kavun: link: "kavun:type/array"
	entry    string          // the link of one of its entries, before the name: "kavun:type/array/"
	entries  map[string]bool // its entries' names
	root     Directive       // its root block
	members  []Directive     // its callable entries: members or functions
	consts   []Directive     // a module's constants, its other entries
	headings map[string]bool // the slugs of its root block's headings
}

// newLinkOwner answers the link owner of a root directive with its entries.
func newLinkOwner(root Directive, entries []Directive) *linkOwner {
	o := &linkOwner{root: root, members: entries, entries: map[string]bool{}, headings: map[string]bool{}}
	switch root.Kind {
	case KindType:
		o.desc, o.link, o.entry = "type "+root.Name, reference.LinkScheme+"type/"+root.Name, reference.LinkScheme+"type/"+root.Name+"/"
	case KindModule:
		o.desc, o.link, o.entry = "module "+root.Name, reference.LinkScheme+"module/"+root.Name, reference.LinkScheme+"module/"+root.Name+"/"
	case KindBuiltins:
		o.desc, o.link, o.entry = "the builtins", reference.LinkScheme+"builtins", reference.LinkScheme+"builtin/"
	}
	for _, e := range entries {
		o.entries[e.Name] = true
	}
	for _, s := range root.Sections {
		o.headings[reference.Slug(s.Title)] = true
	}
	return o
}

// linkOwners answers the link owners of a package: its types, its builtins, its module.
func linkOwners(m *packageModel) []*linkOwner {
	var out []*linkOwner
	for _, t := range m.types {
		out = append(out, newLinkOwner(t.decl, t.members))
	}
	if m.builtins != nil {
		out = append(out, newLinkOwner(*m.builtins, m.builtinFuncs))
	}
	if m.module != nil {
		out = append(out, newLinkOwner(*m.module, m.moduleFuncs).withConsts(m.moduleConsts))
	}
	return out
}

// withConsts adds a module's constants to its entries: a shorthand #x names one, and a root heading may not take
// one's anchor.
func (o *linkOwner) withConsts(consts []Directive) *linkOwner {
	o.consts = consts
	for _, c := range consts {
		o.entries[c.Name] = true
	}
	return o
}

// docs answers the doc blocks of the owner: its root block's, then its entries'.
func (o *linkOwner) docs() []Doc {
	out := []Doc{o.root.Doc}
	for _, e := range o.members {
		out = append(out, e.Doc)
	}
	for _, c := range o.consts {
		out = append(out, c.Doc)
	}
	return out
}

// absolute answers the kavun: form of a shorthand #x: the entry x when the owner has one, else the heading x of its
// root block; ok is false when it has neither.
func (o *linkOwner) absolute(x string) (string, bool) {
	switch {
	case o.entries[x]:
		return o.entry + x, true
	case o.headings[x]:
		return o.link + "#" + x, true
	}
	return "", false
}

// docLink is one link of a doc block with the source position of its line.
type docLink struct {
	reference.Link
	Pos token.Position
}

// links answers the links of a doc block, each with the source position of its line: its summary's and its
// body's.
func (d Doc) links() []docLink {
	var out []docLink
	for _, l := range reference.FindLinks(d.Summary) {
		out = append(out, docLink{l, d.summaryPos(l.Offset)})
	}
	for _, l := range reference.FindLinks(strings.Join(d.Body, "\n")) {
		out = append(out, docLink{l, d.BodyPos[l.Line]})
	}
	return out
}

// summaryPos answers the source position of the summary line holding the byte at offset of the summary.
func (d Doc) summaryPos(offset int) token.Position {
	var pos token.Position
	for i, at := range d.SummaryAt {
		if at > offset {
			break
		}
		pos = d.SummaryPos[i]
	}
	return pos
}

// uriScheme matches the scheme of an absolute URI: "https:", "mailto:", "kavun:".
var uriScheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// checkLinks applies the per-package link rules to every owner of the package: no root-block heading with an
// entry's anchor; every link a kavun: target of the right form, a shorthand that names an entry or a heading of
// its owner, or an absolute URL (any scheme: https:, mailto:, …) — a relative target is refused.
func checkLinks(m *packageModel, errs *errorList) {
	for _, o := range linkOwners(m) {
		for _, s := range o.root.Sections {
			if slug := reference.Slug(s.Title); o.entries[slug] {
				errs.add(s.Pos, "the heading %q of the //kavun:%s block has the anchor #%s, which is the entry %s's on the same page: rename the heading", s.Title, o.root.Kind, slug, slug)
			}
		}
		for _, doc := range o.docs() {
			for _, l := range doc.links() {
				target := l.Target
				switch {
				case strings.HasPrefix(target, "#"):
					if _, ok := o.absolute(target[1:]); !ok {
						errs.add(l.Pos, "link %s: %s has no entry %s and its block no heading with that slug", target, o.desc, target[1:])
					}
				case strings.HasPrefix(target, reference.LinkScheme):
					if _, err := reference.ParseLink(target); err != nil {
						errs.add(l.Pos, "link %v", err)
					}
				case uriScheme.MatchString(target):
					// an absolute URL passes untouched
				default:
					errs.add(l.Pos, "link %s: a relative link names a page of one layout; name the target with a %s link instead (kavun:type/<type>[/<member>], kavun:module/<module>[/<entry>], kavun:builtin/<name>, kavun:doc/<page>, a #heading fragment)", target, reference.LinkScheme)
				}
			}
		}
	}
}

// ownerDeclared reports whether the owner a kavun: link names is declared by a reference of the set: its type or
// module, or for the builtins a builtins block. A malformed link counts as declared, so that it is reported.
func ownerDeclared(set *reference.Set, link string) bool {
	t, err := reference.ParseLink(link)
	if err != nil {
		return true
	}
	switch t.Kind {
	case reference.OwnerType:
		_, ok := set.Type(t.Owner)
		return ok
	case reference.OwnerModule:
		_, ok := set.Module(t.Owner)
		return ok
	case reference.OwnerBuiltins:
		return slices.ContainsFunc(set.References(), func(r *reference.Reference) bool { return r.Builtins != nil })
	}
	return true
}

// absoluteDoc answers a doc block with every shorthand #x of its summary and body written in its absolute kavun:
// form (the reference holds no shorthand), and its kavun: links (those written so and those rewritten) with their
// positions. A shorthand the owner cannot resolve was refused by checkLinks; it is left as it is.
func (o *linkOwner) absoluteDoc(d Doc) (Doc, []docLink) {
	var found []docLink
	rewrite := func(l reference.Link, pos token.Position) string {
		target := l.Target
		if x, ok := strings.CutPrefix(target, "#"); ok {
			if abs, ok := o.absolute(x); ok {
				target = abs
			}
		}
		if strings.HasPrefix(target, reference.LinkScheme) {
			found = append(found, docLink{reference.Link{Target: target, Offset: l.Offset, Line: l.Line}, pos})
		}
		return target
	}
	out := d
	out.Summary = reference.ReplaceLinks(d.Summary, func(l reference.Link) string { return rewrite(l, d.summaryPos(l.Offset)) })
	body := reference.ReplaceLinks(strings.Join(d.Body, "\n"), func(l reference.Link) string { return rewrite(l, d.BodyPos[l.Line]) })
	if d.Body != nil {
		out.Body = strings.Split(body, "\n")
	}
	return out, found
}

// checkTargets resolves every kavun: link of the reference's doc blocks but kavun:doc/ against the reference
// itself and the loaded references of other modules (Kavun's for an embedder, -with), merged — a name declared by
// two of them is refused —, reporting each that names nothing at its position. In a partial run (the packages are
// not the whole module) a target whose owner no reference declares — a type or module of another package, a builtin
// when no package of the run nor a loaded reference holds the builtins — is not checked: it may well exist elsewhere
// in the module.
func checkTargets(ref *reference.Reference, loaded []*reference.Reference, links []docLink, partial bool, errs *errorList) {
	set, err := reference.Merge(append([]*reference.Reference{ref}, loaded...)...)
	if err != nil {
		errs.add(token.Position{}, "%v", err)
		return
	}
	for _, l := range links {
		if partial && !ownerDeclared(set, l.Target) {
			continue
		}
		if _, err := set.Resolve(l.Target); err != nil {
			errs.add(l.Pos, "link %v", err)
		}
	}
}
