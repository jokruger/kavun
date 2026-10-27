package kavungen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// sourceFile is one Go source file of a package: its base name and content.
type sourceFile struct {
	Name string
	Src  []byte
}

// scannedFile is what one source file declares: its directives in source order.
type scannedFile struct {
	name       string // base name, "array_members.go"
	directives []Directive
	// failed reports that a directive of this file was refused (malformed, misplaced, bad doc block) and dropped:
	// the cross-directive checks then skip the follow-up its absence would cause (a type with no members), which
	// would only point away from the real problem.
	failed bool
}

// scannedPackage is what the scan of one package found, before the cross-directive checks in model.go.
type scannedPackage struct {
	dir       string // as displayed in messages
	pkgName   string
	files     []scannedFile // sorted by name
	hasExtras bool          // the package declares func moduleExtras(m *core.ModuleBuilder)
}

// otherToolDirective matches a comment line that is another tool's directive (`//go:build`, `//nolint:x`,
// `//line`, …); such lines belong to neither a directive nor a doc block.
var otherToolDirective = regexp.MustCompile(`^//(line |extern |export |[a-z0-9]+:[a-z0-9])`)

// The expected shapes, with the `core.` qualifier removed so a package other than core spells them the same.
var (
	memberShape = funcShapeOf([]string{"VM", "Value", "member.ID", "[]Value"}, []string{"Value", "error"})
	nativeShape = funcShapeOf([]string{"VM", "[]Value"}, []string{"Value", "error"})
	extrasShape = funcShapeOf([]string{"*ModuleBuilder"}, nil)
)

// extrasFunc is the optional hand-written hook a module package declares to add what the generated init does not:
// constants (ModuleBuilder.Const), owned types (ModuleBuilder.Type) and the rest of the builder's calls.
const extrasFunc = "moduleExtras"

// scanPackage parses the files of one package and collects every directive with its doc block. It reports every
// problem it can see in one file at a time; cross-file rules are model.go's.
func scanPackage(dir string, files []sourceFile) (*scannedPackage, errorList) {
	var errs errorList
	p := &scannedPackage{dir: dir}
	fset := token.NewFileSet()

	files = slices.Clone(files)
	slices.SortFunc(files, func(a, b sourceFile) int { return strings.Compare(a.Name, b.Name) })

	for _, sf := range files {
		display := filepath.ToSlash(filepath.Join(dir, sf.Name))
		f, err := parser.ParseFile(fset, display, sf.Src, parser.ParseComments)
		if err != nil {
			errs.add(token.Position{Filename: display}, "%v", err)
			continue
		}
		if p.pkgName == "" {
			p.pkgName = f.Name.Name
		}

		// Every comment group attached to a declaration, and every function body, so a directive's place can be
		// judged: a function's doc comment, attached elsewhere, free-standing, or inside a body.
		attached := map[*ast.CommentGroup]bool{}
		funcDocs := map[*ast.CommentGroup]*ast.FuncDecl{}
		var bodies []*ast.BlockStmt
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.File:
				attached[n.Doc] = true
			case *ast.FuncDecl:
				attached[n.Doc] = true
				if n.Recv == nil {
					funcDocs[n.Doc] = n
				}
				if n.Body != nil {
					bodies = append(bodies, n.Body)
				}
			case *ast.GenDecl:
				attached[n.Doc] = true
			case *ast.Field:
				attached[n.Doc], attached[n.Comment] = true, true
			case *ast.ValueSpec:
				attached[n.Doc], attached[n.Comment] = true, true
			case *ast.TypeSpec:
				attached[n.Doc], attached[n.Comment] = true, true
			case *ast.ImportSpec:
				attached[n.Doc], attached[n.Comment] = true, true
			}
			return true
		})
		delete(attached, nil)

		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv != nil {
				continue
			}
			if fd.Name.Name == extrasFunc {
				if funcShape(fd) != extrasShape {
					errs.add(fset.Position(fd.Pos()), "func %s must be `func %s(m *core.ModuleBuilder)`", extrasFunc, extrasFunc)
				}
				p.hasExtras = true
			}
		}

		out := scannedFile{name: sf.Name}
		before := len(errs)
		for _, group := range f.Comments {
			d, ok := scanGroup(fset, group, &errs)
			if !ok {
				continue
			}
			inBody := slices.ContainsFunc(bodies, func(b *ast.BlockStmt) bool { return group.Pos() > b.Lbrace && group.End() < b.Rbrace })
			fd := funcDocs[group]
			switch {
			case inBody:
				errs.add(d.Pos, "//kavun:%s inside a function body: directives sit at the top level", d.Kind)
				continue
			case d.Kind.attachedToFunc() && fd == nil:
				errs.add(d.Pos, "//kavun:%s must be in the doc comment of the Go function it declares (no blank line between them)", d.Kind)
				continue
			case !d.Kind.attachedToFunc() && attached[group]:
				errs.add(d.Pos, "//kavun:%s must be free-standing: separate it from the declaration below by a blank line", d.Kind)
				continue
			}
			if fd != nil {
				want, shapeName := nativeShape, "func(vm core.VM, args []core.Value) (core.Value, error)"
				if d.Kind == KindMember {
					want, shapeName = memberShape, "func(vm VM, v Value, id member.ID, args []Value) (Value, error)"
				}
				if funcShape(fd) != want {
					errs.add(d.Pos, "//kavun:%s %s: %s must have the shape %s", d.Kind, d.Name, fd.Name.Name, shapeName)
					continue
				}
				d.GoFunc = fd.Name.Name
			}
			out.directives = append(out.directives, d)
		}
		out.failed = len(errs) > before
		p.files = append(p.files, out)
	}
	return p, errs
}

// scanGroup reads one comment group: ok is false when it holds no directive (or a malformed one, reported).
func scanGroup(fset *token.FileSet, group *ast.CommentGroup, errs *errorList) (Directive, bool) {
	var d Directive
	found := false
	var lines []string
	var linePos []token.Position // the position of each line in lines
	for _, c := range group.List {
		pos := fset.Position(c.Slash)
		switch {
		case strings.HasPrefix(c.Text, directivePrefix):
			if found {
				errs.add(pos, "more than one //kavun: directive in one comment group")
				return d, false
			}
			parsed, err := parseDirective(strings.TrimSpace(strings.TrimPrefix(c.Text, directivePrefix)))
			if err != nil {
				errs.add(pos, "%v", err)
				return d, false
			}
			parsed.Pos = pos
			d, found = parsed, true
		case strings.HasPrefix(c.Text, "/*"), otherToolDirective.MatchString(c.Text):
			// a block comment or another tool's directive: never part of a doc block
		default:
			line := strings.TrimPrefix(c.Text, "//")
			lines = append(lines, strings.TrimPrefix(line, " "))
			linePos = append(linePos, pos)
		}
	}
	if !found {
		return d, false
	}
	if !checkMarkdown(lines, linePos, d.Kind.isPageRoot(), errs) {
		return d, false
	}
	if d.Kind.isPageRoot() {
		fenced := make([]bool, len(lines))
		for _, sp := range fenceSpans(lines) {
			for i := sp.open; i <= sp.close; i++ {
				fenced[i] = true
			}
		}
		for i, line := range lines {
			if title, ok := strings.CutPrefix(line, "## "); ok && !fenced[i] {
				d.Sections = append(d.Sections, Section{Title: strings.TrimSpace(title), Pos: linePos[i]})
			}
		}
	}

	if !d.Kind.needsSummary() { // the builtins' block is an optional intro, all body
		body, bodyPos := trimBlankPos(lines, linePos)
		d.Doc = Doc{Body: body, BodyPos: bodyPos}
		return d, true
	}
	doc, at, err := parseDoc(lines, linePos)
	if err == nil && doc.Summary == "" {
		at, err = d.Pos, fmt.Errorf("//kavun:%s needs a doc block opening with a one-sentence summary", d.Kind)
	}
	if err != nil {
		errs.add(at, "%v", err)
		return d, false
	}
	d.Doc = doc
	return d, true
}

// secondSentence finds where a second sentence starts in a summary: a period, a space and an upper-case letter.
// A heuristic — it would also stop "e.g. Foo" — kept simple on purpose, since a summary is one plain sentence.
var secondSentence = regexp.MustCompile(`\. \p{Lu}`)

// parseDoc splits the doc lines into the summary and the markdown body. The summary is the first paragraph — its
// lines up to the first blank line, joined with single spaces — and must be one sentence ending in a period; the
// body is everything after it. pos holds each line's position; at answers where a problem is.
func parseDoc(lines []string, pos []token.Position) (doc Doc, at token.Position, err error) {
	start := 0
	for start < len(lines) && isBlank(lines[start]) {
		start++
	}
	if start == len(lines) {
		return Doc{}, at, nil
	}
	end := start
	var words []string
	for end < len(lines) && !isBlank(lines[end]) {
		words = append(words, strings.TrimSpace(lines[end]))
		end++
	}
	summary := strings.Join(words, " ")
	if !strings.HasSuffix(summary, ".") {
		return Doc{}, pos[start], fmt.Errorf("summary %q must be one sentence ending in a period", summary)
	}
	if loc := secondSentence.FindStringIndex(summary); loc != nil {
		return Doc{}, pos[start], fmt.Errorf("summary %q must be one sentence: a second one starts at %q; move it to the body, after a blank comment line", summary, summary[loc[0]+2:])
	}
	body, bodyPos := trimBlankPos(lines[end:], pos[end:])
	return Doc{Summary: summary, Body: body, BodyPos: bodyPos}, at, nil
}

// trimBlankPos is trimBlank over lines and their positions together: it answers the same lines and the positions
// that belong to them.
func trimBlankPos(lines []string, pos []token.Position) ([]string, []token.Position) {
	lo, hi := 0, len(lines)
	for lo < hi && strings.TrimSpace(lines[lo]) == "" {
		lo++
	}
	for hi > lo && strings.TrimSpace(lines[hi-1]) == "" {
		hi--
	}
	if lo == hi {
		return nil, nil
	}
	return lines[lo:hi], pos[lo:hi]
}

// trimBlank drops leading and trailing blank lines.
func trimBlank(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return nil
	}
	return lines
}

// funcShapeOf renders a parameter and result type list as one comparable string.
func funcShapeOf(params, results []string) string {
	return "(" + strings.Join(params, ", ") + ") (" + strings.Join(results, ", ") + ")"
}

// funcShape renders a function declaration's types (names dropped, the `core.` qualifier removed) for comparison
// with the expected shapes; a generic function never matches.
func funcShape(fd *ast.FuncDecl) string {
	if fd.Type.TypeParams != nil {
		return "generic"
	}
	list := func(fl *ast.FieldList) []string {
		var out []string
		if fl == nil {
			return out
		}
		for _, field := range fl.List {
			t := strings.ReplaceAll(types.ExprString(field.Type), "core.", "")
			for range max(1, len(field.Names)) {
				out = append(out, t)
			}
		}
		return out
	}
	return funcShapeOf(list(fd.Type.Params), list(fd.Type.Results))
}

// errorList collects problems, each tied to a source position, for one report.
type errorList []string

func (l *errorList) add(pos token.Position, format string, args ...any) {
	*l = append(*l, where(pos)+": "+fmt.Sprintf(format, args...))
}

// where renders a position as file:line (just the file when the line is unknown).
func where(pos token.Position) string {
	if pos.Line > 0 {
		return fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
	}
	return pos.Filename
}

// err answers the list as one error, or nil when it is empty.
func (l errorList) err() error {
	if len(l) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(l, "\n"))
}
