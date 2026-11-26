package kavungen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
)

// Context is what the generator reads from the Kavun source tree itself, once, before it looks at any package: the Go
// spelling of every builtin member id, the constants of core/value (a module's id=value.<Ident> must name one), the
// slot range and the core version.
type Context struct {
	// memberIdents maps a builtin member name to its constant in core/member/members ("pad_start" → "PadStart"),
	// parsed from the builtinNames literal in core/member/builtin.go — never kept by hand.
	memberIdents map[string]string
	// valueConsts holds the names of the constants of core/value/value.go: a module's id=value.<Ident> is checked
	// against them (no naming convention is read).
	valueConsts map[string]bool
	// slotSize is core.ModuleSlotSize (core/types.go): a builtin's or module function's slot is below it.
	slotSize int
	// coreVersion is Kavun's core version, "<major>.<minor>" from coreVersionMajor/coreVersionMinor in vm/bytecode.go:
	// the version a reference is generated against.
	coreVersion string
}

// LoadContext parses core/member/builtin.go, core/value/value.go, core/types.go and vm/bytecode.go under kavunRoot, the root
// directory of the github.com/jokruger/kavun module. It reads source, never imports package core, so the generator
// still runs while core does not compile (a stale kavun_gen.go there is exactly what it is run to fix).
func LoadContext(kavunRoot string) (*Context, error) {
	idents, err := parseMemberIdents(filepath.Join(kavunRoot, "core", "member", "builtin.go"))
	if err != nil {
		return nil, err
	}
	valueConsts, err := parseConstNames(filepath.Join(kavunRoot, "core", "value", "value.go"))
	if err != nil {
		return nil, err
	}
	slotSize, err := parseIntConst(filepath.Join(kavunRoot, "core", "types.go"), "ModuleSlotSize")
	if err != nil {
		return nil, err
	}
	bytecode := filepath.Join(kavunRoot, "vm", "bytecode.go")
	major, err := parseIntConst(bytecode, "coreVersionMajor")
	if err != nil {
		return nil, err
	}
	minor, err := parseIntConst(bytecode, "coreVersionMinor")
	if err != nil {
		return nil, err
	}
	return &Context{memberIdents: idents, valueConsts: valueConsts, slotSize: slotSize, coreVersion: fmt.Sprintf("%d.%d", major, minor)}, nil
}

// parseConstNames collects the names of a file's package-level constants.
func parseConstNames(path string) (map[string]bool, error) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.CONST {
			for _, spec := range gd.Specs {
				for _, n := range spec.(*ast.ValueSpec).Names {
					names[n.Name] = true
				}
			}
		}
	}
	return names, nil
}

// parseIntConst reads the value of an integer constant declared with a literal.
func parseIntConst(path, name string) (int, error) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return 0, err
	}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, n := range vs.Names {
				if n.Name != name || i >= len(vs.Values) {
					continue
				}
				if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.INT {
					return strconv.Atoi(lit.Value)
				}
			}
		}
	}
	return 0, fmt.Errorf("%s: no integer constant %s", path, name)
}

// parseMemberIdents reads the `var builtinNames = [...]string{members.X: "name", …}` literal.
func parseMemberIdents(path string) (map[string]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}
	var lit *ast.CompositeLit
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			if len(vs.Names) == 1 && vs.Names[0].Name == "builtinNames" && len(vs.Values) == 1 {
				lit, _ = vs.Values[0].(*ast.CompositeLit)
			}
		}
	}
	if lit == nil {
		return nil, fmt.Errorf("%s: no `var builtinNames = [...]string{…}` literal", path)
	}

	idents := map[string]string{}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, fmt.Errorf("%s: builtinNames entry is not `members.X: \"name\"`", path)
		}
		sel, ok1 := kv.Key.(*ast.SelectorExpr)
		val, ok2 := kv.Value.(*ast.BasicLit)
		if !ok1 || !ok2 || val.Kind != token.STRING {
			return nil, fmt.Errorf("%s: builtinNames entry is not `members.X: \"name\"`", path)
		}
		if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "members" {
			return nil, fmt.Errorf("%s: builtinNames key is not a members constant", path)
		}
		name, err := strconv.Unquote(val.Value)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		idents[name] = sel.Sel.Name
	}
	return idents, nil
}
