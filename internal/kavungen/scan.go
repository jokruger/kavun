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
	"strconv"
	"strings"

	"github.com/jokruger/kavun/reference"
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
	// consts and vars hold the names of the package-level constants and variables: a //kavun:names key must be a
	// constant (a table's index), a module's bare id= one of either.
	consts, vars map[string]bool
	// refusedModule reports that a //kavun:module of the package was refused (reported): its functions and
	// constants are then not reported again as lacking a module.
	refusedModule bool
}

// valueDecl is a var or const declaration as a doc comment above it sees it: its token and the names it declares.
type valueDecl struct {
	tok    token.Token
	names  []string
	values []ast.Expr // the initial values of a single spec; nil for a group's doc
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
// owned types (ModuleBuilder.Type) and the rest of the builder's calls. A constant has its directive
// (//kavun:const); one added here by hand is not in the reference.
const extrasFunc = "moduleExtras"

// scanPackage parses the files of one package and collects every directive with its doc block. It reports every
// problem it can see in one file at a time; cross-file rules are model.go's.
func scanPackage(dir string, files []sourceFile) (*scannedPackage, errorList) {
	var errs errorList
	p := &scannedPackage{dir: dir, consts: map[string]bool{}, vars: map[string]bool{}}
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
		valueDocs := map[*ast.CommentGroup]valueDecl{}
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
				if n.Tok == token.VAR || n.Tok == token.CONST {
					// the declaration's doc names everything it declares: one spec, or each spec of its group
					v := valueDecl{tok: n.Tok}
					for _, spec := range n.Specs {
						for _, name := range spec.(*ast.ValueSpec).Names {
							v.names = append(v.names, name.Name)
						}
					}
					if len(n.Specs) == 1 {
						v.values = n.Specs[0].(*ast.ValueSpec).Values
					}
					valueDocs[n.Doc] = v
					if n.Lparen.IsValid() { // in a group, a spec's own doc names its names only
						for _, spec := range n.Specs {
							vs := spec.(*ast.ValueSpec)
							v := valueDecl{tok: n.Tok, values: vs.Values}
							for _, name := range vs.Names {
								v.names = append(v.names, name.Name)
							}
							valueDocs[vs.Doc] = v
						}
					}
				}
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
		delete(valueDocs, nil)

		for _, decl := range f.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && (gd.Tok == token.CONST || gd.Tok == token.VAR) {
				for _, spec := range gd.Specs {
					for _, name := range spec.(*ast.ValueSpec).Names {
						if gd.Tok == token.CONST {
							p.consts[name.Name] = true
						} else {
							p.vars[name.Name] = true
						}
					}
				}
			}
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
				p.refusedModule = p.refusedModule || d.Kind == KindModule
				continue
			}
			inBody := slices.ContainsFunc(bodies, func(b *ast.BlockStmt) bool { return group.Pos() > b.Lbrace && group.End() < b.Rbrace })
			fd := funcDocs[group]
			vd, onValue := valueDocs[group]
			switch {
			case inBody:
				errs.add(d.Pos, "//kavun:%s inside a function body: directives sit at the top level", d.Kind)
				continue
			case d.Kind.attachedToFunc() && fd == nil:
				errs.add(d.Pos, "//kavun:%s must be in the doc comment of the Go function it declares (no blank line between them)", d.Kind)
				continue
			case d.Kind.attachedToVar() && !onValue:
				what := "holding its value"
				if d.Kind == KindNames {
					what = "holding the names"
				}
				errs.add(d.Pos, "%s must be in the doc comment of the package-level var %s (no blank line between them)", d.label(), what)
				continue
			case d.Kind == KindNames && vd.tok == token.CONST:
				errs.add(d.Pos, "//kavun:names is on a Go const: put it on the package-level var holding the map[member.ID]string literal")
				continue
			case d.Kind.attachedToVar() && vd.tok == token.CONST:
				errs.add(d.Pos, "//kavun:%s %s is on a Go const: the value is a core.Value, which a Go const cannot hold; declare it with var", d.Kind, d.Name)
				continue
			case d.Kind.attachedToVar() && len(vd.names) != 1:
				errs.add(d.Pos, "%s is on a var declaring %d names (%s): one var, one name", d.label(), len(vd.names), strings.Join(vd.names, ", "))
				continue
			case !d.Kind.attachedToFunc() && !d.Kind.attachedToVar() && attached[group]:
				errs.add(d.Pos, "//kavun:%s must be free-standing: separate it from the declaration below by a blank line", d.Kind)
				continue
			}
			if d.Kind.attachedToVar() {
				if vd.names[0] == "_" {
					errs.add(d.Pos, "%s is on the blank var `_`: name the var", d.label())
					continue
				}
				d.GoVar = vd.names[0]
			}
			if d.Kind == KindNames {
				// the entries that are well-formed stay declared even when another is refused (generation fails
				// anyway): the members using them are not reported as using undeclared names
				names, ok := scanNames(fset, d, vd, &errs)
				if !ok && len(names) == 0 {
					continue
				}
				d.Names = names
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

// label names a directive in a message: "//kavun:const pi", "//kavun:names".
func (d Directive) label() string {
	if d.Name == "" {
		return directivePrefix + string(d.Kind)
	}
	return directivePrefix + string(d.Kind) + " " + d.Name
}

// namesType is the type of a //kavun:names literal.
const namesType = "map[member.ID]string"

// scanNames reads the map literal of a //kavun:names var: `map[member.ID]string{Key: "name", …}`, every key an
// identifier (a constant of the package, checked by the caller) and every value a string literal that is a member
// name. ok is false when the literal is malformed (reported).
func scanNames(fset *token.FileSet, d Directive, vd valueDecl, errs *errorList) ([]NameDecl, bool) {
	var lit *ast.CompositeLit
	if len(vd.values) == 1 {
		lit, _ = vd.values[0].(*ast.CompositeLit)
	}
	if lit == nil || lit.Type == nil || types.ExprString(lit.Type) != namesType {
		errs.add(d.Pos, "//kavun:names %s: the var must hold a %s{Key: \"name\", …} literal", d.GoVar, namesType)
		return nil, false
	}
	var out []NameDecl
	ok := true
	for _, elt := range lit.Elts {
		pos := fset.Position(elt.Pos())
		kv, isKV := elt.(*ast.KeyValueExpr)
		if !isKV {
			errs.add(pos, "//kavun:names %s: an entry is not `Key: \"name\"`", d.GoVar)
			ok = false
			continue
		}
		key, isIdent := kv.Key.(*ast.Ident)
		val, isLit := kv.Value.(*ast.BasicLit)
		if !isIdent {
			errs.add(pos, "//kavun:names %s: the key %s is not an identifier of the package (a member.ID constant)", d.GoVar, types.ExprString(kv.Key))
			ok = false
			continue
		}
		if !isLit || val.Kind != token.STRING {
			errs.add(pos, "//kavun:names %s: the name of %s is not a string literal", d.GoVar, key.Name)
			ok = false
			continue
		}
		name, err := strconv.Unquote(val.Value)
		if err != nil || !namePattern.MatchString(name) {
			errs.add(pos, "//kavun:names %s: %s is not a member name ([a-z_][a-z0-9_]*)", d.GoVar, val.Value)
			ok = false
			continue
		}
		out = append(out, NameDecl{Key: key.Name, Name: name, Pos: pos})
	}
	if ok && len(out) == 0 {
		errs.add(d.Pos, "//kavun:names %s declares no name", d.GoVar)
		return nil, false
	}
	return out, ok
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
				return Directive{Kind: parsed.Kind}, false // its kind, for refusedModule
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
			if title, ok := reference.HeadingText(line); ok && !fenced[i] {
				level := len(line) - len(strings.TrimLeft(line, "#"))
				d.Sections = append(d.Sections, Section{Title: title, Level: level, Pos: linePos[i]})
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
	var starts []int // where each line starts in the summary
	n := 0
	for end < len(lines) && !isBlank(lines[end]) {
		w := strings.TrimSpace(lines[end])
		words = append(words, w)
		starts = append(starts, n)
		n += len(w) + 1
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
	return Doc{Summary: summary, SummaryPos: pos[start:end], SummaryAt: starts, Body: body, BodyPos: bodyPos}, at, nil
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
	msg := fmt.Sprintf(format, args...)
	if w := where(pos); w != "" {
		msg = w + ": " + msg
	}
	*l = append(*l, msg)
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
