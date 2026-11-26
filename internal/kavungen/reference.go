package kavungen

import (
	"bytes"
	"go/token"
	"path"
	"path/filepath"
	"strings"

	"github.com/jokruger/kavun/reference"
)

// The reference emitter: the second output of the same parse (the tables are the first). It builds the module's kavun_reference.json
// (reference.Reference) from the directives and their doc blocks: what every type, builtin function and module
// declares, with no id, slot or anything else of the binding.

// referenceFile answers the reference file at path: nil content when ref is nil, the packages declaring nothing
// (Generate then deletes a file at the default location, and leaves one named by -reference alone).
func referenceFile(ref *reference.Reference, path string) (File, error) {
	f := File{Path: path}
	if ref == nil {
		return f, nil
	}
	var buf bytes.Buffer
	if err := ref.Write(&buf); err != nil {
		return f, err
	}
	f.Content = buf.Bytes()
	return f, nil
}

// buildReference builds the reference of the packages — for every run, whether or not it writes the file: the
// link targets are checked against it. A type or module declared by two packages, or a second builtins block, is
// refused. Every link of the doc blocks is written in its absolute form, and every kavun: target (but kavun:doc/)
// must resolve against the reference and the loaded references of other modules (links.go); partial reports that the
// packages are not the whole module, so a target whose owner neither the run nor a loaded reference declares is not
// checked.
func buildReference(models []*packageModel, origin, coreVersion, docsURL string, partial bool, loaded []*reference.Reference) (*reference.Reference, errorList) {
	var errs errorList
	ref := &reference.Reference{
		Schema:      reference.SchemaVersion,
		Origin:      origin,
		CoreVersion: coreVersion,
		Docs:        docsURL,
		Types:       []reference.Type{},
		Modules:     []reference.Module{},
	}
	types := map[string]Directive{}
	modules := map[string]Directive{}
	var builtins *Directive
	builtinFuncs := map[string]Directive{}
	var links []docLink // every kavun: link, to resolve once the reference is complete

	for _, m := range models {
		for _, t := range m.types {
			if prev, ok := types[t.decl.Name]; ok {
				errs.add(t.decl.Pos, "type %q is declared twice (first at %s)", t.decl.Name, where(prev.Pos))
				continue
			}
			types[t.decl.Name] = t.decl
			owner := ""
			if prefix, _, dotted := strings.Cut(t.decl.Name, "."); dotted {
				owner = prefix
			}
			o := newLinkOwner(t.decl, t.members)
			doc, found := o.absoluteDoc(t.decl.Doc)
			links = append(links, found...)
			ref.Types = append(ref.Types, reference.Type{
				Name:    t.decl.Name,
				Module:  owner,
				Summary: doc.Summary,
				Doc:     docBlocks(doc),
				Pos:     refPos(m, t.decl.Pos),
				Members: refCallables(m, o, &links),
			})
		}
		if m.builtins != nil {
			if builtins != nil {
				errs.add(m.builtins.Pos, "a second //kavun:builtins in the module (the first is at %s)", where(builtins.Pos))
			} else {
				builtins = m.builtins
				for _, d := range m.builtinFuncs {
					builtinFuncs[d.Name] = d
				}
				o := newLinkOwner(*m.builtins, m.builtinFuncs)
				doc, found := o.absoluteDoc(m.builtins.Doc)
				links = append(links, found...)
				ref.Builtins = &reference.Builtins{
					Doc:       docBlocks(doc),
					Pos:       refPos(m, m.builtins.Pos),
					Functions: refCallables(m, o, &links),
				}
			}
		}
		if d := m.module; d != nil {
			if prev, ok := modules[d.Name]; ok {
				errs.add(d.Pos, "module %q is declared twice (first at %s)", d.Name, where(prev.Pos))
				continue
			}
			modules[d.Name] = *d
			o := newLinkOwner(*d, m.moduleFuncs).withConsts(m.moduleConsts)
			doc, found := o.absoluteDoc(d.Doc)
			links = append(links, found...)
			ref.Modules = append(ref.Modules, reference.Module{
				Name:      d.Name,
				Pure:      d.Pure,
				Summary:   doc.Summary,
				Doc:       docBlocks(doc),
				Pos:       refPos(m, d.Pos),
				Functions: refCallables(m, o, &links),
				Constants: refConstants(m, o, &links),
			})
		}
	}
	checkLoaded(origin, types, modules, builtinFuncs, loaded, &errs)
	if len(errs) == 0 {
		checkTargets(ref, loaded, links, partial, &errs)
	}
	return ref, errs
}

// at renders a loaded declaration's position for a message: " (core/string.go:12)", "" when it has none.
func at(p reference.Pos) string {
	if p.File == "" {
		return ""
	}
	return " (" + p.String() + ")"
}

// checkLoaded refuses a type, module or builtin function the run declares that a loaded reference declares too,
// at the run's declaration, and a loaded reference of the run's own origin.
func checkLoaded(origin string, types, modules, builtins map[string]Directive, loaded []*reference.Reference, errs *errorList) {
	for _, r := range loaded {
		if origin != "" && r.Origin == origin {
			errs.add(token.Position{}, "a loaded reference has the origin of the run, %s: give another module's", origin)
		}
		for _, t := range r.Types {
			if d, ok := types[t.Name]; ok {
				errs.add(d.Pos, "type %q is declared by %s too%s: a type name is declared once across the references", t.Name, r.Origin, at(t.Pos))
			}
		}
		for _, m := range r.Modules {
			if d, ok := modules[m.Name]; ok {
				errs.add(d.Pos, "module %q is declared by %s too%s: a module name is declared once across the references", m.Name, r.Origin, at(m.Pos))
			}
		}
		if r.Builtins != nil {
			for _, f := range r.Builtins.Functions {
				if d, ok := builtins[f.Name]; ok {
					errs.add(d.Pos, "builtin %q is declared by %s too%s: a builtin name is declared once across the references", f.Name, r.Origin, at(f.Pos))
				}
			}
		}
	}
}

// refCallables answers the reference entries of an owner's member, builtin or module-function directives, their
// links in absolute form; it adds their kavun: links to links.
func refCallables(m *packageModel, o *linkOwner, links *[]docLink) []reference.Callable {
	out := make([]reference.Callable, 0, len(o.members))
	for _, d := range o.members {
		doc, found := o.absoluteDoc(d.Doc)
		*links = append(*links, found...)
		out = append(out, reference.Callable{
			Name:      d.Name,
			Pure:      d.Pure,
			Signature: reference.SignatureOf(d.Sig),
			Result:    d.Result,
			Summary:   doc.Summary,
			Doc:       docBlocks(doc),
			Pos:       refPos(m, d.Pos),
		})
	}
	return out
}

// refConstants answers the reference entries of a module's constant directives, their links in absolute form; it
// adds their kavun: links to links.
func refConstants(m *packageModel, o *linkOwner, links *[]docLink) []reference.Constant {
	out := make([]reference.Constant, 0, len(o.consts))
	for _, d := range o.consts {
		doc, found := o.absoluteDoc(d.Doc)
		*links = append(*links, found...)
		out = append(out, reference.Constant{
			Name:    d.Name,
			Type:    d.Result,
			Summary: doc.Summary,
			Doc:     docBlocks(doc),
			Pos:     refPos(m, d.Pos),
		})
	}
	return out
}

// refPos answers a source position as the reference states it: the file relative to the module root,
// slash-separated, and the line.
func refPos(m *packageModel, pos token.Position) reference.Pos {
	return reference.Pos{File: path.Join(m.relDir, filepath.Base(pos.Filename)), Line: pos.Line}
}

// docBlocks splits a doc body into the reference's blocks: each fenced ```go block is an example (its code without
// the fence lines, gofmt's code-block tab and the blank lines at its edges; the source line of its first code
// line; the norun marker and reason of its fence), and the text between examples is a text block, verbatim but for
// gofmt's code-block tab (renderBody: a fenced block of another language loses it). Joined back with newlines
// (reference.Doc.Markdown) the blocks are the body a page shows.
func docBlocks(doc Doc) reference.Doc {
	out := reference.Doc{}
	body, pos := doc.Body, doc.BodyPos
	text := func(lines []string) {
		if len(lines) > 0 {
			out = append(out, reference.Block{Kind: reference.BlockText, Markdown: strings.Join(renderBody(lines), "\n")})
		}
	}
	next := 0 // the first line not yet in a block
	for _, sp := range fenceSpans(body) {
		if sp.close < 0 { // refused by checkMarkdown
			break
		}
		lang, noRun, reason := fenceInfo(body[sp.open])
		if lang != "go" {
			continue // stays in the text around it
		}
		text(body[next:sp.open])
		next = sp.close + 1

		content, contentPos := trimBlankPos(body[sp.open+1:sp.close], pos[sp.open+1:sp.close])
		tabbed := containsTabbed(content)
		code := make([]string, len(content))
		for i, l := range content {
			if tabbed {
				l = strings.TrimPrefix(l, "\t")
			}
			code[i] = l
		}
		ex := reference.Block{Kind: reference.BlockExample, Code: strings.Join(code, "\n"), NoRun: noRun, Reason: reason}
		if len(contentPos) > 0 {
			ex.Line = contentPos[0].Line
		}
		out = append(out, ex)
	}
	text(body[next:])
	return out
}
